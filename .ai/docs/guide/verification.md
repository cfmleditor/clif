# Verification discipline

_How to confirm a change: mutation checks, corpus diffs, benchmarks, scaling tests, and the lessons behind them. Moved out of CLAUDE.md unchanged._

Four habits that have each caught a real defect in this codebase, and whose
absence has each let one through:

- **Confirm every new test fails with its fix reverted.** Comment out the change,
  run the test, see it fail, put it back. Tests that passed either way have been
  written here repeatedly — a `lowercaseTags` test with an all-lowercase fixture,
  a `scanWorkspace` test asserting indexing that never happens, a bare-block test
  whose fixture did not reproduce the bug it was named for.
- **Diff the corpus per file, not by totals.** `make corpus BASELINE=<earlier report>`
  does this. A change that breaks one file and fixes another leaves every column
  identical.
- **Run `go test -short -race ./...` before pushing.** CI's race job found a
  pre-existing data race on the workspace roots that a non-race run could not.
- **Print actual output before writing an expected string.** Hand-counting the
  indentation of a nested fixture is wrong more often than right.

**Cost that scales with the workspace hides in per-file operations.** The
things that run once per file, or once per keystroke, are where an
index-sized scan turns into a quadratic: `removeFileEntries` walked every
name bucket on every index write, which made a 5,624-file workspace scan
take 34s and allocate 6.6GB (0.22s and 95MB once it went through
`fileFuncs`/`fileRefs`); `workspace/symbol` materialised all 40,000
definitions per keystroke to return a few dozen;
`LookupComponentRefInFile` — which hover, definition and completion each
ask on the keystroke — searched the variable name's bucket for the file
rather than the file's refs for the name, and the names it gets asked
about (`svc`, `dao`, `qry`) have one entry per file in the workspace.
All three were invisible in a unit test and obvious in one profile. The
general form is the index's read/write asymmetry (see the `snapshot`
comment): make an operation cost what it *uses*, not what the index
*holds*. Reach for
`go test ./internal/index/ ./internal/server/ -bench . -benchmem -run '^$'`
before assuming an LSP path is cheap — the benchmarks there load an index
the size of a real workspace, which is the axis a handler benchmark on an
empty server cannot see. Pin the shape, not the clock, when it matters:
`TestIndexFileFromResultDoesNotScaleWithIndexSize` compares allocations at
two index sizes, so it fails on the regression rather than on a busy runner.

**A rule that asks a question of a file's text reads the code, not the text.** The resolver's
questions of CFML source (does this file write `NAME`, call `setView`, declare `init`, guard with
`structKeyExists`, hold a `<cfloop>`) were regular expressions over the whole text, and then hand
scanners made to match them exactly for speed (`text_rules.go`, now deleted). Both read a
commented-out statement or a word inside a string as code. They now read what the parse reads: the
scanner's tokens (`allTokens`, `codeLists`/`codeCalls` in `code_calls.go`, `codeWrites` in
`name_writes.go`), `parser.CFTags` for tags, and `ParseResult.FunctionSpans`/`parser.FunctionSpansOf`
for functions; `.ai/docs/SCANNER-CONSOLIDATION.md` has each step. **Do the next one the same way:** a
test that puts a commented-out or in-string case beside a real one and fails on the old reader, and
the `unresolved --json` corpus diff, every changed finding explained. Where a reader must keep its
old answer exactly (a fixed-shape expression over one value), keep the expression in the test as the
oracle and compare on generated texts, as `expression_shapes_test.go` and the parser's
`text_shapes_test.go` do. Regexes that remain are user patterns (`componentResolvers`), HQL, and a
few built from a value the code spells and matched against a name, not against CFML.

**Search for the entries you are holding; do not sweep the bucket for them.**
Both index writers on the keystroke path — `removeFileEntries` on every
re-index, `ShiftLines` under the write lock on every edit that changes a line
count — looped over every entry filed under a name asking "is this one of
mine". For the names workspaces share, that bucket is one entry per file, so
the loop was the workspace and the answer was a handful. Worse, the per-entry
work was a *call* (a closure, then a `slices.Contains` over a one-element
group), not a compare. Inverted — `slices.Index` per entry actually touched —
the comparisons are the same and a 4,000-file shared-name workspace goes from
164µs to 28µs for a re-index and 101µs to 24µs for a shift. Neither is *flat*;
flat needs a pointer-to-position map beside the buckets, a few megabytes held
for the life of the index, and the tests say so rather than claiming otherwise.
**Measure such a thing on a file from the middle of the workspace** — the first
file's entries sit at the front of every bucket, where a search finds them at
once, so a sweep and a search look alike on it.

**A handler benchmark is not a request.** For the two responses that carry a
lot of items, encoding the answer dwarfs computing it: `workspace/symbol` on a
5,000-file index is 0.24ms of handler against 3.02ms of marshal, and completion
is 0.002ms against 0.63ms. So a saving inside either handler is rounding error,
and the only thing that makes them cheaper is sending fewer bytes.
`BenchmarkWorkspaceSymbolWithMarshal` and `BenchmarkCompletionWithMarshal`
measure them end to end; the handler-only benchmarks beside them are for
changes to the handler, not for what the request costs. `.ai/docs/PERFORMANCE-GAPS.md`
records the three costs measured this way and not acted on, with what each
option would save and what it would cost the user. Capping `workspace/symbol`
is the obvious fix and is deferred on priority rather than overlooked; section
2 says what to know before picking it up, including that the cheap first step
is measuring a real workspace rather than building the cap.

**Benchmarks in one process contaminate each other, so isolate before believing
a regression.** A run of the whole benchmark set here reported `ScopesToFuncRanges`
69% slower, `documentSymbol` 29%, `workspace/symbol` 27% and `Keystroke` 21%
*faster* — none of which were real. The allocation-dominated ones inherit
whatever heap the benchmarks before them left live, and `benchLoadedServer`
leaves a 5,000-file index. Re-run the single benchmark on its own, both sides,
before acting on any of it. The same caution applies to wins.

**Compare against the branch point, not against your last build, and run the two
binaries alternately.** Five parser fixes in one round were each reported at
around +1% on the tag benchmarks, because each was measured against whatever
binary was left in the scratch directory rather than against `origin/main`.
Re-measured from a clean main baseline the round was **+9.4%** on the plain tag
parse — the editor's keystroke path — and the cause was a byte-at-a-time
`tagEndIndex` running over every tag in the file. No single reading had shown it.

**Then the corrected figure was wrong the other way.** The fix for that was
recorded as -0.7%, measured by running all of one binary's samples and then all
of the other's. Interleaved — one sample of each, alternately, twenty-five times
— the same pair reads **+10%**, because the machine drifts between the two runs
by more than the effect. A sequential A-then-B comparison is not a measurement,
however many samples each half has. Interleaving also settles the contamination
warning below: a `git worktree` of `origin/main` built into the scratch
directory gives a second `.test` binary to alternate with.

So: **keep a `main` build in the scratch directory, re-measure the whole branch
against it, and alternate**; treat min, p10 and median disagreeing as "not yet
measured" rather than as a result. A per-commit bisect then splits a cumulative
cost honestly — which is how the tag walk's +10% came apart into +3.2% for
bracket indexes, +5.1% for stepping over spans and +5% for the quote-aware tag
end, none of them a regression on its own, and how `tagEndIndex`'s two-count
fast path was shown to take the whole thing back to about +4%.

**A parse is held to an allocation budget, because a count can gate a build
and a time cannot.** `TestParseStaysWithinItsAllocationBudget` parses the
benchmark fixtures and fails over a quarter above the count each budget was set
at. bcac43a took a script component from 442 allocations to 3,713 and a sixfold
slower parse, and nothing failed for two weeks: the benchmarks that showed it
gate nothing. Raise a budget in the commit that needs it, saying why; lower it
after a large saving. It skips under `-race`, where `sync.Pool` drops items.

**A cost that scales with the document wants a scaling test, not a timing one.**
Every defect behind the three-second go-to-definition was invisible to the tests
that existed, because all of them returned the right answer. `routepkg.Scan`
lowercased the whole remainder of the file to test a four-character prefix, once
per `?` or `&` — 1.8GB and three seconds on a 64,000-line component, with the
refs correct throughout. `routeAtPosition` scanned the whole document and kept
only the cursor's line. Document links rescanned on every request.

The tests that catch these compare a **ratio**, never a clock:
`TestScanScalesLinearly` gives eight times the input and fails above three times
the input ratio (the defect was 63x, near-exactly quadratic);
`TestScanDoesNotAllocateOverContentWithNoRoutes` grows only the content that
matches nothing and fails if allocation follows it (the defect was 97x);
`TestRouteLookupDoesNotScaleWithDocumentSize` appends 460KB after the cursor and
fails if the lookup notices. A busy runner moves both numbers together, so they
fail on the regression rather than on the machine.

Two traps, both of which this file's tests fell into first. A route test needs
`Routes.Enabled()` to be *true* — it wants a source **and** a controller or view
rule, and without the second half every route path short-circuits and the test
passes whatever the code does. And a window test needs both documents to present
a **full** window, or it compares a five-line window against a fifty-line one and
fails for that instead.

**A worker goroutine needs a recover of its own.** A request handler's
recover (`Handler`, `safeGo`, the timers) covers its own goroutine only, so a
panic on a goroutine a command starts underneath it ends the language server,
and in daemon mode every editor connected to it. Every pool the server can
reach recovers per file: `resolve.forEachFile` (which returns the files that
failed, so a caller does not read one as holding nothing), the unresolved scan
(`scanFileRecovered`, which reports the file as one it could not scan), the
code map (`Stats.Failed`), reference search, the workspace indexing writer,
CFLint's workers (the panic becomes the run's error) and each daemon session.
`cflog.Recovered` prints a line to stderr when there is no logger and no crash
file, which is every command-line run. A new pool gets the same, and a test
that panics on one file (`TestAScanSurvivesAPanicOnOneFile` and its siblings).

**What no recover catches, a supervisor does** (`internal/supervise`,
`cmd/clif/supervise.go`). A stack overflow, a concurrent map write and running
out of memory end the process wherever they happen, so the `clif` an editor
starts runs the server as a child (`CLIF_SUPERVISED=1`) and relays the
session, remembering the initialize request, the configuration and folders,
and each open document's text (applied through `lspdoc.ApplyChanges`, the
server's own code, so the two agree byte for byte). A child that exits unasked
is replaced and the session replayed; its unanswered requests get
ContentModified; a registration the editor already holds is answered by the
supervisor, since registering an id twice is an error to an editor; more than
five exits in three minutes and it gives up. It also closes the child's stdin
after `exit`, which the server otherwise waits for. A message from the editor
during a restart is queued, except document sync, which the replay already
carries. `TestACrashedServerIsRestartedWithTheSessionReplayed` drives the whole
cycle against an in-process fake; `CLIF_NO_SUPERVISOR=1` runs the server
directly.

**A close tag reaches the walk as `</`, not `<c`.** The `</cffunction>` check sat in
`handleCFTag`, which the walk calls only for a tag whose second byte is `c`, so a tag
function's scope ran on until the next `<cffunction>`. Every top-level `<cfset>` after one
was the function's: its pending calls typed a variable of the function, its unscoped names
were read against the function's `var`'d locals, and a qualified call the tag parser
records itself named the function as its caller — Lucee's admin pages, which declare
`addZero()` first, credited it with every call on the page. `closeTag` handles it now;
`TestTagFunctionScopeEndsAtItsCloseTag` fails on each of the three if it goes.

**Where a tag holds an expression, hand it to the script parser.** The tag
parser matches tags and pulls attributes out with string searches; it has no
expression parser and should not grow one. A `<cfif>` condition, a `<cfelseif>`,
a `<cfreturn>`, a `<cfset>` and a `#...#` span all go to a `scriptParser` that
keeps only the calls, which is what a `<cfscript>` body has always done. One
implementation of "what is a call" then serves both syntaxes, and a fix to it
reaches tag files for free. On the six-project corpus that was 3,100 call sites
for `<cfset>` alone — a bare `<cfset arrayAppend(a, b)>`, a call after a
concatenation, a nested call in an argument list, any scope-prefixed left-hand
side — and 207 more for a script tag's attribute *value*, where the identifier
was consumed as a plain token and only its argument list was scanned, so
`array=structKeyArray(rows)` recorded what was inside the parens and never the
call. .ai/docs/PARSER-GAPS.md has the measurement and what is still missing.

**Only `<cfset>` tops up, and that distinction is the subtle one.** Its string
paths run first and carry the refs and pending calls that decide what a variable
now holds, resolving a receiver against *this file's* refs in a way a fresh
sub-parse cannot — so the sub-parse subtracts the count already recorded for
each name on the line and adds only the excess. Counting rather than testing
presence is what keeps `<cfset x = f() + f()>` at two. Doing the same for a
`#...#` span is *wrong*: `#getColdBoxSetting("a")# #getColdBoxSetting("b")#` is
two calls on one line, and the shared merge with the tally on cost the second
one. **A unit test on one span could not have caught that; the corpus did.**

**The grammar is a second opinion on the parser, and `make gapcheck` asks it.**
`internal/parser` and the tree-sitter grammar are independent implementations of
"what is a call", so where they disagree one of them is wrong. Every call-losing
defect fixed here so far was found by hand-probing constructs one at a time;
this asks the question over a corpus instead. It found, in minutes on the repo's
own 40 fixtures, a class nobody had probed: in tag syntax a call in a `<cfif>` condition, a `<cfreturn>` or chained onto an
instantiation was once recorded nowhere; those are closed, and `make gapcheck` now
holds the fixtures to the deliberate differences alone.

Three things about it, each of which cost a wrong conclusion first:

- **Calibrate before believing it.** The first run reported nine calls the
  parser had "invented" — every one a `queryExecute`, which the grammar gives a
  `query_expression` node of its own so SQL can be injected into it. That was
  the oracle's mistake, not the parser's.
- **Deliberate differences are not gaps.** `createObject(…)` is a `ComponentRef`
  to the parser and a call to the grammar, on purpose. `expectedDifferences`
  records both kinds with a reason each, and the test fails both on a *new*
  difference and on a listed one that no longer differs — so a gap cannot be
  closed unnoticed and the list cannot rot into decoration.
- **The declaration axis is much weaker than the call axis, and it is measured
  rather than assumed.** Of the phantom-variable shapes fixed in this parser,
  the grammar disagrees with the old behaviour on exactly one
  (`component extends=`). For the rest it agrees with the *bug*: `force` in
  `svc.save(force = true)` is an `assignment_expression` to the grammar too, and
  so is `name` in `query name="q"`. Those distinctions are semantic, not
  syntactic, so a syntax oracle cannot arbitrate them.
  `TestDeclarationAxisIsWeakerThanTheCallAxis` pins that ratio.

It cannot check resolution at all — the grammar has no idea what a dot-path
points at — and it is only as good as the corpus: the repo's fixtures contain
zero interpolated calls, which is why the size of that gap is still unmeasured.

**A `#...#` span is not a tag boundary, and the two scans that say where one
ends are deliberately different.** Markup inside a span is an argument:
`#ETH.author( content = "<strong>x</strong>" )#` was split at the `<`, losing
that call and — through the hashes left over — the call after it. The walk steps
over a span now, under three rules each measured by removing it: a span's hashes
**pair** whether or not its contents are stepped over (removing this costs 141
files, and it is the one-line mistake that made an earlier attempt take missed
sites from 1,798 to 3,025); a span with **no `(`** does not hide a tag, so a
stylesheet's `#sidebar ul {` stays markup; and a span with **unbalanced quotes**
does not either, so `$( '#search' ).typeahead(` stays a jQuery selector. Hashes
interpolate only inside `<cfoutput>` and in a tag's attributes and the walk
tracks neither, so these are heuristics — chosen so that being wrong costs a
call rather than a tag.

**One narrow case follows the nesting in the walk too** (`callSpanEnd`): a span that
*opens as a call* (a name or dotted name, then `(`), whose strings all close and
that holds no CF tag is closed through its strings. `#ETH.text( "Don't <b>x</b>
##f2dede" )#` was split at its `<` twice over — the quote-parity test counts `'` and
`"` separately, so an apostrophe in prose unbalanced it, and the next hash is the
first half of the `##` — and the call after it was lost with the hashes it
mis-paired. Each of the three conditions is one of the ways the wider attempt lost
tags, and it is asked only of a span the walk's own test declined, which is what
keeps the common span at its old cost (the first version asked every call-shaped
span and parsed a span-dense file 10-13% slower);
`TestAnInterpolatedCallMayHoldProseAndEscapedHashes` has a case for each and
the two guard shapes (a colour, an unclosed string) beside them. Checked per file
over ColdBox, TestBox, FW/1, ContentBox and Lucee (6,438 files): seven improved,
none changed otherwise. **The call scan's `matchingHash` also lets a span hold a
span of its own**: a hash met outside a string with a parenthesis still open is
another span's opening, so `#event.buildLink( a & #encodeForHTMLAttribute( id )# )#`
is one span and both calls are found (`matchingHashAt`, to `maxHashNesting` deep;
`TestAHashSpanMayHoldAHashSpanOfItsOwn`).

`interpolatedSpans` follows CFML's *nesting* instead — a string inside a span may
hold a span (`#html.elixirPath( root='#cb.themeRoot()#/inc' )#`), which is what
`Scanner.scanHashExpr` already does for CFScript. **Do not make the tag walk
share that rule**: it fixed one corpus site and broke thirty-six, because
skipping quoted strings runs a span much further in markup. A wrong span costs
the walk a tag and costs `interpolatedSpans` a call, and they are tuned for their
own error.

**A `RegionSkip` is a literal `<script>` block, and dropping it dropped its
interpolation.** The region exists so JavaScript is never fed to the CFScript
scanner, and the region was then not parsed at all — but a `<script>` body
inside `<cfoutput>` is exactly where a page writes
`var id = "#prc.oContent.getContentID()#";`. It goes to the tag parser now and
needs no mode of its own: `findScriptSkipSpans` only makes a span of a block
holding no `<cf` tag at all, so the walk can find nothing in one *but* its
interpolation. A flag restricting it to the spans was written first and removed,
because it measured identically over the whole corpus *and* against the
tag-shaped-text-in-a-JS-string case it was written for — which is not a skip
region precisely because it contains `<cf`. 124 sites over 34 files with no
invented call.

**A lone `#` in a CFML string is invalid, Lucee tolerates it, and the scanner
cannot tell it from interpolation.** `md.append( "# ColdBox Performance Report" )`
opens a span that closes at the next *real* interpolation two lines below,
swallowing every call between. Requiring a span to open and close on one line
looks obviously right and costs 45 sites net over the corpus, because a span
that genuinely wraps a line is how ContentBox writes a form field —
`#html.inputField(\n name = "authorEmail",\n …\n)#`. Both shapes cross lines,
hold quotes and hold a `(`; telling them apart needs to know the first `#` was
never interpolation, which is a fact about the enclosing expression rather than
the string. .ai/docs/PARSER-GAPS.md §4.2 has the measurement.

**A hand-maintained parallel list wants a reflective test.** Wherever the same
names must appear in two or more places, enumerate them in a test rather than in
a comment. `mergeFormatting`'s field list had one and it caught two omissions
while the untested hops beside it stayed silent; the config and daemon chains
(`TestFormattingKeysReachTheFormatter`, `TestResolvedFormattingReadsEveryKey`) and
the parser's two scope-dispatch switches
(`TestBothDispatchSwitchesHandleEveryScope`) now have theirs. Prefer a structural
check when the rule itself is structural: a behavioural test only catches the
cases whose absence has an observable it happens to assert.
