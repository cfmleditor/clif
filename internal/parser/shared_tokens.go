package parser

import (
	"slices"
	"sync"
	"unsafe"
)

const (
	// sharedTokenMin is the shortest script text whose tokens are kept for
	// the parses and readers that come after: a short text tokenizes faster
	// than it is compared, and it is the large files, a framework's base
	// component, that a batch scan parses over and over (the index, the
	// scan, a caller's inference, each with options of its own).
	sharedTokenMin = 8 << 10
	// sharedTokenSlots is how many texts are held, and sharedTokenBytes how
	// much the token lists of them may come to: a scan's workers each parse a
	// file of their own at once.
	sharedTokenSlots = 16
	sharedTokenBytes = 64 << 20
)

type sharedTokenSlot struct {
	text  string
	toks  *[]Token
	bytes int
}

// tokenShare holds the tokens of the large script texts asked about most
// recently, as tokenize makes them. They are a function of the text alone, so
// every parse and reader of one text may read the one list; it is never
// written, and never returned to tokenPool.
type tokenShare struct {
	limit int // the most bytes of tokens held
	mu    sync.Mutex
	next  int
	held  int
	slots [sharedTokenSlots]sharedTokenSlot
}

var sharedTokens = tokenShare{limit: sharedTokenBytes}

// DefaultSharedTokenBytes is how much token memory a batch scan may keep.
const DefaultSharedTokenBytes = sharedTokenBytes

// SetSharedTokenLimit bounds the bytes of tokens kept for the next parse of a
// large text, dropping the oldest texts to fit, and returns the bound it
// replaced. A long-lived process sets it small, since its texts are mostly
// superseded versions of a document; a batch scan wants the default, for its
// workers each parsing a file of their own at once.
func SetSharedTokenLimit(bytes int) (prev int) {
	sharedTokens.mu.Lock()
	defer sharedTokens.mu.Unlock()

	prev, sharedTokens.limit = sharedTokens.limit, bytes

	for range sharedTokenSlots {
		if sharedTokens.held <= sharedTokens.limit {
			break
		}

		sharedTokens.drop()
	}

	return prev
}

// SharedTokenLimit is the bound SetSharedTokenLimit set.
func SharedTokenLimit() int {
	sharedTokens.mu.Lock()
	defer sharedTokens.mu.Unlock()

	return sharedTokens.limit
}

// find is text's list, if held. The lock is held.
func (c *tokenShare) find(text string) *[]Token {
	for i := range c.slots {
		if s := &c.slots[i]; s.toks != nil && len(s.text) == len(text) && s.text == text {
			return s.toks
		}
	}

	return nil
}

// store keeps toks for text, the oldest slot making room. The lock is held.
func (c *tokenShare) store(text string, toks *[]Token, n int) {
	for c.held+n > c.limit {
		c.drop()
	}

	if c.slots[c.next].toks != nil {
		c.held -= c.slots[c.next].bytes
	}

	c.slots[c.next] = sharedTokenSlot{text: text, toks: toks, bytes: n}
	c.held += n
	c.next = (c.next + 1) % sharedTokenSlots
}

// drop empties the slot next in line.
func (c *tokenShare) drop() {
	s := &c.slots[c.next]
	c.held -= s.bytes
	*s = sharedTokenSlot{}
	c.next = (c.next + 1) % sharedTokenSlots
}

// scriptTokensOf is tokenize(text), as a list that must not be changed, kept
// for the next parse of a large text. A text is recognised by its bytes,
// which for the same file's content is a pointer comparison.
func scriptTokensOf(text string) *[]Token {
	if len(text) < sharedTokenMin {
		toks := tokenize(text)

		return &toks
	}

	sharedTokens.mu.Lock()
	toks := sharedTokens.find(text)
	sharedTokens.mu.Unlock()

	if toks != nil {
		return toks
	}

	made := slices.Clip(tokenize(text))

	n := len(made) * int(unsafe.Sizeof(Token{}))
	if n > sharedTokens.limit/2 {
		return &made
	}

	sharedTokens.mu.Lock()
	defer sharedTokens.mu.Unlock()

	// Another worker may have tokenized it meanwhile: one slot per text.
	if toks := sharedTokens.find(text); toks != nil {
		return toks
	}

	sharedTokens.store(text, &made, n)

	return &made
}
