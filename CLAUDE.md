# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Read `AGENTS.md` and run `bash scripts/dev-env.sh check` at the start of a
fresh environment. `bash scripts/dev-env.sh setup` provisions the pinned
Go toolchain, module cache, and linter; source the emitted `env.sh` afterward.
For sandbox proxy failures, follow the execution-context retry guidance in
`AGENTS.md` before diagnosing an outage.

```bash
make build          # build target/release/clif from the committed docs, with CFLint inside, as
                    # releases are (scripts/fetch-cflint.sh, cached in internal/cflint/embedded/)
make check          # everything CI runs on a PR: build, vet (incl. -tags cflint_embed),
                    # gofmt, go test -short -race, lint, vuln
make test           # go test ./...
make lint           # golangci-lint run ./... (pinned scanner, built from source)
make lint-fix       # golangci-lint run --fix ./...
make vuln           # govulncheck ./... (pinned scanner, GOWORK=off)
make fmt            # gofmt -w . && golangci-lint run --fix ./... (gofumpt included)
make install        # build and copy to `go env GOPATH`/bin
make link           # build + symlink onto PATH for local editor use (LINK_DIR=<dir> to override)
make unlink         # remove that symlink
make link-status    # show the link, the build, and what PATH resolves clif to
make update-grammar # bump tree-sitter-cfml, regen docs + injections.scm, clear build cache
make generate       # fetch the pinned cfdocs (docs/sources.lock) and regenerate the docs
make docs-bump      # move the cfdocs pin and the vendored docs/lucee to upstream's latest; then `make generate`
make framework-stubs # regenerate internal/frameworkapi/stubs from each pinned framework
                    # source (needs git + network; the stubs are committed)
make update-d3      # rebuild the code-map viewer's D3 bundle from assets/vendor/entry.js
                    # (needs Node; the bundle is committed so `go build` does not)
make cfparse        # build + run the parser-benchmark CLI (cmd/cfparse)
make visualtest     # go test -v -run TestFormatOutput ./internal/formatter/
make gapcheck [CORPUS=<dir>[:<dir>...]]
                    # diff what internal/parser extracts against what the tree-sitter
                    # grammar sees in the same file. Without CORPUS it holds the repo's
                    # fixtures to a recorded list of differences; with one it reports.
                    # 6s over 5,629 files. See .ai/docs/PARSER-GAPS.md
make corpus CORPUS=<dir>[:<dir>...] [REPORT=<file>] [BASELINE=<file>] [OPTS=k=v,...]
                    # format a real-world CFML corpus and report what the formatter did to
                    # each file (clean / grammar-refused / guard-rejected / not idempotent /
                    # malformed); skipped without CORPUS, so it never runs in CI.
                    # BASELINE=<an earlier REPORT> fails the run if any file changed verdict —
                    # the totals cannot show that, since one file breaking and another being
                    # fixed leaves every column identical. OPTS sets formatter.Options fields
                    # by name, for sweeping a new setting through its modes without editing
                    # the test. See .ai/docs/FORMATTER-ISSUES.md
make resolution-report CORPUS="name=dir[,dir] ..." [BASELINE=<dir>] [RUNS=<dir>] [LISTS=<dir>]
                    # unresolved --json --candidates per named scan; rewrites the lists in
                    # .ai/docs/resolution-candidates/ (LISTS= skips that) and, with BASELINE, prints
                    # findings removed/added per scan. Skipped without CORPUS; not in CI
make build-wasm     # wasip1/wasm build (needs WASI_SDK, default /opt/wasi-sdk)
make release <ver>  # validate, build, test, lint, changelog, commit, tag, push
make release-dry <ver>
make clean          # remove target/

# Run a single test
go test ./internal/parser/... -run TestParseFunctionDefs_MixedTagAndCFScript
go test -v ./internal/formatter/ -run TestFormatOutput

# Benchmarks live in internal/parser/{benchmark,tassweb_bench}_test.go
go test ./internal/parser/ -bench . -run '^$'
```

**Only `make generate` fetches docs.** `make build` compiles the committed
`internal/docs/generated_docs.go`, as CI and the release workflow do, so it works offline (it does
fetch CFLint, and builds without it when it can't). `make generate` runs `docs`, which git-clones
the pinned cfdocs commit into the gitignored `docs/src/` (Lucee's docs are committed), then
regenerates. Plain `go build ./cmd/clif`, `go test ./...` and `make lint` never fetch docs. (`make
lint`'s first run needs the network once, to fetch and build the pinned linter; after that it is
served from the build cache.)

**The docs pipeline has three sources: cfdocs (fetched), `docs/retained` and Lucee (both
committed).** `internal/docs/generated_docs.go` is generated from them, each read in place:

```
docs/sources.lock              (committed: the cfdocs commit to fetch)
scripts/fetch-docs-cfdocs.sh → docs/src/cfdocs   (the pinned commit; marker docs/.sha-cfdocs)
docs/retained                  (committed, not fetched: entries an upstream stopped publishing)
docs/lucee                     (committed: Lucee's docs, vendored; provenance in docs/lucee/SOURCE)
scripts/assemble-docs.sh     → docs/data/order (each source, lowest precedence first, and its directory)
```

**The sources are pinned, so the same commit regenerates the same docs on any machine.**
`docs/sources.lock` names the cfdocs commit to check out, and the generator refuses unless the
staged cfdocs (its marker, `docs/.sha-cfdocs`, written only once the staged copy is whole) is
exactly that commit. Lucee publishes only its latest zip, so a version of it cannot be fetched again
later; its JSON is vendored in `docs/lucee` instead, and `docs/lucee/SOURCE` records the zip it came
from and a hash of the vendored JSON, which `TestVendoredLuceeMatchesItsSource` checks, so a hand
edit there fails CI. Upstream changes arrive only through `make docs-bump`, which stages cfdocs'
master, checks its commit, refuses a Lucee zip holding a file that does not parse or names no valid
entry, then replaces `docs/lucee` and writes the lock last, so nothing committed changes unless both
succeed: follow it with `make generate` and review the lock, `docs/lucee` and `generated_docs.go`
diffs together. Only `docs-bump` writes the lock; a missing pin is an error.

`make docs` fetches cfdocs at the pinned commit (retrying a failed network step twice) and assembles
the order; a failed fetch fails it. `make docs-cfdocs` does the same; there is no Lucee fetch
outside `make docs-bump`. `SOURCES` in `assemble-docs.sh` is the only place the precedence is
written, and the generator takes each entry from the highest source that documents it, whatever the
file is called: Lucee outranks `docs/retained`, which outranks cfdocs, and when Lucee documents a
retained name again the generator says the retained file can go. Every source is read where it is,
never copied, so a file just added, fixed, broken or pulled always counts. The staged cfdocs and the
vendored `docs/lucee` are each built beside themselves and swapped in (`scripts/lib/swapdir.sh`),
with the old copy restored after a failure or an interrupt, and `docs/data/order` is written to a
temporary file and renamed into place. A lock keeps two docs runs from interleaving: the operating
system's own file lock on `docs/.flock` (`internal/flock`: `flock` on Unix, `LockFileEx` on
Windows), held by `scripts/docslock.go` alone, so it is released however that process ends, `kill
-9` and a power cut included, with no stale lock to judge. docslock gives the command it runs a
fresh token (`DOCS_LOCK_TOKEN`), recorded in `docs/.flock` with its PID; a process proves it runs
under the lock by that token matching while the lock is held (`flock.Verify`). No descriptor is
passed down, so nothing a run leaves behind (git's background gc, the go command's telemetry) can
keep the lock, and a token leaked from a finished run matches nothing. Every docs script runs itself
under it (`scripts/lib/underlock.sh`): if `docslock --held` agrees it carries on, otherwise it
re-runs through docslock from the repository root. docslock is built and run
(`target/docs-gen/docslock`, rebuilt only when older than its source), never `go run`, which turns
every failing exit into 1. `make docs` and `make docs-cfdocs` (`scripts/docs.sh`), `make generate`
(`scripts/generate.sh`, which runs `docs.sh` then the generator) and `make docs-bump` each hold it
across all their steps. The generator, built and run as a binary so its exit code comes through,
refuses unless it runs under the lock, and checks again just before it writes (docslock killed, or
the file deleted, meanwhile); so run it through `make generate`. The command docslock runs leads a
process group of its own, put in the terminal's foreground when docslock was there (so it can
prompt, and write under `stty tostop`), and Ctrl-Z is ignored, since a run stopped holding the lock
could not be resumed. A Ctrl-C reaches that group once; a signal sent to docslock is passed to the
whole group; the scripts trap these, so bash waits for what it is running to stop before its
cleanup, which ignores any further signal. The generator holds a signal that arrives while it writes
the two generated files until both are in place, then exits 128 + the signal's number. While it
runs, docslock keeps the previous version's lock, a `docs/.lock` symlink naming `<pid>@<start
time>`, pointing at itself, so a run of that version finds a live owner and refuses; one left behind
is replaced once its owner is gone. The scripts retry a failed network step twice
(`scripts/lib/retry.sh`), and work from the repository root wherever they are run from.

**`scripts/generate_docs.go` refuses to lose an entry.** It compares the new entries, by type and
name, with `internal/docs/generated_sources.txt` as committed at `HEAD`, or at `DOCS_BASELINE_REF`
when that names another commit (`git show REF:./…`, so it also works with clif inside another
repository), never the working copy, so an uncommitted regeneration that already lost entries cannot
become the baseline. If any are missing it prints a banner listing them and writes nothing. It
refuses the same way when an entry would come from a lower-ranked source than before (Lucee's, or a
retained one, replaced by cfdocs' Adobe entry): the name survives, so nothing is "missing", but the
signature changes. Ranks are judged by the order the baseline was generated under (its `# order:`
line), so a source missing today still ranks where it did. A move to `docs/retained` is never a
fallback: copying in an entry Lucee dropped is the fix. `generated_sources.txt` records each entry's
source; `TestGeneratedSourcesMatchTheDocs` keeps it in step with `generated_docs.go`. Both are
written to temporary files first and renamed into place only once both are written, so a full disk
cannot split them; a rename failing between the two can, and that test then fails until a
regeneration rewrites both. **Commit them together.**

With no baseline to compare with (no `generated_sources.txt` at `HEAD`, a `HEAD` that does not
resolve, or no usable git to ask: not installed, or a checkout git distrusts) it refuses rather than
check less; `DOCS_ALLOW_REMOVALS=1` creates the file from nothing, once, deliberately. A committed
sources file that git cannot read (a shallow clone), is malformed or lists nothing is refused too,
never taken for "no baseline". An upstream file (in the fetched cfdocs or the vendored `docs/lucee`)
that does not parse is skipped with a warning, as upstream's to fix (cfdocs has shipped
`cfcatch.json` and `cfcol.json` with merge-conflict markers in them); if that costs an entry, the
removal check names it. `make docs-bump` refuses to vendor such a Lucee file in the first place. A
`docs/retained` file that does not parse is refused: it is written by hand here. A source directory
with no JSON at all is refused too, except `docs/retained`, which may become empty.

On a refusal `make generate` exits 1 and writes nothing; the banner notes when the generated files
on disk differ from `HEAD`'s or are not committed (`make build` compiles whatever is on disk).
Nothing else regenerates: `make build` and releases use the committed docs, so a failed fetch or an
upstream removal never touches a build. A `docs/retained` file that does not parse or is not a
function or tag is refused like the rest, and so are two retained files for one name (one would be
dropped silently, with its signature). `make update-grammar` bumps the grammar first and runs
`make generate` last, so a docs problem neither blocks the bump nor passes unnoticed.

Two things make entries go missing:

- **A source can't be fetched.** A blocked GitHub means no staged cfdocs, which would drop every
  cfdocs-only entry; a blocked `docs.lucee.org` (some sandboxes and proxies deny it) now only stops
  `make docs-bump`, since Lucee's docs are committed. The generator refuses either way. Fix the
  fetch.
- **An upstream stops publishing part of itself.** In 2026-11 docs.lucee.org stopped listing about 50
  of Lucee's crypto extension functions (`JwtSign`, `BCryptVerify`, …), and the unresolved report
  then flagged every call to them. Hours later it dropped `Argon2CheckHash` and `GenerateArgon2Hash`
  too, which cfdocs' Adobe entries silently replaced. Those now live in `docs/retained/` (see its
  README). If the engine still has a function, add its entry there. If it really is gone, rerun
  `make generate` with `DOCS_ALLOW_REMOVALS="function foo"`: entries by name, comma-separated, so
  accepting one cannot carry an accidental loss through with it (`1` accepts everything).

**Never use `DOCS_ALLOW_REMOVALS` to get past a failed fetch**, and never commit a
`generated_docs.go` diff that deletes entries nobody decided to drop: `git checkout --
internal/docs/generated_docs.go internal/docs/generated_sources.txt`.
`TestRetainedLuceeEntriesAreDocumented` checks that every retained file reaches the generated docs
with its own syntax and parameters (or that Lucee now documents it), and
`TestLuceeOnlyEntriesArePresent` that the Lucee-only entries are there; the release workflow runs
`go test ./internal/docs/...`, once, ahead of the platform builds, rather than grepping the
generated file. CI's `docs-reproducible` job runs `make generate` once and fails if the committed
docs change, so `generated_docs.go` is always what the pinned and vendored sources produce: a hand
edit, or a pin bumped without regenerating, fails it. On a pull request that run compares with the
base the merge commit was built on (`HEAD^1`, as `DOCS_BASELINE_REF`; not the event's base SHA,
stale on a re-run after main moves) rather than `HEAD`, the PR's own commit, which would hold
nothing to miss if the PR itself committed a lossy regeneration (a base from before
`generated_sources.txt` existed has nothing to compare, and only notes so; that exception needs both
`DOCS_BASELINE_REF` and `DOCS_ALLOWED_REMOVALS_FILE`, as CI sets them, and any other run without a
baseline is refused). Only the losses `docs/allowed-removals` lists pass there, and only the lines
the PR adds to that file (those the base's copy lacks) accept anything, so dropping an entry on
purpose takes a reviewed change to it. A line left over from an earlier pull request (unchanged from
the base's copy) accepts nothing, so it can neither wave a later loss through by accident nor fail
anyone; it only warns until cleared. To remove such an entry again on purpose, change its line (a
new `# reason`): a changed line counts as added; a change of spacing alone does not. A changed line
that removes nothing (a comment tidied) only warns; only an entry new to the file that removes
nothing is refused. If the base's copy itself no longer parses, every line counts as added, with a
warning, and a line that removes nothing only warns, so it cannot block the pull request that fixes
it. The generator reads the file itself (`DOCS_ALLOWED_REMOVALS_FILE`, through
`sources.ParseAllowedRemovals`), strictly: a line it cannot read, `1` (which would switch the
comparison off), or an added line that accepts nothing (a typo, a removal since undone) fails the
step. `DOCS_BASELINE_REF` must name a commit; a value starting with `-` is refused, as git would
read it as an option.

Go toolchain is pinned at **1.27.4** (`go.mod`). CGO is required (tree-sitter grammar).

### CLI subcommands

The binary is an LSP server by default; `os.Args[1]` selects a subcommand
(`cmd/clif/main.go`).

| Command | Usage |
|---|---|
| *(none)* | Run the LSP server over stdio (JSON-RPC 2.0, Content-Length framing) |
| `parse` | `parse <file-or-dir> [...]` — parse and report per-file timing/counts |
| `scan` | `scan <file-or-dir> [...]` — report parse errors |
| `format` | `format [-w] [--allow-non-whitespace] [--root <dir>] [--follow-symlinks] <file> [...]` — format to stdout, or in place with `-w`. Reads `formatting` from each file's governing `.clif.json`, or from `--root`'s |
| `unresolved` | `unresolved [--json] [--verbose] [--global-defs] <dir> [...]` — batch scan for unresolvable component/method calls |
| `check` | `check [--checks cflint,unresolved,scan,format] [--format text\|json\|github] [--min-severity <l>] [--no-baseline] <path> [...]` — run the checks the `check` config block names (CFLint by default) and exit 1 when one finds more than its `maxFindings`, 2 when one cannot run. Each check reuses its subcommand's code (`cflint.Runner`, `scanUnresolved`, `parseErrors`, `formatFile`); cflint and unresolved default to the known-issues reports the config generates as baselines (`cmd/clif/check.go`, `config.ResolvedCheck`) |
| `refs` | `refs [--mermaid] <component-or-function> <dir> [...]` — find references |
| `deps` | `deps [--mermaid] <dir-or-file> [...]` — transitive dependency graph, built through `deps.Build` so the CLI and `clif.exportDeps` answer alike |
| `graph` | `graph [--level function\|call\|file\|package] [--format text\|json\|jsonl\|dot\|mermaid\|html] [--db <file>] [--out <f>] [--live\|--detached\|--from <id>] [--under <p>] <dir> [...]` — whole-project code map |
| `mcp` | `mcp [--db <file>] [--root <dir>] [--allow-lint] [--map-only] [--no-explain]` — serve the workspace over MCP on stdio: unresolved, refs and explain from source, the map's tools with `--db`, CFLint with `--allow-lint`. Writes nothing: the map is opened read-only (`store.OpenReadOnly`), and every path a tool is given must lie under `--root` (or the map's root, or the working directory) after symlinks |
| `explain` | `explain [--root <dir>] [--no-infer-args] <file> <line> [call-substring]` — trace how a call site resolved, and what the unresolved report does with it |
| `unresolved-diff` | `unresolved-diff [--show <n>] [--root <dir>] <before.json> <after.json>` — compare two `unresolved --json` runs finding by finding, the removed and added grouped by reason. How to judge a resolution change: run the old and the new build over a corpus, save each `--json`, diff them (`make resolution-report BASELINE=… SHOW=5` does the same through `unresolved.Diff`) |
| `version`, `help` | |

`cmd/cfparse` is a separate debug binary for parser development (timing + `-cpuprofile`).

## The old name: cfmleditor-lsp

This project was cfmleditor-lsp, and every name it had is still honoured, so
neither editors nor projects have to change at once. **Write the new name
everywhere; read both.** README.md's "Formerly cfmleditor-lsp" lists each pair.
The fallbacks live in one place each, with a test that fails if it goes:

- config file: `config.FileNames` (`.clif.json` before `.cfmleditor.json`), used by
  `daemon.FindConfig`, `Server.findConfigUpwards` and the
  code map's config preload. `TestFindConfigPrefersClifJSON`, `TestAClifConfigOutranksALegacyOne`.
- commands: `canonicalCommand` maps `cfmleditor.X` to `clif.X` before dispatch; only
  `clif.*` is advertised. `TestLegacyCommandNamesStillRun`, `TestOnlyTheNewCommandNamesAreAdvertised`.
- default report files: `config.defaultReport` keeps an existing legacy-named report.
  `TestADefaultReportKeepsItsLegacyName`.
- environment: `cflog.Getenv(new, legacy)`. `TestGetenvFallsBackToTheLegacyName`.
- binary: the release workflow publishes `cfmleditor-lsp-<platform>` archives beside
  `clif-<platform>`, and `make build`/`install`/`link` create both names.

A new config key, command, env var or default file name needs no legacy twin — only
names that shipped under cfmleditor-lsp do. Keep CHANGELOG.md's history as it was
written.

## Architecture

An LSP server for CFML/ColdFusion written in Go, backed by the `tree-sitter-cfml` grammar.

**Two runtime modes**, selected at startup by whether `daemon.FindConfig(cwd)` finds a
`.clif.json` (walking from the current directory up to the filesystem root):

- **Daemon mode** — the first LSP client becomes the daemon: it listens on a Unix socket (path
  derived from `workspaceName`) *and* serves that first client over stdio, sharing one
  `index.Index`. Later clients `daemon.Proxy()` into the socket. A `ConnTracker` shuts the
  daemon down when the last client disconnects.
- **Standalone mode** — no config file; a single self-contained session with its own index.
  `FindConfig` returns nil when its walk finds nothing, which is what selects this mode. It used
  to return a `Config` with an empty `Path` and the working directory's base name as its `Name`,
  making the standalone branch unreachable and keying the socket — a hash of `Name` — on that
  base name, so unrelated projects in folders with the same name shared one daemon and one
  index. Every caller already tested for nil.

**A config file another local user could have written is never read.** Discovery walks to
the filesystem root, so a `.clif.json` planted in `/tmp` or `/Users/Shared` would govern every
project below it. Every lookup of a config file goes through `config.CandidatesIn` or
`config.TrustedCandidates` (`internal/config/trust.go`), which on unix skip a file not owned by
the current uid or root, or writable by others, or by a group that is not the user's private
group (`Untrusted`, `trust_unix.go`); the walk then goes on upwards. The server logs a skip;
`cmd/clif` reports one on stderr through `config.ReportUntrustedTo`. Windows and wasm do not
check. `TestEveryConfigLookupChecksOwnership` parses the module and fails on any use of
`config.FileName`, `LegacyFileName` or `FileNames`, or the names spelled as literals, outside
those two functions, so a new walk cannot bypass the rule.

Note the repo root has its own `.clif.json` (`workspaceName: testdata`), so running the
binary from the repo root enters daemon mode against `testdata/`.

**Core data pipeline:**

```
Editor document change
  → server (internal/server) receives textDocument/didChange
  → parser (internal/parser) produces ParseResult
      ├─ ClassifyRegions() splits file into Script vs Tag regions
      ├─ scriptParser extracts function defs + component refs via line scanner
      └─ tagParser extracts function defs + component refs via tag search
  → index (internal/index) stores function definitions, keyed by lowercase name
  → resolver (internal/resolve) maps dot-paths → .cfc file paths
  → cached in server per URI (parseResults, compCache, resolveCache)
```

### Package map

| Package | Responsibility |
|---|---|
| `internal/parser` | Line-scanner/tag-search parsing → `ParseResult`; resolver matching (`ast.go`) |
| `internal/server` | LSP handler wiring, completion, definition, hover, symbols, signature help, code actions, document links, formatting, on-type formatting, watched-file reindexing, workspace commands, bean scanning |
| `internal/index` | Concurrency-safe store of function defs, component refs, beans, ORM entities. `HasFile` answers "indexed at all" — not the same as `FunctionsForFile` returning nothing, since a property-only bean indexes to an empty but present entry. Two views of every entry — the name buckets (`funcs`/`comprefs`) and the per-file lists (`fileFuncs`/`fileRefs`) — hold the same pointers, and **every writer must fill or clear both**; `removeFileEntries` reaches the buckets *through* the per-file lists, so a writer that updates one view alone leaves entries no removal can find. **Accessors return a copy** (`snapshot`), which is what lets writers compact and rewrite buckets in place rather than rebuild a slice that, for a name every component declares, holds one entry per file — pinned by `TestAccessorsReturnStorageWritersDoNotTouch`. On a per-keystroke path reach for `LookupPreferred`/`CountFunctions`, not `Lookup`, which pays that copy. **Bucket order is not an answer**: entries land in the order a parallel workspace scan finished, so it differs between restarts — where several files declare a name and none is the requesting file, `LookupPreferred` picks the nearest by `cfpath.URIDistance` and the lowest URI among equals, and `definition.go` orders its multi-location list the same way |
| `internal/resolve` | The type-inference and call-resolution engine, about a third of the code. Dot-path → `.cfc` file (`component_path.go`), function lookup and the extends chain (`func_lookup.go`), what a function returns (`return_component.go`) and what a receiver holds (`receiver_component.go`), `CanResolveCall`/`ExplainCall` (`resolve.go`), cfinclude scope, the report's policy layer (`classify.go`: an ordered table of named rules, each marked judgement or proof, the judgement ones off in the strict view; `explain` names the one that decided), and one file per framework or inference rule (`wheels*`, `mura*`, `coldbox*`, `producer_*`, …) |
| `internal/path` | Case-insensitive path resolution, mappings, globs, `Application.cfc` mapping/bean/ORM extraction, binary + CFML file detection |
| `internal/config` | `.clif.json` schema (`config.JSON`), defaults, `JavaStubResolver` |
| `internal/daemon` | Unix socket serve/proxy, connection tracking, config discovery |
| `internal/formatter` | tree-sitter CST-walking formatter (elements, cfscript, cfquery/SQL) |
| `internal/language` | tree-sitter language handles (`CFML`, `CFScript`, `CFQuery`) + injection queries |
| `internal/docs` | Built-in CFML function/tag signatures and return types (**generated — do not hand-edit**) |
| `internal/docs/sources` | The docs generator's checks, compiled and tested here rather than in the build-ignored `scripts/generate_docs.go`: parsing `generated_sources.txt` (shared with the `internal/docs` tests, so they agree on a valid line), entry names, removals, fallbacks to a lower source, per-entry precedence (`Pick`), `DOCS_ALLOW_REMOVALS`, and the names in a `generated_docs.go`. Imports nothing from `internal/docs`, so a broken generated file cannot stop the generator |
| `internal/flock` | An exclusive, non-blocking OS file lock (`flock` on Unix, `LockFileEx` on Windows), released however its holder ends, with the holder's record (`Record`, `Holder`, `Verify`) and the process-group helpers its holder needs; `scripts/docslock.go` holds `docs/.flock` with it |
| `internal/cflint` | Downloads/runs the CFLint binary, maps JSON output to LSP diagnostics |
| `internal/cache` | Per-file, per-scope completion item cache with content hashing |
| `internal/refs` | Shared reference-finding + `Trace` (multi-hop wrapper following) for the `refs` CLI, `clif.findRefs` and `textDocument/references` |
| `internal/frameworkapi` | The presets' frameworks' API, bundled: generated stubs (`stubs/`, from `cmd/cfstubgen`), the overlay filesystem that serves them under the virtual `Root`, and their doc comments (`DocAt`). See "Framework presets" in [`resolvers.md`](.ai/docs/guide/resolvers.md) |
| `internal/route` | Convention-based framework routing: the `routes` config grammar, the source scanner, and resolution to a controller method or a view |
| `internal/codemap` | Whole-project map: every function, file, and the calls/instantiations/inheritance/includes between them. The **inverse** of `internal/deps` — see the note in [`codemap.md`](.ai/docs/guide/codemap.md) |
| `internal/codemap/store` | SQLite persistence + the per-file parse cache (`!wasip1`; a stub declines on wasm) |
| `internal/codemap/mcp` | MCP server: the map tools over the store, and the task tools (unresolved, refs, explain, lint) as hooks `cmd/clif/mcp_tasks.go` supplies, so the tool and its CLI command run the same code. A nil hook or store is a tool not advertised |
| `internal/deps` | Transitive dependency graph builder, the single implementation behind both the `deps` CLI and `clif.exportDeps`. Two traversals: file-level, which walks `Index.RefsForFile`; and function-level, which needs an `Options.LoadCalls` hook, because the index stores definitions and refs but no call sites. Without that hook the function-level graph stops after one hop |
| `internal/tsoracle` | Differential check: what `internal/parser` extracted vs what the tree-sitter grammar saw in the same file. See the note in [`verification.md`](.ai/docs/guide/verification.md) |
| `internal/textdiff` | Myers line diff, for range formatting: which lines the formatter changed and what each became |
| `internal/graph` | Graph type + Mermaid renderer |
| `internal/vfs` | `FS` interface + stdio transport, abstracted for native vs WASM builds. **Every file read goes through `ReadRegular`** (`vfs.OS` and every former `os.ReadFile`): it refuses a device, a FIFO or a socket, whatever a symlink at the path points to, and anything over `MaxFileSize` (32MB). A workspace file symlinked to `/dev/zero` took a scan to 17GB in eight seconds, and a FIFO hung indexing for good; the open is non-blocking so a FIFO cannot wait for a writer, and the checks come from the opened file. `TestReadRegularRefusesADevice`, `TestReadRegularDoesNotWaitOnAFIFO` |
| `internal/log` | zap wrapper; `debug: true` in config switches to `zap.NewDevelopment`. `EnableCrashReports` (called from `runServer` only, so test binaries never write to the user's cache) sends the runtime's crash output and every panic record to `CLIF_LOG` or, unset, `<os.UserCacheDir>/clif/crash.log`. A recover site calls `cflog.Recovered`, not `log.Error`, so the record carries its stack and reaches that file |
| `internal/lex` | Byte classes and small string helpers several packages had each written for themselves (`IsIdent`, `IsWord`, `SkipSpace`, `LowerASCII`, `HasPrefixFold`, `ContainsFold`, `UTF16Len`, and the `Wild` backtracking matcher behind `placeholderGlob` and `wildPattern`); imports only the standard library (`strings`, `slices`, `unicode/utf8`), so the parser may use it. Only definitions that were byte-for-byte alike are here: a package whose rule differs keeps its own under a name that says how, and each helper's test holds it to every definition it replaced |
| `internal/testutil` | Helpers the packages' tests share (`Testdata`); production code does not import it |
| `internal/conv` | Range-checked integer conversions (`Uint32`, `Uint32FromUint`, `Int32`) for LSP line and column numbers; imports only `math` |
| `internal/confine` | Keeps a write, or an MCP tool's path, inside given roots after resolving symlinks (a link at the final name and a linked directory above it). A link that stays inside is followed; one that leaves is refused |


## Detailed guide

The rest of what used to be in this file lives in `.ai/docs/guide/`, moved unchanged. Read the
file for the area you are changing before you change it: each records invariants that a test
enforces and a mistake someone already made.

| File | Read it when you touch |
|---|---|
| [`parser.md`](.ai/docs/guide/parser.md) | `internal/parser`: the scanner, the two parse loops, call extraction, chains, interpolation, flow and union returns |
| [`formatter.md`](.ai/docs/guide/formatter.md) | `internal/formatter`, `format`, the whitespace-only guard and its budgets |
| [`codemap.md`](.ai/docs/guide/codemap.md) | `internal/codemap`, `graph`, and the batch pipeline `unresolved`/`explain`/`deps`/MCP share; routes; argument and callback inference |
| [`structure.md`](.ai/docs/guide/structure.md) | which file holds what, and the tests that read source files by name (read before moving code) |
| [`lsp.md`](.ai/docs/guide/lsp.md) | `internal/server`: capabilities, `workspace/executeCommand`, positions, locking, completion; what the VS Code extension has that the server lacks |
| [`text-and-includes.md`](.ai/docs/guide/text-and-includes.md) | `#...#` in text, `cfinclude` scope, included-template and Mura/ColdBox argument rules |
| [`resolvers.md`](.ai/docs/guide/resolvers.md) | `internal/resolve`, `componentResolvers`, framework presets and stubs, expression mappings, `noFollow`, triaging unresolved calls, Java stubs |
| [`testing.md`](.ai/docs/guide/testing.md) | writing tests, and what `make lint` demands and why |
| [`verification.md`](.ai/docs/guide/verification.md) | before you call a change done: mutation checks, corpus diffs, benchmarks, scaling tests |

Research, plans and audits stay in `.ai/docs/`.

## Configuration (`.clif.json`)

The authoritative schema is `config.JSON` in `internal/config/config.go`; README.md documents
the user-facing view and all `formatting` defaults.

| Field | Purpose |
|---|---|
| `workspaceName` | Required for daemon mode; derives the socket path |
| `workspacePaths`, `workspaceIndexGlobs` | Which roots / `.cfc` files to index |
| `mappings` | Virtual dot-path root → directory. A workspace folder implies one of its own name (`Resolver.inFolderNamed`, `cfpath.InFolderNamed`), tried after every mapping and relative lookup, so explicit entries are needed only for roots that are not a folder's name |
| `startupFiles` | Templates whose shared-scope assignments (`REQUEST.`, `SESSION.`, `APPLICATION.`, `SERVER.`) type a variable for the whole workspace; a leading `/` is a template path resolved like a `cfinclude`. The same lookup reads the templates the governing `Application.cfc` includes with no config (`internal/resolve/startup.go`), and is the last step in `receiverComponent`. It types the RHS itself rather than re-entering `canResolveCall`, so its recursion has its own visited set and its answer is deterministic under the parallel scan |
| `expressionMappings` | Runtime `#...#` expression → static substring (see [`resolvers.md`](.ai/docs/guide/resolvers.md)) |
| `componentResolvers` | Call expression → component dot-path (see [`resolvers.md`](.ai/docs/guide/resolvers.md)) |
| `frameworks` | Framework presets (`internal/config/frameworks.go`): each adds `dynamicIfMissing` componentResolvers after the config's own, an implicit base for a file that names no `extends`, and its bundled API for when the source is absent. See "Framework presets" in [`resolvers.md`](.ai/docs/guide/resolvers.md) |
| `propertyResolvers` | `<cfproperty>` attribute → component dot-path (`match`/`resolve`/`attribute`) |
| `servicePropertyResolvers` | `@serviceproperty <var> <kind>\|<name>` doc-comment kind → `${name}` dot-path template, for generically-typed dependencies |
| `beanPaths` | namespace → directory; `.cfc`s registered as `name@namespace`, plus a bare `name` when unique across all namespaces |
| `customTagPaths` | Custom tag roots and tag library prefixes (`config.CustomTagPath`: a directory string, or `{directory, prefix, recursive}`). `parser.CustomTagCalls` reads every call syntax; `resolve.CustomTagTemplate` finds the template, after `this.customTagPaths` (`parser.ApplicationSources`, the same walk as the mappings) and before the CFConfig's `customTagMappings` (`cfpath.ServerCustomTagRoots`) |
| `javaStubsPath` | Auto-synthesizes a `createObject("java", "X")` → `<javaStubsPath>.X` resolver |
| `formatting` | Formatter options |
| `linting.enabled` | Enable CFLint diagnostics |
| `linting.minSeverity` | Least severe CFLint level reported, on CFLint's own scale (`FATAL`…`COSMETIC`); unset reports everything. See `mapSeverity` and `meetsFloor` in [`lsp.md`](.ai/docs/guide/lsp.md) |
| `check` | What `clif check` runs (`checks`, default `["cflint"]`) and, per check, `maxFindings`, `baseline` and (cflint) `minSeverity`. Merged key by key (`mergeCheck`); `config.ResolveCheck` |
| `references.enabled` | Answer `textDocument/references` (off by default; see [`lsp.md`](.ai/docs/guide/lsp.md)) |
| `features` | Per-capability switches: `documentHighlight`, `watchedFiles`, `rangeFormatting` and `folding` default **on** (opt-outs, for when one misbehaves); `typeDefinition`, `structureDiagnostics`, `inlayHints` and `rename` default **off** (opt-in while new). See "`features`" in [`lsp.md`](.ai/docs/guide/lsp.md) |
| `completions` | `tagSnippets`, `functionSnippets`, `globalFunctionResolution` |
| `debug` | Verbose zap development logging to stderr. Without it `Debug` records are dropped before anything is formatted, and never reach the client as `window/logMessage` (`TestDebugRecordsNeedTheDebugFlag`) |

## Debugging why a call site resolved (or didn't)

`clif explain [--root <dir>] <file> <line> [call-substring]` prints, for every call
site on that line, the exact sequence of decisions `CanResolveCall`
(`internal/resolve/resolve.go`) walked through: which mechanism set the receiver's component
(function-scoped ref, file-level ref, `Application.cfc` ref, `<cfargument>` type, extends chain,
a `componentResolver` match on the variable name, a `componentResolver` match on the full line
text), which `FuncLookup`/componentResolver fallback fired for each hop of a chained call, and
why the final method-exists check passed or failed. `--root <dir>` picks which
`.clif.json` to load and which files to index — same semantics as `unresolved`'s directory
argument — and defaults to the target file's own directory if omitted, which matters because a
file's *own* nearest config can differ from the config a batch `unresolved` scan used.

The same trace is available from the editor: `clif.explainCall` (document URI,
0-based line, optional filter), offered as the "Explain call resolution on line N" code
action on any line holding a call. It answers with the running server's resolver and index
rather than a `--root` config, and shows the report as a message.

**Reach for this before manually tracing through
script_parser.go/tag_parser.go/result.go/resolve.go.** A component path that shows up in an
unresolved-call error but doesn't match anything literal in `.clif.json` or on disk is
almost always a `componentResolver` firing on a substring you didn't expect (see "Known resolver
false-positive" in [`resolvers.md`](.ai/docs/guide/resolvers.md)) — `explain` shows the exact resolver and match in one call instead of a
multi-file manual trace. Example: a `VARIABLES._content.createTemplate(...)` call reported "not
found in tassweb.packages.tass.directcontent" — a path absent from config entirely — because two
lines earlier, `VARIABLES._content = VARIABLES._document.getDirectContent()` had its RHS matched
by the generic catch-all resolver `{"match": "get$1()", "resolve":
"tassweb.packages.tass.${1:lower}", "prefix": "get"}` (intended for `getPageTools()`-style
factory methods): `indexFold` found the `"get"` prefix inside `"getDirectContent"`, matched the
whole call, and produced `tassweb.packages.tass.directcontent` — even though it's a genuine
iText/PdfWriter passthrough getter. `explain` surfaces this as a single `resolved "..." to "..."
via componentResolver matching the variable name` step.

## Release

`make release <version>` (`scripts/release.go`) validates, builds, tests, lints, scans for
vulnerabilities, updates `CHANGELOG.md` and `VERSION`, commits, tags, and pushes. Use `make
release-dry <version>` first. Pushing a `v*` tag triggers `.github/workflows/release.yml`, which
builds linux (amd64, arm64) and windows/amd64 on Ubuntu with the distribution's gcc
cross-compilers (`gcc-aarch64-linux-gnu`, `gcc-mingw-w64-x86-64`), and darwin (arm64, amd64) on
a macOS runner with clang, since CGO there needs the real Apple SDK; nothing uses zig any more.
It embeds the tag as `main.version`, and the built-in docs from the committed
`internal/docs/generated_docs.go` rather than a fresh `make docs`, so a tag always builds the
docs that were reviewed (a release fails if that file lacks the Lucee-only entries). The publish
job attests SLSA build provenance for every asset and `checksums.txt`.

**`make vuln` gates the release.** Every check runs *after* `CHANGELOG.md`/`VERSION` are
rewritten but before anything is committed, so a failure leaves those two files modified and
uncommitted — same as an existing test or lint failure. A newly published advisory can therefore
block a release without any code changing; fix it by bumping the Go patch version in `go.mod`
(stdlib findings) or the offending dependency, not by skipping the step.

The `vuln` job in `.github/workflows/ci.yml` runs the same `make vuln` target, so CI, the release
gate, and a local run cannot drift apart.

## Skills

- `/add-formatting-setting` — the config hops a new `formatting` key crosses, the defaults
  that must not move, and how to verify one against the corpus
- `/add-parser-test` — patterns and pitfalls for adding tests to `internal/parser/cfparser_test.go`
- `/run-clif` — build, smoke-test, and drive the binary (CLI subcommands + LSP stdio)
- `/parser-internals` — scanner tokenisation, the two parse loops, call-site extraction, and how
  the `unresolved` command works
