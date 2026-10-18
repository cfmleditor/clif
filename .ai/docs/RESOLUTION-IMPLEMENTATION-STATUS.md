# Relationship implementation status — 2026-10-12

The user authorized implementation of the documented work. The earlier
research-only restriction is superseded. This record distinguishes implemented
contracts from proposed architecture and unresolved proof conditions; it does
not describe the whole research programme as complete.

## Current checkpoint

The latest validated seven-project checkpoint is **12,210 strict findings**
and **285 remaining liberal findings**, after the
[application validation fixes](RESOLUTION-GAPS.md#application-validation-tassweb-and-nine-apps).
The assumption and connection ledgers were last measured at the preceding
checkpoint (12,348 / 289): **2,024 runtime assumptions**, **1,884 speculative
receiver connections**, and **5,094 provisional method connections**.
Coverage is 8,151 indexed files. Provisional links and runtime
assumptions are not proven strict resolutions, and receiver connections are
receiver groups rather than a count of resolved calls.

The [PR review fixes](RESOLUTION-GAPS.md#pr-244-review-ordered-struct-literals)
and [performance measurements](PERFORMANCE-GAPS.md) record the changes since
the milestones below. Performance-only changes must preserve these entries
and their evidence, not merely the headline totals.

## Historical implementation milestones

[Provisional method connections](RESOLUTION-PROVISIONAL-METHOD-CONNECTIONS.md) add
`--speculative-methods`: direct global declaration shortcuts with original
diagnostics, alternatives and unproven ownership retained. They replace some
remaining findings and runtime assumptions with concrete provisional links
without increasing the resolved count. With this additional opt-in flag, remaining
liberal findings drop from 509 to 295; 5,095 new provisional method links replace
214 findings and 4,881 runtime assumptions. Strict findings remain 12,355.

[Method evidence reporting](RESOLUTION-METHOD-EVIDENCE.md) now separates source
name availability from missing receiver/return/binding relationships, searches
method candidates at the failed chain hop, and labels speculative method evidence
as unproven. Existing opt-in acceptance policies remain unchanged.

The [broader method existence audit](RESOLUTION-METHOD-EXISTENCE-AUDIT.md) identifies
99 source-reviewed or native-contract-backed connection gaps, six likely source/API
issues, and five deliberate missing-method fixtures. Remaining entries retain
explicit ownership/existence uncertainty. This is research only; the strict
12,355 and liberal 509 finding counts are unchanged.

The [method connection classification](RESOLUTION-METHOD-CONNECTIONS.md) partitions
all 12,355 current strict entries into 12,041 unresolved connection/context cases,
310 selected-contract mismatches requiring review, and four source-reviewed
intentional absent-call fixtures. Four runtime registration findings are explicitly
reclassified as missing connections. This is documentation and audit work, not
new resolution gains; unmatched names are not treated as proof of absent methods.

The [string producer and reassignment review](RESOLUTION-STRING-PRODUCERS.md)
adds 37 fixed string producers, DateAdd's date result, two hidden Lucee string
members and a bounded native reassignment veto. It distinguishes caller named
arguments and simple struct keys from stores. Strict findings fall by 31 with
zero additions; two removals use existing dynamic acceptance. Ten runtime
assumptions become unnecessary while the liberal finding count stays unchanged.
Branch dominance, same-line assignment order and common alias/capture lifetimes
remain documented flow-model gaps. Variable receivers still account for 79.3%
of the strict findings, directing the next work toward source relationships.

The [builtin catalogue audit](RESOLUTION-BUILTIN-CATALOGUE-AUDIT.md) adds 33
verified array/struct producers, three public array APIs, callable shadow guards,
struct property erasure, and native assignment-order correction. Strict findings
fall by 82 net (84 removed, two reviewed additions); liberal findings fall by three.
It replaces 22 runtime assumptions with stronger evidence and adds one when
withdrawing an unsupported forward type. The catalogue recount is 970 documented
functions, 48 supplemental Lucee functions and 174 tags.

The [twenty-pass follow-up](RESOLUTION-TWENTY-PASS-FOLLOWUP.md) removes
71 strict findings with no additions. It adds bounded local-scope proof and
verified runtime API chains, corrects pool/factory/datasource identities, and
withholds metadata-only and numeric-compatible string guesses. The twenty-pass
ledger records implemented rules, rejected shortcuts and next relationship gaps.

The [InternalRequest lifetime refinement](RESOLUTION-INTERNAL-REQUEST-LIFETIMES.md)
removes 15 more strict findings with no additions by recognizing fresh local
result lifetimes and harmless scalar status reads. It also closes a missing
named nested function capture guard. The liberal finding/assumption partition is
unchanged; twelve net speculative receiver connections become unnecessary.

The [InternalRequest field follow-up](RESOLUTION-INTERNAL-REQUEST-FIELDS.md)
adds bounded Lucee text-result field typing and the string JSON decode contract.
It removes 423 strict findings with no additions, resolves seven runtime
assumptions, and replaces many optional string guesses with engine evidence.
Its initial local lifetime guards conservatively retained eleven cases pending
a more precise reinitialization proof; the follow-up above refines that boundary.

The [query output binding follow-up](RESOLUTION-QUERY-OUTPUT-BINDINGS.md)
corrects script query outputs replacing literal initializers, adds three query
APIs, and withholds ambiguous query guesses that also match primitive
collections. It removes 18 strict findings with no additions. The liberal
finding count drops by 40, including 19 reclassifications into runtime assumptions
and three acceptances through an existing speculative list guess.

The [CFML native member follow-up](RESOLUTION-CFML-NATIVE-MEMBERS.md) adds
17 documented methods, removes 56 strict findings with no additions, and
records the query-output identity gap that overlapping API names can hide.
Its liberal comparison removes 26 assumptions and adds eight every/some
assumptions after withdrawing ambiguous primitive receiver guesses.

The [native array List contract follow-up](RESOLUTION-ARRAY-LIST-CONTRACTS.md)
removes 37 strict findings without additions. It also withdraws the ambiguous
get-only struct guess, exposing 160 additional runtime-receiver assumptions
and removing 14 earlier assumption entries.

The latest measured strict total is **12,355**. Liberal ranked/native plus
open-world mode retains **509** findings and **6,905** runtime assumptions,
plus 1,884 speculative receiver connections. The previous query-guess correction
withheld 140 ambiguous query connections and exposed 223 additional runtime
assumptions. The current follow-up preserves unchanged
8,151-file coverage and 189,154 CLI counting records. These modes are not
equivalent proof levels. See [collection keys and returned struct values](RESOLUTION-COLLECTION-VALUES.md)
for the current implementation boundary and per-entry result, and
[the liberal implementation report](RESOLUTION-LIBERAL-IMPLEMENTATION.md)
for delivered ranking and open-world policies. Older research-only labels below
describe their historical checkpoint.

The [explicit collection declaration and runtime API batch](RESOLUTION-NATIVE-DECLARATIONS.md)
removes 161 strict findings with zero additions. It distinguishes generated
accessors from source declarations and adds verified public collection methods.
Shared registry values and closure writer lifetimes remain open.

The [collection closure ownership follow-up](RESOLUTION-COLLECTION-CLOSURES.md)
preserves independent named-function locals while retaining shared-field and
same-function closure vetoes. Its regression fixtures improve, but all fourteen
corpus comparisons and assumption partitions remain unchanged per entry.

## Historical checkpoint 33/34

The current [completion audit](RESOLUTION-COMPLETION-AUDIT.md) reconciles earlier
findings with code, tests, assumptions, experiments and outstanding obligations.
The historical remaining-work sections below must be read with that ledger.


The strict seven-corpus total was **13,346** after checkpoint 33's
bounded Mura admin owner assumption, following checkpoint 32's report-filter
correction, native contracts and tag script-island origin fix. The reporting-only
control is **19,484**; the cumulative reduction is **6,138 (31.50%)**. The earlier 10,136 total excluded qualified member failures
whose final name was a global builtin; it is not a comparable counting coverage.
Indexed/scanned coverage remains 8,151 files. Checked-record coverage is 189,154,
with 46 base summaries representing 1,133 unchecked calls. Experimental modes
are recorded separately below; checkpoint 34 extends opt-in native guessing
with staged primitive fallback. Strict behavior remains unchanged. The
historical tables retain their own coverage.

Production work now covers equivalent Wheels public records and Model copied
bodies; frozen application ancestry and inherited mappings; selected lifecycle
startup and DI/bean registrations; explicit relative parents; byte identities
for calls and closure stores; provider callback invocation arguments; literal
positional/named invoke dispatch; module discovery eligibility; and private
constructor-specific scalar/literal records, ordered private registrations,
closed fluent map/to chains and ordered constructor configuration overrides.
These are bounded source
contracts, with positive and rejection fixtures, rather than blanket acceptance
of a framework's unresolved methods.

The remaining substantial contracts are listed under Remaining substantive
contracts. The runtime fixture
now has a successful ColdFusion 2025 observation; generic copied defaults still
need a supported engine/invocation policy. Mutable binder execution, actual
migration dependencies and restored Mura state are not supplied by that probe.

## Implemented and measured

### Equivalent public-method records

wheelsIntegrationPlan now recognizes both the contiguous temporary-reference
form and the inline member-reference form within the public filter. The name
and referenced metadata member remain paired, and the reference must come from
the constructed instance. Intervening rewrites in the filter body do not qualify.
Existing producer, directory and consumer conditions still apply.

Regressions cover both layouts, wrong instance/member, private filtering and
record mutation. Disabling inline recognition causes the positive regression to
fail behaviorally. Existing Mapper/Controller boundary and host-return tests pass.

### Model method installation and copied-body context

The existing Controller integration contract is shared with Model using its own
init sequence, package directory and override-set selector. The lookup follows
real Model/Global ancestry, public provider methods, inherited overrides and
super aliases. A descendant replacing init or an integration helper withholds
copied methods. Duplicate public provider methods withhold a stable target.

Model provider helper lookup is limited to report callers that identify an
explicit public method in the provider source. Private bodies and parameter
defaults do not inherit a presumed Model context. A direct qualified provider
lookup does not gain Model methods. File creation uses the existing path
invalidation contract; access edits are observed from source stamps.

Regressions cover public helper calls, Global and sibling methods, super aliases,
private members, disabled/altered loaders, descendant overrides, duplicate files,
access edits, private caller bodies, defaults and direct provider lookup.
Disabling Model integration causes the positive regression to fail behaviorally.

No general Model return rule was added. new/findOne/findAll, cached class versus
instance identity and engine-adapter chains retain their separate obligations.

## Corpus comparison

Same extracted corpora, options and explicit roots as pattern-final. No preset,
configuration override or corpus source change was supplied. Indexed/scanned
counts match for all seven reports.

| Corpus | Before | After | Indexed/scanned files |
|---|---:|---:|---:|
| Wheels | 11,527 | 8,986 | 1,710 |
| Mura | 20,418 | 20,418 | 839 |
| ColdBox | 3,080 | 3,080 | 652 |
| ContentBox | 6,829 | 6,829 | 724 |
| Lucee | 1,440 | 1,440 | 3,776 |
| TestBox | 599 | 599 | 145 |
| FW/1 | 493 | 493 | 305 |
| Total | 44,386 | 41,845 | — |

Per-entry multisets preserve multiplicity. With reasons included, Wheels removes
2,564 findings and adds 23 replacements. Omitting only reason gives **2,541
accepted sites, zero new sites**, across 226 files and 353 case-insensitive method
names. Of these, 386 are Model-provider plain calls; the two investigated defaults
remain reported. The larger total includes application models and tests outside
the original 404-entry provider selection. It must not be compared as though the
Model research promised only those provider consumers.

All six other reports are identical per entry, including reasons. The final
report total is not the CLI's resolved-call counting unit. Raw parser coverage
was not separately instrumented.

Evidence: target/resolution/relationships-implemented holds reports/logs,
audit-sites.py, accepted-sites.json, site-summary.json and mutation logs. These
measure static resolver behavior. CFML runtime binding tests were not executed.

## Remaining documented work at checkpoint 1 (historical)

| Work | Status | Missing prerequisite or boundary |
|---|---|---|
| Engine-adapter return contracts | Unimplemented | Published/staged field provenance, whole-container writes and replacements; the research annotation was hypothetical |
| Model defaults | Deliberately withheld | Evaluation/binding evidence for copied parameter defaults |
| Model factory and selector-dependent returns beyond existing rules | Unimplemented | Class/instance state, callback effects, optional false/query/array/object returns |
| B0/B1 application ancestry and inherited mappings | Proposed architecture | Stable ancestry evidence independent of mapping bootstrap; creation/removal/ambiguity invalidation |
| B2 effective startup | Proposed architecture | Actual lifecycle selection and supported super execution, not all ancestor bodies |
| C configured factory identity and implicit DI | Proposed architecture | Constructor-specific binding ownership, ordered registration, unknown mutations and eligibility |
| D0 closure identity | Proposed architecture | Stable source spans across nested/same-line closures and region offsets |
| D1/D2 callback and migration dispatch | Proposed architecture | Invocation arguments and configured target routing; dependencies and versions in the actual scan |
| Mura receiver origins and template/request propagation | Investigated examples, unimplemented further rules | Per-entry factory/application/input ownership and guarded argument/storage transfers |
| Mura synthesized registry | Investigated, unimplemented further rules | Restored/mutated registry and argument precedence; current resolver already accepts known missing-method receivers dynamically |

These remaining items are not implemented by typing a receiver from its name,
borrowing an unrelated dependency tree or imposing a universal return type.
The current implementation is a completed first batch, not completion of every
research proposal. The next work should close the largest receiver-origin
families while retaining the explicit counterexamples in the research ledgers.

## Validation at checkpoint 1 (historical)

Go build, vet, full short tests, full short race tests, pinned lint (zero issues)
and git diff --check passed after the final caller-context restriction. Both
mutation proofs failed behaviorally with their corresponding implementation
disabled, then the restored Wheels regression suite passed. No PR or CI result
is claimed. Changes are uncommitted; the unrelated TASS support plan is untouched.

## Implementation checkpoint 2: ancestry, startup and callbacks

The next implementation adds the following bounded contracts. These are
production changes, rather than research annotations.

* Parent mapping lookup uses frozen server/editor inputs and declaring-relative
  paths. It never calls ComponentPath while computing mappings and never takes
  an indexed basename winner as ancestry evidence. Cache keys include explicit
  bootstrap inputs. Parent constructors run before child constructors; unused
  method declarations and their includes do not execute during this extraction.
* A selected script onRequest method may contribute outer mapping statements
  before it includes its actual page parameter. Nested conditional statements,
  another include expression and statements after page dispatch are withheld.
  This preserves the real Lucee LDEV4031 request-mapping relationship that a
  constructor-only extraction initially lost.
* Startup shared-variable discovery selects inherited onApplicationStart,
  onSessionStart and onRequestStart implementations. Replacing a lifecycle
  method excludes its parent body. A standalone, zero-argument script super
  statement at the outer body level expands the parent method at that position.
  Guarded super calls and tag super calls do not certify parent execution.
  Cross-directory parent values retain declaring-source paths. This is not yet
  a complete lifecycle execution model for DI discovery or framework hooks.
* Script CallSite metadata now carries method/argument-list source byte spans.
  Mixed cfscript regions, interpolation and lazy script-body parsing retain
  region origins. Unknown/synthetic expression origins remain explicit.
  Closure-local refs and pending refs also carry byte visibility ranges; nested
  and same-line stores no longer use the same line interval as identity.
  Line-only lookup still cannot fully identify a source occurrence.
* Callback parameter typing reads the provider's actual callback invocation.
  Closed straight-line source plans admit direct constructed objects, supplied
  objects and local callback/value aliases. Provider return metadata does not
  type the callback. Conflicting invocation types, unknown arguments, callback
  replacement, stored/returned escapes, and unsupported branch/loop dispatch
  withhold a result. This does not yet route configured migrations or arbitrary
  higher-order dispatch.

Regression evidence includes parent mapping precedence, distinct bootstrap
configurations, unused script/tag methods, parent creation/removal, discovery
invalidation for a missing mapped parent, inherited/overridden/guarded startup,
request-page dispatch, nested/same-line closure stores, shifted source and mixed
regions, and positive/negative callback argument contracts. Six mutation runs
fail behaviorally with the corresponding rule disabled and restore source in
finally blocks. The logs and reproducer are in
`target/resolution/relationship-next/mutations` and `verify_mutations.py`.

The byte metadata adds eight bytes to CallSite (120 bytes total on 64-bit) and
one word to scriptParser (392 bytes). The parser size guard now records this
intentional cost. The field-width/flag grouping avoids the initial 136-byte
CallSite layout. Closure visibility adds its own per-ref metadata.

The seven-corpus checkpoint before the final closure-store visibility extension
is identical per entry to relationships-implemented, including reasons. This
means the additional architecture contracts have fixture coverage but no
additional corpus reduction at that checkpoint; the 2,541 accepted sites from
the first implementation remain the measured gain. The final comparison and
checks are recorded separately when complete.

### Remaining implementation obligations after this checkpoint

The table above predates this checkpoint. B0/B1 now have physical/mapped
bootstrap support; bare names requiring an unproven catalog inference remain
withheld. B2 has bounded shared startup selection, but DI factory discovery and
framework-specific lifecycle routing need the same execution ownership.
D0 now has script call/ref byte identity; tag expression origins, occurrence
ordering within one store, and general arrow/provider contracts remain bounded.
D1 has the closed provider contract described above; D2 remains unimplemented.
Constructor-specific factory/binder values, ordered implicit DI ownership,
engine-adapter field lifecycle, Model selector/class-instance contracts and
Mura request/input origins and synthesized registries are still substantive
work. They are not completed by the new callback or ancestry rules.

## Implementation checkpoint 3: execution ownership and call transfers

Explicit ./ and ../ component parents resolve against the declaring file before
dot-path conversion. They cannot borrow a basename from another workspace root.
Case matching and optional .cfc suffixes have regressions, as do missing and
invalid paths. Lucee's ComponentLoader source supports the relative-path
distinction; its source and retrieval evidence are in relationship-next.

DI/1 and bean registration discovery now use the selected startup projections,
including parent constructors, selected lifecycle bodies and supported super
calls. Local factory ownership includes the lifecycle caller. Unused method
bodies no longer supply registrations. FW/1 setupApplication/setupSession/
setupRequest selection and a literal custom controllersFolder are supported.
Broader inherited/computed framework configuration is still outside this rule.

Unconditional literal constructor includes can declare lifecycle methods in a
separate template. Declarations and later includes retain their source directory.
Conditional declarations, duplicate names, cycles and repeated declaration
templates withhold a stable selected target. Occurrence counts distinguish an
unconditional include from another conditional execution of the same path;
the deduplicated include inventory alone cannot prove this relationship.

Literal invoke dispatch transfers arguments from arrays or structs. Positional
and named builtin forms use the documented object/name/arguments slots and their
aliases. Exact call spans separate same-line calls. Unknown matching targets,
bags, duplicate slots, mixed argument forms and absent/ambiguous source spans
block inference rather than borrowing a known caller. Computed selectors remain
outside the current caller index's coverage. A local invoke method is not treated
as the builtin. Configured migration routing remains a separate obligation.

Module model discovery observes constructor autoMapModels, modelsLocation,
modelNamespace and child-injector eligibility. Unknown or conditional policy
cannot authorize basename fallback. Explicit registrations remain available.
This exposed eight BrowserService@wheels-cli calls: the module disables automatic
model mapping and has no observed explicit BrowserService registration.

Tag conditions, return expressions, interpolations and cfset assignments retain
real call origins where lexical identity is proved. Existing receiver semantics
are preserved. Repeated ambiguous expression occurrences and synthetic closure
origins remain unknown; a unique lexical recovery is not full higher-order tag
identity. The parser size guards still report CallSite 120/scriptParser 392 bytes.

### Measured corpus and review

The implementation-final reports compare with pattern-final using the original
seven roots and configurations. Wheels is 11,527 → 8,865. The other six reports
are identical per entry including reasons. All indexed/scanned counts match.
Reason-free multisets show **2,680 removed sites and 18 new sites**, for a net
**2,662**; including reasons shows 2,703 removals and 41 additions.

The 18 additions were individually reviewed: ten become visible after relative
ancestry resolves, and eight are the disabled module-mapping group described
above. The ten include missing CLI dependency methods and shell receiver
ownership that currently falls through to an unrelated shell basename. These
are exposed obligations, not claimed successful resolutions. The review,
accepted/new entries and audit script are in implementation-reviewed.

The constructor-record and named-invoke checkpoint in factory-records is
identical to implementation-reviewed per entry. No corpus gain is attributed to
those new fixture contracts. Raw parser coverage was not separately instrumented;
AllCalls origin fixtures check call multiplicity and source positions.

## Implementation checkpoint 4: per-instance constructor and ordered state

Source evaluation retains private scalar fields or flat literal records from
each proved construction. An empty pseudo-constructor, no ancestry, a closed
init returning this and closed remaining methods bound the initial contract.
Two instances constructed with different paths or binding tables retain their
own lookup results. Literal-key reads follow that instance's private record.
Duplicate case-insensitive keys, unknown keys/values, nested records, interpolated
values, scope escape and exposed binding tables withhold a snapshot.

A closed private-field setter ending in return this now updates that instance
in source order. It accepts proved literal scalar/record values and withholds
unknown replacements. Statement and chained calls use the same state update.
Constructor chains evaluate their actual final method instead of keeping the
constructor's component guess. Branches, switches, loops, unknown effects,
aliases and reinitialization invalidate retained state conservatively.
Indexed registration writes, registration discovery, arbitrary fluent binders,
mapDirectory and framework injection are not covered by this closed setter rule.

The copied-default runtime fixture was executed on **ColdFusion Server
2025,0,10,331899** in an isolated network-disabled container using an existing
local image. Through Host slots, both interpolated and direct helper defaults
and the body used Host; supplied values remained explicit. Extracted local
function references failed to find the helper. Fixture, normalized observations
and image/configuration provenance are repository fixtures under
testdata/resolution-runtime/copied-defaults. This does not prove other engines,
plugin replacement or the application's deployed host ownership.

Lucee 7.1 source was separately inspected: ComponentImpl._call installs the
receiving component's variables scope before invoking UDFImpl; UDFImpl evaluates
default arguments before its body. This supports the receiving-slot distinction
as a source inference, not a second executed runtime result. The resolver still
withholds copied Model defaults pending an explicit engine/invocation contract.

## Remaining substantive contracts

* Factory/binder identity beyond the closed instance contract: indirect writes,
  framework map/to registrations, discovery roots, mutable aliases, escapes, caches and
  unknown writers. Wheels Injector ignores init's returned value while another
  factory may return invoke's result; they cannot share an assumed contract.
* Configured migration dispatch: deployed manager, dependency versions, schema/
  query factories and migration directory ownership. The extracted ContentBox
  roots omit those dependency sources; adding separate research roots would
  change the comparison's coverage and does not prove deployment ownership.
* Engine-adapter published/staged slots: construction alternatives, priority,
  whole-container writes, restoration and failure paths. A shared Base API is
  insufficient to establish the actual stored receiver or return value.
* Model factory/selector results: cached class versus instance, callback effects,
  optional false and query/array/object alternatives. Copied parameter defaults
  also require the invocation distinction recorded by the runtime probe.
* Mura origins and registry state: actual template/request factory, argument
  precedence, disk-restored or retained entries, updates and iterator element
  identity. Known onMissingMethod receivers already have dynamic acceptance;
  generated name prefixes alone cannot close unknown receiver origins.
* Broader startup and higher-order coverage: computed declaration includes,
  framework configuration inherited through arbitrary parents, execution order
  beyond the supported projection, tag closure origins and arrow/provider rules.

These remain implementation work. They are not counted as completed because
the foundation fixtures pass or because a hypothetical family inventory is large.

## Implementation checkpoint 5: inherited FW/1 ownership

Automatic FW/1 DI follows physically/mapped intermediate parents using the
frozen bootstrap lookup. Missing intermediates, cycles and another framework
cannot establish ownership. The default container retains literal inherited
constructor configuration and selected setup hooks. A selected custom
setBeanFactory excludes automatic DI/1. Nested application discovery uses
parsed ancestry, including unquoted extends, instead of a quoted-string regex.

Configuration comes from constructor projections. Conditional/braceless writes,
configuration changes in selected lifecycle bodies and unsupported tag
configuration withhold an automatic policy. Later framework configuration cannot
be presumed to have created an already initialized container with new roots.
Unused method bodies remain excluded. Unknown indirect configuration effects
and broader factory timing remain separate obligations.

The inherited-fw1 reports remove one additional finding and add none:
examples/subsystems/5helloservice/controllers/main.cfc, report line 4/source 5,
variables.greetingService.greet. Its Application.cfc declares unquoted
extends=framework.one; automatic service discovery supplies model/services/
greeting.cfc. The accepted method exists in that component. FW/1 changes
493 → 492 with all 305 files still indexed/scanned. The other six reports are
identical to ordered-instances per entry, including reasons. The total is
**41,723**, net **2,663** below pattern-final. Typed declaration recognition
adds no further corpus difference at its checkpoint.

## Implementation checkpoint 6: closed private registration state

Declaration masking now includes access/modifier and qualified return-type
headers, preserving byte positions and newlines. These tokens must not appear
as executable pseudo-constructor content. The factory fixture actually executed
on ColdFusion is also consumed by a Go regression, including its public any
methods and required struct/string arguments. Captured instance state requires
compatible supported argument types and required argument presence.

Closed private setters can now write one scalar field or a literal key in a
private flat table. An empty table can receive a literal registration and a
later case-insensitive rebinding; the last proved write supplies the lookup.
Unknown keys/values or unsupported execution invalidate the entire snapshot.
Generic source interpretation still treats these indexed writes as unsafe;
only the proved per-instance setter contract executes them.

The table must have one private stored reference. Constructor or setter aliases,
two fields storing the same argument record, whole-scope capture, externally
supplied record references and exposed tables are withheld. Publishing a factory
in a shared/component scope or passing it to another call (including a nested
argument expression) invalidates its state. A declared return type does not prove
that a call had no effect on a supplied factory. Distinct nearest applications
for construction and factory source also withhold a snapshot, rather than taking
the library's mapping context as the consumer's active application.

The instance-state ColdFusion 2025 probe confirms separate instances, whole-table
replacement/reversal, direct registration, case-insensitive rebinding and mutation
through an original caller-owned table. The latter supports the reference/alias
rejections. This is one executed engine/version, not a universal framework binder
or container-scope proof. Arbitrary fluent registrations, discovery, inherited
factories, caches and framework injection remain outside this source contract.

The registration-state comparison is identical to typed-instances per entry
including reasons. Against pattern-final, reason-free multisets now show
**2,681 removed sites and 18 new sites**, net **2,663**. Five reports are unchanged;
Wheels is 8,865 and FW/1 is 492. The same 8,151 files are indexed/scanned across
the seven runs. The audit script and accepted/new entry lists are in
target/resolution/registration-state. No change of roots, source, presets or
configuration is included in this measurement.

Checkpoint 6 passed go build ./..., go vet ./..., go test -short ./...,
go test -short -race ./..., make lint and git diff --check. Thirty targeted
mutation checks failed behaviorally with their corresponding fix disabled.
No published PR or CI result is claimed.

## Implementation checkpoint 7: configuration and fluent registration order

Repeated unconditional direct framework member writes now replace an earlier
constructor value, with case-insensitive member keys. Whole-struct assignments
replace the earlier configuration. Guarded writes, unsupported nested mutations,
unknown required settings and a final non-DI/1 engine still withhold automatic
DI. Selected lifecycle writes still cannot be assumed to configure the container
at its earlier creation point.

A closed private factory can retain a pending scalar registration key across
map(name).to(path), including a chain used as a statement. Every hop must be a
proved private setter returning the same object. Setter reads see each preceding
write in the same method. Another closed factory construction and its setters
retain a separate identity even when the final return reads only the first one.
Unknown hops, nested argument calls, aliases, publication and uncertain control
flow continue to invalidate the state.

The extended instance-state fixture executed on the same ColdFusion 2025 image
with network disabled and the fixture mounted read-only. It returned A for the
first fluent binding, B after case-insensitive rebinding, A for the ordered
single-method registration, and A for the first of two factories with interleaved
pending keys. This verifies that fixture's behavior; the actual Wheels Injector
still has publication, binder callbacks, guarded writes and caches outside the
closed contract.

The fluent-instances reports are identical to registration-state per entry,
including reasons. Source roots/configuration and the 8,151 indexed/scanned
files are unchanged. The latest total remains **41,723**, net **2,663** below
pattern-final (2,681 removed sites, 18 newly exposed sites). These extensions add
proved behavior but no additional measured corpus reduction.

## Implementation checkpoint 8: typed effects, defaults and tag source identities

A captured factory's closed setter executes its proved writes before a declared
component return type can provide a type-only answer. The returned type must
admit this factory; primitive or unrelated declared types withhold the snapshot.
The runtime fixture now declares Factory on init and all fluent setters, and
the ColdFusion 2025 probe produced the same normalized results. Its source is
also consumed by the Go regression.

Factory defaults are part of execution, before the body. The closed contract
allows literal defaults but withholds contextual reads, calls and interpolation
effects, even on an otherwise read-only getter. Argument binding cannot retain
an instance that escaped while a default was evaluated.

FW/1 configuration supports literal bracket member writes in constructor order.
Computed keys, nested mutations, aliases and references passed to possible writers
withhold automatic DI. Tracking a config reference is independent of the bounded
assignment-token window, so a long call cannot silently erase the escape.
Arbitrary indirect effects and broader lifecycle/container timing remain open.

Tag call origins now retain leading whitespace and actual multiline source lines.
Member-set subparses translate only spans inside the real expression, excluding
the synthetic wrapper. Interpolation origins follow nested quoted strings.
Deduplication uses source spans and conservative unknown-span fallbacks, preserving
ordinary calls alongside interpolated calls of the same name. Tests check call
multiplicity, exact bytes and source lines for same-line and multiline cases.
This changes extracted call/report coverage and must be reported separately from
newly resolved methods. Intermediate source-origins/origins-reviewed reports were
review experiments; the final reviewed report supersedes their totals.

Imported member stores also translate their closure ranges from the synthetic
wrapper to the actual expression. A regression checks two same-line closures
with different member names, including shifted source. Synthetic bytes cannot
be used as a real file visibility range.

The reviewed relationships-verified reports contain **41,707** unresolved entries.
Against fluent-instances, Wheels removes one duplicate report; Mura removes 66
entries and adds 51 when line/multiplicity are retained. Ignoring line only for
the separate location audit, Mura removes 16 duplicate occurrences and restores
one formerly suppressed item.getValue call on a line containing another getValue
outside the cfset expression. Actual same-named ordinary calls alongside string
interpolations remain present. These are extraction/location changes, not
method-resolution gains.

Checked calls (unresolved plus resolved) change from 63,068 to 63,067 in Wheels,
29,928 to 29,909 in Mura and 39,052 to 39,050 in Lucee; the other four totals are
identical. All 8,151 indexed/scanned files, roots and configurations are unchanged.
The reason-free comparison with pattern-final records 2,748 removed entries and
69 added entries, net 2,679. These entry counts include corrected locations and
must not be represented as unique newly resolved source calls. The audit scripts,
per-entry lists and call-coverage.json are retained in the ignored report output.

The final closure-coordinate fix leaves relationships-final-audit identical to
relationships-verified per entry, including reasons/candidates and multiplicity.
The final build, vet, full short tests, full short race tests, lint and diff check
pass. Forty-nine targeted mutation checks fail behaviorally when their respective
fix is disabled; source is restored after every mutation. The copied-defaults
and typed instance-state fixtures executed successfully on the recorded local
ColdFusion 2025 image. No PR/CI result or additional commit is claimed. The
Remaining substantive contracts section still describes unfinished work.

## Implementation checkpoint 9: closed setter forwarding

Closed factory methods can return a direct call to another closed setter on the
same object. Source contracts follow bare, this-qualified and variables-qualified
method calls, literal strings, argument members, named arguments and a sole
argumentCollection=arguments transfer. Every method in the chain must satisfy
the same private-state and self-return rules. Cycles, shadowed methods, init
calls, external receivers, chained forwarding bodies, nested argument calls and
private-scope publication remain outside this contract.

Typed wrappers execute their source before supplying a return type, preserving
delegated writes on the captured instance. An unknown or incompatible forwarding
argument invalidates the snapshot rather than leaving an old binding available.
The exact typed runtime fixture now exercises mapInstance forwarding its whole
argument collection to map and select forwarding a named argument to mapInstance.
ColdFusion Server 2025,0,10,331899 returned B after select/to and A after
mapInstance/to, matching the Go regression. This is not runtime evidence for
other engines or the full Wheels Injector startup/discovery/cache contract.

All seven relationships-forwarding report JSON files are identical to
relationships-final-audit, including findings, reasons, candidates, multiplicity,
checked-call coverage and indexed/scanned files. The unresolved count remains
41,707; this checkpoint adds no measured corpus resolutions. Three new mutation
checks confirm behavioral regression failures without forwarding admission,
typed-wrapper execution or invalidation on failed argument binding. Together
with the previous 49 proofs, there are 52 targeted mutation proofs. Full build,
vet, short tests, race tests, lint and diff checks are recorded after restoration.
The substantive framework contracts above remain unfinished.


## Implementation checkpoint 10: caller startup context and configuration stages

Implemented in the relationship-followup working tree:

- Caller views retain their execution Application root across library traversal.
  Shared startup values crossing that boundary retain their declaring component
  path. Two applications sharing a restrictive library resolve their own service;
  a direct library lookup cannot borrow either caller's startup context. Source
  indexing remains owned by the original resolver.
- An unknown argumentCollection is supplied unknown evidence, rather than an
  omitted argument. Existing whole-arguments forwarding remains separate.
- Provider and criteria-builder callback entry types are withheld after parameter
  rebinding or escape before the queried call. Provider inference now accepts
  parenthesized and single-parameter block arrows and colon-named arguments.
  Expression arrows, conditional provider dispatch and lifecycle callback order
  are not covered by this extension.
- Path invalidation also drops handler parses, caller/argument results, request
  writer results, wrapper/include hosts, lazy getter/source results, Mura event
  classifications and view argument/handoff results. Immutable DI policies and
  bean identities still require resolver replacement after configuration changes.
- Module public properties are read in constructor order, including literal
  bracket writes. configure() writes variables.conventions afterward; public
  this.conventions does not provide that value. Whole-struct replacement resets
  missing fields to the framework default. Unknown/conditional/escaped conventions
  and local shadowing withhold model roots. Environment methods mentioning
  conventions remain conservative because an environment selection is not proved.
  Disabled/inactive modules and child-injector model ownership are withheld.
  classMapping takes precedence over cfmapping; disabled modules supply neither.
  Qualified IDs no longer discard their namespace to borrow a matching basename.

New regressions were observed failing before the corresponding fixes for unknown
argument collections, callback mutation/escape, module lifecycle cases, caller
startup traversal, lazy-source invalidation, block arrows and disabled mappings.
A controlled mutation removing handler-cache invalidation fails the direct stale
handler-parse regression; source was restored. The qualified reporter include
fixture already passed before changes and is not counted as a fix. The missing
base report fixture now uses a bare Service injection so its aggregation check
is independent of the removed unproved Service@app basename fallback.

### Corpus audit

The original seven roots/configurations are retained, including the explicit
Wheels vendor root. All 8,151 indexed/scanned files remain covered. Reports,
per-entry multiset removals/additions and checked-call coverage are in
`target/resolution/relationship-followup/`, compared with relationships-forwarding.

| Corpus | Before | After | Removed entries | Added entries | Checked calls before / after |
| --- | ---: | ---: | ---: | ---: | ---: |
| Wheels | 8,864 | 8,864 | 0 | 0 | 63,067 / 63,067 |
| Mura | 20,403 | 20,265 | 138 | 0 | 29,909 / 29,909 |
| ContentBox | 6,829 | 6,796 | 70 | 37 | 16,818 / 16,815 |
| ColdBox | 3,080 | 3,081 | 1 | 2 | 19,357 / 19,357 |
| Lucee | 1,440 | 1,440 | 0 | 0 | 39,050 / 39,050 |
| TestBox | 599 | 599 | 0 | 0 | 4,027 / 4,027 |
| FW/1 | 492 | 492 | 0 | 0 | 2,611 / 2,611 |
| Total | 41,707 | 41,537 | 209 | 39 | 174,839 / 174,836 |

There are **139 source-backed newly resolved entries**: 138 Mura form-builder
mmRBF.getKeyValue calls now obtain the startup resourceBundleFactory through
includers, and one ColdBox module service obtains its effective model directory.
This is **0.33%** of the original total, far below the requested 30% target.

The additional 70 ContentBox removals are old cbRole addPermission/removePermission
method findings affected by preserving namespaced identity and the existing
uninstalled-module reporting policy. They are not proved new resolutions.
ContentBox exposes 33 oRule, three thisSetting and one thisPage unknown receiver
findings. Its checked-call coverage drops by three under that reporting policy;
there is no extraction or corpus-root expansion. ColdBox additionally exposes
HTMLHelper.toJson and PopulationSpec.populate. The net report drop is 170 (0.41%),
but that number must not be substituted for the 139 demonstrated resolutions.

### Remaining implementation scope

The proposed programme is not complete. Factory registration identity/order,
Mura's selected mutable factory and generated registry, return-mode and iterator
entity state, Wheels configured Injector and source-ordered test hooks, copied
renderer/assertion methods, FW/1 stored collection state and Lucee Field/Group
collections still need their bounded implementations and counterexamples.
The reporter include corpus failure has not been isolated by the passing minimal
fixture. Broad adapters cannot use these candidate families as type evidence.
No 30% improvement, full framework coverage, new commit or PR/CI result is claimed.


Final validation for checkpoint 10: plain build, vet, full short tests, full short
race tests, pinned lint and git diff --check pass. The initial sandboxed short
run could not bind daemon Unix sockets; the approved run outside that sandbox
passed, as did the final race run. No engine execution for the new module or
callback contracts is claimed. The final corpus binaries were built after all
behavioral edits; subsequent lint fixes changed formatting/refactoring only.

## Implementation checkpoint 11: guarded receivers and literal return modes

Positive source-backed `isInstanceOf(receiver, "Component")` checks now supply
unknown receiver identity inside their script/tag true branch. A mandatory AND
condition is accepted; OR, negation, else/elseif and shadowed checks are withheld.
The checked value must remain stable through the queried call: rebinding, escape,
whole-scope escape and includes invalidate the evidence. Quoted hash expressions
are executable code for this purpose. The guard and callback stability checks
reuse the parser's interpolation spans, including nested quotes and escaped hashes.
The content-keyed guard cache retains lexical evidence only.

Literal return modes now follow a proved final dispatch: modes with separate
returns are excluded, while other known literals can select its `return this`
fallback. Unknown modes/argument bags remain unknown. Declared return contracts,
argument signature identity and effect-free defaults must agree before the
shortcut is admitted. A mode-dependent method cannot borrow a context-free
component inferred from just one branch. This does not complete Mura loadBy:
its earlier mutations and dynamic `set(arguments)` require effect tracking.

### Corpus audit

Reports and per-entry multiset comparisons are in
`target/resolution/relationship-guards/`, compared with checkpoint 10's
`relationship-followup/`. The same seven configurations, original roots and
explicit Wheels vendor root cover 8,151 files and 174,836 checked calls.

| Corpus | Before | After | Newly resolved | New findings |
| --- | ---: | ---: | ---: | ---: |
| Wheels | 8,864 | 8,864 | 0 | 0 |
| Mura | 20,265 | 20,264 | 1 | 0 |
| ContentBox | 6,796 | 6,777 | 19 | 0 |
| ColdBox | 3,081 | 3,081 | 0 | 0 |
| Lucee | 1,440 | 1,420 | 20 | 0 |
| TestBox | 599 | 599 | 0 | 0 |
| FW/1 | 492 | 492 | 0 | 0 |
| Total | 41,537 | 41,497 | 40 | 0 |

All 40 removals are demonstrated guard-based resolutions: Lucee Group field
methods (10 getLevel, five getDisplayName, five getDescription), ContentBox
DataExporter/FileExporter methods (19), and Mura subItem.getAsStruct (one).
Return-mode changes add no measured corpus gain. Checked-call coverage is
unchanged per corpus in this checkpoint.

Against the 41,707 baseline, checkpoints 10 and 11 demonstrate **179 newly
resolved entries (0.43%)**. The net report reduction of 210 (0.50%) also includes
checkpoint 10's namespace/reporting effects and newly exposed findings; its
three-call coverage difference still applies. These totals remain far below the
30% target. Mutable factory/registry relationships, iterator entity state,
Wheels Injector and ordered hooks, copied renderer/assertion methods, and stored
collection state remain open. Lucee's guarded uses are improved, but general
Field/Group collection identity is not implemented.

### Validation

Positive and negative regressions were observed failing without their respective
fixes. Eight controlled mutations independently fail the guard admission/stability,
literal-mode selection, declared-return compatibility, signature, default-effect,
context-free return and interpolation regressions; exact source was restored.
The proof output is `relationship-guards/mutation-proofs.json`.

Plain build, vet, full short tests, full short race tests, parser/resolve package
tests, pinned lint and git diff --check pass. Full short/race tests used approved
execution outside the sandbox for daemon socket access. The final preallocation
and whitespace corrections were followed by parser/resolve short tests and lint;
they do not change behavior or corpus results. No CFML engine execution, commit,
published PR or CI result is claimed.

## Implementation checkpoint 12: fixes identified in the user review

The three concrete candidates in RESOLUTION-USER-REVIEW.md are implemented:

- `elseif` is a control keyword in script call extraction and braceless branch
  handling. Conditions and qualified methods named elseif still produce calls.
- Positive type checks can occur as later top-level mandatory AND terms. A call
  in the remaining condition suffix receives that evidence. Compound scoped
  paths are checked before ordinary member lookup; parent/key replacement,
  parent/value escape, interpolation effects and deferred callbacks withhold
  the evidence. OR, negated or compared checks do not establish identity.
- TestBox xUnit test methods recognize literal closure assertions registered in
  the selected beforeTests hook through the source-backed BaseSpec API. The
  contract checks the assertion constructor, registrar signature/overwrite
  semantics, inheritance and hook selection. An explicit super.beforeTests is
  supported only for a parent hook containing literal assignments. Child
  storage writes/field escapes, additional registration sites, primitive members,
  shadowed builtins, and instance escape in the queried method withhold inference.
  The definition target points to the registration hook. Arbitrary direct test
  invocation, BDD hooks, alias calls, callback registration and component return
  types from the registered closure are outside this contract. This is a bounded
  framework entry contract, not whole-program proof of public object immutability.

The intentional invalidFunction call inside TestBox's throws assertion remains
reported. No expected-failure suppression policy was introduced.

### Corpus audit

`target/resolution/review-fixes/` retains all seven reports, scan logs, per-entry
multiset removals/additions and checked-call coverage. It is compared with
checkpoint 11's relationship-guards snapshot using the same roots/options,
including the explicit Wheels vendor root. The 8,151 indexed/scanned files are
unchanged; no corpus source, configuration, dependency or preset was altered.

| Corpus | Before | After | Removed entries | Added entries | Checked calls before / after |
| --- | ---: | ---: | ---: | ---: | ---: |
| Wheels | 8,864 | 8,864 | 0 | 0 | 63,067 / 63,067 |
| Mura | 20,264 | 20,262 | 2 | 0 | 29,909 / 29,909 |
| ContentBox | 6,777 | 6,777 | 0 | 0 | 16,815 / 16,815 |
| ColdBox | 3,081 | 3,081 | 0 | 0 | 19,357 / 19,357 |
| Lucee | 1,420 | 1,417 | 3 | 0 | 39,050 / 39,047 |
| TestBox | 599 | 597 | 2 | 0 | 4,027 / 4,027 |
| FW/1 | 492 | 492 | 0 | 0 | 2,611 / 2,611 |
| Total | 41,497 | 41,490 | 7 | 0 | 174,836 / 174,833 |

Four entries are source-backed resolutions/registration recognitions: Mura
session.mura.editBean.getContentHistID/getUserID in the two loadextendedattributes
views, and TestBox isAwesome/isNotAwesome. Three entries are phantom elseif calls
in Lucee Query.cfc, at one-based lines 77, 90 and 112. They are extraction
corrections, not new receiver resolutions. The exact three-call coverage drop
is therefore expected and attributed, not a skipped root or reporting exemption.
There are now 183 demonstrated resolutions in checkpoints 10–12 against 41,707;
prior reporting differences remain separate. The 30% target remains open.

### Validation

The new parser, conjunction/scoped guard and literal-registration positives
failed before their fixes. Additional counterexamples exposed and closed an
extra registration-site admission and a deferred callback gap. Four controlled
mutations fail the parser, guard admission, compound stability and assertion
registration regressions; original source was restored after each mutation.
Mutation output is retained in review-fixes/mutation-proofs.json.

Final validation for checkpoint 12: plain build, vet, full short tests, full short
race tests, pinned lint and git diff --check pass. The full short/race suites
used approved execution outside the sandbox for daemon Unix socket access.
No CFML engine execution or published PR/CI result is claimed. The final corpus
binary contains all behavioral changes; subsequent lint edits were formatting
only. The user review report retains its original inventory and IDs, with a
prominent update identifying the seven completed cases and current totals.

## Checkpoint 13: immediate Wheels Injector startup result (2026-10-10)

A selected startup body can now bind the result of two adjacent outer statements:
`application.container = new wheels.Injector("wheels.Bindings");`
`application.wo = application.container.getInstance("global");`.
The container and output slots are taken from source; neither `wo`, `g` nor the
Global target is a variable-name convention. The actual literal binding selects
the component; ordinary assignment flow carries it into a local `g` alias.
Verified results take precedence over Application.cfc's parser return guesses.
Conflicting startup writes withhold the result.

The complete reviewed Injector lexical source is pinned, with its source fixture
in internal/resolve/testdata/wheels-injector/. Its initializer empties the mapping
and cache state, configures the selected binder and returns this. Its construction
helper returns the created object, ignoring init/onDIcomplete return values.
A closed binder has one public configure method, no constructor effects or
ancestry, and only literal map/bind/mapInstance-to chains. Duplicate keys,
computed selectors, flags, effects/defaults, escapes, intervening statements,
conditional construction, unused lifecycle methods, missing targets and changed
Injector source acquire no new inference. Later rebinding cannot replace the
already returned object. Arbitrary live container calls remain unsupported.
The full lexical fingerprint is conservative across Wheels versions: updating
it requires reviewing the source contract, not automatically accepting a new hash.

### Default comparison and deployment-context probe

All seven default scans are identical to checkpoint 12 per complete finding
multiset, indexed file count and filtered checked-call count. Total remains
**41,490 unresolved / 174,833 checked calls / 8,151 files**. The checked-in corpus
has no deployment configuration declaring public as its document root; vendor
specs extend ApplicationProxy, which exists under public but is unavailable in
their bootstrap ancestry lookup. The new producer contract cannot supply that
missing deployment context.

A separate same-options before/after probe supplies the explicit root fallback
`/ -> <Wheels corpus>/public`. This is an assumed deployment configuration,
not an automatically discovered root and not a replacement for the baseline.
Both sides index/scan the exact same 1,711 source files. The probe walks all
.cfc/.cfm files itself, whereas the default CLI scans 1,710; its results must not
be directly compared to the default scan as an implementation-only delta.

| Explicit public-root probe | Before fix | After fix |
| --- | ---: | ---: |
| Unresolved findings | 8,861 | 3,438 |
| Resolved checked calls | 54,225 | 60,237 |
| Filtered checked calls | 63,086 | 63,675 |
| Indexed/scanned files | 1,711 | 1,711 |

The per-entry difference is **5,434 removed / 11 added**, net **5,423** (61.2%
of that configured Wheels baseline). Removed receiver groups include g (2,396),
_controller (1,573), application.wo (486), user (324), m (146) and author (125).
The checked-call increase of 589 reflects additional checks reached through
known receivers/chains; these filtered counts are not raw extraction counts.
No corpus source/configuration or parser implementation changed in this batch.
Do not add these configured removals to the default corpus's cumulative gains.

The 11 new findings were reviewed at their sources: one policyScope return
chain; three model/$classData chains with unknown arguments or adapter fields;
four computed database-adapter lookups; fxReinclude installed by a runtime
include; and two plugin-installed Test methods. They expose deeper dynamic
contracts after a receiver resolves, not established application defects.
Reports, exact removals/additions, source excerpts and coverage records are in
`target/resolution/wheels-injector/`.

Positive regressions failed before implementation. Disabling startup integration
again fails literal, later-rebinding and preceding-conditional positive cases;
source is restored byte-for-byte by verify.py. Sixteen boundary cases pass,
including a different binding target that resolves to Other rather than Global.
The configured corpus comparison uses that same controlled mutation on the
before side and final restored source on the after side.

### Next root-context work

The root fallback must be known before Application ancestry is resolved:
ApplicationProxy -> public/Application.cfc -> Injector binding -> application.wo
-> g -> model/controller results. Parent constructors cannot bootstrap discovery
of their own missing ancestors. A future automatic root contract should read an
explicit deployment document-root declaration, resolve it relative to its config
file, and make it available to both bootstrap ancestry and ordinary path lookup.
Respect explicit mapping overrides and application boundaries; track config
changes as discovery dependencies. Missing, conflicting or runtime-computed
root declarations must stay unknown. Folder names alone are insufficient.

Final checkpoint 13 checks pass: go build ./..., go vet ./..., full short tests,
full short race tests, pinned lint and git diff --check. Full suites ran with
GOMAXPROCS=2/-p 1 and approved local Unix-socket access. No CFML runtime execution,
commit, published PR or CI result is claimed.

## Checkpoint 14: automatic project assumptions (2026-10-10)

Supersedes checkpoint 13's future-root note: the user explicitly authorized
documented best guesses, with configuration overrides. Standard layout and
framework context now work without adding corpus configs.

[PROJECT-ASSUMPTIONS.md](PROJECT-ASSUMPTIONS.md) records detection evidence,
precedence, opt-out behavior and failure modes. Wheels receives an assumed public
root before Application ancestry; five source-declared library namespace profiles
provide context to nested components. Exactly one recognized project owner enables
existing framework conventions in unconfigured CLI/editor use. Explicit editor
configs control their framework selection. Unevaluable named source mappings
suppress an assumed default; mixed project owners acquire no global preset.

| Project | Previous default | Current default | Diagnostics removed | Diagnostics added |
| --- | ---: | ---: | ---: | ---: |
| wheels | 8,864 | 2,506 | 6,373 | 15 |
| masa | 20,262 | 19,702 | 1,443 | 883 |
| contentbox | 6,777 | 2,776 | 4,022 | 21 |
| coldbox | 3,081 | 1,141 | 1,993 | 53 |
| lucee | 1,417 | 1,417 | 0 | 0 |
| testbox | 597 | 298 | 301 | 2 |
| fw1 | 492 | 253 | 285 | 46 |
| Total | 41,490 | 28,093 | 14,417 | 1,020 |

Net improvement is 13,397 (32.3% of checkpoint 12/13), or 13,614 (32.6%) below
the original 41,707. These include documented framework assumptions and existing
preset dynamic acceptance, not solely source-proven method resolutions. The
30% numerical target is met under that explicit policy.

Comparisons key diagnostics by file, line, function and reason, excluding candidate
metadata. Ignoring reason replacements, 14,164 calls stop being reported and 767
start being reported. All scans retain the same 8,151 indexed/scanned files.
Filtered checked calls increase 174,833 to 178,930; they are not raw extraction
coverage. Parser call extraction code is unchanged in this batch; no raw-call
multiset equivalence is claimed. Root-only assumptions reduce Wheels to 3,438;
namespace defaults then total 35,325; framework defaults produce the final 28,093.

Additions were grouped and representative sources reviewed, with every added
entry and its source excerpt retained. Masa dominates: 883 additions, including
854 return-type diagnostics reached through known base/receiver components. FW/1
exposes injected services; ColdBox exposes fluent returns, mixins and deliberate
test calls; TestBox exposes runtime mocked methods; Wheels exposes computed
adapters, runtime plugins and an inherited-call summary whose count changed.
These are unresolved relationships, not confirmed defects. Lucee is unchanged
per diagnostic. Candidate annotations can change independently and are not gains.

Per-entry JSON, coverage counts, full addition source excerpts and controlled
regression mutation logs are under `target/resolution/automatic-frameworks/`.

The make check formatting gate now checks tracked and unignored new Go sources
through `scripts/check-go-format.sh`, excluding ignored toolchain/module fixtures.
An initial complete check of this checkpoint caught a test-formatting lint issue;
that issue was corrected before the final validation run.

Final checkpoint 14 `make check` passes: builds and vet (including embedded CFLint),
Go formatting, full short race suite, pinned lint (zero issues), and vulnerability
scan (no vulnerabilities found). Controlled mutations independently disable the
Wheels root, project namespaces, unknown-override guard, CLI preset integration
and editor preset integration; each fails its positive behavioral regression,
and original source bytes are restored. The strengthened dynamic-root case keeps
the valid Wheels layout while withholding an unknown declared root.

A final repeat at `target/resolution/automatic-frameworks-final/` retains all seven
per-entry diagnostic multisets exactly (zero removed/added against the measured
checkpoint), with unchanged file coverage. No CFML runtime execution, commit,
published PR or CI result is claimed.

The separate full `go test -short ./...` suite and final `git diff --check` also pass.

## Checkpoint 15: Mura's existing preset as a source-layout default

Automatic detection omitted the existing Mura preset at checkpoint 14. Recognized
Mura projects now enable its scope receiver and display-template conventions.
No new bean alias, runtime registry or return-type assumption is introduced.
Explicit config still controls its framework selection. A CLI regression for a
real MuraScope method failed before this fix; it now passes while an undeclared
method remains reported. The CLI/editor share the same framework detector.

Masa falls **19,702 -> 16,169**, 4,328 removed / 795 added per diagnostic.
Ignoring reason replacements: 3,632 calls no longer reported, 99 newly reported,
and 696 changed diagnostic reasons. There are 793 added return-type diagnostics
and two template-method diagnostics. Representative sources were reviewed, with
every addition and source excerpt retained in `target/resolution/mura-assumptions/`.
839 indexed/scanned files and 32,938 filtered checked calls remain unchanged.

The combined default total is **24,560**, down 40.8% from checkpoint 12/13 and
41.1% from the original 41,707. This remains an explicit convention-based policy,
not a claim that all removed diagnostics were source-proven method resolutions.

Next research: MuraScope.getBean forwards to cfobject.getBean, which delegates
to application.serviceFactory. Startup constructs mura.bean.beanFactory with
recursive /mura scanning and literal exclusions; aliases such as content ->
contentBean are registered there, including engine-dependent contentGateway
alternatives. Existing BeanResolvers can analyze aliases, but currently requires
configured startup templates and an ID resolver; default Mura has neither. Simply
adding generic basename lookup would skip exclusions, replacement registrations
and runtime entity configuration. Preserve factory ownership and explicit config
precedence when introducing that future default. getContentBean reads event state;
getEvent can retain externally supplied request events, so neither is an unconditional
new-component return proof. No implementation of those additional rules is claimed.

Checkpoint 15 validation: full `make check` passes, including race tests, embedded
CFLint checks, pinned lint (zero issues) and vulnerability scan (none found).
Full `go test -short ./...` and `git diff --check` pass. Repeating all seven scans
reproduces every diagnostic exactly, with unchanged indexed/scanned coverage.
The pre-fix behavioral failure is retained in `automatic-frameworks/mura-before.log`.
No CFML runtime execution, commit, PR or CI result is claimed.

## Checkpoints 16–19: additional documented Mura assumptions

Four independently measured batches extend the policy in PROJECT-ASSUMPTIONS.md.
The standard layout carries the declared `/muraWRM` namespace across restrictive
physical applications; existing DI policy analysis supplies known literal bean IDs.
Explicit bean roots/startup files opt out of that factory default; explicit resolver
rules precede generated rules. Unknown replacements, duplicates, alias cycles,
computed roots and multiple matching policies remain untyped. Alias resolution
now consults the policy's own bean directory when the global bean map has no entry.
Injection keeps its separate lifetime/eligibility checks.

Preset assumptions add zero-argument object forms, five standard application
objects, three named admin request-context beans and content/current-user helpers.
Property getters and setter forms do not acquire these zero-argument types.
Generic `arguments.bean` remains unknown. Runtime manager, renderer, request
object and payload replacements can invalidate these conventions; explicit
component resolver rules or framework selection override them.

| Batch | Masa before | Masa after | Removed diagnostics | Added diagnostics |
| --- | ---: | ---: | ---: | ---: |
| 16: startup namespace and factory policy | 16,169 | 7,999 | 8,385 | 215 |
| 17: zero-argument config/renderer helpers | 7,999 | 7,873 | 143 | 17 |
| 18: standard application objects | 7,873 | 6,216 | 1,663 | 6 |
| 19: request-context beans and content/user helpers | 6,216 | 5,318 | 901 | 3 |

The combined total falls **24,560 -> 13,709**, net 10,851. The six non-Masa
corpora remain unchanged per diagnostic in every batch. Compared with checkpoint
12/13's 41,490, the final difference is 28,500 removed / 719 added (net 27,781);
ignoring reason changes, 28,070 calls stop being reported and 289 start being
reported. This is 67.1% below the original 41,707, including conventions and
existing preset acceptance, not only source-proven methods.

All comparisons retain 8,151 indexed/scanned files (839 Masa). Masa's filtered
checked calls grow 32,938 -> 33,664, and overall 178,930 -> 179,656. These are
filtered checks, not a raw extraction equivalence claim. Candidate metadata is
excluded from diagnostic identity. Audit scripts retain each addition and source
excerpt under target/resolution/mura-factory-final, mura-helpers, mura-shared and
mura-request. Additions were grouped and representative sources reviewed. The
last three are contentBean.getParent/getDisplayInterval/getExtendedData chains;
shared-object additions include legacy Java-loader returns and settingsManager.save.
They are unresolved relationships, not confirmed application defects.

Positive tests fail before enabling the helper/shared/request preset rules and
pass afterward, with property forms, generic bean names and lookalike names still
excluded. Three controlled mutations independently disable factory rules, policy
alias fallback and the host namespace; each fails the strengthened literal fixture
whose lifecycle method is declared behind core/Application.cfc. Files are restored
byte-for-byte. Excluded beans, external replacement, cycles, duplicates, computed
roots, absent presets, configured roots, unknown mappings and explicit precedence
have regression coverage.

Full checks caught missing API stubs for newly named preset targets. Mura's stubs
were regenerated from the existing pinned commit 696383140578f8dea3ece26f80cd7bfb370ddf0f;
24 components are now reachable from its preset. Real framework source retains
priority. No new source release or API signatures were invented.

Next opportunities: ownership of remaining argument beans, engine-dependent
factory aliases with a shared base, entity-dependent getFeed(entityName), event
payload values and source-declared manager/serializer returns. Generic argument
names and runtime data values are insufficient to choose arbitrary components.

Final checkpoints 16–19 validation passes: `make check` includes builds, vet,
embedded CFLint, full short race suite, Go formatting, pinned lint (zero issues)
and vulnerability scan (none found). Separate full `go test -short ./...` and
`git diff --check` pass. The full seven-project repeat in
`target/resolution/assumptions-checkpoint19-repeat/` retains every diagnostic
exactly (zero removed/added against final reports), with unchanged file coverage.
Remaining Masa receiver groups include generic arguments.bean (239),
arguments.feedBean (183), variables.configBean (164), feed (134), arguments.item
(128), content (109), entity (106) and arguments.renderer (103). These are next
review candidates, not automatic guarantees of one component.
No CFML runtime execution, commit, published PR or CI result is claimed.


## Checkpoint 20: conditional factory retrieval

Source-discovered DI/1 retrieval now shares the existing configured-registration
common-base behavior. Mura's engine-specific gateway alternatives retain a known
base API. Exact-target singleton injection is unchanged. Regression fixtures
cover related alternatives, unrelated alternatives and an unknown branch, plus
constructor retrieval versus singleton setter injection. Both positive regressions
fail with the fix removed and pass with it enabled. The injection fixture needed
an explicit model namespace and cache invalidation after changing its source;
the first full check correctly failed on that incomplete fixture.

Seven-project comparison against checkpoint 19: 86 removed, zero added, all in
Masa; current total **13,623**, Masa **5,232**. File coverage is unchanged at 8,151,
and filtered checked calls increase by one. Per-entry reports and representative
source review live under `target/resolution/mura-alternatives/`. This is 67.3%
below the original 41,707 baseline; no corpus configuration was added.

Further research confirms `cfobject.getFeed(beanName,siteid)` forwards its argument
collection to getBean and then invokes the selected bean's getFeed, whereas
MuraScope.getFeed(entityName) forwards an entity name. Their receiver-specific
argument names and runtime entity registry require ownership-aware rules; a
blanket getFeed return assumption would conflate unrelated APIs. Named argument
beans remain a separate opportunity; an untyped argument currently deliberately
shadows same-named fields, so any convention needs explicit precedence tests.


Checkpoint 20 final validation passes: `make check` (builds, vet, embedded CFLint,
full short race suite, formatting, zero lint issues and no reported vulnerabilities),
separate full `go test -short ./...`, and `git diff --check`. The seven-project
repeat in `target/resolution/mura-alternatives-repeat/` has zero removed/added
findings for every project. No CFML execution or published CI result is claimed.


## Checkpoints 21–22: admin payloads and published services

Added eight Mura admin request-context defaults backed by controller assignments
and manager/iterator construction paths. Added 23 application service names from
the literal startup service list, including its eventManager/Portcullis/permission
aliases. Generic bean/item names and engine-specific gateway assumptions remain
excluded. See PROJECT-ASSUMPTIONS.md for complete receivers, source evidence,
runtime limitations and config overrides.

Checkpoint 21: Masa 5,232 -> 4,962, 270 removed, zero added.
Checkpoint 22: 4,962 -> 4,658, 305 removed, one added; 304 net. Combined total is
**13,049**, down 574 this turn and 68.7% from the original 41,707. All six other
projects retain exact diagnostic multisets. File coverage remains 8,151; Masa
filtered checked calls increase 33,665 -> 33,677 -> 33,697, overall by 32.
Candidate metadata is excluded. Representative removed sources were reviewed;
the single addition was reviewed in full context: the legacy SOAP user API calls
readByEmail, which the scanned userManager does not declare. No runtime defect
claim is made.

Positive preset tests fail before each batch. A CLI integration regression also
fails with both batches removed, then passes with the defaults; it checks actual
methods, missing methods, generic rc.bean and explicit resolver precedence.
The custom rule in the test must include its Prefix, matching the real resolver
contract. Pinned API stubs now cover 63 Mura components; no upstream version was
changed. Audits and mutation logs are under target/resolution/mura-admin-context
and target/resolution/mura-application-services.

Next research: arguments.feedBean and variables.configBean retain sizable groups.
Untyped arguments deliberately shadow fields; name-only conventions need to
respect explicit types and distinguish setter ownership from arbitrary payloads.
rc.changesets comes from a feed.getIterator path, so its plural name alone does
not justify selecting changesetManager.getIterator's class. Trash wrappers also
must remain distinct from their polymorphic deleted object payloads.


Full validation initially found a legitimate inherited-only stub: pinned
mura/cache/cacheFactory.cfc is an empty deprecated subclass of cacheSimple.
The stub integrity test now accepts an empty class only when its declared base
is a known bundled stub (existing constants-only exceptions remain); ordinary
empty components and unknown bases still fail. The generator output is left
faithful to upstream source. The full framework API race suite passes this check.


Checkpoints 21–22 final checks pass: make check (builds, vet, embedded CFLint,
full short race suite, formatting, zero lint issues and no reported vulnerabilities),
separate full go test -short ./..., and git diff --check. The full seven-project
repeat in target/resolution/mura-services-repeat has zero removed/added findings
for every project and unchanged file coverage. No CFML runtime execution, commit,
published PR or CI result is claimed.


## Checkpoint 23: discovered factory ownership reaches setters

SetterLookup previously required conventional/configured bean roots or FW/1 scopes
before consulting source-discovered DI/1 policy. Mura's literal factory root is
core/mura, so setter fields were left unknown despite an established factory.
The discovered policy now establishes ownership first; singleton lifetime and
all exclusion/override guards are retained. No generic argument-name preset or
new framework API stub is introduced.

The Mura regression fails before the fix and passes after it. It now also checks
a setter-populated field's generated accessor, rejects transient dependency
injection, rejects excluded consumer files, and keeps files outside the policy
root unmanaged. Existing DI/1 lifetime/configuration and ordinary setter-root
regressions pass. Assertion code was extracted into a helper to meet the existing
cognitive-complexity lint limit rather than changing the limit.

Masa 4,658 -> 4,179: 495 removed, 16 added, 479 net. Twelve additions expose
getRBFactory returns and four expose manager.save returns, at already-reported
calls; there are zero newly-reported call identities. The other six projects have
identical diagnostics. Total **12,570**, 69.9% below the original baseline.
Files remain 8,151; Masa filtered checked calls grow 33,697 -> 33,739 (overall
179,689 -> 179,731). Candidate metadata is excluded. Additions were grouped and
representative source chains reviewed; no runtime defect claim is made.

Further research: settingsBean.getRBFactory constructs standard factories through
layered bundles but also accepts request.muraBaseRBFactory and cached objects.
A return convention would need to document these overrides. The remaining login
provider configBean fields lie outside core/mura and use constructor arguments;
they need caller/constructor ownership evidence. Untyped feedBean arguments still
require receiver-specific caller contracts, not a global same-name type guess.


The final repeat in target/resolution/source-policy-setters-repeat matches every
project exactly (zero removed/added findings), with unchanged file coverage.
Final make check, separate full go test -short ./..., and git diff --check pass.
No CFML runtime execution, commit, published PR or CI result is claimed.
Remaining variables.configBean findings total eleven: eight login-provider calls
outside the factory root and three formBuilderManager constructor-field calls.
The largest remaining Masa receiver groups include arguments.bean (239),
arguments.feedBean (183), arguments.item (128), content (109), entity (106) and
arguments.renderer (103). These remain research candidates, not guaranteed types.


## Checkpoints 24–25: source-shaped tooling defaults and discovery invalidation

Completed 20 investigation, implementation and review passes documented in
RESOLUTION-ITERATION-REVIEW.md. Two measured resolution batches activate existing
cfmigrations and CommandBox presets only after source-shaped tooling entry points
are found under one recognized project owner. Explicit config still suppresses
automatic selection. See PROJECT-ASSUMPTIONS.md for exact guards, global naming
assumptions and runtime overrides. No framework API regeneration was needed.

| Project | Checkpoint 23 | Migration context | CommandBox context |
| --- | ---: | ---: | ---: |
| Wheels | 2,506 | 2,506 | 2,233 |
| Masa | 4,179 | 4,179 | 4,179 |
| ContentBox | 2,776 | 1,640 | 1,378 |
| ColdBox | 1,141 | 1,141 | 1,034 |
| Lucee | 1,417 | 1,417 | 1,417 |
| TestBox | 298 | 298 | 246 |
| FW/1 | 253 | 253 | 253 |
| Total | 12,570 | 11,434 | 10,740 |

Migration context: 1,140 removed, four added (1,136 net). CommandBox context:
702 removed, eight added (694 net), using file/line/function/reason identity and
excluding candidate metadata. Combined 1,830 fewer net, 14.6% of this turn's
starting count. Files stay at 8,151; filtered checked calls grow by 76 to 179,807.
All additions were inspected with source excerpts. They expose configurable grammar
returns, Wheels module invocation paths, version-dependent service APIs and an
untyped test argument. No application-defect claim is made. Print methods include
intentional onMissingMethod behavior and are not all declared-method proofs.

Positive detector tests fail before each batch. Negative tests cover primitive
migration arguments, ordinary/outside folders, comments, local host declarations,
run arguments, deployment config and mixed project owners. The run-argument
counterexample initially selected CommandBox; requiring an actual run function
fixes it. Editor integration verifies real schema API checking, intentional print
dynamic dispatch, explicit opt-out, tracked source reads and directory boundaries.
A controlled mutation removing directory tracking fails the new-source regression;
original bytes are restored. Source/discovery API checks and pinned lint pass.

Directory reads now participate in DiscoveryDependencies, including failures and
new descendants; cached inferred tooling context can therefore be invalidated when
files appear or disappear. The final seven-project scan in tooling-defaults-final
matches assumed-commandbox exactly. No raw parser extraction equivalence claim,
CFML runtime execution, commit, published PR or CI result is made.


Checkpoints 24–25 final validation passes: make check (builds, vet, embedded
CFLint, full short race suite, formatting, zero lint issues and no reported
vulnerabilities), separate full go test -short ./..., and git diff --check.
The final seven-project scan retains exactly the same diagnostics as the measured
CommandBox checkpoint, with unchanged file coverage. All 20 review passes are
complete; no further implementation is implied for the withheld opportunities.

## Checkpoint 26: caller budgets follow method ownership

The batch resolver now applies its 60-caller-file budget after checking whether
calls belong to the target method. Lexical candidates for common names include
other components, declarations and comments; those no longer consume the owner
budget. A separate 512-candidate-file bound limits the search. The existing
40-passed-call-site bound remains. Unknown receivers, unknown supplied values,
unreadable files and exceeded bounds still withhold inference. Constructor
argument inference retains its separate existing 60-candidate-file policy.

This is source-based caller inference, enabled only for batch scans with
InferArgsFiles. It assumes the indexed callers describe the possible values;
external callers, runtime dispatch and plugin replacement are not proven by this
analysis. `--no-infer-args` disables it. No bean-name preset or API stub was added.
It uses existing ownership checks and does not yet build a dedicated index by
resolved owner or change those checks' inheritance/alternative semantics.

The same seven-corpus scan drops **10,740 -> 10,371**, with **369 findings removed
and zero added**, all in Masa (**4,179 -> 3,810**). Other corpora are identical.
The removals are bean calls in settingsDAO (135), userDAO (75),
feedDAO (72), categoryDAO (35), emailDAO (29), memberDAO (10), reminderDAO (8)
and mailinglistDAO (5). For example, feedManager reads a feedBean through feedDAO
and passes it to that DAO's update method; unrelated update receivers no longer
prevent following that source relationship. Plugin events can still mutate runtime
objects, which remains an inference limitation.

Coverage remains **8,151 files**; filtered checked calls increase by one,
**179,807 -> 179,808**. These are not raw parser-call equivalence counts. The
positive ownership-budget regression fails before the fix and passes after it;
unknown receivers, missing methods and all three budget boundaries have tests.
Reports and per-entry diffs are under target/resolution/owner-caller-budget.

Checkpoint 26 validation: `make check` (build, vet, short race tests, pinned lint
and vulnerability check), separate `go test -short ./...`, and `git diff --check`
pass. No CFML runtime execution or published CI result is claimed.

The final repeat scan in target/resolution/owner-caller-repeat matches every
diagnostic and file/filtered-check count across all seven corpora.

## Checkpoint 27: remove caller count limits

At the user's request, ordinary-method and constructor argument inference no
longer stop at a candidate-file, owner-file or supplied-call-site count. The
former 512/60/40 bounds and their bookkeeping are removed. Candidate lookup,
cached parsing and per-argument memoization remain; recursive inference still
has its existing depth bound. This applies only to batch caller inference,
not to unrelated parser/resolver traversal guards or cache capacity policies.

Unknown receivers, unknown argument expressions, missing source and conflicting
source evidence retain their existing handling. Source inference still assumes
indexed callers describe the possible inputs; external calls, dynamic dispatch
and plugin mutation remain limitations. `--no-infer-args` disables caller
inference. Large common-name buckets can now cost more work, since no count
cutoff substitutes an empty answer. No performance equivalence is claimed.

The replacement regressions demonstrate inference past each former limit,
including 66 constructor caller files. All four positive cases fail with the
old limits enabled and pass after removal. An unknown argument after the former
40-site cutoff still withholds inference, and missing methods remain checked.

The seven-corpus report remains **10,371** unresolved across **8,151 files**,
with **zero removed and zero added per diagnostic identity** and identical
filtered coverage (**179,808 checked**). This change removes a general coverage
restriction but adds no measured resolution gains to this corpus. The historical
checkpoint 26 limits above are superseded. Reports, before-fix failures and
per-entry audits are under target/resolution/uncapped-callers.

Checkpoint 27 validation: full `make check`, separate `go test -short ./...`,
and `git diff --check` pass. No CFML runtime or published CI result is claimed.

## Checkpoint 28: optional caller inference limits

Caller file and call-site limits are now optional through `.clif.json`:

```json
{
  "callerInferenceLimits": {
    "maxCandidateFiles": 512,
    "maxCallerFiles": 60,
    "maxCallSites": 40,
    "maxConstructorFiles": 60
  }
}
```

All defaults remain unlimited. Each positive value enables that cutoff;
omitted, zero and negative values leave it unlimited. Candidate files include
lexical matches for ordinary methods; owner files contain calls assigned to the
target method. Constructor candidates have their own file setting. Supplied
argument sites share maxCallSites across method and constructor inference.
Exceeding a limit withholds the type instead of accepting a partial caller set.
The policy block replaces an inherited policy as a whole; an explicit empty
block resets every limit. Unknown receiver/value handling and the recursive
depth guard remain unchanged. `--no-infer-args` disables inference altogether.

The policy flows through config decoding/merge, daemon settings, scan options,
the batch resolver and caller execution-context views. Editor sessions preserve
it when later configs omit the block, but live editor resolution still does not
build the caller index. Server exports retain the policy without enabling caller
inference that was previously off. CLI unresolved/explain and the MCP scan path
share the batch options. Large unlimited searches may require more work.

Five enabled-limit regression cases fail before enforcement and pass afterward.
Fixtures cover default/nonpositive unlimited values, unrelated owners, ordinary
and constructor limits, config-to-scan propagation, exact boundary behavior,
policy replacement/reset and session preservation. The full checks initially
caught the new optional block missing from session preservation; that plumbing
was corrected without weakening the reflective regression.

The default seven-corpus result stays at **10,371 findings**, **8,151 files** and
**179,808 filtered checked calls**, with **zero additions or removals per entry**.
No project corpus configs were added. Reports and failure/audit evidence are in
target/resolution/optional-caller-limits. Earlier unconditional limit removal is
superseded by this optional policy; the unlimited default is retained.

Checkpoint 28 validation: full `make check` passes (build, vet, short race tests,
pinned lint and vulnerability check), as do the full short suite and whitespace
checks. The new session-restoration test initially mutated an indexing session;
its restoration assertions now use an idle session and pass under the race
detector. No CFML runtime execution or published CI result is claimed.

## Checkpoint 29: omitted caller arguments and collection element contracts

An unknown receiver's same-named ordinary call previously vetoed argument
inference before argument presence was checked. It now cannot veto a type for an
argument it demonstrably omits. This requires an exact call byte identity and
balanced argument syntax. Calls with unknown supplied values, opaque
`argumentCollection`, unplaced `invoke` owners or missing spans still withhold
inference. Other owner/value and optional-limit policies remain intact. The
assumption is that only callers supplying the target argument contribute its
component type, consistent with existing handling of known-owner omissions.
This does not infer a type for an omitted argument's default or prove runtime
argument validity.

The change drops the seven-corpus total **10,371 → 10,235**: **136 findings
removed, zero added** (also 136 removed and zero added ignoring reason changes).
Wheels drops **2,233 → 2,184** (47 AnalysisService printer calls, two vendored
UnitRunner calls); Masa drops **3,810 → 3,723** (82 contentRendererUtility calls,
three contentManager calls, two pluginManager calls). The five other corpora
remain identical per diagnostic. All **8,151 files** and **179,808 filtered
checked calls** are retained. Resolved calls increase by 136; the same checked
population is retained. These counts do not prove raw parser extraction equality.
No project configuration was added.

An intermediate scan exposed seven TestBox reason replacements: a factory's
runtime `#arguments.bundlePath#` was mistaken for a literal component name when
caller inference could proceed. A factory-return regression reproduced that
false inference. Caller-derived component names containing interpolation markers
now remain unknown. The final scan has no TestBox replacements or additions;
there is no guessed bundle component.

Also implemented literal `entityLoad` array element typing and native array
`.each()` first-parameter typing from an independently established collection
contract. Accepted load forms are a literal registered persistent entity name,
optionally a literal filter struct and explicit `false` unique flag. The source
contract follows [Lucee's entityLoad documentation](https://docs.lucee.org/reference/functions/entityload.html).
Dynamic names, missing/nonpersistent entities, local shadows, scalar-ID overloads,
true/unknown unique flags and other options do not establish a collection. Native
`each` requires the exact first positional closure and a stable first parameter;
index parameters, unknown collections and callback parameter assignments remain
untyped. Existing ORM relationship and cborm collection element contracts can
also feed it. The homogeneous producer contract is assumed to hold while the
collection is consumed; arbitrary collection mutations/escapes are not proven.
This collection batch supplied **zero diagnostic changes** in the seven corpora
and is not credited for the 136-call improvement.

The collection, omitted-caller and runtime-name positive regressions each fail
without their fixes. Rejection fixtures cover the unsafe alternatives described
above and preserve missing-method checks. An intermediate full corpus repeat was
identical; the final scan with the runtime-name safeguard was audited against
checkpoint 28 by file, line, function and reason. Logs, per-entry additions/removals
and source audits are under `target/resolution/collection-callbacks/` and
`target/resolution/omitted-callers-final/`; before-fix caller/runtime-name logs
are under `target/resolution/omitted-argument-callers/`.

Validation: full `make check` passes (build, vet, short race suite, format,
pinned lint and vulnerability check), as do the separate full short suite and
whitespace check. No new CFML runtime observation or published CI result is
claimed. Remaining large opportunities need mutable instance/iterator contracts,
complete return-path proofs or supported native API models; ignoring unknown
supplied callers or guessing runtime component names is not a next step.

## Planning review after checkpoint 29

Further implementation is deferred pending current root-cause attribution.
[RESOLUTION-REMAINING-CAUSES.md](RESOLUTION-REMAINING-CAUSES.md) records the full
symptom inventory, source traces, 1,132 calls compressed into 46 missing-base
summaries, and evidence required to justify the next shared rule. Unknown
receivers/returns comprise 84.7% of entries, but their counts are not proven
fixability estimates. The recent generic collection batch's zero corpus gain
and remaining contextual/state/native contracts require a prioritization reset.
No production behavior or corpus counts changed in this review.

### Causal trace follow-up (research only)

The current root-cause review now records a controlled Lucee `/admin` mapping
scenario (99 removals, zero additions, control identical to the default), a
supplied Mura loadBy result with explicit bean/array alternatives, and a spanless
chained save that blocks the omitted-argument rule for a 65-entry changeset
cohort. The latter is a parser identity lead, not a verified gain. No production
implementation or default result changed; the total remains 10,235. See the
follow-up section of RESOLUTION-REMAINING-CAUSES.md for evidence and revised
priorities. Size/depth-limit removal is not the observed remedy for these traces.

## Checkpoint 30: chain origins and Lucee engine source context

Implemented the two concrete findings from the causal review. Mura's heterogeneous
bean/array returns remain unknown; no blanket fluent-loadBy rule was added.

Chained continuation calls now capture their byte spans before consuming each
argument list, with the real hop line, then emit in the existing order after
nested arguments. Source offsets remain exact in script, tag and wrapped tag
expressions, including repeated same-named hops and distinct argument lists.
This enables the existing caller omission proof to ignore a known-empty chain
call without ignoring unknown supplied values. Synthetic origins are not guessed.
The source-backed parser and argument-inference regressions fail without the fix.

A recognized Lucee source layout now supplies the assumed `/admin` namespace:
`core/src/main/cfml/context/admin`. Recognition requires the LuceeTestCase resource
and both admin helper sources (services.update.functions.cfm and ext.functions.cfm).
The nearest recognized layout owns the namespace; unrelated/missing-marker layouts
do not acquire it. Common bootstrap policy respects explicit config mappings,
Application mappings including dynamic declarations, and server.json/.cfconfig.json
boundaries. The assumption is that these source helpers run under their engine
namespace; it is not proof of arbitrary deployed Lucee mappings. Set an explicit
`mappings` entry for `/admin` to use a different deployment path. Regression fixtures
cover missing markers, config/source/dynamic overrides and server configuration.
The include integration regression fails when the assumption is disabled.

The seven-corpus total falls **10,235 → 10,136**. Lucee drops **1,417 → 1,318**:
58 LDEV6068 and 41 LDEV6069 helper calls resolve through the existing include graph.
All other corpus totals are unchanged. Comparing file/line/function/reason gives
**391 removals and 292 additions**. The 292 pairs are location corrections, not
new failures: 81 Wheels, 28 Masa, 48 ContentBox, 123 ColdBox and 12 TestBox entries
move to their actual chain hop. Ignoring location while retaining file, caller,
receiver, function, reason and unchecked count gives **99 removals, zero additions**.
Do not credit the moved entries as additional resolution gains.

Coverage remains **8,151 indexed/scanned files**. Resolved calls plus diagnostic
entries change **179,808 → 179,806**; Masa's resolved count falls 30,017 → 30,015.
Verbose before/after audits identify exactly two duplicate extracted getSite records,
in contentUtility.cfc:907 and userUtility.cfc:665. Each interpolated chain previously
emitted three records for its two getSite calls; both real calls remain after the
span-aware merge. A regression fails without the span fix and passes afterward.
This is an extraction correction, not skipped-source coverage. The **46 missing-base
summaries / 1,132 represented unchecked calls** remain unchanged. These reported
counts are still not an exhaustive raw parser extraction audit.

The 65-entry changeset cohort remains unresolved. A post-fix logging-only build
shows the original carch.cfc:567 save now has an exact span and is proven to omit
the argument. Inference advances to another uncertain caller, categoryManager.cfc:274
`obj.save()`, which it cannot place or prove irrelevant. This verifies removal of
one necessary blocker, not closure of the competing caller population. The 104-entry
feed cohort's bean/array ambiguity also remains. Neither is credited as a gain.

Artifacts: target/resolution/chain-spans-lucee-admin contains the final seven reports,
per-entry added/removed/source audits, moved-entry pairs and location-independent
summary, before-fix failures, verbose duplicate audit and changeset next-veto trace.
No production configuration files were added to the external corpora. Earlier
configuration-only experiment counts are now reproduced by the default scan.

Full `make check` and the separate full short suite pass. The additional duplicate
regression also passes under the focused parser/path/resolver race suite; pinned
lint and whitespace checks pass. Initial validation caught a missing import from
a linter autofix in a test; it was corrected before the successful full checks.
No new CFML runtime observation or published CI result is claimed.

## Checkpoint 31: opt-in speculative receiver connections

`clif unresolved --speculative` promotes a sole consistent candidate for an
unknown receiver group, using a strict snapshot from the current scan. Default
behavior stays at checkpoint 30's 10,136 entries, verified per entry across all
seven corpora. `--no-speculative` switches it off; `--verbose` records proposed
connections for review. This CLI-only experiment has no persistent config or
environment switch.

Enabled mode proposes 874 connections and reports 7,392 entries: 2,748 removed,
4 added, net 2,744 (27.07%). These reductions are not verified fixes. File counts
and missing-base entries remain identical; observed records increase by 143.
The rule can accept a method despite a known caller passing a scalar; a test
deliberately records that limitation. Ambiguous candidates, incomplete groups,
and known receiver types retain their ordinary checks. No caller-veto relaxation
or hard-coded project receiver maps were added.

See [policy, assumptions and validation](RESOLUTION-SPECULATIVE-CONNECTIONS.md).
Review artifacts are under target/resolution/speculative-cli, including an
explicitly unproven connections.json and per-entry differences. Build, vet,
full short suite, focused race tests, lint and whitespace checks pass. No new
runtime or CI accuracy claim is made.


## Checkpoint 32: native results, report coverage and completion audit

The [completion ledger](RESOLUTION-COMPLETION-AUDIT.md) reconciles the earlier
research families with code and tests, distinguishes bounded rules from full
relationship closure, and retains open work. The full earlier programme is
still incomplete: mutable registries/factories, copied Model defaults, staged
adapter publication, heterogeneous collection/selector identity, custom template
selection, record/cache alternatives and actual missing dependencies remain.

The current batch adds supported Java/builtin native identities and public
method/result contracts for Object/String/Class, Map/Collection/List/Set/Iterator,
JDBC, java.time, ORM session/factory/statistics and a selected Lucee engine API.
Known results propagate across assignments and chains. Erased or overloaded
results stay dynamic; arrays are not relabeled as Lists. The two Statistics
families and Iterator.next versus ResultSet.next remain distinct. Explicit Java
resolvers and tested custom CFC source returns retain priority. The native
contract catalog lives outside parser; parser contains only the lightweight
class identity map, preserving its dependency boundary.

`--speculative-native`/`--no-speculative-native` add a separate, reversible,
off-by-default usage-family guess. It requires exactly one family to answer all
observed unknown findings in a receiver group. `--native-engine auto|generic|lucee`
controls the batch Lucee producer profile. These methods assume compatible Java
17/Hibernate 5.6/CFML runtime interfaces and bridges; they are not runtime
execution, overload/argument or installed-extension checks. Version configuration
and generic element propagation through arbitrary loops remain open. See the
[native compatibility boundaries](RESOLUTION-NATIVE-TYPE-RESEARCH.md#implementation-follow-up-and-compatibility-assumptions).

The earlier native research found that the unresolved reporter hid qualified
failures whenever the final name was a global builtin. Filtering is now limited
to unqualified globals. A reporting-only binary, without native contracts,
changes 10,136 to **19,484**: 9,349 additions and one removed reason/base replacement.
This is restored diagnostic coverage, not a regression or a gain. The 46 base
summaries now represent 1,133 unchecked calls, one more than the filtered control.

A second earlier finding was still unimplemented: top-level named-call emission
inside tag function script islands captured spans after consuming arguments.
It now captures before consumption. The regression failed before the fix and
passes for repeated, multiline, nested and loop calls, shifted source offsets
and retained ownership. Mura removes **106 sites**, including all 65 investigated
changesetManager.save/arguments.bean findings, 25 contentDAO argument findings
and six trashManager restored-value findings. This closes the specific known
emitter, not every possible parser identity gap.

Review also rejected two overbroad shortcuts. Global promotion of declared
primitive CFC returns displaced source-backed Wheels factory contracts and was
removed. Whole literal inference now withholds an earlier initializer inside a
loop assigning unknown elements, with a fail-without-guard regression. Contract
review excludes protected Object methods, limits public clone, uses Collection
for Map.values, and preserves unknown overload/erased array results.

Same seven scan roots and indexed/scanned file counts as checkpoint 30. No
source/configuration root or preset was added. The reporting-only and final
modes each observe **189,154 checked records**. Current strict resolved count is
175,447. Against reporting-only, the full per-entry multiset comparison removes
**6,193** and adds **416**, a net **5,777**; additions include changed diagnostic
reasons and incomplete native catalog findings and must not be treated as
verified runtime failures.

| Corpus | Reporting only | Strict native/origin fixes | CFC guesses | Native guesses | Both guesses |
| --- | ---: | ---: | ---: | ---: | ---: |
| Wheels | 2,777 | 2,682 | 2,102 | 2,408 | 1,832 |
| Mura | 4,156 | 4,050 | 2,913 | 3,979 | 2,883 |
| ContentBox | 1,861 | 1,837 | 1,278 | 1,746 | 1,187 |
| ColdBox | 1,883 | 1,653 | 1,132 | 1,555 | 1,041 |
| Lucee | 8,048 | 2,730 | 2,621 | 2,595 | 2,490 |
| TestBox | 400 | 396 | 333 | 381 | 318 |
| FW/1 | 359 | 359 | 297 | 351 | 289 |
| Total | 19,484 | 13,707 | 10,676 | 13,015 | 10,040 |

Against current strict, CFC mode removes 3,035 and adds four (1,019 proposed
connections); native mode removes 694 and adds two (378 proposals); both remove
3,673 and add six (1,366 proposals). Native-only net reduction is 692 (5.05%);
both reduce 3,667 (26.75%). These are unreliable diagnostic reductions, not
verified receiver origins or additive implementation gains. More exposed
findings also enlarge the guess population relative to checkpoint 31.

All ten explicit native ORM findings in the earlier audit disappear: ColdBox
ormTest.cfm goes from seven entries to zero, and Lucee ormSessionCheck.cfm from
three to zero. This is source/scan evidence, not a CFML engine execution result.

Final validation: plain build, vet, full short tests, full short race tests,
pinned lint (zero issues) and git diff --check pass. Daemon tests initially could
not bind Unix sockets in the sandbox; the approved full runs passed. Focused
regressions pass, and disabling native contracts or the literal-loop guard makes
their respective regressions fail. No committed/published head, PR CI or new
engine runtime result is claimed.

Artifacts: `target/resolution/native-contracts/` contains reporting/control,
CFC/native/both JSON and logs, per-entry comparisons, connection inventories
with `originProven:false`, validation evidence, `run.py` and `review.py`.
Final scan binary SHA256:
`ae4e77fac9fccb7909747b11f887c0f704c786869bd5577e5e6b839f173277ab`.


## Checkpoint 33: embedded Mura admin owner

The earlier ten-pass admin-context opportunity is implemented as a documented
standard-layout assumption. Otherwise-unbased admin views/layouts inherit their
nearest governing Application only when recognized Mura mappings/layout and the
actual embedded framework's inherited renderer methods, public dispatch and live
parameter includes match. Existing implicit bases retain priority. No extra FW/1
preset is enabled. See [the assumption and its boundaries](PROJECT-ASSUMPTIONS.md#embedded-mura-admin-rendering-owner-checkpoint-33).

Regressions cover views/layouts, disabled framework context, unrelated directories,
nearer Applications, missing markers, mapping and renderer overrides, changed or
commented includes (including nested comments), parameter reassignment, explicit
implicit-base priority and discovery invalidation. The initial positive view and
layout regressions fail without the implementation. Creating a nearer Application
or editing the renderer after caching also invalidates ownership.

| Mode | Checkpoint 32 | Checkpoint 33 | Net reduction |
| --- | ---: | ---: | ---: |
| Strict | 13,707 | 13,346 | 361 |
| CFC guesses | 10,676 | 10,322 | 354 |
| Native guesses | 13,015 | 12,656 | 359 |
| Both guesses | 10,040 | 9,686 | 354 |

All seven strict corpora were rescanned. Mura goes **4,050 → 3,689**: 363 entries
removed (including 253 rbKey and 37 buildURL findings) and two added reason
replacements at existing display-content sites
(getType/getParent and getAllValues/getDisplayInterval unknown returns). The other
six corpora are identical per entry. Mura remains 839 files and 34,171 checked
records. Overall coverage remains **8,151 files / 189,154 checked records**, with
**46 base summaries / 1,133 unchecked calls**; strict resolved count is 175,808.

The three experimental totals combine fresh Mura scans with the unchanged six
checkpoint-32 experimental reports. They are composed totals, not 28 fresh scans.
Mura experimental counts are 2,559 CFC, 3,620 native and 2,529 both. Guess modes
remain off by default and unreliable; fewer diagnostics do not prove runtime
receiver identity.

Validation: plain build, vet, full short tests, full short race tests, pinned lint
(zero issues) and git diff --check pass. Disabling the owner fallback with a Go
overlay makes both final positive regressions fail again. Corpus artifacts and per-entry comparisons are under
`target/resolution/mura-admin-owner/`, including fail-before.log, review.json,
run.py and review.py. No new CFML runtime observation or published CI result is
claimed. Custom template selection, mutable factories/registries, copied defaults,
adapter publication, heterogeneous iterator/record origins and missing dependencies
remain open in the completion ledger.


## Checkpoint 34: staged primitive family fallback

The strongest ten-pass shortcut experiment is delivered under the existing
`--speculative-native` flag, with the existing explicit off switch. It remains
off by default. Existing native families retain a unique answer first; an
unanswered stage tries String/Array/Struct/StringBuilder and requires one family
to answer every observed method. This is priority between separate candidate
sets, not global uniqueness across the combined catalog. Existing CFC guessing
still takes precedence when both flags are enabled.

Verbose output labels “primitive family fallback”; connection records carry
Policy `sole-primitive-family`. Known receivers, unsupported/incomplete groups,
generic Object ambiguity and strict/off behavior retain their boundaries.
Regressions cover all four fallback families, existing List versus String/builder
priority, fallback after ambiguous interfaces, conflicting/unsupported methods,
known CFC rejection and CFC-first precedence. The positive family and wrong-origin
scan regressions fail before implementation and pass afterward. Supplying 42 to
ensureCapacity still demonstrates the deliberately unreliable StringBuilder
connection when enabled; this is documented policy behavior, not runtime proof.

| Mode | Before | After | Removed / added |
| --- | ---: | ---: | ---: |
| Strict | 13,346 | 13,346 | 0 / 0 |
| CFC guesses | 10,322 | 10,322 | 0 / 0 |
| Native guesses | 12,656 | 11,405 | 1,251 / 0 |
| Both guesses | 9,686 | 8,575 | 1,111 / 0 |

The delivered both-mode result exactly matches the staged primitive research
condition per entry. It is **35.75%** below strict, and **11.47%** below the old
both-mode result, with no added findings against that old experimental mode.
These are speculative diagnostic reductions, not 1,111 verified runtime fixes.
Strict stays at 13,346; no new strict resolution win is claimed.

| Corpus | Strict | CFC guesses | Native guesses | Both guesses |
| --- | ---: | ---: | ---: | ---: |
| wheels | 2,682 | 2,102 | 2,305 | 1,737 |
| masa | 3,689 | 2,559 | 3,520 | 2,429 |
| contentbox | 1,837 | 1,278 | 1,679 | 1,120 |
| coldbox | 1,653 | 1,132 | 1,312 | 906 |
| lucee | 2,730 | 2,621 | 1,939 | 1,854 |
| testbox | 396 | 333 | 320 | 261 |
| fw1 | 359 | 297 | 330 | 268 |

Final scan binary SHA256: `fb1a3de0d5ef30e870ed870819fbafc4c796c34d19b46e514eb02856268d8d20`.


All 28 production scans complete: four modes across the same seven roots.
Per-entry comparisons preserve strict and CFC-only findings; both-mode reports
are compared against the strongest research condition per entry. Coverage remains
8,151 files / 189,154 checked records, with identical missing-base entries:
46 summaries / 1,133 unchecked calls. No corpus configuration/source/root changed.
The old experimental comparison reuses six checkpoint-32 reports and Mura's
checkpoint-33 reports; all new production reports are fresh.

Validation: plain build, vet, full short tests, full short race tests, focused
policy regressions, pinned lint (zero issues) and git diff --check pass. Test
whitespace issues found by lint were corrected; one duplicate lint invocation
hit the tool's parallel-instance lock, and the final sequential lint passed.
No new CFML runtime or published CI result is claimed. Artifacts under
`target/resolution/primitive-fallback/` include fail-before.log, the scan binary,
all reports/logs/per-entry differences, run.py and review.py.

Candidate intersections/ranking, truncated-list promotion and caller-veto
relaxation remain research-only. Primitive origin, argument/bridge compatibility,
reassignment identity and original-versus-return-hop role remain open. See
[the delivered policy assumptions](RESOLUTION-SPECULATIVE-CONNECTIONS.md#delivered-primitive-family-fallback-checkpoint-34).

## Checkpoint 35: mass status refresh after #267–#270

Cut on 2026-10-18 after merging #267, #268, #269 and #270. The seven
candidate-list scans were re-measured on `main` (presets-on configs restored
from [RESOLUTION-GAPS.md](RESOLUTION-GAPS.md#measuring), masa-c under
`scripts/corpus/masacms.json`) and the lists under
[candidate lists](resolution-candidates/README.md) re-cut from those
JSONs. Totals: cb-p 1,683; cw-p 2,492 (vendor included); cx-p 1,616; fw-p 358;
lucee 1,985; masa-c 3,484; tb-p 380; 11,998 findings, 2,613 with a
high-confidence candidate. Because the checkout snapshots carry no git pins,
those totals cannot be compared with earlier ones; the controlling comparison
replays the same scans with a binary at the lists' commit (`6ebe727`) and with
current `main`: 121 unresolved call sites resolved, none added, and 38 lines
that still fail only changed reason. Attributions, each verified by replaying
single commits: the PR-244 follow-up's `$any` collection loop elements and
Mura zero-argument `event()` (44 of ContentBox's 46); PR #269's
param-statement fix, whose un-voided guarded shared-field contracts restore
MuraScope typing (all 52 masa-c MuraScope chains gone; TestBox's BaseReporter
receivers likewise); PR #268's missing-base parse for the remainder. The
presets canon of RESOLUTION-GAPS.md measures 7,243 / 16,428 with and without
presets; the "Where the entries are" tables there were refreshed from the
same JSONs. Quantities in this file before this checkpoint retain their date
and commit and were not retroactively rewritten. Replay artifacts:
`target/resolution/vintage-1012/` and `fresh-1018/` runs, the two binaries,
and `delta-by-line.sh` for the per-scan counts.
