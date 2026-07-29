package floatingip

import (
	"context"
	"net/http"
	"net/url"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/floatingips"

type FloatingIP struct {
	ID                string   `json:"id"`
	FloatingNetworkID string   `json:"floating_network_id"`
	SubnetID          *string  `json:"subnet_id"`
	PortID            *string  `json:"port_id"`
	FixedIPAddress    *string  `json:"fixed_ip_address"`
	FloatingIPAddress string   `json:"floating_ip_address"`
	RouterID          *string  `json:"router_id"`
	Description       string   `json:"description"`
	Status            string   `json:"status"`
	QoSPolicyID       *string  `json:"qos_policy_id"`
	ProjectID         string   `json:"project_id"`
	RevisionNumber    int      `json:"revision_number"`
	CreatedAt         string   `json:"created_at"`
	UpdatedAt         string   `json:"updated_at"`
	Tags              []string `json:"tags"`
	Blocked           bool     `json:"blocked"`
	DNSName           string   `json:"dns_name,omitempty"`
	DNSDomain         string   `json:"dns_domain,omitempty"`
}

type CreateRequest struct {
	FloatingNetworkID string  `json:"floating_network_id"`
	SubnetID          *string `json:"subnet_id,omitempty"`
	PortID            *string `json:"port_id,omitempty"`
	FixedIPAddress    *string `json:"fixed_ip_address,omitempty"`
	Description       *string `json:"description,omitempty"`
	ProjectID         *string `json:"project_id,omitempty"`
	DNSName           *string `json:"dns_name,omitempty"`
	DNSDomain         *string `json:"dns_domain,omitempty"`
}

type UpdateRequest struct {
	PortID         *vpc.Optional[string] `json:"port_id,omitempty"`
	FixedIPAddress *string               `json:"fixed_ip_address,omitempty"`
	Description    *string               `json:"description,omitempty"`
}

type envelope struct {
	FloatingIP FloatingIP `json:"floatingip"`
}
type listEnvelope struct {
	FloatingIPs []FloatingIP `json:"floatingips"`
	Links       []link       `json:"floatingips_links"`
}
type link struct{ Rel, Href string }

func Create(ctx context.Context, client *vpc.Client, request CreateRequest) (*FloatingIP, error) {
	var result envelope
	err := client.Request(ctx, http.MethodPost, collectionPath, nil,
		struct {
			FloatingIP CreateRequest `json:"floatingip"`
		}{request}, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusCreated}})
	if err != nil {
		return nil, err
	}
	return &result.FloatingIP, nil
}
func Get(ctx context.Context, client *vpc.Client, id string) (*FloatingIP, error) {
	var result envelope
	err := client.Request(ctx, http.MethodGet, resourcePath(id), nil, nil, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}})
	if err != nil {
		return nil, err
	}
	return &result.FloatingIP, nil
}
func Update(ctx context.Context, client *vpc.Client, id string, request UpdateRequest) (*FloatingIP, error) {
	var result envelope
	err := client.Request(ctx, http.MethodPut, resourcePath(id), nil,
		struct {
			FloatingIP UpdateRequest `json:"floatingip"`
		}{request}, &result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}, ReadBlocked: blockedReader(client, id)})
	if err != nil {
		return nil, err
	}
	return &result.FloatingIP, nil
}
func Delete(ctx context.Context, client *vpc.Client, id string) error {
	return client.Request(ctx, http.MethodDelete, resourcePath(id), nil, nil, nil,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}, ReadBlocked: blockedReader(client, id)})
}
func List(ctx context.Context, client *vpc.Client, options vpc.ListOptions) ([]FloatingIP, error) {
	return vpc.WalkPages(ctx, options.Values(), func(ctx context.Context, q url.Values) (vpc.Page[FloatingIP], error) {
		var result listEnvelope
		err := client.Request(ctx, http.MethodGet, collectionPath, q, nil, &result, vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}})
		return vpc.Page[FloatingIP]{Items: result.FloatingIPs, NextLink: nextLink(result.Links)}, err
	})
}

type Tags struct{ operations vpc.TagOperations }

func TagOperations(client *vpc.Client, id string) Tags {
	return Tags{vpc.NewTagOperations(client, "floatingips", id, blockedReader(client, id))}
}
func (t Tags) Get(ctx context.Context) ([]string, error)         { return t.operations.Get(ctx) }
func (t Tags) Has(ctx context.Context, tag string) (bool, error) { return t.operations.Has(ctx, tag) }
func (t Tags) Add(ctx context.Context, tag string) error         { return t.operations.Add(ctx, tag) }
func (t Tags) Delete(ctx context.Context, tag string) error      { return t.operations.Delete(ctx, tag) }
func (t Tags) Replace(ctx context.Context, v []string) error     { return t.operations.Replace(ctx, v) }
func (t Tags) DeleteAll(ctx context.Context) error               { return t.operations.DeleteAll(ctx) }
func blockedReader(c *vpc.Client, id string) func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		f, e := Get(ctx, c, id)
		if e != nil {
			return false, e
		}
		return f.Blocked, nil
	}
}
func resourcePath(id string) string { return collectionPath + "/" + url.PathEscape(id) }
func nextLink(ls []link) string {
	for _, l := range ls {
		if l.Rel == "next" {
			return l.Href
		}
	}
	return ""
}
