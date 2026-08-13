package floatingip

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

type PortForwarding struct {
	ID                string `json:"id"`
	Protocol          string `json:"protocol"`
	InternalPortID    string `json:"internal_port_id"`
	InternalIPAddress string `json:"internal_ip_address"`
	InternalPort      int    `json:"internal_port"`
	ExternalPort      int    `json:"external_port"`
	Description       string `json:"description"`
}

type PortForwardingCreateRequest struct {
	Protocol          *string `json:"protocol,omitempty"`
	InternalPortID    *string `json:"internal_port_id,omitempty"`
	InternalIPAddress *string `json:"internal_ip_address,omitempty"`
	InternalPort      *int    `json:"internal_port,omitempty"`
	ExternalPort      *int    `json:"external_port,omitempty"`
	Description       *string `json:"description,omitempty"`
	ProjectID         *string `json:"project_id,omitempty"`
}

type PortForwardingUpdateRequest struct {
	Protocol          *string `json:"protocol,omitempty"`
	InternalPortID    *string `json:"internal_port_id,omitempty"`
	InternalIPAddress *string `json:"internal_ip_address,omitempty"`
	InternalPort      *int    `json:"internal_port,omitempty"`
	ExternalPort      *int    `json:"external_port,omitempty"`
	Description       *string `json:"description,omitempty"`
}

type portForwardingEnvelope struct {
	PortForwarding PortForwarding `json:"port_forwarding"`
}

type portForwardingListEnvelope struct {
	PortForwardings []PortForwarding `json:"port_forwardings"`
	Links           []api.PageLink   `json:"port_forwardings_links"`
}

func CreatePortForwarding(
	ctx context.Context,
	client *vpc.Client,
	floatingIPID string,
	request PortForwardingCreateRequest,
) (*PortForwarding, error) {
	var envelope portForwardingEnvelope
	err := api.Request(ctx, client,
		http.MethodPost,
		portForwardingCollectionPath(floatingIPID),
		nil,
		struct {
			PortForwarding PortForwardingCreateRequest `json:"port_forwarding"`
		}{PortForwarding: request},
		&envelope,
		http.StatusCreated,
	)
	if err != nil {
		return nil, err
	}
	return &envelope.PortForwarding, nil
}

func GetPortForwarding(
	ctx context.Context,
	client *vpc.Client,
	floatingIPID string,
	portForwardingID string,
) (*PortForwarding, error) {
	var envelope portForwardingEnvelope
	err := api.Request(ctx, client,
		http.MethodGet,
		portForwardingResourcePath(floatingIPID, portForwardingID),
		nil,
		nil,
		&envelope,
		http.StatusOK,
	)
	if err != nil {
		return nil, err
	}
	return &envelope.PortForwarding, nil
}

func UpdatePortForwarding(
	ctx context.Context,
	client *vpc.Client,
	floatingIPID string,
	portForwardingID string,
	request PortForwardingUpdateRequest,
) (*PortForwarding, error) {
	var envelope portForwardingEnvelope
	err := api.Request(ctx, client,
		http.MethodPut,
		portForwardingResourcePath(floatingIPID, portForwardingID),
		nil,
		struct {
			PortForwarding PortForwardingUpdateRequest `json:"port_forwarding"`
		}{PortForwarding: request},
		&envelope,
		http.StatusOK,
	)
	if err != nil {
		return nil, err
	}
	return &envelope.PortForwarding, nil
}

func DeletePortForwarding(
	ctx context.Context,
	client *vpc.Client,
	floatingIPID string,
	portForwardingID string,
) error {
	return api.Request(ctx, client,
		http.MethodDelete,
		portForwardingResourcePath(floatingIPID, portForwardingID),
		nil,
		nil,
		nil,
		http.StatusNoContent,
	)
}

func ListPortForwardings(
	ctx context.Context,
	client *vpc.Client,
	floatingIPID string,
	options vpc.ListOptions,
) ([]PortForwarding, error) {
	return api.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (api.Page[PortForwarding], error) {
		var envelope portForwardingListEnvelope
		err := api.Request(ctx, client,
			http.MethodGet,
			portForwardingCollectionPath(floatingIPID),
			query,
			nil,
			&envelope,
			http.StatusOK,
		)
		return api.Page[PortForwarding]{
			Items:    envelope.PortForwardings,
			NextLink: api.NextPageLink(envelope.Links),
		}, err
	})
}

func portForwardingCollectionPath(floatingIPID string) string {
	return resourcePath(floatingIPID) + "/port_forwardings"
}

func portForwardingResourcePath(floatingIPID, portForwardingID string) string {
	return portForwardingCollectionPath(floatingIPID) + "/" + url.PathEscape(portForwardingID)
}
