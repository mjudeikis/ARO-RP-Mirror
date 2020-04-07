package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/align"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func baseline(ctx context.Context, log *logrus.Entry) error {
	uuid := uuid.NewV4().String()
	log.Printf("uuid %s", uuid)

	_env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, _env)
	if err != nil {
		return err
	}

	db, err := database.NewDatabase(ctx, log.WithField("component", "database"), _env, &noop.Noop{}, cipher, uuid)
	if err != nil {
		return err
	}

	docs, err := db.OpenShiftClusters.ListAll(ctx, "OpenShiftClusters")
	if err != nil {
		return err
	}

	log.Infof("cluster found %d", len(docs))

	a := align.New(log, _env, db)

	for _, doc := range docs {
		// currently this in our sub gives :
		// App is not authorized to perform non read operations on a System Deny Assignment.
		//log.Infof("creating deny assignment: %s ", doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
		//err := a.CreateOrUpdateDenyAssignment(ctx, doc)
		//if err != nil {
		//	return err
		//}

		err := a.InstallerFixups(ctx, doc)
		if err != nil {
			return err
		}

		err = a.KubeConfigFixup(ctx, doc)
		if err != nil {
			return err
		}

	}

	return nil
}
