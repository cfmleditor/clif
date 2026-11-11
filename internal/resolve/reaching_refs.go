package resolve

import (
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unsafe"

	"github.com/cfmleditor/clif/internal/parser"
)

// maxReachingAssignments bounds how many assignments to one name a use reads
// back through.
const maxReachingAssignments = 64

// reachingUnion is the component a use of name holds when more than one
// assignment to it can reach the use: the alternatives joined, as a return
// joins the branches that reach it (`a|b`). best is the latest assignment at
// or before the use, which is what the lookup answered before; it is the
// answer unless a branch hides assignments earlier in the same scope.
//
// The assignments are read back from the use until one that always runs
// (its block is still open at the use) or the scope's start; every one passed
// on the way is conditional and may be the value. When all are typed, with no
// native or dynamic component among them, the answer is their union. When
// some assignment on the way made no ref (it holds a string, a number, an
// argument) the value may be that; the answer stays best, which accepts the
// member functions of a built-in type on any component already
// (checkMethodOn) and checks every other method against the component. An
// assignment nothing dominates leaves the incoming value (an argument, a
// field) open, so then the answer stays best too.
//
// Blocks are braces in script and the bodies of <cfif>, <cfelseif>, <cfelse>,
// <cfloop>, <cftry>, <cfcatch>, <cfcase> and <cfdefaultcase> in tags
// (blockEvents).
func reachingUnion(pr *parser.ParseResult, scopeStart int, name string, line uint32, named func() []*parser.ComponentRef, best *parser.ComponentRef) *parser.ComponentRef {
	if best == nil || best.Component == "" || !strings.ContainsAny(pr.Content, "{<") {
		return best
	}

	use := int(line)

	typed := named()
	if len(typed) < 2 {
		return best
	}

	// The assignments that reach the use: the typed ones, and the untyped ones
	// the source shows at the start of a statement.
	var at []int

	facts := contentFacts(pr.Content)

	for header := use + 1; len(at) < maxReachingAssignments; header = at[len(at)-1] {
		_, l, ok := facts.assignmentAt(pr.Content, name, scopeStart, header)
		if !ok {
			break
		}

		at = append(at, l)
	}

	for _, ref := range typed {
		if l := int(ref.Line); l <= use && l >= scopeStart && !slices.Contains(at, l) {
			at = append(at, l)
		}
	}

	slices.SortFunc(at, func(a, b int) int { return b - a })

	if len(at) < 2 {
		return best
	}

	paths := blockPaths(pr.Content, scopeStart, use, name, at, tagBlocks(pr, scopeStart))

	var (
		comps     []string
		untyped   bool
		dominated bool
	)

	for _, l := range at {
		ref := refOnLine(typed, l)

		switch {
		case ref == nil || ref.Component == "":
			untyped = true
		case ref.HasClosureSpan || ref.VisibleTo != 0 || strings.HasPrefix(ref.Component, parser.NativeTypePrefix) || strings.HasPrefix(ref.Component, "$"):
			return best
		default:
			comps = append(comps, strings.Split(ref.Component, "|")...)
		}

		if isPrefixPath(paths[l], paths[use]) {
			dominated = true

			break
		}
	}

	if !dominated {
		return best
	}

	if untyped {
		return best
	}

	slices.Sort(comps)
	comps = slices.Compact(comps)

	if len(comps) < 2 {
		return best
	}

	joined := *best
	joined.Component = strings.Join(comps, "|")

	return &joined
}

// isPrefixPath reports whether the block chain a is the start of b: a block
// opened by the assignment's line is still open at the use.
func isPrefixPath(a, b []int) bool {
	return len(a) <= len(b) && slices.Equal(a, b[:len(a)])
}

// blockPaths is, for each of lines and for use, the chain of blocks open at
// the start of that line, counted from the line scopeStart. The block an id
// names is the nth opened, so two chains share an id only for the same block.
func blockPaths(content string, scopeStart, use int, name string, lines []int, tag bool) map[int][]int {
	paths := make(map[int][]int, len(lines)+1)

	from, ok := lineOffset(content, scopeStart)
	if !ok {
		return paths
	}

	end, ok := lineOffset(content, use+1)
	if !ok {
		end = len(content)
	}

	// An assignment's block is the one open where its name is written, which
	// is not the line's start when a tag on the line opened one first
	// (`<cfcase value="1"><cfset z = new B()></cfcase>`).
	type target struct{ line, offset int }

	targets := []target{}

	for _, l := range append([]int{use}, lines...) {
		start, ok := lineOffset(content, l)
		if !ok {
			continue
		}

		offset := start

		if l != use {
			stop, ok := lineOffset(content, l+1)
			if !ok {
				stop = len(content)
			}

			if i := parser.IndexFoldFrom(content[start:stop], name, 0); i >= 0 {
				offset = start + i
			}
		}

		targets = append(targets, target{l, offset})
	}

	slices.SortFunc(targets, func(a, b target) int { return a.offset - b.offset })

	var (
		stack []blockOpen
		next  int
		w     int
	)

	record := func(offset int) {
		for w < len(targets) && targets[w].offset-from <= offset {
			ids := make([]int, len(stack))
			for i := range stack {
				ids[i] = stack[i].id
			}

			paths[targets[w].line] = ids
			w++
		}
	}

	for _, ev := range scopeBlockEvents(content, from, tag) {
		if ev.offset >= end-from {
			break
		}

		record(ev.offset)

		switch ev.op {
		case opOpen:
			next++
			stack = append(stack, blockOpen{next, ev.kind})
		case opClose:
			if len(stack) > 0 && (len(ev.ends) == 0 || ev.closes(stack[len(stack)-1].kind)) {
				stack = stack[:len(stack)-1]
			}
		case opBranch:
			// The next branch of the chain the top block is in: the top
			// closes, a new one opens.
			if len(stack) > 0 && ev.closes(stack[len(stack)-1].kind) {
				stack = stack[:len(stack)-1]
			}

			next++
			stack = append(stack, blockOpen{next, ev.kind})
		}
	}

	record(len(content) - from)

	return paths
}

// maxScopeEvents is how many scopes' block events scopeBlockEvents keeps. A
// scan asks about one file's uses together, so a few per worker is enough.
const maxScopeEvents = 256

// scopeEventKey names a scope's text by where its bytes are, not by the bytes:
// hashing a whole file per lookup would cost what the cache saves. The entry
// holds the text, so the bytes cannot be freed and reused while it is kept.
type scopeEventKey struct {
	data *byte
	n    int
	from int
	tag  bool
}

// contentKey is the key for content's bytes, read from where they are. It
// only names them; the entry it keys holds content and compares it on a hit.
func contentKey(content string) scopeEventKey {
	return scopeEventKey{data: unsafe.StringData(content), n: len(content)} //nolint:gosec // the address is a cache key; the bytes are never written or read through it
}

type scopeEventEntry struct {
	content string
	events  []blockEvent
}

var scopeEvents = struct {
	sync.Mutex
	m map[scopeEventKey]scopeEventEntry
}{m: map[scopeEventKey]scopeEventEntry{}}

// scopeBlockEvents is blockEvents over content from the byte from to its end,
// read once per scope rather than once per use: reachingUnion is asked for
// every lookup of a name assigned more than once, and scanning a template or a
// long function from its start each time was quadratic in its length. A caller
// stops at the end it wants; the events before it are the ones a scan of the
// shorter text gives.
func scopeBlockEvents(content string, from int, tag bool) []blockEvent {
	key := contentKey(content)
	key.from, key.tag = from, tag

	scopeEvents.Lock()
	entry, ok := scopeEvents.m[key]
	scopeEvents.Unlock()

	if ok && entry.content == content {
		return entry.events
	}

	events := blockEvents(content[from:], tag)

	scopeEvents.Lock()
	if len(scopeEvents.m) >= maxScopeEvents {
		clear(scopeEvents.m)
	}

	scopeEvents.m[key] = scopeEventEntry{content, events}
	scopeEvents.Unlock()

	return events
}

var factsCache = struct {
	sync.Mutex
	m map[scopeEventKey]scopeFactsEntry
}{m: map[scopeEventKey]scopeFactsEntry{}}

type scopeFactsEntry struct {
	content string
	facts   *lexicalSourceFacts
}

// contentFacts is scanSourceFacts(content), kept for the content's recent
// uses as scopeBlockEvents keeps events: reachingUnion has no resolver to ask
// for the table, and walking the lines above each use to its scope's start
// was a second of a tassweb scan.
func contentFacts(content string) *lexicalSourceFacts {
	key := contentKey(content)

	factsCache.Lock()
	entry, ok := factsCache.m[key]
	factsCache.Unlock()

	if ok && entry.content == content {
		return entry.facts
	}

	facts := scanSourceFacts(content)

	factsCache.Lock()
	if len(factsCache.m) >= maxScopeEvents {
		clear(factsCache.m)
	}

	factsCache.m[key] = scopeFactsEntry{content, &facts}
	factsCache.Unlock()

	return &facts
}

// blockOpen is one open block: its id and the tag or brace that opened it.
type blockOpen struct {
	id   int
	kind string
}

type blockOp int

const (
	opOpen blockOp = iota
	opClose
	opBranch
)

// blockEvent is a block opening, closing or giving way to its next branch, at
// an offset in the text scanned.
type blockEvent struct {
	offset int
	op     blockOp
	kind   string
	// closes is the kinds of block a branch ends when it is on top.
	ends []string
}

func (e blockEvent) closes(top string) bool {
	return slices.Contains(e.ends, top)
}

// blockEvents reads the blocks of text in order: the braces of CFScript, and
// the bodies of the tags that branch or repeat. Braces count only in script
// (everywhere in a script file, inside <cfscript> in a tag file), so a
// stylesheet or a <script> block does not open one.
func blockEvents(text string, tag bool) []blockEvent {
	if !tag {
		return braceEvents(text, 0)
	}

	var events []blockEvent

	for i := 0; i < len(text); {
		lt := strings.IndexByte(text[i:], '<')
		if lt < 0 {
			break
		}

		i += lt

		if strings.HasPrefix(text[i:], "<!---") {
			end := strings.Index(text[i:], "--->")
			if end < 0 {
				break
			}

			i += end + len("--->")

			continue
		}

		closing := strings.HasPrefix(text[i:], "</")

		j := i + 1
		if closing {
			j++
		}

		k := j
		for k < len(text) && (text[k] >= 'a' && text[k] <= 'z' || text[k] >= 'A' && text[k] <= 'Z' || text[k] >= '0' && text[k] <= '9') {
			k++
		}

		name := strings.ToLower(text[j:k])
		tagEnd := strings.IndexByte(text[k:], '>')

		if tagEnd < 0 {
			break
		}

		selfClosed := text[k+tagEnd-1] == '/'

		if name == "cfscript" && !closing {
			scriptEnd := parser.IndexFoldFrom(text, "</cfscript", k)
			if scriptEnd < 0 {
				scriptEnd = len(text)
			}

			events = append(events, braceEvents(text[k+tagEnd+1:scriptEnd], k+tagEnd+1)...)
			i = scriptEnd

			continue
		}

		if ev, ok := tagBlockEvent(name, closing, selfClosed, i); ok {
			events = append(events, ev)
		}

		i = k + tagEnd + 1
	}

	return events
}

// tagBlocks reports whether the scope starting at line scopeStart is written
// in tags: a template, or a <cffunction> body.
func tagBlocks(pr *parser.ParseResult, scopeStart int) bool {
	switch strings.ToLower(filepath.Ext(pr.URI.Path())) {
	case ".cfm", ".cfml":
		return true
	}

	from, ok := lineOffset(pr.Content, scopeStart)
	if !ok {
		return false
	}

	return parser.IndexFoldFrom(pr.Content[from:min(len(pr.Content), from+200)], "<cffunction", 0) >= 0
}

// tagBlockEvent is the block event a tag makes, if it makes one.
func tagBlockEvent(name string, closing, selfClosed bool, offset int) (blockEvent, bool) {
	ifKinds := []string{"cfif", "cfelseif", "cfelse"}

	switch {
	case closing:
		switch name {
		case "cfif":
			return blockEvent{offset: offset, op: opClose, ends: ifKinds}, true
		case "cfloop", "cfcase", "cfdefaultcase":
			return blockEvent{offset: offset, op: opClose, ends: []string{name}}, true
		case "cfcatch":
			return blockEvent{offset: offset, op: opClose, ends: []string{"cfcatch"}}, true
		case "cftry":
			return blockEvent{offset: offset, op: opClose, ends: []string{"cftry", "cfcatch"}}, true
		}
	case selfClosed:
	case name == "cfif" || name == "cfloop" || name == "cftry" || name == "cfcase" || name == "cfdefaultcase":
		return blockEvent{offset: offset, op: opOpen, kind: name}, true
	case name == "cfelseif" || name == "cfelse":
		return blockEvent{offset: offset, op: opBranch, kind: name, ends: ifKinds}, true
	case name == "cfcatch":
		return blockEvent{offset: offset, op: opBranch, kind: name, ends: []string{"cftry", "cfcatch"}}, true
	}

	return blockEvent{}, false
}

// braceEvents reads CFScript braces in text, which begins base bytes into the
// text the offsets are relative to.
func braceEvents(text string, base int) []blockEvent {
	var events []blockEvent

	scanner := parser.NewSourceScanner(text)

	for {
		switch tok := scanner.NextSkipComments(); tok.Kind {
		case parser.TokEOF:
			return events
		case parser.TokLBrace:
			events = append(events, blockEvent{offset: base + tok.Offset, op: opOpen, kind: "{"})
		case parser.TokRBrace:
			events = append(events, blockEvent{offset: base + tok.Offset, op: opClose})
		default:
		}
	}
}

// refOnLine is the last of refs made by the assignment on line.
func refOnLine(refs []*parser.ComponentRef, line int) *parser.ComponentRef {
	var found *parser.ComponentRef

	for _, ref := range refs {
		if int(ref.Line) == line {
			found = ref
		}
	}

	return found
}
