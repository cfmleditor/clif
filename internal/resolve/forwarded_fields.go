package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// A component handed its dependencies by its constructor often stores each
// through a private setter and reads it back through a private getter:
//
//	public any function init( required any presideObjectService ) {
//		_setPresideObjectService( arguments.presideObjectService );
//	}
//	private any function _getPresideObjectService() { return _presideObjectService; }
//	private void function _setPresideObjectService( required any presideObjectService ) {
//		_presideObjectService = arguments.presideObjectService;
//	}
//
// The parse types a field assigned a typed argument directly, but the
// setter's argument is untyped, so the field and the getter were not, and
// every `_getPresideObjectService().x()` was a finding: about a thousand over
// Preside's services, which take every dependency this way. The getter
// returns what the field holds when every write of the field in the file is
// `field = arguments.p` in a function whose p is a component: declared,
// injected (an `@p.inject` doc), or, for a private setter, passed by every
// call to it in the file as a whole argument of the caller that is one.

// forwardedFieldReturn is what fd returns when its body is `return field;`
// and the field holds one component as above, or "".
func (r *Resolver) forwardedFieldReturn(fd *parser.FunctionDef) string {
	method := r.producerFor(fd)
	if method == nil || len(method.body) != 1 || method.body[0].kind != "return" {
		return ""
	}

	field := producerPath(producerUnwrap(producerTokens(method.body[0].expression)))
	if field == "" || strings.Contains(field, ".") && !strings.HasPrefix(strings.ToLower(field), "variables.") {
		return ""
	}

	path := normalizeProducerPath(field)
	if !strings.Contains(path, ".") {
		path = "variables." + path
	}

	if strings.Count(path, ".") != 1 {
		return ""
	}

	file := fd.URI.Path()
	source := r.sourceInfo(file)
	methods := sourcePlans(source).all()

	eval := producerEvaluation{resolver: r, fd: fd}
	if !eval.fieldTextProved(&source, methods, path) {
		return ""
	}

	for _, a := range source.startupAssignments(fd.URI) {
		if normalizeProducerPath(a.Variable) == path || "variables."+normalizeProducerPath(a.Variable) == path {
			return ""
		}
	}

	component := ""
	writes := 0
	valid := true

	for name, m := range methods {
		walkProducerNodes(m.body, func(node *producerNode) {
			if node.kind != "set" {
				return
			}

			target := normalizeProducerPath(node.target)
			if !strings.Contains(target, ".") {
				target = "variables." + target
			}

			if target != path {
				return
			}

			writes++

			c := r.argumentHolds(file, name, node.expression, methods, 0)
			if c == "" || component != "" && !strings.EqualFold(c, component) {
				valid = false

				return
			}

			component = c
		})
	}

	if !valid || writes == 0 {
		return ""
	}

	return component
}

// argumentHolds is the component `arguments.p` (expression) holds inside the
// file's function fn, or "".
func (r *Resolver) argumentHolds(file, fn, expression string, methods map[string]*producerMethod, depth int) string {
	tokens := producerTokens(expression)
	if len(tokens) != 3 || !strings.EqualFold(tokens[0].Value, "arguments") || tokens[1].Kind != parser.TokDot || tokens[2].Kind != parser.TokIdent {
		return ""
	}

	param := tokens[2].Value

	fd := r.funcInFile(file, fn)
	if fd == nil {
		return ""
	}

	for i := range fd.Arguments {
		a := &fd.Arguments[i]
		if !strings.EqualFold(a.Name, param) {
			continue
		}

		if a.Component != "" {
			return a.Component
		}

		if parser.IsComponentType(a.Type) {
			return a.Type
		}

		break
	}

	if depth > 0 || r.sourceMethods(file)[strings.ToLower(fn)].public {
		return ""
	}

	return r.privateSetterArgument(file, fn, param, methods)
}

// privateSetterArgument is what every call to the private function fn in its
// file passes as param, when each is a statement of its own handing over one
// whole argument of the caller and that argument is a component, all the same.
// A call the plans did not read as such a statement (in an expression, a
// condition, a string) leaves it unknown.
func (r *Resolver) privateSetterArgument(file, fn, param string, methods map[string]*producerMethod) string {
	fd := r.funcInFile(file, fn)
	if fd == nil || len(fd.Arguments) == 0 || !strings.EqualFold(fd.Arguments[0].Name, param) {
		return ""
	}

	component := ""
	read := 0
	valid := true

	for caller, m := range methods {
		walkProducerNodes(m.body, func(node *producerNode) {
			if !mentionsCall(node.target+" "+node.expression, fn) {
				return
			}

			arg, ok := singleArgumentCall(node, fn, param)
			if !ok {
				valid = false

				return
			}

			read++

			c := r.argumentHolds(file, caller, arg, methods, 1)
			if c == "" || component != "" && !strings.EqualFold(c, component) {
				valid = false

				return
			}

			component = c
		})
	}

	if !valid || read == 0 || read != callsInText(withoutComments(r.sourceInfo(file).content), fn) {
		return ""
	}

	return component
}

// singleArgumentCall is the argument of an effect statement `fn( x )` or
// `fn( param = x )`.
func singleArgumentCall(node *producerNode, fn, param string) (string, bool) {
	if node.kind != "effect" {
		return "", false
	}

	tokens := producerTokens(node.expression)
	if len(tokens) < 4 || tokens[0].Kind != parser.TokIdent || !strings.EqualFold(tokens[0].Value, fn) || tokens[1].Kind != parser.TokLParen || tokens[len(tokens)-1].Kind != parser.TokRParen {
		return "", false
	}

	inner := tokens[2 : len(tokens)-1]
	if parts := producerSplit(inner); len(parts) != 1 {
		return "", false
	}

	if len(inner) > 2 && inner[0].Kind == parser.TokIdent && inner[1].Kind == parser.TokEquals {
		if !strings.EqualFold(inner[0].Value, param) {
			return "", false
		}

		inner = inner[2:]
	}

	return producerText(inner), true
}

// mentionsCall reports whether text calls fn.
func mentionsCall(text, fn string) bool {
	tokens := producerTokens(text)
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Kind == parser.TokIdent && strings.EqualFold(tokens[i].Value, fn) && tokens[i+1].Kind == parser.TokLParen {
			return true
		}
	}

	return false
}

// callsInText counts the calls to fn in content: the name followed by a
// parenthesis, other than where it is declared.
func callsInText(content, fn string) int {
	tokens := allTokens(content)
	n := 0

	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, fn) || tokens[i+1].Kind != parser.TokLParen {
			continue
		}

		if i > 0 && strings.EqualFold(tokens[i-1].Value, "function") {
			continue
		}

		n++
	}

	return n
}

// walkProducerNodes visits every node of a plan, nested ones included.
func walkProducerNodes(nodes []producerNode, visit func(*producerNode)) {
	for i := range nodes {
		visit(&nodes[i])
		walkProducerNodes(nodes[i].body, visit)
		walkProducerNodes(nodes[i].alternative, visit)
	}
}
