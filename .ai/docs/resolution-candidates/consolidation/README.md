# Corpus comparison: shared token readers

Baseline `1c9b2b4` versus consolidated parser/resolver helpers. Both binaries run
the same seven corpus roots with existing configurations, explicit Wheels vendor
coverage, `--json --candidates --verbose`, and GOMAXPROCS=1. Complete finding
objects are compared as multisets, including duplicates. No source checkout or
project configuration was changed.

| Project | Indexed / scanned on both | Findings on both | Resolved on both | Removed / added |
|---|---:|---:|---:|---:|
| Wheels | 1,710 / 1,710 | 795 | 63,411 | 0 / 0 |
| ColdBox | 652 / 652 | 183 | 20,005 | 0 / 0 |
| ContentBox | 724 / 724 | 177 | 17,120 | 0 / 0 |
| TestBox | 145 / 145 | 110 | 3,994 | 0 / 0 |
| FW/1 | 305 / 305 | 69 | 2,632 | 0 / 0 |
| Mura | 839 / 839 | 151 | 34,100 | 0 / 0 |
| Lucee | 3,776 / 3,776 | 283 | 45,196 | 0 / 0 |

`comparison.json` preserves machine-readable counts. Reproduce with
`CLIF_BEFORE=<baseline binary> CLIF_AFTER=<final binary> python3 run.py`.
The script writes raw JSON, verbose logs and per-entry deltas under `runs/`;
those large local artifacts are not committed. Parser dump comparison covers
8,249 files independently, in two modes; its checksum is in
`parser/comparison.json`. Four mutation-check logs are under `mutations/`.
See `../../consolidation-validation/README.md` for behavioral and LSP validation.
