package parser

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.lsp.dev/uri"
)

// assignmentsText is each assignment as `target op value`, the way the
// cases below spell what they want, with ^ before a write that starts its
// statement, var before a local one and the tag it was read from after it.
func assignmentsText(content string, as []Assignment) []string {
	var out []string

	for i := range as {
		a := &as[i]

		var b strings.Builder

		if a.Op != AssignAttribute && a.StartsStatement() {
			b.WriteByte('^')
		}

		if a.Local {
			b.WriteString("var ")
		}

		b.WriteString(a.Target)

		switch a.Op {
		case AssignVar:
			b.WriteString(" (var)")
		case AssignForIn:
			b.WriteString(" (for-in)")
		case AssignAttribute:
			fmt.Fprintf(&b, " (attribute) %q", a.Literal)
		default:
			fmt.Fprintf(&b, " %s %s", a.Operator, a.ValueText(content))
		}

		if a.Tag != "" {
			fmt.Fprintf(&b, " <%s>", a.Tag)
		}

		out = append(out, b.String())
	}

	return out
}

// TestAssignmentsAreWhatTheParseReads holds the parse's reading of writes to
// each shape: what a statement writes, and not what an argument list, a
// struct literal, a script-syntax tag's attributes, a comment, a string or
// markup only spells like a write; where a value ends; and which writes
// start their statement.
func TestAssignmentsAreWhatTheParseReads(t *testing.T) {
	for _, c := range []struct {
		name, file, text string
		want             []string
	}{
		{
			"the operators", "t.cfc",
			"component {\n function f() {\n  var a = 1; b += 2; c++; d &= \"x\"; e == 1; var g;\n }\n}\n",
			[]string{"var a = 1", "^b += 2", "^c ++ ", "^d &= \"x\"", "var g (var)"},
		},
		{
			"targets as written", "t.cfc",
			"component {\n function f( rc ) {\n  arguments.rc.user = getUser();\n  local.z[ k ].y = 2;\n  variables\n   .a\n   .b = 3;\n }\n}\n",
			[]string{"^arguments.rc.user = getUser()", "^local.z[ k ].y = 2", "^variables.a.b = 3"},
		},
		{
			"for-in loops", "t.cfc",
			"component {\n function f() {\n  for ( var k in s ) {}\n  for ( x in s ) {}\n  for ( i = 1; i <= 3; i++ ) {}\n }\n}\n",
			[]string{"var k (for-in)", "x (for-in)", "i = 1", "^i ++ "},
		},
		{
			"a value ends with its statement", "t.cfc",
			"component {\n function f() {\n  a = new A()\n  b = c\n   .d()\n  e = 1 // note\n  f = \"g;h\"; i = [ 1,\n 2 ]\n }\n}\n",
			[]string{"^a = new A()", "^b = c\n   .d()", "^e = 1", "^f = \"g;h\"", "^i = [ 1,\n 2 ]"},
		},
		{
			"arguments and literals are not statements", "t.cfc",
			"component {\n function f() {\n  svc.save( force = true, x = 1 );\n  s = { a = 1, b = { c = 2 } };\n  g( function() { inner = 1; } );\n  t = ( u = 1 );\n }\n}\n",
			[]string{"^s = { a = 1, b = { c = 2 } }", "^inner = 1", "^t = ( u = 1 )"},
		},
		{
			"a named argument that names a variable", "t.cfc",
			"component {\n function f() {\n  cfhttp( url = u, result = \"r\" );\n }\n}\n",
			[]string{"result (attribute) \"r\" <cfhttp>"},
		},
		{
			"a script-syntax tag's attributes", "t.cfc",
			"component extends=\"Base\" accessors=true {\n property name=\"svc\" inject=\"Svc\";\n function f() {\n  param name=\"x\" default=\"1\";\n  lock name=\"l\"\n   timeout=\"5\" {\n   y = 1;\n  }\n }\n}\n",
			[]string{"name (attribute) \"svc\" <property>", "name (attribute) \"x\" <param>", "name (attribute) \"l\" <lock>", "^y = 1"},
		},
		{
			"a governed statement starts at its first word", "t.cfc",
			"component {\n function f() {\n  if ( c ) a = 1; else b = 2;\n  switch ( x ) {\n   case \"y\": { d = 3; }\n  }\n  e = f = 4;\n }\n}\n",
			[]string{"^a = 1", "^b = 2", "^d = 3", "^e = f = 4", "f = 4"},
		},
		{
			"comments and strings write nothing", "t.cfc",
			"component {\n function f() {\n  // a = 1;\n  /* b = 2; */\n  s = \"c = 3 #d = 4#\";\n }\n}\n",
			[]string{"^s = \"c = 3 #d = 4#\""},
		},
		{
			"tags", "t.cfm",
			"<cfset prc.x = 1>\n<cfset var y = \"s\" />\n<!--- <cfset z = 1> --->\n<cfloop index=\"i\" from=\"1\" to=\"2\"></cfloop>\n<cfquery name=\"q\">select a = 1</cfquery>\n<input name=\"h\" value=\"v\">\n<cfif x = 1></cfif>\n",
			[]string{"^prc.x = 1 <cfset>", "var y = \"s\" <cfset>", "index (attribute) \"i\" <cfloop>", "name (attribute) \"q\" <cfquery>"},
		},
		{
			"script islands in a template", "t.cfm",
			"<cfoutput>#a = 1#</cfoutput>\n<cfscript>\nb = 2;\n</cfscript>\n<script>c = 3;</script>\n",
			[]string{"^b = 2"},
		},
		{
			"SQL in a script query is no code", "t.cfc",
			"component {\n function f() {\n  q = queryExecute( ```\n   UPDATE t SET a = 1\n  ``` );\n }\n}\n",
			[]string{"^q = queryExecute( ```\n   UPDATE t SET a = 1\n  ``` )"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := assignmentsText(c.text, AssignmentsOf(c.file, c.text))
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got  %q\nwant %q", got, c.want)
			}
		})
	}
}

// TestAssignmentsOfAgreesWithTheParse: the assignments-only reading, a parse
// that recorded them as it ran and a parse asked afterwards answer alike over
// the fixtures; an edit is read again; and a parse records nothing unless it
// is asked to.
func TestAssignmentsOfAgreesWithTheParse(t *testing.T) {
	n := 0

	err := filepath.WalkDir("../../testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".cfc") && !strings.HasSuffix(path, ".cfm") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		content := string(data)
		file := "file:///x/" + filepath.ToSlash(path)

		plain := Parse(uri.URI(file), content)
		if plain.assignments != nil {
			t.Errorf("%s: a parse not asked to record assignments recorded them", path)
		}

		recorded := ParseWithOptions(uri.URI(file), content, &ParseOptions{RecordAssignments: true, ExtractCalls: true, ScanAllScopes: true})
		if recorded.assignments == nil {
			t.Errorf("%s: RecordAssignments recorded nothing", path)
		}

		light := AssignmentsOf(file, content)
		if got := plain.Assignments(); !reflect.DeepEqual(got, light) {
			t.Errorf("%s: a parse's Assignments = %v\nAssignmentsOf = %v", path, assignmentsText(content, got), assignmentsText(content, light))
		}

		if got := recorded.Assignments(); !reflect.DeepEqual(got, light) {
			t.Errorf("%s: recorded = %v\nAssignmentsOf = %v", path, assignmentsText(content, got), assignmentsText(content, light))
		}

		n++

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if n == 0 {
		t.Fatal("found no fixtures under ../../testdata")
	}

	src := "component {\n\tfunction a() {\n\t\tx = 1;\n\t}\n}\n"
	pr := ParseWithOptions("file:///t/E.cfc", src, &ParseOptions{RecordAssignments: true})
	pr.ApplyEdit(2, 2, 2, 2, "y = 2; ")

	if got, want := assignmentsText(pr.Content, pr.Assignments()), []string{"^y = 2", "^x = 1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("after an edit, %q, want %q", got, want)
	}
}

// TestValueLineIsTheValueWrittenAsOneLine: a value over several lines is read
// as one, its lines joined by a space, a chain's `.` by nothing, a comment
// left out, and a string's text as written; cut at a limit, it is the part
// before it.
func TestValueLineIsTheValueWrittenAsOneLine(t *testing.T) {
	const content = "x = svc.get(\n\t1, // one\n\t2 )\n\t.run();\ny = \"a\nb\";\nz = f( 1 );\n"

	as := AssignmentsOf("a.cfc", content)
	if len(as) != 3 {
		t.Fatalf("%d assignments", len(as))
	}

	for i, want := range []string{"svc.get( 1, 2 ).run()", "\"a\nb\"", "f( 1 )"} {
		if got := as[i].ValueLine(content, len(content)); got != want {
			t.Errorf("%s: %q, want %q", as[i].Target, got, want)
		}
	}

	if got := as[0].ValueLine(content, strings.Index(content, "\n\t2")); got != "svc.get( 1," {
		t.Errorf("cut: %q", got)
	}
}
