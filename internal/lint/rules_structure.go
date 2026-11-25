package lint

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func init() {
	registerStructureRules(DefaultRegistry)
}

func registerStructureRules(reg *Registry) {
	reg.Register(Rule{
		Code:            "PARSE_ERROR",
		DefaultSeverity: "ERROR",
		Check:           checkParseError,
	})
	reg.Register(Rule{
		Code:            "OUTPUT_ATTR",
		DefaultSeverity: "INFO",
		Check:           checkOutputAttr,
	})
	reg.Register(Rule{
		Code:            "NESTED_CFOUTPUT",
		DefaultSeverity: "WARNING",
		Check:           checkNestedCFOutput,
	})
	reg.Register(Rule{
		Code:            "ARG_TYPE_MISSING",
		DefaultSeverity: "WARNING",
		Check:           checkArgTypeMissing,
	})
	reg.Register(Rule{
		Code:            "FUNCTION_ACCESS_REMOTE",
		DefaultSeverity: "WARNING",
		Check:           checkFunctionAccessRemote,
	})
	reg.Register(Rule{
		Code:            "NO_DEFAULT_INSIDE_SWITCH",
		DefaultSeverity: "WARNING",
		Check:           checkNoDefaultInsideSwitch,
	})
	reg.Register(Rule{
		Code:            "COMPARE_INSTEAD_OF_ASSIGN",
		DefaultSeverity: "WARNING",
		Check:           checkCompareInsteadOfAssign,
	})
	reg.Register(Rule{
		Code:            "SQL_SELECT_STAR",
		DefaultSeverity: "WARNING",
		Check:           checkSQLSelectStar,
	})
	reg.Register(Rule{
		Code:            "GLOBAL_VAR",
		DefaultSeverity: "WARNING",
		Check:           checkGlobalVar,
	})
}

func checkParseError(fc *FileContext) []cflint.Issue {
	tree := fc.Tree()
	if tree == nil {
		return nil
	}

	root := tree.RootNode()
	if root == nil || !root.HasError() {
		return nil
	}

	var out []cflint.Issue

	var walk func(n *sitter.Node)

	walk = func(n *sitter.Node) {
		if n.IsError() || n.IsMissing() {
			out = append(out, fc.IssueAt(int(n.StartByte()), "Unable to parse CFML syntax.", ""))

			return
		}

		if !n.HasError() {
			return
		}

		for i := range n.ChildCount() {
			if child := n.Child(i); child != nil {
				walk(child)
			}
		}
	}

	walk(root)

	return out
}

func checkOutputAttr(fc *FileContext) []cflint.Issue {
	src := fc.Text()

	var out []cflint.Issue

	// Tag form: <cfcomponent> and <cffunction>.
	for _, tag := range parser.CFTags(src, "cfcomponent", "cffunction") {
		if tag.Close {
			continue
		}

		if _, ok := tag.Attr("output"); !ok {
			name, _ := tag.Attr("name")
			if name == "" {
				name = tag.Name
			}

			out = append(out, fc.IssueAt(tag.Start, fmt.Sprintf("<%s> is missing the output attribute.", tag.Name), name))
		}
	}

	// Script form: `component ... {` and `function name(...) ... {`.
	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript {
			continue
		}

		out = append(out, checkScriptOutputAttr(fc, r.Text, r.Offset)...)
	}

	return out
}

func checkScriptOutputAttr(fc *FileContext, text string, baseOffset int) []cflint.Issue {
	sc := parser.NewSourceScanner(text)

	var out []cflint.Issue

	for cur := sc.NextSkipComments(); cur.Kind != parser.TokEOF; cur = sc.NextSkipComments() {
		if cur.Kind != parser.TokIdent {
			continue
		}

		switch {
		case strings.EqualFold(cur.Value, "component"):
			if !scanHeaderForOutput(sc, false) {
				out = append(out, fc.IssueAt(baseOffset+cur.Offset, "Component is missing the output attribute.", "component"))
			}
		case strings.EqualFold(cur.Value, "function"):
			nameTok := sc.NextSkipComments()
			if nameTok.Kind != parser.TokIdent {
				continue // anonymous closure
			}

			if !skipParenGroup(sc) {
				continue
			}

			if !scanHeaderForOutput(sc, true) {
				out = append(out, fc.IssueAt(baseOffset+nameTok.Offset, fmt.Sprintf("Function %s is missing the output attribute.", nameTok.Value), nameTok.Value))
			}
		}
	}

	return out
}

func skipParenGroup(sc *parser.Scanner) bool {
	if lp := sc.NextSkipComments(); lp.Kind != parser.TokLParen {
		return false
	}

	depth := 1
	for depth > 0 {
		t := sc.NextSkipComments()
		switch t.Kind {
		case parser.TokEOF:
			return true
		case parser.TokLParen:
			depth++
		case parser.TokRParen:
			depth--
		default:
		}
	}

	return true
}

func scanHeaderForOutput(sc *parser.Scanner, stopAtSemi bool) bool {
	hasOutput := false

	for t := sc.NextSkipComments(); t.Kind != parser.TokEOF && t.Kind != parser.TokLBrace; t = sc.NextSkipComments() {
		if stopAtSemi && t.Kind == parser.TokSemicolon {
			break
		}

		if t.Kind == parser.TokIdent && strings.EqualFold(t.Value, "output") {
			hasOutput = true
		}
	}

	return hasOutput
}

func checkNestedCFOutput(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "<cfoutput") < 0 {
		return nil
	}

	var (
		stack []bool // true if that <cfoutput> has a `group` attribute
		out   []cflint.Issue
	)

	for _, tag := range parser.CFTags(src, "cfoutput") {
		if tag.Close {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

			continue
		}

		_, hasGroup := tag.Attr("group")
		if len(stack) > 0 && !stack[len(stack)-1] && !hasGroup {
			out = append(out, fc.IssueAt(tag.Start, "Nested <cfoutput> without group attribute.", "cfoutput"))
		}

		if !strings.HasSuffix(strings.TrimSpace(tag.Text()), "/>") {
			stack = append(stack, hasGroup)
		}
	}

	return out
}

func checkArgTypeMissing(fc *FileContext) []cflint.Issue {
	src := fc.Text()

	var out []cflint.Issue

	// Tag form: <cfargument name="x"> without type="...".
	for _, tag := range parser.CFTags(src, "cfargument") {
		if tag.Close {
			continue
		}

		if _, ok := tag.Attr("type"); !ok {
			name, _ := tag.Attr("name")
			out = append(out, fc.IssueAt(tag.Start, fmt.Sprintf("Argument %s is missing a type.", name), name))
		}
	}

	// Script form: function f(a, required b) -> untyped parameters.
	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript {
			continue
		}

		out = append(out, checkScriptArgTypes(fc, r.Text, r.Offset)...)
	}

	return out
}

func checkScriptArgTypes(fc *FileContext, text string, baseOffset int) []cflint.Issue {
	sc := parser.NewSourceScanner(text)

	var out []cflint.Issue

	for cur := sc.NextSkipComments(); cur.Kind != parser.TokEOF; cur = sc.NextSkipComments() {
		if cur.Kind != parser.TokIdent || !strings.EqualFold(cur.Value, "function") {
			continue
		}

		next := sc.NextSkipComments()
		if next.Kind == parser.TokIdent {
			next = sc.NextSkipComments()
		}

		if next.Kind != parser.TokLParen {
			continue
		}

		// Parse comma-separated parameters inside `(...)`.
		var paramTokens []parser.Token

		depth := 1
		flushParam := func() {
			if len(paramTokens) == 0 {
				return
			}

			// Strip leading `required` and anything from `=` onward.
			var head []parser.Token

			for _, pt := range paramTokens {
				if pt.Kind == parser.TokEquals {
					break
				}

				if len(head) == 0 && pt.Kind == parser.TokIdent && strings.EqualFold(pt.Value, "required") {
					continue
				}

				head = append(head, pt)
			}

			// Only one identifier means untyped (`a` or `required a`); 2+ tokens means typed (`string a` or `models.User a`).
			if len(head) == 1 && head[0].Kind == parser.TokIdent {
				out = append(out, fc.IssueAt(baseOffset+head[0].Offset, fmt.Sprintf("Argument %s is missing a type.", head[0].Value), head[0].Value))
			}

			paramTokens = paramTokens[:0]
		}

		for depth > 0 {
			t := sc.NextSkipComments()
			if t.Kind == parser.TokEOF {
				break
			}

			switch t.Kind {
			case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
				depth++

				paramTokens = append(paramTokens, t)
			case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
				depth--
				if depth == 0 {
					flushParam()
				} else {
					paramTokens = append(paramTokens, t)
				}
			case parser.TokComma:
				if depth == 1 {
					flushParam()
				} else {
					paramTokens = append(paramTokens, t)
				}
			default:
				paramTokens = append(paramTokens, t)
			}
		}
	}

	return out
}

func checkFunctionAccessRemote(fc *FileContext) []cflint.Issue {
	src := fc.Text()

	var out []cflint.Issue

	for _, tag := range parser.CFTags(src, "cffunction") {
		if tag.Close {
			continue
		}

		if access, ok := tag.Attr("access"); ok && strings.EqualFold(strings.TrimSpace(access), "remote") {
			name, _ := tag.Attr("name")
			out = append(out, fc.IssueAt(tag.Start, fmt.Sprintf("Function %s has remote access.", name), name))
		}
	}

	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript || parser.IndexFold(r.Text, "remote") < 0 {
			continue
		}

		sc := parser.NewSourceScanner(r.Text)

		for cur := sc.NextSkipComments(); cur.Kind != parser.TokEOF; cur = sc.NextSkipComments() {
			if cur.Kind != parser.TokIdent || !strings.EqualFold(cur.Value, "remote") {
				continue
			}

			// Look ahead up to 3 tokens for `function` (e.g. `remote string function foo` or `remote function foo`).
			for range 3 {
				t := sc.NextSkipComments()
				if t.Kind == parser.TokEOF || t.Kind == parser.TokSemicolon || t.Kind == parser.TokLBrace {
					break
				}

				if t.Kind == parser.TokIdent && strings.EqualFold(t.Value, "function") {
					nameTok := sc.NextSkipComments()
					name := nameTok.Value

					out = append(out, fc.IssueAt(r.Offset+cur.Offset, fmt.Sprintf("Function %s has remote access.", name), name))

					break
				}
			}
		}
	}

	return out
}

func checkNoDefaultInsideSwitch(fc *FileContext) []cflint.Issue {
	src := fc.Text()

	var out []cflint.Issue

	// Tag form: <cfswitch> ... </cfswitch> without <cfdefaultcase>.
	if parser.IndexFold(src, "<cfswitch") >= 0 {
		type swFrame struct {
			offset     int
			hasDefault bool
		}

		var stack []swFrame

		for _, tag := range parser.CFTags(src, "cfswitch", "cfdefaultcase") {
			switch tag.Name {
			case "cfswitch":
				if !tag.Close {
					stack = append(stack, swFrame{offset: tag.Start})
				} else if len(stack) > 0 {
					top := stack[len(stack)-1]
					stack = stack[:len(stack)-1]

					if !top.hasDefault {
						out = append(out, fc.IssueAt(top.offset, "Switch statement has no default case.", "cfswitch"))
					}
				}
			case "cfdefaultcase":
				if !tag.Close && len(stack) > 0 {
					stack[len(stack)-1].hasDefault = true
				}
			}
		}
	}

	// Script form: `switch (...) { ... }` without `default:`.
	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript || parser.IndexFold(r.Text, "switch") < 0 {
			continue
		}

		out = append(out, checkScriptSwitchDefault(fc, r.Text, r.Offset)...)
	}

	return out
}

func checkScriptSwitchDefault(fc *FileContext, text string, baseOffset int) []cflint.Issue {
	sc := parser.NewSourceScanner(text)

	var out []cflint.Issue

	for cur := sc.NextSkipComments(); cur.Kind != parser.TokEOF; cur = sc.NextSkipComments() {
		if cur.Kind != parser.TokIdent || !strings.EqualFold(cur.Value, "switch") {
			continue
		}

		swOffset := baseOffset + cur.Offset

		// Advance past `(...)` to `{`.
		if !skipParenGroup(sc) {
			continue
		}

		if t := sc.NextSkipComments(); t.Kind != parser.TokLBrace {
			continue
		}

		bDepth := 1
		hasDefault := false

		for bDepth > 0 {
			tok := sc.NextSkipComments()
			if tok.Kind == parser.TokEOF {
				break
			}

			switch tok.Kind {
			case parser.TokLBrace:
				bDepth++
			case parser.TokRBrace:
				bDepth--
			case parser.TokIdent:
				if bDepth == 1 && strings.EqualFold(tok.Value, "default") {
					if next := sc.NextSkipComments(); next.Kind == parser.TokColon {
						hasDefault = true
					}
				}
			default:
			}
		}

		if !hasDefault {
			out = append(out, fc.IssueAt(swOffset, "Switch statement has no default case.", "switch"))
		}
	}

	return out
}

func checkCompareInsteadOfAssign(fc *FileContext) []cflint.Issue {
	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript {
			continue
		}

		out = append(out, scanCompareInsteadOfAssign(fc, r.Text, r.Offset)...)
	}

	return out
}

func scanCompareInsteadOfAssign(fc *FileContext, text string, baseOffset int) []cflint.Issue {
	sc := parser.NewSourceScanner(text)

	var toks []parser.Token
	for t := sc.NextSkipComments(); t.Kind != parser.TokEOF; t = sc.NextSkipComments() {
		toks = append(toks, t)
	}

	var out []cflint.Issue

	// Split into top-level statements separated by `;`, `{`, `}` at paren/bracket depth 0.
	depth := 0
	stmtStart := 0

	checkStmt := func(stmt []parser.Token) {
		if len(stmt) < 3 {
			return
		}

		if stmt[0].Kind == parser.TokIdent && isControlOrDeclKeyword(stmt[0].Value) {
			return
		}

		d := 0
		cmpOff := -1

		for i := range stmt {
			t := stmt[i]
			switch t.Kind {
			case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
				d++
			case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
				d--
			case parser.TokEquals:
				if d != 0 {
					continue
				}

				var prev, next parser.Token
				if i > 0 {
					prev = stmt[i-1]
				}

				if i+1 < len(stmt) {
					next = stmt[i+1]
				}

				if isStandaloneEquals(prev, t, next) {
					// Real assignment statement (e.g. `b1 = x == y;`).
					return
				}

				if next.Kind == parser.TokEquals && next.Offset == t.Offset+1 && cmpOff < 0 {
					cmpOff = t.Offset
				}
			case parser.TokIdent:
				if d == 0 && strings.EqualFold(t.Value, "eq") && cmpOff < 0 {
					cmpOff = t.Offset
				}
			default:
			}
		}

		if cmpOff >= 0 {
			out = append(out, fc.IssueAt(baseOffset+cmpOff, "Comparison operator used instead of assignment.", "=="))
		}
	}

	for i, t := range toks {
		switch t.Kind {
		case parser.TokLParen, parser.TokLBracket:
			depth++
		case parser.TokRParen, parser.TokRBracket:
			if depth > 0 {
				depth--
			}
		case parser.TokSemicolon, parser.TokLBrace, parser.TokRBrace:
			if depth == 0 {
				checkStmt(toks[stmtStart:i])
				stmtStart = i + 1
			}
		default:
		}
	}

	checkStmt(toks[stmtStart:])

	return out
}

func isControlOrDeclKeyword(s string) bool {
	switch strings.ToLower(s) {
	case "if", "elseif", "else", "while", "for", "switch", "case", "default",
		"return", "var", "function", "component", "interface", "property",
		"import", "throw", "rethrow", "try", "catch", "finally", "break",
		"continue", "param", "lock", "transaction", "savecontent", "include",
		"abort", "exit", "public", "private", "package", "remote", "static", "final":
		return true
	default:
		return false
	}
}

// isStandaloneEquals reports whether tok (`=`) is a bare assignment operator
// rather than part of `==`, `===`, `!=`, `<=`, `>=`, `+=`, `-=`, `*=`, `/=`, `%=`, `&=`.
func isStandaloneEquals(prev, tok, next parser.Token) bool {
	if next.Kind == parser.TokEquals && next.Offset == tok.Offset+1 {
		return false
	}

	if prev.Offset+len(prev.Value) == tok.Offset {
		switch prev.Kind {
		case parser.TokEquals, parser.TokBang, parser.TokLT, parser.TokGT,
			parser.TokPlus, parser.TokMinus, parser.TokStar, parser.TokSlash,
			parser.TokPercent, parser.TokAmpersand:
			return false
		default:
		}
	}

	return true
}

var sqlSelectStarRe = regexp.MustCompile(`(?i)\bselect\s+(?:distinct\s+)?(?:top\s+\d+\s+)?\*`)

func checkSQLSelectStar(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "select") < 0 {
		return nil
	}

	var out []cflint.Issue

	// Tag form: <cfquery> ... </cfquery>
	if parser.IndexFold(src, "<cfquery") >= 0 {
		openEnd := -1

		for _, tag := range parser.CFTags(src, "cfquery") {
			if !tag.Close {
				openEnd = tag.End
			} else if openEnd >= 0 {
				body := src[openEnd:tag.Start]
				if loc := sqlSelectStarRe.FindStringIndex(body); loc != nil {
					out = append(out, fc.IssueAt(openEnd+loc[0], "Avoid using SELECT * in queries.", "*"))
				}

				openEnd = -1
			}
		}
	}

	// Script form: string literals inside queryExecute("SELECT * ...")
	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript || parser.IndexFold(r.Text, "queryExecute") < 0 {
			continue
		}

		sc := parser.NewSourceScanner(r.Text)

		for cur := sc.NextSkipComments(); cur.Kind != parser.TokEOF; cur = sc.NextSkipComments() {
			if cur.Kind == parser.TokIdent && strings.EqualFold(cur.Value, "queryExecute") {
				if lp := sc.NextSkipComments(); lp.Kind == parser.TokLParen {
					if str := sc.NextSkipComments(); str.Kind == parser.TokString {
						if loc := sqlSelectStarRe.FindStringIndex(str.Value); loc != nil {
							out = append(out, fc.IssueAt(r.Offset+str.Offset+loc[0], "Avoid using SELECT * in queries.", "*"))
						}
					}
				}
			}
		}
	}

	return out
}

var globalScopes = map[string]bool{
	"application": true,
	"session":     true,
	"server":      true,
	"request":     true,
	"form":        true,
	"url":         true,
	"cookie":      true,
	"cgi":         true,
	"client":      true,
}

func checkGlobalVar(fc *FileContext) []cflint.Issue {
	// CFLint flags global scope access inside CFCs (skip Application.cfc where
	// application/session lifecycle hooks legitimately access them).
	if !strings.EqualFold(filepath.Ext(fc.Path), ".cfc") || strings.EqualFold(filepath.Base(fc.Path), "Application.cfc") {
		return nil
	}

	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript {
			continue
		}

		sc := parser.NewSourceScanner(r.Text)

		var prev parser.Token

		cur := sc.NextSkipComments()
		for cur.Kind != parser.TokEOF {
			next := sc.NextSkipComments()

			if cur.Kind == parser.TokIdent && globalScopes[strings.ToLower(cur.Value)] &&
				(next.Kind == parser.TokDot || next.Kind == parser.TokLBracket) &&
				prev.Kind != parser.TokDot {
				out = append(out, fc.IssueAt(r.Offset+cur.Offset, fmt.Sprintf("Avoid referencing global scope '%s' directly in a component.", cur.Value), cur.Value))
			}

			prev = cur
			cur = next
		}
	}

	return out
}
