package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
)

// tagEnds answers tagEndIndex(s[from:]) for the many offsets a walk over one
// document asks about, without each answer costing a scan to wherever the
// tag ends.
//
// Asked one tag at a time, tagEndIndex is linear in how far the tag's '>' is,
// so a walk is quadratic whenever that is far for many tags at once:
//
//   - Past s's last '>' there is no end to find (lastGT): "<cf" repeated to
//     1.5MB took 5.9s.
//   - A '>' inside a quoted value with no real end after it sends the
//     quote-aware walk to the end of s from every tag: `<cfset x="a>b"`
//     repeated to 560KB took 19.7s. The walk's state is only its position
//     between quoted values, so two walks reaching the same position end
//     alike: walked remembers each position's answer, and every later walk
//     stops at the first position an earlier one passed through.
//   - A long run with no '>' at all costs the scan for it and the quote
//     counts over it from every tag inside the run. Once one tag's '>' is
//     more than longTagSpan away, gaps holds the next '>' and the quote
//     counts for every offset, found once.
type tagEnds struct {
	s      string
	lastGT int
	walked map[int]int
	gaps   *tagGaps
	// visited is the walk's scratch list of positions, kept between walks.
	visited []int
}

// longTagSpan is how far a tag's '>' may be before tagEnds indexes s rather
// than scanning it again for the next tag. Nearly every tag ends within a
// line or two; a run this long with no '>' is not markup anyone writes.
const longTagSpan = 4096

// tagGaps indexes s for tagEnds: for every offset, where the next '>' and the
// next "<!--" are, and how many of each quote come before it.
type tagGaps struct {
	nextGT, nextComment []int32
	dq, sq              []int32
}

// quoteNeverCloses is a walk that met a quote with no closing one, after which
// tagEndIndex settles for the first '>' from where the walk began.
const quoteNeverCloses = -2

func newTagEnds(s string) tagEnds {
	return tagEnds{s: s, lastGT: strings.LastIndexByte(s, '>')}
}

// at is tagEndIndex(t.s[from:]): the offset of the '>' ending the tag that
// opens at from, relative to from, or -1 when the tag has no end.
func (t *tagEnds) at(from int) int {
	if from > t.lastGT {
		return -1
	}

	var gt int

	if t.gaps != nil {
		gt = int(t.gaps.nextGT[from]) - from
	} else {
		gt = strings.IndexByte(t.s[from:], '>')
		if gt > longTagSpan {
			t.gaps = indexTagGaps(t.s)
		}
	}

	if t.hasComment(from, from+gt) {
		return tagEndWithComments(t.s[from:])
	}

	if t.quotesEven(from, from+gt) {
		return gt
	}

	switch end := t.walk(from); end {
	case quoteNeverCloses:
		return gt
	case -1:
		return -1
	default:
		return end - from
	}
}

// hasComment says whether a "<!--" starts within s[from:to].
func (t *tagEnds) hasComment(from, to int) bool {
	if t.gaps != nil {
		return int(t.gaps.nextComment[from])+4 <= to
	}

	return strings.Contains(t.s[from:to], "<!--")
}

// quotesEven says whether s[from:to] holds an even number of each quote, the
// fast path tagEndIndex takes.
func (t *tagEnds) quotesEven(from, to int) bool {
	if g := t.gaps; g != nil {
		return (g.dq[to]-g.dq[from])%2 == 0 && (g.sq[to]-g.sq[from])%2 == 0
	}

	return strings.Count(t.s[from:to], `"`)%2 == 0 && strings.Count(t.s[from:to], "'")%2 == 0
}

// walk is tagEndWalk(t.s[start:]) in absolute offsets, returning the '>' it
// ends at, -1, or quoteNeverCloses, and remembering the answer for every
// position it passed through.
func (t *tagEnds) walk(start int) int {
	visited := t.visited[:0]
	result := -1
	gtAt := -1

	for pos := start; ; {
		if r, ok := t.walked[pos]; ok {
			result = r

			break
		}

		visited = append(visited, pos)

		if gtAt < pos {
			gtAt = t.nextGT(pos)
			if gtAt < 0 {
				break
			}
		}

		q := indexQuote(t.s[pos:gtAt])
		if q < 0 {
			result = gtAt

			break
		}

		j := skipQuotedIn(t.s, pos+q)
		if j < 0 {
			result = quoteNeverCloses

			break
		}

		pos = j + 1
	}

	if t.walked == nil {
		t.walked = map[int]int{}
	}

	for _, p := range visited {
		t.walked[p] = result
	}

	t.visited = visited

	return result
}

// nextGT is the offset of the first '>' at or after pos, or -1.
func (t *tagEnds) nextGT(pos int) int {
	if t.gaps != nil {
		if pos >= len(t.gaps.nextGT) {
			return -1
		}

		return int(t.gaps.nextGT[pos])
	}

	if pos > t.lastGT {
		return -1
	}

	return pos + strings.IndexByte(t.s[pos:], '>')
}

// indexTagGaps builds tagGaps over s in one pass from each end. An offset
// with no '>' after it holds len(s), which no caller reaches, since at checks
// lastGT first.
func indexTagGaps(s string) *tagGaps {
	n := len(s)
	g := &tagGaps{
		nextGT:      make([]int32, n+1),
		nextComment: make([]int32, n+1),
		dq:          make([]int32, n+1),
		sq:          make([]int32, n+1),
	}

	gt, comment := conv.Int32(n), conv.Int32(n)
	g.nextGT[n], g.nextComment[n] = gt, comment

	for i := n - 1; i >= 0; i-- {
		switch s[i] {
		case '>':
			gt = conv.Int32(i)
		case '<':
			if strings.HasPrefix(s[i:], "<!--") {
				comment = conv.Int32(i)
			}
		}

		g.nextGT[i], g.nextComment[i] = gt, comment
	}

	for i := range n {
		g.dq[i+1], g.sq[i+1] = g.dq[i], g.sq[i]

		switch s[i] {
		case '"':
			g.dq[i+1]++
		case '\'':
			g.sq[i+1]++
		}
	}

	return g
}
