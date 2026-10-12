# Method connections versus absent methods — 2026-10-11

This is an evidence classification of the latest **12,355 strict finding entries**,
not a new resolver implementation or a count of unique methods. A single method
can appear in many call sites, and several chain diagnostics can come from one
broken upstream link. Inputs are the seven unchanged project reports in
`target/resolution/string-producer-followup/checked/strict/`.

The subsequent [method existence audit](RESOLUTION-METHOD-EXISTENCE-AUDIT.md) searches broader dependency sources and separates positive ownership evidence, likely API issues and names whose ownership remains unproven. The mechanism partition below is retained as the earlier checkpoint.

## Main classification

| Classification | Entries | What the evidence establishes |
|---|---:|---|
| Connection or execution context unresolved | **12,041** | The target cannot yet be established, or reviewed runtime registration was missed. This does **not** assert that every requested method exists. |
| Selected component/native contract mismatch, pending review | **310** | The selected contract does not answer the method. It may be a real absent method, wrong receiver, missing mixin, incomplete stub or missing engine API. |
| Confirmed deliberate nonexistent calls | **4** | Source-reviewed negative fixtures deliberately invoke undefined functions/methods. These should remain diagnosed. |
| **Total** | **12,355** | Every input entry is assigned once; none is omitted. |

Four is the count **confirmed in this review**, not an exhaustive claim that only
four methods are truly absent. The reports do not provide enough target/ownership
evidence to turn every unmatched spelling into an absence verdict.

## Where the connection is missing

| Missing connection | Entries |
|---|---:|
| Receiver variable → component/interface | 9,799 |
| Upstream method return → receiver of the next call | 1,200 |
| Bare function → lexical/include/inherited/runtime binding | 604 |
| Component/base/dependency → available source and execution context | 434 |
| Reviewed registration/injection → runtime method | 4 |
| **Total** | **12,041** |

Of the 9,799 receiver gaps, **7,683 have an indexed declaration with the reported
method spelling**, while 2,116 have none. This is evidence of available names,
not ownership: an unrelated class can implement the same method. No indexed
name also does not prove absence—engine APIs, runtime UDF keys, generated methods
and external dependencies may have no ordinary declaration in this index.

The 1,200 return gaps mean an upstream callable was found but its result does
not establish the next receiver. The final method spelling has indexed matches
in 1,066 cases and none in 134; the upstream and final methods must be treated
separately. The 604 remaining unbound functions have indexed same-name matches
in 416 cases and none in 188.

## What needs absence review

The 310 pending entries split into **108 selected component contracts** and
**202 native interfaces**. A native method missing from our hand-maintained table
is a contract coverage question before it is an engine absence question.
Selected components can likewise be stubbed, mismatched, or runtime-extended.

The [full grouped review queue](RESOLUTION-METHOD-CONTRACT-REVIEW.md) covers all
310 entries and links representative source locations. The per-entry JSON keeps
all occurrences and the original candidate evidence, including capped candidate
lists and the uncapped candidate counts supplied by the baseline.

## Source-reviewed examples

| Outcome | Call and source evidence |
|---|---|
| Missing connection, 3 calls | TestBox `CustomAssertions.cfc` calls `$assert.assertIsAwesome` and `$assert.assertIsFunky`. `beforeAll` registers the literal `CustomAsserts` component; BaseSpec.addAssertions copies its methods to `$assert`, and both declarations exist in CustomAsserts.cfc. One assertion is expected to fail, but its **method exists**. |
| Missing connection, 1 call | Wheels `memoProbeInjected()` is an anonymous UDF injected under a literal key and promoted from `variables` to `this`. The spec expects `injected`. Its indexed declaration count is zero: this directly demonstrates why “no indexed declaration” cannot mean “no function”. |
| Deliberately absent, 1 call | Lucee LDEV6055 Child invokes `super.nonExistentMethod()`. Its Parent has only publicMethod/publicMethodWithArgs; the surrounding test expects an error. |
| Deliberately absent, 1 call | Lucee LDEV6349 invokes `nonExistentMethod_ldev6349()` on java.util.Date and explicitly expects a missing-method exception. |
| Deliberately absent, 2 calls | Disabled LDEV1201 `xit` fixtures deliberately probe `notexisting()` and `notexisting().a` with Elvis fallbacks. These are source-reviewed intent, not executed engine results. |

Runtime registrations are classified from source relationships and fixture intent.
No CFML engine execution is claimed. The other entries remain mechanistically
classified, rather than manually proven at every target.

## Per-project partition

| Project | Connection/context gaps | Contract review | Confirmed absent fixture calls | Total |
|---|---:|---:|---:|---:|
| coldbox | 1,574 | 53 | 0 | 1,627 |
| contentbox | 1,707 | 36 | 0 | 1,743 |
| fw1 | 354 | 4 | 0 | 358 |
| lucee | 1,818 | 178 | 4 | 2,000 |
| masa | 3,669 | 13 | 0 | 3,682 |
| testbox | 386 | 3 | 0 | 389 |
| wheels | 2,533 | 23 | 0 | 2,556 |

## Evidence limits and reproducibility

The existing `definitions` annotation counts indexed declarations of the
**reported final function name**, not necessarily the first failed chain hop.
**1,221 entries name a different failed hop in their reason.** The export records
both names when recoverable. Treating that annotation as proof that the failing
method exists would misclassify these chains. Generated declarations and stubs
can also appear in indexed counts; source candidates are suggestions, not targets.

Missing bases remain 46 summary entries covering 1,133 unchecked calls. Those
calls are not separately classified here because the baseline compresses them.
The classification preserves their original summary entries. No coverage or
resolver outcome was changed.

Artifacts under `target/resolution/method-connection-classification/`:

- `classified-findings.json`: all 12,355 original entries plus analysis metadata.
- `connection_gap.json`, `contract_review.json`, `confirmed_absent.json`: exclusive partitions.
- `summary.json`: counts, projects, subcategories and annotation limitations.
- `input-sha256.json`: hashes of the exact baseline reports.
- `classify.py`: reproduces the partition and grouped Markdown review queue.

The script asserts the total, exclusive category counts, all eight reviewed
overrides, unchanged base-summary count and unchanged unchecked-call count.
The input files were re-hashed after generation to confirm that analysis did not
modify them. This task changes documentation only; no resolution fixes or full
Go test runs are claimed for it.

Next review should establish the actual receiver and its complete callable
contract before labeling a production call absent. Start with the 108 selected
component cases, then audit the 202 native cases against engine APIs. The much
larger connection queue should be investigated by shared receiver/return/binding
cause, with concrete declaration targets as the success criterion.
