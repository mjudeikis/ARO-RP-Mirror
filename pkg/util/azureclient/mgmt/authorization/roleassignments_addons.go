package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// RoleAssignmentsClientAddons contains addons for RoleassRoleAssignmentsClient
type RoleAssignmentsClientAddons interface {
	ListForScope(ctx context.Context, scope, filter string) (roleassignments []mgmtauthorization.RoleAssignment, err error)
}

func (c *roleAssignmentsClient) ListForScope(ctx context.Context, scope, filter string) (roleassignments []mgmtauthorization.RoleAssignment, err error) {
	page, err := c.RoleAssignmentsClient.ListForScope(ctx, scope, filter)
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
