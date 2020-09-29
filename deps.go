// +build tools

package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// tools is a dummy package that will be ignored for builds, but included for dependencies

import (
	_ "bitbucket.org/ww/goautoneg"                        // intermediate dep - investigate
	_ "github.com/AlekSi/gocov-xml"                       // used in test result publishing
	_ "github.com/alvaroloes/enumer"                      // used in enum type generation
	_ "github.com/axw/gocov/gocov"                        // used in test result publishing
	_ "github.com/go-bindata/go-bindata/go-bindata"       // used in static content generation
	_ "github.com/golang/mock/mockgen"                    // used in tests
	_ "github.com/jim-minter/go-cosmosdb/cmd/gencosmosdb" // used in database client generation
	_ "github.com/jstemmer/go-junit-report"               // used in test result publishing
	_ "github.com/onsi/ginkgo"                            // used in tests
	_ "github.com/onsi/gomega"                            // used in tests
	_ "golang.org/x/tools/cmd/goimports"                  // used in verify tests
	_ "k8s.io/code-generator/cmd/client-gen"              // used in operator code generation
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"   // used in operator code generation
)
