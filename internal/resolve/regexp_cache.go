package resolve

import (
	"regexp"
	"sync"
	"sync/atomic"

	"github.com/cfmleditor/clif/internal/parser"
)

// patterns holds the regular expressions built at run time from a value the
// code spells: an interception point or a Mura object path computed around
// literal parts, an invoke's method selector, a Wheels dispatch over a
// parameter's name. They match a name or a token stream, never CFML text
// (name_writes.go reads the writes and guards of a name from the code). They
// were compiled on every lookup, and the same few values are asked about
// thousands of times in a scan. Past maxPatterns it starts again.
var (
	patterns     sync.Map // pattern → *regexp.Regexp
	patternCount atomic.Int64
)

const maxPatterns = 4096

// mentions reports whether text holds literal, case-insensitively. The
// readers run over whole files and function bodies each need a literal (a
// name, `prc.`, `this.memento`) that a file without it cannot hold, so asking
// this first skips the reader where it would find nothing, which is almost
// everywhere, at the cost of one vectorised search.
func mentions(text, literal string) bool {
	return parser.IndexFold(text, literal) >= 0
}

// compiledPattern is regexp.MustCompile(pattern), compiled once per pattern.
func compiledPattern(pattern string) *regexp.Regexp {
	if re, ok := patterns.Load(pattern); ok {
		compiled, _ := re.(*regexp.Regexp)

		return compiled
	}

	re := regexp.MustCompile(pattern)

	if patternCount.Add(1) > maxPatterns {
		patterns.Clear()
		patternCount.Store(1)
	}

	patterns.Store(pattern, re)

	return re
}
