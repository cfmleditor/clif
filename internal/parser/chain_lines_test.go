package parser

import (
	"strings"
	"testing"
)

// A call is on the line its name is written on, whatever the chain it ends
// holds above it. Three walkers used to stamp a call with the line the chain
// began on — the bare-call chain every hop, a scope chain its first call, and
// an unscoped receiver alone on a line its first call — which made every such
// call disagree with the grammar's reading of the same file (make gapcheck).
// Every call name below is written once, so the line to expect is the line the
// name is found on.
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
	lines := strings.Split(src, "\n")
	pr := ParseWithOptions("file:///t.cfc", src, &ParseOptions{ExtractCalls: true})

	seen := map[string]bool{}

	for _, c := range pr.AllCalls() {
		want := -1

		for i, l := range lines {
			if strings.Contains(l, "."+c.FuncName+"(") {
				want = i

				break
			}
		}

		if want < 0 {
			continue // a call the fixture does not write by name (command's own, say)
		}

		seen[c.FuncName] = true

		if int(c.Line) != want {
			t.Errorf("%s is on line %d, written on line %d", c.FuncName, c.Line+1, want+1)
		}
	}

	for _, name := range []string{
		"lineOne", "lineTwo", "lineThree", "params", "runIt", "eOne", "eTwo",
		"gOne", "gTwo", "iOne", "jOne", "lOne", "lTwo", "nOne", "nTwo",
		"rangeClosed", "toArr", "initIt", "finishIt", "tearDown",
		"toInst", "atZone", "toLocal",
	} {
		if !seen[name] {
			t.Errorf("no call recorded for %s", name)
		}
	}
}
