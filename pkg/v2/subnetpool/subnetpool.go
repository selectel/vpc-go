// Package subnetpool provides operations for subnet pool resources.
package subnetpool

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

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
	Name             *string  `json:"name,omitempty"`
	Description      *string  `json:"description,omitempty"`
	Prefixes         []string `json:"prefixes"`
	DefaultQuota     *int     `json:"default_quota,omitempty"`
	DefaultPrefixLen *int     `json:"default_prefixlen,omitempty"`
	MinPrefixLen     *int     `json:"min_prefixlen,omitempty"`
	MaxPrefixLen     *int     `json:"max_prefixlen,omitempty"`
	ProjectID        *string  `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	Name             *string   `json:"name,omitempty"`
	Description      *string   `json:"description,omitempty"`
	Prefixes         *[]string `json:"prefixes,omitempty"`
	DefaultQuota     *int      `json:"default_quota,omitempty"`
	DefaultPrefixLen *int      `json:"default_prefixlen,omitempty"`
	MinPrefixLen     *int      `json:"min_prefixlen,omitempty"`
	MaxPrefixLen     *int      `json:"max_prefixlen,omitempty"`
}

type envelope struct {
	SubnetPool SubnetPool `json:"subnetpool"`
}

type listEnvelope struct {
	SubnetPools []SubnetPool   `json:"subnetpools"`
	Links       []api.PageLink `json:"subnetpools_links"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*SubnetPool, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodPost, collectionPath, nil,
		struct {
			SubnetPool CreateRequest `json:"subnetpool"`
		}{SubnetPool: request},
		&result, api.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SubnetPool, nil
}

func Get(ctx context.Context, client *vpc.Client, subnetPoolID string) (*SubnetPool, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodGet, resourcePath(subnetPoolID), nil, nil, &result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
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
	err := api.Request(ctx, client, http.MethodPut, resourcePath(subnetPoolID), nil,
		struct {
			SubnetPool UpdateRequest `json:"subnetpool"`
		}{SubnetPool: request},
		&result, api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SubnetPool, nil
}

func Delete(ctx context.Context, client *vpc.Client, subnetPoolID string) error {
	return api.Request(ctx, client, http.MethodDelete, resourcePath(subnetPoolID), nil, nil, nil,
		api.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}},
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
		err := api.Request(ctx, client, http.MethodGet, collectionPath, query, nil, &result,
			api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[SubnetPool]{
			Items: result.SubnetPools, NextLink: api.NextPageLink(result.Links),
		}, err
	})
}

type Tags struct {
	operations api.TagOperations
}

func TagOperations(client *vpc.Client, subnetPoolID string) Tags {
	return Tags{operations: api.NewTagOperations(client, "subnetpools", subnetPoolID)}
}

func (tags Tags) Get(ctx context.Context) ([]string, error) {
	return tags.operations.Get(ctx)
}

func (tags Tags) Has(ctx context.Context, tag string) (bool, error) {
	return tags.operations.Has(ctx, tag)
}

func (tags Tags) Add(ctx context.Context, tag string) error {
	return tags.operations.Add(ctx, tag)
}

func (tags Tags) Delete(ctx context.Context, tag string) error {
	return tags.operations.Delete(ctx, tag)
}

func (tags Tags) Replace(ctx context.Context, values []string) ([]string, error) {
	return tags.operations.Replace(ctx, values)
}

func (tags Tags) DeleteAll(ctx context.Context) error {
	return tags.operations.DeleteAll(ctx)
}

func resourcePath(id string) string {
	return collectionPath + "/" + url.PathEscape(id)
}
