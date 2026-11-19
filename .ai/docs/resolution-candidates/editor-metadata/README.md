# Editor metadata optimization: corpus comparison

Baseline `3e54152`, same seven source trees and existing configurations, explicit
Wheels vendor root, GOMAXPROCS=1, `unresolved --json --candidates --verbose`.
Complete finding objects are compared as multisets, retaining duplicates.

| Project | Indexed/scanned on both | Unresolved on both | Resolved on both | Removed/added |
|---|---:|---:|---:|---:|
| Wheels | 1,710 | 795 | 63,411 | 0/0 |
| ColdBox | 652 | 183 | 20,005 | 0/0 |
| ContentBox | 724 | 177 | 17,120 | 0/0 |
| TestBox | 145 | 110 | 3,994 | 0/0 |
| FW/1 | 305 | 69 | 2,632 | 0/0 |
| Mura | 839 | 151 | 34,100 | 0/0 |
| Lucee | 3,776 | 283 | 45,196 | 0/0 |

`comparison.json` records counts. Use CLIF_BEFORE and CLIF_AFTER binaries with
`run.py` to reproduce; raw logs/JSON go to `runs/` and are not committed. This
checks unchanged batch resolution, not the live editor cache lifecycle, which
has separate regressions. Source freshness and LSP measurements are documented
in `../../editor-metadata-validation/README.md`.
