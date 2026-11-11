# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

Read `AGENTS.md` and run `bash scripts/dev-env.sh check` at the start of a
fresh environment. `bash scripts/dev-env.sh setup` provisions the pinned
Go toolchain, module cache, and linter; source the emitted `env.sh` afterward.
For sandbox proxy failures, follow the execution-context retry guidance in
`AGENTS.md` before diagnosing an outage.

```bash
make build          # generate docs + build target/release/clif with CFLint inside, as
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

**`make build` requires network access.** `build` depends on `generate` → `docs` →
`scripts/fetch-docs-cfdocs.sh`, which git-clones the cfdocs repo into the gitignored
`docs/data/`. The *generated* Go file (`internal/docs/generated_docs.go`) is committed, so plain
`go build ./cmd/clif`, `go test ./...`, and `make lint` all work offline
— use those when the fetch step can't run. (`make lint`'s first run needs the network once, to
fetch and build the pinned linter; after that it is served from the build cache.)

**The docs pipeline has two sources and both must be staged.** `internal/docs/generated_docs.go`
is generated from `docs/data/*.json`, which is *assembled* from per-source staging directories:

```
scripts/fetch-docs-cfdocs.sh → docs/src/cfdocs   (cache marker docs/.sha-cfdocs)
scripts/fetch-docs-lucee.sh  → docs/src/lucee    (cache marker docs/.etag)
scripts/assemble-docs.sh     → docs/data         (cfdocs first, lucee wins on collision)
```

`make docs` runs all three in order; `make docs-cfdocs` / `make docs-lucee` refresh one source
and re-assemble. Each fetch only ever touches its own staging directory and only replaces it
after a successful download, so refreshing one source can't destroy the other and a transient
outage can't destroy the cache.

**If a source can't be fetched, the generated file loses that source's entries.** A blocked
`docs.lucee.org` (some sandboxes and proxies deny it) means no `docs/src/lucee`, and
regeneration then drops every Lucee-only entry (`cfdistributedlock`, `cfstatic`,
`cfauthenticate`, …) — a pure-deletion diff of several hundred lines in
`internal/docs/generated_docs.go`. `assemble-docs.sh` warns loudly when a source is missing, and
`make docs` warns again when a fetch fails, but the build still proceeds. **Never commit that
deletion** — `git checkout -- internal/docs/generated_docs.go`. Only commit a change to that
file when `make docs` reported both sources staged.

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
| `internal/lex` | Byte classes and small string helpers several packages had each written for themselves (`IsIdent`, `IsWord`, `SkipSpace`, `LowerASCII`, `HasPrefixFold`, `ContainsFold`, `UTF16Len`); imports only `strings`, so the parser may use it. Only definitions that were byte-for-byte alike are here: a package whose rule differs keeps its own under a name that says how, and each helper's test holds it to every definition it replaced |
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
| `features` | Per-capability switches: `documentHighlight`, `watchedFiles`, `rangeFormatting` and `folding` default **on** (opt-outs, for when one misbehaves); `typeDefinition` defaults **off** (opt-in while new). See "`features`" in [`lsp.md`](.ai/docs/guide/lsp.md) |
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
