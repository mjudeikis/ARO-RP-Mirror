package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (i *Installer) exposeEtcd(ctx context.Context) error {
	infraID := i.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro"
	}

	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	lbName := infraID + "-internal-lb"

	r, err := azure.ParseResourceID(i.doc.OpenShiftCluster.ID)
	if err != nil {
		return err
	}

	lb, err := i.loadbalancers.Get(ctx, resourceGroup, lbName, "")
	if err != nil {
		return err
	}

	feipc := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/internal-lb-ip-v4",
		r.SubscriptionID,
		resourceGroup,
		infraID+"-internal-lb",
	)

	beap := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s",
		r.SubscriptionID,
		resourceGroup,
		infraID+"-internal-lb",
		infraID+"-internal-controlplane-v4",
	)

	rule := mgmtnetwork.LoadBalancingRule{
		Name: to.StringPtr("internal-etcd-v4"),
		Type: to.StringPtr("Microsoft.Network/loadBalancers/loadBalancingRules"),
		LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
			FrontendIPConfiguration: &mgmtnetwork.SubResource{
				ID: to.StringPtr(feipc),
			},
			BackendAddressPool: &mgmtnetwork.SubResource{
				ID: to.StringPtr(beap),
			},
			FrontendPort: to.Int32Ptr(2379),
			BackendPort:  to.Int32Ptr(2379),
			Protocol:     mgmtnetwork.TransportProtocolTCP,
		},
	}

	var exist = false

	for _, rule := range *lb.LoadBalancingRules {
		if *rule.Name == "internal-etcd-v4" {
			exist = true
		}
	}

	if !exist {
		*lb.LoadBalancingRules = append(*lb.LoadBalancingRules, rule)
	} else {
		i.log.Info("etcd already exposed")
	}

	return i.loadbalancers.CreateOrUpdateAndWait(ctx, resourceGroup, lbName, lb)
}
