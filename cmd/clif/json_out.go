package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// writeJSON is how every command prints a JSON result: two-space indent.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(v)
}

// writeJSONOrExit prints v to stdout, or reports the failure and exits.
func writeJSONOrExit(v any) {
	if err := writeJSON(os.Stdout, v); err != nil {
		fmt.Fprintf(os.Stderr, "writing JSON: %v\n", err)
		os.Exit(1)
	}
}
