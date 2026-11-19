# Wheels route-template scope improvement

Branch: `fix/wheels-route-template-scope`. Validation compared with `3ade589`;
the published branch is rebased onto main to exclude the independent docs-driver
PR #389. That PR changes no generated catalog content or resolver behavior.

Wheels Global.$lockedLoadRoutes includes `/config/routes.cfm` (historical corpus
`vendor/wheels/Global.cfc:1743`). The Wheels preset now supplies Global as the
implicit base of that route template, including nested scaffold and example apps.
This exposes the actual mapper factory and its source-validated Mapper methods.
No dynamic receiver acceptance or new factory return contract was introduced.

The two new regressions failed before the three-line preset change and passed
afterward. They cover nested paths, case-insensitive matching, disabled/foreign
presets, unrelated files/extensions, a real fluent factory chain, a nonexistent
Mapper method, and an absent Global method.

Controlled corpus scans use the same absolute source roots, configs, flags and
GOMAXPROCS=1 on both binaries. Arrays are compared as multisets of complete JSON
objects, including candidate evidence and duplicates. `comparison.json` records
indexed/scanned/resolved counts and deltas. `run.py` reproduces the comparison
using `CLIF_BEFORE` and `CLIF_AFTER` binary paths; raw reports go to `runs/`.

| Corpus | Indexed/scanned both | Unresolved before → after | Removed / added |
|---|---:|---:|---:|
| Wheels, explicit vendor root | 1,710 | 853 → 795 | 58 / 0 |
| ColdBox | 652 | 183 → 183 | 0 / 0 |
| ContentBox | 724 | 177 → 177 | 0 / 0 |
| TestBox | 145 | 110 → 110 | 0 / 0 |
| FW/1 | 305 | 69 → 69 | 0 / 0 |
| Mura | 839 | 151 → 151 | 0 / 0 |
| Lucee | 3,776 | 283 → 283 | 0 / 0 |

All 58 removed entries are mapper calls/chains in four config/routes.cfm files:
CLI app template (5), starter app (35), tweet example (14), and framework test
resource app (4). Wheels resolved calls increase by exactly 58, from 63,353 to
63,411; combined resolved+unresolved coverage is unchanged. Every other project
has identical findings and resolved counts. There is no corpus coverage change.
This is a controlled comparison with the current branch base, not a comparison
with older saved reports or the separate 1,882-file framework cache.

Build, vet, full `go test -short -race ./...`, pinned lint (zero findings), and
`git diff --check` pass locally. Lint whitespace findings were corrected in the
test additions. A sandbox Go-cache denial was retried with approved permissions.
No PR has been published for this improvement, so there is no CI claim for it.
