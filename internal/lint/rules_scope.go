package lint

import (
	"cmp"
	"fmt"
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
	"cfcatch": true, "caller": true, "attributes": true, "thistag": true, "thread": true,
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
	localModeModern bool
	params          map[string]bool
	vars            map[string]int // lowercased name -> first var decl offset
	localAssign     map[string]bool
	catchVars       map[string]bool
	bareWrites      []identUse
	bareReads       []identUse
	argScoped       map[string]bool
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
		localModeModern: span.Start >= 0 && span.Body > span.Start && hasModernLocalMode(content[span.Start:span.Body]),
		params:          make(map[string]bool),
		vars:            make(map[string]int),
		localAssign:     make(map[string]bool),
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

	for _, tag := range parser.CFTags(body, "cfargument") {
		if !tag.Close {
			if name, ok := tag.Attr("name"); ok && name != "" {
				a.params[strings.ToLower(strings.TrimSpace(name))] = true
			}
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

		for _, tag := range parser.CFTags(body, "cfset") {
			if tag.Close {
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

		for _, t := range toks {
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

type parenFrame struct {
	isFor    bool
	callName string // lowercased callee name when this `(` is a call arg list
}

var controlParenKeywords = map[string]bool{
	"if": true, "elseif": true, "while": true, "switch": true, "catch": true, "for": true, "function": true,
}

var scriptTagStmtKeywords = map[string]bool{
	"lock": true, "transaction": true, "param": true, "savecontent": true,
	"setting": true, "mail": true, "http": true, "query": true, "storedproc": true,
	"wddx": true, "file": true, "directory": true, "thread": true, "cookie": true,
	"header": true, "content": true, "location": true, "log": true, "trace": true,
	"cache": true, "document": true, "pdf": true, "zip": true, "ftp": true,
}

func scanScriptBodyScope(text string, baseOffset int, a *funcScopeAnalysis) {
	sc := parser.NewSourceScanner(text)

	var toks []parser.Token
	for t := sc.NextSkipComments(); t.Kind != parser.TokEOF; t = sc.NextSkipComments() {
		toks = append(toks, t)
	}

	var (
		braceKinds  []bool // true = struct literal brace, false = block brace
		parens      []parenFrame
		tagStmtName string // non-empty inside `lock type="exclusive" name="..." {`
	)

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
			}
		case parser.TokLBrace:
			tagStmtName = ""
			isStruct := prev.Kind == parser.TokEquals || prev.Kind == parser.TokLParen ||
				prev.Kind == parser.TokComma || prev.Kind == parser.TokLBracket ||
				prev.Kind == parser.TokColon || prev.Kind == parser.TokQuestion ||
				(prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "return"))
			braceKinds = append(braceKinds, isStruct)
		case parser.TokRBrace:
			if len(braceKinds) > 0 {
				braceKinds = braceKinds[:len(braceKinds)-1]
			}
		case parser.TokLParen:
			f := parenFrame{}

			if prev.Kind == parser.TokIdent {
				lowPrev := strings.ToLower(prev.Value)
				if lowPrev == "for" {
					f.isFor = true
				} else if !controlParenKeywords[lowPrev] {
					f.callName = lowPrev
				}
			}

			parens = append(parens, f)
		case parser.TokRParen:
			if len(parens) > 0 {
				parens = parens[:len(parens)-1]
			}
		case parser.TokIdent:
			// Detect script-tag statement header: `lock type="..." name="..." {`.
			if len(parens) == 0 && tagStmtName == "" && prev.Kind != parser.TokDot && next.Kind == parser.TokIdent {
				if low := strings.ToLower(cur.Value); scriptTagStmtKeywords[low] {
					tagStmtName = "cf" + low

					continue
				}
			}

			inStructLit := len(braceKinds) > 0 && braceKinds[len(braceKinds)-1]

			var topParen parenFrame
			if len(parens) > 0 {
				topParen = parens[len(parens)-1]
			}

			recordScriptIdent(toks, i, baseOffset, prev2, prev, cur, next, inStructLit, topParen, tagStmtName, a)
		default:
		}
	}
}

func recordScriptIdent(toks []parser.Token, idx, baseOffset int, prev2, prev, cur, next parser.Token, inStructLit bool, topParen parenFrame, tagStmtName string, a *funcScopeAnalysis) {
	low := strings.ToLower(cur.Value)

	// Record `arguments.<name>` reads/writes.
	if prev.Kind == parser.TokDot && prev2.Kind == parser.TokIdent && strings.EqualFold(prev2.Value, "arguments") {
		a.argScoped[low] = true

		return
	}

	// Record `local.<name> = ...`.
	if prev.Kind == parser.TokDot && prev2.Kind == parser.TokIdent && strings.EqualFold(prev2.Value, "local") {
		if next.Kind == parser.TokEquals {
			a.localAssign[low] = true
		}

		return
	}

	// Skip member accesses (`.foo`), function calls (`foo(`), struct keys (`foo:`),
	// `new pkg.Component(...)`, and keywords/scopes.
	if prev.Kind == parser.TokDot || next.Kind == parser.TokLParen || next.Kind == parser.TokColon {
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

	a.bareReads = append(a.bareReads, identUse{name: cur.Value, offset: baseOffset + cur.Offset})
}

func tokAt(toks []parser.Token, i int) parser.Token {
	if i < 0 || i >= len(toks) {
		return parser.Token{}
	}

	return toks[i]
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

			if toks[j+1].Kind == parser.TokIdent {
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
	return checkUnscopedUses(fc, func(a *funcScopeAnalysis) []identUse {
		if a.localModeModern {
			return nil
		}

		return a.bareWrites
	}, "Variable %s is not declared with a var statement.")
}

func checkUnscopedUses(fc *FileContext, pick func(*funcScopeAnalysis) []identUse, msgFmt string) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.functionScopes() {
		reported := map[string]bool{}

		for _, u := range pick(a) {
			low := strings.ToLower(u.name)
			if reported[low] || a.params[low] || a.localAssign[low] || a.catchVars[low] {
				continue
			}

			if _, hasVar := a.vars[low]; hasVar {
				continue
			}

			reported[low] = true

			msg := fmt.Sprintf(msgFmt, u.name)
			out = append(out, fc.IssueAt(u.offset, msg, u.name))
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
				out = append(out, fc.IssueAt(off, msg, name))
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
			out = append(out, fc.IssueAt(w.offset, msg, w.name))
		}
	}

	return out
}

func checkArgVarMixed(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.functionScopes() {
		reported := map[string]bool{}

		// Matching ArgVarChecker: referencing a function argument without the
		// `arguments.` scope triggers ARG_VAR_MIXED once per argument per function.
		for _, r := range a.bareReads {
			low := strings.ToLower(r.name)
			if reported[low] || !a.params[low] {
				continue
			}

			reported[low] = true

			msg := fmt.Sprintf("Variable %s should not be referenced in local and argument scope.", r.name)
			out = append(out, fc.IssueAt(r.offset, msg, r.name))
		}
	}

	return out
}

func checkImplicitScope(fc *FileContext) []cflint.Issue {
	return checkUnscopedUses(fc, func(a *funcScopeAnalysis) []identUse { return a.bareReads }, "Variable %s is not explicitly scoped.")
}
