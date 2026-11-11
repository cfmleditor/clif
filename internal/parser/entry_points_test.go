package parser

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.lsp.dev/uri"
)

// The light readings are options of the one parse: a ScopesOnly parse's
// scopes are a full parse's, and a KeepVars parse's declarations are those
// AllVars reads again from a parse that did not keep them.
func TestTheLightReadingsAgreeWithTheParse(t *testing.T) {
	n := 0

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

		content := string(data)
		full := Parse("", content)

		if light := FindFuncScopes(content); !reflect.DeepEqual(full.Scopes, light) {
			t.Errorf("%s: FindFuncScopes = %+v\nthe parse's = %+v", path, light, full.Scopes)
		}

		if kept := ParseVars(content); !reflect.DeepEqual(full.AllVars(), kept) {
			t.Errorf("%s: ParseVars = %+v\nthe parse's AllVars = %+v", path, kept, full.AllVars())
		}

		n++

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if n == 0 {
		t.Fatal("found no fixtures under ../../testdata")
	}
}

// A <cfscript> island inside a <cffunction> is read as part of that function
// by every entry point, as the parse reads it: its var declares a local, a
// later <cfset> of that name is no variables-scope declaration, and its
// component ref is the function's, not the file's. The entry points used to
// walk the island on its own and filed both at file level.
func TestAnIslandBelongsToItsTagFunctionInEveryEntryPoint(t *testing.T) {
	src := `<cfcomponent>
<cffunction name="toXML" returntype="string">
	<cfscript>
	var buffer = createObject( "java", "java.lang.StringBuilder" ).init( "" );
	</cfscript>
	<cfset buffer = "x">
	<cfreturn buffer>
</cffunction>
</cfcomponent>
`

	for _, v := range ParseVars(src) {
		if v.Name == "buffer" && v.Scope != ScopeLocal {
			t.Errorf("ParseVars has %+v, want buffer local only", v)
		}
	}

	if globals := GlobalVars(src); len(globals) != 0 {
		t.Errorf("GlobalVars = %v, want none", globals)
	}

	for _, r := range ParseComponentRefs(uri.URI("file:///t/X.cfc"), src) {
		if r.Variable == "buffer" {
			t.Errorf("ParseComponentRefs has the island's %+v at file level", r)
		}
	}
}
