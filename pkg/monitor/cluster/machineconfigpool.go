package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	v1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var machineConfigPoolConditionsExpected = map[v1.MachineConfigPoolConditionType]corev1.ConditionStatus{
	v1.MachineConfigPoolDegraded:       corev1.ConditionFalse,
	v1.MachineConfigPoolNodeDegraded:   corev1.ConditionFalse,
	v1.MachineConfigPoolRenderDegraded: corev1.ConditionFalse,
	v1.MachineConfigPoolUpdated:        corev1.ConditionTrue,
	v1.MachineConfigPoolUpdating:       corev1.ConditionFalse,
}

func (mon *Monitor) emitMachineConfigPoolMetrics(ctx context.Context) error {
	mcps, err := mon.mcocli.MachineconfigurationV1().MachineConfigPools().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, mcp := range mcps.Items {
		for _, c := range mcp.Status.Conditions {
			if c.Status == machineConfigPoolConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("machineconfigpools.conditions", 1, map[string]string{
				"name":   mcp.Name,
				"type":   string(c.Type),
				"status": string(c.Status),
			})
		}
	}

	return nil
}
