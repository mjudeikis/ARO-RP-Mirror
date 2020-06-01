package purge

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"
	"strings"
	"time"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"
)

func (rc *ResourceCleaner) shouldDelete(resourceGroup mgmtfeatures.ResourceGroup, log *logrus.Entry) bool {
	isDeleteGroup := false
	for _, deleteGroupPrefix := range rc.deleteGroupPrefixes {
		if strings.HasPrefix(*resourceGroup.Name, deleteGroupPrefix) {
			isDeleteGroup = true
			break
		}
	}
	if isDeleteGroup {
		return false
	}

	for t := range resourceGroup.Tags {
		if strings.ToLower(t) == defaultKeepTag {
			log.WithField("resourceGroup", *resourceGroup.Name).Debug("Group set to persist. Skip.")
			return false
		}
	}

	// azure tags is not consistent with lower/upper cases.
	if _, ok := resourceGroup.Tags[defaultCreatedAtTag]; !ok {
		log.WithField("resourceGroup", *resourceGroup.Name).Debug("Group does not have createdAt tag. Skip.")
		return false
	}

	createdAt, err := time.Parse(time.RFC3339Nano, *resourceGroup.Tags[defaultCreatedAtTag])
	if err != nil {
		log.Errorf("%s: %s", *resourceGroup.Name, err)
		return false
	}
	if time.Now().Sub(createdAt) < rc.ttl {
		log.WithField("resourceGroup", *resourceGroup.Name).Debug("Group TTL is less than purge. Skip.")
		return false
	}

	return true
}

// CleanResourceGroups loop through the resourgroups in the subscription
// and deleted everything that is not marked for deletion
// The deletion check is performed by passed function: shouldDelete
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

// cleanNetworking lists subnets in vnets and unassign security groups
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

			rc.log.Debugf("Removing security group from subnet: %s/%s/%s", *resourceGroup.Name, *secGroup.Name, *subnet.Name)

			if subnet.NetworkSecurityGroup == nil {
				continue
			}
			subnet.NetworkSecurityGroup = nil

			if !rc.dryRun {
				err = rc.subnetManager.CreateOrUpdate(ctx, *subnet.ID, subnet)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

// cleanPrivateLink lists and unassign all private links. If they are assigned the delete will fail
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
			rc.log.Debugf("deleting private endpoint connection %s/%s/%s", *resourceGroup.Name, *pls.Name, *peconn.Name)
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
