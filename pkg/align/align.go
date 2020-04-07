package align

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Azure/ARO-RP/pkg/install"

	"github.com/Azure/ARO-RP/pkg/env"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/database"

	"github.com/sirupsen/logrus"
)

var apiVersions = map[string]string{
	"authorization-denyassignment": "2018-07-01-preview",
}

type aligner struct {
	log *logrus.Entry
	db  *database.Database
	env env.Interface
}

func New(log *logrus.Entry, env env.Interface, db *database.Database) *aligner {
	return &aligner{
		log: log,
		db:  db,
		env: env,
	}
}

func (a *aligner) CreateOrUpdateDenyAssignment(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	// initiate installer so we could re-use code
	i, err := install.NewInstaller(ctx, a.log, a.env, a.db.OpenShiftClusters, a.db.Billing, doc)
	if err != nil {
		return err
	}

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

	err = i.DeployARMTemplate(ctx, resourceGroup, "denyassignment", t, nil)
	if err != nil {
		return err
	}

	return nil
}

func (a *aligner) InstallerFixups(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	// initiate installer so we could re-use code
	a.log.Infof("cluster: %s", doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
	i, err := install.NewInstaller(ctx, a.log, a.env, a.db.OpenShiftClusters, a.db.Billing, doc)
	if err != nil {
		return err
	}

	err = i.InitializeKubernetesClients(ctx)
	if err != nil {
		return err
	}

	a.log.Info("creating billing table:")
	err = i.CreateBillingRecord(ctx)
	if err != nil {
		return err
	}

	a.log.Info("creating alertmanager")
	err = i.DisableAlertManagerWarning(ctx)
	if err != nil {
		return err
	}

	a.log.Info("remove ignition config")
	err = i.RemoveBootstrapIgnition(ctx)
	if err != nil {
		return err
	}

	a.log.Info("ensure genevaLogging")
	err = i.EnsureGenevaLogging(ctx)
	if err != nil {
		return err
	}
	return nil

}
