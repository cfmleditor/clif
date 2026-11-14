package parser

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
)

// QueryColumns is the columns of the query variable name, as the last place
// before offset before that gives it one declares them: a <cfquery name="…">
// whose SELECT lists them, `name = queryNew( "a,b" )`, or
// `name = queryExecute( "SELECT a, b FROM t" )`. ok is false when no such
// declaration is found, or the one found cannot say (`SELECT *`, a list
// written with CF tags or `#…#`). A column the declaration names with an
// expression it cannot read, `count(*)` with no alias, is left out of a list
// that is otherwise known.
func QueryColumns(content, name string, before int) (columns []string, ok bool) {
	best := -1

	consider := func(start int, cols []string, known bool) {
		if start < before && start > best {
			best, columns, ok = start, cols, known && len(cols) > 0
		}
	}

	var openAt, openEnd int

	opened := false

	for _, tag := range CFTags(content, "cfquery") {
		if tag.Close {
			if opened {
				cols, known := sqlSelectColumns(content[openEnd:tag.Start])
				consider(openAt, cols, known)
			}

			opened = false

			continue
		}

		if v, has := tag.Attr("name"); has && strings.EqualFold(unscoped(v), name) {
			openAt, openEnd, opened = tag.Start, tag.End, true
		} else {
			opened = false
		}
	}

	sc := NewSourceScanner(content)

	for t := sc.NextSkipComments(); t.Kind != TokEOF; t = sc.NextSkipComments() {
		if t.Kind != TokIdent || !strings.EqualFold(t.Value, name) || sc.PeekSkipComments().Kind != TokEquals {
			continue
		}

		start := t.Offset

		sc.NextSkipComments() // =

		fn := sc.NextSkipComments()
		if fn.Kind != TokIdent || sc.PeekSkipComments().Kind != TokLParen {
			continue
		}

		sc.NextSkipComments() // (

		arg := sc.NextSkipComments()
		if arg.Kind != TokString || len(arg.Value) < 2 {
			continue
		}

		text := arg.Value[1 : len(arg.Value)-1]

		switch {
		case strings.EqualFold(fn.Value, "queryNew"):
			consider(start, splitColumnList(text), true)
		case strings.EqualFold(fn.Value, "queryExecute"):
			cols, known := sqlSelectColumns(text)
			consider(start, cols, known)
		}
	}

	return columns, ok
}

// splitColumnList is the names in a queryNew column list, "id,name".
func splitColumnList(list string) []string {
	var out []string

	for part := range strings.SplitSeq(list, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}

	return out
}

// sqlSelectColumns is the names a SELECT's columns are given: the alias of
// each item, or the column's own name with its table dropped. known is false
// when the list cannot be read whole: `*`, or no SELECT.
func sqlSelectColumns(sql string) (columns []string, known bool) {
	from, to, found := selectListBounds(sql)
	if !found {
		return nil, false
	}

	for _, item := range splitSQLTopLevel(sql[from:to], ',') {
		item = strings.TrimSpace(item)

		switch {
		case item == "":
		case strings.HasSuffix(item, "*"):
			return nil, false
		default:
			if name := columnName(item); name != "" {
				columns = append(columns, name)
			}
		}
	}

	return columns, true
}

// selectListBounds is where the item list of the first SELECT in sql stands:
// after the keyword and any DISTINCT, ALL or TOP n, up to its FROM at the
// statement's own level, or the end when there is none.
func selectListBounds(sql string) (from, to int, ok bool) {
	i := sqlWordAt(sql, 0, "select")
	if i < 0 {
		return 0, 0, false
	}

	from = i

	for _, skip := range []string{"distinct", "all"} {
		if j := sqlWordAt(sql, from, skip); j >= 0 {
			from = j
		}
	}

	if j := sqlWordAt(sql, from, "top"); j >= 0 {
		from = skipSQLNumberOrParens(sql, j)
	}

	depth := 0

	for k := from; k < len(sql); k++ {
		switch c := sql[k]; {
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == '\'' || c == '"':
			k = skipSQLString(sql, k)
		case depth == 0 && isSQLWordStart(sql, k) && sqlWordAt(sql, k, "from") == k+len("from"):
			return from, k, true
		}
	}

	return from, len(sql), true
}

// sqlWordAt is the offset just past the word w, which must follow the blanks
// at from, or -1.
func sqlWordAt(sql string, from int, w string) int {
	i := from
	for i < len(sql) && lex.IsRegexpSpace(sql[i]) {
		i++
	}

	end := i + len(w)
	if end > len(sql) || !strings.EqualFold(sql[i:end], w) || end < len(sql) && lex.IsWord(sql[end]) {
		return -1
	}

	return end
}

// isSQLWordStart reports whether a word begins at k, not part of a longer one.
func isSQLWordStart(sql string, k int) bool { return k == 0 || !lex.IsWord(sql[k-1]) }

// skipSQLNumberOrParens is the offset past the number, or the parenthesised
// number, that follows TOP.
func skipSQLNumberOrParens(sql string, from int) int {
	i := from
	for i < len(sql) && lex.IsRegexpSpace(sql[i]) {
		i++
	}

	if i < len(sql) && sql[i] == '(' {
		if c := strings.IndexByte(sql[i:], ')'); c >= 0 {
			return i + c + 1
		}

		return len(sql)
	}

	for i < len(sql) && sql[i] >= '0' && sql[i] <= '9' {
		i++
	}

	return i
}

// skipSQLString is the offset of the quote that closes the string opened at
// k, or the last byte; a doubled quote is part of the string.
func skipSQLString(sql string, k int) int {
	q := sql[k]

	for i := k + 1; i < len(sql); i++ {
		if sql[i] != q {
			continue
		}

		if i+1 < len(sql) && sql[i+1] == q {
			i++

			continue
		}

		return i
	}

	return len(sql) - 1
}

// splitSQLTopLevel splits s at sep outside parentheses and strings.
func splitSQLTopLevel(s string, sep byte) []string {
	var (
		out   []string
		depth int
		last  int
	)

	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == '\'' || c == '"':
			i = skipSQLString(s, i)
		case c == sep && depth == 0:
			out = append(out, s[last:i])
			last = i + 1
		}
	}

	return append(out, s[last:])
}

// columnName is the name a SELECT item gives its column: its alias, the last
// word after a blank outside any parentheses, which is the word after AS when
// there is one (`count(*) total`, `u.id AS userId`), else the column itself
// with its table dropped (`u.name`). A call or expression with no alias has
// none.
func columnName(item string) string {
	depth, cut := 0, -1

	for i := 0; i < len(item); i++ {
		switch c := item[i]; {
		case c == '(':
			depth++
		case c == ')':
			depth--
		case c == '\'' || c == '"':
			i = skipSQLString(item, i)
		case depth == 0 && lex.IsRegexpSpace(c):
			cut = i
		}
	}

	if cut >= 0 {
		alias := strings.TrimSpace(item[cut:])
		before := strings.TrimRight(item[:cut], " \t\r\n")

		// `a + b` and `a || b` have no alias, and a CASE ends in END: a word
		// after an operator, or the keyword closing an expression, names
		// nothing. (`>` is not counted: it ends a CF tag, `</cfif> total`.)
		if before == "" || strings.ContainsRune("+-*/%|=,&^~(", rune(before[len(before)-1])) || strings.EqualFold(alias, "end") {
			return ""
		}

		return unquoteSQLName(alias)
	}

	if strings.ContainsAny(item, "()+-/|") {
		return ""
	}

	name, _ := lastCut(item, '.')

	return unquoteSQLName(name)
}

// lastCut is the text after the last sep in s, or s.
func lastCut(s string, sep byte) (string, bool) {
	if i := strings.LastIndexByte(s, sep); i >= 0 {
		return s[i+1:], true
	}

	return s, false
}

// unquoteSQLName drops the quoting around a name: "a", [a], `a`.
func unquoteSQLName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) >= 2 {
		switch {
		case name[0] == '"' && name[len(name)-1] == '"', name[0] == '[' && name[len(name)-1] == ']', name[0] == '`' && name[len(name)-1] == '`':
			name = name[1 : len(name)-1]
		default:
		}
	}

	if name == "" || !lex.IsWordStart(name[0]) && name[0] != '_' {
		return ""
	}

	for i := 1; i < len(name); i++ {
		if !lex.IsWord(name[i]) && name[i] != ' ' {
			return ""
		}
	}

	return name
}

// unscoped is a query's name without the scope it is written in:
// `local.rows` and `variables.rows` are `rows` where it is used.
func unscoped(name string) string {
	name = strings.TrimSpace(name)

	for _, scope := range []string{"local.", "variables.", "arguments."} {
		if lex.HasPrefixFold(name, scope) {
			return name[len(scope):]
		}
	}

	return name
}
