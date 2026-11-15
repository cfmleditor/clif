package lex

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// Wild matches text against literal pieces with a run of characters between
// each two, ignoring case in the pieces: the matcher behind a configuration's
// `get$1Service` and the spelled names of interception points and templates,
// which a regular expression of QuoteMeta and a wildcard once matched.
//
// A run is made of whole characters that Accept takes, Least of them at the
// fewest, and the shortest that lets the rest match is taken first. EndAnchored
// makes the last piece end the text; otherwise what follows it is left alone.
type Wild struct {
	// Accept reports whether a character whose first byte is c may stand in a
	// run.
	Accept      func(c byte) bool
	Least       int // 0 or 1
	EndAnchored bool
}

// WildMemo remembers the (piece, position) pairs that failed, for one text: a
// piece that cannot be matched from a position cannot be from it by any other
// route either, so the search is bounded by pieces times positions where
// trying every split of the runs between many wildcards is exponential. Its
// zero value is ready, and it is only used for a pattern with two wildcards or
// more; the map is made on the first failure. One WildMemo serves every start
// of one text.
type WildMemo struct {
	failed map[[2]int]bool
}

func (m *WildMemo) has(piece, pos int) bool { return m != nil && m.failed[[2]int{piece, pos}] }

func (m *WildMemo) add(piece, pos int) {
	if m == nil {
		return
	}

	if m.failed == nil {
		m.failed = map[[2]int]bool{}
	}

	m.failed[[2]int{piece, pos}] = true
}

// MatchAt reports whether s matches pieces with pieces[0] at pos, and the runs
// the wildcards took as [start, end) offsets of s, when capture is set. memo
// is shared across the calls for one text; it may be nil.
func (w Wild) MatchAt(s string, pieces []string, pos int, memo *WildMemo, capture bool) (runs [][2]int, ok bool) {
	if len(pieces) <= 2 {
		memo = nil // one wildcard or none has nothing to explode
	}

	return w.from(s, pieces, 0, pos, memo, capture, nil)
}

func (w Wild) from(s string, pieces []string, piece, pos int, memo *WildMemo, capture bool, runs [][2]int) ([][2]int, bool) {
	lit := pieces[piece]
	if pos+len(lit) > len(s) || !strings.EqualFold(s[pos:pos+len(lit)], lit) || memo.has(piece, pos) {
		return nil, false
	}

	end := pos + len(lit)
	if piece == len(pieces)-1 {
		return runs, !w.EndAnchored || end == len(s)
	}

	for run := 0; ; {
		if run >= w.Least {
			next := runs
			if capture {
				next = slices.Concat(runs, [][2]int{{end, end + run}})
			}

			if got, ok := w.from(s, pieces, piece+1, end+run, memo, capture, next); ok {
				return got, true
			}
		}

		if end+run >= len(s) || !w.Accept(s[end+run]) {
			memo.add(piece, pos)

			return nil, false
		}

		_, width := utf8.DecodeRuneInString(s[end+run:])
		run += width
	}
}
