package resolve

import (
	"regexp"
	"sync"
	"sync/atomic"
)

// patterns holds the regular expressions built from a name at run time
// (`var\s+<name>`, an rc.<name> write, an existence guard on <name>). They
// were compiled on every lookup, and the same few names are asked about
// thousands of times in a scan. Past maxPatterns it starts again.
var (
	patterns     sync.Map // pattern → *regexp.Regexp
	patternCount atomic.Int64
)

const maxPatterns = 4096

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
