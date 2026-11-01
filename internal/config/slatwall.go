package config

import (
	"path/filepath"
	"strings"

	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Slatwall is a Mura plugin, and most of what it calls is reached through
// values its Application.cfc stores at startup rather than through anything a
// declaration types. These are what it stores, as its source states them:
//
//   - $.slatwall: setupRequest() puts `new Slatwall.com.utility.SlatwallScope()`
//     in the request cache and hands it to Mura as the custom scope key
//     "slatwall" (`$.setCustomMuraScopeKey( "slatwall", … )`), so $.slatwall,
//     and an admin view's rc.$.slatwall, is that SlatwallScope.
//   - getFW(): BaseObject's returns the plugin application's "fw", which
//     setupApplication() sets to `this`, the Application.cfc extending FW/1.
//   - getPluginConfig(): application.slatwall.pluginConfig, Mura's
//     pluginConfig that setPluginConfig() is handed.
//   - getValidateThis(): the application's "validateThis", a
//     `new ValidateThis.ValidateThis()`.
//   - $.slatwall.cart(), account(), product() and session(), and the
//     getCurrentCart() family behind them: the session's Order, Account and
//     Product, or a new one its service makes, and the Session itself.
//   - getService( "orderService" ): BaseObject asks the ColdSpring factory
//     for the bean of that id, and every Slatwall service is its file name.
//
// And ColdSpring autowires a service's or controller's untyped
// `property name="orderService"` and `property name="orderDAO"` with the bean
// of that name, which is the service or DAO of that file name.
const (
	slatwallScope = "Slatwall.com.utility.SlatwallScope"
	slatwallApp   = "Slatwall.Application"
)

// isSlatwall reports whether plugin is Slatwall's own checkout: the plugin
// packaged as Slatwall, with its BaseService and SlatwallScope.
func isSlatwall(plugin cfpath.MuraPlugin, read func(string) ([]byte, error)) bool {
	if !strings.EqualFold(plugin.Package, "Slatwall") {
		return false
	}

	for _, marker := range []string{"com/service/BaseService.cfc", "com/utility/SlatwallScope.cfc"} {
		if _, err := read(filepath.Join(plugin.Root, filepath.FromSlash(marker))); err != nil {
			return false
		}
	}

	return true
}

func slatwallResolvers() []Resolver {
	out := []Resolver{variableResolver(slatwallScope, "$.slatwall", "rc.$.slatwall", "request.context.$.slatwall")}
	out = append(out, returnResolver(slatwallApp, "getFW")...)
	out = append(out, returnResolver("mura.plugin.pluginConfig", "getPluginConfig")...)
	out = append(out, returnResolver("ValidateThis.ValidateThis", "getValidateThis")...)

	// SlatwallScope's cart(), account(), product() and session() with no
	// property return getCurrentCart() and its siblings, each the session's
	// own entity or a new one from its service.
	for _, e := range []struct{ entity, scope, current string }{
		{"Order", "cart", "getCurrentCart"},
		{"Account", "account", "getCurrentAccount"},
		{"Product", "product", "getCurrentProduct"},
		{"Session", "session", "getCurrentSession"},
	} {
		out = append(out, returnResolver("Slatwall.com.entity."+e.entity, e.scope, e.current)...)
	}

	return append(out, Resolver{
		Match:            `(?i)(?:^|\.)getService\(\s*(?:\w+\s*=\s*)?["']([A-Za-z_]\w*)["']\s*\)$`,
		Resolve:          "Slatwall.com.service.$1",
		Prefix:           "getService",
		DynamicIfMissing: true,
	})
}

func slatwallPropertyResolvers() []PropResolver {
	return []PropResolver{
		{Match: "$1Service", Resolve: "Slatwall.com.service.$1Service", Attribute: "name"},
		{Match: "$1DAO", Resolve: "Slatwall.com.dao.$1DAO", Attribute: "name"},
	}
}
