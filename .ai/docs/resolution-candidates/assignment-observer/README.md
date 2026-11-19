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
