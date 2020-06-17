package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func (i *Installer) fixPVC(ctx context.Context) error {
	// TODO: this function does not currently reapply a pull secret in
	// development mode.

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pvcs, err := i.kubernetescli.CoreV1().PersistentVolumeClaims(v1.NamespaceAll).List(metav1.ListOptions{})
		switch {
		case errors.IsNotFound(err):
			i.log.Info("no pvc's found")
			return nil
		case err != nil:
			return err
		}

		for _, pvc := range pvcs.Items {
			if pvc.Status.Phase == v1.ClaimPending {
				err := i.kubernetescli.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{})
				i.log.Info(err)
			}
		}
		return nil
	})
}
