package router

import (
	"context"
	"net/http"
	"net/url"

	"github.com/selectel/vpc-go/internal/api"

	vpc "github.com/selectel/vpc-go/pkg/v2"
)

// InterfaceRequest identifies a router interface by either subnet or port.
type InterfaceRequest struct {
	SubnetID string `json:"subnet_id,omitempty"`
	PortID   string `json:"port_id,omitempty"`
}

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
	request InterfaceRequest,
) (*InterfaceResult, error) {
	return interfaceRequest(ctx, client, routerID, "add_router_interface", request)
}

func RemoveInterface(
	ctx context.Context,
	client *vpc.Client,
	routerID string,
	request InterfaceRequest,
) (*InterfaceResult, error) {
	return interfaceRequest(ctx, client, routerID, "remove_router_interface", request)
}

func interfaceRequest(
	ctx context.Context,
	client *vpc.Client,
	routerID string,
	action string,
	request InterfaceRequest,
) (*InterfaceResult, error) {
	var result InterfaceResult
	err := api.Request(ctx, client,
		http.MethodPut,
		collectionPath+"/"+url.PathEscape(routerID)+"/"+action,
		nil,
		request,
		&result,
		http.StatusOK,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
