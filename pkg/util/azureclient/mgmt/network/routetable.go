package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
)

// RouteTablesClient is a minimal interface for azure RouteTablesClient
type RouteTablesClient interface {
	RouteTablesClientAddons
}

type routeTablesClient struct {
	mgmtnetwork.RouteTablesClient
}

var _ RouteTablesClient = &routeTablesClient{}

// NewVirtualNetworksClient creates a new VirtualNetworksClient
func NewRouteTablesClient(subscriptionID string, authorizer autorest.Authorizer) RouteTablesClient {
	client := mgmtnetwork.NewRouteTablesClient(subscriptionID)
	client.Authorizer = authorizer

	return &routeTablesClient{
		RouteTablesClient: client,
	}
}
