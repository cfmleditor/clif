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
