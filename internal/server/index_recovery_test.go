package server

import (
	"testing"

	"go.lsp.dev/uri"
)

// A panic writing one parse result into the index is recorded and the file
// left out; it does not escape the goroutine draining the results, where it
// ended the language server.
func TestApplyingAnIndexResultSurvivesAPanic(t *testing.T) {
	s := newTestServer()

	// A nil parse result panics inside applyIndexResult.
	if s.applyIndexResultRecovered(uri.File("/x/bad.cfc"), "/x/bad.cfc", nil) {
		t.Fatal("a result that panicked was reported as indexed")
	}
}
