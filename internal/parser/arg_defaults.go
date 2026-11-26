package parser

import (
	"strconv"
	"strings"
)

// defaultExpression is an argument's default as written in a script argument
// list: its source text, a line break and the indentation after it read as one
// space (a struct literal written over several lines stays one expression), and
// "" when nothing is written.
//
// A comment inside it is dropped. Kept, a line comment would swallow whatever
// the default is written next to once the lines are joined (`1 // note` ahead
// of the next argument), and a stub that writes the default back would be
// broken. The text is then rebuilt from the scanner's tokens, which keep a
// string whole (`"http://x"` holds no comment) and skip the comments; a
// default with no comment marker in it is its source, as written.
func defaultExpression(src string) string {
	src = strings.TrimSpace(src)

	if strings.Contains(src, "//") || strings.Contains(src, "/*") {
		return withoutComments(src)
	}

	if !strings.ContainsAny(src, "\r\n") {
		return src
	}

	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}

	return strings.Join(lines, " ")
}

// withoutComments is src with its comments left out and each gap between two
// tokens (whitespace, a line break, a comment) read as one space, as the
// scanner sees it.
func withoutComments(src string) string {
	sc := NewScriptScanner(src)

	var b strings.Builder

	prevEnd := 0

	for {
		tok := sc.NextSkipComments()
		if tok.Kind == TokEOF {
			break
		}

		end := sc.Pos()

		if b.Len() > 0 && tok.Offset > prevEnd {
			b.WriteByte(' ')
		}

		b.WriteString(src[tok.Offset:end])

		prevEnd = end
	}

	return b.String()
}

// tagDefaultExpression reads a <cfargument>'s default="..." as the expression a
// script argument would write. A value that is one #expression# is that
// expression; one with interpolation inside is a string with it; a number for
// a numeric argument and true or false for a boolean one are themselves; and
// anything else is a string, with its quotes doubled.
func tagDefaultExpression(value, typ string) string {
	if inner, ok := strings.CutPrefix(value, "#"); ok {
		if inner, ok = strings.CutSuffix(inner, "#"); ok && inner != "" && !strings.Contains(inner, "#") {
			return strings.TrimSpace(inner)
		}
	}

	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "numeric":
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return value
		}
	case "boolean":
		if lower := strings.ToLower(value); lower == "true" || lower == "false" {
			return lower
		}
	default:
	}

	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

// unescapeAttr is the value of the attribute whose text stands at tag[start:end]
// with its quote doubled inside read as one (`"a ""q"" b"` is `a "q" b`), which
// is how CFML spells a quote in a value quoted with it.
func unescapeAttr(tag string, start, end int) string {
	value := tag[start:end]
	if start > 0 && (tag[start-1] == '"' || tag[start-1] == '\'') {
		quote := tag[start-1 : start]
		value = strings.ReplaceAll(value, quote+quote, quote)
	}

	return value
}
