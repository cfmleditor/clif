package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Slatwall's services declare almost none of the methods called on them:
// Slatwall.com.service.BaseService answers newProduct(), getOrder( id ),
// getSkuBySkuCode( code ), saveOrder( order ) and getProductSmartList( rc )
// with onMissingMethod, by the convention its own comment documents (from
// ColdSpring 2.0's AbstractGateway). The name after the verb is an ORM
// entity, which its BaseDAO prefixes with "Slatwall" before loading:
// `getProductService().newProduct()` is a SlatwallProduct. On Slatwall that
// was the last assignment of most controller and service variables, every one
// reported untyped.

// slatwallBaseService is the component whose onMissingMethod does this.
const slatwallBaseService = "Slatwall.com.service.BaseService"

// slatwallSmartList is what getXSmartList() builds (BaseDAO.getSmartList).
const slatwallSmartList = "Slatwall.org.entitySmartList.SmartList"

// slatwallGatewayReturn is what calling method, which the component at path
// does not declare, returns when Slatwall's BaseService answers it, or "".
// new, get and save hand back the entity; get…SmartList the SmartList; the
// list and delete forms return an array and a boolean.
func (r *Resolver) slatwallGatewayReturn(path, method, baseDir string) string {
	if path == "" || r.Index == nil {
		return ""
	}

	name := strings.ToLower(parser.CallHopName(method))
	if expression, hop := parser.CallExpression(method); hop {
		name = strings.ToLower(wheelsCallName(expression))
	}

	verb := ""

	for _, v := range []string{"new", "get", "save"} {
		if strings.HasPrefix(name, v) && len(name) > len(v) {
			verb = v

			break
		}
	}

	if verb == "" {
		return ""
	}

	omm := r.ResolveFunc(path, "onMissingMethod", baseDir)
	if omm == nil || !omm.URI.IsFile() {
		return ""
	}

	declaring := omm.URI.Path()
	if !samePath(r.ComponentPath(slatwallBaseService, filepath.Dir(declaring)), declaring) {
		return ""
	}

	entity := name[len(verb):]

	if verb == "get" {
		if base, ok := strings.CutSuffix(entity, "smartlist"); ok {
			if base == "" {
				return ""
			}

			return r.ComponentPath(slatwallSmartList, filepath.Dir(declaring))
		}

		// getXByY: the entity is what comes before the first "by", as
		// onMissingGetMethod splits it.
		if i := strings.Index(entity, "by"); i > 0 {
			entity = entity[:i]
		}
	}

	if !strings.HasPrefix(entity, "slatwall") {
		entity = "slatwall" + entity
	}

	return r.nearestOf(r.Index.NamedEntities(entity), baseDir)
}
