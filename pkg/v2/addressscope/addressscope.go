// Package addressscope provides owner-scoped operations for Neutron address scopes.
//
// Neutron can mask a policy rejection on update or delete as a 404 response.
// A not-found error from those operations therefore does not prove absence.
package addressscope

import (
	"context"
	"net/http"
	"net/url"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/address-scopes"

type AddressScope struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IPVersion int    `json:"ip_version"`
	Shared    bool   `json:"shared"`
	ProjectID string `json:"project_id"`
}

type CreateRequest struct {
	Name      *string `json:"name,omitempty"`
	IPVersion *int    `json:"ip_version,omitempty"`
	ProjectID *string `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	Name *string `json:"name,omitempty"`
}

type envelope struct {
	AddressScope AddressScope `json:"address_scope"`
}

type listEnvelope struct {
	AddressScopes []AddressScope `json:"address_scopes"`
	Links         []link         `json:"address_scopes_links"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func Create(
	ctx context.Context,
	client *vpc.Client,
	request CreateRequest,
) (*AddressScope, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodPost, collectionPath, nil,
		struct {
			AddressScope CreateRequest `json:"address_scope"`
		}{AddressScope: request},
		&result, vpc.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.AddressScope, nil
}

func Get(ctx context.Context, client *vpc.Client, id string) (*AddressScope, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodGet, resourcePath(id), nil, nil, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.AddressScope, nil
}

func Update(
	ctx context.Context,
	client *vpc.Client,
	id string,
	request UpdateRequest,
) (*AddressScope, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodPut, resourcePath(id), nil,
		struct {
			AddressScope UpdateRequest `json:"address_scope"`
		}{AddressScope: request},
		&result, vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.AddressScope, nil
}

func Delete(ctx context.Context, client *vpc.Client, id string) error {
	return client.Request(
		ctx, http.MethodDelete, resourcePath(id), nil, nil, nil,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}},
	)
}

func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]AddressScope, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[AddressScope], error) {
		var result listEnvelope
		err := client.Request(
			ctx, http.MethodGet, collectionPath, query, nil, &result,
			vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[AddressScope]{
			Items: result.AddressScopes, NextLink: nextLink(result.Links),
		}, err
	})
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
