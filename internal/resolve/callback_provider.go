package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// callbackParameterShadows prevents an unknown callback parameter from inheriting
// an outer binding. A new binding inside that callback may still supply a type.
func (r *Resolver) callbackParameterShadows(variable string, ref *parser.ComponentRef, pr *parser.ParseResult, call *parser.CallSite) bool {
	if call == nil || !call.HasSpan {
		return false
	}

	name := strings.TrimPrefix(strings.ToLower(variable), "arguments.")
	if strings.Contains(name, ".") {
		return false
	}

	var owner *closureSpan

	spans := r.closuresOf(pr)
	for i := range spans {
		s := &spans[i]
		if int(call.StartByte) < s.from || int(call.StartByte) >= s.to {
			continue
		}

		for _, parameter := range s.params {
			if strings.EqualFold(parameter, name) && (owner == nil || s.to-s.from < owner.to-owner.from) {
				owner = s
			}
		}
	}

	return owner != nil && (ref == nil || !ref.HasClosureSpan || int(ref.ClosureFrom) < owner.from)
}

// Callback parameters are actual invocation arguments, independently of the
// provider's return value. This contract admits closed straight-line provider
// bodies and local callback aliases. Escapes, branch/loop dispatch and computed
// callbacks remain unknown.
func (r *Resolver) ownedCallbackComponent(variable string, line uint32, caller, funcName string, pr *parser.ParseResult, baseDir string, tr *callTrace, ctx lookupCtx) string {
	if comp := r.callbackParameterComponent(variable, pr, baseDir, ctx); comp != "" {
		return comp
	}
	// Criteria-builder callbacks also have an explicit invocation contract.
	return r.builderMember(variable, line, caller, funcName, pr, baseDir, tr, ctx)
}

func (r *Resolver) callbackParameterComponent(variable string, pr *parser.ParseResult, baseDir string, ctx lookupCtx) string {
	if ctx.call == nil || !ctx.call.HasSpan || ctx.depth > 4 {
		return ""
	}

	name := strings.TrimPrefix(strings.ToLower(variable), "arguments.")
	if strings.Contains(name, ".") {
		return ""
	}

	closure, parameter := r.closureParameter(pr, int(ctx.call.StartByte), name)
	if closure == nil {
		return ""
	}

	if !callbackParameterStable(pr.Content, closure.from, int(ctx.call.StartByte), name) {
		return ""
	}

	provider := pr.InnermostCallAround(closure.functionFrom, closure.to)
	if provider == nil {
		return ""
	}

	if element := r.collectionCallbackElement(provider, closure, parameter, pr, baseDir, ctx); element != "" {
		return element
	}

	if len(provider.Chain) > 0 {
		return ""
	}

	fd, component := r.callbackProviderFunc(provider, pr, baseDir, ctx)
	if fd == nil {
		return ""
	}

	method := r.producerFor(fd)
	if method == nil {
		return ""
	}

	tokens := allTokens(pr.Content[provider.StartByte:provider.EndByte])
	if len(tokens) < 3 || tokens[1].Kind != parser.TokLParen {
		return ""
	}

	pieces := producerSplit(tokens[2 : len(tokens)-1])
	callback := callbackParameterName(pieces, int(provider.StartByte), closure.functionFrom, method.parameters)

	if callback == "" {
		return ""
	}

	budget := 1024
	eval := producerEvaluation{resolver: r, fd: fd, receiver: component, baseDir: baseDir, budget: &budget, callerArguments: true}

	arguments, known := eval.arguments(callbackValueArguments(pieces, int(provider.StartByte), closure.functionFrom), nil)
	if !known {
		return ""
	}

	for i, piece := range pieces {
		key := producerPosition(i)
		if callbackNamedArgument(piece) {
			key, piece = strings.ToLower(piece[0].Value), piece[2:]
		}

		if path := producerPath(piece); path != "" {
			component, _ := r.receiverComponentD(path, provider.Line, provider.Caller, "", pr, baseDir, nil, lookupCtx{depth: ctx.depth + 1, call: provider, callers: ctx.callers})
			if component != "" {
				arguments[key] = eval.concrete(component, baseDir)
			}
		}
	}

	eval.callerArguments = false
	env := eval.bindArguments(fd, method, arguments, true)

	return callbackInvocationComponent(&eval, method.body, env, "arguments."+callback, parameter)
}

// closureParameter is the innermost closure holding the byte at start, and the
// position of the parameter called name in it; a nil closure when the byte is
// in none or the closure has no such parameter.
func (r *Resolver) closureParameter(pr *parser.ParseResult, start int, name string) (closure *closureSpan, parameter int) {
	spans := r.closuresOf(pr)

	for i := range spans {
		s := &spans[i]
		if start >= s.from && start < s.to && (closure == nil || s.to-s.from < closure.to-closure.from) {
			closure = s
		}
	}

	if closure == nil {
		return nil, -1
	}

	parameter = -1

	for i, p := range closure.params {
		if p == name {
			parameter = i
		}
	}

	if parameter < 0 {
		return nil, -1
	}

	return closure, parameter
}

// callbackProviderFunc is the function a call that is handed a callback runs,
// and the component declaring it: a bare call is the file's own or an
// included function, a qualified one is looked up on the receiver's
// component. A nil function when it cannot be placed.
func (r *Resolver) callbackProviderFunc(provider *parser.CallSite, pr *parser.ParseResult, baseDir string, ctx lookupCtx) (fd *parser.FunctionDef, component string) {
	component = provider.Component

	if provider.Variable == "" && component == "" {
		return r.bareFunc(provider.FuncName, pr, baseDir, provider.Caller), pr.URI.Path()
	}

	if component == "" {
		component, _ = r.receiverComponentD(provider.Variable, provider.Line, provider.Caller, provider.FuncName, pr, baseDir, nil, lookupCtx{depth: ctx.depth + 1, call: provider, callers: ctx.callers})
	}

	if component != "" {
		fd = r.ResolveFunc(component, provider.FuncName, baseDir)
	}

	return fd, component
}

// callbackParameterName is the name, in the provider, of the parameter the
// closure starting at functionFrom is passed as: a named argument gives it,
// a positional one is the provider's parameter at that position.
func callbackParameterName(pieces [][]parser.Token, providerStart, functionFrom int, parameters []string) string {
	callback := ""

	for i, piece := range pieces {
		if callbackNamedArgument(piece) {
			if piece[2].Offset+providerStart == functionFrom {
				callback = strings.ToLower(piece[0].Value)
			}
		} else if len(piece) > 0 && piece[0].Offset+providerStart == functionFrom && i < len(parameters) {
			callback = parameters[i]
		}
	}

	return callback
}

// Callback syntax is opaque to value evaluation. Preserve its slot while
// evaluating the other invocation arguments in the ordinary way.
func callbackValueArguments(pieces [][]parser.Token, offset, functionFrom int) []parser.Token {
	var argumentTokens []parser.Token

	for i, piece := range pieces {
		if i > 0 {
			argumentTokens = append(argumentTokens, parser.Token{Kind: parser.TokComma, Value: ","})
		}

		expr := piece

		named := callbackNamedArgument(piece)
		if named {
			expr = piece[2:]
		}

		if len(expr) > 0 && expr[0].Offset+offset == functionFrom {
			if named {
				argumentTokens = append(argumentTokens, piece[:2]...)
			}

			argumentTokens = append(argumentTokens, parser.Token{Kind: parser.TokIdent, Value: "__clif_callback"})
		} else {
			argumentTokens = append(argumentTokens, piece...)
		}
	}

	return argumentTokens
}

func callbackNamedArgument(piece []parser.Token) bool {
	return len(piece) > 2 && piece[0].Kind == parser.TokIdent && (piece[1].Kind == parser.TokColon || piece[1].Kind == parser.TokEquals && piece[2].Kind != parser.TokGT)
}

// A callback's entry argument is evidence only while the parameter still holds
// that value. Reading it as a receiver is harmless; rebinding or handing it (or
// its arguments scope) to unknown code withholds the entry contract.
func callbackParameterStable(content string, from, to int, name string) bool {
	return callbackParameterStableDepth(content, from, to, name, 0)
}

func callbackParameterStableDepth(content string, from, to int, name string, depth int) bool {
	if depth > 8 {
		return false
	}

	if from < 0 || to < from || to > len(content) {
		return false
	}

	tokens := allTokens(content[from:to])
	for i, t := range tokens {
		if t.Kind == parser.TokString {
			for _, expression := range parser.InterpolatedExpressions(t.Value) {
				if !callbackParameterStableDepth(expression, 0, len(expression), name, depth+1) {
					return false
				}
			}
		}

		if t.Kind != parser.TokIdent {
			continue
		}

		if (strings.EqualFold(t.Value, "arguments") || strings.EqualFold(t.Value, "local")) && (i+1 >= len(tokens) || tokens[i+1].Kind != parser.TokDot) {
			return false
		}

		if !strings.EqualFold(t.Value, name) {
			continue
		}

		if i > 0 && tokens[i-1].Kind == parser.TokDot && (i < 2 || !strings.EqualFold(tokens[i-2].Value, "arguments") && !strings.EqualFold(tokens[i-2].Value, "local")) {
			continue
		}

		if i+1 < len(tokens) && tokens[i+1].Kind == parser.TokDot {
			continue
		}

		return false
	}

	return true
}

func callbackReferenceIn(tokens []parser.Token, aliases map[string]bool) bool {
	for i, token := range tokens {
		if token.Kind != parser.TokIdent || i > 0 && tokens[i-1].Kind == parser.TokDot {
			continue
		}

		path := strings.ToLower(token.Value)
		if i+2 < len(tokens) && tokens[i+1].Kind == parser.TokDot && tokens[i+2].Kind == parser.TokIdent {
			path += "." + strings.ToLower(tokens[i+2].Value)
		}

		if aliases[normalizeProducerPath(path)] {
			return true
		}
	}

	return false
}
