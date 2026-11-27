package lint

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// LineSuppressions holds inline rule suppressions parsed from one source file.
// Line numbers are 1-based to match cflint.Location.Line.
type LineSuppressions struct {
	// byLine maps 1-based line number -> set of uppercased suppressed rule codes.
	byLine map[int]map[string]bool
	// fileWide holds uppercased rule codes suppressed for the whole file.
	fileWide map[string]bool
}

// ParseSuppressions extracts `@CFLintIgnore CODE1,CODE2` and
// `// cflint ignore:CODE1,CODE2` directives from content.
//
// Following CFLint's code-list rule, a code list stops at the first whitespace
// (so `@CFLintIgnore A, B` suppresses A only). A `// cflint ignore:` directive
// applies to the line it appears on; an `@CFLintIgnore` directive applies to
// its own line and the next non-blank line (or the entire file if placed before
// any code on line 1 or on the component header).
func ParseSuppressions(content string) *LineSuppressions {
	s := &LineSuppressions{
		byLine:   make(map[int]map[string]bool),
		fileWide: make(map[string]bool),
	}

	lines := strings.Split(content, "\n")
	for i, raw := range lines {
		lineNum := i + 1
		line := strings.TrimSuffix(raw, "\r")

		// 1. Inline `// cflint ignore:CODE1,CODE2` applies to this line.
		if codes := parseLineIgnore(line); len(codes) > 0 {
			s.addLine(lineNum, codes)
		}

		// 2. Single-line `@CFLintIgnore CODE1,CODE2` applies to this line and
		// the next non-blank line.
		if codes := parseTagIgnore(line); len(codes) > 0 {
			s.addLine(lineNum, codes)
			s.addNextNonBlank(lines, i+1, codes)
		}
	}

	// 3. Multiline `<!--- ... @CFLintIgnore CODE ... --->` comments.
	s.parseMultilineTagIgnores(content, lines)

	return s
}

func (s *LineSuppressions) addNextNonBlank(lines []string, startIdx int, codes []string) {
	for next := startIdx; next < len(lines); next++ {
		trimmed := strings.TrimSpace(lines[next])
		if trimmed == "" {
			continue
		}

		s.addLine(next+1, codes)
		s.extendOverMultilineTag(lines, next, trimmed, codes)

		return
	}
}

// extendOverMultilineTag extends a suppression placed directly above a
// multi-line opening tag (e.g. multi-line <cfif>, <cfset>, <cfloop>) to cover
// all lines of that tag's header, and for <cfquery> covers its SQL body up to
// </cfquery>. It intentionally does NOT extend over container blocks like
// <cffunction>, <cftry>, or <cfcomponent>.
func (s *LineSuppressions) extendOverMultilineTag(lines []string, startIdx int, trimmed string, codes []string) {
	if !strings.HasPrefix(trimmed, "<") || strings.HasPrefix(trimmed, "</") || strings.HasPrefix(trimmed, "<!") {
		return
	}

	low := strings.ToLower(trimmed)

	// 1. Cover all lines of a multi-line opening tag `<cf... \n ... >`.
	endTagIdx := findTagCloseLine(lines, startIdx)
	for l := startIdx + 1; l <= endTagIdx; l++ {
		s.addLine(l+1, codes)
	}

	// 2. For `<cfquery>`, cover the whole query body through `</cfquery>`.
	if strings.HasPrefix(low, "<cfquery") && (len(low) == 8 || !isTagNameChar(low[8])) {
		for l := startIdx; l < len(lines); l++ {
			s.addLine(l+1, codes)

			if strings.Contains(strings.ToLower(lines[l]), "</cfquery>") {
				return
			}
		}
	}
}

func isTagNameChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

// findTagCloseLine returns the 0-based line index where the opening tag
// starting on lines[startIdx] closes with `>` outside quotes.
func findTagCloseLine(lines []string, startIdx int) int {
	var quote byte

	for l := startIdx; l < len(lines); l++ {
		line := lines[l]
		startCol := 0

		if l == startIdx {
			if lt := strings.IndexByte(line, '<'); lt >= 0 {
				startCol = lt + 1
			}
		}

		for i := startCol; i < len(line); i++ {
			c := line[i]
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
				return l
			}
		}
	}

	return startIdx
}

func (s *LineSuppressions) parseMultilineTagIgnores(content string, lines []string) {
	pos := 0
	lineNum := 1 // 1-based line at pos; advanced incrementally so scanning is O(N)

	for {
		openRel := strings.Index(content[pos:], "<!---")
		if openRel < 0 {
			return
		}

		openIdx := pos + openRel
		startLine := lineNum + strings.Count(content[pos:openIdx], "\n")

		closeRel := strings.Index(content[openIdx+5:], "--->")
		if closeRel < 0 {
			return
		}

		closeEnd := openIdx + 5 + closeRel + 4
		body := content[openIdx+5 : openIdx+5+closeRel]
		endLine := startLine + strings.Count(content[openIdx:closeEnd], "\n")

		pos = closeEnd
		lineNum = endLine

		if !strings.Contains(body, "@CFLintIgnore") {
			continue
		}

		codes := parseTagIgnore(body)
		if len(codes) == 0 {
			continue
		}

		for l := startLine; l <= endLine; l++ {
			s.addLine(l, codes)
		}

		s.addNextNonBlank(lines, endLine, codes)
	}
}

func (s *LineSuppressions) addLine(line int, codes []string) {
	m := s.byLine[line]
	if m == nil {
		m = make(map[string]bool, len(codes))
		s.byLine[line] = m
	}

	for _, c := range codes {
		m[c] = true
	}
}

// IsSuppressed reports whether rule code is suppressed on 1-based line.
func (s *LineSuppressions) IsSuppressed(line int, code string) bool {
	if s == nil {
		return false
	}

	k := strings.ToUpper(strings.TrimSpace(code))
	if s.fileWide[k] || s.fileWide["LINE"] {
		return true
	}

	if m := s.byLine[line]; m != nil {
		return m[k] || m["LINE"]
	}

	return false
}

func parseTagIgnore(line string) []string {
	const tag = "@CFLintIgnore"

	var out []string

	for i := 0; ; {
		at := strings.Index(line[i:], tag)
		if at < 0 {
			return out
		}

		at += i
		start := at + len(tag)
		space := lex.SkipRegexpSpace(line, start)

		if space > start {
			if list, end := readCodeList(line, space); list != "" {
				out = append(out, splitCodes(list)...)
				i = end

				continue
			}
		}

		i = at + 1
	}
}

func parseLineIgnore(line string) []string {
	var out []string

	for i := 0; ; {
		at := strings.Index(line[i:], "//")
		if at < 0 {
			return out
		}

		at += i
		j := lex.SkipRegexpSpace(line, at+2)

		rest, ok := strings.CutPrefix(line[j:], "cflint")
		if !ok {
			i = at + 2

			continue
		}

		j += len("cflint")
		after := lex.SkipRegexpSpace(line, j)

		if after == j || !strings.HasPrefix(line[after:], "ignore:") {
			_ = rest
			i = at + 2

			continue
		}

		start := after + len("ignore:")
		if list, end := readCodeList(line, start); list != "" {
			out = append(out, splitCodes(list)...)
			i = end

			continue
		}

		i = at + 2
	}
}

func readCodeList(s string, i int) (string, int) {
	end := i
	for end < len(s) && (lex.IsWord(s[end]) || s[end] == ',') {
		end++
	}

	return s[i:end], end
}

func splitCodes(list string) []string {
	var out []string

	for part := range strings.SplitSeq(list, ",") {
		if c := strings.ToUpper(strings.TrimSpace(part)); c != "" {
			out = append(out, c)
		}
	}

	return out
}
