package parser

import (
	"strings"
)

// expressionWriteCache keeps what scanExpressionWrites read from each <cfset>
// body, from one signature pass to the next. A reparse reads every <cfset> in
// the file again, and an edit changes one of them: on a 65,000-line tag
// component the rest were a tenth of the reparse. Only the bodies the last
// pass read are kept, so the cache is never larger than the file's <cfset>s.
type expressionWriteCache struct {
	prev, next map[string]cachedWrites
}

// cachedWrites is a body's writes with its text, which is the map key's own
// copy: a body carried over to the next pass is stored under that copy rather
// than under the slice of the document the pass found it in, so the cache
// keeps no document text of its own.
type cachedWrites struct {
	key    string
	writes []expressionWrite
}

// begin starts a pass: what the last one read is kept for this one to take.
func (c *expressionWriteCache) begin() {
	c.prev, c.next = c.next, map[string]cachedWrites{}
}

// scan is scanExpressionWrites, answered from the last pass when it read the
// same text. A nil cache scans.
func (c *expressionWriteCache) scan(expression string) []expressionWrite {
	if c == nil {
		return scanExpressionWrites(expression)
	}

	if e, ok := c.next[expression]; ok {
		return e.writes
	}

	e, ok := c.prev[expression]
	if !ok {
		// Read from a copy, so the strings kept refer to it and not to the
		// whole of this pass's content.
		e.key = strings.Clone(expression)
		e.writes = scanExpressionWrites(e.key)
	}

	c.next[e.key] = e

	return e.writes
}

// hasTagRegion reports whether the file has a tag region.
func (pr *ParseResult) hasTagRegion() bool {
	for i := range pr.Regions {
		if pr.Regions[i].Kind == RegionTag {
			return true
		}
	}

	return false
}

// newlineEndsStatement reports whether t, the token after prev, starts a new
// statement because a line ends between them: CFScript does not need the
// ";", and ColdBox and cfwheels leave it out throughout, so
// `services.h = new H( this )` ran on to the next `;` or `}` and its value
// was the rest of the function. It is the folding scan's rule: the line's
// last token could end an expression and the next line starts with a word
// that is not an operator. A line ending on an operator, an open bracket or
// a comma, or a line starting with one, continues the statement.
func newlineEndsStatement(src string, prev, t Token) bool {
	if t.Kind != TokIdent || t.Line == prev.Line || classify(t.Value)&wordOperator != 0 {
		return false
	}

	switch prev.Kind {
	case TokIdent:
		if classify(prev.Value)&(wordOperator|wordLeading) != 0 {
			return false
		}
	case TokString, TokNumber, TokRParen, TokRBracket, TokRBrace:
	default:
		return false
	}

	if prev.Offset >= t.Offset || t.Offset > len(src) {
		return false
	}

	between := src[prev.Offset:t.Offset]

	nl := strings.LastIndexByte(between, '\n')

	return nl >= 0 && strings.TrimSpace(between[nl+1:]) == ""
}
