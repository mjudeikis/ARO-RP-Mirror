package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (mon *Monitor) emitClusterOperatorsVersions(ctx context.Context) error {
	cos, err := mon.configcli.ConfigV1().ClusterOperators().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	sort.Slice(cos.Items, func(i, j int) bool { return cos.Items[i].Name < cos.Items[j].Name })
	for _, co := range cos.Items {
		for _, v := range co.Status.Versions {
			if v.Name != "operator" {
				continue
			}

			if v.Version == version.OpenShiftVersion {
				continue
			}

			mon.emitGauge("clusteroperators.version", 1, map[string]string{
				"name":    co.Name,
				"version": v.Version,
			})

		}
	}

	return nil
}
