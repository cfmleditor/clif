package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// The parser's shared token lists are bounded with the other text caches in a
// long-lived process, lifted for a batch run inside it, and put back after.
func TestLimitTextCachesBoundsTheParsersSharedTokens(t *testing.T) {
	before := parser.SharedTokenLimit()

	t.Cleanup(func() { parser.SetSharedTokenLimit(before) })

	parser.SetSharedTokenLimit(parser.DefaultSharedTokenBytes)

	restore := LimitTextCaches(1 << 20)

	if got := parser.SharedTokenLimit(); got != 1<<20 {
		t.Errorf("limited to %d, want %d", got, 1<<20)
	}

	done := BatchTextCaches()

	if got := parser.SharedTokenLimit(); got != parser.DefaultSharedTokenBytes {
		t.Errorf("during a batch %d, want the default %d", got, parser.DefaultSharedTokenBytes)
	}

	done()

	if got := parser.SharedTokenLimit(); got != 1<<20 {
		t.Errorf("after the batch %d, want %d", got, 1<<20)
	}

	restore()

	if got := parser.SharedTokenLimit(); got != parser.DefaultSharedTokenBytes {
		t.Errorf("restored to %d, want the default %d", got, parser.DefaultSharedTokenBytes)
	}
}
