# tb-p — unresolved findings and candidates

Every finding below is still reported. A candidate is a liberal match offered for a person to weigh, never an answer the resolver took: **high** is the only component declaring every method the function calls on the receiver (or the only one, named like it); **medium** is a sole match on weaker evidence, or the one named like the receiver among several; **low** is one of several. *Defined* is how many indexed files declare a method of that name: 0 means it is missing from the workspace, more means the resolver could not connect the call to it.

| Category | Findings | high | medium | low | none | method defined nowhere |
|---|---:|---:|---:|---:|---:|---:|
| variable | 293 | 19 | 49 | 9 | 216 | 157 |
| return-type | 16 | 3 | 7 | 1 | 5 | 5 |
| method | 59 | 0 | 47 | 4 | 8 | 8 |
| object | 12 | 0 | 0 | 0 | 12 | 10 |

## Variable definitions — a receiver whose component is unknown

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 23 | `controller → tests/resources/coldbox/system/FrameworkSupertype.cfc` | 2 | low | declares getSetting() (2 candidates) | `tests/resources/coldbox/system/FrameworkSupertype.cfc:19` variable 'controller' has no component ref |
| 22 | `out` | 1 | none |  | `system/reports/ANTJUnitReporter.cfc:112` variable 'out' has no component ref |
| 20 | `arguments.actual` | 0 | none |  | `system/Assertion.cfc:1545` variable 'arguments.actual' has no component ref |
| 12 | `arguments.callbacks` | 0 | none |  | `system/TestBox.cfc:844` variable 'arguments.callbacks' has no component ref |
| 8 | `response` | 0 | none |  | `system/TestBox.cfc:688` variable 'response' has no component ref |
| 7 | `arguments.baserunner → system/runners/BaseRunner.cfc` | 1 | high | declares canRunSuite(); named like the receiver 'baseRunner' | `system/TestBox.cfc:1104` variable 'arguments.baseRunner' has no component ref |
| 7 | `modulerecord.moduleconfig → system/util/MixerUtil.cfc` | 1 | medium | declares injectPropertyMixin() | `system/TestBox.cfc:193` variable 'moduleRecord.moduleConfig' has no component ref |
| 7 | `udfout` | 1 | none |  | `system/mockutils/MockGenerator.cfc:415` variable 'udfOut' has no component ref |
| 6 | `arguments.path` | 0 | none |  | `system/TestBox.cfc:115` variable 'arguments.path' has no component ref |
| 6 | `buffer` | 1 | none |  | `system/util/XMLConverter.cfc:230` variable 'buffer' has no component ref |
| 6 | `path` | 0 | none |  | `build/Build.cfc:264` variable 'path' has no component ref |
| 5 | `arguments.absolutepath` | 0 | none |  | `system/coverage/data/TemplateCompiler_Lucee.cfc:71` variable 'arguments.absolutePath' has no component ref |
| 5 | `arguments.controller` | 1 | none |  | `tests/resources/coldbox/system/EventHandler.cfc:1` variable 'arguments.controller' has no component ref |
| 5 | `arguments.expected` | 0 | none |  | `system/Assertion.cfc:1742` variable 'arguments.expected' has no component ref |
| 5 | `arguments.runner` | 1 | none |  | `system/BaseSpec.cfc:1093` variable 'arguments.runner' has no component ref |
| 5 | `arguments.target` | 0 | none |  | `system/Assertion.cfc:2197` variable 'arguments.target' has no component ref |
| 5 | `controller` | 1 | none |  | `tests/resources/coldbox/system/FrameworkSupertype.cfc:96` variable 'controller' has no component ref |
| 4 | `arguments.receiver → system/MockBox.cfc` | 1 | medium | declares $property() | `system/compat/framework/TestCase.cfc:166` variable 'arguments.receiver' has no component ref |
| 4 | `arguments.spec` | 0 | none |  | `system/BaseSpec.cfc:1810` variable 'arguments.spec' has no component ref |
| 4 | `attributes.callbacks` | 0 | none |  | `system/runners/BDDRunner.cfc:265` variable 'attributes.callbacks' has no component ref |
| 4 | `md` | 0 | none |  | `system/Assertion.cfc:289` variable 'md' has no component ref |
| 4 | `modulerecord.moduleconfig` | 1 | none |  | `system/TestBox.cfc:191` variable 'moduleRecord.moduleConfig' has no component ref |
| 4 | `specmd` | 0 | none |  | `system/TestBox.cfc:1295` variable 'specMD' has no component ref |
| 4 | `targetmd` | 0 | none |  | `system/TestBox.cfc:1007` variable 'targetMD' has no component ref |
| 4 | `thread.target → system/BaseSpec.cfc` | 1 | medium | declares runSpec() | `system/runners/BDDRunner.cfc:284` variable 'thread.target' has no component ref |
| 4 | `variables.modules.registry` | 0 | none |  | `system/TestBox.cfc:134` variable 'variables.modules.registry' has no component ref |
| 3 | `arguments.runner → system/TestBox.cfc` | 1 | high | declares announceToModules(); named like the receiver 'TestBox' | `system/BaseSpec.cfc:1131` variable 'arguments.runner' has no component ref |
| 3 | `arguments.runner → system/runners/BaseRunner.cfc` | 1 | high | declares canRunLabel(), canRunSpec() | `system/BaseSpec.cfc:1438` variable 'arguments.runner' has no component ref |
| 3 | `arguments.target → tests/specs/BDDInheritanceTest.cfc` | 38 | low | declares run(), beforeAll(), afterAll() (19 candidates) | `system/runners/BDDRunner.cfc:51` variable 'arguments.target' has no component ref |
| 3 | `attributes.service → system/util/StreamingService.cfc` | 1 | high | declares queueEvent(), streamEvent() | `tests/specs/streaming/StreamingServiceTest.cfc:452` variable 'attributes.service' has no component ref |
| 3 | `item` | 0 | none |  | `build/Build.cfc:279` variable 'item' has no component ref |
| 3 | `server.coldfusion.productname` | 0 | none |  | `system/BaseSpec.cfc:1881` variable 'server.coldfusion.productName' has no component ref |
| 3 | `server.os.name` | 0 | none |  | `system/BaseSpec.cfc:1893` variable 'server.os.name' has no component ref |
| 3 | `spec` | 0 | none |  | `tests/specs/BDDLifecycleTest.cfc:34` variable 'spec' has no component ref |
| 3 | `thefile` | 0 | none |  | `system/coverage/data/CoverageGenerator.cfc:124` variable 'theFile' has no component ref |
| 3 | `variables.spec → system/BaseSpec.cfc` | 1 | medium | declares expect() | `system/Expectation.cfc:1853` variable 'variables.spec' has no component ref |
| 2 | `amended[]` | 0 | none |  | `system/coverage/CoverageReporter.cfc:91` variable 'amended[]' has no component ref |
| 2 | `arguments.collection[]` | 0 | none |  | `system/reports/ANTJUnitReporter.cfc:272` variable 'arguments.collection[]' has no component ref |
| 2 | `arguments.key` | 0 | none |  | `system/Assertion.cfc:430` variable 'arguments.key' has no component ref |
| 2 | `arguments.scope` | 0 | none |  | `system/MockBox.cfc:184` variable 'arguments.scope' has no component ref |
| 2 | `arguments.suite → system/BaseSpec.cfc` | 1 | medium | declares beforeEach() | `system/BaseSpec.cfc:1259` variable 'arguments.suite' has no component ref |
| 2 | `backpath` | 0 | none |  | `cfml/browser/index.cfm:42` variable 'backPath' has no component ref |
| 2 | `bundlestats` | 0 | none |  | `system/util/StreamingService.cfc:236` variable 'bundleStats' has no component ref |
| 2 | `consolidatedlabels` | 1 | none |  | `system/BaseSpec.cfc:1058` variable 'consolidatedLabels' has no component ref |
| 2 | `line → system/Assertion.cfc` | 1 | medium | declares startsWith() | `system/coverage/data/CoverageGenerator.cfc:199` variable 'line' has no component ref |
| 2 | `local.thisspec.failorigin` | 0 | none |  | `system/reports/assets/codexwiki.cfm:64` variable 'local.thisSpec.failOrigin' has no component ref |
| 2 | `matchingstats` | 0 | none |  | `system/util/StreamingService.cfc:241` variable 'matchingStats' has no component ref |
| 2 | `nextclosure` | 0 | none |  | `system/BaseSpec.cfc:1357` variable 'nextClosure' has no component ref |
| 2 | `stats.error.stacktrace` | 0 | none |  | `system/reports/ANTJUnitReporter.cfc:221` variable 'stats.error.stackTrace' has no component ref |
| 2 | `this.$expectedexception` | 0 | none |  | `system/BaseSpec.cfc:1457` variable 'this.$expectedException' has no component ref |
| 2 | `this.mockbox → system/MockBox.cfc` | 1 | high | declares normalizeArguments(); named like the receiver 'mockBox' | `system/MockBox.cfc:396` variable 'this.mockBox' has no component ref |
| 2 | `wirebox` | 0 | none |  | `tests/resources/coldbox/system/FrameworkSupertype.cfc:80` variable 'wirebox' has no component ref |
| 1 | `agg` | 1 | none |  | `tests/specs/coverage/CoverageReporterTest.cfc:90` variable 'agg' has no component ref |
| 1 | `amended` | 0 | none |  | `system/coverage/CoverageReporter.cfc:113` variable 'amended' has no component ref |
| 1 | `appmetadata.customtagpaths` | 0 | none |  | `system/util/CFMappingHelper.cfc:23` variable 'appMetadata.customTagPaths' has no component ref |
| 1 | `appsettings.mappings` | 1 | none |  | `system/util/LuceeMappingHelper.cfc:46` variable 'appSettings.mappings' has no component ref |
| 1 | `argorderedtree[]` | 0 | none |  | `system/MockBox.cfc:645` variable 'argOrderedTree[]' has no component ref |
| 1 | `arguments.arglist` | 0 | none |  | `system/util/MixerUtil.cfc:193` variable 'arguments.argList' has no component ref |
| 1 | `arguments.blacklist` | 0 | none |  | `system/coverage/data/CoverageGenerator.cfc:111` variable 'arguments.blacklist' has no component ref |
| 1 | `arguments.bundlestats.globalexception.stacktrace` | 0 | none |  | `system/reports/ANTJUnitReporter.cfc:121` variable 'arguments.bundleStats.globalException.stackTrace' has no component ref |
| 1 | `arguments.md.extends` | 0 | none |  | `system/mockutils/MockGenerator.cfc:461` variable 'arguments.md.extends' has no component ref |
| 1 | `arguments.targetobject → system/mockutils/MockGenerator.cfc` | 1 | medium | declares $include() | `system/mockutils/MockGenerator.cfc:274` variable 'arguments.targetObject' has no component ref |
| 1 | `arguments.whitelist` | 0 | none |  | `system/coverage/data/CoverageGenerator.cfc:108` variable 'arguments.whitelist' has no component ref |
| 1 | `bundlesfound[]` | 0 | none |  | `system/TestBox.cfc:972` variable 'bundlesFound[]' has no component ref |
| 1 | `config.moduleconfig → tests/resources/testModule/ModuleConfig.cfc` | 1 | high | declares onUnload(); named like the receiver 'moduleConfig' | `system/TestBox.cfc:498` variable 'config.moduleConfig' has no component ref |
| 1 | `controller → tests/resources/coldbox/system/EventHandler.cfc` | 2 | medium | declares getWirebox() | `tests/resources/coldbox/system/FrameworkSupertype.cfc:45` variable 'controller' has no component ref |
| 1 | `currentcoverage` | 0 | none |  | `system/coverage/CoverageReporter.cfc:85` variable 'currentCoverage' has no component ref |
| 1 | `filedata.linedata` | 0 | none |  | `system/coverage/browser/CodeBrowser.cfc:89` variable 'fileData.lineData' has no component ref |
| 1 | `filedata.relativefilepath` | 0 | none |  | `system/coverage/browser/CodeBrowser.cfc:95` variable 'fileData.relativeFilePath' has no component ref |
| 1 | `filenode.xmlchildren` | 1 | none |  | `system/coverage/sonarQube/SonarQube.cfc:48` variable 'fileNode.XMLChildren' has no component ref |
| 1 | `idata.type → system/reports/ANTJUnitReporter.cfc` | 16 | low | declares runReport() (16 candidates) | `system/TestBox.cfc:739` variable 'iData.type' has no component ref |
| 1 | `item → system/BaseSpec.cfc` | 1 | medium | declares beforeEach() | `system/BaseSpec.cfc:1255` variable 'item' has no component ref |
| 1 | `navigator` | 0 | none |  | `system/Assertion.cfc:2243` variable 'navigator' has no component ref |
| 1 | `obj2 → tests/resources/CallPrivate.cfc` | 1 | medium | declares callIt() | `tests/specs/MXUnitCompatTest.cfc:173` variable 'obj2' has no component ref |
| 1 | `omockgenerator → system/mockutils/MockGenerator.cfc` | 1 | high | declares generate(); named like the receiver 'oMockGenerator' | `system/MockBox.cfc:476` variable 'oMockGenerator' has no component ref |
| 1 | `opts.browser.outputdir → system/Assertion.cfc` | 1 | medium | declares endsWith() | `system/coverage/CoverageService.cfc:170` variable 'opts.browser.outputDir' has no component ref |
| 1 | `opts.pathtocapture → system/Assertion.cfc` | 1 | medium | declares endsWith() | `system/coverage/CoverageService.cfc:218` variable 'opts.pathToCapture' has no component ref |
| 1 | `parentsuite → system/BaseSpec.cfc` | 1 | medium | declares afterEach() | `system/BaseSpec.cfc:1402` variable 'parentSuite' has no component ref |
| 1 | `rootnode.xmlchildren` | 1 | none |  | `system/coverage/sonarQube/SonarQube.cfc:50` variable 'rootNode.XMLChildren' has no component ref |
| 1 | `row` | 0 | none |  | `system/coverage/CoverageReporter.cfc:119` variable 'row' has no component ref |
| 1 | `runner → system/runners/BaseRunner.cfc` | 1 | medium | declares isSuiteFocused() | `system/BaseSpec.cfc:1080` variable 'runner' has no component ref |
| 1 | `stats.failorigin` | 0 | none |  | `system/reports/JUnitReporter.cfc:158` variable 'stats.failorigin' has no component ref |
| 1 | `targetpath` | 0 | none |  | `cfml/browser/index.cfm:126` variable 'targetPath' has no component ref |
| 1 | `testbundles` | 0 | none |  | `system/runners/BaseRunner.cfc:206` variable 'testBundles' has no component ref |
| 1 | `thisbundle` | 0 | none |  | `system/reports/assets/simple.cfm:261` variable 'thisBundle' has no component ref |
| 1 | `thisfunction` | 0 | none |  | `system/util/Util.cfc:180` variable 'thisFunction' has no component ref |
| 1 | `thisfunction.annotations` | 0 | none |  | `system/util/Util.cfc:180` variable 'thisFunction.annotations' has no component ref |
| 1 | `thisparam` | 1 | none |  | `system/mockutils/MockGenerator.cfc:129` variable 'thisParam' has no component ref |
| 1 | `url` | 0 | none |  | `tests/Application.cfc:30` variable 'url' has no component ref |
| 1 | `url.path` | 0 | none |  | `cfml/browser/index.cfm:41` variable 'url.path' has no component ref |
| 1 | `variables.bundlespattern` | 0 | none |  | `system/TestBox.cfc:957` variable 'variables.bundlesPattern' has no component ref |
| 1 | `variables.excludes` | 0 | none |  | `build/Build.cfc:263` variable 'variables.excludes' has no component ref |
| 1 | `variables.logbox` | 1 | none |  | `tests/resources/coldbox/system/EventHandler.cfc:1` variable 'variables.logBox' has no component ref |

<details><summary>Groups with several candidates</summary>

- `controller → tests/resources/coldbox/system/FrameworkSupertype.cfc` — 23 finding(s), 2 candidate(s):
  - low `tests/resources/coldbox/system/FrameworkSupertype.cfc` — declares getSetting()
  - low `tests/resources/Test.cfc` — declares getSetting()
- `arguments.target → tests/specs/BDDInheritanceTest.cfc` — 3 finding(s), 19 candidate(s):
  - low `tests/specs/BDDInheritanceTest.cfc` — declares run(), beforeAll(), afterAll()
  - low `tests/specs/BDDLifecycleAnnotationsTest.cfc` — declares run(), beforeAll(), afterAll()
  - low `tests/specs/BDDLifecycleTest.cfc` — declares run(), beforeAll(), afterAll()
  - low `tests/specs/BDDTest.cfc` — declares run(), beforeAll(), afterAll()
  - low `tests/specs/CustomAssertions.cfc` — declares run(), beforeAll(), afterAll()
  - low `tests/specs/DebugTests.cfc` — declares run(), beforeAll(), afterAll()
  - low `tests/specs/ExcludesTest.cfc` — declares run(), beforeAll(), afterAll()
  - low `tests/specs/FocusedSpecs.cfc` — declares run(), beforeAll(), afterAll()
  - … 11 more
- `idata.type → system/reports/ANTJUnitReporter.cfc` — 1 finding(s), 16 candidate(s):
  - low `system/reports/ANTJUnitReporter.cfc` — declares runReport()
  - low `system/reports/CodexWikiReporter.cfc` — declares runReport()
  - low `system/reports/ConsoleReporter.cfc` — declares runReport()
  - low `system/reports/DocReporter.cfc` — declares runReport()
  - low `system/reports/DotReporter.cfc` — declares runReport()
  - low `system/reports/IReporter.cfc` — declares runReport()
  - low `system/reports/JSONReporter.cfc` — declares runReport()
  - low `system/reports/JUnitReporter.cfc` — declares runReport()
  - … 8 more

</details>

## Return types — a call chained on a method that declares no component

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 4 | `method 'getUtility' has no component return type → system/util/Util.cfc` | 1 | medium | declares getAnnotatedMethods() | `system/BaseSpec.cfc:1238` method 'getUtility' has no component return type (chain to 'getAnnotatedMethods') |
| 3 | `method 'getJavaSystem' has no component return type → system/BaseSpec.cfc` | 1 | high | declares getProperty(), getEnv() | `system/util/Env.cfc:15` method 'getJavaSystem' has no component return type (chain to 'getProperty') |
| 2 | `method 'getBundleStats' in testbox.system.TestResult has no component return type` | 0 | none |  | `system/TestBox.cfc:473` method 'getBundleStats' in testbox.system.TestResult has no component return type (chain to 'filter') |
| 1 | `method 'getActiveModules' has no component return type` | 0 | none |  | `system/TestBox.cfc:804` method 'getActiveModules' has no component return type (chain to 'each') |
| 1 | `method 'getCBMockData' has no component return type → system/compat/framework/TestCase.cfc` | 1 | medium | declares mock() | `system/BaseSpec.cfc:1684` method 'getCBMockData' has no component return type (chain to 'mock') |
| 1 | `method 'getCWD' has no component return type` | 0 | none |  | `build/Build.cfc:12` method 'getCWD' has no component return type (chain to 'reReplace') |
| 1 | `method 'getJavaSystem' has no component return type → system/util/Env.cfc` | 4 | low | declares getEnv() (3 candidates) | `system/util/Env.cfc:68` method 'getJavaSystem' has no component return type (chain to 'getEnv') |
| 1 | `method 'getUtility' has no component return type → system/util/MixerUtil.cfc` | 1 | high | declares start(); named like the receiver 'MixerUtil' | `system/BaseSpec.cfc:1643` method 'getUtility' has no component return type (chain to 'start') |
| 1 | `method 'makePublic' has no component return type` | 0 | none |  | `tests/specs/MXUnitCompatTest.cfc:169` method 'makePublic' has no component return type (chain to 'funkyMethod') |
| 1 | `method 'makePublic' has no component return type → tests/resources/test1.cfc` | 1 | medium | declares aPrivateMethod() | `tests/specs/MXUnitCompatTest.cfc:166` method 'makePublic' has no component return type (chain to 'aPrivateMethod') |

<details><summary>Groups with several candidates</summary>

- `method 'getJavaSystem' has no component return type → system/util/Env.cfc` — 1 finding(s), 3 candidate(s):
  - low `system/util/Env.cfc` — declares getEnv()
  - low `system/BaseSpec.cfc` — declares getEnv()
  - low `system/TestBox.cfc` — declares getEnv()

</details>

## Method definitions — a method not found where it was looked for

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 41 | `getstringname` | 1 | medium | declares getStringName; beside the calling file | `system/Expectation.cfc:421` no qualifier, not in file |
| 3 | `assertisfunky` | 1 | medium | declares assertIsFunky | `tests/specs/CustomAssertions.cfc:30` method 'assertIsFunky' not found in testbox.system.Assertion |
| 3 | `getclassmetadata` | 0 | none |  | `system/TestBox.cfc:454` no qualifier, not in file |
| 2 | `assertisawesome` | 1 | medium | declares assertIsAwesome | `tests/specs/CustomAssertions.cfc:26` method 'assertIsAwesome' not found in testbox.system.Assertion |
| 2 | `fail` | 4 | low | declares fail (4 candidates) | `tests/resources/CustomAsserts.cfc:4` no qualifier, not in file |
| 1 | `body` | 0 | none |  | `system/Expectation.cfc:48` no qualifier, not in file |
| 1 | `datanavigate` | 0 | none |  | `system/Assertion.cfc:2242` no qualifier, not in file |
| 1 | `exposemixin` | 1 | medium | declares exposeMixin | `system/BaseSpec.cfc:1645` method 'exposeMixin' not found in CallPrivate\|ComponentInDifferentPackage\|test1 |
| 1 | `exxpect` | 0 | none |  | `tests/specs/BDDTest.cfc:855` not found in extends chain |
| 1 | `getoptions` | 3 | low | declares getOptions (3 candidates) | `system/compat/runner/Results.cfc:62` no qualifier, not in file |
| 1 | `invalidfunction` | 0 | none |  | `tests/specs/AssertionsTest.cfc:28` not found in extends chain |
| 1 | `run` | 38 | low | declares run (36 candidates) | `cfml/browser/index.cfm:29` no qualifier, not in file |
| 1 | `virtualreturn` | 0 | none |  | `tests/specs/mockbox/MockBoxTestHarness.cfm:33` method 'virtualReturn' not found in testbox.tests.resources.Test |

<details><summary>Groups with several candidates</summary>

- `fail` — 2 finding(s), 4 candidate(s):
  - low `system/Assertion.cfc:15` — declares fail
  - low `system/BaseSpec.cfc:75` — declares fail
  - low `system/Expectation.cfc:60` — declares fail
  - low `system/compat/framework/TestCase.cfc:199` — declares fail
- `getoptions` — 1 finding(s), 3 candidate(s):
  - low `system/TestBox.cfc:25` — declares getOptions
  - low `system/runners/BDDRunner.cfc:14` — declares getOptions
  - low `system/runners/UnitRunner.cfc:14` — declares getOptions
- `run` — 1 finding(s), 36 candidate(s):
  - low `build/Build.cfc:53` — declares run
  - low `cfml/tests/specs/BDDTest.cfc:31` — declares run
  - low `cfml/tests/specs/MyFirstSpec.cfc:3` — declares run
  - low `system/TestBox.cfc:333` — declares run
  - low `system/runners/BDDRunner.cfc:38` — declares run
  - low `system/runners/IRunner.cfc:24` — declares run
  - low `system/runners/UnitRunner.cfc:38` — declares run
  - low `tests/specs/AbstractClass.cfc:8` — declares run
  - … 28 more

</details>

## Object references — a component path that names no file

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 4 | `component 'testbox.system.modules.globber.models.PathPatternMatcher' does not exist (calling 'matchPatterns')` | 0 | none |  | `system/TestBox.cfc:957` component 'testbox.system.modules.globber.models.PathPatternMatcher' does not exist (calling 'matchPatterns') |
| 1 | `component '#arguments.bundlePath#' does not exist (calling 'getDebugBuffer')` | 1 | none |  | `system/TestBox.cfc:875` component '#arguments.bundlePath#' does not exist (calling 'getDebugBuffer') |
| 1 | `component '#arguments.bundlePath#' does not exist (calling 'run')` | 38 | none |  | `system/TestBox.cfc:1033` component '#arguments.bundlePath#' does not exist (calling 'run') |
| 1 | `component 'CFIDE.componentutils.cfcexplorer' does not exist (calling 'getcfcs')` | 0 | none |  | `system/compat/runner/DirectoryTestSuite.cfc:49` component 'CFIDE.componentutils.cfcexplorer' does not exist (calling 'getcfcs') |
| 1 | `component 'CFIDE.componentutils.cfcexplorer' does not exist (calling 'normalizePath')` | 0 | none |  | `system/compat/runner/DirectoryTestSuite.cfc:48` component 'CFIDE.componentutils.cfcexplorer' does not exist (calling 'normalizePath') |
| 1 | `component 'testbox.system.modules.cbstreams.models.StreamBuilder' does not exist (chain hop 'new' to 'map')` | 0 | none |  | `system/coverage/browser/CodeBrowser.cfc:65` component 'testbox.system.modules.cbstreams.models.StreamBuilder' does not exist (chain hop 'new' to 'map') |
| 1 | `component 'testbox.system.modules.cbstreams.models.StreamBuilder' does not exist (chain hop 'new' to 'peek')` | 0 | none |  | `system/coverage/browser/CodeBrowser.cfc:68` component 'testbox.system.modules.cbstreams.models.StreamBuilder' does not exist (chain hop 'new' to 'peek') |
| 1 | `component 'testbox.system.modules.cbstreams.models.StreamBuilder' does not exist (chain hop 'new' to 'rangeClosed')` | 0 | none |  | `system/coverage/browser/CodeBrowser.cfc:64` component 'testbox.system.modules.cbstreams.models.StreamBuilder' does not exist (chain hop 'new' to 'rangeClosed') |
| 1 | `component 'testbox.system.modules.cbstreams.models.StreamBuilder' does not exist (chain hop 'new' to 'toArray')` | 0 | none |  | `system/coverage/browser/CodeBrowser.cfc:75` component 'testbox.system.modules.cbstreams.models.StreamBuilder' does not exist (chain hop 'new' to 'toArray') |
