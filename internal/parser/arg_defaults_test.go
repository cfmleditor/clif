package parser

import "testing"

func defaultsOf(t *testing.T, src string) map[string]string {
	t.Helper()

	pr := Parse("file:///w/A.cfc", src)
	if len(pr.Funcs) != 1 {
		t.Fatalf("got %d functions, want 1 in:\n%s", len(pr.Funcs), src)
	}

	got := map[string]string{}

	for _ai := range pr.Funcs[0].Arguments {
		a := &pr.Funcs[0].Arguments[_ai]
		got[a.Name] = a.DefaultExpr()
	}

	return got
}

func TestScriptArgumentDefaultsAreKeptAsWritten(t *testing.T) {
	got := defaultsOf(t, `component {
	function f(
		a,
		string b = "x, y",
		numeric c = 1,
		d = [1, 2],
		struct e = {
			k: "v",
			n: 2
		},
		f = now(),
		required g,
		h = a.b("x,y"),
		i = "it""s",
		j = -1.5
	) {}
}`)

	want := map[string]string{
		"a": "", "b": `"x, y"`, "c": "1", "d": "[1, 2]", "e": `{ k: "v", n: 2 }`,
		"f": "now()", "g": "", "h": `a.b("x,y")`, "i": `"it""s"`, "j": "-1.5",
	}

	for name, w := range want {
		if got[name] != w {
			t.Errorf("argument %s: default = %q, want %q", name, got[name], w)
		}
	}
}

func TestTagArgumentDefaultsReadAsExpressions(t *testing.T) {
	got := defaultsOf(t, `<cfcomponent>
<cffunction name="f">
	<cfargument name="a">
	<cfargument name="n" type="numeric" default="3">
	<cfargument name="b" type="boolean" default="TRUE">
	<cfargument name="s" type="string" default="plain">
	<cfargument name="e" type="string" default="">
	<cfargument name="q" type="string" default="a ""q"" b">
	<cfargument name="d" default="#now()#">
	<cfargument name="w" default="#[1, 2]#">
	<cfargument name="m" default="id-#url.id#">
	<cfargument name="x" type="numeric" default="three">
</cffunction>
</cfcomponent>`)

	want := map[string]string{
		"a": "", "n": "3", "b": "true", "s": `"plain"`, "e": `""`, "q": `"a ""q"" b"`,
		"d": "now()", "w": "[1, 2]", "m": `"id-#url.id#"`, "x": `"three"`,
	}

	for name, w := range want {
		if got[name] != w {
			t.Errorf("argument %s: default = %q, want %q", name, got[name], w)
		}
	}
}

// A comment is not part of a default: written back by a stub it would swallow
// what follows it on the line.
func TestScriptArgumentDefaultsLeaveOutComments(t *testing.T) {
	got := defaultsOf(t, "component {\n\tfunction f(\n\t\ta = 1 // note\n\t\t, b = {\n\t\t\tk: 1, // c1\n\t\t\tn: 2 /* c2 */\n\t\t},\n\t\tc = \"http://x//y\",\n\t\td = /* first */ 5,\n\t\te = [1, /* two */ 2]\n\t) {}\n}")

	want := map[string]string{
		"a": "1", "b": "{ k: 1, n: 2 }", "c": `"http://x//y"`, "d": "5", "e": "[1, 2]",
	}

	for name, w := range want {
		if got[name] != w {
			t.Errorf("argument %s: default = %q, want %q", name, got[name], w)
		}
	}
}
