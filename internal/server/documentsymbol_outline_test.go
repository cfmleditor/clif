package server

import (
	"path/filepath"
	"testing"

	"go.lsp.dev/protocol"
)

func TestWorkflowDocumentSymbolOutlineScriptComponent(t *testing.T) {
	root := t.TempDir()
	client := startWorkflow(t, root, false)

	src := `component accessors="true" {
	property string email;
	property name="age" type="numeric";

	public User function init(required string email) {
		variables.email = arguments.email;
		return this;
	}

	public string function greet() {
		return variables.email;
	}
}
`
	doc := client.open(filepath.Join(root, "User.cfc"), src)

	var symbols []protocol.DocumentSymbol
	client.call(protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{TextDocument: doc}, &symbols)

	if len(symbols) != 1 {
		t.Fatalf("top-level symbols = %+v, want 1 class symbol", symbols)
	}

	cls := &symbols[0]
	assertOutlineInvariants(t, cls)

	if cls.Name != "User" || cls.Kind != protocol.SymbolKindClass {
		t.Fatalf("root = %+v, want Class User", *cls)
	}

	wantSel := conformanceDeclarationRange(t, src, `|component| accessors="true"`)
	if cls.SelectionRange != wantSel {
		t.Errorf("class SelectionRange = %+v, want %+v", cls.SelectionRange, wantSel)
	}

	// Synthetic accessors (getEmail/setEmail/getAge/setAge) must not appear.
	wantChildren := []struct {
		name        string
		kind        protocol.SymbolKind
		declaration string
	}{
		{"email", protocol.SymbolKindProperty, `property string |email|;`},
		{"age", protocol.SymbolKindProperty, `property name="|age|"`},
		{"init", protocol.SymbolKindConstructor, `public User function |init|(`},
		{"greet", protocol.SymbolKindMethod, `public string function |greet|(`},
	}

	if len(cls.Children) != len(wantChildren) {
		t.Fatalf("children = %+v, want %d entries", cls.Children, len(wantChildren))
	}

	for i, want := range wantChildren {
		got := &cls.Children[i]
		if got.Name != want.name || got.Kind != want.kind {
			t.Errorf("child[%d] = (%q, %v), want (%q, %v)", i, got.Name, got.Kind, want.name, want.kind)
		}

		wantRange := conformanceDeclarationRange(t, src, want.declaration)
		if got.SelectionRange != wantRange {
			t.Errorf("child[%d] %s SelectionRange = %+v, want %+v", i, got.Name, got.SelectionRange, wantRange)
		}
	}
}

func TestWorkflowDocumentSymbolOutlineTagComponent(t *testing.T) {
	root := t.TempDir()
	client := startWorkflow(t, root, false)

	src := `<cfcomponent accessors="true" initmethod="setup">
	<cfproperty name="title" type="string">
	<cffunction name="setup" access="public" returntype="Article">
		<cfreturn this>
	</cffunction>
	<cffunction name="render" access="public" returntype="string">
		<cfreturn variables.title>
	</cffunction>
</cfcomponent>
`
	doc := client.open(filepath.Join(root, "Article.cfc"), src)

	var symbols []protocol.DocumentSymbol
	client.call(protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{TextDocument: doc}, &symbols)

	if len(symbols) != 1 {
		t.Fatalf("top-level symbols = %+v, want 1 class symbol", symbols)
	}

	cls := &symbols[0]
	assertOutlineInvariants(t, cls)

	if cls.Name != "Article" || cls.Kind != protocol.SymbolKindClass {
		t.Fatalf("root = %+v, want Class Article", *cls)
	}

	if len(cls.Children) != 3 {
		t.Fatalf("children = %+v, want [title, setup, render]", cls.Children)
	}

	if cls.Children[0].Name != "title" || cls.Children[0].Kind != protocol.SymbolKindProperty {
		t.Errorf("child[0] = %+v, want Property title", cls.Children[0])
	}

	if cls.Children[1].Name != "setup" || cls.Children[1].Kind != protocol.SymbolKindConstructor {
		t.Errorf("child[1] = %+v, want Constructor setup", cls.Children[1])
	}

	if cls.Children[2].Name != "render" || cls.Children[2].Kind != protocol.SymbolKindMethod {
		t.Errorf("child[2] = %+v, want Method render", cls.Children[2])
	}
}

func TestWorkflowDocumentSymbolOutlineInterfacesAndTemplate(t *testing.T) {
	root := t.TempDir()
	client := startWorkflow(t, root, false)

	cases := []struct {
		name string
		file string
		src  string
		sel  string
	}{
		{
			name: "script interface keyword",
			file: "IRepository.cfc",
			src:  "interface {\n\tpublic any function findById(required numeric id);\n}\n",
			sel:  "|interface| {",
		},
		{
			name: "script component interface",
			file: "IFactory.cfc",
			src:  "component interface {\n\tboolean function canCreate(required any other);\n}\n",
			sel:  "|component| interface {",
		},
		{
			name: "tag interface",
			file: "ITagFactory.cfc",
			src:  "<cfinterface>\n\t<cffunction name=\"canCreate\" returntype=\"boolean\">\n\t</cffunction>\n</cfinterface>\n",
			sel:  "<|cfinterface|>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := client.open(filepath.Join(root, tc.file), tc.src)

			var symbols []protocol.DocumentSymbol
			client.call(protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{TextDocument: doc}, &symbols)

			if len(symbols) != 1 {
				t.Fatalf("symbols = %+v, want 1 interface symbol", symbols)
			}

			iface := &symbols[0]
			assertOutlineInvariants(t, iface)

			if iface.Kind != protocol.SymbolKindInterface {
				t.Errorf("kind = %v, want Interface", iface.Kind)
			}

			wantSel := conformanceDeclarationRange(t, tc.src, tc.sel)
			if iface.SelectionRange != wantSel {
				t.Errorf("SelectionRange = %+v, want %+v", iface.SelectionRange, wantSel)
			}

			if len(iface.Children) != 1 || iface.Children[0].Kind != protocol.SymbolKindMethod {
				t.Errorf("children = %+v, want 1 Method child", iface.Children)
			}
		})
	}

	t.Run("cfm template keeps flat user functions with exact ranges", func(t *testing.T) {
		src := "<cffunction name=\"helper\">\n\t<cfreturn 1>\n</cffunction>\n<cfset helper()>\n"
		doc := client.open(filepath.Join(root, "page.cfm"), src)

		var symbols []protocol.DocumentSymbol
		client.call(protocol.MethodTextDocumentDocumentSymbol, protocol.DocumentSymbolParams{TextDocument: doc}, &symbols)

		if len(symbols) != 1 {
			t.Fatalf("symbols = %+v, want 1 function symbol", symbols)
		}

		fn := &symbols[0]
		assertOutlineInvariants(t, fn)

		if fn.Name != "helper" || fn.Kind != protocol.SymbolKindFunction {
			t.Errorf("symbol = %+v, want Function helper", *fn)
		}

		wantSel := conformanceDeclarationRange(t, src, `name="|helper|"`)
		if fn.SelectionRange != wantSel {
			t.Errorf("SelectionRange = %+v, want %+v", fn.SelectionRange, wantSel)
		}
	})
}

func assertOutlineInvariants(t *testing.T, sym *protocol.DocumentSymbol) {
	t.Helper()

	if sym.SelectionRange.Start == sym.SelectionRange.End {
		t.Errorf("symbol %q has zero-width SelectionRange %+v", sym.Name, sym.SelectionRange)
	}

	if posBefore(sym.SelectionRange.Start, sym.Range.Start) || posBefore(sym.Range.End, sym.SelectionRange.End) {
		t.Errorf("symbol %q SelectionRange %+v is not contained in Range %+v", sym.Name, sym.SelectionRange, sym.Range)
	}

	for i := range sym.Children {
		assertOutlineInvariants(t, &sym.Children[i])
	}
}
