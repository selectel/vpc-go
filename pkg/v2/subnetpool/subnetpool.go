package subnetpool

import (
	"context"
	"net/http"
	"net/url"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/subnetpools"

type SubnetPool struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Prefixes         []string `json:"prefixes"`
	DefaultQuota     *int     `json:"default_quota"`
	DefaultPrefixLen int      `json:"default_prefixlen"`
	MinPrefixLen     int      `json:"min_prefixlen"`
	MaxPrefixLen     int      `json:"max_prefixlen"`
	AddressScopeID   *string  `json:"address_scope_id"`
	IPVersion        int      `json:"ip_version"`
	IsDefault        bool     `json:"is_default"`
	Shared           bool     `json:"shared"`
	ProjectID        string   `json:"project_id"`
	RevisionNumber   int      `json:"revision_number"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	Tags             []string `json:"tags"`
}

type CreateRequest struct {
	Name             *string   `json:"name,omitempty"`
	Description      *string   `json:"description,omitempty"`
	Prefixes         *[]string `json:"prefixes,omitempty"`
	DefaultQuota     *int      `json:"default_quota,omitempty"`
	DefaultPrefixLen *int      `json:"default_prefixlen,omitempty"`
	MinPrefixLen     *int      `json:"min_prefixlen,omitempty"`
	MaxPrefixLen     *int      `json:"max_prefixlen,omitempty"`
	AddressScopeID   *string   `json:"address_scope_id,omitempty"`
	ProjectID        *string   `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	Name             *string   `json:"name,omitempty"`
	Description      *string   `json:"description,omitempty"`
	Prefixes         *[]string `json:"prefixes,omitempty"`
	DefaultQuota     *int      `json:"default_quota,omitempty"`
	DefaultPrefixLen *int      `json:"default_prefixlen,omitempty"`
	MinPrefixLen     *int      `json:"min_prefixlen,omitempty"`
	MaxPrefixLen     *int      `json:"max_prefixlen,omitempty"`
	AddressScopeID   *string   `json:"address_scope_id,omitempty"`
}

type envelope struct {
	SubnetPool SubnetPool `json:"subnetpool"`
}

type listEnvelope struct {
	SubnetPools []SubnetPool `json:"subnetpools"`
	Links       []link       `json:"subnetpools_links"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*SubnetPool, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodPost, collectionPath, nil,
		struct {
			SubnetPool CreateRequest `json:"subnetpool"`
		}{SubnetPool: request},
		&result, vpc.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SubnetPool, nil
}

func Get(ctx context.Context, client *vpc.Client, subnetPoolID string) (*SubnetPool, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodGet, resourcePath(subnetPoolID), nil, nil, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SubnetPool, nil
}

func Update(
	ctx context.Context,
	client *vpc.Client,
	subnetPoolID string,
	request UpdateRequest,
) (*SubnetPool, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodPut, resourcePath(subnetPoolID), nil,
		struct {
			SubnetPool UpdateRequest `json:"subnetpool"`
		}{SubnetPool: request},
		&result, vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SubnetPool, nil
}

func Delete(ctx context.Context, client *vpc.Client, subnetPoolID string) error {
	return client.Request(
		ctx, http.MethodDelete, resourcePath(subnetPoolID), nil, nil, nil,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}},
	)
}

func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]SubnetPool, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[SubnetPool], error) {
		var result listEnvelope
		err := client.Request(
			ctx, http.MethodGet, collectionPath, query, nil, &result,
			vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[SubnetPool]{
			Items: result.SubnetPools, NextLink: nextLink(result.Links),
		}, err
	})
}

// Tags intentionally exposes read-only tag operations for a subnet pool.
type Tags struct {
	operations vpc.TagOperations
}

func TagOperations(client *vpc.Client, subnetPoolID string) Tags {
	return Tags{operations: vpc.NewTagOperations(client, "subnetpools", subnetPoolID, nil)}
}

func (tags Tags) Get(ctx context.Context) ([]string, error) {
	return tags.operations.Get(ctx)
}

func (tags Tags) Has(ctx context.Context, tag string) (bool, error) {
	return tags.operations.Has(ctx, tag)
}

func resourcePath(id string) string {
	return collectionPath + "/" + url.PathEscape(id)
}

func nextLink(links []link) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}
	return ""
}
