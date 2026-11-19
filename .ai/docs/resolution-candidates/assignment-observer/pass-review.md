# Token reuse review at 059bc16

Compared the research branch's stock parser, pooled selective capture, and
pooled on-demand tokenization. Temporary overlays enabled each variant for the
lifetime of the CLI or test process. Single-threaded diagnostic increments were
removed for concurrent callers. Each candidate logged its first actual use.
No production code was changed.

## Unresolved

Ran `unresolved --json --candidates` on the same paths under their existing
workspace configurations with GOMAXPROCS=2. Compared multisets of complete
JSON findings, including candidate evidence. Indexed and scanned counts,
resolved counts, and finding multisets matched in all three builds.

| Scan | Files indexed and scanned | Unresolved | Resolved |
| --- | ---: | ---: | ---: |
| ColdBox | 652 | 183 | 20,005 |
| ContentBox | 724 | 177 | 17,120 |
| FW/1 | 305 | 69 | 2,632 |
| Lucee | 3,776 | 283 | 45,196 |
| Mura | 839 | 151 | 34,100 |
| TestBox | 145 | 110 | 3,994 |
| Wheels root | 677 | 395 | 22,828 |
| Wheels root plus vendor/wheels | 1,710 | 795 | 63,411 |

The last row is a separate, overlapping scan. The seven root scans total 1,368
unresolved. The editor workload walks vendored CFML up to 2 MiB and has different
coverage from the CLI.

## Scanner passes

A pass is the number of raw scanner reads divided by the reads for one plain
scan of the acted-on text. Replaying a stored token does not increment that
counter. The seven-project editor experiment used 8,251 files, 5,803,251
plain token reads, an indexed workspace and a 16 MiB editor cache. For each
file it primed the resolver, emptied the shared token cache, measured one
document parse, then immediately parsed the same document again. The passcount
build performed no oracle tokenization in the measured paths. Two independent
process runs per mode gave the following read counts.

| Build | Empty cache, run 1 | Empty cache, run 2 | Immediate repeat, both runs |
| --- | ---: | ---: | ---: |
| Stock | 9,194,684 | 9,198,301 | 7,189,286 |
| Pooled selective capture | 7,820,560 | 7,825,111 | 7,189,286 |
| Pooled on demand | 9,199,291 | 9,197,287 | 7,189,286 |

All modes invoked the parser 8,251 times per condition. The separate
`TestEditorActionPasses` run on ColdBox Controller and RequestContext included
the handler's downstream work and counted the following pass equivalents.

| Editor action | Stock | Selective capture | On-demand pool |
| --- | ---: | ---: | ---: |
| First open | 22.28 | 21.30 | 22.28 |
| Second component open | 13.46 | 13.49 | 13.46 |
| Reopen | 1.87 | 1.87 | 1.87 |
| Folding | 1.00 | 1.00 | 1.00 |
| Document links | 1.00 | 1.00 | 1.00 |
| Hover | 0.00 | 0.00 | 0.00 |
| Edit inside function | 0.09 | 0.09 | 0.09 |
| Edit outside functions | 2.87 | 1.87 | 2.87 |

The targeted handler suite recorded 176 passing tests and subtests per mode.
A separate comparison opened the indexed ColdBox controller and inserted a
leading newline with a whole-document change. Hover, definition, signature
help and document symbols returned identical JSON values at both versions.
Completion returned the same 451 items with the same fields. Their order
varied across processes, including a second stock run.

## Findings at 059bc16

The selective capture decision survives a full replacement of a ParseResult's
content. Adding the first qualifying binding through ApplyFullReplace left
capture disabled; freshly parsing the same text reused one stream. Replacing
that document with a binding-free document made one unused recording. Reset
the demand decision on each signature extraction and test both edit directions.

The cold pooled_scan microbenchmark can pass when the pooling path is disabled.
Forcing its cache-miss guard false still passed the benchmark. Count pooled
misses independently and require positive cold and zero warm activity for that
arm, as the checked-capture benchmark already does for recording.

## Fix verification

The demand decision now expires at the start of signature extraction.
With the cache disabled, adding a qualifying binding on a full replacement
records and consumes one stream, and removing it attempts zero recordings.
Disabling the reset in a disposable overlay made both edit tests fail.

The buffered benchmark and the editor control now read the same atomic
counter from the actual on-demand miss path. Forcing that path off made
the cold benchmark fail even at one iteration. Prewarming occurs outside
the measured warm case; cold arms start from an empty cache.

All three cold benchmark samples reported 1.000 pooled misses per parse;
warm and unused cases reported zero. The parser corpus passed under the race
detector across 8,251 files, as did the indexed ColdBox editor control.
Rechecking the CLI findings produced the same entries and counts in the
table above. The revised builds also matched the stock editor responses and
passed the same 176 targeted handler tests and subtests per mode.

In the post-fix pass-count run the stock, selective, and on-demand variants
made 9,200,657, 7,820,994, and 9,197,362 cold scanner reads respectively.
For an immediate repeat they each made 7,189,286 reads.
