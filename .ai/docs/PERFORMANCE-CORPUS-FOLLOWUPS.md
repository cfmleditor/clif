# Parse and fold performance: what to verify and try on a machine with the corpus

A run of performance work on the parser, the folding handler and the collection
passes (commits `6c85bdd` to `5e6b3a7`, on top of `7749400`) was done in a
sandbox with **none of the corpora**: only this repository's fixtures, the
package tests' own snippets, and two synthetic benchmark components. Every
number below was measured on that, on a 4-core 2.1GHz Xeon whose repeat runs
differ by about ±10%. None of it has been checked against ColdBox, Lucee,
ContentBox, Mura, cfwheels or FW/1.

This file is the list of what to do first when a corpus is available: the
checks that the work changed no answers, the measurements that were only
possible on synthetic input, and the ideas that were left for want of a corpus.
The corpus setup (projects, commits, configs) is in
[RESOLUTION-GAPS.md](RESOLUTION-GAPS.md) under "Measuring".

## 1. What changed, and what each change risks

`7749400` is the commit before the series; build it as the baseline.

| Commit | Change | Measured here | What a corpus could show |
|---|---|---|---|
| `6c85bdd` | `lineCursor` replaces a binary search per line lookup in the tag walk | tag-page fold pass 1.5ms to 0.76ms | none: it is `lineAtOffset`'s answer by construction (`TestLineCursorMatchesLineAtOffset`) |
| `1f794e1`, `5e6b3a7` | `tagEnds.fusedAt`: one pass over a tag's text when it is at most 16 bytes | tag parse of the benchmark component 254us (before) to 244us | **a regression at 128 bytes was found only by bisecting**; see section 4 |
| `63be868` | spans pre-sized from the line count (capped at 16,384); popped tag frames' branch lists reused | tag-page fold pass allocations 623 to 14 | memory on very large tag files; the cap exists for a file of blank lines at the 32MB limit |
| `0e030f1`, `ea94eb5` | identifier byte table; identifier tested first in `next`; identifier loop on a local | fold pass of a script component about 8% faster, scanner about 8% | none expected |
| `68a9177` | `Token` is 32 bytes: `TokenKind` is a `uint8`, `Line` an `int32` | scanner alone 567us to 340us, script parse about 17% faster | any code that assumed `Line` is an `int`; a file is capped at 32MB, so overflow is not reachable |
| `a3cce37` | one reused script parser for a tag file's expressions | tag parse with calls: allocations 907 to 698 | state leaking from one expression into the next, on tag files with thousands of `<cfset>` |
| `550576f` | `collKey`: the collection and member passes key by an interned `uint64`, not a built string | script parse allocations 1,313 to 881, time about 3 to 6% | every function's `CollectionValues`, `ReturnSources` and every member ref; memory (the `keyTable` holds each distinct name for a parse) |

End to end, the folding request on a tag page (83KB, 600 rows) went from
1,706us to 774us at the handler (2,011us to 1,005us with the response
marshalled). A 134KB script component's request barely moved (959us to 896us)
because it is bound by the script scanner. The token change came after that
measurement; the folding request was **not re-measured end to end after it**.

## 2. Verify first: nothing should have changed

No change in the series is meant to alter an answer. The checks, in order of
how much they cost.

**a. The parse dump** (new, in `internal/parser/corpus_dump_test.go`). It
writes every function's return type, component, return sources and collection
values, every component ref, and the call and instantiation counts, for each
corpus file, in two parse modes (with and without a function lookup). Run it
on the baseline and on HEAD and diff:

```bash
git worktree add /tmp/clif-base 7749400
export CFML_CORPUS=/src/Lucee:/src/ContentBox:/src/coldbox-platform:/src/MasaCMS
rm -f /tmp/dump-before.txt /tmp/dump-after.txt     # a failed build leaves the old file behind
(cd /tmp/clif-base && cp <repo>/internal/parser/corpus_dump_test.go internal/parser/ \
  && CLIF_DUMP=/tmp/dump-before.txt go test -count=1 -run TestCorpusResultDump ./internal/parser/)
CLIF_DUMP=/tmp/dump-after.txt go test -count=1 -run TestCorpusResultDump ./internal/parser/
diff /tmp/dump-before.txt /tmp/dump-after.txt && echo identical
```

The baseline checkout has no copy of the test, hence the `cp`; the test needs
no other helper. A difference is a file whose answers changed: find the commit
with `git bisect` over the nine, running the test on that one file.

`CLIF_DUMP_SNIPPETS=1` adds the ~360 CFML snippets from the package's own tests
(small cases written for exactly these passes). They differ between two
commits whenever the tests do, so use that flag for a working-tree change
against its parent, not for baseline against HEAD. During the series the dump,
with the snippets and the 108 uniform-collection cases, was identical across
27,970 lines on this repository's fixtures.

**b. `resolution-report`, before and after.** It builds whatever is checked out,
so run it twice:

```bash
git checkout 7749400 && make resolution-report CORPUS="masa=/src/MasaCMS lucee=/src/Lucee ..." \
    RUNS=target/resolution/before LISTS=
git checkout <branch> && make resolution-report CORPUS="..." \
    RUNS=target/resolution/after BASELINE=target/resolution/before LISTS= SHOW=5
```

The expected result is **0 removed and 0 added on every scan**. `LISTS=` (empty)
compares without rewriting the committed candidate lists.

**c. `make gapcheck CORPUS=...`.** The parser against the grammar, per file. It
must report the counts it reported on the baseline.

**d. The CLI outputs, byte for byte**, as was done on the fixtures: `clif
unresolved --json`, `graph --format jsonl --level call` (strip `buildMillis`),
`deps` and `scan` with the baseline binary and the new one over each project.
All 16 comparisons on this repository's fixtures were identical.

**e. Memory on the largest files.** The interned keys and the tag-expression
parser hold more state per parse than before. Parse the largest corpus files
(a 65,000-line component, the Lucee admin pages) under `/usr/bin/time -v` on
both builds.

## 3. Measure on real files

The benchmark components are 8 to 15KB and written by hand; their mix of
tags, keywords and identifiers is not a real file's. In particular the script
benchmark is keyword-heavy, which can hide a gain that depends on identifier
variety (this is why the `classify` pre-check in section 5 should be re-tried).

New, in the same test file, all skipped without `CFML_CORPUS`:

```bash
# which files are slow to parse? profile those, not a synthetic fixture
CFML_CORPUS=/src/Lucee CLIF_SLOWEST=25 go test -count=1 -run TestCorpusSlowestParses -v ./internal/parser/

# whole-corpus throughput; alternate two builds of the test binary on it
go test -c -o /tmp/new.test ./internal/parser/        # and the same at 7749400 -> /tmp/old.test
for i in $(seq 20); do for b in old new; do
  CFML_CORPUS=/src/Lucee /tmp/$b.test -test.run '^$' -test.bench BenchmarkCorpus -test.benchtime 3x -test.benchmem
done; done
```

`BenchmarkCorpusParse` parses every file with call extraction;
`BenchmarkCorpusStructure` computes the fold spans of each. Report MB/s and
allocations per file, not only the total.

Things worth reading off it that the synthetic files cannot say:
- the **tail**: the slowest 1% of files, and whether any is superlinear in its size;
- whether the tag parse is any faster on **real** tag files (on the benchmark
  component it is 4% faster, against 20% for a script component, because the
  tag parser finds tags by string search and the scanner work does not reach it);
- the folding request's **p99** on the largest files, to decide the
  `features.folding` default (see [FOLDING-PLAN.md](FOLDING-PLAN.md) section 5).

## 4. Method traps that cost time here

- **Alternate the two builds, one run of each in turn, 20 or more times, and
  compare medians.** A sequential A-then-B comparison measured a 10% change as
  -0.7%. Repeat runs of one build differ by ±10% here.
- **A profile's share is not a saving.** Five estimates from a profile missed
  badly: a shared `<` index (profile said 14% of classification; it was 20%
  slower), a key sort in the folding handler (11.6% of the profile; no gain),
  a `classify` pre-check (11.6%; no gain), struct keys for the collection passes
  (fewer allocations, 7 to 9% **slower**) and the main-parse write recording
  (not attempted). Treat a profile share as a ceiling on what a change can save,
  and an estimate as unproven until two builds are alternated.
- **Re-run every benchmark that shares the code, not the one the change was written
  for.** The fused tag-text loop was measured on a fold fixture of short HTML
  tags and was 15% faster there; it made the tag parse of a component 8 to 15%
  slower, because a `<cffunction>` line is 80 bytes of attributes and a byte loop
  loses to the vectorised calls it replaced. Nobody re-ran `Parse_TagCFC`. It was
  found by building each of the nine commits and alternating them. A
  plain revert would also have been wrong: the gain was on attribute-less tags,
  which a limit of 16 bytes keeps.
- **A failed build looks like a pass.** Output files from the last good run stay
  on disk. Delete them first; `cmp` of a stale file against itself says
  "identical".
- **`sed` on a file `gofmt` has reformatted silently does nothing**, so a
  mutation test can "pass" because the mutation never applied. Check that the
  file changed before reading the result.
- **Cost of a rejected design is written down so it is not rebuilt.** A
  `collKey` that is a struct of strings (rather than an interned integer) was
  slower than the strings: Go keys such a map through its generic path, and the
  buckets are three times the size and full of pointers, which raised GC from
  15% to 19.5% of CPU. Do not try it again.

## 5. Things to try, in the order they are worth trying

Each was left because it needs a corpus to be safe or to be measured. The
estimates are ceilings from a profile; see the second trap above.

### 5.1 Record collection writes in the main parse (the one large item)

`ParseResult.collectionWrites` re-reads every script region to find writes
(`x = …`, `x[k] = …`, `x.append(`, `+=`, closures). It is **about 20% of a
script parse** (1.5s of 7.35s in a profile of `Parse_ScriptCFC_ExtractCalls`):
`scanExpressionWritesFrom` 14%, of which `expressionEnds.fill` 5.6%;
`scriptTokens`/`tokenizeInto` 3.5%; `placeWrites` 2.7%. Recording the writes
as the parser walks the tokens would remove the second tokenization and the
second walk.

It was not attempted, for three reasons that a corpus does not remove but
makes survivable:

1. The main parse cannot run on the replay scanner the write pass uses. Replay
   supports only `NextSkipComments`, `PeekSkipComments`, `Save` and `Restore`,
   and its tokens have comments removed. The parser reads `LastBlockComment`
   (`script_parser.go`, doc comments) and `bytesAfterPeek` (raw lookahead).
2. Recording tokens during the parse would see some twice (peek, `Restore`) and
   some not at all (a skipped function body); whether the tape is complete is
   not checkable cheaply.
3. The write recogniser looks ahead from every identifier with scanner copies;
   the parser's six token loops each recognise assignments their own way.
   Teaching each a second recogniser is the "two scans must agree" problem
   `CLAUDE.md` describes for the variable scans (`TestBothVariableScansAgree`).

The safe shape, if it is attempted: an observer on the **one** assignment
site the parser already has for each scope (`parseScopedVar`,
`parseBodyScopedVar`, `parseBodyVarDecl`, `checkAssignRef`) that records the
same `collectionWrite` the scan would, and the scan kept alongside it
**behind a test flag** so section 2a can diff the two on every corpus file
before the scan is deleted. Expect well under the 20%: the recognition work
moves, it does not vanish. Do not start without 2a and 2b green on the
corpus first.

### 5.2 Smaller items in the write pass

- `expressionEnds.fill` calls `classify` twice per identifier token (the token
  and its predecessor). Carrying the predecessor's class forward saves
  roughly 1% of a parse; below this sandbox's noise, so unproven.
- `placeWrites` calls `functionAt` (two binary searches) per write; a cursor over
  `pr.Scopes`, as `lineCursor` did for lines, would likely do.
- `settleCollections` and `invalidateCollectionParents` are 7% and 5.5% of the
  pass; both walk every node. Not profiled below that.

### 5.3 Re-try on real code what lost on a fixture

None of these is likely to win; each cost an afternoon to reject on the
fixtures, and the fixtures may have been unrepresentative:

- **`classify` pre-check** (a table of which first-byte and length pairs any
  keyword has). Measured -1% on a 134KB generated component and -2% on the
  repository's largest real `.cfc`, both noise. The generated component is
  keyword-heavy, which is the worst case for a pre-check. Try it on a
  corpus-wide benchmark; drop it if it is under 3%.
- **A shared index of `<` offsets** across classification and the tag walk.
  It cost a full walk to build and the cursors did not beat `IndexByte`
  (classification 225us to 270us). A corpus of long tag pages might differ.
- **Sorting the folding handler's keys instead of the `FoldingRange` structs**
  (stable sort 10% of the component request). Equivalent output was proven;
  the gain was 2%. Not worth the code.

### 5.4 Tag parsing

The plain tag parse profile is flat (largest items: `IndexByte` 12%,
`EqualFold` 5%, GC 13%), and 4% faster than before the series, against 20%
for script. Profile a **real** tag-heavy file from `CLIF_SLOWEST` before
changing anything; the remaining allocation sites are `CallHop` (the
`"$call:"+expr` string, 4 to 9%), `funcKey` (a `start:end` string per
function), and the `addCall`/`addRef` appends. The sub-parser reuse in
`a3cce37` keeps its maps and lists but not the per-key slices inside them,
which is another few allocations per expression if the corpus shows tag files
with thousands of them.

### 5.5 The scanner

The scanner is about 340us of a 686us fold pass on the 134KB component, and a
token is 32 bytes. Two long shots, both of which change every parser caller:
make a token a kind, an offset, a length and a line (16 bytes) and read its
text from the source, with the few values that are not a slice of it (`?.`
scanned as `.`) special-cased; or return tokens by pointer into a slab. The
first is a type change across the roughly 100 uses of `.Offset` and every
`.Value`; do it with the compiler-driven rewrite that worked for `Line` (feed
the compiler's errors into a script, fix the few left by hand), and only if the
corpus shows scanning is still a leading cost.

### 5.6 The folding default

`features.folding` is opt-in because it was once expensive. With the changes
above it is about twice as fast on tag pages, and the span pass reproduces
93.9% of the old tree-sitter folds (see [FOLDING-PLAN.md](FOLDING-PLAN.md)
section 5). With `BenchmarkCorpusStructure` and the per-request p99 from
section 3, the question of switching the default can be answered with
numbers.

## 6. Reproducing the synthetic numbers here

The benchmarks used (all in the repo):
`BenchmarkParse_ScriptCFC`, `_ScriptCFC_ExtractCalls`, `_TagCFC`,
`_TagCFC_ExtractCalls` (`internal/parser/benchmark_test.go`) and
`BenchmarkFoldingRange` (`internal/server/perf_bench_test.go`, with `benchDoc`
and `benchPage` for larger inputs). `TestParseStaysWithinItsAllocationBudget`
holds the allocation counts: script 1,100, script with calls 1,180, tag 425,
tag with calls 870.

Medians over 20 alternating runs against `7749400`, for reference:

| | before | after |
|---|---:|---:|
| script parse with calls | 820us | 653us (-20%) |
| tag parse | 254us | 244us (-4%) |
| tag parse with calls | 385us | 349us (-9%) |
| script parse allocations | 1,313 | 881 |
| tag-with-calls allocations | 907 | 698 |
