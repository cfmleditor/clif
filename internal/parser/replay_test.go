package parser

import (
	"slices"
	"testing"
)

// TestReplayReadsWhatTheScannerReads: a replaying scanner hands back the
// tokens a live one scans, peeks as it does, and restores to a token's offset
// (the token just read, the one in front, or any other) as it does. The
// passes after a parse read a region's tokens this way, and a difference
// would change what they find without any test of theirs noticing.
func TestReplayReadsWhatTheScannerReads(t *testing.T) {
	src := "component {\n\tfunction f( a ) {\n\t\t/* c */ var x = a.b[ 1 ].c( \"s #g(\"q\")# t\" ); // n\n\t\treturn x;\n\t}\n}\n"
	toks := tokenize(src)
	replay := newReplayScanner(src, &toks)

	live := NewScanner(src)
	live.interpStrings = true

	for i := 0; ; i++ {
		if lp, rp := live.PeekSkipComments(), replay.PeekSkipComments(); lp != rp {
			t.Fatalf("peek %d: live %+v, replay %+v", i, lp, rp)
		}

		lt, rt := live.NextSkipComments(), replay.NextSkipComments()
		if lt != rt {
			t.Fatalf("token %d: live %+v, replay %+v", i, lt, rt)
		}

		if lt.Kind == TokEOF {
			break
		}
	}

	for _, back := range []int{0, 5, len(toks) - 3, len(toks) - 2} {
		at := ScannerState{pos: toks[back].Offset, line: toks[back].Line}
		live.Restore(at)
		replay.Restore(at)

		if lt, rt := live.NextSkipComments(), replay.NextSkipComments(); lt != rt {
			t.Errorf("after restoring to token %d: live %+v, replay %+v", back, lt, rt)
		}
	}
}

// TestTheCollectionPassRunsOnlyWhereItCanAnswer: the pass types a function
// returning an element anywhere, and one returning a whole struct only where
// no tag region marks every collection escaped, so a tag file whose functions
// return variables skips it. Skipping it where it could answer would drop a
// type; running it where it cannot was 30-44% of a tag component's parse.
func TestTheCollectionPassRunsOnlyWhereItCanAnswer(t *testing.T) {
	whole := []pendingCall{{varName: "result", returnExpr: true, funcKey: "1:3"}}
	element := []pendingCall{{varName: "result[ 1 ]", returnExpr: true, funcKey: "1:3"}}

	for _, tc := range []struct {
		name    string
		calls   []pendingCall
		tagFile bool
		runs    bool
	}{
		{"script file, whole value", whole, false, true},
		{"script file, element", element, false, true},
		{"tag file, whole value", whole, true, false},
		{"tag file, element", element, true, true},
	} {
		if got := collectionReturnExpressions(tc.calls, tc.tagFile) != nil; got != tc.runs {
			t.Errorf("%s: runs = %v, want %v", tc.name, got, tc.runs)
		}
	}
}

// TestAParseWithoutCallsStillFindsEveryTag: the output-context gate decides
// how the tag walk finds the next tag, not only which text is scanned for
// calls. A parse without call extraction ran ungated, paired the `#` in a
// <script> body with one in a later tag's attribute, and stepped over the
// <cffunction> between them (Lucee's debug template lost doMore that way).
// The editor parses without calls, so it would have lost the function too.
func TestAParseWithoutCallsStillFindsEveryTag(t *testing.T) {
	src := `<cfcomponent>
<cffunction name="output" returntype="void">
	<script>
		<cfif !structKeyExists(Cookie, variables.cookieName)>
			var luceeStyle = document.createElement("style");
			luceeStyle.type = 'text/css';
			luceeStyle.innerHTML = "#-lucee-debug .collapsed { display: none; }";
			document.getElementById("-lucee-debug").prepend(luceeStyle);
		</cfif>
	</script>
</cffunction>
<cffunction name="doMore" returntype="void">
	<cfargument name="custom" type="struct" required="#true#">
</cffunction>
</cfcomponent>`

	for _, calls := range []bool{true, false} {
		pr := ParseWithOptions(testURI, src, &ParseOptions{ExtractCalls: calls})

		found := false
		for i := range pr.Funcs {
			found = found || pr.Funcs[i].Name == "doMore"
		}

		if !found {
			t.Errorf("ExtractCalls %v: doMore not found", calls)
		}
	}
}

// TestScopedVarsComeFromTheParsesOwnWalk: a tag region's variables- and
// this-scope names are the ones the parse's own walk recorded. They used to
// come from a second, ungated walk of the region, which paired the `#` in two
// HTML attributes (a "(" between them made it a span) and skipped the <cfset>
// between them, so
// completion and go-to-definition did not know the name; tassweb's
// main.credit.apply.cfm lost `delim` that way. The second walk was also 10ms
// of every edit outside a function on a 58,000-line component.
func TestScopedVarsComeFromTheParsesOwnWalk(t *testing.T) {
	src := `<button data-targetref="#validationerrors">1 Validation Error(s)</button>
<cfset delim = "&">
<input type="hidden" data-validationtarget="#validationerrors-btn" />`

	pr := ParseWithOptions(testURI, src, &ParseOptions{})

	if vars := pr.VariablesVars(); !slices.Contains(vars, "delim") {
		t.Errorf("variables scope = %v, want delim among them", vars)
	}
}

// TestCollectionWritesComeFromTheParsesOwnWalk: the <cfset>, <cfparam> and
// output-attribute tags the collection passes read are the ones the parse's
// own walk met, not a third, ungated walk of the region's tags, which paired
// the `#` in a jQuery selector with one in a later attribute and lost the
// <cfloop> index (Mura's form builder), and cost the member pass a walk of
// every tag on each edit.
func TestCollectionWritesComeFromTheParsesOwnWalk(t *testing.T) {
	src := "\t\t$(\"#\"+formDiv).submit();\n<cfloop from=\"1\" to=\"#ArrayLen(local.aFieldOrder)#\" index=\"iix\">"

	pr := ParseWithOptions(testURI, src, &ParseOptions{})

	found := false
	for _, w := range pr.collectWrites(nil) {
		found = found || w.target == "iix"
	}

	if !found {
		t.Error("the loop's write to iix was not collected")
	}
}
