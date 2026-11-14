package resolve

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Source-owned invoke calls use ordinary caller inference, even when the
// declaration has the same name as the builtin dispatcher.
func (r *Resolver) builtinInvoke(call *parser.CallSite, pr *parser.ParseResult, dir string) bool {
	return strings.EqualFold(call.FuncName, "invoke") && call.Variable == "" && call.Component == "" && r.bareFunc("invoke", pr, dir, call.Caller) == nil && !r.builtinCallShadowed("invoke", call.Line, call.Caller, pr)
}

// invokeArgument admits positional and named builtin forms with a literal
// method selector, or one computed around a literal start that may name the
// function (invokeSelects); a selector with no literal start is outside the
// caller index's coverage.
// An unknown target blocks inference rather than borrowing a known caller's
// argument, unless only fd's own class family declares the name
// (declaredInFamilyOnly); an unknown bag always does. Local functions named invoke are ordinary
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

	tokens = slices.Clone(tokens) // allTokens' answer is shared

	for i := range tokens {
		tokens[i].Offset += int(call.StartByte)
		tokens[i].Line += conv.Int32(int(call.Line))
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
	if !invokeSelects(method, fd.Name) {
		return nil, false, false, true
	}

	// A computed selector may name many methods, so one on a target that
	// cannot be placed is passed over, as before selectors were read: it would
	// otherwise withhold inference for every method its literal start allows.
	computed := strings.Contains(method, "#")
	method = fd.Name

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

	switch {
	case target == "" && !r.declaredInFamilyOnly(fd):
		return nil, false, false, computed
	case target != "" && !r.componentCallIs(target, method, fd, dir):
		return nil, false, false, true
	}

	if len(parts[2]) == 0 {
		return nil, false, true, true
	}

	return invokeBagArgument(parts[2], pos, name)
}

// declaredInFamilyOnly reports whether every component in the workspace that
// declares a method of fd's name is in one class tree: the one under the
// topmost component fd's file extends (or is) that declares it. An invoke
// selecting that name on a target of no known type then calls it on that
// tree, whatever the target is, so what it passes is evidence for fd: qb's
// Blueprint hands its grammar, built by WireBox from a setting, every
// compileX() it runs, and each grammar overrides some of BaseGrammar's.
func (r *Resolver) declaredInFamilyOnly(fd *parser.FunctionDef) bool {
	if r.Index == nil || !fd.URI.IsFile() {
		return false
	}

	defs := r.Index.Lookup(fd.Name)
	root := fd.URI.Path()

	for _, def := range defs {
		if def.URI.IsFile() && r.descendsFrom(root, def.URI.Path()) {
			root = def.URI.Path()
		}
	}

	for _, def := range defs {
		if !def.URI.IsFile() {
			return false
		}

		path := def.URI.Path()
		if !cfpath.SamePath(path, root) && !r.descendsFrom(path, root) {
			return false
		}
	}

	return len(defs) > 0
}

// invokeSelects reports whether an invoke's method selector may name the
// function: it is the function's name, or a name computed around literal
// parts the name holds in that order (`compile#command.getType()#` may be
// compileCreate). Such an invoke may call other methods as well; what matters
// is what it passes when it calls this one.
func invokeSelects(selector, name string) bool {
	if !strings.Contains(selector, "#") {
		return strings.EqualFold(selector, name)
	}

	pieces := strings.Split(selector, "#")
	if len(pieces)%2 == 0 || strings.Contains(selector, "##") || pieces[0] == "" {
		return false
	}

	// Each computed part stands for one character or more, the literals
	// between them in order.
	literals := make([]string, 0, len(pieces)/2+1)
	for i := 0; i < len(pieces); i += 2 {
		literals = append(literals, pieces[i])
	}

	return (&wildPattern{pieces: literals, kind: wildLine}).match(name)
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
