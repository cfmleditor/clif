package parser

import (
	"cmp"
	"slices"
	"strings"

	"go.lsp.dev/uri"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
)

// The entry points below are readings of one parse, ParseWithOptions,
// shallow so that nothing is typed or settled after the walk. They used to
// run the script and tag walkers themselves, without the parse's region split
// by file name, its output-context gate, the whole-file tag function scopes it
// gives the tag walk, or its seeding of a <cfscript> island with the tag
// function it sits in, so their answers could differ from a parse of the same
// file.

// ParseFunctionDefs extracts function definitions from CFC content: a
// shallow parse's Funcs.
func ParseFunctionDefs(fileURI uri.URI, content string) []FunctionDef {
	return ParseWithOptions(fileURI, content, &ParseOptions{Shallow: true}).Funcs
}

// ParseComponentRefs extracts component references from source content: a
// shallow parse's ComponentRefs, in line order.
func ParseComponentRefs(fileURI uri.URI, content string) []ComponentRef {
	refs := ParseWithOptions(fileURI, content, &ParseOptions{Shallow: true}).ComponentRefs
	slices.SortStableFunc(refs, func(a, b ComponentRef) int { return cmp.Compare(a.Line, b.Line) })

	return refs
}

// ParseVars extracts variable declarations from content: AllVars of a
// shallow parse that keeps them (ParseOptions.KeepVars). Each is found once
// by name and line, with its function, and a var or local outside any
// function is read as variables.
func ParseVars(content string) []VarDef {
	return ParseWithOptions("", content, &ParseOptions{Shallow: true, KeepVars: true}).allVars
}

// varDefs is AllVars' answer from each region's declarations, found, and
// the file's function scopes: each declaration once by name and line, with
// its function, and a var or local outside any function read as variables.
func varDefs(found [][]VarDef, scopes []FuncScope) []VarDef {
	var defs []VarDef

	type key struct {
		name string
		line uint32
	}

	seen := make(map[key]bool)

	for _, regionVars := range found {
		for _, v := range regionVars {
			k := key{strings.ToLower(v.Name), v.Line}
			if seen[k] {
				continue
			}

			seen[k] = true

			fs := findFuncScope(int(v.Line), scopes)
			// var/local outside a function → variables scope
			if fs.Start == -1 && v.Scope == ScopeLocal {
				v.Scope = ScopeVariables
			}

			v.FuncStart = fs.Start
			v.FuncEnd = fs.End
			defs = append(defs, v)
		}
	}

	return defs
}

// GlobalVars returns this.x and variables.x names declared outside any function.
func GlobalVars(content string) []string {
	allVars := ParseVars(content)
	seen := make(map[string]bool)

	var names []string

	for _, v := range allVars {
		if v.FuncStart != -1 {
			continue
		}

		switch v.Scope {
		case ScopeVariables, ScopeThis:
			if !seen[v.Name] {
				seen[v.Name] = true

				names = append(names, v.Name)
			}
		default:
			// Every other scope is not a file-level name.
		}
	}

	return names
}

// VarsInFunc returns local/arguments variable names within the function spanning [funcStart, funcEnd].
func VarsInFunc(content string, funcStart, funcEnd int) []string {
	allVars := ParseVars(content)
	seen := make(map[string]bool)

	var names []string

	for _, v := range allVars {
		if v.FuncStart != funcStart || v.FuncEnd != funcEnd {
			continue
		}

		switch v.Scope {
		case ScopeLocal, ScopeArguments:
			if !seen[v.Name] {
				seen[v.Name] = true

				names = append(names, v.Name)
			}
		default:
			// Every other scope is not one of the function's own names.
		}
	}

	return names
}

// FindFuncScopes returns function line ranges in the content: the Scopes of
// a shallow parse that reads nothing else (ParseOptions.ScopesOnly).
func FindFuncScopes(content string) []FuncScope {
	return ParseWithOptions("", content, &ParseOptions{Shallow: true, ScopesOnly: true}).Scopes
}

// skipFunctionAttrs consumes the attributes a script function declaration may
// carry after its parameter list — `localmode=true`, `skip="true"`,
// `restPath="x" httpMethod="GET"`, `cachedWithin=createTimeSpan(0,1,0,0)` — so
// the next token is the body's `{`, or the `;` of an abstract declaration. A
// value is optional, as in tag syntax, and Lucee also takes a colon for the
// `=` (`secured:api`, `access:"remote"`).
//
// Every reader of a declaration expected the `{` straight after the `)`, and
// finding an attribute there it recorded a scope ending on the function's own
// line and read the body as component-level code: its locals became variables
// of the component and its calls were attributed to no function. Lucee's test
// suite writes 739 functions this way. It is only for a *named* declaration:
// after an arrow function's parameters comes an expression, and consuming an
// identifier there would take the receiver off a call.
//
// A value is an expression, and value is called with the attribute's name
// and the value's first token once that is consumed; it must leave the
// scanner past the value. That is how the script parser records
// `createTimeSpan` in either spelling — a bare call, or inside a string's
// #...# — as the component-level read of the body used to. With value nil, a
// call's argument list is skipped.
func skipFunctionAttrs(sc *Scanner, value func(name, v Token)) {
	for sc.PeekSkipComments().Kind == TokIdent {
		name := sc.NextSkipComments()

		if k := sc.PeekSkipComments().Kind; k != TokEquals && k != TokColon {
			continue
		}

		sc.NextSkipComments()

		v := sc.PeekSkipComments()
		if v.Kind != TokString && v.Kind != TokIdent && v.Kind != TokNumber {
			continue
		}

		sc.NextSkipComments()

		switch {
		case value != nil:
			value(name, v)
		case v.Kind == TokIdent && sc.PeekSkipComments().Kind == TokLParen:
			sc.NextSkipComments()
			skipToClose(sc, TokLParen, TokRParen)
		default:
		}
	}
}

// skipToClose consumes tokens up to the close that balances an open already
// consumed, and returns the last token it read: that close, or EOF.
func skipToClose(sc *Scanner, open, closing TokenKind) Token {
	depth := 1

	var last Token

	for depth > 0 {
		last = sc.NextSkipComments()
		if last.Kind == TokEOF {
			break
		}

		switch last.Kind {
		case open:
			depth++
		case closing:
			depth--
		default:
		}
	}

	return last
}

// inScriptRegion reports whether offset at falls inside the text of one of
// regions' script regions.
func inScriptRegion(regions []Region, at int) bool {
	script := false

	for i := range regions {
		r := &regions[i]
		if at < r.Offset || at >= r.Offset+len(r.Text) {
			continue
		}

		// A tag island lies inside its script region and is tags.
		if r.Island {
			return false
		}

		script = script || r.Kind == RegionScript
	}

	return script
}

// commentCovering is the end of the <!--- comment that begins at or after
// from and holds offset at, or -1 when at is in none.
func commentCovering(src string, from, at int) int {
	for k := from; k < at; {
		o := strings.Index(src[k:at], "<!---")
		if o < 0 {
			return -1
		}

		end := SkipCFMLComment(src, k+o)
		if end > at {
			return end
		}

		k = end
	}

	return -1
}

// findTagFuncScopesIdx finds <cffunction>...</cffunction> boundaries, for a
// caller holding a line index for exactly this src. ClassifyRegions builds one over the whole file on its
// way to the regions, and extractSignatures then asked for the scopes of that
// same whole file — so the index was built twice over one string, every parse.
// A nil idx means build one.
//
// A `<cffunction` inside one of the file's script regions (regions, in file
// order) is text, not a tag: a string such as `assert( "a <cffunction> …" )`
// in a <cfscript> block took the next function's </cffunction> as its own and
// swallowed that function.
func findTagFuncScopesIdx(src string, baseLine int, idx LineTable, regions []Region) []FuncScope {
	// A modest capacity rather than a counted one. Counting <cffunction first
	// sizes the slice exactly, but it is a second pass over the whole source and
	// it runs on script components too, where it scans everything to find
	// nothing — measurably, at +1.6% on script parsing for a slice that append
	// grows well enough on its own.
	scopes := make([]FuncScope, 0, 8)

	if idx == nil {
		idx = NewLineTable(src)
	}

	pos := 0

	for {
		i := indexCFTag(src[pos:], "cffunction")
		if i < 0 {
			break
		}

		i += pos

		// A <cffunction in a comment is none. The comments are read by the one
		// comment reader, which closes a comment of dashes alone where a count
		// of "<!---" and "--->" left `<!----->` open.
		if end := commentCovering(src, pos, i); end >= 0 {
			pos = end

			continue
		}

		if inScriptRegion(regions, i) {
			pos = i + 11

			continue
		}

		startLine := baseLine + idx.LineOf(i)

		end := min(i+200, len(src))

		funcName := getAttr(src[i:end], "name")
		funcAccess := getAttr(src[i:end], "access")
		funcReturn := getAttr(src[i:end], "returntype")

		// The span reads the whole tag, which ends at its own '>' rather than
		// 200 bytes on, and its body starts past that.
		span := &FunctionSpan{Kind: FunctionTag, Line: startLine, Start: i, Params: -1, ParamsEnd: -1, Body: -1, BodyEnd: -1}
		bodyFrom := -1

		if gt := tagEndIndex(src[i:]); gt >= 0 {
			tag := src[i : i+gt+1]
			span.Name, span.Access = getAttr(tag, "name"), getAttr(tag, "access")
			bodyFrom = i + gt + 1

			// `<cffunction name="x" />`, an interface's, has no body: the
			// next </cffunction> closes the function after it, which the
			// search for it here used to swallow.
			if strings.HasSuffix(strings.TrimRight(tag[:len(tag)-1], " \t\r\n"), "/") {
				span.End = bodyFrom
				scopes = append(scopes, FuncScope{Name: funcName, Access: funcAccess, ReturnType: funcReturn, Start: startLine, End: startLine, span: span})
				pos = bodyFrom

				continue
			}
		} else {
			span.Name, span.Access = funcName, funcAccess
		}

		closeIdx := indexCFTag(src[i+11:], "/cffunction")
		if closeIdx < 0 {
			span.End = len(src)
			scopes = append(scopes, FuncScope{Name: funcName, Access: funcAccess, ReturnType: funcReturn, Start: startLine, End: baseLine + len(idx) - 1, span: span})

			break
		}

		closeEnd := i + 11 + closeIdx

		if bodyFrom >= 0 && bodyFrom <= closeEnd {
			span.Body, span.BodyEnd = bodyFrom, closeEnd
		}

		gt := strings.IndexByte(src[closeEnd:], '>')

		if gt >= 0 {
			closeEnd += gt + 1
		}

		span.End = closeEnd

		endLine := baseLine + idx.LineOf(closeEnd)
		scopes = append(scopes, FuncScope{Name: funcName, Access: funcAccess, ReturnType: funcReturn, Start: startLine, End: endLine, span: span})
		pos = closeEnd
	}

	return scopes
}

// lineCursor answers LineTable.LineOf for a walk that mostly moves forward. A
// document walk asks for the line of each tag's start, its end and, for a
// closing tag, the line of the content before it: offsets within a few lines
// of the last answer, which a short step from that line finds without the
// binary search over every line in the file. An offset further away falls
// back to LineOf, so the answer is always LineOf's.
type lineCursor struct {
	idx  LineTable
	line int
}

// maxCursorSteps is how many lines a lineCursor walks before it searches.
const maxCursorSteps = 8

// at returns c.idx.LineOf(offset).
func (c *lineCursor) at(offset int) int {
	idx, l := c.idx, c.line

	for range maxCursorSteps {
		switch {
		case l+1 < len(idx) && int(idx[l+1]) <= offset:
			l++
		case int(idx[l]) > offset:
			l--
		default:
			c.line = l

			return l
		}

		if l < 0 {
			break
		}
	}

	l = idx.LineOf(offset)
	c.line = max(l, 0)

	return l
}

// ClassifyRegions segments CFC content into script and tag regions.
func ClassifyRegions(content string) []Region {
	regions, _ := ClassifyRegionsIdx(content)

	return regions
}

// ClassifyRegionsIdx is ClassifyRegions plus the line index it built on the way,
// so a caller that also needs line numbers for this content does not build a
// second one. The index is nil for a script file, which never reaches the code
// that wants it.
func ClassifyRegionsIdx(content string) ([]Region, LineTable) {
	if isScriptFile(content) {
		return splitTagIslands([]Region{{Kind: RegionScript, StartLine: 0, Text: content}}), nil
	}

	regions, lines := splitCFScriptBlocks(content)

	return splitTagIslands(regions), lines
}

// isScriptFile checks whether the first non-whitespace, non-comment token
// indicates a script-based CFC, or whether the content has no CF tags.
func isScriptFile(content string) bool {
	sc := NewScanner(content)

	tok := sc.NextSkipComments()
	if tok.Kind == TokIdent {
		lower := strings.ToLower(tok.Value)
		if lower == "component" || lower == "interface" || lower == "property" || lower == "import" {
			return true
		}
	}

	// A literal <script> tag (HTML/JavaScript, not <cfscript>) is strong
	// evidence this is an HTML page with embedded JS rather than a pure
	// CFScript component — even one with zero CF tags of its own (e.g. a
	// plain HTML dialog template). Without this check, "no CF tags found at
	// all" below would misclassify the whole file as CFScript and feed raw
	// JavaScript into the CFML scanner.
	if hasScriptTag(content) {
		return false
	}

	// If no CF tags found at all, treat as script, unless its first line
	// could not begin CFScript: then it is a text file a .cfm extension lets
	// the engine serve, an INI or properties file or Markdown.
	return !containsCFTag(content) && !startsAsText(content)
}

// notSectionName reports whether r cannot be in an INI section's name.
func notSectionName(r rune) bool {
	return r != '_' && r != '-' && r != '.' && r != ' ' && (r < '0' || r > '9') && (r < 'a' || r > 'z') && (r < 'A' || r > 'Z')
}

// startsAsText reports whether content's first non-blank line is one no
// CFScript begins with: a # (a Markdown heading, a properties comment), or an
// INI section header, [name] alone on its line. Mura's settings.ini.cfm and
// its templates, ContentBox's runtime.properties.cfm and cbmarkdown's
// markdown.md.cfm were each read as one script region, and every key = value
// line in them as a declaration; the tree-sitter grammar reads them as text.
// A #! line is the shebang RustCFML runs a script by, and stays script.
func startsAsText(content string) bool {
	content = strings.TrimPrefix(content, bomUTF8)

	for content != "" {
		line, rest, _ := strings.Cut(content, "\n")
		content = rest

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "#!"):
			return false
		case line[0] == '#':
			return true
		case len(line) > 2 && line[0] == '[' && line[len(line)-1] == ']':
			name := line[1 : len(line)-1]

			return name != "" && strings.IndexFunc(name, notSectionName) < 0
		default:
			return false
		}
	}

	return false
}

// hasScriptTag reports whether content contains a literal <script ...> or
// <script> tag (case-insensitive), as opposed to <cfscript>.
// hasScriptTag reports whether content holds an HTML <script> tag.
//
// Walked with strings.IndexByte and gated on one folded byte, for the reason
// indexCFTag is: almost no '<' in a page starts the tag being looked for, so
// EqualFold should not be reached for most of them. This runs over whole files
// during region classification.
func hasScriptTag(content string) bool {
	for off := 0; off < len(content); {
		i := strings.IndexByte(content[off:], '<')
		if i < 0 {
			return false
		}

		at := off + i
		off = at + 1

		if at+7 >= len(content) {
			return false
		}

		if lex.LowerASCII(content[at+1]) != 's' {
			continue
		}

		if !strings.EqualFold(content[at+1:at+7], "script") {
			continue
		}

		switch content[at+7] {
		case '>', '/', ' ', '\t', '\n', '\r':
			return true
		}
	}

	return false
}

// containsCFTag checks if content has any <cf (case-insensitive) without allocating.
func containsCFTag(s string) bool {
	for off := 0; off+2 < len(s); {
		i := strings.IndexByte(s[off:], '<')
		if i < 0 {
			break
		}

		at := off + i
		off = at + 1

		if at+2 < len(s) && lex.LowerASCII(s[at+1]) == 'c' && lex.LowerASCII(s[at+2]) == 'f' {
			return true
		}
	}

	return false
}

// scriptSkipSpan is the byte range [start, end) of a literal <script>...
// </script> block (not <cfscript>) that contains no "<cf" tag of its own.
type scriptSkipSpan struct {
	start, end int
}

// findScriptSkipSpans scans content for <script>...</script> blocks (case-
// insensitive) that contain no "<cf" anywhere inside, returning their byte
// ranges in order. CFML comments are skipped so a <script> tag inside a
// <!--- ... ---> comment doesn't produce a spurious span. A <script> block
// that DOES contain a "<cf" tag is deliberately left out of the result, so
// real CFML mixed into it (a stray <cfoutput>, a nested <cfscript>) is still
// parsed normally rather than being treated as opaque.
func findScriptSkipSpans(content string) []scriptSkipSpan {
	var spans []scriptSkipSpan

	ends := newTagEnds(content)

	pos := 0

	for pos < len(content) {
		i := strings.IndexByte(content[pos:], '<')
		if i < 0 {
			break
		}

		i += pos

		if i+4 < len(content) && content[i:i+5] == "<!---" {
			pos = SkipCFMLComment(content, i)

			continue
		}

		if i+7 >= len(content) || !strings.EqualFold(content[i+1:i+7], "script") {
			pos = i + 1

			continue
		}

		switch content[i+7] {
		case '>', '/', ' ', '\t', '\n', '\r':
			// tag boundary — proceed
		default:
			pos = i + 1

			continue
		}

		gt := ends.at(i)
		if gt < 0 {
			break
		}

		bodyStart := i + gt + 1

		closeIdx := indexCFTag(content[bodyStart:], "/script>")

		var bodyEnd, spanEnd int

		if closeIdx < 0 {
			bodyEnd = len(content)
			spanEnd = len(content)
		} else {
			bodyEnd = bodyStart + closeIdx
			spanEnd = bodyEnd + len("</script>")
		}

		if !containsCFTag(content[bodyStart:bodyEnd]) {
			spans = append(spans, scriptSkipSpan{start: i, end: spanEnd})
		}

		pos = spanEnd
	}

	return spans
}

// splitCFScriptBlocks splits tag-based content into tag, script, and skip
// regions. CFML comments (<!--- ... --->) are skipped so that <cfscript>
// blocks inside comments do not produce spurious script regions. Literal
// <script>...</script> blocks with no CFML inside (see findScriptSkipSpans)
// are emitted as RegionSkip so their JavaScript is never scanned as CFML.
func splitCFScriptBlocks(content string) ([]Region, LineTable) {
	idx := NewLineTable(content)
	skipSpans := findScriptSkipSpans(content)
	skipIdx := 0

	var regions []Region

	// add appends a region of kind for content[from:to], unless it is blank.
	add := func(kind RegionKind, from, to int) {
		if text := content[from:to]; strings.TrimSpace(text) != "" {
			regions = append(regions, Region{Kind: kind, StartLine: idx.LineOf(from), Text: text, Offset: from})
		}
	}

	pos := 0

	// The next <cfscript> is found once and kept until the walk passes it.
	// Searching again for every skip span scanned to the end of a page with
	// no <cfscript> once per <script> block, which was quadratic.
	openIdx := -2 // -1 for none left

	for {
		if openIdx != -1 && openIdx < pos {
			openIdx = nextCFScriptOpen(content, pos)
		}

		// Find the next script-skip span at or after pos.
		for skipIdx < len(skipSpans) && skipSpans[skipIdx].end <= pos {
			skipIdx++
		}

		// A skip span starting before the next <cfscript> (or with no more
		// <cfscript> left at all) is handled first.
		if skipIdx < len(skipSpans) && (openIdx < 0 || skipSpans[skipIdx].start < openIdx) {
			nextSkip := &skipSpans[skipIdx]
			if nextSkip.start > pos {
				add(RegionTag, pos, nextSkip.start)
			}

			regions = append(regions, Region{Kind: RegionSkip, StartLine: idx.LineOf(nextSkip.start), Text: content[nextSkip.start:nextSkip.end], Offset: nextSkip.start})

			pos = nextSkip.end
			skipIdx++

			continue
		}

		if openIdx < 0 {
			break
		}

		if openIdx > pos {
			add(RegionTag, pos, openIdx)
		}

		bodyStart := openIdx + 10 // len("<cfscript>")
		closeIdx := cfscriptClose(content[bodyStart:])

		if closeIdx < 0 {
			add(RegionScript, bodyStart, len(content))

			pos = len(content)

			break
		}

		closeIdx += bodyStart

		add(RegionScript, bodyStart, closeIdx)

		pos = closeIdx + 11 // len("</cfscript>")
	}

	if pos < len(content) {
		add(RegionTag, pos, len(content))
	}

	return regions, idx
}

// nextCFScriptOpen is the offset of the next <cfscript> at or after pos that
// is not inside a CFML comment, or -1.
func nextCFScriptOpen(content string, pos int) int {
	for pos < len(content) {
		i := strings.IndexByte(content[pos:], '<')
		if i < 0 {
			return -1
		}

		i += pos

		if i+4 < len(content) && content[i:i+5] == "<!---" {
			pos = SkipCFMLComment(content, i)

			continue
		}

		if i+10 <= len(content) && strings.EqualFold(content[i:i+10], "<cfscript>") {
			return i
		}

		pos = i + 1
	}

	return -1
}

// indexCFTag finds "<" followed by suffix (case-insensitive) in s.
// Returns the index of '<' or -1.
//
// The walk between candidates is strings.IndexByte rather than a byte-at-a-time
// loop, and the suffix's first byte is folded by hand before EqualFold is called
// at all. Both matter because this is the search the whole tag scanner is built
// on — it was 6% of tag parsing in a profile — and because the ratio is brutal:
// a page is full of '<' and almost none of them start the tag being looked for,
// so the common case should be a vector scan and one byte compare, not a
// function call per angle bracket.
func indexCFTag(s, suffix string) int {
	if suffix == "" {
		return strings.IndexByte(s, '<')
	}

	first := lex.LowerASCII(suffix[0])

	for off := 0; off < len(s); {
		i := strings.IndexByte(s[off:], '<')
		if i < 0 {
			break
		}

		at := off + i
		off = at + 1

		if at+1+len(suffix) > len(s) {
			break
		}

		if lex.LowerASCII(s[at+1]) != first {
			continue
		}

		if strings.EqualFold(s[at+1:at+1+len(suffix)], suffix) {
			return at
		}
	}

	return -1
}

// cfscriptClose is the offset of the `</cfscript>` that ends a <cfscript> body
// which starts at the beginning of body, or -1 when there is none.
//
// It is indexCFTag's answer except in one case: a string in the script holds the
// closing tag (Lucee's tests write `var c = "<cfscript>a=1;</cfscript>"`), and
// the first closer found is inside it. That case has a mark an ordinary file
// lacks, either of two: a `<cfscript>` opener before the first closer, since the
// tag does not nest, or a second closer after it with no new opener between.
// Only then is the body walked with the script scanner, whose string token hides
// a closer inside it. A walk that finds no closer, an unclosed string, say,
// gives the first one back, as the scan always did.
func cfscriptClose(body string) int {
	first := indexCFTag(body, "/cfscript>")
	if first < 0 {
		return first
	}

	if indexCFTag(body[:first], "cfscript>") < 0 {
		rest := body[first+len("</cfscript>"):]

		second := indexCFTag(rest, "/cfscript>")
		if second < 0 || indexCFTag(rest[:second], "cfscript>") >= 0 {
			return first
		}
	}

	sc := NewScanner(body)
	sc.interpStrings = true

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == TokEOF {
			return first
		}

		if tok.Kind == TokLT {
			if at := tok.Offset; at+len("</cfscript>") <= len(body) && strings.EqualFold(body[at:at+len("</cfscript>")], "</cfscript>") {
				return at
			}
		}
	}
}

func findFuncScope(line int, scopes []FuncScope) FuncScope {
	for _, s := range scopes {
		if line >= s.Start && line <= s.End {
			return s
		}
	}

	return FuncScope{Start: -1, End: -1}
}

// scopeIndex answers, for many lines, which scope a scan of scopes in order
// would find first: the scope with Start < line < End, or with inclusive
// Start <= line <= End as findFuncScope tests, as its index, or -1. Asking
// findFuncScope or callerAtLine once per function or per call scanned every
// scope each time, which was quadratic in a file of thousands of functions.
// Past maxScopeScan scopes it maps every line once instead; next skips the
// lines already claimed, so nested scopes cost no more than the lines they
// cover.
func scopeIndex(scopes []FuncScope, inclusive bool) func(line int) int {
	lo, hi := 1, -1
	if inclusive {
		lo, hi = 0, 0
	}

	if len(scopes) <= maxScopeScan {
		return func(line int) int {
			for i := range scopes {
				if line >= scopes[i].Start+lo && line <= scopes[i].End+hi {
					return i
				}
			}

			return -1
		}
	}

	last := 0
	for i := range scopes {
		last = max(last, scopes[i].End+hi)
	}

	owner := make([]int32, last+1)
	next := make([]int32, last+2)

	for l := range next {
		next[l] = conv.Int32(l)
	}

	find := func(l int32) int32 {
		for next[l] != l {
			next[l] = next[next[l]]
			l = next[l]
		}

		return l
	}

	for i := range scopes {
		from, to := max(scopes[i].Start+lo, 0), scopes[i].End+hi
		if from > to {
			continue
		}

		for l := find(conv.Int32(from)); int(l) <= to; l = find(l) {
			owner[l] = conv.Int32(i + 1)
			next[l] = l + 1
		}
	}

	return func(line int) int {
		if line < 0 || line >= len(owner) {
			return -1
		}

		return int(owner[line]) - 1
	}
}

// maxScopeScan is the most scopes scopeIndex scans for each line before it
// maps every line to its scope instead.
const maxScopeScan = 32

// FindFuncScopeAt returns the FuncScope containing the given line, or {-1,-1}.
func FindFuncScopeAt(line int, scopes []FuncScope) FuncScope {
	return findFuncScope(line, scopes)
}

// RefScope is the store a receiver names, when it names one: `this.x` is
// read from this scope and `variables.x` from variables scope, which are
// separate. Anything else — a bare `x`, a deeper path — is RefAny, and sees
// refs made either way.
type RefScope uint8

// The scopes a receiver can name.
const (
	RefAny RefScope = iota
	RefThis
	RefVariables
)

// ReceiverRefScope is the scope a receiver written as `this.x` or
// `variables.x` is read from.
func ReceiverRefScope(recv string) RefScope {
	scope, name, ok := strings.Cut(recv, ".")
	if !ok || name == "" || strings.ContainsAny(name, ".[") {
		return RefAny
	}

	switch {
	case strings.EqualFold(scope, "this"):
		return RefThis
	case strings.EqualFold(scope, "variables"):
		return RefVariables
	default:
		return RefAny
	}
}

// Admits reports whether ref can answer a receiver read from s.
func (s RefScope) Admits(ref *ComponentRef) bool {
	switch s {
	case RefThis:
		return ref.This
	case RefVariables:
		return !ref.This
	default:
		return true
	}
}

// StripReceiverScope strips a receiver expression down to the text after its last
// top-level "." (e.g. "VARIABLES.donorObj" -> "donorObj"), matching how a ComponentRef
// is stored without its scope prefix. "Top-level" skips any "." inside a "[...]"
// subscript: for "linkMap[arguments.startSource]", the only "." is inside the brackets
// (separating "arguments" from "startSource", not a scope prefix on "linkMap"), so it
// must NOT be stripped there — a naive strings.LastIndexByte(s, '.') would cut the
// receiver down to the nonsensical fragment "startSource]", losing the "linkMap["
// prefix entirely. Returns s unchanged if it has no top-level ".".
func StripReceiverScope(s string) string {
	depth := 0
	last := -1

	for i, c := range s {
		switch c {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		case '.':
			if depth == 0 {
				last = i
			}
		}
	}

	if last < 0 {
		return s
	}

	return s[last+1:]
}

func uriFromString(s string) uri.URI {
	return uri.URI(s)
}
