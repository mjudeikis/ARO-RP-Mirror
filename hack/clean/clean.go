package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"

	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/purge"
)

var (
	dryRun = flag.Bool("dryRun", true, "dry run")
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
	log.Infof("Starting the resource cleaner, DryRun: %t", *dryRun)

	rc, err := purge.NewResourceCleaner(log, *dryRun)
	if err != nil {
		return err
	}

	//err = rc.CleanRBAC(ctx)
	//if err != nil {
	//	return err
	//}

	return rc.CleanResourceGroups(ctx)
}
