package graphrbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
)

// ServicePrincipalsClient is a minimal interface for azure ApplicationsClient
type ServicePrincipalsClient interface {
	Get(ctx context.Context, objectID string) (result graphrbac.ServicePrincipal, err error)
	Delete(ctx context.Context, objectID string) (result autorest.Response, err error)
}

type servicePrincipalsClient struct {
	graphrbac.ServicePrincipalsClient
}

var _ ServicePrincipalsClient = &servicePrincipalsClient{}

// NewNewServicePrincipalsClient creates a new ApplicationsClient
func NewServicePrincipalsClient(tenantID string, authorizer autorest.Authorizer) ServicePrincipalsClient {
	client := graphrbac.NewServicePrincipalsClient(tenantID)
	client.Authorizer = authorizer

	return &servicePrincipalsClient{
		ServicePrincipalsClient: client,
	}
}
