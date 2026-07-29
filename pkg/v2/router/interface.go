package router

import (
	"context"
	"net/http"
	"net/url"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

// InterfaceSelector can only be created with BySubnet or ByPort.
type InterfaceSelector interface {
	payload() interfacePayload
}

type interfacePayload struct {
	SubnetID string `json:"subnet_id,omitempty"`
	PortID   string `json:"port_id,omitempty"`
}

type subnetSelector struct{ id string }

func (selector subnetSelector) payload() interfacePayload {
	return interfacePayload{SubnetID: selector.id}
}

type portSelector struct{ id string }

func (selector portSelector) payload() interfacePayload { return interfacePayload{PortID: selector.id} }

// BySubnet addresses a router interface by subnet.
func BySubnet(subnetID string) InterfaceSelector { return subnetSelector{id: subnetID} }

// ByPort addresses a router interface by port.
func ByPort(portID string) InterfaceSelector { return portSelector{id: portID} }

type InterfaceResult struct {
	RouterID  string `json:"id"`
	PortID    string `json:"port_id"`
	SubnetID  string `json:"subnet_id"`
	NetworkID string `json:"network_id"`
}

func AddInterface(
	ctx context.Context,
	client *vpc.Client,
	routerID string,
	selector InterfaceSelector,
) (*InterfaceResult, error) {
	return interfaceRequest(ctx, client, routerID, "add_router_interface", selector)
}

func RemoveInterface(
	ctx context.Context,
	client *vpc.Client,
	routerID string,
	selector InterfaceSelector,
) (*InterfaceResult, error) {
	return interfaceRequest(ctx, client, routerID, "remove_router_interface", selector)
}

func interfaceRequest(
	ctx context.Context,
	client *vpc.Client,
	routerID string,
	action string,
	selector InterfaceSelector,
) (*InterfaceResult, error) {
	var result InterfaceResult
	err := client.Request(
		ctx,
		http.MethodPut,
		collectionPath+"/"+url.PathEscape(routerID)+"/"+action,
		nil,
		selector.payload(),
		&result,
		vpc.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
