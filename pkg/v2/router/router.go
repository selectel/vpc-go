package router

import (
	"context"
	"net/http"
	"net/url"

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
	FlavorID              *string              `json:"flavor_id"`
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
	Routers []Router `json:"routers"`
	Links   []link   `json:"routers_links"`
}
type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*Router, error) {
	var result envelope
	err := client.Request(ctx, http.MethodPost, collectionPath, nil,
		struct {
			Router CreateRequest `json:"router"`
		}{request}, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusCreated}})
	if err != nil {
		return nil, err
	}
	return &result.Router, nil
}

func Get(ctx context.Context, client *vpc.Client, id string) (*Router, error) {
	var result envelope
	err := client.Request(ctx, http.MethodGet, resourcePath(id), nil, nil, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}})
	if err != nil {
		return nil, err
	}
	return &result.Router, nil
}

func Update(ctx context.Context, client *vpc.Client, id string, request UpdateRequest) (*Router, error) {
	var result envelope
	err := client.Request(ctx, http.MethodPut, resourcePath(id), nil,
		struct {
			Router UpdateRequest `json:"router"`
		}{request}, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}, ReadBlocked: blockedReader(client, id)})
	if err != nil {
		return nil, err
	}
	return &result.Router, nil
}

func Delete(ctx context.Context, client *vpc.Client, id string) error {
	return client.Request(ctx, http.MethodDelete, resourcePath(id), nil, nil, nil,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}, ReadBlocked: blockedReader(client, id)})
}

func List(ctx context.Context, client *vpc.Client, options vpc.ListOptions) ([]Router, error) {
	return vpc.WalkPages(ctx, options.Values(), func(ctx context.Context, query url.Values) (vpc.Page[Router], error) {
		var result listEnvelope
		err := client.Request(ctx, http.MethodGet, collectionPath, query, nil, &result,
			vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}})
		return vpc.Page[Router]{Items: result.Routers, NextLink: nextLink(result.Links)}, err
	})
}

type Tags struct{ operations vpc.TagOperations }

func TagOperations(client *vpc.Client, id string) Tags {
	return Tags{vpc.NewTagOperations(client, "routers", id, blockedReader(client, id))}
}
func (tags Tags) Get(ctx context.Context) ([]string, error) { return tags.operations.Get(ctx) }
func (tags Tags) Has(ctx context.Context, tag string) (bool, error) {
	return tags.operations.Has(ctx, tag)
}
func (tags Tags) Add(ctx context.Context, tag string) error { return tags.operations.Add(ctx, tag) }
func (tags Tags) Delete(ctx context.Context, tag string) error {
	return tags.operations.Delete(ctx, tag)
}
func (tags Tags) Replace(ctx context.Context, values []string) error {
	return tags.operations.Replace(ctx, values)
}
func (tags Tags) DeleteAll(ctx context.Context) error { return tags.operations.DeleteAll(ctx) }

func blockedReader(client *vpc.Client, id string) func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		router, err := Get(ctx, client, id)
		if err != nil {
			return false, err
		}
		return router.Blocked, nil
	}
}
func resourcePath(id string) string { return collectionPath + "/" + url.PathEscape(id) }
func nextLink(links []link) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}
	return ""
}
