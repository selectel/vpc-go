package floatingip

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
func response(s int, b string) *http.Response {
	return &http.Response{StatusCode: s, Body: io.NopCloser(strings.NewReader(b))}
}
func newClient(t *testing.T, rs ...*http.Response) (*vpc.Client, *scriptedClient) {
	t.Helper()
	tr := &scriptedClient{responses: rs}
	c, e := vpc.NewClient(vpc.Config{Endpoint: "https://network.example.test", Token: "token", HTTPClient: tr})
	if e != nil {
		t.Fatal(e)
	}
	return c, tr
}

func TestFloatingIPCRUDDetachListAndTags(t *testing.T) {
	model := `{"floatingip":{"id":"id","floating_network_id":"ext","floating_ip_address":"203.0.113.1","status":"DOWN","blocked":true}}`
	c, tr := newClient(t, response(201, model), response(200, model), response(200, `{"floatingip":{"id":"id","port_id":null,"router_id":null}}`), response(204, ""), response(200, `{"floatingips":[{"id":"id"}],"floatingips_links":[]}`), response(200, `{"tags":[]}`))
	if _, e := Create(context.Background(), c, CreateRequest{FloatingNetworkID: "ext"}); e != nil {
		t.Fatal(e)
	}
	if _, e := Get(context.Background(), c, "id"); e != nil {
		t.Fatal(e)
	}
	if _, e := Update(context.Background(), c, "id", UpdateRequest{PortID: vpc.Null[string]()}); e != nil {
		t.Fatal(e)
	}
	if e := Delete(context.Background(), c, "id"); e != nil {
		t.Fatal(e)
	}
	if fs, e := List(context.Background(), c, vpc.ListOptions{}); e != nil || len(fs) != 1 {
		t.Fatalf("List=%+v,%v", fs, e)
	}
	if _, e := TagOperations(c, "id").Get(context.Background()); e != nil {
		t.Fatal(e)
	}
	createBody, _ := io.ReadAll(tr.requests[0].Body)
	updateBody, _ := io.ReadAll(tr.requests[2].Body)
	for _, f := range []string{"floating_ip_address", "qos_policy_id", "blocked"} {
		if strings.Contains(string(createBody), f) {
			t.Fatalf("create contains %s", f)
		}
	}
	if !strings.Contains(string(updateBody), `"port_id":null`) {
		t.Fatalf("update=%s", updateBody)
	}
}

func TestFloatingIPErrorClasses(t *testing.T) {
	for _, x := range []struct {
		s int
		k string
		c vpc.ErrorClass
	}{{409, "OverQuota", vpc.ErrorClassQuotaExceeded}, {409, "IpAddressGenerationFailure", vpc.ErrorClassAddressUnavailable}, {409, "PortInUse", vpc.ErrorClassConflict}, {404, "FloatingIPNotFound", vpc.ErrorClassNotFound}} {
		c, _ := newClient(t, response(x.s, `{"NeutronError":{"type":"`+x.k+`","message":"failure"}}`))
		_, e := Create(context.Background(), c, CreateRequest{FloatingNetworkID: "ext"})
		if !vpc.IsErrorClass(e, x.c) {
			t.Fatalf("%s error=%v want %s", x.k, e, x.c)
		}
	}
}
