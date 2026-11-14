package parser

import (
	"slices"
	"strings"
	"testing"
)

func TestSQLSelectColumns(t *testing.T) {
	for _, c := range []struct {
		name, sql string
		want      []string
		known     bool
	}{
		{"plain", "SELECT id, name FROM users", []string{"id", "name"}, true},
		{"newlines and case", "\n  select\n   ID,\n   Name\n  from users\n", []string{"ID", "Name"}, true},
		{"table dropped", "SELECT u.id, u.name FROM users u", []string{"id", "name"}, true},
		{"aliases", "SELECT u.id AS userId, count(*) total, upper(name) as NAME2 FROM users u", []string{"userId", "total", "NAME2"}, true},
		{"a call with no alias has none", "SELECT id, count(*), name FROM t", []string{"id", "name"}, true},
		{"nested calls and commas", "SELECT coalesce(a, b, 'x,y') AS c, d FROM t", []string{"c", "d"}, true},
		{"a subquery with its own from", "SELECT id, (SELECT max(x) FROM o WHERE o.u = u.id) AS latest FROM u", []string{"id", "latest"}, true},
		{"distinct and top", "SELECT DISTINCT TOP 10 id, name FROM t", []string{"id", "name"}, true},
		{"top in parentheses", "SELECT TOP (5) id FROM t", []string{"id"}, true},
		{"a quoted alias with a blank", `SELECT u.id AS "user id", name FROM u`, []string{"user id", "name"}, true},
		{"quoted names", `SELECT "first name", [last], ` + "`age`" + ` FROM t`, []string{"first name", "last", "age"}, true},
		{"a star", "SELECT * FROM t", nil, false},
		{"a table star", "SELECT u.*, o.id FROM u, o", nil, false},
		{"no from", "SELECT 1 AS one, 2 AS two", []string{"one", "two"}, true},
		{"not a select", "UPDATE t SET a = 1", nil, false},
		{"a word that holds from", "SELECT fromage, id FROM t", []string{"fromage", "id"}, true},
		{"a string holding from", "SELECT 'a from b' AS s, id FROM t", []string{"s", "id"}, true},
		{"an operator before the last word means no alias", "SELECT a + b, a || b, price * qty total, id FROM t", []string{"total", "id"}, true},
		{"a CASE with no alias has none", "SELECT CASE WHEN a = 1 THEN 'x' ELSE 'y' END, CASE WHEN a THEN 1 END AS flag, id FROM t", []string{"flag", "id"}, true},
		{"an alias after a CF tag", "SELECT id, <cfif x>a<cfelse>b</cfif> total FROM t", []string{"id", "total"}, true},
		{"a CF expression is left out", "SELECT id, #cols# FROM t", []string{"id"}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, known := sqlSelectColumns(c.sql)
			if known != c.known || !slices.Equal(got, c.want) {
				t.Errorf("%q: got %q %v, want %q %v", c.sql, got, known, c.want, c.known)
			}
		})
	}
}

func TestQueryColumns(t *testing.T) {
	end := func(s string) int { return len(s) }

	for _, c := range []struct {
		name, src, query string
		want             []string
		ok               bool
	}{
		{"a cfquery", "<cfquery name=\"users\" datasource=\"d\">\n SELECT id, name FROM users\n</cfquery>\n<cfset x = 1>", "users", []string{"id", "name"}, true},
		{"the name is not case sensitive", "<cfquery name=\"Users\">SELECT id FROM u</cfquery>", "USERS", []string{"id"}, true},
		{"a scoped name", "<cfquery name=\"local.rows\">SELECT id FROM t</cfquery>", "rows", []string{"id"}, true},
		{"another query", "<cfquery name=\"a\">SELECT id FROM u</cfquery>", "b", nil, false},
		{"the last declaration wins", "<cfquery name=\"q\">SELECT a FROM t</cfquery><cfquery name=\"q\">SELECT b, c FROM t</cfquery>", "q", []string{"b", "c"}, true},
		{"queryNew", `<cfset q = queryNew( "id,name,email", "integer,varchar,varchar" )>`, "q", []string{"id", "name", "email"}, true},
		{"queryNew in script", "function f() {\n var q = queryNew(\"a, b\");\n q.\n}", "q", []string{"a", "b"}, true},
		{"queryExecute", `var rs = queryExecute( "SELECT id, title FROM posts WHERE id = :id", { id = 1 } );`, "rs", []string{"id", "title"}, true},
		{"queryExecute with a star", `rs = queryExecute( "SELECT * FROM posts" );`, "rs", nil, false},
		{"queryExecute built from a variable", `rs = queryExecute( sqlText );`, "rs", nil, false},
		{"a select that is all hash expression", `rs = queryExecute( "SELECT #cols# FROM t" );`, "rs", nil, false},
		{"a hash expression beside known columns", `rs = queryExecute( "SELECT id, #cols#, name FROM t" );`, "rs", []string{"id", "name"}, true},
		{"a commented declaration", "// q = queryNew(\"a\");\n/* q = queryNew(\"b\") */\nq = queryNew(\"c\");", "q", []string{"c"}, true},
		{"a star is not known", "<cfquery name=\"q\">SELECT * FROM t</cfquery>", "q", nil, false},
		{"a cfquery that never closes", "<cfquery name=\"q\">SELECT id FROM t", "q", nil, false},
		{"not a query", "x = 1; y = arrayNew(1);", "x", nil, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := QueryColumns(c.src, c.query, end(c.src))
			if ok != c.ok || !slices.Equal(got, c.want) {
				t.Errorf("got %q %v, want %q %v", got, ok, c.want, c.ok)
			}
		})
	}
}

// A declaration after the cursor is not one yet.
func TestQueryColumnsStopAtTheCursor(t *testing.T) {
	src := "q = queryNew(\"a\");\n x = q.\n q = queryNew(\"b\");"
	cursor := strings.Index(src, "x = q.") + len("x = q.")

	if got, ok := QueryColumns(src, "q", cursor); !ok || !slices.Equal(got, []string{"a"}) {
		t.Errorf("before the second: %q %v, want [a]", got, ok)
	}

	if _, ok := QueryColumns(src, "q", 0); ok {
		t.Error("a declaration is found before the start")
	}
}
