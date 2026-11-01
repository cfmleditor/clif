package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// virtualBases are the implied bases ColdBox mixes in whether or not the file
// extends something else: HandlerService registers every handler with
// setVirtualInheritance( "coldbox.system.EventHandler" ) and
// InterceptorService every interceptor with "coldbox.system.Interceptor", and
// WireBox adds the base's methods the object does not already have. Preside's
// admin handlers extend preside.system.base.AdminHandler, which extends
// nothing, and call runEvent() and renderView() bare: 600 findings. Other
// implied bases (a view's Renderer, a Wheels class) stand only for a missing
// extends, and are not here.
var virtualBases = map[string]bool{
	"coldbox.system.eventhandler": true,
	"coldbox.system.interceptor":  true,
}

// virtualBaseFunc is name as the virtual base the framework gives pr's file
// declares it, for a file that names its own extends (one that does not
// already has the base as its implied extends). The base is searched after
// the file's declared chain, since a method the object has is never replaced.
func (r *Resolver) virtualBaseFunc(pr *parser.ParseResult, name string) (*parser.FunctionDef, string) {
	if pr.Extends == "" || r.ImplicitExtends == nil || !pr.URI.IsFile() {
		return nil, ""
	}

	path := cfpath.FromURI(string(pr.URI))

	base := r.ImplicitExtends(path)
	if !virtualBases[strings.ToLower(base)] {
		return nil, ""
	}

	basePath := r.ComponentPath(base, filepath.Dir(path))
	if basePath == "" {
		return nil, ""
	}

	for _, p := range r.extendsChainOf(path) {
		if cfpath.SamePath(p, basePath) {
			return nil, ""
		}
	}

	return r.lookupFunc(basePath, name, 0), base
}
