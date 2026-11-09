# Parser design

_The parser (`internal/parser`): scanner, the two parse loops, call extraction, and the invariants each fix established. Moved out of CLAUDE.md unchanged._

## Parser design (`internal/parser`)

The parser is intentionally **not** a full tree-sitter traversal — it uses a fast line-by-line
`Scanner` for script regions and string/regex searches for tag regions. tree-sitter is used by
the *formatter*, not the parser.

- Entry points: `Parse(fileURI, content, resolvers...)` and `ParseWithOptions(fileURI, content,
  ParseOptions{...})`. `ParseOptions` gates the expensive work: `ExtractCalls`, `ExtractLinks`,
  `ScanAllScopes`, `FindCalls`, `Shallow`, plus lookup hooks (`FuncLookup`, `BeanLookup`,
  `BuiltinReturnLookup`).
- **Function bodies are parsed lazily** — `FuncVars`/`FuncRefs`/`FuncCalls`/`FuncLinks` parse
  and memoize per function, keyed `"start:end"`.
- **Incremental edits** (`incremental.go`): `ParseResult.ApplyEdit` classifies an edit as
  `EditInFunc` (shift line numbers + invalidate that one function's caches), `EditGlobal`
  (`reparseShallow()` of signatures), or `EditFull`. Parse entry points recover from panics and
  fall back to a shallow re-parse rather than crashing the daemon.
- `edit_parser.go` holds cursor-context helpers (`FindCallContext`) used by signature help and
  completion.
- **The scanner's API is `NextSkipComments`/`PeekSkipComments`, `Save`/`Restore` and
  `LastBlockComment` — nothing else.** The raw `Next`, `Peek`, `Pos`, `Line` and `Rest` are
  exported and have no caller in or out of the package. That is what makes the one-token
  lookahead safe: a peek caches the token and the position past it, and the matching next takes
  it rather than scanning the same bytes again. A peeked token carries the assignment it made to
  `LastBlockComment`, because a peek's effect on that field has always outlived the peek and
  `parseFunction` reads and clears it between the two. Adding a scanner entry point that moves
  `pos` means clearing `peeked`, as `Restore` and the raw `Next` do.
- **Don't write `switch strings.ToLower(x)` on a parse path.** `strings.ToLower` returns its
  argument untouched when there is nothing to lower, so such a switch is free on all-lowercase
  source and allocates a throwaway string for every identifier with a capital in it — which is
  `getUser`, `userDAO`, every `ARGUMENTS` and `VARIABLES`. It was 22% of everything a parse
  allocated. Use `var buf foldScratch` and `switch string(buf.lowerFold(x))` (`fold.go`): the
  compiler elides the conversion in `switch string(b)` and in `m[string(b)]`, and the array stays
  on the stack. Spell the fold inside the switch expression — binding it to a name invites
  holding it across a nested fold. `TestFoldingAnIdentifierDoesNotAllocate` parses one document
  mixed-case and lowercased and fails if the capitals cost allocations.
- **Keep a struct's small fields together.** A bool between two 8-byte fields
  costs eight bytes. `CallSite`, `scriptParser` and `Scanner` each group theirs
  at the end for that reason; spreading them back out puts `CallSite` from 112
  to 120 bytes in every call slice and moves the parsers up a size class. It is
  only worth doing where it crosses an allocator size class or the struct is
  held by value in a slice — `ParseResult` would shrink 40 bytes and still
  occupy a 640-byte block. `TestParserStructsKeepTheirSize` pins the three;
  .ai/docs/LINT-PLAN.md stage 3 has the measurements.
- **A local `var buf [N]byte` stays on the stack; a struct holding a slice of its own array does
  not.** `chainBuilder` accumulates into an array addressed by length for that reason — the
  shorter `c.rest = c.arr[:0]` defeats escape analysis and moved all eleven of its call sites'
  builders to the heap, which cost more than the buffer growth it saved. Check with
  `go build -gcflags=-m` rather than assuming.
- **Recording a call is gated on `ExtractCalls`; consuming one is not.** The gate lives in
  `addCall` alone, and every dispatch that walks a call and its argument list runs in every
  mode. It used to sit on the dispatch itself, at nine sites, and the consequence was not that
  an argument list was skipped — nothing skips it — but that it was scanned as if it were
  statements. `svc.save(force = true)` then declared a variable called `force`, and
  go-to-definition on `force` anywhere in the file landed on that argument. The same shape is
  why `{force = true}` needs `skipLiteralGroup` and a script-syntax `<cfquery>`'s attributes
  need `parseScriptTagAttrs`: a group the scan walks *into* rather than *over* is read as code.
- **An argument list is the one place a call could hide.** `skipParenBody` discarded a `(...)`
  group a token at a time, so `writeOutput(svc.getName())` recorded only `writeOutput` — on a
  realistic function body 7 of the 12 calls present. A condition, a return, an assignment's
  RHS, a concatenation, a struct or array literal and a ternary all extract correctly. The
  cost was not completeness: a method called only from inside an argument list had no edge
  into it, so `internal/codemap` read it as unreachable and `unresolved` never checked it.
  There were **three** copies of that paren loop and fixing two left `a().b(svc.c())` losing
  `svc.c`; they are one function now. The scan recurses, so `maxArgNesting` caps it at 64 —
  Go cannot `recover()` from stack exhaustion, so the guard on every parse entry point would
  not catch a runaway.
- **`new` needs its own arm wherever a chain is walked.** Treat it as an ordinary keyword and
  the scan meets `models.User(` on its own and invents a call to `User` on `models` — a made-up
  answer, which is worse than the missing one it replaced.
- **There are two variable scans and both must agree.** `scriptParser` is the full one;
  `globalScriptParser` is a much smaller one that skips every function body outright, which is
  what makes `VariablesVars`/`ThisVars` answerable per keystroke. Every rule about what does
  *not* declare a variable therefore has to hold twice — named arguments, struct-literal keys
  and script-tag attributes fooled both, and fixing one left the other reporting the same
  phantom names. `TestBothVariableScansAgree` pins them together.
- **`ident ident =` is what identifies a script-syntax CF tag**, not a list of tag names.
  Importing `internal/docs` for one would pull a generated 6,582-line file with an `init()`
  into a package kept deliberately dependency-light, and a tag it did not list would silently
  go back to declaring variables. The spelling is unambiguous in CFScript. It needs one guard:
  `parseFuncBody` runs the *script* parser over a tag function's raw text, so
  `<cffunction name="save">` reaches the same test — hence the `afterLT` flag, and why the
  attribute scan consumes **pairs** rather than everything up to the body. An earlier version
  that swallowed "up to the body" ate whole `<cffunction>` bodies and every local inside them.
- **A function assigned to a name is a method.** `this.helper = function(a) { … }` and
  `variables.helper = (a) => a` are how a component exposes a method it builds rather than
  declares, and the parser recorded only a variable — so `Funcs` held none of them and the
  component had no completion, no signature help, no go-to-definition and nothing in the index
  for any of them. `parseFunctionValue` fires only outside a function body and only for `this.`,
  `variables.` and an unscoped name: a `var`- or `local.`-scoped closure is a local value, and
  declaring one as a method would put a helper private to one function into every caller's
  completion list. The paren-less single-argument arrow (`this.x = a => a * 2`) is deliberately
  not recognised — telling it from `this.x = a` needs three tokens of lookahead on every
  assignment whose RHS is a bare identifier, which is most of them.
- **Not every binding is an assignment.** The var-decl parsers only ever looked for `=`, so
  `for (var row in qry)` declared nothing — in every for-in loop, which is how CFML iterates a
  query, an array and a struct. `catch (any e)` was the same, in every catch block there is.
  `declareVar` files both; outside a function body they land in variables scope, since there is
  no local scope to put them in, and an unscoped `for (row in qry)` lands there wherever it is,
  which is CFML's rule for any unscoped assignment.
- **Two-character operators are the scanner's job, and the two are not alike.** `?.` is folded
  into `TokDot`, so every chain walk sees `svc?.save()` as `svc.save()` without a case of its
  own — a receiver missed in one of them is recorded as a *bare* call, which resolves as an
  unqualified function: a wrong answer, not a missing one. Adjacency is required, so a ternary
  keeps its `?`. `::` is *not* folded, because its qualifier is a component rather than a
  variable holding one: `TokDoubleColon` exists so the call site carries `Component` and
  resolution looks for the method in `Foo` instead of among the file's own functions. It used
  to be two unrecognised tokens, and `Foo::bar()` was reported as
  `bar (no qualifier, not in file)` — naming the wrong problem and unresolvable even with the
  component on disk.
- **A string can hold an expression that holds a string, and only CFScript may
  assume so.** `"#DayOfWeek("{ts '2000-1-1'}")#"` is one token; taking the first
  matching quote as the end left `"#DayOfWeek("`, so the interpolation had no
  closing `#` and its call was invisible — and the swallowed terminator took the
  rest of the construct with it, 3,165 sites over 461 corpus files. `scanString`
  now steps *over* a `#...#` and over any string opened inside it. It is
  **opt-in** (`Scanner.interpStrings`, set by `scriptParser.asCFScript()`)
  because text that is not CFScript reaches the same scanner — `parseFuncBody`
  hands a tag function's raw body to the script parser — and on markup the rule
  is a *wrong* answer rather than a coarse one: `<a href="#top">x</a><a
  href="#bot">y</a>` pairs the two fragment hashes and swallows the tags between
  them. `newGlobalScriptParser` opts in for correctness, not for calls: the two
  variable scans must tokenise a file alike, and without it `variables.a =
  "#f("x=1")#"` declares a phantom `x` in variables scope. The scan is
  speculative — unclosed nesting falls back to the plain quote-to-quote scan, so
  a token is never worse than before — and capped at `maxStringNesting` for the
  reason `maxArgNesting` exists.
- **CFML's string escapes are not C's, and the scanner used C's.** A quote is
  escaped by **doubling** it (`"say ""hi"""`); a backslash is an ordinary
  character. Honouring `\` meant a string ending in one — `listLast( uri, "/\" )`,
  `"lucee-tests\" & id`, a regex class, any Windows path — did not close where it
  ends: it closed at the next quote anywhere in the file and swallowed every call
  in between. One occurrence cost 333 call sites in a 1,918-line component. The
  doubling is checked *from inside* the string, which is what keeps a bare `""`
  the empty string and `f("", "b")` two arguments. Both scanners must apply it:
  `scanString` tries `scanQuoted` first and falls back to `scanQuotedPlain`, so a
  `scanQuoted` still honouring the backslash is *hidden* by the fallback —
  `TestBothStringScannersApplyTheSameEscapeRule` asserts the token text under the
  flag because a test that only parsed a component passed with that half
  reverted.
- **A UTF-8 BOM is not a token, and 561 of the 5,629 corpus files carry one.**
  `ClassifyRegions` decides script vs tag syntax by asking the scanner for the
  first token and testing it against `component`; a scanner that stops at the BOM
  answers "not a component" and the file goes to the tag splitter. That is
  harmless until the file mentions `<script>`, which `isScriptFile` reads as an
  HTML page — ColdBox's HTMLHelperSpec.cfc mentions it only inside the string
  literals it asserts against and came apart into eight regions, each parsed from
  the middle of an expression. `NewScanner` skips it, rather than `isScriptFile`,
  because it should not be a token anywhere.
- **A `new` expression's argument list is an argument list.** The four paths that
  read one — `parseNewRef`, `parseStandaloneNew`, `checkReturnComponent`,
  `scanChainedCalls` — each kept a `skipBalancedParens` of their own, the
  discard-a-token-at-a-time loop `skipParenBody` replaced everywhere else. They
  were reached from the `new` arm rather than from the argument scan, which is
  how they survived the consolidation, and
  `new Query( datasource = getDatasource() )` recorded no call at all. They go
  through `skipParens` now and that function is gone.
- **A call made directly on a scope needs its own arm, and which receiver it gets
  is the whole question.** `this.init()`, `variables.buildCache()`,
  `request.getRemoteClients()` recorded nothing: both scoped-var handlers read
  `scope` `.` `name` and then looked only for `=` or `.`, so the shape where the
  statement *is* the call fell through. `this.` and `variables.` name a member of
  the component being parsed, so the call is recorded **unqualified** and resolves
  against the file's own functions; every other scope holds a runtime value, so
  the receiver is **`$any`** — recorded unqualified, `request.getRemote()` in a
  file that declares a `getRemote` is an edge to a function the call never
  reaches. **Read the scope from the token, not the `Scope` value**: the two
  groups are "this component" and "anything else", which is not a question the
  enum answers — `request`, `session` and `application` used to be dispatched
  as `ScopeVariables`, and testing the enum put every one of them in the first
  group. **`scopeReceiver` is that rule, and every path that records such a call
  goes through it**: the statement, a `return`, an assignment's right-hand side
  (`recordCallFromChain`), an argument list (`checkBareCall`, and
  `scanNestedCall` lets a scope word past its keyword guard) and the hops chained
  on (`recordChainContinuation`). Each of those had kept the scope as the
  receiver, so `x = variables.f()` reported "variable 'variables' has no
  component ref" where the statement resolved.
- **A `return` records what its expression records as a statement**
  (`returnCall`). It had a chain walk of its own that recorded the first call and
  left the rest to the outer loop, which met `.b()` with nothing before it and
  recorded a bare `b` — and read a named argument, `return f( a = 1 )`, as an
  assignment: 1,081 phantom variables over the corpus. It reaches only the walkers
  that record calls, never the statement dispatch, because `==` is two `=` tokens
  and `return x == 1` read as a statement is the assignment `x = …`;
  `TestReturnOfAComparisonDeclaresNothing` fails if it is routed there.
  `TestReturnRecordsWhatTheStatementRecords` holds the statement, the `return`
  and the assignment forms of each chain shape to the same calls.
- **A call chained onto a bare call is made on what that call returns**, so the
  call goes into the hop's chain: `f().g()` records `g` with Chain `[f]`, which
  resolution walks through `f`'s declared return type. `recordBareCallAndChain`
  left it out unless a resolver had named `f`'s component, and a `g` with no
  receiver and no chain is indistinguishable from a bare call to a function named
  g. 26,014 hops over the corpus. `super::m()` is `super.m()`, not a static call
  on a component named super.
- **A call is on the line its name is written on**, whatever chain it ends. Eleven
  places stamped a call with the line the chain *began* on instead — every hop
  after a bare call, a scope's first call (in a function and at the top level),
  an unscoped receiver alone on its line (`walkChain` reports the last name's
  line for the three assignment forms; `checkBareCall` and `parse()`'s own
  dot-chain walk track it), the hops after `createObject( … )` or `new X()`, and
  a call on a literal or after a `]`. Only the call's own line moves: a ref's and
  a pending call's stays the statement's, which is what typing reads. Moving 315
  findings' lines over the four-project corpus removed two false positives (a
  call put before the assignment that types its variable) and added none; the
  known-issues reports show line churn on regeneration. `make gapcheck` keys the
  grammar's call on its method name's row for the same reason
  (`TestAChainedCallIsOnTheLineItsNameIsWrittenOn`, .ai/docs/PARSER-GAPS.md §4.4).
- **Text that is not script is not scanned as script, and an attribute is an expression.**
  `blankNonScript` (`islands.go`) blanks SQL in a triple-backtick island (`query ds {``` … ```}`, and a
  `<cfquery>` inside any island) and the body of a `java { … }` block before the scan, in both script scans,
  byte for byte and keeping newlines and the expression in each `#…#`; read as code, `VARCHAR(255)` and
  `sum(x)` were calls, 16 findings over Lucee. A property's and a component's attribute strings go through
  `scanInterpolation` (`default="#createUUID()#"`, `hint="#now()#"`), and an unquoted `default=#f()#` is read
  between its two `TokHash` tokens. `property` is a declaration only when `startsPropertyDecl` says a name or
  a type follows (`for( property in x )` is a variable), in the parse and in the global scan alike.
  **`cfscriptClose` finds the end of a `<cfscript>` body for all three places that look**, and walks it with
  the script scanner only when a `<cfscript>` opener precedes the first closer or a second closer follows
  with no opener between, so a string holding `</cfscript>` does not end it. .ai/docs/PARSER-GAPS.md §4.5
  has the six-corpus numbers, the four gaps and the two deliberate differences left.
- **A bracket index is an expression, and `skipBracketIndex` mirrored the *old*
  `skipParens`** — it discarded its group a token at a time. `sorted[ sorted.len() ]`
  and `arr[ f() ]` recorded nothing at all, and `g( arr[ f() ] )` only `g`: the
  defect `skipParenBody` exists to have fixed, left in the one group the
  consolidation did not reach. The `[]` poison marker is unaffected — a dynamic
  key still falls through to an honest "no component ref"; only the key
  expression is read rather than thrown away.
- **A hop chained onto a scoped call needs its receiver carried.** Both
  scoped-var handlers recorded the first call and skipped its argument list, so
  the `.c()` in `variables.a.b().c()` was rediscovered by the outer loop as an
  orphaned *bare* call — in a component declaring a `c`, an edge the call never
  takes. `continueChainCalls` exists to stop that on the unscoped path, so the
  scoped path goes through the same helper rather than growing a second one; a
  call made directly on a scope carries the receiver its first call earns
  (`variables.helper().c()` walks the declared return type, `request.get().c()`
  stays `$any`). **`make gapcheck` cannot see this**: it compares line and method
  name and deliberately not the receiver, so a call against the wrong receiver
  still counts as found.
- **The parser walks a chain in three places, and a construct met mid-chain needs the case in
  all of them.** `walkChain` is the one behind every assignment's right-hand side
  (`checkVarRHS`, which `parseBodyVarDecl` now calls, `parseBodyScopedVar` and
  `checkAssignRef`); `scopedChainCall` walks `scope.name.a.b()` as a statement, and
  `checkBareCall` walks a bare receiver into a list rather than a string. It was five
  hand-copied loops, and three of the five were still reporting a bare `bar` when the first two
  handled `::`. `TestStaticCallCarriesItsComponent` lists every assignment form for that
  reason; add to it rather than fixing one walk. **`checkVarRHS` and `assignFromChain` are not
  the same answer**: after the walk, `checkVarRHS` also tries `tryExtendChain` and the builtin
  return lookup, and `assignFromChain` (behind the scoped and unscoped assignments) never has.
  That is how the code was found, kept as it was; unifying them is a behaviour change to measure
  on the corpus, not a refactor.
- **"This function calls nothing" is not "this is not a function".** `FuncCalls`
  answered both by falling back to every call in the file, so a leaf method was
  handed its siblings' calls — `deps` drew an edge out of an empty function,
  labelled with another function's line number. A whole-file scan asks
  `AllCalls` by name instead, and that one sorts by line, because its four
  callers (`unresolved`, `explain`, the code map, the MCP server) each write a
  report meant to be diffed against an earlier one and the buckets come out of a
  map.
- **`#...#` is where a computed value reaches a string, and both parsers were
  blind to it.** The scanner takes a quoted string as one token and the tag
  parser never tokenises text, so `writeOutput("id #svc.getName()#")` and
  `<cfoutput>#svc.getName()#</cfoutput>` recorded nothing. Both now hand each
  span to a scriptParser — the tag side reusing the path a `<cfscript>` body
  already takes, during the walk rather than after it so `p.inFunc` is live and
  the call lands in the right function. **A span with no `(` is rejected on a
  byte scan**: only calls are recorded, `#user.name#` and `#i#` are most of the
  interpolation in any file, and without that guard tag parsing with call
  extraction cost 21% more rather than 4.6%.
- **Six loops walk tokens and a literal reaches all of them**, so a string or a
  closing bracket goes through `handleLiteralToken` rather than each loop
  growing its own copy of the rule. `TestEveryTokenLoopHandlesLiterals` parses
  the source and fails on a loop that dispatches identifiers to
  `scanNestedCall` without a `TokString` arm beside it.
- **A member call on a literal has a receiver that is a value.** `"abc".ucase()`
  recorded a *bare* `ucase()`, indistinguishable from an unqualified call to a
  function of that name — in a component that declares one, an edge to it that
  does not exist. The component is `$any`, the existing spelling for "genuinely
  dynamic".
- **`throw` is the one keyword invoked like a function**, so it is the one that
  needs its argument list consumed: `throw(type = "x")` declared a variable
  called `type`. `if`, `while` and `switch` hold an expression, which the
  statement scan reads correctly as it is.
- **A component's or interface's attribute list is not a run of assignments**,
  and `parsePlain`'s keyword guard runs *before* its tag-attribute check — both
  names are keywords, so `component extends="models.Base"` put `extends` into
  `VariablesVars`, which is what completion offers and what the index stores.
  Each needs its own arm in both scans.
- **A nested function's body is scanned, not skipped.** An immediately-invoked
  function lost everything it called while the same closure passed as an
  argument kept it, because that path counts parentheses instead. The calls are
  attributed to the enclosing function, which is where the closure runs from.
- **A closure's body is read as statements, and what it declares is its own.**
  It was scanned for calls alone, so every `var`, assignment and parameter in a
  closure declared and typed nothing — most of a TestBox spec, where each test
  is `it( "x", () => { var t = … } )`. `scanClosureBody` runs `handleBodyToken`
  over it inside a function (outside one there is no scope for its locals, so
  the call-only scan stays), for `function(){}` and a block-bodied arrow alike;
  an expression-bodied arrow declares only its parameters, which the argument
  scan remembers from the last `(...)` group (`arrowParams`) because `=>`
  arrives after them. Three things keep the closure's names from leaking into
  the enclosing function, each with a failing case in `closure_test.go`: a ref
  it declares carries the closure's lines (`ComponentRef.VisibleFrom/To`), so
  `FuncComponentRefs` leaves it out and `FuncComponentRefsAt` — which
  `funcScopedRef` reads, innermost closure first — includes it where it is in
  force; `localVarSet` is copied, so its `var t` does not make a later `t = …`
  in the function a local; and its `return` is not the function's return type.
  Its `VarDef`s do count in `FuncVars`, since a declaration list has no lines to
  scope by. `new component { … }` is `$any`: it was read as a component called
  `component`.
- **A *named* nested function is a declaration; an anonymous one is a value.**
  CFML hoists `function setup(){…}` written inside another function into the
  component's variables scope, which is what lets the enclosing function call it
  above the line it is written on. The script parser recorded nothing — no index
  entry, no completion, no go-to-definition — while the tag parser had always
  recorded it, so the same code meant different things in the two syntaxes.
  `skipNestedFunction` files a `FunctionDef` and a `FuncScope` for the named
  case, and `scanNestedCall` dispatches `function` *before* the `isKeyword`
  guard so the rule holds in every one of the six token loops — a helper
  declared inside a TestBox `describe(…, function(){ … })` is the common shape.
  The calls such a function makes stay attributed to the enclosing function, per
  the bullet above; only the declaration is new. A method of an inline
  `new component { … }` becomes a method of the enclosing component, which
  over-declares by one name and matches both the grammar and the tag parser.
  **The call axis could not see this**: the calls were already recorded at the
  right lines under the wrong function, so `make gapcheck` agreed. Asking the
  grammar which *names* it declares found it in one run.
- **A script function may carry attributes after its parameter list**
  (`localmode=true`, `restPath="x"`, a bare `cbMethod`,
  `cachedwithin=createTimeSpan(0,1,0,0)`). Every reader of a declaration
  expected the `{` right after the `)`, so such a function got a scope ending
  on its own line and its body was read as component-level code: 728 functions
  in the corpus, and 794 of their locals reported as variables of the
  component. `skipFunctionAttrs` steps over them in the parse,
  `findScriptFuncScopes` and a named nested function, and the parse reads each
  value as an expression so a call in one is still recorded. It is **only for a
  named declaration**: after an arrow function's parameters comes an
  expression, and consuming an identifier there takes the receiver off a call —
  `TestArrowFunctionBodyKeepsItsReceiver` fails if it moves into `parseBody`.
- **`this.` and `variables.` are separate stores, and a ref says which it was
  assigned through** (`ComponentRef.This`). The lookup had stripped every scope, so
  ColdBox's Injector — `this.SCOPES = new Scopes()` beside a struct `variables.scopes`
  — checked `variables.scopes[ k ].getFromScope()` against `Scopes`, typed
  `getInstance()` as returning one, and so every call on an instance WireBox built,
  29 over the corpus. A receiver qualified with either scope reads its own scope's
  refs, in the resolver (`funcScopedRef`, `fileLevelRef`) and in `baseVarComponent`;
  an unqualified one still reads both, since CFML's own lookup is not what decides
  that here and a narrower rule was not measured. Two things came with it, each with
  a test that fails without it: `x = a[ k ].f()` no longer takes `a` as its base
  (the element is not the variable — a Lucee date had been typed as its entity), and
  `variables.x = this.x = rhs` records a ref in each scope, which TestBox's `$assert`
  relies on.
- **`a = b = rhs` gives both names what `rhs` holds, in either syntax** (`chained.go`).
  Only the scoped script form used to, and `request.calc = calc = new Calculator()` typed
  neither. Each parser handles the inner assignment as the statement it is
  (`chainedAssign`, `tagParser.chainedSet` through `setAssign`), then re-adds what it made
  for the outer name through `addRef` under the outer assignment's own state, so
  `var a = b = new X()` makes a local `a` and a variables-scope `b`. `a = b == c` is a
  comparison and chains nothing. The script check runs on every assignment, so it
  decides from the cached peek and `Scanner.bytesAfterPeek` rather than a save and
  restore, which drops the peek and cost 4.5% of a script parse.
- **A returned variable holds what reaches the return, not its first
  assignment** (`refReaching`), as a receiver does in `funcScopedRef`:
  `var x = new A(); x = new B(); return x;` returns a B, in either syntax. A
  pending call assigning it between that ref and the return defers the answer
  to the pass after calls are typed (`assignedBetween`), and calls left untyped
  are looked at again while returns keep settling or the last pass typed a call
  (`maxReturnRounds`): calls are typed in file order, so one on a receiver
  assigned further down (Lucee's MongoDB tests connect `db` in a later function)
  waited for a return to settle that never would.
  `TestAReturnWaitsForAReceiverTypedLaterInTheFile`.
  `hasRefFor` skips a pending call only for a ref on its own line, and a call on
  a value is never the file's function of that name (`callsOwnFunction`).
  `TestReturnTypeIsTheAssignmentReachingTheReturn`.
- **Branches reaching a return must agree** (`flow.go`). A full parse records
  the block every assignment in a function was made in and the blocks open at
  each return: a brace group in CFScript, the body of `<cfif>`/`<cfelseif>`/
  `<cfelse>`/`<cfloop>`/`<cftry>`/`<cfcatch>`/`<cfcase>`/`<cfdefaultcase>`,
  and the function body. An assignment in a block open at the return always
  ran and replaces what came before; any other may not have and joins it
  (`flowBlocks.reaching`). If what reaches the return disagrees, the
  components join into one union value (`agreedComponent`, `unionComponents`):
  the sorted pipe form `a|b`; `$any` swallows the rest, and an untyped or
  native value in the set keeps the return untyped, since a component
  alternative cannot stand for one (`internal/parser/union_returns_test.go`;
  the chained-call shape end to end is `internal/resolve/
  union_return_test.go`). `<cflock>`, `<cftransaction>` and `<cfoutput>` run
  their body once and are not blocks. A ref notes its block in `addRef`, and a
  pending call is stamped by the body loop (`stampFlow`), so a ref appended
  anywhere else reads as always running, as every ref did before. **A braceless body is a block when it can be bounded**
  (`openBracelessBody`): `var x = new A(); if ( c ) x = new B(); return x;` has
  no type, and so do braceless `else`, `for` and `while` bodies. The body opens a
  block when the whole statement is one line ending in a semicolon with no brace
  in it, found by a lookahead on a saved scanner state at the keyword; it closes
  at that semicolon, or at the first token past it when a statement parser
  consumed it. Anything else (a statement over two lines, or without the
  semicolon CFScript does not require) is still read as an assignment that
  always runs, since bounding it needs the folding pass's newline rule.
  `TestBracelessBodiesAreBlocks` pins both halves. Only the function-body loop
  does this, not a closure body's `handleBodyToken` scan. `TestReturnTypeComparesTheBranchesReachingIt`.
- **A call in `x = x.m()` is made on what x held before the line.** The ref
  the assignment makes carries `ComponentRef.Rebinds`, and the resolver's
  latest-at-or-before rule (`funcScopedRef`, `fileLevelRef`) skips it on its
  own line (`assignedByCallAt`); an assignment on the same line that does not
  rebind (`x = new A(); x.m()`) still reaches the call. The tag parser fixes a
  call's component while parsing (`lookupComponentRef`), from the refs it has
  so far, so it leaves it empty when an assignment only `resolvePendingCalls`
  can type sits between that ref and the call.
  `TestACallInAnAssignmentReadsTheValueBeforeIt`.
- **A function body `FuncRefs` parses on its own is typed as the full parse
  types it** (`typeBodyPendingCalls`). That parse runs for a function the parse
  filed no ref under, and for every function after an edit invalidates it, and
  it resolved no pending calls, so `x = make()` and `variables.y =
  variables.y.next()` added nothing and the variable kept its first type. It
  now types them (`typePendingCalls`, the first pass of `resolvePendingCalls`)
  and settles chained refs (`applyChainedReturnLookup`), reading a receiver
  from the file's refs as they stand; `funcReturnsFor` builds only the return
  types the body's calls name. An unscoped argument or local is not read from
  the component's refs (`pendingCall.baseLocal`), or the answer depended on
  which of the two parses ran first. `TestLazyFuncRefsAreTypedAsTheParseTypesThem`.
- **A word operator followed by a dot is a name** (`operatorWordIsName`): `mod.generate()` is a
  call on a variable called `mod`, which cfwheels' CLI specs hold their module in. `isKeyword`
  turned every token loop away from it and the call was recorded bare. The rule is in each place
  a keyword is refused before a receiver is read; `TestAnOperatorWordCanNameAReceiver`. So is
  `default`: a switch's label is followed by a colon, and ColdBox's CacheBox report page holds its
  cache in a variable of that name (`default.get( "Test" )`).
- **A line break ends a statement where CFScript lets it** (`newlineEndsStatement`): when the
  line's last token could end an expression and the next line starts with a word that is not an
  operator, the folding scan's rule. A collection write's value (`expressionEnds`) and a
  `return` expression (`scriptReturnExpression`, which `TestExpressionEndsMatchScriptReturnExpression`
  holds alike) ran on to the next `;` or `}`, so in ColdBox's semicolon-less Controller
  `services.handlerService = new HandlerService( this )` had the rest of `init()` as its value.
  Two more things kept those members untyped: a `new a.b.c.X()` path was read as a call on the
  record `new a.b.c` (`isNewExpression`), and any closure anywhere withheld every component member
  (`withholdForClosure` now withholds only those its enclosing function writes).
  `TestAComponentFillsAStructFieldWithoutSemicolons`, `TestALineBreakEndsAStatement`.
- **`import models.User;` qualifies a later bare `new User()`.** `import
  models.*;` does not: which component a bare name then means is a question
  about what is on disk, and the parser has no filesystem.
