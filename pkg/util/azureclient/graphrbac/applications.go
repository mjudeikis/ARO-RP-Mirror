package graphrbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
)

// ApplicationsClient is a minimal interface for azure ApplicationsClient
type ApplicationsClient interface {
	GetServicePrincipalsIDByAppID(ctx context.Context, applicationID string) (result graphrbac.ServicePrincipalObjectResult, err error)
	List(ctx context.Context, filter string) (result graphrbac.ApplicationListResultPage, err error)
	Delete(ctx context.Context, applicationObjectID string) (result autorest.Response, err error)
	Get(ctx context.Context, applicationObjectID string) (result graphrbac.Application, err error)
}

type applicationsClient struct {
	graphrbac.ApplicationsClient
}

var _ ApplicationsClient = &applicationsClient{}

// NewApplicationsClient creates a new ApplicationsClient
func NewApplicationsClient(tenantID string, authorizer autorest.Authorizer) ApplicationsClient {
	client := graphrbac.NewApplicationsClient(tenantID)
	client.Authorizer = authorizer

	return &applicationsClient{
		ApplicationsClient: client,
	}
}
