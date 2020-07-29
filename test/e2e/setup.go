package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"

	"github.com/Azure/go-autorest/autorest/to"

	. "github.com/onsi/ginkgo"

	mgmtredhatopenshift "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2020-04-30/redhatopenshift"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	machineapiclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/insights"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
)

type clientSet struct {
	OpenshiftClusters redhatopenshift.OpenShiftClustersClient
	Operations        redhatopenshift.OperationsClient
	VirtualMachines   compute.VirtualMachinesClient
	Resources         features.ResourcesClient
	ActivityLogs      insights.ActivityLogsClient
	Groups            features.ResourceGroupsClient
	Networks          network.VirtualNetworksClient
	RouteTables       network.RouteTablesClient
	Subnets           network.SubnetsClient

	Kubernetes kubernetes.Interface
	MachineAPI machineapiclient.Interface
}

type clusterConfig struct {
	// inputs - passed as env variables
	ClusterName            string
	ResourceGroupName      string
	ClusterResourceGroupID string
	Location               string
	SubscriptionID         string

	// derivatives from inputs
	Domain          string
	ResourceGroupID string
	ResourceID      string
	RouteTableName  string

	// dependencies, created by script
	MasterSubnetID                string
	WorkerSubnetID                string
	ClusterServicePrincipalID     string
	ClusterServicePrincipalSecret string
}

var (
	log     *logrus.Entry
	clients *clientSet
	config  *clusterConfig
)

func init() {
	clients = &clientSet{}
	config = &clusterConfig{}
	// inputs
	config.SubscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	config.ResourceGroupName = os.Getenv("RESOURCEGROUP")
	config.ClusterName = os.Getenv("CLUSTER")
	config.Location = os.Getenv("LOCATION")
	// derivatives
	config.ClusterResourceGroupID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", config.SubscriptionID, config.ClusterName)
	config.ResourceID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", config.SubscriptionID, config.ResourceGroupName, config.ClusterName)
	config.RouteTableName = config.ClusterName + "-rt"
}

func skipIfNotInDevelopmentEnv() {
	if os.Getenv("RP_MODE") != "development" {
		Skip("skipping tests in non-development environment")
	}
}

// setupDependencies created all dependencies for cluster tests
func setupDependencies() error {
	_, err := clients.Groups.CreateOrUpdate(context.TODO(), config.ResourceGroupName, mgmtfeatures.ResourceGroup{
		Location: to.StringPtr(config.Location),
	})
	if err != nil {
		return err
	}

	err = clients.Networks.CreateOrUpdateAndWait(context.TODO(), config.ResourceGroupName, "dev-vnet", mgmtnetwork.VirtualNetwork{
		VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &mgmtnetwork.AddressSpace{
				AddressPrefixes: &[]string{"10.0.0.0/9"},
			},
		},
	})
	if err != nil {
		return err
	}

	err = clients.RouteTables.CreateOrUpdateAndWait(context.TODO(), config.ResourceGroupName, config.RouteTableName, mgmtnetwork.RouteTable{})
	if err != nil {
		return err
	}

	return nil
}

// setAzureClients all azure clients, needed in the tests
func setupAzureClients(subscriptionID string) error {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	clients = &clientSet{}
	clients.OpenshiftClusters = redhatopenshift.NewOpenShiftClustersClient(subscriptionID, authorizer)
	clients.Operations = redhatopenshift.NewOperationsClient(subscriptionID, authorizer)
	clients.VirtualMachines = compute.NewVirtualMachinesClient(subscriptionID, authorizer)
	clients.Resources = features.NewResourcesClient(subscriptionID, authorizer)
	clients.ActivityLogs = insights.NewActivityLogsClient(subscriptionID, authorizer)
	clients.Groups = features.NewResourceGroupsClient(subscriptionID, authorizer)
	clients.Networks = network.NewVirtualNetworksClient(subscriptionID, authorizer)
	clients.RouteTables = network.NewRouteTablesClient(subscriptionID, authorizer)
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
			ClusterProfile: &mgmtredhatopenshift.ClusterProfile{
				Domain:          to.StringPtr(strings.ToLower(os.Getenv("CLUSTER"))),
				ResourceGroupID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("CLUSTER"))),
			},
			MasterProfile: &mgmtredhatopenshift.MasterProfile{
				SubnetID: to.StringPtr(os.Getenv("CLUSTER") + "-master"),
				VMSize:   mgmtredhatopenshift.VMSize("Standard_D8s_v3"),
			},
			NetworkProfile: &mgmtredhatopenshift.NetworkProfile{
				PodCidr:     to.StringPtr("10.128.0.0/14"),
				ServiceCidr: to.StringPtr("172.30.0.0/16"),
			},
			WorkerProfiles: &[]mgmtredhatopenshift.WorkerProfile{
				mgmtredhatopenshift.WorkerProfile{
					SubnetID:   to.StringPtr(os.Getenv("CLUSTER") + "-worker"),
					VMSize:     mgmtredhatopenshift.VMSize1("Standard_D2s_v3"),
					Name:       to.StringPtr("worker"),
					DiskSizeGB: to.Int32Ptr(128),
					Count:      to.Int32Ptr(3),
				},
			},
			ServicePrincipalProfile: &mgmtredhatopenshift.ServicePrincipalProfile{
				ClientID:     to.StringPtr(os.Getenv("CLUSTER_SPN_ID")),
				ClientSecret: to.StringPtr(os.Getenv("CLUSTER_SPN_SECRET")),
			},
		},
	}
	spew.Dump(parameters)

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
