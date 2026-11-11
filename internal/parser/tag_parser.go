package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
)

// tagParser extracts definitions from CFML tag-based source.
type tagParser struct {
	src            string
	fileURI        string
	funcs          []FunctionDef
	vars           []VarDef
	componentRefs  []ComponentRef
	funcRefs       map[string][]ComponentRef // keyed by "start:end"
	funcLinks      map[string][]DocumentLink // keyed by "start:end"
	funcCalls      map[string][]CallSite     // keyed by "start:end"
	links          []DocumentLink            // global scope links
	calls          []CallSite                // global scope call sites
	instantiations []Instantiation           // what its expressions' sub-parses found built
	exprParser     *scriptParser             // the one sub-parser every expression is read with; see subParser
	scopes         []FuncScope
	pendingCalls   []pendingCall
	// pendingByName indexes pendingCalls by lowercased varName, for
	// assignedSince; pendingNamed is how many it has indexed.
	pendingByName map[string][]int
	pendingNamed  int
	properties    []propertyDef
	extends       string
	accessors     bool
	persistent    bool
	lineIndex     LineTable // byte offset of each line start
	resolvers     []Resolver
	resolverSet   *ResolverSet
	extractLinks  bool // whether to extract document links
	extractCalls  bool // whether to extract all call sites
	// callNames, lowercased, are the only names whose calls are recorded
	// (call_names.go); nil records every call.
	callNames []string
	// gated limits #...# scanning of text to where ColdFusion evaluates it,
	// which output tracks as the walk passes each tag; srcOffset is this
	// region's offset in the file. The attributes of CF tags and of
	// importPrefixes custom tags are scanned wherever they stand. See
	// outputContext and outputState.
	gated      bool
	closeKnown bool
	endsReady  bool  // ends is built over p.src
	chainDepth uint8 // chained assignments being parsed; see maxChainedAssignments
	// rhsLead is the line breaks between the <cfset> tag's line and the first
	// character of its value, which trimming the value drops: a call that opens
	// the value is on that many lines below the tag.
	rhsLead int
	output  *outputState
	// collectionTags are the offsets of the tags the walk met that can hold a
	// collection write (collectionTagName), for collectWrites.
	collectionTags []int
	// closeAt is where the first </cffunction> at or after offset closeFrom
	// is, -1 for none, once closeKnown; see functionCloseAfter.
	closeFrom, closeAt int
	// tagCalls is how many calls the current bucket held when the tag being
	// handled was reached, so recordedForExpression reads only that tag's.
	tagCalls            int
	ends                tagEnds
	importPrefixes      []string
	srcOffset           int
	builtinReturnLookup func(string) string
	argumentTypes       func(string, string, []Argument)
	inFunc              string // current function scope key ("start:end"), empty if global
	// localVars holds the var'd/local. names declared in the function being
	// parsed. A slice scanned with EqualFold rather than a map of lowercased
	// keys: a function declares a handful of locals, so the scan is short, and
	// the map cost both its buckets and a strings.ToLower per insert and per
	// lookup — together 7% of everything a tag parse allocated.
	//
	// localSet indexes localVars, lowercased, once a function declares more
	// than maxLocalScan of them: a scan per declaration made a function of
	// thousands of <cfset var> or <cfargument> quadratic.
	localVars   []string
	localSet    map[string]struct{}
	flow        *flowBlocks // the blocks a function's tags open; nil outside a full parse
	forceGlobal bool        // when true, addRef routes to componentRefs regardless of inFunc
	refThis     bool        // the assignment being parsed is through `this.` — see ComponentRef.This

	// baseLine is this region's absolute start line (0 if parsing the whole
	// file as one region). knownScopes, when set by the caller, gives the
	// true absolute (Start, End) of every tag function in the WHOLE file —
	// computed once up front via findTagFuncScopesIdx(pr.Content, 0, idx), which is
	// unaffected by region splitting. Used so a function whose body is
	// interrupted by a nested <cfscript> block (splitting the file into
	// separate regions here) still resolves its real end line, instead of
	// only whatever "/cffunction" this region's own limited text happens to
	// contain.
	baseLine    int
	knownScopes []FuncScope
}

func newTagParser(src, fileURI string) *tagParser {
	p := &tagParser{src: src, fileURI: fileURI}
	p.buildLineIndex()

	return p
}

func (p *tagParser) buildLineIndex() {
	p.lineIndex = NewLineTable(p.src)
}

// lineAt returns the 0-based line number for a byte offset.
func (p *tagParser) lineAt(offset int) int {
	return p.lineIndex.LineOf(offset)
}

// parse scans through tag-based CFML extracting definitions.
func (p *tagParser) parse() {
	pos := 0

	for pos < len(p.src) {
		// Find next < that could be a CF tag
		idx := p.nextTag(pos)
		if idx < 0 {
			// Trailing text, which is also where the attribute list of the last
			// declined tag in the file ends up.
			if p.textEvaluated() {
				p.scanInterpolatedText(p.src[pos:], pos)
			}

			break
		}

		idx += pos

		// Everything between the previous tag and this one is text, and in a
		// tag file that is where #...# interpolation lives: a <cfoutput> body,
		// and the attribute list of any tag the fast skip below declines (which
		// becomes text before the *next* tag). Scanning it here rather than
		// after the walk is what keeps p.inFunc live, so a call inside a
		// function's <cfoutput> is filed against that function.
		//
		// Gated, it is scanned only where ColdFusion evaluates text. Output
		// contexts open and close at tags, so a gap lies wholly inside one or
		// wholly outside; and the attributes of a declined CF or custom tag
		// are scanned below and stepped over, so a gap holds only text and
		// HTML attributes, which are evaluated only inside an output context.
		if p.textEvaluated() {
			p.scanInterpolatedText(p.src[pos:idx], pos)
		}

		// Skip CFML comments (nested)
		if idx+4 < len(p.src) && p.src[idx:idx+5] == "<!---" {
			pos = SkipCFMLComment(p.src, idx)

			continue
		}

		mark := p.flowMark()

		// Check for cfscript block
		if idx+10 <= len(p.src) && strings.EqualFold(p.src[idx:idx+10], "<cfscript>") {
			pos = p.parseNestedScript(idx)
			p.stampFlow(mark)

			continue
		}

		// Check for CF tags we care about
		switch {
		case idx+1 < len(p.src) && p.src[idx+1] == '/':
			p.trackBlock(idx)
			p.trackOutput(idx)
			p.closeTag(idx)

			pos = idx + 1
		case idx+3 < len(p.src) && lex.LowerASCII(p.src[idx+1]) == 'c' && lex.LowerASCII(p.src[idx+2]) == 'f':
			p.trackBlock(idx)
			p.trackOutput(idx)

			if collectionTagName(extractIdent(p.src[idx+1:])) {
				p.collectionTags = append(p.collectionTags, idx)
			}

			pos = p.handleCFTag(idx)
		case p.gated && customPrefixTag(p.src[idx:], p.importPrefixes):
			pos = p.stepOverEvaluatedTag(idx)
		default:
			pos = idx + 1
		}

		p.stampFlow(mark)
	}

	if p.extractLinks {
		p.extractAllLinks()
	}
}

// parseNestedScript parses a <cfscript> block opening at idx in a tag file and
// merges what it found, returning the offset just past its close.
func (p *tagParser) parseNestedScript(idx int) int {
	bodyStart := idx + 10
	closeIdx := cfscriptClose(p.src[bodyStart:])

	var bodyEnd int

	if closeIdx < 0 {
		bodyEnd = len(p.src)
	} else {
		bodyEnd = bodyStart + closeIdx
	}

	baseLine := p.lineAt(bodyStart)
	sp := newScriptParser(p.src[bodyStart:bodyEnd], p.fileURI, baseLine, p.resolvers).asCFScript()
	sp.baseOffset = p.srcOffset + bodyStart
	sp.resolverSet = p.resolverSet
	sp.extractCalls = p.extractCalls
	sp.setCallNames(p.callNames)
	sp.parse()
	p.funcs = append(p.funcs, sp.funcs...)
	p.vars = append(p.vars, sp.vars...)

	// Merge calls from cfscript sub-parser
	if p.extractCalls {
		for i := range sp.calls {
			c := &sp.calls[i]

			p.addCall(c)
		}

		for _, calls := range sp.funcCalls {
			for i := range calls {
				c := &calls[i]

				p.addCall(c)
			}
		}
	}

	if p.inFunc != "" {
		for i := range sp.componentRefs {
			ref := &sp.componentRefs[i]

			p.addRef(ref)
		}
	} else {
		p.componentRefs = append(p.componentRefs, sp.componentRefs...)
	}

	// Merge pending calls (unresolved "x = y.method()" assignments) from
	// the cfscript sub-parser. Without this, chained factory calls inside
	// a nested <cfscript> block (e.g. "conn = uri.openConnection();")
	// never reach resolvePendingCalls, so they can never fall back to
	// baseVar's own component (or FuncLookup's declared return type).
	for i := range sp.pendingCalls {
		c := sp.pendingCalls[i] // a copy: edited below, and kept

		if c.funcKey == "" {
			c.funcKey = p.inFunc
		}

		p.pendingCalls = append(p.pendingCalls, c)
	}

	if closeIdx < 0 {
		return len(p.src)
	}

	return bodyEnd + 11 // len("</cfscript>")
}

// closeTag handles the close tag at idx. </cffunction> ends the function
// scope, so what follows is top-level code again.
//
// This was dead code three times over. "</cffunction>" is exactly 13 bytes
// so `len(tag) > 13` was false, and tag[2:13] is "cffunction>", 11 bytes,
// which never equals the 10-byte "cffunction". Once that was fixed the check
// sat in handleCFTag, which the walk reaches only for a tag whose second byte
// is 'c' — never a close tag's '/'. The scope was never exited, and every
// top-level <cfset> after a function was that function's: its pending calls
// typed a variable of the function, its unscoped names were read against the
// function's var'd locals, and a call the tag parser records itself named
// the function as its caller.
// blockKind is the block a tag named name opens or closes, "" for any other
// tag: the tags whose body may not run, or may run more than once, each the
// opener a close tag of its name is matched to. <cfelseif> and <cfelse> end
// the branch before them and open the next, as a branch of the same <cfif>.
// A <cffunction> body is a block too, the one its statements always run in.
// A <cflock>, <cftransaction> or <cfoutput> runs its body once, so it is not
// one. None of these is written self-closed.
func blockKind(name []byte) string {
	switch string(name) {
	case "cffunction":
		return "cffunction"
	case "cfif", "cfelseif", "cfelse":
		return "cfif"
	case "cfloop":
		return "cfloop"
	case "cftry":
		return "cftry"
	case "cfcatch":
		return "cfcatch"
	case "cfcase":
		return "cfcase"
	case "cfdefaultcase":
		return "cfdefaultcase"
	default:
		return ""
	}
}

// trackBlock opens or closes the block the CF tag at idx opens or closes,
// when blocks are being recorded.
func (p *tagParser) trackBlock(idx int) {
	if p.flow == nil {
		return
	}

	closing := idx+1 < len(p.src) && p.src[idx+1] == '/'

	start := idx + 1
	if closing {
		start++
	}

	// Every block tag's name continues past "cf" with one of these.
	if start+2 >= len(p.src) {
		return
	}

	switch lex.LowerASCII(p.src[start+2]) {
	case 'f', 'i', 'e', 'l', 't', 'c', 'd':
	default:
		return
	}

	end := start
	for end < len(p.src) && lex.IsIdent(p.src[end]) {
		end++
	}

	var buf foldScratch

	name := buf.lowerFold(p.src[start:end])

	kind := blockKind(name)
	if kind == "" {
		return
	}

	if !closing && kind == "cfif" && string(name) != "cfif" {
		p.flow.close(kind)
	} else if closing {
		p.flow.close(kind)

		return
	}

	p.flow.open(idx, kind)
}

// flowMark and stampFlow are the script parser's, for a tag: what a tag
// assigns was assigned in the innermost block open at it.
func (p *tagParser) flowMark() int { return len(p.pendingCalls) }

func (p *tagParser) stampFlow(mark int) {
	stampPending(p.flow, p.pendingCalls[mark:])
}

func (p *tagParser) closeTag(idx int) {
	if !isCloseTagFor(p.src[idx:], "cffunction") {
		return
	}

	p.inFunc = ""
	p.localVars = p.localVars[:0]
	p.localSet = nil
	p.forceGlobal = false
}

// handleCFTag handles the CF tag opening at idx, returning where the walk
// continues.
func (p *tagParser) handleCFTag(idx int) int {
	// Fast skip on the fourth byte: only tags starting cfc/cff/cfl/cfp/cfs/
	// cfo/cfi/cfr/cfe reach the switch below (cfcomponent, cffunction,
	// cfloop, cfparam/cfproperty, cfset, cfobject, cfif/cfinvoke,
	// cfreturn, cfelseif).
	//
	// 'l' and 'e' are the two admitted for a single tag each, and they are
	// the ones to re-measure if this loop ever shows up in a profile.
	// <cfloop index="x"> declares x, so go-to-definition on a loop index
	// has nowhere to land without 'l'; the cost is that every <cflock>,
	// <cflog>, <cflocation> and <cfldap> now pays an IndexByte for its '>'
	// and a lineAt, and <cfloop> is among the most common tags in CFML.
	// 'e' is admitted for <cfelseif>, whose condition holds calls like any
	// other expression, and charges the same to <cfelse>, <cfexit> and
	// <cferror>. Both measured within noise on the parser benchmarks,
	// because that work is a byte scan over a tag that was going to be
	// scanned past anyway.
	ch := lex.LowerASCII(p.src[idx+3])
	if ch != 'c' && ch != 'f' && ch != 'l' && ch != 'p' && ch != 's' && ch != 'o' && ch != 'i' && ch != 'r' && ch != 'e' && ch != '/' {
		return p.stepOverEvaluatedTag(idx)
	}

	tagEnd := p.tagEnds().at(idx)
	if tagEnd < 0 {
		return idx + 1
	}

	tagEnd += idx + 1 // past the >
	tag := p.src[idx:tagEnd]
	line := p.lineAt(idx)

	p.tagCalls = len(p.calls)
	if p.inFunc != "" {
		p.tagCalls = len(p.funcCalls[p.inFunc])
	}

	// A handled tag is stepped over whole, so its own attributes are in
	// no text gap: `<cfloop array="#svc.list()#">` is scanned here. A tag that
	// is only an expression is parsed whole by its handler, spans in its strings
	// included, so scanning the spans too recorded each call twice.
	if !holdsOnlyAnExpression(tag) {
		p.scanInterpolatedText(tag, idx)
	}

	p.dispatchCFTag(ch, tag, idx, tagEnd, line)

	return tagEnd
}

// Dispatch on the fourth byte the fast skip already read, rather than
// testing each prefix in turn. The chain this replaces ran up to nine
// EqualFold calls for every tag that got this far; the byte narrows it
// to one or two, which is what pays for <cfloop> now being admitted to
// the scan at all.
//
// The letters here must stay in step with the fast skip above: a tag
// admitted there with no case here is scanned and then dropped, and a
// case here whose letter the skip rejects is unreachable.
// TestTagDispatchMatchesTheFastSkip fails on either.
func (p *tagParser) dispatchCFTag(ch byte, tag string, idx, tagEnd, line int) {
	switch ch {
	case 'c':
		if hasCFTagPrefix(tag, "<cfcomponent") {
			p.extends = getAttr(tag, "extends")

			p.accessors = isTruthy(getAttr(tag, "accessors"))
			if isTruthy(getAttr(tag, "persistent")) {
				p.persistent = true
			}
		}
	case 'f':
		if hasCFTagPrefix(tag, "<cffunction") {
			p.parseCFFunction(tag, idx, tagEnd, line)
			p.enterFunctionScope(tagEnd, line)
		}
	case 'l':
		if hasCFTagPrefix(tag, "<cfloop") {
			p.parseScopedAttrVar(tag, "index", line)
		}
	case 'p':
		switch {
		case hasCFTagPrefix(tag, "<cfproperty"):
			p.parseCFProperty(tag, line)
		case hasCFTagPrefix(tag, "<cfparam"):
			p.parseScopedAttrVar(tag, "name", line)
		}
	case 's':
		if hasCFTagPrefix(tag, "<cfset") {
			p.parseCFSet(tag, line, idx)
		}
	case 'o':
		if hasCFTagPrefix(tag, "<cfobject") {
			p.parseCFObject(tag, line)
		}
	case 'i':
		switch {
		case hasCFTagPrefix(tag, "<cfinvoke"):
			p.parseCFInvoke(tag, line)
		case hasCFTagPrefix(tag, "<cfif"):
			// A condition is an expression, and `<cfif svc.isValid(x)>`
			// recorded nothing while the same test in script syntax
			// recorded the call.
			body := tagBody(tag)
			p.scanTagExpression(body, bodyLine(tag, body, line), idx+strings.Index(tag, body))
		}
	case 'e':
		// 'e' is admitted for this one tag, the way 'l' is for <cfloop>.
		// The cost is an IndexByte and a lineAt on every <cfelse>,
		// <cfexit> and <cferror>; the alternative is that the second
		// half of every if/else chain in tag syntax has no calls in it.
		if hasCFTagPrefix(tag, "<cfelseif") {
			body := tagBody(tag)
			p.scanTagExpression(body, bodyLine(tag, body, line), idx+strings.Index(tag, body))
		}
	case 'r':
		if hasCFTagPrefix(tag, "<cfreturn") {
			p.parseCFReturn(tag, line, idx)
		}
	}
}

// nextTag is nextTagStart from pos, or, gated and outside an output context, a
// plain search for the next '<'.
//
// nextTagStart steps over a #...# span because markup inside one is an
// argument, not a tag. That holds only where hashes are evaluated. Outside an
// output context a hash is a literal character, and taking one as a span's
// opening hid real tags: a Handlebars "{{#associated_rolls}}" in a kiosk
// template paired with a hash far below, the walk stepped over every tag
// between them, and the <cfif> conditions there lost their calls — which the
// old reading only recovered by scanning the whole run as one expression.
func (p *tagParser) nextTag(pos int) int {
	if p.gated && !p.textEvaluated() {
		return strings.IndexByte(p.src[pos:], '<')
	}

	return nextTagStart(p.src[pos:])
}

// trackOutput moves the output state past the tag at idx, when gated.
func (p *tagParser) trackOutput(idx int) {
	if p.gated {
		p.output.track(p.tagEnds(), idx)
	}
}

// textEvaluated reports whether the text at the walk's position is where ColdFusion evaluates
// #...#. Ungated, all of it is, as the parser has always read it.
func (p *tagParser) textEvaluated() bool {
	return !p.gated || p.output.active()
}

// stepOverEvaluatedTag handles a tag the walk has no handler for but whose
// attributes ColdFusion evaluates in any context: a <cf...> tag the fast skip
// declines (<cfquery>, <cfmail>, <cfmodule>, <cf_custom>) or a cfimport-prefixed
// custom tag. Gated, it scans the attributes and returns the offset past the
// tag, so they are not left in the next text gap for the gate to drop. Ungated,
// it keeps the old walk, which advances one byte and scans the attributes as
// part of that gap.
func (p *tagParser) stepOverEvaluatedTag(idx int) int {
	if !p.gated {
		return idx + 1
	}

	end := p.tagEnds().at(idx)
	if end < 0 {
		return idx + 1
	}

	tagEnd := idx + end + 1
	p.scanInterpolatedText(p.src[idx:tagEnd], idx)

	return tagEnd
}

// enterFunctionScope records the <cffunction> the parser has just entered, so
// assignments inside it are attributed to the function rather than the file.
func (p *tagParser) enterFunctionScope(tagEnd, line int) {
	endLine := -1

	closeIdx := p.functionCloseAfter(tagEnd)
	if closeIdx >= 0 {
		endLine = p.lineAt(tagEnd + closeIdx)
	} else {
		// Not found within this region's text — the function body is interrupted
		// by a nested <cfscript> region split. Fall back to the whole-file
		// pre-scan for the real end.
		absStart := p.baseLine + line

		for _, s := range p.knownScopes {
			if s.Start == absStart {
				endLine = s.End - p.baseLine

				break
			}
		}
	}

	if endLine < 0 {
		return
	}

	p.inFunc = funcKey(line, endLine)
	p.scopes = append(p.scopes, FuncScope{Start: line, End: endLine})

	// The set starts nil and is built on first write. It was a map per
	// <cffunction> whether or not the function declared anything local, which on
	// the benchmark component was 13% of everything the tag parser allocated —
	// and a map is an expensive way to record nothing.
	p.localVars = p.localVars[:0]
	p.localSet = nil

	if len(p.funcs) > 0 {
		for _, arg := range p.funcs[len(p.funcs)-1].Arguments {
			p.markVarLocal(arg.Name)
		}
	}
}

// markVarLocal records name as declared local to the function being parsed,
// allocating the set on first use.
func (p *tagParser) markVarLocal(name string) {
	if p.isVarDeclaredLocal(name) {
		return
	}

	p.localVars = append(p.localVars, name)

	switch {
	case p.localSet != nil:
		p.localSet[strings.ToLower(name)] = struct{}{}
	case len(p.localVars) > maxLocalScan:
		p.localSet = make(map[string]struct{}, 2*len(p.localVars))
		for _, v := range p.localVars {
			p.localSet[strings.ToLower(v)] = struct{}{}
		}
	}
}

// maxLocalScan is the most locals isVarDeclaredLocal scans before it reads
// localSet instead.
const maxLocalScan = 32

// tagEnds is the tag-end search over p.src, built on first use; see tagEnds.
func (p *tagParser) tagEnds() *tagEnds {
	if !p.endsReady {
		p.ends, p.endsReady = newTagEnds(p.src), true
	}

	return &p.ends
}

// functionCloseAfter is the offset from from of the first </cffunction> at or
// after it, -1 for none. Each <cffunction> asks, twice, and the search ran to
// the end of the file whenever there was no close: a page repeating
// <cffunction name="f"><cfset x = g()> with no </cffunction> was quadratic,
// 12,000 of them (432KB) taking 1.6s. The answer is remembered with the offset
// it was searched from, and holds for any later offset up to the close it
// found, since no close lies between the two; with none found it holds for
// every later offset. The walk only moves forward, so each byte is searched
// once.
func (p *tagParser) functionCloseAfter(from int) int {
	if p.closeKnown && from >= p.closeFrom && (p.closeAt < 0 || from <= p.closeAt) {
		if p.closeAt < 0 {
			return -1
		}

		return p.closeAt - from
	}

	ci := indexCFTag(p.src[from:], "/cffunction")
	p.closeKnown, p.closeFrom, p.closeAt = true, from, ci

	if ci >= 0 {
		p.closeAt = from + ci
	}

	return ci
}

// parseCFFunction extracts a function def from <cffunction> and its <cfargument> children.
func (p *tagParser) parseCFFunction(tag string, idx, tagEnd, line int) {
	name := getAttr(tag, "name")
	if name == "" {
		return
	}

	// Check for JSDoc comment preceding this function tag
	docComment := p.precedingComment(idx)

	// Find arguments between this tag and </cffunction> or next <cffunction
	rest := p.src[tagEnd:]

	end := len(rest)

	ci := p.functionCloseAfter(tagEnd)
	if ci >= 0 {
		end = ci
	}

	// Only a <cffunction before the close can end the block sooner, so the
	// search stops at the close rather than running on into the next function.
	if ci := indexCFTag(rest[:end], "cffunction"); ci >= 0 {
		end = ci
	}

	block := rest[:end]

	args := p.parseCFArguments(block, tagEnd)

	// Apply JSDoc @param {type} annotations
	if docComment != "" {
		applyParameterDocs(docComment, args)
	}

	if p.argumentTypes != nil {
		p.argumentTypes(name, getAttr(tag, "access"), args)
	}

	// Create component refs for arguments with component-like types.
	// p.inFunc is not yet set at this call site — the main loop sets it after
	// parseCFFunction returns. Using p.addRef would route refs to componentRefs
	// (global), which causes cachedFuncRefs to miss them when the funcRefsMap
	// entry already exists from resolver-derived refs in the same scope.
	// Instead, compute the funcKey directly and write to funcRefs[key].
	for _, a := range args {
		if comp := argumentComponentType(&a); comp != "" {
			ref := ComponentRef{
				Variable:  a.Name,
				Component: comp,
				URI:       uriFromString(p.fileURI),
				Line:      conv.Uint32(line),
			}

			endLine := -1

			if end < len(rest) {
				endLine = p.lineAt(tagEnd + end)
			} else {
				// Not found within this region's text — the function body is
				// interrupted by a nested <cfscript> region split. Fall back
				// to the whole-file pre-scan for the real end (see baseLine).
				absStart := p.baseLine + line

				for _, s := range p.knownScopes {
					if s.Start == absStart {
						endLine = s.End - p.baseLine

						break
					}
				}
			}

			if endLine >= 0 {
				key := funcKey(line, endLine)

				if p.funcRefs == nil {
					p.funcRefs = make(map[string][]ComponentRef)
				}

				p.funcRefs[key] = append(p.funcRefs[key], ref)
			} else {
				p.componentRefs = append(p.componentRefs, ref)
			}
		}
	}

	returnType := getAttr(tag, "returntype")

	// Promote hint to returntype when returntype is generic and hint looks
	// like a component path — same safe mechanism as cfargument's Type/Hint
	// above (hint is documentation-only, so this carries no runtime
	// type-coercion risk the way editing returntype directly would).
	if returnHint := getAttr(tag, "hint"); isComponentType(returnHint) && !isComponentType(returnType) {
		returnType = returnHint
	}

	p.funcs = append(p.funcs, FunctionDef{
		Name:       name,
		URI:        uriFromString(p.fileURI),
		Line:       conv.Uint32(line),
		Arguments:  args,
		ReturnType: returnType,
	})
}

// parseCFArguments extracts <cfargument> tags from a block of source.
// precedingComment extracts the CFML comment (<!--- ... --->) immediately before position idx.
func (p *tagParser) precedingComment(idx int) string {
	// Walk backward from idx skipping whitespace to find --->
	i := idx - 1
	for i >= 0 && (p.src[i] == ' ' || p.src[i] == '\t' || p.src[i] == '\n' || p.src[i] == '\r') {
		i--
	}

	// Check for ---> ending (i points to '>')
	if i < 3 || p.src[i-3:i+1] != "--->" {
		return ""
	}

	// Find matching <!---
	start := strings.LastIndex(p.src[:i-3], "<!---")
	if start < 0 {
		return ""
	}

	return p.src[start+5 : i-3]
}

// blockStart is the offset of block within p.src, so each <cfargument> can be
// recorded at the line it is written on. An argument is a declaration — it is
// what `ref = argumentVariable` refers to — and without a line there is nowhere
// for go-to-definition to land.
func (p *tagParser) parseCFArguments(block string, blockStart int) []Argument {
	// Most functions take a few arguments, and a nil slice reaching four costs
	// three copies to get there.
	args := make([]Argument, 0, 4)
	ends := newTagEnds(block)

	pos := 0

	for {
		idx := indexCFTag(block[pos:], "cfargument")
		if idx < 0 {
			break
		}

		idx += pos

		end := ends.at(idx)
		if end < 0 {
			break
		}

		tag := block[idx : idx+end+1]

		name := getAttr(tag, "name")
		if name != "" {
			a := Argument{Name: name}
			a.Type = getAttr(tag, "type")
			a.Hint = getAttr(tag, "hint")
			req := getAttr(tag, "required")
			a.Required = strings.EqualFold(req, "true") || strings.EqualFold(req, "yes")
			// Promote hint to type when type is generic and hint looks like a component path
			if isComponentType(a.Hint) && !isComponentType(a.Type) {
				a.Type = a.Hint
			}

			args = append(args, a)

			p.vars = append(p.vars, VarDef{
				Name: name, Scope: ScopeArguments,
				Line: conv.Uint32(p.lineAt(blockStart + idx)),
			})
		}

		pos = idx + end + 1
	}

	return args
}

// addRef appends a component ref to the correct bucket (global or per-function).
// isArgument reports whether name is an argument of the function being
// parsed.
func (p *tagParser) isArgument(name string) bool {
	if len(p.funcs) == 0 {
		return false
	}

	for _, a := range p.funcs[len(p.funcs)-1].Arguments {
		if strings.EqualFold(a.Name, name) {
			return true
		}
	}

	return false
}

// isVarDeclaredLocal returns true if the variable was declared with var or local.
// in the current function.
func (p *tagParser) isVarDeclaredLocal(name string) bool {
	if p.localSet != nil {
		var buf foldScratch

		_, ok := p.localSet[string(buf.lowerFold(name))]

		return ok
	}

	for _, v := range p.localVars {
		if strings.EqualFold(v, name) {
			return true
		}
	}

	return false
}

func (p *tagParser) resolveCall(expr string) string {
	return resolveCallExpr(p.resolverSet, p.resolvers, expr)
}

// Refs assigned to VARIABLES. or this. scopes are always global.
func (p *tagParser) addRef(ref *ComponentRef) {
	ref.This = p.refThis

	if p.flow != nil && p.inFunc != "" {
		p.flow.note(ref.Variable, ref.Line, p.flow.innermost())
	}

	if p.inFunc == "" || p.forceGlobal {
		p.componentRefs = append(p.componentRefs, *ref)
	} else {
		if p.funcRefs == nil {
			p.funcRefs = make(map[string][]ComponentRef)
		}

		p.funcRefs[p.inFunc] = append(p.funcRefs[p.inFunc], *ref)
	}
}

// lookupComponentRef finds a previously-recorded ComponentRef for varName, so
// a call site like `psService.updatePerson(...)` can carry the Component that
// an earlier `psService = getService("person")` assignment already resolved
// (mirrors resolve.CanResolveCall's function-scope-then-global precedence).
// Only sees refs recorded earlier in this single forward pass — same
// limitation goto-definition has for forward references.
//
// Prefers the nearest-preceding assignment at or before atLine over the first
// match in parse order: a scratch variable can be reassigned multiple times in
// the same file (e.g. once per <cfswitch>/<cfcase> branch), and the first ref
// in parse order may belong to an earlier, unrelated branch. Falls back to the
// first match in parse order for a genuine forward reference, where no
// preceding ref exists.
func (p *tagParser) lookupComponentRef(varName string, atLine int) string {
	lookupVar := varName
	if dot := strings.LastIndexByte(lookupVar, '.'); dot >= 0 {
		lookupVar = lookupVar[dot+1:]
	}

	ref := (*ComponentRef)(nil)
	if p.inFunc != "" {
		ref = nearestComponentRef(p.funcRefs[p.inFunc], lookupVar, atLine)
	}

	if ref == nil {
		ref = nearestComponentRef(p.componentRefs, lookupVar, atLine)
	}

	// An assignment between the ref and atLine that only resolvePendingCalls
	// can type replaces what the ref says, and the parse cannot know with
	// what yet: `<cfset z = z.next()>` before `<cfset z.onlyB()>` left onlyB
	// checked against what z held before. Left empty, the resolver reads the
	// refs at check time, once that assignment has its own. One on atLine
	// itself is the statement the call is in, which runs first.
	if ref == nil || atLine > 0 && p.assignedSince(lookupVar, ref.Line, conv.Uint32(atLine-1)) {
		return ""
	}

	// A variable assigned more than once before the call may hold any of the
	// assignments a branch lets through; the resolver reads them all.
	if p.assignedBefore(lookupVar, atLine) > 1 {
		return ""
	}

	return ref.Component
}

// assignedBefore counts the refs of name made before line, in the function
// being parsed and at the top level.
func (p *tagParser) assignedBefore(name string, line int) int {
	n := 0

	count := func(refs []ComponentRef) {
		for i := range refs {
			if int(refs[i].Line) < line && strings.EqualFold(refs[i].Variable, name) {
				n++
			}
		}
	}

	if p.inFunc != "" {
		count(p.funcRefs[p.inFunc])
	}

	count(p.componentRefs)

	return n
}

// assignedSince is assignedBetween(p.pendingCalls, name, from, to), through
// an index by name kept as pendingCalls grows: asked for every call on a
// variable, the walk over every pending call was quadratic in a page of
// assignments and calls.
func (p *tagParser) assignedSince(name string, from, to uint32) bool {
	if p.pendingByName == nil || p.pendingNamed > len(p.pendingCalls) {
		p.pendingByName, p.pendingNamed = map[string][]int{}, 0
	}

	for ; p.pendingNamed < len(p.pendingCalls); p.pendingNamed++ {
		n := strings.ToLower(p.pendingCalls[p.pendingNamed].varName)
		p.pendingByName[n] = append(p.pendingByName[n], p.pendingNamed)
	}

	for _, i := range p.pendingByName[strings.ToLower(name)] {
		if c := &p.pendingCalls[i]; !c.returnExpr && !c.memberSet && c.line > from && c.line <= to && strings.EqualFold(c.varName, name) {
			return true
		}
	}

	return false
}

// nearestComponentRef returns the ref matching lookupVar whose Line is the
// largest value <= atLine, or the first matching ref in slice order if none
// precede atLine.
func nearestComponentRef(refs []ComponentRef, lookupVar string, atLine int) *ComponentRef {
	var best *ComponentRef

	for i := range refs {
		ref := &refs[i]
		if !strings.EqualFold(ref.Variable, lookupVar) || chainPending(ref) {
			continue
		}

		if int(ref.Line) > atLine {
			continue
		}

		if best == nil || ref.Line > best.Line {
			best = ref
		}
	}

	if best != nil {
		return best
	}

	for i := range refs {
		if strings.EqualFold(refs[i].Variable, lookupVar) && !chainPending(&refs[i]) {
			return &refs[i]
		}
	}

	return nil
}

func (p *tagParser) addCall(call *CallSite) {
	if p.callNames != nil && !namedIn(p.callNames, call.FuncName) {
		return
	}

	if p.inFunc == "" {
		p.calls = append(p.calls, *call)
	} else {
		if p.funcCalls == nil {
			p.funcCalls = make(map[string][]CallSite)
		}

		p.funcCalls[p.inFunc] = append(p.funcCalls[p.inFunc], *call)
	}
}

// extractAllLinks scans source lines for document links, routing them to
// global links or funcLinks based on which scope the line falls in.
func (p *tagParser) extractAllLinks() {
	p.funcLinks = extractLinksByScope(p.src, false, 0, p.scopes, p.funcLinks, &p.links)
}

// isWhitespace returns true if the byte is any whitespace character.
// isCloseTagFor reports whether tag is the closing tag for name, e.g.
// "</cffunction>" or "</cffunction >" for "cffunction". The name must be
// followed by whitespace or ">", so "</cffunctionfoo>" does not match.
func isCloseTagFor(tag, name string) bool {
	if len(tag) < len(name)+3 || tag[0] != '<' || tag[1] != '/' {
		return false
	}

	if !strings.EqualFold(tag[2:2+len(name)], name) {
		return false
	}

	rest := tag[2+len(name):]

	return rest[0] == '>' || isWhitespace(rest[0])
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

// hasCFTagPrefix checks if tag starts with prefix (case-insensitive) followed by whitespace.
func hasCFTagPrefix(tag, prefix string) bool {
	n := len(prefix)
	if len(tag) <= n {
		return false
	}

	if !strings.EqualFold(tag[:n], prefix) {
		return false
	}

	c := tag[n]

	return isWhitespace(c)
}

// extractLinksByScope files the links in src (a script region when script,
// else markup; regionLinks), its lines counted from lineNum, under the
// function each is in, or in links when it is in none.
func extractLinksByScope(src string, script bool, lineNum int, scopes []FuncScope, funcLinks map[string][]DocumentLink, links *[]DocumentLink) map[string][]DocumentLink {
	scopeIdx := 0

	for _, link := range regionLinks(src, script, lineNum) {
		line := int(link.Line)

		for scopeIdx < len(scopes) && line > scopes[scopeIdx].End {
			scopeIdx++
		}

		if scopeIdx < len(scopes) && line > scopes[scopeIdx].Start && line < scopes[scopeIdx].End {
			if funcLinks == nil {
				funcLinks = make(map[string][]DocumentLink)
			}

			key := funcKey(scopes[scopeIdx].Start, scopes[scopeIdx].End)
			funcLinks[key] = append(funcLinks[key], link)

			continue
		}

		*links = append(*links, link)
	}

	return funcLinks
}

// plainName reports whether s is a variable and nothing else: a name, or
// names joined by dots. A keyword (`true`, `null`) is not one.
func plainName(s string) bool {
	if s == "" || isKeyword(s) {
		return false
	}

	for part := range strings.SplitSeq(s, ".") {
		if part == "" || !isIdentStart(part[0]) {
			return false
		}

		for i := 1; i < len(part); i++ {
			if !isIdentPart(part[i]) {
				return false
			}
		}
	}

	return true
}

// rootName is the first name of a dotted one.
func rootName(s string) string {
	root, _, _ := strings.Cut(s, ".")

	return root
}

// untypedValueText reports whether rhs is a value no component can be: a
// string, a number or a boolean, or an argument passed through, whose type is
// the caller's. A call is not one: what it returns may be typed later.
func untypedValueText(rhs string) bool {
	rhs = strings.TrimSpace(rhs)
	if rhs == "" {
		return false
	}

	if c := rhs[0]; c == '"' || c == '\'' || c >= '0' && c <= '9' || c == '-' {
		return true
	}

	word := extractIdent(rhs)

	return strings.EqualFold(word, "true") || strings.EqualFold(word, "false") ||
		strings.EqualFold(word, "arguments") && len(rhs) > len(word) && rhs[len(word)] == '.'
}

// nativeJavaHead is the native type of `createObject( "java", "java.time.Instant" )`
// at the head of rhs when a call chain follows it, and whether the head is a
// Java object at all. The script parser types the same chain through
// dynamicCall.
func nativeJavaHead(rhs string) (native string, java bool) {
	if !strings.Contains(rhs[13:], ").") {
		return "", false
	}

	sc := NewScanner(rhs[13:]) // past the opening parenthesis

	first := sc.NextSkipComments()
	if first.Kind != TokString || !strings.EqualFold(unquote(first.Value), "java") || sc.NextSkipComments().Kind != TokComma {
		return "", false
	}

	class := sc.NextSkipComments()
	if class.Kind != TokString || sc.NextSkipComments().Kind != TokRParen {
		return "", false
	}

	return NativeJavaComponent(unquote(class.Value)), true
}
