// +build tools

// tools is a dummy package that will be ignored for builds, but included for dependencies

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
package tools

import (
	_ "github.com/AlekSi/gocov-xml"
	_ "github.com/alvaroloes/enumer"
	_ "github.com/axw/gocov/gocov"
	_ "github.com/go-bindata/go-bindata/go-bindata"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/jim-minter/go-cosmosdb/cmd/gencosmosdb"
	_ "github.com/jstemmer/go-junit-report"
	_ "github.com/onsi/ginkgo"
	_ "github.com/onsi/gomega"
	_ "golang.org/x/tools/cmd/goimports"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
