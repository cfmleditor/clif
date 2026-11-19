# Project review, 19 November 2026

Reviewed merged main at `0dfe40bbdaef9dda664d564b62f0e2309eb5dc16` on macOS arm64 with Go 1.27.4. Five product defects reproduced. The first two can cause a commit gate to miss findings or a command to erase a previous report. A separate finding affects the repeatability of the project's performance gate.

## Findings

| Priority | Finding and reproduction | Proposed fix and regression check |
| --- | --- | --- |
| P2 | **The changed-lines lint gate drops findings for ordinary filenames with spaces.** Stage a one-line change to `with space.cfm`. A real temporary Git repo passed to `gitAddedLines` returned line 1 under a filename ending in a literal tab. With `diff.mnemonicPrefix=true`, a plain `page.cfm` was keyed as `i/page.cfm`. The strict hook uses the same filter, and `KeepLines` drops findings whose path has no matching entry. See `cmd/clif/gitfiles.go:110,148–152` and `internal/cflint/lint.go:242`. | Force predictable diff prefixes and parse the filename and Git's header delimiter separately, preserving quoting. Exercise both repositories through the filter and assert the staged finding survives. |
| P2 | **A mistyped routes format erases an existing report.** With a 16-byte report, `clif routes --format jsno --out report .` exited 1 as expected, but left the report at zero bytes. The command opens the destination before validating the format. See `cmd/clif/routes.go:115,137`. | Validate format before scanning or opening the output. Seed a report with known bytes, run the invalid command and check the exit status and the original contents. |
| P2 | **A closed component replaced by binary content keeps its old definitions.** Index a component declaring `obsolete()`, replace its contents with three bytes beginning with NUL and deliver a watched-file change. The server's index still returned one definition of `obsolete`. The binary-content branch returns “skipped”, bypassing removal and cache invalidation. See `internal/server/watchedfiles.go:251–253`. | Use the removal and invalidation path when an indexed, closed file becomes unreadable as source. Test text → binary → text and verify lookup after each event. |
| P3 | **Repeated route reports on unchanged input produce different site lists.** Give 16 templates the same configured `data-view="home.missing"` reference. Six successive JSON runs produced six different file orders. Workers append as they finish and the final sort compares only the route. Markdown prints just the first three sites, so even which files appear can change. See `cmd/clif/routes.go:244–245` and `cmd/clif/routes_report.go:269`. | Order ties by file, line, source and name before rendering. Assert identical JSON and Markdown for permutations of the same findings. |
| P3 | **The text routes report hides every example with the documented unlimited setting.** With one unresolved `data-view="home.missing"` on line 1, the default report prints `page.cfm:1`. With `--limit 0` it retains the group count and omits the site. The resolved-examples loop also stops at its first item for zero. See `cmd/clif/routes.go:28,310,364`. | Apply the bound only for a positive limit. Test zero, one and the default with resolved and unresolved examples. |

## Performance gate reproducibility

**P3. The stub pass-count gate depends on checkout context.** Both the normal workspace and a `GOWORK=off make check` run failed on the same stub result. An uncached run of the isolated gate on a clean archive of the identical commit passed. Both environments loaded the repository-root config and logged indexing 415 files. The test computes its denominator by walking only the stub directory, then calls the normal CLI scan, which discovers the parent config and uses its workspace roots. See `cmd/clif/scan_passes_test.go:109` and `cmd/clif/unresolved.go:292`.

| Workload | Working checkout, passes / parses per file | Clean archive, passes / parses per file | Limits |
| --- | ---: | ---: | ---: |
| testdata | 4.72 / 2.34 | 4.72 / 2.34 | 4.90 / 2.50 |
| Framework stubs | 3.31 / 2.82 | 2.43 / 2.18 | 2.50 / 2.30 |

Give the fixture an explicit workspace boundary and verify the indexed paths as well as the totals. The experiment isolates a checkout-dependent difference. It leaves the specific extra resolver reads responsible for the delta untraced.

## Validation and coverage

- Build, vet, embedded-CFLint checks, Go formatting and the short race suite passed under the local workspace grammar (v0.26.43) and release pin (v0.26.44). Both local `make check` runs then stopped at the pass-count gate above. Ran lint and the vulnerability scan independently. Lint passed and govulncheck reported no vulnerabilities. GitHub CI succeeded on the exact merge commit. [CI run](https://github.com/cfmleditor/clif/actions/runs/44516454425)
- The parser's fixture grammar-gap check passed against its recorded differences. The pinned formatter corpus run classified all 305 FW/1 files as clean, with zero skips, panics or unstable results.
- The ColdBox editor pass-count workload used the same Controller, RequestContext and cursor positions as the earlier token review. First open took 22.26 pass equivalents, second open 13.46, reopen 1.87, folding 1.00, links 1.00, hover 0.00, inside-function edit 0.09 and outside-function edit 2.87. These count tokens scanned relative to one file scan. They do not measure latency. [Earlier workload and measurements](resolution-candidates/assignment-observer/pass-review.md)
- Manual inspection sampled CLI linting and output, LSP indexing and navigation, parser and resolver scan boundaries, code-map caching, daemon lifecycle, installation and release packaging. Automated Go checks covered the full module. Cross-platform installers, WASM, and interactive editor clients were outside local execution coverage.

### Resolution corpus

Ran the stock CLI with `unresolved --json --candidates` and `GOMAXPROCS=2`. Compared complete JSON finding multisets, including candidate evidence and duplicates, against the saved stock outputs from the earlier token review. Indexed and scanned file counts matched the baseline in every row. Expanded Wheels overlaps the root scan and is reported separately.

| Workspace | Files indexed and scanned | Unresolved | Findings added / removed |
| --- | ---: | ---: | ---: |
| ColdBox | 652 | 183 | 0 / 0 |
| ContentBox | 724 | 177 | 0 / 0 |
| FW/1 | 305 | 69 | 0 / 0 |
| Lucee | 3,776 | 283 | 0 / 0 |
| Mura | 839 | 151 | 0 / 0 |
| TestBox | 145 | 110 | 0 / 0 |
| Wheels root | 677 | 395 | 0 / 0 |
| Wheels root plus vendor/wheels | 1,710 | 795 | 0 / 0 |
