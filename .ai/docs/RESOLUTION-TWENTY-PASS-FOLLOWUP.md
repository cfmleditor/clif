# Twenty-pass resolution follow-up — 2026-10-11

This batch follows the 12,539 strict / 568 liberal checkpoint. The twenty
passes below are focused audits, implementations, and reviews, not twenty
independent code changes. Existing corpus configurations and policy flags remain
unchanged. Measurement artifacts are under
`target/resolution/twenty-pass-followup/`.

## Pass ledger

| Pass | Investigation and evidence | Outcome |
|---:|---|---|
| 1 | Recounted every baseline entry by diagnostic reason. Variable failures account for 9,924 of 12,539 entries. | Most outstanding work needs producer/scope relationships; API-name expansion alone cannot solve it. |
| 2 | Extracted 207 known-native missing-method entries with receiver identities across the corpus (`native-missing.json`). | Separated absent APIs from incorrectly selected receiver types; chose bounded source-backed additions. |
| 3 | Checked Lucee's literal function-level modern local mode against remaining InternalRequest cases. | Implemented current named-function `localmode=true` or quoted modern/true; dynamic, bare modern identifiers, expressions, duplicates and classic values stay unproven. |
| 4 | Reviewed repeated bare result assignments after explicit local initializers. | Implemented same-block dominating local stores; closed branches, sibling callbacks and unbraced conditional stores do not qualify. |
| 5 | Reviewed how a previously established local key can disappear. | Added whole-local-scope escape/operation veto. `structClear(local)`, `local.delete(...)` and passing local to another call invalidate prior-store proof. Negative regressions fail without this guard. |
| 6 | Reviewed named capture scanning with attributed function headers. | Fixed body discovery beyond parameter lists and attributes. A nested attributed writer still blocks result field inference. |
| 7 | Rechecked callback ownership, qualifiers, engine gating and local lifetimes. | Parent modern mode does not establish callback mode; variables/shared scopes cannot borrow a bare-local proof. Existing capture/mutation/escape guards remain. |
| 8 | Inspected MakeRequest wrappers in LDEV0590 and source producer propagation. | Deferred returned result-field propagation: direct builtin proof does not establish a wrapper's returned struct fields or alias safety. |
| 9 | Inspected cfhttp/new Http result usage, including binary-to-string LDEV1976. | Deferred a separate HTTP result shape/content-mode rule; InternalRequest text-field spelling alone is insufficient. |
| 10 | Inspected semicolon-free producer prefixes and branch-contained request calls. | Kept explicit statement boundaries and normal-path guards. Grammar-aware completion/dominance is required before expanding these cases. |
| 11 | Verified struct some/every and callback argument layouts. | Added boolean results, tested string keys/struct collections, retained erased callback values and ambiguous native-family guesses. |
| 12 | Verified string repeatString and numeric/string numberFormat. | Added string result contracts and chained length tests; generic Object is not given a numberFormat method. |
| 13 | Checked all seven Lucee date setter descriptors and Java implementations. | Added date results for setDay/Hour/MilliSecond/Minute/Month/Second/Year. Did not extrapolate to unrelated setters such as setDate. |
| 14 | Checked removeDuplicates and array getMetadata implementation. | Added array and struct result interfaces respectively. No new element preservation rule is claimed for deduplication. |
| 15 | Followed PageContext → PageSource → Mapping in public loader source and inspected initial corpus deltas. | Added bounded identities and Mapping → getPageSource round trip. Initial ten changed diagnoses exposed the missing second half of this chain and prompted its completion. Resource-returning methods are deferred. |
| 16 | Checked Java Thread/ThreadGroup public contracts. | Added group retrieval, parent/name and bounded scalar methods. Nullability/runtime existence is not promised. |
| 17 | Followed Lucee connection pools to factories and datasource instances in source. | Corrected getFactory to a distinct Lucee factory identity, then getDatasource to Lucee datasource, then getName to string. Generic datasource remains without getName. |
| 18 | Reviewed false identities and ORM extensions. | Recorded ordered square-bracket structs misclassified as arrays (LDEV4725), assignment overwrite ordering (FW1 pathInfo.split), and version-specific SessionFactory.getDialect. No array keyList/split or generic Hibernate dialect shortcut was added. |
| 19 | Compared all seven corpus entries in strict and liberal modes, including the explicit Wheels vendor root. | Per-entry findings, assumptions, connections and coverage checked. Initial liberal deltas exposed extension receivers guessed as arrays from getMetadata alone; added a discrimination guard and a failing counterfactual regression. Also withheld numberFormat-only string guesses because numeric receivers support the same API. Rebuilt and repeated the final comparison. |
| 20 | Ran positive counterfactuals, negative guard counterfactuals, targeted regressions and required repository checks. | Final verification and limitations below. |

## Verified contracts and assumptions

Lucee's [local mode guide](https://docs.lucee.org/recipes/migrate.html)
explains modern versus classic local destinations. Application/server defaults
are not inferred by the new rule. The prior-local rule only accepts a matching
lexical owner, with no whole-scope operation/escape. It is conservative about
other uses of the local scope; it is not a general CFG or dynamic-evaluation model.
Result typing still requires a direct unshadowed engine builtin and a bounded
unescaped lifetime. Presence of filecontent is not guaranteed.

The [Lucee function library](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/resource/fld/core-base.fld)
provides the member registrations and result interfaces. See also
[struct.some](https://docs.lucee.org/reference/objects/struct/some.html),
[struct.every](https://docs.lucee.org/reference/objects/struct/every.html),
[repeatString](https://docs.lucee.org/reference/functions/repeatstring.html),
[numberFormat](https://docs.lucee.org/reference/objects/any/numberformat.html),
and [array.removeDuplicates](https://docs.lucee.org/reference/objects/array/removeduplicates.html).
Date setters return their modified DateTime, for example
[SetYear](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/functions/dateTime/SetYear.java).
[GetMetaData](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/functions/other/GetMetaData.java)
dispatches arrays to
[ArrayUtil.getMetaData](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/type/util/ArrayUtil.java),
which returns a Struct. That result is not generalized to every getMetadata receiver.

The engine-specific chains follow the public
[PageContext](https://github.com/lucee/Lucee/blob/7.1/loader/src/main/java/lucee/runtime/PageContext.java),
[PageSource](https://github.com/lucee/Lucee/blob/7.1/loader/src/main/java/lucee/runtime/PageSource.java),
[Mapping](https://github.com/lucee/Lucee/blob/7.1/loader/src/main/java/lucee/runtime/Mapping.java),
[DatasourceConnPool](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/config/DatasourceConnPool.java),
[DatasourceConnectionFactory](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/db/DatasourceConnectionFactory.java),
and [DataSource](https://github.com/lucee/Lucee/blob/7.1/loader/src/main/java/lucee/runtime/db/DataSource.java)
interfaces. The pool factory is not itself a datasource. These chains originate
from the existing Lucee-gated page context contract.

The JDK group chain follows
[Thread](https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/lang/Thread.html#getThreadGroup())
and [ThreadGroup](https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/lang/ThreadGroup.html).
As with existing native contracts, generic CFML member tables describe interfaces
and do not verify method availability for every engine/version. Engine-specific
origins remain gated. No new blanket guessing flag or broad receiver-name mapping
was introduced. Liberal native guessing remains explicitly speculative; overlapping
families can withdraw guesses when the API table becomes more complete. Metadata
plus common Object methods cannot select a native family: unrelated extension,
JDBC and collection objects expose that API. A known array still supplies its
verified metadata result type. Numeric-compatible API sets likewise cannot
select string solely because numeric is outside the existing primitive guessing
family set; formatting does not distinguish those receivers.

## Next relationship opportunities

Ordered bracket structs need a parser distinction between `[D=1,B=1]` and an
array literal, with nested/empty/indexed-expression tests and separate element
versus value contracts. FW1's pathInfo is used as a string before being replaced
by its split result: lookup must use the reaching assignment before the call,
not its later array state. Neither issue is fixed by giving arrays struct/string
methods. Source wrappers and HTTP outputs need independent producer/result-field
contracts. Hibernate dialect access needs a verified implementation/version
identity beyond the standard SessionFactory interface.

## Final measurement and verification

| Entry | Strict before | Strict after | Removed | Added |
|---|---:|---:|---:|---:|
| Wheels, including explicit vendor root | 2,570 | 2,570 | 0 | 0 |
| Masa | 3,687 | 3,687 | 0 | 0 |
| ContentBox | 1,754 | 1,754 | 0 | 0 |
| ColdBox | 1,637 | 1,637 | 0 | 0 |
| Lucee | 2,136 | 2,065 | 71 | 0 |
| TestBox | 396 | 396 | 0 | 0 |
| FW/1 | 359 | 359 | 0 | 0 |

Strict findings: **12,539 → 12,468**,
71 removed and 0 added.
Liberal ranked/native plus open-world findings: **568 → 512**,
56 removed and 0 added. Runtime assumptions:
**6,936 → 6,936**. Speculative receiver connections:
**1,942 → 1,928**. Liberal findings plus runtime assumptions total
**7,448**; these are not equivalent proof levels.

All seven entries retain **8,151 indexed files**, **189,154 CLI
counting records**, and the identical unresolved-base summaries covering
**1,133 unchecked inherited calls**. The explicit Wheels vendor root is
included in both measurements. The strict gains are all in the Lucee corpus.
No configuration edits, exclusions, or file-limit reductions account for the change.

Build, vet (normal and cflint_embed), full short tests, full short race tests,
make lint, the source formatting check and git diff --check pass. The first
sandboxed full test run could not bind loopback/Unix sockets; repeating both
full suites with approved local-socket access passed. New positive fixtures fail
with the scope/API additions disabled. Scope-clear, metadata guessing and numeric
formatting ambiguity regressions fail with their respective guards disabled.
The current source hashes are recorded alongside per-entry JSON deltas and checks.
No CFML engine execution or published PR CI result is claimed.
