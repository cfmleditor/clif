package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.lsp.dev/uri"
)

// findNames are names a FindCalls parse is asked for in these tests: the
// resolver's framework readers' and a few that every file calls.
var findNames = []string{
	"view", "renderView", "setView", "announce", "processState", "hasMany", "belongsTo",
	"map", "to", "init", "includeUDF", "ComponentListPackageAsStruct", "dspObject_include",
	"get", "set", "len", "writeOutput",
}

// callKey is what a FindCalls parse must record of a call as ExtractCalls does.
type callKey struct {
	name, variable, caller string
	line, start, end       uint32
	span, this             bool
	chain                  string
}

func keysOf(calls []CallSite) []callKey {
	out := make([]callKey, 0, len(calls))
	for i := range calls {
		c := &calls[i]
		out = append(out, callKey{
			name: strings.ToLower(c.FuncName), variable: c.Variable, caller: c.Caller,
			line: c.Line, start: c.StartByte, end: c.EndByte, span: c.HasSpan, this: c.This,
			chain: strings.Join(c.Chain, "|"),
		})
	}

	slices.SortFunc(out, func(a, b callKey) int {
		if a.line != b.line {
			return int(a.line) - int(b.line)
		}

		if a.start != b.start {
			return int(a.start) - int(b.start)
		}

		return strings.Compare(a.name+a.variable, b.name+b.variable)
	})

	return out
}

// eachFixture runs f over every CFML file under testdata.
func eachFixture(t *testing.T, f func(path, file, content string)) {
	t.Helper()

	err := filepath.WalkDir("../../testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".cfc") && !strings.HasSuffix(path, ".cfm") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		f(path, "file:///x/"+filepath.ToSlash(path), string(data))

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A parse asked for some names' calls and not for every call records exactly
// the calls of those names an ExtractCalls parse records, each where it is
// written, with its receiver, chain and caller, in every fixture: script and
// tag, in <cfset>, <cfif> and <cfreturn>, inside #...# in output and in
// strings, and in a <script> block's interpolation. It is asked for the
// readers' names, and then for every name the file calls.
func TestFindCallsRecordsWhatExtractCallsRecordsOfThoseNames(t *testing.T) {
	n := 0

	eachFixture(t, func(path, file, content string) {
		extracted := ParseWithOptions(uri.URI(file), content, &ParseOptions{ExtractCalls: true, ScanAllScopes: true})

		every := []string{}
		for _, c := range extracted.AllCalls() {
			if !namedIn(every, c.FuncName) {
				every = append(every, c.FuncName)
			}
		}

		for _, names := range [][]string{findNames, every} {
			want := keysOf(extracted.CallsNamed(names...))
			got := keysOf(ParseWithOptions(uri.URI(file), content, &ParseOptions{FindCalls: names}).CallsWritten(names...))

			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s: FindCalls recorded\n%+v\nExtractCalls recorded\n%+v", path, got, want)
			}

			n += len(want)
		}
	})

	if n < 100 {
		t.Fatalf("only %d calls compared", n)
	}
}

// Recording a few names' calls changes nothing else a parse finds: its
// functions, scopes, refs, properties, links and instantiations are the
// plain parse's.
func TestFindCallsLeavesTheRestOfTheParseAlone(t *testing.T) {
	eachFixture(t, func(path, file, content string) {
		plain := Parse(uri.URI(file), content)
		found := ParseWithOptions(uri.URI(file), content, &ParseOptions{FindCalls: findNames})

		for _, c := range []struct {
			field     string
			got, want any
		}{
			{"Funcs", found.Funcs, plain.Funcs},
			{"Scopes", found.Scopes, plain.Scopes},
			{"ComponentRefs", found.ComponentRefs, plain.ComponentRefs},
			{"Properties", found.Properties, plain.Properties},
			{"Extends", found.Extends, plain.Extends},
			{"Links", found.Links, plain.Links},
			{"Instantiations", found.Instantiations, plain.Instantiations},
			{"FunctionSpans", found.FunctionSpans(), plain.FunctionSpans()},
			{"AllVars", found.AllVars(), plain.AllVars()},
		} {
			if !reflect.DeepEqual(c.got, c.want) {
				t.Errorf("%s: %s differs from the plain parse's:\n%+v\nwant\n%+v", path, c.field, c.got, c.want)
			}
		}

		for _, s := range plain.Scopes {
			got, _ := found.FuncRefs(s.Start, s.End)
			if want, _ := plain.FuncRefs(s.Start, s.End); !reflect.DeepEqual(got, want) {
				t.Errorf("%s: FuncRefs of %s differ:\n%+v\nwant\n%+v", path, s.Name, got, want)
			}
		}
	})
}

// A call is recorded once, where it is written: not where the function is
// declared, not in a comment, and not as text inside a string; a call inside
// a string's #...# is one.
func TestFindCallsRecordsOnlyCalls(t *testing.T) {
	src := "component {\n" +
		"\tfunction setView( view ) {}\n" +
		"\tfunction index( event ) {\n" +
		"\t\t// event.setView( \"old\" );\n" +
		"\t\tvar s = \"setView( 'quoted' )\";\n" +
		"\t\tevent.setView( \"new\" );\n" +
		"\t\treturn \"#renderView( 'partial' )#\";\n" +
		"\t}\n}\n"

	pr := ParseWithOptions("file:///t/h.cfc", src, &ParseOptions{FindCalls: []string{"setView", "renderView"}})

	var got []string
	for _, c := range pr.CallsWritten("setView", "renderView") {
		got = append(got, c.Variable+"."+c.FuncName+"@"+src[c.StartByte:c.EndByte])
	}

	want := []string{`event.setView@setView( "new" )`, `.renderView@renderView( 'partial' )`}
	if !slices.Equal(got, want) {
		t.Errorf("calls = %q, want %q", got, want)
	}

	if all := pr.AllCalls(); all != nil {
		t.Errorf("a FindCalls parse answers AllCalls with %d calls; it extracts none", len(all))
	}
}
