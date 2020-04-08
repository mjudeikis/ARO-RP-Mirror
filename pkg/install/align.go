package install

import (
	"context"
	"log"
	"reflect"
	"strings"
	"time"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (i *Installer) CreateOrUpdateDenyAssignment(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	// this is almost full copy from installStorage
	var clusterSPObjectID string
	spp := doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	conf := auth.NewClientCredentialsConfig(spp.ClientID, string(spp.ClientSecret), spp.TenantID)
	conf.Resource = azure.PublicCloud.GraphEndpoint

	token, err := conf.ServicePrincipalToken()
	if err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// get a token, retrying only on AADSTS700016 errors (slow AAD propagation).
	err = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		err = token.EnsureFresh()
		switch {
		case err == nil:
			return true, nil
		case strings.Contains(err.Error(), "AADSTS700016"):
			log.Print(err)
			return false, nil
		default:
			return false, err
		}
	}, timeoutCtx.Done())
	if err != nil {
		return err
	}

	spGraphAuthorizer := autorest.NewBearerAuthorizer(token)

	applications := graphrbac.NewApplicationsClient(spp.TenantID, spGraphAuthorizer)

	res, err := applications.GetServicePrincipalsIDByAppID(ctx, spp.ClientID)
	if err != nil {
		return err
	}

	clusterSPObjectID = *res.Value

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
	err = i.deployARMTemplate(ctx, resourceGroup, "denyassignment", t, nil)
	if err != nil {
		return err
	}

	return nil
}

func (i *Installer) InstallerFixups(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	err := i.initializeKubernetesClients(ctx)
	if err != nil {
		return err
	}

	i.log.Info("creating billing table:")
	err = i.createBillingRecord(ctx)
	if err != nil {
		return err
	}

	i.log.Info("creating alertmanager")
	err = i.disableAlertManagerWarning(ctx)
	if err != nil {
		return err
	}

	i.log.Info("remove ignition config")
	err = i.removeBootstrapIgnition(ctx)
	if err != nil {
		return err
	}

	i.log.Info("ensure genevaLogging")
	err = i.ensureGenevaLogging(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (i *Installer) ConfigurationFixup(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	priv, err := i.securitycli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
	if err != nil {
		return err
	}
	p := priv.DeepCopy()

	var needsUpdate bool
	users := []string{}
	for _, acc := range p.Users {
		if acc != "system:serviceaccount:openshift-azure-logging:geneva" {
			users = append(users, acc)
		} else {
			needsUpdate = true
		}
	}

	if needsUpdate {
		p.Users = users
		_, err := i.securitycli.SecurityV1().SecurityContextConstraints().Update(p)
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

	adminInternalClient := g[reflect.TypeOf(&kubeconfig.AdminInternalClient{})].(*kubeconfig.AdminInternalClient)
	aroServiceInternalClient, err := i.generateAROServiceKubeconfig(g)
	if err != nil {
		return err
	}

	_, err = i.db.Patch(ctx, doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.AdminKubeconfig = adminInternalClient.File.Data
		doc.OpenShiftCluster.Properties.AROServiceKubeconfig = aroServiceInternalClient.File.Data
		return nil
	})
	return err
}

func (i *Installer) ClusterExits(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error) {
	resourceGroup := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	// if resource group delete was attempted, it will fail with "linked" resource
	// error due to private endpoint to management vnet. We check existence of the
	// cluster by checking aro lb existence because it will be deleted.
	_, err := i.loadbalancers.Get(ctx, resourceGroup, "aro", "")
	if err != nil {
		if !isNotFoundError(err) {
			return false, err
		}
	}
	return true, nil
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
