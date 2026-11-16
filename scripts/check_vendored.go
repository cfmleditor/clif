//go:build ignore
// +build ignore

// Checks and hashes a directory of doc source JSON, for scripts/fetch-docs-lucee.sh:
//
//	go run scripts/check_vendored.go check <dir>      exit 1, listing them, if any *.json is broken
//	go run scripts/check_vendored.go hash <dir>       print the manifest hash of its *.json
//	go run scripts/check_vendored.go checkhash <dir>  both, in one pass: the hash only if none is broken
//
// The rules live in internal/docs/sources (CheckEntryFile, ManifestHash), where
// they are tested; this only applies them to files on disk.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	docsources "github.com/cfmleditor/clif/internal/docs/sources"
)

func main() {
	mode := ""
	if len(os.Args) == 3 {
		mode = os.Args[1]
	}
	if mode != "check" && mode != "hash" && mode != "checkhash" {
		fmt.Fprintln(os.Stderr, "usage: check_vendored.go check|hash|checkhash <dir>")
		os.Exit(2)
	}

	paths, err := filepath.Glob(filepath.Join(os.Args[2], "*.json"))
	if err != nil || len(paths) == 0 {
		fmt.Fprintf(os.Stderr, "no *.json in %s\n", os.Args[2])
		os.Exit(1)
	}

	files := make(map[string][]byte, len(paths))
	var broken []string
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			broken = append(broken, err.Error())

			continue
		}
		files[filepath.Base(p)] = b
		if err := docsources.CheckEntryFile(b); err != nil {
			broken = append(broken, fmt.Sprintf("%s: %v", filepath.Base(p), err))
		}
	}

	if mode != "hash" {
		for _, b := range broken {
			fmt.Fprintln(os.Stderr, "  "+b)
		}
		if len(broken) > 0 {
			os.Exit(1)
		}
	}
	if mode != "check" {
		fmt.Println(docsources.ManifestHash(files))
	}
}
