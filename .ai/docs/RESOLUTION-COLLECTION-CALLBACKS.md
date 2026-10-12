# Collection callback invocation contracts

Follow-up to [the liberal implementation](RESOLUTION-LIBERAL-IMPLEMENTATION.md),
2026-10-11. This change improves source resolution without adding another liberal
flag or changing the open-world classification policies.

## Implemented relationships

An ORM collection's independently established element type now supplies the
first callback parameter for member `each`, `filter`, `map`, `some`, and `every`,
and their unshadowed global `arrayEach`, `arrayFilter`, `arrayMap`, `arraySome`,
and `arrayEvery` forms. Mapping does **not** imply that the output collection
has the same element type: this rule describes the callback's input only.

Native collections have distinct callback layouts:

| Collection | Parameter 1 | Parameter 2 | Parameter 3 |
|---|---|---|---|
| Array | Independently proven element, otherwise unknown | Numeric index | Array |
| Struct | String key | Unknown value | Struct |
| Query | Struct row | Numeric row index | Query |

The corresponding global struct/query builtins impose these input layouts.
Member syntax requires an established native receiver; intermediate hops must
resolve successfully before a callback layout can be applied. The cached CFDocs
sources `docs/src/cfdocs/{array,struct,query}{each,filter,map,some,every}.json`
state these invocation signatures and their member syntax. Those cached files
are research inputs, not new runtime dependencies.

## Assumptions and limits

- The native API behaves according to its documented callback signature. No
  runtime engine execution was performed during this change.
- A global builtin contract is withheld when a function of that name is declared
  locally or found through the current component's function resolution.
- Callback bodies use the existing byte-span ownership rules. Reassignment or
  escape before a receiver use withholds its entry type.
- Unknown array elements and struct values remain unknown. Literal arrays do not
  acquire homogeneous CFC element types from these rules.
- Only positional inline callbacks are supported here. Named argument layouts,
  callback aliases, `reduce`'s accumulator layout, and propagated element types
  through collection chains remain separate work.
- A receiver in the speculative receiver map is withheld from native callback
  layout inference. A guessed array could actually be a struct, whose second
  callback argument is a CFC value rather than an index. Independent ORM element
  evidence is still considered before this guard.
- The guard is conservative for mapped receivers. It does not provide complete
  provenance tracking through all indirect assignments and producer paths.

## Review and negative controls

`TestCollectionCallbackInvocationContracts` checks ORM element methods, absent
methods, native layouts, intermediate native hops, broken hops, erased values,
parameter mutation, and a custom CFC provider named `filter`.
`TestGlobalCollectionCallbackRejectsShadowedAndReassignedParameters` protects
builtin shadowing and unknown input elements.
`TestSpeculativeCollectionDoesNotEstablishCallbackContract` protects the
array-versus-struct distinction when liberal receiver guesses are enabled.

The member and global positive regressions failed before their respective fixes.
An overlay disabling only the speculative guard makes its regression fail with
`ready` incorrectly reported as missing on `native.numeric`.

An initial corpus experiment exposed additional liberal findings after guessed
collection families supplied incorrect callback layouts. The guard was added
before the final comparison. This is a reason to review additions per entry,
even when strict totals improve.

Artifacts, binaries, per-entry diffs and validation logs are under
`target/resolution/liberal-implementation/`; final reports use `callbacks-final/`
and compare against the previous turn's `final/` reports with identical scan
arguments. Both strict and combined ranked/native/open-world scans are measured.

## Next opportunities

The immediate missing connections are collection element preservation through
`filter`/`slice` chains, callback output types for `map`, and heterogeneous
struct value types (for example Logger's appender map). The latter needs record
ownership and value propagation, not an array index assumption. A `keyArray()`
result has string elements, but its current native array identity does not carry
that element contract into a following callback. The RequestScope example from
the prior review therefore remains outstanding. Runtime-native Java collection
methods also need engine-specific evidence before accepting them on CF arrays.

These changes do not claim a large reduction in the remaining unknowns. The
small gain shows that established ORM metadata covers only a narrow part of the
callback population. Expanding method-name heuristics would not establish the
missing collection contents or their construction paths.

## Final measured results

| Mode | Previous retained findings | Current retained findings | Current assumptions | Removed / added retained findings |
|---|---:|---:|---:|---:|
| Strict | 13,300 | 13,293 | 0 | 7 / 0 |
| Ranked/native plus open world | 788 | 788 | 6,758 | 0 / 0 |

The strict removals are all ContentBox: five global `arrayMap` calls on the
`linkedContent`, `children`, `relatedContent`, and `categories` ORM relationships,
plus ModuleService's `aModules.filter` and Site's category filter. Individual
`explain` runs confirm all seven reach their component's methods without a
speculative receiver. Audit output is in `callbacks-target-audits.json`.

The liberal mode previously held 6,764 assumptions. Its assumption multiset loses
exactly six entries and adds none; its accepted-call count increases by six. One
of the seven strict improvements was already accepted under speculation. The
retained total under 1,000 still depends on thousands of assumptions, not verified
resolution of every original finding.

Both modes index and scan the same 8,151 files and account for 189,154 call
records. They retain the exact 46 missing-base summary diagnostics representing
1,133 unchecked inherited calls. No coverage difference is used to lower totals.
The seven-project per-entry reports and `callbacks-final-review.json` record the
comparison and source hashes.

Validation completed: `go build ./...`, `go vet ./...`, the full short suite,
the full short race suite, the pinned `make lint` (0 issues), and
`git diff --check`. Full suites used approved execution with local Unix socket
access for daemon tests. Disabling the collection contract makes both positive
regressions fail; disabling only the speculative guard makes its negative
control fail. Final measured source hashes remain unchanged. No published CI
or CFML runtime execution is claimed.

## Receiver and return opportunity follow-up

See [implemented collection and ORM API opportunities](RESOLUTION-OPPORTUNITIES-IMPLEMENTED.md)
for complete-chain inference, preserving operations, generated shared ORM
contracts, target audits, and the latest corpus counts.

## Collection key and value follow-up

See [collection keys and returned struct values](RESOLUTION-COLLECTION-VALUES.md)
for delivered key-array elements, retained dynamic-scope origins, separate struct
value contracts, edit invalidation, current measurements and remaining boundaries.
