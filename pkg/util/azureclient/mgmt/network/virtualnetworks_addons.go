package network

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
)

// VirtualNetworksClientAddons contains addons for VirtualNetworksClient
type VirtualNetworksClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters mgmtnetwork.VirtualNetwork) (err error)
	List(ctx context.Context, resourceGroupName string) (virtualnetworks []mgmtnetwork.VirtualNetwork, err error)
}

func (c *virtualNetworksClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, virtualNetworkName string, parameters mgmtnetwork.VirtualNetwork) (err error) {
	future, err := c.VirtualNetworksClient.CreateOrUpdate(ctx, resourceGroupName, virtualNetworkName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *virtualNetworksClient) List(ctx context.Context, resourceGroupName string) (virtualnetworks []mgmtnetwork.VirtualNetwork, err error) {
	page, err := c.VirtualNetworksClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		virtualnetworks = append(virtualnetworks, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return virtualnetworks, nil
}
