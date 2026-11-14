package resolve

import "strings"

// A Wheels method may end in a dispatch on a literal mode, a chain of
//
//	if ( arguments.mode eq "a" ) { return x ; } else if ( … ) { return y ; } else { return this ; }
//
// read from the method's body as wheelsWriteToken wrote it: each token
// lowercased and followed by a tab, so the grammar is the tokens with tabs
// between. These readers replace the two expressions that read it, and answer
// as each did (wheels_dispatch_test.go holds them to it). Nothing here needs a
// word boundary: a token the text ends with `if` begins a condition as well.

const (
	// selfDispatchFinal is the last branch, which must end the body.
	selfDispatchFinal = "{\treturn\tthis\t;\t}\t"
	selfDispatchElse  = "\telse\t"
)

// selfDispatchHead is the text a condition on the parameter name begins with.
func selfDispatchHead(name string) string {
	return "if\t(\targuments\t.\t" + name + "\t"
}

// selfDispatchOp reads the comparison, `eq` or `= =`, at i: where it ends.
func selfDispatchOp(s string, i int) (int, bool) {
	switch {
	case strings.HasPrefix(s[i:], "eq"):
		return i + len("eq"), true
	case strings.HasPrefix(s[i:], "=\t="):
		return i + len("=\t="), true
	}

	return 0, false
}

// selfDispatchLiteral reads `"literal"` at i, a string holding no quote or
// hash, returning the literal's bounds and where the closing quote ends.
func selfDispatchLiteral(s string, i int) (from, to, end int, ok bool) {
	if !strings.HasPrefix(s[i:], "\"") {
		return 0, 0, 0, false
	}

	from = i + 1

	to = from
	for to < len(s) && s[to] != '"' && s[to] != '#' {
		to++
	}

	if to >= len(s) || s[to] != '"' {
		return 0, 0, 0, false
	}

	return from, to, to + 1, true
}

// selfDispatchCondition reads one condition at i,
//
//	if ( arguments . NAME (eq|= =) "literal" ) { return EXPRESSION ; }
//
// and returns the index just past its closing brace. The expression is a run
// of characters that are not a semicolon or a brace, and the tab before the
// semicolon is not part of it, so it holds at least one character.
func selfDispatchCondition(s string, i int, name string) (int, bool) {
	head := selfDispatchHead(name)
	if !strings.HasPrefix(s[i:], head) {
		return 0, false
	}

	i, ok := selfDispatchOp(s, i+len(head))
	if !ok || !strings.HasPrefix(s[i:], "\t") {
		return 0, false
	}

	_, _, i, ok = selfDispatchLiteral(s, i+1)
	if !ok {
		return 0, false
	}

	const open = "\t)\t{\treturn\t"
	if !strings.HasPrefix(s[i:], open) {
		return 0, false
	}

	start := i + len(open)

	stop := start
	for stop < len(s) && s[stop] != ';' && s[stop] != '{' && s[stop] != '}' {
		stop++
	}

	if stop >= len(s) || s[stop] != ';' || stop-start < 2 || s[stop-1] != '\t' {
		return 0, false
	}

	const closing = ";\t}"
	if !strings.HasPrefix(s[stop:], closing) {
		return 0, false
	}

	return stop + len(closing), true
}

// selfDispatchChain reports whether the text from i is a chain of conditions,
// each followed by `else`, the last else branch returning this and ending the
// body.
func selfDispatchChain(s string, i int, name string) bool {
	for {
		end, ok := selfDispatchCondition(s, i, name)
		if !ok {
			return false
		}

		rest, ok := strings.CutPrefix(s[end:], selfDispatchElse)
		if !ok {
			return false
		}

		i = len(s) - len(rest)

		if rest == selfDispatchFinal {
			return true
		}
	}
}

// selfDispatchStart is where the final literal-mode dispatch on arguments.NAME
// begins in body, the leftmost chain that runs to its end: the start of
// FindStringIndex of
//
//	COND(?:\telse\tCOND)*\telse\t\{\treturn\tthis\t;\t\}\t$
//
// the match ending with the body.
func selfDispatchStart(body, name string) (int, bool) {
	head := selfDispatchHead(name)

	for from := 0; from <= len(body); {
		at := strings.Index(body[from:], head)
		if at < 0 {
			return 0, false
		}

		at += from
		if selfDispatchChain(body, at, name) {
			return at, true
		}

		from = at + 1
	}

	return 0, false
}

// selfDispatchModes is the literal of each `arguments.NAME eq "literal"` in
// text, in order and not overlapping: FindAllStringSubmatch of
// `arguments\t\.\tNAME\t(?:eq|=\t=)\t"([^"#]*)"`, group one.
func selfDispatchModes(text, name string) []string {
	var out []string

	head := "arguments\t.\t" + name + "\t"

	for from := 0; from <= len(text); {
		at := strings.Index(text[from:], head)
		if at < 0 {
			return out
		}

		at += from

		if i, ok := selfDispatchOp(text, at+len(head)); ok && strings.HasPrefix(text[i:], "\t") {
			if lo, hi, end, ok := selfDispatchLiteral(text, i+1); ok {
				out = append(out, text[lo:hi])
				from = end

				continue
			}
		}

		from = at + 1
	}

	return out
}
