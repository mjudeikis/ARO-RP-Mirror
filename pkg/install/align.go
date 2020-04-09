package install

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (i *Installer) CreateOrUpdateDenyAssignment(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	spp := doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	conf := auth.NewClientCredentialsConfig(spp.ClientID, string(spp.ClientSecret), spp.TenantID)
	conf.Resource = azure.PublicCloud.GraphEndpoint

	spGraphAuthorizer, err := conf.Authorizer()
	if err != nil {
		return err
	}

	applications := graphrbac.NewApplicationsClient(spp.TenantID, spGraphAuthorizer)

	res, err := applications.GetServicePrincipalsIDByAppID(ctx, spp.ClientID)
	if err != nil {
		return err
	}

	clusterSPObjectID := *res.Value

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			{
				Resource: &mgmtauthorization.DenyAssignment{
					Name: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
					Type: to.StringPtr("Microsoft.Authorization/denyAssignments"),
					DenyAssignmentProperties: &mgmtauthorization.DenyAssignmentProperties{
						DenyAssignmentName: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
						Permissions: &[]mgmtauthorization.DenyAssignmentPermission{
							{
								Actions: &[]string{
									"*/action",
									"*/delete",
									"*/write",
								},
								NotActions: &[]string{
									"Microsoft.Network/networkSecurityGroups/join/action",
								},
							},
						},
						Scope: &doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
						Principals: &[]mgmtauthorization.Principal{
							{
								ID:   to.StringPtr("00000000-0000-0000-0000-000000000000"),
								Type: to.StringPtr("SystemDefined"),
							},
						},
						ExcludePrincipals: &[]mgmtauthorization.Principal{
							{
								ID:   &clusterSPObjectID,
								Type: to.StringPtr("ServicePrincipal"),
							},
						},
						IsSystemProtected: to.BoolPtr(true),
					},
				},
				APIVersion: apiVersions["authorization-denyassignment"],
			},
		},
	}

	resourceGroup := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	i.log.Info("deploying")
	err = i.deployments.CreateOrUpdateAndWait(ctx, resourceGroup, "denyassignment", mgmtresources.Deployment{
		Properties: &mgmtresources.DeploymentProperties{
			Template: t,
			Mode:     mgmtresources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	i.log.Info("deleting deployment")
	return i.deployments.DeleteAndWait(ctx, resourceGroup, "denyassignment")
}

func (i *Installer) InstallerFixups(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	err := i.initializeKubernetesClients(ctx)
	if err != nil {
		return err
	}

	i.log.Info("creating billing record")
	err = i.createBillingRecord(ctx)
	if err != nil {
		return err
	}

	i.log.Info("disable alertmanager warning")
	err = i.disableAlertManagerWarning(ctx)
	if err != nil {
		return err
	}

	i.log.Info("remove bootstrap ignition")
	err = i.removeBootstrapIgnition(ctx)
	if err != nil {
		return err
	}

	i.log.Info("ensure genevaLogging")
	return i.ensureGenevaLogging(ctx)
}

func (i *Installer) ConfigurationFixup(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	scc, err := i.securitycli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
	if err != nil {
		return err
	}

	var needsUpdate bool
	var users []string
	for _, u := range scc.Users {
		if u != "system:serviceaccount:openshift-azure-logging:geneva" {
			users = append(users, u)
		} else {
			needsUpdate = true
		}
	}

	if needsUpdate {
		scc.Users = users

		_, err := i.securitycli.SecurityV1().SecurityContextConstraints().Update(scc)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) KubeConfigFixup(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	g, err := i.loadGraph(ctx)
	if err != nil {
		return err
	}

	aroServiceInternalClient, err := i.generateAROServiceKubeconfig(g)
	if err != nil {
		return err
	}

	_, err = i.db.Patch(ctx, doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.AROServiceKubeconfig = aroServiceInternalClient.File.Data
		return nil
	})
	return err
}

func (i *Installer) ClusterExists(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error) {
	resourceGroup := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	// if resource group delete was attempted, it will fail with "linked" resource
	// error due to private endpoint to management vnet. We check existence of the
	// cluster by checking aro lb existence because it will be deleted.
	_, err := i.loadbalancers.Get(ctx, resourceGroup, "aro", "")
	if isNotFoundError(err) {
		return false, nil
	}

	return err == nil, err
}

func isNotFoundError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestError, ok := detailedErr.Original.(*azure.RequestError); ok {
			if requestError.ServiceError.Code == "ResourceNotFound" {
				return true
			}
		}
	}
	return false
}
