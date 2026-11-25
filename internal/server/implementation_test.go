package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func implementationsAt(t *testing.T, s *Server, docURI uri.URI, line, char uint32) []protocol.Location {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentImplementation, protocol.ImplementationParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: line, Character: char},
	})

	got, err := s.handleImplementation(context.Background(), raw)
	if err != nil {
		t.Fatalf("handleImplementation error: %v", err)
	}

	locs, _ := got.([]protocol.Location)

	return locs
}

func TestImplementationProviderAdvertised(t *testing.T) {
	s := newTestServer()
	if p, _ := s.capabilities().ImplementationProvider.(protocol.Boolean); !bool(p) {
		t.Error("implementationProvider is not advertised")
	}
}

func TestImplementationComponentAndMethodOverrides(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "Animal.cfc")
	dogPath := filepath.Join(dir, "Dog.cfc")
	puppyPath := filepath.Join(dir, "Puppy.cfc")
	catPath := filepath.Join(dir, "Cat.cfc")

	baseSrc := "component {\n\tpublic string function speak() {\n\t\treturn \"\";\n\t}\n}\n"
	dogSrc := "component extends=\"Animal\" {\n\tpublic string function speak() {\n\t\treturn \"woof\";\n\t}\n}\n"
	puppySrc := "component extends=\"Dog\" {\n\tpublic string function play() {\n\t\treturn \"fetch\";\n\t}\n}\n"
	catSrc := "component extends=\"Animal\" {\n\tpublic string function speak() {\n\t\treturn \"meow\";\n\t}\n}\n"

	for p, src := range map[string]string{
		basePath:  baseSrc,
		dogPath:   dogSrc,
		puppyPath: puppySrc,
		catPath:   catSrc,
	} {
		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	baseURI := cfpath.ToURI(basePath)
	dogURI := cfpath.ToURI(dogPath)
	puppyURI := cfpath.ToURI(puppyPath)
	catURI := cfpath.ToURI(catPath)

	openDoc(t, s, baseURI, baseSrc)
	openDoc(t, s, dogURI, dogSrc)
	openDoc(t, s, puppyURI, puppySrc)
	openDoc(t, s, catURI, catSrc)

	// On component header (line 0, col 2): should return all 3 transitive subtypes (Cat, Dog, Puppy).
	compLocs := implementationsAt(t, s, baseURI, 0, 2)
	if len(compLocs) != 3 {
		t.Fatalf("component implementations = %+v, want 3 (Cat, Dog, Puppy)", compLocs)
	}

	// On `speak` method declaration (line 1, col 26): should return only Cat and Dog (Puppy does not override speak).
	methodLocs := implementationsAt(t, s, baseURI, 1, 26)
	if len(methodLocs) != 2 {
		t.Fatalf("speak implementations = %+v, want 2 (Cat, Dog)", methodLocs)
	}

	if methodLocs[0].URI != catURI || methodLocs[1].URI != dogURI {
		t.Errorf("speak implementation URIs = [%s, %s], want [%s, %s]", methodLocs[0].URI, methodLocs[1].URI, catURI, dogURI)
	}
}

func writeCFCs(t *testing.T, s *Server, dir string, files map[string]string) map[string]uri.URI {
	t.Helper()

	uris := map[string]uri.URI{}

	for name, src := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		uris[name] = cfpath.ToURI(p)
		openDoc(t, s, uris[name], src)
	}

	return uris
}

func TestImplementationInterfaceFindsImplementers(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	u := writeCFCs(t, s, dir, map[string]string{
		"IShape.cfc": "interface {\n\tpublic numeric function area();\n}\n",
		"Square.cfc": "component implements=\"IShape\" {\n\tpublic numeric function area() { return 1; }\n}\n",
		"Cube.cfc":   "component extends=\"Square\" {\n\tpublic numeric function area() { return 6; }\n}\n",
		"Other.cfc":  "component {\n\tpublic numeric function area() { return 0; }\n}\n",
	})

	locs := implementationsAt(t, s, u["IShape.cfc"], 1, 25)
	if len(locs) != 2 || locs[0].URI != u["Cube.cfc"] || locs[1].URI != u["Square.cfc"] {
		t.Fatalf("area implementations = %+v, want Cube and Square", locs)
	}

	if locs := implementationsAt(t, s, u["IShape.cfc"], 0, 2); len(locs) != 2 {
		t.Fatalf("interface header implementations = %+v, want 2", locs)
	}
}

func TestImplementationMultiLineHeader(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	u := writeCFCs(t, s, dir, map[string]string{
		"Base.cfc":  "component\n\taccessors=true\n{\n}\n",
		"Child.cfc": "component extends=\"Base\" {\n}\n",
	})

	for line := range uint32(3) {
		if locs := implementationsAt(t, s, u["Base.cfc"], line, 1); len(locs) != 1 {
			t.Errorf("line %d: implementations = %+v, want 1", line, locs)
		}
	}
}

func TestImplementationIgnoresUnresolvedSameNameExtends(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	u := writeCFCs(t, s, dir, map[string]string{
		"Base.cfc":      "component {\n}\n",
		"Real.cfc":      "component extends=\"Base\" {\n}\n",
		"Elsewhere.cfc": "component extends=\"vendor.missing.Base\" {\n}\n",
	})

	locs := implementationsAt(t, s, u["Base.cfc"], 0, 2)
	if len(locs) != 1 || locs[0].URI != u["Real.cfc"] {
		t.Fatalf("implementations = %+v, want only Real", locs)
	}
}

// A workspace scan indexes through applyIndexResult, not through an open
// document: the implementers it finds must be the same.
func TestImplementationAfterWorkspaceScan(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"IShape.cfc": "interface {\n\tpublic numeric function area();\n}\n",
		"Square.cfc": "component implements=\"IShape\" {\n\tpublic numeric function area() { return 1; }\n}\n",
	}

	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{dir}
	s.indexWorkspace()

	ifaceURI := cfpath.ToURI(filepath.Join(dir, "IShape.cfc"))
	openDoc(t, s, ifaceURI, files["IShape.cfc"])

	locs := implementationsAt(t, s, ifaceURI, 0, 2)
	if len(locs) != 1 || locs[0].URI != cfpath.ToURI(filepath.Join(dir, "Square.cfc")) {
		t.Fatalf("implementations = %+v, want Square only", locs)
	}
}

func TestImplementationSubInterfaceFindsImplementers(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	u := writeCFCs(t, s, dir, map[string]string{
		"IBase.cfc":  "interface {\n\tpublic numeric function area();\n}\n",
		"ISub.cfc":   "interface extends=\"IBase\" {\n\tpublic numeric function perimeter();\n}\n",
		"Square.cfc": "component implements=\"ISub\" {\n\tpublic numeric function area() { return 1; }\n\tpublic numeric function perimeter() { return 4; }\n}\n",
		"Cube.cfc":   "component extends=\"Square\" {\n\tpublic numeric function area() { return 6; }\n}\n",
		"Unrel.cfc":  "component {\n\tpublic numeric function area() { return 0; }\n}\n",
	})

	// On IBase.area: should find Cube and Square through ISub -> Square -> Cube.
	locs := implementationsAt(t, s, u["IBase.cfc"], 1, 25)
	if len(locs) != 2 || locs[0].URI != u["Cube.cfc"] || locs[1].URI != u["Square.cfc"] {
		t.Fatalf("IBase.area implementations = %+v, want Cube and Square", locs)
	}

	// On IBase header: should find ISub, Square, and Cube.
	headerLocs := implementationsAt(t, s, u["IBase.cfc"], 0, 2)
	if len(headerLocs) != 3 {
		t.Fatalf("IBase header implementations = %+v, want 3 (Cube, ISub, Square)", headerLocs)
	}
}
