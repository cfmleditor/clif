package resolve

import (
	"hash/maphash"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/parser"
)

type producerNode struct {
	kind               string
	target, expression string
	body, alternative  []producerNode
}

type producerMethod struct {
	parameters []string
	defaults   map[string]string
	body       []producerNode
	sensitive  bool
	wanted     map[string]bool
}

// These source plans carry no inferred types. Calls interpret them with their
// own arguments and application context; the byte-checked lexical cache owns
// the plans and refreshes them when a method changes.
func producerMethods(content string) map[string]*producerMethod {
	return newProducerPlans(content).all()
}

// producerPlans is the source plans of a file's methods, made as they are
// asked for. A file's plans were all made the first time any method of it was
// looked up, and a lookup asks about a few: on a 65,000-line tag component,
// 1,183 plans and ~13MB kept for as long as the file was open. A script
// function's plan is still made with the rest, since finding its boundaries
// is most of making it; a tag function's is made on first request.
type producerPlans struct {
	mu     sync.Mutex
	script map[string]*producerMethod // made when the file is read
	tags   []producerTag              // the file's tags, which the tag plans are made from
	spans  map[string][2]int          // tag function → its tags; see tagProducerSpans
	made   map[string]*producerMethod // tag plans made so far
}

func newProducerPlans(content string) *producerPlans {
	p := &producerPlans{script: map[string]*producerMethod{}, made: map[string]*producerMethod{}}

	for _, region := range parser.ClassifyRegions(content) {
		if region.Kind == parser.RegionScript {
			scriptProducerMethods(region.Text, p.script)
			p.tags = append(p.tags, producerTag{name: "cfscript", body: region.Text})
		}

		if region.Kind == parser.RegionTag {
			p.tags = append(p.tags, producerTags(region.Text)...)
		}
	}

	p.spans = tagProducerSpans(p.tags)

	return p
}

// get is the plan of the method name, lowercased; nil when there is none. A
// tag function outranks a script function of the same name, as it always has.
func (p *producerPlans) get(name string) *producerMethod {
	if p == nil {
		return nil
	}

	span, ok := p.spans[name]
	if !ok {
		return p.script[name]
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	method, made := p.made[name]
	if !made {
		method = tagProducerPlan(p.tags[span[0]+1 : span[1]])
		p.made[name] = method
	}

	return method
}

// all is every plan in the file, for a reader that must see them all.
func (p *producerPlans) all() map[string]*producerMethod {
	if p == nil {
		return nil
	}

	methods := make(map[string]*producerMethod, len(p.script)+len(p.spans))
	maps.Copy(methods, p.script)

	for name := range p.spans {
		methods[name] = p.get(name)
	}

	return methods
}

// producerTokens is source's tokens, or nil past 4,096 of them. The producer
// tokenizes the same expression text every time it evaluates a method body,
// so answers are kept (producerTokenCache), as capacity-clipped slices that
// an append cannot write past: callers only read them.
func producerTokens(source string) []parser.Token {
	if tokens, ok := producerTokenCache.get(source); ok {
		return tokens
	}

	tokens := scanProducerTokens(source)
	producerTokenCache.put(source, tokens)

	return tokens
}

// producerTokenCache holds producerTokens' answers, up to maxProducerTokens
// tokens in all; past it, a shard starts again. Tokens depend only on the
// text, so it is shared by every resolver.
var producerTokenCache = &tokenCache{}

const maxProducerTokens = 1 << 17

// tokenShards splits tokenCache so a write locks one shard. With one lock its
// put was the most contended lock in a batch scan, 1.6s of tassweb's workers'
// time waiting, every miss's write blocking every worker's read.
const tokenShards = 16

type tokenCache struct {
	seed   maphash.Seed
	once   sync.Once
	shards [tokenShards]tokenShard
}

type tokenShard struct {
	mu      sync.RWMutex
	entries map[string][]parser.Token
	held    int
}

func (c *tokenCache) shard(source string) *tokenShard {
	c.once.Do(func() { c.seed = maphash.MakeSeed() })

	return &c.shards[maphash.String(c.seed, source)%tokenShards]
}

func (c *tokenCache) get(source string) ([]parser.Token, bool) {
	sh := c.shard(source)

	sh.mu.RLock()
	tokens, ok := sh.entries[source]
	sh.mu.RUnlock()

	return tokens, ok
}

func (c *tokenCache) put(source string, tokens []parser.Token) {
	if len(tokens) > maxProducerTokens/16 {
		return
	}

	sh := c.shard(source)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	if sh.entries == nil || sh.held+len(tokens) > maxProducerTokens/tokenShards {
		sh.entries = map[string][]parser.Token{}
		sh.held = 0
	}

	// The key is copied, so an entry does not keep the file it was cut from.
	sh.entries[strings.Clone(source)] = slices.Clip(tokens)
	sh.held += len(tokens)
}

func scanProducerTokens(source string) []parser.Token {
	scanner := parser.NewScanner(source)

	tokens := []parser.Token{}
	for len(tokens) <= 4096 {
		tok := scanner.NextSkipComments()
		if tok.Kind == parser.TokEOF {
			return tokens
		}

		tokens = append(tokens, tok)
	}

	return nil
}

func producerText(tokens []parser.Token) string {
	values := make([]string, len(tokens))
	for i, t := range tokens {
		values[i] = t.Value
	}

	return strings.Join(values, " ")
}

func scriptProducerMethods(source string, methods map[string]*producerMethod) {
	tokens := producerTokens(source)
	// Whole-file scanning finds function boundaries; individual method plans
	// remain bounded even in large classes.
	if tokens == nil {
		tokens = nil

		scanner := parser.NewScanner(source)
		for {
			tok := scanner.NextSkipComments()
			if tok.Kind == parser.TokEOF {
				break
			}

			tokens = append(tokens, tok)
		}
	}

	for i := 0; i+2 < len(tokens); i++ {
		if !strings.EqualFold(tokens[i].Value, "function") || tokens[i+1].Kind != parser.TokIdent || tokens[i+2].Kind != parser.TokLParen {
			continue
		}

		name := strings.ToLower(tokens[i+1].Value)

		end := producerGroupEnd(tokens, i+2, parser.TokLParen, parser.TokRParen)
		if end < 0 {
			return
		}

		params := producerSplit(tokens[i+3 : end])

		start := end + 1
		for start < len(tokens) && tokens[start].Kind != parser.TokLBrace && tokens[start].Kind != parser.TokSemicolon {
			start++
		}

		if start >= len(tokens) || tokens[start].Kind != parser.TokLBrace {
			continue
		}

		finish := producerGroupEnd(tokens, start, parser.TokLBrace, parser.TokRBrace)
		if finish < 0 {
			return
		}

		if finish-start > 4096 {
			methods[name] = &producerMethod{body: []producerNode{{kind: "unsafe", expression: producerText(tokens[start+1 : finish])}}}
			i = finish

			continue
		}

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

		p := producerScript{tokens: tokens[start+1 : finish]}

		method.body = p.block()
		if p.valid() {
			method.sensitive = (producerSensitive(method.body) || producerCallBinding(method.body)) && !producerReturnsThis(method.body)
			method.wanted = producerWanted(method.body)
			methods[name] = method
		} else {
			expression := producerText(tokens[start+1 : finish])
			method.body = []producerNode{{kind: "unsafe", expression: expression}}
			method.sensitive = producerSensitive(method.body)
			methods[name] = method
		}

		i = finish
	}
}

func producerGroupEnd(tokens []parser.Token, start int, open, closing parser.TokenKind) int {
	depth := 0

	for i := start; i < len(tokens); i++ {
		if tokens[i].Kind == open {
			depth++
		}

		if tokens[i].Kind == closing {
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
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
	case "for", "while":
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

		return producerNode{kind: "loop", target: target, body: body}
	case "try":
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
	case "switch":
		return p.switchStatement()
	case "function", "do", "finally", "break", "continue":
		p.failed = true

		return producerNode{}
	}

	return p.expressionStatement()
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

type producerTag struct{ name, body string }

func producerTags(source string) []producerTag {
	tags := []producerTag{}

	for i := 0; i < len(source); {
		if strings.HasPrefix(source[i:], "<!---") {
			end := strings.Index(source[i+5:], "--->")
			if end < 0 {
				return nil
			}

			i += 5 + end + 4

			continue
		}

		if source[i] != '<' || (!strings.HasPrefix(strings.ToLower(source[i:min(i+4, len(source))]), "<cf") && !strings.HasPrefix(strings.ToLower(source[i:min(i+5, len(source))]), "</cf")) {
			i++

			continue
		}

		start := i
		i++
		quote := byte(0)

	tagEnd:
		for i < len(source) {
			ch := source[i]
			switch {
			case quote != 0:
				if ch == quote {
					if i+1 < len(source) && source[i+1] == quote {
						i += 2

						continue
					}

					quote = 0
				}
			case ch == '\'' || ch == '"':
				quote = ch
			case ch == '>':
				break tagEnd
			}

			i++
		}

		if i >= len(source) {
			return nil
		}

		text := strings.TrimSpace(source[start+1 : i])
		i++

		end := 0
		for end < len(text) && text[end] != ' ' && text[end] != '\t' && text[end] != '\r' && text[end] != '\n' {
			end++
		}

		name := strings.ToLower(text[:end])
		if !strings.HasPrefix(strings.TrimPrefix(name, "/"), "cf") {
			continue
		}

		tags = append(tags, producerTag{name: name, body: strings.TrimSpace(strings.TrimSuffix(text[end:], "/"))})
	}

	return tags
}

func producerAttr(body, key string) (string, bool) {
	tokens := producerTokens(body)
	for i := 0; i+2 < len(tokens); i++ {
		if !strings.EqualFold(tokens[i].Value, key) || tokens[i+1].Kind != parser.TokEquals {
			continue
		}

		t := tokens[i+2]
		if t.Kind == parser.TokString && len(t.Value) >= 2 {
			quote := t.Value[:1]

			return strings.ReplaceAll(t.Value[1:len(t.Value)-1], quote+quote, quote), true
		}

		if t.Kind == parser.TokIdent || t.Kind == parser.TokNumber {
			return t.Value, true
		}
	}

	return "", false
}

// tagProducerSpans locates each tag function a plan can be made for: its
// lowercased name → the indexes in tags of its <cffunction> and </cffunction>.
// A name declared twice is its last declaration, as a map of plans filled in
// order kept it.
func tagProducerSpans(tags []producerTag) map[string][2]int {
	spans := map[string][2]int{}

	for i := 0; i < len(tags); i++ {
		if tags[i].name != "cffunction" {
			continue
		}

		name, ok := producerAttr(tags[i].body, "name")
		if !ok {
			continue
		}

		end := i + 1
		for end < len(tags) && tags[end].name != "/cffunction" {
			if tags[end].name == "cffunction" {
				break
			}

			end++
		}

		if end >= len(tags) || tags[end].name != "/cffunction" {
			continue
		}

		spans[strings.ToLower(name)] = [2]int{i, end}

		i = end
	}

	return spans
}

// tagProducerPlan is the plan of the tag function whose tags, between its
// <cffunction> and </cffunction>, are body.
func tagProducerPlan(body []producerTag) *producerMethod {
	method := &producerMethod{defaults: map[string]string{}}

	for _, tag := range body {
		if tag.name != "cfargument" {
			continue
		}

		arg, found := producerAttr(tag.body, "name")
		if !found {
			continue
		}

		arg = strings.ToLower(arg)
		method.parameters = append(method.parameters, arg)

		if def, found := producerAttr(tag.body, "default"); found {
			// A computed default is present even though its value is unknown.
			method.defaults[arg] = "'" + strings.ReplaceAll(def, "'", "''") + "'"
		}
	}

	p := producerTagParser{tags: body}

	if len(body)+1 <= 4096 {
		method.body = p.block()
	} else {
		p.failed = true
	}

	if !p.failed && p.position == len(p.tags) {
		method.sensitive = (producerSensitive(method.body) || producerCallBinding(method.body)) && !producerReturnsThis(method.body)
		method.wanted = producerWanted(method.body)
	} else {
		var source strings.Builder
		for _, tag := range body {
			source.WriteString(tag.body)
			source.WriteByte('\n')

			if (tag.name == "cfif" || tag.name == "cfscript") && producerSensitive([]producerNode{{kind: "if", expression: tag.body}}) {
				method.sensitive = true
			}
		}

		method.body = []producerNode{{kind: "unsafe", expression: source.String()}}
	}

	return method
}

type producerTagParser struct {
	tags     []producerTag
	position int
	failed   bool
}

func (p *producerTagParser) block() []producerNode {
	nodes := []producerNode{}

	for !p.failed && p.position < len(p.tags) {
		tag := p.tags[p.position]
		if strings.HasPrefix(tag.name, "/") || tag.name == "cfelse" || tag.name == "cfelseif" || tag.name == "cfcatch" {
			break
		}

		p.position++

		switch tag.name {
		case "cfscript":
			tokens := producerTokens(tag.body)
			if tokens == nil {
				nodes = append(nodes, producerNode{kind: "unsafe", expression: tag.body})

				continue
			}

			script := producerScript{tokens: tokens}

			body := script.block()
			if !script.valid() {
				nodes = append(nodes, producerNode{kind: "unsafe", expression: tag.body})

				continue
			}

			nodes = append(nodes, producerNode{kind: "block", body: body})
		case "cfsetting":
			if strings.ContainsAny(tag.body, "#(") {
				nodes = append(nodes, producerNode{kind: "unsafe"})
			}
		case "cfargument", "cfqueryparam", "cfthrow", "cfabort", "cfexit":
			if tag.name == "cfthrow" || tag.name == "cfabort" || tag.name == "cfexit" {
				nodes = append(nodes, producerNode{kind: "stop"})
			}
		case "cfset":
			nodes = append(nodes, producerAssignment(producerTokens(tag.body)))
		case "cfreturn":
			nodes = append(nodes, producerNode{kind: "return", expression: tag.body})
		case "cfif":
			nodes = append(nodes, p.branch(tag.body))
		case "cfloop":
			body := p.block()
			p.close("/cfloop")

			target, _ := producerAttr(tag.body, "index")
			if target == "" {
				target, _ = producerAttr(tag.body, "item")
			}

			nodes = append(nodes, producerNode{kind: "loop", target: strings.ToLower(target), body: body})
		case "cftry":
			body := p.block()

			var alternate []producerNode

			if p.position < len(p.tags) && p.tags[p.position].name == "cfcatch" {
				p.position++
				alternate = p.block()
				p.close("/cfcatch")
			}

			p.close("/cftry")

			nodes = append(nodes, producerNode{kind: "try", body: body, alternative: alternate})
		case "cfquery":
			body := p.block()
			p.close("/cfquery")

			target, _ := producerAttr(tag.body, "name")
			result, _ := producerAttr(tag.body, "result")
			attrs, _ := producerAttr(tag.body, "attributeCollection")

			if strings.Contains(target+result, "#") {
				p.failed = true
			}

			nodes = append(nodes, producerNode{kind: "query", target: target, expression: strings.Trim(attrs, "#"), body: body, alternative: []producerNode{{target: result}}})
		case "cflock", "cftransaction", "cfsavecontent":
			body := p.block()
			p.close("/" + tag.name)

			target, _ := producerAttr(tag.body, "name")
			if tag.name == "cfsavecontent" {
				target, _ = producerAttr(tag.body, "variable")
			}

			if target != "" {
				nodes = append(nodes, producerNode{kind: "set", target: strings.ToLower(target), expression: "''"})
			}

			nodes = append(nodes, producerNode{kind: "block", body: body})
		case "cfparam":
			target, _ := producerAttr(tag.body, "name")
			nodes = append(nodes, producerNode{kind: "param", target: strings.ToLower(target)})
		default:
			p.failed = true
		}
	}

	return nodes
}

func (p *producerTagParser) close(name string) {
	if p.position >= len(p.tags) || p.tags[p.position].name != name {
		p.failed = true

		return
	}

	p.position++
}

func (p *producerTagParser) branch(condition string) producerNode {
	body := p.block()

	var alternate []producerNode

	if p.position < len(p.tags) {
		switch p.tags[p.position].name {
		case "cfelse":
			p.position++
			alternate = p.block()
			p.close("/cfif")
		case "cfelseif":
			next := p.tags[p.position].body
			p.position++
			alternate = []producerNode{p.branch(next)}
		default:
			p.close("/cfif")
		}
	} else {
		p.failed = true
	}

	return producerNode{kind: "if", expression: condition, body: body, alternative: alternate}
}

// producerUnwrap is tokens without the parentheses around the whole of them:
// `return( this );` is how jsonSerializer, and many a CFML style guide, writes
// `return this;`.
func producerUnwrap(tokens []parser.Token) []parser.Token {
	for len(tokens) > 2 && tokens[0].Kind == parser.TokLParen && producerGroupEnd(tokens, 0, parser.TokLParen, parser.TokRParen) == len(tokens)-1 {
		tokens = tokens[1 : len(tokens)-1]
	}

	return tokens
}

func producerReturnsThis(nodes []producerNode) bool {
	found := false

	for i := range nodes {
		node := &nodes[i]
		if node.kind == "return" {
			if producerPath(producerUnwrap(producerTokens(node.expression))) != "this" {
				return false
			}

			found = true
		}

		for _, body := range [][]producerNode{node.body, node.alternative} {
			if producerHasReturns(body) {
				if !producerReturnsThis(body) {
					return false
				}

				found = true
			}
		}
	}

	return found
}

func producerHasReturns(nodes []producerNode) bool {
	for i := range nodes {
		n := &nodes[i]
		if n.kind == "return" || producerHasReturns(n.body) || producerHasReturns(n.alternative) {
			return true
		}
	}

	return false
}

func producerCallText(tokens []parser.Token) string {
	var result strings.Builder

	for i, t := range tokens {
		if i > 0 && t.Kind != parser.TokDot && t.Kind != parser.TokLParen && tokens[i-1].Kind != parser.TokDot {
			result.WriteByte(' ')
		}

		result.WriteString(t.Value)
	}

	return result.String()
}

// A bounded backward slice avoids evaluating bindings that cannot affect the
// returned identity or a guard/output name. Missing dependencies stay unknown.
func producerWanted(nodes []producerNode) map[string]bool {
	wanted := map[string]bool{}

	var seed func([]producerNode)

	seed = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "query" || node.kind == "return" || node.kind == "if" || node.kind == "switch" {
				producerReadPaths(node.expression, wanted)
			}

			seed(node.body)
			seed(node.alternative)
		}
	}
	seed(nodes)

	var visit func([]producerNode)

	visit = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "set" && producerWantedTarget(node.target, wanted) {
				producerReadPaths(node.expression, wanted)
			}

			visit(node.body)
			visit(node.alternative)
		}
	}

	for range 16 {
		count := len(wanted)

		visit(nodes)

		if len(wanted) == count {
			break
		}
	}

	return wanted
}

func producerReadPaths(expression string, wanted map[string]bool) {
	tokens := producerTokens(expression)
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent {
			continue
		}

		start := i
		for i+2 < len(tokens) && tokens[i+1].Kind == parser.TokDot && tokens[i+2].Kind == parser.TokIdent {
			i += 2
		}

		path := producerPath(tokens[start : i+1])
		if path != "" {
			wanted[normalizeProducerPath(path)] = true
		}
	}
}

func producerWantedTarget(target string, wanted map[string]bool) bool {
	target = normalizeProducerPath(target)
	for path := range wanted {
		if path == target || strings.HasPrefix(path, target+".") || strings.HasPrefix(target, path+".") {
			return true
		}

		if strings.TrimPrefix(target, "arguments.") == path || strings.TrimPrefix(target, "variables.") == path {
			return true
		}
	}

	return false
}

// A call-derived scalar return must not inherit the parser's closed-file
// receiver guess. Collection contracts remain handled by their uniform writers.
func producerCallBinding(nodes []producerNode) bool {
	wanted := map[string]bool{}

	var seed func([]producerNode)

	seed = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "return" {
				path := producerPath(producerTokens(node.expression))
				if path != "" && path != "this" {
					wanted[normalizeProducerPath(path)] = true
				}
			}

			seed(node.body)
			seed(node.alternative)
		}
	}
	seed(nodes)

	var expand func([]producerNode)

	expand = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "set" && producerWantedTarget(node.target, wanted) {
				producerReadPaths(node.expression, wanted)
			}

			expand(node.body)
			expand(node.alternative)
		}
	}
	stable := false

	for range 16 {
		count := len(wanted)

		expand(nodes)

		if len(wanted) == count {
			stable = true

			break
		}
	}

	if !stable {
		return true
	}

	var visit func([]producerNode) bool

	visit = func(nodes []producerNode) bool {
		for i := range nodes {
			node := &nodes[i]
			if node.kind == "set" && producerWantedTarget(node.target, wanted) && strings.Contains(node.expression, "(") && !strings.HasPrefix(strings.ToLower(strings.TrimSpace(node.expression)), "new ") {
				return true
			}

			if visit(node.body) || visit(node.alternative) {
				return true
			}
		}

		return false
	}

	return visit(nodes)
}

// Scalar-call specialization replaces component guesses, not the established
// builtin or dynamic contracts. Object guards still need per-call proof.
func producerNeedsSpecialization(method *producerMethod, fd *parser.FunctionDef) bool {
	if method == nil {
		return false
	}

	if fd.DocReturn == "" && fd.ReturnComponent != "$any" && !strings.HasPrefix(fd.ReturnComponent, "$builtin.") {
		for i := range method.body {
			node := &method.body[i]
			if node.kind != "return" {
				continue
			}

			tokens := producerUnwrap(producerTokens(node.expression))
			if len(tokens) == 0 || !strings.EqualFold(tokens[0].Value, "new") {
				continue
			}

			for j, token := range tokens {
				if token.Kind != parser.TokLParen {
					continue
				}

				end := producerGroupEnd(tokens, j, parser.TokLParen, parser.TokRParen)
				if end >= 0 && end+3 < len(tokens) && tokens[end+1].Kind == parser.TokDot && tokens[end+2].Kind == parser.TokIdent && tokens[end+3].Kind == parser.TokLParen {
					return true
				}

				break
			}
		}
	}

	if !method.sensitive {
		return false
	}

	return producerSensitive(method.body) || fd.ReturnComponent != "$any" && !strings.HasPrefix(fd.ReturnComponent, "$builtin.")
}
