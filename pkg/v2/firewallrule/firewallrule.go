package firewallrule

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/fwaas/firewall_rules"

type FirewallRule struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Protocol             *string  `json:"protocol"`
	IPVersion            int      `json:"ip_version"`
	SourceIPAddress      *string  `json:"source_ip_address"`
	DestinationIPAddress *string  `json:"destination_ip_address"`
	SourcePort           *string  `json:"source_port"`
	DestinationPort      *string  `json:"destination_port"`
	Action               string   `json:"action"`
	Enabled              bool     `json:"enabled"`
	Shared               bool     `json:"shared"`
	FirewallPolicyIDs    []string `json:"firewall_policy_id"`
	ProjectID            string   `json:"project_id"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
	RevisionNumber       int      `json:"revision_number"`
}

type CreateRequest struct {
	Name                 *string               `json:"name,omitempty"`
	Description          *string               `json:"description,omitempty"`
	Protocol             *vpc.Optional[string] `json:"protocol,omitempty"`
	IPVersion            *int                  `json:"ip_version,omitempty"`
	SourceIPAddress      *vpc.Optional[string] `json:"source_ip_address,omitempty"`
	DestinationIPAddress *vpc.Optional[string] `json:"destination_ip_address,omitempty"`
	SourcePort           *vpc.Optional[string] `json:"source_port,omitempty"`
	DestinationPort      *vpc.Optional[string] `json:"destination_port,omitempty"`
	Action               *string               `json:"action,omitempty"`
	Enabled              *bool                 `json:"enabled,omitempty"`
	ProjectID            *string               `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	Name                 *string               `json:"name,omitempty"`
	Description          *string               `json:"description,omitempty"`
	Protocol             *vpc.Optional[string] `json:"protocol,omitempty"`
	IPVersion            *int                  `json:"ip_version,omitempty"`
	SourceIPAddress      *vpc.Optional[string] `json:"source_ip_address,omitempty"`
	DestinationIPAddress *vpc.Optional[string] `json:"destination_ip_address,omitempty"`
	SourcePort           *vpc.Optional[string] `json:"source_port,omitempty"`
	DestinationPort      *vpc.Optional[string] `json:"destination_port,omitempty"`
	Action               *string               `json:"action,omitempty"`
	Enabled              *bool                 `json:"enabled,omitempty"`
}

type envelope struct {
	FirewallRule FirewallRule `json:"firewall_rule"`
}

type listEnvelope struct {
	FirewallRules []FirewallRule `json:"firewall_rules"`
	Links         []link         `json:"firewall_rules_links"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*FirewallRule, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodPost,
		collectionPath,
		nil,
		struct {
			FirewallRule CreateRequest `json:"firewall_rule"`
		}{FirewallRule: request},
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallRule, nil
}

func Get(ctx context.Context, client *vpc.Client, firewallRuleID string) (*FirewallRule, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodGet,
		resourcePath(firewallRuleID),
		nil,
		nil,
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallRule, nil
}

func Update(
	ctx context.Context,
	client *vpc.Client,
	firewallRuleID string,
	request UpdateRequest,
) (*FirewallRule, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodPut,
		resourcePath(firewallRuleID),
		nil,
		struct {
			FirewallRule UpdateRequest `json:"firewall_rule"`
		}{FirewallRule: request},
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallRule, nil
}

func Delete(ctx context.Context, client *vpc.Client, firewallRuleID string) error {
	return api.Request(ctx, client,
		http.MethodDelete,
		resourcePath(firewallRuleID),
		nil,
		nil,
		nil,
		api.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}},
	)
}

func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]FirewallRule, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[FirewallRule], error) {
		var result listEnvelope
		err := api.Request(ctx, client,
			http.MethodGet,
			collectionPath,
			query,
			nil,
			&result,
			api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[FirewallRule]{
			Items:    result.FirewallRules,
			NextLink: nextLink(result.Links),
		}, err
	})
}

func resourcePath(firewallRuleID string) string {
	return collectionPath + "/" + url.PathEscape(firewallRuleID)
}

func nextLink(links []link) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}
	return ""
}
