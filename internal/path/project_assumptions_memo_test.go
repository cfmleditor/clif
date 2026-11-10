package path

import (
	"os"
	"path/filepath"
	"testing"
)

// A source's mappings are kept, but every file the walk read is read again
// on reuse (a caller recording its reads sees them all), and an edit to any
// of them, here an included one, is noticed.
func TestAssumedSourceMappingsAreKeptUntilAReadChanges(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "Application.cfc")
	inc := filepath.Join(dir, "settings.cfm")

	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	write(app, `component { include "settings.cfm"; }`)
	write(inc, `<cfset this.mappings["/lib"] = "/a">`)

	var reads []string

	read := func(p string) ([]byte, error) {
		reads = append(reads, p)

		return os.ReadFile(p)
	}

	first := assumedSourceMappings(app, read)
	walked := len(reads)

	reads = nil
	again := assumedSourceMappings(app, read)

	if again["lib"] != first["lib"] || first["lib"] == "" || len(reads) != walked {
		t.Fatalf("reuse: %v (first %v), %d reads, want the %d the walk made", again, first, len(reads), walked)
	}

	write(inc, `<cfset this.mappings["/lib"] = "/b">`)

	if got := assumedSourceMappings(app, read)["lib"]; got == first["lib"] {
		t.Fatalf("after the include changed: %q, want the new mapping", got)
	}
}
