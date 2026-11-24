package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// acceptsScopeReference reports whether a call qualified by a scope word is a
// call of a function the scope holds, which a call's receiver cannot be: a
// scope is not a component.
func (r *Resolver) acceptsScopeReference(call *parser.CallSite, pr *parser.ParseResult, tr *callTrace) bool {
	funcName := call.FuncName
	variable := call.Variable

	// Bare VARIABLES qualifier (no further dot, e.g. "VARIABLES.someName(...)") —
	// a scope can't itself be a component ref, so treating "VARIABLES" as the thing
	// needing a ComponentRef is a category error. What's actually being called is a
	// property assigned onto the scope (e.g. "VARIABLES.someName = ARGUMENTS.callback;
	// ... VARIABLES.someName(...)"), a function-reference value passed in and stored —
	// same reasoning as the ARGUMENTS-as-function-reference case below, just for a
	// VARIABLES-scoped property instead of a bare argument. Accept if funcName was
	// ever assigned in VARIABLES scope anywhere in the file (assignments inside any
	// function are visible from every other function, unlike a var-scoped local).
	if strings.EqualFold(variable, "VARIABLES") {
		tr.addf("VARIABLES.%s called as a function reference — checking for a VARIABLES-scoped assignment", funcName)

		if pr.HasScopedAssignment(parser.ScopeVariables, funcName) {
			tr.addf("%q was assigned in VARIABLES scope — treating as a call through a function-reference property", funcName)

			return true
		}
	}

	// ARGUMENTS qualifier — funcName is being called as a function-reference argument.
	// These are dynamic (type="any"), so we can't verify them statically; accept if the
	// argument is declared in the enclosing function.
	if strings.EqualFold(variable, "ARGUMENTS") {
		tr.addf("ARGUMENTS.%s called as a function reference — checking caller %q's argument list", funcName, call.Caller)

		if declaresArgument(pr, call.Caller, funcName) {
			return true
		}
	}

	return false
}

// matchReceiverResolvers is the component the configured resolvers give a
// receiver nothing in the file types: by the variable's name, then by the
// whole line's text (a chain like x.method().prop.func()). soft is the
// component when a dynamicIfMissing resolver gave it. accepted says a resolver
// marked noFollow gave it, which takes the call without looking for the method; the
// caller records the target.
func (r *Resolver) matchReceiverResolvers(call *parser.CallSite, tr *callTrace) (comp, softComp string, accepted bool) {
	variable := call.Variable

	var noFollow, soft bool

	comp, noFollow, soft = r.matchResolver(variable, tr, func(c, desc string, nf bool) {
		tr.addf("resolved %q to %q via componentResolver matching the variable name [%s] (noFollow=%v)",
			variable, c, desc, nf)
	})
	if soft {
		softComp = comp
	}

	if noFollow && comp != "" {
		return comp, softComp, true
	}

	if comp == "" && call.Text != "" {
		comp, noFollow, soft = r.matchResolver(call.Text, tr, func(c, desc string, nf bool) {
			tr.addf("resolved %q to %q via componentResolver matching the full line text %q [%s] (noFollow=%v)",
				variable, c, call.Text, desc, nf)
		})
		if soft {
			softComp = comp
		}

		if noFollow && comp != "" {
			return comp, softComp, true
		}
	}

	return comp, softComp, false
}

// lateReceiverComponent is the answer for a receiver every other step left
// untyped: a loop element, a view's prc or rc, then what its arguments' callers
// pass and what init stores. They come last so an inference never turns a call
// something else accepted into a finding.
func (r *Resolver) lateReceiverComponent(call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) string {
	variable := call.Variable
	funcName := call.FuncName

	comp := ""

	if comp == "" {
		comp = r.loopElement(call, pr, baseDir, tr)
	}

	if comp == "" {
		comp = r.viewPrc(variable, funcName, pr, tr)
	}

	if comp == "" {
		comp = r.fw1ViewRc(variable, funcName, pr, tr)
	}

	// Last, an untyped argument is what every caller passes it, when the scan
	// was given the files to look through. After every other answer, so an
	// inference never turns a call something else accepted into a finding.
	if comp == "" {
		// callers: a caller passing on an argument of its own is typed by its
		// own callers in turn (Mura's gateways take what the manager forwards).
		if comp = r.argumentFromCallers(variable, call.Caller, pr, lookupCtx{callers: true}); comp != "" {
			tr.addf("resolved %q to %q: what every caller of %s passes it", variable, comp, call.Caller)
		}
	}

	// A receiver built from an argument (`var item = arguments.iterator
	// .next()`, `var result = arguments.filter`) is looked up again with
	// inference allowed inside it: only here, for a call nothing else
	// answered, so the cost is that of the calls left unresolved. Without
	// files to infer from it would only repeat the lookup.
	if comp == "" && len(r.InferArgsFiles) > 0 {
		if comp, _ = r.receiverComponentD(variable, call.Line, call.Caller, funcName, pr, baseDir, nil, lookupCtx{call: call, callers: true}); comp != "" {
			tr.addf("resolved %q to %q: looked up again with what its arguments' callers pass", variable, comp)
		}
	}

	if comp == "" {
		if comp = r.initArgMember(variable, pr, lookupCtx{}); comp != "" {
			tr.addf("resolved %q to %q: init stores its argument there, and every construction passes it that", variable, comp)
		}
	}

	return comp
}

// hopCallReturn is what a hop's own call returns: the configured bean
// rules' answer for a factory call (the container's getBean( "content" ),
// whose construction the policy spells), or the expression's own.
func (r *Resolver) hopCallReturn(fd *parser.FunctionDef, expression, baseDir, comp string) string {
	if rc, _, _ := parser.ResolveFromCallMatch(expression, r.Resolvers); rc != "" {
		return rc
	}

	return r.expressionReturn(fd, expression, baseDir, comp)
}

// mockHop is what a chain holds at a hop as far as MockBox is concerned. A hop
// that makes a mock makes one of the class it names, whatever it is called on
// (getMockBox().createMock( "X" ).init()), and made reports it: the hop is
// done. A decoration that reports on a mock ($callLog()) leaves the rest of the
// chain dynamic. A mock reaching a hop is its class, or dynamic where that
// names none.
func (r *Resolver) mockHop(comp, hop, expression string, callHop bool, baseDir string, tr *callTrace) (string, bool) {
	if mock := parser.MockCall(expression); callHop && mock != "" {
		tr.addf("chain hop %q makes a MockBox mock: %q", hop, mock)

		return mock, true
	}

	if parser.IsMockReport(hop) {
		tr.addf("chain hop %q reports on a MockBox mock and does not return it", hop)

		return "$any", false
	}

	if cls, ok := strings.CutPrefix(comp, parser.MockPrefix); ok {
		if r.ComponentPath(cls, baseDir) == "" {
			return "$any", false
		}

		return cls, false
	}

	return comp, false
}

// walkHops follows call.Chain from comp through each hop's return type. It
// reports the component the chain reaches, or done with the answer when a hop
// settles the call itself.
func (r *Resolver) walkHops(comp, softComp string, call *parser.CallSite, pr *parser.ParseResult, baseDir string, tr *callTrace) (reached, soft, reason string, done bool) {
	funcName := call.FuncName

	for i, hop := range call.Chain {
		// A hop carries its call's arguments when the parser could read
		// them; every check and every reason below is about its name.
		expression, callHop := parser.CallExpression(hop)
		if callHop {
			hop = parser.CallHopName(hop)
		}

		var made bool
		if comp, made = r.mockHop(comp, hop, expression, callHop, baseDir, tr); made {
			continue
		}

		// A hop that returned "$any" makes the rest of the chain dynamic.
		// Walking on asked "$any" for the next method, which it can never
		// define: getSandBox("x").getAttendanceObj().getLog() was reported
		// as getAttendanceObj missing from $any. The accept after the walk
		// is the answer for a dynamic receiver.
		if comp == "$any" {
			tr.addf("chain hop %q is on a dynamic ($any) value — the rest of the chain is dynamic", hop)

			break
		}

		// A registry family is its members' set, which no single hop's
		// declared return type can name: what it returns is an array of the
		// element the members' own files prove, and the chain past the
		// getter runs on that array, not on the family. Dynamic, once the
		// hop itself is every family's method.
		if paths := familyPaths(comp); len(paths) > 0 {
			hopCall := *call
			hopCall.FuncName = hop

			if why := r.checkFamiliesMethod(familyRefs(comp), &hopCall, pr, tr); why != "" {
				return comp, softComp, why + " (chain to '" + funcName + "')", true
			}

			tr.addf("chain hop %q is on a registry family (%s) — the rest of the chain is dynamic", hop, displayFamilyMembers(paths))
			tr.hit(TargetDynamic, "", nil)

			return comp, softComp, "", true
		}

		if strings.HasPrefix(comp, "$builtin.") {
			ret, ok := docs.LookupBuiltinMethodReturn(comp, hop)
			if !ok {
				return comp, softComp, "method '" + hop + "' not found on builtin " + comp[9:] + " (chain to '" + funcName + "')", true
			}

			tr.addf("native chain hop %q on %q returns %q", hop, comp, ret)
			comp = ret

			continue
		}

		if property, ok := parser.PropertyName(hop); ok {
			ret, soft, reason, done := r.propertyHop(comp, property, call, i, pr, baseDir, tr)
			if done {
				return comp, softComp, reason, true
			}

			if soft {
				softComp = ret
			}

			comp = ret

			continue
		}

		fd := r.ResolveFunc(comp, hop, baseDir)

		if fd == nil {
			if next := r.undeclaredHopReturn(comp, hop, baseDir, tr); next != "" {
				comp = next

				continue
			}

			return comp, softComp, r.missingChainHop(comp, softComp, hop, funcName, pr, baseDir, tr), true
		}

		ret, noFollow, soft := r.hopReturn(comp, hop, hopArgsText(expression), fd, baseDir, tr)
		if ret == "" && !callHop {
			ret = r.expressionReturn(fd, factoryCallExpression(pr.Content, hop, int(call.Line)), baseDir, comp)
		}

		if ret == "" && callHop {
			ret = r.hopCallReturn(fd, expression, baseDir, comp)
		}

		if soft {
			softComp = ret
		}

		if noFollow && ret != "" {
			tr.hit(TargetDynamic, ret, nil)

			return comp, softComp, "", true
		}

		if ret == "" {
			next, reason, done := r.untypedHop(fd, comp, hop, call, i, tr)
			if done {
				return comp, softComp, reason, true
			}

			ret = next
		}

		comp = ret
	}

	return comp, softComp, "", false
}

// undeclaredHopReturn is what a chain hop comp does not declare returns when
// something still says, or "". A decoration returns the mock it is called
// on, so the chain goes on from the same component: model.$( "getCache", c
// ).getFoo() is checked against model's class. Slatwall's BaseService
// answers this.newProduct() and its family with an entity.
func (r *Resolver) undeclaredHopReturn(comp, hop, baseDir string, tr *callTrace) string {
	if mockDecoration(hop) {
		tr.addf("chain hop %q is a method MockBox adds to a mock, and returns the mock — %q", hop, comp)

		return comp
	}

	e := r.slatwallGatewayReturn(r.ComponentPath(comp, baseDir), hop, baseDir)
	if e != "" {
		tr.addf("chain hop %q is answered by Slatwall's BaseService onMissingMethod, which returns %q", hop, e)
	}

	return e
}

// untypedHop decides a chain hop whose function declares no component: what
// it returns when something else says (next), or the call's answer (done).
func (r *Resolver) untypedHop(fd *parser.FunctionDef, comp, hop string, call *parser.CallSite, i int, tr *callTrace) (next, reason string, done bool) {
	funcName := call.FuncName

	// What an untyped hop returns is a mock when the next call on it is one
	// MockBox adds: `c.getRequestService().$( "getContext", x )`.
	after := funcName
	if i+1 < len(call.Chain) {
		after = parser.CallHopName(call.Chain[i+1])
	}

	if mockDecoration(after) {
		tr.addf("chain hop %q declares no component, and %q is called on what it returns — a mock, so the rest of the chain is dynamic", hop, after)
		tr.hit(TargetDynamic, "", nil)

		return "", "", true
	}

	if reason, done := r.closureStructHops(fd, comp, hop, call, i, tr); done {
		return "", reason, true
	}

	if ret := r.initArgGetter(fd); ret != "" {
		tr.addf("chain hop %q returns the property init stores from its argument, which every construction passes as %q", hop, ret)

		return ret, "", false
	}

	if r.engineValueReturn(fd, comp) {
		tr.addf("chain hop %q returns what a built-in function hands back — the rest of the chain is dynamic", hop)
		tr.hit(TargetDynamic, "", nil)

		return "", "", true
	}

	return "", "method '" + hop + "' in " + displayComponent(comp) + " has no component return type (chain to '" + funcName + "')", true
}

// propertyHop is walkHops for a property hop: what the property holds, from
// the component's typed properties or else a componentResolver matched
// against the property read, or done with the answer when that settles the
// call.
func (r *Resolver) propertyHop(comp, property string, call *parser.CallSite, i int, pr *parser.ParseResult, baseDir string, tr *callTrace) (ret string, soft bool, reason string, done bool) {
	if ret = r.publicPropertyComponent(comp, property, baseDir); ret != "" {
		return ret, false, "", false
	}

	var noFollow bool

	// The parse may have folded the calls before the property into
	// call.Component, so the source line is asked too.
	for _, text := range []string{propertyHopText(call, i, property), propertyReadText(lineOfContent(pr.Content, int(call.Line)), property)} {
		if text == "" {
			continue
		}

		if ret, noFollow, soft = r.propertyResolver(text, property, tr); ret != "" {
			break
		}
	}

	switch {
	case ret == "":
		return "", false, "property '" + property + "' in " + displayComponent(comp) + " has no component type (chain to '" + call.FuncName + "')", true
	case noFollow:
		tr.hit(TargetDynamic, ret, nil)

		return "", false, "", true
	}

	return ret, soft, "", false
}

// propertyHopText is a property hop as a resolver sees it: the call or
// variable the property is read from, then the property. A struct a factory
// fills at run time is typed nowhere in the source — tassweb's
// getSubServices() is one key per subservice, each a component created from
// a list — so the property's component can only come from configuration,
// `{"match": "getSubServices().$1", "resolve": "subservices.${1:lower}.service"}`,
// as a method hop's can from one matched against `hop()`.
func propertyHopText(call *parser.CallSite, i int, property string) string {
	if i == 0 {
		return call.Variable + "." + property
	}

	prev := call.Chain[i-1]
	if name, ok := parser.PropertyName(prev); ok {
		return propertyHopText(call, i-1, name) + "." + property
	}

	expression, callHop := parser.CallExpression(prev)
	if !callHop {
		expression = prev + "()"
	}

	return expression + "." + property
}

// propertyResolver is the componentResolver answer for a property read, or
// "" when the answer does not depend on the property. A resolver's prefix is
// found anywhere in the text, so the catch-all `get$1()` matches
// `foo.getBar().baz` on its `getBar()` and would type baz as getBar's
// component; an answer the text without the property gives too is that.
func (r *Resolver) propertyResolver(text, property string, tr *callTrace) (comp string, noFollow, soft bool) {
	comp, noFollow, soft = r.matchResolver(text, nil, nil)
	if comp == "" {
		return "", false, false
	}

	receiver := strings.TrimSuffix(text, text[len(text)-len(property):])
	receiver = strings.TrimSuffix(receiver, ".")

	if without, _, _ := r.matchResolver(receiver, nil, nil); without == comp {
		return "", false, false
	}

	_, _, idx := parser.ResolveFromCallMatch(text, r.Resolvers)
	tr.addf("property hop %q: resolved %q to %q via %s", property, text, comp, r.describeResolver(idx))

	return comp, noFollow, soft
}

// lineOfContent is the 0-based line n of content, or "".
func lineOfContent(content string, n int) string {
	for range n {
		i := strings.IndexByte(content, '\n')
		if i < 0 {
			return ""
		}

		content = content[i+1:]
	}

	line, _, _ := strings.Cut(content, "\n")

	return line
}

// propertyReadText is the expression that reads property on line, from the
// start of its receiver through the property: `a.getB( x ).prop` for a line
// holding `a.getB( x ).prop.c()`. "" when the line reads it nowhere, or more
// than once, since the two reads could be of different receivers.
func propertyReadText(line, property string) string {
	end := -1

	for from := 0; ; {
		i := parser.IndexFoldFrom(line, "."+property, from)
		if i < 0 {
			break
		}

		from = i + 1

		after := i + 1 + len(property)
		if after < len(line) && lex.IsIdent(line[after]) {
			continue
		}

		if end >= 0 {
			return ""
		}

		end = after
	}

	if end < 0 {
		return ""
	}

	start := end - len(property) - 1

	for start > 0 {
		j := start

		if line[j-1] == ')' {
			depth := 0

			for j > 0 {
				j--

				switch line[j] {
				case ')':
					depth++
				case '(':
					depth--
				}

				if depth == 0 {
					break
				}
			}

			if depth != 0 {
				return ""
			}
		}

		k := j
		for k > 0 && lex.IsIdent(line[k-1]) {
			k--
		}

		if k == j {
			// An argument list with no name before it was not walked
			// correctly (a parenthesis inside a string); decline.
			if j < start {
				return ""
			}

			break
		}

		start = k
		if start == 0 || line[start-1] != '.' {
			break
		}

		start--
	}

	return strings.TrimPrefix(line[start:end], ".")
}

// hopArgsText is a hop's call expression when it passes arguments, "" when it
// passes none or the parser did not carry the call.
func hopArgsText(expression string) string {
	open := strings.IndexByte(expression, '(')
	if open < 0 || strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(expression[open+1:]), ")")) == "" {
		return ""
	}

	return strings.TrimSpace(expression)
}

// hopReturn is chainHopReturn, and failing that what the receiver binds the
// method to: a cborm service's entity, a Wheels model's own type.
func (r *Resolver) hopReturn(comp, hop, callText string, fd *parser.FunctionDef, baseDir string, tr *callTrace) (ret string, noFollow, soft bool) {
	ret, noFollow, soft = r.chainHopReturn(comp, hop, callText, fd, tr)
	if sub := r.selfTyped(comp, baseDir, fd, ret); sub != "" {
		tr.addf("chain hop %q returns the class declaring it, and is called on its subclass %q", hop, sub)
		ret = sub
	}

	if ret == "" {
		if e := r.receiverReturn(r.ComponentPath(comp, baseDir), hop); e != "" {
			tr.addf("chain hop %q returns what it is called on binds it to: %q", hop, e)
			ret = e
		}
	}

	return ret, noFollow, soft
}
