package dbload

import (
	"context"
	"testing"

	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func TestQuery(t *testing.T) {
	ctx := context.Background()

	c, err := get(ctx)
	if err != nil {
		t.Fatal(err)
	}

	docs, err := c.QueryAll(ctx, "", &cosmosdb.Query{
		Query: `SELECT * FROM OpenShiftClusters doc ` +
			`WHERE doc.openShiftCluster.properties.provisioningState = "Creating" ` +
			`AND (doc.leaseExpires ?? 0) < 1000000`,
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Log(docs.Count)
}
