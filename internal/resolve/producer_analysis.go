package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// producerUnwrap is tokens without the parentheses around the whole of them:
// `return( this );` is how jsonSerializer, and many a CFML style guide, writes
// `return this;`.
func producerUnwrap(tokens []parser.Token) []parser.Token {
	for len(tokens) > 2 && tokens[0].Kind == parser.TokLParen && producerGroupEnd(tokens, 0, parser.TokLParen, parser.TokRParen) == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}

	return tokens
}

func producerReturnsThis(nodes []producerNode) bool {
	found := false

	for i := range nodes {
		node := &nodes[i]
		if node.kind == "return" {
			if producerPath(producerUnwrap(producerTokens(node.expression))) != "this" {
				return false
			}

			found = true
		}

		for _, body := range [][]producerNode{node.body, node.alternative} {
			if producerHasReturns(body) {
				if !producerReturnsThis(body) {
					return false
				}

				found = true
			}
		}
	}

	return found
}

func producerHasReturns(nodes []producerNode) bool {
	for i := range nodes {
		n := &nodes[i]
		if n.kind == "return" || producerHasReturns(n.body) || producerHasReturns(n.alternative) {
			return true
		}
	}

	return false
}

func producerCallText(tokens []parser.Token) string {
	var result strings.Builder

	for i, t := range tokens {
		if i > 0 && t.Kind != parser.TokDot && t.Kind != parser.TokLParen && tokens[i-1].Kind != parser.TokDot {
			result.WriteByte(' ')
		}

		result.WriteString(t.Value)
	}

	return result.String()
}

// A bounded backward slice avoids evaluating bindings that cannot affect the
// returned identity or a guard/output name. Missing dependencies stay unknown.
func producerWanted(nodes []producerNode) map[string]bool {
	wanted := map[string]bool{}

	var seed func([]producerNode)

	seed = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "query" || node.kind == "return" || node.kind == "if" || node.kind == "switch" {
				producerReadPaths(node.expression, wanted)
			}

			seed(node.body)
			seed(node.alternative)
		}
	}
	seed(nodes)

	var visit func([]producerNode)

	visit = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "set" && producerWantedTarget(node.target, wanted) {
				producerReadPaths(node.expression, wanted)
			}

			visit(node.body)
			visit(node.alternative)
		}
	}

	for range 16 {
		count := len(wanted)

		visit(nodes)

		if len(wanted) == count {
			break
		}
	}

	return wanted
}

func producerReadPaths(expression string, wanted map[string]bool) {
	tokens := producerTokens(expression)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent {
			continue
		}

		start := i
		for i+2 < len(tokens) && tokens[i+1].Kind == parser.TokDot && tokens[i+2].Kind == parser.TokIdent {
			i += 2
		}

		path := producerPath(tokens[start : i+1])
		if path != "" {
			wanted[normalizeProducerPath(path)] = true
		}
	}
}

func producerWantedTarget(target string, wanted map[string]bool) bool {
	target = normalizeProducerPath(target)
	for path := range wanted {
		if path == target || strings.HasPrefix(path, target+".") || strings.HasPrefix(target, path+".") {
			return true
		}

		if strings.TrimPrefix(target, "arguments.") == path || strings.TrimPrefix(target, "variables.") == path {
			return true
		}
	}

	return false
}

// A call-derived scalar return must not inherit the parser's closed-file
// receiver guess. Collection contracts remain handled by their uniform writers.
func producerCallBinding(nodes []producerNode) bool {
	wanted := map[string]bool{}

	var seed func([]producerNode)

	seed = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "return" {
				path := producerPath(producerTokens(node.expression))
				if path != "" && path != "this" {
					wanted[normalizeProducerPath(path)] = true
				}
			}

			seed(node.body)
			seed(node.alternative)
		}
	}
	seed(nodes)

	var expand func([]producerNode)

	expand = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "set" && producerWantedTarget(node.target, wanted) {
				producerReadPaths(node.expression, wanted)
			}

			expand(node.body)
			expand(node.alternative)
		}
	}
	stable := false

	for range 16 {
		count := len(wanted)

		expand(nodes)

		if len(wanted) == count {
			stable = true

			break
		}
	}

	if !stable {
		return true
	}

	var visit func([]producerNode) bool

	visit = func(nodes []producerNode) bool {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "set" && producerWantedTarget(node.target, wanted) && strings.Contains(node.expression, "(") && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(node.expression)), "new ") {
				return true
			}

			if visit(node.body) || visit(node.alternative) {
				return true
			}
		}

		return false
	}

	return visit(nodes)
}

// Scalar-call specialization replaces component guesses, not the established
// builtin or dynamic contracts. Object guards still need per-call proof.
func producerNeedsSpecialization(method *producerMethod, fd *parser.FunctionDef) bool {
	if method == nil {
		return false
	}

	if fd.DocReturn == "" && fd.ReturnComponent != "$any" && !strings.HasPrefix(fd.ReturnComponent, "$builtin.") {
		for i := range method.body {
			node := &method.body[i]
			if node.kind != "return" {
				continue
			}

			tokens := producerUnwrap(producerTokens(node.expression))
			if len(tokens) == 0 || !strings.EqualFold(tokens[0].Value, "new") {
				continue
			}

			for j, token := range tokens {
				if token.Kind != parser.TokLParen {
					continue
				}

				end := producerGroupEnd(tokens, j, parser.TokLParen, parser.TokRParen)
				if end >= 0 && end+3 < len(tokens) && tokens[end+1].Kind == parser.TokDot && tokens[end+2].Kind == parser.TokIdent && tokens[end+3].Kind == parser.TokLParen {
					return true
				}

				break
			}
		}
	}

	if !method.sensitive {
		return false
	}

	return producerSensitive(method.body) || fd.ReturnComponent != "$any" && !strings.HasPrefix(fd.ReturnComponent, "$builtin.")
}

// branchUnionCallReturn is the union the parse built for fd's return, for a
// call of it that evaluating the body could not decide: Mura's
// feedBean.getIterator() gets a beanIterator or a contentIterator on its
// countOnly argument and on the feed's own state, which no call shows.
//
// The union names every branch reaching the body's only return
// (branchUnionReturn), so it holds whichever the call takes. It is offered
// only when every argument the call passes is a literal: the call is then
// known in full and what decides the branch is something else. A call passing
// a value it does not spell may be one evaluation would decide, so the union
// would be a guess at a branch it could have named.
func (r *Resolver) branchUnionCallReturn(fd *parser.FunctionDef, expression string) string {
	if fd == nil || !literalArguments(expression) || !branchUnionReturn(r.producerFor(fd), fd) {
		return ""
	}

	alternatives := strings.Split(fd.ReturnComponent, "|")
	for i, alt := range alternatives {
		if p := r.besideDeclaring(fd, alt); p != "" {
			alternatives[i] = p
		}
	}

	slices.Sort(alternatives)

	return strings.Join(slices.Compact(alternatives), "|")
}

// literalArguments reports whether the call expression passes nothing but
// literals: strings without interpolation, numbers and booleans, by name or
// by position.
func literalArguments(expression string) bool {
	tokens := significantTokens(expression)

	open := slices.IndexFunc(tokens, func(t parser.Token) bool { return t.Kind == parser.TokLParen })
	if open < 0 {
		return false
	}

	end := producerGroupEnd(tokens, open, parser.TokLParen, parser.TokRParen)
	if end < 0 || end != len(tokens)-1 {
		return false
	}

	if end == open+1 {
		return true
	}

	for _, piece := range producerSplit(tokens[open+1 : end]) {
		if len(piece) == 3 && piece[0].Kind == parser.TokIdent && piece[1].Kind == parser.TokEquals {
			piece = piece[2:]
		}

		if len(piece) != 1 || !literalToken(piece[0]) {
			return false
		}
	}

	return true
}

func literalToken(t parser.Token) bool {
	switch t.Kind {
	case parser.TokString:
		return !strings.Contains(t.Value, "#")
	case parser.TokNumber:
		return true
	case parser.TokIdent:
		return strings.EqualFold(t.Value, "true") || strings.EqualFold(t.Value, "false")
	default:
		return false
	}
}

// branchUnionReturn reports whether fd's inferred return is a union of
// components the parse built from the branches reaching the body's only
// return, which the plan reads whole, of a local the body declares: then the
// union names every component the return can be. A second return, or a
// statement the plan could not read, could return something the union does
// not hold; so could a field, which code outside the function writes too.
func branchUnionReturn(method *producerMethod, fd *parser.FunctionDef) bool {
	if method == nil || !strings.Contains(fd.ReturnComponent, "|") {
		return false
	}

	for alt := range strings.SplitSeq(fd.ReturnComponent, "|") {
		if alt == "" || strings.HasPrefix(alt, "$") {
			return false
		}
	}

	var (
		returned []string
		locals   = map[string]bool{}
	)

	var walk func([]producerNode) bool

	walk = func(nodes []producerNode) bool {
		for i := range nodes {
			node := &nodes[i]

			switch node.kind {
			case "unsafe":
				return false
			case "return":
				returned = append(returned, strings.ToLower(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(node.expression)), "local.")))
			case "set":
				if name, ok := strings.CutPrefix(strings.ToLower(node.target), "local."); ok {
					locals[name] = true
				}
			}

			if !walk(node.body) || !walk(node.alternative) {
				return false
			}
		}

		return true
	}

	return walk(method.body) && len(returned) == 1 && locals[returned[0]]
}
