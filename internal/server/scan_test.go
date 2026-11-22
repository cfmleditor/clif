package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/language"
	"go.lsp.dev/protocol"
)

func TestExecuteCommandScanWorkspace(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "good.cfm"), []byte("<cfoutput>hello</cfoutput>"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "bad.cfm"), []byte("<cfoutput><cfif>unclosed"), 0o644)

	srv := newTestServer()
	srv.WorkspaceFolders = []string{dir}

	req := makeCall(t, protocol.MethodWorkspaceExecuteCommand, protocol.ExecuteCommandParams{
		Command: "clif.scanWorkspace",
	})

	if _, err := srv.handleExecuteCommand(context.Background(), req); err != nil {
		t.Fatal(err)
	}
}

func TestCollectErrorDiagnostics(t *testing.T) {
	src := []byte("<cfoutput><cfif>unclosed")

	tree := language.Parse(language.CFML, src, nil)
	defer tree.Close()

	if !tree.RootNode().HasError() {
		t.Skip("test source does not produce parse errors")
	}

	diags := collectErrorDiagnostics(tree.RootNode(), src)
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic")
	}

	for _, d := range diags {
		if d.Severity != protocol.DiagnosticSeverityError {
			t.Errorf("expected error severity, got %v", d.Severity)
		}

		if src, _ := d.Source.Get(); src != "cfmleditor" {
			t.Errorf("expected source 'cfmleditor', got %q", src)
		}
	}
}

func TestScanWorkspaceWritesParseErrorReport(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "good.cfm"), []byte("<cfoutput>hello</cfoutput>\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "bad.cfm"), []byte("<cfoutput>\n<cfif>unclosed\n</cfoutput>\n"), 0o644)

	srv := newTestServer()
	srv.WorkspaceFolders = []string{dir}
	srv.scanWorkspace(context.Background(), "")

	reportPath := filepath.Join(dir, defaultParseErrorReport)

	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected parse error report at %s: %v", reportPath, err)
	}

	text := string(data)
	for _, want := range []string{"# Parse Errors", "## `bad.cfm`", "bad.cfm:", "```cfml"} {
		if !strings.Contains(text, want) {
			t.Errorf("report missing %q:\n%s", want, text)
		}
	}

	if strings.Contains(text, "good.cfm") {
		t.Errorf("clean file good.cfm should not appear in report:\n%s", text)
	}
}
