# Remaining resolution causes: review before more implementation

Latest follow-up: [Ten research passes](RESOLUTION-TEN-PASS-RESEARCH.md), using
checkpoint 30's 10,136 entries. It confirms a second span-emission path and tests
embedded Mura admin context: a scoped research scenario removes 362/adds two
entries (net 360), while adding the whole FW/1 preset regresses. No new production
implementation is included. The initial checkpoint-29 inventory below remains
historical.

Review date: 10 October 2026. Input: checkpoint 29's seven JSON reports in
`target/resolution/omitted-callers-final/`. No resolver, parser, configuration,
stub or test implementation was changed for this review. Existing changes remain
in the workspace. This is a prioritization correction, not a new resolution win.

The recent fixes are measured, but we have not connected enough of the remaining
findings to their upstream causes to justify a large next implementation batch.
A failing regression demonstrates a rule's behavior; it does not establish that
that rule addresses a significant current corpus population. The collection
batch's zero corpus delta makes that distinction particularly clear.

## Current symptoms, counted exhaustively

These categories describe the last reported failure, **not its root cause**.

| Corpus | Unknown receiver | Unknown component return | Unqualified call | Missing method | Missing component | Base summaries | Total |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Wheels | 1,600 | 119 | 179 | 134 | 112 | 40 | 2,184 |
| Masa | 2,776 | 601 | 304 | 34 | 8 | 0 | 3,723 |
| ContentBox | 1,149 | 73 | 30 | 45 | 80 | 1 | 1,378 |
| ColdBox | 757 | 130 | 48 | 85 | 14 | 0 | 1,034 |
| Lucee | 1,040 | 17 | 84 | 217 | 55 | 4 | 1,417 |
| TestBox | 157 | 20 | 50 | 9 | 10 | 0 | 246 |
| FW/1 | 224 | 8 | 7 | 13 | 0 | 1 | 253 |
| **Total** | **7,703** | **968** | **702** | **537** | **279** | **46** | **10,235** |

Unknown receivers are 75.3%; unknown component returns are 9.5%. Together they
account for 84.7% of report entries. Naming candidates do not prove receiver
identity. More method declarations or more liberal name matching alone cannot
establish these missing relationships.

Among the 7,703 unknown receiver entries, 1,575 explicitly name `arguments.*`,
725 name `variables.*`, 405 name `request/rc/prc.*`, 175 name `application.*`, and
4,823 are local, unscoped or other expressions. This is a syntax partition;
local values may themselves originate from arguments, returns or shared state.
It cannot be used to promise a separate gain for each inference mechanism.

Masa, Wheels and ContentBox contribute 7,285 entries (71.2%). Masa's 2,776
unknown receivers alone are 27.1% of the whole report. Even resolving every one
would not meet a new 30% target; the other families matter too. A 30% reduction
from 10,235 would require at least 3,071 net removed entries, with coverage
preserved. We do not currently have a source-verified, nonoverlapping plan for
that number. Historical hypothetical inventories are not such a plan.

## The headline total compresses unchecked calls

The report deliberately groups calls behind a missing base into one entry per
file/base. Its `unchecked` field retains the number of calls involved; see
`internal/unresolved/unresolved.go`'s `scanFile` and base-group emission.

| Corpus | Base summary entries | Calls represented as unchecked |
| --- | ---: | ---: |
| Wheels | 40 | 1,118 |
| ContentBox | 1 | 1 |
| Lucee | 4 | 10 |
| FW/1 | 1 | 3 |
| **Total** | **46** | **1,132** |

Thus 10,235 is a diagnostic-entry total, not an exact count of individual calls
without a checked definition. Expanding just these summaries gives 11,321
reported unresolved/unchecked call occurrences: `10,235 - 46 + 1,132`. That is
still not a raw parser extraction total and says nothing about excluded builtins
or runtime-dynamic calls accepted by existing policies.

The earlier 179,808 figure is resolved-count plus diagnostic entries. It is
stable in checkpoint 29, but also reflects this compression. A future base fix
can reveal many new method or return findings even when it correctly restores
coverage. Measure base summaries and their represented calls separately before
claiming that lower diagnostic totals mean better checking.

The largest example is Wheels `cli/lucli/Module.cfc`, extending
`modules.BaseModule`: **one entry represents 763 unchecked calls**. The checkout
contains `cli/lucli/tests/_modules/BaseModule.cfc`, explicitly labeled a test
double. Its test Application maps `/modules` there. This does not establish the
production host base for Module.cfc outside that test execution context. Resolving
it through an arbitrary basename or the test double would inflate apparent
coverage with the wrong contract. The real host version and invocation context
must be established first.

## Source-backed investigations: what actually blocks the large cohorts

Counts below are investigation populations, **not promised resolutions**. Some
are exact receiver/context matches; some are whole-file workloads containing
multiple causes. Do not sum them as independent gains.

| Investigation | Current population | Source relationship and missing proof |
| --- | ---: | --- |
| Mura feed argument forwarding | 104 `arguments.feedBean` entries in `feedGateway.getFeed` | `content/feed/feedBean.getQuery` sets `arguments.feedBean=this`, then forwards to feedManager; feedManager forwards to feedGateway. The existing explanation still ends at an untyped gateway argument. Identify the precise veto or lost ownership on this chain and every competing supplying caller before changing a rule. |
| Mura API entity/iterator provenance | 281 entries in JSON v1 apiUtility; many causes | API routes compute entity names and pass results into helpers; iterator loops use `iterator.next()`. beanORM.getIterator sets entityName on a generic beanIterator; beanFeed.getIterator has entity-specific, generic, distinct and aggregation alternatives. beanIterator.packageRecord can return an existing object from an array. A generic element-return assumption cannot close all alternatives. |
| Mura content load/return forwarding | Broad receiver and return cohort, not yet enumerated by proven origin | contentBean.loadBy passes `this` to contentManager.read. read dispatches into six helpers, whose cache/DAO paths can replace that value; getContentVersion explicitly tests whether a returned bean is an array. Prove return alternatives and preserved identity per path, rather than making every loadBy fluent. |
| Wheels engine-adapter publication | 55 exact `application.wheels.engineAdapter` receiver findings; related helper returns are separate | Startup creates four engine-specific alternatives in `application.$wheels.engineAdapter`; Global.$engineAdapter reads either staged or published state. Follow publication/copy and replacement paths before applying constructor alternatives to the public slot. Shared methods alone do not establish the stored object. |
| Lucee admin field/driver contracts | 140 exact `field` receiver entries across templates | getCustomFields returns driver fields; Gateway helpers construct Field and Group. The consumer branches on `isInstanceOf(field,"Group")` and continues before using Field-only methods. It needs driver origins, heterogeneous element alternatives and branch narrowing, not the homogeneous ORM array rule just implemented. |
| Lucee engine mappings and included helpers | 58 entries in LDEV6068 and 41 in LDEV6069 | beforeAll includes `/admin/services.update.functions.cfm` and `/admin/ext.functions.cfm`; those sources exist under `core/src/main/cfml/context/admin`. The current explanation checks local/extends and fails. Determine engine mapping context and include reachability; this is not evidence that 99 methods are absent. Include execution/scope still needs review after path resolution. |
| ColdBox native object returns | 46 exact `now` receiver entries in ScheduledTask | Local now comes from getJavaNow → DateTimeHelper.toLocalDateTime → a native time-object chain. CFML component declarations alone cannot supply its native methods. Define the supported native API/type boundary and source version; do not label every native method as resolved. |
| ContentBox criteria alternatives and result records | 14 `c` entries in findPublishedContent; related result values separate | `c = isNull(arguments.criteria) ? newCriteria() : arguments.criteria`. The constructor branch alone does not prove the supplied branch. The function documents a `{count,content}` record, whose elements require their own origin. Trace caller alternatives and returned field shapes. |
| ContentBox absent/legacy dependencies | 54 exact ORMUtilFactory diagnostics within 80 missing-component entries | Both getORMUtil calls and chains name `coldbox.system.orm.hibernate.util.ORMUtilFactory`, whose source is absent from this scan root. Establish compatibility/version and whether this is deployment absence or an outdated contract before adding mappings/stubs. Missing sources cannot be repaired by receiver inference alone. |

The feed and Lucee include CLI explanations are saved in
`target/resolution/remaining-cause-review/`. They confirm the observed failure
but **do not expose every failed caller-inference candidate or inference veto**.
That is an observability gap: a terse final reason is not a causal trace. We need
that evidence for the high-count cohorts before selecting an inference rewrite.

These investigations span four distinct needs: contextual invocation/mappings;
mutable object and iterator identity; argument/return/record propagation; and
native or absent dependency contracts. One universal naming assumption cannot
safely cover them.

## What our process missed

1. We did not maintain a current mapping from the remaining entries to source
   producers and shared blockers. The reproducible inventory now contains 4,235
   file/caller/receiver/reason groups, but those are context groups, not 4,235
   established causes. Most still need attribution.
2. We moved from a general feature idea to regression/implementation before
   proving its use in the current workload. Literal entityLoad and homogeneous
   native each were useful supported rules, but not the shapes in the large
   remaining driver/iterator families. Removing caller count limits also yielded
   zero further gain because count limits were not those callers' only blocker.
3. We mixed headline diagnostics with coverage and retained obsolete review
   inventories. The original user report's body still contains checkpoint 11
   examples and counts beneath many later updates. It is historical review
   material; use this inventory for current prioritization.
4. We have not separately established test negatives, real source defects,
   native-runtime contracts and missing deployment dependencies. A test path is
   not automatically an intentional invalid call: the two Lucee helper examples
   are positive tests with existing source definitions. Conversely, a method
   name match does not establish that a reported test should pass static checks.
5. We checked exact duplicate source files: 17 groups, 40 finding-bearing files,
   84 entries. Duplicate files are not the main explanation for 10,235 findings.
   The same bytes can have different execution contexts and different findings;
   do not remove those entries or deduplicate gains by bytes alone.

## Required evidence for the next implementation choice

First prioritize Mura argument/return/iterator chains and execution-context gaps
in Lucee/Wheels. Keep native API and absent-dependency work separate so its
coverage and assumptions can be reviewed independently. Use the complete current
context inventory to select examples, not names recalled from old plans.

For each proposed shared blocker, record:

- Current entry identities and represented unchecked calls, plus the producer,
  ownership, transfer and consumer path. Assign overlapping entries once.
- The exact resolver stage where evidence is lost, including failed/competing
  callers and unsupported return/state alternatives. Separate a missing rule
  from source ambiguity, missing dependency and intentional dynamic behavior.
- A minimum supported contract, replacement/mutation counterexamples and the
  deployment assumptions or override required. Identify any valid methods that
  would remain unchecked by an acceptance rule.
- A deduplicated impact population and an explicitly provisional estimate.
  Count deeper newly exposed findings separately; do not call a whole matching
  receiver/file population fixable.

Only then choose a production change and demonstrate that its failing regression
represents the measured cohort. Follow it with the existing per-entry and coverage
audits. A large architectural feature is justified when several verified families
need it, not because its hypothetical scope sounds broad. A small fix remains
worth doing when justified, but it must not substitute for the larger attribution
work the user requested.

## Reproduction and validation of this review

`python3 target/resolution/remaining-cause-review/inventory.py` reads checkpoint
29 JSON and current corpus sources. It writes `inventory-summary.json` (including
input hashes), complete `receiver-contexts.json` with original findings/source
samples, and `duplicate-source-files.json`. Every entry is assigned one symptom
bucket and one context group; totals reconcile to 10,235 and 1,132 unchecked calls.

This review reused the last fully checked implementation and its corpus outputs;
no new corpus resolution result, runtime observation or CI pass is claimed.
Only documentation and ignored local analysis artifacts were added. Repository
whitespace checks pass; production tests were not rerun for this documentation
review.

## Follow-up causal traces: context, supplied values and lexical identity

Further research used Go build overlays stored under
`target/resolution/remaining-cause-review/`. Logging code is compiled from copies
of source files; it is not written into the working resolver/parser/CLI files.
An explicit configuration scenario is enabled only in a separate research binary.
No production default, corpus source or project config was changed.

### Lucee: a verified configuration opportunity, without a new include rule

The baseline trace for LDEV6068's isStableVersion confirms both `/admin` include
paths fail `Resolver.IncludePath`. An explicit research mapping of `/admin` to
`core/src/main/cfml/context/admin` resolves that call and LDEV6069's toOSGiVersion
through the existing include graph. This checks both missing path resolution and
the subsequent function lookup; it is stronger evidence than finding a declaration
with the same name somewhere in the corpus.

A full Lucee configuration-sensitivity scan yields:

| Scenario | Entries | Removed | Added | Files | Resolved + entries | Unchecked calls represented by bases |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Current default | 1,417 | — | — | 3,776 | 39,047 | 10 |
| Explicit research `/admin` mapping | 1,318 | 99 | 0 | 3,776 | 39,047 | 10 |

The removals are exactly 58 in LDEV6068 and 41 in LDEV6069. Per-entry identity is
file, line, function and reason. The research binary with its scenario switch
**off** reproduces the default report identically per entry. The 140 `field`
receiver diagnostics remain; engine include context and driver element identity
are independent problems in this workload.

This does not add 99 fixes to checkpoint 29: the default seven-corpus total is
still **10,235**. Applying only this mapping would conditionally yield 10,136;
it has not been deployed or selected automatically. A future default needs an
engine/source-layout ownership rule, collision/override handling and applicability
boundaries. The test source uses the virtual path, but this extracted CFML corpus
does not by itself supply all engine deployment configuration. We must document
the assumption rather than describe it as universally proven runtime setup.

Evidence: `lucee-include-path-baseline.log`,
`lucee-include-path-configured.log`, `lucee-ext-path-configured.log`,
`lucee-admin-mapping.json`, `lucee-admin-{removed,added}.json`,
`lucee-admin-audit-summary.json` and `lucee-scenario-control.json`.
The mapping experiment changes configuration only; no include algorithm was fixed.

### Mura feed: the first veto is a supplied value, not a size/depth cutoff

The 104-entry cohort is `arguments.feedBean` in feedGateway.getFeed. Logging the
current caller logic shows this sequence:

1. Gateway inference reaches feedManager.getFeed's forwarding call, whose owner
   is known and whose feedBean argument is supplied.
2. Recursive manager argument inference runs at depth 1. Its first relevant known
   caller is `core/modules/v1/feedslideshow/index.cfm:192`, passing
   `variables.feedBean`.
3. That expression yields no component. Since this is an explicitly supplied
   argument, not a synthetic forwarded-argument marker, inference returns unknown.
   The gateway therefore also remains unknown.

The template creates its value at lines 173/175 with
`variables.$.getBean("feed").loadBy(...)`, selecting feedID or name. feedBean.loadBy
passes `this` into feedManager.read; read selects readByName when name/siteID are
present and feedID is empty. feedDAO.readByName has an explicit multiple-record
branch returning beanArray (lines 166–175), while other branches return a bean.
Consequently the caller population cannot be made uniformly component-valued by
an unconditional fluent-loadBy rule. Cache and custom receiver paths also need
separate proof. This is a source counterexample, not a runtime observation or a
claim that the application's particular slideshow receives an array in practice.

The observed veto occurs before the depth-3 guard, and there are only 25 lexical
candidate files for getFeed. Raising recursion depth or removing count limits
cannot repair this particular first failure. It might expose different paths
elsewhere; no global claim about all depth-limited calls follows from this trace.
Do not ignore this supplied caller to manufacture a consensus type.

A proposed return model must preserve bean-versus-array alternatives and narrow
safe consumers by selector/guard/context. The current argument inference interface
returns a component string or unknown; it is not a complete representation of
these heterogeneous result alternatives. Investigation of that broader contract
is justified here; implementation and its impact remain unproven.

Evidence: `mura-feed-caller-trace.log`, `mura-feed-blocked-cohort.json`,
feedslideshow/index.cfm:173–192, content/feed/feedBean.cfc:612–619,
content/feed/feedManager.cfc:158–174 and content/feed/feedDAO.cfc:146–186.
The 104 findings are one downstream cohort, not 104 independently fixable defects.

### Mura changesets: a parser identity gap blocks the omission rule

The distinct 65-entry cohort is `arguments.bean` in changesetManager.save.
The current trace searches 85 candidate files and stops at
`admin/core/controllers/carch.cfc:567`:

```cfml
local.contentBean.getStats().setLockID(session.mura.userID).setLockType('file').save();
```

The final save's owner cannot be placed; its recorded `HasSpan` is false and its
start/end bytes are zero. Although the visible call has no arguments,
`callerOmitsArgument` intentionally cannot certify an omission without lexical
identity. It therefore vetoes the changeset argument inference. The record's
base variable is local.contentBean; the relevant receiver is the chain result,
so the base variable name alone is not proof of the save owner.

There is a concrete parser path explaining why downstream hops can lose their
identity: `recordChainContinuationFrom` calls `skipParens` before `addCall`.
`addCall` invokes `setCallSpan`, which requires the scanner still at the argument
list. That condition is no longer true. This path emits semantic call records
but can leave the bytes unavailable. The observed corpus record is spanless;
confirm the exact emitting path with a small chain fixture before choosing a fix.

A future span-preservation change must record each real hop's identity before
consuming its arguments, retaining current receiver/chain semantics and synthetic
origin safeguards. Counterexamples must include repeated same-named calls on one
line, different hop argument lists, script/tag expressions, wrappers and genuine
spanless/synthetic calls. Reading the first same-named call on the line would
violate the invariants already documented in CLAUDE.md.

This establishes a necessary blocker for the 65-entry cohort, not a 65-call gain.
After the omission is proven, later unknown owners or supplied values may still
prevent inference. Other chain span gaps may affect many owners; that population
has not yet been measured. Capture the complete competing caller set and the next
veto before crediting an impact estimate.

Evidence: `mura-changeset-caller-trace.log`, carch.cfc:561–567,
changesetBean.cfc:151, and `internal/parser/script_parser.go`'s
recordChainContinuationFrom/addCall together with `internal/parser/call_spans.go`.
No span fix has been implemented in this follow-up.

### Updated priorities and limits of this attribution

| Priority | Work to establish before implementation | Confidence / measured impact |
| --- | --- | --- |
| 1 | Survey missing byte identity in chained call records; prove a representative parser path and enumerate later competing caller blockers | Concrete observed gap; 65 downstream changeset entries, no verified gain yet |
| 2 | Define automatic engine-context mapping boundaries and explicit overrides for Lucee `/admin` | Configuration scenario removes exactly 99 entries; current default unchanged |
| 3 | Describe selector-dependent return alternatives and identity through Mura managers, caches and iterators | Concrete bean/array counterexample; 104 feed entries remain blocked, no safe global type assumption |
| 4 | Reuse the existing Wheels adapter publication/reload research and connect its open writer/retained-state obligations to the current exact cohorts | Historical common-API and return-annotation probes do not establish current stored-object identity |

The three exact current cohorts above are disjoint: 99 Lucee helper entries,
104 feed entries and 65 changeset entries, totalling 268 entries (2.6% of 10,235).
Only the 99 are shown to disappear under a tested configuration scenario. The
others have observed blockers, not completed resolution proofs. The 140 Lucee
field entries are a separate investigation cohort and are unchanged by that
scenario. These figures still do not constitute a 30% implementation plan.

The stronger pattern is a cascade: missing producer/result or lexical-identity
facts can veto inference at a known caller, leaving an entire downstream method
untyped. Conversely, a declared method in an existing helper becomes resolvable
as soon as its execution path is connected. We should preserve and expose those
intermediate facts, including ambiguity, instead of treating the final unknown
receiver message as a single universal failure mode.

Validation for this follow-up: the scenario's off-switch control is identical
per diagnostic to checkpoint 29; its configured diff has 99 removals/zero
additions with unchanged coverage and base-summary counts. The working caller
source hash matches the pre-probe hash. Only research documentation and ignored
artifacts changed. No production test or runtime result is claimed.

## Implementation follow-up: checkpoint 30

The chain-origin and Lucee source-context findings are implemented. Default total:
10,136, down 99; 292 diagnostic pairs correct hop locations. Two duplicate extracted
getSite records are removed, retaining the real calls. Missing-base coverage is
unchanged. The changeset chain omission is now proven, but another uncertain
caller still blocks the 65-entry cohort. The heterogeneous feed return is also
withheld. See checkpoint 30 of RESOLUTION-IMPLEMENTATION-STATUS.md for the complete
per-entry, coverage and validation audit; earlier research-only figures above
remain historical. No broader return/state or native API rule was implemented.

## Traced and left blocked (25 October 2026)

Each group below was traced to its root and left reported on purpose: the fix
would need a guess, or a guard that protects other cases. Counts are from the
scans of that date. The Mura iterator element and cfwheels' JDBC chain are in
[MURA-TEMPLATE-PARAMS-RESEARCH.md](MURA-TEMPLATE-PARAMS-RESEARCH.md) step 3 and
[RESOLUTION-NATIVE-TYPE-RESEARCH.md](RESOLUTION-NATIVE-TYPE-RESEARCH.md).

- **cfwheels `ssh` (49) and `variables.sshPool` (23)**, in the deploy CLI.
  `ssh` is a callback parameter: `variables.sshPool.onEach( hosts,
  function( ssh, host ) { ssh.run( … ) } )`. Two blocks, either enough:
  - `DeployMainCli.init( any sshPool = "" )` defaults the pool to a string,
    so the pool has no component type, and a caller omitting it hands in "".
  - Even typed, the callback runs where nothing can follow it. The real
    `SshPool.onEach` stores the closure in an `SshPoolTask` and submits it to
    a Java executor through `createDynamicProxy`; the task's `call()` invokes
    `callback( pool.getConnection( host ), host )`. The callback rule
    (`callbackParameterComponent`) reads an invocation in the provider's own
    straight-line body, and treats escapes and dispatch inside loops as
    unknown, which also covers `FakeSshPool.onEach`'s `for` loop. Following a
    closure through object storage and a Java executor is not something the
    resolver models.
- **cfwheels `variables.helpers` (55)**, in the CLI's service classes
  (`Admin`, `Templates`, …). `init( helpers )` stores the argument, and both
  constructors now type it (`getService( "helpers" )` since the keyed lazy
  cache rule, `new Helpers()` in the specs). The constructor scan
  (`inferInitArgument`) still declines: it treats any quoted string ending in
  the class name as a possible construction by name it cannot read, and
  `Admin.cfc` holds `"/app/controllers/admin"`. Exempting switch labels,
  comments and keys passed to the file's own functions was tried; each
  further exemption weakens the guard against factory construction
  (`getInstance( "Admin" )`, including through a variable), so it was left
  as it is.
- **Mura `arguments.feedBean` in feedGateway.cfc (111).** `getFeed( feedBean )`
  has two callers in feedManager.cfc: `getFeed()`, which types, and
  `getFeedIterator()`, which hands on its own `arguments.feedBean` and is
  called nowhere in the corpus. Every caller must type the argument, and
  skipping a caller nothing calls by name would break
  `TestACallerPlacedByTheParseIsACaller`: a framework or plugin may call a
  public method by convention.
- **Mura `entity` (100)**, beanFactory.cfc and apiUtility.cfc.
  `entity = getBean( arguments.entityname )`: the bean is named at run time.
- **Mura event values (164)**, "method … in mura.servletEvent|mura.event has
  no component return type", mostly in standardEventsHandler.cfc and
  contentRenderer.cfc: chains on `event.getValue( "contentBean" )` (140 of
  the getValue calls name that key), `event.getSite()` and
  `event.getHandler()`. A servletEvent's getValue reads the request scope,
  which `servletEvent.init()` fills with `structAppend( request, url, "no" )`
  and the same for form, so any URL or form parameter can be an event value;
  contentServer.cfc also writes event values under a computed key
  (`event.setValue( currentParam, … )`). Typing a key from its known writers
  would assume no unseen one. `getSite()` returns
  `settingsManager.getSite()`, which reads `variables.sites[ siteid ]`, a
  cache filled under computed keys.
- **cfwheels missing components (158)**, "component … does not exist", in
  two shapes, both a matter of project configuration rather than a rule:
  - `modules.wheels.services.deploy.…` and the like (about 90) in the CLI
    under `cli/lucli`, which names its own components as LuCLI's installed
    `wheels` module. Nothing in the source says `cli/lucli` is
    `modules/wheels`: a build step installs it there. A `mappings` entry in
    `.clif.json` would resolve them.
  - `wheels.middleware.RateLimiter` (40) in `tools/article-tests`, which does
    exist at `vendor/wheels/middleware/RateLimiter.cfc`. The mapping is
    `public/Application.cfc`'s `this.mappings["/wheels"]`, and
    `tools/article-tests` is outside `public/`, so no Application.cfc
    governs it; which mappings its harness runs under is not in the source.
