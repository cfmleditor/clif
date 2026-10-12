# Earlier findings: implementation completion audit

Review date: 2026-10-11. This is the current closure checklist for the earlier
research and implementation programme. User authorization to implement remains
in effect. Historical statements that implementation was paused describe their
checkpoint; they do not revoke that authorization.

**The full programme is not complete.** Passing foundation tests, a lower total,
and a speculative connection do not establish that every proposed relationship
is implemented. The rows below map the documented families to code and tests,
identify the supported boundary, and retain outstanding obligations.

Later bounded deliveries are recorded in [the liberal implementation](RESOLUTION-LIBERAL-IMPLEMENTATION.md),
[collection callbacks](RESOLUTION-COLLECTION-CALLBACKS.md),
[collection/ORM opportunities](RESOLUTION-OPPORTUNITIES-IMPLEMENTED.md), and
[collection keys and struct values](RESOLUTION-COLLECTION-VALUES.md).
Top-level callback ownership, ranked candidates, chain argument preservation,
collection callback inputs, array-preserving chains, current cborm ORM utilities,
key-array elements and closed returned-struct value metadata have implementations.
Whole registry lifetime/closure ownership and map output contracts remain open.
The latest strict count is 13,249; liberal mode retains 788 plus 6,725 assumptions.

“Implemented” means a bounded rule exists with regression tests. It does not mean
all unresolved findings in that framework are fixed. “Assumption” means an
implemented source-layout/default policy with documented override boundaries.
“Experimental” means an explicitly enabled unreliable rule. “Open” means there
is no complete implementation of the specified relationship. Native work has passed the local build, vet, short/race and lint checks;
its measured follow-ups are checkpoints 32 and 34 in the implementation status
record, with the bounded Mura owner recorded in checkpoint 33.

## Closure ledger

| Earlier finding | Current evidence | Closure and remaining scope |
| --- | --- | --- |
| Wheels inline public method records and Model copied bodies | `wheels_controller.go`; `TestWheelsInlineIntegrationRecord`, `TestWheelsModelIntegration`, boundary/source-change tests in `wheels_relationship_test.go`; checkpoints 1–3 | **Implemented, bounded.** Public providers, override selection and receiving body ownership. Copied Model parameter defaults remain **open** despite the ColdFusion runtime observation. |
| Application ancestry, lifecycle selection and inherited mappings | `startup_application.go`, `startup.go`, `application_sections.go`; `TestSelectedApplicationLifecycle`, `TestMissingMappedParentIsDiscoveryDependency`, `TestIncludedLifecycleDeclarationsAreSelected`; checkpoints 2, 8, 10 | **Implemented, bounded.** Arbitrary conditional/computed startup execution is **open**. |
| Provider callback and literal invoke argument transfer | `callback_provider.go`, `invoke_arguments.go`; invocation, block-arrow, mutation and ambiguous-tag tests; checkpoints 2–3, 12 | **Implemented, bounded.** Unsupported callback/arrow origins and unknown argument collections withhold inference. |
| Call identity, closure source bytes and chain hops | `call_spans.go`; origin, tag closure and chained-hop tests; checkpoints 3, 8, 30 | **Implemented, partial.** The separate top-level named-call emission in tag script islands identified in ten-pass research is fixed in the current batch, with `TestTagFunctionScriptIslandNamedCallsKeepIdentity` failing before the fix. The final Mura scan removes all 65 identified changeset argument findings and 41 other sites; overall Mura falls by 106. |
| Private constructor state, ordered setters and literal registration tables | `producer_instance.go`, `producer_flow.go`; ordered-registration, constructor, mutation, alias, exposure and cross-application tests in `producer_instance_test.go`; checkpoints 4, 6–9 | **Implemented, bounded.** Mutable aliases, escaped tables, unknown writers, caches and arbitrary framework binders remain **open**. |
| FW/1 DI ownership, selected hooks, subsystem/configuration policy | `di.go`; inherited policy, ordered configuration, conditional retrieval and lifetime tests in `di_test.go`; checkpoints 5, 7, 10, 20 | **Implemented, bounded.** Arbitrary inherited/configured execution and stored collection state remain **open**. |
| WireBox module discovery and duplicate model IDs | `wirebox.go`, `discovery_cache.go`; module eligibility and duplicate-basename tests in `wirebox_test.go` | **Implemented, bounded.** Discovery eligibility is not proof of arbitrary runtime map/to mutation. |
| Literal assertion registration and reporter/include host arguments | `assertion_registration.go`, `includes.go`; selected setup/override tests and `TestReporterIncludeTransfersQualifiedArgument`; checkpoints 10, 12 | **Implemented, bounded.** Runtime installation/replacement of renderer/assertion methods is not generally modeled. |
| Exact positive type guards and literal return-mode selection | `guarded_receiver.go`, `producer_flow.go`, `wheels_mapper.go`; `guarded_receiver_test.go`, existing Mapper and producer fixtures; checkpoint 11 | **Implemented, bounded.** Negative guards and correlated heterogeneous alternatives are **open**. |
| Immediate Wheels Injector result and application.wo aliases | `wheels_injector.go`; `TestWheelsInjectorStartup`; checkpoint 13 | **Implemented, bounded.** This does not close mutable factory registration or every model selector. |
| Out-of-box project mappings/root assumptions | `project_assumptions.go`, `wheels_root.go`, `lucee_admin.go`; layout, competing-root and explicit-override tests; checkpoints 14, 30; `PROJECT-ASSUMPTIONS.md` | **Assumption implemented.** `/ → public`, library namespaces and marked Lucee `/admin` context require matching layouts. Explicit configuration remains authoritative. |
| Source-shaped framework/tooling defaults and invalidation | `assumed_frameworks.go`, `assumed_source_tools.go`, discovery caches; config/CLI/server tests; checkpoints 15, 24–25; `PROJECT-ASSUMPTIONS.md` | **Assumption implemented.** Missing host markers, competing applications and explicit overrides reject automatic selection. Not equivalent to adding every preset project-wide. |
| Mura service-name loop, literal beans, factory/setter ownership, admin payloads and published services | `mura_beans.go`, `beans.go`, `setters.go`, `startup.go`; literal startup-policy, registration/lifetime and managed-setter tests; checkpoints 16–23 | **Implemented as documented bounded assumptions.** Mutable selected factories, generated objectMappings, disk restoration and arbitrary registry updates remain **open**. |
| Caller file/site budgets and optional limits | `arg_callers.go`, `caller_inference.go`; ownership, former-limit and optional-limit tests; checkpoints 26–28 | **Implemented.** Defaults unlimited; configured positive cutoffs apply. Recursive/depth safety and incomplete-origin vetoes remain intentional. |
| Omitted arguments versus unknown supplied values | `arg_callers.go`, `invoke_arguments.go`; unknown omission, unknown collection and chained omission tests; checkpoints 29–30 | **Implemented, bounded.** Unknown supplied values still veto. Mura feed forwarding's competing unknown supplier is **open**, not a forgotten size-limit fix. |
| Closed entity collections and callback elements | `loop_element.go`, `collection_callbacks.go`; entity-load, loops/each and shadow/unknown tests; checkpoint 29 | **Implemented, bounded.** Generic Mura iterator instance state, heterogeneous Field/Group sets and HQL result-shape parsing remain **open**. |
| Lucee embedded admin helpers | `lucee_admin.go`, existing include traversal; source-layout mapping tests; checkpoint 30 | **Assumption implemented and measured:** 99 historical removals. Ordinary include reachability still applies. |
| Sole consistent CFC candidate connections | `speculative.go` in unresolved/resolve; opt-in, wrong-origin, ambiguity and incomplete-group tests; checkpoint 31; `RESOLUTION-SPECULATIVE-CONNECTIONS.md` | **Experimental implemented.** Off by default; origin remains unproven. Caller-veto relaxation, hardcoded origin overlays and lexical-first selection remain **research only**. |
| Native ORM Session versus factory Statistics | `nativetype/contracts.go`, parser producers and resolver native return propagation; `native_types_test.go`, `contracts_test.go`; ORM/native research docs | **Implemented and measured in current batch:** all ten earlier explicit ORM findings removed. Hibernate 5.6 API assumption does not prove installed extensions or bridge compatibility. HQL entity loops are a separate **open** rule. |
| Native Object/String/Class, Java collection/JDBC/time and Lucee engine results | `nativetype/contracts.go`, `native_types.go`, builtin return lookup and chain traversal; family/result/protected-method tests | **Implemented, bounded in current batch.** Erased results remain dynamic; native loop generic propagation, all API overloads/version selection and arbitrary Java classes are not complete. |
| Qualified builtin failures hidden by reporting | `unresolved.go`, `native_types_test.go` | **Implemented in current batch.** Only unqualified global builtin failures are filtered. Restored findings change the counting coverage and must be compared against a reporting-only control. |
| Native family guesses with a separate flag | `unresolved/speculative.go`, `--speculative-native`, `--no-speculative-native`; ambiguity, incomplete-group and reversible-option tests | **Experimental implemented in current batch.** Checkpoint 34 adds staged String/Array/Struct/StringBuilder fallback, preserving earlier family answers and labelling primitive proposals. Unique usage-family match and stage priority are not source provenance. Engine/version, origin and chain-role obligations remain open. |
| Wheels staged/published engine adapter, selected models and return alternatives | `WHEELS-ADAPTER-RESEARCH.md`, `WHEELS-MODEL-RESEARCH.md`; existing common-base registration is only a foundation | **Open.** Need publication/copy/replacement/restore ownership and all constructor/return alternatives. The 55 adapter cohort is not a verified gain. |
| Mura renderer/template arguments and iterator records | `MURA-DYNAMIC-RELATIONSHIPS.md`, ten-pass sections 5/7, remaining-causes | **Open.** Need theme/custom-directory selection, retained include argument scope, entity selection and iterator state. 128 template and 42 selector entries are investigation populations. |
| Record fields, cache returns and dynamic module config | ten-pass section 6; existing `member_receiver.go` and producer records are partial foundations | **Open.** ColdBox ehBean cached/fluent return and dynamic config identity differ. ContentBox criteria supplied alternatives and `{count,content}` returns also require closure. |
| Mura embedded admin FW/1 view ownership | `mura_admin.go`, Mura implicit-base fallback, governing-owner/override/invalidation tests in `mura_admin_test.go`; checkpoint 33 | **Assumption implemented, bounded.** Standard marked admin owner, actual renderer dispatch and parameter includes are required; no additional FW/1 preset. Net strict reduction 361 with unchanged checked-record coverage. Custom admin bases, manual template execution and runtime method mutation remain outside the convention. |
| Lucee Field/Group driver collections and terminating negative guards | ten-pass section 7, remaining-causes; positive guard/closed homogeneous collection tests do not cover this | **Open.** Need closed Field/Group alternatives, driver provenance, relative identity and branch termination before subtracting Group. |
| Missing migration/ORM dependencies and broken test bases | remaining-causes, engine ORM research, gaps-plan section 9 | **Open or external source/configuration requirement.** Do not substitute engine natives for ORMUtilFactory or host source for test doubles. Adding roots changes coverage and needs a separately measured scan. |

Paths in the evidence column are under `internal/resolve` unless another package
is named. Test names are searchable anchors, not a claim that every earlier
proposal has an individual fixture.

## Research source reconciliation

The ledger reviews the programme's ten categories in `RESOLUTION-GAPS-PLAN.md`
and the follow-up checkpoints in `RESOLUTION-IMPLEMENTATION-STATUS.md`, against:

- `RESOLUTION-RELATIONSHIPS.md`, `DI-RULES-RESEARCH.md` and
  `RESOLUTION-CALL-CHAIN-LEDGER.md`: foundations exist; mutable lifetime,
  receiving-slot/default invocation and dependency ownership remain bounded.
- `RESOLUTION-RESEARCH-COVERAGE.md`, `RESOLUTION-IMPROVEMENT-PROBES.md`,
  `RESOLUTION-ITERATION-REVIEW.md`, `RESOLUTION-USER-REVIEW.md`: historical
  populations/probes are not fresh counts or automatically completed contracts.
- `MURA-DYNAMIC-RELATIONSHIPS.md`, `WHEELS-MODEL-RESEARCH.md`,
  `WHEELS-ADAPTER-RESEARCH.md`: body integration and startup assumptions do not
  close registry/model/adapter state or copied defaults.
- `PROJECT-ASSUMPTIONS.md`: default layout/preset policy is implemented with
  explicit configuration priority; those assumptions are distinct from proof
  of runtime object origin.
- `RESOLUTION-CONTEXT-AUDIT.md`, `RESOLUTION-REMAINING-CAUSES.md` and
  `RESOLUTION-TEN-PASS-RESEARCH.md`: Lucee admin mapping and chain continuation
  origins were delivered; checkpoint 32 closes tag script-island identity and checkpoint 33 closes the
  bounded standard Mura admin owner. Custom template arguments, selectors,
  records and heterogeneous collections remain.
- `RESOLUTION-SPECULATIVE-CONNECTIONS.md`: sole CFC candidates are implemented;
  other overlays have counterexamples and must not be called production fixes.
- `RESOLUTION-ENGINE-ORM-RESEARCH.md`, `RESOLUTION-NATIVE-TYPE-RESEARCH.md`:
  current native batch addresses producers/results/reporting and a separate
  guess flag; runtime/version and collection-loop boundaries remain explicit.

`TASS-SUPPORT-PLAN.md` is a separate project-support plan, not a measured
seven-corpus closure claim. Its requirements must be evaluated against that
project and configuration before declaring project support complete.

## Review order for remaining implementation

1. The known script-island emitter and all 65 identified changeset argument
   findings are closed by the current fix. Audit other source-identity paths
   separately; the wider parser-origin programme is not declared complete.
2. Standard governing-owner-scoped embedded Mura admin context is implemented
   in checkpoint 33. Custom base/directory and runtime mutation remain separate.
3. Preserve closed value alternatives and record/iterator origins. Start with
   source-proven records, then Field/Group termination and Mura entity selectors.
4. Extend actual factory/registry/adapter state and template selection only
   when writer, escape, replacement and lifetime boundaries are represented.
5. Evaluate absent dependencies in separately configured scans, reporting added
   files and unchecked-base coverage rather than suppressing missing sources.

## Completion criteria

Close a row only with its implementation boundary, positive regression,
rejection/counterexample coverage, fail-before result and corpus per-entry
comparison. Record files scanned, checked-record coverage and unchecked base
calls. Add the exact tested commit/CI result only when a published head was
actually checked. Count experiments separately from strict resolutions.

No blanket “all earlier findings implemented” claim is justified by this audit.
