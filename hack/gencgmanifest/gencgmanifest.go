//go:generate go run . -o ../../cgmanifest.json

package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var (
	outputFile = flag.String("o", "", "output file")
)

type cgmanifest struct {
	Registrations []*registration `json:"registration,omitempty"`
	Version       int             `json:"version,omitempty"`
}

type registration struct {
	Component *typedComponent `json:"component,omitempty"`
}

type typedComponent struct {
	Type string       `json:"type,omitempty"`
	Go   *goComponent `json:"go,omitempty"`
}

type goComponent struct {
	Version string `json:"version,omitempty"`
	Name    string `json:"name,omitempty"`
}

func run() error {
	file, err := os.Open("../../go.sum")
	if err != nil {
		return err
	}
	defer file.Close()

	deps := []goComponent{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), " ")
		deps = append(deps, goComponent{
			Name:    parts[0],
			Version: strings.Split(parts[1], "/")[0], //vX.Y.Z/go.mod to vX.Y.Z
		})
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	deps = unique(deps)
	cgmanifest := &cgmanifest{
		Registrations: make([]*registration, 0, len(deps)),
		Version:       1,
	}

	for _, dep := range deps {
		cgmanifest.Registrations = append(cgmanifest.Registrations, &registration{
			Component: &typedComponent{
				Type: "go",
				Go: &goComponent{
					Name:    dep.Name,
					Version: dep.Version,
				},
			},
		})
	}

	b, err := json.MarshalIndent(cgmanifest, "", "    ")
	if err != nil {
		return err
	}
	b = append(b, byte('\n'))

	if *outputFile != "" {
		err = ioutil.WriteFile(*outputFile, b, 0666)
	} else {
		_, err = fmt.Print(string(b))
	}

	return err
}

func unique(input []goComponent) []goComponent {
	keys := make(map[string]bool)
	output := []goComponent{}
	for _, entry := range input {
		if _, value := keys[entry.Version]; !value {
			keys[entry.Version] = true
			output = append(output, entry)
		}
	}
	return output
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		panic(err)
	}
}
