package securitygroup

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
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client, transport
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func TestSecurityGroupCRUDAndFields(t *testing.T) {
	model := `{"security_group":{"id":"sg-id","name":"web","description":"web traffic",` +
		`"project_id":"project","security_group_rules":[{"id":"default-rule"}]}}`
	client, transport := newClient(
		t,
		response(http.StatusCreated, model),
		response(http.StatusOK, model),
		response(http.StatusOK, model),
		response(http.StatusNoContent, ""),
	)
	name, description := "web", "web traffic"
	group, err := Create(context.Background(), client, CreateRequest{
		Name: &name, Description: &description,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(group.SecurityGroupRules) != 1 || group.SecurityGroupRules[0].ID != "default-rule" {
		t.Fatalf("group=%+v", group)
	}
	if _, err = Get(context.Background(), client, "sg-id"); err != nil {
		t.Fatal(err)
	}
	if _, err = Update(context.Background(), client, "sg-id", UpdateRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if err = Delete(context.Background(), client, "sg-id"); err != nil {
		t.Fatal(err)
	}
	for i, method := range []string{http.MethodPost, http.MethodGet, http.MethodPut, http.MethodDelete} {
		if transport.requests[i].Method != method {
			t.Fatalf("request %d method=%s", i, transport.requests[i].Method)
		}
	}
	if len(transport.requests) != 4 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
	createBody, _ := io.ReadAll(transport.requests[0].Body)
	if strings.Contains(string(createBody), "security_group_rules") {
		t.Fatalf("create body=%s", createBody)
	}
	if _, exists := reflect.TypeOf(CreateRequest{}).FieldByName("SecurityGroupRules"); exists {
		t.Fatal("CreateRequest must not expose SecurityGroupRules")
	}
}

func TestSecurityGroupStatefulRoundTrips(t *testing.T) {
	model := `{"security_group":{"id":"sg-id","name":"web","stateful":false,"shared":true}}`
	client, transport := newClient(
		t,
		response(http.StatusCreated, model),
		response(http.StatusOK, model),
	)
	stateful := false
	group, err := Create(context.Background(), client, CreateRequest{Stateful: &stateful})
	if err != nil {
		t.Fatal(err)
	}
	if group.Stateful {
		t.Fatalf("Stateful=%t, want false", group.Stateful)
	}
	if !group.Shared {
		t.Fatalf("Shared=%t, want true", group.Shared)
	}
	if _, err = Update(context.Background(), client, "sg-id", UpdateRequest{Stateful: &stateful}); err != nil {
		t.Fatal(err)
	}
	createBody, _ := io.ReadAll(transport.requests[0].Body)
	updateBody, _ := io.ReadAll(transport.requests[1].Body)
	for _, body := range []string{string(createBody), string(updateBody)} {
		if !strings.Contains(body, `"stateful":false`) {
			t.Fatalf("body=%s, want an explicit stateful=false", body)
		}
		// shared is read-only on the API; sending it would be rejected.
		if strings.Contains(body, "shared") {
			t.Fatalf("body=%s must not contain shared", body)
		}
	}
}

func TestSecurityGroupOmittedStatefulKeepsAPIDefault(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusCreated, `{"security_group":{"id":"sg-id","stateful":true}}`),
	)
	if _, err := Create(context.Background(), client, CreateRequest{}); err != nil {
		t.Fatal(err)
	}
	createBody, _ := io.ReadAll(transport.requests[0].Body)
	// An omitted pointer must not serialise as false, which would silently make
	// every group stateless.
	if strings.Contains(string(createBody), "stateful") {
		t.Fatalf("create body=%s must omit stateful entirely", createBody)
	}
}

func TestSecurityGroupSharedIsReadOnly(t *testing.T) {
	for _, request := range []any{CreateRequest{}, UpdateRequest{}} {
		if _, exists := reflect.TypeOf(request).FieldByName("Shared"); exists {
			t.Fatalf("%T must not expose Shared: it is read-only on the API", request)
		}
	}
}

func TestSecurityGroupListWalksAllPages(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"security_groups":[{"id":"one"}],"security_groups_links":[{"rel":"next","href":"https://ignored.invalid/v2.0/security-groups?marker=one"}]}`),
		response(http.StatusOK, `{"security_groups":[{"id":"two"}],"security_groups_links":[]}`),
	)
	groups, err := List(context.Background(), client, vpc.ListOptions{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || len(transport.requests) != 2 {
		t.Fatalf("groups=%+v requests=%d", groups, len(transport.requests))
	}
}

func TestSecurityGroupTagsUseHyphenatedPath(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, `{"tags":["one","two"]}`))
	tags, err := TagOperations(client, "sg-id").Replace(context.Background(), []string{"one", "two"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0] != "one" || tags[1] != "two" {
		t.Fatalf("Replace()=%v", tags)
	}
	if len(transport.requests) != 1 ||
		transport.requests[0].Method != http.MethodPut ||
		transport.requests[0].URL.Path != "/v2.0/security-groups/sg-id/tags" {
		t.Fatalf("request=%+v", transport.requests)
	}
}

func TestSecurityGroupCRUDConflictErrorsAreNotRewritten(t *testing.T) {
	for _, operation := range []string{"update default", "delete default", "delete in use"} {
		t.Run(operation, func(t *testing.T) {
			client, transport := newClient(t, response(
				http.StatusConflict,
				`{"NeutronError":{"type":"SecurityGroupConflict","message":"group cannot be changed"}}`,
			))
			var err error
			if operation == "update default" {
				_, err = Update(context.Background(), client, "default", UpdateRequest{})
			} else {
				err = Delete(context.Background(), client, "sg-id")
			}
			if !vpc.IsErrorClass(err, vpc.ErrorClassConflict) ||
				!strings.Contains(err.Error(), "group cannot be changed") {
				t.Fatalf("error=%v", err)
			}
			if len(transport.requests) != 1 {
				t.Fatalf("requests=%d", len(transport.requests))
			}
		})
	}
}
