package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/purge"
)

const (
	defaultTTL     = 48
	defaultTTLTag  = "createdAt"
	defaultKeepTag = "persist"
	defaultDryRun  = false
)

func main() {
	ctx := context.Background()
	log := utillog.GetLogger()

	flag.Parse()

	if err := run(ctx, log); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, log *logrus.Entry) error {
	var ttl time.Duration
	if os.Getenv("AZURE_PURGE_TTL") != "" {
		var err error
		ttl, err = time.ParseDuration(os.Getenv("AZURE_PURGE_TTL"))
		if err != nil {
			return err
		}
	} else {
		ttl = defaultTTL
	}

	var createdTag = defaultTTLTag
	if os.Getenv("AZURE_PURGE_CREATED_TAG") != "" {
		createdTag = os.Getenv("AZURE_PURGE_CREATED_TAG")
	}

	deleteGroupPrefixes := []string{}
	if os.Getenv("AZURE_PURGE_RG_DEL_PREFIX") != "" {
		deleteGroupPrefixes = strings.Split(os.Getenv("AZURE_PURGE_RG_DEL_PREFIX"), ",")
	}

	dryRun := defaultDryRun
	if os.Getenv("AZURE_PURGE_DRYRUN") != "" {
		var err error
		dryRun, err = strconv.ParseBool(os.Getenv("AZURE_PURGE_DRYRUN"))
		if err != nil {
			return err
		}
	}

	shouldDelete := func(resourceGroup mgmtfeatures.ResourceGroup, log *logrus.Entry) bool {
		isDeleteGroup := false
		for _, deleteGroupPrefix := range deleteGroupPrefixes {
			if strings.HasPrefix(*resourceGroup.Name, deleteGroupPrefix) {
				isDeleteGroup = true
				break
			}
		}
		if isDeleteGroup {
			return false
		}

		for t := range resourceGroup.Tags {
			if strings.ToLower(t) == defaultKeepTag {
				log.Infof("Group %s is to persist. SKIP.", *resourceGroup.Name)
				return false
			}
		}

		// azure tags is not consistent with lower/upper cases.
		if _, ok := resourceGroup.Tags[createdTag]; !ok {
			log.Infof("Group %s does not have createdAt tag. SKIP.", *resourceGroup.Name)
			return false
		}

		createdAt, err := time.Parse(time.RFC3339Nano, *resourceGroup.Tags[createdTag])
		if err != nil {
			log.Errorf("%s: %s", *resourceGroup.Name, err)
			return false
		}
		if time.Now().Sub(createdAt) < ttl {
			log.Infof("Group %s is still less than TTL. SKIP.", *resourceGroup.Name)
			return false
		}

		return true
	}

	log.Infof("Starting the resource cleaner, DryRun: %t", dryRun)

	rc, err := purge.NewResourceCleaner(log, shouldDelete, dryRun)
	if err != nil {
		return err
	}

	return rc.CleanResourceGroups(ctx)
}
