package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Vendor tool helps to vendor in static non-go files into go mod project.
// Any file, which has go files inside is vendored using utils/tools pkg.

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

var (
	copyPatFlag = flag.String("copy", "", "copy copies all files in the given folder")
	verboseFlag = flag.Bool("v", false, "verbose output")
)

type Mod struct {
	ImportPath    string
	SourcePath    string
	Version       string
	SourceVersion string
	Dir           string          // full path, $GOPATH/pkg/mod/
	Pkgs          []string        // sub-pkg import paths
	VendorList    map[string]bool // files to vendor
}

func main() {
	log := utillog.GetLogger()
	flag.Parse()

	if err := run(log); err != nil {
		log.Fatal(err)
	}

}

func run(log *logrus.Entry) error {
	// Ensure go.mod file exists and we're running from the project root,
	// and that ./vendor/modules.txt file exists.
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(cwd, "go.mod")); os.IsNotExist(err) {
		return err
	}
	modtxtPath := filepath.Join(cwd, "vendor", "modules.txt")
	if _, err := os.Stat(modtxtPath); os.IsNotExist(err) {
		return err
	}

	// Prepare vendor copy patterns
	var copyPat []string
	if *copyPatFlag != "" {
		copyPat = strings.Split(strings.TrimSpace(*copyPatFlag), " ")
		if len(copyPat) == 0 {
			return err
		}
	}

	// Parse/process modules.txt file of pkgs
	f, _ := os.Open(modtxtPath)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var mod *Mod
	modules := []*Mod{}

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") {
			s := strings.Split(line, " ")
			if (len(s) != 6 && len(s) != 3) || s[1] == "explicit" { // skip lines we don't know about
				continue
			}

			mod = &Mod{
				ImportPath: s[1],
				Version:    s[2],
			}

			// Handle "replace" in module file if any
			if len(s) > 3 && s[3] == "=>" {
				mod.SourcePath = s[4]
				mod.SourceVersion = s[5]
				mod.Dir = pkgModPath(mod.SourcePath, mod.SourceVersion)
			} else {
				mod.Dir = pkgModPath(mod.ImportPath, mod.Version)
			}

			if _, err := os.Stat(mod.Dir); os.IsNotExist(err) {
				return err
			}

			// Build list of files to module path source to project vendor folder
			mod.VendorList = buildModVendorList(copyPat, mod)

			modules = append(modules, mod)

			continue
		}

		mod.Pkgs = append(mod.Pkgs, line)
	}

	// Filter out files not part of the mod.Pkgs
	for _, mod := range modules {
		if len(mod.VendorList) == 0 {
			continue
		}
		for vendorFile, _ := range mod.VendorList {
			for _, subpkg := range mod.Pkgs {
				path := filepath.Join(mod.Dir, importPathIntersect(mod.ImportPath, subpkg))

				x := strings.Index(vendorFile, path)
				if x == 0 {
					mod.VendorList[vendorFile] = true
				}
			}
		}
		for vendorFile, toggle := range mod.VendorList {
			if !toggle {
				delete(mod.VendorList, vendorFile)
			}
		}
	}

	// Copy mod vendor list files to ./vendor/
	for _, mod := range modules {
		for vendorFile := range mod.VendorList {
			x := strings.Index(vendorFile, mod.Dir)
			if x < 0 {
				return fmt.Errorf("vendor file doesn't belong to mod, strange")
			}

			localPath := fmt.Sprintf("%s%s", mod.ImportPath, vendorFile[len(mod.Dir):])
			localFile := fmt.Sprintf("./vendor/%s", localPath)

			log.Infof("vendoring %s\n", localPath)

			os.MkdirAll(filepath.Dir(localFile), os.ModePerm)
			if _, err := copyFile(vendorFile, localFile); err != nil {
				return fmt.Errorf("%s - unable to copy file %s", err.Error(), vendorFile)
			}
		}
	}
	return nil
}

func buildModVendorList(copyPat []string, mod *Mod) map[string]bool {
	vendorList := map[string]bool{}

	for _, pat := range copyPat {
		err := filepath.Walk(filepath.Join(mod.Dir, pat),
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				fileInfo, err := os.Stat(path)
				if err != nil {
					return err
				}

				if !fileInfo.IsDir() {
					vendorList[path] = false
				}
				return nil
			})
		if err != nil {
			// if we were not able to find package/version/pattern, just ignore
			//fmt.Errorf("Error! filepath.Walk match failure:", err)
		}
	}

	return vendorList
}

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

func importPathIntersect(basePath, pkgPath string) string {
	if strings.Index(pkgPath, basePath) != 0 {
		return ""
	}
	return pkgPath[len(basePath):]
}

func pkgModPath(importPath, version string) string {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		// the default GOPATH for go v1.11
		goPath = filepath.Join(os.Getenv("HOME"), "go")
	}

	var normPath string

	for _, char := range importPath {
		if unicode.IsUpper(char) {
			normPath += "!" + string(unicode.ToLower(char))
		} else {
			normPath += string(char)
		}
	}

	return filepath.Join(goPath, "pkg", "mod", fmt.Sprintf("%s@%s", normPath, version))
}

func copyFile(src, dst string) (int64, error) {
	srcStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !srcStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}
