package floatingip

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

func TestPortForwardingCRUDUsesNestedPathsAndSparseBodies(t *testing.T) {
	model := `{"port_forwarding":{"id":"pf-id","protocol":"tcp","internal_port_id":"port-id","internal_ip_address":"192.0.2.10","internal_port":8080,"external_port":80,"description":"web"}}`
	client, transport := newClient(
		t,
		response(http.StatusCreated, model),
		response(http.StatusOK, model),
		response(http.StatusOK, model),
		response(http.StatusNoContent, ""),
	)
	protocol, portID, address, description, projectID := "tcp", "port-id", "192.0.2.10", "web", "project-id"
	internalPort, externalPort, replacementPort := 8080, 80, 443

	created, err := CreatePortForwarding(context.Background(), client, "fip/id", PortForwardingCreateRequest{
		Protocol:          &protocol,
		InternalPortID:    &portID,
		InternalIPAddress: &address,
		InternalPort:      &internalPort,
		ExternalPort:      &externalPort,
		Description:       &description,
		ProjectID:         &projectID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != "pf-id" {
		t.Fatalf("created ID=%q", created.ID)
	}
	if _, err = GetPortForwarding(context.Background(), client, "fip/id", "pf/id"); err != nil {
		t.Fatal(err)
	}
	if _, err = UpdatePortForwarding(
		context.Background(),
		client,
		"fip/id",
		"pf/id",
		PortForwardingUpdateRequest{ExternalPort: &replacementPort},
	); err != nil {
		t.Fatal(err)
	}
	if err = DeletePortForwarding(context.Background(), client, "fip/id", "pf/id"); err != nil {
		t.Fatal(err)
	}

	wantMethods := []string{http.MethodPost, http.MethodGet, http.MethodPut, http.MethodDelete}
	wantPaths := []string{
		"/v2.0/floatingips/fip%2Fid/port_forwardings",
		"/v2.0/floatingips/fip%2Fid/port_forwardings/pf%2Fid",
		"/v2.0/floatingips/fip%2Fid/port_forwardings/pf%2Fid",
		"/v2.0/floatingips/fip%2Fid/port_forwardings/pf%2Fid",
	}
	for i, request := range transport.requests {
		if request.Method != wantMethods[i] || request.URL.EscapedPath() != wantPaths[i] {
			t.Fatalf("request %d = %s %s", i, request.Method, request.URL.EscapedPath())
		}
	}

	createBody, _ := io.ReadAll(transport.requests[0].Body)
	var createEnvelope map[string]map[string]any
	if err = json.Unmarshal(createBody, &createEnvelope); err != nil {
		t.Fatal(err)
	}
	if len(createEnvelope["port_forwarding"]) != 7 {
		t.Fatalf("create body=%s", createBody)
	}

	updateBody, _ := io.ReadAll(transport.requests[2].Body)
	if string(updateBody) != `{"port_forwarding":{"external_port":443}}` {
		t.Fatalf("update body=%s", updateBody)
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
	if len(transport.requests) != 2 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
	for _, request := range transport.requests {
		if request.URL.Path != "/v2.0/floatingips/fip-id/port_forwardings" {
			t.Fatalf("path=%s", request.URL.Path)
		}
	}
	if transport.requests[1].URL.Query().Get("marker") != "one" {
		t.Fatalf("second query=%s", transport.requests[1].URL.RawQuery)
	}
}

func TestPortForwardingErrorsPreserveClassAndMessageWithoutExtraRequests(t *testing.T) {
	testCases := []struct {
		name   string
		status int
		kind   string
		class  vpc.ErrorClass
		call   func(context.Context, *vpc.Client) error
	}{
		{
			name:   "deleted rule",
			status: http.StatusNotFound,
			kind:   "PortForwardingNotFound",
			class:  vpc.ErrorClassNotFound,
			call: func(ctx context.Context, client *vpc.Client) error {
				return DeletePortForwarding(ctx, client, "fip", "pf")
			},
		},
		{
			name:   "deleted floating IP",
			status: http.StatusNotFound,
			kind:   "FloatingIPNotFound",
			class:  vpc.ErrorClassNotFound,
			call: func(ctx context.Context, client *vpc.Client) error {
				_, err := GetPortForwarding(ctx, client, "fip", "pf")
				return err
			},
		},
		{
			name:   "occupied external port",
			status: http.StatusConflict,
			kind:   "PortForwardingConflict",
			class:  vpc.ErrorClassConflict,
			call: func(ctx context.Context, client *vpc.Client) error {
				_, err := CreatePortForwarding(ctx, client, "fip", PortForwardingCreateRequest{})
				return err
			},
		},
		{
			name:   "floating IP binding conflict",
			status: http.StatusConflict,
			kind:   "FloatingIPPortAlreadyAssociated",
			class:  vpc.ErrorClassConflict,
			call: func(ctx context.Context, client *vpc.Client) error {
				_, err := CreatePortForwarding(ctx, client, "fip", PortForwardingCreateRequest{})
				return err
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client, transport := newClient(t, response(
				testCase.status,
				`{"NeutronError":{"type":"`+testCase.kind+`","message":"diagnostic message"}}`,
			))
			err := testCase.call(context.Background(), client)
			if !vpc.IsErrorClass(err, testCase.class) || !strings.Contains(err.Error(), "diagnostic message") {
				t.Fatalf("error=%v", err)
			}
			if len(transport.requests) != 1 {
				t.Fatalf("requests=%d", len(transport.requests))
			}
		})
	}
}

func TestPortForwardingResponseHasNoProjectID(t *testing.T) {
	typ := reflect.TypeOf(PortForwarding{})
	if _, exists := typ.FieldByName("ProjectID"); exists {
		t.Fatal("PortForwarding must not expose ProjectID")
	}
}
