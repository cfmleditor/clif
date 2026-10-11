package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
)

// Lucee uses filecontent for text and filecontent_binary for bytes. The field
// contract describes its value when present, not a guarantee of presence.
// Local initializers require explicit scope, a dominating local store, or a
// literal modern-mode named function, plus a bounded, unescaped lifetime.
func (r *Resolver) builtinResultField(variable string, line uint32, caller string, pr *parser.ParseResult, baseDir string, call *parser.CallSite) string {
	if !strings.EqualFold(r.NativeEngine, "lucee") || call == nil {
		return ""
	}

	path := strings.Split(strings.ToLower(variable), ".")
	if len(path) == 3 && path[0] == "local" {
		path = path[1:]
	}

	if len(path) != 2 || path[1] != "filecontent" || isScopeWord(path[0]) {
		return ""
	}

	scope, ok := enclosingScope(pr, line)
	if !ok {
		return ""
	}

	// Anonymous callback locals belong to their byte owner, not the named
	// function's line scope. Sibling test callbacks often reuse result.
	for _, span := range r.closuresOf(pr) {
		if span.from <= int(call.StartByte) && int(call.StartByte) < span.to && span.start >= scope.Start && span.end <= scope.End {
			scope.Start, scope.End = span.start, span.end
		}
	}

	rhs, assigned, ok := localAssignmentAt(pr.Content, path[0], scope.Start, int(line))
	if !ok {
		return ""
	}

	lines := strings.Split(pr.Content, "\n")

	prefix := producerTokens(strings.Join(lines[scope.Start:assigned], "\n"))
	if prefix == nil || !queryOutputStatement(prefix, len(prefix)) {
		return ""
	}

	declaration := strings.ToLower(strings.TrimSpace(lines[assigned]))
	if !internalRequestLocalInitializer(declaration, path[0], scope.Name, prefix) {
		return ""
	}

	recv, calls, ok := callChain(rhs)
	if !ok || recv != "" || len(calls) != 1 {
		return ""
	}

	producer := calls[0].name
	if !strings.EqualFold(producer, "InternalRequest") && !strings.EqualFold(producer, "_InternalRequest") {
		return ""
	}

	if pr.Declares(producer) || argumentOf(pr, caller, producer) != nil || r.bareFunc(producer, pr, baseDir, caller) != nil {
		return ""
	}

	for _, span := range r.closuresOf(pr) {
		if span.from <= int(call.StartByte) && int(call.StartByte) < span.to {
			for _, param := range span.params {
				if strings.EqualFold(param, producer) {
					return ""
				}
			}
		}
	}

	if r.internalRequestCaptured(pr, scope, call, path[0]) {
		return ""
	}

	offset := 0
	for i := range assigned {
		offset += len(lines[i]) + 1
	}

	if int(call.StartByte) <= offset || int(call.StartByte) > len(pr.Content) {
		return ""
	}

	tokens := producerTokens(pr.Content[offset:call.StartByte])
	if tokens == nil {
		return ""
	}

	at := 0
	for at < len(tokens) && tokens[at].Kind != parser.TokEquals {
		at++
	}

	if at+2 >= len(tokens) || !strings.EqualFold(tokens[at+1].Value, producer) || tokens[at+2].Kind != parser.TokLParen {
		return ""
	}

	end := producerGroupEnd(tokens, at+2, parser.TokLParen, parser.TokRParen)
	if end < 0 {
		return ""
	}

	if !internalRequestFieldUntouched(tokens[end+1:], path[0]) {
		return ""
	}

	return nativetype.Type("string")
}

func internalRequestFieldUntouched(tokens []parser.Token, name string) bool {
	depth := 0

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t.Kind {
		case parser.TokLBrace:
			depth++
		case parser.TokRBrace:
			if depth == 0 {
				return false
			}

			depth--
		case parser.TokString:
			if strings.Contains(t.Value, "#") {
				return false
			}
		case parser.TokIdent:
			if strings.EqualFold(t.Value, "function") {
				return false
			}

			if !strings.EqualFold(t.Value, name) {
				continue
			}

			if i > 0 && tokens[i-1].Kind == parser.TokDot && (i < 2 || !strings.EqualFold(tokens[i-2].Value, "local")) {
				return false
			}

			if i+2 >= len(tokens) || tokens[i+1].Kind != parser.TokDot {
				return false
			}

			family := internalRequestScalarField(tokens[i+2].Value)
			if family == "" {
				return false
			}

			if !internalRequestScalarRead(tokens, i+3, family) {
				return false
			}

			i += 2
		default:
		}
	}

	return true
}

// A closure can retain the container and mutate it on a later invocation, even
// when its declaration is outside the initializer-to-lookup interval.
func (r *Resolver) internalRequestCaptured(pr *parser.ParseResult, scope parser.FuncScope, call *parser.CallSite, name string) bool {
	lines := strings.Split(pr.Content, "\n")
	from, to := 0, 0

	for i, line := range lines {
		if i < scope.Start {
			from += len(line) + 1
		}

		if i <= scope.End {
			to += len(line) + 1
		}
	}

	to = min(to, len(pr.Content))

	spans := r.closuresOf(pr)
	for _, span := range spans {
		if span.from <= int(call.StartByte) && int(call.StartByte) < span.to && span.from >= from && span.to <= to {
			from, to = span.from, span.to
		}
	}

	for _, span := range spans {
		if span.from < from || span.to > to || span.from == from && span.to == to {
			continue
		}

		tokens := producerTokens(pr.Content[span.from:span.to])
		if tokens == nil {
			return true
		}

		for _, token := range tokens {
			if token.Kind == parser.TokIdent && strings.EqualFold(token.Value, name) {
				return true
			}
		}
	}

	return internalRequestNamedCapture(producerTokens(pr.Content[from:to]), name, int(call.StartByte)-from)
}

// Status fields and text are scalar values; reading one does not expose the
// mutable result struct. Struct/array fields still need a separate alias proof.
func internalRequestScalarField(field string) string {
	switch strings.ToLower(field) {
	case "filecontent":
		return nativetype.Type("string")
	case "status", "status_code":
		return nativetype.Type("numeric")
	default:
		return ""
	}
}

// Named nested functions can capture a local and be called before their source
// declaration. Do not mistake the enclosing function itself for a capture.
func internalRequestNamedCapture(tokens []parser.Token, name string, lookup int) bool {
	if tokens == nil {
		return true
	}

	for i := 0; i+2 < len(tokens); i++ {
		if !strings.EqualFold(tokens[i].Value, "function") || tokens[i+1].Kind != parser.TokIdent || tokens[i+2].Kind != parser.TokLParen {
			continue
		}

		paramsEnd := producerGroupEnd(tokens, i+2, parser.TokLParen, parser.TokRParen)
		if paramsEnd < 0 {
			return true
		}

		body := paramsEnd + 1
		for body < len(tokens) && tokens[body].Kind != parser.TokLBrace {
			if tokens[body].Kind == parser.TokSemicolon || tokens[body].Kind == parser.TokRBrace {
				return true
			}

			body++
		}

		if body >= len(tokens) {
			return true
		}

		end := producerGroupEnd(tokens, body, parser.TokLBrace, parser.TokRBrace)
		if end < 0 {
			return true
		}

		if tokens[body].Offset <= lookup && lookup < tokens[end].Offset {
			continue
		}

		for _, token := range tokens[body+1 : end] {
			if token.Kind == parser.TokIdent && strings.EqualFold(token.Value, name) {
				return true
			}
		}
	}

	return false
}

// Validate the operation following a scalar property independently of the
// enclosing block scan. Only reads and supported immutable-value members pass.
func internalRequestScalarRead(tokens []parser.Token, next int, family string) bool {
	if next >= len(tokens) {
		return true
	}

	if tokens[next].Kind == parser.TokDot {
		if next+1 == len(tokens) {
			return true
		}

		if next+2 >= len(tokens) || tokens[next+2].Kind != parser.TokLParen {
			return false
		}

		_, supported := nativetype.Method(family, tokens[next+1].Value)

		return supported
	}

	switch tokens[next].Kind {
	case parser.TokRParen, parser.TokComma, parser.TokSemicolon, parser.TokLT, parser.TokGT:
		return true
	case parser.TokEquals:
		return next+1 < len(tokens) && tokens[next+1].Kind == parser.TokEquals
	default:
		return false
	}
}
