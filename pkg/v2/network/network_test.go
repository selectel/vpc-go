package network

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

type scriptedClient struct {
	requests  []*http.Request
	responses []*http.Response
	errors    []error
}

func (client *scriptedClient) Do(request *http.Request) (*http.Response, error) {
	client.requests = append(client.requests, request)
	index := len(client.requests) - 1
	if index < len(client.errors) && client.errors[index] != nil {
		return nil, client.errors[index]
	}
	response := client.responses[index]
	return response, nil
}

func newTestClient(t *testing.T, responses ...*http.Response) (*vpc.Client, *scriptedClient) {
	t.Helper()
	httpClient := &scriptedClient{responses: responses}
	client, err := vpc.NewClient(vpc.Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client, httpClient
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func TestNetworkCRUDAndFields(t *testing.T) {
	model := `{"network":{"id":"id","name":"net","status":"BUILD","shared":true,` +
		`"router:external":true,"provider:network_type":"vxlan","blocked":true,` +
		`"is_public":true,"is_dns_enabled":true,"revision_number":2}}`
	client, transport := newTestClient(
		t,
		response(http.StatusCreated, model),
		response(http.StatusOK, model),
		response(http.StatusOK, model),
		response(http.StatusNoContent, ""),
	)
	name := "net"
	description := "description"
	adminStateUp := true
	hints := []string{"ru-1a"}

	created, err := Create(context.Background(), client, CreateRequest{
		Name:                  &name,
		Description:           &description,
		AdminStateUp:          &adminStateUp,
		AvailabilityZoneHints: &hints,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != "BUILD" || !created.Blocked || !created.RouterExternal {
		t.Fatalf("created network = %+v", created)
	}
	if _, err := Get(context.Background(), client, "id"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := Update(
		context.Background(),
		client,
		"id",
		UpdateRequest{Name: &name},
	); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	wantMethods := []string{"POST", "GET", "PUT", "DELETE"}
	wantPaths := []string{"/v2.0/networks", "/v2.0/networks/id", "/v2.0/networks/id", "/v2.0/networks/id"}
	for index, request := range transport.requests {
		if request.Method != wantMethods[index] || request.URL.Path != wantPaths[index] {
			t.Fatalf("request %d = %s %s", index, request.Method, request.URL.Path)
		}
	}

	body, err := io.ReadAll(transport.requests[0].Body)
	if err != nil {
		t.Fatalf("read create body: %v", err)
	}
	createJSON := string(body)
	for _, forbidden := range []string{
		`"shared"`, `"router:external"`, `"provider:"`, `"segments"`,
		`"qos_policy_id"`, `"mtu"`, `"port_security_enabled"`, `"blocked"`,
		`"is_public"`, `"is_dns_enabled"`,
	} {
		if strings.Contains(createJSON, forbidden) {
			t.Fatalf("create body %s contains %s", createJSON, forbidden)
		}
	}
}

func TestNetworkListWalksPagesOnConfiguredEndpoint(t *testing.T) {
	client, transport := newTestClient(
		t,
		response(http.StatusOK, `{"networks":[{"id":"one"}],"networks_links":[`+
			`{"rel":"next","href":"https://other.example.test/wrong?marker=one&limit=1"}]}`),
		response(http.StatusOK, `{"networks":[{"id":"two"}],"networks_links":[]}`),
	)

	networks, err := List(context.Background(), client, vpc.ListOptions{
		SelectionOptions: vpc.SelectionOptions{
			Filters: map[string][]string{"status": {"ACTIVE"}},
			Fields:  []string{"id"},
			SortKey: "name",
			SortDir: "asc",
		},
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(networks) != 2 || networks[0].ID != "one" || networks[1].ID != "two" {
		t.Fatalf("networks = %+v", networks)
	}
	for _, request := range transport.requests {
		if request.URL.Host != "network.example.test" || request.URL.Path != collectionPath {
			t.Fatalf("request escaped endpoint: %s", request.URL)
		}
	}
	if transport.requests[1].URL.Query().Get("marker") != "one" {
		t.Fatalf("second query = %v", transport.requests[1].URL.Query())
	}
}

func TestNetworkListIncomplete(t *testing.T) {
	connectionErr := errors.New("connection lost")
	httpClient := &scriptedClient{
		responses: []*http.Response{
			response(http.StatusOK, `{"networks":[{"id":"one"}],"networks_links":[`+
				`{"rel":"next","href":"?marker=one"}]}`),
			nil,
		},
		errors: []error{nil, connectionErr},
	}
	client, err := vpc.NewClient(vpc.Config{
		Endpoint:   "https://network.example.test",
		Token:      "token",
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	networks, err := List(context.Background(), client, vpc.ListOptions{})
	if networks != nil {
		t.Fatalf("networks = %+v, want nil", networks)
	}
	if !vpc.IsIncompleteList(err) || !errors.Is(err, connectionErr) {
		t.Fatalf("List() error = %v, want incomplete list", err)
	}
}

func TestNetworkDeleteErrorClasses(t *testing.T) {
	tests := []struct {
		status int
		body   string
		class  vpc.ErrorClass
	}{
		{404, `{"NeutronError":{"type":"NetworkNotFound","message":"missing"}}`, vpc.ErrorClassNotFound},
		{409, `{"NeutronError":{"type":"NetworkInUse","message":"ports remain"}}`, vpc.ErrorClassConflict},
	}
	for _, test := range tests {
		client, transport := newTestClient(t, response(test.status, test.body))
		err := Delete(context.Background(), client, "id")
		if !vpc.IsErrorClass(err, test.class) {
			t.Fatalf("Delete() error = %v, want %s", err, test.class)
		}
		if len(transport.requests) != 1 {
			t.Fatalf("request count = %d, want 1", len(transport.requests))
		}
	}
}

func TestNetworkTagsReplaceAndBlocked(t *testing.T) {
	client, transport := newTestClient(
		t,
		response(http.StatusOK, `{"tags":["new"]}`),
		response(http.StatusForbidden, `{"NeutronError":{"type":"PolicyNotAuthorized","message":"blocked"}}`),
		response(http.StatusOK, `{"network":{"id":"id","blocked":true}}`),
	)
	tags := TagOperations(client, "id")

	if err := tags.Replace(context.Background(), []string{"new"}); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	err := tags.Add(context.Background(), "tag")
	if !vpc.IsErrorClass(err, vpc.ErrorClassResourceBlocked) {
		t.Fatalf("Add() error = %v, want blocked", err)
	}
	if len(transport.requests) != 3 {
		t.Fatalf("request count = %d, want 3", len(transport.requests))
	}
}
