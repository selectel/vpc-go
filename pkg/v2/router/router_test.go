package router

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

func response(status int, body string) *http.Response {
	return testutil.Response(status, body)
}

func newClient(t *testing.T, responses ...*http.Response) (*vpc.Client, *testutil.Transport) {
	return testutil.NewClient(t, responses...)
}

const routerModel = `{"router":{"id":"id","status":"ACTIVE","blocked":true,"ha":true,` +
	`"external_gateway_info":{"network_id":"ext","enable_snat":true,` +
	`"external_fixed_ips":[{"subnet_id":"sub","ip_address":"192.0.2.1"}]}}}`

func TestRouterCreate(t *testing.T) {
	client, transport := newClient(t, response(http.StatusCreated, routerModel))
	enable := true
	created, err := Create(context.Background(), client, CreateRequest{
		ExternalGateway: vpc.Value(ExternalGatewayRequest{NetworkID: "ext", EnableSNAT: &enable}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "id" {
		t.Fatalf("created router = %+v", created)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/routers")
	testutil.AssertJSONBody(t, transport.Requests[0],
		`{"router":{"external_gateway_info":{"network_id":"ext","enable_snat":true}}}`)
}

func TestRouterCreateWithHA(t *testing.T) {
	client, transport := newClient(t, response(http.StatusCreated, routerModel))
	ha := true
	if _, err := Create(context.Background(), client, CreateRequest{HA: &ha}); err != nil {
		t.Fatal(err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/routers")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"router":{"ha":true}}`)
}

func TestRouterCreateOmitsNullHA(t *testing.T) {
	client, transport := newClient(t, response(http.StatusCreated, routerModel))
	if _, err := Create(context.Background(), client, CreateRequest{}); err != nil {
		t.Fatal(err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/routers")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"router":{}}`)
}

func TestRouterGet(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, routerModel))
	got, err := Get(context.Background(), client, "id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "id" || got.Status != "ACTIVE" || !got.HA {
		t.Fatalf("Get() = %+v", got)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/routers/id")
}

func TestRouterUpdateHA(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, routerModel))
	ha := false
	if _, err := Update(context.Background(), client, "id", UpdateRequest{HA: &ha}); err != nil {
		t.Fatal(err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/routers/id")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"router":{"ha":false}}`)
}

func TestRouterUpdateClearsExternalGateway(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, routerModel))
	if _, err := Update(context.Background(), client, "id", UpdateRequest{ExternalGateway: vpc.Null[ExternalGatewayRequest]()}); err != nil {
		t.Fatal(err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/routers/id")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"router":{"external_gateway_info":null}}`)
}

func TestRouterDelete(t *testing.T) {
	client, transport := newClient(t, response(http.StatusNoContent, ""))
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatal(err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodDelete, "/v2.0/routers/id")
}

func TestRouterList(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK,
		`{"routers":[{"id":"id"}],"routers_links":[]}`))
	if routers, err := List(context.Background(), client, vpc.ListOptions{}); err != nil || len(routers) != 1 {
		t.Fatalf("List=%+v,%v", routers, err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/routers")
}

func TestRouterTagsGet(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, `{"tags":[]}`))
	if _, err := TagOperations(client, "id").Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/routers/id/tags")
}
