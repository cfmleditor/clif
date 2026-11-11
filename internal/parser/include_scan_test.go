package parser

import (
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// The expressions the listing scanners replaced, kept as the reference the
// scanners must agree with.
var (
	refListingDir      = regexp.MustCompile(`(?i)^#\s*getDirectoryFromPath\s*\(\s*getCurrentTemplatePath\s*\(\s*\)\s*\)\s*#([\w./-]+?)/?$`)
	refListingIncluded = regexp.MustCompile(`(?i)^([\w./-]*)#\s*([\w$]+)\.name\s*#$`)
)

func TestListingScannersMatchTheirExpressions(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 6))
	pieces := []string{"#", " ", "\t", "getDirectoryFromPath", "GETCURRENTTEMPLATEPATH", "getCurrentTemplatePath", "(", ")", "/", "a", ".", "-", "_", "x y", "#", "", "1"}
	includePieces := []string{"#", " ", "\t", "q", "$", ".name", ".NAME", ".nam", "/", "up", ".", "-", "_", "'", "x"}

	for range 100000 {
		var dir, inc strings.Builder
		for range rng.IntN(12) {
			dir.WriteString(pieces[rng.IntN(len(pieces))])
		}

		for range rng.IntN(10) {
			inc.WriteString(includePieces[rng.IntN(len(includePieces))])
		}

		gotDir, ok := listingDir(dir.String())

		want := refListingDir.FindStringSubmatch(dir.String())
		if ok != (want != nil) || ok && gotDir != want[1] {
			t.Fatalf("listingDir(%q) = %q, %v; expression %v", dir.String(), gotDir, ok, want)
		}

		prefix, query, ok := listingIncludePath(inc.String())

		want = refListingIncluded.FindStringSubmatch(inc.String())
		if ok != (want != nil) || ok && (prefix != want[1] || query != want[2]) {
			t.Fatalf("listingIncludePath(%q) = %q, %q, %v; expression %v", inc.String(), prefix, query, ok, want)
		}
	}
}
