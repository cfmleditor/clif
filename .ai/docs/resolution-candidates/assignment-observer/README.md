# Collection writes and the main parser walk

An isolated Go overlay recorded the offsets that the main CFScript parser
visited while recognizing ordinary declarations, scoped assignments, bare
assignments, and dotted assignments. The prototype formed records at those
offsets with the existing target and expression-end readers. It compared the
full ordered records, including targets, expression text, offsets, and flags,
with `scanExpressionWritesFrom` on the same script regions.

The corpus walk read 8,251 `.cfc` and `.cfm` files across seven local projects,
including vendored sources. It skipped `.git`, `node_modules`, and files over
2 MiB. The observer matched 1,286 of 6,904 script regions. It produced 109,899
of 241,259 existing write records and zero extra records. The missing records
include indexed writes such as `this.mappings[...]`, closure markers, compound
writes, and lexical records from named arguments and declaration attributes.
The main parser's assignment events cover a different set of syntax than the
collection scanner. Replacing the latter from those events would change
behavior in 5,531 files. [Per-project counts](comparison.json)

## Reusing lexer tokens

A second overlay captured tokens from the main parser's scanner. It accepted
a tape only when the scanner input matched the region text and the tape
covered a continuous scan up to EOF. The source check matters because the
main parser blanks foreign-language bodies before parsing. Without this check,
the prototype accepted three streams with missing tokens in Lucee fixtures.
With it, all 6,597 accepted tapes matched standalone tokenization exactly.
The other 307 regions used standalone tokenization. The existing collection
scanner still walked the accepted tapes and kept its write semantics.

A parser microbenchmark fed checked tapes to collection and return inference
when available. It used the ColdBox controller, `ExtractCalls`, `GOMAXPROCS=2`,
and three 350 ms samples per case. Two alternating texts fit the existing
shared token cache. A rotation of 32 whitespace variants exceeds its 16 slots.
The prototype supplied one tape read per parse in both cases.

| Text rotation | Baseline time | Checked tape time | Baseline bytes/op | Checked tape bytes/op |
| --- | ---: | ---: | ---: | ---: |
| Two variants | 1.319 ms | 1.497 ms | 777,141 | 1,386,174 |
| 32 variants | 1.408 ms | 1.295 ms | 1,332,310 | 1,350,185 |

Capturing tokens removes one tokenization on a miss, while a cache hit already
has the token list. The prototype captured in both cases and allocated an
extra 609 KB per warm parse. The production parser is unchanged. A follow-up
would need to identify token-cache misses before recording and show that the
second tokenization is avoided without paying the recording cost on hits or
in files that never run collection inference.

## Reproduce

The archived patches target `b99c6b6`. The runner applies the selected
patch in a temporary directory and passes an overlay to `go test`.
Run from the repository root with the project checkout paths as arguments.

```sh
python3 .ai/docs/resolution-candidates/assignment-observer/run-probes.py assignments "$COLDBOX" "$WHEELS"
python3 .ai/docs/resolution-candidates/assignment-observer/run-probes.py tokens "$COLDBOX" "$WHEELS"
python3 .ai/docs/resolution-candidates/assignment-observer/run-probes.py benchmark "$COLDBOX/system/web/Controller.cfc"
```

For the full seven-project result, pass ColdBox, Wheels, ContentBox, TestBox,
FW/1, Mura, and Lucee. Pass the Wheels checkout root so the walk includes its
vendored framework.

## Recording only on a predicted miss

The selective overlay checks the shared cache before recording a script region
of at least 8 KiB. On a miss it asks the existing member-binding predicate once
per file, using only the text and the lookups available before the region walk.
A positive answer predicts that member inference will ask for every script
region's tokens. A negative answer leaves tokenization to the existing reader.
Collection-return candidates and some typed member references only become known
during the parse. The normal inference pass still decides independently whether
to read tokens. Only a consumer publishes a complete captured stream to the
shared cache. Shallow and partial-body parses skip capture.

Each corpus run started with an empty shared cache and visited the same 8,251
files and 6,904 script regions. The lookup variant supplied a function lookup
that always returned the empty string, exercising the parser's lookup-enabled
path without a project resolver. Every consumed capture was compared token for
token with standalone tokenization of the original text.

| Parse options | Recording attempts | Exact streams reused | Incomplete recordings | Complete recordings unused | Large misses served by standalone tokenization |
| --- | ---: | ---: | ---: | ---: | ---: |
| ExtractCalls | 11 | 9 | 2 | 0 | 412 |
| ExtractCalls with function lookup | 372 | 346 | 26 | 0 | 187 |

Incomplete recordings fall back and are included in standalone misses when
requested. The prototype avoided 2.1% and 64.9% of large-region tokenizations
on demand respectively. Both runs had zero token mismatches. Disabling capture
made both ColdBox runs fail the positive-coverage assertion.

Three 350 ms samples per case on the same ColdBox controller gave the following
medians. The baseline disables recording in the instrumented build. The warm
pair is preloaded, the 32 variants exercise eviction, and the no-demand case is
a synthetic script over 8 KiB with a single local assignment.

| Workload | Baseline time | Selective time | Baseline bytes/op | Selective bytes/op | Recording attempts/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| Warm pair | 1.308 ms | 1.424 ms | 776,106 | 770,764 | 0 |
| 32 variants | 1.536 ms | 1.556 ms | 1,343,506 | 1,342,895 | 0.98–1.00 |
| No demand | 18.4 µs | 17.6 µs | 4,840 | 4,840 | 0 |

Warm parses allocated 3,952 times in both modes. The earlier 609 KB warm
recording penalty disappeared. Avoiding the second tokenization on a cold
controller parse did not deliver a consistent timing improvement in these
samples. The cost of checking the cache and predicting demand falls on the
parse; real lookup-enabled editor workloads would need measuring before
adopting this in production.

Run the same two commands above with `selective` in place of `tokens` and
`selective-benchmark` in place of `benchmark`.

## Indexed editor corpus

The `editor` probe configures the LSP server for each of the same seven
workspace roots, indexes it, and parses each file with the document-open
handler's real resolver and parse options. It primes the resolver, empties the
shared token cache for each arm, and compares the baseline and selective parse
for that file. It checks function definitions (including inferred returns and
collection values), file and function-scoped component references, and scopes.
It also compares every consumed capture to standalone tokenization.

| Workspace | Files | Captures consumed | Large misses served by standalone tokenization | Incomplete captures | Files with differing results |
| --- | ---: | ---: | ---: | ---: | ---: |
| ColdBox | 652 | 20 | 44 | 20 | 0 |
| Wheels | 1,713 | 206 | 78 | 2 | 0 |
| ContentBox | 724 | 38 | 31 | 0 | 0 |
| TestBox | 145 | 9 | 7 | 3 | 0 |
| FW/1 | 305 | 5 | 0 | 0 | 0 |
| Mura | 937 | 40 | 8 | 1 | 0 |
| Lucee | 3,775 | 29 | 19 | 0 | 0 |
| **Total** | **8,251** | **347** | **187** | **26** | **0** |

The timing probe samples up to 12 files per project from each of three groups:
files that consumed captures, other files at least 8 KiB, and smaller files.
It picks evenly spaced paths within each group. That gives 233 files, weighted
toward opportunities for reuse. It times the synchronous `textDocument/didOpen`
handler, including JSON decoding, parsing, and reindexing. Background completion
work is joined outside the timer before closing the document. Structure diagnostics
are disabled. The server uses the editor's 16 MiB cache budget and both modes
use the instrumented binary. Each arm primes the resolver and index with an
untimed open, clears the token cache, then times an open and an immediate
reopen. Three paired repetitions alternate arm order by path and repetition.
Each timing percentile therefore covers 699 opens per arm and cache condition,
per trial. These are local handler timings on an indexed workspace.

Reproduce with `python3 run-probes.py editor ROOT...` from this directory. The
order used was ColdBox, Wheels, ContentBox, TestBox, FW/1, Mura, then Lucee.

| Token cache | Baseline p50 | Selective p50 | Baseline p95 | Selective p95 |
| --- | ---: | ---: | ---: | ---: |
| Empty before open | 0.458 ms | 0.467 ms | 4.994 ms | 4.830 ms |
| Immediate reopen | 0.439 ms | 0.453 ms | 4.969 ms | 4.839 ms |

These are the medians of the three trials' pooled percentiles. Selective reuse
removed 347 of 534 standalone tokenizations on large-region misses in the
parity run. On the timed sample, the selective arm consumed 222 captures on
empty-cache opens per trial and zero on reopens. The p95 improvement also
appeared on reopens, making it difficult to attribute it to token reuse.
Typical open latency showed no consistent benefit on cold tokens, and the
reopen median was slightly higher in every trial.
[Per-project and combined timings](editor-open-timings.tsv)

## Reusing the recording buffer

The original selective recorder grew a fresh token slice for each attempted
capture. The revised recorder borrows a growable buffer from a pool. When
inference consumes a complete stream, it copies the tokens once into an
exact-sized slice owned by the shared cache. It clears the scratch token
references before returning the buffer, and discards buffers with capacity
above 32,768 tokens. Incomplete and unconsumed recordings never get copied
into the shared cache. The pool can retain scratch capacity between parses
and Go can drop it at garbage collection.

A second variant buffers only the ordinary tokenizer on a large cache miss.
It uses the same copy and release operation, and starts only when a consumer
requests tokens. It requires no early demand check or recording during the
main scanner walk. The two variants isolate the savings from reusing storage
and from avoiding a second tokenization.

ColdBox controller, 32 rotating texts, medians of three 350 ms samples:

| Implementation | Time per parse | Bytes allocated per parse | Allocations per parse |
| --- | ---: | ---: | ---: |
| Ordinary tokenizer (control in revised build) | 1.363 ms | 1,339,610 | 3,967 |
| Original selective recorder (separate build) | 1.350 ms | 1,336,914 | 3,970 |
| Pooled ordinary tokenizer on demand | 1.335 ms | 951,469 | 3,954 |
| Pooled selective recorder | 1.230 ms | 951,281 | 3,956 |

The original build's ordinary-tokenizer control measured 1.415 ms and
1,342,449 bytes per parse. With two warm texts, all three arms in the revised
build allocated 3,952 times per parse and neither variant recorded anything.
The synthetic file without inference demand allocated 4,840 bytes in all arms.
These figures measure allocation traffic, not peak or retained process memory.

The indexed editor workload used the same per-file parity checks and 233
timed documents as before. The ordinary-tokenizer experiment also ran the
selective eligibility probe outside the timer to preserve the exact same
sampling groups. Each variant was compared against capture disabled in its own
instrumented build. Each trial has 699 opens per arm per cache condition.

Run `buffered-benchmark` with the controller path, and `buffered-editor` and
`pooled-editor` with the seven workspace roots. `buffered-selective` runs the
parser-only corpus comparison plus a check that recycling the source buffer
leaves the shared cache's tokens intact.

All three trials of each variant matched all 8,251 files. Selective capture
reused 347 streams and fell back to ordinary tokenization on 187 other large
misses. The demand-only variant exercised its pooled tokenizer on all 534
large misses. Every checked stream matched standalone tokenization.

| Variant and token cache | Control p50 | Variant p50 | Control p95 | Variant p95 |
| --- | ---: | ---: | ---: | ---: |
| Pooled capture, empty | 0.507 ms | 0.495 ms | 5.321 ms | 4.965 ms |
| Pooled capture, reopen | 0.476 ms | 0.469 ms | 5.082 ms | 4.927 ms |
| Pooled tokenizer, empty | 0.491 ms | 0.495 ms | 5.148 ms | 5.130 ms |
| Pooled tokenizer, reopen | 0.478 ms | 0.468 ms | 5.120 ms | 5.148 ms |

The values are medians of three trials' pooled percentiles. The clearest
benefit was lower allocation on the controller misses. Pooled capture also
lowered empty-cache p95 in each editor trial. Reopen timings moved despite
zero captures on reopens, so the size of any user-visible latency gain is
uncertain. The demand-only variant gives a smaller production candidate if
allocation traffic is the objective. [Trial timings](buffered-open-timings.tsv)
