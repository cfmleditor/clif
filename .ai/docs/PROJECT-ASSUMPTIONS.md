# Project assumptions for resolution

Clif uses recognizable source layouts to supply missing deployment and framework
context. These are best guesses about how a project runs, authorized as defaults;
they are not proofs that a particular server has that configuration. Explicit
configuration and source declarations can replace them. No corpus project needs
a new `.clif.json` to obtain the default improvements measured below.

## Defaults and their evidence

| Context | Required source evidence | Assumption |
| --- | --- | --- |
| Wheels public root | `public/Application.cfc` sets `this.wheels.rootPath` from its template directory, declares `/wheels` pointing at `vendor/wheels`, and has adjacent outer startup statements constructing `wheels.Injector("wheels.Bindings")` and retrieving `global`; exact minimal `public/ApplicationProxy.cfc` extends Application; Injector and Bindings exist | `/` points at `public`, so framework tests inherit the application's startup context |
| TestBox namespace | `system/BaseSpec.cfc`, `system/Assertion.cfc`, and `tests/Application.cfc` declaring `/testbox` to the project root | Nested library components use the project's `/testbox` namespace |
| ColdBox namespace | `system/Bootstrap.cfc`, `system/web/Controller.cfc`, and `tests/Application.cfc` declaring `/coldbox` to the project root | Nested library components use the project's `/coldbox` namespace |
| FW/1 namespace | `framework/one.cfc`, `framework/ioc.cfc`, and root Application declaring `/framework` to that directory | Nested library components use the project's `/framework` namespace |
| Masa/Mura namespace | `core/mura/configBean.cfc`, `core/mura/bean/beanFactory.cfc`, `core/appcfc/applicationSettings.cfm`, and root Application's evaluated `/mura` declaration | Nested library components use the project's `/mura` namespace |
| ContentBox namespace | `modules/contentbox/ModuleConfig.cfc`, `models/content/ContentService.cfc` under that module, and root Application declaring `/contentbox` to the module directory | Nested library components use the project's `/contentbox` namespace |
| Framework context | The profiles above identify exactly one project owner among workspace roots | With no editor config, enable the existing Wheels, ColdBox, ContentBox, TestBox, Mura or FW/1 presets matching that project; ContentBox includes ColdBox |

Namespace assumptions do not import lifecycle methods, beans or shared variables
from a host application. Mapping expressions must evaluate to the expected
physical directory. The search visits at most 32 ancestors. The nearest recognized
layout owns a namespace even if its declaration cannot be evaluated; it does not
borrow a conflicting outer namespace. An encountered Wheels public application
that fails the contract stops root discovery.

An existing `server.json` or `.cfconfig.json` stops layout assumptions at that
level. Known deployment declarations use existing resolution rules; the presence
of these files does not imply their web-root settings are all understood. Add an
explicit root mapping if the deployed root is unavailable to the analyzer.

No uniform framework context is assumed for Lucee's heterogeneous test corpus.
Masa receives its namespace assumption and the existing Mura scope/template preset.
Multiple recognized project owners receive no global automatic framework preset.

## Overrides

Editor `mappings` have priority over application, deployment and assumed mappings.
A concrete application mapping replaces an assumption; a named mapping assigned
an unevaluable expression suppresses its assumed fallback. Names are compared
case-insensitively.

For example, at a Wheels project root, a different deployment can declare:

```json
{
  "workspaceName": "my-project",
  "mappings": { "/": "./deployment-root", "wheels": "./vendor/wheels" },
  "frameworks": ["wheels"]
}
```

The existence of an editor config makes its `frameworks` selection authoritative.
`"frameworks": []` opts out of automatic presets; add desired names explicitly
when creating a config for unrelated settings. Inherited configured frameworks
still follow the existing config merge rules. Nonempty editor framework settings
also take priority. Root and namespace defaults remain independently
available unless overridden by mappings or stopped by deployment configuration.

Custom `componentResolvers` precede preset rules. Explicit component inheritance
precedes implicit bases. Real framework source precedes bundled API stubs. A
preset still has its existing dynamic acceptance behavior where neither source
nor a bundled API describes the target, so disappearing findings are not all
source-verified method resolutions. The CLI prints the inferred framework names;
the editor tracks the source and missing marker paths as discovery dependencies.

## Failure modes and next configuration opportunities

These defaults can be wrong when a library runs under a different host application,
a deployment uses a different document root, or conventionally named framework
receivers actually refer to another object. Correct the root/namespace mappings,
select frameworks explicitly, or add a receiver rule. Dynamic module names,
external bean directories, startup includes outside the project, runtime plugins,
helper hosts and application-specific return types still need source evidence or
configuration (`beanPaths`, `startupFiles`, `componentResolvers`, mappings).
Those settings are opportunities for later evidence-backed defaults, not permission
to pick arbitrary matching components.

The bounded Wheels Injector startup inference is separate from a preset: it checks
a reviewed normalized Injector source fingerprint and a closed literal binder.
A changed container or effectful binding cannot acquire that source proof. A Wheels
preset may nevertheless supply framework types as an explicitly documented
convention. See the checkpoint 13/14 records for the distinction.

## Measured default corpus outcome

Same seven scan roots, including the explicit Wheels vendor scan, with no new
project configs and the same 8,151 indexed/scanned files:

| Corpus | Previous default | Assumed default |
| --- | ---: | ---: |
| Wheels | 8,864 | 2,184 |
| Masa | 20,262 | 3,723 |
| ContentBox | 6,777 | 1,378 |
| ColdBox | 3,081 | 1,034 |
| Lucee | 1,417 | 1,318 |
| TestBox | 597 | 246 |
| FW/1 | 492 | 253 |
| Total | 41,490 | 10,136 |

This is a net 31,354 fewer diagnostics (75.6% of the preceding default and 75.7%
below the original 41,707 baseline). Diagnostic identity is file, line, function,
and reason: 32,352 removed and 998 added. Using file/line/function identity alone,
31,921 entries disappear and 567 appear. These counts include 292 location
corrections in checkpoint 30; those are not additional resolved or newly broken
calls. Candidate metadata is excluded
from these counts. Resolved calls plus diagnostic entries increase from 174,833
to 179,806 as more
chains and inherited methods become checkable; these are not raw extraction counts.

The additions include Masa return-type diagnostics, FW/1 injected services,
ColdBox fluent/mixin chains, mocked TestBox methods, Wheels runtime plugins and
computed adapters. They remain unresolved relationships, not established defects
in those applications. Per-entry reports, additions and source excerpts are under
`target/resolution/automatic-frameworks/` (historical) and the later checkpoint
directories, most recently `target/resolution/chain-spans-lucee-admin/`; historical counts remain in
`RESOLUTION-IMPLEMENTATION-STATUS.md` and `RESOLUTION-USER-REVIEW.md`.

Checkpoint 15 enables the existing Mura preset that automatic detection previously
omitted. Incrementally, Masa falls 19,702 to 16,169 (4,328 diagnostics removed,
795 added). Of the additions, 793 are deeper return-type gaps and two are template
methods. Ignoring reason replacements, 3,632 calls stop being reported and 99
begin being reported; 696 diagnostics change reason. Masa retains 839 files and
32,938 filtered checked calls. Detailed additions and source excerpts are under
`target/resolution/mura-assumptions/`.

## Mura startup factory and object helpers (checkpoints 16–17)

The recognized Mura layout also supplies its application's declared `/muraWRM`
namespace to nested source. This lets startup includes retain their host namespace
behind the restrictive physical `core/Application.cfc`. Explicit or unknown local
mappings still override that default; the actual root declaration must evaluate
to the project directory.

With a Mura scope preset and no configured bean roots or startup files, one
recognized startup DI/1 policy rooted exactly at `/mura` can supply literal
`getBean` result types. Existing analysis supplies recursive/shallow discovery,
exclusions, aliases, declarations, external registrations and ambiguity guards.
Only known component IDs acquire concrete rules; there is no generic basename
fallback for arbitrary IDs. Rules are appended after explicit resolvers. Unknown
external replacements, cycles, duplicate names and computed roots withhold types.
Infrastructure components are excluded. Literal retrieval permits transients;
this does not relax the separate singleton-only injection policy.

The runtime assumption is that normal Mura initialization runs and its recognized
factory serves calls in this project's host context. Static registration analysis
is conservative but does not prove all runtime plugins, database-registered entities,
factory replacements or request overrides are absent. Multiple matching factory
policies acquire no default. Specify `beanPaths`, `startupFiles` or a custom
`componentResolvers` rule when the deployment uses another factory context.

Mura's preset also defines zero-argument object forms: `siteConfig()` returns the
standard `mura.settings.settingsBean`; `globalConfig()` returns
`mura.configBean`; `getContentRenderer()` implements
`mura.content.contentRenderer`. The first two follow the object's branch in
MuraScope's implementation. The renderer contract is an assumption about custom
site renderers and theme overlays, not proof of their concrete runtime class.
Property getter/setter forms and calls with explicit force arguments are excluded.
Real component methods remain checked; these rules do not accept unknown APIs
just because a target source is missing. Custom resolver rules can replace any
of these contracts, and explicit framework selection can omit Mura entirely.

The factory-context step reduces Masa from 16,169 to 7,999: 8,385 diagnostics
removed and 215 added. Of those additions, 209 expose deeper returns, three are
methods and three are variables. Files remain 839; filtered checked calls grow
32,938 to 33,645 as chains become checkable. The remaining six projects retain
exact diagnostic multisets. This step's combined total is 16,390, subject to the
object-helper measurement recorded next in the implementation status.

The Mura preset also assumes normal startup application objects:

| Receiver | Standard component | Startup evidence |
| --- | --- | --- |
| `application.configBean` | `mura.configBean` | Literal construction before factory setup |
| `application.serviceFactory` | `mura.bean.beanFactory` | Startup publishes its constructed service factory |
| `application.settingsManager` | `mura.settings.settingsManager` | Literal service list populates application objects through getBean |
| `application.rbFactory` | `mura.resourceBundle.resourceBundleFactory` | Literal construction after service installation |
| `application.mura` | `mura.MuraScope` | Factory's mura alias is published to application scope |

These are conventions, not guaranteed runtime values: startup's custom-manager
include can replace them, and another factory can override registered services.
Use an earlier explicit component resolver for each replacement or omit the Mura
preset. Argument names such as `arguments.bean` are not sufficient evidence to
choose one of the many different bean types. API stubs for the new preset targets
are generated from Mura's existing pinned source commit
`696383140578f8dea3ece26f80cd7bfb370ddf0f`; real workspace source still wins.

Standard Mura admin controllers establish three further request-context defaults:
`rc.contentBean` is `mura.content.contentBean`, `rc.siteBean` is
`mura.settings.settingsBean`, and `rc.feedBean` is `mura.content.feed.feedBean`.
This includes an `arguments.rc` prefix through the existing name-only rule behavior.
Controller replacements and alternate payload types can override these rules.
Unqualified `bean` and `arguments.bean` have no default component.

The zero-argument Mura helpers `content()`/`getContentBean()` assume the standard
primary content bean, and `currentUser()`/`getCurrentUser()` assume
`mura.user.sessionUserFacade`. The latter is cfobject's lazy construction when
request.currentUser is absent. The former is an event payload convention.
Existing request objects may be replaced by custom code, so these are explicitly
runtime assumptions. Calls that pass a property argument or set a value are not
covered by the zero-argument rules; `getFeed(entityName)` needs an entity argument
and has no generic zero-argument feed assumption.


## Conditional factory targets (checkpoint 20)

A literal bean alias may select different components at runtime. For retrieval,
Clif now uses an actual registered target that every other alternative extends.
This reuses the configured-registration common-base rule in source-discovered
DI/1 policies; it does not invent an ancestor or combine unrelated APIs.
Every alternative must resolve under the existing policy. Unknown targets,
external replacements, cycles and unrelated classes still withhold a result.
Singleton setter/property injection retains its requirement for one exact target.

Mura startup selects `contentGatewayAdobe` or `contentGatewayLucee` according to
engine (`core/appcfc/onApplicationStart_include.cfm:212–214`); the latter extends
`contentGatewayAdobe` (`core/mura/content/contentGatewayLucee.cfc:44`). Thus only
the Adobe base API is assured by the recognized alternatives. Runtime plugins
that replace the factory or alias remain outside this source model; an explicit
component resolver can describe a deployment replacement, and configured bean
paths/startup files replace the automatic Mura factory context.

This removes 86 findings with zero additions: Masa 5,318 -> 5,232 and the seven
projects 13,709 -> 13,623. The other six diagnostic multisets are unchanged.
Coverage stays at 8,151 files (839 Masa). Masa filtered checked calls increase
33,664 -> 33,665; overall 179,656 -> 179,657. Candidate annotations are excluded
from diagnostic identity. Representative removed calls include controller gateway
methods, application services, literal getBean chains and two iterator methods.
These figures measure resolution coverage, not executed CFML runtime behavior.


## Additional admin payloads and application services (checkpoints 21–22)

The Mura preset supplies these standard admin request-context objects. The exact
receiver names also match the existing `arguments.rc` spelling; generic rc.bean
and rc.item have no component default.

| Receiver | Component | Assignment evidence under admin/core/controllers |
| --- | --- | --- |
| rc.userBean | mura.user.userBean | cusers:192/213/252 and ceditprofile:86 read the user manager |
| rc.categoryBean | mura.category.categoryBean | ccategory:64/71/77 reads, updates or creates a category |
| rc.emailBean | mura.email.emailBean | cemail:81 reads the email manager |
| rc.listBean | mura.mailinglist.mailinglistBean | cmailinglist:62/66/73/100 reads or creates a mailing list |
| rc.changeset | mura.content.changeset.changesetBean | cchangesets:131/142/147 reads a changeset |
| rc.pluginConfig | mura.plugin.pluginConfig | csettings:105 calls pluginManager.getConfig |
| rc.trashItem | mura.trash.trashItemBean | ctrash:74 obtains trashManager.getTrashItem; trashIterator.packageRecord constructs trashItemBean |
| rc.trashIterator | mura.trash.trashIterator | ctrash:64 obtains trashManager.getIterator, which constructs trashIterator |

These are payload conventions under normal startup/controller execution.
Existing request objects, custom controllers or plugins can change them. They
are not general argument-name inference or guarantees about every runtime value.
An earlier explicit component resolver can replace the convention; framework
selection can omit Mura. Tests confirm a replacement resolver wins, unknown
methods on actual source remain reported, and lookalike names do not match.

Startup's literal serviceList at core/appcfc/onApplicationStart_include.cfm:468
publishes additional standard application objects through the recognized factory:

| Application receiver suffix | Component |
| --- | --- |
| utility | mura.utility |
| pluginManager | mura.plugin.pluginManager |
| contentManager | mura.content.contentManager |
| eventManager | mura.plugin.pluginManager |
| contentRenderer | mura.content.contentRenderer |
| contentUtility | mura.content.contentUtility |
| categoryManager | mura.category.categoryManager |
| clusterManager | mura.cluster.clusterManager |
| contentServer | mura.content.contentServer |
| changesetManager | mura.content.changeset.changesetManager |
| scriptProtectionFilter | mura.Portcullis |
| permUtility | mura.permission |
| emailManager | mura.email.emailManager |
| loginManager | mura.login.loginManager |
| mailinglistManager | mura.mailinglist.mailinglistManager |
| userManager | mura.user.userManager |
| dataCollectionManager | mura.content.dataCollection.dataCollectionManager |
| feedManager | mura.content.feed.feedManager |
| sessionTrackingManager | mura.user.sessionTracking.sessionTrackingManager |
| favoriteManager | mura.content.favorite.favoriteManager |
| raterManager | mura.content.rater.raterManager |
| dashboardManager | mura.dashboard.dashboardManager |
| autoUpdater | mura.autoUpdater.autoUpdater |

The declared aliases are material: eventManager uses pluginManager,
scriptProtectionFilter uses Portcullis, and permUtility uses permission
(startup:300–302). contentGateway remains governed by source-verified conditional
factory resolution, rather than an unconditional engine class assumption.
As with existing application defaults, custom-manager startup includes and factory
replacements can invalidate these conventions; override the affected receiver
through componentResolvers or omit the Mura preset. API stubs were regenerated
from the same pinned commit 696383140578f8dea3ece26f80cd7bfb370ddf0f (63 components).
Real workspace source continues to take priority over bundled stubs.

Checkpoint 21 removes 270 diagnostics with no additions (Masa 5,232 -> 4,962).
Checkpoint 22 removes 305 and adds one (4,962 -> 4,658): the legacy SOAP call
application.userManager.readByEmail now has a known receiver but no declared
method in the scanned userManager. This is not confirmed runtime breakage.
Together they remove 574 net findings, leaving 13,049 overall, 68.7% below the
original baseline. All other six projects retain identical diagnostic multisets.
File coverage stays at 8,151 (839 Masa). Filtered checked calls grow by 32,
179,657 -> 179,689; this is not a raw extraction equivalence claim.


## Source-discovered factory ownership for setters (checkpoint 23)

Setter injection now consults discovered DI/1 factory ownership before testing
conventional bean-folder layouts. A component under a recognized factory root
can acquire setter dependency types without also requiring that root in beanPaths.
This uses the existing source policy rather than a name-only configBean assumption.
The same singleton lifetime, exclusions, unknown overrides, literal aliases,
conflicts and blocked-policy checks still apply. Components outside the factory's
managed scope receive no new injection context. Existing configured bean roots
and FW/1 fallback behavior remain available when no factory policy serves a file.

For Mura this inherits the documented normal-startup assumption that its discovered
service factory owns core/mura components. Custom construction that bypasses
injection is outside that model; explicit beanPaths/startupFiles select a different
factory context, and explicit types or resolver rules can describe replacements.
Generic feed/bean argument names are not assigned new global preset types.

The seven-project comparison removes 495 diagnostics and adds 16 (479 net), all
in Masa: 4,658 -> 4,179. The 16 additions reclassify already-reported calls into
return-type gaps: twelve settingsBean.getRBFactory chains and four manager.save
chains. No new call sites are reported. File coverage remains 8,151 (839 Masa);
filtered checked calls increase 179,689 -> 179,731. All six other diagnostic
multisets are identical. Per-entry/source excerpts are retained under
`target/resolution/source-policy-setters/`. Representative removals include
injected configBean fields, manager fields and downstream feed/address arguments.
These are resolution results, not executed CFML runtime behavior.


## Source-shaped tooling contexts (checkpoints 24–25)

After recognizing exactly one project owner, automatic framework selection also
looks for tooling entry points. This runs only without explicit editor config or
framework selection; deployment-config and mixed-owner guards remain unchanged.
Directory reads and failed source reads are tracked for editor invalidation.

| Context | Required source shape | Project-wide assumption |
| --- | --- | --- |
| cfmigrations | A CFC directly under migrations or modules/contentbox/migrations declares both up and down, with first argument schema and second qb or query; types are absent, any, or the corresponding qb API type | Standard schema/table/query/qb names use the existing cfmigrations preset and bundled qb APIs |
| CommandBox | task.cfc, a direct build CFC, or a CFC under commands/cli/src/commands declares run; it explicitly extends commandbox.*, calls undeclared bare getCWD, or calls shell.getUtil | Tasks/commands run under CommandBox, supplying its print and command APIs and conventional task/command bases |

Each directory probe reads at most 128 CFCs and visits at most 128 directories;
command traversal goes at most five levels deep. Migration/build folders are
not recursively scanned. Hidden entries are excluded. A run parameter alone,
commented-out calls, a locally declared getCWD, ordinary build folders and
primitive-typed migration arguments do not select these contexts.

These are explicitly project-wide naming conventions, not source declarations
that every table, schema, query or print value in the project holds that type.
A project combining unrelated objects under these names should select frameworks
explicitly and omit the conflicting preset, or provide earlier componentResolvers
for replacements. The presets' existing dynamicIfMissing behavior is retained.
CommandBox Print/PrintBuffer deliberately implement onMissingMethod; acceptance
of dynamic printing methods is framework behavior, not proof of a declared method
for every print call. Unknown schema methods remain checked against bundled APIs.
No new stub source revision or generated API signature was introduced.

Editor discovery now records ReadDir roots, including failures, so adding or
removing tooling sources invalidates cached context. Boundary checks exclude
similarly named sibling directories. This also benefits other source discovery;
configuration still replaces the resolver policy rather than reusing it.

Migration context removes 1,140 diagnostics and adds four (1,136 net), all in
ContentBox. CommandBox context removes 702 and adds eight (694 net) across
Wheels, ContentBox, ColdBox and TestBox. Diagnostic identity is file, line,
function and reason; candidate metadata is excluded. Combined: 1,842 removed,
12 added, 1,830 net; raw call identities lose 1,831 and gain one. Additions expose
four configurable qb grammar returns, four unresolved Wheels command bases,
three service API mismatches and one untyped test argument. They are resolution
gaps, not confirmed runtime defects. The total is 10,740, 14.6% below checkpoint
23 and 74.2% below the original 41,707. Files remain 8,151, and filtered checked
calls grow 179,731 -> 179,807. The repeat scan retains all diagnostics exactly.
See RESOLUTION-ITERATION-REVIEW.md for the 20-pass review and remaining limits.

## Checkpoint 26: caller budgets follow method ownership

The batch resolver now applies its 60-caller-file budget after checking whether
calls belong to the target method. Lexical candidates for common names include
other components, declarations and comments; those no longer consume the owner
budget. A separate 512-candidate-file bound limits the search. The existing
40-passed-call-site bound remains. Unknown receivers, unknown supplied values,
unreadable files and exceeded bounds still withhold inference. Constructor
argument inference retains its separate existing 60-candidate-file policy.

This is source-based caller inference, enabled only for batch scans with
InferArgsFiles. It assumes the indexed callers describe the possible values;
external callers, runtime dispatch and plugin replacement are not proven by this
analysis. `--no-infer-args` disables it. No bean-name preset or API stub was added.
It uses existing ownership checks and does not yet build a dedicated index by
resolved owner or change those checks' inheritance/alternative semantics.

The same seven-corpus scan drops **10,740 -> 10,371**, with **369 findings removed
and zero added**, all in Masa (**4,179 -> 3,810**). Other corpora are identical.
The removals are bean calls in settingsDAO (135), userDAO (75),
feedDAO (72), categoryDAO (35), emailDAO (29), memberDAO (10), reminderDAO (8)
and mailinglistDAO (5). For example, feedManager reads a feedBean through feedDAO
and passes it to that DAO's update method; unrelated update receivers no longer
prevent following that source relationship. Plugin events can still mutate runtime
objects, which remains an inference limitation.

Coverage remains **8,151 files**; filtered checked calls increase by one,
**179,807 -> 179,808**. These are not raw parser-call equivalence counts. The
positive ownership-budget regression fails before the fix and passes after it;
unknown receivers, missing methods and all three budget boundaries have tests.
Reports and per-entry diffs are under target/resolution/owner-caller-budget.

Checkpoint 26 validation: `make check` (build, vet, short race tests, pinned lint
and vulnerability check), separate `go test -short ./...`, and `git diff --check`
pass. No CFML runtime execution or published CI result is claimed.

The final repeat scan in target/resolution/owner-caller-repeat matches every
diagnostic and file/filtered-check count across all seven corpora.

## Checkpoint 27: remove caller count limits

At the user's request, ordinary-method and constructor argument inference no
longer stop at a candidate-file, owner-file or supplied-call-site count. The
former 512/60/40 bounds and their bookkeeping are removed. Candidate lookup,
cached parsing and per-argument memoization remain; recursive inference still
has its existing depth bound. This applies only to batch caller inference,
not to unrelated parser/resolver traversal guards or cache capacity policies.

Unknown receivers, unknown argument expressions, missing source and conflicting
source evidence retain their existing handling. Source inference still assumes
indexed callers describe the possible inputs; external calls, dynamic dispatch
and plugin mutation remain limitations. `--no-infer-args` disables caller
inference. Large common-name buckets can now cost more work, since no count
cutoff substitutes an empty answer. No performance equivalence is claimed.

The replacement regressions demonstrate inference past each former limit,
including 66 constructor caller files. All four positive cases fail with the
old limits enabled and pass after removal. An unknown argument after the former
40-site cutoff still withholds inference, and missing methods remain checked.

The seven-corpus report remains **10,371** unresolved across **8,151 files**,
with **zero removed and zero added per diagnostic identity** and identical
filtered coverage (**179,808 checked**). This change removes a general coverage
restriction but adds no measured resolution gains to this corpus. The historical
checkpoint 26 limits above are superseded. Reports, before-fix failures and
per-entry audits are under target/resolution/uncapped-callers.

Checkpoint 27 validation: full `make check`, separate `go test -short ./...`,
and `git diff --check` pass. No CFML runtime or published CI result is claimed.

## Checkpoint 28: optional caller inference limits

Caller file and call-site limits are now optional through `.clif.json`:

```json
{
  "callerInferenceLimits": {
    "maxCandidateFiles": 512,
    "maxCallerFiles": 60,
    "maxCallSites": 40,
    "maxConstructorFiles": 60
  }
}
```

All defaults remain unlimited. Each positive value enables that cutoff;
omitted, zero and negative values leave it unlimited. Candidate files include
lexical matches for ordinary methods; owner files contain calls assigned to the
target method. Constructor candidates have their own file setting. Supplied
argument sites share maxCallSites across method and constructor inference.
Exceeding a limit withholds the type instead of accepting a partial caller set.
The policy block replaces an inherited policy as a whole; an explicit empty
block resets every limit. Unknown receiver/value handling and the recursive
depth guard remain unchanged. `--no-infer-args` disables inference altogether.

The policy flows through config decoding/merge, daemon settings, scan options,
the batch resolver and caller execution-context views. Editor sessions preserve
it when later configs omit the block, but live editor resolution still does not
build the caller index. Server exports retain the policy without enabling caller
inference that was previously off. CLI unresolved/explain and the MCP scan path
share the batch options. Large unlimited searches may require more work.

Five enabled-limit regression cases fail before enforcement and pass afterward.
Fixtures cover default/nonpositive unlimited values, unrelated owners, ordinary
and constructor limits, config-to-scan propagation, exact boundary behavior,
policy replacement/reset and session preservation. The full checks initially
caught the new optional block missing from session preservation; that plumbing
was corrected without weakening the reflective regression.

The default seven-corpus result stays at **10,371 findings**, **8,151 files** and
**179,808 filtered checked calls**, with **zero additions or removals per entry**.
No project corpus configs were added. Reports and failure/audit evidence are in
target/resolution/optional-caller-limits. Earlier unconditional limit removal is
superseded by this optional policy; the unlimited default is retained.

Checkpoint 28 validation: full `make check` passes (build, vet, short race tests,
pinned lint and vulnerability check), as do the full short suite and whitespace
checks. The new session-restoration test initially mutated an indexing session;
its restoration assertions now use an idle session and pass under the race
detector. No CFML runtime execution or published CI result is claimed.

## Checkpoint 29: omitted caller arguments and collection element contracts

An unknown receiver's same-named ordinary call previously vetoed argument
inference before argument presence was checked. It now cannot veto a type for an
argument it demonstrably omits. This requires an exact call byte identity and
balanced argument syntax. Calls with unknown supplied values, opaque
`argumentCollection`, unplaced `invoke` owners or missing spans still withhold
inference. Other owner/value and optional-limit policies remain intact. The
assumption is that only callers supplying the target argument contribute its
component type, consistent with existing handling of known-owner omissions.
This does not infer a type for an omitted argument's default or prove runtime
argument validity.

The change drops the seven-corpus total **10,371 → 10,235**: **136 findings
removed, zero added** (also 136 removed and zero added ignoring reason changes).
Wheels drops **2,233 → 2,184** (47 AnalysisService printer calls, two vendored
UnitRunner calls); Masa drops **3,810 → 3,723** (82 contentRendererUtility calls,
three contentManager calls, two pluginManager calls). The five other corpora
remain identical per diagnostic. All **8,151 files** and **179,808 filtered
checked calls** are retained. Resolved calls increase by 136; the same checked
population is retained. These counts do not prove raw parser extraction equality.
No project configuration was added.

An intermediate scan exposed seven TestBox reason replacements: a factory's
runtime `#arguments.bundlePath#` was mistaken for a literal component name when
caller inference could proceed. A factory-return regression reproduced that
false inference. Caller-derived component names containing interpolation markers
now remain unknown. The final scan has no TestBox replacements or additions;
there is no guessed bundle component.

Also implemented literal `entityLoad` array element typing and native array
`.each()` first-parameter typing from an independently established collection
contract. Accepted load forms are a literal registered persistent entity name,
optionally a literal filter struct and explicit `false` unique flag. The source
contract follows [Lucee's entityLoad documentation](https://docs.lucee.org/reference/functions/entityload.html).
Dynamic names, missing/nonpersistent entities, local shadows, scalar-ID overloads,
true/unknown unique flags and other options do not establish a collection. Native
`each` requires the exact first positional closure and a stable first parameter;
index parameters, unknown collections and callback parameter assignments remain
untyped. Existing ORM relationship and cborm collection element contracts can
also feed it. The homogeneous producer contract is assumed to hold while the
collection is consumed; arbitrary collection mutations/escapes are not proven.
This collection batch supplied **zero diagnostic changes** in the seven corpora
and is not credited for the 136-call improvement.

The collection, omitted-caller and runtime-name positive regressions each fail
without their fixes. Rejection fixtures cover the unsafe alternatives described
above and preserve missing-method checks. An intermediate full corpus repeat was
identical; the final scan with the runtime-name safeguard was audited against
checkpoint 28 by file, line, function and reason. Logs, per-entry additions/removals
and source audits are under `target/resolution/collection-callbacks/` and
`target/resolution/omitted-callers-final/`; before-fix caller/runtime-name logs
are under `target/resolution/omitted-argument-callers/`.

Validation: full `make check` passes (build, vet, short race suite, format,
pinned lint and vulnerability check), as do the separate full short suite and
whitespace check. No new CFML runtime observation or published CI result is
claimed. Remaining large opportunities need mutable instance/iterator contracts,
complete return-path proofs or supported native API models; ignoring unknown
supplied callers or guessing runtime component names is not a next step.

## Checkpoint 30: chain origins and Lucee engine source context

Implemented the two concrete findings from the causal review. Mura's heterogeneous
bean/array returns remain unknown; no blanket fluent-loadBy rule was added.

Chained continuation calls now capture their byte spans before consuming each
argument list, with the real hop line, then emit in the existing order after
nested arguments. Source offsets remain exact in script, tag and wrapped tag
expressions, including repeated same-named hops and distinct argument lists.
This enables the existing caller omission proof to ignore a known-empty chain
call without ignoring unknown supplied values. Synthetic origins are not guessed.
The source-backed parser and argument-inference regressions fail without the fix.

A recognized Lucee source layout now supplies the assumed `/admin` namespace:
`core/src/main/cfml/context/admin`. Recognition requires the LuceeTestCase resource
and both admin helper sources (services.update.functions.cfm and ext.functions.cfm).
The nearest recognized layout owns the namespace; unrelated/missing-marker layouts
do not acquire it. Common bootstrap policy respects explicit config mappings,
Application mappings including dynamic declarations, and server.json/.cfconfig.json
boundaries. The assumption is that these source helpers run under their engine
namespace; it is not proof of arbitrary deployed Lucee mappings. Set an explicit
`mappings` entry for `/admin` to use a different deployment path. Regression fixtures
cover missing markers, config/source/dynamic overrides and server configuration.
The include integration regression fails when the assumption is disabled.

The seven-corpus total falls **10,235 → 10,136**. Lucee drops **1,417 → 1,318**:
58 LDEV6068 and 41 LDEV6069 helper calls resolve through the existing include graph.
All other corpus totals are unchanged. Comparing file/line/function/reason gives
**391 removals and 292 additions**. The 292 pairs are location corrections, not
new failures: 81 Wheels, 28 Masa, 48 ContentBox, 123 ColdBox and 12 TestBox entries
move to their actual chain hop. Ignoring location while retaining file, caller,
receiver, function, reason and unchecked count gives **99 removals, zero additions**.
Do not credit the moved entries as additional resolution gains.

Coverage remains **8,151 indexed/scanned files**. Resolved calls plus diagnostic
entries change **179,808 → 179,806**; Masa's resolved count falls 30,017 → 30,015.
Verbose before/after audits identify exactly two duplicate extracted getSite records,
in contentUtility.cfc:907 and userUtility.cfc:665. Each interpolated chain previously
emitted three records for its two getSite calls; both real calls remain after the
span-aware merge. A regression fails without the span fix and passes afterward.
This is an extraction correction, not skipped-source coverage. The **46 missing-base
summaries / 1,132 represented unchecked calls** remain unchanged. These reported
counts are still not an exhaustive raw parser extraction audit.

The 65-entry changeset cohort remains unresolved. A post-fix logging-only build
shows the original carch.cfc:567 save now has an exact span and is proven to omit
the argument. Inference advances to another uncertain caller, categoryManager.cfc:274
`obj.save()`, which it cannot place or prove irrelevant. This verifies removal of
one necessary blocker, not closure of the competing caller population. The 104-entry
feed cohort's bean/array ambiguity also remains. Neither is credited as a gain.

Artifacts: target/resolution/chain-spans-lucee-admin contains the final seven reports,
per-entry added/removed/source audits, moved-entry pairs and location-independent
summary, before-fix failures, verbose duplicate audit and changeset next-veto trace.
No production configuration files were added to the external corpora. Earlier
configuration-only experiment counts are now reproduced by the default scan.

Full `make check` and the separate full short suite pass. The additional duplicate
regression also passes under the focused parser/path/resolver race suite; pinned
lint and whitespace checks pass. Initial validation caught a missing import from
a linter autofix in a test; it was corrected before the successful full checks.
No new CFML runtime observation or published CI result is claimed.


## Embedded Mura admin rendering owner (checkpoint 33)

With the Mura framework context selected, otherwise-unbased CFM views/layouts
can acquire the nearest governing Application as their execution owner. Existing
module/display-object and other framework implicit bases retain priority. No
additional FW/1 preset, resolver aliases or fallback stubs are installed.

The assumption requires the standard recognized Mura namespace/layout and a
governing Application exactly under that project's `admin` directory. The admin
Application must inherit view, layout, buildURL, internalView and internalLayout
from the actual adjacent `framework.cfc`. Public view/layout bodies must call the
corresponding internal method, and each internal body must contain exactly one
live include of its path parameter, without a direct parameter reassignment.
Commented includes, including nested CFML comments, cannot establish this shape.

This assumes a view/layout in that admin application's subtree is rendered via
that embedded framework instance, with ordinary include receiving scope. It does
not reconstruct each request's computed template choice, prove runtime method
mutation absent, or support custom admin directory/base deployments. A manually
executed template or changed renderer can fall outside this convention; the
assumed Application methods then need not be available at runtime.

Different source/config `/mura` mappings, a nearer Application, missing markers,
renderer overrides, changed include paths, or missing public-to-internal dispatch
withhold the assumption. Omitting Mura from an explicit `.clif.json` frameworks
selection disables it; existing explicit implicit-base policies take priority.
An explicit configuration retains its existing framework selection behavior.

Owner/source reads and failed nearer-Application candidates are registered for
discovery invalidation; caller-context resolvers share that dependency tracker.
Owner results are cached per governing application and cleared by ordinary
resolver path invalidation. Tests cover renderer edits and creating a nearer
Application after a positive cached result.

This closes the bounded admin-owner opportunity in the earlier ten-pass research.
Custom template selection, mutable method installation and the other Mura
factory/iterator/record relationships remain separate open work. Measurements
and current coverage are in checkpoint 33 of `RESOLUTION-IMPLEMENTATION-STATUS.md`.
