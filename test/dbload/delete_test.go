package dbload

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

func get(ctx context.Context) (cosmosdb.OpenShiftClusterDocumentClient, error) {
	err := os.Chdir("../..")
	if err != nil {
		return nil, err
	}

	env, err := env.NewEnv(ctx, utillog.GetLogger())
	if err != nil {
		return nil, err
	}

	databaseAccount, masterKey := env.CosmosDB()

	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
	}

	err = api.AddExtensions(&h.BasicHandle)
	if err != nil {
		return nil, err
	}

	hc := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100,
			// disable HTTP/2 for now: https://github.com/golang/go/issues/36026
			TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
		},
		Timeout: 30 * time.Second,
	}

	dbc, err := cosmosdb.NewDatabaseClient(hc, h, databaseAccount, masterKey)
	if err != nil {
		return nil, err
	}

	collc := cosmosdb.NewCollectionClient(dbc, env.DatabaseName())

	return cosmosdb.NewOpenShiftClusterDocumentClient(collc, "OpenShiftClusters"), nil
}

func TestDelete(t *testing.T) {
	ctx := context.Background()

	c, err := get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	docs, err := c.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	ch := make(chan *api.OpenShiftClusterDocument)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for doc := range ch {
				err = c.Delete(ctx, doc.PartitionKey, doc, &cosmosdb.Options{NoETag: true})
				if err != nil {
					t.Log(err)
				}
			}
		}()
	}

	for _, doc := range docs.OpenShiftClusterDocuments {
		ch <- doc
	}
	close(ch)
	wg.Wait()
}
