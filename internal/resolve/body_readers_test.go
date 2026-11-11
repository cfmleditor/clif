package resolve

import "testing"

// The resolver's last readers of CFML bodies read them as the parse does: a
// script loop, a lazy getter's shape, a Preside service's header and a line's
// local declaration from the scanner's tokens and parser.CFTags. Each read
// its text with a regular expression of its own, which took a commented-out
// statement or a word inside a string as code. The returns an element read
// counts are held to the same cases in TestASourceKnowsWhetherItHoldsAnElementReturn.
func TestTheBodyReadersReadOnlyCode(t *testing.T) {
	t.Run("script loop", func(t *testing.T) {
		src := "// loop collection=driverNames index=\"ghost\" item=\"ghostPath\" {\n" +
			"loop collection=driverNames index=\"name\" item=\"componentPath\" {\n" +
			"\t// a } in a comment does not close the body\n" +
			"\tdrivers[ name ] = createObject( \"component\", componentPath );\n" +
			"}\n"

		span, ok := listingLoopSpan(src, "driverNames", 0)
		if !ok || span.item != "componentPath" || span.index != "name" {
			t.Fatalf("listingLoopSpan = %+v, %v; want the live loop", span, ok)
		}

		if body := src[span.start:span.end]; !mentions(body, "createObject") {
			t.Fatalf("the loop's body %q stops before its write", body)
		}
	})

	t.Run("lazy getter", func(t *testing.T) {
		for _, c := range []struct{ name, body, want string }{
			{"tag", "<cfif not isObject(variables.mgr)><cfset loadMgr()></cfif><cfreturn variables.mgr />", "mgr"},
			{"tag, a commented-out return in the guard", "<cfif not isObject(variables.mgr)><!--- <cfreturn x> ---><cfset loadMgr()></cfif><cfreturn variables.mgr />", "mgr"},
			{"tag, a live return in the guard", "<cfif not isObject(variables.mgr)><cfreturn x></cfif><cfreturn variables.mgr />", ""},
			{"script", "if ( !isObject( variables.mgr ) ) { loadMgr(); } return variables.mgr;", "mgr"},
			{"script, a comment in the guard", "if ( !isObject( variables.mgr ) ) { // load, then return it\n loadMgr(); } return variables.mgr;", "mgr"},
			{"script, a brace in a string", "if ( not isObject( variables.mgr ) ) { loadMgr( \"{}\" ); } return variables.mgr;", "mgr"},
			{"script, a live return in the guard", "if ( !isObject( variables.mgr ) ) { return x; } return variables.mgr;", ""},
			{"script, another field", "if ( !isObject( variables.mgr ) ) { loadMgr(); } return variables.other;", ""},
		} {
			if got := lazyField(c.body); got != c.want {
				t.Errorf("%s: lazyField = %q, want %q", c.name, got, c.want)
			}
		}
	})

	t.Run("preside service", func(t *testing.T) {
		for _, c := range []struct {
			name, content string
			want          bool
		}{
			{"doc comment", "/**\n * @presideService true\n */\ncomponent { }", true},
			{"attribute", "component presideService=true { }", true},
			{"tag attribute", `<cfcomponent presideService="true"></cfcomponent>`, true},
			{"in a string", `component hint="not a presideService = thing" { }`, false},
			{"in a line comment", "// @presideService\ncomponent { }", false},
			{"in a commented-out tag", `<!--- <cfcomponent presideService="true"> ---><cfcomponent></cfcomponent>`, false},
			{"in the body", "component { /** @presideService */ function f() {} }", false},
			{"a longer annotation", "/** @presideServiceName x */\ncomponent { }", false},
		} {
			if got := wantsPresideInjection(c.content); got != c.want {
				t.Errorf("%s: wantsPresideInjection = %v, want %v", c.name, got, c.want)
			}
		}
	})

	t.Run("local declaration", func(t *testing.T) {
		for _, c := range []struct {
			line string
			want bool
		}{
			{"var svc = getSvc();", true},
			{"local.svc = getSvc();", true},
			{`<cfset var svc = getSvc()>`, true},
			{`svc = getSvc( "var x" );`, false},
			{"svc = getSvc(); // var was here", false},
			{"svc = this.local.x;", false},
		} {
			if got := lineDeclaresLocal(c.line); got != c.want {
				t.Errorf("%q: lineDeclaresLocal = %v, want %v", c.line, got, c.want)
			}
		}
	})
}
