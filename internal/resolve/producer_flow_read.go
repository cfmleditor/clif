package resolve

import (
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

func (e *producerEvaluation) read(path string, env producerEnvironment) producerValue {
	path = normalizeProducerPath(path)

	if e.callerArguments {
		return producerUnknown()
	}

	if value, ok := env[path]; ok {
		return value
	}

	for parent := path; strings.Contains(parent, "."); {
		parent, _, _ = strings.CutLast(parent, ".")
		if _, replaced := env[parent]; replaced {
			return producerUnknown()
		}
	}

	if path == "this" {
		value := e.concrete(e.receiver, e.baseDir)
		value.instance = e.instance

		return value
	}

	if path == "arguments" {
		if !e.argumentsKnown {
			return producerUnknown()
		}

		if _, escaped := env["@arguments"]; escaped {
			return producerUnknown()
		}

		fields := map[string]producerValue{}

		for key, v := range env {
			if strings.HasPrefix(key, "arguments.") && !v.presenceKnown {
				v = producerUnknown()
			}

			if name, ok := strings.CutPrefix(key, "arguments."); ok && !strings.Contains(name, ".") && !v.absent {
				fields[name] = v
			}
		}

		return producerValue{fields: fields}
	}

	if !strings.Contains(path, ".") {
		if value, ok := env["arguments."+path]; ok {
			return value
		}

		path = "variables." + path
		if value, ok := env[path]; ok {
			return value
		}
	}

	if e.resolver.Index == nil {
		return producerUnknown()
	}

	name := strings.TrimPrefix(strings.TrimPrefix(path, "variables."), "this.")
	if name == path {
		return producerUnknown()
	}

	if _, escaped := env["@"+strings.Split(path, ".")[0]]; escaped {
		return producerUnknown()
	}

	var answer producerValue

	found := false

	for _, ref := range e.resolver.Index.RefsForFile(e.fd.URI) {
		if !strings.EqualFold(ref.Variable, name) || ref.This != strings.HasPrefix(path, "this.") {
			continue
		}

		// The index's parse could not look the method up, and the receiver's
		// own type is no statement of what a call on it returns.
		if ref.BaseGuess {
			return producerUnknown()
		}

		value := e.concrete(ref.Component, filepath.Dir(e.fd.URI.Path()))
		if !found {
			answer = value
			found = true
		} else {
			answer = producerMerge(answer, value)
		}
	}

	if !found {
		return producerUnknown()
	}

	return answer
}

func (e *producerEvaluation) expression(expression string, env producerEnvironment) producerValue {
	tokens := producerUnwrap(producerTokens(expression))
	if len(tokens) == 0 {
		return producerUnknown()
	}

	if len(tokens) == 1 && (tokens[0].Kind == parser.TokString && !strings.Contains(tokens[0].Value, "#") || tokens[0].Kind == parser.TokNumber || strings.EqualFold(tokens[0].Value, "true") || strings.EqualFold(tokens[0].Value, "false")) {
		if tokens[0].Kind == parser.TokString {
			value := tokens[0].Value
			quote := value[:1]

			return producerValue{primitive: true, literalKnown: true, literal: strings.ReplaceAll(value[1:len(value)-1], quote+quote, quote)}
		}

		return producerValue{primitive: true}
	}

	if tokens[0].Kind == parser.TokLBrace && producerGroupEnd(tokens, 0, parser.TokLBrace, parser.TokRBrace) == len(tokens)-1 {
		fields := map[string]producerValue{}

		for _, entry := range producerSplit(tokens[1 : len(tokens)-1]) {
			if len(entry) == 0 {
				continue
			}

			if len(entry) < 3 || entry[1].Kind != parser.TokEquals && entry[1].Kind != parser.TokColon || entry[0].Kind != parser.TokIdent && entry[0].Kind != parser.TokString {
				return producerUnknown()
			}

			key := strings.ToLower(strings.Trim(entry[0].Value, "\"'"))
			if strings.Contains(key, "#") {
				return producerUnknown()
			}

			if _, duplicate := fields[key]; duplicate {
				return producerUnknown()
			}

			value := e.expression(producerText(entry[2:]), env)
			value.presenceKnown = true
			fields[key] = value
		}

		return producerValue{primitive: true, fields: fields}
	}

	if path := producerPath(tokens); path != "" {
		return e.read(path, env)
	}

	if tokens[len(tokens)-1].Kind == parser.TokRBracket {
		return e.recordElement(tokens, env)
	}

	construct := strings.EqualFold(tokens[0].Value, "new")
	if construct {
		tokens = tokens[1:]
	}

	open := -1

	for i, t := range tokens {
		if t.Kind == parser.TokLParen {
			open = i

			break
		}
	}

	if open < 0 {
		return producerUnknown()
	}

	endRoot := producerGroupEnd(tokens, open, parser.TokLParen, parser.TokRParen)
	if endRoot < 0 {
		return producerUnknown()
	}

	before := producerPath(tokens[:open])
	if before == "" {
		return producerUnknown()
	}

	value := e.rootCall(tokens, open, endRoot, before, construct, env)

	position := endRoot + 1
	for position < len(tokens) {
		if position+2 >= len(tokens) || tokens[position].Kind != parser.TokDot || tokens[position+1].Kind != parser.TokIdent || tokens[position+2].Kind != parser.TokLParen {
			return producerUnknown()
		}

		method := tokens[position+1].Value

		end := producerGroupEnd(tokens, position+2, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			return producerUnknown()
		}

		// A mock is the class it names whatever made it, so the receiver
		// need not be known: getMockBox() is a spec's, from TestBox.
		mock, isMock := e.mock(tokens[position+1 : end+1])

		switch {
		case isMock:
			value = mock
		case value.component() == "":
			return producerUnknown()
		case !strings.EqualFold(method, "init"):
			args, known := e.arguments(tokens[position+3:end], env)
			if !known {
				return producerUnknown()
			}

			value = e.methodResult(value, method, args)
		case value.instance != nil:
			value.instance.valid = false
		}

		position = end + 1
	}

	return value
}

func (e *producerEvaluation) recordElement(tokens []parser.Token, env producerEnvironment) producerValue {
	for i, token := range tokens {
		if token.Kind != parser.TokLBracket {
			continue
		}

		if producerGroupEnd(tokens, i, parser.TokLBracket, parser.TokRBracket) != len(tokens)-1 {
			break
		}

		path := producerPath(tokens[:i])

		key := e.expression(producerText(tokens[i+1:len(tokens)-1]), env)
		if path == "" || !key.literalKnown {
			return producerUnknown()
		}

		// A field the plan wrote, or a guard proved, under this key:
		// `return variables.services[ name ]` after a keyed cache's build.
		if name, ok := e.literalKey(tokens[i+1:len(tokens)-1], env); ok {
			if field, known := env[normalizeProducerPath(path+"."+name)]; known {
				return field
			}
		}

		value := e.read(path, env)
		if field, exists := value.fields[strings.ToLower(key.literal)]; exists {
			return field
		}

		return producerUnknown()
	}

	return producerUnknown()
}

func (e *producerEvaluation) arguments(tokens []parser.Token, env producerEnvironment) (map[string]producerValue, bool) {
	arguments := map[string]producerValue{}

	pieces := producerSplit(tokens)
	for i, piece := range pieces {
		if len(piece) == 0 {
			if len(tokens) > 0 {
				return nil, false
			}

			continue
		}

		name := ""
		if len(piece) > 2 && piece[0].Kind == parser.TokIdent && (piece[1].Kind == parser.TokEquals || piece[1].Kind == parser.TokColon) {
			name = strings.ToLower(piece[0].Value)
			piece = piece[2:]
		}

		if name == "argumentcollection" {
			if len(pieces) != 1 {
				return nil, false
			}

			value := e.expression(producerText(piece), env)
			if value.fields == nil {
				return nil, false
			}

			for _, field := range value.fields {
				if field.instance != nil {
					field.instance.valid = false
				}

				if !field.presenceKnown {
					return nil, false
				}
			}

			return maps.Clone(value.fields), true
		}
		// Positional names are assigned by the callee at dispatch time.
		if name == "" {
			name = producerPosition(i)
		}

		if _, duplicate := arguments[name]; duplicate {
			return nil, false
		}

		value := e.expression(producerText(piece), env)
		if value.instance != nil {
			value.instance.valid = false
		}

		if value.fields != nil && (!producerLiteralRecord(value) || len(piece) == 0 || piece[0].Kind != parser.TokLBrace) {
			value = producerUnknown()
		}

		value.presenceKnown = true
		arguments[name] = value
	}

	return arguments, true
}

func producerPosition(position int) string { return "@" + strconv.Itoa(position) }

func (e *producerEvaluation) effect(expression string, env producerEnvironment) {
	tokens := producerTokens(expression)
	if e.instanceSetterEffect(tokens, env) || e.closedConstructionEffect(tokens, env) {
		return
	}

	for _, value := range env {
		if value.instance != nil {
			value.instance.valid = false
		}
	}

	if e.argumentMutation(tokens, env) {
		return
	}

	for i, tok := range tokens {
		// A closure writes the enclosing function's locals by reference, and an
		// included template runs in its scope; neither write is visible here.
		hidden := tok.Kind == parser.TokIdent && (strings.EqualFold(tok.Value, "function") || strings.EqualFold(tok.Value, "cfinclude") || i == 0 && strings.EqualFold(tok.Value, "include")) ||
			tok.Kind == parser.TokEquals && i+1 < len(tokens) && tokens[i+1].Kind == parser.TokGT
		if hidden || strings.EqualFold(tok.Value, "evaluate") {
			env["@variables"] = producerUnknown()
			for key := range env {
				env[key] = producerUnknown()
			}

			return
		}
	}

	for i, tok := range tokens {
		if tok.Kind != parser.TokIdent || i == 0 || tokens[i-1].Kind != parser.TokLParen && tokens[i-1].Kind != parser.TokComma && tokens[i-1].Kind != parser.TokEquals {
			continue
		}

		end := i + 1
		for end+1 < len(tokens) && tokens[end].Kind == parser.TokDot && tokens[end+1].Kind == parser.TokIdent {
			end += 2
		}

		if end > i+1 {
			path := producerPath(tokens[i:end])
			if e.read(path, env).fields != nil {
				env[normalizeProducerPath(path)] = producerUnknown()
			}

			continue
		}

		if !slices.Contains([]string{"arguments", "local", "variables", "this"}, strings.ToLower(tok.Value)) {
			if value, ok := env[normalizeProducerPath(tok.Value)]; ok && value.fields != nil {
				env[normalizeProducerPath(tok.Value)] = producerUnknown()
			}

			continue
		}

		if i+1 < len(tokens) && (tokens[i+1].Kind == parser.TokRParen || tokens[i+1].Kind == parser.TokComma) {
			scope := strings.ToLower(tok.Value)

			env["@"+scope] = producerUnknown()
			for key := range env {
				if strings.HasPrefix(key, scope+".") || scope == "local" && !strings.Contains(key, ".") {
					env[key] = producerUnknown()
				}
			}
		}
	}
}

func (e *producerEvaluation) methodOwner() string {
	if e.receiver != "" {
		return e.receiver
	}

	return e.fd.URI.Path()
}
