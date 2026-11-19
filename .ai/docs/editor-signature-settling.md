# Signature settling during ColdBox hover

Two `TestEditorSession` runs on clif `b99c6b6` used Go 1.27.4 on an Apple
M1 Pro with `GOMAXPROCS=2`. The harness indexed 530 source files and made
400 edits of each kind. It waited 15 ms between each edit and hover.
Outside-function hover p50/p95 were 7.13/15.25 ms and 4.63/12.59 ms in the
two runs. These are repeated observations of the same build.

An overlay around the test's `handleHover` call set the pprof label `phase`
to the edit group name. The table counts sampled CPU time with the
`keystroke outside any function` label and a `handleHover` frame. The passes
below `settleSignatures` are grouped by its immediate callee, so the rows
within settling do not overlap.

| Path | Run 1 | Run 2 |
| --- | ---: | ---: |
| Hover handler | 1,960 ms | 1,420 ms |
| Deferred reparse within hover | 1,950 ms | 1,420 ms |
| Signature settling within reparse | 1,480 ms | 1,030 ms |
| ↳ Pending calls | 590 ms | 400 ms |
| ↳ Member bindings | 410 ms | 240 ms |
| ↳ Chained returns | 250 ms | 120 ms |
| ↳ Collection returns | 130 ms | 160 ms |
| ↳ Generated property accessors | 80 ms | 80 ms |
| ↳ Other settling work | 20 ms | 30 ms |

Pending calls were the largest settling pass in both runs. `hasSelfDefaultParam`
accounted for 280 ms and 230 ms within that pass, all attributed to `Stat` in
`sourceInfo`. In a separate logging-only run with 12 outside-function edits,
11 hover handler calls checked the default-parameter rule 16 times each. Each
of those handlers made one successful `sourceInfo` stat, always on the open
`Controller.cfc`. The remaining hover had zero calls and zero stats on the
instrumented paths. The logging run was used for counts only.

The per-file index revision tracks buffer changes as well as disk-driven index
changes. `sourceInfo` reads disk and checks its cached stamp when that revision
moves, so even a whitespace edit causes it to stat the controller again. A
follow-up could test separate invalidation for disk-backed metadata, with
coverage for saves, watched changes, and deletion and recreation. The earlier
profile at `9aa4634` sampled only 10 ms in `hasSelfDefaultParam`, so the sample
share here should be verified in an interleaved comparison before attributing
a latency improvement to removing the stat. No production code changed.

## Reproduce the CPU breakdown

Use the `TestEditorSession` harness with an overlay of
`internal/server/editor_session_test.go`. Pass `c.name` to the hover closure
and wrap only the `handleHover` call in
`pprof.Do(ctx, pprof.Labels("phase", phase), ...)`, passing the labeled context
to the handler. Build and run with these settings and `-cpuprofile`.

| Setting | Value |
| --- | --- |
| `CLIF_EDITOR_SESSION` | Local ColdBox workspace root |
| `CLIF_EDITOR_FILE` | `system/web/Controller.cfc` |
| `CLIF_EDITOR_IN` | `102` |
| `CLIF_EDITOR_OUT` | `88` |
| `CLIF_EDITOR_HOVER` | `128:33` |
| `CLIF_EDITOR_EDITS` | `400` |

The controller fixture SHA-256 was
`dea87c1f9bbf968703037c27b259bc4a87cb314c70f89e86c7a49cca9f5ff2af`.
Filter and inspect with `go tool pprof -tagfocus='phase=keystroke outside any function' -traces`.
