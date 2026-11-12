package resolve

import (
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

	scanner := parser.NewSourceScanner(content[from:to])
	depth := 0

	for {
		switch tok := scanner.NextSkipComments(); tok.Kind {
		case parser.TokEOF:
			return false
		case parser.TokLBrace:
			depth++
		case parser.TokRBrace:
			depth--
			if depth < 0 {
				return true
			}
		default:
		}
	}
}

// nativeRefMissesUse reports whether the native type a ref gives name at
// refLine is not the value at useLine: the variable is assigned again between
// them, or another path may reach the use.
func nativeRefMissesUse(as []parser.Assignment, content, name string, scopeStart, refLine, useLine int) bool {
	return nativeReassignedAt(as, content, name, refLine+1, useLine) || nativeClaimMayNotReach(as, content, name, scopeStart, refLine, useLine)
}
