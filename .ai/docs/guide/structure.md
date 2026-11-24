# Key structural notes

_Which source file holds what, and the tests that read source files by name. Moved out of CLAUDE.md unchanged._

**Files are split by concern, and some tests read source by file name.** `TestTagDispatchMatchesTheFastSkip`
reads `tag_parser.go` and `TestDidCloseCoversEveryPerDocumentMap` reads `handler.go`, so the tag dispatch
and `handleDidClose` stay in them; `TestEveryAcceptPathRecordsATarget` parses `resolve.go`, so a
`return ""` that accepts a call stays in `canResolveCall` after its `tr.hit` (a helper decides, the caller
records the target), or is a `return true` in `bareCallInComponent` / `bareCallThroughContext` / `methodAccepted` (the helpers the test names, which record their own); the parser's loop and dispatch-switch tests read every non-test file of the package
(`readAllParserSource`). Move code with a declaration comparison (the same top-level declarations and
code lines before and after), then `unresolved` byte-identical on a corpus.

- `internal/parser/result.go` — `ParseResult`, `ParseOptions`, `Parse()`/`ParseWithOptions()`,
  lazy per-function caches, `replaceExpressions`, `resolvePendingCalls`, property accessors
- `internal/parser/cfparser.go` — `ClassifyRegions()` plus the standalone
  `ParseFunctionDefs`/`ParseComponentRefs` helpers
- `internal/parser/script_parser.go` — `scriptParser`, its constructor and the top-level
  `parse()` loop; see the scope-dispatch note below. The rest of `scriptParser` is split by
  concern, all one type: `script_body.go` (`parseBody`, `handleBodyToken`, return handling),
  `script_functions.go` (function declarations, access modifiers, loop/catch vars),
  `script_assign_refs.go` (assignments, `new`/`createObject` refs, chains in bodies),
  `script_chain_record.go` (recording calls and chain hops), `script_mocks.go` (call
  resolution helpers, mock markers), `script_arglist.go` (argument lists),
  `script_property_parse.go` (`property` declarations), `script_skip.go` (paren/bracket
  skipping, arrow params, mark/rewind)
- `internal/parser/tag_parser.go` — `tagParser` for `<cffunction>`, `<cfcomponent>`, etc.
- `internal/parser/scanner.go` — byte-level tokenizer (`isIdentStart` treats `_` and `$` as
  identifier starts)
- `internal/parser/ast.go` — resolver matching: `BuildResolverSet`, `ResolveFromCall(Full)`,
  `matchResolverWithCache`, `splitPrefix`/`findPrefixPos`, `substitutePlaceholder`
- `internal/server/server.go` — `Server` struct, capabilities, per-URI caches
- `internal/server/handler.go` — LSP method dispatch and `workspace/executeCommand`
- `internal/parser/rename_local.go` — `PlanLocalRename`: one walk over a function's tokens finds the
  ranges a local variable's rename edits and the names it already uses (`Taken`), and refuses with a
  reason where a use cannot be followed; `ClassifyVariableName` for a new name
- `internal/parser/query_columns.go` — `QueryColumns`/`DeclaredQueryColumns`: the columns of a query
  variable from its `<cfquery>`, `queryNew` or `queryExecute`, branch-aware; `sqlSelectColumns`
- `internal/parser/tag_problems.go` — `TagProblems`: unclosed, unmatched, misplaced and missing or
  repeated attributes of CF tags; `duplicate_arguments.go` repeats of a function's arguments
- `internal/parser/component_header.go` — `ComponentDeclarationBounds`/`ComponentBodyEnd`: where a
  component's declaration and body end, script or tags
- `internal/lex/wild.go` — `Wild`/`WildMemo`: the bounded backtracking matcher behind
  `placeholderGlob` (`parser/matchers.go`) and `wildPattern` (`resolve/wild_pattern.go`)
- `internal/server/structure_diagnostics.go` — the debounced tag-structure check and its publish;
  `rename.go`, `inlay_hint.go`, `docs_command.go` and `lifecycle_actions.go` are the rename,
  inlay-hint, documentation-command and Application-event code-action handlers
- `internal/resolve/resolve.go` — `ComponentPath`, `CanResolveCall`, `ExplainCall`, extends walking
- `internal/config/config.go` — `.clif.json` schema and defaults
- `internal/docs/` — generated; regenerate via `make generate`, never hand-edit

**Scope-prefixed assignments:** each handled scope (`local.`, `variables.`, `this.`,
`arguments.`, `request.`, `session.`, `application.`, `server.`) needs its own `case` in *both* dispatch
switches (`scriptParser.parse()` and `handleBodyToken`) routing to
`parseScopedVar`/`parseBodyScopedVar` — that handler is the only one that correctly
distinguishes `scope.name = rhs` (assignment) from `scope.name.method()` (bare call) for a
two-token-prefixed LHS. Any scope keyword *not* listed falls through to `checkAssignRef`'s
default path, which only recognizes a bare `x = ...` (single identifier directly followed by
`=`); for a scope-prefixed LHS the next token is `.` not `=`, so the statement is silently
misread as a bare-call check and any component type the RHS establishes is dropped.
`server.` is listed too, since `Server.cfc` declares into it and go-to-definition looks there.
`url.`/`form.`/`cookie.`/`cgi.`/`client.` deliberately aren't listed (those scopes hold primitive
request/config data, not component instances) — add them the same way if a project assigns
components through one.

**The shared scopes keep their own scope in the declaration.** `request.`, `session.`,
`application.` and `server.` share the variables-scope handling for everything *except* the
`VarDef` they record: the component ref is filed globally and the calls are extracted alike.
They used to be dispatched as `ScopeVariables` outright, which put `application.cache` among
the component's variables and left go-to-definition on it nothing to find in a script-syntax
`Application.cfc` (tag syntax always had it right), and made `application.helper =
function(){}` at component level a method of the component. `sharedScopeOf` maps the keyword;
`TestSharedScopeAssignmentsKeepTheirScope` pins all three.
