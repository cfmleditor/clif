// Typing the variables a parse leaves untyped: pending calls, returned variables and the refs that reach a line.

package parser

import (
	"slices"
	"strings"
)

// resolvePendingCalls resolves varName = funcCall(...) assignments against same-file functions.
func (pr *ParseResult) resolvePendingCalls(calls []pendingCall) {
	returnPendings := pr.pendingReturnVars()

	// Resolve ReturnComponent from return var before processing calls
	pr.settleReturnVars(returnPendings, calls)

	pr.refIdx = map[*ComponentRef]*refNames{}
	defer func() { pr.refIdx = nil }()

	funcReturns := pr.funcReturns()
	untyped, typed := pr.typePendingCalls(calls, funcReturns, nil)

	// Later passes: resolve ReturnComponent for functions whose return var was
	// just added by calls, and look again at the calls left untyped, which
	// may assign what such a function returns or call a method on a variable
	// typed by one. A function whose return waited for a call to type its
	// variable settles here, and `variables.binder = buildBinder()` had been
	// left untyped because the call was looked at first. A pass that typed a
	// call earns another look as a settled return does: the calls are typed
	// in file order, so a receiver assigned further down the file
	// (`db = MongoDBConnect( … )` in a later function) had no type when a call
	// on it above was looked at.
	for range maxReturnRounds {
		settled := pr.settleReturnVars(returnPendings, nil)
		if settled == 0 && typed == 0 || len(untyped) == 0 {
			return
		}

		funcReturns = pr.funcReturns()

		var still []int

		for _, j := range untyped {
			if c := &calls[j]; !pr.addPendingRef(c, pr.pendingComponent(c, funcReturns, nil)) {
				still = append(still, j)
			}
		}

		typed = len(untyped) - len(still)
		untyped = still
	}

	pr.settleReturnVars(returnPendings, nil)
}

// typePendingCalls files a ref for each assignment in calls whose call says
// what it returns, and reports the ones nothing typed and how many it typed. A member set is kept
// for applyMemberBindings instead. globals are the file's refs a receiver is
// looked up in, nil for pr's own; see baseVarComponent.
func (pr *ParseResult) typePendingCalls(calls []pendingCall, funcReturns map[string]string, globals []ComponentRef) (untyped []int, typed int) {
	for j := range calls {
		c := &calls[j]

		if c.returnExpr {
			continue
		}

		if c.memberSet {
			pr.memberSets = append(pr.memberSets, *c)

			continue
		}

		// Skip if this variable already has a ref (e.g. from appendResolverRefs)
		if pr.hasRefFor(c) {
			continue
		}

		if pr.addPendingRef(c, pr.pendingComponent(c, funcReturns, globals)) {
			typed++
		} else {
			untyped = append(untyped, j)
		}
	}

	return untyped, typed
}

// maxReturnRounds bounds how many times resolvePendingCalls goes back for the
// calls left untyped once a function's return has settled late: each round
// follows one more step of a call typed by a return typed by a call.
const maxReturnRounds = 4

// funcReturns maps each function's lowercased name to the component it
// returns, inferred or declared.
func (pr *ParseResult) funcReturns() map[string]string {
	out := make(map[string]string, len(pr.Funcs))

	for i := range pr.Funcs {
		f := &pr.Funcs[i]

		comp := f.ReturnComponent
		if comp == "" && isComponentType(f.ReturnType) {
			comp = f.ReturnType
		}

		if comp != "" {
			out[strings.ToLower(f.Name)] = comp
		}
	}

	return out
}

// pendingCallComponent is the component the call c assigns its variable, or
// "" when nothing says. It may clear c.rest, when the chain ends in a mock.
// pendingComponent is what c's assignment holds: a copy's source, or what
// its call returns.
func (pr *ParseResult) pendingComponent(c *pendingCall, funcReturns map[string]string, globals []ComponentRef) string {
	if c.alias {
		return pr.aliasComponent(c, globals)
	}

	return pr.pendingCallComponent(c, funcReturns, globals)
}

func (pr *ParseResult) pendingCallComponent(c *pendingCall, funcReturns map[string]string, globals []ComponentRef) string {
	if c.baseVar == "" && (c.baseLocal || pr.pendingArgumentShadows(c)) {
		return ""
	}

	var comp string
	if c.callsOwnFunction() || pr.receiverIsThisFile(c) {
		comp = funcReturns[strings.ToLower(c.funcName)]
	}

	// Fallback: x = baseVar.method() — assign x same component as baseVar
	if comp == "" && c.baseVar != "" {
		comp = pr.baseVarComponent(c, globals)
	}

	// x = inherited(): a method the file does not declare is its base's,
	// which FuncLookup reaches from the file itself. ContentBox's
	// services write `var c = newCriteria()`, a method of cborm's
	// BaseORMService, and c was untyped. `this.inherited()` and
	// `variables.inherited()` are the same call on the same object.
	selfCall := c.baseVar == "" || strings.EqualFold(c.baseVar, "this") || strings.EqualFold(c.baseVar, "variables")
	if comp == "" && selfCall && pr.FuncLookup != nil && pr.URI.IsFile() {
		comp = pr.FuncLookup(pr.URI.Path(), c.funcName)
	}

	if comp == "" && selfCall && c.expression != "" && pr.FuncLookup != nil && pr.URI.IsFile() {
		comp = pr.FuncLookup(pr.URI.Path(), CallHop(c.expression))
	}

	// Native globals are typed only after declared functions have had priority.
	if comp == "" && c.baseVar == "" && pr.BuiltinReturnLookup != nil {
		declared := false

		for i := range pr.Funcs {
			if strings.EqualFold(pr.Funcs[i].Name, c.funcName) {
				declared = true

				break
			}
		}

		if !declared {
			comp = pr.BuiltinReturnLookup(c.funcName)
		}
	}

	// A mock made through the MockBox a spec holds,
	// getMockBox().createMock(…), is what one made directly is: the class
	// its argument names, whatever the receiver's method is declared to
	// return. A chain going on from it is walked on that class, as qb's specs
	// build their builder with createMock( "qb.models.Query.QueryBuilder" ).init( grammar ).
	if mock, rest, ok := chainedMock(c.rest); ok {
		comp, c.rest = mock, rest
	} else if comp == "" {
		last := c.funcName
		if len(c.rest) > 0 {
			last = callHopName(c.rest[len(c.rest)-1])
		}

		if comp = dynamicCall(last + "()"); comp != "" {
			c.rest = nil // the chain ends in the mock; nothing to walk
		}
	}

	// A chain that decorates what it is made on is made on a mock, and
	// a mock of a class nothing here names is dynamic:
	// `variables.iService = model.init( c ).$( "getCache", x )`, where
	// model is the base class's createMock( annotations.model ).
	if comp == "" && (IsMockDecoration(c.funcName) || slices.ContainsFunc(c.rest, func(hop string) bool { return IsMockDecoration(callHopName(hop)) })) {
		comp, c.rest = "$any", nil
	}

	return comp
}

// chainedMock finds the last hop of a chain that makes a MockBox mock and
// returns what it makes with the hops after it, which are made on the mock.
// A mock of a class no argument names is dynamic, and so is everything
// chained on it.
func chainedMock(rest []string) (mock string, after []string, ok bool) {
	for i, hop := range slices.Backward(rest) {
		expression, isCall := CallExpression(hop)
		if !isCall {
			expression = hop + "()"
		}

		if mock = dynamicCall(expression); mock == "" {
			continue
		}

		if mock == "$any" || i == len(rest)-1 {
			return mock, nil, true
		}

		return mock, rest[i+1:], true
	}

	return "", nil, false
}

// A callable argument is source-owned, even when its name matches a builtin.
// Enclosing named-function arguments also remain visible inside callbacks.
func (pr *ParseResult) pendingArgumentShadows(c *pendingCall) bool {
	for _, scope := range pr.Scopes {
		if int(c.line) < scope.Start || int(c.line) > scope.End {
			continue
		}

		for _, arg := range pr.argsOfFuncAt(scope.Start) {
			if strings.EqualFold(arg.Name, c.funcName) {
				return true
			}
		}
	}

	return false
}

// callsOwnFunction reports whether c calls a function of the file's own
// component, bare or through this., variables. or super., rather than a
// method of some value: `x = arguments.binder.init()` is not the file's
// init(), and ColdBox's Injector, whose init() returns this, was taken for
// the binder it builds.
func (c *pendingCall) callsOwnFunction() bool {
	switch {
	case c.baseVar == "":
		return true
	case strings.EqualFold(c.baseVar, "this"), strings.EqualFold(c.baseVar, "variables"), strings.EqualFold(c.baseVar, "super"):
		return true
	default:
		return false
	}
}

// addPendingRef files the ref c's assignment makes, when comp says what it
// holds, and reports whether it did.
func (pr *ParseResult) addPendingRef(c *pendingCall, comp string) bool {
	if comp == "" {
		return false
	}

	ref := ComponentRef{
		Variable: c.varName, Component: comp, ChainRest: c.rest,
		URI: pr.URI, Line: c.line, This: c.refThis, Rebinds: c.rebinds, BaseGuess: c.baseGuess,
		ChainMethod: guessedMethod(c),
		VisibleFrom: c.visibleFrom, VisibleTo: c.visibleTo,
		ClosureFrom: c.closureFrom, ClosureTo: c.closureTo, HasClosureSpan: c.hasClosureSpan,
	}
	if !c.funcKey.in && !c.hasClosureSpan || c.global {
		pr.ComponentRefs = append(pr.ComponentRefs, ref)
	} else {
		pr.funcRefsMap = appendKeyed(pr.funcRefsMap, c.funcKey, []ComponentRef{ref})
	}

	if pr.flow != nil && c.funcKey.in {
		pr.flow.note(c.varName, c.line, c.block)
	}

	return true
}

// receiverIsThisFile reports whether a pending call is made on a variable
// this file has typed as itself (`var node = this; node = node.getParent()`),
// which callsOwnFunction cannot see from the receiver's name.
func (pr *ParseResult) receiverIsThisFile(c *pendingCall) bool {
	if c.baseVar == "" {
		return false
	}

	self, ok := strings.CutPrefix(string(pr.URI), "file://")
	if !ok || self == "" {
		return false
	}

	ref := pr.firstRef(pr.ComponentRefs, c.baseVar, c.baseScope)
	if ref == nil && c.funcKey.in {
		ref = pr.firstRef(pr.funcRefsMap[c.funcKey], c.baseVar, c.baseScope)
	}

	return ref != nil && ref.Component == self
}

// returnPending is a function whose return type is whatever a variable it
// returns holds, still to be looked up.
type returnPending struct {
	funcIdx int
	varName string
	funcKey fnKey
	line    uint32 // the return's
}

// pendingReturnVars lists the functions that return a variable and have no
// return component yet.
func (pr *ParseResult) pendingReturnVars() []returnPending {
	var out []returnPending

	scopeAt := scopeIndex(pr.Scopes, true)

	for i := range pr.Funcs {
		f := &pr.Funcs[i]
		if f.ReturnComponent != "" || f.returnVar == "" {
			continue
		}

		// `return this;` returns the component that declares the function:
		// its file, named by path, which is how the resolver takes it.
		if f.returnVar == returnsThis {
			if path, ok := strings.CutPrefix(string(f.URI), "file://"); ok && path != "" {
				f.ReturnComponent = pr.componentReturnFor(f, path)
			}

			continue
		}

		if k := scopeAt(int(f.Line)); k >= 0 {
			scope := pr.Scopes[k]
			out = append(out, returnPending{
				funcIdx: i,
				varName: f.returnVar,
				funcKey: funcKey(scope.Start, scope.End),
				line:    f.returnLine,
			})
		}
	}

	return out
}

// returnsThis is the returnVar of a function whose return is `this` alone,
// a name no variable can have.
const returnsThis = "$this"

// returnsCallOn prefixes the returnVar of a function returning a call made
// on a variable — `return di.toClazz( c )` — for which the variable's
// component answers only when it is dynamic. It holds a colon, which no
// variable name can.
const returnsCallOn = "$call:"

// returnsVariablesVar and returnsThisVar prefix the returnVar of a function
// returning a variable read through its scope — `return variables.print;`,
// `return this.other;`. Such a variable is the component's, never a local of
// the function, and `this.x` and `variables.x` are separate stores, so each
// is read only from the component's refs made through its own scope. Like
// returnsCallOn, each holds a colon, which no variable name can.
const (
	returnsVariablesVar = "$variables:"
	returnsThisVar      = "$this:"
)

// scopedReturnVar is the returnVar of a function returning name read through
// scope, which is RefThis or RefVariables.
func scopedReturnVar(scope RefScope, name string) string {
	if scope == RefThis {
		return returnsThisVar + name
	}

	return returnsVariablesVar + name
}

// returnedVar is the variable a returnVar names, whether the function
// returns a call made on it rather than the variable itself, and the scope
// it was read through: RefAny for a bare name, which may be a local.
func returnedVar(returnVar string) (name string, called bool, scope RefScope) {
	if name, ok := strings.CutPrefix(returnVar, returnsCallOn); ok {
		return name, true, RefAny
	}

	if name, ok := strings.CutPrefix(returnVar, returnsVariablesVar); ok {
		return name, false, RefVariables
	}

	if name, ok := strings.CutPrefix(returnVar, returnsThisVar); ok {
		return name, false, RefThis
	}

	return returnVar, false, RefAny
}

// returnedComponent is what a function returns, given the component its
// return variable holds: that component, or for a call made on the variable,
// "$any" when the variable is dynamic and nothing otherwise — what a method
// returns is not the component it is a method of.
func returnedComponent(comp string, called bool) string {
	if !called || comp == "$any" {
		return comp
	}

	return ""
}

// settleReturnVars gives each pending function without a return component
// the component its return variable holds, when the ref reaching the return
// says. A pending call in calls that assigns the variable between that ref
// and the return is what the variable holds instead, so such a function waits
// for the pass after calls are typed, which passes none. It reports how many
// functions it settled.
func (pr *ParseResult) settleReturnVars(pending []returnPending, calls []pendingCall) (settled int) {
	for _, rp := range pending {
		if pr.Funcs[rp.funcIdx].ReturnComponent != "" {
			continue
		}

		name, called, scope := returnedVar(rp.varName)

		// A scoped name is the component's variable, not a local of the
		// function's. Of its refs, every one that may reach the return is
		// read (flowBlocks.reaching), and they must agree.
		refs, admit := pr.ComponentRefs, scope.Admits

		if scope == RefAny {
			// A closure's local of the same name is not what the function
			// returns. An unscoped name the function never declared is a
			// variables-scope one, as settleReturnComponent reads it: a
			// pending call assigning it settles at component level.
			refs, admit = pr.funcRefsMap[rp.funcKey], func(ref *ComponentRef) bool { return ref.VisibleTo == 0 && !ref.HasClosureSpan }
			if refReaching(refs, name, rp.line, admit) == nil {
				refs, admit = pr.ComponentRefs, func(*ComponentRef) bool { return true }
			}
		}

		reaching := pr.flow.reaching(refs, name, rp.line, admit)

		// A receiver's own type, guessed for a call the parse could not look
		// up, is not what the function returns; the resolver reads the body,
		// or, when one such call alone reaches the return, finishes it.
		if slices.ContainsFunc(reaching, func(ref *ComponentRef) bool { return ref.BaseGuess }) {
			if g := reaching[0]; len(reaching) == 1 && !called && g.ChainMethod != "" && len(g.ChainRest) == 0 {
				pr.Funcs[rp.funcIdx].ReturnVia = &ReturnCall{Component: g.Component, Method: g.ChainMethod}
			}

			continue
		}

		if len(reaching) > 0 && !assignedBetween(calls, name, reaching[0].Line, rp.line) {
			comp := agreedComponent(reaching, pr.settledComponent)
			if comp != "" && pr.flow.untypedReaches(reaching, name, pr.Funcs[rp.funcIdx].Line, rp.line) {
				comp = "$any" // a branch assigned it something untyped
				pr.Funcs[rp.funcIdx].ReturnsUntyped = true
			}

			if scope == RefAny {
				comp = returnedComponent(comp, called)
			}

			pr.Funcs[rp.funcIdx].ReturnComponent = pr.componentReturnFor(&pr.Funcs[rp.funcIdx], comp)
		}

		if pr.Funcs[rp.funcIdx].ReturnComponent != "" {
			settled++
		}
	}

	return settled
}

// refReaching is the ref for name in refs that reaches line: of those admit
// accepts, the latest assigned at or before it, the later of two on one line.
// When every one is assigned after it, the first, as fileLevelRef reads a
// forward reference. `var x = new A(); x = new B(); return x;` returns a B.
func refReaching(refs []ComponentRef, name string, line uint32, admit func(*ComponentRef) bool) *ComponentRef {
	var first, best *ComponentRef

	for i := range refs {
		ref := &refs[i]
		if !strings.EqualFold(ref.Variable, name) || !admit(ref) {
			continue
		}

		if first == nil {
			first = ref
		}

		if ref.Line <= line && (best == nil || ref.Line >= best.Line) {
			best = ref
		}
	}

	if best != nil {
		return best
	}

	return first
}

// assignedBetween reports whether one of calls assigns name after line from
// and at or before line to.
func assignedBetween(calls []pendingCall, name string, from, to uint32) bool {
	for i := range calls {
		c := &calls[i]
		if !c.returnExpr && !c.memberSet && c.line > from && c.line <= to && strings.EqualFold(c.varName, name) {
			return true
		}
	}

	return false
}

// hasRefFor reports whether the assignment a pending call makes already has
// a ref, in its function or at file level: one for the same variable on the
// same line.
//
// Only a ref in the pending call's own scope counts: a sibling closure's `t`
// is another variable, and letting it stand for this one left this `t`
// untyped whenever an earlier test in the spec declared a `t` of its own.
// Nor does a ref from another assignment: `var x = new A(); x = makeB();`
// left x an A for good, and a return of x after it returning one.
func (pr *ParseResult) hasRefFor(c *pendingCall) bool {
	if c.funcKey.in {
		local := pr.refsOnLine(pr.funcRefsMap[c.funcKey], c.line, func(ref *ComponentRef) bool {
			return strings.EqualFold(ref.Variable, c.varName) &&
				ref.VisibleFrom == c.visibleFrom && ref.VisibleTo == c.visibleTo && ref.HasClosureSpan == c.hasClosureSpan && ref.ClosureFrom == c.closureFrom && ref.ClosureTo == c.closureTo
		})
		if local {
			return true
		}
	}

	return pr.refsOnLine(pr.ComponentRefs, c.line, func(ref *ComponentRef) bool {
		return strings.EqualFold(ref.Variable, c.varName)
	})
}

// baseVarComponent is the component `x = baseVar.method()` gives x: the
// method's declared return type when FuncLookup can say, else baseVar's own
// component, or "$any" when the method is known to declare none.
//
// An argument or a local is the function's own, so `arguments.binder.init()`
// is not read from the component's refs: ColdBox's Injector holds a
// `variables.binder` built from the argument, and taking that for it made the
// argument the injector. Nor is an unscoped name the function declared:
// Lucee's mail tests pass `smtpServer` to a helper whose argument has that
// name, and the helper's `it = smtpServer.getReceivedEmail()` was typed by
// the callers' variables-scope one, but only once those had refs, so the
// answer depended on which parse got there first.
//
// globals are the file-level refs to look in, nil for pr's own: a function
// body parsed on its own (funcRefsUncached) reads the file's.
func (pr *ParseResult) baseVarComponent(c *pendingCall, globals []ComponentRef) string {
	var comp string

	c.baseGuess = false

	if globals == nil {
		globals = pr.ComponentRefs
	}

	if !c.baseLocal {
		if ref := pr.firstRef(globals, c.baseVar, c.baseScope); ref != nil {
			comp = pr.settledComponent(ref)
		}
	}

	if comp == "" && c.funcKey.in {
		if ref := pr.firstRef(pr.funcRefsMap[c.funcKey], c.baseVar, c.baseScope); ref != nil {
			comp = pr.settledComponent(ref)
		}
	}

	// If FuncLookup is available, prefer the called method's own declared
	// return type over the "same as baseVar" guess — it may differ from the
	// receiver's type. If the method declares no component return type, don't
	// propagate the base variable's component at all.
	// `x = m.$( "get", 1 )`: a decoration returns the mock it is called on.
	if comp != "" && IsMockDecoration(c.funcName) {
		return comp
	}

	if comp != "" && c.funcName != "" && pr.FuncLookup != nil {
		if ret := pr.FuncLookup(comp, c.funcName); ret != "" {
			return ret
		}

		if c.expression != "" {
			if ret := pr.FuncLookup(comp, CallHop(c.expression)); ret != "" {
				return ret
			}
		}

		return "$any"
	}

	// A framework object an injection typed hands back other things: an
	// injector's getInstance() is not an injector. The guess is for a
	// project's own fluent components, as a preset's name-only resolvers keep
	// it from the objects they type.
	if isInjectedFrameworkComponent(comp) {
		return ""
	}

	c.baseGuess = comp != ""

	return comp
}

// rebinds reports whether an assignment to varName whose right-hand side is
// a call on recv assigns the variable the call is made on: `x = x.next()`,
// `variables.x = variables.x.next()`. An argument of the same name is another
// variable, and so is a member: `x = y.x.next()` is made on y.
func rebinds(varName, recv string) bool {
	if varName == "" || recv == "" {
		return false
	}

	if scope, name, ok := strings.Cut(recv, "."); ok {
		if strings.Contains(name, ".") {
			return false
		}

		switch {
		case strings.EqualFold(scope, "local"), strings.EqualFold(scope, "variables"), strings.EqualFold(scope, "this"):
			recv = name
		default:
			return false
		}
	}

	return strings.EqualFold(StripReceiverScope(varName), recv)
}

// readThroughLocal reports whether the receiver recv is written
// `arguments.x` or `local.x`.
func readThroughLocal(recv string) bool {
	scope, name, ok := strings.Cut(recv, ".")

	return ok && name != "" && (strings.EqualFold(scope, "arguments") || strings.EqualFold(scope, "local"))
}

// refNames indexes a refs slice by lowercased variable name: the first ref of
// each name assigned through this., and the first assigned otherwise, which
// is all RefScope.Admits tells apart. n is how many refs it covers.
type refNames struct {
	n     int
	this  map[string]int
	other map[string]int
	// lines lists, for each line, the refs on it, for hasRefFor.
	lines map[uint32][]int
}

// firstRef is firstRefIn, answered from an index while resolvePendingCalls
// runs. It looked a receiver up by scanning every ref the file had, once per
// pending call, so a function of n locals each assigned from the one before
// (`v2 = v1.next()`) was quadratic: 4,000 of them took 330ms. The index is
// keyed by the slice's first element and extended as refs are appended,
// which is all the pass does to them; it is dropped when the pass ends,
// because a reparse refills ComponentRefs in place.
func (pr *ParseResult) firstRef(refs []ComponentRef, name string, scope RefScope) *ComponentRef {
	if pr.refIdx == nil || len(refs) <= maxRefScan {
		return firstRefIn(refs, name, scope)
	}

	idx := pr.refIndex(refs)

	var buf foldScratch

	key := buf.lowerFold(name)
	t, inThis := idx.this[string(key)]
	o, inOther := idx.other[string(key)]

	switch {
	case scope == RefThis && inThis:
		return &refs[t]
	case scope == RefVariables && inOther:
		return &refs[o]
	case scope != RefThis && scope != RefVariables && (inThis || inOther):
		if inThis && (!inOther || t < o) {
			return &refs[t]
		}

		return &refs[o]
	}

	return nil
}

// refIndex is refs' index for firstRef and refsOnLine, brought up to date
// with any refs appended since it was last asked.
func (pr *ParseResult) refIndex(refs []ComponentRef) *refNames {
	idx := pr.refIdx[&refs[0]]
	if idx == nil || idx.n > len(refs) {
		idx = &refNames{this: map[string]int{}, other: map[string]int{}, lines: map[uint32][]int{}}
		pr.refIdx[&refs[0]] = idx
	}

	for i := idx.n; i < len(refs); i++ {
		m := idx.other
		if refs[i].This {
			m = idx.this
		}

		key := strings.ToLower(refs[i].Variable)
		if _, ok := m[key]; !ok {
			m[key] = i
		}

		idx.lines[refs[i].Line] = append(idx.lines[refs[i].Line], i)
	}

	idx.n = len(refs)

	return idx
}

// refsOnLine calls fn with each ref in refs on line, stopping when fn says
// so, from the index while resolvePendingCalls runs. hasRefFor scanned every
// ref for every pending call, which was quadratic in a function of
// thousands of assignments.
func (pr *ParseResult) refsOnLine(refs []ComponentRef, line uint32, fn func(*ComponentRef) bool) bool {
	if pr.refIdx == nil || len(refs) <= maxRefScan {
		for i := range refs {
			if refs[i].Line == line && fn(&refs[i]) {
				return true
			}
		}

		return false
	}

	for _, i := range pr.refIndex(refs).lines[line] {
		if fn(&refs[i]) {
			return true
		}
	}

	return false
}

// maxRefScan is the most refs firstRef scans before it indexes them.
const maxRefScan = 32

// firstRefIn is firstRefNamed among the refs scope admits.
func firstRefIn(refs []ComponentRef, name string, scope RefScope) *ComponentRef {
	for i := range refs {
		if strings.EqualFold(refs[i].Variable, name) && scope.Admits(&refs[i]) {
			return &refs[i]
		}
	}

	return nil
}

// firstRefNamed is the first ref in refs for the variable name, compared
// case-insensitively, or nil.
func firstRefNamed(refs []ComponentRef, name string) *ComponentRef {
	for i := range refs {
		if strings.EqualFold(refs[i].Variable, name) {
			return &refs[i]
		}
	}

	return nil
}
