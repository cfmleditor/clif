package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type decodedSemToken struct {
	line    uint32
	col     uint32
	length  uint32
	tokType string
	mods    []string
	text    string
}

func decodeSemanticTokens(t *testing.T, content string, data []uint32) []decodedSemToken {
	t.Helper()

	if len(data)%5 != 0 {
		t.Fatalf("semantic token data length %d is not a multiple of 5", len(data))
	}

	lines := strings.Split(content, "\n")
	out := make([]decodedSemToken, 0, len(data)/5)

	var line, col uint32

	for i := 0; i < len(data); i += 5 {
		dLine, dCol, length, tokIdx, modMask := data[i], data[i+1], data[i+2], data[i+3], data[i+4]
		if dLine == 0 {
			col += dCol
		} else {
			line += dLine
			col = dCol
		}

		var mods []string

		for bit, name := range semanticTokensLegend.TokenModifiers {
			if modMask&(1<<uint(bit)) != 0 {
				mods = append(mods, name)
			}
		}

		var text string

		if int(line) < len(lines) {
			ln := lines[line]
			if int(col+length) <= len(ln) {
				text = ln[col : col+length]
			}
		}

		out = append(out, decodedSemToken{
			line:    line,
			col:     col,
			length:  length,
			tokType: semanticTokensLegend.TokenTypes[tokIdx],
			mods:    mods,
			text:    text,
		})
	}

	return out
}

func TestSemanticTokensOnByDefaultAndCanBeDisabled(t *testing.T) {
	s := newTestServer()
	if !s.Features.SemanticTokens {
		t.Fatal("expected SemanticTokens to be on by default")
	}

	s.Features.SemanticTokens = false

	caps := s.capabilities()
	if caps.SemanticTokensProvider != nil {
		t.Fatalf("expected SemanticTokensProvider to be nil when disabled, got %#v", caps.SemanticTokensProvider)
	}

	docURI := uri.URI("file:///workspace/Test.cfc")
	content := "component {\n\tfunction hello() { return 1; }\n}\n"
	s.setDocument(docURI, content)

	raw, err := json.Marshal(protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	})
	if err != nil {
		t.Fatal(err)
	}

	res, err := s.handleSemanticTokensFull(context.Background(), raw)
	if err != nil {
		t.Fatalf("handleSemanticTokensFull error: %v", err)
	}

	if res != nil {
		t.Fatalf("expected nil result when disabled, got %#v", res)
	}
}

func TestSemanticTokensCapabilityWhenEnabled(t *testing.T) {
	s := newTestServer()
	s.Features.SemanticTokens = true

	caps := s.capabilities()
	opts, ok := caps.SemanticTokensProvider.(*protocol.SemanticTokensOptions)

	if !ok || opts == nil {
		t.Fatalf("expected *protocol.SemanticTokensOptions when enabled, got %#v", caps.SemanticTokensProvider)
	}

	if opts.Full != protocol.Boolean(true) || opts.Range != protocol.Boolean(true) {
		t.Fatalf("expected Full=true and Range=true, got Full=%#v Range=%#v", opts.Full, opts.Range)
	}

	if len(opts.Legend.TokenTypes) == 0 || len(opts.Legend.TokenModifiers) == 0 {
		t.Fatalf("expected non-empty legend, got %#v", opts.Legend)
	}
}

func TestSemanticTokensScriptFullAndRange(t *testing.T) {
	s := newTestServer()
	s.Features.SemanticTokens = true

	content := strings.Join([]string{
		"component {",
		"\tpublic any function greet(required string userId) {",
		"\t\tvar svc = new models.UserService();",
		"\t\tvar cleaned = trim(arguments.userId);",
		"\t\tlocal.out = svc.findUser(userId);",
		"\t\treturn helper(cleaned);",
		"\t}",
		"}",
	}, "\n")

	docURI := uri.URI("file:///workspace/Greeter.cfc")
	s.setDocument(docURI, content)
	s.mu.Lock()
	s.parseResults[docURI] = parser.Parse(docURI, content)
	s.mu.Unlock()

	rawFull, err := json.Marshal(protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	})
	if err != nil {
		t.Fatal(err)
	}

	resAny, err := s.handleSemanticTokensFull(context.Background(), rawFull)
	if err != nil {
		t.Fatalf("handleSemanticTokensFull error: %v", err)
	}

	res, ok := resAny.(*protocol.SemanticTokens)
	if !ok || res == nil {
		t.Fatalf("expected *protocol.SemanticTokens, got %#v", resAny)
	}

	toks := decodeSemanticTokens(t, content, res.Data)

	want := []struct {
		text    string
		tokType string
		mod     string
	}{
		{"greet", "method", "declaration"},
		{"userId", "parameter", "declaration"},
		{"models", "type", ""},
		{"UserService", "type", ""},
		{"trim", "function", "defaultLibrary"},
		{"arguments", "namespace", ""},
		{"local", "namespace", ""},
		{"findUser", "method", ""},
		{"userId", "parameter", ""},
		{"helper", "function", ""},
	}

	if len(toks) != len(want) {
		t.Fatalf("got %d tokens %+v, want %d", len(toks), toks, len(want))
	}

	for i, w := range want {
		got := toks[i]
		if got.text != w.text || got.tokType != w.tokType {
			t.Errorf("token[%d]: got (%q, %q), want (%q, %q)", i, got.text, got.tokType, w.text, w.tokType)
		}

		gotMod := ""
		if len(got.mods) > 0 {
			gotMod = strings.Join(got.mods, ",")
		}

		if gotMod != w.mod {
			t.Errorf("token[%d] %q mods: got %q, want %q", i, got.text, gotMod, w.mod)
		}
	}

	// Range request for line 3 only (`var cleaned = trim(arguments.userId);`).
	rawRange, err := json.Marshal(protocol.SemanticTokensRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Range: protocol.Range{
			Start: protocol.Position{Line: 3, Character: 0},
			End:   protocol.Position{Line: 3, Character: 60},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rangeAny, err := s.handleSemanticTokensRange(context.Background(), rawRange)
	if err != nil {
		t.Fatalf("handleSemanticTokensRange error: %v", err)
	}

	rangeRes, ok := rangeAny.(*protocol.SemanticTokens)
	if !ok || rangeRes == nil {
		t.Fatalf("expected *protocol.SemanticTokens, got %#v", rangeAny)
	}

	rangeToks := decodeSemanticTokens(t, content, rangeRes.Data)
	if len(rangeToks) != 2 {
		t.Fatalf("range tokens: got %d (%+v), want 2", len(rangeToks), rangeToks)
	}

	if rangeToks[0].text != "trim" || rangeToks[1].text != "arguments" {
		t.Fatalf("unexpected range tokens: %+v", rangeToks)
	}
}

func TestSemanticTokensTagFile(t *testing.T) {
	s := newTestServer()
	s.Features.SemanticTokens = true

	content := strings.Join([]string{
		`<cfcomponent>`,
		`	<cffunction name="renderItem" access="public">`,
		`		<cfargument name="itemId" type="numeric">`,
		`		<cfset local.x = len(itemId)>`,
		`		<cfreturn variables.svc.load(itemId)>`,
		`	</cffunction>`,
		`</cfcomponent>`,
	}, "\n")

	docURI := uri.URI("file:///workspace/Item.cfc")
	s.setDocument(docURI, content)
	s.mu.Lock()
	s.parseResults[docURI] = parser.Parse(docURI, content)
	s.mu.Unlock()

	raw, err := json.Marshal(protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
	})
	if err != nil {
		t.Fatal(err)
	}

	resAny, err := s.handleSemanticTokensFull(context.Background(), raw)
	if err != nil {
		t.Fatalf("handleSemanticTokensFull error: %v", err)
	}

	res, ok := resAny.(*protocol.SemanticTokens)
	if !ok || res == nil {
		t.Fatalf("expected *protocol.SemanticTokens, got %#v", resAny)
	}

	toks := decodeSemanticTokens(t, content, res.Data)

	want := []struct {
		text    string
		tokType string
		mod     string
	}{
		{"renderItem", "method", "declaration"},
		{"itemId", "parameter", "declaration"},
		{"local", "namespace", ""},
		{"len", "function", "defaultLibrary"},
		{"itemId", "parameter", ""},
		{"variables", "namespace", ""},
		{"load", "method", ""},
		{"itemId", "parameter", ""},
	}

	if len(toks) != len(want) {
		t.Fatalf("got %d tokens %+v, want %d", len(toks), toks, len(want))
	}

	for i, w := range want {
		got := toks[i]
		if got.text != w.text || got.tokType != w.tokType {
			t.Errorf("token[%d]: got (%q, %q), want (%q, %q)", i, got.text, got.tokType, w.text, w.tokType)
		}

		gotMod := ""
		if len(got.mods) > 0 {
			gotMod = strings.Join(got.mods, ",")
		}

		if gotMod != w.mod {
			t.Errorf("token[%d] %q mods: got %q, want %q", i, got.text, gotMod, w.mod)
		}
	}
}

func TestSemanticTokensLinearScalingOnLongSingleLine(t *testing.T) {
	// Pathological single-line script with 2,000 calls: must stay strictly O(N)
	// (incremental UTF-16 column tracking prevents O(L^2) line rescans).
	var b strings.Builder

	b.WriteString("component { function big(a) { ")

	for range 2000 {
		b.WriteString("local.x = trim(a); ")
	}

	b.WriteString("} }")

	content := b.String()
	docURI := uri.URI("file:///workspace/LongLine.cfc")
	pr := parser.Parse(docURI, content)

	start := time.Now()
	data := computeSemanticTokens(content, "/workspace/LongLine.cfc", pr, nil)
	dur := time.Since(start)

	if len(data) == 0 {
		t.Fatal("expected non-empty semantic tokens")
	}

	if dur > 250*time.Millisecond {
		t.Fatalf("computeSemanticTokens took %v on 2000-call single line; expected < 250ms", dur)
	}
}

func makeSyntheticSemanticCFC(funcs int) string {
	var b strings.Builder

	b.WriteString("component {\n")

	for i := range funcs {
		fmt.Fprintf(&b, "\tpublic any function handle%d(required string id, numeric limit=10) {\n", i)
		b.WriteString("\t\tvar svc = new models.UserService();\n")
		b.WriteString("\t\tlocal.clean = trim(arguments.id);\n")
		b.WriteString("\t\tlocal.res = svc.find(id, limit);\n")
		b.WriteString("\t\treturn helper(local.res);\n")
		b.WriteString("\t}\n")
	}

	b.WriteString("}\n")

	return b.String()
}

func BenchmarkSemanticTokens(b *testing.B) {
	content := makeSyntheticSemanticCFC(200) // ~1,200 lines
	docURI := uri.URI("file:///workspace/Bench.cfc")
	pr := parser.Parse(docURI, content)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		data := computeSemanticTokens(content, "/workspace/Bench.cfc", pr, nil)
		if len(data) == 0 {
			b.Fatal("empty data")
		}
	}
}
