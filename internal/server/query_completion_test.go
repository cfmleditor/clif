package server

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"

	"go.lsp.dev/protocol"
)

func dotCompletionLabels(t *testing.T, text string, line, char uint32) []string {
	t.Helper()

	return completionLabels(t, text, line, char, protocol.CompletionContext{
		TriggerKind:      protocol.CompletionTriggerKindTriggerCharacter,
		TriggerCharacter: new("."),
	})
}

func TestACfqueryOffersItsColumns(t *testing.T) {
	text := "<cfquery name=\"users\" datasource=\"d\">\nSELECT id, name AS title, count(*) FROM u\n</cfquery>\n<cfoutput>#users.</cfoutput>\n"

	got := dotCompletionLabels(t, text, 3, uint32(len("<cfoutput>#users.")))
	for _, want := range []string{"id", "title", "recordCount", "currentRow", "columnList"} {
		if !hasLabel(got, want) {
			t.Errorf("%q is not offered after `users.`: %q", want, got)
		}
	}

	if hasLabel(got, "name") {
		t.Error("the column the query aliases away is offered")
	}

	// Not in the middle of someone else's variable.
	if other := dotCompletionLabels(t, "<cfquery name=\"users\">SELECT id FROM u</cfquery>\n<cfoutput>#orders.</cfoutput>\n", 1, uint32(len("<cfoutput>#orders."))); hasLabel(other, "id") {
		t.Error("another variable is offered the query's columns")
	}
}

func TestQueryNewAndQueryExecuteOfferTheirColumns(t *testing.T) {
	text := "<cfscript>\nq = queryNew( \"sku,price\" );\nrs = queryExecute( \"SELECT id, label FROM t\" );\nq.\n</cfscript>\n"

	if got := dotCompletionLabels(t, text, 3, 2); !hasLabel(got, "sku") || !hasLabel(got, "price") {
		t.Errorf("queryNew columns missing: %q", got)
	}

	text = "<cfscript>\nq = queryNew( \"sku,price\" );\nrs = queryExecute( \"SELECT id, label FROM t\" );\nrs.\n</cfscript>\n"

	if got := dotCompletionLabels(t, text, 3, 3); !hasLabel(got, "id") || !hasLabel(got, "label") {
		t.Errorf("queryExecute columns missing: %q", got)
	}
}

func TestASelectStarOffersNoColumns(t *testing.T) {
	text := "<cfscript>\nrs = queryExecute( \"SELECT * FROM t\" );\nrs.\n</cfscript>\n"

	if items := queryColumnItems(text, "rs", 2, 3, nil); len(items) != 0 {
		t.Errorf("a query whose columns are not known was given %d items", len(items))
	}

	// What is known is offered with its kind, columns first.
	items := queryColumnItems("q = queryNew(\"a\");\nq.", "q", 1, 2, nil)
	if len(items) != 4 || items[0].Label != "a" || items[0].Kind != protocol.CompletionItemKindField || items[1].Kind != protocol.CompletionItemKindProperty {
		t.Errorf("items %+v", items)
	}
}

// A query is the variable of the function that declares it.
func TestQueryColumnsStayInTheirFunction(t *testing.T) {
	src := "component {\n function a() { var q = queryNew(\"id,name\"); q. }\n function b( required User q ) {\n  q.\n }\n}\n"
	pr := parser.ParseWithOptions("file:///t.cfc", src, &parser.ParseOptions{})
	spans := pr.FunctionSpans()

	if items := queryColumnItems(src, "q", 3, 4, spans); len(items) != 0 {
		t.Errorf("an argument of b got a's columns: %d items", len(items))
	}

	if items := queryColumnItems(src, "q", 1, 44, spans); len(items) == 0 {
		t.Error("the query in its own function offers no columns")
	}
}

// At the top of a file, what a function declared is not the page's.
func TestQueryColumnsOutsideFunctionsIgnoreTheirBodies(t *testing.T) {
	src := "<cfscript>\nfunction a() { var users = queryNew(\"x\"); }\nusers.\n</cfscript>\n"
	pr := parser.ParseWithOptions("file:///t.cfm", src, &parser.ParseOptions{})

	if items := queryColumnItems(src, "users", 2, 6, pr.FunctionSpans()); len(items) != 0 {
		t.Errorf("a function's query offered %d items at the top level", len(items))
	}
}

// A word that is another object's member is not the query of that name.
func TestQueryColumnsAreNotOfAnotherObjectsMember(t *testing.T) {
	for text, want := range map[string]bool{
		"cfg.users.":       true,
		"session.users.":   true,
		"local.users.":     false,
		"variables.users.": false,
		"users.":           false,
		"x = f(users.":     false,
	} {
		if got := memberOfAnother(text, 0, len(text)); got != want {
			t.Errorf("%q: got %v, want %v", text, got, want)
		}
	}
}

// A function in a <cfscript> block is read as the file is: a hash expression in
// a string before the query is declared does not lose it.
func TestQueryColumnsInAFunctionOfACFScriptBlock(t *testing.T) {
	src := "<cfscript>\nfunction a() {\n var s = \"#now()# \\\"x\\\"\";\n var q = queryNew(\"id,name\");\n q.\n}\n</cfscript>\n"
	pr := parser.ParseWithOptions("file:///t.cfm", src, &parser.ParseOptions{})

	if items := queryColumnItems(src, "q", 4, 3, pr.FunctionSpans()); len(items) == 0 {
		t.Error("the query declared after a string with a hash expression is lost")
	}
}

func TestQueryColumnsOfAnEmptyNameAreNone(t *testing.T) {
	src := "<cfquery name=\"\">select a from t</cfquery>\n<cfoutput>#.</cfoutput>\n"
	if items := queryColumnItems(src, "", 1, 11, nil); len(items) != 0 {
		t.Errorf("a dot after no word offered %d items", len(items))
	}
}

func TestQueryColumnsInAScriptComponentScanOnlyTheFunction(t *testing.T) {
	src := "component {\n function a() { var q = queryNew(\"x\"); }\n function b() {\n  var s = \"#now()#\";\n  var q = queryNew(\"id,name\");\n  q.\n }\n}\n"
	pr := parser.ParseWithOptions("file:///t.cfc", src, &parser.ParseOptions{})

	items := queryColumnItems(src, "q", 5, 4, pr.FunctionSpans())

	var labels []string
	for _, it := range items {
		labels = append(labels, it.Label)
	}

	if !hasLabel(labels, "id") || hasLabel(labels, "x") {
		t.Errorf("labels %q, want b's columns id and name, not a's x", labels)
	}
}

func TestMemberOfAnotherReadsALongerLine(t *testing.T) {
	content := "x = 1;\ny = cfg.users."
	if !memberOfAnother(content, 1, len("y = cfg.users.")) {
		t.Error("cfg.users. is not a member of another object")
	}

	if memberOfAnother("users.", 0, len("users.")) {
		t.Error("a bare name is a member")
	}
}

func TestQueryColumnsDistinguishExplicitScopes(t *testing.T) {
	text := "component {\nfunction f() {\n local.rows = queryNew(\"local_id\");\n variables.rows = queryNew(\"component_id\");\n local.rows.\n variables.rows.\n}\n}\n"

	local := dotCompletionLabels(t, text, 4, uint32(len(" local.rows.")))
	if !hasLabel(local, "local_id") || hasLabel(local, "component_id") {
		t.Errorf("local.rows columns = %q, want local_id only", local)
	}

	variables := dotCompletionLabels(t, text, 5, uint32(len(" variables.rows.")))
	if !hasLabel(variables, "component_id") || hasLabel(variables, "local_id") {
		t.Errorf("variables.rows columns = %q, want component_id only", variables)
	}
}
