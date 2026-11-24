package parser

import "strings"

// SplitList splits a comma-separated attribute value (`implements="A, b.C"`)
// into its trimmed, non-empty items.
func SplitList(value string) []string {
	var out []string

	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}

	return out
}
