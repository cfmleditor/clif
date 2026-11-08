// The tags that assign, call or return: cfset, cfobject, cfinvoke, cfreturn and cfproperty, and typing their right-hand sides.

package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
)

// parseReadOnlyScopeSet records `<cfset application.x = ...>` and its siblings,
// returning whether it matched so parseCFSet's switch can use it as a case.
//
// One helper over a case per scope: the scopes behave identically here, and nine
// near-identical branches is the shape that acquires an inconsistency nobody
// notices.
func (p *tagParser) parseReadOnlyScopeSet(inner string, line int) bool {
	dot := strings.IndexByte(inner, '.')
	if dot <= 0 {
		return false
	}

	scope, ok := readOnlyScopePrefixes[strings.ToLower(strings.TrimSpace(inner[:dot]))]
	if !ok {
		return false
	}

	name, rhs := splitAssign(inner[dot+1:])
	if name == "" {
		return false
	}

	p.vars = append(p.vars, VarDef{Name: name, Scope: scope, Line: conv.Uint32(line)})

	// The shared scopes hold components as well as data, and the script parser
	// follows their right-hand side (sharedScopeOf): `<cfset application.svc =
	// new Svc()>` left the tag form untyped, so a call on application.svc in the
	// same function had no edge where the script form had one.
	if isSharedScope(scope) && rhs != "" {
		p.forceGlobal = true
		p.checkSetRHSStr(rhs, name, line)
		p.forceGlobal = false
	}

	return true
}

// isSharedScope reports whether scope is one of the four the script parser
// dispatches to parseScopedVar with a scope of their own (sharedScopeOf).
func isSharedScope(scope Scope) bool {
	switch scope {
	case ScopeRequest, ScopeSession, ScopeApplication, ScopeServer:
		return true
	default:
		return false
	}
}

// parseScopedAttrVar records the variable a tag attribute declares —
// `<cfparam name="url.x">` and `<cfloop index="variables.i">`. Both write a
// scope-qualified name into an attribute, and both create the variable rather
// than referring to one, which is what makes them definition targets.
//
// An unqualified value (`<cfloop index="i">`) lands in the variables scope,
// which is where CFML puts it.
func (p *tagParser) parseScopedAttrVar(tag, attr string, line int) {
	value := getAttr(tag, attr)
	if value == "" {
		return
	}

	scope := ScopeVariables
	name := value

	if dot := strings.IndexByte(value, '.'); dot > 0 {
		if sc, ok := ScopeForPrefix(value[:dot]); ok {
			scope = sc
			name = value[dot+1:]
		}
	}

	if name == "" || strings.ContainsAny(name, ".[#") {
		return
	}

	p.vars = append(p.vars, VarDef{Name: name, Scope: scope, Line: conv.Uint32(line)})
}

// parseCFSet handles <cfset var x = ...>, <cfset local.x = ...>, etc.
func (p *tagParser) parseCFSet(tag string, line, offset int) {
	// Strip <cfset and trailing >
	inner := tag
	if i := strings.IndexAny(inner, " \t\r\n"); i >= 0 {
		inner = inner[i+1:]
	}

	inner = strings.TrimSuffix(inner, ">")
	inner = strings.TrimSuffix(strings.TrimSpace(inner), "/")
	inner = strings.TrimSpace(inner)

	// A <cfset> holds an expression, and the shapes matched below are a few of
	// the ways one can hold a call. Measured against six open-source projects,
	// they missed about 3,100 call sites: a bare `<cfset arrayAppend(a, b)>`, a
	// call after a concatenation, a nested call in an argument list, and any
	// assignment whose left-hand side is scope-prefixed. The expression goes to
	// the script parser afterwards for the rest, which is what <cfif> and
	// <cfreturn> already do; the string paths stay because they carry the refs
	// and pending calls that decide what a variable now holds.
	before := len(p.calls)
	if p.inFunc != "" {
		before = len(p.funcCalls[p.inFunc])
	}

	start := strings.Index(tag, inner)
	line += strings.Count(tag[:start], "\n")
	origin := offset + start

	p.setAssign(inner, line)

	bucket := p.calls
	if p.inFunc != "" {
		bucket = p.funcCalls[p.inFunc]
	}

	for i := before; i < len(bucket); i++ {
		if !bucket[i].HasSpan {
			expressionCallOrigin(&bucket[i], inner, p.srcOffset+origin, line, true)
		}
	}

	p.mergeExpressionCalls(inner, line, true, true, origin, before)
}

// setAssign handles the assignment a <cfset> holds, whose calls
// parseCFSet's caller scans.
func (p *tagParser) setAssign(inner string, line int) {
	if _, rhs, ok := strings.Cut(inner, "="); ok && p.flow != nil && p.inFunc != "" && untypedValueText(rhs) {
		// Every assignment below records its VarDef first, except a local's
		// reassignment, which notes itself (flowBlocks.untypedReaches).
		n := len(p.vars)
		defer func() {
			if len(p.vars) > n {
				p.flow.noteAssign(p.vars[n].Name, p.vars[n].Line)
			}
		}()
	}

	switch {
	case hasPrefixFold(inner, "var "):
		rest := strings.TrimSpace(inner[4:])

		name := extractIdent(rest)
		if name != "" {
			p.vars = append(p.vars, VarDef{Name: name, Scope: ScopeLocal, Line: conv.Uint32(line)})

			if p.inFunc != "" {
				p.markVarLocal(name)
			}

			p.checkSetRHS(rest, name, line)
		}
	case hasPrefixFold(inner, "local."):
		rest := inner[6:]

		name, rhs := splitAssign(rest)
		if name != "" {
			p.vars = append(p.vars, VarDef{Name: name, Scope: ScopeLocal, Line: conv.Uint32(line)})

			if p.inFunc != "" {
				p.markVarLocal(name)
			}

			p.checkSetRHSLead(rest, rhs, name, line)
		}
	case hasPrefixFold(inner, "arguments."):
		rest := inner[10:]

		name, rhs := splitAssign(rest)
		if name != "" {
			p.vars = append(p.vars, VarDef{Name: name, Scope: ScopeArguments, Line: conv.Uint32(line)})
			p.checkSetRHSLead(rest, rhs, name, line)
		}
	case hasPrefixFold(inner, "this."):
		rest := inner[5:]

		name, rhs := splitAssign(rest)
		if name != "" {
			p.vars = append(p.vars, VarDef{Name: name, Scope: ScopeThis, Line: conv.Uint32(line)})
			p.forceGlobal, p.refThis = true, true
			p.checkSetRHSLead(rest, rhs, name, line)
			p.forceGlobal, p.refThis = false, false
		}
	case hasPrefixFold(inner, "variables."):
		rest := inner[10:]

		name, rhs := splitAssign(rest)
		if name != "" {
			p.vars = append(p.vars, VarDef{Name: name, Scope: ScopeVariables, Line: conv.Uint32(line)})
			p.forceGlobal = true
			p.checkSetRHSLead(rest, rhs, name, line)
			p.forceGlobal = false
		}
	case p.parseReadOnlyScopeSet(inner, line):
		// Handled: an assignment into url./application./request./session./etc.
		// The four shared scopes follow their right-hand side as variables. does;
		// the others hold request data and only declare the name.
	default:
		name, rhs := splitAssign(inner)
		if name != "" && (!isKeyword(name) || operatorWordIsNameStr(inner, name)) {
			if rhs == "" && p.extractCalls {
				// No assignment — check for bare call: obj.method(...)
				p.checkBareCallStr(inner, line)
			} else if rhs != "" {
				// If var was previously declared local in this function, keep it local
				isLocal := p.inFunc != "" && p.isVarDeclaredLocal(name)
				if !isLocal {
					p.vars = append(p.vars, VarDef{Name: name, Scope: ScopeVariables, Line: conv.Uint32(line)})
				} else if p.flow != nil && untypedValueText(rhs) {
					p.flow.noteAssign(name, conv.Uint32(line))
				}

				p.forceGlobal = !isLocal
				p.checkSetRHSLead(inner, rhs, name, line)
				p.forceGlobal = false
			}
		}
	}
}

// parseCFObject handles <cfobject component="path" name="var">.
func (p *tagParser) parseCFObject(tag string, line int) {
	component := getAttr(tag, "component")
	name := getAttr(tag, "name")

	if component != "" && name != "" {
		p.addRef(&ComponentRef{
			Variable:  name,
			Component: component,
			URI:       uriFromString(p.fileURI),
			Line:      conv.Uint32(line),
		})
	}
}

// parseCFInvoke handles <cfinvoke component="…" method="m" returnvariable="v">,
// which is the assignment `v = <component>.m()`: v holds what m returns, not
// an instance of the component. It used to be recorded as the component, so
// kernel2's getSandBox — `var result = getService("sandbox")`, then a
// cfinvoke of `get#name#` on it into result — was typed as the sandbox
// service, and 333 calls on what it really returns were reported against it.
//
// The assignment goes through setAssign, as a <cfset> would, so a component
// path, a variable holding one and a method's declared return are all read
// the way the script form reads them. A method whose name is computed at run
// time makes v dynamic.
func (p *tagParser) parseCFInvoke(tag string, line int) {
	variable := getAttr(tag, "returnvariable")
	if variable == "" {
		return
	}

	if rhs := invokeExpression(getAttr(tag, "component"), getAttr(tag, "method")); rhs != "" {
		p.setAssign(variable+" = "+rhs, line)

		return
	}

	p.addRef(&ComponentRef{
		Variable:  variable,
		Component: "$any",
		URI:       uriFromString(p.fileURI),
		Line:      conv.Uint32(line),
	})
}

// invokeExpression spells a <cfinvoke> as the call it makes, or "" when its
// target is not known from the source: no component, or a computed method.
// A component attribute is a dot-path, or `#expr#` naming an object.
func invokeExpression(component, method string) string {
	component, method = strings.TrimSpace(component), strings.TrimSpace(method)
	if component == "" || !isIdentText(method) {
		return ""
	}

	if expr, ok := wholeHashExpr(component); ok {
		if !isDottedIdentText(expr) {
			return ""
		}

		// The attribute is a component path or an object. A name ending in
		// Path holds the first (application.jsoupComponentPath), and the
		// component it names is whatever the path computes.
		if strings.HasSuffix(strings.ToLower(expr), "path") {
			return `createObject("component", ` + expr + `).` + method + "()"
		}

		return expr + "." + method + "()"
	}

	if strings.ContainsAny(component, "#\"'") {
		return ""
	}

	return `createObject("component", "` + component + `").` + method + "()"
}

// wholeHashExpr returns expr when s is exactly `#expr#`.
func wholeHashExpr(s string) (string, bool) {
	if len(s) < 3 || s[0] != '#' || s[len(s)-1] != '#' || strings.Contains(s[1:len(s)-1], "#") {
		return "", false
	}

	return strings.TrimSpace(s[1 : len(s)-1]), true
}

// isIdentText reports whether s is one identifier; a computed method name
// (`get#name#`) is not.
func isIdentText(s string) bool {
	if s == "" || !isIdentStart(s[0]) {
		return false
	}

	for i := 1; i < len(s); i++ {
		if !isIdentStart(s[i]) && (s[i] < '0' || s[i] > '9') {
			return false
		}
	}

	return true
}

func isDottedIdentText(s string) bool {
	for part := range strings.SplitSeq(s, ".") {
		if !isIdentText(part) {
			return false
		}
	}

	return true
}

// parseCFReturn handles <cfreturn expr /> to infer function return component.
func (p *tagParser) parseCFReturn(tag string, line, idx int) {
	inner := tagBody(tag)

	// A returned expression holds calls whether or not this is the return that
	// sets the component, and whether or not we are inside a function at all —
	// so the scan happens before every early exit below. `<cfreturn
	// svc.value()>` used to record nothing.
	p.scanTagExpression(inner, bodyLine(tag, inner, line), idx+strings.Index(tag, inner))

	if p.inFunc == "" {
		return
	}

	p.pendingCalls = append(p.pendingCalls, pendingCall{
		varName: strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(inner), "/")),
		funcKey: p.inFunc, returnExpr: true,
	})

	if len(p.funcs) == 0 {
		return
	}

	f := &p.funcs[len(p.funcs)-1]
	if f.returnVar != "" {
		return // a variable returned earlier is settled later, from its refs
	}

	f.returnLine = conv.Uint32(line)

	if p.flow != nil {
		p.flow.noteReturn(f.returnLine)
	}

	if inner == "" {
		return
	}

	returned := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(inner), "/"))

	// Check for direct new/createObject, read as a branch of a conditional
	// is (constructionHead: Lucee's java: and cfml: prefixes, a
	// non-component createObject through the resolvers). A component the
	// source names is the return whatever follows, as it always has been
	// (`new X().init()`); a Java object's chained calls return other things.
	switch {
	case hasPrefixFold(inner, "new "), hasPrefixFold(inner, "createobject("):
		comp, sc, named := constructionHead(returned, p.resolveCall, nil)
		if comp != "" && (named || sc.NextSkipComments().Kind == TokEOF) {
			f.ReturnComponent = unionReturn(f.ReturnComponent, comp)
		}
	case hasPrefixFold(returned, "entitynew("), hasPrefixFold(returned, "entityloadbypk("):
		// An entity, or none (entityLoadByPK finds nothing): as the same
		// call assigned to a variable is typed.
		open := strings.IndexByte(returned, '(') + 1
		if comp := entityRef(extractEntityNewArg(returned[open:])); comp != "" {
			f.ReturnComponent = unionReturn(f.ReturnComponent, comp)
		}
	case strings.EqualFold(returned, "this"):
		f.returnVar = returnsThis
	default:
		// `flag ? new A() : new B()` returns one or the other, as in script.
		if comp := conditionalConstructions(returned, p.resolveCall, nil); comp != "" {
			f.ReturnComponent = unionReturn(f.ReturnComponent, comp)

			return
		}

		// return varName — store for deferred resolution. A name read
		// through variables. or this. is the component's, and is read from
		// that scope's refs alone, as the script parser reads it.
		if name, scope := scopedReturnExpr(inner); name != "" {
			f.returnVar = scopedReturnVar(scope, name)

			return
		}

		// As in script syntax, only a name standing alone names the returned
		// value. `x.y` is a member of x, not x itself; claiming the function
		// returns x's component makes later chains resolve against the wrong
		// component. Member types are not represented by the parser, so leave
		// that expression untyped rather than inventing an answer.
		expr := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(inner), "/"))

		varName := extractIdent(expr)
		if varName != "" && varName == expr {
			f.returnVar = varName
		}
	}
}

// scopedReturnExpr is the name a `<cfreturn>` expression reads through
// variables. or this. when that is the whole expression, and the scope it
// names; "" for anything else.
func scopedReturnExpr(expr string) (string, RefScope) {
	expr = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(expr), "/"))

	scopeWord, name, ok := strings.Cut(expr, ".")
	if !ok || name == "" || extractIdent(name) != name {
		return "", RefAny
	}

	switch {
	case strings.EqualFold(scopeWord, "variables"):
		return name, RefVariables
	case strings.EqualFold(scopeWord, "this"):
		return name, RefThis
	default:
		return "", RefAny
	}
}

// parseCFProperty handles <cfproperty name="x" type="y" />.
func (p *tagParser) parseCFProperty(tag string, line int) {
	name := getAttr(tag, "name")
	if name == "" {
		return
	}

	typeName := getAttr(tag, "type")
	attrs := extractAllAttrs(tag)
	addBareFlags(tag, attrs)
	p.properties = append(p.properties, propertyDef{name: name, typeName: typeName, line: conv.Uint32(line), attrs: attrs})
}

// addBareFlags records the propertyFlags a tag writes without a value —
// `<cfproperty name="m" inject delegate>` — as attributes whose value is "",
// as the script parser does. extractAllAttrs skips a valueless attribute.
func addBareFlags(tag string, attrs map[string]string) {
	i := strings.IndexAny(tag, " \t\r\n")
	if i < 0 {
		return
	}

	for i < len(tag) {
		for i < len(tag) && isWhitespace(tag[i]) {
			i++
		}

		start := i
		for i < len(tag) && !isWhitespace(tag[i]) && tag[i] != '=' && tag[i] != '>' && tag[i] != '/' {
			i++
		}

		key := strings.ToLower(tag[start:i])

		j := i
		for j < len(tag) && isWhitespace(tag[j]) {
			j++
		}

		if j < len(tag) && tag[j] == '=' {
			i = skipAttrValue(tag, j+1)

			continue
		}

		if propertyFlags[key] {
			if _, set := attrs[key]; !set {
				attrs[key] = ""
			}
		}

		if i == start {
			i++
		}
	}
}

// skipAttrValue returns the index past the attribute value starting at i.
func skipAttrValue(tag string, i int) int {
	for i < len(tag) && isWhitespace(tag[i]) {
		i++
	}

	if i < len(tag) && (tag[i] == '"' || tag[i] == '\'') {
		if end := strings.IndexByte(tag[i+1:], tag[i]); end >= 0 {
			return i + end + 2
		}

		return len(tag)
	}

	for i < len(tag) && !isWhitespace(tag[i]) && tag[i] != '>' {
		i++
	}

	return i
}

func (p *tagParser) checkSetRHS(rest, varName string, line int) {
	// Find = and check what's after it
	if _, after, ok := strings.Cut(rest, "="); ok {
		p.checkSetRHSLead(rest, strings.TrimSpace(after), varName, line)
	}
}

// checkSetRHSLead is checkSetRHSStr for a value split off rest, the text of the
// tag after the part that named the variable, with the value's own line.
func (p *tagParser) checkSetRHSLead(rest, rhs, varName string, line int) {
	saved := p.rhsLead
	p.rhsLead = leadLines(rest, rhs)

	p.checkSetRHSStr(rhs, varName, line)

	p.rhsLead = saved
}

// leadLines is the number of line breaks in rest before the value rhs that
// follows its first "=".
func leadLines(rest, rhs string) int {
	eq := strings.IndexByte(rest, '=')
	if eq < 0 || rhs == "" {
		return 0
	}

	i := strings.Index(rest[eq+1:], rhs)
	if i < 0 {
		return 0
	}

	return strings.Count(rest[:eq+1+i], "\n")
}

func (p *tagParser) checkSetRHSStr(rhs, varName string, line int) {
	rhs = strings.TrimSpace(rhs)

	if p.chainDepth < maxChainedAssignments && chainedSetTarget(rhs) {
		p.chainedSet(rhs, varName, line)

		return
	}

	if comp := wholeArgumentComponent(rhs, p.inFunc, p.funcs); comp != "" {
		p.addRef(&ComponentRef{
			Variable: varName, Component: comp,
			URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
		})

		return
	}

	switch {
	case strings.EqualFold(rhs, "this"):
		if selfPath := strings.TrimPrefix(p.fileURI, "file://"); selfPath != "" {
			p.addRef(&ComponentRef{
				Variable: varName, Component: selfPath,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})
		}
	case hasPrefixFold(rhs, "new "):
		comp := extractComponentPath(rhs[4:])
		if comp != "" {
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp, ChainRest: trailingCalls(rhs),
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})
		}

	case hasPrefixFold(rhs, "createobject("):
		comp := extractCreateObjectArg(rhs[13:])
		if comp == "" {
			comp, _ = nativeJavaHead(rhs)
		}

		if comp != "" {
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp, ChainRest: trailingCalls(rhs),
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})
		} else if object := withoutInit(rhs); p.resolveCall(object) != "" {
			// Not only when resolvers are configured: with none, resolveCall
			// still answers through dynamicCall, which makes an unstubbed Java
			// object dynamic, as the script parser always has. The init() on
			// the end hands back the object it was called on, so the object is
			// what is asked about: dynamicCall looks at a chain's last call.
			p.addRef(&ComponentRef{
				Variable: varName, Component: p.resolveCall(object), ChainRest: trailingCalls(object),
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})
		} else if _, java := nativeJavaHead(rhs); java {
			// A Java class nothing describes is a dynamic value, as the
			// script parser reads it.
			p.addRef(&ComponentRef{
				Variable: varName, Component: "$any",
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})
		}

	case hasPrefixFold(rhs, "entitynew("):
		comp := entityRef(extractEntityNewArg(rhs[10:]))
		if comp != "" {
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})
		}
	case hasPrefixFold(rhs, "entityload("):
		comp := entityRef(extractEntityNewArg(rhs[11:]))
		if comp != "" {
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})
		}
	case hasPrefixFold(rhs, "entityloadbypk("):
		comp := entityRef(extractEntityNewArg(rhs[15:]))
		if comp != "" {
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})
		}
	default:
		if p.resolveRHS(rhs, varName, line) {
			return
		}

		if plainName(rhs) && !strings.EqualFold(rhs, varName) {
			p.pendingCalls = append(p.pendingCalls, pendingCall{
				varName: varName, baseVar: rhs, alias: true,
				line: conv.Uint32(line), funcKey: p.inFunc,
				refThis: p.refThis, global: p.forceGlobal,
				baseScope: ReceiverRefScope(rhs),
				baseLocal: readThroughLocal(rhs) || p.inFunc != "" && (p.isVarDeclaredLocal(rootName(rhs)) || p.isArgument(rootName(rhs))),
			})

			return
		}

		// Detect x = someVar.method(...) or x = funcName(...) pattern
		if baseVar := extractMethodCallBase(rhs); baseVar != "" {
			p.methodCallRHS(rhs, baseVar, varName, line)
		} else if paren := strings.IndexByte(rhs, '('); paren > 0 {
			p.funcCallRHS(rhs, paren, varName, line)
		}
	}
}

// currentCaller is the function the walk is inside, or "".
// callsOnResolvedReceiver lists the calls rhs makes on a receiver that comp
// already describes, outermost last. A regex resolver is unanchored at its
// end, so tassweb's `subsObj\.([a-zA-Z]+)` claims
// `variables.subsObj.subsLib.startCFC(x)` on its first two segments: the
// value is what startCFC returns, and recording comp alone typed it as
// subsLib. Each call peeled off while the receiver keeps comp is a hop to
// walk. `init()` returns its receiver, so it is not one.
func (p *tagParser) callsOnResolvedReceiver(rhs, comp string) []string {
	var methods []string

	for expr := rhs; ; {
		receiver, method := callReceiver(expr)
		if receiver == "" || p.resolveCall(receiver) != comp {
			break
		}

		if !strings.EqualFold(method, "init") {
			methods = append([]string{method}, methods...)
		}

		expr = receiver
	}

	return methods
}

// withoutInit is expr without a final .init( … ) call made directly on a
// createObject( … ), which returns the object it is called on; any other
// expression is returned as it is.
func withoutInit(expr string) string {
	if receiver, method := callReceiver(expr); strings.EqualFold(method, "init") && hasPrefixFold(strings.TrimSpace(receiver), "createobject(") {
		return strings.TrimSpace(receiver)
	}

	return expr
}

// callReceiver splits `<receiver>.m(args)` into receiver and m, or returns
// "" when expr is not a method call ending at its argument list.
func callReceiver(expr string) (receiver, method string) {
	expr = strings.TrimSpace(expr)
	if !strings.HasSuffix(expr, ")") {
		return "", ""
	}

	depth, open := 0, -1

	var quote byte

	for i := range len(expr) {
		c := expr[i]

		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '(':
			if depth == 0 {
				open = i
			}

			depth++
		case c == ')':
			depth--
			if depth < 0 {
				return "", ""
			}
		}
	}

	if depth != 0 || quote != 0 || open <= 0 {
		return "", ""
	}

	j := open
	for j > 0 && lex.IsIdent(expr[j-1]) {
		j--
	}

	if j == open || j == 0 || expr[j-1] != '.' {
		return "", ""
	}

	return strings.TrimSpace(expr[:j-1]), expr[j:open]
}

func (p *tagParser) currentCaller() string {
	if p.inFunc != "" && len(p.funcs) > 0 {
		return p.funcs[len(p.funcs)-1].Name
	}

	return ""
}

// chainedSet handles `<cfset a = b = rhs>`: the inner assignment is made as
// the statement it is, and what it made is made again for varName under the
// state the outer assignment set up. See chained.go.
func (p *tagParser) chainedSet(inner, varName string, line int) {
	forceGlobal, refThis := p.forceGlobal, p.refThis
	nGlobal, nLocal, nPending := len(p.componentRefs), len(p.funcRefs[p.inFunc]), len(p.pendingCalls)

	p.chainDepth++
	p.setAssign(inner, line)
	p.chainDepth--

	made := refsMadeFor(p.componentRefs, nGlobal, p.funcRefs[p.inFunc], nLocal, varName)
	p.forceGlobal, p.refThis = forceGlobal, refThis

	for i := range made {
		p.addRef(&made[i])
	}

	p.pendingCalls = appendPendingFor(p.pendingCalls, nPending, varName, refThis)
}

// resolveRHS types varName through a componentResolver matching the whole
// right-hand side, or its bare function name for exact-match resolvers, and
// reports whether one did.
func (p *tagParser) resolveRHS(rhs, varName string, line int) bool {
	if len(p.resolvers) == 0 {
		return false
	}

	if comp := p.resolveCall(rhs); comp != "" {
		methods := p.callsOnResolvedReceiver(rhs, comp)

		p.addRef(&ComponentRef{
			Variable: varName, Component: comp, ChainRest: methods, strictChain: len(methods) > 0,
			URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
		})

		return true
	}

	// A configured factory describes the first call, not the value produced
	// by later calls. Keep the suffix for the normal return-type settlement.
	if root, methods := factoryReturnChain(rhs); root != "" && len(methods) > 0 {
		var comp string
		if p.resolverSet != nil {
			comp = p.resolverSet.Resolve(root)
		} else {
			comp = ResolveFromCall(root, p.resolvers)
		}

		if comp != "" {
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp, ChainRest: methods, strictChain: true,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})

			return true
		}
	}

	// Try bare function name for exact-match resolvers
	funcName := extractIdent(rhs)
	if funcName == "" {
		return false
	}

	// Anything after the name — a call on it, an index into it — makes the
	// value something the variable produced, which a NameOnly resolver has
	// nothing to say about; `x = event` alone is still event.
	derived := strings.TrimSpace(rhs[len(funcName):]) != ""

	for i := range p.resolvers {
		r := &p.resolvers[i]
		if r.Prefix == "" || (r.NameOnly && derived) || !prefixEqualFold(funcName, r.Prefix) {
			continue
		}

		if comp := matchResolverWithCache(funcName, r); comp != "" {
			p.resolverSet.noteSoft(r, comp)

			p.addRef(&ComponentRef{
				Variable: varName, Component: comp,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})

			return true
		}
	}

	return false
}

// methodCallRHS handles `x = baseVar.method(...)`: the call, and a pending
// call that types x once method's return type is known.
func (p *tagParser) methodCallRHS(rhs, baseVar, varName string, line int) {
	// Extract method name for pendingCall
	var methodForPending string

	before, _, _ := strings.Cut(rhs, "(")

	varChain, methodName, hasDot := strings.CutLast(before, ".")
	if hasDot {
		methodForPending = methodName
	}

	// Record full call: extract method name from rhs
	if p.extractCalls && hasDot && isIdentifier(methodName) && isValidVarChain(varChain) {
		comp := p.lookupComponentRef(varChain, line)

		p.addCall(&CallSite{
			FuncName:  methodName,
			Variable:  varChain,
			Component: comp,
			Resolved:  comp != "",
			Line:      conv.Uint32(line + p.rhsLead),
			Caller:    p.currentCaller(),
		})
	}

	p.pendingCalls = append(p.pendingCalls, pendingCall{
		varName:    varName,
		funcName:   methodForPending,
		expression: callExpressionAt(NewScanner(rhs[strings.IndexByte(rhs, '('):]), methodForPending),
		baseVar:    baseVar,
		line:       conv.Uint32(line),
		funcKey:    p.inFunc,
		rest:       trailingCalls(rhs),
		refThis:    p.refThis,
		global:     p.forceGlobal,
		// varChain is the receiver: `variables.a.m()` reads a from
		// variables scope only.
		baseScope: ReceiverRefScope(varChain),
		baseLocal: readThroughLocal(varChain) || p.inFunc != "" && (p.isVarDeclaredLocal(varChain) || p.isArgument(varChain)),
		rebinds:   rebinds(varName, varChain),
	})
}

// funcCallRHS handles `x = funcName(...)`, whose paren is at paren: a builtin's
// return type, else the call and a pending call that types x later.
func (p *tagParser) funcCallRHS(rhs string, paren int, varName string, line int) {
	funcName := extractIdent(rhs)
	// Confirm the parens actually belong to funcName (only whitespace between
	// them) — otherwise rhs is a larger expression like `temp & "~" & DateFormat(...)`
	// where extractIdent grabbed the leading token ("temp"), not the identifier the
	// paren is attached to, and this isn't a `x = funcName(...)` shape at all.
	if funcName == "" || isKeyword(funcName) || strings.TrimSpace(rhs[len(funcName):paren]) != "" {
		return
	}

	if p.builtinReturnLookup != nil {
		if comp := p.builtinReturnLookup(funcName); comp != "" && !strings.HasPrefix(comp, NativeTypePrefix) {
			p.addRef(&ComponentRef{
				Variable: varName, Component: comp,
				URI: uriFromString(p.fileURI), Line: conv.Uint32(line),
			})

			return
		}
	}

	if p.extractCalls {
		p.addCall(&CallSite{
			FuncName: funcName,
			Line:     conv.Uint32(line + p.rhsLead),
			Caller:   p.currentCaller(),
		})
	}

	p.pendingCalls = append(p.pendingCalls, pendingCall{
		varName:    varName,
		funcName:   funcName,
		expression: callExpressionAt(NewScanner(rhs[paren:]), funcName),
		line:       conv.Uint32(line),
		funcKey:    p.inFunc,
		rest:       trailingCalls(rhs),
		refThis:    p.refThis,
		global:     p.forceGlobal,
	})
}

// checkBareCallStr detects bare obj.method(...) patterns in a <cfset> without assignment.
func (p *tagParser) checkBareCallStr(expr string, line int) {
	before, _, ok := strings.Cut(expr, "(")
	if !ok {
		return
	}

	// If there's an = before the (, it's an assignment, not a bare call
	if strings.IndexByte(before, '=') >= 0 {
		return
	}

	recv, method, ok := strings.CutLast(before, ".")
	if !ok {
		return
	}

	varName := strings.TrimSpace(recv)
	methodName := strings.TrimSpace(method)

	if methodName == "" || varName == "" || !isIdentifier(methodName) {
		return
	}

	if !isValidVarChain(varName) {
		return
	}

	caller := ""
	if p.inFunc != "" && len(p.funcs) > 0 {
		caller = p.funcs[len(p.funcs)-1].Name
	}

	comp := p.lookupComponentRef(varName, line)

	p.addCall(&CallSite{
		FuncName:  methodName,
		Variable:  varName,
		Component: comp,
		Resolved:  comp != "",
		Line:      conv.Uint32(line),
		Caller:    caller,
	})
}
