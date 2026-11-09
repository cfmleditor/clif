# Component resolvers and call resolution

_`componentResolvers`, framework presets, expression mappings, `noFollow`, triaging lint errors, and Java stubs. Moved out of CLAUDE.md unchanged._

## Component resolvers

Component resolvers (`componentResolvers`) teach the LSP how to map a call-site expression to a
component dot-path. They are tried in order by `ResolveFromCall` / `ResolveFromCallFull` in
`internal/parser/ast.go`.

**Component-type resolution order** (`CanResolveCall`, `internal/resolve/resolve.go`): for a
qualified call `x.method()`, the receiver's component is looked up in this order, stopping at
the first hit — (1) `call.Component`, if already set at parse time (e.g. a chained
`new`/`createObject`, or a bare-call site where the tag/script parser resolved the receiver
inline via `lookupComponentRef`); (2) a function-scoped `ComponentRef` for `x`; (3) a file-level
(global/`VARIABLES.`/`this.`) `ComponentRef` — for (2) and (3) a receiver written `this.x`
reads only refs assigned through `this.` and one written `variables.x` only the rest
(`parser.ReceiverRefScope`); (4) a `ComponentRef` on
`Application.cfc`/`Application.cfm`; (5) for `ARGUMENTS.x`, the `<cfargument type>` if it's a
dotted path; (6) walking the `extends` chain's own `ComponentRef`s; (7) a `componentResolver`
matched against the variable name text; (8) a `componentResolver` matched against the full line
text (handles chains like `x.method().prop.func()`). If the call is itself chained
(`call.Chain`), each hop repeats a scaled-down version of this: what the hop function returns
(`Resolver.ReturnComponentOf` — its inferred `ReturnComponent`, a dotted `ReturnType`, or a
bare-word `ReturnType` naming a component beside the declaring file, never a CFML type name),
falling back to a `componentResolver` matched against `hop()`. `ReturnComponentOf` is also
behind `Resolver.FuncLookup`, the parser hook every scan passes, so a hop and a variable
assigned from the same call cannot disagree; it replaced five copies of the rule. Because steps (7)/(8) and the per-hop fallback all go through the same substring-prefix
matching described below, a broad catch-all resolver (e.g. `get$1()`) can win at *any* of these
steps, not just the ones that look like factory-method calls.

Two component values short-circuit the method-exists check unconditionally: `$any` (dynamic —
see "Expression mappings") and `$builtin.<fn>` (a built-in CFML function's return type, from
`internal/docs/builtin_returns.go`). A `ReturnComponent` may also be a pipe-delimited union of
candidates (`a|b`); the method lookups treat it as a set, accepting a method any candidate
declares and reporting one none does against the joined display.

**How matching works** (`internal/parser/ast.go: matchResolverWithCache`):
- Each resolver has a `prefix` (fast-rejection substring) and a `match` pattern.
- `prefix` is searched for anywhere in the expression unless the resolver sets `"anchored":
  true`, which requires it at position 0 (see "Resolver false-positive" below).
- Resolvers are tried in **array order** — so when two resolvers could both match the same
  expression (e.g. a specific `getDirectContent()` entry and a generic `get$1()` catch-all),
  whichever is listed **earlier** wins, regardless of specificity. To make a specific case win
  over an existing broad catch-all, add it *before* the catch-all in the array, not after.

  Two implementations honour this and must stay in agreement: `ResolveFromCallMatch` (behind
  `ResolveFromCall`/`ResolveFromCallFull`, and so behind `CanResolveCall`) walks the slice
  directly, while `ResolverSet.Resolve` (behind completion and hover) first narrows candidates
  through a first-byte index and then sorts them back into array order. The byte index decides
  only *which* resolvers are worth trying, never which one wins — it used to leak its own order
  through, so for resolvers whose prefixes start with different letters the winner was decided by
  where each prefix happened to appear in the call-site text, and the two paths could return
  different components for the same expression. `TestResolverSetMatchesArrayOrder`
  (`internal/parser/resolver_order_test.go`) pins the two paths together; add a case there rather
  than to only one path.
- `ResolveFromCall` finds the prefix inside the expression, takes the substring from that
  position, and tries to match the full pattern against it.
- If `match` contains no `\` escapes and no `$N` placeholders → **simple exact match** (or
  `match()` suffix).
- If `match` contains `$1` but no `\` → **simple prefix/suffix match**: the text before `$1` and
  after `$1` are checked as plain strings; the captured value replaces `$1` in `resolve`.
- If `match` contains `\` → **regex match**.

**Pipe-delimited `prefix` alternatives** (`splitPrefix`, `findPrefixPos`):

`prefix` may list alternatives separated by `|` (e.g. `"createModel|buildModel"`), letting one
resolver's `match`/`resolve` pair cover call-site shapes that don't share a common leading
substring. `findPrefixPos` tries each alternative via `indexFold` and returns the position of
whichever is found; `BuildResolverSet`'s byte-bucket index registers the resolver under every
alternative's first byte so fast rejection still finds it. Which alternative matched has no
effect on the subsequent `match`/`resolve` step — only the matched *position* is used to slice
the expression — so this is purely a fast-lookup mechanism, not a change to match semantics.

Gotcha when merging plain bare-word entries this way: `match`'s regex-vs-simple decision
(`isRegexPattern`) is triggered *only* by a literal backslash in `match` — it has nothing to do
with `prefix` or with `|` in `match`. A merged pattern like `match: "kernel|_kernel"` has no
backslash, so it is compared as one literal string (never equal to either bare word), not as
alternation — silently matching nothing. Give merged bare-word alternatives a real anchor that
needs escaping, e.g. `"^kernel(?:\\(\\))?$|^_kernel(?:\\(\\))?$"`.

**Case-folded placeholders in `resolve`** (`substitutePlaceholder`):

Both simple prefix/suffix and regex matches substitute captures through the same helper, which
supports `${N:lower}` and `${N:upper}` in addition to plain `$N`. Use `${1:lower}` to fold a
captured value: `{"match": "get$1()", "resolve": "packages.tass.${1:lower}", "prefix": "get"}`
turns `getPageTools()`, `getLockBroker()`, etc. into one resolver instead of one entry per name.
This only helps when the target path is a mechanical case-fold of the captured text — if the
target name isn't derived from it at all (e.g. an `itextObj.Foo` family mapping to unrelated
Java class names), each name still needs its own entry. Since path resolution is itself fully
case-insensitive, `${N:lower}`/`${N:upper}` are for config explicitness, not correctness.

**Defaults a configuration still overrides.** When no mapping, directory or workspace folder
resolves a dot-path, `slugRoot` looks above the calling file for a `box.json` whose `slug` is
the path's first segment (coldbox-platform spells its own `system/` as `coldbox.system`). When no
resolver types a call, `dynamicCall` makes a MockBox mock and an unstubbed Java object `$any`,
from both `resolveCall`s, `tryResolveCall` and `resolvePendingCalls` — the chained
`getMockBox().createEmptyMock()` reaches only the last. A component the parse names by file
path (a function returning `this`) is reported by `displayComponent` under its file name,
since reasons land in committed known-issues files. The methods MockBox's `decorateMock` adds
(`$`, `$results`, `$never`, … — `mockDecorations`) are accepted on any component, as a last
call or a chain hop, since a test mocks a real component in place. An engine component — bare `Query`, `http`, `dbinfo` and the
other script-tag names, or `org.lucee.cfml.*`/`com.adobe.coldfusion.*` — is `engineComponent`:
once nothing configured resolves it, `engineSource` looks for an indexed file ending in its
path (a Lucee checkout's `org/lucee/cfml/`), and without one `canResolveCall` accepts calls
on it as dynamic. It never falls to the file-name search, which answered `new dbinfo()` with
whatever `dbinfo.cfc` the workspace held. Lucee functions its docs omit (hidden ones, like
`struct()`) are `docs.undocumentedFunctions`, behind `docs.IsBuiltinFunction` — the one
builtin test `unresolved` and the code map share.

**Framework presets** (`frameworks`, `internal/config/frameworks.go`) are data, not code
paths: a list of resolvers and implicit bases per framework, so a preset can only say what a
project could have written by hand. Three rules hold them together, each with a test:
- **Every preset resolver is `dynamicIfMissing`**, and a chain that breaks after passing
  through an implied link is dynamic too, so an app without the framework checked out gets
  nothing reported against components that are not on disk, while one with it is checked
  for real. `chainBreak` is the one walk behind it: it names the first link that does not
  resolve and whether any link up to it was implied — the file's own base, a base class that
  names none (`handlers/Base.cfc`), or the base of a component a call is made on
  (`componentMissingBase`). A chain every file wrote is still reported where it breaks. An
  implied base may be a list of alternatives (a Wheels view is its controller and every
  mixin); `MissingBase` answers nothing for a list, so `chainBreak` checks each alternative
  of an implied one. The bare-chain walk carries a `dynamicIfMissing` return type's flag
  too (`resolveBareChain`), which it used to drop.
- **Variables match whole names**, anchored and escaped (`variableResolver`): unanchored,
  prefix `event` finds itself inside `oEvent`. Return types are unanchored on purpose
  (`returnResolver`), so `x = variables.controller.getRequestContext()` types `x`; the match
  is still the whole remainder, so a further hop is not claimed.
- **The implicit base is the resolver's, not the parse's.** `Resolver.ImplicitExtends` is a
  hook `config.ImplicitExtends` builds, because `resolve` must not import `config` (which
  pulls in tree-sitter). Every read of a file's extends in `resolve.go` goes through
  `fileExtends`/`extendsFor`, memoised per path; the deepest matching directory wins. Each of
  the seven places that builds a `Resolver` sets the hook from its config.
- **Helpers a framework mixes in are found like includes.** `Resolver.HelperScope` (from a
  preset's `helperDirs`) says which files receive them; `internal/resolve/helpers.go` finds
  them — every `ModuleConfig.cfc`'s `this.applicationHelper` and
  `includeUDF( "#moduleMapping#/…" )`, each `config/ColdBox.cfc`'s `applicationHelper`, and a
  view's `<view>Helper.cfm`/`<folder>Helper.cfm` — and `findThroughIncludes` tries them last.
  Application helpers are workspace-wide, cached per set of config files. With ColdBox mapped
  into ContentBox it resolves all 40 of the admin module's helper calls.
- **Preside's checkout is ColdBox with Preside's request context.** Its
  `tests/Application.cfc` maps `/preside` to the root, which
  `AssumedProjectMappings` recognises (markers `system/Bootstrap.cfc` and
  `system/services/PresideSuperClass.cfc`); `config.Assumed` then assumes the
  coldbox preset and types `event` as
  `preside.system.coldboxModifications.RequestContextDecorator` ahead of the
  preset's own. Preside's application helpers are every `.cfm` under
  `system/helpers` (Config.cfc's computed `_getUdfFiles`), found beside each
  indexed `system/services/PresideSuperClass.cfc` (`presideHelpers`). Preside
  went from 15,869 findings to 9,753. A component whose header (doc comment
  or attributes), or an ancestor's, holds `presideService` gets
  PresideSuperClass's `$` methods, as Preside's `config/WireBox.cfc` gives
  them by virtual inheritance (`presideSuperClassFunc`; 9,753 to 8,344). Not
  yet handled: `setNextEvent`, which Preside's older ColdBox has and the stubs
  do not (373); the test base's `_getPresideObjectService()`, a request cache
  under a computed key (849 `poService` calls); and Preside's own
  `coldboxModifications/Controller` standing in for ColdBox's. `TestAPresideCheckoutIsColdBoxWithPresidesRequestContext`,
  `TestPresideHelpersReachTheFilesColdBoxMixesThemInto`,
  `TestAPresideServiceHasPresideSuperClassMethods`.
- **An argument documented `@arg.inject id` is the component the id names**
  (`applyInjectDoc`, as an injected property is: `injectedComponent`), and a
  getter returning a field is what the field is handed when every write of it
  is `field = arguments.p` in a function whose p is a component, or in a
  private setter every call to which in the file is a statement passing one
  whole argument that is (`forwardedFieldReturn`). That is how Preside's
  services hold every dependency: `_setX( arguments.x )` in init and
  `_getX().m()` everywhere (8,162 to 7,191). A public setter is not followed,
  since WireBox may call it.
  `TestAFieldSetThroughAPrivateSetterIsWhatInitIsHanded`.
- **A handler has EventHandler whatever it extends** (`virtualBaseFunc`):
  ColdBox's HandlerService registers every handler with
  `setVirtualInheritance( "coldbox.system.EventHandler" )`, and
  InterceptorService every interceptor with `Interceptor`, so a bare call
  the declared chain does not answer is looked for there. Only those two
  bases (`virtualBases`): a view's Renderer and a Wheels class stand only
  for a missing extends. Preside's admin handlers extend an AdminHandler that
  extends nothing (8,344 to 8,162). `TestAHandlerGetsEventHandlerWhateverItExtends`.
- **A Mura plugin's checkout has the mappings Mura's plugin manager writes**
  (`cfpath.AssumedMuraPlugin`, from `plugin/config.xml(.cfm)`): its `<package>`
  maps to the plugin's directory and each `<mapping name directory>` to a
  directory inside it, and the Mura preset is assumed. Slatwall's own checkout
  (`config.isSlatwall`) also assumes FW/1 and types what its Application.cfc
  stores (`$.slatwall`, `getFW()`, `getService( "x" )`, the SlatwallScope's
  `cart()` family) and its ColdSpring-autowired `xService`/`xDAO` properties
  (`config/slatwall.go`). Its BaseService answers `this.newX()`, `getX()`,
  `getXByY()`, `saveX()` and `getXSmartList()` through onMissingMethod with
  the entity `SlatwallX` (`resolve.slatwallGatewayReturn`). A chain on
  `this.f()` carries `CallSite.This`, since only that one reaches
  onMissingMethod. Slatwall went from 7,533 findings to 3,187.
  `TestAMuraPluginCheckoutHasTheMappingsMuraGivesIt`,
  `TestASlatwallCheckoutIsAMuraPluginOnFW1WithItsStoredValues`,
  `TestSlatwallServicesReturnTheEntityTheirMissingMethodNames`.
- **Two presets can claim one name.** CommandBox's `task()` and a ColdBox scheduler's
  `task()` return different things, so CommandBox's preset leaves `task()` alone.
  `TestEveryPresetResolverMatchesItsOwnNames` lists such a case beside each preset's own.
- **Without the framework's source, its API comes from stubs** (`internal/frameworkapi`).
  `cmd/cfstubgen` generates one script component per class a preset names — plus the
  `Extra` bases projects extend by name, their extends chains, and the components their
  methods return — holding every signature and doc comment with an empty body, from each
  `Sources` entry at its pinned commit (`make framework-stubs`, reproducible byte for
  byte). Wheels' `$integrateComponents` mixins and included templates are folded in. The
  resolver's `Stubs` answers a dot-path **last**, after every mapping, directory and
  folder, so a checkout of the framework always wins (`TestTheFrameworksOwnSourceOutranksItsStubs`);
  the stubs are served from a virtual directory, `frameworkapi.Root`, through `r.fs()` and
  a wrapped `cfpath.DefaultFS`. Checked against the real thing: ContentBox reports the same
  3,776 entries with the ColdBox stubs as with ColdBox mapped in, and cfwheels fewer with
  its stubs than with its source. Three rules, each with a test: **a stub is never a
  location** — `handleDefinition`/`handleTypeDefinition` filter them (`withoutStubLocations`)
  and workspace symbols skip them; **a stub is documented** — hover, signature help and
  member completion append `frameworkapi.DocAt`, which reads ColdBox's `@name text`
  argument docs as well as `@param`; and **a chain breaking beyond a stub is dynamic**
  (`chainBreak` marks a stub link implied), since that is the framework's absence. The
  code map keeps stubs out as it kept the missing framework out: a stub call is an
  external node only with unresolved calls shown. Every `Resolver` construction site sets
  `Stubs` beside `ImplicitExtends`.
- **What a factory builds is typed by the id it is handed** (`idResolver`): ColdBox's
  `getInstance( "X@module" )` and Wheels' `model( "X" )` are the component `X`, by path and
  then by file name nearest the caller, as an injected property's id is. The match is the
  whole remainder, so `getInstance( "X" ).init()` is not claimed, and a computed id is left
  alone. `getInstance( "logbox" )` is **not** a component called logbox: the WireBox DSL a
  getInstance() may be handed is `dslResolvers`, built from `parser.InjectionDSL` and listed
  *before* `idResolver`. `TestEveryPresetResolverMatchesItsOwnNames` has each shape.
- **The injection DSL names ColdBox's classes, not files** (`parser.injectedComponent`,
  `injectionDSL`, `coldboxModels`): `inject="coldbox:requestService"`, `wirebox:populator`,
  `logbox:logger:{this}`, `cachebox:template`, `provider:cachebox`, `Renderer@coldbox`.
  Each is a `coldbox.system.*` path, and **every such class is soft** in
  `ParseResult.IsSoftComponent` whatever file is asking: the property is usually declared
  by a base class, so a record kept on the declaring file's parse never reached the
  subclass making the call, and 45 calls in ContentBox without ColdBox came back as
  "does not exist". A cache is `CacheBoxColdBoxProvider`, not the `ICacheProvider`
  interface, which has no `getOrSet()`. `TestEveryInjectionHasItsStub` fails on a DSL
  target the stubs lack. **`x = base.m()` does not take base's type when base is one of
  these** (`baseVarComponent`): an injector's `getInstance()` is not an injector, and the
  guess made every WireBox-built local an Injector and then a function returning one
  declare it returned an Injector, outranking its declared `SearchResults`.
- **A stub's return type may come from its doc comment** (`cmd/cfstubgen`, `docReturn`).
  ColdBox declares few return types and documents most, so `@return <dotted.path>` becomes
  the stub's declared type when the source declares none: `execute()` returns a
  `RequestContext`. A documented path that does not exist (execute's says
  `coldbox.system.context.RequestContext`) is found by file name when exactly one file in
  the framework has it; an **interface is left out** (`isInterface`), since it declares less
  than what the call hands back. This is deliberately *not* done for a real checkout of the
  framework: without the generator's check that the path exists, a wrong doc path would be
  reported as a component that does not exist. `TestAStubReturnsWhatItsDocSays`.
- **Methods a framework adds at run time are found where its docs say they come from.**
  WireBox delegates (`parser/delegates.go`): `property name="m" inject delegate
  delegatePrefix;` and `component delegates=">Memory, Worker=vacation"` give the host the
  target's methods as prefix + name + suffix. Both parsers record a valueless property flag
  (`propertyFlags`) as an attribute with an empty value — they used to drop it — and a bare
  or empty `inject` is the model named by the property, WireBox's default DSL. The index
  keeps a file's delegates beside its extends and drops them with it (`removeFileEntries`),
  and `lookupFunc` searches the whole extends chain before any delegate, since a delegated
  method never replaces one the object has; `maxDelegateDepth` stops two components
  delegating to each other. CFML's ORM relationship methods (`has`/`add`/`remove`, per
  `relationshipMethods`) are generated beside a property's accessors. Wheels mixes the
  application's `global/functions.cfm` and what it includes into controllers, models and
  views (`wheelsGlobals`, through the wheels preset's `helperDirs`).
- **A path in a framework's own namespace needs no preset** (`frameworkapi.Namespaced`):
  `coldbox.system.`, `testbox.system.`, `commandbox.system.`, `qb.models.` and
  `contentbox.models.` can only mean that framework, so the resolver answers them from the
  stubs after everything on disk, whatever the config names. Lucee's test suite extends
  `testbox.system.compat.framework.TestCase` with no preset and was 1,885 broken chains.
  `framework.` (FW/1) and `wheels.` are left to their presets, since a project may have a
  folder of that name. `r.fs()` therefore always mounts the stubs. Tests that need a base
  that does not resolve use a made-up one (`vendor.missing.BaseSpec`), not a framework's.
- **A stub keeps what the framework's variables hold** (`writeTypedVariables`): a
  `variables.`/`this.` variable the source assigns a framework component becomes a
  statement in the stub's pseudo-constructor, so `$assert` (BaseSpec's constructor sets it)
  is still an Assertion with the body gone.
- **A preset's variable resolvers are `nameOnly`.** The tag parser's bare-name fallback
  (`resolveRHS`) types `<cfset x = svc.load()>` as `svc`'s component, which is deliberate for a
  project's own resolvers (`TestResolverMatch_PipeDelimitedPrefix_BareNameFallback`) and wrong
  for `event.getValue()`. `NameOnly` keeps the alias `x = event` and drops anything derived
  from it. **Every config-to-parser conversion is `config.Resolver.Parser()`** — there were
  ten struct literals, and a field one of them forgot is a setting that parses and does
  nothing; `TestParserCarriesEveryResolverField` sets every field and fails on one dropped.

**What each framework says about its own components is applied without a preset**, each
rule in its own file with a test that fails without it. `resolve/wirebox.go`: a
`Name@namespace` id is the Name in the module whose `this.modelNamespace` (default: its
directory) is that namespace, a `binder.map( id ).to( path )` in any `ModuleConfig.cfc` or
`config/WireBox.cfc` is exact, a module's `this.cfmapping` maps its directory, and an id of a
module the workspace lacks is dynamic (`uninstalledModule`). **`injectedComponent` keeps the
`@module`** so this can run; the stubs' `IDPackages` answer a bare CommandBox id.
`resolve/orm.go`: `entityname` is indexed (`ParseResult.EntityName`) and looked up last in
`lastResortPath`; a cborm `VirtualEntityService` bound by `super.init( entityName = … )`
returns its entity from `new`/`get`/`getOrFail`/`findWhere`. `resolve/wheels.go`: a Wheels
model's finders return the receiver, and `hasMany`/`belongsTo`/`hasOne` generate typed
methods. `resolve/mura_loadby.go`: a Mura bean's `loadBy()` (a chain reaching
`mura.bean.bean`) returns the receiver, as Mura documents and no return can show; where
the bean's `loadBy` takes a `returnFormat` (beanORM's) only a call naming `"self"` or
none, so `receiverReturn` takes a `CallHop` and `funcLookupUncached` passes it one
(159 of the 669 masa-c findings it removed). `TestAMuraBeansLoadByReturnsTheBean`.
`resolve/wheels_controller_paths.go`: `controller( "name" )` is the class
Global's `$createControllerClass` (checked against its pinned body) instantiates —
`<path>/<name>.cfc`, else the last path's `Controller.cfc` — under the literal
`controllerPath` the nearest directory above the caller writes (a test runner's), else
the framework default; a computed write there withholds it. `resolve/modules.go`: mementifier's `getMemento()` on a `this.memento` component,
and cbi18n/cbfs/HTMLHelper helpers on a component whose chain reaches `coldbox.system.` —
the lists are read from each module's source at the commit named there.
`resolve/missing_method.go`: `this.x()` on a component with `onMissingMethod` is dynamic (an
unscoped call is not — CFML never routes it there, and the parser records both alike, so the
line is read); a method declared to return its own class, called on a subclass, returns the
subclass (`selfTyped`, which only widens what is accepted). A call on an interface that lacks
the method is dynamic, since an implementation has more. DI/1 (`path/beans.go`,
`ParseAppBeanPaths`): default folders `model,controllers`, aliases name+singular(folder).
MockBox: `createMock( "cls" )` is `parser.MockPrefix`+cls, checked against cls, made directly
or on whatever holds the MockBox (`getMockBox().createMock( "cls" ).init( … )`): the parser
(`chainedMock`), the producer interpreter (`producerEvaluation.mock`) and the hop walk
(`mockHop`) all read the class through `parser.MockCall`, and hops after it are walked on
the class. The chained form used to lose its argument and become `$any`. The parser
side: a `@return` doc is kept (`FunctionDef.DocReturn`, used only where it names a file);
generated setters return their component and getters what the property holds;
**`return x.m()` is not `return x`** (`returnsCallOn`: it is `$any` when x is, nothing
otherwise — it used to declare CommandBox's `getCWD()` a `Shell`); and an unqualified call to
an inherited method types what it is assigned to through `FuncLookup`. cborm's stubs are
pinned to 4.12.1, the version ContentBox's stubs use: 5.x dropped `getBeanPopulator`.
`.ai/docs/RESOLUTION-GAPS.md` lists what `unresolved` still reports over the corpus, the cause of
each large group and where its fix goes, with the corpus setup to measure a change against.

**`Application.cfc` mappings are evaluated, not just matched** (`parser.evaluatedMappings`).
The literal-only regex read 20 of the corpus's 110 `this.mappings` assignments; the rest are
built from the file's own directory and from variables set a line above —
`local.projectRoot = expandPath( "../../../" ); this.mappings[ "/cli" ] = local.projectRoot & "cli/";`.
`evalPathExpr` handles `&`, string literals, `expandPath()`,
`getDirectoryFromPath( getCurrentTemplatePath() )`, a variable assigned earlier in the file
and an earlier mapping, and declines anything else whole. **A mapping name may have several
segments** (`/modules/wheels` beside `/modules`): `resolveNestedMapping` tries those, longest
first, before the first-segment lookup, which could never see them. Together they were 1,700
corpus entries, most of cfwheels' CLI specs.

**A component has the functions of the templates it includes** (`includedFunc`): Wheels'
`Global.cfc` is little but includes of `global/*.cfm`, and a qualified call
(`application.wo.$simpleLock()`) looked only at the component's own functions. It reads the
component's own include statements from the index (`Index.IncludesForFile`) rather than the
workspace include graph, which is rebuilt whenever an include is indexed.

**A chain headed by a built-in function is dynamic** (`resolveBareChain`):
`getPageContext().getRequest()` is a call on what the engine returns, and the file's own
function of that name is looked for first. **A bare inferred return is beside the declaring
file** (`ReturnComponentOf` → `besideDeclaring`), as a bare return type already was:
`return new Expectation()` in a base class read from the caller's directory named nothing.
**`Index.FindFilesByBasename` returns a file's real case** from `fileURIs`: a component with no
functions had only its lowercased key, which names nothing on a case-sensitive filesystem, and
the extends walk stopped at Lucee's empty `LuceeTestCase.cfc`.

**Case-insensitive path resolution** (`internal/path/path.go`): `match`/`prefix` matching
(`indexFold`, `EqualFold`, `(?i)`-compiled regexes) has always been case-insensitive. Turning a
resolved dot-path into an actual `.cfc` file is also fully case-insensitive at every path
segment, not just the filename: `ResolvePath` walks the path one directory at a time via
`resolveSegments`, matching each segment against a real directory listing (exact case first,
then `EqualFold`), and `mappings` keys are matched case-insensitively too (`lookupFold`). This
matters because `os.Stat` alone can't be trusted: on a case-insensitive filesystem (APFS, NTFS
defaults) `Stat` succeeds for any case variant, silently returning the *requested* case rather
than the real on-disk case — which breaks on ext4 and most Linux filesystems. Only a directory
listing reveals the true on-disk name on every platform.

**Resolver false-positive: prefix substring matching, and the `anchored` fix**

By default `ResolveFromCall` finds the resolver's `prefix` *anywhere* in the expression via
`indexFold`, not just at position 0. A resolver with prefix `"document"` will also fire when the
variable name merely contains "document" (e.g. `domobject_document`). The sub passed to the
pattern matcher starts at the prefix position, so it exactly matches the short pattern and
produces a wrong component — confidently, rather than declining.

`{"anchored": true}` on a resolver requires the prefix at position 0
(`config.Resolver.Anchored`/`parser.Resolver.Anchored`, enforced in `findPrefixPos`), so the
resolver only claims expressions that genuinely start with it. It is off by default because
unanchored matching is what makes a *call* resolver qualifier-insensitive: `getService("$1")`
is meant to fire on `VARIABLES._parent.getService("x")` too. Reach for `anchored` when the
resolver targets a variable name, or when a broad catch-all is producing wrong answers. The
older workarounds still apply where anchoring doesn't fit: rename the variable in the CFML
source, or add a more-specific resolver for the false-positive variable *earlier* in the array.

The same class of issue can appear *within* a single pipe-delimited `prefix`: unanchored,
`findPrefixPos` tries alternatives in the order written and returns the position of whichever is
found first in that order — not the earliest position in the expression, and not the alternative
that would let `match` succeed. If one alternative is a substring of another (e.g. `prefix:
"File|getFile"` against `getFile()`), the shorter one found first fixes the slice position (`sub
= "File()"`) and the longer never gets a chance. Order alternatives so a shorter potential
substring comes *after* the longer one — or set `anchored`, which makes the order irrelevant
since every alternative that matches matches at 0.

**Generic catch-all resolvers are the highest-risk case of this**, because they're deliberately
broad. `{"match": "get$1()", "resolve": "tassweb.packages.tass.${1:lower}", "prefix": "get"}` is
meant to cover a family of factory methods without one entry per name — but unanchored, `prefix:
"get"` is found inside *any* identifier containing "get", including ordinary getters unrelated
to the intended family, and including the `getDirectContent()` at the tail of
`VARIABLES._document.getDirectContent()`. A catch-all this broad will win over a
correct-but-absent answer (e.g. a generic `returntype="any"` that should fall through to `$any`)
essentially every time a same-named `getXxx()` exists anywhere in the workspace. Anchoring it
is usually the right call: the bare factory calls it was written for (`getPageTools()`) start at
position 0, and the chained getters it was never meant to claim do not. Otherwise the options
are narrowing the `match` regex to an explicit alternation of the real factory names, or
accepting the false positive and overriding it with a more specific resolver listed earlier.

`explain` names the resolver that fired (`match "...", prefix "...")`, so a wrong component can
be traced to the exact `componentResolvers` entry instead of just "a componentResolver".

## Expression mappings

`expressionMappings` is a flat `map[string]string` of runtime expression → static value
substitutions (e.g. `"#VARIABLES._core#": "packages.tass.core."`), applied to component-path
strings before resolution. Unlike `componentResolvers`, there is no `match`/regex support — each
key is matched with a plain `strings.Contains` and replaced with `strings.ReplaceAll`.

A key may list alternatives separated by `|` (e.g. `"#ROOT#|#LEGACY_ROOT#": "app."`), so several
runtime expressions collapsing to the same static value don't need separate entries. Each
alternative is checked and replaced independently — plain substring alternation, unrelated to
the regex-triggering `\` in `componentResolvers.match`. Implemented in
`internal/resolve/resolve.go: ComponentPath` and `internal/parser/result.go: replaceExpressions`.

Keys apply **longest first**, then by key (`parser.ExpressionMappingOrder`, which both
implementations use). They were applied in map order, so two overlapping keys — `#core#`
and `#core#legacy.` — replaced in either order from run to run, and one path named two
different components.

**Unmapped `#...#` expressions become `$any`, not literal garbage.** Any component-path string
captured from CFML source — `CreateObject("component", "...")`, `<cfinvoke component="...">`,
`new "..."()` — may contain a runtime `#...#` interpolation no `expressionMappings` entry covers
(e.g. `CreateObject("component", "tools.templates.#ARGUMENTS.template#.generator")`).
`replaceExpressions` runs its substitution pass unconditionally (even with zero
`expressionMappings` configured), and if the result still contains `#`, returns `"$any"` — so
`CanResolveCall` accepts calls through it silently, the same as any other "genuinely dynamic"
case, rather than surfacing the literal text as a nonsensical "not found in" component name.
This applies to `ComponentRef`s (`pr.ComponentRefs`/`funcRefsMap`) *and* to `CallSite.Component`
(`pr.Calls`/`funcCallsMap`) — the latter matters because a bare unassigned call's
`CallSite.Component` can be baked in at parse time (`tag_parser.go`'s `lookupComponentRef`),
before this pass would otherwise run.

**Bracket-indexed chain segments** (`REQUEST['a' & b & 'c'].method()`, `arr[i].method()`): the
script parser's chain-walking (`checkVarRHS`, `parseBodyVarDecl`, `parseBodyScopedVar`,
`checkAssignRef`, `checkBareCall`, `recordBareCallAndChain` in `script_parser.go`) skips a
`[...]` group via `skipBracketIndex()` (mirrors `skipParens()`) and, critically, **poisons** the
chain text with a literal `"[]"` marker (`REQUEST['a'&b&'c'].method()` → `"REQUEST[].method"`)
instead of silently dropping the bracket and treating the bare identifier before it as the
receiver. The marker can't collide with any real identifier or resolver `match`, so it reliably
falls through to an honest `variable 'REQUEST[]' has no component ref` — the alternative would
let `REQUEST[dynamicKey].method()` resolve as if it were plain `REQUEST.method()`, which is
wrong *and* confident. To make these fully silent, add a `noFollow` resolver matching the
marker, e.g. `{"match": "^REQUEST\\[\\]$", "resolve": "nocheck", "prefix": "REQUEST[]",
"noFollow": true}` — the parser deliberately does not do this automatically, since suppressing
vs. surfacing "genuinely dynamic" is a project-level judgment call.

**A CFML comment inside a tag's attribute list** is skipped when the tag's end is
found: `<cffunction name="real" <!--- <cfset p = getThing()><cfset p.gone()> --->
output="false">` once ended at the `>` inside the comment, and the rest of it was
scanned as live tags. `tagEndIndex` is the one scan for a tag's closing `>`; it
steps over quoted strings, and when `<!--` appears before the first `>` it
walks byte by byte (`tagEndWithComments`), skipping comments and quotes. The
test is one `Contains` over the short prefix, so the common tag keeps its
two-count fast path. A `--->` inside a quoted string still ends a comment
early, on purpose: tree-sitter and an engine do the same, so do not "fix" it.
`TestACommentBetweenTagAttributesDoesNotEndTheTag` and `TestTagEndSkipsComments`
pin it. Routed through it: the walk, `<cfargument>` and the `<cfscript>` open
tag. **A walk over a whole document asks through `tagEnds` (`tag_ends.go`)**,
which gives `tagEndIndex`'s answer at every offset without each answer scanning
to the tag's end: past the last `>` there is none, the quote-aware walk
remembers each position between quoted values it passed (two walks there end
alike), and after `longSpansBeforeIndex` tags whose `>` is over 4KB away it
indexes the next `>`, next comment and quote counts once. Without it
`<cfset x="a>b">` repeated to 560KB took 19.7s. It restates `tagEndIndex`'s
fast path, comment check and walk, so a change to one is a change to both;
`TestTagEndsAgreesWithTagEndIndex` compares them at every offset. Every
document-level walk uses it: the tag walk, output-context tracking, managed
metadata, script skip spans, `<cfargument>`, collection writes and folding.
Still by a bare `IndexByte`: the `</cffunction>` close in `cfparser.go`
and the cursor lookups in `edit_parser.go`/`tags.go`, where the text is a
closer or a position inside an edit rather than a tag being read.

## `noFollow` flag

When a resolver has `"noFollow": true`, `CanResolveCall` accepts the call immediately without
verifying the method exists in the resolved component
(`config.Resolver.NoFollow`/`parser.Resolver.NoFollow`). Use it for dynamic factory methods
where the resolved component is approximate, Java objects whose stub coverage is incomplete, or
any pattern where "this variable came from X" is enough and method checking is noise.
`CanResolveCall` checks it at three points: the primary variable resolver, the full-line-text
fallback resolver, and the altComp fallback resolver.

## Triaging "method not found" / "no component ref" lint errors

| Error form | Meaning | Fix |
|---|---|---|
| `extends chain breaks at 'X', which does not resolve` / `X (base component does not resolve; N inherited calls not checked)` | A component in the file's extends chain names no file, so an inherited call — bare, `this.`, `super.`, or on a receiver the file never declares (`print`, `$assert`) — was never checked. `unresolved` reports it once per file, on the `extends` line, and totals the bases after the list | Add the `mappings` entry or `workspacePaths` directory that makes `X` resolve. `resolve.MissingBase` finds the link; `inheritedFromMissingBase` decides a receiver is the base's when neither the file nor the enclosing function's `FuncVars` declares it — closure locals and parameters included. A call on a component whose *own* chain breaks gets the same reason (`componentMissingBase`), and `unresolved` groups those per file and base, on the first call |
| `chained on 'f', which is not found` / `method 'f' has no component return type (chain to 'g')` | `f().g()`: the call is on what `f` returns (`resolveBareChain`), and `f` is not found as a bare call, or declares no component return type | Declare `f`'s return type; a `return this;` body is typed by itself (`returnsThis`) |
| `variable 'x' has no component ref` | Parser never established what component `x` is | Add a `componentResolver` covering the RHS of the assignment or the variable name. An injected property is typed by its id without one (`injectedComponent`: `inject="X@module"`, `id:X`, `model:X`, `provider:X` → `X`, found by path then file name; the DSL's `coldbox:…`, `wirebox:…`, `logbox:…`, `cachebox:…` → ColdBox's class); add a `beanPaths` entry when the id is not the file's name |
| `method 'f' not found in pkg.path` | Component is known but the method is missing | Add the method to the stub CFC at that path, or `"noFollow": true` on the resolver |
| `method 'f' not found in persist` | `persist` resolves correctly but the method is absent | Genuinely missing from the real CFC — implement it |
| Dynamic keys: `x[y].f`, `arr[i].f` | Type can't be tracked through runtime keys | Not fixable with resolvers; suppress with `noFollow` on the resolver that produces `x` |
| `ARGUMENTS.x` with `type="any"` | Argument has no type annotation | Add `type="pkg.path"` to `<cfargument>`, add an `ARGUMENTS.x` exact-match resolver, or document it with a `@serviceproperty` comment if `servicePropertyResolvers` is configured |
| One component, many unrelated-looking missing methods (e.g. 50+ hits all "not found in studadmin") | `CanResolveCall`'s file-level fallback (`fileLevelRef`) and its function-scoped lookup (`funcScopedRef`, which took the first until `var x` redeclared per `<cfcase>` was checked against the first branch's component) both pick the `ComponentRef` with the highest line number *at or before* the call site (`internal/resolve/resolve.go`), falling back to file order only for a genuine forward reference — so a scratch variable reassigned per `<cfcase>`/`<cfif>` branch is read against the assignment actually in scope. A large single-component cluster is therefore real, unless the reassignments sit inside a construct the parser doesn't line-order the way the runtime does; confirm with `explain`, which prints the line of the ref it chose. |
| A resolver with `"resolve": "nocheck", "noFollow": true` doesn't suppress the check | The resolver's `match` includes `(...)` (a call expression) rather than a bare variable name | `noFollow` only survives when the resolver re-runs live inside `CanResolveCall` (bare-word `ResolveFromCallFull(variable, ...)` lookups). A call-expression match fires once at parse time and is baked into `ComponentRef.Component`, which has no `NoFollow` field. Use `"resolve": "$any"` instead (checked unconditionally) for any resolver whose `match` contains parens. |

## Java stubs

Stubs are plain CFCs containing empty function stubs, mirroring the Java package path, so the
LSP can verify method calls without running Java. Single-line format, matching existing files:

```cfml
// Java stub for com.example.ClassName
component { function init(...) {} function methodName(required type argName) {} }
```

Set `"javaStubsPath": "<dot.path.to.stubs>"` to auto-resolve any `createObject("java",
"some.Class.Name")` to `<javaStubsPath>.some.Class.Name` without hand-writing the equivalent
regex resolver — it's synthesized and appended alongside your own `componentResolvers`
(`config.JavaStubResolver`, wired in `config.Resolve`, which every reader of a config goes through). It covers the `createObject("java",
...)` call site and Lucee's `new java:some.Class.Name()` — the latter because
`readNewComponent` recognises the `java:` type prefix and re-spells it as the equivalent
`createObject` expression before resolving, rather than a second resolver existing for it.
Chained factory calls (e.g. `someJavaObj.getInstance()` returning another instance) still need
their own resolver entry or a stub method modelling the return.

`new cfml:a.b.C()` is the sibling case and needs no configuration — the prefix is dropped and
`a.b.C` resolves as an ordinary CFC path. Both prefixes are handled in one place; the three
`new`-reading paths (`parseNewRef`, `parseStandaloneNew`, `checkReturnComponent`) all route
through `readNewComponent`.

**Two resolver shapes for struct-member Java handles.** Code that stores handles as struct keys
(`VARIABLES.itextObj.Foo = getJavaClass("Foo","itext")`) can't be tracked by the parser, so:

1. **Assignment RHS** — `var local = itextObj.Foo.init(...)`. Use `match: "itextObj.Foo.$1"`.
   The `$1` matches whatever follows the dot; since `resolve` has no `$1` it is discarded. The
   parser's `resolveCall(rhs)` picks this up and creates a `ComponentRef` for the local variable.
2. **Direct call** — `itextObj.Foo.bar(...)` where the variable *is* `itextObj.Foo`. Use
   `match: "itextObj.Foo"` (exact, no `$1`); `CanResolveCall` passes the raw variable name to
   `ResolveFromCall`, which exact-matches it.

Some handles need both shapes; others only one, depending on how the code uses them.
