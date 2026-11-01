package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestAConditionalAssignmentHoldsItsBranches: `x = c ? a : b` and `x = a ?: b`
// left x untyped whatever the branches held (#308). Each branch is typed as an
// assignment of its own and x is their union; a branch holding an untyped
// argument types nothing.
func TestAConditionalAssignmentHoldsItsBranches(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"shared/Svc.cfc":  `component { function search(){} function onlySvc(){} }`,
		"shared/Fake.cfc": `component { function search(){} }`,
		"Client.cfc": `component {
function typedArg( shared.Svc injected ) {
	variables.c1 = isObject( arguments.injected ) ? arguments.injected : new shared.Svc();
	variables.c1.search();
}
function untypedArg( any injected ) {
	var c2 = arguments.injected ?: new shared.Svc();
	c2.search();
}
function twoComponents( flag ) {
	var c3 = flag ? new shared.Svc() : ( new shared.Fake() );
	c3.search();
	c3.neither();
}
function selfScoped() {
	variables.c6 = isNull( variables.c6 ) ? new shared.Svc() : variables.c6;
	c6.onlySvc();
}
function selfUnscoped() {
	variables.c7 = isNull( c7 ) ? new shared.Svc() : c7;
	variables.c7.onlySvc();
}
function nullFallback( flag ) {
	var c8 = flag ? new shared.Svc() : javacast( "null", "" );
	c8.onlySvc();
}
function overLines( flag ) {
	var c9 = flag
		? new shared.Svc()
		: new shared.Fake();
	c9.neither();
}
function nullLiteral( flag ) {
	var c11 = flag ? new shared.Svc() : null;
	c11.onlySvc();
	var c12 = flag ? nullValue() : new shared.Svc();
	c12.onlySvc();
}
function commented( flag ) {
	var c14 = flag ? new shared.Svc() : new shared.Fake(); // pick?
	c14.neither();
	var c15 = flag ? new shared.Svc() : new shared.Fake(); // see (docs
	c15.neither();
}
function reassignedAfter( flag ) {
	var a = new shared.Svc();
	var c10 = flag ? a : new shared.Fake();
	a = new shared.Fake();
	c10.onlySvc();
}
function nested( a, b ) {
	var c4 = a ? new shared.Svc() : b ? createObject( "component", "shared.Svc" ) : new shared.Svc();
	c4.search();
}
}`,
		"Tags.cfc": `<cfcomponent>
<cffunction name="f">
	<cfset var c5 = true ? new shared.Svc() : new shared.Svc()>
	<cfset c5.search()>
</cffunction>
</cfcomponent>`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Client.cfc"), map[string]string{
		"variables.c1.search":  "",
		"c2.search":            "variable 'c2' has no component ref",
		"c3.search":            "",
		"c3.neither":           "method 'neither' not found in shared.Fake|shared.Svc",
		"c4.search":            "",
		"c6.onlySvc":           "",
		"variables.c7.onlySvc": "",
		"c8.onlySvc":           "",
		"c9.neither":           "method 'neither' not found in shared.Fake|shared.Svc",
		"c10.onlySvc":          "",
		"c11.onlySvc":          "",
		"c12.onlySvc":          "",
		"c14.neither":          "method 'neither' not found in shared.Fake|shared.Svc",
		"c15.neither":          "method 'neither' not found in shared.Fake|shared.Svc",
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Tags.cfc"), map[string]string{
		"c5.search": "",
	})
}

func TestTernaryBranches(t *testing.T) {
	for rhs, want := range map[string][]string{
		"c ? a : b":                       {"a", "b"},
		"a ?: b":                          {"a", "b"},
		"f( x ? 1 : 2 ) ? a : b":          {"a", "b"},
		"c ? ( d ? a : b ) : e":           {"a", "b", "e"},
		"c ? { k: 1 } : b":                {"{ k: 1 }", "b"},
		"svc.get( c ? 1 : 2 )":            nil,
		"new X()":                         nil,
		"c ? a":                           nil,
		"a ?: javacast( \"null\", \"\" )": {"a", "javacast( \"null\", \"\" )"},
	} {
		got, ok := ternaryBranches(rhs)
		if ok != (want != nil) || len(got) != len(want) {
			t.Errorf("%q: got %q (%v), want %q", rhs, got, ok, want)

			continue
		}

		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%q: got %q, want %q", rhs, got, want)
			}
		}
	}
}

// TestANullFallbackIsDroppedWhereverItIs: the null fallback need not be the
// value's first conditional.
func TestANullFallbackIsDroppedWhereverItIs(t *testing.T) {
	value := significantTokens(`a ? b : c ?: javacast( "null", "" )`)
	if got := producerText(withoutNullFallback(value)); got != "a ? b : c" {
		t.Errorf("got %q", got)
	}
}

// TestAnElvisReturnIsTheWeakerOfItsOperands: `x ?: ""` is x or "", not the
// `:` after the `?`.
func TestAnElvisReturnIsTheWeakerOfItsOperands(t *testing.T) {
	tokens := significantTokens(`"a" ?: "";`)
	if got := returnKind(tokens, 0, len(tokens), &parser.ParseResult{}); got != retLiteral {
		t.Errorf("returnKind = %d, want literal", got)
	}
}

// TestACommentIsNotAConditional: a trailing comment ending in `?` or `:` does
// not carry the assignment on to the next line; a conditional's `?` does.
func TestACommentIsNotAConditional(t *testing.T) {
	for rhs, more := range map[string]bool{
		`new Query(); // which datasource?`: false,
		`new Query(); // note:`:             false,
		`"http://x?"`:                       false,
		`flag ?`:                            true,
		`flag ? new A() : // either:`:       true,
		`"#fmt("a//b")#" ?`:                 true,
		`a; /* it's */ // why?`:             false,
	} {
		if _, got := joinAssignmentLine(rhs, "q.execute(", false); got != more {
			t.Errorf("%s: joined %v, want %v", rhs, got, more)
		}
	}

	// A chain or an open call joins its next line after the comment, not
	// into it.
	for rhs, want := range map[string]string{
		`$.getBean( "feed" ) // the feed`: `$.getBean( "feed" ).getIterator()`,
		`f( // first`:                     `f( .getIterator()`,
	} {
		if got, _ := joinAssignmentLine(rhs, ".getIterator()", false); got != want {
			t.Errorf("%s: joined %q, want %q", rhs, got, want)
		}
	}
}

// TestWithoutLineCommentStepsOverStrings: a `//` in a string, in a string
// inside an interpolation, or after a block comment holding a quote is read
// as CFScript reads it.
func TestWithoutLineCommentStepsOverStrings(t *testing.T) {
	for s, want := range map[string]string{
		`a // c`:               `a `,
		`"http://x" // c`:      `"http://x" `,
		`"say ""//"" ok" // c`: `"say ""//"" ok" `,
		`"#fmt("a//b")#" // c`: `"#fmt("a//b")#" `,
		`a /* it's */ // c`:    `a /* it's */ `,
		`"## //" // c`:         `"## //" `,
		`'#f( "x" )#' // c`:    `'#f( "x" )#' `,
		`"# Report" // ready?`: `"# Report" `,
	} {
		if got := withoutLineComment(s); got != want {
			t.Errorf("%s: %q, want %q", s, got, want)
		}
	}
}
