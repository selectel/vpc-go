package subnetpool

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const subnetPoolModel = `{"subnetpool":{"id":"id","prefixes":["192.0.2.0/24"],` +
	`"ip_version":4,"is_default":false,"shared":true}}`

func TestSubnetPoolCreate(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusCreated, subnetPoolModel))
	prefixes := []string{"192.0.2.0/24"}
	created, err := Create(context.Background(), client, CreateRequest{Prefixes: prefixes})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != "id" || !created.Shared {
		t.Fatalf("Create() = %+v", created)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/subnetpools")
	testutil.AssertJSONBody(t, transport.Requests[0],
		`{"subnetpool":{"prefixes":["192.0.2.0/24"]}}`)
}

func TestSubnetPoolGet(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, subnetPoolModel))
	got, err := Get(context.Background(), client, "id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "id" || len(got.Prefixes) != 1 {
		t.Fatalf("Get() = %+v", got)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/subnetpools/id")
}

func TestSubnetPoolUpdateReplacesPrefixes(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, subnetPoolModel))
	prefixes := []string{"192.0.2.0/24"}
	if _, err := Update(
		context.Background(), client, "id", UpdateRequest{
			Prefixes: &prefixes,
		},
	); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/subnetpools/id")
	testutil.AssertJSONBody(t, transport.Requests[0],
		`{"subnetpool":{"prefixes":["192.0.2.0/24"]}}`)
}

func TestSubnetPoolDelete(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusNoContent, ""))
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodDelete, "/v2.0/subnetpools/id")
}

func TestSubnetPoolListIncludesServiceResponse(t *testing.T) {
	client, _ := testutil.NewClient(t, testutil.Response(200,
		`{"subnetpools":[{"id":"project","shared":false},{"id":"public","shared":true}],`+
			`"subnetpools_links":[]}`,
	))
	pools, err := List(context.Background(), client, vpc.ListOptions{})
	if err != nil || len(pools) != 2 || !pools[1].Shared {
		t.Fatalf("List() = %+v, %v", pools, err)
	}
}

func TestSubnetPoolTagsReplace(t *testing.T) {
	client, transport := testutil.NewClient(t,
		testutil.Response(http.StatusOK, `{"tags":["two"]}`))
	values, err := TagOperations(client, "id").Replace(context.Background(), []string{"two"})
	if err != nil || len(values) != 1 || values[0] != "two" {
		t.Fatalf("tags.Replace() = %+v, %v", values, err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/subnetpools/id/tags")
}
