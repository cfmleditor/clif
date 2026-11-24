package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func prepareCallHierarchy(t *testing.T, s *Server, docURI uri.URI, pos protocol.Position) []protocol.CallHierarchyItem {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentPrepareCallHierarchy, protocol.CallHierarchyPrepareParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     pos,
	})

	got, err := s.handlePrepareCallHierarchy(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	items, _ := got.([]protocol.CallHierarchyItem)

	return items
}

func incomingCalls(t *testing.T, s *Server, item *protocol.CallHierarchyItem) []protocol.CallHierarchyIncomingCall {
	t.Helper()

	raw := makeCall(t, protocol.MethodCallHierarchyIncomingCalls, protocol.CallHierarchyIncomingCallsParams{Item: *item})

	got, err := s.handleCallHierarchyIncomingCalls(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	calls, _ := got.([]protocol.CallHierarchyIncomingCall)

	return calls
}

func outgoingCalls(t *testing.T, s *Server, item *protocol.CallHierarchyItem) []protocol.CallHierarchyOutgoingCall {
	t.Helper()

	raw := makeCall(t, protocol.MethodCallHierarchyOutgoingCalls, protocol.CallHierarchyOutgoingCallsParams{Item: *item})

	got, err := s.handleCallHierarchyOutgoingCalls(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}

	calls, _ := got.([]protocol.CallHierarchyOutgoingCall)

	return calls
}

func TestCallHierarchyIsAdvertised(t *testing.T) {
	s := newTestServer()
	if p, _ := s.capabilities().CallHierarchyProvider.(protocol.Boolean); !bool(p) {
		t.Error("callHierarchyProvider is not advertised")
	}
}

func TestCallHierarchyIncomingAndOutgoingCalls(t *testing.T) {
	dir := t.TempDir()
	modelsDir := filepath.Join(dir, "models")

	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	daoPath := filepath.Join(modelsDir, "UserDAO.cfc")
	daoSrc := "component {\n\tfunction load(required numeric id) {\n\t\treturn {};\n\t}\n}\n"

	svcPath := filepath.Join(modelsDir, "UserService.cfc")
	svcSrc := `component {
	function getUser(required numeric id) {
		var dao = new models.UserDAO();
		return dao.load(arguments.id);
	}

	function handleRequest() {
		return getUser(42);
	}
}
`

	if err := os.WriteFile(daoPath, []byte(daoSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(svcPath, []byte(svcSrc), 0o600); err != nil {
		t.Fatal(err)
	}

	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	daoURI := uri.File(daoPath)
	svcURI := uri.File(svcPath)

	openDoc(t, s, daoURI, daoSrc)
	openDoc(t, s, svcURI, svcSrc)

	// Prepare on `getUser` declaration (line 1, col 12).
	prepared := prepareCallHierarchy(t, s, svcURI, protocol.Position{Line: 1, Character: 12})
	if len(prepared) != 1 || prepared[0].Name != "getUser" {
		t.Fatalf("prepare on getUser = %+v, want [getUser]", prepared)
	}

	// Incoming calls to getUser -> handleRequest.
	in := incomingCalls(t, s, &prepared[0])
	if len(in) != 1 || in[0].From.Name != "handleRequest" {
		t.Fatalf("incomingCalls(getUser) = %+v, want [handleRequest]", in)
	}

	// Outgoing calls from getUser -> UserDAO.load.
	out := outgoingCalls(t, s, &prepared[0])
	if len(out) != 1 || out[0].To.Name != "load" {
		t.Fatalf("outgoingCalls(getUser) = %+v, want [load]", out)
	}
}
