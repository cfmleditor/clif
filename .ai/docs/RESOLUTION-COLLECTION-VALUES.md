# Collection keys and returned struct values

Implementation review: 2026-10-11. This follows
[the preceding collection and ORM batch](RESOLUTION-OPPORTUNITIES-IMPLEMENTED.md).
The larger resolution programme remains incomplete.

## Implemented relationships

`keyArray()` on a proven native struct and an unshadowed global
`structKeyArray(input)` establish string elements. Existing array-preserving
operations carry those elements to loops and callback inputs. Direct CFML scopes
such as `request` can establish the native struct producer when ordinary source
lookup has no concrete answer. Concrete source/configuration answers keep
priority. A speculative receiver family cannot establish this contract.

The script parser now retains producer hops on calls chained from dynamic scopes.
It previously recorded `request.keyArray().filter(...)` as a dynamic filter call
without its `keyArray()` origin. Dynamic call acceptance remains in place; the
retained origin supplies the collection callback relationship. Chained scope
calls also retain their own source line through the ordinary continuation path.

Whole returned structs now carry separate, compact `CollectionValues` metadata.
The existing collection graph supplies value producers from observed static
writes, whole aliases and grounded dependencies. A struct remains a container;
it does not acquire its values' CFC type. Closed-file indexing retains producer
contracts, and resolution follows current indexed method definitions under the
existing source/depth budgets. Every producer must resolve to the same component.

This metadata supplies the second parameter of struct callbacks and the elements
of member `valueArray()` and unshadowed global `structValueArray(...)` results.
Iteration over a proven returned struct supplies string keys. Array callback
inputs continue to use the separate array element contract. Transformed `map()`
output does not inherit its input contract.

Metadata compaction owns its source strings and method slices. Incremental edits
invalidate whole-struct contracts throughout the file because an edited writer can
change another getter's values. A full parse rebuilds them. An indexed producer
return edit also changes deferred resolution immediately. A Wheels Mapper
function rebound to return the receiving component clears collection metadata.

## Assumptions and withholding boundaries

These are static source/API contracts, conditional on valid runtime invocations.
They do not prove that a registry was populated on a particular execution path,
that a particular key exists, or that unobserved code cannot modify returned
storage. Existing constructor and declared-return conventions still apply.
Runtime code installation, reflection and external mutation need separate flow
and lifetime evidence.

The new whole-struct rule requires an observed empty struct seed and known value
writes. Arrays do not receive this struct contract. Unknown writes, conflicting
producer results, unsupported replacement, argument-supplied collections,
computed scope/parent replacement and exhausted budgets withhold value types.
Explicit non-struct return declarations take priority. A source-local
`structNew` declaration withholds that initializer's new struct seed.

Whole arguments passed to calls, nested escapes and exposed public aliases veto
the new metadata. Alias escapes propagate through shared storage. Interpolated
expressions are checked with bounded nesting. Member invocations on a source
collection are conservatively treated as possible mutation. This extra escape
veto does not broaden or replace historical indexed-getter inference.

At this checkpoint, files containing script closures retained the collection
graph's file-wide veto. The [closure ownership follow-up](RESOLUTION-COLLECTION-CLOSURES.md)
narrows it for independent named-function locals; shared fields and same-function
locals still withhold inference. Tag-region files withhold the new whole-struct metadata. Consumer locals
must have an untouched assignment reaching the use. Intervening uses, aliases,
escapes, element writes and unsupported same-line operations withhold their value
contract. Earlier read-only uses are deliberately conservative until their
lifetimes are represented. This prevents a populated getter from incorrectly
typing a caller's subsequently modified copy.

Custom methods named `keyArray`, shadowed global builtins, missing input to
`structKeyArray`, broken intermediate calls, unknown/guessed receivers and
reassigned callback parameters do not gain string element proof.

The public API basis is the documented array-of-keys behavior of
[Lucee StructKeyArray](https://docs.lucee.org/reference/functions/structkeyarray.html)
and [Adobe StructKeyArray](https://guides.adobe.com/coldfusion/en/docs/coldfusion-tools/__references__/structkeyarray.html).
Struct value arrays follow
[Lucee StructValueArray](https://docs.lucee.org/reference/functions/structvaluearray.html)
and [Adobe StructValueArray](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/__references__/structvaluearray.html).
Lucee documents introduction in 5.3.8.117; Adobe documents ColdFusion 2025.
As with the existing native API profile, this contract does not verify installed
engine versions. No CFML engine execution is claimed.

## Corpus result

The same seven roots, configurations and strict/liberal flags are compared per
entry with `collection-preservation/complete/`.

| Mode | Before | After | Removed | Added | Remaining assumptions |
|---|---:|---:|---:|---:|---:|
| Strict | 13,250 | 13,249 | 1 | 0 | 0 |
| Ranked/native plus open world | 788 | 788 | 0 | 0 | 6,725 |

The sole strict removal is ColdBox `RequestScope.clear()`'s
`arguments.key.findNoCase("wirebox:")` inside its request-key filter. The actual
scan target audit reaches `builtin`, component `$builtin.native.string`, with no
reason. This is native API type evidence, not a CFC definition or a speculative
connection. Exactly that runtime-receiver assumption is removed in liberal mode.

Whole struct value propagation adds no verified corpus reductions in this batch.
Its positive source relationships and rejection boundaries are covered by focused
regressions. The remaining registries require more than recognizing collection
operation names: their values have unknown writers or mixed types, and closure
ownership, selection and lifetime information is incomplete. The file-wide closure
veto was a specific obstacle, now narrowed for independent function locals in the
[follow-up](RESOLUTION-COLLECTION-CLOSURES.md); parser callback ownership improvements do
not automatically close the collection graph's separate ownership analysis.

Coverage remains 8,151 indexed/scanned files and 189,154 accepted/reported/assumed
call records. These are CLI counting records, not a claim of identical raw parser
metadata: scope chain origins intentionally change. The exact 46 missing-base
summaries and their 1,133 unchecked inherited calls are unchanged.

Retained liberal findings plus assumptions total 7,513. The assumption categories
are runtime receiver 4,602, runtime return 1,252, runtime function/mixin 605, and
external component 266. The retained 788 alone does not mean all other calls have
proven relationships.

Artifacts under `target/resolution/collection-values/` retain before/after
binaries, preliminary and final reports, per-entry differences, counterfactual
failures, target audits and validation logs. `checked/` is the final-source scan;
`preliminary/` and `final/` retain earlier stages.

## Verification and next implementation work

The key-array positives failed before implementation. Disabling metadata emission
and restoring the old dynamic-scope path fails the parser and resolver positives.
The incremental writer-edit regression separately fails before invalidation.
Regressions cover wrong container types, unknown writes, conflicting returns,
mutators, escaping aliases, nested/interpolated escapes, public exposure,
consumer mutations, custom methods, shadowing, broken chains and producer edits.

Further work remains on collection graph closure identities, heterogeneous values
and common contracts, runtime registry writer/escape lifetimes, map callback output
contracts, factory discriminants, and version-specific dependency relationships.
The new bounded value metadata is a foundation for that work, not closure of all
registry findings.

Validation completed: `go build ./...`, `go vet ./...`, full
`go test -short` and `go test -short -race` suites, pinned `make lint`, and
`git diff --check`. Full suites use approved local Unix socket access for daemon
tests. The final inherited-initializer rejection control also passes focused
short and race checks. Final corpus production hashes remain unchanged; the last
review adds only that test control and formatting. No published CI result is
claimed.
