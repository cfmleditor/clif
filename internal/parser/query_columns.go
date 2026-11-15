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
	// Dot completion asks on every `.`: a file that declares no query cannot
	// have this one, and the tag search and tokenize below are passes over it.
	if !MayDeclareQuery(content) {
		return nil, false
	}

	return DeclaredQueryColumns(content, name, before, nil)
}

// QueryScope limits where a query is looked for. A query is a variable of the
// function that declares it: the search is within the function around the
// cursor, or, outside every function, passes over them all. The text is the
// whole file either way, so it is read as the file is, script or tags.
type QueryScope struct {
	From, To int            // only declarations from From to To are read; To <= 0 is the end
	Skip     []FunctionSpan // declarations in these are not read
}

// DeclaredQueryColumns is QueryColumns without the test of MayDeclareQuery, for
// a caller that has made it, and within scope, nil for the whole text.
func DeclaredQueryColumns(content, name string, before int, scope *QueryScope) (columns []string, ok bool) {
	if name == "" {
		return nil, false
	}

	inSkip := func(off int) bool {
		if scope == nil {
			return false
		}

		if off < scope.From || scope.To > 0 && off >= scope.To {
			return true
		}

		for i := range scope.Skip {
			if off >= scope.Skip[i].Start && off < scope.Skip[i].End {
				return true
			}
		}

		return false
	}

	best := -1

	consider := func(start int, cols []string, known bool) {
		if start < before && start > best {
			best, columns, ok = start, cols, known && len(cols) > 0
		}
	}

	// A scope within a script file is scanned as that range alone, read as the
	// file is, and its tokens are relative to base; any other text is scanned
	// whole, and its tags are searched.
	from, to := 0, len(content)
	if scope != nil && scope.To > 0 {
		from, to = scope.From, scope.To
	}

	script := isScriptFile(content)
	sc, base := sourceScannerRange(content, from, to, script)

	if !script {
		queryTagColumns(content, name, inSkip, consider)
	}

	// read is the next token, with the two before it kept: a `.` before a
	// name, and what that `.` follows.
	var prev, owner Token

	var last Token

	read := func() Token {
		t := sc.NextSkipComments()
		prev, owner, last = last, prev, t

		return t
	}

	for t := read(); t.Kind != TokEOF; t = read() {
		if t.Kind != TokIdent || !strings.EqualFold(t.Value, name) || sc.PeekSkipComments().Kind != TokEquals || inSkip(base+t.Offset) {
			continue
		}

		// `cfg.users = ...` sets another object's member, and a struct's
		// `{ a = 1, users = ... }` a key; `local.users` is the variable.
		if prev.Kind == TokComma || prev.Kind == TokDot && !isLocalScope(owner) || prev.Kind == TokLBrace && opensStruct(owner) {
			continue
		}

		// A statement sets the variable; a named argument, a condition and an
		// arrow's parameter `users => ...` stand elsewhere.
		if !startsAssignment(prev, owner) {
			continue
		}

		start := base + t.Offset

		eq := read() // =

		// `users == 1` compares, and `users => ...` is an arrow: neither sets it.
		if next := sc.PeekSkipComments(); (next.Kind == TokEquals || next.Kind == TokGT) && next.Offset == eq.Offset+1 {
			continue
		}

		// What the variable is set to, when it is not a query that can be
		// read: its earlier columns no longer hold.
		fn := read()
		if fn.Kind != TokIdent || sc.PeekSkipComments().Kind != TokLParen {
			consider(start, nil, false)

			continue
		}

		read() // (

		arg := read()
		if arg.Kind != TokString || len(arg.Value) < 2 {
			consider(start, nil, false)

			continue
		}

		// A string that goes on, `"SELECT id, " & cols & " FROM t"`, is the
		// front of one that is not known.
		if next := sc.PeekSkipComments(); next.Kind != TokComma && next.Kind != TokRParen {
			consider(start, nil, false)

			continue
		}

		text := arg.Value[1 : len(arg.Value)-1]

		switch {
		case strings.EqualFold(fn.Value, "queryNew"):
			// A list with a `#...#` is not the text it spells.
			consider(start, splitColumnList(text), !strings.Contains(text, "#"))
		case strings.EqualFold(fn.Value, "queryExecute"):
			cols, known := sqlSelectColumns(text)
			consider(start, cols, known)
		default:
			consider(start, nil, false)
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
	sql = blankSQLComments(sql)

	from, to, found := selectListBounds(sql)
	if !found {
		return nil, false
	}

	// A list written with `#…#` or CF tags is not what its text says.
	if list := sql[from:to]; strings.Contains(list, "#") || strings.Contains(strings.ToLower(list), "<cf") || strings.Contains(strings.ToLower(list), "</cf") {
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
		case c == '\'' || c == '"' || c == '`':
			i = skipSQLString(item, i)
		case c == '[':
			// `[user id]`: a name in brackets holds blanks.
			if end := strings.IndexByte(item[i:], ']'); end >= 0 {
				i += end
			}
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
		case name[0] == '"' && name[len(name)-1] == '"', name[0] == '[' && name[len(name)-1] == ']', name[0] == '`' && name[len(name)-1] == '`', name[0] == '\'' && name[len(name)-1] == '\'':
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

// isLocalScope reports whether t names a scope whose members are the
// function's own variables.
func isLocalScope(t Token) bool {
	return t.Kind == TokIdent && (strings.EqualFold(t.Value, "local") || strings.EqualFold(t.Value, "variables"))
}

// opensStruct reports whether a `{` after t starts a struct literal, which
// stands where a value does, rather than a block.
func opensStruct(t Token) bool {
	switch t.Kind {
	case TokEquals, TokLParen, TokComma, TokColon, TokLBracket:
		return true
	default:
		return false
	}
}

// MayDeclareQuery reports whether content spells anything that declares a
// query variable. A cheap test, for the per-keystroke callers of
// QueryColumns to ask before they build anything for it.
func MayDeclareQuery(content string) bool {
	return IndexFold(content, "cfquery") >= 0 || IndexFold(content, "querynew") >= 0 || IndexFold(content, "queryexecute") >= 0
}

// startsAssignment reports whether a word after prev, which follows owner, can
// be the target of an assignment statement: it opens the text or a statement,
// the `>` of a tag, or follows `var`, a type after it, `else`, a condition's parenthesis, or the
// `local.` of a scoped name.
func startsAssignment(prev, owner Token) bool {
	switch prev.Kind {
	case TokSemicolon, TokLBrace, TokRBrace, TokRParen, TokDot, TokGT:
		return true
	case TokIdent:
		switch strings.ToLower(prev.Value) {
		case "var", "cfset", "set", "else", "final", "static":
			return true
		default:
			// `var query users = ...`
			return owner.Kind == TokIdent && strings.EqualFold(owner.Value, "var")
		}
	default:
		return prev == Token{}
	}
}

// blankSQLComments is sql with each `--` and `/* */` comment written over with
// blanks, newlines kept, so a word or a quote in a comment is read as neither.
// A string is stepped over whole.
func blankSQLComments(sql string) string {
	if !strings.Contains(sql, "--") && !strings.Contains(sql, "/*") {
		return sql
	}

	b := []byte(sql)

	blank := func(from, to int) {
		for i := from; i < to && i < len(b); i++ {
			if b[i] != '\n' {
				b[i] = ' '
			}
		}
	}

	for k := 0; k < len(b); k++ {
		switch {
		case b[k] == '\'' || b[k] == '"':
			k = skipSQLString(sql, k)
		case b[k] == '-' && k+1 < len(b) && b[k+1] == '-':
			end := strings.IndexByte(sql[k:], '\n')
			if end < 0 {
				end = len(b) - k
			}

			blank(k, k+end)

			k += end
		case b[k] == '/' && k+1 < len(b) && b[k+1] == '*':
			end := strings.Index(sql[k+2:], "*/")
			if end < 0 {
				end = len(b) - k - 2
			} else {
				end += 2
			}

			blank(k, k+2+end)

			k += 1 + end
		}
	}

	return string(b)
}

// queryTagColumns reports to consider the columns of each <cfquery> named name
// that is not in a skipped place: the query's SELECT, where it opens.
func queryTagColumns(content, name string, inSkip func(int) bool, consider func(int, []string, bool)) {
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

		v, has := tag.Attr("name")

		switch {
		case has && inSkip(tag.Start):
			opened = false
		case has && strings.Contains(v, "#"):
			// A name computed at run time may be any: what was known of the
			// variable before it no longer holds.
			consider(tag.Start, nil, false)

			opened = false
		case has && strings.EqualFold(unscoped(v), name):
			openAt, openEnd, opened = tag.Start, tag.End, true
		default:
			opened = false
		}
	}
}
