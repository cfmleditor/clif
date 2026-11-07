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
	// KeepUntypedHops reports a call chained on a method that declares no
	// return type, as the open-world mode sets such calls apart as assumed
	// rather than leaving them out.
	KeepUntypedHops bool
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

	c.Verdict, c.Base = r.ClassifyReasonIn(call, c.Reason, opt, pr, baseDir)

	// A call set aside for its reason alone is accepted as dynamic.
	if c.Verdict == Accepted && c.Reason != "" && opt.WantTarget {
		c.Target = CallTarget{Kind: TargetDynamic, FuncName: call.FuncName, Line: call.Line}
	}

	return c
}

// ClassifyReason is the verdict on a call whose resolution has already been
// run and gave reason (empty when it resolved), and the missing base when that
// is the verdict. It is the half of [Resolver.ClassifyCall] after the
// resolver, for a caller that holds the reason already, as explain does.
func (r *Resolver) ClassifyReason(call *parser.CallSite, reason string, opt ClassifyOptions) (Verdict, string) {
	return r.ClassifyReasonIn(call, reason, opt, nil, "")
}

// ClassifyReasonIn is ClassifyReason for a call whose file is at hand: with it
// a variable can be traced to the assignment that made it.
func (r *Resolver) ClassifyReasonIn(call *parser.CallSite, reason string, opt ClassifyOptions, pr *parser.ParseResult, baseDir string) (Verdict, string) {
	switch {
	case reason == "":
		return Accepted, ""
	case IsBuiltinCall(call):
		return Builtin, ""
	case opt.GlobalDefs && r.Index.CountFunctions(call.FuncName) > 0:
		return GlobalDef, ""
	case !opt.KeepUntypedHops && isUntypedHop(reason):
		return Accepted, ""
	case r.isUndeclaredMemberCall(call, reason):
		return Builtin, ""
	case r.assignedFromUntypedFunction(call, reason, pr, baseDir):
		return Accepted, ""
	case !opt.KeepUntypedHops && r.isSharedMemberCall(call, reason):
		return Accepted, ""
	}

	if base, ok := MissingBaseOf(reason); ok {
		return MissingBase, base
	}

	return Unresolved, ""
}

// isUntypedHop reports whether reason is a chain hop whose method declares no
// return type: `svc.get().foo()` with get() untyped. What it returns is as
// dynamic as `x = svc.get()` makes x, which the same run accepts, so the run
// does not report the chained form.
func isUntypedHop(reason string) bool {
	return strings.HasPrefix(reason, "method '") && strings.Contains(reason, " has no component return type (chain to '")
}

// isUndeclaredMemberCall reports whether a call on a receiver nothing is known
// of is by the name of a member function the engine's own types have, which no
// component the workspace holds declares: it can only be that member function,
// of whatever value the receiver is. `listenerChain.len()` and
// `registry.keyExists( k )` name no component to check.
func (r *Resolver) isUndeclaredMemberCall(call *parser.CallSite, reason string) bool {
	if call.Variable == "" || !strings.HasSuffix(reason, "' has no component ref") || !strings.HasPrefix(reason, "variable '") {
		return false
	}

	return IsBuiltin(call.FuncName) && r.Index != nil && r.Index.CountFunctions(call.FuncName) == 0
}

// sharedMemberFiles is how many components must declare a method for a call of
// it on an untyped receiver to be taken as a call on one of them.
const sharedMemberFiles = 5

// isSharedMemberCall reports whether a call on a receiver nothing is known of
// is of a method that many components of the workspace declare (`getValue`,
// `validate`, `getErrors`): a duck-typed call on one of those components, which
// nothing in the source says which. Unlike a method no component declares, it
// can be a real call, and the receiver is as untyped as `x = svc.get()` makes it.
func (r *Resolver) isSharedMemberCall(call *parser.CallSite, reason string) bool {
	if call.Variable == "" || len(call.Chain) > 0 || !strings.HasSuffix(reason, "' has no component ref") || !strings.HasPrefix(reason, "variable '") {
		return false
	}

	return r.Index != nil && r.Index.CountFunctions(call.FuncName) >= sharedMemberFiles
}

// Describe says, in a line, what a batch report does with a call of this
// verdict, so a trace can end with the answer to "and then?".
func (v Verdict) Describe(base string) string {
	switch v {
	case Accepted:
		return "counted as resolved"
	case Builtin:
		return "set aside as a built-in function or member method; the report and the map leave it out"
	case GlobalDef:
		return "accepted because a function of that name is defined somewhere in the workspace"
	case MissingBase:
		return "left unchecked because " + base + " in its extends chain does not resolve; reported once per file, not per call"
	case Unresolved:
		return "reported as unresolved"
	}

	return ""
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

// assignedFromUntypedFunction reports whether the receiver nothing is known of
// was assigned the result of a call to a function of the file or its bases
// that declares no return type and whose body gives none to infer: what it
// returns is as dynamic as what `x = svc.get()` gives x when get() is untyped,
// which the same run accepts.
func (r *Resolver) assignedFromUntypedFunction(call *parser.CallSite, reason string, pr *parser.ParseResult, baseDir string) bool {
	if pr == nil || call.Variable == "" || !strings.HasSuffix(reason, "' has no component ref") || !strings.HasPrefix(reason, "variable '") {
		return false
	}

	name := strings.TrimPrefix(call.Variable, "local.")
	if name == "" || strings.ContainsAny(name, ".[(") {
		return false
	}

	start := 0
	if scope, ok := enclosingScope(pr, call.Line); ok {
		start = scope.Start
	}

	rhs, _, ok := localAssignmentAt(pr.Content, name, start, int(call.Line)+1)
	if !ok {
		return false
	}

	receiver, calls, ok := callChain(rhs)
	if !ok || receiver != "" || len(calls) != 1 || !pr.URI.IsFile() {
		return false
	}

	fd := r.ResolveFunc(pr.URI.Path(), calls[0].name, baseDir)

	return fd != nil && declaresNoReturnType(fd) && fd.ReturnComponent == "" && fd.DocReturn == "" && len(fd.ReturnSources) == 0
}

// declaresNoReturnType reports whether fd declares no return type or `any`:
// it may return anything, and nothing here says what.
func declaresNoReturnType(fd *parser.FunctionDef) bool {
	switch strings.ToLower(strings.TrimSpace(fd.ReturnType)) {
	case "", "any":
		return true
	}

	return false
}
