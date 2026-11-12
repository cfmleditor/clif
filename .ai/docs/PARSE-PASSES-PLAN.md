# Fewer passes over each document

How many times clif reads a document's text, why, and the order in which to
cut it. Measured 2026-11-10 on the `custom-tag-paths` branch.

## What was measured

Temporary counters (not committed) on `Scanner.reset` and `Scanner.next`,
`parser.Parse`/`ParseWithOptions`, the region split and tree-sitter's parse,
each attributed to the function that started it. "Passes" is tokens produced
divided by the tokens of one plain scan of the same text.

| Run | Passes | Full parses per component | Region splits |
|---|---|---|---|
| `unresolved` on ContentBox (724 files) | 16× the corpus | about 5× | 2,054 |
| `unresolved` on Mura (937 files) | 25× the corpus | about 10× | 3,567 |
| LSP `didOpen`, ContentBox `authors.cfc` | 12.1× the file | 6 (`ParseFile`) | 8 |
| `foldingRange` / `documentLink` | 1.0 each | 0 | 1 / 0 |
| an edit taken as a reparse | 3.8 | 1 | 1 |

Tree-sitter does not run in either path. A tag template barely uses the
scanner: tag regions are read by byte search, so its cost is the region split
and the tag walk, not tokens.

Two caveats on the LSP figures:

- The test session had no workspace index. Five of `didOpen`'s six parses
  were `ensureFileIndexed` parses of the files `authors.cfc` looks up (its
  base handler and injected services). A real session indexes those already.
- An edit inside a function body does not reparse (`ApplyEditResult` shifts
  and invalidates one function). The measured edit went down the deferred
  path (`flushReparse` → `ApplyFullReplace` → `reparseShallow`). That path
  runs the full `extractSignatures`, because the editor's parse is not
  shallow, whatever the name says.

## Why the text is read again

1. **Three token dialects.**
   - The main parse scans each script region with islands blanked and
     `interpStrings` on.
   - The post-parse passes (`scriptTokens`, `collectWrites`) scan the raw
     region.
   - The resolver scans the whole file, markup included, with a plain
     scanner.

   They tokenise `#f("x")#` strings and SQL or Java islands differently, so
   one stream cannot simply replace another (parser.md, "interpStrings" and
   "islands").
2. **Each consumer asks again, by path or text.** Its memo is per resolver
   view, or keyed differently, or capped below one workspace pass. For
   example, `tokenMemo` is capped at 2¹⁹ tokens, about one ContentBox pass,
   and clears everything when full.
3. **Whole parses made to read one field** (`.Extends`, `.Funcs`).
4. **Lazy per-function reparses** for functions the full parse already
   walked, because a memo keeps entries only for functions that had a result.

## Outcome (branch `parse-passes`, 2026-11-10)

| | Before | After |
|---|---|---|
| ContentBox, token passes | 15.9× | 13.2–13.6× |
| Mura, token passes | 25.5× | 14.9–16.2× |
| Mura, full parses | 3,151 | about 2,700 |

`unresolved --json` is byte-identical on all six corpora after every step.
Mura's range comes from `resolve.tokenize`'s shared memo under parallel
workers; it swung 15.6–18.9× before step 11 made the memo generational.

| Step | Result |
|---|---|
| 1. Plain parses with a cached twin | Done (`plainParseOf`) |
| 2. `AssumedProjectMappings` | Done: memo by source file, checked by re-reading what the walk read |
| 3. Producer plans | Done: lazy, one scan without the 4096-token false start |
| 4. `applicationScan` | Done: bounded memo by text |
| 5. Keep the main parse's script variables | **Not equivalent.** The global scanner and the main parse disagree on 975 of 14,949 corpus files |
| 6. `AllVars` from kept declarations | **Not equivalent.** 87 of 14,949 files differ even with the same function scopes (317 with the parse's scopes) |
| 7. Pre-fill empty function refs | **Not equivalent.** The lazy body parse finds refs in 1,309 of 48,268 functions the full parse filed none under |
| 8. Collection answers | Done: bounded memo by expression |
| 9. Folding and custom tags | Done: spans from the parse's regions, kept per content (0 of 14,949 files differ); byte gate on script regions |
| 10. Owner-level caches | Done for startup source facts and the WireBox module memos. `handlerCache` cannot be shared: views are one per application root, and a handler parse depends on its view's lookups |
| 11. Token memo | Made generational, which removed most of the run-to-run swing; sub-slicing file tokens was not attempted |
| 12. Prior hand-off | `managedPropertyMetadata` is memoised by text (Mura 2.5M to 0.5M tokens). Handing regions between parses was not done: it saves only byte scans |

Steps 5 to 7 failed for one reason: the parser has several walkers that do
the same job (find variables, find function scopes, read a body) and they
disagree on real files. Reusing one's answer in place of another's is then a
behaviour change, not a cache. Consolidating them means deciding, case by
case, which answer is right; the differential harnesses used here (parse
every corpus file, compare the two answers) are the way to find the cases.

## Second round (branch `parse-passes-2`)

- **`withoutComments` is kept per source.** A forwarded field was checked by
  tokenizing its file's whole text once per call. Mura went from 14.4× to
  13.6×.
- **`exprMemo` is split across 64 locks.** One lock made tassweb's workers
  wait: the scheduler's sleeps and wakes rose by 1.5s of CPU.
- **Handler parses repeated across views are left alone:** 274 on Mura, of
  676. 133 are the same view's parse of the same handler run twice by
  parallel workers. Waiting for the first would need deadlock detection
  across goroutines: the re-entry guard (`handler_reentry.go`) tracks each
  goroutine alone, so two workers could each wait on the other. The other 150
  are different views, whose parses read different inputs.

### Per editor action

ContentBox's `authors.cfc`, with the workspace indexed:

| Action | Passes |
|---|---|
| First open in a session | 11.9: the file's parse (about 2), plus the base handler's, injected services' and module configs' facts, read once |
| A second component's open | 2.4 |
| Reopening the first | 2.8 |
| Folding, document links | 1.0 each |
| Hover on a call | 0.9 |
| A keystroke inside a function | ~0 |
| A keystroke outside any function | 2.8 (the full reparse) |

### tassweb

46,000 files and 1.0GB; 6,930 components holding 657MB.

- **Tokens:** a scan tokenizes about 0.96× the text of a single pass over
  every file, because it never reads most templates.
- **Full parses:** 14,736 of 321MB, about one per component read.
- **No repeat passes to remove:** `unresolved` is byte-identical, wall time
  is the same within noise, and CPU is about 1% higher than the build before
  #356 (40.4s against 39.9s).
- **Memory is about 80MB higher** (1,430MB against 1,350MB). It is held by
  the memos: the text-keyed ones (application sections, managed metadata,
  module reads) keep whole files alive, and so do the token memo's two
  generations and the expression memos.
- **Fixed in the same branch.** The memos now hold no file's text:
  - The application-section, managed-metadata and module memos key by a
    `TextFingerprint` (length and two 64-bit hashes) and copy what they keep.
    The section memo keeps spans and rebuilds the constructor by a byte copy.
  - Whole files no longer pass through the shared token memo: `fileTokens`
    tokenizes into its own cache, and `withoutComments` walks a scanner.
    Before, about 20MB of whole-file token lists a scan held twice, or used
    once, evicted the expression windows the memo is for. Its generations
    halve.
  - `exprMemo`'s bound halves.

  Peak live heap on tassweb (from `gctrace`) is now level with the build
  before #356: 699MB and 693MB against 702MB and 690MB, with fewer
  collections. Peak RSS moves by about 70MB between runs of the same build,
  too much to compare on. Passes are unchanged within noise: Mura 14.1 to
  14.5, ContentBox 11.4 to 11.8.

## After the consolidation (2026-11-11)

The consolidation rounds (#363–#370) replaced the resolver's regular
expressions with token readers. Re-measured on `main` at 5d412ce against the
build after #362 (51e3b71), with the counters above (tokens produced over the
tokens of one plain scan):

| | #362 | `main` | `pass-regressions` |
|---|---|---|---|
| ContentBox token passes | 11.7× | 18.2× | 16.1× |
| Mura token passes | 14.2× | 20.6× | 17.4× |
| ContentBox CPU / wall / RSS (best of 3) | 2.00s / 0.33s / 140MB | 2.00s / 0.36s / 158MB | 2.01s / 0.34s / 148MB |
| Mura CPU / wall / RSS (best of 3) | 6.66s / 1.02s / 294MB | 6.84s / 1.13s / 331MB | 6.47s / 1.12s / 310MB |

- **Time did not regress; passes and memory did.** Tokenizing a file costs
  about what the regular expressions it replaced cost to run over it, so CPU
  stayed within 3%. Each round compared timing only against its own base,
  and passes were not counted, so the growth went unseen.
- **Measure time with the counters off.** A shared atomic counter bumped per
  token makes parallel workers contend, and the counted build ran 35% more
  CPU on `main` than on #362, which the uncounted builds did not.
- **Where the passes came from:** one reader after another tokenizing whole
  files on its own (`computedIncludeOffsets`, `fw1ViewCalls`,
  `muraObjectCalls`, `holdsElementRead`, the `#…#` expressions inside strings),
  three separate whole-file token caches holding the same tokens (by text,
  by path, on the parse), and word checks dropped or too broad.
- **Fixed in `pass-regressions`:** one entry per text (`codeMemo`'s
  `textTokens`: the raw tokens and the code lists) serves `textCode`,
  `rawTokens` (the resolver's `fileTokens`) and the parse's `fileTokens`; the
  word checks are back, with a tight one for computed includes
  (`mayIncludeComputed`); and the include walk skips a script region that
  names no include. `unresolved` is byte-identical on eight corpora.
- **What is left:** the resolver still tokenizes whole files the parse has
  already tokenized, in its own dialect (ContentBox: `viewRenders` on each
  handler, `rawTokens`), and `FunctionSpansOf` walks a file again where a
  parse's spans would do. Each reader also walks the cached tokens again for
  its own names; one per-file index of calls and writes, built once, would
  answer all of them.
- **The call index (`textCalls`, branch `code-index`).** Every `name (` in a
  text's code is indexed by name once, on the text's `codeMemo` entry, and the
  whole-file call readers look their names up instead of walking the tokens.
  Measured against `pass-regressions` (five alternating runs, uncounted): no
  difference, Mura 6.67s against 6.68s median CPU and ContentBox 2.06s against
  2.03s. In a Mura profile the token walks it replaces (`codeCalls`) do not
  register, and the resolver's whole-file tokenizing is about 2% of CPU; the
  parse (`ParseWithOptions`) is 31%. It is kept for the shape it gives the
  readers, not for speed. The index is built after the entry's size is
  counted, so `codeMemo`'s budget does not include it.

## Counted in CI (2026-11-12)

The counters are now part of the code, behind a build tag: the scanner's
`next` and the parse's two entries count tokens and parses only in a build
with `passcount` (`internal/parser/scan_counts*.go`), where a constant turns
the counting on; in every other build it compiles away.

- **`TestABatchScanReadsEachFileFewTimes`** (`cmd/clif/scan_passes_test.go`)
  runs an `unresolved` scan of the repo's fixtures and of the bundled
  framework stubs, and fails when the scan reads its text more times, or
  parses each file more times, than it does now (ceilings about 5% above:
  testdata 5.9 passes and 2.5 parses a file, the stubs 3.1 and 2.3). `make
  check` and CI's test job run it, alone, since the resolver's caches outlive
  a test. With `CLIF_PASSES_CORPUS` it reports any directories as well. An
  extra walk of each parsed file takes it from 5.6 to 7.7 and fails it.
- **`TestEditorActionPasses`** (`internal/server/editor_passes_test.go`) is
  measurement only: with `CLIF_PASSES_EDITOR` and friends set, it drives a
  server over a real workspace and reports each editor action's passes over
  the file acted on.

The denominator here is one plain scan of the CFML files under the scanned
directory, every template included, so the figures are lower than the
earlier tables' for a corpus whose scan skips templates.

### Batch scans, `main` (33161f1) against `parse-readers`

| | `main` | `parse-readers` |
|---|---|---|
| testdata | 5.33 passes, 2.45 parses a file | 5.55, 2.34 |
| framework stubs | 2.91, 2.18 | 2.91, 2.18 |
| ContentBox | 13.6, 3.16 | 14.1, 3.13 |
| Mura | 15.2, 3.30 | 13.7, 3.23 |
| tassweb (46,000 files) | 0.69, 0.25 (RSS 1,716MB) | 0.73, 0.25 (1,770MB) |

### Editor actions regressed during the consolidation

ContentBox's `authors.cfc`, the same harness at the build that recorded the
table above (7f35e71) and at `main`:

| Action | 7f35e71 | `main` |
|---|---|---|
| First open in a session | 12.0 passes, 6 parses | 20.4, 50 |
| A second component's open | 2.4 | 2.4 |
| Reopening the first | 2.8 | 3.7 |
| Folding | 1.0 | 1.0 |
| Document links | 1.0 | 3.0 |
| A keystroke inside a function | 0.03 | 0.03 |
| A keystroke outside any function (the full reparse) | 2.8 | 3.7 |

Bisected:

- **Document links, 1 to 3** at 8e0e130 ("Read document links with the
  attribute and script scanners"). A script region is walked by the
  custom-tag scan (`scriptCustomTags`) and again by the include walk
  (`scriptFileRefs`).
- **The reparse and reopen, 2.8 to 3.7** at aaa3ec2 ("Read includes with
  the document links walk"): every parse now runs that include walk over
  its script regions after its own walk.
- **First open, 12 to 20 passes** crept up over the rounds (16.2 by the
  regex batch, 19.2 by #370, 20.4 by #373), and its parses rose from 6 to
  50 with #373: the entry points' shallow parses (24) and the call readers'
  plain parses (20) of files the open reads facts from. Those parses are
  small, so passes rose by only 1.4.

The fix for the first two is the same as for every walker before it: the
parse's own walk records includes and links (an option, as `FindCalls`
records calls), and the separate link walk goes. That is the next step.

## The plan

Each step stands alone, lands as its own change, and keeps `unresolved --json`
byte-identical on the six-corpus set (RustCFML, Lucee, Mura, FW/1, ContentBox,
ColdBox). Each step is measured with the counters before and after. The
expected gains are estimates from the measured breakdown, to be confirmed by
those counts.

### Phase 1: remove repeats that need no new design

Cheap, low risk, mostly Mura and DI-heavy code.

1. **Plain parses that already have a cached twin.**
   - `producerEvaluation.constructedInstance` (producer_instance.go:39),
     `applicationStartupSources` (startup_application.go:81), `isDI1`
     (di.go:700) and `diExecutionSources`' plain parse (di.go:451) each call
     `parser.Parse` on text whose `sourceInfo` is in hand. Use
     `sourceInfo.plainParse`, which has the same options and is memoised by
     file stamp.
   - Fold `sourceInfo.startupAssignments`' own parse into `plainParse`
     (wheels_mapper.go:353-360).
   - Where only `.Extends` is read, call `declaredExtendsOf`.
2. **`AssumedProjectMappings` walks the same Application.cfc once per
   profile.** `mura` and `murawrm` share a source and markers, and so do
   `testbox`, `coldbox` and `preside`. It is uncached and called from
   `bootstrapMappings` (twice per app directory), `ApplicationParentLookup`,
   `mura_beans` and `mura_admin`.
   - Memo by source path within one call.
   - Then memo by (directory, initial mappings), still issuing the `read`
     calls: they record Discovery dependencies that watched-file
     invalidation needs.
   - Expected: most of Mura's 3.2M `mappingSourceState.content` tokens.
3. **`newProducerPlans` runs eagerly, with a false start.**
   - It is built whenever a `sourceInfo` is made, which is for nearly every
     file a lookup touches.
   - `producerTokens` gives up at 4096 tokens, after which the region is
     scanned again in full.
   - Fix: make it lazy (a `Once` in `sourceDerived`), and tokenise once
     without the cap.
4. **`applicationScan` with no memo.** It is called from about ten places,
   twice on the same file in `wirebox.moduleModelRoot` and
   `startup_application`.
   - Add a `Once` in `sourceDerived` for (masked, sections).
   - Call `ApplicationSections` once where both halves are needed.

### Phase 2: keep what the main parse already computed

These need care with the parser's invariants; each has a test that pins
them.

5. **Keep the main parse's top-level script variables.**
   - `newGlobalScriptParser` (`computeScopedVars`) re-parses every script
     region to find what the main parse found and discarded
     (`mergeScriptRegion` drops `sp.vars`).
   - Keep them per region, exactly as the tag side keeps `tagScopeVars`.
   - `TestBothVariableScansAgree` is what licenses this (parser.md, "two
     variable scans"); keep it green.
6. **`AllVars` and `ParseVars` split the file afresh.**
   - Every `pr.Declares` and LSP definition request runs a free
     `ClassifyRegions`, `findFuncScopesIn` and a full script parse of every
     region.
   - Build from `pr.Regions` and `pr.Scopes` while `regionsOf == Content`.
     This is the extra region split measured on definition for a template.
7. **Lazy body reparses for functions with nothing to find.**
   - `funcRefsMap` holds keys only for functions with refs, so
     `funcRefsUncached` re-parses every function with calls but no refs.
   - After a full parse, add an empty entry for each scope. Do this only
     where links were extracted and no `findCalls` is pending.
   - `TestLazyFuncRefsAreTypedAsTheParseTypesThem` pins the typing.
8. **Collection answers derived from the same write.**
   - `emptyCollection`, `collectionRead`, `factoryReturnChain` and
     `applyMemberBindings` start tens of thousands of scanners each, re-reading
     each write's right-hand side in every pass that asks.
   - Store the answers beside the write in `writesMemo`.
   - In `scanExpressionWrites`, tokenise a `<cfset>` body once and replay it.
     Script regions already do this, and the replay scanner exists.
9. **Folding and custom tag links.**
   - `foldingRange` runs `StructureSpans(content)`, which splits regions
     again. Pass it `pr.Regions` and memo the spans on the `ParseResult` by
     content, as `CustomTagCalls` and `textLinks` already are.
   - `scriptCustomTags` could skip any script region that spells none of
     `cf_`, `cfmodule` or `module`.

### Phase 3: share across parses and views

The largest gains, with the most design.

10. **Per-view caches moved to the owner.** `startupCache`, `wb` (WireBox)
    and `handlerCache` live on each `forCaller` view. A workspace with N
    application roots therefore rebuilds overlapping chains N times; Mura's
    354 `startupAssigns` parses are this.
    - Move them to `owner()`, keyed with `mappingsKey`.
    - Memo the per-source startup facts (shared assignments, variables,
      bean bindings, includes) by content: they depend only on the text.
    - `handlerParse`'s re-entry rules (resolvers.md, and its two tests)
      must hold.
11. **One whole-file token cache.**
    - These all tokenise whole files in the resolver's plain dialect:
      `allTokens(content)`, `fileTokens`, `wheelsMethods`,
      `variablesReferences`, `withoutComments`, `callsInText`.
    - Give them one owner-level cache keyed by content, sized above one
      workspace pass.
    - A window `content[from:to]` could be served as a sub-slice of the file's
      tokens, found by offset. Check the edges, where a window tokenised alone
      can differ (a string cut in two).
    - Keys cut from a file must be cloned, or they keep the file alive.
12. **Content-only facts handed from one parse to the next.** A batch scan
    parses each component under `IndexSpec` to index it, then under
    `BatchSpec` to check it, and `handlerParse` parses handlers again.
    - The two specs cannot merge: the index parse must not read another
      file's declared return (codemap.md), and the scan's `FuncLookup`
      settles refs in place.
    - Much does not depend on the spec, though: regions and line index,
      scoped variables, managed property metadata, tag writes and custom
      tag calls.
    - Add `ParseOptions.Prior *ParseResult`, borrowed when
      `Prior.Content == content`.
    - `managedTagMetadata` (2.6M tokens on Mura) should also find
      `<cfcomponent>` and `<cfproperty>` with the tag search rather than a
      token walk, and be memoised by content.

### Not planned: one token stream replayed into the main parse

The scanner can replay a token list, and two post-parse passes already do,
but the main parse cannot take one as things stand:

- Replay drops `LastBlockComment`, which doc comments are read from.
- In replay, `pos` is the next token's start, not the last one's end. Four
  callers depend on the latter.
- `bytesAfterPeek` reads `peekPos` as a byte offset, but in replay it is a
  token index.
- The main parse scans blanked text, not raw text.

Keeping tokens per `ParseResult` would also cost about 32 bytes a token for
every cached parse, against the pooling the scanner was built around
(scanner.go, collection_returns.go: "nothing the passes keep refers to a token
list") and the size and allocation budgets (`TestParserStructsKeepTheirSize`,
`TestParseStaysWithinItsAllocationBudget`). The steps above cut the repeats
between parses instead, which is where most of the passes are.

## How each step is checked

- **The counters**, put back for the measurement and removed after. A sketch
  of what they did:
  - In `lex`: a per-caller map of calls and bytes, and a per-scanner token
    counter.
  - Hooks in `Scanner.reset` and `next`, `Parse`, `ParseWithOptions`,
    `classifyRegions`, `ClassifyRegionsIdx` and `language.Parse`.
  - A report printed after the subcommand runs.

  Report passes before and after on ContentBox and Mura, and on the LSP
  sequence (`didOpen`, folding, links, hover, definition, edit, completion)
  for one component and one template.
- **Six-corpus `unresolved --json`, byte-identical.** Diff with
  `unresolved-diff` when it is not.
- **The tests named in each step**, `go test -short ./...` and `make lint`.
  The parser's race run (about 10 minutes) before any step that changes
  shared state.
- **Benchmarks** in `internal/parser` for steps 5 to 8, and a wall-time
  comparison of `unresolved` on tassweb for steps 10 to 12.
