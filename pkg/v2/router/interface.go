package router

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"

	"github.com/selectel/vpc-go/internal/api"

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
	if selector == nil || isNilSelector(selector) {
		return nil, &vpc.ClientError{Err: errors.New("router interface selector is required")}
	}
	payload := selector.payload()
	if (payload.SubnetID == "") == (payload.PortID == "") {
		return nil, &vpc.ClientError{
			Err: errors.New("router interface selector must contain exactly one non-empty ID"),
		}
	}

	var result InterfaceResult
	err := api.Request(ctx, client,
		http.MethodPut,
		collectionPath+"/"+url.PathEscape(routerID)+"/"+action,
		nil,
		payload,
		&result,
		api.RequestOptions{ExpectedStatus: []int{http.StatusOK}},
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func isNilSelector(selector InterfaceSelector) bool {
	value := reflect.ValueOf(selector)
	return value.Kind() == reflect.Pointer && value.IsNil()
}
