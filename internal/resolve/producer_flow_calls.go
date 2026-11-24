package resolve

import (
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
)

func (e *producerEvaluation) loop(node *producerNode, environment producerEnvironment) producerResult {
	invalidateProducerInstances(environment)

	result := producerResult{environment: maps.Clone(environment)}
	for range 16 {
		input := maps.Clone(result.environment)
		if node.target != "" {
			input[normalizeProducerPath(node.target)] = producerUnknown()
		}

		child := e.execute(node.body, input)

		result.returns = append(result.returns, child.returns...)
		if child.stopped {
			return result
		}

		next := mergeProducerEnvironments(result.environment, child.environment)
		if producerEnvironmentsEqual(result.environment, next) {
			return result
		}

		result.environment = next
	}

	return producerResult{environment: environment, returns: []producerValue{producerUnknown()}, stopped: true}
}

func producerEnvironmentsEqual(a, b producerEnvironment) bool {
	if len(a) != len(b) {
		return false
	}

	for key, x := range a {
		y, ok := b[key]
		if !ok || x.unknown != y.unknown || x.primitive != y.primitive || x.absent != y.absent || x.presenceKnown != y.presenceKnown || x.literalKnown != y.literalKnown || x.literal != y.literal || x.instance != y.instance || !reflect.DeepEqual(x.fields, y.fields) || len(x.components) != len(y.components) {
			return false
		}

		for _, c := range x.components {
			if !slices.Contains(y.components, c) {
				return false
			}
		}
	}

	return true
}

func (e *producerEvaluation) argumentMutation(tokens []parser.Token, env producerEnvironment) bool {
	if len(tokens) < 5 || tokens[0].Kind != parser.TokIdent || tokens[1].Kind != parser.TokLParen || tokens[len(tokens)-1].Kind != parser.TokRParen {
		return false
	}

	name := strings.ToLower(tokens[0].Value)
	if name != "structappend" && name != "structdelete" {
		return false
	}

	pieces := producerSplit(tokens[2 : len(tokens)-1])
	if len(pieces) < 2 || producerPath(pieces[0]) != "arguments" || e.resolver.ResolveFunc(e.methodOwner(), name, e.baseDir) != nil {
		return false
	}

	if name == "structdelete" {
		key := e.expression(producerText(pieces[1]), env)
		if !key.literalKnown {
			return false
		}

		env["arguments."+strings.ToLower(key.literal)] = producerValue{unknown: true, absent: true, presenceKnown: true}

		return true
	}

	source := e.expression(producerText(pieces[1]), env)
	if source.fields == nil {
		return false
	}

	overwrite := true

	if len(pieces) > 2 {
		if len(pieces[2]) != 1 {
			return false
		}

		if strings.EqualFold(pieces[2][0].Value, "false") {
			overwrite = false
		} else if !strings.EqualFold(pieces[2][0].Value, "true") {
			return false
		}
	}

	for name, value := range source.fields {
		key := "arguments." + name

		before, exists := env[key]
		if !overwrite && exists && !before.absent {
			continue
		}

		value.presenceKnown = true
		env[key] = value
	}

	return true
}

// Query output names can replace the value we are about to return. Attribute
// bags must prove those names even when unrelated database options are unknown.
func (e *producerEvaluation) query(node *producerNode, env producerEnvironment) bool {
	fields := map[string]producerValue{}

	if node.expression != "" {
		attrs := e.expression(node.expression, env)
		if attrs.fields == nil {
			return false
		}

		fields = attrs.fields
	}

	explicit := map[string]string{"name": node.target, "result": node.alternative[0].target}
	for key, literal := range explicit {
		value, exists := fields[key]
		if literal != "" {
			if exists && (!value.literalKnown || value.literal != literal) {
				return false
			}

			value = producerValue{literalKnown: true, literal: literal}
			exists = true
		}

		if !exists {
			continue
		}

		if !value.literalKnown || producerPath(producerTokens(value.literal)) == "" {
			return false
		}

		path := normalizeProducerPath(strings.ToLower(value.literal))
		if !strings.Contains(path, ".") {
			if _, local := env[path]; !local {
				path = "variables." + path
			}
		}

		env[path] = producerValue{primitive: true}
	}

	return true
}

func (e *producerEvaluation) rootCall(tokens []parser.Token, open, end int, before string, construct bool, env producerEnvironment) producerValue {
	if construct {
		dir := filepath.Dir(e.fd.URI.Path())
		if e.callerArguments {
			dir = e.baseDir
		}

		value := e.concrete(before, dir)

		args, known := e.arguments(tokens[open+1:end], env)
		if known {
			value.instance = e.constructedInstance(value.component(), dir, args)
		}

		return value
	}

	path := normalizeProducerPath(before)
	if _, shadowed := env[path]; shadowed {
		return producerUnknown()
	}

	if !strings.Contains(path, ".") {
		if _, shadowed := env["arguments."+path]; shadowed {
			return producerUnknown()
		}

		if _, shadowed := env["variables."+path]; shadowed {
			return producerUnknown()
		}
	}

	if mock, ok := e.mock(tokens[open-1 : end+1]); ok {
		return mock
	}

	if strings.EqualFold(before, "createObject") && e.resolver.ResolveFunc(e.methodOwner(), "createObject", e.baseDir) == nil {
		return e.createObjectResult(tokens, open, end, env)
	}

	component, noFollow, soft := e.resolver.matchResolver(producerCallText(tokens[:end+1]), nil, nil)
	if component != "" && !noFollow && (!soft || e.resolver.componentExists(component, e.baseDir)) {
		// A dynamicIfMissing rule softens a component that names no file; one
		// that does is the rule's answer like any other's.
		return e.concrete(component, e.baseDir)
	}

	if !strings.Contains(before, ".") && e.resolver.ResolveFunc(e.methodOwner(), before, e.baseDir) == nil {
		if ret := e.resolver.BuiltinReturnLookup(before); ret != "" {
			return e.concrete(ret, e.baseDir)
		}
	}

	if e.callerArguments {
		return producerUnknown()
	}

	receiver, method, qualified := strings.CutLast(before, ".")

	target := e.receiver

	instance := e.instance
	if target == "" {
		target = e.fd.URI.Path()
	}

	// this.f() and variables.f() call the component's own f(), as a bare
	// f() does: ColdBox's request() is `return this.execute( … )` and get()
	// `return variables.request( … )`.
	switch {
	case !qualified:
		method = before
	case !strings.EqualFold(receiver, "this") && !strings.EqualFold(receiver, "variables"):
		value := e.read(receiver, env)
		target, instance = value.component(), value.instance
	}

	if target == "" {
		return producerUnknown()
	}

	args, known := e.arguments(tokens[open+1:end], env)
	if !known {
		return producerUnknown()
	}

	if strings.EqualFold(method, "init") && instance != nil {
		instance.valid = false
	}

	if ret, ok := docs.LookupBuiltinMethodReturn(target, method); ok {
		return e.concrete(ret, e.baseDir)
	}

	return e.call(e.resolver.ResolveFunc(target, method, e.baseDir), target, args, true, instance)
}

// mock is the value a MockBox call makes (createMock( "X" ) on whatever holds
// the MockBox): the class, as the parser types it (parser.MockCall). A mock of
// a class that does not resolve is unknown.
func (e *producerEvaluation) mock(call []parser.Token) (producerValue, bool) {
	if len(call) < 2 {
		return producerValue{}, false
	}

	made := parser.MockCall(call[0].Value + producerText(call[1:]))
	if made == "" {
		return producerValue{}, false
	}

	class, ok := strings.CutPrefix(made, parser.MockPrefix)
	if !ok {
		return producerUnknown(), true
	}

	path := e.resolver.ComponentPath(class, e.baseDir)
	if path == "" {
		return producerUnknown(), true
	}

	return producerValue{components: []string{path}}, true
}

func (e *producerEvaluation) bindArguments(fd *parser.FunctionDef, method *producerMethod, arguments map[string]producerValue, specific bool) producerEnvironment {
	env := producerEnvironment{}

	for i, name := range method.parameters {
		value, ok := arguments[name]
		if !ok {
			value, ok = arguments[producerPosition(i)]
		}

		if !ok && specific {
			if def, exists := method.defaults[name]; exists {
				value = e.expression(def, env)
				ok = true
			}
		}

		if !ok {
			value = producerUnknown()

			if i < len(fd.Arguments) {
				arg := fd.Arguments[i]
				if arg.Type != "" && !cfmlTypes[strings.ToLower(arg.Type)] {
					value = e.concrete(arg.Type, filepath.Dir(fd.URI.Path()))
				}
			}
		}

		if ok || specific {
			value.presenceKnown = true
			value.absent = !ok
		}

		env["arguments."+name] = value
	}

	for name, value := range arguments {
		if !strings.HasPrefix(name, "@") {
			value.presenceKnown = true
			env["arguments."+name] = value
		}
	}

	return env
}

func (e *producerEvaluation) branches(node *producerNode, env producerEnvironment) producerResult {
	invalidateProducerInstances(env)
	result := producerResult{environment: env}

	// yes and no are fresh copies whichever way they are made (condition,
	// maps.Clone, mergeProducerEnvironments), so the branches own them.
	var (
		yes, no producerEnvironment
		choice  int
	)

	if node.kind == "if" {
		yes, no, choice = e.condition(node.expression, env)
	} else {
		yes, no = maps.Clone(env), maps.Clone(env)
	}

	a, b := producerResult{environment: env}, producerResult{environment: env}
	if choice >= 0 {
		a = e.executeOwned(node.body, yes, node.kind == "try")
	}

	if node.kind == "try" {
		no = mergeProducerEnvironments(env, a.observed)
	}

	if choice <= 0 {
		b = e.executeOwned(node.alternative, no, false)
	}

	result.returns = append(result.returns, a.returns...)

	result.returns = append(result.returns, b.returns...)
	switch {
	case choice > 0:
		result.environment = a.environment
		result.stopped = a.stopped
	case choice < 0:
		result.environment = b.environment
		result.stopped = b.stopped
	case a.stopped && b.stopped:
		result.stopped = true
	case a.stopped:
		result.environment = b.environment
	case b.stopped:
		result.environment = a.environment
	default:
		result.environment = mergeProducerEnvironments(a.environment, b.environment)
	}

	return result
}

func invalidateProducerInstances(env producerEnvironment) {
	for _, value := range env {
		if value.instance != nil {
			value.instance.valid = false
		}
	}
}

func producerConditionMutation(tokens []parser.Token) bool {
	calls := []bool{}

	for i, t := range tokens {
		switch t.Kind {
		case parser.TokLParen:
			calls = append(calls, i > 0 && tokens[i-1].Kind == parser.TokIdent)
		case parser.TokRParen:
			if len(calls) > 0 {
				calls = calls[:len(calls)-1]
			}
		case parser.TokEquals:
			if i+1 < len(tokens) && tokens[i+1].Kind == parser.TokEquals || i > 0 && (tokens[i-1].Kind == parser.TokEquals || tokens[i-1].Kind == parser.TokBang || tokens[i-1].Kind == parser.TokLT || tokens[i-1].Kind == parser.TokGT) {
				continue
			}

			if len(calls) == 0 || !calls[len(calls)-1] || i < 2 || tokens[i-1].Kind != parser.TokIdent || (tokens[i-2].Kind != parser.TokLParen && tokens[i-2].Kind != parser.TokComma) {
				return true
			}
		case parser.TokPlus, parser.TokMinus:
			if i+1 < len(tokens) && tokens[i+1].Kind == t.Kind {
				return true
			}
		default:
		}
	}

	return false
}
