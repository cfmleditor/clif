package config

import (
	"slices"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

// TestColdBoxImplicitBases: a file that extends nothing is given the base its
// place in a ColdBox app implies — a directory anywhere in its path, the
// deepest winning, and the right extension; a file rule only for that file.
func TestColdBoxImplicitBases(t *testing.T) {
	implicit := ImplicitExtends([]string{"coldbox"})

	for path, want := range map[string]string{
		"/app/handlers/Main.cfc":                         "coldbox.system.EventHandler",
		"/app/modules_app/blog/handlers/admin/Posts.cfc": "coldbox.system.EventHandler",
		"/app/interceptors/Security.cfc":                 "coldbox.system.Interceptor",
		"/app/views/main/index.cfm":                      "coldbox.system.web.Renderer",
		"/app/layouts/Main.cfm":                          "coldbox.system.web.Renderer",
		"/app/views/handlers/list.cfm":                   "coldbox.system.web.Renderer", // a .cfc rule is not a .cfm's
		"/app/interceptors/handlers/Audit.cfc":           "coldbox.system.EventHandler", // deepest directory wins
		"/app/config/Router.cfc":                         "coldbox.system.web.routing.Router",
		"/app/config/Scheduler.cfc":                      "coldbox.system.web.tasks.ColdBoxScheduler",
		"/app/config/Coldbox.cfc":                        "", // no base for the config itself
		"/app/handlers/helper.cfm":                       "", // a template in handlers is not a handler
		"/app/models/UserService.cfc":                    "",
		"/app/Handlers/Main.cfc":                         "coldbox.system.EventHandler", // any case
	} {
		if got := implicit(path); got != want {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
	}

	if ImplicitExtends(nil) != nil || ImplicitExtends([]string{"nosuch"}) != nil {
		t.Error("no framework, no rule: the resolver should pay nothing")
	}
}

// TestColdBoxResolversMatchWholeNames: `event` is the request context in every
// handler, view and interceptor, and so are its scoped spellings — but not
// `oEvent`, `prc.event` or a call on it. A method's return type answers the
// chain on it, however it was reached, and not a longer name or a further hop.
func TestColdBoxResolversMatchWholeNames(t *testing.T) {
	var rs []parser.Resolver
	for _, r := range FrameworkResolvers([]string{"coldbox"}) {
		rs = append(rs, r.Parser())
	}

	const rc = "coldbox.system.web.context.RequestContext"

	for expr, want := range map[string]string{
		"event":               rc,
		"arguments.event":     rc,
		"VARIABLES.Event":     rc,
		"oEvent":              "",
		"prc.event":           "",
		"eventManager":        "",
		"wirebox":             "coldbox.system.ioc.Injector",
		"variables.injector":  "coldbox.system.ioc.Injector",
		"cbController":        "coldbox.system.web.Controller",
		"getRequestContext()": rc,
		"variables.controller.getRequestContext()": rc,
		"getRequestContextFor()":                   "",
		"getRequestContext().getValue()":           "",
	} {
		if got, _ := parser.ResolveFromCallFull(expr, rs); got != want {
			t.Errorf("%s: %q, want %q", expr, got, want)
		}
	}

	for _, r := range FrameworkResolvers([]string{"coldbox"}) {
		if !r.DynamicIfMissing {
			t.Errorf("%s: a preset resolver must be dynamicIfMissing, or an app without the framework checked out reports it missing", r.Match)
		}
	}
}

// TestFrameworksMergeAndOrder: a child config adds to its parent's frameworks,
// each name once; the presets' resolvers follow the config's own, so a
// project's rule for a name wins over the preset's.
func TestFrameworksMergeAndOrder(t *testing.T) {
	got := Merge(&JSON{Frameworks: []string{"coldbox"}}, &JSON{Frameworks: []string{"ColdBox", "wheels"}})
	if !slices.Equal(got.Frameworks, []string{"ColdBox", "wheels"}) {
		t.Errorf("merged frameworks = %v, want the child's with the parent's duplicate dropped", got.Frameworks)
	}

	if got := Merge(&JSON{Frameworks: []string{"coldbox"}}, &JSON{}).Frameworks; !slices.Equal(got, []string{"coldbox"}) {
		t.Errorf("a child naming none keeps the parent's: %v", got)
	}

	own := Resolver{Match: "event", Resolve: "my.Event", Prefix: "event"}
	r := Resolve(&JSON{Frameworks: []string{"coldbox"}, ComponentResolvers: []Resolver{own}}, "/w")

	if len(r.ComponentResolvers) < 2 || r.ComponentResolvers[0] != own {
		t.Errorf("the config's own resolver must come first: %v", r.ComponentResolvers)
	}

	if !slices.Equal(r.Frameworks, []string{"coldbox"}) {
		t.Errorf("Resolved.Frameworks = %v", r.Frameworks)
	}
}

func TestUnknownFrameworks(t *testing.T) {
	if got := UnknownFrameworks([]string{"ColdBox", "coldbx", "rails"}); !slices.Equal(got, []string{"coldbx", "rails"}) {
		t.Errorf("UnknownFrameworks = %v", got)
	}

	if !slices.Contains(KnownFrameworks(), "coldbox") {
		t.Errorf("KnownFrameworks = %v", KnownFrameworks())
	}
}

// TestEveryPresetResolverMatchesItsOwnNames: a name is taken literally — the
// $ in TestBox's $assert is a character, not an end-of-line anchor, and the
// dots in prc.oCurrentAuthor are dots — so each preset's variables resolve to
// what it says, and a near miss resolves to nothing.
func TestEveryPresetResolverMatchesItsOwnNames(t *testing.T) {
	for _, tc := range []struct{ framework, expr, want string }{
		{"testbox", "$assert", "testbox.system.Assertion"},
		{"testbox", "variables.assert", "testbox.system.Assertion"},
		{"testbox", "getMockBox()", "testbox.system.MockBox"},
		{"commandbox", "print", "commandbox.system.util.PrintBuffer"},
		{"commandbox", "command()", "commandbox.system.util.CommandDSL"},
		{"commandbox", "task()", ""}, // a ColdBox scheduler's task() is not CommandBox's
		{"commandbox", `getInstance( "DetailOutputService@wheels-cli" )`, "DetailOutputService@wheels-cli"},
		{"coldbox", "task()", "coldbox.system.web.tasks.ColdBoxScheduledTask"},
		{"contentbox", "prc.oContent", "contentbox.models.content.BaseContent|contentbox.models.content.Entry|contentbox.models.content.Page|contentbox.models.content.ContentStore"},
		{"cfmigrations", "table", "qb.models.Schema.Blueprint"},
		{"cfmigrations", "arguments.qb", "qb.models.Query.QueryBuilder"},
		{"contentbox", "prc.oCurrentAuthor", "contentbox.models.security.Author"},
		{"contentbox", "prcXoCurrentAuthor", ""},
		{"wheels", "application.wo", "wheels.Global"},
		{"fw1", "variables.fw", "framework.one"},
		// What WireBox builds is the component its id names; a DSL string
		// is ColdBox's own class; a computed id, or a call chained on the
		// result, is left alone.
		{"coldbox", `getInstance( "UserService@users" )`, "UserService@users"},
		{"coldbox", `wirebox.getInstance( name = "id:models.UserService" )`, "models.UserService"},
		{"coldbox", `wirebox.getInstance( name: "id:models.UserService" )`, "models.UserService"},
		{"coldbox", `getInstance( name: "wirebox:populator" )`, "coldbox.system.core.dynamic.ObjectPopulator"},
		{"coldbox", `getInstance( "UserService", { a = 1 } )`, "UserService"},
		{"coldbox", `getInstance( "wirebox:populator" )`, "coldbox.system.core.dynamic.ObjectPopulator"},
		{"coldbox", `getInstance( "logbox" )`, "coldbox.system.logging.LogBox"},
		{"coldbox", `getInstance( "#svc#Service" )`, ""},
		{"coldbox", `getInstance( "UserService" ).init()`, ""},
		{"coldbox", `Calendar.getInstance()`, ""},
		{"wheels", `model( "User" )`, "User"},
		{"fw1", `getBeanFactory().getBean( "userService" )`, "userService"},
		{"wheels", `model( "User" ).findAll()`, ""},
		{"mura", "$", "mura.MuraScope"},
		{"mura", "variables.$", "mura.MuraScope"},
		{"mura", "arguments.m", "mura.MuraScope"},
		{"mura", "rc.$", "mura.MuraScope"},
		{"mura", "mura", "mura.MuraScope"},
		{"mura", "application.mura", "mura.MuraScope"},
		{"mura", "application.configBean", "mura.configBean"},
		{"mura", "application.rbFactory", "mura.resourceBundle.resourceBundleFactory"},
		{"mura", "application.settingsManager", "mura.settings.settingsManager"},
		{"mura", "application.serviceFactory", "mura.bean.beanFactory"},
		{"mura", "application.settingsManagerX", ""},
		{"mura", "application.utility", "mura.utility"},
		{"mura", "application.utilityX", ""},
		{"mura", "application.pluginManager", "mura.plugin.pluginManager"},
		{"mura", "application.pluginManagerX", ""},
		{"mura", "application.contentManager", "mura.content.contentManager"},
		{"mura", "application.contentManagerX", ""},
		{"mura", "application.eventManager", "mura.plugin.pluginManager"},
		{"mura", "application.eventManagerX", ""},
		{"mura", "application.contentRenderer", "mura.content.contentRenderer"},
		{"mura", "application.contentRendererX", ""},
		{"mura", "application.contentUtility", "mura.content.contentUtility"},
		{"mura", "application.contentUtilityX", ""},
		{"mura", "application.categoryManager", "mura.category.categoryManager"},
		{"mura", "application.categoryManagerX", ""},
		{"mura", "application.clusterManager", "mura.cluster.clusterManager"},
		{"mura", "application.clusterManagerX", ""},
		{"mura", "application.contentServer", "mura.content.contentServer"},
		{"mura", "application.contentServerX", ""},
		{"mura", "application.changesetManager", "mura.content.changeset.changesetManager"},
		{"mura", "application.changesetManagerX", ""},
		{"mura", "application.scriptProtectionFilter", "mura.Portcullis"},
		{"mura", "application.scriptProtectionFilterX", ""},
		{"mura", "application.permUtility", "mura.permission"},
		{"mura", "application.permUtilityX", ""},
		{"mura", "application.emailManager", "mura.email.emailManager"},
		{"mura", "application.emailManagerX", ""},
		{"mura", "application.loginManager", "mura.login.loginManager"},
		{"mura", "application.loginManagerX", ""},
		{"mura", "application.mailinglistManager", "mura.mailinglist.mailinglistManager"},
		{"mura", "application.mailinglistManagerX", ""},
		{"mura", "application.userManager", "mura.user.userManager"},
		{"mura", "application.userManagerX", ""},
		{"mura", "application.dataCollectionManager", "mura.content.dataCollection.dataCollectionManager"},
		{"mura", "application.dataCollectionManagerX", ""},
		{"mura", "application.feedManager", "mura.content.feed.feedManager"},
		{"mura", "application.feedManagerX", ""},
		{"mura", "application.sessionTrackingManager", "mura.user.sessionTracking.sessionTrackingManager"},
		{"mura", "application.sessionTrackingManagerX", ""},
		{"mura", "application.favoriteManager", "mura.content.favorite.favoriteManager"},
		{"mura", "application.favoriteManagerX", ""},
		{"mura", "application.raterManager", "mura.content.rater.raterManager"},
		{"mura", "application.raterManagerX", ""},
		{"mura", "application.dashboardManager", "mura.dashboard.dashboardManager"},
		{"mura", "application.dashboardManagerX", ""},
		{"mura", "application.autoUpdater", "mura.autoUpdater.autoUpdater"},
		{"mura", "application.autoUpdaterX", ""},
		{"mura", "arguments.rc.contentBean", "mura.content.contentBean"},
		{"mura", "rc.siteBean", "mura.settings.settingsBean"},
		{"mura", "rc.feedBean", "mura.content.feed.feedBean"},
		{"mura", "rc.userBean", "mura.user.userBean"},
		{"mura", "arguments.rc.userBean", "mura.user.userBean"},
		{"mura", "rc.userBeanX", ""},
		{"mura", "rc.categoryBean", "mura.category.categoryBean"},
		{"mura", "arguments.rc.categoryBean", "mura.category.categoryBean"},
		{"mura", "rc.categoryBeanX", ""},
		{"mura", "rc.emailBean", "mura.email.emailBean"},
		{"mura", "arguments.rc.emailBean", "mura.email.emailBean"},
		{"mura", "rc.emailBeanX", ""},
		{"mura", "rc.listBean", "mura.mailinglist.mailinglistBean"},
		{"mura", "arguments.rc.listBean", "mura.mailinglist.mailinglistBean"},
		{"mura", "rc.listBeanX", ""},
		{"mura", "rc.changeset", "mura.content.changeset.changesetBean"},
		{"mura", "arguments.rc.changeset", "mura.content.changeset.changesetBean"},
		{"mura", "rc.changesetX", ""},
		{"mura", "rc.pluginConfig", "mura.plugin.pluginConfig"},
		{"mura", "arguments.rc.pluginConfig", "mura.plugin.pluginConfig"},
		{"mura", "rc.pluginConfigX", ""},
		{"mura", "rc.trashItem", "mura.trash.trashItemBean"},
		{"mura", "arguments.rc.trashItem", "mura.trash.trashItemBean"},
		{"mura", "rc.trashItemX", ""},
		{"mura", "rc.trashIterator", "mura.trash.trashIterator"},
		{"mura", "arguments.rc.trashIterator", "mura.trash.trashIterator"},
		{"mura", "rc.trashIteratorX", ""},
		{"mura", "rc.bean", ""},
		{"mura", "rc.item", ""},
		{"mura", "arguments.bean", ""},
		{"mura", `$.content()`, "mura.content.contentBean"},
		{"mura", `$.getContentBean()`, "mura.content.contentBean"},
		{"mura", `$.content("title")`, ""},
		{"mura", `$.currentUser()`, "mura.user.sessionUserFacade"},
		{"mura", `$.getCurrentUser()`, "mura.user.sessionUserFacade"},
		{"mura", `$.currentUser("userID")`, ""},
		{"mura", `$ . siteConfig( )`, "mura.settings.settingsBean"},
		{"mura", `$.siteConfig("siteid")`, ""},
		{"mura", `$.siteConfig("siteid","x")`, ""},
		{"mura", `$.globalConfig()`, "mura.configBean"},
		{"mura", `$.globalConfig("siteDir")`, ""},
		{"mura", `$.getContentRenderer()`, "mura.content.contentRenderer"},
		{"mura", `$.getContentRenderer(true)`, ""},
		{"mura", "$x", ""},
		{"mura", "item", ""},
		{"mura", `variables.event.getValue("muraScope")`, "mura.MuraScope"},
		{"mura", `event.getValue( 'MuraScope' )`, "mura.MuraScope"},
		{"mura", `event.getValue("other")`, ""},
		{"mura", "getMuraScope()", "mura.MuraScope"},
	} {
		var rs []parser.Resolver
		for _, r := range FrameworkResolvers([]string{tc.framework}) {
			rs = append(rs, r.Parser())
		}

		if got, _ := parser.ResolveFromCallFull(tc.expr, rs); got != tc.want {
			t.Errorf("%s %s: %q, want %q", tc.framework, tc.expr, got, tc.want)
		}
	}

	implicit := ImplicitExtends([]string{"commandbox", "fw1"})
	for path, want := range map[string]string{
		"/p/task.cfc":                     "commandbox.system.BaseTask",
		"/p/deep/in/tree/Task.cfc":        "commandbox.system.BaseTask",
		"/p/commands/wheels/Generate.cfc": "commandbox.system.BaseCommand",
		// The governing Application.cfc when it is the framework instance;
		// the resolver decides that (resolve.applicationBase).
		"/p/views/main/default.cfm": parser.ApplicationBase + "framework.one",
		"/p/models/task.cfm":        "",
	} {
		if got := implicit(path); got != want {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
	}
}

// TestAMuraDisplayObjectRunsInTheContentRenderer: Mura includes a display
// object's template from contentRenderer, under a modules or display_objects
// directory at any depth; a component there and a template elsewhere are not
// display objects.
func TestAMuraDisplayObjectRunsInTheContentRenderer(t *testing.T) {
	implicit := ImplicitExtends([]string{"mura"})
	for path, want := range map[string]string{
		"/p/core/modules/v1/gallery/index.cfm":          "mura.content.contentRenderer",
		"/p/sites/default/display_objects/nav/x.cfm":    "mura.content.contentRenderer",
		"/p/sites/default/modules/custom/index.cfm":     "mura.content.contentRenderer",
		"/p/core/modules/v1/cookie_consent/Handler.cfc": "",
		"/p/admin/core/views/carch/edit.cfm":            parser.MuraAdminBase,
		"/p/modules/blog/views/list.cfm":                "mura.content.contentRenderer",
		"/p/display_objects/blog/layouts/main.cfm":      "mura.content.contentRenderer",
	} {
		if got := implicit(path); got != want {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
	}
}

// TestAWheelsGlobalTemplateRunsInGlobal: app/global/*.cfm is mixed into
// every controller, model and view, and vendor/wheels/global/*.cfm into
// Global, so both read Global's functions; the Seeder includes seeds.cfm
// and seeds/<env>.cfm.
func TestAWheelsGlobalTemplateRunsInGlobal(t *testing.T) {
	implicit := ImplicitExtends([]string{"wheels"})
	for path, want := range map[string]string{
		"/p/app/global/auth.cfm":              "wheels.Global",
		"/p/vendor/wheels/global/strings.cfm": "wheels.Global",
		"/p/app/global/Helper.cfc":            "",
		"/p/app/db/seeds.cfm":                 "wheels.Seeder",
		"/p/app/db/seeds/testing.cfm":         "wheels.Seeder",
		"/p/app/db/seed-notes.cfm":            "",
	} {
		if got := implicit(path); got != want {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
	}
}

// TestSuggestFrameworksFromBoxJSON: a project finds out a preset exists from
// its own box.json — a dependency, a dev dependency or the package's own slug
// (coldbox-platform is the package named coldbox) — minus the presets the
// config already names. A dependency that says nothing about the framework
// the code is written against suggests nothing.
func TestSuggestFrameworksFromBoxJSON(t *testing.T) {
	for _, tc := range []struct {
		box  string
		have []string
		want []string
	}{
		{`{"dependencies":{"coldbox":"^7"},"devDependencies":{"testbox":"*","commandbox-migrations":"*"}}`, nil, []string{"cfmigrations", "coldbox", "testbox"}},
		{`{"slug":"coldbox","devDependencies":{"testbox":"*"}}`, []string{"ColdBox"}, []string{"testbox"}},
		{`{"dependencies":{"wheels-core":"^3"}}`, nil, []string{"wheels"}},
		{`{"slug":"fw1"}`, nil, []string{"fw1"}},
		{`{"devDependencies":{"commandbox-cfformat":"*","cbproxies":"*"}}`, nil, nil},
		{`not json`, nil, nil},
	} {
		if got := SuggestFrameworks([]byte(tc.box), tc.have); !slices.Equal(got, tc.want) {
			t.Errorf("%s with %v: %v, want %v", tc.box, tc.have, got, tc.want)
		}
	}
}

// TestColdBoxHelperScope: ColdBox mixes its helpers into handlers,
// interceptors, views and layouts, at any depth, and into nothing else; a
// preset with no helpers gives no scope, so the resolver pays nothing.
func TestColdBoxHelperScope(t *testing.T) {
	scope := HelperScope([]string{"coldbox"})
	for path, want := range map[string]bool{
		"/app/handlers/Main.cfc":                true,
		"/app/modules/blog/views/post/show.cfm": true,
		"/app/layouts/Main.cfm":                 true,
		"/app/interceptors/Security.cfc":        true,
		"/app/models/User.cfc":                  false,
		"/app/config/Router.cfc":                false,
	} {
		if got := scope(path); got != want {
			t.Errorf("%s: %v, want %v", path, got, want)
		}
	}

	if HelperScope([]string{"testbox"}) != nil {
		t.Error("a preset with no helper directories should give no scope")
	}
}
