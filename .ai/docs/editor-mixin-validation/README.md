# Deferred hover and Wheels mixin probes

Baseline `9aa4634` includes the file metadata reuse in #392. The workload is
`TestEditorSession` on ColdBox's `system/web/Controller.cfc`, with Go 1.27.4 on
Apple M1 Pro and `GOMAXPROCS=2`.

An isolated test overlay added a `runtime/pprof` phase label around the hover
request, then ran 400 edits of each kind. Filtering the profile to hover after
outside-function edits yielded 2.24 CPU seconds in hover, of which 2.23 seconds
were under `flushReparse` and 1.58 seconds under `settleSignatures`. Within
that path, samples included 0.51 seconds in member binding and 0.40 seconds
in resolver matching. These are cumulative samples.

A candidate skipped the prefix-alternative iterator for simple prefixes and
removed a repeated factory match. All seven corpus reports matched per entry.
Five paired editor sessions gave outside-function hover p50 medians of 7.48 ms
before and 8.58 ms after, and p95 medians of 14.16 and 15.54 ms. Benchmark
binary compilation overlapped one pair, but the other samples also failed to
show a consistent gain. The candidate was reverted.

The same profile attributed 0.14 CPU seconds in source metadata reads to
`wheelsObjectIntegrates`. A controller or model at the end of its ancestry
chain cannot inherit the required `wheels.Global` loader. The follow-up
checks that chain boundary before reading source. The targeted regression
observes one stat before the change and zero after. Other candidates keep the
existing source and ancestry checks.

Three interleaved sessions per binary, with 200 edits of each kind, measured
outside-function hover p50 at 8.38 ms before and 8.53 ms after. The median
session p95 moved from 15.96 ms to 14.50 ms, with overlapping ranges. This
sample gives mixed latency results. The regression test establishes the
filesystem work removed by the change.

The seven-project comparison used the same roots and settings as the metadata
experiment. Indexed and scanned counts, resolved calls and unresolved entries
matched for every project. The per-project counts are under
`../resolution-candidates/editor-mixin/`.
