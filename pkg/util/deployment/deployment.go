package deployment

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"strings"
)

type mode int

type Mode interface {
	DeploymentMode() Mode
}

const (
	Production mode = iota
	Integration
	Development
)

func (m mode) DeploymentMode() Mode {
	return m
}

func NewMode() Mode {
	switch strings.ToLower(os.Getenv("RP_MODE")) {
	case "development":
		return Development
	case "int":
		return Integration
	default:
		return Production
	}
}
