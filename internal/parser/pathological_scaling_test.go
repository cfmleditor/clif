package parser

import (
	"strings"
	"testing"

	"go.lsp.dev/uri"
)

// scalesLinearly fails when eight times the input costs more than three times
// that ratio (TestScanScalesLinearly's rule): quadratic work is 64x, and a
// busy runner moves both timings together.
func scalesLinearly(t *testing.T, small, large string, run func(string)) {
	t.Helper()

	bench := func(content string) int64 {
		r := testing.Benchmark(func(b *testing.B) {
			for b.Loop() {
				run(content)
			}
		})

		return r.NsPerOp()
	}

	bench(small)

	ratio := float64(len(large)) / float64(len(small))
	smallNs, largeNs := bench(small), bench(large)
	got := float64(largeNs) / float64(smallNs)

	t.Logf("input %.0fx -> time %.1fx (%dns vs %dns)", ratio, got, smallNs, largeNs)

	if got > ratio*3 {
		t.Errorf("%.0fx the input took %.1fx the time; expected roughly linear", ratio, got)
	}
}

// The include scan's comment reader searched for the next close again after
// every nested open, so a comment nested N deep read its body N times: 200,000
// levels hung indexing. Includes are read by the document walk now, through
// skipTagComment.
func TestNestedCommentEndScalesLinearly(t *testing.T) {
	nested := func(n int) string { return strings.Repeat("<!--- ", n) + strings.Repeat(" --->", n) }

	scalesLinearly(t, nested(2000), nested(16000), func(s string) { _ = IncludeSites(s + `<cfinclude template="a.cfm">`) })
}

// Each call in a <cfset> with no span was found by rescanning the whole
// expression, so thousands of unclosed interpolations were quadratic and hung
// the parse; maxOriginScan bounds it.
func TestATagExpressionOfManyCallsScalesLinearly(t *testing.T) {
	page := func(n int) string { return `<cfset x = "` + strings.Repeat(`#a(""`, n) + `">` }

	scalesLinearly(t, page(2000), page(16000), func(s string) {
		_ = ParseWithOptions(uri.File("/x/p.cfm"), s, &ParseOptions{ExtractCalls: true, ScanAllScopes: true})
	})
}
