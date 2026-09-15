package network

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const networkModel = `{"network":{"id":"id","name":"net","status":"BUILD","shared":true,` +
	`"router:external":true,"blocked":true,` +
	`"is_public":true,"is_dns_enabled":true,"dns_domain":"example.test.","revision_number":2}}`

func TestNetworkCreate(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusCreated, networkModel))
	name := "net"
	description := "description"
	adminStateUp := true
	hints := []string{"ru-1a"}
	dnsDomain := "example.test."

	created, err := Create(context.Background(), client, CreateRequest{
		Name:                  &name,
		Description:           &description,
		AdminStateUp:          &adminStateUp,
		AvailabilityZoneHints: &hints,
		DNSDomain:             &dnsDomain,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != "BUILD" || !created.Blocked || !created.RouterExternal ||
		created.DNSDomain != "example.test." {
		t.Fatalf("created network = %+v", created)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/networks")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"network":{`+
		`"name":"net","description":"description","admin_state_up":true,`+
		`"availability_zone_hints":["ru-1a"],"dns_domain":"example.test."}}`)
}

func TestNetworkGet(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, networkModel))
	got, err := Get(context.Background(), client, "id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "id" || got.Name != "net" {
		t.Fatalf("Get() = %+v", got)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/networks/id")
}

func TestNetworkUpdate(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, networkModel))
	name := "renamed"
	dnsDomain := "new.example.test."
	if _, err := Update(context.Background(), client, "id", UpdateRequest{
		Name: &name, DNSDomain: &dnsDomain,
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/networks/id")
	testutil.AssertJSONBody(t, transport.Requests[0],
		`{"network":{"name":"renamed","dns_domain":"new.example.test."}}`)
}

func TestNetworkDelete(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusNoContent, ""))
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodDelete, "/v2.0/networks/id")
}

func TestNetworkListWalksPagesOnConfiguredEndpoint(t *testing.T) {
	client, transport := testutil.NewClient(
		t,
		testutil.Response(http.StatusOK, `{"networks":[{"id":"one"}],"networks_links":[`+
			`{"rel":"next","href":"https://other.example.test/wrong?marker=one&limit=1"}]}`),
		testutil.Response(http.StatusOK, `{"networks":[{"id":"two"}],"networks_links":[]}`),
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
	for _, request := range transport.Requests {
		if request.URL.Host != "network.example.test" || request.URL.Path != collectionPath {
			t.Fatalf("request escaped endpoint: %s", request.URL)
		}
	}
	if transport.Requests[1].URL.Query().Get("marker") != "one" {
		t.Fatalf("second query = %v", transport.Requests[1].URL.Query())
	}
}

func TestNetworkListIncomplete(t *testing.T) {
	connectionErr := errors.New("connection lost")
	httpClient := &testutil.Transport{
		Responses: []*http.Response{
			testutil.Response(http.StatusOK, `{"networks":[{"id":"one"}],"networks_links":[`+
				`{"rel":"next","href":"?marker=one"}]}`),
			nil,
		},
		Errors: []error{nil, connectionErr},
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
	if !errors.Is(err, connectionErr) {
		t.Fatalf("List() error = %v, want connection error", err)
	}
}

func TestNetworkTagsReplace(t *testing.T) {
	client, transport := testutil.NewClient(t,
		testutil.Response(http.StatusOK, `{"tags":["new"]}`))
	replaced, err := TagOperations(client, "id").Replace(context.Background(), []string{"new"})
	if err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	if len(replaced) != 1 || replaced[0] != "new" {
		t.Fatalf("Replace() = %v, want [new]", replaced)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/networks/id/tags")
}
