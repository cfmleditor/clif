# Project review, 24 November 2026

Reviewed `main` at `e9502f0` (v0.7.5) on Linux amd64 with Go 1.27.4, then fixed what it found on branch `ccr-3b1edfe7-27btc1` (PR #449). Build, vet and the short suite passed before any change. The earlier review (`PROJECT-REVIEW-2026-11-19.md`) was re-checked; its findings are noted below.

## Findings and what became of them

| Priority | Finding | Outcome |
| --- | --- | --- |
| P2 | `textDocument/implementation` (`internal/server/implementation.go`) followed only `extends`, so a component that `implements` an interface was never found, though the handler's comment promised "CFC/interface". | Fixed. The index now records `implements` (`parser.ParseResult.Implements`, `Index.SetImplements`, `Index.ImplementersByName`); an interface's implementers seed the subtype walk. Recorded by both the open-document and workspace-scan paths. |
| P2 | A child whose `extends` did not resolve was accepted when its last segment matched, so a common name (`Base`, `Service`) returned unrelated components. | Fixed. A child counts only when its `extends` resolves to the file it extends from, as `typeHierarchy/subtypes` already did. |
| P3 | The header check matched only the line holding the `{` or `>`, so a multi-line declaration answered on one line. | Fixed. The whole declaration, keyword to terminator, answers. |
| P3 | The document lock was held while other files were read and indexed. | Fixed. It now covers only the parse and the cursor test. |
| P3 | MCP prompt arguments were placed in the message the model reads, bounded only by `%q` quoting. | Fixed. Arguments with control characters or over 512 bytes are refused (`checkPromptArgument`). Low risk before: the client chooses the arguments. |
| Maintainability | `script_parser.go` 5,283 lines; `result.go`, `cfscript_formatter.go`, `resolve.go`, `producer_flow.go` each over 2,000. | Split by concern into 30 files, none over 872 lines except `script_parser.go` (1,318). Declarations moved unchanged. |

Not defects: `scripts/dev-env.sh check` reports the pinned Go as unavailable when only a toolchain-downloading `go` is on `PATH`. It tests with `GOTOOLCHAIN=local` on purpose, and `setup` is the remedy.

## The 2026-11-19 findings, re-checked

Read from the code, not reproduced: the routes format is validated before the output file is opened; a watched file that becomes binary is removed from the index; the changed-lines git diff forces its prefixes and handles the tab-terminated header; route findings sort on route, file, line, source and name; `--limit 0` means unlimited in the text and Markdown reports. The stub pass-count gate passed in `make check`.

## Verification

- `make check` passed: build, vet (including `-tags cflint_embed`), gofmt, `go test -short -race`, the pass-count gates, lint (0 issues) and govulncheck (none).
- New tests: interface implementers, multi-line header, unresolved same-name `extends`, `implements` after a workspace scan, parser recording (script quoted and unquoted, tag), index recording, re-index and removal, MCP prompt argument limits.
- Every split was checked by comparing the sorted non-blank code lines before and after: only import lines differ. `TestEveryAcceptPathRecordsATarget` now reads every `resolve*.go` file.
- A build from before the splits and a build from this branch gave byte-identical `unresolved --json` on `testdata` and `internal/frameworkapi/stubs`, and identical `format` output for `testdata`.

## Not done

- No real-world corpus was available, so `make corpus` and `make resolution-report` did not run. Those are the repository's check for moved resolver code and its formatter guard. Run `make resolution-report CORPUS="name=dir" BASELINE=<dir>` against `main` and this branch before merging.
- The implementers lookup resolves each candidate's `implements` names with the component resolver. An interface named by a `componentResolvers` or mapping rule the resolver cannot follow is not found. `implements` written outside the component header is not read.
- `script_parser.go` (1,318 lines) and `result_merge_regions.go` (436) are the largest of the split files; splitting further would cut through closely coupled code.
