package floatingip

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

// PortForwarding is a port-forwarding rule belonging to one floating IP.
type PortForwarding struct {
	ID                string `json:"id"`
	Protocol          string `json:"protocol"`
	InternalPortID    string `json:"internal_port_id"`
	InternalIPAddress string `json:"internal_ip_address"`
	InternalPort      int    `json:"internal_port"`
	ExternalPort      int    `json:"external_port"`
	Description       string `json:"description"`
}

// PortForwardingCreateRequest contains caller-writable attributes for a new rule.
type PortForwardingCreateRequest struct {
	Protocol          *string `json:"protocol,omitempty"`
	InternalPortID    *string `json:"internal_port_id,omitempty"`
	InternalIPAddress *string `json:"internal_ip_address,omitempty"`
	InternalPort      *int    `json:"internal_port,omitempty"`
	ExternalPort      *int    `json:"external_port,omitempty"`
	Description       *string `json:"description,omitempty"`
	ProjectID         *string `json:"project_id,omitempty"`
}

// PortForwardingUpdateRequest contains caller-writable attributes for a rule.
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
	Links           []link           `json:"port_forwardings_links"`
}

// CreatePortForwarding creates one rule under floatingIPID.
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
		api.RequestOptions{ExpectedStatus: []int{http.StatusCreated}},
	)
	if err != nil {
		return nil, err
	}
	return &envelope.PortForwarding, nil
}

// GetPortForwarding reads one rule under floatingIPID.
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
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &envelope.PortForwarding, nil
}

// UpdatePortForwarding updates one rule under floatingIPID.
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
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &envelope.PortForwarding, nil
}

// DeletePortForwarding deletes one rule under floatingIPID.
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
		api.RequestOptions{ExpectedStatus: []int{http.StatusNoContent}},
	)
}

// ListPortForwardings reads the complete rule collection under floatingIPID.
func ListPortForwardings(
	ctx context.Context,
	client *vpc.Client,
	floatingIPID string,
	options vpc.ListOptions,
) ([]PortForwarding, error) {
	return vpc.WalkPages(ctx, options.Values(), func(
		ctx context.Context,
		query url.Values,
	) (vpc.Page[PortForwarding], error) {
		var envelope portForwardingListEnvelope
		err := api.Request(ctx, client,
			http.MethodGet,
			portForwardingCollectionPath(floatingIPID),
			query,
			nil,
			&envelope,
			api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
		)
		return vpc.Page[PortForwarding]{
			Items:    envelope.PortForwardings,
			NextLink: nextLink(envelope.Links),
		}, err
	})
}

func portForwardingCollectionPath(floatingIPID string) string {
	return resourcePath(floatingIPID) + "/port_forwardings"
}

func portForwardingResourcePath(floatingIPID, portForwardingID string) string {
	return portForwardingCollectionPath(floatingIPID) + "/" + url.PathEscape(portForwardingID)
}
