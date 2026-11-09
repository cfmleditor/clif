# Testing and lint conventions

_Fixtures, golangci-lint rules and why, and the gocritic/gofumpt conventions. Moved out of CLAUDE.md unchanged._

- Test fixtures live in `testdata/` (`beans/`, `chain/`, `deps/`, `refs/`, `models/`,
  `services/`, `includes/`, plus `Application.cfc` and assorted `.cfm`/`.cfc` files). Server,
  deps, and resolve tests locate them relative to the source file via `runtime.Caller(0)`.
- Formatter golden-output tests: `make visualtest` (`TestFormatOutput`); comparison fixtures
  `testdata/comparison.cfm` / `comparison.html`.
- The formatter's whitespace-only claim is checked against external corpora, not fixtures:
  `make corpus CORPUS=<dir>` (`internal/formatter/corpus_test.go`). It skips without a corpus.
  Reach for it after any formatter change — a rule that reads as obviously safe has repeatedly
  turned out to delete code on some construct no fixture contains. `.ai/docs/FORMATTER-ISSUES.md` records
  the current numbers and the six projects they were measured against.
- `.golangci.yml` (v2 config) enables `wsl_v5`, `nlreturn`, `revive`, `gocritic`, `gosec`,
  `errorlint`, `exhaustive`, `prealloc`, and others. **`wsl_v5` + `nlreturn` demand a blank line
  before `return` and around block statements** — this is why existing code looks the way it
  does; match it or `make lint` fails. **Lint via `make lint`, never a `golangci-lint` on
  `PATH`.** The Makefile pins the version (`GOLANGCI`) and builds it from source under this
  module's toolchain, because golangci-lint refuses to load a config whose module targets a newer
  Go than the linter binary was built with — `can't load config: the Go language version (go1.25)
  used to build golangci-lint is lower than the targeted Go version (1.26.6)`. That is a refusal
  to start, not a finding, and it is what a distro or Homebrew binary does for weeks after each Go
  bump. It builds once, into `target/tools/golangci-lint-<version>-go<version>`, and CI caches
  that file and golangci-lint's analysis cache between runs; a cold lint job was about 50s
  building the linter and 90s analysing. `gosec` runs everywhere except, in `cmd/`, the rules about opening and writing the paths
  the CLI is given; anything else it flags is fixed or carries its reason. **Convert a line or
  column with `conv.Uint32`, not `uint32(n)`**: `internal/conv` clamps where a bare conversion
  wraps a negative to about four billion, and `G115` flags the bare form. Test files are exempted from `prealloc` and `gosec` only: a test's
  slices are not worth pre-sizing, and its paths and permissions are fixtures.
  `staticcheck` and `unparam` run on tests too, since a deprecated API or a
  helper parameter nobody varies is as real there as anywhere.
- **gocritic runs every `performance` check, plus `whyNoLint` and
  `emptyStringTest`, in test files too.** `hugeParam` and `rangeValCopy` are at
  gocritic's own 80 bytes: a large struct goes by pointer, and a loop over large
  elements indexes (`x := &xs[i]`). The trap is a callee that edits what it was
  given: `formatter.New`, `codemap.Build` and `refs.Trace` all write to their
  options, so each takes `o := *opts` first — and
  `TestNewDoesNotWriteToTheCallersOptions`,
  `TestBuildDoesNotWriteToTheCallersOptions` and the `Trace` test fail if that
  copy is dropped. Where a copy is the point (appending the value to a result,
  a loop that edits the element and keeps it, a snapshot of `s.Formatting`
  taken before the read loop is released), write it as `*p`, `v := xs[i]` or
  `cfg := s.Formatting` rather than suppressing the check. Passing
  `&CallSite{…}` to `addCall` does not allocate — the parameter only leaks its
  content — and the parse benchmarks' bytes and allocations were unchanged when
  it went in; check `go build -gcflags=-m` if a new such helper keeps the
  pointer. `whyNoLint` wants a reason after every `//nolint:x`, and
  `nolintlint` fails on one that no longer suppresses anything.
- **Formatting is gofumpt, enforced by `make lint`.** It is `gofmt` plus a few
  rules: no blank line at the start or end of a block, one element per line
  once a composite literal spans lines, no redundant parentheses, and
  standard-library imports in one group — `encoding/json/v2` included. Run
  `make fmt` rather than fixing a finding by hand.
- **The linter set is chosen for what it catches, not for style.** Beyond
  `default: standard`, `.golangci.yml` enables linters that flag defects and
  cost nothing on clean code (`bidichk`, `nilnesserr`, `recvcheck`, `musttag`,
  `sqlclosecheck`, `rowserrcheck`, `noctx`, `errchkjson` and others), and
  `staticcheck` runs every check. Each was measured over the whole codebase
  before it went in; the high-volume style linters (`varnamelen`,
  `exhaustruct`, `paralleltest`, `lll`, the complexity limits) were measured
  too and left off. `noctx` is excluded for `internal/codemap/store`, whose
  callers hold no context to pass; `dupword` checks only the short words typed
  twice by accident, since checking every word flags deliberate repetition.
  **Naming a revive rule replaces revive's default set**, so
  `.golangci.yml` lists the 23 defaults before the additions; a rule added
  without them switches the defaults off with nothing to say so. Prove a new
  check runs with a throwaway file that breaks it — a check that finds nothing
  and a check that is not running look the same — and run that with
  `--uniq-by-line=false`, or a second finding on a line is hidden behind the
  first. **A type assertion in a test is checked or discarded, never bare**:
  `v, ok := x.(T)` with a `t.Fatalf`, or `v, _ := x.(T)` where the next check
  fails on the zero value — which is only true if that check does not
  dereference it. `thelper` does not check benchmark functions, because every
  one here is a body handed to `testing.Benchmark`.
  **Complexity has limits, and nothing is excused from them.**
  `gocognit` 50, `nestif` 10, `funlen` 80 statements. The 42 functions that
  were over one when the limits went in have all been split; do not add a
  `//nolint` to get past a limit — split the function. `dupl` runs outside
  tests for the same reason: the two parsers' `extractAllLinks` and three
  copies of the chain walk were what it found. `depguard` holds three package boundaries
  (`internal/parser` imports only `internal/log` and the dependency-free `internal/conv` and `internal/lex`; only `daemon` and `cmd`
  import `internal/server`; only `cmd` imports the code-map store and MCP
  server), and `forbidigo` bans printing to stdout under `internal/`.
  **Write an empty slice as `[]T{}`, and keep it one where it is marshalled.**
  `enforce-slice-style` is `literal`, not `nil`, because a nil slice encodes
  as JSON `null` and an LSP response or the code map expecting `[]` then
  breaks a client. `use-slices-concat` pulls the other way — `slices.Concat`
  of empty inputs returns nil — so where a concatenation is marshalled,
  build it on an explicit `[]T{}` instead. An `if` init statement stays on one line
  (`multiline-if-init`): when the call wraps, assign above the `if` and test
  `err` alone.
  `.ai/docs/LINT-PLAN.md` holds the stages still to come, the measurements behind them,
  and the rules left off with the reason for each; update it when a stage
  lands or the pinned golangci-lint moves.
- `internal/docs/` content is generated — regenerate rather than hand-editing, but see the
  lossy-regeneration warning under Commands before committing any change to it.
- `.github/workflows/ci.yml` runs on every pull request: `build-test` (build, vet, gofmt, `go
  test -short`), `race`, `lint` (`make lint`), `vuln` (`make vuln`), and an informational
  `perf` job that never gates a merge. None of them run `make docs`, since
  `internal/docs/generated_docs.go` is committed.
