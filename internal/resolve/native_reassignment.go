package resolve

import (
	"math/bits"
	"sort"

	"github.com/cfmleditor/clif/internal/parser"
)

// nativeReassignedAt reports whether name is assigned on a line from start to
// before header. The assignments are the parse's, which stand only where a
// statement can, so a named argument or a struct key of the same name is
// none. It is bounded to separate statement lines; it is not branch
// dominance analysis.
func nativeReassignedAt(as []parser.Assignment, content, name string, start, header int) bool {
	if header <= start {
		return false
	}

	_, _, assigned := lastAssignment(as, content, name, start, header)

	return assigned
}

// nativeClaimMayNotReach reports whether the native type a ref gives name at
// refLine may not be the value at useLine: the ref's block closed before the
// use, so another path reaches the use without it, and an assignment earlier
// in the function (from scopeStart) may be the value on that path. A ref whose
// block is still open at the use dominates it, and a variable assigned once
// has nothing else to hold. It reads braces, so it speaks for script only; a
// tag body reports false, as the check it joins always has.
func nativeClaimMayNotReach(as []parser.Assignment, content, name string, scopeStart, refLine, useLine int) bool {
	if useLine <= refLine || !blockClosedBetween(content, refLine, useLine) {
		return false
	}

	_, _, earlier := lastAssignment(as, content, name, scopeStart, refLine)

	return earlier
}

// blockClosedBetween reports whether the braces from the start of fromLine to
// the start of toLine close more blocks than they open at some point: the
// block fromLine is in has ended by toLine.
func blockClosedBetween(content string, fromLine, toLine int) bool {
	from, ok := lineOffset(content, fromLine)
	if !ok {
		return false
	}

	to, ok := lineOffset(content, toLine)
	if !ok || to <= from {
		return false
	}

	return braces.of(content, newBraceIndex).closesBetween(from, to)
}

// braces remembers the brace index of the texts asked about most recently. A
// file is asked about once per use of a variable a native type was claimed
// for, over the lines from its ref to the use, and each ask scanned those
// lines: on Lucee that was 36 of the 46 times the scan read its text.
var braces = &textCache[braceIndex]{size: func(b braceIndex) int { return 8*len(b.offs) + 4*len(b.depth) + 4*len(b.floor) }}

// braceIndex is where a text's braces are and how deep each leaves it, so
// whether a span closes a block it did not open is a search and a minimum
// rather than a scan of the span.
type braceIndex struct {
	offs  []int     // the byte offset of each brace, in order
	depth []int32   // the depth after each, from the top of the text
	floor [][]int32 // floor[k][i] is the least depth after braces i to i+2^k-1
}

func newBraceIndex(content string) braceIndex {
	var b braceIndex

	sc := parser.NewSourceScanner(content)
	d := int32(0)

	for tok := sc.NextSkipComments(); tok.Kind != parser.TokEOF; tok = sc.NextSkipComments() {
		switch tok.Kind {
		case parser.TokLBrace:
			d++
		case parser.TokRBrace:
			d--
		default:
			continue
		}

		b.offs = append(b.offs, tok.Offset)
		b.depth = append(b.depth, d)
	}

	b.floor = append(b.floor, b.depth)

	for k := 1; 1<<k <= len(b.depth); k++ {
		prev := b.floor[k-1]
		row := make([]int32, len(b.depth)-1<<k+1)

		for i := range row {
			row[i] = min(prev[i], prev[i+1<<(k-1)])
		}

		b.floor = append(b.floor, row)
	}

	return b
}

// closesBetween reports whether the braces at byte offsets from to before to
// take the depth below where it stood at from.
func (b braceIndex) closesBetween(from, to int) bool {
	lo := sort.SearchInts(b.offs, from)
	hi := sort.SearchInts(b.offs, to)

	if lo >= hi {
		return false
	}

	base := int32(0)
	if lo > 0 {
		base = b.depth[lo-1]
	}

	k := bits.Len(uint(hi-lo)) - 1
	least := min(b.floor[k][lo], b.floor[k][hi-1<<k])

	return least < base
}

// nativeRefMissesUse reports whether the native type a ref gives name at
// refLine is not the value at useLine: the variable is assigned again between
// them, or another path may reach the use.
func nativeRefMissesUse(as []parser.Assignment, content, name string, scopeStart, refLine, useLine int) bool {
	return nativeReassignedAt(as, content, name, refLine+1, useLine) || nativeClaimMayNotReach(as, content, name, scopeStart, refLine, useLine)
}
