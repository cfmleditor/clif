package lint

import (
	"fmt"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerUnusedRules(DefaultRegistry)
}

func registerUnusedRules(reg *Registry) {
	reg.Register(Rule{
		Code:            "UNUSED_LOCAL_VARIABLE",
		DefaultSeverity: "INFO",
		Check:           checkUnusedLocalVariable,
	})
	reg.Register(Rule{
		Code:            "UNUSED_METHOD_ARGUMENT",
		DefaultSeverity: "INFO",
		Check:           checkUnusedMethodArgument,
	})
}

type localDecl struct {
	name   string
	offset int
}

type unusedFuncAnalysis struct {
	funcName    string
	params      []localDecl
	locals      []localDecl
	localSeen   map[string]bool
	used        map[string]bool
	usesAllArgs bool
	paramNames  map[string]bool
}

func checkUnusedLocalVariable(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.unusedAnalyses() {
		for _, d := range a.locals {
			low := strings.ToLower(d.name)
			if low == "local" || a.used[low] {
				continue
			}

			out = append(out, issueInFunc(fc, d.offset, fmt.Sprintf("Local variable %s is not used in function %s. Consider removing it.", d.name, a.funcName), d.name, a.funcName))
		}
	}

	return out
}

func checkUnusedMethodArgument(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.unusedAnalyses() {
		if a.usesAllArgs {
			continue
		}

		for _, p := range a.params {
			low := strings.ToLower(p.name)
			if a.used[low] {
				continue
			}

			out = append(out, issueInFunc(fc, p.offset, fmt.Sprintf("Argument %s is not used in function %s. Consider removing it.", p.name, a.funcName), p.name, a.funcName))
		}
	}

	return out
}

func analyzeFuncUnused(src string, sp *parser.FunctionSpan) *unusedFuncAnalysis {
	a := &unusedFuncAnalysis{
		funcName:   sp.Name,
		localSeen:  map[string]bool{},
		used:       map[string]bool{},
		paramNames: map[string]bool{},
	}

	body := sp.BodyText(src)

	// 1. Collect script parameters or <cfargument> tags.
	if sp.Params >= 0 {
		if pText := sp.ParamsText(src); pText != "" {
			collectUnusedScriptParams(pText, sp.Params+1, a)
		}
	}

	for _, tag := range parser.CFTags(body, "cfargument") {
		if tag.Close {
			continue
		}

		if name, ok := tag.Attr("name"); ok {
			trimmed := strings.TrimSpace(name)
			if trimmed != "" {
				a.params = append(a.params, localDecl{name: trimmed, offset: sp.Body + tag.Start})
				a.paramNames[strings.ToLower(trimmed)] = true
			}
		}
	}

	if body == "" {
		return a
	}

	// 2. Collect tag attributes that reference or bind local/arg names.
	scanTagsForUnused(body, sp.Body, a)

	// 3. Scan script regions (or <cfset>/<cfif>/<cfelseif>/<cfreturn> in tag functions).
	if sp.Params >= 0 {
		scanScriptForUnused(body, sp.Body, a)
	} else {
		for _, r := range parser.ClassifyRegions(body) {
			if r.Kind == parser.RegionScript {
				scanScriptForUnused(r.Text, sp.Body+r.Offset, a)
			} else {
				scanHashInterpolationsForUnused(r.Text, a)
			}
		}

		for _, tag := range parser.CFTags(body, "cfset", "cfif", "cfelseif", "cfreturn") {
			if tag.Close {
				continue
			}

			txt := tag.Text()
			if len(txt) <= len(tag.Name)+2 {
				continue
			}

			innerStart := len(tag.Name) + 1
			inner := txt[innerStart : len(txt)-1]
			scanScriptForUnused(inner, sp.Body+tag.Start+innerStart, a)
		}
	}

	return a
}

func collectUnusedScriptParams(paramsText string, baseOffset int, a *unusedFuncAnalysis) {
	sc := parser.NewSourceScanner(paramsText)

	var (
		toks  []parser.Token
		depth = 0
	)

	flush := func() {
		if len(toks) == 0 {
			return
		}

		var nameTok parser.Token

		for _, t := range toks {
			if t.Kind == parser.TokEquals {
				break
			}

			if t.Kind == parser.TokIdent {
				nameTok = t
			}
		}

		if nameTok.Kind == parser.TokIdent {
			a.params = append(a.params, localDecl{name: nameTok.Value, offset: baseOffset + nameTok.Offset})
			a.paramNames[strings.ToLower(nameTok.Value)] = true
		}

		toks = toks[:0]
	}

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			flush()

			break
		}

		switch tok.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			if depth > 0 {
				depth--
			}
		case parser.TokComma:
			if depth == 0 {
				flush()

				continue
			}
		default:
		}

		toks = append(toks, tok)
	}
}

func (a *unusedFuncAnalysis) addLocal(name string, offset int) {
	low := strings.ToLower(name)
	if low == "" || low == "local" || a.localSeen[low] {
		return
	}

	a.localSeen[low] = true
	a.locals = append(a.locals, localDecl{name: name, offset: offset})
}

func scanTagsForUnused(body string, baseOffset int, a *unusedFuncAnalysis) {
	// Tag output attributes (e.g. `<cfloop index="iIdx">`, `<cfquery name="qWork">`):
	// mark matching pre-declared var as used; if `local.foo`, record as local decl.
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

			trimmed := strings.TrimSpace(val)
			if trimmed == "" || strings.Contains(trimmed, "#") {
				continue
			}

			if prefix, after, found := strings.Cut(trimmed, "."); found {
				if strings.EqualFold(prefix, "local") && after != "" {
					a.addLocal(after, baseOffset+tag.Start)
				}
			} else {
				a.used[strings.ToLower(trimmed)] = true
			}
		}
	}
}

func scanScriptForUnused(text string, baseOffset int, a *unusedFuncAnalysis) {
	sc := parser.NewSourceScanner(text)

	var toks []parser.Token

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			break
		}

		toks = append(toks, tok)
	}

	for i := 0; i < len(toks); i++ {
		tok := toks[i]

		// Scan `#...#` interpolations inside string literals.
		if tok.Kind == parser.TokString {
			scanHashInterpolationsForUnused(unquoteCFMLString(tok.Value), a)

			continue
		}

		if tok.Kind != parser.TokIdent {
			continue
		}

		low := strings.ToLower(tok.Value)

		prevKind := parser.TokEOF
		if i > 0 {
			prevKind = toks[i-1].Kind
		}

		// 1. `var name` declaration
		if low == "var" && prevKind != parser.TokDot && i+1 < len(toks) && toks[i+1].Kind == parser.TokIdent {
			nameTok := toks[i+1]
			a.addLocal(nameTok.Value, baseOffset+nameTok.Offset)

			i++

			continue
		}

		// 2. `local.name` (assignment declares if not yet declared; otherwise marks used)
		if low == "local" && prevKind != parser.TokDot && i+2 < len(toks) &&
			toks[i+1].Kind == parser.TokDot && toks[i+2].Kind == parser.TokIdent {
			nameTok := toks[i+2]

			nameLow := strings.ToLower(nameTok.Value)
			if isDirectAssignLHS(toks, i+2) {
				a.addLocal(nameTok.Value, baseOffset+nameTok.Offset)
			} else {
				a.used[nameLow] = true
			}

			i += 2

			continue
		}

		// 3. `arguments.name` or bare `arguments`
		if low == "arguments" && prevKind != parser.TokDot {
			if i+2 < len(toks) && toks[i+1].Kind == parser.TokDot && toks[i+2].Kind == parser.TokIdent {
				a.used[strings.ToLower(toks[i+2].Value)] = true
				i += 2
			} else {
				a.usesAllArgs = true
			}

			continue
		}

		// 4. Script-tag call `cfhttp(..., result="local.httpResponse")`
		if prevKind != parser.TokDot && i+1 < len(toks) && toks[i+1].Kind == parser.TokLParen {
			if attrs := scriptTagOutputAttrs[low]; len(attrs) > 0 {
				scanScriptTagCallForUnused(toks, i+2, baseOffset, attrs, a)
			}
		}

		// 5. Member access after `.` does not count as a standalone variable use.
		if prevKind == parser.TokDot {
			continue
		}

		// 6. Plain reassignment `x = expr` is a write, not a read.
		if isDirectAssignLHS(toks, i) {
			continue
		}

		// 7. Struct literal key `key:` is not a variable read.
		if i+1 < len(toks) && toks[i+1].Kind == parser.TokColon {
			continue
		}

		a.used[low] = true
	}
}

// isDirectAssignLHS reports whether toks[idx] is directly followed by `=` (and not
// `==`, `!=`, `<=`, `>=`, or a subscript/member like `foo[bar] = 1`).
func isDirectAssignLHS(toks []parser.Token, idx int) bool {
	if idx+1 >= len(toks) {
		return false
	}

	if toks[idx+1].Kind != parser.TokEquals {
		return false
	}

	// Ensure it's not `==` (two adjacent TokEquals or TokEqEq).
	if idx+2 < len(toks) && toks[idx+2].Kind == parser.TokEquals && toks[idx+2].Offset == toks[idx+1].Offset+1 {
		return false
	}

	return true
}

func scanScriptTagCallForUnused(toks []parser.Token, startIdx, baseOffset int, attrs map[string]bool, a *unusedFuncAnalysis) {
	depth := 1
	for j := startIdx; j < len(toks) && depth > 0; j++ {
		switch toks[j].Kind {
		case parser.TokLParen:
			depth++
		case parser.TokRParen:
			depth--
		case parser.TokIdent:
			if depth == 1 && attrs[strings.ToLower(toks[j].Value)] &&
				j+2 < len(toks) && toks[j+1].Kind == parser.TokEquals && toks[j+2].Kind == parser.TokString {
				val := strings.TrimSpace(unquoteCFMLString(toks[j+2].Value))
				if prefix, after, found := strings.Cut(val, "."); found && strings.EqualFold(prefix, "local") && after != "" {
					a.addLocal(after, baseOffset+toks[j].Offset)
				}
			}
		default:
		}
	}
}

func scanHashInterpolationsForUnused(text string, a *unusedFuncAnalysis) {
	for i := 0; i < len(text); {
		open := strings.IndexByte(text[i:], '#')
		if open < 0 {
			return
		}

		open += i
		if open+1 < len(text) && text[open+1] == '#' {
			i = open + 2

			continue
		}

		closeRel := strings.IndexByte(text[open+1:], '#')
		if closeRel < 0 {
			return
		}

		expr := text[open+1 : open+1+closeRel]
		i = open + 1 + closeRel + 1

		sc := parser.NewSourceScanner(expr)

		var prev2, prev parser.Token

		for {
			tok := sc.NextSkipComments()
			if tok.Kind == parser.TokEOF {
				break
			}

			if tok.Kind != parser.TokIdent {
				prev2 = prev
				prev = tok

				continue
			}

			low := strings.ToLower(tok.Value)
			isAllowedDot := prev.Kind == parser.TokDot &&
				(strings.EqualFold(prev2.Value, "local") || strings.EqualFold(prev2.Value, "arguments") || strings.EqualFold(prev2.Value, "variables"))

			if prev.Kind != parser.TokDot || isAllowedDot {
				if low == "arguments" {
					prev2, prev = consumeHashArgumentsUse(tok, sc, a)

					continue
				}

				a.used[low] = true
			}

			prev2 = prev
			prev = tok
		}
	}
}

func consumeHashArgumentsUse(argTok parser.Token, sc *parser.Scanner, a *unusedFuncAnalysis) (prev2, prev parser.Token) {
	next := sc.NextSkipComments()
	if next.Kind == parser.TokDot {
		member := sc.NextSkipComments()
		if member.Kind == parser.TokIdent {
			a.used[strings.ToLower(member.Value)] = true
		}

		return next, member
	}

	a.usesAllArgs = true

	return argTok, next
}
