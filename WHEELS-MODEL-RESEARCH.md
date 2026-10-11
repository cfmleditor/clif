# Wheels Model resolution research — 2026-10-09

Current implementation checkpoint: [implementation status](RESOLUTION-IMPLEMENTATION-STATUS.md).
The user has authorized implementation; research-only statements below describe
the earlier checkpoint. Only the explicitly recorded contracts are implemented.

Research and planning only. This follows the measured record-layout opportunity
in [Improvement probes](RESOLUTION-IMPROVEMENT-PROBES.md). The Model findings below
are a declaration audit, not an implemented or experimentally measured reduction.
Source paths are relative to `/Users/garethedwards/corpus/cfwheels_cfwheels`.

## A bounded next opportunity

The saved report has 404 unscoped findings under vendor/wheels/model. The prior
record-layout experiment left all 404 unchanged. Of these, 388 have reason
`no qualifier, not in file`; all 388 names have exactly one direct public
candidate across Model.cfc, Global.cfc and the model provider files.

| Candidate location | Plain calls | Additional chained calls | Total |
|---|---:|---:|---:|
| Global | 151 | 0 | 151 |
| Model | 6 | 0 | 6 |
| Model provider files | 231 | 2 | 233 |
| No direct declaration in that set | 0 | 14 | 14 |
| Total | 388 | 16 | 404 |

There are no duplicate candidate names in this audited declaration set. This
says nothing about application overrides, plugin writes, inherited provider
metadata or declarations outside the set. Of the 404 sites, 402 have report
callers matching explicit public declarations in their source file. The two
remaining calls occur in parameter defaults and are discussed separately below.

The largest name groups are $args (36), $callback (26), primaryKeys (25),
$quotedTableName (17), $invoke/$get/$listClean (16 each), and $engineAdapter (14).
This supports a focused method-host investigation rather than hundreds of
individual method-name exceptions.

## Relationships that create the Model API

Model.cfc extends wheels.Global and its init (:4–7) integrates wheels.model.
The integration routine (:575–587) asks for a shared component integration plan,
gets the model mixin override set, and passes each publicMethods record array
to $integrateFunctions (:598–620).

The consumer preserves an existing name in variables/this. Otherwise it copies
the record's reference there. When an existing name is present it installs the
framework reference as superName; a planned mixin override also installs that
alias. Global's public record builder is the same one examined in the previous
source-copy experiment. Recognizing its inline layout is necessary but does not
add Model support by itself.

The current wheelsMixinHostFunc in internal/resolve/wheels_controller.go
supports mapper, controller and view. It rejects model. Controller's package
lookup enumerates only controller/view. Existing wheelsModelReturn proves
factory results under selected source contracts; it does not implement this
method installation relationship. These are distinct missing links.

Representative source/report evidence:

- model/associations.cfc report 23 / source 24: belongsTo calls $args, supplied
  by Global after host binding.
- model/sql.cfc report 952 / source 953: $whereClause calls $softDeletion,
  declared directly on Model at source 559.
- Calls to $callback and primaryKeys use public methods in sibling provider
  files, supplied by the same completed integration loop.

A prospective rule must retain the actual Model host and the original function
identity. Direct invocation on a provider component is a separate case; the
provider's directory name alone cannot prove that invocation uses Model state.

## Lifecycle ordering and overrides

The extracted Global.$createObjectFromRoot (:1097–1108) uses the Injector to
obtain an instance, then invokes the requested initializer for non-init methods.
Injector.getInstance calls init and then onDIcomplete (:233–250).
Model.onDIcomplete (:656–660) prepares the engine-specific state and initializes
plugin mixins. The later $initModelClass initializes class state and calls the
application config method if present (:71–73).

For this source path, the sequence is therefore:

construction → inherited/overridden init → method integration → onDIcomplete
and mixins → requested $initModelClass → application config.

This explains how application config can call table/property/validation methods
that are not directly declared in its ancestry. For example starter-app's
app/models/auth/LDAP.cfc extends app.models.Model, which extends wheels.Model;
LDAP.config calls table(false), property and validatesPresenceOf. But that
ancestry must be resolved in the example application's context. Selecting some
other app.models.Model by basename is not sufficient.

The existing Controller safeguards provide useful questions, not an automatic
Model implementation: does a descendant replace init, integration routines,
plan generation or override-set generation? Does init call its parent? What
methods or slots are replaced by onDIcomplete? Which method survives as the
normal name and which as superName? Model support must answer these for its
actual lifecycle.

## Two default expressions require distinct evidence

The report leaves caller empty for:

- model/read.cfc report 529 / source 530: findFirst's interpolated default
  `property="#primaryKey()#"`.
- model/sql.cfc report 1703 / source 1704: $keyWhereString's default
  `properties=primaryKeys()`.

Both names have public sibling-provider candidates. Neither site is an ordinary
statement inside the method body. Keep them apart from the 386 plain calls with
known public callers. A runtime fixture should establish the binding/evaluation
context of defaults on copied methods for supported engines, including a caller
that supplies the argument so the default is not used. Do not infer a top-level
execution context merely from the missing parser caller field.

## Method availability does not prove chained return types

Fourteen chained sites call engine-adapter methods: isBoxLang (7),
coerceOracleObject (3), and one each for dynamicFinderProperties,
queryKeyColumnArgName, isOracleJdbcObject and isNumericStrict. They are not
methods expected on Model/Global/provider files. Their upstream $engineAdapter
helper must first bind to the host; its lifecycle-dependent return remains the
separate research obligation recorded in Improvement probes.

Two other chains start at model(...), then call $expandedAssociations or
$queryRowToStruct. Finding those downstream names in providers is only candidate
evidence. The selector, target model class and its installed API must be proved.

The same separation matters beyond these 404 sites:

- model/create.cfc:new (:55–69) calls $createInstance, then returns its result.
- $createInstance (:100–133) selects a file using the host's class.modelName and
  class.path, invokes $initModelObject through $createObjectFromRoot, executes
  callbacks, then returns the new value.
- Model.$initModelObject (:483 onward) initializes instance state and retrieves
  cached class state. A returned instance is not automatically the cached class
  object from model(name), even when their component path is the same.
- model/read.cfc:findAll (:362–399) has a returnType escape to a query plus
  returnAs branches for SQL, queries, structs, arrays and objects.
- findOne (:508–517) delegates to findAll, can unwrap an array to its first
  element, and returns false for an empty array when returnAs is not array.

Thus adding public method lookup must not introduce universal rules such as
`findOne -> Model`, `save -> this`, or `new -> this`. save is explicitly Boolean
in this source. Return identity and optional values need their own evidence.

## Proposed next slice and acceptance criteria

First prioritize the already measured inline-record compatibility change.
For Model, split the next proposed work into separate reviewable contracts:

1. **API installed on a Model host:** exact init/producer/consumer and path
   evidence; preserve overrides, collision behavior, aliases and invalidation.
2. **Calls inside copied public bodies:** use the actual host to find Model,
   Global and peer-provider methods; keep direct-provider calls distinct.
3. **Defaults and returns:** leave as independent obligations until their
   evaluation context, selectors and identities are established.

Required counterexamples include a private method, a provider with its own
ancestry, an unused loader, altered copy policy, duplicate methods, descendant
init/loader overrides, plugin replacement, direct provider invocation and a
copied default expression. Check files being added/removed from integration
directories as well as source content changes when reviewing caches.

The declaration audit makes 388 plain sites a plausible opportunity, but no
claim that all will be resolved is justified yet. Do not add 388 to the earlier
1,525 experiment as a proven cumulative gain. Application-level consumers lie
outside this 404-site selection and have not been counted here.

## Verification record

Read-only Go extraction used the repository's ParseFunctionDefs; a Python audit
matched report names and caller names to those declarations and checked explicit
access on declaration lines. All 390 matching target declarations for this
selection are explicitly public. The two unknown caller entries were inspected
in source and are defaults. Artifacts and source hashes are under
`target/relationship-research/model-audit/` (main.go, declarations.json, audit.py,
sites.json and summary.json). This is not a full CFML metadata emulator.

No resolver code, corpus source or configuration changed. No CFML runtime tests,
new corpus reduction, or commit is claimed. Documentation whitespace checks pass.

## Runtime follow-up during implementation — 2026-10-10

The earlier no-runtime statement describes the research checkpoint. The copied
method fixture at testdata/resolution-runtime/copied-defaults has now run on
ColdFusion Server 2025,0,10,331899. Host-slot calls used Host for both default
expressions and bodies; extracted local references failed to find the helpers.
The fixture README records the existing image ID and isolated configuration;
observations-cf2025.json preserves the normalized result. This proves neither
other engines nor deployed Wheels initialization or plugin replacement.

Lucee's 7.1 ComponentImpl._call calls beforeCall to select the receiving variables
scope, then invokes UDFImpl. UDFImpl defines arguments (including defaults) before
the body. This is a source inference supporting receiving-slot evaluation:
[ComponentImpl source](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/ComponentImpl.java)
and [UDFImpl source](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/type/UDFImpl.java).
The source files were retrieved separately; a Lucee runtime fixture has not run.
Generic copied default resolution is still withheld until an explicit supported
engine and invocation policy separates slot calls from extracted references.
