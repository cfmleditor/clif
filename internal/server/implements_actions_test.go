package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func refactorActionsAt(t *testing.T, s *Server, docURI uri.URI, line uint32) []protocol.CodeAction {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentCodeAction, protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Range: protocol.Range{
			Start: protocol.Position{Line: line, Character: 0},
			End:   protocol.Position{Line: line, Character: 0},
		},
		Context: protocol.CodeActionContext{
			Only: []protocol.CodeActionKind{protocol.CodeActionKindRefactor},
		},
	})

	got, err := s.handleCodeAction(context.Background(), raw)
	if err != nil {
		t.Fatalf("handleCodeAction error: %v", err)
	}

	actions, _ := got.([]protocol.CodeAction)

	return actions
}

func TestImplementsActionsScriptAndTransitiveParent(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	u := writeCFCs(t, s, dir, map[string]string{
		"IBase.cfc":  "interface {\n\tpublic numeric function area();\n}\n",
		"ISub.cfc":   "interface extends=\"IBase\" {\n\tpublic numeric function perimeter(required numeric scale);\n}\n",
		"Square.cfc": "component implements=\"ISub\" {\n}\n",
	})

	actions := refactorActionsAt(t, s, u["Square.cfc"], 0)
	if len(actions) != 3 {
		t.Fatalf("got %d actions, want 3 (bulk + perimeter + area): %+v", len(actions), actions)
	}

	if actions[0].Title != "Implement all missing interface methods" {
		t.Errorf("actions[0].Title = %q, want bulk title", actions[0].Title)
	}

	edits := actions[0].Edit.Changes[u["Square.cfc"]]
	if len(edits) != 1 {
		t.Fatalf("bulk edit count = %d, want 1", len(edits))
	}

	text := edits[0].NewText
	if !strings.Contains(text, "public numeric function perimeter( required numeric scale ) {") {
		t.Errorf("bulk stub missing perimeter signature:\n%s", text)
	}

	if !strings.Contains(text, "public numeric function area() {") {
		t.Errorf("bulk stub missing inherited IBase.area signature:\n%s", text)
	}
}

func TestImplementsActionsTagCFCAndInheritedExclusion(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	files := map[string]string{
		"IShape.cfc": "interface {\n\tpublic numeric function area();\n\tpublic string function label(required string prefix);\n}\n",
		"Parent.cfc": "component {\n\tpublic numeric function area() { return 1; }\n}\n",
		"Child.cfc":  "<cfcomponent extends=\"Parent\" implements=\"IShape\">\n</cfcomponent>\n",
	}

	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	s.indexWorkspace()

	childURI := cfpath.ToURI(filepath.Join(dir, "Child.cfc"))
	openDoc(t, s, childURI, files["Child.cfc"])

	// `area` is inherited from Parent, so only `label` should be offered (single action, no bulk).
	actions := refactorActionsAt(t, s, childURI, 0)
	if len(actions) != 1 {
		t.Fatalf("got %d actions, want 1 (label only): %+v", len(actions), actions)
	}

	if actions[0].Title != "Implement label() from IShape" {
		t.Errorf("title = %q, want Implement label() from IShape", actions[0].Title)
	}

	text := actions[0].Edit.Changes[childURI][0].NewText
	if !strings.Contains(text, `<cffunction name="label" access="public" returntype="string">`) ||
		!strings.Contains(text, `<cfargument name="prefix" type="string" required="true">`) {
		t.Errorf("tag stub mismatch:\n%s", text)
	}
}

func TestImplementsActionsOnlyOnHeaderOrCloser(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	u := writeCFCs(t, s, dir, map[string]string{
		"IShape.cfc": "interface {\n\tpublic numeric function area();\n}\n",
		"Square.cfc": "component implements=\"IShape\" {\n\tpublic void function helper() {}\n}\n",
	})

	// Line 1 is inside the body, not the header or closing brace.
	if actions := refactorActionsAt(t, s, u["Square.cfc"], 1); len(actions) != 0 {
		t.Errorf("expected no actions on body line, got %+v", actions)
	}

	// Line 2 is the closing brace.
	if actions := refactorActionsAt(t, s, u["Square.cfc"], 2); len(actions) != 1 {
		t.Errorf("expected 1 action on closing brace, got %+v", actions)
	}
}

func TestRangeMayHoldComponentClose(t *testing.T) {
	src := "component implements=\"I\" {\n\tfunction a() {\n\t\treturn 1;\n\t}\n\n\tfunction b() { return 2; }\n}\n"
	// Lines: 0 header, 1 `function a() {`, 2 return, 3 `}`, 4 blank, 5 `function b() { ... }`, 6 `}`.
	cases := []struct {
		name       string
		start, end int
		want       bool
	}{
		{"a line with no closing token", 2, 2, false},
		{"a blank line", 4, 4, false},
		{"a line holding a brace", 3, 3, true},
		{"the closing line", 6, 6, true},
		{"a range that reaches a brace", 2, 3, true},
		{"a range past the text", 20, 20, false},
		{"an inverted range", 3, 1, true},
		{"a very wide range is not read", 0, 10_000, true},
	}

	for _, c := range cases {
		if got := rangeMayHoldComponentClose(src, c.start, c.end); got != c.want {
			t.Errorf("%s: lines %d-%d = %v, want %v", c.name, c.start, c.end, got, c.want)
		}
	}

	tags := "<cfcomponent implements=\"I\">\n\t<cffunction name=\"a\">\n\t</cffunction>\n</CFCOMPONENT>\n"
	if !rangeMayHoldComponentClose(tags, 3, 3) {
		t.Error("the closing </CFCOMPONENT> line is not found, whatever its case")
	}

	if rangeMayHoldComponentClose(tags, 1, 1) {
		t.Error("a line with an opening tag only was taken for the closer")
	}
}

// The closing-line prefilter only skips work: on every line of components
// written as script and as tags, the actions offered are exactly those of the
// declaration and the closing line, the places the unfiltered test chose.
func TestImplementsActionsOfferedOnlyOnTheDeclarationAndTheCloser(t *testing.T) {
	dir := t.TempDir()
	s := newTestServer()
	s.WorkspaceFolders = []string{dir}

	files := map[string]string{
		"IShape.cfc": "interface {\n\tpublic numeric function area();\n}\n",
		"Script.cfc": "component\n\timplements=\"IShape\"\n{\n\tfunction other() {\n\t\treturn { a = 1 };\n\t}\n\n\t// }\n}\n",
		"Tags.cfc":   "<cfcomponent\n\timplements=\"IShape\">\n\t<cffunction name=\"other\">\n\t</cffunction>\n\n</CFCOMPONENT>\n",
	}

	u := writeCFCs(t, s, dir, files)

	// Script.cfc: the declaration spans lines 0-2, the closing brace is line 8.
	// Tags.cfc: the declaration spans lines 0-1, the closing tag is line 5.
	want := map[string]map[int]bool{
		"Script.cfc": {0: true, 1: true, 2: true, 8: true},
		"Tags.cfc":   {0: true, 1: true, 5: true},
	}

	for name, onLines := range want {
		for line := range strings.Count(files[name], "\n") + 1 {
			got := len(refactorActionsAt(t, s, u[name], uint32(line))) > 0
			if got != onLines[line] {
				t.Errorf("%s line %d: actions offered = %v, want %v", name, line, got, onLines[line])
			}
		}
	}
}

// cursorInDeclarationBefore is cursorInDeclaration as it was written first,
// from the outline of the whole component: the oracle for the bounded one.
func cursorInDeclarationBefore(content string, line int) bool {
	openOff, _, found := parser.ComponentDeclarationBounds(content)
	if !found {
		return false
	}

	lines := parser.NewLineTable(content)
	first := lines.LineOf(openOff)

	if header, ok := parser.ComponentOutline(content); ok && header.NameStart <= openOff {
		first = lines.LineOf(header.NameStart)
	}

	return line >= first && line <= lines.LineOf(openOff)
}

func TestCursorInDeclarationAgreesWithTheOutlineOnEveryLine(t *testing.T) {
	texts := map[string]string{
		"one line":          "component extends=\"A\" {\n\tfunction f() {}\n}\n",
		"split script":      "component\n\textends=\"A\"\n\timplements=\"I\"\n{\n\tfunction f() {}\n}\n",
		"after a comment":   "/**\n * Docs.\n */\ncomponent {\n\tfunction f() {}\n}\n",
		"interface":         "interface extends=\"I\" {\n\tpublic void function f();\n}\n",
		"tag":               "<cfcomponent\n\timplements=\"I\">\n\t<cffunction name=\"f\">\n\t</cffunction>\n</cfcomponent>\n",
		"tag after a set":   "<cfset x = {a = 1}>\n<cfcomponent output=\"false\">\n</cfcomponent>\n",
		"crlf":              "component\r\n\timplements=\"I\"\r\n{\r\n\tfunction f() {}\r\n}\r\n",
		"no declaration":    "<cfset x = 1>\n<cfoutput>#x#</cfoutput>\n",
		"script page":       "<cfscript>\n\tx = 1;\n</cfscript>\n",
		"empty":             "",
		"keyword as member": "struct.component = 1;\n",
	}

	for name, src := range texts {
		for line := range strings.Count(src, "\n") + 3 {
			if got, want := cursorInDeclaration(src, line), cursorInDeclarationBefore(src, line); got != want {
				t.Errorf("%s, line %d: cursorInDeclaration = %v, the outline's answer is %v", name, line, got, want)
			}
		}
	}
}

func TestInterfaceStubsOfAMethodThatReturnsAValueThrow(t *testing.T) {
	arg := []parser.Argument{{Name: "scale", Type: "numeric", Required: true}}

	for _, c := range []struct {
		returnType string
		throws     bool
	}{
		{"numeric", true},
		{"string", true},
		{"models.User", true},
		{"Struct", true},
		{"void", false},
		{"VOID", false},
		{"any", false},
		{" Any ", false},
		{"", false},
	} {
		fn := &parser.FunctionDef{Name: "area", ReturnType: c.returnType, Arguments: arg}

		script := formatScriptMethodStub(fn)
		tag := formatTagMethodStub(fn)

		const (
			scriptThrow = `throw( type = "NotImplemented", message = "area() is not implemented" );`
			tagThrow    = `<cfthrow type="NotImplemented" message="area() is not implemented">`
		)

		if got := strings.Contains(script, scriptThrow); got != c.throws {
			t.Errorf("return type %q: script stub throws = %v, want %v:\n%s", c.returnType, got, c.throws, script)
		}

		if got := strings.Contains(tag, tagThrow); got != c.throws {
			t.Errorf("return type %q: tag stub throws = %v, want %v:\n%s", c.returnType, got, c.throws, tag)
		}

		if !strings.Contains(script, "// TODO: implement") {
			t.Errorf("return type %q: script stub lost its TODO:\n%s", c.returnType, script)
		}
	}
}

// What the action inserts is CFML the parser reads back as the method it stands
// for: a stub that does not parse would be worse than none.
func TestInterfaceStubsParseBackAsTheMethod(t *testing.T) {
	fn := &parser.FunctionDef{Name: "area", ReturnType: "numeric", Arguments: []parser.Argument{{Name: "scale", Type: "numeric", Required: true}}}

	script := parser.Parse("file:///w/S.cfc", "component {\n"+formatScriptMethodStub(fn)+"\n}\n")
	if len(script.Funcs) != 1 || script.Funcs[0].Name != "area" || script.Funcs[0].ReturnType != "numeric" {
		t.Errorf("script stub parses as %+v, want area returning numeric", script.Funcs)
	}

	tag := parser.Parse("file:///w/T.cfc", "<cfcomponent>\n"+formatTagMethodStub(fn)+"\n</cfcomponent>\n")
	if len(tag.Funcs) != 1 || tag.Funcs[0].Name != "area" || tag.Funcs[0].ReturnType != "numeric" {
		t.Errorf("tag stub parses as %+v, want area returning numeric", tag.Funcs)
	}
}
