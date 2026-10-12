# Query output bindings

This follow-up fixes a receiver relationship that native API additions could
hide: a local initialized as a string can later be replaced by a script query
output. The resolver now follows the completed output before deciding whether
a member exists. It also adds query some/every (boolean) and reduce (erased
result) to the public native API profile.

## Contract and assumptions

For bare/local variables with an earlier literal initializer, a completed
script query/cfquery/cfdbinfo output with a static matching name establishes
a native query. Query result metadata establishes a native struct. Both
attribute-form and parenthesized tag bodies must finish before their outputs
are used. The writer must be a standalone statement, not a potentially
skipped short-circuit/ternary call, in a lexical frame still enclosing the lookup.
A lookup on an assignment RHS runs before that assignment writes its result.

This is a normal-completion contract. SQL errors do not establish output on a
catch path. Closed conditionals, sibling closures, unbraced controls and switches
withhold the evidence. Unknown later uses/escapes/writes discard the new identity;
a small list of native read operations preserves it. Nonmatching output bodies
are still scanned for uses that can invalidate the receiver.

Static output names may include local.; other explicit scopes do not establish
a bare local binding. Attribute collections, interpolated/dynamic destinations,
duplicate attributes and colliding name/result destinations withhold evidence.
Shadowed callable tag names also withhold it. A query(...) literal function is
not treated as a tag output.

Query defaults are assumed only without HQL, dynamic dbtype, asynchronous
execution or an explicit listener. Explicit return types other than query are
withheld. Deployment-wide listeners or engine settings that change the standard
output behavior remain outside the source contract; this is not engine/version
certification.

The scan reads the current parse content, including unsaved text, and uses the
existing bounded producer tokenizer (4,096 tokens). Exceeding that proof bound
withholds the type rather than skipping a file. Markup, shared fields, outputs
without an earlier literal initializer, and outputs on the lookup's own line
remain outside this implementation. Method-return producer plans are unchanged.
These limits are different from corpus file limits.

Primary references: [query outputs, result metadata and return types](https://docs.lucee.org/reference/tags/query.html),
[cfdbinfo query objects](https://guides.adobe.com/coldfusion/en/docs/cfml-reference/__references__/cfdbinfo.html),
[script cfdbinfo](https://docs.lucee.org/reference/tags/dbinfo.html),
[query.some](https://docs.lucee.org/reference/objects/query/some.html),
[query.every](https://docs.lucee.org/reference/objects/query/every.html),
and [query.reduce](https://docs.lucee.org/reference/objects/query/reduce.html).
Reducer accumulator types remain unknown even when the initial value is a
literal: ColdBox SchemaInfo's results.append callback remains a separate gap.

## Native guess ambiguity

Adding the query reducer and predicates exposed a false family preference:
a reduce-only receiver could become a query guess even when the source's
shared value is an array or struct. Query is now withheld when any primitive
string/array/struct family supports the entire observed method set. Distinct
query operations such as columnData still allow an opt-in query guess.
This changes speculative classification, not source identity. It can move calls
from accepted guesses or guessed-receiver method errors into runtime assumptions.
The initial liberal count of 591 plus 6,703 runtime assumptions was rejected:
the per-entry audit showed structurally known structs guessed as queries.

## Validation and corpus comparison

Compared with the previous CFML native member checkpoint:

| Mode | Before | After |
|---|---:|---:|
| Strict findings | 12,995 | 12,977 |
| Liberal findings | 610 | 570 |
| Liberal runtime assumptions | 6,720 | 6,943 |
| Liberal speculative receiver connections | 2,253 | 2,113 |

Strict removes 18 findings and adds zero: 14 stale-string columnData findings
in LDEV3102, plus four queryNew some/every failures. ColdBox's query output
identity is corrected without a strict count change because filter/reduce
were already accepted as string operations.

The liberal comparison removes 40 finding records and adds zero. Only 18 are
the strict fixes. Nineteen former guessed-query method errors become runtime
assumptions; three matches.len calls become accepted through an already-existing
liberal list guess after removing the upstream sentEvents query guess. They
are not three newly proved source relationships.

The guess audit removes 140 speculative query connections and adds no
connections. Runtime assumptions add 223 records and remove none. The combined
remaining findings plus runtime assumptions therefore rise from 7,330 to 7,513:
more uncertainty is visible after correcting ambiguous native family selection.
This aggregate still does not include all calls accepted via speculative
connections. A small liberal findings total is not a source-proof total.

| Project | Strict findings | Liberal findings | Runtime assumptions |
|---|---:|---:|---:|
| Wheels | 2,570 | 143 | 1,337 |
| Masa | 3,687 | 21 | 2,139 |
| ContentBox | 1,754 | 44 | 827 |
| ColdBox | 1,637 | 93 | 727 |
| Lucee | 2,574 | 256 | 1,422 |
| TestBox | 396 | 7 | 258 |
| FW/1 | 359 | 6 | 233 |

All seven projects and both modes retain 8,151 indexed files and 189,154 CLI
counting records. The 46 per-entry missing-base summaries remain identical,
representing 1,133 unchecked inherited calls. These are accounting/coverage
checks, not an assertion that every parsed call is represented individually.

Baseline artifacts: target/resolution/cfml-native-members/checked.
This batch's per-entry artifacts: target/resolution/query-output-bindings/checked.

Regression fixtures verify positive query/metadata/collection callback typing
and negative lifetime, scope, shadowing, output and execution-mode cases.
Counterfactual Go overlays disable the output rule/native methods and the
pending-assignment guard; the corresponding final regressions fail.
ColdBox SchemaInfo line 91 filter and line 94 reduce trace to native.query,
correcting the previous string identity despite both method names overlapping.

Build, vet (including cflint_embed), full short tests, full short race tests,
lint and git diff --check pass. The source hashes used by the final corpus
binary remain unchanged through validation. Counterfactuals also demonstrate
the need for the execution-mode guards, completed bodies, standalone statements,
pending assignment ordering and scanning nonmatching output bodies. No CFML
engine runtime or new CI run is claimed.

## Remaining opportunities

The strict checkpoint has 10,361 receiver findings, 1,200 return-type findings,
982 method findings and 434 object findings. Only 263 findings report a missing
builtin method. Native API additions alone cannot address most of the remainder.

Recurring receiver names include result.filecontent (250), field (205),
content (180), item (144) and arguments.item (135). Those names are workload
groups, not established resolution rules. Inspect their producers and write
lifetimes before treating them as a common class. Priority directions are tag
output structs and member fields, callback/reducer accumulator inputs, outputs
without literal initializers, and shared registry values with a provable lifetime.

Struct some/every are also documented APIs
([some](https://docs.lucee.org/reference/objects/struct/some.html),
[every](https://docs.lucee.org/reference/objects/struct/every.html)).
Their missing API contracts remain a distinct follow-up; completing them does
not establish the lifetime of a struct stored and used in sibling closures.
