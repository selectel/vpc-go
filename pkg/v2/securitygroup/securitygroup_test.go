package securitygroup

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

func newClient(t *testing.T, responses ...*http.Response) (*vpc.Client, *testutil.Transport) {
	return testutil.NewClient(t, responses...)
}

func response(status int, body string) *http.Response {
	return testutil.Response(status, body)
}

const securityGroupModel = `{"security_group":{"id":"sg-id","name":"web","description":"web traffic",` +
	`"project_id":"project","security_group_rules":[{"id":"default-rule"}]}}`

const securityGroupName = "web"

func TestSecurityGroupCreate(t *testing.T) {
	client, transport := newClient(t, response(http.StatusCreated, securityGroupModel))
	name, description := securityGroupName, "web traffic"
	group, err := Create(context.Background(), client, CreateRequest{
		Name: &name, Description: &description,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(group.SecurityGroupRules) != 1 || group.SecurityGroupRules[0].ID != "default-rule" {
		t.Fatalf("group=%+v", group)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/security-groups")
	testutil.AssertJSONBody(t, transport.Requests[0],
		`{"security_group":{"name":"web","description":"web traffic"}}`)
}

func TestSecurityGroupGet(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, securityGroupModel))
	got, err := Get(context.Background(), client, "sg-id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "sg-id" || got.Name != securityGroupName {
		t.Fatalf("Get() = %+v", got)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/security-groups/sg-id")
}

func TestSecurityGroupUpdate(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, securityGroupModel))
	name := securityGroupName
	if _, err := Update(context.Background(), client, "sg-id", UpdateRequest{Name: &name}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/security-groups/sg-id")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"security_group":{"name":"web"}}`)
}

func TestSecurityGroupDelete(t *testing.T) {
	client, transport := newClient(t, response(http.StatusNoContent, ""))
	if err := Delete(context.Background(), client, "sg-id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodDelete, "/v2.0/security-groups/sg-id")
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
	testutil.AssertJSONBody(t, transport.Requests[0], `{"security_group":{"stateful":false}}`)
	testutil.AssertJSONBody(t, transport.Requests[1], `{"security_group":{"stateful":false}}`)
}

func TestSecurityGroupOmittedStatefulKeepsAPIDefault(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusCreated, `{"security_group":{"id":"sg-id","stateful":true}}`),
	)
	if _, err := Create(context.Background(), client, CreateRequest{}); err != nil {
		t.Fatal(err)
	}
	// An omitted pointer must not serialise as false, which would silently make
	// every group stateless.
	testutil.AssertJSONBody(t, transport.Requests[0], `{"security_group":{}}`)
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
	if len(groups) != 2 || len(transport.Requests) != 2 {
		t.Fatalf("groups=%+v requests=%d", groups, len(transport.Requests))
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
	if len(transport.Requests) != 1 ||
		transport.Requests[0].Method != http.MethodPut ||
		transport.Requests[0].URL.Path != "/v2.0/security-groups/sg-id/tags" {
		t.Fatalf("request=%+v", transport.Requests)
	}
}
