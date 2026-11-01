package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Preside's PresideObjectService.getObject( name ) returns the object's own
// component decorated by PresideObjectDecorator.decorate(), which injects
// the decorator's getDsn(), getName(), getTablename(), getDbAdapter() and
// onMissingMethod() into it; onMissingMethod hands selectData(), insertData()
// and the rest of the data API to the service with the object's name. The
// parse sees only `return decorated`, an argument. The decorator itself has
// every injected method, so a call on what getObject() returns is checked
// against it, and onMissingMethod answers the rest as on any component that
// declares one. $getPresideObject(), which every Preside service calls, is
// a call to getObject().

// presideObjectReturn is PresideObjectDecorator.cfc when fd is getObject()
// of a Preside checkout's PresideObjectService, or "".
func (r *Resolver) presideObjectReturn(fd *parser.FunctionDef) string {
	if fd == nil || !fd.URI.IsFile() || !strings.EqualFold(fd.Name, "getObject") || r.Index == nil {
		return ""
	}

	file := fd.URI.Path()
	if !strings.EqualFold(filepath.Base(file), "PresideObjectService.cfc") {
		return ""
	}

	for _, system := range r.presideRoots() {
		if cfpath.SamePath(file, filepath.Join(system, "services", "presideObjects", "PresideObjectService.cfc")) {
			decorator := filepath.Join(filepath.Dir(file), "PresideObjectDecorator.cfc")
			if info, err := r.fs().Stat(decorator); err == nil && !info.IsDir() {
				return decorator
			}
		}
	}

	return ""
}
