# Remaining findings: research coverage and next plan

Latest follow-up: [the call-chain ledger](RESOLUTION-CALL-CHAIN-LEDGER.md) expands
the family inventory to 43.62%; the 35.86% audit below is the earlier checkpoint.
Neither percentage measures explained or fixable findings.

Review checkpoint: 2026-10-09. Research and planning only; no production changes.

## Answer to the coverage question

The earlier research did not cover a large proportion of the remaining findings.
Its four selected families contain 6,236 of 44,386 saved report entries (14.05%).
The Mura and ColdBox/ContentBox investigation below expands the selected families
to 15,916 entries (35.86%), with no overlap between the listed groups. This still
leaves 28,470 entries outside those families.

These percentages measure the families receiving research attention. They do
not mean every entry is explained, implementable or expected to disappear.
Only representative source chains have been followed; full per-entry causal
coverage has not been measured. No new reduction or runtime validation is claimed.

The denominator is `target/resolution/pattern-final`: all seven saved reports,
with their original roots/options. Entries retain multiplicity; counts are not
unique method names. Read-only evidence artifacts are
`target/relationship-research/coverage-inventory.json` and
`target/relationship-research/research-coverage.json`.

| Corpus | Remaining report entries | Share of total |
|---|---:|---:|
| Mura (report key masa) | 20,418 | 46.00% |
| Wheels | 11,527 | 25.97% |
| ContentBox | 6,829 | 15.39% |
| ColdBox | 3,080 | 6.94% |
| Lucee | 1,440 | 3.24% |
| TestBox | 599 | 1.35% |
| FW/1 | 493 | 1.11% |
| Total | 44,386 | 100% |

## Families selected and their actual research status

Receiver selection is case-insensitive. The migration selection uses the
`/migrations/` path segment; Mapper uses its missing-method reason. Family
membership is an inventory rule, not proof of a shared cause.

| Family | Entries | Files | What is understood | What is not yet established for all entries |
|---|---:|---:|---|---|
| Wheels g/application.wo | 2,958 | 193 | Factory, binding, stored object and local alias paths | Effective application ancestry and reachable writers |
| Wheels _controller | 1,594 | 51 | Factory/path-selection relationship investigated | Every controller's constructor and subsequent method/return chain |
| Wheels Mapper methods | 543 | 10 | Public method records and copying into host slots | Per-site return chains and competing overrides |
| ContentBox migrations | 1,141 | 52 | Manager invocation and callback argument sources in separately extracted dependencies | Active manager/configuration and dependency availability in the baseline |
| Mura $, variables.$, rc.$ | 3,917 | 268 | Alias registration, event construction and request/renderer propagation | Each consumer's entry path; dynamic dispatch and custom scope keys |
| Mura config/settings fields | 2,875 | 233 | Explicit config instance, service-name loop, admin setter wiring | Ownership/writers for every consumer and reload/factory state |
| ContentBox event/arguments.event | 1,390 | 170 | Handler arguments, renderer context and ordinary request-service return paths | Which path supplies each entry; interceptor and custom invocation variants |
| ColdBox event/arguments.event | 365 | 69 | Same framework-level suppliers | Version/context and actual callers per entry |
| ContentBox html/cb | 1,133 | 137 | Renderer event mutation, copied helper and template-scope propagation | Module activation, compatible dependency version, template reachability |

## Mura: relationships created in different places

Source paths here are relative to `/Users/garethedwards/corpus/MSU-NatSci_MuraCMS`.

### Config object and service-name loop

core/appcfc/onApplicationStart_include.cfm:188 constructs configBean, calls set
with the configuration data and stores it in application.configBean.
core/mura/configBean.cfc:240–337 supplies the struct-configuration path returning
this. This is a constructor-plus-method-return relationship, not a filename
match for a field named configBean.

Startup then creates mura.bean.beanFactory over /mura with recursion, explicit
exclusions and a transient-name pattern (:191–195). configBean is excluded from
discovery and explicitly registered as the already-created value (:204).
Therefore its discovery exclusion does not make that dependency unavailable.

At :468–477 a literal service list includes settingsManager. A loop retrieves
each selected bean and assigns application[name]. The settingsManager field is
created through correlated key/value iteration, not a literal assignment in the
source. The optional advertiserManager append is separately guarded.

Boundary: startup distinguishes variables.serviceFactory and
application.serviceFactory. The initial assignment to the application slot is
conditional (:197–199), and later initialization/reload code must be followed
before treating both expressions as the same factory at every point.

### Controller fields come from the supplied factory and framework lifecycle

admin/Application.cfc:146–166 includes startup when needed and installs
application.serviceFactory through setBeanFactory. The admin framework's
controller lookup (admin/framework.cfc:1373–1403) has separate branches:
retrieve a registered controller from a factory, or create its component,
invoke init and autowire it with the selected factory.

The private autowire routine at :1250–1267 enumerates set-prefixed members,
checks containsBean(property), calls getBean(property), and invokes the setter
with that named argument. This routine does not apply the singleton-only check
found in DI/1's own setter discovery. Do not transfer the modern DI/1 policy
wholesale to this older FW/1 controller path.

admin/core/controllers/controller.cfc:60–65 stores supplied configBean and
settingsManager arguments in variables. carch extends that controller and reads
variables.configBean in before() at :38. This is a worked candidate chain:

startup object/registration → installed factory → framework-created controller
→ inherited setter invocation → variables field → carch.before read.

It is not proof for a manually constructed carch or a controller returned by a
different factory branch. Those are required counterexamples. The upstream
factory's own injection and the framework's autowire invocation are distinct
ways of establishing the field.

### Dollar scope is an alias, a request value and sometimes a copied reference

Startup explicitly aliases $, m and mura to MuraScope at :339–341. Its transient
pattern includes MuraScope, so these names do not imply a shared singleton.
admin/assets/js/frontendtools.js.cfm:13 obtains MuraScope through the factory
and calls init(siteID), establishing one concrete local $ origin.

Another path starts in core/mura/event.cfc:50–69: init stores supplied data,
merges request values, takes an explicitly supplied $ or creates mura.MuraScope,
stores it under MuraScope, and links that scope back to the event. Admin setup
creates an event from request.context, then assigns its MuraScope to
request.context.$ and request.muraScope (admin/Application.cfc:383–385).
Relating a later rc.$ read to this value requires the framework's request-context
argument handoff; the name rc alone does not prove it.

core/mura/content/contentRenderer.cfc:594–605 establishes variables.$ from its
event and aliases variables.mura and variables.m to it. Views rendered or
included in that object's scope need that execution-context edge. Merely placing
a .cfm file in a view directory does not establish it.

MuraScope.init returns this but can install event state and append custom keys
(:50–67). Its event() method at :225 returns the event with no selector, a stored
value with a selector, or this after a write. siteConfig at :272 similarly has
selector-dependent results. These calls cannot all share one return component.

OnMissingMethod at :73 onward checks event members, content-renderer members,
custom scope keys and content-bean methods/accessors in that order, then may
throw. Identifying the receiver as MuraScope explains the dispatch entry point;
it does not prove a concrete target or an accepted method at every site.

### Version-specific factory behavior

Mura vendors its own core/mura/bean/ioc.cfc (declaring DI/1 1.1.1). Its duplicate
handling at :427–486 differs from the modern framework-one corpus inspected
earlier: there is no equivalent per-folder duplicate tracker, and the catch
constructs an error string without rethrowing it. Its getBean signature also
differs. These bodies need their own contract and runtime probes.

beanFactory.getInstance at :281 is a compatibility wrapper returning
getBean(name); its dsl and initArguments parameters are not forwarded. A method
named getInstance therefore does not imply WireBox's argument or DSL semantics.

## ColdBox/ContentBox: caller context supplies otherwise invisible values

Framework paths here are under `/Users/garethedwards/corpus/ColdBox_coldbox-platform`;
application paths are under `/Users/garethedwards/corpus/Ortus-Solutions_ContentBox`.

### Event has several origins

system/web/Controller.cfc:_runEvent:784 builds named event, rc and prc arguments
from the request context, resolves the handler/method, then invokes it through
argsMain (:1004–1010). Eligibility comes from that dispatch path, not an argument
spelling. Extra eventArguments and interceptor effects require their own
precedence review.

For views, system/web/Renderer.cfc:439–452 passes event/rc/prc and rendererVariables
to RendererEncapsulator.cfm. That template copies renderer variables, incorporates
viewVariables, assigns event/rc/prc explicitly, includes helpers and finally
includes the requested view. Thus a .cfm consumer can receive a component without
an assignment in its own source. Helper code and multiple rendering entry paths
are competing context/writer boundaries.

A third example, build/patches/3.7.0/Update.cfc:174–175, assigns event from
controller.getRequestService().getContext() before buildLink. It is not a handler
parameter or implicit template variable. Grouping every event finding under a
single framework-parameter fix would conflate different missing edges.

### html is installed through a renderer event

Renderer initialization announces afterRendererInit with its variables and this
scopes (Renderer.cfc:127). system/modules/HTMLHelper/ModuleConfig.cfc:47–48 handles
the event and assigns data.variables.html from getInstance(HTMLHelper@HTMLHelper).
The module also configures the legacy HTMLHelper@coldbox binding (:29).
The encapsulator's variable copy is then the bridge to a view's html.doctype or
html.startForm. This requires an activated module and a compatible renderer
lifecycle, not a universal variable-name rule.

### cb combines DI, mutation and a copied method

ContentBox's contentbox-ui ModuleConfig registers CBRequest as an interceptor.
Its CBRequest.cfc:14 injects CBHelper; configure at :18–25 obtains the renderer,
assigns renderer.cb, copies $cbinject onto it and invokes that copied method.
$cbinject at :153–155 sets variables.cb = this.cb. The renderer's variables then
flow through RendererEncapsulator into the view.

This chain combines injected property → receiver mutation → copied method's
runtime host → variables-scope mutation → include scope. It connects two themes
previously researched separately: DI bindings and copied-method ownership.
Runtime host binding is a required fixture, not something to assume merely
because the method body was found on the interceptor class.

ColdBox source is in a separate extracted corpus. Its inspected version is not
thereby proven to be ContentBox's deployed dependency. Cross-project source is
research evidence; adding it to a scan changes coverage and needs a separate
report. The HTMLHelper and CBRequest traces establish candidate mechanisms,
not automatic proof for every template in the saved ContentBox report.

## Review corrections and next plan

The earlier Wheels line-drift claim was incorrect. RESOLUTION-GAPS.md specifies
zero-based report lines; source/editor line 33 corresponds to report line 32.
The affected research documents have been corrected. Normalize coordinates
before source correlation rather than treating a one-line offset as drift.

The next research should be driven by the 28,470 entries outside these selections
and by incomplete proofs inside them. The largest residuals are Mura (13,626),
Wheels (6,432) and ContentBox (3,165). Mura's next receiver leads include bean
(466), rc.contentBean (453), arguments.event (412), arguments.bean (398),
arguments.contentBean (383) and application.rbFactory (315). Wheels has 3,126
remaining unscoped entries across categories; ContentBox has 924. Those broad
unscoped buckets must be split by method, owner and call path before assigning
them one cause.

Planned research sequence:

1. Build a per-entry evidence ledger for representative and counterexample
   sites in the Mura controller and request-scope groups. Prove the active
   factory/application and include/dispatch path before following method returns.
2. Trace generic bean/content/event values in the Mura residual; distinguish
   framework invocation, property-bag storage, factory lookup and collection
   iteration. Follow selectors through event()/siteConfig() and dynamic dispatch.
3. Split Wheels unscoped calls into inherited declarations, copied helpers,
   include-provided functions and genuinely unknown names. Reuse the existing
   method-copy research only where its source/host conditions hold.
4. Verify ContentBox helper activation and compatible dependency bodies, then
   trace one handler event and one rendered view through their actual suppliers.
5. Design small runtime counterexamples for the relationships whose outcome
   depends on CFML host binding, lifecycle order or mutable argument bags.

For each entry, record the first missing relationship, downstream relationships,
the evidence supporting each, and whether the last step is a real definition,
dynamic dispatch, an unavailable dependency or an actual missing method. Only
then can we report a defensible percentage explained. Implementation remains
outside this research request.

Review evidence: `target/relationship-research/cross-corpus-review.json` verifies
the totals and zero overlap between selected families. All 28 saved urlforSpec g
entries match the function name on their zero-based source line. The companion
`cross-corpus-sources.json` fingerprints 18 inspected framework/application files.
These checks validate the research record, not CFML runtime behavior. No runtime
test, production change or new corpus scan was performed; git diff --check passed.

## Follow-up call-chain review (2026-10-09)

[The call-chain ledger](RESOLUTION-CALL-CHAIN-LEDGER.md) follows nine representative
entries through copied-method hosts, selector-dependent returns, argument
identity, guarded assignments, dynamic bean accessors and event wrappers. Six
additional Mura receiver families and Wheels provider-body calls contain 3,447
disjoint entries. The research-family union is now 19,363 / 44,386 (43.62%), with
25,023 outside it. This still measures inventory scope, not explained coverage
or expected reductions. The ledger records counterexamples and remaining proof
obligations; implementation remains paused. Its reproducible read-only audit is
`target/relationship-research/call-chain-audit.py`.
