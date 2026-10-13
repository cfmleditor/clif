package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
)

// Preservation follows the complete expression, never a regex prefix. The
// source must already have a collection element contract. Map, erased returns,
// properties, and unrecognized operations do not preserve that contract.
func (r *Resolver) collectionExpressionElement(receiver string, calls []chainCall, header, start int, caller string, pr *parser.ParseResult, baseDir string, depth int, leaf string) string {
	last := calls[len(calls)-1]
	if strings.EqualFold(last.name, "valueArray") && strings.TrimSpace(last.args) == "" {
		return r.structCollectionValue(collectionExpression(receiver, calls[:len(calls)-1]), header, start, caller, pr, baseDir, depth+1, leaf)
	}

	if strings.EqualFold(last.name, "keyArray") && strings.TrimSpace(last.args) == "" && r.collectionExpressionComponent(receiver, calls[:len(calls)-1], header, caller, pr, baseDir, leaf) == nativetype.Type("struct") {
		return nativetype.Type("string")
	}

	if preservesArrayElements(last.name) && (receiver != "" || len(calls) > 1) {
		return r.elementOf(collectionExpression(receiver, calls[:len(calls)-1]), header, start, caller, pr, baseDir, depth+1, leaf)
	}

	if len(calls) != 1 {
		return ""
	}

	self := pr.URI.Path()
	if receiver == "" && !pr.Declares(last.name) && r.ResolveFunc(self, last.name, baseDir) == nil {
		if strings.EqualFold(last.name, "structValueArray") {
			pieces := producerSplit(producerTokens(last.args))
			if len(pieces) == 1 {
				return r.structCollectionValue(producerText(pieces[0]), header, start, caller, pr, baseDir, depth+1, leaf)
			}
		}

		if strings.EqualFold(last.name, "structKeyArray") && strings.TrimSpace(last.args) != "" && len(producerSplit(producerTokens(last.args))) == 1 {
			return nativetype.Type("string")
		}

		if strings.EqualFold(last.name, "entityLoad") {
			return r.loadedEntityElement(last.args, baseDir)
		}

		if globalPreservesArrayElements(last.name) {
			pieces := producerSplit(producerTokens(last.args))
			if len(pieces) > 0 {
				return r.elementOf(producerText(pieces[0]), header, start, caller, pr, baseDir, depth+1, leaf)
			}
		}
	}

	target := self
	if receiver != "" && !strings.EqualFold(receiver, "this") && !strings.EqualFold(receiver, "variables") {
		target, _ = r.receiverComponentD(receiver, conv.Uint32(header), caller, last.name, pr, baseDir, nil, lookupCtx{leaf: leaf})

		// A receiver the call itself is accepted on as dynamic hands back a
		// dynamic collection: ContentBox's 1.0.4 patch loops over
		// `pageService.getAll()` from `pageService@cb`, a namespace the current
		// modules no longer declare, and its loop variable was reported as
		// untyped where every call on pageService was accepted.
		if target == "$any" || r.uninstalledModule(target) {
			return "$any"
		}
	}

	return r.methodElement(target, last.name, last.args, baseDir)
}

func (r *Resolver) structCollectionValue(expression string, header, start int, caller string, pr *parser.ParseResult, baseDir string, depth int, leaf string) string {
	if depth > maxElementHops || expression == "" {
		return ""
	}

	if match := loopPathRe.FindStringSubmatch(expression); match != nil && (match[1] == "" || strings.EqualFold(match[1], "local")) {
		if rhs, assigned, ok := r.localAssignmentAt(pr, match[2], start, header); ok {
			if !collectionLocalUntouched(pr.Content, match[2], assigned, header) {
				return ""
			}

			return r.structCollectionValue(rhs, header, start, caller, pr, baseDir, depth+1, leaf)
		}
	}

	receiver, calls, ok := callChain(expression)
	if !ok {
		return ""
	}

	comp := r.collectionExpressionComponent(receiver, calls[:len(calls)-1], header, caller, pr, baseDir, leaf)

	fd := r.ResolveFunc(comp, calls[len(calls)-1].name, baseDir)
	if fd == nil || fd.CollectionValues == nil {
		return ""
	}

	budget := 128

	return r.collectionSourcesOf(fd, fd.CollectionValues.Sources, 0, &budget)
}

// Withhold value contracts after any intervening use of a local container.
// This conservative boundary includes aliases, escapes and element writes;
// earlier read-only uses can be admitted when their lifetime is modeled.
func collectionLocalUntouched(content, name string, assigned, header int) bool {
	for line := range strings.Lines(sourceText(content, assigned+1, header)) {
		for _, token := range producerTokens(line) {
			if token.Kind == parser.TokString && len(parser.InterpolatedExpressions(token.Value)) > 0 {
				return false
			}

			if token.Kind == parser.TokIdent && strings.EqualFold(token.Value, name) {
				return false
			}
		}
	}

	at, ok := lineOffset(content, header)
	if !ok {
		return false
	}

	headerLine, _, _ := strings.Cut(content[at:], "\n")

	tokens := producerTokens(headerLine)
	for i, token := range tokens {
		if token.Kind == parser.TokString && len(parser.InterpolatedExpressions(token.Value)) > 0 {
			return false
		}

		if token.Kind != parser.TokIdent || !strings.EqualFold(token.Value, name) {
			continue
		}

		if i+3 < len(tokens) && tokens[i+1].Kind == parser.TokDot && tokens[i+3].Kind == parser.TokLParen {
			switch strings.ToLower(tokens[i+2].Value) {
			case "each", "filter", "map", "some", "every", "valuearray":
				continue
			}
		}

		if i >= 2 && tokens[i-1].Kind == parser.TokLParen && (collectionBuiltin(tokens[i-2].Value) == "struct" || strings.EqualFold(tokens[i-2].Value, "structValueArray")) {
			continue
		}

		return false
	}

	return true
}

// Resolve the complete producer before applying a native element contract.
// In particular a speculative family, a custom method with the same spelling,
// and a failed intermediate call cannot establish the native operation.
func (r *Resolver) collectionExpressionComponent(receiver string, calls []chainCall, header int, caller string, pr *parser.ParseResult, baseDir, leaf string) string {
	key := SpeculativeReceiver{File: pr.URI.Path(), Caller: caller, Variable: strings.ToLower(receiver)}
	if r.SpeculativeReceivers[key] != "" {
		return ""
	}

	call := &parser.CallSite{Variable: receiver, Caller: caller, Line: conv.Uint32(header), FuncName: "keyArray"}

	comp := pr.URI.Path()
	if receiver != "" {
		comp, _ = r.receiverComponentD(receiver, call.Line, caller, call.FuncName, pr, baseDir, nil, lookupCtx{leaf: leaf, call: call})
		if comp == "" || comp == "$any" {
			switch strings.ToLower(receiver) {
			case "request", "application", "session", "server", "url", "form", "cgi", "cookie", "client", "local", "arguments", "variables":
				comp = nativetype.Type("struct")
			}
		}
	} else if len(calls) > 0 && !pr.Declares(calls[0].name) && r.ResolveFunc(comp, calls[0].name, baseDir) == nil {
		comp = r.BuiltinReturnLookup(calls[0].name)
		calls = calls[1:]
	}

	if comp == "" {
		return ""
	}

	for _, hop := range calls {
		call.Chain = append(call.Chain, parser.CallHop(hop.name+"("+hop.args+")"))
	}

	comp, _, _, done := r.walkHops(comp, "", call, pr, baseDir, nil)
	if done {
		return ""
	}

	return comp
}

func preservesArrayElements(method string) bool {
	switch strings.ToLower(method) {
	case "filter", "slice", "reverse", "duplicate":
		return true
	}

	return false
}

func globalPreservesArrayElements(method string) bool {
	switch strings.ToLower(method) {
	case "arrayfilter", "arrayslice", "arrayreverse", "duplicate":
		return true
	}

	return false
}

func collectionExpression(receiver string, calls []chainCall) string {
	var expression strings.Builder
	expression.WriteString(receiver)

	for _, call := range calls {
		if expression.Len() > 0 {
			expression.WriteByte('.')
		}

		expression.WriteString(call.name)
		expression.WriteByte('(')
		expression.WriteString(call.args)
		expression.WriteByte(')')
	}

	return expression.String()
}

func callbackCollectionExpression(provider *parser.CallSite) string {
	calls := make([]chainCall, 0, len(provider.Chain))
	for _, hop := range provider.Chain {
		if _, property := parser.PropertyName(hop); property {
			return ""
		}

		if expression, ok := parser.CallExpression(hop); ok {
			_, parsed, valid := callChain(expression)
			if !valid || len(parsed) != 1 {
				return ""
			}

			calls = append(calls, parsed[0])
		} else {
			calls = append(calls, chainCall{name: parser.CallHopName(hop)})
		}
	}

	return collectionExpression(provider.Variable, calls)
}
