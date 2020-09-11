package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	mgmtprivatedns "github.com/Azure/azure-sdk-for-go/services/privatedns/mgmt/2018-09-01/privatedns"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/ignition/machine"
	"github.com/openshift/installer/pkg/asset/installconfig"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (i *manager) deployResourceTemplate(ctx context.Context) error {
	g, err := i.loadGraph(ctx)
	if err != nil {
		return err
	}

	installConfig := g[reflect.TypeOf(&installconfig.InstallConfig{})].(*installconfig.InstallConfig)
	machineMaster := g[reflect.TypeOf(&machine.Master{})].(*machine.Master)

	infraID := i.doc.OpenShiftCluster.Properties.InfraID
	if infraID == "" {
		infraID = "aro" // TODO: remove after deploy
	}

	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	vnetID, _, err := subnet.Split(i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	srvRecords := make([]mgmtprivatedns.SrvRecord, *installConfig.Config.ControlPlane.Replicas)
	for i := 0; i < int(*installConfig.Config.ControlPlane.Replicas); i++ {
		srvRecords[i] = mgmtprivatedns.SrvRecord{
			Priority: to.Int32Ptr(10),
			Weight:   to.Int32Ptr(10),
			Port:     to.Int32Ptr(2380),
			Target:   to.StringPtr(fmt.Sprintf("etcd-%d.%s", i, installConfig.Config.ObjectMeta.Name+"."+installConfig.Config.BaseDomain)),
		}
	}

	zones, err := zones(installConfig)
	if err != nil {
		return err
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters: map[string]*arm.TemplateParameter{
			"sas": {
				Type: "object",
			},
		},
		Resources: []*arm.Resource{
			dnsPrivateZone(installConfig),                                                                     // upstream
			dnsPrivateRecordAPIINT(infraID, installConfig),                                                    // upstream
			dnsPrivateRecordAPI(infraID, installConfig),                                                       // upstream
			dnsVirtualNetworkLink(vnetID, installConfig),                                                      // aro
			networkPrivateLinkService(infraID, i.env.SubscriptionID(), i.doc.OpenShiftCluster, installConfig), // aro
			networkPublicIPAddress(infraID, installConfig),                                                    // upstream
			networkPublicIPAddressOutbound(infraID, installConfig),                                            // aro
			networkInternalLoadBalancer(infraID, i.doc.OpenShiftCluster, installConfig),                       // upstream
			networkPublicLoadBalancer(infraID, i.doc.OpenShiftCluster, installConfig),                         // upstream
			networkBootstrapNIC(infraID, i.doc.OpenShiftCluster, installConfig),                               // upstream
			networkMasterNICs(infraID, i.doc.OpenShiftCluster, installConfig),                                 // upstream
			computeBoostrapVM(infraID, i.doc.OpenShiftCluster, installConfig),                                 // upstream
			computeMasterVMs(infraID, zones, machineMaster, i.doc.OpenShiftCluster, installConfig),            // upstream
		},
	}
	return i.deployARMTemplate(ctx, resourceGroup, "resources", t, map[string]interface{}{
		"sas": map[string]interface{}{
			"value": map[string]interface{}{
				"signedStart":         i.doc.OpenShiftCluster.Properties.Install.Now.Format(time.RFC3339),
				"signedExpiry":        i.doc.OpenShiftCluster.Properties.Install.Now.Add(24 * time.Hour).Format(time.RFC3339),
				"signedPermission":    "rl",
				"signedResourceTypes": "o",
				"signedServices":      "b",
				"signedProtocol":      "https",
			},
		},
	})
}

// zones configures how master nodes are distributed across availability zones. In regions where the number of zones matches
// the number of nodes, it's one node per zone. In regions where there are no zones, all the nodes are in the same place.
// Anything else (e.g. 2-zone regions) is currently unsupported.
func zones(installConfig *installconfig.InstallConfig) (zones *[]string, err error) {
	zoneCount := len(installConfig.Config.ControlPlane.Platform.Azure.Zones)
	replicas := int(*installConfig.Config.ControlPlane.Replicas)
	if reflect.DeepEqual(installConfig.Config.ControlPlane.Platform.Azure.Zones, []string{""}) {
		// []string{""} indicates that there are no Azure Zones, so "zones" return value will be nil
	} else if zoneCount == replicas {
		zones = &[]string{"[copyIndex(1)]"}
	} else {
		err = fmt.Errorf("cluster creation with %d zone(s) and %d replica(s) is unimplemented", zoneCount, replicas)
	}
	return
}

func (i *manager) deployARMTemplate(ctx context.Context, rg string, tName string, t *arm.Template, params map[string]interface{}) error {
	i.log.Printf("deploying %s template", tName)

	err := i.deployments.CreateOrUpdateAndWait(ctx, rg, deploymentName, mgmtfeatures.Deployment{
		Properties: &mgmtfeatures.DeploymentProperties{
			Template:   t,
			Parameters: params,
			Mode:       mgmtfeatures.Incremental,
		},
	})

	if azureerrors.IsDeploymentActiveError(err) {
		i.log.Printf("waiting for %s template to be deployed", tName)
		err = i.deployments.Wait(ctx, rg, deploymentName)
	}

	if azureerrors.HasAuthorizationFailedError(err) ||
		azureerrors.HasLinkedAuthorizationFailedError(err) {
		return err
	}

	serviceErr, _ := err.(*azure.ServiceError) // futures return *azure.ServiceError directly

	// CreateOrUpdate() returns a wrapped *azure.ServiceError
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		serviceErr, _ = detailedErr.Original.(*azure.ServiceError)
	}

	if serviceErr != nil {
		b, _ := json.Marshal(serviceErr)

		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: "Deployment failed.",
				Details: []api.CloudErrorBody{
					{
						Message: string(b),
					},
				},
			},
		}
	}

	return err
}
