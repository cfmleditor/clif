package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
)

// Verdict is what a batch run makes of one call site.
type Verdict int

const (
	// Accepted means the call resolved, to a definition or to something dynamic.
	// Target says which.
	Accepted Verdict = iota
	// Builtin means the call is a bare call to an engine function or member
	// method that the workspace does not define, which no report or map is about.
	Builtin
	// GlobalDef means the call was accepted only because
	// ClassifyOptions.GlobalDefs counts a definition anywhere in the workspace.
	GlobalDef
	// MissingBase means the call is left unchecked because a component in its
	// extends chain names no file; Classified.Base is that component.
	MissingBase
	// Unresolved means nothing accounts for the call; Classified.Reason says why.
	Unresolved
)

// ClassifyOptions are the choices a batch run makes beyond what the resolver
// itself decides.
type ClassifyOptions struct {
	// GlobalDefs accepts a bare call that any indexed file defines.
	GlobalDefs bool
	// WantTarget also records where an accepted call landed. It allocates a
	// recorder per call, so a run that only counts leaves it off.
	WantTarget bool
}

// Classified is the verdict on one call, with what produced it.
type Classified struct {
	Target  CallTarget
	Reason  string
	Base    string
	Verdict Verdict
}

// ClassifyCall is the one place a batch run decides whether a call resolved.
//
// `unresolved` and the code map each had their own copy of this decision, and
// the copies had drifted: the map hid a call as a builtin when it had a chain
// (`f().len()`), and by a smaller list of names, so the two disagreed about what
// was left to report. The order here is `unresolved`'s: the resolver decides
// first, so a function the workspace declares is not mistaken for the builtin
// that shares its name; only then is a builtin set aside, then the global
// definition, then a missing base.
func (r *Resolver) ClassifyCall(call *parser.CallSite, pr *parser.ParseResult, baseDir string, opt ClassifyOptions) Classified {
	var c Classified

	if opt.WantTarget {
		c.Target, c.Reason = r.ResolveCallTarget(call, pr, baseDir)
	} else {
		c.Reason = r.CanResolveCall(call, pr, baseDir)
	}

	switch {
	case c.Reason == "":
		c.Verdict = Accepted
	case IsBuiltinCall(call):
		c.Verdict = Builtin
	case opt.GlobalDefs && r.Index.CountFunctions(call.FuncName) > 0:
		c.Verdict = GlobalDef
	default:
		c.Verdict = Unresolved

		if base, ok := MissingBaseOf(c.Reason); ok {
			c.Verdict, c.Base = MissingBase, base
		}
	}

	return c
}

// IsBuiltinCall reports whether a call is a bare call by a built-in's name:
// no receiver and no chain, since `f().len()` is a call on what f returns.
func IsBuiltinCall(call *parser.CallSite) bool {
	return call.Variable == "" && len(call.Chain) == 0 && IsBuiltin(call.FuncName)
}

// IsBuiltin reports whether name is a built-in function or member function,
// which a report or a map leaves out.
func IsBuiltin(name string) bool {
	return docs.IsBuiltinFunction(name) || parser.IsMemberMethod(name) || IsMemberFunction(name)
}

var memberFuncSet = func() map[string]bool {
	m := make(map[string]bool)
	for _, mf := range docs.AllMemberFunctions() {
		m[strings.ToLower(mf.Name)] = true
	}

	return m
}()

// IsMemberFunction reports whether name is a documented member function.
func IsMemberFunction(name string) bool {
	return memberFuncSet[strings.ToLower(name)]
}
