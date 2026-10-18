# lucee — unresolved findings and candidates

Every finding below is still reported. A candidate is a liberal match offered for a person to weigh, never an answer the resolver took: **high** is the only component declaring every method the function calls on the receiver (or the only one, named like it); **medium** is a sole match on weaker evidence, or the one named like the receiver among several; **low** is one of several. *Defined* is how many indexed files declare a method of that name: 0 means it is missing from the workspace, more means the resolver could not connect the call to it.

| Category | Findings | high | medium | low | none | method defined nowhere |
|---|---:|---:|---:|---:|---:|---:|
| variable | 1571 | 53 | 62 | 286 | 1170 | 853 |
| return-type | 31 | 0 | 3 | 9 | 19 | 18 |
| method | 310 | 0 | 31 | 24 | 255 | 239 |
| object | 73 | 0 | 3 | 12 | 58 | 37 |

## Variable definitions — a receiver whose component is unknown

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 180 | `field` | 67 | none |  | `core/src/main/cfml/context/admin/debugging.templates.create.cfm:145` variable 'field' has no component ref |
| 96 | `coll` | 0 | none |  | `test/a_debug_build/_MongoDB.cfc:54` variable 'coll' has no component ref |
| 44 | `driver` | 15 | none |  | `core/src/main/cfml/context/admin/debugging.templates.create.cfm:47` variable 'driver' has no component ref |
| 30 | `msg` | 1 | none |  | `test/tags/Mail2.cfc:52` variable 'msg' has no component ref |
| 29 | `result` | 0 | none |  | `core/src/main/cfml/context/admin/ext.functions.cfm:11` variable 'result' has no component ref |
| 24 | `res` | 0 | none |  | `test/functions/StructToSorted.cfc:138` variable 'res' has no component ref |
| 20 | `animals` | 0 | none |  | `test/functions/StructKeyTranslate.cfc:18` variable 'animals' has no component ref |
| 20 | `merger → test/tickets/LDEV5758.cfc` | 1 | medium | declares merge() | `test/tickets/LDEV5758.cfc:15` variable 'merger' has no component ref |
| 20 | `qry` | 0 | none |  | `test/functions/QueryColumnData.cfc:27` variable 'qry' has no component ref |
| 19 | `field → core/src/main/java/resource/context/admin/dbdriver/types/Field.cfc` | 11 | high | declares getType(), getName(), getDefaultValue(), getDisplayName(), getRequired(), getDefaultValueIndex(), getDescription(); named like the receiver 'field' | `core/src/main/cfml/context/admin/services.datasource.create.cfm:532` variable 'field' has no component ref |
| 17 | `mysql` | 0 | none |  | `test/jira/Jira3043.cfc:74` variable 'mySQL' has no component ref |
| 16 | `result.filecontent` | 0 | none |  | `test/tags/ProcessingDirective.cfc:9` variable 'result.fileContent' has no component ref |
| 14 | `docsfound` | 0 | none |  | `test/a_debug_build/_MongoDB.cfc:166` variable 'docsFound' has no component ref |
| 14 | `driver → core/src/main/java/resource/context/admin/mailservers/GMX.cfc` | 8 | low | declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS() (8 candidates) | `core/src/main/cfml/context/admin/services.mail.form.cfm:67` variable 'driver' has no component ref |
| 14 | `props` | 0 | none |  | `test/tickets/LDEV0719.cfc:24` variable 'props' has no component ref |
| 14 | `results` | 0 | none |  | `core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestSuiteRunner.cfc:43` variable 'results' has no component ref |
| 13 | `tz` | 0 | none |  | `test/datasource/Oracle.cfc:44` variable 'tz' has no component ref |
| 12 | `c` | 0 | none |  | `test/general/CodeIsland.cfc:47` variable 'c' has no component ref |
| 12 | `db` | 0 | none |  | `test/a_debug_build/_MongoDB.cfc:45` variable 'db' has no component ref |
| 12 | `sss` | 0 | none |  | `test/general/Resources.cfc:258` variable 'sss' has no component ref |
| 11 | `_cookies` | 0 | none |  | `test/tickets/LDEV3478.cfc:26` variable '_cookies' has no component ref |
| 11 | `engine` | 0 | none |  | `test/general/JSR223.cfc:60` variable 'engine' has no component ref |
| 11 | `s` | 1 | none |  | `test/general/Resources.cfc:274` variable 's' has no component ref |
| 9 | `application.objects.utils → core/src/main/cfml/context/doc/DocUtils.cfc` | 3 | low | declares getAllFunctions() (3 candidates) | `core/src/main/cfml/context/doc/categories.cfm:4` variable 'Application.objects.utils' has no component ref |
| 9 | `b` | 0 | none |  | `test/functions/BooleanFormat.cfc:40` variable 'b' has no component ref |
| 9 | `db[]` | 0 | none |  | `test/a_debug_build/_MongoDB.cfc:115` variable 'db[]' has no component ref |
| 9 | `mail` | 0 | none |  | `test/tags/Mail2.cfc:47` variable 'mail' has no component ref |
| 9 | `parts` | 2 | none |  | `test/tags/Mail2.cfc:206` variable 'parts' has no component ref |
| 8 | `admin` | 0 | none |  | `core/src/main/cfml/context/admin/messaging.cfm:18` variable 'admin' has no component ref |
| 8 | `arrallitems` | 0 | none |  | `core/src/main/cfml/context/doc/components.cfm:10` variable 'arrAllItems' has no component ref |
| 8 | `data.attributes` | 0 | none |  | `core/src/main/cfml/context/debug/modern/reference.cfm:949` variable 'data.attributes' has no component ref |
| 8 | `future` | 0 | none |  | `test/tickets/LDEV1841.cfc:44` variable 'future' has no component ref |
| 8 | `out` | 0 | none |  | `test/functions/configTranslate.cfc:19` variable 'out' has no component ref |
| 8 | `row` | 0 | none |  | `test/tickets/Issue0103.cfc:29` variable 'row' has no component ref |
| 7 | `ext` | 0 | none |  | `test/_setupTestServices.cfc:867` variable 'ext' has no component ref |
| 7 | `m` | 0 | none |  | `core/src/main/cfml/context/doc/Application.cfc:107` variable 'm' has no component ref |
| 7 | `o` | 1 | none |  | `test/general/Resources.cfc:379` variable 'o' has no component ref |
| 7 | `testcase` | 1 | none |  | `core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestSuiteRunner.cfc:21` variable 'testCase' has no component ref |
| 6 | `attr` | 0 | none |  | `core/src/main/cfml/context/debug/modern/reference.cfm:971` variable 'attr' has no component ref |
| 6 | `attr[]` | 0 | none |  | `core/src/main/cfml/context/debug/modern/reference.cfm:971` variable 'attr[]' has no component ref |
| 6 | `cd` | 3 | none |  | `test/_setupTestServices.cfc:847` variable 'cd' has no component ref |
| 6 | `class2` | 67 | none |  | `test/tickets/LDEV6226.cfc:78` variable 'class2' has no component ref |
| 6 | `data` | 0 | none |  | `core/src/main/cfml/context/doc/DocUtils.cfc:77` variable 'data' has no component ref |
| 6 | `data.extensions` | 0 | none |  | `core/src/main/cfml/context/admin/ext.functions.cfm:373` variable 'data.extensions' has no component ref |
| 6 | `field → core/src/main/java/resource/context/admin/logging/appender/Field.cfc` | 67 | high | declares getName(), getDefaultValue(), setDefaultValue(); named like the receiver 'field' | `core/src/main/java/resource/context/admin/logging/appender/DatasourceAppender.cfc:24` variable 'field' has no component ref |
| 6 | `froms[]` | 0 | none |  | `test/tags/Mail2.cfc:54` variable 'froms[]' has no component ref |
| 6 | `msg → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tags/Mail2.cfc:62` variable 'msg' has no component ref |
| 6 | `q` | 0 | none |  | `test/jira/Jira2360.cfc:38` variable 'q' has no component ref |
| 6 | `q.columnlist` | 0 | none |  | `test/tickets/LDEV2924.cfc:16` variable 'q.columnList' has no component ref |
| 6 | `result.num → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/tags/InvokeArgument.cfc:14` variable 'result.num' has no component ref |
| 6 | `result.num → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tags/InvokeArgument.cfc:14` variable 'result.num' has no component ref |
| 6 | `result.str → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/tags/InvokeArgument.cfc:13` variable 'result.str' has no component ref |
| 6 | `result.str → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tags/InvokeArgument.cfc:13` variable 'result.str' has no component ref |
| 6 | `rtn → test/jira/Jira1460.cfc` | 5 | low | declares a(), b() (5 candidates) | `test/jira/Jira2633/Test.cfc:40` variable 'rtn' has no component ref |
| 6 | `s → test/jira/Jira0713/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/jira/Jira2384.cfc:45` variable 's' has no component ref |
| 6 | `s → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/jira/Jira2384.cfc:45` variable 's' has no component ref |
| 6 | `tos[]` | 0 | none |  | `test/tags/Mail2.cfc:59` variable 'tos[]' has no component ref |
| 5 | `application.adminfunctions → core/src/main/cfml/context/admin/adminfunctions.cfc` | 1 | high | declares canAccessContext(), isFavorite(); named like the receiver 'adminfunctions' | `core/src/main/cfml/context/admin/admin_layout.cfm:71` variable 'application.adminfunctions' has no component ref |
| 5 | `driver → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 40 | low | declares getLabel(), getClass() (21 candidates) | `core/src/main/cfml/context/admin/services.cache.list.cfm:135` variable 'driver' has no component ref |
| 5 | `driver → core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` | 40 | high | declares getLabel(), getClass(), getLabelLong(), getDescription() | `core/src/main/cfml/context/admin/services.ai.list.cfm:94` variable 'driver' has no component ref |
| 5 | `drivers[] → core/src/main/java/resource/context/admin/mailservers/GMX.cfc` | 8 | low | declares getHost(), getPort(), useSSL(), useTLS(), getLabel() (8 candidates) | `core/src/main/cfml/context/admin/services.mail.serverlist.cfm:12` variable 'drivers[]' has no component ref |
| 5 | `extmeta` | 67 | none |  | `test/tickets/LDEV6088.cfc:18` variable 'extMeta' has no component ref |
| 5 | `file` | 0 | none |  | `core/src/main/cfml/context/gateway/WatchService.cfc:29` variable 'file' has no component ref |
| 5 | `pgsql` | 0 | none |  | `test/datasource/PostgreSQL.cfc:101` variable 'pgsql' has no component ref |
| 5 | `sct` | 0 | none |  | `core/src/main/cfml/context/admin/ext.functions.cfm:715` variable 'sct' has no component ref |
| 5 | `src.debugtemplates` | 0 | none |  | `test/tickets/LDEV5758.cfc:579` variable 'src.debugTemplates' has no component ref |
| 5 | `sum` | 0 | none |  | `test/tickets/LDEV1296.cfc:11` variable 'sum' has no component ref |
| 4 | `_driver → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` | 26 | low | declares getClass(); named like the receiver 'driver' (26 candidates) | `core/src/main/cfml/context/admin/server.logging.create.cfm:235` variable '_driver' has no component ref |
| 4 | `afterstored` | 0 | none |  | `test/tickets/LDEV5941/testCommitNoMetadataBump.cfm:19` variable 'afterStored' has no component ref |
| 4 | `appender → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 40 | low | declares getLabel(), getClass() (21 candidates) | `core/src/main/cfml/context/admin/server.logging.list.cfm:97` variable 'appender' has no component ref |
| 4 | `arguments.spec` | 0 | none |  | `test/tickets/LDEV0359.cfc:37` variable 'arguments.spec' has no component ref |
| 4 | `before` | 0 | none |  | `test/tickets/LDEV5941/testCommitNoMetadataBump.cfm:12` variable 'before' has no component ref |
| 4 | `body` | 0 | none |  | `ant/upload_to_s3.cfm:170` variable 'body' has no component ref |
| 4 | `callercache` | 0 | none |  | `test/tickets/LDEV6240.cfc:170` variable 'callerCache' has no component ref |
| 4 | `class2 → test/general/static/StaticCreateInstance.cfc` | 1 | medium | declares newInstance() | `test/tickets/LDEV6226.cfc:94` variable 'class2' has no component ref |
| 4 | `currfunc.parameters` | 1 | none |  | `core/src/main/cfml/context/doc/components.cfm:245` variable 'currFunc.parameters' has no component ref |
| 4 | `drivers[] → core/src/main/java/resource/context/admin/debug/Classic.cfc` | 48 | low | declares getDescription(), getId(), getLabel() (4 candidates) | `core/src/main/cfml/context/admin/debugging.logs.detail.cfm:72` variable 'drivers[]' has no component ref |
| 4 | `e` | 0 | none |  | `test/tickets/LDEV0321.cfc:36` variable 'e' has no component ref |
| 4 | `form` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.cfm:34` variable 'form' has no component ref |
| 4 | `id` | 0 | none |  | `test/a_debug_build/_MongoDB.cfc:132` variable 'id' has no component ref |
| 4 | `javaproxies` | 0 | none |  | `test/tickets/LDEV5459.cfc:72` variable 'javaProxies' has no component ref |
| 4 | `jdbc → test/functions/serializeJSON/_Json.cfc` | 5 | low | declares get() (4 candidates) | `test/_setupTestServices.cfc:874` variable 'jdbc' has no component ref |
| 4 | `layout → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 40 | low | declares getLabel(), getClass() (21 candidates) | `core/src/main/cfml/context/admin/server.logging.list.cfm:98` variable 'layout' has no component ref |
| 4 | `locals` | 0 | none |  | `core/src/main/cfml/context/admin/ext.functions.cfm:311` variable 'locals' has no component ref |
| 4 | `mongo → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/a_debug_build/_MongoDB.cfc:89` variable 'mongo' has no component ref |
| 4 | `mystruct` | 0 | none |  | `test/functions/structEvery.cfc:9` variable 'myStruct' has no component ref |
| 4 | `notexa.b.c → test/general/InlineComponent.cfc` | 2 | low | declares d() (2 candidates) | `test/general/Elvis.cfc:65` variable 'notexa.b.c' has no component ref |
| 4 | `orcl` | 0 | none |  | `test/datasource/Oracle.cfc:39` variable 'orcl' has no component ref |
| 4 | `qry.id[]` | 0 | none |  | `test/tickets/LDEV4665.cfc:21` variable 'qry.id[]' has no component ref |
| 4 | `res.filecontent` | 0 | none |  | `test/general/ApplicationListener.cfc:48` variable 'res.filecontent' has no component ref |
| 4 | `result.tnum → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV5629.cfc:14` variable 'result.tNum' has no component ref |
| 4 | `result.tnum → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV5629.cfc:14` variable 'result.tNum' has no component ref |
| 4 | `result.tstr → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV5629.cfc:13` variable 'result.tStr' has no component ref |
| 4 | `result.tstr → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV5629.cfc:13` variable 'result.tStr' has no component ref |
| 4 | `st` | 0 | none |  | `test/functions/StructToSorted.cfc:137` variable 'st' has no component ref |
| 4 | `test[]` | 0 | none |  | `test/tickets/LDEV4665.cfc:21` variable 'test[]' has no component ref |
| 4 | `tz → test/jira/Jira2275/orm/FieldLink.cfc` | 14 | low | declares getDisplayName() (14 candidates) | `test/tickets/LDEV1761.cfc:38` variable 'tz' has no component ref |
| 4 | `world1 → test/tickets/LDEV0270.cfc` | 2 | low | declares delete() (2 candidates) | `test/functions/structdelete.cfc:17` variable 'world1' has no component ref |
| 3 | `a → test/general/InlineComponent.cfc` | 6 | low | declares b() (5 candidates) | `test/general/SafeNavigator.cfc:7` variable 'a' has no component ref |
| 3 | `a.b.c → test/general/InlineComponent.cfc` | 2 | low | declares d() (2 candidates) | `test/general/Elvis.cfc:80` variable 'a.b.c' has no component ref |
| 3 | `arguments.path` | 1 | none |  | `core/src/main/cfml/context/gateway/WatchService.cfc:20` variable 'arguments.path' has no component ref |
| 3 | `attachment` | 0 | none |  | `test/tags/Mail2.cfc:264` variable 'attachment' has no component ref |
| 3 | `bundle` | 0 | none |  | `test/tickets/LDEV6088.cfc:26` variable 'bundle' has no component ref |
| 3 | `cfc → core/src/main/cfml/context/admin/ExtensionProviderProxy.cfc` | 1 | high | declares listApplications(), getInfo() | `core/src/main/cfml/context/admin/ExtensionProviderProxy.cfc:60` variable 'cfc' has no component ref |
| 3 | `cfcc → test/jira/Jira0713/Test.cfc` | 67 | low | declares getName(), setName() (56 candidates) | `test/jira/Jira0713.cfc:29` variable 'cfcc' has no component ref |
| 3 | `class1` | 67 | none |  | `test/tickets/LDEV6226.cfc:78` variable 'class1' has no component ref |
| 3 | `content` | 2 | none |  | `test/tags/Mail2.cfc:255` variable 'content' has no component ref |
| 3 | `dr → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 26 | low | declares getClass() (26 candidates) | `core/src/main/cfml/context/admin/services.datasource.cfm:26` variable 'dr' has no component ref |
| 3 | `driver → core/src/main/java/resource/context/admin/debug/Classic.cfc` | 5 | low | declares output() (5 candidates) | `core/src/main/cfml/context/admin/debugging.logs.detail.cfm:104` variable 'driver' has no component ref |
| 3 | `email` | 0 | none |  | `test/tags/_Mail.cfc:59` variable 'email' has no component ref |
| 3 | `findanimals` | 0 | none |  | `test/functions/structFind.cfc:11` variable 'findAnimals' has no component ref |
| 3 | `fruit → test/tickets/LDEV0423/Fruit.cfc` | 67 | medium | declares getName(); named like the receiver 'fruit' (67 candidates) | `test/tickets/LDEV0423/Basket.cfc:19` variable 'fruit' has no component ref |
| 3 | `item → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV5578/LDEV5578.cfc:56` variable 'item' has no component ref |
| 3 | `itemlist[]` | 0 | none |  | `core/src/main/cfml/context/doc/DocUtils.cfc:37` variable 'itemList[]' has no component ref |
| 3 | `jsonobj` | 0 | none |  | `test/tickets/LDEV5530.cfc:35` variable 'jsonObj' has no component ref |
| 3 | `list` | 0 | none |  | `test/general/StaticMembersInvoke.cfc:40` variable 'list' has no component ref |
| 3 | `local.res.errors` | 0 | none |  | `test/tickets/LDEV2113.cfc:68` variable 'local.res.errors' has no component ref |
| 3 | `local.res.fatalerrors` | 0 | none |  | `test/tickets/LDEV2113.cfc:69` variable 'local.res.fatalerrors' has no component ref |
| 3 | `local.res.warnings` | 0 | none |  | `test/tickets/LDEV2113.cfc:70` variable 'local.res.warnings' has no component ref |
| 3 | `os` | 0 | none |  | `test/tickets/LDEV5510.cfc:27` variable 'os' has no component ref |
| 3 | `plugin.component → core/src/main/cfml/context/admin/plugin/Plugin.cfc` | 1 | high | declares _action(), action(), _display() | `core/src/main/cfml/context/admin/plugin.cfm:65` variable 'plugin.component' has no component ref |
| 3 | `proxy → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV5074.cfc:10` variable 'proxy' has no component ref |
| 3 | `proxy → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV5074.cfc:10` variable 'proxy' has no component ref |
| 3 | `res.object → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV5459.cfc:24` variable 'res.object' has no component ref |
| 3 | `res.object → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV5459.cfc:24` variable 'res.object' has no component ref |
| 3 | `sct → test/tickets/LDEV3963/test3963quoted.cfc` | 5 | low | declares get() (4 candidates) | `test/tickets/LDEV4826.cfc:10` variable 'sct' has no component ref |
| 3 | `src.rest` | 0 | none |  | `test/tickets/LDEV5758.cfc:353` variable 'src.rest' has no component ref |
| 3 | `str` | 0 | none |  | `test/jira/Jira2397.cfc:29` variable 'str' has no component ref |
| 3 | `url` | 0 | none |  | `test/orm/events/Application.cfc:24` variable 'url' has no component ref |
| 2 | `_defaultdriver → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 40 | low | declares getLabel() (40 candidates) | `core/src/main/cfml/context/admin/services.mail.form.cfm:76` variable '_DefaultDriver' has no component ref |
| 2 | `antisamyresult` | 0 | none |  | `test/tickets/LDEV5085/AntiSamyService.cfc:20` variable 'antiSamyResult' has no component ref |
| 2 | `args.num → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/tags/InvokeArgument.cfc:38` variable 'args.num' has no component ref |
| 2 | `args.num → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tags/InvokeArgument.cfc:38` variable 'args.num' has no component ref |
| 2 | `args.str → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/tags/InvokeArgument.cfc:37` variable 'args.str' has no component ref |
| 2 | `args.str → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tags/InvokeArgument.cfc:37` variable 'args.str' has no component ref |
| 2 | `arguments.arr` | 0 | none |  | `test/tickets/LDEV3668.cfc:36` variable 'arguments.arr' has no component ref |
| 2 | `arguments.debugtemplate → core/src/main/java/resource/context/admin/debug/Classic.cfc` | 5 | low | declares output() (5 candidates) | `core/src/main/java/resource/context/admin/info/Info.cfc:31` variable 'arguments.debugTemplate' has no component ref |
| 2 | `arguments.results` | 0 | none |  | `test/tickets/LDEV4892.cfc:34` variable 'arguments.results' has no component ref |
| 2 | `arr` | 0 | none |  | `test/tickets/LDEV2005.cfc:14` variable 'arr' has no component ref |
| 2 | `arr[]` | 0 | none |  | `test/tickets/LDEV1680.cfc:17` variable 'arr[]' has no component ref |
| 2 | `ast.body` | 0 | none |  | `test/tickets/LDEV5839.cfc:47` variable 'ast.body' has no component ref |
| 2 | `c → test/functions/serializeJSON/_Json.cfc` | 5 | low | declares get() (4 candidates) | `test/functions/Duplicate.cfc:44` variable 'c' has no component ref |
| 2 | `cgi` | 0 | none |  | `test/tickets/LDEV3841/cfc/index.cfm:3` variable 'cgi' has no component ref |
| 2 | `code → test/orm/clearSession/Code.cfc` | 96 | low | declares getId(), getCode(); named like the receiver 'code' (7 candidates) | `test/orm/Hibernate.cfc:52` variable 'code' has no component ref |
| 2 | `code → test/tickets/LDEV0881/Code.cfc` | 96 | low | declares getId(), getCode(); named like the receiver 'code' (7 candidates) | `test/tickets/LDEV0881.cfc:38` variable 'code' has no component ref |
| 2 | `coll → core/src/main/java/resource/component/org/lucee/cfml/Ftp.cfc` | 1 | medium | declares rename() | `test/a_debug_build/_MongoDB.cfc:330` variable 'coll' has no component ref |
| 2 | `componentmappings[]` | 0 | none |  | `test/general/Mappings.cfc:70` variable 'componentMappings[]' has no component ref |
| 2 | `config` | 0 | none |  | `core/src/main/cfml/context/admin/messaging.cfm:105` variable 'config' has no component ref |
| 2 | `customtagmappings[]` | 0 | none |  | `test/general/Mappings.cfc:81` variable 'customTagMappings[]' has no component ref |
| 2 | `d → core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` | 6 | low | declares getCFCPath(), getClass() (6 candidates) | `core/src/main/cfml/context/admin/services.gateway.cfm:45` variable 'd' has no component ref |
| 2 | `data.content.cookie` | 0 | none |  | `test/tickets/LDEV0581.cfc:32` variable 'data.content.cookie' has no component ref |
| 2 | `data.content.form` | 0 | none |  | `test/tickets/LDEV0581.cfc:36` variable 'data.content.form' has no component ref |
| 2 | `data.content.httprequestdata.headers` | 0 | none |  | `test/tickets/LDEV0581.cfc:44` variable 'data.content.HTTPRequestData.headers' has no component ref |
| 2 | `data.content.url` | 0 | none |  | `test/tickets/LDEV0581.cfc:40` variable 'data.content.url' has no component ref |
| 2 | `data.ps → core/src/main/cfml/context/gateway/WatchService.cfc` | 2 | low | declares close() (2 candidates) | `test/jira/_Jira2672.cfc:67` variable 'data.ps' has no component ref |
| 2 | `dd` | 0 | none |  | `test/general/Resources.cfc:417` variable 'dd' has no component ref |
| 2 | `dnsrecordservice → core/src/main/java/resource/component/org/lucee/cfml/Query.cfc` | 1 | medium | declares new() | `test/tickets/LDEV3060/invalidcomponent.cfc:3` variable 'DnsRecordservice' has no component ref |
| 2 | `driver → core/src/main/java/resource/context/admin/aidriver/Field.cfc` | 67 | low | declares getName() (67 candidates) | `core/src/main/cfml/context/admin/services.datasource.list.cfm:371` variable 'driver' has no component ref |
| 2 | `dupmap → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV3805.cfc:7` variable 'dupMap' has no component ref |
| 2 | `dupmap → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV3805.cfc:7` variable 'dupMap' has no component ref |
| 2 | `entries` | 0 | none |  | `test/tickets/LDEV1109.cfc:44` variable 'entries' has no component ref |
| 2 | `entriesfield` | 0 | none |  | `test/tickets/LDEV1109.cfc:37` variable 'entriesField' has no component ref |
| 2 | `eval` | 26 | none |  | `test/functions/Serialize.cfc:43` variable 'eval' has no component ref |
| 2 | `event` | 0 | none |  | `core/src/main/cfml/context/gateway/WatchService.cfc:62` variable 'event' has no component ref |
| 2 | `externals` | 0 | none |  | `core/src/main/cfml/context/admin/ext.functions.cfm:425` variable 'externals' has no component ref |
| 2 | `failedtestcase.cfmlstacktrace` | 0 | none |  | `test/_testRunner.cfc:196` variable 'failedTestCase.cfmlStackTrace' has no component ref |
| 2 | `fruits` | 0 | none |  | `test/tickets/LDEV0423/index.cfm:11` variable 'fruits' has no component ref |
| 2 | `header` | 67 | none |  | `test/tags/Mail.cfc:142` variable 'header' has no component ref |
| 2 | `headers` | 0 | none |  | `test/tags/Mail.cfc:140` variable 'headers' has no component ref |
| 2 | `installed` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.detail.cfm:25` variable 'installed' has no component ref |
| 2 | `it` | 0 | none |  | `test/tags/_Mail.cfc:57` variable 'it' has no component ref |
| 2 | `itemlist` | 0 | none |  | `core/src/main/cfml/context/doc/objects.cfm:6` variable 'itemList' has no component ref |
| 2 | `iter` | 0 | none |  | `test/tickets/LDEV1109.cfc:45` variable 'iter' has no component ref |
| 2 | `loadobj → test/tickets/LDEV0506.cfc` | 2 | high | declares cachedFunction(), nonCachedFunction() | `test/tickets/LDEV0506.cfc:17` variable 'loadObj' has no component ref |
| 2 | `local.result` | 0 | none |  | `test/tickets/LDEV1337.cfc:6` variable 'local.result' has no component ref |
| 2 | `mails[]` | 0 | none |  | `test/tags/Mail.cfc:190` variable 'mails[]' has no component ref |
| 2 | `mongo` | 0 | none |  | `test/a_debug_build/_MongoDB.cfc:106` variable 'mongo' has no component ref |
| 2 | `myvar` | 0 | none |  | `test/general/SafeNavOp.cfc:20` variable 'myvar' has no component ref |
| 2 | `myvar.firstlevel` | 0 | none |  | `test/general/SafeNavOp.cfc:21` variable 'myvar.firstlevel' has no component ref |
| 2 | `myvar.firstlevel.nextlevel` | 0 | none |  | `test/general/SafeNavOp.cfc:22` variable 'myvar.firstlevel.nextlevel' has no component ref |
| 2 | `names` | 0 | none |  | `test/tags/_Mail.cfc:61` variable 'names' has no component ref |
| 2 | `newclass → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` | 124 | low | declares init(), getValue() (2 candidates) | `test/tickets/LDEV4001/LDEV4001.cfm:17` variable 'newClass' has no component ref |
| 2 | `newperson.testclosurethis.mappers` | 0 | none |  | `test/tickets/LDEV4067/LDEV4067.cfm:6` variable 'newPerson.testClosureThis.MAPPERS' has no component ref |
| 2 | `newperson.testclosurevar.mappers` | 0 | none |  | `test/tickets/LDEV4067/LDEV4067.cfm:11` variable 'newPerson.testClosureVar.MAPPERS' has no component ref |
| 2 | `newperson.testlambdathis.mappers` | 0 | none |  | `test/tickets/LDEV4067/LDEV4067.cfm:16` variable 'newPerson.testLambdaThis.MAPPERS' has no component ref |
| 2 | `newperson.testlambdavar.mappers` | 0 | none |  | `test/tickets/LDEV4067/LDEV4067.cfm:21` variable 'newPerson.testLambdaVar.MAPPERS' has no component ref |
| 2 | `numbers` | 0 | none |  | `test/tickets/LDEV4725.cfc:8` variable 'numbers' has no component ref |
| 2 | `obj` | 7 | none |  | `test/tickets/LDEV4972.cfc:14` variable 'obj' has no component ref |
| 2 | `odnsdomain → test/tickets/LDEV0078/Test.cfc` | 96 | low | declares getId() (96 candidates) | `test/tickets/LDEV3060/invalidcomponent.cfc:4` variable 'oDnsDomain' has no component ref |
| 2 | `p` | 0 | none |  | `test/tickets/LDEV1109.cfc:77` variable 'p' has no component ref |
| 2 | `points` | 0 | none |  | `test/tickets/LDEV4491.cfc:119` variable 'points' has no component ref |
| 2 | `presideobjectservice` | 1 | none |  | `test/tickets/LDEV4901.cfc:4942` variable 'presideObjectService' has no component ref |
| 2 | `projects` | 0 | none |  | `test/tickets/LDEV2793.cfc:39` variable 'projects' has no component ref |
| 2 | `provider` | 0 | none |  | `test/general/Resources.cfc:696` variable 'provider' has no component ref |
| 2 | `qry.driver → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 40 | low | declares getLabel() (40 candidates) | `core/src/main/cfml/context/admin/debugging.templates.list.cfm:137` variable 'qry.driver' has no component ref |
| 2 | `r → test/tickets/LDEV6138/TestEntity.cfc` | 67 | low | declares getName(), setName() (56 candidates) | `test/tickets/LDEV6138/LDEV6138.cfm:34` variable 'r' has no component ref |
| 2 | `ramcache` | 1 | none |  | `test/tickets/LDEV1109.cfc:18` variable 'ramCache' has no component ref |
| 2 | `rawcause` | 26 | none |  | `test/tickets/LDEV6347.cfc:54` variable 'rawCause' has no component ref |
| 2 | `ref → test/orm/many2one/Ref.cfc` | 96 | low | declares getId(), getCode(); named like the receiver 'ref' (7 candidates) | `test/orm/Hibernate.cfc:48` variable 'ref' has no component ref |
| 2 | `ref → test/tickets/LDEV0881/Ref.cfc` | 96 | low | declares getId(), getCode(); named like the receiver 'ref' (7 candidates) | `test/tickets/LDEV0881.cfc:34` variable 'ref' has no component ref |
| 2 | `res → test/jira/Jira2644/State.cfc` | 1 | high | declares getStateCode(), getCountryCode() | `test/jira/Jira2644.cfc:34` variable 'res' has no component ref |
| 2 | `res.client[]` | 0 | none |  | `test/tickets/LDEV0215_mysql.cfc:24` variable 'res.client[]' has no component ref |
| 2 | `res.session[]` | 0 | none |  | `test/tickets/LDEV0215_mysql.cfc:20` variable 'res.session[]' has no component ref |
| 2 | `result → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/LDEV4531/LDEV4531.cfm:7` variable 'result' has no component ref |
| 2 | `s → core/src/main/java/resource/component/org/lucee/cfml/Ftp.cfc` | 1 | medium | declares remove() | `test/general/Resources.cfc:324` variable 's' has no component ref |
| 2 | `session.ldev3125` | 0 | none |  | `test/tickets/LDEV2135/cfml-session/secondRequest.cfm:7` variable 'session.ldev3125' has no component ref |
| 2 | `storedvalue` | 0 | none |  | `test/tickets/LDEV5941/testCommitOnly.cfm:16` variable 'storedValue' has no component ref |
| 2 | `struct` | 0 | none |  | `test/tickets/LDEV1295.cfc:21` variable 'struct' has no component ref |
| 2 | `struct1 → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` | 4 | low | declares equals() (4 candidates) | `test/tickets/LDEV2608.cfc:10` variable 'struct1' has no component ref |
| 2 | `student` | 0 | none |  | `test/functions/structfindkey.cfc:48` variable 'student' has no component ref |
| 2 | `studentnew` | 0 | none |  | `test/functions/structkeylist.cfc:41` variable 'studentnew' has no component ref |
| 2 | `t3[] → test/tickets/LDEV0374/users.cfc` | 1 | medium | declares getDateJoined() | `test/tickets/LDEV0374/test.cfm:24` variable 't3[]' has no component ref |
| 2 | `templ` | 0 | none |  | `core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestSuite.cfc:26` variable 'templ' has no component ref |
| 2 | `this.mimetypes` | 0 | none |  | `core/src/main/cfml/context/doc/StaticResourceProvider.cfc:91` variable 'this.mimeTypes' has no component ref |
| 2 | `threadsct` | 0 | none |  | `test/tickets/LDEV3780.cfc:41` variable 'threadsct' has no component ref |
| 2 | `tmpstr.componentdetails.pack` | 0 | none |  | `core/src/main/java/resource/context/admin/debug/Modern.cfc:1230` variable 'tmpStr.componentDetails.pack' has no component ref |
| 2 | `val → test/tickets/LDEV0966/orm/tblitem.cfc` | 11 | low | declares getType() (11 candidates) | `test/tickets/LDEV1109.cfc:52` variable 'val' has no component ref |
| 2 | `variables` | 2 | none |  | `core/src/main/cfml/context/admin/chartProcess.cfm:34` variable 'variables' has no component ref |
| 2 | `wc` | 0 | none |  | `test/a_debug_build/_MongoDB.cfc:256` variable 'wc' has no component ref |
| 2 | `world` | 0 | none |  | `test/functions/structAppend.cfc:24` variable 'world' has no component ref |
| 1 | `_bundle` | 0 | none |  | `test/_setupTestServices.cfc:786` variable '_bundle' has no component ref |
| 1 | `a → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` | 4 | low | declares equals() (4 candidates) | `test/tickets/_LDEV1851.cfc:22` variable 'a' has no component ref |
| 1 | `a.b → test/general/InlineComponent.cfc` | 5 | low | declares c() (3 candidates) | `test/general/SafeNavigator.cfc:8` variable 'a.b' has no component ref |
| 1 | `adm → test/tickets/LDEV6066.cfc` | 2 | low | declares removeDatasource() (2 candidates) | `test/tickets/LDEV4827.cfc:52` variable 'adm' has no component ref |
| 1 | `adminupdatedcompilesettings.templatecharset → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/components/Administrator.cfc:778` variable 'adminUpdatedCompileSettings.templateCharset' has no component ref |
| 1 | `adminurls` | 0 | none |  | `core/src/main/cfml/context/admin/web.cfm:493` variable 'adminUrls' has no component ref |
| 1 | `adminwebupdatedcompilesettings.templatecharset → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/components/Administrator.cfc:778` variable 'adminWebupdatedCompileSettings.templateCharset' has no component ref |
| 1 | `afterstored → test/tickets/LDEV3963/test3963quoted.cfc` | 5 | low | declares get() (4 candidates) | `test/tickets/LDEV5941/testCommitNoMetadataBump.cfm:19` variable 'afterStored' has no component ref |
| 1 | `animals → test/tickets/LDEV1247.cfc` | 1 | medium | declares Filter() | `test/functions/structFilter.cfc:17` variable 'animals' has no component ref |
| 1 | `application.ormeventlog` | 0 | none |  | `test/orm/events/eventHandler.cfc:72` variable 'application.ormEventLog' has no component ref |
| 1 | `appsessions → core/src/main/java/resource/component/org/lucee/cfml/Ftp.cfc` | 1 | medium | declares remove() | `test/tickets/LDEV5930/evictSession.cfm:19` variable 'appSessions' has no component ref |
| 1 | `args` | 0 | none |  | `test/tickets/LDEV5324/student.cfc:30` variable 'args' has no component ref |
| 1 | `arguments.a.b.c → test/general/InlineComponent.cfc` | 2 | low | declares d() (2 candidates) | `test/general/FunctionListener.cfc:99` variable 'arguments.a.b.c' has no component ref |
| 1 | `arguments.envcl` | 0 | none |  | `test/tickets/LDEV6240.cfc:73` variable 'arguments.envCL' has no component ref |
| 1 | `arguments.key` | 0 | none |  | `core/src/main/cfml/context/gateway/WatchService.cfc:61` variable 'arguments.key' has no component ref |
| 1 | `arguments.l → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 40 | low | declares getLabel() (40 candidates) | `core/src/main/cfml/context/admin/server.logging.cfm:17` variable 'arguments.l' has no component ref |
| 1 | `arguments.name` | 0 | none |  | `test/tickets/LDEV5473.cfc:8` variable 'arguments.name' has no component ref |
| 1 | `arguments.props` | 0 | none |  | `test/_setupTestServices.cfc:516` variable 'arguments.props' has no component ref |
| 1 | `arguments.r → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 40 | low | declares getLabel() (40 candidates) | `core/src/main/cfml/context/admin/server.logging.cfm:17` variable 'arguments.r' has no component ref |
| 1 | `attributes.b` | 0 | none |  | `test/tickets/LDEV5967.cfc:64` variable 'attributes.b' has no component ref |
| 1 | `available` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.detail.cfm:88` variable 'available' has no component ref |
| 1 | `before → test/tickets/LDEV3963/test3963quoted.cfc` | 5 | low | declares get() (4 candidates) | `test/tickets/LDEV5941/testCommitNoMetadataBump.cfm:12` variable 'before' has no component ref |
| 1 | `bookmark → test/orm/many2many/model/Bookmark.cfc` | 1 | high | declares getTags(); named like the receiver 'bookmark' | `test/orm/many2many/index.cfm:69` variable 'bookmark' has no component ref |
| 1 | `bundle.description` | 0 | none |  | `core/src/main/cfml/context/admin/info.bundle.create.cfm:13` variable 'bundle.description' has no component ref |
| 1 | `castedarrayint → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/LDEV5238.cfc:8` variable 'castedArrayInt' has no component ref |
| 1 | `cfftp.returnvalue` | 0 | none |  | `test/_setupTestServices.cfc:372` variable 'cfftp.returnValue' has no component ref |
| 1 | `cfhttp.filecontent` | 0 | none |  | `test/tickets/LDEV1856.cfc:7` variable 'cfhttp.Filecontent' has no component ref |
| 1 | `cfthread.ldev5320_getsession` | 0 | none |  | `test/tickets/LDEV5320/cfml-session/threadSession.cfm:21` variable 'cfthread.ldev5320_getSession' has no component ref |
| 1 | `cl` | 0 | none |  | `test/run-tests.cfm:20` variable 'cl' has no component ref |
| 1 | `class1 → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/LDEV6226.cfc:79` variable 'class1' has no component ref |
| 1 | `class2 → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/LDEV6226.cfc:80` variable 'class2' has no component ref |
| 1 | `classinstance → test/jira/Jira2902/AbsAbs.cfc` | 2 | low | declares execute() (2 candidates) | `test/tickets/LDEV4001/LDEV4001.cfm:29` variable 'classInstance' has no component ref |
| 1 | `classpath` | 0 | none |  | `test/jira/Jira2446.cfc:26` variable 'classpath' has no component ref |
| 1 | `clazz` | 0 | none |  | `test/run-tests.cfm:7` variable 'clazz' has no component ref |
| 1 | `compmetadata.functions` | 0 | none |  | `test/tickets/LDEV2242.cfc:10` variable 'compMetaData.functions' has no component ref |
| 1 | `compmetadata.properties` | 0 | none |  | `test/tickets/LDEV2242.cfc:9` variable 'compMetaData.properties' has no component ref |
| 1 | `configutil → core/src/main/java/resource/component/org/lucee/cfml/Ftp.cfc` | 1 | medium | declares getFile() | `test/tickets/LDEV-4796.cfc:22` variable 'ConfigUtil' has no component ref |
| 1 | `conn` | 0 | none |  | `test/_setupTestServices.cfc:338` variable 'conn' has no component ref |
| 1 | `cookie` | 0 | none |  | `test/tickets/LDEV5288/ldev5288-createJsession.cfm:6` variable 'cookie' has no component ref |
| 1 | `d → core/src/main/java/resource/component/org/lucee/cfml/Ftp.cfc` | 1 | medium | declares remove() | `test/general/Resources.cfc:297` variable 'd' has no component ref |
| 1 | `d → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/general/Resources.cfc:413` variable 'd' has no component ref |
| 1 | `d → test/functions/serializeJSON/_Json.cfc` | 5 | low | declares get() (4 candidates) | `test/functions/Duplicate.cfc:48` variable 'd' has no component ref |
| 1 | `d2 → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/general/Resources.cfc:414` variable 'd2' has no component ref |
| 1 | `data.keywords` | 0 | none |  | `core/src/main/cfml/context/doc/categories.cfm:36` variable 'data.keywords' has no component ref |
| 1 | `data[]` | 0 | none |  | `core/src/main/cfml/context/doc/DocUtils.cfc:79` variable 'data[]' has no component ref |
| 1 | `datasource` | 0 | none |  | `core/src/main/cfml/context/admin/services.datasource.create.cfm:558` variable 'datasource' has no component ref |
| 1 | `datasource.custom` | 0 | none |  | `core/src/main/cfml/context/admin/services.datasource.create.cfm:558` variable 'datasource.custom' has no component ref |
| 1 | `datasources` | 0 | none |  | `core/src/main/cfml/context/admin/data.cfm:14` variable 'datasources' has no component ref |
| 1 | `date1` | 0 | none |  | `test/tickets/LDEV0374/test.cfm:31` variable 'date1' has no component ref |
| 1 | `date2` | 0 | none |  | `test/tickets/LDEV0374/test.cfm:29` variable 'date2' has no component ref |
| 1 | `dd → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/general/Resources.cfc:417` variable 'dd' has no component ref |
| 1 | `deleted` | 0 | none |  | `test/tickets/LDEV4670/ldev4670.cfm:19` variable 'deleted' has no component ref |
| 1 | `desc` | 0 | none |  | `test/tickets/LDEV4889.cfc:118` variable 'desc' has no component ref |
| 1 | `detail.provider → core/src/main/cfml/context/admin/ExtensionProviderProxy.cfc` | 2 | medium | declares getDownloadDetails() | `core/src/main/cfml/context/admin/ext.functions.cfm:153` variable 'detail.provider' has no component ref |
| 1 | `drivers[] → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 48 | low | declares getDescription() (48 candidates) | `core/src/main/cfml/context/admin/debugging.templates.list.cfm:82` variable 'drivers[]' has no component ref |
| 1 | `ds` | 0 | none |  | `test/_setupTestServices.cfc:604` variable 'ds' has no component ref |
| 1 | `e → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV3878.cfc:16` variable 'e' has no component ref |
| 1 | `e → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV3878.cfc:16` variable 'e' has no component ref |
| 1 | `e.message` | 0 | none |  | `test/tickets/LDEV0757.cfc:35` variable 'e.message' has no component ref |
| 1 | `e1` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.detail.cfm:292` variable 'e1' has no component ref |
| 1 | `e2` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.detail.cfm:293` variable 'e2' has no component ref |
| 1 | `el → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` | 26 | low | declares getClass() (26 candidates) | `core/src/main/cfml/context/admin/server.logging.cfm:21` variable 'el' has no component ref |
| 1 | `entry` | 0 | none |  | `core/src/main/cfml/context/debug/modern/reference.cfm:274` variable 'entry' has no component ref |
| 1 | `eval → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/functions/Serialize.cfc:43` variable 'eval' has no component ref |
| 1 | `eventhandlers` | 0 | none |  | `core/src/main/cfml/context/gateway/DirectoryWatcher.cfc:53` variable 'eventHandlers' has no component ref |
| 1 | `ext → test/tickets/LDEV0809.cfc` | 1 | medium | declares ToStruct() | `test/tickets/LDEV5327.cfc:13` variable 'ext' has no component ref |
| 1 | `extensions` | 0 | none |  | `core/src/main/cfml/context/admin/update.cfm:69` variable 'extensions' has no component ref |
| 1 | `extractedfiles → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/LDEV5034.cfc:124` variable 'extractedFiles' has no component ref |
| 1 | `f → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/general/Resources.cfc:412` variable 'f' has no component ref |
| 1 | `files → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/LDEV5034.cfc:122` variable 'files' has no component ref |
| 1 | `headers → test/tickets/LDEV3963/test3963quoted.cfc` | 5 | low | declares get() (4 candidates) | `test/tickets/LDEV6088.cfc:139` variable 'headers' has no component ref |
| 1 | `httpbin` | 0 | none |  | `test/_setupTestServices.cfc:748` variable 'httpbin' has no component ref |
| 1 | `httpresult.cookies` | 0 | none |  | `test/tickets/LDEV4756.cfc:202` variable 'httpResult.cookies' has no component ref |
| 1 | `httpresult.filecontent` | 0 | none |  | `test/tickets/LDEV6178.cfc:79` variable 'httpResult.fileContent' has no component ref |
| 1 | `httpsendresult → core/src/main/java/resource/component/org/lucee/cfml/Result.cfc` | 1 | medium | declares getPrefix() | `test/tickets/LDEV1020.cfc:30` variable 'httpSendResult' has no component ref |
| 1 | `human` | 0 | none |  | `test/functions/structAppend.cfc:28` variable 'human' has no component ref |
| 1 | `inp2 → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV3772.cfc:39` variable 'inp2' has no component ref |
| 1 | `inp2 → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV3772.cfc:39` variable 'inp2' has no component ref |
| 1 | `input` | 0 | none |  | `test/functions/REEscape.cfc:70` variable 'input' has no component ref |
| 1 | `installed.bundles` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.detail.cfm:215` variable 'installed.bundles' has no component ref |
| 1 | `installedversion` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.detail.cfm:302` variable 'installedVersion' has no component ref |
| 1 | `instance → test/general/BinaryTemplates.cfc` | 191 | low | declares test() (175 candidates) | `test/general/Static.cfc:163` variable 'instance' has no component ref |
| 1 | `instr` | 0 | none |  | `test/run-tests.cfm:6` variable 'instr' has no component ref |
| 1 | `interfaces[]` | 0 | none |  | `test/tickets/LDEV1121/test.cfm:4` variable 'interfaces[]' has no component ref |
| 1 | `interfaces[] → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/LDEV1121/test.cfm:4` variable 'interfaces[]' has no component ref |
| 1 | `item → test/tickets/LDEV2056/Base.cfc` | 1 | medium | declares getMemento() | `test/tickets/LDEV2056.cfc:19` variable 'item' has no component ref |
| 1 | `java.lang` | 0 | none |  | `test/tickets/LDEV5807.cfc:66` variable 'java.lang' has no component ref |
| 1 | `java.lang → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV5807.cfc:66` variable 'java.lang' has no component ref |
| 1 | `jcompletor → test/functions/CreateDynamicProxy/osgi/Completor.cfc` | 1 | medium | declares complete() | `test/functions/CreateDynamicProxy/osgi/index.cfm:16` variable 'jCompletor' has no component ref |
| 1 | `jdbcs` | 0 | none |  | `test/_setupTestServices.cfc:873` variable 'jdbcs' has no component ref |
| 1 | `kind → test/tickets/LDEV1818/Sample.cfc` | 4 | low | declares name() (4 candidates) | `core/src/main/cfml/context/gateway/WatchService.cfc:84` variable 'kind' has no component ref |
| 1 | `ldap` | 0 | none |  | `test/_setupTestServices.cfc:742` variable 'ldap' has no component ref |
| 1 | `legend` | 0 | none |  | `test/functions/structCount.cfc:11` variable 'legend' has no component ref |
| 1 | `ljkl.jljl` | 0 | none |  | `test/general/Elvis.cfc:59` variable 'ljkl.jljl' has no component ref |
| 1 | `local.a.b.c → test/general/InlineComponent.cfc` | 2 | low | declares d() (2 candidates) | `test/general/FunctionListener.cfc:79` variable 'local.a.b.c' has no component ref |
| 1 | `local.allchanges.add` | 0 | none |  | `core/src/main/cfml/context/gateway/DirectoryWatcher.cfc:173` variable 'local.allChanges.add' has no component ref |
| 1 | `local.allchanges.change` | 0 | none |  | `core/src/main/cfml/context/gateway/DirectoryWatcher.cfc:173` variable 'local.allChanges.change' has no component ref |
| 1 | `local.allchanges.delete` | 0 | none |  | `core/src/main/cfml/context/gateway/DirectoryWatcher.cfc:173` variable 'local.allChanges.delete' has no component ref |
| 1 | `local.c → test/functions/Duplicate/comps/some/Hello.cfc` | 1 | medium | declares set() | `test/functions/Duplicate.cfc:46` variable 'local.c' has no component ref |
| 1 | `local.currhttpresult.filecontent` | 0 | none |  | `test/tickets/_LDEV1130.cfc:42` variable 'local.currHttpResult.filecontent' has no component ref |
| 1 | `local.resourceproviders` | 0 | none |  | `test/tickets/LDEV4717.cfc:18` variable 'local.resourceProviders' has no component ref |
| 1 | `local.result.filecontent → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/_LDEV1976.cfc:7` variable 'local.result.fileContent' has no component ref |
| 1 | `mapping` | 0 | none |  | `test/general/ZipResource.cfc:27` variable 'mapping' has no component ref |
| 1 | `mappings[]` | 0 | none |  | `test/general/Mappings.cfc:59` variable 'mappings[]' has no component ref |
| 1 | `md` | 0 | none |  | `test/_setupTestServices.cfc:868` variable 'md' has no component ref |
| 1 | `memberfunctionlist` | 0 | none |  | `core/src/main/cfml/context/doc/DocUtils.cfc:130` variable 'memberFunctionList' has no component ref |
| 1 | `memberfunctionlist[]` | 0 | none |  | `core/src/main/cfml/context/doc/DocUtils.cfc:130` variable 'memberFunctionList[]' has no component ref |
| 1 | `memcached` | 0 | none |  | `test/_setupTestServices.cfc:730` variable 'memcached' has no component ref |
| 1 | `messages[] → test/tickets/LDEV0752/SupportTicket.cfc` | 1 | medium | declares getSubject() | `test/tags/Mail2.cfc:301` variable 'messages[]' has no component ref |
| 1 | `metadata.properties` | 0 | none |  | `test/tickets/LDEV4633.cfc:11` variable 'metadata.properties' has no component ref |
| 1 | `mssql` | 0 | none |  | `test/tickets/LDEV1760/AutoDetectMsSql/Application.cfc:9` variable 'msSQL' has no component ref |
| 1 | `mynumb` | 0 | none |  | `test/functions/structsort.cfc:93` variable 'myNumb' has no component ref |
| 1 | `nestedstruct` | 0 | none |  | `test/functions/StructToSorted.cfc:74` variable 'nestedStruct' has no component ref |
| 1 | `obj[] → test/tickets/LDEV1741/App1/model/Foo.cfc` | 40 | low | declares getLabel() (40 candidates) | `test/tickets/LDEV1741/App1/index.cfm:14` variable 'obj[]' has no component ref |
| 1 | `obj[] → test/tickets/LDEV1741/App2/model/Foo.cfc` | 40 | low | declares getLabel() (40 candidates) | `test/tickets/LDEV1741/App2/index.cfm:14` variable 'obj[]' has no component ref |
| 1 | `obj[] → test/tickets/LDEV1984/App1/model/Foo.cfc` | 40 | low | declares getLabel() (40 candidates) | `test/tickets/LDEV1984/App1/index.cfm:14` variable 'obj[]' has no component ref |
| 1 | `oldfiles` | 0 | none |  | `test/datasource/HSQLDB.cfc:64` variable 'oldFiles' has no component ref |
| 1 | `oracle` | 0 | none |  | `test/tickets/LDEV1760/AutoDetectOracle/Application.cfc:9` variable 'oracle' has no component ref |
| 1 | `orderedstruct` | 0 | none |  | `test/functions/structValueArray.cfc:42` variable 'orderedStruct' has no component ref |
| 1 | `paramtypes[] → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV6226.cfc:117` variable 'paramTypes[]' has no component ref |
| 1 | `policies` | 0 | none |  | `test/tickets/LDEV5085/AntiSamyService.cfc:62` variable 'policies' has no component ref |
| 1 | `postgres` | 0 | none |  | `test/tickets/LDEV1760/AutoDetectPostgres/Application.cfc:9` variable 'postgres' has no component ref |
| 1 | `process` | 0 | none |  | `test/tickets/LDEV5510.cfc:26` variable 'process' has no component ref |
| 1 | `propbyname.myproperty1` | 0 | none |  | `test/tickets/LDEV0719.cfc:36` variable 'propByName.myProperty1' has no component ref |
| 1 | `propbyname.myproperty2` | 0 | none |  | `test/tickets/LDEV0719.cfc:41` variable 'propByName.myProperty2' has no component ref |
| 1 | `propbyname.myproperty3` | 0 | none |  | `test/tickets/LDEV0719.cfc:46` variable 'propByName.myProperty3' has no component ref |
| 1 | `propbyname.myproperty4` | 0 | none |  | `test/tickets/LDEV0719.cfc:52` variable 'propByName.myProperty4' has no component ref |
| 1 | `proxiedobject → test/tickets/LDEV-4676.cfc` | 1475 | low | declares run() (1474 candidates) | `test/tickets/LDEV1778.cfc:11` variable 'proxiedObject' has no component ref |
| 1 | `proxy` | 0 | none |  | `core/src/main/cfml/context/admin/server.proxy.cfm:98` variable 'proxy' has no component ref |
| 1 | `proxy → test/functions/CreateDynamicProxy/javaSetting/Test.cfc` | 2 | low | declares hello() (2 candidates) | `test/functions/CreateDynamicProxy/javaSetting/index.cfm:27` variable 'proxy' has no component ref |
| 1 | `proxy → test/jira/Jira3096/Test.cfc` | 2 | low | declares hello() (2 candidates) | `test/jira/Jira3096/index.cfm:27` variable 'proxy' has no component ref |
| 1 | `ps` | 0 | none |  | `test/general/ZipResource.cfc:28` variable 'ps' has no component ref |
| 1 | `q.currentrow` | 0 | none |  | `test/tickets/LDEV2924.cfc:29` variable 'q.currentrow' has no component ref |
| 1 | `q.recordcount` | 0 | none |  | `test/tickets/LDEV2924.cfc:25` variable 'q.recordcount' has no component ref |
| 1 | `q_sessions` | 0 | none |  | `test/tickets/LDEV4670/ldev4670.cfm:25` variable 'q_sessions' has no component ref |
| 1 | `q_ssorted` | 0 | none |  | `test/tickets/LDEV1294.cfc:29` variable 'q_ssorted' has no component ref |
| 1 | `qoq` | 0 | none |  | `test/tickets/LDEV1525/ldev1525.cfm:26` variable 'qoq' has no component ref |
| 1 | `qry → test/tickets/LDEV1247.cfc` | 1 | medium | declares filter() | `core/src/main/cfml/context/admin/web.cfm:385` variable 'qry' has no component ref |
| 1 | `qry.columnlist` | 0 | none |  | `test/tickets/LDEV4581.cfc:7` variable 'qry.columnList' has no component ref |
| 1 | `qry.id → test/tickets/LDEV5074.cfc` | 3 | low | declares toString() (2 candidates) | `test/tickets/LDEV4181.cfc:75` variable 'qry.id' has no component ref |
| 1 | `qry.maxbigint[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:180` variable 'qry.MaxBigInt[]' has no component ref |
| 1 | `qry.maxbigint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:180` variable 'qry.MaxBigInt[]' has no component ref |
| 1 | `qry.maxdecimal[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:239` variable 'qry.MaxDecimal[]' has no component ref |
| 1 | `qry.maxdecimal[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:239` variable 'qry.MaxDecimal[]' has no component ref |
| 1 | `qry.maxdouble[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:221` variable 'qry.MaxDouble[]' has no component ref |
| 1 | `qry.maxdouble[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:221` variable 'qry.MaxDouble[]' has no component ref |
| 1 | `qry.maxfloat[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:203` variable 'qry.MaxFloat[]' has no component ref |
| 1 | `qry.maxfloat[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:203` variable 'qry.MaxFloat[]' has no component ref |
| 1 | `qry.minbigint[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:176` variable 'qry.MinBigInt[]' has no component ref |
| 1 | `qry.minbigint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:176` variable 'qry.MinBigInt[]' has no component ref |
| 1 | `qry.mindecimal[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:230` variable 'qry.MinDecimal[]' has no component ref |
| 1 | `qry.mindecimal[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:230` variable 'qry.MinDecimal[]' has no component ref |
| 1 | `qry.mindouble[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:212` variable 'qry.MinDouble[]' has no component ref |
| 1 | `qry.mindouble[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:212` variable 'qry.MinDouble[]' has no component ref |
| 1 | `qry.minfloat[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:193` variable 'qry.MinFloat[]' has no component ref |
| 1 | `qry.minfloat[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:193` variable 'qry.MinFloat[]' has no component ref |
| 1 | `qry.minint[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:164` variable 'qry.MinInt[]' has no component ref |
| 1 | `qry.minint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:164` variable 'qry.MinInt[]' has no component ref |
| 1 | `qry.minunsignedbigint[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:184` variable 'qry.MinUnsignedBigInt[]' has no component ref |
| 1 | `qry.minunsignedbigint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:184` variable 'qry.MinUnsignedBigInt[]' has no component ref |
| 1 | `qry.minunsignedint[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:168` variable 'qry.MinUnsignedInt[]' has no component ref |
| 1 | `qry.minunsignedint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:168` variable 'qry.MinUnsignedInt[]' has no component ref |
| 1 | `qry.unsignedmaxbigint[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:188` variable 'qry.UnsignedMaxBigInt[]' has no component ref |
| 1 | `qry.unsignedmaxbigint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:188` variable 'qry.UnsignedMaxBigInt[]' has no component ref |
| 1 | `qry.unsignedmaxint[] → test/functions/ORMExecuteQuery/Person.cfc` | 67 | low | declares getName() (67 candidates) | `test/datasource/MySQL.cfc:172` variable 'qry.UnsignedMaxInt[]' has no component ref |
| 1 | `qry.unsignedmaxint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/datasource/MySQL.cfc:172` variable 'qry.UnsignedMaxInt[]' has no component ref |
| 1 | `ramcache → core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` | 26 | medium | declares getClass(); named like the receiver 'ramCache' (26 candidates) | `test/tickets/LDEV1109.cfc:36` variable 'ramCache' has no component ref |
| 1 | `rawcause → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV6347.cfc:54` variable 'rawCause' has no component ref |
| 1 | `rce → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` | 2 | low | declares getValue() (2 candidates) | `test/tickets/LDEV1109.cfc:49` variable 'rce' has no component ref |
| 1 | `redis` | 0 | none |  | `test/_setupTestServices.cfc:736` variable 'redis' has no component ref |
| 1 | `ref → test/tickets/LDEV3963/test3963quoted.cfc` | 5 | low | declares get() (4 candidates) | `test/tickets/LDEV1109.cfc:47` variable 'ref' has no component ref |
| 1 | `repo → test/tickets/LDEV2309.cfc` | 1 | medium | declares getUrl() | `test/tickets/LDEV6317.cfc:10` variable 'repo' has no component ref |
| 1 | `request.componentdetails.pack` | 0 | none |  | `core/src/main/cfml/context/doc/Application.cfc:63` variable 'request.componentDetails.pack' has no component ref |
| 1 | `request.results757` | 0 | none |  | `test/tickets/LDEV0757.cfc:45` variable 'request.results757' has no component ref |
| 1 | `res1.filecontent` | 0 | none |  | `test/tickets/LDEV0677.cfc:32` variable 'res1.filecontent' has no component ref |
| 1 | `res2.filecontent` | 0 | none |  | `test/tickets/LDEV0677.cfc:33` variable 'res2.filecontent' has no component ref |
| 1 | `result → test/tickets/LDEV1214/RandomEntity.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV1214/index.cfm:18` variable 'result' has no component ref |
| 1 | `result → test/tickets/LDEV1962/component1.cfc` | 1 | medium | declares getmessage() | `test/tickets/_LDEV1501.cfc:7` variable 'result' has no component ref |
| 1 | `result.cookies` | 0 | none |  | `test/tickets/LDEV4756.cfc:202` variable 'result.cookies' has no component ref |
| 1 | `result4.filecontent` | 0 | none |  | `test/tickets/_LDEV1659.cfc:12` variable 'result4.filecontent' has no component ref |
| 1 | `results.filecontent` | 0 | none |  | `test/tickets/_LDEV1130.cfc:42` variable 'results.filecontent' has no component ref |
| 1 | `resultvar` | 0 | none |  | `test/tickets/LDEV4425/test4425.cfm:12` variable 'resultVar' has no component ref |
| 1 | `rtn` | 0 | none |  | `core/src/main/cfml/context/admin/ext.functions.cfm:129` variable 'rtn' has no component ref |
| 1 | `s → test/jira/Jira2620.cfc` | 1 | medium | declares createFile() | `test/general/Resources.cfc:337` variable 's' has no component ref |
| 1 | `security.file_access` | 0 | none |  | `test/tickets/LDEV6242.cfc:43` variable 'security.file_access' has no component ref |
| 1 | `server.ldev3478[]` | 0 | none |  | `test/tickets/LDEV3478.cfc:143` variable 'server.LDEV3478[]' has no component ref |
| 1 | `server.system.environment` | 0 | none |  | `core/src/main/java/resource/component/org/lucee/cfml/Administrator.cfc:2989` variable 'Server.system.environment' has no component ref |
| 1 | `server.system.properties` | 0 | none |  | `core/src/main/java/resource/component/org/lucee/cfml/Administrator.cfc:2993` variable 'Server.system.properties' has no component ref |
| 1 | `serverinstalled` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.detail.cfm:84` variable 'serverInstalled' has no component ref |
| 1 | `settings.sessioncommitinterval` | 0 | none |  | `test/tickets/LDEV6331/settings.cfm:5` variable 'settings.sessionCommitInterval' has no component ref |
| 1 | `settings.sessiontimeout` | 0 | none |  | `test/tickets/LDEV6331/settings.cfm:6` variable 'settings.sessionTimeout' has no component ref |
| 1 | `settings.xmlfeatures` | 0 | none |  | `test/tickets/LDEV4348/LDEV4348.cfm:3` variable 'settings.xmlFeatures' has no component ref |
| 1 | `settingsafter.templatecharset → test/tickets/LDEV1818/Sample.cfc` | 4 | low | declares name() (4 candidates) | `test/tickets/LDEV6085.cfc:116` variable 'settingsAfter.templateCharset' has no component ref |
| 1 | `settingsbefore.templatecharset → test/tickets/LDEV1818/Sample.cfc` | 4 | low | declares name() (4 candidates) | `test/tickets/LDEV6085.cfc:91` variable 'settingsBefore.templateCharset' has no component ref |
| 1 | `smtpserver` | 0 | none |  | `test/tags/_Mail.cfc:56` variable 'smtpServer' has no component ref |
| 1 | `src.dumpwriters` | 0 | none |  | `test/tickets/LDEV5758.cfc:259` variable 'src.dumpWriters' has no component ref |
| 1 | `ss` | 0 | none |  | `test/general/Resources.cfc:339` variable 'ss' has no component ref |
| 1 | `ss → core/src/main/java/resource/component/org/lucee/cfml/Query.cfc` | 1 | medium | declares new() | `test/functions/GetComponentStaticScope.cfc:28` variable 'ss' has no component ref |
| 1 | `sss → test/jira/Jira2620.cfc` | 1 | medium | declares createFile() | `test/general/Resources.cfc:341` variable 'sss' has no component ref |
| 1 | `st.template` | 0 | none |  | `test/_testRunner.cfc:247` variable 'st.template' has no component ref |
| 1 | `static → test/tickets/LDEV1227/comp1.cfc` | 1 | medium | declares myUCase() | `test/tickets/LDEV1227/comp2.cfc:26` variable 'static' has no component ref |
| 1 | `static → test/tickets/LDEV1227/comp2.cfc` | 1 | medium | declares myUCase2() | `test/tickets/LDEV1227/comp2.cfc:23` variable 'static' has no component ref |
| 1 | `static → test/tickets/LDEV1431/test.cfc` | 1 | medium | declares testStatic() | `test/tickets/LDEV1431/test.cfc:7` variable 'static' has no component ref |
| 1 | `stream` | 0 | none |  | `test/run-tests.cfm:23` variable 'stream' has no component ref |
| 1 | `struct1` | 0 | none |  | `test/tickets/_LDEV2681.cfc:16` variable 'struct1' has no component ref |
| 1 | `struct1 → test/tickets/LDEV0270.cfc` | 2 | low | declares delete() (2 candidates) | `test/tickets/_LDEV2681.cfc:18` variable 'struct1' has no component ref |
| 1 | `struct1.children` | 0 | none |  | `test/tickets/LDEV2608.cfc:27` variable 'struct1.children' has no component ref |
| 1 | `struct2` | 0 | none |  | `test/tickets/_LDEV2681.cfc:21` variable 'struct2' has no component ref |
| 1 | `struct2 → test/tickets/LDEV0270.cfc` | 2 | low | declares delete() (2 candidates) | `test/tickets/_LDEV2681.cfc:23` variable 'struct2' has no component ref |
| 1 | `struct2.parent.children` | 0 | none |  | `test/tickets/LDEV2608.cfc:28` variable 'struct2.parent.children' has no component ref |
| 1 | `t → test/orm/many2many/model/Bookmark.cfc` | 96 | low | declares getId() (96 candidates) | `test/orm/many2many/index.cfm:72` variable 't' has no component ref |
| 1 | `t2 → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV1831.cfc:25` variable 't2' has no component ref |
| 1 | `tags` | 0 | none |  | `test/orm/many2many/index.cfm:71` variable 'tags' has no component ref |
| 1 | `tb → test/tickets/LDEV5601.cfc` | 4 | medium | declares getVersion() | `test/run-tests.cfm:454` variable 'tb' has no component ref |
| 1 | `test` | 0 | none |  | `test/tickets/LDEV3070/LDEV3070.cfm:8` variable 'test' has no component ref |
| 1 | `test → test/tickets/LDEV2390/Test.cfc` | 12 | medium | declares foo(); named like the receiver 'test' (10 candidates) | `test/tickets/LDEV1487.cfc:34` variable 'test' has no component ref |
| 1 | `testquery` | 0 | none |  | `test/tickets/LDEV3640.cfc:85` variable 'testquery' has no component ref |
| 1 | `testquery → test/tickets/LDEV3963/test3963quoted.cfc` | 5 | low | declares get() (4 candidates) | `test/tickets/LDEV3640.cfc:85` variable 'testquery' has no component ref |
| 1 | `testquery2` | 0 | none |  | `test/tickets/LDEV3640.cfc:96` variable 'testquery2' has no component ref |
| 1 | `testquery2 → test/tickets/LDEV3963/test3963quoted.cfc` | 5 | low | declares get() (4 candidates) | `test/tickets/LDEV3640.cfc:96` variable 'testquery2' has no component ref |
| 1 | `testresults` | 0 | none |  | `test/_testRunner.cfc:111` variable 'testResults' has no component ref |
| 1 | `tmplist` | 0 | none |  | `test/tickets/LDEV1142.cfc:7` variable 'tmpList' has no component ref |
| 1 | `tmpqry.columnlist` | 0 | none |  | `test/tickets/LDEV1142.cfc:13` variable 'tmpQry.columnList' has no component ref |
| 1 | `uncleaned` | 0 | none |  | `test/tickets/LDEV5085/AntiSamyService.cfc:73` variable 'uncleaned' has no component ref |
| 1 | `updatedata.changelog` | 0 | none |  | `core/src/main/cfml/context/admin/services.updateold.cfm:87` variable 'updateData.changelog' has no component ref |
| 1 | `url.pluginaction` | 0 | none |  | `core/src/main/cfml/context/admin/plugin.cfm:5` variable 'url.pluginAction' has no component ref |
| 1 | `val` | 0 | none |  | `core/src/main/cfml/context/debug/modern/reference.cfm:280` variable 'val' has no component ref |
| 1 | `variables.a.b.c → test/general/InlineComponent.cfc` | 2 | low | declares d() (2 candidates) | `test/general/FunctionListener.cfc:89` variable 'variables.a.b.c' has no component ref |
| 1 | `variables.qry → test/tickets/LDEV0809.cfc` | 1 | medium | declares ToStruct() | `test/functions/QueryToStruct.cfc:26` variable 'variables.qry' has no component ref |
| 1 | `vs` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.detail.cfm:303` variable 'vs' has no component ref |
| 1 | `wb` | 0 | none |  | `test/tickets/_LDEV1501.cfc:6` variable 'wb' has no component ref |
| 1 | `white2` | 0 | none |  | `test/tickets/LDEV5640.cfc:16` variable 'white2' has no component ref |
| 1 | `world → core/src/main/java/resource/component/org/lucee/cfml/HelperBase.cfc` | 1 | medium | declares clear() | `test/functions/structclear.cfc:14` variable 'world' has no component ref |
| 1 | `ws → test/tickets/LDEV3629/test.cfc` | 1 | medium | declares getCustomer() | `test/tickets/LDEV3629.cfc:8` variable 'ws' has no component ref |

<details><summary>Groups with several candidates</summary>

- `driver → core/src/main/java/resource/context/admin/mailservers/GMX.cfc` — 14 finding(s), 8 candidate(s):
  - low `core/src/main/java/resource/context/admin/mailservers/GMX.cfc` — declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS()
  - low `core/src/main/java/resource/context/admin/mailservers/GMail.cfc` — declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS()
  - low `core/src/main/java/resource/context/admin/mailservers/MailCom.cfc` — declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS()
  - low `core/src/main/java/resource/context/admin/mailservers/MailServer.cfc` — declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS()
  - low `core/src/main/java/resource/context/admin/mailservers/Other.cfc` — declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS()
  - low `core/src/main/java/resource/context/admin/mailservers/Outlook.cfc` — declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS()
  - low `core/src/main/java/resource/context/admin/mailservers/Yahoo.cfc` — declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS()
  - low `core/src/main/java/resource/context/admin/mailservers/iCloud.cfc` — declares getShortName(), getLabel(), getHost(), getPort(), useSSL(), useTLS()
- `application.objects.utils → core/src/main/cfml/context/doc/DocUtils.cfc` — 9 finding(s), 3 candidate(s):
  - low `core/src/main/cfml/context/doc/DocUtils.cfc` — declares getAllFunctions()
  - low `core/src/main/java/resource/context/admin/debug/Modern.cfc` — declares getAllFunctions()
  - low `core/src/main/java/resource/context/admin/info/Info.cfc` — declares getAllFunctions()
- `msg → test/tickets/LDEV5074.cfc` — 6 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `result.num → test/functions/ORMExecuteQuery/Person.cfc` — 6 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `result.num → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 6 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `result.str → test/functions/ORMExecuteQuery/Person.cfc` — 6 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `result.str → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 6 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `rtn → test/jira/Jira1460.cfc` — 6 finding(s), 5 candidate(s):
  - low `test/jira/Jira1460.cfc` — declares a(), b()
  - low `test/jira/Jira2726.cfc` — declares a(), b()
  - low `test/functions/_SerializeJSON2.cfc` — declares a(), b()
  - low `test/general/InlineComponent.cfc` — declares a(), b()
  - low `test/general/subComponent/TestSubScript.cfc` — declares a(), b()
- `s → test/jira/Jira0713/Test.cfc` — 6 finding(s), 67 candidate(s):
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/jira/Jira2275/orm/Field.cfc` — declares getName()
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - … 59 more
- `s → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 6 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `driver → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 5 finding(s), 21 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/AsynchronousEvents.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/DirectoryWatcher.cfc` — declares getLabel(), getClass()
  - … 13 more
- `drivers[] → core/src/main/java/resource/context/admin/mailservers/GMX.cfc` — 5 finding(s), 8 candidate(s):
  - low `core/src/main/java/resource/context/admin/mailservers/GMX.cfc` — declares getHost(), getPort(), useSSL(), useTLS(), getLabel()
  - low `core/src/main/java/resource/context/admin/mailservers/GMail.cfc` — declares getHost(), getPort(), useSSL(), useTLS(), getLabel()
  - low `core/src/main/java/resource/context/admin/mailservers/MailCom.cfc` — declares getHost(), getPort(), useSSL(), useTLS(), getLabel()
  - low `core/src/main/java/resource/context/admin/mailservers/MailServer.cfc` — declares getHost(), getPort(), useSSL(), useTLS(), getLabel()
  - low `core/src/main/java/resource/context/admin/mailservers/Other.cfc` — declares getHost(), getPort(), useSSL(), useTLS(), getLabel()
  - low `core/src/main/java/resource/context/admin/mailservers/Outlook.cfc` — declares getHost(), getPort(), useSSL(), useTLS(), getLabel()
  - low `core/src/main/java/resource/context/admin/mailservers/Yahoo.cfc` — declares getHost(), getPort(), useSSL(), useTLS(), getLabel()
  - low `core/src/main/java/resource/context/admin/mailservers/iCloud.cfc` — declares getHost(), getPort(), useSSL(), useTLS(), getLabel()
- `_driver → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — 4 finding(s), 26 candidate(s):
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass(); named like the receiver 'driver'
  - low `core/src/main/java/resource/context/admin/dbdriver/types/Driver.cfc` — declares getClass(); named like the receiver 'driver'
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` — declares getClass()
  - … 18 more
- `appender → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 4 finding(s), 21 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/AsynchronousEvents.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/DirectoryWatcher.cfc` — declares getLabel(), getClass()
  - … 13 more
- `drivers[] → core/src/main/java/resource/context/admin/debug/Classic.cfc` — 4 finding(s), 4 candidate(s):
  - low `core/src/main/java/resource/context/admin/debug/Classic.cfc` — declares getDescription(), getId(), getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Comment.cfc` — declares getDescription(), getId(), getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Modern.cfc` — declares getDescription(), getId(), getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Simple.cfc` — declares getDescription(), getId(), getLabel()
- `jdbc → test/functions/serializeJSON/_Json.cfc` — 4 finding(s), 4 candidate(s):
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
- `layout → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 4 finding(s), 21 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/AsynchronousEvents.cfc` — declares getLabel(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/DirectoryWatcher.cfc` — declares getLabel(), getClass()
  - … 13 more
- `mongo → test/functions/ORMExecuteQuery/Person.cfc` — 4 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `notexa.b.c → test/general/InlineComponent.cfc` — 4 finding(s), 2 candidate(s):
  - low `test/general/InlineComponent.cfc` — declares d()
  - low `test/general/subComponent/TestSubScript.cfc` — declares d()
- `result.tnum → test/tickets/LDEV0078/Test.cfc` — 4 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `result.tnum → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 4 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `result.tstr → test/tickets/LDEV0078/Test.cfc` — 4 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `result.tstr → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 4 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `tz → test/jira/Jira2275/orm/FieldLink.cfc` — 4 finding(s), 14 candidate(s):
  - low `test/jira/Jira2275/orm/FieldLink.cfc` — declares getDisplayName()
  - low `core/src/main/java/resource/context/admin/aidriver/Field.cfc` — declares getDisplayName()
  - low `core/src/main/java/resource/context/admin/aidriver/Group.cfc` — declares getDisplayName()
  - low `core/src/main/java/resource/context/admin/cdriver/Field.cfc` — declares getDisplayName()
  - low `core/src/main/java/resource/context/admin/cdriver/Group.cfc` — declares getDisplayName()
  - low `core/src/main/java/resource/context/admin/debug/Field.cfc` — declares getDisplayName()
  - low `core/src/main/java/resource/context/admin/debug/Group.cfc` — declares getDisplayName()
  - low `core/src/main/java/resource/context/admin/gdriver/Field.cfc` — declares getDisplayName()
  - … 6 more
- `world1 → test/tickets/LDEV0270.cfc` — 4 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV0270.cfc` — declares delete()
  - low `core/src/main/cfml/context/admin/plugin/Resource/Action.cfc` — declares delete()
- `a → test/general/InlineComponent.cfc` — 3 finding(s), 5 candidate(s):
  - low `test/general/InlineComponent.cfc` — declares b()
  - low `test/general/subComponent/TestSubScript.cfc` — declares b()
  - low `test/functions/_SerializeJSON2.cfc` — declares b()
  - low `test/jira/Jira1460.cfc` — declares b()
  - low `test/jira/Jira2726.cfc` — declares b()
- `a.b.c → test/general/InlineComponent.cfc` — 3 finding(s), 2 candidate(s):
  - low `test/general/InlineComponent.cfc` — declares d()
  - low `test/general/subComponent/TestSubScript.cfc` — declares d()
- `cfcc → test/jira/Jira0713/Test.cfc` — 3 finding(s), 56 candidate(s):
  - low `test/jira/Jira0713/Test.cfc` — declares getName(), setName()
  - low `test/jira/Jira2275/orm/Field.cfc` — declares getName(), setName()
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName(), setName()
  - low `test/orm/transRollback/Person.cfc` — declares getName(), setName()
  - low `test/orm/transSave/Person.cfc` — declares getName(), setName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName(), setName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName(), setName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName(), setName()
  - … 48 more
- `dr → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 3 finding(s), 26 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/AsynchronousEvents.cfc` — declares getClass()
  - … 18 more
- `driver → core/src/main/java/resource/context/admin/debug/Classic.cfc` — 3 finding(s), 5 candidate(s):
  - low `core/src/main/java/resource/context/admin/debug/Classic.cfc` — declares output()
  - low `core/src/main/java/resource/context/admin/debug/Comment.cfc` — declares output()
  - low `core/src/main/java/resource/context/admin/debug/Debug.cfc` — declares output()
  - low `core/src/main/java/resource/context/admin/debug/Modern.cfc` — declares output()
  - low `core/src/main/java/resource/context/admin/debug/Simple.cfc` — declares output()
- `fruit → test/tickets/LDEV0423/Fruit.cfc` — 3 finding(s), 67 candidate(s):
  - medium `test/tickets/LDEV0423/Fruit.cfc` — declares getName(); named like the receiver 'fruit'
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `item → test/tickets/LDEV0078/Test.cfc` — 3 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `proxy → test/tickets/LDEV0078/Test.cfc` — 3 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `proxy → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 3 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `res.object → test/tickets/LDEV0078/Test.cfc` — 3 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `res.object → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 3 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `sct → test/tickets/LDEV3963/test3963quoted.cfc` — 3 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
- `_defaultdriver → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 2 finding(s), 40 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Classic.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Comment.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Modern.cfc` — declares getLabel()
  - … 32 more
- `args.num → test/functions/ORMExecuteQuery/Person.cfc` — 2 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `args.num → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 2 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `args.str → test/functions/ORMExecuteQuery/Person.cfc` — 2 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `args.str → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 2 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `arguments.debugtemplate → core/src/main/java/resource/context/admin/debug/Classic.cfc` — 2 finding(s), 5 candidate(s):
  - low `core/src/main/java/resource/context/admin/debug/Classic.cfc` — declares output()
  - low `core/src/main/java/resource/context/admin/debug/Comment.cfc` — declares output()
  - low `core/src/main/java/resource/context/admin/debug/Debug.cfc` — declares output()
  - low `core/src/main/java/resource/context/admin/debug/Modern.cfc` — declares output()
  - low `core/src/main/java/resource/context/admin/debug/Simple.cfc` — declares output()
- `c → test/functions/serializeJSON/_Json.cfc` — 2 finding(s), 4 candidate(s):
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
- `code → test/orm/clearSession/Code.cfc` — 2 finding(s), 7 candidate(s):
  - low `test/orm/clearSession/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/orm/events/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/orm/many2one/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/orm/simple/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/tickets/LDEV0881/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/orm/many2one/Ref.cfc` — declares getId(), getCode()
  - low `test/tickets/LDEV0881/Ref.cfc` — declares getId(), getCode()
- `code → test/tickets/LDEV0881/Code.cfc` — 2 finding(s), 7 candidate(s):
  - low `test/tickets/LDEV0881/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/orm/clearSession/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/orm/events/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/orm/many2one/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/orm/simple/Code.cfc` — declares getId(), getCode(); named like the receiver 'code'
  - low `test/tickets/LDEV0881/Ref.cfc` — declares getId(), getCode()
  - low `test/orm/many2one/Ref.cfc` — declares getId(), getCode()
- `d → core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` — 2 finding(s), 6 candidate(s):
  - low `core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` — declares getCFCPath(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/AsynchronousEvents.cfc` — declares getCFCPath(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/DirectoryWatcher.cfc` — declares getCFCPath(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/JMS.cfc` — declares getCFCPath(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/MailWatcher.cfc` — declares getCFCPath(), getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/TaskGatewayDriver.cfc` — declares getCFCPath(), getClass()
- `data.ps → core/src/main/cfml/context/gateway/WatchService.cfc` — 2 finding(s), 2 candidate(s):
  - low `core/src/main/cfml/context/gateway/WatchService.cfc` — declares close()
  - low `core/src/main/java/resource/component/org/lucee/cfml/Ftp.cfc` — declares close()
- `driver → core/src/main/java/resource/context/admin/aidriver/Field.cfc` — 2 finding(s), 67 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Field.cfc` — declares getName()
  - low `core/src/main/java/resource/context/admin/cdriver/Field.cfc` — declares getName()
  - low `core/src/main/java/resource/context/admin/dbdriver/Other.cfc` — declares getName()
  - low `core/src/main/java/resource/context/admin/debug/Field.cfc` — declares getName()
  - low `core/src/main/java/resource/context/admin/gdriver/Field.cfc` — declares getName()
  - low `core/src/main/java/resource/component/org/lucee/cfml/Query.cfc` — declares getName()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/Field.cfc` — declares getName()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/IDriver.cfc` — declares getName()
  - … 59 more
- `dupmap → test/tickets/LDEV0078/Test.cfc` — 2 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `dupmap → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 2 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `newclass → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — 2 finding(s), 2 candidate(s):
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares init(), getValue()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/Driver.cfc` — declares init(), getValue()
- `odnsdomain → test/tickets/LDEV0078/Test.cfc` — 2 finding(s), 96 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getId()
  - low `test/tickets/LDEV0087/Entity.cfc` — declares getId()
  - low `test/tickets/LDEV0096/Entity.cfc` — declares getId()
  - low `test/tickets/LDEV0374/users.cfc` — declares getId()
  - low `test/tickets/LDEV0421/test.cfc` — declares getId()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getId()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getId()
  - low `test/tickets/LDEV0490/haspersistent.cfc` — declares getId()
  - … 88 more
- `qry.driver → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 2 finding(s), 40 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Classic.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Comment.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Modern.cfc` — declares getLabel()
  - … 32 more
- `r → test/tickets/LDEV6138/TestEntity.cfc` — 2 finding(s), 56 candidate(s):
  - low `test/tickets/LDEV6138/TestEntity.cfc` — declares getName(), setName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName(), setName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName(), setName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName(), setName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName(), setName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName(), setName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName(), setName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName(), setName()
  - … 48 more
- `ref → test/orm/many2one/Ref.cfc` — 2 finding(s), 7 candidate(s):
  - low `test/orm/many2one/Ref.cfc` — declares getId(), getCode(); named like the receiver 'ref'
  - low `test/tickets/LDEV0881/Ref.cfc` — declares getId(), getCode(); named like the receiver 'ref'
  - low `test/orm/clearSession/Code.cfc` — declares getId(), getCode()
  - low `test/orm/events/Code.cfc` — declares getId(), getCode()
  - low `test/orm/many2one/Code.cfc` — declares getId(), getCode()
  - low `test/orm/simple/Code.cfc` — declares getId(), getCode()
  - low `test/tickets/LDEV0881/Code.cfc` — declares getId(), getCode()
- `ref → test/tickets/LDEV0881/Ref.cfc` — 2 finding(s), 7 candidate(s):
  - low `test/tickets/LDEV0881/Ref.cfc` — declares getId(), getCode(); named like the receiver 'ref'
  - low `test/orm/many2one/Ref.cfc` — declares getId(), getCode(); named like the receiver 'ref'
  - low `test/tickets/LDEV0881/Code.cfc` — declares getId(), getCode()
  - low `test/orm/clearSession/Code.cfc` — declares getId(), getCode()
  - low `test/orm/events/Code.cfc` — declares getId(), getCode()
  - low `test/orm/many2one/Code.cfc` — declares getId(), getCode()
  - low `test/orm/simple/Code.cfc` — declares getId(), getCode()
- `result → test/tickets/LDEV5074.cfc` — 2 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `struct1 → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — 2 finding(s), 4 candidate(s):
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares equals()
  - low `core/src/main/java/resource/context/admin/dbdriver/Other.cfc` — declares equals()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/Driver.cfc` — declares equals()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/IDatasource.cfc` — declares equals()
- `val → test/tickets/LDEV0966/orm/tblitem.cfc` — 2 finding(s), 11 candidate(s):
  - low `test/tickets/LDEV0966/orm/tblitem.cfc` — declares getType()
  - low `test/jira/Jira2275/orm/FieldLink.cfc` — declares getType()
  - low `core/src/main/java/resource/context/admin/aidriver/Field.cfc` — declares getType()
  - low `core/src/main/java/resource/context/admin/cdriver/Field.cfc` — declares getType()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getType()
  - low `core/src/main/java/resource/context/admin/debug/Field.cfc` — declares getType()
  - low `core/src/main/java/resource/context/admin/gdriver/Field.cfc` — declares getType()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/Driver.cfc` — declares getType()
  - … 3 more
- `a → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — 1 finding(s), 4 candidate(s):
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares equals()
  - low `core/src/main/java/resource/context/admin/dbdriver/Other.cfc` — declares equals()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/Driver.cfc` — declares equals()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/IDatasource.cfc` — declares equals()
- `a.b → test/general/InlineComponent.cfc` — 1 finding(s), 3 candidate(s):
  - low `test/general/InlineComponent.cfc` — declares c()
  - low `test/general/subComponent/TestSubScript.cfc` — declares c()
  - low `test/jira/Jira2726.cfc` — declares c()
- `adm → test/tickets/LDEV6066.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV6066.cfc` — declares removeDatasource()
  - low `core/src/main/java/resource/component/org/lucee/cfml/Administrator.cfc` — declares removeDatasource()
- `adminupdatedcompilesettings.templatecharset → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `adminwebupdatedcompilesettings.templatecharset → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `afterstored → test/tickets/LDEV3963/test3963quoted.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
- `arguments.a.b.c → test/general/InlineComponent.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/general/InlineComponent.cfc` — declares d()
  - low `test/general/subComponent/TestSubScript.cfc` — declares d()
- `arguments.l → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 1 finding(s), 40 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Classic.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Comment.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Modern.cfc` — declares getLabel()
  - … 32 more
- `arguments.r → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 1 finding(s), 40 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Classic.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Comment.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/debug/Modern.cfc` — declares getLabel()
  - … 32 more
- `before → test/tickets/LDEV3963/test3963quoted.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
- `castedarrayint → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `class1 → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `class2 → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `classinstance → test/jira/Jira2902/AbsAbs.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/jira/Jira2902/AbsAbs.cfc` — declares execute()
  - low `core/src/main/java/resource/component/org/lucee/cfml/Query.cfc` — declares execute()
- `d → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `d → test/functions/serializeJSON/_Json.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
- `d2 → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `dd → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `drivers[] → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 1 finding(s), 48 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getDescription()
  - low `core/src/main/java/resource/context/admin/aidriver/Field.cfc` — declares getDescription()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getDescription()
  - low `core/src/main/java/resource/context/admin/aidriver/Group.cfc` — declares getDescription()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getDescription()
  - low `core/src/main/java/resource/context/admin/cdriver/Field.cfc` — declares getDescription()
  - low `core/src/main/java/resource/context/admin/cdriver/Group.cfc` — declares getDescription()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getDescription()
  - … 40 more
- `e → test/tickets/LDEV0078/Test.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `e → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `el → core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — 1 finding(s), 26 candidate(s):
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/ActiveMQ.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/gdriver/AsynchronousEvents.cfc` — declares getClass()
  - … 18 more
- `eval → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `extractedfiles → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `f → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `files → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `headers → test/tickets/LDEV3963/test3963quoted.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
- `inp2 → test/tickets/LDEV0078/Test.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `inp2 → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `instance → test/general/BinaryTemplates.cfc` — 1 finding(s), 175 candidate(s):
  - low `test/general/BinaryTemplates.cfc` — declares test()
  - low `test/general/Resources.cfc` — declares test()
  - low `test/general/SafeNavigator.cfc` — declares test()
  - low `test/general/ApplicationListener2/Test.cfc` — declares test()
  - low `test/general/abstract/AbstractFunctionsOnlyInAbstractComponents.cfc` — declares test()
  - low `test/general/modifiers/AccessModifier.cfc` — declares test()
  - low `test/general/modifiers/All.cfc` — declares test()
  - low `test/general/modifiers/NoModifiers.cfc` — declares test()
  - … 167 more
- `interfaces[] → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `java.lang → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `kind → test/tickets/LDEV1818/Sample.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV1818/Sample.cfc` — declares name()
  - low `test/tickets/LDEV1818/test.cfc` — declares name()
  - low `test/tickets/LDEV0279/success/Irate.cfc` — declares name()
  - low `test/tickets/LDEV0279/success/Rate.cfc` — declares name()
- `local.a.b.c → test/general/InlineComponent.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/general/InlineComponent.cfc` — declares d()
  - low `test/general/subComponent/TestSubScript.cfc` — declares d()
- `local.result.filecontent → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `obj[] → test/tickets/LDEV1741/App1/model/Foo.cfc` — 1 finding(s), 40 candidate(s):
  - low `test/tickets/LDEV1741/App1/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1741/App2/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1937/msSql.cfc` — declares getLabel()
  - low `test/tickets/LDEV1937/mySql.cfc` — declares getLabel()
  - low `test/tickets/LDEV1641/specs/CardProcessorType.cfc` — declares getLabel()
  - low `test/tickets/LDEV1659/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1984/App1/model/Foo.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel()
  - … 32 more
- `obj[] → test/tickets/LDEV1741/App2/model/Foo.cfc` — 1 finding(s), 40 candidate(s):
  - low `test/tickets/LDEV1741/App2/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1741/App1/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1937/msSql.cfc` — declares getLabel()
  - low `test/tickets/LDEV1937/mySql.cfc` — declares getLabel()
  - low `test/tickets/LDEV1641/specs/CardProcessorType.cfc` — declares getLabel()
  - low `test/tickets/LDEV1659/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1984/App1/model/Foo.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel()
  - … 32 more
- `obj[] → test/tickets/LDEV1984/App1/model/Foo.cfc` — 1 finding(s), 40 candidate(s):
  - low `test/tickets/LDEV1984/App1/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1937/msSql.cfc` — declares getLabel()
  - low `test/tickets/LDEV1937/mySql.cfc` — declares getLabel()
  - low `test/tickets/LDEV1641/specs/CardProcessorType.cfc` — declares getLabel()
  - low `test/tickets/LDEV1659/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1741/App1/model/Foo.cfc` — declares getLabel()
  - low `test/tickets/LDEV1741/App2/model/Foo.cfc` — declares getLabel()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getLabel()
  - … 32 more
- `paramtypes[] → test/tickets/LDEV0078/Test.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `proxiedobject → test/tickets/LDEV-4676.cfc` — 1 finding(s), 1474 candidate(s):
  - low `test/tickets/LDEV-4676.cfc` — declares run()
  - low `test/tickets/LDEV-4680.cfc` — declares run()
  - low `test/tickets/LDEV-4695.cfc` — declares run()
  - low `test/tickets/LDEV-4795.cfc` — declares run()
  - low `test/tickets/LDEV-4796.cfc` — declares run()
  - low `test/tickets/LDEV-5093.cfc` — declares run()
  - low `test/tickets/LDEV0028.cfc` — declares run()
  - low `test/tickets/LDEV0066.cfc` — declares run()
  - … 1466 more
- `proxy → test/functions/CreateDynamicProxy/javaSetting/Test.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/functions/CreateDynamicProxy/javaSetting/Test.cfc` — declares hello()
  - low `test/jira/Jira3096/Test.cfc` — declares hello()
- `proxy → test/jira/Jira3096/Test.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/jira/Jira3096/Test.cfc` — declares hello()
  - low `test/functions/CreateDynamicProxy/javaSetting/Test.cfc` — declares hello()
- `qry.id → test/tickets/LDEV5074.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV5074.cfc` — declares toString()
  - low `test/tickets/LDEV5074/Test.cfc` — declares toString()
- `qry.maxbigint[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.maxbigint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.maxdecimal[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.maxdecimal[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.maxdouble[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.maxdouble[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.maxfloat[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.maxfloat[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.minbigint[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.minbigint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.mindecimal[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.mindecimal[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.mindouble[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.mindouble[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.minfloat[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.minfloat[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.minint[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.minint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.minunsignedbigint[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.minunsignedbigint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.minunsignedint[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.minunsignedint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.unsignedmaxbigint[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.unsignedmaxbigint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `qry.unsignedmaxint[] → test/functions/ORMExecuteQuery/Person.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc` — declares getName()
  - low `test/jira/Jira0713/Test.cfc` — declares getName()
  - low `test/orm/transRollback/Person.cfc` — declares getName()
  - low `test/orm/transSave/Person.cfc` — declares getName()
  - low `test/orm/transSaveExCommit/Person.cfc` — declares getName()
  - low `test/orm/transSaveFlush/Person.cfc` — declares getName()
  - low `test/orm/transSavepoint/Person.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - … 59 more
- `qry.unsignedmaxint[] → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `ramcache → core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — 1 finding(s), 26 candidate(s):
  - medium `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass(); named like the receiver 'ramCache'
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `rawcause → test/tickets/LDEV0078/Test.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `rce → core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — 1 finding(s), 2 candidate(s):
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getValue()
  - low `core/src/main/java/resource/context/admin/dbdriver/types/Driver.cfc` — declares getValue()
- `ref → test/tickets/LDEV3963/test3963quoted.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
- `result → test/tickets/LDEV1214/RandomEntity.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `settingsafter.templatecharset → test/tickets/LDEV1818/Sample.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV1818/Sample.cfc` — declares name()
  - low `test/tickets/LDEV1818/test.cfc` — declares name()
  - low `test/tickets/LDEV0279/success/Irate.cfc` — declares name()
  - low `test/tickets/LDEV0279/success/Rate.cfc` — declares name()
- `settingsbefore.templatecharset → test/tickets/LDEV1818/Sample.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV1818/Sample.cfc` — declares name()
  - low `test/tickets/LDEV1818/test.cfc` — declares name()
  - low `test/tickets/LDEV0279/success/Irate.cfc` — declares name()
  - low `test/tickets/LDEV0279/success/Rate.cfc` — declares name()
- `struct1 → test/tickets/LDEV0270.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV0270.cfc` — declares delete()
  - low `core/src/main/cfml/context/admin/plugin/Resource/Action.cfc` — declares delete()
- `struct2 → test/tickets/LDEV0270.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV0270.cfc` — declares delete()
  - low `core/src/main/cfml/context/admin/plugin/Resource/Action.cfc` — declares delete()
- `t → test/orm/many2many/model/Bookmark.cfc` — 1 finding(s), 96 candidate(s):
  - low `test/orm/many2many/model/Bookmark.cfc` — declares getId()
  - low `test/orm/many2many/model/Tag.cfc` — declares getId()
  - low `test/orm/many2many/model/module.cfc` — declares getId()
  - low `test/orm/many2many/model/moduleLang.cfc` — declares getId()
  - low `test/orm/clearSession/Code.cfc` — declares getId()
  - low `test/orm/events/Code.cfc` — declares getId()
  - low `test/orm/many2one/Code.cfc` — declares getId()
  - low `test/orm/many2one/Ref.cfc` — declares getId()
  - … 88 more
- `t2 → test/tickets/LDEV0078/Test.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `test → test/tickets/LDEV2390/Test.cfc` — 1 finding(s), 10 candidate(s):
  - medium `test/tickets/LDEV2390/Test.cfc` — declares foo(); named like the receiver 'test'
  - low `test/tickets/LDEV3714.cfc` — declares foo()
  - low `test/tickets/LDEV0240/comp2.cfc` — declares foo()
  - low `test/tickets/LDEV0600/comp1.cfc` — declares foo()
  - low `test/tickets/LDEV0600/comp2.cfc` — declares foo()
  - low `test/tickets/LDEV1059/content.cfc` — declares foo()
  - low `test/tickets/LDEV1221/static.cfc` — declares foo()
  - low `test/tickets/LDEV3184/LDEV3184.cfc` — declares foo()
  - … 2 more
- `testquery → test/tickets/LDEV3963/test3963quoted.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
- `testquery2 → test/tickets/LDEV3963/test3963quoted.cfc` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV3963/test3963quoted.cfc` — declares get()
  - low `test/tickets/LDEV3963/test3963unquoted.cfc` — declares get()
  - low `test/functions/serializeJSON/_Json.cfc` — declares get()
  - low `test/functions/Duplicate/comps/some/Hello.cfc` — declares get()
- `variables.a.b.c → test/general/InlineComponent.cfc` — 1 finding(s), 2 candidate(s):
  - low `test/general/InlineComponent.cfc` — declares d()
  - low `test/general/subComponent/TestSubScript.cfc` — declares d()

</details>

## Return types — a call chained on a method that declares no component

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 10 | `method 'exe' has no component return type` | 0 | none |  | `test/tickets/LDEV0907.cfc:34` method 'exe' has no component return type (chain to 'isCached') |
| 2 | `method '_getAntiSamy' has no component return type` | 0 | none |  | `test/tickets/LDEV5085/AntiSamyService.cfc:19` method '_getAntiSamy' has no component return type (chain to 'scan') |
| 2 | `method 'func' has no component return type → test/general/InlineComponent.cfc` | 6 | low | declares b(), c() (3 candidates) | `test/general/InlineComponent.cfc:65` method 'func' has no component return type (chain to 'b') |
| 2 | `method 'getResult' in Result has no component return type` | 0 | none |  | `test/tickets/LDEV4555.cfc:24` method 'getResult' in Result has no component return type (chain to 'recordCount') |
| 2 | `method 'send' in http has no component return type → core/src/main/java/resource/component/org/lucee/cfml/Result.cfc` | 1 | medium | declares getPrefix() | `test/tickets/_LDEV1454.cfc:21` method 'send' in http has no component return type (chain to 'getPrefix') |
| 2 | `method 'test' has no component return type` | 0 | none |  | `test/tickets/LDEV4756.cfc:137` method 'test' has no component return type (chain to 'trim') |
| 1 | `method 'buildApp' has no component return type → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV6084.cfc:23` method 'buildApp' has no component return type (chain to 'getName') |
| 1 | `method 'buildApp' has no component return type → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV6084.cfc:23` method 'buildApp' has no component return type (chain to 'getClass') |
| 1 | `method 'getBundles' in testbox.system.TestBox has no component return type` | 0 | none |  | `test/_testRunner.cfc:80` method 'getBundles' in testbox.system.TestBox has no component return type (chain to 'len') |
| 1 | `method 'getFruits' has no component return type` | 0 | none |  | `test/tickets/LDEV0423/BasketEntity.cfc:15` method 'getFruits' has no component return type (chain to 'each') |
| 1 | `method 'getId' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV801.cfc:98` method 'getId' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type (chain to 'getName') |
| 1 | `method 'getId' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV801.cfc:98` method 'getId' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type (chain to 'getClass') |
| 1 | `method 'getLuceeRocks' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV0078/Test.cfc` | 67 | low | declares getName() (67 candidates) | `test/tickets/LDEV801.cfc:101` method 'getLuceeRocks' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type (chain to 'getName') |
| 1 | `method 'getLuceeRocks' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV801.cfc:101` method 'getLuceeRocks' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type (chain to 'getClass') |
| 1 | `method 'getName' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type` | 67 | none |  | `test/tickets/LDEV801.cfc:95` method 'getName' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type (chain to 'getName') |
| 1 | `method 'getName' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` | 26 | low | declares getClass() (26 candidates) | `test/tickets/LDEV801.cfc:95` method 'getName' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type (chain to 'getClass') |
| 1 | `method 'send' in Http has no component return type → core/src/main/java/resource/component/org/lucee/cfml/Result.cfc` | 1 | medium | declares getPrefix() | `test/tickets/_LDEV1976.cfc:6` method 'send' in Http has no component return type (chain to 'getPrefix') |

<details><summary>Groups with several candidates</summary>

- `method 'func' has no component return type → test/general/InlineComponent.cfc` — 2 finding(s), 3 candidate(s):
  - low `test/general/InlineComponent.cfc` — declares b(), c()
  - low `test/general/subComponent/TestSubScript.cfc` — declares b(), c()
  - low `test/jira/Jira2726.cfc` — declares b(), c()
- `method 'buildApp' has no component return type → test/tickets/LDEV0078/Test.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `method 'buildApp' has no component return type → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `method 'getId' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV0078/Test.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `method 'getId' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `method 'getLuceeRocks' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV0078/Test.cfc` — 1 finding(s), 67 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Basket.cfc` — declares getName()
  - low `test/tickets/LDEV0423/BasketEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0423/Fruit.cfc` — declares getName()
  - low `test/tickets/LDEV0423/FruitEntity.cfc` — declares getName()
  - low `test/tickets/LDEV0921/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1214/RandomEntity.cfc` — declares getName()
  - low `test/tickets/LDEV1569/test.cfc` — declares getName()
  - … 59 more
- `method 'getLuceeRocks' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more
- `method 'getName' in LDEV801.ComponentWithSimpleDefaultProperties has no component return type → test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — 1 finding(s), 26 candidate(s):
  - low `test/tickets/LDEV5627/ClassPropertyAccessor.cfc` — declares getClass()
  - low `test/tickets/LDEV5627/CustomGetClassMethod.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Claude.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/Gemini.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/aidriver/OpenAI.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/MemCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/cdriver/RamCache.cfc` — declares getClass()
  - low `core/src/main/java/resource/context/admin/dbdriver/Driver.cfc` — declares getClass()
  - … 18 more

</details>

## Method definitions — a method not found where it was looked for

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 33 | `writeln` | 0 | none |  | `test/tickets/LDEV6377_regression.cfm:5` no qualifier, not in file |
| 9 | `mongodbconnect` | 0 | none |  | `test/_setupTestServices.cfc:320` no qualifier, not in file |
| 7 | `getname` | 67 | low | declares getName (67 candidates) | `test/functions/QueryExecute.cfc:51` method 'getName' not found on builtin queryexecute |
| 7 | `tostring` | 3 | none |  | `core/src/main/cfml/context/admin/login.new.cfm:6` method 'getConfigServerDir' not found on builtin native.lucee.config (chain to 'toString') |
| 6 | `getcolumncount` | 0 | none |  | `test/functions/QueryAddRow.cfc:11` method 'getColumnCount' not found on builtin querynew |
| 6 | `mongodbid` | 0 | none |  | `test/a_debug_build/_MongoDB.cfc:125` not found in extends chain |
| 5 | `typeof` | 1 | none |  | `test/tickets/LDEV6377_regression.cfm:17` no qualifier, not in file |
| 4 | `action` | 1 | medium | declares action | `core/src/main/cfml/context/admin/plugin/DDNS/overview.cfm:4` no qualifier, not in file |
| 4 | `getdata` | 13 | low | declares getData; beside the calling file (13 candidates) | `test/tickets/LDEV1221/test1.cfm:6` no qualifier, not in file |
| 4 | `getdialect` | 0 | none |  | `test/orm/testDialects/testH2/index.cfm:2` method 'getDialect' not found on builtin native.sessionfactory |
| 4 | `getred` | 3 | low | declares getRed (3 candidates) | `test/tickets/LDEV0554.cfc:27` not found in extends chain |
| 4 | `luceeaihas` | 0 | none |  | `core/src/main/cfml/context/debug/modern/error.cfm:5` no qualifier, not in file |
| 4 | `round` | 0 | none |  | `test/tickets/LDEV1824.cfc:32` method 'round' not found on builtin native.numeric |
| 4 | `tojson` | 0 | none |  | `test/_testFilter.cfc:108` no qualifier, not in file |
| 3 | `cf_dollar` | 0 | none |  | `test/tickets/LDEV3689.cfc:68` not found in extends chain |
| 3 | `clone` | 1 | medium | declares clone | `test/tickets/LDEV5353.cfc:17` method 'clone' not found on builtin native.array |
| 3 | `getcolumnlist` | 0 | none |  | `test/tickets/LDEV1160.cfc:10` method 'getColumnList' not found on builtin querynew |
| 3 | `getresourceproviders` | 0 | none |  | `test/general/Resources.cfc:695` method 'getResourceProviders' not found on builtin native.lucee.config |
| 3 | `getrestmappings` | 1 | medium | declares getRestMappings | `test/tickets/LDEV4835.cfc:14` method 'getRestMappings' not found on builtin native.lucee.config |
| 3 | `isempty` | 1 | none |  | `test/functions/QueryIsEmpty.cfc:13` method 'isEmpty' not found on builtin querynew |
| 3 | `iterator` | 0 | none |  | `test/tickets/LDEV0866.cfc:34` method 'keySet' not found on builtin native.struct (chain to 'iterator') |
| 3 | `keyexists` | 0 | none |  | `test/functions/QueryKeyExists.cfc:34` method 'keyExists' not found on builtin querynew |
| 3 | `luceeaigetmetadata` | 0 | none |  | `core/src/main/cfml/context/debug/modern/error.cfm:44` no qualifier, not in file |
| 2 | `a` | 5 | low | declares a; beside the calling file (5 candidates) | `test/general/SafeNavigator.cfc:6` not found in extends chain |
| 2 | `cleandatasourceconnectionpools` | 0 | none |  | `test/tickets/LDEV5967.cfc:93` method 'cleanDatasourceConnectionPools' not found on builtin native.lucee.config |
| 2 | `duplicate` | 0 | none |  | `test/tickets/LDEV1295.cfc:13` method 'Duplicate' not found on builtin querynew |
| 2 | `getcgiscopereadonly` | 0 | none |  | `test/tickets/LDEV0372.cfc:40` method 'getCGIScopeReadOnly' not found on builtin native.lucee.config |
| 2 | `getcomponentmappings` | 1 | medium | declares getComponentMappings | `core/src/main/cfml/context/doc/Application.cfc:103` method 'getComponentMappings' not found on builtin native.lucee.config |
| 2 | `getconfigserverdir` | 0 | none |  | `core/src/main/cfml/context/admin/login.new.cfm:6` method 'getConfigServerDir' not found on builtin native.lucee.config |
| 2 | `getfullnullsupport` | 0 | none |  | `test/jira/Jira2384.cfc:91` method 'getFullNullSupport' not found on builtin native.lucee.config |
| 2 | `getmessage` | 1 | medium | declares getMessage | `test/tickets/LDEV1962.cfc:28` method 'getMessage' not found in LDEV1962.component2 |
| 2 | `getmessage2` | 1 | medium | declares getMessage2 | `test/tickets/LDEV1962.cfc:33` method 'getMessage2' not found in LDEV1962.component2 |
| 2 | `getmode` | 0 | none |  | `test/functions/FileOpen.cfc:10` method 'getMode' not found on builtin fileopen |
| 2 | `getresource` | 0 | none |  | `test/functions/DirectoryDelete.cfc:77` method 'getResource' not found on builtin fileopen |
| 2 | `getsecuritydirectory` | 0 | none |  | `test/functions/SSLCertificateInstall.cfc:14` method 'getSecurityDirectory' not found on builtin native.lucee.config |
| 2 | `getsize` | 0 | none |  | `test/functions/FileOpen.cfc:57` method 'getSize' not found on builtin fileopen |
| 2 | `getsql` | 0 | none |  | `test/tickets/LDEV3863.cfc:12` method 'getSql' not found on builtin queryexecute |
| 2 | `getstatus` | 1 | medium | declares getStatus | `test/functions/FileOpen.cfc:11` method 'getStatus' not found on builtin fileopen |
| 2 | `hascfsession` | 0 | none |  | `test/tickets/LDEV5292/cfml-session/testThreadCookies.cfm:20` method 'hasCFSession' not found on builtin native.lucee.pagecontext |
| 2 | `len` | 0 | none |  | `test/functions/TrimWhitespace.cfc:11` method 'trimWhitespace' not found on builtin native.string (chain to 'len') |
| 2 | `listitemtrim` | 0 | none |  | `test/functions/listitemtrim.cfc:12` method 'listItemTrim' not found on builtin native.string |
| 2 | `luceeinquiryaisession` | 0 | none |  | `core/src/main/cfml/context/debug/modern/error.cfm:55` no qualifier, not in file |
| 2 | `mixin` | 0 | none |  | `test/tickets/LDEV3473/test.cfm:7` method 'mixin' not found in User |
| 2 | `notexisting` | 0 | none |  | `test/tickets/LDEV1201.cfc:70` not found in extends chain |
| 2 | `setdate` | 0 | none |  | `test/functions/SetDay.cfc:43` method 'setDate' not found on builtin native.date |
| 2 | `sgn` | 0 | none |  | `test/functions/sgn.cfc:9` method 'sgn' not found on builtin native.numeric |
| 2 | `sin` | 0 | none |  | `test/functions/sin.cfc:30` method 'sin' not found on builtin native.numeric |
| 2 | `sp_xmltest_bug` | 0 | none |  | `test/datasource/Oracle.cfc:122` not found in extends chain |
| 2 | `sqr` | 0 | none |  | `test/functions/sqr.cfc:7` method 'sqr' not found on builtin native.numeric |
| 2 | `testfunctioninsidetagisland` | 0 | none |  | `test/tickets/LDEV3740.cfc:14` not found in extends chain |
| 2 | `to_char` | 0 | none |  | `test/datasource/Oracle.cfc:127` not found in extends chain |
| 2 | `tokey` | 0 | none |  | `test/tickets/LDEV3167.cfc:31` method 'tokey' not found on builtin native.array |
| 2 | `valueequals` | 4 | low | declares valueEquals; beside the calling file (4 candidates) | `test/functions/DayOfWeekAsString.cfc:35` not found in extends chain |
| 2 | `valueof` | 0 | none |  | `test/general/StaticMembersInvoke.cfc:31` not found in extends chain |
| 2 | `varchar` | 0 | none |  | `test/datasource/MySQL.cfc:75` not found in extends chain |
| 2 | `whodoyoulove` | 1 | medium | declares whoDoYouLove; beside the calling file | `test/tickets/LDEV4642/LDEV4642.cfm:5` no qualifier, not in file |
| 2 | `writebase64` | 0 | none |  | `test/functions/imageReadbase64.cfc:22` method 'writeBase64' not found on builtin imageread |
| 2 | `xmlelement` | 0 | none |  | `test/datasource/Oracle.cfc:127` not found in extends chain |
| 1 | `abc` | 1 | medium | declares abc | `test/tickets/LDEV1487.cfc:11` not found in extends chain |
| 1 | `abs` | 8 | low | declares abs (8 candidates) | `test/tickets/LDEV1824.cfc:10` method 'abs' not found on builtin native.numeric |
| 1 | `accessimportinapp1` | 1 | medium | declares accessImportInApp1 | `test/tickets/LDEV3912.cfc:5` not found in extends chain |
| 1 | `acos` | 0 | none |  | `test/tickets/LDEV1824.cfc:11` method 'aCos' not found on builtin native.numeric |
| 1 | `arraynew[]` | 0 | none |  | `test/tickets/LDEV4370/LDEV4370.cfm:9` no qualifier, not in file |
| 1 | `asin` | 0 | none |  | `test/tickets/LDEV1824.cfc:12` method 'asin' not found on builtin native.numeric |
| 1 | `atn` | 0 | none |  | `test/tickets/LDEV1824.cfc:13` method 'atn' not found on builtin native.numeric |
| 1 | `authority` | 0 | none |  | `test/tickets/LDEV6084.cfc:17` not found in extends chain |
| 1 | `bitand` | 0 | none |  | `test/tickets/LDEV1824.cfc:14` method 'bitAnd' not found on builtin native.numeric |
| 1 | `bitmaskclear` | 0 | none |  | `test/tickets/LDEV1824.cfc:15` method 'bitMaskClear' not found on builtin native.numeric |
| 1 | `bitmaskread` | 0 | none |  | `test/tickets/LDEV1824.cfc:16` method 'bitMaskRead' not found on builtin native.numeric |
| 1 | `bitmaskset` | 0 | none |  | `test/tickets/LDEV1824.cfc:17` method 'bitMaskset' not found on builtin native.numeric |
| 1 | `bitnot` | 0 | none |  | `test/tickets/LDEV1824.cfc:18` method 'bitNot' not found on builtin native.numeric |
| 1 | `bitor` | 0 | none |  | `test/tickets/LDEV1824.cfc:19` method 'bitOR' not found on builtin native.numeric |
| 1 | `bitshln` | 0 | none |  | `test/tickets/LDEV1824.cfc:20` method 'bitSHLN' not found on builtin native.numeric |
| 1 | `bitshrn` | 0 | none |  | `test/tickets/LDEV1824.cfc:37` method 'BitSHRN' not found on builtin native.numeric |
| 1 | `bitxor` | 0 | none |  | `test/tickets/LDEV1824.cfc:21` method 'bitXor' not found on builtin native.numeric |
| 1 | `booleanformat` | 0 | none |  | `test/tickets/LDEV4589/LDEV4589.cfm:6` method 'booleanFormat' not found on builtin native.numeric |
| 1 | `bytagreference` | 0 | none |  | `test/tickets/LDEV3668.cfc:8` not found in extends chain |
| 1 | `bytagvalue` | 0 | none |  | `test/tickets/LDEV3668.cfc:14` not found in extends chain |
| 1 | `cast` | 0 | none |  | `test/tickets/LDEV4827.cfc:29` not found in extends chain |
| 1 | `ceiling` | 0 | none |  | `test/tickets/LDEV1824.cfc:22` method 'ceiling' not found on builtin native.numeric |
| 1 | `cf_anothertag` | 0 | none |  | `test/tickets/LDEV5836/test-customtags.cfm:17` no qualifier, not in file |
| 1 | `cf_mytag` | 0 | none |  | `test/tickets/LDEV3722.cfc:8` not found in extends chain |
| 1 | `cf_redden` | 0 | none |  | `test/tickets/LDEV1276/test.cfm:6` no qualifier, not in file |
| 1 | `clearrect` | 0 | none |  | `test/functions/imageClearRect.cfc:22` method 'ClearRect' not found on builtin imageread |
| 1 | `closure` | 0 | none |  | `test/jira/Jira2902/AbsAbs.cfc:25` no qualifier, not in file |
| 1 | `container` | 4 | low | declares container; beside the calling file (4 candidates) | `test/tickets/LDEV0285/App4.cfc:6` no qualifier, not in file |
| 1 | `copy` | 0 | none |  | `test/functions/imageCopy.cfc:22` method 'copy' not found on builtin imagenew |
| 1 | `cos` | 0 | none |  | `test/tickets/LDEV1824.cfc:23` method 'cos' not found on builtin native.numeric |
| 1 | `createid` | 1 | medium | declares createId | `core/src/main/cfml/context/admin/ext.applications.upload.cfm:159` no qualifier, not in file |
| 1 | `current` | 0 | none |  | `test/tickets/LDEV5632.cfc:50` not found in extends chain |
| 1 | `currentrow` | 0 | none |  | `test/functions/QueryCurrentRow.cfc:11` method 'currentRow' not found on builtin querynew |
| 1 | `drawbeveledrect` | 0 | none |  | `test/functions/ImageDrawBeveledRect.cfc:22` method 'DrawBeveledRect' not found on builtin imagenew |
| 1 | `elemnew` | 0 | none |  | `test/tickets/LDEV2936.cfc:15` method 'elemNew' not found on builtin xmlnew |
| 1 | `emoji_test` | 0 | none |  | `test/datasource/MySQL.cfc:73` not found in extends chain |
| 1 | `entryset` | 0 | none |  | `test/tickets/LDEV0866.cfc:44` method 'entrySet' not found on builtin native.struct |
| 1 | `firstdayofmonth` | 0 | none |  | `test/functions/FirstDayOfMonth.cfc:15` method 'firstDayOfMonth' not found on builtin native.date |
| 1 | `fix` | 0 | none |  | `test/tickets/LDEV1824.cfc:25` method 'fix' not found on builtin native.numeric |
| 1 | `floor` | 0 | none |  | `test/tickets/LDEV1824.cfc:26` method 'floor' not found on builtin native.numeric |
| 1 | `foo` | 12 | low | declares foo; beside the calling file (12 candidates) | `test/tickets/LDEV0441.cfc:16` not found in extends chain |
| 1 | `formatbasen` | 0 | none |  | `test/tickets/LDEV1824.cfc:27` method 'formatBaseN' not found on builtin native.numeric |
| 1 | `getadminmode` | 0 | none |  | `test/components/Administrator.cfc:1681` method 'getAdminMode' not found on builtin native.lucee.config |
| 1 | `getbase64string` | 0 | none |  | `test/tickets/LDEV5640.cfc:16` method 'getBase64String' not found on builtin imagenew |
| 1 | `getcfid` | 0 | none |  | `test/general/ScopeStorage/call.cfm:7` method 'getCFID' not found on builtin native.lucee.pagecontext |
| 1 | `getclassdirectory` | 0 | none |  | `test/general/BinaryTemplates.cfc:6` method 'getClassDirectory' not found on builtin native.lucee.config |
| 1 | `getcolumnlabels` | 0 | none |  | `test/tickets/LDEV1289/test.cfm:4` method 'getMeta' not found on builtin querynew (chain to 'getcolumnlabels') |
| 1 | `getconfigserverimpl` | 0 | none |  | `test/tickets/LDEV5185.cfc:34` method 'getConfigServerImpl' not found on builtin native.lucee.config |
| 1 | `getcustomtagmappings` | 1 | medium | declares getCustomTagMappings | `test/general/Mappings.cfc:79` method 'getCustomTagMappings' not found on builtin native.lucee.config |
| 1 | `getdatasourcename` | 0 | none |  | `test/tickets/LDEV3070/LDEV3070.cfm:13` method 'getDatasourceName' not found on builtin queryexecute |
| 1 | `getimagebytes` | 0 | none |  | `test/tickets/LDEV1576/test.cfm:4` method 'getImageBytes' not found on builtin imagenew |
| 1 | `getjmxname` | 0 | none |  | `test/tickets/LDEV5967.cfc:148` method 'getJmxName' not found on builtin native.pool |
| 1 | `getlistener` | 0 | none |  | `core/src/main/cfml/context/gateway/AsynchronousEvents.cfc:66` no qualifier, not in file |
| 1 | `getmaxtotal` | 0 | none |  | `test/tickets/LDEV6051.cfc:98` method 'getMaxTotal' not found on builtin native.pool |
| 1 | `getmessage2x` | 0 | none |  | `test/tickets/LDEV1962/component2.cfc:17` no qualifier, not in file |
| 1 | `getmessagex` | 0 | none |  | `test/tickets/LDEV1962/component2.cfc:9` no qualifier, not in file |
| 1 | `getmeta` | 0 | none |  | `test/tickets/LDEV1289/test.cfm:4` method 'getMeta' not found on builtin querynew |
| 1 | `getmetadata` | 0 | none |  | `test/functions/FileOpen.cfc:45` method 'getMetaData' not found on builtin fileopen |
| 1 | `getproviderdata` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.list.cfm:79` no qualifier, not in file |
| 1 | `getresourceasstream` | 0 | none |  | `test/tickets/LDEV6171.cfc:13` method 'getResourceAsStream' not found on builtin native.class |
| 1 | `getresult` | 2 | medium | declares getResult | `test/tickets/LDEV2616/LDEV2616.cfm:6` method 'getresult' not found on builtin queryexecute |
| 1 | `getserverrhextensions` | 0 | none |  | `test/tickets/LDEV2558.cfc:14` method 'getServerRHExtensions' not found on builtin native.lucee.config |
| 1 | `getservlet` | 0 | none |  | `core/src/main/cfml/context/admin/overview.cfm:56` method 'getServlet' not found on builtin native.lucee.factory |
| 1 | `getstr` | 1 | medium | declares getStr; beside the calling file | `test/tickets/LDEV4156.cfc:15` method 'getStr' not found in Component |
| 1 | `insertat` | 0 | none |  | `test/tickets/LDEV1844.cfc:6` method 'insertAt' not found on builtin native.boolean |
| 1 | `invokestatic` | 0 | none |  | `test/tickets/_LDEV1706.cfc:23` not found in extends chain |
| 1 | `isstring` | 0 | none |  | `test/tickets/LDEV6377_regression.cfm:82` no qualifier, not in file |
| 1 | `keyset` | 0 | none |  | `test/tickets/LDEV0866.cfc:34` method 'keySet' not found on builtin native.struct |
| 1 | `listener[]` | 0 | none |  | `core/src/main/cfml/context/gateway/MailWatcher.cfc:56` no qualifier, not in file |
| 1 | `ljklkju` | 0 | none |  | `test/general/Elvis.cfc:60` not found in extends chain |
| 1 | `loadprovidersdata` | 0 | none |  | `core/src/main/cfml/context/admin/ext.applications.upload.cfm:140` no qualifier, not in file |
| 1 | `log` | 1 | medium | declares log | `test/tickets/LDEV1824.cfc:28` method 'log' not found on builtin native.numeric |
| 1 | `log10` | 0 | none |  | `test/tickets/LDEV1824.cfc:29` method 'log10' not found on builtin native.numeric |
| 1 | `max` | 0 | none |  | `test/tickets/LDEV1824.cfc:30` method 'max' not found on builtin native.numeric |
| 1 | `min` | 0 | none |  | `test/tickets/LDEV1824.cfc:31` method 'min' not found on builtin native.numeric |
| 1 | `myfunc` | 1 | medium | declares myFunc | `test/tickets/LDEV1995/test.cfm:1` no qualifier, not in file |
| 1 | `myfunc2` | 1 | medium | declares myFunc2 | `test/tickets/LDEV1995/test.cfm:1` no qualifier, not in file |
| 1 | `nonexistentmethod` | 0 | none |  | `test/tickets/LDEV6055/Child.cfc:4` not found in parent component |
| 1 | `nonexistentmethod_ldev6349` | 0 | none |  | `test/tickets/LDEV6349.cfc:41` method 'nonExistentMethod_ldev6349' not found on builtin native.date |
| 1 | `oncomplete` | 0 | none |  | `test/tickets/LDEV2213/test.cfm:5` no qualifier, not in file |
| 1 | `parsedatetime` | 0 | none |  | `test/tickets/LDEV0028.cfc:13` method 'parseDateTime' not found on builtin native.string |
| 1 | `part` | 0 | none |  | `test/functions/DatePart.cfc:13` method 'part' not found on builtin native.date |
| 1 | `push` | 0 | none |  | `test/functions/ArrayPush.cfc:11` method 'push' not found on builtin native.array |
| 1 | `renamecolumn` | 0 | none |  | `test/functions/QueryRenameColumn.cfc:31` method 'renameColumn' not found on builtin querynew |
| 1 | `reportservicefailed` | 1 | medium | declares reportServiceFailed; beside the calling file | `test/run-tests.cfm:617` no qualifier, not in file |
| 1 | `resetall` | 0 | none |  | `test/tickets/LDEV5185.cfc:34` method 'getConfigServerImpl' not found on builtin native.lucee.config (chain to 'resetAll') |
| 1 | `resize` | 0 | none |  | `test/tickets/LDEV1844.cfc:10` method 'Resize' not found on builtin native.boolean |
| 1 | `returnsany` | 2 | low | declares returnsAny (2 candidates) | `test/tickets/_LDEV1835.cfc:6` method 'returnsany' not found in LDEV1835.Comp |
| 1 | `rluceemonoblock` | 0 | none |  | `core/src/main/cfml/context/templates/error/error.cfm:252` no qualifier, not in file |
| 1 | `rotatedrawingaxis` | 0 | none |  | `test/functions/ImageRotateDrawingAxis.cfc:23` method 'rotateDrawingAxis' not found on builtin imagenew |
| 1 | `set` | 1 | none |  | `test/tickets/LDEV1844.cfc:10` method 'Resize' not found on builtin native.boolean (chain to 'set') |
| 1 | `setlastborrowed` | 0 | none |  | `test/tickets/LDEV5967.cfc:170` method 'setLastBorrowed' not found on builtin native.pool |
| 1 | `setminevictableidletimemillis` | 0 | none |  | `test/tickets/LDEV5963.cfc:69` method 'setMinEvictableIdleTimeMillis' not found on builtin native.pool |
| 1 | `setminutes` | 0 | none |  | `test/functions/SetMinute.cfc:41` method 'setMinutes' not found on builtin native.date |
| 1 | `setseconds` | 0 | none |  | `test/functions/SetSecond.cfc:41` method 'setSeconds' not found on builtin native.date |
| 1 | `setyear` | 0 | none |  | `test/tickets/LDEV1041.cfc:41` not found in extends chain |
| 1 | `shear` | 0 | none |  | `test/tickets/LDEV4459.cfc:7` method 'Shear' not found on builtin imagenew |
| 1 | `sheardrawingaxis` | 0 | none |  | `test/functions/ImageShearDrawingAxis.cfc:23` method 'shearDrawingAxis' not found on builtin imagenew |
| 1 | `sleeo` | 0 | none |  | `test/tickets/LDEV4967.cfc:11` not found in extends chain |
| 1 | `slice` | 0 | none |  | `test/functions/QuerySlice.cfc:12` method 'slice' not found on builtin querynew |
| 1 | `ss[]` | 0 | none |  | `test/functions/GetComponentStaticScope.cfc:31` not found in extends chain |
| 1 | `susi` | 2 | low | declares susi (2 candidates) | `test/tickets/LDEV4826.cfc:54` not found in extends chain |
| 1 | `tan` | 0 | none |  | `test/tickets/LDEV1824.cfc:36` method 'tan' not found on builtin native.numeric |
| 1 | `testtagattr` | 0 | none |  | `test/tickets/LDEV4008.cfc:7` not found in extends chain |
| 1 | `this[]` | 0 | none |  | `core/src/main/cfml/context/admin/plugin/Plugin.cfc:74` no qualifier, not in file |
| 1 | `tobase64` | 0 | none |  | `test/functions/toBase64.cfc:7` method 'toBase64' not found on builtin native.string |
| 1 | `tolist` | 0 | none |  | `test/functions/functionHasTests.cfc:20` method 'toList' not found on builtin native.boolean |
| 1 | `tostruct` | 1 | medium | declares toStruct; beside the calling file | `test/tickets/LDEV5327.cfc:44` method 'ToStruct' not found on builtin queryexecute |
| 1 | `trimwhitespace` | 0 | none |  | `test/functions/TrimWhitespace.cfc:11` method 'trimWhitespace' not found on builtin native.string |
| 1 | `unshift` | 0 | none |  | `test/functions/ArrayUnShift.cfc:11` method 'unshift' not found on builtin native.array |
| 1 | `values` | 0 | none |  | `test/tickets/LDEV0866.cfc:55` method 'values' not found on builtin native.struct |
| 1 | `xmlserialize` | 0 | none |  | `test/datasource/Oracle.cfc:127` not found in extends chain |

<details><summary>Groups with several candidates</summary>

- `getname` — 7 finding(s), 67 candidate(s):
  - low `test/functions/ORMExecuteQuery/Person.cfc:4` — declares getName
  - low `test/jira/Jira0713/Test.cfc:20` — declares getName
  - low `test/orm/transRollback/Person.cfc:3` — declares getName
  - low `test/orm/transSave/Person.cfc:3` — declares getName
  - low `test/orm/transSaveExCommit/Person.cfc:3` — declares getName
  - low `test/orm/transSaveFlush/Person.cfc:3` — declares getName
  - low `test/orm/transSavepoint/Person.cfc:3` — declares getName
  - low `test/tickets/LDEV0078/Test.cfc:4` — declares getName
  - … 59 more
- `getdata` — 4 finding(s), 13 candidate(s):
  - low `test/tickets/LDEV1221/static.cfc:10` — declares getData; beside the calling file
  - low `test/tickets/LDEV0835/A.cfc:7` — declares getData
  - low `test/tickets/LDEV0835/B.cfc:6` — declares getData
  - low `test/tickets/LDEV0835/Base.cfc:7` — declares getData
  - low `test/tickets/LDEV5792/ldev5792_nested_tag.cfm:5` — declares getData
  - low `core/src/main/cfml/context/admin/adminfunctions.cfc:55` — declares getdata
  - low `core/src/main/java/resource/context/admin/aidriver/Field.cfc:51` — declares getData
  - low `core/src/main/java/resource/context/admin/cdriver/Field.cfc:51` — declares getData
  - … 5 more
- `getred` — 4 finding(s), 3 candidate(s):
  - low `test/tickets/LDEV0554/Color.cfc:33` — declares getRed
  - low `test/tickets/LDEV0554/Color3.cfc:31` — declares getRed
  - low `test/tickets/LDEV0554/Color4.cfc:28` — declares getRed
- `a` — 2 finding(s), 5 candidate(s):
  - low `test/general/InlineComponent.cfc:29` — declares a; beside the calling file
  - low `test/general/subComponent/TestSubScript.cfc:6` — declares a
  - low `test/functions/_SerializeJSON2.cfc:220` — declares a
  - low `test/jira/Jira1460.cfc:83` — declares a
  - low `test/jira/Jira2726.cfc:34` — declares a
- `valueequals` — 2 finding(s), 4 candidate(s):
  - low `test/functions/DateConvert.cfc:36` — declares valueEquals; beside the calling file
  - low `test/functions/Decrypt.cfc:34` — declares valueEquals; beside the calling file
  - low `test/functions/DeleteClientVariable.cfc:44` — declares valueEquals; beside the calling file
  - low `test/functions/EncodeForCSS.cfc:35` — declares valueEquals; beside the calling file
- `abs` — 1 finding(s), 8 candidate(s):
  - low `test/general/abstract/Abs.cfc:2` — declares abs
  - low `test/general/abstract/AbsAltSyntax1.cfc:2` — declares abs
  - low `test/general/abstract/AbsAltSyntax2.cfc:2` — declares abs
  - low `test/general/abstract/Final.cfc:3` — declares abs
  - low `test/general/abstract/FinalAltSyntax1.cfc:3` — declares abs
  - low `test/general/abstract/FinalAltSyntax2.cfc:3` — declares abs
  - low `test/general/modifiers/Abs.cfc:2` — declares abs
  - low `test/general/modifiers/Final.cfc:3` — declares abs
- `container` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV0285/App1.cfc:8` — declares container; beside the calling file
  - low `test/tickets/LDEV0285/App2.cfc:8` — declares container; beside the calling file
  - low `test/tickets/LDEV0285/App3.cfc:8` — declares container; beside the calling file
  - low `test/tickets/LDEV0590/test.cfc:6` — declares container
- `foo` — 1 finding(s), 12 candidate(s):
  - low `test/tickets/LDEV3714.cfc:42` — declares foo; beside the calling file
  - low `test/tickets/LDEV0240/comp2.cfc:2` — declares foo
  - low `test/tickets/LDEV0600/comp1.cfc:6` — declares foo
  - low `test/tickets/LDEV0600/comp2.cfc:6` — declares foo
  - low `test/tickets/LDEV1059/content.cfc:3` — declares foo
  - low `test/tickets/LDEV1221/static.cfc:18` — declares foo
  - low `test/tickets/LDEV2390/Test.cfc:6` — declares foo
  - low `test/tickets/LDEV3184/LDEV3184.cfc:3` — declares foo
  - … 4 more
- `returnsany` — 1 finding(s), 2 candidate(s):
  - low `test/tickets/LDEV1812/ReturnsString.cfc:3` — declares returnsAny
  - low `test/tickets/LDEV1835/interface.cfc:2` — declares returnsany
- `susi` — 1 finding(s), 2 candidate(s):
  - low `test/general/Struct/invalid3.cfm:2` — declares susi
  - low `test/jira/Jira3099/Component1.cfc:40` — declares susi

</details>

## Object references — a component path that names no file

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 3 | `base component does not resolve; 2 inherited calls not checked` | ? | none |  | `core/src/main/cfml/context/admin/plugin/DDNS/Action.cfc:17` base component does not resolve; 2 inherited calls not checked |
| 3 | `chained on 'append', which is not found (calling 'map')` | 0 | none |  | `test/tickets/LDEV5578/LDEV5578.cfc:56` chained on 'append', which is not found (calling 'map') |
| 3 | `component 'LDEV0273.Test' does not exist (calling 'exclaim')` | 0 | low | file named Test.cfc (56 candidates) | `test/tickets/LDEV0273.cfc:23` component 'LDEV0273.Test' does not exist (calling 'exclaim') |
| 3 | `component 'LDEV0273.Test' does not exist (calling 'exclaim2')` | 0 | low | file named Test.cfc (56 candidates) | `test/tickets/LDEV0273.cfc:27` component 'LDEV0273.Test' does not exist (calling 'exclaim2') |
| 3 | `component 'System' does not exist (calling 'setProperty')` | 0 | none |  | `core/src/main/cfml/context/debug/modern/reference.cfm:398` component 'System' does not exist (calling 'setProperty') |
| 3 | `component 'collection' does not exist (calling 'list')` | 0 | none |  | `test/tickets/_LDEV1684.cfc:13` component 'collection' does not exist (calling 'list') |
| 2 | `component 'BundleProvider' does not exist (calling 'getMappings')` | 3 | none |  | `test/tickets/LDEV5567/LDEV5660.cfc:9` component 'BundleProvider' does not exist (calling 'getMappings') |
| 2 | `component 'Categories' does not exist (chain hop 'getClass' to 'getName')` | 67 | none |  | `test/tickets/LDEV6250.cfc:27` component 'Categories' does not exist (chain hop 'getClass' to 'getName') |
| 2 | `component 'Version' does not exist (calling 'getVersion')` | 4 | none |  | `test/tickets/LDEV5601.cfc:44` component 'Version' does not exist (calling 'getVersion') |
| 2 | `component 'java.lang.Double' does not exist (calling 'valueOf')` | 0 | none |  | `test/general/StaticMembersInvoke.cfc:23` component 'java.lang.Double' does not exist (calling 'valueOf') |
| 2 | `component 'lucee.runtime.op.Caster' does not exist (calling 'cfTypeToClass')` | 0 | none |  | `test/tickets/LDEV5708.cfc:10` component 'lucee.runtime.op.Caster' does not exist (calling 'cfTypeToClass') |
| 2 | `component 'lucee.runtime.type.StructImpl' does not exist (calling 'size')` | 0 | none |  | `test/general/NewOperator.cfc:37` component 'lucee.runtime.type.StructImpl' does not exist (calling 'size') |
| 1 | `base component does not resolve; 4 inherited calls not checked` | ? | none |  | `core/src/main/java/resource/component/org/lucee/cfml/test/LuceeTestSuiteRunner.cfc:1` base component does not resolve; 4 inherited calls not checked |
| 1 | `chained on 'a', which is not found (calling 'b')` | 6 | none |  | `test/general/SafeNavigator.cfc:11` chained on 'a', which is not found (calling 'b') |
| 1 | `chained on 'a', which is not found (calling 'c')` | 5 | none |  | `test/general/SafeNavigator.cfc:11` chained on 'a', which is not found (calling 'c') |
| 1 | `chained on 'a', which is not found (calling 'd')` | 2 | none |  | `test/general/SafeNavigator.cfc:11` chained on 'a', which is not found (calling 'd') |
| 1 | `chained on 'authority', which is not found (calling 'build')` | 3 | none |  | `test/tickets/LDEV6084.cfc:17` chained on 'authority', which is not found (calling 'build') |
| 1 | `chained on 'getClass', which is not found (calling 'getName')` | 67 | none |  | `test/tickets/LDEV5457.cfc:13` chained on 'getClass', which is not found (calling 'getName') |
| 1 | `chained on 'getListener', which is not found (calling 'onIncomingMessage')` | 1 | none |  | `core/src/main/cfml/context/gateway/AsynchronousEvents.cfc:66` chained on 'getListener', which is not found (calling 'onIncomingMessage') |
| 1 | `chained on 'susi', which is not found (calling 'sorglos')` | 0 | none |  | `test/tickets/LDEV4826.cfc:54` chained on 'susi', which is not found (calling 'sorglos') |
| 1 | `component 'ArchiveGreeter' does not exist (calling 'greet')` | 0 | none |  | `test/general/archives/componentPaths/index.cfm:2` component 'ArchiveGreeter' does not exist (calling 'greet') |
| 1 | `component 'AwsBasicCredentials' does not exist (calling 'create')` | 1 | none |  | `test/tickets/LDEV5457.cfc:13` component 'AwsBasicCredentials' does not exist (calling 'create') |
| 1 | `component 'ClientCredentialFactory' does not exist (calling 'createFromSecret')` | 0 | none |  | `test/tickets/LDEV6084.cfc:15` component 'ClientCredentialFactory' does not exist (calling 'createFromSecret') |
| 1 | `component 'ConfidentialClientApplication' does not exist (calling 'builder')` | 0 | none |  | `test/tickets/LDEV6084.cfc:16` component 'ConfidentialClientApplication' does not exist (calling 'builder') |
| 1 | `component 'FileOutputStream' does not exist (calling 'flush')` | 0 | none |  | `test/tickets/LDEV5810.cfc:49` component 'FileOutputStream' does not exist (calling 'flush') |
| 1 | `component 'HashMap' does not exist (calling 'size')` | 0 | none |  | `test/general/NewOperator.cfc:32` component 'HashMap' does not exist (calling 'size') |
| 1 | `component 'LDEV1706.foo' does not exist (calling 'bar')` | 1 | low | file named foo.cfc (5 candidates) | `test/tickets/_LDEV1706.cfc:27` component 'LDEV1706.foo' does not exist (calling 'bar') |
| 1 | `component 'LDEV1707.foo' does not exist (calling 'bar')` | 1 | low | file named foo.cfc (5 candidates) | `test/tickets/_LDEV1707.cfc:28` component 'LDEV1707.foo' does not exist (calling 'bar') |
| 1 | `component 'LSDateFormat.LSDateFormat' does not exist (calling 'testFunction')` | 9 | medium | file named LSDateFormat.cfc | `test/functions/LSDateFormat.cfc:33` component 'LSDateFormat.LSDateFormat' does not exist (calling 'testFunction') |
| 1 | `component 'LSDateFormat.LSDateFormat' does not exist (calling 'testMemberFunction')` | 16 | medium | file named LSDateFormat.cfc | `test/functions/LSDateFormat.cfc:29` component 'LSDateFormat.LSDateFormat' does not exist (calling 'testMemberFunction') |
| 1 | `component 'MyComponent' does not exist (calling 'someMethod')` | 0 | none |  | `test/tickets/LDEV5836/test-component-methods.cfc:9` component 'MyComponent' does not exist (calling 'someMethod') |
| 1 | `component 'PNGTranscoder' does not exist (calling 'transcode')` | 0 | none |  | `test/tickets/LDEV5810.cfc:48` component 'PNGTranscoder' does not exist (calling 'transcode') |
| 1 | `component 'XWPFDocument' does not exist (calling 'createParagraph')` | 0 | none |  | `test/tickets/LDEV5610.cfc:20` component 'XWPFDocument' does not exist (calling 'createParagraph') |
| 1 | `component 'collection' does not exist (calling 'CREATE')` | 1 | none |  | `test/tickets/_LDEV1684.cfc:12` component 'collection' does not exist (calling 'CREATE') |
| 1 | `component 'dynamiVar' does not exist (calling 'getData')` | 13 | none |  | `test/tickets/LDEV1221/test4.cfm:5` component 'dynamiVar' does not exist (calling 'getData') |
| 1 | `component 'java.lang.StringBuilder' does not exist (chain hop 'append' to 'toString')` | 3 | none |  | `test/general/javaFunctions/toString.cfm:3` component 'java.lang.StringBuilder' does not exist (chain hop 'append' to 'toString') |
| 1 | `component 'java.lang.System' does not exist (calling 'getenv')` | 2 | none |  | `test/tickets/LDEV5181.cfc:11` component 'java.lang.System' does not exist (calling 'getenv') |
| 1 | `component 'java.util.Collections' does not exist (calling 'emptyList')` | 0 | none |  | `test/general/StaticMembersInvoke.cfc:39` component 'java.util.Collections' does not exist (calling 'emptyList') |
| 1 | `component 'java.util.prefs.BackingStoreException' does not exist (calling 'getMessage')` | 1 | none |  | `test/tickets/LDEV5120.cfc:13` component 'java.util.prefs.BackingStoreException' does not exist (calling 'getMessage') |
| 1 | `component 'lucee.runtime.functions.other.GeneratePBKDFKey' does not exist (calling 'getSupportedAlgorithms')` | 0 | medium | file named GeneratePBKDFKey.cfc | `test/tickets/LDEV0256.cfc:10` component 'lucee.runtime.functions.other.GeneratePBKDFKey' does not exist (calling 'getSupportedAlgorithms') |
| 1 | `component 'lucee.runtime.instrumentation.InstrumentationFactory' does not exist (calling 'getInstrumentation')` | 0 | none |  | `test/run-tests.cfm:4` component 'lucee.runtime.instrumentation.InstrumentationFactory' does not exist (calling 'getInstrumentation') |
| 1 | `component 'lucee.runtime.util.AstUtil' does not exist (calling 'astFromString')` | 0 | none |  | `test/functions/astFromString.cfc:15` component 'lucee.runtime.util.AstUtil' does not exist (calling 'astFromString') |
| 1 | `component 'objInst' does not exist (calling 'getData')` | 13 | none |  | `test/tickets/LDEV1221/test6.cfm:9` component 'objInst' does not exist (calling 'getData') |
| 1 | `component 'org.apache.poi.Version' does not exist (calling 'getVersion')` | 4 | none |  | `test/tickets/LDEV5601.cfc:73` component 'org.apache.poi.Version' does not exist (calling 'getVersion') |
| 1 | `component 'org.lucee.test.Test' does not exist (calling 'foo')` | 12 | low | file named Test.cfc (56 candidates) | `test/tickets/LDEV4772.cfc:32` component 'org.lucee.test.Test' does not exist (calling 'foo') |
| 1 | `component 'subComponent.TestSubScript$sub' does not exist (calling 'c')` | 5 | none |  | `test/general/SubComponent.cfc:34` component 'subComponent.TestSubScript$sub' does not exist (calling 'c') |
| 1 | `component 'subComponent.TestSubScript$sub' does not exist (calling 'd')` | 2 | none |  | `test/general/SubComponent.cfc:35` component 'subComponent.TestSubScript$sub' does not exist (calling 'd') |
| 1 | `component 'subComponent.TestSubTag$sub' does not exist (calling 'bb')` | 1 | none |  | `test/general/SubComponent.cfc:23` component 'subComponent.TestSubTag$sub' does not exist (calling 'bb') |
| 1 | `component 'subComponent.TestSubTag$sub' does not exist (calling 'subtest')` | 3 | none |  | `test/general/SubComponent.cfc:22` component 'subComponent.TestSubTag$sub' does not exist (calling 'subtest') |
| 1 | `component 'test.testcases.LDEV1102.test' does not exist (calling 'getuserId')` | 4 | low | file named test.cfc (56 candidates) | `test/tickets/_LDEV1102.cfc:7` component 'test.testcases.LDEV1102.test' does not exist (calling 'getuserId') |
| 1 | `component 'test.testcases.LDEV1102.test1' does not exist (calling 'getuserId')` | 4 | low | file named test1.cfc (4 candidates) | `test/tickets/_LDEV1102.cfc:17` component 'test.testcases.LDEV1102.test1' does not exist (calling 'getuserId') |
| 1 | `component 'test.testcases.LDEV1102.test2' does not exist (calling 'getuserId')` | 4 | low | file named test2.cfc (7 candidates) | `test/tickets/_LDEV1102.cfc:27` component 'test.testcases.LDEV1102.test2' does not exist (calling 'getuserId') |
| 1 | `component 'testComp$testSub' does not exist (calling 'addiFunc')` | 0 | none |  | `test/general/subComponent/index.cfm:14` component 'testComp$testSub' does not exist (calling 'addiFunc') |
| 1 | `component 'testComp$testSub' does not exist (calling 'subFunc')` | 0 | none |  | `test/general/subComponent/index.cfm:8` component 'testComp$testSub' does not exist (calling 'subFunc') |
| 1 | `component 'testbox.system.reports.JUnitReporter' does not exist (calling 'runReport')` | 0 | none |  | `test/run-tests.cfm:444` component 'testbox.system.reports.JUnitReporter' does not exist (calling 'runReport') |

<details><summary>Groups with several candidates</summary>

- `component 'LDEV0273.Test' does not exist (calling 'exclaim')` — 3 finding(s), 56 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0104/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0255/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0280/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0421/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0460/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0486/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0581/Test.cfc` — file named Test.cfc
  - … 48 more
- `component 'LDEV0273.Test' does not exist (calling 'exclaim2')` — 3 finding(s), 56 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0104/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0255/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0280/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0421/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0460/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0486/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0581/Test.cfc` — file named Test.cfc
  - … 48 more
- `component 'LDEV1706.foo' does not exist (calling 'bar')` — 1 finding(s), 5 candidate(s):
  - low `test/tickets/LDEV1659/model/Foo.cfc` — file named foo.cfc
  - low `test/tickets/LDEV3900/otherCfc/foo.cfc` — file named foo.cfc
  - low `test/tickets/LDEV1741/App1/model/Foo.cfc` — file named foo.cfc
  - low `test/tickets/LDEV1741/App2/model/Foo.cfc` — file named foo.cfc
  - low `test/tickets/LDEV1984/App1/model/Foo.cfc` — file named foo.cfc
- `component 'LDEV1707.foo' does not exist (calling 'bar')` — 1 finding(s), 5 candidate(s):
  - low `test/tickets/LDEV1659/model/Foo.cfc` — file named foo.cfc
  - low `test/tickets/LDEV3900/otherCfc/foo.cfc` — file named foo.cfc
  - low `test/tickets/LDEV1741/App1/model/Foo.cfc` — file named foo.cfc
  - low `test/tickets/LDEV1741/App2/model/Foo.cfc` — file named foo.cfc
  - low `test/tickets/LDEV1984/App1/model/Foo.cfc` — file named foo.cfc
- `component 'org.lucee.test.Test' does not exist (calling 'foo')` — 1 finding(s), 56 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0104/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0255/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0280/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0421/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0460/Test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0486/test.cfc` — file named Test.cfc
  - low `test/tickets/LDEV0581/Test.cfc` — file named Test.cfc
  - … 48 more
- `component 'test.testcases.LDEV1102.test' does not exist (calling 'getuserId')` — 1 finding(s), 56 candidate(s):
  - low `test/tickets/LDEV0078/Test.cfc` — file named test.cfc
  - low `test/tickets/LDEV0104/Test.cfc` — file named test.cfc
  - low `test/tickets/LDEV0255/Test.cfc` — file named test.cfc
  - low `test/tickets/LDEV0280/test.cfc` — file named test.cfc
  - low `test/tickets/LDEV0421/test.cfc` — file named test.cfc
  - low `test/tickets/LDEV0460/Test.cfc` — file named test.cfc
  - low `test/tickets/LDEV0486/test.cfc` — file named test.cfc
  - low `test/tickets/LDEV0581/Test.cfc` — file named test.cfc
  - … 48 more
- `component 'test.testcases.LDEV1102.test1' does not exist (calling 'getuserId')` — 1 finding(s), 4 candidate(s):
  - low `test/tickets/LDEV1102/test1.cfc` — file named test1.cfc
  - low `test/tickets/LDEV1152/test1.cfc` — file named test1.cfc
  - low `test/tickets/LDEV1589/test1.cfc` — file named test1.cfc
  - low `test/tickets/LDEV2645/test1.cfc` — file named test1.cfc
- `component 'test.testcases.LDEV1102.test2' does not exist (calling 'getuserId')` — 1 finding(s), 7 candidate(s):
  - low `test/tickets/LDEV1102/test2.cfc` — file named test2.cfc
  - low `test/tickets/LDEV1152/test2.cfc` — file named test2.cfc
  - low `test/tickets/LDEV1202/Test2.cfc` — file named test2.cfc
  - low `test/tickets/LDEV1431/test2.cfc` — file named test2.cfc
  - low `test/tickets/LDEV1589/test2.cfc` — file named test2.cfc
  - low `test/tickets/LDEV2862/test2.cfc` — file named test2.cfc
  - low `test/tickets/LDEV3634/test2.cfc` — file named test2.cfc

</details>
