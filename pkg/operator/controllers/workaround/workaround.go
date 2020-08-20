package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type Workaround interface {
	Name() string
	IsRequired(clusterVersion *version.Version) bool
	Ensure() error
	Remove() error
}
