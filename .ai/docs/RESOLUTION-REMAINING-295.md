# Review of the remaining 295 findings

Research-only review of the frozen combined liberal scan (`--speculative-ranked --speculative-native --speculative-methods --assume-open-world`). Every original entry is retained once below, with source context and evidence in the machine-readable audit. No resolver behavior or corpus source was changed.

The 295 are findings, not 295 proven missing methods. There are 249 call/chain findings and 46 unresolved-base summaries. The base summaries cover 1,133 inherited calls that were **not checked**; those calls are neither proven valid nor additional independently reviewed findings.

| Classification | Entries |
|---|---:|
| Existing base: package/mapping gap | 28 |
| Versioned engine contract | 26 |
| Likely stale/source API issue | 20 |
| Receiver/return relationship gap | 32 |
| Helper/dependency relationship gap | 14 |
| Engine API coverage gap | 135 |
| Unavailable base/dependency | 5 |
| Runtime/version review needed | 10 |
| Intentional missing-call test | 6 |
| Runtime callable binding gap | 6 |
| Generator placeholder | 13 |

## What this tells us

The largest opportunity is qualified engine/member contracts, followed by receiver/return refinement and dependency/mapping context. A globally available same-named method does not solve a wrong receiver. The source-issue bucket consists of review candidates, including obsolete patch/test APIs; it is not a list of confirmed production defects.

Notable source candidates: `getDisplayExpiredDate` versus `getDisplayExpireDate`, `generatePasswordResetToken` versus `generateResetToken`, CommandDSL `flag` versus `flags`, FileSystem `getCWD`, and the redundant DetailOutputService `statusInfo(...).toConsole()` chain. The last returns the service itself, which has no `toConsole` method.

The largest single base summary is LuCLI `modules.BaseModule` (764 unchecked inherited calls). Production host source is unavailable; the similarly named local test fixture cannot establish that contract. Thirteen migration templates deliberately contain `[extends]`. Six findings deliberately exercise missing calls.

Some otherwise valid contracts depend on runtime version: historical Adobe DBInfo facades, Hibernate SessionFactory implementation APIs, image extensions and Java collection access. Adobe documents removal of the CFC facade family in CF2025. See [Adobe runtime/version guidance](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html), [Hibernate 5.6 implementation contract](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/engine/spi/SessionFactoryImplementor.html), and [BoxLang module builtin](https://apidocs.ortussolutions.com/boxlang/1.0.0-beta7/ortus/boxlang/runtime/bifs/global/system/GetModuleList.html).

## Implicit getter/setter follow-up

A missing explicit function declaration is insufficient to identify a missing CFML accessor. Before classifying getX/setX as absent, inspect the exact component and available parents for property X, accessor/ORM generation, per-property getter/setter controls, explicit overrides, onMissingMethod and runtime UDF bindings. A missing base or unresolved runtime binding must stay uncertain. Property-dot dispatch (invokeImplicitAccessor) is a separate relationship from generating the getX/setX methods. See [Lucee accessor rules](https://docs.lucee.org/recipes/component-accessors.html), [Adobe generated accessors](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/__references__/generated-accessors.html), and [Adobe implicit property dispatch](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/__references__/cfc-implicit-notation.html).

| Reviewed case | Accessor evidence and conclusion |
|---|---|
| ContentTemplate.getSchema (R032) | Persistent entity declares schema and initializes it to a struct. Already classified as a generated getter/return relationship gap. |
| LogBox.getConfig (R221–R222) | accessors=true and property config. Already classified as a generated getter/import/return relationship gap. |
| ServerService.getServerInfoJSON (R233, R240) | accessors=true, but serverInfoJSON is a nested serverInfo struct field, not a component property. No generated accessor established. |
| Controller.get/setAppStartHandlerFired (R223–R225) | accessors=true, but no matching property or method in the selected component. Remains a likely stale API candidate. |
| Page.getDisplayExpiredDate (R040) | No matching property found in the inspected content hierarchy; explicit BaseContent method is getDisplayExpireDate. Remains a spelling/API candidate. |
| FileSystem.getCWD (R242) | accessors=true, but no CWD property. BaseCommand declares getCWD on a different receiver. |
| Mura form/field service getters and trash translator setter (R287–R290, R292) | Selected components/parent do not establish matching properties or accessor generation. Runtime injection/augmentation remains open; these are not confirmed missing methods. |

This accessor follow-up does not reclassify any entry or change the 295 total. clif already distinguishes generated accessors from explicit functions for script/tag components; further work concerns connecting the correct owner and getter return, plus any remaining engine-specific generation rules.

## Evidence and limits

[Committed classification summary](resolution-candidates/remaining-295-summary.json) and [committed per-project corpus comparison](resolution-candidates/method-shortcuts-summary.json) preserve the totals independently of the local ignored artifacts. The per-entry Markdown review is committed; full JSON/source caches and absolute corpus links refer to the local research environment.

- Input SHA-256: `5bd0f30b1e8c427269a92c4b64a3661fabbb2c092b7d0f695d93639487bc90a2`.
- Same coverage as the prior method-shortcut audit: 8,151 files / 189,154 count records; Wheels vendor/wheels was scanned explicitly. This review does not perform a new scan or reduce the unresolved count.
- Corpus source, supplemental source inventory, Lucee 7.1 Java source/FLD, image extension master, Commons Pool 2.12.0 and local JDK javap were examined. Public-source download hashes/URLs are in the source manifest. Mutable branches establish inspected API existence, not the deployed runtime version.
- No CFML engine tests were executed. “Engine API coverage gap” means the named qualified API exists in inspected contracts; it does not certify overload compatibility, branch execution or a complete earlier-hop relationship tree.
- Historical Adobe columns/version facade signatures, legacy query getMeta, Config.getAdminMode, Mura service injection and unavailable bases remain explicitly qualified or open.
- [Complete JSON audit](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/classified.json), [summary](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/summary.json), [source manifest](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/manifest.json), [review script](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/classify.py). Each JSON entry retains its original zero-based line, diagnostic, source context, group, confidence and next action. Markdown links use one-based lines.

## Per-entry review

### adobe-dbinfo — 26 entries

**Versioned engine contract; medium confidence.** Historical patches explicitly choose Adobe new dbinfo().columns()/version() outside the Lucee/Railo branch. This is a runtime-supplied CFC facade, not a missing project method. Adobe documents the facade family as removed in CF2025; do not assume these patches run on every engine/version. Exact columns/version facade signatures still need an installed older Adobe runtime or archived API source.

Next: Add an Adobe-version-specific facade contract after checking signatures; preserve incompatibility diagnostics on CF2025.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R002 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.0.0-beta/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.0.0-beta/Update.cfc:497) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R003 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.0.0-beta/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.0.0-beta/Update.cfc:505) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R004 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.0.0-rc/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.0.0-rc/Update.cfc:467) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R005 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.0.0-rc/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.0.0-rc/Update.cfc:475) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R006 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.0.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.0.0/Update.cfc:470) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R007 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.0.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.0.0/Update.cfc:478) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R009 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.1.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.1.0/Update.cfc:412) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R010 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.1.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.1.0/Update.cfc:420) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R012 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.5.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.5.0/Update.cfc:433) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R013 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.5.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.5.0/Update.cfc:441) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R015 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.5.1/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.5.1/Update.cfc:433) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R016 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.5.1/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.5.1/Update.cfc:441) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R018 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.6.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.6.0/Update.cfc:433) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R019 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.6.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.6.0/Update.cfc:441) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R020 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.7.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.7.0/Update.cfc:475) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R021 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.7.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.7.0/Update.cfc:483) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R022 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.7.1/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.7.1/Update.cfc:475) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R023 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.7.1/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.7.1/Update.cfc:483) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R024 | contentbox: [Ortus-Solutions_ContentBox/build/patches/4.0.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/4.0.0/Update.cfc:490) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R025 | contentbox: [Ortus-Solutions_ContentBox/build/patches/4.0.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/4.0.0/Update.cfc:498) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R026 | contentbox: [Ortus-Solutions_ContentBox/build/patches/4.1.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/4.1.0/Update.cfc:490) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R027 | contentbox: [Ortus-Solutions_ContentBox/build/patches/4.1.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/4.1.0/Update.cfc:498) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R028 | contentbox: [Ortus-Solutions_ContentBox/build/patches/4.2.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/4.2.0/Update.cfc:490) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R029 | contentbox: [Ortus-Solutions_ContentBox/build/patches/4.2.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/4.2.0/Update.cfc:498) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R030 | contentbox: [Ortus-Solutions_ContentBox/build/patches/4.2.1/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/4.2.1/Update.cfc:490) | `method 'columns' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |
| R031 | contentbox: [Ortus-Solutions_ContentBox/build/patches/4.2.1/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/4.2.1/Update.cfc:498) | `method 'version' not found in dbinfo` | [official API](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/script-functions-implemented-as-cfcs.html) |

### legacy-coldbox-patches — 4 entries

**Likely stale/source API issue; medium confidence.** Old migration patches call Controller.setNextEvent(). The selected current Controller has no such API; historical FrameworkSupertype fixtures do. This is a framework-version mismatch candidate, not proof that an old patch was broken on its original runtime.

Next: Match each patch to its intended ColdBox version before changing calls or adding versioned contracts.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R008 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.1.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.1.0/Update.cfc:94) | `method 'setNextEvent' not found in coldbox.system.web.Controller` | Call-site context / declaration inventory (see JSON). |
| R011 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.5.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.5.0/Update.cfc:158) | `method 'setNextEvent' not found in coldbox.system.web.Controller` | Call-site context / declaration inventory (see JSON). |
| R014 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.5.1/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.5.1/Update.cfc:158) | `method 'setNextEvent' not found in coldbox.system.web.Controller` | Call-site context / declaration inventory (see JSON). |
| R017 | contentbox: [Ortus-Solutions_ContentBox/build/patches/3.6.0/Update.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.6.0/Update.cfc:158) | `method 'setNextEvent' not found in coldbox.system.web.Controller` | Call-site context / declaration inventory (see JSON). |

### template-schema — 1 entries

**Receiver/return relationship gap; high confidence.** schema comes from newTemplate.getSchema(), whose ContentTemplate property is initialized to a struct. Selecting qb.SchemaBuilder for this local variable is incorrect; keyArray().filter() should follow the struct return.

Next: Trace generated getter returns through the ORM new() result instead of selecting a same-named service.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R032 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/models/content/ContentTemplateService.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/models/content/ContentTemplateService.cfc:177) | `method 'keyArray' not found in qb.models.Schema.SchemaBuilder (chain to 'filter')` | [ContentTemplate.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/models/content/ContentTemplate.cfc:133) |

### cbfs-helper — 5 entries

**Helper/dependency relationship gap; high confidence.** Supplemental cbfs helpers/Mixins.cfm defines cbfs(): without a name it returns DiskService@cbfs. Calls need helper inclusion and dependency return contracts, not global name matching.

Next: Connect registered module helpers and follow the argument-dependent return.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R034 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/handlers/sites.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/handlers/sites.cfc:67) | `chained on 'cbfs', which is not found (calling 'filter')` | [Mixins.cfm](/Users/garethedwards/corpus/coldbox-modules_cbfs/helpers/Mixins.cfm:7) |
| R035 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/handlers/sites.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/handlers/sites.cfc:67) | `chained on 'cbfs', which is not found (calling 'getDisks')` | [Mixins.cfm](/Users/garethedwards/corpus/coldbox-modules_cbfs/helpers/Mixins.cfm:7) |
| R036 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/handlers/sites.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/handlers/sites.cfc:67) | `chained on 'cbfs', which is not found (calling 'keyArray')` | [Mixins.cfm](/Users/garethedwards/corpus/coldbox-modules_cbfs/helpers/Mixins.cfm:7) |
| R038 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Editor.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Editor.cfc:11) | `chained on 'cbfs', which is not found (calling 'get')` | [Mixins.cfm](/Users/garethedwards/corpus/coldbox-modules_cbfs/helpers/Mixins.cfm:7) |
| R039 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Home.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Home.cfc:33) | `chained on 'cbfs', which is not found (calling 'get')` | [Mixins.cfm](/Users/garethedwards/corpus/coldbox-modules_cbfs/helpers/Mixins.cfm:7) |

### cache-helper — 1 entries

**Helper/dependency relationship gap; high confidence.** The included administrative mixin calls a framework getCache() helper. FrameworkSupertype declares that helper; the include owner/provider context is missing.

Next: Carry the framework owner into included helpers and follow the cache-provider return.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R037 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/helpers/Mixins.cfm](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/helpers/Mixins.cfm:33) | `chained on 'getCache', which is not found (calling 'getOrSet')` | [FrameworkSupertype.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/FrameworkSupertype.cfc:299) |

### expiry-spelling — 1 entries

**Likely stale/source API issue; high confidence.** The view requests getDisplayExpiredDate; BaseContent defines getDisplayExpireDate (without the extra d). Page inherits this API. Strong spelling mismatch in the scanned source.

Next: Review/fix the corpus call, rather than inventing an alias.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R040 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/views/content/quickLook.cfm](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/views/content/quickLook.cfm:15) | `method 'getDisplayExpiredDate' not found in Page` | [BaseContent.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/models/content/BaseContent.cfc:1498) |

### reset-token-spelling — 2 entries

**Likely stale/source API issue; high confidence.** Tests request generatePasswordResetToken; SecurityService declares generateResetToken. No matching requested API was found in the indexed or supplemental declarations.

Next: Check the tests against the current SecurityService API.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R041 | contentbox: [Ortus-Solutions_ContentBox/tests/specs/contentbox-api/authSpec.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/tests/specs/contentbox-api/authSpec.cfc:224) | `method 'generatePasswordResetToken' not found in securityService@contentbox` | [SecurityService.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/models/security/SecurityService.cfc:221) |
| R042 | contentbox: [Ortus-Solutions_ContentBox/tests/specs/contentbox-api/authSpec.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/tests/specs/contentbox-api/authSpec.cfc:259) | `method 'generatePasswordResetToken' not found in securityService@contentbox` | [SecurityService.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/models/security/SecurityService.cfc:221) |

### negative-tests — 6 entries

**Intentional missing-call test; high confidence.** These fixtures deliberately exercise nonexistent calls: invalid array/date tokey members, Elvis fallback for susi().sorglos(), missing super method, missing Date method, and getValuesss error handling. Their absence is the tested behavior.

Next: Keep diagnostics visible; optionally label deliberate negative fixtures without claiming a definition exists.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R165 | lucee: [lucee_Lucee/test/tickets/LDEV3167.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3167.cfc:31) | `method 'tokey' not found on builtin native.array` | Call-site context / declaration inventory (see JSON). |
| R166 | lucee: [lucee_Lucee/test/tickets/LDEV3167.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3167.cfc:45) | `method 'tokey' not found on builtin native.date` | Call-site context / declaration inventory (see JSON). |
| R180 | lucee: [lucee_Lucee/test/tickets/LDEV4826.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4826.cfc:54) | `chained on 'susi', which is not found (calling 'sorglos')` | Call-site context / declaration inventory (see JSON). |
| R195 | lucee: [lucee_Lucee/test/tickets/LDEV6055/Child.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV6055/Child.cfc:4) | `not found in parent component` | Call-site context / declaration inventory (see JSON). |
| R197 | lucee: [lucee_Lucee/test/tickets/LDEV6349.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV6349.cfc:41) | `method 'nonExistentMethod_ldev6349' not found on builtin native.date` | Call-site context / declaration inventory (see JSON). |
| R216 | coldbox: [ColdBox_coldbox-platform/test-harness/handlers/testerror.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/test-harness/handlers/testerror.cfc:32) | `method 'getValuesss' not found in coldbox.system.web.context.RequestContext` | Call-site context / declaration inventory (see JSON). |

### ordered-struct — 5 entries

**Receiver/return relationship gap; medium confidence.** Bracket key:value literals in structinsert/LDEV4725 are ordered structs, not arrays. The admin mappings result is also consumed as a keyed struct. The array receiver classification loses these struct members.

Next: Distinguish ordered struct literals and the admin mappings result shape; then use struct member contracts.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R048 | lucee: [lucee_Lucee/core/src/main/cfml/context/admin/web.cfm](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/web.cfm:303) | `method 'keyList' not found on builtin native.array` | [structinsert.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structinsert.cfc:4); [LDEV4725.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:6) |
| R100 | lucee: [lucee_Lucee/test/functions/structinsert.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structinsert.cfc:15) | `method 'insert' not found on builtin native.array` | [structinsert.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structinsert.cfc:4); [LDEV4725.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:6) |
| R101 | lucee: [lucee_Lucee/test/functions/structinsert.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structinsert.cfc:22) | `method 'insert' not found on builtin native.array` | [structinsert.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structinsert.cfc:4); [LDEV4725.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:6) |
| R178 | lucee: [lucee_Lucee/test/tickets/LDEV4725.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:14) | `method 'KeyList' not found on builtin native.array` | [structinsert.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structinsert.cfc:4); [LDEV4725.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:6) |
| R179 | lucee: [lucee_Lucee/test/tickets/LDEV4725.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:26) | `method 'KeyArray' not found on builtin native.array` | [structinsert.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structinsert.cfc:4); [LDEV4725.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:6) |

### member-chaining — 8 entries

**Receiver/return relationship gap; high confidence.** Array sort/deleteAt/clear/append member forms return the receiver for chaining. Using the standalone function boolean return produces false boolean/unknown receivers downstream.

Next: Apply member-chaining metadata separately from global function returns.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R049 | lucee: [lucee_Lucee/core/src/main/cfml/context/doc/objects.cfm](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/doc/objects.cfm:9) | `method 'findNoCase' not found on builtin native.boolean` | Call-site context / declaration inventory (see JSON). |
| R050 | lucee: [lucee_Lucee/core/src/main/cfml/context/doc/objects.cfm](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/doc/objects.cfm:14) | `method 'len' not found on builtin native.boolean` | Call-site context / declaration inventory (see JSON). |
| R090 | lucee: [lucee_Lucee/test/functions/functionHasTests.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/functionHasTests.cfc:20) | `method 'toList' not found on builtin native.boolean` | Call-site context / declaration inventory (see JSON). |
| R160 | lucee: [lucee_Lucee/test/tickets/LDEV1844.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1844.cfc:6) | `method 'insertAt' not found on builtin native.boolean` | Call-site context / declaration inventory (see JSON). |
| R161 | lucee: [lucee_Lucee/test/tickets/LDEV1844.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1844.cfc:10) | `method 'Resize' not found on builtin native.boolean` | Call-site context / declaration inventory (see JSON). |
| R185 | lucee: [lucee_Lucee/test/tickets/LDEV5578/LDEV5578.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5578/LDEV5578.cfc:56) | `chained on 'append', which is not found (calling 'map')` | Call-site context / declaration inventory (see JSON). |
| R186 | lucee: [lucee_Lucee/test/tickets/LDEV5578/LDEV5578.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5578/LDEV5578.cfc:68) | `chained on 'append', which is not found (calling 'map')` | Call-site context / declaration inventory (see JSON). |
| R187 | lucee: [lucee_Lucee/test/tickets/LDEV5578/LDEV5578.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5578/LDEV5578.cfc:80) | `chained on 'append', which is not found (calling 'map')` | Call-site context / declaration inventory (see JSON). |

### query-legacy — 3 entries

**Runtime/version review needed; medium confidence.** The zero-argument getColumnList call is inside xit. The getMeta().getcolumnlabels() fixture is an underscore-prefixed historical suite that expects success, but Lucee 7.1 QueryImpl has no getMeta method. These need the intended engine version; neither is a proven current API gap or deliberate error assertion.

Next: Identify the fixture runtime/version and legacy query API before resolving or suppressing.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R127 | lucee: [lucee_Lucee/test/tickets/LDEV1160.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1160.cfc:10) | `method 'getColumnList' not found on builtin querynew` | Call-site context / declaration inventory (see JSON). |
| R130 | lucee: [lucee_Lucee/test/tickets/LDEV1289/test.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1289/test.cfm:4) | `method 'getMeta' not found on builtin querynew` | Call-site context / declaration inventory (see JSON). |
| R131 | lucee: [lucee_Lucee/test/tickets/LDEV1289/test.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1289/test.cfm:4) | `method 'getMeta' not found on builtin querynew (chain to 'getcolumnlabels')` | Call-site context / declaration inventory (see JSON). |

### config-admin-mode — 1 entries

**Runtime/version review needed; medium confidence.** The test Administrator calls Config.getAdminMode; the fetched Lucee 7.1 Config interfaces and implementations do not declare it. The snapshot/runtime version relationship is unresolved.

Next: Check the corresponding older Config contract and actual runtime configuration.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R052 | lucee: [lucee_Lucee/test/components/Administrator.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/components/Administrator.cfc:1681) | `method 'getAdminMode' not found on builtin native.lucee.config` | Call-site context / declaration inventory (see JSON). |

### coercion — 4 entries

**Receiver/return relationship gap; medium confidence.** Tests expect numeric Len, numeric-string round, and numeric booleanFormat. Registered string/numeric/boolean member contracts alone do not express CFML value coercion. These are engine conversion paths, not missing user functions.

Next: Model supported engine coercions with value/argument constraints; verify against an installed runtime.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R167 | lucee: [lucee_Lucee/test/tickets/LDEV332.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV332.cfc:26) | `method 'Len' not found on builtin native.numeric` | Call-site context / declaration inventory (see JSON). |
| R169 | lucee: [lucee_Lucee/test/tickets/LDEV332.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV332.cfc:69) | `method 'round' not found on builtin native.string` | Call-site context / declaration inventory (see JSON). |
| R170 | lucee: [lucee_Lucee/test/tickets/LDEV332.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV332.cfc:74) | `method 'round' not found on builtin native.string` | Call-site context / declaration inventory (see JSON). |
| R177 | lucee: [lucee_Lucee/test/tickets/LDEV4589/LDEV4589.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4589/LDEV4589.cfm:6) | `method 'booleanFormat' not found on builtin native.numeric` | Call-site context / declaration inventory (see JSON). |

### session-mixin — 2 entries

**Runtime callable binding gap; high confidence.** Scene 1 assigns tmp to session.user.mixin before invoking it; scene 2 reuses the session object. A User source declaration cannot enumerate this runtime-added UDF.

Next: Connect literal-key callable assignment and session lifetime across requests.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R171 | lucee: [lucee_Lucee/test/tickets/LDEV3473/test.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3473/test.cfm:7) | `method 'mixin' not found in User` | Call-site context / declaration inventory (see JSON). |
| R172 | lucee: [lucee_Lucee/test/tickets/LDEV3473/test.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3473/test.cfm:11) | `method 'mixin' not found in User` | Call-site context / declaration inventory (see JSON). |

### boxlang-cache — 3 entries

**Engine API coverage gap; high confidence.** BoxLangProvider invokes cache().clearStats() and cache().get/getQuiet(...).getOrDefault(). Official BoxLang contracts identify the provider and Attempt return; getOrDefault belongs to Attempt, not the provider itself.

Next: Pin the BoxLang provider API and add a qualified runtime contract.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R199 | coldbox: [ColdBox_coldbox-platform/system/cache/providers/BoxLangProvider.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/providers/BoxLangProvider.cfc:212) | `chained on 'cache', which is not found (calling 'clearStats')` | [official API](https://s3.amazonaws.com/apidocs.ortussolutions.com/boxlang/1.0.0/index-all.html); [official API](https://boxlang.ortusbooks.com/boxlang-framework/caching); [official API](https://apidocs.ortussolutions.com/boxlang/1.0.0-beta7/ortus/boxlang/runtime/dynamic/Attempt.html) |
| R200 | coldbox: [ColdBox_coldbox-platform/system/cache/providers/BoxLangProvider.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/providers/BoxLangProvider.cfc:270) | `chained on 'cache', which is not found (calling 'getOrDefault')` | [official API](https://s3.amazonaws.com/apidocs.ortussolutions.com/boxlang/1.0.0/index-all.html); [official API](https://boxlang.ortusbooks.com/boxlang-framework/caching); [official API](https://apidocs.ortussolutions.com/boxlang/1.0.0-beta7/ortus/boxlang/runtime/dynamic/Attempt.html) |
| R201 | coldbox: [ColdBox_coldbox-platform/system/cache/providers/BoxLangProvider.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/providers/BoxLangProvider.cfc:279) | `chained on 'cache', which is not found (calling 'getOrDefault')` | [official API](https://s3.amazonaws.com/apidocs.ortussolutions.com/boxlang/1.0.0/index-all.html); [official API](https://boxlang.ortusbooks.com/boxlang-framework/caching); [official API](https://apidocs.ortussolutions.com/boxlang/1.0.0-beta7/ortus/boxlang/runtime/dynamic/Attempt.html) |

### concurrent-map — 1 entries

**Receiver/return relationship gap; high confidence.** ConcurrentStore creates java.util.concurrent.ConcurrentHashMap, which exposes keys(). A generic java.util.Map contract omits this concrete-class method.

Next: Preserve the concrete Java constructor type instead of reducing it to Map.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R202 | coldbox: [ColdBox_coldbox-platform/system/cache/store/ConcurrentStore.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/store/ConcurrentStore.cfc:69) | `method 'keys' not found on builtin native.map` | [java.util.concurrent.ConcurrentHashMap.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/java.util.concurrent.ConcurrentHashMap.javap.txt) |

### struct-scope-filter — 4 entries

**Receiver/return relationship gap; high confidence.** this.filter operates on the CFC public scope as a struct; getFlash is explicitly struct-returning. The collection members need scope/return contracts rather than AbstractFlashScope method lookup.

Next: Follow declared struct return and CFC-scope collection behavior.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R206 | coldbox: [ColdBox_coldbox-platform/system/ioc/Scopes.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/ioc/Scopes.cfc:36) | `chained on 'filter', which is not found (calling 'keyArray')` | [AbstractFlashScope.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:85) |
| R208 | coldbox: [ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:115) | `method 'filter' not found in coldbox.system.web.flash.AbstractFlashScope (chain to 'keyArray')` | [AbstractFlashScope.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:85) |
| R209 | coldbox: [ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:116) | `method 'filter' not found in coldbox.system.web.flash.AbstractFlashScope (chain to 'each')` | [AbstractFlashScope.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:85) |
| R210 | coldbox: [ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:144) | `method 'filter' not found in coldbox.system.web.flash.AbstractFlashScope (chain to 'each')` | [AbstractFlashScope.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:85) |

### expect-dsl — 1 entries

**Helper/dependency relationship gap; medium confidence.** expect(...).toInclude(...) is an assertion helper chain in CustomMatchers. Nested argument/lexical DSL context prevents following the helper return; this is not evidence that toInclude is absent.

Next: Trace the assertion helper owner and nested call boundaries.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R207 | coldbox: [ColdBox_coldbox-platform/system/testing/CustomMatchers.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/testing/CustomMatchers.cfc:89) | `chained on 'expect', which is not found (calling 'toInclude')` | Call-site context / declaration inventory (see JSON). |

### boxlang-module-list — 2 entries

**Engine API coverage gap; high confidence.** getModuleList is a BoxLang builtin returning IStruct, inside a BoxLang-only guard. It is not a ColdBox Router method.

Next: Add a BoxLang builtin and its struct return, qualified by runtime.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R211 | coldbox: [ColdBox_coldbox-platform/system/web/routing/Router.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/routing/Router.cfc:2102) | `chained on 'getModuleList', which is not found (calling 'findNoCase')` | [official API](https://apidocs.ortussolutions.com/boxlang/1.0.0-beta7/ortus/boxlang/runtime/bifs/global/system/GetModuleList.html) |
| R212 | coldbox: [ColdBox_coldbox-platform/system/web/routing/Router.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/routing/Router.cfc:2102) | `chained on 'getModuleList', which is not found (calling 'keyArray')` | [official API](https://apidocs.ortussolutions.com/boxlang/1.0.0-beta7/ortus/boxlang/runtime/bifs/global/system/GetModuleList.html) |

### event-suffix — 1 entries

**Runtime callable binding gap; high confidence.** EVENT_CACHE_SUFFIX is a field and is called only after isCustomFunction(field) succeeds. Default EventHandler has a string value; applications may replace it with a UDF. A mandatory method declaration is the wrong requirement.

Next: Refine callable fields under isCustomFunction guards; do not fabricate a method.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R213 | coldbox: [ColdBox_coldbox-platform/system/web/services/HandlerService.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/services/HandlerService.cfc:818) | `method 'EVENT_CACHE_SUFFIX' not found in EventHandler` | Call-site context / declaration inventory (see JSON). |

### legacy-cache-parse — 1 entries

**Likely stale/source API issue; medium confidence.** LoaderService still calls parseAndLoad on CacheBoxConfig in the legacy XML configuration branch; the selected class supplies no such method.

Next: Review support for this legacy branch or the intended historical API.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R214 | coldbox: [ColdBox_coldbox-platform/system/web/services/LoaderService.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/services/LoaderService.cfc:233) | `method 'parseAndLoad' not found in coldbox.system.cache.config.CacheBoxConfig` | Call-site context / declaration inventory (see JSON). |

### boxlang-mcp — 1 entries

**Engine API coverage gap; high confidence.** The guarded MCPServer(...).registerTool(...) chain belongs to the bx-ai runtime module. Official module source documents this fluent API.

Next: Represent the module factory return and registerTool contract only with bx-ai present.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R215 | coldbox: [ColdBox_coldbox-platform/test-harness/config/Router.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/test-harness/config/Router.cfc:11) | `chained on 'MCPServer', which is not found (calling 'registerTool')` | [official API](https://github.com/ortus-boxlang/bx-ai) |

### command-params — 2 entries

**Helper/dependency relationship gap; high confidence.** CommandBox BaseCommand.command() returns CommandDSL, which declares params(). The helper/dependency connection is missing.

Next: Connect the CommandBox host helper and its factory return.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R218 | coldbox: [ColdBox_coldbox-platform/tests/perf-harness/PerformanceSuite.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/perf-harness/PerformanceSuite.cfc:281) | `chained on 'command', which is not found (calling 'params')` | [BaseCommand.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/BaseCommand.cfc:151); [CommandDSL.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/util/CommandDSL.cfc:65) |
| R220 | coldbox: [ColdBox_coldbox-platform/tests/perf-harness/PerformanceSuite.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/perf-harness/PerformanceSuite.cfc:292) | `chained on 'command', which is not found (calling 'params')` | [BaseCommand.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/BaseCommand.cfc:151); [CommandDSL.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/util/CommandDSL.cfc:65) |

### command-flag — 2 entries

**Likely stale/source API issue; medium confidence.** After following command() to CommandDSL, the source defines flags(), not flag(), and no onMissingMethod handler supplies that singular spelling. The missing helper connection hides an additional API mismatch.

Next: Review singular flag calls against the CommandBox dependency version.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R217 | coldbox: [ColdBox_coldbox-platform/tests/perf-harness/PerformanceSuite.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/perf-harness/PerformanceSuite.cfc:281) | `chained on 'command', which is not found (calling 'flag')` | [CommandDSL.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/util/CommandDSL.cfc:142) |
| R219 | coldbox: [ColdBox_coldbox-platform/tests/perf-harness/PerformanceSuite.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/perf-harness/PerformanceSuite.cfc:292) | `chained on 'command', which is not found (calling 'flag')` | [CommandDSL.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/util/CommandDSL.cfc:142) |

### logbox-getter — 2 entries

**Receiver/return relationship gap; high confidence.** LogBox declares accessors=true and a config property. getConfig is generated; the imported short constructor name and getter return need to connect to the correct LogBox class.

Next: Connect wildcard imports and generated accessor returns.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R221 | coldbox: [ColdBox_coldbox-platform/tests/specs/logging/LogBoxBuildTests.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/logging/LogBoxBuildTests.cfc:35) | `method 'getConfig' not found in LogBox (chain to 'keyArray')` | [LogBox.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/logging/LogBox.cfc:42) |
| R222 | coldbox: [ColdBox_coldbox-platform/tests/specs/logging/LogBoxBuildTests.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/logging/LogBoxBuildTests.cfc:56) | `method 'getConfig' not found in LogBox (chain to 'keyArray')` | [LogBox.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/logging/LogBox.cfc:42) |

### controller-test-state — 3 entries

**Likely stale/source API issue; medium confidence.** Tests call get/setAppStartHandlerFired but the current Controller has no matching method/property. This looks like a stale state-accessor test contract.

Next: Review tests against the current lifecycle API.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R223 | coldbox: [ColdBox_coldbox-platform/tests/specs/web/ControllerTest.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/web/ControllerTest.cfc:75) | `method 'getAppStartHandlerFired' not found in coldbox.system.web.Controller` | Call-site context / declaration inventory (see JSON). |
| R224 | coldbox: [ColdBox_coldbox-platform/tests/specs/web/ControllerTest.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/web/ControllerTest.cfc:76) | `method 'setAppStartHandlerFired' not found in coldbox.system.web.Controller` | Call-site context / declaration inventory (see JSON). |
| R225 | coldbox: [ColdBox_coldbox-platform/tests/specs/web/ControllerTest.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/web/ControllerTest.cfc:77) | `method 'getAppStartHandlerFired' not found in coldbox.system.web.Controller` | Call-site context / declaration inventory (see JSON). |

### debug-panel — 1 entries

**Likely stale/source API issue; medium confidence.** The old test layout calls RequestContext.showdebugpanel; no matching API/property was found on the selected component.

Next: Confirm historical version or correct/remove the stale layout call.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R226 | coldbox: [ColdBox_coldbox-platform/tests/suites/eventCachingCollisions/layouts/Layout.None.cfm](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/suites/eventCachingCollisions/layouts/Layout.None.cfm:2) | `method 'showdebugpanel' not found in coldbox.system.web.context.RequestContext` | Call-site context / declaration inventory (see JSON). |

### server-json — 2 entries

**Likely stale/source API issue; medium confidence.** CommandBox ServerService has accessors=true, but serverInfoJSON is a field on the returned serverInfo struct, not a declared component property. Neither a generated getServerInfoJSON accessor nor an explicit getter is established by this source. Do not infer a component accessor from a nested struct field.

Next: Check dependency version and supported server-info lookup API.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R233 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/benchmark.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/benchmark.cfc:156) | `method 'getServerInfoJSON' not found in ServerService` | [ServerService.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/services/ServerService.cfc:11); [ServerService.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/services/ServerService.cfc:1480) |
| R240 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/profile.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/profile.cfc:166) | `method 'getServerInfoJSON' not found in ServerService` | [ServerService.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/services/ServerService.cfc:11); [ServerService.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/services/ServerService.cfc:1480) |

### detail-output — 1 entries

**Likely stale/source API issue; high confidence.** DetailOutputService.statusInfo already prints and returns this, not PrintBuffer. Chaining .toConsole() therefore targets DetailOutputService, which has no such method.

Next: Review/remove the extra toConsole chain in the corpus.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R237 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/docker/push.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/docker/push.cfc:393) | `method 'toConsole' not found in DetailOutputService` | [DetailOutputService.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/DetailOutputService.cfc:200) |

### filesystem-cwd — 1 entries

**Likely stale/source API issue; medium confidence.** fileSystem is the CommandBox FileSystem service, which lacks getCWD; getCWD belongs to BaseCommand and pwd to Shell. A globally found method would connect the wrong receiver.

Next: Use the appropriate host/service API after confirming dependency version.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R242 | wheels: [cfwheels_cfwheels/cli/src/models/MCPService.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/MCPService.cfc:28) | `method 'getCWD' not found in fileSystem` | [BaseCommand.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/BaseCommand.cfc:86) |

### shell-dependency — 5 entries

**Helper/dependency relationship gap; high confidence.** The injected shell has an exact pwd definition in supplemental CommandBox Shell.cfc. The target dependency/DI alias is not connected in these scans.

Next: Connect the shell injection to its actual dependency declaration.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R243 | wheels: [cfwheels_cfwheels/cli/src/models/MCPService.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/MCPService.cfc:164) | `method 'pwd' not found in shell` | [Shell.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/Shell.cfc:498) |
| R244 | wheels: [cfwheels_cfwheels/cli/src/models/MCPService.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/MCPService.cfc:237) | `method 'pwd' not found in shell` | [Shell.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/Shell.cfc:498) |
| R245 | wheels: [cfwheels_cfwheels/cli/src/models/MCPService.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/MCPService.cfc:320) | `method 'pwd' not found in shell` | [Shell.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/Shell.cfc:498) |
| R246 | wheels: [cfwheels_cfwheels/cli/src/models/helpers.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/helpers.cfc:30) | `method 'pwd' not found in shell` | [Shell.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/Shell.cfc:498) |
| R247 | wheels: [cfwheels_cfwheels/cli/src/models/helpers.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/helpers.cfc:53) | `method 'pwd' not found in shell` | [Shell.cfc](/Users/garethedwards/corpus/Ortus-Solutions_commandbox/src/cfml/system/Shell.cfc:498) |

### boxlang-execute — 1 entries

**Engine API coverage gap; high confidence.** The guarded BoxLang branch invokes getBoxRuntime().executeStatement(expression). Official BoxLang APIs declare GetBoxRuntime and BoxRuntime.executeStatement(String); the host builtin-to-Java return connection is missing.

Next: Connect the BoxLang builtin to BoxRuntime and its qualified execution API.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R262 | wheels: [cfwheels_cfwheels/vendor/wheels/Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/Test.cfc:772) | `chained on 'getBoxRuntime', which is not found (calling 'executeStatement')` | [official API](https://s3.amazonaws.com/apidocs.ortussolutions.com/boxlang/1.0.0/ortus/boxlang/runtime/bifs/global/system/GetBoxRuntime.html); [official API](https://s3.amazonaws.com/apidocs.ortussolutions.com/boxlang/1.0.0/index-all.html) |

### runtime-injected-tests — 3 entries

**Runtime callable binding gap; high confidence.** Fixtures inject memoProbeInjected, generate/include a CFML file defining fxReinclude, and register virtualReturn through MockBox. These methods are created at runtime; absence from static declarations is expected.

Next: Connect literal UDF promotion, generated include content and MockBox registration; keep dynamic evidence provenance.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R278 | wheels: [cfwheels_cfwheels/vendor/wheels/tests/specs/global/promoteIncludedGlobalsMemoSpec.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/tests/specs/global/promoteIncludedGlobalsMemoSpec.cfc:172) | `method 'memoProbeInjected' not found in wheels.tests._assets.global.PromoteMemoFixture` | Call-site context / declaration inventory (see JSON). |
| R279 | wheels: [cfwheels_cfwheels/vendor/wheels/tests/specs/global/reloadGlobalsSpec.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/tests/specs/global/reloadGlobalsSpec.cfc:110) | `method 'fxReinclude' not found in Global` | Call-site context / declaration inventory (see JSON). |
| R295 | testbox: [Ortus-Solutions_TestBox/tests/specs/mockbox/MockBoxTestHarness.cfm](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/tests/specs/mockbox/MockBoxTestHarness.cfm:33) | `method 'virtualReturn' not found in testbox.tests.resources.Test` | Call-site context / declaration inventory (see JSON). |

### guarded-java-date — 6 entries

**Receiver/return relationship gap; medium confidence.** isInstanceOf guards explicitly select java.time.LocalDateTime or oracle.sql.TIMESTAMP. The resolver retains native.date across these branches, losing the concrete Java API.

Next: Refine receiver types inside positive instance guards; verify Oracle dependency separately.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R280 | wheels: [cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) | `method 'getDayOfMonth' not found on builtin native.date` | [java.time.LocalDateTime.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/java.time.LocalDateTime.javap.txt) |
| R281 | wheels: [cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) | `method 'getHour' not found on builtin native.date` | [java.time.LocalDateTime.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/java.time.LocalDateTime.javap.txt) |
| R282 | wheels: [cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) | `method 'getMinute' not found on builtin native.date` | [java.time.LocalDateTime.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/java.time.LocalDateTime.javap.txt) |
| R283 | wheels: [cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) | `method 'getMonthValue' not found on builtin native.date` | [java.time.LocalDateTime.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/java.time.LocalDateTime.javap.txt) |
| R284 | wheels: [cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) | `method 'getSecond' not found on builtin native.date` | [java.time.LocalDateTime.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/java.time.LocalDateTime.javap.txt) |
| R285 | wheels: [cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:128) | `method 'timestampValue' not found on builtin native.date` | [java.time.LocalDateTime.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/java.time.LocalDateTime.javap.txt) |

### mura-legacy-read — 1 entries

**Likely stale/source API issue; medium confidence.** The deprecated SOAP user API requests userManager.readByEmail, absent from the current userManager and parent declarations.

Next: Review the deprecated API against its historical version and dynamic extensions.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R286 | masa: [MSU-NatSci_MuraCMS/core/mura/client/api/soap/v1/user.cfc](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/client/api/soap/v1/user.cfc:70) | `method 'readByEmail' not found in mura.user.userManager` | Call-site context / declaration inventory (see JSON). |

### mura-form-services — 4 entries

**Runtime/version review needed; medium confidence.** Legacy form beans call getFieldService/getFormService; no corresponding source property/getter/setter or initial connection was found. Runtime bean injection/backports could supply these, so absence is not yet a proven source defect.

Next: Find the historical bean registration or deployed DI contract; otherwise treat as an unbound legacy service.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R287 | masa: [MSU-NatSci_MuraCMS/core/mura/formBuilder/fieldBean.cfc](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/formBuilder/fieldBean.cfc:411) | `chained on 'getFieldService', which is not found (calling 'getBeanByAttributes')` | Call-site context / declaration inventory (see JSON). |
| R288 | masa: [MSU-NatSci_MuraCMS/core/mura/formBuilder/fieldBean.cfc](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/formBuilder/fieldBean.cfc:411) | `chained on 'getFieldService', which is not found (calling 'getFieldTypeService')` | Call-site context / declaration inventory (see JSON). |
| R289 | masa: [MSU-NatSci_MuraCMS/core/mura/formBuilder/formBean.cfc](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/formBuilder/formBean.cfc:217) | `chained on 'getFormService', which is not found (calling 'getFormAttributeservice')` | Call-site context / declaration inventory (see JSON). |
| R290 | masa: [MSU-NatSci_MuraCMS/core/mura/formBuilder/formBean.cfc](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/formBuilder/formBean.cfc:230) | `chained on 'getFormService', which is not found (calling 'getFieldService')` | Call-site context / declaration inventory (see JSON). |

### mura-hasvalue — 1 entries

**Likely stale/source API issue; medium confidence.** publisherKeys calls hasValue on mura.cfobject. The selected source supplies valueExists instead, with no hasValue declaration. Runtime extensions remain a caveat.

Next: Review the requested API spelling and runtime augmentation.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R291 | masa: [MSU-NatSci_MuraCMS/core/mura/publisherKeys.cfc](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/publisherKeys.cfc:67) | `method 'hasValue' not found in mura.cfobject` | [cfobject.cfc](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/cfobject.cfc:85) |

### mura-translator — 1 entries

**Runtime/version review needed; medium confidence.** trashIterator conditionally calls trashItemBean.setTranslator, but the child and parent source expose no translator setter. Whether a deployment adds it dynamically is unproven.

Next: Check runtime bean augmentation or historical iterator/bean versions before calling this broken.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R292 | masa: [MSU-NatSci_MuraCMS/core/mura/trash/trashIterator.cfc](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/trash/trashIterator.cfc:54) | `method 'setTranslator' not found in trashItemBean` | Call-site context / declaration inventory (see JSON). |

### coverage-map — 1 entries

**Receiver/return relationship gap; medium confidence.** lineMetricMap is returned by FusionReactor Java metrics; its fallback is an empty CFML struct. containsKey is a Java Map API, and the guarded nonempty path needs the Java return contract rather than a universal struct classification.

Next: Preserve conditional Java-map return and fallback paths.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R294 | testbox: [Ortus-Solutions_TestBox/system/coverage/data/CoverageGenerator.cfc](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/system/coverage/data/CoverageGenerator.cfc:190) | `method 'containsKey' not found on builtin native.struct` | [java.util.Map.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/java.util.Map.javap.txt) |

### available-base-mappings — 18 entries

**Existing base: package/mapping gap; high confidence.** The base implementation is available in the source or supplemental dependency corpus; the requested virtual package/module alias does not reach it. Candidate paths below are evidence to establish a mapping, not permission to match unrelated short names.

Next: Connect engine/module namespaces and declared dependency roots with qualified mappings.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R001 | fw1: [framework-one_fw1/framework/WireBoxAdapter.cfc](/Users/garethedwards/corpus/framework-one_fw1/framework/WireBoxAdapter.cfc:1) | `extends wirebox.system.ioc.Injector (3 unchecked calls)` | [Injector.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/ioc/Injector.cfc) |
| R033 | contentbox: [Ortus-Solutions_ContentBox/modules/contentbox/models/security/SecurityValidator.cfc](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/models/security/SecurityValidator.cfc:9) | `extends cbsecurity.models.validators.AuthValidator (1 unchecked calls)` | [AuthValidator.cfc](/Users/garethedwards/corpus/coldbox-modules_cbsecurity/models/validators/AuthValidator.cfc) |
| R045 | lucee: [lucee_Lucee/core/src/main/cfml/context/admin/plugin/DDNS/Action.cfc](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/plugin/DDNS/Action.cfc:17) | `extends lucee.admin.plugin.Plugin (2 unchecked calls)` | [Plugin.cfc](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/plugin/Plugin.cfc) |
| R046 | lucee: [lucee_Lucee/core/src/main/cfml/context/admin/plugin/Note/Action.cfc](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/plugin/Note/Action.cfc:17) | `extends lucee.admin.plugin.Plugin (2 unchecked calls)` | [Plugin.cfc](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/plugin/Plugin.cfc) |
| R047 | lucee: [lucee_Lucee/core/src/main/cfml/context/admin/plugin/NoteTab/Action.cfc](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/plugin/NoteTab/Action.cfc:17) | `extends lucee.admin.plugin.Plugin (2 unchecked calls)` | [Plugin.cfc](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/plugin/Plugin.cfc) |
| R228 | wheels: [cfwheels_cfwheels/cli/lucli/services/deploy/secrets/AwsSecretsAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/AwsSecretsAdapter.cfc:14) | `extends modules.wheels.services.deploy.secrets.BaseAdapter (1 unchecked calls)` | [BaseAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/BaseAdapter.cfc) |
| R229 | wheels: [cfwheels_cfwheels/cli/lucli/services/deploy/secrets/BitwardenAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/BitwardenAdapter.cfc:12) | `extends modules.wheels.services.deploy.secrets.BaseAdapter (1 unchecked calls)` | [BaseAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/BaseAdapter.cfc) |
| R230 | wheels: [cfwheels_cfwheels/cli/lucli/services/deploy/secrets/DopplerAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/DopplerAdapter.cfc:9) | `extends modules.wheels.services.deploy.secrets.BaseAdapter (1 unchecked calls)` | [BaseAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/BaseAdapter.cfc) |
| R231 | wheels: [cfwheels_cfwheels/cli/lucli/services/deploy/secrets/LastPassAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/LastPassAdapter.cfc:9) | `extends modules.wheels.services.deploy.secrets.BaseAdapter (1 unchecked calls)` | [BaseAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/BaseAdapter.cfc) |
| R232 | wheels: [cfwheels_cfwheels/cli/lucli/services/deploy/secrets/OnePasswordAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/OnePasswordAdapter.cfc:11) | `extends modules.wheels.services.deploy.secrets.BaseAdapter (1 unchecked calls)` | [BaseAdapter.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/services/deploy/secrets/BaseAdapter.cfc) |
| R234 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/config/check.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/config/check.cfc:12) | `extends commandbox.modules.wheels-cli.commands.wheels.base (11 unchecked calls)` | [base.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/base.cfc) |
| R235 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/config/diff.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/config/diff.cfc:13) | `extends commandbox.modules.wheels-cli.commands.wheels.base (8 unchecked calls)` | [base.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/base.cfc) |
| R236 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/config/dump.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/config/dump.cfc:13) | `extends commandbox.modules.wheels-cli.commands.wheels.base (9 unchecked calls)` | [base.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/base.cfc) |
| R238 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/env/merge.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/env/merge.cfc:11) | `extends commandbox.modules.wheels-cli.commands.wheels.base (7 unchecked calls)` | [base.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/base.cfc) |
| R239 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/env/set.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/env/set.cfc:11) | `extends commandbox.modules.wheels-cli.commands.wheels.base (6 unchecked calls)` | [base.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/base.cfc) |
| R241 | wheels: [cfwheels_cfwheels/cli/src/commands/wheels/secret.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/secret.cfc:13) | `extends commandbox.modules.wheels-cli.commands.wheels.base (4 unchecked calls)` | [base.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/base.cfc) |
| R261 | wheels: [cfwheels_cfwheels/tools/lucee-extensions/sqlite/src/SQLite.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/tools/lucee-extensions/sqlite/src/SQLite.cfc:1) | `extends types.Driver (1 unchecked calls)` | [Driver.cfc](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/java/resource/context/admin/dbdriver/types/Driver.cfc) |
| R277 | wheels: [cfwheels_cfwheels/vendor/wheels/rocketunit_tests/_assets/testframework/ValidTestPackage.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/rocketunit_tests/_assets/testframework/ValidTestPackage.cfc:1) | `extends wheels.tests.Test (2 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/Test.cfc) |

### nested-test-bases — 10 entries

**Existing base: package/mapping gap; medium confidence.** Starter-app test classes use tests.Test/app.tests.Test while the local shared base lives under tests/RocketUnit/Test.cfc. This is a nested test-root/package-layout discrepancy; a runtime mapping or old layout is needed.

Next: Establish the test runner root/mapping before assigning the existing base.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R250 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/Auth.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/Auth.cfc:1) | `extends tests.Test (15 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R251 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/Permissions.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/Permissions.cfc:1) | `extends tests.Test (68 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R252 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/Utils.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/Utils.cfc:1) | `extends tests.Test (5 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R253 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/models/Auditlog.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/models/Auditlog.cfc:1) | `extends app.tests.Test (15 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R254 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/models/Permission.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/models/Permission.cfc:1) | `extends app.tests.Test (16 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R255 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/models/Role.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/models/Role.cfc:1) | `extends app.tests.Test (4 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R256 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/models/User.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/functions/models/User.cfc:1) | `extends app.tests.Test (29 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R257 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/requests/Accounts.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/requests/Accounts.cfc:1) | `extends tests.Test (12 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R258 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/requests/Main.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/requests/Main.cfc:1) | `extends tests.Test (2 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |
| R259 | wheels: [cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/requests/Sessions.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/requests/Sessions.cfc:1) | `extends tests.Test (10 unchecked calls)` | [Test.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/tests/RocketUnit/Test.cfc:2) |

### unavailable-base — 5 entries

**Unavailable base/dependency; medium confidence.** Required bases are missing from the production corpus: MXUnit TestSuiteRunner, LuCLI host modules.BaseModule, CLI tests.BaseSpec, and two wheels.Mailer sample bases. A similarly named test fixture is not a substitute for the production host base.

Next: Obtain the matching runtime/dependency/sample version, then inspect its API; do not declare inherited calls valid by assumption.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R051 | lucee: [lucee_Lucee/core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestSuiteRunner.cfc](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestSuiteRunner.cfc:1) | `extends mxunit.framework.TestSuiteRunner (4 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R227 | wheels: [cfwheels_cfwheels/cli/lucli/Module.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/lucli/Module.cfc:16) | `extends modules.BaseModule (764 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R248 | wheels: [cfwheels_cfwheels/cli/tests/specs/commands/ConfigCommandsTest.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/tests/specs/commands/ConfigCommandsTest.cfc:4) | `extends tests.BaseSpec (94 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R249 | wheels: [cfwheels_cfwheels/examples/starter-app/app/mailers/UserNotificationsMailer.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/starter-app/app/mailers/UserNotificationsMailer.cfc:4) | `extends wheels.Mailer (4 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R260 | wheels: [cfwheels_cfwheels/examples/tweet/app/mailers/UserNotificationsMailer.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/examples/tweet/app/mailers/UserNotificationsMailer.cfc:4) | `extends wheels.Mailer (4 unchecked calls)` | Call-site context / declaration inventory (see JSON). |

### migration-templates — 13 entries

**Generator placeholder; high confidence.** These files extend literal [extends], a generator substitution token. An unexpanded template has no concrete inheritance tree.

Next: Label source templates and validate generated output with its selected base, rather than resolving the token.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R264 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/announce.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/announce.cfc:1) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R265 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/change-column.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/change-column.cfc:20) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R266 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/change-table.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/change-table.cfc:13) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R267 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/create-column.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/create-column.cfc:19) | `extends [extends] (1 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R268 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/create-index.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/create-index.cfc:14) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R269 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/create-record.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/create-record.cfc:12) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R270 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/create-table.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/create-table.cfc:29) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R271 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/remove-column.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/remove-column.cfc:13) | `extends [extends] (1 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R272 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/remove-index.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/remove-index.cfc:12) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R273 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/remove-record.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/remove-record.cfc:12) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R274 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/remove-table.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/remove-table.cfc:11) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R275 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/rename-table.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/rename-table.cfc:12) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |
| R276 | wheels: [cfwheels_cfwheels/vendor/wheels/migrator/templates/update-record.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/migrator/templates/update-record.cfc:13) | `extends [extends] (2 unchecked calls)` | Call-site context / declaration inventory (see JSON). |

### native-api-contracts — 128 entries

**Engine API coverage gap; high confidence.** The exact member has engine registration, a public Java implementation, or an official runtime API contract. The qualified receiver/member relationship is absent from clif coverage. Java source and metadata prove existence, not that every engine version/overload can execute the call. Hibernate contracts are versioned; linked-map Java access fixtures explicitly skip Java >8.

Next: Add qualified, version-aware native contracts and proper return/arity behavior; avoid global name shortcuts.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R043 | lucee: [lucee_Lucee/core/src/main/cfml/context/admin/login.new.cfm](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/login.new.cfm:6) | `method 'getConfigServerDir' not found on builtin native.lucee.config` | [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1506) |
| R044 | lucee: [lucee_Lucee/core/src/main/cfml/context/admin/overview.cfm](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/overview.cfm:56) | `method 'getServlet' not found on builtin native.lucee.factory` | [CFMLFactoryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/CFMLFactoryImpl.java:536) |
| R053 | lucee: [lucee_Lucee/test/functions/ArrayPush.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ArrayPush.cfc:11) | `method 'push' not found on builtin native.array` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R054 | lucee: [lucee_Lucee/test/functions/ArrayUnShift.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ArrayUnShift.cfc:11) | `method 'unshift' not found on builtin native.array` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R055 | lucee: [lucee_Lucee/test/functions/DatePart.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/DatePart.cfc:13) | `method 'part' not found on builtin native.date` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R056 | lucee: [lucee_Lucee/test/functions/DirectoryDelete.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/DirectoryDelete.cfc:77) | `method 'getResource' not found on builtin fileopen` | [FileStreamWrapper.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/FileStreamWrapper.java:122) |
| R057 | lucee: [lucee_Lucee/test/functions/FileDelete.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileDelete.cfc:72) | `method 'getResource' not found on builtin fileopen` | [FileStreamWrapper.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/FileStreamWrapper.java:122) |
| R058 | lucee: [lucee_Lucee/test/functions/FileOpen.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:10) | `method 'getMode' not found on builtin fileopen` | [FileStreamWrapper.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/FileStreamWrapper.java:113) |
| R059 | lucee: [lucee_Lucee/test/functions/FileOpen.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:30) | `method 'getMode' not found on builtin fileopen` | [FileStreamWrapper.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/FileStreamWrapper.java:113) |
| R060 | lucee: [lucee_Lucee/test/functions/FileOpen.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:45) | `method 'getMetaData' not found on builtin fileopen` | [FileStreamWrapper.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/FileStreamWrapper.java:78); [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R061 | lucee: [lucee_Lucee/test/functions/FileOpen.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:57) | `method 'getSize' not found on builtin fileopen` | [FileStreamWrapper.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/FileStreamWrapper.java:97) |
| R062 | lucee: [lucee_Lucee/test/functions/FileOpen.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:58) | `method 'getSize' not found on builtin fileopen` | [FileStreamWrapper.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/FileStreamWrapper.java:97) |
| R063 | lucee: [lucee_Lucee/test/functions/FirstDayOfMonth.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FirstDayOfMonth.cfc:15) | `method 'firstDayOfMonth' not found on builtin native.date` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R064 | lucee: [lucee_Lucee/test/functions/GetMetaData.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/GetMetaData.cfc:58) | `method 'getmetadata' not found on builtin native.struct` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R065 | lucee: [lucee_Lucee/test/functions/ImageDrawBeveledRect.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ImageDrawBeveledRect.cfc:22) | `method 'DrawBeveledRect' not found on builtin imagenew` | [official API](https://docs.lucee.org/reference/functions/imagedrawbeveledrect.html) |
| R066 | lucee: [lucee_Lucee/test/functions/ImageRotateDrawingAxis.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ImageRotateDrawingAxis.cfc:23) | `method 'rotateDrawingAxis' not found on builtin imagenew` | [official API](https://docs.lucee.org/reference/functions/imagerotatedrawingaxis.html) |
| R067 | lucee: [lucee_Lucee/test/functions/ImageShearDrawingAxis.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ImageShearDrawingAxis.cfc:23) | `method 'shearDrawingAxis' not found on builtin imagenew` | [official API](https://docs.lucee.org/reference/functions/imagesheardrawingaxis.html) |
| R068 | lucee: [lucee_Lucee/test/functions/QueryAddRow.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryAddRow.cfc:11) | `method 'getColumnCount' not found on builtin querynew` | [Query.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Query.java:392); [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1409) |
| R069 | lucee: [lucee_Lucee/test/functions/QueryCurrentRow.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryCurrentRow.cfc:11) | `method 'currentRow' not found on builtin querynew` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R070 | lucee: [lucee_Lucee/test/functions/QueryDeleteColumn.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryDeleteColumn.cfc:10) | `method 'getColumnCount' not found on builtin querynew` | [Query.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Query.java:392); [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1409) |
| R071 | lucee: [lucee_Lucee/test/functions/QueryDeleteColumn.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryDeleteColumn.cfc:20) | `method 'getColumnCount' not found on builtin querynew` | [Query.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Query.java:392); [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1409) |
| R072 | lucee: [lucee_Lucee/test/functions/QueryDeleteRow.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryDeleteRow.cfc:9) | `method 'getColumnCount' not found on builtin querynew` | [Query.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Query.java:392); [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1409) |
| R073 | lucee: [lucee_Lucee/test/functions/QueryDeleteRow.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryDeleteRow.cfc:16) | `method 'getColumnCount' not found on builtin querynew` | [Query.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Query.java:392); [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1409) |
| R074 | lucee: [lucee_Lucee/test/functions/QueryDeleteRow.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryDeleteRow.cfc:27) | `method 'getColumnCount' not found on builtin querynew` | [Query.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Query.java:392); [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1409) |
| R075 | lucee: [lucee_Lucee/test/functions/QueryIsEmpty.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryIsEmpty.cfc:13) | `method 'isEmpty' not found on builtin querynew` | [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1464); [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R076 | lucee: [lucee_Lucee/test/functions/QueryIsEmpty.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryIsEmpty.cfc:14) | `method 'isEmpty' not found on builtin querynew` | [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1464); [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R077 | lucee: [lucee_Lucee/test/functions/QueryIsEmpty.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryIsEmpty.cfc:15) | `method 'isEmpty' not found on builtin querynew` | [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1464); [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R078 | lucee: [lucee_Lucee/test/functions/QueryKeyExists.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryKeyExists.cfc:34) | `method 'keyExists' not found on builtin querynew` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R079 | lucee: [lucee_Lucee/test/functions/QueryKeyExists.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryKeyExists.cfc:35) | `method 'keyExists' not found on builtin querynew` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R080 | lucee: [lucee_Lucee/test/functions/QueryRenameColumn.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryRenameColumn.cfc:31) | `method 'renameColumn' not found on builtin querynew` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R081 | lucee: [lucee_Lucee/test/functions/QuerySlice.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QuerySlice.cfc:12) | `method 'slice' not found on builtin querynew` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R082 | lucee: [lucee_Lucee/test/functions/SSLCertificateInstall.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SSLCertificateInstall.cfc:14) | `method 'getSecurityDirectory' not found on builtin native.lucee.config` | [Config.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Config.java:756); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1475); [ConfigServerImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigServerImpl.java:893) |
| R083 | lucee: [lucee_Lucee/test/functions/SSLCertificateInstall.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SSLCertificateInstall.cfc:20) | `method 'getSecurityDirectory' not found on builtin native.lucee.config` | [Config.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Config.java:756); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1475); [ConfigServerImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigServerImpl.java:893) |
| R084 | lucee: [lucee_Lucee/test/functions/SetDay.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SetDay.cfc:43) | `method 'setDate' not found on builtin native.date` | [java.util.Date.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/java.util.Date.javap.txt) |
| R085 | lucee: [lucee_Lucee/test/functions/SetDay.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SetDay.cfc:57) | `method 'setDate' not found on builtin native.date` | [java.util.Date.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/java.util.Date.javap.txt) |
| R086 | lucee: [lucee_Lucee/test/functions/SetMinute.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SetMinute.cfc:41) | `method 'setMinutes' not found on builtin native.date` | [java.util.Date.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/java.util.Date.javap.txt) |
| R087 | lucee: [lucee_Lucee/test/functions/SetSecond.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SetSecond.cfc:41) | `method 'setSeconds' not found on builtin native.date` | [java.util.Date.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/java.util.Date.javap.txt) |
| R088 | lucee: [lucee_Lucee/test/functions/TrimWhitespace.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/TrimWhitespace.cfc:11) | `method 'trimWhitespace' not found on builtin native.string (chain to 'len')` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R089 | lucee: [lucee_Lucee/test/functions/TrimWhitespace.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/TrimWhitespace.cfc:11) | `method 'trimWhitespace' not found on builtin native.string` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R091 | lucee: [lucee_Lucee/test/functions/imageClearRect.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/imageClearRect.cfc:22) | `method 'ClearRect' not found on builtin imageread` | [Image.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Image.java:479) |
| R092 | lucee: [lucee_Lucee/test/functions/imageCopy.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/imageCopy.cfc:22) | `method 'copy' not found on builtin imagenew` | [Image.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Image.java:790) |
| R093 | lucee: [lucee_Lucee/test/functions/imageReadbase64.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/imageReadbase64.cfc:22) | `method 'writeBase64' not found on builtin imageread` | [Image.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Image.java:1425) |
| R094 | lucee: [lucee_Lucee/test/functions/imageWritebase64.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/imageWritebase64.cfc:21) | `method 'writeBase64' not found on builtin imagenew` | [Image.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Image.java:1425) |
| R095 | lucee: [lucee_Lucee/test/functions/listitemtrim.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/listitemtrim.cfc:12) | `method 'listItemTrim' not found on builtin native.string` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R096 | lucee: [lucee_Lucee/test/functions/listitemtrim.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/listitemtrim.cfc:14) | `method 'listItemTrim' not found on builtin native.string` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R097 | lucee: [lucee_Lucee/test/functions/sgn.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/sgn.cfc:9) | `method 'sgn' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R098 | lucee: [lucee_Lucee/test/functions/sin.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/sin.cfc:30) | `method 'sin' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R099 | lucee: [lucee_Lucee/test/functions/sqr.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/sqr.cfc:7) | `method 'sqr' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R102 | lucee: [lucee_Lucee/test/functions/structsort.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structsort.cfc:93) | `method 'sort' not found on builtin native.struct` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R103 | lucee: [lucee_Lucee/test/functions/toBase64.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/toBase64.cfc:7) | `method 'toBase64' not found on builtin native.string` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R104 | lucee: [lucee_Lucee/test/functions/xmlNew.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/functions/xmlNew.cfc:9) | `method 'keyExists' not found on builtin xmlnew` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R105 | lucee: [lucee_Lucee/test/general/BinaryTemplates.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/general/BinaryTemplates.cfc:6) | `method 'getClassDirectory' not found on builtin native.lucee.config` | [Config.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Config.java:390); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:2640); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:527) |
| R106 | lucee: [lucee_Lucee/test/general/Resources.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/general/Resources.cfc:695) | `method 'getResourceProviders' not found on builtin native.lucee.config` | [Config.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Config.java:730); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:3038); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:652) |
| R107 | lucee: [lucee_Lucee/test/general/Resources.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/general/Resources.cfc:701) | `method 'getResourceProviders' not found on builtin native.lucee.config` | [Config.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Config.java:730); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:3038); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:652) |
| R108 | lucee: [lucee_Lucee/test/general/ScopeStorage/call.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/general/ScopeStorage/call.cfm:7) | `method 'getCFID' not found on builtin native.lucee.pagecontext` | [PageContextImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/PageContextImpl.java:3015) |
| R109 | lucee: [lucee_Lucee/test/jira/Jira2384.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/jira/Jira2384.cfc:91) | `method 'getFullNullSupport' not found on builtin native.lucee.config` | [Config.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Config.java:795); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:6049); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1418) |
| R110 | lucee: [lucee_Lucee/test/jira/Jira2415.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/jira/Jira2415.cfc:63) | `method 'getFullNullSupport' not found on builtin native.lucee.config` | [Config.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Config.java:795); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:6049); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1418) |
| R111 | lucee: [lucee_Lucee/test/orm/testDialects/testH2/index.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/orm/testDialects/testH2/index.cfm:2) | `method 'getDialect' not found on builtin native.sessionfactory` | [official API](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/engine/spi/SessionFactoryImplementor.html) |
| R112 | lucee: [lucee_Lucee/test/orm/testDialects/testMSSQL/index.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/orm/testDialects/testMSSQL/index.cfm:2) | `method 'getDialect' not found on builtin native.sessionfactory` | [official API](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/engine/spi/SessionFactoryImplementor.html) |
| R113 | lucee: [lucee_Lucee/test/orm/testDialects/testMysql/index.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/orm/testDialects/testMysql/index.cfm:2) | `method 'getDialect' not found on builtin native.sessionfactory` | [official API](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/engine/spi/SessionFactoryImplementor.html) |
| R114 | lucee: [lucee_Lucee/test/orm/testDialects/testPostgres/index.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/orm/testDialects/testPostgres/index.cfm:2) | `method 'getDialect' not found on builtin native.sessionfactory` | [official API](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/engine/spi/SessionFactoryImplementor.html) |
| R115 | lucee: [lucee_Lucee/test/run-tests.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/run-tests.cfm:383) | `method 'getConfigServerDir' not found on builtin native.lucee.config` | [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1506) |
| R116 | lucee: [lucee_Lucee/test/tickets/LDEV0028.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0028.cfc:13) | `method 'parseDateTime' not found on builtin native.string` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R117 | lucee: [lucee_Lucee/test/tickets/LDEV0372.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0372.cfc:40) | `method 'getCGIScopeReadOnly' not found on builtin native.lucee.config` | [ConfigPro.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigPro.java:225); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:5898); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1303) |
| R118 | lucee: [lucee_Lucee/test/tickets/LDEV0372.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0372.cfc:53) | `method 'getCGIScopeReadOnly' not found on builtin native.lucee.config` | [ConfigPro.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigPro.java:225); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:5898); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1303) |
| R119 | lucee: [lucee_Lucee/test/tickets/LDEV0866.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:34) | `method 'keySet' not found on builtin native.struct (chain to 'iterator')` | [StructSupport.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/StructSupport.java:104) |
| R120 | lucee: [lucee_Lucee/test/tickets/LDEV0866.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:34) | `method 'keySet' not found on builtin native.struct` | [StructSupport.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/StructSupport.java:104) |
| R121 | lucee: [lucee_Lucee/test/tickets/LDEV0866.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:44) | `method 'entrySet' not found on builtin native.struct` | [StructSupport.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/StructSupport.java:89) |
| R122 | lucee: [lucee_Lucee/test/tickets/LDEV0866.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:44) | `method 'entrySet' not found on builtin native.struct (chain to 'iterator')` | [StructSupport.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/StructSupport.java:89) |
| R123 | lucee: [lucee_Lucee/test/tickets/LDEV0866.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:55) | `method 'values' not found on builtin native.struct (chain to 'iterator')` | [StructImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/StructImpl.java:352); [StructSupport.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/StructSupport.java:240) |
| R124 | lucee: [lucee_Lucee/test/tickets/LDEV0866.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:55) | `method 'values' not found on builtin native.struct` | [StructImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/StructImpl.java:352); [StructSupport.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/StructSupport.java:240) |
| R126 | lucee: [lucee_Lucee/test/tickets/LDEV1109.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1109.cfc:75) | `method 'getResourceProviders' not found on builtin native.lucee.config` | [Config.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Config.java:730); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:3038); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:652) |
| R128 | lucee: [lucee_Lucee/test/tickets/LDEV1160.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1160.cfc:21) | `method 'getColumnList' not found on builtin querynew` | [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1431) |
| R129 | lucee: [lucee_Lucee/test/tickets/LDEV1160.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1160.cfc:30) | `method 'getColumnList' not found on builtin querynew` | [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1431) |
| R132 | lucee: [lucee_Lucee/test/tickets/LDEV1295.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1295.cfc:13) | `method 'Duplicate' not found on builtin querynew` | [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:1629); [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R133 | lucee: [lucee_Lucee/test/tickets/LDEV1295.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1295.cfc:36) | `method 'duplicate' not found on builtin native.date` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R134 | lucee: [lucee_Lucee/test/tickets/LDEV1576/test.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1576/test.cfm:4) | `method 'getImageBytes' not found on builtin imagenew` | [Image.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Image.java:2133) |
| R135 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:11) | `method 'aCos' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R136 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:12) | `method 'asin' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R137 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:13) | `method 'atn' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R138 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:14) | `method 'bitAnd' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R139 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:15) | `method 'bitMaskClear' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R140 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:16) | `method 'bitMaskRead' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R141 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:17) | `method 'bitMaskset' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R142 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:18) | `method 'bitNot' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R143 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:19) | `method 'bitOR' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R144 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:20) | `method 'bitSHLN' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R145 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:21) | `method 'bitXor' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R146 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:22) | `method 'ceiling' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R147 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:23) | `method 'cos' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R148 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:25) | `method 'fix' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R149 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:26) | `method 'floor' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R150 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:27) | `method 'formatBaseN' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R151 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:29) | `method 'log10' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R152 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:30) | `method 'max' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R153 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:31) | `method 'min' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R154 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:32) | `method 'round' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R155 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:33) | `method 'sgn' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R156 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:34) | `method 'sin' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R157 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:35) | `method 'sqr' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R158 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:36) | `method 'tan' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R159 | lucee: [lucee_Lucee/test/tickets/LDEV1824.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:37) | `method 'BitSHRN' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R162 | lucee: [lucee_Lucee/test/tickets/LDEV2558.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV2558.cfc:14) | `method 'getServerRHExtensions' not found on builtin native.lucee.config` | [ConfigPro.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigPro.java:157); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1348); [ConfigServerImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigServerImpl.java:1209) |
| R163 | lucee: [lucee_Lucee/test/tickets/LDEV2936.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV2936.cfc:15) | `method 'elemNew' not found on builtin xmlnew` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R164 | lucee: [lucee_Lucee/test/tickets/LDEV3070/LDEV3070.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3070/LDEV3070.cfm:13) | `method 'getDatasourceName' not found on builtin queryexecute` | [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:179) |
| R168 | lucee: [lucee_Lucee/test/tickets/LDEV332.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV332.cfc:66) | `method 'round' not found on builtin native.numeric` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R173 | lucee: [lucee_Lucee/test/tickets/LDEV3863.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3863.cfc:12) | `method 'getSql' not found on builtin queryexecute` | [Query.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Query.java:396); [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:2188) |
| R174 | lucee: [lucee_Lucee/test/tickets/LDEV3863.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3863.cfc:21) | `method 'getSql' not found on builtin queryexecute` | [Query.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Query.java:396); [QueryImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/QueryImpl.java:2188) |
| R175 | lucee: [lucee_Lucee/test/tickets/LDEV4185/LDEV4185.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4185/LDEV4185.cfm:2) | `method 'beginTransaction' not found on builtin native.hibernatesession` | [official API](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/Session.html) |
| R176 | lucee: [lucee_Lucee/test/tickets/LDEV4459.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4459.cfc:7) | `method 'Shear' not found on builtin imagenew` | [Image.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Image.java:1238) |
| R181 | lucee: [lucee_Lucee/test/tickets/LDEV5185.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5185.cfc:34) | `method 'getConfigServerImpl' not found on builtin native.lucee.config` | [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:157) |
| R182 | lucee: [lucee_Lucee/test/tickets/LDEV5185.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5185.cfc:34) | `method 'getConfigServerImpl' not found on builtin native.lucee.config (chain to 'resetAll')` | [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:157) |
| R183 | lucee: [lucee_Lucee/test/tickets/LDEV5292/cfml-session/testThreadCookies.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5292/cfml-session/testThreadCookies.cfm:20) | `method 'hasCFSession' not found on builtin native.lucee.pagecontext` | [PageContextImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/PageContextImpl.java:1732) |
| R184 | lucee: [lucee_Lucee/test/tickets/LDEV5292/j2ee-session/testThreadCookies.cfm](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5292/j2ee-session/testThreadCookies.cfm:17) | `method 'hasCFSession' not found on builtin native.lucee.pagecontext` | [PageContextImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/PageContextImpl.java:1732) |
| R188 | lucee: [lucee_Lucee/test/tickets/LDEV5640.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5640.cfc:16) | `method 'getBase64String' not found on builtin imagenew` | [Image.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/Image.java:1442) |
| R189 | lucee: [lucee_Lucee/test/tickets/LDEV5963.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5963.cfc:69) | `method 'setMinEvictableIdleTimeMillis' not found on builtin native.pool` | [BaseGenericObjectPool.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/BaseGenericObjectPool.java:1676) |
| R190 | lucee: [lucee_Lucee/test/tickets/LDEV5967.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5967.cfc:93) | `method 'cleanDatasourceConnectionPools' not found on builtin native.lucee.config` | [ConfigPro.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigPro.java:170); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:3624); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1877) |
| R191 | lucee: [lucee_Lucee/test/tickets/LDEV5967.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5967.cfc:94) | `method 'cleanDatasourceConnectionPools' not found on builtin native.lucee.config` | [ConfigPro.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigPro.java:170); [ConfigImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigImpl.java:3624); [ConfigWebImpl.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/ConfigWebImpl.java:1877) |
| R192 | lucee: [lucee_Lucee/test/tickets/LDEV5967.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5967.cfc:148) | `method 'getJmxName' not found on builtin native.pool` | [BaseGenericObjectPool.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/BaseGenericObjectPool.java:672) |
| R193 | lucee: [lucee_Lucee/test/tickets/LDEV5967.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5967.cfc:170) | `method 'setLastBorrowed' not found on builtin native.pool` | [DatasourceConnPool.java](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/DatasourceConnPool.java:57) |
| R194 | lucee: [lucee_Lucee/test/tickets/LDEV6051.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV6051.cfc:98) | `method 'getMaxTotal' not found on builtin native.pool` | [BaseGenericObjectPool.java](/Users/garethedwards/development/github/clif/target/resolution/remaining-295/sources/BaseGenericObjectPool.java:737) |
| R196 | lucee: [lucee_Lucee/test/tickets/LDEV6171.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV6171.cfc:13) | `method 'getResourceAsStream' not found on builtin native.class` | [java.lang.Class.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/java.lang.Class.javap.txt) |
| R198 | coldbox: [ColdBox_coldbox-platform/build/Build.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/build/Build.cfc:485) | `method 'reFindNoCase' not found on builtin native.string` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |
| R203 | coldbox: [ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc:609) | `method 'getMetaModel' not found on builtin native.sessionfactory (chain to 'getAllEntityNames')` | [official API](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/engine/spi/SessionFactoryImplementor.html) |
| R204 | coldbox: [ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc:609) | `method 'getMetaModel' not found on builtin native.sessionfactory` | [official API](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/engine/spi/SessionFactoryImplementor.html) |
| R205 | coldbox: [ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc:612) | `method 'getAllClassMetadata' not found on builtin native.sessionfactory` | [official API](https://docs.jboss.org/hibernate/orm/5.6/javadocs/org/hibernate/engine/spi/SessionFactoryImplementor.html) |
| R263 | wheels: [cfwheels_cfwheels/vendor/wheels/auth/PasswordHasher.cfc](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/auth/PasswordHasher.cfc:253) | `method 'getMethod' not found on builtin native.class` | [java.lang.Class.javap.txt](/Users/garethedwards/development/github/clif/target/resolution/method-existence-audit/java.lang.Class.javap.txt) |
| R293 | testbox: [Ortus-Solutions_TestBox/build/Build.cfc](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/build/Build.cfc:264) | `method 'reFindNoCase' not found on builtin native.string` | [core-base.fld](/Users/garethedwards/development/github/clif/target/resolution/twenty-pass-followup/sources/core-base.fld) |

### native-contract-review — 1 entries

**Runtime/version review needed; medium confidence.** The setMetadata call is inside xit in LDEV0974. No Lucee 7.1 declaration or member registration was found; Adobe supports this serialization metadata API. This is a disabled cross-engine/unsupported-feature fixture, not an established missing Lucee resolver connection.

Next: Keep this disabled cross-engine fixture distinct from supported Lucee contracts; verify the intended serialization feature before implementation.

| ID | Project / source | Reported call or base | Evidence |
|---|---|---|---|
| R125 | lucee: [lucee_Lucee/test/tickets/LDEV0974.cfc](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0974.cfc:98) | `method 'setMetadata' not found on builtin native.struct` | [official API](https://helpx.adobe.com/coldfusion/cfml-reference/coldfusion-functions/functions-s/serializejson.html) |
