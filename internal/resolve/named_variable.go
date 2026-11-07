package resolve

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
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

	// The name itself, then without an object prefix: each candidate a unique
	// component that declares the method.
	for _, candidate := range namedCandidates(name) {
		file := r.uniqueComponentNamed(candidate)
		if file == "" {
			continue
		}

		if r.lookupFunc(file, call.FuncName, 0) != nil {
			return file
		}

		return ""
	}

	return r.fewDeclarersNamedLike(name, call.FuncName)
}

// genericWords are the words of a name that say nothing about which component
// it is: sharing one with a component's name is no evidence.
var genericWords = map[string]bool{
	"bean": true, "service": true, "manager": true, "object": true, "model": true, "base": true, "abstract": true,
	"item": true, "data": true, "info": true, "this": true, "that": true, "current": true, "result": true,
	"value": true, "list": true, "args": true, "instance": true, "target": true, "config": true,
}

// camelWords splits a name at its capitals: `thisAuthorBean` is this, author, bean.
func camelWords(name string) []string {
	var words []string

	start := 0

	for i := 1; i < len(name); i++ {
		if unicode.IsUpper(rune(name[i])) && !unicode.IsUpper(rune(name[i-1])) {
			words = append(words, strings.ToLower(name[start:i]))
			start = i
		}
	}

	return append(words, strings.ToLower(name[start:]))
}

// namedCandidates are the component names a variable may be called after: the
// name, and the name without its object prefix.
func namedCandidates(name string) []string {
	out := []string{name}
	if bare := withoutObjectPrefix(name); bare != name {
		out = append(out, bare)
	}

	return out
}

// maxFewDeclarers is how many components may declare a method for it to say
// which component a receiver is.
const maxFewDeclarers = 4

// fewDeclarersNamedLike is the one component, of the few that declare the
// method, whose name shares a meaningful word with the variable's
// (`settings.getSiteEmail()` with only Setup and SiteSettings declaring it).
// Two components that share a word is no answer. The method is the evidence
// and the name the check that it is not coincidence.
func (r *Resolver) fewDeclarersNamedLike(name, method string) string {
	if r.Index == nil {
		return ""
	}

	defs := r.Index.Lookup(method)
	if len(defs) == 0 || len(defs) > maxFewDeclarers {
		return ""
	}

	vars := map[string]bool{}

	for _, w := range camelWords(withoutObjectPrefix(name)) {
		if len(w) >= minNamedVariable && !genericWords[w] {
			vars[w] = true
		}
	}

	found := ""

	for _, d := range defs {
		if d.URI == "" || !strings.EqualFold(filepath.Ext(cfpath.FromURI(string(d.URI))), ".cfc") || frameworkapi.IsStub(cfpath.FromURI(string(d.URI))) {
			continue
		}

		file := cfpath.FromURI(string(d.URI))
		base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

		for _, w := range camelWords(base) {
			if !vars[w] {
				continue
			}

			if found != "" && !cfpath.SamePath(found, file) {
				return ""
			}

			found = file
		}
	}

	return found
}

// variableBaseName is the name a variable is known by: `arguments.x` and
// `variables.x` are x, a longer path or an index is none.
func variableBaseName(variable string) string {
	name := variable

	for _, scope := range []string{"arguments.", "variables.", "local.", "prc.", "rc.", "args.", "request.", "attributes."} {
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

// noRefReceiver reports whether reason is a receiver the source never typed,
// for a call made on it directly.
func noRefReceiver(call *parser.CallSite, reason string) bool {
	return call.Variable != "" && len(call.Chain) == 0 && strings.HasPrefix(reason, "variable '") && strings.HasSuffix(reason, "' has no component ref")
}

// isNativeMemberOfUnknown reports whether a call on an untyped receiver is of a
// member function the engine gives a struct, array, string or query
// (`keyExists`, `isEmpty`), though a component or two declares one of that name
// as well: the receiver is far more likely the native value than that
// component. Names five or more components declare are the shared-member rule's.
func (r *Resolver) isNativeMemberOfUnknown(call *parser.CallSite, reason string) bool {
	return noRefReceiver(call, reason) && (IsMemberFunction(call.FuncName) || parser.IsMemberMethod(call.FuncName))
}

// minDistinctiveName is the length of a method name, in characters, that is
// specific enough to say which component it belongs to.
const minDistinctiveName = 10

// isDistinctiveMember reports whether a call on an untyped receiver is of a
// distinctive method (two words or more, ten characters or more, not a
// built-in) that exactly one component of the workspace declares. A name that
// specific is not shared by chance, so the receiver is that component or
// something that quacks like it; a generic name (`get`, `process`) says nothing.
func (r *Resolver) isDistinctiveMember(call *parser.CallSite, reason string) bool {
	if !noRefReceiver(call, reason) || len(call.FuncName) < minDistinctiveName || len(camelWords(call.FuncName)) < 2 || IsBuiltin(call.FuncName) || r.Index == nil {
		return false
	}

	defs := r.Index.Lookup(call.FuncName)
	if len(defs) != 1 {
		return false
	}

	file := cfpath.FromURI(string(defs[0].URI))

	return strings.EqualFold(filepath.Ext(file), ".cfc") && !frameworkapi.IsStub(file)
}
