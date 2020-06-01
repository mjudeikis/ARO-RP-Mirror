package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// all the purge functions are located here

import (
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

const (
	defaultTTL          = 48
	defaultCreatedAtTag = "createdAt"
	defaultKeepTag      = "persist"
	defaultDryRun       = false
)

// ResourceCleaner hold the context required for cleaning
type ResourceCleaner struct {
	log                 *logrus.Entry
	subscriptionID      string
	tenantID            string
	dryRun              bool
	ttl                 time.Duration
	resourceRegions     []string
	deleteGroupPrefixes []string

	resourcegroupscli      features.ResourceGroupsClient
	vnetscli               network.VirtualNetworksClient
	privatelinkservicescli network.PrivateLinkServicesClient
	securitygroupscli      network.SecurityGroupsClient
	roleassignmentcli      authorization.RoleAssignmentsClient
	applicationscli        graphrbac.ApplicationsClient
	serviceprincipalcli    graphrbac.ServicePrincipalsClient

	subnetManager subnet.Manager
}

// NewResourceCleaner instantiates the new RC object
func NewResourceCleaner(log *logrus.Entry, dryRun bool) (*ResourceCleaner, error) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	var ttl time.Duration
	if os.Getenv("AZURE_PURGE_TTL") != "" {
		var err error
		ttl, err = time.ParseDuration(os.Getenv("AZURE_PURGE_TTL"))
		if err != nil {
			return nil, err
		}
	} else {
		ttl = defaultTTL
	}

	var deleteGroupPrefixes []string
	if os.Getenv("AZURE_PURGE_RG_DEL_PREFIX") != "" {
		deleteGroupPrefixes = strings.Split(os.Getenv("AZURE_PURGE_RG_DEL_PREFIX"), ",")
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	graphAuthorizer, err := auth.NewAuthorizerFromEnvironmentWithResource(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return &ResourceCleaner{
		log:            log,
		subscriptionID: subscriptionID,
		tenantID:       tenantID,
		ttl:            ttl,
		dryRun:         dryRun,

		resourceRegions:     []string{"v4-eastus", "v4-westeurope", "v4-australiasoutheast"},
		deleteGroupPrefixes: deleteGroupPrefixes,

		resourcegroupscli:      features.NewResourceGroupsClient(subscriptionID, authorizer),
		vnetscli:               network.NewVirtualNetworksClient(subscriptionID, authorizer),
		privatelinkservicescli: network.NewPrivateLinkServicesClient(subscriptionID, authorizer),
		securitygroupscli:      network.NewSecurityGroupsClient(subscriptionID, authorizer),
		roleassignmentcli:      authorization.NewRoleAssignmentsClient(subscriptionID, authorizer),
		applicationscli:        graphrbac.NewApplicationsClient(tenantID, graphAuthorizer),
		serviceprincipalcli:    graphrbac.NewServicePrincipalsClient(tenantID, graphAuthorizer),

		subnetManager: subnet.NewManager(subscriptionID, authorizer),
	}, nil
}
