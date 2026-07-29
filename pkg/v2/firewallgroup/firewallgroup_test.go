package firewallgroup

import (
	"context"
	"errors"
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

func TestFirewallGroupCRUDReturnsCurrentStatus(t *testing.T) {
	model := `{"firewall_group":{"id":"fg-id","name":"edge","status":"PENDING_CREATE","shared":false}}`
	client, transport := newClient(
		t,
		response(http.StatusCreated, model),
		response(http.StatusOK, model),
		response(http.StatusOK, model),
		response(http.StatusNoContent, ""),
	)
	name := "edge"
	group, err := Create(context.Background(), client, CreateRequest{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if group.Status != "PENDING_CREATE" || group.Shared {
		t.Fatalf("group=%+v", group)
	}
	if _, err = Get(context.Background(), client, "fg-id"); err != nil {
		t.Fatal(err)
	}
	if _, err = Update(context.Background(), client, "fg-id", UpdateRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if err = Delete(context.Background(), client, "fg-id"); err != nil {
		t.Fatal(err)
	}
	if len(transport.requests) != 4 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
	for _, request := range transport.requests {
		if !strings.HasPrefix(request.URL.Path, "/v2.0/fwaas/firewall_groups") {
			t.Fatalf("path=%s", request.URL.Path)
		}
	}
	body, _ := io.ReadAll(transport.requests[0].Body)
	if strings.Contains(string(body), "shared") {
		t.Fatalf("create body=%s", body)
	}
}

func TestFirewallGroupPolicyNullAndEmptyPortsAreExplicit(t *testing.T) {
	emptyPorts := []string{}
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"firewall_group":{"id":"fg-id"}}`),
		response(http.StatusOK, `{"firewall_group":{"id":"fg-id"}}`),
	)
	name := "renamed"
	if _, err := Update(context.Background(), client, "fg-id", UpdateRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(context.Background(), client, "fg-id", UpdateRequest{
		Ports:                   &emptyPorts,
		IngressFirewallPolicyID: vpc.Null[string](),
	}); err != nil {
		t.Fatal(err)
	}
	firstBody, _ := io.ReadAll(transport.requests[0].Body)
	secondBody, _ := io.ReadAll(transport.requests[1].Body)
	if strings.Contains(string(firstBody), "ingress_firewall_policy_id") {
		t.Fatalf("first body=%s", firstBody)
	}
	if !strings.Contains(string(secondBody), `"ports":[]`) ||
		!strings.Contains(string(secondBody), `"ingress_firewall_policy_id":null`) {
		t.Fatalf("second body=%s", secondBody)
	}
	if len(transport.requests) != 2 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
}

func TestFirewallGroupListAndIncompleteList(t *testing.T) {
	client, _ := newClient(
		t,
		response(http.StatusOK, `{"firewall_groups":[{"id":"one"}],"firewall_groups_links":[{"rel":"next","href":"?marker=one"}]}`),
		response(http.StatusOK, `{"firewall_groups":[{"id":"two"}],"firewall_groups_links":[]}`),
	)
	groups, err := List(context.Background(), client, vpc.ListOptions{Limit: 1})
	if err != nil || len(groups) != 2 {
		t.Fatalf("groups=%+v error=%v", groups, err)
	}

	client, _ = newClient(
		t,
		response(http.StatusOK, `{"firewall_groups":[{"id":"one"}],"firewall_groups_links":[{"rel":"next","href":"?marker=one"}]}`),
		response(http.StatusInternalServerError, `{"NeutronError":{"message":"failure"}}`),
	)
	_, err = List(context.Background(), client, vpc.ListOptions{})
	if !vpc.IsIncompleteList(err) {
		t.Fatalf("error=%v", err)
	}
}

func TestFirewallGroupErrorsPreserveClassStatusAndMessage(t *testing.T) {
	client, transport := newClient(t, response(
		http.StatusConflict,
		`{"NeutronError":{"type":"FirewallGroupInPendingState","message":"try later","status":"PENDING_UPDATE"}}`,
	))
	_, err := Update(context.Background(), client, "fg-id", UpdateRequest{})
	var apiErr *vpc.APIError
	if !errors.As(err, &apiErr) ||
		apiErr.Class != vpc.ErrorClassConflict ||
		apiErr.ResourceStatus != "PENDING_UPDATE" ||
		apiErr.Message != "try later" {
		t.Fatalf("error=%+v", err)
	}
	if len(transport.requests) != 1 {
		t.Fatalf("requests=%d", len(transport.requests))
	}

	client, _ = newClient(t, response(
		http.StatusNotFound,
		`{"NeutronError":{"type":"FirewallGroupNotFound","message":"missing"}}`,
	))
	err = Delete(context.Background(), client, "fg-id")
	if !vpc.IsErrorClass(err, vpc.ErrorClassNotFound) {
		t.Fatalf("error=%v", err)
	}
}
