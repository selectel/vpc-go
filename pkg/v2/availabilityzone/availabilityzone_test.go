package availabilityzone

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

type scriptedClient struct {
	requests  []*http.Request
	responses []*http.Response
}

func (client *scriptedClient) Do(request *http.Request) (*http.Response, error) {
	client.requests = append(client.requests, request)
	return client.responses[len(client.requests)-1], nil
}

func newClient(t *testing.T, responses ...*http.Response) (*vpc.Client, *scriptedClient) {
	t.Helper()
	transport := &scriptedClient{responses: responses}
	client, err := vpc.NewClient(vpc.Config{
		Endpoint: "https://network.example.test", Token: "token", HTTPClient: transport,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client, transport
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

// The same zone name appears once per resource it serves, so a caller looking for
// router zones has to filter — both fields are therefore decoded.
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
	if path := transport.requests[0].URL.Path; path != collectionPath {
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

	if got := transport.requests[0].URL.Query().Get("resource"); got != ResourceRouter {
		t.Fatalf("resource filter = %q, want %q", got, ResourceRouter)
	}
}
