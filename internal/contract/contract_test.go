package contract

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
	"github.com/selectel/vpc-go/pkg/v2/addressscope"
	"github.com/selectel/vpc-go/pkg/v2/firewallrule"
	"github.com/selectel/vpc-go/pkg/v2/floatingip"
	"github.com/selectel/vpc-go/pkg/v2/network"
	"github.com/selectel/vpc-go/pkg/v2/port"
	"github.com/selectel/vpc-go/pkg/v2/router"
	"github.com/selectel/vpc-go/pkg/v2/securitygroup"
	"github.com/selectel/vpc-go/pkg/v2/subnet"
	"github.com/selectel/vpc-go/pkg/v2/subnetpool"
)

// Sources:
// neutron/extensions/dns.py, neutron/extensions/dns_domain_ports.py,
// neutron/extensions/extra_dhcp_opt.py and neutron/objects/ports.py.
func TestContractNetworkAndPortAttributeMatrix(t *testing.T) {
	assertFields(t, network.Network{}, []string{"DNSDomain"}, nil)
	assertFields(t, network.CreateRequest{}, []string{"DNSDomain"},
		[]string{"Shared", "RouterExternal", "Segments"})
	assertFields(t, network.UpdateRequest{}, []string{"DNSDomain"},
		[]string{"Shared", "RouterExternal", "Segments"})

	assertFields(t, port.Port{},
		[]string{"DNSName", "DNSDomain", "DNSAssignment", "ExtraDHCPOptions", "BindingVNICType"}, nil)
	assertFields(t, port.CreateRequest{},
		[]string{"DNSName", "DNSDomain", "ExtraDHCPOptions", "BindingVNICType"}, nil)
	assertFields(t, port.UpdateRequest{},
		[]string{"DNSName", "DNSDomain", "ExtraDHCPOptions", "BindingVNICType"}, nil)

	assertFields(t, subnet.Subnet{}, []string{"DNSPublishFixedIP"}, []string{"SegmentID"})
	assertFields(t, subnet.CreateRequest{}, []string{"DNSPublishFixedIP"}, []string{"SegmentID"})
	assertFields(t, subnet.UpdateRequest{}, []string{"DNSPublishFixedIP"}, []string{"SegmentID"})

	assertFields(t, router.Router{}, []string{"FlavorID"}, nil)
	assertFields(t, router.CreateRequest{}, []string{"FlavorID"}, nil)
	assertFields(t, router.UpdateRequest{}, nil, []string{"FlavorID"})
}

// Sources:
// neutron_lib/api/definitions/stateful_security_group.py, which sets allow_post
// and allow_put for stateful, and
// neutron_lib/api/definitions/security_groups_shared_filtering.py, which sets
// neither for shared - a group is shared through an RBAC policy instead.
func TestContractSecurityGroupAttributeMatrix(t *testing.T) {
	assertFields(t, securitygroup.SecurityGroup{}, []string{"Stateful", "Shared"}, nil)
	assertFields(t, securitygroup.CreateRequest{}, []string{"Stateful"}, []string{"Shared"})
	assertFields(t, securitygroup.UpdateRequest{}, []string{"Stateful"}, []string{"Shared"})

	// A pointer keeps an omitted stateful out of the request, so the API default
	// of true survives instead of being overwritten with false.
	pointerBool := reflect.TypeOf((*bool)(nil))
	assertFieldType(t, securitygroup.CreateRequest{}, "Stateful", pointerBool)
	assertFieldType(t, securitygroup.UpdateRequest{}, "Stateful", pointerBool)
}

// Sources:
// neutron/extensions/l3.py, neutron/extensions/floatingip_pools.py,
// neutron/extensions/subnetpool_prefix_ops.py and neutron/db/models_v2.py.
func TestContractFloatingIPAndSubnetPoolAttributeMatrix(t *testing.T) {
	assertFields(t, floatingip.FloatingIP{}, []string{"DNSName", "DNSDomain"},
		[]string{"SubnetID"})
	assertFields(t, floatingip.CreateRequest{},
		[]string{"SubnetID", "DNSName", "DNSDomain"}, nil)
	assertFields(t, floatingip.UpdateRequest{},
		[]string{"PortID", "FixedIPAddress", "DNSName", "DNSDomain"}, nil)

	optionalType := reflect.TypeOf((*vpc.Optional[string])(nil))
	assertFieldType(t, floatingip.UpdateRequest{}, "PortID", optionalType)
	assertFieldType(t, floatingip.UpdateRequest{}, "FixedIPAddress", optionalType)
	assertFieldType(t, subnetpool.UpdateRequest{}, "AddressScopeID", optionalType)
}

func TestContractRequiredCreateFields(t *testing.T) {
	assertRequiredJSONField(t, addressscope.CreateRequest{}, "IPVersion", "ip_version")
	assertRequiredJSONField(t, subnet.CreateRequest{}, "NetworkID", "network_id")
	assertRequiredJSONField(t, subnet.CreateRequest{}, "IPVersion", "ip_version")
	assertRequiredJSONField(t, subnetpool.CreateRequest{}, "Prefixes", "prefixes")
}

// Source: neutron_fwaas/services/firewall/service_drivers/driver_api.py and
// neutron_fwaas/db/firewall/v2/firewall_db_v2.py.
func TestContractFirewallRuleNullableResponseFields(t *testing.T) {
	pointerString := reflect.TypeOf((*string)(nil))
	for _, name := range []string{
		"Protocol", "SourceIPAddress", "DestinationIPAddress", "SourcePort", "DestinationPort",
	} {
		assertFieldType(t, firewallrule.FirewallRule{}, name, pointerString)
	}
}

// Sources: neutron_lib/exceptions/__init__.py and the Neutron v2 API
// serializers. The low-level executor and tag helper are module-internal.
func TestContractErrorAndPublicAPIBoundary(t *testing.T) {
	assertFields(t, vpc.APIError{},
		[]string{"StatusCode", "Type", "Message", "Detail", "RawBody"},
		[]string{"ResourceStatus"})

	clientType := reflect.TypeOf((*vpc.Client)(nil))
	for _, name := range []string{"Request", "Do"} {
		if _, exists := clientType.MethodByName(name); exists {
			t.Fatalf("Client exposes %s", name)
		}
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate contract test")
	}
	rootPackage := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "pkg", "v2"))
	packages, err := parser.ParseDir(token.NewFileSet(), rootPackage, func(info os.FileInfo) bool {
		return filepath.Ext(info.Name()) == ".go" && !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range packages["v2"].Files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Recv == nil && function.Name.Name == "NewTagOperations" {
				t.Fatal("pkg/v2 exports NewTagOperations")
			}
		}
	}
}

func assertFields(t *testing.T, value any, required, absent []string) {
	t.Helper()
	typ := reflect.TypeOf(value)
	for _, name := range required {
		if _, exists := typ.FieldByName(name); !exists {
			t.Errorf("%s lacks %s", typ, name)
		}
	}
	for _, name := range absent {
		if _, exists := typ.FieldByName(name); exists {
			t.Errorf("%s exposes %s", typ, name)
		}
	}
}

func assertFieldType(t *testing.T, value any, name string, expected reflect.Type) {
	t.Helper()
	field, exists := reflect.TypeOf(value).FieldByName(name)
	if !exists {
		t.Fatalf("%T lacks %s", value, name)
	}
	if field.Type != expected {
		t.Errorf("%T.%s type = %s, want %s", value, name, field.Type, expected)
	}
}

func assertRequiredJSONField(t *testing.T, value any, name, jsonName string) {
	t.Helper()
	field, exists := reflect.TypeOf(value).FieldByName(name)
	if !exists {
		t.Fatalf("%T lacks %s", value, name)
	}
	if field.Type.Kind() == reflect.Pointer {
		t.Errorf("%T.%s is a pointer", value, name)
	}
	tag := field.Tag.Get("json")
	parts := strings.Split(tag, ",")
	if parts[0] != jsonName {
		t.Errorf("%T.%s JSON name = %q, want %q", value, name, parts[0], jsonName)
	}
	for _, option := range parts[1:] {
		if option == "omitempty" {
			t.Errorf("%T.%s uses omitempty", value, name)
		}
	}
}
