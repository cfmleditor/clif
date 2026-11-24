package resolve

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// Values retain object identity separately from primitive, absent and unknown
// alternatives. A concrete receiver requires every surviving return to agree.
type producerValue struct {
	components                                []string
	primitive, unknown, absent, presenceKnown bool
	literal                                   string
	literalKnown                              bool
	fields                                    map[string]producerValue
	instance                                  *producerInstance
}

func producerUnknown() producerValue { return producerValue{unknown: true} }
func producerMerge(a, b producerValue) producerValue {
	presenceKnown := a.presenceKnown && b.presenceKnown && a.absent == b.absent

	absent := a.absent && b.absent
	if a.unknown || b.unknown {
		return producerValue{unknown: true, presenceKnown: presenceKnown, absent: absent}
	}

	result := producerValue{literal: a.literal, literalKnown: a.literalKnown && b.literalKnown && a.literal == b.literal, primitive: a.primitive || b.primitive, components: slices.Clone(a.components), presenceKnown: presenceKnown, absent: absent}
	if a.instance == b.instance {
		result.instance = a.instance
	}

	for _, comp := range b.components {
		if !slices.Contains(result.components, comp) {
			result.components = append(result.components, comp)
		}
	}

	if len(result.components) > 8 {
		return producerUnknown()
	}

	return result
}

func (v producerValue) component() string {
	if v.unknown || v.primitive || len(v.components) != 1 {
		return ""
	}

	return v.components[0]
}

type producerEnvironment map[string]producerValue

func mergeProducerEnvironments(a, b producerEnvironment) producerEnvironment {
	result := maps.Clone(a)
	for key, value := range b {
		before, ok := a[key]
		if !ok {
			before = producerUnknown()
		}

		result[key] = producerMerge(before, value)
	}

	for key := range a {
		if _, ok := b[key]; !ok {
			result[key] = producerUnknown()
		}
	}

	return result
}

type producerEvaluation struct {
	resolver          *Resolver
	fd                *parser.FunctionDef
	receiver, baseDir string
	depth             int
	budget            *int
	wanted            map[string]bool
	argumentsKnown    bool
	callerArguments   bool
	instance          *producerInstance
}
type producerResult struct {
	environment producerEnvironment
	returns     []producerValue
	stopped     bool
	observed    producerEnvironment
}

func (r *Resolver) producerExpressionReturn(fd *parser.FunctionDef, expression, baseDir, component string) string {
	tokens := producerTokens(expression)
	open := -1

	for i, t := range tokens {
		if t.Kind == parser.TokLParen {
			open = i

			break
		}
	}

	if open < 0 || producerGroupEnd(tokens, open, parser.TokLParen, parser.TokRParen) != len(tokens)-1 {
		return ""
	}

	budget := 1024

	return r.producerExpressionValue(fd, expression, baseDir, component, 0, &budget).component()
}

func (r *Resolver) producerExpressionValue(fd *parser.FunctionDef, expression, baseDir, component string, depth int, budget *int) producerValue {
	tokens := producerTokens(expression)
	open := -1

	for i, t := range tokens {
		if t.Kind == parser.TokLParen {
			open = i

			break
		}
	}

	if open < 0 || producerGroupEnd(tokens, open, parser.TokLParen, parser.TokRParen) != len(tokens)-1 {
		return producerUnknown()
	}

	if r.producerIgnoresArguments(fd) {
		return producerUnknown()
	}

	eval := producerEvaluation{resolver: r, fd: fd, receiver: component, baseDir: baseDir, budget: budget, depth: depth, callerArguments: true}

	arguments, known := eval.arguments(tokens[open+1:len(tokens)-1], nil)
	if !known {
		return producerUnknown()
	}

	return eval.call(fd, component, arguments, true)
}

func (e *producerEvaluation) call(fd *parser.FunctionDef, receiver string, arguments map[string]producerValue, specific bool, instances ...*producerInstance) producerValue {
	if fd == nil || e.depth > 8 || *e.budget <= 0 {
		return producerUnknown()
	}

	*e.budget--

	// A method declared only to throw runs one of the overrides below it.
	if ret := e.resolver.abstractOverridesReturn(fd, e.depth+1, e.budget); ret != "" {
		return abstractReturnValue(ret)
	}

	method := e.resolver.producerFor(fd)

	forwarding := false
	if method != nil && len(instances) > 0 && instances[0] != nil && instances[0].valid {
		forwarding = !instanceSetter(method.body) && instanceSetterContract(fd.Name, sourcePlans(e.resolver.sourceInfo(fd.URI.Path())).all(), instances[0].values, nil)
	}

	if method != nil && len(instances) > 0 && instances[0] != nil && instances[0].valid && instanceSetter(method.body) {
		if !e.instanceSelfReturnCompatible(fd, receiver) {
			instances[0].valid = false

			return producerUnknown()
		}

		return e.updateInstance(fd, method, receiver, instances[0], arguments)
	}

	if !forwarding && specific && receiver != "" {
		if value, ok := e.selfModeReturn(fd, receiver, arguments); ok {
			return value
		}
	}

	returnType := strings.ToLower(fd.ReturnType)
	if forwarding {
		if !e.instanceSelfReturnCompatible(fd, receiver) {
			instances[0].valid = false

			return producerUnknown()
		}

		returnType = "any"
	}

	switch returnType {
	case "", "any", "object", "component":
	default:
		return e.declaredReturnValue(fd, receiver)
	}

	// Existing concrete/declared contracts take priority unless the source
	// contains an object guard, whose supplied-object path needs specialization.
	if !producerNeedsSpecialization(method, fd) && (len(instances) == 0 || instances[0] == nil || !instances[0].valid) {
		if fd.ReturnComponent != "" && fd.ReturnComponent != "$any" || len(fd.ReturnSources) > 0 || fd.DocReturn != "" {
			if ret := e.resolver.returnComponentOf(fd, e.depth+1, e.budget); ret != "" && ret != "$any" {
				if self := e.resolver.selfTyped(receiver, e.baseDir, fd, ret); self != "" {
					ret = self
				}

				return e.concrete(ret, filepath.Dir(fd.URI.Path()))
			}
		}
	}

	if ret := e.resolver.argumentFreeReturn(fd); ret != "" {
		return e.concrete(ret, filepath.Dir(fd.URI.Path()))
	}

	if method == nil || producerAugmentsReturn(method.body) && !e.resolver.producerScalarSelfSetter(method, fd, receiver) {
		return producerUnknown()
	}

	local := *e
	local.fd = fd
	local.callerArguments = false
	local.argumentsKnown = specific
	local.wanted = method.wanted

	local.instance = nil
	if len(instances) > 0 && instances[0] != nil && instances[0].valid {
		local.instance = instances[0]
	}

	local.receiver = receiver
	if receiver == "" && producerReturnsThis(method.body) {
		local.receiver = fd.URI.Path()
	}

	local.depth++

	return local.executeBoundCall(fd, method, arguments, specific, forwarding)
}

// declaredReturnValue is what a call to fd returns by its declared return
// type, which names a CFML type or a component.
func (e *producerEvaluation) declaredReturnValue(fd *parser.FunctionDef, receiver string) producerValue {
	if cfmlTypes[strings.ToLower(fd.ReturnType)] {
		return producerValue{primitive: true}
	}

	if ret := e.resolver.ReturnComponentOf(fd); ret != "" {
		if self := e.resolver.selfTyped(receiver, e.baseDir, fd, ret); self != "" {
			ret = self
		}

		return e.concrete(ret, filepath.Dir(fd.URI.Path()))
	}

	return producerUnknown()
}

// abstractReturnValue is the value abstractOverridesReturn's answer stands
// for: its alternatives, or nothing known for $any.
func abstractReturnValue(ret string) producerValue {
	if ret == "$any" {
		return producerUnknown()
	}

	return producerValue{components: strings.Split(ret, "|")}
}

// selfModeReturn is a specific call's answer from fd's self-returning mode
// parameter, when that decides it: unknown for a mode defaulted to anything
// but a literal, the receiver when the call selects the self mode.
func (e *producerEvaluation) selfModeReturn(fd *parser.FunctionDef, receiver string, arguments map[string]producerValue) (producerValue, bool) {
	source := e.resolver.sourceMethods(fd.URI.Path())[strings.ToLower(fd.Name)]
	if source.selfDefaultParam != "" && !source.literalDefaults {
		return producerUnknown(), true
	}

	if source.selfDefaultParam != "" && e.instanceSelfReturnCompatible(fd, receiver) && e.selfModeSelected(fd, source.selfDefaultParam, source.selfExcludedModes, arguments) {
		return e.concrete(receiver, e.baseDir), true
	}

	return producerValue{}, false
}

// argumentFreeReturn is what fd returns whatever it is passed, where the
// plan cannot show it: a factory caching under a key only it writes
// (ownedKeyReturn), or Preside's getObject() (presideObjectReturn).
func (r *Resolver) argumentFreeReturn(fd *parser.FunctionDef) string {
	if ret := r.ownedKeyReturn(fd); ret != "" {
		return ret
	}

	return r.presideObjectReturn(fd)
}

func (e *producerEvaluation) executeBoundCall(fd *parser.FunctionDef, method *producerMethod, arguments map[string]producerValue, specific, forwarding bool) producerValue {
	env := e.bindArguments(fd, method, arguments, specific)
	if e.instance != nil {
		if !e.instance.valid || !instanceArgumentsCompatible(fd, env) {
			e.instance.valid = false

			return producerUnknown()
		}

		maps.Copy(env, e.instance.values)
	}

	result := e.execute(method.body, env)
	if !result.stopped || len(result.returns) == 0 {
		if forwarding {
			e.instance.valid = false
		}

		return producerUnknown()
	}

	value := result.returns[0]
	for _, ret := range result.returns[1:] {
		value = producerMerge(value, ret)
	}

	if forwarding && value.instance != e.instance {
		e.instance.valid = false

		return producerUnknown()
	}

	return value
}

func (e *producerEvaluation) selfModeSelected(fd *parser.FunctionDef, name string, excluded []string, arguments map[string]producerValue) bool {
	position := -1

	for i := range fd.Arguments {
		if strings.EqualFold(fd.Arguments[i].Name, name) {
			position = i

			break
		}
	}

	if position < 0 {
		return false
	}

	value, supplied := arguments[name]
	if !supplied {
		value, supplied = arguments[producerPosition(position)]
	}

	if !supplied || value.presenceKnown && value.absent {
		return true
	}

	return value.literalKnown && !slices.Contains(excluded, strings.ToLower(value.literal))
}

// producerFor is fd's source plan. The plan is read from the file while fd
// comes from the index, which follows an unsaved buffer in the editor; a
// plan whose parameters are not fd's describes another version of the
// function, and binding arguments by position through it would misplace them.
func (r *Resolver) producerFor(fd *parser.FunctionDef) *producerMethod {
	method := sourcePlans(r.sourceInfo(fd.URI.Path())).get(strings.ToLower(fd.Name))
	if method == nil || len(method.parameters) != len(fd.Arguments) {
		return nil
	}

	for i, name := range method.parameters {
		if !strings.EqualFold(name, fd.Arguments[i].Name) {
			return nil
		}
	}

	return method
}

func (e *producerEvaluation) concrete(component, baseDir string) producerValue {
	if strings.HasPrefix(component, "$builtin.") {
		return producerValue{components: []string{component}}
	}

	if component == "" || strings.HasPrefix(component, "$") || strings.Contains(component, "|") {
		return producerUnknown()
	}

	if path := e.resolver.ComponentPath(component, baseDir); path != "" {
		component = path
	}

	return producerValue{components: []string{component}}
}
