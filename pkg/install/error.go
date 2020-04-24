package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
)

// hasAuthorizationFailedError returns true it the error is, or contains, an
// AuthorizationFailed error
func hasAuthorizationFailedError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			if serviceErr.Code == "AuthorizationFailed" {
				return true
			}
		}
	}

	if serviceErr, ok := err.(*azure.ServiceError); ok &&
		serviceErr.Code == "DeploymentFailed" {
		for _, d := range serviceErr.Details {
			if code, ok := d["code"].(string); ok &&
				code == "Forbidden" {
				if message, ok := d["message"].(string); ok {
					var ce *api.CloudError
					if json.Unmarshal([]byte(message), &ce) == nil &&
						ce.CloudErrorBody != nil &&
						ce.CloudErrorBody.Code == "AuthorizationFailed" {
						return true
					}
				}
			}
		}
	}

	return false
}

// hasResourceQuotaExceededError returns true and the original error message if
// the error contains a QuotaExceeded error
func hasResourceQuotaExceededError(err error) (bool, string) {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if serviceErr, ok := detailedErr.Original.(*azure.ServiceError); ok {
			for _, d := range serviceErr.Details {
				if code, ok := d["code"].(string); ok && code == "QuotaExceeded" {
					if message, ok := d["message"].(string); ok {
						return true, message
					}
				}
			}
		}
	}
	return false, ""
}

// hasPrivateLinkError returns true and the original error message if
// the error contains a PrivateLinkServiceCannotBeCreatedInSubnetThatHasNetworkPoliciesEnabled error
func hasPrivateLinkError(err error) (bool, string) {
	if serviceErr, ok := err.(*azure.ServiceError); ok &&
		serviceErr.Code == "DeploymentFailed" {
		for _, d := range serviceErr.Details {
			if d["code"] == "BadRequest" &&
				strings.Contains(d["message"].(string), "PrivateLinkServiceCannotBeCreatedInSubnetThatHasNetworkPoliciesEnabled") {
				var e struct {
					Err azure.ServiceError `json:"error"`
				}
				// this particular error comes as string and not error :/
				err := json.Unmarshal([]byte(d["message"].(string)), &e)
				if err == nil && e.Err.Message != "" {
					return true, e.Err.Message
				}
			}
		}
	}
	return false, ""
}

// isDeploymentActiveError returns true it the error is a DeploymentActive error
func isDeploymentActiveError(err error) bool {
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "DeploymentActive" {
			return true
		}
	}
	return false
}
