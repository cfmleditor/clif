package server

import (
	"context"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

const renameSource = "component {\n\tfunction f( a ) {\n\t\tvar count = 0;\n\t\tcount = count + a;\n\t\treturn \"é #count#\";\n\t}\n}\n"

func renameServer(t *testing.T, src string) (*Server, uri.URI) {
	t.Helper()

	s := newTestServer()
	s.Features.Rename = true

	docURI := uri.File("/tmp/rename_test.cfc")
	openDoc(t, s, docURI, src)

	return s, docURI
}

func prepareRename(t *testing.T, s *Server, docURI uri.URI, line, char uint32) (any, error) {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentPrepareRename, protocol.PrepareRenameParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: line, Character: char},
	})

	return s.handlePrepareRename(context.Background(), raw)
}

func renameTo(t *testing.T, s *Server, docURI uri.URI, line, char uint32, name string) (*protocol.WorkspaceEdit, error) {
	t.Helper()

	raw := makeCall(t, protocol.MethodTextDocumentRename, protocol.RenameParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Position:     protocol.Position{Line: line, Character: char},
		NewName:      name,
	})

	got, err := s.handleRename(context.Background(), raw)
	edit, _ := got.(*protocol.WorkspaceEdit)

	return edit, err
}

func TestRenameAVariableEditsEveryUse(t *testing.T) {
	s, docURI := renameServer(t, renameSource)

	// On the declaration, `count` after `var `: line 2, columns 6 to 11.
	got, err := prepareRename(t, s, docURI, 2, 7)
	if err != nil {
		t.Fatal(err)
	}

	ph, ok := got.(protocol.PrepareRenamePlaceholder)
	if !ok || ph.Placeholder != "count" || ph.Range.Start.Line != 2 || ph.Range.Start.Character != 6 || ph.Range.End.Character != 11 {
		t.Fatalf("prepareRename answered %+v", got)
	}

	edit, err := renameTo(t, s, docURI, 3, 10, "total")
	if err != nil {
		t.Fatal(err)
	}

	edits := edit.Changes[docURI]
	if len(edits) != 4 {
		t.Fatalf("%d edits, want 4: %+v", len(edits), edits)
	}

	// Applied, the function reads as written with the new name, including the
	// one in the string's #...# after an é (columns are UTF-16).
	text := renameSource

	lines := strings.Split(text, "\n")

	for _, e := range slices.Backward(edits) {
		l := []rune(lines[e.Range.Start.Line])
		lines[e.Range.Start.Line] = string(l[:e.Range.Start.Character]) + e.NewText + string(l[e.Range.End.Character:])
	}

	want := "component {\n\tfunction f( a ) {\n\t\tvar total = 0;\n\t\ttotal = total + a;\n\t\treturn \"é #total#\";\n\t}\n}\n"
	if got := strings.Join(lines, "\n"); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenameRefusesWhatItCannotMakeSafe(t *testing.T) {
	s, docURI := renameServer(t, renameSource)

	// An argument: its callers name it.
	if _, err := prepareRename(t, s, docURI, 1, 14); err == nil || !strings.Contains(err.Error(), "argument") {
		t.Errorf("an argument: %v", err)
	}

	// Outside any function.
	if _, err := prepareRename(t, s, docURI, 0, 3); err == nil {
		t.Error("the component keyword can be renamed")
	}

	for name, why := range map[string]string{
		"a b": "not a valid", "": "not a valid", "1x": "not a valid", "return": "reserved", "Local": "reserved",
		"a": "already has an argument named", "A": "already has an argument named",
	} {
		if _, err := renameTo(t, s, docURI, 2, 8, name); err == nil || !strings.Contains(err.Error(), why) {
			t.Errorf("renaming to %q: %v, want an error mentioning %q", name, err, why)
		}
	}

	// A change of case alone is a rename.
	if _, err := renameTo(t, s, docURI, 2, 8, "COUNT"); err != nil {
		t.Errorf("a change of case: %v", err)
	}
}

func TestRenameNeedsTheFeature(t *testing.T) {
	s := newTestServer()

	if p, _ := s.capabilities().RenameProvider.(protocol.Boolean); bool(p) {
		t.Error("rename advertised with the feature off")
	}

	docURI := uri.File("/tmp/rename_off.cfc")
	openDoc(t, s, docURI, renameSource)

	if _, err := renameTo(t, s, docURI, 2, 8, "total"); err == nil {
		t.Error("a rename was done with the feature off")
	}

	s.Features.Rename = true

	opts, ok := s.capabilities().RenameProvider.(*protocol.RenameOptions)
	if !ok || opts.PrepareProvider == nil || !*opts.PrepareProvider {
		t.Errorf("rename is advertised as %#v with the feature on, want one that asks to be prepared", s.capabilities().RenameProvider)
	}
}

// posOf is the position of the ¦ in needle within src: the needle is found in
// src without it, and the cursor stands where the ¦ was.
func posOf(t *testing.T, src, needle string) (line, char uint32) {
	t.Helper()

	pre, post, found := strings.Cut(needle, "¦")
	if !found {
		t.Fatalf("no cursor marker in %q", needle)
	}

	off := strings.Index(src, pre+post)
	if off < 0 {
		t.Fatalf("%q is not in the source", pre+post)
	}

	before := src[:off+len(pre)]
	line = uint32(strings.Count(before, "\n"))
	char = uint32(len([]rune(before[strings.LastIndex(before, "\n")+1:])))

	return line, char
}

// A client that skips prepareRename can send the rename from any word: on a
// struct key or another object's member spelled like the variable, it is a
// different thing than the variable, and nothing is renamed.
func TestRenameFromANameSpelledLikeTheVariableIsRefused(t *testing.T) {
	src := "component {\n\tfunction f( o ) {\n\t\tvar x = 1;\n\t\tvar s = { x: x };\n\t\treturn o.x + g( x = x ) + s.x;\n\t}\n}\n"
	s, docURI := renameServer(t, src)

	cases := []struct {
		name   string
		needle string
		ok     bool
	}{
		{"the declaration", "var ¦x", true},
		{"the value in the literal", ": ¦x }", true},
		{"the struct key", "{ ¦x:", false},
		{"another object's member", "o.¦x", false},
		{"a named argument's label", "g( ¦x =", false},
		{"the named argument's value", "= ¦x )", true},
		{"a member of s", "s.¦x", false},
	}

	for _, c := range cases {
		line, char := posOf(t, src, c.needle)

		_, err := renameTo(t, s, docURI, line, char, "total")
		if c.ok && err != nil {
			t.Errorf("%s: %v", c.name, err)
		}

		if !c.ok && (err == nil || !strings.Contains(err.Error(), "not on the variable")) {
			t.Errorf("%s: %v, want a refusal that the cursor is not on the variable", c.name, err)
		}

		if _, perr := prepareRename(t, s, docURI, line, char); (perr == nil) != c.ok {
			t.Errorf("%s: prepareRename %v, rename ok %v", c.name, perr, c.ok)
		}
	}
}

// A new name the function already means something by is refused: a variable
// it reads from outside, a function it calls, a catch's variable, a loop's,
// and, for a closure's variable, what the function around it declares.
func TestRenameRefusesANameTheFunctionAlreadyUses(t *testing.T) {
	src := "component {\n" +
		"\tfunction f( items, n = limit ) {\n" +
		"\t\tvar t = config.size;\n" +
		"\t\tvar total = 0;\n" +
		"\t\ttry { run(); } catch ( any err ) { log( err ); }\n" +
		"\t\tfor ( k in items ) { total += k; }\n" +
		"\t\titems.each( function( i ) {\n" +
		"\t\t\tvar u = i + total;\n" +
		"\t\t\treturn u;\n" +
		"\t\t} );\n" +
		"\t\treturn t + helper( t );\n" +
		"\t}\n" +
		"}\n"
	s, docURI := renameServer(t, src)

	tLine, tChar := posOf(t, src, "var ¦t =")
	uLine, uChar := posOf(t, src, "var ¦u =")

	for _, c := range []struct {
		why        string
		line, char uint32
		to, said   string
	}{
		{"read from outside", tLine, tChar, "config", "reads or calls 'config'"},
		{"read in a parameter's default", tLine, tChar, "limit", "reads or calls 'limit'"},
		{"a function it calls", tLine, tChar, "helper", "reads or calls 'helper'"},
		{"a catch variable", tLine, tChar, "err", "a catch variable named 'err'"},
		{"an unscoped loop variable", tLine, tChar, "k", "reads or calls 'k'"},
		{"a variable of the function around a closure", uLine, uChar, "total", "a variable named 'total'"},
		{"an argument of the function around a closure", uLine, uChar, "items", "an argument named 'items'"},
	} {
		if _, err := renameTo(t, s, docURI, c.line, c.char, c.to); err == nil || !strings.Contains(err.Error(), c.said) {
			t.Errorf("%s: renaming to %q: %v, want one saying %q", c.why, c.to, err, c.said)
		}
	}

	// A name nothing in the function uses is free.
	if _, err := renameTo(t, s, docURI, tLine, tChar, "size"); err != nil {
		t.Errorf("an unused name: %v", err)
	}

	for _, reserved := range []string{"return", "Local", "true", "var", "this", "new", "default", "else"} {
		if _, err := renameTo(t, s, docURI, tLine, tChar, reserved); err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Errorf("%q: %v, want it reserved", reserved, err)
		}
	}

	// A statement word is a fine name where an expression stands: t is only
	// ever read, never written at the start of a statement.
	for _, word := range []string{"lock", "trace", "exit", "param", "static", "final", "public"} {
		if _, err := renameTo(t, s, docURI, tLine, tChar, word); err != nil {
			t.Errorf("%q for a variable that is never a statement's first word: %v", word, err)
		}
	}
}

// Written at the start of a statement, `count = count + a;`, a statement word
// would be read as that statement; elsewhere it is only a name.
func TestRenameRefusesAStatementWordWhereItWouldBeOne(t *testing.T) {
	s, docURI := renameServer(t, renameSource)

	for _, word := range []string{"lock", "param", "thread", "static", "abstract", "trace"} {
		if _, err := renameTo(t, s, docURI, 2, 8, word); err == nil || !strings.Contains(err.Error(), "begins a statement") {
			t.Errorf("%q: %v, want a refusal that it begins a statement", word, err)
		}
	}

	if _, err := renameTo(t, s, docURI, 2, 8, "total"); err != nil {
		t.Errorf("an ordinary name: %v", err)
	}
}

// A struct key and a named argument's label are not names the function means
// anything by: a variable is renamed to one without being refused.
func TestRenameAllowsAKeyOrLabelName(t *testing.T) {
	src := "component {\n\tfunction f() {\n\t\tvar t = 1;\n\t\treturn g( name = t, other: { id: t } );\n\t}\n}\n"
	s, docURI := renameServer(t, src)

	for _, name := range []string{"name", "other", "id"} {
		if _, err := renameTo(t, s, docURI, 2, 7, name); err != nil {
			t.Errorf("%q is only a label or a key here: %v", name, err)
		}
	}
}

// The statement forms that follow a condition or a label count as statements.
func TestRenameRefusesAStatementWordAfterAConditionOrALabel(t *testing.T) {
	for name, body := range map[string]string{
		"if":    "if ( a ) count = 1;",
		"else":  "if ( a ) { g(); } else count = 1;",
		"case":  "switch ( a ) { case 1: count = 1; break; }",
		"while": "while ( a ) count++;",
	} {
		src := "component {\n\tfunction f( a ) {\n\t\tvar count = 0;\n\t\t" + body + "\n\t\treturn g( count );\n\t}\n}\n"
		s, docURI := renameServer(t, src)

		line, char := posOf(t, src, "var ¦count")
		if _, err := renameTo(t, s, docURI, line, char, "lock"); err == nil || !strings.Contains(err.Error(), "begins a statement") {
			t.Errorf("%s: renaming to lock: %v, want a refusal that it begins a statement", name, err)
		}
	}
}
