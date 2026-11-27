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

		// 2. Single-line `@CFLintIgnore CODE1,CODE2` or `<!--- CFLINT-DISABLE CODE1, CODE2 --->`
		// applies to this line and the next non-blank line.
		if codes := parseTagIgnore(line); len(codes) > 0 {
			s.addLine(lineNum, codes)
			s.addNextNonBlank(lines, i+1, codes)
		}

		if codes := parseCFLintDisable(line); len(codes) > 0 {
			s.addLine(lineNum, codes)
			s.addNextNonBlank(lines, i+1, codes)
		}
	}

	// 3. Multiline `<!--- ... @CFLintIgnore CODE ... --->` and `/* ... @CFLintIgnore CODE ... */` comments.
	s.parseMultilineBlockIgnores(content, lines, "<!---", "--->")
	s.parseMultilineBlockIgnores(content, lines, "/*", "*/")

	return s
}

func (s *LineSuppressions) addNextNonBlank(lines []string, startIdx int, codes []string) {
	for next := startIdx; next < len(lines); next++ {
		trimmed := strings.TrimSpace(lines[next])
		if trimmed == "" {
			continue
		}

		s.addLine(next+1, codes)
		s.extendOverTarget(lines, next, trimmed, codes)

		return
	}
}

// extendOverTarget extends a suppression placed directly above a multi-line
// opening tag (e.g. `<cfif>`, `<cfset>`, `<cfloop>`), a `<cfquery>`,
// `<cfscript>`, `<cffunction>`, `<cfcomponent>`, or a script `component`/`function`
// declaration to cover that target's body. It intentionally does NOT extend over
// arbitrary container tags like `<cftry>`.
func (s *LineSuppressions) extendOverTarget(lines []string, startIdx int, trimmed string, codes []string) {
	low := strings.ToLower(trimmed)

	// Script `component ... {` suppresses the whole component file.
	if isScriptComponentHeader(low) {
		s.addFileWide(codes)

		return
	}

	// Script `[access] [returntype] function name(...) {` covers the function body.
	if isScriptFunctionHeader(low) {
		endIdx := findBraceBlockEndLine(lines, startIdx)
		for l := startIdx; l <= endIdx; l++ {
			s.addLine(l+1, codes)
		}

		return
	}

	if !strings.HasPrefix(trimmed, "<") || strings.HasPrefix(trimmed, "</") || strings.HasPrefix(trimmed, "<!") {
		return
	}

	// 1. Cover all lines of a multi-line opening tag `<cf... \n ... >`.
	endTagIdx := findTagCloseLine(lines, startIdx)
	for l := startIdx + 1; l <= endTagIdx; l++ {
		s.addLine(l+1, codes)
	}

	// 2. `<cfcomponent>` suppresses the entire component file.
	if hasTagPrefix(low, "<cfcomponent") {
		s.addFileWide(codes)

		return
	}

	// 3. Cover body of `<cfquery>`, `<cffunction>`, and `<cfscript>` through closing tag.
	for _, pair := range [][2]string{
		{"<cfquery", "</cfquery>"},
		{"<cffunction", "</cffunction>"},
		{"<cfscript", "</cfscript>"},
	} {
		if hasTagPrefix(low, pair[0]) {
			for l := startIdx; l < len(lines); l++ {
				s.addLine(l+1, codes)

				if strings.Contains(strings.ToLower(lines[l]), pair[1]) {
					return
				}
			}
		}
	}
}

func hasTagPrefix(low, prefix string) bool {
	return strings.HasPrefix(low, prefix) && (len(low) == len(prefix) || !isTagNameChar(low[len(prefix)]))
}

func isScriptComponentHeader(low string) bool {
	if !strings.HasPrefix(low, "component") {
		return false
	}

	return len(low) == len("component") || !isTagNameChar(low[len("component")])
}

func isScriptFunctionHeader(low string) bool {
	idx := strings.Index(low, "function")
	if idx < 0 {
		return false
	}

	if idx > 0 && isTagNameChar(low[idx-1]) {
		return false
	}

	after := idx + len("function")
	if after < len(low) && isTagNameChar(low[after]) {
		return false
	}

	return strings.Contains(low[after:], "(")
}

func findBraceBlockEndLine(lines []string, startIdx int) int {
	depth := 0
	opened := false

	for l := startIdx; l < len(lines); l++ {
		line := lines[l]
		for i := range len(line) {
			switch line[i] {
			case '{':
				depth++
				opened = true
			case '}':
				depth--
				if opened && depth == 0 {
					return l
				}
			}
		}
	}

	return startIdx
}

func (s *LineSuppressions) addFileWide(codes []string) {
	for _, c := range codes {
		s.fileWide[c] = true
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

func (s *LineSuppressions) parseMultilineBlockIgnores(content string, lines []string, openTok, closeTok string) {
	pos := 0
	lineNum := 1 // 1-based line at pos; advanced incrementally so scanning is O(N)

	for {
		openRel := strings.Index(content[pos:], openTok)
		if openRel < 0 {
			return
		}

		openIdx := pos + openRel
		startLine := lineNum + strings.Count(content[pos:openIdx], "\n")

		closeRel := strings.Index(content[openIdx+len(openTok):], closeTok)
		if closeRel < 0 {
			return
		}

		closeEnd := openIdx + len(openTok) + closeRel + len(closeTok)
		body := content[openIdx+len(openTok) : openIdx+len(openTok)+closeRel]
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

// parseCFLintDisable parses `<!--- CFLINT-DISABLE [CODE1, CODE2] --->`
// matching CFLint's CFLINT_DISABLE_PATTERN. An empty code list disables all
// rules ("LINE") on the following element.
func parseCFLintDisable(line string) []string {
	_, rest, ok := strings.Cut(strings.ToUpper(line), "CFLINT-DISABLE")
	if !ok {
		return nil
	}

	if before, _, found := strings.Cut(rest, "---"); found {
		rest = before
	}

	rest = strings.TrimSpace(rest)
	if rest == "" {
		return []string{"LINE"}
	}

	var out []string

	for part := range strings.SplitSeq(rest, ",") {
		if code := strings.TrimSpace(part); code != "" {
			out = append(out, code)
		}
	}

	return out
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
