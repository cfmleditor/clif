# CFML native member follow-up

This batch fills documented native APIs after reviewing the remaining failures.
It changes the shared public API profile, not engine/version validation. A
known native receiver remains required in strict mode; liberal family selection
continues to be an opt-in assumption.

## Implemented return contracts

| Family | Methods | Result |
|---|---|---|
| string | hash, listGetDuplicates, listMap, listFilter, map, filter | native string |
| string | listSome, listEvery, some, every | native boolean |
| string | each, listEach | native void |
| string | reduce, listReduce | erased $any |
| struct | toSorted | native struct |
| struct | toJson | native string |
| query | columnData | native array; elements unknown |

These 17 methods are supported by Lucee's primary documentation:
[string hash](https://docs.lucee.org/reference/objects/string/hash.html),
[listGetDuplicates](https://docs.lucee.org/reference/objects/string/listgetduplicates.html),
[listMap](https://docs.lucee.org/reference/objects/string/listmap.html),
[listFilter](https://docs.lucee.org/reference/objects/string/listfilter.html),
[listSome](https://docs.lucee.org/reference/objects/string/listsome.html),
[ListEvery](https://docs.lucee.org/reference/functions/listevery.html),
[string map](https://docs.lucee.org/reference/objects/string/map.html),
[string filter](https://docs.lucee.org/reference/objects/string/filter.html),
[string every](https://docs.lucee.org/reference/objects/string/every.html),
[string some](https://docs.lucee.org/reference/objects/string/some.html),
[string each](https://docs.lucee.org/reference/objects/string/each.html),
[listEach](https://docs.lucee.org/reference/objects/string/listeach.html),
[string reduce](https://docs.lucee.org/reference/objects/string/reduce.html),
[listReduce](https://docs.lucee.org/reference/objects/string/listreduce.html),
[struct toSorted](https://docs.lucee.org/reference/objects/struct/tosorted.html),
[any toJson](https://docs.lucee.org/reference/objects/any/tojson.html), and
[query columnData](https://docs.lucee.org/reference/objects/query/columndata.html).
The corpus's listEvery.cfc independently demonstrates the member spelling.

Character operations and delimited-list operations have distinct callback
layouts. This batch establishes their public methods and results only; it does
not add callback argument or reducer accumulator inference. Void results cannot
supply a subsequent length method. Custom CFC methods with these names do not
inherit native result contracts. Query methods are not admitted on strings.

The documented minimum versions matter: character each/map/filter/every/reduce
were introduced in Lucee 6, some in 6.0.0.126, toSorted in 6.2.1.6, and
listGetDuplicates in 7.0.0.107. These are public API availability assumptions
of the existing shared profile, not verified support on every engine/version.

## Producer identity gaps uncovered during review

The fourteen LDEV3102 columnData findings are not missing string APIs. Each
receiver is initialized as an empty string, then overwritten by a script query
name output. Native literal fallback still reads the initializer. The new
query.columnData contract does not establish that missing query origin.

ColdBox SchemaInfo.cfc has the analogous cfdbinfo(name="qResults") form.
Its filter/reduce calls are legal query operations, but native fallback reads
qResults' earlier empty-string assignment. Adding the legal string operations
can remove these three diagnostics without correcting the producer identity.
Treat those removals as overlapping API symptoms, not completed query-output
relationships. Callback row ownership and query result preservation still need
the actual output producer.

The next flow work should record tag output writes in all three forms: script
attribute syntax, parenthesized cf-tag calls, and tag markup. Static name
outputs must be distinguished from result metadata. Dynamic output names and
attributeCollection require conservative invalidation unless independently
resolved. A completed output must precede the lookup, later assignments must
win, and branches, exception paths and closure ownership must not let a query
contract leak to paths that retain the initializer. Start with regressions for
straight-line query/cfdbinfo writes, then conditional outputs, failed query
catch paths, repeated names in sibling closures, custom cfdbinfo shadowing,
unknown names, and explicit reassignments. Do not infer query from columnData.

Other reviewed failures need their own interpretation. ColdBox's abstract
provider stats field has a string sentinel replaced by subclass setup; that
requires field lifetime/subclass proof. Mura's servlet calls sit behind
isDefined checks while the local servlet remains a string initializer in the
reviewed function, so they are not evidence of an omitted string API. Binder's
currentMapping is iterated as a collection in adjacent methods; setDelegates
on the array itself must not be whitelisted by its spelling. LDEV332 deliberately
exercises numeric coercion and a failing non-numeric string round call, so a
blanket string.round contract would hide its negative case.

## Validation and artifacts

Native table and resolver regressions failed before the changes. The five
initial toSorted/toJson additions exposed another real missing public API;
the toJson regression also failed before adding that contract. A disabled
production overlay rechecks the final tests without this batch. Fixtures cover
result chains, invalid methods, void chaining and custom CFC ownership.
Artifacts and fourteen per-entry corpus comparisons are in
`target/resolution/cfml-native-members/`, against the prior array-list-contracts
checkpoint. Final measurements and validation are appended after review.

## Final measurement

| Mode | Before | After | Removed | Added |
|---|---:|---:|---:|---:|
| Strict | 13,051 | 12,995 | 56 | 0 |
| Liberal retained findings | 665 | 610 | 55 | 0 |

Lucee removes 53 strict entries; ColdBox removes three. The other five projects
have no strict per-entry changes. The ColdBox caveat above applies to all three
of its removals. Lucee removals include eleven listGetDuplicates, nine hash,
five toSorted, and the remaining documented callback/member operations. One
additional toJson finding is removed beyond the initial 16-method probe.

Liberal assumptions fall from 6,738 to **6,720**: **26 removed, eight added**
per entry. The eight additions are every/some runtime-receiver assumptions.
Those method names now overlap array and string APIs and cannot justify the
previous unique primitive array guess. The reviewed source includes actual
struct and query receivers as well, illustrating why method-only receiver
selection is weak. Removed assumptions include seven each of toJson, keyList,
and valueArray that now follow toSorted's known struct result. Some removals
still depend on an upstream speculative guess; they are not independent new
strict producer proofs. Policy origin needs to remain visible when interpreting
these downstream acceptances.

Combined liberal retained-plus-assumed records fall from **7,403 to 7,330**.
Coverage remains **8,151 files**, **189,154 CLI counting records**, and the same
46 missing-base summaries representing **1,133 unchecked calls**, compared
per entry. This is CLI accounting, not a new raw-parser call-coverage probe.
All fourteen comparisons pass their file/counting coverage checks. Production
hashes match the final scanned build; the disabled overlay makes the final
positive tests fail without the new contracts.

The fourteen LDEV3102 string columnData findings remain in the final report;
`remaining-query-output.json` retains their exact entries. Query output
identity was researched and planned, not implemented in this batch. Four
known queryNew every/some findings also remain; the next catalogue review
should verify those query contracts separately from method-only guesses.

Build, vet, focused native/resolver tests, full short/race suites, lint and
git diff --check pass. Logs are in the artifact directory. No CFML engine runtime
or published CI validation is claimed.
