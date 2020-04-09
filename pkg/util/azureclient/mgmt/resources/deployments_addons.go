package resources

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"k8s.io/apimachinery/pkg/util/wait"
)

// DeploymentsClientAddons contains addons for DeploymentsClient
type DeploymentsClientAddons interface {
	CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, deploymentName string, parameters mgmtresources.Deployment) error
	CreateOrUpdateAtSubscriptionScopeAndWait(ctx context.Context, deploymentName string, parameters mgmtresources.Deployment) error
	Wait(ctx context.Context, resourceGroupName string, deploymentName string) error
	DeleteAndWait(ctx context.Context, resourceGroupName string, deploymentName string) error
}

func (c *deploymentsClient) DeleteAndWait(ctx context.Context, resourceGroupName string, deploymentName string) error {
	future, err := c.Delete(ctx, resourceGroupName, deploymentName)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *deploymentsClient) CreateOrUpdateAtSubscriptionScopeAndWait(ctx context.Context, deploymentName string, parameters mgmtresources.Deployment) error {
	future, err := c.CreateOrUpdateAtSubscriptionScope(ctx, deploymentName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *deploymentsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, deploymentName string, parameters mgmtresources.Deployment) error {
	future, err := c.CreateOrUpdate(ctx, resourceGroupName, deploymentName, parameters)
	if err != nil {
		return err
	}

	return future.WaitForCompletionRef(ctx, c.Client)
}

func (c *deploymentsClient) Wait(ctx context.Context, resourceGroupName string, deploymentName string) error {
	return wait.Poll(c.Client.PollingDelay, c.Client.PollingDuration, func() (bool, error) {
		deployment, err := c.Get(ctx, resourceGroupName, deploymentName)
		if err != nil {
			return false, err
		}

		switch *deployment.Properties.ProvisioningState {
		case "Canceled", "Failed":
			return false, fmt.Errorf("got provisioningState %q", *deployment.Properties.ProvisioningState)
		}

		return *deployment.Properties.ProvisioningState == "Succeeded", nil
	})
}
