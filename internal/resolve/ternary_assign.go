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
	tokens := significantTokens(rhs)

	q := -1
	depth := 0

	for i, t := range tokens {
		switch t.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--
		case parser.TokQuestion:
			if depth == 0 && q < 0 {
				q = i
			}
		default:
		}
	}

	if q <= 0 || q+1 >= len(tokens) {
		return nil, false
	}

	var a, b string

	if tokens[q+1].Kind == parser.TokColon && tokens[q+1].Offset == tokens[q].Offset+1 {
		// `a ?: b`: a when it is defined, b otherwise.
		a, b = rhs[:tokens[q].Offset], rhs[tokens[q+1].Offset+1:]
	} else {
		colon := ternaryColon(tokens, q)
		if colon < 0 {
			return nil, false
		}

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

// ternaryColon is the index of the `:` closing the `?` at q, stepping over
// brackets and over the `? :` of a conditional nested in the first branch,
// or -1.
func ternaryColon(tokens []parser.Token, q int) int {
	depth, pending := 0, 0

	for i := q + 1; i < len(tokens); i++ {
		switch tokens[i].Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--
		case parser.TokQuestion:
			if depth == 0 {
				pending++
			}
		case parser.TokColon:
			if depth != 0 {
				continue
			}

			if pending == 0 {
				return i
			}

			pending--
		default:
		}
	}

	return -1
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

	dynamic := false

	for _, branch := range branches {
		comp := r.ternaryBranch(variable, branch, line, caller, pr, baseDir, ctx)

		switch {
		case comp == "$any":
			dynamic = true
		case comp == "" || strings.HasPrefix(comp, "$"):
			return ""
		default:
			comps = append(comps, strings.Split(comp, "|")...)
		}
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
func (r *Resolver) ternaryBranch(variable, branch string, line uint32, caller string, pr *parser.ParseResult, baseDir string, ctx lookupCtx) string {
	for _, re := range [...]interface{ FindStringSubmatch(string) []string }{createdComponent, newComponent} {
		if m := re.FindStringSubmatch(branch); m != nil {
			return r.createdThenCalled(r.staticPath(m[1]), branch, baseDir)
		}
	}

	if identRe.MatchString(branch) || aliasRe.MatchString(branch) {
		if strings.EqualFold(branch, variable) {
			return ""
		}

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
