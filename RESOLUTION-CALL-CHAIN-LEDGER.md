# Call-chain research ledger — 2026-10-09

Research and planning only. No resolver implementation is authorized by this
ledger. It extends [the coverage audit](RESOLUTION-RESEARCH-COVERAGE.md) with
representative chains and counterexamples. Source evidence establishes what
these checked-out bodies do; it does not establish every reachable application
configuration or CFML runtime behavior.

## Measurement and coordinates

The baseline remains all seven `target/resolution/pattern-final` reports:
44,386 entries, preserving multiplicity. Report lines below are zero-based;
source lines are one-based. `target/relationship-research/call-chain-audit.py`
reconstructs the earlier selections, asserts their 15,916 count, checks new
selection disjointness, and correlates nine representative report sites with
source text. Its JSON output records the exact entries.

| Additional inventory selection | Entries | Files |
|---|---:|---:|
| Mura `bean` | 466 | 29 |
| Mura `rc.contentBean`, case-insensitive | 457 | 33 |
| Mura `arguments.event` | 412 | 8 |
| Mura `arguments.bean` | 398 | 6 |
| Mura `arguments.contentBean` | 383 | 7 |
| Mura `application.rbFactory` | 315 | 43 |
| Wheels unscoped calls in vendor/wheels/controller, model and view | 1,016 | 45 |
| Additional selected entries | 3,447 | — |

The union of earlier and additional selections contains 19,363 entries (43.62%);
25,023 remain outside it. This is **family footprint, not explained coverage,
expected fixes, or causal classification of every member**. The earlier residual
list used 453 for exact-case rc.contentBean; case-insensitive selection adds four.
Within the Wheels selection, 994 entries say `no qualifier, not in file`; 22
have downstream chain/return reasons. They cannot all be assigned one cause.

## W1: a copied method needs the receiving object's context

Representative entries, relative to `/Users/garethedwards/corpus/cfwheels_cfwheels`:

- vendor/wheels/controller/authorization.cfc report 34 / source 35:
  authorize calls `$get`.
- vendor/wheels/controller/caching.cfc report 13 / source 14:
  caches calls `$args`.

These source components do not extend Global. Their public methods are selected
by Global.$buildComponentIntegrationPlan (Global.cfc:1163 onward), stored as
`{name, ref}` records, and consumed by Controller.$integrateComponents (:377)
and $integrateFunctions (:408). Controller.init (:3) integrates both
wheels.controller and wheels.view, and Controller extends wheels.Global.
Global declares $get (:707) and $args (:3024). Model.init similarly integrates
wheels.model onto a Global-derived object.

Required relationships, in order:

1. The actual instance is constructed through the relevant init path.
2. The active mapping resolves the integration directory and its files.
3. Metadata selects the public method and retains its function reference.
4. Assignment installs that reference in the host's variables/this slots.
5. Invocation uses the host context; inherited helper lookup reaches Global.

The source comment explicitly describes late binding of shared references, but
this remains a runtime fixture obligation for each supported engine. The same
source method invoked directly on its standalone provider is a different case.
A name such as `$args` alone is not evidence: MockBox also declares that name.

Copy policy matters: an existing host name is preserved, the framework reference
is exposed under `super<name>`, and a planned mixin override also creates that
alias. Later onDIcomplete prepares the object and initializes plugin mixins.
Do not flatten these writes into an unordered set of possible methods. Preserve
source function identity separately from host, installed name, and lifecycle.
This explains both directions of a gap: consumers cannot find copied methods,
and copied method bodies cannot find helpers inherited by their host.

Open: per-entry access eligibility, init overrides, actual directory mappings,
competing writes and downstream return values. The 1,016-entry inventory does
not prove those conditions for every site.

## M1: settings read preserves a supplied object or creates one

Relative Mura paths below use `/Users/garethedwards/corpus/MSU-NatSci_MuraCMS`.
Representative: admin/core/controllers/csettings.cfc report 139 / source 140,
bean.getIsNew in updateSite. Source 139 assigns
`bean=variables.settingsManager.read(siteid=rc.siteid)`.

settingsManager.init (:46–60) stores its supplied settingsDAO in variables.DAO.
read (:212–215) forwards siteid and settingsBean, whose default is the empty
string. settingsDAO.read (:65–85) starts with that argument, substitutes
getBean("site") only when it is not an object, optionally mutates it with query
results, and returns bean. Startup registers site → settingsBean at
onApplicationStart_include.cfm:308.

For the representative call, omission of settingsBean selects the factory
branch. For a caller passing an object, the result retains that object's
identity. A universal `read -> settingsBean` summary would erase that distinction.
The manager's constructor supplier and active factory still need to be proved
for a complete site-specific chain; this ledger does not infer them from names.

There is another obligation after origin: settingsBean extends beanExtendable,
which extends bean. bean.cfc declares property isNew (:48), but its
OnMissingMethod (:80–155) supplies a get/set fallback through getValue/setValue.
Registered synthesized methods take precedence. The endpoint is therefore a
dynamic dispatch contract, not an invented explicit getIsNew declaration.

## M2: loadBy is not a universal fluent operation

beanORM.loadBy (:718) has returnFormat="self". After querying/mutating state,
:886–891 returns a query for `query`, a configured beanIterator for `iterator`,
and this for the remaining branch. Preserve selectors, defaults and receiver
identity; an unknown selector cannot justify a single bean return.

contentBean.loadBy (:1199) is a different body: it defaults siteID, adds
`contentBean=this` to arguments and forwards them to contentManager.read.
That read (:327–358) chooses a path by contentHistID, filename, remoteID, title,
urltitle, then contentID. The contentHistID path reaches getContentVersion
(:360–404), including cache hit, cache miss and catch paths, then
contentDAO.readVersion (:70–152) where needed. readVersion preserves a supplied
object, creates getBean("content") otherwise, mutates it, and returns it.
Startup's content → contentBean alias is at :303.

Thus the version path supports an identity-preserving return argument chain,
subject to active suppliers and mappings. It is not sufficient to observe a
`return this` in an unrelated loadBy implementation. Other read selectors
remain separate proof obligations. A useful review correction: readActiveByRemoteID
builds beanArray for multiple results, but the inspected body returns bean
(:315); a temporary array or manager `isArray` check is not proof of an array
return. Follow actual return expressions.

## M3: origin evidence does not establish assignment on every path

Representative: cchain.cfc report 78 / source 79, bean.hasErrors in save.
At :76 bean is assigned through getBean('approvalChain').loadBy(...).set(...).save()
only inside the successful CSRF branch. The subsequent read is outside it.
A successful-chain explanation must retain the unassigned/unknown incoming
state on the other branch. This is a counterexample to treating any earlier
assignment as a dominating definition, not a claim that runtime always fails.

Representative: carch.cfc report 397 / source 398, rc.contentBean.getParentID.
In update (:331 onward), three selector branches under allowAction/action='add'
assign arguments.rc.contentBean (:372,374,376); a successful CSRF branch then
replaces it with save() (:379). The later read has a related action guard.
This needs arguments.rc ↔ rc identity, branch joins, guard correlation and the
save return relation. A shared spelling is not sufficient. These two controller
examples should not receive the same unconditional provenance rule.

Representative: approvalRequestBean.cfc report 161 / source 162,
arguments.contentBean.getSiteID in sendActionMessage. Local call sites pass a
content value built through getBean('content').loadBy(...), e.g. approve at :83
and :102, but construction is nested inside lifecycle/status branches. Review
all incoming calls and their path conditions before attaching one parameter
type to the entire function. This parameter is not a DI injection point.

## M4: an argument can be an object or a factory selector

Representative: core/mura/client/api/json/v1/apiUtility.cfc report 1679 /
source 1680, arguments.bean.getEntityName in AllowAccess.

AllowAccess (:1672 onward) branches on isObject(bean). The object branch checks
available members and queries entityName; it does not prove a concrete class.
The other branch checks containsBean(bean), rejects an unknown ID, or replaces
arguments.bean with arguments.$.getBean(entityName). Its result depends on the
selector and on the factory behind the supplied scope object. Subsequent
allowAccess dispatch is also guarded by member existence.

Plan separate input refinements: object identity, scalar bean ID, rejection
path, factory result. Do not turn the parameter name bean into a global type,
and do not treat a capability check as proof of a particular component.

## M5: event normalization occurs at the invocation boundary

Representative: core/mura/Handler/standardEventsHandler.cfc report 68 /
source 69, arguments.event.getValue in standardWrongDomainHandler.

The standard path is distinct from general plugin announceEvent. The domain
validator (:674–680) calls event.getHandler("standardWrongDomain").handle(event).
event.getHandler (:114–115), or servletEvent's corresponding method (:156–157),
appends `Handler` and asks its stored HandlerFactory for the key. The factory
origin is set through pluginManager.getStandardEventFactory (event:185 or
servletEvent:81; pluginManager:2867 onward).

pluginStandardEventFactory.get checks local, site and global listeners, plugin
records, cached entries and the standard handler/translator fallback. The
cache and its check key affect later selection; "standard" is not unconditional
ownership. The selected handler and key are wrapped together.

pluginStandardEventWrapper.splitContexts (:60) separates a MuraScope into its
event() result and scope, or retains another context as event and reads its
muraScope property. handle (:72) builds named arguments event/mura/m/$ and
invokes the selected method, with a handle fallback. General pluginManager
executeScript (:1376 onward) separately normalizes structs, request context,
new event instances and MuraScope before cfinvoke.

The rule belongs to a proved dispatch path and its input, not every event-named
parameter. Direct/custom invocations remain possible; the wrapper's else branch
accepts the supplied context without proving it is a concrete mura.event.
For this report site, selecting the default standard handler and tracking the
actual incoming event remain explicit conditions.

## M6: direct application fields still have lifecycle boundaries

Representative: admin/assets/js/frontendtools.js.cfm report 15 / source 16,
application.rbFactory.resetSessionLocale. Startup :492 assigns
new mura.resourceBundle.resourceBundleFactory(), which declares
resetSessionLocale (:670). However, :495–497 can include
/muraWRM/config/settings.custom.managers.cfm after that assignment.

The straight-line origin is strong evidence. A complete proof also needs startup
reachability, the effective application owner and possible subsequent writers,
including that optional include. The same rule applies to the other 314 family
entries only after checking their execution context.

## Reviewed research plan

The central relationship is a conditional transfer of identity or behavior:
creation/lookup → storage or argument → branch/selector → host or return → call.
Keep each link and its uncertainty; recognizing one link must not silently
prove the rest.

Next research, before implementation:

1. Split the 994 Wheels no-qualifier sites by actual declaration access and
   host/init path; separate direct inherited helpers from downstream returns.
2. Finish settingsDAO/contentDAO construction and factory ownership for the
   worked Mura examples, then enumerate remaining return selectors and writers.
3. Trace synthesized-function registration (including objectMappings mutation)
   before deciding how to represent dynamic getter/relation endpoints.
4. Audit a stratified sample from each new family, including arbitrary public
   callers, supplied-object returns and unsuccessful construction branches.
   Record explained / conditional / blocked / possible-real-defect separately.
5. Specify runtime fixtures for method copying, collision/super aliases,
   supplied-object identity, selector returns and event wrapper invocation.
   Exercise direct-call and alternate-host counterexamples on supported engines.
6. Only after per-entry attribution, calculate causal coverage. The current
   43.62% inventory is below a majority and is not a substitute for that audit.
   The 25,023 unselected entries must continue to drive sampling priorities.

Review: nine saved sites match their source lines, selection counts reproduce
the earlier audit and additions do not overlap it. Source hashes are recorded
in `target/relationship-research/call-chain-sources.json`. No runtime fixtures,
resolver changes or new corpus scan were performed for this ledger.
