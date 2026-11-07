package resolve

import (
	"strings"
	"unicode"

	"github.com/cfmleditor/clif/internal/parser"
)

// minNamedVariable is the shortest variable name taken as a component's name:
// `c`, `e` and `qs` are names of anything.
const minNamedVariable = 4

// componentNamedLikeVariable is the file of the one component in the workspace
// named as the variable is (`addressBean` for addressBean.cfc, or `oAddressBean`
// by the o prefix convention), for a call of a method that component, or one
// above it, declares. It is the last answer for a receiver nothing typed, and
// it answers only when the method exists there, so it turns no finding into a
// different finding: a name that fits the component and a call that does not
// stays unresolved as it was. The name is a convention, not a proof, so the
// report applies it (namedComponentDeclares, in ClassifyReasonIn) and
// CanResolveCall does not.
func (r *Resolver) componentNamedLikeVariable(variable string, call *parser.CallSite) string {
	name := variableBaseName(variable)
	if len(name) < minNamedVariable || len(call.Chain) > 0 || call.FuncName == "" {
		return ""
	}

	for _, candidate := range []string{name, withoutObjectPrefix(name)} {
		file := r.uniqueComponentNamed(candidate)
		if file == "" {
			continue
		}

		if r.lookupFunc(file, call.FuncName, 0) != nil {
			return file
		}

		return ""
	}

	return ""
}

// variableBaseName is the name a variable is known by: `arguments.x` and
// `variables.x` are x, a longer path or an index is none.
func variableBaseName(variable string) string {
	name := variable

	for _, scope := range []string{"arguments.", "variables.", "local."} {
		if rest, ok := strings.CutPrefix(strings.ToLower(name), scope); ok {
			name = name[len(name)-len(rest):]

			break
		}
	}

	if name == "" || strings.ContainsAny(name, ".[(") {
		return ""
	}

	return name
}

// withoutObjectPrefix is name minus a leading `o` that begins a capitalised
// word (`oAddressBean` is addressBean).
func withoutObjectPrefix(name string) string {
	if len(name) > minNamedVariable && name[0] == 'o' && unicode.IsUpper(rune(name[1])) {
		return name[1:]
	}

	return name
}

// namedComponentDeclares reports whether the call's reason is a receiver with
// no ref that componentNamedLikeVariable names a component for.
func (r *Resolver) namedComponentDeclares(call *parser.CallSite, reason string) bool {
	if call.Variable == "" || !strings.HasPrefix(reason, "variable '") || !strings.HasSuffix(reason, "' has no component ref") {
		return false
	}

	return r.componentNamedLikeVariable(call.Variable, call) != ""
}
