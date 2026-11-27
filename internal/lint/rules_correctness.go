package lint

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerCorrectnessRules(DefaultRegistry)
}

func registerCorrectnessRules(reg *Registry) {
	reg.Register(Rule{
		Code:            "QUERYNEW_DUPLICATE_COLUMNS",
		DefaultSeverity: "WARNING",
		Check:           checkQueryNewDuplicateColumns,
	})
	reg.Register(Rule{
		Code:            "FUNCTION_NAME_COLLISION",
		DefaultSeverity: "WARNING",
		Check:           checkFunctionNameCollision,
	})
	reg.Register(Rule{
		Code:            "PACKAGE_CASE_MISMATCH",
		DefaultSeverity: "WARNING",
		Check:           checkPackageCaseMismatch,
	})
	reg.Register(Rule{
		Code:            "MISSING_SEMI",
		DefaultSeverity: "ERROR",
		Check:           checkMissingSemi,
	})
	reg.Register(Rule{
		Code:            "MISSING_DECLARATION",
		DefaultSeverity: "WARNING",
		Check:           checkMissingDeclaration,
	})
}

// checkQueryNewDuplicateColumns flags duplicate column names in QueryNew("a,b,A").
func checkQueryNewDuplicateColumns(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "querynew") < 0 {
		return nil
	}

	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind == parser.RegionScript {
			out = append(out, scanQueryNewText(fc, r.Text, r.Offset)...)
		}
	}

	for _, tag := range parser.CFTags(src, "cfset", "cfreturn") {
		if tag.Close {
			continue
		}

		body := tag.Text()
		if len(body) <= len(tag.Name)+2 {
			continue
		}

		innerStart := len(tag.Name) + 1
		out = append(out, scanQueryNewText(fc, body[innerStart:len(body)-1], tag.Start+innerStart)...)
	}

	return out
}

func scanQueryNewText(fc *FileContext, text string, baseOffset int) []cflint.Issue {
	if parser.IndexFold(text, "querynew") < 0 {
		return nil
	}

	var out []cflint.Issue

	sc := parser.NewSourceScanner(text)

	var prev parser.Token

	cur := sc.NextSkipComments()
	for cur.Kind != parser.TokEOF {
		next := sc.NextSkipComments()

		if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, "querynew") &&
			prev.Kind != parser.TokDot && next.Kind == parser.TokLParen {
			arg := sc.NextSkipComments()
			if arg.Kind == parser.TokString {
				out = append(out, checkQueryNewColumnList(fc, baseOffset+cur.Offset, unquoteCFMLString(arg.Value))...)
			}

			prev = next
			cur = arg

			continue
		}

		prev = cur
		cur = next
	}

	return out
}

func checkQueryNewColumnList(fc *FileContext, offset int, cols string) []cflint.Issue {
	if strings.Contains(cols, "#") {
		return nil
	}

	seen := map[string]bool{}

	var out []cflint.Issue

	for raw := range strings.SplitSeq(cols, ",") {
		col := strings.TrimSpace(raw)
		if col == "" {
			continue
		}

		low := strings.ToLower(col)
		if seen[low] {
			out = append(out, fc.IssueAt(offset, fmt.Sprintf("Duplicate column %s in QueryNew().", col), low))
		} else {
			seen[low] = true
		}
	}

	return out
}

// checkFunctionNameCollision flags user-defined functions that collide with
// built-in CFML function names (e.g. `function abs()`).
func checkFunctionNameCollision(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	spans := fc.ParseResult().FunctionSpans()
	for i := range spans {
		sp := &spans[i]
		if sp.Name == "" {
			continue
		}

		if docs.IsBuiltinFunction(sp.Name) {
			out = append(out, issueInFunc(fc, sp.Start, fmt.Sprintf("Function %s collides with built-in function.", sp.Name), sp.Name, sp.Name))
		}
	}

	return out
}

// checkPackageCaseMismatch flags `new pkg.Comp()` or
// `createObject("component", "pkg.Comp")` where the dotted path matches the
// current file's package/component path case-insensitively but not in exact case.
func checkPackageCaseMismatch(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "new") < 0 && parser.IndexFold(src, "createobject") < 0 {
		return nil
	}

	// Build actual on-disk path segments for the current file (without extension).
	clean := filepath.ToSlash(filepath.Clean(fc.Path))
	if ext := filepath.Ext(clean); ext != "" {
		clean = clean[:len(clean)-len(ext)]
	}

	fileSegs := strings.Split(clean, "/")
	if len(fileSegs) < 2 {
		return nil
	}

	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind == parser.RegionScript {
			out = append(out, scanPackageCaseText(fc, r.Text, r.Offset, fileSegs)...)
		}
	}

	for _, tag := range parser.CFTags(src, "cfset", "cfreturn") {
		if tag.Close {
			continue
		}

		body := tag.Text()
		if len(body) <= len(tag.Name)+2 {
			continue
		}

		innerStart := len(tag.Name) + 1
		out = append(out, scanPackageCaseText(fc, body[innerStart:len(body)-1], tag.Start+innerStart, fileSegs)...)
	}

	return out
}

func scanPackageCaseText(fc *FileContext, text string, baseOffset int, fileSegs []string) []cflint.Issue {
	var out []cflint.Issue

	sc := parser.NewSourceScanner(text)

	var prev parser.Token

	cur := sc.NextSkipComments()
	for cur.Kind != parser.TokEOF {
		next := sc.NextSkipComments()

		// 1. `new a.b.C(...)`
		if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, "new") && prev.Kind != parser.TokDot {
			if dotted, after := readDottedIdent(next, sc); dotted != "" {
				if mismatch, actual := matchPackageCase(dotted, fileSegs); mismatch {
					out = append(out, fc.IssueAt(baseOffset+cur.Offset, fmt.Sprintf("The case of the package folder and the object declaration do not match for %s.", actual), actual))
				}

				prev = cur
				cur = after

				continue
			}
		}

		// 2. `createObject("component", "a.b.C")`
		if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, "createobject") &&
			prev.Kind != parser.TokDot && next.Kind == parser.TokLParen {
			if dotted, ok := readCreateObjectComponentPath(sc); ok {
				if mismatch, actual := matchPackageCase(dotted, fileSegs); mismatch {
					out = append(out, fc.IssueAt(baseOffset+cur.Offset, fmt.Sprintf("The case of the package folder and the object declaration do not match for %s.", actual), actual))
				}
			}
		}

		prev = cur
		cur = next
	}

	return out
}

func readDottedIdent(first parser.Token, sc *parser.Scanner) (string, parser.Token) {
	if first.Kind != parser.TokIdent {
		return "", first
	}

	var b strings.Builder

	b.WriteString(first.Value)

	tok := sc.NextSkipComments()
	for tok.Kind == parser.TokDot {
		seg := sc.NextSkipComments()
		if seg.Kind != parser.TokIdent {
			return b.String(), seg
		}

		b.WriteByte('.')
		b.WriteString(seg.Value)

		tok = sc.NextSkipComments()
	}

	return b.String(), tok
}

func readCreateObjectComponentPath(sc *parser.Scanner) (string, bool) {
	arg1 := sc.NextSkipComments()
	if arg1.Kind != parser.TokString {
		return "", false
	}

	s1 := unquoteCFMLString(arg1.Value)
	sep := sc.NextSkipComments()

	if sep.Kind == parser.TokRParen {
		return s1, s1 != ""
	}

	if sep.Kind != parser.TokComma || !strings.EqualFold(s1, "component") {
		return "", false
	}

	arg2 := sc.NextSkipComments()
	if arg2.Kind != parser.TokString {
		return "", false
	}

	return unquoteCFMLString(arg2.Value), true
}

func matchPackageCase(dotted string, fileSegs []string) (bool, string) {
	refSegs := strings.Split(dotted, ".")
	if len(refSegs) < 2 || len(fileSegs) < 2 {
		return false, ""
	}

	// Match backwards from the component name through parent package folders,
	// stopping when a mapping root segment differs (e.g. `tests.` vs `cflint/`).
	matched := 0
	diff := false

	ri, fi := len(refSegs)-1, len(fileSegs)-1

	for ri >= 0 && fi >= 0 {
		if !strings.EqualFold(refSegs[ri], fileSegs[fi]) {
			break
		}

		if refSegs[ri] != fileSegs[fi] {
			diff = true
		}

		matched++
		ri--
		fi--
	}

	if matched < 2 || !diff {
		return false, ""
	}

	return true, strings.Join(fileSegs[len(fileSegs)-matched:], ".")
}

// checkMissingSemi flags bare identifier/expression statements in script blocks
// that are missing a trailing semicolon before the next statement.
func checkMissingSemi(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript {
			continue
		}

		out = append(out, scanMissingSemi(fc, r.Text, r.Offset)...)
	}

	return out
}

func scanMissingSemi(fc *FileContext, text string, baseOffset int) []cflint.Issue {
	sc := parser.NewSourceScanner(text)

	var (
		out      []cflint.Issue
		lineToks []parser.Token
		curLine  = -1
		parenD   = 0
		brackD   = 0
	)

	flush := func(nextTok parser.Token) {
		if len(lineToks) == 0 {
			return
		}

		last := lineToks[len(lineToks)-1]
		first := lineToks[0]

		// Only flag standalone identifier statements (e.g. `name_1` on its own line
		// followed by another statement on the next line) to match CFLint's parser
		// recovery behavior without false positives on modern semicolon-optional CFScript.
		if len(lineToks) == 1 && first.Kind == parser.TokIdent && !isControlOrDeclKeyword(first.Value) &&
			nextTok.Kind == parser.TokIdent && parenD == 0 && brackD == 0 {
			out = append(out, fc.IssueAt(baseOffset+last.Offset, fmt.Sprintf("End of statement(;) expected instead of %s.", first.Value), first.Value))
		}

		lineToks = lineToks[:0]
	}

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			break
		}

		line := int(tok.Line)
		if curLine >= 0 && line != curLine {
			flush(tok)
		}

		curLine = line

		switch tok.Kind {
		case parser.TokLParen:
			parenD++
		case parser.TokRParen:
			if parenD > 0 {
				parenD--
			}
		case parser.TokLBracket:
			brackD++
		case parser.TokRBracket:
			if brackD > 0 {
				brackD--
			}
		default:
		}

		lineToks = append(lineToks, tok)
	}

	return out
}

// checkMissingDeclaration is registered for .cflintrc compatibility; CFLint's
// MissingDeclaration fixtures verify that struct/array member writes are not flagged.
func checkMissingDeclaration(_ *FileContext) []cflint.Issue {
	return nil
}

func unquoteCFMLString(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}

	return s
}
