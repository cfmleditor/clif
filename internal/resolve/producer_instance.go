package resolve

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Constructor values belong to this object, not its class. This first contract
// is intentionally closed: an empty pseudo-constructor, no ancestry, a simple
// init returning this, private scalar fields or flat literal binding records,
// and read-only methods or closed private setters. Discovery, container aliases
// or escape and framework injection require separate execution contracts.
type producerInstance struct {
	values producerEnvironment
	valid  bool
}

func (e *producerEvaluation) constructedInstance(component, dir string, arguments map[string]producerValue) *producerInstance {
	if e.depth > 6 || component == "" || *e.budget <= 0 {
		return nil
	}

	file := e.resolver.ComponentPath(component, dir)
	if file == "" {
		return nil
	}

	if pathKey(e.resolver.FindApplicationRoot(dir)) != pathKey(e.resolver.FindApplicationRoot(filepath.Dir(file))) {
		return nil
	}

	source := e.resolver.wheelsSource(file)

	pr := parser.Parse(cfpath.ToURI(file), source.content, nil)
	if pr.Extends != "" {
		return nil
	}

	names := map[string]bool{}

	for i := range pr.Funcs {
		name := strings.ToLower(pr.Funcs[i].Name)
		if names[name] {
			return nil
		}

		names[name] = true
	}

	constructor, _ := parser.ApplicationSections(source.content)
	if wheelsTokens(constructor) != wheelsTokens(`component {}`) {
		return nil
	}

	fd := e.resolver.ResolveFunc(file, "init", dir)
	if fd == nil || !e.instanceSelfReturnCompatible(fd, file) {
		return nil
	}

	method := e.resolver.producerFor(fd)
	if method == nil || len(method.body) == 0 || !instanceLiteralDefaults(method) {
		return nil
	}

	local := *e
	local.fd, local.receiver, local.baseDir = fd, file, filepath.Dir(file)
	local.callerArguments, local.argumentsKnown = false, true
	local.depth++
	local.instance = nil

	env := local.bindArguments(fd, method, arguments, true)
	if !instanceArgumentsCompatible(fd, env) {
		return nil
	}

	values := producerEnvironment{}
	recordInputs := map[string]bool{}
	returned := false

	for i := range method.body {
		node := &method.body[i]
		switch node.kind {
		case "set":
			path := normalizeProducerPath(node.target)
			if !strings.HasPrefix(path, "variables.") || strings.Count(path, ".") != 1 {
				return nil
			}

			value := local.expression(node.expression, env)
			if !value.literalKnown && !producerLiteralRecord(value) || value.component() != "" {
				return nil
			}

			if value.fields != nil && !instanceRecordInput(node.expression, recordInputs) {
				return nil
			}

			env[path], values[path] = value, value
		case "return":
			if i != len(method.body)-1 || !strings.EqualFold(strings.TrimSpace(node.expression), "this") {
				return nil
			}

			returned = true
		default:
			return nil
		}
	}

	if !returned || len(values) == 0 {
		return nil
	}

	plans := source.producers.all()
	for name, other := range plans {
		if !instanceLiteralDefaults(other) {
			return nil
		}

		setter := instanceSetterContract(name, plans, values, nil)
		if other != method && !instanceReadOnly(other.body, values) && !setter {
			return nil
		}

		if other != method && setter && !e.instanceSelfReturnCompatible(e.resolver.ResolveFunc(file, name, dir), file) {
			return nil
		}
	}

	return &producerInstance{values: values, valid: true}
}

// Defaults execute before the body. Calls or scope references in a default
// may mutate or expose the factory even when the body is a read-only getter.
func instanceLiteralDefaults(method *producerMethod) bool {
	for _, expression := range method.defaults {
		tokens := producerTokens(expression)
		if len(tokens) == 0 {
			return false
		}

		for i, token := range tokens {
			if token.Kind == parser.TokString && !strings.Contains(token.Value, "#") || token.Kind == parser.TokNumber {
				continue
			}

			if token.Kind == parser.TokIdent && (strings.EqualFold(token.Value, "true") || strings.EqualFold(token.Value, "false") || i+1 < len(tokens) && (tokens[i+1].Kind == parser.TokColon || tokens[i+1].Kind == parser.TokEquals)) {
				continue
			}

			if token.Kind != parser.TokLBrace && token.Kind != parser.TokRBrace && token.Kind != parser.TokLBracket && token.Kind != parser.TokRBracket && token.Kind != parser.TokComma && token.Kind != parser.TokColon && token.Kind != parser.TokEquals {
				return false
			}
		}
	}

	return true
}

// A closed setter returns this. A declared primitive or unrelated component
// type cannot establish successful execution of that source contract.
func (e *producerEvaluation) instanceSelfReturnCompatible(fd *parser.FunctionDef, receiver string) bool {
	if fd == nil {
		return false
	}

	switch strings.ToLower(fd.ReturnType) {
	case "", "any", "object", "component":
		return true
	}

	returnType := e.resolver.ComponentPath(fd.ReturnType, filepath.Dir(fd.URI.Path()))

	return returnType != "" && cfpath.SamePath(returnType, receiver)
}

// A setter is a closed list of private field writes followed by return this.
// Calls, branches, indirect indexed writes and exposed records need broader execution
// contracts. Declaring this setter is safe while the object remains unescaped;
// its actual arguments are evaluated separately at the invocation.
func instanceSetter(nodes []producerNode) bool {
	if len(nodes) < 2 || nodes[len(nodes)-1].kind != "return" || !strings.EqualFold(strings.TrimSpace(nodes[len(nodes)-1].expression), "this") {
		return false
	}

	for i := range len(nodes) - 1 {
		node := &nodes[i]

		path := normalizeProducerPath(node.target)
		if node.kind == "private-index-set" {
			var ok bool

			path, _, ok = instanceIndexTarget(producerTokens(node.target))
			if !ok {
				return false
			}
		} else if node.kind != "set" {
			return false
		}

		if !strings.HasPrefix(path, "variables.") || strings.Count(path, ".") != 1 {
			return false
		}

		tokens := producerUnwrap(producerTokens(node.expression))

		source := producerPath(tokens)
		if (strings.HasPrefix(source, "arguments.") || strings.HasPrefix(source, "variables.")) && strings.Count(source, ".") == 1 {
			continue
		}

		if len(tokens) != 1 || tokens[0].Kind != parser.TokString || strings.Contains(tokens[0].Value, "#") {
			return false
		}
	}

	return true
}

// A forwarding alias must call another closed setter on this same object.
// Check every source hop before admitting the object; cycles, shadowed methods
// and expressions that could publish private state are not forwarding aliases.
func instanceSetterContract(name string, plans map[string]*producerMethod, fields producerEnvironment, seen []string) bool {
	name = strings.ToLower(name)
	if name == "init" || len(seen) >= 8 {
		return false
	}

	if slices.Contains(seen, name) {
		return false
	}

	method := plans[name]
	if method == nil {
		return false
	}

	if instanceSetter(method.body) {
		return true
	}

	target, bare := instanceForwardTarget(method)
	if target == "" {
		return false
	}

	if _, shadowed := fields["variables."+target]; shadowed {
		return false
	}

	if bare {
		for _, parameter := range method.parameters {
			if strings.EqualFold(parameter, target) {
				return false
			}
		}
	}

	return instanceSetterContract(target, plans, fields, append(seen, name))
}

func instanceForwardTarget(method *producerMethod) (target string, bare bool) {
	if len(method.body) != 1 || method.body[0].kind != "return" {
		return "", false
	}

	tokens := producerUnwrap(producerTokens(method.body[0].expression))

	open := 1
	if len(tokens) > 3 && tokens[1].Kind == parser.TokDot {
		open = 3
	}

	if len(tokens) <= open || tokens[open].Kind != parser.TokLParen || producerGroupEnd(tokens, open, parser.TokLParen, parser.TokRParen) != len(tokens)-1 {
		return "", false
	}

	path := producerPath(tokens[:open])

	target, bare = path, open == 1
	if !bare {
		scope, name, _ := strings.Cut(path, ".")
		if scope != "this" && scope != "variables" {
			return "", false
		}

		target = name
	}

	if !instanceForwardArguments(tokens[open+1 : len(tokens)-1]) {
		return "", false
	}

	return target, bare
}

func instanceForwardArguments(tokens []parser.Token) bool {
	parts := producerSplit(tokens)
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}

		if len(part) > 2 && part[0].Kind == parser.TokIdent && part[1].Kind == parser.TokEquals {
			if strings.EqualFold(part[0].Value, "argumentCollection") {
				return len(parts) == 1 && producerPath(part[2:]) == "arguments"
			}

			part = part[2:]
		}

		path := producerPath(part)
		if strings.HasPrefix(path, "arguments.") && strings.Count(path, ".") == 1 {
			continue
		}

		if len(part) != 1 || part[0].Kind != parser.TokString || strings.Contains(part[0].Value, "#") {
			return false
		}
	}

	return true
}

func (e *producerEvaluation) updateInstance(fd *parser.FunctionDef, method *producerMethod, receiver string, instance *producerInstance, arguments map[string]producerValue) producerValue {
	if e.depth > 6 || *e.budget <= 0 {
		instance.valid = false

		return producerUnknown()
	}

	*e.budget--
	local := *e
	local.fd, local.receiver, local.baseDir = fd, receiver, filepath.Dir(fd.URI.Path())
	local.callerArguments, local.argumentsKnown, local.instance = false, true, instance
	local.depth++

	env := local.bindArguments(fd, method, arguments, true)
	if !instanceArgumentsCompatible(fd, env) {
		instance.valid = false

		return producerUnknown()
	}

	values := maps.Clone(instance.values)
	maps.Copy(env, values)

	recordInputs := map[string]bool{}

	for i := range len(method.body) - 1 {
		node := &method.body[i]

		value := local.expression(node.expression, env)
		if !value.literalKnown && !producerLiteralRecord(value) || value.component() != "" {
			instance.valid = false

			return producerUnknown()
		}

		if value.fields != nil && !instanceRecordInput(node.expression, recordInputs) {
			instance.valid = false

			return producerUnknown()
		}

		path := normalizeProducerPath(node.target)
		if node.kind == "private-index-set" {
			var (
				selector string
				ok       bool
			)

			path, selector, ok = instanceIndexTarget(producerTokens(node.target))
			key := local.expression(selector, env)

			record := values[path]
			if !ok || !key.literalKnown || !value.literalKnown || !producerLiteralRecord(record) {
				instance.valid = false

				return producerUnknown()
			}

			record.fields = maps.Clone(record.fields)
			value.presenceKnown = true
			record.fields[strings.ToLower(key.literal)] = value
			value = record
		}

		values[path] = value
		env[path] = value
	}

	instance.values = values
	value := local.concrete(receiver, local.baseDir)
	value.instance = instance

	return value
}

// A private table must have one stored reference. Reusing an argument record
// for two fields or assigning an existing private table introduces aliases
// whose later indexed writes cannot be modeled as independent records.
func instanceRecordInput(expression string, seen map[string]bool) bool {
	tokens := producerUnwrap(producerTokens(expression))
	if len(tokens) > 0 && tokens[0].Kind == parser.TokLBrace {
		return true
	}

	path := producerPath(tokens)
	if !strings.HasPrefix(path, "arguments.") || strings.Count(path, ".") != 1 || seen[path] {
		return false
	}

	seen[path] = true

	return true
}

func instanceIndexTarget(tokens []parser.Token) (path, selector string, ok bool) {
	for i, token := range tokens {
		if token.Kind != parser.TokLBracket {
			continue
		}

		if producerGroupEnd(tokens, i, parser.TokLBracket, parser.TokRBracket) != len(tokens)-1 {
			return "", "", false
		}

		path = producerPath(tokens[:i])
		if !strings.HasPrefix(path, "variables.") || strings.Count(path, ".") != 1 {
			return "", "", false
		}

		key := tokens[i+1 : len(tokens)-1]

		keyPath := producerPath(key)
		if (strings.HasPrefix(keyPath, "arguments.") || strings.HasPrefix(keyPath, "variables.")) && strings.Count(keyPath, ".") == 1 || len(key) == 1 && key[0].Kind == parser.TokString && !strings.Contains(key[0].Value, "#") {
			return path, producerText(key), true
		}

		return "", "", false
	}

	return "", "", false
}

func instanceArgumentsCompatible(fd *parser.FunctionDef, env producerEnvironment) bool {
	for _, argument := range fd.Arguments {
		value := env["arguments."+strings.ToLower(argument.Name)]
		if value.absent {
			if argument.Required {
				return false
			}

			continue
		}

		switch strings.ToLower(argument.Type) {
		case "", "any":
		case "string":
			if !value.literalKnown {
				return false
			}
		case "struct":
			if !producerLiteralRecord(value) {
				return false
			}
		default:
			return false
		}
	}

	return true
}

// Creating another closed, unescaped factory cannot mutate an existing one.
// Nested calls and component arguments still require an effect contract.
func (e *producerEvaluation) closedConstructionEffect(tokens []parser.Token, env producerEnvironment) bool {
	if len(tokens) < 4 || !strings.EqualFold(tokens[0].Value, "new") {
		return false
	}

	open := -1

	for i, token := range tokens {
		if token.Kind == parser.TokLParen {
			open = i

			break
		}
	}

	if open < 0 || producerGroupEnd(tokens, open, parser.TokLParen, parser.TokRParen) != len(tokens)-1 {
		return false
	}

	for _, token := range tokens[open+1 : len(tokens)-1] {
		if token.Kind == parser.TokLParen || strings.EqualFold(token.Value, "function") || e.read(token.Value, env).instance != nil {
			return false
		}
	}

	value := e.expression(producerText(tokens), env)

	return value.instance != nil && value.instance.valid
}

// Every hop in an effect chain must be a closed setter returning the same
// instance. Validate the whole shape before executing any of its writes.
func (e *producerEvaluation) instanceSetterEffect(tokens []parser.Token, env producerEnvironment) bool {
	open := -1

	for i, token := range tokens {
		if token.Kind == parser.TokLParen {
			open = i

			break
		}
	}

	if open < 0 {
		return false
	}

	path := producerPath(tokens[:open])

	receiver, name, qualified := strings.CutLast(path, ".")
	if !qualified {
		return false
	}

	value := e.read(receiver, env)
	if value.instance == nil || !value.instance.valid || value.component() == "" {
		return false
	}

	for {
		end := producerGroupEnd(tokens, open, parser.TokLParen, parser.TokRParen)
		if end < 0 || strings.EqualFold(name, "init") {
			return false
		}

		fd := e.resolver.ResolveFunc(value.component(), name, e.baseDir)
		if fd == nil {
			return false
		}

		method := e.resolver.producerFor(fd)
		if method == nil || !instanceSetterContract(name, e.resolver.wheelsSource(fd.URI.Path()).producers.all(), value.instance.values, nil) {
			return false
		}

		for _, token := range tokens[open+1 : end] {
			if token.Kind == parser.TokLParen || strings.EqualFold(token.Value, "function") {
				return false
			}
		}

		if end == len(tokens)-1 {
			break
		}

		if end+3 >= len(tokens) || tokens[end+1].Kind != parser.TokDot || tokens[end+2].Kind != parser.TokIdent || tokens[end+3].Kind != parser.TokLParen {
			return false
		}

		name, open = tokens[end+2].Value, end+3
	}

	e.expression(producerText(tokens), env)

	return true
}

func instanceReadOnly(nodes []producerNode, fields producerEnvironment) bool {
	for i := range nodes {
		node := &nodes[i]
		if node.kind != "return" && node.kind != "noop" {
			return false
		}
		// Only literal values, private scalar reads and direct createObject
		// returns are closed here. Returning a scope or calling another method
		// can expose or mutate the captured container.
		tokens := producerTokens(node.expression)
		if node.kind == "noop" {
			continue
		}

		if len(tokens) == 0 {
			return false
		}

		for _, token := range tokens {
			if token.Kind == parser.TokString && strings.Contains(token.Value, "#") {
				return false
			}
		}

		path := producerPath(tokens)
		if path != "" && !strings.HasPrefix(path, "variables.") {
			return false
		}

		if fields[normalizeProducerPath(path)].fields != nil {
			return false
		}

		if path == "" && len(tokens) > 1 && !instanceConstructionReturn(tokens) {
			return false
		}
	}

	return true
}

func instanceConstructionReturn(tokens []parser.Token) bool {
	if len(tokens) < 6 || !strings.EqualFold(tokens[0].Value, "createObject") || tokens[1].Kind != parser.TokLParen || producerGroupEnd(tokens, 1, parser.TokLParen, parser.TokRParen) != len(tokens)-1 {
		return false
	}

	parts := producerSplit(tokens[2 : len(tokens)-1])
	if len(parts) != 2 || len(parts[0]) != 1 || parts[0][0].Kind != parser.TokString || !strings.EqualFold(strings.Trim(parts[0][0].Value, `"'`), "component") {
		return false
	}

	if len(parts[1]) == 1 && parts[1][0].Kind == parser.TokString && !strings.Contains(parts[1][0].Value, "#") {
		return true
	}

	path := producerPath(parts[1])
	if path != "" {
		return strings.HasPrefix(path, "variables.") && strings.Count(path, ".") == 1
	}

	for i, token := range parts[1] {
		if token.Kind != parser.TokLBracket {
			continue
		}

		if producerGroupEnd(parts[1], i, parser.TokLBracket, parser.TokRBracket) != len(parts[1])-1 {
			return false
		}

		path = producerPath(parts[1][:i])
		key := parts[1][i+1 : len(parts[1])-1]
		selector := producerPath(key)

		return strings.HasPrefix(path, "variables.") && strings.Count(path, ".") == 1 &&
			(strings.HasPrefix(selector, "arguments.") && strings.Count(selector, ".") == 1 || len(key) == 1 && key[0].Kind == parser.TokString && !strings.Contains(key[0].Value, "#"))
	}

	return false
}

func producerLiteralRecord(value producerValue) bool {
	if value.fields == nil || value.unknown || len(value.components) > 0 {
		return false
	}

	for _, field := range value.fields {
		if !field.literalKnown || field.fields != nil || field.unknown || len(field.components) > 0 {
			return false
		}
	}

	return true
}
