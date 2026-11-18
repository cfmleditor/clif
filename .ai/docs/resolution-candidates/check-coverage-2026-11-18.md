# Check coverage validation — 2026-11-18

Compared f5e9819 with the coverage-error fix using identical arguments, existing configuration discovery, and `GOMAXPROCS=1`. Each complete JSON entry was compared as a multiset, retaining duplicates, rather than comparing totals alone.

| Cached source | Indexed before/after | Scanned before/after | Findings | Removed | Added |
|---|---|---|---:|---:|---:|
| fw1 | 415/415 | 305/305 | 209 | 0 | 0 |
| testbox | 415/415 | 146/146 | 300 | 0 | 0 |
| coldbox | 415/415 | 664/664 | 1023 | 0 | 0 |
| contentbox | 415/415 | 724/724 | 3813 | 0 | 0 |
| wheels | 415/415 | 1882/1882 | 13306 | 0 | 0 |

## Coverage and repeatability limits

These are the cached FW/1, TestBox, ColdBox, ContentBox and Wheels source trees, using their existing configs and parent config discovery. Wheels explicitly includes vendor/wheels when present. This differs from the canonical six-project corpus and prescribed preset configurations: Lucee is absent, and no configs were replaced for this check. Findings and indexed/scanned counts were identical per entry under the fixed single-worker setting. Dedicated regressions cover unreadable files, directory traversal failures, grammar refusals, MCP error propagation and preserving reports after an incomplete scan. No resolution acceptance rules changed.

An initial default-worker FW/1 comparison differed. Repeating the unchanged old binary produced 204, 195 and 208 findings; the new binary produced 208, 196 and 174. The old binary therefore already varies between runs. Two subsequent interleaved runs of each binary with GOMAXPROCS=1 all produced the same 209-entry report. The table uses this setting for a reproducible comparison; it does not establish default-worker determinism, which remains a separate issue.

Canonical after-report SHA-256 values (sorted keys and entries, duplicates retained):

- fw1: `f427432bffc4f16db2b19680978b84c5658844cb518a5ea1871fa6a8f59b30e4`
- testbox: `974b3e54da8602e07264be2ef1878a5a6ea947c1ff976c67efb20eccfeef6c4d`
- coldbox: `1a26bf9a6b51d5210444eadbdaff554de22e01009729d6a944ba18e7dadd05e9`
- contentbox: `7aeac37df92224b951b4cf58324be316ae3dc9182e9088b4b0e3c884a306bd39`
- wheels: `89c86f263197725ad41b82f05455be7f675ad1429263860c98214c2491d9b667`
