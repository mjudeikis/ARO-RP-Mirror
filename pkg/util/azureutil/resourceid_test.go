package azureutil

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
)

func TestResourceID(t *testing.T) {
	for _, tt := range []struct {
		name     string
		resource *azure.Resource
		want     string
		wantErr  error
	}{
		{
			name:    "nil",
			wantErr: fmt.Errorf("resourceId create failed. resource is nil"),
		},
		{
			name: "good",
			resource: &azure.Resource{
				SubscriptionID: "subscriptionId",
				ResourceGroup:  "resourceGroup",
				Provider:       "Microsoft.RedHatOpenShift",
				ResourceType:   "OpenShiftClusters",
				ResourceName:   "resourceName",
			},
			want: "/subscriptions/subscriptionId/resourceGroups/resourceGroup/providers/Microsoft.RedHatOpenShift/OpenShiftClusters/resourceName",
		},
		{
			name: "bad",
			resource: &azure.Resource{
				SubscriptionID: "subscriptionId",
				ResourceGroup:  "resourceGroup",
				Provider:       "Microsoft.Network",
				ResourceType:   "loadBalancers",
			},
			wantErr: fmt.Errorf("resourceId create failed. Field 'ResourceName' is empty"),
		},
		{
			name: "complex bad",
			resource: &azure.Resource{
				SubscriptionID: "subscriptionId",
				ResourceGroup:  "resourceGroup",
				Provider:       "Microsoft.Network",
				ResourceType:   "loadBalancers",
				ResourceName:   "name/subresource",
			},
			wantErr: fmt.Errorf("resourceId create failed. Field 'ResourceName' contains /"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResourceID(tt.resource)
			if err != nil && tt.wantErr != nil &&
				!reflect.DeepEqual(tt.wantErr, err) {
				t.Error(err)
				t.FailNow()
			}
			if tt.want != got {
				t.Error(got + " != " + tt.want)
			}
		})
	}
}
