package resolve

import (
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/uri"
)

func TestCallerTokensReuseAndRefresh(t *testing.T) {
	r := &Resolver{}
	pr := &parser.ParseResult{URI: uri.URI("file:///caller.cfc"), Content: strings.Repeat("service.call(value);\n", 1000)}

	first := r.callerFileTokens(pr)
	if len(first) == 0 {
		t.Fatal("empty tokens")
	}

	view := &Resolver{indexer: r}
	if got := view.callerFileTokens(pr); &got[0] != &first[0] {
		t.Fatal("caller view rebuilt tokens")
	}

	allocs := testing.AllocsPerRun(10, func() { r.callerFileTokens(pr) })
	if allocs > 1 {
		t.Fatalf("unchanged caller allocated %g times", allocs)
	}

	changed := &parser.ParseResult{URI: pr.URI, Content: "other();"}
	if got := r.callerFileTokens(changed); got[0].Value != "other" {
		t.Fatal("stale tokens after edit")
	}

	if got := r.callerFileTokens(pr); got[0].Value != "service" {
		t.Fatal("old parse lost its lexical identity")
	}
}
