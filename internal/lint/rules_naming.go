package lint

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerNamingRules(DefaultRegistry)
}

func registerNamingRules(reg *Registry) {
	// Component naming rules.
	reg.Register(Rule{Code: "COMPONENT_INVALID_NAME", DefaultSeverity: "INFO", Check: checkComponentInvalidName})
	reg.Register(Rule{Code: "COMPONENT_ALLCAPS_NAME", DefaultSeverity: "INFO", Check: checkComponentAllCapsName})
	reg.Register(Rule{Code: "COMPONENT_TOO_SHORT", DefaultSeverity: "INFO", Check: checkComponentTooShort})
	reg.Register(Rule{Code: "COMPONENT_TOO_LONG", DefaultSeverity: "INFO", Check: checkComponentTooLong})
	reg.Register(Rule{Code: "COMPONENT_TOO_WORDY", DefaultSeverity: "INFO", Check: checkComponentTooWordy})
	reg.Register(Rule{Code: "COMPONENT_IS_TEMPORARY", DefaultSeverity: "INFO", Check: checkComponentIsTemporary})
	reg.Register(Rule{Code: "COMPONENT_HAS_PREFIX_OR_POSTFIX", DefaultSeverity: "INFO", Check: checkComponentHasPrefixOrPostfix})

	// Method naming rules.
	reg.Register(Rule{Code: "METHOD_ALLCAPS_NAME", DefaultSeverity: "INFO", Check: checkMethodAllCapsName})
	reg.Register(Rule{Code: "METHOD_TOO_LONG", DefaultSeverity: "INFO", Check: checkMethodTooLong})

	// Argument naming rules.
	reg.Register(Rule{Code: "ARGUMENT_HAS_PREFIX_OR_POSTFIX", DefaultSeverity: "INFO", Check: checkArgHasPrefixOrPostfix})
	reg.Register(Rule{Code: "ARGUMENT_IS_TEMPORARY", DefaultSeverity: "INFO", Check: checkArgIsTemporary})

	// Scope & variable naming rules.
	reg.Register(Rule{Code: "SCOPE_ALLCAPS_NAME", DefaultSeverity: "INFO", Check: checkScopeAllCapsName})
	reg.Register(Rule{Code: "VAR_INVALID_NAME", DefaultSeverity: "INFO", Check: checkVarInvalidName})
	reg.Register(Rule{Code: "VAR_ALLCAPS_NAME", DefaultSeverity: "INFO", Check: checkVarAllCapsName})
	reg.Register(Rule{Code: "VAR_TOO_SHORT", DefaultSeverity: "INFO", Check: checkVarTooShort})
	reg.Register(Rule{Code: "VAR_TOO_LONG", DefaultSeverity: "INFO", Check: checkVarTooLong})
	reg.Register(Rule{Code: "VAR_TOO_WORDY", DefaultSeverity: "INFO", Check: checkVarTooWordy})
	reg.Register(Rule{Code: "VAR_IS_TEMPORARY", DefaultSeverity: "INFO", Check: checkVarIsTemporary})
	reg.Register(Rule{Code: "VAR_HAS_PREFIX_OR_POSTFIX", DefaultSeverity: "INFO", Check: checkVarHasPrefixOrPostfix})
}

// --- Pure-Go naming classifiers (no regex) ---

func isAllCapsName(s string) bool {
	hasLetter := false

	for i := range len(s) {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			return false
		}

		if c >= 'A' && c <= 'Z' {
			hasLetter = true
		}
	}

	return hasLetter
}

func isValidPascalCase(s string) bool {
	if s == "" || s[0] < 'A' || s[0] > 'Z' {
		return false
	}

	for i := 1; i < len(s); i++ {
		c := s[i]

		isAlphaNum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !isAlphaNum {
			return false
		}
	}

	return true
}

func isValidCamelVarName(s string) bool {
	if s == "" {
		return false
	}

	// Standard CFML variable names start with a lowercase letter (all-caps is
	// checked separately by VAR_ALLCAPS_NAME).
	if s[0] >= 'A' && s[0] <= 'Z' {
		return isAllCapsName(s)
	}

	return (s[0] >= 'a' && s[0] <= 'z') || s[0] == '_'
}

func countCamelWords(s string) int {
	if s == "" {
		return 0
	}

	words := 1

	for i := 1; i < len(s); i++ {
		prev := s[i-1]
		cur := s[i]

		if cur == '_' {
			if i+1 < len(s) && s[i+1] != '_' {
				words++
			}

			continue
		}

		if prev >= 'a' && prev <= 'z' && cur >= 'A' && cur <= 'Z' {
			words++
		}
	}

	return words
}

var temporaryWords = []string{
	"temp", "tmp", "var", "func", "obj", "object", "bool", "struct",
	"string", "array", "comp", "my",
}

func isTemporaryName(s string) bool {
	low := strings.ToLower(s)
	for _, w := range temporaryWords {
		if low == w {
			return true
		}

		// Prefix match followed by uppercase or `_` (e.g. `tempStruct`).
		if len(s) > len(w) && strings.HasPrefix(low, w) {
			next := s[len(w)]
			if (next >= 'A' && next <= 'Z') || next == '_' {
				return true
			}
		}

		// Postfix match starting with uppercase or `_` (e.g. `dateObj`).
		if len(s) > len(w) && strings.HasSuffix(low, w) {
			pos := len(s) - len(w)
			if (s[pos] >= 'A' && s[pos] <= 'Z') || (pos > 0 && s[pos-1] == '_') {
				return true
			}
		}
	}

	return false
}

var hungarianPrefixes = []string{
	"s", "st", "str", "o", "obj", "b", "bool", "a", "arr",
	"q", "qry", "i", "int", "n", "num", "d", "dt", "fn", "v", "l", "g",
}

var hungarianPostfixes = []string{
	"str", "string", "obj", "object", "arr", "array",
	"qry", "bool", "int", "num", "list", "date",
}

func hasPrefixOrPostfix(s string) bool {
	low := strings.ToLower(s)
	if low == "thistag" || low == "thattag" {
		return false
	}

	for _, p := range hungarianPrefixes {
		if len(s) > len(p) && strings.HasPrefix(low, p) {
			next := s[len(p)]
			if (next >= 'A' && next <= 'Z') || next == '_' {
				return true
			}
		}
	}

	for _, p := range hungarianPostfixes {
		if len(s) > len(p) && strings.HasSuffix(low, p) {
			pos := len(s) - len(p)
			if s[pos] >= 'A' && s[pos] <= 'Z' && pos > 0 && s[pos-1] >= 'a' && s[pos-1] <= 'z' {
				return true
			}

			if pos > 0 && s[pos-1] == '_' {
				return true
			}
		}
	}

	return false
}

// --- Component naming ---

func componentNameAndOffset(fc *FileContext) (string, int, bool) {
	if !strings.EqualFold(filepath.Ext(fc.Path), ".cfc") {
		return "", 0, false
	}

	compName := strings.TrimSuffix(filepath.Base(fc.Path), filepath.Ext(fc.Path))
	src := fc.Text()

	for _, tag := range parser.CFTags(src, "cfcomponent") {
		if !tag.Close {
			return compName, tag.Start, true
		}
	}

	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript {
			continue
		}

		sc := parser.NewSourceScanner(r.Text)
		for {
			tok := sc.NextSkipComments()
			if tok.Kind == parser.TokEOF {
				break
			}

			if tok.Kind == parser.TokIdent && strings.EqualFold(tok.Value, "component") {
				return compName, r.Offset + tok.Offset, true
			}
		}
	}

	return compName, 0, true
}

func checkComponentInvalidName(fc *FileContext) []cflint.Issue {
	name, off, ok := componentNameAndOffset(fc)
	if !ok || isValidPascalCase(name) || isAllCapsName(name) {
		return nil
	}

	return []cflint.Issue{fc.IssueAt(off, fmt.Sprintf("Component name %s is not a valid name. Please use PascalCase.", name), "")}
}

func checkComponentAllCapsName(fc *FileContext) []cflint.Issue {
	name, off, ok := componentNameAndOffset(fc)
	if !ok || !isAllCapsName(name) {
		return nil
	}

	return []cflint.Issue{fc.IssueAt(off, fmt.Sprintf("Component name %s should not be all caps.", name), "")}
}

func checkComponentTooShort(fc *FileContext) []cflint.Issue {
	name, off, ok := componentNameAndOffset(fc)
	minLen := rcIntParam(fc, "ComponentNameChecker.minLength", 3)

	if !ok || len(name) >= minLen {
		return nil
	}

	return []cflint.Issue{fc.IssueAt(off, fmt.Sprintf("Component name %s is shorter than %d characters.", name, minLen), "")}
}

func checkComponentTooLong(fc *FileContext) []cflint.Issue {
	name, off, ok := componentNameAndOffset(fc)
	maxLen := rcIntParam(fc, "ComponentNameChecker.maxLength", 20)

	if !ok || len(name) <= maxLen {
		return nil
	}

	return []cflint.Issue{fc.IssueAt(off, fmt.Sprintf("Component name %s is longer than %d characters.", name, maxLen), "")}
}

func checkComponentTooWordy(fc *FileContext) []cflint.Issue {
	name, off, ok := componentNameAndOffset(fc)
	maxWords := rcIntParam(fc, "ComponentNameChecker.maxWords", 3)

	if !ok || countCamelWords(name) <= maxWords {
		return nil
	}

	return []cflint.Issue{fc.IssueAt(off, fmt.Sprintf("Component name %s is too wordy (more than %d words).", name, maxWords), "")}
}

func checkComponentIsTemporary(fc *FileContext) []cflint.Issue {
	name, off, ok := componentNameAndOffset(fc)
	if !ok || !isTemporaryName(name) {
		return nil
	}

	return []cflint.Issue{fc.IssueAt(off, fmt.Sprintf("Component name %s could be named better.", name), "")}
}

func checkComponentHasPrefixOrPostfix(fc *FileContext) []cflint.Issue {
	name, off, ok := componentNameAndOffset(fc)
	if !ok || !hasPrefixOrPostfix(name) {
		return nil
	}

	return []cflint.Issue{fc.IssueAt(off, fmt.Sprintf("Component name %s has prefix or postfix.", name), "")}
}

// --- Method naming ---

func checkMethodAllCapsName(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	spans := fc.ParseResult().FunctionSpans()
	for i := range spans {
		sp := &spans[i]
		if sp.Name != "" && isAllCapsName(sp.Name) {
			out = append(out, issueInFunc(fc, sp.Start, fmt.Sprintf("Method name %s should not be all caps.", sp.Name), "", sp.Name))
		}
	}

	return out
}

func checkMethodTooLong(fc *FileContext) []cflint.Issue {
	maxLen := rcIntParam(fc, "MethodNameChecker.maxLength", 25)

	var out []cflint.Issue

	spans := fc.ParseResult().FunctionSpans()
	for i := range spans {
		sp := &spans[i]
		if len(sp.Name) > maxLen {
			out = append(out, issueInFunc(fc, sp.Start, fmt.Sprintf("Method name %s is longer than %d characters.", sp.Name, maxLen), sp.Name, sp.Name))
		}
	}

	return out
}

// --- Argument naming ---

type namedArg struct {
	name     string
	offset   int
	funcName string
}

func collectAllFunctionArgs(fc *FileContext) []namedArg {
	src := fc.Text()
	spans := fc.ParseResult().FunctionSpans()

	var out []namedArg

	for i := range spans {
		sp := &spans[i]
		if sp.Params >= 0 {
			if pText := sp.ParamsText(src); pText != "" {
				a := &unusedFuncAnalysis{localSeen: map[string]bool{}, used: map[string]bool{}, paramNames: map[string]bool{}}
				collectUnusedScriptParams(pText, sp.Params+1, a)

				for _, p := range a.params {
					out = append(out, namedArg{name: p.name, offset: p.offset, funcName: sp.Name})
				}
			}
		}

		body := sp.BodyText(src)
		for _, tag := range parser.CFTags(body, "cfargument") {
			if tag.Close {
				continue
			}

			if name, ok := tag.Attr("name"); ok && strings.TrimSpace(name) != "" {
				out = append(out, namedArg{name: strings.TrimSpace(name), offset: sp.Body + tag.Start, funcName: sp.Name})
			}
		}
	}

	return out
}

func checkArgHasPrefixOrPostfix(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.functionArgs() {
		if hasPrefixOrPostfix(a.name) {
			out = append(out, issueInFunc(fc, a.offset, fmt.Sprintf("Argument %s has prefix or postfix.", a.name), a.name, a.funcName))
		}
	}

	return out
}

func checkArgIsTemporary(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, a := range fc.functionArgs() {
		if isTemporaryName(a.name) {
			out = append(out, issueInFunc(fc, a.offset, fmt.Sprintf("Argument %s could be named better.", a.name), a.name, a.funcName))
		}
	}

	return out
}

// --- Scope & variable naming ---

var defaultIgnoreUpperCaseScopes = map[string]bool{
	"cgi": true, "url": true, "request": true, "form": true,
	"cookie": true, "session": true, "application": true, "server": true, "client": true,
}

func checkScopeAllCapsName(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	reported := map[string]bool{}

	scan := func(text string, baseOffset int) {
		sc := parser.NewSourceScanner(text)

		var prev parser.Token

		cur := sc.NextSkipComments()
		for cur.Kind != parser.TokEOF {
			next := sc.NextSkipComments()

			if cur.Kind == parser.TokIdent && prev.Kind != parser.TokDot && next.Kind == parser.TokDot {
				low := strings.ToLower(cur.Value)
				if cfmlScopes[low] && !defaultIgnoreUpperCaseScopes[low] && isAllCapsName(cur.Value) && !reported[low] {
					reported[low] = true

					out = append(out, fc.IssueAt(baseOffset+cur.Offset, fmt.Sprintf("Scope %s should not be all caps.", cur.Value), cur.Value))
				}
			}

			prev = cur
			cur = next
		}
	}

	for _, r := range fc.Regions() {
		if r.Kind == parser.RegionScript {
			scan(r.Text, r.Offset)
		}
	}

	for _, tag := range parser.CFTags(fc.Text(), "cfset", "cfif", "cfelseif", "cfreturn") {
		if tag.Close {
			continue
		}

		txt := tag.Text()
		if len(txt) <= len(tag.Name)+2 {
			continue
		}

		innerStart := len(tag.Name) + 1
		scan(txt[innerStart:len(txt)-1], tag.Start+innerStart)
	}

	return out
}

type varNameSite struct {
	name          string
	offset        int
	ignoreAllCaps bool
}

// collectVarNameSites gathers variable declaration, assignment, and query-name
// sites checked by VariableNameChecker.
func collectVarNameSites(fc *FileContext) []varNameSite {
	src := fc.Text()
	spans := fc.ParseResult().FunctionSpans()

	var sites []varNameSite

	// 1. `<cfquery name="...">`
	for _, tag := range parser.CFTags(src, "cfquery") {
		if tag.Close {
			continue
		}

		raw, ok := tag.Attr("name")
		if !ok || strings.TrimSpace(raw) == "" {
			continue
		}

		trimmed := strings.TrimSpace(raw)
		if strings.Contains(trimmed, "#") {
			// Check identifiers inside `#...#` in dynamic query names (e.g. `q_count_total_#q_get_sites.site_id#`).
			for _, id := range extractHashRootIdents(trimmed) {
				sites = append(sites, varNameSite{name: id, offset: tag.Start})
			}

			continue
		}

		if prefix, after, found := strings.Cut(trimmed, "."); found {
			sites = append(sites, varNameSite{name: after, offset: tag.Start, ignoreAllCaps: strings.EqualFold(prefix, "variables") || strings.EqualFold(prefix, "this")})
		} else {
			sites = append(sites, varNameSite{name: trimmed, offset: tag.Start})
		}
	}

	// 2. Script regions & `<cfset>` / `<cfif>` / `<cfreturn>` expressions.
	scan := func(text string, baseOffset int) {
		sites = append(sites, scanScriptVarNameSites(text, baseOffset, spans)...)
	}

	for _, r := range fc.Regions() {
		if r.Kind == parser.RegionScript {
			scan(r.Text, r.Offset)
		}
	}

	for _, tag := range parser.CFTags(src, "cfset", "cfif", "cfelseif", "cfreturn") {
		if tag.Close {
			continue
		}

		txt := tag.Text()
		if len(txt) <= len(tag.Name)+2 {
			continue
		}

		innerStart := len(tag.Name) + 1
		scan(txt[innerStart:len(txt)-1], tag.Start+innerStart)
	}

	return sites
}

func extractHashRootIdents(s string) []string {
	var out []string

	for i := 0; i < len(s); {
		open := strings.IndexByte(s[i:], '#')
		if open < 0 {
			break
		}

		open += i

		closeRel := strings.IndexByte(s[open+1:], '#')
		if closeRel < 0 {
			break
		}

		expr := s[open+1 : open+1+closeRel]
		i = open + 1 + closeRel + 1

		sc := parser.NewSourceScanner(expr)

		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokIdent && !cfmlScopes[strings.ToLower(tok.Value)] {
			out = append(out, tok.Value)
		}
	}

	return out
}

func scanScriptVarNameSites(text string, baseOffset int, spans []parser.FunctionSpan) []varNameSite {
	sc := parser.NewSourceScanner(text)

	var toks []parser.Token

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			break
		}

		toks = append(toks, tok)
	}

	var (
		sites      []varNameSite
		braceStack []bool
	)

	for i := 0; i < len(toks); i++ {
		tok := toks[i]

		switch tok.Kind {
		case parser.TokLBrace:
			isStruct := i > 0 && isStructLiteralPredecessor(toks[i-1])
			braceStack = append(braceStack, isStruct)

			continue
		case parser.TokRBrace:
			if len(braceStack) > 0 {
				braceStack = braceStack[:len(braceStack)-1]
			}

			continue
		default:
		}

		if tok.Kind != parser.TokIdent {
			continue
		}

		low := strings.ToLower(tok.Value)

		prevKind := parser.TokEOF
		if i > 0 {
			prevKind = toks[i-1].Kind
		}

		// 1. `var name`
		if low == "var" && prevKind != parser.TokDot && i+1 < len(toks) && toks[i+1].Kind == parser.TokIdent {
			sites = append(sites, varNameSite{name: toks[i+1].Value, offset: baseOffset + toks[i+1].Offset})
			i++

			continue
		}

		// 2. `local.name` / `variables.name` / `this.name`
		if prevKind != parser.TokDot && cfmlScopes[low] && i+2 < len(toks) &&
			toks[i+1].Kind == parser.TokDot && toks[i+2].Kind == parser.TokIdent {
			ignoreCaps := low == "this" || low == "variables"
			sites = append(sites, varNameSite{name: toks[i+2].Value, offset: baseOffset + toks[i+2].Offset, ignoreAllCaps: ignoreCaps})
			i += 2

			continue
		}

		// 3. Struct literal key `key = val` (inside `{ ... }`)
		if len(braceStack) > 0 && braceStack[len(braceStack)-1] &&
			i+1 < len(toks) && (toks[i+1].Kind == parser.TokEquals || toks[i+1].Kind == parser.TokColon) &&
			(i == 0 || toks[i-1].Kind == parser.TokLBrace || toks[i-1].Kind == parser.TokComma) {
			if !cfmlKeywords[low] {
				sites = append(sites, varNameSite{name: tok.Value, offset: baseOffset + tok.Offset})
			}

			continue
		}

		// 4. Bare assignment `name = ...` (outside functions -> ignoreAllCaps, e.g. `VARIABLE = 10` in component body)
		if prevKind != parser.TokDot && isDirectAssignLHS(toks, i) && !cfmlKeywords[low] && !cfmlScopes[low] {
			inFn := offsetInsideAnySpan(baseOffset+tok.Offset, spans)
			sites = append(sites, varNameSite{name: tok.Value, offset: baseOffset + tok.Offset, ignoreAllCaps: !inFn})

			continue
		}

		// 5. Bare read of an uppercase-starting identifier inside a function (e.g. `LocalData` in sample1_228.cfc)
		if prevKind != parser.TokDot && !cfmlKeywords[low] && !cfmlScopes[low] &&
			tok.Value[0] >= 'A' && tok.Value[0] <= 'Z' && !isAllCapsName(tok.Value) &&
			(i+1 >= len(toks) || toks[i+1].Kind != parser.TokLParen) &&
			offsetInsideAnySpan(baseOffset+tok.Offset, spans) {
			sites = append(sites, varNameSite{name: tok.Value, offset: baseOffset + tok.Offset})
		}
	}

	return sites
}

func checkVarNames(fc *FileContext, pred func(site varNameSite) (bool, string)) []cflint.Issue {
	var out []cflint.Issue

	seen := map[string]bool{}

	for _, site := range fc.varNameSites() {
		if site.ignoreAllCaps && isAllCapsName(site.name) {
			continue
		}

		low := strings.ToLower(site.name)
		if seen[low] {
			continue
		}

		if ok, msg := pred(site); ok {
			seen[low] = true

			out = append(out, fc.IssueAt(site.offset, msg, site.name))
		}
	}

	return out
}

func checkVarInvalidName(fc *FileContext) []cflint.Issue {
	return checkVarNames(fc, func(site varNameSite) (bool, string) {
		if !isValidCamelVarName(site.name) {
			return true, fmt.Sprintf("Variable %s is not a valid name. Please use camelCase or underscores.", site.name)
		}

		return false, ""
	})
}

func checkVarAllCapsName(fc *FileContext) []cflint.Issue {
	return checkVarNames(fc, func(site varNameSite) (bool, string) {
		if !site.ignoreAllCaps && isAllCapsName(site.name) {
			return true, fmt.Sprintf("Variable %s should not be all caps.", site.name)
		}

		return false, ""
	})
}

func checkVarTooShort(fc *FileContext) []cflint.Issue {
	minLen := rcIntParam(fc, "VariableNameChecker.minLength", 3)

	return checkVarNames(fc, func(site varNameSite) (bool, string) {
		if len(site.name) < minLen {
			return true, fmt.Sprintf("Variable %s is shorter than %d characters.", site.name, minLen)
		}

		return false, ""
	})
}

func checkVarTooLong(fc *FileContext) []cflint.Issue {
	maxLen := rcIntParam(fc, "VariableNameChecker.maxLength", 20)

	return checkVarNames(fc, func(site varNameSite) (bool, string) {
		if len(site.name) > maxLen {
			return true, fmt.Sprintf("Variable %s is longer than %d characters.", site.name, maxLen)
		}

		return false, ""
	})
}

func checkVarTooWordy(fc *FileContext) []cflint.Issue {
	maxWords := rcIntParam(fc, "VariableNameChecker.maxWords", 4)

	return checkVarNames(fc, func(site varNameSite) (bool, string) {
		if countCamelWords(site.name) > maxWords {
			return true, fmt.Sprintf("Variable %s is too wordy (more than %d words).", site.name, maxWords)
		}

		return false, ""
	})
}

func checkVarIsTemporary(fc *FileContext) []cflint.Issue {
	return checkVarNames(fc, func(site varNameSite) (bool, string) {
		if isTemporaryName(site.name) {
			return true, fmt.Sprintf("Variable %s could be named better.", site.name)
		}

		return false, ""
	})
}

func checkVarHasPrefixOrPostfix(fc *FileContext) []cflint.Issue {
	return checkVarNames(fc, func(site varNameSite) (bool, string) {
		if hasPrefixOrPostfix(site.name) {
			return true, fmt.Sprintf("Variable %s has prefix or postfix.", site.name)
		}

		return false, ""
	})
}
