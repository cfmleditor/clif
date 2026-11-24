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
