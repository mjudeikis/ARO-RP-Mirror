package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	deployer "github.com/Azure/ARO-RP/pkg/deploy"
)

type config struct {
	configFile    string
	deployVersion string
	location      string
	mode          deployer.DeploymentPhase
}

func deploy(ctx context.Context, log *logrus.Entry) error {
	c, err := parseFlags()
	if err != nil {
		return err
	}

	config, err := deployer.GetConfig(c.configFile, c.location)
	if err != nil {
		return err
	}

	d, err := deployer.New(ctx, log, config, c.deployVersion)
	if err != nil {
		return err
	}

	switch c.mode {
	case deployer.DeploymentPhaseP:
		return d.PreDeploy(ctx)
	case deployer.DeploymentPhaseD:
		return d.Deploy(ctx)
	case deployer.DeploymentPhaseU:
		return d.Upgrade(ctx)
	}
	return nil
}

func parseFlags() (*config, error) {
	c := &config{}

	c.deployVersion = gitCommit
	if os.Getenv("RP_VERSION") != "" {
		c.deployVersion = os.Getenv("RP_VERSION")
	}

	if c.deployVersion == "unknown" || strings.Contains(c.deployVersion, "dirty") {
		return nil, fmt.Errorf("invalid deploy version %q", c.deployVersion)
	}

	if strings.ToLower(flag.Arg(3)) != flag.Arg(3) {
		return nil, fmt.Errorf("location %s must be lower case", flag.Arg(3))
	}

	c.configFile = flag.Arg(1)
	c.location = flag.Arg(3)

	switch strings.ToLower(flag.Arg(2)) {
	case "p", "predeploy":
		c.mode = deployer.DeploymentPhaseP
	case "d", "deploy":
		c.mode = deployer.DeploymentPhaseD
	case "u", "upgrade":
		c.mode = deployer.DeploymentPhaseU
	default:
		return nil, fmt.Errorf("deployment phase %s not found", flag.Arg(2))
	}

	return c, nil
}
