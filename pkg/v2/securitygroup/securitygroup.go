// Package securitygroup provides operations for security group resources.
package securitygroup

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/security-groups"

// SecurityGroup represents a security group.
type SecurityGroup struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Description        string              `json:"description"`
	Stateful           bool                `json:"stateful"`
	Shared             bool                `json:"shared"`
	SecurityGroupRules []SecurityGroupRule `json:"security_group_rules"`
	ProjectID          string              `json:"project_id"`
	RevisionNumber     int                 `json:"revision_number"`
	CreatedAt          string              `json:"created_at"`
	UpdatedAt          string              `json:"updated_at"`
	Tags               []string            `json:"tags"`
}

// SecurityGroupRule is an observable rule embedded in a security group.
//
//nolint:revive // The explicit name avoids ambiguity with rules from other resource packages.
type SecurityGroupRule struct {
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

// CreateRequest contains the caller-writable attributes of a new group.
//
// Stateful is a pointer so an omitted value keeps the API default of true,
// distinct from an explicit false.
type CreateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Stateful    *bool   `json:"stateful,omitempty"`
	ProjectID   *string `json:"project_id,omitempty"`
}

// UpdateRequest contains the caller-writable attributes of an existing group.
type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Stateful    *bool   `json:"stateful,omitempty"`
}

type envelope struct {
	SecurityGroup SecurityGroup `json:"security_group"`
}

type listEnvelope struct {
	SecurityGroups []SecurityGroup `json:"security_groups"`
	Links          []api.PageLink  `json:"security_groups_links"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*SecurityGroup, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodPost,
		collectionPath,
		nil,
		struct {
			SecurityGroup CreateRequest `json:"security_group"`
		}{SecurityGroup: request},
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SecurityGroup, nil
}

func Get(ctx context.Context, client *vpc.Client, securityGroupID string) (*SecurityGroup, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodGet,
		resourcePath(securityGroupID),
		nil,
		nil,
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SecurityGroup, nil
}

func Update(
	ctx context.Context,
	client *vpc.Client,
	securityGroupID string,
	request UpdateRequest,
) (*SecurityGroup, error) {
	var result envelope
	err := api.Request(ctx, client,
		http.MethodPut,
		resourcePath(securityGroupID),
		nil,
		struct {
			SecurityGroup UpdateRequest `json:"security_group"`
		}{SecurityGroup: request},
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.SecurityGroup, nil
}

func Delete(ctx context.Context, client *vpc.Client, securityGroupID string) error {
	return api.Request(ctx, client,
		http.MethodDelete,
		resourcePath(securityGroupID),
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
) ([]SecurityGroup, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[SecurityGroup], error) {
		var result listEnvelope
		err := api.Request(ctx, client,
			http.MethodGet,
			collectionPath,
			query,
			nil,
			&result,
			api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[SecurityGroup]{
			Items:    result.SecurityGroups,
			NextLink: api.NextPageLink(result.Links),
		}, err
	})
}

type Tags struct {
	operations api.TagOperations
}

func TagOperations(client *vpc.Client, securityGroupID string) Tags {
	return Tags{operations: api.NewTagOperations(client, "security-groups", securityGroupID)}
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

func resourcePath(securityGroupID string) string {
	return collectionPath + "/" + url.PathEscape(securityGroupID)
}
