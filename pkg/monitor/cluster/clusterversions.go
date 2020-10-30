package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"

	pkgoperator "github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (mon *Monitor) emitClusterVersions(ctx context.Context) error {
	cv, err := mon.getClusterVersion(ctx)
	if err != nil {
		return err
	}

	dl, err := mon.listDeployments(ctx)
	if err != nil {
		return err
	}

	operatorVersion := "unknown" // TODO(mj): Once unknown is not present anymore, simplify this
	for _, d := range dl.Items {
		if d.Namespace == pkgoperator.Namespace && d.Name == "aro-operator-master" {
			if d.Labels != nil {
				if val, ok := d.Labels["version"]; ok {
					operatorVersion = val
				}
			}
		}
	}

	mon.emitGauge("cluster.versions", 1, map[string]string{
		"actualVersion":                        actualVersion(cv),
		"desiredVersion":                       desiredVersion(cv),
		"provisionedByResourceProviderVersion": mon.oc.Properties.ProvisionedBy, // last successful Put or Patch
		"resourceProviderVersion":              version.GitCommit,               // RP version currently running
		"operatorVersion":                      operatorVersion,                 // operator version in the cluster
	})

	return nil
}

// actualVersion finds the actual current cluster state. The history is ordered by most
// recent first, so find the latest "Completed" status to get current
// cluster version
func actualVersion(cv *configv1.ClusterVersion) string {
	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return history.Version
		}
	}
	return ""
}

func desiredVersion(cv *configv1.ClusterVersion) string {
	if cv.Spec.DesiredUpdate != nil &&
		cv.Spec.DesiredUpdate.Version != "" {
		return cv.Spec.DesiredUpdate.Version
	}

	return cv.Status.Desired.Version
}
