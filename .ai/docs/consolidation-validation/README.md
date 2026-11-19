# Shared token readers and ancestor traversal

Baseline: `1c9b2b4` (Wheels route-template scope). The refactor shares four
mechanics without changing acceptance policy:

- Newline filtering returns the original slice without allocations when there
  is nothing to remove, and never modifies cached token storage.
- Wheels factories and absent-argument returns share a streaming reader through
  the call's opening parenthesis. Argument interpretation and end-of-expression
  checks stay local; no token list is allocated for the zero-argument check.
- Single-delimiter matching shares `parser.GroupEnd`. Mixed-delimiter counting
  and the parser's EOF-stopping bracket reader retain their different contracts.
- Four framework ancestry readers share cycle detection and traversal bounds.
  Their 8/12-step limits, current-file checks, implicit ancestry, matching rules,
  and fallback lookup directory are preserved by each visitor.

Caches, script/markup quoting and parser dispatch were not merged. Production
code is reduced by 16 lines; the value is one implementation of each shared
mechanic, not a claimed speed improvement.

## Behavioral validation

The parser and resolver short suites pass. Differential tests retain the former
brace reader and call-opening behavior, including malformed input. Separate
tests cover cached-slice ownership, a zero-allocation fast path, EOF versus
mixed-delimiter policies, ancestor depth bounds, cycles and early matches.

Four isolated overlays break filtering, call opening, group depth and cycle
checks. Each produces a behavioral test failure. No production source was
edited for mutation checks; see `../consolidation-mutations.py` and the logs in
`../resolution-candidates/consolidation/mutations/`.

The exact baseline and final parser binaries produce byte-identical dumps for
8,249 files in two parse modes (9,123,435 bytes, SHA-256
`352ba865516567e7c4e801e9078ad0110f569cc2c3fbf0fa286fc1672daa388a`).
The dumps include function return metadata, collection values, references, and
call/instantiation counts. Detailed call-site evidence comes from the resolver
comparison, not these aggregate call counts.

The seven configured corpus projects have identical complete finding objects,
including duplicates and candidate metadata, and identical indexed, scanned and
resolved counts. There are zero added or removed findings. Collection differs
between the parser dump and configured resolver scans; neither changes between
revisions. See `../resolution-candidates/consolidation/` for the per-project
summary and reproduction runner. Raw reports and parser dumps are local
validation artifacts and are not included in the commit.

ColdBox editor pass-count measurements match exactly for every action:

| Action | Token passes | Parses |
|---|---:|---:|
| First open | 22.29 | 11 |
| Second component open | 13.47 | 6 |
| Reopen | 1.90 | 1 |
| Folding / document links | 1.00 each | 0 |
| Hover | 0.00 | 0 |
| Edit inside a function | 0.09 | 1 |
| Edit outside a function | 2.90 | 0 |

## LSP comparison

Apple M1 Pro, Go 1.27.4, GOMAXPROCS=2. Each synthetic benchmark runs in a fresh
process with a 5,000-file index. Three 300ms samples per binary are interleaved;
the two flagged cases were repeated with five isolated 1s samples per binary.
No race or pass-count instrumentation is used for timings.

| Operation | Before median | After median |
|---|---:|---:|
| Completion handler | 1.589 µs | 1.623 µs |
| Completion plus JSON, full | 0.465 ms | 0.481 ms |
| Completion plus JSON, deferred | 0.297 ms | 0.288 ms |
| Symbols plus JSON, populated query (repeat) | 3.984 ms | 3.721 ms |
| Typing, 20 functions (repeat) | 3.230 µs | 3.290 µs |
| Typing, 200 functions | 12.444 µs | 12.380 µs |
| Hover | 0.861 µs | 0.856 µs |
| Definition | 1.908 µs | 1.920 µs |

The initial short run flagged +20.7% for symbols and +10.9% for small-file
typing. Those did not reproduce in the longer isolated repeat (-6.6% and +1.9%,
respectively). Symbols do not execute the changed resolver helpers. Allocations
are unchanged for completion, typing, hover and definition; asynchronous line
edit allocations vary by one across samples. No consistent performance
regression or speedup is established. Full samples are in `benchmarks.json`
and `repeated-benchmarks.json`. Warm-handler cases are not editor transport or
UI timings and do not establish latency for every resolution scenario.

Real ColdBox session, 200 edits inside and 200 outside a function:

| Following hover | Before p50 / p95 | After p50 / p95 |
|---|---:|---:|
| Inside-function edit | 73 / 201 µs | 90 / 203 µs |
| Outside-function edit | 8.932 / 13.283 ms | 6.589 / 11.971 ms |

These are individual session traces, not a statistical improvement claim. The
test indexes 530 files under its server configuration. Post-GC heap stays around
13–14 MB through edits and ends at 8–9 MB after line shifts and close/open cycles.

## Deferred-reparse profile and follow-up

A separate CPU profile captures the editing session. Of 1s of sampled CPU,
470ms is under `extractSignatures`, 380ms under
`handleHover -> lockDoc -> flushReparse -> reparseShallow`, and 370ms under
`frameworkapi.overlay.Stat` through return inference. These are overlapping
cumulative frames, not additive costs. The whole session also includes indexing,
explicit garbage collection, and close/open loops; isolate the edit phase and
count metadata calls before attributing a precise per-edit share.

This points to repeated filesystem metadata lookups during return-signature
settling as the next performance investigation. Any reuse must preserve fresh
source detection after edits and filesystem changes; a global stale stat cache
is not introduced by this refactor. Profile summaries are committed; the binary
CPU profile and raw timing logs remain local. `../consolidation-verify.py`
reproduces the parser, pass-count, benchmark and session checks using the named
baseline/final binaries.

Final build, vet, the full short race suite, pinned lint (zero findings), and whitespace checks pass locally. Published-head CI is checked separately when the PR is created.
