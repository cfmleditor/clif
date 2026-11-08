package server

import (
	"bytes"
	"testing"

	"go.lsp.dev/uri"
)

// A binary document opened by hand is parsed as empty, not as source.
func TestAnOpenedBinaryDocumentParsesAsEmpty(t *testing.T) {
	s := newTestServer()

	src := string(append(bytes.Repeat([]byte("a"), 100), 0)) + " function leaked() {}"

	pr := s.parseContent(uri.File("/tmp/blob.cfc"), src)
	if len(pr.Funcs) != 0 {
		t.Fatalf("a function was read from binary: %+v", pr.Funcs)
	}

	if pr := s.parseContent(uri.File("/tmp/ok.cfc"), "component { function real() {} }"); len(pr.Funcs) != 1 {
		t.Fatalf("a text component should still parse, got %+v", pr.Funcs)
	}
}
