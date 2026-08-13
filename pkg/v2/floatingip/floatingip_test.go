package floatingip

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

func response(s int, b string) *http.Response {
	return testutil.Response(s, b)
}

func newClient(t *testing.T, rs ...*http.Response) (*vpc.Client, *testutil.Transport) {
	return testutil.NewClient(t, rs...)
}

const floatingIPModel = `{"floatingip":{"id":"id",` +
	`"floating_ip_address":"203.0.113.1","port_id":"port-id",` +
	`"fixed_ip_address":"192.0.2.10","status":"DOWN","blocked":true}}`

func TestFloatingIPGet(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, floatingIPModel))
	got, err := Get(context.Background(), client, "id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "id" || got.FloatingIPAddress != "203.0.113.1" ||
		got.PortID == nil || *got.PortID != "port-id" ||
		got.FixedIPAddress == nil || *got.FixedIPAddress != "192.0.2.10" {
		t.Fatalf("Get() = %+v", got)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/floatingips/id")
}

func TestFloatingIPUpdateDetachesPort(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK,
		`{"floatingip":{"id":"id","port_id":null,"router_id":null}}`))
	if _, err := Update(context.Background(), client, "id", UpdateRequest{
		PortID: vpc.Null[string](), FixedIPAddress: vpc.Null[string](),
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/floatingips/id")
	testutil.AssertJSONBody(t, transport.Requests[0],
		`{"floatingip":{"port_id":null,"fixed_ip_address":null}}`)
}

func TestFloatingIPList(t *testing.T) {
	client, transport := newClient(t,
		response(http.StatusOK, `{"floatingips":[{"id":"id"}],"floatingips_links":[]}`))
	items, err := List(context.Background(), client, vpc.ListOptions{})
	if err != nil || len(items) != 1 {
		t.Fatalf("List() = %+v, %v", items, err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/floatingips")
}
