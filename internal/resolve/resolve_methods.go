package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
)

// checkMethodOn is canResolveCall's last step: whether the component the
// receiver resolved to defines the method, with the dynamic, builtin,
// member-method, onMissingMethod and altComp fallbacks.
func (r *Resolver) checkMethodOn(comp, softComp string, call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	funcName, variable := call.FuncName, call.Variable

	if comp == "" {
		return r.receiverlessReason(variable, call.Line, pr, baseDir, tr)
	}

	// A MockBox mock of a class: the class's methods, where the class
	// resolves, and dynamic where it does not — mocking a class of a module
	// the workspace lacks is not a finding.
	if cls, ok := strings.CutPrefix(comp, parser.MockPrefix); ok {
		if r.ComponentPath(cls, baseDir) == "" {
			tr.addf("a mock of %q, which does not resolve — accepted as dynamic", cls)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		tr.addf("a mock of %q — checking the class", cls)
		comp = cls
	}

	// Dynamic return type — method called on a result of a function returning "any".
	if comp == "$any" {
		tr.addf("component is $any (dynamic) — accepted without a method check")
		tr.hit(TargetDynamic, "$any", nil)

		return ""
	}

	if variable != "" && len(call.Chain) == 0 && pr.AssignsMember(variable, funcName, call.Line) {
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	// A registry family: the receiver is any one of the package directory's
	// components, so the method must be each member's own, or its base's. A
	// receiver several families name — the container read through a
	// list-loop item's concatenation — holds each of them on one of the
	// loop's turns, so every family must answer the call.
	if refs := familyRefs(comp); len(refs) > 0 {
		return r.checkFamiliesMethod(refs, call, pr, tr)
	}

	// Builtin return type — check method exists in-memory
	if strings.HasPrefix(comp, "$builtin.") {
		builtinName := comp[9:]
		if docs.LookupBuiltinMethod(builtinName, funcName) {
			tr.hit(TargetBuiltin, comp, nil)

			return ""
		}

		return "method '" + funcName + "' not found on builtin " + builtinName
	}

	tr.addf("checking whether %q defines method %q", comp, funcName)

	if r.methodAccepted(comp, call, pr, baseDir, tr) {
		return ""
	}

	return r.missingMethod(comp, softComp, call, pr, baseDir, tr)
}

// receiverlessReason is the verdict on a call whose receiver no step typed:
// accepted as dynamic when it belongs to a framework base that does not
// resolve, the base's reason when the file's own base does not, else that the
// variable has no component ref.
func (r *Resolver) receiverlessReason(variable string, line uint32, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	if base, implied := r.inheritedFromMissingBase(variable, line, pr, baseDir); base != "" {
		if implied {
			tr.addf("%q is the framework base's, which does not resolve (%s) — accepted as dynamic", variable, base)
			tr.hit(TargetDynamic, "", nil)

			return ""
		}

		return MissingBaseReason(base)
	}

	return "variable '" + variable + "' has no component ref"
}

// methodAccepted reports whether comp, which does not make the receiver
// untyped, answers a call of the method, recording where: a method it declares
// or inherits, an assertion a test setup installs, one MockBox adds, a member
// method every value has, onMissingMethod, an interface (which declares less
// than what implements it), or the component a resolver names by the variable.
func (r *Resolver) methodAccepted(comp string, call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) bool {
	funcName, variable := call.FuncName, call.Variable

	if def := r.ResolveFunc(comp, funcName, baseDir); def != nil {
		tr.hit(TargetComponent, comp, def)

		return true
	}

	if def := r.registeredAssertion(comp, call, pr, baseDir); def != nil {
		tr.addf("%q is a literal assertion installed by the selected beforeTests setup", funcName)
		tr.hit(TargetComponent, comp, def)

		return true
	}

	if mockDecoration(funcName) {
		tr.addf("%q is a method MockBox adds to a mock of %q — accepted", funcName, comp)
		tr.hit(TargetDynamic, comp, nil)

		return true
	}

	if parser.IsMemberMethod(funcName) {
		tr.addf("%q is a known member/Java-interop method — accepted without finding it in %q", funcName, comp)
		tr.hit(TargetMember, comp, nil)

		return true
	}

	// Component defines onMissingMethod — any method call is valid.
	if r.ResolveFunc(comp, "onMissingMethod", baseDir) != nil {
		tr.addf("%q defines onMissingMethod — any method call accepted", comp)
		tr.hit(TargetDynamic, comp, nil)

		return true
	}

	// An interface declares less than the object behind it: ColdBox types
	// its cache providers as ICacheProvider and calls getOrSet() on them,
	// which every provider has and the interface does not declare. A method
	// the interface does declare was found above.
	if r.isInterface(r.ComponentPath(comp, baseDir)) {
		tr.addf("%q is an interface, and what implements it may have %q — accepted as dynamic", comp, funcName)
		tr.hit(TargetDynamic, comp, nil)

		return true
	}

	// The ref-derived component didn't have the method. Try the variable-name resolver
	// as a fallback — a pendingCall propagation may have assigned the wrong component
	// (e.g. var objFile = _parent.getFile() inherits _parent's component, but the
	// "objFile" resolver names the real type).
	if altComp, altNoFollow := parser.ResolveFromCallFull(variable, r.Resolvers); altComp != "" && altComp != comp {
		tr.addf("%q not found in %q — trying altComp fallback: componentResolver on variable name gives %q (noFollow=%v)", funcName, comp, altComp, altNoFollow)

		if altNoFollow {
			tr.hit(TargetDynamic, altComp, nil)

			return true
		}

		if altDef := r.ResolveFunc(altComp, funcName, baseDir); altDef != nil {
			tr.hit(TargetComponent, altComp, altDef)

			return true
		}
	}

	return false
}

// missingMethod is checkMethodOn's answer once comp is known not to declare
// funcName: a component that is not there, a base that is not, or a method
// the code checks for or that comp lacks.
func (r *Resolver) missingMethod(comp, softComp string, call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	funcName, variable := call.FuncName, call.Variable

	// A component that names no file is a different finding from a method a
	// real component lacks, and reporting it as the second hid it. It is
	// almost always a componentResolver producing a path — a broad get$1()
	// catch-all turning getInjectorController() into tass.injectorcontroller —
	// so saying so points at the config rather than at the code.
	if !r.componentExists(comp, baseDir) {
		// Unless a resolver marked dynamicIfMissing produced it: that is a
		// broad pattern's guess, and a guess that names nothing says the value
		// is not one of the components it was written for — not that one is
		// missing. A component named any other way is still reported.
		if softMissing(comp, softComp, pr) {
			tr.addf("%q names no file and came from a dynamicIfMissing resolver — accepted as dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		// Nor a model of a module the workspace does not have: that is the
		// module not being installed.
		if r.uninstalledModule(comp) {
			tr.addf("%q is a module's model and no such module is in the workspace — accepted as dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		// Nor a class in a framework's own namespace the bundled stubs lack:
		// the stubs hold the API the presets need, so their silence is not
		// the framework's.
		if frameworkapi.NamespaceOf(comp) != "" {
			tr.addf("%q is in the %s namespace and the bundled stubs do not hold it — accepted as dynamic", comp, frameworkapi.NamespaceOf(comp))
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		// Nor one the engine provides: `new Query()` is Lucee's or Adobe's
		// own component, whose methods are the engine's to answer for.
		if _, ok := engineComponent(comp); ok {
			tr.addf("%q is a component the engine provides and its source is not in the workspace — accepted as dynamic", comp)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		return "component '" + displayComponent(comp) + "' does not exist (calling '" + funcName + "')"
	}

	if base, implied := r.componentMissingBase(comp, baseDir); base != "" {
		if implied {
			tr.addf("%q's framework base does not resolve (%s) — accepted as dynamic", comp, base)
			tr.hit(TargetDynamic, comp, nil)

			return ""
		}

		return MissingBaseReason(base)
	}

	// Nor one a type check before the call says the object has.
	for _, c := range r.guardedInstanceOf(pr, variable, funcName, int(call.Line)) {
		if def := r.ResolveFunc(c, funcName, baseDir); def != nil {
			tr.addf("%q is called only after the code checks %q is a %q, which declares it", funcName, variable, c)
			tr.hit(TargetComponent, c, def)

			return ""
		}
	}

	// Nor a method the code checks for before calling it.
	if r.guardedByExistsCheck(pr, variable, funcName, int(call.Line)) {
		tr.addf("%q is called only after the code checks %q has it — accepted as dynamic", funcName, variable)
		tr.hit(TargetDynamic, comp, nil)

		return ""
	}

	return "method '" + funcName + "' not found in " + displayComponent(comp)
}

// displayComponent is how a reason names comp. A component named by its file
// — what a function returning `this` returns — is named by the file's name:
// a reason is written into a known-issues file that is committed and read on
// other machines, where an absolute path would not mean anything.
func displayComponent(comp string) string {
	if strings.Contains(comp, "|") {
		// Each alternative of a union, which may mix a file and a dot-path.
		alts := strings.Split(comp, "|")
		for i, alt := range alts {
			alts[i] = displayComponent(alt)
		}

		return strings.Join(alts, "|")
	}

	comp = strings.TrimPrefix(comp, parser.EntityPrefix)

	if filepath.IsAbs(comp) {
		return strings.TrimSuffix(filepath.Base(comp), filepath.Ext(comp))
	}

	return comp
}

// sharedGetterReturn is what a function whose whole body is `return
// application.x` (or another shared scope) returns: what the startup
// templates assign x, the type it already has as a receiver. Mura's
// getPluginManager() and getServiceFactory() are this. Anything else in the
// body withholds it, a call above all, since a call can write the scope:
// TestAbsentOptionalArgumentBoundaries holds the guarded and computed shapes
// to no answer.
func (r *Resolver) sharedGetterReturn(fd *parser.FunctionDef) string {
	method := r.producerFor(fd)
	if method == nil || len(method.body) != 1 || method.body[0].kind != "return" {
		return ""
	}

	path := producerPath(producerUnwrap(producerTokens(method.body[0].expression)))
	if _, shared := splitSharedScope(path); !shared {
		return ""
	}

	dir := filepath.Dir(fd.URI.Path())

	comp := r.startupComponent(path, dir, nil)
	if comp == "" || strings.HasPrefix(comp, "$") {
		return ""
	}

	if p := r.ComponentPath(comp, dir); p != "" {
		return p
	}

	return comp
}

func (r *Resolver) declaredReturnType(fd *parser.FunctionDef) string {
	if path := r.besideDeclaring(fd, fd.ReturnType); path != "" {
		return path
	}

	if native := nativetype.Java(fd.ReturnType); native != "" {
		return native
	}

	return fd.ReturnType
}

// returnViaComponent finishes what the index's parse could not: fd returns
// a variable last assigned `x = recv.m()`, recv holding fd.ReturnVia's
// component, and fd returns what m returns there — "$any" included, as the
// parse would give it with FuncLookup. tassweb's itext.createParagraph
// returns `VARIABLES._itextUtils.createParagraph( … )`, whose branches return
// a string or one of several paragraphs. Depth and budget are carried, so
// two functions returning each other's calls end rather than recurse.
func (r *Resolver) returnViaComponent(fd *parser.FunctionDef, depth int, budget *int) string {
	via := fd.ReturnVia
	if via == nil || !fd.URI.IsFile() || depth >= 8 || *budget <= 0 {
		return ""
	}

	*budget--

	target := r.ResolveFunc(via.Component, via.Method, filepath.Dir(fd.URI.Path()))
	if target == nil {
		return ""
	}

	return r.returnComponentOf(target, depth+1, budget)
}

// unevaluatedReturn is what fd returns when evaluating its body gave no
// answer: what the call it returns returns (returnViaComponent), the element
// of a collection it returns (elementReturn), or "$any"
// when the parse found a branch returning no component (a string or an
// argument passed through), which no choice of the caller's arguments makes
// a component on every path.
func (r *Resolver) unevaluatedReturn(fd *parser.FunctionDef, depth int, budget *int) string {
	if ret := r.returnViaComponent(fd, depth, budget); ret != "" {
		return ret
	}

	if ret := r.elementReturn(fd); ret != "" {
		return ret
	}

	if fd.ReturnsUntyped && fd.ReturnComponent == "$any" {
		return "$any"
	}

	return ""
}

// nativeArgumentFamily is the nativetype family a declared argument type
// names, or "" for a component or a type with no contract.
func nativeArgumentFamily(declared string) string {
	switch t := strings.ToLower(strings.TrimSpace(declared)); t {
	case "query", "struct", "array", "string", "numeric", "boolean", "date", "datetime":
		if t == "datetime" {
			return "date"
		}

		return t
	}

	return ""
}

// topLevelLine reports whether line is outside every function of pr: a
// template's own code, whose assignments run in the order they are written.
func topLevelLine(pr *parser.ParseResult, line uint32) bool {
	for i := range pr.Scopes {
		if int(line) >= pr.Scopes[i].Start && int(line) <= pr.Scopes[i].End {
			return false
		}
	}

	return true
}

// namedRefs is a lookup of the refs for name among refs, run only if the union
// of reaching assignments needs them.
func namedRefs(refs []parser.ComponentRef, name string) func() []*parser.ComponentRef {
	return func() []*parser.ComponentRef {
		var found []*parser.ComponentRef

		for i := range refs {
			if strings.EqualFold(refs[i].Variable, name) {
				found = append(found, &refs[i])
			}
		}

		return found
	}
}
