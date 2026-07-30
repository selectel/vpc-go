package securitygroup

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const ruleCollectionPath = "/v2.0/security-group-rules"

// RemoteSource is an optional, mutually exclusive source selector for a rule.
// Values can only be constructed with ByRemoteIPPrefix or ByRemoteGroupID.
type RemoteSource interface {
	apply(*ruleCreatePayload)
}

type remoteIPPrefixSource struct {
	value string
}

func (source remoteIPPrefixSource) apply(payload *ruleCreatePayload) {
	payload.RemoteIPPrefix = &source.value
}

type remoteGroupSource struct {
	value string
}

func (source remoteGroupSource) apply(payload *ruleCreatePayload) {
	payload.RemoteGroupID = &source.value
}

// ByRemoteIPPrefix selects traffic by an IP prefix.
func ByRemoteIPPrefix(prefix string) RemoteSource {
	return remoteIPPrefixSource{value: prefix}
}

// ByRemoteGroupID selects traffic by a security group ID.
func ByRemoteGroupID(securityGroupID string) RemoteSource {
	return remoteGroupSource{value: securityGroupID}
}

// RuleCreateRequest contains caller-writable attributes for a new rule.
type RuleCreateRequest struct {
	SecurityGroupID string
	Direction       string
	EtherType       *string
	Protocol        *string
	PortRangeMin    *int
	PortRangeMax    *int
	RemoteSource    RemoteSource
	Description     *string
	ProjectID       *string
}

type ruleCreatePayload struct {
	SecurityGroupID string  `json:"security_group_id"`
	Direction       string  `json:"direction"`
	EtherType       *string `json:"ethertype,omitempty"`
	Protocol        *string `json:"protocol,omitempty"`
	PortRangeMin    *int    `json:"port_range_min,omitempty"`
	PortRangeMax    *int    `json:"port_range_max,omitempty"`
	RemoteIPPrefix  *string `json:"remote_ip_prefix,omitempty"`
	RemoteGroupID   *string `json:"remote_group_id,omitempty"`
	Description     *string `json:"description,omitempty"`
	ProjectID       *string `json:"project_id,omitempty"`
}

func (request RuleCreateRequest) payload() ruleCreatePayload {
	payload := ruleCreatePayload{
		SecurityGroupID: request.SecurityGroupID,
		Direction:       request.Direction,
		EtherType:       request.EtherType,
		Protocol:        request.Protocol,
		PortRangeMin:    request.PortRangeMin,
		PortRangeMax:    request.PortRangeMax,
		Description:     request.Description,
		ProjectID:       request.ProjectID,
	}
	if request.RemoteSource != nil {
		request.RemoteSource.apply(&payload)
	}
	return payload
}

type ruleEnvelope struct {
	SecurityGroupRule SecurityGroupRule `json:"security_group_rule"`
}

type ruleListEnvelope struct {
	SecurityGroupRules []SecurityGroupRule `json:"security_group_rules"`
	Links              []link              `json:"security_group_rules_links"`
}

func CreateRule(
	ctx context.Context,
	client *vpc.Client,
	request RuleCreateRequest,
) (*SecurityGroupRule, error) {
	var result ruleEnvelope
	err := api.Request(ctx, client,
		http.MethodPost,
		ruleCollectionPath,
		nil,
		struct {
			SecurityGroupRule ruleCreatePayload `json:"security_group_rule"`
		}{SecurityGroupRule: request.payload()},
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SecurityGroupRule, nil
}

func GetRule(
	ctx context.Context,
	client *vpc.Client,
	securityGroupRuleID string,
) (*SecurityGroupRule, error) {
	var result ruleEnvelope
	err := api.Request(ctx, client,
		http.MethodGet,
		ruleResourcePath(securityGroupRuleID),
		nil,
		nil,
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SecurityGroupRule, nil
}

func DeleteRule(
	ctx context.Context,
	client *vpc.Client,
	securityGroupRuleID string,
) error {
	return api.Request(ctx, client,
		http.MethodDelete,
		ruleResourcePath(securityGroupRuleID),
		nil,
		nil,
		nil,
		api.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}},
	)
}

func ListRules(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]SecurityGroupRule, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[SecurityGroupRule], error) {
		var result ruleListEnvelope
		err := api.Request(ctx, client,
			http.MethodGet,
			ruleCollectionPath,
			query,
			nil,
			&result,
			api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[SecurityGroupRule]{
			Items:    result.SecurityGroupRules,
			NextLink: nextLink(result.Links),
		}, err
	})
}

func ruleResourcePath(securityGroupRuleID string) string {
	return ruleCollectionPath + "/" + url.PathEscape(securityGroupRuleID)
}
