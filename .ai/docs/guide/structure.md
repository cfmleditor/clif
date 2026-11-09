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
- `internal/parser/script_parser.go` — `scriptParser` and the two parse loops (top-level
  `parse()` and `handleBodyToken`); see the scope-dispatch note below
- `internal/parser/tag_parser.go` — `tagParser` for `<cffunction>`, `<cfcomponent>`, etc.
- `internal/parser/scanner.go` — byte-level tokenizer (`isIdentStart` treats `_` and `$` as
  identifier starts)
- `internal/parser/ast.go` — resolver matching: `BuildResolverSet`, `ResolveFromCall(Full)`,
  `matchResolverWithCache`, `splitPrefix`/`findPrefixPos`, `substitutePlaceholder`
- `internal/server/server.go` — `Server` struct, capabilities, per-URI caches
- `internal/server/handler.go` — LSP method dispatch and `workspace/executeCommand`
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
