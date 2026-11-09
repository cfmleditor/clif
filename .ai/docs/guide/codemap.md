# Code map and the shared batch pipeline

_`internal/codemap`, and the batch pipeline `unresolved`, `explain`, `deps`, the MCP tools and the map share. Moved out of CLAUDE.md unchanged._

## Code map (`internal/codemap`)

**It is not `deps` run once per file, and the difference is node identity.**
`deps.Build` is a *seeded* BFS whose labels carry the call's line number
(`Base.cfc (line 42)`) so repeated refs stay distinct inside one tree. Run it per
file and the same method becomes a different node in every seed's graph, so tens of
thousands of definitions become hundreds of thousands of unjoinable nodes.
`codemap.Build` is the inverted shape: one streaming pass over every file emitting
canonically-identified nodes and deduplicated edges — structurally the pass
`unresolved` already makes, carrying edges instead of error records.

**Node identity is the file path, never the component dot-path.** Dot-paths are
many-to-one (mappings, per-directory resolution, expression substitution) and one
spelling can name different files from different base directories. A function is
`<relpath>::<lowercased name>`; dot-paths ride along as display labels.

**Unreachable code stays in the map.** Every declared function is a node whether or
not anything calls it. `Node.Reachable` says whether an entry point reaches it and
`Node.Island` groups it, so a renderer draws a detached subsystem as its own tree
rather than the map quietly describing a tidier codebase than the one on disk.
`Reachable(nil)` and `Detached()` are the two destructive views and neither is the
default.

**Two walks, two different edge sets, and swapping them breaks both** (each has a
test that fails if you do):
- **Islands** count `contains` (file→function). Without it every uncalled helper is
  a one-node island and the count becomes a count of uncalled functions — 103
  islands on `testdata`, against 18 with it.
- **Reachability** does *not*. Reaching a file must not reach the functions it
  declares, or every method of every live component is "reachable" and the map
  reports a codebase with no dead code at all. A `.cfm`'s top-level code needs no
  special case: `callerID` attributes a call outside any function to the file node,
  so the file node *is* that code.

**Cost.** `indexPass` reads every `.cfc` twice over (index + fingerprint) and the
scan is parallel and **streaming** — a ParseResult is turned into edges and dropped
before the next file is read, because a few thousand `ExtractCalls` trees held at
once is gigabytes. Roughly 10s for 11,769 files / 48,715 functions, 7s warm.

**`resolve.ResolveCallTarget`** is the one new resolver entry point: it runs exactly
what `CanResolveCall` runs and also reports where the call landed. `canResolveCall`
already computed the callee and threw it away. Rather than widen its return across
twenty-odd `return ""` sites — the hand-maintained parallel list this file warns
about elsewhere — each accept path calls `tr.hit(...)` beside its existing
`tr.addf(...)`, and **`TestEveryAcceptPathRecordsATarget` parses the source** and
fails on a `return ""` with no preceding hit. A forgotten hit costs a graph edge,
not a wrong one.

**The HTML viewer embeds its JavaScript.** `assets/vendor/d3.bundle.js` is a 76KB
esbuild bundle of the eight D3 modules the four views use, committed so `go build`
needs no Node and a report opens with no network. `make update-d3` rebuilds it from
`entry.js` — **adding a `d3.` call to the viewer means adding its module there**, or
it is undefined at runtime. `TestEmbeddedBundleCannotCloseTheScriptElement` fails if
a re-bundle ever introduces `</script`, which cannot be escaped inside inline JS.

**The wire format is compact, and it has to be.** 60,000 nodes as JSON objects with
full string ids was a 25MB page; interning every string, addressing edge endpoints
by node position, deriving a node's id when possible, and dropping `contains`
entirely makes it 4.4MB. `compact.go` and the viewer's `decode()` are two halves of
one format — `TestCompactRoundTripsEveryNode` and `TestCompactFlagsMatchTheViewer`
pin them together.

**The store's cache has a two-part key.** Content hash *and* a fingerprint of every
indexed `.cfc`, because resolving a call site reads the index built from every other
component: keying on content alone serves a stale edge set after an unrelated file
moves a method. Editing any `.cfc` therefore invalidates everything; editing `.cfm`
pages does not, unless the edit changes what the page `<cfinclude>`s. `PruneCache` keeps
one generation, since older ones can never match.

**`Options.InferArgs` makes the map accept what `unresolved` accepts.** `unresolved` types an
untyped argument from what every caller passes it (`Resolver.InferArgsFiles`); the map's
resolver never set that, so `thing.ping()` in `use( thing )` was an unresolved edge to an
external node where `unresolved` reported nothing. `clif graph` turns it on (`--no-infer-args`
opts out); `Build` sets it on each resolver `ConfigFor` hands out for the length of the build
and restores it, so the editor's `generateCodeMap`, which shares a live resolver, leaves it off.
A caller may be a page, so with it on the fingerprint covers every template's content and
editing a `.cfm` invalidates the cache, which it otherwise does not.
`TestTheMapTypesAnArgumentAsUnresolvedDoes`, `TestInferringArgumentsWidensTheFingerprintToPages`.

**`unresolved`, `explain`, `deps`, the MCP tools and the code map share one batch pipeline, and a
change goes into it, not into one command.** Each had its own copy and each had drifted: on Masa CMS
(no `.clif.json`) `unresolved` assumed the mura preset and added the startup beans' resolvers, and
the map did neither, so the map reported 21,962 unresolved calls to `unresolved`'s 3,786. The
pieces, each used by every command that resolves in bulk:
- `batchSettings` (`cmd/clif/batch_settings.go`) reduces a config, or the framework a project is
  assumed to be, to a `resolve.Settings`; `unresolved.Options.Settings`/`UseSettings` convert to and
  from it. A new setting goes into `Settings` and `batchSettings`.
- `resolve.NewBatchResolver` builds the resolver (snapshot scan, `vfs.CacheMetadata`, Lucee
  detection, `BeanResolvers`); `Resolver.IndexFiles`/`IndexForBatch` index in parallel with the
  resolver's own rules; `Resolver.BatchParseOptions` is what a scan parses calls with (setter,
  constructor and injected-bean lookups included).
- `Resolver.ClassifyCall` decides what a call is: accepted, builtin, global definition, missing
  base or unresolved. Two reasons are set aside here, not in `CanResolveCall`, so the resolver's
  own negative-control tests and `explain`'s trace keep the reason: a chain hop whose method
  declares no return type (`isUntypedHop`, accepted as dynamic, as `x = svc.get()` is) and a
  member-function name called on a receiver with no ref when no workspace component declares
  that name (`isUndeclaredMemberCall`, a builtin), and a receiver assigned the result of a
  function declaring no return type with none to infer (`assignedFromUntypedFunction`, accepted
  as dynamic). The map had its own copy that hid a call as a builtin when it had a chain
  (`f().len()`) and by a shorter list of names, which is why it stayed a few dozen to a hundred
  calls from `unresolved` after everything else matched. `resolve.IsBuiltin` is the one list.
- `explainEnv` (`cmd/clif/explain.go`) is the CLI's and the MCP tool's explain, built with the
  scan's own options (inference on, `--no-infer-args` to match a run that used it). Its trace is
  `canResolveCall` with a recorder, and `WriteExplanation` ends every call with a `report:` line,
  `ClassifyReason` applied to the reason just printed, so a built-in or a missing-base call says what
  the report does with it instead of a bare "unresolved". The editor's `clif.explainCall` prints
  the same line but traces with the session's live resolver, which does not infer arguments.
  `batchFileConfig` is the map's per-file environment, fallback and per-config alike.
**There is one way to parse a file of a workspace, and it is the resolver's.** `Resolver.ParseFile(file,
content, spec)` parses a file as the workspace types it: the resolvers, mappings and property rules the
resolver was built with (`PropertyResolvers`, `ServicePropertyResolvers`, `InterpolateAllText` are fields
of it, set from `Settings`) and the five lookups that type an injected dependency or a function's
return. The caller says only what it wants (`ParseSpec`: `Calls`, `AllScopes`, `Links`, `Shallow`,
`FindCalls`, a logger). Every parse of a workspace file goes through it: the unresolved scan and
`explain` (`BatchSpec`), the code map (`BatchSpec` plus links), the editor's live parse (calls only on
request, signatures only for an ordinary file's index entry), `refs`, `deps`, and the index a batch run
builds (`IndexSpec`). Two specs type a call by the configured resolvers alone (`ByResolversOnly`) and
that is deliberate: a scan wants `getService("X")` to be what getService returns, which is `$any`, and a
dependency graph or a reference search wants the component the resolver names; an index parse must not
read another file's declared return, which indexing in parallel would make depend on which worker got
there first (`IndexSpec`; it was the first thing the move changed, caught by `unresolved`'s output no
longer being byte-identical). What does not go through it is not a workspace file as the workspace types
it: `resolve`'s own reads of framework internals, `config`'s detection of a tool, `cfstubgen`, and the
editor's folding and docblock reads of text. The index has `Index.IndexParsed` for a parse the caller made.

**One resolver, built in one place.** `resolve.NewResolver` builds every resolver from a `Settings` (field
mapping, Lucee detection); `NewBatchResolver` adds snapshot mode and `CacheMetadata`; the editor adds its
discovery and `IndexTracksFiles`. `Resolver.Settings()` reads a resolver back for a tool that builds
another run from it (the editor's export of the unresolved report, which also infers arguments as the CLI
does). The code map's `FileConfig` is the resolver and its routes and nothing else: the six fields it
copied from the resolver are gone, and so is `refs`' own bean lookup and its parse.

**A `.clif.json` is read one way: `config.ResolveFile`.** The command line (`daemon.Config`), the daemon
(`daemon.SettingsFrom`) and the standalone editor (`configureSession`) all resolve a config file through it,
which is `config.Resolve` plus the one thing only a file has: the folder holding it maps under its own
name (`ImpliedMapping`). `daemon.Config` is `Resolved()` and one-line delegates; `server.Settings` and
`Server` both embed `config.Resolved`, so a config key is added once, to `Resolved`, and reaches the CLI,
every daemon session and the standalone editor. A project with no config is `config.Assumed` (the framework
its source is, as a `Resolved`), and `resolve.SettingsFromConfig` turns any `Resolved` into the resolver's
settings: the batch commands and the editor's `getResolver` both call it, and `unresolved.Options` embeds
`resolve.Settings` instead of repeating its fields. Things that moved with it, each with a test that
fails without it: an absent `formatting` block resolves to the documented defaults, not the zero value that
left the whitespace-only guard off (`TestResolve_FormattingAbsentIsTheDocumentedDefaults`); the standalone
editor applies the implied mapping (`TestTheEditorMapsAConfigsOwnFolderAsTheCommandLineDoes`) and takes
`workspacePaths` and `workspaceIndexGlobs`, which it ignored, from the config
(`TestTheEditorHonoursWorkspacePathsAsADaemonSessionDoes`). The server's `restoreUnmentionedBlocks` decides
from the JSON (`merged.Formatting == nil`), not from `Resolved`, which is why the formatting default is safe.

**A dependency graph is `deps.BuildDocument`, for both `clif deps` and `clif.exportDeps`.** They differ in
how a file is parsed and in nothing else: the document's calls (`deps.DocumentCalls`), the refs fallback
and the memoising walk (`deps.CallLoader`) are shared, and both parse with `resolve.DepsSpec()`, which types
a call by the configured resolvers and not by what the called function declares it returns. The editor
used the scan's parse and drew `$any` for `getService("UserService")` where the command line drew the
component, while a comment in `deps.go` said they matched
(`TestExportDepsDrawsTheComponentTheResolverNames`). Bean maps load through `Resolver.LoadBeans`, which the
server's lazy load had written out again; the server's assumed-framework decision is `assumedConfig`,
`config.Assumed` read through the discovery filesystem. Two of the three private resolvers in `resolve` and
`server` (the discovery ones) are minimal on purpose, with a private index; the third, the caller-context
view, is a full copy, and `TestACallerContextViewCarriesEveryField` fails when a field is added to
`Resolver` and the view forgets it.

**Walking a workspace is `cfpath.CollectCFML`**, with the skip rule in `cfpath.SkipScanDir` (below): the CLI,
the code map, the editor's scans and `refs` each had a walker, two with their own copy of the rule and one
with none. Component indexing is one door, `Index.IndexParsed` over `Resolver.ParseFile` with `IndexSpec`
(a batch run, the lazy `EnsureIndexed` and the editor's re-index on save); `Index.IndexFile` is the bare
parse with no workspace rules, for the index package's own fixtures.

**`refs` is the same question everywhere.** `refs.Options` takes the workspace's `Resolver`: a call's
component must define the function, and with a `SourceFile` be the one searched for, and `Explain` gives an
unresolved call its reason from `CanResolveCall`. The editor's find-references, `clif refs` and the MCP
`find_references` each wired their own closures, or none, and answered differently; on ColdBox the new
CLI drops 2 of 113 hits for `getInstance` and 3 of 78 for `announce`, calls on a chain's first receiver
(`controller.getInterceptorService().announce()`) that the old code credited to the wrong component. A
component the resolver cannot locate defines nothing, so its calls are dropped too, as the editor always
did. `TestRefsVerifiesAHitAgainstItsComponent`.
**`unresolved` reads each file under its own nearest `.clif.json`, as the map does**
(`unresolved.Options.ResolverFor`, wired by `perFileResolver` over the map's `configSet`; indexing
is grouped by governing resolver too, all sharing one index). `--one-config` restores the single
config above the first path. Speculative scans ignore the hook.
`TestUnresolvedReadsEachFileUnderItsOwnConfig`.
After the change the map's unresolved count equals `unresolved`'s exactly on Masa (3,786 with
inference, 5,417 without), ColdBox (3,974, 3,981) and TestBox (929, 976), and `unresolved`'s own
output is byte-identical to before. Differences that remain are on purpose: `deps` parses with its
own light options (a scan types `getService("X")` by what getService returns, which is `$any`
where the resolver names the component, and a dependency graph wants the component); `unresolved`
reports a missing base once per file and the map counts the calls; and the editor server's
resolver is a live per-session one, not a batch one.
`TestGraphAssumesTheFrameworkUnresolvedAssumes` (cmd), `TestTheMapResolvesWhatUnresolvedResolves`
and `TestClassifyCallIsTheOneDecision`. The second needs the index, the parse and the bean lookup
all reverted before it fails, because any one types the fixture's property.

**`indexPass` records every template's includes before the scan starts.** A bare call
in a page resolves through the include graph (a page calling a function its includer
declares, as Lucee's admin pages do), and it recorded only components' includes, so a
template's reached the index only when something indexed it mid-scan and the edge came
and went with goroutine order: Lucee's map differed by a dozen edges between runs of one
build. The include paths are in the fingerprint for the same reason.
`TestAPageReachesItsIncludersFunctionsInAnyOrder`.

**`Store.Path` is a Go BFS, not a recursive CTE.** A CTE can express the walk but not
a visited set shared across branches — SQLite can only stop a trail revisiting its
own nodes — so a search from a high-fanout node with no answer to find is
exponential in the depth limit, and "is there any path" is asked most often about
pairs that have none.

**Framework routes (`internal/route`) are how routed code stops looking dead.** A
dispatcher's target is named nowhere in the source, so every routed controller
method reads as uncalled. The convention lives in the `routes` config block, not
in Go — `TestFw1Convention` resolves FW/1 with the same machinery as TASS, and is
there to stop the grammar quietly becoming one framework's rules.

Three things carry their reasons, each with a test:
- **The method must exist** before a controller rule claims a route. A component
  template built from `${1}` matches every route starting with that segment, so
  without the check the first rule shadows all the ones below it.
- **Alias keys are sorted, longest first.** Ranging the map directly made the
  winner depend on Go's randomised map order, so the same route resolved
  differently between runs of one build.
- **An alias may have several replacements and all of them are returned.**
  `ui.web` means the product serving the page, which is a runtime fact; the map
  marks such an edge `Dynamic` and go-to-definition returns several locations.

Routes are written four ways and the scanner handles all of them: HTML attributes,
`?do=` query parameters (unquoted, so they end at a URL delimiter, `&amp;` included),
JavaScript object keys quoted or bare, and function arguments (cut at the first
delimiter, since `redirect('a.b.c&x=1')` names `a.b.c`). Two boundary rules earn
their tests: `isNamePart` counts `:` as part of a name, which is right for
`xlink:href` and exactly wrong for a JS key where the colon *ends* the name, so
properties use `isPropNamePart`; and of two overlapping matches the **shorter**
wins, or every `href` swallows the `?do=` inside it.

`${N+:search}` enumerates method start points longest-first, because a route does
not say where the controller's name stops and the method's begins. It replaced one
hand-written rule per start point and is safe only because each candidate is still
checked against the component's real methods.

`clif routes --unresolved` groups what did not resolve by leading
segments. On tassweb: 82% of 2,741 occurrences resolve, and 257 of the 292
unresolved distinct routes share their first two segments with routes that *do* —
so the controller is found and the method name is what the route does not spell.
`dialog`-segment routes are 46% of the remainder.

`longestDir` exists because the view split cannot be templated: the file name
carries dots of its own, so the split between directory and file depends on what
is on disk. Measured on a real workspace: 79% of 1,022 route occurrences resolve.

**Each file is resolved under its own `.clif.json` (`Options.ConfigFor`).**
A workspace of several applications has one config each, and every one lists the
others in `workspacePaths`, so a scan rooted anywhere reads all of them. Under a
single config the other applications' `componentResolvers` never fire — it does not
error, it just resolves nothing. `cmd/clif/graph_config.go` memoises a
`resolve.Resolver` per config file, **all sharing one `index.Index`** (signatures
are a property of the workspace, not of whose resolvers you read them under), and
mixes every config's content hash into the cache fingerprint via `ConfigExtra`,
since the resolver chain decides what an edge points at. `--one-config` opts out.

**`Options.EntryGlobs` marks code a runner invokes by a constructed name.** No
static analysis can see `createObject("component", "prs" & version)`, so those
files look unreferenced and are not. On one workspace `../prs` alone was 10,639 of
11,374 apparently-unreferenced functions — 2,600 release scripts, correctly
unreferenced and entirely wrong to read as dead code. A bare directory name matches
everything beneath it, because `path.Match` has no `**`. Private methods are never
marked.

**A file node is a caller, not just a container.** Top-level `.cfm` code has no
enclosing function, so `callerID` attributes it to the file — and that is the
larger half of the graph, not an edge case: 74,226 of 107,683 call sites on a real
workspace come from file-level callers. `TestTopLevelPageCodeIsACaller` pins it.

**`--under` keeps boundary nodes, and the strict version is the trap.**
`FilterUnder` keeps a node outside the prefix when an edge crosses into it
(`Node.Boundary`). `Filter` — keeping only edges with *both* ends inside — drops
every caller from elsewhere, which is what you scoped the map down to find: on
`packages/tass/core` that was 472 edges against 7,999. A scoped map with no callers
reads as a broken tool rather than a narrow view.

**The builtin check runs *after* resolution, not before.** About sixty names in one
real workspace (`init`, `add`, `close`, `get`, `isValid`, the `onXxx` handlers) are
both declared functions and something `isBuiltin` recognises. Testing the name
first discarded every bare call to them before anything looked for a definition.

**`cfpath.SkipScanDir` skips every dot-directory**, not a list of named ones. The list
was `.git`/`.svn`/`node_modules`/`target`/`vendor`, and on a real workspace it let in
`.claude/worktrees` — git worktrees holding a complete second copy of the codebase,
which doubled every count and added thousands of phantom entries to the unreferenced
list.

**The same workspace gives the same map, and a map is never the order to decide
with.** A bare component name no path answers is looked up by file name
(`Index.FindFilesByBasename`), and ContentBox keeps a `RailoDBInfo.cfc` in several
patch directories, all equally near a patch that has none. The index lists them out of
a map, filled by a parallel scan, and the resolver took the first, so an edge — and the
island numbering after it — moved between runs of one build. The list is sorted now,
and `componentPathUncached` breaks a tie by the lowest path, as `LookupPreferred` does;
`expressionMappings` apply longest key first (`parser.ExpressionMappingOrder`) for the
same reason. `TestTheSameWorkspaceGivesTheSameMap` builds one workspace ten times.
Check a change here by building a project's map several times and comparing the files
byte for byte (only `buildMillis` differs).

**The rules that compare a name's declaration count with a threshold are decided after the scan**
(`ClassifyOptions.DeferCounts`, `resolve.Pending`, `internal/resolve/classify_pending.go`). The count
includes the framework stubs the scan has reached, a stub is indexed the first time something
resolves to one, and in a parallel scan that is whenever a worker gets there; classified at once,
a call decided by `sharedMember` (five files), `distinctiveMember` (exactly one), `undeclaredMember`
or `knownNativeMember` (none) moved between two rules that both accept it, or between "built-in"
and "unresolved" in the map's stats, by one or two a run (ContentBox 855/856/857; main showed 854
about one run in 30). Each of those four rules has a `guard` (everything but the count); a call
in the "variable has no component ref" family that no guard passes is decided at once, as is one
a rule that reads no count decides before any count rule could apply. Otherwise `classifyPending`
keeps the call's `Variable`, `FuncName` and `Chain` and the chain of rules that could still decide
it, and `Pending.Settle` asks them in order of the index as it stands once the run is done:
`unresolved` settles after its workers finish (`settlePending`), the map after its scan, before a
file's graph is merged or stored in a `Cache` (a stored graph has the pending calls counted). 30
runs on ContentBox give one map and 25 one rule table. It is not applied with `GlobalDefs`, whose
check counts too, nor in `explain` or the editor, which classify one call against the live index.
Counting against the finished index is the intended meaning, and it moved some calls from where
the early count had put them: one Lucee finding (`file.toPath` in `WatchService.cfc` is now
accepted, by one of the count rules), nine calls on Masa and one on ContentBox from "built-in" to
"unresolved" in the map's stats; the other four corpora are unchanged. Two cheaper fixes were tried
and rejected because they moved the findings further: not counting the stubs at all (testbox -1,
coldbox-platform +5, ContentBox -16/+2), and counting every embedded stub (`frameworkapi`, Masa
+16/-17, coldbox-platform +18/-3, ContentBox +14/-4: most stubs are for frameworks the project
does not use). `unresolved --verbose` prints the per-rule counts.

