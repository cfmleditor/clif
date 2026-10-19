# cx-p — unresolved findings and candidates

Every finding below is still reported. A candidate is a liberal match offered for a person to weigh, never an answer the resolver took: **high** is the only component declaring every method the function calls on the receiver (or the only one, named like it); **medium** is a sole match on weaker evidence, or the one named like the receiver among several; **low** is one of several. *Defined* is how many indexed files declare a method of that name: 0 means it is missing from the workspace, more means the resolver could not connect the call to it.

| Category | Findings | high | medium | low | none | method defined nowhere |
|---|---:|---:|---:|---:|---:|---:|
| variable | 1246 | 284 | 375 | 177 | 410 | 309 |
| return-type | 184 | 4 | 57 | 63 | 60 | 55 |
| method | 126 | 0 | 53 | 30 | 43 | 37 |
| object | 52 | 0 | 0 | 5 | 47 | 14 |

## Variable definitions — a receiver whose component is unknown

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 43 | `mapping → system/ioc/config/Mapping.cfc` | 1 | high | declares isAspect(), getName(); named like the receiver 'mapping' | `system/aop/Mixer.cfc:104` variable 'mapping' has no component ref |
| 37 | `variables.withclosure → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/web/routing/Router.cfc:601` variable 'variables.withClosure' has no component ref |
| 26 | `arguments.ehbean → system/web/context/EventHandlerBean.cfc` | 1 | high | declares getModule(), isModule(), getRunnable() | `system/web/services/HandlerService.cfc:103` variable 'arguments.ehBean' has no component ref |
| 25 | `results.ehbean → system/web/context/EventHandlerBean.cfc` | 1 | high | declares getViewDispatch(), getActionMetadata(), getMethod(), isMissingAction(), getMissingAction() | `system/web/Controller.cfc:826` variable 'results.ehBean' has no component ref |
| 24 | `variables.router → system/web/routing/Router.cfc` | 1 | high | declares isValidExtension(), getMimeExtensionAlias(), route(), getRoutes(); named like the receiver 'router' | `tests/specs/web/routing/RouterSSETest.cfc:49` variable 'variables.router' has no component ref |
| 23 | `item` | 0 | none |  | `build/Build.cfc:498` variable 'item' has no component ref |
| 22 | `cacheconfig` | 0 | none |  | `system/cache/providers/CFProvider.cfc:503` variable 'cacheConfig' has no component ref |
| 20 | `e` | 1 | none |  | `tests/specs/integration/EventExecutionsSpec.cfc:16` variable 'e' has no component ref |
| 18 | `arguments.mapping → system/ioc/config/Mapping.cfc` | 2 | medium | declares getObjectMetadata(); named like the receiver 'mapping' (2 candidates) | `system/aop/Mixer.cfc:215` variable 'arguments.mapping' has no component ref |
| 17 | `orequestcontext → system/web/context/RequestContext.cfc` | 2 | high | declares getSESBaseURL(); named like the receiver 'oRequestContext' | `system/web/Controller.cfc:544` variable 'oRequestContext' has no component ref |
| 16 | `eng.versions → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/perf-harness/PerformanceSuite.cfc:539` variable 'eng.versions' has no component ref |
| 16 | `ohandler` | 2 | none |  | `system/web/Controller.cfc:859` variable 'oHandler' has no component ref |
| 16 | `results.config → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares injectPropertyMixin() | `system/web/services/ModuleService.cfc:1291` variable 'results.config' has no component ref |
| 15 | `application[] → system/web/Controller.cfc` | 1 | high | declares getLoaderService(), getSetting(), runEvent(), getLogBox() | `system/Bootstrap.cfc:101` variable 'application[]' has no component ref |
| 15 | `variables.native → system/async/executors/Executor.cfc` | 1 | medium | declares submit() | `system/async/executors/Executor.cfc:149` variable 'variables.native' has no component ref |
| 14 | `arguments.now` | 0 | none |  | `system/async/tasks/ScheduledTask.cfc:1635` variable 'arguments.now' has no component ref |
| 14 | `arguments.oconfig → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares getPropertyMixin() | `system/web/config/ApplicationLoader.cfc:233` variable 'arguments.oConfig' has no component ref |
| 14 | `emitter → system/web/context/SSEEmitter.cfc` | 2 | low | declares isClosed(), send() (2 candidates) | `system/web/routing/Router.cfc:2235` variable 'emitter' has no component ref |
| 12 | `cachestats → system/cache/util/CacheStats.cfc` | 5 | medium | declares getHits(), getMisses(), getGarbageCollections(), getEvictionCount(); named like the receiver 'cacheStats' (5 candidates) | `system/cache/report/skins/default/CacheCharting.cfm:37` variable 'cacheStats' has no component ref |
| 12 | `event1 → system/web/context/RequestContext.cfc` | 2 | medium | declares getPrivateCollection() | `tests/specs/integration/EventCachingSpec.cfc:41` variable 'event1' has no component ref |
| 12 | `variables.asyncmanager → system/async/AsyncManager.cfc` | 5 | medium | declares out(); named like the receiver 'asyncManager' (5 candidates) | `system/async/tasks/Scheduler.cfc:153` variable 'variables.asyncManager' has no component ref |
| 11 | `e → system/testing/BaseTestCase.cfc` | 1 | high | declares getHandlerResults(), getRenderedContent(), getRenderData() | `tests/specs/integration/RenderingsSpec.cfc:16` variable 'e' has no component ref |
| 11 | `expectation.actual → system/web/context/Response.cfc` | 3 | high | declares getStatusCode(), getMemento(), getData() | `system/testing/CustomMatchers.cfc:64` variable 'expectation.actual' has no component ref |
| 11 | `results.config` | 1 | none |  | `system/web/services/ModuleService.cfc:1289` variable 'results.config' has no component ref |
| 10 | `arguments.target` | 1 | none |  | `system/async/time/DateTimeHelper.cfc:78` variable 'arguments.target' has no component ref |
| 10 | `obean → test-harness/models/formBean.cfc` | 1 | high | declares getFname(), getLname() | `tests/specs/FrameworkSuperTypeTest.cfc:70` variable 'oBean' has no component ref |
| 9 | `caches.cache1` | 1 | none |  | `tests/specs/cache/CacheFactoryTest.cfc:62` variable 'caches.cache1' has no component ref |
| 9 | `event2 → system/web/context/RequestContext.cfc` | 2 | medium | declares getPrivateCollection() | `tests/specs/integration/EventCachingSpec.cfc:52` variable 'event2' has no component ref |
| 9 | `url → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `apidocs/Application.cfc:15` variable 'url' has no component ref |
| 8 | `result` | 0 | none |  | `system/core/delegates/StringUtil.cfc:213` variable 'result' has no component ref |
| 8 | `services.interceptorservice → system/web/services/InterceptorService.cfc` | 6 | medium | declares announce(); named like the receiver 'interceptorService' (5 candidates) | `system/web/Controller.cfc:570` variable 'services.interceptorService' has no component ref |
| 8 | `variables.native` | 0 | none |  | `system/async/executors/Executor.cfc:164` variable 'variables.native' has no component ref |
| 7 | `arguments.target → system/async/time/Duration.cfc` | 2 | high | declares plusDays(), plusHours(), plusMinutes(), plusSeconds(), plusNanos() | `system/async/time/DateTimeHelper.cfc:399` variable 'arguments.target' has no component ref |
| 7 | `arguments.targetid` | 0 | none |  | `system/ioc/Injector.cfc:1268` variable 'arguments.targetID' has no component ref |
| 7 | `exception → system/web/context/ExceptionBean.cfc` | 2 | high | declares getType(), getMessage(), getDetail(), getExtendedInfo(), getTagContext(), getStackTrace(); named like the receiver 'exception' | `test-harness/views/_templates/generic_error.cfm:12` variable 'exception' has no component ref |
| 7 | `oscheduler → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares injectPropertyMixin() | `system/web/services/SchedulerService.cfc:126` variable 'oScheduler' has no component ref |
| 7 | `services.handlerservice → system/web/services/HandlerService.cfc` | 1 | high | declares getHandlerBean(), getHandler(); named like the receiver 'handlerService' | `system/web/Controller.cfc:821` variable 'services.handlerService' has no component ref |
| 7 | `services.requestservice → system/web/services/RequestService.cfc` | 4 | high | declares getContext(), getFlashScope(); named like the receiver 'requestService' | `system/web/Controller.cfc:459` variable 'services.requestService' has no component ref |
| 7 | `taskrecord.task → system/async/tasks/ScheduledTask.cfc` | 1 | high | declares isDisabled(), getName(), start() | `system/async/tasks/Scheduler.cfc:185` variable 'taskRecord.task' has no component ref |
| 6 | `annotations → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/core/dynamic/ObjectPopulator.cfc:587` variable 'annotations' has no component ref |
| 6 | `arguments.item` | 0 | none |  | `system/ioc/config/Mapping.cfc:808` variable 'arguments.item' has no component ref |
| 6 | `caches.cache2` | 1 | none |  | `tests/specs/cache/CacheFactoryTest.cfc:63` variable 'caches.cache2' has no component ref |
| 6 | `loc.orm` | 0 | none |  | `system/modules/HTMLHelper/models/HTMLHelper.cfc:2135` variable 'loc.orm' has no component ref |
| 6 | `my.out` | 0 | none |  | `system/remote/RemotingUtil.cfc:29` variable 'my.out' has no component ref |
| 6 | `variables.emitter → system/web/context/SSEEmitter.cfc` | 2 | low | declares isClosed() (2 candidates) | `system/web/context/SSEEmitter.cfc:60` variable 'variables.emitter' has no component ref |
| 6 | `variables.executor → system/async/executors/Executor.cfc` | 5 | medium | declares getNative(); named like the receiver 'executor' (5 candidates) | `system/async/tasks/Future.cfc:66` variable 'variables.executor' has no component ref |
| 6 | `variables.thisroute → system/web/routing/Router.cfc` | 3 | low | declares append() (2 candidates) | `system/web/routing/Router.cfc:1803` variable 'variables.thisRoute' has no component ref |
| 5 | `arguments.target → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares injectPropertyMixin() | `system/core/dynamic/ObjectPopulator.cfc:417` variable 'arguments.target' has no component ref |
| 5 | `buffer` | 3 | none |  | `system/core/conversion/XMLConverter.cfc:230` variable 'buffer' has no component ref |
| 5 | `expectation.actual → system/web/context/RequestContext.cfc` | 3 | low | declares getStatusCode(), getMemento() (2 candidates) | `system/testing/CustomMatchers.cfc:29` variable 'expectation.actual' has no component ref |
| 5 | `mconfig.router → system/web/routing/Router.cfc` | 92 | high | declares configure(), getRoutes(), route(); named like the receiver 'router' | `system/web/services/ModuleService.cfc:839` variable 'mConfig.router' has no component ref |
| 5 | `md → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/testing/BaseInterceptorTest.cfc:16` variable 'md' has no component ref |
| 5 | `request.testbox → system/logging/Logger.cfc` | 5 | low | declares debug() (3 candidates) | `system/testing/CustomMatchers.cfc:32` variable 'request.testbox' has no component ref |
| 5 | `routeresults.route.module` | 0 | none |  | `system/web/services/RoutingService.cfc:272` variable 'routeResults.route.module' has no component ref |
| 5 | `runnableinstance` | 80 | none |  | `system/web/routing/Router.cfc:2201` variable 'runnableInstance' has no component ref |
| 5 | `variables.caches → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/cache/CacheFactory.cfc:298` variable 'variables.caches' has no component ref |
| 5 | `variables.loglistener.queue` | 0 | none |  | `system/logging/AbstractAppender.cfc:374` variable 'variables.logListener.queue' has no component ref |
| 5 | `variables.mappings` | 0 | none |  | `system/ioc/config/Binder.cfc:549` variable 'variables.mappings' has no component ref |
| 5 | `variables.native → system/async/tasks/Future.cfc` | 2 | low | declares cancel() (2 candidates) | `system/async/tasks/FutureTask.cfc:38` variable 'variables.native' has no component ref |
| 4 | `arguments.entity` | 0 | none |  | `test-harness/models/entities/BoxLangEventHandler.cfc:33` variable 'arguments.entity' has no component ref |
| 4 | `arguments.targetobject → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares injectMixin() | `system/ioc/Injector.cfc:1185` variable 'arguments.targetObject' has no component ref |
| 4 | `arguments.thismapping → system/ioc/config/Mapping.cfc` | 1 | high | declares isEagerInit(), getName() | `system/ioc/config/Binder.cfc:1344` variable 'arguments.thisMapping' has no component ref |
| 4 | `bet → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:559` variable 'beT' has no component ref |
| 4 | `discoveredevent` | 0 | none |  | `system/web/services/RoutingService.cfc:196` variable 'discoveredEvent' has no component ref |
| 4 | `mconfig.injector → system/ioc/Injector.cfc` | 2 | high | declares getBinder(), registerNewInstance(), getInstance(); named like the receiver 'injector' | `system/web/services/ModuleService.cfc:726` variable 'mConfig.injector' has no component ref |
| 4 | `oscheduler` | 21 | none |  | `system/web/services/SchedulerService.cfc:117` variable 'oScheduler' has no component ref |
| 4 | `results.injector → system/ioc/Injector.cfc` | 2 | high | declares setParent(), getInstance(), getBinder(); named like the receiver 'injector' | `system/web/services/ModuleService.cfc:1253` variable 'results.injector' has no component ref |
| 4 | `results.viewhelperpath → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/web/Renderer.cfc:1007` variable 'results.viewHelperPath' has no component ref |
| 4 | `routeresults.route.verbs` | 0 | none |  | `system/web/services/RoutingService.cfc:277` variable 'routeResults.route.verbs' has no component ref |
| 4 | `server.coldfusion.productname` | 0 | none |  | `system/core/util/CFMLEngine.cfc:68` variable 'server.coldfusion.productName' has no component ref |
| 4 | `services[] → system/web/services/BaseService.cfc` | 7 | low | declares onConfigurationLoad(), afterAspectsLoad() (2 candidates) | `system/web/services/LoaderService.cfc:73` variable 'services[]' has no component ref |
| 4 | `sharedargs → build/Build.cfc` | 1 | medium | declares copy() | `system/web/routing/Router.cfc:2192` variable 'sharedArgs' has no component ref |
| 4 | `sharedargs → system/web/routing/Router.cfc` | 3 | low | declares append() (2 candidates) | `system/web/routing/Router.cfc:2194` variable 'sharedArgs' has no component ref |
| 4 | `stt → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:560` variable 'stT' has no component ref |
| 4 | `variables.locatelayoutcache → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/Renderer.cfc:660` variable 'variables.locateLayoutCache' has no component ref |
| 4 | `variables.locateviewcache → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/Renderer.cfc:817` variable 'variables.locateViewCache' has no component ref |
| 3 | `application[] → system/FrameworkSupertype.cfc` | 3 | low | declares getSetting(), getLog() (2 candidates) | `system/Bootstrap.cfc:449` variable 'application[]' has no component ref |
| 3 | `args.target → system/core/dynamic/MixerUtil.cfc` | 1 | high | declares injectPropertyMixin(), injectMixin() | `system/ioc/Builder.cfc:1115` variable 'args.target' has no component ref |
| 3 | `arguments.logboxconfig → system/cache/config/CacheBoxConfig.cfc` | 2 | low | declares validate() (2 candidates) | `system/web/config/ApplicationLoader.cfc:851` variable 'arguments.logBoxConfig' has no component ref |
| 3 | `arguments.newappender → system/logging/AbstractAppender.cfc` | 1 | high | declares getName(), isInitialized(), onRegistration(), setInitialized() | `system/logging/Logger.cfc:150` variable 'arguments.newAppender' has no component ref |
| 3 | `arguments.relationalmeta.properties[] → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/core/dynamic/ObjectPopulator.cfc:559` variable 'arguments.relationalMeta.properties[]' has no component ref |
| 3 | `arguments.target → system/aop/MixerUtil.cfc` | 1 | high | declares $wbAOPStoreJointPoint(), $wbAOPRemove(), $wbAOPInclude() | `system/aop/Mixer.cfc:383` variable 'arguments.target' has no component ref |
| 3 | `asyncmanager.$executors → system/async/executors/ExecutorBuilder.cfc` | 1 | high | declares newFixedThreadPool(), newCachedThreadPool() | `tests/specs/async/AsyncManagerSpec.cfc:25` variable 'asyncManager.$executors' has no component ref |
| 3 | `bes → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:542` variable 'beS' has no component ref |
| 3 | `exceptionbean → system/web/context/ExceptionBean.cfc` | 2 | high | declares getMessage(), getExceptionStruct(); named like the receiver 'exceptionBean' | `test-harness/handlers/main.cfc:159` variable 'exceptionBean' has no component ref |
| 3 | `explicitview → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/Renderer.cfc:223` variable 'explicitView' has no component ref |
| 3 | `filename` | 0 | none |  | `system/web/services/ModuleService.cfc:103` variable 'fileName' has no component ref |
| 3 | `header.name` | 0 | none |  | `system/web/context/Response.cfc:232` variable 'header.name' has no component ref |
| 3 | `match` | 0 | none |  | `system/web/services/RoutingService.cfc:786` variable 'match' has no component ref |
| 3 | `matches` | 0 | none |  | `system/testing/CustomMatchers.cfc:173` variable 'matches' has no component ref |
| 3 | `mconfig.injector → system/ioc/config/Mapping.cfc` | 1 | medium | declares setVirtualInheritance() | `system/web/services/ModuleService.cfc:828` variable 'mConfig.injector' has no component ref |
| 3 | `oeventhandler → system/EventHandler.cfc` | 2 | high | declares _actionExists(), _actionMetadata(); named like the receiver 'oEventHandler' | `system/web/services/HandlerService.cfc:148` variable 'oEventHandler' has no component ref |
| 3 | `rctarget` | 0 | none |  | `system/cache/util/EventURLFacade.cfc:68` variable 'rcTarget' has no component ref |
| 3 | `response → system/web/context/Response.cfc` | 1 | high | declares getDataPacket(), getError(), getStatusCode(); named like the receiver 'response' | `tests/specs/integration/SubscribersSpec.cfc:27` variable 'response' has no component ref |
| 3 | `sts → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:542` variable 'stS' has no component ref |
| 3 | `targetobject` | 0 | none |  | `system/ioc/Injector.cfc:904` variable 'targetObject' has no component ref |
| 3 | `token` | 0 | none |  | `system/core/util/Util.cfc:204` variable 'token' has no component ref |
| 3 | `variables.endtime` | 1 | none |  | `system/async/tasks/ScheduledTask.cfc:1644` variable 'variables.endTime' has no component ref |
| 3 | `variables.executor → system/async/executors/ScheduledExecutor.cfc` | 1 | high | declares scheduleWithFixedDelay(), scheduleAtFixedRate(), schedule() | `system/async/tasks/ScheduledTask.cfc:852` variable 'variables.executor' has no component ref |
| 3 | `variables.executors` | 0 | none |  | `system/async/AsyncManager.cfc:215` variable 'variables.executors' has no component ref |
| 3 | `variables.libraries` | 0 | none |  | `build/Build.cfc:108` variable 'variables.libraries' has no component ref |
| 3 | `variables.native → system/async/executors/ScheduledExecutor.cfc` | 1 | medium | declares schedule() | `system/async/executors/ScheduledExecutor.cfc:57` variable 'variables.native' has no component ref |
| 2 | `_routes[]` | 0 | none |  | `system/web/services/RoutingService.cfc:511` variable '_routes[]' has no component ref |
| 2 | `acceptheader` | 0 | none |  | `system/web/services/RoutingService.cfc:762` variable 'acceptHeader' has no component ref |
| 2 | `application.cbcontroller → system/FrameworkSupertype.cfc` | 3 | low | declares setSetting() (2 candidates) | `tests/specs/remote/coldboxproxytest.cfc:42` variable 'application.cbController' has no component ref |
| 2 | `application[] → system/logging/Logger.cfc` | 2 | medium | declares info(); named like the receiver 'Logger' (2 candidates) | `system/Bootstrap.cfc:134` variable 'application[]' has no component ref |
| 2 | `application[] → system/web/services/LoaderService.cfc` | 1 | high | declares loadApplication(); named like the receiver 'LoaderService' | `system/Bootstrap.cfc:103` variable 'application[]' has no component ref |
| 2 | `application[] → system/web/services/ModuleService.cfc` | 1 | high | declares loadMappings(); named like the receiver 'ModuleService' | `system/Bootstrap.cfc:163` variable 'application[]' has no component ref |
| 2 | `arguments.appscope.cbbootstrap → tests/perf-harness/be-app/Application.cfc` | 6 | low | declares onApplicationEnd() (5 candidates) | `tests/perf-harness/be-app/Application.cfc:65` variable 'arguments.appScope.cbBootstrap' has no component ref |
| 2 | `arguments.appscope.cbbootstrap → tests/perf-harness/stable-app/Application.cfc` | 6 | low | declares onApplicationEnd() (5 candidates) | `tests/perf-harness/stable-app/Application.cfc:64` variable 'arguments.appScope.cbBootstrap' has no component ref |
| 2 | `arguments.appscope.cbbootstrap → tests/suites/loadtests/beload/Application.cfc` | 6 | low | declares onApplicationEnd() (5 candidates) | `tests/suites/loadtests/beload/Application.cfc:41` variable 'arguments.appScope.cbBootstrap' has no component ref |
| 2 | `arguments.childinstance → system/ioc/IInjector.cfc` | 15 | low | declares shutdown() (15 candidates) | `system/ioc/Injector.cfc:435` variable 'arguments.childInstance' has no component ref |
| 2 | `arguments.future → system/async/tasks/Future.cfc` | 5 | medium | declares getNative(); named like the receiver 'future' (5 candidates) | `system/async/tasks/Future.cfc:754` variable 'arguments.future' has no component ref |
| 2 | `arguments.invocation → system/aop/MethodInvocation.cfc` | 1 | medium | declares proceed() | `tests/specs/ioc/aop/MethodInvocationTest.cfc:102` variable 'arguments.invocation' has no component ref |
| 2 | `arguments.logboxconfig → system/logging/config/LogBoxConfig.cfc` | 7 | medium | declares reset(), init(), getMemento(); named like the receiver 'logBoxConfig' (4 candidates) | `system/web/config/ApplicationLoader.cfc:844` variable 'arguments.logBoxConfig' has no component ref |
| 2 | `arguments.mediatype` | 0 | none |  | `system/web/routing/Router.cfc:253` variable 'arguments.mediaType' has no component ref |
| 2 | `arguments.metadata.functions` | 0 | none |  | `system/ioc/config/Mapping.cfc:1218` variable 'arguments.metadata.functions' has no component ref |
| 2 | `arguments.metadata.properties` | 0 | none |  | `system/ioc/config/Mapping.cfc:1205` variable 'arguments.metadata.properties' has no component ref |
| 2 | `arguments.oconfig` | 1 | none |  | `system/web/config/ApplicationLoader.cfc:800` variable 'arguments.oConfig' has no component ref |
| 2 | `arguments.oeventhandler → system/EventHandler.cfc` | 2 | high | declares _actionExists(), _privateInvoker(); named like the receiver 'oEventHandler' | `system/web/services/HandlerService.cfc:826` variable 'arguments.oEventHandler' has no component ref |
| 2 | `arguments.result → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/web/Controller.cfc:479` variable 'arguments.result' has no component ref |
| 2 | `arguments.routeresults.route → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/services/RoutingService.cfc:709` variable 'arguments.routeResults.route' has no component ref |
| 2 | `arguments.schedule → system/async/executors/Executor.cfc` | 1 | high | declares shutdownNow(), shutdownAndAwaitTermination() | `system/async/AsyncManager.cfc:287` variable 'arguments.schedule' has no component ref |
| 2 | `arguments.scheduler → system/async/tasks/Scheduler.cfc` | 1 | high | declares getInetHost(), getLocalIp(); named like the receiver 'scheduler' | `system/async/tasks/ScheduledTask.cfc:270` variable 'arguments.scheduler' has no component ref |
| 2 | `arguments.targetobject` | 1 | none |  | `system/ioc/Injector.cfc:1225` variable 'arguments.targetObject' has no component ref |
| 2 | `arguments.thisroute.pattern` | 0 | none |  | `system/web/routing/Router.cfc:2518` variable 'arguments.thisRoute.pattern' has no component ref |
| 2 | `arguments.viewhelperpath` | 0 | none |  | `system/web/RendererEncapsulator.cfm:48` variable 'arguments.viewHelperPath' has no component ref |
| 2 | `arguments.viewhelperpath → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/web/RendererEncapsulator.cfm:49` variable 'arguments.viewHelperPath' has no component ref |
| 2 | `backpath` | 0 | none |  | `tests/index.cfm:42` variable 'backPath' has no component ref |
| 2 | `cacheconfig → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/cache/providers/CFProvider.cfc:546` variable 'cacheConfig' has no component ref |
| 2 | `cacheprovider → system/cache/AbstractCacheBoxProvider.cfc` | 16 | low | declares getName(), isReportingEnabled() (2 candidates) | `system/cache/report/skins/default/CacheReport.cfm:10` variable 'cacheProvider' has no component ref |
| 2 | `e → system/web/context/ExceptionBean.cfc` | 2 | high | declares getType(), getMessage() | `tests/specs/integration/EventExecutionsSpec.cfc:59` variable 'e' has no component ref |
| 2 | `e.tagcontext` | 0 | none |  | `system/ioc/Builder.cfc:276` variable 'e.tagContext' has no component ref |
| 2 | `executor → system/async/executors/Executor.cfc` | 16 | medium | declares getName(); named like the receiver 'executor' (16 candidates) | `test-harness/modules_app/resourcesTest/handlers/Home.cfc:10` variable 'executor' has no component ref |
| 2 | `firstmetadata → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/modules/HTMLHelper/models/HTMLHelper.cfc:428` variable 'firstMetadata' has no component ref |
| 2 | `idata → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/web/Controller.cfc:1045` variable 'iData' has no component ref |
| 2 | `interceptorentries` | 0 | none |  | `system/web/context/InterceptorState.cfc:487` variable 'interceptorEntries' has no component ref |
| 2 | `interceptorentry` | 0 | none |  | `system/web/context/InterceptorState.cfc:489` variable 'interceptorEntry' has no component ref |
| 2 | `item → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/core/dynamic/ObjectPopulator.cfc:653` variable 'item' has no component ref |
| 2 | `loc.parent → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/core/util/Util.cfc:484` variable 'loc.parent' has no component ref |
| 2 | `local.results.ehbean → system/web/context/EventHandlerBean.cfc` | 1 | medium | declares getActionMetadata() | `system/web/Controller.cfc:744` variable 'local.results.ehBean' has no component ref |
| 2 | `mapping` | 0 | none |  | `system/modules/HTMLHelper/models/HTMLHelper.cfc:2523` variable 'mapping' has no component ref |
| 2 | `mconfig.settings → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/web/services/ModuleService.cfc:1347` variable 'mConfig.settings' has no component ref |
| 2 | `md.properties` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:650` variable 'md.properties' has no component ref |
| 2 | `my.headdata` | 0 | none |  | `system/remote/RemotingUtil.cfc:41` variable 'my.headData' has no component ref |
| 2 | `my.method` | 0 | none |  | `system/remote/RemotingUtil.cfc:32` variable 'my.method' has no component ref |
| 2 | `mystruct → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/cache/util/EventURLFacade.cfc:126` variable 'myStruct' has no component ref |
| 2 | `omodel → test-harness/remote/Echo.cfc` | 3 | low | declares echo() (3 candidates) | `tests/specs/integration/ModuleSpec.cfc:154` variable 'oModel' has no component ref |
| 2 | `optionals` | 0 | none |  | `system/web/routing/Router.cfc:2531` variable 'optionals' has no component ref |
| 2 | `path` | 0 | none |  | `build/Build.cfc:485` variable 'path' has no component ref |
| 2 | `results.route.patternparams` | 0 | none |  | `system/web/services/RoutingService.cfc:582` variable 'results.route.patternParams' has no component ref |
| 2 | `router → system/web/routing/Router.cfc` | 3 | high | declares route(), getRoutes(); named like the receiver 'router' | `tests/specs/web/routing/RouterSSETest.cfc:180` variable 'router' has no component ref |
| 2 | `routeresults.params → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/services/RoutingService.cfc:228` variable 'routeResults.params' has no component ref |
| 2 | `sentevents` | 0 | none |  | `system/testing/CustomMatchers.cfc:171` variable 'sentEvents' has no component ref |
| 2 | `services.requestservice → system/web/context/RequestContext.cfc` | 2 | high | declares getCurrentEvent(), renderdata() | `system/web/Controller.cfc:704` variable 'services.requestService' has no component ref |
| 2 | `services.schedulerservice → system/web/services/SchedulerService.cfc` | 1 | high | declares loadGlobalScheduler(), startupSchedulers(); named like the receiver 'schedulerService' | `system/web/services/LoaderService.cfc:101` variable 'services.schedulerService' has no component ref |
| 2 | `showproperties` | 0 | none |  | `system/modules/HTMLHelper/models/HTMLHelper.cfc:2641` variable 'showProperties' has no component ref |
| 2 | `sorted` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:415` variable 'sorted' has no component ref |
| 2 | `target → system/cache/store/BlackHoleStore.cfc` | 24 | low | declares get() (24 candidates) | `system/cache/store/ConcurrentSoftReferenceStore.cfc:100` variable 'target' has no component ref |
| 2 | `targetobject → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares injectMixin() | `system/ioc/Injector.cfc:1139` variable 'targetObject' has no component ref |
| 2 | `task → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `test-harness/config/Scheduler.cfc:41` variable 'task' has no component ref |
| 2 | `task → system/cache/AbstractCacheBoxProvider.cfc` | 8 | low | declares getStats() (8 candidates) | `test-harness/config/Scheduler.cfc:41` variable 'task' has no component ref |
| 2 | `thisdidata → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/ioc/Injector.cfc:1336` variable 'thisDIData' has no component ref |
| 2 | `thismapping → system/ioc/config/Mapping.cfc` | 1 | medium | declares setVirtualInheritance() | `system/ioc/config/Binder.cfc:786` variable 'thisMapping' has no component ref |
| 2 | `timeunits` | 0 | none |  | `tests/specs/async/tasks/ScheduledTaskSpec.cfc:102` variable 'timeUnits' has no component ref |
| 2 | `variables.actionmetadata → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/context/EventHandlerBean.cfc:115` variable 'variables.actionMetadata' has no component ref |
| 2 | `variables.cache` | 0 | none |  | `system/cache/providers/MockProvider.cfc:106` variable 'variables.cache' has no component ref |
| 2 | `variables.configsettings` | 0 | none |  | `system/web/Controller.cfc:284` variable 'variables.configSettings' has no component ref |
| 2 | `variables.configuration → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/cache/providers/BoxLangProvider.cfc:164` variable 'variables.configuration' has no component ref |
| 2 | `variables.data → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `test-harness/models/delegates/Disk.cfc:19` variable 'variables.data' has no component ref |
| 2 | `variables.elementcleaner → system/cache/util/ElementCleaner.cfc` | 2 | medium | declares clearByKeySnippet(); named like the receiver 'elementCleaner' (2 candidates) | `system/cache/AbstractCacheBoxProvider.cfc:663` variable 'variables.elementCleaner' has no component ref |
| 2 | `variables.elixirmanifests → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/modules/HTMLHelper/models/HTMLHelper.cfc:2528` variable 'variables.elixirManifests' has no component ref |
| 2 | `variables.eventstateindex → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/core/events/EventPoolManager.cfc:177` variable 'variables.eventStateIndex' has no component ref |
| 2 | `variables.executors → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/async/AsyncManager.cfc:94` variable 'variables.executors' has no component ref |
| 2 | `variables.native → system/async/tasks/ScheduledFuture.cfc` | 1 | medium | declares isPeriodic() | `system/async/tasks/ScheduledFuture.cfc:28` variable 'variables.native' has no component ref |
| 2 | `variables.starttime` | 0 | none |  | `system/async/tasks/ScheduledTask.cfc:1636` variable 'variables.startTime' has no component ref |
| 2 | `variables.stats.lastresult` | 0 | none |  | `system/async/tasks/ScheduledTask.cfc:714` variable 'variables.stats.lastResult' has no component ref |
| 2 | `variables.system.err` | 1 | none |  | `system/async/AsyncManager.cfc:457` variable 'variables.System.err' has no component ref |
| 2 | `variables.system.out` | 1 | none |  | `system/async/AsyncManager.cfc:447` variable 'variables.System.out' has no component ref |
| 2 | `variables.thisroute → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/routing/Router.cfc:1258` variable 'variables.thisRoute' has no component ref |
| 2 | `variables.withclosure` | 0 | none |  | `system/web/routing/Router.cfc:481` variable 'variables.withClosure' has no component ref |
| 2 | `vd.scenarios → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/perf-harness/PerformanceSuite.cfc:529` variable 'vd.scenarios' has no component ref |
| 1 | `$parent → test-harness/models/delegates/Computer.cfc` | 1 | medium | declares getOutput() | `test-harness/models/delegates/Worker.cfc:21` variable '$parent' has no component ref |
| 1 | `_routes[].domain` | 0 | none |  | `system/web/services/RoutingService.cfc:524` variable '_routes[].domain' has no component ref |
| 1 | `actions → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/routing/Router.cfc:863` variable 'actions' has no component ref |
| 1 | `aggregator` | 0 | none |  | `system/aop/Mixer.cfc:169` variable 'aggregator' has no component ref |
| 1 | `application.cbbootstrap → system/remote/ColdboxProxy.cfc` | 3 | low | declares getCOLDBOX_APP_KEY() (2 candidates) | `system/remote/ColdboxProxy.cfc:32` variable 'application.cbBootstrap' has no component ref |
| 1 | `application.wirebox → system/aop/Mixer.cfc` | 2 | low | declares getBinder() (2 candidates) | `test-harness/index.cfm:11` variable 'application.wirebox' has no component ref |
| 1 | `application.wirebox → system/ioc/config/Binder.cfc` | 2 | medium | declares getMappings(); named like the receiver 'Binder' (2 candidates) | `test-harness/index.cfm:11` variable 'application.wirebox' has no component ref |
| 1 | `application[] → system/logging/LogBox.cfc` | 9 | medium | declares getLogger(); named like the receiver 'LogBox' (8 candidates) | `system/Bootstrap.cfc:133` variable 'application[]' has no component ref |
| 1 | `application[] → system/web/services/InterceptorService.cfc` | 6 | medium | declares announce(); named like the receiver 'InterceptorService' (5 candidates) | `system/Bootstrap.cfc:165` variable 'application[]' has no component ref |
| 1 | `appmetadata.customtagpaths` | 0 | none |  | `system/core/util/CFMappingHelper.cfc:23` variable 'appMetadata.customTagPaths' has no component ref |
| 1 | `appscope.cbbootstrap → tests/suites/eventCachingCollisions/Application.cfc` | 10 | low | declares onSessionEnd() (9 candidates) | `tests/suites/eventCachingCollisions/Application.cfc:91` variable 'appScope.cbBootstrap' has no component ref |
| 1 | `appsettings.mappings → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/core/util/LuceeMappingHelper.cfc:46` variable 'appSettings.mappings' has no component ref |
| 1 | `arguments.appender → system/logging/AbstractAppender.cfc` | 15 | low | declares shutdown() (15 candidates) | `system/logging/LogBox.cfc:216` variable 'arguments.appender' has no component ref |
| 1 | `arguments.appenders` | 0 | none |  | `system/logging/LogBox.cfc:429` variable 'arguments.appenders' has no component ref |
| 1 | `arguments.appscope.cbbootstrap → test-harness/Application.cfc` | 10 | low | declares onSessionEnd() (9 candidates) | `test-harness/Application.cfc:107` variable 'arguments.appScope.cbBootStrap' has no component ref |
| 1 | `arguments.arglist` | 0 | none |  | `system/core/dynamic/MixerUtil.cfc:198` variable 'arguments.argList' has no component ref |
| 1 | `arguments.args` | 0 | none |  | `system/cache/util/EventURLFacade.cfc:106` variable 'arguments.args' has no component ref |
| 1 | `arguments.asset` | 0 | none |  | `system/modules/HTMLHelper/models/HTMLHelper.cfc:104` variable 'arguments.asset' has no component ref |
| 1 | `arguments.asyncmanager → system/async/AsyncManager.cfc` | 5 | medium | declares out(); named like the receiver 'asyncManager' (5 candidates) | `system/async/tasks/Scheduler.cfc:102` variable 'arguments.asyncManager' has no component ref |
| 1 | `arguments.buffer → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `tests/specs/web/services/InterceptorserviceTest.cfc:170` variable 'arguments.buffer' has no component ref |
| 1 | `arguments.child → system/ioc/IInjector.cfc` | 2 | low | declares setParent() (2 candidates) | `system/ioc/Injector.cfc:278` variable 'arguments.child' has no component ref |
| 1 | `arguments.content` | 0 | none |  | `system/web/context/SSEEmitter.cfc:317` variable 'arguments.content' has no component ref |
| 1 | `arguments.data → test-harness/models/RenderConvention.cfc` | 1 | medium | declares $renderdata() | `system/core/conversion/DataMarshaller.cfc:60` variable 'arguments.data' has no component ref |
| 1 | `arguments.delegate → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares injectPropertyMixin() | `system/ioc/Injector.cfc:1415` variable 'arguments.delegate' has no component ref |
| 1 | `arguments.dictionary[].aspects → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/aop/Mixer.cfc:264` variable 'arguments.dictionary[].aspects' has no component ref |
| 1 | `arguments.eventdictionary` | 0 | none |  | `system/cache/util/EventURLFacade.cfc:60` variable 'arguments.eventDictionary' has no component ref |
| 1 | `arguments.eventdictionary.cacheexclude` | 0 | none |  | `system/cache/util/EventURLFacade.cfc:76` variable 'arguments.eventDictionary.cacheExclude' has no component ref |
| 1 | `arguments.eventdictionary.cacheinclude` | 0 | none |  | `system/cache/util/EventURLFacade.cfc:67` variable 'arguments.eventDictionary.cacheInclude' has no component ref |
| 1 | `arguments.exception.tagcontext → system/ioc/config/Binder.cfc` | 1 | medium | declares map() | `system/RestHandler.cfc:569` variable 'arguments.exception.tagContext' has no component ref |
| 1 | `arguments.exclude` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:342` variable 'arguments.exclude' has no component ref |
| 1 | `arguments.expression` | 0 | none |  | `system/ioc/config/Mapping.cfc:804` variable 'arguments.expression' has no component ref |
| 1 | `arguments.include` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:339` variable 'arguments.include' has no component ref |
| 1 | `arguments.jfuture → system/async/tasks/Future.cfc` | 24 | low | declares get() (24 candidates) | `system/async/tasks/Future.cfc:792` variable 'arguments.jFuture' has no component ref |
| 1 | `arguments.md.extends → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/core/util/Util.cfc:432` variable 'arguments.md.extends' has no component ref |
| 1 | `arguments.metadata.extends → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/core/events/EventPoolManager.cfc:251` variable 'arguments.metadata.extends' has no component ref |
| 1 | `arguments.ocontext → system/web/context/ExceptionBean.cfc` | 18 | low | declares getMemento() (16 candidates) | `system/web/context/RequestContextDecorator.cfc:21` variable 'arguments.oContext' has no component ref |
| 1 | `arguments.parentinjector → system/ioc/Injector.cfc` | 1 | medium | declares registerChildInjector() | `system/web/services/ModuleService.cfc:1254` variable 'arguments.parentInjector' has no component ref |
| 1 | `arguments.prc.exception → system/web/context/ExceptionBean.cfc` | 1 | high | declares getExceptionStruct(); named like the receiver 'exception' | `system/RestHandler.cfc:191` variable 'arguments.prc.exception' has no component ref |
| 1 | `arguments.record.task → system/async/tasks/ScheduledTask.cfc` | 8 | low | declares getStats() (8 candidates) | `system/async/tasks/Scheduler.cfc:417` variable 'arguments.record.task' has no component ref |
| 1 | `arguments.requestcontext → system/web/context/RequestContext.cfc` | 2 | high | declares overrideEvent(); named like the receiver 'requestContext' | `system/web/services/HandlerService.cfc:725` variable 'arguments.requestContext' has no component ref |
| 1 | `arguments.result.value → system/aop/Matcher.cfc` | 18 | low | declares getMemento() (16 candidates) | `tests/specs/async/AsyncManagerSpec.cfc:373` variable 'arguments.result.value' has no component ref |
| 1 | `arguments.scope[] → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/web/context/ExceptionBean.cfc:483` variable 'arguments.scope[]' has no component ref |
| 1 | `arguments.target → system/EventHandler.cfc` | 2 | medium | declares _privateInvoker() | `system/web/Controller.cfc:1268` variable 'arguments.target' has no component ref |
| 1 | `arguments.target.population.exclude` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:343` variable 'arguments.target.population.exclude' has no component ref |
| 1 | `arguments.target.population.include` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:340` variable 'arguments.target.population.include' has no component ref |
| 1 | `arguments.thisexecutor → system/async/executors/Executor.cfc` | 8 | low | declares getStats() (8 candidates) | `system/async/AsyncManager.cfc:309` variable 'arguments.thisExecutor' has no component ref |
| 1 | `arguments.thisfuture → system/async/tasks/Future.cfc` | 24 | low | declares get() (24 candidates) | `system/async/tasks/Future.cfc:890` variable 'arguments.thisFuture' has no component ref |
| 1 | `arguments.thisitem` | 0 | none |  | `system/async/time/Duration.cfc:117` variable 'arguments.thisItem' has no component ref |
| 1 | `arguments.thismapping → system/ioc/config/Binder.cfc` | 11 | low | declares process() (11 candidates) | `system/ioc/config/Binder.cfc:550` variable 'arguments.thisMapping' has no component ref |
| 1 | `aroute` | 0 | none |  | `system/web/services/RoutingService.cfc:830` variable 'aRoute' has no component ref |
| 1 | `aspectbindings[].classes → system/aop/Matcher.cfc` | 1 | medium | declares matchClass() | `system/aop/Mixer.cfc:156` variable 'aspectBindings[].classes' has no component ref |
| 1 | `aspectlist` | 0 | none |  | `system/aop/Mixer.cfc:167` variable 'aspectList' has no component ref |
| 1 | `baseproperties → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/ioc/Builder.cfc:1119` variable 'baseProperties' has no component ref |
| 1 | `buffer → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/core/conversion/XMLConverter.cfc:264` variable 'buffer' has no component ref |
| 1 | `cache → system/cache/AbstractCacheBoxProvider.cfc` | 15 | low | declares shutdown() (15 candidates) | `system/cache/CacheFactory.cfc:527` variable 'cache' has no component ref |
| 1 | `cacheconfig[] → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/cache/report/skins/default/CacheReport.cfm:72` variable 'cacheConfig[]' has no component ref |
| 1 | `cachedobjectmd → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/cache/providers/CacheBoxProvider.cfc:589` variable 'cachedObjectMD' has no component ref |
| 1 | `cachesession` | 0 | none |  | `system/cache/providers/CFProvider.cfc:499` variable 'cacheSession' has no component ref |
| 1 | `cgi` | 0 | none |  | `test-harness/config/Scheduler.cfc:18` variable 'cgi' has no component ref |
| 1 | `cgi → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/context/RequestContext.cfc:915` variable 'CGI' has no component ref |
| 1 | `childinjector → system/ioc/IInjector.cfc` | 9 | low | declares getInstance() (7 candidates) | `system/ioc/Injector.cfc:499` variable 'childInjector' has no component ref |
| 1 | `config → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/cache/providers/CFProvider.cfc:51` variable 'config' has no component ref |
| 1 | `contextroute.route → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/web/services/RoutingService.cfc:597` variable 'contextRoute.route' has no component ref |
| 1 | `currentmetadata.extends → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/web/services/InterceptorService.cfc:709` variable 'currentMetadata.extends' has no component ref |
| 1 | `currentroutedurl` | 0 | none |  | `system/web/context/RequestContext.cfc:800` variable 'currentRoutedURL' has no component ref |
| 1 | `d → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `test-harness/models/RenderConvention.cfc:26` variable 'd' has no component ref |
| 1 | `data.unit` | 0 | none |  | `tests/specs/async/time/DateTimeHelperSpec.cfc:74` variable 'data.unit' has no component ref |
| 1 | `daysoftheweek` | 0 | none |  | `tests/specs/async/tasks/ScheduledTaskSpec.cfc:225` variable 'daysOfTheWeek' has no component ref |
| 1 | `delegateincludes` | 0 | none |  | `system/ioc/Injector.cfc:1445` variable 'delegateIncludes' has no component ref |
| 1 | `discoveredlayout` | 0 | none |  | `system/web/context/RequestContext.cfc:998` variable 'discoveredLayout' has no component ref |
| 1 | `e → system/logging/LogEvent.cfc` | 2 | low | declares getMessage() (2 candidates) | `tests/specs/ioc/InjectorTest.cfc:120` variable 'e' has no component ref |
| 1 | `e.message` | 0 | none |  | `tests/specs/web/context/RequestContextSSETest.cfc:296` variable 'e.message' has no component ref |
| 1 | `e.tagcontext → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/core/dynamic/ObjectPopulator.cfc:508` variable 'e.tagContext' has no component ref |
| 1 | `ehbean → system/web/context/EventHandlerBean.cfc` | 1 | medium | declares getFullEvent() | `system/web/services/HandlerService.cfc:465` variable 'ehBean' has no component ref |
| 1 | `eng.versions.be.appbootstrap → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/perf-harness/PerformanceSuite.cfc:596` variable 'eng.versions.be.appBootstrap' has no component ref |
| 1 | `eng.versions.be.coldstart → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:598` variable 'eng.versions.be.coldStart' has no component ref |
| 1 | `eng.versions.be.scenarios → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/perf-harness/PerformanceSuite.cfc:616` variable 'eng.versions.be.scenarios' has no component ref |
| 1 | `eng.versions.be.throughput → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:600` variable 'eng.versions.be.throughput' has no component ref |
| 1 | `eng.versions.stable.appbootstrap → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/perf-harness/PerformanceSuite.cfc:597` variable 'eng.versions.stable.appBootstrap' has no component ref |
| 1 | `eng.versions.stable.coldstart → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:599` variable 'eng.versions.stable.coldStart' has no component ref |
| 1 | `eng.versions.stable.scenarios → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/perf-harness/PerformanceSuite.cfc:617` variable 'eng.versions.stable.scenarios' has no component ref |
| 1 | `eng.versions.stable.throughput → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:601` variable 'eng.versions.stable.throughput' has no component ref |
| 1 | `eventcache` | 3 | none |  | `system/web/services/RequestService.cfc:167` variable 'eventCache' has no component ref |
| 1 | `eventcache → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/web/services/RequestService.cfc:189` variable 'eventCache' has no component ref |
| 1 | `ex → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `tests/specs/async/AsyncManagerSpec.cfc:129` variable 'ex' has no component ref |
| 1 | `exception.stacktrace` | 0 | none |  | `test-harness/config/Scheduler.cfc:27` variable 'exception.stacktrace' has no component ref |
| 1 | `excludekeys` | 0 | none |  | `system/cache/util/EventURLFacade.cfc:78` variable 'excludeKeys' has no component ref |
| 1 | `expectation.actual → system/testing/mock/web/MockSSEEmitter.cfc` | 1 | medium | declares getSentEvents() | `system/testing/CustomMatchers.cfc:170` variable 'expectation.actual' has no component ref |
| 1 | `expectation.actual → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/testing/CustomMatchers.cfc:71` variable 'expectation.actual' has no component ref |
| 1 | `explicitview.view` | 0 | none |  | `system/web/Renderer.cfc:223` variable 'explicitView.view' has no component ref |
| 1 | `file.template` | 0 | none |  | `system/ioc/Builder.cfc:279` variable 'file.template' has no component ref |
| 1 | `form` | 0 | none |  | `test-harness/config/Scheduler.cfc:20` variable 'form' has no component ref |
| 1 | `includekeys` | 0 | none |  | `system/cache/util/EventURLFacade.cfc:69` variable 'includeKeys' has no component ref |
| 1 | `indentedkeywords` | 0 | none |  | `system/core/delegates/StringUtil.cfc:69` variable 'indentedKeywords' has no component ref |
| 1 | `initialnextrun → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/async/tasks/ScheduledTask.cfc:1577` variable 'initialNextRun' has no component ref |
| 1 | `inputs → system/ioc/config/Binder.cfc` | 1 | medium | declares map() | `system/web/routing/Router.cfc:2280` variable 'inputs' has no component ref |
| 1 | `interceptionstate → system/web/context/InterceptorState.cfc` | 11 | low | declares process() (11 candidates) | `system/web/services/InterceptorService.cfc:222` variable 'interceptionState' has no component ref |
| 1 | `interceptorchain` | 0 | none |  | `system/web/context/InterceptorState.cfc:354` variable 'interceptorChain' has no component ref |
| 1 | `item → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/cache/CacheFactory.cfc:273` variable 'item' has no component ref |
| 1 | `item.module` | 0 | none |  | `system/web/services/ModuleService.cfc:855` variable 'item.module' has no component ref |
| 1 | `item.name` | 0 | none |  | `system/web/services/ModuleService.cfc:1512` variable 'item.name' has no component ref |
| 1 | `itemarray[] → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/specs/core/util/UtilTest.cfc:134` variable 'itemArray[]' has no component ref |
| 1 | `itemarray[].annotations → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/specs/core/util/UtilTest.cfc:138` variable 'itemArray[].annotations' has no component ref |
| 1 | `itemarray[].documentation → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/specs/core/util/UtilTest.cfc:142` variable 'itemArray[].documentation' has no component ref |
| 1 | `jfutures → system/ioc/config/Binder.cfc` | 1 | medium | declares map() | `system/async/tasks/Future.cfc:791` variable 'jFutures' has no component ref |
| 1 | `key` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:265` variable 'key' has no component ref |
| 1 | `keywords` | 0 | none |  | `system/core/delegates/StringUtil.cfc:61` variable 'keywords' has no component ref |
| 1 | `librecord.sources` | 0 | none |  | `build/Build.cfc:275` variable 'libRecord.sources' has no component ref |
| 1 | `listenerchain` | 0 | none |  | `system/core/events/EventPool.cfc:96` variable 'listenerChain' has no component ref |
| 1 | `loc.values → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/modules/HTMLHelper/models/HTMLHelper.cfc:2209` variable 'loc.values' has no component ref |
| 1 | `local.docresults.filecontent → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `build/Build.cfc:413` variable 'local.docResults.filecontent' has no component ref |
| 1 | `local.idata → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/web/Renderer.cfc:343` variable 'local.iData' has no component ref |
| 1 | `local.refresults.eventcaching.responseheaders` | 0 | none |  | `system/Bootstrap.cfc:265` variable 'local.refResults.eventCaching.responseHeaders' has no component ref |
| 1 | `local.results → system/web/routing/Router.cfc` | 6 | low | declares delete() (6 candidates) | `system/web/Controller.cfc:737` variable 'local.results' has no component ref |
| 1 | `local.results.data → test-harness/models/RenderConvention.cfc` | 1 | medium | declares $renderdata() | `system/web/Controller.cfc:733` variable 'local.results.data' has no component ref |
| 1 | `local.thisarg → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/ioc/Builder.cfc:432` variable 'local.thisArg' has no component ref |
| 1 | `logevent → system/logging/LogEvent.cfc` | 1 | high | declares getTimestamp(); named like the receiver 'logevent' | `tests/specs/logging/MockLayout.cfc:4` variable 'logevent' has no component ref |
| 1 | `mappings` | 0 | none |  | `system/ioc/config/Binder.cfc:1443` variable 'mappings' has no component ref |
| 1 | `mappings[] → tests/specs/core/collections/ScopeStorageTest.cfc` | 5 | low | declares getScope() (5 candidates) | `tests/specs/ioc/config/BinderTest.cfc:543` variable 'mappings[]' has no component ref |
| 1 | `matchingroutes → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/web/routing/Router.cfc:846` variable 'matchingRoutes' has no component ref |
| 1 | `mconfig.applicationhelper → system/ioc/config/Binder.cfc` | 1 | medium | declares map() | `system/web/services/ModuleService.cfc:866` variable 'mConfig.applicationHelper' has no component ref |
| 1 | `mconfig.conventions.routerlocation` | 0 | none |  | `system/web/services/ModuleService.cfc:500` variable 'mConfig.conventions.routerLocation' has no component ref |
| 1 | `mconfig.conventions.schedulerlocation` | 0 | none |  | `system/web/services/ModuleService.cfc:503` variable 'mConfig.conventions.schedulerLocation' has no component ref |
| 1 | `mconfig.entrypoint` | 0 | none |  | `system/web/services/ModuleService.cfc:789` variable 'mConfig.entryPoint' has no component ref |
| 1 | `mconfig.injector → system/ioc/config/Binder.cfc` | 1 | high | declares map(); named like the receiver 'Binder' | `system/web/services/ModuleService.cfc:757` variable 'mConfig.injector' has no component ref |
| 1 | `mconfig.injector → system/web/routing/Router.cfc` | 2 | low | declares to() (2 candidates) | `system/web/services/ModuleService.cfc:761` variable 'mConfig.injector' has no component ref |
| 1 | `mconfig.router` | 0 | none |  | `system/web/services/ModuleService.cfc:845` variable 'mConfig.router' has no component ref |
| 1 | `mconfig.scheduler → system/async/tasks/Scheduler.cfc` | 16 | medium | declares getName(); named like the receiver 'scheduler' (16 candidates) | `system/web/services/ModuleService.cfc:1051` variable 'mConfig.scheduler' has no component ref |
| 1 | `mediatypeswithnoalias` | 0 | none |  | `tests/specs/web/routing/RouterSSETest.cfc:100` variable 'mediaTypesWithNoAlias' has no component ref |
| 1 | `message` | 0 | none |  | `system/async/tasks/ScheduledTask.cfc:1683` variable 'message' has no component ref |
| 1 | `message → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/async/tasks/ScheduledTask.cfc:1680` variable 'message' has no component ref |
| 1 | `methods` | 0 | none |  | `tests/specs/web/routing/RouterTest.cfc:249` variable 'methods' has no component ref |
| 1 | `mixer.afterinstanceautowire` | 0 | none |  | `tests/specs/integration/WireBoxSpec.cfc:14` variable 'mixer.afterInstanceAutowire' has no component ref |
| 1 | `mixer.afterinstanceautowire → system/cache/store/ConcurrentStore.cfc` | 2 | low | declares getPool() (2 candidates) | `tests/specs/integration/WireBoxSpec.cfc:14` variable 'mixer.afterInstanceAutowire' has no component ref |
| 1 | `modlocations` | 0 | none |  | `system/web/services/ModuleService.cfc:144` variable 'modLocations' has no component ref |
| 1 | `modules` | 0 | none |  | `system/web/services/ModuleService.cfc:636` variable 'modules' has no component ref |
| 1 | `modulesettings[] → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/context/RequestContext.cfc:1791` variable 'moduleSettings[]' has no component ref |
| 1 | `modulesettings[].settings` | 0 | none |  | `system/ioc/dsl/ColdBoxDSL.cfc:201` variable 'moduleSettings[].settings' has no component ref |
| 1 | `modulesettings[].sse → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/context/RequestContext.cfc:1793` variable 'moduleSettings[].sse' has no component ref |
| 1 | `oappender → system/logging/AbstractAppender.cfc` | 2 | low | declares onUnRegistration() (2 candidates) | `system/logging/Logger.cfc:179` variable 'oAppender' has no component ref |
| 1 | `oeventhandler` | 0 | none |  | `system/web/services/HandlerService.cfc:818` variable 'oEventHandler' has no component ref |
| 1 | `ointerceptor → system/Interceptor.cfc` | 92 | medium | declares configure(); named like the receiver 'oInterceptor' (92 candidates) | `system/web/services/InterceptorService.cfc:385` variable 'oInterceptor' has no component ref |
| 1 | `options → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/cache/store/JDBCStore.cfc:481` variable 'options' has no component ref |
| 1 | `oscheduler → system/logging/LogEvent.cfc` | 3 | low | declares setCategory() (3 candidates) | `system/web/services/SchedulerService.cfc:117` variable 'oScheduler' has no component ref |
| 1 | `prc → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `tests/perf-harness/app/views/main/index.cfm:6` variable 'prc' has no component ref |
| 1 | `prc.products` | 0 | none |  | `tests/perf-harness/app/views/main/complex.cfm:13` variable 'prc.products' has no component ref |
| 1 | `prc.users` | 0 | none |  | `tests/perf-harness/app/views/main/complex.cfm:4` variable 'prc.users' has no component ref |
| 1 | `preexistingextensions` | 0 | none |  | `tests/specs/web/routing/RouterSSETest.cfc:69` variable 'preExistingExtensions' has no component ref |
| 1 | `previous` | 0 | none |  | `system/web/services/RoutingService.cfc:767` variable 'previous' has no component ref |
| 1 | `propertyvalue` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:443` variable 'propertyValue' has no component ref |
| 1 | `propertyvalue → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/core/dynamic/ObjectPopulator.cfc:443` variable 'propertyValue' has no component ref |
| 1 | `providertest.coolpizza → system/ioc/IProvider.cfc` | 2 | low | declares $get() (2 candidates) | `tests/specs/ioc/InjectorCreationTest.cfc:118` variable 'providerTest.coolPizza' has no component ref |
| 1 | `querystring` | 0 | none |  | `system/testing/BaseTestCase.cfc:919` variable 'queryString' has no component ref |
| 1 | `r.pattern` | 0 | none |  | `tests/specs/web/routing/RouterSSETest.cfc:166` variable 'r.pattern' has no component ref |
| 1 | `rc → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/routing/Router.cfc:2405` variable 'rc' has no component ref |
| 1 | `redirectto` | 0 | none |  | `system/web/services/RoutingService.cfc:705` variable 'redirectTo' has no component ref |
| 1 | `resourcespool → test-harness/models/RenderConvention.cfc` | 16 | low | declares getName() (16 candidates) | `test-harness/modules_app/resourcesTest/handlers/Home.cfc:11` variable 'resourcesPool' has no component ref |
| 1 | `result → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/web/context/RequestContext.cfc:1428` variable 'result' has no component ref |
| 1 | `result.rows → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/ioc/Builder.cfc:282` variable 'result.rows' has no component ref |
| 1 | `result[] → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/core/dynamic/ObjectPopulator.cfc:663` variable 'result[]' has no component ref |
| 1 | `results.route` | 1 | none |  | `system/web/services/RoutingService.cfc:559` variable 'results.route' has no component ref |
| 1 | `results.route → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/web/services/RoutingService.cfc:549` variable 'results.route' has no component ref |
| 1 | `results.route.domainparams` | 0 | none |  | `system/web/services/RoutingService.cfc:649` variable 'results.route.domainParams' has no component ref |
| 1 | `results.viewpath` | 0 | none |  | `system/web/Renderer.cfc:1014` variable 'results.viewPath' has no component ref |
| 1 | `routeresults.params` | 0 | none |  | `system/web/services/RoutingService.cfc:227` variable 'routeResults.params' has no component ref |
| 1 | `routeresults.route → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/web/services/RoutingService.cfc:220` variable 'routeResults.route' has no component ref |
| 1 | `routeresults.route → test-harness/handlers/rendering.cfc` | 1 | medium | declares redirect() | `system/web/services/RoutingService.cfc:695` variable 'routeResults.route' has no component ref |
| 1 | `routeresults.route.action` | 0 | none |  | `system/web/services/RoutingService.cfc:347` variable 'routeResults.route.action' has no component ref |
| 1 | `routeresults.route.event` | 0 | none |  | `system/web/services/RoutingService.cfc:269` variable 'routeResults.route.event' has no component ref |
| 1 | `routeresults.route.handler` | 0 | none |  | `system/web/services/RoutingService.cfc:290` variable 'routeResults.route.handler' has no component ref |
| 1 | `routeresults.route.headers` | 0 | none |  | `system/web/services/RoutingService.cfc:378` variable 'routeResults.route.headers' has no component ref |
| 1 | `routeresults.route.layout` | 0 | none |  | `system/web/services/RoutingService.cfc:369` variable 'routeResults.route.layout' has no component ref |
| 1 | `routeresults.route.redirect` | 0 | none |  | `system/web/services/RoutingService.cfc:251` variable 'routeResults.route.redirect' has no component ref |
| 1 | `routeresults.route.response` | 0 | none |  | `system/web/services/RoutingService.cfc:408` variable 'routeResults.route.response' has no component ref |
| 1 | `routeresults.route.view` | 0 | none |  | `system/web/services/RoutingService.cfc:358` variable 'routeResults.route.view' has no component ref |
| 1 | `routes` | 0 | none |  | `tests/specs/web/routing/RouterSSETest.cfc:166` variable 'routes' has no component ref |
| 1 | `runners` | 0 | none |  | `build/Build.cfc:214` variable 'runners' has no component ref |
| 1 | `scheduler → system/async/tasks/Scheduler.cfc` | 2 | medium | declares restart(); named like the receiver 'scheduler' (2 candidates) | `system/web/services/SchedulerService.cfc:221` variable 'scheduler' has no component ref |
| 1 | `scopeinfo → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/ioc/Provider.cfc:79` variable 'scopeInfo' has no component ref |
| 1 | `sentevents → system/ioc/config/Binder.cfc` | 1 | medium | declares map() | `system/testing/CustomMatchers.cfc:174` variable 'sentEvents' has no component ref |
| 1 | `services.handlerservice → system/web/context/EventHandlerBean.cfc` | 1 | medium | declares setIsPrivate() | `system/web/Controller.cfc:823` variable 'services.handlerService' has no component ref |
| 1 | `services.loaderservice → system/web/services/LoaderService.cfc` | 1 | high | declares createDefaultLogBox(); named like the receiver 'loaderService' | `system/web/Controller.cfc:140` variable 'services.loaderService' has no component ref |
| 1 | `services.moduleservice → system/web/services/ModuleService.cfc` | 1 | high | declares activateAllModules(); named like the receiver 'moduleService' | `system/web/services/LoaderService.cfc:91` variable 'services.moduleService' has no component ref |
| 1 | `services.requestservice → system/web/flash/AbstractFlashScope.cfc` | 4 | low | declares saveFlash() (4 candidates) | `system/web/Controller.cfc:575` variable 'services.requestService' has no component ref |
| 1 | `settings → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `tests/specs/web/context/RequestContextSSETest.cfc:36` variable 'settings' has no component ref |
| 1 | `stime → system/async/time/Duration.cfc` | 2 | low | declares plusDays() (2 candidates) | `system/async/tasks/ScheduledTask.cfc:1657` variable 'sTime' has no component ref |
| 1 | `target → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares injectMixin() | `system/ioc/Injector.cfc:1431` variable 'target' has no component ref |
| 1 | `targetmixer` | 3 | none |  | `system/cache/util/EventURLFacade.cfc:90` variable 'targetMixer' has no component ref |
| 1 | `targetmixer → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/cache/util/EventURLFacade.cfc:95` variable 'targetmixer' has no component ref |
| 1 | `targetpath` | 0 | none |  | `tests/index.cfm:126` variable 'targetPath' has no component ref |
| 1 | `targetscope → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/core/collections/ScopeStorage.cfc:82` variable 'targetScope' has no component ref |
| 1 | `targetvariables → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/ioc/Builder.cfc:1097` variable 'targetVariables' has no component ref |
| 1 | `tasklist → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/async/executors/Executor.cfc:245` variable 'taskList' has no component ref |
| 1 | `taskrecord.future → system/async/tasks/Future.cfc` | 2 | medium | declares cancel(); named like the receiver 'future' (2 candidates) | `system/async/tasks/Scheduler.cfc:468` variable 'taskRecord.future' has no component ref |
| 1 | `this.$wbdelegatemap → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/core/dynamic/MixerUtil.cfc:219` variable 'this.$wbDelegateMap' has no component ref |
| 1 | `this.$wbinjector → system/ioc/IInjector.cfc` | 9 | low | declares getInstance() (7 candidates) | `system/ioc/Builder.cfc:149` variable 'this.$wbInjector' has no component ref |
| 1 | `this.mappings → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `apidocs/Application.cfc:27` variable 'this.mappings' has no component ref |
| 1 | `thisaccept` | 0 | none |  | `system/web/services/RoutingService.cfc:781` variable 'thisAccept' has no component ref |
| 1 | `thisallowedmethods → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/web/Controller.cfc:841` variable 'thisAllowedMethods' has no component ref |
| 1 | `thisappender → system/logging/AbstractAppender.cfc` | 2 | high | declares canLog(), getProperty(), logMessage() | `system/logging/Logger.cfc:378` variable 'thisAppender' has no component ref |
| 1 | `thisfunction → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/core/events/EventPoolManager.cfc:226` variable 'thisFunction' has no component ref |
| 1 | `thisroute → system/web/routing/Router.cfc` | 3 | low | declares append() (2 candidates) | `system/web/routing/Router.cfc:817` variable 'thisRoute' has no component ref |
| 1 | `transientcache` | 0 | none |  | `system/ioc/Injector.cfc:1268` variable 'transientCache' has no component ref |
| 1 | `unit → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `tests/specs/async/time/DateTimeHelperSpec.cfc:74` variable 'unit' has no component ref |
| 1 | `units` | 0 | none |  | `tests/specs/async/time/DateTimeHelperSpec.cfc:68` variable 'units' has no component ref |
| 1 | `url` | 0 | none |  | `test-harness/config/Scheduler.cfc:19` variable 'url' has no component ref |
| 1 | `url.path` | 0 | none |  | `tests/index.cfm:41` variable 'url.path' has no component ref |
| 1 | `value → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/flash/AbstractFlashScope.cfc:110` variable 'value' has no component ref |
| 1 | `variables.appconfigmodules → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/services/ModuleService.cfc:1350` variable 'variables.appConfigModules' has no component ref |
| 1 | `variables.appenders → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/logging/Logger.cfc:121` variable 'variables.appenders' has no component ref |
| 1 | `variables.appenders → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/logging/Logger.cfc:86` variable 'variables.appenders' has no component ref |
| 1 | `variables.appmapping` | 0 | none |  | `system/web/services/RoutingService.cfc:121` variable 'variables.appMapping' has no component ref |
| 1 | `variables.appsettings.coldboxconfig → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares getPropertyMixin() | `system/web/services/ModuleService.cfc:69` variable 'variables.appSettings.coldBoxConfig' has no component ref |
| 1 | `variables.cachedpaths → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/modules/HTMLHelper/models/HTMLHelper.cfc:2497` variable 'variables.cachedPaths' has no component ref |
| 1 | `variables.cacheprovider → system/cache/AbstractCacheBoxProvider.cfc` | 7 | low | declares lookupQuiet() (7 candidates) | `system/ioc/scopes/CacheBox.cfc:138` variable 'variables.cacheProvider' has no component ref |
| 1 | `variables.cacheprovider → system/cache/CacheFactory.cfc` | 14 | low | declares getColdBox() (14 candidates) | `system/cache/util/EventURLFacade.cfc:29` variable 'variables.cacheProvider' has no component ref |
| 1 | `variables.cacheprovider → system/cache/providers/BoxLangColdBoxProvider.cfc` | 6 | low | declares getEventCacheKeyPrefix() (6 candidates) | `system/cache/util/EventURLFacade.cfc:170` variable 'variables.cacheProvider' has no component ref |
| 1 | `variables.cacheprovider → system/web/Controller.cfc` | 3 | low | declares getRequestService() (3 candidates) | `system/cache/util/EventURLFacade.cfc:31` variable 'variables.cacheProvider' has no component ref |
| 1 | `variables.cacheprovider → system/web/context/RequestContext.cfc` | 2 | medium | declares getSesBaseUrl() | `system/cache/util/EventURLFacade.cfc:33` variable 'variables.cacheProvider' has no component ref |
| 1 | `variables.cacheprovider → system/web/services/RequestService.cfc` | 4 | medium | declares getContext(); named like the receiver 'RequestService' (3 candidates) | `system/cache/util/EventURLFacade.cfc:32` variable 'variables.cacheProvider' has no component ref |
| 1 | `variables.cbinjectedhelpers → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/FrameworkSupertype.cfc:880` variable 'variables.cbInjectedHelpers' has no component ref |
| 1 | `variables.coldboxsettings` | 0 | none |  | `system/web/Controller.cfc:320` variable 'variables.coldboxSettings' has no component ref |
| 1 | `variables.coldboxsettings → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/Controller.cfc:309` variable 'variables.coldboxSettings' has no component ref |
| 1 | `variables.configsettings → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/Controller.cfc:273` variable 'variables.configSettings' has no component ref |
| 1 | `variables.customdsl → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/ioc/Builder.cfc:530` variable 'variables.customDSL' has no component ref |
| 1 | `variables.engines` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:253` variable 'variables.ENGINES' has no component ref |
| 1 | `variables.environments` | 0 | none |  | `system/web/tasks/ColdBoxScheduledTask.cfc:129` variable 'variables.environments' has no component ref |
| 1 | `variables.eventcachedictionary → system/web/context/InterceptorBuffer.cfc` | 18 | low | declares clear() (17 candidates) | `system/web/services/HandlerService.cfc:566` variable 'variables.eventCacheDictionary' has no component ref |
| 1 | `variables.eventstates → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/core/events/EventPoolManager.cfc:178` variable 'variables.eventStates' has no component ref |
| 1 | `variables.eviction_policies → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/cache/providers/CacheBoxProvider.cfc:658` variable 'variables.EVICTION_POLICIES' has no component ref |
| 1 | `variables.executor` | 1 | none |  | `system/async/tasks/Scheduler.cfc:370` variable 'variables.executor' has no component ref |
| 1 | `variables.executors → system/ioc/config/Binder.cfc` | 1 | medium | declares map() | `system/async/AsyncManager.cfc:308` variable 'variables.executors' has no component ref |
| 1 | `variables.executors → system/testing/BaseTestCase.cfc` | 6 | low | declares delete() (6 candidates) | `system/async/AsyncManager.cfc:248` variable 'variables.executors' has no component ref |
| 1 | `variables.expandpathcache → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/Renderer.cfc:1037` variable 'variables.expandPathCache' has no component ref |
| 1 | `variables.extrainfo → tests/specs/logging/ExtraInfo.cfc` | 2 | medium | declares $toString(); named like the receiver 'extraInfo' (2 candidates) | `system/logging/LogEvent.cfc:95` variable 'variables.extraInfo' has no component ref |
| 1 | `variables.features → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/async/executors/Executor.cfc:165` variable 'variables.features' has no component ref |
| 1 | `variables.fields` | 0 | none |  | `system/cache/store/indexers/JDBCMetadataIndexer.cfc:158` variable 'variables.fields' has no component ref |
| 1 | `variables.flashdata → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/web/services/RequestService.cfc:36` variable 'variables.flashData' has no component ref |
| 1 | `variables.injectorreferencemap` | 0 | none |  | `system/ioc/Injector.cfc:343` variable 'variables.injectorReferenceMap' has no component ref |
| 1 | `variables.interceptionpoints` | 0 | none |  | `system/web/services/InterceptorService.cfc:602` variable 'variables.interceptionPoints' has no component ref |
| 1 | `variables.interceptionpoints → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/web/services/InterceptorService.cfc:621` variable 'variables.interceptionPoints' has no component ref |
| 1 | `variables.interceptorconfig.custominterceptionpoints → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/web/services/InterceptorService.cfc:168` variable 'variables.interceptorConfig.customInterceptionPoints' has no component ref |
| 1 | `variables.loglistener.queue → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/logging/AbstractAppender.cfc:560` variable 'variables.logListener.queue' has no component ref |
| 1 | `variables.mappings → system/testing/BaseTestCase.cfc` | 6 | low | declares delete() (6 candidates) | `system/ioc/config/Binder.cfc:1371` variable 'variables.mappings' has no component ref |
| 1 | `variables.meta → system/testing/BaseTestCase.cfc` | 6 | low | declares delete() (6 candidates) | `system/async/tasks/ScheduledTask.cfc:450` variable 'variables.meta' has no component ref |
| 1 | `variables.mime_extension_aliases → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/routing/Router.cfc:257` variable 'variables.MIME_EXTENSION_ALIASES' has no component ref |
| 1 | `variables.object_stores → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/cache/providers/CacheBoxProvider.cfc:672` variable 'variables.OBJECT_STORES' has no component ref |
| 1 | `variables.objectmetadata → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/ioc/config/Mapping.cfc:855` variable 'variables.objectMetadata' has no component ref |
| 1 | `variables.properties → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/logging/AbstractAppender.cfc:269` variable 'variables.properties' has no component ref |
| 1 | `variables.renderedhelpers → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/web/RendererEncapsulator.cfm:49` variable 'variables.renderedHelpers' has no component ref |
| 1 | `variables.reserved_keys` | 0 | none |  | `system/ioc/scopes/Singleton.cfc:169` variable 'variables.RESERVED_KEYS' has no component ref |
| 1 | `variables.scheduler → system/async/tasks/Scheduler.cfc` | 16 | medium | declares getUtil(); named like the receiver 'scheduler' (15 candidates) | `system/async/tasks/ScheduledTask.cfc:207` variable 'variables.scheduler' has no component ref |
| 1 | `variables.scopes` | 0 | none |  | `system/ioc/Injector.cfc:1061` variable 'variables.scopes' has no component ref |
| 1 | `variables.scopes → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares keyExists() | `system/ioc/Injector.cfc:1058` variable 'variables.scopes' has no component ref |
| 1 | `variables.scopes[] → system/ioc/scopes/CFScopes.cfc` | 6 | low | declares getFromScope() (6 candidates) | `system/ioc/Injector.cfc:591` variable 'variables.scopes[]' has no component ref |
| 1 | `variables.target → system/aop/MixerUtil.cfc` | 1 | medium | declares $wbAOPInvokeProxy() | `system/aop/MethodInvocation.cfc:129` variable 'variables.target' has no component ref |
| 1 | `variables.thisroute.headers` | 0 | none |  | `system/web/routing/Router.cfc:1401` variable 'variables.thisRoute.headers' has no component ref |
| 1 | `variables.thisroute.headers → system/web/routing/Router.cfc` | 3 | low | declares append() (2 candidates) | `system/web/routing/Router.cfc:1423` variable 'variables.thisRoute.headers' has no component ref |
| 1 | `variables.thisroute.meta → system/web/routing/Router.cfc` | 3 | low | declares append() (2 candidates) | `system/web/routing/Router.cfc:1441` variable 'variables.thisRoute.meta' has no component ref |
| 1 | `variables.thisroute.prc` | 0 | none |  | `system/web/routing/Router.cfc:1526` variable 'variables.thisRoute.prc' has no component ref |
| 1 | `variables.thisroute.prc → system/web/routing/Router.cfc` | 3 | low | declares append() (2 candidates) | `system/web/routing/Router.cfc:1548` variable 'variables.thisRoute.prc' has no component ref |
| 1 | `variables.thisroute.rc` | 0 | none |  | `system/web/routing/Router.cfc:1481` variable 'variables.thisRoute.rc' has no component ref |
| 1 | `variables.thisroute.rc → system/web/routing/Router.cfc` | 3 | low | declares append() (2 candidates) | `system/web/routing/Router.cfc:1503` variable 'variables.thisRoute.rc' has no component ref |
| 1 | `variables.validextensions` | 0 | none |  | `system/web/routing/Router.cfc:242` variable 'variables.validExtensions' has no component ref |
| 1 | `variables.viewshelper` | 0 | none |  | `system/web/Renderer.cfc:108` variable 'variables.viewsHelper' has no component ref |
| 1 | `variables.withclosure → system/web/routing/Router.cfc` | 3 | low | declares append() (2 candidates) | `system/web/routing/Router.cfc:521` variable 'variables.withClosure' has no component ref |
| 1 | `vd.coldstart → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `tests/perf-harness/PerformanceSuite.cfc:665` variable 'vd.coldStart' has no component ref |
| 1 | `xmlroot.xmlchildren` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:123` variable 'xmlRoot.XMLChildren' has no component ref |

<details><summary>Groups with several candidates</summary>

- `arguments.mapping → system/ioc/config/Mapping.cfc` — 18 finding(s), 2 candidate(s):
  - medium `system/ioc/config/Mapping.cfc` — declares getObjectMetadata(); named like the receiver 'mapping'
  - low `system/cache/store/indexers/JDBCMetadataIndexer.cfc` — declares getObjectMetadata()
- `emitter → system/web/context/SSEEmitter.cfc` — 14 finding(s), 2 candidate(s):
  - low `system/web/context/SSEEmitter.cfc` — declares isClosed(), send()
  - low `system/testing/mock/web/MockSSEEmitter.cfc` — declares isClosed(), send()
- `cachestats → system/cache/util/CacheStats.cfc` — 12 finding(s), 5 candidate(s):
  - medium `system/cache/util/CacheStats.cfc` — declares getHits(), getMisses(), getGarbageCollections(), getEvictionCount(); named like the receiver 'cacheStats'
  - low `system/cache/util/IStats.cfc` — declares getHits(), getMisses(), getGarbageCollections(), getEvictionCount()
  - low `system/cache/providers/stats/BoxLangStats.cfc` — declares getHits(), getMisses(), getGarbageCollections(), getEvictionCount()
  - low `system/cache/providers/stats/CFStats.cfc` — declares getHits(), getMisses(), getGarbageCollections(), getEvictionCount()
  - low `system/cache/providers/stats/LuceeStats.cfc` — declares getHits(), getMisses(), getGarbageCollections(), getEvictionCount()
- `variables.asyncmanager → system/async/AsyncManager.cfc` — 12 finding(s), 5 candidate(s):
  - medium `system/async/AsyncManager.cfc` — declares out(); named like the receiver 'asyncManager'
  - low `system/async/tasks/ScheduledTask.cfc` — declares out()
  - low `system/async/executors/Executor.cfc` — declares out()
  - low `system/logging/AbstractAppender.cfc` — declares out()
  - low `system/web/tasks/ColdBoxScheduledTask.cfc` — declares out()
- `services.interceptorservice → system/web/services/InterceptorService.cfc` — 8 finding(s), 5 candidate(s):
  - medium `system/web/services/InterceptorService.cfc` — declares announce(); named like the receiver 'interceptorService'
  - low `system/FrameworkSupertype.cfc` — declares announce()
  - low `system/remote/ColdboxProxy.cfc` — declares announce()
  - low `system/testing/BaseTestCase.cfc` — declares announce()
  - low `system/core/events/EventPoolManager.cfc` — declares announce()
- `variables.emitter → system/web/context/SSEEmitter.cfc` — 6 finding(s), 2 candidate(s):
  - low `system/web/context/SSEEmitter.cfc` — declares isClosed()
  - low `system/testing/mock/web/MockSSEEmitter.cfc` — declares isClosed()
- `variables.executor → system/async/executors/Executor.cfc` — 6 finding(s), 5 candidate(s):
  - medium `system/async/executors/Executor.cfc` — declares getNative(); named like the receiver 'executor'
  - low `system/async/tasks/Future.cfc` — declares getNative()
  - low `system/async/tasks/FutureTask.cfc` — declares getNative()
  - low `system/async/time/Duration.cfc` — declares getNative()
  - low `system/async/time/Period.cfc` — declares getNative()
- `variables.thisroute → system/web/routing/Router.cfc` — 6 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares append()
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
- `expectation.actual → system/web/context/RequestContext.cfc` — 5 finding(s), 2 candidate(s):
  - low `system/web/context/RequestContext.cfc` — declares getStatusCode(), getMemento()
  - low `system/web/context/Response.cfc` — declares getStatusCode(), getMemento()
- `request.testbox → system/logging/Logger.cfc` — 5 finding(s), 3 candidate(s):
  - low `system/logging/Logger.cfc` — declares debug()
  - low `system/async/tasks/ScheduledTask.cfc` — declares debug()
  - low `system/logging/config/LogBoxConfig.cfc` — declares debug()
- `variables.native → system/async/tasks/Future.cfc` — 5 finding(s), 2 candidate(s):
  - low `system/async/tasks/Future.cfc` — declares cancel()
  - low `system/async/tasks/FutureTask.cfc` — declares cancel()
- `results.viewhelperpath → system/web/context/InterceptorBuffer.cfc` — 4 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `services[] → system/web/services/BaseService.cfc` — 4 finding(s), 2 candidate(s):
  - low `system/web/services/BaseService.cfc` — declares onConfigurationLoad(), afterAspectsLoad()
  - low `system/web/services/RequestService.cfc` — declares onConfigurationLoad(), afterAspectsLoad()
- `sharedargs → system/web/routing/Router.cfc` — 4 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares append()
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
- `application[] → system/FrameworkSupertype.cfc` — 3 finding(s), 2 candidate(s):
  - low `system/FrameworkSupertype.cfc` — declares getSetting(), getLog()
  - low `system/web/Controller.cfc` — declares getSetting(), getLog()
- `arguments.logboxconfig → system/cache/config/CacheBoxConfig.cfc` — 3 finding(s), 2 candidate(s):
  - low `system/cache/config/CacheBoxConfig.cfc` — declares validate()
  - low `system/logging/config/LogBoxConfig.cfc` — declares validate()
- `application.cbcontroller → system/FrameworkSupertype.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/FrameworkSupertype.cfc` — declares setSetting()
  - low `system/web/Controller.cfc` — declares setSetting()
- `application[] → system/logging/Logger.cfc` — 2 finding(s), 2 candidate(s):
  - medium `system/logging/Logger.cfc` — declares info(); named like the receiver 'Logger'
  - low `system/logging/config/LogBoxConfig.cfc` — declares info()
- `arguments.appscope.cbbootstrap → tests/perf-harness/be-app/Application.cfc` — 2 finding(s), 5 candidate(s):
  - low `tests/perf-harness/be-app/Application.cfc` — declares onApplicationEnd()
  - low `tests/perf-harness/stable-app/Application.cfc` — declares onApplicationEnd()
  - low `system/Bootstrap.cfc` — declares onApplicationEnd()
  - low `tests/suites/eventCachingCollisions/Application.cfc` — declares onApplicationEnd()
  - low `tests/suites/loadtests/beload/Application.cfc` — declares onApplicationEnd()
- `arguments.appscope.cbbootstrap → tests/perf-harness/stable-app/Application.cfc` — 2 finding(s), 5 candidate(s):
  - low `tests/perf-harness/stable-app/Application.cfc` — declares onApplicationEnd()
  - low `tests/perf-harness/be-app/Application.cfc` — declares onApplicationEnd()
  - low `system/Bootstrap.cfc` — declares onApplicationEnd()
  - low `tests/suites/eventCachingCollisions/Application.cfc` — declares onApplicationEnd()
  - low `tests/suites/loadtests/beload/Application.cfc` — declares onApplicationEnd()
- `arguments.appscope.cbbootstrap → tests/suites/loadtests/beload/Application.cfc` — 2 finding(s), 5 candidate(s):
  - low `tests/suites/loadtests/beload/Application.cfc` — declares onApplicationEnd()
  - low `tests/suites/eventCachingCollisions/Application.cfc` — declares onApplicationEnd()
  - low `system/Bootstrap.cfc` — declares onApplicationEnd()
  - low `tests/perf-harness/be-app/Application.cfc` — declares onApplicationEnd()
  - low `tests/perf-harness/stable-app/Application.cfc` — declares onApplicationEnd()
- `arguments.childinstance → system/ioc/IInjector.cfc` — 2 finding(s), 15 candidate(s):
  - low `system/ioc/IInjector.cfc` — declares shutdown()
  - low `system/ioc/Injector.cfc` — declares shutdown()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares shutdown()
  - low `system/cache/CacheFactory.cfc` — declares shutdown()
  - low `system/logging/AbstractAppender.cfc` — declares shutdown()
  - low `system/logging/LogBox.cfc` — declares shutdown()
  - low `system/testing/VirtualApp.cfc` — declares shutdown()
  - low `system/async/executors/Executor.cfc` — declares shutdown()
  - … 7 more
- `arguments.future → system/async/tasks/Future.cfc` — 2 finding(s), 5 candidate(s):
  - medium `system/async/tasks/Future.cfc` — declares getNative(); named like the receiver 'future'
  - low `system/async/tasks/FutureTask.cfc` — declares getNative()
  - low `system/async/executors/Executor.cfc` — declares getNative()
  - low `system/async/time/Duration.cfc` — declares getNative()
  - low `system/async/time/Period.cfc` — declares getNative()
- `arguments.logboxconfig → system/logging/config/LogBoxConfig.cfc` — 2 finding(s), 4 candidate(s):
  - medium `system/logging/config/LogBoxConfig.cfc` — declares reset(), init(), getMemento(); named like the receiver 'logBoxConfig'
  - low `system/aop/Matcher.cfc` — declares reset(), init(), getMemento()
  - low `system/cache/config/CacheBoxConfig.cfc` — declares reset(), init(), getMemento()
  - low `system/ioc/config/Binder.cfc` — declares reset(), init(), getMemento()
- `arguments.result → system/web/context/InterceptorBuffer.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `arguments.viewhelperpath → system/async/time/Duration.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `cacheconfig → system/async/time/Duration.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `cacheprovider → system/cache/AbstractCacheBoxProvider.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getName(), isReportingEnabled()
  - low `system/cache/providers/ICacheProvider.cfc` — declares getName(), isReportingEnabled()
- `executor → system/async/executors/Executor.cfc` — 2 finding(s), 16 candidate(s):
  - medium `system/async/executors/Executor.cfc` — declares getName(); named like the receiver 'executor'
  - low `test-harness/models/RenderConvention.cfc` — declares getName()
  - low `test-harness/models/ioc/Simple.cfc` — declares getName()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getName()
  - low `system/ioc/Injector.cfc` — declares getName()
  - low `system/ioc/Provider.cfc` — declares getName()
  - low `system/logging/AbstractAppender.cfc` — declares getName()
  - low `tests/resources/Test.cfc` — declares getName()
  - … 8 more
- `idata → system/web/context/InterceptorBuffer.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `mconfig.settings → system/web/context/InterceptorBuffer.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `mystruct → system/async/time/Duration.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `omodel → test-harness/remote/Echo.cfc` — 2 finding(s), 3 candidate(s):
  - low `test-harness/remote/Echo.cfc` — declares echo()
  - low `test-harness/remote/MyProxy.cfc` — declares echo()
  - low `test-harness/modules_app/conventionsTest/system/model/conventionsTest.cfc` — declares echo()
- `target → system/cache/store/BlackHoleStore.cfc` — 2 finding(s), 24 candidate(s):
  - low `system/cache/store/BlackHoleStore.cfc` — declares get()
  - low `system/cache/store/ConcurrentSoftReferenceStore.cfc` — declares get()
  - low `system/cache/store/ConcurrentStore.cfc` — declares get()
  - low `system/cache/store/DiskStore.cfc` — declares get()
  - low `system/cache/store/IObjectStore.cfc` — declares get()
  - low `system/cache/store/JDBCStore.cfc` — declares get()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares get()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares get()
  - … 16 more
- `task → system/async/time/Duration.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `task → system/cache/AbstractCacheBoxProvider.cfc` — 2 finding(s), 8 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getStats()
  - low `system/async/executors/Executor.cfc` — declares getStats()
  - low `system/async/tasks/ScheduledTask.cfc` — declares getStats()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares getStats()
  - low `system/cache/providers/CFProvider.cfc` — declares getStats()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares getStats()
  - low `system/cache/providers/ICacheProvider.cfc` — declares getStats()
  - low `system/cache/providers/LuceeProvider.cfc` — declares getStats()
- `thisdidata → system/async/time/Duration.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `variables.configuration → system/async/time/Duration.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `variables.data → system/web/context/InterceptorBuffer.cfc` — 2 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `variables.elementcleaner → system/cache/util/ElementCleaner.cfc` — 2 finding(s), 2 candidate(s):
  - medium `system/cache/util/ElementCleaner.cfc` — declares clearByKeySnippet(); named like the receiver 'elementCleaner'
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares clearByKeySnippet()
- `application.cbbootstrap → system/remote/ColdboxProxy.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/remote/ColdboxProxy.cfc` — declares getCOLDBOX_APP_KEY()
  - low `system/Bootstrap.cfc` — declares getCOLDBOX_APP_KEY()
- `application.wirebox → system/aop/Mixer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/aop/Mixer.cfc` — declares getBinder()
  - low `system/ioc/Injector.cfc` — declares getBinder()
- `application.wirebox → system/ioc/config/Binder.cfc` — 1 finding(s), 2 candidate(s):
  - medium `system/ioc/config/Binder.cfc` — declares getMappings(); named like the receiver 'Binder'
  - low `system/aop/Matcher.cfc` — declares getMappings()
- `application[] → system/logging/LogBox.cfc` — 1 finding(s), 8 candidate(s):
  - medium `system/logging/LogBox.cfc` — declares getLogger(); named like the receiver 'LogBox'
  - low `system/remote/ColdboxProxy.cfc` — declares getLogger()
  - low `system/cache/policies/AbstractEvictionPolicy.cfc` — declares getLogger()
  - low `system/web/context/InterceptorState.cfc` — declares getLogger()
  - low `system/web/services/BaseService.cfc` — declares getLogger()
  - low `system/web/services/ModuleService.cfc` — declares getLogger()
  - low `test-harness/models/testModel.cfc` — declares getLogger()
  - low `tests/specs/integration/MainSpec.cfc` — declares getLogger()
- `application[] → system/web/services/InterceptorService.cfc` — 1 finding(s), 5 candidate(s):
  - medium `system/web/services/InterceptorService.cfc` — declares announce(); named like the receiver 'InterceptorService'
  - low `system/FrameworkSupertype.cfc` — declares announce()
  - low `system/remote/ColdboxProxy.cfc` — declares announce()
  - low `system/testing/BaseTestCase.cfc` — declares announce()
  - low `system/core/events/EventPoolManager.cfc` — declares announce()
- `appscope.cbbootstrap → tests/suites/eventCachingCollisions/Application.cfc` — 1 finding(s), 9 candidate(s):
  - low `tests/suites/eventCachingCollisions/Application.cfc` — declares onSessionEnd()
  - low `tests/suites/loadtests/beload/Application.cfc` — declares onSessionEnd()
  - low `system/Bootstrap.cfc` — declares onSessionEnd()
  - low `test-harness/Application.cfc` — declares onSessionEnd()
  - low `tests/perf-harness/be-app/Application.cfc` — declares onSessionEnd()
  - low `tests/perf-harness/stable-app/Application.cfc` — declares onSessionEnd()
  - low `tests/suites/loadtests/beload/handlers/Main.cfc` — declares onSessionEnd()
  - low `tests/suites/loadtests/beload/handlers/Simple.cfc` — declares onSessionEnd()
  - … 1 more
- `appsettings.mappings → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `arguments.appender → system/logging/AbstractAppender.cfc` — 1 finding(s), 15 candidate(s):
  - low `system/logging/AbstractAppender.cfc` — declares shutdown()
  - low `system/logging/LogBox.cfc` — declares shutdown()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares shutdown()
  - low `system/cache/CacheFactory.cfc` — declares shutdown()
  - low `system/ioc/IInjector.cfc` — declares shutdown()
  - low `system/ioc/Injector.cfc` — declares shutdown()
  - low `system/testing/VirtualApp.cfc` — declares shutdown()
  - low `system/async/executors/Executor.cfc` — declares shutdown()
  - … 7 more
- `arguments.appscope.cbbootstrap → test-harness/Application.cfc` — 1 finding(s), 9 candidate(s):
  - low `test-harness/Application.cfc` — declares onSessionEnd()
  - low `test-harness/handlers/main.cfc` — declares onSessionEnd()
  - low `system/Bootstrap.cfc` — declares onSessionEnd()
  - low `tests/perf-harness/be-app/Application.cfc` — declares onSessionEnd()
  - low `tests/perf-harness/stable-app/Application.cfc` — declares onSessionEnd()
  - low `tests/suites/eventCachingCollisions/Application.cfc` — declares onSessionEnd()
  - low `tests/suites/loadtests/beload/Application.cfc` — declares onSessionEnd()
  - low `tests/suites/loadtests/beload/handlers/Main.cfc` — declares onSessionEnd()
  - … 1 more
- `arguments.asyncmanager → system/async/AsyncManager.cfc` — 1 finding(s), 5 candidate(s):
  - medium `system/async/AsyncManager.cfc` — declares out(); named like the receiver 'asyncManager'
  - low `system/async/tasks/ScheduledTask.cfc` — declares out()
  - low `system/async/executors/Executor.cfc` — declares out()
  - low `system/logging/AbstractAppender.cfc` — declares out()
  - low `system/web/tasks/ColdBoxScheduledTask.cfc` — declares out()
- `arguments.buffer → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `arguments.child → system/ioc/IInjector.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/ioc/IInjector.cfc` — declares setParent()
  - low `system/ioc/Injector.cfc` — declares setParent()
- `arguments.dictionary[].aspects → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `arguments.jfuture → system/async/tasks/Future.cfc` — 1 finding(s), 24 candidate(s):
  - low `system/async/tasks/Future.cfc` — declares get()
  - low `system/async/tasks/FutureTask.cfc` — declares get()
  - low `system/async/time/Duration.cfc` — declares get()
  - low `system/async/time/Period.cfc` — declares get()
  - low `system/async/time/TimeUnit.cfc` — declares get()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares get()
  - low `system/testing/BaseTestCase.cfc` — declares get()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares get()
  - … 16 more
- `arguments.ocontext → system/web/context/ExceptionBean.cfc` — 1 finding(s), 16 candidate(s):
  - low `system/web/context/ExceptionBean.cfc` — declares getMemento()
  - low `system/web/context/RequestContext.cfc` — declares getMemento()
  - low `system/web/context/Response.cfc` — declares getMemento()
  - low `system/web/Controller.cfc` — declares getMemento()
  - low `system/web/routing/Router.cfc` — declares getMemento()
  - low `system/aop/Matcher.cfc` — declares getMemento()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getMemento()
  - low `system/async/tasks/ScheduledTask.cfc` — declares getMemento()
  - … 8 more
- `arguments.record.task → system/async/tasks/ScheduledTask.cfc` — 1 finding(s), 8 candidate(s):
  - low `system/async/tasks/ScheduledTask.cfc` — declares getStats()
  - low `system/async/executors/Executor.cfc` — declares getStats()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getStats()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares getStats()
  - low `system/cache/providers/CFProvider.cfc` — declares getStats()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares getStats()
  - low `system/cache/providers/ICacheProvider.cfc` — declares getStats()
  - low `system/cache/providers/LuceeProvider.cfc` — declares getStats()
- `arguments.result.value → system/aop/Matcher.cfc` — 1 finding(s), 16 candidate(s):
  - low `system/aop/Matcher.cfc` — declares getMemento()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getMemento()
  - low `system/web/Controller.cfc` — declares getMemento()
  - low `system/async/tasks/ScheduledTask.cfc` — declares getMemento()
  - low `system/cache/config/CacheBoxConfig.cfc` — declares getMemento()
  - low `system/cache/util/CacheStats.cfc` — declares getMemento()
  - low `system/ioc/config/Binder.cfc` — declares getMemento()
  - low `system/ioc/config/Mapping.cfc` — declares getMemento()
  - … 8 more
- `arguments.scope[] → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `arguments.thisexecutor → system/async/executors/Executor.cfc` — 1 finding(s), 8 candidate(s):
  - low `system/async/executors/Executor.cfc` — declares getStats()
  - low `system/async/tasks/ScheduledTask.cfc` — declares getStats()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getStats()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares getStats()
  - low `system/cache/providers/CFProvider.cfc` — declares getStats()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares getStats()
  - low `system/cache/providers/ICacheProvider.cfc` — declares getStats()
  - low `system/cache/providers/LuceeProvider.cfc` — declares getStats()
- `arguments.thisfuture → system/async/tasks/Future.cfc` — 1 finding(s), 24 candidate(s):
  - low `system/async/tasks/Future.cfc` — declares get()
  - low `system/async/tasks/FutureTask.cfc` — declares get()
  - low `system/async/time/Duration.cfc` — declares get()
  - low `system/async/time/Period.cfc` — declares get()
  - low `system/async/time/TimeUnit.cfc` — declares get()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares get()
  - low `system/testing/BaseTestCase.cfc` — declares get()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares get()
  - … 16 more
- `arguments.thismapping → system/ioc/config/Binder.cfc` — 1 finding(s), 11 candidate(s):
  - low `system/ioc/config/Binder.cfc` — declares process()
  - low `system/ioc/config/Mapping.cfc` — declares process()
  - low `system/ioc/dsl/CacheBoxDSL.cfc` — declares process()
  - low `system/ioc/dsl/ColdBoxDSL.cfc` — declares process()
  - low `system/ioc/dsl/IDSLBuilder.cfc` — declares process()
  - low `system/ioc/dsl/LogBoxDSL.cfc` — declares process()
  - low `system/remote/ColdboxProxy.cfc` — declares process()
  - low `system/core/events/EventPool.cfc` — declares process()
  - … 3 more
- `buffer → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `cache → system/cache/AbstractCacheBoxProvider.cfc` — 1 finding(s), 15 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares shutdown()
  - low `system/cache/CacheFactory.cfc` — declares shutdown()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares shutdown()
  - low `system/cache/providers/CFProvider.cfc` — declares shutdown()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares shutdown()
  - low `system/cache/providers/ICacheProvider.cfc` — declares shutdown()
  - low `system/cache/providers/LuceeProvider.cfc` — declares shutdown()
  - low `system/cache/providers/MockProvider.cfc` — declares shutdown()
  - … 7 more
- `cacheconfig[] → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `childinjector → system/ioc/IInjector.cfc` — 1 finding(s), 7 candidate(s):
  - low `system/ioc/IInjector.cfc` — declares getInstance()
  - low `system/ioc/Injector.cfc` — declares getInstance()
  - low `system/FrameworkSupertype.cfc` — declares getInstance()
  - low `system/remote/ColdboxProxy.cfc` — declares getInstance()
  - low `system/testing/BaseTestCase.cfc` — declares getInstance()
  - low `test-harness/models/formBean.cfc` — declares getInstance()
  - low `test-harness/models/formImplicitBean.cfc` — declares getInstance()
- `config → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `d → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `e → system/logging/LogEvent.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/logging/LogEvent.cfc` — declares getMessage()
  - low `system/web/context/ExceptionBean.cfc` — declares getMessage()
- `e.tagcontext → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `eventcache → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `ex → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `initialnextrun → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `interceptionstate → system/web/context/InterceptorState.cfc` — 1 finding(s), 11 candidate(s):
  - low `system/web/context/InterceptorState.cfc` — declares process()
  - low `system/remote/ColdboxProxy.cfc` — declares process()
  - low `system/core/events/EventPool.cfc` — declares process()
  - low `system/ioc/config/Binder.cfc` — declares process()
  - low `system/ioc/config/Mapping.cfc` — declares process()
  - low `system/ioc/dsl/CacheBoxDSL.cfc` — declares process()
  - low `system/ioc/dsl/ColdBoxDSL.cfc` — declares process()
  - low `system/ioc/dsl/IDSLBuilder.cfc` — declares process()
  - … 3 more
- `item → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `loc.values → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `local.docresults.filecontent → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `local.idata → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `local.results → system/web/routing/Router.cfc` — 1 finding(s), 6 candidate(s):
  - low `system/web/routing/Router.cfc` — declares delete()
  - low `system/testing/BaseTestCase.cfc` — declares delete()
  - low `system/core/collections/ScopeStorage.cfc` — declares delete()
  - low `test-harness/handlers/photos.cfc` — declares delete()
  - low `test-harness/models/PhotosService.cfc` — declares delete()
  - low `test-harness/modules_app/resourcesTest/handlers/photos.cfc` — declares delete()
- `local.thisarg → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `mappings[] → tests/specs/core/collections/ScopeStorageTest.cfc` — 1 finding(s), 5 candidate(s):
  - low `tests/specs/core/collections/ScopeStorageTest.cfc` — declares getScope()
  - low `system/ioc/Injector.cfc` — declares getScope()
  - low `system/core/collections/ScopeStorage.cfc` — declares getScope()
  - low `system/ioc/config/Mapping.cfc` — declares getScope()
  - low `system/web/flash/AbstractFlashScope.cfc` — declares getScope()
- `mconfig.injector → system/web/routing/Router.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares to()
  - low `system/ioc/config/Binder.cfc` — declares to()
- `mconfig.scheduler → system/async/tasks/Scheduler.cfc` — 1 finding(s), 16 candidate(s):
  - medium `system/async/tasks/Scheduler.cfc` — declares getName(); named like the receiver 'scheduler'
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getName()
  - low `system/ioc/Injector.cfc` — declares getName()
  - low `system/ioc/Provider.cfc` — declares getName()
  - low `system/logging/AbstractAppender.cfc` — declares getName()
  - low `system/async/executors/Executor.cfc` — declares getName()
  - low `system/async/tasks/ScheduledTask.cfc` — declares getName()
  - low `system/cache/providers/ICacheProvider.cfc` — declares getName()
  - … 8 more
- `message → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `mixer.afterinstanceautowire → system/cache/store/ConcurrentStore.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/cache/store/ConcurrentStore.cfc` — declares getPool()
  - low `system/core/events/EventPool.cfc` — declares getPool()
- `oappender → system/logging/AbstractAppender.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/logging/AbstractAppender.cfc` — declares onUnRegistration()
  - low `system/logging/appenders/SocketAppender.cfc` — declares onUnRegistration()
- `ointerceptor → system/Interceptor.cfc` — 1 finding(s), 92 candidate(s):
  - medium `system/Interceptor.cfc` — declares configure(); named like the receiver 'oInterceptor'
  - low `system/web/services/InterceptorService.cfc` — declares configure()
  - low `system/web/ControllerDecorator.cfc` — declares configure()
  - low `system/web/config/CacheBox.cfc` — declares configure()
  - low `system/web/config/LogBox.cfc` — declares configure()
  - low `system/web/config/Settings.cfc` — declares configure()
  - low `system/web/context/RequestContextDecorator.cfc` — declares configure()
  - low `system/web/routing/Router.cfc` — declares configure()
  - … 84 more
- `options → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `oscheduler → system/logging/LogEvent.cfc` — 1 finding(s), 3 candidate(s):
  - low `system/logging/LogEvent.cfc` — declares setCategory()
  - low `system/logging/Logger.cfc` — declares setCategory()
  - low `test-harness/models/entities/Category.cfc` — declares setCategory()
- `propertyvalue → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `providertest.coolpizza → system/ioc/IProvider.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/ioc/IProvider.cfc` — declares $get()
  - low `system/ioc/Provider.cfc` — declares $get()
- `resourcespool → test-harness/models/RenderConvention.cfc` — 1 finding(s), 16 candidate(s):
  - low `test-harness/models/RenderConvention.cfc` — declares getName()
  - low `test-harness/models/ioc/Simple.cfc` — declares getName()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getName()
  - low `system/ioc/Injector.cfc` — declares getName()
  - low `system/ioc/Provider.cfc` — declares getName()
  - low `system/logging/AbstractAppender.cfc` — declares getName()
  - low `tests/resources/Test.cfc` — declares getName()
  - low `tests/tmp/Person.cfc` — declares getName()
  - … 8 more
- `result → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `result.rows → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `result[] → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `results.route → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `scheduler → system/async/tasks/Scheduler.cfc` — 1 finding(s), 2 candidate(s):
  - medium `system/async/tasks/Scheduler.cfc` — declares restart(); named like the receiver 'scheduler'
  - low `system/testing/VirtualApp.cfc` — declares restart()
- `scopeinfo → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `services.requestservice → system/web/flash/AbstractFlashScope.cfc` — 1 finding(s), 4 candidate(s):
  - low `system/web/flash/AbstractFlashScope.cfc` — declares saveFlash()
  - low `system/web/flash/ColdboxCacheFlash.cfc` — declares saveFlash()
  - low `system/web/flash/MockFlash.cfc` — declares saveFlash()
  - low `system/web/flash/SessionFlash.cfc` — declares saveFlash()
- `settings → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `stime → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares plusDays()
  - low `system/async/time/Period.cfc` — declares plusDays()
- `targetmixer → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `tasklist → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `taskrecord.future → system/async/tasks/Future.cfc` — 1 finding(s), 2 candidate(s):
  - medium `system/async/tasks/Future.cfc` — declares cancel(); named like the receiver 'future'
  - low `system/async/tasks/FutureTask.cfc` — declares cancel()
- `this.$wbinjector → system/ioc/IInjector.cfc` — 1 finding(s), 7 candidate(s):
  - low `system/ioc/IInjector.cfc` — declares getInstance()
  - low `system/ioc/Injector.cfc` — declares getInstance()
  - low `system/FrameworkSupertype.cfc` — declares getInstance()
  - low `system/remote/ColdboxProxy.cfc` — declares getInstance()
  - low `system/testing/BaseTestCase.cfc` — declares getInstance()
  - low `test-harness/models/formBean.cfc` — declares getInstance()
  - low `test-harness/models/formImplicitBean.cfc` — declares getInstance()
- `this.mappings → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `thisroute → system/web/routing/Router.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares append()
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
- `unit → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `variables.cacheprovider → system/cache/AbstractCacheBoxProvider.cfc` — 1 finding(s), 7 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares lookupQuiet()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares lookupQuiet()
  - low `system/cache/providers/CFProvider.cfc` — declares lookupQuiet()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares lookupQuiet()
  - low `system/cache/providers/ICacheProvider.cfc` — declares lookupQuiet()
  - low `system/cache/providers/LuceeProvider.cfc` — declares lookupQuiet()
  - low `system/cache/providers/MockProvider.cfc` — declares lookupQuiet()
- `variables.cacheprovider → system/cache/CacheFactory.cfc` — 1 finding(s), 14 candidate(s):
  - low `system/cache/CacheFactory.cfc` — declares getColdBox()
  - low `system/cache/providers/BoxLangColdBoxProvider.cfc` — declares getColdBox()
  - low `system/cache/providers/CFColdBoxProvider.cfc` — declares getColdBox()
  - low `system/cache/providers/CacheBoxColdBoxProvider.cfc` — declares getColdBox()
  - low `system/cache/providers/IColdBoxProvider.cfc` — declares getColdBox()
  - low `system/cache/providers/LuceeColdboxProvider.cfc` — declares getColdBox()
  - low `system/cache/providers/MockProvider.cfc` — declares getColdBox()
  - low `system/async/AsyncManager.cfc` — declares getColdBox()
  - … 6 more
- `variables.cacheprovider → system/cache/providers/BoxLangColdBoxProvider.cfc` — 1 finding(s), 6 candidate(s):
  - low `system/cache/providers/BoxLangColdBoxProvider.cfc` — declares getEventCacheKeyPrefix()
  - low `system/cache/providers/CFColdBoxProvider.cfc` — declares getEventCacheKeyPrefix()
  - low `system/cache/providers/CacheBoxColdBoxProvider.cfc` — declares getEventCacheKeyPrefix()
  - low `system/cache/providers/IColdBoxProvider.cfc` — declares getEventCacheKeyPrefix()
  - low `system/cache/providers/LuceeColdboxProvider.cfc` — declares getEventCacheKeyPrefix()
  - low `system/cache/providers/MockProvider.cfc` — declares getEventCacheKeyPrefix()
- `variables.cacheprovider → system/web/Controller.cfc` — 1 finding(s), 3 candidate(s):
  - low `system/web/Controller.cfc` — declares getRequestService()
  - low `system/core/conversion/DataMarshaller.cfc` — declares getRequestService()
  - low `system/modules/HTMLHelper/models/HTMLHelper.cfc` — declares getRequestService()
- `variables.cacheprovider → system/web/services/RequestService.cfc` — 1 finding(s), 3 candidate(s):
  - medium `system/web/services/RequestService.cfc` — declares getContext(); named like the receiver 'RequestService'
  - low `system/web/context/RequestContext.cfc` — declares getContext()
  - low `system/web/delegates/Routable.cfc` — declares getContext()
- `variables.eventcachedictionary → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 17 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares clear()
  - low `system/web/flash/AbstractFlashScope.cfc` — declares clear()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares clear()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares clear()
  - low `system/cache/providers/CFProvider.cfc` — declares clear()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares clear()
  - low `system/cache/providers/ICacheProvider.cfc` — declares clear()
  - low `system/cache/providers/LuceeProvider.cfc` — declares clear()
  - … 9 more
- `variables.eventstates → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `variables.executors → system/testing/BaseTestCase.cfc` — 1 finding(s), 6 candidate(s):
  - low `system/testing/BaseTestCase.cfc` — declares delete()
  - low `system/core/collections/ScopeStorage.cfc` — declares delete()
  - low `system/web/routing/Router.cfc` — declares delete()
  - low `test-harness/handlers/photos.cfc` — declares delete()
  - low `test-harness/models/PhotosService.cfc` — declares delete()
  - low `test-harness/modules_app/resourcesTest/handlers/photos.cfc` — declares delete()
- `variables.extrainfo → tests/specs/logging/ExtraInfo.cfc` — 1 finding(s), 2 candidate(s):
  - medium `tests/specs/logging/ExtraInfo.cfc` — declares $toString(); named like the receiver 'extraInfo'
  - low `system/web/context/ExceptionBean.cfc` — declares $toString()
- `variables.flashdata → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `variables.interceptionpoints → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `variables.interceptorconfig.custominterceptionpoints → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `variables.loglistener.queue → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `variables.mappings → system/testing/BaseTestCase.cfc` — 1 finding(s), 6 candidate(s):
  - low `system/testing/BaseTestCase.cfc` — declares delete()
  - low `system/core/collections/ScopeStorage.cfc` — declares delete()
  - low `system/web/routing/Router.cfc` — declares delete()
  - low `test-harness/handlers/photos.cfc` — declares delete()
  - low `test-harness/models/PhotosService.cfc` — declares delete()
  - low `test-harness/modules_app/resourcesTest/handlers/photos.cfc` — declares delete()
- `variables.meta → system/testing/BaseTestCase.cfc` — 1 finding(s), 6 candidate(s):
  - low `system/testing/BaseTestCase.cfc` — declares delete()
  - low `system/core/collections/ScopeStorage.cfc` — declares delete()
  - low `system/web/routing/Router.cfc` — declares delete()
  - low `test-harness/handlers/photos.cfc` — declares delete()
  - low `test-harness/models/PhotosService.cfc` — declares delete()
  - low `test-harness/modules_app/resourcesTest/handlers/photos.cfc` — declares delete()
- `variables.scheduler → system/async/tasks/Scheduler.cfc` — 1 finding(s), 15 candidate(s):
  - medium `system/async/tasks/Scheduler.cfc` — declares getUtil(); named like the receiver 'scheduler'
  - low `system/logging/AbstractAppender.cfc` — declares getUtil()
  - low `system/logging/LogEvent.cfc` — declares getUtil()
  - low `system/remote/ColdboxProxy.cfc` — declares getUtil()
  - low `system/testing/BaseTestCase.cfc` — declares getUtil()
  - low `system/web/Controller.cfc` — declares getUtil()
  - low `system/cache/policies/AbstractEvictionPolicy.cfc` — declares getUtil()
  - low `system/cache/report/ReportHandler.cfc` — declares getUtil()
  - … 7 more
- `variables.scopes[] → system/ioc/scopes/CFScopes.cfc` — 1 finding(s), 6 candidate(s):
  - low `system/ioc/scopes/CFScopes.cfc` — declares getFromScope()
  - low `system/ioc/scopes/CacheBox.cfc` — declares getFromScope()
  - low `system/ioc/scopes/IScope.cfc` — declares getFromScope()
  - low `system/ioc/scopes/NoScope.cfc` — declares getFromScope()
  - low `system/ioc/scopes/RequestScope.cfc` — declares getFromScope()
  - low `system/ioc/scopes/Singleton.cfc` — declares getFromScope()
- `variables.thisroute.headers → system/web/routing/Router.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares append()
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
- `variables.thisroute.meta → system/web/routing/Router.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares append()
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
- `variables.thisroute.prc → system/web/routing/Router.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares append()
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
- `variables.thisroute.rc → system/web/routing/Router.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares append()
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
- `variables.withclosure → system/web/routing/Router.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/routing/Router.cfc` — declares append()
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()

</details>

## Return types — a call chained on a method that declares no component

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 21 | `method 'registerNewInstance' in coldbox.system.ioc.Injector has no component return type → system/ioc/config/Mapping.cfc` | 1 | medium | declares setVirtualInheritance() | `system/web/services/HandlerService.cfc:113` method 'registerNewInstance' in coldbox.system.ioc.Injector has no component return type (chain to 'setVirtualInheritance') |
| 14 | `method 'getEventManager' has no component return type → system/FrameworkSupertype.cfc` | 6 | low | declares announce() (5 candidates) | `system/cache/providers/BoxLangProvider.cfc:370` method 'getEventManager' has no component return type (chain to 'announce') |
| 10 | `method 'flattenAttributes' has no component return type → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/modules/HTMLHelper/models/HTMLHelper.cfc:416` method 'flattenAttributes' has no component return type (chain to 'append') |
| 9 | `method 'getRoutes' in Router has no component return type` | 0 | none |  | `tests/specs/web/routing/RouterTest.cfc:262` method 'getRoutes' in Router has no component return type (chain to 'reduce') |
| 8 | `method 'getQueue' has no component return type` | 0 | none |  | `system/async/executors/Executor.cfc:431` method 'getQueue' has no component return type (chain to 'remainingCapacity') |
| 7 | `method 'getScheduler' has no component return type → system/async/tasks/Scheduler.cfc` | 3 | low | declares beforeAnyTask(), afterAnyTask(), onAnyTaskSuccess(), onAnyTaskError(); named like the receiver 'Scheduler' (3 candidates) | `system/async/tasks/ScheduledTask.cfc:706` method 'getScheduler' has no component return type (chain to 'beforeAnyTask') |
| 5 | `method 'start' in MixerUtil has no component return type → system/core/dynamic/MixerUtil.cfc` | 1 | medium | declares injectPropertyMixin() | `system/ioc/Injector.cfc:1716` method 'start' in MixerUtil has no component return type (chain to 'injectPropertyMixin') |
| 4 | `method 'getAnnotationValue' has no component return type` | 0 | none |  | `system/ioc/config/Mapping.cfc:747` method 'getAnnotationValue' has no component return type (chain to 'trim') |
| 4 | `method 'getExecutor' in coldbox.system.async.AsyncManager has no component return type → system/async/executors/Executor.cfc` | 8 | medium | declares getStats(); named like the receiver 'Executor' (8 candidates) | `tests/specs/async/tasks/ScheduledExecutorSpec.cfc:85` method 'getExecutor' in coldbox.system.async.AsyncManager has no component return type (chain to 'getStats') |
| 4 | `method 'getInetAddress' has no component return type` | 0 | none |  | `system/core/util/Util.cfc:101` method 'getInetAddress' has no component return type (chain to 'getHostName') |
| 4 | `method 'getLastName' in User has no component return type` | 0 | none |  | `tests/specs/orm-enabled/HTMLHelperSpec.cfc:447` method 'getLastName' in User has no component return type (chain to 'encodeForHTMLAttribute') |
| 4 | `method 'getTaskScheduler' in coldbox.system.cache.CacheFactory has no component return type → system/async/tasks/ScheduledTask.cfc` | 1 | medium | declares delay() | `system/cache/providers/CacheBoxProvider.cfc:243` method 'getTaskScheduler' in coldbox.system.cache.CacheFactory has no component return type (chain to 'delay') |
| 3 | `method 'get' in Future has no component return type → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `tests/specs/async/AsyncManagerSpec.cfc:206` method 'get' in Future has no component return type (chain to 'toString') |
| 3 | `method 'getCacheStats' has no component return type` | 0 | none |  | `system/cache/providers/stats/CFStats.cfc:67` method 'getCacheStats' has no component return type (chain to 'cacheEvictionCount') |
| 3 | `method 'getColdBoxVirtualApp' has no component return type → system/testing/VirtualApp.cfc` | 5 | low | declares startup() (4 candidates) | `system/testing/BaseTestCase.cfc:115` method 'getColdBoxVirtualApp' has no component return type (chain to 'startup') |
| 3 | `method 'getJavaSystem' has no component return type → system/Interceptor.cfc` | 5 | high | declares getProperty(), getEnv() | `system/core/delegates/Env.cfc:15` method 'getJavaSystem' has no component return type (chain to 'getProperty') |
| 3 | `method 'getName' in coldbox.system.ioc.Injector has no component return type` | 0 | none |  | `system/ioc/dsl/ColdBoxDSL.cfc:214` method 'getName' in coldbox.system.ioc.Injector has no component return type (chain to 'listFirst') |
| 3 | `method 'getTaskScheduler' in coldbox.system.logging.LogBox has no component return type → system/async/tasks/ScheduledTask.cfc` | 1 | medium | declares delay() | `system/logging/appenders/RollingFileAppender.cfc:86` method 'getTaskScheduler' in coldbox.system.logging.LogBox has no component return type (chain to 'delay') |
| 3 | `method 'getTimezone' has no component return type → tests/tmp/User.cfc` | 2 | low | declares getId() (2 candidates) | `system/web/tasks/ColdBoxScheduledTask.cfc:329` method 'getTimezone' has no component return type (chain to 'getId') |
| 2 | `method 'getHeaders' in coldbox.system.web.context.Response has no component return type` | 0 | none |  | `tests/specs/web/context/ResponseTest.cfc:49` method 'getHeaders' in coldbox.system.web.context.Response has no component return type (chain to 'len') |
| 2 | `method 'getInterceptors' has no component return type` | 0 | none |  | `system/web/context/InterceptorState.cfc:485` method 'getInterceptors' has no component return type (chain to 'entrySet') |
| 2 | `method 'getInterceptors' has no component return type → system/web/context/InterceptorBuffer.cfc` | 24 | low | declares get() (24 candidates) | `system/web/context/InterceptorState.cfc:280` method 'getInterceptors' has no component return type (chain to 'get') |
| 2 | `method 'getLastResult' in ScheduledTask has no component return type` | 0 | none |  | `tests/specs/async/tasks/ScheduledTaskSpec.cfc:51` method 'getLastResult' in ScheduledTask has no component return type (chain to 'isPresent') |
| 2 | `method 'getSetting' in coldbox.system.web.Controller has no component return type` | 0 | none |  | `system/modules/HTMLHelper/models/HTMLHelper.cfc:2584` method 'getSetting' in coldbox.system.web.Controller has no component return type (chain to 'find') |
| 2 | `method 'getTaskScheduler' in coldbox.system.logging.LogBox has no component return type → system/async/executors/ScheduledExecutor.cfc` | 1 | medium | declares schedule() | `system/logging/AbstractAppender.cfc:321` method 'getTaskScheduler' in coldbox.system.logging.LogBox has no component return type (chain to 'schedule') |
| 1 | `method 'createMock' in testbox.system.MockBox has no component return type` | 1 | none |  | `tests/specs/ioc/aop/MixerTest.cfc:22` method 'createMock' in testbox.system.MockBox has no component return type (chain to '$') |
| 1 | `method 'createMock' in testbox.system.MockBox has no component return type → system/testing/VirtualApp.cfc` | 190 | low | declares init() (176 candidates) | `system/testing/BaseTestCase.cfc:260` method 'createMock' in testbox.system.MockBox has no component return type (chain to 'init') |
| 1 | `method 'createMock' in testbox.system.MockBox has no component return type → tests/specs/ioc/dsl/MyTestingDSL.cfc` | 190 | low | declares init() (176 candidates) | `tests/specs/ioc/aop/MixerTest.cfc:21` method 'createMock' in testbox.system.MockBox has no component return type (chain to 'init') |
| 1 | `method 'get' in coldbox.system.async.time.TimeUnit has no component return type → system/async/time/Duration.cfc` | 1 | medium | declares toSeconds() | `system/async/tasks/ScheduledTask.cfc:1549` method 'get' in coldbox.system.async.time.TimeUnit has no component return type (chain to 'toSeconds') |
| 1 | `method 'getAppenders' in Logger has no component return type` | 0 | none |  | `system/logging/Logger.cfc:377` method 'getAppenders' in Logger has no component return type (chain to 'filter') |
| 1 | `method 'getCWD' has no component return type` | 0 | none |  | `build/Build.cfc:18` method 'getCWD' has no component return type (chain to 'reReplace') |
| 1 | `method 'getCache' has no component return type → system/cache/providers/BoxLangColdBoxProvider.cfc` | 7 | low | declares clearAllEvents() (7 candidates) | `tests/suites/eventCachingCollisions/handlers/general.cfc:1` method 'getCache' has no component return type (chain to 'clearAllEvents') |
| 1 | `method 'getCacheBoxDSL' has no component return type → system/ioc/dsl/CacheBoxDSL.cfc` | 11 | medium | declares process(); named like the receiver 'CacheBoxDSL' (11 candidates) | `system/ioc/Builder.cfc:611` method 'getCacheBoxDSL' has no component return type (chain to 'process') |
| 1 | `method 'getCacheStats' has no component return type → system/cache/providers/BoxLangProvider.cfc` | 15 | low | declares getSize() (14 candidates) | `system/cache/providers/stats/CFStats.cfc:44` method 'getCacheStats' has no component return type (chain to 'getSize') |
| 1 | `method 'getCaches' in CacheBoxConfig has no component return type` | 0 | none |  | `system/cache/CacheFactory.cfc:239` method 'getCaches' in CacheBoxConfig has no component return type (chain to 'each') |
| 1 | `method 'getCategory' in coldbox.system.logging.LogEvent has no component return type → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/logging/appenders/DBAppender.cfc:119` method 'getCategory' in coldbox.system.logging.LogEvent has no component return type (chain to 'isEmpty') |
| 1 | `method 'getClassMappingHelper' has no component return type → system/core/util/BoxLangMappingHelper.cfc` | 4 | low | declares addCustomTagPath() (4 candidates) | `system/core/util/Util.cfc:30` method 'getClassMappingHelper' has no component return type (chain to 'addCustomTagPath') |
| 1 | `method 'getColdBoxDSL' has no component return type → system/ioc/dsl/ColdBoxDSL.cfc` | 11 | medium | declares process(); named like the receiver 'ColdBoxDSL' (11 candidates) | `system/ioc/Builder.cfc:630` method 'getColdBoxDSL' has no component return type (chain to 'process') |
| 1 | `method 'getColdBoxVirtualApp' has no component return type → system/cache/AbstractCacheBoxProvider.cfc` | 15 | low | declares shutdown() (15 candidates) | `tests/resources/BaseIntegrationTest.cfc:92` method 'getColdBoxVirtualApp' has no component return type (chain to 'shutdown') |
| 1 | `method 'getConfig' in coldbox.system.logging.LogBox has no component return type → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/web/config/ApplicationLoader.cfc:43` method 'getConfig' in coldbox.system.logging.LogBox has no component return type (chain to 'toString') |
| 1 | `method 'getConfig' in coldbox.system.logging.LogBox has no component return type → system/logging/config/LogBoxConfig.cfc` | 1 | medium | declares getAllAppenders() | `tests/specs/cache/CacheBoxStandaloneSpec.cfc:39` method 'getConfig' in coldbox.system.logging.LogBox has no component return type (chain to 'getAllAppenders') |
| 1 | `method 'getConfig' in coldbox.system.logging.LogBox has no component return type → test-harness/models/lib/Config.cfc` | 18 | medium | declares getMemento(); named like the receiver 'Config' (16 candidates) | `system/web/config/ApplicationLoader.cfc:42` method 'getConfig' in coldbox.system.logging.LogBox has no component return type (chain to 'getMemento') |
| 1 | `method 'getCurrentView' has no component return type` | 0 | none |  | `system/web/context/RequestContext.cfc:1753` method 'getCurrentView' has no component return type (chain to 'len') |
| 1 | `method 'getDIConstructorArguments' in Mapping has no component return type` | 0 | none |  | `system/ioc/Builder.cfc:244` method 'getDIConstructorArguments' in Mapping has no component return type (chain to 'filter') |
| 1 | `method 'getDetail' has no component return type` | 0 | none |  | `system/web/context/ExceptionBean.cfc:273` method 'getDetail' has no component return type (chain to 'len') |
| 1 | `method 'getExecutor' has no component return type → system/async/executors/Executor.cfc` | 8 | medium | declares getStats(); named like the receiver 'Executor' (8 candidates) | `system/async/AsyncManager.cfc:305` method 'getExecutor' has no component return type (chain to 'getStats') |
| 1 | `method 'getExecutor' in coldbox.system.async.tasks.Scheduler has no component return type → system/async/executors/Executor.cfc` | 16 | medium | declares getName(); named like the receiver 'Executor' (16 candidates) | `tests/specs/async/tasks/SchedulerSpec.cfc:20` method 'getExecutor' in coldbox.system.async.tasks.Scheduler has no component return type (chain to 'getName') |
| 1 | `method 'getFirstData' in coldbox.system.testing.mock.web.MockSSEEmitter has no component return type` | 0 | none |  | `tests/specs/web/context/SSEEmitterTest.cfc:170` method 'getFirstData' in coldbox.system.testing.mock.web.MockSSEEmitter has no component return type (chain to 'listLen') |
| 1 | `method 'getINterceptors' in coldbox.system.web.context.InterceptorState has no component return type → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares size() | `tests/specs/web/context/InterceptorStateTest.cfc:104` method 'getINterceptors' in coldbox.system.web.context.InterceptorState has no component return type (chain to 'size') |
| 1 | `method 'getInstance' has no component return type → system/ioc/IProvider.cfc` | 2 | low | declares $get() (2 candidates) | `system/ioc/Injector.cfc:653` method 'getInstance' has no component return type (chain to '$get') |
| 1 | `method 'getInstance' in coldbox.system.ioc.Injector has no component return type → system/cache/AbstractCacheBoxProvider.cfc` | 15 | low | declares setName() (15 candidates) | `system/web/services/SchedulerService.cfc:113` method 'getInstance' in coldbox.system.ioc.Injector has no component return type (chain to 'setName') |
| 1 | `method 'getInterceptors' in coldbox.system.web.context.InterceptorState has no component return type → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares size() | `tests/specs/web/context/InterceptorStateTest.cfc:39` method 'getInterceptors' in coldbox.system.web.context.InterceptorState has no component return type (chain to 'size') |
| 1 | `method 'getJavaCollections' has no component return type → test-harness/models/PhotosService.cfc` | 3 | low | declares list() (3 candidates) | `system/cache/store/ConcurrentStore.cfc:69` method 'getJavaCollections' has no component return type (chain to 'list') |
| 1 | `method 'getJavaSystem' has no component return type → system/core/delegates/Env.cfc` | 7 | low | declares getEnv() (3 candidates) | `system/core/delegates/Env.cfc:68` method 'getJavaSystem' has no component return type (chain to 'getEnv') |
| 1 | `method 'getLastBusinessDayOfTheMonth' in coldbox.system.async.time.DateTimeHelper has no component return type` | 0 | none |  | `system/async/tasks/ScheduledTask.cfc:615` method 'getLastBusinessDayOfTheMonth' in coldbox.system.async.time.DateTimeHelper has no component return type (chain to 'getDayOfMonth') |
| 1 | `method 'getListeners' in CacheBoxConfig has no component return type` | 0 | none |  | `system/cache/CacheFactory.cfc:264` method 'getListeners' in CacheBoxConfig has no component return type (chain to 'each') |
| 1 | `method 'getLogBoxDSL' has no component return type → system/ioc/dsl/LogBoxDSL.cfc` | 11 | medium | declares process(); named like the receiver 'LogBoxDSL' (11 candidates) | `system/ioc/Builder.cfc:655` method 'getLogBoxDSL' has no component return type (chain to 'process') |
| 1 | `method 'getMessage' has no component return type` | 0 | none |  | `system/web/context/ExceptionBean.cfc:276` method 'getMessage' has no component return type (chain to 'len') |
| 1 | `method 'getMessages' has no component return type` | 0 | none |  | `system/web/context/Response.cfc:206` method 'getMessages' has no component return type (chain to 'toList') |
| 1 | `method 'getMessages' has no component return type → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/web/context/Response.cfc:195` method 'getMessages' has no component return type (chain to 'isEmpty') |
| 1 | `method 'getNewDIDefinition' has no component return type → system/web/context/InterceptorBuffer.cfc` | 3 | low | declares append() (2 candidates) | `system/ioc/config/Mapping.cfc:463` method 'getNewDIDefinition' has no component return type (chain to 'append') |
| 1 | `method 'getPath' in coldbox.system.ioc.config.Mapping has no component return type → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/ioc/Injector.cfc:683` method 'getPath' in coldbox.system.ioc.config.Mapping has no component return type (chain to 'toString') |
| 1 | `method 'getProperties' has no component return type → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/logging/appenders/SocketAppender.cfc:94` method 'getProperties' has no component return type (chain to 'toString') |
| 1 | `method 'getRenderData' has no component return type → system/web/flash/AbstractFlashScope.cfc` | 1 | medium | declares isEmpty() | `system/web/context/RequestContext.cfc:1753` method 'getRenderData' has no component return type (chain to 'isEmpty') |
| 1 | `method 'getScope' has no component return type → system/ioc/config/Binder.cfc` | 1 | medium | declares map() | `system/web/flash/AbstractFlashScope.cfc:352` method 'getScope' has no component return type (chain to 'map') |
| 1 | `method 'getScope' has no component return type → system/ioc/scopes/RequestScope.cfc` | 18 | low | declares clear() (17 candidates) | `system/ioc/Injector.cfc:1073` method 'getScope' has no component return type (chain to 'clear') |
| 1 | `method 'getScope' has no component return type → system/ioc/scopes/Singleton.cfc` | 1 | medium | declares clearAppOnly() | `system/ioc/Injector.cfc:1081` method 'getScope' has no component return type (chain to 'clearAppOnly') |
| 1 | `method 'getScopeRegistration' in coldbox.system.cache.CacheFactory has no component return type → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/cache/report/skins/default/CachePanel.cfm:70` method 'getScopeRegistration' in coldbox.system.cache.CacheFactory has no component return type (chain to 'toString') |
| 1 | `method 'getSetting' in coldbox.system.web.Controller has no component return type → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `system/web/services/RequestService.cfc:264` method 'getSetting' in coldbox.system.web.Controller has no component return type (chain to 'toString') |
| 1 | `method 'getSetting' in coldbox.system.web.Controller has no component return type → system/web/context/InterceptorBuffer.cfc` | 1 | medium | declares length() | `system/exceptions/Whoops.cfm:218` method 'getSetting' in coldbox.system.web.Controller has no component return type (chain to 'length') |
| 1 | `method 'getStackTrace' has no component return type` | 0 | none |  | `system/web/context/ExceptionBean.cfc:279` method 'getStackTrace' has no component return type (chain to 'len') |
| 1 | `method 'getTagContextAsString' has no component return type` | 0 | none |  | `system/web/context/ExceptionBean.cfc:282` method 'getTagContextAsString' has no component return type (chain to 'len') |
| 1 | `method 'getTaskRecord' in coldbox.system.async.tasks.Scheduler has no component return type → tests/resources/Test.cfc` | 16 | low | declares getName() (16 candidates) | `tests/specs/async/tasks/SchedulerSpec.cfc:32` method 'getTaskRecord' in coldbox.system.async.tasks.Scheduler has no component return type (chain to 'getName') |
| 1 | `method 'getTaskScheduler' in coldbox.system.cache.CacheFactory has no component return type → system/async/executors/ScheduledExecutor.cfc` | 1 | medium | declares newTask() | `system/cache/providers/CacheBoxProvider.cfc:238` method 'getTaskScheduler' in coldbox.system.cache.CacheFactory has no component return type (chain to 'newTask') |
| 1 | `method 'getTimezone' in ScheduledTask has no component return type → system/async/time/Duration.cfc` | 2 | low | declares toString() (2 candidates) | `tests/specs/async/tasks/ScheduledTaskSpec.cfc:37` method 'getTimezone' in ScheduledTask has no component return type (chain to 'toString') |
| 1 | `method 'getType' has no component return type` | 0 | none |  | `system/web/context/ExceptionBean.cfc:270` method 'getType' has no component return type (chain to 'len') |
| 1 | `method 'getUtility' has no component return type → system/cache/AbstractCacheBoxProvider.cfc` | 2 | low | declares inThread() (2 candidates) | `system/cache/AbstractCacheBoxProvider.cfc:744` method 'getUtility' has no component return type (chain to 'inThread') |
| 1 | `method 'getXmlConverter' has no component return type → system/core/conversion/XMLConverter.cfc` | 1 | high | declares toXML(); named like the receiver 'XmlConverter' | `system/logging/LogEvent.cfc:121` method 'getXmlConverter' has no component return type (chain to 'toXML') |
| 1 | `method 'locateInstance' has no component return type` | 0 | none |  | `system/ioc/Injector.cfc:758` method 'locateInstance' has no component return type (chain to 'len') |
| 1 | `method 'renderdata' in coldbox.system.web.context.RequestContext has no component return type → system/web/context/RequestContext.cfc` | 2 | medium | declares noExecution() | `system/web/services/RoutingService.cfc:844` method 'renderdata' in coldbox.system.web.context.RequestContext has no component return type (chain to 'noExecution') |
| 1 | `method 'routeDefinitionShape' has no component return type` | 0 | none |  | `system/web/routing/Router.cfc:1218` method 'routeDefinitionShape' has no component return type (chain to 'keyArray') |
| 1 | `method 'unless' has no component return type → system/FrameworkSupertype.cfc` | 6 | low | declares when() (4 candidates) | `test-harness/handlers/main.cfc:59` method 'unless' has no component return type (chain to 'when') |

<details><summary>Groups with several candidates</summary>

- `method 'getEventManager' has no component return type → system/FrameworkSupertype.cfc` — 14 finding(s), 5 candidate(s):
  - low `system/FrameworkSupertype.cfc` — declares announce()
  - low `system/remote/ColdboxProxy.cfc` — declares announce()
  - low `system/testing/BaseTestCase.cfc` — declares announce()
  - low `system/core/events/EventPoolManager.cfc` — declares announce()
  - low `system/web/services/InterceptorService.cfc` — declares announce()
- `method 'flattenAttributes' has no component return type → system/web/context/InterceptorBuffer.cfc` — 10 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `method 'getScheduler' has no component return type → system/async/tasks/Scheduler.cfc` — 7 finding(s), 3 candidate(s):
  - low `system/async/tasks/Scheduler.cfc` — declares beforeAnyTask(), afterAnyTask(), onAnyTaskSuccess(), onAnyTaskError(); named like the receiver 'Scheduler'
  - low `test-harness/config/Scheduler.cfc` — declares beforeAnyTask(), afterAnyTask(), onAnyTaskSuccess(), onAnyTaskError(); named like the receiver 'Scheduler'
  - low `tests/suites/loadtests/beload/config/Scheduler.cfc` — declares beforeAnyTask(), afterAnyTask(), onAnyTaskSuccess(), onAnyTaskError(); named like the receiver 'Scheduler'
- `method 'getExecutor' in coldbox.system.async.AsyncManager has no component return type → system/async/executors/Executor.cfc` — 4 finding(s), 8 candidate(s):
  - medium `system/async/executors/Executor.cfc` — declares getStats(); named like the receiver 'Executor'
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getStats()
  - low `system/async/tasks/ScheduledTask.cfc` — declares getStats()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares getStats()
  - low `system/cache/providers/CFProvider.cfc` — declares getStats()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares getStats()
  - low `system/cache/providers/ICacheProvider.cfc` — declares getStats()
  - low `system/cache/providers/LuceeProvider.cfc` — declares getStats()
- `method 'get' in Future has no component return type → system/async/time/Duration.cfc` — 3 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `method 'getColdBoxVirtualApp' has no component return type → system/testing/VirtualApp.cfc` — 3 finding(s), 4 candidate(s):
  - low `system/testing/VirtualApp.cfc` — declares startup()
  - low `system/web/Renderer.cfc` — declares startup()
  - low `system/async/tasks/Scheduler.cfc` — declares startup()
  - low `system/web/routing/Router.cfc` — declares startup()
- `method 'getTimezone' has no component return type → tests/tmp/User.cfc` — 3 finding(s), 2 candidate(s):
  - low `tests/tmp/User.cfc` — declares getId()
  - low `test-harness/models/entities/User.cfc` — declares getId()
- `method 'getInterceptors' has no component return type → system/web/context/InterceptorBuffer.cfc` — 2 finding(s), 24 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares get()
  - low `system/web/flash/AbstractFlashScope.cfc` — declares get()
  - low `system/web/routing/Router.cfc` — declares get()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares get()
  - low `system/testing/BaseTestCase.cfc` — declares get()
  - low `system/async/tasks/Future.cfc` — declares get()
  - low `system/async/tasks/FutureTask.cfc` — declares get()
  - low `system/async/time/Duration.cfc` — declares get()
  - … 16 more
- `method 'createMock' in testbox.system.MockBox has no component return type → system/testing/VirtualApp.cfc` — 1 finding(s), 176 candidate(s):
  - low `system/testing/VirtualApp.cfc` — declares init()
  - low `system/Bootstrap.cfc` — declares init()
  - low `system/EventHandler.cfc` — declares init()
  - low `system/FrameworkSupertype.cfc` — declares init()
  - low `system/Interceptor.cfc` — declares init()
  - low `system/aop/Matcher.cfc` — declares init()
  - low `system/aop/MethodInvocation.cfc` — declares init()
  - low `system/aop/MixerUtil.cfc` — declares init()
  - … 168 more
- `method 'createMock' in testbox.system.MockBox has no component return type → tests/specs/ioc/dsl/MyTestingDSL.cfc` — 1 finding(s), 176 candidate(s):
  - low `tests/specs/ioc/dsl/MyTestingDSL.cfc` — declares init()
  - low `tests/resources/ChildInjectorSample.cfc` — declares init()
  - low `tests/resources/Test.cfc` — declares init()
  - low `tests/resources/VirtualParentClass.cfc` — declares init()
  - low `tests/tmp/Person.cfc` — declares init()
  - low `tests/tmp/Ref1.cfc` — declares init()
  - low `tests/tmp/Ref2.cfc` — declares init()
  - low `build/Build.cfc` — declares init()
  - … 168 more
- `method 'getCache' has no component return type → system/cache/providers/BoxLangColdBoxProvider.cfc` — 1 finding(s), 7 candidate(s):
  - low `system/cache/providers/BoxLangColdBoxProvider.cfc` — declares clearAllEvents()
  - low `system/cache/providers/CFColdBoxProvider.cfc` — declares clearAllEvents()
  - low `system/cache/providers/CacheBoxColdBoxProvider.cfc` — declares clearAllEvents()
  - low `system/cache/providers/IColdBoxProvider.cfc` — declares clearAllEvents()
  - low `system/cache/providers/LuceeColdboxProvider.cfc` — declares clearAllEvents()
  - low `system/cache/providers/MockProvider.cfc` — declares clearAllEvents()
  - low `system/cache/util/ElementCleaner.cfc` — declares clearAllEvents()
- `method 'getCacheBoxDSL' has no component return type → system/ioc/dsl/CacheBoxDSL.cfc` — 1 finding(s), 11 candidate(s):
  - medium `system/ioc/dsl/CacheBoxDSL.cfc` — declares process(); named like the receiver 'CacheBoxDSL'
  - low `system/ioc/config/Binder.cfc` — declares process()
  - low `system/ioc/config/Mapping.cfc` — declares process()
  - low `system/ioc/dsl/ColdBoxDSL.cfc` — declares process()
  - low `system/ioc/dsl/IDSLBuilder.cfc` — declares process()
  - low `system/ioc/dsl/LogBoxDSL.cfc` — declares process()
  - low `system/remote/ColdboxProxy.cfc` — declares process()
  - low `system/core/events/EventPool.cfc` — declares process()
  - … 3 more
- `method 'getCacheStats' has no component return type → system/cache/providers/BoxLangProvider.cfc` — 1 finding(s), 14 candidate(s):
  - low `system/cache/providers/BoxLangProvider.cfc` — declares getSize()
  - low `system/cache/providers/CFProvider.cfc` — declares getSize()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares getSize()
  - low `system/cache/providers/ICacheProvider.cfc` — declares getSize()
  - low `system/cache/providers/LuceeProvider.cfc` — declares getSize()
  - low `system/cache/providers/MockProvider.cfc` — declares getSize()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getSize()
  - low `system/cache/store/BlackHoleStore.cfc` — declares getSize()
  - … 6 more
- `method 'getClassMappingHelper' has no component return type → system/core/util/BoxLangMappingHelper.cfc` — 1 finding(s), 4 candidate(s):
  - low `system/core/util/BoxLangMappingHelper.cfc` — declares addCustomTagPath()
  - low `system/core/util/CFMappingHelper.cfc` — declares addCustomTagPath()
  - low `system/core/util/LuceeMappingHelper.cfc` — declares addCustomTagPath()
  - low `system/core/util/Util.cfc` — declares addCustomTagPath()
- `method 'getColdBoxDSL' has no component return type → system/ioc/dsl/ColdBoxDSL.cfc` — 1 finding(s), 11 candidate(s):
  - medium `system/ioc/dsl/ColdBoxDSL.cfc` — declares process(); named like the receiver 'ColdBoxDSL'
  - low `system/ioc/config/Binder.cfc` — declares process()
  - low `system/ioc/config/Mapping.cfc` — declares process()
  - low `system/ioc/dsl/CacheBoxDSL.cfc` — declares process()
  - low `system/ioc/dsl/IDSLBuilder.cfc` — declares process()
  - low `system/ioc/dsl/LogBoxDSL.cfc` — declares process()
  - low `system/remote/ColdboxProxy.cfc` — declares process()
  - low `system/core/events/EventPool.cfc` — declares process()
  - … 3 more
- `method 'getColdBoxVirtualApp' has no component return type → system/cache/AbstractCacheBoxProvider.cfc` — 1 finding(s), 15 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares shutdown()
  - low `system/cache/CacheFactory.cfc` — declares shutdown()
  - low `system/ioc/IInjector.cfc` — declares shutdown()
  - low `system/ioc/Injector.cfc` — declares shutdown()
  - low `system/logging/AbstractAppender.cfc` — declares shutdown()
  - low `system/logging/LogBox.cfc` — declares shutdown()
  - low `system/testing/VirtualApp.cfc` — declares shutdown()
  - low `system/async/executors/Executor.cfc` — declares shutdown()
  - … 7 more
- `method 'getConfig' in coldbox.system.logging.LogBox has no component return type → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `method 'getConfig' in coldbox.system.logging.LogBox has no component return type → test-harness/models/lib/Config.cfc` — 1 finding(s), 16 candidate(s):
  - medium `test-harness/models/lib/Config.cfc` — declares getMemento(); named like the receiver 'Config'
  - low `system/web/Controller.cfc` — declares getMemento()
  - low `system/web/context/ExceptionBean.cfc` — declares getMemento()
  - low `system/web/context/RequestContext.cfc` — declares getMemento()
  - low `system/web/context/Response.cfc` — declares getMemento()
  - low `system/web/routing/Router.cfc` — declares getMemento()
  - low `system/aop/Matcher.cfc` — declares getMemento()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getMemento()
  - … 8 more
- `method 'getExecutor' has no component return type → system/async/executors/Executor.cfc` — 1 finding(s), 8 candidate(s):
  - medium `system/async/executors/Executor.cfc` — declares getStats(); named like the receiver 'Executor'
  - low `system/async/tasks/ScheduledTask.cfc` — declares getStats()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getStats()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares getStats()
  - low `system/cache/providers/CFProvider.cfc` — declares getStats()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares getStats()
  - low `system/cache/providers/ICacheProvider.cfc` — declares getStats()
  - low `system/cache/providers/LuceeProvider.cfc` — declares getStats()
- `method 'getExecutor' in coldbox.system.async.tasks.Scheduler has no component return type → system/async/executors/Executor.cfc` — 1 finding(s), 16 candidate(s):
  - medium `system/async/executors/Executor.cfc` — declares getName(); named like the receiver 'Executor'
  - low `tests/resources/Test.cfc` — declares getName()
  - low `tests/specs/ioc/dsl/MyTestingDSL.cfc` — declares getName()
  - low `tests/tmp/Person.cfc` — declares getName()
  - low `tests/tmp/Ref1.cfc` — declares getName()
  - low `tests/tmp/Ref2.cfc` — declares getName()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getName()
  - low `system/ioc/Injector.cfc` — declares getName()
  - … 8 more
- `method 'getInstance' has no component return type → system/ioc/IProvider.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/ioc/IProvider.cfc` — declares $get()
  - low `system/ioc/Provider.cfc` — declares $get()
- `method 'getInstance' in coldbox.system.ioc.Injector has no component return type → system/cache/AbstractCacheBoxProvider.cfc` — 1 finding(s), 15 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares setName()
  - low `system/ioc/Injector.cfc` — declares setName()
  - low `system/ioc/Provider.cfc` — declares setName()
  - low `system/logging/AbstractAppender.cfc` — declares setName()
  - low `system/async/executors/Executor.cfc` — declares setName()
  - low `system/async/tasks/ScheduledTask.cfc` — declares setName()
  - low `system/async/tasks/Scheduler.cfc` — declares setName()
  - low `system/cache/providers/ICacheProvider.cfc` — declares setName()
  - … 7 more
- `method 'getJavaCollections' has no component return type → test-harness/models/PhotosService.cfc` — 1 finding(s), 3 candidate(s):
  - low `test-harness/models/PhotosService.cfc` — declares list()
  - low `test-harness/modules_app/resourcesTest/models/PhotosService.cfc` — declares list()
  - low `tests/perf-harness/app/handlers/Api.cfc` — declares list()
- `method 'getJavaSystem' has no component return type → system/core/delegates/Env.cfc` — 1 finding(s), 3 candidate(s):
  - low `system/core/delegates/Env.cfc` — declares getEnv()
  - low `system/FrameworkSupertype.cfc` — declares getEnv()
  - low `system/testing/BaseTestCase.cfc` — declares getEnv()
- `method 'getLogBoxDSL' has no component return type → system/ioc/dsl/LogBoxDSL.cfc` — 1 finding(s), 11 candidate(s):
  - medium `system/ioc/dsl/LogBoxDSL.cfc` — declares process(); named like the receiver 'LogBoxDSL'
  - low `system/ioc/config/Binder.cfc` — declares process()
  - low `system/ioc/config/Mapping.cfc` — declares process()
  - low `system/ioc/dsl/CacheBoxDSL.cfc` — declares process()
  - low `system/ioc/dsl/ColdBoxDSL.cfc` — declares process()
  - low `system/ioc/dsl/IDSLBuilder.cfc` — declares process()
  - low `system/remote/ColdboxProxy.cfc` — declares process()
  - low `system/core/events/EventPool.cfc` — declares process()
  - … 3 more
- `method 'getNewDIDefinition' has no component return type → system/web/context/InterceptorBuffer.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/web/context/InterceptorBuffer.cfc` — declares append()
  - low `system/web/routing/Router.cfc` — declares append()
- `method 'getPath' in coldbox.system.ioc.config.Mapping has no component return type → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `method 'getProperties' has no component return type → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `method 'getScope' has no component return type → system/ioc/scopes/RequestScope.cfc` — 1 finding(s), 17 candidate(s):
  - low `system/ioc/scopes/RequestScope.cfc` — declares clear()
  - low `system/ioc/scopes/Singleton.cfc` — declares clear()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares clear()
  - low `system/cache/providers/BoxLangProvider.cfc` — declares clear()
  - low `system/cache/providers/CFProvider.cfc` — declares clear()
  - low `system/cache/providers/CacheBoxProvider.cfc` — declares clear()
  - low `system/cache/providers/ICacheProvider.cfc` — declares clear()
  - low `system/cache/providers/LuceeProvider.cfc` — declares clear()
  - … 9 more
- `method 'getScopeRegistration' in coldbox.system.cache.CacheFactory has no component return type → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `method 'getSetting' in coldbox.system.web.Controller has no component return type → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `method 'getTaskRecord' in coldbox.system.async.tasks.Scheduler has no component return type → tests/resources/Test.cfc` — 1 finding(s), 16 candidate(s):
  - low `tests/resources/Test.cfc` — declares getName()
  - low `tests/specs/ioc/dsl/MyTestingDSL.cfc` — declares getName()
  - low `tests/tmp/Person.cfc` — declares getName()
  - low `tests/tmp/Ref1.cfc` — declares getName()
  - low `tests/tmp/Ref2.cfc` — declares getName()
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares getName()
  - low `system/ioc/Injector.cfc` — declares getName()
  - low `system/ioc/Provider.cfc` — declares getName()
  - … 8 more
- `method 'getTimezone' in ScheduledTask has no component return type → system/async/time/Duration.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc` — declares toString()
  - low `system/async/time/Period.cfc` — declares toString()
- `method 'getUtility' has no component return type → system/cache/AbstractCacheBoxProvider.cfc` — 1 finding(s), 2 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc` — declares inThread()
  - low `system/core/util/Util.cfc` — declares inThread()
- `method 'unless' has no component return type → system/FrameworkSupertype.cfc` — 1 finding(s), 4 candidate(s):
  - low `system/FrameworkSupertype.cfc` — declares when()
  - low `test-harness/models/delegates/FlowHelpers.cfc` — declares when()
  - low `system/async/tasks/ScheduledTask.cfc` — declares when()
  - low `system/core/delegates/Flow.cfc` — declares when()

</details>

## Method definitions — a method not found where it was looked for

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 17 | `cache` | 1 | medium | declares cache | `system/cache/providers/BoxLangProvider.cfc:212` not found in extends chain |
| 5 | `command` | 1 | none |  | `tests/perf-harness/PerformanceSuite.cfc:266` no qualifier, not in file |
| 4 | `getobjectstate` | 1 | medium | declares getObjectState; beside the calling file | `tests/suites/async/performance-parallel-tests.cfm:58` method 'getObjectState' not found in tests.tmp.User |
| 4 | `getrenderedcontent` | 1 | medium | declares getRenderedContent | `tests/specs/integration/EventExecutionsSpec.cfc:70` method 'getRenderedContent' not found in coldbox.system.web.context.RequestContext |
| 4 | `getsetting` | 3 | low | declares getSetting (2 candidates) | `system/modules/HTMLHelper/models/HTMLHelper.cfc:87` no qualifier, not in file |
| 3 | `getnormalizedid` | 1 | medium | declares getNormalizedId | `system/cache/store/indexers/JDBCMetadataIndexer.cfc:98` method 'getNormalizedID' not found in coldbox.system.cache.store.indexers.JDBCMetadataIndexer |
| 3 | `injectstate` | 1 | medium | declares injectState; beside the calling file | `tests/suites/async/performance-parallel-tests.cfm:30` method 'injectState' not found in tests.tmp.User |
| 3 | `isinstancecheck` | 1 | medium | declares isInstanceCheck; beside the calling file | `tests/specs/core/util/UtilTest.cfc:10` method 'isInstanceCheck' not found in coldbox.system.core.util.Util |
| 3 | `keyarray` | 0 | none |  | `system/web/flash/AbstractFlashScope.cfc:115` method 'filter' not found in coldbox.system.web.flash.AbstractFlashScope (chain to 'keyArray') |
| 2 | `each` | 0 | none |  | `system/web/flash/AbstractFlashScope.cfc:116` method 'filter' not found in coldbox.system.web.flash.AbstractFlashScope (chain to 'each') |
| 2 | `getappstarthandlerfired` | 0 | none |  | `tests/specs/web/ControllerTest.cfc:75` method 'getAppStartHandlerFired' not found in coldbox.system.web.Controller |
| 2 | `getcategories` | 1 | low | declares getConfig (4 candidates) | `tests/specs/logging/LogBoxBuildTests.cfc:34` method 'getConfig' not found in LogBox (chain to 'getCategories') |
| 2 | `getclassmetadata` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:583` no qualifier, not in file |
| 2 | `getvalue` | 3 | low | declares getValue (2 candidates) | `system/testing/BaseTestCase.cfc:754` not found in extends chain |
| 2 | `println` | 1 | none |  | `system/async/executors/Executor.cfc:685` no qualifier, not in file |
| 2 | `setbaseurl` | 1 | medium | declares setBaseURL | `tests/suites/eventCachingCollisions/config/routes.cfm:41` no qualifier, not in file |
| 2 | `setisactive` | 2 | low | declares setIsActive (2 candidates) | `tests/specs/orm-enabled/HTMLHelperSpec.cfc:528` method 'setIsActive' not found in User |
| 2 | `sse` | 2 | medium | declares sse; beside the calling file | `system/web/context/SSEStreamer.cfc:36` no qualifier, not in file |
| 2 | `stable` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:500` no qualifier, not in file |
| 2 | `start` | 3 | low | declares start (3 candidates) | `tests/perf-harness/PerformanceSuite.cfc:481` no qualifier, not in file |
| 2 | `time` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:498` no qualifier, not in file |
| 2 | `view` | 4 | low | declares view (2 candidates) | `test-harness/external/testLayouts/ext.cfm:45` no qualifier, not in file |
| 1 | `_ucfirst` | 0 | none |  | `system/core/delegates/StringUtil.cfc:127` no qualifier, not in file |
| 1 | `addasset` | 2 | low | declares addAsset (2 candidates) | `test-harness/external/testLayouts/ext.cfm:1` no qualifier, not in file |
| 1 | `addroute` | 1 | medium | declares addRoute | `tests/suites/eventCachingCollisions/config/routes.cfm:111` no qualifier, not in file |
| 1 | `addto` | 2 | low | declares addTo; beside the calling file (2 candidates) | `system/async/time/Duration.cfc:227` method 'addTo' not found on builtin native.duration |
| 1 | `aiagent` | 0 | none |  | `test-harness/config/Router.cfc:20` not found in extends chain |
| 1 | `aitool` | 0 | none |  | `test-harness/config/Router.cfc:13` not found in extends chain |
| 1 | `be` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:500` no qualifier, not in file |
| 1 | `clearstatistics` | 10 | low | declares clearStatistics; beside the calling file (10 candidates) | `system/cache/AbstractCacheBoxProvider.cfc:155` method 'clearStatistics' not found on builtin native.string |
| 1 | `debug` | 5 | low | declares debug (3 candidates) | `system/testing/CustomMatchers.cfc:134` no qualifier, not in file |
| 1 | `edge` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:733` no qualifier, not in file |
| 1 | `expect` | 1 | none |  | `system/testing/CustomMatchers.cfc:89` no qualifier, not in file |
| 1 | `get` | 24 | low | declares get; beside the calling file (24 candidates) | `system/async/time/Duration.cfc:95` method 'get' not found on builtin native.duration |
| 1 | `getallclassmetadata` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:612` method 'getAllClassMetadata' not found on builtin native.sessionfactory |
| 1 | `getallentitynames` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:609` method 'getMetaModel' not found on builtin native.sessionfactory (chain to 'getAllEntityNames') |
| 1 | `getinstance` | 9 | low | declares getInstance (7 candidates) | `system/modules/HTMLHelper/helpers/Mixins.cfm:9` no qualifier, not in file |
| 1 | `getmetamodel` | 0 | none |  | `system/core/dynamic/ObjectPopulator.cfc:609` method 'getMetaModel' not found on builtin native.sessionfactory |
| 1 | `getmodulelist` | 0 | none |  | `system/web/routing/Router.cfc:2102` no qualifier, not in file |
| 1 | `getprivatevalue` | 2 | medium | declares getPrivateValue | `system/testing/BaseTestCase.cfc:772` not found in extends chain |
| 1 | `getrandom` | 1 | medium | declares getRandom | `test-harness/modules_app/resourcesTest/config/Scheduler.cfc:9` method 'getRandom' not found in PhotosService |
| 1 | `getunits` | 2 | low | declares getUnits; beside the calling file (2 candidates) | `system/async/time/Duration.cfc:116` method 'getUnits' not found on builtin native.duration |
| 1 | `getvaluesss` | 0 | none |  | `test-harness/handlers/testerror.cfc:32` method 'getValuesss' not found in coldbox.system.web.context.RequestContext |
| 1 | `includeroutes` | 0 | none |  | `system/web/routing/Router.cfc:353` no qualifier, not in file |
| 1 | `influence` | 0 | none |  | `system/ioc/config/Binder.cfc:521` no qualifier, not in file |
| 1 | `keys` | 0 | none |  | `system/cache/store/ConcurrentStore.cfc:69` method 'keys' not found on builtin native.map |
| 1 | `mcpserver` | 0 | none |  | `test-harness/config/Router.cfc:11` not found in extends chain |
| 1 | `objectdeserialize` | 0 | none |  | `system/core/conversion/ObjectMarshaller.cfc:82` no qualifier, not in file |
| 1 | `objectserialize` | 0 | none |  | `system/core/conversion/ObjectMarshaller.cfc:63` no qualifier, not in file |
| 1 | `of` | 2 | low | declares of; beside the calling file (2 candidates) | `system/async/time/Duration.cfc:435` method 'of' not found on builtin native.duration |
| 1 | `ormgethibernateversion` | 0 | none |  | `system/core/util/Util.cfc:503` no qualifier, not in file |
| 1 | `parseandload` | 0 | none |  | `system/web/services/LoaderService.cfc:233` method 'parseAndLoad' not found in coldbox.system.cache.config.CacheBoxConfig |
| 1 | `plusnanos` | 1 | medium | declares plusNanos | `system/web/tasks/ColdBoxScheduledTask.cfc:351` method 'plusNanos' not found on builtin native.localdatetime |
| 1 | `populate` | 4 | low | declares populate (3 candidates) | `tests/specs/core/delegates/PopulationSpec.cfc:25` not found in extends chain |
| 1 | `rendercachecontentreport` | 1 | medium | declares renderCacheContentReport | `system/cache/report/skins/default/CacheReport.cfm:125` no qualifier, not in file |
| 1 | `rendercachereport` | 1 | medium | declares renderCacheReport | `system/cache/report/skins/default/CachePanel.cfm:96` no qualifier, not in file |
| 1 | `response` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:483` no qualifier, not in file |
| 1 | `rps` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:750` no qualifier, not in file |
| 1 | `run` | 80 | low | declares run (77 candidates) | `tests/index.cfm:29` no qualifier, not in file |
| 1 | `setappstarthandlerfired` | 0 | none |  | `tests/specs/web/ControllerTest.cfc:76` method 'setAppStartHandlerFired' not found in coldbox.system.web.Controller |
| 1 | `setdelegates` | 1 | medium | declares setDelegates; beside the calling file | `system/ioc/config/Binder.cfc:927` method 'setDelegates' not found on builtin native.array |
| 1 | `setenabled` | 2 | low | declares setEnabled (2 candidates) | `tests/suites/eventCachingCollisions/config/routes.cfm:23` no qualifier, not in file |
| 1 | `setlogbox` | 9 | low | declares setLogBox; beside the calling file (8 candidates) | `system/logging/LogBox.cfc:354` no qualifier, not in file |
| 1 | `showdebugpanel` | 0 | none |  | `tests/suites/eventCachingCollisions/layouts/Layout.None.cfm:2` method 'showdebugpanel' not found in coldbox.system.web.context.RequestContext |
| 1 | `size` | 1 | medium | declares size | `system/cache/store/indexers/JDBCMetadataIndexer.cfc:204` method 'size' not found in coldbox.system.cache.store.indexers.JDBCMetadataIndexer |
| 1 | `subtractfrom` | 2 | low | declares subtractFrom; beside the calling file (2 candidates) | `system/async/time/Duration.cfc:240` method 'subtractFrom' not found on builtin native.duration |
| 1 | `this[]` | 0 | none |  | `system/RestHandler.cfc:541` not found in extends chain |
| 1 | `todayspart` | 1 | medium | declares toDaysPart; beside the calling file | `system/async/time/Duration.cfc:132` method 'toDaysPart' not found on builtin native.duration |
| 1 | `tohourspart` | 1 | medium | declares toHoursPart; beside the calling file | `system/async/time/Duration.cfc:146` method 'toHoursPart' not found on builtin native.duration |
| 1 | `tojson` | 3 | low | declares toJson (2 candidates) | `system/modules/HTMLHelper/models/HTMLHelper.cfc:2492` no qualifier, not in file |
| 1 | `tomillispart` | 1 | medium | declares toMillisPart; beside the calling file | `system/async/time/Duration.cfc:160` method 'toMillisPart' not found on builtin native.duration |
| 1 | `tominutespart` | 1 | medium | declares toMinutesPart; beside the calling file | `system/async/time/Duration.cfc:174` method 'toMinutesPart' not found on builtin native.duration |
| 1 | `tonanos` | 1 | medium | declares toNanos; beside the calling file | `system/async/time/Duration.cfc:181` method 'toNanos' not found on builtin native.duration |
| 1 | `tonanospart` | 1 | medium | declares toNanosPart; beside the calling file | `system/async/time/Duration.cfc:188` method 'toNanosPart' not found on builtin native.duration |
| 1 | `tosecondspart` | 1 | medium | declares toSecondsPart; beside the calling file | `system/async/time/Duration.cfc:202` method 'toSecondsPart' not found on builtin native.duration |
| 1 | `total` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:483` no qualifier, not in file |

<details><summary>Groups with several candidates</summary>

- `getsetting` — 4 finding(s), 2 candidate(s):
  - low `system/FrameworkSupertype.cfc:317` — declares getSetting
  - low `system/web/Controller.cfc:272` — declares getSetting
- `getcategories` — 2 finding(s), 4 candidate(s):
  - low `system/cache/CacheFactory.cfc:26` — declares getConfig
  - low `system/logging/LogBox.cfc:42` — declares getConfig
  - low `system/ioc/config/Binder.cfc:43` — declares getConfig
  - low `system/cache/store/indexers/JDBCMetadataIndexer.cfc:26` — declares getConfig
- `getvalue` — 2 finding(s), 2 candidate(s):
  - low `system/ioc/config/Mapping.cfc:41` — declares getValue
  - low `system/web/context/RequestContext.cfc:416` — declares getValue
- `setisactive` — 2 finding(s), 2 candidate(s):
  - low `tests/tmp/Person.cfc:11` — declares setIsActive
  - low `test-harness/models/entities/User.cfc:25` — declares setIsActive
- `start` — 2 finding(s), 3 candidate(s):
  - low `system/async/tasks/ScheduledTask.cfc:807` — declares start
  - low `system/core/dynamic/MixerUtil.cfc:35` — declares start
  - low `system/web/tasks/ColdBoxScheduledTask.cfc:255` — declares start
- `view` — 2 finding(s), 2 candidate(s):
  - low `system/FrameworkSupertype.cfc:173` — declares view
  - low `system/web/Renderer.cfc:189` — declares view
- `addasset` — 1 finding(s), 2 candidate(s):
  - low `system/modules/HTMLHelper/helpers/Mixins.cfm:8` — declares addAsset
  - low `system/modules/HTMLHelper/models/HTMLHelper.cfc:77` — declares addAsset
- `addto` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc:226` — declares addTo; beside the calling file
  - low `system/async/time/Period.cfc:172` — declares addTo; beside the calling file
- `clearstatistics` — 1 finding(s), 10 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc:154` — declares clearStatistics; beside the calling file
  - low `system/cache/providers/BoxLangProvider.cfc:211` — declares clearStatistics
  - low `system/cache/providers/CFProvider.cfc:132` — declares clearStatistics
  - low `system/cache/providers/ICacheProvider.cfc:48` — declares clearStatistics
  - low `system/cache/providers/LuceeProvider.cfc:95` — declares clearStatistics
  - low `system/cache/util/CacheStats.cfc:86` — declares clearStatistics
  - low `system/cache/util/IStats.cfc:27` — declares clearStatistics
  - low `system/cache/providers/stats/BoxLangStats.cfc:47` — declares clearStatistics
  - … 2 more
- `debug` — 1 finding(s), 3 candidate(s):
  - low `system/logging/Logger.cfc:270` — declares debug
  - low `system/async/tasks/ScheduledTask.cfc:380` — declares debug
  - low `system/logging/config/LogBoxConfig.cfc:398` — declares debug
- `get` — 1 finding(s), 24 candidate(s):
  - low `system/async/time/Duration.cfc:94` — declares get; beside the calling file
  - low `system/async/time/Period.cfc:101` — declares get; beside the calling file
  - low `system/async/time/TimeUnit.cfc:18` — declares get; beside the calling file
  - low `system/async/tasks/Future.cfc:217` — declares get
  - low `system/async/tasks/FutureTask.cfc:53` — declares get
  - low `system/cache/AbstractCacheBoxProvider.cfc:304` — declares get
  - low `system/testing/BaseTestCase.cfc:639` — declares get
  - low `system/cache/providers/BoxLangProvider.cfc:269` — declares get
  - … 16 more
- `getinstance` — 1 finding(s), 7 candidate(s):
  - low `system/FrameworkSupertype.cfc:88` — declares getInstance
  - low `system/ioc/IInjector.cfc:35` — declares getInstance
  - low `system/ioc/Injector.cfc:487` — declares getInstance
  - low `system/remote/ColdboxProxy.cfc:268` — declares getInstance
  - low `system/testing/BaseTestCase.cfc:854` — declares getInstance
  - low `test-harness/models/formBean.cfc:29` — declares getInstance
  - low `test-harness/models/formImplicitBean.cfc:21` — declares getInstance
- `getunits` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc:115` — declares getUnits; beside the calling file
  - low `system/async/time/Period.cfc:138` — declares getUnits; beside the calling file
- `of` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc:434` — declares of; beside the calling file
  - low `system/async/time/Period.cfc:341` — declares of; beside the calling file
- `populate` — 1 finding(s), 3 candidate(s):
  - low `system/FrameworkSupertype.cfc:119` — declares populate
  - low `system/core/delegates/Population.cfc:27` — declares populate
  - low `system/web/delegates/Population.cfc:35` — declares populate
- `run` — 1 finding(s), 77 candidate(s):
  - low `tests/perf-harness/PerformanceSuite.cfc:115` — declares run
  - low `tests/specs/EventHandlerTest.cfc:58` — declares run
  - low `tests/specs/FrameworkSuperTypeTest.cfc:5` — declares run
  - low `tests/specs/RestHandlerTest.cfc:15` — declares run
  - low `build/Build.cfc:131` — declares run
  - low `tests/specs/async/AsyncManagerSpec.cfc:10` — declares run
  - low `tests/specs/async/ExecutorServicesSpec.cfc:14` — declares run
  - low `tests/specs/cache/CacheBoxBuildTests.cfc:12` — declares run
  - … 69 more
- `setenabled` — 1 finding(s), 2 candidate(s):
  - low `system/cache/AbstractCacheBoxProvider.cfc:26` — declares setEnabled
  - low `system/web/routing/Router.cfc:67` — declares setEnabled
- `setlogbox` — 1 finding(s), 8 candidate(s):
  - low `system/logging/AbstractAppender.cfc:55` — declares setLogBox; beside the calling file
  - low `system/FrameworkSupertype.cfc:19` — declares setLogBox
  - low `system/cache/CacheFactory.cfc:41` — declares setLogbox
  - low `system/ioc/Builder.cfc:17` — declares setLogBox
  - low `system/ioc/Injector.cfc:80` — declares setLogBox
  - low `system/web/Controller.cfc:72` — declares setLogbox
  - low `system/ioc/dsl/LogBoxDSL.cfc:17` — declares setLogBox
  - low `system/web/routing/Router.cfc:36` — declares setLogBox
- `subtractfrom` — 1 finding(s), 2 candidate(s):
  - low `system/async/time/Duration.cfc:239` — declares subtractFrom; beside the calling file
  - low `system/async/time/Period.cfc:185` — declares subtractFrom; beside the calling file
- `tojson` — 1 finding(s), 2 candidate(s):
  - low `system/FrameworkSupertype.cfc:741` — declares toJson
  - low `system/core/util/Util.cfc:235` — declares toJson

</details>

## Object references — a component path that names no file

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 5 | `chained on 'command', which is not found (calling 'run')` | 80 | none |  | `tests/perf-harness/PerformanceSuite.cfc:266` chained on 'command', which is not found (calling 'run') |
| 5 | `component 'testbox.system.modules.mockdatacfc.models.MockData' does not exist (calling 'mock')` | 1 | none |  | `tests/suites/async/performance-parallel-tests.cfm:16` component 'testbox.system.modules.mockdatacfc.models.MockData' does not exist (calling 'mock') |
| 4 | `component 'docbox.DocBox' does not exist (calling 'generate')` | 0 | none |  | `apidocs/cachebox.cfm:16` component 'docbox.DocBox' does not exist (calling 'generate') |
| 2 | `chained on 'cache', which is not found (calling 'getOrDefault')` | 0 | none |  | `system/cache/providers/BoxLangProvider.cfc:270` chained on 'cache', which is not found (calling 'getOrDefault') |
| 2 | `chained on 'cache', which is not found (calling 'set')` | 13 | none |  | `system/cache/providers/BoxLangProvider.cfc:354` chained on 'cache', which is not found (calling 'set') |
| 2 | `chained on 'command', which is not found (calling 'flag')` | 0 | none |  | `tests/perf-harness/PerformanceSuite.cfc:281` chained on 'command', which is not found (calling 'flag') |
| 2 | `chained on 'command', which is not found (calling 'params')` | 1 | none |  | `tests/perf-harness/PerformanceSuite.cfc:281` chained on 'command', which is not found (calling 'params') |
| 1 | `chained on 'MCPServer', which is not found (calling 'registerTool')` | 0 | none |  | `test-harness/config/Router.cfc:11` chained on 'MCPServer', which is not found (calling 'registerTool') |
| 1 | `chained on 'cache', which is not found (calling 'clear')` | 18 | none |  | `system/cache/providers/BoxLangProvider.cfc:439` chained on 'cache', which is not found (calling 'clear') |
| 1 | `chained on 'cache', which is not found (calling 'clearAll')` | 16 | none |  | `system/cache/providers/BoxLangProvider.cfc:427` chained on 'cache', which is not found (calling 'clearAll') |
| 1 | `chained on 'cache', which is not found (calling 'clearQuiet')` | 7 | none |  | `system/cache/providers/BoxLangProvider.cfc:457` chained on 'cache', which is not found (calling 'clearQuiet') |
| 1 | `chained on 'cache', which is not found (calling 'clearStats')` | 0 | none |  | `system/cache/providers/BoxLangProvider.cfc:212` chained on 'cache', which is not found (calling 'clearStats') |
| 1 | `chained on 'cache', which is not found (calling 'get')` | 24 | none |  | `system/cache/providers/BoxLangProvider.cfc:270` chained on 'cache', which is not found (calling 'get') |
| 1 | `chained on 'cache', which is not found (calling 'getCachedObjectMetadata')` | 13 | none |  | `system/cache/providers/BoxLangProvider.cfc:261` chained on 'cache', which is not found (calling 'getCachedObjectMetadata') |
| 1 | `chained on 'cache', which is not found (calling 'getKeys')` | 13 | none |  | `system/cache/providers/BoxLangProvider.cfc:252` chained on 'cache', which is not found (calling 'getKeys') |
| 1 | `chained on 'cache', which is not found (calling 'getObjectStore')` | 7 | none |  | `system/cache/providers/BoxLangProvider.cfc:221` chained on 'cache', which is not found (calling 'getObjectStore') |
| 1 | `chained on 'cache', which is not found (calling 'getOrSet')` | 4 | none |  | `system/cache/providers/BoxLangProvider.cfc:327` chained on 'cache', which is not found (calling 'getOrSet') |
| 1 | `chained on 'cache', which is not found (calling 'getQuiet')` | 13 | none |  | `system/cache/providers/BoxLangProvider.cfc:279` chained on 'cache', which is not found (calling 'getQuiet') |
| 1 | `chained on 'cache', which is not found (calling 'getSize')` | 15 | none |  | `system/cache/providers/BoxLangProvider.cfc:408` chained on 'cache', which is not found (calling 'getSize') |
| 1 | `chained on 'cache', which is not found (calling 'getStoreMetadataReport')` | 7 | none |  | `system/cache/providers/BoxLangProvider.cfc:228` chained on 'cache', which is not found (calling 'getStoreMetadataReport') |
| 1 | `chained on 'cache', which is not found (calling 'lookup')` | 14 | none |  | `system/cache/providers/BoxLangProvider.cfc:297` chained on 'cache', which is not found (calling 'lookup') |
| 1 | `chained on 'cache', which is not found (calling 'lookupQuiet')` | 7 | none |  | `system/cache/providers/BoxLangProvider.cfc:306` chained on 'cache', which is not found (calling 'lookupQuiet') |
| 1 | `chained on 'cache', which is not found (calling 'reap')` | 13 | none |  | `system/cache/providers/BoxLangProvider.cfc:417` chained on 'cache', which is not found (calling 'reap') |
| 1 | `chained on 'filter', which is not found (calling 'keyArray')` | 0 | none |  | `system/ioc/Scopes.cfc:36` chained on 'filter', which is not found (calling 'keyArray') |
| 1 | `chained on 'getInstance', which is not found (calling 'addAsset')` | 2 | none |  | `system/modules/HTMLHelper/helpers/Mixins.cfm:9` chained on 'getInstance', which is not found (calling 'addAsset') |
| 1 | `chained on 'getModuleList', which is not found (calling 'findNoCase')` | 0 | none |  | `system/web/routing/Router.cfc:2102` chained on 'getModuleList', which is not found (calling 'findNoCase') |
| 1 | `chained on 'getModuleList', which is not found (calling 'keyArray')` | 0 | none |  | `system/web/routing/Router.cfc:2102` chained on 'getModuleList', which is not found (calling 'keyArray') |
| 1 | `chained on 'setLogBox', which is not found (calling 'onRegistration')` | 6 | none |  | `system/logging/LogBox.cfc:354` chained on 'setLogBox', which is not found (calling 'onRegistration') |
| 1 | `chained on 'setLogBox', which is not found (calling 'setColdBox')` | 14 | none |  | `system/logging/LogBox.cfc:354` chained on 'setLogBox', which is not found (calling 'setColdBox') |
| 1 | `chained on 'setLogBox', which is not found (calling 'setInitialized')` | 1 | none |  | `system/logging/LogBox.cfc:354` chained on 'setLogBox', which is not found (calling 'setInitialized') |
| 1 | `chained on 'setLogBox', which is not found (calling 'setWireBox')` | 14 | none |  | `system/logging/LogBox.cfc:354` chained on 'setLogBox', which is not found (calling 'setWireBox') |
| 1 | `component 'bxModules.bxai.models.mcp.MCPRequestProcessor' does not exist (calling 'processHttp')` | 0 | none |  | `system/web/routing/Router.cfc:2406` component 'bxModules.bxai.models.mcp.MCPRequestProcessor' does not exist (calling 'processHttp') |
| 1 | `component 'coldbox.system.Coldbox' does not exist (calling 'isfwReinit')` | 2 | low | file named Coldbox.cfc (4 candidates) | `tests/suites/eventCachingCollisions/Application.cfc:52` component 'coldbox.system.Coldbox' does not exist (calling 'isfwReinit') |
| 1 | `component 'coldbox.system.Coldbox' does not exist (calling 'loadColdbox')` | 3 | low | file named Coldbox.cfc (4 candidates) | `tests/suites/eventCachingCollisions/Application.cfc:39` component 'coldbox.system.Coldbox' does not exist (calling 'loadColdbox') |
| 1 | `component 'coldbox.system.Coldbox' does not exist (calling 'onSessionStart')` | 10 | low | file named Coldbox.cfc (4 candidates) | `tests/suites/eventCachingCollisions/Application.cfc:81` component 'coldbox.system.Coldbox' does not exist (calling 'onSessionStart') |
| 1 | `component 'coldbox.system.Coldbox' does not exist (calling 'processColdBoxRequest')` | 2 | low | file named Coldbox.cfc (4 candidates) | `tests/suites/eventCachingCollisions/Application.cfc:63` component 'coldbox.system.Coldbox' does not exist (calling 'processColdBoxRequest') |
| 1 | `component 'coldbox.system.Coldbox' does not exist (calling 'reloadChecks')` | 2 | low | file named Coldbox.cfc (4 candidates) | `tests/suites/eventCachingCollisions/Application.cfc:59` component 'coldbox.system.Coldbox' does not exist (calling 'reloadChecks') |

<details><summary>Groups with several candidates</summary>

- `component 'coldbox.system.Coldbox' does not exist (calling 'isfwReinit')` — 1 finding(s), 4 candidate(s):
  - low `tests/suites/eventCachingCollisions/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/suites/loadtests/beload/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `test-harness/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/perf-harness/app/config/ColdBox.cfc` — file named Coldbox.cfc
- `component 'coldbox.system.Coldbox' does not exist (calling 'loadColdbox')` — 1 finding(s), 4 candidate(s):
  - low `tests/suites/eventCachingCollisions/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/suites/loadtests/beload/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `test-harness/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/perf-harness/app/config/ColdBox.cfc` — file named Coldbox.cfc
- `component 'coldbox.system.Coldbox' does not exist (calling 'onSessionStart')` — 1 finding(s), 4 candidate(s):
  - low `tests/suites/eventCachingCollisions/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/suites/loadtests/beload/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `test-harness/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/perf-harness/app/config/ColdBox.cfc` — file named Coldbox.cfc
- `component 'coldbox.system.Coldbox' does not exist (calling 'processColdBoxRequest')` — 1 finding(s), 4 candidate(s):
  - low `tests/suites/eventCachingCollisions/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/suites/loadtests/beload/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `test-harness/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/perf-harness/app/config/ColdBox.cfc` — file named Coldbox.cfc
- `component 'coldbox.system.Coldbox' does not exist (calling 'reloadChecks')` — 1 finding(s), 4 candidate(s):
  - low `tests/suites/eventCachingCollisions/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/suites/loadtests/beload/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `test-harness/config/Coldbox.cfc` — file named Coldbox.cfc
  - low `tests/perf-harness/app/config/ColdBox.cfc` — file named Coldbox.cfc

</details>
