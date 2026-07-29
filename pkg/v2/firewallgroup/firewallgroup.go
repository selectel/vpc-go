package firewallgroup

import (
	"context"
	"net/http"
	"net/url"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/fwaas/firewall_groups"

type FirewallGroup struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	Description             string   `json:"description"`
	Status                  string   `json:"status"`
	AdminStateUp            bool     `json:"admin_state_up"`
	Ports                   []string `json:"ports"`
	IngressFirewallPolicyID *string  `json:"ingress_firewall_policy_id"`
	EgressFirewallPolicyID  *string  `json:"egress_firewall_policy_id"`
	Shared                  bool     `json:"shared"`
	ProjectID               string   `json:"project_id"`
	CreatedAt               string   `json:"created_at"`
	UpdatedAt               string   `json:"updated_at"`
	RevisionNumber          int      `json:"revision_number"`
}

type CreateRequest struct {
	Name                    *string               `json:"name,omitempty"`
	Description             *string               `json:"description,omitempty"`
	AdminStateUp            *bool                 `json:"admin_state_up,omitempty"`
	Ports                   *[]string             `json:"ports,omitempty"`
	IngressFirewallPolicyID *vpc.Optional[string] `json:"ingress_firewall_policy_id,omitempty"`
	EgressFirewallPolicyID  *vpc.Optional[string] `json:"egress_firewall_policy_id,omitempty"`
	ProjectID               *string               `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	Name                    *string               `json:"name,omitempty"`
	Description             *string               `json:"description,omitempty"`
	AdminStateUp            *bool                 `json:"admin_state_up,omitempty"`
	Ports                   *[]string             `json:"ports,omitempty"`
	IngressFirewallPolicyID *vpc.Optional[string] `json:"ingress_firewall_policy_id,omitempty"`
	EgressFirewallPolicyID  *vpc.Optional[string] `json:"egress_firewall_policy_id,omitempty"`
}

type envelope struct {
	FirewallGroup FirewallGroup `json:"firewall_group"`
}

type listEnvelope struct {
	FirewallGroups []FirewallGroup `json:"firewall_groups"`
	Links          []link          `json:"firewall_groups_links"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*FirewallGroup, error) {
	var result envelope
	err := client.Request(
		ctx,
		http.MethodPost,
		collectionPath,
		nil,
		struct {
			FirewallGroup CreateRequest `json:"firewall_group"`
		}{FirewallGroup: request},
		&result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallGroup, nil
}

func Get(ctx context.Context, client *vpc.Client, firewallGroupID string) (*FirewallGroup, error) {
	var result envelope
	err := client.Request(
		ctx,
		http.MethodGet,
		resourcePath(firewallGroupID),
		nil,
		nil,
		&result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallGroup, nil
}

func Update(
	ctx context.Context,
	client *vpc.Client,
	firewallGroupID string,
	request UpdateRequest,
) (*FirewallGroup, error) {
	var result envelope
	err := client.Request(
		ctx,
		http.MethodPut,
		resourcePath(firewallGroupID),
		nil,
		struct {
			FirewallGroup UpdateRequest `json:"firewall_group"`
		}{FirewallGroup: request},
		&result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallGroup, nil
}

func Delete(ctx context.Context, client *vpc.Client, firewallGroupID string) error {
	return client.Request(
		ctx,
		http.MethodDelete,
		resourcePath(firewallGroupID),
		nil,
		nil,
		nil,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}},
	)
}

func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]FirewallGroup, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[FirewallGroup], error) {
		var result listEnvelope
		err := client.Request(
			ctx,
			http.MethodGet,
			collectionPath,
			query,
			nil,
			&result,
			vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[FirewallGroup]{
			Items:    result.FirewallGroups,
			NextLink: nextLink(result.Links),
		}, err
	})
}

func resourcePath(firewallGroupID string) string {
	return collectionPath + "/" + url.PathEscape(firewallGroupID)
}

func nextLink(links []link) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}
	return ""
}
