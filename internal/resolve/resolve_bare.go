package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// resolveBareCall is canResolveCall for an unqualified call: this file, a
// function-reference local, the extends chain, cfinclude, and a
// variables-scope function reference.
func (r *Resolver) resolveBareCall(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	funcName := call.FuncName

	tr.addf("unqualified call to %q — checking same file", funcName)

	if r.bareCallInComponent(call, pr, baseDir, tr) || r.bareCallThroughContext(call, pr, baseDir, tr) {
		return ""
	}

	if r.fileExtends(pr) != "" {
		if base, implied := r.fileMissingBase(pr, baseDir); base != "" {
			if implied {
				tr.addf("the framework base %s implies does not resolve (%s) — accepted as dynamic", r.fileExtends(pr), base)
				tr.hit(TargetDynamic, "", nil)

				return ""
			}

			return MissingBaseReason(base)
		}

		return "not found in extends chain"
	}

	return "no qualifier, not in file"
}

// bareCallInComponent reports whether an unqualified call is to a function the
// component has of its own or by inheritance, recording where it landed: this
// file, a function-reference local, the extends chain, the framework's virtual
// base, a WireBox delegate, or PresideSuperClass.
func (r *Resolver) bareCallInComponent(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) bool {
	funcName := call.FuncName

	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, funcName) {
			tr.hit(TargetSameFile, "", &pr.Funcs[i])
			tr.addf("found %q defined in this file", funcName)

			return true
		}
	}

	// A bare call whose name matches a declared local variable or argument in
	// the enclosing function is a call through a function-reference value (e.g.
	// "var columnfn = ARGUMENTS.extend[x].column; ... columnfn(...)"), not a call
	// to a missing/undefined function — CFML supports first-class function
	// values assigned to locals/arguments. This is a genuinely different case
	// from "no qualifier, not in file": the identifier IS declared, its value is
	// just dynamic (unknown until runtime), so — same reasoning as the
	// ARGUMENTS.x-as-function-reference case below — there's nothing further to
	// verify statically, and it should not be reported as if it were a missing
	// function.
	if scope := pr.FuncScopeAt(int(call.Line)); scope.Start != -1 {
		if pr.FuncDeclares(scope.Start, scope.End, funcName) {
			tr.addf("%q is a declared local variable/argument in the enclosing function — treating as a call through a function-reference value, not a missing function", funcName)
			tr.hit(TargetDynamic, "", nil)

			return true
		}
	}

	// A script function's parameters are not among the names the body walk
	// declares (FuncDeclares reads `var` locals and <cfargument> tags), so
	// `function apply( fn, v ) { return fn( v ); }` was reported as a call to a
	// function called fn. It is a call through the argument's value.
	if call.Caller != "" && pr != nil &&
		slices.ContainsFunc(pr.Funcs, func(f parser.FunctionDef) bool {
			return strings.EqualFold(f.Name, call.Caller) &&
				slices.ContainsFunc(f.Arguments, func(a parser.Argument) bool { return strings.EqualFold(a.Name, funcName) })
		}) {
		tr.addf("%q is an argument of the enclosing function %q — treating as a call through a function-reference value, not a missing function", funcName, call.Caller)
		tr.hit(TargetDynamic, "", nil)

		return true
	}

	// Check extends chain
	if r.fileExtends(pr) != "" {
		tr.addf("not in this file — checking extends chain (%s)", r.fileExtends(pr))

		if def := r.inheritedFunc(pr, funcName, baseDir); def != nil {
			tr.hit(TargetExtends, r.fileExtends(pr), def)
			tr.addf("found %q in extends chain", funcName)

			return true
		}
	}

	if def, base := r.virtualBaseFunc(pr, funcName); def != nil {
		tr.hit(TargetExtends, base, def)
		tr.addf("found %q on %s, which the framework gives this component by virtual inheritance", funcName, base)

		return true
	}

	if def := r.delegatedFunc(pr.Delegates, baseDir, funcName, 0); def != nil {
		tr.hit(TargetExtends, "", def)
		tr.addf("found %q among the methods a WireBox delegate gives this component", funcName)

		return true
	}

	if pr.URI.IsFile() && strings.HasPrefix(funcName, "$") {
		if def := r.presideSuperClassFunc(r.extendsChainOf(pr.URI.Path()), funcName, 0); def != nil {
			tr.hit(TargetExtends, "", def)
			tr.addf("found %q on PresideSuperClass, which Preside's WireBox gives a presideService component", funcName)

			return true
		}
	}

	return false
}

// bareCallThroughContext reports whether an unqualified call is to a function
// the component reaches from outside itself, recording where it landed: a
// cfinclude, a Wheels host or include wrapper, a variables-scope function
// reference, a module helper, an onMissingMethod, or what every subclass
// declares.
func (r *Resolver) bareCallThroughContext(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) bool {
	funcName := call.FuncName

	if def, via := r.findThroughIncludes(pr, funcName); def != nil {
		tr.hit(TargetInclude, "", def)
		tr.addf("found %q through cfinclude, in %s", funcName, via)

		return true
	}

	if def := r.wheelsMixinHostFunc(pr, funcName, call.Caller); def != nil {
		tr.hit(TargetExtends, "", def)
		tr.addf("found %q on the Wheels host that integrates this component's public methods", funcName)

		return true
	}

	if def := r.wheelsWrappedTemplateFunc(pr, funcName); def != nil {
		tr.hit(TargetInclude, "", def)
		tr.addf("found %q on the component that includes this template through a Wheels include wrapper", funcName)

		return true
	}

	// CFML looks a bare name up in the variables scope, so a bare call to
	// one the file assigns there — VARIABLES.render = ARGUMENTS.render —
	// is a call through that function-reference property, the same as the
	// qualified VARIABLES.render() accepted below. The parser records
	// VARIABLES.name() as a bare call wherever it reads it as a member of
	// this component, which is what left `#VARIABLES._renderTemplate()#`
	// in a string "no qualifier, not in file".
	if pr.HasScopedAssignment(parser.ScopeVariables, funcName) {
		tr.addf("%q is assigned in VARIABLES scope — a call through a function-reference property", funcName)
		tr.hit(TargetDynamic, "", nil)

		return true
	}

	if module, ok := r.moduleHelper(pr, funcName, baseDir); ok {
		tr.addf("%q is a helper the %s module mixes into ColdBox components", funcName, module)
		tr.hit(TargetDynamic, "", nil)

		return true
	}

	if r.thisCallAnswered(call, pr, baseDir) {
		tr.addf("%q is called through this., and the component answers a missing method with onMissingMethod", funcName)
		tr.hit(TargetDynamic, "", nil)

		return true
	}

	if comps := r.subclassesDeclare(pr, funcName, tr); comps != "" {
		tr.hit(TargetDynamic, comps, nil)

		return true
	}

	return false
}

// resolveBareChain is canResolveCall for a call chained onto a bare one,
// `expect( x ).toBe( 1 )`: the first call is looked up as a bare call is —
// this file, its extends chain, what it includes — and the rest of the chain
// is checked on what it returns.
//
// It was resolved as a bare call to the *last* name, which looked for toBe
// among the spec's own methods and reported it "not found in extends chain"
// — once the chain resolved, in every TestBox assertion.
func (r *Resolver) resolveBareChain(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	first := parser.CallHopName(call.Chain[0])

	// The hop carries its arguments when the parser read them. Recovering
	// them from the line instead finds nothing when the name is called
	// twice there.
	expression, ok := parser.CallExpression(call.Chain[0])
	if !ok {
		expression = factoryCallExpression(pr.Content, first, int(call.Line))
	}

	tr.addf("chained on a call to %q — looking it up as an unqualified call", first)

	def := r.bareFunc(first, pr, baseDir, call.Caller)
	if def == nil && call.This && pr.URI.IsFile() {
		if e := r.slatwallGatewayReturn(pr.URI.Path(), call.Chain[0], baseDir); e != "" {
			tr.addf("this.%s() is answered by Slatwall's BaseService onMissingMethod, which returns %q", first, e)

			rest := *call
			rest.Chain = call.Chain[1:]
			rest.Component = e

			return r.canResolveCall(&rest, pr, baseDir, tr)
		}
	}

	if def == nil {
		if answer, ok := r.unboundChainHop(first, expression, call, pr, baseDir, tr); ok {
			return answer
		}

		return "chained on '" + first + "', which is not found (calling '" + call.FuncName + "')"
	}

	ret, noFollow, soft := r.chainHopReturn("this component", first, "", def, tr)

	// A method returning the class declaring it, called on this subclass,
	// returns the subclass: as hopReturn does for a qualified call.
	if sub := r.selfTyped(pr.URI.Path(), baseDir, def, ret); sub != "" {
		ret = sub
	}

	if ret == "" && expression != "" {
		ret = r.expressionReturn(def, expression, baseDir, pr.URI.Path())
	}

	if ret == "" {
		if e := r.receiverReturn(cfpath.FromURI(string(pr.URI)), first); e != "" {
			tr.addf("%q on this component returns what it binds it to: %q", first, e)
			ret = e
		}
	}

	if noFollow && ret != "" {
		tr.hit(TargetDynamic, ret, nil)

		return ""
	}

	// A dynamicIfMissing resolver's answer that names no file is dynamic here
	// as on every other path; recursing lost the flag, and the next hop
	// reported the component as one that does not exist.
	if soft && ret != "" && !r.componentExists(ret, baseDir) {
		tr.addf("%q names no file and came from a dynamicIfMissing resolver — the rest of the chain is dynamic", ret)
		tr.hit(TargetDynamic, ret, nil)

		return ""
	}

	if ret == "" {
		if reason, done := r.closureStructHops(def, pr.URI.Path(), first, call, 0, tr); done {
			return reason
		}

		ret = r.initArgGetter(def)
		if ret != "" {
			tr.addf("%q returns the property init stores from its argument, which every construction passes as %q", first, ret)
		}
	}

	if ret == "" {
		if r.engineValueReturn(def, pr.URI.Path()) {
			tr.addf("%q returns what a built-in function hands back — the rest of the chain is dynamic", first)
			tr.hit(TargetDynamic, "", nil)

			return ""
		}

		return "method '" + first + "' has no component return type (chain to '" + call.FuncName + "')"
	}

	rest := *call
	rest.Chain = call.Chain[1:]
	rest.Component = ret

	return r.canResolveCall(&rest, pr, baseDir, tr)
}

// bareFunc finds what an unqualified call to name reaches: this file's own
// function, one its extends chain declares, or one a file it includes or is
// included by declares.
func (r *Resolver) bareFunc(name string, pr *parser.ParseResult, baseDir string, caller ...string) *parser.FunctionDef {
	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, name) {
			return &pr.Funcs[i]
		}
	}

	if r.fileExtends(pr) != "" {
		if def := r.inheritedFunc(pr, name, baseDir); def != nil {
			return def
		}
	}

	// The same order a call standing alone is looked up in (canResolveCall):
	// a chain headed by $getPresideObject() or a handler's getModel() found
	// nothing where the call alone resolved.
	if def, _ := r.virtualBaseFunc(pr, name); def != nil {
		return def
	}

	if def := r.delegatedFunc(pr.Delegates, baseDir, name, 0); def != nil {
		return def
	}

	if pr.URI.IsFile() && strings.HasPrefix(name, "$") {
		if def := r.presideSuperClassFunc(r.extendsChainOf(pr.URI.Path()), name, 0); def != nil {
			return def
		}
	}

	if def, _ := r.findThroughIncludes(pr, name); def != nil {
		return def
	}

	if def := r.wheelsMixinHostFunc(pr, name, caller...); def != nil {
		return def
	}

	return r.wheelsWrappedTemplateFunc(pr, name)
}

// resolveThisCall is canResolveCall for `this.name()`.
func (r *Resolver) resolveThisCall(funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	tr.addf("'this' qualifier — checking same file")

	for i := range pr.Funcs {
		if strings.EqualFold(pr.Funcs[i].Name, funcName) {
			tr.hit(TargetSameFile, "", &pr.Funcs[i])

			return ""
		}
	}

	if r.fileExtends(pr) != "" {
		tr.addf("not in this file — checking extends chain (%s)", r.fileExtends(pr))

		if def := r.inheritedFunc(pr, funcName, baseDir); def != nil {
			tr.hit(TargetExtends, r.fileExtends(pr), def)

			return ""
		}
	}

	if def := r.delegatedFunc(pr.Delegates, baseDir, funcName, 0); def != nil {
		tr.hit(TargetExtends, "", def)
		tr.addf("found %q among the methods a WireBox delegate gives this component", funcName)

		return ""
	}

	if def, via := r.findThroughIncludes(pr, funcName); def != nil {
		tr.hit(TargetInclude, "", def)
		tr.addf("found %q through cfinclude, in %s", funcName, via)

		return ""
	}

	if comps := r.subclassesDeclare(pr, funcName, tr); comps != "" {
		tr.hit(TargetDynamic, comps, nil)

		return ""
	}

	if r.fileExtends(pr) != "" {
		if base, implied := r.fileMissingBase(pr, baseDir); base != "" {
			if implied {
				tr.addf("the framework base %s implies does not resolve (%s) — accepted as dynamic", r.fileExtends(pr), base)
				tr.hit(TargetDynamic, "", nil)

				return ""
			}

			return MissingBaseReason(base)
		}
	}

	return "method '" + funcName + "' not found in current component"
}

// resolveSuperCall is canResolveCall for `super.name()`.
func (r *Resolver) resolveSuperCall(funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	if r.fileExtends(pr) == "" {
		return "super used but no extends"
	}

	tr.addf("'super' qualifier — checking extends chain (%s)", r.fileExtends(pr))

	if def := r.ResolveFunc(r.fileExtends(pr), funcName, baseDir); def != nil {
		tr.hit(TargetExtends, r.fileExtends(pr), def)

		return ""
	}

	if base, implied := r.fileMissingBase(pr, baseDir); base != "" {
		if implied {
			tr.addf("the framework base %s implies does not resolve (%s) — accepted as dynamic", r.fileExtends(pr), base)
			tr.hit(TargetDynamic, "", nil)

			return ""
		}

		return MissingBaseReason(base)
	}

	return "not found in parent component"
}
