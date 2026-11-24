package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/nativetype"
	"github.com/cfmleditor/clif/internal/parser"
)

func (e *producerEvaluation) assign(node *producerNode, env producerEnvironment) bool {
	_, existingLocal := env[normalizeProducerPath(node.target)]
	localTarget := strings.HasPrefix(strings.ToLower(node.target), "local.") || !strings.Contains(node.target, ".") && existingLocal

	sourcePath := producerPath(producerTokens(node.expression))
	if sourcePath == "arguments" || sourcePath == "variables" || sourcePath == "local" || sourcePath != "" && e.read(sourcePath, env).fields != nil {
		return false
	}

	target := normalizeProducerPath(node.target)
	if !strings.Contains(node.target, ".") {
		if _, local := env[target]; !local {
			if _, arg := env["arguments."+target]; arg {
				target = "arguments." + target
			} else {
				target = "variables." + target
			}
		}
	}

	e.effect(node.expression, env)

	for key := range env {
		if strings.HasPrefix(key, target+".") {
			env[key] = producerUnknown()
		}
	}

	for parent := target; strings.Contains(parent, "."); {
		parent, _, _ = strings.CutLast(parent, ".")
		if value, ok := env[parent]; ok && value.fields != nil {
			env[parent] = producerUnknown()
		}
	}

	// Keep another closed factory even when the final return does not read it:
	// its proved setters must retain their own identity and effects.
	if producerWantedTarget(node.target, e.wanted) || localTarget && e.closedConstructionEffect(producerTokens(node.expression), env) {
		env[target] = e.expression(node.expression, env)
		if value := env[target]; value.instance != nil && !localTarget {
			value.instance.valid = false
		}

		if strings.HasPrefix(target, "arguments.") {
			value := env[target]
			value.presenceKnown = true
			env[target] = value
		}
	} else {
		env[target] = producerValue{unknown: true, presenceKnown: strings.HasPrefix(target, "arguments.")}
	}

	return true
}

// producerIgnoresArguments reports that call would answer unknown for fd
// whatever arguments it were given: a generic return type, no declared or
// inferred return, no source plan and no self-mode parameter. Evaluating the
// arguments first resolves every call inside them, and a parse asks this
// once per call site with its own argument text, so on a large component
// that evaluation was most of what a return lookup cost.
func (r *Resolver) producerIgnoresArguments(fd *parser.FunctionDef) bool {
	if fd == nil || !fd.URI.IsFile() {
		return false
	}

	switch strings.ToLower(fd.ReturnType) {
	case "", "any", "object", "component":
	default:
		return false
	}

	if fd.ReturnComponent != "" && fd.ReturnComponent != "$any" || len(fd.ReturnSources) > 0 || fd.DocReturn != "" {
		return false
	}

	if r.hasSelfDefaultParam(fd) {
		return false
	}

	return r.producerFor(fd) == nil
}

func (e *producerEvaluation) methodResult(value producerValue, method string, args map[string]producerValue) producerValue {
	if ret, ok := docs.LookupBuiltinMethodReturn(value.component(), method); ok {
		return e.concrete(ret, e.baseDir)
	}

	fd := e.resolver.ResolveFunc(value.component(), method, e.baseDir)

	return e.call(fd, value.component(), args, true, value.instance)
}

func (e *producerEvaluation) createObjectResult(tokens []parser.Token, open, end int, env producerEnvironment) producerValue {
	if configured, noFollow, soft := e.resolver.matchResolver(producerCallText(tokens[:end+1]), nil, nil); configured != "" {
		if noFollow || soft && !e.resolver.componentExists(configured, e.baseDir) {
			return producerUnknown()
		}

		return e.concrete(configured, e.baseDir)
	}

	parts := producerSplit(tokens[open+1 : end])
	if len(parts) == 2 {
		kind := e.expression(producerText(parts[0]), env)

		name := e.expression(producerText(parts[1]), env)
		if kind.literalKnown && strings.EqualFold(kind.literal, "java") && name.literalKnown {
			return e.concrete(nativetype.Java(name.literal), e.baseDir)
		}

		if kind.literalKnown && strings.EqualFold(kind.literal, "component") && name.literalKnown {
			dir := filepath.Dir(e.fd.URI.Path())
			if e.callerArguments {
				dir = e.baseDir
			}

			return e.concrete(name.literal, dir)
		}
	}

	return producerUnknown()
}

// variablesRoot is the variable a variables-scope field path is held in:
// services for variables.services.helpers. "" for any other path.
func variablesRoot(path string) string {
	rest, ok := strings.CutPrefix(path, "variables.")
	if !ok || rest == "" {
		return ""
	}

	root, _, _ := strings.Cut(rest, ".")

	return root
}

// variablesReferences is how a source's text refers to its variables scope:
// how many tokens refer to each variable in it (variablesRootReferences), and
// whether the scope is ever read in a way that hides which variable:
// indexed with brackets, passed or assigned whole.
type variablesReferences struct {
	roots   map[string]int
	escapes bool
}

// proves reports whether read references to root, counted in the statements
// a plan read, are all the text holds, in a source whose scope never escapes:
// then no statement the plan could not read touches root.
func (v variablesReferences) proves(root string, read int) bool {
	return !v.escapes && read > 0 && v.roots[strings.ToLower(root)] == read
}

func countVariablesReferences(tokens []parser.Token) variablesReferences {
	v := variablesReferences{roots: map[string]int{}}

	for i := range tokens {
		t := tokens[i]
		if t.Kind != parser.TokIdent {
			continue
		}

		if strings.EqualFold(t.Value, "variables") && (i+1 >= len(tokens) || tokens[i+1].Kind != parser.TokDot) {
			if key, ok := scopeKeyArgument(tokens, i); ok {
				v.roots[key]++
			} else {
				v.escapes = true
			}
		}

		if variablesReference(tokens, i) {
			v.roots[strings.ToLower(t.Value)]++
		}
	}

	return v
}

// variablesRootReferences is how many tokens refer to the variable root.
func variablesRootReferences(tokens []parser.Token, root string) int {
	if root == "" {
		return 0
	}

	n := 0

	for i := range tokens {
		if strings.EqualFold(tokens[i].Value, root) && variablesReference(tokens, i) {
			n++
		}
	}

	return n
}

// variablesReference reports whether the identifier at tokens[i] names a
// variable of the variables scope: `variables.name`, or a bare name, but not
// a member of something else (`x.name`) or a component path (`new name.X`).
func variablesReference(tokens []parser.Token, i int) bool {
	if tokens[i].Kind != parser.TokIdent || strings.EqualFold(tokens[i].Value, "variables") {
		return false
	}

	if i == 0 {
		return true
	}

	prev := tokens[i-1]

	switch {
	case prev.Kind == parser.TokIdent && strings.EqualFold(prev.Value, "new"):
		return false
	case prev.Kind == parser.TokDot:
		return i >= 2 && tokens[i-2].Kind == parser.TokIdent && strings.EqualFold(tokens[i-2].Value, "variables") && (i < 3 || tokens[i-3].Kind != parser.TokDot)
	default:
		return true
	}
}

// producerStatementKey is an assignment's identity across a plan and the
// startup assignments: its target and its expression, spacing aside.
func producerStatementKey(target, expression string) string {
	return normalizeProducerPath(target) + "=" + strings.ToLower(strings.Join(strings.Fields(producerText(producerTokens(expression))), ""))
}

// scopeKeyArgument is the key a whole-scope call names at tokens[i]:
// `structDelete( variables, "x" )`, `structKeyExists( variables, "x" )`, which
// touch that key alone, lowercased. A reference to the key's variable, then.
func scopeKeyArgument(tokens []parser.Token, i int) (string, bool) {
	if i < 2 || i+3 >= len(tokens) || tokens[i-1].Kind != parser.TokLParen || tokens[i+1].Kind != parser.TokComma || tokens[i+2].Kind != parser.TokString || tokens[i+3].Kind != parser.TokRParen {
		return "", false
	}

	switch strings.ToLower(tokens[i-2].Value) {
	case "structdelete", "structkeyexists":
	default:
		return "", false
	}

	key := strings.ToLower(strings.Trim(tokens[i+2].Value, `"'`))
	if key == "" || strings.ContainsAny(key, "#.") {
		return "", false
	}

	return key, true
}
