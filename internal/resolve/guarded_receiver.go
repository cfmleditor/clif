package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// A positive, exact builtin check establishes a receiver even when its incoming
// type is unknown. The checked value must survive unchanged up to this call.
// Disjunctions, negative/alternative branches and opaque callback effects do
// not establish this contract.
func (r *Resolver) guardedReceiver(variable string, pr *parser.ParseResult, baseDir string, ctx lookupCtx) string {
	if pr == nil || ctx.call == nil || !ctx.call.HasSpan || int(ctx.call.StartByte) > len(pr.Content) {
		return ""
	}

	name := strings.ToLower(variable)

	name = strings.TrimPrefix(strings.TrimPrefix(name, "arguments."), "local.")
	if strings.ContainsAny(name, "[(") {
		return ""
	}

	at := int(ctx.call.StartByte)

	o := r.owner()

	spans := o.typeGuardCache.get(&o.mu, pr, func() []receiverTypeGuard {
		// Every guard is an isInstanceOf() check: a file that never names it
		// has none, and is not tokenized to find that out.
		if parser.IndexFold(pr.Content, "isinstanceof") < 0 {
			return nil
		}

		return receiverTypeGuards(o.fileTokens(pr))
	})
	for _, guard := range spans {
		if !strings.EqualFold(guard.variable, variable) || at < guard.from || at >= guard.to || !guardValueStable(pr.Content, guard.from, at, name) {
			continue
		}

		if r.bareFunc("isInstanceOf", pr, baseDir, ctx.call.Caller) != nil {
			return ""
		}

		if argumentOf(pr, ctx.call.Caller, "isInstanceOf") != nil {
			return ""
		}

		scope := parser.FindFuncScopeAt(int(ctx.call.Line), pr.Scopes)
		for _, local := range pr.FuncVars(max(0, scope.Start), scope.End) {
			if strings.EqualFold(local, "isInstanceOf") {
				return ""
			}
		}

		if path := r.ComponentPath(guard.component, baseDir); path != "" {
			return path
		}
	}

	return ""
}

func guardValueStable(content string, from, to int, name string) bool {
	return guardValueStableDepth(content, from, to, name, 0)
}

func guardValueStableDepth(content string, from, to int, name string, depth int) bool {
	if depth > 8 {
		return false
	}

	if !guardPathStable(content, from, to, name) {
		return false
	}

	tokens := allTokens(content[from:to])
	for i, token := range tokens {
		if token.Kind == parser.TokString {
			for _, expression := range parser.InterpolatedExpressions(token.Value) {
				if !guardValueStableDepth(expression, 0, len(expression), name, depth+1) {
					return false
				}
			}
		}

		if token.Kind != parser.TokIdent {
			continue
		}

		if strings.EqualFold(token.Value, "function") {
			return false
		}

		if strings.EqualFold(token.Value, "cfinclude") || strings.EqualFold(token.Value, "include") && (i == 0 || tokens[i-1].Kind != parser.TokDot) {
			return false
		}

		if (strings.EqualFold(token.Value, "variables") || strings.EqualFold(token.Value, "this") || strings.EqualFold(token.Value, "caller")) && (i+1 >= len(tokens) || tokens[i+1].Kind != parser.TokDot) {
			return false
		}
	}

	return true
}

type receiverTypeGuard struct {
	variable, component string
	from, to            int
}

func receiverTypeGuards(tokens []parser.Token) []receiverTypeGuard {
	var result []receiverTypeGuard

	// An assignment to the check's name can replace builtin dispatch. This
	// conservative file-level exclusion also covers literal bracket writes.
	for i, token := range tokens {
		if strings.EqualFold(token.Value, "isInstanceOf") && i+1 < len(tokens) && tokens[i+1].Kind == parser.TokEquals {
			return nil
		}

		if token.Kind == parser.TokString && strings.EqualFold(diString(tokens[i:i+1]), "isInstanceOf") && i+2 < len(tokens) && tokens[i+1].Kind == parser.TokRBracket && tokens[i+2].Kind == parser.TokEquals {
			return nil
		}
	}

	for i, token := range tokens {
		if token.Kind != parser.TokIdent {
			continue
		}

		tag := strings.EqualFold(token.Value, "cfif") && i > 0 && tokens[i-1].Kind == parser.TokLT

		script := strings.EqualFold(token.Value, "if") && (i == 0 || tokens[i-1].Kind != parser.TokDot)
		if !tag && !script {
			continue
		}

		start := i + 1
		if script {
			if start >= len(tokens) || tokens[start].Kind != parser.TokLParen {
				continue
			}

			start++
		}

		delimiter := parser.TokGT
		if script {
			delimiter = parser.TokRParen
		}

		end, checks := positiveGuardConjuncts(tokens, start, delimiter)
		if end < 0 {
			continue
		}

		to := -1

		if script {
			if end+1 >= len(tokens) || tokens[end+1].Kind != parser.TokLBrace {
				continue
			}

			closing := producerGroupEnd(tokens, end+1, parser.TokLBrace, parser.TokRBrace)
			if closing >= 0 {
				to = tokens[closing].Offset
			}
		} else {
			to = positiveTagEnd(tokens, end+1)
		}

		for _, check := range checks {
			variable, comp, checkEnd := exactInstanceCheck(tokens, check)
			if comp == "" {
				continue
			}

			from := tokens[checkEnd-1].Offset + 1
			if to > from {
				result = append(result, receiverTypeGuard{variable, comp, from, to})
			}
		}
	}

	return result
}

func exactInstanceCheck(tokens []parser.Token, start int) (variable, component string, endIndex int) {
	if start+1 >= len(tokens) || !strings.EqualFold(tokens[start].Value, "isInstanceOf") || tokens[start+1].Kind != parser.TokLParen {
		return "", "", start
	}

	end := producerGroupEnd(tokens, start+1, parser.TokLParen, parser.TokRParen)
	if end < 0 {
		return "", "", start
	}

	pieces := producerSplit(tokens[start+2 : end])
	if len(pieces) != 2 || producerPath(pieces[0]) == "" {
		return "", "", start
	}

	comp := diString(pieces[1])
	if comp == "" || strings.ContainsAny(comp, "#|@") {
		return "", "", start
	}

	return producerPath(pieces[0]), comp, end + 1
}

// Only exact top-level mandatory conjuncts establish identity. A check buried
// in a call, comparison, negation or disjunction does not.
func positiveGuardConjuncts(tokens []parser.Token, start int, delimiter parser.TokenKind) (int, []int) {
	depth, term := 0, start

	var checks []int

	admit := func(end int) {
		_, comp, next := exactInstanceCheck(tokens, term)
		if comp != "" && next == end {
			checks = append(checks, term)
		}
	}

	for i := start; i < len(tokens); i++ {
		t := tokens[i]
		if t.Kind == delimiter && depth == 0 {
			if i == term {
				return -1, nil
			}

			admit(i)

			return i, checks
		}

		if t.Kind == parser.TokPipe || t.Kind == parser.TokSemicolon || strings.EqualFold(t.Value, "or") || strings.EqualFold(t.Value, "xor") {
			return -1, nil
		}

		if depth == 0 && (strings.EqualFold(t.Value, "and") || t.Kind == parser.TokAmpersand && i+1 < len(tokens) && tokens[i+1].Kind == parser.TokAmpersand) {
			if i == term {
				return -1, nil
			}

			admit(i)

			if t.Kind == parser.TokAmpersand {
				i++
			}

			term = i + 1

			continue
		}

		switch t.Kind {
		case parser.TokLParen, parser.TokLBracket:
			depth++
		case parser.TokRParen, parser.TokRBracket:
			depth--
			if depth < 0 {
				return -1, nil
			}
		case parser.TokLBrace, parser.TokRBrace:
			return -1, nil
		default:
		}
	}

	return -1, nil
}

// Every read of a compound receiver's root must reach the exact checked slot.
// Parent reads/escapes, indexed access and replacements may change that slot.
func guardPathStable(content string, from, to int, name string) bool {
	if from < 0 || to < from || to > len(content) {
		return false
	}

	if !strings.Contains(name, ".") {
		return callbackParameterStable(content, from, to, name)
	}

	parts := strings.Split(name, ".")

	tokens := allTokens(content[from:to])
	for i, t := range tokens {
		if t.Kind != parser.TokIdent || !strings.EqualFold(t.Value, parts[0]) || i > 0 && tokens[i-1].Kind == parser.TokDot {
			continue
		}

		next := i + 1
		for _, part := range parts[1:] {
			if next+1 >= len(tokens) || tokens[next].Kind != parser.TokDot || tokens[next+1].Kind != parser.TokIdent || !strings.EqualFold(tokens[next+1].Value, part) {
				return false
			}

			next += 2
		}

		if next >= len(tokens) || tokens[next].Kind != parser.TokDot {
			return false
		}
	}

	return true
}

func positiveTagEnd(tokens []parser.Token, start int) int {
	depth := 1

	for i := start; i+1 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokLT {
			continue
		}

		next := i + 1

		closing := tokens[next].Kind == parser.TokSlash
		if closing {
			next++
			if next >= len(tokens) {
				return tokens[i].Offset
			}
		}

		if strings.EqualFold(tokens[next].Value, "cfif") {
			if closing {
				depth--
			} else {
				depth++
			}

			if depth == 0 {
				return tokens[i].Offset
			}
		} else if depth == 1 && (strings.EqualFold(tokens[next].Value, "cfelse") || strings.EqualFold(tokens[next].Value, "cfelseif")) {
			return tokens[i].Offset
		}
	}

	return -1
}
