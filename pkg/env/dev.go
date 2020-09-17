package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/clientauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/refreshable"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type conn struct {
	net.Conn
	r *bufio.Reader
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

var _ Interface = &dev{}

type dev struct {
	*prod

	permissions     authorization.PermissionsClient
	roleassignments authorization.RoleAssignmentsClient
	applications    graphrbac.ApplicationsClient
	deployments     features.DeploymentsClient

	proxyPool       *x509.CertPool
	proxyClientCert []byte
	proxyClientKey  *rsa.PrivateKey
}

func newDev(ctx context.Context, log *logrus.Entry) (*dev, error) {
	for _, key := range []string{
		"AZURE_ARM_CLIENT_ID",
		"AZURE_ARM_CLIENT_SECRET",
		"AZURE_FP_CLIENT_ID",
		"DATABASE_NAME",
		"PROXY_HOSTNAME",
	} {
		if _, found := os.LookupEnv(key); !found {
			return nil, fmt.Errorf("environment variable %q unset", key)
		}
	}

	// This assumes we are running from an ARO-RP checkout in development
	_, curmod, _, _ := runtime.Caller(0)
	basepath, err := filepath.Abs(filepath.Join(filepath.Dir(curmod), "../.."))
	if err != nil {
		return nil, err
	}

	d := &dev{}

	d.prod, err = newProd(ctx, log)
	if err != nil {
		return nil, err
	}

	armAuthorizer, err := auth.NewClientCredentialsConfig(os.Getenv("AZURE_ARM_CLIENT_ID"), os.Getenv("AZURE_ARM_CLIENT_SECRET"), d.TenantID()).Authorizer()
	if err != nil {
		return nil, err
	}

	d.roleassignments = authorization.NewRoleAssignmentsClient(d.SubscriptionID(), armAuthorizer)
	d.prod.clustersGenevaLoggingEnvironment = "Test"
	d.prod.clustersGenevaLoggingConfigVersion = "2.3"

	fpGraphAuthorizer, err := d.FPAuthorizer(d.TenantID(), azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	d.applications = graphrbac.NewApplicationsClient(d.TenantID(), fpGraphAuthorizer)

	fpAuthorizer, err := d.FPAuthorizer(d.TenantID(), azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	d.permissions = authorization.NewPermissionsClient(d.SubscriptionID(), fpAuthorizer)

	d.deployments = features.NewDeploymentsClient(d.TenantID(), fpAuthorizer)

	b, err := ioutil.ReadFile(path.Join(basepath, "secrets/proxy.crt"))
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(b)
	if err != nil {
		return nil, err
	}

	d.proxyPool = x509.NewCertPool()
	d.proxyPool.AddCert(cert)

	d.proxyClientCert, err = ioutil.ReadFile(path.Join(basepath, "secrets/proxy-client.crt"))
	if err != nil {
		return nil, err
	}

	b, err = ioutil.ReadFile(path.Join(basepath, "secrets/proxy-client.key"))
	if err != nil {
		return nil, err
	}

	d.proxyClientKey, err = x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *dev) InitializeAuthorizers() error {
	d.armClientAuthorizer = clientauthorizer.NewAll()
	d.adminClientAuthorizer = clientauthorizer.NewAll()
	return nil
}

func (d *dev) AROOperatorImage() string {
	override := os.Getenv("ARO_IMAGE")
	if override != "" {
		return override
	}

	return fmt.Sprintf("%s.azurecr.io/aro:%s", d.acrName, version.GitCommit)
}

func (d *dev) DatabaseName() string {
	return os.Getenv("DATABASE_NAME")
}

func (d *dev) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, fmt.Errorf("unimplemented network %q", network)
	}

	c, err := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext(ctx, network, os.Getenv("PROXY_HOSTNAME")+":443")
	if err != nil {
		return nil, err
	}

	c = tls.Client(c, &tls.Config{
		RootCAs: d.proxyPool,
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					d.proxyClientCert,
				},
				PrivateKey: d.proxyClientKey,
			},
		},
		ServerName: "proxy",
	})

	err = c.(*tls.Conn).Handshake()
	if err != nil {
		c.Close()
		return nil, err
	}

	r := bufio.NewReader(c)

	req, err := http.NewRequest(http.MethodConnect, "", nil)
	if err != nil {
		c.Close()
		return nil, err
	}
	req.Host = address

	err = req.Write(c)
	if err != nil {
		c.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(r, req)
	if err != nil {
		c.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		c.Close()
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return &conn{Conn: c, r: r}, nil
}

func (d *dev) Listen() (net.Listener, error) {
	// in dev mode there is no authentication, so for safety we only listen on
	// localhost
	return net.Listen("tcp", "localhost:8443")
}

func (d *dev) FPAuthorizer(tenantID, resource string) (refreshable.Authorizer, error) {
	oauthConfig, err := adal.NewOAuthConfig(azure.PublicCloud.ActiveDirectoryEndpoint, tenantID)
	if err != nil {
		return nil, err
	}

	sp, err := adal.NewServicePrincipalTokenFromCertificate(*oauthConfig, os.Getenv("AZURE_FP_CLIENT_ID"), d.fpCertificate, d.fpPrivateKey, resource)
	if err != nil {
		return nil, err
	}

	return refreshable.NewAuthorizer(sp), nil
}

func (d *dev) MetricsSocketPath() string {
	return "mdm_statsd.socket"
}

func (d *dev) CreateARMResourceGroupRoleAssignment(ctx context.Context, fpAuthorizer refreshable.Authorizer, resourceGroup string) error {
	d.log.Print("development mode: applying resource group role assignment")

	res, err := d.applications.GetServicePrincipalsIDByAppID(ctx, os.Getenv("AZURE_FP_CLIENT_ID"))
	if err != nil {
		return err
	}

	_, err = d.roleassignments.Create(ctx, "/subscriptions/"+d.SubscriptionID()+"/resourceGroups/"+resourceGroup, uuid.NewV4().String(), mgmtauthorization.RoleAssignmentCreateParameters{
		RoleAssignmentProperties: &mgmtauthorization.RoleAssignmentProperties{
			RoleDefinitionID: to.StringPtr("/subscriptions/" + d.SubscriptionID() + "/providers/Microsoft.Authorization/roleDefinitions/8e3af657-a8ff-443c-a75c-2fe8c4bcb635"),
			PrincipalID:      res.Value,
		},
	})
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok &&
			requestErr.ServiceError != nil &&
			requestErr.ServiceError.Code == "RoleAssignmentExists" {
			err = nil
		}
	}
	return err
}

func (d *dev) E2EStorageAccountName() string {
	return "arov4e2eint"
}

func (d *dev) E2EStorageAccountRGName() string {
	return "global-infra"
}

func (d *dev) E2EStorageAccountSubID() string {
	return "0cc1cafa-578f-4fa5-8d6b-ddfd8d82e6ea"
}

func (d *dev) ShouldDeployDenyAssignment() bool {
	return false
}
