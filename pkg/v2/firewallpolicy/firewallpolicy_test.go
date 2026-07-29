package firewallpolicy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

type scriptedClient struct {
	requests  []*http.Request
	responses []*http.Response
}

func (client *scriptedClient) Do(request *http.Request) (*http.Response, error) {
	client.requests = append(client.requests, request)
	return client.responses[len(client.requests)-1], nil
}

func newClient(t *testing.T, responses ...*http.Response) (*vpc.Client, *scriptedClient) {
	t.Helper()
	transport := &scriptedClient{responses: responses}
	client, err := vpc.NewClient(vpc.Config{
		Endpoint: "https://network.example.test", Token: "token", HTTPClient: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client, transport
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func TestFirewallPolicyCRUDPreservesRuleOrder(t *testing.T) {
	model := `{"firewall_policy":{"id":"fp-id","name":"edge","firewall_rules":["r2","r1"],` +
		`"audited":false,"shared":false}}`
	client, transport := newClient(
		t,
		response(http.StatusCreated, model),
		response(http.StatusOK, model),
		response(http.StatusOK, model),
		response(http.StatusNoContent, ""),
	)
	rules := []string{"r2", "r1"}
	policy, err := Create(context.Background(), client, CreateRequest{FirewallRules: &rules})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(policy.FirewallRules, ",") != "r2,r1" {
		t.Fatalf("rules=%v", policy.FirewallRules)
	}
	if _, err = Get(context.Background(), client, "fp-id"); err != nil {
		t.Fatal(err)
	}
	if _, err = Update(context.Background(), client, "fp-id", UpdateRequest{FirewallRules: &rules}); err != nil {
		t.Fatal(err)
	}
	if err = Delete(context.Background(), client, "fp-id"); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 4 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
	for _, index := range []int{0, 2} {
		body, _ := io.ReadAll(transport.requests[index].Body)
		if !strings.Contains(string(body), `"firewall_rules":["r2","r1"]`) {
			t.Fatalf("body=%s", body)
		}
		if strings.Contains(string(body), "shared") {
			t.Fatalf("body=%s", body)
		}
	}
}

func TestFirewallPolicyFieldsKeepAuditedCallerControlled(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"firewall_policy":{"id":"fp-id","audited":false}}`),
		response(http.StatusOK, `{"firewall_policy":{"id":"fp-id","audited":true}}`),
	)
	name := "renamed"
	policy, err := Update(context.Background(), client, "fp-id", UpdateRequest{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if policy.Audited {
		t.Fatalf("policy=%+v", policy)
	}
	audited := true
	rules := []string{"r3", "r1"}
	policy, err = Update(context.Background(), client, "fp-id", UpdateRequest{
		Name: &name, FirewallRules: &rules, Audited: &audited,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Audited {
		t.Fatalf("policy=%+v", policy)
	}
	firstBody, _ := io.ReadAll(transport.requests[0].Body)
	secondBody, _ := io.ReadAll(transport.requests[1].Body)
	if strings.Contains(string(firstBody), "audited") {
		t.Fatalf("first body=%s", firstBody)
	}
	if !strings.Contains(string(secondBody), `"audited":true`) ||
		!strings.Contains(string(secondBody), `"firewall_rules":["r3","r1"]`) {
		t.Fatalf("second body=%s", secondBody)
	}
	if len(transport.requests) != 2 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
}

func TestFirewallPolicyListWalksAllPages(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"firewall_policies":[{"id":"one"}],"firewall_policies_links":[{"rel":"next","href":"?marker=one"}]}`),
		response(http.StatusOK, `{"firewall_policies":[{"id":"two"}],"firewall_policies_links":[]}`),
	)
	policies, err := List(context.Background(), client, vpc.ListOptions{Limit: 1})
	if err != nil || len(policies) != 2 || len(transport.requests) != 2 {
		t.Fatalf("policies=%+v requests=%d error=%v", policies, len(transport.requests), err)
	}
}

func TestFirewallPolicyCRUDErrorsAreNotRewritten(t *testing.T) {
	for _, testCase := range []struct {
		status int
		class  vpc.ErrorClass
	}{
		{http.StatusNotFound, vpc.ErrorClassNotFound},
		{http.StatusConflict, vpc.ErrorClassConflict},
	} {
		client, transport := newClient(t, response(
			testCase.status,
			`{"NeutronError":{"type":"FirewallPolicyInUse","message":"diagnostic"}}`,
		))
		err := Delete(context.Background(), client, "fp-id")
		if !vpc.IsErrorClass(err, testCase.class) || len(transport.requests) != 1 {
			t.Fatalf("error=%v requests=%d", err, len(transport.requests))
		}
	}
}
