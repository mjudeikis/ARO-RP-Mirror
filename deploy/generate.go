//go:generate go run ../hack/gendeploy
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -ignore=generate.go -o ../pkg/deploy/bindata.go .
//go:generate gofmt -s -l -w ../pkg/deploy/bindata.go

package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.
