package securitygroup

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const securityGroupRuleModel = `{"security_group_rule":{"id":"rule-id",` +
	`"security_group_id":"sg-id","protocol":"6"}}`

func TestSecurityGroupRuleCreateUsesTopLevelCollection(t *testing.T) {
	client, transport := newClient(t, response(http.StatusCreated, securityGroupRuleModel))
	protocol := "6"
	rule, err := CreateRule(context.Background(), client, RuleCreateRequest{
		SecurityGroupID: "sg-id",
		Direction:       "ingress",
		Protocol:        &protocol,
		RemoteSource:    ByRemoteIPPrefix("192.0.2.0/24"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if rule.ID != "rule-id" {
		t.Fatalf("rule=%+v", rule)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodPost || request.URL.Path != "/v2.0/security-group-rules" {
		t.Fatalf("request = %s %s", request.Method, request.URL.Path)
	}
	testutil.AssertJSONBody(t, request, `{"security_group_rule":{`+
		`"security_group_id":"sg-id","direction":"ingress","protocol":"6",`+
		`"remote_ip_prefix":"192.0.2.0/24"}}`)
}

func TestSecurityGroupRuleGetUsesTopLevelCollection(t *testing.T) {
	client, transport := newClient(t, response(http.StatusOK, securityGroupRuleModel))
	got, err := GetRule(context.Background(), client, "rule/id")
	if err != nil {
		t.Fatalf("GetRule() error = %v", err)
	}
	if got.ID != "rule-id" {
		t.Fatalf("GetRule() = %+v", got)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodGet ||
		request.URL.EscapedPath() != "/v2.0/security-group-rules/rule%2Fid" {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
}

func TestSecurityGroupRuleDeleteUsesTopLevelCollection(t *testing.T) {
	client, transport := newClient(t, response(http.StatusNoContent, ""))
	if err := DeleteRule(context.Background(), client, "rule/id"); err != nil {
		t.Fatalf("DeleteRule() error = %v", err)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodDelete ||
		request.URL.EscapedPath() != "/v2.0/security-group-rules/rule%2Fid" {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
}

func TestSecurityGroupRuleRemoteSourceConstructorsAreExclusive(t *testing.T) {
	requests := []RuleCreateRequest{
		{RemoteSource: ByRemoteIPPrefix("192.0.2.0/24")},
		{RemoteSource: ByRemoteGroupID("remote-sg")},
	}
	for index, request := range requests {
		payload := request.payload()
		if (payload.RemoteIPPrefix == nil) == (payload.RemoteGroupID == nil) {
			t.Fatalf("request %d has invalid sources: %+v", index, payload)
		}
	}
}

func TestSecurityGroupRuleListPassesFilterAndWalksPages(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"security_group_rules":[{"id":"one"}],"security_group_rules_links":[{"rel":"next","href":"https://ignored.invalid/v2.0/security-group-rules?security_group_id=sg-id&marker=one"}]}`),
		response(http.StatusOK, `{"security_group_rules":[{"id":"two"}],"security_group_rules_links":[]}`),
	)
	rules, err := ListRules(context.Background(), client, vpc.ListOptions{
		SelectionOptions: vpc.SelectionOptions{
			Filters: map[string][]string{"security_group_id": {"sg-id"}},
		},
		Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || len(transport.Requests) != 2 {
		t.Fatalf("rules=%+v requests=%d", rules, len(transport.Requests))
	}
	for _, request := range transport.Requests {
		if request.URL.Query().Get("security_group_id") != "sg-id" {
			t.Fatalf("query=%s", request.URL.RawQuery)
		}
	}
}
