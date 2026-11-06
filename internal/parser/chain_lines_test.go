package parser

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

// A call is on the line its name is written on, whatever the chain it ends
// holds above it. Walkers used to stamp a call with the line the chain began
// on — the bare-call chain every hop, a scope chain its first call, an
// unscoped receiver alone on a line its first call, a constructor's hops, and
// parse()'s own dot-chain walk — which made every such call disagree with the
// grammar's reading of the same file (make gapcheck). Every call name in a
// fixture is written once, with a dot before it, so the line to expect is the
// line `.name(` is found on.
func TestAChainedCallIsOnTheLineItsNameIsWrittenOn(t *testing.T) {
	t.Parallel()

	src := `component {
	function f() {
		variables.print
			.lineOne()
			.lineTwo( "a" )
			.lineThree();
		command( "x" )
			.params(
				a = 1
			)
			.runIt();
		svc
			.eOne()
			.eTwo();
		var z = svc
			.gOne()
			.gTwo();
		x = svc
			.iOne();
		variables.a
			.jOne();
		request.k
			.lOne()
			.lTwo();
		var arr = createObject( "java", "java.util.stream.IntStream" )
			.rangeClosed( 1, 2 )
			.toArr();
		var w = new Foo()
			.initIt()
			.finishIt();
		var q = createObject(
			"component",
			"x.Y"
		).setUp( 1 )
			.tearDown();
		return svc
			.nOne()
			.nTwo();
	}
	function g() {
		return this
			.toInst( 1 )
			.atZone(
				2
			)
			.toLocal();
	}
}
`

	checkChainLines(t, src,
		"lineOne", "lineTwo", "lineThree", "params", "runIt", "eOne", "eTwo",
		"gOne", "gTwo", "iOne", "jOne", "lOne", "lTwo", "nOne", "nTwo",
		"rangeClosed", "toArr", "initIt", "finishIt", "tearDown",
		"toInst", "atZone", "toLocal")
}

// The same shapes with no function round them: a statement in a component body
// and one in a <cfscript> of a tag file reach parse()'s own dot-chain walk,
// not handleBodyToken's.
func TestAChainedCallOutsideAFunctionIsOnTheLineItsNameIsWrittenOn(t *testing.T) {
	t.Parallel()

	for name, src := range map[string]string{
		"component body": "component {\n\tinstance\n\t\t.ioFile\n\t\t.firstOne( 1 )\n\t\t.secondOne();\n" +
			"\tsvc\n\t\t.thirdOne();\n\tthis.eh\n\t\t.fourthOne( 1 )\n\t\t.fifthOne();\n\tvariables.x\n\t\t.sixthOne( 1 );\n}\n",
		"cfscript in a tag function": "<cfcomponent>\n<cffunction name=\"z\">\n<cfscript>\n\tinstance\n\t\t.ioFile\n" +
			"\t\t.firstOne( 1 )\n\t\t.secondOne();\n\tsvc\n\t\t.thirdOne();\n\tthis.eh\n\t\t.fourthOne( 1 )\n" +
			"\t\t.fifthOne();\n\tvariables.x\n\t\t.sixthOne( 1 );\n</cfscript>\n</cffunction>\n</cfcomponent>\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			checkChainLines(t, src, "firstOne", "secondOne", "thirdOne", "fourthOne", "fifthOne", "sixthOne")
		})
	}
}

// A call on a value that is not a variable — a closing bracket, a string — is on
// its own line, not the receiver's: `getData()[ k ]` written above `.mapIt(…)`.
func TestACallOnAnIndexedValueIsOnTheLineItsNameIsWrittenOn(t *testing.T) {
	t.Parallel()

	checkChainLines(t, `component {
	function f() {
		expect(
			actual.getData()[ k ]
				.mapIt( function( i ){ return 1; } )
				.toListIt()
		);
		var s = "a,b"
			.ucaseIt()
			.trimIt();
	}
}
`, "mapIt", "toListIt", "ucaseIt", "trimIt")
}

// checkChainLines parses src and fails for each call whose recorded line is not
// the one `.name(` is written on, and for each name in want that has no call.
func checkChainLines(t *testing.T, src string, want ...string) {
	t.Helper()

	lines := strings.Split(src, "\n")
	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{ExtractCalls: true})
	seen := map[string]bool{}

	calls := pr.AllCalls()

	for i := range calls {
		c := &calls[i]
		at := -1

		for i, l := range lines {
			if strings.Contains(l, "."+c.FuncName+"(") {
				at = i

				break
			}
		}

		if at < 0 {
			continue // a call the fixture does not write after a dot
		}

		seen[c.FuncName] = true

		if int(c.Line) != at {
			t.Errorf("%s is on line %d, written on line %d", c.FuncName, c.Line+1, at+1)
		}
	}

	for _, name := range want {
		if !seen[name] {
			t.Errorf("no call recorded for %s", name)
		}
	}
}

// A <cfset> whose value starts on a later line than the tag: the call that opens
// the value is on the line it is written on, and is recorded once. The text
// after the "=" was trimmed before the line was counted, so the call was stamped
// on the tag's line as well as its own — 1 extra `array` in Lucee's menu.cfm and
// one in ext.providers.cfm.
func TestACallOpeningAMultiLineSetValueIsOnItsOwnLine(t *testing.T) {
	src := "<cffunction name=\"f\">\n<cfset var m = \n\tarray(\n\t\tstruct(\n\t\t\taction:\"o\",label:a.b,\n\t\t\tchildren:array(\n\t\t\t\tstruct(action:\"\",label:a.c)\n\t\t\t)\n\t\t)\n\t)>\n</cffunction>\n"

	src += "<cffunction name=\"g\">\n<cfset hasData = \n\tStructKeyExists( d, u ) and\n\t!isSimpleValue( d[ u ] )/>\n</cffunction>\n"

	want := []string{"array:2", "struct:3", "array:5", "struct:6", "structkeyexists:13", "issimplevalue:14"}

	var got []string

	for _, c := range ParseWithOptions("file:///x.cfm", src, &ParseOptions{ExtractCalls: true}).AllCalls() {
		got = append(got, strings.ToLower(c.FuncName)+":"+strconv.Itoa(int(c.Line)))
	}

	slices.Sort(got)
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Errorf("calls = %v, want %v", got, want)
	}
}
