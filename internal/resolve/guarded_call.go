package resolve

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// A call the code checks for before making it is not a missing method:
//
//	if ( structKeyExists( variables.config, "onShutdown" ) ) {
//		variables.config.onShutdown( this );
//	}
//
// is LogBox calling a convention its config may or may not follow. The guard
// is structKeyExists( receiver, "method" ), receiver.keyExists( "method" ) or
// isDefined( "receiver.method" ), not negated, in the function making the
// call, and the call must sit in what the guard controls: the block of the if
// it opens (a brace group, or a <cfif> up to its close), or the rest of the
// statement when there is no block.

// The guards are the parse's calls (guardCalls): a check in a comment, in
// markup or in a string that is no expression is none, and where a call
// starts and ends is where the parse read it.

// existsGuardNames, instanceGuardNames and nullGuardNames are the calls the
// guards are written with; the plain parse records them (readerCallNames).
var (
	existsGuardNames   = []string{"structKeyExists", "keyExists", "isDefined"}
	instanceGuardNames = []string{"isInstanceOf", "checkForInstanceOf"}
	nullGuardNames     = []string{"isNull", "structKeyExists"}
)

// guardCalls are the calls of names pr's file makes that start at or after
// from and end by to, in the order they are written: pr's own record when
// it holds these names' calls, else the file's plain parse's.
func (r *Resolver) guardCalls(pr *parser.ParseResult, from, to int, names ...string) []parsedCall {
	if !pr.RecordsCallsOf(names...) {
		if pr.URI.IsFile() {
			pr = r.plainParseOf(pr.URI.Path(), pr.Content)
		} else {
			pr = parser.ParseWithOptions(pr.URI, pr.Content, &parser.ParseOptions{FindCalls: readerCallNames})
		}
	}

	var out []parsedCall

	for _, c := range parsedCalls(pr, names...) {
		if int(c.site.StartByte) >= from && int(c.site.EndByte) <= to {
			out = append(out, c)
		}
	}

	return out
}

// guardedByExistsCheck reports whether the call to funcName on variable at
// line (0-based) is made only after its containing code checked that the
// method exists.
func (r *Resolver) guardedByExistsCheck(pr *parser.ParseResult, variable, funcName string, line int) bool {
	if funcName == "" {
		return false
	}

	text, start, ok := guardText(pr, variable, funcName, line)
	if !ok {
		return false
	}

	// Every form of the guard names the method.
	if !mentions(text, funcName) {
		return false
	}

	end := start + len(text)

	for _, m := range existsGuards(pr.Content, r.guardCalls(pr, start, end, existsGuardNames...), variable, funcName) {
		if guards(text, m[0]-start, m[1]-start) {
			return true
		}
	}

	return false
}

// guardText is the code of the function holding line, from its start to the
// call to variable.funcName on that line, and the offset it starts at.
func guardText(pr *parser.ParseResult, variable, funcName string, line int) (string, int, bool) {
	if pr == nil || variable == "" {
		return "", 0, false
	}

	from := 0
	if scope := pr.FuncScopeAt(line); scope.Start != -1 {
		from = scope.Start
	}

	start, ok := lineOffset(pr.Content, from)
	if !ok {
		return "", 0, false
	}

	at, ok := lineOffset(pr.Content, line)
	if !ok {
		return "", 0, false
	}

	if i := parser.IndexFold(lineOfContent(pr.Content, line), variable+"."+funcName); i >= 0 {
		at += i
	}

	return pr.Content[start:at], start, true
}

// guards reports whether the check at text[from:to] guards the end of text:
// not negated, and what it opens still open there. A qualifier on the
// check's function (variables.utility.checkForInstanceOf) is stepped over.
func guards(text string, from, to int) bool {
	head := strings.TrimRight(text[:from], "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_$.")
	if !strings.HasSuffix(text[:from], ".") {
		head = text[:from]
	}

	return !negatedGuard(head) && controls(head, text[to:])
}

// negatedGuard reports whether the text before a guard ends in a negation.
func negatedGuard(before string) bool {
	before = strings.TrimRight(before, " \t")
	if strings.HasSuffix(before, "!") {
		return true
	}

	l := strings.ToLower(before)

	return strings.HasSuffix(l, " not") || strings.HasSuffix(l, "(not")
}

// controls reports whether the code the guard opens is still open at the end
// of after, the text from the guard to the call.
func controls(before, after string) bool {
	if isTagGuard(before) {
		depth := 1

		for l := strings.ToLower(after); ; {
			o, c := strings.Index(l, "<cfif"), strings.Index(l, "</cfif")
			switch {
			case c < 0 && o < 0:
				return depth > 0
			case c < 0 || (o >= 0 && o < c):
				depth++
				l = l[o+5:]
			default:
				depth--
				if depth == 0 {
					return false
				}

				l = l[c+6:]
			}
		}
	}

	// The braces and semicolons are the script scanner's: one in a string or
	// a comment between the guard and the call ends nothing. A byte count
	// took `log( "done; }" )` inside the guarded block for its end.
	depth, opened := 0, false
	scanner := parser.NewScriptScanner(after)

	for {
		switch tok := scanner.NextSkipComments(); tok.Kind {
		case parser.TokEOF:
			return opened && depth > 0 || !opened && strings.Count(after, "\n") <= 1
		case parser.TokLBrace:
			depth++
			opened = true
		case parser.TokRBrace:
			depth--
			if depth <= 0 {
				return false
			}
		case parser.TokSemicolon:
			if !opened {
				return false
			}
		default:
		}
	}
}

// isTagGuard reports whether the guard is in a <cfif> or <cfelseif>.
func isTagGuard(before string) bool {
	l := strings.ToLower(before)
	tag, script := strings.LastIndex(l, "<cf"), strings.LastIndexAny(l, ">;{}")

	return tag >= 0 && tag > script
}

func lineOffset(content string, n int) (int, bool) {
	return lineStarts.of(content, parser.NewLineTable).Start(n)
}

// lineStarts remembers the line table of the texts lineOffset was asked about
// most recently. lineOffset counted newlines from the top on every call, and
// it is asked once or more per call site a file holds, so a file was walked
// again for each: 0.7s of tassweb's code map. A text is recognised by its
// bytes, which for the same file's content is a pointer comparison.
var lineStarts = &textCache[parser.LineTable]{size: func(s parser.LineTable) int { return 4 * len(s) }}

const (
	textCacheSlots = 64
	// textCacheBytes bounds the texts a textCache keeps alive, and what it
	// derived from them: in the editor's long-lived process each is a version
	// of a document, and 64 versions of a large component were hundreds of
	// megabytes held for nothing. It is sized for a batch scan, whose workers
	// each ask about a file of their own at once: a budget the files in flight
	// outgrow evicts one worker's file for another's, and every call then
	// rebuilds its file's table.
	textCacheBytes = 128 << 20
)

// LimitTextCaches bounds each of the process's text caches to bytes, for a
// long-lived process (the editor's, the MCP server's) whose entries are mostly
// superseded versions of a document; a batch scan keeps the default, sized
// for its workers. It applies to what is cached from then on, except while a
// batch runs (BatchTextCaches), and returns a function putting back the
// budgets it replaced.
func LimitTextCaches(bytes int) (restore func()) {
	prevLines, prevDecls, prevCode := lineStarts.setBudget(bytes), declaredNames.setBudget(bytes), codeMemo.setBudget(bytes)
	prevIncludes := includePaths.setBudget(bytes)
	prevTokens := longLivedTokenBytes.Swap(int64(bytes))

	// The parser's shared token lists are bounded with the rest, unless a
	// batch is running and has them at its own budget.
	if batchesRunning.Load() == 0 {
		parser.SetSharedTokenLimit(bytes)
	}

	return func() {
		lineStarts.setBudget(prevLines)
		declaredNames.setBudget(prevDecls)
		codeMemo.setBudget(prevCode)
		includePaths.setBudget(prevIncludes)
		longLivedTokenBytes.Store(prevTokens)

		if batchesRunning.Load() == 0 {
			parser.SetSharedTokenLimit(tokenLimitNow())
		}
	}
}

// longLivedTokenBytes is the bound LimitTextCaches set on the parser's shared
// token lists, or zero when none is set.
var longLivedTokenBytes atomic.Int64

// tokenLimitNow is the bound the shared token lists are held to outside a batch.
func tokenLimitNow() int {
	if n := int(longLivedTokenBytes.Load()); n > 0 {
		return n
	}

	return parser.DefaultSharedTokenBytes
}

// BatchTextCaches lifts LimitTextCaches' budget to the batch default until
// the returned function is called. A scan or map run inside a long-lived
// process has a worker per core asking about a file of its own at once, and
// the long-lived budget would evict one worker's file for another's on every
// call.
func BatchTextCaches() (done func()) {
	if batchesRunning.Add(1) == 1 {
		parser.SetSharedTokenLimit(parser.DefaultSharedTokenBytes)
	}

	return func() {
		// The last batch out gives back what it was allowed beyond the limit.
		if batchesRunning.Add(-1) == 0 {
			parser.SetSharedTokenLimit(tokenLimitNow())
			lineStarts.trim()
			declaredNames.trim()
			codeMemo.trim()
			includePaths.trim()
		}
	}
}

// batchesRunning counts the BatchTextCaches calls not yet done.
var batchesRunning atomic.Int32

// effectiveBudget is the budget an insert is held to now. c.mu is held.
func (c *textCache[T]) effectiveBudget() int {
	if c.budget <= 0 || batchesRunning.Load() > 0 {
		return textCacheBytes
	}

	return c.budget
}

// trim drops the oldest texts until the cache is within its budget.
func (c *textCache[T]) trim() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.evictFrom(c.next, -1, c.effectiveBudget())
}

// evictFrom drops texts oldest first, from slot k, until held fits budget,
// never dropping slot keep. c.mu is held.
func (c *textCache[T]) evictFrom(k, keep, budget int) {
	var zero T

	for range textCacheSlots {
		if c.held <= budget {
			return
		}

		if k != keep && c.slots[k].set {
			c.held -= c.slots[k].bytes
			c.slots[k].content, c.slots[k].value, c.slots[k].bytes, c.slots[k].set = "", zero, 0, false
		}

		k = (k + 1) % textCacheSlots
	}
}

// budgetNow is the budget an insert would be held to, for tests.
func (c *textCache[T]) budgetNow() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.effectiveBudget()
}

func (c *textCache[T]) setBudget(bytes int) (prev int) {
	c.mu.Lock()
	prev, c.budget = c.budget, bytes
	c.mu.Unlock()

	return prev
}

// textCache remembers a value derived from each of the texts asked about most
// recently, keeping at most textCacheSlots texts and textCacheBytes of them.
type textCache[T any] struct {
	size   func(T) int // the bytes a value holds beyond its text
	budget int         // textCacheBytes when zero; see LimitTextCaches
	mu     sync.Mutex
	next   int
	held   int
	slots  [textCacheSlots]struct {
		content string
		value   T
		bytes   int
		set     bool
	}
}

// of is build(content), from the cache when content was asked about recently.
func (c *textCache[T]) of(content string, build func(string) T) T {
	c.mu.Lock()
	for i := range c.slots {
		if s := &c.slots[i]; s.set && len(s.content) == len(content) && s.content == content {
			v := s.value
			c.mu.Unlock()

			return v
		}
	}
	c.mu.Unlock()

	v := build(content)

	n := len(content)
	if c.size != nil {
		n += c.size(v)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	budget := c.effectiveBudget()

	// Another goroutine may have built it meanwhile: one slot per text.
	for i := range c.slots {
		if s := &c.slots[i]; s.set && len(s.content) == len(content) && s.content == content {
			return s.value
		}
	}

	at := c.next
	c.held += n - c.slots[at].bytes

	c.slots[at].content, c.slots[at].value, c.slots[at].bytes, c.slots[at].set = content, v, n, true
	c.next = (at + 1) % textCacheSlots

	// Over budget: drop the oldest texts until it fits. The newest is kept
	// even when it alone is over: a file that large is the one whose table is
	// most expensive to rebuild on every call.
	c.evictFrom(c.next, at, budget)

	return v
}

// guardedInstanceOf is the components a type check guarding the call names
// for variable: inside `<cfif checkForInstanceOf( arguments.event,
// "mura.MuraScope" )>` Mura's pluginManager calls arguments.event.event(),
// which the MuraScope declares and the event it is otherwise handed does not.
// isInstanceOf() and Mura's utility.checkForInstanceOf() are read, under the
// rules guardedByExistsCheck applies.
func (r *Resolver) guardedInstanceOf(pr *parser.ParseResult, variable, funcName string, line int) []string {
	text, start, ok := guardText(pr, variable, funcName, line)
	if !ok {
		return nil
	}

	if !mentions(text, "InstanceOf(") {
		return nil
	}

	var comps []string

	end := start + len(text)

	for _, g := range instanceGuards(r.guardCalls(pr, start, end, instanceGuardNames...), variable) {
		if guards(text, g.start-start, g.end-start) {
			comps = append(comps, g.comp)
		}
	}

	return comps
}

// existsGuards are the [start, end) offsets of the checks among calls that
// variable has a member funcName: structKeyExists( variable, "funcName" ),
// variable.keyExists( "funcName" ) and isDefined( "variable.funcName" ). A
// check's start is its function's name, or the variable's for keyExists.
// content is the text the calls were read from.
func existsGuards(content string, calls []parsedCall, variable, funcName string) [][2]int {
	var out [][2]int

	for _, c := range calls {
		pieces := c.pieces()

		switch strings.ToLower(c.name().Value) {
		case "structkeyexists":
			if len(pieces) == 2 && spells(pieces[0], variable) && soleStringIs(pieces[1], funcName) {
				out = append(out, [2]int{c.name().Offset, c.end()})
			}
		case "keyexists":
			if len(pieces) != 1 || !soleStringIs(pieces[0], funcName) {
				continue
			}

			if start, ok := receiverWrittenBefore(content, c.name().Offset, variable); ok {
				out = append(out, [2]int{start, c.end()})
			}
		default:
			if len(pieces) == 1 && soleStringIs(pieces[0], variable+"."+funcName) {
				out = append(out, [2]int{c.name().Offset, c.end()})
			}
		}
	}

	return out
}

// receiverWrittenBefore is where variable starts when content holds it, and
// a dot, just before the called name at offset at: `variable.keyExists(`,
// with space allowed around the dot, ignoring case. The variable is a whole
// name, not the tail of a longer one.
func receiverWrittenBefore(content string, at int, variable string) (int, bool) {
	before := strings.TrimRight(content[:at], " \t\r\n")
	if !strings.HasSuffix(before, ".") {
		return 0, false
	}

	before = strings.TrimRight(before[:len(before)-1], " \t\r\n")

	start := len(before) - len(variable)
	if variable == "" || start < 0 || !strings.EqualFold(before[start:], variable) {
		return 0, false
	}

	if start > 0 && (lex.IsWord(before[start-1]) || before[start-1] == '.') {
		return 0, false
	}

	return start, true
}

// instanceGuard is one isInstanceOf or checkForInstanceOf check of a
// variable: where it is, and the component it names.
type instanceGuard struct {
	start, end int
	comp       string
}

// instanceGuards are the checks among calls that variable is an instance of
// a component: isInstanceOf( variable, "a.b.C" ) or Mura's
// checkForInstanceOf( variable, "a.b.C" ), which may be called on a
// qualifier (variables.utility.checkForInstanceOf).
func instanceGuards(calls []parsedCall, variable string) []instanceGuard {
	var out []instanceGuard

	for _, c := range calls {
		pieces := c.pieces()
		if len(pieces) != 2 || !spells(pieces[0], variable) {
			continue
		}

		if comp, ok := soleString(pieces[1]); ok && comp != "" && strings.IndexFunc(comp, func(r rune) bool { return r != '.' && !isWordRune(r) }) < 0 {
			out = append(out, instanceGuard{start: c.name().Offset, end: c.end(), comp: comp})
		}
	}

	return out
}
