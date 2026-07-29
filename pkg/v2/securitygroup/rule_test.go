package securitygroup

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

func TestSecurityGroupRuleCRUDUsesTopLevelCollection(t *testing.T) {
	model := `{"security_group_rule":{"id":"rule-id","security_group_id":"sg-id","protocol":"6"}}`
	client, transport := newClient(
		t,
		response(http.StatusCreated, model),
		response(http.StatusOK, model),
		response(http.StatusNoContent, ""),
	)
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
	if _, err = GetRule(context.Background(), client, "rule/id"); err != nil {
		t.Fatal(err)
	}
	if err = DeleteRule(context.Background(), client, "rule/id"); err != nil {
		t.Fatal(err)
	}
	for i, expected := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/v2.0/security-group-rules"},
		{http.MethodGet, "/v2.0/security-group-rules/rule%2Fid"},
		{http.MethodDelete, "/v2.0/security-group-rules/rule%2Fid"},
	} {
		request := transport.requests[i]
		if request.Method != expected.method || request.URL.EscapedPath() != expected.path {
			t.Fatalf("request %d = %s %s", i, request.Method, request.URL.EscapedPath())
		}
	}
	body, _ := io.ReadAll(transport.requests[0].Body)
	if !strings.Contains(string(body), `"protocol":"6"`) ||
		!strings.Contains(string(body), `"remote_ip_prefix":"192.0.2.0/24"`) ||
		strings.Contains(string(body), "remote_group_id") {
		t.Fatalf("create body=%s", body)
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
	if len(rules) != 2 || len(transport.requests) != 2 {
		t.Fatalf("rules=%+v requests=%d", rules, len(transport.requests))
	}
	for _, request := range transport.requests {
		if request.URL.Query().Get("security_group_id") != "sg-id" {
			t.Fatalf("query=%s", request.URL.RawQuery)
		}
	}
}

func TestSecurityGroupRuleErrorsUseNotFoundAndConflictClasses(t *testing.T) {
	for _, testCase := range []struct {
		status int
		kind   string
		class  vpc.ErrorClass
		create bool
	}{
		{http.StatusNotFound, "SecurityGroupRuleNotFound", vpc.ErrorClassNotFound, false},
		{http.StatusConflict, "SecurityGroupRuleExists", vpc.ErrorClassConflict, true},
	} {
		client, transport := newClient(t, response(
			testCase.status,
			`{"NeutronError":{"type":"`+testCase.kind+`","message":"diagnostic"}}`,
		))
		var err error
		if testCase.create {
			_, err = CreateRule(context.Background(), client, RuleCreateRequest{})
		} else {
			err = DeleteRule(context.Background(), client, "rule-id")
		}
		if !vpc.IsErrorClass(err, testCase.class) {
			t.Fatalf("error=%v", err)
		}
		if len(transport.requests) != 1 {
			t.Fatalf("requests=%d", len(transport.requests))
		}
	}
}

func TestSecurityGroupRuleHasNoUpdateRequest(t *testing.T) {
	packageType := reflect.TypeOf(RuleCreateRequest{})
	if _, exists := packageType.FieldByName("Update"); exists {
		t.Fatal("rule update must not be exposed")
	}
}
