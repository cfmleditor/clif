package resolve

import (
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
)

// handlerParses counts the parses handlerParse runs, for the test that holds
// a self-referring file to a few of them.
var handlerParses atomic.Int64

// A handler parse asks this resolver what functions return, and answering can
// need the parse of the very file being parsed: Preside's PageTypesService
// returns an element of what _getRegisteredPageTypes() returns, reading that
// function's body takes the file's parse (fnBody, builtCollectionElement), and
// that parse types `getPageType( id )` by asking getPageType's return again.
// The file is cached only once its parse finishes, so each ask parsed it anew:
// a 50-line cut of that file took ten seconds and the whole checkout never
// finished. A parse asked for while the same goroutine is already parsing that
// file is the plain parse instead (as elementReturn reads it), which holds the
// text, its scopes and its declarations, and asks nothing.
//
// Which parse a lookup answers from must not depend on which file the scan
// happened to reach first, or the report changes between runs. A parse is
// therefore cached only when everything it read was its own: one that met an
// outer file's parse in progress, and so read that file's plain parse, is used
// by the parse that asked for it and then dropped. What is cached is always the
// file parsed with itself at the root, whoever asked.

// handlerParseFrame is one handler parse in progress on a goroutine.
type handlerParseFrame struct {
	key string
	// borrowed is set when the parse, or one it started, met a file further
	// out on the goroutine that was still being parsed.
	borrowed bool
}

var handlerParsing = struct {
	sync.Mutex
	stacks map[uint64][]*handlerParseFrame
}{stacks: map[uint64][]*handlerParseFrame{}}

// enterHandlerParse records that the current goroutine starts parsing key. It
// reports nil and true when the goroutine is already parsing key, marking
// every parse begun since as having read an unfinished file.
func enterHandlerParse(key string) (*handlerParseFrame, bool) {
	id := goroutineID()

	handlerParsing.Lock()
	defer handlerParsing.Unlock()

	stack := handlerParsing.stacks[id]

	for i, f := range slices.Backward(stack) {
		if f.key != key {
			continue
		}

		for _, f := range stack[i+1:] {
			f.borrowed = true
		}

		return nil, true
	}

	frame := &handlerParseFrame{key: key}
	handlerParsing.stacks[id] = append(stack, frame)

	return frame, false
}

// leaveHandlerParse records that the current goroutine finished frame's parse.
func leaveHandlerParse(frame *handlerParseFrame) {
	id := goroutineID()

	handlerParsing.Lock()
	defer handlerParsing.Unlock()

	stack := handlerParsing.stacks[id]
	if n := len(stack); n > 0 && stack[n-1] == frame {
		stack = stack[:n-1]
	}

	if len(stack) == 0 {
		delete(handlerParsing.stacks, id)

		return
	}

	handlerParsing.stacks[id] = stack
}

// goroutineID is the running goroutine's number, read from the header of its
// stack trace ("goroutine 18 [running]:"). It is asked only when a handler
// parse misses the cache, once per file.
func goroutineID() uint64 {
	var buf [64]byte

	n := runtime.Stack(buf[:], false)

	const prefix = "goroutine "

	var id uint64

	for _, c := range buf[min(len(prefix), n):n] {
		if c < '0' || c > '9' {
			break
		}

		id = id*10 + uint64(c-'0')
	}

	return id
}
