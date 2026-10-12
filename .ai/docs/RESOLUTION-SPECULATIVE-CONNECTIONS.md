# Speculative connection experiments

10 October 2026. The user explicitly requested unreliable connections to measure
their effect. The initial experiments used ignored Go overlays under
`target/resolution/speculative-connections/`. Their strict baseline is
**10,136 diagnostic entries**. These experiments measure diagnostic reductions,
not accuracy or verified fixes. The subsequent opt-in CLI policy is described
below; the other overlay policies remain research experiments.

## Result

The strongest combination produces **6,393 entries**, a net reduction of
**3,743 (36.93%)**. Structural candidate promotion has substantially more reach
than relaxing caller consensus alone.

| Experimental policy | Entries remaining | Removed / added | Net reduction | Reduction |
| --- | ---: | ---: | ---: | ---: |
| Off-switch control | 10,136 | 0 / 0 | 0 | 0% |
| Ignore uncertain caller ownership | 9,937 | 199 / 0 | 199 | 1.96% |
| Ignore untyped values supplied by known callers | 9,887 | 251 / 2 | 249 | 2.46% |
| Both caller relaxations | 9,672 | 466 / 2 | 464 | 4.58% |
| Selected receiver guesses | 9,675 | 462 / 1 | 461 | 4.55% |
| Caller relaxations + guesses + scoped admin context | 8,836 | 1,305 / 5 | 1,300 | 12.83% |
| Promote consistent high-ranked candidates | 7,939 | 2,198 / 1 | 2,197 | 21.68% |
| Promote a sole consistent candidate | 7,392 | 2,748 / 4 | 2,744 | 27.07% |
| Sole-candidate policy + the combined policies above | **6,393** | **3,752 / 9** | **3,743** | **36.93%** |

These are separate measured runs. Their reductions overlap and interact; adding
the individual counts is incorrect. The strongest combined row includes the
earlier scoped admin experiment, so its whole reduction is not attributable to
newly unreliable caller/candidate rules.

| Corpus | Control | Sole consistent candidate | Strongest combination | Combined net reduction |
| --- | ---: | ---: | ---: | ---: |
| Wheels | 2,184 | 1,638 | 1,613 | 571 |
| Masa/Mura | 3,723 | 2,609 | 1,820 | 1,903 |
| ContentBox | 1,378 | 851 | 791 | 587 |
| ColdBox | 1,034 | 680 | 678 | 356 |
| Lucee | 1,318 | 1,233 | 1,114 | 204 |
| TestBox | 246 | 187 | 186 | 60 |
| FW/1 | 253 | 194 | 191 | 62 |
| Total | 10,136 | 7,392 | 6,393 | 3,743 |

## Exactly which unreliable connections were tested

**Caller ownership:** when argument inference encounters a same-named call whose
receiver cannot be placed, continue searching instead of vetoing the inferred
argument. This can discard a real supplying caller, including opaque
argumentCollection calls. It is broader than fixing source identity so that a
genuine omission can be proved.

**Supplied values:** when a known supplying caller passes an expression whose
component cannot be established, discard that site and use the remaining known
sites. Require at least one retained supplying site. Unknown forwarded arguments
already have their existing treatment; the experiment expands skipping to other
unknown values. It does not prove consensus across all callers.

**Selected guesses:** add fallback receiver types for these contexts:

- Mura collection templates' arguments.item → contentNavBean.
- Mura apiUtility entity/arguments.entity → contentBean, disregarding entity
  selectors and heterogeneous producers.
- ColdBox Controller results.ehBean → EventHandlerBean.
- Lucee admin field → the Field component selected from the template family,
  without proving driver/element alternatives and negative-guard narrowing.
- Wheels application engineAdapter slots → the four engine adapter alternatives,
  without proving publication, retained state or all writers.

These guesses apply when ordinary receiver lookup fails; established receivers
and positive guards retain priority. Normal method checking then continues.
The special component paths and project markers are experiment fixtures, not a
general detector or proposed production configuration.

**Candidate promotion:** take the checkpoint-30 suggestion reports and group
unknown receiver findings by exact file, caller function and receiver spelling.
For each group, require a common candidate component across every finding in
that group. The high-ranked variant intersects candidates labelled high; if
several remain, choose the lexically first path. The sole-candidate variant
requires every finding's candidateCount to be one and all to name the same file.
It ignores groups without such agreement. Both inject the selected component
as a fallback in receiver lookup and retain ordinary method/chain checking.

The high-ranked input map has 502 groups representing 2,122 baseline entries;
the sole-candidate map has 874 groups representing 2,657. Actual removals differ
because guessed receivers can unlock downstream inference. These maps are
derived from this baseline, so this is an in-sample experiment. It does not
establish generalization to another project or unseen revision. “High” is a
ranking label, not a measured probability of correctness. A sole indexed
candidate can still be the wrong object.

## The uncertainty has observable consequences

The five existing regression families selected for caller uncertainty pass with
the overlay's off switch. With owner skipping enabled, three fail: unknown-owner
supplied arguments and unknown argumentCollection calls are ignored. With value
skipping enabled, two fail: a placed caller supplying an untyped value and a late
unknown value are ignored. These are deliberate counterexamples to the proposed
inference, not unexplained failures of production behavior.

A separate observation probe makes the candidate failure concrete:

```cfml
// Service.cfc
component {
    function use(bean) { arguments.bean.ready(); }
}
// A different component A declares ready().
new Service().use(42);
```

The control with caller inference enabled retains an unknown-receiver diagnostic.
Promoting A as the receiver candidate accepts ready despite the visible numeric
argument. The probe checks that the call was extracted and exercises the actual
candidate fallback through an overlay test file. It demonstrates loss of an
origin constraint; it does not execute CFML or estimate how often this occurs
in the corpus.

Mura's 104 feed argument entries and 65 changeset argument entries disappear in
the combined caller relaxation. Earlier research established a bean/array return
alternative in a feed caller and multiple missing origins in competing save
calls. Their disappearance confirms the impact of ignoring those blockers; it
does not resolve their underlying ambiguity or establish legitimate omission.

The strongest combination introduces nine diagnostics: three Wheels getTestBox
return-chain failures; three Mura return-chain failures; one ContentBox
getActiveContent return failure; ColdBox EVENT_CACHE_SUFFIX missing on the guessed
EventHandler; and TestBox size missing on guessed MockBox. Exact additions are
saved per corpus. Some are replacement reasons at existing sites, not new call
sites. Conversely, few added diagnostics do not imply few incorrect acceptances:
an incorrect connection can make a diagnostic vanish.

Receiver guesses and candidate promotion can reach components with
onMissingMethod. Existing resolution then accepts dynamic dispatch without
proving the requested method succeeds. The [execution-context audit](RESOLUTION-CONTEXT-AUDIT.md)
already demonstrates this distinction. We have not classified every speculative
removal into declaration-backed, native/member or dynamic acceptance.

## Coverage and reproducibility

All 63 scans completed: nine conditions over the same seven corpora. The
off-switch control matches checkpoint 30 per diagnostic, including multiplicity.
All conditions retain the same indexed/scanned file counts, totaling 8,151, and
the same 46 missing-base summaries representing 1,132 unchecked calls.

| Condition family | Change in resolved-plus-diagnostic record count |
| --- | ---: |
| Caller-only relaxations | 0 |
| Selected guesses | +68 |
| Combined callers/guesses/admin context | +69 |
| High-ranked candidate promotion | +125 |
| Sole-candidate promotion | +143 |
| Strongest combination | +205 |

The control has 179,806 records; the strongest combination has 180,011. No
condition decreases that aggregate. These counts and unchanged file/base coverage
are useful checks, but do not constitute a full identity audit of every resolved
call: simultaneous extraction changes could be hidden by aggregate totals.

Each condition has JSON findings, stderr coverage counts, per-entry additions and
removals, and a summary in the local artifact directory. `aggregate.json` contains
the tables above; `candidate-choices.json` records every injected candidate;
`production-source-hashes.json` verifies original files were not edited. Overlay
setup and run scripts retain the exact mechanisms. These experiment mode flags
exist only in those overlays. No corpus source or
configuration was rewritten.

## Opt-in CLI policy

`clif unresolved --speculative <directories>` now implements the sole consistent
candidate policy. It is off by default; `--no-speculative` explicitly disables
it, and the last of these flags wins. LSP, MCP and ordinary scans retain their
existing behavior. There is no environment-variable switch or persistent config
setting for this first policy.

```sh
# Strict control and proposed connections, captured separately for review.
clif unresolved --json --candidates project > strict.json
clif unresolved --speculative --json --candidates --verbose project > speculative.json 2> speculative-review.log
```

Verbose stderr marks each proposed fallback with `? speculative`, source path,
line, calling function, receiver spelling, and chosen component. The final summary
reports the number of proposed connections, not proven successes. JSON stdout
keeps the existing unresolved-array format. Compare the two JSON reports per entry
to review removals and additions. `explain` still uses strict resolution.

The implementation builds candidates from a strict scan of the current input;
there are no embedded corpus paths or saved candidate choices. For each unknown
receiver group (file, caller function, case-folded receiver spelling), every
unknown-receiver finding must offer exactly one candidate and agree on its file.
A second scan uses those immutable guesses only after ordinary receiver lookup
fails. Method and chain checks then continue, including existing dynamic dispatch
and member-method acceptance. The snapshot is used once; guesses are not repeatedly
promoted to a fixed point. Both scans rebuild their indexes to isolate caches,
so enabled mode adds scan/indexing cost. Input should remain stable during a run.

Assumptions: the sole indexed candidate is the runtime object; a receiver spelling
within one function represents compatible values; the indexed workspace is
representative. These assumptions can fail for numeric or scalar callers,
external/unindexed implementations, reassignment, heterogeneous collections,
test doubles, and dynamic dispatch. A regression explicitly demonstrates accepting
ready() even when the known caller supplies 42. Other tests retain known receiver
types, ambiguous candidates, incomplete method groups, and strict off behavior.

This first CLI policy excludes the earlier unknown-caller/value relaxations,
hard-coded framework guesses, high-rank tie breaking, and scoped admin assumption.
The historical 36.93% experiment is consequently not the expected CLI reduction.

### Current CLI validation

Fourteen fresh scans (strict and enabled over seven corpora) reproduce the
sole-candidate overlay result exactly: **10,136 → 7,392**, with **2,748 removed,
4 added, net 2,744 (27.07%)**. All strict per-entry multisets match checkpoint 30.
Enabled mode proposes 874 receiver connections from the current index.

| Corpus | Proposed connections | Strict entries | Enabled entries |
| --- | ---: | ---: | ---: |
| Wheels | 210 | 2,184 | 1,638 |
| Masa/Mura | 219 | 3,723 | 2,609 |
| ContentBox | 199 | 1,378 | 851 |
| ColdBox | 142 | 1,034 | 680 |
| Lucee | 43 | 1,318 | 1,233 |
| TestBox | 36 | 246 | 187 |
| FW/1 | 25 | 253 | 194 |

File coverage stays at 8,151. Missing-base entries are identical per entry,
retaining 1,132 unchecked calls. Resolved-plus-diagnostic records increase from
179,806 to 179,949 (+143); this aggregate does not prove the identity of every
accepted call. The four additions are three Wheels BaseSpec chain failures
(getTestBox has no component return type for announceToModules) and one ColdBox
EVENT_CACHE_SUFFIX missing on the guessed EventHandler. They expose downstream
gaps and do not establish that the guesses are correct.

Artifacts under `target/resolution/speculative-cli/` retain all findings, per-entry
removals/additions, verbose logs, and aggregate coverage. `connections.json`
records all 874 proposed connections with project, source, caller, receiver,
chosen component and `originProven: false`; these were extracted from verbose
output. They are review candidates, not a list of successful verified resolutions.

The opt-in behavior test failed with the option fields present but the policy
absent, then passed with the implementation. Build, vet, full short tests,
focused race tests and lint pass. Full short tests initially failed only because
the sandbox denied Unix socket binds; the approved outside-sandbox rerun passed.
No CFML execution, full runtime accuracy evaluation, or new CI result is claimed.

Required preflight passes with approved network access following sandbox DNS
failure. All research binaries build. The control regressions pass; speculative
regressions fail as described, and the candidate observation probe passes by
confirming its unreliable acceptance. No full-suite or runtime accuracy pass
is claimed.

## What this tells us to investigate next

There is enough potential connectivity to exceed 30% under these speculative
policies. The dominant source is candidate-to-receiver identity, followed by the
Mura execution context and caller consensus gaps. The 3,743 disappearing entries
are a concrete review population, not a list of 3,743 verified fixes.

Next, inspect grouped candidate origins: where was the object constructed,
injected, returned, selected or installed? Preserve rejected evidence such as
primitive callers, conflicting producers, alternate templates and state writers.
Where the connection cannot be proved, a separately labelled speculative
suggestion can retain that uncertainty. Promoting these rules into the default
resolver would currently discard constraints that the counterexamples show are
necessary.

## Native-family policy follow-up

The current batch adds a separate `--speculative-native` policy, off by default,
with `--no-speculative-native` for an explicit off switch. It requires exactly
one catalogued interface family to answer every unknown finding's final method
name in the file/caller/receiver group. Generic Object names alone cannot select
one family; unsupported members and ambiguous families withhold a guess.
When both flags are set, a consistent sole CFC candidate takes precedence;
otherwise the native policy is tried. Subsequent method/return checks still run.

Verbose output distinguishes “sole candidate” from “sole native family” and
marks either origin unproven. `SpeculativeConnection.Policy` also distinguishes
them internally. No source-origin, engine bridge/version, argument or actual
runtime-value proof is implied. For example, observed Map methods can nominate
Map even when a supplying caller passes a scalar. The method/result catalog
cannot repair that origin error. Existing caller-veto relaxation and hardcoded
framework-origin overlays remain research only.

See [native assumptions](RESOLUTION-NATIVE-TYPE-RESEARCH.md#implementation-follow-up-and-compatibility-assumptions),
[the completion ledger](RESOLUTION-COMPLETION-AUDIT.md), and checkpoint 32 in
`RESOLUTION-IMPLEMENTATION-STATUS.md` for final comparisons. Reports, per-entry
diffs and connection inventories are under `target/resolution/native-contracts/`.


## Checkpoint-33 follow-up: ten shortcut research passes

See [the current ten-pass shortcut review](RESOLUTION-SHORTCUT-TEN-PASS.md).
Production policies remain unchanged. Forty-two overlay scans test complete
candidate intersections, unique rank winners, truncated intersections and
primitive-family extensions. Strongest measured policy preserves existing
native guesses before primitive fallback: 13,346 strict → 8,575 experimental;
1,111 fewer than current both-flags 9,686. File/record/base coverage is preserved.
This is unreliable usage inference, including a demonstrated StringBuilder
guess for a numeric supplied argument. Candidate chain-role mismatches and
nonmonotonic native ambiguity are documented; no production default or new
CLI flag is delivered in this research batch.


## Delivered primitive-family fallback (checkpoint 34)

The staged primitive policy from the ten-pass review is now delivered through
`--speculative-native`; `--no-speculative-native` restores strict behavior.
The feature remains off by default. Existing native families retain their unique
answer first. If that stage has no answer, a unique String/Array/Struct/StringBuilder
family matching every observed method can be guessed. When both flags are set,
a consistent sole CFC candidate still takes precedence.

A unique answer is required within the selected stage, not across the combined
catalog. Thus existing List usage may win even where String also answers it.
This explicit compatibility policy preserves earlier guesses; it does not prove
which family was supplied. An ambiguous existing stage can still fall back to a
unique primitive stage. Unsupported/mixed primitive methods and generic Object
names withhold the fallback. Known receiver origins retain ordinary resolution
priority. Receiver groups still share file/caller/spelling and one strict
snapshot; reassignment/value identities and original-versus-return-hop roles
remain uncertain.

Verbose stderr labels primitive guesses as “primitive family fallback” and
records them with Policy `sole-primitive-family`. The public unresolved JSON
array remains diagnostic findings; proposed connections are exposed through
verbose output as before. The wrong-origin fixture is retained: supplying 42
while calling ensureCapacity yields a StringBuilder guess when enabled.
That passing regression confirms unreliable behavior, not runtime correctness.
Family matching does not validate call arguments, engine coercion, installed
bridges or generic collection element types. Array/Struct/String catalog coverage
is bounded; unsupported methods can prevent otherwise plausible groups.

Candidate intersection/ranking, truncated uniqueness, caller-veto relaxation and
iterative promotion remain research-only. They are not implicitly enabled by
this native-family extension. Corpus comparison and validation are recorded in
checkpoint 34 of [implementation status](RESOLUTION-IMPLEMENTATION-STATUS.md).


## Research after checkpoint 34

[The follow-up review](RESOLUTION-SHORTCUT-FOLLOWUP.md) measures ranking with
staged primitive fallback, complete candidate sets, chain-group withholding,
caller relaxation and acceptance classes. Forty-two successful overlay scans
produce a strongest unreliable total of 7,733 (842 fewer than delivered both-mode
8,575), retaining file/record/base coverage. Ranking with full candidates alone
leaves 8,017. A chain probe and a caller uncertainty regression demonstrate
specific constraint losses; acceptance auditing does not establish object origin.
Production stays at 13,346 strict / 8,575 with both flags. None of the new overlay
policies is delivered by this research batch.


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


## Twenty-idea research follow-up

[The twenty-idea review](RESOLUTION-TWENTY-IDEA-RESEARCH.md) completes 49 corpus
scans and 112 paired structural probe scans. Proximity leaves 6,824 experimental
findings, but selects Probe for explicitly constructed RateLimiter receivers
in 36 removed corpus entries. Role + caller relaxation leaves 7,315. The study
identifies scalar factory return precedence, literal collection elements, union
guards, record fields and closure lifetime as connected source-based targets;
blanket if specialization is rejected after dynamic acceptance and corpus losses.
Production remains 13,346 strict / 8,575 with both flags; no new policy is delivered.
