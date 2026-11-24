package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// scriptProducerMethods plans each script function the parse found in
// content (spans): the functions declared as statements and the named ones
// nested in their bodies, each with a body. Tag functions are planned from
// their tags, and a function assigned to a name has no plan. A nested
// function does not displace a declared one of its name, which is often the
// method of an inline component inside it. Individual method plans stay
// bounded even in large classes.
func scriptProducerMethods(content string, spans []parser.FunctionSpan, methods map[string]*producerMethod) {
	scriptProducerMethodsFrom(content, spans, nil, methods)
}

// scriptProducerMethodsFrom is scriptProducerMethods with bodies, each
// span's body tokens (functionBodyTokens), or nil to read them here.
func scriptProducerMethodsFrom(content string, spans []parser.FunctionSpan, bodies [][]parser.Token, methods map[string]*producerMethod) {
	declared := map[string]bool{}

	for i := range spans {
		if spans[i].Kind == parser.FunctionDeclared && spans[i].Body >= 0 {
			declared[strings.ToLower(spans[i].Name)] = true
		}
	}

	for i := range spans {
		s := &spans[i]
		if s.Kind != parser.FunctionDeclared && s.Kind != parser.FunctionNested || s.Body < 0 || s.Params < 0 {
			continue
		}

		name := strings.ToLower(s.Name)
		if s.Kind == parser.FunctionNested && declared[name] {
			continue
		}
		// The body is script, so its strings are read as the parse reads
		// them, or a quote inside a #...# ends one early.
		var body []parser.Token
		if bodies != nil {
			body = bodies[i]
		} else {
			body = scriptTokens(s.BodyText(content))
		}

		if len(body) >= 4096 {
			methods[name] = &producerMethod{body: []producerNode{{kind: "unsafe", expression: producerText(body)}}}

			continue
		}

		methods[name] = scriptProducerMethod(producerSplit(scriptTokens(s.ParamsText(content))), body)
	}
}

// functionBodyTokens is the script tokens of the body of each script method
// in content (declared or nested, with a parameter list: what the Wheels
// reader and the producer plans read), by the span's index; nil for any
// other span.
func functionBodyTokens(content string, spans []parser.FunctionSpan) [][]parser.Token {
	bodies := make([][]parser.Token, len(spans))

	for i := range spans {
		if s := &spans[i]; (s.Kind == parser.FunctionDeclared || s.Kind == parser.FunctionNested) && s.Body >= 0 && s.Params >= 0 {
			bodies[i] = scriptTokens(s.BodyText(content))
		}
	}

	return bodies
}

// scriptTokens is text's tokens, read as CFScript.
func scriptTokens(text string) []parser.Token {
	var tokens []parser.Token

	scanner := parser.NewScriptScanner(text)
	for tok := scanner.NextSkipComments(); tok.Kind != parser.TokEOF; tok = scanner.NextSkipComments() {
		tokens = append(tokens, tok)
	}

	return tokens
}

// scriptProducerMethod is the plan of a script function with params and the
// tokens of its body.
func scriptProducerMethod(params [][]parser.Token, body []parser.Token) *producerMethod {
	method := &producerMethod{defaults: map[string]string{}}

	for _, param := range params {
		if len(param) == 0 {
			continue
		}

		eq := -1

		for j, t := range param {
			if t.Kind == parser.TokEquals {
				eq = j

				break
			}
		}

		header := param
		if eq >= 0 {
			header = param[:eq]
		}

		if len(header) == 0 {
			continue
		}

		key := strings.ToLower(header[len(header)-1].Value)

		method.parameters = append(method.parameters, key)
		if eq >= 0 {
			method.defaults[key] = producerText(param[eq+1:])
		}
	}

	p := producerScript{tokens: body}

	method.body = p.block()
	if p.valid() {
		method.sensitive = (producerSensitive(method.body) || producerCallBinding(method.body)) && !producerReturnsThis(method.body)
		method.wanted = producerWanted(method.body)
	} else {
		method.body = []producerNode{{kind: "unsafe", expression: producerText(body)}}
		method.sensitive = producerSensitive(method.body)
	}

	return method
}

func producerGroupEnd(tokens []parser.Token, start int, open, closing parser.TokenKind) int {
	return parser.GroupEnd(tokens, start, open, closing)
}

func producerSplit(tokens []parser.Token) [][]parser.Token {
	result := [][]parser.Token{}
	start, depth := 0, 0

	for i, t := range tokens {
		switch t.Kind {
		case parser.TokLParen, parser.TokLBrace, parser.TokLBracket:
			depth++
		case parser.TokRParen, parser.TokRBrace, parser.TokRBracket:
			depth--
		default:
		}

		if depth == 0 && t.Kind == parser.TokComma {
			result = append(result, tokens[start:i])
			start = i + 1
		}
	}

	return append(result, tokens[start:])
}

type producerScript struct {
	tokens   []parser.Token
	position int
	failed   bool
}

func (p *producerScript) valid() bool { return !p.failed && p.position == len(p.tokens) }

// takeParens consumes a parenthesised group and returns what it holds.
func (p *producerScript) takeParens() []parser.Token {
	if p.position >= len(p.tokens) || p.tokens[p.position].Kind != parser.TokLParen {
		p.failed = true

		return nil
	}

	end := producerGroupEnd(p.tokens, p.position, parser.TokLParen, parser.TokRParen)
	if end < 0 {
		p.failed = true
		p.position = len(p.tokens)

		return nil
	}

	group := p.tokens[p.position+1 : end]
	p.position = end + 1

	return group
}

func (p *producerScript) block() []producerNode {
	nodes := []producerNode{}
	for !p.failed && p.position < len(p.tokens) && p.tokens[p.position].Kind != parser.TokRBrace {
		nodes = append(nodes, p.statement())
	}

	return nodes
}

func (p *producerScript) statement() producerNode {
	tok := p.tokens[p.position]
	p.position++

	if tok.Kind == parser.TokSemicolon {
		return producerNode{kind: "noop"}
	}

	if tok.Kind == parser.TokLBrace {
		body := p.block()
		if p.position >= len(p.tokens) {
			p.failed = true

			return producerNode{}
		}

		p.position++

		return producerNode{kind: "block", body: body}
	}

	switch strings.ToLower(tok.Value) {
	case "if":
		return p.ifStatement()
	case "for", "while":
		return p.loopStatement()
	case "try":
		return p.tryStatement()
	case "switch":
		return p.switchStatement()
	case "function", "do", "finally", "break", "continue":
		p.failed = true

		return producerNode{}
	}

	return p.expressionStatement()
}

// ifStatement reads what follows an `if`: the condition, the statement it
// guards, and the one after an `else`. Running out of input anywhere fails
// the plan.
func (p *producerScript) ifStatement() producerNode {
	condition := producerText(p.takeParens())
	if p.position >= len(p.tokens) {
		p.failed = true

		return producerNode{}
	}

	body := []producerNode{p.statement()}

	var alternative []producerNode

	if p.position < len(p.tokens) && strings.EqualFold(p.tokens[p.position].Value, "else") {
		p.position++
		if p.position >= len(p.tokens) {
			p.failed = true

			return producerNode{}
		}

		alternative = []producerNode{p.statement()}
	}

	return producerNode{kind: "if", expression: condition, body: body, alternative: alternative}
}

// loopStatement reads what follows a `for` or `while`. A header that changes
// what its own condition reads makes the loop unsafe to evaluate.
func (p *producerScript) loopStatement() producerNode {
	header := p.takeParens()
	target := ""

	if len(header) > 0 && strings.EqualFold(header[0].Value, "var") {
		header = header[1:]
	}

	if len(header) > 0 && header[0].Kind == parser.TokIdent {
		target = strings.ToLower(header[0].Value)
	}

	if p.position >= len(p.tokens) {
		p.failed = true

		return producerNode{}
	}

	body := []producerNode{p.statement()}

	if producerConditionMutation(header) {
		return producerNode{kind: "unsafe"}
	}

	return producerNode{kind: "loop", target: target, body: body, header: producerText(header)}
}

// tryStatement reads what follows a `try`: the body and every catch clause.
// A `finally` fails the plan, as a body with no catch does.
func (p *producerScript) tryStatement() producerNode {
	if p.position >= len(p.tokens) {
		p.failed = true

		return producerNode{}
	}

	body := []producerNode{p.statement()}
	if p.position >= len(p.tokens) || !strings.EqualFold(p.tokens[p.position].Value, "catch") {
		p.failed = true

		return producerNode{}
	}

	p.position++
	p.takeParens()

	if p.position >= len(p.tokens) {
		p.failed = true

		return producerNode{}
	}

	catches := [][]producerNode{{p.statement()}}

	// Every further catch clause is another handler the same failure can
	// reach. Read as one, the second was an expression statement that ran
	// on into the statement after it, so ColdBox's execute() lost its
	// `requestContext = getRequestContext();` and kept a stale value.
	for p.position < len(p.tokens) && strings.EqualFold(p.tokens[p.position].Value, "catch") {
		p.position++
		p.takeParens()

		if p.position >= len(p.tokens) {
			p.failed = true

			return producerNode{}
		}

		catches = append(catches, []producerNode{p.statement()})
	}

	if p.position < len(p.tokens) && strings.EqualFold(p.tokens[p.position].Value, "finally") {
		p.failed = true
	}

	return producerNode{kind: "try", body: body, alternative: producerHandlers(catches)}
}

// Dispatch is limited to literal string labels. Keep
// top-level breaks on the arm rather than the method: falling through an arm
// continues the switch, while a return still exits the method. Nested breaks
// remain unsupported by statement(), so they cannot be mistaken for returns.
func (p *producerScript) switchStatement() producerNode {
	node := producerNode{kind: "switch", expression: producerText(p.takeParens())}
	if p.position >= len(p.tokens) || p.tokens[p.position].Kind != parser.TokLBrace {
		p.failed = true

		return node
	}

	p.position++
	seen := map[string]bool{}

	for !p.failed && p.position < len(p.tokens) && p.tokens[p.position].Kind != parser.TokRBrace {
		label := p.tokens[p.position]
		p.position++

		arm := producerNode{kind: strings.ToLower(label.Value)}
		switch arm.kind {
		case "case":
			if p.position >= len(p.tokens) || p.tokens[p.position].Kind != parser.TokString || strings.Contains(p.tokens[p.position].Value, "#") {
				p.failed = true

				return node
			}

			arm.expression = p.tokens[p.position].Value
			if len(arm.expression) < 2 || arm.expression[len(arm.expression)-1:] != arm.expression[:1] {
				p.failed = true

				return node
			}

			p.position++
		case "default":
		default:
			p.failed = true

			return node
		}

		key := arm.kind
		if arm.kind == "case" {
			quote := arm.expression[:1]
			key += strings.ToLower(strings.ReplaceAll(arm.expression[1:len(arm.expression)-1], quote+quote, quote))
		}

		if seen[key] || p.position >= len(p.tokens) || p.tokens[p.position].Kind != parser.TokColon {
			p.failed = true

			return node
		}

		seen[key] = true

		p.position++
		for !p.failed && p.position < len(p.tokens) && p.tokens[p.position].Kind != parser.TokRBrace && !strings.EqualFold(p.tokens[p.position].Value, "case") && !strings.EqualFold(p.tokens[p.position].Value, "default") {
			if strings.EqualFold(p.tokens[p.position].Value, "break") {
				p.position++
				if p.position < len(p.tokens) && p.tokens[p.position].Kind == parser.TokSemicolon {
					p.position++
				}

				arm.target = "break"

				break
			}

			arm.body = append(arm.body, p.statement())
		}

		node.body = append(node.body, arm)
	}

	if p.position >= len(p.tokens) {
		p.failed = true
	} else {
		p.position++
	}

	return node
}
