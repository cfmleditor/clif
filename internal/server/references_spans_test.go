package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
)

// utf16Col is where text starts on its line, in UTF-16 units.
func utf16Col(prefix string) uint32 { return uint32(len(utf16.Encode([]rune(prefix)))) }

// Two calls of one name on a line are two references, each at its own place.
// The place used to be the first match of the name on the line, so the second
// call was reported at the first one's column and then dropped as a repeat of
// it: a line with three calls showed one. An unrelated component's call to a
// method of the same name, on the same line, is not one of them.
func TestReferencesPlaceEveryCallOnALine(t *testing.T) {
	for name, eol := range map[string]string{"lf": "\n", "crlf": "\r\n"} {
		dir := t.TempDir()
		svc := "component {" + eol + "\tpublic void function foo() {}" + eol + "}" + eol

		files := map[string]string{
			"Svc.cfc":       svc,
			"Unrelated.cfc": "component {" + eol + "\tpublic void function foo() {}" + eol + "}" + eol,
		}

		lines := []string{
			"<cfscript>",
			"svc = new Svc(); u = new Unrelated();",
			"svc.foo(1); u.foo(0); svc.foo(2); x = svc.foo(3);",
			`é = "😀"; svc.foo(4); u.foo(5); y = svc.foo(6);`,
			"</cfscript>",
		}
		files["page.cfm"] = strings.Join(lines, eol) + eol

		for file, src := range files {
			if err := os.WriteFile(filepath.Join(dir, file), []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
		}

		srv := newTestServer()
		srv.References = true
		srv.WorkspaceFolders = []string{dir}
		srv.indexWorkspace()

		svcURI := cfpath.ToURI(filepath.Join(dir, "Svc.cfc"))
		openDoc(t, srv, svcURI, svc)

		got := map[protocol.Position]bool{}

		for _, l := range referencesAt(t, srv, svcURI, 1, 25, false) {
			if cfpath.FromURI(string(l.URI)) != filepath.Join(dir, "page.cfm") {
				t.Errorf("%s: a reference outside page.cfm: %s", name, l.URI)

				continue
			}

			if l.Range.End.Character-l.Range.Start.Character != 3 {
				t.Errorf("%s: reference %+v does not span the name", name, l.Range)
			}

			got[l.Range.Start] = true
		}

		// Every svc.foo, by where it stands; none of the u.foo.
		want := map[protocol.Position]bool{}

		for i, line := range lines {
			for from := 0; ; {
				at := strings.Index(line[from:], "svc.foo(")
				if at < 0 {
					break
				}

				at += from
				want[protocol.Position{Line: uint32(i), Character: utf16Col(line[:at+len("svc.")])}] = true
				from = at + 1
			}
		}

		if len(want) != 5 {
			t.Fatalf("test setup: %d calls expected, want 5", len(want))
		}

		if len(got) != len(want) {
			t.Errorf("%s: %d references, want %d: got %v", name, len(got), len(want), got)
		}

		for pos := range want {
			if !got[pos] {
				t.Errorf("%s: no reference at %d:%d (got %v)", name, pos.Line, pos.Character, got)
			}
		}
	}
}

// A file changed since the scan no longer has the name where the parse put it,
// and the reference falls back to the line rather than point at other text.
func TestEntryFileSpanRangeRefusesAMovedName(t *testing.T) {
	f := &entryFile{text: "abc foo(1);\nxyz foo(2);\n"}

	if rng, ok := f.spanRange(4, "foo"); !ok || rng.Start.Line != 0 || rng.Start.Character != 4 || rng.End.Character != 7 {
		t.Errorf("a name where it was put = %+v, %v", rng, ok)
	}

	if rng, ok := f.spanRange(16, "foo"); !ok || rng.Start.Line != 1 || rng.Start.Character != 4 {
		t.Errorf("a name on the second line = %+v, %v", rng, ok)
	}

	for _, start := range []int{0, 3, 5, -1, 1000} {
		if _, ok := f.spanRange(start, "foo"); ok {
			t.Errorf("start %d answered, but foo is not there", start)
		}
	}

	if _, ok := (*entryFile)(nil).spanRange(4, "foo"); ok {
		t.Error("an unreadable file answered")
	}
}

// The incoming calls to a method give each call its own range: a caller with
// two calls on a line had the first one's range twice.
func TestIncomingCallsPlaceEveryCallOnALine(t *testing.T) {
	dir := t.TempDir()
	svc := "component {\n\tpublic void function foo() {}\n}\n"
	page := "<cfscript>\nsvc = new Svc();\nsvc.foo(1); é = \"😀\"; svc.foo(2);\n</cfscript>\n"

	for file, src := range map[string]string{"Svc.cfc": svc, "page.cfm": page} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	srv := newTestServer()
	srv.References = true
	srv.WorkspaceFolders = []string{dir}
	srv.indexWorkspace()

	svcURI := cfpath.ToURI(filepath.Join(dir, "Svc.cfc"))
	openDoc(t, srv, svcURI, svc)

	items := prepareCallHierarchy(t, srv, svcURI, protocol.Position{Line: 1, Character: 25})
	if len(items) != 1 {
		t.Fatalf("prepareCallHierarchy = %+v, want one item", items)
	}

	var cols []uint32

	for _, in := range incomingCalls(t, srv, &items[0]) {
		for _, r := range in.FromRanges {
			cols = append(cols, r.Start.Character)
		}
	}

	want := []uint32{utf16Col("svc."), utf16Col(`svc.foo(1); é = "😀"; svc.`)}
	if len(cols) != 2 || cols[0] != want[0] || cols[1] != want[1] {
		t.Errorf("call ranges start at columns %v, want %v", cols, want)
	}
}

// The same for the outgoing calls of a function: two calls to one callee on a
// line are two ranges.
func TestOutgoingCallsPlaceEveryCallOnALine(t *testing.T) {
	dir := t.TempDir()
	src := "component {\n\tpublic void function helper() {}\n\tpublic void function run() { helper(); é = \"😀\"; helper(); }\n}\n"

	if err := os.WriteFile(filepath.Join(dir, "Svc.cfc"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	srv := newTestServer()
	srv.References = true
	srv.WorkspaceFolders = []string{dir}
	srv.indexWorkspace()

	svcURI := cfpath.ToURI(filepath.Join(dir, "Svc.cfc"))
	openDoc(t, srv, svcURI, src)

	items := prepareCallHierarchy(t, srv, svcURI, protocol.Position{Line: 2, Character: 25})
	if len(items) != 1 || items[0].Name != "run" {
		t.Fatalf("prepareCallHierarchy = %+v, want run", items)
	}

	line := strings.Split(src, "\n")[2]

	var cols []uint32

	for _, out := range outgoingCalls(t, srv, &items[0]) {
		for _, r := range out.FromRanges {
			cols = append(cols, r.Start.Character)
		}
	}

	beforeFirst, _, _ := strings.Cut(line, "helper();")
	beforeSecond, _, _ := strings.CutLast(line, "helper();")

	want := []uint32{utf16Col(beforeFirst), utf16Col(beforeSecond)}
	if len(cols) != 2 || cols[0] != want[0] || cols[1] != want[1] {
		t.Errorf("call ranges start at columns %v, want %v", cols, want)
	}
}
