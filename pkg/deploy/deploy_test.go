package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func TestGetParameters(t *testing.T) {
	databaseAccountName := "databaseAccountName"
	adminApiCaBundle := "adminApiCaBundle"
	extraClusterKeyVaultAccessPolicies := []interface{}{"a", "b", 1}
	for _, tt := range []struct {
		name   string
		ps     map[string]interface{}
		config Configuration
		want   arm.Parameters
	}{
		{
			name: "no parameters",
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"fullDeploy": {
						Value: false,
					},
				},
			},
		},
		{
			name: "all parameters present",
			ps: map[string]interface{}{
				"adminApiCaBundle":                   nil,
				"databaseAccountName":                nil,
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{
				DatabaseAccountName:                &databaseAccountName,
				AdminAPICABundle:                   &adminApiCaBundle,
				ExtraClusterKeyvaultAccessPolicies: extraClusterKeyVaultAccessPolicies,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"databaseAccountName": {
						Value: databaseAccountName,
					},
					"extraClusterKeyvaultAccessPolicies": {
						Value: extraClusterKeyVaultAccessPolicies,
					},
					"adminApiCaBundle": {
						Value: adminApiCaBundle,
					},
					"fullDeploy": {
						Value: false,
					},
				},
			},
		},
		{
			name: "skip nil values",
			ps: map[string]interface{}{
				"adminApiCaBundle":                   nil,
				"databaseAccountName":                nil,
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{
				DatabaseAccountName: &databaseAccountName,
			},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"databaseAccountName": {
						Value: databaseAccountName,
					},
					"fullDeploy": {
						Value: false,
					},
				},
			},
		},
		{
			name: "nil slice",
			ps: map[string]interface{}{
				"extraClusterKeyvaultAccessPolicies": nil,
			},
			config: Configuration{},
			want: arm.Parameters{
				Parameters: map[string]*arm.ParametersParameter{
					"fullDeploy": {
						Value: false,
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := deployer{
				config: &RPConfig{Configuration: &tt.config},
			}

			got := d.getParameters(tt.ps)

			if !reflect.DeepEqual(got, &tt.want) {
				t.Errorf("%#v", got)
			}

		})
	}
}
