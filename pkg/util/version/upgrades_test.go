package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
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

func TestLatest(t *testing.T) {
	count := 0
	for _, u := range Upgrades {
		if u.Latest {
			count++
		}
	}
	if count != 1 {
		t.Error("multiple upgrade.Latest=true found")
	}
}

func TestUnique(t *testing.T) {
	unique := make(map[string]int, len(Upgrades))
	for _, u := range Upgrades {
		unique[fmt.Sprintf("%d.%d", u.Version.V[0], u.Version.V[1])]++
	}

	for i, j := range unique {
		if j > 1 {
			t.Errorf("multiple x.Y version upgrade path found for %s", i)
		}
	}

}
