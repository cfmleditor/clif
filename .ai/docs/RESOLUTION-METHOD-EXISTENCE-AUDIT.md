# Missing methods: existence and ownership audit — 2026-10-11

The broader search finds **99 source-reviewed or contract-backed missing connections**, **six likely source/API issues**, and **five deliberate missing-method fixtures**. The other entries remain explicitly uncertain. This replaces the earlier mechanism-only classification with an existence search; it does not implement resolver changes or remove findings.

All **12,355 strict finding entries** are retained. These counts describe diagnostics, not distinct methods or independent bugs. A broken upstream receiver can produce several chain diagnostics.

| Evidence category | Entries | Meaning |
|---|---:|---|
| Missing connection: native contract exists | 74 | The missing method exists in the cached Lucee member registrations for the selected receiver family, or the corresponding local JDK public class contract. Engine/deployment version compatibility still matters. |
| Missing connection: reviewed source relationship | 25 | Source establishes registration, assignment, dependency identity, or the returned receiver contract. Includes four reviewed cases from the previous classification. |
| Likely source/API issue | 6 | The selected source component lacks the API; matches elsewhere belong to different owners, or no declaration was found. Runtime execution has not confirmed failure. |
| Intentional missing-method fixture | 5 | The source deliberately asks for an undefined method/function to exercise an error or undefined-value path. |
| Name exists, ownership unproven | 11,038 | A declaration or member registration with the failed name exists somewhere searched. That does **not** prove the call can reach it. |
| No declaration found, existence unproven | 773 | No searched declaration or member registration matches. Runtime UDFs, external dependencies and version-specific APIs can still supply the method. |
| Missing source or execution context | 434 | The component/base/context is unresolved before method ownership can be checked. Includes 46 missing-base summary entries representing 1,133 unchecked calls, not additional resolved or classified calls. |
| **Total** | **12,355** | Every baseline entry is assigned once. |

The previous report called 12,041 entries connection/context gaps based on the failure mechanism. That was not proof that all those methods exist. This audit deliberately keeps name presence and receiver ownership separate.

## What was searched

The search parsed **14,972 CFML source files**, including templates, components and generated property/ORM accessors, across the available projects under `/Users/garethedwards/corpus`. It included vendor trees and supplemental dependencies such as CommandBox. Dot directories, `node_modules` and `target` were excluded; no file-count limit was applied. There were no read failures. The original measured scan covered 8,151 files: this expanded search is research coverage, not a changed resolution baseline.

There were **2,495 requested spellings**, combining reported functions and failed method names extracted from reasons. In 1,221 baseline entries those names differ: searching only the final method would investigate the wrong hop. The inventory stores exact source locations and distinguishes generated accessors from ordinary declarations. It uses clif's parser, so parser omissions remain a limitation.

The audit also searched the cached official Lucee 7.1 function library member registrations, checking the receiver family, and inspected public JDK APIs with `javap` for Duration, LocalDateTime, Date, String, Class, Map and List. A Java declaration search found zero `.java` files in these available corpus checkouts; engine Java implementation coverage is therefore incomplete. JDK results describe this computer's installed JDK, not every deployment. The FLD snapshot is the source already examined in [the twenty-pass review](RESOLUTION-TWENTY-PASS-FOLLOWUP.md).

A native contract match establishes that the method exists on the contract corresponding to the reported type; it does not validate argument signatures, accessibility from every engine, the inferred receiver at runtime, or the return type needed for a later chain hop. No CFML engine tests were run.

## Actual issue candidates versus missing connections

| Classification | Findings | Source evidence and implication |
|---|---:|---|
| Likely stale test API: `Util.isInstanceCheck` | 3 | [UtilTest.cfc:10](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/core/util/UtilTest.cfc:10) creates a mock of Util and invokes the method. The same-name declaration belongs to the test itself, not Util. Util has no parent or missing-method handler. Check whether these tests need the current family/type API. |
| Likely wrong helper API: `helpers.humanize` | 2 | [AdminIntrospectionService.cfc:333](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/AdminIntrospectionService.cfc:333) and [AdminViewService.cfc:419](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/AdminViewService.cfc:419) inject `helpers@wheels-cli`. That [helper component](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/helpers.cfc:1) has no humanize, parent or missing-method handler. Humanize exists on Wheels Global and another command component. The guard checks whether helpers exists, not whether humanize exists. Runtime injection could change the outcome, so this remains a likely issue. |
| Likely legacy branch/API: `CacheBoxConfig.parseAndLoad` | 1 | [LoaderService.cfc:233](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/services/LoaderService.cfc:233) calls this API in the XML configuration branch. Selected CacheBoxConfig has no parent or matching declaration; none was found in the broader source search. Review branch support/version compatibility before changing the application. |
| Missing dependency connection: `shell.pwd` | 5 | Wheels injects the CommandBox Shell service. The supplemental [Shell.cfc:498](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/Shell.cfc:498) declares pwd. This is stronger evidence than an unrelated same-name match; dependency source was outside the Wheels scan. |
| Missing assigned UDF connection: `injectState` / `getObjectState` | 7 | [performance-parallel-tests.cfm:2](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/suites/async/performance-parallel-tests.cfm:2) defines both UDFs, then assigns them onto thisItem/thisPrototype before use. The methods exist as runtime bindings, not declarations on tests.tmp.User. |
| Missing injected UDF connection: `getMessage` / `getMessage2` | 2 | [LDEV1962.cfc:12](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1962.cfc:12) installs injectMixin and copies component1's functions into component2's variables and this scopes. Component2 intentionally lacks ordinary declarations for those names. |
| Missing mock registration: `virtualReturn` | 1 | [MockBoxTestHarness.cfm:31](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/tests/specs/mockbox/MockBoxTestHarness.cfm:31) registers `test.$("virtualReturn").$results(...)` before invoking the virtual method. |
| Wrong receiver connection: `getNormalizedID` | 3 | [JDBCStore.cfc:104](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/store/JDBCStore.cfc:104) passes this into JDBCMetadataIndexer; the indexer assigns arguments.store to variables.store. JDBCStore declares [getNormalizedID:506](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/store/JDBCStore.cfc:506). The diagnostic chooses the indexer instead of the stored dependency. |
| Wrong return receiver connection: `filter` | 3 | [AbstractFlashScope.cfc:85](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:85) declares getFlash's result as struct. The calls filter that returned struct; the diagnostic selects AbstractFlashScope. |
| Prior reviewed runtime registrations | 4 | Three CustomAssertions calls and one memoProbeInjected call have binding evidence recorded in [the earlier review](RESOLUTION-METHOD-CONNECTIONS.md). |
| Missing native contract coverage | 74 | Examples include Duration.get/parts/addTo, numeric sin/sqr/round, query slice/renameColumn/isEmpty, Date setters, and Class.getMethod/getResourceAsStream. Exact matched contracts are stored per entry; these are API existence findings, not implemented return-chain fixes. |

The five deliberate missing calls comprise the four previously reviewed Lucee negative fixtures plus [ColdBox testerror.cfc:32](/Users/garethedwards/corpus/ColdBox_coldbox-platform/test-harness/handlers/testerror.cfc:32): the comment explicitly says `getValuesss` will throw an error. These should not be hidden by liberal method matching.

## The former 310-entry contract review queue

| Current evidence | Entries |
|---|---:|
| Native contract exists | 74 |
| Source relationship reviewed | 21 |
| Likely source/API issue | 6 |
| Deliberate missing-method fixture | 1 |
| Same name exists, ownership pending | 145 |
| No declaration found, existence pending | 63 |
| **Total** | **310** |

Thus **95 of the 310 entries have positive connection evidence**. Another seven have source issue or negative-fixture evidence. The remaining 208 still need ownership, runtime registration, dependency or version review. No unmatched entry is automatically promoted to an actual bug.

High-priority unresolved examples include engine query/config APIs; ContentBox historical patch APIs and dbinfo; CommandBox ServerService.getServerInfoJSON; Mura's deprecated SOAP readByEmail and dynamic cfobject contracts; generated/inherited framework accessors; and methods selected on overly broad base types. Searching more names alone cannot finish these: trace the producer, injected dependency, subclass or runtime registration that establishes the receiver.

## Artifacts and reproducibility

[Audit summary](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/summary.json), [all classified findings](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/classified-findings.json), [full grouped Markdown findings](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/FINDINGS.md), and [manual source reviews](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/manual-reviews.json) preserve all entries and their evidence. Large declaration sets are intentionally retained rather than capped. These detailed artifacts live under ignored target; this report records the durable results and limitations.

The [inventory utility](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/inventory/main.go) and [classification script](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/audit.py) can regenerate the audit against these local paths:

```sh
source target/dev-env/env.sh
go run target/resolution/method-existence-audit/inventory/main.go > target/resolution/method-existence-audit/source-inventory.json
python3 target/resolution/method-existence-audit/audit.py
```

Validation checks the seven baseline SHA-256 hashes against the previous audit, partitions all 12,355 entries exactly once, and accounts for all 310 former contract-review entries. Baseline strict findings remain **12,355** and liberal findings remain **509**; no resolution improvement is claimed by this research pass.
