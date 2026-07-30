package addressscope

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

func TestAddressScopeCRUDListAndFields(t *testing.T) {
	model := `{"address_scope":{"id":"id","name":"scope","ip_version":4,` +
		`"shared":false,"project_id":"project"}}`
	client, transport := newClient(
		t,
		response(201, model), response(200, model), response(200, model), response(204, ""),
		response(200, `{"address_scopes":[{"id":"id"}],"address_scopes_links":[]}`),
	)
	name, version := "scope", 4
	if _, err := Create(context.Background(), client, CreateRequest{
		Name: &name, IPVersion: version,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Get(context.Background(), client, "id"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := Update(context.Background(), client, "id", UpdateRequest{Name: &name}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if scopes, err := List(context.Background(), client, vpc.ListOptions{}); err != nil ||
		len(scopes) != 1 {
		t.Fatalf("List() = %+v, %v", scopes, err)
	}

	createBody, _ := io.ReadAll(transport.requests[0].Body)
	updateBody, _ := io.ReadAll(transport.requests[2].Body)
	if strings.Contains(string(createBody), `"shared"`) ||
		strings.Contains(string(updateBody), `"shared"`) ||
		strings.Contains(string(updateBody), `"ip_version"`) {
		t.Fatalf("invalid DTO body: create=%s update=%s", createBody, updateBody)
	}
}

func TestAddressScopeErrorClassesAreNotRewritten(t *testing.T) {
	tests := []struct {
		operation string
		status    int
		kind      string
		class     vpc.ErrorClass
	}{
		{"create", 403, "PolicyNotAuthorized", vpc.ErrorClassForbidden},
		{"update", 404, "AddressScopeNotFound", vpc.ErrorClassNotFound},
		{"delete", 404, "AddressScopeNotFound", vpc.ErrorClassNotFound},
		{"delete", 409, "AddressScopeInUse", vpc.ErrorClassConflict},
	}

	for _, test := range tests {
		client, transport := newClient(t, response(
			test.status,
			`{"NeutronError":{"type":"`+test.kind+`","message":"failure"}}`,
		))
		var err error
		switch test.operation {
		case "create":
			_, err = Create(context.Background(), client, CreateRequest{})
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
