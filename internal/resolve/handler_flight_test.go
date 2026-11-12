package resolve

import (
	"testing"
	"time"

	"github.com/cfmleditor/clif/internal/parser"
)

// startFlight makes key's flight on a goroutine of its own, as another worker
// would, and returns it once it is in the air.
func startFlight(t *testing.T, r *Resolver, key string) (*handlerFlight, uint64) {
	t.Helper()

	got := make(chan *handlerFlight)
	owner := make(chan uint64)

	go func() {
		f, mine := r.joinHandlerFlight(key)
		if !mine {
			t.Error("the first worker did not get the flight")
		}

		owner <- goroutineID()

		got <- f
	}()

	id := <-owner

	return <-got, id
}

// A worker that misses while another parses the file takes that parse.
func TestAWorkerWaitsForTheParseInFlight(t *testing.T) {
	r := &Resolver{}
	f, _ := startFlight(t, r, "/w/a.cfc")
	want := &parser.ParseResult{}

	go func() {
		time.Sleep(10 * time.Millisecond)

		f.pr, f.ok = want, true
		f.land(r, "/w/a.cfc")
	}()

	got, mine := r.joinHandlerFlight("/w/a.cfc")
	if mine || got == nil || got.pr != want {
		t.Fatalf("the waiting worker got %+v (mine %v), want the flight's parse", got, mine)
	}
}

// A parse that was not kept answers no one else: the waiter parses itself.
func TestAParseNotKeptIsNoAnswer(t *testing.T) {
	r := &Resolver{}
	f, _ := startFlight(t, r, "/w/b.cfc")

	go func() {
		time.Sleep(10 * time.Millisecond)

		f.pr = &parser.ParseResult{}
		f.land(r, "/w/b.cfc")
	}()

	if got, mine := r.joinHandlerFlight("/w/b.cfc"); mine || got != nil {
		t.Fatalf("a borrowed parse was handed to another worker: %+v", got)
	}
}

// A worker never waits for one that waits, through others, for it.
func TestAWaitThatWouldDeadlockIsNotMade(t *testing.T) {
	r := &Resolver{}
	f, owner := startFlight(t, r, "/w/c.cfc")

	defer f.land(r, "/w/c.cfc")

	me := goroutineID()

	handlerFlights.Lock()
	handlerFlights.waitsFor[owner] = 4242
	handlerFlights.waitsFor[4242] = me
	handlerFlights.Unlock()

	defer func() {
		handlerFlights.Lock()
		delete(handlerFlights.waitsFor, owner)
		delete(handlerFlights.waitsFor, 4242)
		handlerFlights.Unlock()
	}()

	start := time.Now()

	if got, mine := r.joinHandlerFlight("/w/c.cfc"); mine || got != nil {
		t.Fatalf("joined a flight whose owner waits for this worker: %+v", got)
	}

	if waited := time.Since(start); waited > time.Second {
		t.Errorf("waited %v before giving up a wait that would deadlock", waited)
	}
}

// A wait that runs past the patience ends, and the worker parses itself.
func TestALongWaitGivesUp(t *testing.T) {
	saved := handlerFlightPatience
	handlerFlightPatience = 20 * time.Millisecond

	defer func() { handlerFlightPatience = saved }()

	r := &Resolver{}
	f, _ := startFlight(t, r, "/w/d.cfc")

	defer f.land(r, "/w/d.cfc")

	if got, mine := r.joinHandlerFlight("/w/d.cfc"); mine || got != nil {
		t.Fatalf("a flight that never landed answered: %+v", got)
	}
}
