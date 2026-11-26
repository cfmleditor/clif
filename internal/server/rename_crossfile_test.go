package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/cfmleditor/clif/internal/config"
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
	rngs := callNameRangesOnLine([]string{line}, 0, "fetchUser", "svc")

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

// A rename must not reach into code the project does not own. A handler's
// override of a framework method has the bundled stub in its family, a virtual
// file: the edit would name a file that is not there, and the framework calls
// the method by its name.
func TestCrossFileRenameRefusesFrameworkStubs(t *testing.T) {
	dir := t.TempDir()
	src := "component extends=\"coldbox.system.EventHandler\" {\n\tfunction onHandlerDIComplete() { return; }\n}\n"
	path := filepath.Join(dir, "Main.cfc")

	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.Features.Rename = true
	s.Features.RenameCrossFile = true
	s.Frameworks = []string{"coldbox"}
	s.ComponentResolvers = config.FrameworkResolvers(s.Frameworks)
	s.WorkspaceFolders = []string{dir}
	s.indexWorkspace()

	docURI := cfpath.ToURI(path)
	openDoc(t, s, docURI, src)

	edit, err := renameTo(t, s, docURI, 1, 12, "afterDI")
	if err == nil || !strings.Contains(err.Error(), "framework code (EventHandler.cfc)") {
		t.Fatalf("rename of a framework override: err = %v, want a refusal naming EventHandler.cfc", err)
	}

	if edit != nil {
		t.Errorf("a refused rename returned edits: %+v", edit.Changes)
	}
}

// The workspace search cannot see the callers of a method declared outside it,
// so a family reaching out of the workspace is refused, not half renamed.
func TestCrossFileRenameRefusesFamilyOutsideTheWorkspace(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "app")
	lib := filepath.Join(root, "lib")

	for _, d := range []string{app, lib} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	baseSrc := "component {\n\tfunction save() { return; }\n}\n"
	childSrc := "component extends=\"lib.Base\" {\n\tfunction save() { return; }\n}\n"

	if err := os.WriteFile(filepath.Join(lib, "Base.cfc"), []byte(baseSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	childPath := filepath.Join(app, "Child.cfc")
	if err := os.WriteFile(childPath, []byte(childSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.Features.Rename = true
	s.Features.RenameCrossFile = true
	s.Mappings = map[string]string{"lib": lib}
	s.WorkspaceFolders = []string{app}
	s.indexWorkspace()

	childURI := cfpath.ToURI(childPath)
	openDoc(t, s, childURI, childSrc)

	if _, err := renameTo(t, s, childURI, 1, 12, "persist"); err == nil || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("rename with a family member outside the workspace: err = %v, want a refusal", err)
	}
}

// The columns of the calls on a line run on from one to the next, and are the
// same as when each was measured from the line's start, with wide characters
// (the emoji is two UTF-16 units) between them.
func TestCallNameRangesOnALineKeepExactColumns(t *testing.T) {
	line := `é svc.foo(); 😀 svc.foo(); x = svc.foo( 日本 ) ;`

	rngs := callNameRangesOnLine([]string{line}, 0, "foo", "svc")
	if len(rngs) != 3 {
		t.Fatalf("got %d calls, want 3: %+v", len(rngs), rngs)
	}

	col := func(prefix string) uint32 { return uint32(len(utf16.Encode([]rune(prefix)))) }

	for i, before := range []string{"é svc.", "é svc.foo(); 😀 svc.", "é svc.foo(); 😀 svc.foo(); x = svc."} {
		if rngs[i].Start.Character != col(before) || rngs[i].End.Character != col(before)+3 {
			t.Errorf("call %d spans %d-%d, want %d-%d", i, rngs[i].Start.Character, rngs[i].End.Character, col(before), col(before)+3)
		}
	}
}

// Calls to one name written on one very long line (16,000 of them took 2.2 s
// when each column was measured from the line's start).
func TestCallNameRangesOnALongLineAreLinear(t *testing.T) {
	line := strings.Repeat("svc.foo(); ", 16000)

	start := time.Now()
	rngs := callNameRangesOnLine([]string{line}, 0, "foo", "svc")

	if took := time.Since(start); took > time.Second {
		t.Errorf("16,000 calls on one line took %v, want well under a second", took)
	}

	if len(rngs) != 16000 {
		t.Errorf("found %d calls, want 16000", len(rngs))
	}
}

// A caller file is read once for a rename, not once for each call site in it:
// with a call to rename on each of 300 lines, the file was read for every one
// of them and split into lines again each time (313 ms for 4,000 calls, which
// grew with the square of the file), and a line with two calls was answered
// twice. The reads are counted, not timed, so the test cannot flake.
func TestCrossFileRenameReadsACallerOnce(t *testing.T) {
	dir := t.TempDir()
	svcSrc := "component {\n\tpublic void function foo() {}\n}\n"

	if err := os.WriteFile(filepath.Join(dir, "Svc.cfc"), []byte(svcSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	page := "<cfscript>\nsvc = new Svc();\n" + strings.Repeat("svc.foo(1);\n", 300) + "svc.foo(1); svc.foo(2);\n</cfscript>\n"
	if err := os.WriteFile(filepath.Join(dir, "page.cfm"), []byte(page), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.Features.Rename = true
	s.Features.RenameCrossFile = true
	s.WorkspaceFolders = []string{dir}
	s.indexWorkspace()

	counter := &countingReadFS{FS: s.FS}
	s.FS = counter

	svcURI := cfpath.ToURI(filepath.Join(dir, "Svc.cfc"))
	openDoc(t, s, svcURI, svcSrc)
	counter.reads.Store(0)

	edit, err := renameTo(t, s, svcURI, 1, 25, "bar")
	if err != nil {
		t.Fatalf("rename: %v", err)
	}

	// The declaration, 300 calls on their own lines, and the two on the last.
	pageEdits := edit.Changes[cfpath.ToURI(filepath.Join(dir, "page.cfm"))]
	if len(pageEdits) != 302 {
		t.Errorf("page.cfm has %d edits, want 302 (one per call, none twice)", len(pageEdits))
	}

	if reads := counter.reads.Load(); reads > 40 {
		t.Errorf("a rename of 302 calls made %d reads, want a few: the caller is read once", reads)
	}
}
