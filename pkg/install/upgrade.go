package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (i *Installer) upgradeCluster(ctx context.Context) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cv, err := i.configcli.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
		if err != nil {
			return err
		}

		if cv.Spec.Channel != "" {
			i.log.Printf("not upgrading: cvo channel is %s", cv.Spec.Channel)
			return nil
		}

		desired, err := version.ParseVersion(cv.Status.Desired.Version)
		if err != nil {
			return err
		}

		// Get Cluster upgrade version based on desired version
		// If desired is 4.3.x we return 4.3 channel update
		// If desired is 4.4.x we return 4.4 channel update
		upgrade, err := version.GetUpgrade(desired)
		if err != nil {
			return err
		}

		if !desired.Lt(upgrade.Version) {
			i.log.Printf("not upgrading: cvo desired version is %s", cv.Status.Desired.Version)
			return nil
		}

		i.log.Printf("initiating cluster upgrade, target version %s", upgrade.Version.String())

		cv.Spec.DesiredUpdate = &configv1.Update{
			Version: upgrade.Version.String(),
			Image:   upgrade.PullSpec,
		}

		_, err = i.configcli.ConfigV1().ClusterVersions().Update(cv)
		return err
	})
}
