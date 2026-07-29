package subnet

import (
	"context"
	"net/http"
	"net/url"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/subnets"

type AllocationPool struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type HostRoute struct {
	Destination string `json:"destination"`
	NextHop     string `json:"nexthop"`
}

type Subnet struct {
	ID              string           `json:"id"`
	NetworkID       string           `json:"network_id"`
	IPVersion       int              `json:"ip_version"`
	CIDR            string           `json:"cidr"`
	SubnetPoolID    *string          `json:"subnetpool_id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	GatewayIP       *string          `json:"gateway_ip"`
	AllocationPools []AllocationPool `json:"allocation_pools"`
	DNSNameservers  []string         `json:"dns_nameservers"`
	HostRoutes      []HostRoute      `json:"host_routes"`
	EnableDHCP      bool             `json:"enable_dhcp"`
	IPv6RAMode      *string          `json:"ipv6_ra_mode"`
	IPv6AddressMode *string          `json:"ipv6_address_mode"`
	ProjectID       string           `json:"project_id"`
	ServiceTypes    []string         `json:"service_types"`
	RevisionNumber  int              `json:"revision_number"`
	CreatedAt       string           `json:"created_at"`
	UpdatedAt       string           `json:"updated_at"`
	Tags            []string         `json:"tags"`
	Blocked         bool             `json:"blocked"`
}

type CreateRequest struct {
	NetworkID       string                `json:"network_id"`
	IPVersion       *int                  `json:"ip_version,omitempty"`
	CIDR            *string               `json:"cidr,omitempty"`
	SubnetPoolID    *string               `json:"subnetpool_id,omitempty"`
	PrefixLength    *int                  `json:"prefixlen,omitempty"`
	Name            *string               `json:"name,omitempty"`
	Description     *string               `json:"description,omitempty"`
	GatewayIP       *vpc.Optional[string] `json:"gateway_ip,omitempty"`
	AllocationPools *[]AllocationPool     `json:"allocation_pools,omitempty"`
	DNSNameservers  *[]string             `json:"dns_nameservers,omitempty"`
	HostRoutes      *[]HostRoute          `json:"host_routes,omitempty"`
	EnableDHCP      *bool                 `json:"enable_dhcp,omitempty"`
	IPv6RAMode      *string               `json:"ipv6_ra_mode,omitempty"`
	IPv6AddressMode *string               `json:"ipv6_address_mode,omitempty"`
	ProjectID       *string               `json:"project_id,omitempty"`
}

type UpdateRequest struct {
	Name            *string               `json:"name,omitempty"`
	Description     *string               `json:"description,omitempty"`
	GatewayIP       *vpc.Optional[string] `json:"gateway_ip,omitempty"`
	AllocationPools *[]AllocationPool     `json:"allocation_pools,omitempty"`
	DNSNameservers  *[]string             `json:"dns_nameservers,omitempty"`
	HostRoutes      *[]HostRoute          `json:"host_routes,omitempty"`
	EnableDHCP      *bool                 `json:"enable_dhcp,omitempty"`
}

type envelope struct {
	Subnet Subnet `json:"subnet"`
}

type listEnvelope struct {
	Subnets []Subnet `json:"subnets"`
	Links   []link   `json:"subnets_links"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*Subnet, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodPost, collectionPath, nil,
		struct {
			Subnet CreateRequest `json:"subnet"`
		}{Subnet: request},
		&result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &result.Subnet, nil
}

func Get(ctx context.Context, client *vpc.Client, subnetID string) (*Subnet, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodGet, resourcePath(subnetID), nil, nil, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result.Subnet, nil
}

func Update(
	ctx context.Context,
	client *vpc.Client,
	subnetID string,
	request UpdateRequest,
) (*Subnet, error) {
	var result envelope
	err := client.Request(
		ctx, http.MethodPut, resourcePath(subnetID), nil,
		struct {
			Subnet UpdateRequest `json:"subnet"`
		}{Subnet: request},
		&result,
		vpc.RequestOptions{
			ExpectedStatus: []int{http.StatusOK},
			ReadBlocked:    blockedReader(client, subnetID),
		},
	)
	if err != nil {
		return nil, err
	}
	return &result.Subnet, nil
}

func Delete(ctx context.Context, client *vpc.Client, subnetID string) error {
	return client.Request(
		ctx, http.MethodDelete, resourcePath(subnetID), nil, nil, nil,
		vpc.RequestOptions{
			ExpectedStatus: []int{http.StatusNoContent},
			ReadBlocked:    blockedReader(client, subnetID),
		},
	)
}

func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]Subnet, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[Subnet], error) {
		var result listEnvelope
		err := client.Request(
			ctx, http.MethodGet, collectionPath, query, nil, &result,
			vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[Subnet]{
			Items:    result.Subnets,
			NextLink: nextLink(result.Links),
		}, err
	})
}

type Tags struct {
	operations vpc.TagOperations
}

func TagOperations(client *vpc.Client, subnetID string) Tags {
	return Tags{operations: vpc.NewTagOperations(
		client, "subnets", subnetID, blockedReader(client, subnetID),
	)}
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
func (tags Tags) Replace(ctx context.Context, values []string) error {
	return tags.operations.Replace(ctx, values)
}
func (tags Tags) DeleteAll(ctx context.Context) error {
	return tags.operations.DeleteAll(ctx)
}

func blockedReader(client *vpc.Client, subnetID string) func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		subnet, err := Get(ctx, client, subnetID)
		if err != nil {
			return false, err
		}
		return subnet.Blocked, nil
	}
}

func resourcePath(subnetID string) string {
	return collectionPath + "/" + url.PathEscape(subnetID)
}

func nextLink(links []link) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}
	return ""
}
