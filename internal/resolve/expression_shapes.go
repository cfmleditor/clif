// Fixed shapes of one expression or value the resolver asks about: a keyed
// read, a createObject call, a null fallback, an init's argument copy and the
// like. Each was a regular expression, usually anchored at both ends, and
// each is read here byte by byte with a cursor, as the parser's text shapes
// are (internal/parser/text_shapes.go); the expression it replaces is kept in
// expression_shapes_test.go, which holds the two to the same matches and
// captures over generated texts and under a fuzz test.
//
// A reader that stands for FindStringSubmatch answers as it did: nil for no
// match, else the whole match and then each group, "" for one that did not
// take part. Case is folded in ASCII alone, as lex.HasPrefixFold does
// everywhere else: a regular expression's (?i) also folds the Kelvin sign
// into k and the long s into s, which no CFML source spells.

package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

// exprCursor reads s from i.
type exprCursor struct {
	s string
	i int
}

// spaces skips \s*.
func (c *exprCursor) spaces() { c.i = lex.SkipRegexpSpace(c.s, c.i) }

// spaces1 skips \s+, and reports whether there was any.
func (c *exprCursor) spaces1() bool {
	from := c.i
	c.spaces()

	return c.i > from
}

// fold steps over lit, matched without case.
func (c *exprCursor) fold(lit string) bool {
	if !lex.HasPrefixFold(c.s[c.i:], lit) {
		return false
	}

	c.i += len(lit)

	return true
}

// lit steps over lit, matched with its case.
func (c *exprCursor) lit(lit string) bool {
	if !strings.HasPrefix(c.s[c.i:], lit) {
		return false
	}

	c.i += len(lit)

	return true
}

// byte steps over b.
func (c *exprCursor) byte(b byte) bool {
	if c.i < len(c.s) && c.s[c.i] == b {
		c.i++

		return true
	}

	return false
}

// quote steps over ["'].
func (c *exprCursor) quote() bool {
	if c.i < len(c.s) && lex.IsQuote(c.s[c.i]) {
		c.i++

		return true
	}

	return false
}

// run steps over the bytes in reports true for, and is them.
func (c *exprCursor) run(in func(byte) bool) string {
	from := c.i
	for c.i < len(c.s) && in(c.s[c.i]) {
		c.i++
	}

	return c.s[from:c.i]
}

// name steps over [A-Za-z_]\w* and is it, or "".
func (c *exprCursor) name() string {
	if c.i >= len(c.s) || !lex.IsWordStart(c.s[c.i]) {
		return ""
	}

	return c.run(lex.IsWord)
}

// end reports whether the cursor is at the end: $.
func (c *exprCursor) end() bool { return c.i == len(c.s) }

// keyedTail steps over \s*\[[^\]]*\]$.
func (c *exprCursor) keyedTail() bool {
	c.spaces()

	if !c.byte('[') {
		return false
	}

	c.run(func(b byte) bool { return b != ']' })

	return c.byte(']') && c.end()
}

// isWordRun reports whether s is \w+.
func isWordRun(s string) bool {
	if s == "" {
		return false
	}

	for i := range len(s) {
		if !lex.IsWord(s[i]) {
			return false
		}
	}

	return true
}

// isWordOrDot is [\w.].
func isWordOrDot(b byte) bool { return lex.IsWord(b) || b == '.' }

// isNotQuote is [^"'].
func isNotQuote(b byte) bool { return !lex.IsQuote(b) }

// scopeWords are the scopes loopPath and indexedRead name, in the order
// their expressions tried them.
var scopeWords = [...]string{"local", "variables", "this", "arguments"}

// structField is ^(?:local\.)?(\w+)\.(\w+)$: a field of a local's struct.
func structField(s string) []string {
	read := func(from int) []string {
		c := exprCursor{s: s, i: from}

		obj := c.run(lex.IsWord)
		if obj == "" || !c.byte('.') {
			return nil
		}

		if field := c.run(lex.IsWord); field != "" && c.end() {
			return []string{s, obj, field}
		}

		return nil
	}

	if strings.HasPrefix(s, "local.") {
		if m := read(len("local.")); m != nil {
			return m
		}
	}

	return read(0)
}

// keyedField is (?i)^(?:local\s*\.\s*)?(\w+)\s*\[\s*variables\s*\.\s*(\w+)\s*\]$:
// a local's struct read under a key the variables scope holds.
func keyedField(s string) []string {
	read := func(c exprCursor) []string {
		obj := c.run(lex.IsWord)
		if obj == "" {
			return nil
		}

		c.spaces()

		if !c.byte('[') {
			return nil
		}

		c.spaces()

		if !c.fold("variables") {
			return nil
		}

		c.spaces()

		if !c.byte('.') {
			return nil
		}

		c.spaces()

		key := c.run(lex.IsWord)
		if key == "" {
			return nil
		}

		c.spaces()

		if !c.byte(']') || !c.end() {
			return nil
		}

		return []string{s, obj, key}
	}

	c := exprCursor{s: s}
	if c.fold("local") {
		c.spaces()

		if c.byte('.') {
			c.spaces()

			if m := read(c); m != nil {
				return m
			}
		}
	}

	return read(exprCursor{s: s})
}

// keyedReceiver is ^([A-Za-z_]\w*)\[[^[\]]*\]$: a name and one key.
func keyedReceiver(s string) []string {
	c := exprCursor{s: s}

	name := c.name()
	if name == "" || !c.byte('[') {
		return nil
	}

	c.run(func(b byte) bool { return b != '[' && b != ']' })

	if !c.byte(']') || !c.end() {
		return nil
	}

	return []string{s, name}
}

// isNullCast is (?i)^(?:javacast\s*\(\s*["']null["']|nullvalue\s*\(\s*\)$|null$).
// A null a conditional falls back to, however it is written: `javacast(
// "null", … )` (`prc.x = rc.x ?: javacast( "null", "" )`), Lucee's
// `nullValue()`, or the `null` literal.
func isNullCast(s string) bool {
	if len(s) == len("null") && lex.HasPrefixFold(s, "null") {
		return true
	}

	c := exprCursor{s: s}
	if c.fold("javacast") {
		c.spaces()

		if !c.byte('(') {
			return false
		}

		c.spaces()

		return c.quote() && c.fold("null") && c.quote()
	}

	return c.fold("nullvalue") && emptyCallAtEnd(&c)
}

// emptyCallAtEnd steps over \s*\(\s*\)$.
func emptyCallAtEnd(c *exprCursor) bool {
	c.spaces()

	if !c.byte('(') {
		return false
	}

	c.spaces()

	return c.byte(')') && c.end()
}

// concatPickup is
// (?i)^variables\.?\[\s*([A-Za-z_]\w*)\s*&\s*["']([^"']*)["']\s*\]$.
func concatPickup(s string) []string {
	c := exprCursor{s: s}
	if !c.fold("variables") {
		return nil
	}

	c.byte('.')

	if !c.byte('[') {
		return nil
	}

	c.spaces()

	name := c.name()
	if name == "" {
		return nil
	}

	c.spaces()

	if !c.byte('&') {
		return nil
	}

	c.spaces()

	if !c.quote() {
		return nil
	}

	literal := c.run(isNotQuote)

	if !c.quote() {
		return nil
	}

	c.spaces()

	if !c.byte(']') || !c.end() {
		return nil
	}

	return []string{s, name, literal}
}

// nullFallback is
// (?is)^(.+?)\s*\?:\s*(?:nullValue\s*\(\s*\)|javaCast\s*\(\s*["']null["']\s*,\s*["']{2}\s*\))$:
// an expression with an Elvis fallback to null, and the expression. The
// lazy group ends at the first place the rest reads; that place is a space
// or a ?, an ASCII byte, which no rune runs across.
func nullFallback(s string) []string {
	for k := 1; k < len(s); k++ {
		if (s[k] == '?' || lex.IsRegexpSpace(s[k])) && nullFallbackTail(s[k:]) {
			return []string{s, s[:k]}
		}
	}

	return nil
}

// nullFallbackTail is the rest of nullFallback after its group.
func nullFallbackTail(s string) bool {
	c := exprCursor{s: s}
	c.spaces()

	if !c.byte('?') || !c.byte(':') {
		return false
	}

	c.spaces()

	if c.fold("nullValue") {
		return emptyCallAtEnd(&c)
	}

	if !c.fold("javaCast") {
		return false
	}

	c.spaces()

	if !c.byte('(') {
		return false
	}

	c.spaces()

	if !c.quote() || !c.fold("null") || !c.quote() {
		return false
	}

	c.spaces()

	if !c.byte(',') {
		return false
	}

	c.spaces()

	// ["']{2}: either quote may open the empty string and either close it.
	for range 2 {
		if !c.quote() {
			return false
		}
	}

	c.spaces()

	return c.byte(')') && c.end()
}

// argumentCopyEnd steps over arguments\.(\w+)\s*;?\s*$, or with
// closeParen arguments\.(\w+)\s*\)\s*;?\s*$, and is the argument's name,
// or "".
func argumentCopyEnd(c *exprCursor, closeParen bool) string {
	if !c.fold("arguments.") {
		return ""
	}

	arg := c.run(lex.IsWord)
	if arg == "" {
		return ""
	}

	c.spaces()

	if closeParen {
		if !c.byte(')') {
			return ""
		}

		c.spaces()
	}

	c.byte(';')
	c.spaces()

	if !c.end() {
		return ""
	}

	return arg
}

// initArgAssign is
// (?i)^\s*(?:variables\.)?(\w+)\s*=\s*arguments\.(\w+)\s*;?\s*$: a line
// copying an argument to a variable.
func initArgAssign(s string) []string {
	read := func(c exprCursor) []string {
		name := c.run(lex.IsWord)
		if name == "" {
			return nil
		}

		c.spaces()

		if !c.byte('=') {
			return nil
		}

		c.spaces()

		if arg := argumentCopyEnd(&c, false); arg != "" {
			return []string{s, name, arg}
		}

		return nil
	}

	c := exprCursor{s: s}
	c.spaces()

	if scoped := c; scoped.fold("variables.") {
		if m := read(scoped); m != nil {
			return m
		}
	}

	return read(c)
}

// initArgSet is
// (?i)^\s*(?:this\.|variables\.)?set(\w+)\(\s*arguments\.(\w+)\s*\)\s*;?\s*$:
// a line handing an argument to a setter.
func initArgSet(s string) []string {
	read := func(c exprCursor) []string {
		if !c.fold("set") {
			return nil
		}

		name := c.run(lex.IsWord)
		if name == "" || !c.byte('(') {
			return nil
		}

		c.spaces()

		if arg := argumentCopyEnd(&c, true); arg != "" {
			return []string{s, name, arg}
		}

		return nil
	}

	c := exprCursor{s: s}
	c.spaces()

	for _, scope := range []string{"this.", "variables."} {
		if scoped := c; scoped.fold(scope) {
			if m := read(scoped); m != nil {
				return m
			}
		}
	}

	return read(c)
}

// createObjectOf steps over createObject\s*\(\s*["']<kind>["'], matched
// without case.
func createObjectOf(c *exprCursor, kind string) bool {
	if !c.fold("createObject") {
		return false
	}

	c.spaces()

	if !c.byte('(') {
		return false
	}

	c.spaces()

	return c.quote() && c.fold(kind) && c.quote()
}

// isJavaObject is (?is)^(?:createObject\s*\(\s*["']java["']|new\s+java:):
// an expression that makes a Java object: createObject with the java type,
// or Lucee's new java:, whatever is chained on it.
func isJavaObject(s string) bool {
	if c := (exprCursor{s: s}); createObjectOf(&c, "java") {
		return true
	}

	c := exprCursor{s: s}

	return c.fold("new") && c.spaces1() && c.fold("java:")
}

// computedComponent is
// (?is)^createObject\s*\(\s*["']component["']\s*,\s*["'][^"']*#[^"']*\.(\w+)["']\s*\)$:
// createObject of a component whose path is computed but whose last
// segment, the file's name, is literal, and that segment.
func computedComponent(s string) []string {
	c := exprCursor{s: s}
	if !createObjectOf(&c, "component") {
		return nil
	}

	c.spaces()

	if !c.byte(',') {
		return nil
	}

	c.spaces()

	if !c.quote() {
		return nil
	}

	path := c.run(isNotQuote)

	if !c.quote() {
		return nil
	}

	c.spaces()

	if !c.byte(')') || !c.end() {
		return nil
	}

	dot := strings.LastIndexByte(path, '.')
	if dot < 0 || !isWordRun(path[dot+1:]) || strings.IndexByte(path[:dot], '#') < 0 {
		return nil
	}

	return []string{s, path[dot+1:]}
}

// isMuraEventSource is
// (?i)(?:^|\.)(?:getEvent|event)\(\s*\)$|mura\.(?:servlet)?event["']: an
// expression that hands back Mura's event: a renderer's getEvent(), a
// MuraScope's event(), or a new one.
func isMuraEventSource(s string) bool {
	if j := len(s) - 1; j >= 0 && s[j] == ')' {
		j--
		for j >= 0 && lex.IsRegexpSpace(s[j]) {
			j--
		}

		if j >= 0 && s[j] == '(' {
			head := s[:j]

			for _, name := range []string{"getEvent", "event"} {
				if p := len(head) - len(name); p >= 0 && lex.HasPrefixFold(head[p:], name) && (p == 0 || head[p-1] == '.') {
					return true
				}
			}
		}
	}

	for from := 0; ; {
		i := parser.IndexFoldFrom(s, "mura.", from)
		if i < 0 {
			return false
		}

		c := exprCursor{s: s, i: i + len("mura.")}
		c.fold("servlet")

		if c.fold("event") && c.quote() {
			return true
		}

		from = i + 1
	}
}

// returnFormatAt calls match with a cursor past each returnFormat\s*= in s
// that starts a word, matched without case, until one reports true.
func returnFormatAt(s string, match func(c *exprCursor) bool) bool {
	for from := 0; ; {
		i := parser.IndexFoldFrom(s, "returnFormat", from)
		if i < 0 {
			return false
		}

		from = i + 1

		if i > 0 && lex.IsWord(s[i-1]) {
			continue
		}

		c := exprCursor{s: s, i: i + len("returnFormat")}
		c.spaces()

		if c.byte('=') && match(&c) {
			return true
		}
	}
}

// namesReturnFormat is (?i)\breturnFormat\s*=: the arguments name the format.
func namesReturnFormat(args string) bool {
	return returnFormatAt(args, func(*exprCursor) bool { return true })
}

// namesSelfFormat is (?i)\breturnFormat\s*=\s*["']self["']: the arguments
// name the format "self".
func namesSelfFormat(args string) bool {
	return returnFormatAt(args, func(c *exprCursor) bool {
		c.spaces()

		return c.quote() && c.fold("self") && c.quote()
	})
}

// positionalFirst is ^\s*[^\s=]+\s*(?:,|$): the first argument is positional.
// The run may stop at any comma inside it, so a comma after its first byte
// is a match.
func positionalFirst(args string) bool {
	c := exprCursor{s: args}
	c.spaces()

	arg := c.run(func(b byte) bool { return b != '=' && !lex.IsRegexpSpace(b) })
	if arg == "" {
		return false
	}

	if strings.IndexByte(arg[1:], ',') >= 0 {
		return true
	}

	c.spaces()

	return c.end() || c.byte(',')
}

// loopPath is ^(?:(local|variables|this|arguments)\.)?(\w+)$: a plain or
// scoped name.
func loopPath(s string) []string {
	for _, scope := range scopeWords {
		if name, ok := strings.CutPrefix(s, scope+"."); ok && isWordRun(name) {
			return []string{s, scope, name}
		}
	}

	if isWordRun(s) {
		return []string{s, "", s}
	}

	return nil
}

// indexedRead is
// ^((?:local|variables|this|arguments)\.[\w.]+|[A-Za-z_]\w*)\s*\[[^\]]*\]$:
// a keyed element read of a plain or scoped name, and the name.
func indexedRead(s string) []string {
	for _, scope := range scopeWords {
		c := exprCursor{s: s}
		if c.lit(scope+".") && c.run(isWordOrDot) != "" {
			name := s[:c.i]
			if c.keyedTail() {
				return []string{s, name}
			}
		}
	}

	c := exprCursor{s: s}
	if name := c.name(); name != "" && c.keyedTail() {
		return []string{s, name}
	}

	return nil
}

// isExactKeyedRead is
// ^(?:[A-Za-z_]\w*|(?:local|variables|this|arguments)\.[\w.]+)\s*\[[^\]]*\]$,
// indexedRead's alternatives the other way round: a whole keyed element
// read, the collection under one key and no chain after it.
func isExactKeyedRead(s string) bool { return indexedRead(s) != nil }

// chainKeyedRead is
// ^([A-Za-z_]\w*(?:\.\w+)*?(?:\.\w+\s*\([^()]*\))+)\s*\[[^\]]*\]$: a keyed
// read of what a call chain returns, and the chain. Members come first and
// calls after, and a call's ) followed by anything but a . or the key ends
// the chain, so there is one way to read it.
func chainKeyedRead(s string) []string {
	c := exprCursor{s: s}
	if c.name() == "" {
		return nil
	}

	calls := 0

	for c.byte('.') {
		if c.run(lex.IsWord) == "" {
			return nil
		}

		if c.i < len(s) && s[c.i] == '.' {
			if calls > 0 {
				return nil
			}

			continue
		}

		c.spaces()

		if !c.byte('(') {
			return nil
		}

		c.run(func(b byte) bool { return b != '(' && b != ')' })

		if !c.byte(')') {
			return nil
		}

		calls++

		if chain := s[:c.i]; c.i < len(s) && s[c.i] != '.' {
			if c.keyedTail() {
				return []string{s, chain}
			}

			return nil
		}
	}

	return nil
}

// isIdent is ^[A-Za-z_]\w*$.
func isIdent(s string) bool {
	c := exprCursor{s: s}

	return c.name() != "" && c.end()
}

// isDottedName is ^[A-Za-z_$][\w$]*(?:\.[A-Za-z_$][\w$]*)+$: a dotted name
// of two or more segments, no call or index.
func isDottedName(s string) bool {
	segments := strings.Split(s, ".")
	if len(segments) < 2 {
		return false
	}

	for _, seg := range segments {
		if seg == "" || !lex.IsWordStart(seg[0]) && seg[0] != '$' {
			return false
		}

		for i := 1; i < len(seg); i++ {
			if !lex.IsIdent(seg[i]) {
				return false
			}
		}
	}

	return true
}

// listingRead is ^((?:variables\.)?[A-Za-z_]\w*)\s*\[: the struct a
// registry listing's element read reads: the struct the listing merges
// into, under a key. Group 1 is the listing's name.
func listingRead(s string) []string {
	read := func(c exprCursor) []string {
		if c.name() == "" {
			return nil
		}

		name := s[:c.i]
		c.spaces()

		if !c.byte('[') {
			return nil
		}

		return []string{s[:c.i], name}
	}

	if c := (exprCursor{s: s}); c.lit("variables.") {
		if m := read(c); m != nil {
			return m
		}
	}

	return read(exprCursor{s: s})
}

// duplicateCall is
// (?i)^duplicate\s*\(\s*([A-Za-z_][\w.]*)\s*(?:,\s*(?:true|false)\s*)?\)$:
// a whole duplicate( name ) of a plain or scoped name, with the optional
// deep-copy flag, and the name: the copy of a component is an instance of it.
func duplicateCall(s string) []string {
	c := exprCursor{s: s}
	if !c.fold("duplicate") {
		return nil
	}

	c.spaces()

	if !c.byte('(') {
		return nil
	}

	c.spaces()

	if c.i >= len(s) || !lex.IsWordStart(s[c.i]) {
		return nil
	}

	name := c.run(isWordOrDot)
	c.spaces()

	if c.byte(',') {
		c.spaces()

		if !c.fold("true") && !c.fold("false") {
			return nil
		}

		c.spaces()
	}

	if !c.byte(')') || !c.end() {
		return nil
	}

	return []string{s, name}
}

// duplicatePlainCall is
// (?i)^duplicate\s*\(\s*(?:var\s+)?(?:variables\.)?([A-Za-z_]\w*)\s*\): a
// duplicate( name ) at the start of s, and the name.
func duplicatePlainCall(s string) []string {
	c := exprCursor{s: s}
	if !c.fold("duplicate") {
		return nil
	}

	c.spaces()

	if !c.byte('(') {
		return nil
	}

	c.spaces()

	// The optional var and variables. are each tried taken, then left.
	for _, declared := range []bool{true, false} {
		for _, scoped := range []bool{true, false} {
			at := c
			if declared && (!at.fold("var") || !at.spaces1()) {
				continue
			}

			if scoped && !at.fold("variables.") {
				continue
			}

			name := at.name()
			if name == "" {
				continue
			}

			at.spaces()

			if at.byte(')') {
				return []string{s[:at.i], name}
			}
		}
	}

	return nil
}

// createdComponent is
// (?i)^createObject\s*\(\s*["']component["']\s*,\s*["']([^"']+)["']: a
// createObject of a literal component path at the start of s, and the path.
func createdComponent(s string) []string {
	c := exprCursor{s: s}
	if !createObjectOf(&c, "component") {
		return nil
	}

	c.spaces()

	if !c.byte(',') {
		return nil
	}

	c.spaces()

	if !c.quote() {
		return nil
	}

	path := c.run(isNotQuote)
	if path == "" || !c.quote() {
		return nil
	}

	return []string{s[:c.i], path}
}

// newComponent is (?i)^new\s+(?:cfml:)?([A-Za-z_$#][\w$.#]*)\s*\(: a new
// of a component at the start of s, and the component's name.
func newComponent(s string) []string {
	c := exprCursor{s: s}
	if !c.fold("new") || !c.spaces1() {
		return nil
	}

	read := func(c exprCursor) []string {
		if c.i >= len(s) || !lex.IsWordStart(s[c.i]) && s[c.i] != '$' && s[c.i] != '#' {
			return nil
		}

		name := c.run(func(b byte) bool { return lex.IsIdent(b) || b == '.' || b == '#' })
		c.spaces()

		if !c.byte('(') {
			return nil
		}

		return []string{s[:c.i], name}
	}

	if cfml := c; cfml.fold("cfml:") {
		if m := read(cfml); m != nil {
			return m
		}
	}

	return read(c)
}

// isEmptyLiteral reports whether s is an empty string literal and nothing
// else: two double quotes, or two single quotes.
func isEmptyLiteral(s string) bool { return s == `""` || s == `''` }

// asQueryFalseIn is \basquery\s*[=:]\s*(?:false|no|0)\b, in lowercased
// arguments: they set asQuery to false.
func asQueryFalseIn(lower string) bool {
	for from := 0; ; {
		i := strings.Index(lower[from:], "asquery")
		if i < 0 {
			return false
		}

		i += from
		from = i + 1

		if i > 0 && lex.IsWord(lower[i-1]) {
			continue
		}

		c := exprCursor{s: lower, i: i + len("asquery")}
		c.spaces()

		if !c.byte('=') && !c.byte(':') {
			continue
		}

		c.spaces()

		if (c.lit("false") || c.lit("no") || c.lit("0")) && (c.end() || !lex.IsWord(lower[c.i])) {
			return true
		}
	}
}

// quotedValues is what "([^"]*)"|'([^']*)' finds in s, each string's text
// without its quotes, in order.
func quotedValues(s string) []string {
	var out []string

	for i := 0; i < len(s); i++ {
		if !lex.IsQuote(s[i]) {
			continue
		}

		if j := strings.IndexByte(s[i+1:], s[i]); j >= 0 {
			out = append(out, s[i+1:i+1+j])
			i += 1 + j
		}
	}

	return out
}
