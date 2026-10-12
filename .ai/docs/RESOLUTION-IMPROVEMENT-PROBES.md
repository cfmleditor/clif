# Resolution improvement probes — 2026-10-09

Current implementation checkpoint: [implementation status](RESOLUTION-IMPLEMENTATION-STATUS.md).
The user has authorized implementation; research-only statements below describe
the earlier checkpoint. Only the explicitly recorded contracts are implemented.

Research only. This checkpoint prioritizes measurable resolution opportunities
without changing production resolver code or the original corpus. It complements
[the relationship model](RESOLUTION-RELATIONSHIPS.md) and
[Mura dynamic relationship research](MURA-DYNAMIC-RELATIONSHIPS.md).

## First priority: equivalent Wheels public-method records

A controlled source-copy experiment isolates the integration-plan record shape.
The current resolver recognizes a temporary reference followed by a record:

```cfml
local.ref = local.instance[local.fns[local.f].name];
ArrayAppend(local.publicMethods, {name=local.fns[local.f].name, ref=local.ref});
```

The extracted Global.cfc instead puts the same member reference directly in the
record. Both expressions are inside the same public-access filter. The earlier
research identified the mismatch; this experiment measures its downstream scope.

The existing wheelsIntegrationPlan check is shared by Mapper and Controller.
Controller lookup also uses it when resolving calls inside controller/view
method providers. That shared dependency is why the potential effect is larger
than the 543 findings explicitly naming Mapper.

### Experimental controls

- Built the current committed resolver into an ignored research directory.
- Copied the complete 14 MB Wheels corpus to a temporary directory; retained
  the original `cfwheels_cfwheels` basename and both explicit scan roots.
- Scanned the unchanged copy first. All 1,710 indexed/scanned files and all
  11,527 findings matched the saved baseline per entry after root normalization.
- Changed only the inline reference into the temporary-reference form in the
  copy. Preserved line counts, public filter, name expression and consumer code.
- Ran a negative variant sourcing the temporary from local.other instead of
  local.instance. Its findings matched the unchanged baseline exactly.
- Restored the copy and verified its Global.cfc bytes match the original.
  The original file's SHA-256 remained
  `79479347606190655ff3896d3dad97782f32497b1793c6e2889da7207511d213`.

Mirror construction mattered: naming its root `wheels` initially changed path
lookup and produced 55 extra findings. That run was rejected by the baseline
comparison. Using the original basename restored exact equivalence. This is
additional evidence that path context must be preserved in corpus experiments.

### Measured result

| Metric | Unchanged copy | Equivalent record copy |
|---|---:|---:|
| Indexed/scanned files | 1,710 | 1,710 |
| Unresolved report entries | 11,527 | 10,002 |
| CLI resolved-call count | 48,484 | 50,090 |

The per-entry multiset comparison, preserving multiplicity and omitting only
reason for site identity, finds **1,525 disappearing sites and zero new sites**.
Including reasons gives 1,532 removals and seven additions: seven existing sites
change diagnosis rather than disappear. The disappearing sites span 97 files
and 182 case-insensitive method names.

This is 13.23% of the Wheels baseline and 3.44% of the seven-report denominator.
It is an experiment on equivalent source syntax, **not a shipped reduction or
a guarantee that every accepted call is semantically correct**. Existing
resolver policies still require review, especially their host-binding assumptions.
Other corpora were not rerun; no cross-corpus gain is claimed. The CLI's resolved
count uses a different counting unit and is not substituted for site deltas.

| Location of disappearing sites | Count |
|---|---:|
| vendor/wheels/tests | 600 |
| vendor/wheels/view | 417 |
| vendor/wheels/controller | 189 |
| examples | 182 |
| vendor/wheels/mapper | 55 |
| app | 1 |
| Other Wheels corpus locations | 81 |
| Total | 1,525 |

Frequent methods include end (205), $draw (165), redirectTo (105), $get (88),
$args (83) and renderText (45). Representative original-source sites are:

- MapperNullPatternSpec.cfc report 42 / source 43: $draw on wheels.Mapper.
- examples/starter-app/app/controllers/Accounts.cfc report 27 / source 28:
  redirectTo through the Controller ancestry and copied methods.
- vendor/wheels/controller/caching.cfc report 13 / source 14: $args from the
  receiving Controller's Global context.

All seven changed diagnoses concern calls chained through $engineAdapter. They
now find the helper but lack its component return type: isBoxLang in rendering,
formsdate and links; globRegex/extractGlobVariable in matching; and image-format
helpers in assets. These seven remain unresolved and must not count as fixes.

### Narrow implementation proposal, still unimplemented

Recognize the two bounded forms of the same name/reference record while retaining
public filtering, source-instance identity, nonrecursive directory selection,
consumer execution, collision policy and host-specific return behavior. Avoid a
generic rule accepting any publicMethods-shaped structure.

Required validation before claiming the improvement:

1. Positive fixtures for both forms with identical definition and return results.
2. Negative fixtures for mismatched name/reference, wrong instance, private
   access, unused producer, changed consumer and competing methods.
3. Host-versus-direct-provider and inherited override checks; copying this-returning
   methods must not change the indexed source holder's return contract.
4. Disable the new recognition and confirm positive regressions fail.
5. Compare all corpus entries with the original sources. Separate disappearing
   sites, reclassified sites and newly exposed problems, retaining multiplicity.

The source-copy probe is evidence for prioritization, not a substitute for these
implementation regressions or runtime verification of function binding.

## Second priority: Model method integration

The probe leaves all 404 unscoped findings in vendor/wheels/model unchanged.
Model.cfc:init integrates wheels.model; its $integrateComponents (:575) and
$integrateFunctions (:598) use public records and preserve existing host names,
with super aliases. The current wheelsMixinHostFunc explicitly supports
controller/view/mapper, not model. Model factory-origin inference is present,
but it is a different relationship from installing Model methods.

This gives a concrete next research target rather than a presumed 404 fixes:
partition those calls by public source method, inherited Global helper,
peer-package method and downstream return. Trace application subclasses and
init/loader overrides before generalizing Controller support. Also inspect
calls on model instances outside the provider directory; their overlap with
factory/return work is not yet measured.

## Third priority: engine adapter lifecycle returns

Global.$engineAdapter (:2837–2855) returns application.wheels.engineAdapter when
available, otherwise application.$wheels.engineAdapter during initialization,
otherwise throws. events/onapplicationstart.cfc:49–55 constructs BoxLang, Lucee,
RustCFML or Adobe adapters according to engine state. Tests also replace/remove
these fields. A constructor in startup is not enough to infer a single permanent
class: active slot, lifecycle, branch and later writes matter.

Start with the seven newly exposed return gaps and prove the actual alternatives
and method availability. Do not assume one installed engine or count the method
names as a completed return proof.

## Reprioritizing Mura for measurable gains

The synthesized-registry research is useful for precise targets and return types,
but the current resolver already accepts missing object methods when it finds
onMissingMethod (resolve.go:2838 onward). missingChainHop (:2345 onward) also
accepts such a hop and treats the rest of the chain as dynamic.

Therefore a known Mura bean often does not need enumeration of its generated
methods merely to remove an unresolved finding. The approval-view findings in
the dynamic-relationship note report unknown receivers. Their upstream factory,
loadBy and template-scope origins should be researched first for count reduction.
Generated return contracts remain valuable for accurate navigation and continued
checking, which is a stricter outcome than dynamic acceptance.

This distinction prevents spending a large implementation effort on generated
method tables that would not remove the currently observed unknown-receiver
findings. It also prevents counting a broadly accepted dynamic call as a precise
resolved definition.

## Evidence and review

Ignored artifacts under `target/relationship-research/wheels-record-probe/`:
run.py, before/split/wrong-instance JSON reports and logs, accepted.json,
summary.json and attribution.json. The script asserts baseline equivalence,
negative equivalence and preserved line counts; finally restores copied source.
The attribution audit pairs changed diagnoses with multisets, not a dictionary
that collapses same-line duplicate calls.

No production implementation, original-corpus edit, new configuration, runtime
CFML test or commit was made during this research. The previous 43.62% family
inventory remains a scope metric; this experiment adds causal evidence for one
specific opportunity rather than inflating that percentage.

## Model audit follow-up

[Model research](WHEELS-MODEL-RESEARCH.md) partitions the 404 unaffected provider
findings: 388 plain calls have unique public candidates (151 Global, six Model,
231 sibling providers); 16 require downstream return proofs. Two plain calls
are parameter defaults rather than method-body statements. These are candidate
relationships, not another measured gain. The lifecycle trace separates API
installation from class/instance construction and selector-dependent returns.

## Adapter return follow-up

[Adapter research](WHEELS-ADAPTER-RESEARCH.md) establishes a shared public API for
all 20 method names in 39 adapter-chain findings. An artificial return-annotation
probe removes 18 sites alone or 25 after the record-layout transformation, with
zero new sites. The annotation supplies an unproved lifecycle assumption; this
is prioritization evidence, not a source-backed fix. Fourteen Model chains still
need helper lookup. Published/staged precedence and receiver binding remain
explicit proof obligations.
