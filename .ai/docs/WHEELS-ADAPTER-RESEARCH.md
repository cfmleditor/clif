# Wheels engine-adapter resolution research — 2026-10-09

Current implementation checkpoint: [implementation status](RESOLUTION-IMPLEMENTATION-STATUS.md).
The user has authorized implementation; research-only statements below describe
the earlier checkpoint. Only the explicitly recorded contracts are implemented.

Research only. This follows [the Model audit](WHEELS-MODEL-RESEARCH.md) and
[the measured method-record probe](RESOLUTION-IMPROVEMENT-PROBES.md). Source paths
below are relative to `/Users/garethedwards/corpus/cfwheels_cfwheels`.

## What can be proved without choosing an engine

The baseline contains 39 findings whose reason names $engineAdapter as a chain
supplier: 18 already find the helper but cannot establish its return component;
21 cannot find the helper. The 39 sites call 20 distinct downstream methods.

All 20 have explicit public declarations on wheels.engineAdapters.Base. Adobe,
BoxLang, Lucee and RustCFML adapters each extend that Base. Their overrides for
these methods retain the base's declared return type. None declares its own init;
Base.init (:14–18) returns this after recording the version.

Thus, **if the receiver is proved to be one of the four startup-created adapters**,
method availability does not require knowing the engine. This is a common API
proof, not permission to choose Base as the exact definition for an overridden
method. Navigation should retain concrete alternatives where known. Likewise,
methods declared any do not gain precise component returns from this audit.

The audit uses explicit source declarations and inheritance. It does not execute
constructors or establish runtime substitutability for arbitrary custom objects.

## Startup, publication and reload state

vendor/wheels/events/onapplicationstart.cfc:$init first copies supplied keys
into application (:11–14), then retains selected values from application.wheels
when it already exists (:22 onward). It replaces application.$wheels with a new
struct (:34), detects the engine, and constructs one of four adapters (:49–55).
The else branch constructs AdobeAdapter; do not assume a runtime installation
based on the machine running this research.

The new struct remains staged while further initialization executes. Routes,
Mapper and dispatcher are initialized before publication. At :443–444,
application.wheels is assigned the staged struct and application.$wheels is
deleted. Failures before that point can leave staged state alongside prior
published state or without a complete published adapter.

Global.$engineAdapter (:2837–2855) checks the published wheels slot first,
then the staged $wheels slot, then throws. Therefore "during initialization"
does not unconditionally mean it returns the newly constructed staged object:
if the published slot still has an adapter, that wins. Distinguish two object
identities even when both belong to the same adapter class.

The helper validates struct/key existence, not object ancestry. A custom field
write can therefore defeat the proposed common-base contract. Searching literal
engineAdapter assignments finds the four startup constructors plus test writes,
but cannot establish absence of computed writes, whole-struct replacement,
configuration includes or aliases. This is not a complete writer proof.

The existing engineAdapterScopedReadersSpec explicitly models startup failure:
it stores a saved adapter in a replacement staged struct, removes the published
field, invokes a reader, then restores both scopes. These tests are source
examples of the state transition, not runtime tests executed in this research.

## Controlled hypothetical return-contract probe

Using the already baseline-matched isolated corpus copy, a second probe changed
only the return declaration of Global.$engineAdapter from any to the fully
qualified wheels.engineAdapters.Base. It preserved line counts and restored the
copied source in a finally block. The original corpus and resolver were untouched.

Unlike the earlier inline-to-temporary record transformation, **this annotation
adds an assumption that the original body has not proved**. Its purpose is to
measure the value of closing the return gap, not validate the annotation itself.

| Source-copy condition | Before | After | Disappearing sites | New sites |
|---|---:|---:|---:|---:|
| Original layout + hypothetical adapter return | 11,527 | 11,509 | 18 | 0 |
| Equivalent record layout + hypothetical adapter return | 10,002 | 9,977 | 25 | 0 |

The additional seven are exactly the adapter chains exposed by the method-record
probe. Fourteen still-blocked chains occur in Model providers, where the helper
lookup relationship is absent. This matches the earlier Model audit: seven
isBoxLang, three coerceOracleObject, and one each of dynamicFinderProperties,
queryKeyColumnArgName, isOracleJdbcObject and isNumericStrict.

These counts measure two conditional experiments, not additive independent
fixes. In particular, 18 + 25 double-counts overlapping sites. Nor should the
1,525 record-layout result be described as an implemented reduction; the combined
hypothetical result is 1,550 disappearing sites relative to 11,527, subject to
all the assumptions and validation obligations in both research notes.

The 39-chain classification is narrower than an inventory including calls to
$engineAdapter itself. Keep helper availability and its downstream return
contract as different measurements.

## A runtime binding counterexample relevant to earlier research

BoxLangAdapter.prepareDIComplete (:192–194) assigns variables.this through the
supplied scope, whereas the Base method is a no-op. BoxLangAdapter.invokeMethod
(:200–211) also deliberately performs object[methodName]() in one expression.
Its source comment reports that extracting a method reference and then calling
that local loses the receiver; the Base implementation uses invoke().

This does not disprove Wheels' copied-method pattern. It identifies distinct
operations that must not be merged in a future rule:

- storing a reference in a receiving object's method slot and invoking that slot;
- obtaining a reference in a local variable and invoking the local;
- dispatching through an engine-specific invocation adapter.

A runtime fixture should cover all three, including an internal call to another
host method and a read of host state. The source comment's regression reference
is evidence to investigate, not an executed result in this session.

## Proposed resolution contract and required evidence

For the adapter opportunity, retain:

1. Active application owner and initialization path.
2. Published versus staged slot, with the helper's actual precedence.
3. Known constructor alternatives and later writes/whole-container replacement.
4. Common callable API across all proved alternatives.
5. Exact definition alternatives and return contracts, without treating a
   common ancestor as the only runtime implementation.
6. The throw path when neither slot contains the key.

Counterexamples should include both slots present with different objects,
staged-only startup, neither slot present, replacement by a non-adapter object,
subclass override, a test restoring a saved object, and direct adapter access
that bypasses the helper. No engine or type should be inferred from the field
name alone.

Prioritization remains: the measured record-layout mismatch first; bounded Model
method integration next; adapter return provenance as a smaller follow-up.
The common API narrows the return problem, but does not remove its lifecycle
conditions. Continue researching receiver origins for larger Mura groups rather
than substituting a broad dynamic acceptance rule.

## Evidence and review

`target/relationship-research/adapter-contracts.json` records declarations,
overrides and hashes for the five adapter sources. The hypothetical probe and
per-entry results are in `target/relationship-research/wheels-record-probe/` as
adapter_probe.py, adapter-summary.json and the two oracle reports/logs. All scans
used the same two roots and 1,710 indexed/scanned files; site comparisons preserve
multiplicity. No runtime validation, production change or new commit occurred.
