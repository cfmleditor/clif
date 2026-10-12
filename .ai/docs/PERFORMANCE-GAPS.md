# Performance gaps: costs identified and not acted on

Costs found while profiling the per-keystroke and batch-analysis paths. Sections
2, 3, 6, 7 and 8 are done and record their measurements. Section 4 is settled as not
worth doing without a real-workspace profile; the remaining follow-ups are
called out in sections 3, 5, 7 and 8.

This file exists so they are not rediscovered from scratch, and so the reasons
are on record rather than remembered. Every number below is measured, and the
method is in section 1 — including the mistake that made one of them look
eighty times smaller than it is.

## 1. Method, and why the earlier numbers were wrong

Measured on a synthetic index of 5,000 files declaring 8 methods each — 40,000
definitions — built by `benchLoadedServer` in
`internal/server/perf_bench_test.go`. Each figure is the minimum of several
runs of a single benchmark run **on its own**; see the contamination note in
CLAUDE.md for why that matters.

**The handler is not the request.** A handler benchmark measures the answer
being computed and not the answer being sent, and for these two the sending is
almost all of it:

| | handler | marshal | handler's share |
|---|---:|---:|---:|
| `workspace/symbol`, query `method3` | 0.24ms | 3.02ms | 7% |
| `workspace/symbol`, query `m` | 1.6ms | 24.4ms | 6% |
| `textDocument/completion` | 0.002ms | 0.63ms | 0.3% |

`BenchmarkWorkspaceSymbol` measures only the handler, which is how the
`workspace/symbol` cost below was first recorded as 296µs when the request
costs 3.3ms. `BenchmarkWorkspaceSymbolWithMarshal` now measures it end to end,
next to the `BenchmarkCompletionWithMarshal` that already did.

The consequence for anything on this page: **the only way to make these
requests cheaper is to send fewer bytes.** Work saved inside the handler is
rounding error against the encode.

Two smaller notes from the same measurements:

- The wire already uses `encoding/json/v2` — `go.lsp.dev/jsonrpc2`'s default
  codec. There is no encoder swap left to make. On this payload v2 is in fact
  *slower* than v1 (748µs against 625µs for the completion list), so the
  benchmarks, which marshal with v1, understate the real cost slightly rather
  than overstating it.
- The completion payload has no redundant fields to drop. `sortText` carries a
  sort-priority prefix (`8abs`), so it is not the label; `insertText` is a
  snippet (`abs(${1:number})`), so it is not the label either; `filterText` is
  already omitted on every item. There is no free saving here — only the two
  fields in section 3.

**Synthetic-index caveat.** Every definition in the bench index is named
`method0`–`method7`, so the query `m` matches all 40,000. A real workspace has
varied names and a one-character query would match some fraction of them. The
shape is what carries over, not the multiplier: matching is a case-folded
*substring* test over every distinct name, so short queries match a large share
of a real workspace too, and short queries are the ones always sent.

## 2. `workspace/symbol` bounds transient short-query results — done

Done with a 1,000-result bound for zero-, one- and two-character queries. Those
are transient prefixes sent on the way to a specific search; queries of three
or more characters remain complete. Before applying the bound, candidates are
ranked by exact match, prefix match, name, URI and line. A bounded max-heap
retains the best 1,000 without sorting the complete match set, so results do
not depend on parallel indexing order.

Measured end to end on `BenchmarkWorkspaceSymbolWithMarshal/query_m`, three
runs before and after:

| | symbols | JSON | time/op | allocated/op | allocations/op |
|---|---:|---:|---:|---:|---:|
| unbounded | 40,000 | 6,211,121 B | 32.0–32.6 ms | 11.3–13.3 MB | 40,010–40,013 |
| bounded | 1,000 | 155,279 B | 5.5–5.9 ms | 600–607 KB | 1,009 |

That is a 97.5% payload reduction, about 5.8x lower end-to-end latency and
about 95% less allocation in the synthetic worst case. The remaining text in
this section records the design analysis that led to the bound.

The earlier measurements are kept in full for comparison.

`handleWorkspaceSymbol` (`internal/server/symbol.go`) turns every matching
definition into a `protocol.SymbolInformation` and returns all of them. The
picker issues a request per keystroke, and the first keystrokes are the short
queries that match the most:

| query | matches | JSON | total |
|---|---:|---:|---:|
| `m` | 40,000 | 6.2MB | **26ms** |
| `me` | 40,000 | 6.2MB | 26ms |
| `method` | 40,000 | 6.2MB | 26ms |
| `method3` | 5,000 | 776KB | 3.3ms |
| `zzz` | 0 | 2 B | 0.003ms |

A symbol is 151 bytes on the wire:

```json
{"name":"method3","kind":12,"location":{"uri":"file:///ws/pkg0/File0.cfc",
 "range":{"start":{"line":25,"character":0},"end":{"line":25,"character":0}}}}
```

### Options

**(a) Cap the result count.** The standard answer — gopls caps at 100 by
default (`symbolLimit`), rust-analyzer at 128 for inexact queries. Measured
against the same index:

| cap | matches sent | JSON | marshal |
|---|---:|---:|---:|
| none | 40,000 | 6.2MB | 24.4ms |
| 1,000 | 1,000 | 154KB | 0.60ms |
| 100 | 100 | 15KB | 0.06ms |

A cap of 100 is roughly **13x** on the `method3` query and **400x** on `m`.

The cost is real: a symbol past the cap cannot be found, and with an arbitrary
cut there is no reason the ones kept are the ones wanted. If this is done, the
cap should be applied to a *ranked* set — the same `cfpath.URIDistance` and
lowest-URI ordering `LookupPreferred` and `definition.go` already use — so the
survivors are the near ones rather than whichever bucket the parallel scan
filled first. Ranking 40,000 entries costs something, but far less than
encoding them.

**(b) LSP 3.17 `WorkspaceSymbol` with a URI-only location**, resolved through
`workspaceSymbol/resolve`, gated on the client's
`workspace.symbol.resolveSupport`. Drops the range from each entry — 151 bytes
to about 75 — so roughly half, with **no result lost**. It does not solve the
problem on its own (6.2MB becomes ~3MB, 26ms becomes ~13ms) and it needs a new
handler, but it composes with (a) and it costs the user nothing.

**(c) Require a minimum query length.** One or two characters return nothing
until the query is worth answering. Cheap, but it makes the picker feel broken
rather than fast.

### What to know before picking this up

The cost is real, the fix is known, and it is a priority call rather than a
technical one. Three things about the shape of it, so whoever gets to it is not
starting from the top:

- **A cap is the only option that touches the real cost, and it is not free.**
  It buys latency in a feature whose entire job is *finding things*, by making
  some things unfindable — and a symbol search that is fast and silently
  incomplete is a harder failure to notice than one that is slow, because the
  user cannot tell "no such symbol" from "past the cut". Ranking before the cut
  makes the survivors plausible rather than arbitrary, which is the difference
  between a defensible cap and a careless one, but it does not make the cut
  lossless. That is what makes the default a judgement rather than a constant:
  gopls settled on 100, rust-analyzer on 128, and the repo's convention would
  be a documented default with a config key over it.
- **The headline number is synthetic.** Every definition in the bench index is
  named `method0`–`method7`, so the query `m` matches all 40,000 and the 26ms
  is a worst case built to be one. A real workspace has varied names; the shape
  carries over (case-folded substring matching over every distinct name, so
  short queries match a large share) but the multiplier does not, and nobody
  has measured a real one. Measuring that first would say how urgent this
  actually is, and is much cheaper than building the cap.
- **The picker is used deliberately, not continuously**, which is the main
  reason this can wait while the parse and index paths could not. It is a
  per-keystroke cost inside a burst the user chose to start; those run while
  someone is simply typing code. A slow symbol search is felt once per search.

Option (b) — the LSP 3.17 `WorkspaceSymbol` with a URI-only location — is the
one piece with no downside to the user, and is the place to start if this is
picked up: it drops the range from every entry with no result lost. On its own
it halves a number that needs an order of magnitude, and it wants a client
advertising `workspace.symbol.resolveSupport` to be worth the handler, which
is why it is not a fix by itself.

## 3. Completion sends documentation and detail for every item — done

Done as option (a) below, gated on the client's `resolveSupport` as described
there (`internal/server/completion_resolve.go`). A client listing
`documentation` and `detail` there is sent the built-in and member-function
items without them, and `completionItem/resolve` fills them in for the item it
highlights; a client that does not is sent the full items, exactly as before.
Only those two lists are deferred: they are the only items carrying
documentation, and the only ones whose text can be found again from what comes
back on resolve.

Measured with `BenchmarkCompletionWithMarshal`, run alternately, three rounds:

| | JSON | handler + marshal | allocated |
|---|---:|---:|---:|
| `full` | 322KB | 840µs | 340KB |
| `deferred` | 144KB (−55%) | 510µs (−40%) | 157KB |

Option (c), `itemDefaults`, is not done: the 6% it covers is `insertTextFormat`,
and it needs a client advertising `completionList.itemDefaults` as well.

What follows is the analysis as it stood before, kept for its reasoning.

A completion response is 977 items and 331KB, and the handler is 2µs of the
633µs it takes. Two fields are two thirds of the payload:

| variant | JSON | marshal |
|---|---:|---:|
| as sent today | 331KB | 633µs |
| without `documentation` | 218KB (−34%) | 446µs (−30%) |
| without `detail` | 264KB (−20%) | 519µs (−18%) |
| without either | **151KB (−54%)** | **351µs (−45%)** |

`documentation` averages 100 bytes an item and `detail` 57, and both are for
the *one* item the user eventually looks at.

### Options

**(a) `completionItem/resolve`.** Send the list without those two fields and
fill them in when the client asks about a highlighted item. This is what the
request exists for. It needs `completionProvider.resolveProvider: true`, a
`completionItem/resolve` handler, and enough in `CompletionItem.Data` to find
the item again.

It must be gated on the client's advertised
`textDocument.completion.completionItem.resolveSupport` — a client that does
not resolve would show a popup with no documentation at all, which is a
regression, not an optimisation. The server already reads client capabilities
in `handleInitialize` (see `clientWatchesFiles`), so the gate is cheap; the
handler is the work.

**(b) Defer only `documentation`, keep `detail`.** `detail` is the signature
line, shown inline in the popup list rather than in the expanded panel, so it
is the one a user reads without asking. Two thirds of the saving for a much
smaller behavioural change — but the same resolve machinery, so it is not
cheaper to build.

**(c) `CompletionList.itemDefaults` (LSP 3.17)** hoists fields shared by every
item out of the list. It covers `insertTextFormat`, which is `2` on every
snippet item — about 21KB, 6% — and does not cover `documentation` or
`detail`. Worth doing alongside (a), not instead of it.

Not done here because it is a feature with capability negotiation and a
resolve handler, not a tune-up, and because it is the second-largest item on
this page rather than the first.

## 4. `LookupPreferred` scans the name's bucket — measured, and not worth it

When a bare function name is not declared in the requesting file,
`nearestTo` (`internal/index/index.go`) calls `cfpath.URIDistance` against
every definition of that name to pick the closest. On a shared name across
5,000 files that is ~150µs, on hover, signature help and argument completion.

The only saving available is that the reference URI is fixed across the loop,
so the part of `URIDistance` that depends only on it — the backward scan to the
last separator, and the count of separators after it — can be computed once.
Prototyped and measured over 5,000 entries: **205µs to 168µs, −18.4%**, with
the prepared version agreeing with the current one on every URI.

That is under a fifth of a cost that is already a fraction of the parse it
accompanies, in exchange for a second implementation of a function whose
comment explains at length why its answer must be exactly what it is. A
smaller constant does not change the shape either — the minimum of N distances
needs all N.

Skipping the scan entirely would need the buckets indexed by directory, which
is the same few-megabytes-per-index trade rejected for the bucket search in
#101 and rejected again here for less benefit.

**Not worth doing.** Unlike sections 2 and 3, this one is closed rather than
deferred: if the 150µs ever matters, the answer is to stop asking — cache the
preferred definition per (name, file) and invalidate it on an index write —
not to make the scan 18% faster.

## 5. What would change these decisions

- **Section 2** — done for the dominant short-query payload. A real-workspace
  measurement can still inform whether 1,000 is the right bound; queries of
  three or more characters intentionally remain complete.
- **Section 3** — done; see the section.
- **Section 4** — a profile showing `nearestTo` mattering on a real workspace
  rather than on a bucket built to be worst-case. Then cache the answer rather
  than speed up the scan.

Anything here that gets done should re-measure end to end, with the marshal,
on the benchmark named in section 1 — not on a handler benchmark.

## 6. Unresolved caller inference reread and reparsed the workspace — done

Batch `unresolved` already reads every file while building its definition
index. Caller inference used to read every file a second time when its lazy
name-to-caller index was first needed. The workspace-index pass now records
those call-shaped names from the bytes it already holds, so caller lookup adds
no filesystem pass.

The parsed-handler cache also used the editor's 512-entry reset policy during
batch scans. On a workspace with more than 512 possible callers, filling the
cache cleared every earlier parse; later arguments then reread and reparsed the
same handlers. Batch inference has a finite `InferArgsFiles` set and now retains
those parses for the scan's lifetime, while interactive resolver instances keep
the existing bounded policy.

`TestBatchCallerIndexReusesBytesFromWorkspaceIndexing` pins the eliminated
second read pass. `TestBatchHandlerParseCacheDoesNotThrashAtEditorLimit` crosses
the old 512-entry boundary twice and requires exactly one read per handler.

The caller-name pass itself no longer runs a regular expression that allocates
one submatch slice per call. A single linear scan finds `(` and walks backward
over the same ASCII whitespace and word characters; only distinct normalized
names are cloned, so the index cannot retain a complete source string through
one substring. On `BenchmarkCallerNames` with 10,000 calls and 100 distinct
names, five baseline runs took 27.6–28.9 ms and allocated 1.30 MB in 10,031–
10,032 allocations. Three optimized runs took 0.50–0.52 ms and allocated 7.8 KB
in 111 allocations: about 56x faster and over 99% less allocation.

Parallel workspace indexing can encounter the same caller name in an arbitrary
file order. Caller lookup therefore sorts and compacts its small candidate list
before inference, preserving stable alternative ordering and preventing a file
re-indexed during setup from being counted twice. The returned slice is a copy,
so later index updates cannot race a caller walking it.

## 7. Batch scans repeatedly stat the same source files — done

A strict Wheels corpus scan at `f26d68d` spent 79.46 of 99.36 sampled CPU
seconds in system calls. `wheelsSource` alone accounted for 53.44 cumulative
seconds, repeatedly validating metadata for unchanged framework sources.
The profile allocated 28.51 GB over the scan (total allocations, not peak RAM).

`unresolved.Scan` now wraps its filesystem in a concurrency-safe metadata
cache. Each distinct Stat or ReadDir path reaches the underlying filesystem
once, including negative lookups. Nested strict/speculative passes share the
wrapper; a subsequent top-level scan gets a new one. Directory result slices
are copied so callers cannot corrupt another lookup. ReadFile and Walk retain
their existing behavior. Editor and standalone resolver instances are unchanged.

**Assumption:** the input tree does not change during a batch scan. This is
not an atomic snapshot: bytes are still read on demand. Metadata and errors
remain cached until the operation ends, so edits, creates, deletes or permission
changes during a scan require a new scan. Cache storage scales with distinct
queried paths and directory entries, and is released with the scan; no inference
or file-coverage limits were added.

On macOS arm64 / Go 1.27.2, serial alternating runs used the same command:
`unresolved --json --candidates` with the Wheels checkout and its vendor/wheels
root (1,710 indexed and scanned files). No profiling or tests ran alongside
the timed scans. The order was before, after, main, after, before, main,
before, after; binaries were built from `f26d68d`, this change, and
`origin/main` (`769495b`) respectively.

| Binary | Wall seconds | Median |
|---|---|---|
| Before | 28.073, 28.596, 28.737 | 28.596 |
| Metadata cache | 17.248, 16.709, 16.850 | 16.850 |
| origin/main | 4.066, 3.810 | 3.938 |

This change reduces median scan time by **41.1% (1.70x throughput)** against
the task baseline. The full branch remains **4.3x slower than main** on this
command, with materially different resolution work: main reports 11,855
unresolved calls and the branch reports 2,556. That is a remaining performance
gap, not evidence that this branch is faster than main.

`TestMetadataCacheCoalescesConcurrentLookups` failed with caching disabled:
100 concurrent repetitions made 200 stats and 100 directory reads, versus
2 and 1 with the cache. Additional tests cover per-scan refresh, missing-file
results, directory-slice ownership, and live file reads. `BenchmarkMetadataLookups`
is an isolated, repeatable direct-versus-cached filesystem benchmark.

All seven corpus projects were compared per entry in strict and liberal modes,
including candidate details, assumed calls, speculative receiver connections,
and speculative method connections. There were **zero additions or removals**
in any category. Findings remain **12,348 strict / 289 liberal**; indexed-file
and resolved-call counts are unchanged for every project (8,151 indexed files
across the corpus).

A separate optimized Wheels profile sampled 60.40 CPU seconds, of which 34.93
were system calls, and allocated 25.86 GB total (about 9% less). Remaining large
allocation sources include map cloning (~7.9 GB), `allTokens` (~3.9 GB) and
`producerTokens` (~2.0 GB). Investigate reuse within producer evaluation next,
with content/context invalidation and corpus equivalence tests before caching
any resolution results. These allocation figures are sampled totals, not peak
memory or a claim that the remaining work is unnecessary.

The empty-directory metadata microbenchmark measured 37.95–38.50 microseconds
and 1,672 bytes / 8 allocations for a direct Stat+ReadDir pair, versus
37.92–38.11 nanoseconds and zero allocations after cache warmup. Nonempty
ReadDir results still allocate a defensive slice copy; whole-corpus timings
above are the user-visible evidence, not an extrapolation of this microbenchmark.

Validation: `go build ./...`, `go vet ./...`, `go test -short ./...`,
`go test -short -race ./...`, `make lint`, and `git diff --check` passed.

## 8. Association reads, caller tokens and unused producer history — implemented

The post-metadata-cache Wheels profile still spent 33.81 sampled CPU seconds
in filesystem ReadFile calls; association lookup accounted for 27.10 cumulative
seconds. A missing association method caused the same model source to be read
and searched again on every lookup. Association declarations now live in the
existing source cache. They depend only on source bytes, retain first-declaration
precedence, and return a fresh definition value with the requested spelling.
The cache's existing file stamp/index-generation checks and invalidation apply.

Caller argument inference also tokenized complete caller files repeatedly,
including while checking whether an unknown receiver omitted an argument.
Both paths now share a content-checked token cache through the owning resolver.
It uses the existing bounded span-cache policy (reset at 4,096 files), replaces
old content for a file, and stores no inferred receiver or argument types.
Keeping whole-file tokens preserves lexical context and exact source offsets.
Returned tokens are read-only; current consumers only inspect or slice them.

Producer evaluation built and merged an intermediate-state history after every
statement, although only a try body's catch path consumes that history. Normal
evaluation now retains just the final environment and returns. Try bodies still
collect the same intermediate states, including the initial environment; this
is a work-elimination change, not a weaker control-flow rule. Copies needed to
isolate branches and callers remain in place.

All three structural regressions failed with their corresponding optimization
withdrawn. They verify read-once association lookup (including absent methods),
first-match precedence, returned-definition ownership, refresh after edits,
shared caller tokens with allocation bounds and content replacement, and absence
of unused producer history while catch paths retain merged intermediate values.

## 9. Per-call rescans of whole sources — implemented

Profiles of cfwheels and MuraCMS scans showed the time going to work repeated
per call over whole files: regular expressions, tokenising, reading and
parsing the same sources again. Each is now one pass per file or one answer
per key, cached for the scan (reset with the discovery caches):

| Hotspot | Was | Now |
|---|---|---|
| `onlyFileWritesRequest` | every batch file read, lowercased and matched with a regex compiled per request key | `requestWriters`: one scan of each file, an index of key → writing files (`requestWriteCounts`, held to the old regex by `TestRequestWriteCountsMatchTheRegexItReplaced`) |
| `sharedFieldContract` | the request-write regex and a full `parser.Parse` of the file on every evaluation | counts and startup assignments read once per `wheelsSource` (`sourceDerived`) |
| `wheelsWrapperHosts` | each wrapper caller tokenised once per template asked about (3.2 GB of allocation on cfwheels) | `wheelsWrapperCalls`: once per caller |
| `fw1Actions` | a regex over every controller's source per `rc.` call (Mura's `editsite.cfm`: 160 calls, 3s) | cached per section.item |
| `boundEntity` | the service and its bases read and regex-searched per cborm call | cached per path |
| caller inference | caller files parsed one by one at the end of the scan (wheelstest runners, 6s each) | `prefetchHandlerParses` parses them 4 at a time first |
| producer branches | both branch environments copied by `condition` and again by `execute` | `executeOwned` takes the fresh copies (1.1 GB less allocation on cfwheels) |
| batch reads | file bytes always re-read | `CacheMetadata` keeps them, 2 MB per file, 128 MB in all |

Wall time, alternating runs on this machine (10 cores): cfwheels 11.0s → 8.3s,
MuraCMS 8.2s → 5.0s (v0.5.2: 3.5s each). Output is identical per entry on the
seven-project corpus in strict and liberal modes (resolved counts included),
tassweb and the nine validation apps (`target/resolution/pr244-perf/`).

That left a gap, read at the time as caller inference for common method
names. Measured rather than inferred, it was not; see section 10.

## 10. Repeated method lookups, and memory — implemented

Section 9 ended on a guess: that the lucli deploy commands' inference of
`run`/`start`/`stop` arguments was the critical path and needed a call-site
index. Instrumented, the call-site checks cost 0.2s on cfwheels and 0.7s on
MuraCMS in all; an index would have saved little. What the counters and a
per-file timing found instead:

| Hotspot | Was | Now |
|---|---|---|
| `lookupFunc` | 405,000 lookups on a cfwheels scan, 97% of them repeats, each walking the extends chain and, in a Wheels controller or model, the mixin plan (TestBox's two runners, 3.9s each, were the critical path) | `funcCache`: one answer per path, method and depth for an index generation, kept only where a source cannot change unseen (`IndexTracksFiles`, or a batch scan, `SnapshotScan`) |
| `computedIncludeHosts` | every file naming a template's directory read and lowercased whole for each template asked about: 111 MB live at tassweb's peak, and most of its scan time | `computedIncludeLines`: each candidate's computed includes found once; a file with none is never read again |
| `inferArgument` | every file mentioning the name parsed before the first is checked: `run` is in 675 files, and the loop stops at the fifth on an `ssh.run()` nothing types | parsed `prefetchWindow` (8) files at a time as the loop reaches them |
| `receiverTypeGuards` | every file tokenised whole (half of `allTokens`' 4 GB on tassweb) | a file that never names `isInstanceOf` is not tokenised |
| `localAssignmentAt`, `variablesAssignment`, `nativeReassignedAt` | `strings.Split` of the whole file per lookup (1.6 GB on tassweb) | `sourceLines`: only the lines read, held to the split by `TestSourceLinesMatchSplittingTheFile` |

| | v0.5.2 | before section 9 | before this | now |
|---|---|---|---|---|
| cfwheels | 3.5s | 11.0s | 7.5s, 400 MB | 3.6s, 360 MB |
| MuraCMS | 3.5s | 8.2s | 4.3s, 310 MB | 3.9s, 285 MB |
| tassweb | | | 30s, 3.4 GB | 11.6s, 1.4 GB |

Peak resident memory, `/usr/bin/time -l`, alternating runs. tassweb's live heap
at its peak went from 900 MB to 500 MB; resident is about twice that with the
collector's headroom. Output is identical per entry on the seven-project
corpus in both modes (resolved counts and coverage included), tassweb and the
nine validation apps (`target/resolution/pr244-perf2/`).

What holds tassweb's peak now: the batch content cache (140 MB, 128 MB
budget; turning it off saves about 150 MB resident and costs 1 to 3s), the
parses in flight in the scan workers, and `wheelsSource` (55 MB of method maps
and producer plans per file, which every producer evaluation reads, Wheels or
not). The largest allocation left is `findClosures` tokenising each file
whole (1.9 GB over a tassweb scan); it could share one tokenisation per file
with the other span caches, at the cost of holding the tokens.
