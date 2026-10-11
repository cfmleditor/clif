# Receiver roles and remaining resolution opportunities

11 October 2026. Research after checkpoint 34. Production remains **13,346
strict / 8,575 with both speculative flags**. This batch changes documentation
and ignored research overlays only; it delivers no resolver changes.

## Measured result

Fourteen successful corpus scans compare two overlays over seven projects.
The role-aware condition uses complete candidate lists and unique score ranking,
as in the previous full-candidate experiment, but separates original receiver
methods from methods on returned objects. It does not relax caller consensus.

| Condition | Findings | Removed / added against delivered both flags | Net reduction |
| --- | ---: | ---: | ---: |
| Delivered both speculative flags | 8,575 | — | — |
| Earlier complete candidates + ranking | 8,017 | 571 / 13 | 558 |
| Role-aware complete candidates + ranking | **7,593** | **1,060 / 78** | **982 (11.45%)** |
| Same roles, requiring two distinct observed base methods | 10,299 | 623 / 2,347 | -1,724 |

The role-aware condition removes 523/adds 99 against earlier full-candidate
ranking, net **424**. It also leaves 140 fewer findings than the previous
7,733 caller-relaxation combination, but these are different experiments with
different removal sets. Their gains must not be added. Diagnostic reductions
are not verified runtime fixes.

| Corpus | Delivered both flags | Role-aware | Two-method condition |
| --- | ---: | ---: | ---: |
| wheels | 1,737 | 1,545 | 2,101 |
| masa | 2,429 | 2,069 | 2,369 |
| contentbox | 1,120 | 899 | 1,221 |
| coldbox | 906 | 788 | 1,311 |
| lucee | 1,854 | 1,812 | 2,656 |
| testbox | 261 | 259 | 365 |
| fw1 | 268 | 221 | 276 |

All conditions preserve **8,151 files / 189,154 checked records**, with exact
missing-base entry multisets representing **46 summaries / 1,133 unchecked
calls**. Each change is compared per entry with multiplicity using file, line,
function, reason, caller, variable and unchecked count. Artifact scripts, source
hashes, binaries, logs, diffs and review.json are in
`target/resolution/receiver-role-research/`.

## What the role correction actually does

For variable-category candidates, an unchained call constrains its original
receiver with the direct method. A chained call constrains that receiver with
its **first method hop**, not the final method. Thus `value.getProduct().ready()`
asks which original objects declare getProduct; ready belongs to the returned
object. Candidate generation gathers these base methods across parser sites
in the same file/function/receiver group. Ranking uses the original receiver
name. Return-type-category suggestions retain their existing behavior.

The speculative native method list also uses first hops for unknown base
receivers. First property hops and unmatched sites withhold the entire group's
promotion because method guessing cannot establish a field's owner/value.
After selecting a receiver, normal source/native return walking checks chains.
There is no new chain-success filter or dynamic-acceptance veto.

The earlier four-file counterexample now retains both unknown findings. Factory
and Product both declare getProduct; Product alone declares ready. Old ranking
used ready to select Product as the original object. Role-aware ranking leaves
Factory/Product tied and declines that guess. This is a scanner counterexample,
not CFML execution or an estimate of corpus error frequency.

The two-method condition requires at least two distinct first-hop/direct method
names among baseline unknown findings, before either CFC or native guessing.
It is applied to all groups, including ones production currently guesses. Its
higher total therefore includes restored existing guesses; it is not evidence
that the new role algorithm introduces 2,347 production regressions. It also
does not prove origin and should not be treated as a confidence threshold.

## Origin remains a separate problem

A second isolated four-file probe deliberately supplies `42` to
Service.use(product), which calls product.reset() and
product.getProduct().ready(). Factory and Product declare reset/getProduct;
Product alone declares ready. Name affinity breaks the base-candidate tie in
favor of Product. **Both role-aware and two-method conditions accept all three
method records despite the visible numeric supplier.** The strict control
retains all three findings. More compatible methods do not override contrary
origin evidence.

The first attempt placed this probe under the repository's target directory,
where inherited `.cfmleditor.json` changed indexing. That attempt was discarded;
the measured probe runs from an isolated `/private/tmp` directory and indexes
its four files. The retained path, control and experiment logs record this
boundary. Empty JSON stdout accompanies the CLI's successful zero-findings
result; stderr records the accepted methods and guess.

The next selection contract needs separate evidence fields: original receiver
identity, method compatibility, reached return object, source suppliers and
contradictions. Known incompatible suppliers must veto a usage guess. Unknown
suppliers should retain an explicit uncertainty label. Assignment changes,
closures and different values sharing a receiver spelling remain outside this
prototype's file/function/variable grouping.

Scores still accumulate existing high/medium/low labels across diagnostics;
repeated calls can overweight repeated evidence. These labels are preferences,
not calibrated probabilities. Candidate generation can omit non-focal builtin
member names from CFC constraints under the existing policy. Native constraints
come from unresolved sites rather than all observed base calls. Neither evidence
set establishes every use of the value.

## Additions expose downstream gaps and competing guesses

Role-aware additions include Mura contentBean returns getDisplayInterval and
getParent; ContentBox BaseContent.getActiveContent; and ColdBox guesses involving
sharedArgs.copy and matches. Some become return-type/method findings after the
base is guessed; others restore unknown origins when the prior CFC/native choice
changes. A changed reason must be reviewed rather than counted as a solved call.

Remaining role-aware findings are **4,656 variable, 1,252 return-type, 1,225
method and 460 object**. Unknown receivers are still the largest category.
The 104 feedGateway.getFeed / arguments.feedBean group remains unresolved;
this role correction does not reconcile its caller alternatives. The previous
caller-relaxation disappearance is still unsupported by consensus.

## A concrete collection opportunity: Lucee admin Field/Group

The delivered report contains **178 unresolved `field` findings in five admin
creation templates**: AI 37, logging 36, gateway 36, cache 35, debug 34. All five
sources contain a loop over driver.getCustomFields(), a positive
isInstanceOf(field,"Group") branch, and cfcontinue. The artifact
`lucee-field-cohort.json` records each file, method count and source markers.
This is a scoped investigation population, not 178 measured fixes.

In the sampled AI family, AI.cfc creates sibling Field and Group objects through
field()/group(), returns variables.fields, and OpenAI.cfc extends AI and stores
both helper results in that array. Field declares getName/getDefaultValue/
getType/getRequired/getValues as well as display/description methods. Group
has display/description/getLevel. Source therefore describes a mixed collection,
not a single global class named field. Existing positive guarded receiver
handling does not establish the complementary type after cfcontinue; existing
collection inference needs the heterogeneous element relationship first.

A source-grounded plan is to propagate finite element alternatives through
helper returns, subclass array assignments and getCustomFields(); narrow Group
in its branch; then remove Group from the surviving loop body after a guaranteed
continue. Bare Group must resolve within the driver family, respecting duplicate
Field/Group names in other families. Family lookup, collection alternatives and
control flow are distinct prerequisites. The sampled AI chain is reviewed;
other driver producers still require tracing before claiming all 178 are covered.

Counterexamples must include a third/custom element type, conditionally executed
continue, reassignment, shadowed isInstanceOf, multiple driver families, and
missing producer source. Excluding Group alone cannot prove Field if the full
collection alternatives are unknown. Configuration may declare an explicit
provider/element contract when source is unavailable; a global name convention
would obscure this distinction.

## Review and next priorities

1. Preserve complete internal candidate sets without expanding every public
   report. Keep original and returned receiver roles explicit.
2. Add an origin-contradiction veto before treating method evidence as eligible
   for an opt-in guess. Review dynamic return acceptance separately.
3. Investigate finite collection alternatives and terminating-branch narrowing
   using the 178-entry Lucee cohort before widening name heuristics.
4. Continue tracing supplier alternatives for the largest Mura groups; retain
   caller-relaxation counterexamples and report discarded constraints explicitly.

Research binaries compiled; both isolated probes completed; all 14 corpus
comparisons and coverage assertions passed. Production source hashes are
unchanged. No production test suite, engine execution or published CI result
is claimed for this documentation/overlay batch.


## Twenty-idea research follow-up

[The twenty-idea review](RESOLUTION-TWENTY-IDEA-RESEARCH.md) completes 49 corpus
scans and 112 paired structural probe scans. Proximity leaves 6,824 experimental
findings, but selects Probe for explicitly constructed RateLimiter receivers
in 36 removed corpus entries. Role + caller relaxation leaves 7,315. The study
identifies scalar factory return precedence, literal collection elements, union
guards, record fields and closure lifetime as connected source-based targets;
blanket if specialization is rejected after dynamic acceptance and corpus losses.
Production remains 13,346 strict / 8,575 with both flags; no new policy is delivered.
