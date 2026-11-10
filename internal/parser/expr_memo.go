package parser

import (
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
	mu sync.Mutex
	m  map[string]T
}

// maxExprMemo bounds each exprMemo.
const maxExprMemo = 1 << 15

// maxExprKey is the longest expression kept: a longer one is answered afresh,
// since copying it would cost about what the scan does.
const maxExprKey = 512

func (e *exprMemo[T]) get(expr string, answer func(string) T) T {
	if len(expr) > maxExprKey {
		return answer(expr)
	}

	e.mu.Lock()
	held, ok := e.m[expr]
	e.mu.Unlock()

	if ok {
		return held
	}

	held = answer(expr)

	e.mu.Lock()
	if e.m == nil || len(e.m) >= maxExprMemo {
		e.m = make(map[string]T)
	}

	e.m[strings.Clone(expr)] = held
	e.mu.Unlock()

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
