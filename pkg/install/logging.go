package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/logging/events"
	"github.com/Azure/ARO-RP/pkg/logging/geneva"
)

func (i *Installer) ensureLogging(ctx context.Context) error {
	gl := geneva.New(i.log, i.env, i.doc.OpenShiftCluster, i.kubernetescli, i.securitycli)
	err := gl.CreateOrUpdate(ctx)
	if err != nil {
		return err
	}

	ev := events.New(i.log, i.env, i.kubernetescli)
	return ev.CreateOrUpdate()
}
