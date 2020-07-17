package generator

import "github.com/Azure/ARO-RP/pkg/deploy/config"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Generator defines generator main interface
type Generator interface {
	RPTemplate() (map[string]interface{}, error)
	RPGlobalTemplate() (map[string]interface{}, error)
	RPGlobalSubscriptionTemplate() (map[string]interface{}, error)
	RPSubscriptionTemplate() (map[string]interface{}, error)
	ManagedIdentityTemplate() (map[string]interface{}, error)
	PreDeployTemplate() (map[string]interface{}, error)
}

type generator struct {
	mode config.Mode
}

// New return new instance of generators. Generators returns different
// resource configuration, depending on the Mode of the execution.
func New(mode config.Mode) Generator {
	return &generator{
		mode: mode,
	}
}
