//go:build !wasip1

package mcp_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/codemap/mcp"
)

// Every tool, called through the wire as a client calls it: what it answers
// for good arguments, and the error it gives for missing ones.
func TestEveryToolAnswersAndRefusesMissingArguments(t *testing.T) {
	s := server(t, true)
	s.FindRefs = func(target string, _ []string, _ int) (any, error) { return map[string]any{"refs": target}, nil }
	s.Unresolved = func(paths []string, _ bool, _ int) (any, error) { return map[string]any{"scanned": paths}, nil }
	s.Lint = func(paths []string, _ int) (any, error) { return map[string]any{"linted": paths}, nil }

	for _, c := range []struct {
		call    string
		want    string
		isError bool
	}{
		{`{"name":"search_symbols","arguments":{"query":"getuser"}}`, "svc.cfc::getuser", false},
		{`{"name":"get_symbol","arguments":{"id":"svc.cfc::getuser"}}`, "getUser", false},
		{`{"name":"get_symbol","arguments":{}}`, "missing required argument", true},
		{`{"name":"get_callers","arguments":{"id":"svc.cfc::getuser"}}`, "page.cfm", false},
		{`{"name":"get_callers","arguments":{}}`, "missing required argument", true},
		{`{"name":"get_callees","arguments":{"id":"page.cfm"}}`, "svc.cfc::getuser", false},
		{`{"name":"get_callees","arguments":{}}`, "missing required argument", true},
		{`{"name":"find_path","arguments":{"from":"page.cfm","to":"svc.cfc::getuser"}}`, `"found": true`, false},
		{`{"name":"find_path","arguments":{"from":"svc.cfc::getuser","to":"page.cfm"}}`, `"found": false`, false},
		{`{"name":"find_path","arguments":{"from":"page.cfm"}}`, "missing required argument", true},
		{`{"name":"list_islands","arguments":{}}`, "islands", false},
		{`{"name":"list_orphans","arguments":{}}`, "orphans", false},
		{`{"name":"get_stats","arguments":{}}`, "callSites", false},
		{`{"name":"explain_call","arguments":{"file":"svc.cfc","line":4}}`, "traced svc.cfc", false},
		{`{"name":"explain_call","arguments":{"file":"svc.cfc"}}`, "missing required argument", true},
		{`{"name":"find_unresolved_calls","arguments":{"paths":["a"]}}`, "scanned", false},
		{`{"name":"find_references","arguments":{"target":"getUser","paths":["a"]}}`, "getUser", false},
		{`{"name":"find_references","arguments":{"target":"  ","paths":["a"]}}`, "missing required argument", true},
		{`{"name":"lint","arguments":{"paths":["a"]}}`, "linted", false},
		{`{"name":"lint","arguments":{}}`, "missing required argument", true},
		{`{"name":"nope","arguments":{}}`, "unknown tool", true},
		{`{"name":"get_stats","arguments":{"limit":"many"}}`, "bad arguments for get_stats", true},
	} {
		text, isError := callResult(t, s, c.call)
		if isError != c.isError || !strings.Contains(text, c.want) {
			t.Errorf("%s = %q (isError %v), want %q (isError %v)", c.call, text, isError, c.want, c.isError)
		}
	}
}

// A tool whose hook the server was started without says so, and a map tool on
// a server with no map says how to build one.
func TestAToolWithoutItsBackingIsRefusedWithTheRemedy(t *testing.T) {
	bare := &mcp.Server{Name: "test", Version: "0"}

	for _, c := range []struct{ call, want string }{
		{`{"name":"explain_call","arguments":{"file":"a","line":1}}`, "not available"},
		{`{"name":"find_unresolved_calls","arguments":{"paths":["a"]}}`, "not available"},
		{`{"name":"find_references","arguments":{"target":"f","paths":["a"]}}`, "not available"},
		{`{"name":"lint","arguments":{"paths":["a"]}}`, "--allow-lint"},
		{`{"name":"get_stats","arguments":{}}`, "clif graph --db"},
		{`{"name":"search_symbols","arguments":{"query":"x"}}`, "clif graph --db"},
	} {
		if text, isError := callResult(t, bare, c.call); !isError || !strings.Contains(text, c.want) {
			t.Errorf("%s = %q (isError %v), want an error containing %q", c.call, text, isError, c.want)
		}
	}
}

// A failing hook or store is a tool error carrying its message.
func TestAToolErrorCarriesTheFailure(t *testing.T) {
	s := server(t, true)
	s.Explain = func(string, int, string) (string, error) { return "", errors.New("no such file") }

	if text, isError := callResult(t, s, `{"name":"explain_call","arguments":{"file":"x","line":1}}`); !isError || !strings.Contains(text, "no such file") {
		t.Errorf("explain_call = %q (isError %v), want the hook's error", text, isError)
	}

	// Nodes the map does not hold have no path between them: an answer, not an error.
	if text, isError := callResult(t, s, `{"name":"find_path","arguments":{"from":"nope","to":"also-nope"}}`); isError || !strings.Contains(text, `"found": false`) {
		t.Errorf("find_path between unknown nodes = %q (isError %v), want found: false", text, isError)
	}
}

// A tool the server advertises has a handler: calling each with no arguments
// may be refused for them, never as an unknown tool.
func TestEveryAdvertisedToolIsRunnable(t *testing.T) {
	s := server(t, true)
	s.FindRefs = func(string, []string, int) (any, error) { return nil, nil }
	s.Unresolved = func([]string, bool, int) (any, error) { return nil, nil }
	s.Lint = func([]string, int) (any, error) { return nil, nil }

	names := listTools(t, s)
	if len(names) < 12 {
		t.Fatalf("only %d tools advertised: %v", len(names), names)
	}

	for _, name := range names {
		text, _ := callResult(t, s, `{"name":"`+name+`","arguments":{}}`)
		if strings.Contains(text, "unknown tool") {
			t.Errorf("%s is advertised and has no handler", name)
		}
	}
}
