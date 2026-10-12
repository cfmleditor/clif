# Methods missing from selected contracts — review queue

Generated from the 12,355-entry strict baseline. These 310 occurrences are
**not classified as nonexistent**: receiver selection, inheritance, runtime
registration, engine/version APIs and dependency contracts still need review.
Same-name declarations elsewhere do not prove receiver ownership.
The JSON export retains all occurrences, candidate evidence and original reasons.

| Project | Contract kind | Diagnostic | Entries | Example source |
|---|---|---|---:|---|
| contentbox | selected_component | method 'columns' not found in dbinfo | 13 | [Update.cfc:497](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.0.0-beta/Update.cfc:497) |
| contentbox | selected_component | method 'version' not found in dbinfo | 13 | [Update.cfc:505](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.0.0-beta/Update.cfc:505) |
| lucee | native_interface | method 'getName' not found on builtin queryexecute | 7 | [QueryExecute.cfc:51](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryExecute.cfc:51) |
| lucee | native_interface | method 'getColumnCount' not found on builtin querynew | 6 | [QueryAddRow.cfc:11](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryAddRow.cfc:11) |
| wheels | selected_component | method 'pwd' not found in shell | 5 | [MCPService.cfc:164](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/MCPService.cfc:164) |
| coldbox | selected_component | method 'getObjectState' not found in tests.tmp.User | 4 | [performance-parallel-tests.cfm:58](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/suites/async/performance-parallel-tests.cfm:58) |
| coldbox | selected_component | method 'getRenderedContent' not found in coldbox.system.web.context.RequestContext | 4 | [EventExecutionsSpec.cfc:70](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/integration/EventExecutionsSpec.cfc:70) |
| contentbox | selected_component | method 'setNextEvent' not found in coldbox.system.web.Controller | 4 | [Update.cfc:94](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/build/patches/3.1.0/Update.cfc:94) |
| lucee | native_interface | method 'getDialect' not found on builtin native.sessionfactory | 4 | [index.cfm:2](/Users/garethedwards/corpus/lucee_Lucee/test/orm/testDialects/testH2/index.cfm:2) |
| lucee | native_interface | method 'getDialect' not found on builtin native.sessionfactory (chain to 'toString') | 4 | [index.cfm:2](/Users/garethedwards/corpus/lucee_Lucee/test/orm/testDialects/testH2/index.cfm:2) |
| coldbox | selected_component | method 'getNormalizedID' not found in coldbox.system.cache.store.indexers.JDBCMetadataIndexer | 3 | [JDBCMetadataIndexer.cfc:98](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/store/indexers/JDBCMetadataIndexer.cfc:98) |
| coldbox | selected_component | method 'injectState' not found in tests.tmp.User | 3 | [performance-parallel-tests.cfm:30](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/suites/async/performance-parallel-tests.cfm:30) |
| coldbox | selected_component | method 'isInstanceCheck' not found in coldbox.system.core.util.Util | 3 | [UtilTest.cfc:10](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/core/util/UtilTest.cfc:10) |
| lucee | native_interface | method 'getColumnList' not found on builtin querynew | 3 | [LDEV1160.cfc:10](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1160.cfc:10) |
| lucee | native_interface | method 'getResourceProviders' not found on builtin native.lucee.config | 3 | [Resources.cfc:695](/Users/garethedwards/corpus/lucee_Lucee/test/general/Resources.cfc:695) |
| lucee | native_interface | method 'getRestMappings' not found on builtin native.lucee.config | 3 | [LDEV4835.cfc:14](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4835.cfc:14) |
| lucee | native_interface | method 'isEmpty' not found on builtin querynew | 3 | [QueryIsEmpty.cfc:13](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryIsEmpty.cfc:13) |
| coldbox | selected_component | method 'filter' not found in coldbox.system.web.flash.AbstractFlashScope (chain to 'each') | 2 | [AbstractFlashScope.cfc:116](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:116) |
| coldbox | selected_component | method 'getAppStartHandlerFired' not found in coldbox.system.web.Controller | 2 | [ControllerTest.cfc:75](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/web/ControllerTest.cfc:75) |
| coldbox | selected_component | method 'getConfig' not found in LogBox (chain to 'getCategories') | 2 | [LogBoxBuildTests.cfc:34](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/logging/LogBoxBuildTests.cfc:34) |
| coldbox | selected_component | method 'getConfig' not found in LogBox (chain to 'keyArray') | 2 | [LogBoxBuildTests.cfc:35](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/logging/LogBoxBuildTests.cfc:35) |
| coldbox | selected_component | method 'setIsActive' not found in User | 2 | [HTMLHelperSpec.cfc:528](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/orm-enabled/HTMLHelperSpec.cfc:528) |
| contentbox | selected_component | method 'generatePasswordResetToken' not found in securityService@contentbox | 2 | [authSpec.cfc:224](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/tests/specs/contentbox-api/authSpec.cfc:224) |
| fw1 | selected_component | method 'itemTest' not found in user | 2 | [ExtraBeansTest.cfc:40](/Users/garethedwards/corpus/framework-one_fw1/tests/ExtraBeansTest.cfc:40) |
| lucee | native_interface | method 'cleanDatasourceConnectionPools' not found on builtin native.lucee.config | 2 | [LDEV5967.cfc:93](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5967.cfc:93) |
| lucee | native_interface | method 'getCGIScopeReadOnly' not found on builtin native.lucee.config | 2 | [LDEV0372.cfc:40](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0372.cfc:40) |
| lucee | native_interface | method 'getComponentMappings' not found on builtin native.lucee.config | 2 | [Application.cfc:103](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/doc/Application.cfc:103) |
| lucee | native_interface | method 'getConfigServerDir' not found on builtin native.lucee.config | 2 | [login.new.cfm:6](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/login.new.cfm:6) |
| lucee | native_interface | method 'getFullNullSupport' not found on builtin native.lucee.config | 2 | [Jira2384.cfc:91](/Users/garethedwards/corpus/lucee_Lucee/test/jira/Jira2384.cfc:91) |
| lucee | native_interface | method 'getMode' not found on builtin fileopen | 2 | [FileOpen.cfc:10](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:10) |
| lucee | native_interface | method 'getResource' not found on builtin fileopen | 2 | [DirectoryDelete.cfc:77](/Users/garethedwards/corpus/lucee_Lucee/test/functions/DirectoryDelete.cfc:77) |
| lucee | native_interface | method 'getSecurityDirectory' not found on builtin native.lucee.config | 2 | [SSLCertificateInstall.cfc:14](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SSLCertificateInstall.cfc:14) |
| lucee | native_interface | method 'getSize' not found on builtin fileopen | 2 | [FileOpen.cfc:57](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:57) |
| lucee | native_interface | method 'getSql' not found on builtin queryexecute | 2 | [LDEV3863.cfc:12](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3863.cfc:12) |
| lucee | native_interface | method 'getSql' not found on builtin queryexecute (chain to 'toString') | 2 | [LDEV3863.cfc:12](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3863.cfc:12) |
| lucee | native_interface | method 'getStatus' not found on builtin fileopen | 2 | [FileOpen.cfc:11](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:11) |
| lucee | native_interface | method 'hasCFSession' not found on builtin native.lucee.pagecontext | 2 | [testThreadCookies.cfm:20](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5292/cfml-session/testThreadCookies.cfm:20) |
| lucee | native_interface | method 'insert' not found on builtin native.array | 2 | [structinsert.cfc:15](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structinsert.cfc:15) |
| lucee | native_interface | method 'keyExists' not found on builtin querynew | 2 | [QueryKeyExists.cfc:34](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryKeyExists.cfc:34) |
| lucee | native_interface | method 'listItemTrim' not found on builtin native.string | 2 | [listitemtrim.cfc:12](/Users/garethedwards/corpus/lucee_Lucee/test/functions/listitemtrim.cfc:12) |
| lucee | native_interface | method 'round' not found on builtin native.numeric | 2 | [LDEV1824.cfc:32](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:32) |
| lucee | native_interface | method 'round' not found on builtin native.string | 2 | [LDEV332.cfc:69](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV332.cfc:69) |
| lucee | native_interface | method 'setDate' not found on builtin native.date | 2 | [SetDay.cfc:43](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SetDay.cfc:43) |
| lucee | native_interface | method 'sgn' not found on builtin native.numeric | 2 | [sgn.cfc:9](/Users/garethedwards/corpus/lucee_Lucee/test/functions/sgn.cfc:9) |
| lucee | native_interface | method 'sin' not found on builtin native.numeric | 2 | [sin.cfc:30](/Users/garethedwards/corpus/lucee_Lucee/test/functions/sin.cfc:30) |
| lucee | native_interface | method 'sqr' not found on builtin native.numeric | 2 | [sqr.cfc:7](/Users/garethedwards/corpus/lucee_Lucee/test/functions/sqr.cfc:7) |
| lucee | selected_component | method 'mixin' not found in User | 2 | [test.cfm:7](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3473/test.cfm:7) |
| wheels | selected_component | method 'getServerInfoJSON' not found in ServerService | 2 | [benchmark.cfc:156](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/benchmark.cfc:156) |
| wheels | selected_component | method 'humanize' not found in helpers@wheels-cli | 2 | [AdminIntrospectionService.cfc:333](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/AdminIntrospectionService.cfc:333) |
| coldbox | native_interface | method 'addTo' not found on builtin native.duration | 1 | [Duration.cfc:227](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:227) |
| coldbox | native_interface | method 'clearStatistics' not found on builtin native.string | 1 | [AbstractCacheBoxProvider.cfc:155](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/AbstractCacheBoxProvider.cfc:155) |
| coldbox | native_interface | method 'get' not found on builtin native.duration | 1 | [Duration.cfc:95](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:95) |
| coldbox | native_interface | method 'getAllClassMetadata' not found on builtin native.sessionfactory | 1 | [ObjectPopulator.cfc:612](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc:612) |
| coldbox | native_interface | method 'getMetaModel' not found on builtin native.sessionfactory | 1 | [ObjectPopulator.cfc:609](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc:609) |
| coldbox | native_interface | method 'getMetaModel' not found on builtin native.sessionfactory (chain to 'getAllEntityNames') | 1 | [ObjectPopulator.cfc:609](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/core/dynamic/ObjectPopulator.cfc:609) |
| coldbox | native_interface | method 'getUnits' not found on builtin native.duration | 1 | [Duration.cfc:116](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:116) |
| coldbox | native_interface | method 'keys' not found on builtin native.map | 1 | [ConcurrentStore.cfc:69](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/store/ConcurrentStore.cfc:69) |
| coldbox | native_interface | method 'of' not found on builtin native.duration | 1 | [Duration.cfc:435](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:435) |
| coldbox | native_interface | method 'plusNanos' not found on builtin native.localdatetime | 1 | [ColdBoxScheduledTask.cfc:351](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/tasks/ColdBoxScheduledTask.cfc:351) |
| coldbox | native_interface | method 'setDelegates' not found on builtin native.array | 1 | [Binder.cfc:927](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/ioc/config/Binder.cfc:927) |
| coldbox | native_interface | method 'subtractFrom' not found on builtin native.duration | 1 | [Duration.cfc:240](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:240) |
| coldbox | native_interface | method 'toDaysPart' not found on builtin native.duration | 1 | [Duration.cfc:132](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:132) |
| coldbox | native_interface | method 'toHoursPart' not found on builtin native.duration | 1 | [Duration.cfc:146](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:146) |
| coldbox | native_interface | method 'toMillisPart' not found on builtin native.duration | 1 | [Duration.cfc:160](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:160) |
| coldbox | native_interface | method 'toMinutesPart' not found on builtin native.duration | 1 | [Duration.cfc:174](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:174) |
| coldbox | native_interface | method 'toNanos' not found on builtin native.duration | 1 | [Duration.cfc:181](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:181) |
| coldbox | native_interface | method 'toNanosPart' not found on builtin native.duration | 1 | [Duration.cfc:188](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:188) |
| coldbox | native_interface | method 'toSecondsPart' not found on builtin native.duration | 1 | [Duration.cfc:202](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/async/time/Duration.cfc:202) |
| coldbox | selected_component | method 'filter' not found in coldbox.system.web.flash.AbstractFlashScope (chain to 'keyArray') | 1 | [AbstractFlashScope.cfc:115](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/flash/AbstractFlashScope.cfc:115) |
| coldbox | selected_component | method 'getRandom' not found in PhotosService | 1 | [Scheduler.cfc:9](/Users/garethedwards/corpus/ColdBox_coldbox-platform/test-harness/modules_app/resourcesTest/config/Scheduler.cfc:9) |
| coldbox | selected_component | method 'getValuesss' not found in coldbox.system.web.context.RequestContext | 1 | [testerror.cfc:32](/Users/garethedwards/corpus/ColdBox_coldbox-platform/test-harness/handlers/testerror.cfc:32) |
| coldbox | selected_component | method 'parseAndLoad' not found in coldbox.system.cache.config.CacheBoxConfig | 1 | [LoaderService.cfc:233](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/web/services/LoaderService.cfc:233) |
| coldbox | selected_component | method 'setAppStartHandlerFired' not found in coldbox.system.web.Controller | 1 | [ControllerTest.cfc:76](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/specs/web/ControllerTest.cfc:76) |
| coldbox | selected_component | method 'showdebugpanel' not found in coldbox.system.web.context.RequestContext | 1 | [Layout.None.cfm:2](/Users/garethedwards/corpus/ColdBox_coldbox-platform/tests/suites/eventCachingCollisions/layouts/Layout.None.cfm:2) |
| coldbox | selected_component | method 'size' not found in coldbox.system.cache.store.indexers.JDBCMetadataIndexer | 1 | [JDBCMetadataIndexer.cfc:204](/Users/garethedwards/corpus/ColdBox_coldbox-platform/system/cache/store/indexers/JDBCMetadataIndexer.cfc:204) |
| contentbox | selected_component | method 'deliverMedia' not found in BaseProvider | 1 | [media.cfc:87](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-ui/handlers/media.cfc:87) |
| contentbox | selected_component | method 'getDisplayExpiredDate' not found in Page | 1 | [quickLook.cfm:15](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-admin/views/content/quickLook.cfm:15) |
| contentbox | selected_component | method 'getLayout' not found in BaseContent | 1 | [index.cfm:76](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/modules/contentbox-ui/views/adminbar/index.cfm:76) |
| contentbox | selected_component | method 'keyArray' not found in qb.models.Schema.SchemaBuilder (chain to 'filter') | 1 | [ContentTemplateService.cfc:177](/Users/garethedwards/corpus/Ortus-Solutions_ContentBox/modules/contentbox/models/content/ContentTemplateService.cfc:177) |
| fw1 | selected_component | method 'getItemLamb' not found in user | 1 | [ExtraBeansTest.cfc:43](/Users/garethedwards/corpus/framework-one_fw1/tests/ExtraBeansTest.cfc:43) |
| fw1 | selected_component | method 'getProduct' not found in user | 1 | [ModelTest.cfc:36](/Users/garethedwards/corpus/framework-one_fw1/tests/ModelTest.cfc:36) |
| lucee | native_interface | method 'BitSHRN' not found on builtin native.numeric | 1 | [LDEV1824.cfc:37](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:37) |
| lucee | native_interface | method 'ClearRect' not found on builtin imageread | 1 | [imageClearRect.cfc:22](/Users/garethedwards/corpus/lucee_Lucee/test/functions/imageClearRect.cfc:22) |
| lucee | native_interface | method 'DrawBeveledRect' not found on builtin imagenew | 1 | [ImageDrawBeveledRect.cfc:22](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ImageDrawBeveledRect.cfc:22) |
| lucee | native_interface | method 'Duplicate' not found on builtin querynew | 1 | [LDEV1295.cfc:13](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1295.cfc:13) |
| lucee | native_interface | method 'KeyArray' not found on builtin native.array | 1 | [LDEV4725.cfc:26](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:26) |
| lucee | native_interface | method 'KeyList' not found on builtin native.array | 1 | [LDEV4725.cfc:14](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4725.cfc:14) |
| lucee | native_interface | method 'Len' not found on builtin native.numeric | 1 | [LDEV332.cfc:26](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV332.cfc:26) |
| lucee | native_interface | method 'Resize' not found on builtin native.boolean | 1 | [LDEV1844.cfc:10](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1844.cfc:10) |
| lucee | native_interface | method 'Resize' not found on builtin native.boolean (chain to 'set') | 1 | [LDEV1844.cfc:10](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1844.cfc:10) |
| lucee | native_interface | method 'Shear' not found on builtin imagenew | 1 | [LDEV4459.cfc:7](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4459.cfc:7) |
| lucee | native_interface | method 'ToStruct' not found on builtin queryexecute | 1 | [LDEV5327.cfc:44](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5327.cfc:44) |
| lucee | native_interface | method 'aCos' not found on builtin native.numeric | 1 | [LDEV1824.cfc:11](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:11) |
| lucee | native_interface | method 'abs' not found on builtin native.numeric | 1 | [LDEV1824.cfc:10](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:10) |
| lucee | native_interface | method 'asin' not found on builtin native.numeric | 1 | [LDEV1824.cfc:12](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:12) |
| lucee | native_interface | method 'atn' not found on builtin native.numeric | 1 | [LDEV1824.cfc:13](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:13) |
| lucee | native_interface | method 'beginTransaction' not found on builtin native.hibernatesession | 1 | [LDEV4185.cfm:2](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4185/LDEV4185.cfm:2) |
| lucee | native_interface | method 'bitAnd' not found on builtin native.numeric | 1 | [LDEV1824.cfc:14](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:14) |
| lucee | native_interface | method 'bitMaskClear' not found on builtin native.numeric | 1 | [LDEV1824.cfc:15](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:15) |
| lucee | native_interface | method 'bitMaskRead' not found on builtin native.numeric | 1 | [LDEV1824.cfc:16](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:16) |
| lucee | native_interface | method 'bitMaskset' not found on builtin native.numeric | 1 | [LDEV1824.cfc:17](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:17) |
| lucee | native_interface | method 'bitNot' not found on builtin native.numeric | 1 | [LDEV1824.cfc:18](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:18) |
| lucee | native_interface | method 'bitOR' not found on builtin native.numeric | 1 | [LDEV1824.cfc:19](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:19) |
| lucee | native_interface | method 'bitSHLN' not found on builtin native.numeric | 1 | [LDEV1824.cfc:20](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:20) |
| lucee | native_interface | method 'bitXor' not found on builtin native.numeric | 1 | [LDEV1824.cfc:21](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:21) |
| lucee | native_interface | method 'booleanFormat' not found on builtin native.numeric | 1 | [LDEV4589.cfm:6](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4589/LDEV4589.cfm:6) |
| lucee | native_interface | method 'ceiling' not found on builtin native.numeric | 1 | [LDEV1824.cfc:22](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:22) |
| lucee | native_interface | method 'clone' not found on builtin native.array | 1 | [LDEV5353.cfc:17](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5353.cfc:17) |
| lucee | native_interface | method 'clone' not found on builtin native.date | 1 | [LDEV5862.cfc:35](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5862.cfc:35) |
| lucee | native_interface | method 'clone' not found on builtin native.list | 1 | [LDEV5862.cfc:23](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5862.cfc:23) |
| lucee | native_interface | method 'copy' not found on builtin imagenew | 1 | [imageCopy.cfc:22](/Users/garethedwards/corpus/lucee_Lucee/test/functions/imageCopy.cfc:22) |
| lucee | native_interface | method 'cos' not found on builtin native.numeric | 1 | [LDEV1824.cfc:23](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:23) |
| lucee | native_interface | method 'currentRow' not found on builtin querynew | 1 | [QueryCurrentRow.cfc:11](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryCurrentRow.cfc:11) |
| lucee | native_interface | method 'duplicate' not found on builtin native.date | 1 | [LDEV1295.cfc:36](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1295.cfc:36) |
| lucee | native_interface | method 'elemNew' not found on builtin xmlnew | 1 | [LDEV2936.cfc:15](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV2936.cfc:15) |
| lucee | native_interface | method 'entrySet' not found on builtin native.struct | 1 | [LDEV0866.cfc:44](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:44) |
| lucee | native_interface | method 'entrySet' not found on builtin native.struct (chain to 'iterator') | 1 | [LDEV0866.cfc:44](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:44) |
| lucee | native_interface | method 'firstDayOfMonth' not found on builtin native.date | 1 | [FirstDayOfMonth.cfc:15](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FirstDayOfMonth.cfc:15) |
| lucee | native_interface | method 'fix' not found on builtin native.numeric | 1 | [LDEV1824.cfc:25](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:25) |
| lucee | native_interface | method 'floor' not found on builtin native.numeric | 1 | [LDEV1824.cfc:26](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:26) |
| lucee | native_interface | method 'formatBaseN' not found on builtin native.numeric | 1 | [LDEV1824.cfc:27](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:27) |
| lucee | native_interface | method 'getAdminMode' not found on builtin native.lucee.config | 1 | [Administrator.cfc:1681](/Users/garethedwards/corpus/lucee_Lucee/test/components/Administrator.cfc:1681) |
| lucee | native_interface | method 'getBase64String' not found on builtin imagenew | 1 | [LDEV5640.cfc:16](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5640.cfc:16) |
| lucee | native_interface | method 'getCFID' not found on builtin native.lucee.pagecontext | 1 | [call.cfm:7](/Users/garethedwards/corpus/lucee_Lucee/test/general/ScopeStorage/call.cfm:7) |
| lucee | native_interface | method 'getClassDirectory' not found on builtin native.lucee.config | 1 | [BinaryTemplates.cfc:6](/Users/garethedwards/corpus/lucee_Lucee/test/general/BinaryTemplates.cfc:6) |
| lucee | native_interface | method 'getConfigServerDir' not found on builtin native.lucee.config (chain to 'toString') | 1 | [login.new.cfm:6](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/login.new.cfm:6) |
| lucee | native_interface | method 'getConfigServerImpl' not found on builtin native.lucee.config | 1 | [LDEV5185.cfc:34](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5185.cfc:34) |
| lucee | native_interface | method 'getConfigServerImpl' not found on builtin native.lucee.config (chain to 'resetAll') | 1 | [LDEV5185.cfc:34](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5185.cfc:34) |
| lucee | native_interface | method 'getCustomTagMappings' not found on builtin native.lucee.config | 1 | [Mappings.cfc:79](/Users/garethedwards/corpus/lucee_Lucee/test/general/Mappings.cfc:79) |
| lucee | native_interface | method 'getDatasourceName' not found on builtin queryexecute | 1 | [LDEV3070.cfm:13](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3070/LDEV3070.cfm:13) |
| lucee | native_interface | method 'getImageBytes' not found on builtin imagenew | 1 | [test.cfm:4](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1576/test.cfm:4) |
| lucee | native_interface | method 'getJmxName' not found on builtin native.pool | 1 | [LDEV5967.cfc:148](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5967.cfc:148) |
| lucee | native_interface | method 'getMaxTotal' not found on builtin native.pool | 1 | [LDEV6051.cfc:98](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV6051.cfc:98) |
| lucee | native_interface | method 'getMeta' not found on builtin querynew | 1 | [test.cfm:4](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1289/test.cfm:4) |
| lucee | native_interface | method 'getMeta' not found on builtin querynew (chain to 'getcolumnlabels') | 1 | [test.cfm:4](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1289/test.cfm:4) |
| lucee | native_interface | method 'getMetaData' not found on builtin fileopen | 1 | [FileOpen.cfc:45](/Users/garethedwards/corpus/lucee_Lucee/test/functions/FileOpen.cfc:45) |
| lucee | native_interface | method 'getResourceAsStream' not found on builtin native.class | 1 | [LDEV6171.cfc:13](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV6171.cfc:13) |
| lucee | native_interface | method 'getServerRHExtensions' not found on builtin native.lucee.config | 1 | [LDEV2558.cfc:14](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV2558.cfc:14) |
| lucee | native_interface | method 'getServlet' not found on builtin native.lucee.factory | 1 | [overview.cfm:56](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/overview.cfm:56) |
| lucee | native_interface | method 'getmetadata' not found on builtin native.struct | 1 | [GetMetaData.cfc:58](/Users/garethedwards/corpus/lucee_Lucee/test/functions/GetMetaData.cfc:58) |
| lucee | native_interface | method 'getresult' not found on builtin queryexecute | 1 | [LDEV2616.cfm:6](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV2616/LDEV2616.cfm:6) |
| lucee | native_interface | method 'insertAt' not found on builtin native.boolean | 1 | [LDEV1844.cfc:6](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1844.cfc:6) |
| lucee | native_interface | method 'keyExists' not found on builtin xmlnew | 1 | [xmlNew.cfc:9](/Users/garethedwards/corpus/lucee_Lucee/test/functions/xmlNew.cfc:9) |
| lucee | native_interface | method 'keyList' not found on builtin native.array | 1 | [web.cfm:303](/Users/garethedwards/corpus/lucee_Lucee/core/src/main/cfml/context/admin/web.cfm:303) |
| lucee | native_interface | method 'keySet' not found on builtin native.struct | 1 | [LDEV0866.cfc:34](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:34) |
| lucee | native_interface | method 'keySet' not found on builtin native.struct (chain to 'iterator') | 1 | [LDEV0866.cfc:34](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:34) |
| lucee | native_interface | method 'log' not found on builtin native.numeric | 1 | [LDEV1824.cfc:28](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:28) |
| lucee | native_interface | method 'log10' not found on builtin native.numeric | 1 | [LDEV1824.cfc:29](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:29) |
| lucee | native_interface | method 'max' not found on builtin native.numeric | 1 | [LDEV1824.cfc:30](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:30) |
| lucee | native_interface | method 'min' not found on builtin native.numeric | 1 | [LDEV1824.cfc:31](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:31) |
| lucee | native_interface | method 'parseDateTime' not found on builtin native.string | 1 | [LDEV0028.cfc:13](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0028.cfc:13) |
| lucee | native_interface | method 'part' not found on builtin native.date | 1 | [DatePart.cfc:13](/Users/garethedwards/corpus/lucee_Lucee/test/functions/DatePart.cfc:13) |
| lucee | native_interface | method 'push' not found on builtin native.array | 1 | [ArrayPush.cfc:11](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ArrayPush.cfc:11) |
| lucee | native_interface | method 'renameColumn' not found on builtin querynew | 1 | [QueryRenameColumn.cfc:31](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QueryRenameColumn.cfc:31) |
| lucee | native_interface | method 'rotateDrawingAxis' not found on builtin imagenew | 1 | [ImageRotateDrawingAxis.cfc:23](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ImageRotateDrawingAxis.cfc:23) |
| lucee | native_interface | method 'setLastBorrowed' not found on builtin native.pool | 1 | [LDEV5967.cfc:170](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5967.cfc:170) |
| lucee | native_interface | method 'setMetadata' not found on builtin native.struct | 1 | [LDEV0974.cfc:98](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0974.cfc:98) |
| lucee | native_interface | method 'setMinEvictableIdleTimeMillis' not found on builtin native.pool | 1 | [LDEV5963.cfc:69](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV5963.cfc:69) |
| lucee | native_interface | method 'setMinutes' not found on builtin native.date | 1 | [SetMinute.cfc:41](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SetMinute.cfc:41) |
| lucee | native_interface | method 'setSeconds' not found on builtin native.date | 1 | [SetSecond.cfc:41](/Users/garethedwards/corpus/lucee_Lucee/test/functions/SetSecond.cfc:41) |
| lucee | native_interface | method 'shearDrawingAxis' not found on builtin imagenew | 1 | [ImageShearDrawingAxis.cfc:23](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ImageShearDrawingAxis.cfc:23) |
| lucee | native_interface | method 'slice' not found on builtin querynew | 1 | [QuerySlice.cfc:12](/Users/garethedwards/corpus/lucee_Lucee/test/functions/QuerySlice.cfc:12) |
| lucee | native_interface | method 'sort' not found on builtin native.struct | 1 | [structsort.cfc:93](/Users/garethedwards/corpus/lucee_Lucee/test/functions/structsort.cfc:93) |
| lucee | native_interface | method 'tan' not found on builtin native.numeric | 1 | [LDEV1824.cfc:36](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1824.cfc:36) |
| lucee | native_interface | method 'toBase64' not found on builtin native.string | 1 | [toBase64.cfc:7](/Users/garethedwards/corpus/lucee_Lucee/test/functions/toBase64.cfc:7) |
| lucee | native_interface | method 'toList' not found on builtin native.boolean | 1 | [functionHasTests.cfc:20](/Users/garethedwards/corpus/lucee_Lucee/test/functions/functionHasTests.cfc:20) |
| lucee | native_interface | method 'tokey' not found on builtin native.array | 1 | [LDEV3167.cfc:31](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3167.cfc:31) |
| lucee | native_interface | method 'tokey' not found on builtin native.date | 1 | [LDEV3167.cfc:45](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV3167.cfc:45) |
| lucee | native_interface | method 'trimWhitespace' not found on builtin native.string | 1 | [TrimWhitespace.cfc:11](/Users/garethedwards/corpus/lucee_Lucee/test/functions/TrimWhitespace.cfc:11) |
| lucee | native_interface | method 'trimWhitespace' not found on builtin native.string (chain to 'len') | 1 | [TrimWhitespace.cfc:11](/Users/garethedwards/corpus/lucee_Lucee/test/functions/TrimWhitespace.cfc:11) |
| lucee | native_interface | method 'unshift' not found on builtin native.array | 1 | [ArrayUnShift.cfc:11](/Users/garethedwards/corpus/lucee_Lucee/test/functions/ArrayUnShift.cfc:11) |
| lucee | native_interface | method 'update' not found on builtin native.array | 1 | [StructUpdate.cfc:10](/Users/garethedwards/corpus/lucee_Lucee/test/functions/StructUpdate.cfc:10) |
| lucee | native_interface | method 'values' not found on builtin native.struct | 1 | [LDEV0866.cfc:55](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:55) |
| lucee | native_interface | method 'values' not found on builtin native.struct (chain to 'iterator') | 1 | [LDEV0866.cfc:55](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV0866.cfc:55) |
| lucee | native_interface | method 'writeBase64' not found on builtin imagenew | 1 | [imageWritebase64.cfc:21](/Users/garethedwards/corpus/lucee_Lucee/test/functions/imageWritebase64.cfc:21) |
| lucee | native_interface | method 'writeBase64' not found on builtin imageread | 1 | [imageReadbase64.cfc:22](/Users/garethedwards/corpus/lucee_Lucee/test/functions/imageReadbase64.cfc:22) |
| lucee | selected_component | method 'getMessage' not found in LDEV1962.component2 | 1 | [LDEV1962.cfc:28](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1962.cfc:28) |
| lucee | selected_component | method 'getMessage2' not found in LDEV1962.component2 | 1 | [LDEV1962.cfc:33](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV1962.cfc:33) |
| lucee | selected_component | method 'getStr' not found in Component | 1 | [LDEV4156.cfc:15](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/LDEV4156.cfc:15) |
| lucee | selected_component | method 'returnsany' not found in LDEV1835.Comp | 1 | [_LDEV1835.cfc:6](/Users/garethedwards/corpus/lucee_Lucee/test/tickets/_LDEV1835.cfc:6) |
| masa | native_interface | method 'doRequest' not found on builtin native.string | 1 | [contentServer.cfc:797](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/content/contentServer.cfc:797) |
| masa | native_interface | method 'onRequestEnd' not found on builtin native.string | 1 | [contentServer.cfc:851](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/content/contentServer.cfc:851) |
| masa | native_interface | method 'onRequestStart' not found on builtin native.string | 1 | [contentServer.cfc:793](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/content/contentServer.cfc:793) |
| masa | selected_component | method 'getDbType' not found in dbUtility | 1 | [beanORM.cfc:189](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/bean/beanORM.cfc:189) |
| masa | selected_component | method 'getReportData' not found in contentUtility | 1 | [contentManager.cfc:1989](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/content/contentManager.cfc:1989) |
| masa | selected_component | method 'getSessionSearch' not found in emailGateway | 1 | [dashboardManager.cfc:139](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/dashboard/dashboardManager.cfc:139) |
| masa | selected_component | method 'hasValue' not found in mura.cfobject | 1 | [publisherKeys.cfc:67](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/publisherKeys.cfc:67) |
| masa | selected_component | method 'readByEmail' not found in mura.user.userManager | 1 | [user.cfc:70](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/client/api/soap/v1/user.cfc:70) |
| masa | selected_component | method 'setTranslator' not found in trashItemBean | 1 | [trashIterator.cfc:54](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/trash/trashIterator.cfc:54) |
| masa | selected_component | method 'setValidations' not found in apiUtility | 1 | [apiUtility.cfc:3662](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/client/api/json/v1/apiUtility.cfc:3662) |
| masa | selected_component | method 'setValidations' not found in apiUtility (chain to 'getErrors') | 1 | [apiUtility.cfc:3662](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/client/api/json/v1/apiUtility.cfc:3662) |
| masa | selected_component | method 'setValidations' not found in apiUtility (chain to 'validate') | 1 | [apiUtility.cfc:3662](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/client/api/json/v1/apiUtility.cfc:3662) |
| masa | selected_component | not found in parent component | 1 | [user.cfc:82](/Users/garethedwards/corpus/MSU-NatSci_MuraCMS/core/mura/client/api/soap/v1/user.cfc:82) |
| testbox | native_interface | method 'containsKey' not found on builtin native.struct | 1 | [CoverageGenerator.cfc:190](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/system/coverage/data/CoverageGenerator.cfc:190) |
| testbox | selected_component | method 'exposeMixin' not found in test1 | 1 | [BaseSpec.cfc:1645](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/system/BaseSpec.cfc:1645) |
| testbox | selected_component | method 'virtualReturn' not found in testbox.tests.resources.Test | 1 | [MockBoxTestHarness.cfm:33](/Users/garethedwards/corpus/Ortus-Solutions_TestBox/tests/specs/mockbox/MockBoxTestHarness.cfm:33) |
| wheels | native_interface | method 'getDayOfMonth' not found on builtin native.date | 1 | [formsdate.cfc:126](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) |
| wheels | native_interface | method 'getHour' not found on builtin native.date | 1 | [formsdate.cfc:126](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) |
| wheels | native_interface | method 'getMethod' not found on builtin native.class | 1 | [PasswordHasher.cfc:253](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/auth/PasswordHasher.cfc:253) |
| wheels | native_interface | method 'getMinute' not found on builtin native.date | 1 | [formsdate.cfc:126](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) |
| wheels | native_interface | method 'getMonthValue' not found on builtin native.date | 1 | [formsdate.cfc:126](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) |
| wheels | native_interface | method 'getSecond' not found on builtin native.date | 1 | [formsdate.cfc:126](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:126) |
| wheels | native_interface | method 'timestampValue' not found on builtin native.date | 1 | [formsdate.cfc:128](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/view/formsdate.cfc:128) |
| wheels | selected_component | method '$$pluginOnlyMethod' not found in Test | 1 | [pluginsSpec.cfc:257](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/tests/specs/pluginsSpec.cfc:257) |
| wheels | selected_component | method '$helper01' not found in Test | 1 | [pluginsSpec.cfc:207](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/tests/specs/pluginsSpec.cfc:207) |
| wheels | selected_component | method 'columnNames' not found in superOverride | 1 | [SuperOverrideSpec.cfc:42](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/tests/specs/controller/SuperOverrideSpec.cfc:42) |
| wheels | selected_component | method 'exposeMixin' not found in ANTJUnitReporter | 1 | [BaseSpec.cfc:1578](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/wheelstest/system/BaseSpec.cfc:1578) |
| wheels | selected_component | method 'fxReinclude' not found in Global | 1 | [reloadGlobalsSpec.cfc:110](/Users/garethedwards/corpus/cfwheels_cfwheels/vendor/wheels/tests/specs/global/reloadGlobalsSpec.cfc:110) |
| wheels | selected_component | method 'getCWD' not found in fileSystem | 1 | [MCPService.cfc:28](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/models/MCPService.cfc:28) |
| wheels | selected_component | method 'toConsole' not found in DetailOutputService | 1 | [push.cfc:393](/Users/garethedwards/corpus/cfwheels_cfwheels/cli/src/commands/wheels/docker/push.cfc:393) |
