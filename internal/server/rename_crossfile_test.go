package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestCrossFileRenameOnByDefaultAndCanBeDisabled(t *testing.T) {
	s := newTestServer()
	if !s.Features.RenameCrossFile {
		t.Fatal("expected RenameCrossFile to be on by default")
	}

	s.Features.RenameCrossFile = false

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

func TestCrossFileRenameCoordinatesAcrossImplementsAndExtends(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"IBase.cfc":  "interface {\n\tpublic numeric function area();\n}\n",
		"ISub.cfc":   "interface extends=\"IBase\" {\n\tpublic numeric function perimeter();\n}\n",
		"Square.cfc": "component implements=\"ISub\" {\n\tpublic numeric function area() { return 1; }\n\tpublic numeric function perimeter() { return 4; }\n}\n",
		"Circle.cfc": "component implements=\"IBase\" {\n\tpublic numeric function area() { return 3; }\n}\n",
		"Cube.cfc":   "component extends=\"Square\" {\n\tpublic numeric function area() { return 6; }\n}\n",
		"Other.cfc":  "component {\n\tpublic numeric function area() { return 0; }\n}\n",
		"Caller.cfc": "component {\n\tpublic numeric function run() {\n\t\tvar s = new Square();\n\t\tvar o = new Other();\n\t\treturn s.area() + o.area();\n\t}\n}\n",
	}

	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.indexWorkspace()

	baseURI := cfpath.ToURI(filepath.Join(dir, "IBase.cfc"))
	squareURI := cfpath.ToURI(filepath.Join(dir, "Square.cfc"))
	circleURI := cfpath.ToURI(filepath.Join(dir, "Circle.cfc"))
	cubeURI := cfpath.ToURI(filepath.Join(dir, "Cube.cfc"))
	otherURI := cfpath.ToURI(filepath.Join(dir, "Other.cfc"))
	callerURI := cfpath.ToURI(filepath.Join(dir, "Caller.cfc"))

	openDoc(t, s, squareURI, files["Square.cfc"])

	// Renaming Square.area must rename IBase, Square, Circle (sibling), Cube (subclass),
	// and Caller's `s.area()`, while leaving Other.cfc and `o.area()` untouched.
	edit, err := renameTo(t, s, squareURI, 1, 25, "size")
	if err != nil {
		t.Fatalf("hierarchy rename error: %v", err)
	}

	for _, want := range []struct {
		name string
		u    uri.URI
		n    int
	}{
		{"IBase", baseURI, 1},
		{"Square", squareURI, 1},
		{"Circle", circleURI, 1},
		{"Cube", cubeURI, 1},
		{"Caller", callerURI, 1},
	} {
		if got := edit.Changes[want.u]; len(got) != want.n {
			t.Errorf("%s edits = %+v, want %d", want.name, got, want.n)
		}
	}

	if got := edit.Changes[otherURI]; len(got) != 0 {
		t.Errorf("Other.cfc must not be edited, got %+v", got)
	}

	// Collision in any member of the hierarchy (e.g. perimeter already exists on Square/ISub) must refuse.
	if _, err := renameTo(t, s, squareURI, 1, 25, "perimeter"); err == nil || !strings.Contains(err.Error(), "already declares a method named 'perimeter'") {
		t.Errorf("rename colliding in hierarchy: err = %v, want collision error", err)
	}
}
