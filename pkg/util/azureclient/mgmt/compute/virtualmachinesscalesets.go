package compute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest"
)

type VirtualMachineScaleSetsClient interface {
	VirtualMachineScaleSetsClientAddons
	Get(ctx context.Context, resourceGroupName string, VMScaleSetName string) (result mgmtcompute.VirtualMachineScaleSet, err error)
}

type virtualMachineScaleSetsClient struct {
	mgmtcompute.VirtualMachineScaleSetsClient
}

var _ VirtualMachineScaleSetsClient = &virtualMachineScaleSetsClient{}

// NewVirtualMachineScaleSetsClient creates a new VirtualMachineScaleSetsClient
func NewVirtualMachineScaleSetsClient(subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetsClient {
	client := mgmtcompute.NewVirtualMachineScaleSetsClient(subscriptionID)
	client.Authorizer = authorizer

	return &virtualMachineScaleSetsClient{
		VirtualMachineScaleSetsClient: client,
	}
}
