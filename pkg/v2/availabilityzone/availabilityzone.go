// Package availabilityzone provides read access to Neutron availability zones.
//
// The collection is read-only: zones come from the deployment's agents, not from
// an API call. Note that it reports the zones that EXIST, which is not the same as
// the zones a caller may ask for — the Selectel fork restricts availability zone
// hints of a router to the allowed_router_zones configuration, and that allowlist
// is not exposed by any endpoint.
package availabilityzone

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/availability_zones"

// Resource values an availability zone can serve.
const (
	ResourceNetwork = "network"
	ResourceRouter  = "router"
)

// State values an availability zone can be in.
const (
	StateAvailable   = "available"
	StateUnavailable = "unavailable"
)

// AvailabilityZone is one zone as it serves one kind of resource: the same name
// appears once per resource it serves.
type AvailabilityZone struct {
	Name     string `json:"name"`
	Resource string `json:"resource"`
	State    string `json:"state"`
}

type listEnvelope struct {
	AvailabilityZones []AvailabilityZone `json:"availability_zones"`
	Links             []link             `json:"availability_zones_links"`
}

type link struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// List returns the availability zones of the region the client is scoped to.
//
// The name, resource and state filters are supported by the API through
// vpc.ListOptions.
func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]AvailabilityZone, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[AvailabilityZone], error) {
		var result listEnvelope
		err := api.Request(ctx, client, http.MethodGet, collectionPath, query, nil, &result,
			api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)

		return vpc.Page[AvailabilityZone]{
			Items: result.AvailabilityZones, NextLink: nextLink(result.Links),
		}, err
	})
}

func nextLink(links []link) string {
	for _, link := range links {
		if link.Rel == "next" {
			return link.Href
		}
	}

	return ""
}
