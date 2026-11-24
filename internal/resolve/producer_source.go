package resolve

import (
	"hash/maphash"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/parser"
)

type producerNode struct {
	kind               string
	target, expression string
	body, alternative  []producerNode

	// header is a loop's header (its parenthesised clause, or its tag's
	// attributes), which nothing evaluates. It is kept so a reader looking
	// for a name in the plan sees one written there (callbackWalk).
	header string
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
	return newProducerPlans(content, parser.FunctionSpansOf("", content)).all()
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

// newProducerPlans is content's plans: its script functions' from the
// functions the parse found in it (spans, its FunctionSpans), its tag
// functions' from its tags.
func newProducerPlans(content string, spans []parser.FunctionSpan) *producerPlans {
	return newProducerPlansFrom(content, spans, nil)
}

// newProducerPlansFrom is newProducerPlans with bodies, each span's body
// tokens (functionBodyTokens), or nil to read them here.
func newProducerPlansFrom(content string, spans []parser.FunctionSpan, bodies [][]parser.Token) *producerPlans {
	p := &producerPlans{script: map[string]*producerMethod{}, made: map[string]*producerMethod{}}

	scriptProducerMethodsFrom(content, spans, bodies, p.script)

	for _, region := range parser.ClassifyRegions(content) {
		if region.Kind == parser.RegionScript {
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
	scanner := parser.NewSourceScanner(source)

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
