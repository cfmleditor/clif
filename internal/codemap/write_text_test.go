package codemap

import (
	"strings"
	"testing"
)

// writeTextFixture is a small map with every section WriteText can print: a
// main graph, a detached island, a utility hub beside an application hub, a
// cycle, and unreferenced functions one of which is utility.
func writeTextFixture() *Map {
	fn := func(file, name string, line uint32) Node {
		return Node{ID: file + "::" + strings.ToLower(name), Kind: KindFunction, Name: name, File: file, Line: line}
	}

	page := Node{ID: "index.cfm", Kind: KindFile, Name: "index.cfm", File: "index.cfm", Entry: true}
	svc := fn("svc.cfc", "save", 4)
	log := fn("log.cfc", "write", 1)
	log.Utility = true
	a, b := fn("loop.cfc", "ping", 2), fn("loop.cfc", "pong", 9)
	dead := fn("dead.cfc", "helper", 0)
	deadLog := fn("deadlog.cfc", "trace", 6)
	deadLog.Utility = true
	lonely := fn("lonely.cfc", "orphan", 3)

	call := func(from, to string) Edge { return Edge{From: from, To: to, Kind: EdgeCalls, Count: 1} }

	m := &Map{
		Root:  "/ws",
		Level: LevelFunction,
		Nodes: []Node{page, svc, log, a, b, dead, deadLog, lonely},
		Edges: []Edge{
			call(page.ID, svc.ID), call(svc.ID, log.ID), call(page.ID, log.ID),
			call(a.ID, b.ID), call(b.ID, a.ID), call(dead.ID, log.ID),
		},
		Stats: Stats{
			Files: 6, Functions: 7, CallSites: 10, Resolved: 6, Dynamic: 1, Unresolved: 2, Builtin: 1,
			Failed: 1, Unreadable: 2, Routes: 4, RoutesUnresolved: 1, Utility: 2, BuildMillis: 12,
		},
	}
	m.Annotate()

	return m
}

func TestWriteTextPrintsEverySection(t *testing.T) {
	for _, c := range []struct {
		name  string
		m     *Map
		limit int
		want  string
	}{
		{"every section", writeTextFixture(), 10, wantFull},
		{"a limit of one cuts each list", writeTextFixture(), 1, wantLimit},
		{"an empty map prints only its counts", &Map{Root: "/empty", Level: LevelFile}, 10, wantEmpty},
	} {
		var b strings.Builder
		if err := c.m.WriteText(&b, c.limit); err != nil {
			t.Fatal(err)
		}

		if b.String() != c.want {
			t.Errorf("%s:\n%s\nwant\n%s", c.name, b.String(), c.want)
		}
	}
}

var (
	wantFull = `Code map of /ws (function level)

  8 nodes, 6 edges
  6 files, 7 functions
  10 call sites: 6 resolved, 1 dynamic, 2 unresolved, 1 builtin
  60.0% of call sites resolved to a definition
  1 files could not be scanned (an internal error, recorded on stderr or in the crash log)
  2 files unreadable or binary
  4 framework routes: 3 resolved (75.0%), 1 not
  2 nodes marked utility
  4 islands, 5 nodes no entry point reaches
  built in 12ms

Detached islands (3 besides the main graph):
  island 3: 2 nodes, rooted at loop.cfc:3 ping()
  island 1: 1 nodes, rooted at deadlog.cfc:7 trace()
  island 2: 1 nodes, rooted at lonely.cfc:4 orphan()

Most depended on (utility excluded) (3):
      1  loop.cfc:3 ping()
      1  loop.cfc:10 pong()
      1  svc.cfc:5 save()

Most depended on, utility (1):
      3  log.cfc:2 write()

Cycles (1, largest first):
  2 nodes: loop.cfc::ping -> loop.cfc::pong

Unreferenced (3 — candidates, not a verdict; 1 of them utility; see unresolved count above):
  dead.cfc:1 helper()
  deadlog.cfc:7 trace()  [utility]
  lonely.cfc:4 orphan()
`
	wantLimit = `Code map of /ws (function level)

  8 nodes, 6 edges
  6 files, 7 functions
  10 call sites: 6 resolved, 1 dynamic, 2 unresolved, 1 builtin
  60.0% of call sites resolved to a definition
  1 files could not be scanned (an internal error, recorded on stderr or in the crash log)
  2 files unreadable or binary
  4 framework routes: 3 resolved (75.0%), 1 not
  2 nodes marked utility
  4 islands, 5 nodes no entry point reaches
  built in 12ms

Detached islands (3 besides the main graph):
  island 3: 2 nodes, rooted at loop.cfc:3 ping()

Most depended on (utility excluded) (1):
      1  loop.cfc:3 ping()

Most depended on, utility (1):
      3  log.cfc:2 write()

Cycles (1, largest first):
  2 nodes: loop.cfc::ping -> loop.cfc::pong

Unreferenced (3 — candidates, not a verdict; 1 of them utility; see unresolved count above):
  dead.cfc:1 helper()
`
	wantEmpty = `Code map of /empty (file level)

  0 nodes, 0 edges
  0 files, 0 functions
  0 call sites: 0 resolved, 0 dynamic, 0 unresolved, 0 builtin
  0 islands, 0 nodes no entry point reaches
  built in 0ms
`
)
