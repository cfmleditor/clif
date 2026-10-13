package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Two cborm facts the stubs cannot state, both from cborm's BaseBuilder and
// CriteriaBuilder source:
//
//   - `when( test, target )` hands its closure the current builder: "the
//     closure to execute if test is true, it receives the current criteria as
//     the argument". ContentBox writes `newCriteria().when( len( search ),
//     function( c ){ c.$or( c.restrictions.like( … ) ) } )`, and the closure's
//     parameter has no declared type to read.
//   - a builder's `restrictions` is a cborm.models.criterion.Restrictions
//     (this.restrictions, assigned from an argument the stub cannot follow).
//
// The closure rule applies only to a closure written as an argument of `when`
// on a chain that is still a builder: newCriteria() followed by builder
// methods, as isNewCriteriaChain reads one, or a local assigned such a chain.

const (
	builderComponent      = "cborm.models.criterion.CriteriaBuilder"
	restrictionsComponent = "cborm.models.criterion.Restrictions"
)

// closureSpan is a function literal with parameters, the lines of its body and
// whether it is the target of a builder's when().
type closureSpan struct {
	params       []string // lowercased
	start, end   int      // body lines
	from, to     int      // byte interval, end exclusive
	functionFrom int
	builder      bool
}

// closureParamBuilder is the builder component when name is the first
// parameter of the innermost closure around line that is a builder chain's
// when() target, or "".
func (r *Resolver) closureParamBuilder(name string, line uint32, pr *parser.ParseResult, call *parser.CallSite) string {
	name = strings.ToLower(strings.TrimPrefix(strings.ToLower(name), "arguments."))
	if name == "" || strings.Contains(name, ".") {
		return ""
	}

	var best *closureSpan

	spans := r.closuresOf(pr)

	for i := range spans {
		s := &spans[i]
		if int(line) < s.start || int(line) > s.end {
			continue
		}

		if call != nil && call.HasSpan && (int(call.StartByte) < s.from || int(call.StartByte) >= s.to) {
			continue
		}

		if best == nil || s.to-s.from < best.to-best.from {
			best = s
		}
	}
	// A line-only request cannot identify sibling or nested same-line closures.
	if best != nil && (call == nil || !call.HasSpan) {
		for _, s := range spans {
			if int(line) >= s.start && int(line) <= s.end && s.from != best.from &&
				(s.to <= best.from || s.from >= best.to || s.start == best.start && s.end == best.end) {
				return ""
			}
		}
	}

	if best == nil || !best.builder || len(best.params) == 0 || best.params[0] != name {
		return ""
	}

	if call != nil && call.HasSpan && !callbackParameterStable(pr.Content, best.from, int(call.StartByte), name) {
		return ""
	}

	return builderComponent
}

// isBuilderComponent reports whether comp is cborm's criteria builder or one
// that extends it.
func isBuilderComponent(comp string) bool {
	if comp == "" {
		return false
	}

	for alt := range strings.SplitSeq(comp, "|") {
		p := strings.ToLower(filepath.ToSlash(alt))
		p = strings.ReplaceAll(p, ".", "/")

		if !strings.HasSuffix(p, "criterion/criteriabuilder") && !strings.HasSuffix(p, "criterion/basebuilder") &&
			!strings.HasSuffix(p, "criterion/detachedcriteriabuilder") && !strings.HasSuffix(p, "criterion/criteriabuilder.cfc") &&
			!strings.HasSuffix(p, "criterion/basebuilder.cfc") && !strings.HasSuffix(p, "criterion/detachedcriteriabuilder.cfc") {
			return false
		}
	}

	return true
}

// closuresOf stays per view, unlike the loop and guard spans: whether a
// closure is a builder's is resolved under the view's mappings.
func (r *Resolver) closuresOf(pr *parser.ParseResult) []closureSpan {
	return r.closureCache.get(&r.mu, pr, func() []closureSpan { return r.findClosures(pr, r.fileTokens(pr)) })
}

func (r *Resolver) findClosures(pr *parser.ParseResult, tokens []parser.Token) []closureSpan {
	var out []closureSpan

	for i := 0; i+2 < len(tokens); i++ {
		body, paramsFrom, paramsTo := -1, -1, -1

		switch {
		case strings.EqualFold(tokens[i].Value, "function") && tokens[i+1].Kind == parser.TokLParen:
			endParams := producerGroupEnd(tokens, i+1, parser.TokLParen, parser.TokRParen)
			if endParams >= 0 && endParams+1 < len(tokens) && tokens[endParams+1].Kind == parser.TokLBrace {
				body, paramsFrom, paramsTo = endParams+1, i+2, endParams
			}
		case tokens[i].Kind == parser.TokLParen:
			endParams := producerGroupEnd(tokens, i, parser.TokLParen, parser.TokRParen)
			if endParams >= 0 && endParams+3 < len(tokens) && tokens[endParams+1].Kind == parser.TokEquals && tokens[endParams+2].Kind == parser.TokGT && tokens[endParams+3].Kind == parser.TokLBrace {
				body, paramsFrom, paramsTo = endParams+3, i+1, endParams
			}
		case tokens[i].Kind == parser.TokIdent && tokens[i+1].Kind == parser.TokEquals && tokens[i+2].Kind == parser.TokGT && i+3 < len(tokens) && tokens[i+3].Kind == parser.TokLBrace:
			if i == 0 || tokens[i-1].Kind != parser.TokDot {
				body, paramsFrom, paramsTo = i+3, i, i+1
			}
		}

		if body < 0 {
			continue
		}

		closeBrace := producerGroupEnd(tokens, body, parser.TokLBrace, parser.TokRBrace)
		if closeBrace < 0 {
			continue
		}

		span := closureSpan{start: tokens[body].Line, end: tokens[closeBrace].Line, from: tokens[body].Offset, to: tokens[closeBrace].Offset + 1, functionFrom: tokens[i].Offset}
		span.params = paramNames(tokens[paramsFrom:paramsTo])
		span.builder = r.isBuilderWhenTarget(tokens, i, pr, span.start)

		out = append(out, span)
	}

	return out
}

// paramNames are the parameter names of a function's parameter list: in each
// comma-separated piece, the identifier before a default's `=`, or the last.
func paramNames(tokens []parser.Token) []string {
	var out []string

	for _, piece := range producerSplit(tokens) {
		name := ""

		for _, t := range piece {
			if t.Kind == parser.TokEquals {
				break
			}

			if t.Kind == parser.TokIdent {
				name = t.Value
			}
		}

		if name != "" {
			out = append(out, strings.ToLower(name))
		}
	}

	return out
}

// isBuilderWhenTarget reports whether the function literal at tokens[fn] is an
// argument of a when() call whose receiver is a builder chain.
func (r *Resolver) isBuilderWhenTarget(tokens []parser.Token, fn int, pr *parser.ParseResult, bodyLine int) bool {
	// Walk back to the unmatched `(` of the call the literal is an argument of.
	depth := 0
	open := -1

	for j := fn - 1; j >= 0 && open < 0; j-- {
		switch tokens[j].Kind {
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth++
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			if depth == 0 {
				if tokens[j].Kind != parser.TokLParen {
					return false
				}

				open = j
			} else {
				depth--
			}
		case parser.TokSemicolon:
			if depth == 0 {
				return false
			}
		default:
		}
	}

	if open < 2 || !strings.EqualFold(tokens[open-1].Value, "when") || tokens[open-2].Kind != parser.TokDot {
		return false
	}

	return r.builderChainEndingAt(tokens, open-2, pr, bodyLine)
}

// builderChainEndingAt reports whether the chain written before the dot at
// tokens[dot] is a builder: newCriteria() and builder methods, or a local
// assigned one.
func (r *Resolver) builderChainEndingAt(tokens []parser.Token, dot int, pr *parser.ParseResult, line int) bool {
	end := dot // exclusive end of the chain

	start := end

	for start > 0 {
		j := start - 1

		switch tokens[j].Kind {
		case parser.TokRParen:
			depth := 0

			for ; j >= 0; j-- {
				if tokens[j].Kind == parser.TokRParen {
					depth++
				} else if tokens[j].Kind == parser.TokLParen {
					depth--
					if depth == 0 {
						break
					}
				}
			}

			if j < 1 || tokens[j-1].Kind != parser.TokIdent {
				return false
			}

			start = j - 1
		case parser.TokIdent:
			start = j
		default:
			return false
		}

		if start > 0 && tokens[start-1].Kind == parser.TokDot {
			start--

			continue
		}

		break
	}

	chain := tokens[start:end]
	if isNewCriteriaChain(chain, r.isBuilderMethod) {
		return true
	}

	// A local holding a builder: `var c = newCriteria(); c.when( … )`.
	if len(chain) == 1 && chain[0].Kind == parser.TokIdent {
		if rhs, ok := r.localAssignmentIn(pr, chain[0].Value, 0, line); ok {
			return isNewCriteriaChain(producerTokens(rhs), r.isBuilderMethod)
		}
	}

	return false
}

// builderMember types a closure's builder parameter (`c` in `when( test,
// function( c ){ … } )`) and a builder's `restrictions`.
func (r *Resolver) builderMember(variable string, line uint32, caller, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) string {
	if pr == nil || strings.ContainsAny(variable, "[(") {
		return ""
	}

	if root, member, ok := strings.Cut(variable, "."); ok && strings.EqualFold(member, "restrictions") && !isScopeWord(root) {
		comp, _ := r.receiverComponentD(root, line, caller, funcName, pr, baseDir, nil, ctx)
		if isBuilderComponent(comp) {
			tr.addf("resolved %q to %q: a criteria builder's restrictions", variable, restrictionsComponent)

			return restrictionsComponent
		}

		return ""
	}

	if comp := r.closureParamBuilder(variable, line, pr, ctx.call); comp != "" {
		tr.addf("resolved %q to %q: the builder when() hands its closure", variable, comp)

		return comp
	}

	if comp := r.callbackParameterComponent(variable, pr, baseDir, ctx); comp != "" {
		tr.addf("resolved %q to %q from the provider's callback invocation", variable, comp)

		return comp
	}

	return ""
}
