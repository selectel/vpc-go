package port

import (
	"context"
	"io"
	"net/http"
	"reflect"
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

func TestPortCRUDCollectionsAndFields(t *testing.T) {
	model := `{"port":{"id":"id","network_id":"net","status":"DOWN","binding:vnic_type":"normal",` +
		`"blocked":true,"dhcp_blocked":true,"dns_name":"host","dns_domain":"example.test.",` +
		`"dns_assignment":[{"ip_address":"192.0.2.10","hostname":"host","fqdn":"host.example.test."}]}}`
	client, transport := newClient(t, response(201, model), response(200, model), response(200, model), response(204, ""))
	emptyStrings := []string{}
	emptyIPs := []FixedIP{}
	emptyPairs := []AllowedAddressPair{}
	dnsName := "host"
	dnsDomain := "example.test."
	created, err := Create(context.Background(), client, CreateRequest{
		NetworkID: "net", SecurityGroups: &emptyStrings,
		DNSName: &dnsName, DNSDomain: &dnsDomain,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.DNSDomain != dnsDomain || len(created.DNSAssignment) != 1 {
		t.Fatalf("created=%+v", created)
	}
	if _, err := Get(context.Background(), client, "id"); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(context.Background(), client, "id", UpdateRequest{
		FixedIPs: &emptyIPs, SecurityGroups: &emptyStrings,
		AllowedAddressPairs: &emptyPairs, DNSName: &dnsName, DNSDomain: &dnsDomain,
	}); err != nil {
		t.Fatal(err)
	}
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(transport.requests[2].Body)
	for _, field := range []string{
		`"fixed_ips":[]`, `"security_groups":[]`, `"allowed_address_pairs":[]`,
		`"dns_name":"host"`, `"dns_domain":"example.test."`,
	} {
		if !strings.Contains(string(body), field) {
			t.Fatalf("update body %s lacks %s", body, field)
		}
	}
	for _, forbidden := range []string{"mac_address", "blocked", "dhcp_blocked", "qos_policy_id", "port_security_enabled"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("update body contains %s", forbidden)
		}
	}
}

func TestPortTagsReplaceReturnsResult(t *testing.T) {
	client, transport := newClient(t, response(200, `{"tags":["one","two"]}`))
	tags, err := TagOperations(client, "id").Replace(
		context.Background(), []string{"one", "two"},
	)
	if err != nil || len(tags) != 2 || len(transport.requests) != 1 {
		t.Fatalf("Replace()=%v,%v requests=%d", tags, err, len(transport.requests))
	}
}

func TestPortListErrorsAndTags(t *testing.T) {
	client, _ := newClient(t,
		response(200, `{"ports":[{"id":"one"}],"ports_links":[]}`),
	)
	ports, err := List(context.Background(), client, vpc.ListOptions{SelectionOptions: vpc.SelectionOptions{Filters: map[string][]string{"network_id": {"net"}}}})
	if err != nil || len(ports) != 1 {
		t.Fatalf("List()=%+v,%v", ports, err)
	}

	for _, test := range []struct {
		status int
		kind   string
		class  vpc.ErrorClass
	}{
		{409, "IpAddressInUse", vpc.ErrorClassConflict},
		{409, "IpAddressGenerationFailure", vpc.ErrorClassAddressUnavailable},
		{400, "BadRequest", vpc.ErrorClassBadRequest},
	} {
		client, _ := newClient(t, response(test.status, `{"NeutronError":{"type":"`+test.kind+`","message":"failure"}}`))
		_, err := Create(context.Background(), client, CreateRequest{NetworkID: "net"})
		if !vpc.IsErrorClass(err, test.class) {
			t.Fatalf("%s error=%v want %s", test.kind, err, test.class)
		}
	}

	client, transport := newClient(t,
		response(403, `{"NeutronError":{"type":"PolicyNotAuthorized","message":"blocked"}}`),
		response(200, `{"port":{"id":"id","blocked":true}}`),
	)
	_, err = TagOperations(client, "id").Replace(context.Background(), []string{})
	if !vpc.IsErrorClass(err, vpc.ErrorClassForbidden) || len(transport.requests) != 1 {
		t.Fatalf("tag error=%v requests=%d", err, len(transport.requests))
	}
}

func TestPortPublicContractExcludesExtraDHCPOptions(t *testing.T) {
	for _, value := range []any{Port{}, CreateRequest{}, UpdateRequest{}} {
		if _, exists := reflect.TypeOf(value).FieldByName("ExtraDHCPOptions"); exists {
			t.Fatalf("%T exposes ExtraDHCPOptions", value)
		}
	}
	for _, value := range []any{CreateRequest{}, UpdateRequest{}} {
		if _, exists := reflect.TypeOf(value).FieldByName("DNSDomain"); !exists {
			t.Fatalf("%T does not expose DNSDomain", value)
		}
	}
}
