package server

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// completionLabels opens text as a document and returns the labels
// handleCompletion offers at (line, char), however the request was triggered.
func completionLabels(t *testing.T, text string, line, char uint32, ctx protocol.CompletionContext) []string {
	t.Helper()

	s := newTestServer()
	docURI := uri.File("/ws/Doc.cfm")

	open, err := json.Marshal(protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "cfml", Version: 1, Text: text},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.handleDidOpen(context.Background(), open); err != nil {
		t.Fatal(err)
	}

	req, err := json.Marshal(protocol.CompletionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: line, Character: char},
		Context:      ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.handleCompletion(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	list, ok := res.(*protocol.CompletionList)
	if !ok {
		t.Fatalf("not a completion list: %#v", res)
	}

	labels := make([]string, 0, len(list.Items))
	for i := range list.Items {
		labels = append(labels, list.Items[i].Label)
	}

	return labels
}

func hasLabel(labels []string, want string) bool {
	return slices.Contains(labels, want)
}

// Inside an attribute's value the tag's own values come first and every
// built-in function follows, since the value may be an expression.
func TestAnAttributeValueOffersTheTagsValuesAndTheBuiltins(t *testing.T) {
	text := `<cffunction output="">`
	got := completionLabels(t, text, 0, 20, protocol.CompletionContext{})

	for _, want := range []string{"true", "false", "arrayLen"} {
		if !hasLabel(got, want) {
			t.Errorf("no %q among %d items inside output=\"\"", want, len(got))
		}
	}

	if hasLabel(got, "cfset") {
		t.Error("a tag name is offered inside an attribute value")
	}
}

// A tag name being typed, with no trigger character, offers tag names and not
// the functions or the ordinary scope's variables.
func TestATagNameBeingTypedOffersTags(t *testing.T) {
	got := completionLabels(t, `<cf`, 0, 3, protocol.CompletionContext{})

	if !hasLabel(got, "cfset") || !hasLabel(got, "cfabort") {
		t.Errorf("tag names missing among %d items after <cf", len(got))
	}

	if hasLabel(got, "arrayLen") {
		t.Error("a function is offered where a tag name is being typed")
	}
}

// A dot after a receiver that is no component of the workspace offers the
// member functions every value has.
func TestADotAfterAnUnknownReceiverOffersMemberFunctions(t *testing.T) {
	got := completionLabels(t, `<cfset y = x.`, 0, 13, protocol.CompletionContext{})

	if !hasLabel(got, "len") || !hasLabel(got, "append") {
		t.Errorf("member functions missing among %d items after x.", len(got))
	}

	if hasLabel(got, "cfset") || hasLabel(got, "arrayLen") {
		t.Error("a tag or a global function is offered after a dot")
	}
}

func TestTheTriggerIsNamedForTheLog(t *testing.T) {
	for _, c := range []struct {
		kind protocol.CompletionTriggerKind
		char string
		want string
	}{
		{protocol.CompletionTriggerKindInvoked, "", "invoked"},
		{protocol.CompletionTriggerKindTriggerCharacter, ".", "char:."},
		{protocol.CompletionTriggerKindTriggerForIncompleteCompletions, "", "incomplete"},
		{0, "", ""},
	} {
		if got := triggerKindName(c.kind, c.char); got != c.want {
			t.Errorf("triggerKindName(%v, %q) = %q, want %q", c.kind, c.char, got, c.want)
		}
	}
}
