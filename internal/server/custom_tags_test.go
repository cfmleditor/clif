package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/cfmleditor/clif/internal/config"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Go-to-definition and document links reach a custom tag's template from
// each syntax that names one, and the word under the cursor is never looked
// up as a function instead.
func TestCustomTagDefinitionAndLinks(t *testing.T) {
	dir := t.TempDir()

	page := strings.Join([]string{
		`<cfimport taglib="lib" prefix="ui">`,
		`<cf_layout title="x">`,
		`<ui:button label="Go"/>`,
		`<cfabridge/>`,
		`<ui:missing/>`,
		`<cfscript>cf_layout(title="y");</cfscript>`,
		`<cfscript>cf_format(1);</cfscript>`,
	}, "\n")

	files := map[string]string{
		"app/Application.cfc":   `component { this.customTagPaths = expandPath("./tags"); }`,
		"app/index.cfm":         page,
		"app/tags/layout.cfm":   ``,
		"app/lib/button.cfm":    ``,
		"bridge/bridge.cfm":     ``,
		"app/layoutHelpers.cfc": `component { function layout() {} function missing() {} function cf_format() {} }`,
	}
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfpath.InvalidateAppMappingsCache()

	s := newTestServer()
	s.GlobalFunctionResolution = true
	s.CustomTagPaths = config.ResolveCustomTagPaths([]config.CustomTagPath{{Directory: "bridge", Prefix: "cfa"}}, dir)

	helpers := filepath.Join(dir, "app", "layoutHelpers.cfc")
	s.index.IndexFile(cfpath.ToURI(helpers), files["app/layoutHelpers.cfc"])

	docURI := uri.File(filepath.Join(dir, "app", "index.cfm"))
	open := mustJSON(t, protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: docURI, LanguageID: "cfml", Version: 1, Text: page},
	})

	if _, err := s.handleDidOpen(context.Background(), open); err != nil {
		t.Fatal(err)
	}

	target := func(rel string) uri.URI { return cfpath.ToURI(filepath.Join(dir, filepath.FromSlash(rel))) }

	for _, tc := range []struct {
		line, char uint32
		want       uri.URI
	}{
		{1, 6, target("app/tags/layout.cfm")},
		{2, 5, target("app/lib/button.cfm")},
		{3, 6, target("bridge/bridge.cfm")},
		{4, 5, ""}, // a tag nothing answers is not the function missing()
		{5, 15, target("app/tags/layout.cfm")},
		{6, 15, target("app/layoutHelpers.cfc")}, // a script call no tag answers may be a function
	} {
		res, err := s.handleDefinition(context.Background(), mustJSON(t, protocol.DefinitionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: tc.line, Character: tc.char},
		}))
		if err != nil {
			t.Fatal(err)
		}

		var got uri.URI
		if loc, ok := res.(protocol.Location); ok {
			got = loc.URI
		} else if res != nil {
			t.Errorf("%d:%d got %#v, want one location", tc.line, tc.char, res)

			continue
		}

		if got != tc.want {
			t.Errorf("%d:%d went to %q, want %q", tc.line, tc.char, got, tc.want)
		}
	}

	res, err := s.handleDocumentLink(context.Background(), mustJSON(t, protocol.DocumentLinkParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	}))
	if err != nil {
		t.Fatal(err)
	}

	links, _ := res.([]protocol.DocumentLink)

	var got []string

	for _, l := range links {
		if l.Target != nil && strings.HasPrefix(filepath.ToSlash(cfpath.FromURI(string(*l.Target))), filepath.ToSlash(dir)) {
			got = append(got, lineText(page, l.Range.Start.Line)[l.Range.Start.Character:l.Range.End.Character])
		}
	}

	if want := "layout button bridge layout"; strings.Join(got, " ") != want {
		t.Errorf("links underline %q, want %q", strings.Join(got, " "), want)
	}
}

func lineText(content string, line uint32) string {
	return strings.Split(content, "\n")[line]
}
