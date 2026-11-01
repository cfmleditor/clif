package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/parser"
)

// The bundled ColdBox stubs are generated from one pinned ColdBox, newer than
// many applications run. Where a project's ColdBox is not in the workspace
// (the chain reaches ColdBox only through the stubs), which version it runs
// is unknown, so a method ColdBox removed is still taken as the method it
// was an alias of. setNextEvent() was relocate() under its ColdBox 3 to 5
// name; Preside, whose ColdBox is an external not in its checkout, calls it
// 412 times from its handlers. A ColdBox checkout in the workspace is read
// as it is, and a call it lacks is still reported.

// coldboxRemovedAliases maps a removed ColdBox method, lowercased, to the
// method it was an alias of, for the stubbed classes that declared it.
var coldboxRemovedAliases = map[string]struct {
	target  string
	classes []string
}{
	"setnextevent": {target: "relocate", classes: []string{"system/FrameworkSupertype.cfc", "system/web/Controller.cfc"}},
}

// legacyColdBoxFunc is the method a removed ColdBox method name stood for,
// when a component of chain is a stubbed ColdBox class that declared it.
func (r *Resolver) legacyColdBoxFunc(chain []string, name string, depth int) *parser.FunctionDef {
	alias, ok := coldboxRemovedAliases[strings.ToLower(name)]
	if !ok || depth > maxDelegateDepth {
		return nil
	}

	for _, p := range chain {
		if !frameworkapi.IsStub(p) {
			continue
		}

		slashed := filepath.ToSlash(p)
		for _, class := range alias.classes {
			if strings.HasSuffix(strings.ToLower(slashed), "/coldbox/"+strings.ToLower(class)) {
				return r.lookupFunc(p, alias.target, depth+1)
			}
		}
	}

	return nil
}
