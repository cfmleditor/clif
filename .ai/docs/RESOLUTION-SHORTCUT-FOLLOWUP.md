# Follow-up research on speculative resolution

11 October 2026. Research after checkpoint 34; no production resolver, parser,
configuration or test changes in this batch. Strict remains 13,346. The measured
research control is the delivered **8,575 with both speculative flags**, covering
8,151 files / 189,154 checked records and 46 base summaries / 1,133 unchecked calls.
Overlays and review artifacts are under `target/resolution/shortcut-followup/`.

## What remains and where new gains come from

The delivered both-flags report contains 5,706 unknown-receiver findings in
2,072 receiver groups, 1,230 return-type findings, 1,179 method findings and 460
object findings. These categories identify the last failure, not its root cause.
Large remaining receiver groups include Mura feedBean (104), collection item
(60), admin audit item (42), API entity (42) and modal contentBean (37).

This follow-up measures interactions the earlier ten-pass study left open:
ranked CFC guessing now runs with the delivered staged primitive fallback,
complete candidate sets are used instead of treating displayed lists as complete,
caller ownership/value vetoes are relaxed on the current baseline, and an
instrumented control audits how existing guesses accept calls.

## Completed comparisons

All reductions below compare with **8,575**, not strict. They are experimental
diagnostic reductions, not verified runtime fixes. Conditions overlap.

| Condition | Remaining | Removed / added | Net reduction | Reduction |
| --- | ---: | ---: | ---: | ---: |
| Ranking + current staged primitive fallback | 8,148 | 435 / 8 | 427 | 4.98% |
| Ranking with chain-group guesses withheld | 8,367 | 365 / 157 | 208 | 2.43% |
| Ranking with complete candidate lists | 8,017 | 571 / 13 | 558 | 6.51% |
| Discard unknown-owner/value suppliers | 8,241 | 337 / 3 | 334 | 3.90% |
| Complete candidates + ranking + caller relaxation | 7,733 | 857 / 15 | 842 | 9.82% |
| Instrumented current-policy control | 8,575 | 0 / 0 | 0 | 0.00% |

The strongest combination leaves **7,733**, down **842 (9.82%)** from delivered
both-mode 8,575. It removes 857/adds 15. Its total is 42.06% below strict 13,346,
but the wider removal set remains unreliable. The independent full-candidate
and caller reductions sum to 892; the measured combination yields 842. The
difference reflects overlap/interactions, not a separately verified 50-entry
cohort. The combined condition has not been implemented.

| Corpus | Delivered both flags | Strongest combined experiment |
| --- | ---: | ---: |
| wheels | 1,737 | 1,614 |
| masa | 2,429 | 1,908 |
| contentbox | 1,120 | 994 |
| coldbox | 906 | 850 |
| lucee | 1,854 | 1,849 |
| testbox | 261 | 260 |
| fw1 | 268 | 258 |

All **42 successful scans** completed: six conditions over seven corpora.
Every condition retains 8,151 files, 189,154 checked records and identical
missing-base entries representing 1,133 unchecked calls. The instrumented
audit control is identical to production per entry.

The full-candidate condition changes the ignored research overlay's presentation
limit from eight to 1,000,000. The underlying candidate generator already counted
all candidates; this lets the experiment inspect complete lists. It still
intersects all findings and requires a unique score winner; tied scores withhold
promotion. Scores sum existing high/medium/low labels across findings, so repeated
observations can overweight the same evidence. Those labels are not probabilities.
A production design should expose complete candidate sets internally without
expanding every public JSON report to its full size.

Compared with capped-list ranking, complete sets remove another 136 findings and
add five, net **131**. The largest additional improvement is Mura. Candidate
listing limits therefore affect speculative eligibility, even though they do not
prevent strict source resolution. Removing the listing limit alone is not a
strict-resolution fix and can create competing CFC/native answers.

## The chain guard exposes a real tradeoff

The direct-only condition uses ranked CFC candidates but rejects every receiver
group containing a chained call or an unmatched parser site. It applies to CFC
and native guesses, preserving ordinary source resolution. All currently eligible
unknown sites were matched by the preceding parser probe, but this guard also
fails closed for future unmatched shapes.

Direct-only ends at 8,367, versus 8,148 with ranking alone: **219 more findings**.
It removes 365/adds 157 against the delivered control. Restored diagnostics cluster
in ColdBox, ContentBox and TestBox. This does not prove those earlier connections
were incorrect; it identifies dependence on chaining and a loss of inferred
context when a whole receiver group is withheld.

A separate four-file probe makes the wrong-origin risk concrete. Factory and
Product both declare getProduct; only Product declares ready. Service.use(value)
calls value.getProduct().ready(), and the visible caller supplies 42. The current
policy retains both findings. Ranking intersects the base/getProduct and
returned/ready candidate lists, guesses Product for the original value, and
accepts both calls. The direct-only guard retains both findings.

This probe executes the scanner, not CFML. It demonstrates a connection that
loses visible origin information and mixes receiver roles. It does not estimate
corpus error frequency. A declared method at the guessed destination does not
prove the caller supplied that object.

The next structural correction should identify roles before scoring:

- For an unknown original receiver, gather its direct methods and the first
  method hop of each chain. Do not use the final result's method as an original
  receiver constraint. Property hops need their own field/record policy.
- Walk source/native returns separately and check downstream methods at the
  reached object. Withhold or label dynamic/unknown return acceptance explicitly.
- Keep caller origin constraints, assignment/value identity and instance lifetime
  separate from usage compatibility. File/function/receiver spelling alone can
  group different values and closures.

These are proposed contracts, not implemented corrections. No measured gain is
credited to the proposed role-aware algorithm.

## Caller relaxation retains its known counterexample

The overlay discards same-named suppliers whose owner cannot be established,
and placed suppliers whose expression has no component type. At least one typed
supplier is still required. Proven omissions and forwarded arguments retain
existing treatment. This deliberately abandons consensus across all callers.

The current-baseline experiment removes all **104 feedGateway.getFeed /
arguments.feedBean** findings. Earlier source research identified bean/array
alternatives and missing suppliers; disappearance is not evidence those have
been reconciled. Most net reduction is in Mura; ContentBox is second.

The focused caller-uncertainty regressions pass under production. Under the
relaxation, TestAnUntypedArgumentHoldsWhatEveryCallerPasses fails: useOpen is typed
despite a call on an unknown receiver supplying a competing object. The unknown
argumentCollection regression selected in the same run remains passing; this is
not a claim that every opaque-collection case is safe under the relaxed policy.

Any future opt-in caller policy should expose skipped sites and expressions,
retained suppliers, conflicting alternatives and whether unknown ownership or
unknown value was discarded. An unexplained lower count would hide these losses.

## How current disappearing findings are accepted

An instrumented production-policy scan uses ResolveCallTarget for strict-report
sites and produces exactly the delivered both-mode findings per entry. It audits
all **4,777 removals against strict**; strict-to-both also has six added entries.

| Acceptance class | Removed entries |
| --- | ---: |
| Source component declaration | 3,011 |
| Builtin/native contract | 1,747 |
| Dynamic acceptance | 13 |
| Unmatched or mixed target observations | 6 |

The audit joins file, line, caller, receiver and method with diagnostic
multiplicity. Those keys can describe multiple calls on one line; six entries
are not assigned a definitive class. Source declarations include interface
contracts, not necessarily concrete runtime implementations. Builtin acceptance
uses the existing catalog, not engine execution. Both still depend on the
unproven guessed receiver origin.

This corrects the prioritization: the current gains are overwhelmingly method
compatibility with declarations/contracts, rather than wholesale onMissingMethod
suppression. The missing proof is principally the object's origin and receiving
role. Dynamic examples include MockBox decorations and Mura contentNavBean calls;
the audit records actual target kinds without attributing every dynamic outcome
to onMissingMethod specifically.

## Proposed next work and validation boundaries

Prioritize complete internal candidate sets, role-aware original/returned-object
constraints, and explicit caller contradictions before broadening ranking.
Retain separate controls for candidate selection and caller relaxation; both
remain research-only. If iterative promotion is investigated, preserve a strict
snapshot, round bounds and dependency provenance to prevent guessed evidence
from validating itself.

The combined experiment tests ranking with full candidates and caller relaxation
together; its reduction must be measured rather than obtained by adding rows.
No missing dependency, host base or unchecked summary is suppressed by these
experiments. Actual missing roots remain a separate coverage-changing task.

All successful conditions retain indexed files, checked-record totals and exact
missing-base entry multisets. Research binaries compile; the audit is checked for
per-entry equality with production. Production source hashes are unchanged.
The focused control/counterexample runs and git diff --check are retained. No
full production test suite, new runtime accuracy result or published CI result
is claimed for this research-only batch.


## Receiver-role research follow-up

[The receiver-role study](RESOLUTION-RECEIVER-ROLE-RESEARCH.md) completes 14
additional corpus scans. Separating original receivers from returned objects
leaves 7,593 experimental findings (1,060 removed / 78 added against delivered
both-mode 8,575), preserving coverage. A new numeric-supplier probe still exposes
wrong-origin guessing. Requiring two observed methods restores many existing
guesses and leaves 10,299. The next source-grounded opportunity is a 178-entry
Lucee Field/Group collection cohort; its improvement is not yet measured.
Production remains 13,346 strict / 8,575 with both flags; all new policies remain
research overlays.
