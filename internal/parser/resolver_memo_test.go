package parser

import (
	"fmt"
	"sync"
	"testing"
	"unsafe"
)

// The shared memo keeps every answer while no shard is past its cap, under
// parse workers asking at once (run with -race), and a set built for one parse
// has a single shard. 20,000 expressions over 64 shards is about 312 a shard,
// well inside the 1,024 each may hold: a cap computed wrongly empties shards
// and the count falls short.
func TestTheResolverMemoKeepsWhatItsShardsHold(t *testing.T) {
	const exprs = 20000

	resolvers := []Resolver{{Match: "svc$1", Resolve: "models.Svc", Prefix: "svc"}}

	shared := BuildResolverSet(resolvers)

	var wg sync.WaitGroup

	for w := range 8 {
		wg.Go(func() {
			set := shared.ForParse()
			for i := w; i < exprs; i += 8 {
				set.Resolve(fmt.Sprintf("x%d.get()", i))
			}
		})
	}

	wg.Wait()

	if n := shared.memo.size(); n != exprs {
		t.Fatalf("the shared memo holds %d answers, want %d", n, exprs)
	}

	private := newPrivateResolverSet(resolvers)
	if n := len(private.memo.shards); n != 1 {
		t.Fatalf("a set for one parse has %d shards, want 1", n)
	}

	for i := range exprs {
		private.Resolve(fmt.Sprintf("x%d.get()", i))
	}

	if n := private.memo.size(); n != exprs {
		t.Fatalf("the private memo holds %d answers, want %d", n, exprs)
	}

	if size := unsafe.Sizeof(resolverMemoShard{}); size != 64 {
		t.Fatalf("a memo shard is %d bytes; it should fill one 64-byte cache line", size)
	}
}
