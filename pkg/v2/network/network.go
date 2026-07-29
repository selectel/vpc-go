package network

import (
	"context"
	"net/http"
	"net/url"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/networks"

// Network is the observable network representation returned by Neutron.
type Network struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Status                string   `json:"status"`
	AdminStateUp          bool     `json:"admin_state_up"`
	Subnets               []string `json:"subnets"`
	ProjectID             string   `json:"project_id"`
	Shared                bool     `json:"shared"`
	RouterExternal        bool     `json:"router:external"`
	MTU                   int      `json:"mtu"`
	PortSecurityEnabled   bool     `json:"port_security_enabled"`
	AvailabilityZones     []string `json:"availability_zones"`
	AvailabilityZoneHints []string `json:"availability_zone_hints"`
	IPv4AddressScope      *string  `json:"ipv4_address_scope"`
	IPv6AddressScope      *string  `json:"ipv6_address_scope"`
	QoSPolicyID           *string  `json:"qos_policy_id"`
	ProviderNetworkType   string   `json:"provider:network_type"`
	ProviderPhysicalNet   *string  `json:"provider:physical_network"`
	ProviderSegmentation  *int     `json:"provider:segmentation_id"`
	RevisionNumber        int      `json:"revision_number"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
	Tags                  []string `json:"tags"`
	Blocked               bool     `json:"blocked"`
	IsPublic              bool     `json:"is_public"`
	IsDNSEnabled          bool     `json:"is_dns_enabled"`
}

// CreateRequest contains caller-writable network attributes.
type CreateRequest struct {
	Name                  *string   `json:"name,omitempty"`
	Description           *string   `json:"description,omitempty"`
	AdminStateUp          *bool     `json:"admin_state_up,omitempty"`
	AvailabilityZoneHints *[]string `json:"availability_zone_hints,omitempty"`
	ProjectID             *string   `json:"project_id,omitempty"`
}

// UpdateRequest contains caller-writable network attributes.
type UpdateRequest struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	AdminStateUp *bool   `json:"admin_state_up,omitempty"`
}

type networkEnvelope struct {
	Network Network `json:"network"`
}

type listEnvelope struct {
	Networks []Network `json:"networks"`
	Links    []link    `json:"networks_links"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// Create creates one network with one HTTP request.
func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*Network, error) {
	var envelope networkEnvelope
	err := client.Request(
		ctx,
		http.MethodPost,
		collectionPath,
		nil,
		struct {
			Network CreateRequest `json:"network"`
		}{Network: request},
		&envelope,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &envelope.Network, nil
}

// Get reads one network.
func Get(ctx context.Context, client *vpc.Client, networkID string) (*Network, error) {
	var envelope networkEnvelope
	err := client.Request(
		ctx,
		http.MethodGet,
		resourcePath(networkID),
		nil,
		nil,
		&envelope,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &envelope.Network, nil
}

// Update updates one network with one HTTP request.
func Update(
	ctx context.Context,
	client *vpc.Client,
	networkID string,
	request UpdateRequest,
) (*Network, error) {
	var envelope networkEnvelope
	err := client.Request(
		ctx,
		http.MethodPut,
		resourcePath(networkID),
		nil,
		struct {
			Network UpdateRequest `json:"network"`
		}{Network: request},
		&envelope,
		vpc.RequestOptions{
			ExpectedStatus: []int{http.StatusOK},
			ReadBlocked:    blockedReader(client, networkID),
		},
	)
	if err != nil {
		return nil, err
	}
	return &envelope.Network, nil
}

// Delete deletes one network without deleting its ports.
func Delete(ctx context.Context, client *vpc.Client, networkID string) error {
	return client.Request(
		ctx,
		http.MethodDelete,
		resourcePath(networkID),
		nil,
		nil,
		nil,
		vpc.RequestOptions{
			ExpectedStatus: []int{http.StatusNoContent},
			ReadBlocked:    blockedReader(client, networkID),
		},
	)
}

// List reads the complete network collection.
func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]Network, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[Network], error) {
		var envelope listEnvelope
		err := client.Request(
			ctx,
			http.MethodGet,
			collectionPath,
			query,
			nil,
			&envelope,
			vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[Network]{
			Items:    envelope.Networks,
			NextLink: nextLink(envelope.Links),
		}, err
	})
}

// Tags provides typed tag operations for one network.
type Tags struct {
	operations vpc.TagOperations
}

// TagOperations returns tag operations bound to networkID.
func TagOperations(client *vpc.Client, networkID string) Tags {
	return Tags{operations: vpc.NewTagOperations(
		client,
		"networks",
		networkID,
		blockedReader(client, networkID),
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

func blockedReader(client *vpc.Client, networkID string) func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		network, err := Get(ctx, client, networkID)
		if err != nil {
			return false, err
		}
		return network.Blocked, nil
	}
}

func resourcePath(networkID string) string {
	return collectionPath + "/" + url.PathEscape(networkID)
}

func nextLink(links []link) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}
	return ""
}
