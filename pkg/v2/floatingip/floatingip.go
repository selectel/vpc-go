// Package floatingip provides operations for floating IP resources.
package floatingip

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/floatingips"

type FloatingIP struct {
	ID                string   `json:"id"`
	FloatingNetworkID string   `json:"floating_network_id"`
	PortID            *string  `json:"port_id"`
	FixedIPAddress    *string  `json:"fixed_ip_address"`
	FloatingIPAddress string   `json:"floating_ip_address"`
	RouterID          *string  `json:"router_id"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	ProjectID         string   `json:"project_id"`
	RevisionNumber    int      `json:"revision_number"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	Tags              []string `json:"tags"`
	Blocked           bool     `json:"blocked"`
}

type CreateRequest struct {
	FloatingNetworkID string  `json:"floating_network_id"`
	SubnetID          *string `json:"subnet_id,omitempty"`
	PortID            *string `json:"port_id,omitempty"`
	FixedIPAddress    *string `json:"fixed_ip_address,omitempty"`
	Description       *string `json:"description,omitempty"`
	ProjectID         *string `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	PortID         *vpc.Optional[string] `json:"port_id,omitempty"`
	FixedIPAddress *vpc.Optional[string] `json:"fixed_ip_address,omitempty"`
	Description    *string               `json:"description,omitempty"`
}

type envelope struct {
	FloatingIP FloatingIP `json:"floatingip"`
}
type listEnvelope struct {
	FloatingIPs []FloatingIP   `json:"floatingips"`
	Links       []api.PageLink `json:"floatingips_links"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*FloatingIP, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodPost, collectionPath, nil,
		struct {
			FloatingIP CreateRequest `json:"floatingip"`
		}{request}, &result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusCreated}})
	if err != nil {
		return nil, err
	}
	return &result.FloatingIP, nil
}

func Get(ctx context.Context, client *vpc.Client, id string) (*FloatingIP, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodGet, resourcePath(id), nil, nil, &result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}})
	if err != nil {
		return nil, err
	}
	return &result.FloatingIP, nil
}

func Update(ctx context.Context, client *vpc.Client, id string, request UpdateRequest) (*FloatingIP, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodPut, resourcePath(id), nil,
		struct {
			FloatingIP UpdateRequest `json:"floatingip"`
		}{request}, &result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}})
	if err != nil {
		return nil, err
	}
	return &result.FloatingIP, nil
}

func Delete(ctx context.Context, client *vpc.Client, id string) error {
	return api.Request(ctx, client, http.MethodDelete, resourcePath(id), nil, nil, nil,
		api.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}})
}

func List(ctx context.Context, client *vpc.Client, options vpc.ListOptions) ([]FloatingIP, error) {
	return vpc.WalkPages(ctx, options.Values(), func(ctx context.Context, q url.Values) (vpc.Page[FloatingIP], error) {
		var result listEnvelope
		err := api.Request(ctx, client, http.MethodGet, collectionPath, q, nil, &result, api.RequestOptions{ExpectedStatus: []int{http.StatusOK}})
		return vpc.Page[FloatingIP]{Items: result.FloatingIPs, NextLink: api.NextPageLink(result.Links)}, err
	})
}

type Tags struct{ operations api.TagOperations }

func TagOperations(client *vpc.Client, id string) Tags {
	return Tags{api.NewTagOperations(client, "floatingips", id)}
}
func (t Tags) Get(ctx context.Context) ([]string, error)         { return t.operations.Get(ctx) }
func (t Tags) Has(ctx context.Context, tag string) (bool, error) { return t.operations.Has(ctx, tag) }
func (t Tags) Add(ctx context.Context, tag string) error         { return t.operations.Add(ctx, tag) }
func (t Tags) Delete(ctx context.Context, tag string) error      { return t.operations.Delete(ctx, tag) }
func (t Tags) Replace(ctx context.Context, v []string) ([]string, error) {
	return t.operations.Replace(ctx, v)
}
func (t Tags) DeleteAll(ctx context.Context) error { return t.operations.DeleteAll(ctx) }
func resourcePath(id string) string                { return collectionPath + "/" + url.PathEscape(id) }
