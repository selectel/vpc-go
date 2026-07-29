package firewallpolicy

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

func TestFirewallPolicyRuleInsertAndRemoveUseAtomicPaths(t *testing.T) {
	client, transport := newClient(
		t,
		response(http.StatusOK, `{"firewall_policy":{"id":"fp-id","firewall_rules":["new","old"],"audited":false}}`),
		response(http.StatusOK, `{"firewall_policy":{"id":"fp-id","firewall_rules":["new"],"audited":false}}`),
	)
	policy, err := InsertRule(
		context.Background(),
		client,
		"fp/id",
		"new",
		BeforeRule("old"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(policy.FirewallRules, ",") != "new,old" {
		t.Fatalf("policy=%+v", policy)
	}
	policy, err = RemoveRule(context.Background(), client, "fp/id", "old")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(policy.FirewallRules, ",") != "new" {
		t.Fatalf("policy=%+v", policy)
	}
	for i, path := range []string{
		"/v2.0/fwaas/firewall_policies/fp%2Fid/insert_rule",
		"/v2.0/fwaas/firewall_policies/fp%2Fid/remove_rule",
	} {
		request := transport.requests[i]
		if request.Method != http.MethodPut || request.URL.EscapedPath() != path {
			t.Fatalf("request %d = %s %s", i, request.Method, request.URL.EscapedPath())
		}
		body, _ := io.ReadAll(request.Body)
		if strings.Contains(string(body), "audited") {
			t.Fatalf("body=%s", body)
		}
	}
	if len(transport.requests) != 2 {
		t.Fatalf("requests=%d", len(transport.requests))
	}
}

func TestFirewallPolicyRulePositionConstructorsAreExclusive(t *testing.T) {
	tests := []struct {
		position InsertPosition
		before   string
		after    string
	}{
		{AtBeginning(), "", ""},
		{BeforeRule("before"), "before", ""},
		{AfterRule("after"), "", "after"},
	}
	for _, test := range tests {
		request := insertRuleRequest{FirewallRuleID: "new"}
		test.position.apply(&request)
		if request.InsertBefore != test.before || request.InsertAfter != test.after {
			t.Fatalf("request=%+v", request)
		}
	}
}

func TestFirewallPolicyRuleErrorsRemainFailures(t *testing.T) {
	for _, action := range []string{"duplicate insert", "missing remove"} {
		t.Run(action, func(t *testing.T) {
			client, transport := newClient(t, response(
				http.StatusConflict,
				`{"NeutronError":{"type":"FirewallRuleConflict","message":"cannot change policy","status":"PENDING_UPDATE"}}`,
			))
			var err error
			if action == "duplicate insert" {
				_, err = InsertRule(context.Background(), client, "fp-id", "rule-id", AtBeginning())
			} else {
				_, err = RemoveRule(context.Background(), client, "fp-id", "rule-id")
			}
			if !vpc.IsErrorClass(err, vpc.ErrorClassConflict) ||
				!strings.Contains(err.Error(), "cannot change policy") {
				t.Fatalf("error=%v", err)
			}
			if len(transport.requests) != 1 {
				t.Fatalf("requests=%d", len(transport.requests))
			}
		})
	}
}
