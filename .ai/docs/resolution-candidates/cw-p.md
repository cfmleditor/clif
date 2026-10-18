# cw-p — unresolved findings and candidates

Every finding below is still reported. A candidate is a liberal match offered for a person to weigh, never an answer the resolver took: **high** is the only component declaring every method the function calls on the receiver (or the only one, named like it); **medium** is a sole match on weaker evidence, or the one named like the receiver among several; **low** is one of several. *Defined* is how many indexed files declare a method of that name: 0 means it is missing from the workspace, more means the resolver could not connect the call to it.

| Category | Findings | high | medium | low | none | method defined nowhere |
|---|---:|---:|---:|---:|---:|---:|
| variable | 1867 | 375 | 296 | 605 | 591 | 403 |
| return-type | 133 | 14 | 39 | 65 | 15 | 15 |
| method | 278 | 0 | 136 | 97 | 45 | 37 |
| object | 214 | 0 | 103 | 2 | 109 | 3 |

## Variable definitions — a receiver whose component is unknown

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 81 | `local.stmt` | 0 | none |  | `cli/src/commands/wheels/base.cfc:1090` variable 'local.stmt' has no component ref |
| 66 | `local.rs` | 0 | none |  | `cli/src/commands/wheels/base.cfc:1091` variable 'local.rs' has no component ref |
| 58 | `application.wheels.engineadapter → vendor/wheels/engineAdapters/Base.cfc` | 2 | low | declares parseFormKey() (2 candidates) | `vendor/wheels/Dispatch.cfc:150` variable 'application.wheels.engineAdapter' has no component ref |
| 53 | `variables.helpers → cli/lucli/services/Helpers.cfc` | 7 | low | declares pluralize(), capitalize(); named like the receiver 'helpers' (4 candidates) | `cli/lucli/services/Admin.cfc:31` variable 'variables.helpers' has no component ref |
| 45 | `migration.adapter → vendor/wheels/databaseAdapters/CockroachDB/CockroachDBMigrator.cfc` | 7 | low | declares adapterName() (7 candidates) | `vendor/wheels/tests/specs/database/CockroachDBCrudSpec.cfc:11` variable 'migration.adapter' has no component ref |
| 36 | `variables.config → cli/lucli/services/deploy/config/Config.cfc` | 1 | high | declares destination(); named like the receiver 'config' | `cli/lucli/services/deploy/commands/AccessoryCommands.cfc:72` variable 'variables.config' has no component ref |
| 32 | `binder → vendor/wheels/Injector.cfc` | 3 | low | declares map() (3 candidates) | `cli/src/ModuleConfig.cfc:44` variable 'binder' has no component ref |
| 30 | `ssh → cli/lucli/services/deploy/lib/SshClient.cfc` | 1 | high | declares uploadString(), run() | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:188` variable 'ssh' has no component ref |
| 29 | `adapter → vendor/wheels/engineAdapters/Base.cfc` | 2 | low | declares invokeMethod() (2 candidates) | `vendor/wheels/tests/specs/dispatch/InvokeMethodSpec.cfc:22` variable 'adapter' has no component ref |
| 29 | `state.publiccfc → vendor/wheels/Public.cfc` | 1 | high | declares $cliCommandIsMutating(), $cliMutationGateCheck(), $cliResolveDumpPath(), $cliFormatMigrationStatus(), $cliDatabaseType() | `vendor/wheels/tests/specs/security/CliEndpointHardeningSpec.cfc:70` variable 'state.publicCfc' has no component ref |
| 24 | `ctx.g → vendor/wheels/Global.cfc` | 1 | high | declares $cachedModelLookup(), $cachedControllerLookup(), model(), controller() | `vendor/wheels/tests/specs/global/cachedLookupSpec.cfc:27` variable 'ctx.g' has no component ref |
| 23 | `local.conn` | 0 | none |  | `cli/src/commands/wheels/base.cfc:1205` variable 'local.conn' has no component ref |
| 22 | `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/Base.cfc` | 7 | low | declares $quoteIdentifier() (7 candidates) | `vendor/wheels/model/miscellaneous.cfc:227` variable 'variables.wheels.class.adapter' has no component ref |
| 20 | `mapper → vendor/wheels/mapper/mapping.cfc` | 1 | medium | declares $draw() | `vendor/wheels/tests/specs/global/urlforSpec.cfc:29` variable 'mapper' has no component ref |
| 20 | `variables.sshpool → cli/lucli/services/deploy/lib/SshPool.cfc` | 2 | medium | declares onEach(); named like the receiver 'sshPool' (2 candidates) | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:119` variable 'variables.sshPool' has no component ref |
| 19 | `arguments.injector → vendor/wheels/Injector.cfc` | 3 | low | declares to() (3 candidates) | `vendor/wheels/Bindings.cfc:10` variable 'arguments.injector' has no component ref |
| 18 | `arguments.conn` | 0 | none |  | `cli/src/commands/wheels/base.cfc:1088` variable 'arguments.conn' has no component ref |
| 17 | `arguments.accessory → cli/lucli/services/deploy/config/Accessory.cfc` | 1 | high | declares containerName(), image(), cmd(); named like the receiver 'accessory' | `cli/lucli/services/deploy/commands/AccessoryCommands.cfc:21` variable 'arguments.accessory' has no component ref |
| 17 | `ssh → cli/lucli/services/TestRunner.cfc` | 637 | low | declares run() (633 candidates) | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:119` variable 'ssh' has no component ref |
| 16 | `variables.modelreference → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/model/query/QueryBuilder.cfc:338` variable 'variables.modelReference' has no component ref |
| 15 | `variables.modelreference → vendor/wheels/model/query/QueryBuilder.cfc` | 4 | low | declares findOne() (4 candidates) | `vendor/wheels/model/query/QueryBuilder.cfc:360` variable 'variables.modelReference' has no component ref |
| 14 | `arguments.context.migrator → vendor/wheels/Migrator.cfc` | 3 | medium | declares createMigration(); named like the receiver 'migrator' (3 candidates) | `vendor/wheels/public/CliBridge.cfc:107` variable 'arguments.context.migrator' has no component ref |
| 14 | `local.associatedclass → vendor/wheels/Model.cfc` | 1 | high | declares $classData(), $softDeletion(), $softDeleteColumn(), tableName() | `vendor/wheels/model/sql.cfc:1426` variable 'local.associatedClass' has no component ref |
| 14 | `stack.cli → cli/lucli/services/packages/PackagesMainCli.cfc` | 3 | high | declares list(), search(), show(), install(), remove() | `cli/lucli/tests/specs/packages/PackagesMainCliSpec.cfc:73` variable 'stack.cli' has no component ref |
| 14 | `user.birthtime` | 0 | none |  | `vendor/wheels/tests/specs/model/propertiesSpec.cfc:110` variable 'user.birthTime' has no component ref |
| 13 | `out` | 1 | none |  | `vendor/wheels/wheelstest/system/reports/ANTJUnitReporter.cfc:111` variable 'out' has no component ref |
| 13 | `variables.page` | 0 | none |  | `vendor/wheels/wheelstest/BrowserClient.cfc:39` variable 'variables.page' has no component ref |
| 12 | `application[].pluginobj → vendor/wheels/Plugins.cfc` | 1 | high | declares getPlugins(), getPluginMeta(), getIncompatiblePlugins(), getDependantPlugins(), getVersionMismatchPlugins(), getMixinCollisions(), getMixins(), getP… | `vendor/wheels/Global.cfc:3690` variable 'application[].PluginObj' has no component ref |
| 12 | `details → cli/src/models/DetailOutputService.cfc` | 2 | medium | declares error() | `cli/src/commands/wheels/generate/route.cfc:142` variable 'details' has no component ref |
| 12 | `local.response` | 0 | none |  | `vendor/wheels/Global.cfc:2501` variable 'local.response' has no component ref |
| 12 | `migrator → vendor/wheels/Migrator.cfc` | 1 | high | declares getAvailableMigrations(), getCurrentMigrationVersion(); named like the receiver 'migrator' | `cli/lucli/services/MigrationRunner.cfc:46` variable 'migrator' has no component ref |
| 12 | `role → cli/lucli/services/deploy/config/Role.cfc` | 2 | medium | declares hosts(); named like the receiver 'role' (2 candidates) | `cli/lucli/services/deploy/cli/DeployBuildCli.cfc:140` variable 'role' has no component ref |
| 12 | `stack.cache → cli/lucli/services/packages/ManifestCache.cfc` | 7 | low | declares refresh() (7 candidates) | `cli/lucli/tests/specs/packages/PackagesMainCliSpec.cfc:76` variable 'stack.cache' has no component ref |
| 11 | `application.wheelsdi → vendor/wheels/Injector.cfc` | 3 | low | declares getInstance() (2 candidates) | `vendor/wheels/Global.cfc:1102` variable 'application.wheelsdi' has no component ref |
| 11 | `application[].packageloaderobj → vendor/wheels/PackageLoader.cfc` | 1 | high | declares getPackages(), getPackageMeta(), getFailedPackages(), getMixinCollisions(), getMixins(), getMethodProviders(), getPackageMiddleware(), $hasServicePr… | `vendor/wheels/Global.cfc:3733` variable 'application[].PackageLoaderObj' has no component ref |
| 11 | `map → vendor/wheels/mapper/resources.cfc` | 2 | low | declares resources(), resource() (2 candidates) | `vendor/wheels/tests/specs/mapper/NestedResourcesSpec.cfc:24` variable 'map' has no component ref |
| 10 | `arguments.callbacks` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:757` variable 'arguments.callbacks' has no component ref |
| 10 | `arguments.cfg → cli/lucli/services/deploy/config/Config.cfc` | 1 | medium | declares roles() | `cli/lucli/services/deploy/cli/DeployBuildCli.cfc:139` variable 'arguments.cfg' has no component ref |
| 10 | `codegen → cli/lucli/services/CodeGen.cfc` | 2 | medium | declares validateName(), generateModel(); named like the receiver 'codegen' (2 candidates) | `cli/lucli/Module.cfc:3557` variable 'codegen' has no component ref |
| 10 | `mapper → vendor/wheels/mapper/matching.cfc` | 1 | high | declares $match(), wildcard() | `vendor/wheels/tests/specs/global/urlforSpec.cfc:31` variable 'mapper' has no component ref |
| 10 | `response` | 0 | none |  | `cli/lucli/tests/StubHttpServer.cfc:38` variable 'response' has no component ref |
| 10 | `variables.wheels.class.adapter` | 2 | none |  | `vendor/wheels/Model.cfc:111` variable 'variables.wheels.class.adapter' has no component ref |
| 9 | `application.wheels.migrator → vendor/wheels/Migrator.cfc` | 2 | medium | declares migrateToLatest(); named like the receiver 'migrator' (2 candidates) | `cli/lucli/templates/app/tests/populate.cfm:36` variable 'application.wheels.migrator' has no component ref |
| 9 | `arguments.column → vendor/wheels/migrator/ColumnDefinition.cfc` | 2 | low | declares toSQL() (2 candidates) | `vendor/wheels/databaseAdapters/Abstract.cfc:220` variable 'arguments.column' has no component ref |
| 9 | `variables.$sshj` | 0 | none |  | `cli/lucli/services/deploy/lib/SshClient.cfc:70` variable 'variables.$sshj' has no component ref |
| 9 | `variables.registry → cli/lucli/services/packages/Registry.cfc` | 2 | low | declares listPackageNames(), fetchManifest(); named like the receiver 'registry' (2 candidates) | `cli/lucli/services/packages/PackagesMainCli.cfc:52` variable 'variables.registry' has no component ref |
| 8 | `local.controller` | 1 | none |  | `vendor/wheels/Global.cfc:4122` variable 'local.controller' has no component ref |
| 8 | `local.testresults → vendor/wheels/wheelstest/system/TestResult.cfc` | 2 | high | declares getTotalSpecs(), getTotalPass(), getTotalFail(), getTotalError(), getTotalSkipped(), getTotalDuration(), getBundleStats(), getCoverageData() | `vendor/wheels/controllers/Tests.cfc:83` variable 'local.testResults' has no component ref |
| 8 | `manager → vendor/wheels/storage/StorageManager.cfc` | 1 | high | declares disk(), getDefaultDiskName(), diskNames() | `vendor/wheels/tests/specs/storage/StorageSpec.cfc:294` variable 'manager' has no component ref |
| 8 | `rec` | 0 | none |  | `cli/src/commands/wheels/optimize/performance.cfc:82` variable 'rec' has no component ref |
| 8 | `variables.cache → cli/lucli/services/packages/ManifestCache.cfc` | 2 | low | declares hasFreshIndex(), readIndex(), writeIndex() (2 candidates) | `cli/lucli/services/packages/Registry.cfc:47` variable 'variables.cache' has no component ref |
| 8 | `variables.cache → vendor/wheels/services/packages/ManifestCache.cfc` | 2 | low | declares hasFreshIndex(), readIndex(), writeIndex() (2 candidates) | `vendor/wheels/services/packages/Registry.cfc:52` variable 'variables.cache' has no component ref |
| 8 | `variables.codegenservice → cli/lucli/services/CodeGen.cfc` | 3 | low | declares generateModel(), generateController(), generateView(), generateTest() (3 candidates) | `cli/lucli/services/Scaffold.cfc:62` variable 'variables.codeGenService' has no component ref |
| 8 | `variables.installer → cli/lucli/services/packages/Installer.cfc` | 1 | high | declares isInstalled(), installedVersion(); named like the receiver 'installer' | `cli/lucli/services/packages/PackagesMainCli.cfc:130` variable 'variables.installer' has no component ref |
| 8 | `variables.page → vendor/wheels/wheelstest/BrowserClient.cfc` | 1 | medium | declares click() | `vendor/wheels/wheelstest/BrowserClient.cfc:126` variable 'variables.page' has no component ref |
| 7 | `cache → vendor/wheels/services/packages/ManifestCache.cfc` | 7 | low | declares refresh(), writeManifest(), hasFreshManifest() (2 candidates) | `vendor/wheels/tests/specs/packages/RegistryFetchManifestSpec.cfc:40` variable 'cache' has no component ref |
| 7 | `cfg → cli/lucli/services/deploy/config/Accessory.cfc` | 2 | low | declares image() (2 candidates) | `cli/lucli/services/deploy/cli/DeployBuildCli.cfc:34` variable 'cfg' has no component ref |
| 7 | `local.proc` | 0 | none |  | `cli/src/commands/wheels/db/dump.cfc:2010` variable 'local.proc' has no component ref |
| 7 | `modulerecord.moduleconfig → vendor/wheels/wheelstest/system/util/MixerUtil.cfc` | 1 | medium | declares injectPropertyMixin() | `vendor/wheels/wheelstest/system/TestBox.cfc:196` variable 'moduleRecord.moduleConfig' has no component ref |
| 7 | `optlauncher → vendor/wheels/wheelstest/BrowserLauncher.cfc` | 2 | high | declares $classpathJarPaths(), resolveInstallDir(), $loadJars(), getState(), release(), $buildOption() | `vendor/wheels/tests/specs/wheelstest/BrowserLauncherSpec.cfc:280` variable 'optLauncher' has no component ref |
| 7 | `prop` | 0 | none |  | `cli/lucli/services/Templates.cfc:456` variable 'prop' has no component ref |
| 7 | `udfout` | 1 | none |  | `vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc:415` variable 'udfOut' has no component ref |
| 7 | `variables.$launcher → vendor/wheels/wheelstest/BrowserLauncher.cfc` | 1 | medium | declares $buildOption() | `vendor/wheels/wheelstest/BrowserClient.cfc:275` variable 'variables.$launcher' has no component ref |
| 7 | `variables.modelreference → vendor/wheels/model/query/ScopeChain.cfc` | 3 | low | declares findByKey() (3 candidates) | `vendor/wheels/model/query/ScopeChain.cfc:114` variable 'variables.modelReference' has no component ref |
| 7 | `variables.templateservice → cli/src/models/TemplateService.cfc` | 2 | medium | declares generateFromTemplate(); named like the receiver 'templateService' (2 candidates) | `cli/lucli/services/CodeGen.cfc:57` variable 'variables.templateService' has no component ref |
| 6 | `arguments.path` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:118` variable 'arguments.path' has no component ref |
| 6 | `arguments.value` | 0 | none |  | `vendor/wheels/tests/specs/model/propertiesSpec.cfc:768` variable 'arguments.value' has no component ref |
| 6 | `buffer` | 1 | none |  | `vendor/wheels/wheelstest/system/util/XMLConverter.cfc:230` variable 'buffer' has no component ref |
| 6 | `cache → cli/lucli/services/packages/ManifestCache.cfc` | 7 | low | declares refresh() (7 candidates) | `cli/lucli/tests/specs/packages/LegacyAdapterResolutionSpec.cfc:47` variable 'cache' has no component ref |
| 6 | `local.countrs` | 0 | none |  | `cli/src/commands/wheels/db/dump.cfc:736` variable 'local.countRs' has no component ref |
| 6 | `map → vendor/wheels/Global.cfc` | 13 | low | declares get() (13 candidates) | `vendor/wheels/tests/specs/mapper/MapperRobustnessSpec.cfc:148` variable 'map' has no component ref |
| 6 | `methods[]` | 14 | none |  | `cli/lucli/services/deploy/lib/Mustache.cfc:75` variable 'methods[]' has no component ref |
| 6 | `r.file` | 0 | none |  | `vendor/wheels/tests/specs/controller/miscellaneousSpec.cfc:105` variable 'r.file' has no component ref |
| 5 | `acc → cli/lucli/services/deploy/config/Accessory.cfc` | 4 | low | declares env(), hosts(), name() (2 candidates) | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:77` variable 'acc' has no component ref |
| 5 | `adapter → vendor/wheels/databaseAdapters/Abstract.cfc` | 8 | low | declares quoteTableName() (8 candidates) | `vendor/wheels/migrator/Base.cfc:121` variable 'adapter' has no component ref |
| 5 | `arguments.role → cli/lucli/services/deploy/config/Role.cfc` | 2 | medium | declares cmd(); named like the receiver 'role' (2 candidates) | `cli/lucli/services/deploy/commands/AppCommands.cfc:28` variable 'arguments.role' has no component ref |
| 5 | `arguments.runner` | 1 | none |  | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1026` variable 'arguments.runner' has no component ref |
| 5 | `arguments.writer → cli/lucli/services/deploy/lib/Output.cfc` | 1 | high | declares write(), flush() | `vendor/wheels/controller/sse.cfc:108` variable 'arguments.writer' has no component ref |
| 5 | `command` | 0 | none |  | `cli/lucli/services/deploy/lib/SshClient.cfc:136` variable 'command' has no component ref |
| 5 | `ctx.s → cli/lucli/services/deploy/config/Ssh.cfc` | 1 | medium | declares $expandHome() | `cli/lucli/tests/specs/deploy/config/SshSpec.cfc:53` variable 'ctx.s' has no component ref |
| 5 | `local.class → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/model/sql.cfc:1354` variable 'local.class' has no component ref |
| 5 | `local.ctrl` | 1 | none |  | `vendor/wheels/Dispatch.cfc:440` variable 'local.ctrl' has no component ref |
| 5 | `local.instance` | 142 | none |  | `vendor/wheels/Test.cfc:518` variable 'local.instance' has no component ref |
| 5 | `local.process` | 0 | none |  | `cli/src/commands/wheels/db/dump.cfc:1888` variable 'local.process' has no component ref |
| 5 | `pkg.controller → vendor/wheels/view/formsdate.cfc` | 1 | medium | declares $yearMonthHourMinuteSecondSelectTagContent() | `vendor/wheels/tests/specs/view/formsdateSpec.cfc:14` variable 'pkg.controller' has no component ref |
| 5 | `pub → vendor/wheels/Public.cfc` | 2 | low | declares $loadRegistryPackages() (2 candidates) | `vendor/wheels/tests/specs/packages/LoadRegistryPackagesSpec.cfc:32` variable 'pub' has no component ref |
| 5 | `r → vendor/wheels/services/packages/Registry.cfc` | 2 | low | declares fetchManifest(), listAll() (2 candidates) | `vendor/wheels/tests/specs/packages/RegistryFetchManifestSpec.cfc:33` variable 'r' has no component ref |
| 5 | `r.name` | 0 | none |  | `vendor/wheels/tests/specs/controller/miscellaneousSpec.cfc:107` variable 'r.name' has no component ref |
| 5 | `scaffold → cli/lucli/services/Scaffold.cfc` | 2 | medium | declares createMigrationWithProperties(); named like the receiver 'scaffold' (2 candidates) | `cli/lucli/Module.cfc:3581` variable 'scaffold' has no component ref |
| 5 | `svc → cli/lucli/services/Destroy.cfc` | 1 | high | declares previewDestroy(), destroyResource(), destroyModel(), destroyController(), destroyView() | `cli/lucli/Module.cfc:2143` variable 'svc' has no component ref |
| 5 | `t → vendor/wheels/migrator/TableDefinition.cfc` | 1 | medium | declares change() | `vendor/wheels/migrator/templates/change-table.cfc:20` variable 't' has no component ref |
| 5 | `variables.page → vendor/wheels/interfaces/StorageDiskInterface.cfc` | 3 | low | declares url() (3 candidates) | `vendor/wheels/wheelstest/BrowserClient.cfc:671` variable 'variables.page' has no component ref |
| 5 | `variables.semver → cli/lucli/services/SemVer.cfc` | 2 | low | declares satisfiesAll(), compare(); named like the receiver 'semver' (2 candidates) | `cli/lucli/services/packages/VersionResolver.cfc:57` variable 'variables.semver' has no component ref |
| 4 | `api → vendor/wheels/BuildInfo.cfc` | 6 | low | declares version() (6 candidates) | `vendor/wheels/tests/specs/mapperModernSpec.cfc:138` variable 'api' has no component ref |
| 4 | `application.wheels.dispatch → vendor/wheels/Dispatch.cfc` | 2 | medium | declares $request(); named like the receiver 'dispatch' (2 candidates) | `cli/lucli/templates/app/public/index.cfm:5` variable 'application.wheels.dispatch' has no component ref |
| 4 | `application.wirebox → vendor/wheels/Injector.cfc` | 3 | low | declares getInstance() (2 candidates) | `cli/src/commands/wheels/generate/app-wizard.cfc:76` variable 'application.wirebox' has no component ref |
| 4 | `arguments.dialog` | 0 | none |  | `vendor/wheels/wheelstest/DialogConsumer.cfc:24` variable 'arguments.dialog' has no component ref |
| 4 | `arguments.migration.cfc → vendor/wheels/migrator/Migration.cfc` | 44 | low | declares down(), up() (44 candidates) | `vendor/wheels/Migrator.cfc:371` variable 'arguments.migration.cfc' has no component ref |
| 4 | `arguments.target` | 142 | none |  | `vendor/wheels/wheelstest/system/runners/UnitRunner.cfc:73` variable 'arguments.target' has no component ref |
| 4 | `attributes.callbacks` | 0 | none |  | `vendor/wheels/wheelstest/system/runners/BDDRunner.cfc:259` variable 'attributes.callbacks' has no component ref |
| 4 | `b → cli/lucli/services/deploy/config/Builder.cfc` | 1 | high | declares dockerfile(), context() | `cli/lucli/services/deploy/commands/BuilderCommands.cfc:18` variable 'b' has no component ref |
| 4 | `cmds → cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc` | 9 | low | declares start() (8 candidates) | `cli/lucli/services/deploy/cli/DeployAppCli.cfc:38` variable 'cmds' has no component ref |
| 4 | `cmds → cli/lucli/services/deploy/cli/DeployAppCli.cfc` | 4 | low | declares containers() (4 candidates) | `cli/lucli/services/deploy/cli/DeployAppCli.cfc:59` variable 'cmds' has no component ref |
| 4 | `constructors[]` | 0 | none |  | `vendor/wheels/wheelstest/BrowserLauncher.cfc:472` variable 'constructors[]' has no component ref |
| 4 | `env → cli/lucli/services/deploy/config/Env.cfc` | 3 | high | declares clear(), secret(); named like the receiver 'env' | `cli/lucli/services/deploy/commands/AccessoryCommands.cfc:99` variable 'env' has no component ref |
| 4 | `loc → vendor/wheels/wheelstest/BrowserClient.cfc` | 1 | medium | declares waitFor() | `vendor/wheels/wheelstest/BrowserClient.cfc:297` variable 'loc' has no component ref |
| 4 | `local.conn → cli/lucli/services/deploy/lib/SshClient.cfc` | 2 | low | declares close() (2 candidates) | `cli/src/commands/wheels/db/create.cfc:321` variable 'local.conn' has no component ref |
| 4 | `local.env → vendor/wheels/interfaces/StorageDiskInterface.cfc` | 6 | low | declares put() (6 candidates) | `cli/src/commands/wheels/db/dump.cfc:453` variable 'local.env' has no component ref |
| 4 | `local.errorstream` | 0 | none |  | `cli/src/commands/wheels/db/dump.cfc:1922` variable 'local.errorStream' has no component ref |
| 4 | `local.fknamesource → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/model/sql.cfc:1405` variable 'local.fkNameSource' has no component ref |
| 4 | `local.inputstream` | 0 | none |  | `cli/src/commands/wheels/db/dump.cfc:1909` variable 'local.inputStream' has no component ref |
| 4 | `local.pb` | 0 | none |  | `cli/src/commands/wheels/db/shell.cfc:514` variable 'local.pb' has no component ref |
| 4 | `map → vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` | 13 | low | declares get(), group() (2 candidates) | `vendor/wheels/tests/specs/mapperModernSpec.cfc:32` variable 'map' has no component ref |
| 4 | `modulerecord.moduleconfig` | 1 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:194` variable 'moduleRecord.moduleConfig' has no component ref |
| 4 | `s → cli/lucli/services/deploy/config/Ssh.cfc` | 1 | high | declares user(), port(), keys(), $expandHome() | `cli/lucli/services/deploy/lib/SshPoolFactory.cfc:45` variable 's' has no component ref |
| 4 | `s3down → vendor/wheels/interfaces/StorageDiskInterface.cfc` | 6 | low | declares put(), exists() (3 candidates) | `vendor/wheels/tests/specs/storage/StorageSpec.cfc:250` variable 's3down' has no component ref |
| 4 | `sftp` | 0 | none |  | `cli/lucli/services/deploy/lib/SshClient.cfc:285` variable 'sftp' has no component ref |
| 4 | `sock` | 0 | none |  | `cli/lucli/tests/StubHttpServer.cfc:45` variable 'sock' has no component ref |
| 4 | `system.out` | 3 | none |  | `cli/lucli/Module.cfc:1608` variable 'System.out' has no component ref |
| 4 | `this.adapter → vendor/wheels/databaseAdapters/Abstract.cfc` | 2 | low | declares foreignKeySQL() (2 candidates) | `vendor/wheels/migrator/ForeignKeyDefinition.cfc:37` variable 'this.adapter' has no component ref |
| 4 | `thread.target → vendor/wheels/wheelstest/system/BaseSpec.cfc` | 2 | medium | declares runSpec() | `vendor/wheels/wheelstest/system/runners/BDDRunner.cfc:278` variable 'thread.target' has no component ref |
| 4 | `user.author → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/tests/specs/model/validationsSpec.cfc:26` variable 'user.author' has no component ref |
| 4 | `user.author.profile → vendor/wheels/Model.cfc` | 2 | high | declares addError(), $classData() | `vendor/wheels/tests/specs/model/errorsSpec.cfc:24` variable 'user.author.profile' has no component ref |
| 4 | `variables.modules.registry` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:137` variable 'variables.modules.registry' has no component ref |
| 4 | `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/CockroachDB/CockroachDBModel.cfc` | 7 | low | declares $querySetup() (7 candidates) | `vendor/wheels/model/delete.cfc:198` variable 'variables.wheels.class.adapter' has no component ref |
| 4 | `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerModel.cfc` | 7 | low | declares $quoteIdentifier(), $getType() (5 candidates) | `vendor/wheels/model/sql.cfc:954` variable 'variables.wheels.class.adapter' has no component ref |
| 3 | `application.wheels.mapper → vendor/wheels/Mapper.cfc` | 1 | high | declares $patternToRegex(); named like the receiver 'mapper' | `vendor/wheels/Global.cfc:4396` variable 'application.wheels.mapper' has no component ref |
| 3 | `arguments.columnowner → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/model/sql.cfc:1620` variable 'arguments.columnOwner' has no component ref |
| 3 | `arguments.container → vendor/wheels/tests/_assets/plugins/serviceprovider/FakeContainer.cfc` | 3 | low | declares mapInstance() (3 candidates) | `vendor/wheels/tests/_assets/plugins/serviceprovider/TestServiceProvider/TestServiceProvider.cfc:25` variable 'arguments.container' has no component ref |
| 3 | `arguments.node` | 0 | none |  | `cli/lucli/services/deploy/lib/Yaml.cfc:181` variable 'arguments.node' has no component ref |
| 3 | `arguments.proc` | 0 | none |  | `cli/lucli/services/deploy/lib/SecretResolver.cfc:212` variable 'arguments.proc' has no component ref |
| 3 | `arguments.runner → vendor/wheels/wheelstest/system/TestBox.cfc` | 2 | high | declares announceToModules(); named like the receiver 'TestBox' | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1064` variable 'arguments.runner' has no component ref |
| 3 | `arguments.runner → vendor/wheels/wheelstest/system/runners/BaseRunner.cfc` | 1 | high | declares canRunLabel(), canRunSpec() | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1371` variable 'arguments.runner' has no component ref |
| 3 | `arguments.target → vendor/wheels/tests/specs/mapperModernSpec.cfc` | 637 | low | declares run(), beforeAll(), afterAll() (60 candidates) | `vendor/wheels/wheelstest/system/runners/BDDRunner.cfc:63` variable 'arguments.target' has no component ref |
| 3 | `auth` | 2 | none |  | `examples/starter-app/app/controllers/Sessions.cfc:24` variable 'auth' has no component ref |
| 3 | `cfg → cli/lucli/services/deploy/config/Config.cfc` | 2 | low | declares service() (2 candidates) | `cli/lucli/services/deploy/cli/DeployLockCli.cfc:33` variable 'cfg' has no component ref |
| 3 | `compiler` | 0 | none |  | `cli/lucli/services/deploy/lib/Mustache.cfc:92` variable 'compiler' has no component ref |
| 3 | `core → vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc` | 6 | low | declares URLFor() (6 candidates) | `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc:8` variable 'core' has no component ref |
| 3 | `core → vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc` | 6 | low | declares URLFor() (6 candidates) | `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc:8` variable 'core' has no component ref |
| 3 | `ctors[]` | 0 | none |  | `cli/lucli/services/deploy/lib/JarLoader.cfc:111` variable 'ctors[]' has no component ref |
| 3 | `fixtures.session.scaffold → cli/lucli/services/Scaffold.cfc` | 2 | medium | declares generateAuth(); named like the receiver 'scaffold' (2 candidates) | `cli/lucli/tests/specs/services/GenerateAuthSpec.cfc:411` variable 'fixtures.session.scaffold' has no component ref |
| 3 | `local.engine → vendor/wheels/Channel.cfc` | 4 | low | declares publish() (3 candidates) | `vendor/wheels/Global.cfc:1524` variable 'local.engine' has no component ref |
| 3 | `local.evalresult → vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` | 2 | high | declares properties(), key(), isNew() | `vendor/wheels/public/views/consoleeval.cfm:198` variable 'local.evalResult' has no component ref |
| 3 | `local.modelobj → vendor/wheels/Model.cfc` | 2 | high | declares tableName(), primaryKeys(), $classData() | `vendor/wheels/migrator/AutoMigrator.cfc:21` variable 'local.modelObj' has no component ref |
| 3 | `local.newrecord` | 2 | none |  | `vendor/wheels/Seeder.cfc:208` variable 'local.newRecord' has no component ref |
| 3 | `local.object → vendor/wheels/model/errors.cfc` | 2 | low | declares allErrors() (2 candidates) | `vendor/wheels/view/errors.cfc:26` variable 'local.object' has no component ref |
| 3 | `local.object → vendor/wheels/model/properties.cfc` | 1 | medium | declares $propertyInfo() | `vendor/wheels/view/forms.cfc:355` variable 'local.object' has no component ref |
| 3 | `local.rs2` | 0 | none |  | `cli/src/commands/wheels/db/dump.cfc:1636` variable 'local.rs2' has no component ref |
| 3 | `local.rv → vendor/wheels/model/callbacks.cfc` | 3 | low | declares $callback() (2 candidates) | `vendor/wheels/model/create.cfc:126` variable 'local.rv' has no component ref |
| 3 | `local.stmt → vendor/wheels/migrator/Migration.cfc` | 1 | medium | declares execute() | `cli/src/commands/wheels/db/create.cfc:1299` variable 'local.stmt' has no component ref |
| 3 | `mapper` | 1 | none |  | `vendor/wheels/tests/specs/dispatch/routeModelBindingSpec.cfc:232` variable 'mapper' has no component ref |
| 3 | `md` | 0 | none |  | `vendor/wheels/wheelstest/system/Assertion.cfc:289` variable 'md' has no component ref |
| 3 | `pkg.controller → vendor/wheels/view/formsdateplain.cfc` | 1 | medium | declares dateTimeSelectTags() | `vendor/wheels/tests/specs/view/formsdateplainSpec.cfc:67` variable 'pkg.controller' has no component ref |
| 3 | `plugin` | 0 | none |  | `cli/src/commands/wheels/plugins/list.cfc:188` variable 'plugin' has no component ref |
| 3 | `registry → cli/lucli/services/ServerRegistry.cfc` | 1 | high | declares serverNameFor(), inspect(), clean() | `cli/lucli/Module.cfc:1031` variable 'registry' has no component ref |
| 3 | `result.config` | 0 | none |  | `cli/src/commands/wheels/env/setup.cfc:210` variable 'result.config' has no component ref |
| 3 | `result.post → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `vendor/wheels/tests/specs/dispatch/routeModelBindingSpec.cfc:52` variable 'result.post' has no component ref |
| 3 | `s3 → vendor/wheels/interfaces/StorageDiskInterface.cfc` | 3 | low | declares url(), signedUrl() (3 candidates) | `vendor/wheels/tests/specs/storage/StorageSpec.cfc:200` variable 's3' has no component ref |
| 3 | `server.coldfusion.productname` | 0 | none |  | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1814` variable 'server.coldfusion.productName' has no component ref |
| 3 | `server.os.name` | 0 | none |  | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1826` variable 'server.os.name' has no component ref |
| 3 | `settings.bcrypt` | 0 | none |  | `examples/starter-app/plugins/authenticateThis/authenticateThis.cfc:54` variable 'settings.bCrypt' has no component ref |
| 3 | `sshjref` | 0 | none |  | `cli/lucli/services/deploy/lib/SshClient.cfc:129` variable 'sshjRef' has no component ref |
| 3 | `stdoutstream` | 0 | none |  | `cli/lucli/services/deploy/lib/SecretResolver.cfc:227` variable 'stdoutStream' has no component ref |
| 3 | `this.adapter → vendor/wheels/migrator/Migration.cfc` | 5 | low | declares dropTable(), createTable() (5 candidates) | `vendor/wheels/migrator/TableDefinition.cfc:393` variable 'this.adapter' has no component ref |
| 3 | `this.mockbox → vendor/wheels/wheelstest/system/MockBox.cfc` | 2 | high | declares normalizeArguments(); named like the receiver 'mockBox' | `public/testbox/system/stubs/F952D54F1096E25C030C8E3149ABD8C4.cfm:18` variable 'this.mockBox' has no component ref |
| 3 | `user.author → vendor/wheels/model/errors.cfc` | 2 | low | declares addError() (2 candidates) | `vendor/wheels/tests/specs/model/errorsSpec.cfc:13` variable 'user.author' has no component ref |
| 3 | `v1 → vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` | 13 | high | declares get(), resources() | `vendor/wheels/tests/specs/mapperModernSpec.cfc:83` variable 'v1' has no component ref |
| 3 | `variables.$playwright` | 0 | none |  | `vendor/wheels/wheelstest/BrowserLauncher.cfc:528` variable 'variables.$playwright' has no component ref |
| 3 | `variables.context → vendor/wheels/wheelstest/BrowserClient.cfc` | 1 | medium | declares clearCookies() | `vendor/wheels/wheelstest/BrowserClient.cfc:502` variable 'variables.context' has no component ref |
| 3 | `variables.http → cli/lucli/services/packages/HttpClient.cfc` | 3 | low | declares download() (3 candidates) | `cli/lucli/services/packages/Installer.cfc:90` variable 'variables.http' has no component ref |
| 3 | `variables.page → vendor/wheels/model/query/QueryBuilder.cfc` | 1 | medium | declares first() | `vendor/wheels/wheelstest/BrowserClient.cfc:126` variable 'variables.page' has no component ref |
| 3 | `variables.page → vendor/wheels/wheelstest/TestClient.cfc` | 1 | medium | declares content() | `vendor/wheels/wheelstest/BrowserClient.cfc:609` variable 'variables.page' has no component ref |
| 3 | `variables.resolver → cli/lucli/services/packages/VersionResolver.cfc` | 1 | medium | declares compatibleVersions() | `cli/lucli/services/packages/PackagesMainCli.cfc:137` variable 'variables.resolver' has no component ref |
| 3 | `variables.sshpool → cli/lucli/services/deploy/lib/FakeSshPool.cfc` | 2 | low | declares $setSecretValues() (2 candidates) | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:147` variable 'variables.sshPool' has no component ref |
| 3 | `variables.templatemap` | 0 | none |  | `cli/src/commands/wheels/generate/app.cfc:125` variable 'variables.templateMap' has no component ref |
| 3 | `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` | 5 | low | declares $defaultValues(), $querySetup(), $generatedKey() (3 candidates) | `vendor/wheels/model/create.cfc:303` variable 'variables.wheels.class.adapter' has no component ref |
| 3 | `variables.wheels.class.adapter → vendor/wheels/model/query/QueryBuilder.cfc` | 3 | low | declares $quoteValue() (3 candidates) | `vendor/wheels/model/onmissingmethod.cfc:170` variable 'variables.wheels.class.adapter' has no component ref |
| 2 | `adapter → vendor/wheels/migrator/ColumnDefinition.cfc` | 4 | low | declares addColumnOptions() (4 candidates) | `vendor/wheels/migrator/ColumnDefinition.cfc:59` variable 'adapter' has no component ref |
| 2 | `analysis → cli/lucli/services/Analysis.cfc` | 3 | medium | declares analyze(); named like the receiver 'analysis' (3 candidates) | `cli/lucli/Module.cfc:1938` variable 'analysis' has no component ref |
| 2 | `api → vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` | 13 | high | declares get(), version() | `vendor/wheels/tests/specs/mapperModernSpec.cfc:116` variable 'api' has no component ref |
| 2 | `application.log → cli/lucli/Module.cfc` | 9 | low | declares info() (9 candidates) | `examples/starter-app/app/jobs/ProcessOrdersJob.cfc:123` variable 'application.log' has no component ref |
| 2 | `application.log → cli/src/models/DetailOutputService.cfc` | 2 | medium | declares error() | `examples/starter-app/app/jobs/ProcessOrdersJob.cfc:133` variable 'application.log' has no component ref |
| 2 | `arg → vendor/wheels/wheelstest/system/Assertion.cfc` | 2 | medium | declares startsWith() | `cli/lucli/Module.cfc:7220` variable 'arg' has no component ref |
| 2 | `arguments.app` | 0 | none |  | `vendor/wheels/tests/_assets/plugins/middleware/TestMiddlewarePluginA/TestMiddlewarePluginA.cfc:10` variable 'arguments.app' has no component ref |
| 2 | `arguments.collection → vendor/wheels/mapper/matching.cfc` | 3 | low | declares whereIn() (3 candidates) | `vendor/wheels/Policy.cfc:110` variable 'arguments.collection' has no component ref |
| 2 | `arguments.context.host → vendor/wheels/Global.cfc` | 1 | medium | declares model() | `vendor/wheels/public/CliBridge.cfc:467` variable 'arguments.context.host' has no component ref |
| 2 | `arguments.context.host → vendor/wheels/Public.cfc` | 1 | medium | declares $cliFormatMigrationStatus() | `vendor/wheels/public/CliBridge.cfc:355` variable 'arguments.context.host' has no component ref |
| 2 | `arguments.foreignkey → vendor/wheels/migrator/ColumnDefinition.cfc` | 2 | low | declares toSQL() (2 candidates) | `vendor/wheels/databaseAdapters/Abstract.cfc:252` variable 'arguments.foreignKey' has no component ref |
| 2 | `arguments.key` | 0 | none |  | `vendor/wheels/wheelstest/system/Assertion.cfc:430` variable 'arguments.key' has no component ref |
| 2 | `arguments.klass` | 0 | none |  | `vendor/wheels/wheelstest/BrowserLauncher.cfc:470` variable 'arguments.klass' has no component ref |
| 2 | `arguments.results.metrics.queries` | 0 | none |  | `cli/src/commands/wheels/analyze/performance.cfc:485` variable 'arguments.results.metrics.queries' has no component ref |
| 2 | `arguments.results.metrics.requests` | 0 | none |  | `cli/src/commands/wheels/analyze/performance.cfc:463` variable 'arguments.results.metrics.requests' has no component ref |
| 2 | `arguments.scope` | 0 | none |  | `vendor/wheels/wheelstest/system/MockBox.cfc:184` variable 'arguments.scope' has no component ref |
| 2 | `arguments.spec` | 0 | none |  | `vendor/wheels/wheelstest/BrowserTest.cfc:152` variable 'arguments.spec' has no component ref |
| 2 | `arguments.stream` | 0 | none |  | `cli/lucli/services/deploy/lib/SshClient.cfc:356` variable 'arguments.stream' has no component ref |
| 2 | `arguments.suite → vendor/wheels/wheelstest/system/BaseSpec.cfc` | 2 | medium | declares beforeEach() | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1192` variable 'arguments.suite' has no component ref |
| 2 | `col → vendor/wheels/migrator/ColumnDefinition.cfc` | 2 | low | declares toSQL() (2 candidates) | `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc:55` variable 'col' has no component ref |
| 2 | `consolidatedlabels` | 1 | none |  | `vendor/wheels/wheelstest/system/BaseSpec.cfc:991` variable 'consolidatedLabels' has no component ref |
| 2 | `content` | 0 | none |  | `cli/tests/specs/e2e/McpToolsTest.cfc:393` variable 'content' has no component ref |
| 2 | `contents` | 0 | none |  | `cli/src/commands/wheels/cleanup/logs.cfc:183` variable 'contents' has no component ref |
| 2 | `cookies` | 0 | none |  | `vendor/wheels/wheelstest/BrowserClient.cfc:513` variable 'cookies' has no component ref |
| 2 | `core → examples/starter-app/plugins/jsconfirm/JSConfirm.cfc` | 4 | low | declares linkTo() (4 candidates) | `examples/starter-app/plugins/jsconfirm/JSConfirm.cfc:13` variable 'core' has no component ref |
| 2 | `core → vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner02/Runner02.cfc` | 6 | low | declares URLFor() (6 candidates) | `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner02/Runner02.cfc:8` variable 'core' has no component ref |
| 2 | `core → vendor/wheels/tests/_assets/plugins/runner/runner02/Runner02.cfc` | 6 | low | declares URLFor() (6 candidates) | `vendor/wheels/tests/_assets/plugins/runner/runner02/Runner02.cfc:8` variable 'core' has no component ref |
| 2 | `ctx.ctrl → vendor/wheels/controller/rendering.cfc` | 2 | low | declares renderText(), renderNothing() (2 candidates) | `vendor/wheels/tests/specs/controller/renderingSpec.cfc:868` variable 'ctx.ctrl' has no component ref |
| 2 | `dirs` | 0 | none |  | `cli/src/commands/wheels/cleanup/logs.cfc:175` variable 'dirs' has no component ref |
| 2 | `filesbydir[]` | 1 | none |  | `cli/src/commands/wheels/cleanup/tmp.cfc:118` variable 'filesByDir[]' has no component ref |
| 2 | `item` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:115` variable 'item' has no component ref |
| 2 | `local._model → vendor/wheels/model/create.cfc` | 1 | medium | declares $createInstance() | `vendor/wheels/model/serialize.cfc:59` variable 'local._model' has no component ref |
| 2 | `local._model → vendor/wheels/model/serialize.cfc` | 1 | high | declares $queryRowToStruct(), $keyFromStruct() | `vendor/wheels/model/serialize.cfc:129` variable 'local._model' has no component ref |
| 2 | `local.adapter → vendor/wheels/databaseAdapters/Base.cfc` | 8 | low | declares $acquireAdvisoryLock(), $releaseAdvisoryLock() (8 candidates) | `vendor/wheels/model/locking.cfc:41` variable 'local.adapter' has no component ref |
| 2 | `local.assoc → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `vendor/wheels/view/formsassociation.cfc:141` variable 'local.assoc' has no component ref |
| 2 | `local.auth → vendor/wheels/auth/Authenticator.cfc` | 1 | high | declares authenticateWith(), authenticate() | `vendor/wheels/middleware/AuthMiddleware.cfc:80` variable 'local.auth' has no component ref |
| 2 | `local.entry` | 0 | none |  | `vendor/wheels/Channel.cfc:94` variable 'local.entry' has no component ref |
| 2 | `local.entry.strategy → vendor/wheels/auth/AuthStrategy.cfc` | 12 | low | declares authenticate() (12 candidates) | `vendor/wheels/auth/Authenticator.cfc:151` variable 'local.entry.strategy' has no component ref |
| 2 | `local.envmap` | 0 | none |  | `cli/src/commands/wheels/base.cfc:900` variable 'local.envMap' has no component ref |
| 2 | `local.model` | 3 | none |  | `vendor/wheels/model/nestedproperties.cfc:314` variable 'local.model' has no component ref |
| 2 | `local.proc → vendor/wheels/wheelstest/BrowserClient.cfc` | 1 | medium | declares waitFor() | `cli/src/commands/wheels/db/dump.cfc:465` variable 'local.proc' has no component ref |
| 2 | `local.process → vendor/wheels/wheelstest/BrowserClient.cfc` | 1 | medium | declares waitFor() | `cli/src/commands/wheels/db/restore.cfc:301` variable 'local.process' has no component ref |
| 2 | `local.sessionmanager → vendor/wheels/public/mcp/SessionManager.cfc` | 1 | high | declares updateSession(); named like the receiver 'sessionManager' | `vendor/wheels/public/mcp/McpServer.cfc:164` variable 'local.sessionManager' has no component ref |
| 2 | `local.stmt → cli/lucli/services/deploy/lib/SshClient.cfc` | 2 | low | declares close() (2 candidates) | `cli/src/commands/wheels/db/create.cfc:1308` variable 'local.stmt' has no component ref |
| 2 | `local.stmt2` | 0 | none |  | `cli/src/commands/wheels/db/dump.cfc:1634` variable 'local.stmt2' has no component ref |
| 2 | `local.writer` | 0 | none |  | `vendor/wheels/controller/channels.cfc:209` variable 'local.writer' has no component ref |
| 2 | `log → cli/lucli/Module.cfc` | 9 | low | declares info() (9 candidates) | `cli/src/ModuleConfig.cfc:77` variable 'log' has no component ref |
| 2 | `mapper2 → vendor/wheels/mapper/mapping.cfc` | 1 | medium | declares $draw() | `vendor/wheels/tests/specs/global/urlforSpec.cfc:248` variable 'mapper2' has no component ref |
| 2 | `migration.adapter → vendor/wheels/databaseAdapters/Abstract.cfc` | 7 | low | declares changeColumnInTable() (7 candidates) | `vendor/wheels/tests/specs/migrator/sqliteChangeColumnTransactionSpec.cfc:50` variable 'migration.adapter' has no component ref |
| 2 | `migration.cfc → vendor/wheels/migrator/Migration.cfc` | 44 | low | declares up(), down() (44 candidates) | `vendor/wheels/public/migrator/sql.cfm:23` variable 'migration.CFC' has no component ref |
| 2 | `migrations` | 0 | none |  | `cli/src/commands/wheels/dbmigrate/down.cfc:44` variable 'migrations' has no component ref |
| 2 | `modelinstance → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/Seeder.cfc:275` variable 'modelInstance' has no component ref |
| 2 | `nextclosure` | 0 | none |  | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1290` variable 'nextClosure' has no component ref |
| 2 | `obj.gallery` | 2 | none |  | `vendor/wheels/tests/specs/model/callbacksSpec.cfc:466` variable 'obj.gallery' has no component ref |
| 2 | `os → cli/lucli/services/deploy/lib/Output.cfc` | 1 | high | declares write(), flush() | `cli/lucli/services/deploy/lib/SshClient.cfc:137` variable 'os' has no component ref |
| 2 | `outstream → cli/lucli/services/deploy/lib/Output.cfc` | 1 | high | declares write(), flush() | `cli/lucli/tests/StubHttpServer.cfc:58` variable 'outStream' has no component ref |
| 2 | `permission → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `examples/starter-app/app/views/admin/permissions/_form.cfm:9` variable 'permission' has no component ref |
| 2 | `reg → cli/lucli/services/deploy/config/Registry.cfc` | 1 | high | declares server(), username() | `cli/lucli/services/deploy/commands/RegistryCommands.cfc:24` variable 'reg' has no component ref |
| 2 | `result.plugin` | 0 | none |  | `cli/src/commands/wheels/plugins/install.cfc:54` variable 'result.plugin' has no component ref |
| 2 | `results.assets` | 0 | none |  | `cli/src/commands/wheels/optimize/performance.cfc:67` variable 'results.assets' has no component ref |
| 2 | `results.author → vendor/wheels/migrator/TableDefinition.cfc` | 3 | low | declares primaryKey() (3 candidates) | `vendor/wheels/tests/specs/model/crudSpec.cfc:360` variable 'results.author' has no component ref |
| 2 | `results.shop → vendor/wheels/migrator/TableDefinition.cfc` | 3 | low | declares primaryKey() (3 candidates) | `vendor/wheels/tests/specs/model/crudSpec.cfc:372` variable 'results.shop' has no component ref |
| 2 | `setter` | 0 | none |  | `vendor/wheels/wheelstest/BrowserLauncher.cfc:448` variable 'setter' has no component ref |
| 2 | `spec.browser → vendor/wheels/wheelstest/BrowserClient.cfc` | 1 | high | declares visitUrl(), assertSee() | `vendor/wheels/tests/specs/wheelstest/BrowserTestNotWiredSpec.cfc:15` variable 'spec.browser' has no component ref |
| 2 | `specmd` | 0 | none |  | `vendor/wheels/wheelstest/system/runners/UnitRunner.cfc:453` variable 'specMD' has no component ref |
| 2 | `ssh → cli/lucli/tests/specs/deploy/lib/FakeSshPoolSpec.cfc` | 637 | low | declares run() (633 candidates) | `cli/lucli/tests/specs/deploy/lib/SshPoolSpec.cfc:21` variable 'ssh' has no component ref |
| 2 | `state.results → vendor/wheels/wheelstest/system/TestResult.cfc` | 2 | high | declares getTotalError(), getBundleStats() | `vendor/wheels/tests/specs/wheelstest/BDDRunnerErrorReportingSpec.cfc:23` variable 'state.results' has no component ref |
| 2 | `svc → cli/lucli/services/Stats.cfc` | 2 | low | declares getStats() (2 candidates) | `cli/lucli/Module.cfc:2842` variable 'svc' has no component ref |
| 2 | `targetmd` | 0 | none |  | `vendor/wheels/wheelstest/system/runners/BDDRunner.cfc:45` variable 'targetMD' has no component ref |
| 2 | `testauthor.profile → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `vendor/wheels/tests/specs/model/nestedpropertiesSpec.cfc:383` variable 'testAuthor.profile' has no component ref |
| 2 | `this.$expectedexception` | 0 | none |  | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1390` variable 'this.$expectedException' has no component ref |
| 2 | `variables.$browser` | 0 | none |  | `vendor/wheels/wheelstest/BrowserTest.cfc:185` variable 'variables.$browser' has no component ref |
| 2 | `variables.$class.plugins[] → vendor/wheels/ServiceProviderInterface.cfc` | 10 | low | declares register() (9 candidates) | `vendor/wheels/Plugins.cfc:554` variable 'variables.$class.plugins[]' has no component ref |
| 2 | `variables.$classloader → cli/lucli/services/deploy/lib/JarLoader.cfc` | 1 | medium | declares loadClass() | `vendor/wheels/wheelstest/BrowserLauncher.cfc:285` variable 'variables.$classLoader' has no component ref |
| 2 | `variables.context` | 0 | none |  | `vendor/wheels/wheelstest/BrowserClient.cfc:484` variable 'variables.context' has no component ref |
| 2 | `variables.http → vendor/wheels/services/packages/HttpClient.cfc` | 13 | low | declares get() (13 candidates) | `vendor/wheels/services/packages/Registry.cfc:56` variable 'variables.http' has no component ref |
| 2 | `variables.sink → vendor/wheels/wheelstest/system/BaseSpec.cfc` | 2 | medium | declares println() | `cli/lucli/services/deploy/lib/Output.cfc:29` variable 'variables.sink' has no component ref |
| 2 | `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/H2/H2Model.cfc` | 7 | low | declares $upsertSQL(), $querySetup() (6 candidates) | `vendor/wheels/model/bulk.cfc:133` variable 'variables.wheels.class.adapter' has no component ref |
| 2 | `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/Oracle/OracleModel.cfc` | 2 | high | declares $bulkInsertSQL(), $querySetup() | `vendor/wheels/model/bulk.cfc:45` variable 'variables.wheels.class.adapter' has no component ref |
| 1 | `a → cli/lucli/services/deploy/config/Accessory.cfc` | 9 | low | declares name() (9 candidates) | `cli/lucli/tests/specs/deploy/config/AccessorySpec.cfc:15` variable 'a' has no component ref |
| 1 | `a → vendor/wheels/wheelstest/system/Assertion.cfc` | 2 | medium | declares startsWith() | `cli/lucli/Module.cfc:4651` variable 'a' has no component ref |
| 1 | `acc → cli/lucli/services/deploy/config/Env.cfc` | 1 | high | declares secret(); named like the receiver 'env' | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:77` variable 'acc' has no component ref |
| 1 | `adapter → cli/lucli/services/deploy/cli/DeploySecretsCli.cfc` | 8 | low | declares fetch() (8 candidates) | `cli/lucli/services/deploy/cli/DeploySecretsCli.cfc:51` variable 'adapter' has no component ref |
| 1 | `adapter → vendor/wheels/migrator/ForeignKeyDefinition.cfc` | 6 | low | declares addForeignKeyOptions() (6 candidates) | `vendor/wheels/migrator/ForeignKeyDefinition.cfc:57` variable 'adapter' has no component ref |
| 1 | `apimap → vendor/wheels/BuildInfo.cfc` | 6 | low | declares version() (6 candidates) | `vendor/wheels/tests/specs/mapper/MapperRobustnessSpec.cfc:171` variable 'apiMap' has no component ref |
| 1 | `apimap → vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` | 13 | high | declares get(), version() | `vendor/wheels/tests/specs/mapper/MapperRobustnessSpec.cfc:159` variable 'apiMap' has no component ref |
| 1 | `application.$wheels.buildinfo → vendor/wheels/BuildInfo.cfc` | 6 | medium | declares version(); named like the receiver 'buildInfo' (6 candidates) | `vendor/wheels/events/onapplicationstart.cfc:88` variable 'application.$wheels.buildInfo' has no component ref |
| 1 | `application.$wheels.engineadapter → vendor/wheels/engineAdapters/Base.cfc` | 2 | low | declares prepareDIComplete() (2 candidates) | `vendor/wheels/events/onapplicationstart.cfc:424` variable 'application.$wheels.engineAdapter' has no component ref |
| 1 | `application.wheels.buildinfo → vendor/wheels/BuildInfo.cfc` | 6 | medium | declares version(); named like the receiver 'buildInfo' (6 candidates) | `vendor/wheels/tests/specs/events/frameworkVersionSpec.cfc:26` variable 'application.wheels.buildInfo' has no component ref |
| 1 | `application.wheels.controllers[] → vendor/wheels/Controller.cfc` | 1 | medium | declares $getControllerClassData() | `vendor/wheels/Controller.cfc:128` variable 'application.wheels.controllers[]' has no component ref |
| 1 | `application.wheels.public → vendor/wheels/Public.cfc` | 2 | medium | declares $loadRegistryPackages(); named like the receiver 'public' (2 candidates) | `vendor/wheels/public/views/packagelist.cfm:44` variable 'application.wheels.public' has no component ref |
| 1 | `application.wheels.seeder → vendor/wheels/Seeder.cfc` | 1 | high | declares hasSeedFiles(); named like the receiver 'seeder' | `vendor/wheels/public/CliBridge.cfc:893` variable 'application.wheels.seeder' has no component ref |
| 1 | `application[].mapper → vendor/wheels/mapper/mapping.cfc` | 1 | medium | declares $draw() | `vendor/wheels/Global.cfc:2002` variable 'application[].mapper' has no component ref |
| 1 | `application[].pluginobj → vendor/wheels/PackageLoader.cfc` | 2 | low | declares getMethodProviders() (2 candidates) | `vendor/wheels/Global.cfc:3752` variable 'application[].PluginObj' has no component ref |
| 1 | `appmetadata.customtagpaths` | 0 | none |  | `vendor/wheels/wheelstest/system/util/CFMappingHelper.cfc:23` variable 'appMetadata.customTagPaths' has no component ref |
| 1 | `appsettings.mappings` | 1 | none |  | `vendor/wheels/wheelstest/system/util/LuceeMappingHelper.cfc:46` variable 'appSettings.mappings' has no component ref |
| 1 | `argorderedtree[]` | 0 | none |  | `vendor/wheels/wheelstest/system/MockBox.cfc:645` variable 'argOrderedTree[]' has no component ref |
| 1 | `arguments.$partial → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/controller/rendering.cfc:524` variable 'arguments.$partial' has no component ref |
| 1 | `arguments.$partial[] → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/controller/rendering.cfc:527` variable 'arguments.$partial[]' has no component ref |
| 1 | `arguments.adapter → vendor/wheels/databaseAdapters/CockroachDB/CockroachDBMigrator.cfc` | 7 | low | declares adapterName() (7 candidates) | `vendor/wheels/migrator/TableDefinition.cfc:111` variable 'arguments.adapter' has no component ref |
| 1 | `arguments.applicationscope.$wheelsbrowserlauncher → cli/lucli/services/deploy/cli/DeployLockCli.cfc` | 3 | low | declares release() (3 candidates) | `cli/lucli/templates/app/public/Application.cfc:153` variable 'arguments.applicationScope.$wheelsBrowserLauncher' has no component ref |
| 1 | `arguments.applicationscope.$wheelsbrowserlauncher → vendor/wheels/wheelstest/BrowserLauncher.cfc` | 3 | low | declares release() (3 candidates) | `public/Application.cfc:153` variable 'arguments.applicationScope.$wheelsBrowserLauncher' has no component ref |
| 1 | `arguments.arglist` | 0 | none |  | `vendor/wheels/wheelstest/system/util/MixerUtil.cfc:193` variable 'arguments.argList' has no component ref |
| 1 | `arguments.bundlestats.globalexception.stacktrace` | 0 | none |  | `vendor/wheels/wheelstest/system/reports/ANTJUnitReporter.cfc:120` variable 'arguments.bundleStats.globalException.stackTrace' has no component ref |
| 1 | `arguments.cfg → cli/lucli/services/deploy/config/Registry.cfc` | 1 | high | declares password(); named like the receiver 'registry' | `cli/lucli/services/deploy/cli/DeployRegistryCli.cfc:79` variable 'arguments.cfg' has no component ref |
| 1 | `arguments.collection → vendor/wheels/model/query/QueryBuilder.cfc` | 1 | medium | declares where() | `vendor/wheels/tests/_assets/policies/PostPolicy.cfc:31` variable 'arguments.collection' has no component ref |
| 1 | `arguments.collection[]` | 0 | none |  | `vendor/wheels/wheelstest/system/reports/ANTJUnitReporter.cfc:271` variable 'arguments.collection[]' has no component ref |
| 1 | `arguments.config → cli/lucli/services/deploy/config/Config.cfc` | 1 | high | declares ssh(); named like the receiver 'config' | `cli/lucli/services/deploy/lib/SshPoolFactory.cfc:43` variable 'arguments.config' has no component ref |
| 1 | `arguments.config.datasourceinfo` | 0 | none |  | `cli/src/models/EnvironmentService.cfc:824` variable 'arguments.config.datasourceInfo' has no component ref |
| 1 | `arguments.mapper → vendor/wheels/Mapper.cfc` | 1 | high | declares $patternToRegex(); named like the receiver 'mapper' | `vendor/wheels/Dispatch.cfc:229` variable 'arguments.mapper' has no component ref |
| 1 | `arguments.md.extends → vendor/wheels/model/calculations.cfc` | 4 | low | declares count() (4 candidates) | `vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc:461` variable 'arguments.md.extends' has no component ref |
| 1 | `arguments.modelinstance → vendor/wheels/model/properties.cfc` | 1 | medium | declares classInfo() | `cli/src/models/AdminIntrospectionService.cfc:32` variable 'arguments.modelInstance' has no component ref |
| 1 | `arguments.obj → vendor/wheels/model/errors.cfc` | 2 | low | declares errorsOn() (2 candidates) | `vendor/wheels/tests/specs/model/validationsSpec.cfc:1117` variable 'arguments.obj' has no component ref |
| 1 | `arguments.obj → vendor/wheels/model/validations.cfc` | 2 | low | declares valid() (2 candidates) | `vendor/wheels/tests/specs/model/validationsSpec.cfc:1107` variable 'arguments.obj' has no component ref |
| 1 | `arguments.object → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/controller/rendering.cfc:749` variable 'arguments.object' has no component ref |
| 1 | `arguments.object → vendor/wheels/model/miscellaneous.cfc` | 1 | medium | declares $objectId() | `vendor/wheels/model/miscellaneous.cfc:292` variable 'arguments.object' has no component ref |
| 1 | `arguments.plugin → vendor/wheels/tests/_assets/plugins/lifecycle/TestLifecyclePluginA/TestLifecyclePluginA.cfc` | 6 | low | declares onPluginLoad() (6 candidates) | `vendor/wheels/Plugins.cfc:750` variable 'arguments.plugin' has no component ref |
| 1 | `arguments.primarykeys[] → vendor/wheels/migrator/ColumnDefinition.cfc` | 1 | medium | declares toPrimaryKeySQL() | `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc:51` variable 'arguments.primaryKeys[]' has no component ref |
| 1 | `arguments.record → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/controller/authorization.cfc:199` variable 'arguments.record' has no component ref |
| 1 | `arguments.targetobject → vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc` | 3 | low | declares $include() (3 candidates) | `vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc:274` variable 'arguments.targetObject' has no component ref |
| 1 | `arguments.user → vendor/wheels/model/properties.cfc` | 2 | low | declares properties() (2 candidates) | `examples/starter-app/app/global/auth.cfm:99` variable 'arguments.user' has no component ref |
| 1 | `arguments[] → vendor/wheels/model/properties.cfc` | 2 | low | declares properties() (2 candidates) | `vendor/wheels/controller/rendering.cfc:752` variable 'arguments[]' has no component ref |
| 1 | `attributes.srv → vendor/wheels/wheelstest/DialogConsumer.cfc` | 1 | medium | declares accept() | `cli/lucli/tests/StubHttpServer.cfc:41` variable 'attributes.srv' has no component ref |
| 1 | `boxdata.dependencies` | 0 | none |  | `cli/src/models/BaseCommand.cfc:21` variable 'boxData.dependencies' has no component ref |
| 1 | `browsertype → vendor/wheels/rocketunit_tests/_assets/models/Post.cfc` | 2 | low | declares getClass() (2 candidates) | `vendor/wheels/wheelstest/BrowserLauncher.cfc:293` variable 'browserType' has no component ref |
| 1 | `bundlesfound[]` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:885` variable 'bundlesFound[]' has no component ref |
| 1 | `byname.api → cli/lucli/services/deploy/config/Role.cfc` | 1 | medium | declares runningProxy() | `cli/lucli/tests/specs/deploy/config/RoleSpec.cfc:42` variable 'byName.api' has no component ref |
| 1 | `byname.workers → cli/lucli/services/deploy/config/Role.cfc` | 1 | medium | declares runningProxy() | `cli/lucli/tests/specs/deploy/config/RoleSpec.cfc:43` variable 'byName.workers' has no component ref |
| 1 | `cached` | 0 | none |  | `vendor/wheels/wheelstest/BrowserLauncher.cfc:258` variable 'cached' has no component ref |
| 1 | `cmds → cli/lucli/services/TestRunner.cfc` | 637 | low | declares run() (633 candidates) | `cli/lucli/services/deploy/cli/DeployAppCli.cfc:31` variable 'cmds' has no component ref |
| 1 | `cmds → cli/lucli/services/deploy/cli/DeployLockCli.cfc` | 3 | low | declares status() (3 candidates) | `cli/lucli/services/deploy/cli/DeployAppCli.cfc:52` variable 'cmds' has no component ref |
| 1 | `comparewith` | 0 | none |  | `vendor/wheels/tests/specs/model/crudSpec.cfc:125` variable 'compareWith' has no component ref |
| 1 | `compilermethod → cli/src/models/DetailOutputService.cfc` | 1 | medium | declares invoke() | `cli/lucli/services/deploy/lib/Mustache.cfc:88` variable 'compilerMethod' has no component ref |
| 1 | `config.moduleconfig → cli/src/ModuleConfig.cfc` | 1 | high | declares onUnload(); named like the receiver 'moduleConfig' | `vendor/wheels/wheelstest/system/TestBox.cfc:501` variable 'config.moduleConfig' has no component ref |
| 1 | `containerreceived → vendor/wheels/Injector.cfc` | 3 | low | declares getInstance() (2 candidates) | `vendor/wheels/tests/_assets/plugins/serviceprovider/TestServiceProvider/TestServiceProvider.cfc:36` variable 'containerReceived' has no component ref |
| 1 | `core → examples/starter-app/plugins/FlashMessagesBootstrap/FlashMessagesBootstrap.cfc` | 2 | low | declares flashMessages() (2 candidates) | `examples/starter-app/plugins/FlashMessagesBootstrap/FlashMessagesBootstrap.cfc:10` variable 'core' has no component ref |
| 1 | `createmethod → cli/src/models/DetailOutputService.cfc` | 1 | medium | declares invoke() | `vendor/wheels/wheelstest/BrowserLauncher.cfc:287` variable 'createMethod' has no component ref |
| 1 | `ctx.mw → vendor/wheels/middleware/AuthMiddleware.cfc` | 16 | low | declares handle() (16 candidates) | `vendor/wheels/middleware/Pipeline.cfc:55` variable 'ctx.mw' has no component ref |
| 1 | `dummycontroller → vendor/wheels/controller/csrf.cfc` | 1 | medium | declares $generateCookieAuthenticityToken() | `vendor/wheels/rocketunit_tests/env.cfm:58` variable 'dummyController' has no component ref |
| 1 | `e.cause` | 0 | none |  | `vendor/wheels/wheelstest/BrowserLauncher.cfc:310` variable 'e.cause' has no component ref |
| 1 | `fk → vendor/wheels/migrator/ForeignKeyDefinition.cfc` | 1 | medium | declares toForeignKeySQL() | `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc:71` variable 'fk' has no component ref |
| 1 | `future → cli/lucli/services/deploy/lib/SecretResolver.cfc` | 13 | low | declares get() (13 candidates) | `cli/lucli/services/deploy/lib/SshPool.cfc:77` variable 'future' has no component ref |
| 1 | `generateseeder → vendor/wheels/Seeder.cfc` | 1 | medium | declares generateSeeds() | `vendor/wheels/public/CliBridge.cfc:920` variable 'generateSeeder' has no component ref |
| 1 | `i → vendor/wheels/model/miscellaneous.cfc` | 1 | medium | declares $objectid() | `vendor/wheels/tests/specs/model/miscellaneousSpec.cfc:42` variable 'i' has no component ref |
| 1 | `idata.type → vendor/wheels/wheelstest/system/reports/ANTJUnitReporter.cfc` | 4 | low | declares runReport() (4 candidates) | `vendor/wheels/wheelstest/system/TestBox.cfc:652` variable 'iData.type' has no component ref |
| 1 | `idx.i → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `vendor/wheels/tests/specs/model/nestedpropertiesSpec.cfc:149` variable 'idx.i' has no component ref |
| 1 | `instream` | 0 | none |  | `cli/lucli/tests/StubHttpServer.cfc:52` variable 'inStream' has no component ref |
| 1 | `item → vendor/wheels/wheelstest/system/BaseSpec.cfc` | 2 | medium | declares beforeEach() | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1188` variable 'item' has no component ref |
| 1 | `klass` | 0 | none |  | `cli/lucli/services/deploy/lib/JarLoader.cfc:109` variable 'klass' has no component ref |
| 1 | `launcher → vendor/wheels/tests/_assets/dispatch/InvokeMethodFixture.cfc` | 2 | low | declares getState() (2 candidates) | `vendor/wheels/tests/specs/wheelstest/BrowserTestLifecycleSpec.cfc:38` variable 'launcher' has no component ref |
| 1 | `launchmethod → cli/src/models/DetailOutputService.cfc` | 1 | medium | declares invoke() | `vendor/wheels/wheelstest/BrowserLauncher.cfc:294` variable 'launchMethod' has no component ref |
| 1 | `loaderoptionsctor → cli/lucli/services/deploy/lib/JarLoader.cfc` | 1 | medium | declares newInstance() | `cli/lucli/services/deploy/lib/Yaml.cfc:123` variable 'loaderOptionsCtor' has no component ref |
| 1 | `local.associationmodel → vendor/wheels/model/errors.cfc` | 2 | low | declares allErrors() (2 candidates) | `vendor/wheels/model/errors.cfc:72` variable 'local.associationModel' has no component ref |
| 1 | `local.candidate.strategy → vendor/wheels/auth/AuthStrategy.cfc` | 8 | low | declares supports() (8 candidates) | `vendor/wheels/auth/Authenticator.cfc:120` variable 'local.candidate.strategy' has no component ref |
| 1 | `local.classdata.adapter → vendor/wheels/model/query/QueryBuilder.cfc` | 3 | low | declares $quoteValue() (3 candidates) | `vendor/wheels/model/query/QueryBuilder.cfc:534` variable 'local.classData.adapter' has no component ref |
| 1 | `local.coerced` | 0 | none |  | `vendor/wheels/Global.cfc:3132` variable 'local.coerced' has no component ref |
| 1 | `local.componentreference → vendor/wheels/model/miscellaneous.cfc` | 2 | low | declares primaryKeys() (2 candidates) | `vendor/wheels/model/onmissingmethod.cfc:511` variable 'local.componentReference' has no component ref |
| 1 | `local.currentmodelobject → vendor/wheels/model/sql.cfc` | 1 | medium | declares $expandedAssociations() | `vendor/wheels/view/miscellaneous.cfc:576` variable 'local.currentModelObject' has no component ref |
| 1 | `local.dbadapter → vendor/wheels/channel/DatabaseAdapter.cfc` | 1 | medium | declares poll() | `vendor/wheels/controller/channels.cfc:250` variable 'local.dbAdapter' has no component ref |
| 1 | `local.defaultentry.strategy → vendor/wheels/auth/AuthStrategy.cfc` | 8 | low | declares supports() (8 candidates) | `vendor/wheels/auth/Authenticator.cfc:285` variable 'local.defaultEntry.strategy' has no component ref |
| 1 | `local.expectedsig` | 0 | none |  | `vendor/wheels/auth/JwtService.cfc:136` variable 'local.expectedSig' has no component ref |
| 1 | `local.intermediatemodel → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/model/sql.cfc:1201` variable 'local.intermediateModel' has no component ref |
| 1 | `local.jobinstance → vendor/wheels/Job.cfc` | 6 | low | declares perform() (6 candidates) | `vendor/wheels/Job.cfc:373` variable 'local.jobInstance' has no component ref |
| 1 | `local.model → vendor/wheels/model/miscellaneous.cfc` | 3 | low | declares primaryKey() (3 candidates) | `vendor/wheels/model/nestedproperties.cfc:211` variable 'local.model' has no component ref |
| 1 | `local.modelclass → vendor/wheels/model/read.cfc` | 3 | low | declares findByKey() (3 candidates) | `vendor/wheels/Dispatch.cfc:791` variable 'local.modelClass' has no component ref |
| 1 | `local.modelinstance → vendor/wheels/Model.cfc` | 1 | medium | declares $classData() | `vendor/wheels/public/CliBridge.cfc:468` variable 'local.modelInstance' has no component ref |
| 1 | `local.modelobj` | 14 | none |  | `vendor/wheels/Seeder.cfc:207` variable 'local.modelObj' has no component ref |
| 1 | `local.modelobj → vendor/wheels/model/miscellaneous.cfc` | 2 | low | declares tableName() (2 candidates) | `vendor/wheels/migrator/AutoMigrator.cfc:197` variable 'local.modelObj' has no component ref |
| 1 | `local.modelobj → vendor/wheels/model/read.cfc` | 4 | low | declares findOne() (4 candidates) | `vendor/wheels/Seeder.cfc:193` variable 'local.modelObj' has no component ref |
| 1 | `local.object → vendor/wheels/model/delete.cfc` | 19 | low | declares delete() (19 candidates) | `vendor/wheels/model/delete.cfc:155` variable 'local.object' has no component ref |
| 1 | `local.object → vendor/wheels/model/update.cfc` | 18 | low | declares update() (18 candidates) | `vendor/wheels/model/update.cfc:214` variable 'local.object' has no component ref |
| 1 | `local.parts[]` | 0 | none |  | `vendor/wheels/auth/JwtService.cfc:137` variable 'local.parts[]' has no component ref |
| 1 | `local.pattern → vendor/wheels/wheelstest/system/Assertion.cfc` | 2 | medium | declares startsWith() | `cli/src/commands/wheels/watch.cfc:192` variable 'local.pattern' has no component ref |
| 1 | `local.plugin → vendor/wheels/tests/_assets/plugins/lifecycle/TestLifecyclePluginA/TestLifecyclePluginA.cfc` | 4 | low | declares onPluginActivate() (4 candidates) | `vendor/wheels/Plugins.cfc:525` variable 'local.plugin' has no component ref |
| 1 | `local.policy → vendor/wheels/Policy.cfc` | 6 | medium | declares scope(); named like the receiver 'policy' (6 candidates) | `vendor/wheels/controller/authorization.cfc:148` variable 'local.policy' has no component ref |
| 1 | `local.reg → vendor/wheels/services/packages/Registry.cfc` | 3 | low | declares listAll() (3 candidates) | `vendor/wheels/Public.cfc:287` variable 'local.reg' has no component ref |
| 1 | `local.request` | 0 | none |  | `vendor/wheels/engineAdapters/BoxLang/BoxLangAdapter.cfc:43` variable 'local.request' has no component ref |
| 1 | `local.results.migrations` | 0 | none |  | `cli/src/commands/wheels/dbmigrate/info.cfc:17` variable 'local.results.migrations' has no component ref |
| 1 | `local.rsmd` | 0 | none |  | `cli/src/commands/wheels/db/dump.cfc:756` variable 'local.rsmd' has no component ref |
| 1 | `local.rv → vendor/wheels/Controller.cfc` | 1 | medium | declares $createControllerObject() | `vendor/wheels/Global.cfc:1428` variable 'local.rv' has no component ref |
| 1 | `local.rv → vendor/wheels/model/create.cfc` | 2 | low | declares save() (2 candidates) | `vendor/wheels/model/create.cfc:33` variable 'local.rv' has no component ref |
| 1 | `local.rv → vendor/wheels/model/properties.cfc` | 1 | medium | declares $setDefaultValues() | `vendor/wheels/model/create.cfc:66` variable 'local.rv' has no component ref |
| 1 | `local.s → vendor/wheels/auth/AuthStrategy.cfc` | 14 | low | declares getName() (14 candidates) | `vendor/wheels/auth/Authenticator.cfc:39` variable 'local.s' has no component ref |
| 1 | `local.strategy → vendor/wheels/auth/SessionStrategy.cfc` | 1 | medium | declares currentUser() | `vendor/wheels/controller/authorization.cfc:241` variable 'local.strategy' has no component ref |
| 1 | `local.temp.folderpath → vendor/wheels/wheelstest/system/Assertion.cfc` | 2 | medium | declares startsWith() | `vendor/wheels/Plugins.cfc:95` variable 'local.temp.folderPath' has no component ref |
| 1 | `local.val` | 0 | none |  | `vendor/wheels/Global.cfc:3132` variable 'local.val' has no component ref |
| 1 | `local.value → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `vendor/wheels/Global.cfc:1923` variable 'local.value' has no component ref |
| 1 | `loctors[]` | 0 | none |  | `cli/lucli/services/deploy/lib/Yaml.cfc:112` variable 'loCtors[]' has no component ref |
| 1 | `map → vendor/wheels/mapper/scoping.cfc` | 3 | low | declares group() (3 candidates) | `vendor/wheels/tests/specs/mapperModernSpec.cfc:82` variable 'map' has no component ref |
| 1 | `mapper2 → vendor/wheels/Global.cfc` | 13 | low | declares get() (13 candidates) | `vendor/wheels/tests/specs/global/urlforSpec.cfc:250` variable 'mapper2' has no component ref |
| 1 | `mustacheclass` | 0 | none |  | `cli/lucli/services/deploy/lib/Mustache.cfc:73` variable 'mustacheClass' has no component ref |
| 1 | `newrecord → vendor/wheels/model/create.cfc` | 2 | low | declares save() (2 candidates) | `vendor/wheels/Seeder.cfc:287` variable 'newRecord' has no component ref |
| 1 | `obj.gallery.photos[] → vendor/wheels/model/properties.cfc` | 2 | low | declares properties() (2 candidates) | `vendor/wheels/tests/specs/model/callbacksSpec.cfc:468` variable 'obj.gallery.photos[]' has no component ref |
| 1 | `omockgenerator → vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc` | 2 | medium | declares generate(); named like the receiver 'oMockGenerator' (2 candidates) | `vendor/wheels/wheelstest/system/MockBox.cfc:476` variable 'oMockGenerator' has no component ref |
| 1 | `os` | 2 | none |  | `cli/lucli/services/deploy/lib/SshClient.cfc:139` variable 'os' has no component ref |
| 1 | `pagecontext → cli/src/models/AnalysisService.cfc` | 2 | medium | declares getConfig() | `tools/ci/setup-datasources.cfm:24` variable 'pageContext' has no component ref |
| 1 | `paramtypes[] → vendor/wheels/auth/AuthStrategy.cfc` | 14 | low | declares getName() (14 candidates) | `cli/lucli/services/deploy/lib/Yaml.cfc:130` variable 'paramTypes[]' has no component ref |
| 1 | `parentsuite → vendor/wheels/wheelstest/system/BaseSpec.cfc` | 2 | medium | declares afterEach() | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1335` variable 'parentSuite' has no component ref |
| 1 | `permission → examples/starter-app/app/controllers/Accounts.cfc` | 18 | low | declares update() (18 candidates) | `examples/starter-app/app/models/Permission.cfc:56` variable 'permission' has no component ref |
| 1 | `photo.filedata` | 0 | none |  | `vendor/wheels/tests/specs/global/internalSpec.cfc:304` variable 'photo.filedata' has no component ref |
| 1 | `pkginfo.lastmodified` | 0 | none |  | `cli/lucli/tests/specs/packages/InstallerSpec.cfc:76` variable 'pkgInfo.lastModified' has no component ref |
| 1 | `r[].file` | 0 | none |  | `vendor/wheels/tests/specs/controller/miscellaneousSpec.cfc:116` variable 'r[].file' has no component ref |
| 1 | `request.wheels.toxml → vendor/wheels/wheelstest/system/util/XMLConverter.cfc` | 1 | medium | declares toXml() | `vendor/wheels/Global.cfc:3343` variable 'request.wheels.toXml' has no component ref |
| 1 | `result.author → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `vendor/wheels/tests/specs/dispatch/routeModelBindingSpec.cfc:106` variable 'result.author' has no component ref |
| 1 | `result.belongsto` | 1 | none |  | `cli/lucli/Module.cfc:7213` variable 'result.belongsTo' has no component ref |
| 1 | `result.config.datasourceinfo` | 0 | none |  | `cli/src/commands/wheels/env/setup.cfc:211` variable 'result.config.datasourceInfo' has no component ref |
| 1 | `result.hasmany` | 1 | none |  | `cli/lucli/Module.cfc:7216` variable 'result.hasMany' has no component ref |
| 1 | `result.hasone` | 1 | none |  | `cli/lucli/Module.cfc:7219` variable 'result.hasOne' has no component ref |
| 1 | `results.issues` | 1 | none |  | `cli/src/models/SecurityService.cfc:33` variable 'results.issues' has no component ref |
| 1 | `runner → vendor/wheels/wheelstest/system/runners/BaseRunner.cfc` | 1 | medium | declares isSuiteFocused() | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1013` variable 'runner' has no component ref |
| 1 | `safector → cli/lucli/services/deploy/lib/JarLoader.cfc` | 1 | medium | declares newInstance() | `cli/lucli/services/deploy/lib/Yaml.cfc:143` variable 'safeCtor' has no component ref |
| 1 | `scoped → vendor/wheels/wheelstest/BrowserClient.cfc` | 1 | medium | declares fill() | `vendor/wheels/tests/specs/wheelstest/BrowserIntegrationSpec.cfc:316` variable 'scoped' has no component ref |
| 1 | `seeder → vendor/wheels/Seeder.cfc` | 1 | high | declares runSeeds(); named like the receiver 'seeder' | `vendor/wheels/public/CliBridge.cfc:900` variable 'seeder' has no component ref |
| 1 | `server.system.out → vendor/wheels/wheelstest/system/BaseSpec.cfc` | 2 | medium | declares println() | `vendor/wheels/Migrator.cfc:1218` variable 'server.system.out' has no component ref |
| 1 | `serverjson` | 0 | none |  | `cli/src/models/EnvironmentService.cfc:1037` variable 'serverJson' has no component ref |
| 1 | `sess` | 2 | none |  | `cli/lucli/services/deploy/lib/SshClient.cfc:173` variable 'sess' has no component ref |
| 1 | `sess → cli/lucli/services/deploy/cli/DeployServerCli.cfc` | 1 | medium | declares exec() | `cli/lucli/services/deploy/lib/SshClient.cfc:131` variable 'sess' has no component ref |
| 1 | `setting → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `examples/starter-app/app/views/admin/settings/edit.cfm:4` variable 'setting' has no component ref |
| 1 | `sftp → cli/lucli/services/deploy/lib/SecretResolver.cfc` | 13 | low | declares get() (13 candidates) | `cli/lucli/services/deploy/lib/SshClient.cfc:323` variable 'sftp' has no component ref |
| 1 | `sftp → cli/lucli/services/deploy/lib/SshClient.cfc` | 2 | low | declares close() (2 candidates) | `cli/lucli/services/deploy/lib/SshClient.cfc:325` variable 'sftp' has no component ref |
| 1 | `snippet → cli/lucli/Module.cfc` | 2 | low | declares generate() (2 candidates) | `cli/lucli/Module.cfc:4065` variable 'snippet' has no component ref |
| 1 | `stats.error.stacktrace` | 0 | none |  | `vendor/wheels/wheelstest/system/reports/ANTJUnitReporter.cfc:220` variable 'stats.error.stackTrace' has no component ref |
| 1 | `svc` | 2 | none |  | `vendor/wheels/tests/specs/di/InjectorSpec.cfc:337` variable 'svc' has no component ref |
| 1 | `svc → cli/lucli/Module.cfc` | 2 | low | declares generateAdmin() (2 candidates) | `cli/lucli/Module.cfc:4132` variable 'svc' has no component ref |
| 1 | `svc → cli/lucli/services/Doctor.cfc` | 1 | medium | declares runChecks() | `cli/lucli/Module.cfc:2235` variable 'svc' has no component ref |
| 1 | `template → vendor/wheels/migrator/Migration.cfc` | 1 | medium | declares execute() | `cli/lucli/services/deploy/lib/Mustache.cfc:99` variable 'template' has no component ref |
| 1 | `templates → cli/lucli/services/Templates.cfc` | 1 | high | declares getTemplateDir(); named like the receiver 'templates' | `cli/lucli/Module.cfc:4434` variable 'templates' has no component ref |
| 1 | `testbundles → vendor/wheels/Injector.cfc` | 3 | low | declares map() (3 candidates) | `vendor/wheels/wheelstest/system/runners/BaseRunner.cfc:206` variable 'testBundles' has no component ref |
| 1 | `this._mockcallbacks[] → vendor/wheels/model/query/QueryBuilder.cfc` | 1 | medium | declares first() | `public/testbox/system/stubs/F952D54F1096E25C030C8E3149ABD8C4.cfm:69` variable 'this._mockCallbacks[]' has no component ref |
| 1 | `this.containerreceived → vendor/wheels/Injector.cfc` | 2 | low | declares containsInstance() (2 candidates) | `vendor/wheels/tests/_assets/plugins/serviceprovider/TestServiceProvider/TestServiceProvider.cfc:35` variable 'this.containerReceived' has no component ref |
| 1 | `this[] → vendor/wheels/model/properties.cfc` | 2 | low | declares setProperties() (2 candidates) | `vendor/wheels/model/nestedproperties.cfc:176` variable 'this[]' has no component ref |
| 1 | `this[][] → vendor/wheels/model/miscellaneous.cfc` | 1 | medium | declares compareTo() | `vendor/wheels/model/nestedproperties.cfc:236` variable 'this[][]' has no component ref |
| 1 | `this[][] → vendor/wheels/model/properties.cfc` | 2 | low | declares setProperties() (2 candidates) | `vendor/wheels/model/nestedproperties.cfc:291` variable 'this[][]' has no component ref |
| 1 | `thisfunction` | 0 | none |  | `vendor/wheels/wheelstest/system/util/Util.cfc:180` variable 'thisFunction' has no component ref |
| 1 | `thisfunction.annotations` | 0 | none |  | `vendor/wheels/wheelstest/system/util/Util.cfc:180` variable 'thisFunction.annotations' has no component ref |
| 1 | `thisparam` | 1 | none |  | `vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc:129` variable 'thisParam' has no component ref |
| 1 | `user → examples/starter-app/app/controllers/Accounts.cfc` | 18 | low | declares update() (18 candidates) | `examples/starter-app/app/models/User.cfc:147` variable 'user' has no component ref |
| 1 | `user → vendor/wheels/model/create.cfc` | 2 | low | declares save() (2 candidates) | `examples/starter-app/app/models/User.cfc:135` variable 'user' has no component ref |
| 1 | `user.author.profile → vendor/wheels/model/errors.cfc` | 2 | low | declares addError() (2 candidates) | `vendor/wheels/tests/_assets/controllers/ControllerWithNestedModelErrors.cfc:8` variable 'user.author.profile' has no component ref |
| 1 | `v1 → vendor/wheels/Global.cfc` | 13 | low | declares get() (13 candidates) | `vendor/wheels/tests/specs/mapper/MapperRobustnessSpec.cfc:172` variable 'v1' has no component ref |
| 1 | `v1 → vendor/wheels/mapper/resources.cfc` | 2 | low | declares resources() (2 candidates) | `vendor/wheels/tests/specs/mapperModernSpec.cfc:171` variable 'v1' has no component ref |
| 1 | `v2 → vendor/wheels/Global.cfc` | 13 | low | declares get() (13 candidates) | `vendor/wheels/tests/specs/mapperModernSpec.cfc:142` variable 'v2' has no component ref |
| 1 | `variables.$class.layouts` | 1 | none |  | `vendor/wheels/controller/layouts.cfc:70` variable 'variables.$class.layouts' has no component ref |
| 1 | `variables.$classloader → cli/lucli/services/deploy/lib/SshClient.cfc` | 2 | low | declares close() (2 candidates) | `vendor/wheels/wheelstest/BrowserLauncher.cfc:576` variable 'variables.$classLoader' has no component ref |
| 1 | `variables.$context` | 0 | none |  | `vendor/wheels/wheelstest/BrowserTest.cfc:189` variable 'variables.$context' has no component ref |
| 1 | `variables.$context → cli/lucli/services/deploy/lib/SshClient.cfc` | 2 | low | declares close() (2 candidates) | `vendor/wheels/wheelstest/BrowserTest.cfc:203` variable 'variables.$context' has no component ref |
| 1 | `variables.$playwright → cli/lucli/services/deploy/lib/SshClient.cfc` | 2 | low | declares close() (2 candidates) | `vendor/wheels/wheelstest/BrowserLauncher.cfc:562` variable 'variables.$playwright' has no component ref |
| 1 | `variables.$scope` | 0 | none |  | `vendor/wheels/wheelstest/BrowserClient.cfc:856` variable 'variables.$scope' has no component ref |
| 1 | `variables.bundlespattern` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:870` variable 'variables.bundlesPattern' has no component ref |
| 1 | `variables.config → cli/lucli/services/deploy/config/Proxy.cfc` | 1 | high | declares healthcheck(); named like the receiver 'proxy' | `cli/lucli/services/deploy/commands/ProxyCommands.cfc:36` variable 'variables.config' has no component ref |
| 1 | `variables.config → cli/lucli/services/deploy/config/Registry.cfc` | 1 | high | declares server(); named like the receiver 'registry' | `cli/lucli/services/deploy/commands/RegistryCommands.cfc:31` variable 'variables.config' has no component ref |
| 1 | `variables.config → cli/lucli/services/deploy/config/Ssh.cfc` | 1 | high | declares user(); named like the receiver 'ssh' | `cli/lucli/services/deploy/commands/ProxyCommands.cfc:102` variable 'variables.config' has no component ref |
| 1 | `variables.jwtservice → vendor/wheels/auth/JwtService.cfc` | 1 | high | declares decode(); named like the receiver 'jwtService' | `vendor/wheels/auth/JwtStrategy.cfc:70` variable 'variables.jwtService' has no component ref |
| 1 | `variables.modelreference → vendor/wheels/model/properties.cfc` | 1 | medium | declares $escapeSqlValue() | `vendor/wheels/model/query/ScopeChain.cfc:46` variable 'variables.modelReference' has no component ref |
| 1 | `variables.page → vendor/wheels/model/read.cfc` | 4 | low | declares reload() (3 candidates) | `vendor/wheels/wheelstest/BrowserClient.cfc:60` variable 'variables.page' has no component ref |
| 1 | `variables.pool → cli/lucli/services/deploy/lib/SshPool.cfc` | 1 | medium | declares getConnection() | `cli/lucli/services/deploy/lib/SshPoolTask.cfc:25` variable 'variables.pool' has no component ref |
| 1 | `variables.simpleservice → vendor/wheels/tests/_assets/di/SimpleService.cfc` | 2 | medium | declares greet(); named like the receiver 'simpleService' (2 candidates) | `vendor/wheels/tests/_assets/di/DependentService.cfc:17` variable 'variables.simpleService' has no component ref |
| 1 | `yamlctor → cli/lucli/services/deploy/lib/JarLoader.cfc` | 1 | medium | declares newInstance() | `cli/lucli/services/deploy/lib/Yaml.cfc:162` variable 'yamlCtor' has no component ref |
| 1 | `yamlctors[]` | 0 | none |  | `cli/lucli/services/deploy/lib/Yaml.cfc:150` variable 'yamlCtors[]' has no component ref |
| 1 | `yparams[] → vendor/wheels/auth/AuthStrategy.cfc` | 14 | low | declares getName() (14 candidates) | `cli/lucli/services/deploy/lib/Yaml.cfc:151` variable 'yParams[]' has no component ref |

<details><summary>Groups with several candidates</summary>

- `application.wheels.engineadapter → vendor/wheels/engineAdapters/Base.cfc` — 58 finding(s), 2 candidate(s):
  - low `vendor/wheels/engineAdapters/Base.cfc` — declares parseFormKey()
  - low `vendor/wheels/engineAdapters/BoxLang/BoxLangAdapter.cfc` — declares parseFormKey()
- `variables.helpers → cli/lucli/services/Helpers.cfc` — 53 finding(s), 4 candidate(s):
  - low `cli/lucli/services/Helpers.cfc` — declares pluralize(), capitalize(); named like the receiver 'helpers'
  - low `cli/src/models/helpers.cfc` — declares pluralize(), capitalize(); named like the receiver 'helpers'
  - low `cli/src/models/CodeGenerationService.cfc` — declares pluralize(), capitalize()
  - low `vendor/wheels/Global.cfc` — declares pluralize(), capitalize()
- `migration.adapter → vendor/wheels/databaseAdapters/CockroachDB/CockroachDBMigrator.cfc` — 45 finding(s), 7 candidate(s):
  - low `vendor/wheels/databaseAdapters/CockroachDB/CockroachDBMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/H2/H2Migrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteMigrator.cfc` — declares adapterName()
- `binder → vendor/wheels/Injector.cfc` — 32 finding(s), 3 candidate(s):
  - low `vendor/wheels/Injector.cfc` — declares map()
  - low `vendor/wheels/interfaces/di/InjectorInterface.cfc` — declares map()
  - low `vendor/wheels/tests/_assets/plugins/serviceprovider/FakeContainer.cfc` — declares map()
- `adapter → vendor/wheels/engineAdapters/Base.cfc` — 29 finding(s), 2 candidate(s):
  - low `vendor/wheels/engineAdapters/Base.cfc` — declares invokeMethod()
  - low `vendor/wheels/engineAdapters/BoxLang/BoxLangAdapter.cfc` — declares invokeMethod()
- `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/Base.cfc` — 22 finding(s), 7 candidate(s):
  - low `vendor/wheels/databaseAdapters/Base.cfc` — declares $quoteIdentifier()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerModel.cfc` — declares $quoteIdentifier()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` — declares $quoteIdentifier()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleModel.cfc` — declares $quoteIdentifier()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLModel.cfc` — declares $quoteIdentifier()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteModel.cfc` — declares $quoteIdentifier()
  - low `vendor/wheels/interfaces/database/DatabaseModelAdapterInterface.cfc` — declares $quoteIdentifier()
- `variables.sshpool → cli/lucli/services/deploy/lib/SshPool.cfc` — 20 finding(s), 2 candidate(s):
  - medium `cli/lucli/services/deploy/lib/SshPool.cfc` — declares onEach(); named like the receiver 'sshPool'
  - low `cli/lucli/services/deploy/lib/FakeSshPool.cfc` — declares onEach()
- `arguments.injector → vendor/wheels/Injector.cfc` — 19 finding(s), 3 candidate(s):
  - low `vendor/wheels/Injector.cfc` — declares to()
  - low `vendor/wheels/interfaces/di/InjectorInterface.cfc` — declares to()
  - low `vendor/wheels/tests/_assets/plugins/serviceprovider/FakeContainer.cfc` — declares to()
- `ssh → cli/lucli/services/TestRunner.cfc` — 17 finding(s), 633 candidate(s):
  - low `cli/lucli/services/TestRunner.cfc` — declares run()
  - low `cli/lucli/services/deploy/commands/AccessoryCommands.cfc` — declares run()
  - low `cli/lucli/services/deploy/commands/AppCommands.cfc` — declares run()
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares run()
  - low `cli/lucli/tests/specs/commands/CommandArgParsingSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/commands/ConsoleCommandSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/commands/CreateCommandSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/commands/DbCommandSpec.cfc` — declares run()
  - … 625 more
- `variables.modelreference → vendor/wheels/model/query/QueryBuilder.cfc` — 15 finding(s), 4 candidate(s):
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares findOne()
  - low `vendor/wheels/model/query/ScopeChain.cfc` — declares findOne()
  - low `vendor/wheels/model/read.cfc` — declares findOne()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares findOne()
- `arguments.context.migrator → vendor/wheels/Migrator.cfc` — 14 finding(s), 3 candidate(s):
  - medium `vendor/wheels/Migrator.cfc` — declares createMigration(); named like the receiver 'migrator'
  - low `vendor/wheels/public/CliBridge.cfc` — declares createMigration()
  - low `cli/src/models/MigrationService.cfc` — declares createMigration()
- `role → cli/lucli/services/deploy/config/Role.cfc` — 12 finding(s), 2 candidate(s):
  - medium `cli/lucli/services/deploy/config/Role.cfc` — declares hosts(); named like the receiver 'role'
  - low `cli/lucli/services/deploy/config/Accessory.cfc` — declares hosts()
- `stack.cache → cli/lucli/services/packages/ManifestCache.cfc` — 12 finding(s), 7 candidate(s):
  - low `cli/lucli/services/packages/ManifestCache.cfc` — declares refresh()
  - low `cli/lucli/services/packages/PackagesRegistryCli.cfc` — declares refresh()
  - low `cli/lucli/services/packages/Registry.cfc` — declares refresh()
  - low `vendor/wheels/auth/JwtService.cfc` — declares refresh()
  - low `vendor/wheels/wheelstest/BrowserClient.cfc` — declares refresh()
  - low `vendor/wheels/services/packages/ManifestCache.cfc` — declares refresh()
  - low `vendor/wheels/services/packages/Registry.cfc` — declares refresh()
- `application.wheelsdi → vendor/wheels/Injector.cfc` — 11 finding(s), 2 candidate(s):
  - low `vendor/wheels/Injector.cfc` — declares getInstance()
  - low `vendor/wheels/interfaces/di/InjectorInterface.cfc` — declares getInstance()
- `map → vendor/wheels/mapper/resources.cfc` — 11 finding(s), 2 candidate(s):
  - low `vendor/wheels/mapper/resources.cfc` — declares resources(), resource()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares resources(), resource()
- `codegen → cli/lucli/services/CodeGen.cfc` — 10 finding(s), 2 candidate(s):
  - medium `cli/lucli/services/CodeGen.cfc` — declares validateName(), generateModel(); named like the receiver 'codegen'
  - low `cli/src/models/CodeGenerationService.cfc` — declares validateName(), generateModel()
- `application.wheels.migrator → vendor/wheels/Migrator.cfc` — 9 finding(s), 2 candidate(s):
  - medium `vendor/wheels/Migrator.cfc` — declares migrateToLatest(); named like the receiver 'migrator'
  - low `vendor/wheels/public/CliBridge.cfc` — declares migrateToLatest()
- `arguments.column → vendor/wheels/migrator/ColumnDefinition.cfc` — 9 finding(s), 2 candidate(s):
  - low `vendor/wheels/migrator/ColumnDefinition.cfc` — declares toSQL()
  - low `vendor/wheels/migrator/ForeignKeyDefinition.cfc` — declares toSQL()
- `variables.registry → cli/lucli/services/packages/Registry.cfc` — 9 finding(s), 2 candidate(s):
  - low `cli/lucli/services/packages/Registry.cfc` — declares listPackageNames(), fetchManifest(); named like the receiver 'registry'
  - low `vendor/wheels/services/packages/Registry.cfc` — declares listPackageNames(), fetchManifest(); named like the receiver 'registry'
- `variables.cache → cli/lucli/services/packages/ManifestCache.cfc` — 8 finding(s), 2 candidate(s):
  - low `cli/lucli/services/packages/ManifestCache.cfc` — declares hasFreshIndex(), readIndex(), writeIndex()
  - low `vendor/wheels/services/packages/ManifestCache.cfc` — declares hasFreshIndex(), readIndex(), writeIndex()
- `variables.cache → vendor/wheels/services/packages/ManifestCache.cfc` — 8 finding(s), 2 candidate(s):
  - low `vendor/wheels/services/packages/ManifestCache.cfc` — declares hasFreshIndex(), readIndex(), writeIndex()
  - low `cli/lucli/services/packages/ManifestCache.cfc` — declares hasFreshIndex(), readIndex(), writeIndex()
- `variables.codegenservice → cli/lucli/services/CodeGen.cfc` — 8 finding(s), 3 candidate(s):
  - low `cli/lucli/services/CodeGen.cfc` — declares generateModel(), generateController(), generateView(), generateTest()
  - low `cli/lucli/Module.cfc` — declares generateModel(), generateController(), generateView(), generateTest()
  - low `cli/src/models/CodeGenerationService.cfc` — declares generateModel(), generateController(), generateView(), generateTest()
- `cache → vendor/wheels/services/packages/ManifestCache.cfc` — 7 finding(s), 2 candidate(s):
  - low `vendor/wheels/services/packages/ManifestCache.cfc` — declares refresh(), writeManifest(), hasFreshManifest()
  - low `cli/lucli/services/packages/ManifestCache.cfc` — declares refresh(), writeManifest(), hasFreshManifest()
- `cfg → cli/lucli/services/deploy/config/Accessory.cfc` — 7 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/config/Accessory.cfc` — declares image()
  - low `cli/lucli/services/deploy/config/Config.cfc` — declares image()
- `variables.modelreference → vendor/wheels/model/query/ScopeChain.cfc` — 7 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/query/ScopeChain.cfc` — declares findByKey()
  - low `vendor/wheels/model/read.cfc` — declares findByKey()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares findByKey()
- `variables.templateservice → cli/src/models/TemplateService.cfc` — 7 finding(s), 2 candidate(s):
  - medium `cli/src/models/TemplateService.cfc` — declares generateFromTemplate(); named like the receiver 'templateService'
  - low `cli/lucli/services/Templates.cfc` — declares generateFromTemplate()
- `cache → cli/lucli/services/packages/ManifestCache.cfc` — 6 finding(s), 7 candidate(s):
  - low `cli/lucli/services/packages/ManifestCache.cfc` — declares refresh()
  - low `cli/lucli/services/packages/PackagesRegistryCli.cfc` — declares refresh()
  - low `cli/lucli/services/packages/Registry.cfc` — declares refresh()
  - low `vendor/wheels/auth/JwtService.cfc` — declares refresh()
  - low `vendor/wheels/wheelstest/BrowserClient.cfc` — declares refresh()
  - low `vendor/wheels/services/packages/ManifestCache.cfc` — declares refresh()
  - low `vendor/wheels/services/packages/Registry.cfc` — declares refresh()
- `map → vendor/wheels/Global.cfc` — 6 finding(s), 13 candidate(s):
  - low `vendor/wheels/Global.cfc` — declares get()
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares get()
  - low `vendor/wheels/mapper/matching.cfc` — declares get()
  - low `vendor/wheels/tests/_assets/packages/FakeHttpClient.cfc` — declares get()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares get()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares get()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares get()
  - low `vendor/wheels/services/packages/HttpClient.cfc` — declares get()
  - … 5 more
- `acc → cli/lucli/services/deploy/config/Accessory.cfc` — 5 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/config/Accessory.cfc` — declares env(), hosts(), name()
  - low `cli/lucli/services/deploy/config/Role.cfc` — declares env(), hosts(), name()
- `adapter → vendor/wheels/databaseAdapters/Abstract.cfc` — 5 finding(s), 8 candidate(s):
  - low `vendor/wheels/databaseAdapters/Abstract.cfc` — declares quoteTableName()
  - low `vendor/wheels/databaseAdapters/H2/H2Migrator.cfc` — declares quoteTableName()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerMigrator.cfc` — declares quoteTableName()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLMigrator.cfc` — declares quoteTableName()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc` — declares quoteTableName()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLMigrator.cfc` — declares quoteTableName()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteMigrator.cfc` — declares quoteTableName()
  - low `vendor/wheels/interfaces/database/DatabaseMigratorAdapterInterface.cfc` — declares quoteTableName()
- `arguments.role → cli/lucli/services/deploy/config/Role.cfc` — 5 finding(s), 2 candidate(s):
  - medium `cli/lucli/services/deploy/config/Role.cfc` — declares cmd(); named like the receiver 'role'
  - low `cli/lucli/services/deploy/config/Accessory.cfc` — declares cmd()
- `pub → vendor/wheels/Public.cfc` — 5 finding(s), 2 candidate(s):
  - low `vendor/wheels/Public.cfc` — declares $loadRegistryPackages()
  - low `vendor/wheels/tests/_assets/packages/FakePublic.cfc` — declares $loadRegistryPackages()
- `r → vendor/wheels/services/packages/Registry.cfc` — 5 finding(s), 2 candidate(s):
  - low `vendor/wheels/services/packages/Registry.cfc` — declares fetchManifest(), listAll()
  - low `cli/lucli/services/packages/Registry.cfc` — declares fetchManifest(), listAll()
- `scaffold → cli/lucli/services/Scaffold.cfc` — 5 finding(s), 2 candidate(s):
  - medium `cli/lucli/services/Scaffold.cfc` — declares createMigrationWithProperties(); named like the receiver 'scaffold'
  - low `cli/src/models/ScaffoldService.cfc` — declares createMigrationWithProperties()
- `variables.page → vendor/wheels/interfaces/StorageDiskInterface.cfc` — 5 finding(s), 3 candidate(s):
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares url()
  - low `vendor/wheels/storage/drivers/LocalDisk.cfc` — declares url()
  - low `vendor/wheels/storage/drivers/S3Disk.cfc` — declares url()
- `variables.semver → cli/lucli/services/SemVer.cfc` — 5 finding(s), 2 candidate(s):
  - low `cli/lucli/services/SemVer.cfc` — declares satisfiesAll(), compare(); named like the receiver 'semver'
  - low `vendor/wheels/SemVer.cfc` — declares satisfiesAll(), compare(); named like the receiver 'semver'
- `api → vendor/wheels/BuildInfo.cfc` — 4 finding(s), 6 candidate(s):
  - low `vendor/wheels/BuildInfo.cfc` — declares version()
  - low `vendor/wheels/mapper/scoping.cfc` — declares version()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares version()
  - low `cli/lucli/Module.cfc` — declares version()
  - low `cli/lucli/tests/_modules/BaseModule.cfc` — declares version()
  - low `cli/lucli/services/deploy/cli/DeployMainCli.cfc` — declares version()
- `application.wheels.dispatch → vendor/wheels/Dispatch.cfc` — 4 finding(s), 2 candidate(s):
  - medium `vendor/wheels/Dispatch.cfc` — declares $request(); named like the receiver 'dispatch'
  - low `vendor/wheels/storage/drivers/S3Disk.cfc` — declares $request()
- `application.wirebox → vendor/wheels/Injector.cfc` — 4 finding(s), 2 candidate(s):
  - low `vendor/wheels/Injector.cfc` — declares getInstance()
  - low `vendor/wheels/interfaces/di/InjectorInterface.cfc` — declares getInstance()
- `arguments.migration.cfc → vendor/wheels/migrator/Migration.cfc` — 4 finding(s), 44 candidate(s):
  - low `vendor/wheels/migrator/Migration.cfc` — declares down(), up()
  - low `vendor/wheels/migrator/templates/announce.cfc` — declares down(), up()
  - low `vendor/wheels/migrator/templates/blank.cfc` — declares down(), up()
  - low `vendor/wheels/migrator/templates/change-column.cfc` — declares down(), up()
  - low `vendor/wheels/migrator/templates/change-table.cfc` — declares down(), up()
  - low `vendor/wheels/migrator/templates/create-column.cfc` — declares down(), up()
  - low `vendor/wheels/migrator/templates/create-index.cfc` — declares down(), up()
  - low `vendor/wheels/migrator/templates/create-record.cfc` — declares down(), up()
  - … 36 more
- `cmds → cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc` — 4 finding(s), 8 candidate(s):
  - low `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc` — declares start()
  - low `cli/lucli/services/deploy/cli/DeployAppCli.cfc` — declares start()
  - low `cli/lucli/services/deploy/cli/DeployProxyCli.cfc` — declares start()
  - low `cli/lucli/services/deploy/commands/AccessoryCommands.cfc` — declares start()
  - low `cli/lucli/services/deploy/commands/AppCommands.cfc` — declares start()
  - low `cli/lucli/services/deploy/commands/ProxyCommands.cfc` — declares start()
  - low `cli/lucli/Module.cfc` — declares start()
  - low `vendor/wheels/wheelstest/system/util/MixerUtil.cfc` — declares start()
- `cmds → cli/lucli/services/deploy/cli/DeployAppCli.cfc` — 4 finding(s), 4 candidate(s):
  - low `cli/lucli/services/deploy/cli/DeployAppCli.cfc` — declares containers()
  - low `cli/lucli/services/deploy/cli/DeployPruneCli.cfc` — declares containers()
  - low `cli/lucli/services/deploy/commands/AppCommands.cfc` — declares containers()
  - low `cli/lucli/services/deploy/commands/PruneCommands.cfc` — declares containers()
- `local.conn → cli/lucli/services/deploy/lib/SshClient.cfc` — 4 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares close()
  - low `cli/lucli/services/deploy/lib/SshPool.cfc` — declares close()
- `local.env → vendor/wheels/interfaces/StorageDiskInterface.cfc` — 4 finding(s), 6 candidate(s):
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares put()
  - low `vendor/wheels/mapper/matching.cfc` — declares put()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares put()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares put()
  - low `vendor/wheels/storage/drivers/LocalDisk.cfc` — declares put()
  - low `vendor/wheels/storage/drivers/S3Disk.cfc` — declares put()
- `map → vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — 4 finding(s), 2 candidate(s):
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares get(), group()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares get(), group()
- `s3down → vendor/wheels/interfaces/StorageDiskInterface.cfc` — 4 finding(s), 3 candidate(s):
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares put(), exists()
  - low `vendor/wheels/storage/drivers/LocalDisk.cfc` — declares put(), exists()
  - low `vendor/wheels/storage/drivers/S3Disk.cfc` — declares put(), exists()
- `this.adapter → vendor/wheels/databaseAdapters/Abstract.cfc` — 4 finding(s), 2 candidate(s):
  - low `vendor/wheels/databaseAdapters/Abstract.cfc` — declares foreignKeySQL()
  - low `vendor/wheels/interfaces/database/DatabaseMigratorAdapterInterface.cfc` — declares foreignKeySQL()
- `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/CockroachDB/CockroachDBModel.cfc` — 4 finding(s), 7 candidate(s):
  - low `vendor/wheels/databaseAdapters/CockroachDB/CockroachDBModel.cfc` — declares $querySetup()
  - low `vendor/wheels/databaseAdapters/H2/H2Model.cfc` — declares $querySetup()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerModel.cfc` — declares $querySetup()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` — declares $querySetup()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleModel.cfc` — declares $querySetup()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLModel.cfc` — declares $querySetup()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteModel.cfc` — declares $querySetup()
- `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerModel.cfc` — 4 finding(s), 5 candidate(s):
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerModel.cfc` — declares $quoteIdentifier(), $getType()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` — declares $quoteIdentifier(), $getType()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleModel.cfc` — declares $quoteIdentifier(), $getType()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLModel.cfc` — declares $quoteIdentifier(), $getType()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteModel.cfc` — declares $quoteIdentifier(), $getType()
- `arguments.container → vendor/wheels/tests/_assets/plugins/serviceprovider/FakeContainer.cfc` — 3 finding(s), 3 candidate(s):
  - low `vendor/wheels/tests/_assets/plugins/serviceprovider/FakeContainer.cfc` — declares mapInstance()
  - low `vendor/wheels/Injector.cfc` — declares mapInstance()
  - low `vendor/wheels/interfaces/di/InjectorInterface.cfc` — declares mapInstance()
- `arguments.target → vendor/wheels/tests/specs/mapperModernSpec.cfc` — 3 finding(s), 60 candidate(s):
  - low `vendor/wheels/tests/specs/mapperModernSpec.cfc` — declares run(), beforeAll(), afterAll()
  - low `vendor/wheels/tests/specs/mapperSpec.cfc` — declares run(), beforeAll(), afterAll()
  - low `vendor/wheels/tests/specs/controller/requestSpec.cfc` — declares run(), beforeAll(), afterAll()
  - low `vendor/wheels/tests/specs/dispatch/RoutePrecedenceSpec.cfc` — declares run(), beforeAll(), afterAll()
  - low `vendor/wheels/tests/specs/dispatch/findMatchingRouteMegaSpec.cfc` — declares run(), beforeAll(), afterAll()
  - low `vendor/wheels/tests/specs/dispatch/findMatchingRouteSpec.cfc` — declares run(), beforeAll(), afterAll()
  - low `vendor/wheels/tests/specs/environment/ipbasedaccessSpec.cfc` — declares run(), beforeAll(), afterAll()
  - low `vendor/wheels/tests/specs/global/addToCacheSpec.cfc` — declares run(), beforeAll(), afterAll()
  - … 52 more
- `cfg → cli/lucli/services/deploy/config/Config.cfc` — 3 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/config/Config.cfc` — declares service()
  - low `vendor/wheels/Global.cfc` — declares service()
- `core → vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc` — 3 finding(s), 6 candidate(s):
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc` — declares URLFor()
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner02/Runner02.cfc` — declares URLFor()
  - low `vendor/wheels/Global.cfc` — declares URLFor()
  - low `vendor/wheels/interfaces/view/ViewLinkInterface.cfc` — declares URLFor()
  - low `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc` — declares URLFor()
  - low `vendor/wheels/tests/_assets/plugins/runner/runner02/Runner02.cfc` — declares URLFor()
- `core → vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc` — 3 finding(s), 6 candidate(s):
  - low `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc` — declares URLFor()
  - low `vendor/wheels/tests/_assets/plugins/runner/runner02/Runner02.cfc` — declares URLFor()
  - low `vendor/wheels/Global.cfc` — declares URLFor()
  - low `vendor/wheels/interfaces/view/ViewLinkInterface.cfc` — declares URLFor()
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc` — declares URLFor()
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner02/Runner02.cfc` — declares URLFor()
- `fixtures.session.scaffold → cli/lucli/services/Scaffold.cfc` — 3 finding(s), 2 candidate(s):
  - medium `cli/lucli/services/Scaffold.cfc` — declares generateAuth(); named like the receiver 'scaffold'
  - low `cli/lucli/Module.cfc` — declares generateAuth()
- `local.engine → vendor/wheels/Channel.cfc` — 3 finding(s), 3 candidate(s):
  - low `vendor/wheels/Channel.cfc` — declares publish()
  - low `vendor/wheels/Global.cfc` — declares publish()
  - low `vendor/wheels/channel/DatabaseAdapter.cfc` — declares publish()
- `local.object → vendor/wheels/model/errors.cfc` — 3 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/errors.cfc` — declares allErrors()
  - low `vendor/wheels/interfaces/model/ModelErrorInterface.cfc` — declares allErrors()
- `local.rv → vendor/wheels/model/callbacks.cfc` — 3 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/callbacks.cfc` — declares $callback()
  - low `vendor/wheels/wheelstest/system/MockBox.cfc` — declares $callback()
- `result.post → vendor/wheels/model/properties.cfc` — 3 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `s3 → vendor/wheels/interfaces/StorageDiskInterface.cfc` — 3 finding(s), 3 candidate(s):
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares url(), signedUrl()
  - low `vendor/wheels/storage/drivers/LocalDisk.cfc` — declares url(), signedUrl()
  - low `vendor/wheels/storage/drivers/S3Disk.cfc` — declares url(), signedUrl()
- `this.adapter → vendor/wheels/migrator/Migration.cfc` — 3 finding(s), 5 candidate(s):
  - low `vendor/wheels/migrator/Migration.cfc` — declares dropTable(), createTable()
  - low `vendor/wheels/databaseAdapters/Abstract.cfc` — declares dropTable(), createTable()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerMigrator.cfc` — declares dropTable(), createTable()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc` — declares dropTable(), createTable()
  - low `vendor/wheels/interfaces/database/DatabaseMigratorAdapterInterface.cfc` — declares dropTable(), createTable()
- `user.author → vendor/wheels/model/errors.cfc` — 3 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/errors.cfc` — declares addError()
  - low `vendor/wheels/interfaces/model/ModelErrorInterface.cfc` — declares addError()
- `variables.http → cli/lucli/services/packages/HttpClient.cfc` — 3 finding(s), 3 candidate(s):
  - low `cli/lucli/services/packages/HttpClient.cfc` — declares download()
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares download()
  - low `cli/lucli/tests/specs/packages/_stubs/FakeHttpClient.cfc` — declares download()
- `variables.sshpool → cli/lucli/services/deploy/lib/FakeSshPool.cfc` — 3 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/lib/FakeSshPool.cfc` — declares $setSecretValues()
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares $setSecretValues()
- `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` — 3 finding(s), 3 candidate(s):
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` — declares $defaultValues(), $querySetup(), $generatedKey()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleModel.cfc` — declares $defaultValues(), $querySetup(), $generatedKey()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteModel.cfc` — declares $defaultValues(), $querySetup(), $generatedKey()
- `variables.wheels.class.adapter → vendor/wheels/model/query/QueryBuilder.cfc` — 3 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares $quoteValue()
  - low `vendor/wheels/databaseAdapters/Base.cfc` — declares $quoteValue()
  - low `vendor/wheels/interfaces/database/DatabaseModelAdapterInterface.cfc` — declares $quoteValue()
- `adapter → vendor/wheels/migrator/ColumnDefinition.cfc` — 2 finding(s), 4 candidate(s):
  - low `vendor/wheels/migrator/ColumnDefinition.cfc` — declares addColumnOptions()
  - low `vendor/wheels/databaseAdapters/Abstract.cfc` — declares addColumnOptions()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLMigrator.cfc` — declares addColumnOptions()
  - low `vendor/wheels/interfaces/database/DatabaseMigratorAdapterInterface.cfc` — declares addColumnOptions()
- `analysis → cli/lucli/services/Analysis.cfc` — 2 finding(s), 3 candidate(s):
  - medium `cli/lucli/services/Analysis.cfc` — declares analyze(); named like the receiver 'analysis'
  - low `cli/lucli/Module.cfc` — declares analyze()
  - low `cli/src/models/AnalysisService.cfc` — declares analyze()
- `application.log → cli/lucli/Module.cfc` — 2 finding(s), 9 candidate(s):
  - low `cli/lucli/Module.cfc` — declares info()
  - low `vendor/wheels/Public.cfc` — declares info()
  - low `cli/lucli/services/MigrationRunner.cfc` — declares info()
  - low `vendor/wheels/public/CliBridge.cfc` — declares info()
  - low `cli/lucli/services/packages/ManifestCache.cfc` — declares info()
  - low `cli/lucli/services/packages/PackagesRegistryCli.cfc` — declares info()
  - low `cli/lucli/services/packages/Registry.cfc` — declares info()
  - low `vendor/wheels/services/packages/ManifestCache.cfc` — declares info()
  - … 1 more
- `arguments.collection → vendor/wheels/mapper/matching.cfc` — 2 finding(s), 3 candidate(s):
  - low `vendor/wheels/mapper/matching.cfc` — declares whereIn()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares whereIn()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares whereIn()
- `arguments.foreignkey → vendor/wheels/migrator/ColumnDefinition.cfc` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/migrator/ColumnDefinition.cfc` — declares toSQL()
  - low `vendor/wheels/migrator/ForeignKeyDefinition.cfc` — declares toSQL()
- `col → vendor/wheels/migrator/ColumnDefinition.cfc` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/migrator/ColumnDefinition.cfc` — declares toSQL()
  - low `vendor/wheels/migrator/ForeignKeyDefinition.cfc` — declares toSQL()
- `core → examples/starter-app/plugins/jsconfirm/JSConfirm.cfc` — 2 finding(s), 4 candidate(s):
  - low `examples/starter-app/plugins/jsconfirm/JSConfirm.cfc` — declares linkTo()
  - low `vendor/wheels/view/links.cfc` — declares linkTo()
  - low `vendor/wheels/interfaces/view/ViewLinkInterface.cfc` — declares linkTo()
  - low `vendor/wheels/tests/_assets/controllers/SuperOverride.cfc` — declares linkTo()
- `core → vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner02/Runner02.cfc` — 2 finding(s), 6 candidate(s):
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner02/Runner02.cfc` — declares URLFor()
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc` — declares URLFor()
  - low `vendor/wheels/Global.cfc` — declares URLFor()
  - low `vendor/wheels/interfaces/view/ViewLinkInterface.cfc` — declares URLFor()
  - low `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc` — declares URLFor()
  - low `vendor/wheels/tests/_assets/plugins/runner/runner02/Runner02.cfc` — declares URLFor()
- `core → vendor/wheels/tests/_assets/plugins/runner/runner02/Runner02.cfc` — 2 finding(s), 6 candidate(s):
  - low `vendor/wheels/tests/_assets/plugins/runner/runner02/Runner02.cfc` — declares URLFor()
  - low `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc` — declares URLFor()
  - low `vendor/wheels/Global.cfc` — declares URLFor()
  - low `vendor/wheels/interfaces/view/ViewLinkInterface.cfc` — declares URLFor()
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc` — declares URLFor()
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner02/Runner02.cfc` — declares URLFor()
- `ctx.ctrl → vendor/wheels/controller/rendering.cfc` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/controller/rendering.cfc` — declares renderText(), renderNothing()
  - low `vendor/wheels/interfaces/controller/ControllerRenderingInterface.cfc` — declares renderText(), renderNothing()
- `local.adapter → vendor/wheels/databaseAdapters/Base.cfc` — 2 finding(s), 8 candidate(s):
  - low `vendor/wheels/databaseAdapters/Base.cfc` — declares $acquireAdvisoryLock(), $releaseAdvisoryLock()
  - low `vendor/wheels/databaseAdapters/CockroachDB/CockroachDBModel.cfc` — declares $acquireAdvisoryLock(), $releaseAdvisoryLock()
  - low `vendor/wheels/databaseAdapters/H2/H2Model.cfc` — declares $acquireAdvisoryLock(), $releaseAdvisoryLock()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerModel.cfc` — declares $acquireAdvisoryLock(), $releaseAdvisoryLock()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` — declares $acquireAdvisoryLock(), $releaseAdvisoryLock()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleModel.cfc` — declares $acquireAdvisoryLock(), $releaseAdvisoryLock()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLModel.cfc` — declares $acquireAdvisoryLock(), $releaseAdvisoryLock()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteModel.cfc` — declares $acquireAdvisoryLock(), $releaseAdvisoryLock()
- `local.assoc → vendor/wheels/model/properties.cfc` — 2 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `local.entry.strategy → vendor/wheels/auth/AuthStrategy.cfc` — 2 finding(s), 12 candidate(s):
  - low `vendor/wheels/auth/AuthStrategy.cfc` — declares authenticate()
  - low `vendor/wheels/auth/Authenticator.cfc` — declares authenticate()
  - low `vendor/wheels/auth/AuthenticatorInterface.cfc` — declares authenticate()
  - low `vendor/wheels/auth/JwtStrategy.cfc` — declares authenticate()
  - low `vendor/wheels/auth/SessionStrategy.cfc` — declares authenticate()
  - low `vendor/wheels/auth/TokenStrategy.cfc` — declares authenticate()
  - low `vendor/wheels/tests/_assets/auth/AlwaysFailStrategy.cfc` — declares authenticate()
  - low `vendor/wheels/tests/_assets/auth/AlwaysPassStrategy.cfc` — declares authenticate()
  - … 4 more
- `local.stmt → cli/lucli/services/deploy/lib/SshClient.cfc` — 2 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares close()
  - low `cli/lucli/services/deploy/lib/SshPool.cfc` — declares close()
- `log → cli/lucli/Module.cfc` — 2 finding(s), 9 candidate(s):
  - low `cli/lucli/Module.cfc` — declares info()
  - low `cli/lucli/services/MigrationRunner.cfc` — declares info()
  - low `cli/lucli/services/packages/ManifestCache.cfc` — declares info()
  - low `cli/lucli/services/packages/PackagesRegistryCli.cfc` — declares info()
  - low `cli/lucli/services/packages/Registry.cfc` — declares info()
  - low `vendor/wheels/Public.cfc` — declares info()
  - low `vendor/wheels/public/CliBridge.cfc` — declares info()
  - low `vendor/wheels/services/packages/ManifestCache.cfc` — declares info()
  - … 1 more
- `migration.adapter → vendor/wheels/databaseAdapters/Abstract.cfc` — 2 finding(s), 7 candidate(s):
  - low `vendor/wheels/databaseAdapters/Abstract.cfc` — declares changeColumnInTable()
  - low `vendor/wheels/databaseAdapters/H2/H2Migrator.cfc` — declares changeColumnInTable()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerMigrator.cfc` — declares changeColumnInTable()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc` — declares changeColumnInTable()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLMigrator.cfc` — declares changeColumnInTable()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteMigrator.cfc` — declares changeColumnInTable()
  - low `vendor/wheels/interfaces/database/DatabaseMigratorAdapterInterface.cfc` — declares changeColumnInTable()
- `migration.cfc → vendor/wheels/migrator/Migration.cfc` — 2 finding(s), 44 candidate(s):
  - low `vendor/wheels/migrator/Migration.cfc` — declares up(), down()
  - low `vendor/wheels/migrator/templates/announce.cfc` — declares up(), down()
  - low `vendor/wheels/migrator/templates/blank.cfc` — declares up(), down()
  - low `vendor/wheels/migrator/templates/change-column.cfc` — declares up(), down()
  - low `vendor/wheels/migrator/templates/change-table.cfc` — declares up(), down()
  - low `vendor/wheels/migrator/templates/create-column.cfc` — declares up(), down()
  - low `vendor/wheels/migrator/templates/create-index.cfc` — declares up(), down()
  - low `vendor/wheels/migrator/templates/create-record.cfc` — declares up(), down()
  - … 36 more
- `permission → vendor/wheels/model/properties.cfc` — 2 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `results.author → vendor/wheels/migrator/TableDefinition.cfc` — 2 finding(s), 3 candidate(s):
  - low `vendor/wheels/migrator/TableDefinition.cfc` — declares primaryKey()
  - low `vendor/wheels/model/miscellaneous.cfc` — declares primaryKey()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares primaryKey()
- `results.shop → vendor/wheels/migrator/TableDefinition.cfc` — 2 finding(s), 3 candidate(s):
  - low `vendor/wheels/migrator/TableDefinition.cfc` — declares primaryKey()
  - low `vendor/wheels/model/miscellaneous.cfc` — declares primaryKey()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares primaryKey()
- `ssh → cli/lucli/tests/specs/deploy/lib/FakeSshPoolSpec.cfc` — 2 finding(s), 633 candidate(s):
  - low `cli/lucli/tests/specs/deploy/lib/FakeSshPoolSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/deploy/lib/MustacheSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/deploy/lib/OutputSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/deploy/lib/SecretResolverSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/deploy/lib/SshClassloadSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/deploy/lib/SshClientRedactionSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/deploy/lib/SshClientSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/deploy/lib/SshPoolDefaultsSpec.cfc` — declares run()
  - … 625 more
- `svc → cli/lucli/services/Stats.cfc` — 2 finding(s), 2 candidate(s):
  - low `cli/lucli/services/Stats.cfc` — declares getStats()
  - low `vendor/wheels/JobWorker.cfc` — declares getStats()
- `testauthor.profile → vendor/wheels/model/properties.cfc` — 2 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `variables.$class.plugins[] → vendor/wheels/ServiceProviderInterface.cfc` — 2 finding(s), 9 candidate(s):
  - low `vendor/wheels/ServiceProviderInterface.cfc` — declares register()
  - low `vendor/wheels/tests/_assets/packages_lazy_sp/lazylate/Lazylate.cfc` — declares register()
  - low `vendor/wheels/tests/_assets/packages_lazy_sp/lazysvc/Lazysvc.cfc` — declares register()
  - low `vendor/wheels/tests/_assets/packages_sp/failboot/Failboot.cfc` — declares register()
  - low `vendor/wheels/tests/_assets/packages_sp/failregister/Failregister.cfc` — declares register()
  - low `vendor/wheels/tests/_assets/packages_sp/goodsp/Goodsp.cfc` — declares register()
  - low `vendor/wheels/tests/_assets/plugins/serviceprovider/TestServiceProvider/TestServiceProvider.cfc` — declares register()
  - low `vendor/wheels/tests/_assets/plugins/serviceproviderfailing/FailingProvider/FailingProvider.cfc` — declares register()
  - … 1 more
- `variables.http → vendor/wheels/services/packages/HttpClient.cfc` — 2 finding(s), 13 candidate(s):
  - low `vendor/wheels/services/packages/HttpClient.cfc` — declares get()
  - low `vendor/wheels/Global.cfc` — declares get()
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares get()
  - low `vendor/wheels/mapper/matching.cfc` — declares get()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares get()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares get()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares get()
  - low `vendor/wheels/storage/drivers/LocalDisk.cfc` — declares get()
  - … 5 more
- `variables.wheels.class.adapter → vendor/wheels/databaseAdapters/H2/H2Model.cfc` — 2 finding(s), 6 candidate(s):
  - low `vendor/wheels/databaseAdapters/H2/H2Model.cfc` — declares $upsertSQL(), $querySetup()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerModel.cfc` — declares $upsertSQL(), $querySetup()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` — declares $upsertSQL(), $querySetup()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleModel.cfc` — declares $upsertSQL(), $querySetup()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLModel.cfc` — declares $upsertSQL(), $querySetup()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteModel.cfc` — declares $upsertSQL(), $querySetup()
- `a → cli/lucli/services/deploy/config/Accessory.cfc` — 1 finding(s), 9 candidate(s):
  - low `cli/lucli/services/deploy/config/Accessory.cfc` — declares name()
  - low `cli/lucli/services/deploy/config/Role.cfc` — declares name()
  - low `cli/lucli/services/deploy/secrets/AwsSecretsAdapter.cfc` — declares name()
  - low `cli/lucli/services/deploy/secrets/BaseAdapter.cfc` — declares name()
  - low `cli/lucli/services/deploy/secrets/BitwardenAdapter.cfc` — declares name()
  - low `cli/lucli/services/deploy/secrets/DopplerAdapter.cfc` — declares name()
  - low `cli/lucli/services/deploy/secrets/LastPassAdapter.cfc` — declares name()
  - low `cli/lucli/services/deploy/secrets/OnePasswordAdapter.cfc` — declares name()
  - … 1 more
- `adapter → cli/lucli/services/deploy/cli/DeploySecretsCli.cfc` — 1 finding(s), 8 candidate(s):
  - low `cli/lucli/services/deploy/cli/DeploySecretsCli.cfc` — declares fetch()
  - low `cli/lucli/services/deploy/secrets/AwsSecretsAdapter.cfc` — declares fetch()
  - low `cli/lucli/services/deploy/secrets/BaseAdapter.cfc` — declares fetch()
  - low `cli/lucli/services/deploy/secrets/BitwardenAdapter.cfc` — declares fetch()
  - low `cli/lucli/services/deploy/secrets/DopplerAdapter.cfc` — declares fetch()
  - low `cli/lucli/services/deploy/secrets/LastPassAdapter.cfc` — declares fetch()
  - low `cli/lucli/services/deploy/secrets/OnePasswordAdapter.cfc` — declares fetch()
  - low `cli/lucli/services/deploy/secrets/SecretAdapterInterface.cfc` — declares fetch()
- `adapter → vendor/wheels/migrator/ForeignKeyDefinition.cfc` — 1 finding(s), 6 candidate(s):
  - low `vendor/wheels/migrator/ForeignKeyDefinition.cfc` — declares addForeignKeyOptions()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerMigrator.cfc` — declares addForeignKeyOptions()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLMigrator.cfc` — declares addForeignKeyOptions()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc` — declares addForeignKeyOptions()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLMigrator.cfc` — declares addForeignKeyOptions()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteMigrator.cfc` — declares addForeignKeyOptions()
- `apimap → vendor/wheels/BuildInfo.cfc` — 1 finding(s), 6 candidate(s):
  - low `vendor/wheels/BuildInfo.cfc` — declares version()
  - low `vendor/wheels/mapper/scoping.cfc` — declares version()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares version()
  - low `cli/lucli/Module.cfc` — declares version()
  - low `cli/lucli/tests/_modules/BaseModule.cfc` — declares version()
  - low `cli/lucli/services/deploy/cli/DeployMainCli.cfc` — declares version()
- `application.$wheels.buildinfo → vendor/wheels/BuildInfo.cfc` — 1 finding(s), 6 candidate(s):
  - medium `vendor/wheels/BuildInfo.cfc` — declares version(); named like the receiver 'buildInfo'
  - low `vendor/wheels/mapper/scoping.cfc` — declares version()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares version()
  - low `cli/lucli/Module.cfc` — declares version()
  - low `cli/lucli/tests/_modules/BaseModule.cfc` — declares version()
  - low `cli/lucli/services/deploy/cli/DeployMainCli.cfc` — declares version()
- `application.$wheels.engineadapter → vendor/wheels/engineAdapters/Base.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/engineAdapters/Base.cfc` — declares prepareDIComplete()
  - low `vendor/wheels/engineAdapters/BoxLang/BoxLangAdapter.cfc` — declares prepareDIComplete()
- `application.wheels.buildinfo → vendor/wheels/BuildInfo.cfc` — 1 finding(s), 6 candidate(s):
  - medium `vendor/wheels/BuildInfo.cfc` — declares version(); named like the receiver 'buildInfo'
  - low `vendor/wheels/mapper/scoping.cfc` — declares version()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares version()
  - low `cli/lucli/Module.cfc` — declares version()
  - low `cli/lucli/tests/_modules/BaseModule.cfc` — declares version()
  - low `cli/lucli/services/deploy/cli/DeployMainCli.cfc` — declares version()
- `application.wheels.public → vendor/wheels/Public.cfc` — 1 finding(s), 2 candidate(s):
  - medium `vendor/wheels/Public.cfc` — declares $loadRegistryPackages(); named like the receiver 'public'
  - low `vendor/wheels/tests/_assets/packages/FakePublic.cfc` — declares $loadRegistryPackages()
- `application[].pluginobj → vendor/wheels/PackageLoader.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/PackageLoader.cfc` — declares getMethodProviders()
  - low `vendor/wheels/Plugins.cfc` — declares getMethodProviders()
- `arguments.adapter → vendor/wheels/databaseAdapters/CockroachDB/CockroachDBMigrator.cfc` — 1 finding(s), 7 candidate(s):
  - low `vendor/wheels/databaseAdapters/CockroachDB/CockroachDBMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/H2/H2Migrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLMigrator.cfc` — declares adapterName()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteMigrator.cfc` — declares adapterName()
- `arguments.applicationscope.$wheelsbrowserlauncher → cli/lucli/services/deploy/cli/DeployLockCli.cfc` — 1 finding(s), 3 candidate(s):
  - low `cli/lucli/services/deploy/cli/DeployLockCli.cfc` — declares release()
  - low `cli/lucli/services/deploy/commands/LockCommands.cfc` — declares release()
  - low `vendor/wheels/wheelstest/BrowserLauncher.cfc` — declares release()
- `arguments.applicationscope.$wheelsbrowserlauncher → vendor/wheels/wheelstest/BrowserLauncher.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/wheelstest/BrowserLauncher.cfc` — declares release()
  - low `cli/lucli/services/deploy/cli/DeployLockCli.cfc` — declares release()
  - low `cli/lucli/services/deploy/commands/LockCommands.cfc` — declares release()
- `arguments.md.extends → vendor/wheels/model/calculations.cfc` — 1 finding(s), 4 candidate(s):
  - low `vendor/wheels/model/calculations.cfc` — declares count()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares count()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares count()
  - low `vendor/wheels/model/query/ScopeChain.cfc` — declares count()
- `arguments.obj → vendor/wheels/model/errors.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/errors.cfc` — declares errorsOn()
  - low `vendor/wheels/interfaces/model/ModelErrorInterface.cfc` — declares errorsOn()
- `arguments.obj → vendor/wheels/model/validations.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/validations.cfc` — declares valid()
  - low `vendor/wheels/interfaces/model/ModelValidationInterface.cfc` — declares valid()
- `arguments.plugin → vendor/wheels/tests/_assets/plugins/lifecycle/TestLifecyclePluginA/TestLifecyclePluginA.cfc` — 1 finding(s), 6 candidate(s):
  - low `vendor/wheels/tests/_assets/plugins/lifecycle/TestLifecyclePluginA/TestLifecyclePluginA.cfc` — declares onPluginLoad()
  - low `vendor/wheels/tests/_assets/plugins/lifecycle/TestLifecyclePluginB/TestLifecyclePluginB.cfc` — declares onPluginLoad()
  - low `vendor/wheels/tests/_assets/plugins/lifecyclefailing/TestLifecycleFailingA/TestLifecycleFailingA.cfc` — declares onPluginLoad()
  - low `vendor/wheels/tests/_assets/plugins/lifecyclefailing/TestLifecycleWorkingB/TestLifecycleWorkingB.cfc` — declares onPluginLoad()
  - low `vendor/wheels/tests/_assets/plugins/middleware/TestMiddlewarePluginA/TestMiddlewarePluginA.cfc` — declares onPluginLoad()
  - low `vendor/wheels/tests/_assets/plugins/middleware/TestMiddlewarePluginB/TestMiddlewarePluginB.cfc` — declares onPluginLoad()
- `arguments.targetobject → vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc` — declares $include()
  - low `vendor/wheels/Global.cfc` — declares $include()
  - low `vendor/wheels/tests/_assets/events/CorsArbitrationEventDouble.cfc` — declares $include()
- `arguments.user → vendor/wheels/model/properties.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares properties()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares properties()
- `arguments[] → vendor/wheels/model/properties.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares properties()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares properties()
- `browsertype → vendor/wheels/rocketunit_tests/_assets/models/Post.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/rocketunit_tests/_assets/models/Post.cfc` — declares getClass()
  - low `vendor/wheels/tests/_assets/models/Post.cfc` — declares getClass()
- `cmds → cli/lucli/services/TestRunner.cfc` — 1 finding(s), 633 candidate(s):
  - low `cli/lucli/services/TestRunner.cfc` — declares run()
  - low `cli/lucli/services/deploy/commands/AccessoryCommands.cfc` — declares run()
  - low `cli/lucli/services/deploy/commands/AppCommands.cfc` — declares run()
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares run()
  - low `cli/lucli/tests/specs/commands/CommandArgParsingSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/commands/ConsoleCommandSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/commands/CreateCommandSpec.cfc` — declares run()
  - low `cli/lucli/tests/specs/commands/DbCommandSpec.cfc` — declares run()
  - … 625 more
- `cmds → cli/lucli/services/deploy/cli/DeployLockCli.cfc` — 1 finding(s), 3 candidate(s):
  - low `cli/lucli/services/deploy/cli/DeployLockCli.cfc` — declares status()
  - low `cli/lucli/services/deploy/commands/AppCommands.cfc` — declares status()
  - low `cli/lucli/services/deploy/commands/LockCommands.cfc` — declares status()
- `containerreceived → vendor/wheels/Injector.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/Injector.cfc` — declares getInstance()
  - low `vendor/wheels/interfaces/di/InjectorInterface.cfc` — declares getInstance()
- `core → examples/starter-app/plugins/FlashMessagesBootstrap/FlashMessagesBootstrap.cfc` — 1 finding(s), 2 candidate(s):
  - low `examples/starter-app/plugins/FlashMessagesBootstrap/FlashMessagesBootstrap.cfc` — declares flashMessages()
  - low `vendor/wheels/view/miscellaneous.cfc` — declares flashMessages()
- `ctx.mw → vendor/wheels/middleware/AuthMiddleware.cfc` — 1 finding(s), 16 candidate(s):
  - low `vendor/wheels/middleware/AuthMiddleware.cfc` — declares handle()
  - low `vendor/wheels/middleware/BrowserTestFixtureGuard.cfc` — declares handle()
  - low `vendor/wheels/middleware/Cors.cfc` — declares handle()
  - low `vendor/wheels/middleware/MiddlewareInterface.cfc` — declares handle()
  - low `vendor/wheels/middleware/RateLimiter.cfc` — declares handle()
  - low `vendor/wheels/middleware/RequestId.cfc` — declares handle()
  - low `vendor/wheels/middleware/SecurityHeaders.cfc` — declares handle()
  - low `vendor/wheels/middleware/TenantResolver.cfc` — declares handle()
  - … 8 more
- `future → cli/lucli/services/deploy/lib/SecretResolver.cfc` — 1 finding(s), 13 candidate(s):
  - low `cli/lucli/services/deploy/lib/SecretResolver.cfc` — declares get()
  - low `cli/lucli/services/packages/HttpClient.cfc` — declares get()
  - low `cli/lucli/tests/specs/packages/_stubs/FakeHttpClient.cfc` — declares get()
  - low `vendor/wheels/Global.cfc` — declares get()
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares get()
  - low `vendor/wheels/mapper/matching.cfc` — declares get()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares get()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares get()
  - … 5 more
- `idata.type → vendor/wheels/wheelstest/system/reports/ANTJUnitReporter.cfc` — 1 finding(s), 4 candidate(s):
  - low `vendor/wheels/wheelstest/system/reports/ANTJUnitReporter.cfc` — declares runReport()
  - low `vendor/wheels/wheelstest/system/reports/IReporter.cfc` — declares runReport()
  - low `vendor/wheels/wheelstest/system/reports/JSONReporter.cfc` — declares runReport()
  - low `vendor/wheels/wheelstest/system/reports/TextReporter.cfc` — declares runReport()
- `idx.i → vendor/wheels/model/properties.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `launcher → vendor/wheels/tests/_assets/dispatch/InvokeMethodFixture.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/tests/_assets/dispatch/InvokeMethodFixture.cfc` — declares getState()
  - low `vendor/wheels/wheelstest/BrowserLauncher.cfc` — declares getState()
- `local.associationmodel → vendor/wheels/model/errors.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/errors.cfc` — declares allErrors()
  - low `vendor/wheels/interfaces/model/ModelErrorInterface.cfc` — declares allErrors()
- `local.candidate.strategy → vendor/wheels/auth/AuthStrategy.cfc` — 1 finding(s), 8 candidate(s):
  - low `vendor/wheels/auth/AuthStrategy.cfc` — declares supports()
  - low `vendor/wheels/auth/JwtStrategy.cfc` — declares supports()
  - low `vendor/wheels/auth/SessionStrategy.cfc` — declares supports()
  - low `vendor/wheels/auth/TokenStrategy.cfc` — declares supports()
  - low `vendor/wheels/tests/_assets/auth/AlwaysFailStrategy.cfc` — declares supports()
  - low `vendor/wheels/tests/_assets/auth/AlwaysPassStrategy.cfc` — declares supports()
  - low `vendor/wheels/tests/_assets/auth/HeaderTokenStrategy.cfc` — declares supports()
  - low `vendor/wheels/tests/_assets/auth/UnsupportedStrategy.cfc` — declares supports()
- `local.classdata.adapter → vendor/wheels/model/query/QueryBuilder.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares $quoteValue()
  - low `vendor/wheels/databaseAdapters/Base.cfc` — declares $quoteValue()
  - low `vendor/wheels/interfaces/database/DatabaseModelAdapterInterface.cfc` — declares $quoteValue()
- `local.componentreference → vendor/wheels/model/miscellaneous.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc` — declares primaryKeys()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares primaryKeys()
- `local.defaultentry.strategy → vendor/wheels/auth/AuthStrategy.cfc` — 1 finding(s), 8 candidate(s):
  - low `vendor/wheels/auth/AuthStrategy.cfc` — declares supports()
  - low `vendor/wheels/auth/JwtStrategy.cfc` — declares supports()
  - low `vendor/wheels/auth/SessionStrategy.cfc` — declares supports()
  - low `vendor/wheels/auth/TokenStrategy.cfc` — declares supports()
  - low `vendor/wheels/tests/_assets/auth/AlwaysFailStrategy.cfc` — declares supports()
  - low `vendor/wheels/tests/_assets/auth/AlwaysPassStrategy.cfc` — declares supports()
  - low `vendor/wheels/tests/_assets/auth/HeaderTokenStrategy.cfc` — declares supports()
  - low `vendor/wheels/tests/_assets/auth/UnsupportedStrategy.cfc` — declares supports()
- `local.jobinstance → vendor/wheels/Job.cfc` — 1 finding(s), 6 candidate(s):
  - low `vendor/wheels/Job.cfc` — declares perform()
  - low `vendor/wheels/tests/_assets/jobs/FailingBackoffJob.cfc` — declares perform()
  - low `vendor/wheels/tests/_assets/jobs/ProbeJob.cfc` — declares perform()
  - low `app/jobs/ProcessOrdersJob.cfc` — declares perform()
  - low `examples/starter-app/app/jobs/ProcessOrdersJob.cfc` — declares perform()
  - low `examples/tweet/app/jobs/ProcessOrdersJob.cfc` — declares perform()
- `local.model → vendor/wheels/model/miscellaneous.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc` — declares primaryKey()
  - low `vendor/wheels/migrator/TableDefinition.cfc` — declares primaryKey()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares primaryKey()
- `local.modelclass → vendor/wheels/model/read.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/read.cfc` — declares findByKey()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares findByKey()
  - low `vendor/wheels/model/query/ScopeChain.cfc` — declares findByKey()
- `local.modelobj → vendor/wheels/model/miscellaneous.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc` — declares tableName()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares tableName()
- `local.modelobj → vendor/wheels/model/read.cfc` — 1 finding(s), 4 candidate(s):
  - low `vendor/wheels/model/read.cfc` — declares findOne()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares findOne()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares findOne()
  - low `vendor/wheels/model/query/ScopeChain.cfc` — declares findOne()
- `local.object → vendor/wheels/model/delete.cfc` — 1 finding(s), 19 candidate(s):
  - low `vendor/wheels/model/delete.cfc` — declares delete()
  - low `vendor/wheels/Policy.cfc` — declares delete()
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares delete()
  - low `vendor/wheels/mapper/matching.cfc` — declares delete()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares delete()
  - low `vendor/wheels/interfaces/model/ModelPersistenceInterface.cfc` — declares delete()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares delete()
  - low `vendor/wheels/storage/drivers/LocalDisk.cfc` — declares delete()
  - … 11 more
- `local.object → vendor/wheels/model/update.cfc` — 1 finding(s), 18 candidate(s):
  - low `vendor/wheels/model/update.cfc` — declares update()
  - low `vendor/wheels/Policy.cfc` — declares update()
  - low `vendor/wheels/interfaces/model/ModelPersistenceInterface.cfc` — declares update()
  - low `vendor/wheels/rocketunit_tests/_assets/controllers/CsrfProtectedExcept.cfc` — declares update()
  - low `vendor/wheels/rocketunit_tests/_assets/controllers/CsrfProtectedWithException.cfc` — declares update()
  - low `vendor/wheels/tests/_assets/controllers/CsrfProtectedExcept.cfc` — declares update()
  - low `vendor/wheels/tests/_assets/controllers/CsrfProtectedWithException.cfc` — declares update()
  - low `vendor/wheels/tests/_assets/policies/PostPolicy.cfc` — declares update()
  - … 10 more
- `local.plugin → vendor/wheels/tests/_assets/plugins/lifecycle/TestLifecyclePluginA/TestLifecyclePluginA.cfc` — 1 finding(s), 4 candidate(s):
  - low `vendor/wheels/tests/_assets/plugins/lifecycle/TestLifecyclePluginA/TestLifecyclePluginA.cfc` — declares onPluginActivate()
  - low `vendor/wheels/tests/_assets/plugins/lifecycle/TestLifecyclePluginB/TestLifecyclePluginB.cfc` — declares onPluginActivate()
  - low `vendor/wheels/tests/_assets/plugins/lifecyclefailing/TestLifecycleFailingA/TestLifecycleFailingA.cfc` — declares onPluginActivate()
  - low `vendor/wheels/tests/_assets/plugins/lifecyclefailing/TestLifecycleWorkingB/TestLifecycleWorkingB.cfc` — declares onPluginActivate()
- `local.policy → vendor/wheels/Policy.cfc` — 1 finding(s), 6 candidate(s):
  - medium `vendor/wheels/Policy.cfc` — declares scope(); named like the receiver 'policy'
  - low `vendor/wheels/mapper/scoping.cfc` — declares scope()
  - low `vendor/wheels/model/properties.cfc` — declares scope()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares scope()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares scope()
  - low `vendor/wheels/tests/_assets/policies/PostPolicy.cfc` — declares scope()
- `local.reg → vendor/wheels/services/packages/Registry.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/services/packages/Registry.cfc` — declares listAll()
  - low `vendor/wheels/tests/_assets/packages/FakeRegistry.cfc` — declares listAll()
  - low `cli/lucli/services/packages/Registry.cfc` — declares listAll()
- `local.rv → vendor/wheels/model/create.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/create.cfc` — declares save()
  - low `vendor/wheels/interfaces/model/ModelPersistenceInterface.cfc` — declares save()
- `local.s → vendor/wheels/auth/AuthStrategy.cfc` — 1 finding(s), 14 candidate(s):
  - low `vendor/wheels/auth/AuthStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/JwtStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/SessionStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/TokenStrategy.cfc` — declares getName()
  - low `vendor/wheels/engineAdapters/Base.cfc` — declares getName()
  - low `vendor/wheels/tests/_assets/auth/AlwaysFailStrategy.cfc` — declares getName()
  - low `vendor/wheels/tests/_assets/auth/AlwaysPassStrategy.cfc` — declares getName()
  - low `vendor/wheels/tests/_assets/auth/HeaderTokenStrategy.cfc` — declares getName()
  - … 6 more
- `local.value → vendor/wheels/model/properties.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `map → vendor/wheels/mapper/scoping.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/mapper/scoping.cfc` — declares group()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares group()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares group()
- `mapper2 → vendor/wheels/Global.cfc` — 1 finding(s), 13 candidate(s):
  - low `vendor/wheels/Global.cfc` — declares get()
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares get()
  - low `vendor/wheels/mapper/matching.cfc` — declares get()
  - low `vendor/wheels/tests/_assets/packages/FakeHttpClient.cfc` — declares get()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares get()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares get()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares get()
  - low `vendor/wheels/services/packages/HttpClient.cfc` — declares get()
  - … 5 more
- `newrecord → vendor/wheels/model/create.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/create.cfc` — declares save()
  - low `vendor/wheels/interfaces/model/ModelPersistenceInterface.cfc` — declares save()
- `obj.gallery.photos[] → vendor/wheels/model/properties.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares properties()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares properties()
- `omockgenerator → vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc` — 1 finding(s), 2 candidate(s):
  - medium `vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc` — declares generate(); named like the receiver 'oMockGenerator'
  - low `cli/lucli/Module.cfc` — declares generate()
- `paramtypes[] → vendor/wheels/auth/AuthStrategy.cfc` — 1 finding(s), 14 candidate(s):
  - low `vendor/wheels/auth/AuthStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/JwtStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/SessionStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/TokenStrategy.cfc` — declares getName()
  - low `vendor/wheels/engineAdapters/Base.cfc` — declares getName()
  - low `tools/lucee-extensions/sqlite/src/SQLite.cfc` — declares getName()
  - low `vendor/wheels/tests/_assets/auth/AlwaysFailStrategy.cfc` — declares getName()
  - low `vendor/wheels/tests/_assets/auth/AlwaysPassStrategy.cfc` — declares getName()
  - … 6 more
- `permission → examples/starter-app/app/controllers/Accounts.cfc` — 1 finding(s), 18 candidate(s):
  - low `examples/starter-app/app/controllers/Accounts.cfc` — declares update()
  - low `examples/starter-app/app/controllers/PasswordResets.cfc` — declares update()
  - low `examples/starter-app/app/controllers/admin/Permissions.cfc` — declares update()
  - low `examples/starter-app/app/controllers/admin/Roles.cfc` — declares update()
  - low `examples/starter-app/app/controllers/admin/Settings.cfc` — declares update()
  - low `examples/starter-app/app/controllers/admin/Users.cfc` — declares update()
  - low `examples/tweet/app/controllers/Users.cfc` — declares update()
  - low `vendor/wheels/Policy.cfc` — declares update()
  - … 10 more
- `result.author → vendor/wheels/model/properties.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `setting → vendor/wheels/model/properties.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `sftp → cli/lucli/services/deploy/lib/SecretResolver.cfc` — 1 finding(s), 13 candidate(s):
  - low `cli/lucli/services/deploy/lib/SecretResolver.cfc` — declares get()
  - low `cli/lucli/services/packages/HttpClient.cfc` — declares get()
  - low `cli/lucli/tests/specs/packages/_stubs/FakeHttpClient.cfc` — declares get()
  - low `vendor/wheels/Global.cfc` — declares get()
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares get()
  - low `vendor/wheels/mapper/matching.cfc` — declares get()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares get()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares get()
  - … 5 more
- `sftp → cli/lucli/services/deploy/lib/SshClient.cfc` — 1 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares close()
  - low `cli/lucli/services/deploy/lib/SshPool.cfc` — declares close()
- `snippet → cli/lucli/Module.cfc` — 1 finding(s), 2 candidate(s):
  - low `cli/lucli/Module.cfc` — declares generate()
  - low `vendor/wheels/wheelstest/system/mockutils/MockGenerator.cfc` — declares generate()
- `svc → cli/lucli/Module.cfc` — 1 finding(s), 2 candidate(s):
  - low `cli/lucli/Module.cfc` — declares generateAdmin()
  - low `cli/lucli/services/Admin.cfc` — declares generateAdmin()
- `testbundles → vendor/wheels/Injector.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/Injector.cfc` — declares map()
  - low `vendor/wheels/interfaces/di/InjectorInterface.cfc` — declares map()
  - low `vendor/wheels/tests/_assets/plugins/serviceprovider/FakeContainer.cfc` — declares map()
- `this.containerreceived → vendor/wheels/Injector.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/Injector.cfc` — declares containsInstance()
  - low `vendor/wheels/interfaces/di/InjectorInterface.cfc` — declares containsInstance()
- `this[] → vendor/wheels/model/properties.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares setProperties()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares setProperties()
- `this[][] → vendor/wheels/model/properties.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares setProperties()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares setProperties()
- `user → examples/starter-app/app/controllers/Accounts.cfc` — 1 finding(s), 18 candidate(s):
  - low `examples/starter-app/app/controllers/Accounts.cfc` — declares update()
  - low `examples/starter-app/app/controllers/PasswordResets.cfc` — declares update()
  - low `examples/starter-app/app/controllers/admin/Permissions.cfc` — declares update()
  - low `examples/starter-app/app/controllers/admin/Roles.cfc` — declares update()
  - low `examples/starter-app/app/controllers/admin/Settings.cfc` — declares update()
  - low `examples/starter-app/app/controllers/admin/Users.cfc` — declares update()
  - low `examples/tweet/app/controllers/Users.cfc` — declares update()
  - low `vendor/wheels/Policy.cfc` — declares update()
  - … 10 more
- `user → vendor/wheels/model/create.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/create.cfc` — declares save()
  - low `vendor/wheels/interfaces/model/ModelPersistenceInterface.cfc` — declares save()
- `user.author.profile → vendor/wheels/model/errors.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/errors.cfc` — declares addError()
  - low `vendor/wheels/interfaces/model/ModelErrorInterface.cfc` — declares addError()
- `v1 → vendor/wheels/Global.cfc` — 1 finding(s), 13 candidate(s):
  - low `vendor/wheels/Global.cfc` — declares get()
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares get()
  - low `vendor/wheels/mapper/matching.cfc` — declares get()
  - low `vendor/wheels/tests/_assets/packages/FakeHttpClient.cfc` — declares get()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares get()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares get()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares get()
  - low `vendor/wheels/services/packages/HttpClient.cfc` — declares get()
  - … 5 more
- `v1 → vendor/wheels/mapper/resources.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/mapper/resources.cfc` — declares resources()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares resources()
- `v2 → vendor/wheels/Global.cfc` — 1 finding(s), 13 candidate(s):
  - low `vendor/wheels/Global.cfc` — declares get()
  - low `vendor/wheels/interfaces/StorageDiskInterface.cfc` — declares get()
  - low `vendor/wheels/mapper/matching.cfc` — declares get()
  - low `vendor/wheels/tests/_assets/packages/FakeHttpClient.cfc` — declares get()
  - low `vendor/wheels/wheelstest/TestClient.cfc` — declares get()
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc` — declares get()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares get()
  - low `vendor/wheels/services/packages/HttpClient.cfc` — declares get()
  - … 5 more
- `variables.$classloader → cli/lucli/services/deploy/lib/SshClient.cfc` — 1 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares close()
  - low `cli/lucli/services/deploy/lib/SshPool.cfc` — declares close()
- `variables.$context → cli/lucli/services/deploy/lib/SshClient.cfc` — 1 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares close()
  - low `cli/lucli/services/deploy/lib/SshPool.cfc` — declares close()
- `variables.$playwright → cli/lucli/services/deploy/lib/SshClient.cfc` — 1 finding(s), 2 candidate(s):
  - low `cli/lucli/services/deploy/lib/SshClient.cfc` — declares close()
  - low `cli/lucli/services/deploy/lib/SshPool.cfc` — declares close()
- `variables.page → vendor/wheels/model/read.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/read.cfc` — declares reload()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares reload()
  - low `cli/lucli/Module.cfc` — declares reload()
- `variables.simpleservice → vendor/wheels/tests/_assets/di/SimpleService.cfc` — 1 finding(s), 2 candidate(s):
  - medium `vendor/wheels/tests/_assets/di/SimpleService.cfc` — declares greet(); named like the receiver 'simpleService'
  - low `vendor/wheels/tests/_assets/plugins/serviceprovider/TestServiceProvider/PluginGreetingService.cfc` — declares greet()
- `yparams[] → vendor/wheels/auth/AuthStrategy.cfc` — 1 finding(s), 14 candidate(s):
  - low `vendor/wheels/auth/AuthStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/JwtStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/SessionStrategy.cfc` — declares getName()
  - low `vendor/wheels/auth/TokenStrategy.cfc` — declares getName()
  - low `vendor/wheels/engineAdapters/Base.cfc` — declares getName()
  - low `tools/lucee-extensions/sqlite/src/SQLite.cfc` — declares getName()
  - low `vendor/wheels/tests/_assets/auth/AlwaysFailStrategy.cfc` — declares getName()
  - low `vendor/wheels/tests/_assets/auth/AlwaysPassStrategy.cfc` — declares getName()
  - … 6 more

</details>

## Return types — a call chained on a method that declares no component

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 40 | `method '$engineAdapter' has no component return type → vendor/wheels/engineAdapters/Base.cfc` | 2 | low | declares prepareDIComplete() (2 candidates) | `vendor/wheels/Controller.cfc:433` method '$engineAdapter' has no component return type (chain to 'prepareDIComplete') |
| 13 | `method 'error' in DetailOutputService@wheels-cli has no component return type → cli/src/models/DetailOutputService.cfc` | 1 | medium | declares output() | `cli/src/commands/wheels/destroy.cfc:29` method 'error' in DetailOutputService@wheels-cli has no component return type (chain to 'output') |
| 11 | `method '$locator' has no component return type` | 0 | none |  | `vendor/wheels/wheelstest/BrowserClient.cfc:149` method '$locator' has no component return type (chain to 'pressSequentially') |
| 7 | `method '$locator' has no component return type → vendor/wheels/wheelstest/BrowserClient.cfc` | 1 | medium | declares click() | `vendor/wheels/wheelstest/BrowserClient.cfc:109` method '$locator' has no component return type (chain to 'click') |
| 7 | `method 'accessory' in Config has no component return type → cli/lucli/services/deploy/config/Accessory.cfc` | 2 | high | declares port(), volumes(), containerName(), labelService(); named like the receiver 'accessory' | `cli/lucli/tests/specs/deploy/config/AccessorySpec.cfc:49` method 'accessory' in Config has no component return type (chain to 'port') |
| 6 | `method 'getService' has no component return type → cli/lucli/services/Helpers.cfc` | 2 | low | declares generateMigrationTimestamp() (2 candidates) | `cli/lucli/Module.cfc:3664` method 'getService' has no component return type (chain to 'generateMigrationTimestamp') |
| 5 | `method '$jobBridge' has no component return type → vendor/wheels/Job.cfc` | 1 | high | declares $instantiateJobClass(), $restoreTenantContext(), $clearTenantContext() | `vendor/wheels/JobWorker.cfc:454` method '$jobBridge' has no component return type (chain to '$instantiateJobClass') |
| 5 | `method 'model' in wheels.Global has no component return type → vendor/wheels/model/read.cfc` | 4 | low | declares findAll() (4 candidates) | `vendor/wheels/tests/specs/migrator/migrationSpec.cfc:1012` method 'model' in wheels.Global has no component return type (chain to 'findAll') |
| 4 | `method 'getUtility' has no component return type → vendor/wheels/wheelstest/system/util/Util.cfc` | 2 | medium | declares getAnnotatedMethods() | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1171` method 'getUtility' has no component return type (chain to 'getAnnotatedMethods') |
| 3 | `method '$engineCapabilities' in wheels.wheelstest.BrowserLauncher has no component return type → vendor/wheels/wheelstest/EngineCapabilities.cfc` | 1 | medium | declares hasJvmClassLoading() | `vendor/wheels/tests/specs/wheelstest/BrowserIntegrationSpec.cfc:10` method '$engineCapabilities' in wheels.wheelstest.BrowserLauncher has no component return type (chain to 'hasJvmClassLoading') |
| 3 | `method '$locator' has no component return type → vendor/wheels/model/calculations.cfc` | 4 | low | declares count() (4 candidates) | `vendor/wheels/wheelstest/BrowserClient.cfc:642` method '$locator' has no component return type (chain to 'count') |
| 3 | `method '$mapper' has no component return type → vendor/wheels/mapper/mapping.cfc` | 1 | medium | declares $draw() | `vendor/wheels/tests/specs/mapperSpec.cfc:61` method '$mapper' has no component return type (chain to '$draw') |
| 3 | `method 'getJavaSystem' has no component return type → vendor/wheels/wheelstest/system/BaseSpec.cfc` | 2 | high | declares getProperty(), getEnv() | `vendor/wheels/wheelstest/system/util/Env.cfc:15` method 'getJavaSystem' has no component return type (chain to 'getProperty') |
| 2 | `method '$locator' has no component return type → vendor/wheels/model/query/QueryBuilder.cfc` | 1 | medium | declares first() | `vendor/wheels/wheelstest/BrowserClient.cfc:295` method '$locator' has no component return type (chain to 'first') |
| 2 | `method 'getBundleStats' in wheels.wheelstest.system.TestResult has no component return type` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:476` method 'getBundleStats' in wheels.wheelstest.system.TestResult has no component return type (chain to 'filter') |
| 2 | `method 'policyScope' in Authorization has no component return type → vendor/wheels/model/calculations.cfc` | 4 | low | declares count() (4 candidates) | `vendor/wheels/tests/specs/controller/AuthorizationSpec.cfc:58` method 'policyScope' in Authorization has no component return type (chain to 'count') |
| 2 | `method 'policyScope' in Authorization has no component return type → vendor/wheels/model/query/QueryBuilder.cfc` | 1 | medium | declares where() | `vendor/wheels/tests/specs/controller/AuthorizationSpec.cfc:176` method 'policyScope' in Authorization has no component return type (chain to 'where') |
| 1 | `method '$classData' in author has no component return type → vendor/wheels/databaseAdapters/Base.cfc` | 6 | low | declares $supportsAdvisoryLocks() (6 candidates) | `vendor/wheels/tests/specs/model/lockingSpec.cfc:20` method '$classData' in author has no component return type (chain to '$supportsAdvisoryLocks') |
| 1 | `method '$classData' in post has no component return type → vendor/wheels/databaseAdapters/Base.cfc` | 2 | low | declares $isSharedModel() (2 candidates) | `vendor/wheels/tests/specs/model/MultiTenantSpec.cfc:147` method '$classData' in post has no component return type (chain to '$isSharedModel') |
| 1 | `method '$engineCapabilities' has no component return type → vendor/wheels/wheelstest/EngineCapabilities.cfc` | 1 | medium | declares hasJvmClassLoading() | `vendor/wheels/wheelstest/BrowserLauncher.cfc:155` method '$engineCapabilities' has no component return type (chain to 'hasJvmClassLoading') |
| 1 | `method '$getObject' has no component return type → vendor/wheels/model/properties.cfc` | 4 | low | declares key() (3 candidates) | `vendor/wheels/view/miscellaneous.cfc:695` method '$getObject' has no component return type (chain to 'key') |
| 1 | `method 'getActiveModules' has no component return type` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:717` method 'getActiveModules' has no component return type (chain to 'each') |
| 1 | `method 'getCBMockData' has no component return type` | 0 | none |  | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1617` method 'getCBMockData' has no component return type (chain to 'mock') |
| 1 | `method 'getClassLoader' has no component return type → cli/lucli/services/deploy/lib/JarLoader.cfc` | 1 | medium | declares loadClass() | `cli/lucli/services/deploy/lib/JarLoader.cfc:86` method 'getClassLoader' has no component return type (chain to 'loadClass') |
| 1 | `method 'getClassLoader' in wheels.wheelstest.BrowserLauncher has no component return type → cli/lucli/services/deploy/lib/JarLoader.cfc` | 1 | medium | declares loadClass() | `vendor/wheels/tests/specs/wheelstest/BrowserLauncherSpec.cfc:93` method 'getClassLoader' in wheels.wheelstest.BrowserLauncher has no component return type (chain to 'loadClass') |
| 1 | `method 'getJavaSystem' has no component return type → vendor/wheels/wheelstest/system/util/Env.cfc` | 8 | low | declares getEnv() (4 candidates) | `vendor/wheels/wheelstest/system/util/Env.cfc:68` method 'getJavaSystem' has no component return type (chain to 'getEnv') |
| 1 | `method 'getLauncher' in wheels.wheelstest.BrowserClient has no component return type → vendor/wheels/tests/_assets/dispatch/InvokeMethodFixture.cfc` | 2 | low | declares getState() (2 candidates) | `vendor/wheels/tests/specs/wheelstest/BrowserIntegrationSpec.cfc:99` method 'getLauncher' in wheels.wheelstest.BrowserClient has no component return type (chain to 'getState') |
| 1 | `method 'getService' has no component return type → cli/lucli/services/PortProbe.cfc` | 1 | medium | declares portInUse() | `cli/lucli/Module.cfc:1070` method 'getService' has no component return type (chain to 'portInUse') |
| 1 | `method 'getUtility' has no component return type → vendor/wheels/wheelstest/system/util/MixerUtil.cfc` | 9 | medium | declares start(); named like the receiver 'MixerUtil' (8 candidates) | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1576` method 'getUtility' has no component return type (chain to 'start') |
| 1 | `method 'model' has no component return type → vendor/wheels/model/serialize.cfc` | 1 | medium | declares $queryRowToStruct() | `vendor/wheels/model/serialize.cfc:160` method 'model' has no component return type (chain to '$queryRowToStruct') |
| 1 | `method 'model' has no component return type → vendor/wheels/model/sql.cfc` | 1 | medium | declares $expandedAssociations() | `vendor/wheels/model/onmissingmethod.cfc:291` method 'model' has no component return type (chain to '$expandedAssociations') |
| 1 | `method 'model' in Global has no component return type → vendor/wheels/Model.cfc` | 1 | high | declares $classData(); named like the receiver 'model' | `vendor/wheels/tests/specs/model/ExpandedAssociationsJoinMemoSpec.cfc:87` method 'model' in Global has no component return type (chain to '$classData') |

<details><summary>Groups with several candidates</summary>

- `method '$engineAdapter' has no component return type → vendor/wheels/engineAdapters/Base.cfc` — 40 finding(s), 2 candidate(s):
  - low `vendor/wheels/engineAdapters/Base.cfc` — declares prepareDIComplete()
  - low `vendor/wheels/engineAdapters/BoxLang/BoxLangAdapter.cfc` — declares prepareDIComplete()
- `method 'getService' has no component return type → cli/lucli/services/Helpers.cfc` — 6 finding(s), 2 candidate(s):
  - low `cli/lucli/services/Helpers.cfc` — declares generateMigrationTimestamp()
  - low `cli/src/models/helpers.cfc` — declares generateMigrationTimestamp()
- `method 'model' in wheels.Global has no component return type → vendor/wheels/model/read.cfc` — 5 finding(s), 4 candidate(s):
  - low `vendor/wheels/model/read.cfc` — declares findAll()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares findAll()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares findAll()
  - low `vendor/wheels/model/query/ScopeChain.cfc` — declares findAll()
- `method '$locator' has no component return type → vendor/wheels/model/calculations.cfc` — 3 finding(s), 4 candidate(s):
  - low `vendor/wheels/model/calculations.cfc` — declares count()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares count()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares count()
  - low `vendor/wheels/model/query/ScopeChain.cfc` — declares count()
- `method 'policyScope' in Authorization has no component return type → vendor/wheels/model/calculations.cfc` — 2 finding(s), 4 candidate(s):
  - low `vendor/wheels/model/calculations.cfc` — declares count()
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc` — declares count()
  - low `vendor/wheels/model/query/QueryBuilder.cfc` — declares count()
  - low `vendor/wheels/model/query/ScopeChain.cfc` — declares count()
- `method '$classData' in author has no component return type → vendor/wheels/databaseAdapters/Base.cfc` — 1 finding(s), 6 candidate(s):
  - low `vendor/wheels/databaseAdapters/Base.cfc` — declares $supportsAdvisoryLocks()
  - low `vendor/wheels/databaseAdapters/CockroachDB/CockroachDBModel.cfc` — declares $supportsAdvisoryLocks()
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerModel.cfc` — declares $supportsAdvisoryLocks()
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLModel.cfc` — declares $supportsAdvisoryLocks()
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLModel.cfc` — declares $supportsAdvisoryLocks()
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteModel.cfc` — declares $supportsAdvisoryLocks()
- `method '$classData' in post has no component return type → vendor/wheels/databaseAdapters/Base.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/databaseAdapters/Base.cfc` — declares $isSharedModel()
  - low `vendor/wheels/interfaces/database/DatabaseModelAdapterInterface.cfc` — declares $isSharedModel()
- `method '$getObject' has no component return type → vendor/wheels/model/properties.cfc` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc` — declares key()
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc` — declares key()
  - low `vendor/wheels/wheelstest/system/Assertion.cfc` — declares key()
- `method 'getJavaSystem' has no component return type → vendor/wheels/wheelstest/system/util/Env.cfc` — 1 finding(s), 4 candidate(s):
  - low `vendor/wheels/wheelstest/system/util/Env.cfc` — declares getEnv()
  - low `vendor/wheels/wheelstest/system/BaseSpec.cfc` — declares getEnv()
  - low `vendor/wheels/wheelstest/system/TestBox.cfc` — declares getEnv()
  - low `cli/lucli/tests/_modules/BaseModule.cfc` — declares getEnv()
- `method 'getLauncher' in wheels.wheelstest.BrowserClient has no component return type → vendor/wheels/tests/_assets/dispatch/InvokeMethodFixture.cfc` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/tests/_assets/dispatch/InvokeMethodFixture.cfc` — declares getState()
  - low `vendor/wheels/wheelstest/BrowserLauncher.cfc` — declares getState()
- `method 'getUtility' has no component return type → vendor/wheels/wheelstest/system/util/MixerUtil.cfc` — 1 finding(s), 8 candidate(s):
  - medium `vendor/wheels/wheelstest/system/util/MixerUtil.cfc` — declares start(); named like the receiver 'MixerUtil'
  - low `cli/lucli/Module.cfc` — declares start()
  - low `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc` — declares start()
  - low `cli/lucli/services/deploy/cli/DeployAppCli.cfc` — declares start()
  - low `cli/lucli/services/deploy/cli/DeployProxyCli.cfc` — declares start()
  - low `cli/lucli/services/deploy/commands/AccessoryCommands.cfc` — declares start()
  - low `cli/lucli/services/deploy/commands/AppCommands.cfc` — declares start()
  - low `cli/lucli/services/deploy/commands/ProxyCommands.cfc` — declares start()

</details>

## Method definitions — a method not found where it was looked for

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 47 | `model` | 1 | medium | declares model | `tools/vscode-ext/assets/templates/controller.cfc:13` not found in extends chain |
| 29 | `$get` | 1 | medium | declares $get | `vendor/wheels/events/onerror/wheelserror.cfm:23` no qualifier, not in file |
| 12 | `flashinsert` | 2 | low | declares flashInsert (2 candidates) | `examples/starter-app/plugins/FlashMessagesBootstrap/index.cfm:6` no qualifier, not in file |
| 10 | `urlfor` | 6 | low | declares URLFor (6 candidates) | `vendor/wheels/events/onrequestend/debug.cfm:437` no qualifier, not in file |
| 7 | `$appkey` | 1 | medium | declares $appKey; beside the calling file | `vendor/wheels/Mapper.cfc:250` no qualifier, not in file |
| 7 | `linkto` | 4 | low | declares linkTo (4 candidates) | `app/snippets/bootstrap/layout.cfm:50` no qualifier, not in file |
| 5 | `$validationexists` | 1 | medium | declares $validationExists | `vendor/wheels/Model.cfc:250` not found in extends chain |
| 5 | `body` | 0 | none |  | `tools/article-tests/edge-cases.cfm:9` no qualifier, not in file |
| 5 | `csrfmetatags` | 1 | medium | declares csrfMetaTags | `app/snippets/bootstrap/layout.cfm:9` no qualifier, not in file |
| 5 | `flashmessages` | 2 | low | declares flashMessages (2 candidates) | `app/snippets/bootstrap/layout.cfm:55` no qualifier, not in file |
| 5 | `pwd` | 1 | none |  | `cli/src/models/MCPService.cfc:164` method 'pwd' not found in shell |
| 4 | `capitalize` | 7 | low | declares capitalize (7 candidates) | `vendor/wheels/events/onrequestend/debug.cfm:132` no qualifier, not in file |
| 4 | `includecontent` | 2 | low | declares includeContent (2 candidates) | `app/snippets/bootstrap/layout.cfm:57` no qualifier, not in file |
| 4 | `javascriptincludetag` | 1 | medium | declares javaScriptIncludeTag | `app/snippets/bootstrap/layout.cfm:65` no qualifier, not in file |
| 4 | `load` | 1 | medium | declares load | `cli/lucli/tests/specs/deploy/cli/DeployRegistryCliSpec.cfc:51` not found in extends chain |
| 4 | `mapper` | 1 | medium | declares mapper | `cli/lucli/templates/app/config/routes.cfm:7` no qualifier, not in file |
| 4 | `primarykeys` | 2 | low | declares primaryKeys (2 candidates) | `vendor/wheels/Model.cfc:244` not found in extends chain |
| 4 | `req` | 0 | none |  | `tools/article-tests/run.cfm:378` no qualifier, not in file |
| 4 | `tablename` | 2 | low | declares tableName (2 candidates) | `vendor/wheels/Model.cfc:103` not found in extends chain |
| 4 | `validatespresenceof` | 2 | low | declares validatesPresenceOf (2 candidates) | `examples/starter-app/plugins/authenticateThis/authenticateThis.cfc:36` no qualifier, not in file |
| 3 | `isdev` | 1 | medium | declares isDev | `vendor/wheels/tests/specs/buildInfoSpec.cfc:34` not found in extends chain |
| 3 | `issnapshot` | 1 | medium | declares isSnapshot | `vendor/wheels/tests/specs/buildInfoSpec.cfc:67` not found in extends chain |
| 3 | `mergescopemiddleware` | 0 | none |  | `tools/article-tests/run.cfm:332` no qualifier, not in file |
| 3 | `next` | 0 | none |  | `vendor/wheels/tests/_assets/middleware/TestMiddlewareA.cfc:16` no qualifier, not in file |
| 3 | `redirectto` | 2 | low | declares redirectTo (2 candidates) | `tools/vscode-ext/assets/templates/controller.cfc:36` not found in extends chain |
| 2 | `adaptername` | 7 | low | declares adapterName (7 candidates) | `vendor/wheels/tests/specs/migrator/MigratorOuterTransactionSpec.cfc:17` not found in extends chain |
| 2 | `authenticatethis` | 1 | medium | declares authenticateThis | `examples/starter-app/app/models/User.cfc:34` not found in extends chain |
| 2 | `config` | 151 | low | declares config; beside the calling file (151 candidates) | `vendor/wheels/Controller.cfc:61` not found in extends chain |
| 2 | `endformtag` | 2 | low | declares endFormTag (2 candidates) | `tools/vscode-ext/assets/templates/view-edit.cfm:9` no qualifier, not in file |
| 2 | `env` | 4 | low | declares env (4 candidates) | `cli/lucli/templates/app/config/settings.cfm:26` no qualifier, not in file |
| 2 | `errormessagesfor` | 1 | medium | declares errorMessagesFor | `tools/vscode-ext/assets/templates/view-edit.cfm:5` no qualifier, not in file |
| 2 | `filegetattribute` | 0 | none |  | `cli/src/commands/wheels/deploy/hooks.cfc:244` not found in extends chain |
| 2 | `getclassmetadata` | 0 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:457` no qualifier, not in file |
| 2 | `getserverinfojson` | 0 | none |  | `cli/src/commands/wheels/benchmark.cfc:156` method 'getServerInfoJSON' not found in ServerService |
| 2 | `humanize` | 2 | low | declares humanize (2 candidates) | `cli/src/models/AdminIntrospectionService.cfc:333` method 'humanize' not found in helpers@wheels-cli |
| 2 | `includepartial` | 4 | low | declares includePartial (4 candidates) | `tools/vscode-ext/assets/templates/view-edit.cfm:7` no qualifier, not in file |
| 2 | `key` | 4 | low | declares key (3 candidates) | `tools/vscode-ext/assets/templates/controller.cfc:36` not found in extends chain |
| 2 | `renderview` | 2 | low | declares renderView (2 candidates) | `tools/vscode-ext/assets/templates/controller.cfc:39` not found in extends chain |
| 2 | `startformtag` | 2 | low | declares startFormTag (2 candidates) | `tools/vscode-ext/assets/templates/view-edit.cfm:6` no qualifier, not in file |
| 2 | `structvalues` | 0 | none |  | `vendor/wheels/public/mcp/SessionManager.cfc:103` no qualifier, not in file |
| 2 | `submittag` | 2 | low | declares submitTag (2 candidates) | `tools/vscode-ext/assets/templates/view-edit.cfm:8` no qualifier, not in file |
| 2 | `table` | 4 | low | declares table (2 candidates) | `vendor/wheels/Model.cfc:48` not found in extends chain |
| 2 | `validatesformatof` | 2 | low | declares validatesFormatOf (2 candidates) | `examples/starter-app/plugins/authenticateThis/authenticateThis.cfc:39` no qualifier, not in file |
| 1 | `$$findmatchingroutes` | 1 | medium | declares $$findMatchingRoutes | `vendor/wheels/public/views/routetesterprocess.cfm:7` not found in extends chain |
| 1 | `$$pluginonlymethod` | 2 | low | declares $$pluginOnlyMethod (2 candidates) | `vendor/wheels/tests/specs/pluginsSpec.cfc:257` method '$$pluginOnlyMethod' not found in Test |
| 1 | `$builddebugreloadurl` | 1 | medium | declares $buildDebugReloadUrl | `vendor/wheels/events/onrequestend/debug.cfm:20` no qualifier, not in file |
| 1 | `$componentintegrationplan` | 1 | medium | declares $componentIntegrationPlan; beside the calling file | `vendor/wheels/Mapper.cfc:387` no qualifier, not in file |
| 1 | `$createobjectfromroot` | 1 | medium | declares $createObjectFromRoot | `cli/lucli/services/TestRunner.cfc:64` no qualifier, not in file |
| 1 | `$dbinfo` | 1 | medium | declares $dbinfo | `vendor/wheels/rocketunit_tests/env.cfm:67` no qualifier, not in file |
| 1 | `$getdbtype` | 1 | medium | declares $getDBType | `vendor/wheels/Global.cfc:2331` no qualifier, not in file |
| 1 | `$header` | 1 | medium | declares $header | `vendor/wheels/rocketunit_tests/env.cfm:28` no qualifier, not in file |
| 1 | `$helper01` | 2 | low | declares $helper01 (2 candidates) | `vendor/wheels/tests/specs/pluginsSpec.cfc:207` method '$helper01' not found in Test |
| 1 | `$normalizepath` | 3 | low | declares $normalizePath (3 candidates) | `vendor/wheels/events/onerror/cfmlerror.cfm:148` no qualifier, not in file |
| 1 | `$queryrowtostruct` | 1 | medium | declares $queryRowToStruct | `vendor/wheels/Model.cfc:536` not found in extends chain |
| 1 | `$resolveinjectedservices` | 1 | medium | declares $resolveInjectedServices | `vendor/wheels/Controller.cfc:118` not found in extends chain |
| 1 | `$setflashappend` | 1 | medium | declares $setFlashAppend | `vendor/wheels/Controller.cfc:57` not found in extends chain |
| 1 | `$setflashstorage` | 1 | medium | declares $setFlashStorage | `vendor/wheels/Controller.cfc:56` not found in extends chain |
| 1 | `$setproperties` | 1 | medium | declares $setProperties | `vendor/wheels/Model.cfc:539` not found in extends chain |
| 1 | `$updatepersistedproperties` | 1 | medium | declares $updatePersistedProperties | `vendor/wheels/Model.cfc:542` not found in extends chain |
| 1 | `beforevalidation` | 2 | low | declares beforeValidation (2 candidates) | `examples/starter-app/plugins/authenticateThis/authenticateThis.cfc:41` no qualifier, not in file |
| 1 | `buttonto` | 3 | low | declares buttonTo (3 candidates) | `tools/vscode-ext/assets/templates/view-index.cfm:28` no qualifier, not in file |
| 1 | `columnnames` | 3 | low | declares columnNames (3 candidates) | `vendor/wheels/tests/specs/controller/SuperOverrideSpec.cfc:42` method 'columnNames' not found in superOverride |
| 1 | `command` | 1 | none |  | `cli/src/models/TestService.cfc:53` no qualifier, not in file |
| 1 | `controller` | 3 | low | declares controller (3 candidates) | `vendor/wheels/rocketunit_tests/env.cfm:57` no qualifier, not in file |
| 1 | `createapplicationsettings` | 1 | medium | declares createApplicationSettings | `examples/starter-app/app/events/onapplicationstart.cfm:11` no qualifier, not in file |
| 1 | `createslug` | 0 | none |  | `tools/vscode-ext/assets/templates/model.cfc:32` not found in extends chain |
| 1 | `exposemixin` | 1 | medium | declares exposeMixin | `vendor/wheels/wheelstest/system/BaseSpec.cfc:1578` method 'exposeMixin' not found in Module\|Job\|JobWorker\|Mapper\|RateLimiter\|ANTJUnitReporter |
| 1 | `findone` | 4 | low | declares findOne (4 candidates) | `tools/vscode-ext/assets/templates/model.cfc:42` not found in extends chain |
| 1 | `fn` | 0 | none |  | `vendor/wheels/tests/specs/packages/LoadRegistryPackagesSpec.cfc:23` not found in extends chain |
| 1 | `fxreinclude` | 0 | none |  | `vendor/wheels/tests/specs/global/reloadGlobalsSpec.cfc:110` method 'fxReinclude' not found in Global |
| 1 | `getboxruntime` | 0 | none |  | `vendor/wheels/Test.cfc:772` not found in extends chain |
| 1 | `getcurrentdirectory` | 0 | none |  | `cli/src/models/EnvironmentService.cfc:1704` no qualifier, not in file |
| 1 | `getcwd` | 1 | none |  | `cli/src/models/MCPService.cfc:28` method 'getCWD' not found in fileSystem |
| 1 | `getdayofmonth` | 0 | none |  | `vendor/wheels/view/formsdate.cfc:126` method 'getDayOfMonth' not found on builtin native.date |
| 1 | `gethour` | 0 | none |  | `vendor/wheels/view/formsdate.cfc:126` method 'getHour' not found on builtin native.date |
| 1 | `getmethod` | 0 | none |  | `vendor/wheels/auth/PasswordHasher.cfc:253` method 'getMethod' not found on builtin native.class |
| 1 | `getminute` | 0 | none |  | `vendor/wheels/view/formsdate.cfc:126` method 'getMinute' not found on builtin native.date |
| 1 | `getmonthvalue` | 0 | none |  | `vendor/wheels/view/formsdate.cfc:126` method 'getMonthValue' not found on builtin native.date |
| 1 | `getsecond` | 0 | none |  | `vendor/wheels/view/formsdate.cfc:126` method 'getSecond' not found on builtin native.date |
| 1 | `gettablenameprefix` | 1 | medium | declares getTableNamePrefix | `vendor/wheels/Model.cfc:103` not found in extends chain |
| 1 | `haschanged` | 1 | medium | declares hasChanged | `vendor/wheels/tests/specs/model/callbacksSpec.cfc:554` not found in extends chain |
| 1 | `isauthenticated` | 1 | medium | declares isAuthenticated | `examples/starter-app/app/events/onrequeststart.cfm:7` no qualifier, not in file |
| 1 | `memoprobeinjected` | 0 | none |  | `vendor/wheels/tests/specs/global/promoteIncludedGlobalsMemoSpec.cfc:172` method 'memoProbeInjected' not found in wheels.tests._assets.global.PromoteMemoFixture |
| 1 | `primarykey` | 3 | low | declares primaryKey; beside the calling file (3 candidates) | `vendor/wheels/model/read.cfc:530` no qualifier, not in file |
| 1 | `println` | 2 | medium | declares println | `vendor/wheels/wheelstest/BrowserClient.cfc:417` no qualifier, not in file |
| 1 | `save` | 2 | low | declares save (2 candidates) | `tools/vscode-ext/assets/templates/controller.cfc:34` not found in extends chain |
| 1 | `setprimarykey` | 2 | low | declares setPrimaryKey (2 candidates) | `vendor/wheels/Model.cfc:222` not found in extends chain |
| 1 | `settablenameprefix` | 1 | medium | declares setTableNamePrefix | `vendor/wheels/Model.cfc:47` not found in extends chain |
| 1 | `setup` | 15 | low | declares setup (15 candidates) | `vendor/wheels/Test.cfc:323` not found in extends chain |
| 1 | `teardown` | 12 | low | declares teardown (12 candidates) | `vendor/wheels/Test.cfc:379` not found in extends chain |
| 1 | `timestampvalue` | 0 | none |  | `vendor/wheels/view/formsdate.cfc:128` method 'timestampValue' not found on builtin native.date |
| 1 | `toargv` | 1 | medium | declares toArgv | `cli/lucli/tests/specs/services/ArgSpecSpec.cfc:195` not found in extends chain |
| 1 | `toconsole` | 1 | none |  | `cli/src/commands/wheels/docker/push.cfc:393` method 'toConsole' not found in DetailOutputService |
| 1 | `validatesconfirmationof` | 2 | low | declares validatesConfirmationOf (2 candidates) | `examples/starter-app/plugins/authenticateThis/authenticateThis.cfc:40` no qualifier, not in file |
| 1 | `validateslengthof` | 2 | low | declares validatesLengthOf (2 candidates) | `vendor/wheels/Model.cfc:272` not found in extends chain |
| 1 | `validatesnumericalityof` | 2 | low | declares validatesNumericalityOf (2 candidates) | `vendor/wheels/Model.cfc:284` not found in extends chain |
| 1 | `variables[]` | 0 | none |  | `vendor/wheels/Test.cfc:756` not found in extends chain |

<details><summary>Groups with several candidates</summary>

- `flashinsert` — 12 finding(s), 2 candidate(s):
  - low `vendor/wheels/controller/flash.cfc:69` — declares flashInsert
  - low `vendor/wheels/interfaces/controller/ControllerFlashInterface.cfc:26` — declares flashInsert
- `urlfor` — 10 finding(s), 6 candidate(s):
  - low `vendor/wheels/Global.cfc:1786` — declares URLFor
  - low `vendor/wheels/interfaces/view/ViewLinkInterface.cfc:149` — declares urlFor
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc:7` — declares URLFor
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner02/Runner02.cfc:7` — declares URLFor
  - low `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc:7` — declares URLFor
  - low `vendor/wheels/tests/_assets/plugins/runner/runner02/Runner02.cfc:7` — declares URLFor
- `linkto` — 7 finding(s), 4 candidate(s):
  - low `vendor/wheels/view/links.cfc:24` — declares linkTo
  - low `examples/starter-app/plugins/jsconfirm/JSConfirm.cfc:11` — declares linkTo
  - low `vendor/wheels/interfaces/view/ViewLinkInterface.cfc:29` — declares linkTo
  - low `vendor/wheels/tests/_assets/controllers/SuperOverride.cfc:9` — declares linkTo
- `flashmessages` — 5 finding(s), 2 candidate(s):
  - low `vendor/wheels/view/miscellaneous.cfc:116` — declares flashMessages
  - low `examples/starter-app/plugins/FlashMessagesBootstrap/FlashMessagesBootstrap.cfc:8` — declares flashMessages
- `capitalize` — 4 finding(s), 7 candidate(s):
  - low `vendor/wheels/Global.cfc:2105` — declares capitalize
  - low `cli/lucli/Module.cfc:7854` — declares capitalize
  - low `cli/lucli/services/Helpers.cfc:13` — declares capitalize
  - low `cli/src/models/AdminIntrospectionService.cfc:310` — declares capitalize
  - low `cli/src/models/AdminViewService.cfc:402` — declares capitalize
  - low `cli/src/models/CodeGenerationService.cfc:16` — declares capitalize
  - low `cli/src/models/helpers.cfc:88` — declares capitalize
- `includecontent` — 4 finding(s), 2 candidate(s):
  - low `vendor/wheels/view/miscellaneous.cfc:325` — declares includeContent
  - low `vendor/wheels/interfaces/view/ViewContentInterface.cfc:31` — declares includeContent
- `primarykeys` — 4 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc:189` — declares primaryKeys
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc:79` — declares primaryKeys
- `tablename` — 4 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc:207` — declares tableName
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc:22` — declares tableName
- `validatespresenceof` — 4 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/validations.cfc:276` — declares validatesPresenceOf
  - low `vendor/wheels/interfaces/model/ModelValidationInterface.cfc:59` — declares validatesPresenceOf
- `redirectto` — 3 finding(s), 2 candidate(s):
  - low `vendor/wheels/controller/redirection.cfc:27` — declares redirectTo
  - low `vendor/wheels/interfaces/controller/ControllerRenderingInterface.cfc:115` — declares redirectTo
- `adaptername` — 2 finding(s), 7 candidate(s):
  - low `vendor/wheels/databaseAdapters/CockroachDB/CockroachDBMigrator.cfc:21` — declares adapterName
  - low `vendor/wheels/databaseAdapters/H2/H2Migrator.cfc:21` — declares adapterName
  - low `vendor/wheels/databaseAdapters/MicrosoftSQLServer/MicrosoftSQLServerMigrator.cfc:19` — declares adapterName
  - low `vendor/wheels/databaseAdapters/MySQL/MySQLMigrator.cfc:23` — declares adapterName
  - low `vendor/wheels/databaseAdapters/Oracle/OracleMigrator.cfc:27` — declares adapterName
  - low `vendor/wheels/databaseAdapters/PostgreSQL/PostgreSQLMigrator.cfc:19` — declares adapterName
  - low `vendor/wheels/databaseAdapters/SQLite/SQLiteMigrator.cfc:24` — declares adapterName
- `config` — 2 finding(s), 151 candidate(s):
  - low `vendor/wheels/Job.cfc:52` — declares config; beside the calling file
  - low `vendor/wheels/controllers/Tests.cfc:6` — declares config
  - low `vendor/wheels/public/browser-fixtures/controllers/BrowserTestHome.cfc:14` — declares config
  - low `vendor/wheels/public/browser-fixtures/controllers/BrowserTestLogin.cfc:13` — declares config
  - low `vendor/wheels/rocketunit_tests/_assets/controllers/ApiTest.cfc:3` — declares config
  - low `vendor/wheels/rocketunit_tests/_assets/controllers/CsrfProtectedExcept.cfc:3` — declares config
  - low `vendor/wheels/rocketunit_tests/_assets/controllers/CsrfProtectedOnly.cfc:3` — declares config
  - low `vendor/wheels/rocketunit_tests/_assets/controllers/CsrfProtectedWithException.cfc:3` — declares config
  - … 143 more
- `endformtag` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/view/forms.cfc:12` — declares endFormTag
  - low `vendor/wheels/interfaces/view/ViewFormInterface.cfc:60` — declares endFormTag
- `env` — 2 finding(s), 4 candidate(s):
  - low `cli/lucli/services/deploy/config/Accessory.cfc:62` — declares env
  - low `cli/lucli/services/deploy/config/Config.cfc:38` — declares env
  - low `cli/lucli/services/deploy/config/Role.cfc:25` — declares env
  - low `vendor/wheels/Global.cfc:672` — declares env
- `humanize` — 2 finding(s), 2 candidate(s):
  - low `cli/src/commands/wheels/generate/helper.cfc:407` — declares humanize
  - low `vendor/wheels/Global.cfc:2123` — declares humanize
- `includepartial` — 2 finding(s), 4 candidate(s):
  - low `vendor/wheels/view/miscellaneous.cfc:291` — declares includePartial
  - low `vendor/wheels/interfaces/view/ViewContentInterface.cfc:43` — declares includePartial
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc:42` — declares includePartial
  - low `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc:42` — declares includePartial
- `key` — 2 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/properties.cfc:206` — declares key
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc:66` — declares key
  - low `vendor/wheels/wheelstest/system/Assertion.cfc:416` — declares key
- `renderview` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/controller/rendering.cfc:18` — declares renderView
  - low `vendor/wheels/interfaces/controller/ControllerRenderingInterface.cfc:24` — declares renderView
- `startformtag` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/view/forms.cfc:48` — declares startFormTag
  - low `vendor/wheels/interfaces/view/ViewFormInterface.cfc:35` — declares startFormTag
- `submittag` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/view/forms.cfc:210` — declares submitTag
  - low `vendor/wheels/interfaces/view/ViewFormInterface.cfc:125` — declares submitTag
- `table` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc:57` — declares table
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc:36` — declares table
- `validatesformatof` — 2 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/validations.cfc:129` — declares validatesFormatOf
  - low `vendor/wheels/interfaces/model/ModelValidationInterface.cfc:123` — declares validatesFormatOf
- `$$pluginonlymethod` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc:17` — declares $$pluginOnlyMethod
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc:17` — declares $$pluginOnlyMethod
- `$helper01` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/tests/_assets/plugins/runner/runner01/Runner01.cfc:30` — declares $helper01
  - low `vendor/wheels/rocketunit_tests/_assets/plugins/runner/runner01/Runner01.cfc:30` — declares $helper01
- `$normalizepath` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/Global.cfc:4229` — declares $normalizePath
  - low `cli/lucli/Module.cfc:63` — declares $normalizePath
  - low `cli/lucli/services/Doctor.cfc:606` — declares $normalizePath
- `beforevalidation` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/callbacks.cfc:178` — declares beforeValidation
  - low `vendor/wheels/interfaces/model/ModelCallbackInterface.cfc:19` — declares beforeValidation
- `buttonto` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/view/links.cfc:104` — declares buttonTo
  - low `examples/starter-app/plugins/jsconfirm/JSConfirm.cfc:21` — declares buttonTo
  - low `vendor/wheels/interfaces/view/ViewLinkInterface.cfc:63` — declares buttonTo
- `columnnames` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc:157` — declares columnNames
  - low `vendor/wheels/tests/_assets/models/SuperOverride.cfc:12` — declares columnNames
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc:71` — declares columnNames
- `controller` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/Global.cfc:1410` — declares controller
  - low `vendor/wheels/mapper/scoping.cfc:158` — declares controller
  - low `vendor/wheels/interfaces/routing/RouteMapperInterface.cfc:225` — declares controller
- `findone` — 1 finding(s), 4 candidate(s):
  - low `vendor/wheels/model/read.cfc:475` — declares findOne
  - low `vendor/wheels/interfaces/model/ModelFinderInterface.cfc:80` — declares findOne
  - low `vendor/wheels/model/query/QueryBuilder.cfc:355` — declares findOne
  - low `vendor/wheels/model/query/ScopeChain.cfc:104` — declares findOne
- `primarykey` — 1 finding(s), 3 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc:172` — declares primaryKey; beside the calling file
  - low `vendor/wheels/migrator/TableDefinition.cfc:44` — declares primaryKey
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc:86` — declares primaryKey
- `save` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/create.cfc:83` — declares save
  - low `vendor/wheels/interfaces/model/ModelPersistenceInterface.cfc:53` — declares save
- `setprimarykey` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/miscellaneous.cfc:83` — declares setPrimaryKey
  - low `vendor/wheels/interfaces/model/ModelPropertyInterface.cfc:29` — declares setPrimaryKey
- `setup` — 1 finding(s), 15 candidate(s):
  - low `cli/src/models/EnvironmentService.cfc:10` — declares setup
  - low `examples/starter-app/tests/RocketUnit/Test.cfc:23` — declares setup
  - low `cli/lucli/services/deploy/cli/DeployMainCli.cfc:274` — declares setup
  - low `cli/lucli/services/deploy/cli/DeployRegistryCli.cfc:24` — declares setup
  - low `examples/starter-app/plugins/authenticateThis/tests/AuthenticateTest.cfc:2` — declares setup
  - low `examples/starter-app/tests/RocketUnit/functions/Auth.cfc:4` — declares setup
  - low `examples/starter-app/tests/RocketUnit/functions/Permissions.cfc:4` — declares setup
  - low `examples/starter-app/tests/RocketUnit/functions/Utils.cfc:3` — declares setup
  - … 7 more
- `teardown` — 1 finding(s), 12 candidate(s):
  - low `examples/starter-app/tests/RocketUnit/Test.cfc:45` — declares teardown
  - low `examples/starter-app/plugins/authenticateThis/tests/AuthenticateTest.cfc:7` — declares teardown
  - low `examples/starter-app/tests/RocketUnit/functions/Auth.cfc:8` — declares teardown
  - low `examples/starter-app/tests/RocketUnit/functions/Permissions.cfc:7` — declares teardown
  - low `examples/starter-app/tests/RocketUnit/functions/Utils.cfc:6` — declares teardown
  - low `examples/starter-app/tests/RocketUnit/requests/Accounts.cfc:7` — declares teardown
  - low `examples/starter-app/tests/RocketUnit/requests/Main.cfc:7` — declares teardown
  - low `examples/starter-app/tests/RocketUnit/requests/Sessions.cfc:7` — declares teardown
  - … 4 more
- `validatesconfirmationof` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/validations.cfc:74` — declares validatesConfirmationOf
  - low `vendor/wheels/interfaces/model/ModelValidationInterface.cfc:221` — declares validatesConfirmationOf
- `validateslengthof` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/validations.cfc:204` — declares validatesLengthOf
  - low `vendor/wheels/interfaces/model/ModelValidationInterface.cfc:98` — declares validatesLengthOf
- `validatesnumericalityof` — 1 finding(s), 2 candidate(s):
  - low `vendor/wheels/model/validations.cfc:244` — declares validatesNumericalityOf
  - low `vendor/wheels/interfaces/model/ModelValidationInterface.cfc:152` — declares validatesNumericalityOf

</details>

## Object references — a component path that names no file

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 16 | `component 'modules.wheels.services.deploy.config.ConfigLoader' does not exist (calling 'load')` | 1 | medium | file named ConfigLoader.cfc | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:54` component 'modules.wheels.services.deploy.config.ConfigLoader' does not exist (calling 'load') |
| 14 | `chained on 'mapper', which is not found (calling 'get')` | 13 | none |  | `cli/lucli/templates/app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'get') |
| 13 | `base component does not resolve; 2 inherited calls not checked` | ? | none |  | `examples/starter-app/tests/RocketUnit/requests/Main.cfc:1` base component does not resolve; 2 inherited calls not checked |
| 8 | `base component does not resolve; 1 inherited call not checked` | ? | none |  | `cli/lucli/services/deploy/secrets/AwsSecretsAdapter.cfc:14` base component does not resolve; 1 inherited call not checked |
| 8 | `chained on 'mapper', which is not found (calling 'end')` | 4 | none |  | `cli/lucli/templates/app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'end') |
| 8 | `chained on 'mapper', which is not found (calling 'resources')` | 2 | none |  | `examples/starter-app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'resources') |
| 7 | `chained on 'mapper', which is not found (calling 'post')` | 3 | none |  | `examples/starter-app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'post') |
| 7 | `component 'modules.wheels.services.deploy.lib.JarLoader' does not exist (calling 'withIsolatedTCCL')` | 1 | medium | file named JarLoader.cfc | `cli/lucli/services/deploy/lib/Mustache.cfc:67` component 'modules.wheels.services.deploy.lib.JarLoader' does not exist (calling 'withIsolatedTCCL') |
| 5 | `chained on 'mapper', which is not found (calling 'root')` | 4 | none |  | `cli/lucli/templates/app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'root') |
| 4 | `base component does not resolve; 4 inherited calls not checked` | ? | none |  | `cli/src/commands/wheels/secret.cfc:13` base component does not resolve; 4 inherited calls not checked |
| 4 | `chained on 'mapper', which is not found (calling 'put')` | 6 | none |  | `examples/starter-app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'put') |
| 4 | `component 'modules.wheels.services.deploy.commands.AuditorCommands' does not exist (calling 'record')` | 1 | medium | file named AuditorCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:128` component 'modules.wheels.services.deploy.commands.AuditorCommands' does not exist (calling 'record') |
| 4 | `component 'modules.wheels.services.deploy.commands.LockCommands' does not exist (calling 'release')` | 3 | medium | file named LockCommands.cfc | `cli/lucli/services/deploy/cli/DeployLockCli.cfc:48` component 'modules.wheels.services.deploy.commands.LockCommands' does not exist (calling 'release') |
| 4 | `component 'modules.wheels.services.deploy.lib.JarLoader' does not exist (calling 'loadClass')` | 1 | medium | file named JarLoader.cfc | `cli/lucli/services/deploy/lib/Mustache.cfc:22` component 'modules.wheels.services.deploy.lib.JarLoader' does not exist (calling 'loadClass') |
| 4 | `component 'modules.wheels.services.deploy.lib.Mustache' does not exist (calling 'render')` | 1 | medium | file named Mustache.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:505` component 'modules.wheels.services.deploy.lib.Mustache' does not exist (calling 'render') |
| 3 | `chained on 'mapper', which is not found (calling 'wildcard')` | 2 | none |  | `cli/lucli/templates/app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'wildcard') |
| 3 | `component 'modules.wheels.services.deploy.config.ConfigLoader' does not exist (calling 'secretResolver')` | 1 | medium | file named ConfigLoader.cfc | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:127` component 'modules.wheels.services.deploy.config.ConfigLoader' does not exist (calling 'secretResolver') |
| 2 | `base component does not resolve; 15 inherited calls not checked` | ? | none |  | `examples/starter-app/tests/RocketUnit/functions/Auth.cfc:1` base component does not resolve; 15 inherited calls not checked |
| 2 | `chained on 'mapper', which is not found (calling 'scope')` | 6 | none |  | `examples/starter-app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'scope') |
| 2 | `chained on 'model', which is not found (calling 'findByKey')` | 3 | none |  | `tools/vscode-ext/assets/templates/controller.cfc:51` chained on 'model', which is not found (calling 'findByKey') |
| 2 | `chained on 'model', which is not found (calling 'new')` | 14 | none |  | `tools/vscode-ext/assets/templates/controller.cfc:27` chained on 'model', which is not found (calling 'new') |
| 2 | `component 'BrowserService@wheels-cli' does not exist (calling 'getManifest')` | 2 | none |  | `cli/src/commands/wheels/browser/install.cfc:28` component 'BrowserService@wheels-cli' does not exist (calling 'getManifest') |
| 2 | `component 'BrowserService@wheels-cli' does not exist (calling 'resolveInstallDir')` | 2 | none |  | `cli/src/commands/wheels/browser/install.cfc:34` component 'BrowserService@wheels-cli' does not exist (calling 'resolveInstallDir') |
| 2 | `component 'BrowserService@wheels-cli' does not exist (calling 'sha256')` | 1 | none |  | `cli/src/commands/wheels/browser/install.cfc:48` component 'BrowserService@wheels-cli' does not exist (calling 'sha256') |
| 2 | `component 'lucee.admin' does not exist (calling 'updateDatasource')` | 0 | low | file named admin.cfc (4 candidates) | `tools/ci/setup-datasources.cfm:7` component 'lucee.admin' does not exist (calling 'updateDatasource') |
| 2 | `component 'modules.wheels.services.deploy.cli.DeployServerCli' does not exist (calling 'bootstrap')` | 1 | medium | file named DeployServerCli.cfc | `cli/lucli/Module.cfc:2497` component 'modules.wheels.services.deploy.cli.DeployServerCli' does not exist (calling 'bootstrap') |
| 2 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'container_name')` | 1 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:165` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'container_name') |
| 2 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'ensure_env_file')` | 2 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployAppCli.cfc:148` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'ensure_env_file') |
| 2 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'env_file_content')` | 1 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployAppCli.cfc:139` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'env_file_content') |
| 2 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'env_file_path')` | 2 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployAppCli.cfc:151` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'env_file_path') |
| 2 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'relock_env_file')` | 2 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployAppCli.cfc:149` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'relock_env_file') |
| 2 | `component 'modules.wheels.services.deploy.commands.BuilderCommands' does not exist (calling 'pull')` | 2 | medium | file named BuilderCommands.cfc | `cli/lucli/services/deploy/cli/DeployBuildCli.cfc:55` component 'modules.wheels.services.deploy.commands.BuilderCommands' does not exist (calling 'pull') |
| 2 | `component 'modules.wheels.services.deploy.commands.DockerCommands' does not exist (calling 'ensure_network')` | 1 | medium | file named DockerCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:133` component 'modules.wheels.services.deploy.commands.DockerCommands' does not exist (calling 'ensure_network') |
| 2 | `component 'modules.wheels.services.deploy.commands.LockCommands' does not exist (calling 'acquire')` | 2 | medium | file named LockCommands.cfc | `cli/lucli/services/deploy/cli/DeployLockCli.cfc:25` component 'modules.wheels.services.deploy.commands.LockCommands' does not exist (calling 'acquire') |
| 2 | `component 'modules.wheels.services.deploy.commands.ProxyCommands' does not exist (calling 'deploy')` | 3 | medium | file named ProxyCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:165` component 'modules.wheels.services.deploy.commands.ProxyCommands' does not exist (calling 'deploy') |
| 2 | `component 'modules.wheels.services.deploy.commands.RegistryCommands' does not exist (calling 'logout')` | 6 | medium | file named RegistryCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:431` component 'modules.wheels.services.deploy.commands.RegistryCommands' does not exist (calling 'logout') |
| 2 | `component 'modules.wheels.services.deploy.lib.Yaml' does not exist (calling 'parse')` | 5 | medium | file named Yaml.cfc | `cli/lucli/services/deploy/config/ConfigLoader.cfc:39` component 'modules.wheels.services.deploy.lib.Yaml' does not exist (calling 'parse') |
| 1 | `base component does not resolve; 10 inherited calls not checked` | ? | none |  | `examples/starter-app/tests/RocketUnit/requests/Sessions.cfc:1` base component does not resolve; 10 inherited calls not checked |
| 1 | `base component does not resolve; 11 inherited calls not checked` | ? | none |  | `cli/src/commands/wheels/config/check.cfc:12` base component does not resolve; 11 inherited calls not checked |
| 1 | `base component does not resolve; 12 inherited calls not checked` | ? | none |  | `examples/starter-app/tests/RocketUnit/requests/Accounts.cfc:1` base component does not resolve; 12 inherited calls not checked |
| 1 | `base component does not resolve; 16 inherited calls not checked` | ? | none |  | `examples/starter-app/tests/RocketUnit/functions/models/Permission.cfc:1` base component does not resolve; 16 inherited calls not checked |
| 1 | `base component does not resolve; 29 inherited calls not checked` | ? | none |  | `examples/starter-app/tests/RocketUnit/functions/models/User.cfc:1` base component does not resolve; 29 inherited calls not checked |
| 1 | `base component does not resolve; 5 inherited calls not checked` | ? | none |  | `examples/starter-app/tests/RocketUnit/functions/Utils.cfc:1` base component does not resolve; 5 inherited calls not checked |
| 1 | `base component does not resolve; 6 inherited calls not checked` | ? | none |  | `cli/src/commands/wheels/env/set.cfc:11` base component does not resolve; 6 inherited calls not checked |
| 1 | `base component does not resolve; 61 inherited calls not checked` | ? | none |  | `cli/tests/specs/commands/ConfigCommandsTest.cfc:4` base component does not resolve; 61 inherited calls not checked |
| 1 | `base component does not resolve; 68 inherited calls not checked` | ? | none |  | `examples/starter-app/tests/RocketUnit/functions/Permissions.cfc:1` base component does not resolve; 68 inherited calls not checked |
| 1 | `base component does not resolve; 7 inherited calls not checked` | ? | none |  | `cli/src/commands/wheels/env/merge.cfc:11` base component does not resolve; 7 inherited calls not checked |
| 1 | `base component does not resolve; 764 inherited calls not checked` | ? | none |  | `cli/lucli/Module.cfc:16` base component does not resolve; 764 inherited calls not checked |
| 1 | `base component does not resolve; 8 inherited calls not checked` | ? | none |  | `cli/src/commands/wheels/config/diff.cfc:13` base component does not resolve; 8 inherited calls not checked |
| 1 | `base component does not resolve; 9 inherited calls not checked` | ? | none |  | `cli/src/commands/wheels/config/dump.cfc:13` base component does not resolve; 9 inherited calls not checked |
| 1 | `chained on 'getBoxRuntime', which is not found (calling 'executeStatement')` | 0 | none |  | `vendor/wheels/Test.cfc:772` chained on 'getBoxRuntime', which is not found (calling 'executeStatement') |
| 1 | `chained on 'mapper', which is not found (calling 'delete')` | 19 | none |  | `examples/starter-app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'delete') |
| 1 | `chained on 'mapper', which is not found (calling 'member')` | 2 | none |  | `examples/starter-app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'member') |
| 1 | `chained on 'mapper', which is not found (calling 'resource')` | 2 | none |  | `examples/starter-app/config/routes.cfm:7` chained on 'mapper', which is not found (calling 'resource') |
| 1 | `chained on 'model', which is not found (calling 'findAll')` | 4 | none |  | `tools/vscode-ext/assets/templates/controller.cfc:13` chained on 'model', which is not found (calling 'findAll') |
| 1 | `component '#arguments.bundlePath#' does not exist (calling 'getDebugBuffer')` | 2 | none |  | `vendor/wheels/wheelstest/system/TestBox.cfc:788` component '#arguments.bundlePath#' does not exist (calling 'getDebugBuffer') |
| 1 | `component 'BrowserService@wheels-cli' does not exist (calling 'downloadJar')` | 1 | none |  | `cli/src/commands/wheels/browser/install.cfc:58` component 'BrowserService@wheels-cli' does not exist (calling 'downloadJar') |
| 1 | `component 'BrowserService@wheels-cli' does not exist (calling 'verifyInstall')` | 1 | none |  | `cli/src/commands/wheels/browser/test.cfc:31` component 'BrowserService@wheels-cli' does not exist (calling 'verifyInstall') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployArgsParser' does not exist (calling 'parse')` | 5 | medium | file named DeployArgsParser.cfc | `cli/lucli/Module.cfc:2610` component 'modules.wheels.services.deploy.cli.DeployArgsParser' does not exist (calling 'parse') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'audit')` | 1 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2358` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'audit') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'config')` | 151 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2350` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'config') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'deploy')` | 3 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2340` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'deploy') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'details')` | 8 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2364` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'details') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'docs')` | 1 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2362` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'docs') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'init_stub')` | 1 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2352` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'init_stub') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'redeploy')` | 1 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2342` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'redeploy') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'rollback')` | 1 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2348` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'rollback') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'setup')` | 15 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2354` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'setup') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'version')` | 6 | medium | file named DeployMainCli.cfc | `cli/lucli/Module.cfc:2356` component 'modules.wheels.services.deploy.cli.DeployMainCli' does not exist (calling 'version') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeploySecretsCli' does not exist (calling 'extract')` | 1 | medium | file named DeploySecretsCli.cfc | `cli/lucli/Module.cfc:2569` component 'modules.wheels.services.deploy.cli.DeploySecretsCli' does not exist (calling 'extract') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeploySecretsCli' does not exist (calling 'fetch')` | 8 | medium | file named DeploySecretsCli.cfc | `cli/lucli/Module.cfc:2565` component 'modules.wheels.services.deploy.cli.DeploySecretsCli' does not exist (calling 'fetch') |
| 1 | `component 'modules.wheels.services.deploy.cli.DeploySecretsCli' does not exist (calling 'print')` | 3 | medium | file named DeploySecretsCli.cfc | `cli/lucli/Module.cfc:2572` component 'modules.wheels.services.deploy.cli.DeploySecretsCli' does not exist (calling 'print') |
| 1 | `component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'details')` | 8 | medium | file named AccessoryCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:375` component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'details') |
| 1 | `component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'ensure_env_file')` | 2 | medium | file named AccessoryCommands.cfc | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:81` component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'ensure_env_file') |
| 1 | `component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'env_file_content')` | 1 | medium | file named AccessoryCommands.cfc | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:83` component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'env_file_content') |
| 1 | `component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'env_file_path')` | 2 | medium | file named AccessoryCommands.cfc | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:84` component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'env_file_path') |
| 1 | `component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'relock_env_file')` | 2 | medium | file named AccessoryCommands.cfc | `cli/lucli/services/deploy/cli/DeployAccessoryCli.cfc:82` component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'relock_env_file') |
| 1 | `component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'run')` | 637 | medium | file named AccessoryCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:291` component 'modules.wheels.services.deploy.commands.AccessoryCommands' does not exist (calling 'run') |
| 1 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'containers')` | 4 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:368` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'containers') |
| 1 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'remove_conflicting')` | 1 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:147` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'remove_conflicting') |
| 1 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'run')` | 637 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:159` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'run') |
| 1 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'start')` | 9 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:246` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'start') |
| 1 | `component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'stop_old_versions')` | 1 | medium | file named AppCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:173` component 'modules.wheels.services.deploy.commands.AppCommands' does not exist (calling 'stop_old_versions') |
| 1 | `component 'modules.wheels.services.deploy.commands.HookCommands' does not exist (calling 'forHook')` | 1 | medium | file named HookCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:882` component 'modules.wheels.services.deploy.commands.HookCommands' does not exist (calling 'forHook') |
| 1 | `component 'modules.wheels.services.deploy.commands.LockCommands' does not exist (calling 'status')` | 3 | medium | file named LockCommands.cfc | `cli/lucli/services/deploy/cli/DeployLockCli.cfc:69` component 'modules.wheels.services.deploy.commands.LockCommands' does not exist (calling 'status') |
| 1 | `component 'modules.wheels.services.deploy.commands.ProxyCommands' does not exist (calling 'details')` | 8 | medium | file named ProxyCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:370` component 'modules.wheels.services.deploy.commands.ProxyCommands' does not exist (calling 'details') |
| 1 | `component 'modules.wheels.services.deploy.commands.ProxyCommands' does not exist (calling 'start_or_run')` | 1 | medium | file named ProxyCommands.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:140` component 'modules.wheels.services.deploy.commands.ProxyCommands' does not exist (calling 'start_or_run') |
| 1 | `component 'modules.wheels.services.deploy.commands.RegistryCommands' does not exist (calling 'login')` | 6 | medium | file named RegistryCommands.cfc | `cli/lucli/services/deploy/cli/DeployRegistryCli.cfc:56` component 'modules.wheels.services.deploy.commands.RegistryCommands' does not exist (calling 'login') |
| 1 | `component 'modules.wheels.services.deploy.lib.SecretResolver' does not exist (calling 'all')` | 4 | medium | file named SecretResolver.cfc | `cli/lucli/services/deploy/cli/DeploySecretsCli.cfc:89` component 'modules.wheels.services.deploy.lib.SecretResolver' does not exist (calling 'all') |
| 1 | `component 'modules.wheels.services.deploy.lib.SshPoolFactory' does not exist (calling 'fromConfigPath')` | 1 | medium | file named SshPoolFactory.cfc | `cli/lucli/Module.cfc:2601` component 'modules.wheels.services.deploy.lib.SshPoolFactory' does not exist (calling 'fromConfigPath') |
| 1 | `component 'modules.wheels.services.deploy.lib.Yaml' does not exist (calling 'deepMerge')` | 1 | medium | file named Yaml.cfc | `cli/lucli/services/deploy/config/ConfigLoader.cfc:46` component 'modules.wheels.services.deploy.lib.Yaml' does not exist (calling 'deepMerge') |
| 1 | `component 'modules.wheels.services.deploy.lib.Yaml' does not exist (calling 'dump')` | 1 | medium | file named Yaml.cfc | `cli/lucli/services/deploy/cli/DeployMainCli.cfc:46` component 'modules.wheels.services.deploy.lib.Yaml' does not exist (calling 'dump') |
| 1 | `component 'modules.wheels.services.packages.PackagesMainCli' does not exist (calling 'list')` | 3 | medium | file named PackagesMainCli.cfc | `cli/lucli/Module.cfc:2661` component 'modules.wheels.services.packages.PackagesMainCli' does not exist (calling 'list') |
| 1 | `component 'modules.wheels.services.packages.PackagesMainCli' does not exist (calling 'show')` | 10 | medium | file named PackagesMainCli.cfc | `cli/lucli/Module.cfc:2675` component 'modules.wheels.services.packages.PackagesMainCli' does not exist (calling 'show') |
| 1 | `component 'modules.wheels.vendor.wheels.BuildInfo' does not exist (calling 'version')` | 6 | medium | file named BuildInfo.cfc | `cli/lucli/services/packages/PackagesMainCli.cfc:345` component 'modules.wheels.vendor.wheels.BuildInfo' does not exist (calling 'version') |

<details><summary>Groups with several candidates</summary>

- `component 'lucee.admin' does not exist (calling 'updateDatasource')` — 2 finding(s), 4 candidate(s):
  - low `cli/lucli/services/Admin.cfc` — file named admin.cfc
  - low `cli/src/commands/wheels/generate/admin.cfc` — file named admin.cfc
  - low `vendor/wheels/rocketunit_tests/_assets/controllers/admin/Admin.cfc` — file named admin.cfc
  - low `vendor/wheels/tests/_assets/controllers/admin/Admin.cfc` — file named admin.cfc

</details>
