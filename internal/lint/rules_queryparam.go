package lint

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/parser"
)

func init() {
	registerQueryParamRules(DefaultRegistry)
}

func registerQueryParamRules(reg *Registry) {
	reg.Register(Rule{
		Code:            "CFQUERYPARAM_REQ",
		DefaultSeverity: "WARNING",
		Check:           checkCFQueryParamReq,
	})
	reg.Register(Rule{
		Code:            "QUERYPARAM_REQ",
		DefaultSeverity: "WARNING",
		Check:           checkQueryParamReq,
	})
}

type queryParamMatcher struct {
	allowVarRe  *regexp.Regexp
	allowLineTm string
}

func newQueryParamMatcher(rc *RCConfig) queryParamMatcher {
	var m queryParamMatcher

	if rc == nil {
		return m
	}

	if pat := rc.ParamString("QueryParamChecker.allowVariableExpression"); pat != "" {
		if re, err := regexp.Compile(pat); err == nil {
			m.allowVarRe = re
		}
	}

	m.allowLineTm = rc.ParamString("QueryParamChecker.allowLineExpression")

	return m
}

func (m *queryParamMatcher) allowed(expr, lineText string) bool {
	if m.allowVarRe != nil && m.allowVarRe.MatchString(expr) {
		return true
	}

	if m.allowLineTm != "" {
		pat := strings.ReplaceAll(m.allowLineTm, "${variable}", regexp.QuoteMeta(expr))
		if re, err := regexp.Compile(pat); err == nil && re.MatchString(lineText) {
			return true
		}
	}

	return false
}

func checkCFQueryParamReq(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "<cfquery") < 0 {
		return nil
	}

	matcher := newQueryParamMatcher(fc.RC)
	lines := fc.Lines()

	var out []cflint.Issue

	openEnd := -1

	for _, tag := range parser.CFTags(src, "cfquery") {
		if !tag.Close {
			openEnd = tag.End

			continue
		}

		if openEnd < 0 || tag.Start <= openEnd {
			openEnd = -1

			continue
		}

		body := src[openEnd:tag.Start]
		out = append(out, scanQueryBodyHashes(fc, body, openEnd, lines, &matcher)...)
		openEnd = -1
	}

	return out
}

// scanQueryBodyHashes finds `#expr#` interpolations inside a <cfquery> body
// that are outside <cfqueryparam>, <cfprocparam>, and CFML comments.
func scanQueryBodyHashes(fc *FileContext, body string, baseOffset int, lines []string, matcher *queryParamMatcher) []cflint.Issue {
	var out []cflint.Issue

	for i := 0; i < len(body); {
		switch body[i] {
		case '<':
			if strings.HasPrefix(body[i:], "<!---") {
				i = parser.SkipCFMLComment(body, i)

				continue
			}

			// Skip over any CFML tag (<cfqueryparam ...>, <cfif ...>, <cfloop ...>, etc.)
			// because attributes of <cfqueryparam value="#x#"> are parameterized.
			if end := skipTag(body, i); end > i {
				i = end

				continue
			}

			i++
		case '#':
			// Escaped literal `##`.
			if i+1 < len(body) && body[i+1] == '#' {
				i += 2

				continue
			}

			closeIdx := strings.IndexByte(body[i+1:], '#')
			if closeIdx < 0 {
				return out
			}

			expr := strings.TrimSpace(body[i+1 : i+1+closeIdx])
			off := baseOffset + i
			lineNum, _ := fc.Pos1(off)

			lineText := ""
			if lineNum-1 >= 0 && lineNum-1 < len(lines) {
				lineText = lines[lineNum-1]
			}

			if expr != "" && !matcher.allowed(expr, lineText) {
				msg := fmt.Sprintf("<cfquery> should use <cfqueryparam> for variable '%s'.", expr)
				out = append(out, fc.IssueAt(off, msg, expr))
			}

			i += closeIdx + 2
		default:
			i++
		}
	}

	return out
}

func skipTag(s string, start int) int {
	if start >= len(s) || s[start] != '<' {
		return start
	}

	var quote byte

	for i := start + 1; i < len(s); i++ {
		c := s[i]
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

	return start
}

func checkQueryParamReq(fc *FileContext) []cflint.Issue {
	src := fc.Text()
	if parser.IndexFold(src, "queryExecute") < 0 && parser.IndexFold(src, "setSQL") < 0 {
		return nil
	}

	matcher := newQueryParamMatcher(fc.RC)
	lines := fc.Lines()

	var out []cflint.Issue

	for _, r := range fc.Regions() {
		if r.Kind != parser.RegionScript {
			continue
		}

		sc := parser.NewSourceScanner(r.Text)

		for cur := sc.NextSkipComments(); cur.Kind != parser.TokEOF; cur = sc.NextSkipComments() {
			if cur.Kind != parser.TokIdent {
				continue
			}

			if !strings.EqualFold(cur.Value, "queryExecute") && !strings.EqualFold(cur.Value, "setSQL") {
				continue
			}

			if lp := sc.NextSkipComments(); lp.Kind != parser.TokLParen {
				continue
			}

			str := sc.NextSkipComments()
			if str.Kind != parser.TokString {
				continue
			}

			// Scan `#expr#` inside the SQL string literal. Report at most once
			// per setSQL/queryExecute call to match CFLint's QUERYPARAM_REQ.
			if issues := scanQueryBodyHashes(fc, str.Value, r.Offset+str.Offset, lines, &matcher); len(issues) > 0 {
				out = append(out, issues[0])
			}
		}
	}

	return out
}
