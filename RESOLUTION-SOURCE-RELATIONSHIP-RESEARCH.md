# Source relationships after the twenty-idea review

11 October 2026. Research overlays and documentation only. Production remains
**13,346 strict / 8,575 with both speculative flags**. This batch follows concrete
source gaps instead of extending proximity guessing. It completes **42 comparative
corpus scans, one acceptance-audit scan and 120 current isolated probe scans**.
The initial, superseded prototypes also retain 56 successful probe scans.

## Measured comparisons

| Condition | Before | After | Removed / added |
| --- | ---: | ---: | ---: |
| Callback constructors / strict | 13,346 | 13,300 | 46 / 0 |
| Callback constructors / both flags | 8,575 | 8,533 | 42 / 0 |
| Refined literal arrays / strict | 13,346 | 13,346 | 0 / 0 |
| Refined literal arrays / both flags | 8,575 | 8,575 | 0 / 0 |
| Scalar + hops + cautious fallback / strict | 13,346 | 15,899 | 16 / 2,569 |
| Scalar + hops + cautious fallback / both flags | 8,575 | 10,456 | 60 / 1,941 |

All comparisons preserve **8,151 files / 189,154 checked records**, with identical
missing-base entry multisets representing **46 summaries / 1,133 unchecked calls**.
Deltas compare diagnostic multisets, including reason changes and multiplicity.
There is no aggregate claim that the three approaches' removals can be added.

## Top-level callback constructors: a source-based improvement

The previous proximity experiment selected article-tests/Probe.cfc for 36 `rl`
findings even though the source constructs wheels.middleware.RateLimiter.
The new prototype recovers the constructor relationship instead.

The missing link has two parts. scanClosureBody deliberately reads statements
only inside a named function, leaving top-level anonymous callbacks in its older
call-only mode. Consequently these locals produce no ComponentRefs. A resolver
fallback alone cannot recover references the parser never recorded; its isolated
controls remain unresolved.

The second prototype reads top-level closure statements, stores their references
in the closure-ref bucket, and preserves closure line/byte visibility. It leaves
calls attributed to their existing global context. funcScopedRef adds a whole-file
fallback only after finding no enclosing named function; this fallback accepts
only references carrying closure byte ranges. Ordinary global refs remain on
the existing lookup path. This is an ignored prototype, not a shipped parser rule.

The full-corpus result removes **46 strict findings, with zero additions**. All
are in Wheels article test files:

| Receiver / method | Strict removals |
| --- | ---: |
| rl.handle in run.cfm | 30 |
| rl.handle in edge-cases.cfm | 6 |
| limiter.handle | 4 |
| sh.$headers | 3 |
| pipeline.run | 2 |
| probes.tryRateLimiterProxyStrategy | 1 |

An instrumented scan with **both Wheels roots** matches the prototype strict
report per entry and checks all 46 removals through ResolveCallTarget. All 46
reach **source component declarations** with empty failure reasons; none uses
speculative guessing, a builtin target or dynamic acceptance. RateLimiter.handle
points to vendor/wheels/middleware/RateLimiter.cfc. This verifies static source
connections, not execution or runtime ownership in every possible invocation.

A separate explain run using only the project root recovers the RateLimiter
receiver but cannot resolve its component. That run excludes the explicitly
supplied vendor/wheels root and is not equivalent to corpus coverage. The retained
trace records this difference; it does not contradict the two-root acceptance
audit or justify silently changing coverage.

With delivered speculative flags, the prototype removes 42 findings, with no
additions. Four strict removals were already accepted or otherwise represented
under the existing speculative policy; the totals alone do not identify a
separately verified four-entry accuracy improvement.

### Remaining callback boundaries

Isolated siblings and nested callbacks preserve the incompatible Other.ready
finding while resolving Product.ready. Reassignment to Other remains a missing
method, and using the callback-local name outside its callback stays unknown.
Existing focused byte-identity, provider-argument, mutation and arrow-provider
regressions pass under the overlay.

A different probe shows **an existing parameter-shadow gap**: a top-level callback
parameter named item can borrow the outer Product variable. Production, the
resolver-only fallback and the constructor prototype all accept ready in this
case. The new rule does not fix it. Parameter ownership needs its own binding and
shadow barrier; a same-named outer reference cannot supply it merely because the
callback has no named FuncScope. Named functions, arrow expressions, tag/script
boundaries, catch variables, explicit shared scopes, closure escapes and same-line
calls need a broader regression matrix before implementation.

## Literal array elements: useful infrastructure, no measured corpus gain

A first prototype recognizes arrays of direct constructor expressions and
requires every element to name the same resolvable component. It resolves both
loop and native each-callback controls through the existing shared elementOf
path. A mixed Product/42 array remains unknown.

That first version ignores mutation after initialization and does not examine
constructor return behavior. Its retained counterexamples accept ready after
items.append(42), and accept the constructed Product type even when init returns
Other. The latter exposes a constructor-model assumption; no CFML engine run is
claimed to establish its runtime behavior.

The refined version withholds literal inference if a constructor has any init
method, and rejects intervening uses of the collection name, includes or calls
between its local literal assignment and the loop/callback header. This is a
conservative experimental restriction, not a general effect analysis. It resolves
the two simple controls and retains primitive, mutation and constructor-return
findings. It does not handle finite heterogeneous alternatives, nested records,
factory-created elements, arbitrary aliases or every mutation during iteration.

The refined overlay is **identical to production per entry across all seven
corpora**, in strict and both-flags modes. No gain is credited to this infrastructure.
The larger Lucee Field/Group cohort uses helper-created elements and driver
collections, rather than this closed literal shape. The next useful collection
work is producer element contracts and finite alternatives, followed by branch
narrowing and ownership/effect checks. Homogeneous literal inference alone does
not reach that cohort.

## Scalar factory returns: three connected failure points

The earlier Factory.make(kind) probe returns Product for "product" and Other
otherwise. Production uses inferred ReturnComponent=Other for both calls.
This batch identifies three separate relationships required to improve it:

1. **Recognize the supported scalar predicate.** A narrow prototype recognizes
   lexical-path versus string-literal equality/inequality, marks that source plan
   for specialization and evaluates the comparison when both values are known.
   Bare/scoped variables, unsupported compound expressions and unknown values
   retain uncertainty rather than selecting a branch without evidence.
2. **Retain uncertainty through the parser.** walkChainRest and baseVarComponent
   currently turn an empty lookup answer into $any, which later accepts methods
   without checking a destination. The cautious prototype withholds that fallback
   in those two paths. This is broader than the scalar-factory fix and affects
   many existing contracts; explicit dynamic paths elsewhere remain available.
3. **Preserve arguments after an inline constructor.** scanChainedCalls stores
   argument-bearing hops for call sites, but bare method names for the assigned
   receiver's ChainRest. Thus `factory.make("product")` can specialize while
   `new Factory().make("product")` loses the selection argument. The final
   prototype retains the argument-bearing hop in both representations.

The intermediate scalar prototype resolves the valid call through a variable
factory but also accepts unknown arguments and the wrong inline-constructor
branch through dynamic fallback. Withholding fallback restores uncertainty but
leaves inline construction unresolved. Preserving its arguments finally makes
both known Product shapes resolve, while Other.ready and unknown-argument ready
remain findings in the isolated controls. This is a source-based path through the
probe, not proof that every supported string comparison has correct engine
semantics or that the entire corpus combination is safe to ship.

### Why the scalar corpus numbers rise

The final prototype combines predicate specialization, argument preservation
and a broader change to unknown-return fallback. Its strict comparison removes
16/adds **2,569**, ending at **15,899**. With speculative flags it removes 60/adds
1,941, ending at 10,456. These additions are not automatically runtime defects,
confirmed false positives or regressions to discard. They expose dependencies on
existing return inference and acceptance policy under this combined experiment.

Prominent newly reported groups include Mura loadsiteflat item (111), entity-test
widget (66), API processRequest token (55), Wheels validation user (40), and
ColdBox scheduled/async service receivers (35/34). Some methods legitimately
produce dynamic or native values; others need object, collection or record
contracts. The artifact scalar-cohorts.json retains source entries and counts.

The 16 strict removals include nine Injector mapping calls, six Mura approval
content calls and one Wheels validation argument call. They are **not all credited
to literal scalar selection**: changing fallback and hop arguments can alter
caller inference and competing relationships. Unlike the callback removals,
these do not have a definitive acceptance/origin audit in this batch.

The next experiment should separate the three changes, rather than treat the
combined total as a standalone scalar-rule score. Distinguish explicit dynamic
contracts from failed analysis, and classify the restored entries before proposing
an opt-in diagnostic mode or changing default acceptance. A lower count obtained
by converting an unknown return to $any is not improved source resolution.

## Current isolated probe matrix

Counts below are focal ready findings; other diagnostics are retained. Each
focal call is checked against parser extraction and verbose accepted records,
so zero is not attributed to silently dropped calls.

| Probe | Production strict | Callback prototype | Refined arrays | Final scalar prototype |
| --- | ---: | ---: | ---: | ---: |
| callback-escape | 2 | 2 | 2 | 2 |
| callback-local | 1 | 0 | 1 | 1 |
| callback-nested | 2 | 1 | 2 | 2 |
| callback-parameter-shadow | 0 | 0 | 0 | 0 |
| callback-siblings | 2 | 1 | 2 | 2 |
| literal-callback | 1 | 1 | 0 | 1 |
| literal-init-other | 1 | 1 | 1 | 1 |
| literal-loop | 1 | 1 | 0 | 1 |
| literal-mutated | 1 | 1 | 1 | 1 |
| literal-primitive | 1 | 1 | 1 | 1 |
| scalar-inline-other | 1 | 1 | 1 | 1 |
| scalar-inline-product | 1 | 1 | 1 | 0 |
| scalar-other | 1 | 1 | 1 | 1 |
| scalar-product | 1 | 1 | 1 | 0 |
| scalar-unknown | 1 | 1 | 1 | 1 |

The full eight-condition matrix also includes the failed resolver-only fallback,
initial array and initial/intermediate scalar prototypes. A parameter-shadow
zero is a limitation identified above, not a passing ownership check.

## Proposed implementation order

First prepare a production-sized top-level closure rule with explicit parameter
ownership and shadowing regressions, preserving byte identities and all global
call records. Recover constructor origins rather than introducing proximity
selection. Re-run its source-target audit and corpus diff after the final design.

Next isolate scalar argument-bearing hops from the acceptance-policy changes.
Retain literal argument context through assignments, direct chains, wrappers and
named calls, then add narrowly supported comparison evaluation with positive,
negative and unknown controls. Keep documented and explicit dynamic contracts
separate from an inference failure.

For collections, prioritize helper/producer element contracts and finite
alternatives. Reuse those contracts for loops and callbacks, then eliminate
terminating guard branches only when the full set of element alternatives is
known. Record-field value propagation and mutation/alias tracking are shared
prerequisites. None of the unimplemented population counts is credited as a fix.

## Evidence and checks

Artifacts are in `target/resolution/source-relationships/`: versioned probes,
all overlays and binaries, per-entry corpus diffs, acceptance targets, parser
reference dumps, explain traces, source hashes and comparison scripts. Current
120 probe scans pass focal extraction/observation assertions. All six comparative
conditions retain exact file/record/missing-base coverage; the audit equals the
callback strict report per entry.

Research binaries compile. Focused existing callback, collection and producer
regressions pass under their respective overlays. Seven production source-file
hashes remain unchanged. `git diff --check` passes. No full production suite,
CFML engine execution or published CI result is claimed for this research batch.

## Implementation follow-up

The [liberal implementation and audit](RESOLUTION-LIBERAL-IMPLEMENTATION.md)
implements the callback rule with parameter ownership, direct top-level reference
storage, and factory-local scoping regressions. Constructor-chain argument
preservation is implemented independently of scalar predicate specialization
and broad unknown-return fallback changes. The scalar combined experiment and
literal-array inference remain research. Ranked matching and open-world
classification are separately enabled policies with retained audit evidence.
The research measurements above describe their original overlays, not the final
production behavior; use the follow-up report for current numbers and checks.

## Collection callback follow-up

See [collection callback invocation contracts](RESOLUTION-COLLECTION-CALLBACKS.md)
for the next implemented source relationships, speculative provenance guard,
per-entry corpus review, and remaining collection ownership gaps.

## Receiver and return opportunity follow-up

See [implemented collection and ORM API opportunities](RESOLUTION-OPPORTUNITIES-IMPLEMENTED.md)
for complete-chain inference, preserving operations, generated shared ORM
contracts, target audits, and the latest corpus counts.

## Collection key and value follow-up

See [collection keys and returned struct values](RESOLUTION-COLLECTION-VALUES.md)
for delivered key-array elements, retained dynamic-scope origins, separate struct
value contracts, edit invalidation, current measurements and remaining boundaries.
