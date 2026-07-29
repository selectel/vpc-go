package router

import (
	"context"
	"io"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

func TestRouterInterfaceSelectorsAndPaths(t *testing.T) {
	result := `{"id":"router","port_id":"port","subnet_id":"subnet","network_id":"network"}`
	client, transport := newClient(t, response(200, result), response(200, result))

	added, err := AddInterface(context.Background(), client, "router", BySubnet("subnet"))
	if err != nil {
		t.Fatal(err)
	}
	if added.PortID != "port" || added.NetworkID != "network" {
		t.Fatalf("result=%+v", added)
	}
	if _, err := RemoveInterface(context.Background(), client, "router", ByPort("port")); err != nil {
		t.Fatal(err)
	}

	if transport.requests[0].URL.Path != "/v2.0/routers/router/add_router_interface" ||
		transport.requests[1].URL.Path != "/v2.0/routers/router/remove_router_interface" {
		t.Fatalf("paths=%s,%s", transport.requests[0].URL.Path, transport.requests[1].URL.Path)
	}
	firstBody, _ := io.ReadAll(transport.requests[0].Body)
	secondBody, _ := io.ReadAll(transport.requests[1].Body)
	if string(firstBody) != `{"subnet_id":"subnet"}` || string(secondBody) != `{"port_id":"port"}` {
		t.Fatalf("bodies=%s,%s", firstBody, secondBody)
	}
}

func TestRouterInterfaceErrorClasses(t *testing.T) {
	for _, test := range []struct {
		status int
		kind   string
		class  vpc.ErrorClass
	}{
		{400, "BadRequest", vpc.ErrorClassBadRequest},
		{404, "RouterInterfaceNotFound", vpc.ErrorClassNotFound},
		{409, "RouterInterfaceInUse", vpc.ErrorClassConflict},
	} {
		client, transport := newClient(t, response(test.status,
			`{"NeutronError":{"type":"`+test.kind+`","message":"failure"}}`))
		_, err := AddInterface(context.Background(), client, "router", BySubnet("subnet"))
		if !vpc.IsErrorClass(err, test.class) || len(transport.requests) != 1 {
			t.Fatalf("%s error=%v requests=%d", test.kind, err, len(transport.requests))
		}
	}
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
	firstBody, _ := io.ReadAll(transport.requests[0].Body)
	secondBody, _ := io.ReadAll(transport.requests[1].Body)
	if strings.Contains(string(firstBody), `"routes"`) {
		t.Fatalf("first body=%s", firstBody)
	}
	if !strings.Contains(string(secondBody), `"routes":[]`) {
		t.Fatalf("second body=%s", secondBody)
	}
	if len(transport.requests) != 2 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
}
