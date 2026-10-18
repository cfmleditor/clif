# cb-p — unresolved findings and candidates

Every finding below is still reported. A candidate is a liberal match offered for a person to weigh, never an answer the resolver took: **high** is the only component declaring every method the function calls on the receiver (or the only one, named like it); **medium** is a sole match on weaker evidence, or the one named like the receiver among several; **low** is one of several. *Defined* is how many indexed files declare a method of that name: 0 means it is missing from the workspace, more means the resolver could not connect the call to it.

| Category | Findings | high | medium | low | none | method defined nowhere |
|---|---:|---:|---:|---:|---:|---:|
| variable | 1439 | 538 | 169 | 155 | 577 | 253 |
| return-type | 109 | 9 | 38 | 19 | 43 | 29 |
| method | 74 | 1 | 0 | 5 | 68 | 28 |
| object | 61 | 0 | 0 | 0 | 61 | 3 |

## Variable definitions — a receiver whose component is unknown

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 35 | `arguments.content → modules/contentbox/models/content/BaseContent.cfc` | 3 | high | declares hasParent(), getParent(), hasCategories(), removeAllCategories(), hasRelatedContent(), getRelatedContent(), hasLinkedContent(), removeAllLinkedConte… | `modules/contentbox/models/content/ContentService.cfc:417` variable 'arguments.content' has no component ref |
| 30 | `entry` | 7 | none |  | `modules/contentbox/models/system/NotificationService.cfc:164` variable 'entry' has no component ref |
| 26 | `content` | 7 | none |  | `modules/contentbox/models/system/NotificationService.cfc:426` variable 'content' has no component ref |
| 24 | `c` | 1 | none |  | `modules/contentbox/models/content/ContentService.cfc:510` variable 'c' has no component ref |
| 24 | `content → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getCommentSubscriptions() | `modules/contentbox/models/comments/CommentService.cfc:247` variable 'content' has no component ref |
| 24 | `rc.comment → modules/contentbox/models/comments/Comment.cfc` | 3 | high | declares getAuthorEmail(), getAuthor(), getRelatedContent(), getAuthorURL(), getAuthorIP(), getDisplayCreatedDate(), getDisplayContent(); named like the rece… | `modules/contentbox/modules/contentbox-admin/views/comments/moderate.cfm:30` variable 'rc.comment' has no component ref |
| 23 | `args.menuitem` | 1 | none |  | `modules/contentbox/models/menu/views/content/display.cfm:2` variable 'args.menuItem' has no component ref |
| 22 | `arguments.setup → modules/contentbox-installer/models/Setup.cfc` | 1 | high | declares getFullRewrite(), getPopulateData(), getCreateDevSite(); named like the receiver 'setup' | `modules/contentbox-installer/models/InstallerService.cfc:41` variable 'arguments.setup' has no component ref |
| 21 | `incomment → modules/contentbox/models/comments/Comment.cfc` | 7 | high | declares getRelatedContent(), setAuthorIP(), setIsApproved(), getIsApproved(), getMemento() | `modules/contentbox/models/comments/CommentService.cfc:197` variable 'inComment' has no component ref |
| 21 | `response` | 0 | none |  | `modules/contentbox/models/media/ForwardMediaProvider.cfc:34` variable 'response' has no component ref |
| 21 | `setting → modules/contentbox/models/system/Setting.cfc` | 1 | high | declares getSettingID(), getName(), getValue(), getIsCore(), hasSite(), getSite(), getsiteID(); named like the receiver 'setting' | `modules/contentbox/modules/contentbox-admin/views/settings/rawSettingsTable.cfm:25` variable 'setting' has no component ref |
| 19 | `arguments.original → modules/contentbox/models/content/BaseContent.cfc` | 1 | high | declares getHTMLKeywords(), getHTMLDescription(), getHTMLTitle(), getMarkup(), getCache(), getCacheTimeout(), getCacheLastAccessTimeout(), getShowInSearch(),… | `modules/contentbox/models/content/BaseContent.cfc:1324` variable 'arguments.original' has no component ref |
| 19 | `prc.activedisk` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Editor.cfc:29` variable 'prc.activeDisk' has no component ref |
| 19 | `prc.author → modules/contentbox/models/security/Author.cfc` | 2 | high | declares getUsername(), getEmail(), isLoaded(), getAuthorID(); named like the receiver 'author' | `modules/contentbox/modules/contentbox-admin/views/authors/editorHelper.cfm:38` variable 'prc.author' has no component ref |
| 19 | `site → modules/contentbox/models/system/Site.cfc` | 1 | high | declares getIsSSL(); named like the receiver 'site' | `modules/contentbox/models/comments/CommentService.cfc:506` variable 'site' has no component ref |
| 18 | `arguments.site → modules/contentbox/models/system/Site.cfc` | 5 | medium | declares getsiteID(); named like the receiver 'site' (5 candidates) | `modules/contentbox/models/content/BaseContent.cfc:1784` variable 'arguments.site' has no component ref |
| 18 | `thisperm → modules/contentbox/models/security/Permission.cfc` | 1 | high | declares getPermission(), getPermissionID() | `modules/contentbox/modules/contentbox-admin/views/authors/permissions.cfm:151` variable 'thisPerm' has no component ref |
| 17 | `permission → modules/contentbox/models/security/Permission.cfc` | 1 | high | declares getpermissionID(), getPermission(), getDescription(), getNumberOfRoles(), getNumberOfGroups(); named like the receiver 'permission' | `modules/contentbox/modules/contentbox-admin/views/permissions/index.cfm:111` variable 'permission' has no component ref |
| 17 | `prc.cbhelper → modules/contentbox/models/system/CBHelper.cfc` | 1 | high | declares adminRoot(), siteRoot(); named like the receiver 'cbHelper' | `modules/contentbox/modules/contentbox-admin/layouts/admin.cfm:21` variable 'prc.cbHelper' has no component ref |
| 16 | `thiscontent → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares removeCategories() | `modules/contentbox/models/content/CategoryService.cfc:324` variable 'thisContent' has no component ref |
| 15 | `group → modules/contentbox/models/security/PermissionGroup.cfc` | 1 | high | declares getPermissionGroupID(), getName(), getPermissions() | `modules/contentbox/modules/contentbox-admin/views/authors/permissions.cfm:99` variable 'group' has no component ref |
| 15 | `item` | 0 | none |  | `build/patches/5.0.0/Updater.cfc:155` variable 'item' has no component ref |
| 14 | `arguments.page → modules/contentbox/models/content/BaseContent.cfc` | 5 | low | declares getSlug() (5 candidates) | `modules/contentbox/models/content/PageService.cfc:45` variable 'arguments.page' has no component ref |
| 14 | `author → modules/contentbox/models/security/Author.cfc` | 1 | high | declares getFullName(), getRole(), getPermissionGroupsList(), getEmail(), getAuthorID(); named like the receiver 'author' | `modules/contentbox/models/system/NotificationService.cfc:56` variable 'author' has no component ref |
| 14 | `expectation.actual` | 2 | none |  | `tests/resources/BaseApiTest.cfc:51` variable 'expectation.actual' has no component ref |
| 14 | `instance.zipoutput` | 223 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:69` variable 'instance.zipOutput' has no component ref |
| 13 | `arguments.author → modules/contentbox/models/security/Author.cfc` | 3 | high | declares clearPermissions(), clearPermissionGroups(); named like the receiver 'author' | `modules/contentbox/models/security/AuthorService.cfc:169` variable 'arguments.author' has no component ref |
| 13 | `arguments.data.content → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getMarkup() | `modules/contentbox/models/content/renderers/MarkdownRenderer.cfc:26` variable 'arguments.data.content' has no component ref |
| 13 | `comment → modules/contentbox/models/comments/Comment.cfc` | 3 | medium | declares getAuthorEmail(), getRelatedContent(); named like the receiver 'comment' (3 candidates) | `modules/contentbox/models/subscriptions/SubscriptionListener.cfc:35` variable 'comment' has no component ref |
| 13 | `instance.zipfile` | 223 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:195` variable 'instance.zipFile' has no component ref |
| 13 | `role → modules/contentbox/models/security/Role.cfc` | 1 | high | declares getRoleID(), getName(), getRole(), getDescription(), getNumberOfPermissions(), getNumberOfAuthors(); named like the receiver 'role' | `modules/contentbox/modules/contentbox-admin/views/roles/index.cfm:121` variable 'role' has no component ref |
| 12 | `arguments.c` | 0 | none |  | `modules/contentbox/models/content/ContentService.cfc:518` variable 'arguments.c' has no component ref |
| 12 | `ocurrentcontent → modules/contentbox/models/content/BaseContent.cfc` | 1 | high | declares getHTMLTitle(), getTitle() | `modules/contentbox/models/system/CBHelper.cfc:909` variable 'oCurrentContent' has no component ref |
| 12 | `prc.entry → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getComments() | `modules/contentbox/models/exporters/StaticExporter.cfc:188` variable 'prc.entry' has no component ref |
| 11 | `osite → modules/contentbox/models/system/Site.cfc` | 2 | high | declares isLoaded(), getSlug(), removeAllSettings(), setSettings(), setCategories(); named like the receiver 'oSite' | `modules/contentbox/models/system/SiteService.cfc:471` variable 'oSite' has no component ref |
| 10 | `category → modules/contentbox/models/content/Category.cfc` | 5 | high | declares getCategory(), getCategoryID(); named like the receiver 'category' | `modules/contentbox/modules/contentbox-admin/views/contentStore/index.cfm:188` variable 'category' has no component ref |
| 10 | `thismodule → modules/contentbox/models/modules/Module.cfc` | 27 | low | declares getName() (19 candidates) | `modules/contentbox/models/modules/ModuleService.cfc:343` variable 'thisModule' has no component ref |
| 10 | `variables.logger → modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Editor.cfc` | 3 | medium | declares info() | `modules/contentbox/models/content/ContentService.cfc:887` variable 'variables.logger' has no component ref |
| 9 | `e.tagcontext` | 0 | none |  | `build/patches/3.5.0/Update.cfc:164` variable 'e.tagContext' has no component ref |
| 9 | `ocontent → modules/contentbox/models/content/BaseContent.cfc` | 10 | high | declares getMemento(), getCache(), isContentPublished(), isLoaded(), getContentID(), getCacheTimeout(), getCacheLastAccessTimeout() | `modules/contentbox/modules/contentbox-ui/handlers/content.cfc:290` variable 'oContent' has no component ref |
| 9 | `page` | 7 | none |  | `modules/contentbox/models/system/NotificationService.cfc:295` variable 'page' has no component ref |
| 9 | `prc.page` | 1 | none |  | `modules/contentbox/modules/contentbox-ui/handlers/page.cfc:65` variable 'prc.page' has no component ref |
| 8 | `arguments.category → modules/contentbox/models/content/Category.cfc` | 5 | medium | declares getSlug(), getCategory(); named like the receiver 'category' (2 candidates) | `modules/contentbox/models/content/CategoryService.cfc:96` variable 'arguments.category' has no component ref |
| 8 | `arguments.comment → modules/contentbox/models/comments/Comment.cfc` | 7 | medium | declares getRelatedContent(), getAuthorEmail(), getMemento(), getParentTitle(); named like the receiver 'comment' (2 candidates) | `modules/contentbox/models/comments/CommentService.cfc:245` variable 'arguments.comment' has no component ref |
| 8 | `arguments.content → modules/contentbox/models/content/Page.cfc` | 1 | high | declares getLayoutWithInheritance(), renderContent(), getSite(), getSlug(), hasChild(), getChildren() | `modules/contentbox/models/exporters/StaticExporter.cfc:318` variable 'arguments.content' has no component ref |
| 8 | `owidget → modules/contentbox/models/ui/Widget.cfc` | 3 | medium | declares getAuthorURL(), getName(), getDescription(), getVersion(), getAuthor(); named like the receiver 'oWidget' (2 candidates) | `modules/contentbox/modules/contentbox-admin/views/widgets/widgetList.cfm:112` variable 'oWidget' has no component ref |
| 8 | `requestservice` | 2 | none |  | `modules/contentbox/models/ui/AdminMenuService.cfc:203` variable 'requestService' has no component ref |
| 7 | `aperms` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:105` variable 'aPerms' has no component ref |
| 7 | `arguments.menuitem → modules/contentbox/models/menu/item/ContentMenuItem.cfc` | 1 | high | declares getContentSlug(), getMenu() | `modules/contentbox/models/menu/providers/ContentProvider.cfc:54` variable 'arguments.menuItem' has no component ref |
| 7 | `arguments.module → modules/contentbox/models/modules/Module.cfc` | 4 | high | declares setTitle(), setAuthor(), setWebURL(), setDescription(), setVersion(), setEntryPoint(), setForgeBoxSlug(); named like the receiver 'module' | `modules/contentbox/models/modules/ModuleService.cfc:117` variable 'arguments.module' has no component ref |
| 7 | `entryresults.content[]` | 5 | none |  | `modules/contentbox/models/rss/RSSService.cfc:212` variable 'entryResults.content[]' has no component ref |
| 7 | `exporter` | 1 | none |  | `modules/contentbox/models/exporters/ContentBoxExporter.cfc:422` variable 'exporter' has no component ref |
| 7 | `instance.ioinput` | 0 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:131` variable 'instance.ioInput' has no component ref |
| 7 | `pageresults.content[]` | 5 | none |  | `modules/contentbox/models/rss/RSSService.cfc:287` variable 'pageResults.content[]' has no component ref |
| 7 | `settingservice → modules/contentbox/models/system/SettingService.cfc` | 2 | medium | declares getAllSettings(); named like the receiver 'settingService' (2 candidates) | `modules/contentbox/models/rss/RSSService.cfc:36` variable 'settingService' has no component ref |
| 7 | `thisauthor → modules/contentbox/models/security/Author.cfc` | 2 | high | declares getEmail(), getRole(), getAUthorID(), getFullName() | `modules/contentbox/modules/contentbox-admin/views/content/search.cfm:66` variable 'thisAuthor' has no component ref |
| 6 | `arguments.target → modules/contentbox/models/content/BaseContent.cfc` | 50 | high | declares getContentService(), getContentType(), getContentID(), isLoaded(), hasSite(), getSite() | `modules/contentbox/models/content/BaseContent.cfc:568` variable 'arguments.target' has no component ref |
| 6 | `arguments.thiscontent → modules/contentbox/models/content/BaseContent.cfc` | 3 | low | declares hasParent() (2 candidates) | `modules/contentbox/models/system/Site.cfc:573` variable 'arguments.thisContent' has no component ref |
| 6 | `c.restrictions` | 1 | none |  | `modules/contentbox/models/content/ContentService.cfc:513` variable 'c.restrictions' has no component ref |
| 6 | `contentresults.content[] → modules/contentbox/models/content/BaseContent.cfc` | 3 | high | declares getAuthorEmail(), getAuthorName(), getCategoriesList(), renderContent() | `modules/contentbox/models/rss/RSSService.cfc:362` variable 'contentResults.content[]' has no component ref |
| 6 | `entries` | 0 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:203` variable 'entries' has no component ref |
| 6 | `item → modules/contentbox/models/system/Setting.cfc` | 6 | high | declares hasSite(), getName(), getSite(), getValue() | `modules/contentbox/models/system/SettingService.cfc:582` variable 'item' has no component ref |
| 6 | `menuitem → modules/contentbox/models/menu/item/ContentMenuItem.cfc` | 1 | medium | declares setContentSlug() | `modules/contentbox/modules/contentbox-admin/interceptors/MenuCleanup.cfc:27` variable 'menuItem' has no component ref |
| 6 | `row` | 10 | none |  | `modules/contentbox/models/security/AuthorService.cfc:135` variable 'row' has no component ref |
| 5 | `args.menuitem → modules/contentbox/models/menu/item/BaseMenuItem.cfc` | 1 | medium | declares getLabel() | `modules/contentbox/models/menu/views/free/display.cfm:2` variable 'args.menuItem' has no component ref |
| 5 | `args.provider` | 12 | none |  | `modules/contentbox/modules/contentbox-admin/views/menus/provider.cfm:11` variable 'args.provider' has no component ref |
| 5 | `arguments.target → modules/contentbox/models/content/Category.cfc` | 25 | high | declares getCategoryService(), getCategoryID(), isLoaded(), hasSite(), getSite() | `modules/contentbox/models/content/Category.cfc:120` variable 'arguments.target' has no component ref |
| 5 | `entry → modules/contentbox/models/content/Entry.cfc` | 7 | medium | declares getSite(), getTitle(), hasExcerpt(), renderExcerpt(), renderContent(); named like the receiver 'entry' (2 candidates) | `modules/contentbox/models/system/NotificationService.cfc:229` variable 'entry' has no component ref |
| 5 | `instance.iooutput → modules/contentbox/models/util/MobileDetector.cfc` | 223 | low | declares init() (169 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:67` variable 'instance.ioOutput' has no component ref |
| 5 | `instream` | 0 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:223` variable 'inStream' has no component ref |
| 5 | `item → modules/contentbox/models/content/BaseContent.cfc` | 4 | high | declares getTitle(), renderContent(), getContentType(), hasCategories(), getCategoriesList() | `modules/contentbox/models/search/DBSearch.cfc:139` variable 'item' has no component ref |
| 5 | `locpage → modules/contentbox/models/content/BaseContent.cfc` | 1 | high | declares getContentID(), hasParent(), getParent() | `modules/contentbox/models/system/CBHelper.cfc:2459` variable 'locPage' has no component ref |
| 5 | `ouser → modules/contentbox/models/security/Author.cfc` | 4 | high | declares setPermissions(), addPermissionGroup(), setRole(), isLoaded() | `modules/contentbox/models/security/AuthorService.cfc:481` variable 'oUser' has no component ref |
| 5 | `page → modules/contentbox/models/content/Page.cfc` | 7 | medium | declares getSite(), getTitle(), hasExcerpt(), renderExcerpt(), renderContent(); named like the receiver 'page' (2 candidates) | `modules/contentbox/models/system/NotificationService.cfc:359` variable 'page' has no component ref |
| 5 | `prc.availabletemplates` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/_components/editor/sidebar/Modifiers.cfm:24` variable 'prc.availableTemplates' has no component ref |
| 4 | `aauthors` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:239` variable 'aAuthors' has no component ref |
| 4 | `arguments.categories[] → modules/contentbox/models/content/Category.cfc` | 5 | low | declares getCategory(), getNumberOfEntries() (2 candidates) | `modules/contentbox/widgets/Categories.cfc:86` variable 'arguments.categories[]' has no component ref |
| 4 | `arguments.original → modules/contentbox/models/content/Page.cfc` | 1 | high | declares getLayout(), getShowInMenu(), hasExcerpt(), getExcerpt() | `modules/contentbox/models/content/Page.cfc:183` variable 'arguments.original' has no component ref |
| 4 | `arguments.populate.model → modules/contentbox/models/content/BaseContent.cfc` | 2 | high | declares setCreator(), isLoaded(), getSlug() | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseContentHandler.cfc:129` variable 'arguments.populate.model' has no component ref |
| 4 | `arguments.target` | 5 | none |  | `modules/contentbox/models/validators/UniqueSiteFieldValidator.cfc:48` variable 'arguments.target' has no component ref |
| 4 | `arguments.task` | 4 | none |  | `config/Scheduler.cfc:60` variable 'arguments.task' has no component ref |
| 4 | `arguments.thissite → modules/contentbox/models/content/BaseContent.cfc` | 5 | low | declares getSlug() (5 candidates) | `modules/contentbox/config/Scheduler.cfc:41` variable 'arguments.thisSite' has no component ref |
| 4 | `cbmessagebox` | 3 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/contentTemplates.cfc:111` variable 'cbMessagebox' has no component ref |
| 4 | `childtables` | 0 | none |  | `modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:166` variable 'childTables' has no component ref |
| 4 | `fkmap` | 0 | none |  | `modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:195` variable 'FKMap' has no component ref |
| 4 | `idtables` | 0 | none |  | `modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:252` variable 'idTables' has no component ref |
| 4 | `instance.iofile` | 223 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:125` variable 'instance.ioFile' has no component ref |
| 4 | `instance.iofile → modules/contentbox/models/util/MobileDetector.cfc` | 223 | low | declares init() (169 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:245` variable 'instance.ioFile' has no component ref |
| 4 | `instance.iooutput` | 0 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:397` variable 'instance.ioOutput' has no component ref |
| 4 | `original → modules/contentbox/models/content/BaseContent.cfc` | 5 | high | declares getTitle(), isSameSite(), getParent(), getSlug(), hasParent() | `modules/contentbox/modules/contentbox-admin/handlers/baseContentHandler.cfc:588` variable 'original' has no component ref |
| 4 | `path` | 0 | none |  | `build/patches/5.0.0/Updater.cfc:143` variable 'path' has no component ref |
| 4 | `perm → modules/contentbox/models/security/Permission.cfc` | 1 | high | declares getPermission(), getPermissionID() | `modules/contentbox/modules/contentbox-admin/views/authors/permissions.cfm:21` variable 'perm' has no component ref |
| 4 | `prc.oentity → modules/contentbox/models/content/BaseContent.cfc` | 1 | high | declares addNewContentVersion(), setCategories(), setCustomFields(), getMemento() | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseContentHandler.cfc:162` variable 'prc.oEntity' has no component ref |
| 4 | `prc.owidget → modules/contentbox/models/modules/Module.cfc` | 27 | low | declares getName(), getVersion(), getForgeBoxSlug(), getDescription() (2 candidates) | `modules/contentbox/modules/contentbox-admin/views/widgets/docs.cfm:7` variable 'prc.oWidget' has no component ref |
| 4 | `prc.page → modules/contentbox/models/content/BaseContent.cfc` | 4 | high | declares getTitle(), getSlug(), renderContent(), getNumberOfApprovedComments() | `modules/contentbox/themes/default/views/page.cfm:50` variable 'prc.page' has no component ref |
| 4 | `rc.slug` | 0 | none |  | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/contentStore.cfc:46` variable 'rc.slug' has no component ref |
| 4 | `result` | 3 | none |  | `modules/contentbox/models/security/Author.cfc:333` variable 'result' has no component ref |
| 4 | `securityinterceptor` | 63 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/securityRules.cfc:47` variable 'securityInterceptor' has no component ref |
| 4 | `stat → modules/contentbox/models/content/Stats.cfc` | 7 | high | declares getRelatedContent(), getHits() | `modules/contentbox/models/content/ContentService.cfc:791` variable 'stat' has no component ref |
| 4 | `thisgroup → modules/contentbox/models/security/PermissionGroup.cfc` | 1 | high | declares getPermissionGroupID(), getName() | `modules/contentbox/modules/contentbox-admin/views/authors/new.cfm:132` variable 'thisGroup' has no component ref |
| 3 | `allperms` | 0 | none |  | `modules/contentbox/migrations/init/seed_roles.cfc:36` variable 'allPerms' has no component ref |
| 3 | `arguments.author` | 2 | none |  | `modules/contentbox/models/security/AuthorService.cfc:225` variable 'arguments.author' has no component ref |
| 3 | `arguments.c.restrictions` | 1 | none |  | `modules/contentbox/models/content/ContentService.cfc:535` variable 'arguments.c.restrictions' has no component ref |
| 3 | `arguments.data.builder` | 0 | none |  | `modules/contentbox/models/content/renderers/MarkdownRenderer.cfc:34` variable 'arguments.data.builder' has no component ref |
| 3 | `arguments.entity` | 2 | none |  | `modules/contentbox/models/security/SecurityRuleService.cfc:55` variable 'arguments.entity' has no component ref |
| 3 | `arguments.entry → modules/contentbox/models/system/Site.cfc` | 5 | low | declares getSlug() (5 candidates) | `modules/contentbox/models/system/CBHelper.cfc:1433` variable 'arguments.entry' has no component ref |
| 3 | `arguments.menuitem → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/models/menu/providers/FreeProvider.cfc:55` variable 'arguments.menuItem' has no component ref |
| 3 | `arguments.page → modules/contentbox/models/system/Site.cfc` | 5 | low | declares getSlug() (5 candidates) | `modules/contentbox/models/system/CBHelper.cfc:1510` variable 'arguments.page' has no component ref |
| 3 | `arguments.parent → modules/contentbox/models/content/BaseContent.cfc` | 2 | high | declares addChild(), getSlug() | `modules/contentbox/models/content/BaseContent.cfc:1803` variable 'arguments.parent' has no component ref |
| 3 | `arguments.site` | 2 | none |  | `modules/contentbox/models/system/SiteService.cfc:615` variable 'arguments.site' has no component ref |
| 3 | `arguments.thischild → modules/contentbox/models/content/BaseContent.cfc` | 4 | low | declares getTitle(), getSlug() (2 candidates) | `modules/contentbox/models/content/BaseContent.cfc:1415` variable 'arguments.thisChild' has no component ref |
| 3 | `commentresults.comments[] → modules/contentbox/models/comments/Comment.cfc` | 2 | low | declares getParentTitle() (2 candidates) | `modules/contentbox/models/rss/RSSService.cfc:433` variable 'commentResults.comments[]' has no component ref |
| 3 | `fileutils` | 0 | none |  | `build/patches/1-0-5/Update.cfc:57` variable 'fileUtils' has no component ref |
| 3 | `instance.ioinput → modules/contentbox/models/util/MobileDetector.cfc` | 223 | low | declares init() (169 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:126` variable 'instance.ioInput' has no component ref |
| 3 | `item → modules/contentbox/models/security/Permission.cfc` | 1 | medium | declares getPermission() | `modules/contentbox/models/security/Author.cfc:333` variable 'item' has no component ref |
| 3 | `key` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/comments.cfc:400` variable 'key' has no component ref |
| 3 | `local.thissubmenu` | 1 | none |  | `modules/contentbox/models/ui/templates/navAdminMenu.cfm:84` variable 'local.thisSubMenu' has no component ref |
| 3 | `local.topmenu` | 1 | none |  | `modules/contentbox/models/ui/templates/navAdminMenu.cfm:20` variable 'local.topMenu' has no component ref |
| 3 | `menu → modules/contentbox/models/menu/Menu.cfc` | 5 | medium | declares getSlug(), getTitle(); named like the receiver 'menu' (2 candidates) | `modules/contentbox/modules/contentbox-admin/views/menus/providers/submenu/admin.cfm:9` variable 'menu' has no component ref |
| 3 | `ocategory → modules/contentbox/models/content/BaseContent.cfc` | 6 | high | declares setSite(), isLoaded() | `modules/contentbox/models/content/CategoryService.cfc:462` variable 'oCategory' has no component ref |
| 3 | `ogroup` | 4 | none |  | `modules/contentbox/models/security/PermissionGroupService.cfc:125` variable 'oGroup' has no component ref |
| 3 | `orole` | 4 | none |  | `modules/contentbox/models/security/RoleService.cfc:127` variable 'oRole' has no component ref |
| 3 | `otemplate → modules/contentbox/models/content/BaseContent.cfc` | 6 | high | declares setSite(), isLoaded() | `modules/contentbox/models/content/ContentTemplateService.cfc:311` variable 'oTemplate' has no component ref |
| 3 | `prc.activedisk → modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/authors.cfc` | 13 | low | declares create() (11 candidates) | `modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Editor.cfc:214` variable 'prc.activeDisk' has no component ref |
| 3 | `prc.adminmenuservice → modules/contentbox/models/ui/AdminMenuService.cfc` | 1 | high | declares generateUtilsMenu(), generateProfileMenu(), generateMenu(); named like the receiver 'adminMenuService' | `modules/contentbox/modules/contentbox-admin/layouts/admin.cfm:244` variable 'prc.adminMenuService' has no component ref |
| 3 | `prc.oeditordriver → modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` | 9 | low | declares startup(), shutdown(), loadAssets() (4 candidates) | `modules/contentbox/modules/contentbox-admin/views/content/editorHelper.cfm:30` variable 'prc.oEditorDriver' has no component ref |
| 3 | `this.site_defaults` | 0 | none |  | `modules/contentbox/models/system/SettingService.cfc:305` variable 'this.SITE_DEFAULTS' has no component ref |
| 3 | `thisitem` | 0 | none |  | `tests/specs/contentbox-api/AuthorsSpec.cfc:59` variable 'thisItem' has no component ref |
| 3 | `thisitem → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/models/content/CategoryService.cfc:338` variable 'thisItem' has no component ref |
| 3 | `thispage → modules/contentbox/models/content/BaseContent.cfc` | 5 | low | declares getSlug(), setSlug() (5 candidates) | `modules/contentbox/models/content/PageService.cfc:40` variable 'thisPage' has no component ref |
| 3 | `thisrule` | 0 | none |  | `modules/contentbox/models/security/SecurityRuleService.cfc:186` variable 'thisRule' has no component ref |
| 3 | `variables.customfields` | 7 | none |  | `modules/contentbox/models/content/BaseContent.cfc:998` variable 'variables.customFields' has no component ref |
| 3 | `variables.ormservice → modules/contentbox/models/system/SiteService.cfc` | 4 | medium | declares getOrFail() | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseHandler.cfc:93` variable 'variables.ormService' has no component ref |
| 3 | `variables.permissionlist` | 0 | none |  | `modules/contentbox/models/security/Author.cfc:379` variable 'variables.permissionList' has no component ref |
| 2 | `activecontent → modules/contentbox/models/content/ContentVersion.cfc` | 6 | high | declares getAuthor(), hasAuthor() | `modules/contentbox/models/content/BaseContent.cfc:661` variable 'activeContent' has no component ref |
| 2 | `args.menuitem → modules/contentbox/models/menu/item/MediaMenuItem.cfc` | 1 | medium | declares getMediaPath() | `modules/contentbox/modules/contentbox-admin/views/menus/providers/media/admin.cfm:10` variable 'args.menuItem' has no component ref |
| 2 | `arguments.appscope.cbbootstrap → Application.cfc` | 3 | low | declares onSessionEnd() (2 candidates) | `Application.cfc:182` variable 'arguments.appScope.cbBootStrap' has no component ref |
| 2 | `arguments.content → modules/contentbox/models/system/Site.cfc` | 1 | high | declares getSiteRoot(); named like the receiver 'Site' | `modules/contentbox/models/exporters/StaticExporter.cfc:276` variable 'arguments.content' has no component ref |
| 2 | `arguments.data.menu → modules/contentbox/models/menu/Menu.cfc` | 5 | medium | declares getSlug(); named like the receiver 'menu' (5 candidates) | `modules/contentbox/modules/contentbox-admin/interceptors/MenuCleanup.cfc:100` variable 'arguments.data.menu' has no component ref |
| 2 | `arguments.menu` | 1 | none |  | `modules/contentbox/models/ui/AdminMenuService.cfc:156` variable 'arguments.menu' has no component ref |
| 2 | `arguments.menuitem → modules/contentbox/models/menu/Menu.cfc` | 5 | medium | declares getSiteID(); named like the receiver 'Menu' (5 candidates) | `modules/contentbox/models/menu/providers/ContentProvider.cfc:59` variable 'arguments.menuItem' has no component ref |
| 2 | `arguments.menuitem → modules/contentbox/models/menu/item/SubMenuItem.cfc` | 1 | medium | declares getMenuSlug() | `modules/contentbox/models/menu/providers/SubMenuProvider.cfc:53` variable 'arguments.menuItem' has no component ref |
| 2 | `arguments.original → modules/contentbox/models/content/ContentStore.cfc` | 12 | high | declares getDescription(), getOrder() | `modules/contentbox/models/content/ContentStore.cfc:98` variable 'arguments.original' has no component ref |
| 2 | `arguments.original → modules/contentbox/models/content/Entry.cfc` | 2 | low | declares hasExcerpt(), getExcerpt() (2 candidates) | `modules/contentbox/models/content/Entry.cfc:126` variable 'arguments.original' has no component ref |
| 2 | `arguments.originalservice → modules/contentbox/models/content/CategoryService.cfc` | 24 | low | declares save() (23 candidates) | `modules/contentbox/models/content/BaseContent.cfc:1405` variable 'arguments.originalService' has no component ref |
| 2 | `arguments.relatedcontent[] → modules/contentbox/models/content/BaseContent.cfc` | 4 | low | declares getTitle() (4 candidates) | `modules/contentbox/widgets/RelatedContent.cfc:80` variable 'arguments.relatedContent[]' has no component ref |
| 2 | `arguments.site → modules/contentbox/models/content/BaseContent.cfc` | 2 | high | declares isLoaded(), getsiteID() | `modules/contentbox/models/content/CategoryService.cfc:129` variable 'arguments.site' has no component ref |
| 2 | `arguments.target → modules/contentbox/models/system/Site.cfc` | 5 | medium | declares getSiteID(); named like the receiver 'Site' (5 candidates) | `modules/contentbox/models/content/BaseContent.cfc:579` variable 'arguments.target' has no component ref |
| 2 | `arguments.thisentry → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/modules/contentbox-ui/handlers/blog.cfc:147` variable 'arguments.thisEntry' has no component ref |
| 2 | `arguments.thisfield → modules/contentbox/models/content/CustomField.cfc` | 3 | high | declares getKey(), getValue() | `modules/contentbox/models/content/BaseContent.cfc:1374` variable 'arguments.thisField' has no component ref |
| 2 | `arguments.thisitem → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getContentID() | `modules/contentbox/modules/contentbox-admin/handlers/baseContentHandler.cfc:235` variable 'arguments.thisItem' has no component ref |
| 2 | `asites` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:186` variable 'aSites' has no component ref |
| 2 | `backpath` | 0 | none |  | `tests/index.cfm:42` variable 'backPath' has no component ref |
| 2 | `cats[] → modules/contentbox/models/system/Site.cfc` | 5 | low | declares getCategory() (4 candidates) | `modules/contentbox/models/system/CBHelper.cfc:1848` variable 'cats[]' has no component ref |
| 2 | `clone → modules/contentbox/models/content/BaseContent.cfc` | 9 | high | declares clone(), getSlug() | `modules/contentbox/modules/contentbox-admin/handlers/baseContentHandler.cfc:583` variable 'clone' has no component ref |
| 2 | `content → modules/contentbox/models/comments/Comment.cfc` | 4 | low | declares getContent() (4 candidates) | `modules/contentbox/models/system/NotificationService.cfc:450` variable 'content' has no component ref |
| 2 | `contentitem → modules/contentbox/models/content/BaseContent.cfc` | 1 | high | declares setContentTemplate(), setChildContentTemplate() | `modules/contentbox/models/content/ContentTemplateService.cfc:109` variable 'contentItem' has no component ref |
| 2 | `contentobjects[] → modules/contentbox/models/content/BaseContent.cfc` | 1 | high | declares setpublishedDate(), setisPublished() | `modules/contentbox/models/content/ContentService.cfc:627` variable 'contentObjects[]' has no component ref |
| 2 | `e → modules/contentbox/models/system/Site.cfc` | 2 | low | declares hasCategories(), getCategories() (2 candidates) | `modules/contentbox/models/system/CBHelper.cfc:1839` variable 'e' has no component ref |
| 2 | `entry → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getIsPublished() | `modules/contentbox/models/content/EntryService.cfc:32` variable 'entry' has no component ref |
| 2 | `fkeys` | 0 | none |  | `modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:203` variable 'FKeys' has no component ref |
| 2 | `incomment → modules/contentbox/models/content/BaseContent.cfc` | 3 | low | declares getSiteSlug() (3 candidates) | `modules/contentbox/models/comments/CommentService.cfc:197` variable 'inComment' has no component ref |
| 2 | `instance.gzinput` | 0 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:592` variable 'instance.gzInput' has no component ref |
| 2 | `instance.gzoutput` | 0 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:525` variable 'instance.gzOutput' has no component ref |
| 2 | `instance.iobufoutput` | 0 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:387` variable 'instance.ioBufOutput' has no component ref |
| 2 | `instance.iofile → modules/contentbox/models/content/CategoryService.cfc` | 22 | low | declares delete() (17 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:248` variable 'instance.ioFile' has no component ref |
| 2 | `instance.zipentry → modules/contentbox/models/util/MobileDetector.cfc` | 223 | low | declares init() (169 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:128` variable 'instance.zipEntry' has no component ref |
| 2 | `menuitem → modules/contentbox/models/menu/item/MediaMenuItem.cfc` | 1 | high | declares setMediaPath(), setActive() | `modules/contentbox/modules/contentbox-admin/interceptors/MenuCleanup.cfc:163` variable 'menuItem' has no component ref |
| 2 | `menuitem → modules/contentbox/models/menu/item/SubMenuItem.cfc` | 1 | high | declares setMenuSlug(), setActive() | `modules/contentbox/modules/contentbox-admin/interceptors/MenuCleanup.cfc:122` variable 'menuItem' has no component ref |
| 2 | `module → modules/contentbox/models/modules/Module.cfc` | 27 | high | declares getName(), getTitle(); named like the receiver 'module' | `modules/contentbox/modules/contentbox-admin/views/tools/exporter.cfm:286` variable 'module' has no component ref |
| 2 | `newchild → modules/contentbox/models/content/BaseContent.cfc` | 9 | high | declares clone(), getSlug() | `modules/contentbox/models/content/BaseContent.cfc:1421` variable 'newChild' has no component ref |
| 2 | `newtemplate → modules/contentbox/models/content/ContentTemplate.cfc` | 1 | high | declares getSchema(), setDefinition() | `modules/contentbox/models/content/ContentTemplateService.cfc:176` variable 'newTemplate' has no component ref |
| 2 | `ocurrentcontent → modules/contentbox/models/system/Site.cfc` | 5 | low | declares getslug() (5 candidates) | `modules/contentbox/models/system/CBHelper.cfc:1055` variable 'oCurrentContent' has no component ref |
| 2 | `opermission → modules/contentbox/models/BaseEntityMethods.cfc` | 2 | low | declares isLoaded() (2 candidates) | `modules/contentbox/models/security/PermissionService.cfc:155` variable 'oPermission' has no component ref |
| 2 | `orule → modules/contentbox/models/BaseEntityMethods.cfc` | 2 | low | declares isLoaded() (2 candidates) | `modules/contentbox/models/security/SecurityRuleService.cfc:185` variable 'oRule' has no component ref |
| 2 | `osite → modules/contentbox/models/BaseEntityMethods.cfc` | 2 | low | declares isLoaded() (2 candidates) | `modules/contentbox/models/system/SiteService.cfc:392` variable 'oSite' has no component ref |
| 2 | `outputstream` | 0 | none |  | `modules/contentbox/models/media/ForwardMediaProvider.cfc:64` variable 'outputStream' has no component ref |
| 2 | `pageresults.content[] → modules/contentbox/models/content/BaseContent.cfc` | 2 | low | declares hasChild() (2 candidates) | `modules/contentbox/models/system/CBHelper.cfc:2536` variable 'pageResults.content[]' has no component ref |
| 2 | `prc.activedisk → modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` | 27 | low | declares getName() (19 candidates) | `modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/views/home/filelisting.cfm:33` variable 'prc.activeDisk' has no component ref |
| 2 | `prc.entries` | 1 | none |  | `modules/contentbox/modules/contentbox-ui/handlers/blog.cfc:145` variable 'prc.entries' has no component ref |
| 2 | `prc.fblisting` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/views/home/filelisting.cfm:81` variable 'prc.fbListing' has no component ref |
| 2 | `prc.oentity → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseHandler.cfc:107` variable 'prc.oEntity' has no component ref |
| 2 | `prc.opaging → modules/contentbox/models/ui/Paging.cfc` | 26 | medium | declares renderit(); named like the receiver 'oPaging' (25 candidates) | `modules/contentbox/models/system/CBHelper.cfc:1219` variable 'prc.oPaging' has no component ref |
| 2 | `prc.orelatedcontent → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getContentID() | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/comments.cfc:57` variable 'prc.oRelatedContent' has no component ref |
| 2 | `prc.widget.widget → modules/contentbox/models/ui/Widget.cfc` | 27 | medium | declares getName(); named like the receiver 'widget' (19 candidates) | `modules/contentbox/modules/contentbox-admin/views/widgets/instance.cfm:7` variable 'prc.widget.widget' has no component ref |
| 2 | `rc.authorid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/authors.cfc:820` variable 'rc.authorID' has no component ref |
| 2 | `rc.categoryid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/categories.cfc:191` variable 'rc.categoryID' has no component ref |
| 2 | `rc.commentid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/comments.cfc:263` variable 'rc.commentID' has no component ref |
| 2 | `rc.contentid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/baseContentHandler.cfc:739` variable 'rc.contentID' has no component ref |
| 2 | `rc.menuid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/menus.cfc:328` variable 'rc.menuID' has no component ref |
| 2 | `rc.permissiongroupid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/permissionGroups.cfc:170` variable 'rc.permissionGroupID' has no component ref |
| 2 | `rc.permissionid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/permissions.cfc:135` variable 'rc.permissionID' has no component ref |
| 2 | `rc.roleid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/roles.cfc:185` variable 'rc.roleID' has no component ref |
| 2 | `rc.securityruleid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/securityRules.cfc:188` variable 'rc.securityRuleID' has no component ref |
| 2 | `rc.settingid` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/settings.cfc:214` variable 'rc.settingID' has no component ref |
| 2 | `relatedcontent[] → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares isContentPublished() | `modules/contentbox/widgets/RelatedContent.cfc:78` variable 'relatedContent[]' has no component ref |
| 2 | `server.lucee.version` | 1 | none |  | `modules/contentbox/models/content/renderers/WidgetRenderer.cfc:95` variable 'server.lucee.version' has no component ref |
| 2 | `settings` | 0 | none |  | `modules/contentbox-installer/models/InstallerService.cfc:156` variable 'settings' has no component ref |
| 2 | `stat → modules/contentbox/models/content/BaseContent.cfc` | 1 | high | declares getContentID(), getTitle() | `modules/contentbox/models/content/ContentService.cfc:791` variable 'stat' has no component ref |
| 2 | `subscriber → modules/contentbox/models/subscriptions/Subscriber.cfc` | 1 | high | declares getSubscriberEmail(); named like the receiver 'subscriber' | `modules/contentbox/models/comments/CommentService.cfc:261` variable 'subscriber' has no component ref |
| 2 | `subscription → modules/contentbox/models/subscriptions/BaseSubscription.cfc` | 1 | high | declares getSubscriber(), getSubscriptionToken() | `modules/contentbox/models/comments/CommentService.cfc:258` variable 'subscription' has no component ref |
| 2 | `target → modules/contentbox/models/content/ContentTemplate.cfc` | 1 | high | declares isNameUniqueInSite(), isGlobalUniqueInSite() | `modules/contentbox/models/content/ContentTemplate.cfc:308` variable 'target' has no component ref |
| 2 | `targetsite → modules/contentbox/models/system/Site.cfc` | 5 | low | declares getSlug() (5 candidates) | `modules/contentbox/models/system/SettingService.cfc:314` variable 'targetSite' has no component ref |
| 2 | `templates` | 1 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/contentTemplates.cfc:113` variable 'templates' has no component ref |
| 2 | `themerecord.descriptor → modules/contentbox/themes/default/Theme.cfc` | 1 | medium | declares onDeactivation() | `modules/contentbox/models/ui/ThemeService.cfc:364` variable 'themeRecord.descriptor' has no component ref |
| 2 | `themerecord.modules` | 0 | none |  | `modules/contentbox/models/ui/ThemeService.cfc:246` variable 'themeRecord.modules' has no component ref |
| 2 | `this.defaults` | 0 | none |  | `modules/contentbox/models/system/SettingService.cfc:259` variable 'this.DEFAULTS' has no component ref |
| 2 | `thiscontent.categories` | 1 | none |  | `modules/contentbox/models/content/ContentService.cfc:1061` variable 'thisContent.categories' has no component ref |
| 2 | `thisrole → modules/contentbox/models/security/Author.cfc` | 2 | low | declares getRole() (2 candidates) | `modules/contentbox/modules/contentbox-admin/views/securityRules/editor.cfm:173` variable 'thisRole' has no component ref |
| 2 | `topcommented → modules/contentbox/models/content/BaseContent.cfc` | 4 | high | declares getTitle(), getNumberOfComments() | `modules/contentbox/modules/contentbox-admin/views/dashboard/latestSnapshot.cfm:55` variable 'topCommented' has no component ref |
| 2 | `validationresult` | 1 | none |  | `modules/contentbox/models/validators/UniqueSiteFieldValidator.cfc:75` variable 'validationResult' has no component ref |
| 2 | `variables.activecontentversions` | 7 | none |  | `modules/contentbox/models/content/BaseContent.cfc:859` variable 'variables.activeContentVersions' has no component ref |
| 2 | `variables.comments` | 7 | none |  | `modules/contentbox/models/content/BaseContent.cfc:985` variable 'variables.comments' has no component ref |
| 2 | `variables.excludes` | 0 | none |  | `build/patches/5.0.0/Updater.cfc:142` variable 'variables.excludes' has no component ref |
| 2 | `variables.idtables` | 0 | none |  | `modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:134` variable 'variables.idTables' has no component ref |
| 2 | `variables.relatedcontent` | 7 | none |  | `modules/contentbox/models/content/BaseContent.cfc:932` variable 'variables.relatedContent' has no component ref |
| 2 | `variables.settingservice → modules/contentbox/models/system/SettingService.cfc` | 2 | medium | declares getAllSettings(); named like the receiver 'settingService' (2 candidates) | `modules/contentbox/models/rss/RSSService.cfc:326` variable 'variables.settingService' has no component ref |
| 2 | `variables.themeregistry` | 0 | none |  | `modules/contentbox/models/ui/ThemeService.cfc:408` variable 'variables.themeRegistry' has no component ref |
| 2 | `variables.topmenumap` | 0 | none |  | `modules/contentbox/models/ui/AdminMenuService.cfc:796` variable 'variables.topMenuMap' has no component ref |
| 2 | `variables[] → modules/contentbox/models/content/CategoryService.cfc` | 14 | low | declares getAllForExport() (14 candidates) | `modules/contentbox/models/exporters/ContentBoxExporter.cfc:235` variable 'variables[]' has no component ref |
| 2 | `version → modules/contentbox/models/content/ContentVersion.cfc` | 4 | low | declares getIsActive(), setIsActive() (4 candidates) | `modules/contentbox/models/content/BaseContent.cfc:831` variable 'version' has no component ref |
| 2 | `widget → modules/contentbox/models/ui/Widget.cfc` | 2 | medium | declares getIcon(); named like the receiver 'widget' (2 candidates) | `modules/contentbox/models/ui/WidgetService.cfc:506` variable 'widget' has no component ref |
| 1 | `acategories` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:222` variable 'aCategories' has no component ref |
| 1 | `acc[].value` | 3 | none |  | `modules/contentbox/models/content/ContentTemplateService.cfc:192` variable 'acc[].value' has no component ref |
| 1 | `acomments` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:292` variable 'aComments' has no component ref |
| 1 | `acommentsubscriptions` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:314` variable 'aCommentSubscriptions' has no component ref |
| 1 | `acontent` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:285` variable 'aContent' has no component ref |
| 1 | `acontentcategories` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:360` variable 'aContentCategories' has no component ref |
| 1 | `acontentstore` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:420` variable 'aContentStore' has no component ref |
| 1 | `acontentversions` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:429` variable 'aContentVersions' has no component ref |
| 1 | `activecontent → modules/contentbox/models/security/Author.cfc` | 5 | medium | declares getInfoSnapshot(); named like the receiver 'Author' (5 candidates) | `modules/contentbox/models/content/BaseContent.cfc:661` variable 'activeContent' has no component ref |
| 1 | `acustomfields` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:457` variable 'aCustomFields' has no component ref |
| 1 | `aentries` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:466` variable 'aEntries' has no component ref |
| 1 | `agrouppermissions` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:160` variable 'aGroupPermissions' has no component ref |
| 1 | `alias` | 0 | none |  | `modules/contentbox/models/system/Site.cfc:502` variable 'alias' has no component ref |
| 1 | `aloginattempts` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:624` variable 'aLoginAttempts' has no component ref |
| 1 | `amenuitems` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:554` variable 'aMenuItems' has no component ref |
| 1 | `amenus` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:502` variable 'aMenus' has no component ref |
| 1 | `apages` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:564` variable 'aPages' has no component ref |
| 1 | `apermissiongroups` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:130` variable 'aPermissionGroups' has no component ref |
| 1 | `application.cbcontroller → modules/contentbox/models/system/SettingService.cfc` | 3 | medium | declares getSetting() | `Application.cfc:156` variable 'application.cbController' has no component ref |
| 1 | `application.wirebox` | 4 | none |  | `config/modules/cbfs.cfc:21` variable 'application.wirebox' has no component ref |
| 1 | `arelatedcontent` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:578` variable 'aRelatedContent' has no component ref |
| 1 | `arguments.args` | 0 | none |  | `modules/contentbox/widgets/Renderview.cfc:52` variable 'arguments.args' has no component ref |
| 1 | `arguments.comment → modules/contentbox/models/content/BaseContent.cfc` | 3 | low | declares getContentType() (2 candidates) | `modules/contentbox/models/system/CBHelper.cfc:1751` variable 'arguments.comment' has no component ref |
| 1 | `arguments.criteria` | 1 | none |  | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseHandler.cfc:66` variable 'arguments.criteria' has no component ref |
| 1 | `arguments.data.comment → modules/contentbox/models/comments/Comment.cfc` | 7 | medium | declares getRelatedContent(); named like the receiver 'comment' (7 candidates) | `modules/contentbox/models/rss/RSSCacheCleanup.cfc:40` variable 'arguments.data.comment' has no component ref |
| 1 | `arguments.data.comment → modules/contentbox/models/content/BaseContent.cfc` | 5 | low | declares getSlug() (5 candidates) | `modules/contentbox/models/rss/RSSCacheCleanup.cfc:44` variable 'arguments.data.comment' has no component ref |
| 1 | `arguments.field → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/modules/contentbox-admin/views/_components/editor/CustomFieldsHelper.cfm:6` variable 'arguments.field' has no component ref |
| 1 | `arguments.group → modules/contentbox/models/security/PermissionGroup.cfc` | 1 | medium | declares addAuthor() | `modules/contentbox/models/security/Author.cfc:446` variable 'arguments.group' has no component ref |
| 1 | `arguments.item` | 0 | none |  | `modules/contentbox/models/importers/ContentBoxImporter.cfc:88` variable 'arguments.item' has no component ref |
| 1 | `arguments.menuitem → modules/contentbox/models/menu/item/BaseMenuItem.cfc` | 1 | medium | declares setMenu() | `modules/contentbox/models/menu/Menu.cfc:173` variable 'arguments.menuItem' has no component ref |
| 1 | `arguments.result` | 3 | none |  | `modules/contentbox/models/system/SiteService.cfc:622` variable 'arguments.result' has no component ref |
| 1 | `arguments.rule.roles` | 0 | none |  | `modules/contentbox/models/security/SecurityValidator.cfc:69` variable 'arguments.rule.roles' has no component ref |
| 1 | `arguments.spec` | 0 | none |  | `tests/specs/contentbox-web/contentbox/unit/modules/ModuleServiceTest.cfc:34` variable 'arguments.spec' has no component ref |
| 1 | `arguments.target → modules/contentbox/models/content/CategoryService.cfc` | 3 | medium | declares isSlugUnique(); named like the receiver 'CategoryService' (3 candidates) | `modules/contentbox/models/content/Category.cfc:123` variable 'arguments.target' has no component ref |
| 1 | `arguments.target → modules/contentbox/models/content/ContentService.cfc` | 3 | medium | declares isSlugUnique(); named like the receiver 'ContentService' (3 candidates) | `modules/contentbox/models/content/BaseContent.cfc:571` variable 'arguments.target' has no component ref |
| 1 | `arguments.thisitem → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/models/content/ContentService.cfc:833` variable 'arguments.thisItem' has no component ref |
| 1 | `arguments.thispage → modules/contentbox/models/content/BaseContent.cfc` | 3 | low | declares hasParent() (2 candidates) | `modules/contentbox/models/system/Site.cfc:558` variable 'arguments.thisPage' has no component ref |
| 1 | `arolepermission` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:178` variable 'aRolePermission' has no component ref |
| 1 | `aroles` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:169` variable 'aRoles' has no component ref |
| 1 | `asecurityrules` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:231` variable 'aSecurityRules' has no component ref |
| 1 | `asettings` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:196` variable 'aSettings' has no component ref |
| 1 | `astats` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:588` variable 'aStats' has no component ref |
| 1 | `asubscribers` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:597` variable 'aSubscribers' has no component ref |
| 1 | `asubscriptions` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:606` variable 'aSubscriptions' has no component ref |
| 1 | `c → modules/contentbox/models/util/ZipUtil.cfc` | 4 | medium | declares list() | `modules/contentbox/models/content/ContentService.cfc:600` variable 'c' has no component ref |
| 1 | `child → modules/contentbox/models/menu/item/BaseMenuItem.cfc` | 3 | low | declares setParent() (2 candidates) | `modules/contentbox/models/menu/Menu.cfc:276` variable 'child' has no component ref |
| 1 | `config → modules/contentbox/modules_user/Hello/ModuleConfig.cfc` | 2 | low | declares onDeactivate() (2 candidates) | `modules/contentbox/models/modules/ModuleService.cfc:237` variable 'config' has no component ref |
| 1 | `contentitem.getparent → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getContentID() | `modules/contentbox/models/content/ContentTemplateService.cfc:217` variable 'contentItem.getParent' has no component ref |
| 1 | `data.newsettings` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/interceptors/UnenrollTwoFactorOnProviderChange.cfc:24` variable 'data.newSettings' has no component ref |
| 1 | `data.oldsettings` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/interceptors/UnenrollTwoFactorOnProviderChange.cfc:23` variable 'data.oldSettings' has no component ref |
| 1 | `definition[]` | 0 | none |  | `modules/contentbox/models/content/BaseContent.cfc:1833` variable 'definition[]' has no component ref |
| 1 | `definition[].value` | 3 | none |  | `modules/contentbox/models/content/BaseContent.cfc:1856` variable 'definition[].value' has no component ref |
| 1 | `e → modules/contentbox/models/security/LoginAttempt.cfc` | 1 | medium | declares setIsBlocked() | `modules/contentbox/models/security/LoginTrackerService.cfc:63` variable 'e' has no component ref |
| 1 | `editorperms` | 0 | none |  | `modules/contentbox/migrations/init/seed_roles.cfc:89` variable 'editorPerms' has no component ref |
| 1 | `entries[] → modules/contentbox/models/comments/Comment.cfc` | 6 | low | declares getAuthor() (6 candidates) | `tests/specs/contentbox-web/contentbox/unit/content/EntryServiceTest.cfc:65` variable 'entries[]' has no component ref |
| 1 | `entries[] → modules/contentbox/models/security/Author.cfc` | 1 | high | declares getAuthorID(); named like the receiver 'Author' | `tests/specs/contentbox-web/contentbox/unit/content/EntryServiceTest.cfc:65` variable 'entries[]' has no component ref |
| 1 | `entry → modules/contentbox/models/comments/Comment.cfc` | 4 | low | declares getContent() (4 candidates) | `modules/contentbox/models/system/NotificationService.cfc:190` variable 'entry' has no component ref |
| 1 | `entryresults.content[] → modules/contentbox/models/content/BaseContent.cfc` | 4 | low | declares getTitle() (4 candidates) | `modules/contentbox/widgets/RecentEntries.cfc:95` variable 'entryResults.content[]' has no component ref |
| 1 | `existingfields` | 0 | none |  | `modules/contentbox/models/content/BaseContent.cfc:1835` variable 'existingFields' has no component ref |
| 1 | `filesystemutil` | 0 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:44` variable 'fileSystemUtil' has no component ref |
| 1 | `grammar` | 0 | none |  | `modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:535` variable 'grammar' has no component ref |
| 1 | `headermenumap` | 0 | none |  | `modules/contentbox/models/ui/AdminMenuService.cfc:850` variable 'headerMenuMap' has no component ref |
| 1 | `importer → modules/contentbox/modules/contentbox-admin/handlers/modules.cfc` | 9 | low | declares execute() (8 candidates) | `modules/contentbox/modules/contentbox-admin/handlers/tools.cfc:133` variable 'importer' has no component ref |
| 1 | `instance.gzinput → modules/contentbox/models/util/MobileDetector.cfc` | 223 | low | declares init() (169 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:584` variable 'instance.gzInput' has no component ref |
| 1 | `instance.gzoutput → modules/contentbox/models/util/MobileDetector.cfc` | 223 | low | declares init() (169 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:520` variable 'instance.gzOutput' has no component ref |
| 1 | `instance.iobufoutput → modules/contentbox/models/util/MobileDetector.cfc` | 223 | low | declares init() (169 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:381` variable 'instance.ioBufOutput' has no component ref |
| 1 | `instance.objdate → modules/contentbox/models/util/MobileDetector.cfc` | 223 | low | declares init() (169 candidates) | `modules/contentbox/models/util/ZipUtil.cfc:460` variable 'instance.objDate' has no component ref |
| 1 | `item → modules/contentbox/models/menu/item/MediaMenuItem.cfc` | 1 | medium | declares setMediaPath() | `modules/contentbox/modules/contentbox-admin/interceptors/MenuCleanup.cfc:150` variable 'item' has no component ref |
| 1 | `item → modules/contentbox/models/menu/item/SubMenuItem.cfc` | 1 | medium | declares setMenuSlug() | `modules/contentbox/modules/contentbox-admin/interceptors/MenuCleanup.cfc:100` variable 'item' has no component ref |
| 1 | `item → modules/contentbox/models/menu/providers/ContentProvider.cfc` | 7 | low | declares getDisplayTemplate() (7 candidates) | `modules/contentbox/models/system/CBHelper.cfc:2404` variable 'item' has no component ref |
| 1 | `item → modules/contentbox/models/security/PermissionGroup.cfc` | 27 | low | declares getName() (19 candidates) | `modules/contentbox/models/security/Author.cfc:430` variable 'item' has no component ref |
| 1 | `item → modules/contentbox/models/system/Site.cfc` | 5 | medium | declares getSlug(); named like the receiver 'Site' (5 candidates) | `modules/contentbox/models/system/SettingService.cfc:583` variable 'item' has no component ref |
| 1 | `manifest` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/helpers/Mixins.cfm:42` variable 'manifest' has no component ref |
| 1 | `md` | 0 | none |  | `modules/contentbox/models/BaseEntityMethods.cfc:84` variable 'md' has no component ref |
| 1 | `migrations` | 0 | none |  | `modules/contentbox/migrations/2020_01_01_150933_init.cfc:53` variable 'migrations' has no component ref |
| 1 | `msgstruct.message` | 0 | none |  | `modules/contentbox/models/ui/templates/messagebox.cfm:41` variable 'msgStruct.message' has no component ref |
| 1 | `nullparentfields` | 0 | none |  | `modules/contentbox/models/content/ContentService.cfc:499` variable 'nullParentFields' has no component ref |
| 1 | `ocategory → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/modules/contentbox-admin/handlers/categories.cfc:114` variable 'oCategory' has no component ref |
| 1 | `oconfig → modules/contentbox/modules_user/Hello/ModuleConfig.cfc` | 2 | low | declares onActivate() (2 candidates) | `modules/contentbox/models/modules/ModuleService.cfc:286` variable 'oConfig' has no component ref |
| 1 | `originalservice → modules/contentbox/models/content/BaseContent.cfc` | 3 | low | declares setParent() (2 candidates) | `modules/contentbox/models/content/BaseContent.cfc:1419` variable 'originalService' has no component ref |
| 1 | `otheme → modules/contentbox/themes/default/Theme.cfc` | 1 | high | declares onActivation(); named like the receiver 'oTheme' | `modules/contentbox/models/ui/ThemeService.cfc:270` variable 'oTheme' has no component ref |
| 1 | `page → modules/contentbox/models/comments/Comment.cfc` | 4 | low | declares getContent() (4 candidates) | `modules/contentbox/models/system/NotificationService.cfc:320` variable 'page' has no component ref |
| 1 | `page → modules/contentbox/models/content/BaseContent.cfc` | 3 | low | declares renderContent() (2 candidates) | `modules/contentbox/widgets/PageInclude.cfc:39` variable 'page' has no component ref |
| 1 | `pages[] → modules/contentbox/models/comments/Comment.cfc` | 6 | low | declares getAuthor() (6 candidates) | `tests/specs/contentbox-web/contentbox/unit/content/PageServiceTest.cfc:45` variable 'pages[]' has no component ref |
| 1 | `pages[] → modules/contentbox/models/security/Author.cfc` | 1 | high | declares getAuthorID(); named like the receiver 'Author' | `tests/specs/contentbox-web/contentbox/unit/content/PageServiceTest.cfc:45` variable 'pages[]' has no component ref |
| 1 | `perms` | 1 | none |  | `modules/contentbox/migrations/init/seed_permissions.cfc:188` variable 'perms' has no component ref |
| 1 | `prc` | 0 | none |  | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseContentHandler.cfc:140` variable 'prc' has no component ref |
| 1 | `prc → modules/contentbox/models/security/Author.cfc` | 3 | low | declares hasPermission() (3 candidates) | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseContentHandler.cfc:140` variable 'prc' has no component ref |
| 1 | `prc.activedisk → modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Editor.cfc` | 3 | medium | declares info() | `modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Home.cfc:459` variable 'prc.activeDisk' has no component ref |
| 1 | `prc.activedisk → modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Home.cfc` | 1 | medium | declares download() | `modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Home.cfc:351` variable 'prc.activeDisk' has no component ref |
| 1 | `prc.activetheme.settings` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/themes/active.cfm:101` variable 'prc.activeTheme.settings' has no component ref |
| 1 | `prc.categories[] → modules/contentbox/models/content/Category.cfc` | 1 | medium | declares getCategoryID() | `modules/contentbox/modules/contentbox-admin/views/_components/editor/sidebar/Categories.cfm:20` variable 'prc.categories[]' has no component ref |
| 1 | `prc.owidget → modules/contentbox/models/ui/BaseWidget.cfc` | 1 | medium | declares getPublicMethods() | `modules/contentbox/modules/contentbox-admin/handlers/widgets.cfc:52` variable 'prc.oWidget' has no component ref |
| 1 | `prc.page → modules/contentbox/models/BaseEntityMethods.cfc` | 2 | low | declares isLoaded() (2 candidates) | `modules/contentbox/models/system/CBHelper.cfc:2457` variable 'prc.page' has no component ref |
| 1 | `prc.page → modules/contentbox/models/content/Page.cfc` | 2 | medium | declares isHomePage(); named like the receiver 'page' (2 candidates) | `modules/contentbox/models/system/CBHelper.cfc:753` variable 'prc.page' has no component ref |
| 1 | `prc.provider → modules/contentbox/models/BaseEntityMethods.cfc` | 5 | low | declares getEntityName() (3 candidates) | `modules/contentbox/modules/contentbox-admin/handlers/menus.cfc:145` variable 'prc.provider' has no component ref |
| 1 | `prc.site → modules/contentbox/models/system/Site.cfc` | 1 | high | declares getDomainAliasesAsJSON(); named like the receiver 'site' | `modules/contentbox/modules/contentbox-admin/views/sites/editorHelper.cfm:12` variable 'prc.site' has no component ref |
| 1 | `prc.themes` | 2 | none |  | `modules/contentbox/modules/contentbox-admin/views/tools/exporter.cfm:342` variable 'prc.themes' has no component ref |
| 1 | `prc.widgets` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/widgets/widgetList.cfm:95` variable 'prc.widgets' has no component ref |
| 1 | `qsettingcolumns` | 0 | none |  | `modules/contentbox/migrations/util/MigrationUtils.cfm:44` variable 'qSettingColumns' has no component ref |
| 1 | `r → modules/contentbox/models/content/BaseContent.cfc` | 7 | low | declares getSite() (7 candidates) | `tests/specs/contentbox-web/contentbox/unit/content/RelocationTest.cfc:53` variable 'r' has no component ref |
| 1 | `r → modules/contentbox/models/system/Site.cfc` | 5 | medium | declares getSiteID(); named like the receiver 'Site' (5 candidates) | `tests/specs/contentbox-web/contentbox/unit/content/RelocationTest.cfc:53` variable 'r' has no component ref |
| 1 | `rc.comment → modules/contentbox/models/content/BaseContent.cfc` | 4 | low | declares getTitle() (4 candidates) | `modules/contentbox/modules/contentbox-admin/views/comments/moderate.cfm:44` variable 'rc.comment' has no component ref |
| 1 | `rc.customfields` | 1 | none |  | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseContentHandler.cfc:190` variable 'rc.customFields' has no component ref |
| 1 | `rc.relocationurl` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/modules/contentbox-security/handlers/twofactorEnrollment.cfc:48` variable 'rc.relocationURL' has no component ref |
| 1 | `response.relocations` | 0 | none |  | `tests/specs/contentbox-web/contentbox/unit/content/RelocationTest.cfc:51` variable 'response.relocations' has no component ref |
| 1 | `results.author → modules/contentbox/models/security/Author.cfc` | 2 | medium | declares getPassword(); named like the receiver 'author' (2 candidates) | `modules/contentbox/modules/contentbox-admin/modules/contentbox-security/handlers/security.cfc:307` variable 'results.author' has no component ref |
| 1 | `results.categories` | 1 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/categories.cfc:66` variable 'results.categories' has no component ref |
| 1 | `results.exportlog` | 3 | none |  | `modules/contentbox/models/exporters/StaticExporter.cfc:75` variable 'results.exportLog' has no component ref |
| 1 | `routeresults.route.action` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/modules.cfc:73` variable 'routeResults.route.action' has no component ref |
| 1 | `schema[]` | 0 | none |  | `modules/contentbox/models/content/ContentTemplateService.cfc:179` variable 'schema[]' has no component ref |
| 1 | `schema[].excludetypes` | 0 | none |  | `modules/contentbox/models/content/ContentTemplateService.cfc:180` variable 'schema[].excludeTypes' has no component ref |
| 1 | `setting → modules/contentbox/models/system/Site.cfc` | 5 | medium | declares getSlug(); named like the receiver 'Site' (5 candidates) | `modules/contentbox/modules/contentbox-admin/views/settings/rawSettingsTable.cfm:54` variable 'setting' has no component ref |
| 1 | `setup → modules/contentbox-installer/models/Setup.cfc` | 1 | high | declares getUserData(); named like the receiver 'setup' | `modules/contentbox-installer/models/InstallerService.cfc:182` variable 'setup' has no component ref |
| 1 | `sitedata.categories` | 1 | none |  | `modules/contentbox/models/system/SiteService.cfc:501` variable 'siteData.categories' has no component ref |
| 1 | `sitedata.settings` | 1 | none |  | `modules/contentbox/models/system/SiteService.cfc:481` variable 'siteData.settings' has no component ref |
| 1 | `sitesettings[]` | 0 | none |  | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/siteSettings.cfc:33` variable 'siteSettings[]' has no component ref |
| 1 | `sortedarray` | 0 | none |  | `modules/contentbox/models/ui/editors/EditorService.cfc:105` variable 'sortedArray' has no component ref |
| 1 | `t → modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` | 27 | low | declares getName() (19 candidates) | `modules/contentbox/modules/contentbox-admin/handlers/contentTemplates.cfc:114` variable 't' has no component ref |
| 1 | `targetpath` | 0 | none |  | `tests/index.cfm:126` variable 'targetPath' has no component ref |
| 1 | `template` | 0 | none |  | `modules/contentbox/models/content/BaseContent.cfc:1820` variable 'template' has no component ref |
| 1 | `template → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/modules/contentbox-admin/handlers/contentTemplates.cfc:65` variable 'template' has no component ref |
| 1 | `templateproperties` | 0 | none |  | `modules/contentbox/models/content/ContentTemplateService.cfc:184` variable 'templateProperties' has no component ref |
| 1 | `templatservice → modules/contentbox/modules/contentbox-admin/handlers/authors.cfc` | 3 | medium | declares new() | `modules/contentbox/modules/contentbox-admin/handlers/contentTemplates.cfc:92` variable 'templatService' has no component ref |
| 1 | `testsite → modules/contentbox/models/content/BaseContent.cfc` | 5 | low | declares getSiteID() (5 candidates) | `tests/specs/contentbox-api/SitesSpec.cfc:46` variable 'testSite' has no component ref |
| 1 | `themerecord.widgets` | 0 | none |  | `modules/contentbox/models/ui/ThemeService.cfc:230` variable 'themeRecord.widgets' has no component ref |
| 1 | `this.memento.defaultexcludes` | 3 | none |  | `modules/contentbox/models/BaseEntityMethods.cfc:67` variable 'this.memento.defaultExcludes' has no component ref |
| 1 | `this.memento.defaultincludes` | 3 | none |  | `modules/contentbox/models/BaseEntityMethods.cfc:60` variable 'this.memento.defaultIncludes' has no component ref |
| 1 | `this.memento.profiles.export.defaultincludes` | 3 | none |  | `modules/contentbox/models/system/Site.cfc:408` variable 'this.memento.profiles.export.defaultIncludes' has no component ref |
| 1 | `this.memento.profiles[][]` | 3 | none |  | `modules/contentbox/models/BaseEntityMethods.cfc:123` variable 'this.memento.profiles[][]' has no component ref |
| 1 | `thiscategory → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/modules/contentbox-admin/handlers/categories.cfc:68` variable 'thisCategory' has no component ref |
| 1 | `thiscontent.comments` | 1 | none |  | `modules/contentbox/models/content/ContentService.cfc:1202` variable 'thisContent.comments' has no component ref |
| 1 | `thiscontent.contentversions` | 1 | none |  | `modules/contentbox/models/content/ContentService.cfc:1266` variable 'thisContent.contentVersions' has no component ref |
| 1 | `thiscontent.parent` | 0 | none |  | `modules/contentbox/models/content/ContentService.cfc:1053` variable 'thisContent.parent' has no component ref |
| 1 | `typeservice` | 10 | none |  | `modules/contentbox/modules/contentbox-ui/handlers/page.cfc:73` variable 'typeService' has no component ref |
| 1 | `typeservice → modules/contentbox/modules/contentbox-admin/handlers/authors.cfc` | 3 | medium | declares new() | `modules/contentbox/modules/contentbox-ui/handlers/page.cfc:48` variable 'typeService' has no component ref |
| 1 | `url.path` | 0 | none |  | `tests/index.cfm:41` variable 'url.path' has no component ref |
| 1 | `variables.activecontent → modules/contentbox/models/content/BaseContent.cfc` | 4 | low | declares getContent() (4 candidates) | `modules/contentbox/models/content/BaseContent.cfc:1148` variable 'variables.activeContent' has no component ref |
| 1 | `variables.blogroutes` | 0 | none |  | `modules/contentbox/modules/contentbox-ui/ModuleConfig.cfc:355` variable 'variables.blogRoutes' has no component ref |
| 1 | `variables.categories` | 7 | none |  | `modules/contentbox/models/content/BaseContent.cfc:1089` variable 'variables.categories' has no component ref |
| 1 | `variables.cbcsrf → modules/contentbox/models/security/LoginTrackerService.cfc` | 1 | medium | declares rotate() | `modules/contentbox/models/security/SecurityService.cfc:137` variable 'variables.cbCSRF' has no component ref |
| 1 | `variables.contentversions[] → modules/contentbox/models/content/ContentVersion.cfc` | 8 | low | declares getVersion() (4 candidates) | `modules/contentbox/models/content/BaseContent.cfc:847` variable 'variables.contentVersions[]' has no component ref |
| 1 | `variables.editors` | 0 | none |  | `modules/contentbox/models/ui/editors/EditorService.cfc:104` variable 'variables.editors' has no component ref |
| 1 | `variables.editors[] → modules/contentbox/models/ui/editors/IEditor.cfc` | 8 | low | declares getDisplayName() (8 candidates) | `modules/contentbox/models/ui/editors/EditorService.cfc:128` variable 'variables.editors[]' has no component ref |
| 1 | `variables.headermenumap` | 0 | none |  | `modules/contentbox/models/ui/AdminMenuService.cfc:896` variable 'variables.headerMenuMap' has no component ref |
| 1 | `variables.logger` | 3 | none |  | `modules/contentbox/models/security/AuthorService.cfc:205` variable 'variables.logger' has no component ref |
| 1 | `variables.menuitems` | 7 | none |  | `modules/contentbox/models/menu/Menu.cfc:198` variable 'variables.menuItems' has no component ref |
| 1 | `variables.menus` | 7 | none |  | `modules/contentbox/models/system/Site.cfc:461` variable 'variables.menus' has no component ref |
| 1 | `variables.modulemap` | 0 | none |  | `modules/contentbox/models/modules/ModuleService.cfc:480` variable 'variables.moduleMap' has no component ref |
| 1 | `variables.newpermissions` | 0 | none |  | `modules/contentbox/migrations/2020_08_24_150933_v5Upgrade.cfc:1009` variable 'variables.newPermissions' has no component ref |
| 1 | `variables.ormservice` | 3 | none |  | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseHandler.cfc:248` variable 'variables.ormService' has no component ref |
| 1 | `variables.ormservice → modules/contentbox/modules/contentbox-admin/handlers/authors.cfc` | 3 | medium | declares new() | `modules/contentbox/models/menu/Menu.cfc:261` variable 'variables.ORMService' has no component ref |
| 1 | `variables.permissions` | 7 | none |  | `modules/contentbox/models/security/Author.cfc:388` variable 'variables.permissions' has no component ref |
| 1 | `variables.providers[] → modules/contentbox/models/security/twofactor/ITwoFactorProvider.cfc` | 8 | low | declares getDisplayName() (8 candidates) | `modules/contentbox/models/security/twofactor/TwoFactorService.cfc:114` variable 'variables.providers[]' has no component ref |
| 1 | `variables.routes` | 0 | none |  | `modules/contentbox/modules/contentbox-ui/ModuleConfig.cfc:312` variable 'variables.routes' has no component ref |
| 1 | `variables.siteservice` | 2 | none |  | `tests/resources/BaseApiTest.cfc:164` variable 'variables.siteService' has no component ref |
| 1 | `w` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/tools/exporter.cfm:409` variable 'w' has no component ref |
| 1 | `ziptemp` | 0 | none |  | `modules/contentbox/models/util/ZipUtil.cfc:253` variable 'zipTemp' has no component ref |

<details><summary>Groups with several candidates</summary>

- `arguments.site → modules/contentbox/models/system/Site.cfc` — 18 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/system/Site.cfc` — declares getsiteID(); named like the receiver 'site'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getsiteID()
  - low `modules/contentbox/models/content/Category.cfc` — declares getsiteID()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getsiteID()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getsiteID()
- `arguments.page → modules/contentbox/models/content/BaseContent.cfc` — 14 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
- `comment → modules/contentbox/models/comments/Comment.cfc` — 13 finding(s), 3 candidate(s):
  - medium `modules/contentbox/models/comments/Comment.cfc` — declares getAuthorEmail(), getRelatedContent(); named like the receiver 'comment'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getAuthorEmail(), getRelatedContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getAuthorEmail(), getRelatedContent()
- `thismodule → modules/contentbox/models/modules/Module.cfc` — 10 finding(s), 19 candidate(s):
  - low `modules/contentbox/models/modules/Module.cfc` — declares getName()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares getName()
  - low `modules/contentbox/models/exporters/BaseExporter.cfc` — declares getName()
  - low `modules/contentbox/models/media/BaseProvider.cfc` — declares getName()
  - low `modules/contentbox/models/security/PermissionGroup.cfc` — declares getName()
  - low `modules/contentbox/models/security/Role.cfc` — declares getName()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getName()
  - low `modules/contentbox/models/system/Site.cfc` — declares getName()
  - … 11 more
- `arguments.category → modules/contentbox/models/content/Category.cfc` — 8 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/content/Category.cfc` — declares getSlug(), getCategory(); named like the receiver 'category'
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug(), getCategory()
- `arguments.comment → modules/contentbox/models/comments/Comment.cfc` — 8 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/comments/Comment.cfc` — declares getRelatedContent(), getAuthorEmail(), getMemento(), getParentTitle(); named like the receiver 'comment'
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getRelatedContent(), getAuthorEmail(), getMemento(), getParentTitle()
- `owidget → modules/contentbox/models/ui/Widget.cfc` — 8 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/ui/Widget.cfc` — declares getAuthorURL(), getName(), getDescription(), getVersion(), getAuthor(); named like the receiver 'oWidget'
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares getAuthorURL(), getName(), getDescription(), getVersion(), getAuthor()
- `settingservice → modules/contentbox/models/system/SettingService.cfc` — 7 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/system/SettingService.cfc` — declares getAllSettings(); named like the receiver 'settingService'
  - low `modules/contentbox/models/security/twofactor/BaseTwoFactorProvider.cfc` — declares getAllSettings()
- `arguments.thiscontent → modules/contentbox/models/content/BaseContent.cfc` — 6 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares hasParent()
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares hasParent()
- `entry → modules/contentbox/models/content/Entry.cfc` — 5 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/content/Entry.cfc` — declares getSite(), getTitle(), hasExcerpt(), renderExcerpt(), renderContent(); named like the receiver 'entry'
  - low `modules/contentbox/models/content/Page.cfc` — declares getSite(), getTitle(), hasExcerpt(), renderExcerpt(), renderContent()
- `instance.iooutput → modules/contentbox/models/util/MobileDetector.cfc` — 5 finding(s), 169 candidate(s):
  - low `modules/contentbox/models/util/MobileDetector.cfc` — declares init()
  - low `modules/contentbox/models/util/ZipUtil.cfc` — declares init()
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares init()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares init()
  - low `modules/contentbox/models/comments/CommentService.cfc` — declares init()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares init()
  - low `modules/contentbox/models/content/Category.cfc` — declares init()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares init()
  - … 161 more
- `page → modules/contentbox/models/content/Page.cfc` — 5 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/content/Page.cfc` — declares getSite(), getTitle(), hasExcerpt(), renderExcerpt(), renderContent(); named like the receiver 'page'
  - low `modules/contentbox/models/content/Entry.cfc` — declares getSite(), getTitle(), hasExcerpt(), renderExcerpt(), renderContent()
- `arguments.categories[] → modules/contentbox/models/content/Category.cfc` — 4 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/Category.cfc` — declares getCategory(), getNumberOfEntries()
  - low `modules/contentbox/models/system/Site.cfc` — declares getCategory(), getNumberOfEntries()
- `arguments.thissite → modules/contentbox/models/content/BaseContent.cfc` — 4 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
- `instance.iofile → modules/contentbox/models/util/MobileDetector.cfc` — 4 finding(s), 169 candidate(s):
  - low `modules/contentbox/models/util/MobileDetector.cfc` — declares init()
  - low `modules/contentbox/models/util/ZipUtil.cfc` — declares init()
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares init()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares init()
  - low `modules/contentbox/models/comments/CommentService.cfc` — declares init()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares init()
  - low `modules/contentbox/models/content/Category.cfc` — declares init()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares init()
  - … 161 more
- `prc.owidget → modules/contentbox/models/modules/Module.cfc` — 4 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/modules/Module.cfc` — declares getName(), getVersion(), getForgeBoxSlug(), getDescription()
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares getName(), getVersion(), getForgeBoxSlug(), getDescription()
- `arguments.entry → modules/contentbox/models/system/Site.cfc` — 3 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
- `arguments.page → modules/contentbox/models/system/Site.cfc` — 3 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
- `arguments.thischild → modules/contentbox/models/content/BaseContent.cfc` — 3 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getTitle(), getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getTitle(), getSlug()
- `commentresults.comments[] → modules/contentbox/models/comments/Comment.cfc` — 3 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/comments/Comment.cfc` — declares getParentTitle()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getParentTitle()
- `instance.ioinput → modules/contentbox/models/util/MobileDetector.cfc` — 3 finding(s), 169 candidate(s):
  - low `modules/contentbox/models/util/MobileDetector.cfc` — declares init()
  - low `modules/contentbox/models/util/ZipUtil.cfc` — declares init()
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares init()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares init()
  - low `modules/contentbox/models/comments/CommentService.cfc` — declares init()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares init()
  - low `modules/contentbox/models/content/Category.cfc` — declares init()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares init()
  - … 161 more
- `menu → modules/contentbox/models/menu/Menu.cfc` — 3 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/menu/Menu.cfc` — declares getSlug(), getTitle(); named like the receiver 'menu'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug(), getTitle()
- `prc.activedisk → modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/authors.cfc` — 3 finding(s), 11 candidate(s):
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/authors.cfc` — declares create()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseHandler.cfc` — declares create()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/categories.cfc` — declares create()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/comments.cfc` — declares create()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/contentStore.cfc` — declares create()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/contentTemplates.cfc` — declares create()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/entries.cfc` — declares create()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/menus.cfc` — declares create()
  - … 3 more
- `prc.oeditordriver → modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — 3 finding(s), 4 candidate(s):
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — declares startup(), shutdown(), loadAssets()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-markdowneditor/models/MarkdownEditor.cfc` — declares startup(), shutdown(), loadAssets()
  - low `modules/contentbox/models/ui/editors/IEditor.cfc` — declares startup(), shutdown(), loadAssets()
  - low `tests/specs/contentbox-web/contentbox/unit/ui/editors/MockEditor.cfc` — declares startup(), shutdown(), loadAssets()
- `thispage → modules/contentbox/models/content/BaseContent.cfc` — 3 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug(), setSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug(), setSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug(), setSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug(), setSlug()
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug(), setSlug()
- `arguments.appscope.cbbootstrap → Application.cfc` — 2 finding(s), 2 candidate(s):
  - low `Application.cfc` — declares onSessionEnd()
  - low `handlers/Main.cfc` — declares onSessionEnd()
- `arguments.data.menu → modules/contentbox/models/menu/Menu.cfc` — 2 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/menu/Menu.cfc` — declares getSlug(); named like the receiver 'menu'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
- `arguments.menuitem → modules/contentbox/models/menu/Menu.cfc` — 2 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/menu/Menu.cfc` — declares getSiteID(); named like the receiver 'Menu'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSiteID()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSiteID()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getSiteID()
  - low `modules/contentbox/models/system/Site.cfc` — declares getSiteID()
- `arguments.original → modules/contentbox/models/content/Entry.cfc` — 2 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/Entry.cfc` — declares hasExcerpt(), getExcerpt()
  - low `modules/contentbox/models/content/Page.cfc` — declares hasExcerpt(), getExcerpt()
- `arguments.originalservice → modules/contentbox/models/content/CategoryService.cfc` — 2 finding(s), 23 candidate(s):
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares save()
  - low `modules/contentbox/models/content/ContentStoreService.cfc` — declares save()
  - low `modules/contentbox/models/content/EntryService.cfc` — declares save()
  - low `modules/contentbox/models/content/PageService.cfc` — declares save()
  - low `modules/contentbox/models/menu/MenuService.cfc` — declares save()
  - low `modules/contentbox/models/security/AuthorService.cfc` — declares save()
  - low `modules/contentbox/models/system/SiteService.cfc` — declares save()
  - low `modules/contentbox/modules/contentbox-admin/handlers/authors.cfc` — declares save()
  - … 15 more
- `arguments.relatedcontent[] → modules/contentbox/models/content/BaseContent.cfc` — 2 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getTitle()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getTitle()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getTitle()
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares getTitle()
- `arguments.target → modules/contentbox/models/system/Site.cfc` — 2 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/system/Site.cfc` — declares getSiteID(); named like the receiver 'Site'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSiteID()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSiteID()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSiteID()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getSiteID()
- `cats[] → modules/contentbox/models/system/Site.cfc` — 2 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/system/Site.cfc` — declares getCategory()
  - low `modules/contentbox/models/content/Category.cfc` — declares getCategory()
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares getCategory()
  - low `modules/contentbox/models/ui/Widget.cfc` — declares getCategory()
- `content → modules/contentbox/models/comments/Comment.cfc` — 2 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/comments/Comment.cfc` — declares getContent()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getContent()
  - low `modules/contentbox/models/exporters/DataExporter.cfc` — declares getContent()
- `e → modules/contentbox/models/system/Site.cfc` — 2 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/system/Site.cfc` — declares hasCategories(), getCategories()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares hasCategories(), getCategories()
- `incomment → modules/contentbox/models/content/BaseContent.cfc` — 2 finding(s), 3 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSiteSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSiteSlug()
  - low `modules/contentbox-installer/models/Setup.cfc` — declares getSiteSlug()
- `instance.iofile → modules/contentbox/models/content/CategoryService.cfc` — 2 finding(s), 17 candidate(s):
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares delete()
  - low `modules/contentbox/models/content/ContentService.cfc` — declares delete()
  - low `modules/contentbox/models/content/ContentTemplateService.cfc` — declares delete()
  - low `modules/contentbox/models/security/AuthorService.cfc` — declares delete()
  - low `modules/contentbox/models/system/SiteService.cfc` — declares delete()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/authors.cfc` — declares delete()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/baseHandler.cfc` — declares delete()
  - low `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/categories.cfc` — declares delete()
  - … 9 more
- `instance.zipentry → modules/contentbox/models/util/MobileDetector.cfc` — 2 finding(s), 169 candidate(s):
  - low `modules/contentbox/models/util/MobileDetector.cfc` — declares init()
  - low `modules/contentbox/models/util/ZipUtil.cfc` — declares init()
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares init()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares init()
  - low `modules/contentbox/models/comments/CommentService.cfc` — declares init()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares init()
  - low `modules/contentbox/models/content/Category.cfc` — declares init()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares init()
  - … 161 more
- `ocurrentcontent → modules/contentbox/models/system/Site.cfc` — 2 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/system/Site.cfc` — declares getslug()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getslug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getslug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getslug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getslug()
- `opermission → modules/contentbox/models/BaseEntityMethods.cfc` — 2 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares isLoaded()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares isLoaded()
- `orule → modules/contentbox/models/BaseEntityMethods.cfc` — 2 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares isLoaded()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares isLoaded()
- `osite → modules/contentbox/models/BaseEntityMethods.cfc` — 2 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares isLoaded()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares isLoaded()
- `pageresults.content[] → modules/contentbox/models/content/BaseContent.cfc` — 2 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares hasChild()
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares hasChild()
- `prc.activedisk → modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — 2 finding(s), 19 candidate(s):
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — declares getName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-markdowneditor/models/MarkdownEditor.cfc` — declares getName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-security/modules/contentbox-email-twofactor/models/EmailTwoFactorProvider.cfc` — declares getName()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares getName()
  - low `modules/contentbox/models/exporters/BaseExporter.cfc` — declares getName()
  - low `modules/contentbox/models/media/BaseProvider.cfc` — declares getName()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getName()
  - low `modules/contentbox/models/security/PermissionGroup.cfc` — declares getName()
  - … 11 more
- `prc.opaging → modules/contentbox/models/ui/Paging.cfc` — 2 finding(s), 25 candidate(s):
  - medium `modules/contentbox/models/ui/Paging.cfc` — declares renderit(); named like the receiver 'oPaging'
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares renderit()
  - low `modules/contentbox/widgets/Archives.cfc` — declares renderit()
  - low `modules/contentbox/widgets/Categories.cfc` — declares renderit()
  - low `modules/contentbox/widgets/CommentForm.cfc` — declares renderit()
  - low `modules/contentbox/widgets/ContentStore.cfc` — declares renderit()
  - low `modules/contentbox/widgets/EntryInclude.cfc` — declares renderit()
  - low `modules/contentbox/widgets/Menu.cfc` — declares renderit()
  - … 17 more
- `prc.widget.widget → modules/contentbox/models/ui/Widget.cfc` — 2 finding(s), 19 candidate(s):
  - medium `modules/contentbox/models/ui/Widget.cfc` — declares getName(); named like the receiver 'widget'
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — declares getName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-markdowneditor/models/MarkdownEditor.cfc` — declares getName()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares getName()
  - low `modules/contentbox/models/exporters/BaseExporter.cfc` — declares getName()
  - low `modules/contentbox/models/media/BaseProvider.cfc` — declares getName()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getName()
  - low `modules/contentbox/models/security/PermissionGroup.cfc` — declares getName()
  - … 11 more
- `targetsite → modules/contentbox/models/system/Site.cfc` — 2 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
- `thisrole → modules/contentbox/models/security/Author.cfc` — 2 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/security/Author.cfc` — declares getRole()
  - low `modules/contentbox/models/security/Role.cfc` — declares getRole()
- `variables.settingservice → modules/contentbox/models/system/SettingService.cfc` — 2 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/system/SettingService.cfc` — declares getAllSettings(); named like the receiver 'settingService'
  - low `modules/contentbox/models/security/twofactor/BaseTwoFactorProvider.cfc` — declares getAllSettings()
- `variables[] → modules/contentbox/models/content/CategoryService.cfc` — 2 finding(s), 14 candidate(s):
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares getAllForExport()
  - low `modules/contentbox/models/content/ContentService.cfc` — declares getAllForExport()
  - low `modules/contentbox/models/content/ContentStoreService.cfc` — declares getAllForExport()
  - low `modules/contentbox/models/content/ContentTemplateService.cfc` — declares getAllForExport()
  - low `modules/contentbox/models/content/EntryService.cfc` — declares getAllForExport()
  - low `modules/contentbox/models/content/PageService.cfc` — declares getAllForExport()
  - low `modules/contentbox/models/menu/MenuService.cfc` — declares getAllForExport()
  - low `modules/contentbox/models/security/AuthorService.cfc` — declares getAllForExport()
  - … 6 more
- `version → modules/contentbox/models/content/ContentVersion.cfc` — 2 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getIsActive(), setIsActive()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getIsActive(), setIsActive()
  - low `modules/contentbox/models/security/Author.cfc` — declares getIsActive(), setIsActive()
  - low `modules/contentbox/models/system/Site.cfc` — declares getIsActive(), setIsActive()
- `widget → modules/contentbox/models/ui/Widget.cfc` — 2 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/ui/Widget.cfc` — declares getIcon(); named like the receiver 'widget'
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares getIcon()
- `activecontent → modules/contentbox/models/security/Author.cfc` — 1 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/security/Author.cfc` — declares getInfoSnapshot(); named like the receiver 'Author'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getInfoSnapshot()
  - low `modules/contentbox/models/subscriptions/Subscriber.cfc` — declares getInfoSnapshot()
  - low `modules/contentbox/models/system/Site.cfc` — declares getInfoSnapshot()
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares getInfoSnapshot()
- `arguments.comment → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getContentType()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares getContentType()
- `arguments.data.comment → modules/contentbox/models/comments/Comment.cfc` — 1 finding(s), 7 candidate(s):
  - medium `modules/contentbox/models/comments/Comment.cfc` — declares getRelatedContent(); named like the receiver 'comment'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getRelatedContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getRelatedContent()
  - low `modules/contentbox/models/content/CustomField.cfc` — declares getRelatedContent()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getRelatedContent()
  - low `modules/contentbox/models/content/Stats.cfc` — declares getRelatedContent()
  - low `modules/contentbox/models/subscriptions/CommentSubscription.cfc` — declares getRelatedContent()
- `arguments.data.comment → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
- `arguments.target → modules/contentbox/models/content/CategoryService.cfc` — 1 finding(s), 3 candidate(s):
  - medium `modules/contentbox/models/content/CategoryService.cfc` — declares isSlugUnique(); named like the receiver 'CategoryService'
  - low `modules/contentbox/models/content/ContentService.cfc` — declares isSlugUnique()
  - low `modules/contentbox/models/menu/MenuService.cfc` — declares isSlugUnique()
- `arguments.target → modules/contentbox/models/content/ContentService.cfc` — 1 finding(s), 3 candidate(s):
  - medium `modules/contentbox/models/content/ContentService.cfc` — declares isSlugUnique(); named like the receiver 'ContentService'
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares isSlugUnique()
  - low `modules/contentbox/models/menu/MenuService.cfc` — declares isSlugUnique()
- `arguments.thispage → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares hasParent()
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares hasParent()
- `child → modules/contentbox/models/menu/item/BaseMenuItem.cfc` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares setParent()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares setParent()
- `config → modules/contentbox/modules_user/Hello/ModuleConfig.cfc` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/modules_user/Hello/ModuleConfig.cfc` — declares onDeactivate()
  - low `modules/contentbox/modules_user/Hello/modules/hello-child/ModuleConfig.cfc` — declares onDeactivate()
- `entries[] → modules/contentbox/models/comments/Comment.cfc` — 1 finding(s), 6 candidate(s):
  - low `modules/contentbox/models/comments/Comment.cfc` — declares getAuthor()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getAuthor()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getAuthor()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getAuthor()
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares getAuthor()
  - low `modules/contentbox/models/ui/Widget.cfc` — declares getAuthor()
- `entry → modules/contentbox/models/comments/Comment.cfc` — 1 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/comments/Comment.cfc` — declares getContent()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getContent()
  - low `modules/contentbox/models/exporters/DataExporter.cfc` — declares getContent()
- `entryresults.content[] → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getTitle()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getTitle()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getTitle()
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares getTitle()
- `importer → modules/contentbox/modules/contentbox-admin/handlers/modules.cfc` — 1 finding(s), 8 candidate(s):
  - low `modules/contentbox/modules/contentbox-admin/handlers/modules.cfc` — declares execute()
  - low `modules/contentbox/models/importers/BlogCFCImporter.cfc` — declares execute()
  - low `modules/contentbox/models/importers/ContentBoxImporter.cfc` — declares execute()
  - low `modules/contentbox/models/importers/ICBImporter.cfc` — declares execute()
  - low `modules/contentbox/models/importers/MachBlogImporter.cfc` — declares execute()
  - low `modules/contentbox/models/importers/MangoImporter.cfc` — declares execute()
  - low `modules/contentbox/models/importers/WordpressImporter.cfc` — declares execute()
  - low `modules/contentbox-installer/models/InstallerService.cfc` — declares execute()
- `instance.gzinput → modules/contentbox/models/util/MobileDetector.cfc` — 1 finding(s), 169 candidate(s):
  - low `modules/contentbox/models/util/MobileDetector.cfc` — declares init()
  - low `modules/contentbox/models/util/ZipUtil.cfc` — declares init()
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares init()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares init()
  - low `modules/contentbox/models/comments/CommentService.cfc` — declares init()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares init()
  - low `modules/contentbox/models/content/Category.cfc` — declares init()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares init()
  - … 161 more
- `instance.gzoutput → modules/contentbox/models/util/MobileDetector.cfc` — 1 finding(s), 169 candidate(s):
  - low `modules/contentbox/models/util/MobileDetector.cfc` — declares init()
  - low `modules/contentbox/models/util/ZipUtil.cfc` — declares init()
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares init()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares init()
  - low `modules/contentbox/models/comments/CommentService.cfc` — declares init()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares init()
  - low `modules/contentbox/models/content/Category.cfc` — declares init()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares init()
  - … 161 more
- `instance.iobufoutput → modules/contentbox/models/util/MobileDetector.cfc` — 1 finding(s), 169 candidate(s):
  - low `modules/contentbox/models/util/MobileDetector.cfc` — declares init()
  - low `modules/contentbox/models/util/ZipUtil.cfc` — declares init()
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares init()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares init()
  - low `modules/contentbox/models/comments/CommentService.cfc` — declares init()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares init()
  - low `modules/contentbox/models/content/Category.cfc` — declares init()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares init()
  - … 161 more
- `instance.objdate → modules/contentbox/models/util/MobileDetector.cfc` — 1 finding(s), 169 candidate(s):
  - low `modules/contentbox/models/util/MobileDetector.cfc` — declares init()
  - low `modules/contentbox/models/util/ZipUtil.cfc` — declares init()
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares init()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares init()
  - low `modules/contentbox/models/comments/CommentService.cfc` — declares init()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares init()
  - low `modules/contentbox/models/content/Category.cfc` — declares init()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares init()
  - … 161 more
- `item → modules/contentbox/models/menu/providers/ContentProvider.cfc` — 1 finding(s), 7 candidate(s):
  - low `modules/contentbox/models/menu/providers/ContentProvider.cfc` — declares getDisplayTemplate()
  - low `modules/contentbox/models/menu/providers/FreeProvider.cfc` — declares getDisplayTemplate()
  - low `modules/contentbox/models/menu/providers/IMenuItemProvider.cfc` — declares getDisplayTemplate()
  - low `modules/contentbox/models/menu/providers/JSProvider.cfc` — declares getDisplayTemplate()
  - low `modules/contentbox/models/menu/providers/MediaProvider.cfc` — declares getDisplayTemplate()
  - low `modules/contentbox/models/menu/providers/SubMenuProvider.cfc` — declares getDisplayTemplate()
  - low `modules/contentbox/models/menu/providers/URLProvider.cfc` — declares getDisplayTemplate()
- `item → modules/contentbox/models/security/PermissionGroup.cfc` — 1 finding(s), 19 candidate(s):
  - low `modules/contentbox/models/security/PermissionGroup.cfc` — declares getName()
  - low `modules/contentbox/models/security/Role.cfc` — declares getName()
  - low `modules/contentbox/models/security/twofactor/ITwoFactorProvider.cfc` — declares getName()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares getName()
  - low `modules/contentbox/models/exporters/BaseExporter.cfc` — declares getName()
  - low `modules/contentbox/models/media/BaseProvider.cfc` — declares getName()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getName()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getName()
  - … 11 more
- `item → modules/contentbox/models/system/Site.cfc` — 1 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/system/Site.cfc` — declares getSlug(); named like the receiver 'Site'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
- `oconfig → modules/contentbox/modules_user/Hello/ModuleConfig.cfc` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/modules_user/Hello/ModuleConfig.cfc` — declares onActivate()
  - low `modules/contentbox/modules_user/Hello/modules/hello-child/ModuleConfig.cfc` — declares onActivate()
- `originalservice → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares setParent()
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares setParent()
- `page → modules/contentbox/models/comments/Comment.cfc` — 1 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/comments/Comment.cfc` — declares getContent()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getContent()
  - low `modules/contentbox/models/exporters/DataExporter.cfc` — declares getContent()
- `page → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares renderContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares renderContent()
- `pages[] → modules/contentbox/models/comments/Comment.cfc` — 1 finding(s), 6 candidate(s):
  - low `modules/contentbox/models/comments/Comment.cfc` — declares getAuthor()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getAuthor()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getAuthor()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getAuthor()
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares getAuthor()
  - low `modules/contentbox/models/ui/Widget.cfc` — declares getAuthor()
- `prc → modules/contentbox/models/security/Author.cfc` — 1 finding(s), 3 candidate(s):
  - low `modules/contentbox/models/security/Author.cfc` — declares hasPermission()
  - low `modules/contentbox/models/security/PermissionGroup.cfc` — declares hasPermission()
  - low `modules/contentbox/models/security/Role.cfc` — declares hasPermission()
- `prc.page → modules/contentbox/models/BaseEntityMethods.cfc` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares isLoaded()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares isLoaded()
- `prc.page → modules/contentbox/models/content/Page.cfc` — 1 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/content/Page.cfc` — declares isHomePage(); named like the receiver 'page'
  - low `modules/contentbox/models/system/CBHelper.cfc` — declares isHomePage()
- `prc.provider → modules/contentbox/models/BaseEntityMethods.cfc` — 1 finding(s), 3 candidate(s):
  - low `modules/contentbox/models/BaseEntityMethods.cfc` — declares getEntityName()
  - low `modules/contentbox/models/menu/providers/BaseProvider.cfc` — declares getEntityName()
  - low `modules/contentbox/models/menu/providers/IMenuItemProvider.cfc` — declares getEntityName()
- `r → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 7 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSite()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSite()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares getSite()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSite()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSite()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getSite()
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares getSite()
- `r → modules/contentbox/models/system/Site.cfc` — 1 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/system/Site.cfc` — declares getSiteID(); named like the receiver 'Site'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSiteID()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSiteID()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSiteID()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getSiteID()
- `rc.comment → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getTitle()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getTitle()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getTitle()
  - low `modules/contentbox/models/menu/item/BaseMenuItem.cfc` — declares getTitle()
- `results.author → modules/contentbox/models/security/Author.cfc` — 1 finding(s), 2 candidate(s):
  - medium `modules/contentbox/models/security/Author.cfc` — declares getPassword(); named like the receiver 'author'
  - low `modules/contentbox-installer/models/Setup.cfc` — declares getPassword()
- `setting → modules/contentbox/models/system/Site.cfc` — 1 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/system/Site.cfc` — declares getSlug(); named like the receiver 'Site'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
- `t → modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — 1 finding(s), 19 candidate(s):
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — declares getName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-markdowneditor/models/MarkdownEditor.cfc` — declares getName()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares getName()
  - low `modules/contentbox/models/exporters/BaseExporter.cfc` — declares getName()
  - low `modules/contentbox/models/media/BaseProvider.cfc` — declares getName()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getName()
  - low `modules/contentbox/models/security/PermissionGroup.cfc` — declares getName()
  - low `modules/contentbox/models/security/Role.cfc` — declares getName()
  - … 11 more
- `testsite → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSiteID()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSiteID()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSiteID()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getSiteID()
  - low `modules/contentbox/models/system/Site.cfc` — declares getSiteID()
- `variables.activecontent → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getContent()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares getContent()
  - low `modules/contentbox/models/exporters/DataExporter.cfc` — declares getContent()
- `variables.contentversions[] → modules/contentbox/models/content/ContentVersion.cfc` — 1 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getVersion()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getVersion()
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares getVersion()
  - low `modules/contentbox/models/ui/Widget.cfc` — declares getVersion()
- `variables.editors[] → modules/contentbox/models/ui/editors/IEditor.cfc` — 1 finding(s), 8 candidate(s):
  - low `modules/contentbox/models/ui/editors/IEditor.cfc` — declares getDisplayName()
  - low `modules/contentbox/models/exporters/BaseExporter.cfc` — declares getDisplayName()
  - low `modules/contentbox/models/media/BaseProvider.cfc` — declares getDisplayName()
  - low `modules/contentbox/models/security/twofactor/ITwoFactorProvider.cfc` — declares getDisplayName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — declares getDisplayName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-markdowneditor/models/MarkdownEditor.cfc` — declares getDisplayName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-security/modules/contentbox-email-twofactor/models/EmailTwoFactorProvider.cfc` — declares getDisplayName()
  - low `tests/specs/contentbox-web/contentbox/unit/ui/editors/MockEditor.cfc` — declares getDisplayName()
- `variables.providers[] → modules/contentbox/models/security/twofactor/ITwoFactorProvider.cfc` — 1 finding(s), 8 candidate(s):
  - low `modules/contentbox/models/security/twofactor/ITwoFactorProvider.cfc` — declares getDisplayName()
  - low `modules/contentbox/models/exporters/BaseExporter.cfc` — declares getDisplayName()
  - low `modules/contentbox/models/media/BaseProvider.cfc` — declares getDisplayName()
  - low `modules/contentbox/models/ui/editors/IEditor.cfc` — declares getDisplayName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-ckeditor/models/CKEditor.cfc` — declares getDisplayName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-markdowneditor/models/MarkdownEditor.cfc` — declares getDisplayName()
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-security/modules/contentbox-email-twofactor/models/EmailTwoFactorProvider.cfc` — declares getDisplayName()
  - low `tests/specs/contentbox-web/contentbox/unit/ui/editors/MockEditor.cfc` — declares getDisplayName()

</details>

## Return types — a call chained on a method that declares no component

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 17 | `method 'site' has no component return type → modules/contentbox/models/system/Site.cfc` | 5 | medium | declares getsiteID(); named like the receiver 'site' (5 candidates) | `modules/contentbox/models/system/CBHelper.cfc:222` method 'site' has no component return type (chain to 'getsiteID') |
| 7 | `method 'getData' in coldbox.system.web.context.Response has no component return type` | 0 | none |  | `tests/specs/contentbox-api/AuthorsSpec.cfc:57` method 'getData' in coldbox.system.web.context.Response has no component return type (chain to 'each') |
| 7 | `method 'getSite' has no component return type → modules/contentbox/models/system/Site.cfc` | 5 | medium | declares getSiteId(); named like the receiver 'Site' (5 candidates) | `modules/contentbox/widgets/Categories.cfc:45` method 'getSite' has no component return type (chain to 'getSiteId') |
| 4 | `method 'getGrammar' in qb.models.Query.QueryBuilder has no component return type` | 0 | none |  | `modules/contentbox/migrations/2022_11_23_142439_v_6_0_0_Convert-Featured-cbfs.cfc:14` method 'getGrammar' in qb.models.Query.QueryBuilder has no component return type (chain to 'convertToBooleanType') |
| 4 | `method 'getInstance' in coldbox.system.ioc.Injector has no component return type → modules/contentbox/models/system/SettingService.cfc` | 11 | low | declares importFromData() (11 candidates) | `modules/contentbox/models/system/SiteService.cfc:534` method 'getInstance' in coldbox.system.ioc.Injector has no component return type (chain to 'importFromData') |
| 4 | `method 'site' in CBHelper@contentbox has no component return type → modules/contentbox/models/system/Site.cfc` | 1 | high | declares getMediaDisk(); named like the receiver 'site' | `modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Editor.cfc:15` method 'site' in CBHelper@contentbox has no component return type (chain to 'getMediaDisk') |
| 3 | `method 'getActiveContent' has no component return type → modules/contentbox/models/content/BaseContent.cfc` | 2 | low | declares getAuthorName() (2 candidates) | `modules/contentbox/models/content/BaseContent.cfc:1222` method 'getActiveContent' has no component return type (chain to 'getAuthorName') |
| 2 | `method 'getActiveContent' in ContentStore\|Entry\|Page has no component return type → modules/contentbox/models/content/ContentVersion.cfc` | 1 | medium | declares getChangelog() | `modules/contentbox/modules/contentbox-admin/views/content/quickLook.cfm:108` method 'getActiveContent' in ContentStore\|Entry\|Page has no component return type (chain to 'getChangelog') |
| 2 | `method 'getActiveContent' in contentbox.models.content.BaseContent\|contentbox.models.content.Entry\|contentbox.models.content.Page\|contentbox.models.conten…` | 8 | none |  | `modules/contentbox/modules/contentbox-admin/views/_components/editor/sidebar/InfoTable.cfm:56` method 'getActiveContent' in contentbox.models.content.BaseContent\|contentbox.models.content.Entry\|contentbox.models.content.Page\|contentbox.models.conten… |
| 2 | `method 'getContentType' in ContentStore\|Entry\|Page has no component return type` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/content/editorSelectorEntries.cfm:60` method 'getContentType' in ContentStore\|Entry\|Page has no component return type (chain to 'lcase') |
| 2 | `method 'getContentVersions' has no component return type` | 0 | none |  | `modules/contentbox/models/content/BaseContent.cfc:831` method 'getContentVersions' has no component return type (chain to 'each') |
| 2 | `method 'getCurrentEntry' has no component return type → modules/contentbox/models/comments/Comment.cfc` | 7 | low | declares hasRelatedContent(), getRelatedContent() (7 candidates) | `modules/contentbox/models/system/CBHelper.cfc:765` method 'getCurrentEntry' has no component return type (chain to 'hasRelatedContent') |
| 2 | `method 'getCurrentPage' has no component return type → modules/contentbox/models/comments/Comment.cfc` | 7 | low | declares hasRelatedContent(), getRelatedContent() (7 candidates) | `modules/contentbox/models/system/CBHelper.cfc:763` method 'getCurrentPage' has no component return type (chain to 'hasRelatedContent') |
| 2 | `method 'getCurrentPage' in contentbox.models.system.CBHelper has no component return type → modules/contentbox/models/content/BaseContent.cfc` | 5 | low | declares getSlug() (5 candidates) | `modules/contentbox/themes/default/views/_header.cfm:31` method 'getCurrentPage' in contentbox.models.system.CBHelper has no component return type (chain to 'getSlug') |
| 2 | `method 'getMemento' in coldbox.system.web.context.Response has no component return type` | 0 | none |  | `tests/specs/contentbox-api/authSpec.cfc:150` method 'getMemento' in coldbox.system.web.context.Response has no component return type (chain to 'toString') |
| 2 | `method 'list' in CriteriaBuilder has no component return type` | 1 | none |  | `modules/contentbox/models/content/ContentService.cfc:790` method 'list' in CriteriaBuilder has no component return type (chain to 'map') |
| 2 | `method 'populate' in contentStoreService@contentbox\|entryService@contentbox\|pageService@contentbox has no component return type → modules/contentbox/mode…` | 1 | medium | declares addJoinedPublishedtime() | `modules/contentbox/modules/contentbox-admin/handlers/baseContentHandler.cfc:433` method 'populate' in contentStoreService@contentbox\|entryService@contentbox\|pageService@contentbox has no component return type (chain to 'addJoinedPublish… |
| 2 | `method 'site' in CBHelper@contentBox has no component return type → modules/contentbox/models/system/Site.cfc` | 1 | high | declares getSiteRoot(); named like the receiver 'site' | `config/modules/cbfs.cfc:25` method 'site' in CBHelper@contentBox has no component return type (chain to 'getSiteRoot') |
| 2 | `method 'site' in contentbox.models.system.CBHelper has no component return type → modules/contentbox/models/BaseEntityMethods.cfc` | 1 | medium | declares getId() | `modules/contentbox/modules/contentbox-admin/views/contentTemplates/indexHelper.cfm:23` method 'site' in contentbox.models.system.CBHelper has no component return type (chain to 'getId') |
| 1 | `method 'executeQuery' has no component return type` | 1 | none |  | `modules/contentbox/models/content/ContentService.cfc:805` method 'executeQuery' has no component return type (chain to 'map') |
| 1 | `method 'getActiveContent' has no component return type → modules/contentbox/models/content/ContentVersion.cfc` | 4 | low | declares getIsActive() (4 candidates) | `modules/contentbox/models/content/BaseContent.cfc:1178` method 'getActiveContent' has no component return type (chain to 'getIsActive') |
| 1 | `method 'getActiveContent' in BaseContent has no component return type → modules/contentbox/models/BaseEntityMethods.cfc` | 1 | medium | declares getDisplayCreatedDate() | `modules/contentbox/modules/contentbox-ui/views/adminbar/index.cfm:63` method 'getActiveContent' in BaseContent has no component return type (chain to 'getDisplayCreatedDate') |
| 1 | `method 'getActiveContent' in ContentStore\|Entry\|Page has no component return type → modules/contentbox/models/BaseEntityMethods.cfc` | 1 | medium | declares getDisplayCreatedDate() | `modules/contentbox/modules/contentbox-admin/views/content/pager.cfm:54` method 'getActiveContent' in ContentStore\|Entry\|Page has no component return type (chain to 'getDisplayCreatedDate') |
| 1 | `method 'getActiveContent' in ContentStore\|Entry\|Page\|BaseContent has no component return type → modules/contentbox/models/BaseEntityMethods.cfc` | 1 | medium | declares getDisplayCreatedDate() | `modules/contentbox/modules/contentbox-admin/views/content/contentViewlet.cfm:91` method 'getActiveContent' in ContentStore\|Entry\|Page\|BaseContent has no component return type (chain to 'getDisplayCreatedDate') |
| 1 | `method 'getAllPreferences' in cbAuthor has no component return type` | 3 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/authors.cfc:518` method 'getAllPreferences' in cbAuthor has no component return type (chain to 'append') |
| 1 | `method 'getAllSettings' in SettingService@contentbox has no component return type` | 0 | none |  | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/settings.cfc:21` method 'getAllSettings' in SettingService@contentbox has no component return type (chain to 'filter') |
| 1 | `method 'getCache' has no component return type` | 4 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/dashboard.cfc:275` method 'getCache' has no component return type (chain to 'clearAll') |
| 1 | `method 'getCategories' has no component return type` | 1 | none |  | `modules/contentbox/models/content/BaseContent.cfc:1851` method 'getCategories' has no component return type (chain to 'map') |
| 1 | `method 'getCategories' in BaseContent has no component return type` | 1 | none |  | `modules/contentbox/models/content/ContentTemplateService.cfc:206` method 'getCategories' in BaseContent has no component return type (chain to 'map') |
| 1 | `method 'getContent' has no component return type` | 0 | none |  | `modules/contentbox/models/comments/Comment.cfc:131` method 'getContent' has no component return type (chain to 'reReplace') |
| 1 | `method 'getContentID' in ContentStore has no component return type` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/contentStore/indexTable.cfm:44` method 'getContentID' in ContentStore has no component return type (chain to 'replace') |
| 1 | `method 'getContentID' in Page has no component return type` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/pages/indexTable.cfm:53` method 'getContentID' in Page has no component return type (chain to 'replace') |
| 1 | `method 'getContentTemplate' in contentbox.models.content.BaseContent\|contentbox.models.content.Entry\|contentbox.models.content.Page\|contentbox.models.cont…` | 1 | high | declares getTemplateID(); named like the receiver 'ContentTemplate' | `modules/contentbox/modules/contentbox-admin/views/_components/editor/sidebar/Modifiers.cfm:33` method 'getContentTemplate' in contentbox.models.content.BaseContent\|contentbox.models.content.Entry\|contentbox.models.content.Page\|contentbox.models.cont… |
| 1 | `method 'getContentTemplate' in contentbox.models.content.BaseContent\|contentbox.models.content.Entry\|contentbox.models.content.Page\|contentbox.models.cont…` | 10 | medium | declares getMemento() | `modules/contentbox/modules/contentbox-admin/views/content/editorHelper.cfm:4` method 'getContentTemplate' in contentbox.models.content.BaseContent\|contentbox.models.content.Entry\|contentbox.models.content.Page\|contentbox.models.cont… |
| 1 | `method 'getCurrentEntry' has no component return type → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getCustomFieldsAsStruct() | `modules/contentbox/models/system/CBHelper.cfc:778` method 'getCurrentEntry' has no component return type (chain to 'getCustomFieldsAsStruct') |
| 1 | `method 'getCurrentPage' has no component return type → modules/contentbox/models/content/BaseContent.cfc` | 1 | medium | declares getCustomFieldsAsStruct() | `modules/contentbox/models/system/CBHelper.cfc:776` method 'getCurrentPage' has no component return type (chain to 'getCustomFieldsAsStruct') |
| 1 | `method 'getCurrentPage' has no component return type → modules/contentbox/models/system/Site.cfc` | 5 | low | declares getSlug() (5 candidates) | `modules/contentbox/models/system/CBHelper.cfc:579` method 'getCurrentPage' has no component return type (chain to 'getSlug') |
| 1 | `method 'getCurrentRoutedURL' in coldbox.system.web.context.RequestContext has no component return type` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/modules.cfc:50` method 'getCurrentRoutedURL' in coldbox.system.web.context.RequestContext has no component return type (chain to 'replacenocase') |
| 1 | `method 'getCustomFields' has no component return type` | 7 | none |  | `modules/contentbox/models/content/BaseContent.cfc:1699` method 'getCustomFields' has no component return type (chain to 'clear') |
| 1 | `method 'getCustomFields' in BaseContent has no component return type` | 0 | none |  | `modules/contentbox/models/content/ContentTemplateService.cfc:190` method 'getCustomFields' in BaseContent has no component return type (chain to 'each') |
| 1 | `method 'getCustomFields' in contentbox.models.content.BaseContent\|contentbox.models.content.Entry\|contentbox.models.content.Page\|contentbox.models.content…` | 1 | none |  | `modules/contentbox/modules/contentbox-admin/views/_components/editor/CustomFieldsHelper.cfm:6` method 'getCustomFields' in contentbox.models.content.BaseContent\|contentbox.models.content.Entry\|contentbox.models.content.Page\|contentbox.models.content… |
| 1 | `method 'getCustomFieldsAsStruct' has no component return type` | 0 | none |  | `modules/contentbox/models/content/BaseContent.cfc:1832` method 'getCustomFieldsAsStruct' has no component return type (chain to 'keyArray') |
| 1 | `method 'getInstance' in coldbox.system.ioc.Injector has no component return type` | 0 | none |  | `modules/contentbox/models/content/BaseContent.cfc:615` method 'getInstance' in coldbox.system.ioc.Injector has no component return type (chain to 'each') |
| 1 | `method 'getInstance' in coldbox.system.ioc.Injector has no component return type → modules/contentbox/models/security/SecurityRule.cfc` | 2 | medium | declares setMessage() | `modules/contentbox/models/security/SecurityValidator.cfc:90` method 'getInstance' in coldbox.system.ioc.Injector has no component return type (chain to 'setMessage') |
| 1 | `method 'getInstance' in coldbox.system.ioc.Injector has no component return type → modules/contentbox/models/system/CBHelper.cfc` | 1 | medium | declares site() | `modules/contentbox/models/content/ContentTemplateService.cfc:142` method 'getInstance' in coldbox.system.ioc.Injector has no component return type (chain to 'site') |
| 1 | `method 'getInterceptor' in coldbox.system.web.services.InterceptorService has no component return type → modules/contentbox-installer/ModuleConfig.cfc` | 63 | low | declares configure() (53 candidates) | `modules/contentbox-installer/models/InstallerService.cfc:73` method 'getInterceptor' in coldbox.system.web.services.InterceptorService has no component return type (chain to 'configure') |
| 1 | `method 'getMenuItems' has no component return type` | 0 | none |  | `modules/contentbox/models/menu/Menu.cfc:207` method 'getMenuItems' has no component return type (chain to 'filter') |
| 1 | `method 'getResults' in SearchResults@contentbox has no component return type` | 1 | none |  | `tests/specs/contentbox-web/contentbox/unit/search/DBSearchTest.cfc:42` method 'getResults' in SearchResults@contentbox has no component return type (chain to 'size') |
| 1 | `method 'getRuleId' in SecurityRule has no component return type` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/securityRules/rulesTable.cfm:25` method 'getRuleId' in SecurityRule has no component return type (chain to 'replace') |
| 1 | `method 'getServiceByType' has no component return type` | 2 | none |  | `modules/contentbox/models/content/ContentService.cfc:1175` method 'getServiceByType' has no component return type (chain to 'findWhere') |
| 1 | `method 'getSetting' in coldbox.system.web.Controller has no component return type` | 0 | none |  | `modules/contentbox/modules/contentbox-ui/interceptors/CBRequest.cfc:41` method 'getSetting' in coldbox.system.web.Controller has no component return type (chain to 'keyExists') |
| 1 | `method 'getSiteSettingDefaults' in settingService@contentbox has no component return type` | 0 | none |  | `modules/contentbox/models/system/SiteService.cfc:620` method 'getSiteSettingDefaults' in settingService@contentbox has no component return type (chain to 'reduce') |
| 1 | `method 'getSubscriptions' has no component return type` | 1 | none |  | `modules/contentbox/models/subscriptions/Subscriber.cfc:103` method 'getSubscriptions' has no component return type (chain to 'map') |
| 1 | `method 'getUser' in cbsecurity.models.jwt.JwtService has no component return type → modules/contentbox/models/search/SearchResults.cfc` | 10 | medium | declares getMemento() | `modules/contentbox/modules/contentbox-api/modules/contentbox-api-v1/handlers/auth.cfc:118` method 'getUser' in cbsecurity.models.jwt.JwtService has no component return type (chain to 'getMemento') |
| 1 | `method 'getWidget' has no component return type → modules/contentbox/models/ui/BaseWidget.cfc` | 26 | low | declares renderit() (25 candidates) | `modules/contentbox/models/system/CBHelper.cfc:1814` method 'getWidget' has no component return type (chain to 'renderit') |
| 1 | `method 'populate' has no component return type → modules/contentbox/models/content/BaseContent.cfc` | 6 | low | declares setSite() (6 candidates) | `modules/contentbox/modules/contentbox-admin/handlers/categories.cfc:97` method 'populate' has no component return type (chain to 'setSite') |
| 1 | `method 'populateFromStruct' in coldbox.system.core.dynamic.ObjectPopulator has no component return type → modules/contentbox/models/content/BaseContent.cfc` | 7 | low | declares setRelatedContent() (7 candidates) | `modules/contentbox/models/content/ContentService.cfc:1204` method 'populateFromStruct' in coldbox.system.core.dynamic.ObjectPopulator has no component return type (chain to 'setRelatedContent') |
| 1 | `method 'populateFromStruct' in coldbox.system.core.dynamic.ObjectPopulator has no component return type → modules/contentbox/models/system/Setting.cfc` | 6 | low | declares setSite() (6 candidates) | `modules/contentbox/models/system/SiteService.cfc:492` method 'populateFromStruct' in coldbox.system.core.dynamic.ObjectPopulator has no component return type (chain to 'setSite') |

<details><summary>Groups with several candidates</summary>

- `method 'site' has no component return type → modules/contentbox/models/system/Site.cfc` — 17 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/system/Site.cfc` — declares getsiteID(); named like the receiver 'site'
  - low `modules/contentbox/models/system/Setting.cfc` — declares getsiteID()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getsiteID()
  - low `modules/contentbox/models/content/Category.cfc` — declares getsiteID()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getsiteID()
- `method 'getSite' has no component return type → modules/contentbox/models/system/Site.cfc` — 7 finding(s), 5 candidate(s):
  - medium `modules/contentbox/models/system/Site.cfc` — declares getSiteId(); named like the receiver 'Site'
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSiteId()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSiteId()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSiteId()
  - low `modules/contentbox/models/system/Setting.cfc` — declares getSiteId()
- `method 'getInstance' in coldbox.system.ioc.Injector has no component return type → modules/contentbox/models/system/SettingService.cfc` — 4 finding(s), 11 candidate(s):
  - low `modules/contentbox/models/system/SettingService.cfc` — declares importFromData()
  - low `modules/contentbox/models/system/SiteService.cfc` — declares importFromData()
  - low `modules/contentbox/models/content/CategoryService.cfc` — declares importFromData()
  - low `modules/contentbox/models/content/ContentService.cfc` — declares importFromData()
  - low `modules/contentbox/models/content/ContentTemplateService.cfc` — declares importFromData()
  - low `modules/contentbox/models/menu/MenuService.cfc` — declares importFromData()
  - low `modules/contentbox/models/security/AuthorService.cfc` — declares importFromData()
  - low `modules/contentbox/models/security/PermissionGroupService.cfc` — declares importFromData()
  - … 3 more
- `method 'getActiveContent' has no component return type → modules/contentbox/models/content/BaseContent.cfc` — 3 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getAuthorName()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getAuthorName()
- `method 'getCurrentEntry' has no component return type → modules/contentbox/models/comments/Comment.cfc` — 2 finding(s), 7 candidate(s):
  - low `modules/contentbox/models/comments/Comment.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/CustomField.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/Stats.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/subscriptions/CommentSubscription.cfc` — declares hasRelatedContent(), getRelatedContent()
- `method 'getCurrentPage' has no component return type → modules/contentbox/models/comments/Comment.cfc` — 2 finding(s), 7 candidate(s):
  - low `modules/contentbox/models/comments/Comment.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/CustomField.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/content/Stats.cfc` — declares hasRelatedContent(), getRelatedContent()
  - low `modules/contentbox/models/subscriptions/CommentSubscription.cfc` — declares hasRelatedContent(), getRelatedContent()
- `method 'getCurrentPage' in contentbox.models.system.CBHelper has no component return type → modules/contentbox/models/content/BaseContent.cfc` — 2 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
- `method 'getActiveContent' has no component return type → modules/contentbox/models/content/ContentVersion.cfc` — 1 finding(s), 4 candidate(s):
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares getIsActive()
  - low `modules/contentbox/models/modules/Module.cfc` — declares getIsActive()
  - low `modules/contentbox/models/security/Author.cfc` — declares getIsActive()
  - low `modules/contentbox/models/system/Site.cfc` — declares getIsActive()
- `method 'getCurrentPage' has no component return type → modules/contentbox/models/system/Site.cfc` — 1 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/system/Site.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Category.cfc` — declares getSlug()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares getSlug()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares getSlug()
- `method 'getInterceptor' in coldbox.system.web.services.InterceptorService has no component return type → modules/contentbox-installer/ModuleConfig.cfc` — 1 finding(s), 53 candidate(s):
  - low `modules/contentbox-installer/ModuleConfig.cfc` — declares configure()
  - low `modules/contentbox/ModuleConfig.cfc` — declares configure()
  - low `config/CacheBox.cfc` — declares configure()
  - low `config/Coldbox.cfc` — declares configure()
  - low `config/Router.cfc` — declares configure()
  - low `config/Scheduler.cfc` — declares configure()
  - low `config/WireBox.cfc` — declares configure()
  - low `modules/contentbox/config/Scheduler.cfc` — declares configure()
  - … 45 more
- `method 'getWidget' has no component return type → modules/contentbox/models/ui/BaseWidget.cfc` — 1 finding(s), 25 candidate(s):
  - low `modules/contentbox/models/ui/BaseWidget.cfc` — declares renderit()
  - low `modules/contentbox/models/ui/Paging.cfc` — declares renderit()
  - low `modules/contentbox/widgets/Archives.cfc` — declares renderit()
  - low `modules/contentbox/widgets/Categories.cfc` — declares renderit()
  - low `modules/contentbox/widgets/CommentForm.cfc` — declares renderit()
  - low `modules/contentbox/widgets/ContentStore.cfc` — declares renderit()
  - low `modules/contentbox/widgets/EntryInclude.cfc` — declares renderit()
  - low `modules/contentbox/widgets/Menu.cfc` — declares renderit()
  - … 17 more
- `method 'populate' has no component return type → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 6 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares setSite()
  - low `modules/contentbox/models/content/Category.cfc` — declares setSite()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares setSite()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares setSite()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares setSite()
  - low `modules/contentbox/models/system/Setting.cfc` — declares setSite()
- `method 'populateFromStruct' in coldbox.system.core.dynamic.ObjectPopulator has no component return type → modules/contentbox/models/content/BaseContent.cfc` — 1 finding(s), 7 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares setRelatedContent()
  - low `modules/contentbox/models/content/ContentVersion.cfc` — declares setRelatedContent()
  - low `modules/contentbox/models/content/CustomField.cfc` — declares setRelatedContent()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares setRelatedContent()
  - low `modules/contentbox/models/content/Stats.cfc` — declares setRelatedContent()
  - low `modules/contentbox/models/comments/Comment.cfc` — declares setRelatedContent()
  - low `modules/contentbox/models/subscriptions/CommentSubscription.cfc` — declares setRelatedContent()
- `method 'populateFromStruct' in coldbox.system.core.dynamic.ObjectPopulator has no component return type → modules/contentbox/models/system/Setting.cfc` — 1 finding(s), 6 candidate(s):
  - low `modules/contentbox/models/system/Setting.cfc` — declares setSite()
  - low `modules/contentbox/models/content/BaseContent.cfc` — declares setSite()
  - low `modules/contentbox/models/content/Category.cfc` — declares setSite()
  - low `modules/contentbox/models/content/ContentTemplate.cfc` — declares setSite()
  - low `modules/contentbox/models/content/Relocation.cfc` — declares setSite()
  - low `modules/contentbox/models/menu/Menu.cfc` — declares setSite()

</details>

## Method definitions — a method not found where it was looked for

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 16 | `getinstance` | 4 | none |  | `modules/contentbox/email_templates/author_new.cfm:2` no qualifier, not in file |
| 13 | `columns` | 2 | none |  | `build/patches/3.0.0-beta/Update.cfc:497` method 'columns' not found in dbinfo |
| 13 | `version` | 0 | none |  | `build/patches/3.0.0-beta/Update.cfc:505` method 'version' not found in dbinfo |
| 5 | `getsystemsetting` | 2 | none |  | `config/Coldbox.cfc:23` no qualifier, not in file |
| 4 | `setnextevent` | 0 | none |  | `build/patches/3.1.0/Update.cfc:94` method 'setNextEvent' not found in coldbox.system.web.Controller |
| 3 | `csrftoken` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/modules/contentbox-security/views/security/login.cfm:24` not found in extends chain |
| 3 | `csrfverify` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/modules/contentbox-security/handlers/security.cfc:55` not found in extends chain |
| 2 | `error` | 3 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:75` no qualifier, not in file |
| 2 | `generatepasswordresettoken` | 0 | none |  | `tests/specs/contentbox-api/authSpec.cfc:224` method 'generatePasswordResetToken' not found in securityService@contentbox |
| 2 | `getcwd` | 1 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:44` no qualifier, not in file |
| 1 | `confirm` | 2 | none |  | `tests/resources/seeds/archive/BaseSeeder.cfc:90` no qualifier, not in file |
| 1 | `delivermedia` | 3 | low | declares deliverMedia; extends BaseProvider.cfc (3 candidates) | `modules/contentbox/modules/contentbox-ui/handlers/media.cfc:87` method 'deliverMedia' not found in BaseProvider |
| 1 | `evictentity` | 0 | none |  | `modules/contentbox/models/comments/CommentService.cfc:228` not found in extends chain |
| 1 | `filter` | 0 | none |  | `modules/contentbox/models/content/ContentTemplateService.cfc:177` method 'keyArray' not found in qb.models.Schema.SchemaBuilder (chain to 'filter') |
| 1 | `getcache` | 9 | low | declares getCache (3 candidates) | `modules/contentbox/modules/contentbox-admin/helpers/Mixins.cfm:33` no qualifier, not in file |
| 1 | `getdisplayexpireddate` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/views/content/quickLook.cfm:15` method 'getDisplayExpiredDate' not found in ContentStore\|Entry\|Page |
| 1 | `getlayout` | 1 | high | declares getLayout; extends BaseContent.cfc | `modules/contentbox/modules/contentbox-ui/views/adminbar/index.cfm:76` method 'getLayout' not found in BaseContent |
| 1 | `renderexcerpt` | 2 | low | declares renderExcerpt; beside the calling file (2 candidates) | `modules/contentbox/models/content/BaseContent.cfc:1624` not found in extends chain |
| 1 | `run` | 55 | low | declares run (49 candidates) | `tests/index.cfm:29` no qualifier, not in file |
| 1 | `validate` | 6 | low | declares validate; beside the calling file (5 candidates) | `modules/contentbox/models/exporters/BaseExporter.cfc:60` no qualifier, not in file |
| 1 | `view` | 2 | none |  | `modules/contentbox/modules/contentbox-admin/helpers/Mixins.cfm:11` no qualifier, not in file |

<details><summary>Groups with several candidates</summary>

- `delivermedia` — 1 finding(s), 3 candidate(s):
  - low `modules/contentbox/models/media/CFContentMediaProvider.cfc:26` — declares deliverMedia; extends BaseProvider.cfc
  - low `modules/contentbox/models/media/ForwardMediaProvider.cfc:26` — declares deliverMedia; extends BaseProvider.cfc
  - low `modules/contentbox/models/media/RelocationMediaProvider.cfc:26` — declares deliverMedia; extends BaseProvider.cfc
- `getcache` — 1 finding(s), 3 candidate(s):
  - low `modules/contentbox/models/content/BaseContent.cfc:212` — declares getCache
  - low `modules/contentbox/models/security/SecurityService.cfc:30` — declares getCache
  - low `modules/contentbox/modules/contentbox-admin/modules/contentbox-security/modules/contentbox-email-twofactor/models/EmailTwoFactorProvider.cfc:18` — declares getCache
- `renderexcerpt` — 1 finding(s), 2 candidate(s):
  - low `modules/contentbox/models/content/Entry.cfc:81` — declares renderExcerpt; beside the calling file
  - low `modules/contentbox/models/content/Page.cfc:118` — declares renderExcerpt; beside the calling file
- `run` — 1 finding(s), 49 candidate(s):
  - low `tests/resources/SeedData.cfc:26` — declares run
  - low `build/BuildDocs.cfc:43` — declares run
  - low `tests/resources/seeds/TestFixtures.cfc:9` — declares run
  - low `tests/specs/contentbox-api/AuthorsSpec.cfc:30` — declares run
  - low `tests/specs/contentbox-api/CategoriesSpec.cfc:26` — declares run
  - low `tests/specs/contentbox-api/ContentStoreSpec.cfc:28` — declares run
  - low `tests/specs/contentbox-api/EchoSpec.cfc:19` — declares run
  - low `tests/specs/contentbox-api/EntriesSpec.cfc:28` — declares run
  - … 41 more
- `validate` — 1 finding(s), 5 candidate(s):
  - low `modules/contentbox/models/exporters/DataExporter.cfc:47` — declares validate; beside the calling file
  - low `modules/contentbox/models/exporters/FileExporter.cfc:56` — declares validate; beside the calling file
  - low `modules/contentbox/models/exporters/ICBExporter.cfc:13` — declares validate; beside the calling file
  - low `modules/contentbox/models/ui/Widget.cfc:37` — declares validate
  - low `modules/contentbox/models/validators/UniqueSiteFieldValidator.cfc:32` — declares validate

</details>

## Object references — a component path that names no file

| Findings | Group | Defined | Confidence | Evidence | Example |
|---:|---|---:|---|---|---|
| 27 | `component 'coldbox.system.orm.hibernate.util.ORMUtilFactory' does not exist (calling 'getORMUtil')` | 1 | none |  | `build/patches/1-0-7/Update.cfc:156` component 'coldbox.system.orm.hibernate.util.ORMUtilFactory' does not exist (calling 'getORMUtil') |
| 27 | `component 'coldbox.system.orm.hibernate.util.ORMUtilFactory' does not exist (chain hop 'getORMUtil' to 'getDefaultDatasource')` | 1 | none |  | `build/patches/1-0-7/Update.cfc:156` component 'coldbox.system.orm.hibernate.util.ORMUtilFactory' does not exist (chain hop 'getORMUtil' to 'getDefaultDatasource') |
| 2 | `chained on 'cbfs', which is not found (calling 'get')` | 10 | none |  | `modules/contentbox/modules/contentbox-admin/modules/contentbox-filebrowser/handlers/Editor.cfc:11` chained on 'cbfs', which is not found (calling 'get') |
| 1 | `base component does not resolve; 1 inherited call not checked` | ? | none |  | `modules/contentbox/models/security/SecurityValidator.cfc:9` base component does not resolve; 1 inherited call not checked |
| 1 | `chained on 'cbfs', which is not found (calling 'filter')` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/sites.cfc:67` chained on 'cbfs', which is not found (calling 'filter') |
| 1 | `chained on 'cbfs', which is not found (calling 'getDisks')` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/sites.cfc:67` chained on 'cbfs', which is not found (calling 'getDisks') |
| 1 | `chained on 'cbfs', which is not found (calling 'keyArray')` | 0 | none |  | `modules/contentbox/modules/contentbox-admin/handlers/sites.cfc:67` chained on 'cbfs', which is not found (calling 'keyArray') |
| 1 | `chained on 'getCache', which is not found (calling 'getOrSet')` | 1 | none |  | `modules/contentbox/modules/contentbox-admin/helpers/Mixins.cfm:33` chained on 'getCache', which is not found (calling 'getOrSet') |
