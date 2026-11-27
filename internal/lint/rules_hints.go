package lint

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerHintAndTypeRules(DefaultRegistry)
}

func registerHintAndTypeRules(reg *Registry) {
	reg.Register(Rule{
		Code:            "FUNCTION_TYPE_MISSING",
		DefaultSeverity: "WARNING",
		Check:           checkFunctionTypeMissing,
	})
	reg.Register(Rule{
		Code:            "FUNCTION_TYPE_ANY",
		DefaultSeverity: "WARNING",
		Check:           checkFunctionTypeAny,
	})
	reg.Register(Rule{
		Code:            "COMPONENT_HINT_MISSING",
		DefaultSeverity: "INFO",
		Check:           checkComponentHintMissing,
	})
	reg.Register(Rule{
		Code:            "FUNCTION_HINT_MISSING",
		DefaultSeverity: "INFO",
		Check:           checkFunctionHintMissing,
	})
	reg.Register(Rule{
		Code:            "ARG_HINT_MISSING",
		DefaultSeverity: "INFO",
		Check:           checkArgHintMissing,
	})
	reg.Register(Rule{
		Code:            "ARG_DEFAULT_MISSING",
		DefaultSeverity: "WARNING",
		Check:           checkArgDefaultMissing,
	})
}

var funcAccessModifiers = map[string]bool{
	"public":   true,
	"private":  true,
	"package":  true,
	"remote":   true,
	"static":   true,
	"final":    true,
	"abstract": true,
	"default":  true,
}

// extractFuncReturnType returns the declared return type of a FunctionSpan,
// or "" if omitted.
func extractFuncReturnType(src string, sp *parser.FunctionSpan) string {
	if sp.Start < 0 || sp.Body <= sp.Start {
		return ""
	}

	header := src[sp.Start:sp.Body]

	// Tag function: `<cffunction ...>`
	if sp.Params < 0 {
		for _, tag := range parser.CFTags(header, "cffunction") {
			if !tag.Close {
				if rt, ok := tag.Attr("returntype"); ok {
					return strings.TrimSpace(rt)
				}

				return ""
			}
		}

		return ""
	}

	// Script function: scan tokens before `function` keyword.
	sc := parser.NewSourceScanner(header)

	var retType strings.Builder

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			break
		}

		if tok.Kind == parser.TokIdent && strings.EqualFold(tok.Value, "function") {
			break
		}

		if tok.Kind == parser.TokIdent && funcAccessModifiers[strings.ToLower(tok.Value)] && retType.Len() == 0 {
			continue
		}

		if tok.Kind == parser.TokIdent || tok.Kind == parser.TokDot {
			retType.WriteString(tok.Value)
		}
	}

	return retType.String()
}

func checkFunctionTypeMissing(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	spans := fc.ParseResult().FunctionSpans()

	var out []cflint.Issue

	for i := range spans {
		sp := &spans[i]
		if sp.Name == "" {
			continue
		}

		if extractFuncReturnType(src, sp) == "" {
			out = append(out, issueInFunc(fc, sp.Start, fmt.Sprintf("Function %s is missing a return type.", sp.Name), sp.Name, sp.Name))
		}
	}

	return out
}

func checkFunctionTypeAny(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	spans := fc.ParseResult().FunctionSpans()

	var out []cflint.Issue

	for i := range spans {
		sp := &spans[i]
		if sp.Name == "" {
			continue
		}

		if strings.EqualFold(extractFuncReturnType(src, sp), "any") {
			out = append(out, issueInFunc(fc, sp.Start, fmt.Sprintf("Function %s return type is any. Please change to be the correct type.", sp.Name), sp.Name, sp.Name))
		}
	}

	return out
}

// hasPrecedingDocComment reports whether a non-empty block comment (`/* ... */`
// or `<!--- ... --->`) immediately precedes offset (ignoring whitespace).
func hasPrecedingDocComment(src string, offset int) bool {
	if offset <= 0 || offset > len(src) {
		return false
	}

	before := strings.TrimRight(src[:offset], " \t\r\n")
	if strings.HasSuffix(before, "*/") {
		openIdx := strings.LastIndex(before, "/*")
		if openIdx < 0 {
			return false
		}

		body := before[openIdx+2 : len(before)-2]

		return hasNonEmptyDocText(body)
	}

	if strings.HasSuffix(before, "--->") {
		openIdx := strings.LastIndex(before, "<!---")
		if openIdx < 0 {
			return false
		}

		body := before[openIdx+5 : len(before)-4]

		return hasNonEmptyDocText(body)
	}

	return false
}

func hasNonEmptyDocText(body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		t := strings.TrimSpace(line)
		t = strings.TrimLeft(t, "*")
		t = strings.TrimSpace(t)

		if t != "" {
			return true
		}
	}

	return false
}

// hasHintAttrInHeader checks whether a script component/function header contains
// a non-empty `hint="..."` or `hint='...'` attribute.
func hasHintAttrInHeader(header string) bool {
	if parser.IndexFold(header, "hint") < 0 {
		return false
	}

	sc := parser.NewSourceScanner(header)
	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			return false
		}

		if tok.Kind == parser.TokIdent && strings.EqualFold(tok.Value, "hint") {
			eq := sc.NextSkipComments()
			if eq.Kind == parser.TokEquals {
				val := sc.NextSkipComments()
				if val.Kind == parser.TokString && strings.TrimSpace(unquoteCFMLString(val.Value)) != "" {
					return true
				}
			}
		}
	}
}

func checkComponentHintMissing(fc *FileContext) []cflint.Issue {
	if !strings.EqualFold(filepath.Ext(fc.Path), ".cfc") {
		return nil
	}

	src := fc.Text()
	compName := strings.TrimSuffix(filepath.Base(fc.Path), filepath.Ext(fc.Path))

	// Tag component: `<cfcomponent ...>`
	for _, tag := range parser.CFTags(src, "cfcomponent") {
		if tag.Close {
			continue
		}

		if h, ok := tag.Attr("hint"); ok && strings.TrimSpace(h) != "" {
			return nil
		}

		if hasPrecedingDocComment(src, tag.Start) {
			return nil
		}

		return []cflint.Issue{fc.IssueAt(tag.Start, fmt.Sprintf("Component %s is missing a hint.", compName), compName)}
	}

	// Script component: `component ... {`
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
				absOff := r.Offset + tok.Offset
				braceRel := strings.IndexByte(r.Text[tok.Offset:], '{')

				header := r.Text[tok.Offset:]
				if braceRel >= 0 {
					header = r.Text[tok.Offset : tok.Offset+braceRel]
				}

				if hasHintAttrInHeader(header) || hasPrecedingDocComment(src, absOff) {
					return nil
				}

				return []cflint.Issue{fc.IssueAt(absOff, fmt.Sprintf("Component %s is missing a hint.", compName), compName)}
			}
		}
	}

	return nil
}

func checkFunctionHintMissing(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	spans := fc.ParseResult().FunctionSpans()

	var out []cflint.Issue

	for i := range spans {
		sp := &spans[i]
		if sp.Name == "" || sp.Start < 0 || sp.Body <= sp.Start {
			continue
		}

		header := src[sp.Start:sp.Body]
		hasHint := false

		if sp.Params < 0 {
			for _, tag := range parser.CFTags(header, "cffunction") {
				if !tag.Close {
					if h, ok := tag.Attr("hint"); ok && strings.TrimSpace(h) != "" {
						hasHint = true
					}

					break
				}
			}
		} else {
			hasHint = hasHintAttrInHeader(header)
		}

		if !hasHint && !hasPrecedingDocComment(src, sp.Start) {
			out = append(out, issueInFunc(fc, sp.Start, fmt.Sprintf("Function %s is missing a hint.", sp.Name), sp.Name, sp.Name))
		}
	}

	return out
}

func checkArgHintMissing(fc *FileContext) []cflint.Issue {
	src := fc.Text()

	var out []cflint.Issue

	for _, tag := range parser.CFTags(src, "cfargument") {
		if tag.Close {
			continue
		}

		name, ok := tag.Attr("name")
		if !ok || strings.TrimSpace(name) == "" {
			continue
		}

		if h, has := tag.Attr("hint"); has && strings.TrimSpace(h) != "" {
			continue
		}

		out = append(out, fc.IssueAt(tag.Start, fmt.Sprintf("Argument %s is missing a hint.", name), name))
	}

	return out
}

func checkArgDefaultMissing(fc *FileContext) []cflint.Issue {
	src := fc.Text()

	var out []cflint.Issue

	// 1. Tag `<cfargument>` without `required="true"` and without `default`.
	for _, tag := range parser.CFTags(src, "cfargument") {
		if tag.Close {
			continue
		}

		name, ok := tag.Attr("name")
		if !ok || strings.TrimSpace(name) == "" {
			continue
		}

		if req, hasReq := tag.Attr("required"); hasReq && (strings.EqualFold(req, "true") || strings.EqualFold(req, "yes") || req == "1") {
			continue
		}

		if _, hasDef := tag.Attr("default"); hasDef {
			continue
		}

		out = append(out, fc.IssueAt(tag.Start, fmt.Sprintf("Argument %s is not required and does not define a default value.", name), name))
	}

	// 2. Script function parameters without `required` and without `=`.
	spans := fc.ParseResult().FunctionSpans()
	for i := range spans {
		sp := &spans[i]

		pText := sp.ParamsText(src)
		if pText == "" || sp.Params < 0 {
			continue
		}

		out = append(out, checkScriptArgDefaults(fc, pText, sp.Params+1, sp.Name)...)
	}

	return out
}

func checkScriptArgDefaults(fc *FileContext, paramsText string, baseOffset int, funcName string) []cflint.Issue {
	sc := parser.NewSourceScanner(paramsText)

	var (
		out   []cflint.Issue
		toks  []parser.Token
		depth = 0
	)

	flush := func() {
		if len(toks) == 0 {
			return
		}

		required := false
		hasDefault := false

		var nameTok parser.Token

		for _, t := range toks {
			if t.Kind == parser.TokEquals {
				hasDefault = true

				break
			}

			if t.Kind == parser.TokIdent && strings.EqualFold(t.Value, "required") {
				required = true

				continue
			}

			if t.Kind == parser.TokIdent {
				nameTok = t
			}
		}

		if !required && !hasDefault && nameTok.Kind == parser.TokIdent {
			out = append(out, issueInFunc(fc, baseOffset+nameTok.Offset, fmt.Sprintf("Argument %s is not required and does not define a default value.", nameTok.Value), nameTok.Value, funcName))
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

	return out
}
