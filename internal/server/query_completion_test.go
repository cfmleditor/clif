package server

import (
	"testing"

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

	if items := queryColumnItems(text, "rs", 2, 3); len(items) != 0 {
		t.Errorf("a query whose columns are not known was given %d items", len(items))
	}

	// What is known is offered with its kind, columns first.
	items := queryColumnItems("q = queryNew(\"a\");\nq.", "q", 1, 2)
	if len(items) != 4 || items[0].Label != "a" || items[0].Kind != protocol.CompletionItemKindField || items[1].Kind != protocol.CompletionItemKindProperty {
		t.Errorf("items %+v", items)
	}
}
