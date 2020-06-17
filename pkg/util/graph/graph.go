package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/url"
	"reflect"
	"time"

	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/ignition/bootstrap"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"

	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

type Graph map[reflect.Type]asset.Asset

func (g Graph) Resolve(a asset.Asset) (asset.Asset, error) {
	if _, found := g[reflect.TypeOf(a)]; !found {
		for _, dep := range a.Dependencies() {
			_, err := g.Resolve(dep)
			if err != nil {
				return nil, err
			}
		}

		err := a.Generate(asset.Parents(g))
		if err != nil {
			return nil, err
		}

		g[reflect.TypeOf(a)] = a
	}

	return g[reflect.TypeOf(a)], nil
}

type Interface interface {
	Exists(ctx context.Context) (bool, error)
	Load(ctx context.Context) (Graph, error)
	Save(ctx context.Context, gr Graph) error
}

type graph struct {
	log    *logrus.Entry
	env    env.Interface
	cipher encryption.Cipher

	oc *api.OpenShiftCluster

	accounts storage.AccountsClient
}

// New return new graph object
func New(ctx context.Context, log *logrus.Entry, _env env.Interface, oc *api.OpenShiftCluster) (Interface, error) {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return nil, err
	}

	fpAuthorizer, err := _env.FPAuthorizer(oc.Properties.ServicePrincipalProfile.TenantID, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, _env, env.EncryptionSecretName)
	if err != nil {
		return nil, err
	}

	return &graph{
		log:      log,
		env:      _env,
		oc:       oc,
		cipher:   cipher,
		accounts: storage.NewAccountsClient(r.SubscriptionID, fpAuthorizer),
	}, nil
}

func (g *graph) Exists(ctx context.Context) (bool, error) {
	g.log.Print("checking if graph exists")

	blobService, err := g.getBlobService(ctx, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return false, err
	}

	aro := blobService.GetContainerReference("aro")
	return aro.GetBlobReference("graph").Exists()
}

func (g *graph) getBlobService(ctx context.Context, p mgmtstorage.Permissions, r mgmtstorage.SignedResourceTypes) (*azstorage.BlobStorageClient, error) {
	resourceGroup := stringutils.LastTokenByte(g.oc.Properties.ClusterProfile.ResourceGroupID, '/')

	t := time.Now().UTC().Truncate(time.Second)
	res, err := g.accounts.ListAccountSAS(ctx, resourceGroup, "cluster"+g.oc.Properties.StorageSuffix, mgmtstorage.AccountSasParameters{
		Services:               "b",
		ResourceTypes:          r,
		Permissions:            p,
		Protocols:              mgmtstorage.HTTPS,
		SharedAccessStartTime:  &date.Time{Time: t},
		SharedAccessExpiryTime: &date.Time{Time: t.Add(24 * time.Hour)},
	})
	if err != nil {
		return nil, err
	}

	v, err := url.ParseQuery(*res.AccountSasToken)
	if err != nil {
		return nil, err
	}

	c := azstorage.NewAccountSASClient("cluster"+g.oc.Properties.StorageSuffix, v, azure.PublicCloud).GetBlobService()

	return &c, nil
}

func (g *graph) Load(ctx context.Context) (Graph, error) {
	g.log.Print("load graph")

	blobService, err := g.getBlobService(ctx, mgmtstorage.Permissions("r"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return nil, err
	}

	aro := blobService.GetContainerReference("aro")
	cluster := aro.GetBlobReference("graph")
	rc, err := cluster.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	encrypted, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	output, err := g.cipher.Decrypt(encrypted)
	if err != nil {
		return nil, err
	}

	var gr Graph
	err = json.Unmarshal(output, &gr)
	if err != nil {
		return nil, err
	}

	return gr, nil
}

func (g *graph) Save(ctx context.Context, gr Graph) error {
	g.log.Print("save graph")

	blobService, err := g.getBlobService(ctx, mgmtstorage.Permissions("cw"), mgmtstorage.SignedResourceTypesO)
	if err != nil {
		return err
	}

	bootstrap := gr[reflect.TypeOf(&bootstrap.Bootstrap{})].(*bootstrap.Bootstrap)
	bootstrapIgn := blobService.GetContainerReference("ignition").GetBlobReference("bootstrap.ign")
	err = bootstrapIgn.CreateBlockBlobFromReader(bytes.NewReader(bootstrap.File.Data), nil)
	if err != nil {
		return err
	}

	graph := blobService.GetContainerReference("aro").GetBlobReference("graph")
	b, err := json.MarshalIndent(g, "", "    ")
	if err != nil {
		return err
	}

	output, err := g.cipher.Encrypt(b)
	if err != nil {
		return err
	}

	return graph.CreateBlockBlobFromReader(bytes.NewReader([]byte(output)), nil)
}
