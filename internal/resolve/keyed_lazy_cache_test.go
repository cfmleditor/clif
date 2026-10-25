package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestAKeyedLazyCacheReturnsWhatItsKeyBuilds: the cfwheels CLI builds its
// services on first use, in one function keyed by the service's name:
//
//	private any function getService( required string name ) {
//		if ( !structKeyExists( variables.services, name ) ) {
//			switch ( name ) {
//				case "helpers": variables.services.helpers = new services.Helpers(); break;
//				…
//			}
//		}
//		return variables.services[ name ];
//	}
//
// getService( "helpers" ) is what the key's field holds, on a hit or after the
// build: every write to variables.services.helpers, which must agree. The
// guarded-field rule read only a literal key in structKeyExists; the key here
// is the argument the call spells. A key the call does not spell, and a field
// another write sets to something else, still give nothing.
func TestAKeyedLazyCacheReturnsWhatItsKeyBuilds(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"services/Helpers.cfc": `component { function pluralize( word ) {} }`,
		"services/Admin.cfc":   `component { function generate() {} }`,
		"services/Other.cfc":   `component {}`,
		"Module.cfc": `component {
	function init() {
		variables.services = {};
		return this;
	}
	private any function getService( required string name ) {
		if ( !structKeyExists( variables.services, name ) ) {
			switch ( name ) {
				case "helpers":
					variables.services.helpers = new services.Helpers();
					break;
				case "admin":
					variables.services.admin = new services.Admin();
					break;
				case "reset":
					variables.services.reset = new services.Admin();
					break;
			}
		}
		return variables.services[ name ];
	}
	function rebuild() {
		variables.services.reset = new services.Other();
	}
}`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Module.cfc")

	lookup := r.FuncLookup(dir)

	for call, want := range map[string]string{
		`getService( "helpers" )`: "services/Helpers.cfc",
		`getService( "admin" )`:   "services/Admin.cfc",
		`getService( name )`:      "",
		`getService( "reset" )`:   "",
	} {
		got := lookup("Module", parser.CallHop(call))

		switch {
		case want == "" && got != "":
			t.Errorf("Module.%s guessed %q", call, got)
		case want != "" && r.ComponentPath(got, dir) != filepath.Join(dir, filepath.FromSlash(want)):
			t.Errorf("Module.%s = %q, want %s", call, got, want)
		}
	}
}

// TestAGuardedFieldIsProvedFromTheText: a statement the plan cannot read
// (`variables.hits++`) may write any field it mentions, and a large
// component always holds one, so the guarded-field rule gave up on every
// field of the cfwheels CLI's 8,000-line Module.cfc. When every reference to
// the field's variable is in a statement the plan read, the text shows the
// unread one does not touch it. A reference in an unread statement, and the
// scope used whole, still give nothing; `structDelete( variables, "x" )`
// touches only x.
func TestAGuardedFieldIsProvedFromTheText(t *testing.T) {
	for _, tc := range []struct{ name, extra, want string }{
		{"unread statement elsewhere", `function bump() { variables.hits++; structDelete( variables, "tracker" ); }`, "services/Helpers.cfc"},
		{"unread statement on the field", `function bump() { variables.services.hits++; }`, ""},
		{"scope used whole", `function bump() { variables.hits++; structClear( variables ); }`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"services/Helpers.cfc": `component {}`,
				"Module.cfc": `component {
	function init() {
		variables.services = {};
		variables.hits = 0;
		return this;
	}
	private any function getService( required string name ) {
		if ( !structKeyExists( variables.services, name ) ) {
			switch ( name ) {
				case "helpers":
					variables.services.helpers = new services.Helpers();
					break;
			}
		}
		return variables.services[ name ];
	}
	` + tc.extra + `
}`,
			})

			r := &Resolver{}
			reasonsWith(t, r, dir, "Module.cfc")

			got := r.FuncLookup(dir)("Module", parser.CallHop(`getService( "helpers" )`))

			switch {
			case tc.want == "" && got != "":
				t.Errorf("guessed %q", got)
			case tc.want != "" && r.ComponentPath(got, dir) != filepath.Join(dir, filepath.FromSlash(tc.want)):
				t.Errorf("got %q, want %s", got, tc.want)
			}
		})
	}
}
