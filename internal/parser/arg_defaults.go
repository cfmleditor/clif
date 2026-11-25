package parser

import (
	"strconv"
	"strings"
)

// defaultExpression is an argument's default as written in a script argument
// list: its source text, a line break and the indentation after it read as one
// space (a struct literal written over several lines stays one expression), and
// "" when nothing is written.
func defaultExpression(src string) string {
	src = strings.TrimSpace(src)
	if !strings.ContainsAny(src, "\r\n") {
		return src
	}

	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}

	return strings.Join(lines, " ")
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
