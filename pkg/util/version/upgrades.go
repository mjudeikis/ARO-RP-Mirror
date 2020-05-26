package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type Upgrade struct {
	Version    *Version
	PullSpec   string
	MustGather string
	Latest     bool
}

var (
	Upgrades = []Upgrade{
		{
			Version:    NewVersion(4, 3, 18),
			PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:1f0fd38ac0640646ab8e7fec6821c8928341ad93ac5ca3a48c513ab1fb63bc4b",
			MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:2e10ad0fc17f39c7a83aac32a725c78d7dd39cd9bbe3ec5ca0b76dcaa98416fa",
		},
		{
			Version:    NewVersion(4, 4, 3),
			PullSpec:   "quay.io/openshift-release-dev/ocp-release@sha256:039a4ef7c128a049ccf916a1d68ce93e8f5494b44d5a75df60c85e9e7191dacc",
			MustGather: "quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:6326b78d6c05d9925576ae8c6768f7ee846b8e99618da670b5e7b698e1dd433a",
			Latest:     true,
		},
	}
)
