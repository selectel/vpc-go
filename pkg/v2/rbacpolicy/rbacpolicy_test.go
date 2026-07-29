package rbacpolicy

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
		t.Fatalf("NewClient() error = %v", err)
	}
	return client, transport
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func TestRBACPolicyCRUDAndNonPaginatedList(t *testing.T) {
	model := `{"rbac_policy":{"id":"id","object_type":"custom_type",` +
		`"object_id":"object","action":"custom_action","target_tenant":"target"}}`
	client, transport := newClient(
		t,
		response(201, model), response(200, model), response(200, model), response(204, ""),
		response(200, `{"rbac_policies":[{"id":"id"}]}`),
	)
	create := CreateRequest{
		ObjectType: "custom_type", ObjectID: "object",
		Action: "custom_action", TargetTenant: "target",
	}
	if _, err := Create(context.Background(), client, create); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Get(context.Background(), client, "id"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := Update(
		context.Background(), client, "id", UpdateRequest{TargetTenant: "other"},
	); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	policies, err := List(context.Background(), client, vpc.SelectionOptions{
		Fields: []string{"id"}, SortKey: "object_id", SortDir: "asc",
	})
	if err != nil || len(policies) != 1 {
		t.Fatalf("List() = %+v, %v", policies, err)
	}
	query := transport.requests[4].URL.Query()
	if query.Has("limit") || query.Has("marker") {
		t.Fatalf("RBAC list query exposes pagination: %v", query)
	}
	updateBody, _ := io.ReadAll(transport.requests[2].Body)
	for _, forbidden := range []string{"object_type", "object_id", "action", "project_id"} {
		if strings.Contains(string(updateBody), forbidden) {
			t.Fatalf("update body contains %s: %s", forbidden, updateBody)
		}
	}
}

func TestRBACPolicyErrorClasses(t *testing.T) {
	tests := []struct {
		operation string
		status    int
		kind      string
		class     vpc.ErrorClass
	}{
		{"create", 409, "RbacPolicyDuplicate", vpc.ErrorClassConflict},
		{"update", 409, "RbacPolicyInUse", vpc.ErrorClassConflict},
		{"delete", 409, "RbacPolicyInUse", vpc.ErrorClassConflict},
		{"delete", 404, "RbacPolicyNotFound", vpc.ErrorClassNotFound},
		{"create", 403, "PolicyNotAuthorized", vpc.ErrorClassForbidden},
		{"update", 404, "RbacPolicyNotFound", vpc.ErrorClassNotFound},
	}

	for _, test := range tests {
		client, transport := newClient(t, response(
			test.status,
			`{"NeutronError":{"type":"`+test.kind+`","message":"failure"}}`,
		))
		var err error
		switch test.operation {
		case "create":
			_, err = Create(context.Background(), client, CreateRequest{
				ObjectType: "unknown", Action: "unknown",
			})
		case "update":
			_, err = Update(context.Background(), client, "id", UpdateRequest{})
		default:
			err = Delete(context.Background(), client, "id")
		}
		if !vpc.IsErrorClass(err, test.class) {
			t.Fatalf("%s error = %v, want %s", test.operation, err, test.class)
		}
		if len(transport.requests) != 1 {
			t.Fatalf("%s request count = %d, want 1", test.operation, len(transport.requests))
		}
	}
}
