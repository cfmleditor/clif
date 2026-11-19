# Editor source metadata reuse

Baseline `3e54152` versus `perf/editor-source-metadata`, Go 1.27.4 on Apple M1
Pro, GOMAXPROCS=2. The baseline was built from an isolated `git archive` checkout.

## Change and lifetime

The live resolver already trusts an unchanged index generation for source
metadata. An unrelated edit advanced that global generation and made each
unchanged library stat its source again. Retain the global fast path, then use
an indexed file's own revision to distinguish an unrelated edit from a change
to that file. This is reuse under the existing IndexTracksFiles contract, not
a time-based or process-wide filesystem cache.

Index revisions advance when parsed/file-result content is indexed and when
lines shift. File and workspace removal discard the record; recreation gets a
new revision from the monotonic index write counter. Unindexed inputs retain
the previous stat behavior. Watch/save invalidation still clears source caches,
including for a replacement with the same size and timestamp. Batch scans do
not opt into the new source reuse.

The source's association map moves into its already-shared derived facts so
the additional revision does not enlarge the frequently copied sourceInfo
value beyond its previous 72 bytes. The revision map adds one entry per indexed
file and releases entries on file/workspace removal; no missing-path history
is retained. General method/return answer caches still use the global generation.

## Behavioral checks

- Regression: 20 writes to another indexed file followed by reads of an unchanged
  library required 21 stats before the fix and one afterward.
- Disabling the file-revision reuse through a Go overlay makes that regression
  fail with 21 checks again (`no-reuse-mutation.log`).
- Index tests cover function-free sources, both indexing APIs, unrelated
  changes, line shifts, deletion, recreation and workspace removal.
- Resolver tests cover changed, deleted and recreated inputs, unindexed/live
  inputs, and watched replacement with unchanged size/time.
- An instrumented real editor session confirms the edited Controller keeps
  refreshing while its six repeatedly checked libraries stop being checked on
  each edit. Instrumented timings are not used as performance evidence.

## Targeted metadata benchmark

One unrelated edit followed by validation of 32 unchanged libraries. Three
interleaved fresh processes per binary, 1s benchmark windows, no race/profile
instrumentation. The identical new benchmark fixture was copied into the
isolated baseline's tests; baseline production sources stayed unchanged.

| Per operation, median | Before | After |
|---|---:|---:|
| Time | 120.496 µs | 6.459 µs |
| Filesystem Stat calls | 32 | 0 |
| Allocated bytes | 10,882 | 4,224 |
| Allocations | 65 | 33 |

This isolates metadata validation; it does not measure full return inference or
an end-to-end LSP request. Raw samples are in `metadata-benchmark.json`.

## End-to-end editing sessions

Five interleaved fresh sessions per binary against the existing ColdBox corpus,
editing system/web/Controller.cfc. The server test indexes 530 files. Each
session performs 200 inside-function edits, 200 outside-function edits, 100
line add/remove pairs and 50 close/open cycles. The harness waits 15ms between
didChange and hover; that wait is excluded from handler timing. No other
validation jobs ran during this clean comparison. Earlier exploratory sessions
with the larger struct layout and concurrent lint are excluded.

Medians of each session's quantiles:

| Following hover | Before p50 / p95 | After p50 / p95 |
|---|---:|---:|
| Inside-function edit | 71 / 122 µs | 70 / 125 µs |
| Outside-function edit | 6.729 / 11.419 ms | 6.688 / 11.942 ms |

These results do not establish an overall editor speedup. The outside-edit p95
ranges overlap (before 9.983–12.413ms; after 10.142–12.824ms). Heap stays about
13–14MB through edits and returns to 8MB after close/open cycles. Full session
samples and summaries are in `clean/`; the raw logs are kept locally.

`compare.py` reproduces these sessions with CLIF_BEFORE_SERVER and
CLIF_AFTER_SERVER binaries. CLIF_EDITOR_ROOT may select another ColdBox tree;
hold that source, binary toolchain and configuration constant across both sides.

## Corpus and quality checks

The seven configured corpus projects remain identical per complete finding,
including duplicates and candidate metadata. Indexed, scanned and resolved
counts match, with zero findings added or removed. See
`../resolution-candidates/editor-metadata/` for counts and the runner.

Build, vet, full short race tests, and pinned lint pass locally. The initial
lint findings (struct size and test whitespace) were corrected. No latency
target or safety threshold was weakened. Published-head CI is checked separately.

The broader deferred-reparse delay remains open: eliminating these redundant
stats is useful, but their observed cost does not explain the entire measured
hover latency. A phase-specific profile of signature settling is the next step;
the earlier whole-session profile alone overstates how much latency this cache
can remove.
