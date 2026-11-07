package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// maxReachingAssignments bounds how many assignments to one name a use reads
// back through.
const maxReachingAssignments = 64

// reachingUnion is the component a use of name holds when more than one
// assignment to it can reach the use: the alternatives joined, as a return
// joins the branches that reach it (`a|b`). best is the latest assignment at
// or before the use, which is what the lookup answered before; it is the
// answer unless a branch hides assignments earlier in the function.
//
// The assignments are read back from the use until one that always runs
// (its block is still open at the use) or the function's start; every one
// passed on the way is conditional and may be the value. All of them must be
// typed, with no native or dynamic component among them, and an assignment
// nothing dominates leaves the incoming value (an argument, a field) open,
// so then the answer stays best. Braces decide blocks, so a tag body, which
// has none, is never joined.
func reachingUnion(pr *parser.ParseResult, scopeStart int, name string, line uint32, refs []parser.ComponentRef, best *parser.ComponentRef) *parser.ComponentRef {
	if best == nil || best.Component == "" || !strings.Contains(pr.Content, "{") {
		return best
	}

	use := int(line)

	var at []int

	for header := use + 1; len(at) < maxReachingAssignments; header = at[len(at)-1] {
		_, l, ok := localAssignmentAt(pr.Content, name, scopeStart, header)
		if !ok {
			break
		}

		at = append(at, l)
	}

	if len(at) < 2 {
		return best
	}

	paths := blockPaths(pr.Content, scopeStart, use, at)

	var comps []string

	dominated := false

	for _, l := range at {
		ref := refAtLine(refs, name, l)
		if ref == nil || ref.Component == "" || ref.HasClosureSpan || ref.VisibleTo != 0 ||
			strings.HasPrefix(ref.Component, parser.NativeTypePrefix) || strings.HasPrefix(ref.Component, "$") {
			return best
		}

		comps = append(comps, strings.Split(ref.Component, "|")...)

		if isPrefixPath(paths[l], paths[use]) {
			dominated = true

			break
		}
	}

	if !dominated {
		return best
	}

	slices.Sort(comps)
	comps = slices.Compact(comps)

	if len(comps) < 2 {
		return best
	}

	joined := *best
	joined.Component = strings.Join(comps, "|")

	return &joined
}

// refAtLine is the ref the assignment to name on line made, the last when a
// line made several.
func refAtLine(refs []parser.ComponentRef, name string, line int) *parser.ComponentRef {
	var found *parser.ComponentRef

	for i := range refs {
		if int(refs[i].Line) == line && strings.EqualFold(refs[i].Variable, name) {
			found = &refs[i]
		}
	}

	return found
}

// isPrefixPath reports whether the block chain a is the start of b: a block
// opened by the assignment's line is still open at the use.
func isPrefixPath(a, b []int) bool {
	return len(a) <= len(b) && slices.Equal(a, b[:len(a)])
}

// blockPaths is, for each of lines and for use, the chain of braces open at
// the start of that line, counted from the line scopeStart. The block an id
// names is the nth `{` the scan met, so two chains share an id only for the
// same block.
func blockPaths(content string, scopeStart, use int, lines []int) map[int][]int {
	paths := make(map[int][]int, len(lines)+1)

	from, ok := lineOffset(content, scopeStart)
	if !ok {
		return paths
	}

	wanted := append([]int{use}, lines...)
	slices.Sort(wanted)

	scanner := parser.NewScanner(content[from:])

	var (
		stack []int
		next  int
		w     int
	)

	record := func(offset int) {
		for w < len(wanted) {
			start, ok := lineOffset(content, wanted[w])
			if !ok || start-from > offset {
				return
			}

			paths[wanted[w]] = slices.Clone(stack)
			w++
		}
	}

	for {
		tok := scanner.NextSkipComments()
		record(tok.Offset)

		switch tok.Kind {
		case parser.TokEOF:
			record(len(content))

			return paths
		case parser.TokLBrace:
			next++
			stack = append(stack, next)
		case parser.TokRBrace:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		default:
		}

		if w >= len(wanted) {
			return paths
		}
	}
}
