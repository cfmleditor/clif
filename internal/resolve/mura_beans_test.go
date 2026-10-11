package resolve_test

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

func TestMuraLiteralBeansUseStartupPolicy(t *testing.T) {
	for _, mode := range []string{"literal", "related alternatives", "unrelated alternatives", "unknown alternative", "excluded", "replacement", "cycle", "duplicate", "computed root", "no preset", "configured roots", "unknown startup mapping", "explicit rule"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			write := startupFixtureWriter(t, dir)

			folders := `"/mura"`
			if mode == "computed root" {
				folders = `runtimeFolders`
			}

			app := `component {this.mappings["/mura"]=expandPath("./core/mura");this.mappings["/muraWRM"]=expandPath("./"); include "core/method.cfm";}`
			if mode == "unknown startup mapping" {
				app = `component {this.mappings["/mura"]=expandPath("./core/mura");this.mappings["/muraWRM"]=unknown; include "core/method.cfm";}`
			}

			write("Application.cfc", app)
			write("core/Application.cfc", `component {}`)
			write("core/method.cfm", `<cfscript>function onApplicationStart(){include "/muraWRM/core/startup.cfm";}</cfscript>`)
			write("core/appcfc/applicationSettings.cfm", `<cfscript></cfscript>`)
			write("core/mura/configBean.cfc", `component {}`)
			write("core/mura/bean/beanFactory.cfc", `component {function init(folders,config={}){} function getBean(beanName){} function isSingleton(beanName){} function beanIsTransient(singleDir,dir,beanName){} function findSetters(cfc,iocMeta){} function addAlias(aliasName,beanName){} function addBean(beanName,beanValue){}}`)
			write("core/mura/beans/Item.cfc", `component accessors=true {property name="configBean";function setConfigBean(configBean){variables.configBean=arguments.configBean;}function known(){}}`)

			item := filepath.Join(dir, "core", "mura", "beans", "Item.cfc")
			startup := `<cfscript>variables.serviceFactory=new mura.bean.beanFactory(` + folders + `,{recurse=true,exclude:[]}); variables.serviceFactory.addAlias("selected","Item");`

			switch mode {
			case "related alternatives", "unrelated alternatives", "unknown alternative":
				extends := ` extends="Item"`
				if mode == "unrelated alternatives" {
					extends = ""
				}

				write("core/mura/beans/Other.cfc", `component`+extends+` {function subclassOnly(){}}`)

				other := "Other"
				if mode == "unknown alternative" {
					other = "Missing"
				}

				startup = `<cfscript>variables.serviceFactory=new mura.bean.beanFactory("/mura",{recurse=true,exclude:[]}); if(runtimeEngine){variables.serviceFactory.addAlias("selected","Item");}else{variables.serviceFactory.addAlias("selected","` + other + `");}`
			case "excluded":
				startup = `<cfscript>variables.serviceFactory=new mura.bean.beanFactory("/mura",{exclude:["/beans/"]});variables.serviceFactory.addAlias("selected","Item");`
			case "replacement":
				startup += `variables.serviceFactory.addBean("selected",runtimeValue);`
			case "cycle":
				startup += `variables.serviceFactory.addAlias("Item","selected");`
			case "duplicate":
				write("core/mura/other/Item.cfc", `component {}`)
			}

			write("core/startup.cfm", startup+`</cfscript>`)

			r := &resolve.Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}, Resolvers: []parser.Resolver{{Match: `(?i)^\$$`, Resolve: "mura.MuraScope", NameOnly: true}}}
			if mode == "no preset" {
				r.Resolvers = nil
			}

			if mode == "configured roots" {
				r.BeanPaths = map[string]string{"custom": dir}
			}

			if mode == "explicit rule" {
				r.Resolvers = append([]parser.Resolver{{Match: `getBean("selected")`, Resolve: "explicit", Prefix: "getBean"}}, r.Resolvers...)
			}

			got := parser.ResolveFromCall(`getBean("selected")`, r.BeanResolvers(r.BeanPaths))
			if mode == "explicit rule" {
				if got != "explicit" {
					t.Fatal(got)
				}

				return
			}

			assertMuraSetterOwnership(t, r, dir, item, mode)

			if mode == "literal" || mode == "related alternatives" {
				if got != item {
					t.Fatalf("got %q, want %q", got, item)
				}
			} else if filepath.IsAbs(got) {
				t.Fatalf("unsafe lookup: %s", got)
			}
		})
	}
}

func assertMuraSetterOwnership(t *testing.T, r *resolve.Resolver, dir, item, mode string) {
	t.Helper()

	if mode == "excluded" {
		if lookup := r.SetterLookup(item); lookup != nil && lookup("configBean") != "" {
			t.Fatal("excluded component acquired setter injection")
		}
	}

	if mode == "literal" {
		lookup := r.SetterLookup(item)
		if lookup == nil {
			t.Fatal("discovered factory root did not establish setter ownership")
		}

		want := filepath.Join(dir, "core", "mura", "configBean.cfc")
		if got := lookup("configBean"); got != want {
			t.Fatalf("setter dependency %q, want %q", got, want)
		}

		if got := lookup("selected"); got != "" {
			t.Fatalf("transient acquired singleton injection: %q", got)
		}

		getter := r.ResolveFunc(item, "getConfigBean", dir)
		if getter == nil || getter.ReturnComponent != want {
			t.Fatalf("setter field/accessor lost source policy: %+v", getter)
		}

		if r.SetterLookup(filepath.Join(dir, "Manual.cfc")) != nil {
			t.Fatal("factory ownership escaped its root")
		}
	}
}
