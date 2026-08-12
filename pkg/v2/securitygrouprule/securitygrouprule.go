// Package securitygrouprule provides operations for security group rule resources.
package securitygrouprule

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/security-group-rules"

// Rule represents a security group rule.
type Rule struct {
	ID              string  `json:"id"`
	SecurityGroupID string  `json:"security_group_id"`
	Direction       string  `json:"direction"`
	EtherType       string  `json:"ethertype"`
	Protocol        *string `json:"protocol"`
	PortRangeMin    *int    `json:"port_range_min"`
	PortRangeMax    *int    `json:"port_range_max"`
	RemoteIPPrefix  *string `json:"remote_ip_prefix"`
	RemoteGroupID   *string `json:"remote_group_id"`
	Description     string  `json:"description"`
	ProjectID       string  `json:"project_id"`
	RevisionNumber  int     `json:"revision_number"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// CreateRequest contains caller-writable attributes for a new rule.
type CreateRequest struct {
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

type envelope struct {
	SecurityGroupRule Rule `json:"security_group_rule"`
}

type listEnvelope struct {
	SecurityGroupRules []Rule         `json:"security_group_rules"`
	Links              []api.PageLink `json:"security_group_rules_links"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*Rule, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodPost,
		collectionPath,
		nil,
		struct {
			SecurityGroupRule CreateRequest `json:"security_group_rule"`
		}{SecurityGroupRule: request},
		&result,
		http.StatusCreated,
	)
	if err != nil {
		return nil, err
	}
	return &result.SecurityGroupRule, nil
}

func Get(ctx context.Context, client *vpc.Client, securityGroupRuleID string) (*Rule, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodGet,
		resourcePath(securityGroupRuleID),
		nil,
		nil,
		&result,
		http.StatusOK,
	)
	if err != nil {
		return nil, err
	}
	return &result.SecurityGroupRule, nil
}

func Delete(ctx context.Context, client *vpc.Client, securityGroupRuleID string) error {
	return api.Request(ctx, client,
		http.MethodDelete,
		resourcePath(securityGroupRuleID),
		nil,
		nil,
		nil,
		http.StatusNoContent,
	)
}

func List(ctx context.Context, client *vpc.Client, options vpc.ListOptions) ([]Rule, error) {
	return api.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (api.Page[Rule], error) {
		var result listEnvelope
		err := api.Request(ctx, client,
			http.MethodGet,
			collectionPath,
			query,
			nil,
			&result,
			http.StatusOK,
		)
		return api.Page[Rule]{
			Items:    result.SecurityGroupRules,
			NextLink: api.NextPageLink(result.Links),
		}, err
	})
}

func resourcePath(securityGroupRuleID string) string {
	return collectionPath + "/" + url.PathEscape(securityGroupRuleID)
}
