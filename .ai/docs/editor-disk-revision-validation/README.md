# Disk-backed source metadata after buffer edits

This experiment follows the [signature settling profile](../editor-signature-settling.md).
The baseline is `b99c6b6`. The prototype, built through a Go overlay, kept the
disk-source revision stable when reindexing an editor buffer or shifting its
lines. Disk indexing advanced the revision, and file or workspace removal
cleared it. The save path still invalidated resolver caches. Watcher events on
open files also invalidated those caches while preserving indexed buffer
symbols.

## Behavior

In separate instrumented runs of 12 outside-function edits, the baseline
made 12 `sourceInfo` stats on ColdBox's open `Controller.cfc`. The prototype
made zero. Both runs encountered 16 `hasSelfDefaultParam` calls per hover
that reached the instrumented path, on 12 baseline hovers and 11 prototype
hovers. The instrumented runs were used for counts only.

Short index, resolver, and server suites passed with the prototype. A targeted
test checked the same disk-backed source after 20 buffer edits and line shifts.
It observed one stat with the prototype and failed with 21 when the overlay
was mutated to advance the disk revision on each buffer update. Tests also
covered disk changes, deletion, recreation, same-stamp replacement, saves,
and watcher invalidation while an application file was open.

## Editor latency

Five alternating baseline/prototype pairs used the same ColdBox controller
and positions as the profile, with Go 1.27.4, an Apple M1 Pro, and
`GOMAXPROCS=2`. Each `TestEditorSession` run made 200 inside-function and 200
outside-function edits. The harness's 15 ms delay between edit and hover
is excluded from the handler timings. The table shows the median of each
session quantile. [Per-session samples](samples.json)

| Hover following an edit | Baseline p50 | Prototype p50 | Baseline p95 | Prototype p95 |
| --- | ---: | ---: | ---: | ---: |
| Outside a function | 7.967 ms | 8.239 ms | 13.050 ms | 13.055 ms |
| Inside a function | 68 µs | 88 µs | 209 µs | 302 µs |

Outside-function p50 was higher in four of five pairs and p95 was higher in
three. The outside-function p95 ranges overlapped (11.230–13.676 ms and
11.922–16.011 ms). Skipping the disk check produced no observed hover
improvement in this first pass.

## Implementation follow-up

The implementation keeps the disk-revision optimization for the filesystem
work it removes. Saved and watched changes invalidate the resolver's disk
facts, including when a file has a separate open buffer. A lookup already in
flight may finish from its snapshot. An invalidation prevents that lookup
from repopulating either the source cache or the return cache. Files without
a disk revision continue to use the existing recheck path.

The server lifecycle test exercises a buffer edit, save, watched change,
deletion and recreation. It holds the disk file's size and timestamp fixed
when changing the return contract and checks that document symbols still
come from the buffer. Mutation checks independently restore the repeated
stat, ignore watcher events for open buffers, and allow stale lookups to
refill each cache. Each mutation fails the corresponding test.

Three alternating editor sessions per build, using the workload above, gave
outside-function hover medians of 9.371/15.068 ms (p50/p95) on the baseline
and 9.481/15.490 ms on the implementation. The pairs vary in direction.
These sessions establish no consistent latency change. [Implementation samples](implementation-samples.json)

## Resolution corpus

The seven-project comparison matched all 1,768 unresolved findings per
complete entry, including duplicate entries and candidate details. Both
builds scanned 8,151 files and resolved 186,458 calls. Wheels included its
explicit `vendor/wheels` root and had 795 identical findings. The batch
resolver does not use the editor's disk-revision fast path, so this compares
batch behavior rather than testing that path. [Per-project comparison](../resolution-candidates/editor-disk-revision/comparison.json)
