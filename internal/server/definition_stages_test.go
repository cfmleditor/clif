package server

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// openText opens content as a document of the testdata workspace.
func openText(srv *Server, name, content string) uri.URI {
	docURI := uri.URI("file://" + filepath.Join(testutil.Testdata(), name))
	srv.setDocument(docURI, content)

	return docURI
}

// col is the column of the first byte of needle in line, plus offset.
func col(line, needle string, offset int) uint32 {
	return uint32(strings.Index(line, needle) + offset)
}

// definitionAnswer asks the stages in a fixed order: a route, a resolver's
// argument, a component path, a file path, a cfinvoke component, a variable,
// then a qualified or unqualified function. These tests pin the stages whose
// answers the order decides and nothing else covered.

// A route attribute is a dotted string the word under the cursor would read
// as one fragment; the whole value is what is pointed at, and it resolves to
// the controller's method.
func TestARouteIsResolvedBeforeTheWordUnderTheCursor(t *testing.T) {
	srv := routeServer(t)
	srv.Mappings = map[string]string{"models": filepath.Join(testutil.Testdata(), "models")}

	line := `<a data-view="Base.getClassName">go</a>`
	docURI := openText(srv, "route.cfm", line)

	got := definitionAt(t, srv, docURI, 0, col(line, "getClassName", 2))
	loc := assertLocationFile(t, got, "Base.cfc")

	if loc.Range.Start.Line != 6 {
		t.Errorf("the route resolved to line %d of Base.cfc, want the line getClassName is declared on (6)", loc.Range.Start.Line)
	}
}

// A route that resolves to nothing leaves the cursor to the ordinary lookup
// rather than answering for it.
func TestARouteThatResolvesToNothingFallsThrough(t *testing.T) {
	srv := routeServer(t)
	srv.Mappings = map[string]string{"models": filepath.Join(testutil.Testdata(), "models")}

	line := `<a data-view="Nothing.here">go</a>`
	docURI := openText(srv, "route.cfm", line)

	if got := definitionAt(t, srv, docURI, 0, col(line, "here", 1)); got != nil {
		if locs, ok := got.([]protocol.Location); !ok || len(locs) != 0 {
			t.Errorf("an unresolved route answered %#v, want nothing", got)
		}
	}
}

// The argument of a call a componentResolver matches is the component it
// names: the cursor on "UserService" in getService( "UserService" ).
func TestAResolverArgumentIsTheComponentItNames(t *testing.T) {
	srv := newTestdataServer()

	line := `<cfset svc = getService("UserService")>`
	docURI := openText(srv, "arg.cfm", line)

	got := definitionAt(t, srv, docURI, 0, col(line, "UserService", 3))
	assertLocationFile(t, got, "UserService.cfc")
}

// A cursor inside a file path never falls through to the word under it: a
// template that does not exist is no reason to look for a function named like
// one of its segments.
func TestACursorInsideAnUnresolvedFilePathAnswersNothing(t *testing.T) {
	srv := newTestdataServer()

	line := `<cfinclude template="getClassName.cfm">`
	docURI := openText(srv, "inc.cfm", line)

	if got := definitionAt(t, srv, docURI, 0, col(line, "getClassName", 2)); got != nil {
		t.Errorf("an unresolved include answered %#v, want nothing", got)
	}
}
