package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// `x = cond ? a : b` and `x = a ?: b` hold one branch or the other, and the
// parse typed neither: the inject-or-default shape,
//
//	variables.client = isObject( arguments.client ) ? arguments.client : new Client();
//
// left every call on variables.client unresolved. assignedTernary types each
// branch as the right-hand side of an assignment of its own (a construction,
// a name, a chain of calls) and holds their union, the `a|b` the flow join
// writes. Every branch must be typed: one that is not may hold anything, so a
// branch naming an untyped argument types the variable only when the argument
// is typed, by its declaration or (in a batch scan) by what its callers pass.
// A dynamic branch makes the whole dynamic, as in the join.

// maxTernaryBranches bounds a chain of nested conditionals.
const maxTernaryBranches = 8

// ternaryBranches splits rhs into the values a conditional expression may
// take: a and b of `cond ? a : b`, a and b of `a ?: b`, with a nested
// conditional in either taken apart in turn. ok is false when rhs is not a
// conditional at its top level.
func ternaryBranches(rhs string) ([]string, bool) {
	// Nearly every right-hand side asked about holds no `?` at all.
	if strings.IndexByte(rhs, '?') < 0 {
		return nil, false
	}

	tokens := significantTokens(rhs)

	q, colon := parser.ConditionalAt(tokens)
	if q <= 0 || colon < 0 {
		return nil, false
	}

	var a, b string

	if colon == q+1 {
		// `a ?: b`: a when it is defined, b otherwise.
		a, b = rhs[:tokens[q].Offset], rhs[tokens[colon].Offset+1:]
	} else {
		a, b = rhs[tokens[q].Offset+1:tokens[colon].Offset], rhs[tokens[colon].Offset+1:]
	}

	var out []string

	for _, branch := range []string{a, b} {
		branch = unwrapParens(strings.TrimSpace(branch))
		if branch == "" {
			return nil, false
		}

		if nested, ok := ternaryBranches(branch); ok {
			out = append(out, nested...)
		} else {
			out = append(out, branch)
		}
	}

	if len(out) > maxTernaryBranches {
		return nil, false
	}

	return out, true
}

// sameVariable reports whether branch names variable, either spelling with
// or without the variables scope.
func sameVariable(branch, variable string) bool {
	strip := func(s string) string {
		if len(s) > 10 && strings.EqualFold(s[:10], "variables.") {
			return s[10:]
		}

		return s
	}

	return strings.EqualFold(strip(branch), strip(variable))
}

// unwrapParens is s without parentheses enclosing the whole of it.
func unwrapParens(s string) string {
	for strings.HasPrefix(s, "(") && closingParen(s) == len(s)-1 {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}

	return s
}

// assignedTernary is the union of what each branch of a conditional assigned
// to variable holds at line, or "" when a branch holds nothing known.
func (r *Resolver) assignedTernary(variable string, branches []string, line uint32, caller string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) string {
	var comps []string

	dynamic, unknown := false, false

	for _, branch := range branches {
		// A null holds no component, and a branch naming the variable keeps
		// what it already held (`x = isNull( x ) ? new X() : x`): neither
		// says what the assignment sets it to.
		if isNullCast(branch) || sameVariable(branch, variable) {
			continue
		}

		comp := r.ternaryBranch(branch, line, caller, pr, baseDir, ctx)

		switch {
		case comp == "$any":
			dynamic = true
		case comp == "" || strings.HasPrefix(comp, "$"):
			unknown = true
		default:
			comps = append(comps, strings.Split(comp, "|")...)
		}
	}

	// A branch nothing is known of leaves the conditional unknown, unless
	// another is dynamic: whatever the first holds, the dynamic one covers it.
	if unknown && !dynamic || len(comps) == 0 && !dynamic {
		return ""
	}

	comps = r.distinctComponents(comps, baseDir)

	answer := "$any"

	if !dynamic {
		slices.Sort(comps)
		answer = strings.Join(comps, "|")
	}

	tr.addf("resolved %q to %q: it is assigned a conditional, each of whose %d branches is typed", variable, answer, len(branches))

	return answer
}

// ternaryBranch is what one branch of a conditional holds: a construction,
// a name typed as any receiver is at line, or a chain of calls.
func (r *Resolver) ternaryBranch(branch string, line uint32, caller string, pr *parser.ParseResult, baseDir string, ctx lookupCtx) string {
	for _, read := range [...]func(string) []string{createdComponent, newComponent} {
		if m := read(branch); m != nil {
			return r.createdThenCalled(r.staticPath(m[1]), branch, baseDir)
		}
	}

	if isIdent(branch) || isDottedName(branch) {
		comp, _ := r.receiverComponentD(branch, line, caller, "", pr, baseDir, nil, lookupCtx{depth: ctx.depth + 1, leaf: ctx.leaf, callers: ctx.callers})

		return comp
	}

	return r.typeCallExpr(branch, line, caller, pr, baseDir, lookupCtx{depth: ctx.depth + 1, leaf: ctx.leaf, callers: ctx.callers}, true)
}

// distinctComponents is comps with each component once: two spellings of one
// file (a dotted path and the file a caller-typed argument names) are one
// alternative, written as the dotted path.
func (r *Resolver) distinctComponents(comps []string, baseDir string) []string {
	var out []string

	seen := map[string]int{}

	for _, c := range comps {
		key := strings.ToLower(c)
		if p := r.ComponentPath(c, baseDir); p != "" {
			key = pathKey(p)
		}

		if i, ok := seen[key]; ok {
			if filepath.IsAbs(out[i]) && !filepath.IsAbs(c) {
				out[i] = c
			}

			continue
		}

		seen[key] = len(out)
		out = append(out, c)
	}

	return out
}
