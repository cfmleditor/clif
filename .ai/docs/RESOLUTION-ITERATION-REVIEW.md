# Resolution iteration review: checkpoints 24–25

Twenty focused passes covered investigation, implementation, counterexamples and
validation. These are not twenty independent resolution features. Two measured
resolution batches and one discovery-cache correction were implemented.

The same seven corpora now report **10,740** unresolved findings, down from
12,570: **1,830 fewer net (14.6%)**, or 74.2% below the original 41,707 baseline.
Coverage stays at 8,151 files. Preset assumptions and override options are detailed
in [PROJECT-ASSUMPTIONS.md](PROJECT-ASSUMPTIONS.md).

| Pass | Question or action | Result |
| --- | --- | --- |
| 1 | Rebucket the checkpoint 23 findings | 9,735 receiver-type findings dominate the remaining 12,570 |
| 2 | Inspect ContentBox's 884 table findings | Sampled calls are schema.create/alter migration callbacks |
| 3 | Trace migration entry-point relationships | Shipped components declare up/down(schema,qb/query); existing preset has matching APIs |
| 4 | Define automatic-selection boundaries | One recognized owner, expected migration directories, compatible argument names/types |
| 5 | Test counterexamples before implementing | Ordinary folders, primitive arguments, outside paths and mixed owners acquire no migration context |
| 6 | Measure migration selection per diagnostic | 1,140 removed, four deeper grammar returns added: 1,136 net |
| 7 | Inspect task and command host relationships | Build components use run/getCWD/shell; CLI components extend CommandBox classes |
| 8 | Challenge host-name detection | A parameter named run initially matched; require an actual run function |
| 9 | Challenge lexical host evidence | Comments and locally declared getCWD do not establish CommandBox context |
| 10 | Review naming breadth and override behavior | Presets remain project-wide assumptions; explicit configuration can omit or replace them |
| 11 | Check dynamic printing semantics | Pinned Print/PrintBuffer define onMissingMethod; unknown printing methods intentionally remain dynamic |
| 12 | Compare CLI and editor selection paths | Both use the same detector; schema methods check against stubs and explicit config opts out |
| 13 | Test context freshness after source additions | ReadDir was untracked; recording roots/failures fixes new-file invalidation, with a failing mutation regression |
| 14 | Review CommandBox module invocation paths | Pinned loader derives invocation paths from installed module names; cfMapping may differ, so no guessed alias was added |
| 15 | Review newly exposed qb grammar returns | Existing API documents a default grammar, but the property is configurable; a general return assumption is withheld |
| 16 | Review Mura feed argument ownership | DAO callers exist, but common method names can exceed the existing 60-file/40-site inference bounds before owner filtering |
| 17 | Review Lucee field collections | Driver.getCustomFields feeds a loop with Group discrimination; a uniform field type needs driver/collection contracts |
| 18 | Audit the complete two-batch diagnostic diff | 1,842 removed, 12 added; raw call identities lose 1,831 and gain one; additions inspected with source excerpts |
| 19 | Run required checks and counterexample regressions | Full make check passes; separate short suite and whitespace check complete the validation |
| 20 | Repeat the seven-project measurement and review documentation | Final diagnostics match exactly; assumptions, limitations and next opportunities are recorded |

Per-entry audits and source excerpts are in target/resolution/assumed-migrations
and target/resolution/assumed-commandbox. The final scan is
target/resolution/tooling-defaults-final. Candidate annotations are excluded from
diagnostic identity. Filtered checked calls increase from 179,731 to 179,807;
these are not raw extraction counts. No new API stubs or upstream versions were
introduced. No CFML runtime execution or published CI result is claimed.

The twelve additions comprise four grammar-return gaps, four unresolved Wheels
command bases, three service API mismatches and one untyped test argument. They
are not confirmed application defects. The base diagnostics summarize inherited
calls that remain unchecked; they are not individual method-resolution wins.

The pinned [CommandBox module loader](https://raw.githubusercontent.com/Ortus-Solutions/commandbox/28a136f9b477abfa50259bb0cab2af0097cc68e8/src/cfml/system/services/ModuleService.cfc)
sets an invocation path from the module's registered name and separately registers
its CF mapping. External locations, custom registration names and runtime include
settings can change that relationship. A source checkout's folder name or declared
cfMapping is therefore insufficient to prove commandbox.modules.<name> aliases.

The next broad work should establish a bounded caller index keyed by resolved
function ownership before applying argument-inference limits. It must retain
unknown/conflicting caller guards and avoid inferring from only a convenient
subset of calls. Collection work should trace driver/registry/iterator element
contracts and branch narrowing rather than assign global field/item types.


Final validation passes: make check, separate full go test -short ./... and
git diff --check. The repeat has zero removed/added diagnostics for every corpus.

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

## Checkpoint 29 review: 136 more resolutions, no additions

The caller-argument omission fix is implemented and audited: seven-corpus totals
fall **10,371 → 10,235**, with 49 removals in Wheels and 87 in Masa, zero additions,
and unchanged 8,151-file / 179,808-filtered-check coverage. Unknown ordinary calls
can be ignored only when exact balanced syntax proves the target argument absent;
unknown supplied arguments and opaque argument collections still block inference.
An intermediate TestBox reason change uncovered runtime component interpolation
being treated as a literal; its regression-backed guard restores the previous
seven diagnostics. Literal ORM load/array `each` element support is also implemented
with rejection coverage, but has zero measured corpus gains.

Full `make check`, the separate short suite and whitespace checks pass. Details,
assumptions, rejected forms and next proof gaps are recorded in checkpoint 29 of
`RESOLUTION-IMPLEMENTATION-STATUS.md` and `PROJECT-ASSUMPTIONS.md`. Final audit
artifacts are in `target/resolution/omitted-callers-final/`. Future work should
prioritize source-backed mutable iterator/instance contracts and return paths;
this checkpoint does not complete those remaining plans.
