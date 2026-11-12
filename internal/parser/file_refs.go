// File references: the paths a file names and the custom tags it calls,
// recorded by the parse's walk as it reads each region.

package parser

import (
	"cmp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"go.lsp.dev/uri"
)

// Includes, document links and custom tag calls were each read by a walk of
// their own over the parse's regions: a script region's tokens were scanned
// once for its custom tags and again for its includes, after the parse had
// scanned them, and a document's links scanned them twice more. The parse's
// walk now records them as it goes (fileRefRecord):
//
//   - In a script region, the region's scanner hands every token it produces
//     to a recorder (scriptRefs), which reads include statements, cfinclude()
//     and cfmodule() calls, Lucee's module statement, cf_name() and the CF
//     tags written among the statements from the tokens the walk reads
//     anyway. A token the walk reads twice (a lookahead it backs out of) is
//     recorded once, and what the walk leaves unread at the region's end is
//     scanned for the record alone.
//   - In a tag region, each tag the walk meets has its path attributes and
//     custom tag name read there (tagRefReader), with the tags the walk steps
//     over (a <cfscript> block, a #...# span) read as it catches up.
//
// Includes are recorded by every parse; links, custom tags and the names a
// configured prefix may make custom tags only with ExtractLinks, since a tag
// region's every tag then has its attributes read. ParseOptions.FileRefsOnly
// is the reading for a caller with no parse: the regions are read for their
// references and nothing else.

// fileRef is one path the walk met, at content offsets: the offset its
// statement or tag starts at, and the path's bounds, quotes excluded (a
// listing's are its tag's).
type fileRef struct {
	at, start, end int
	// region is the offset of the region the walk met it in.
	region int
	kind   fileRefKind
	// module marks a cfmodule template the script reading of custom tags
	// found; a script region's links list those first, as the walk that
	// read them first did.
	module bool
}

// tagNameSpan is a tag's name in a tag region that is not otherwise a custom
// tag: a configured prefix (CustomTagPrefixes) may make it one.
type tagNameSpan struct {
	start, end int32
	closing    bool
}

// fileRefRecord is what the walk recorded of the file references in of.
type fileRefRecord struct {
	of string
	// full is set when links, custom tags and tag names were read, not only
	// what includes need.
	full  bool
	refs  []fileRef
	tags  []CustomTagCall // Line unset
	names []tagNameSpan
	// imports is the cfimport taglibs by lowercased prefix the tag regions
	// declared so far, while the walk records: a prefix applies from its
	// cfimport on.
	imports map[string]string
}

// add records one path.
func (rec *fileRefRecord) add(kind fileRefKind, at, start, end, region int, module bool) {
	rec.refs = append(rec.refs, fileRef{at: at, start: start, end: end, region: region, kind: kind, module: module})
}

// scriptRefTriggers reports whether a script region of text can hold a
// reference the record keeps: every include spells "include", a listing
// "<cfdirectory", a custom tag call "cf_" or "module", and a tag among the
// statements a '<'. A region without one is not watched.
func scriptRefTriggers(text string, full bool) bool {
	if IndexFold(text, "include") >= 0 {
		return true
	}

	if !full {
		return IndexFold(text, "<cfdirectory") >= 0
	}

	return strings.IndexByte(text, '<') >= 0 || IndexFold(text, "cf_") >= 0 || IndexFold(text, "module") >= 0
}

// The include statement the recorder is part way through, by what it has
// read since the keyword.
const (
	incNone     = iota
	incKeyword  // include or cfinclude
	incParen    // and the '(' of cfinclude(…)
	incTemplate // and `template`
	incOp       // and the = or : after it
)

// The custom tag call the recorder is part way through.
const (
	tagNone        = iota
	tagUnderscore  // cf_name, a call when a '(' follows
	tagModuleCall  // cfmodule, a call when a '(' follows
	tagModuleStmt  // module first in a statement, a call when a name follows
	tagModuleAttrs // a cfmodule's attributes
)

// scriptRefs records the references in a script region from the tokens its
// scanner produces (Scanner.NextSkipComments hands each one over once, in
// order). It reads what the old walks read with a scanner of their own and a
// token of lookahead: here the lookahead is the next token handed over.
type scriptRefs struct {
	src  string // the region's text
	base int    // its offset in the content
	rec  *fileRefRecord
	full bool
	last int // the offset of the last token recorded

	// The include reading: the token before the one read, where it is in an
	// include statement, the keyword's offset, and a '<' whose next token
	// says whether a tag opens there (-1 for none).
	prev  Token
	stage uint8
	incAt int
	ltAt  int

	// The custom tag reading, with its own previous token: a cf_name token and
	// the one before it, and a cfmodule's attributes.
	tprev   Token
	tstage  uint8
	tTok    Token
	tBefore Token
	mod     moduleRead
}

// moduleRead is the attributes of a script cfmodule read so far.
type moduleRead struct {
	template, name         CustomTagCall
	pending                CustomTagCall // a value whose next token says whether it is computed
	key                    string
	depth                  int
	stop                   TokenKind
	hasTemplate, hasName   bool
	hasPending, isTemplate bool
}

// newScriptRefs is a recorder for the script region text at base.
func newScriptRefs(text string, base int, rec *fileRefRecord) *scriptRefs {
	return &scriptRefs{
		src: text, base: base, rec: rec, full: rec.full, last: -1,
		prev: Token{Kind: TokSemicolon}, tprev: Token{Kind: TokSemicolon}, ltAt: -1,
	}
}

// see records t, the next token the region's scanner produced past any
// comment. A token at or before the last one recorded is the walk reading
// again after a lookahead, and is not recorded twice.
func (w *scriptRefs) see(t Token) {
	if t.Offset <= w.last {
		return
	}

	w.last = t.Offset

	w.seeInclude(t)

	if w.full {
		w.seeCustomTag(t)
	}
}

// seeInclude reads the include statements: `include "x"`, Lucee's `include
// template="x"`, the call `cfinclude(template="x")`, and a CF tag written
// among the statements, read as the tag it is.
func (w *scriptRefs) seeInclude(t Token) {
	if w.ltAt >= 0 {
		at := w.ltAt
		w.ltAt = -1

		if t.Kind == TokIdent && HasPrefixFold(t.Value, "cf") {
			if gt := tagStartOutsideQuotesEnd(w.src[at:]); gt >= 0 {
				r := tagRefReader{text: w.src[at : at+gt+1], base: w.base + at, region: w.base, rec: w.rec, full: w.full}
				ends := newTagEnds(r.text)
				r.ends = &ends
				r.reach(len(r.text))
			}
		}
	}

	switch w.stage {
	case incKeyword:
		if t.Kind == TokLParen {
			w.stage = incParen

			return
		}

		fallthrough
	case incParen:
		if t.Kind == TokIdent && identEq(t.Value, "template") {
			w.stage = incTemplate

			return
		}

		if w.includePath(t) {
			return
		}
	case incTemplate:
		w.stage = incNone

		if t.Kind == TokEquals || t.Kind == TokColon {
			w.stage = incOp

			return
		}
	case incOp:
		if w.includePath(t) {
			return
		}
	}

	w.statement(t)
}

// includePath reads t as the path an include statement gives, when it is a
// string.
func (w *scriptRefs) includePath(t Token) bool {
	w.stage = incNone

	if t.Kind != TokString || len(t.Value) < 2 {
		return false
	}

	w.prev = t
	w.rec.add(refInclude, w.base+w.incAt, w.base+t.Offset+1, w.base+t.Offset+len(t.Value)-1, w.base, false)

	return true
}

// statement reads t where no include is open. After a '<' the keyword is the
// tag's name, read with the tag; after a '.' it is a member call,
// arr.include(x).
func (w *scriptRefs) statement(t Token) {
	before := w.prev
	w.prev = t

	if t.Kind == TokLT {
		w.ltAt = t.Offset

		return
	}

	if t.Kind != TokIdent || len(t.Value) != len("include") && len(t.Value) != len("cfinclude") ||
		before.Kind == TokDot || before.Kind == TokLT || !identEq(t.Value, "include") && !identEq(t.Value, "cfinclude") {
		return
	}

	w.stage, w.incAt = incKeyword, t.Offset
}

// seeCustomTag reads the custom tag calls: cf_name(...), cfmodule(...) and
// Lucee's module statement.
func (w *scriptRefs) seeCustomTag(t Token) {
	switch w.tstage {
	case tagUnderscore:
		w.tstage = tagNone

		// `function cf_name()` declares a function, and calls nothing.
		if t.Kind == TokLParen && (w.tBefore.Kind != TokIdent || !strings.EqualFold(w.tBefore.Value, "function")) {
			n := w.tTok
			w.rec.tags = append(w.rec.tags, CustomTagCall{Kind: CustomTagUnderscore, Name: n.Value[3:], Start: w.base + n.Offset + 3, End: w.base + n.Offset + len(n.Value), Script: true})
		}
	case tagModuleCall:
		w.tstage = tagNone

		if t.Kind == TokLParen {
			w.startModule(TokRParen, t)

			return
		}
	case tagModuleStmt:
		w.tstage = tagNone

		if t.Kind == TokIdent {
			w.startModule(TokSemicolon, t)

			return
		}
	case tagModuleAttrs:
		w.moduleToken(t)

		return
	}

	before := w.tprev
	w.tprev = t

	// Every word this reads starts with a c or an m.
	if t.Kind != TokIdent || before.Kind == TokDot || len(t.Value) < 4 {
		return
	}

	if c := t.Value[0] | 0x20; c != 'c' && c != 'm' {
		return
	}

	switch {
	case HasPrefixFold(t.Value, "cf_"):
		w.tstage, w.tTok, w.tBefore = tagUnderscore, t, before
	case strings.EqualFold(t.Value, "cfmodule"):
		w.tstage = tagModuleCall
	case strings.EqualFold(t.Value, "module") && statementStart(before):
		w.tstage = tagModuleStmt
	}
}

// startModule begins a cfmodule's attributes at t, up to stop at depth zero.
func (w *scriptRefs) startModule(stop TokenKind, t Token) {
	w.mod = moduleRead{stop: stop}
	w.tstage = tagModuleAttrs
	w.moduleToken(t)
}

// moduleToken reads t among a script cfmodule's attributes: the name =
// "value" pairs up to the stop token at depth zero (or a { for the statement
// form's body). A key inside a nested struct or call is not one of the tag's;
// a value that is more than its string literal ("a" & b) is computed, and the
// call Dynamic.
func (w *scriptRefs) moduleToken(t Token) {
	m := &w.mod

	if m.hasPending {
		c := m.pending
		c.Dynamic = c.Dynamic || !endsValue(t.Kind)
		m.hasPending = false

		if m.isTemplate {
			m.template, m.hasTemplate = c, true
		} else {
			c.Kind = CustomTagModuleName
			m.name, m.hasName = c, true
		}
	}

	if t.Kind == TokEOF || t.Kind == TokLBrace && m.depth == 0 {
		w.endModule()

		return
	}

	switch t.Kind {
	case TokLParen, TokLBracket, TokLBrace:
		m.depth++
	case TokRParen, TokRBracket, TokRBrace:
		m.depth--
	default:
	}

	if m.depth <= 0 && (t.Kind == m.stop || t.Kind == TokRParen || t.Kind == TokRBrace) {
		w.endModule()

		return
	}

	switch {
	case t.Kind == TokIdent && m.depth <= 1:
		m.key = strings.ToLower(t.Value)
	case t.Kind == TokString && m.depth <= 1 && (m.key == "template" || m.key == "name") && len(t.Value) >= 2:
		value := t.Value[1 : len(t.Value)-1]
		m.pending = CustomTagCall{Kind: CustomTagModuleTemplate, Name: value, Start: w.base + t.Offset + 1, End: w.base + t.Offset + len(t.Value) - 1, Script: true, Dynamic: strings.Contains(value, "#")}
		m.hasPending, m.isTemplate = true, m.key == "template"
		m.key = ""
	case t.Kind != TokEquals && t.Kind != TokColon:
		m.key = ""
	}
}

// endModule records the call a cfmodule's attributes make: by template, else
// by name. A literal template is a link too.
func (w *scriptRefs) endModule() {
	m := &w.mod

	var (
		c  CustomTagCall
		ok bool
	)

	switch {
	case m.hasTemplate:
		c, ok = m.template, true
	case m.hasName:
		c, ok = m.name, true
	}

	if ok {
		w.rec.tags = append(w.rec.tags, c)

		if c.Kind == CustomTagModuleTemplate && !c.Dynamic {
			w.rec.add(refTemplate, c.Start, c.Start, c.End, w.base, true)
		}
	}

	w.tprev = Token{Kind: m.stop}
	w.mod = moduleRead{}
	w.tstage = tagNone
}

// tagRefReader reads the references in a tag region, a tag at a time. The
// region's walk calls reach at each tag it meets, and the reader reads every
// tag up to it, the ones the walk stepped over included, so what it records
// does not depend on where the walk went.
//
// In markup, a tag's attributes are read outside CFML comments, as a CF tag
// reads them or as HTML does by the tag (attrScanner): template links on a CF
// tag (an include on cfinclude), href on markup, action only on a form; a
// value must be quoted, and an expression tag (cfset, cfif, cfelseif,
// cfreturn) links nothing. A markup tag a CF tag interrupts (<a <cfif
// x>id="y"</cfif> href="z">) ends where the CF tag begins (tagEnd), and its
// attributes after the CF tags are read from the text between them, up to its
// own '>'.
type tagRefReader struct {
	text   string
	base   int // the text's offset in the content
	region int // the offset of the region it is in
	ends   *tagEnds
	rec    *fileRefRecord
	// full reads every path; otherwise only what an include needs,
	// <cfinclude> and <cfdirectory>: most tags name no path, and every parse
	// records includes. It skips work, not a rule.
	full bool
	// tags records the custom tag calls and the other tags' names.
	tags bool

	// pos is where the reading goes on; outer is the markup tag interrupted
	// by a CF tag, still open, and outerAt where it began ("" for none).
	pos     int
	outer   string
	outerAt int
}

// reach reads every tag opening at or before limit; limit len(text) reads
// the rest.
func (r *tagRefReader) reach(limit int) {
	text := r.text

	for r.pos < len(text) && r.pos <= limit {
		i := strings.IndexByte(text[r.pos:], '<')
		if i < 0 {
			i = len(text) - r.pos
		}

		i += r.pos

		if i > limit {
			return
		}

		// The text before the next '<' continues an interrupted tag's
		// attributes, up to its '>'.
		if r.outer != "" && i > r.pos {
			seg := text[r.pos:i]
			if gt := tagStartOutsideQuotesEnd(seg); gt >= 0 {
				r.attrs(seg[:gt], r.pos, r.outerAt, r.outer, false)
				r.outer = ""
			} else {
				r.attrs(seg, r.pos, r.outerAt, r.outer, false)
			}
		}

		if i >= len(text) {
			r.pos = len(text)

			return
		}

		r.pos = r.tagAt(i)
	}
}

// tagAt reads the tag at i, whose '<' it is, and returns where the reading
// goes on.
func (r *tagRefReader) tagAt(i int) int {
	text := r.text

	// A closing tag ends where its '>' does, or where a tag opens before
	// it (tagEnd): `</p` left unclosed before a <cfelse> ends there, and the
	// <cfelse> is read. A `</` with no name is a stray '<', and `</<cfelseif
	// x>` reads the <cfelseif>. Both were stepped over to the next '>', for
	// paths, and swallowed the tag after them.
	if i+1 < len(text) && text[i+1] == '/' {
		j := i + 2

		k := j
		for k < len(text) && isTagNameByte(text[k]) {
			k++
		}

		if k > j {
			end := tagEnd(r.ends, k)
			if r.tags {
				r.customTag(j, k, end, true)
			}

			return end
		}

		return i + 1
	}

	j, k, next := tagNameAt(text, i)
	if k == j {
		return next
	}

	end := tagEnd(r.ends, k)
	name := text[j:k]

	// A CF tag's values are read as CF reads them, a custom tag's included
	// (<cf_x>, <p:x>); any other tag's as markup's.
	cf := HasPrefixFold(name, "cf") || strings.IndexByte(name, ':') > 0

	switch {
	case strings.EqualFold(name, "cfdirectory"):
		r.rec.add(refListing, r.base+i, r.base+i, r.base+end, r.region, false)
	case !r.full:
		if strings.EqualFold(name, "cfinclude") {
			r.attrs(text[i:end], i, i, name, true)
		}
	default:
		if !cf || !expressionTag(name) {
			r.attrs(text[i:end], i, i, name, cf)
		}

		if !cf && (end >= len(text) || text[end-1] != '>') {
			r.outer, r.outerAt = name, i
		}
	}

	if r.tags {
		r.customTag(j, k, end, false)
	}

	return end
}

// attrs records the path attributes of s, a tag's text at base in text, the
// tag named name starting at at, a CF tag when cf.
func (r *tagRefReader) attrs(s string, base, at int, name string, cf bool) {
	mode := htmlAttrs
	if cf {
		mode = cfAttrs
	}

	sc := newAttrScanner(s, mode)

	for {
		a, more := sc.next()
		if !more {
			return
		}

		// A path is quoted; an unquoted value is an expression or a word.
		if a.valStart == 0 || !isQuoteByte(s[a.valStart-1]) {
			continue
		}

		if kind := attrRef(name, s[a.nameStart:a.nameEnd], cf); kind != 0 {
			r.rec.add(kind, r.base+at, r.base+base+a.valStart, r.base+base+a.valEnd, r.region, false)
		}
	}
}

// customTag records the custom tag a tag whose name is text[j:k] and whose
// attributes end at end calls, or its name for a configured prefix to read.
// A cfimport prefix applies from its cfimport on, as the engines compile it.
func (r *tagRefReader) customTag(j, k, end int, closing bool) {
	text, base, rec := r.text, r.base, r.rec
	name := text[j:k]

	switch {
	case len(name) > 3 && HasPrefixFold(name, "cf_"):
		rec.tags = append(rec.tags, CustomTagCall{Kind: CustomTagUnderscore, Name: name[3:], Start: base + j + 3, End: base + k, AttrStart: base + k, AttrEnd: base + end, Closing: closing})
	case strings.EqualFold(name, "cfmodule"):
		if !closing {
			rec.tags = moduleAttrs(text[k:end], base+k, rec.tags)
		}
	case strings.EqualFold(name, "cfimport"):
		if !closing {
			rec.imports = cfimportTaglib(text[k:end], rec.imports)
		}
	case strings.IndexByte(name, ':') > 0:
		prefix, local, _ := strings.Cut(name, ":")
		if taglib, ok := rec.imports[strings.ToLower(prefix)]; ok && local != "" {
			rec.tags = append(rec.tags, CustomTagCall{Kind: CustomTagImported, Name: local, Prefix: strings.ToLower(prefix), Taglib: taglib, Start: base + j + len(prefix) + 1, End: base + k, Closing: closing})
		}
	default:
		rec.names = append(rec.names, tagNameSpan{start: int32(base + j), end: int32(base + k), closing: closing}) //nolint:gosec // a file is at most vfs.MaxFileSize
	}
}

// recordingRefs reports whether this parse records file references: every
// parse but one reading only its functions' scopes, and a walk of one body
// only when it files that body's links (ExtractLinks), since the record of
// the file is the whole walk's.
func (pr *ParseResult) recordingRefs() bool {
	return !pr.scopesOnly && (pr.bodies.only == nil || pr.extractLinks)
}

// startRefs begins the record of the walk about to run.
func (pr *ParseResult) startRefs() {
	pr.recorded = nil
	if pr.recordingRefs() {
		pr.refsRec = &fileRefRecord{of: pr.Content, full: pr.extractLinks}
	}
}

// refsMark is where the record stands, for refsSince.
func (pr *ParseResult) refsMark() int {
	if pr.refsRec == nil {
		return 0
	}

	return len(pr.refsRec.refs)
}

// refsSince is the paths recorded since mark.
func (pr *ParseResult) refsSince(mark int) []fileRef {
	if pr.refsRec == nil {
		return nil
	}

	return pr.refsRec.refs[mark:]
}

// watchScript has the scanner of the script region r, about to be walked,
// hand its tokens to the record, when the region can hold a reference.
func (pr *ParseResult) watchScript(sc *Scanner, r *Region) *scriptRefs {
	rec := pr.refsRec
	if rec == nil || !scriptRefTriggers(r.Text, rec.full) {
		return nil
	}

	w := newScriptRefs(r.Text, r.Offset, rec)
	sc.watch(w)

	return w
}

// finishScript scans what the walk of a script region left unread, for the
// record alone: a walk that stops early leaves the rest of the region.
func finishScript(sc *Scanner, w *scriptRefs) {
	if w == nil {
		return
	}

	for t := sc.NextSkipComments(); t.Kind != TokEOF; t = sc.NextSkipComments() {
		_ = t // the scanner hands each token to w
	}

	sc.watch(nil)
}

// tagReader is the reference reader for the tag region r, whose walk shares
// its tag ends, or nil when the parse records none.
func (pr *ParseResult) tagReader(r *Region, ends *tagEnds) *tagRefReader {
	rec := pr.refsRec
	if rec == nil || r.Kind != RegionTag {
		return nil
	}

	return &tagRefReader{text: r.Text, base: r.Offset, region: r.Offset, ends: ends, rec: rec, full: rec.full, tags: rec.full}
}

// endRefs keeps the record the walk made. A file with no CF tag is one
// script region to ClassifyRegions, which is the parse's rule, and its walk
// recorded what a script reading finds; one that begins with markup has its
// paths read as the markup it is, as StructureSpans reads it, and keeps the
// custom tags its script reading found.
func (pr *ParseResult) endRefs() {
	rec := pr.refsRec
	pr.refsRec = nil

	if rec == nil {
		return
	}

	rec.imports = nil

	// A body's walk reads the references of one function, which are not
	// the file's.
	if pr.bodies.only != nil {
		return
	}

	if markupScript(pr.Content, pr.Regions) {
		rec.refs = rec.refs[:0]
		r := tagRefReader{text: pr.Content, rec: rec, full: rec.full}
		ends := newTagEnds(pr.Content)
		r.ends = &ends
		r.reach(len(pr.Content))
	}

	pr.mu.Lock()
	pr.recorded = rec
	pr.mu.Unlock()
}

// markupScript reports whether regions, split from content, are one script
// region holding all of it that begins with markup (fileRegions).
func markupScript(content string, regions []Region) bool {
	return len(regions) == 1 && regions[0].Kind == RegionScript && len(regions[0].Text) == len(content) && startsWithMarkup(content)
}

// refsOnlyRegions is the walk of a FileRefsOnly parse: each script region's
// tokens are scanned for the record and nothing else, and each tag region's
// tags read by the reference reader alone.
func (pr *ParseResult) refsOnlyRegions() {
	regions := pr.Regions
	if body := pr.bodies.only; body != nil {
		regions = pr.bodyRegions(body)
	}

	for i := range regions {
		r := &regions[i]

		switch r.Kind {
		case RegionScript:
			sc := NewScriptScanner(r.Text)
			finishScript(sc, pr.watchScript(sc, r))
		case RegionTag:
			ends := newTagEnds(r.Text)
			if rd := pr.tagReader(r, &ends); rd != nil {
				rd.reach(len(r.Text))
			}
		case RegionSkip:
		}
	}
}

// refRecord is the record of the file references in Content, with links and
// custom tags when full: the parse's own when it read them from this
// content, otherwise read now (readRefs) and kept.
func (pr *ParseResult) refRecord(full bool) *fileRefRecord {
	content := pr.Content

	pr.mu.Lock()
	rec := pr.recorded
	pr.mu.Unlock()

	if rec != nil && rec.of == content && (rec.full || !full) {
		return rec
	}

	rec = readRefs(pr.URI, content, full, pr.interpolateAll)

	pr.mu.Lock()
	if pr.Content == content {
		pr.recorded = rec
	}
	pr.mu.Unlock()

	return rec
}

// readRefs is the record of content's file references, from a parse that
// reads nothing else (FileRefsOnly), its regions split as a parse of fileURI
// splits them.
func readRefs(fileURI uri.URI, content string, full, interpolateAll bool) *fileRefRecord {
	walk := ParseWithOptions(fileURI, content, &ParseOptions{FileRefsOnly: true, ExtractLinks: full, InterpolateAllText: interpolateAll})

	return walk.recorded
}

// links is the record's document links, lines and columns counted in
// content by lines (nil to build them).
func (rec *fileRefRecord) links(content string, lines LineTable) []DocumentLink {
	return linksOf(rec.refs, content, 0, 0, lines)
}

// linksOf is the links among refs, read from text at base in the content,
// with lines counted from baseLine and columns from the start of each line
// of text (by lines, nil to build them), in the order they are listed: a
// region's cfmodule templates first, then the rest as the walk met them. A
// region's walk files its own (ExtractLinks) by function.
func linksOf(refs []fileRef, text string, base, baseLine int, lines LineTable) []DocumentLink {
	var links []DocumentLink

	for from := 0; from < len(refs); {
		to := from + 1
		for to < len(refs) && refs[to].region == refs[from].region {
			to++
		}

		for _, module := range [2]bool{true, false} {
			for i := from; i < to; i++ {
				ref := &refs[i]
				start, end := ref.start-base, ref.end-base

				if ref.module != module || !isLink(ref.kind, text[start:end]) {
					continue
				}

				if lines == nil {
					lines = NewLineTable(text)
				}

				line := lines.LineOf(start)
				lineStart := int(lines[line])

				links = append(links, DocumentLink{
					Path:  text[start:end],
					Line:  conv.Uint32(baseLine + line),
					Start: conv.Uint32(start - lineStart),
					End:   conv.Uint32(end - lineStart),
				})
			}
		}

		from = to
	}

	return links
}

// customTagCalls is the record's custom tag calls for the configured
// prefixes (sorted longest first), with each tag name a prefix makes one, in
// the order the walk met them.
func (rec *fileRefRecord) customTagCalls(content string, lines LineTable, prefixes []string) []CustomTagCall {
	// The longest prefix decides: "cfa" before "cf".
	prefixes = slices.SortedFunc(slices.Values(prefixes), func(a, b string) int { return cmp.Compare(len(b), len(a)) })

	out := make([]CustomTagCall, 0, len(rec.tags))
	tags := rec.tags

	for _, n := range rec.names {
		name := content[n.start:n.end]

		for _, p := range prefixes {
			if len(name) > len(p) && HasPrefixFold(name, p) {
				for len(tags) > 0 && tags[0].Start < int(n.start) {
					out = append(out, tags[0])
					tags = tags[1:]
				}

				out = append(out, CustomTagCall{Kind: CustomTagPrefixed, Name: name[len(p):], Prefix: p, Start: int(n.start) + len(p), End: int(n.end), Closing: n.closing})

				break
			}
		}
	}

	out = append(out, tags...)

	if len(out) > 0 {
		if lines == nil {
			lines = NewLineTable(content)
		}

		for i := range out {
			out[i].Line = lines.LineOf(out[i].Start)
		}
	}

	return out
}
