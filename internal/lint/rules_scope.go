package lint

import (
	"fmt"
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
	params      map[string]bool
	vars        map[string]int // lowercased name -> first var decl offset
	localAssign map[string]bool
	tagBound    map[string]bool
	catchVars   map[string]bool
	bareWrites  []identUse
	bareReads   []identUse
	argScoped   map[string]bool
}

type identUse struct {
	name   string
	offset int
}

func analyzeFunctionScope(content string, span *parser.FunctionSpan) *funcScopeAnalysis {
	a := &funcScopeAnalysis{
		params:      make(map[string]bool),
		vars:        make(map[string]int),
		localAssign: make(map[string]bool),
		tagBound:    make(map[string]bool),
		catchVars:   make(map[string]bool),
		argScoped:   make(map[string]bool),
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

	// 2. Collect tag-bound variables inside the function body.
	for _, tag := range parser.CFTags(body, "cfloop", "cfquery", "cfhttp", "cffile", "cfsavecontent", "cfdirectory", "cfprocresult") {
		if tag.Close {
			continue
		}

		for _, attr := range []string{"index", "item", "name", "result", "variable"} {
			if val, ok := tag.Attr(attr); ok && val != "" {
				base := strings.ToLower(strings.TrimSpace(val))
				if prefix, after, found := strings.Cut(base, "."); found {
					if prefix == "local" {
						a.localAssign[after] = true
					}
				} else {
					a.tagBound[base] = true
				}
			}
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

	return a
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

func scanScriptBodyScope(text string, baseOffset int, a *funcScopeAnalysis) {
	sc := parser.NewSourceScanner(text)

	var (
		prev2, prev parser.Token
		braceKinds  []bool // true = struct literal brace, false = block brace
	)

	cur := sc.NextSkipComments()
	for cur.Kind != parser.TokEOF {
		next := sc.NextSkipComments()

		switch cur.Kind {
		case parser.TokLBrace:
			isStruct := prev.Kind == parser.TokEquals || prev.Kind == parser.TokLParen ||
				prev.Kind == parser.TokComma || prev.Kind == parser.TokLBracket ||
				prev.Kind == parser.TokColon || prev.Kind == parser.TokQuestion ||
				(prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "return"))
			braceKinds = append(braceKinds, isStruct)
		case parser.TokRBrace:
			if len(braceKinds) > 0 {
				braceKinds = braceKinds[:len(braceKinds)-1]
			}
		case parser.TokIdent:
			inStructLit := len(braceKinds) > 0 && braceKinds[len(braceKinds)-1]
			recordScriptIdent(baseOffset, prev2, prev, cur, next, inStructLit, a)
		default:
		}

		prev2 = prev
		prev = cur
		cur = next
	}
}

func recordScriptIdent(baseOffset int, prev2, prev, cur, next parser.Token, inStructLit bool, a *funcScopeAnalysis) {
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

	// Skip member accesses (`.foo`), function calls (`foo(`), struct keys (`foo:`), and keywords.
	if prev.Kind == parser.TokDot || next.Kind == parser.TokLParen || next.Kind == parser.TokColon {
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

	// Struct literal key `{ a = 1 }`.
	if inStructLit && next.Kind == parser.TokEquals {
		return
	}

	// Bare assignment `x = ...` (exclude `==`, `!=`, etc. by checking single `=`).
	if next.Kind == parser.TokEquals && !strings.HasPrefix(next.Value, "==") {
		a.bareWrites = append(a.bareWrites, identUse{name: cur.Value, offset: baseOffset + cur.Offset})

		return
	}

	a.bareReads = append(a.bareReads, identUse{name: cur.Value, offset: baseOffset + cur.Offset})
}

func checkMissingVar(fc *FileContext) []cflint.Issue {
	return checkUnscopedUses(fc, func(a *funcScopeAnalysis) []identUse { return a.bareWrites }, "Variable %s is not declared with a var statement.")
}

func checkUnscopedUses(fc *FileContext, pick func(*funcScopeAnalysis) []identUse, msgFmt string) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.functionScopes() {
		reported := map[string]bool{}

		for _, u := range pick(a) {
			low := strings.ToLower(u.name)
			if reported[low] || a.params[low] || a.localAssign[low] || a.tagBound[low] || a.catchVars[low] {
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
		for name, off := range a.vars {
			if a.params[name] {
				msg := fmt.Sprintf("Variable %s conflicts with function argument of the same name.", name)
				out = append(out, fc.IssueAt(off, msg, name))
			}
		}
	}

	return out
}

func checkArgVarMixed(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.functionScopes() {
		reported := map[string]bool{}

		checkUse := func(u identUse) {
			low := strings.ToLower(u.name)
			if reported[low] || !a.params[low] || !a.argScoped[low] {
				return
			}

			reported[low] = true

			msg := fmt.Sprintf("Argument %s is used both scoped (arguments.%s) and unscoped.", u.name, u.name)
			out = append(out, fc.IssueAt(u.offset, msg, u.name))
		}

		for _, r := range a.bareReads {
			checkUse(r)
		}

		for _, w := range a.bareWrites {
			checkUse(w)
		}
	}

	return out
}

func checkImplicitScope(fc *FileContext) []cflint.Issue {
	return checkUnscopedUses(fc, func(a *funcScopeAnalysis) []identUse { return a.bareReads }, "Variable %s is not explicitly scoped.")
}
