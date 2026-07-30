package firewallpolicy

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

// InsertPosition selects exactly one placement strategy for InsertRule.
//
// AtBeginning places the rule at position 1. To append a rule, use AfterRule
// with the current last rule. Neutron itself accepts both insert_before and
// insert_after but silently gives insert_before precedence; this API prevents
// that ambiguous request.
type InsertPosition interface {
	apply(*insertRuleRequest)
}

type beginningPosition struct{}

func (beginningPosition) apply(*insertRuleRequest) {}

type beforePosition struct {
	ruleID string
}

func (position beforePosition) apply(request *insertRuleRequest) {
	request.InsertBefore = position.ruleID
}

type afterPosition struct {
	ruleID string
}

func (position afterPosition) apply(request *insertRuleRequest) {
	request.InsertAfter = position.ruleID
}

// AtBeginning inserts a rule at position 1, before all current rules.
func AtBeginning() InsertPosition {
	return beginningPosition{}
}

// BeforeRule inserts a rule immediately before neighboringRuleID.
func BeforeRule(neighboringRuleID string) InsertPosition {
	return beforePosition{ruleID: neighboringRuleID}
}

// AfterRule inserts a rule immediately after neighboringRuleID.
//
// Passing the current last rule appends the new rule to the policy.
func AfterRule(neighboringRuleID string) InsertPosition {
	return afterPosition{ruleID: neighboringRuleID}
}

type insertRuleRequest struct {
	FirewallRuleID string `json:"firewall_rule_id"`
	InsertBefore   string `json:"insert_before,omitempty"`
	InsertAfter    string `json:"insert_after,omitempty"`
}

type removeRuleRequest struct {
	FirewallRuleID string `json:"firewall_rule_id"`
}

// InsertRule atomically inserts firewallRuleID and returns the current policy.
func InsertRule(
	ctx context.Context,
	client *vpc.Client,
	firewallPolicyID string,
	firewallRuleID string,
	position InsertPosition,
) (*FirewallPolicy, error) {
	request := insertRuleRequest{FirewallRuleID: firewallRuleID}
	if position == nil || isNilPosition(position) {
		return nil, &vpc.ClientError{Err: errors.New("firewall rule insert position is required")}
	}
	position.apply(&request)
	if request.InsertBefore == "" && request.InsertAfter == "" {
		if _, beginning := position.(beginningPosition); !beginning {
			return nil, &vpc.ClientError{
				Err: errors.New("neighboring firewall rule ID must not be empty"),
			}
		}
	}

	var result envelope
	err := api.Request(ctx, client,
		http.MethodPut,
		resourcePath(firewallPolicyID)+"/insert_rule",
		nil,
		request,
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallPolicy, nil
}

func isNilPosition(position InsertPosition) bool {
	value := reflect.ValueOf(position)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

// RemoveRule atomically removes firewallRuleID and returns the current policy.
func RemoveRule(
	ctx context.Context,
	client *vpc.Client,
	firewallPolicyID string,
	firewallRuleID string,
) (*FirewallPolicy, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodPut,
		resourcePath(firewallPolicyID)+"/remove_rule",
		nil,
		removeRuleRequest{FirewallRuleID: firewallRuleID},
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallPolicy, nil
}
