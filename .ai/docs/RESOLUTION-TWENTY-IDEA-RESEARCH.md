# Twenty resolution ideas: experiments and review

11 October 2026. Research after the receiver-role study. **Twenty distinct idea
iterations**: six corpus selection experiments and fourteen paired structural
probes. Iteration 11 also receives a deliberately broad branch-specialization
corpus experiment after its probe exposes a return-selection problem. Production
remains **13,346 strict / 8,575 with both speculative flags**. No production code
or configuration is changed.

## Corpus experiments: iterations 1–6 and follow-up to 11

Each policy starts with the previous role-aware, complete-candidate overlay,
which left 7,593 findings. Corpus deltas below also compare with delivered
both-mode 8,575 so the baseline is explicit. Candidate preferences remain
unreliable; lower totals are not verified runtime fixes.

| Iteration / condition | Findings | Removed / added vs 8,575 | Net vs 8,575 | Removed / added vs 7,593 role control |
| --- | ---: | ---: | ---: | ---: |
| 1. Deduplicate scores | 7,593 | 1,060 / 78 | +982 | 0 / 0 |
| 2. Require name affinity | 9,908 | 749 / 2,082 | -1,333 | 34 / 2,349 |
| 3. Two-method CFC selection | 8,586 | 730 / 741 | -11 | 32 / 1,025 |
| 4. Native before CFC | 7,598 | 1,060 / 83 | +977 | 0 / 5 |
| 5. Proximity tie-break | 6,824 | 1,818 / 67 | +1,751 | 774 / 5 |
| 6. Role + caller relaxation | 7,315 | 1,340 / 80 | +1,260 | 280 / 2 |
| 11. Blanket if specialization | 9,216 | 1,120 / 1,761 | -641 | 60 / 1,683 |

Proximity is the largest diagnostic reduction: **1,751 (20.42%)**, but has
source-origin counterexamples. Role + caller relaxation leaves **7,315**, net
**1,260 (14.69%)**. These gains overlap and cannot be added.

1. **Deduplicate method evidence.** Intersect candidates for every finding, but
   add preference scores only once per distinct original-receiver method. This
   tests whether repeated calls bias selection. The output equals the earlier
   role-aware report per entry. In this prototype, each site's candidate evidence
   is already based on the same grouped base methods, so this score correction
   does not yield new resolution. Keep deduplication as an evidence-design issue,
   not a measured gain.
2. **Require receiver-name affinity.** Keep only CFC candidates whose existing
   basis says they are named like the receiver; retain ordinary native fallback.
   This removes unnamed sole candidates as well as ranked choices. Restored
   findings show the cost of making naming a prerequisite; a matching name still
   does not prove origin.
3. **Require multiple methods for CFC guesses only.** Require two distinct base
   method names for CFC selection, while preserving native fallback. Unlike the
   preceding all-group two-method experiment, this does not disable single-method
   native guesses. It distinguishes CFC evidence requirements from native family
   eligibility, but neither proves the supplied value.
4. **Try native selection before CFC selection.** Replace a CFC winner with a
   unique native family when available. The resulting diagnostic changes expose
   competing source/native interpretations rather than new origin evidence.
   Native-family compatibility should not silently overrule source suppliers.
5. **Break score ties with source proximity.** Among equally scored candidates,
   choose a unique nearest directory distance; equal-distance ties remain
   unresolved. The isolated counterexample leaves Factory/Product ambiguous
   under role ranking but picks nearby Product under this policy, accepting
   getProduct/ready even though the visible caller supplies 42. Its reductions
   cannot be promoted as trusted fixes.
6. **Combine role-aware selection with relaxed caller consensus.** Use the
   previously researched overlay that ignores unknown-owner/value suppliers
   while retaining at least one typed supplier. Measure the combination rather
   than adding independent gains. Existing supplier-uncertainty counterexamples
   still apply; the paired probes do not establish safe consensus.

| Corpus | Delivered both | Proximity | Role + caller relaxation |
| --- | ---: | ---: | ---: |
| wheels | 1,737 | 1,198 | 1,537 |
| masa | 2,429 | 1,904 | 1,821 |
| contentbox | 1,120 | 839 | 882 |
| coldbox | 906 | 713 | 787 |
| lucee | 1,854 | 1,765 | 1,812 |
| testbox | 261 | 254 | 258 |
| fw1 | 268 | 151 | 218 |

## Structural probes: iterations 7–20

Every iteration has an isolated control and an opaque, contradictory or altered
variant. Each runs strict, delivered both-flags, role-aware and relaxed-caller
modes: **112 successful probe scans**. Parser extraction and verbose accepted
calls are checked against focal findings for every run, so a zero here does not
mean the scanner silently dropped the method. The table counts only ready/other
method findings; other diagnostics are retained in the artifacts. All three
speculative modes have the same focal totals for these paired probes.

| Iteration / idea | Strict control / variant | Speculative control / variant |
| --- | ---: | ---: |
| 7. homogeneous collection | 1 / 1 | 0 / 0 |
| 8. heterogeneous collection | 1 / 1 | 1 / 1 |
| 9. terminating union branch | 1 / 1 | 0 / 0 |
| 10. post-return guard | 1 / 1 | 0 / 0 |
| 11. argument-sensitive factory | 1 / 1 | 1 / 1 |
| 12. collection callback | 1 / 1 | 0 / 0 |
| 13. closure capture lifetime | 0 / 0 | 0 / 0 |
| 14. assignment lifetime | 0 / 1 | 0 / 1 |
| 15. record property path | 1 / 1 | 0 / 0 |
| 16. generated accessor return | 0 / 1 | 0 / 1 |
| 17. fluent self return | 0 / 1 | 0 / 1 |
| 18. forwarded argumentCollection | 0 / 1 | 0 / 0 |
| 19. include context contract | 0 / 1 | 0 / 0 |
| 20. caller origin contradictions | 0 / 1 | 0 / 0 |

7. **Literal homogeneous collection elements.** Strict cannot connect an array
   containing new Product() to a loop's item.ready(). It also leaves the opaque
   external collection unknown. Speculation accepts both. Source inspection of
   elementOf confirms literal array expressions have no element-inference path.
   A useful strict rule would evaluate every element, require compatible known
   object alternatives and retain unknown/primitive alternatives.
8. **Finite heterogeneous collections.** Product and Other both declaring ready
   still leave the strict loop unresolved; speculation also declines the tied
   candidates. The primitive-containing variant remains unresolved. This exposes
   the need to check a method against every proven collection alternative, rather
   than demanding one component identity or choosing any compatible member.
9. **Eliminate a union branch after continue.** A Product/Group collection with
   a Group guard and continue leaves ready unknown. Adding 42 as a third element
   produces the same result under strict and is still accepted by speculation.
   Excluding Group cannot prove Product without the full element set. This
   supports the previously documented 178-entry Lucee cohort, without crediting
   those 178 entries as fixes.
10. **Propagate a terminating negative guard.** After
    `if(!isInstanceOf(item,"Product")){return;}`, strict still leaves ready
    unknown. Removing return also leaves it unknown. Speculation accepts both.
    Positive guarded receiver support exists; complement/termination handling
    is a separate rule and must honor shadowing, writes and escapes.
11. **Specialize a scalar-controlled factory return.** make("product") and
    make("other") both resolve their assigned receiver to Other under production.
    Only the latter matches the source branch. Parser evidence records
    ReturnComponent=Other; explain reports that concrete receiver. Multiline
    formatting, an arguments.kind predicate, a typed string parameter, direct
    chaining and a variable factory receiver all preserve the wrong branch
    selection. A switch version and a single-final-return version accept the
    valid Product call. This is a missing relationship between branch semantics,
    provisional parser returns and call-specific source evaluation.
12. **Infer collection callback parameters from elements.** An each callback on
    `[new Product()]` stays unknown under strict; `[42]` does too. Speculation
    accepts both. collectionCallbackElement already delegates to elementOf, so
    literal collection inference would serve loops and callbacks together.
13. **Respect deferred closure value lifetime.** Both strict controls accept
    a captured receiver declared as Product. A later replacement with Other
    before invoking the closure also accepts ready. Explicit variables scope
    preserves the same behavior. Parser/explain evidence shows lookup at the
    closure body's source position chooses the earlier Product assignment.
    This deserves a lifetime audit; the probes execute the scanner, not CFML,
    and do not establish runtime capture semantics or corpus error frequency.
14. **Keep assignment instances separate.** Product.ready followed by a new
    Other assignment and Other.other resolves both; Other.ready after replacement
    remains a method-not-found diagnostic in all modes. This existing behavior
    should be preserved. It does not justify grouping every value sharing a
    spelling into one speculative receiver.
15. **Carry direct record-field values.** `{product:new Product()}` followed by
    holder.product.ready remains unknown under strict. Replacing the field with
    42 also remains unknown; all speculative modes accept both. producerValue
    already carries field and primitive information for source evaluation, but
    this direct receiver path does not expose it. A value-aware lookup should
    distinguish known incompatible fields from missing evidence.
16. **Use generated accessor contracts.** A typed Product property with accessors
    resolves getProduct().ready already. getMissing().ready retains failures in
    every mode. This probe supports the existing contract; another broad accessor
    shortcut would duplicate working behavior. A property type is a static
    contract, not proof of initialization at runtime.
17. **Preserve fluent self identity.** Product.reset returning this reaches
    Product.ready already. The same chain on Other retains missing ready. The
    observed gap is not a universal inability to follow untyped return-this.
18. **Forward argumentCollection with provenance.** A typed Relay argument passed
    through argumentCollection=arguments lets strict type Service.use's parameter.
    An additional opaque collection supplier restores an unknown finding under
    strict; speculation accepts it. Keep typed forwarding and explicit unknown
    supplier constraints separate.
19. **Propagate include caller context.** An included template using an item
    initialized to Product resolves already. A numeric include caller restores
    the unknown finding, which speculation accepts. Further include work should
    retain caller alternatives and incompatible values instead of guessing from
    the included template's method names.
20. **Retain contradictory caller values.** A Service parameter supplied only
    Product resolves strictly; adding a numeric supplier makes it unknown.
    Delivered and research guesses still accept ready. The needed improvement
    is an explicit contradiction veto, not discarding the competing supplier.

## Iteration 11 follow-up rejects a tempting shortcut

A research overlay marks every if as sensitive so source analysis precedes
inferred returns. It accepts both factory probes, including the known Other
branch calling ready. Explain identifies the reason: the receiver becomes
**$any and is accepted without a method check**. This condition is deliberately
broader than a proposed production fix; it tests the consequences of changing
return precedence without preserving uncertainty.

The full corpus comparison is retained above, including restored diagnostics.
This blanket policy is rejected regardless of any removed findings. A narrower
plan must recognize supported scalar comparisons, bind actual arguments, retain
all surviving return alternatives, and distinguish an explicit dynamic contract
from analysis that could not produce a concrete object. An unknown analysis
result must not automatically become dynamic acceptance. Counterexamples need
both literal values, unknown predicates, reassignment, defaults, named arguments,
side effects and documented return contracts.


## Reviewing the largest proximity changes

Against role-aware ranking, proximity removes 774/adds five, net 769. The largest
new group is **30 rl findings** in Wheels tools/article-tests/run.cfm; another
six occur in edge-cases.cfm. The guessed component is **tools/article-tests/Probe.cfc**.
The source instead explicitly constructs **wheels.middleware.RateLimiter** for
these local rl values, and that class exists under vendor/wheels/middleware.
Probe shares handle but is a different source class. These 36 removals illustrate
wrong object selection inside the corpus, beyond the isolated numeric probe.
A future source fix should preserve callback-local constructor bindings and
resolve their component paths; it should not credit Probe method compatibility.

Other prominent choices include ModuleConfig.onLoad binder mapped to Wheels
Injector and Mura variables.fw mapped to a nearby controller declaring redirect.
Those require the module-host injection and framework supplier relationships
before they can be credited. The retained nearest-cohorts.json records groups,
examples and counts; no runtime error rate is inferred from this sample.

## Priority changes after twenty iterations

The strongest next source-based work is shared literal collection inference
for loops/callbacks, finite element alternatives, terminating branch narrowing,
record-field values and scalar factory branch selection. These form connected
relationships rather than independent name shortcuts. Existing accessors,
return-this, typed forwarding, includes and ordinary assignments already pass
focused controls and should retain those boundaries.

Receiver selection needs a value result that records **known object, known
primitive, finite alternatives, unknown origin and explicit dynamic contract**
separately. A missing component string currently loses distinctions needed for a
contradiction veto. Preserve supplier evidence, branch provenance and assignment
identity through that result before weighting names or proximity. The broad
all-if experiment demonstrates why changing precedence alone is insufficient.

The 178-entry Lucee Field/Group population remains a concrete source-tracing
cohort. This study does not credit additional corpus coverage to unimplemented
literal/union/guard rules. No combination of populations is advertised as a
30% verified gain. Candidate counts, speculative removals and runtime-correct
source resolutions remain different measurements.

## Reproducibility and validation

Artifacts are under `target/resolution/twenty-ideas/`: setup/runner/review scripts,
overlays, binary snapshots, source hashes, per-entry corpus additions/removals,
112 paired-probe reports, extraction/observation assertions, supplementary
factory/closure probes, explain traces and the proximity counterexample. Fixtures
are isolated under `/private/tmp`; saved path files identify them and avoid
inherited repository configuration.

All **49 corpus scans** completed over the same seven projects. Every condition
retains **8,151 files / 189,154 checked records**, with exact missing-base entry
multisets representing **46 summaries / 1,133 unchecked calls**. The reviewer
compares diagnostic multisets per entry, including reason changes and multiplicity.
Source hashes for all four overlaid production files remain unchanged. Research
binaries compiled; probe extraction/observation assertions and corpus coverage
assertions passed. `git diff --check` passes.

No production test suite, CFML runtime execution or published CI result is
claimed for this research/documentation batch. Default strict resolution and
opt-in production policies are unchanged.


## Source-relationship research follow-up

[The source-relationship study](RESOLUTION-SOURCE-RELATIONSHIP-RESEARCH.md)
measures top-level callback constructors, literal collection elements and scalar
factory relationships. The callback prototype removes 46 strict findings / 42
with both flags, with no additions and a source-declaration acceptance audit.
Refined literal arrays change no corpus findings. Scalar selection needs argument
preservation and separate uncertainty handling; its conservative combined policy
restores 2,569 strict findings and remains research-only. Production is unchanged.
