package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"regexp"

	"github.com/Azure/ARO-RP/pkg/install"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func baseline(ctx context.Context, log *logrus.Entry, r string) error {
	uuid := uuid.NewV4().String()
	log.Printf("uuid %s", uuid)

	_env, err := env.NewEnv(ctx, log)
	if err != nil {
		return err
	}

	cipher, err := encryption.NewXChaCha20Poly1305(ctx, _env, env.EncryptionSecretName)
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

	log.Infof("cluster count %d", len(docs))

	// example:
	// ^\/subscriptions\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\/resourceGroups\/[-a-z0-9_().]{0,89}[-a-z0-9_()]\/providers\/Microsoft.RedHatOpenShift\/openShiftClusters\/[0-9a-z_]{0,89}$
	log.Infof("running with rx %s", r)
	RxResourceID := regexp.MustCompile(r)

	for _, doc := range docs {
		if RxResourceID.MatchString(doc.OpenShiftCluster.ID) {
			log.Infof("cluster: %s", doc.OpenShiftCluster.ID)

			i, err := install.NewInstaller(ctx, log, _env, db.OpenShiftClusters, db.Billing, doc)
			if err != nil {
				return err
			}

			exits, err := i.ClusterExits(ctx, doc)
			if err != nil {
				return err
			}
			if !exits {
				log.Info("RG not found. Skipping.")
				continue
			}

			// currently this in our sub gives :
			// App is not authorized to perform non read operations on a System Deny Assignment.
			log.Info("creating deny assignment")
			err = i.CreateOrUpdateDenyAssignment(ctx, doc)
			if err != nil {
				return err
			}

			err = i.InstallerFixups(ctx, doc)
			if err != nil {
				return err
			}

			err = i.KubeConfigFixup(ctx, doc)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
