package rbacpolicy

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

const rbacPolicyModel = `{"rbac_policy":{"id":"id","object_type":"custom_type",` +
	`"object_id":"object","action":"custom_action","target_tenant":"target"}}`

func TestRBACPolicyCreate(t *testing.T) {
	client, transport := newClient(t, response(http.StatusCreated, rbacPolicyModel))
	create := CreateRequest{
		ObjectType: "custom_type", ObjectID: "object",
		Action: "custom_action", TargetTenant: "target",
	}
	created, err := Create(context.Background(), client, create)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != "id" || created.TargetTenant != "target" {
		t.Fatalf("Create() = %+v", created)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/rbac-policies")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"rbac_policy":{`+
		`"object_type":"custom_type","object_id":"object",`+
		`"action":"custom_action","target_tenant":"target"}}`)
}

func TestRBACPolicyGet(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, rbacPolicyModel))
	got, err := Get(context.Background(), client, "id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "id" || got.ObjectID != "object" {
		t.Fatalf("Get() = %+v", got)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/rbac-policies/id")
}

func TestRBACPolicyUpdateOnlySendsTargetTenant(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, rbacPolicyModel))
	if _, err := Update(
		context.Background(), client, "id", UpdateRequest{TargetTenant: "other"},
	); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/rbac-policies/id")
	testutil.AssertJSONBody(t, transport.Requests[0],
		`{"rbac_policy":{"target_tenant":"other"}}`)
}

func TestRBACPolicyDelete(t *testing.T) {
	client, transport := newClient(t, response(http.StatusNoContent, ""))
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodDelete, "/v2.0/rbac-policies/id")
}

func TestRBACPolicyList(t *testing.T) {
	client, transport := newClient(t,
		response(http.StatusOK, `{"rbac_policies":[{"id":"one"}],`+
			`"rbac_policies_links":[{"rel":"next",`+
			`"href":"https://ignored.invalid/v2.0/rbac-policies?fields=id&marker=one"}]}`),
		response(http.StatusOK, `{"rbac_policies":[{"id":"two"}],`+
			`"rbac_policies_links":[]}`),
	)
	policies, err := List(context.Background(), client, vpc.SelectionOptions{
		Fields: []string{"id"}, SortKey: "object_id", SortDir: "asc",
	})
	if err != nil || len(policies) != 2 {
		t.Fatalf("List() = %+v, %v", policies, err)
	}
	query := transport.Requests[0].URL.Query()
	if query.Get("fields") != "id" || query.Get("sort_key") != "object_id" ||
		query.Get("sort_dir") != "asc" {
		t.Fatalf("List() query = %v", query)
	}
	if len(transport.Requests) != 2 || transport.Requests[1].URL.Query().Get("marker") != "one" {
		t.Fatalf("List() requests = %+v", transport.Requests)
	}
}
