package api

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"go/types"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"golang.org/x/tools/go/packages"
)

func TestMissingFields(t *testing.T) {
	pkgs, err := packages.Load(&packages.Config{Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo}, "github.com/Azure/ARO-RP/pkg/api")
	if err != nil {
		t.Error(err)
	}
	if len(pkgs) != 1 {
		t.Errorf("found %d packages, expected 1", len(pkgs))
	}

	for _, d := range pkgs[0].Types.Scope().Names() {
		if strings.HasSuffix(d, "Document") {
			spew.Dump("-------------------------------------------")
			err := finder(*pkgs[0], d)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func finder(pkg packages.Package, t string) error {
	var finder func(string, map[string]bool) error
	visited := make(map[string]bool)
	finder = func(t string, visited map[string]bool) error {
		spew.Dump("Visiting : " + t)
		visited[t] = true
		o := pkg.Types.Scope().Lookup(t)
		// TODO: Type case somewhere here
		if !o.Exported() {
			return nil
		}
		if s, ok := o.Type().Underlying().(*types.Struct); ok {
			if o.Name() != "MissingFields" && s.Field(0).Name() != "MissingFields" {
				return fmt.Errorf("missing fields not found in " + o.Name())
			}
			for f := 0; f < s.NumFields(); f++ {
				err := finder(s.Field(f).Name(), visited)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	return finder(t, visited)

}
