# Following receiver and return-type opportunities

2026-10-11. Baseline: [collection callback contracts](RESOLUTION-COLLECTION-CALLBACKS.md),
13,293 strict findings and 6,758 assumptions in combined liberal mode.

## Implemented source relationships

### Complete collection expressions

Collection element inference now parses the complete call chain instead of a
regular expression that could consume a chain as one call's argument list.
Previously `holder.getItems().map(...)` and `holder.getItems().missing()` could
inherit `getItems()`'s element type. Regression fixtures demonstrate both false
acceptances. The same problem affected collection fields read from a returned
struct, such as `service.search().missing()` followed by `result.rows`; that
path now requires a single established producer call.

For a collection with an independently established ORM element contract,
`filter`, `slice`, `reverse`, and `duplicate` preserve that contract. The global
`arrayFilter`, `arraySlice`, `arrayReverse`, and `duplicate` forms preserve it
when their builtin names are not shadowed. The existing recursion budget still
bounds propagation. This extends the previous callback rules to direct getter
chains and chained collection operations, including bare `entityLoad(...)`
producers. A callback consuming `map` gets the input element type; the output
of `map` does not inherit that type.

Unknown producers, intervening property reads after calls, unrecognized
operations, and transformed `map` output remain unknown. No homogeneous element
inference from array literals was added. Normal API calls and valid ORM metadata
remain assumptions; no CFML runtime execution is claimed.

### Missing ORM utility API and shared return contract

The repository's existing cborm revision,
`3cd94ac1b256bbd02838fd67b453734c91013822`, contains
[ORMUtilFactory](https://github.com/coldbox-modules/cborm/blob/3cd94ac1b256bbd02838fd67b453734c91013822/models/util/ORMUtilFactory.cfc)
and the shared
[IORMUtil interface](https://github.com/coldbox-modules/cborm/blob/3cd94ac1b256bbd02838fd67b453734c91013822/models/util/support/IORMUtil.cfc).
Neither was selected for generation in `frameworkapi.Sources`.

The factory chooses an implementation according to the engine and documents its
shared interface as the return contract. Both components are now explicit stub
roots. For an interface explicitly selected in `Sources.Extra`, the generator
prefers its documented contract to an inferred concrete factory branch. This
also corrects the generated `getOrm()` return declarations in BaseORMService and
EventHandler. Other undocumented or unselected interface returns retain the
existing generator behavior. The interface name fallback follows the selected
root's unique basename; conflicting selected identities withhold that choice.

The cborm stubs were regenerated from the same pinned source, not hand-written.
A second generation into a separate directory reproduces them byte for byte.
The source checkout and regeneration artifacts remain under ignored `target/`.
Real workspace component files continue to take precedence over framework stubs.

These are public API contracts, not proof of an installed dependency or a
particular runtime engine. Existing interface behavior still accepts extension
methods dynamically without a declaration. The new target regression explicitly
distinguishes that dynamic acceptance from shared methods with API definitions.
No legacy `coldbox.system.orm.hibernate.util.ORMUtilFactory` alias was introduced;
its historical version and deployment relationship still need separate evidence.

## Measured results

The final `complete/` reports compare directly with the preceding
`liberal-implementation/callbacks-final/` reports, using the same seven projects,
scan arguments, and flags.

| Mode | Before | After | Removed | Added | Remaining assumptions |
|---|---:|---:|---:|---:|---:|
| Strict | 13,293 | 13,250 | 43 | 0 | 0 |
| Ranked/native plus open world | 788 | 788 | 0 | 0 | 6,726 |

The 43 strict removals comprise 17 collection relationships (16 ContentBox and
one Lucee) and 26 ORM API calls in ContentBox. The latter are 13 factory calls
and 13 calls to the shared datasource method. Target audits of the actual batch
scan confirm all 43 have a component/API declaration: 17 targets in corpus
checkout files and 26 in the pinned virtual framework stubs. Instrumented scans
produce the same finding multisets as the ordinary binary.

Combined liberal mode loses exactly 32 assumptions and adds none: six missing
receiver relationships and 26 missing-component findings now reach definitions.
The other 11 strict improvements were already accepted under speculation.
Remaining assumption categories are:

| Category | Calls |
|---|---:|
| Runtime receiver | 4,603 |
| Runtime return | 1,252 |
| Runtime function/mixin | 605 |
| External component | 266 |
| Total | 6,726 |

Coverage remains 8,151 indexed/scanned files and 189,154 call records. The exact
46 missing-base summaries, representing 1,133 unchecked inherited calls, remain
unchanged. Retained liberal findings plus assumptions total 7,514; the retained
788 alone is not a claim that all other calls are proven.

Reports, per-entry diffs, stage comparisons, target audits, source hashes,
mutation controls, and validation logs are in
`target/resolution/collection-preservation/`. Earlier `reports/` and `final/`
directories retain the collection-only stages; `complete/` includes the ORM API.

## Review controls and next opportunities

The collection regression failed before the change, both on missing positive
relationships and false acceptance through `map` and broken chains. The returned
struct-field regression failed because a broken chain was accepted. The generator
regression initially selected the concrete branch rather than the selected
interface. Disabling the factory stub makes the API target regression fail.

The major remaining receiver gap is still collection contents and record values,
not operation names. Struct callback values need evidence from their actual
writes and producers, particularly appender/executor/bean registries. The parser's
closed collection-return graph already records compact dependencies for indexed
reads, but a whole collection is deliberately not typed as its values. Reusing
those dependencies needs a separate container/value contract and the existing
conflict, alias, mutation, closure, and invalidation guards. It cannot safely be
implemented by treating every struct as an array of one guessed CFC.

Other pending opportunities are `map` callback output contracts, `keyArray()`
string element propagation, and factory return alternatives with discriminants.
Legacy missing components require version/mapping evidence rather than a
blanket alias to the current cborm API. This batch establishes more definitions
without changing the open-world policies or adding a speculative flag.

Validation completed: `go build ./...`, `go vet ./...`, full
`go test -short` and `go test -short -race` suites, the pinned `make lint`
(0 issues), and `git diff --check`. Full suites used approved local Unix socket
access for daemon tests. Stub regeneration is reproducible and the measured
production source/stub hashes remain unchanged. No published CI result or CFML
engine execution is claimed. Scan coverage is unchanged; API metadata coverage
adds the two generated ORM utility components described above.

## Collection key and value follow-up

See [collection keys and returned struct values](RESOLUTION-COLLECTION-VALUES.md)
for delivered key-array elements, retained dynamic-scope origins, separate struct
value contracts, edit invalidation, current measurements and remaining boundaries.
