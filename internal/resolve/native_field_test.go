package resolve

import (
	"strings"
	"testing"
)

// TestAFieldOnlyEverAnEmptyCollectionIsThatCollection: ColdBox's
// FrameworkSupertype sets `variables.cbInjectedHelpers = {}` in init() and
// then asks `variables.cbInjectedHelpers.keyExists( h )` and writes its keys;
// the field was typed nowhere, since a struct literal types only a
// pseudo-constructor assignment. When every direct write to the field in the
// component, the classes above it and the classes below it, is the same empty
// collection, the field is that collection: a key or element write, and a
// member call, do not change what it is. A write of anything else anywhere in
// that family, a struct in one place and an array in another, or the
// variables scope indexed by a computed key, leaves it untyped.
func TestAFieldOnlyEverAnEmptyCollectionIsThatCollection(t *testing.T) {
	base := `component {
	function init() {
		variables.helpers = {};
		variables.queue = [];
		return this;
	}
	function inject( h ) {
		if ( !variables.helpers.keyExists( h ) ) {
			variables.helpers[ h ] = true;
		}
		variables.queue.append( h );
		helpers.clear();
	}
	function reset() {
		variables.helpers = structNew();
	}
}`

	for _, tc := range []struct {
		name  string
		files map[string]string
		want  map[string]string
	}{
		{"only empty collections", nil, map[string]string{
			"variables.helpers.keyExists": "",
			"variables.queue.append":      "",
			"helpers.clear":               "",
		}},
		{"a subclass writes something else", map[string]string{
			"Sub.cfc": `component extends="Base" { function swap() { variables.helpers = makeHelpers(); } }`,
		}, map[string]string{"variables.helpers.keyExists": "no component ref"}},
		{"a base class writes something else", map[string]string{
			"Root.cfc": `component { function swap() { variables.helpers = makeHelpers(); } }`,
			"Base.cfc": strings.Replace(base, "component {", `component extends="Root" {`, 1),
		}, map[string]string{"variables.helpers.keyExists": "no component ref"}},
		{"a struct and an array", map[string]string{
			"Base.cfc": strings.Replace(base, "variables.helpers = structNew();", "variables.helpers = [];", 1),
		}, map[string]string{"variables.helpers.keyExists": "no component ref"}},
		{"the scope indexed by a computed key", map[string]string{
			"Base.cfc": strings.Replace(base, "variables.helpers = structNew();", "variables[ arguments.name ] = 1;", 1),
		}, map[string]string{"variables.helpers.keyExists": "no component ref"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Base.cfc": base, "Other.cfc": `component {}`})
			writeFiles(t, dir, tc.files)

			got := reasonsWith(t, &Resolver{}, dir, "Base.cfc")
			for call, want := range tc.want {
				g, ok := got[call]

				switch {
				case !ok:
					t.Errorf("%s not recorded: %v", call, got)
				case want == "" && g != "":
					t.Errorf("%s: %q, want resolved", call, g)
				case want != "" && !strings.Contains(g, want):
					t.Errorf("%s: %q, want %q", call, g, want)
				}
			}
		})
	}
}

// A field written a literal with entries is a struct or an array as much as
// one written empty, so a key of it is a value of no shape, and a method no
// struct has is still not answered.
func TestAFieldWrittenOnlyLiteralsIsThatCollection(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Route.cfc": `component {
	function init() {
		variables.route = { "name" : "x", "headers" : {}, "verbs" : [ "GET" ] };
		variables.list = [ 1, 2 ];
		return this;
	}
	function run() {
		variables.route.headers.append( { a : 1 } );
		variables.route.verbs.len();
		variables.list.len();
		variables.route.someKeyFunction();
	}
}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Route.cfc")

	for _, k := range []string{"variables.route.headers.append", "variables.route.verbs.len", "variables.list.len"} {
		if got[k] != "" {
			t.Errorf("%s: %q, want it accepted", k, got[k])
		}
	}

	if !strings.Contains(got["variables.route.someKeyFunction"], "no component ref") {
		t.Errorf("a method no struct has was answered: %q", got["variables.route.someKeyFunction"])
	}
}

// cborm's list() answers an array of the service's entity once it is told
// asQuery = false, and a query otherwise.
func TestAListOfAnEntityServiceIsAnArrayOfTheEntityWhenNotAQuery(t *testing.T) {
	if !returnsEntityArray("list", "sortOrder = \"role\", asQuery = false") || !returnsEntityArray("list", "asQuery=false") {
		t.Error("list( asQuery = false ) is an array of the entity")
	}

	for _, c := range [][2]string{{"list", "sortOrder = \"role\""}, {"list", "asQuery = true"}, {"list", "asQuery = false, properties = \"a\""}, {"get", "asQuery = false"}} {
		if returnsEntityArray(c[0], c[1]) {
			t.Errorf("%s( %s ) is not an array of the entity", c[0], c[1])
		}
	}

	if !returnsEntityArray("findAllWhere", "criteria = x") || !returnsEntityArray("getAll", "") || returnsEntityArray("getAll", "properties = \"a\"") {
		t.Error("getAll and findAllWhere")
	}
}
