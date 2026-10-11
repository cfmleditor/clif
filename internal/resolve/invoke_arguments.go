package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Source-owned invoke calls use ordinary caller inference, even when the
// declaration has the same name as the builtin dispatcher.
func (r *Resolver) builtinInvoke(call *parser.CallSite, pr *parser.ParseResult, dir string) bool {
	return strings.EqualFold(call.FuncName, "invoke") && call.Variable == "" && call.Component == "" && r.bareFunc("invoke", pr, dir, call.Caller) == nil && !r.builtinCallShadowed("invoke", call.Line, call.Caller, pr)
}

// invokeArgument admits positional and named builtin forms with a literal
// method selector. Unknown selectors are outside the caller index's coverage.
// Unknown targets/bags for a matching selector block inference rather than
// borrowing a known caller's argument. Local functions named invoke are ordinary
// calls and cannot establish builtin dispatch.
func (r *Resolver) invokeArgument(call *parser.CallSite, pr *parser.ParseResult, fd *parser.FunctionDef, pos int, name, dir string, ctx lookupCtx) (expr []parser.Token, passed, ours, known bool) {
	if r.bareFunc("invoke", pr, dir, call.Caller) != nil {
		return nil, false, false, true
	}

	if !call.HasSpan || int(call.EndByte) > len(pr.Content) {
		return nil, false, false, false
	}

	tokens := allTokens(pr.Content[call.StartByte:call.EndByte])
	if len(tokens) < 3 || tokens[1].Kind != parser.TokLParen {
		return nil, false, false, true
	}

	for i := range tokens {
		tokens[i].Offset += int(call.StartByte)
		tokens[i].Line += int(call.Line)
	}

	parts := producerSplit(tokens[2 : len(tokens)-1])

	parts, valid := invokeParts(parts)
	if !valid {
		return nil, false, false, false
	}

	if len(parts[1]) != 1 || parts[1][0].Kind != parser.TokString {
		return nil, false, false, true
	}

	method := strings.Trim(parts[1][0].Value, `"'`)
	if strings.Contains(method, "#") || !strings.EqualFold(method, fd.Name) {
		return nil, false, false, true
	}

	target := r.argumentExprComponent(parts[0], call.Line, call.Caller, pr, dir, ctx)
	if len(parts[0]) == 1 && parts[0][0].Kind == parser.TokString {
		literal := strings.Trim(parts[0][0].Value, `"'`)
		if !strings.Contains(literal, "#") {
			if literal == "" {
				target = pr.URI.Path()
			} else {
				target = r.ComponentPath(literal, dir)
			}
		}
	}

	if target == "" {
		return nil, false, false, false
	}

	if !r.componentCallIs(target, method, fd, dir) {
		return nil, false, false, true
	}

	if len(parts[2]) == 0 {
		return nil, false, true, true
	}

	return invokeBagArgument(parts[2], pos, name)
}

// Argument aliases follow the builtin signature. A duplicate semantic slot,
// unknown name or mixture of named and positional arguments has no stable
// transfer here, even when one occurrence happens to contain a known object.
func invokeParts(parts [][]parser.Token) ([][]parser.Token, bool) {
	if len(parts) < 2 || len(parts) > 3 {
		return nil, false
	}

	result := make([][]parser.Token, 3)

	named := len(parts[0]) > 1 && (parts[0][1].Kind == parser.TokEquals || parts[0][1].Kind == parser.TokColon)
	for i, part := range parts {
		isNamed := len(part) > 1 && (part[1].Kind == parser.TokEquals || part[1].Kind == parser.TokColon)
		if named != isNamed {
			return nil, false
		}

		if !named {
			result[i] = part

			continue
		}

		if len(part) < 3 || part[0].Kind != parser.TokIdent {
			return nil, false
		}

		var slot int

		switch strings.ToLower(part[0].Value) {
		case "object", "cfcinstance", "obj", "value", "variable":
			slot = 0
		case "name", "methodname", "method", "functionname", "function":
			slot = 1
		case "arguments", "params", "args", "argumentcollection":
			slot = 2
		default:
			return nil, false
		}

		if result[slot] != nil {
			return nil, false
		}

		result[slot] = part[2:]
	}

	return result, len(result[0]) > 0 && len(result[1]) > 0
}

func invokeBagArgument(tokens []parser.Token, pos int, name string) (expr []parser.Token, passed, ours, known bool) {
	bag := producerUnwrap(tokens)
	if len(bag) < 2 {
		return nil, false, false, false
	}

	switch bag[0].Kind {
	case parser.TokLBracket:
		if producerGroupEnd(bag, 0, parser.TokLBracket, parser.TokRBracket) != len(bag)-1 {
			return nil, false, false, false
		}

		args := producerSplit(bag[1 : len(bag)-1])
		if pos >= len(args) || len(args[pos]) == 0 {
			return nil, false, true, true
		}

		return args[pos], true, true, true
	case parser.TokLBrace:
		if producerGroupEnd(bag, 0, parser.TokLBrace, parser.TokRBrace) != len(bag)-1 {
			return nil, false, false, false
		}

		var value []parser.Token

		for _, arg := range producerSplit(bag[1 : len(bag)-1]) {
			if len(arg) == 0 {
				continue
			}

			if len(arg) < 3 || arg[0].Kind != parser.TokIdent && arg[0].Kind != parser.TokString || arg[1].Kind != parser.TokEquals && arg[1].Kind != parser.TokColon {
				return nil, false, false, false
			}

			key := strings.Trim(arg[0].Value, `"'`)
			if strings.Contains(key, "#") {
				return nil, false, false, false
			}

			if strings.EqualFold(key, name) {
				if value != nil {
					return nil, false, false, false
				}

				value = arg[2:]
			}
		}

		return value, len(value) > 0, true, true
	default:
		return nil, false, false, false
	}
}
