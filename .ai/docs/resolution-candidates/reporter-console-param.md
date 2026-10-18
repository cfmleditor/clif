# The reporter lazy-cache failure: isolated and fixed

## The failure

TestBox `system/reports/BaseReporter.cfc` reported eight entries:

```
BaseReporter.cfc:36/43/50/57/64/79/98/102  method 'getConsoleUtil' has no
component return type (chain to 'color'/'getStatusIndicator'/…)
```

`getConsoleUtil()` is TestBox's guarded lazy field: it builds
`variables.consoleUtils = new testbox.system.util.ConsoleUtil()` under
`!structKeyExists( variables, "consoleUtils" )` and returns it. Every
`getConsoleUtil().color()` style chain in the reporter family hung off that
return being untyped, so the `getConsoleUtil` members could not be connected.

Isolated by bisection to one unrelated function in the same component:

```
function prepareIncomingParams(){
    param url = {};
    …
}
```

Minimal reproducer (before/after:

```
system/reports/{WithParam,WithoutParam}.cfc
```

in the scratchpad run) — the identical component with a single
`param url = {};` in a sibling method loses the contract; without it, the
component identity `testbox.system.util.consoleutil` is computed.

## Root cause

`producerAssignment` (internal/resolve/producer_source.go) read
`param url = {};` as a plain assignment. `producerPath` rejects the
two-ident `param url` head, so the statement degraded to kind `unsafe`
with an empty expression. `sharedFieldContract` (internal/resolve/
producer_flow.go) invokes `producerMayTouchField("", path)` for such a
node, and an empty token slice conservatively "may touch" every field —
so any unreadable statement anywhere in the component invalidated every
guarded shared field's all-writes-agree proof. TestBox's whole
reporter family lost the return typing because of one `param` the plan
had no way to read.

## The fix

Remove the two-ident head (`param` + a bare variable path, optional
`= value`) from opaque-unsafe classification:

```go
if len(tokens) > 1 && strings.EqualFold(tokens[0].Value, "param") && tokens[1].Kind == parser.TokIdent {
    …
}
if target := producerPath(tokens[1:max(2, at)]); target != "" {
    return producerNode{kind: "param", target: target}
}
```

after which `param` degrades to `unsafe` or a plain assignment, matching
the conversative treatment of other statement forms. `param` is a
keyword in CFML/Lucee/ACF so the intent that a bare `param x …` in a
function body has the tag-form (cfparam) semantics — a presence-default
for `x` alone — applies; anything else (e.g. `param.x = 1`, a call
`param(1)`) falls through to the original shape because
`tokens[1].Kind == TokIdent` and `producerPath` reject.

## Verification

- New regression: `TestAParamStatementDoesNotInvalidateAComponentContract`
  (internal/resolve/param_statement_test.go) with four statement shapes
  (default, no default, dotted target, two params); fails before the
  change with the corpus reason reproduces, passes after.
- Corpus per-entry (fresh binaries: pre-fix `clif-new` vs post-fix
  `clif-fix`, `unresolved --json` sorted by file/line/function/reason):
  - TestBox 388 -> 380 scoped per call: 8 removed (the getConsoleUtil
    chains listed above), 0 added.
  - ContentBox, ColdBox, fw1, Mura, cfwheels, Lucee: 0 removed / 0 added
    (byte-identical sorted unions).
- `go build ./...`, `go vet ./...`, `go test -short ./...`,
  `go test -short -race ./internal/resolve/`, `golangci-lint run
  ./internal/resolve/` (0 issues), `gofmt -l` (empty), `git diff
  --check` all clean.

## Not the same as the other reporter-include item

`RESOLUTION-IMPLEMENTATION-STATUS.md` line 728 ("The reporter include
corpus failure has not been isolated by the passing minimal fixture")
refers to the *include* case (the `include "assets/*.cfm"` families
inside the reporters) and remains open. Current full-scan of TestBox
leaves only three entries in the reporter asset templates, none of them
about the include boundary:

```
assets/codexwiki.cfm:64  local.thisSpec.failOrigin.toString
assets/doc.cfm:96        local.thisSpec.failOrigin.toString
assets/simple.cfm:261    thisBundle.keyExists
```

These are unknown return shapes (`failOrigin` as a struct-typed member
of a loop index the resolver typed as a struct of unknown members) and
one `keyExists` member call, not include-scope transfers. They belong
to the unknown-returns gap family, not the reporter-include
isolation item.
