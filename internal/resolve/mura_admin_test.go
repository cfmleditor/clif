package resolve

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
)

const embeddedMuraFramework = `<cfcomponent>
<cffunction name="view"><cfargument name="path"><cfreturn internalView(arguments.path)></cffunction>
<cffunction name="layout"><cfargument name="path"><cfreturn internalLayout(arguments.path)></cffunction>
<cffunction name="buildURL"><cfargument name="action"></cffunction>
<cffunction name="internalView"><cfargument name="viewPath"><cfsavecontent variable="response"><cfinclude template="#viewPath#"></cfsavecontent></cffunction>
<cffunction name="internalLayout"><cfargument name="layoutPath"><cfsavecontent variable="response"><cfinclude template="#layoutPath#"></cfsavecontent></cffunction>
</cfcomponent>`

func TestEmbeddedMuraViewsUseOnlyTheirGoverningOwner(t *testing.T) {
	for _, tc := range []struct {
		name          string
		changes       map[string]string
		page          string
		enabled, want bool
	}{
		{"view", nil, "admin/core/views/list.cfm", true, true},
		{"layout", nil, "admin/common/layouts/main.cfm", true, true},
		{"disabled", nil, "admin/core/views/list.cfm", false, false},
		{"outside admin", nil, "public/views/list.cfm", true, false},
		{"nested application", map[string]string{"admin/core/views/isolated/Application.cfc": `component {}`}, "admin/core/views/isolated/list.cfm", true, false},
		{"missing marker", map[string]string{"core/mura/configBean.cfc": ""}, "admin/core/views/list.cfm", true, false},
		{"mapping override", map[string]string{"admin/Application.cfc": `component extends="framework" {this.mappings["/mura"]=expandPath("./custom");function rbKey(key){}}`}, "admin/core/views/list.cfm", true, false},
		{"renderer override", map[string]string{"admin/Application.cfc": `component extends="framework" {function internalView(viewPath){} function rbKey(key){}}`}, "admin/core/views/list.cfm", true, false},
		{"changed include", map[string]string{"admin/framework.cfc": strings.ReplaceAll(embeddedMuraFramework, "#viewPath#", "other.cfm")}, "admin/core/views/list.cfm", true, false},
		{"nested commented include", map[string]string{"admin/framework.cfc": strings.ReplaceAll(embeddedMuraFramework, `<cfinclude template="#viewPath#">`, `<!--- outer <!--- inner ---> <cfinclude template="#viewPath#"> --->`)}, "admin/core/views/list.cfm", true, false},
		{"path mutation", map[string]string{"admin/framework.cfc": strings.ReplaceAll(embeddedMuraFramework, `<cfinclude template="#viewPath#">`, `<cfset arguments.viewPath=unknown><cfinclude template="#viewPath#">`)}, "admin/core/views/list.cfm", true, false},
		{"view dispatch override", map[string]string{"admin/framework.cfc": strings.ReplaceAll(embeddedMuraFramework, `<cfreturn internalView(arguments.path)>`, `<cfreturn unknownRenderer(arguments.path)>`)}, "admin/core/views/list.cfm", true, false},
		{"commented include", map[string]string{"admin/framework.cfc": strings.ReplaceAll(embeddedMuraFramework, `<cfinclude template="#viewPath#">`, `<!--- <cfinclude template="#viewPath#"> --->`)}, "admin/core/views/list.cfm", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			files := map[string]string{
				"Application.cfc":                     `component {include "core/appcfc/applicationSettings.cfm";}`,
				"core/appcfc/applicationSettings.cfm": `<cfscript>this.mappings["/mura"]=expandPath("../mura");</cfscript>`,
				"core/mura/configBean.cfc":            `component {}`, "core/mura/bean/beanFactory.cfc": `component {}`,
				"admin/Application.cfc": `component extends="framework" {function rbKey(key){}}`,
				"admin/framework.cfc":   embeddedMuraFramework,
				tc.page:                 `<cfoutput>#rbKey("x")# #missingHelper()#</cfoutput>`,
			}
			maps.Copy(files, tc.changes)

			writeFiles(t, dir, files)

			if tc.name == "missing marker" { // The layout recognizer requires an existing file.
				if err := os.Remove(filepath.Join(dir, "core", "mura", "configBean.cfc")); err != nil {
					t.Fatal(err)
				}
			}

			var frameworks []string
			if tc.enabled {
				frameworks = []string{"mura"}
			}

			got := reasonsWith(t, &Resolver{ImplicitExtends: config.ImplicitExtends(frameworks)}, dir, tc.page)
			if (got["rbKey"] == "") != tc.want {
				t.Fatalf("rbKey: %q; want resolved %v", got["rbKey"], tc.want)
			}

			if got["missingHelper"] == "" {
				t.Fatal("unknown methods must still be checked")
			}
		})
	}
}

func TestEmbeddedMuraOwnerInvalidatesWhenRendererChanges(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"Application.cfc":                     `component {include "core/appcfc/applicationSettings.cfm";}`,
		"core/appcfc/applicationSettings.cfm": `<cfscript>this.mappings["/mura"]=expandPath("../mura");</cfscript>`,
		"core/mura/configBean.cfc":            `component {}`, "core/mura/bean/beanFactory.cfc": `component {}`,
		"admin/Application.cfc":     `component extends="framework" {function rbKey(key){}}`,
		"admin/framework.cfc":       embeddedMuraFramework,
		"admin/core/views/list.cfm": `<cfoutput>#rbKey("x")#</cfoutput>`,
	}
	writeFiles(t, dir, files)

	r := &Resolver{ImplicitExtends: config.ImplicitExtends([]string{"mura"}), Discovery: &DiscoveryDependencies{}}
	if got := reasonsWith(t, r, dir, "admin/core/views/list.cfm"); got["rbKey"] != "" {
		t.Fatal(got)
	}

	if !r.DiscoveryAffected(filepath.Join(dir, "admin", "framework.cfc")) || !r.DiscoveryAffected(filepath.Join(dir, "admin", "Application.cfc")) {
		t.Fatal("owner source not tracked for invalidation")
	}

	nested := filepath.Join(dir, "admin", "core", "views", "Application.cfc")
	if !r.DiscoveryAffected(nested) {
		t.Fatal("new nearer Application would not invalidate ownership")
	}

	writeFiles(t, dir, map[string]string{"admin/core/views/Application.cfc": `component {}`})
	r.InvalidatePaths()

	if got := reasonsWith(t, r, dir, "admin/core/views/list.cfm"); got["rbKey"] == "" {
		t.Fatal("new application borrowed the outer owner")
	}

	if err := os.Remove(nested); err != nil {
		t.Fatal(err)
	}

	r.InvalidatePaths()
	writeFiles(t, dir, map[string]string{"admin/framework.cfc": strings.ReplaceAll(embeddedMuraFramework, "#viewPath#", "other.cfm")})
	r.InvalidatePaths()

	if got := reasonsWith(t, r, dir, "admin/core/views/list.cfm"); got["rbKey"] == "" {
		t.Fatal("stale owner survived renderer edit")
	}
}

func TestEmbeddedMuraViewsPreserveExplicitImplicitBase(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Chosen.cfc":                `component {function chosen(){}}`,
		"admin/core/views/list.cfm": `<cfoutput>#chosen()# #rbKey("x")#</cfoutput>`,
	})

	got := reasonsWith(t, &Resolver{ImplicitExtends: func(string) string { return "Chosen" }}, dir, "admin/core/views/list.cfm")
	if got["chosen"] != "" || got["rbKey"] == "" {
		t.Fatal(got)
	}
}
