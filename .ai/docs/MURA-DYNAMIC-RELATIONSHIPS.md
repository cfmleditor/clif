# Mura dynamic relationship research — 2026-10-09

Research only; no resolver change or runtime validation. This extends the
[call-chain ledger](RESOLUTION-CALL-CHAIN-LEDGER.md), particularly M1/M2 and its
open question about synthesized-method registration. Sources are under
`/Users/garethedwards/corpus/MSU-NatSci_MuraCMS`; coordinates below are one-based
unless explicitly labelled as report lines.

## Main finding

A Mura bean's effective API is partly a mutable application registry, assembled
from metadata and potentially restored from disk. Loading entity A can register
methods for entity B. Resolving the class and reading B's own properties is
therefore insufficient. The explanation must retain registry ownership,
registration order, caller arguments and the actual dispatch body.

## Creation and lifetime of the registry

bean.init (:57–73) initializes instance state and calls getProperties.
beanORM.init (:49–55) calls super.init, resets its entity-name field, and asks for
properties again. getEntityName (:1079–1096) selects the last segment of explicit
entityName metadata, otherwise the component name with a trailing Bean removed
(except bean itself). This is a runtime metadata key, not necessarily a unique
source path or the original factory lookup string.

bean.getProperties (:588 onward) builds under an entity/application-specific
lock when properties are absent or rebuild=true. It starts from metadata for
the actual receiver and traverses its ancestors (:768 onward). A property is
installed only if that name is absent (:783), so the traversal's first property
wins. The loop proceeds while the current metadata has an extends entry; do not
silently replace this with an unrestricted flattened property union.

The builder resets properties and primarykey (:612–613), but uses defaults for
synthedFunctions, hasMany and hasOne (:606–609). A forced rebuild is consequently
not evidence that old generated methods or relationship arrays were cleared.
Rebuilding a changed declaration can retain earlier registry state. The extent
of that behavior needs a runtime fixture; the write/reset distinction is visible
in the source.

Startup (core/appcfc/onApplicationStart_include.cfm:389–427) initializes
application.objectMappings. Depending on conditions it applies database updates,
loads `/muraWRM/config/objectMappings.json.cfm` as JSON, or constructs a list of
beans including approvalRequest and approvalAction. These are different sources
of registry state. It is unsafe to assume the source metadata builder runs on
every startup or that restored data matches current declarations.

beanFactory.undeclareBean (:146–161) can delete an entity registry entry under
its dynamic-entity and authorization conditions. This is another invalidation
boundary, separate from editing the corresponding component file.

getSynthedFunctions (:1234–1245) ensures an entity name and defaults an empty
registry map; it does not itself call getProperties. An available map therefore
does not establish that all relationships have been registered.

## Rules the builder actually writes

A property with relatesTo gets that value assigned to cfc (:825). Relationship
processing requires cfc, defaults fkcolumn to primaryKey and loadkey to fkcolumn,
and changes loadkey to parentid for nested properties (:835–845).

| Declaration | Generated behavior on declaring entity | Boundary |
|---|---|---|
| one-to-many / many-to-many | getNameIterator and getName default to target loadBy with iterator; getNameQuery uses query; hasName reads recordcount; addName/removeName delegate to object helpers | Defaults can be overridden by caller arguments; target loadBy body still matters |
| singularname on a many relation | Iterator, Query, add, has and remove aliases | Bare getSingularName assignment is commented out; do not invent it |
| many-to-one / one-to-one | getName loads target with returnFormat="this"; setName updates foreign-key value | siteid foreign keys take a separate settingsManager.getSite path |
| relation to another entity | Inverse methods can be written into that other entity's registry | Availability depends on the registering entity being processed |

Evidence: bean.cfc:850–976. These entries hold expression strings and argument
records, not ordinary declared functions. The expressions must be traced through
the actual evaluator and callees to infer returns. For example hasName returns
recordcount, not necessarily a Boolean value; addObject/removeObject return this
(:177–207), after their own mutations.

Inverse registration has two materially different policies. For the listed core
entities content/user/feed/category/address/site/comment, writes use assignment.
For non-core targets, corresponding branches use param defaults, preserving
existing entries. Collisions and initialization order can therefore change which
relationship supplies a method. Some potential inverse has/set methods are
commented out. In the many-to-many branch the bare alias is keyed with prop.name,
while other inverse branches use the declaring entity name; follow the actual
keys instead of imposing a symmetric naming convention.

## Dispatch and caller precedence

bean.OnMissingMethod (:80–155) gets the registry and, when a matching entry
exists, uses its args.cfc to obtain a target bean. It can repair the entry's
loadKey using primary-key metadata, then calls synthArgs (:210–237). That helper
translates keys, reads the source value, can generate/store a UUID for an empty
value, and supplies returnFormat, optional siteid and ordering.

The merge at :113–120 fills only missing named arguments. An explicitly supplied
returnFormat wins over the generated default. Thus, conditional on reaching
this registered entry and the expected beanORM.loadBy:

- getActionsIterator() defaults to an iterator.
- getActionsIterator(returnFormat="query") selects a query instead.
- An unknown returnFormat keeps the return alternatives unresolved.

beanORM.loadBy (:886–891) is the body supporting these alternatives. No CFML
execution is claimed here. A different target's loadBy implementation requires
its own return proof. Registry loadKey writes also mean the entry is not simply
immutable metadata; alias/reference behavior needs engine-level validation.

The generated expression is evaluated at :125. If there is no registry match,
get/set prefixes fall back to getValue/setValue; other names throw. getValue
(:417–430) can invoke an existing getter, read stored state, install a supplied
default, or return an empty string. setValue (:385–406) can invoke a setter or
store a value, then returns this. Consequently a callable get-prefixed name does
not prove either a declared property or a component-valued result.

Ordinary explicit methods must remain distinct from missing-method dispatch.
The worked example below has both in consecutive links.

## Worked unresolved chain: approval view

Saved report: `target/resolution/pattern-final/masa.json`.

- admin/core/views/carch/statusmodal.cfm report line 51 / source 52:
  content.getApprovalRequest().
- Same file report line 53 / source 54:
  approvalRequest.getActionsIterator().

At source 45 the template obtains content through
`$.getBean('content').loadBy(contenthistid=rc.contenthistid)`. The caller-supplied
scope/factory remains an upstream obligation; the contentHistID return path is
traced in the earlier ledger.

contentBean explicitly declares getApprovalRequest at :1391–1393. It gets the
approvalRequest bean and calls loadBy with contenthistid/chainID/siteID, omitting
returnFormat. Startup aliases approvalRequest → approvalRequestBean and
approvalAction → approvalActionBean (:343–344). approvalRequestBean extends
beanORM, whose default loadBy returns the receiver on the non-query/non-iterator
branch. This first method is **not evidence of inverse synthesized dispatch**,
even though a similarly named registry entry can be created elsewhere.

approvalRequestBean's actions property (:11) declares one-to-many,
singularname=action, cfc=approvalAction and orderby="created asc". When its
metadata has populated the active registry, the builder supplies
getActionsIterator. Missing-method dispatch creates the approvalAction bean,
merges generated arguments and executes loadBy. With the no-argument call shown,
its default is iterator. beanORM constructs beanIterator, sets its entityName to
the target entity and sets its query. beanIterator.setEntityName (:55–60)
returns this; queryIterator.setQuery (:282 onward) is the inherited next step.

The iterator's element type is a separate relationship. beanIterator.packageRecord
(:67 onward) uses its stored entityName and record representation: query rows
create/initialize a bean, array objects pass through unchanged, and non-object
array rows are used to populate a bean. Do not globally type every iterator
result as approvalAction from the iterator class alone.

Outstanding proof conditions: the template's actual scope object and factory,
constructor execution, active registry snapshot, competing registrations,
target overrides, queryIterator's complete setter/iteration chain, and any
custom invocation. This is a conditional source trace, not a newly resolved site.

## Counterexamples and review notes

1. A generated getter's explicit returnFormat defeats suffix-based typing.
2. getAction is not registered by the actions singular-name branch; it can still
   reach the generic getter fallback, with a different result contract.
3. Metadata on a different entity can supply an inverse method; only reading
   the receiver's source misses that dependency.
4. A disk-restored registry or a retained entry after rebuild can differ from
   freshly generated metadata. Absence in current declarations is not enough.
5. getApprovalRequest in the worked chain is explicit; choosing a generated
   relationship solely from its name would select the wrong origin.
6. The site relation's set expression at bean.cfc:928 contains an apparent extra
   closing parenthesis. Record it as a suspected source defect requiring a
   runtime reproduction; do not silently normalize it into a working rule.

The report contains 12,611 get-prefixed and 1,566 set-prefixed findings out of
20,418 Mura entries. Those are spelling counts only. They include unrelated
objects, ordinary methods and unknown origins; they are **not** dynamic-bean
coverage and must not be added to the existing 43.62% family inventory.

## Next research and acceptance criteria

- Trace the worked template's actual scope injection and factory, then close the
  iterator setter/next/packageRecord path with source evidence for each return.
- Inventory registrations by declaring entity, target registry, installed name,
  assignment/default policy, expression and default arguments. Keep provenance
  for reverse registrations; do not reduce it to a set of method names.
- Identify saved-registry loading, rebuild and deletion dependencies before
  planning cache invalidation. Separate source edits from application lifecycle.
- Design paired runtime fixtures: fresh/restored registry; rebuild after removing
  a relationship; reversed registration order; explicit returnFormat; explicit
  method versus registry entry; singular alias absent; inverse method before and
  after other-entity initialization. Fixtures remain planned, not implemented.
- Attribute actual findings only after origin and dispatch are proved. Keep
  conditional traces, unsupported dynamic state and suspected source defects
  separate from proven method resolutions.

Review artifacts: `target/relationship-research/mura-dynamic-review.json` records
source fingerprints, sample line correlations and lexical counts. The baseline
and production code remain unchanged. Full causal coverage remains unmeasured.
