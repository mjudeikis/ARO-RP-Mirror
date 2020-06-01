package purge

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/davecgh/go-spew/spew"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
)

var (
	appNameRx = regexp.MustCompile("aro-[a-z0-9]{8}")
	appUrlRx  = regexp.MustCompile("https://az.aro.azure.com")
)

func (rc *ResourceCleaner) CleanRBAC(ctx context.Context) error {
	return rc.cleanVNETRoleBinding(ctx)
}

func (rc *ResourceCleaner) cleanVNETRoleBinding(ctx context.Context) error {
	for _, resourceGroup := range rc.resourceRegions {
		// list all vnets and those vnets rolebindings
		vnets, err := rc.vnetscli.List(ctx, resourceGroup)
		if err != nil {
			return err
		}

		var servicePrincipals []string
		for _, vnet := range vnets {
			scope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s", rc.subscriptionID, resourceGroup, *vnet.Name)
			ras, err := rc.roleassignmentcli.ListForScope(ctx, scope, "")
			if err != nil {
				return err
			}
			for _, ra := range ras {
				if ra.RoleAssignmentPropertiesWithScope.PrincipalType == mgmtauthorization.ServicePrincipal {
					servicePrincipals = append(servicePrincipals, *ra.RoleAssignmentPropertiesWithScope.PrincipalID)
				}
			}
		}
		err = rc.cleanServicePrincipals(ctx, servicePrincipals)
		if err != nil {
			return err
		}

	}
	return nil
}

func (rc *ResourceCleaner) cleanServicePrincipals(ctx context.Context, servicePrincipals []string) error {
	for _, sp := range servicePrincipals {
		_sp, err := rc.serviceprincipalcli.Get(ctx, sp)
		if err != nil {
			if detailedErr, ok := err.(autorest.DetailedError); ok &&
				detailedErr.StatusCode == http.StatusNotFound {
				continue
			}
			return err
		}
		spew.Dump(_sp.DisplayName)

		if !rgNameRx.MatchString(*_sp.DisplayName) ||
			len(*_sp.IdentifierUris) != 1 ||
			strings.HasPrefix((*_sp.)[0], "https://az.aro.azure.com") {
			continue
		}
		spew.Dump(*sp.DisplayName)

	}
	return nil

}
