// Package availabilityzone provides read access to VPC availability zones.
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
	Links             []api.PageLink     `json:"availability_zones_links"`
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
			Items: result.AvailabilityZones, NextLink: api.NextPageLink(result.Links),
		}, err
	})
}
