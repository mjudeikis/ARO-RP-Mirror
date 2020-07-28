package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/go-autorest/autorest/to"

	. "github.com/onsi/ginkgo"

	mgmtredhatopenshift "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	machineapiclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/insights"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
)

type clientSet struct {
	OpenshiftClusters redhatopenshift.OpenShiftClustersClient
	Operations        redhatopenshift.OperationsClient
	VirtualMachines   compute.VirtualMachinesClient
	Resources         features.ResourcesClient
	ActivityLogs      insights.ActivityLogsClient

	Kubernetes kubernetes.Interface
	MachineAPI machineapiclient.Interface
}

var (
	log     *logrus.Entry
	clients *clientSet
)

func skipIfNotInDevelopmentEnv() {
	if os.Getenv("RP_MODE") != "development" {
		Skip("skipping tests in non-development environment")
	}
}

func resourceIDFromEnv() string {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroup := os.Getenv("RESOURCEGROUP")
	clusterName := os.Getenv("CLUSTER")
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscriptionID, resourceGroup, clusterName)
}

func setAzureClients(subscriptionID string) error {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	clients.OpenshiftClusters = redhatopenshift.NewOpenShiftClustersClient(subscriptionID, authorizer)
	clients.Operations = redhatopenshift.NewOperationsClient(subscriptionID, authorizer)
	clients.VirtualMachines = compute.NewVirtualMachinesClient(subscriptionID, authorizer)
	clients.Resources = features.NewResourcesClient(subscriptionID, authorizer)
	clients.ActivityLogs = insights.NewActivityLogsClient(subscriptionID, authorizer)
	return nil
}

func setKuberentesClients() error {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return err
	}

	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	machineapicli, err := machineapiclient.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	clients.Kubernetes = cli
	clients.MachineAPI = machineapicli
	return nil
}

func createCluster() error {
	parameters := mgmtredhatopenshift.OpenShiftCluster{
		Location: to.StringPtr(os.Getenv("LOCATION")),
		Name:     to.StringPtr(os.Getenv("CLUSTER")),
		OpenShiftClusterProperties: &mgmtredhatopenshift.OpenShiftClusterProperties{
			MasterProfile: &mgmtredhatopenshift.MasterProfile{
				SubnetID: to.StringPtr(os.Getenv("CLUSTER") + "-master"),
			},
			WorkerProfiles: &[]mgmtredhatopenshift.WorkerProfile{
				mgmtredhatopenshift.WorkerProfile{
					SubnetID: to.StringPtr(os.Getenv("CLUSTER") + "-worker"),
				},
			},
			ServicePrincipalProfile: &mgmtredhatopenshift.ServicePrincipalProfile{
				ClientID:     to.StringPtr(os.Getenv("CLUSTER_SPN_ID")),
				ClientSecret: to.StringPtr(os.Getenv("CLUSTER_SPN_SECRET")),
			},
		},
	}

	err := clients.OpenshiftClusters.CreateOrUpdateAndWait(context.Background(), os.Getenv("ARO_RESOURCEGROUP"), os.Getenv("CLUSTER"), parameters)
	if err != nil {
		return err
	}

	return nil
}

var _ = BeforeSuite(func() {
	log.Info("BeforeSuite")
	for _, key := range []string{
		"AZURE_SUBSCRIPTION_ID",
		"CLUSTER",
		"RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			panic(fmt.Sprintf("environment variable %q unset", key))
		}
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	err := setAzureClients(subscriptionID)
	if err != nil {
		panic(err)
	}

	err = createCluster()
	if err != nil {
		panic(err)
	}

	err = setKuberentesClients()
	if err != nil {
		panic(err)
	}
})
