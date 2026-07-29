package firewallrule

import (
	"context"
	"io"
	"net/http"
	"reflect"
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

func TestFirewallRuleCRUDDoesNotManagePolicies(t *testing.T) {
	created := `{"firewall_rule":{"id":"rule-id","firewall_policy_id":[],"shared":false}}`
	read := `{"firewall_rule":{"id":"rule-id","firewall_policy_id":["policy-one","policy-two"]}}`
	client, transport := newClient(
		t,
		response(http.StatusCreated, created),
		response(http.StatusOK, read),
		response(http.StatusOK, created),
		response(http.StatusNoContent, ""),
	)
	destinationPort := "443:8443"
	rule, err := Create(context.Background(), client, CreateRequest{
		DestinationPort: vpc.Value(destinationPort),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rule.FirewallPolicyIDs) != 0 {
		t.Fatalf("created rule=%+v", rule)
	}
	rule, err = Get(context.Background(), client, "rule-id")
	if err != nil || len(rule.FirewallPolicyIDs) != 2 {
		t.Fatalf("read rule=%+v error=%v", rule, err)
	}
	if _, err = Update(context.Background(), client, "rule-id", UpdateRequest{}); err != nil {
		t.Fatal(err)
	}
	if err = Delete(context.Background(), client, "rule-id"); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 4 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
	body, _ := io.ReadAll(transport.requests[0].Body)
	if !strings.Contains(string(body), `"destination_port":"443:8443"`) {
		t.Fatalf("create body=%s", body)
	}
	for _, forbidden := range []string{
		"shared", "position", "firewall_policy_id",
		"source_firewall_group_id", "destination_firewall_group_id",
	} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("create body contains %s: %s", forbidden, body)
		}
	}
}

func TestFirewallRuleListKeepsServicePolicyView(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"firewall_rules":[{"id":"rule-id","firewall_policy_id":[]}],"firewall_rules_links":[{"rel":"next","href":"?marker=rule-id"}]}`),
		response(http.StatusOK, `{"firewall_rules":[{"id":"other","firewall_policy_id":[]}],"firewall_rules_links":[]}`),
	)
	rules, err := List(context.Background(), client, vpc.ListOptions{Limit: 1})
	if err != nil || len(rules) != 2 || len(rules[0].FirewallPolicyIDs) != 0 {
		t.Fatalf("rules=%+v error=%v", rules, err)
	}
	if len(transport.requests) != 2 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
}

func TestFirewallRuleExplicitEmptySourceAddress(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"firewall_rule":{"id":"rule-id"}}`),
		response(http.StatusOK, `{"firewall_rule":{"id":"rule-id","source_ip_address":""}}`),
	)
	name := "renamed"
	if _, err := Update(context.Background(), client, "rule-id", UpdateRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(context.Background(), client, "rule-id", UpdateRequest{
		SourceIPAddress: vpc.Value(""),
	}); err != nil {
		t.Fatal(err)
	}
	firstBody, _ := io.ReadAll(transport.requests[0].Body)
	secondBody, _ := io.ReadAll(transport.requests[1].Body)
	if strings.Contains(string(firstBody), "source_ip_address") ||
		!strings.Contains(string(secondBody), `"source_ip_address":""`) {
		t.Fatalf("first=%s second=%s", firstBody, secondBody)
	}
}

func TestFirewallRulePublicTypesExcludeUnsupportedFields(t *testing.T) {
	for _, value := range []any{FirewallRule{}, CreateRequest{}, UpdateRequest{}} {
		typ := reflect.TypeOf(value)
		for _, field := range []string{"Position", "SourceFirewallGroupID", "DestinationFirewallGroupID"} {
			if _, exists := typ.FieldByName(field); exists {
				t.Fatalf("%s exposes %s", typ, field)
			}
		}
	}
	for _, value := range []any{CreateRequest{}, UpdateRequest{}} {
		if _, exists := reflect.TypeOf(value).FieldByName("Shared"); exists {
			t.Fatalf("%T exposes Shared", value)
		}
	}
}

func TestFirewallRuleDeleteErrorsRemainDistinct(t *testing.T) {
	for _, testCase := range []struct {
		status int
		class  vpc.ErrorClass
	}{
		{http.StatusNotFound, vpc.ErrorClassNotFound},
		{http.StatusConflict, vpc.ErrorClassConflict},
	} {
		client, transport := newClient(t, response(
			testCase.status,
			`{"NeutronError":{"type":"FirewallRuleInUse","message":"diagnostic"}}`,
		))
		err := Delete(context.Background(), client, "rule-id")
		if !vpc.IsErrorClass(err, testCase.class) || len(transport.requests) != 1 {
			t.Fatalf("error=%v requests=%d", err, len(transport.requests))
		}
	}
}
