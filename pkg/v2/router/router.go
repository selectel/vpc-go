package router

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/routers"

type ExternalFixedIP struct {
	SubnetID  string `json:"subnet_id"`
	IPAddress string `json:"ip_address"`
}

type ExternalGatewayInfo struct {
	NetworkID        string            `json:"network_id"`
	EnableSNAT       bool              `json:"enable_snat"`
	ExternalFixedIPs []ExternalFixedIP `json:"external_fixed_ips"`
}

type ExternalGatewayRequest struct {
	NetworkID  string `json:"network_id"`
	EnableSNAT *bool  `json:"enable_snat,omitempty"`
}

type Route struct {
	Destination string `json:"destination"`
	NextHop     string `json:"nexthop"`
}

type Router struct {
	ID                    string               `json:"id"`
	Name                  string               `json:"name"`
	Description           string               `json:"description"`
	Status                string               `json:"status"`
	AdminStateUp          bool                 `json:"admin_state_up"`
	ExternalGateway       *ExternalGatewayInfo `json:"external_gateway_info"`
	Routes                []Route              `json:"routes"`
	AvailabilityZones     []string             `json:"availability_zones"`
	AvailabilityZoneHints []string             `json:"availability_zone_hints"`
	ProjectID             string               `json:"project_id"`
	RevisionNumber        int                  `json:"revision_number"`
	CreatedAt             string               `json:"created_at"`
	UpdatedAt             string               `json:"updated_at"`
	Tags                  []string             `json:"tags"`
	Blocked               bool                 `json:"blocked"`
}

type CreateRequest struct {
	Name                  *string                               `json:"name,omitempty"`
	Description           *string                               `json:"description,omitempty"`
	AdminStateUp          *bool                                 `json:"admin_state_up,omitempty"`
	ExternalGateway       *vpc.Optional[ExternalGatewayRequest] `json:"external_gateway_info,omitempty"`
	AvailabilityZoneHints *[]string                             `json:"availability_zone_hints,omitempty"`
	ProjectID             *string                               `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	Name            *string                               `json:"name,omitempty"`
	Description     *string                               `json:"description,omitempty"`
	AdminStateUp    *bool                                 `json:"admin_state_up,omitempty"`
	ExternalGateway *vpc.Optional[ExternalGatewayRequest] `json:"external_gateway_info,omitempty"`
	Routes          *[]Route                              `json:"routes,omitempty"`
}

type envelope struct {
	Router Router `json:"router"`
}
type listEnvelope struct {
	Routers []Router       `json:"routers"`
	Links   []api.PageLink `json:"routers_links"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*Router, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodPost, collectionPath, nil,
		struct {
			Router CreateRequest `json:"router"`
		}{Router: request}, &result,
		http.StatusCreated)
	if err != nil {
		return nil, err
	}
	return &result.Router, nil
}

func Get(ctx context.Context, client *vpc.Client, id string) (*Router, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodGet, resourcePath(id), nil, nil, &result,
		http.StatusOK)
	if err != nil {
		return nil, err
	}
	return &result.Router, nil
}

func Update(ctx context.Context, client *vpc.Client, id string, request UpdateRequest) (*Router, error) {
	var result envelope
	err := api.Request(ctx, client, http.MethodPut, resourcePath(id), nil,
		struct {
			Router UpdateRequest `json:"router"`
		}{Router: request}, &result,
		http.StatusOK)
	if err != nil {
		return nil, err
	}
	return &result.Router, nil
}

func Delete(ctx context.Context, client *vpc.Client, id string) error {
	return api.Request(ctx, client, http.MethodDelete, resourcePath(id), nil, nil, nil,
		http.StatusNoContent)
}

func List(ctx context.Context, client *vpc.Client, options vpc.ListOptions) ([]Router, error) {
	return api.WalkPages(ctx, options.Values(), func(ctx context.Context, query url.Values) (api.Page[Router], error) {
		var result listEnvelope
		err := api.Request(ctx, client, http.MethodGet, collectionPath, query, nil, &result,
			http.StatusOK)
		return api.Page[Router]{Items: result.Routers, NextLink: api.NextPageLink(result.Links)}, err
	})
}

type Tags struct{ operations api.TagOperations }

func TagOperations(client *vpc.Client, id string) Tags {
	return Tags{api.NewTagOperations(client, "routers", id)}
}
func (tags Tags) Get(ctx context.Context) ([]string, error) { return tags.operations.Get(ctx) }
func (tags Tags) Has(ctx context.Context, tag string) (bool, error) {
	return tags.operations.Has(ctx, tag)
}
func (tags Tags) Add(ctx context.Context, tag string) error { return tags.operations.Add(ctx, tag) }
func (tags Tags) Delete(ctx context.Context, tag string) error {
	return tags.operations.Delete(ctx, tag)
}

func (tags Tags) Replace(ctx context.Context, values []string) ([]string, error) {
	return tags.operations.Replace(ctx, values)
}
func (tags Tags) DeleteAll(ctx context.Context) error { return tags.operations.DeleteAll(ctx) }

func resourcePath(id string) string { return collectionPath + "/" + url.PathEscape(id) }
