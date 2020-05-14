package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// RoleAssignmentsClientAddons contains addons for RoleassRoleAssignmentsClient
type RoleAssignmentsClientAddons interface {
	List(ctx context.Context, filter string) (roleassignments []mgmtauthorization.RoleAssignment, err error)
}

func (c *roleAssignmentsClient) List(ctx context.Context, resourceGroupName string) (roleassignments []mgmtauthorization.RoleAssignment, err error) {
	page, err := c.RoleAssignmentsClient.List(ctx, resourceGroupName)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		roleassignments = append(roleassignments, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return roleassignments, nil
}
