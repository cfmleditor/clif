package resolve

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// lineHolding is the 0-based line of text holding sub.
func lineHolding(t *testing.T, text, sub string) int {
	t.Helper()

	i := strings.Index(text, sub)
	if i < 0 {
		t.Fatalf("%q is not in the text", sub)
	}

	return strings.Count(text[:i], "\n")
}

// TestTheWriteReadersReadOnlyCode holds every reader that asks whether code
// writes or guards a name to the code: a write or a guard in a comment, or
// spelled inside a string, is none, and the real one beside it is still
// read. Each was a regular expression built per name over the text, and
// answered otherwise.
func TestTheWriteReadersReadOnlyCode(t *testing.T) {
	cfc := func(text string) *parser.ParseResult { return parser.Parse("file:///t/C.cfc", text) }

	for _, c := range []struct {
		name string
		read func(t *testing.T, text string) string
		text string
		want string
	}{
		{
			"setsName, commented out and quoted", func(_ *testing.T, text string) string { return strconv.FormatBool(setsName(text, "x")) },
			"<cfscript>\n// x = 1;\n/* var x; for (x in a) {} */\ns = \"x = 2\";\n</cfscript>\n<!--- <cfloop item=\"x\" array=\"#a#\"> --->\n",
			"false",
		},
		{
			"setsName", func(_ *testing.T, text string) string { return strconv.FormatBool(setsName(text, "x")) },
			"<cfscript>\n// x = 1;\ns = \"x = 2\";\n</cfscript>\n<cfset x = 3>\n",
			"true",
		},
		{
			"membersWrittenOn, commented out and quoted", func(t *testing.T, text string) string {
				t.Helper()

				return strconv.FormatBool((&Resolver{}).membersWrittenOn(cfc(text), uint32(lineHolding(t, text, "item.go")), "item"))
			},
			"component {\n function f() {\n  // item.extra = 1;\n  s = 'item.extra = 2';\n  item.go();\n }\n}\n",
			"false",
		},
		{
			"membersWrittenOn", func(t *testing.T, text string) string {
				t.Helper()

				return strconv.FormatBool((&Resolver{}).membersWrittenOn(cfc(text), uint32(lineHolding(t, text, "item.go")), "item"))
			},
			"component {\n function f() {\n  // item.extra = 1;\n  item[ \"extra\" ] = 3;\n  item.go();\n }\n}\n",
			"true",
		},
		{
			"fw1Assigns, commented out and quoted", readFW1Assigns,
			"component {\n function default( rc ) {\n  // rc.user = 1;\n  var s = \"rc.user = 2\";\n }\n}\n",
			"false",
		},
		{
			"fw1Assigns", readFW1Assigns,
			"component {\n function default( rc ) {\n  // rc.user = 1;\n  arguments.rc.user = getUser();\n }\n}\n",
			"true",
		},
		{
			"throughAssigns, commented out and quoted", readThroughAssigns,
			"<!--- <cfset prc.user = 1> --->\n<cfoutput>#prc.other# <a title=\"prc.user = 2\"></a></cfoutput>\n",
			"false",
		},
		{
			"throughAssigns", readThroughAssigns,
			"<!--- <cfset prc.user = 1> --->\n<cfset prc.user = getUser()>\n",
			"true",
		},
		{
			"the last prc.x written, one in a block comment after it", func(_ *testing.T, text string) string {
				rhs, line, ok := lastMemberAssignmentAt(text, "prc", "x")

				return fmt.Sprint(rhs, "@", line, " ", ok)
			},
			"prc.x = new A();\n/*\n\tprc.x = new B();\n*/\n",
			"new A()@0 true",
		},
		{
			"an exists guard, commented out", readExistsGuard,
			"component {\n function f() {\n  // if ( structKeyExists( variables.config, \"onShutdown\" ) ) {\n  if ( true ) {\n   variables.config.onShutdown( this );\n  }\n }\n}\n",
			"false",
		},
		{
			"an exists guard", readExistsGuard,
			"component {\n function f() {\n  // if ( true ) {\n  if ( structKeyExists( variables.config, \"onShutdown\" ) ) {\n   variables.config.onShutdown( this );\n  }\n }\n}\n",
			"true",
		},
		{
			"an instance guard, commented out", readInstanceGuard,
			"component {\n function f() {\n  // if ( isInstanceOf( arguments.event, \"mura.Old\" ) ) {\n  if ( isInstanceOf( arguments.event, \"mura.MuraScope\" ) ) {\n   arguments.event.event( \"x\" );\n  }\n }\n}\n",
			"[mura.MuraScope]",
		},
		{
			"initStoredArg, beside a commented-out write", func(_ *testing.T, text string) string { return initStoredArg(cfc(text), "helpers") },
			"component {\n function init( any helpers ) {\n  variables.helpers = arguments.helpers;\n  return this;\n }\n function other() {\n  // variables.helpers = 'x';\n  var s = \"variables.helpers = 1\";\n }\n}\n",
			"helpers",
		},
		{
			"quotedNameLines, beside a commented-out construction", readQuotedNameLines,
			"component {\n // x = createObject( \"component\", \"services.Templates\" );\n function f() {\n  return createObject( \"component\", \"services.Templates\" );\n }\n}\n",
			"[3]",
		},
		{
			"a lazy field's writes, beside commented-out and quoted ones", readFieldWrites,
			"component {\n function getSvc() {\n  if ( !isObject( variables.svc ) ) {\n   variables.svc = new A();\n  }\n  return variables.svc;\n }\n // variables.svc = new B();\n log = \"svc = new B()\";\n}\n",
			"A.cfc",
		},
		{
			"a registry method guard, commented out", readRegistryGuard,
			"<cfscript>\n// if ( structKeyExists( driver, \"getBundleName\" ) )\nx = driver.getBundleName();\n</cfscript>\n",
			"false",
		},
		{
			"a registry method guard", readRegistryGuard,
			"<cfscript>\n// if ( !isNull( driver.other ) )\nx = isNull( driver.getBundleName ) ? \"\" : driver.getBundleName();\n</cfscript>\n",
			"true",
		},
		{
			"a subclass's literal, beside a commented-out one", readLeafLiteral,
			"component extends=\"Base\" {\n // variables.entityPlural = \"old\";\n variables.entityPlural = \"authors\";\n}\n",
			"authors",
		},
		{
			"a ColdBox test's model, beside a commented-out one", readTestSubject,
			"/* component model=\"Wrong\" */\ncomponent extends=\"coldbox.system.testing.BaseModelTest\" model=\"Right\" {\n}\n",
			"Right.cfc",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.read(t, c.text); got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}

func readFW1Assigns(t *testing.T, text string) string {
	t.Helper()

	pr := parser.Parse("file:///t/C.cfc", text)

	scope, ok := funcScope(pr, "default")
	if !ok {
		t.Fatal("no default()")
	}

	return strconv.FormatBool(fw1Assigns(pr, scope, "user"))
}

func readThroughAssigns(t *testing.T, text string) string {
	t.Helper()

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"v.cfm": text})

	return strconv.FormatBool((&Resolver{FS: vfs.OS{}, Index: index.New()}).throughAssigns([]string{filepath.Join(dir, "v.cfm")}, "user"))
}

func readExistsGuard(t *testing.T, text string) string {
	t.Helper()

	return strconv.FormatBool(guardedByExistsCheck(parser.Parse("file:///t/C.cfc", text), "variables.config", "onShutdown", lineHolding(t, text, "config.onShutdown(")))
}

func readInstanceGuard(t *testing.T, text string) string {
	t.Helper()

	return fmt.Sprint(guardedInstanceOf(parser.Parse("file:///t/C.cfc", text), "arguments.event", "event", lineHolding(t, text, "event.event(")))
}

func readQuotedNameLines(_ *testing.T, text string) string {
	return fmt.Sprint(quotedNameLines(parser.Parse("file:///t/C.cfc", text), "Templates"))
}

func readFieldWrites(t *testing.T, text string) string {
	t.Helper()

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A.cfc": "component {}", "B.cfc": "component {}"})

	return filepath.Base((&Resolver{FS: vfs.OS{}, Index: index.New()}).fieldWrites(text, "svc", dir))
}

func readRegistryGuard(t *testing.T, text string) string {
	t.Helper()

	call := &parser.CallSite{Line: uint32(lineHolding(t, text, "x = "))}

	return strconv.FormatBool((&Resolver{}).methodExistenceGuarded(parser.Parse("file:///t/t.cfm", text), call, "getBundleName"))
}

func readLeafLiteral(t *testing.T, text string) string {
	t.Helper()

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Leaf.cfc": text, "Base.cfc": "component {}"})

	r := &Resolver{FS: vfs.OS{}, Index: index.New()}

	return r.leafLiteral(filepath.Join(dir, "Leaf.cfc"), filepath.Join(dir, "Base.cfc"), "entityPlural")
}

func readTestSubject(t *testing.T, text string) string {
	t.Helper()

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Spec.cfc": text, "Right.cfc": "component {}", "Wrong.cfc": "component {}"})

	path := filepath.Join(dir, "Spec.cfc")
	r := &Resolver{FS: vfs.OS{}, Index: index.New()}

	return filepath.Base(r.coldboxTestSubject("variables.model", parser.Parse(cfpath.ToURI(path), text)))
}

// TestSetsNameOnItsShapes: what setsName takes for a write or a declaration of
// a name, one shape at a time: the shapes the regular expression it replaced
// took, read as tokens, so that a name spelled with spaces around its dot is
// the same name, and one in a comment or a string is none, though one in a
// #...# expression inside a string is code.
func TestSetsNameOnItsShapes(t *testing.T) {
	for _, tt := range []struct {
		text, name string
		want       bool
	}{
		{"x = 1", "x", true},
		{"x=1", "x", true},
		{"x += 1", "x", true},
		{"x&=a", "x", true},
		{"x++", "x", true},
		{"x --", "x", true},
		{"x == 1", "x", false},
		{"x\v= 1", "x", false},
		{"x === 1", "x", false},
		{"x != 1", "x", false},
		{"x <= 1", "x", false},
		{"x =", "x", false},
		{"x +=", "x", false},
		{"x +==1", "x", false},
		{"a.x = 1", "x", false},
		{"$x = 1", "x", false},
		{"_x = 1", "x", false},
		{"(x = 1", "x", true},
		{"local.x = 1", "x", true},
		{"VARIABLES.x = 1", "x", true},
		{"local . x = 1", "x", true},
		{"alocal.x = 1", "x", false},
		{"this.x = 1", "x", false},
		{"a.local.x = 1", "x", false},
		{"var x", "x", true},
		{"VAR\tx;", "x", true},
		{"var xy", "x", false},
		{"var x_", "x", false},
		{"avar x", "x", false},
		{"for (x in a)", "x", true},
		{"for(var x in a)", "x", true},
		{"FOR ( VAR\nx\nIN a", "x", true},
		{"for (var x of a)", "x", true}, // `var x` is a declaration whatever follows
		{"for (x int a)", "x", false},
		{"xfor (x in a)", "x", false},
		{"for (xx in a)", "x", false},
		{"for (var in a)", "x", false},
		{"item=\"x\"", "x", true},
		{"returnVariable = 'x'", "x", true},
		{"<cfloop index=\"x\" from=\"1\">", "x", true},
		{"variable=\"x\"", "x", true},
		{"avariable=\"x\"", "x", false},
		{"names=\"x\"", "x", false},
		{"item=\"xy\"", "x", false},
		{"result==\"x\"", "x", false},
		{"x = 1", "arguments.x", false},
		{"arguments.x = 1", "arguments.x", true},
		{"a.arguments.x = 1", "arguments.x", false},
		{"// x = 1", "x", false},
		{"/* var x */", "x", false},
		{"s = \"x = 1\"", "x", false},
		{"<!--- <cfset x = 1> --->", "x", false},
		{"s = \"#x = 1#\"", "x", true},
	} {
		if got := setsName(tt.text, tt.name); got != tt.want {
			t.Errorf("setsName(%q, %q) = %v, want %v", tt.text, tt.name, got, tt.want)
		}
	}
}

// TestAWriteValueEndsWithItsStatement: the value a write assigns runs to its
// statement's end, a ; or a <cfset>'s >, over the lines its brackets hold
// open, and not past a comment.
func TestAWriteValueEndsWithItsStatement(t *testing.T) {
	for _, tt := range []struct{ text, want string }{
		{"x = new A(); y = 2;", "new A()"},
		{"<cfset x = new A() />", "new A()"},
		{"<cfset x = f( a )>rest", "f( a )"},
		{"x = new A(\n\tb = 1\n);", "new A(\n\tb = 1\n)"},
		{"x = a // note\ny = 2", "a"},
		{"x = \"a;b\";", "\"a;b\""},
		{"if ( c ) { x = 1 }", "1"},
		{"x = 1\ny = 2", "1"},
	} {
		writes := codeWrites(codeOf(tt.text), "x", writeRule{})
		if len(writes) != 1 {
			t.Errorf("%q: %d writes, want 1", tt.text, len(writes))

			continue
		}

		if got := tokensText(tt.text, writes[0].value()); got != tt.want {
			t.Errorf("%q: value %q, want %q", tt.text, got, tt.want)
		}
	}
}
