package azureutil

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
)

// ResourceID returns resource ID from Resource object
func ResourceID(r *azure.Resource) (string, error) {
	if r == nil {
		return "", fmt.Errorf("resourceId create failed. resource is nil")
	}
	rValues := reflect.ValueOf(r).Elem()
	rField := reflect.TypeOf(r).Elem()

	for i := 0; i < rValues.NumField(); i++ {
		if rValues.Field(i).IsZero() {
			return "", fmt.Errorf("resourceId create failed. Field '%s' is empty", rField.Field(i).Name)
		}
		if strings.Contains(rValues.Field(i).String(), "/") {
			return "", fmt.Errorf("resourceId create failed. Field '%s' contains /", rField.Field(i).Name)
		}
	}

	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s/%s",
		r.SubscriptionID,
		r.ResourceGroup,
		r.Provider,
		r.ResourceType,
		r.ResourceName,
	), nil
}
