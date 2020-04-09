package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Azure/ARO-RP/pkg/install"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/metrics/noop"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
)

func baseline(ctx context.Context, log *logrus.Entry, regex string) error {
	if regex == "" {
		return fmt.Errorf("must provide regex")
	}

	// example:
	// ^\/subscriptions\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\/resourceGroups\/[-a-z0-9_().]{0,89}[-a-z0-9_()]\/providers\/Microsoft.RedHatOpenShift\/openShiftClusters\/[0-9a-z_]{0,89}$
	rxResourceID, err := regexp.Compile(regex)
	if err != nil {
		return err
	}

	log.Infof("regex %s", regex)

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

	docs, err := db.OpenShiftClusters.ListAll(ctx)
	if err != nil {
		return err
	}

	log.Infof("cluster count %d", len(docs.OpenShiftClusterDocuments))

	for _, doc := range docs.OpenShiftClusterDocuments {
		if rxResourceID.MatchString(doc.OpenShiftCluster.ID) {
			log.Infof("cluster %s", doc.OpenShiftCluster.ID)

			i, err := install.NewInstaller(ctx, log, _env, db.OpenShiftClusters, db.Billing, doc)
			if err != nil {
				return err
			}

			exists, err := i.ClusterExists(ctx, doc)
			if err != nil {
				return err
			}
			if !exists {
				log.Info("load balancer not found, skipping")
				continue
			}

			log.Info("creating deny assignment")
			err = i.CreateOrUpdateDenyAssignment(ctx, doc)
			if err != nil {
				return err
			}

			log.Info("installer fixups")
			err = i.InstallerFixups(ctx, doc)
			if err != nil {
				return err
			}

			log.Info("configuration fixup")
			err = i.ConfigurationFixup(ctx, doc)
			if err != nil {
				return err
			}

			log.Info("kubeconfig fixup")
			err = i.KubeConfigFixup(ctx, doc)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
