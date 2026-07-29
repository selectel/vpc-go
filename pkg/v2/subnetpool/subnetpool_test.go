package subnetpool

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

func TestSubnetPoolCRUDAndPrefixReplacement(t *testing.T) {
	model := `{"subnetpool":{"id":"id","prefixes":["192.0.2.0/24"],` +
		`"ip_version":4,"is_default":false,"shared":true}}`
	client, transport := newClient(
		t, response(201, model), response(200, model), response(200, model), response(204, ""),
	)
	prefixes := []string{"192.0.2.0/24"}
	if _, err := Create(context.Background(), client, CreateRequest{Prefixes: &prefixes}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Get(context.Background(), client, "id"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := Update(
		context.Background(), client, "id", UpdateRequest{Prefixes: &prefixes},
	); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	body, err := io.ReadAll(transport.requests[2].Body)
	if err != nil {
		t.Fatalf("read update body: %v", err)
	}
	if !strings.Contains(string(body), `"prefixes":["192.0.2.0/24"]`) {
		t.Fatalf("update body = %s", body)
	}
	for _, forbidden := range []string{`"is_default"`, `"shared"`, `"ip_version"`} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("update body contains %s: %s", forbidden, body)
		}
	}
}

func TestSubnetPoolListIncludesServiceResponse(t *testing.T) {
	client, _ := newClient(t, response(200,
		`{"subnetpools":[{"id":"project","shared":false},{"id":"public","shared":true}],`+
			`"subnetpools_links":[]}`,
	))
	pools, err := List(context.Background(), client, vpc.ListOptions{})
	if err != nil || len(pools) != 2 || !pools[1].Shared {
		t.Fatalf("List() = %+v, %v", pools, err)
	}
}

func TestSubnetPoolConflictAndReadOnlyTags(t *testing.T) {
	client, _ := newClient(
		t,
		response(409, `{"NeutronError":{"type":"SubnetPoolInUse","message":"in use"}}`),
		response(200, `{"tags":["one"]}`),
		response(204, ""),
	)
	err := Delete(context.Background(), client, "id")
	if !vpc.IsErrorClass(err, vpc.ErrorClassConflict) {
		t.Fatalf("Delete() error = %v, want conflict", err)
	}
	tags := TagOperations(client, "id")
	values, err := tags.Get(context.Background())
	if err != nil || len(values) != 1 {
		t.Fatalf("tags.Get() = %+v, %v", values, err)
	}
	present, err := tags.Has(context.Background(), "one")
	if err != nil || !present {
		t.Fatalf("tags.Has() = %v, %v", present, err)
	}
}
