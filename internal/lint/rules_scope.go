package lint

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerScopeRules(DefaultRegistry)
}

func registerScopeRules(reg *Registry) {
	reg.Register(Rule{
		Code:            "MISSING_VAR",
		DefaultSeverity: "ERROR",
		Check:           checkMissingVar,
	})
	reg.Register(Rule{
		Code:            "ARG_VAR_CONFLICT",
		DefaultSeverity: "ERROR",
		Check:           checkArgVarConflict,
	})
	reg.Register(Rule{
		Code:            "ARG_VAR_MIXED",
		DefaultSeverity: "INFO",
		Check:           checkArgVarMixed,
	})
	reg.Register(Rule{
		Code:            "IMPLICIT_SCOPE",
		DefaultSeverity: "INFO",
		Check:           checkImplicitScope,
	})
}

var cfmlScopes = map[string]bool{
	"local": true, "arguments": true, "variables": true, "this": true, "super": true,
	"application": true, "session": true, "request": true, "server": true,
	"form": true, "url": true, "cookie": true, "cgi": true, "client": true,
	"cfcatch": true, "caller": true, "attributes": true, "thistag": true,
	"thread": true, "cfthread": true, "file": true,
	"cffile": true, "cfhttp": true, "cfquery": true, "cfstoredproc": true,
}

var cfmlKeywords = map[string]bool{
	"true": true, "false": true, "null": true, "yes": true, "no": true,
	"return": true, "if": true, "else": true, "elseif": true, "for": true, "in": true,
	"while": true, "do": true, "switch": true, "case": true, "default": true,
	"break": true, "continue": true, "try": true, "catch": true, "finally": true,
	"throw": true, "rethrow": true, "new": true, "var": true, "function": true,
	"component": true, "interface": true, "property": true, "import": true, "include": true,
	"param": true, "lock": true, "transaction": true, "savecontent": true, "abort": true,
	"exit": true, "and": true, "or": true, "not": true, "xor": true, "imp": true, "eqv": true,
	"eq": true, "neq": true, "lt": true, "lte": true, "le": true, "gt": true, "gte": true, "ge": true,
	"is": true, "contains": true, "mod": true, "required": true, "public": true, "private": true,
	"package": true, "remote": true, "static": true, "final": true, "abstract": true,
	"any": true, "string": true, "numeric": true, "boolean": true, "struct": true,
	"array": true, "query": true, "void": true, "date": true, "binary": true, "xml": true,
}

type funcScopeAnalysis struct {
	funcName        string
	localModeModern bool
	params          map[string]bool
	vars            map[string]int // lowercased name -> first var decl offset
	localAssign     map[string]bool
	argAssign       map[string]bool
	catchVars       map[string]bool
	bareWrites      []identUse
	bareReads       []identUse // all unscoped reads (used by ARG_VAR_MIXED)
	implicitReads   []identUse // single-identifier reads without `.member` (used by IMPLICIT_SCOPE)
	localOrVarReads []identUse // non-assignment `local.<name>` / `variables.<name>` reads for ARG_VAR_MIXED
	argScoped       map[string]bool
}

var queryLoopProperties = map[string]bool{
	"currentrow":  true,
	"recordcount": true,
	"columnlist":  true,
}

type identUse struct {
	name   string
	offset int
}

var scriptTagOutputAttrs = map[string]map[string]bool{
	"cfloop":        {"index": true, "item": true},
	"cfquery":       {"name": true, "result": true},
	"cfhttp":        {"result": true, "name": true},
	"cffile":        {"variable": true, "result": true},
	"cfsavecontent": {"variable": true},
	"cfdirectory":   {"name": true},
	"cfprocresult":  {"name": true},
	"cfstoredproc":  {"result": true},
	"cfinvoke":      {"returnvariable": true},
	"cfwddx":        {"output": true},
	"cfxml":         {"variable": true},
	"cfpdf":         {"name": true},
	"cfimage":       {"name": true, "structname": true},
}

func hasModernLocalMode(header string) bool {
	low := strings.ToLower(header)

	for pos := 0; pos < len(low); {
		rel := strings.Index(low[pos:], "localmode")
		if rel < 0 {
			return false
		}

		idx := pos + rel
		pos = idx + len("localmode")

		if idx > 0 && isIdentChar(low[idx-1]) {
			continue
		}

		rest := strings.TrimSpace(low[pos:])
		if !strings.HasPrefix(rest, "=") {
			continue
		}

		val := strings.TrimSpace(rest[1:])
		val = strings.TrimLeft(val, `"'#`)

		if strings.HasPrefix(val, "modern") || strings.HasPrefix(val, "true") {
			return true
		}
	}

	return false
}

func isIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_'
}

func analyzeFunctionScope(content string, span *parser.FunctionSpan) *funcScopeAnalysis {
	a := &funcScopeAnalysis{
		funcName:        span.Name,
		localModeModern: span.Start >= 0 && span.Body > span.Start && hasModernLocalMode(content[span.Start:span.Body]),
		params:          make(map[string]bool),
		vars:            make(map[string]int),
		localAssign:     make(map[string]bool),
		argAssign:       make(map[string]bool),
		catchVars:       make(map[string]bool),
		argScoped:       make(map[string]bool),
	}

	body := span.BodyText(content)
	if body == "" {
		return a
	}

	// 1. Collect parameter names (script params or <cfargument>).
	if pText := span.ParamsText(content); pText != "" {
		collectScriptParamNames(pText, a.params)
	}

	for _, tag := range parser.CFTags(body, "cfargument", "cfcatch") {
		if tag.Close {
			continue
		}

		name, ok := tag.Attr("name")
		if !ok || name == "" {
			continue
		}

		low := strings.ToLower(strings.TrimSpace(name))
		if strings.EqualFold(tag.Name, "cfcatch") {
			a.catchVars[low] = true
		} else {
			a.params[low] = true
		}
	}

	// 2. Collect tag output attributes inside the function body.
	for _, tag := range parser.CFTags(body, "cfloop", "cfquery", "cfhttp", "cffile", "cfsavecontent", "cfdirectory", "cfprocresult", "cfstoredproc", "cfinvoke", "cfwddx", "cfxml") {
		if tag.Close {
			continue
		}

		attrs := scriptTagOutputAttrs[strings.ToLower(tag.Name)]
		for attr := range attrs {
			val, ok := tag.Attr(attr)
			if !ok {
				continue
			}

			recordOutputVarName(val, span.Body+tag.Start, a)
		}
	}

	// 3. Scan script regions inside the function body (or <cfset> tags in tag functions).
	if span.Params >= 0 {
		scanScriptBodyScope(body, span.Body, a)
	} else {
		for _, r := range parser.ClassifyRegions(body) {
			if r.Kind == parser.RegionScript {
				scanScriptBodyScope(r.Text, span.Body+r.Offset, a)
			}
		}

		// Include cfif/cfelseif/cfreturn so CFTags advances past their quoted
		// string literals (e.g. regexes containing `"<cfset..."`).
		for _, tag := range parser.CFTags(body, "cfset", "cfif", "cfelseif", "cfreturn") {
			if tag.Close || tag.Name != "cfset" {
				continue
			}

			txt := tag.Text()
			if len(txt) <= len(tag.Name)+2 {
				continue
			}

			innerStart := len(tag.Name) + 1
			inner := txt[innerStart : len(txt)-1]
			scanScriptBodyScope(inner, span.Body+tag.Start+innerStart, a)
		}
	}

	slices.SortStableFunc(a.bareWrites, func(x, y identUse) int { return cmp.Compare(x.offset, y.offset) })
	slices.SortStableFunc(a.bareReads, func(x, y identUse) int { return cmp.Compare(x.offset, y.offset) })
	slices.SortStableFunc(a.implicitReads, func(x, y identUse) int { return cmp.Compare(x.offset, y.offset) })
	slices.SortStableFunc(a.localOrVarReads, func(x, y identUse) int { return cmp.Compare(x.offset, y.offset) })

	return a
}

func recordOutputVarName(raw string, offset int, a *funcScopeAnalysis) {
	val := strings.TrimSpace(raw)
	if val == "" || strings.Contains(val, "#") {
		return
	}

	low := strings.ToLower(val)
	if prefix, after, found := strings.Cut(low, "."); found {
		if prefix == "local" && after != "" {
			a.localAssign[after] = true
		}

		if !cfmlScopes[prefix] && !cfmlKeywords[prefix] {
			a.bareWrites = append(a.bareWrites, identUse{name: val[:len(prefix)], offset: offset})
		}

		return
	}

	if cfmlScopes[low] || cfmlKeywords[low] {
		return
	}

	a.bareWrites = append(a.bareWrites, identUse{name: val, offset: offset})
}

func collectScriptParamNames(paramsText string, dst map[string]bool) {
	sc := parser.NewSourceScanner(paramsText)

	var toks []parser.Token

	depth := 0
	flush := func() {
		var head []parser.Token

		hasEq := false

		for _, t := range toks {
			if t.Kind == parser.TokEquals {
				hasEq = true

				break
			}

			head = append(head, t)
		}

		if name := extractScriptParamName(head, hasEq); name != "" {
			dst[strings.ToLower(name)] = true
		}

		toks = toks[:0]
	}

	for cur := sc.NextSkipComments(); cur.Kind != parser.TokEOF; cur = sc.NextSkipComments() {
		switch cur.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++

			toks = append(toks, cur)
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--

			toks = append(toks, cur)
		case parser.TokComma:
			if depth == 0 {
				flush()
			} else {
				toks = append(toks, cur)
			}
		default:
			toks = append(toks, cur)
		}
	}

	flush()
}

// extractScriptParamName extracts the parameter name from the tokens before `=`
// in a script parameter declaration, handling optional `required`, dotted types
// (`models.User u`), and trailing metadata attributes (`string beanPath hint="..."`).
func extractScriptParamName(head []parser.Token, hasEq bool) string {
	if len(head) > 0 && head[0].Kind == parser.TokIdent && strings.EqualFold(head[0].Value, "required") {
		head = head[1:]
	}

	if len(head) == 0 {
		return ""
	}

	// Skip an optional dotted type `ident(.ident)*` when followed by at least
	// one more identifier (the parameter name).
	idx := 0

	if head[0].Kind == parser.TokIdent {
		typeEnd := 1
		for typeEnd+1 < len(head) && head[typeEnd].Kind == parser.TokDot && head[typeEnd+1].Kind == parser.TokIdent {
			typeEnd += 2
		}

		if typeEnd < len(head) && head[typeEnd].Kind == parser.TokIdent {
			// Distinguish `[type, paramName, ...]` from untyped `[paramName, hint]`
			// followed by `="..."`.
			isUntypedHint := typeEnd == 1 && len(head) == 2 && hasEq && strings.EqualFold(head[1].Value, "hint")
			if !isUntypedHint {
				idx = typeEnd
			}
		}
	}

	if idx < len(head) && head[idx].Kind == parser.TokIdent {
		return head[idx].Value
	}

	return ""
}

type parenFrame struct {
	isFor           bool
	isClosureParams bool
	callName        string // lowercased callee name when this `(` is a call arg list
}

type braceKind uint8

const (
	braceBlock braceKind = iota
	braceStruct
	braceClosure
)

var controlParenKeywords = map[string]bool{
	"if": true, "elseif": true, "while": true, "switch": true, "catch": true, "for": true, "function": true,
}

var scriptTagStmtKeywords = map[string]bool{
	"application": true, "lock": true, "transaction": true, "param": true, "savecontent": true,
	"setting": true, "mail": true, "http": true, "query": true, "storedproc": true,
	"wddx": true, "file": true, "directory": true, "thread": true, "cookie": true,
	"header": true, "content": true, "location": true, "log": true, "trace": true,
	"cache": true, "document": true, "pdf": true, "zip": true, "ftp": true,
	"schedule": true, "htmlhead": true, "htmlbody": true, "processingdirective": true,
	"flush": true, "exit": true, "abort": true, "throw": true, "invoke": true,
	"module": true, "execute": true, "dbinfo": true, "ldap": true, "imap": true,
	"pop": true, "feed": true, "image": true, "spreadsheet": true, "xml": true,
	"objectcache": true, "silent": true, "timer": true, "login": true,
	"loginuser": true, "logout": true, "registry": true, "index": true,
	"collection": true, "search": true, "exchange": true, "chart": true,
	"report": true, "print": true, "presentation": true, "sharepoint": true,
}

func scanScriptBodyScope(text string, baseOffset int, a *funcScopeAnalysis) {
	scanScriptBodyScopeDepth(text, baseOffset, a, 0)
}

func scanScriptBodyScopeDepth(text string, baseOffset int, a *funcScopeAnalysis, initClosureDepth int) {
	toks := scanCFMLScriptTokens(text)

	var (
		braceKinds          []braceKind
		parens              []parenFrame
		tagStmtName         string // non-empty inside `lock type="exclusive" name="..." {`
		closureDepth        = initClosureDepth
		pendingClosureBrace bool
		arrowExprParenDepth []int
	)

	popArrowExprs := func(maxParenDepth int) {
		for len(arrowExprParenDepth) > 0 && arrowExprParenDepth[len(arrowExprParenDepth)-1] >= maxParenDepth {
			arrowExprParenDepth = arrowExprParenDepth[:len(arrowExprParenDepth)-1]
			closureDepth--
		}
	}

	for i, cur := range toks {
		var prev2, prev, next parser.Token
		if i >= 2 {
			prev2 = toks[i-2]
		}

		if i >= 1 {
			prev = toks[i-1]
		}

		if i+1 < len(toks) {
			next = toks[i+1]
		}

		switch cur.Kind {
		case parser.TokSemicolon:
			if len(parens) == 0 {
				tagStmtName = ""
				pendingClosureBrace = false

				popArrowExprs(0)
			}
		case parser.TokComma:
			popArrowExprs(len(parens))
		case parser.TokEquals:
			// Arrow operator `=>` starts either a block closure (`=> { ... }`) or
			// an expression-bodied closure (`=> expr`).
			if next.Kind == parser.TokGT && next.Offset == cur.Offset+1 {
				if tokAt(toks, i+2).Kind == parser.TokLBrace {
					pendingClosureBrace = true
				} else {
					arrowExprParenDepth = append(arrowExprParenDepth, len(parens))
					closureDepth++
				}
			}
		case parser.TokLBrace:
			tagStmtName = ""

			kind := classifyScriptLBrace(prev, pendingClosureBrace)
			if kind == braceClosure {
				pendingClosureBrace = false
				closureDepth++
			}

			braceKinds = append(braceKinds, kind)
		case parser.TokRBrace:
			if len(braceKinds) > 0 {
				if braceKinds[len(braceKinds)-1] == braceClosure && closureDepth > initClosureDepth {
					closureDepth--
				}

				braceKinds = braceKinds[:len(braceKinds)-1]
			}
		case parser.TokLParen:
			parens = append(parens, classifyScriptLParen(toks, i, prev2, prev, a))
		case parser.TokRParen:
			if len(parens) > 0 {
				top := parens[len(parens)-1]
				parens = parens[:len(parens)-1]

				if top.isClosureParams {
					pendingClosureBrace = true
				}

				popArrowExprs(len(parens) + 1)
			}
		case parser.TokString:
			// Scan `#expr#` interpolations inside script string literals
			// (e.g. `"hello #event.getCurrentEvent()#"`).
			forEachStringHashExpr(cur, func(expr string, relOff int) {
				scanScriptBodyScopeDepth(expr, baseOffset+cur.Offset+relOff, a, closureDepth)
			})
		case parser.TokIdent:
			if handleScriptIdentToken(toks, i, baseOffset, prev2, prev, cur, next, braceKinds, parens, &tagStmtName, closureDepth > 0, a) {
				continue
			}
		default:
		}
	}
}

func handleScriptIdentToken(toks []parser.Token, idx, baseOffset int, prev2, prev, cur, next parser.Token, braceKinds []braceKind, parens []parenFrame, tagStmtName *string, inClosure bool, a *funcScopeAnalysis) bool {
	// Single-parameter arrow closure `item => ...`.
	if next.Kind == parser.TokEquals && tokAt(toks, idx+2).Kind == parser.TokGT && tokAt(toks, idx+2).Offset == next.Offset+1 {
		a.catchVars[strings.ToLower(cur.Value)] = true

		return true
	}

	// Detect script-tag statement header: `lock type="..." name="..." {` or
	// `application action="update" ...;`.
	if len(parens) == 0 && *tagStmtName == "" && prev.Kind != parser.TokDot &&
		next.Kind == parser.TokIdent && tokAt(toks, idx+2).Kind == parser.TokEquals {
		if low := strings.ToLower(cur.Value); scriptTagStmtKeywords[low] {
			*tagStmtName = "cf" + low

			return true
		}
	}

	inStructLit := len(braceKinds) > 0 && braceKinds[len(braceKinds)-1] == braceStruct

	var topParen parenFrame
	if len(parens) > 0 {
		topParen = parens[len(parens)-1]
	}

	recordScriptIdent(toks, idx, baseOffset, prev2, prev, cur, next, inStructLit, inClosure, topParen, *tagStmtName, a)

	return false
}

func classifyScriptLBrace(prev parser.Token, pendingClosureBrace bool) braceKind {
	if pendingClosureBrace {
		return braceClosure
	}

	if prev.Kind == parser.TokEquals || prev.Kind == parser.TokLParen ||
		prev.Kind == parser.TokComma || prev.Kind == parser.TokLBracket ||
		prev.Kind == parser.TokColon || prev.Kind == parser.TokQuestion ||
		(prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "return")) {
		return braceStruct
	}

	return braceBlock
}

func classifyScriptLParen(toks []parser.Token, lpIdx int, prev2, prev parser.Token, a *funcScopeAnalysis) parenFrame {
	f := parenFrame{}

	switch {
	case isArrowParamList(toks, lpIdx):
		collectClosureParamNames(toks, lpIdx, a.catchVars)
	case prev.Kind == parser.TokRBracket || prev.Kind == parser.TokRParen || prev.Kind == parser.TokString:
		f.callName = "<call>"
	case prev.Kind == parser.TokIdent:
		lowPrev := strings.ToLower(prev.Value)
		switch {
		case lowPrev == "for":
			f.isFor = true
		case lowPrev == "catch":
			collectCatchVarName(toks, lpIdx, a.catchVars)
		case lowPrev == "function":
			collectClosureParamNames(toks, lpIdx, a.catchVars)

			f.isClosureParams = true
		case prev2.Kind == parser.TokIdent && strings.EqualFold(prev2.Value, "function"):
			// Named function declaration inside a .cfm <cfscript> block:
			// `function helper(any a, any b) { ... }`.
			collectClosureParamNames(toks, lpIdx, a.catchVars)
		case !controlParenKeywords[lowPrev]:
			f.callName = lowPrev
		}
	}

	return f
}

func recordScriptIdent(toks []parser.Token, idx, baseOffset int, prev2, prev, cur, next parser.Token, inStructLit, inClosure bool, topParen parenFrame, tagStmtName string, a *funcScopeAnalysis) {
	low := strings.ToLower(cur.Value)

	if recordScopedMemberIdent(toks, idx, baseOffset, prev2, prev, cur, a) {
		return
	}

	// Skip member accesses (`.foo`), numeric struct key continuations (`.1.title`),
	// function calls (`foo(`), struct keys (`foo:`), `new pkg.Component(...)`,
	// and keywords/scopes.
	if prev.Kind == parser.TokDot || next.Kind == parser.TokLParen || next.Kind == parser.TokColon {
		return
	}

	if prev.Kind == parser.TokNumber && strings.HasSuffix(prev.Value, ".") {
		return
	}

	if prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "new") {
		return
	}

	if cfmlKeywords[low] || cfmlScopes[low] {
		return
	}

	// `catch (any e)` or `catch (e)`: record catch variable.
	if prev.Kind == parser.TokLParen && prev2.Kind == parser.TokIdent && strings.EqualFold(prev2.Value, "catch") && next.Kind == parser.TokRParen {
		a.catchVars[low] = true

		return
	}

	if prev.Kind == parser.TokIdent && next.Kind == parser.TokRParen && prev2.Kind == parser.TokLParen {
		// e.g. `catch (any e)` -> `prev` is `any`, `cur` is `e`.
		a.catchVars[low] = true

		return
	}

	// `var x` declaration.
	if prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "var") {
		if _, exists := a.vars[low]; !exists {
			a.vars[low] = baseOffset + cur.Offset
		}

		return
	}

	// Script-tag statement attribute (`lock type="exclusive" name="..." {`).
	if tagStmtName != "" && next.Kind == parser.TokEquals && isStandaloneEquals(cur, next, tokAt(toks, idx+2)) {
		if attrs := scriptTagOutputAttrs[tagStmtName]; attrs != nil && attrs[low] {
			if valTok := tokAt(toks, idx+2); valTok.Kind == parser.TokString && len(valTok.Value) >= 2 {
				recordOutputVarName(valTok.Value[1:len(valTok.Value)-1], baseOffset+cur.Offset, a)
			}
		}

		return
	}

	// Named argument inside a function or script-tag call `fn(arg=val)`.
	if topParen.callName != "" && next.Kind == parser.TokEquals && isStandaloneEquals(cur, next, tokAt(toks, idx+2)) {
		if attrs := scriptTagOutputAttrs[topParen.callName]; attrs != nil && attrs[low] {
			if valTok := tokAt(toks, idx+2); valTok.Kind == parser.TokString && len(valTok.Value) >= 2 {
				recordOutputVarName(valTok.Value[1:len(valTok.Value)-1], baseOffset+cur.Offset, a)
			}
		}

		return
	}

	// `for (item in expr)` without `var`.
	if topParen.isFor && next.Kind == parser.TokIdent && strings.EqualFold(next.Value, "in") {
		a.bareWrites = append(a.bareWrites, identUse{name: cur.Value, offset: baseOffset + cur.Offset})

		return
	}

	// Struct literal key `{ a = 1 }`.
	if inStructLit && next.Kind == parser.TokEquals {
		return
	}

	// Bare or dotted/indexed assignment `x = ...`, `x.y = ...`, `x["y"] = ...`.
	if isAssignLHS(toks, idx) {
		a.bareWrites = append(a.bareWrites, identUse{name: cur.Value, offset: baseOffset + cur.Offset})

		return
	}

	// Inside an inner closure, `arguments` refers to the closure's own arguments,
	// so referencing an enclosing function's parameter unscoped is required by
	// CFML lexical capture and is not ARG_VAR_MIXED.
	if !inClosure {
		a.bareReads = append(a.bareReads, identUse{name: cur.Value, offset: baseOffset + cur.Offset})
	}

	// Matching ImplicitScopeChecker: only single-identifier reads (not dotted
	// object/member chains like `persist.getThing(...)` or `_objInit.valLeft(...)`)
	// and non-query-loop properties are candidates for IMPLICIT_SCOPE.
	if next.Kind != parser.TokDot && !queryLoopProperties[low] {
		a.implicitReads = append(a.implicitReads, identUse{name: cur.Value, offset: baseOffset + cur.Offset})
	}
}

func recordScopedMemberIdent(toks []parser.Token, idx, baseOffset int, prev2, prev, cur parser.Token, a *funcScopeAnalysis) bool {
	if prev.Kind != parser.TokDot || prev2.Kind != parser.TokIdent {
		return false
	}

	low := strings.ToLower(cur.Value)
	isAssign := isAssignLHS(toks, idx)

	switch strings.ToLower(prev2.Value) {
	case "arguments":
		a.argScoped[low] = true
		if isAssign {
			a.argAssign[low] = true
		}

		return true
	case "local":
		if isAssign {
			a.localAssign[low] = true
		} else {
			a.localOrVarReads = append(a.localOrVarReads, identUse{name: cur.Value, offset: baseOffset + cur.Offset})
		}

		return true
	case "variables":
		if !isAssign {
			a.localOrVarReads = append(a.localOrVarReads, identUse{name: cur.Value, offset: baseOffset + cur.Offset})
		}

		return true
	default:
		return false
	}
}

func tokAt(toks []parser.Token, i int) parser.Token {
	if i < 0 || i >= len(toks) {
		return parser.Token{}
	}

	return toks[i]
}

// scanCFMLScriptTokens tokenizes CFScript/CFML expressions while properly
// keeping CFML string literals with nested quotes inside `#...#` interpolations
// (e.g. `"#result[i]["item#j#_desc"]#"`) as single TokString tokens.
func scanCFMLScriptTokens(text string) []parser.Token {
	if !strings.ContainsRune(text, '#') && !strings.Contains(text, "<!---") {
		sc := parser.NewSourceScanner(text)

		var toks []parser.Token
		for t := sc.NextSkipComments(); t.Kind != parser.TokEOF; t = sc.NextSkipComments() {
			toks = append(toks, t)
		}

		return toks
	}

	var toks []parser.Token

	pos := 0
	for pos < len(text) {
		// Find next quote or comment start; scan non-string segments with SourceScanner.
		qIdx := findNextQuoteOrComment(text, pos)
		if qIdx < 0 {
			toks = appendScannerTokens(toks, text[pos:], pos)

			break
		}

		if qIdx > pos {
			toks = appendScannerTokens(toks, text[pos:qIdx], pos)
		}

		if strings.HasPrefix(text[qIdx:], "<!---") {
			pos = parser.SkipCFMLComment(text, qIdx)

			continue
		}

		if strings.HasPrefix(text[qIdx:], "//") {
			if nl := strings.IndexByte(text[qIdx:], '\n'); nl >= 0 {
				pos = qIdx + nl + 1
			} else {
				break
			}

			continue
		}

		if strings.HasPrefix(text[qIdx:], "/*") {
			if end := strings.Index(text[qIdx+2:], "*/"); end >= 0 {
				pos = qIdx + 2 + end + 2
			} else {
				break
			}

			continue
		}

		end := readCFMLStringEnd(text, qIdx)
		toks = append(toks, parser.Token{
			Kind:   parser.TokString,
			Value:  text[qIdx:end],
			Offset: qIdx,
		})
		pos = end
	}

	return toks
}

func appendScannerTokens(dst []parser.Token, segment string, baseOffset int) []parser.Token {
	sc := parser.NewSourceScanner(segment)
	for t := sc.NextSkipComments(); t.Kind != parser.TokEOF; t = sc.NextSkipComments() {
		t.Offset += baseOffset
		dst = append(dst, t)
	}

	return dst
}

func findNextQuoteOrComment(s string, start int) int {
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '"', '\'':
			return i
		case '/':
			if i+1 < len(s) && (s[i+1] == '/' || s[i+1] == '*') {
				return i
			}
		case '<':
			if strings.HasPrefix(s[i:], "<!---") {
				return i
			}
		}
	}

	return -1
}

// readCFMLStringEnd returns the byte offset just past the closing quote of the
// CFML string literal starting at start, handling doubled-quote escapes and
// nested hash expressions that may contain inner quotes.
func readCFMLStringEnd(s string, start int) int {
	quote := s[start]
	i := start + 1

	for i < len(s) {
		c := s[i]
		if c == quote {
			if i+1 < len(s) && s[i+1] == quote {
				i += 2

				continue
			}

			return i + 1
		}

		if c == '#' {
			if i+1 < len(s) && s[i+1] == '#' {
				i += 2

				continue
			}

			i = skipHashExprInString(s, i+1)

			continue
		}

		i++
	}

	return len(s)
}

func skipHashExprInString(s string, start int) int {
	i := start
	for i < len(s) {
		c := s[i]
		if c == '#' {
			return i + 1
		}

		if c == '"' || c == '\'' {
			i = readCFMLStringEnd(s, i)

			continue
		}

		i++
	}

	return len(s)
}

// forEachStringHashExpr invokes fn for each unescaped `#expr#` interpolation
// inside a CFML string literal token, passing the expression text and its byte
// offset relative to tok.Offset while respecting nested quotes inside `#...#`.
func forEachStringHashExpr(tok parser.Token, fn func(expr string, relOff int)) {
	val := tok.Value
	if len(val) < 4 || !strings.ContainsRune(val, '#') {
		return
	}

	// Strip surrounding quotes; track offset 1 for the opening quote.
	inner := val[1 : len(val)-1]

	for i := 0; i < len(inner); {
		if inner[i] != '#' {
			i++

			continue
		}

		// Escaped `##`.
		if i+1 < len(inner) && inner[i+1] == '#' {
			i += 2

			continue
		}

		end := skipHashExprInString(inner, i+1)
		if end <= i+1 || inner[end-1] != '#' {
			return
		}

		expr := inner[i+1 : end-1]
		if strings.TrimSpace(expr) != "" {
			fn(expr, 1+i+1)
		}

		i = end
	}
}

// isArrowParamList reports whether the `(` at lpIdx is an arrow-function
// parameter list `( a, b ) => ...`.
func isArrowParamList(toks []parser.Token, lpIdx int) bool {
	depth := 1
	j := lpIdx + 1

	for j < len(toks) && depth > 0 {
		switch toks[j].Kind {
		case parser.TokLParen:
			depth++
		case parser.TokRParen:
			depth--
		default:
		}

		j++
	}

	if depth != 0 || j+1 >= len(toks) {
		return false
	}

	return toks[j].Kind == parser.TokEquals && toks[j+1].Kind == parser.TokGT && toks[j+1].Offset == toks[j].Offset+1
}

// collectCatchVarName records the catch variable from a `catch (...)` header
// starting at the `(` token index lpIdx (handles `catch (e)`, `catch (any e)`,
// `catch ("Custom.Type" e)`, `catch (java.lang.Exception e)`, `catch (any var e)`).
func collectCatchVarName(toks []parser.Token, lpIdx int, dst map[string]bool) {
	depth := 1

	var lastIdent string

	for j := lpIdx + 1; j < len(toks) && depth > 0; j++ {
		switch toks[j].Kind {
		case parser.TokLParen:
			depth++
		case parser.TokRParen:
			depth--
		case parser.TokIdent:
			if depth == 1 {
				lastIdent = toks[j].Value
			}
		default:
		}
	}

	if lastIdent != "" {
		dst[strings.ToLower(lastIdent)] = true
	}
}

// collectClosureParamNames extracts parameter names from a `function(...)`
// token sequence starting at the `(` token index lpIdx.
func collectClosureParamNames(toks []parser.Token, lpIdx int, dst map[string]bool) {
	depth := 1

	var param []parser.Token

	flush := func() {
		var head []parser.Token

		for _, t := range param {
			if t.Kind == parser.TokEquals {
				break
			}

			head = append(head, t)
		}

		if len(head) > 0 {
			last := head[len(head)-1]
			if last.Kind == parser.TokIdent {
				dst[strings.ToLower(last.Value)] = true
			}
		}

		param = param[:0]
	}

	for j := lpIdx + 1; j < len(toks) && depth > 0; j++ {
		t := toks[j]
		switch t.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++

			param = append(param, t)
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--
			if depth == 0 {
				flush()
			} else {
				param = append(param, t)
			}
		case parser.TokComma:
			if depth == 1 {
				flush()
			} else {
				param = append(param, t)
			}
		default:
			param = append(param, t)
		}
	}
}

// isAssignLHS reports whether the identifier at toks[idx] is the root of an
// assignment LHS (`x = ...`, `x.y = ...`, `x["y"] = ...`, `x.y.[z] = ...`).
func isAssignLHS(toks []parser.Token, idx int) bool {
	j := idx + 1
	for j < len(toks) {
		switch toks[j].Kind {
		case parser.TokDot:
			if j+1 >= len(toks) {
				return false
			}

			if toks[j+1].Kind == parser.TokIdent || toks[j+1].Kind == parser.TokNumber {
				j += 2

				continue
			}

			if toks[j+1].Kind == parser.TokLBracket {
				j = skipBracketGroup(toks, j+1)
				if j < 0 {
					return false
				}

				continue
			}

			return false
		case parser.TokLBracket:
			j = skipBracketGroup(toks, j)
			if j < 0 {
				return false
			}
		case parser.TokEquals:
			return isStandaloneEquals(toks[j-1], toks[j], tokAt(toks, j+1))
		default:
			return false
		}
	}

	return false
}

func skipBracketGroup(toks []parser.Token, start int) int {
	depth := 1

	for j := start + 1; j < len(toks); j++ {
		switch toks[j].Kind {
		case parser.TokLBracket:
			depth++
		case parser.TokRBracket:
			depth--
			if depth == 0 {
				return j + 1
			}
		default:
		}
	}

	return -1
}

func checkMissingVar(fc *FileContext) []cflint.Issue {
	qSpans := queryLoopSpans(fc.Text())

	return checkUnscopedUses(fc, func(a *funcScopeAnalysis) []identUse {
		if a.localModeModern {
			return nil
		}

		if len(qSpans) == 0 {
			return a.bareWrites
		}

		filtered := make([]identUse, 0, len(a.bareWrites))
		for _, u := range a.bareWrites {
			if !offsetInAnyRange(u.offset, qSpans) {
				filtered = append(filtered, u)
			}
		}

		return filtered
	}, "Variable %s is not declared with a var statement.")
}

func issueInFunc(fc *FileContext, offset int, msg, variable, funcName string) cflint.Issue {
	iss := fc.IssueAt(offset, msg, variable)
	if len(iss.Locations) > 0 {
		iss.Locations[0].Function = funcName
	}

	return iss
}

func checkUnscopedUses(fc *FileContext, pick func(*funcScopeAnalysis) []identUse, msgFmt string) []cflint.Issue {
	compVars := collectComponentVars(fc)

	var out []cflint.Issue

	for _, a := range fc.functionScopes() {
		reported := map[string]bool{}

		for _, u := range pick(a) {
			low := strings.ToLower(u.name)
			if reported[low] || a.params[low] || a.localAssign[low] || a.argAssign[low] || a.catchVars[low] || compVars[low] {
				continue
			}

			if _, hasVar := a.vars[low]; hasVar {
				continue
			}

			reported[low] = true

			msg := fmt.Sprintf(msgFmt, u.name)
			out = append(out, issueInFunc(fc, u.offset, msg, u.name, a.funcName))
		}
	}

	return out
}

func checkArgVarConflict(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.functionScopes() {
		reported := map[string]bool{}

		for name, off := range a.vars {
			if a.params[name] {
				reported[name] = true
				msg := fmt.Sprintf("Variable %s conflicts with function argument of the same name.", name)
				out = append(out, issueInFunc(fc, off, msg, name, a.funcName))
			}
		}

		// Matching ArgVarChecker: unscoped assignment to an argument name in a
		// function body also triggers ARG_VAR_CONFLICT.
		for _, w := range a.bareWrites {
			low := strings.ToLower(w.name)
			if !a.params[low] || reported[low] {
				continue
			}

			reported[low] = true
			msg := fmt.Sprintf("Variable %s conflicts with function argument of the same name.", w.name)
			out = append(out, issueInFunc(fc, w.offset, msg, w.name, a.funcName))
		}
	}

	return out
}

func checkArgVarMixed(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.functionScopes() {
		reported := map[string]bool{}

		// Merge bareReads and localOrVarReads in source order so the first
		// offending reference wins per argument (matching ArgVarChecker).
		all := make([]identUse, 0, len(a.bareReads)+len(a.localOrVarReads))
		all = append(all, a.bareReads...)
		all = append(all, a.localOrVarReads...)
		slices.SortStableFunc(all, func(x, y identUse) int { return cmp.Compare(x.offset, y.offset) })

		for _, r := range all {
			low := strings.ToLower(r.name)
			if reported[low] || !a.params[low] {
				continue
			}

			reported[low] = true

			msg := fmt.Sprintf("Variable %s should not be referenced in local and argument scope.", r.name)
			out = append(out, issueInFunc(fc, r.offset, msg, r.name, a.funcName))
		}
	}

	return out
}

func checkImplicitScope(fc *FileContext) []cflint.Issue {
	compVars := collectComponentVars(fc)
	qSpans := queryLoopSpans(fc.Text())

	if strings.EqualFold(filepath.Ext(fc.Path), ".cfm") {
		return checkCFMImplicitScope(fc, compVars, qSpans)
	}

	var out []cflint.Issue

	for _, a := range fc.functionScopes() {
		reported := map[string]bool{}

		// Matching ImplicitScopeChecker: bare writes establish variables/local
		// scope in source order, so a read before the first write is still
		// flagged as IMPLICIT_SCOPE.
		firstWrite := firstWriteOffsets(a.bareWrites)

		for _, u := range a.implicitReads {
			low := strings.ToLower(u.name)
			if reported[low] || a.params[low] || a.localAssign[low] || a.catchVars[low] || compVars[low] || offsetInAnyRange(u.offset, qSpans) {
				continue
			}

			if wOff, ok := firstWrite[low]; ok && wOff <= u.offset {
				continue
			}

			if _, hasVar := a.vars[low]; hasVar {
				continue
			}

			reported[low] = true

			msg := fmt.Sprintf("Variable %s is not explicitly scoped.", u.name)
			out = append(out, issueInFunc(fc, u.offset, msg, u.name, a.funcName))
		}
	}

	return out
}

// checkCFMImplicitScope matches ImplicitScopeChecker's file-wide mode for .cfm
// templates: scans script blocks, expression tags, and #...# interpolations in
// the template and flags single-identifier reads never established in the file.
func checkCFMImplicitScope(fc *FileContext, compVars map[string]bool, qSpans [][2]int) []cflint.Issue {
	src := fc.Text()
	a := &funcScopeAnalysis{
		params:      make(map[string]bool),
		vars:        make(map[string]int),
		localAssign: make(map[string]bool),
		argAssign:   make(map[string]bool),
		catchVars:   make(map[string]bool),
		argScoped:   make(map[string]bool),
	}

	// Collect tag-bound variables (<cfparam name>, <cfquery name>, <cfloop index/item>, etc.).
	for _, tag := range parser.CFTags(src, "cfparam", "cfloop", "cfquery", "cfhttp", "cffile", "cfsavecontent", "cfdirectory", "cfprocresult", "cfstoredproc", "cfinvoke", "cfwddx", "cfxml") {
		if tag.Close {
			continue
		}

		if strings.EqualFold(tag.Name, "cfparam") {
			if name, ok := tag.Attr("name"); ok {
				recordOutputVarName(name, tag.Start, a)
			}

			continue
		}

		for attr := range scriptTagOutputAttrs[strings.ToLower(tag.Name)] {
			if val, ok := tag.Attr(attr); ok {
				recordOutputVarName(val, tag.Start, a)
			}
		}
	}

	cfoutputDepth := 0

	for _, r := range fc.Regions() {
		if r.Kind == parser.RegionScript {
			scanScriptBodyScope(r.Text, r.Offset, a)

			continue
		}

		// Scan expression tags (<cfset>, <cfif>, <cfelseif>, <cfreturn>).
		for _, tag := range parser.CFTags(r.Text, "cfset", "cfif", "cfelseif", "cfreturn") {
			if tag.Close {
				continue
			}

			txt := tag.Text()
			if len(txt) <= len(tag.Name)+2 {
				continue
			}

			innerStart := len(tag.Name) + 1
			scanScriptBodyScope(txt[innerStart:len(txt)-1], r.Offset+tag.Start+innerStart, a)
		}

		// Scan `#expr#` interpolations only inside `<cfoutput>` blocks and
		// CF tag attributes (never in raw HTML outside `<cfoutput>`).
		scanTagHashesForScope(r.Text, r.Offset, &cfoutputDepth, a)
	}

	slices.SortStableFunc(a.implicitReads, func(x, y identUse) int { return cmp.Compare(x.offset, y.offset) })

	firstWrite := firstWriteOffsets(a.bareWrites)

	var out []cflint.Issue

	reported := map[string]bool{}

	for _, u := range a.implicitReads {
		low := strings.ToLower(u.name)
		if reported[low] || a.params[low] || a.localAssign[low] || a.catchVars[low] || compVars[low] || offsetInAnyRange(u.offset, qSpans) {
			continue
		}

		if wOff, ok := firstWrite[low]; ok && wOff <= u.offset {
			continue
		}

		if _, hasVar := a.vars[low]; hasVar {
			continue
		}

		reported[low] = true

		msg := fmt.Sprintf("Variable %s is not explicitly scoped.", u.name)
		out = append(out, fc.IssueAt(u.offset, msg, u.name))
	}

	return out
}

// firstWriteOffsets returns the earliest byte offset at which each lowercased
// variable name is assigned.
func firstWriteOffsets(writes []identUse) map[string]int {
	first := make(map[string]int, len(writes))
	for _, w := range writes {
		low := strings.ToLower(w.name)
		if prev, ok := first[low]; !ok || w.offset < prev {
			first[low] = w.offset
		}
	}

	return first
}

func scanTagHashesForScope(text string, baseOffset int, cfoutputDepth *int, a *funcScopeAnalysis) {
	for i := 0; i < len(text); {
		if strings.HasPrefix(text[i:], "<!---") {
			i = parser.SkipCFMLComment(text, i)

			continue
		}

		if text[i] == '<' {
			if next, handled := scanCFTagForScope(text, i, baseOffset, cfoutputDepth, a); handled {
				i = next

				continue
			}
		}

		if *cfoutputDepth <= 0 || text[i] != '#' {
			i++

			continue
		}

		if i+1 < len(text) && text[i+1] == '#' {
			i += 2

			continue
		}

		end := skipHashExprInString(text, i+1)
		if end <= i+1 || text[end-1] != '#' {
			return
		}

		expr := text[i+1 : end-1]
		if strings.TrimSpace(expr) != "" {
			scanScriptBodyScope(expr, baseOffset+i+1, a)
		}

		i = end
	}
}

// scanCFTagForScope handles a `<cf...>` or `</cf...>` tag starting at pos:
// updates cfoutputDepth for `<cfoutput>` / `</cfoutput>` and scans `#expr#`
// interpolations inside opening CF tag attributes (skipping `<cfset>`, `<cfif>`,
// `<cfelseif>`, `<cfreturn>` which are already scanned as expressions).
func scanCFTagForScope(text string, pos, baseOffset int, cfoutputDepth *int, a *funcScopeAnalysis) (int, bool) {
	j := pos + 1
	closing := false

	if j < len(text) && text[j] == '/' {
		closing = true
		j++
	}

	if j+2 > len(text) || !strings.EqualFold(text[j:j+2], "cf") {
		return pos, false
	}

	nameStart := j
	for j < len(text) && (isIdentChar(text[j]) || (text[j] >= 'A' && text[j] <= 'Z') || text[j] == '_') {
		j++
	}

	tagName := strings.ToLower(text[nameStart:j])
	end := findTagCloseOffset(text, j)

	if closing {
		if tagName == "cfoutput" && *cfoutputDepth > 0 {
			*cfoutputDepth--
		}

		return end, true
	}

	selfClosing := end >= 2 && text[end-2] == '/'
	if tagName == "cfoutput" && !selfClosing {
		*cfoutputDepth++
	}

	if tagName != "cfset" && tagName != "cfif" && tagName != "cfelseif" && tagName != "cfreturn" && end > j {
		attrEnd := end
		if text[end-1] == '>' {
			attrEnd--
		}

		attrText := text[j:attrEnd]
		if strings.ContainsRune(attrText, '#') {
			scanHashExprsInRange(attrText, baseOffset+j, a)
		}
	}

	return end, true
}

func findTagCloseOffset(text string, start int) int {
	var quote byte

	for i := start; i < len(text); i++ {
		c := text[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}

			continue
		}

		if c == '"' || c == '\'' {
			quote = c

			continue
		}

		if c == '>' {
			return i + 1
		}
	}

	return len(text)
}

func scanHashExprsInRange(text string, baseOffset int, a *funcScopeAnalysis) {
	for i := 0; i < len(text); {
		if text[i] != '#' {
			i++

			continue
		}

		if i+1 < len(text) && text[i+1] == '#' {
			i += 2

			continue
		}

		end := skipHashExprInString(text, i+1)
		if end <= i+1 || text[end-1] != '#' {
			return
		}

		expr := text[i+1 : end-1]
		if strings.TrimSpace(expr) != "" {
			scanScriptBodyScope(expr, baseOffset+i+1, a)
		}

		i = end
	}
}

// queryLoopSpans returns the body byte ranges [openEnd, closeStart) of every
// `<cfloop query="...">...</cfloop>` and `<cfoutput query="...">...</cfoutput>`
// in src. Inside these spans, unscoped identifiers resolve to query columns and
// should not trigger IMPLICIT_SCOPE.
func queryLoopSpans(src string) [][2]int {
	if parser.IndexFold(src, "query") < 0 {
		return nil
	}

	type frame struct {
		openEnd int
		isQuery bool
	}

	var (
		loopStack []frame
		outStack  []frame
		spans     [][2]int
	)

	for _, tag := range parser.CFTags(src, "cfloop", "cfoutput") {
		stack := &loopStack
		if tag.Name == "cfoutput" {
			stack = &outStack
		}

		if tag.Close {
			if n := len(*stack); n > 0 {
				top := (*stack)[n-1]
				*stack = (*stack)[:n-1]

				if top.isQuery && tag.Start > top.openEnd {
					spans = append(spans, [2]int{top.openEnd, tag.Start})
				}
			}

			continue
		}

		if strings.HasSuffix(strings.TrimSpace(tag.Text()), "/>") {
			continue
		}

		_, hasQuery := tag.Attr("query")
		*stack = append(*stack, frame{openEnd: tag.End, isQuery: hasQuery})
	}

	return spans
}

func offsetInAnyRange(off int, ranges [][2]int) bool {
	for _, r := range ranges {
		if off >= r[0] && off < r[1] {
			return true
		}
	}

	return false
}

// collectComponentVars collects declared component properties (`property foo;`,
// `property name="foo";`, `<cfproperty name="foo">`), pseudo-constructor
// variables in `.cfc` files, and `variables.<name>` references across script
// blocks and `<cfset>` tags so component fields are not flagged as
// IMPLICIT_SCOPE (matching ImplicitScopeChecker).
func collectComponentVars(fc *FileContext) map[string]bool {
	vars := map[string]bool{}
	src := fc.Text()

	if strings.EqualFold(filepath.Ext(fc.Path), ".cfc") {
		for _, v := range fc.ParseResult().AllVars() {
			if v.FuncStart < 0 && v.Scope == parser.ScopeVariables {
				vars[strings.ToLower(v.Name)] = true
			}
		}
	}

	for _, tag := range parser.CFTags(src, "cfproperty") {
		if !tag.Close {
			if name, ok := tag.Attr("name"); ok && name != "" {
				vars[strings.ToLower(strings.TrimSpace(name))] = true
			}
		}
	}

	scanVarsInText := func(text string) {
		sc := parser.NewSourceScanner(text)

		var prev2, prev parser.Token

		cur := sc.NextSkipComments()
		for cur.Kind != parser.TokEOF {
			next := sc.NextSkipComments()

			// `variables.<name>` (matches ImplicitScopeChecker.variableScopedVariables)
			if cur.Kind == parser.TokIdent && prev.Kind == parser.TokDot &&
				prev2.Kind == parser.TokIdent && strings.EqualFold(prev2.Value, "variables") {
				vars[strings.ToLower(cur.Value)] = true
			}

			// Script `property [type] name;` or `property name="foo" ...;`
			if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, "property") && prev.Kind != parser.TokDot {
				collectScriptPropertyName(next, sc, vars)

				prev2 = prev
				prev = cur
				cur = sc.NextSkipComments()

				continue
			}

			prev2 = prev
			prev = cur
			cur = next
		}
	}

	for _, r := range fc.Regions() {
		if r.Kind == parser.RegionScript {
			scanVarsInText(r.Text)
		}
	}

	for _, tag := range parser.CFTags(src, "cfset", "cfif", "cfelseif", "cfreturn") {
		if tag.Close || tag.Name != "cfset" {
			continue
		}

		txt := tag.Text()
		if len(txt) <= len(tag.Name)+2 {
			continue
		}

		scanVarsInText(txt[len(tag.Name)+1 : len(txt)-1])
	}

	return vars
}

func collectScriptPropertyName(first parser.Token, sc *parser.Scanner, dst map[string]bool) {
	var head []parser.Token

	for t := first; t.Kind != parser.TokEOF && t.Kind != parser.TokSemicolon && t.Kind != parser.TokLBrace; t = sc.NextSkipComments() {
		if t.Kind == parser.TokIdent && strings.EqualFold(t.Value, "name") {
			if eq := sc.NextSkipComments(); eq.Kind == parser.TokEquals {
				if val := sc.NextSkipComments(); val.Kind == parser.TokString && len(val.Value) >= 2 {
					dst[strings.ToLower(strings.TrimSpace(val.Value[1:len(val.Value)-1]))] = true

					return
				}
			}

			continue
		}

		if t.Kind == parser.TokEquals {
			break
		}

		head = append(head, t)
	}

	if len(head) > 0 {
		last := head[len(head)-1]
		if last.Kind == parser.TokIdent {
			dst[strings.ToLower(last.Value)] = true
		}
	}
}
