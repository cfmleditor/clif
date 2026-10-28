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
