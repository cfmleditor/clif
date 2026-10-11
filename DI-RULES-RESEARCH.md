# DI rules: source evidence before implementation

Current implementation checkpoint: [implementation status](RESOLUTION-IMPLEMENTATION-STATUS.md).
The user has authorized implementation; research-only statements below describe
the earlier checkpoint. Only the explicitly recorded contracts are implemented.

Research review: 2026-10-09. Production implementation is paused while the
relationships and their counterexamples are investigated. This ledger supersedes
the immediate implementation recommendation in RESOLUTION-RELATIONSHIPS.md.
The work packages there remain hypotheses about implementation boundaries.

The findings below come from reading local framework implementations and their
tests. Framework tests were inspected, not executed in a CFML runtime. Predicted
examples are deductions from those bodies, not observed runtime results. The
corpus is extracted source without Git metadata; file hashes below identify the
actual evidence instead of implying a known release or commit.

## What a rule must explain

For an unresolved call, establish the chain in this order: the active application
and lifecycle, the specific factory, its effective registration/discovery state,
the selected object, its assignment to the receiver, and the method available on
that object. Keep separate answers for object identity, dependency eligibility,
method availability and subsequent return values.

Each proposed rule needs an enabling condition, a selection procedure, competing
writes, a counterexample and a source dependency. Source text resembling a known
pattern is a lead; it is not sufficient evidence that the pattern executes.

## Rules recovered from framework bodies

Paths below are relative to `/Users/garethedwards/corpus`.

### WireBox registration is an ordered sequence of mutations

Evidence: ColdBox_coldbox-platform/system/ioc/config/Binder.cfc, map at 561,
to at 611, mapDirectory at 437, and Mapping.cfc alias processing at 698.

- map(alias) reuses an existing mapping unless force is true. A following to(path)
  mutates the current mapping even when map reused it. Therefore ordinary
  repeated map(id).to(path) can change a target without force.
- mapDirectory recursively selects files, derives basename IDs through mapPath,
  and calls mapPath with force=true. An earlier explicit map is not protected
  from a later directory registration. Reversing the executed order can change
  the result.
- mapDirectory preserves all selected mappings as the current mapping set;
  a chained asSingleton applies to that set. It is not just a modifier on the
  final enumerated file. BinderTest.cfc:539 checks this behavior.
- When process=true, each mapping is processed immediately. Mapping metadata
  can insert alias keys directly into the binder's mapping table. With lazy
  processing, those aliases may appear only after a mapping is requested.
- Different directory entries with the same basename can replace each other.
  The selected source does not supply an explicit sort for directoryList;
  do not invent alphabetical collision precedence. Alias references to an old
  mapping can also survive replacement of its primary key.

Counterexamples to investigate: explicit map before/after directory mapping;
two basenames with different metadata aliases; alias lookup before/after eager
processing; and repeated map(id).to(path) with a retained secondary alias.
These require runtime fixtures before certifying collision or temporal rules.

### Directory filters combine differently than a generic filter pipeline

Evidence: Binder.cfc:474–493. After its initial hidden-path check, acceptance is
the OR of: include regex matches; exclude regex does not match; custom filter
returns true; or no include/exclude/filter is supplied.

Deduction: when include and exclude are both present, a file failing include
can still pass because it is not excluded. A true custom filter can admit a file
that another clause rejects. Calling this an include-then-exclude pipeline
would change the runtime rule. The framework tests at BinderTest.cfc:512 onward
exercise each option separately; they do not establish the combined-option
matrix. Regex compatibility and the literal hidden-path predicate need their
own runtime review.

### WireBox discovery is ordered, contextual and stateful

Evidence: Binder.cfc:237, :1111, :1237; Injector.cfc:487–587, :706 and :785.

1. An explicitly named child injector redirects lookup before normal local
   resolution. DSL requests take their own object-builder path.
2. Existing local mappings outrank lazy discovery at lookup time. This is
   distinct from registration-time precedence above.
3. locateInstance checks the binder's ordered scan locations, then the direct
   instantiation path. It does not search every workspace basename.
4. If local discovery fails, getInstance checks child injectors, then the parent;
   a missing-dependency event can supply an instance before lookup throws.
5. A discovered path is registered locally. Later lookups can consequently use
   the mapping rather than repeat filesystem selection. Metadata processing can
   then add further aliases.

Binder.scanLocations accepts lists/arrays, ignores empty/already-present keys,
and stores expanded roots in an ordered struct. loadDataDSL adds ColdBox's
ModelsInvocationPath before DSL scanLocations, although the effective order
still depends on any earlier configuration calls and removals.

Counterexample: adding an earlier filesystem hit after an ID has already been
registered does not imply the next request switches paths. Distinguish an
uninitialized lookup policy from a live factory's accumulated state. Unknown
child ordering, event handlers and runtime mutations remain open boundaries.

### A module folder alone does not enable its model bindings

Evidence: ModuleService.cfc:628–746 and :907–923.

activateModule requires registration, respects the activate flag, skips already
activated modules and activates dependencies first. Within activation,
autoMapModels plus an existing models directory enable directory registration.
The package comes from classMapping/modelsLocation or modelsInvocationPath.
moduleInjector selects local ownership; modelNamespace controls the qualified
registration. The module's onLoad runs later in activation and can add bindings.

Concrete application evidence: ContentBox's ModuleConfig.cfc:144 adds @cb via
mapDirectory in onLoad. A lexical search of modules/contentbox found @cb only
at that registration, so this mechanism is not yet linked to a literal unresolved
consumer in that subtree. It is evidence of a rule, not evidence of large yield.

### Wheels retrieval and automatic injection use different eligibility rules

Evidence: cfwheels_cfwheels/vendor/wheels/Injector.cfc:159, :206–254, :282,
:318–322 and :352–361; InjectorSpec.cfc:114 and :213 onward.

getInstance chooses an alias mapping when present and otherwise uses the name
as the component path. containsInstance checks registered mappings only.
Automatic init argument supply uses containsInstance, so an existing directly
constructible component does not automatically become an injectable dependency.

A nonempty initArguments struct bypasses automatic argument supply entirely;
it is not merged with inferred dependencies. The nearest inherited init is
selected by metadata traversal. Construction invokes init but returns the
created component, ignoring init's return value, then calls onDIcomplete if
present. A dependency's name, not its declared argument type alone, drives this
container's automatic lookup.

Counterexamples: direct getInstance("models.Service") versus an unregistered
constructor parameter Service; partial explicit initArguments with another
required dependency; and an init returning a different object.

### DI/1 has different collision, eligibility and constructor-return rules

Evidence: framework-one_fw1/framework/ioc.cfc:199, :512–592, :601–629 and
:823–886; framework/one.cfc:2478 and :2706.

Discovery follows configured folders in order, uses configured recursion and
substring exclusions, reads metadata and skips interfaces. It generates
basename and basename+singular(folder) identities, unless aliases are omitted.
Within discoverBeansInFolder, repeated bare names are removed and tracked as
duplicates, while a colliding qualified name throws. The duplicate tracker is
local to that folder invocation: cross-root interactions need examination
before simplifying this to a global uniqueness algorithm.

Direct retrieval allows transient beans; setter discovery prunes transient
dependencies. Required constructor arguments missing both an override and a
bean cause failure; missing optional arguments are omitted. A non-null init
return replaces the created bean. WireBox's buildCFC and Wheels construction
ignore that return instead. DI/1 constructorArgs/registration override collision
precedence remains a runtime question because the source uses structAppend
without an explicit overwrite flag.

FW/1 supplies DI/1 defaults through its own configuration: diLocations is model
plus the configured controllersFolder, not invariably literal controllers.
Its WireBox setup has separate binder-path and legacy-properties branches;
only the latter explicitly adds diLocations as scan locations. A generic
"FW/1 means DI/1 over model/controllers" rule loses these enabling conditions.

## Trace one real unresolved relationship end to end

The subsequent [Wheels relationship trace](WHEELS-RELATIONSHIP-TRACE.md)
reviews the actual startup sequence, live versus retained factory references,
provider failure side effects, method copying and report line coordinates. It
qualifies the simple chain below rather than claiming that every edge is closed.

The Wheels g/application.wo group has a concrete candidate chain:

tests/Application.cfc extends ApplicationProxy; public/ApplicationProxy.cfc
extends Application; public/Application.cfc:onApplicationStart constructs the
Injector with wheels.Bindings, then assigns getInstance("global") to
application.wo. Bindings.configure maps global to wheels.Global. Injector
construction returns the created Global; test files such as urlforSpec.cfc
assign g = application.wo before invoking its methods.

This proves the relevant source edges exist. It does not yet prove all runtime
application ancestry, startup reachability or receiver writers. The test
Application overrides onRequestEnd but not onApplicationStart. onError has
separately guarded factory/object recovery and must not be flattened into an
unconditional second startup. Later plugin/service registration can mutate the
container; a later rebind does not retroactively change an already stored wo
object. Test injectors also self-register application.wheelsdi during init.

The next research review should inventory writers to application.wo and g,
identify the effective mapping/parent context, and distinguish the object
already stored in wo from future factory lookup. Only then inspect which Global
methods and subsequent returns each unresolved site needs.

## Research acceptance gate

Before resuming implementation of a relationship, require a worked source trace
for at least one actual unresolved use, a paired counterexample, a defined
runtime/context boundary and a list of unsupported mutations. Compare framework
tests with the recovered rule; execute small CFML fixtures where engine semantics
or ordering determine the answer. A source-derived prediction is not a passing
runtime test. Prioritize real broken chains rather than the number of conventions
found. Keep these questions open until evidence resolves them:

- Directory collision order, aliases retained after replacement, and lazy alias
  registration timing.
- Combined filter semantics across supported runtime versions.
- DI/1 cross-root duplicate handling and constructor override collision behavior.
- Effective module activation/injector hierarchy for each reported consumer.
- Application ancestry and all reachable writes for the stored Wheels Global.

## Source fingerprints

SHA-256 of the inspected local bodies; all roots are under the corpus directory.

| File | SHA-256 |
|---|---|
| ColdBox_coldbox-platform/system/ioc/Injector.cfc | 0f72221dd85ee621e500cd9b8c2db34cf4bebd897e4789ed0970e7c53a6ebd21 |
| ColdBox_coldbox-platform/system/ioc/config/Binder.cfc | e3fa40362d6ed53802f01c5071e9aca7220dc5c26fe29da4e81be1a50c54f745 |
| ColdBox_coldbox-platform/system/ioc/config/Mapping.cfc | 56990af5698374addbe453ca5d6fb9cc22a8aa59c4ba0ba285435f2d6903445f |
| ColdBox_coldbox-platform/system/ioc/Builder.cfc | 36c45a462ea73c696db484d2ae61883a22687ba33eaf82154129b89e505f3251 |
| ColdBox_coldbox-platform/system/web/services/ModuleService.cfc | cfa74f8e5ffa2b465ea5c29d8536f3193471b1e1577abff8798c9198965bf870 |
| framework-one_fw1/framework/ioc.cfc | f00c03a51bbb68bedf128342ae35715dcb31bcc995571120b56b7d5a4df0181b |
| framework-one_fw1/framework/one.cfc | c01bbffeb1a55e2038bb6d1ad9fd01c9e81d65eb3d9b6b2159e1e9a9aee8af5a |
| cfwheels_cfwheels/vendor/wheels/Injector.cfc | 1b629346db7fd62a45b3dbbe3a394f608f36bd107e6040ea8dfd64a58e7ee1e2 |
| cfwheels_cfwheels/vendor/wheels/Bindings.cfc | 02f32a3e59c5d6f5118eb21becf5ab79438a771bd2c87efc6074a1fc52826882 |
| cfwheels_cfwheels/public/Application.cfc | c20c3eeb00136e0972d1eb6002bb468de52fe44b5972cd7c71a63735d2d6cb1e |
| Ortus-Solutions_ContentBox/modules/contentbox/ModuleConfig.cfc | 8d8c6bbcbb4376cef2a85b743c8f48f1dfa0bfe9f8aa60c8dfa87d3481449d87 |

Implementation checkpoint: application shared-startup discovery now follows
bounded inherited lifecycle selection, and provider callbacks can transfer
actual invocation argument identity. This does not yet make diSources an
effective lifecycle evaluator or give a factory class constructor-specific
binder identity. See RESOLUTION-IMPLEMENTATION-STATUS.md checkpoint 2.
