package resolve

import (
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/parser"
)

// A batch scan's workers ask for the same handler parse at once: a file many
// others call is parsed for each of their callers' arguments, and every worker
// that missed the cache parsed it, 98 of MasaCMS's 651 handler parses. A
// worker that misses while another is parsing the file now waits for that
// parse and takes its answer, which is the one it would have read from the
// cache a moment later.
//
// Waiting must never deadlock. A parse asks other files, so the worker it
// waits for may come to wait for it, directly or through others: before
// waiting, a worker follows the chain of whom each waits for, and when it
// leads back to itself it parses the file itself instead, as it did before. A
// wait can also meet a lock the chain does not show (a sync.Once another
// parse holds while asking for this one), so it gives up after
// handlerFlightPatience and parses the file itself. Either way the answer is
// the same parse; only the work is repeated.

// handlerFlightPatience is how long a worker waits for another's parse before
// making its own. A handler parse takes milliseconds; this is far beyond one.
var handlerFlightPatience = 2 * time.Second

// handlerFlight is a handler parse in progress: whose it is, and, once done
// is closed, its result and whether it was kept (a parse that borrowed an
// outer file's is not, and is no answer for anyone else).
type handlerFlight struct {
	owner uint64
	done  chan struct{}
	pr    *parser.ParseResult
	ok    bool
}

type handlerFlightKey struct {
	r   *Resolver
	key string
}

var handlerFlights = struct {
	sync.Mutex
	inFlight map[handlerFlightKey]*handlerFlight
	// waitsFor is the worker each waiting worker waits for.
	waitsFor map[uint64]uint64
}{inFlight: map[handlerFlightKey]*handlerFlight{}, waitsFor: map[uint64]uint64{}}

// joinHandlerFlight is the flight for key's parse. When no worker is parsing
// it, the flight is new and this worker's to fly (mine). Otherwise it waits
// for the flight and returns it when its parse was kept; it returns nil when
// the parse was not kept, when waiting could deadlock, or when it waited too
// long, and the caller parses the file itself.
func (r *Resolver) joinHandlerFlight(key string) (flight *handlerFlight, mine bool) {
	me := goroutineID()
	fk := handlerFlightKey{r: r, key: key}

	handlerFlights.Lock()

	flight = handlerFlights.inFlight[fk]
	if flight == nil {
		flight = &handlerFlight{owner: me, done: make(chan struct{})}
		handlerFlights.inFlight[fk] = flight
		handlerFlights.Unlock()

		return flight, true
	}

	// The worker already parsing the file is this one (handlerParse's
	// reentry answers that), or waits, through others, for this one.
	if flight.owner == me || waitsOn(flight.owner, me) {
		handlerFlights.Unlock()

		return nil, false
	}

	handlerFlights.waitsFor[me] = flight.owner
	handlerFlights.Unlock()

	timer := time.NewTimer(handlerFlightPatience)

	landed := false

	select {
	case <-flight.done:
		landed = true
	case <-timer.C:
	}

	timer.Stop()

	handlerFlights.Lock()
	delete(handlerFlights.waitsFor, me)
	handlerFlights.Unlock()

	if landed && flight.ok {
		return flight, false
	}

	return nil, false
}

// waitsOn reports whether worker from waits, through any chain of waiting
// workers, for worker target. Callers hold handlerFlights.
func waitsOn(from, target uint64) bool {
	for steps := 0; steps <= len(handlerFlights.waitsFor); steps++ {
		if from == target {
			return true
		}

		next, ok := handlerFlights.waitsFor[from]
		if !ok {
			return false
		}

		from = next
	}

	return false
}

// land ends the flight: its result is set, and every worker waiting on it
// takes it.
func (f *handlerFlight) land(r *Resolver, key string) {
	handlerFlights.Lock()
	delete(handlerFlights.inFlight, handlerFlightKey{r: r, key: key})
	handlerFlights.Unlock()

	close(f.done)
}
