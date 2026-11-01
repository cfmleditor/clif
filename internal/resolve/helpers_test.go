package resolve

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestFrameworkHelpersReachTheFilesTheyAreMixedInto: ColdBox mixes helper
// templates into handlers, views, layouts and interceptors — a module's
// (ModuleConfig.cfc's this.applicationHelper, relative to the module), the
// application's (config/ColdBox.cfc's applicationHelper, relative to the app)
// and a view's own <view>Helper.cfm and <folder>Helper.cfm. A bare call to one
// was "no qualifier, not in file": cbMessageBox() alone, 270 times in
// ContentBox's admin with its modules installed. A file the framework does not
// mix them into, a model, still does not see them.
func TestFrameworkHelpersReachTheFilesTheyAreMixedInto(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"modules/msg/ModuleConfig.cfc":     `component { this.applicationHelper = [ "helpers/Mixins.cfm" ]; function configure() {} }`,
		"modules/msg/helpers/Mixins.cfm":   `<cfscript>function cbMessageBox() { return 1; }</cfscript>`,
		"modules/admin/ModuleConfig.cfc":   `component { function onLoad() { wirebox.getInstance( "Renderer@coldbox" ).includeUDF( "#moduleMapping#/helpers/Mixins.cfm" ); } }`,
		"modules/admin/helpers/Mixins.cfm": `<cfscript>function cbAdminComponent() { return 1; }</cfscript>`,
		"config/ColdBox.cfc":               `component { function configure() { variables.coldbox = { applicationHelper : "includes/App.cfm" }; } }`,
		"includes/App.cfm":                 `<cfscript>function appHelp() { return 1; }</cfscript>`,
		"views/main/indexHelper.cfm":       `<cfscript>function viewHelp() { return 1; }</cfscript>`,
		"views/main/mainHelper.cfm":        `<cfscript>function folderHelp() { return 1; }</cfscript>`,
		"views/main/index.cfm":             `<cfoutput>#cbMessageBox()# #cbAdminComponent()# #appHelp()# #viewHelp()# #folderHelp()# #nope()#</cfoutput>`,
		"handlers/Main.cfc":                `component { function index() { cbMessageBox(); viewHelp(); } }`,
		"models/User.cfc":                  `component { function f() { cbMessageBox(); } }`,
	})

	scope := func(path string) bool {
		p := filepath.ToSlash(path)

		return strings.Contains(p, "/views/") || strings.Contains(p, "/handlers/")
	}

	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "views/main/index.cfm"), map[string]string{
		"cbMessageBox":     "",
		"cbAdminComponent": "",
		"appHelp":          "",
		"viewHelp":         "",
		"folderHelp":       "",
		"nope":             "no qualifier, not in file",
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "handlers/Main.cfc"), map[string]string{
		"cbMessageBox": "",
		"viewHelp":     "no qualifier, not in file", // a view's helper is the view's
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "models/User.cfc"), map[string]string{
		"cbMessageBox": "no qualifier, not in file",
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "views/main/index.cfm"), map[string]string{
		"cbMessageBox": "no qualifier, not in file",
	})
}

// TestPresideHelpersReachTheFilesColdBoxMixesThemInto: Preside's config hands
// ColdBox every template under system/helpers, at any depth, as application
// helpers, through a computed list (_getUdfFiles) the applicationHelper
// reader cannot see. translateResource() alone was 2,552 findings over
// Preside's handlers and views. A directory that is not Preside's (no
// services/PresideSuperClass.cfc beside it) gives nothing, and a service is
// not handed them.
func TestPresideHelpersReachTheFilesColdBoxMixesThemInto(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"system/services/PresideSuperClass.cfc":   `component { function $helper() {} }`,
		"system/helpers/presideProxies.cfm":       `<cffunction name="translateResource" access="public" returntype="any" output="false"><cfreturn "" /></cffunction>`,
		"system/helpers/sitetree/pageHelpers.cfm": `<cfscript>function pageHelp() { return translateResource( "y" ); }</cfscript>`,
		"system/handlers/Main.cfc":                `component { function index() { translateResource( "x" ); pageHelp(); } }`,
		"system/views/main/index.cfm":             `<cfoutput>#translateResource( "x" )# #nope()#</cfoutput>`,
		"system/services/Svc.cfc":                 `component { function f() { translateResource( "x" ); } }`,
		"other/helpers/orphan.cfm":                `<cfscript>function orphanHelp() { return 1; }</cfscript>`,
		"other/handlers/Main.cfc":                 `component { function index() { orphanHelp(); } }`,
	})

	scope := func(path string) bool {
		p := filepath.ToSlash(path)

		return strings.Contains(p, "/views/") || strings.Contains(p, "/handlers/")
	}

	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "system/handlers/Main.cfc"), map[string]string{
		"translateResource": "",
		"pageHelp":          "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "system/views/main/index.cfm"), map[string]string{
		"translateResource": "",
		"nope":              "no qualifier, not in file",
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "system/services/Svc.cfc"), map[string]string{
		"translateResource": "no qualifier, not in file",
	})
	// A helper is mixed in beside the others and calls them.
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "system/helpers/sitetree/pageHelpers.cfm"), map[string]string{
		"translateResource": "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "other/handlers/Main.cfc"), map[string]string{
		"orphanHelp": "no qualifier, not in file",
	})
}

// TestWheelsGlobalsReachEveryControllerModelAndView: Wheels includes the
// application's global/functions.cfm into every controller, model and view,
// and that file includes the rest (the starter app's auth.cfm, logging.cfm).
// Its hasPermission() was 17 findings in the starter app's views alone. A
// template it does not include, or an app with no global/functions.cfm, gives
// nothing.
func TestWheelsGlobalsReachEveryControllerModelAndView(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"app/global/functions.cfm":     `<cfscript> include "auth.cfm"; function util() {} </cfscript>`,
		"app/global/auth.cfm":          `<cfscript> function hasPermission( p ) { return true; } </cfscript>`,
		"app/global/unused.cfm":        `<cfscript> function notIncluded() {} </cfscript>`,
		"app/views/main/index.cfm":     `<cfoutput>#hasPermission( "x" )# #util()# #notIncluded()#</cfoutput>`,
		"app/controllers/Main.cfc":     `component { function index() { hasPermission( "x" ); } }`,
		"other/controllers/Orphan.cfc": `component { function index() { hasPermission( "x" ); } }`,
	})

	scope := func(path string) bool {
		p := filepath.ToSlash(path)

		return strings.Contains(p, "/views/") || strings.Contains(p, "/controllers/")
	}

	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "app/views/main/index.cfm"), map[string]string{
		"hasPermission": "",
		"util":          "",
		"notIncluded":   "no qualifier, not in file",
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "app/controllers/Main.cfc"), map[string]string{
		"hasPermission": "",
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: scope}, dir, "other/controllers/Orphan.cfc"), map[string]string{
		"hasPermission": "no qualifier, not in file",
	})
}

// TestWheelsViewHelpersReachTheViewsAndControllers: Controller.cfc includes
// <viewPath>/helpers.cfm into every controller, and a controller's own
// views/<name>/helpers.cfm into it, so a view calls both bare and so does the
// controller. A view in another folder does not get that folder's helpers.
func TestWheelsViewHelpersReachTheViewsAndControllers(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"app/views/helpers.cfm":          `<cfscript>function panel(){} include "more.cfm";</cfscript>`,
		"app/views/more.cfm":             `<cfscript>function pageHeader(){}</cfscript>`,
		"app/views/accounts/helpers.cfm": `<cfscript>function accountBadge(){}</cfscript>`,
		"app/views/accounts/edit.cfm":    `<cfoutput>#panel()# #pageHeader()# #accountBadge()# #missing()#</cfoutput>`,
		"app/views/other/show.cfm":       `<cfoutput>#panel()# #accountBadge()#</cfoutput>`,
		"app/controllers/Accounts.cfc":   `component { function edit(){ panel(); accountBadge(); } }`,
	})

	helpers := func(string) bool { return true }

	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: helpers}, dir, "app/views/accounts/edit.cfm"), map[string]string{
		"panel": "", "pageHeader": "", "accountBadge": "", "missing": "no qualifier, not in file",
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: helpers}, dir, "app/views/other/show.cfm"), map[string]string{
		"panel": "", "accountBadge": "no qualifier, not in file",
	})
	expectReasons(t, reasonsWith(t, &Resolver{HelperScope: helpers}, dir, "app/controllers/Accounts.cfc"), map[string]string{
		"panel": "", "accountBadge": "",
	})
}
