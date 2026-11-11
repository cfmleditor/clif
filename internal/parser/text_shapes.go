// Fixed shapes of source text the parser looks for without tokenising it: an
// Application.cfc's bean paths, a doc comment's @serviceproperty, a markup
// template's closing tag, and the like. Each was a regular expression, and
// each is read here byte by byte with a cursor, as the scanner reads; the
// expression it replaces is kept beside its test, which checks the two agree
// (text_shapes_test.go).
//
// Case is folded in ASCII alone, as lex.HasPrefixFold does everywhere else:
// a regular expression's (?i) also folds the Kelvin sign into k and the long
// s into s, which no CFML source spells.

package parser

import (
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// textCursor reads s from i.
type textCursor struct {
	s string
	i int
}

// spaces skips \s*.
func (c *textCursor) spaces() {
	for c.i < len(c.s) && lex.IsRegexpSpace(c.s[c.i]) {
		c.i++
	}
}

// spaces1 skips \s+, and reports whether there was any.
func (c *textCursor) spaces1() bool {
	from := c.i
	c.spaces()

	return c.i > from
}

// fold steps over lit, matched without case.
func (c *textCursor) fold(lit string) bool {
	if !lex.HasPrefixFold(c.s[c.i:], lit) {
		return false
	}

	c.i += len(lit)

	return true
}

// byte steps over b.
func (c *textCursor) byte(b byte) bool {
	if c.i < len(c.s) && c.s[c.i] == b {
		c.i++

		return true
	}

	return false
}

// run steps over the bytes in reports true for, and is them.
func (c *textCursor) run(in func(byte) bool) string {
	from := c.i
	for c.i < len(c.s) && in(c.s[c.i]) {
		c.i++
	}

	return c.s[from:c.i]
}

// quoted steps over ["']([^"'<stop>]{min,})["'] and is the text between the
// quotes. Either quote may open and either close it, as the expressions this
// replaces read one.
func (c *textCursor) quoted(stop string, minLen int) (string, bool) {
	from := c.i
	if c.i >= len(c.s) || !lex.IsQuote(c.s[c.i]) {
		return "", false
	}

	c.i++
	value := c.run(func(b byte) bool { return !lex.IsQuote(b) && strings.IndexByte(stop, b) < 0 })

	if len(value) < minLen || c.i >= len(c.s) || !lex.IsQuote(c.s[c.i]) {
		c.i = from

		return "", false
	}

	c.i++

	return value, true
}

// foldedMatches calls match at each place in s that lit starts, matched
// without case, with a cursor past lit, and moves on from where a match
// ends, or one byte on from a failed one: a regular expression's FindAll for
// a shape that starts with lit.
func foldedMatches(s, lit string, match func(c *textCursor, start int) bool) {
	for from := 0; from < len(s); {
		i := IndexFold(s[from:], lit)
		if i < 0 {
			return
		}

		i += from
		c := textCursor{s: s, i: i + len(lit)}

		if match(&c, i) {
			from = c.i
		} else {
			from = i + 1
		}
	}
}

// firstFolded is foldedMatches stopped at the first match: whether there is
// one, a regular expression's FindString.
func firstFolded(s, lit string, match func(c *textCursor, start int) bool) bool {
	for from := 0; from < len(s); {
		i := IndexFold(s[from:], lit)
		if i < 0 {
			return false
		}

		i += from
		c := textCursor{s: s, i: i + len(lit)}

		if match(&c, i) {
			return true
		}

		from = i + 1
	}

	return false
}

// beanPathAssignment is one this.beanPaths["ns"] = expandPath("dir") or
// this.beanPaths["ns"] = "dir".
type beanPathAssignment struct{ namespace, dir string }

// beanPathAssignments are content's this.beanPaths assignments, in order.
func beanPathAssignments(content string) []beanPathAssignment {
	var out []beanPathAssignment

	foldedMatches(content, "this.beanPaths[", func(c *textCursor, _ int) bool {
		c.spaces()

		ns, ok := c.quoted("", 0)
		if !ok {
			return false
		}

		c.spaces()

		if !c.byte(']') {
			return false
		}

		c.spaces()

		if !c.byte('=') {
			return false
		}

		c.spaces()

		var dir string

		if c.fold("expandPath(") {
			c.spaces()

			if dir, ok = c.quoted("", 1); !ok {
				return false
			}

			c.spaces()

			if !c.byte(')') {
				return false
			}
		} else if dir, ok = c.quoted("", 1); !ok {
			return false
		}

		out = append(out, beanPathAssignment{namespace: ns, dir: dir})

		return true
	})

	return out
}

// diLocations is the value of the first [variables.]framework.diLocations =
// "a,b" in content.
func diLocations(content string) (string, bool) {
	var value string

	found := firstFolded(content, "framework.diLocations", func(c *textCursor, _ int) bool {
		c.spaces()

		if !c.byte('=') {
			return false
		}

		c.spaces()

		v, ok := c.quoted("", 1)
		value = v

		return ok
	})

	return value, found
}

// extendsFW1 reports whether content says extends="framework.one".
func extendsFW1(content string) bool {
	return firstFolded(content, "extends", func(c *textCursor, _ int) bool {
		c.spaces()

		if !c.byte('=') {
			return false
		}

		c.spaces()

		if c.i >= len(c.s) || !lex.IsQuote(c.s[c.i]) {
			return false
		}

		c.i++

		return c.fold("framework.one") && c.i < len(c.s) && lex.IsQuote(c.s[c.i])
	})
}

// mappingTargetKey is the key of term when term is this.mappings["/key"],
// alone, in any case and either quote.
func mappingTargetKey(term string) (string, bool) {
	c := textCursor{s: term}
	if !c.fold("this.mappings[") {
		return "", false
	}

	c.spaces()

	key, ok := c.quoted("", 1)
	if !ok {
		return "", false
	}

	c.spaces()

	if !c.byte(']') || c.i != len(term) {
		return "", false
	}

	return key, true
}

// serviceProperty is one "@serviceproperty name kind|target" doc annotation,
// and the offset it starts at.
type serviceProperty struct {
	start              int
	name, kind, target string
}

// servicePropertyAnnotations are content's @serviceproperty annotations, in
// order: "@serviceproperty varName kind|name", usually inside a CFML tag
// comment, e.g. "<!--- @serviceproperty donorObj package|sandbox.tasscom.components.donor --->".
// The tag is matched with its case.
//
// This is a project-specific documentation convention (not standard CFML/CFC syntax) for
// annotating the real component type of a dependency that's declared with a generic
// <cfargument type="struct"> (CF doesn't enforce component types on constructor args, so
// projects sometimes type them loosely and document the real type in a comment instead).
// varName allows "." and "[...]" (e.g. "result.data[i]") so the same convention can
// annotate a bracket-indexed receiver — a case CanResolveCall otherwise can't resolve at
// all without a global componentResolver keyed on the bare receiver text, which risks
// matching an unrelated same-named receiver anywhere else in the workspace. An in-source
// annotation is scoped to this one file, so it doesn't carry that risk.
func servicePropertyAnnotations(content string) []serviceProperty {
	var out []serviceProperty

	const tag = "@serviceproperty"

	for from := 0; from < len(content); {
		i := strings.Index(content[from:], tag)
		if i < 0 {
			break
		}

		i += from
		from = i + 1
		c := textCursor{s: content, i: i + len(tag)}

		if !c.spaces1() || c.i >= len(content) || !lex.IsWordStart(content[c.i]) {
			continue
		}

		name := c.run(func(b byte) bool {
			return lex.IsWordStart(b) || b >= '0' && b <= '9' || b == '.' || b == '[' || b == ']'
		})

		if !c.spaces1() {
			continue
		}

		kind := c.run(lex.IsASCIILetter)
		if kind == "" || !c.byte('|') {
			continue
		}

		target := c.run(func(b byte) bool { return !lex.IsRegexpSpace(b) })
		if target == "" {
			continue
		}

		out = append(out, serviceProperty{start: i, name: name, kind: kind, target: target})
		from = c.i
	}

	return out
}

// mockClassName is the value of the first className="X" or className: "X" in
// args that starts a word, X holding no #.
func mockClassName(args string) string {
	value := ""

	firstFolded(args, "className", func(c *textCursor, start int) bool {
		if start > 0 && lex.IsWord(args[start-1]) {
			return false
		}

		c.spaces()

		if !c.byte('=') && !c.byte(':') {
			return false
		}

		c.spaces()

		v, ok := c.quoted("#", 1)
		value = v

		return ok
	})

	return value
}

// hasMarkupEvidence reports whether content holds what only HTML does: a
// closing tag (</name>), a doctype, or an HTML comment (<!-- not followed by
// a third -, which would open a CFML one). A script fragment can hold "a<b",
// but not "</div>".
func hasMarkupEvidence(content string) bool {
	for from := 0; ; {
		i := strings.IndexByte(content[from:], '<')
		if i < 0 {
			return false
		}

		i += from
		from = i + 1
		rest := content[i:]

		switch {
		case strings.HasPrefix(rest, "</"):
			c := textCursor{s: rest, i: 2}
			if c.i < len(rest) && lex.IsASCIILetter(rest[c.i]) {
				c.run(func(b byte) bool { return lex.IsASCIILetter(b) || b >= '0' && b <= '9' || b == '-' })
				c.spaces()

				if c.byte('>') {
					return true
				}
			}
		case lex.HasPrefixFold(rest, "<!doctype"):
			return true
		case strings.HasPrefix(rest, "<!--") && len(rest) > 4 && rest[4] != '-':
			return true
		}
	}
}

// placeholderSpans are the offsets of each $N placeholder in pattern, a $
// and its digits, as start and end pairs.
func placeholderSpans(pattern string) [][2]int {
	var out [][2]int

	for i := 0; i < len(pattern); i++ {
		if pattern[i] != '$' {
			continue
		}

		j := i + 1
		for j < len(pattern) && pattern[j] >= '0' && pattern[j] <= '9' {
			j++
		}

		if j > i+1 {
			out = append(out, [2]int{i, j})
			i = j - 1
		}
	}

	return out
}

// hasPlaceholder reports whether pattern holds a $N placeholder.
func hasPlaceholder(pattern string) bool { return len(placeholderSpans(pattern)) > 0 }

// splitPlaceholders is pattern cut at each $N placeholder: the text before,
// between and after them.
func splitPlaceholders(pattern string) []string {
	out := make([]string, 0, 2)
	from := 0

	for _, s := range placeholderSpans(pattern) {
		out = append(out, pattern[from:s[0]])
		from = s[1]
	}

	return append(out, pattern[from:])
}

// withoutPlaceholders is pattern with its $N placeholders cut out.
func withoutPlaceholders(pattern string) string {
	return strings.Join(splitPlaceholders(pattern), "")
}

// serviceLoop is the head of a startup service loop: for ( name in
// listToArray( list ) ) {, its iterator, the list expression and where its
// body starts.
type serviceLoop struct {
	iterator, list string
	bodyStart      int
}

// serviceLoopHead reads a service loop's head at the start of s.
func serviceLoopHead(s string) (serviceLoop, bool) {
	c := textCursor{s: s}
	if !c.fold("for") {
		return serviceLoop{}, false
	}

	c.spaces()

	if !c.byte('(') {
		return serviceLoop{}, false
	}

	c.spaces()

	iterator, ok := c.scopedName()
	if !ok || !c.spaces1() || !c.fold("in") || !c.spaces1() || !c.fold("listToArray") {
		return serviceLoop{}, false
	}

	c.spaces()

	if !c.byte('(') {
		return serviceLoop{}, false
	}

	// \s*([^()]+): the leading space is skipped, unless it is all there is,
	// when the last of it is the list.
	from := c.i
	c.spaces()

	list := c.run(func(b byte) bool { return b != '(' && b != ')' })
	if list == "" {
		if c.i == from {
			return serviceLoop{}, false
		}

		list = s[c.i-1 : c.i]
	}

	for _, b := range []byte{')', ')'} {
		c.spaces()

		if !c.byte(b) {
			return serviceLoop{}, false
		}
	}

	c.spaces()

	if !c.byte('{') {
		return serviceLoop{}, false
	}

	return serviceLoop{iterator: iterator, list: list, bodyStart: c.i}, true
}

// scopedName steps over (variables.|local.)?[\w$]+ and is it.
func (c *textCursor) scopedName() (string, bool) {
	from := c.i

	for _, scope := range []string{"variables.", "local."} {
		if c.fold(scope) {
			if c.run(lex.IsIdent) != "" {
				return c.s[from:c.i], true
			}

			c.i = from
		}
	}

	if c.run(lex.IsIdent) == "" {
		return "", false
	}

	return c.s[from:c.i], true
}

// serviceWrite is application[ "#key#" ] = factory.getBean( "#bean#" );, read
// from the start of a loop body statement.
type serviceWrite struct{ scope, key, factory, bean string }

// serviceMemberWriteAt reads a service write at the start of s.
func serviceMemberWriteAt(s string) (serviceWrite, bool) {
	var w serviceWrite

	scopes := []string{"application", "request", "session", "server"}

	n := slices.IndexFunc(scopes, func(scope string) bool { return lex.HasPrefixFold(s, scope) })
	if n < 0 {
		return w, false
	}

	c := textCursor{s: s, i: len(scopes[n])}
	w.scope = s[:c.i]

	isName := func(b byte) bool { return lex.IsIdent(b) || b == '.' }

	c.spaces()

	if !c.byte('[') {
		return w, false
	}

	c.spaces()

	key, ok := c.hashName(isName)
	if !ok {
		return w, false
	}

	w.key = key

	c.spaces()

	if !c.byte(']') {
		return w, false
	}

	c.spaces()

	if !c.byte('=') {
		return w, false
	}

	c.spaces()

	// ([\w.$]+)\.getBean: the name runs on through .getBean, which must end it.
	const getBean = ".getBean"

	name := c.run(isName)
	if len(name) <= len(getBean) || !strings.EqualFold(name[len(name)-len(getBean):], getBean) {
		return w, false
	}

	w.factory = name[:len(name)-len(getBean)]

	c.spaces()

	if !c.byte('(') {
		return w, false
	}

	c.spaces()

	if w.bean, ok = c.hashName(isName); !ok {
		return w, false
	}

	c.spaces()

	if !c.byte(')') {
		return w, false
	}

	c.spaces()

	return w, c.byte(';')
}

// hashName steps over ["']#\s*(name)\s*#["'] and is the name.
func (c *textCursor) hashName(isName func(byte) bool) (string, bool) {
	if c.i >= len(c.s) || !lex.IsQuote(c.s[c.i]) {
		return "", false
	}

	c.i++

	if !c.byte('#') {
		return "", false
	}

	c.spaces()

	name := c.run(isName)
	if name == "" {
		return "", false
	}

	c.spaces()

	if !c.byte('#') || c.i >= len(c.s) || !lex.IsQuote(c.s[c.i]) {
		return "", false
	}

	c.i++

	return name, true
}

// listAppendCall reads expr when it is listAppend( list, "item" ) and nothing
// else: the list's name, and the item with its quotes.
func listAppendCall(expr string) (list, item string, ok bool) {
	c := textCursor{s: expr}
	if !c.fold("listAppend") {
		return "", "", false
	}

	c.spaces()

	if !c.byte('(') {
		return "", "", false
	}

	c.spaces()

	if list, ok = c.scopedName(); !ok {
		return "", "", false
	}

	c.spaces()

	if !c.byte(',') {
		return "", "", false
	}

	c.spaces()

	from := c.i
	if _, ok = c.quoted("#", 0); !ok {
		return "", "", false
	}

	item = expr[from:c.i]

	c.spaces()

	if !c.byte(')') || c.i != len(expr) {
		return "", "", false
	}

	return list, item, true
}
