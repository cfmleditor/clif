package parser

import (
	"runtime"
	"slices"
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
