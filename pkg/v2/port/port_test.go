package port

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const portModel = `{"port":{"id":"id","network_id":"net","status":"DOWN",` +
	`"blocked":true,"dhcp_blocked":true,"dns_name":"host",` +
	`"extra_dhcp_opts":[{"opt_name":"domain-name","opt_value":"example.test","ip_version":4}]}}`

func TestPortCreate(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusCreated, portModel))
	emptyStrings := []string{}
	dhcpOptions := []ExtraDHCPOption{{Name: "domain-name", Value: "example.test"}}
	dnsName := "host"
	created, err := Create(context.Background(), client, CreateRequest{
		NetworkID: "net", SecurityGroups: &emptyStrings,
		ExtraDHCPOptions: &dhcpOptions,
		DNSName:          &dnsName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.DNSName != dnsName || len(created.ExtraDHCPOptions) != 1 {
		t.Fatalf("created=%+v", created)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPost, "/v2.0/ports")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"port":{"network_id":"net",`+
		`"security_groups":[],"extra_dhcp_opts":[{"opt_name":"domain-name",`+
		`"opt_value":"example.test"}],"dns_name":"host"}}`)
}

func TestPortGet(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, portModel))
	got, err := Get(context.Background(), client, "id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "id" || got.NetworkID != "net" {
		t.Fatalf("Get() = %+v", got)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodGet, "/v2.0/ports/id")
}

func TestPortUpdateClearsCollectionsAndRemovesDHCPOption(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, portModel))
	emptyStrings := []string{}
	emptyIPs := []FixedIP{}
	emptyPairs := []AllowedAddressPair{}
	dnsName := "host"
	keptValue := "example.test"
	updateOptions := []UpdateExtraDHCPOption{
		{Name: "domain-name", Value: &keptValue},
		{Name: "bootfile-name", Value: nil},
	}
	if _, err := Update(context.Background(), client, "id", UpdateRequest{
		FixedIPs: &emptyIPs, SecurityGroups: &emptyStrings,
		AllowedAddressPairs: &emptyPairs, DNSName: &dnsName,
		ExtraDHCPOptions: &updateOptions,
	}); err != nil {
		t.Fatal(err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/ports/id")
	testutil.AssertJSONBody(t, transport.Requests[0], `{"port":{"fixed_ips":[],`+
		`"security_groups":[],"allowed_address_pairs":[],"dns_name":"host",`+
		`"extra_dhcp_opts":[{"opt_name":"domain-name","opt_value":"example.test"},`+
		`{"opt_name":"bootfile-name","opt_value":null}]}}`)
}

func TestPortDelete(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusNoContent, ""))
	if err := Delete(context.Background(), client, "id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodDelete, "/v2.0/ports/id")
}

func TestPortTagsReplaceReturnsResult(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(200, `{"tags":["one","two"]}`))
	tags, err := TagOperations(client, "id").Replace(
		context.Background(), []string{"one", "two"},
	)
	if err != nil || len(tags) != 2 || len(transport.Requests) != 1 {
		t.Fatalf("Replace()=%v,%v requests=%d", tags, err, len(transport.Requests))
	}
	testutil.AssertRequest(t, transport.Requests[0], http.MethodPut, "/v2.0/ports/id/tags")
}

func TestPortList(t *testing.T) {
	client, _ := testutil.NewClient(t,
		testutil.Response(200, `{"ports":[{"id":"one"}],"ports_links":[]}`),
	)
	ports, err := List(context.Background(), client, vpc.ListOptions{SelectionOptions: vpc.SelectionOptions{Filters: map[string][]string{"network_id": {"net"}}}})
	if err != nil || len(ports) != 1 {
		t.Fatalf("List()=%+v,%v", ports, err)
	}
}
