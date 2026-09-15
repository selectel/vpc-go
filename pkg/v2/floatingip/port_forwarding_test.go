package floatingip

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const portForwardingModel = `{"port_forwarding":{"id":"pf-id","protocol":"tcp",` +
	`"internal_port_id":"port-id","internal_ip_address":"192.0.2.10",` +
	`"internal_port":8080,"external_port":80,` +
	`"internal_port_range":"8080:8080","external_port_range":"80:80","description":"web"}}`

const portForwardingRangeModel = `{"port_forwarding":{"id":"pf-id","protocol":"tcp",` +
	`"internal_port_id":"port-id","internal_ip_address":"192.0.2.10",` +
	`"internal_port":null,"external_port":null,` +
	`"internal_port_range":"22:24","external_port_range":"2222:2224","description":"ssh"}}`

const escapedPortForwardingPath = "/v2.0/floatingips/fip%2Fid/port_forwardings/pf%2Fid"

func TestPortForwardingCreateUsesNestedPath(t *testing.T) {
	client, transport := newClient(t, response(http.StatusCreated, portForwardingModel))
	description, projectID := "web", "project-id"
	internalPort, externalPort := 8080, 80

	created, err := CreatePortForwarding(context.Background(), client, "fip/id", PortForwardingCreateRequest{
		Protocol:          "tcp",
		InternalPortID:    "port-id",
		InternalIPAddress: "192.0.2.10",
		InternalPort:      &internalPort,
		ExternalPort:      &externalPort,
		Description:       &description,
		ProjectID:         &projectID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "pf-id" || created.ExternalPortRange != "80:80" {
		t.Fatalf("created=%+v", created)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodPost ||
		request.URL.EscapedPath() != "/v2.0/floatingips/fip%2Fid/port_forwardings" {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
	testutil.AssertJSONBody(t, request, `{"port_forwarding":{`+
		`"protocol":"tcp","internal_port_id":"port-id",`+
		`"internal_ip_address":"192.0.2.10","internal_port":8080,`+
		`"external_port":80,"description":"web","project_id":"project-id"}}`)
}

func TestPortForwardingGetUsesNestedPath(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, portForwardingModel))
	got, err := GetPortForwarding(context.Background(), client, "fip/id", "pf/id")
	if err != nil {
		t.Fatalf("GetPortForwarding() error = %v", err)
	}
	if got.ID != "pf-id" || got.ExternalPort == nil || *got.ExternalPort != 80 ||
		got.InternalPortRange != "8080:8080" {
		t.Fatalf("GetPortForwarding() = %+v", got)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodGet ||
		request.URL.EscapedPath() != escapedPortForwardingPath {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
}

func TestPortForwardingUpdateSendsSparseBody(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, portForwardingRangeModel))
	internalRange, externalRange := "22:24", "2222:2224"
	updated, err := UpdatePortForwarding(
		context.Background(),
		client,
		"fip/id",
		"pf/id",
		PortForwardingUpdateRequest{InternalPortRange: &internalRange, ExternalPortRange: &externalRange},
	)
	if err != nil {
		t.Fatalf("UpdatePortForwarding() error = %v", err)
	}
	if updated.InternalPort != nil || updated.ExternalPort != nil ||
		updated.InternalPortRange != "22:24" || updated.ExternalPortRange != "2222:2224" {
		t.Fatalf("UpdatePortForwarding() = %+v", updated)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodPut ||
		request.URL.EscapedPath() != escapedPortForwardingPath {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
	testutil.AssertJSONBody(t, request,
		`{"port_forwarding":{"internal_port_range":"22:24","external_port_range":"2222:2224"}}`)
}

func TestPortForwardingDeleteUsesNestedPath(t *testing.T) {
	client, transport := newClient(t, response(http.StatusNoContent, ""))
	if err := DeletePortForwarding(context.Background(), client, "fip/id", "pf/id"); err != nil {
		t.Fatalf("DeletePortForwarding() error = %v", err)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodDelete ||
		request.URL.EscapedPath() != escapedPortForwardingPath {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
}

func TestPortForwardingListWalksAllPages(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"port_forwardings":[{"id":"one"}],"port_forwardings_links":[{"rel":"next","href":"https://ignored.invalid/v2.0/floatingips/wrong/port_forwardings?marker=one"}]}`),
		response(http.StatusOK, `{"port_forwardings":[{"id":"two"}],"port_forwardings_links":[]}`),
	)

	rules, err := ListPortForwardings(
		context.Background(),
		client,
		"fip-id",
		vpc.ListOptions{Limit: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || rules[0].ID != "one" || rules[1].ID != "two" {
		t.Fatalf("rules=%+v", rules)
	}
	if len(transport.Requests) != 2 {
		t.Fatalf("requests=%d", len(transport.Requests))
	}
	for _, request := range transport.Requests {
		if request.URL.Path != "/v2.0/floatingips/fip-id/port_forwardings" {
			t.Fatalf("path=%s", request.URL.Path)
		}
	}
	if transport.Requests[1].URL.Query().Get("marker") != "one" {
		t.Fatalf("second query=%s", transport.Requests[1].URL.RawQuery)
	}
}
