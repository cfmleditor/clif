# Method evidence for strict and speculative resolution — 2026-10-11

The [existence audit](RESOLUTION-METHOD-EXISTENCE-AUDIT.md) separates available method names from proven receiver relationships. Batch scans now expose that distinction directly with `--json --candidates`. This is an additive reporting improvement and a failed-hop candidate lookup correction; it does not add a new acceptance heuristic.

## Two independent distinctions

| Name evidence | What it supports | What remains unproven |
|---|---|---|
| `no-source-declaration` | Investigate engine APIs, dependencies, runtime registrations or a possible source defect. | Absence at runtime: dynamically bound functions need no ordinary declaration. |
| `unique-source-declaration` | A single indexed source declaration is a candidate for relationship tracing or speculative inference. | The intended receiver can reach that declaration. |
| `ambiguous-source-declarations` | Compare the possible owners against constructor, DI, inheritance and observed method evidence. | Which owner is intended; multiple names do not prevent strict resolution when ownership is established. |

`methodEvidence.relationship` separately identifies `receiver-type-unproven`, `return-type-unproven`, `binding-unproven`, `inheritance-binding-unproven`, `source-or-context-missing`, `selected-contract-mismatch`, or `native-contract-mismatch`. These are evidence gaps, not automatic source-bug classifications. The reviewed actual-issue candidates and deliberate negative fixtures remain documented in the existence audit.

`methodEvidence.ownership` is `unproven` on unresolved findings. The source declaration count includes generated accessors and excludes framework stubs. The indexed declaration count also includes stubs. Both describe the index populated for this scan, not every project on disk or every runtime dependency. Broken-base summary entries are not individual method searches and omit the new evidence.

For example, `known.missing().finish()` can report `function: "finish"` while failing at `missing`. The new evidence counts and searches declarations of `missing`; method candidates likewise point to that failed hop. Existing `function` and `definitions` fields retain their earlier final-name meaning for compatibility. This prevents a unique final-method spelling from being mistaken for evidence about an earlier missing method.

## Using the distinctions

| Relationship evidence | Appropriate action |
|---|---|
| Receiver/return unknown, unique declaration | Trace the value producer and return contract first. The sole compatible component may be considered by the existing opt-in speculative mode. |
| Receiver/return unknown, several declarations | Intersect owners answering the receiver's observed methods; inspect constructor/DI/caller evidence. Ranked speculation can pick a unique best eligible owner, but ties remain unresolved. |
| Selected component missing the method | Inspect parent, subclass selection, mixins, generated accessors and version compatibility. Do not replace a known receiver with an unrelated unique-name owner. |
| Selected native contract missing the method | Verify the engine/JDK API and deployment version; a missing local table entry is not proof the method is absent. |
| Bare binding unknown | Trace lexical scope, includes, inheritance and registration; the separate existing `--global-defs` policy can accept workspace-wide bare function definitions when explicitly enabled. |
| No declaration or unavailable source/context | Search dependencies and runtime registrations; classify source defects only when ownership and absence have appropriate evidence. |

Global uniqueness and a sole compatible receiver are different. Several components can declare `getBody`, while only one answers both `getBody` and `getTitle`. Candidate confidence describes ranking, not proof of origin. A component name matching a receiver variable likewise remains circumstantial.

Existing `--speculative`, `--speculative-ranked`, `--speculative-native` and `--assume-open-world` retain their opt-in behavior. Speculative connection JSON now also records `ownership: "unproven"` and `methods`, alongside the proposed component and policy. In ranked mode those method names describe the original receiver's first hop, rather than unrelated methods on returned objects. These records expose the evidence behind an inference without presenting it as a strict relationship.

## Validation

New regressions fail when the evidence, speculative labels and failed-hop lookup correction are withdrawn. They cover unique/ambiguous/unmatched source names, a failed intermediate hop with unrelated final-method declarations, explicit speculative evidence, and preservation of a known receiver mismatch even under speculation. Existing candidate and speculative tests cover compatible-owner intersection and opt-in behavior.

Corpus comparisons use the same seven roots, including the explicitly scanned Wheels vendor root, and compare finding keys per entry rather than only totals. Detailed results are under `target/resolution/method-evidence/`; metadata additions and corrected candidate lists are not counted as resolution gains.

### Measured declaration evidence

Across the unchanged strict scan, 12,309 individual findings receive method evidence; 46 broken-base summary entries omit it.

| Source name status in the scan index | Findings |
|---|---:|
| Unique source declaration | 3,214 |
| Multiple source declarations | 5,964 |
| No source declaration | 3,131 |
| Broken-base summaries, not individual method searches | 46 |
| **Total** | **12,355** |

These counts differ from the broader existence audit because the latter searches supplemental corpus dependencies and cached native API contracts. The scan evidence deliberately measures only indexed source declarations. The 3,214 unique matches are useful tracing candidates, not 3,214 proven fixes or automatically eligible speculative connections.

The corrected failed-hop search changes 14 candidate lists. Finding identities remain unchanged. The relationship labels partition individual entries into 9,799 receiver gaps, 1,200 return gaps, 412 other binding gaps, 196 inheritance binding gaps, 388 source/context gaps, 111 selected-component mismatches and 203 native-contract mismatches. Source-reviewed manual reclassifications in the existence audit remain a separate layer.

Final comparisons retain **12,355 strict findings**, **509 liberal findings**, **6,905 runtime assumptions** and **1,884 speculative receiver connections**. All fourteen project/mode finding comparisons have zero additions and removals. Assumption identities/policies and proposed receiver/component identities/policies also remain unchanged per entry. Coverage remains 8,151 indexed files and 189,154 CLI counting records per mode, with unchanged broken-base summaries and unchecked-call counts.

Normal `go build ./...`, `go vet ./...`, full short tests, full short race tests, `make lint`, the Go formatting check and `git diff --check` pass. Full tests used approved local socket access after the sandbox denied daemon socket binding. No published PR/CI result is claimed.
