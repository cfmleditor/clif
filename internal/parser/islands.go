package parser

import "strings"

// A script may hold text that is not script, and the scanner reads all of it as
// code, so SQL's `VARCHAR(255)` and `sum(x)` and a Java method's signature
// became calls: findings for functions nobody called.
//
//	query datasource=ds {``` CREATE TABLE t ( v VARCHAR(255) ) ```}
//	```<cfquery name="q"> select sum(x) from t where a = #f(y)# </cfquery>```
//	obj = java{ public class c { public String execute() { … } } }
//
// blankNonScript replaces that text with spaces before the scan, byte for byte
// and keeping every newline, so no line or offset moves. What a `#...#` holds is
// kept, since it is CFML and a call in it is a call.

// blankNonScript is src with its SQL islands and Java blocks blanked, and src
// itself, unchanged and without a copy, when it holds neither.
func blankNonScript(src string) string {
	island := strings.Contains(src, "```")

	java := false
	if !island {
		java = IndexFold(src, "java") >= 0
	}

	if !island && !java {
		return src
	}

	lb := &lazyBytes{src: src}
	changed := false

	if island {
		changed = blankIslands(lb, src) || changed
	}

	if island || java {
		changed = blankJavaBlocks(lb, src) || changed
	}

	if !changed {
		return src
	}

	return string(lb.b)
}

// lazyBytes is src as a writable copy made on the first write. A file that
// mentions "java" (createObject( "java", … ) is everywhere) almost never holds a
// block to blank, and copying it twice to find that out was most of this
// function's cost.
type lazyBytes struct {
	src string
	b   []byte
}

func (l *lazyBytes) get() []byte {
	if l.b == nil {
		l.b = []byte(l.src)
	}

	return l.b
}

// blankIslands blanks the SQL in the triple-backtick islands of src: the whole
// body of a script `query { … }`, and the body of each <cfquery> in any island.
func blankIslands(b *lazyBytes, src string) bool {
	changed := false

	for open := strings.Index(src, "```"); open >= 0; {
		from := open + 3

		end := strings.Index(src[from:], "```")
		if end < 0 {
			break
		}

		end += from

		if opensScriptQuery(src, open) {
			blankKeepingExpressions(b, src, from, end)

			changed = true
		} else if blankQueryBodies(b, src, from, end) {
			changed = true
		}

		next := strings.Index(src[end+3:], "```")
		if next < 0 {
			break
		}

		open = end + 3 + next
	}

	return changed
}

// opensScriptQuery reports whether the island opening at offset open is the body
// of a script-syntax `query … {`: a '{' stands before it, and the statement that
// brace opens begins with the word query.
func opensScriptQuery(src string, open int) bool {
	i := open
	for i > 0 && isWhitespace(src[i-1]) {
		i--
	}

	if i == 0 || src[i-1] != '{' {
		return false
	}

	start := i - 1
	for start > 0 && src[start-1] != ';' && src[start-1] != '{' && src[start-1] != '}' {
		start--
	}

	head := strings.TrimLeft(src[start:i-1], " \t\r\n")

	return len(head) >= 5 && strings.EqualFold(head[:5], "query") && (len(head) == 5 || !isWordByte(head[5]))
}

// blankQueryBodies blanks the text between each <cfquery …> and </cfquery> in
// src[from:to], and reports whether it blanked any.
func blankQueryBodies(b *lazyBytes, src string, from, to int) bool {
	changed := false

	for pos := from; pos < to; {
		q := IndexFoldFrom(src[:to], "<cfquery", pos)
		if q < 0 {
			break
		}

		afterName := q + len("<cfquery")
		if afterName < to && isWordByte(src[afterName]) {
			pos = afterName

			continue
		}

		gt := tagEndIndex(src[afterName:to])
		if gt < 0 {
			break
		}

		body := afterName + gt + 1

		closeAt := IndexFoldFrom(src[:to], "</cfquery>", body)
		if closeAt < 0 {
			closeAt = to
		}

		blankKeepingExpressions(b, src, body, closeAt)

		changed = true
		pos = closeAt
	}

	return changed
}

// blankKeepingExpressions replaces the bytes of src[from:to] with spaces, except
// newlines and the expression inside each #...# span, which is left to be read
// as the script it is. The hashes themselves go.
func blankKeepingExpressions(lb *lazyBytes, src string, from, to int) {
	spans := interpolatedSpans(src[from:to])
	b := lb.get()
	next := 0

	for i := from; i < to; i++ {
		for next < len(spans) && i >= from+spans[next].end {
			next++
		}

		if next < len(spans) && i >= from+spans[next].start {
			continue
		}

		if b[i] != '\n' && b[i] != '\r' {
			b[i] = ' '
		}
	}
}

// blankJavaBlocks blanks the body of each `java { … }` block in src, and reports
// whether it blanked any.
func blankJavaBlocks(lb *lazyBytes, src string) bool {
	changed := false

	for pos := 0; pos < len(src); {
		j := IndexFoldFrom(src, "java", pos)
		if j < 0 {
			break
		}

		pos = j + len("java")

		if j > 0 && (isWordByte(src[j-1]) || src[j-1] == '.' || src[j-1] == '"' || src[j-1] == '\'') {
			continue
		}

		open := pos
		for open < len(src) && isWhitespace(src[open]) {
			open++
		}

		if open >= len(src) || src[open] != '{' {
			continue
		}

		closeAt := javaBlockEnd(src, open)
		if closeAt < 0 {
			continue
		}

		b := lb.get()

		for i := open + 1; i < closeAt; i++ {
			if b[i] != '\n' && b[i] != '\r' {
				b[i] = ' '
			}
		}

		changed = true
		pos = closeAt
	}

	return changed
}

// javaBlockEnd is the offset of the '}' that closes the '{' at open, reading the
// Java between them for what holds a brace that is not a block: strings,
// character literals and comments. -1 when it does not close.
func javaBlockEnd(src string, open int) int {
	depth := 0

	for i := open; i < len(src); i++ {
		switch c := src[i]; c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		case '"', '\'':
			for i++; i < len(src) && src[i] != c; i++ {
				if src[i] == '\\' {
					i++
				}
			}
		case '/':
			switch {
			case i+1 < len(src) && src[i+1] == '/':
				for i < len(src) && src[i] != '\n' {
					i++
				}
			case i+1 < len(src) && src[i+1] == '*':
				end := strings.Index(src[i+2:], "*/")
				if end < 0 {
					return -1
				}

				i += 2 + end + 1
			}
		}
	}

	return -1
}
