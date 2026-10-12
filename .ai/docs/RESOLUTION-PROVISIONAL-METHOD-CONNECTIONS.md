# Provisional method connections — 2026-10-11

The [remaining 295 findings review](RESOLUTION-REMAINING-295.md) accounts for every residual entry by source/API evidence, relationship gap, unavailable dependency, template or intentional test. It separates likely source defects from unresolved runtime/version questions; it does not change resolution behavior.

The user authorized closing gaps with less reliable connections, then replacing them with reliable source relationships later. The new **`--speculative-methods`** batch option creates direct call-to-declaration shortcuts for remaining calls whose final method name has indexed source declarations. It is a broader policy than guessing a single coherent component for the receiver.

## Behavior and retained evidence

The ordinary strict scan and existing receiver guesses run first. Each remaining call searches indexed source declarations of its reported final function. The nearest declaration is selected; source-path ordering breaks ties. A sole source declaration uses `unique-global-method`; multiple declarations use `ranked-global-method`. Both policies mark ownership as unproven and target confidence as low. Framework stubs are excluded; generated source accessors remain eligible.

Each `speculativeMethodConnections` record keeps the original call, source location, diagnostic, method evidence and compatible candidates, plus the chosen target, up to seven alternative targets and the uncapped target count. Alternative display limits do not restrict the matching pool. The original method evidence can describe an earlier failed hop while the shortcut targets the final method: that difference explicitly preserves the chain being bypassed.

Calls with a proposed target move out of the unresolved list into this provisional connection ledger. **The resolved count does not increase.** With open-world classification enabled, this happens before residual runtime assumptions are classified, so some assumptions become concrete provisional source links. These categories account for different evidence, rather than asserting that the relationship has become reliable.

The option deliberately permits incompatible per-call targets for the same receiver spelling, a known receiver's method connecting to another component, private/inaccessible declarations and a final method connecting despite a broken upstream chain. It does not prove arguments, access, instance origin, return compatibility or runtime execution. It can therefore connect real source bugs to unrelated definitions; original diagnostics remain available for review. The method shortcut policy leaves calls with no matching source declaration and broken-base summary entries unresolved. When composed with open-world classification, that existing policy can still classify unqualified missing functions as runtime assumptions. No special fixture-name suppression rule is used.

A broken intermediate call that has its own diagnostic remains unresolved if its own method has no declaration, even when the downstream final-method diagnostic receives a shortcut. This prevents that missing hop from disappearing merely because another final method exists.

## Use and restoration

```sh
clif unresolved --json --speculative-ranked --speculative-native --speculative-methods --assume-open-world project > provisional-connections.json
```

`--speculative-methods` also works without receiver guesses or open-world classification. It emits an audit object with `calls`, `assumedCalls`, `speculativeConnections`, `speculativeMethodConnections`, `resolved`, `indexed` and `scanned`. The new field is omitted when empty. It is incompatible with writing a known-issues file; use the audit or verbose output to retain the provisional evidence. It is off by default and `--no-speculative-methods` switches it off; the last flag wins. Editor resolution does not change.

## Replacing shortcuts with reliable relationships

The ledger is the backlog for the next source-resolution pass. Prioritize entries by the preserved `methodEvidence.relationship`, then group by caller and receiver. For receiver gaps, trace constructors, DI, argument forwarding and assignments. For return gaps, trace the producer and concrete returned object. For binding gaps, trace lexical includes, mixins and registration. For selected-contract mismatches, verify the chosen base/subclass, injected provider and API version. Preserve competing targets until those relationships identify the real owner.

A genuine strict fix should remove the original strict diagnostic and make its provisional shortcut unnecessary with the flag enabled. Compare those identities per entry; a smaller provisional ledger alone is not proof of a reliable fix. Triage shortcuts selecting unrelated test functions or component APIs as possible source defects, rather than teaching strict resolution to accept the selected name.

## Validation and artifacts

New regressions cover unique and ambiguous names, wrong known receivers, bypassed upstream chains, no-declaration retention, flag disablement, composition with open-world reporting, low confidence, original diagnostic retention, JSON serialization and unchanged resolved counts. With classification disabled, the new behavioral regression fails. Source and checks are accompanied by corpus reports in `target/resolution/method-shortcuts/`.

## Corpus result

| Project | Previous liberal findings | With method shortcuts | Provisional method links | Remaining runtime assumptions |
|---|---:|---:|---:|---:|
| wheels | 143 | 59 | 1,151 | 265 |
| masa | 21 | 7 | 2,018 | 133 |
| contentbox | 44 | 41 | 394 | 432 |
| coldbox | 93 | 29 | 529 | 255 |
| lucee | 196 | 155 | 699 | 744 |
| testbox | 7 | 3 | 101 | 161 |
| fw1 | 5 | 1 | 203 | 34 |
| **Total** | **509** | **295** | **5,095** | **2,024** |

Remaining liberal findings decrease by **214 (42.0%)**, with zero additions. The new ledger contains **5,095** provisional method connections: **1,191 unique-name** targets and **3,904 ranked** targets. Of these, 214 replace previously remaining findings and **4,881** replace runtime assumptions. Runtime assumptions fall from **6,905 to 2,024**; existing speculative receiver connections remain **1,884**.

**Strict findings remain 12,355**. Proven/speculatively receiver-resolved counts do not change relative to the preceding modes; direct method shortcuts do not increment them. The per-entry union of remaining calls, runtime assumptions and new method shortcuts equals the preceding per-entry union, with no dropped diagnostic. All seven strict finding comparisons have zero additions and removals. Coverage remains **8,151 indexed files and 189,154 counting records per mode**, with unchanged broken-base entries and unchecked-call counts.

The new links group into 3,281 receiver gaps, 1,077 return gaps, 259 other binding gaps, 117 inheritance binding gaps, 244 missing-source/context gaps, 68 selected-component mismatches and 49 native-contract mismatches. Native mismatches here use same-name CFML source shortcuts, not newly verified engine API implementations.

### Reviewed weak selections

The three `Util.isInstanceCheck` calls select the test method named isInstanceCheck, not an API on Util. The two `helpers.humanize` calls select the generate/helper command component, not the injected helper. These five earlier likely-issue candidates now have explicit provisional source connections; they are **not reliable fixes**. `CacheBoxConfig.parseAndLoad` has no source target and remains a finding. These examples demonstrate why the original diagnostic and chosen-owner evidence must survive.

The deliberate ColdBox getValuesss call and Lucee nonExistentMethod / nonExistentMethod_ldev6349 calls remain findings. Two disabled Lucee notexisting probes remain open-world runtime assumptions, as before; none of the five deliberate missing calls becomes a method shortcut.

The remaining 295 entries include 46 broken-base summaries; native/config/query APIs, source-unavailable dependencies and unmatched source contracts need further research. The 2,024 runtime assumptions also lack concrete targets and remain distinct from the 5,095 declaration links.

[Combined provisional method ledger](/Users/garethedwards/development/github/clif/target/resolution/method-shortcuts/all-method-connections.json) and [per-project comparisons](/Users/garethedwards/development/github/clif/target/resolution/method-shortcuts/checked/summary.json) retain the full measured data. Detailed artifacts are under ignored target; this report records the durable results and reviewed limitations.

Normal build, vet, full short tests, full short race tests, final targeted race tests, lint, Go formatting and diff whitespace checks pass. Full tests use approved local socket access. No published PR/CI result is claimed.

[Grouped Markdown connection ledger](/Users/garethedwards/development/github/clif/target/resolution/method-shortcuts/CONNECTIONS.md) links each original call to its provisional target for follow-up review.
