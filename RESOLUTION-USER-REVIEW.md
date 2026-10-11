# Resolver review report

> **Current update (checkpoint 30):** The two concrete findings are implemented.
> Lucee's source-layout `/admin` assumption removes 99 findings by default:
> **10,235 → 10,136**. Chain calls retain exact spans; 292 diagnostics move to
> correct hop lines, and two duplicate extracted calls are removed. The original
> changeset omission blocker is fixed, but later callers still block that cohort.
> Assumptions, overrides and validation are documented in checkpoint 30 of
> [RESOLUTION-IMPLEMENTATION-STATUS.md](RESOLUTION-IMPLEMENTATION-STATUS.md).

> **Further causal research:** An isolated `/admin` configuration scenario
> removes 99 Lucee findings; it is not a default change. Mura traces distinguish
> a supplied bean/array return ambiguity from a missing byte span on a chained
> call. The latter affects a 65-entry cohort but has no verified gain yet.
> Evidence and priorities are in the follow-up section of
> [RESOLUTION-REMAINING-CAUSES.md](RESOLUTION-REMAINING-CAUSES.md).

> **Planning review after checkpoint 29:** Current causal inventory and the
> coverage caveat are in [RESOLUTION-REMAINING-CAUSES.md](RESOLUTION-REMAINING-CAUSES.md).
> 7,703 of 10,235 entries have unknown receivers; 968 lack component returns.
> The 46 base summaries also represent 1,132 unchecked calls. Historical counts
> and examples below are not a current root-cause inventory. Further production
> changes are deferred while the large families are attributed to source paths.

> **Current update (checkpoint 29):** 136 more calls resolve: Wheels removes 49
> findings and Masa removes 87. The total falls **10,371 → 10,235**, with zero
> added diagnostics and unchanged file/filtered-check coverage. Unknown calls
> proven to omit an argument no longer block its inferred type. Runtime component
> names remain unknown. Literal ORM collection and native `each` support was
> also added, but supplies no measured gains in these corpora.

> **Previous update (checkpoint 28):** Caller limits are optional via
> `callerInferenceLimits` in `.clif.json`; the default remains unlimited.
> Default corpus diagnostics and coverage are unchanged at 10,371 findings.


> **Current update (checkpoint 27):** Caller file and call-site limits are removed,
> including constructors. The seven-corpus result stays at 10,371 findings with
> zero diagnostic changes. Unknown receivers and values still withhold inference.


> **Current update (checkpoint 26):** 10,371 findings across the same 8,151 files.
> Caller budgets now count files belonging to the target method, removing 369
> Masa DAO findings with no additions. Unknown receivers and bounded searches
> still withhold inference. See PROJECT-ASSUMPTIONS.md for limitations.


> **Current update (checkpoint 25):** 10,740 findings across the same 8,151 files.
> Source-shaped migration and CommandBox defaults remove another 1,830 net findings
> (14.6% this turn). The 20-pass review is in RESOLUTION-ITERATION-REVIEW.md;
> assumptions, global naming limits and overrides are in PROJECT-ASSUMPTIONS.md.

> **Checkpoint 23 update:** 12,570 findings across the same 8,151 files;
> Masa 4,179. Discovered factory ownership now reaches setter fields, removing
> 479 more net findings. Sixteen diagnostics become more precise return-type gaps
> at already-reported calls. See PROJECT-ASSUMPTIONS.md for policy and limits.

> **Checkpoint 22 update:** 13,049 findings across the same 8,151 files;
> Masa 4,658. Admin payload and startup service defaults remove another 574 net
> findings. The one new diagnostic is an undeclared readByEmail method in the
> legacy SOAP API. See PROJECT-ASSUMPTIONS.md for assumptions and overrides.

> **Checkpoint 20 update:** 13,623 findings across the same 8,151 files;
> Masa 5,232. Conditional factory targets with a source-verified shared base remove
> another 86 findings, with zero additions. See PROJECT-ASSUMPTIONS.md for limits.

> **Checkpoint 19 update:** 13,709 findings across the same 8,151 files;
> Masa 5,318. The latest four documented assumption batches remove 10,851 net
> findings. This includes runtime conventions and existing preset behavior.
> See PROJECT-ASSUMPTIONS.md for assumptions and overrides; inventories below
> remain historical snapshots.

> **Checkpoint 15 update:** current total 24,560; Masa 16,169. The existing Mura
> preset now activates from its recognized source layout, exposing 795 deeper
> diagnostics while removing 4,328. See PROJECT-ASSUMPTIONS.md for current policy
> and overrides. Checkpoint 14 counts and the original review examples below
> remain historical.

> **Current default update (checkpoint 14):** 28,093 findings over the same
> 8,151 files, down from 41,490 at checkpoint 12/13. Automatic project assumptions
> account for much of the improvement; they can be overridden through config.
> See [PROJECT-ASSUMPTIONS.md](PROJECT-ASSUMPTIONS.md). There are 14,417 removed
> diagnostics and 1,020 added, mostly deeper return/DI gaps. The examples and
> inventory below remain historical review material, not a current total.

**Implementation update:** The three identified fixes below are implemented and audited in [checkpoint 12](/Users/garethedwards/development/github/clif/RESOLUTION-IMPLEMENTATION-STATUS.md). All seven targeted entries disappeared, with zero new findings: four source-backed resolutions and three phantom-call extraction corrections. Outstanding entries are now **41,490**, with **174,833** checked calls. The original review inventory and example IDs below retain the checkpoint 11 snapshot for your notes. E24 remains an intentional negative test; its finding is retained.

**Wheels follow-up:** The immediate Injector startup result now resolves `application.wo` and local `g` aliases where application ancestry is available. Default totals remain **41,490**, unchanged per entry. A separate probe with an explicit `/ -> public` mapping removes **5,434** entries and adds **11** (8,861 -> 3,438); this is a conditional deployment-context result, not an automatic root inference. See checkpoint 13 for coverage differences and the remaining root declaration requirement.

Snapshot: 10 October 2026, implementation checkpoint 11. Generated from the existing seven-corpus reports in `target/resolution/relationship-guards/`. Source excerpts are read from the extracted corpora on this computer. The workspace contains uncommitted implementation and research changes; HEAD is `45849c4`, which alone does not reproduce this snapshot.

## What your review can help establish

The resolver has **41,497 outstanding report entries** across **8,151 files** and **174,836 checked calls**. An outstanding entry does not necessarily mean broken application code: it may be missing type propagation, runtime method installation, absent deployment configuration, an optional dependency, intentionally dynamic dispatch, or a real invalid call.

The most useful contribution is a source path connecting a receiver to its constructor, registration or invocation. A variable name or a matching method in another file is a lead, but does not establish that relationship. One counterexample—an override, replacement, alternate return or different caller—can prevent an overly broad rule.

No new implementation or corpus scan was performed to create this report. Existing measured results are retained. No preset/configuration override was supplied for the audited scans; these extracted roots have no root `.clif.json`/`.cfmleditor.json`. Real deployment mappings, installed dependencies and runner setup are especially useful evidence. A configured scan would be a separate comparison, not a substitute for this baseline.

| Corpus | Outstanding | Checked calls | Files |
| --- | ---: | ---: | ---: |
| Wheels | 8,864 | 63,067 | 1,710 |
| Mura | 20,264 | 29,909 | 839 |
| ContentBox | 6,777 | 16,815 | 724 |
| ColdBox | 3,081 | 19,357 | 652 |
| Lucee | 1,420 | 39,050 | 3,776 |
| TestBox | 599 | 4,027 | 145 |
| FW/1 | 492 | 2,611 | 305 |
| **Total** | **41,497** | **174,836** | **8,151** |

Checkpoint 11 added 40 verified resolutions. Checkpoints 10–11 together added 179 verified resolutions against 41,707; their net report reduction of 210 also includes previously documented reporting changes and new findings. The requested 30% reduction from 41,707 would mean 29,194 or fewer outstanding entries. From this snapshot that requires another 12,303 net reductions, with coverage preserved and resolutions attributed separately.

## Identified fixes versus investigation leads

The report contains some concrete fixes, not just broad opportunity groups.
These three fixes are now implemented. Positive regressions failed before the
fixes, counterexamples pass, and the unchanged seven-corpus audit verifies their
outcomes. The table retains the original matching-entry counts.

| Case | Assessment | Implementation / next step | Snapshot matching entries |
| --- | --- | --- | ---: |
| E23: Lucee `elseif` | Fixed extraction defect: branch openers no longer become calls | Regression verifies branch exclusion, condition calls and qualified method names | 3 |
| TestBox `isAwesome` / `isNotAwesome` | Source-backed runtime registration gap: AssertionsTest.beforeTests installs both closures via addAssertions; BaseSpec.addAssertions appends them to this.$assert | Implemented bounded literal registration for the selected setup hook, with override/write counterexamples | 2 |
| Mura session.mura.editBean checks | Bounded type-guard gap: two templates check a literal component before calling its method in a later AND term | Implemented conjunction discovery and scoped-path stability; replacement/escape counterexamples pass | 2 |
| E24: invalidFunction inside throws | Intentional negative test, not a missing receiver relationship | Consider an expected-failure annotation/report classification; preserve the finding until that policy is explicit | 1 |

The first three have source evidence, failing-before regressions and measured
fixes. Their seven snapshot entries are gone; the first three are extraction
corrections rather than newly resolved calls.
The Mura examples are admin/core/views/carch/loadextendedattributes.cfm and
admin/core/views/cusers/loadextendedattributes.cfm. The TestBox registration is
[AssertionsTest.beforeTests](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/tests/specs/AssertionsTest.cfc:7),
implemented by [BaseSpec.addAssertions](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/system/BaseSpec.cfc:973).
The parser omission is in [isKeyword](/Users/garethedwards/development/github/clif/internal/parser/script_parser.go:5283).

E25/E26 are configuration/runtime-host leads with a likely remedy, but their
live entry mappings still need confirmation. R01–R16 are larger relationship
inventories: they have researched directions, not proof that every entry can be
resolved by one rule. The largest gains require those ownership and lifecycle
contracts to be completed.

## Review priorities

Counts below are exact **receiver-name inventories**, not forecasts. Each entry belongs to at most one listed receiver group in its corpus, but a single implementation may address several groups and may resolve only a subset. The first six contain **13,831 entries (33.3% of the outstanding report)**. They are a sensible starting point for the requested larger gains.

| ID | Corpus / relationship | Entries | Files | Frequent reported methods |
| --- | --- | ---: | ---: | --- |
| [R01](#r01) | Wheels: Wheels application factory and controller origins | 4,552 | 193 | `model` (1204), `controller` (367), `findOne` (238) |
| [R02](#r02) | Mura: Mura scope object origins | 3,916 | 268 | `event` (665), `rbKey` (553), `siteConfig` (374) |
| [R03](#r03) | Mura: Mura configBean ownership | 1,598 | 123 | `getReadOnlyQRYAttrs` (251), `getContext` (89), `getValue` (83) |
| [R04](#r04) | Mura: Mura settingsManager ownership | 1,264 | 159 | `getSite` (645), `getPrivateUserPoolID` (134), `getPublicUserPoolID` (46) |
| [R05](#r05) | ContentBox: ContentBox event entry contracts | 1,368 | 166 | `getResponse` (322), `paramValue` (264), `buildLink` (255) |
| [R06](#r06) | ContentBox: ContentBox helper installation | 1,133 | 137 | `hiddenField` (126), `r` (100), `label` (98) |
| [R07](#r07) | ContentBox: Migration callback parameter identity | 884 | 47 | `string` (223), `nullable` (114), `default` (78) |
| [R08](#r08) | Wheels: Wheels model instances and selected returns | 721 | 32 | `valid` (103), `hasErrors` (42), `validatesLengthOf` (40) |
| [R09](#r09) | ContentBox: ContentBox request collection handoff | 351 | 95 | `hasPermission` (185), `getsiteID` (27), `getAuthorID` (21) |
| [R10](#r10) | ColdBox: ColdBox event invocation | 361 | 69 | `setView` (51), `renderData` (24), `getCollection` (23) |
| [R11](#r11) | ColdBox: ColdBox controller storage and injection | 412 | 39 | `getSetting` (76), `getInterceptorService` (28), `getRequestService` (22) |
| [R12](#r12) | Lucee: Lucee driver and Field/Group collections | 198 | 16 | `getRequired` (40), `getDisplayName` (33), `getDescription` (26) |
| [R13](#r13) | TestBox: TestBox assertion method installation | 123 | 1 | `isFalse` (2), `isTrue` (2), `isNotEqual` (2) |
| [R14](#r14) | FW/1: FW/1 stored user and department collections | 163 | 13 | `getId` (33), `setId` (24), `setName` (12) |
| [R15](#r15) | ContentBox: CommandBox task print object | 204 | 4 | `greenLine` (47), `toConsole` (46), `line` (39) |
| [R16](#r16) | Wheels: Wheels engine adapter selection | 148 | 24 | `$quoteIdentifier` (28), `$querySetup` (8), `isBoxLang` (6) |

These 16 groups contain **17,396 distinct entries (41.9%)**. The remaining entries are not individually covered by these review cards. Counts include current diagnoses; some named receivers are already typed but fail at a later relationship.

## How to reply

Start with any group you know well; reviewing all of them is unnecessary. Copy this template into your reply or add notes below a card:

```text
ID / example: R01 / E01
Assessment: inference gap / configuration gap / missing dependency /
            intentionally dynamic / invalid call / uncertain
Expected receiver or definition:
Evidence: source file and function/line, or actual runner/configuration
How the object reaches this call:
Counterexample or exception:
```

An assessment is provisional until the source relationship is checked. “This runs successfully” is useful runtime evidence, but please include the engine/framework version and entry path when possible. “Not used anymore” helps classify an archived example; it does not justify silently removing it from the unchanged corpus audit.

## Representative calls

Report lines are zero-based; all lines and links below are **one-based**. Excerpts preserve the source text around the reported line. Each example is a starting point, not a validated interpretation of every call in its group.

<a id="r01"></a>

### R01 — Wheels application factory and controller origins

The documented path constructs wheels.Injector("wheels.Bindings"), selects global, then creates controller/model instances. Mutable binding state and test setup order still block the complete proof.

**Your input:** Which Application/proxy and test runner actually own these calls? Point to the selected binder, alias registration and beforeAll/beforeEach chain. Can tests replace application.wo or mappings after startup?

Research: [RESOLUTION-RELATIONSHIPS.md](/Users/garethedwards/development/github/clif/RESOLUTION-RELATIONSHIPS.md).

- [ ] **E01** — `g.findOne` at [cfwheels_cfwheels/vendor/wheels/tests/specs/controller/AuthorizationSpec.cfc:14](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/tests/specs/controller/AuthorizationSpec.cfc:14)

  Current diagnosis: `variable 'g' has no component ref`.

```cfml
   12  				application.wheels.policyPath = "/wheels/tests/_assets/policies"
   13
   14  				author = g.model("author").findOne(where = "firstName = 'Per'", order = "id")
   15  				otherAuthor = g.model("author").findOne(where = "firstName = 'Tony'", order = "id")
   16  				post = g.model("post").findOne(where = "authorid = #author.id#", order = "id")
```

- [ ] **E02** — `application.wo.model` at [cfwheels_cfwheels/cli/src/commands/wheels/generate/admin.cfc:52](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/generate/admin.cfc:52)

  Current diagnosis: `variable 'application.wo' has no component ref`.

```cfml
   50  		var metadata = {};
   51  		try {
   52  			var modelInstance = application.wo.model(modelName);
   53  			metadata = adminIntrospectionService.introspect(modelInstance);
   54  		} catch (any e) {
```

- [ ] **E03** — `_controller.can` at [cfwheels_cfwheels/vendor/wheels/tests/specs/controller/AuthorizationSpec.cfc:55](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/tests/specs/controller/AuthorizationSpec.cfc:55)

  Current diagnosis: `variable '_controller' has no component ref`.

```cfml
   53  					request.$policyTestUser = {id = author.id}
   54
   55  					expect(_controller.can("index", author)).toBeFalse()
   56  					expect(_controller.can("show", author)).toBeFalse()
   57  					expect(_controller.can("update", author)).toBeFalse()
```

Review notes:

> Assessment / evidence / exception:

<a id="r02"></a>

### R02 — Mura scope object origins

These names are widely used for Mura scope objects, but the incoming object and active factory must be established at each entry point. A name alone cannot identify its type.

**Your input:** Where is each scope object constructed and handed to the controller, view or plugin? Identify the injection/request assignment and any replacements. Are all three aliases always the same kind of object?

Research: [MURA-DYNAMIC-RELATIONSHIPS.md](/Users/garethedwards/development/github/clif/MURA-DYNAMIC-RELATIONSHIPS.md).

- [ ] **E04** — `$.getContentRenderer` at [MSU-NatSci_MuraCMS/admin/assets/js/frontendtools.js.cfm:22](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/admin/assets/js/frontendtools.js.cfm:22)

  Current diagnosis: `variable '$' has no component ref`.

```cfml
   20  	<cfoutput>(function(window){
   21  		window.Mura=window.Mura || window.mura || {};
   22  		window.Mura.layoutmanager=#$.getContentRenderer().useLayoutManager()#;
   23
   24  		var utility=Mura;
```

- [ ] **E05** — `variables.$.getAdminPath` at [MSU-NatSci_MuraCMS/admin/core/utilities/modal/layoutmanager.cfm:158](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/admin/core/utilities/modal/layoutmanager.cfm:158)

  Current diagnosis: `variable 'variables.$' has no component ref`.

```cfml
  156  			Mura.deInitLayoutManager();
  157  	}
  158  	Mura.loader().load('#variables.$.siteConfig().getAdminPath(complete=completeurls)#/assets/js/layoutmanager.js',
  159  		function(){
  160  			<cfif $.content('type') eq 'Variation'>
```

- [ ] **E06** — `rc.$.event` at [MSU-NatSci_MuraCMS/admin/common/layouts/compact.cfm:125](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/admin/common/layouts/compact.cfm:125)

  Current diagnosis: `variable 'rc.$' has no component ref`.

```cfml
  123  		<script type="text/javascript" src="#application.configBean.getContext()#/core/vendor/ckeditor/adapters/jquery.js"></script>
  124
  125  		<cfif rc.$.event('contenttype') neq 'Variation' and not len(rc.$.event('remoteurl')) and not len(rc.$.event('preloadOnly'))>
  126  			<script>
  127  				try{
```

Review notes:

> Assessment / evidence / exception:

<a id="r03"></a>

### R03 — Mura configBean ownership

Application startup, DI arguments and component fields are different origins. The resolver cannot safely merge managers from unrelated applications.

**Your input:** Show the live configBean registration and the constructor/setter that fills these fields. Are there alternate implementations or site-specific overrides?

Research: [DI-RULES-RESEARCH.md](/Users/garethedwards/development/github/clif/DI-RULES-RESEARCH.md).

- [ ] **E07** — `variables.configBean.getDatasource` at [MSU-NatSci_MuraCMS/admin/core/controllers/carch.cfc:39](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/admin/core/controllers/carch.cfc:39)

  Current diagnosis: `variable 'variables.configBean' has no component ref`.

```cfml
   37  			param default="" name="arguments.rc.returnURL";
   38  			param default="" name="arguments.rc.homeID";
   39  			param default=variables.configBean.getDatasource(), name="arguments.rc.datasource";
   40  			param default="" name="arguments.rc.objectid";
   41  			param default=false name="arguments.rc.locking";
```

- [ ] **E08** — `application.configBean.getContext` at [MSU-NatSci_MuraCMS/admin/assets/js/frontendtools.js.cfm:2272](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/admin/assets/js/frontendtools.js.cfm:2272)

  Current diagnosis: `variable 'application.configBean' has no component ref`.

```cfml
 2270  				CKFinder.setupCKEditor(
 2271  				instance, {
 2272  					basePath: '#application.configBean.getContext()#/core/vendor/ckfinder/',
 2273  					rememberLastFolder: true
 2274  				});
```

Review notes:

> Assessment / evidence / exception:

<a id="r04"></a>

### R04 — Mura settingsManager ownership

Resolving the manager first may enable downstream getSite returns, but those return relationships need their own evidence.

**Your input:** Which settingsManager is registered for this application? Where do controller fields receive it, and what does getSite return for the shown arguments?

Research: [MURA-DYNAMIC-RELATIONSHIPS.md](/Users/garethedwards/development/github/clif/MURA-DYNAMIC-RELATIONSHIPS.md).

- [ ] **E09** — `application.settingsManager.validate` at [MSU-NatSci_MuraCMS/admin/Application.cfc:180](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/admin/Application.cfc:180)

  Current diagnosis: `variable 'application.settingsManager' has no component ref`.

```cfml
  178
  179  		try{
  180  			if(not (structKeyExists(application.settingsManager,'validate') and application.settingsManager.validate() and isStruct(application.configBean.getAllValues()))){
  181  				application.appInitialized=false;
  182  			}
```

- [ ] **E10** — `variables.settingsManager.getSite` at [MSU-NatSci_MuraCMS/admin/core/controllers/carch.cfc:45](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/admin/core/controllers/carch.cfc:45)

  Current diagnosis: `variable 'variables.settingsManager' has no component ref`.

```cfml
   43  			param default=0 name="arguments.rc.orderno";
   44  			param default=false name="arguments.rc.cancelPendingApproval";
   45  			param default=variables.settingsManager.getSite(arguments.rc.siteid).getviewdepth(), name="session.mura.viewDepth";
   46  			param default=variables.settingsManager.getSite(arguments.rc.siteid).getnextN(), name="session.mura.nextN";
   47  			param default="" name="session.keywords";
```

Review notes:

> Assessment / evidence / exception:

<a id="r05"></a>

### R05 — ContentBox event entry contracts

The same receiver spelling occurs in handlers, templates and archived update code. The selected framework invocation must supply the type; treating every event parameter as RequestContext is too broad.

**Your input:** Which invoker supplies event for these files? Confirm its concrete class and any wrappers/overrides. Are archived update files still executed under the current framework?

Research: [RESOLUTION-RELATIONSHIPS.md](/Users/garethedwards/development/github/clif/RESOLUTION-RELATIONSHIPS.md).

- [ ] **E11** — `event.setView` at [Ortus-Solutions_ContentBox/handlers/Main.cfc:6](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/handlers/Main.cfc:6)

  Current diagnosis: `variable 'event' has no component ref`.

```cfml
    4  	function index( event, rc, prc ) {
    5  		prc.welcomeMessage = "Welcome to ContentBox Modular CMS!";
    6  		event.setView( "main/index" );
    7  	}
    8
```

Review notes:

> Assessment / evidence / exception:

<a id="r06"></a>

### R06 — ContentBox helper installation

Helpers may be supplied by layout/view setup, module injection or framework presets. This audit supplied no preset override; deployment configuration may explain some findings.

**Your input:** Show where html and cb are installed in the view scope, including module ownership and overrides. Would the real project normally configure framework presets or explicit mappings?

Research: [DI-RULES-RESEARCH.md](/Users/garethedwards/development/github/clif/DI-RULES-RESEARCH.md).

- [ ] **E12** — `html.doctype` at [Ortus-Solutions_ContentBox/layouts/Main.cfm:2](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/layouts/Main.cfm:2)

  Current diagnosis: `variable 'html' has no component ref`.

```cfml
    1  <cfoutput>
    2  #html.doctype()#
    3  <html lang="en">
    4  <head>
```

Review notes:

> Assessment / evidence / exception:

<a id="r07"></a>

### R07 — Migration callback parameter identity

884 findings use table. The callback argument must come from the resolved schema provider invocation; its return type is not sufficient. Other schema/query/grammar findings are outside this exact group.

**Your input:** Which migration runner and schema library version invoke these closures? Point to create/alter callback invocation and the supplied Blueprint constructor. Can extensions supply another object?

Research: [RESOLUTION-RELATIONSHIPS.md](/Users/garethedwards/development/github/clif/RESOLUTION-RELATIONSHIPS.md).

- [ ] **E13** — `table.modifyColumn` at [Ortus-Solutions_ContentBox/modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:65](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:65)

  Current diagnosis: `variable 'table' has no component ref`.

```cfml
   63  						"cb_content",
   64  						function( table ) {
   65  							table.modifyColumn( "title", table.string( "title", 500 ) );
   66  							table.modifyColumn( "slug", table.string( "slug", 500 ) );
   67  							table.modifyColumn( "featuredImage", table.string( "featuredImage", 500 ).nullable() );
```

Review notes:

> Assessment / evidence / exception:

<a id="r08"></a>

### R08 — Wheels model instances and selected returns

These exact names include multiple models and creation/query modes. The factory must distinguish a cached model class, a new instance, an array and a query.

**Your input:** For the examples, identify the model class and selected return mode. Which callbacks/defaults or test mocks can change the return? Avoid assuming all user variables name one model.

Research: [WHEELS-MODEL-RESEARCH.md](/Users/garethedwards/development/github/clif/WHEELS-MODEL-RESEARCH.md).

- [ ] **E14** — `user.save` at [cfwheels_cfwheels/examples/starter-app/app/models/User.cfc:135](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/app/models/User.cfc:135)

  Current diagnosis: `variable 'user' has no component ref`.

```cfml
  133  		user.roleid = arguments.userData.roleid;
  134  		user.verified = arguments.userData.verified;
  135  		if(user.save()) {
  136  			return true;
  137  		} else {
```

Review notes:

> Assessment / evidence / exception:

<a id="r09"></a>

### R09 — ContentBox request collection handoff

Request/private collection fields require the handler-to-view route and preceding assignment. Several rendering actions can target the same view.

**Your input:** Which handler/action renders each example? Show the final field assignment before rendering, plus other actions or interceptors that can replace the same field.

Research: [RESOLUTION-GAPS-PLAN.md](/Users/garethedwards/development/github/clif/RESOLUTION-GAPS-PLAN.md).

- [ ] **E15** — `prc.oCurrentAuthor.getEmail` at [Ortus-Solutions_ContentBox/modules/contentbox/models/ui/AdminMenuService.cfc:315](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/models/ui/AdminMenuService.cfc:315)

  Current diagnosis: `variable 'prc.oCurrentAuthor' has no component ref`.

```cfml
  313  			writeOutput(
  314  				variables.avatar.renderAvatar(
  315  						email = prc.oCurrentAuthor.getEmail(),
  316  						size  = "35",
  317  						class = "img-circle"
```

Review notes:

> Assessment / evidence / exception:

<a id="r10"></a>

### R10 — ColdBox event invocation

Framework parameters and local aliases need the actual caller contract. Test fixtures can invoke the same functions with mocks or unrelated objects.

**Your input:** Which invocation path supplies the example event? Is it a RequestContext, a wrapper or a mock? Identify fixtures that intentionally use another type.

Research: [RESOLUTION-RELATIONSHIPS.md](/Users/garethedwards/development/github/clif/RESOLUTION-RELATIONSHIPS.md).

- [ ] **E16** — `event.getEventCacheableEntry` at [ColdBox_coldbox-platform/system/Bootstrap.cfc:245](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/Bootstrap.cfc:245)

  Current diagnosis: `variable 'event' has no component ref`.

```cfml
  243  			// ****** EVENT CACHING CONTENT DELIVERY *******/
  244  			var refResults  = {};
  245  			var eCacheEntry = event.getEventCacheableEntry();
  246
  247  			// Verify if event caching item is in selected cache
```

Review notes:

> Assessment / evidence / exception:

<a id="r11"></a>

### R11 — ColdBox controller storage and injection

Controller identity may come from startup, constructor injection or a caller argument. These origins cannot be inferred solely from the variable name.

**Your input:** Show each controller origin and injection rule. Are any examples constructing a subsystem with a custom controller or test double?

Research: [DI-RULES-RESEARCH.md](/Users/garethedwards/development/github/clif/DI-RULES-RESEARCH.md).

- [ ] **E17** — `variables.controller.announce` at [ColdBox_coldbox-platform/system/FrameworkSupertype.cfc:52](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/FrameworkSupertype.cfc:52)

  Current diagnosis: `method 'getInterceptorService' in coldbox.system.web.Controller has no component return type (chain to 'announce')`.

```cfml
   50  			callStackGet()
   51  		);
   52  		variables.controller.getInterceptorService().announce( argumentCollection = arguments );
   53  	}
   54  	function populateModel() cbMethod{
```

Review notes:

> Assessment / evidence / exception:

<a id="r12"></a>

### R12 — Lucee driver and Field/Group collections

The latest pass resolved 20 guarded Group calls. Remaining field/driver calls need the selected driver registry and collection element alternatives; a negative Group check alone does not prove Field.

**Your input:** Where is the driver registry filled and which drivers can be selected? Are getCustomFields elements exclusively Field and Group, including extensions and later writes?

Research: [RESOLUTION-IMPLEMENTATION-STATUS.md](/Users/garethedwards/development/github/clif/RESOLUTION-IMPLEMENTATION-STATUS.md).

- [ ] **E18** — `field.getDefaultValue` at [lucee_Lucee/core/src/main/cfml/context/admin/debugging.templates.create.cfm:148](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/debugging.templates.create.cfm:148)

  Current diagnosis: `variable 'field' has no component ref`.

```cfml
  146  				<cfset default=custom[field.getName()]>
  147  			<cfelseif isNew>
  148  				<cfset default=field.getDefaultValue()>
  149  			<cfelse>
  150  				<cfset default="">
```

Review notes:

> Assessment / evidence / exception:

<a id="r13"></a>

### R13 — TestBox assertion method installation

The assertion object and copied/dynamically installed methods are distinct relationships. A declared base API may not contain the runtime methods.

**Your input:** Show the assertion object constructor and the method-copy/registration loop. Which methods are public and do copied bodies retain the receiving object as this?

Research: [RESOLUTION-IMPLEMENTATION-STATUS.md](/Users/garethedwards/development/github/clif/RESOLUTION-IMPLEMENTATION-STATUS.md).

- [ ] **E19** — `variables.assert.fail` at [Ortus-Solutions_TestBox/system/Expectation.cfc:62](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/system/Expectation.cfc:62)

  Current diagnosis: `variable 'variables.assert' has no component ref`.

```cfml
   60  	function fail( message = "", detail = "" ){
   61  		arguments.message = resolveMessage( arguments.message );
   62  		variables.assert.fail( argumentCollection = arguments );
   63  	}
   64
```

Review notes:

> Assessment / evidence / exception:

<a id="r14"></a>

### R14 — FW/1 stored user and department collections

Example services store factory-produced beans in structs. Untyped save arguments and all writers must agree before a collection element type can be inferred.

**Your input:** Confirm userBean/departmentBean factory mappings and every save caller. Can other objects enter these stores? Is list() a struct of beans or sometimes another representation?

Research: [RESOLUTION-GAPS-PLAN.md](/Users/garethedwards/development/github/clif/RESOLUTION-GAPS-PLAN.md).

- [ ] **E20** — `user.setDepartmentId` at [framework-one_fw1/examples/userManager/controllers/user.cfc:37](/Users/garethedwards/corpus/framework-one_fw1/examples/userManager/controllers/user.cfc:37)

  Current diagnosis: `variable 'user' has no component ref`.

```cfml
   35          variables.fw.populate( cfc = user, trim = true );
   36          if ( structKeyExists( rc, "departmentId" ) && len( rc.departmentId ) ) {
   37              user.setDepartmentId( rc.departmentId );
   38              user.setDepartment( variables.departmentService.get( rc.departmentId ) );
   39          }
```

Review notes:

> Assessment / evidence / exception:

<a id="r15"></a>

### R15 — CommandBox task print object

Most examples are task/update files whose runtime host may supply print. The source does not necessarily declare that host through extends.

**Your input:** Are these files always invoked as CommandBox tasks? Show the runner that supplies BaseTask/print and identify historical scripts no longer run.

Research: [RESOLUTION-GAPS-PLAN.md](/Users/garethedwards/development/github/clif/RESOLUTION-GAPS-PLAN.md).

- [ ] **E21** — `print.greenLine` at [Ortus-Solutions_ContentBox/build/BuildDocs.cfc:52](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/BuildDocs.cfc:52)

  Current diagnosis: `variable 'print' has no component ref`.

```cfml
   50  		// Build API Docs using DocBox
   51  		apiDocs( arguments.version );
   52  		print.greenLine( "Zipping apidocs to #variables.exportsDir#" ).toConsole();
   53  		cfzip(
   54  			action    = "zip",
```

Review notes:

> Assessment / evidence / exception:

<a id="r16"></a>

### R16 — Wheels engine adapter selection

Known engine constructors have a common API, but staged/published fields, replacement writes and method overrides must be checked. Prior source-copy probes are hypothetical, not implemented gains.

**Your input:** Which engine selects each adapter and where is the slot published? Identify all replacement writers and whether an extension may install a different adapter.

Research: [WHEELS-ADAPTER-RESEARCH.md](/Users/garethedwards/development/github/clif/WHEELS-ADAPTER-RESEARCH.md).

- [ ] **E22** — `application.wheels.engineAdapter.parseFormKey` at [cfwheels_cfwheels/vendor/wheels/Dispatch.cfc:150](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/Dispatch.cfc:150)

  Current diagnosis: `variable 'application.wheels.engineAdapter' has no component ref`.

```cfml
  148
  149  				// Use engine adapter for cross-engine bracket-parsing differences
  150  				local.nested = application.wheels.engineAdapter.parseFormKey(local.key, local.name);
  151  				if (!StructKeyExists(local.rv, local.name)) {
  152  					local.rv[local.name] = {};
```

Review notes:

> Assessment / evidence / exception:

## Four useful false-positive triage cases

These examples exercise different diagnoses. The initial assessments below are source observations, not completed resolver fixes. Their counts are not added to the receiver-group total.

### E23 — A control-flow word reported as a function

Three Lucee entries report `elseif`. In this excerpt it follows an if branch and appears to introduce another branch. This is a strong extraction-gap candidate. Confirm the intended engine syntax and whether it is valid in this exact script context; it should not be fixed by inventing a component method.

- [ ] Review [lucee_Lucee/core/src/main/java/resource/component/org/lucee/cfml/Query.cfc:77](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/java/resource/component/org/lucee/cfml/Query.cfc:77)

Current diagnosis: `not found in extends chain`.

```cfml
   75  			}
   76  			// If SQL comment found, consume until closing comment.
   77  			elseif ( Mid(Sql,Pos,2) === '/*' )
   78  			{
   79  				var Len = 2 ;
```

> Assessment / evidence / exception:

### E24 — An intentionally invalid call inside an exception assertion

This call is inside `$assert.throws(function(){...})`. The source deliberately expects a failure, so this is an intentionally invalid test call rather than a missing receiver relationship. Help decide whether a future report should annotate expected failures. A blanket exemption for calls inside closures would hide real problems.

- [ ] Review [Ortus-Solutions_TestBox/tests/specs/AssertionsTest.cfc:28](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/tests/specs/AssertionsTest.cfc:28)

Current diagnosis: `not found in extends chain`.

```cfml
   26  	function testThrows(){
   27  		$assert.throws( function(){
   28  			var hello = invalidFunction();
   29  		} );
   30  	}
```

> Assessment / evidence / exception:

### E25 — A framework helper provided to a view

FW/1 has 61 case-insensitive buildURL findings across this snapshot. This example is a view explicitly exercising that helper. Show the view invocation that supplies it and whether a configuration/preset is required. That would distinguish a runtime-host inference gap from an incomplete extracted configuration.

- [ ] Review [framework-one_fw1/examples/buildurl/views/main/default.cfm:3](/Users/garethedwards/corpus/framework-one_fw1/examples/buildurl/views/main/default.cfm:3)

Current diagnosis: `no qualifier, not in file`.

```cfml
    1  <cfoutput>
    2  	<h1>Testbed for buildURL()</h1>
    3  	<p><a href="#buildURL('')#">buildURL('')</a></p>
    4  	<p><a href="#buildURL('.')#">buildURL('.')</a></p>
    5  	<p><a href="#buildURL(':')#">buildURL(':') -- only valid in a subsystem-based app</a></p>
```

> Assessment / evidence / exception:

### E26 — A component exists locally but its qualified path does not resolve

The scan reports `testbox.system.runners.BDDRunner` missing, while the extracted repository contains `system/runners/BDDRunner.cfc`. This suggests an effective mapping/entry-root question; finding that file alone does not prove the deployed mapping. Confirm how `/testbox` is set for the runner. Do not solve it by stripping arbitrary namespaces.

- [ ] Review [Ortus-Solutions_TestBox/system/TestBox.cfc:853](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/system/TestBox.cfc:853)

Current diagnosis: `component 'testbox.system.runners.BDDRunner' does not exist (calling 'run')`.

```cfml
  851  			if ( structKeyExists( target, "run" ) ) {
  852  				// Run via BDD Style
  853  				new testbox.system.runners.BDDRunner( options = variables.options, testbox = this ).run(
  854  					target,
  855  					arguments.testResults,
```

> Assessment / evidence / exception:

## False-positive and configuration checks

Before proposing a resolver rule, these checks can explain a finding more directly:

- **Runtime host:** Does a framework runner supply a base class, helper or parameter that the extracted source does not declare? Supply the actual entry path.
- **Mappings and dependencies:** Does deployment install the named component/module outside these roots? Supply its mapping and version. An absent source file cannot provide a verified method definition.
- **Conditional dispatch:** Is the call guarded by a type/method existence check? Show the complete condition and any reassignment or escape before the call.
- **Dynamic methods:** Does onMissingMethod, a plugin, a mixin or a registration loop install the method? Show the producer and registration order. Dynamic acceptance and a precise definition are separate outcomes.
- **Historical code:** Is the file an archived migration, generated template or fixture for an older API? Confirm its intended framework version; a missing method may be a real incompatibility in the current source snapshot.
- **Parser location:** If the excerpt points to the wrong call, or repeats a call that does not exist, flag the example and exact source span. That is an extraction issue rather than receiver inference.


The scan logs also report unresolved base components. For example, Wheels reports `modules.BaseModule` (765 affected calls) and `tests.Test` (112); Mura reports unresolved framework bases; ContentBox reports `cbsecurity.models.validators.AuthValidator` (one). These log counts are **not added** to the receiver groups: they may overlap and have different aggregation semantics. Deployment mapping evidence is more useful here than guessing a base from its name.

## Find an example outside these cards

Full JSON reports, with reasons and candidate definitions where available:
- [Wheels report](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/wheels.json) · [scan log](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/wheels.log)
- [Mura report](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/masa.json) · [scan log](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/masa.log)
- [ContentBox report](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/contentbox.json) · [scan log](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/contentbox.log)
- [ColdBox report](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/coldbox.json) · [scan log](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/coldbox.log)
- [Lucee report](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/lucee.json) · [scan log](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/lucee.log)
- [TestBox report](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/testbox.json) · [scan log](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/testbox.log)
- [FW/1 report](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/fw1.json) · [scan log](/Users/garethedwards/development/github/clif/target/resolution/relationship-guards/fw1.log)

You can search those files for a receiver/function or send a source file and call line. The example-to-report record mapping is saved in [review examples](/Users/garethedwards/development/github/clif/target/resolution/user-review/examples.json).

## Review boundary

This report identifies promising relationships and requests evidence; it does not label all listed entries false positives or promise a percentage gain. No generated registry, factory, adapter or collection rule is considered complete because its examples appear here. A proposed fix still needs counterexamples, failing-before regressions and a per-entry audit on the same roots/configuration.
