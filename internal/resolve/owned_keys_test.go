package resolve

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/vfs"
)

// TestAFactoryCachingUnderItsOwnKeyReturnsWhatItStores: Preside's test base
// builds its PresideObjectService into request[ key ], key a string it
// computes from a literal no other code spells, mocks it in place, and returns
// request[ key ]; a wrapper returning the factory's call returns the same.
// The guard (`force || !request.keyExists( key )`) cannot be followed, but only
// the factory writes the key. A second factory spelling the prefix that stores
// another component, a key handed to another function, or a scan that does
// not know every file withholds the answer.
func TestAFactoryCachingUnderItsOwnKeyReturnsWhatItStores(t *testing.T) {
	factory := func(stored string) string {
		return `<cfcomponent output="false">
	<cffunction name="_getService" access="private" returntype="any" output="false">
		<cfargument name="force" type="boolean" required="false" default="false" />
		<cfscript>
			var key = "_cachedService" & Hash( SerializeJson( arguments ) );
			var other = _getCache( cacheKey="_cacheBox" & key );

			if ( arguments.force || !request.keyExists( key ) ) {
				request[ key ] = new ` + stored + `();
				request[ key ] = getMockbox().createMock( object=request[ key ] );
			}

			return request[ key ];
		</cfscript>
	</cffunction>
</cfcomponent>`
	}

	base := func() map[string]string {
		return map[string]string{
			"svc/Thing.cfc": `component { function go() {} }`,
			"svc/Other.cfc": `component {}`,
			"Base.cfc":      factory("svc.Thing"),
			"Child.cfc": `<cfcomponent output="false" extends="Base">
	<cffunction name="_wrapper" access="private" returntype="any" output="false">
		<cfscript>
			return _getService(
				  force = true
			)
		</cfscript>
	</cffunction>
</cfcomponent>`,
		}
	}

	returns := func(files map[string]string, file, fn string, batch bool) string {
		dir := t.TempDir()
		writeFiles(t, dir, files)

		r := &Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}}
		if batch {
			r.InferArgsFiles = cfmlFilesIn(t, dir)
		}

		return r.ReturnComponentOf(r.ResolveFunc(filepath.Join(dir, file), fn, dir))
	}

	if got := returns(base(), "Base.cfc", "_getService", true); filepath.Base(got) != "Thing.cfc" {
		t.Errorf("factory: %q, want Thing.cfc", got)
	}

	if got := returns(base(), "Child.cfc", "_wrapper", true); filepath.Base(got) != "Thing.cfc" {
		t.Errorf("wrapper: %q, want Thing.cfc", got)
	}

	if got := returns(base(), "Base.cfc", "_getService", false); got != "" {
		t.Errorf("outside a batch scan: %q, want nothing", got)
	}

	same := base()
	same["Base2.cfc"] = factory("svc.Thing")

	if got := returns(same, "Base.cfc", "_getService", true); filepath.Base(got) != "Thing.cfc" {
		t.Errorf("a second factory storing the same: %q, want Thing.cfc", got)
	}

	differs := base()
	differs["Base2.cfc"] = factory("svc.Other")

	if got := returns(differs, "Base.cfc", "_getService", true); got != "" {
		t.Errorf("a second factory storing another: %q, want nothing", got)
	}

	handed := base()
	handed["Base.cfc"] = strings.Replace(factory("svc.Thing"), "var other =", "store( key ); var other =", 1)

	if got := returns(handed, "Base.cfc", "_getService", true); got != "" {
		t.Errorf("a key handed elsewhere: %q, want nothing", got)
	}

	spelled := base()
	spelled["page.cfm"] = `<cfset request[ "_cachedService" & x ] = "">`

	if got := returns(spelled, "Base.cfc", "_getService", true); got != "" {
		t.Errorf("another file spelling the prefix: %q, want nothing", got)
	}
}
