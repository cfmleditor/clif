# Collection closure ownership: bounded implementation

The collection graph previously invalidated every collection in a file when it
encountered any anonymous function or arrow closure. It now invalidates shared
fields and locals owned by the closure's enclosing named function. A closure in
another named function, or at component top level, no longer invalidates an
independent function-local collection.

This follows a limited ownership assumption: separate, non-nested named functions
cannot capture each other's locals. Overlapping named-function scopes retain the
old file-wide veto: nested functions can capture their enclosing locals, and
same-line function scopes are ambiguous in this graph. Closure bodies still lack their own collection
graph identities, so same-function captures and shadowing remain conservative.
Every component field remains invalidated, even if the closure appears unrelated.
Whole aliases propagate that invalidation back to exposed local collections.
The shared write scanner and member-binding closure rules are unchanged.

Both indexed returns and whole-struct value contracts use this boundary. Tests
cover unrelated anonymous and arrow closures, equal local names in separate
functions, top-level closures, same-function captures and shadowing, and local
collections exposed through component fields. Resolver integration checks actual
item methods, missing-method diagnostics, and struct callback values.

A nested named-function capture regression fails when only the overlap guard is
disabled: mutation inside the nested closure incorrectly leaves the enclosing
local's indexed return typed as the item. It passes with the guard restored.

The positive parser cases fail without the change. Disabling only the new
invalidation rule also fails the resolver integration: callback values lose their
component identity and a missing method on an assigned selected item is accepted.
Restoring the rule resolves the known callback method and diagnoses that missing
method. This is evidence of fixture correctness, not a claimed corpus reduction.

The seven-project comparison against the returned-struct-values checkpoint is
unchanged per finding in all fourteen strict/liberal scans: zero removed and zero
added. Strict retains **13,249**; liberal retains **788** plus **6,725** assumptions.
Assumptions are also identical per entry. Coverage remains **8,151** indexed/scanned
files and **189,154** CLI accepted/reported/assumed records. Missing-base summaries
are unchanged per entry, representing **1,133** unchecked calls. This is a CLI
coverage comparison, not an instrumented raw parser-record comparison.

This change has no measured corpus gain. It removes a demonstrated ownership
defect for independent local collections but does not solve the shared registry
cohort. Reducing that cohort requires closure-local/captured identities, complete
writer and alias ownership, and evidence for unknown or heterogeneous values.
Dropping the shared-field veto would conceal those remaining proof gaps.

Validation and same-coverage corpus artifacts are recorded under
`target/resolution/collection-closures/`: counterfactual `disabled.log`, complete
per-entry reports/diffs in `checked/`, assumption and coverage audit `review.json`,
and final build, vet, lint and test logs. No CFML engine execution or published-head
CI claim is made.

Final-source `go build ./...`, `go vet ./...`, full short tests, full short race
tests, pinned `make lint` and `git diff --check` pass. Full tests used approved
local socket access, `GOMAXPROCS=2` and `-p 1`, after corpus scans completed.
Source hashes in the audit match the files used for the final corpus comparison.
