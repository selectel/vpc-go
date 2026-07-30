package subnet

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

func newTestClient(t *testing.T, responses ...*http.Response) (*vpc.Client, *scriptedClient) {
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

func TestSubnetCRUDAndGatewayNull(t *testing.T) {
	model := `{"subnet":{"id":"id","network_id":"network","ip_version":4,` +
		`"cidr":"192.0.2.0/24","service_types":["network:floatingip_agent_gateway"],` +
		`"blocked":true}}`
	client, transport := newTestClient(
		t,
		response(201, model),
		response(200, model),
		response(200, model),
		response(204, ""),
	)
	cidr := "192.0.2.0/24"
	if _, err := Create(context.Background(), client, CreateRequest{
		NetworkID: "network", CIDR: &cidr,
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := Get(context.Background(), client, "id"); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := Update(context.Background(), client, "id", UpdateRequest{
		GatewayIP: vpc.Null[string](),
	}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	updateBody, err := io.ReadAll(transport.requests[2].Body)
	if err != nil {
		t.Fatalf("read update body: %v", err)
	}
	if !strings.Contains(string(updateBody), `"gateway_ip":null`) {
		t.Fatalf("update body = %s", updateBody)
	}
	if strings.Contains(string(updateBody), "service_types") ||
		strings.Contains(string(updateBody), "segment_id") {
		t.Fatalf("update body contains administrative fields: %s", updateBody)
	}
}

func TestSubnetListAndErrors(t *testing.T) {
	client, transport := newTestClient(
		t,
		response(200, `{"subnets":[{"id":"one"}],"subnets_links":[{"rel":"next","href":"?marker=one"}]}`),
		response(200, `{"subnets":[{"id":"two"}],"subnets_links":[]}`),
	)
	subnets, err := List(context.Background(), client, vpc.ListOptions{Limit: 1})
	if err != nil || len(subnets) != 2 {
		t.Fatalf("List() = %+v, %v", subnets, err)
	}
	if len(transport.requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(transport.requests))
	}

	errorCases := []struct {
		status int
		kind   string
		class  vpc.ErrorClass
	}{
		{404, "SubnetNotFound", vpc.ErrorClassNotFound},
		{409, "SubnetInUse", vpc.ErrorClassConflict},
		{409, "IpAddressGenerationFailure", vpc.ErrorClassAddressUnavailable},
	}
	for _, test := range errorCases {
		client, _ := newTestClient(t, response(
			test.status,
			`{"NeutronError":{"type":"`+test.kind+`","message":"failure"}}`,
		))
		var err error
		if test.kind == "IpAddressGenerationFailure" {
			err = func() error {
				_, createErr := Create(context.Background(), client, CreateRequest{NetworkID: "id"})
				return createErr
			}()
		} else {
			err = Delete(context.Background(), client, "id")
		}
		if !vpc.IsErrorClass(err, test.class) {
			t.Fatalf("error = %v, want %s", err, test.class)
		}
	}
}

func TestSubnetTagsBlocked(t *testing.T) {
	client, transport := newTestClient(
		t,
		response(403, `{"NeutronError":{"type":"PolicyNotAuthorized","message":"blocked"}}`),
		response(200, `{"subnet":{"id":"id","blocked":true}}`),
	)
	_, err := TagOperations(client, "id").Replace(context.Background(), []string{"tag"})
	if !vpc.IsErrorClass(err, vpc.ErrorClassForbidden) {
		t.Fatalf("Replace() error = %v, want forbidden", err)
	}
	if len(transport.requests) != 1 {
		t.Fatalf("request count = %d, want 1", len(transport.requests))
	}
}
