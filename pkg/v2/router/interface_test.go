package router

import (
	"context"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
)

func TestRouterInterfaceRequestsAndPaths(t *testing.T) {
	result := `{"id":"router","port_id":"port","subnet_id":"subnet","network_id":"network"}`
	client, transport := newClient(t, response(200, result), response(200, result))

	added, err := AddInterface(context.Background(), client, "router", InterfaceRequest{SubnetID: "subnet"})
	if err != nil {
		t.Fatal(err)
	}
	if added.PortID != "port" || added.NetworkID != "network" {
		t.Fatalf("result=%+v", added)
	}
	if _, err := RemoveInterface(
		context.Background(), client, "router", InterfaceRequest{PortID: "port"},
	); err != nil {
		t.Fatal(err)
	}

	if transport.Requests[0].URL.Path != "/v2.0/routers/router/add_router_interface" ||
		transport.Requests[1].URL.Path != "/v2.0/routers/router/remove_router_interface" {
		t.Fatalf("paths=%s,%s", transport.Requests[0].URL.Path, transport.Requests[1].URL.Path)
	}
	testutil.AssertJSONBody(t, transport.Requests[0], `{"subnet_id":"subnet"}`)
	testutil.AssertJSONBody(t, transport.Requests[1], `{"port_id":"port"}`)
}

func TestRouterRoutesReplaceAndOmit(t *testing.T) {
	model := `{"router":{"id":"router","routes":[]}}`
	client, transport := newClient(t, response(200, model), response(200, model))
	name := "renamed"
	if _, err := Update(context.Background(), client, "router", UpdateRequest{Name: &name}); err != nil {
		t.Fatal(err)
	}
	empty := []Route{}
	if _, err := Update(context.Background(), client, "router", UpdateRequest{Routes: &empty}); err != nil {
		t.Fatal(err)
	}
	testutil.AssertJSONBody(t, transport.Requests[0], `{"router":{"name":"renamed"}}`)
	testutil.AssertJSONBody(t, transport.Requests[1], `{"router":{"routes":[]}}`)
	if len(transport.Requests) != 2 {
		t.Fatalf("requests=%d", len(transport.Requests))
	}
}
