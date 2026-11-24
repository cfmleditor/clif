package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

func TestCrossFileRenameOffByDefault(t *testing.T) {
	s := newTestServer()
	if s.Features.RenameCrossFile {
		t.Fatal("expected RenameCrossFile to be off by default")
	}

	s.Features.Rename = true

	dir := t.TempDir()
	svcPath := filepath.Join(dir, "UserService.cfc")
	svcSrc := "component {\n\tpublic string function getName() {\n\t\treturn \"a\";\n\t}\n}\n"

	if err := os.WriteFile(svcPath, []byte(svcSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	svcURI := cfpath.ToURI(svcPath)
	openDoc(t, s, svcURI, svcSrc)

	// Cursor on `getName` method declaration: with RenameCrossFile=false, prepareRename returns nil.
	got, err := prepareRename(t, s, svcURI, 1, 25)
	if err != nil {
		t.Fatalf("prepareRename error: %v", err)
	}

	if got != nil {
		t.Fatalf("expected nil when RenameCrossFile is false, got %#v", got)
	}
}

func TestCrossFileRenameComponentMethodAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, "models")

	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	svcPath := filepath.Join(modelsDir, "UserService.cfc")
	callerPath := filepath.Join(dir, "Consumer.cfc")

	svcSrc := strings.Join([]string{
		`component {`,
		`	public string function fetchUser(required numeric id) {`,
		`		return "";`,
		`	}`,
		`	public void function selfCaller() {`,
		`		fetchUser(1);`,
		`	}`,
		`}`,
	}, "\n")

	callerSrc := strings.Join([]string{
		`component {`,
		`	public void function run() {`,
		`		var svc = new models.UserService();`,
		`		svc.fetchUser(42);`,
		`	}`,
		`}`,
	}, "\n")

	if err := os.WriteFile(svcPath, []byte(svcSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(callerPath, []byte(callerSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.Features.Rename = true
	s.Features.RenameCrossFile = true
	s.WorkspaceFolders = []string{dir}

	svcURI := cfpath.ToURI(svcPath)
	callerURI := cfpath.ToURI(callerPath)

	openDoc(t, s, svcURI, svcSrc)
	s.index.IndexFile(callerURI, callerSrc)

	// prepareRename on `fetchUser` declaration (line 1, col 25).
	got, err := prepareRename(t, s, svcURI, 1, 25)
	if err != nil {
		t.Fatalf("prepareRename error: %v", err)
	}

	ph, ok := got.(protocol.PrepareRenamePlaceholder)
	if !ok || ph.Placeholder != "fetchUser" {
		t.Fatalf("prepareRename = %#v, want placeholder fetchUser", got)
	}

	// Perform cross-file rename to `loadUser`.
	edit, err := renameTo(t, s, svcURI, 1, 25, "loadUser")
	if err != nil {
		t.Fatalf("renameTo error: %v", err)
	}

	if edit == nil {
		t.Fatal("expected non-nil WorkspaceEdit")
	}

	svcEdits := edit.Changes[svcURI]
	if len(svcEdits) != 2 {
		t.Fatalf("svcEdits = %+v, want 2 (decl + selfCaller)", svcEdits)
	}

	callerEdits := edit.Changes[callerURI]
	if len(callerEdits) != 1 || callerEdits[0].Range.Start.Line != 3 || callerEdits[0].NewText != "loadUser" {
		t.Fatalf("callerEdits = %+v, want 1 edit on line 3", callerEdits)
	}

	// Adding an ambiguous/unresolved caller must cause rename to refuse safely.
	ambigPath := filepath.Join(dir, "Ambig.cfc")
	ambigSrc := "component {\n\tfunction f(x) { x.fetchUser(1); }\n}\n"

	if err := os.WriteFile(ambigPath, []byte(ambigSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := renameTo(t, s, svcURI, 1, 25, "loadUser"); err == nil || !strings.Contains(err.Error(), "unresolved/ambiguous") {
		t.Fatalf("expected refusal on ambiguous caller, got err=%v", err)
	}
}

func TestCrossFileRenameRefusesAmbiguousAndAccessorProperties(t *testing.T) {
	dir := t.TempDir()
	accPath := filepath.Join(dir, "Bean.cfc")
	accSrc := "component accessors=\"true\" {\n\tproperty name=\"email\" type=\"string\";\n}\n"

	if err := os.WriteFile(accPath, []byte(accSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.Features.Rename = true
	s.Features.RenameCrossFile = true
	s.WorkspaceFolders = []string{dir}

	accURI := cfpath.ToURI(accPath)
	openDoc(t, s, accURI, accSrc)

	if _, err := renameTo(t, s, accURI, 1, 17, "mail"); err == nil || !strings.Contains(err.Error(), "accessors=true") {
		t.Fatalf("expected refusal for accessors=true property, got err=%v", err)
	}

	// Plain property (no accessors=true) renames declaration + this/variables uses.
	plainPath := filepath.Join(dir, "Plain.cfc")
	plainSrc := strings.Join([]string{
		`component {`,
		`	property name="title" type="string";`,
		`	function show() {`,
		`		return variables.title & this.title;`,
		`	}`,
		`}`,
	}, "\n")

	if err := os.WriteFile(plainPath, []byte(plainSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	plainURI := cfpath.ToURI(plainPath)
	openDoc(t, s, plainURI, plainSrc)

	edit, err := renameTo(t, s, plainURI, 1, 17, "heading")
	if err != nil {
		t.Fatalf("plain property rename error: %v", err)
	}

	if edits := edit.Changes[plainURI]; len(edits) != 3 {
		t.Fatalf("plain property edits = %+v, want 3 (decl + variables.title + this.title)", edits)
	}

	// Shorthand `property user user;` must select the property name, not the type.
	sCol, eCol, ok := findPropNameCols("\tproperty user user;", "user")
	if !ok || sCol != 15 || eCol != 19 {
		t.Fatalf("findPropNameCols shorthand = (%d, %d, %v), want (15, 19, true)", sCol, eCol, ok)
	}

	// callNameRangesOnLine with receiver must only match that receiver's call.
	line := "var x = other.fetchUser(1) + svc.fetchUser(2);"
	rngs := callNameRangesOnLine(line, 0, "fetchUser", "svc")

	if len(rngs) != 1 || rngs[0].Start.Character != 33 {
		t.Fatalf("callNameRangesOnLine(svc) = %+v, want single match at col 33", rngs)
	}
}
