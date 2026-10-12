# Ten passes on less strict resolution shortcuts

Implementation follow-up: staged primitive fallback is now delivered through
`--speculative-native`; see checkpoint 34 in
[implementation status](RESOLUTION-IMPLEMENTATION-STATUS.md). The other policies
remain research-only. The evidence below describes the preceding research batch.

11 October 2026. Research only: production resolver/parser/configuration files
are unchanged. Experiments use ignored Go overlays under
`target/resolution/shortcut-ten-pass/`. The checkpoint-33 strict baseline is
**13,346 findings**, 8,151 files and 189,154 checked records; 46 base summaries
represent 1,133 unchecked calls. These are diagnostic entries, not an exhaustive
count of runtime failures. Fewer entries do not establish connection accuracy.

## Pass 1 — Rebase and partition the opportunity

The current strict reports contain 10,483 unknown-receiver entries in 4,063
file/caller/receiver groups, 1,227 unknown-return entries, 1,176 method entries and
460 object entries. The existing experimental CFC/native combination has 9,686
entries, including 6,817 unknown receivers. Historical 6,393/36.93% experiments
used different reporting coverage and cannot be presented as current results.

This pass uses all seven fresh strict reports from checkpoint 33. The existing
flag totals reuse six checkpoint-32 experimental reports and checkpoint-33 Mura,
as documented there. Subsequent overlay conditions scan all seven roots afresh.

## Pass 2 — Common candidates across ambiguous method observations

The current policy requires exactly one candidate per finding. A weaker policy
intersects candidate files across every unknown finding for the same file,
calling function and case-folded receiver spelling. Require a unique survivor
and complete candidate lists; otherwise withhold the guess. Ordinary method and
return checks still run after attaching the receiver.

Eligibility expands from 1,017 groups/2,931 findings to 1,033/3,013: only **16 new
groups / 82 represented findings**. Candidate annotation already combines many
method observations, so intersecting again often adds no information. Examples
include ContentBox BaseContent.clone's arguments.original and ContentService.delete's
arguments.content. This is structural compatibility, not object origin.

## Pass 3 — A unique ranking winner

Intersect complete candidate sets, then choose the sole highest scoring survivor
when several remain. The overlay sums existing confidence labels (high=3,
medium=2, low=1) over findings; equal winners are rejected. It neither picks the
lexically first path nor claims those labels are probabilities.

This admits 1,229 groups/3,372 findings: **212 new groups / 441 represented
findings** over sole candidates. Mura configurator content/feed variables are
useful review candidates. Name affinity drives existing preference; repeated
findings weight the same evidence repeatedly. A future policy should report
that dependence and consider scoring distinct observations instead. Source
nearness/name likeness cannot defeat visible contradictory origins.

## Pass 4 — Candidate truncation and artificial uniqueness

Reports list at most eight candidates while candidateCount retains the total.
**806 unknown-receiver entries have truncated lists.** Intersecting just those
lists and treating a sole survivor as exhaustive admits 1,041 groups/3,088
findings: eight more groups/75 findings than complete intersections. This is a
separate deliberately less reliable condition, not evidence to remove the cap
from the production guess policy blindly. Omitted alternatives can create false
uniqueness. A proper experiment should expose complete internal candidate sets,
then re-evaluate ambiguity and performance separately from presentation limits.

## Pass 5 — Primitive/native families omitted from the guess catalog

Source review finds that native Guess enumerates Java collection/JDBC/ORM/time
families, but omits supported String, Array, Struct and StringBuilder. With both
current flags enabled, 508 unknown trim findings remain; sampled Lucee examples
read result.fileContent after HTTP activity. Other remaining names include append
(162), keyExists (107), listToArray (49), toString (109) and getClass (87).
These counts overlap receiver groups and are not an additive gains estimate.

The experiment adds those four families. One condition searches all old and new
families together, requiring a unique answer. Another preserves an existing
native-family answer first and searches primitive families only if it has none.
Adding Struct can make a formerly unique Map match ambiguous: native catalog
expansion is not necessarily monotonic. The staged condition deliberately gives
old families priority, an assumption that needs an explicit user-visible policy.
A combined condition also applies ranked CFC guessing before expanded natives.

An isolated two-file CLI probe supplies 42 to Service.use(value), whose body
calls arguments.value.ensureCapacity(20). Strict reports the unknown receiver;
the primitive overlay guesses StringBuilder and removes the finding. The
verbose log preserves the unproven connection. This confirms a concrete origin
constraint loss; it does not execute CFML or estimate corpus error frequency.
An initial trim probe under the repository inherited its configuration and was
already accepted by the control, so it is not counted as evidence for this policy.

No blanket Object promotion was added. Common Object methods alone cannot
identify the receiver family, and primitive versus object behavior, argument
signatures, CFML coercion and engine bridges remain outside these usage guesses.

## Pass 6 — Majority method agreement

Offline eligibility allows a unique candidate supported by at least 80% of a
group's complete candidate lists, dropping dissenting observations. It admits
1,052 groups/3,187 findings: 35 new groups/256 findings over sole candidates.
This is an eligibility profile only; no measured scan reduction is attributed
to it. Most new groups involve chains (31 of 35), suggesting receiver roles
rather than a tolerable random error rate. A minority method can identify a real
alternate value or mutation. Do not discard it as noise by default.

## Pass 7 — Discard unknown callers or supplied values

The current arg_callers.go still vetoes an inferred component when ownership of
a supplying call is unknown or a placed caller supplies an untyped value. The
existing exceptions for proven omission and forwarded arguments are distinct.
Historical overlays demonstrated removals and regression counterexamples when
these vetoes were ignored. Those historical reductions are not freshly measured
here; the earlier changeset blocker is now fixed, so the old gain is stale.

The 104-entry Mura feedGateway.getFeed/arguments.feedBean group still remains
with both flags. It is a review target for an opt-in known-suppliers-only policy,
not a forgotten limit. Any such policy should retain discarded caller sites,
expressions and known alternatives as evidence and keep explicit numeric/array
contradictions visible. The current candidate wrong-origin regression still
accepts ready() despite a caller passing 42 under an enabled usage guess.

## Pass 8 — Framework context and conventional record slots

Current strict reports contain 588 findings using the receiver spellings of the
older framework guesses: arguments.item/entity, entity, field, results.ehBean
and Wheels engineAdapter slots. This is a syntax population, not a measured
reduction or a claim those rules apply at every site. Large remaining groups
include Mura collection item, Lucee admin Field methods and ColdBox ehBean.

Source review of ColdBox Controller shows results.ehBean initialized empty,
then written by handlerService and replaced on exception paths. A shortcut can
assume EventHandlerBean after the established writes while recording those
initial/alternate states. Lucee Field requires driver/template family scope;
Mura entity cannot always be contentBean. Prefer project-scoped receiver rules
with source markers, explicit mappings and configuration overrides over
repository-name strings or a project-wide framework preset.

## Pass 9 — Separate original receivers from return-hop roles

A parser-only probe matches every eligible unknown finding back to its call
chain. Of the 16 additional complete-intersection groups, **12 contain chains**.
Existing sole candidates already have 15 such groups; complete intersection has
27, ranking 28 and capped intersection 35. The probe is an origin-role audit,
not proof that every chained candidate is incorrect.

Candidate annotation groups observations by chain depth and final hop, but the
guess map is keyed only by file/caller/original receiver spelling. A candidate
for a final getCurrentEvent/renderdata result can be RequestContext while the
original receiver is services.requestService. Mura rbFactory chains nominate
resourceBundle even though resourceBundleFactory and its result have different
roles. These examples must be checked at the precise hop before being promoted.

There are 1,227 return-type findings, 1,027 with suggestions. Promoting their
final-method candidates as original receivers would repeat this error. A useful
future shortcut would attach a labelled return guess to producer+hop+call shape,
respect known returns and check downstream methods. Existing onMissingMethod
acceptance can hide an entire chain dynamically; a declaration-backed success
and a dynamic acceptance must be reported separately when auditing removals.

## Pass 10 — Review interactions, controls and next steps

The strongest next experiment is primitive-family fallback with role-aware CFC
ranking. Keep switches independent and default off: candidate policy
(sole/intersection/ranked), native policy (current/primitive fallback), and a
separate caller-consensus policy if pursued. These names are design proposals,
not delivered CLI flags. Show component/family, origin-unproven status, evidence,
truncation, rejected observations and whether acceptance was declared or dynamic.

Do not iterate guesses silently to a fixed point. A guessed origin can make its
own downstream suggestions appear supported. An iterative experiment needs a
round limit and provenance linking each derived guess to its original strict
snapshot. Primitive contradiction checks, reassignment/value identity, chain
roles and dynamic-dispatch classification should precede a wider production
opt-in policy.

Missing-base shortcuts are a different problem: 46 summaries represent 1,133
unchecked calls. Substituting a nearby basename or test double can add checking
under the wrong host contract. Such dependency scans need actual roots and a
separate coverage audit. Do not hide those summaries as speculative successes.

## Measured overlay comparisons

Completed per-entry comparisons follow. All conditions
use the same scan roots. Eligibility counts above are not credited as measured
reductions. Production strict behavior remains unchanged.

| Research condition | Remaining | Removed / added vs strict | Reduction vs strict | Additional net vs existing comparable flags |
| --- | ---: | ---: | ---: | ---: |
| Complete unique intersection (CFC only) | 10,248 | 3,110 / 12 | 23.21% | 74 |
| Unique rank winner (CFC only) | 9,893 | 3,465 / 12 | 25.87% | 429 |
| Truncated intersection (CFC only) | 10,177 | 3,186 / 17 | 23.74% | 145 |
| Expanded native families + sole CFC | 9,068 | 4,282 / 4 | 32.05% | 618 |
| Expanded natives + ranked CFC | 8,641 | 4,717 / 12 | 35.25% | 1,045 |
| Existing native priority + primitive fallback + sole CFC | 8,575 | 4,777 / 6 | 35.75% | 1,111 |

The first three conditions compare additionally against current CFC-only
10,322; the last three compare against current both-flags 9,686. Strict stays
13,346. The strongest measured condition is staged primitive fallback: **8,575**,
a net **4,771 (35.75%)** below strict and **1,111 (11.47%)** below existing both
flags. Against both, it removes 1,111 and adds zero. Against strict, it removes
4,777 and adds six. These are different baselines; six additions are not new
relative to the current experimental combination.

The all-families search removes 938 but adds 320 compared with current both
flags. Ranked CFC plus that search removes 1,373/adds 328. This confirms that
native-family ambiguity can reintroduce formerly guessed diagnostics despite
improving the total. Prefer separately visible family priority over assuming
catalog expansion only adds successes.

| Corpus | Strict | Existing both flags | Staged primitive fallback |
| --- | ---: | ---: | ---: |
| wheels | 2,682 | 1,832 | 1,737 |
| masa | 3,689 | 2,529 | 2,429 |
| contentbox | 1,837 | 1,187 | 1,120 |
| coldbox | 1,653 | 1,041 | 906 |
| lucee | 2,730 | 2,490 | 1,854 |
| testbox | 396 | 318 | 261 |
| fw1 | 359 | 289 | 268 |

All **42 scans** completed: six conditions over seven corpora. Every condition
retains 8,151 indexed files, 189,154 checked records and 1,133 unchecked calls.
The exact missing-base entries are checked separately against each control.
No corpus source/configuration was edited. Individual condition gains overlap;
ranked CFC plus staged primitive priority is not a measured combination here.
Per-entry comparisons against both strict and existing flags are retained in
review.json and condition artifacts.

Recommended next work is a configurable primitive fallback policy with explicit
family priority and chain-role auditing, followed by complete-set CFC ranking.
Caller-majority relaxation, truncated uniqueness and final-hop promotion remain
lower-trust review experiments. The observed wrong-origin probes and dynamic
acceptance risk prevent treating any of these numbers as verified fixes.

Artifacts retain setup scripts, overlay sources, binaries, profiles, eligibility
maps, parser chain audit, per-entry findings/removals/additions, source hash
verification and current speculative-policy regression output. The focused
intersection-policy tests pass, including the test that demonstrates accepting
a wrong numeric origin; this is policy behavior validation, not accuracy proof.
No new runtime execution or CI result is claimed. No full build/test/lint rerun is
needed for documentation and ignored research overlays; each research binary is
built and the final documentation is checked for whitespace errors.
