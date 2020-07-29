package network

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// RouteTablesClientAddons contains addons for RouteTablesClient
type RouteTablesClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, routeTableName string, parameters mgmtnetwork.RouteTable) (err error)
}

func (c *routeTablesClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, routeTableName string, parameters mgmtnetwork.RouteTable) (err error) {
	future, err := c.RouteTablesClient.CreateOrUpdate(ctx, resourceGroupName, routeTableName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}
