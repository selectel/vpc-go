package firewallpolicy

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/fwaas/firewall_policies"

type FirewallPolicy struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	FirewallRules  []string `json:"firewall_rules"`
	Audited        bool     `json:"audited"`
	Shared         bool     `json:"shared"`
	ProjectID      string   `json:"project_id"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	RevisionNumber int      `json:"revision_number"`
}

type CreateRequest struct {
	Name          *string   `json:"name,omitempty"`
	Description   *string   `json:"description,omitempty"`
	FirewallRules *[]string `json:"firewall_rules,omitempty"`
	Audited       *bool     `json:"audited,omitempty"`
	ProjectID     *string   `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	Name          *string   `json:"name,omitempty"`
	Description   *string   `json:"description,omitempty"`
	FirewallRules *[]string `json:"firewall_rules,omitempty"`
	Audited       *bool     `json:"audited,omitempty"`
}

type envelope struct {
	FirewallPolicy FirewallPolicy `json:"firewall_policy"`
}

type listEnvelope struct {
	FirewallPolicies []FirewallPolicy `json:"firewall_policies"`
	Links            []link           `json:"firewall_policies_links"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*FirewallPolicy, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodPost,
		collectionPath,
		nil,
		struct {
			FirewallPolicy CreateRequest `json:"firewall_policy"`
		}{FirewallPolicy: request},
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallPolicy, nil
}

func Get(ctx context.Context, client *vpc.Client, firewallPolicyID string) (*FirewallPolicy, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodGet,
		resourcePath(firewallPolicyID),
		nil,
		nil,
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallPolicy, nil
}

func Update(
	ctx context.Context,
	client *vpc.Client,
	firewallPolicyID string,
	request UpdateRequest,
) (*FirewallPolicy, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodPut,
		resourcePath(firewallPolicyID),
		nil,
		struct {
			FirewallPolicy UpdateRequest `json:"firewall_policy"`
		}{FirewallPolicy: request},
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.FirewallPolicy, nil
}

func Delete(ctx context.Context, client *vpc.Client, firewallPolicyID string) error {
	return api.Request(ctx, client,
		http.MethodDelete,
		resourcePath(firewallPolicyID),
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
) ([]FirewallPolicy, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[FirewallPolicy], error) {
		var result listEnvelope
		err := api.Request(ctx, client,
			http.MethodGet,
			collectionPath,
			query,
			nil,
			&result,
			api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[FirewallPolicy]{
			Items:    result.FirewallPolicies,
			NextLink: nextLink(result.Links),
		}, err
	})
}

func resourcePath(firewallPolicyID string) string {
	return collectionPath + "/" + url.PathEscape(firewallPolicyID)
}

func nextLink(links []link) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}
	return ""
}
