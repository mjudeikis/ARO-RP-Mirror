package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestOpenShiftVersions(t *testing.T) {
	for _, u := range Upgrades {
		_, err := ParseVersion(u.Version.String())
		if err != nil {
			t.Error(err)
		}
	}
}
