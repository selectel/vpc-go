package securitygrouprule

import (
	"context"
	"net/http"
	"testing"

	"github.com/selectel/vpc-go/internal/testutil"
	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const ruleModel = `{"security_group_rule":{"id":"rule-id",` +
	`"security_group_id":"sg-id","protocol":"6"}}`

func TestCreateUsesTopLevelCollection(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusCreated, ruleModel))
	protocol, remoteIPPrefix := "6", "192.0.2.0/24"
	rule, err := Create(context.Background(), client, CreateRequest{
		SecurityGroupID: "sg-id",
		Direction:       "ingress",
		Protocol:        &protocol,
		RemoteIPPrefix:  &remoteIPPrefix,
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

func TestGetUsesTopLevelCollection(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusOK, ruleModel))
	got, err := Get(context.Background(), client, "rule/id")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != "rule-id" {
		t.Fatalf("Get() = %+v", got)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodGet ||
		request.URL.EscapedPath() != "/v2.0/security-group-rules/rule%2Fid" {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
}

func TestDeleteUsesTopLevelCollection(t *testing.T) {
	client, transport := testutil.NewClient(t, testutil.Response(http.StatusNoContent, ""))
	if err := Delete(context.Background(), client, "rule/id"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	request := transport.Requests[0]
	if request.Method != http.MethodDelete ||
		request.URL.EscapedPath() != "/v2.0/security-group-rules/rule%2Fid" {
		t.Fatalf("request = %s %s", request.Method, request.URL.EscapedPath())
	}
}

func TestListPassesFilterAndWalksPages(t *testing.T) {
	client, transport := testutil.NewClient(
		t,
		testutil.Response(http.StatusOK, `{"security_group_rules":[{"id":"one"}],"security_group_rules_links":[{"rel":"next","href":"https://ignored.invalid/v2.0/security-group-rules?security_group_id=sg-id&marker=one"}]}`),
		testutil.Response(http.StatusOK, `{"security_group_rules":[{"id":"two"}],"security_group_rules_links":[]}`),
	)
	rules, err := List(context.Background(), client, vpc.ListOptions{
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
