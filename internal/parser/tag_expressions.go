// Handing a tag's expressions and #...# spans to the script parser, and merging the calls it finds.

package parser

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
)

// scanInterpolatedText records the calls written inside #...# spans of a piece
// of tag-region source, at byte offset `offset` within p.src.
//
// `<cfoutput>#svc.getName()#</cfoutput>` is how tag-syntax CFML emits a
// computed value, and none of it was visible: the tag parser matches tags and
// never tokenises text. The cost was the argument-list gap's — no edge into the
// callee, so the code map read it as unreachable and `unresolved` never checked
// it.
//
// Each span is handed to a scriptParser, the same way a <cfscript> body already
// is, so one implementation of "what is a call" serves both. A span that is not
// an expression yields no calls and is dropped, which is what keeps a stray
// pair of hashes in prose or a CSS colour from inventing one.
func (p *tagParser) scanInterpolatedText(text string, offset int) {
	// Only calls are merged below, and a call needs a '('. Most interpolation in
	// a tag file is `#user.name#` or `#i#`, so rejecting a chunk on one byte
	// scan is what keeps this from costing a fifth of a tag parse.
	if !p.extractCalls || strings.IndexByte(text, '#') < 0 || strings.IndexByte(text, '(') < 0 || !p.mayCall(text) {
		return
	}

	for _, span := range interpolatedSpans(text) {
		spanText := text[span.start:span.end]
		if strings.IndexByte(spanText, '(') < 0 {
			continue
		}

		p.scanExpressionCalls(spanText, p.lineAt(offset+span.start), offset+span.start)
	}
}

// scanTagExpression is scanExpressionCalls for a tag that is only an expression
// (<cfif>, <cfelseif>, <cfreturn>), told where it sits in the file so each call
// takes its line from its own source span: a chain written over lines
// (`new X(\n…\n).m()`) is on the line of `.m`, not of the tag.
func (p *tagParser) scanTagExpression(expr string, line, origin int) {
	p.mergeExpressionCalls(expr, line, false, true, origin)
}

// scanExpressionCalls records the calls written in one CFML expression.
//
// The tag parser matches tags and pulls attributes out with string searches; it
// has no expression parser and should not grow one. Where a tag *holds* an
// expression — a <cfif> condition, a <cfreturn>, a #...# span — it hands the
// text to a scriptParser and keeps the calls, which is the same thing a
// <cfscript> body already does. One implementation of "what is a call" then
// serves both syntaxes, and a fix to it reaches tag files for free.
//
// Calls and dynamic member assignments are merged. The refs and vars a
// sub-parse also produces belong to whatever the tag handler already recorded
// for them, and taking them here would record each one twice.
func (p *tagParser) scanExpressionCalls(expr string, line int, offsets ...int) {
	p.mergeExpressionCalls(expr, line, false, false, offsets...)
}

// Only <cfset> needs this. A #...# span and a <cfif> condition have nothing
// recorded before them, and deduping there is actively wrong: two
// interpolations of the same function on one line —
// `#getColdBoxSetting("a")# #getColdBoxSetting("b")#` — are two calls, and
// running the shared merge with the tally on cost the second one. The corpus
// caught that; a unit test on one span could not have.
// mergeExpressionCalls parses one CFML expression and records the calls in it.
//
// The tag parser matches tags and pulls attributes out with string searches; it
// has no expression parser and should not grow one. Where a tag *holds* an
// expression — a <cfif> condition, a <cfreturn>, a <cfset>, a #...# span — it
// hands the text to a scriptParser and keeps the calls, which is the same thing
// a <cfscript> body already does. One implementation of "what is a call" then
// serves both syntaxes, and a fix to it reaches tag files for free.
//
// Calls are merged, plus the member assignments that affect call resolution.
// The refs and vars a sub-parse also produces belong to whatever the tag
// handler already recorded for them, and taking them here would record each
// one twice.
//
// With topUp, the count already recorded on the starting line for each name is
// subtracted and only the excess is added, so a <cfset> keeps the component its
// own string path resolved against this file's refs while still gaining the
// calls that path never matched. Counting rather than testing presence is what
// keeps `<cfset x = f() + f()>` at two.
// exprCallKey identifies a call when mergeExpressionCalls subtracts the calls
// a <cfset> already recorded from its sub-parse.
type exprCallKey struct {
	name        string
	line, start uint32
	span        bool
}

// takeCounted reports whether have holds k (by span, else by name and line)
// and, when it does, takes one.
func takeCounted(have map[exprCallKey]int, k exprCallKey) bool {
	if have[k] == 0 && k.span {
		k.start, k.span = 0, false
	}

	if have[k] == 0 {
		return false
	}

	have[k]--

	return true
}

// expressionSubParse parses expr, a sub-parse of the text at offsets[0], for
// its calls. When the expression holds member assignments (memberSets) one
// parse answers both questions: those need the body path, so the expression is
// wrapped in a function; the calls are the same either way, and parsing the
// <cfset> twice was a tenth of a reparse of a large tag component.
func (p *tagParser) expressionSubParse(expr string, line int, memberSets bool, origin int, offsets []int) *scriptParser {
	if memberSets {
		sub := p.memberSetParser(expr, line)
		if origin >= 0 {
			// Spans are initially relative to the synthetic wrapper. Only
			// calls wholly inside its real expression are translated.
			sub.baseOffset = 0
		}

		sub.extractCalls = true
		sub.setCallNames(p.callNames)
		sub.parse()
		p.keepMemberSets(sub, origin)

		return sub
	}

	sub := p.subParser(expr, line)
	if len(offsets) > 0 && offsets[0] >= 0 {
		sub.baseOffset = p.srcOffset + offsets[0]
	}

	sub.resolverSet = p.resolverSet
	sub.extractCalls = true
	sub.setCallNames(p.callNames)
	sub.parse()

	return sub
}

// recordInstantiations adds the constructions a sub-parse found. A construction
// is written in an expression, so the sub-parse that reads the expression is
// what finds it.
func (p *tagParser) recordInstantiations(sub *scriptParser, wrapped, knownOrigin bool, origin int) {
	for _, in := range sub.instantiationsFound() {
		if wrapped && knownOrigin {
			// Counted from the wrapper's start, as its calls' spans are.
			in.at += origin - len(memberSetPrefix)
		}

		if !slices.Contains(p.instantiations, in) {
			p.instantiations = append(p.instantiations, in)
		}
	}
}

// exprKeyOf is how a call is matched against those the string paths recorded:
// by name, and with spanLines by line and by source span.
func exprKeyOf(call *CallSite, spanLines bool) exprCallKey {
	k := exprCallKey{name: strings.ToLower(call.FuncName)}
	if spanLines {
		k.line = call.Line
		if call.HasSpan {
			k.start, k.span = call.StartByte, true
		}
	}

	return k
}

// mergeExpressionCalls merges the calls of expr, a sub-parse of the text at
// offsets[0]. With topUp, offsets[1] is how many calls the bucket held before
// the string paths ran, and only calls beyond those are added. With spanLines,
// each call takes its line from its own source span.
func (p *tagParser) mergeExpressionCalls(expr string, line int, topUp, spanLines bool, offsets ...int) {
	if !p.extractCalls || !p.mayCall(expr) {
		return
	}

	// A parse recording a few names' calls keeps nothing else, as a parse
	// that extracts no call does not: the member writes the <cfset> sub-parse
	// reads are an ExtractCalls parse's alone.
	memberSets := topUp && p.callNames == nil && hasMemberSetShape(expr)
	calls := strings.IndexByte(expr, '(') >= 0

	if !calls {
		if memberSets {
			p.mergeExpressionMemberSets(expr, line)
		}

		return
	}

	knownOrigin := len(offsets) > 0 && offsets[0] >= 0

	origin := -1
	if knownOrigin {
		origin = p.srcOffset + offsets[0]
	}

	wrapped := memberSets
	sub := p.expressionSubParse(expr, line, memberSets, origin, offsets)

	p.recordInstantiations(sub, wrapped, knownOrigin, origin)

	key := func(call *CallSite) exprCallKey { return exprKeyOf(call, spanLines) }
	have := map[exprCallKey]int{}

	if topUp {
		cs := p.recordedForExpression(expr, line, offsets)
		for i := range cs {
			c := &cs[i]

			have[key(c)]++
		}
	}

	exprLines := lineCounter{text: expr}

	// A call without a span is found by scanning the whole expression again,
	// so an expression holding thousands of them was quadratic: a 1MB
	// <cfset> of unclosed interpolations hung the parse, and so the editor,
	// for good. Past originBudget bytes scanned the rest keep an unknown
	// span, which is what an ambiguous origin already gets.
	originBudget := maxOriginScan

	emit := func(c CallSite) {
		if knownOrigin && (c.HasSpan || originBudget >= len(expr)) {
			if !c.HasSpan {
				originBudget -= len(expr)
			}

			expressionWrappedCallOrigin(&c, expr, p.srcOffset+offsets[0], line, wrapped, spanLines, &exprLines)
		}

		// The wrapper is not a function of the file: a call it encloses has
		// no caller of its own, as it has unwrapped, and fillCallers gives it
		// the one whose lines hold it.
		if c.Caller == memberSetWrapper {
			c.Caller = ""
		}

		// Only a top-up has calls to subtract; without one, nothing is looked
		// up and no name is lowercased for it (a mixed-case name allocates).
		if len(have) > 0 && takeCounted(have, key(&c)) {
			return
		}

		p.addCall(&c)
	}

	for i := range sub.calls {
		c := &sub.calls[i]

		emit(*c)
	}

	for _, calls := range sub.funcCalls {
		for i := range calls {
			c := &calls[i]

			emit(*c)
		}
	}
}

// recordedForExpression is the calls already recorded for the expression a
// <cfset> top-up is about to parse: the string paths' calls on its line, or,
// with offsets naming where the expression sits and how many calls the
// bucket held before them, those calls and the earlier ones inside its span.
func (p *tagParser) recordedForExpression(expr string, line int, offsets []int) []CallSite {
	if len(offsets) <= 1 {
		return p.callsOnLine(conv.Uint32(line))
	}

	bucket := p.calls
	if p.inFunc.in {
		bucket = p.funcCalls[p.inFunc]
	}

	cs := slices.Clone(bucket[offsets[1]:])

	// The calls recorded before the string paths ran that lie inside this
	// expression were recorded while this tag was handled, from tagCalls on.
	// Reading every call the function had made so far for every <cfset> was
	// quadratic in a function's tags: 24,000 took 0.4s.
	exprStart, exprEnd := p.srcOffset+offsets[0], p.srcOffset+offsets[0]+len(expr)

	for i := min(p.tagCalls, offsets[1]); i < offsets[1]; i++ {
		call := &bucket[i]
		if call.HasSpan && int(call.StartByte) >= exprStart && int(call.EndByte) <= exprEnd {
			cs = append(cs, *call)
		}
	}

	return cs
}

// mergeExpressionMemberSets records a member assignment held by <cfset>.
// Wrapping the expression in a function deliberately sends it through the
// script parser's body path, the one implementation that distinguishes
// `a.m = value` from `a.m == value` and indexed receivers.
func (p *tagParser) mergeExpressionMemberSets(expr string, line int) {
	if !hasMemberSetShape(expr) {
		return
	}

	sub := p.memberSetParser(expr, line)
	sub.parse()
	p.keepMemberSets(sub)
}

// hasMemberSetShape is the cheap test for text that may assign a member.
func hasMemberSetShape(expr string) bool {
	return strings.Contains(expr, ".") && strings.Contains(expr, "=")
}

// memberSetParser is the sub-parser mergeExpressionMemberSets reads an
// expression with: wrapped in a function, so it goes through the script
// parser's body path.
func (p *tagParser) memberSetParser(expr string, line int) *scriptParser {
	sub := p.subParser(memberSetPrefix+expr+"}", line)
	sub.resolverSet = p.resolverSet

	return sub
}

// freshExprParsers makes subParser build a parser each time, as every
// expression did before they were reused. Only a test sets it, to compare the
// two.
var freshExprParsers bool

// subParser is the script parser an expression of this file is read with,
// reset for it. There is one, reused: the expression's results are read out of
// it before the next is parsed, and nothing keeps a reference into it. A
// caller finishes with the sub-parser it was given before asking for another.
func (p *tagParser) subParser(src string, line int) *scriptParser {
	if p.exprParser == nil || freshExprParsers {
		p.exprParser = newScriptParser(src, p.fileURI, line, p.resolvers)
	} else {
		p.exprParser.reset(src, p.fileURI, line, p.resolvers)
	}

	return p.exprParser.asCFScript()
}

// memberSetWrapper names the function memberSetParser wraps an expression in.
const memberSetWrapper = "__cfset"

// memberSetPrefix is the text memberSetParser puts before the expression.
const memberSetPrefix = "function " + memberSetWrapper + "(){"

// keepMemberSets takes the member assignments a memberSetParser found.
func (p *tagParser) keepMemberSets(sub *scriptParser, origins ...int) {
	for i := range sub.pendingCalls {
		c := sub.pendingCalls[i]
		if !c.memberSet {
			continue
		}

		if c.hasClosureSpan && len(origins) > 0 && origins[0] >= 0 {
			prefix := len(memberSetPrefix)
			if int(c.closureFrom) >= prefix && int(c.closureTo) <= len(sub.sc.src)-1 {
				c.closureFrom = conv.Uint32(origins[0] + int(c.closureFrom) - prefix)
				c.closureTo = conv.Uint32(origins[0] + int(c.closureTo) - prefix)
			} else {
				c.hasClosureSpan = false
			}
		}

		c.funcKey = p.inFunc
		p.pendingCalls = append(p.pendingCalls, c)
	}
}

// callsOnLine returns the calls already recorded in the current bucket for a
// line. Calls are appended in source order, so the run at the end of the bucket
// is the whole answer and this costs what it finds rather than what the file
// holds.
func (p *tagParser) callsOnLine(line uint32) []CallSite {
	bucket := p.calls
	if p.inFunc.in {
		bucket = p.funcCalls[p.inFunc]
	}

	i := len(bucket)
	for i > 0 && bucket[i-1].Line == line {
		i--
	}

	return bucket[i:]
}

// tagBody returns what a tag holds between its name and its closing '>': the
// condition of a <cfif>, the expression of a <cfreturn>.
func tagBody(tag string) string {
	inner := tag

	// The first whitespace byte of any kind ends the tag name: a condition
	// written on the line after `<cfif` follows a newline or a tab. Looking
	// for a space alone found no body at all in `<cfif\n\tf( a )>`, and cut
	// `<cfif\n\t( f( a ) and g( b ) )>` at the space before `and`.
	i := 0
	for i < len(inner) && !isWhitespace(inner[i]) {
		i++
	}

	if i == len(inner) {
		return ""
	}

	inner = inner[i+1:]

	inner = strings.TrimSuffix(inner, ">")
	inner = strings.TrimSuffix(inner, "/")

	return strings.TrimSpace(inner)
}

// holdsOnlyAnExpression reports whether a tag is nothing but the expression its
// handler parses: <cfif>, <cfelseif> and <cfreturn> have no attributes, so the
// whole tag after its name is one CFML expression. The handler's parse reads
// the `#…#` spans in its strings and outside them alike.
func holdsOnlyAnExpression(tag string) bool {
	return hasCFTagPrefix(tag, "<cfif") || hasCFTagPrefix(tag, "<cfelseif") || hasCFTagPrefix(tag, "<cfreturn")
}

// bodyLine is the line a tag's body starts on: the tag's own line plus the
// newlines before the body, which tagBody trims. A condition written under
// `<cfif` is on the next line, and its calls were reported on the tag's.
func bodyLine(tag, body string, line int) int {
	i := strings.Index(tag, body)
	if i < 0 || body == "" {
		return line
	}

	return line + strings.Count(tag[:i], "\n")
}

// trailingCalls returns the names of the method calls chained after the first
// call in expr, in order: for "b.width(1).height(2).build()" it is
// ["height", "build"], for createObject("java", "x").builder().build()
// ["builder", "build"]. It stops at anything that is not another .name(...)
// hop, and returns nil when expr has no complete first call.
func trailingCalls(expr string) []string {
	i := strings.IndexByte(expr, '(')
	if i < 0 {
		return nil
	}

	i = skipCallGroup(expr, i)

	var hops []string

	for i >= 0 {
		j := lex.SkipSpace(expr, i)
		if j >= len(expr) || expr[j] != '.' {
			break
		}

		j = lex.SkipSpace(expr, j+1)
		start := j

		for j < len(expr) && lex.IsIdent(expr[j]) {
			j++
		}

		name := expr[start:j]

		j = lex.SkipSpace(expr, j)

		if name == "" {
			break
		}

		if j >= len(expr) || expr[j] != '(' {
			hops = append(hops, PropertyHop(name))
			i = j

			continue
		}

		end := skipCallGroup(expr, j)
		if end < 0 {
			break
		}

		hops = append(hops, CallHop(expr[start:end]))
		i = end
	}

	return hops
}

// skipCallGroup returns the offset just past the ( … ) group opening at open,
// stepping over quoted strings, or -1 when the group does not close.
func skipCallGroup(expr string, open int) int {
	depth := 0

	for i := open; i < len(expr); i++ {
		switch c := expr[i]; c {
		case '"', '\'':
			end := strings.IndexByte(expr[i+1:], c)
			if end < 0 {
				return -1
			}

			i += end + 1
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return -1
}
