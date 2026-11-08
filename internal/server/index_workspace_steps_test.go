package server

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.lsp.dev/uri"
)

func writeIndexFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	for name, content := range map[string]string{
		"A.cfc":     "component { function a() {} }",
		"sub/B.cfc": "component { function b() {} }",
		"page.cfm":  "<cfoutput>x</cfoutput>",
		"notes.txt": "not cfml",
	} {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func baseNames(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, filepath.Base(p))
	}

	slices.Sort(out)

	return out
}

// The components an indexing run reads come from the workspace folders, or from
// the globs when the config lists them, or from the editor's roots when it
// names no folder: and only components, never a template or other file.
func TestTheFilesIndexedComeFromTheFoldersTheGlobsOrTheEditorRoots(t *testing.T) {
	dir := writeIndexFixture(t)

	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	files, source := s.workspaceCFCFiles()
	if source != "workspaceFolders" || !slices.Equal(baseNames(files), []string{"A.cfc", "B.cfc"}) {
		t.Errorf("folders: %q %v", source, baseNames(files))
	}

	s.IndexGlobs = []string{filepath.Join(dir, "sub", "*")}

	files, source = s.workspaceCFCFiles()
	if source != "globs" || !slices.Equal(baseNames(files), []string{"B.cfc"}) {
		t.Errorf("globs: %q %v", source, baseNames(files))
	}

	r := newTestServer()
	r.addWorkspaceRoot(dir)

	files, source = r.workspaceCFCFiles()
	if source != "editorRoots" || !slices.Equal(baseNames(files), []string{"A.cfc", "B.cfc"}) {
		t.Errorf("editor roots: %q %v", source, baseNames(files))
	}
}

// A file is parsed for the index unless the editor has it open (its buffer is
// the editor's to report), it cannot be read, or it is not text.
func TestAFileIsParsedForTheIndexUnlessItIsOpenUnreadableOrBinary(t *testing.T) {
	dir := writeIndexFixture(t)

	if err := os.WriteFile(filepath.Join(dir, "Bin.cfc"), []byte("component {\x00\x01\x02}"), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()

	pr, ok := s.parseForIndex(filepath.Join(dir, "A.cfc"))
	if !ok || pr == nil || len(pr.Funcs) != 1 || pr.Funcs[0].Name != "a" {
		t.Fatalf("a plain component: ok=%v pr=%v", ok, pr)
	}

	if _, ok := s.parseForIndex(filepath.Join(dir, "Missing.cfc")); ok {
		t.Error("a file that cannot be read was parsed")
	}

	if _, ok := s.parseForIndex(filepath.Join(dir, "Bin.cfc")); ok {
		t.Error("a binary file was parsed")
	}

	s.setDocument(uri.File(filepath.Join(dir, "A.cfc")), "component {}")

	if _, ok := s.parseForIndex(filepath.Join(dir, "A.cfc")); ok {
		t.Error("a file the editor has open was parsed from disk")
	}
}
