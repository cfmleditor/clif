package docs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/docs/sources"
)

// TestRetainedLuceeEntriesAreDocumented: docs.lucee.org stopped publishing
// Lucee's crypto extension functions (JwtSign, BCryptVerify, the TOTP, Cbor
// and Pem families, ...), and a regeneration without docs/retained dropped
// all of them. Every call to one then read as a call to a function nobody
// declared. Each file in docs/retained must reach the generated docs as
// itself: the same syntax and parameters, not another source's entry of the
// same name (cfdocs documents several with Adobe's signatures), which a bare
// "is it a builtin" check would accept.
func TestRetainedLuceeEntriesAreDocumented(t *testing.T) {
	gen := generatedSources(t)

	files, err := filepath.Glob(filepath.Join("..", "..", "docs", "retained", "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("no docs/retained/*.json files: the retained Lucee entries are gone; check docs/retained/ was added to git")
	}

	for _, f := range files {
		base := filepath.Base(f)

		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		var want struct {
			Name   string `json:"name"`
			Type   string `json:"type"`
			Syntax string `json:"syntax"`
			Params []struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		if err := json.Unmarshal(data, &want); err != nil || want.Name == "" {
			t.Errorf("%s: not a docs entry (%v)", base, err)

			continue
		}

		var (
			got *Entry
			ok  bool
		)

		switch want.Type {
		case "function":
			got, ok = LookupFunction(want.Name)
		case "tag":
			got, ok = LookupTag(want.Name)
		default:
			t.Errorf("%s: type %q, want function or tag", base, want.Type)

			continue
		}

		if !ok {
			t.Errorf("%s (%s) is not in the generated docs: regenerate them (make generate)", want.Name, base)

			continue
		}

		// A source ranked above docs/retained (Lucee) wins when it
		// publishes the name again, with whatever signature it now has: that
		// is the design working, not a failure, and the retained file has
		// done its job. One ranked below it (cfdocs) must never win. Ranks
		// come from the file's own order line, so no source is named here
		// but the retained one.
		src := gen.Entries[sources.Key(want.Type, want.Name)]
		switch rank, retained := sources.Rank(gen.Order, src), sources.Rank(gen.Order, sources.Retained); {
		case src == sources.Retained:
		case rank > retained:
			t.Logf("%s now documents %s; docs/retained/%s can be deleted", src, want.Name, base)

			continue
		default:
			t.Errorf("%s comes from %q, ranked below docs/retained: check SOURCES in scripts/assemble-docs.sh", want.Name, src)

			continue
		}

		gotParams := make([]string, 0, len(got.Params))
		for _, p := range got.Params {
			gotParams = append(gotParams, p.Name)
		}

		wantParams := make([]string, 0, len(want.Params))
		for _, p := range want.Params {
			wantParams = append(wantParams, p.Name)
		}

		if got.Syntax != want.Syntax || !slices.Equal(gotParams, wantParams) {
			t.Errorf("%s: generated docs have %q %v, docs/retained/%s has %q %v: regenerate (make generate)",
				want.Name, got.Syntax, gotParams, base, want.Syntax, wantParams)
		}
	}
}

// TestLuceeCryptoFunctionsKeepLuceeSignatures: cfdocs also documents the
// BCrypt, SCrypt and Argon2 functions, with Adobe's (plaintext, options) and
// (plaintext, hashedstring). When Lucee dropped its entries, cfdocs' took
// their place, so a Lucee user got completion for parameters Lucee does not
// take. The retained Lucee entries must outrank cfdocs'. Once Lucee
// documents one again, its own signature is the right one, whatever it is.
func TestLuceeCryptoFunctionsKeepLuceeSignatures(t *testing.T) {
	gen := generatedSources(t)
	retained := sources.Rank(gen.Order, sources.Retained)

	for name, want := range map[string][]string{
		"GenerateBCryptHash": {"input", "cost"},
		"VerifyBCryptHash":   {"input", "hash", "throwOnError"},
		"GenerateSCryptHash": {"input", "costParameter", "blockSize", "parallelization"},
		"VerifySCryptHash":   {"input", "hash", "throwOnError"},
		"BCryptVerify":       {"input", "hash", "throwOnError"},
		"Argon2CheckHash":    {"input", "hash"},
		"GenerateArgon2Hash": {"input", "variant", "parallelismFactor", "memoryCost", "iterations"},
	} {
		e, ok := LookupFunction(name)
		if !ok {
			t.Errorf("%s is not documented", name)

			continue
		}

		switch src := gen.Entries[sources.Key("function", name)]; {
		case sources.Rank(gen.Order, src) < retained:
			t.Errorf("%s comes from %s, ranked below docs/retained: Adobe's signature replaced Lucee's", name, src)

			continue
		case src != sources.Retained:
			continue // a higher source (Lucee) documents it again: its signature is right
		}

		got := make([]string, 0, len(e.Params))
		for _, p := range e.Params {
			got = append(got, p.Name)
		}

		if !slices.Equal(got, want) {
			t.Errorf("%s params = %v, want %v", name, got, want)
		}
	}
}

// TestGeneratedSourcesMatchTheDocs: generated_sources.txt is the baseline
// scripts/generate_docs.go compares a regeneration with, both for entries that
// went missing and for Lucee entries that fell back to cfdocs'. Committed
// without it, or out of step with generated_docs.go, the next regeneration
// checks against the wrong set, or none.
func TestGeneratedSourcesMatchTheDocs(t *testing.T) {
	listed := generatedSources(t).Entries

	documented := map[string]bool{}
	for _, e := range builtinEntries() {
		documented[sources.Key(e.Type, e.Name)] = true
	}

	for k := range documented {
		if listed[k] == "" {
			t.Errorf("%s is in generated_docs.go but not generated_sources.txt: regenerate both (make generate)", k)
		}
	}

	for k := range listed {
		if !documented[k] {
			t.Errorf("%s is in generated_sources.txt but not generated_docs.go: regenerate both (make generate)", k)
		}
	}
}

// generatedSources parses generated_sources.txt with the generator's own
// parser (internal/docs/sources), so the test and every build agree on what a
// valid file is.
func generatedSources(t *testing.T) sources.File {
	t.Helper()

	data, err := os.ReadFile("generated_sources.txt")
	if err != nil {
		t.Fatalf("%v: it is generated with generated_docs.go and committed with it; check it was added to git", err)
	}

	f, err := sources.Parse("generated_sources.txt", data)
	if err != nil {
		t.Fatal(err)
	}

	if f.Order == nil || sources.Rank(f.Order, sources.Retained) == 0 {
		t.Fatalf("generated_sources.txt: order line %v does not name %q: regenerate (make generate)", f.Order, sources.Retained)
	}

	return f
}

// TestLuceeOnlyEntriesArePresent: a regeneration without Lucee's docs (the
// vendored docs/lucee missing or emptied, or a bad `make docs-bump`) drops
// every Lucee-only entry, and one without docs/retained drops the crypto
// functions Lucee stopped publishing. The generator refuses both, but this is
// the net behind it (the release workflow runs this package's tests): two
// Lucee-only tags stand in for the Lucee docs, two retained functions for
// docs/retained.
func TestLuceeOnlyEntriesArePresent(t *testing.T) {
	for _, tag := range []string{"cfdistributedlock", "cfstatic"} {
		if _, ok := LookupTag(tag); !ok {
			t.Errorf("no %s tag: the docs were regenerated without Lucee's (check docs/lucee)", tag)
		}
	}

	for _, fn := range []string{"JwtSign", "BCryptVerify"} {
		if _, ok := LookupFunction(fn); !ok {
			t.Errorf("no %s function: the docs were regenerated without docs/retained", fn)
		}
	}
}

// TestVendoredLuceeMatchesItsSource: docs/lucee holds Lucee's docs as
// downloaded, and docs/lucee/SOURCE says which zip they came from and records
// a hash of the JSON vendored from it. A hand edit to a vendored file would
// regenerate consistently (CI's docs-reproducible job cannot see it) while
// SOURCE still claimed the zip's contents: this is what catches it. Change
// docs/lucee only through `make docs-bump`.
func TestVendoredLuceeMatchesItsSource(t *testing.T) {
	dir := filepath.Join("..", "..", "docs", "lucee")

	src, err := os.ReadFile(filepath.Join(dir, "SOURCE"))
	if err != nil {
		t.Fatalf("%v: check docs/lucee was added to git", err)
	}

	want := ""

	for _, line := range sources.Lines(src) {
		if k, v, ok := strings.Cut(line, " "); ok && k == "json-sha256" {
			want = v
		}
	}

	if want == "" {
		t.Fatal("docs/lucee/SOURCE records no json-sha256: rerun `make docs-bump`")
	}

	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no docs/lucee/*.json (%v): check docs/lucee was added to git", err)
	}

	files := make(map[string][]byte, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}

		files[filepath.Base(p)] = b
	}

	if got := sources.ManifestHash(files); got != want {
		t.Errorf("docs/lucee's JSON hashes to %s, but SOURCE records %s: a vendored file was edited by hand, "+
			"or added or removed; restore it (git checkout -- docs/lucee), or take Lucee's latest with `make docs-bump`", got, want)
	}
}
