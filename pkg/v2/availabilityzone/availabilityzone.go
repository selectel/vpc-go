package availabilityzone

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

const collectionPath = "/v2.0/availability_zones"

const (
	ResourceNetwork = "network"
	ResourceRouter  = "router"
)

const (
	StateAvailable   = "available"
	StateUnavailable = "unavailable"
)

type AvailabilityZone struct {
	Name     string `json:"name"`
	Resource string `json:"resource"`
	State    string `json:"state"`
}

type listEnvelope struct {
	AvailabilityZones []AvailabilityZone `json:"availability_zones"`
	Links             []api.PageLink     `json:"availability_zones_links"`
}

func List(
	ctx context.Context,
	client *vpc.Client,
	options vpc.ListOptions,
) ([]AvailabilityZone, error) {
	return api.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (api.Page[AvailabilityZone], error) {
		var result listEnvelope
		err := api.Request(ctx, client, http.MethodGet, collectionPath, query, nil, &result,
			http.StatusOK,
		)

		return api.Page[AvailabilityZone]{
			Items: result.AvailabilityZones, NextLink: api.NextPageLink(result.Links),
		}, err
	})
}
