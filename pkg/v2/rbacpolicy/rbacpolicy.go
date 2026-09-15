// Package rbacpolicy provides operations for RBAC policy resources.
package rbacpolicy

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/rbac-policies"

type Policy struct {
	ID           string `json:"id"`
	ObjectType   string `json:"object_type"`
	ObjectID     string `json:"object_id"`
	Action       string `json:"action"`
	TargetTenant string `json:"target_tenant"`
	ProjectID    string `json:"project_id"`
}

type CreateRequest struct {
	ObjectType   string  `json:"object_type"`
	ObjectID     string  `json:"object_id"`
	Action       string  `json:"action"`
	TargetTenant string  `json:"target_tenant"`
	ProjectID    *string `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	TargetTenant string `json:"target_tenant"`
}

type envelope struct {
	Policy Policy `json:"rbac_policy"`
}

type listEnvelope struct {
	Policies []Policy       `json:"rbac_policies"`
	Links    []api.PageLink `json:"rbac_policies_links"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*Policy, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodPost, collectionPath, nil,
		struct {
			Policy CreateRequest `json:"rbac_policy"`
		}{Policy: request},
		&result, http.StatusCreated,
	)
	if err != nil {
		return nil, err
	}
	return &result.Policy, nil
}

func Get(ctx context.Context, client *vpc.Client, id string) (*Policy, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodGet, resourcePath(id), nil, nil, &result,
		http.StatusOK,
	)
	if err != nil {
		return nil, err
	}
	return &result.Policy, nil
}

func Update(
	ctx context.Context,
	client *vpc.Client,
	id string,
	request UpdateRequest,
) (*Policy, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodPut, resourcePath(id), nil,
		struct {
			Policy UpdateRequest `json:"rbac_policy"`
		}{Policy: request},
		&result, http.StatusOK,
	)
	if err != nil {
		return nil, err
	}
	return &result.Policy, nil
}

func Delete(ctx context.Context, client *vpc.Client, id string) error {
	return api.Request(ctx, client, http.MethodDelete, resourcePath(id), nil, nil, nil,
		http.StatusNoContent,
	)
}

func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.SelectionOptions,
) ([]Policy, error) {
	return api.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (api.Page[Policy], error) {
		var result listEnvelope
		err := api.Request(ctx, client, http.MethodGet, collectionPath, query, nil, &result,
			http.StatusOK,
		)
		return api.Page[Policy]{
			Items:    result.Policies,
			NextLink: api.NextPageLink(result.Links),
		}, err
	})
}

func resourcePath(id string) string {
	return collectionPath + "/" + url.PathEscape(id)
}
