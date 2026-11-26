package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
	"go.lsp.dev/protocol"
)

func TestInitLinterNativeEngine(t *testing.T) {
	dir := t.TempDir()
	rc := `{"inheritParent": false, "includes": [{"code": "AVOID_USING_CFDUMP_TAG"}, {"code": "MISSING_VAR"}]}`

	if err := os.WriteFile(filepath.Join(dir, ".cflintrc"), []byte(rc), 0o600); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(dir, "sample.cfc")
	src := "component {\n\tfunction run() {\n\t\tx = 1;\n\t}\n}\n"

	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := newTestServer()
	srv.Linting = true
	srv.LintEngine = config.LintEngineNative
	srv.Version = "test"

	srv.initLinter()

	srv.mu.RLock()
	linter := srv.linter
	srv.mu.RUnlock()

	if linter == nil {
		t.Fatal("expected initLinter with native engine to set srv.linter without CFLint")
	}

	diags, err := linter.Scan(context.Background(), file)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(diags) != 1 {
		t.Fatalf("Scan diagnostics = %d, want 1 (%+v)", len(diags), diags)
	}

	if code, _ := diags[0].Code.(protocol.String); code != "MISSING_VAR" {
		t.Errorf("diagnostic code = %q, want MISSING_VAR", code)
	}

	byFile, err := linter.ScanFiles(context.Background(), []string{file})
	if err != nil {
		t.Fatalf("ScanFiles: %v", err)
	}

	if len(byFile[file]) != 1 {
		t.Fatalf("ScanFiles[%s] = %d, want 1", file, len(byFile[file]))
	}

	// Verify in-memory ScanContent lints unsaved editor buffers directly.
	cs, ok := linter.(contentLintScanner)
	if !ok {
		t.Fatal("expected native linter to implement contentLintScanner")
	}

	unsaved := []byte("component {\n\tfunction run() {\n\t\tvar x = 1;\n\t}\n}\n")

	liveDiags, err := cs.ScanContent(context.Background(), file, unsaved)
	if err != nil {
		t.Fatalf("ScanContent: %v", err)
	}

	if len(liveDiags) != 0 {
		t.Fatalf("ScanContent(fixed buffer) = %d, want 0 (%+v)", len(liveDiags), liveDiags)
	}
}
