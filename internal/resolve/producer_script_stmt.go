package resolve

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// producerStatementBreak reports whether a line break between prev and next
// ends a statement. CFScript does not require the semicolon, and ColdBox's
// request() omits it throughout; read as one statement, its body lost its
// return. The break is a new line whose first token is a word that is not an
// operator, after a token that can end an expression. A line starting with
// `.`, an operator or an operator word continues the statement, so a chain
// written one call per line stays one.
func producerStatementBreak(prev, next parser.Token) bool {
	if next.Line <= prev.Line || next.Kind != parser.TokIdent || producerOperatorWord(next.Value) {
		return false
	}

	switch prev.Kind {
	case parser.TokString, parser.TokNumber, parser.TokRParen, parser.TokRBracket:
		return true
	case parser.TokIdent:
		// A word that wants what follows it does not end an expression:
		// `return` with its value on the next line, `var`, `new`, an operator.
		return !producerOperatorWord(prev.Value) && !producerLeadingKeyword(prev.Value)
	default:
		return false
	}
}

func producerOperatorWord(word string) bool {
	switch strings.ToLower(word) {
	case "and", "or", "not", "xor", "eqv", "imp", "eq", "neq", "is", "gt", "lt", "gte", "lte", "ge", "le", "contains", "mod":
		return true
	default:
		return false
	}
}

func producerLeadingKeyword(word string) bool {
	switch strings.ToLower(word) {
	case "return", "var", "new", "throw", "in", "case", "else":
		return true
	default:
		return false
	}
}

// producerAugmentsReturn reports whether the plan writes a member onto a
// variable it returns: `requestContext.getRenderedContent = …; return
// requestContext;`. The value returned then has members its component does not
// declare, and typed as the plain component every call to one would be
// reported missing. ColdBox's execute() adds three test helpers this way.
func producerAugmentsReturn(nodes []producerNode) bool {
	returned := map[string]bool{}

	var collect func([]producerNode)

	collect = func(nodes []producerNode) {
		for i := range nodes {
			if nodes[i].kind == "return" {
				if path := producerPath(producerUnwrap(producerTokens(nodes[i].expression))); path != "" {
					returned[normalizeProducerPath(path)] = true
				}
			}

			collect(nodes[i].body)
			collect(nodes[i].alternative)
		}
	}
	collect(nodes)

	var writes func([]producerNode) bool

	writes = func(nodes []producerNode) bool {
		for i := range nodes {
			if nodes[i].kind == "set" {
				if root, _, member := strings.Cut(normalizeProducerPath(nodes[i].target), "."); member && returned[root] {
					return true
				}
			}

			if writes(nodes[i].body) || writes(nodes[i].alternative) {
				return true
			}
		}

		return false
	}

	return writes(nodes)
}

// A scalar setter adds state, rather than a callable member, to this object.
// Restrict the proof to one assignment from a declared scalar parameter and
// an immediate self return, so argument mutation and callback installation
// cannot be mistaken for the same contract.
func (r *Resolver) producerScalarSelfSetter(method *producerMethod, fd *parser.FunctionDef, receiver string) bool {
	if method == nil || fd == nil || len(method.body) != 2 {
		return false
	}

	assignment, returned := &method.body[0], &method.body[1]
	if assignment.kind != "set" || !strings.HasPrefix(assignment.target, "this.") || strings.Count(assignment.target, ".") != 1 || returned.kind != "return" || producerPath(producerUnwrap(producerTokens(returned.expression))) != "this" {
		return false
	}

	if receiver == "" {
		receiver = fd.URI.Path()
	}

	if r.ResolveFunc(receiver, strings.TrimPrefix(assignment.target, "this."), filepath.Dir(fd.URI.Path())) != nil {
		return false
	}

	path := producerPath(producerTokens(assignment.expression))

	name, ok := strings.CutPrefix(path, "arguments.")
	if !ok {
		return false
	}

	for _, argument := range fd.Arguments {
		if !strings.EqualFold(argument.Name, name) {
			continue
		}

		switch strings.ToLower(argument.Type) {
		case "string", "numeric", "boolean":
			return true
		default:
			return false
		}
	}

	return false
}

// producerHandlers is a try's catch clauses as one alternative: each starts
// from the state the failure left, and which one runs is not in the source,
// so they are the branches of an if whose condition is unknown.
func producerHandlers(catches [][]producerNode) []producerNode {
	alternative := catches[len(catches)-1]
	for i := len(catches) - 2; i >= 0; i-- {
		alternative = []producerNode{{kind: "if", body: catches[i], alternative: alternative}}
	}

	return alternative
}

func (p *producerScript) expressionStatement() producerNode {
	start := p.position - 1
	tok := p.tokens[start]
	depth := 0

	for p.position < len(p.tokens) {
		t := p.tokens[p.position]
		if depth == 0 && (t.Kind == parser.TokSemicolon || t.Kind == parser.TokRBrace) {
			break
		}

		if depth == 0 && p.position > start+1 && producerStatementBreak(p.tokens[p.position-1], t) {
			break
		}

		switch t.Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--
		default:
		}

		p.position++
	}

	expression := p.tokens[start:p.position]
	if p.position < len(p.tokens) && p.tokens[p.position].Kind == parser.TokSemicolon {
		p.position++
	}

	if strings.EqualFold(tok.Value, "return") {
		return producerNode{kind: "return", expression: producerText(expression[1:])}
	}

	if strings.EqualFold(tok.Value, "throw") || strings.EqualFold(tok.Value, "abort") {
		return producerNode{kind: "stop"}
	}

	return producerAssignment(expression)
}

func producerAssignment(tokens []parser.Token) producerNode {
	if node, ok := paramStatement(tokens); ok {
		return node
	}

	local := len(tokens) > 0 && strings.EqualFold(tokens[0].Value, "var")
	if local {
		tokens = tokens[1:]
	}

	for i, t := range tokens {
		if t.Kind == parser.TokLParen {
			break
		}

		if t.Kind == parser.TokEquals && i > 0 {
			if i+1 < len(tokens) && tokens[i+1].Kind == parser.TokEquals {
				break
			}

			target := producerPath(tokens[:i])
			if target != "" {
				if !local {
					if node, ok := producerChainedAssignment(tokens, i, target); ok {
						return node
					}
				}

				if local {
					target = "local." + target
				}

				return producerNode{kind: "set", target: target, expression: producerText(tokens[i+1:])}
			}

			return producerIndexedAssignment(tokens, i)
		}
	}

	for i, t := range tokens {
		if t.Kind == parser.TokPlus || t.Kind == parser.TokMinus {
			if i+1 < len(tokens) && tokens[i+1].Kind == t.Kind {
				return producerNode{kind: "unsafe"}
			}
		}
	}

	if local {
		if name := producerPath(tokens); name != "" && !strings.Contains(name, ".") {
			return producerNode{kind: "set", target: "local." + name}
		}
	}

	return producerNode{kind: "effect", expression: producerText(tokens)}
}

func producerIndexedAssignment(tokens []parser.Token, at int) producerNode {
	if _, _, ok := instanceIndexTarget(tokens[:at]); ok {
		return producerNode{kind: "private-index-set", target: producerText(tokens[:at]), expression: producerText(tokens[at+1:])}
	}

	return producerNode{kind: "unsafe"}
}

func producerChainedAssignment(tokens []parser.Token, assignment int, target string) (producerNode, bool) {
	if !producerSharedSlot(target) {
		return producerNode{}, false
	}

	for j := assignment + 1; j < len(tokens) && tokens[j].Kind != parser.TokLParen; j++ {
		if tokens[j].Kind != parser.TokEquals {
			continue
		}

		right := producerPath(tokens[assignment+1 : j])
		if producerSharedSlot(right) && j+1 < len(tokens) && strings.EqualFold(tokens[j+1].Value, "new") {
			return producerNode{kind: "block", body: []producerNode{
				{kind: "set", target: right, expression: producerText(tokens[j+1:])},
				{kind: "set", target: target, expression: right},
			}}, true
		}

		break
	}

	return producerNode{}, false
}

func producerSharedSlot(path string) bool {
	scope, name, ok := strings.Cut(path, ".")

	return ok && name != "" && !strings.Contains(name, ".") && (scope == "this" || scope == "variables")
}

// paramStatement reads a cfscript param statement, in either form, as a
// param node: what the variable it names is given when absent.
func paramStatement(tokens []parser.Token) (producerNode, bool) {
	if len(tokens) < 2 || !strings.EqualFold(tokens[0].Value, "param") || tokens[1].Kind != parser.TokIdent {
		return producerNode{}, false
	}

	// cfscript `param x = value;` is the tag form's statement alias: a
	// default that only creates a variable the caller left absent. Reading
	// it as an assignment made it an opaque write to the whole component's
	// plan, and every guarded shared field lost its all-writes-agree
	// contract behind one the statement can never write (TestBox's
	// BaseReporter.getConsoleUtil went unresolved behind a `param url = {}`
	// in an unrelated method).
	at := -1

	for i := 1; i < len(tokens); i++ {
		if tokens[i].Kind == parser.TokEquals {
			at = i

			break
		}
	}

	target, attrs := paramAttributeTarget(tokens[1:])
	if !attrs {
		target = producerPath(tokens[1:max(2, at)])
	}

	if target != "" {
		return producerNode{kind: "param", target: target}, true
	}

	return producerNode{}, false
}

// paramAttributeTarget reads a param's attribute form — pairs of
// `attribute=value`, one of them name= with a string — and gives the variable
// name= names, with attrs true. `param name="x" default=0` defaults x; read as
// the alias form `param x = value`, its target was the word name. A value
// other than name='s may be any expression. A lone `name = "x"` pair is the
// attribute form too, as an engine reads it. A computed name gives "" with
// attrs true: the param's target is unknown.
func paramAttributeTarget(tokens []parser.Token) (target string, attrs bool) {
	if n := len(tokens); n > 0 && tokens[n-1].Kind == parser.TokSemicolon {
		tokens = tokens[:n-1]
	}

	pairs, name := 0, -1

	for i := 0; i < len(tokens); {
		if i+2 >= len(tokens) || tokens[i].Kind != parser.TokIdent || tokens[i+1].Kind != parser.TokEquals {
			return "", false
		}

		end := attributeValueEnd(tokens, i+2)
		if end < 0 {
			return "", false
		}

		if strings.EqualFold(tokens[i].Value, "name") {
			if end != i+3 || tokens[i+2].Kind != parser.TokString {
				return "", false
			}

			name = i + 2
		}

		pairs++
		i = end
	}

	if name < 0 || pairs == 1 && !strings.EqualFold(tokens[0].Value, "name") {
		return "", false
	}

	value := strings.Trim(tokens[name].Value, `"'`)
	if strings.Contains(value, "#") {
		return "", true
	}

	return producerPath(producerTokens(value)), true
}

// attributeValueEnd is the index past the attribute value starting at from:
// the next `ident =` at its own nesting level, or the end. -1 when the value
// is empty or its brackets do not balance.
func attributeValueEnd(tokens []parser.Token, from int) int {
	depth := 0

	for j := from; j < len(tokens); j++ {
		switch tokens[j].Kind {
		case parser.TokLParen, parser.TokLBracket, parser.TokLBrace:
			depth++
		case parser.TokRParen, parser.TokRBracket, parser.TokRBrace:
			depth--
			if depth < 0 {
				return -1
			}
		case parser.TokIdent:
			if depth == 0 && j > from && j+1 < len(tokens) && tokens[j+1].Kind == parser.TokEquals &&
				(j+2 >= len(tokens) || tokens[j+2].Kind != parser.TokEquals) {
				return j
			}
		default:
		}
	}

	if depth != 0 || from >= len(tokens) {
		return -1
	}

	return len(tokens)
}

func producerPath(tokens []parser.Token) string {
	if len(tokens) == 0 || len(tokens)%2 == 0 {
		return ""
	}

	for i, t := range tokens {
		if i%2 == 0 && t.Kind != parser.TokIdent || i%2 == 1 && t.Kind != parser.TokDot {
			return ""
		}
	}

	return strings.ToLower(strings.ReplaceAll(producerText(tokens), " ", ""))
}

func producerSensitive(nodes []producerNode) bool {
	for i := range nodes {
		n := &nodes[i]
		if n.kind == "switch" {
			return true
		}

		if n.kind == "if" || n.kind == "unsafe" {
			tokens := producerTokens(n.expression)
			for j := range tokens {
				if n.kind == "unsafe" && strings.EqualFold(tokens[j].Value, "switch") && j+1 < len(tokens) && tokens[j+1].Kind == parser.TokLParen {
					return true
				}

				if producerGuardSensitive(tokens, j) {
					return true
				}
			}
		}

		if producerSensitive(n.body) || producerSensitive(n.alternative) {
			return true
		}
	}

	return false
}

func producerGuardSensitive(tokens []parser.Token, i int) bool {
	if i+1 >= len(tokens) || tokens[i+1].Kind != parser.TokLParen {
		return false
	}

	name := tokens[i].Value
	if strings.EqualFold(name, "isObject") {
		return true
	}

	shared := i+4 < len(tokens) && tokens[i+2].Kind == parser.TokIdent && (strings.EqualFold(tokens[i+2].Value, "variables") || strings.EqualFold(tokens[i+2].Value, "this")) && tokens[i+3].Kind == parser.TokDot
	if strings.EqualFold(name, "isSimpleValue") || strings.EqualFold(name, "isNull") {
		return shared
	}

	if !strings.EqualFold(name, "structKeyExists") || i+3 >= len(tokens) || tokens[i+2].Kind != parser.TokIdent {
		return false
	}

	// Only a bare scope is a guard the interpreter reads (structKeyCondition):
	// structKeyExists(arguments.config, "x") tests a member of an argument,
	// and treating it as one sends a method's declared return through a plan
	// that cannot answer.
	if tokens[i+3].Kind != parser.TokComma {
		return false
	}

	if strings.EqualFold(tokens[i+2].Value, "arguments") {
		return true
	}

	return (strings.EqualFold(tokens[i+2].Value, "variables") || strings.EqualFold(tokens[i+2].Value, "this")) && i+4 < len(tokens) && tokens[i+4].Kind == parser.TokString && !strings.Contains(tokens[i+4].Value, "#")
}
