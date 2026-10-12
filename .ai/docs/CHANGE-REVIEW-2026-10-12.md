# Change review — 2026-10-12

Reviewed local HEAD `6ebe727` against the recorded main baseline `769495b`.
The branch contains four commits and changes 331 files. This review examines
purpose and recorded performance across the branch, with code-level inspection
focused on the latest ownership corrections, batch metadata cache, and document
relocation. It is not a new exhaustive line-by-line audit of all 331 files.

## Findings and limitations

1. **The current implementation summary is stale.**
   [Current checkpoint](RESOLUTION-IMPLEMENTATION-STATUS.md#current-checkpoint)
   still presents 12,355 strict / 295 liberal findings and 5,095 provisional
   method links. The latest saved per-entry comparisons give 12,348 / 289 and
   5,094 links. Historical milestone reports can retain their original figures,
   but the current checkpoint should identify the latest validated result and
   link to the review fixes and performance report.
2. **End-to-end performance remains a branch-level gap.** The metadata cache
   improves the preceding branch version substantially, but the current Wheels
   workload is still 4.3x slower than main. More resolution work explains a
   difference in scope; it does not establish that all extra cost is necessary.
   Map cloning and repeated tokenization are the next measured allocation targets.
3. **Performance coverage is narrower than correctness coverage.** Seven projects
   were compared for output equivalence, but the alternating timing measurements
   cover one strict Wheels command. They do not establish a 41% improvement for
   every project, liberal mode, editor latency, or peak resident memory. The
   benchmarks and corpus reports under ignored `target/` also require the local
   corpus to reproduce; the committed performance document preserves the method
   and results, not every raw artifact.

No additional blocking correctness defect was identified in the focused cache
and relocation review. This is bounded by the inspection scope and tests below.

## Purpose of the changes

| Area | Purpose | Important boundary |
|---|---|---|
| Source-backed resolution | Trace constructors, DI, application mappings, arguments, returns, collections, framework mixins and implicit accessors to callable definitions. | A relationship requires evidence; a matching name alone does not prove receiver ownership. |
| Native and engine contracts | Recognize supported built-in producers and member APIs, including ordered structs. | Source-owned functions must shadow builtins; engine/version assumptions matter. |
| Method evidence and optional shortcuts | Separate unavailable declarations from missing connections, and expose provisional targets for investigation. | Optional shortcuts can select inaccessible or unrelated declarations and bypass broken chains. Their ownership remains explicitly unproven. |
| Review corrections (`f26d68d`) | Distinguish ordered structs from arrays, preserve source ownership for builtin-name collisions and `invoke`, and carry framework settings into editor exports. | The new guards avoid confident but incorrect native typing; export fixes address the copied framework settings rather than proving universal CLI/editor parity. |
| Metadata cache (`6ebe727`) | Avoid repeated filesystem Stat/ReadDir work during batch resolution and speculative passes. | Metadata and errors are retained for one operation; files must remain stable during that scan. |
| Documentation relocation (`6ebe727`) | Keep research, plans and corpus evidence under `.ai/docs`, with updated references and generation defaults. | Root entry-point docs and fixture READMEs remain in place. |

## Current resolution counts

The latest seven-project strict report has **12,348 findings**, with 8,151
indexed files. The combined liberal report separately contains:

| Category | Count |
|---|---:|
| Remaining findings | 289 |
| Runtime assumptions | 2,024 |
| Speculative receiver connections | 1,884 |
| Provisional method connections | 5,094 |

These are distinct evidence categories, not interchangeable measures of proven
resolution. Receiver connections describe groups and should not be added to
call counts. In particular, 289 does not mean that only 289 strict relationships
remain unknown. The provisional method policy moves calls into an audit ledger
without increasing the resolved count.

## Performance evidence

The saved alternating run sequence uses the same strict Wheels command,
1,710 indexed/scanned files, three before/after samples, and two main samples.

| Version | Median wall time | Wheels findings |
|---|---:|---:|
| Main (`769495b`) | 3.938 s | 11,855 |
| Before cache (`f26d68d`) | 28.596 s | 2,556 |
| After cache | 16.850 s | 2,556 |

The cache reduces median elapsed time by **41.1%**, with identical output to
its immediate baseline. Main performs materially different resolution work;
its result establishes the full-branch cost, not an equivalent-output comparison.

Separate sampled profiles show total allocated bytes falling from **28.51 GB
to 25.86 GB**, approximately **9.3%**. This is allocation volume, not allocation
count or peak RAM. The remaining dominant allocation sites are map cloning
(~7.9 GB), `allTokens` (~3.9 GB), and `producerTokens` (~2.0 GB).

The isolated empty-directory benchmark demonstrates the repeated-I/O mechanism,
not representative whole-workspace speed. Nonempty directory reads still copy a
slice to protect cached results from caller mutation.

## Correctness and validation reviewed

- `sync.OnceValues` coalesces concurrent requests for each path; separate maps
  retain Stat and ReadDir results. Tests require two underlying stats and one
  directory read for 100 concurrent repetitions, including a missing path.
- Directory slices are copied before returning. File bytes and Walk remain
  live; this is explicitly not an atomic filesystem snapshot.
- `Scan` creates the wrapper before its recursive strict/speculative passes;
  passing that wrapper through nested scans preserves its lifetime. Ordinary
  subsequent scans using the original filesystem create a fresh cache.
- A new component is visible to a subsequent scan in the integration regression.
  Editor resolver instances are not switched to this cache.
- Rechecked the saved fourteen project/mode comparison records: zero changed
  evidence entries and unchanged coverage in every comparison.
- Re-ran `go test -short -race` for `internal/vfs`, `internal/unresolved`, and
  `internal/resolve`; all pass. The preceding implementation run also passed
  full build, vet, short tests, race tests and lint.
- Documentation move validation checked local Markdown links, shell syntax,
  Makefile report output defaults, generator compilation/help, and diff whitespace.

## Recommended next work

1. Update the current checkpoint while preserving historical milestone reports.
2. Measure representative strict and liberal workloads from the other projects,
   plus peak RSS, before making broader performance claims.
3. Profile producer environment cloning and repeated tokenization. Any reuse
   must preserve content, argument, caller-application and index invalidation.
4. Require per-entry corpus equivalence for each optimization; retain original
   diagnostics and uncertainty labels when improving provisional connections.

This review makes no production-code changes and does not claim fresh CI results.
The reviewed commits remain local following the blocked push.
