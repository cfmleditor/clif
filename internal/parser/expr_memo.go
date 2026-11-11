package parser

import (
	"hash/maphash"
	"strings"
	"sync"
)

// exprMemo keeps a pure function's answers by the expression text it was
// given. The collection readers (collectionRead, emptyCollection,
// factoryReturnChain) are asked of each write's right-hand side again by every
// pass over the writes, of every parse of the file (index, scan, handler), and
// each answer started a scanner: tens of thousands a scan. An answer depends
// on the text alone, so a memo cannot change one.
//
// The key is copied, and an answer must copy what it keeps of the text, so an
// entry keeps no file alive. The memo is bounded by a count, cleared when
// reached.
type exprMemo[T any] struct {
	shards [exprMemoShards]exprMemoShard[T]
}

// exprMemoShard is one lock's share of an exprMemo. Every worker of a scan
// reads every write's right-hand side through these memos, and behind one
// lock the waiting cost more than the scans they save (tassweb: the
// scheduler's sleeps and wakes rose by a second and a half of CPU).
type exprMemoShard[T any] struct {
	mu sync.Mutex
	m  map[string]T
	_  [40]byte // apart from the next shard's lock
}

// exprMemoShards is how many locks an exprMemo is split across.
const exprMemoShards = 64

// maxExprMemo bounds each exprMemo, spread over its shards: about 2MB of keys
// at most, beside the answers.
const maxExprMemo = 1 << 14

// maxExprKey is the longest expression kept: a longer one is answered afresh,
// since copying it would cost about what the scan does.
const maxExprKey = 512

var exprMemoSeed = maphash.MakeSeed()

func (e *exprMemo[T]) get(expr string, answer func(string) T) T {
	if len(expr) > maxExprKey {
		return answer(expr)
	}

	sh := &e.shards[maphash.String(exprMemoSeed, expr)%exprMemoShards]

	sh.mu.Lock()
	held, ok := sh.m[expr]
	sh.mu.Unlock()

	if ok {
		return held
	}

	held = answer(expr)

	sh.mu.Lock()
	if sh.m == nil || len(sh.m) >= maxExprMemo/exprMemoShards {
		sh.m = make(map[string]T)
	}

	sh.m[strings.Clone(expr)] = held
	sh.mu.Unlock()

	return held
}

func cloneStrings(s []string) []string {
	if s == nil {
		return nil
	}

	out := make([]string, len(s))
	for i, v := range s {
		out[i] = strings.Clone(v)
	}

	return out
}
