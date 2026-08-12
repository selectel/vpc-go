package subnet

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const subnetModel = `{"subnet":{"id":"id","network_id":"network","ip_version":4,` +
	`"cidr":"192.0.2.0/24","service_types":["network:floatingip_agent_gateway"],` +
	`"blocked":true}}`

func TestSubnetCreate(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusCreated, subnetModel))
	cidr := "192.0.2.0/24"
	created, err := Create(context.Background(), client, CreateRequest{
		NetworkID: "network", IPVersion: 4, CIDR: &cidr,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created.Blocked {
		t.Fatalf("created subnet = %+v", created)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/subnets")
	testutil.AssertJSONBody(t, transport.Requests[0],
		`{"subnet":{"network_id":"network","ip_version":4,"cidr":"192.0.2.0/24"}}`)
}

func TestSubnetGet(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, subnetModel))
	got, err := Get(context.Background(), client, "id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "id" || got.CIDR != "192.0.2.0/24" {
		t.Fatalf("Get() = %+v", got)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/subnets/id")
}

func TestSubnetUpdateCanClearGateway(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, subnetModel))
	if _, err := Update(context.Background(), client, "id", UpdateRequest{
		GatewayIP: vpc.Null[string](),
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/subnets/id")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"subnet":{"gateway_ip":null}}`)
}

func TestSubnetDelete(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusNoContent, ""))
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodDelete, "/v2.0/subnets/id")
}

func TestSubnetListWalksAllPages(t *testing.T) {
	client, transport := testutil.NewClient(
		t,
		testutil.Response(200, `{"subnets":[{"id":"one"}],"subnets_links":[{"rel":"next","href":"?marker=one"}]}`),
		testutil.Response(200, `{"subnets":[{"id":"two"}],"subnets_links":[]}`),
	)
	subnets, err := List(context.Background(), client, vpc.ListOptions{Limit: 1})
	if err != nil || len(subnets) != 2 {
		t.Fatalf("List() = %+v, %v", subnets, err)
	}
	if len(transport.Requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(transport.Requests))
	}
}

func TestSubnetTagsUseCollectionPath(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, `{"tags":[]}`))
	if _, err := TagOperations(client, "id").Replace(context.Background(), nil); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/subnets/id/tags")
}
