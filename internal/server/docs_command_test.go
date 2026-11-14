package server

import (
	"context"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
)

func runCommand(t *testing.T, s *Server, command string, args ...any) any {
	t.Helper()

	raw := makeCall(t, protocol.MethodWorkspaceExecuteCommand, protocol.ExecuteCommandParams{
		Command:   command,
		Arguments: lspAnyArgs(args...),
	})

	got, err := s.handleExecuteCommand(context.Background(), raw)
	if err != nil {
		t.Fatalf("%s: %v", command, err)
	}

	return got
}

func TestSearchDocsRanksNamesBeforeDescriptions(t *testing.T) {
	got, _ := runCommand(t, newTestServer(), "clif.searchDocs", "query", "all", 200).([]docSummary)
	if len(got) < 3 {
		t.Fatalf("%d results for query, want several", len(got))
	}

	// cfquery and the function queryNew hold the word in their names; whatever
	// only mentions it in a description follows them.
	nameHits := 0

	for _, d := range got {
		holds := strings.Contains(strings.ToLower(d.Name), "query")
		if !holds && nameHits == 0 {
			t.Fatalf("%s, which holds the word only in its description, comes first", d.Name)
		}

		if holds {
			nameHits++
		} else if nameHits > 0 {
			break
		}
	}

	if got[0].Name != "cfquery" && got[0].Name != "query" {
		t.Errorf("the first result is %s, want the exact name", got[0].Name)
	}
}

func TestSearchDocsFiltersByKindAndLimits(t *testing.T) {
	s := newTestServer()

	tags, _ := runCommand(t, s, "clif.searchDocs", "", "tag", 5).([]docSummary)
	if len(tags) != 5 {
		t.Fatalf("%d tags, want the limit of 5", len(tags))
	}

	for _, d := range tags {
		if d.Type != "tag" {
			t.Errorf("%s is a %s in a tags search", d.Name, d.Type)
		}
	}

	funcs, _ := runCommand(t, s, "clif.searchDocs", "arrayappend", "function").([]docSummary)
	if len(funcs) == 0 || funcs[0].Name != "arrayAppend" || funcs[0].Type != "function" {
		t.Errorf("arrayappend finds %+v, want arrayAppend first", funcs)
	}

	if all, _ := runCommand(t, s, "clif.searchDocs").([]docSummary); len(all) != defaultDocResults {
		t.Errorf("%d results with no arguments, want the default %d", len(all), defaultDocResults)
	}
}

func TestDocEntryAnswersTheWholeEntry(t *testing.T) {
	s := newTestServer()

	d, ok := runCommand(t, s, "clif.docEntry", "CFLOCATION").(docDetail)
	if !ok || d.Name != "cflocation" || d.Type != "tag" || len(d.Params) == 0 || !strings.Contains(d.Markdown, "**Attributes:**") {
		t.Fatalf("cflocation: %+v", d)
	}

	fn, ok := runCommand(t, s, "clif.docEntry", "arrayAppend", "function").(docDetail)
	if !ok || fn.Type != "function" || fn.Returns == "" || !strings.Contains(fn.Markdown, "**Parameters:**") {
		t.Fatalf("arrayAppend: %+v", fn)
	}

	if got := runCommand(t, s, "clif.docEntry", "nosuchthing"); got != nil {
		t.Errorf("an unknown name answers %+v, want null", got)
	}
}

func TestDocsCommandsAreAdvertised(t *testing.T) {
	cmds := newTestServer().capabilities().ExecuteCommandProvider.Commands

	for _, want := range []string{"clif.searchDocs", "clif.docEntry"} {
		found := false

		for _, c := range cmds {
			found = found || c == want
		}

		if !found {
			t.Errorf("%s is not advertised", want)
		}
	}
}
