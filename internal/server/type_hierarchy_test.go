package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func prepareTypeHierarchy(t *testing.T, s *Server, docURI uri.URI, pos protocol.Position) []protocol.TypeHierarchyItem {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentPrepareTypeHierarchy, protocol.TypeHierarchyPrepareParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     pos,
	})

	got, err := s.handlePrepareTypeHierarchy(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	items, _ := got.([]protocol.TypeHierarchyItem)

	return items
}

func typeSupertypes(t *testing.T, s *Server, item *protocol.TypeHierarchyItem) []protocol.TypeHierarchyItem {
	t.Helper()

	raw := makeCall(t, protocol.MethodTypeHierarchySupertypes, protocol.TypeHierarchySupertypesParams{Item: *item})

	got, err := s.handleTypeHierarchySupertypes(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	items, _ := got.([]protocol.TypeHierarchyItem)

	return items
}

func typeSubtypes(t *testing.T, s *Server, item *protocol.TypeHierarchyItem) []protocol.TypeHierarchyItem {
	t.Helper()

	raw := makeCall(t, protocol.MethodTypeHierarchySubtypes, protocol.TypeHierarchySubtypesParams{Item: *item})

	got, err := s.handleTypeHierarchySubtypes(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	items, _ := got.([]protocol.TypeHierarchyItem)

	return items
}

func TestTypeHierarchyIsAdvertised(t *testing.T) {
	s := newTestServer()
	if p, _ := s.capabilities().TypeHierarchyProvider.(protocol.Boolean); !bool(p) {
		t.Error("typeHierarchyProvider is not advertised")
	}
}

func TestTypeHierarchyResolvesSupertypesAndSubtypes(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, "models")

	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(modelsDir, "Base.cfc"):  "component {\n\tfunction id() {}\n}\n",
		filepath.Join(modelsDir, "User.cfc"):  "component extends=\"models.Base\" {\n\tfunction name() {}\n}\n",
		filepath.Join(modelsDir, "Admin.cfc"): "component extends=\"models.User\" {\n\tfunction role() {}\n}\n",
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	for path, src := range files {
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}

		openDoc(t, s, uri.File(path), src)
	}

	userURI := uri.File(filepath.Join(modelsDir, "User.cfc"))

	// Prepare on User.cfc itself.
	prepared := prepareTypeHierarchy(t, s, userURI, protocol.Position{Line: 0, Character: 0})
	if len(prepared) != 1 || prepared[0].Name != "User" {
		t.Fatalf("prepare on User.cfc = %+v, want User", prepared)
	}

	// Supertypes of User -> Base.
	supers := typeSupertypes(t, s, &prepared[0])
	if len(supers) != 1 || supers[0].Name != "Base" {
		t.Fatalf("supertypes(User) = %+v, want [Base]", supers)
	}

	// Supertypes of Base -> none.
	if rootSupers := typeSupertypes(t, s, &supers[0]); len(rootSupers) != 0 {
		t.Errorf("supertypes(Base) = %+v, want empty", rootSupers)
	}

	// Subtypes of User -> Admin.
	subs := typeSubtypes(t, s, &prepared[0])
	if len(subs) != 1 || subs[0].Name != "Admin" {
		t.Fatalf("subtypes(User) = %+v, want [Admin]", subs)
	}

	// Subtypes of Base -> User.
	baseSubs := typeSubtypes(t, s, &supers[0])
	if len(baseSubs) != 1 || baseSubs[0].Name != "User" {
		t.Fatalf("subtypes(Base) = %+v, want [User]", baseSubs)
	}

	// Prepare with cursor on `models.Base` inside `extends="models.Base"` -> Base.
	onExtends := prepareTypeHierarchy(t, s, userURI, protocol.Position{Line: 0, Character: 25})
	if len(onExtends) != 1 || onExtends[0].Name != "Base" {
		t.Fatalf("prepare on extends=\"models.Base\" = %+v, want [Base]", onExtends)
	}
}
