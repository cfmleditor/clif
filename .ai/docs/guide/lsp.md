# LSP surface

_Every LSP capability and workspace command the server answers, and the VS Code extension's capabilities it does not. Moved out of CLAUDE.md unchanged._

Declared in `Server.capabilities()` (`internal/server/server.go`):

- Incremental text sync, completion (trigger chars `<`, `/`, `.`, `>`), definition, type definition, hover,
  signature help (`(`, `,`), document + workspace symbols, document links (with resolve), code
  actions, document formatting, range formatting, on-type formatting (`>`), document highlight,
  folding ranges, workspace folders.
- `textDocument/typeDefinition` (`internal/server/typedefinition.go`) goes to the component
  the symbol under the cursor holds: a variable, argument or property's component, or the
  component a called function returns. **Opt-in** (`features.typeDefinition`, default off,
  `config.typeDefinitionDefault`) while it is new. It is the extension's `CFMLTypeDefinitionProvider`,
  and the extension's source is the spec, since it has no tests for it.

  **What a variable holds comes from `resolve.ComponentOf`, which is `CanResolveCall`'s own
  receiver lookup** — `receiverComponent`, extracted from `canResolveCall` rather than
  copied, so go-to-type-definition and `unresolved` cannot disagree about a receiver. The
  extraction was checked by running `unresolved` over the testdata and the six-project
  corpus before and after: 91,405 entries, identical. `ComponentOf` tries a
  `componentResolver` against the variable name but not against the whole line, since that
  fallback exists for chained calls and would answer for whatever else the line holds, and
  reports `$any` and `$builtin.` as unknown. A return type counts only when dotted: a bare
  word is far more often `struct` than a component in the same directory.
- `textDocument/inlayHint` (`internal/server/inlay_hint.go`) shows a component's own dot path beside
  its declaration, `Path: (models.User)`, from `parser.ComponentDeclarationEnd` and the path
  `clif.copyPackage` answers. **Opt-in** (`features.inlayHints`, default off; advertised only when
  on). An Application.cfc and anything that is not a .cfc has none.
- `clif.searchDocs` (query, "tag" | "function" | "all", limit) and `clif.docEntry` (name, kind)
  answer the compiled-in CFML docs, so an editor can offer a documentation browser without carrying
  a copy of them (`internal/server/docs_command.go`). A name match ranks above a description match.
- **Rename** (`internal/server/rename.go`, `parser.PlanLocalRename`; `features.rename`, opt-in): a
  local variable of a script function, in its document. The plan is every place the function
  writes the variable, found by walking the body's tokens with a bracket stack, so a struct key
  (`{ x = 1 }`), a named argument (`f( x = 1 )`) and another object's member (`o.x`) are not the
  variable, while `local.x` and a `#x#` in a string are. Closures are scopes of their own, searched
  innermost first (`closuresAround`), so a variable a callback declares is renamed in the callback.
  It **refuses**, with a reason, where it cannot show it has every use: the name as a string
  (`structKeyExists( local, "x" )`, `"local.x"`), `evaluate()`, a nested function or arrow function
  that takes or declares the name, a `catch` that binds it, CF tags in the body, an argument (a
  caller passes it by name), or a new name the function already writes without a qualifier
  (`LocalRename.Taken`: every unqualified name in the whole function, closures and parameters' defaults
  included, so a variable it reads from outside or a function it calls counts, and a closure's
  variable is checked against the function around it). A request whose cursor is not on one of the
  plan's ranges, a struct key or `o.x` spelled like the variable, is refused too. Measured over 54,000 `var`
  declarations in real code: 44,000 planned with no range that is not the name, and applying one
  rename per function to four corpora left every file parsing as before. The refusals are mostly
  `var` in tag functions (7,700), names in strings (1,700) and closures that shadow.
  Function rename and cross-file rename are not here: the references engine cannot see dynamic
  calls, so a rename over it could leave a caller behind without saying so.
- **Query columns** complete after `name.` when the document declares the query
  (`parser.QueryColumns`, `queryColumnItems` in `completion.go`): a `<cfquery name>` whose SELECT list
  names them, `queryNew( "a,b" )`, or `queryExecute( "SELECT a, b …" )` as a string literal, the last
  declaration before the cursor. Columns come first, then `recordCount`, `currentRow` and
  `columnList`. A column the list names by `*`, or an expression with no alias, is not guessed:
  `SELECT *` offers nothing, and `count(*)` with no alias is left out of a list that is otherwise
  known. Also offered inside `#users.` in a `<cfoutput>`. It replaced no existing completion; the
  generic member-function list is what it falls back to.
- **Application lifecycle code actions** (`internal/server/lifecycle_actions.go`): in an
  Application.cfc, one refactor action for each application event the component does not declare
  (`onSessionStart`, `onRequestStart`, …) and one for all of them, written as the component is
  written (script or tags) and inserted before the body's closing `}` or `</cfcomponent>`
  (`parser.ComponentBodyEnd`). What is declared is read from the parse, so case does not matter.
- **Structure diagnostics** (`internal/server/structure_diagnostics.go`, `parser.TagProblems`) publish
  what is wrong with a document's CF tags as it is opened and edited, under the source `clif`:
  unclosed tags, closing tags with no opener, tags outside the tag they belong to (`<cfcatch>` in
  `<cftry>`), a missing `name`/`template`/`url`, a repeated attribute (warning) and a repeated
  function argument. **Opt-in** (`features.structureDiagnostics`, default off). The checker walks
  the page's tag regions with the folding pass's own tag walk (comments, quoted values, `<` that
  starts no tag), and holds only tags that always take a body to a closing tag: a tag that can
  stand alone is never reported, so a page that runs reports nothing. Measured over 10,407 files of
  real code, every finding was a real fault. Two shapes were false positives before they were
  handled and have tests: a CF tag written inside an HTML attribute value
  (`class="a<cfif b>c</cfif>"`), and a `"<!--"` string in a `#...#` expression (markup's comment
  never follows a quote). `CLIF_TAGPROBLEMS_CORPUS=<dirs> go test -run TestTagProblemsCorpus -v
  ./internal/parser/` lists what it finds in a corpus; read each by hand. `TestTagProblems` holds
  the rules, and the mutation checks in its history show each one fails when removed.
- `textDocument/documentHighlight` (`internal/server/documenthighlight.go`) shades the other
  occurrences of the identifier under the cursor. Deliberately a *textual* answer, reported as
  `DocumentHighlightKindText`: matching is whole-identifier and case-folded through the same
  `identSpan` the references handler uses. It never leaves the open buffer, which is what
  separates it from `textDocument/references` and why it is on by default.

  The protocol's Read/Write distinction is deliberately not attempted. CFML spells too many
  things with `=` — an assignment, a named argument (`f( name = 1 )`), a tag attribute
  (`<cffunction name="x">`) — so a cheap "is the next token `=`" rule would mark the attribute
  *name* in every tag as a write. A wrong Write badge is worse than none.
- `textDocument/foldingRange` (`internal/server/folding.go`) folds from the parser. Functions
  come from the document's cached ParseResult (`pr.Scopes`), so the handler takes `lockDoc`,
  which pays any reparse didChange deferred — `TestFoldingFollowsEdits` fails if it takes
  `lockDocOnly`. Everything else comes from `parser.StructureSpans`, one pass per request over
  the text split by `ClassifyRegions`: in a tag region it finds comments, and in a script region
  it runs the parse's own scanner in CFScript mode (a `#...#` inside a string can hold strings of
  its own, and a plain quote-to-quote scan pairs them wrongly) with a stack of open `{`, `(` and
  `[`, reporting comments, blocks, multi-line argument and parameter lists and array literals,
  and switch cases. The markup around the script regions is walked once by `tagStructure` (a
  stack of open tags; a close pops to its opener and discards what is left open above it), so a
  `<cfscript>` pairs with its `</cfscript>` across the region between them. A fold ends on the
  line before the one holding its closer, so the `}`, `)` or `</cfif>` stays on screen; a
  comment folds to its last line, and a `<cfelse>` branch or switch case to the line before its
  last content. `TestTagStructureSpans` has a case per tag rule.

  The rules that make the bracket pass agree with the tree-sitter folding it replaced, each with
  a case in `TestStructureSpans` that fails without it:
  - **A fold starts on its statement's first line**, not the bracket's, so a wrapped `if` folds
    from the `if`; a bracket further down folds from its own line too. A statement spanning lines
    folds from its first line to its last bracket even when that bracket is on one line, which is
    how a chain written a call per line gets its fold.
  - **`if`/`else` and `try`/`catch` chains fold whole** as well as branch by branch, and an
    `else` that is not the last folds to the end of the chain. Two folds on one line are one arrow
    in VS Code, and with folds sorted outermost first the chain is the one shown.
  - **A parenthesis after `if`, `while`, an operator or `=` groups; one after a name holds
    arguments.** Only the second folds. A `[` after a name is an index and does not.
  - **A newline ends a statement** in a block when the last token could end an expression and the
    next is a word that is not an operator: CFScript does not require the semicolon, and cfwheels
    omits it throughout. Not after a lone word (`admin` then its attributes), and not inside an
    **attribute statement** (`component` or `property` with attributes over lines, a
    script-syntax tag `word word=`), where a new line's word is another attribute if `=`, a `{`
    or the next line follows it and a new statement if it is a keyword or anything else follows.
    `default="x"` is the commonest attribute and `default` a keyword, so `=` is checked first.
  - A word is classified once, by `classify`: one switch on the word lowercased without
    allocating. Lists compared with `EqualFold` were a fifth of the scan.

  Over the six-project corpus it reproduces 93.9% of tree-sitter's 156,241 folds, and makes about
  2,000 that tree-sitter did not. The rest is comments where tree-sitter's node started a line
  early (about 3,100), multi-line binary expressions and conditions left out on purpose, method
  chains and concatenations inside declarations, and pages where tree-sitter closes `<td>` or
  `<li>` implicitly. About 80µs for a 60-function component and 116µs for a 490-line page,
  against 3.5ms for tree-sitter on the component.

  **tree-sitter allocates through libc, not Go.** go-tree-sitter's `init` installs allocator
  hooks that call back into Go for every `malloc` and `free` a parse makes;
  `internal/language/alloc.go` resets them to tree-sitter's defaults. That was 24% of a
  tree-sitter folding request, and it applies to every tree-sitter parse — the formatter's
  now. `TestTreeSitterAllocatesThroughLibc` fails if the reset is lost.
- `textDocument/rangeFormatting` (`internal/server/range_formatting.go`) formats the **whole**
  document and returns only the edits inside the requested lines. Formatting the selected text
  alone is the obvious approach and wrong twice over: a selection rarely parses standalone, and
  its indentation depends on everything enclosing it. Going through the whole document makes the
  result identical line-for-line to format-on-save, so the two commands cannot disagree.

  Lines are matched on their **trimmed** text (`internal/textdiff`, a Myers line diff). Matching
  raw lines collapses a reindented block into one hunk with nothing to anchor on; trimmed, a
  reindented line still matches itself and the diff sees only the lines the formatter genuinely
  added or removed.

  An edit is kept only when it lies **entirely** within the selection. Overlap is not enough: a
  formatter change is not always divisible — joining a five-line `<cfif …>` header onto one line
  is a single hunk covering all five — and keeping it because it reaches into the selection
  rewrites lines the user did not select. The overlap rule did exactly that in 1,745 of the
  5,509 corpus files that format. The guarantee is therefore: every change fitting inside the
  selection is applied, and one straddling its edge waits for a wider selection or a document
  format.
- `workspace/didChangeWatchedFiles` (`internal/server/watchedfiles.go`) keeps the index current
  when files change outside the editor. There is no static way to ask for this — the protocol
  offers only dynamic registration — so `initialized` sends `client/registerCapability` for
  `**/*.cfc` and `**/*.cfm` when the client advertised
  `workspace.didChangeWatchedFiles.dynamicRegistration`, and logs once when it did not. Nothing
  is needed from the extension. Before this the index was a startup snapshot: `indexWorkspace`
  ran once and only `didOpen`/`didChange`/`didSave` updated it afterwards, so a checkout or a
  second editor left the server resolving to components that no longer existed, with
  `clif.reindex` the only cure — and in daemon mode one stale snapshot served every
  client and outlived the editor.

  Three rules decide what an event does, and each has a test that fails without it. An **open
  document's buffer outranks disk** for content, since the event is either the editor's own
  save or a change the editor will reload and report itself. A **deletion is honoured whether
  or not the file is open**: content is the editor's to report, existence is not, and an entry
  pointing at a path that is gone cannot recover, while re-saving a buffer re-indexes it. A
  **read that fails is treated as a deletion**, so a file caught mid-write or created and
  removed inside one batch does not leave its previous contents indexed as current.
- `textDocument/references` (`internal/server/references.go`) is **opt-in**: off unless
  `"references": {"enabled": true}`, and `capabilities()` advertises `referencesProvider` only
  when the flag is on, so a client that has not opted in never offers the command. It is gated
  because one request walks and parses every CFML file under `searchRoots()` — the same scan
  `clif.findRefs` and the `refs` CLI do — with no incremental call-site index to answer
  from. A dot-path under the cursor searches `refs.Options.Component`; anything else is a
  function name and searches `refs.Options.FuncName`. The search is scoped by the file that
  *declares* the function (`declarationOf`, which follows go-to-definition's order of
  preference), not by the requesting document, so it works from a call site. `refs.Entry` has a
  line and no column, so `entryRange` recovers the column by finding the identifier on the line;
  component entries whose line does not name the component are dropped rather than reported as a
  whole-line match, because the parser also records the variables a component ref flows into
  (`report = myCtrl.getReport()` is a ref to myCtrl's component on a line that never names it).
- **`features`** (`config.Features`/`ResolvedFeatures`) switches off individual capabilities.
  Every switch but `typeDefinition` defaults **on**, as an opt-out. **`folding`** (`config.foldingDefault`) was opt-in
  while it was expensive, and is on since the span pass covers 93.9% of what the tree-sitter
  version folded and folds the corpus at 273 MB/s (`.ai/docs/FOLDING-PLAN.md` §5).
  **`typeDefinition` defaults off** (`config.typeDefinitionDefault`), as the newest capability. The fields are `*bool` for the reason the `completions` block documents — a
  defaults-true flag as a plain bool cannot tell "turned off" from "not mentioned", so naming one
  key would switch off its siblings. `mergeFeatures` unions key by key for the same reason
  `mergeFormatting` does. `featureDefaults` in `features_chain_test.go` states every default and
  fails if a new switch is added without one.

  Two traps this shape sets, both with a test:
  - **The zero value is every switch off.** `NewServer` therefore seeds `config.ResolveFeatures(nil)`,
    exactly as it does for `completions`; a session that never reaches `applyConfig` would
    otherwise advertise none of them and look like a build without the features.
  - **A gate in the handler is one a direct call slips past.** The watched-files gate lives in
    `applyWatchedFileChanges`, not in its handler, because the first test of it called that
    function directly and passed with the switch ignored.

  Disabling a feature **un-advertises** its capability rather than declining the request, so the
  editor falls back to its own behaviour. `rangeFormatting` is a second gate *under*
  `formatting.enabled`, not an alternative to it — it is the only one of the four that writes to
  the buffer, and it shares all its machinery with format-on-save.
- **Columns are UTF-16 at the edge and bytes inside.** An LSP column counts
  UTF-16 code units, and every parser helper reads a line in bytes, so a
  handler converts once where a column arrives (`byteCol`) and once where one
  leaves (`lineCol`, or a `colMapper` for a response carrying many ranges) —
  never in between (`internal/server/position.go`). Casting
  `params.Position.Character` to an int reads the line at the wrong byte on
  any line with an `é` or an emoji before the cursor;
  `TestNoHandlerReadsAClientColumnRaw` fails on that cast. `didChange` is
  the exception by design: it hands its ranges to `parser.ApplyEdit`
  unconverted, because the parser converts them against text only it holds.
- **A large didChange batch is applied in one pass.** More than 50 changes in one
  notification (the "rapid" path) go through `parser.ApplyEdits`, which streams edits in
  document order instead of walking and copying the whole document for each. Reverting a
  reformat in Zed sends exactly that, an edit per line: 6,000 against a 110 KB file took
  435ms under the document's lock one at a time and takes about 5ms streamed. An edit out of
  order falls back to `ApplyEdit`, and `TestApplyEditsMatchesApplyEdit` holds the two to the
  same result on random input, UTF-16 columns and past-the-end positions included.
- **An edit outside a function defers its reparse; `lockDoc` pays it.** `ApplyEditResult`
  reparses every signature in the file for an edit outside a function body, and didChange
  runs on the read goroutine: 95ms per keystroke at component level in tassweb's
  `kiosk.cfc`. didChange now applies the text, marks the document with `scheduleReparse`,
  and returns (0.35 to 3ms there). `lockDoc` runs the owed reparse (`flushReparse`) before
  handing over the lock, so no handler, timer or background job ever reads a lagging
  ParseResult; the first one after a run of typing pays once. didChange itself takes
  `lockDocOnly`, as does didClose. While a reparse is owed, later edits are applied as text
  only, since their positions no longer match the parse's scopes. A timer (`reparseDelay`,
  200ms) catches up a document nobody asks about, which bounds how stale the index's view
  of it can be for requests on *other* files. The rapid-change path uses the same
  mechanism. Code that reads `s.parseResults` must hold the lock from `lockDoc`, which the
  entry points already do; `TestDeferredReparseIsCaughtUpByTheNextHandler` and its
  neighbours pin it.
- **Formatting answers with the changed runs of lines, not the document.** `lineEdits`
  diffs the source against the formatted text (a patience diff on trimmed lines, so a
  reindented line still anchors, with adjacent changed lines merged into one edit) and
  `TestLineEditsReproduceTheTarget` holds applying them to exactly the formatted text. It
  replaced a single whole-document TextEdit, which the editor diffed again itself and, on a
  65,000-line file, was 3.6MB. `internal/textdiff` (range formatting's Myers) is not used
  here: it keeps a frontier per edit distance, quadratic memory in the changed lines, and a
  first reformat changes most of them.
- **Slow handlers release the read loop.** Handlers run inline on the read goroutine, so
  one slow request held every message behind it. Formatting, range formatting,
  `explainCall`, `exportDeps` and `findRefs` call `releaseReadLoop` once they have the
  document text and settings, and do the rest concurrently. Only handlers that work from
  that captured text may: the cached ParseResult is edited in place by didChange, so a
  handler that reads it must stay inline. `TestFormattingDoesNotHoldTheReadLoop` holds the
  formatter on a channel and requires a didChange and a request sent meanwhile to be
  handled.
- **Completion hands its cached items over, it does not copy them.** The
  `inHashExpr` branch and the default branch both return
  `completionFromCache`'s slice directly when there is nothing to merge with it,
  so the response can be the process-wide `getBuiltinFuncItems` list (a
  `sync.Once` shared by every session in the process) or a document's cache
  entry. **Nothing downstream may write to `items`, and nothing may append to it
  after that point** — an append would write past the length into an array a
  concurrent request is appending to as well. `applySnippetPolicy` therefore
  copies before editing; it used to edit in place, which permanently rewrote the
  shared list for every session the first time a config turned snippets off.
  `TestCompletionDoesNotWriteToTheSharedBuiltinList` pins this, and fails if the
  in-place edit comes back. Removing the copy took a completion request from
  383KB to 3.4KB, roughly half the bytes of the whole round trip including its
  JSON marshalling.
- **Documentation and detail are resolved lazily when the client can.** A
  client listing them in `completionItem.resolveSupport.properties` is sent the
  built-in and member-function items without them (`s.builtinFuncItems()` and
  `s.memberFuncItems()`, one shape per combination, built once per process), and
  `completionItem/resolve` looks the entry up again: a built-in by its label,
  recognised by its `SortBuiltinFuncs` sort text, and a member by the entry name
  in its `data`, because member names repeat — `len` is `arrayLen`'s,
  `stringLen`'s and `structLen`'s. A client that lists neither is sent the full
  items and is not told `resolveProvider`. The deferred shapes are shared like
  the full lists and the same rule holds: nothing may write to them. Reach for
  `s.builtinFuncItems()`, not `getBuiltinFuncItems()`, anywhere a list goes to
  a client. `TestResolvedItemsMatchTheFullOnes` resolves every deferred item and
  compares it with the full one.
- Diagnostics come from CFLint when `"linting": {"enabled": true}`. `ensureBinary` takes
  the file `CLIF_CFLINT` names first (an error if missing), then a `cflint` on PATH, then the CFLint a release build embeds (`embedded.go`, behind
  `-tags cflint_embed`; the release workflow writes `internal/cflint/embedded/cflint.archive`
  with `scripts/fetch-cflint.sh`, which pins the version and checksums), unpacked once into
  the cache under `pinnedVersion`, then a download of that same version from
  `cfmleditor/CFLint` releases. A plain `go build` embeds nothing (`embed_off.go`).
  `CLIF_CFLINT_DIR` replaces the cache directory, for machines that will not run programs
  from it. `pinnedVersion` equals the script's pin (`TestPinnedVersionIsTheEmbeddedVersion`).

  **A download is the pinned version, checked, or nothing.** It used to follow
  `releases/latest` and run whatever came back, so whoever could publish a CFLint release
  could run code on every source-built clif with linting on. `executableSums` holds the
  sha256 of each platform's executable (the raw asset's digest, which is also what each
  archive holds), `installAsset` hashes what it writes and refuses a mismatch before the
  rename, and a cached file without the sum is downloaded again rather than run, since an
  older clif filled the same directory unchecked. `TestFetchAssetRefusesTheWrongChecksum`
  and `TestTheDownloadIsThePinnedVersionChecked`.

  **`mapSeverity` may return only Error or Warning.** CFLint's seven levels
  (`com.cflint.Levels`: FATAL, CRITICAL, ERROR, WARNING, CAUTION, INFO, COSMETIC,
  plus UNKNOWN) fold onto the LSP's four, and an editor shows neither Hint nor
  Information by default — VS Code draws a Hint as a faint underline and keeps it
  out of the Problems panel, and hides Information behind its "Show Infos" filter.
  A level mapped to either is published, counted in the `cflint scan complete` log
  line, and then invisible, which reads as a diagnostic that was never produced.
  That is exactly how CRITICAL and CAUTION went missing: both were absent from the
  switch, so the second-most-severe level CFLint has was reported more quietly than
  COSMETIC. `TestEveryCFLintLevelIsVisible` states the enum in full and fails on a
  level that maps to Hint or Information; the default arm returns Warning for the
  same reason, so a level added upstream surfaces rather than disappearing.

  **`linting.minSeverity` filters on CFLint's raw scale, not the mapped severity.**
  Because INFO and COSMETIC fold up onto Warning, a floor applied after the fold
  could not tell them from a real WARNING and would keep every INFO it was meant
  to drop. `meetsFloor` therefore runs in `toDiagnostics` against `issue.Severity`
  before `mapSeverity` sees it, and a severity the enum does not list is kept
  whatever the floor — a setting that never mentioned a level should not be what
  hides it.

  `Linting.Enabled` is a `*bool` for the reason `completions` documents: the block
  has two keys now, so a plain bool could not tell "turned off" from "not
  mentioned", and `mergeLinting` unions key by key — otherwise a child config
  naming only `minSeverity` would switch linting off while appearing to tune it.
- `clif.findRefs` writes its `refs-<name>.md`/`.dot` report only when its third argument is
  `true`, and — like `.exportDeps`' `deps-<name>.md` — only through `reportPath`, which refuses a
  name that is not a plain file name and a directory outside the workspace roots. Both come from
  the command's arguments: `x/../../escaped` as a function name, which `filepath.Join` cleans, wrote
  a file one directory above the source. `TestFindRefsReportNameCannotEscape` pins it. `reportPath` checks text, so the write itself
  goes through `writeReport`/`createReport` (the code map too): an `os.Root` on the report's
  directory, which refuses a symlink planted at the report's name — the old write followed one
  and emptied its target — and owner-only permissions. Only the name is confined, so a
  workspace reached through a symlinked directory still gets its reports. The code map's
  directory is also resolved (`internal/confine`) and must stay inside the workspace, since a
  committed `.clif -> /elsewhere` had `MkdirAll` and the map land there. It used to write unconditionally, which meant the code action on an ordinary "find all
  references" gesture dropped two files beside the source file being read. The plain code actions
  pass two arguments; a separate "Export references to X to a file" action passes the third.
- **A known-issues report named in the config is written only where `internal/confine`
  allows** (`exportScope`'s `writeRoots`; the CLI's `reportRoots` and `cflint.WriteTargets`): the
  folders open in the editor and the config's directory for `clif.exportUnresolved`/`.exportCFLint`,
  the directories scanned and the config's directory for `unresolved --write`, the directories
  linted for `cflint --write`, compared after symlinks. The configured `workspacePaths` do not
  count, since the config naming the report would then name its own licence. The config comes
  with the repository, so `"file": "../../.zshrc"`, or a committed symlink at the default
  report's name, had the export code action overwrite that file. A link that stays inside is
  still followed, which is why these do not go through `writeReport`.
  `TestExportRefusesAReportOutsideTheWorkspace`, `TestUnresolvedWriteStaysUnderTheScan`,
  `TestWriteTargetsRefusesALinkOutOfTheLintedFolder`.
- `workspace/executeCommand`: `clif.reindex`, `.format`, `.showComponentPath`,
  `.restartDaemon`, `.showResolvers`, `.showFileIndex`, `.showConnections`,
  `.openActiveApplicationFile`, `.goToMatchingTag`, `.copyPackage`, `.findRefs`, `.exportDeps`,
  `.scanWorkspace`, `.generateCodeMap`, `.showCodeMapStats`, `.resolveRoute`,
  `.exportUnresolved`, `.exportCFLint`, `.explainCall`, `.searchDocs`, `.docEntry`.

  **Everything a CLI subcommand did for an editor task has a server route**, because Zed
  asked the zed-cfml extension to drop its `tasks.json` tasks and reach the server instead.
  Zed has two routes: code actions (`cmd-.`, every version) and, from 1.21, an LSP command
  picker listing `executeCommandProvider`. A command missing from that list is not in the
  picker, so a new one goes in the list as well as in `handleExecuteCommand`'s switch. **Zed
  ignores a command's return value**, so anything the user must read goes through
  `window/showMessage` or a written file; returning it as well costs nothing and serves
  clients that do read it. The mapping from the removed tasks: `scan` →
  `.scanWorkspace`, `format` → `textDocument/formatting`, `explain` → `.explainCall`,
  `unresolved` → `.exportUnresolved`, `cflint` → `.exportCFLint`, `refs` → `.findRefs`
  (and `textDocument/references`), `deps` → `.exportDeps`.

  Code actions (`internal/server/codeaction.go`), in order: "Explain call resolution on line
  N" when the cursor's line holds a call; the find/export actions for the word under the
  cursor; then, wherever the cursor is, "Export dependency graph for <file>" (`.exportDeps`
  with the URI alone, the whole-file graph) and the workspace actions, "Scan workspace for
  parse errors", "Export unresolved calls report for the workspace" and "Export CFLint report
  for the workspace".

  **`explainCall` parses the buffer afresh rather than using the cached ParseResult**
  (`parseForCalls`). The cache cannot answer "what calls are on this line": an edit outside a
  function reparses it shallowly, which drops every call recorded inside one
  (`resetFuncCaches`), and an edit inside one shifts scopes and refs but not calls. One
  keystroke therefore left it reporting no calls, or calls on the wrong line.
  `TestExplainCallAfterAnEditOutsideAFunction` fails if the command goes back to the
  cache. The private parse needs no document lock. The code action does not parse at all:
  `lineMayHoldCall` looks for a name before a `(` on the cursor's line. It was a memoised
  full parse, which missed after every edit, and Zed asks for code actions each time the
  cursor settles, on the read goroutine: 11ms on a 12,000-line component and 87ms on a
  65,000-line one. Against the parser on tassweb's `kiosk.cfc` and `timesheet/persist.cfc`
  the text check misses 5 of about 23,500 call lines, each a call split over lines, and
  the command's own answer is exact regardless. The report text and the selection by line and filter are
  `resolve.WriteExplanation` and `resolve.CallsOnLine`, shared with the CLI, so the two print
  the same thing. The server's answer uses the session's resolver and index, where the CLI
  builds its own from the file's nearest config, so the two can disagree for the reason
  `--root` exists.

  `.exportDeps` had the same cache problem and now parses afresh the same way. Reading the
  cached ParseResult, a comment added above the component left it no calls, and the graph fell
  back to the index's component refs: `controller.cfc --> service.cfc (line 2)` in place of
  the functions called. `TestExportDepsAfterAnEditOutsideAFunction` pins it.

  **`resolveRoute` exists so an editor does not keep its own copy of the
  convention.** A command that takes a route by hand and go-to-definition on a
  route in source must answer alike; two resolvers reading two configs disagree
  quietly, and the wrong one still opens a file. It answers rather than errors
  when no convention is configured, because a JSON-RPC error is not something a
  command handler can show anyone.

  **`generateCodeMap` reuses the server's index instead of building one** — it is
  already current, and re-reading every `.cfc` is the more expensive half of a cold
  CLI run. It runs on `safeGo` with `context.Background()` (the handler's ctx is
  pooled and reset on return, same reasoning as `scanWorkspace`) and reports through
  `window/showMessage`, because a blocking `executeCommand` freezes the editor for
  the ten-plus seconds a large workspace takes. `codeMapOutputPath` refuses a target
  outside the workspace: the arguments come from whatever asked the editor to run
  the command.

**A base component's variable only its subclasses set** (`subclassComponent`,
`subclass_refs.go`): after the extends chain, a `variables.x` or unscoped receiver
the file never types, and no function of it declares, is what the leaf subclasses
under the file hold for it, as `a|b` alternatives (the spelling `withSubclasses`
uses). **Every leaf must type it**, by a ref of its own or of a component between
it and the file; one that does not, or a file nothing extends, gives nothing,
since the others' answer would be a guess. `this.x` and other scopes are not
asked. The walk costs a scan of the index's extends map per untyped receiver and
is capped at 16 components. `TestABaseComponentsVariableIsWhatItsSubclassesHold`.

**A base component calls what its subclasses declare** (`subclass_overrides.go`), the
method side of the rule above. A bare or `this.` call the file and its chain do not
answer is accepted (as `TargetDynamic`) when **every** leaf subclass under the file
declares the method, itself or through a component between it and the file: qb's
AbstractQueryBuilderSpec calls each grammar spec's `selectAllColumns()`. And a method
the file declares **only to throw** (`abstractBody`: its plan is nothing but `throw`)
returns what every leaf's override returns, as `a|b` alternatives
(`abstractOverridesReturn`, first in `returnComponentOf` and in the producer's `call`):
`getBuilder()` throws "Must be implemented in a subclass" and each spec returns a
QueryBuilder. One leaf without the method, or without a component return, gives nothing.
`TestAnAbstractBaseCallsWhatEverySubclassDeclares`.

**A variable the parse left untyped is read from its last assignment at lookup**
(`assignedFromCall`, `assigned_call.go`): `x = receiver.method( … )` or
`prc.x = …` on one line, the receiver typed as any is, the method's return per
alternative (`a|b` receivers), every one of which must return a component. It is
the last step of `receiverComponent`, bounded by `receiverComponentD`. It exists
because the parse types an assignment from its own file's refs, and an abstract
handler's `variables.ormService` is held only by subclasses.
`TestAVariableAssignedFromASubclassHeldReceiverIsTyped`. **A method that exists on every
alternative and returns no component makes the variable `$any`** (`untypedOnEvery`), which is
what the parse's `walkChainRest` makes it when it can see the receiver: the two cannot
disagree about one assignment. Only the last call of the right-hand side, and a method one
alternative lacks, or a component one alternative returns, is still no answer.
`TestAVariableAssignedAnUntypedReturnIsDynamic`; 259 findings over the corpus, 195 of them
MasaCMS's `event.getValue(…)`. **The assignment runs on to the
next line while a parenthesis is open and while that line starts with `.`**
(`joinAssignmentLine`, shared by the table in `source_facts.go` and the line walk in
`loop_element.go`, which `TestTheAssignmentTableAnswersAsTheLineWalkDid` holds alike): a
chain written a call to a line was read to its first line and typed as the head, which on
MasaCMS was a feedBean whose `onMissingMethod` accepted every call on the iterator.
`TestAChainOverSeveralLinesIsReadWhole`.

**A branch union holds for a call passing only literals** (`branchUnionCallReturn`,
`producer_source.go`): a body that branches on an argument is evaluated per call, and when
that cannot decide (feedBean.getIterator() also tests the feed's own state), the parse's
union of the branches reaching the body's only return stands, if that return is a local the
body declares. A call passing a value it does not spell gets nothing, as before, since
evaluation might have named one branch; a field, or a second return, may hold something the
union does not. `TestABranchUnionReturnHoldsWhateverTheCallTakes`, beside
`TestLiteralSwitchProducer` and `TestGuardedLazyFieldsFailClosed`, which pin the cases it
must not answer.

**A guarded field is the one component every write gives it** (`sharedFieldContract`,
`producer_flow.go`): `if ( isNull( variables.x ) ) variables.x = new X(); return variables.x;`.
Two extensions. The `structKeyExists` guard and a keyed read take a key the call makes
literal (`structKeyCondition`, `literalKey`), so the cfwheels CLI's
`getService( "helpers" )` over `variables.services[ name ]` is what
`variables.services.helpers` is written. And a statement the plans cannot read no longer
voids the proof on its own (`fieldTextProved`): when every reference in the file's text to
the variable holding the field is in a statement a plan read, and the variables scope is
never used whole (`structDelete( variables, "x" )` touches x alone), the unread statement
cannot touch it. A large component always has one, and lost every guarded field to it.
`TestAKeyedLazyCacheReturnsWhatItsKeyBuilds`, `TestAGuardedFieldIsProvedFromTheText`. A **request**
key is a guarded field when the file writes it and every other file that can write it in
the same request writes it as the same component, each write a statement its plans read
(`requestWriterComponents`): writes are counted outside comments (`withoutComments`), and
another application's `Application.cfc`, neither extending the other, writes another
request (`otherApplication`). FW/1's five example applications each create
`request._framework_one` in their own `Application.cfc`. `TestALazyRequestFieldIsWhatItsApplicationStores`.

**A base handler's computed view name and key are read per subclass.**
`moduleActions` expands `view( "#variables.handler#/x" )` for each leaf subclass
under the handler, with the literal the leaf (or a component between) assigns, and
the handoff then reads the action *on behalf of that leaf* (`lookupCtx.leaf`):
`variables.ormService` is its service, and `results[ variables.entityPlural ]` is
the field its literal names (`keyedFieldRe`). The same base file read without a
leaf still answers the union of its subclasses. `TestAComputedViewAndKeyAreReadPerSubclass`.

**An untyped argument is what every caller passes it** (`argumentFromCallers`,
`arg_callers.go`), in a batch scan only: `Resolver.InferArgsFiles` is the file set to
search, set by `unresolved.NewResolver` when `Options.InferArgs` is (the CLI's
default; `--no-infer-args` turns it off). Every caller that passes it must type
it, from the argument's tokens (a multi-line call reads), and one that cannot be
placed or typed leaves it untyped. It is the **last** step of `canResolveCall`:
inside `receiverComponent` it ran before the dynamic rules and turned accepted
calls into findings. `TestAnUntypedArgumentHoldsWhatEveryCallerPasses`.

**A computed `invoke` selector is a call to every method its literal parts allow**
(`invokeSelects`): `invoke( getGrammar(), "compile#command.getType()#", { blueprint: this } )`
calls compileCreate, compileDrop and the rest, and was read as calling none of them. The
caller index records the literal start of such a string (`computedCallerKey`), and a
selector with no literal start is still not read. **An untyped invoke target** still
withholds inference, unless every workspace declaration of the method lies in one class
tree, under the topmost component fd's file extends that declares it
(`declaredInFamilyOnly`): then whatever the target is, a call to that name lands in that
tree with that bag. qb's grammar is built by WireBox from a setting. Sibling overrides
(two grammars) are in the tree; an unrelated component declaring the name is not.
`TestAComputedInvokeSelectorPassesItsArguments`,
`TestAnInvokeOnAnUntypedTargetReachesTheOnlyFamilyDeclaringTheMethod`.

**A bare argument type names the one component of that name** (`namedType`,
`uniqueComponentNamed`): beside the declaring file first, else the only `.cfc` in the
workspace with that file name (none or several: untyped), since the engine checks a
component argument by name. qb's grammars declare `required QueryBuilder query` for a
file in another directory. A generated getter stored from an init argument reads a bare
type the same way (`initArgDeclared`). A bare **return** type is still read beside the
declaring file only (`TestABareReturnTypeNamesTheComponentBesideIt`).
`TestABareArgumentTypeNamesTheOnlyComponentOfThatName`,
`TestAGeneratedGetterIsWhatInitStoresFromATypedArgument`.

**A callback parameter is what every path through its provider invokes it with**
(`callbackParameterComponent`, `callback_walk.go`): `testCase( function( builder ) { … } )`
types `builder` from the argument testCase passes `callback( builder )`. The provider's
plan is walked path by path, an invocation inside an if, try or loop counting with the
value on that path, and the names holding the callback are tracked per path: reassigning
one drops it there, and a guard proving a name holds no function (`isNull( x )` true,
`isClosure( x )` and `isCustomFunction( x )` both false) drops it on that branch
(`narrowCallbackAliases`), which is what QueryBuilder's union() and with() rely on. A
callback handed whole to a method of a known component (`union( arguments.input, true )`,
`with( argumentCollection = arguments )`) is followed there, up to
`maxCallbackForwards`. Every invocation must agree; storing it, passing it to code that
cannot be followed, invoking it inside a larger expression or condition gives nothing.
`TestACallbackParameterIsWhatEveryPathInvokesItWith`.

Chained-call continuation records capture each hop's byte origin before consuming
its arguments; emission still follows nested argument calls. The hop's line is
its own token line. This lets the ordinary caller omission proof distinguish
same-named calls and prevents interpolated tag-expression top-up from duplicating
one lexical call. Synthetic/ambiguous origins retain their existing safeguards.
`TestChainedCallsKeepEachHopByteIdentity` and
`TestChainedInterpolationCallsAreNotDuplicated` cover source offsets, tag/script
forms and repeated names.

Caller inference scans all indexed candidates without file or call-site count
cutoffs by default, including constructor callers. Optional `.clif.json`
`callerInferenceLimits` bounds candidate files, owner files, sites and constructor
files independently; positive values enable limits and nonpositive values leave
them unlimited. The policy block replaces an inherited block as a whole. Unplaced receivers and unknown supplied
values still withhold inference. An ordinary call with an unknown receiver is
ignored only when exact byte identity and balanced syntax prove that it omits
the target argument. Opaque `argumentCollection` and unknown `invoke` owners
still withhold inference. Runtime component paths containing interpolation
markers cannot become caller-derived literal types; recursive inference retains
its depth bound.
It runs only in batch scans and can cost more for common names in large corpora.
`TestArgumentCallerBudgetUsesOwnership` and
`TestArgumentInferenceBeyondFormerLimits` cover unrelated owners, former limits,
constructors and an unknown value after the former site cutoff.

**`argumentCollection = <a local struct>` passes what its key was written**
(`localCollectionArgument`, `local_collection_args.go`, from `callArgumentAt`): Mura builds
`var sArgs = structNew()` a key at a time and hands it to its publisher. The struct must be
declared in the caller (`structNew()`, `{}` or a literal), and every reference to it up to the
call must write a key (`sArgs.Key = v`, `sArgs[ "Key" ] = v`) or read one; anything else
(passing it on, `structDelete`, replacing it) leaves the argument untyped, as does a key
written twice or a written name the caller assigns again before the call. A struct that never
holds the key does not pass it. MasaCMS 2,465 → 2,400.
`TestAnArgumentPassedInALocalStructIsTypedByItsKey`.

Literal array-returning `entityLoad` forms use indexed persistent-entity metadata
for loop element types: a literal entity name, optionally a filter struct and an
explicit `false` unique flag. Shadowed calls, dynamic names, scalar-ID overloads,
unknown/true unique flags and unsupported options stay unknown. Native array
`.each(function(element){...})` can use the same independently established
collection element contract (including existing ORM relationships and cborm
`getAll`). Only the first positional callback parameter is typed, only for an
exact closure call span and stable parameter. This assumes the collection keeps
its producer's homogeneous element contract; it does not establish types after
arbitrary runtime mutations or escapes. See `collection_callbacks.go` and
`TestLoadedEntityCollectionsTypeLoopsAndEachCallbacks`.

**A field only ever written an empty collection is that collection** (`emptyCollectionField`,
`native_field.go`, after `builderMember` in `inferredReceiver`): ColdBox's `variables.executors = {}`
in init() and keys written later is a struct, which a struct literal typed only in a
pseudo-constructor. Every direct write across the component, the classes above it and those below
it must be the same empty collection (`{}`, `[:]`, `structNew()`, `[]`, `arrayNew(1)`); a file of the
family with includes, or one indexing the variables scope by a computed key, declines. A struct key
may hold a closure, so a call of a method no struct has is not answered. A struct also takes the
`java.util.Map` methods (`nativetype.Method`), which Lucee's `keySet`/`entrySet` tests call. Cached
per file, name and index generation. `TestAFieldOnlyEverAnEmptyCollectionIsThatCollection`.

**A struct default is set aside for a method a struct lacks** (`nativeDefaultOverwritten`):
`StructNew()` and `{}` type a variable as `native.struct`, and tassweb sets
`VARIABLES._parent = StructNew()` in every component's pseudo-constructor before
`init()` replaces it with an argument declared `struct` (CFML lets a component
through). Taken as the nearest assignment, 7,000 `_parent.getService()` calls were
checked against a struct. A struct-typed ref, or a bare placeholder a variable was
last assigned, is passed over when the method called is not one a struct has
(`docs.LookupBuiltinMethod`, the method-exists check's own test); the lookup then
takes the next ref, a `componentResolver` or the extends chain. Only structs: any
other native value missing a method is a finding to report.
`TestANativeDefaultDoesNotOutrankWhatReplacesIt`.

## Capabilities the VS Code extension has and this server does not

The `cfmleditor` extension stands its own language providers down whenever this
server is running, on the rule that enabling the server hands it the language.
That rule is simpler to hold than a per-capability list, and it costs two
things the extension could answer and this server cannot. They are listed here
so the loss is deliberate and so whoever implements one knows what it has to
match.

It was four. Variable definitions were the largest, and they are closed — the
conformance suite below is what made that a measured change rather than a claim.
`textDocument/typeDefinition` is closed too, behind `features.typeDefinition`; see the LSP
surface. Docblock completion is closed (`internal/server/docblock.go`): `/**` expands to a
comment block for the function, property, component or interface that follows it, with a
tag line per argument and a blank line after the hint; inside a `/** … */`, `@` offers the
attributes of the tag it documents (from `internal/docs`) and a function's argument names,
and `@arg.` a cfargument's attributes. `capabilities()` advertises `*` and `@` as well. The
extension's `cfml.docBlock.gap` and `.extra` settings are the `docBlock` config block
(`config.DocBlock`, default a gap and no extras), carried through every config hop like
`completions`. Differences from the extension, both deliberate: tag completion needs the
cursor inside a doc block (the extension offered it anywhere in a component file), and the
function's arguments are read from the declaration text rather than the document's parse,
because the comment being typed is usually unclosed and hides the rest of the file from it.
`TestSlashStarStarExpandsToADocBlock`, `TestAtInADocBlockOffersTheTagsAttributes`.

| Missing here | Extension's implementation | Notes |
|---|---|---|
| `textDocument/documentColor` | `CFMLDocumentColorProvider` | Colour swatches and the picker for colour literals. Wholly absent here; nothing in the parser records them. |

`documentColor` is the one with no foundation at all. Until it lands, a user who enables the
server loses it — which is worth remembering when one is reported as a regression rather than
a gap.

**The list is measured, not maintained by hand.**
`internal/server/definition_conformance_test.go` replays the extension's own
`provideDefinition` suite — same cases, same cursor-marker syntax, fixtures
copied into `internal/server/testdata/conformance` with their source commit
recorded. All 41 pass. `knownGaps` is empty and kept, because **the test fails when a
known gap starts passing**, so closing
one cannot go unnoticed and the list cannot rot into decoration.
`TestKnownGapsAreRealCases` fails on a gap naming a case that no longer exists,
so a renamed case cannot hide as a fixed one.

The fixtures live in the *package's* testdata rather than the repo root's. They
are a whole second workspace, and putting them in the shared `testdata/` changed
what the repo-wide scans find — `TestReachabilityDoesNotFollowContains` failed
on the fixture `Application.cfc`'s `onRequestStart`, correctly, because a second
application had appeared in the tree it walks.
