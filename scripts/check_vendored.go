//go:build ignore
// +build ignore

// Checks and hashes a directory of doc source JSON, for scripts/fetch-docs-lucee.sh:
//
//	go run scripts/check_vendored.go checkhash <dir>
//
// prints the manifest hash of its *.json, or lists the broken ones and exits 1.
// The rules live in internal/docs/sources (HashDir), where they are tested and
// shared with TestVendoredLuceeMatchesItsSource.
package main

import (
	"fmt"
	"os"

	docsources "github.com/cfmleditor/clif/internal/docs/sources"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "checkhash" {
		fmt.Fprintln(os.Stderr, "usage: check_vendored.go checkhash <dir>")
		os.Exit(2)
	}

	hash, unhashed, invalid, err := docsources.HashDir(os.Args[2])
	broken := append(unhashed, invalid...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(broken) > 0 {
		for _, b := range broken {
			fmt.Fprintln(os.Stderr, "  "+b)
		}
		os.Exit(1)
	}

	fmt.Println(hash)
}
