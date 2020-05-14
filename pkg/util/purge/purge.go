package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// all the purge functions are located here

import (
	"context"
	"os"
	"sort"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

type checkFn func(mgmtfeatures.ResourceGroup, *logrus.Entry) bool

// ResourceCleaner hold the context required for cleaning
type ResourceCleaner struct {
	log    *logrus.Entry
	dryRun bool

	resourceRegions []string

	resourcegroupscli      features.ResourceGroupsClient
	vnetscli               network.VirtualNetworksClient
	privatelinkservicescli network.PrivateLinkServicesClient
	securitygroupscli      network.SecurityGroupsClient
	roleassignmentcli      authorization.RoleAssignmentsClient
	applicationscli        graphrbac.ApplicationsClient
	serviceprincipalcli    graphrbac.ServicePrincipalsClient

	subnetManager subnet.Manager

	shouldDelete checkFn
}

// NewResourceCleaner instantiates the new RC object
func NewResourceCleaner(log *logrus.Entry, shouldDelete checkFn, dryRun bool) (*ResourceCleaner, error) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	graphAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return &ResourceCleaner{
		log:    log,
		dryRun: dryRun,

		resourceRegions: []string{"v4-eastus", "v4-westeurope", "v4-australiasoutheast"},

		resourcegroupscli:      features.NewResourceGroupsClient(subscriptionID, authorizer),
		vnetscli:               network.NewVirtualNetworksClient(subscriptionID, authorizer),
		privatelinkservicescli: network.NewPrivateLinkServicesClient(subscriptionID, authorizer),
		securitygroupscli:      network.NewSecurityGroupsClient(subscriptionID, authorizer),
		roleassignmentcli:      authorization.NewRoleAssignmentsClient(subscriptionID, authorizer),
		applicationscli:        graphrbac.NewApplicationsClient(tenantID, graphAuthorizer),
		serviceprincipalcli:    graphrbac.NewServicePrincipalsClient(tenantID, graphAuthorizer),

		subnetManager: subnet.NewManager(subscriptionID, authorizer),

		// DeleteCheck decides whether the resource group gets deleted
		shouldDelete: shouldDelete,
	}, nil
}

// CleanResourceGroups loop through the resourgroups in the subscription
// and deleted everything that is not marked for deletion
// The deletion check is performed by passed function: DeleteCheck
func (rc *ResourceCleaner) CleanResourceGroups(ctx context.Context) error {
	// every resource have to live in the group, therefore deletion clean the unused groups at first
	gs, err := rc.resourcegroupscli.List(ctx, "", nil)
	if err != nil {
		return err
	}

	sort.Slice(gs, func(i, j int) bool { return *gs[i].Name < *gs[j].Name })
	for _, g := range gs {
		err := rc.cleanResourceGroup(ctx, g)
		if err != nil {
			return err
		}

	}

	return nil
}

// cleanResourceGroup checkes whether the resource group can be deleted if yes proceed to clean the group in an order:
//     - unassign subnets
//     - clean private links
//     - checks ARO presence -> store app object ID for futher use
//     - deletes resource group
func (rc *ResourceCleaner) cleanResourceGroup(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	if rc.shouldDelete(resourceGroup, rc.log) {
		rc.log.Printf("Deleting ResourceGroup: %s", *resourceGroup.Name)
		err := rc.cleanNetworking(ctx, resourceGroup)
		if err != nil {
			return err
		}

		err = rc.cleanPrivateLink(ctx, resourceGroup)
		if err != nil {
			return err
		}

		if !rc.dryRun {
			_, err := rc.resourcegroupscli.Delete(ctx, *resourceGroup.Name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// cleanNetworking lists subnets in vnets and unnassign security groups
func (rc *ResourceCleaner) cleanNetworking(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	secGroups, err := rc.securitygroupscli.List(ctx, *resourceGroup.Name)
	if err != nil {
		return err
	}

	for _, secGroup := range secGroups {

		if secGroup.SecurityGroupPropertiesFormat == nil || secGroup.SecurityGroupPropertiesFormat.Subnets == nil {
			continue
		}

		for _, secGroupSubnet := range *secGroup.SecurityGroupPropertiesFormat.Subnets {
			subnet, err := rc.subnetManager.Get(ctx, *secGroupSubnet.ID)
			if err != nil {
				return err
			}

			rc.log.Infof("Removing security group from subnet: %s/%s/%s", *resourceGroup.Name, *secGroup.Name, *subnet.Name)

			if !rc.dryRun {
				if subnet.NetworkSecurityGroup == nil {
					continue
				}

				subnet.NetworkSecurityGroup = nil

				err = rc.subnetManager.CreateOrUpdate(ctx, *subnet.ID, subnet)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

// cleanPrivateLink lists and unassigns all private links. If they are assigned the deletoin will fail
func (rc *ResourceCleaner) cleanPrivateLink(ctx context.Context, resourceGroup mgmtfeatures.ResourceGroup) error {
	plss, err := rc.privatelinkservicescli.List(ctx, *resourceGroup.Name)
	if err != nil {
		return err
	}
	for _, pls := range plss {
		if pls.PrivateEndpointConnections == nil {
			continue
		}

		for _, peconn := range *pls.PrivateEndpointConnections {
			rc.log.Infof("deleting private endpoint connection %s/%s/%s", *resourceGroup.Name, *pls.Name, *peconn.Name)
			if !rc.dryRun {
				_, err := rc.privatelinkservicescli.DeletePrivateEndpointConnection(ctx, *resourceGroup.Name, *pls.Name, *peconn.Name)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
