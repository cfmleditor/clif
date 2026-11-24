package resolve

import (
	"maps"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
)

// Interpret only lexical assignments and supported control flow; CFML is never
// executed. Unknown branches join values, and scope escapes discard evidence.
func (e *producerEvaluation) execute(nodes []producerNode, environment producerEnvironment) producerResult {
	return e.executeObserved(nodes, environment, false)
}

// Only a try body's catch path consumes the merged intermediate environments.
// Ordinary evaluations need their final environment and returns, not a copy
// of every state visited on the way there.
func (e *producerEvaluation) executeObserved(nodes []producerNode, environment producerEnvironment, observe bool) producerResult {
	return e.executeOwned(nodes, maps.Clone(environment), observe)
}

// executeOwned is executeObserved for an environment the caller has just made
// and hands over, which it writes to in place rather than copying first: a
// branch's environments are copies already, and copying them again was most
// of the map cloning in a cfwheels scan.
func (e *producerEvaluation) executeOwned(nodes []producerNode, environment producerEnvironment, observe bool) producerResult {
	result := producerResult{environment: environment}
	if observe {
		result.observed = maps.Clone(environment)
	}

	for i := range nodes {
		node := &nodes[i]

		if *e.budget <= 0 {
			return producerResult{environment: environment, returns: []producerValue{producerUnknown()}, stopped: true}
		}

		*e.budget--
		env := result.environment

		switch node.kind {
		case "set":
			if !e.assign(node, env) {
				result.returns = append(result.returns, producerUnknown())
				result.stopped = true
			}
		case "return":
			result.returns = append(result.returns, e.expression(node.expression, env))
			result.stopped = true
		case "stop":
			result.stopped = true
		case "unsafe", "private-index-set":
			result.returns = append(result.returns, producerUnknown())
			result.stopped = true
		case "query":
			if !e.query(node, env) {
				result.returns = append(result.returns, producerUnknown())
				result.stopped = true
			}

			child := e.execute(node.body, env)
			result.environment = child.environment
			result.returns = append(result.returns, child.returns...)
			result.stopped = result.stopped || child.stopped
		case "param":
			if _, ok := env[node.target]; !ok {
				env[node.target] = producerUnknown()
			}
		case "effect":
			e.effect(node.expression, env)
		case "block":
			child := e.execute(node.body, env)
			result.environment = child.environment
			result.returns = append(result.returns, child.returns...)
			result.stopped = child.stopped
		case "loop":
			child := e.loop(node, env)
			result.environment = child.environment
			result.returns = append(result.returns, child.returns...)
		case "if", "try":
			child := e.branches(node, env)
			result.environment = child.environment
			result.returns = append(result.returns, child.returns...)
			result.stopped = child.stopped
		case "switch":
			child := e.switchBranch(node, env)
			result.environment = child.environment
			result.returns = append(result.returns, child.returns...)
			result.stopped = child.stopped

		default:
		}

		if observe {
			result.observed = mergeProducerEnvironments(result.observed, result.environment)
		}

		if result.stopped {
			return result
		}
	}

	return result
}

func (e *producerEvaluation) switchBranch(node *producerNode, env producerEnvironment) producerResult {
	invalidateProducerInstances(env)

	unknown := producerResult{environment: env, returns: []producerValue{producerUnknown()}, stopped: true}
	if producerConditionMutation(producerTokens(node.expression)) {
		return unknown
	}

	selector := e.expression(node.expression, env)
	if selector.unknown || !selector.literalKnown || !producerSwitchString(selector.literal) {
		return unknown
	}

	selected, fallback := -1, -1

	for i := range node.body {
		arm := &node.body[i]
		if arm.kind == "default" {
			fallback = i

			continue
		}

		label := e.expression(arm.expression, env)
		if !label.literalKnown || !producerSwitchString(label.literal) {
			return unknown
		}

		if strings.EqualFold(selector.literal, label.literal) {
			if selected >= 0 {
				return unknown
			}

			selected = i
		}
	}

	if selected < 0 {
		selected = fallback
	}

	if selected < 0 {
		return producerResult{environment: env}
	}

	var body []producerNode
	for i := selected; i < len(node.body); i++ {
		arm := &node.body[i]

		body = append(body, arm.body...)
		if arm.target == "break" {
			break
		}
	}

	return e.execute(body, env)
}

// This rule proves textual dispatch; numeric and boolean coercion is outside
// its contract.
func producerSwitchString(value string) bool {
	if _, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
		return false
	}

	for _, boolean := range []string{"true", "false", "yes", "no"} {
		if strings.EqualFold(strings.TrimSpace(value), boolean) {
			return false
		}
	}

	return true
}

func normalizeProducerPath(path string) string {
	return strings.TrimPrefix(strings.ToLower(path), "local.")
}

func (e *producerEvaluation) condition(expression string, env producerEnvironment) (yes, no producerEnvironment, choice int) {
	yes, no = maps.Clone(env), maps.Clone(env)

	tokens := producerTokens(expression)
	if producerConditionMutation(tokens) {
		for key := range yes {
			yes[key] = producerUnknown()
			no[key] = producerUnknown()
		}

		return yes, no, 0
	}

	negate := false
	if len(tokens) > 0 && (tokens[0].Kind == parser.TokBang || strings.EqualFold(tokens[0].Value, "not")) {
		negate = true
		tokens = tokens[1:]
	}

	if len(tokens) >= 4 && strings.EqualFold(tokens[0].Value, "len") && tokens[1].Kind == parser.TokLParen && tokens[len(tokens)-1].Kind == parser.TokRParen && e.resolver.ResolveFunc(e.methodOwner(), "len", e.baseDir) == nil {
		value := e.expression(producerText(tokens[2:len(tokens)-1]), env)
		if value.literalKnown {
			choice = -1
			if value.literal != "" {
				choice = 1
			}

			if negate {
				choice = -choice
			}

			return yes, no, choice
		}
	}

	if specialYes, specialNo, specialChoice, ok := e.specialCondition(tokens, env, negate); ok {
		return specialYes, specialNo, specialChoice
	}

	if len(tokens) < 4 || !strings.EqualFold(tokens[0].Value, "isObject") || tokens[1].Kind != parser.TokLParen || tokens[len(tokens)-1].Kind != parser.TokRParen || e.resolver.ResolveFunc(e.methodOwner(), "isObject", e.baseDir) != nil {
		e.effect(expression, yes)
		e.effect(expression, no)

		return yes, no, 0
	}

	path := producerPath(tokens[2 : len(tokens)-1])
	if path == "" {
		return yes, no, 0
	}

	value := e.read(path, env)
	choice = 0

	if !value.unknown {
		if value.primitive && len(value.components) == 0 {
			choice = -1
		}

		if !value.primitive && len(value.components) > 0 {
			choice = 1
		}

		if value.component() == nativetype.Type("string") || value.component() == nativetype.Type("numeric") || value.component() == nativetype.Type("boolean") {
			choice = -1
		}
	}

	object, primitive := value, producerValue{primitive: true}
	object.primitive = false
	yes[normalizeProducerPath(path)] = object

	no[normalizeProducerPath(path)] = primitive
	if negate {
		return no, yes, -choice
	}

	return yes, no, choice
}

// specialCondition copies env for the two branches only once a condition it
// handles is found: most conditions are none of them, and copying first was a
// third of the producer's environment copies.
func (e *producerEvaluation) specialCondition(tokens []parser.Token, env producerEnvironment, negate bool) (yes, no producerEnvironment, choice int, ok bool) {
	if path, matched := e.structKeyCondition(tokens, env); matched {
		yes, no = maps.Clone(env), maps.Clone(env)

		value, known := env[path]
		if e.guardedField(path) {
			value = e.sharedFieldContract(path, false)
			known = !value.unknown
		} else if !known {
			value = producerUnknown()
		}

		choice = 0
		if !known && e.argumentsKnown && strings.HasPrefix(path, "arguments.") {
			choice = -1
		}

		if known && value.presenceKnown {
			if value.absent {
				choice = -1
			} else {
				choice = 1
			}
		}

		if known {
			present := value
			present.presenceKnown = true
			present.absent = false
			yes[path] = present
			no[path] = producerValue{unknown: true, absent: true, presenceKnown: true}
		}

		if negate {
			return no, yes, -choice, true
		}

		return yes, no, choice, true
	}

	if path, matched := e.unaryGuardCondition(tokens, "isSimpleValue"); matched {
		yes, no = maps.Clone(env), maps.Clone(env)

		value := e.read(path, env)
		if strings.HasPrefix(path, "variables.") || strings.HasPrefix(path, "this.") {
			value = e.sharedFieldContract(path, true)
		}

		object := value
		object.primitive = false
		primitive := producerValue{primitive: true}
		yes[normalizeProducerPath(path)] = primitive
		no[normalizeProducerPath(path)] = object

		if negate {
			return no, yes, 0, true
		}

		return yes, no, 0, true
	}

	if path, matched := e.unaryGuardCondition(tokens, "isNull"); matched {
		yes, no = maps.Clone(env), maps.Clone(env)

		value := e.read(path, env)
		if strings.HasPrefix(path, "variables.") || strings.HasPrefix(path, "this.") {
			value = e.sharedFieldContract(path, false)
		}

		missing := producerValue{unknown: true, absent: true, presenceKnown: true}
		value.presenceKnown = true
		value.absent = false
		yes[normalizeProducerPath(path)] = missing
		no[normalizeProducerPath(path)] = value

		if negate {
			return no, yes, 0, true
		}

		return yes, no, 0, true
	}

	return nil, nil, 0, false
}

func (e *producerEvaluation) unaryGuardCondition(tokens []parser.Token, name string) (string, bool) {
	if len(tokens) < 4 || !strings.EqualFold(tokens[0].Value, name) || tokens[1].Kind != parser.TokLParen || tokens[len(tokens)-1].Kind != parser.TokRParen || e.resolver.ResolveFunc(e.methodOwner(), name, e.baseDir) != nil {
		return "", false
	}

	path := producerPath(tokens[2 : len(tokens)-1])
	if path == "" {
		return "", false
	}

	return normalizeProducerPath(path), true
}

// structKeyCondition is the field `structKeyExists( scope, key )` tests: a
// literal key, or one the call makes literal, as getService( "helpers" )
// does for `structKeyExists( variables.services, name )` in a keyed cache.
func (e *producerEvaluation) structKeyCondition(tokens []parser.Token, env producerEnvironment) (string, bool) {
	if len(tokens) < 6 || !strings.EqualFold(tokens[0].Value, "structKeyExists") || tokens[1].Kind != parser.TokLParen || tokens[len(tokens)-1].Kind != parser.TokRParen || e.resolver.ResolveFunc(e.methodOwner(), "structKeyExists", e.baseDir) != nil {
		return "", false
	}

	parts := producerSplit(tokens[2 : len(tokens)-1])
	if len(parts) != 2 {
		return "", false
	}

	scope := producerPath(parts[0])

	name, ok := e.literalKey(parts[1], env)
	if scope == "" || !ok {
		return "", false
	}

	return normalizeProducerPath(scope + "." + name), true
}

// literalKey is the key tokens name, lowercased: a string literal, or a value
// the environment holds as a known literal. A key that could not be a field
// name on its own is not one.
func (e *producerEvaluation) literalKey(tokens []parser.Token, env producerEnvironment) (string, bool) {
	var name string

	switch {
	case len(tokens) == 1 && tokens[0].Kind == parser.TokString:
		name = strings.Trim(tokens[0].Value, "\"'")
	case len(tokens) > 0 && tokens[0].Kind == parser.TokIdent:
		value := e.expression(producerText(tokens), env)
		if !value.literalKnown {
			return "", false
		}

		name = value.literal
	default:
		return "", false
	}

	name = strings.ToLower(name)
	if name == "" || strings.ContainsAny(name, ".#[]() \t") {
		return "", false
	}

	return name, true
}
