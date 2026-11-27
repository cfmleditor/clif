package lint

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerMetricsAndStructRules(DefaultRegistry)
}

func registerMetricsAndStructRules(reg *Registry) {
	reg.Register(Rule{
		Code:            "EXCESSIVE_FUNCTIONS",
		DefaultSeverity: "WARNING",
		Check:           checkExcessiveFunctions,
	})
	reg.Register(Rule{
		Code:            "FUNCTION_TOO_COMPLEX",
		DefaultSeverity: "WARNING",
		Check:           checkFunctionTooComplex,
	})
	reg.Register(Rule{
		Code:            "LOCAL_LITERAL_VALUE_USED_TOO_OFTEN",
		DefaultSeverity: "WARNING",
		Check:           checkLocalLiteralUsedTooOften,
	})
	reg.Register(Rule{
		Code:            "GLOBAL_LITERAL_VALUE_USED_TOO_OFTEN",
		DefaultSeverity: "WARNING",
		Check:           checkGlobalLiteralUsedTooOften,
	})
	reg.Register(Rule{
		Code:            "UNQUOTED_STRUCT_KEY",
		DefaultSeverity: "WARNING",
		Check:           checkUnquotedStructKey,
	})
	reg.Register(Rule{
		Code:            "STRUCT_ARRAY_NOTATION",
		DefaultSeverity: "WARNING",
		Check:           checkStructArrayNotation,
	})
	reg.Register(Rule{
		Code:            "PARSE_NOTHING",
		DefaultSeverity: "INFO",
		Check:           func(_ *FileContext) []cflint.Issue { return nil },
	})
}

func rcIntParam(fc *FileContext, key string, fallback int) int {
	if fc.RC == nil {
		return fallback
	}

	raw := strings.TrimSpace(fc.RC.ParamString(key))
	if raw == "" {
		return fallback
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}

func checkExcessiveFunctions(fc *FileContext) []cflint.Issue {
	maxFuncs := rcIntParam(fc, "ExcessiveFunctionsChecker.maximum", 10)
	count := len(fc.ParseResult().FunctionSpans())

	// Also count self-closing `<cffunction ... />` tags which FunctionSpans may skip.
	tagCount := 0

	for _, tag := range parser.CFTags(fc.Text(), "cffunction") {
		if !tag.Close {
			tagCount++
		}
	}

	if tagCount > count {
		count = tagCount
	}

	if count > maxFuncs {
		return []cflint.Issue{fc.IssueAt(0, fmt.Sprintf("Component has too many functions (%d). Should be less than %d.", count, maxFuncs), "")}
	}

	return nil
}

func checkFunctionTooComplex(fc *FileContext) []cflint.Issue {
	maxComplexity := rcIntParam(fc, "SimpleComplexityChecker.maximum", 10)
	src := fc.Text()
	spans := fc.ParseResult().FunctionSpans()

	var out []cflint.Issue

	for i := range spans {
		sp := &spans[i]

		body := sp.BodyText(src)
		if body == "" {
			continue
		}

		c := computeFunctionComplexity(body, sp.Params >= 0)
		if c > maxComplexity {
			out = append(out, issueInFunc(fc, sp.Start, fmt.Sprintf("Function %s is too complex (%d). Consider breaking it down.", sp.Name, c), "", sp.Name))
		}
	}

	return out
}

func computeFunctionComplexity(body string, isScript bool) int {
	complexity := 1

	countScript := func(text string) {
		sc := parser.NewSourceScanner(text)

		var prev parser.Token

		for {
			tok := sc.NextSkipComments()
			if tok.Kind == parser.TokEOF {
				break
			}

			if tok.Kind == parser.TokIdent && prev.Kind != parser.TokDot {
				switch strings.ToLower(tok.Value) {
				case "if", "elseif", "case", "for", "while", "catch", "and", "or":
					complexity++
				}
			} else if tok.Kind == parser.TokOther && (tok.Value == "?" || tok.Value == "&&" || tok.Value == "||") {
				complexity++
			}

			prev = tok
		}
	}

	if isScript {
		countScript(body)

		return complexity
	}

	for _, r := range parser.ClassifyRegions(body) {
		if r.Kind == parser.RegionScript {
			countScript(r.Text)
		}
	}

	for _, tag := range parser.CFTags(body, "cfif", "cfelseif", "cfcase", "cfloop", "cfcatch") {
		if !tag.Close {
			complexity++
		}
	}

	return complexity
}

var defaultIgnoredLiterals = map[string]bool{
	"":           true,
	"0":          true,
	"1":          true,
	"true":       true,
	"false":      true,
	"yes":        true,
	"no":         true,
	",":          true,
	".":          true,
	" ":          true,
	"asc":        true,
	"desc":       true,
	"numeric":    true,
	"text":       true,
	"textnocase": true,
}

func checkLocalLiteralUsedTooOften(fc *FileContext) []cflint.Issue {
	maxOcc := rcIntParam(fc, "LiteralChecker.maxOccurancesOfLocal", 3)
	src := fc.Text()
	spans := fc.ParseResult().FunctionSpans()

	var out []cflint.Issue

	for i := range spans {
		sp := &spans[i]

		body := sp.BodyText(src)
		if body == "" {
			continue
		}

		counts := map[string]int{}
		if sp.Params >= 0 {
			out = append(out, scanLiteralsText(fc, body, sp.Body, maxOcc, counts, sp.Name)...)
		} else {
			for _, r := range parser.ClassifyRegions(body) {
				if r.Kind == parser.RegionScript {
					out = append(out, scanLiteralsText(fc, r.Text, sp.Body+r.Offset, maxOcc, counts, sp.Name)...)
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
				out = append(out, scanLiteralsText(fc, txt[innerStart:len(txt)-1], sp.Body+tag.Start+innerStart, maxOcc, counts, sp.Name)...)
			}
		}
	}

	return out
}

func checkGlobalLiteralUsedTooOften(fc *FileContext) []cflint.Issue {
	maxOcc := rcIntParam(fc, "LiteralChecker.maxOccurancesOfGlobal", 3)
	src := fc.Text()
	counts := map[string]int{}

	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind == parser.RegionScript {
			out = append(out, scanLiteralsText(fc, r.Text, r.Offset, maxOcc, counts, "")...)
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
		out = append(out, scanLiteralsText(fc, txt[innerStart:len(txt)-1], tag.Start+innerStart, maxOcc, counts, "")...)
	}

	return out
}

func scanLiteralsText(fc *FileContext, text string, baseOffset, maxOcc int, counts map[string]int, funcName string) []cflint.Issue {
	sc := parser.NewSourceScanner(text)

	var out []cflint.Issue

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			break
		}

		if tok.Kind != parser.TokString && tok.Kind != parser.TokNumber {
			continue
		}

		val := tok.Value
		if tok.Kind == parser.TokString {
			val = unquoteCFMLString(val)
		}

		low := strings.ToLower(strings.TrimSpace(val))
		if defaultIgnoredLiterals[low] {
			continue
		}

		counts[val]++
		if counts[val] > maxOcc {
			msg := fmt.Sprintf("Literal %s occurs several times in one or more files. Consider giving it a name and not hard coding values.", val)
			if funcName != "" {
				out = append(out, issueInFunc(fc, baseOffset+tok.Offset, msg, val, funcName))
			} else {
				out = append(out, fc.IssueAt(baseOffset+tok.Offset, msg, val))
			}
		}
	}

	return out
}

func checkUnquotedStructKey(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind == parser.RegionScript {
			out = append(out, scanUnquotedStructKeys(fc, r.Text, r.Offset)...)
		}
	}

	for _, tag := range parser.CFTags(fc.Text(), "cfset", "cfreturn") {
		if tag.Close {
			continue
		}

		txt := tag.Text()
		if len(txt) <= len(tag.Name)+2 {
			continue
		}

		innerStart := len(tag.Name) + 1
		out = append(out, scanUnquotedStructKeys(fc, txt[innerStart:len(txt)-1], tag.Start+innerStart)...)
	}

	return out
}

func scanUnquotedStructKeys(fc *FileContext, text string, baseOffset int) []cflint.Issue {
	sc := parser.NewSourceScanner(text)

	var toks []parser.Token

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			break
		}

		toks = append(toks, tok)
	}

	// Track whether each `{` brace level is a struct literal (true) or a code block (false).
	var braceStack []bool

	var out []cflint.Issue

	for i := range toks {
		tok := toks[i]

		switch tok.Kind {
		case parser.TokLBrace:
			isStruct := i > 0 && isStructLiteralPredecessor(toks[i-1])
			braceStack = append(braceStack, isStruct)
		case parser.TokRBrace:
			if len(braceStack) > 0 {
				braceStack = braceStack[:len(braceStack)-1]
			}
		case parser.TokIdent:
			if len(braceStack) > 0 && braceStack[len(braceStack)-1] &&
				i+1 < len(toks) && (toks[i+1].Kind == parser.TokEquals || toks[i+1].Kind == parser.TokColon) &&
				(i == 0 || toks[i-1].Kind == parser.TokLBrace || toks[i-1].Kind == parser.TokComma) {
				out = append(out, fc.IssueAt(baseOffset+tok.Offset, fmt.Sprintf("Unquoted struct key %s is not case-sensitive. Quoting it is recommended.", tok.Value), tok.Value))
			}
		default:
		}
	}

	return out
}

func isStructLiteralPredecessor(prev parser.Token) bool {
	switch prev.Kind {
	case parser.TokEquals, parser.TokColon, parser.TokLParen, parser.TokComma, parser.TokLBracket:
		return true
	case parser.TokIdent:
		return strings.EqualFold(prev.Value, "return")
	case parser.TokOther:
		return prev.Value == "?"
	default:
		return false
	}
}

func checkStructArrayNotation(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	spans := fc.ParseResult().FunctionSpans()

	var out []cflint.Issue

	for i := range spans {
		sp := &spans[i]

		body := sp.BodyText(src)
		if body == "" {
			continue
		}

		if sp.Params >= 0 {
			out = append(out, scanStructDotAssign(fc, body, sp.Body, sp.Name)...)
		} else {
			for _, r := range parser.ClassifyRegions(body) {
				if r.Kind == parser.RegionScript {
					out = append(out, scanStructDotAssign(fc, r.Text, sp.Body+r.Offset, sp.Name)...)
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
				out = append(out, scanStructDotAssign(fc, txt[innerStart:len(txt)-1], sp.Body+tag.Start+innerStart, sp.Name)...)
			}
		}
	}

	return out
}

func scanStructDotAssign(fc *FileContext, text string, baseOffset int, funcName string) []cflint.Issue {
	sc := parser.NewSourceScanner(text)

	var toks []parser.Token

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			break
		}

		toks = append(toks, tok)
	}

	var out []cflint.Issue

	for i := 2; i < len(toks); i++ {
		if toks[i].Kind != parser.TokIdent || toks[i-1].Kind != parser.TokDot {
			continue
		}

		if !isDirectAssignLHS(toks, i) {
			continue
		}

		// Skip direct scope assignments like `variables.x = 1`, `local.x = 1`, `this.x = 1`.
		if toks[i-2].Kind == parser.TokIdent && cfmlScopes[strings.ToLower(toks[i-2].Value)] &&
			(i < 3 || toks[i-3].Kind != parser.TokDot) {
			continue
		}

		out = append(out, issueInFunc(fc, baseOffset+toks[i].Offset, fmt.Sprintf("Unquoted struct key %s is not case-sensitive. Use array notation instead.", toks[i].Value), toks[i].Value, funcName))
	}

	return out
}
