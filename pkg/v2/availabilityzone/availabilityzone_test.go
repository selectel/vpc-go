package availabilityzone

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

func newClient(t *testing.T, responses ...*http.Response) (*vpc.Client, *testutil.Transport) {
	return testutil.NewClient(t, responses...)
}

func response(status int, body string) *http.Response {
	return testutil.Response(status, body)
}

func TestAvailabilityZoneList(t *testing.T) {
	body := `{"availability_zones":[` +
		`{"state":"available","name":"cloudnet-1a","resource":"network"},` +
		`{"state":"available","name":"cloudnet-1a","resource":"router"}]}`
	client, transport := newClient(t, response(200, body))

	zones, err := List(context.Background(), client, vpc.ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(zones) != 2 {
		t.Fatalf("List() returned %d zones, want 2", len(zones))
	}
	if zones[1].Name != "cloudnet-1a" ||
		zones[1].Resource != ResourceRouter ||
		zones[1].State != StateAvailable {
		t.Fatalf("zones[1] = %+v", zones[1])
	}
	if path := transport.Requests[0].URL.Path; path != collectionPath {
		t.Fatalf("requested %s, want %s", path, collectionPath)
	}
}

func TestAvailabilityZoneListFilters(t *testing.T) {
	client, transport := newClient(t, response(200, `{"availability_zones":[]}`))

	_, err := List(context.Background(), client, vpc.ListOptions{
		SelectionOptions: vpc.SelectionOptions{
			Filters: map[string][]string{"resource": {ResourceRouter}},
		},
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if got := transport.Requests[0].URL.Query().Get("resource"); got != ResourceRouter {
		t.Fatalf("resource filter = %q, want %q", got, ResourceRouter)
	}
}
