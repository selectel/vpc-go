package router

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

func (c *scriptedClient) Do(r *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, r)
	return c.responses[len(c.requests)-1], nil
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func newClient(t *testing.T, responses ...*http.Response) (*vpc.Client, *scriptedClient) {
	t.Helper()
	transport := &scriptedClient{responses: responses}
	client, err := vpc.NewClient(vpc.Config{Endpoint: "https://network.example.test", Token: "token", HTTPClient: transport})
	if err != nil {
		t.Fatal(err)
	}
	return client, transport
}

func TestRouterCRUDFieldsListTags(t *testing.T) {
	model := `{"router":{"id":"id","status":"BUILD","blocked":true,"external_gateway_info":{"network_id":"ext","enable_snat":true,"external_fixed_ips":[{"subnet_id":"sub","ip_address":"192.0.2.1"}]}}}`
	client, transport := newClient(t, response(201, model), response(200, model), response(200, model), response(204, ""), response(200, `{"routers":[{"id":"id"}],"routers_links":[]}`), response(200, `{"tags":[]}`))
	enable := true
	if _, err := Create(context.Background(), client, CreateRequest{ExternalGateway: vpc.Value(ExternalGatewayRequest{NetworkID: "ext", EnableSNAT: &enable})}); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(context.Background(), client, "id"); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(context.Background(), client, "id", UpdateRequest{ExternalGateway: vpc.Null[ExternalGatewayRequest]()}); err != nil {
		t.Fatal(err)
	}
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatal(err)
	}
	if routers, err := List(context.Background(), client, vpc.ListOptions{}); err != nil || len(routers) != 1 {
		t.Fatalf("List=%+v,%v", routers, err)
	}
	if _, err := TagOperations(client, "id").Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	createBody, _ := io.ReadAll(transport.requests[0].Body)
	updateBody, _ := io.ReadAll(transport.requests[2].Body)
	for _, forbidden := range []string{"external_fixed_ips", "qos_policy_id", `"ha"`, `"distributed"`} {
		if strings.Contains(string(createBody), forbidden) {
			t.Fatalf("create body contains %s: %s", forbidden, createBody)
		}
	}
	if !strings.Contains(string(updateBody), `"external_gateway_info":null`) {
		t.Fatalf("update body=%s", updateBody)
	}
}

func TestRouterDeleteConflictAndBlockedTags(t *testing.T) {
	client, _ := newClient(t, response(409, `{"NeutronError":{"type":"RouterInUse","message":"interfaces"}}`))
	if err := Delete(context.Background(), client, "id"); !vpc.IsErrorClass(err, vpc.ErrorClassConflict) {
		t.Fatalf("Delete error=%v", err)
	}
	client, transport := newClient(t, response(403, `{"NeutronError":{"type":"PolicyNotAuthorized","message":"blocked"}}`), response(200, `{"router":{"id":"id","blocked":true}}`))
	_, err := TagOperations(client, "id").Replace(context.Background(), []string{})
	if !vpc.IsErrorClass(err, vpc.ErrorClassForbidden) || len(transport.requests) != 1 {
		t.Fatalf("tag error=%v requests=%d", err, len(transport.requests))
	}
}
