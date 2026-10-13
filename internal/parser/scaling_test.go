package parser

import (
	"fmt"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.lsp.dev/uri"
)

// These tests hold the parser to work proportional to its input on inputs
// built to make it quadratic. Each parses n and 8n copies of one construct and
// compares the two costs, never a cost against a clock: a busy runner slows
// both runs alike. Linear is 8x and quadratic 64x; failing above three times
// the input ratio leaves room for cache effects and still fails long before
// anything quadratic.

const scalingInputRatio = 8

// parseAllCalls parses src as the batch scans do, with every call recorded.
func parseAllCalls(u uri.URI, src string) *ParseResult {
	return ParseWithOptions(u, src, &ParseOptions{ExtractCalls: true})
}

// bytesToParse is what one parse of src allocates. It is a count rather than a
// duration, so it does not move with the machine's load.
func bytesToParse(u uri.URI, src string) uint64 {
	var before, after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	_ = parseAllCalls(u, src)

	runtime.ReadMemStats(&after)

	return after.TotalAlloc - before.TotalAlloc
}

// nsToParse is the best of several rounds of the time one parse of src takes,
// each round repeating the parse for long enough to time. The minimum is the
// run least disturbed by whatever else the machine was doing.
func nsToParse(u uri.URI, src string) float64 {
	best := 0.0

	for range 5 {
		n := 0
		start := time.Now()

		for n == 0 || time.Since(start) < 20*time.Millisecond {
			_ = parseAllCalls(u, src)
			n++
		}

		perOp := float64(time.Since(start).Nanoseconds()) / float64(n)
		if best == 0 || perOp < best {
			best = perOp
		}
	}

	return best
}

// assertScalesLinearly fails when 8x the input costs more than three times 8x.
func assertScalesLinearly(t *testing.T, what string, small, large float64) {
	t.Helper()

	got := large / small
	t.Logf("%s: input %dx -> %.1fx (%.0f vs %.0f)", what, scalingInputRatio, got, small, large)

	if got > scalingInputRatio*3 {
		t.Errorf("%s: %dx the input cost %.1fx as much; expected roughly linear", what, scalingInputRatio, got)
	}
}

// Every hop of a chain records a call holding the hops before it, so a chain
// of n hops records n chains of up to n hops. Copying each one allocated
// quadratically: 8,000 hops, 32KB of source, allocated 520MB. Each of the
// three walks that record a chain is driven here.
func TestLongCallChainsAllocateLinearly(t *testing.T) {
	cases := map[string]func(n int) string{
		"variable chain":  func(n int) string { return "component { function f() { x = a" + strings.Repeat(".b()", n) + "; } }" },
		"bare call chain": func(n int) string { return "component { function f() { a()" + strings.Repeat(".b()", n) + "; } }" },
		"constructed chain": func(n int) string {
			return "component { function f() { x = new Foo()" + strings.Repeat(".b()", n) + "; } }"
		},
	}

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			const n = 500

			small := bytesToParse(testURI, input(n))
			large := bytesToParse(testURI, input(n*scalingInputRatio))

			assertScalesLinearly(t, name, float64(small), float64(large))
		})
	}
}

// The calls of one chain share the hops they hold rather than copying them, so
// each must still hold exactly the hops before it, and appending to one must
// not reach into another.
func TestCallChainsHoldTheirOwnHops(t *testing.T) {
	for _, src := range []string{
		"component { function f() { x = a.b().c().d().e(); } }",
		"component { function f() { a().b().c().d().e(); } }",
		"component { function f() { x = new Foo().b().c().d().e(); } }",
	} {
		calls := parseAllCalls(testURI, src).AllCalls()

		var chained []CallSite

		for i := range calls {
			if calls[i].FuncName != "a" && calls[i].FuncName != "init" {
				chained = append(chained, calls[i])
			}
		}

		if len(chained) < 4 {
			t.Fatalf("%s: expected four chained calls, got %+v", src, calls)
		}

		want := make([][]string, len(chained))
		for i := range chained {
			want[i] = slices.Clone(chained[i].Chain)
		}

		for i := range chained {
			if i > 0 && len(chained[i].Chain) != len(chained[i-1].Chain)+1 {
				t.Errorf("%s: call %s holds %v after %v", src, chained[i].FuncName, chained[i].Chain, chained[i-1].Chain)
			}

			_ = append(chained[i].Chain, "written")
		}

		for i := range chained {
			if !slices.Equal(chained[i].Chain, want[i]) {
				t.Errorf("%s: appending to one call's chain changed %s's to %v, was %v", src, chained[i].FuncName, chained[i].Chain, want[i])
			}
		}
	}
}

// Each <cffunction> looks for its </cffunction>, and with none in the file the
// search ran to the end of it from every function: 24,000 unclosed functions,
// 1MB, took 5.6s.
func TestUnclosedFunctionsParseInLinearTime(t *testing.T) {
	page := uri.File("/tmp/page.cfm")
	input := func(n int) string {
		return strings.Repeat("<cffunction name=\"f\"><cfargument name=\"a\">\n", n)
	}

	const n = 400

	small := nsToParse(page, input(n))
	large := nsToParse(page, input(n*scalingInputRatio))

	assertScalesLinearly(t, "unclosed <cffunction>", small, large)
}

// `a = a = … = rhs` refiled, for every name, what every name inside it had
// made, in a recursion as deep as the chain: 20,000 names took 1.1s.
func TestLongChainedAssignmentsParseInLinearTime(t *testing.T) {
	cases := map[string]struct {
		u     uri.URI
		input func(n int) string
	}{
		"script": {testURI, func(n int) string { return "component { function f() { " + strings.Repeat("a=", n) + "new X(); } }" }},
		"tag":    {uri.File("/tmp/page.cfm"), func(n int) string { return "<cfset " + strings.Repeat("a=", n) + "new X()>" }},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			const n = 1000

			small := nsToParse(c.u, c.input(n))
			large := nsToParse(c.u, c.input(n*scalingInputRatio))

			assertScalesLinearly(t, name+" chained assignment", small, large)
		})
	}
}

// A tag with no '>' after it has no end, and every walk that asked for one
// stepped a byte past the tag and asked again from the next, each time
// scanning to the end of the file: "<cf" repeated to 1.5MB took 5.9s. The
// tags here reach each walk that asks: the tag walk's handled and declined
// tags, and the output-context tracking of <cfoutput> and <cfcomponent>.
func TestUnterminatedTagsParseInLinearTime(t *testing.T) {
	page := uri.File("/tmp/page.cfm")

	for _, unit := range []string{"<cf", "<cfset ", "<cfoutput ", "<cfcomponent "} {
		t.Run(unit, func(t *testing.T) {
			const n = 10000

			small := nsToParse(page, strings.Repeat(unit, n))
			large := nsToParse(page, strings.Repeat(unit, n*scalingInputRatio))

			assertScalesLinearly(t, unit, small, large)
		})
	}
}

// A '>' inside a quoted value, with no real end after it, sent the quote-aware
// search to the end of the file from every tag: `<cfset x="a>b"` repeated to
// 560KB took 19.7s. A run with a single '>' at its very end cost the plain
// search and its quote counts the same way. Each shape reaches the tag walk,
// the output-context tracking and the collection-write scan.
func TestTagsWithNoRealEndParseInLinearTime(t *testing.T) {
	page := uri.File("/tmp/page.cfm")

	for name, build := range map[string]func(n int) string{
		"quoted '>' never closed":     func(n int) string { return strings.Repeat(`<cfset x="a>b"`, n) },
		"one '>' at the end":          func(n int) string { return strings.Repeat("<cf", n) + ">" },
		"quoted '>', one '>' at end":  func(n int) string { return strings.Repeat(`<cfoutput x="a>b"`, n) + ">" },
		"single quotes, never closed": func(n int) string { return strings.Repeat(`<cfset x='a>b'`, n) },
		"comment and quoted '>'":      func(n int) string { return strings.Repeat(`<cf <!-- x="a>b"`, n) },
		"comment, one '>' at the end": func(n int) string { return strings.Repeat("<cf<!--", n) + ">" },
	} {
		t.Run(name, func(t *testing.T) {
			const n = 4000

			small := nsToParse(page, build(n))
			large := nsToParse(page, build(n*scalingInputRatio))

			assertScalesLinearly(t, name, small, large)
		})
	}
}

// Every <cfset> looked for calls recorded before it that lie inside its own
// expression by reading every call the enclosing function had recorded so
// far, so a function of many <cfset> tags was quadratic: 24,000 took 0.4s.
func TestManyCFSetsInOneFunctionParseInLinearTime(t *testing.T) {
	page := uri.File("/tmp/Page.cfc")
	build := func(n int) string {
		return "<cfcomponent><cffunction name=\"f\">" + strings.Repeat("<cfset x = g(a)>\n", n) + "</cffunction></cfcomponent>"
	}

	const n = 2000

	small := nsToParse(page, build(n))
	large := nsToParse(page, build(n*scalingInputRatio))

	assertScalesLinearly(t, "<cfset> in one function", small, large)
}

// Folding reads tags with the same search, so a page of unterminated tags
// was quadratic there too.
func TestFoldingUnterminatedTagsScalesLinearly(t *testing.T) {
	cost := func(src string) float64 {
		best := 0.0

		for range 5 {
			start := time.Now()
			_ = StructureSpans(src)

			if d := float64(time.Since(start).Nanoseconds()); best == 0 || d < best {
				best = d
			}
		}

		return best
	}

	const n = 5000

	build := func(n int) string { return strings.Repeat(`<a x="b>c"`, n) + ">" }

	assertScalesLinearly(t, "folding unterminated tags", cost(build(n)), cost(build(n*scalingInputRatio)))
}

// A tag name running to the end of the text has no '>' and no attributes,
// and folding panicked on it.
func TestFoldingATagNameAtTheEndDoesNotPanic(t *testing.T) {
	for _, src := range []string{"<a", "<cf", "x\n<div", "</a", "<p>\n<cfif"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("StructureSpans(%q) panicked: %v", src, r)
				}
			}()

			_ = StructureSpans(src)
		}()
	}
}

// The shared search gives every offset the answer tagEndIndex gives for the
// text from there, which is what the walks used before.
func TestTagEndsAgreesWithTagEndIndex(t *testing.T) {
	texts := []string{
		`<cfset x="a>b"><cfif a>`,
		`<cfset x='a>b' y="c">d<cfset z="e>f">`,
		`<cfset x="a""b>c"> <cfset y = 1>`,
		`<cffunction name="f" <!--- a > b ---> output="false">`,
		`<cfset x="never closed>`,
		strings.Repeat(`<cfset x="a>b"`, 50) + `<cfset y=1>`,
		strings.Repeat("<cf", 2000) + `x="a>b" >` + strings.Repeat(`<cfset q='>'>`, 20),
		// Enough long spans to build the index, then a walk that steps past
		// the last '>': the index's "no '>' after here" is not an end.
		strings.Repeat("<cf"+strings.Repeat(" ", longTagSpan+1), longSpansBeforeIndex+1) + `x="a>b"`,
		`<input <cfif a>checked</cfif>>`,
		strings.Repeat(`<cf <!-- x="a>b"`, 30) + `<cfset y=1>`,
		`<cffunction <!--- a "q" ---> name="f" <!--- unclosed`,
		`<cf <!-- x='a"b' y="c>d" >` + strings.Repeat(`<cf<!--`, 20) + ">",
		"",
		">",
	}

	// Each text is asked twice: as a walk meets it, and with the index built
	// before the first answer, since the walk's memo can otherwise answer
	// from a walk made before the index existed and hide where they differ.
	for _, text := range texts {
		for _, indexed := range []bool{false, true} {
			ends := newTagEnds(text)
			if indexed {
				ends.gaps = indexTagGaps(text)
			}

			for from := range len(text) {
				if got, want := ends.at(from), tagEndIndex(text[from:]); got != want {
					t.Errorf("%q from %d (indexed %v): tagEnds %d, tagEndIndex %d", text[:min(len(text), 40)], from, indexed, got, want)

					break
				}
			}
		}
	}
}

// A document that keeps asking for ends far away is indexed once, which is
// what keeps "<cf" repeated before a single '>' linear; the timing sub-case
// for that shape cannot tell, since the old scan is fast enough per call.
func TestRepeatedLongSpansBuildTheIndex(t *testing.T) {
	text := strings.Repeat("<cf", 3000) + ">"

	ends := newTagEnds(text)
	for i := 0; i < len(text); i += 3 {
		_ = ends.at(i)
	}

	if ends.gaps == nil {
		t.Error("tag ends were scanned again for every tag; want the long spans indexed")
	}

	// Asked in any order, distinct long tags are what count.
	backwards := newTagEnds(text)
	for i := len(text) - 4; i >= 0; i -= 3 {
		_ = backwards.at(i)
	}

	if backwards.gaps == nil {
		t.Error("long tags asked about in descending order never built the index")
	}

	one := newTagEnds("<cfset q = \"" + strings.Repeat("x", longTagSpan+1) + "\">")
	for range longSpansBeforeIndex {
		_ = one.at(0) // the same tag asked again counts once
	}

	if one.gaps != nil {
		t.Error("a single long tag built the index; want it kept for documents that keep asking")
	}
}

// Ten shapes that were each quadratic in something a real file repeats: a
// search for the next '<' or <cfscript> redone per '#' or <script> block, a
// linear scan of a function's locals, of a file's refs, scopes, pending calls
// or returns, and newline counts from a string's start for every span or
// call. Each ran 22x to 62x for 8x the input; the parse and one function's
// lazily parsed variables and refs are held to linear here.
func TestRepeatedConstructsParseInLinearTime(t *testing.T) {
	cfc, cfm := uri.File("/tmp/Shape.cfc"), uri.File("/tmp/shape.cfm")
	script := func(body string) string { return "component {\nfunction f() {\n" + body + "\n}\n}" }
	tagFn := func(body string) string {
		return "<cfcomponent><cffunction name=\"f\">\n" + body + "\n</cffunction></cfcomponent>"
	}
	each := func(n int, line func(i int) string) string {
		var b strings.Builder
		for i := range n {
			b.WriteString(line(i))
		}

		return b.String()
	}

	// n is the smaller size; the milder shapes need more of themselves
	// before the quadratic part outweighs the rest of a parse.
	shapes := map[string]struct {
		u     uri.URI
		build func(n int) string
		n     int
	}{
		"hashes in <cfoutput>": {cfm, func(n int) string { return "<cfoutput>" + strings.Repeat("#", 40*n) + "</cfoutput>" }, 1000},
		"<script> blocks":      {cfm, func(n int) string { return strings.Repeat("<script>var a = '#x()#';</script>\n", n) }, 1000},
		"spans in <cfoutput>":  {cfm, func(n int) string { return "<cfoutput>" + strings.Repeat("#a.b()# text ", n) + "</cfoutput>" }, 6000},
		"spans in a string":    {cfc, func(n int) string { return script(`x = "` + strings.Repeat("#a()# ", n) + `";`) }, 4000},
		"named arguments": {cfm, func(n int) string {
			return "<cfset x = f(" + each(n, func(i int) string { return fmt.Sprintf("a%d=b(),", i) }) + "1)>"
		}, 6000},
		"<cfset var> locals": {cfc, func(n int) string {
			return tagFn(each(n, func(i int) string { return fmt.Sprintf("<cfset var v%d = %d>\n", i, i) }))
		}, 1000},
		"<cfargument> list": {cfc, func(n int) string {
			return tagFn(each(n, func(i int) string { return fmt.Sprintf("<cfargument name=\"a%d\" type=\"A\">\n", i) }) + "<cfreturn a1.m()>")
		}, 1000},
		"locals from the last": {cfc, func(n int) string {
			return script("var v0 = new A();\n" + each(n, func(i int) string { return fmt.Sprintf("var v%d = v%d.next();\n", i+1, i) }))
		}, 1000},
		"functions calling next": {cfc, func(n int) string {
			return "component {\n" + each(n, func(i int) string { return fmt.Sprintf("function f%d() { return f%d(); }\n", i, (i+1)%n) }) + "}"
		}, 1000},
		"tag functions": {cfc, func(n int) string {
			return "<cfcomponent>" + each(n, func(i int) string {
				return fmt.Sprintf("<cffunction name=\"f%d\"><cfreturn f%d()></cffunction>\n", i, i)
			}) + "</cfcomponent>"
		}, 4000},
	}

	for name, shape := range shapes {
		t.Run(name, func(t *testing.T) {
			n := shape.n

			small := nsToParse(shape.u, shape.build(n))
			large := nsToParse(shape.u, shape.build(n*scalingInputRatio))

			assertScalesLinearly(t, name, small, large)

			lazy := func(n int) float64 {
				src := shape.build(n)
				scopes := Parse(shape.u, src).Scopes

				if len(scopes) == 0 {
					return 0
				}

				s := scopes[len(scopes)/2]

				return nsToParse(shape.u, src) + nsToRun(func() {
					pr := Parse(shape.u, src)
					_ = pr.FuncVars(s.Start, s.End)
					_, _ = pr.FuncRefs(s.Start, s.End)
				})
			}

			if small := lazy(n); small > 0 {
				assertScalesLinearly(t, name+", one function's variables and refs", small, lazy(n*scalingInputRatio))
			}
		})
	}
}

// nsToRun is nsToParse for any function.
func nsToRun(f func()) float64 {
	best := 0.0

	for range 5 {
		n := 0
		start := time.Now()

		for n == 0 || time.Since(start) < 20*time.Millisecond {
			f()

			n++
		}

		if perOp := float64(time.Since(start).Nanoseconds()) / float64(n); best == 0 || perOp < best {
			best = perOp
		}
	}

	return best
}

// scopeIndex answers as the scans it replaced do, past the size where it
// builds its table: the first scope in order holding the line, nested and
// overlapping scopes included, with and without the end lines.
func TestScopeIndexAgreesWithTheScan(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))

	for round := range 50 {
		var scopes []FuncScope

		for range maxScopeScan + 1 + rng.IntN(60) {
			start := rng.IntN(400)
			scopes = append(scopes, FuncScope{Name: strconv.Itoa(len(scopes)), Start: start, End: start + rng.IntN(40)})
		}

		inclusive := scopeIndex(scopes, true)
		exclusive := scopeIndex(scopes, false)
		pr := &ParseResult{Scopes: scopes}

		for line := -2; line < 460; line++ {
			want := -1

			if s := findFuncScope(line, scopes); s.Start >= 0 {
				for i := range scopes {
					if scopes[i] == s {
						want = i

						break
					}
				}
			}

			if got := inclusive(line); got != want {
				t.Fatalf("round %d line %d: inclusive %d, findFuncScope %d", round, line, got, want)
			}

			wantName, got := pr.callerAtLine(line), ""
			if k := exclusive(line); k >= 0 {
				got = scopes[k].Name
			}

			if got != wantName {
				t.Fatalf("round %d line %d: exclusive %q, callerAtLine %q", round, line, got, wantName)
			}
		}
	}
}

// firstRef and refsOnLine answer from their index as the scans they replaced
// do, for every name and scope, and keep doing so as refs are appended.
func TestRefIndexAgreesWithTheScan(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 4))
	names := []string{"a", "B", "svc", "Svc", "x", "dao"}
	pr := &ParseResult{refIdx: map[*ComponentRef]*refNames{}}

	refs := make([]ComponentRef, 0, 400)

	for range 300 {
		refs = append(refs, ComponentRef{
			Variable: names[rng.IntN(len(names))],
			This:     rng.IntN(3) == 0,
			Line:     uint32(rng.IntN(50)),
		})

		if len(refs) <= maxRefScan {
			continue
		}

		for _, name := range names {
			for _, scope := range []RefScope{RefAny, RefThis, RefVariables} {
				if got, want := pr.firstRef(refs, name, scope), firstRefIn(refs, name, scope); got != want {
					t.Fatalf("%d refs, %q scope %d: index %v, scan %v", len(refs), name, scope, got, want)
				}
			}
		}

		line := uint32(rng.IntN(50))

		var got, want []*ComponentRef

		pr.refsOnLine(refs, line, func(r *ComponentRef) bool {
			got = append(got, r)

			return false
		})

		for i := range refs {
			if refs[i].Line == line {
				want = append(want, &refs[i])
			}
		}

		if !slices.Equal(got, want) {
			t.Fatalf("%d refs, line %d: index %d refs, scan %d", len(refs), line, len(got), len(want))
		}
	}
}
