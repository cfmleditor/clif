package resolve_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.lsp.dev/uri"

	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

func writeDIApp(t *testing.T, app, startup string) (*resolve.Resolver, string) {
	t.Helper()
	dir := t.TempDir()

	sources := map[string]string{
		"Application.cfc": app,
		"startup.cfm":     startup,
		"framework/ioc.cfc": `component {
 function init(folders,config={}) {}
 function getBean(beanName) {}
 function isSingleton(beanName) {}
 function beanIsTransient(singleDir,dir,beanName) {}
 function findSetters(cfc,iocMeta) { isSingleton("name"); }
 function declareBean(beanName,dottedPath,isSingleton=true) {}
 function addAlias(aliasName,beanName) {}
 function addBean(beanName,beanValue) {}
 }`,
		"model/beans/User.cfc":        `component { function run() {} }`,
		"model/services/Reporter.cfc": `component { function run() {} }`,
		"model/jobs/Job.cfc":          `component { function run() {} }`,
		"model/entities/Entity.cfc":   `component { function run() {} }`,
		"model/services/Consumer.cfc": `component accessors=true {
 property name="user";
 property name="reporter";
 function setUser(user) { variables.user=arguments.user; }
 function setReporter(reporter) { variables.reporter=arguments.reporter; }
 function use() { variables.user.run(); variables.reporter.run(); }
 }`,
	}
	for name, source := range sources {
		file := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	r := &resolve.Resolver{FS: vfs.OS{}, Index: index.New(), WorkspaceFolders: []string{dir}, BeanPaths: map[string]string{"": filepath.Join(dir, "model")}, StartupFiles: []string{filepath.Join(dir, "startup.cfm")}, Resolvers: []parser.Resolver{{Match: `getBean("$1")`, Resolve: "$1", Prefix: "getBean"}}}
	r.Index.SetBeans(cfpath.BuildBeanMap(r.BeanPaths, r.FS))
	r.Resolvers = r.BeanResolvers(r.BeanPaths)

	return r, dir
}

func TestDI1InjectionUsesLifetimeAndConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, config string
		blocked      []string
	}{
		{"defaults", `{}`, []string{"user", "userBean"}},
		{"folder transients", `{transients:["services"]}`, []string{"user", "userBean", "reporter", "reporterService"}},
		{"pattern", `{transientPattern:"Job$"}`, []string{"user", "userBean", "job"}},
		{"singleton pattern", `{singletonPattern:"Reporter$"}`, []string{"user", "userBean", "job", "entity"}},
		{"singular mapping", `{singulars:{entities:"bean"}}`, []string{"user", "userBean", "entity"}},
		{"constant shadow", `{constants:{reporter:"not a CFC"}}`, []string{"user", "userBean", "reporter"}},
		{"aliases omitted", `{omitDirectoryAliases:true}`, []string{"user", "userBean", "reporterService"}},
		{"dynamic pattern", `{transientPattern:settings.pattern()}`, []string{"user", "userBean", "reporter", "reporterService", "job", "entity"}},
		{"unsupported regex", `{transientPattern:"(?<=prefix)Job"}`, []string{"user", "userBean", "reporter", "reporterService", "job", "entity"}},
		{"conflicting patterns", `{transientPattern:"Job$",singletonPattern:"Reporter$"}`, []string{"user", "userBean", "reporter", "reporterService", "job", "entity"}},
		{"literal exclusion", `{exclude:["/services/"]}`, []string{"user", "userBean", "reporter", "reporterService", "job", "entity"}},
		{"nonrecursive", `{recurse:false}`, []string{"user", "userBean", "reporter", "reporterService", "job", "entity"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, dir := writeDIApp(t, `component extends="framework.one" { variables.framework={diConfig:`+test.config+`}; }`, "")
			file := filepath.Join(dir, "model", "services", "Consumer.cfc")

			lookup := r.SetterLookup(file)
			if lookup == nil {
				t.Fatal("missing managed lookup")
			}

			blocked := map[string]bool{}
			for _, name := range test.blocked {
				blocked[name] = true
			}

			for _, name := range []string{"user", "userBean", "reporter", "reporterService", "job", "entity"} {
				got := lookup(name)

				want := r.BeanLookup(name)
				if blocked[name] {
					want = ""
				}

				if got != want {
					t.Errorf("%s: %q, want %q", name, got, want)
				}
			}
			// Direct bean retrieval continues to resolve transients.
			if r.BeanLookup("user") == "" {
				t.Fatal("transient retrieval lost its identity")
			}
		})
	}
}

func TestDI1ExplicitRegistrationLifetimeAndAliases(t *testing.T) {
	startup := `<cfscript>
 f=new framework.ioc("/model",{transientPattern:"User$"});
 f.declareBean("forced", "model.beans.User", true);
 f.addAlias("aliasForced","forced");
 f.declareBean(beanName="never",dottedPath="model.services.Reporter",isSingleton=false);
 f.addAlias("aliasNever","never");
 f.declareBean("uncertain","model.services.Reporter",settings.singleton);
 f.addAlias("cycleA","cycleB"); f.addAlias("cycleB","cycleA");
 f.addBean("reporter","a string");
 alternate=new model.jobs.Job(); f.addBean("jobSingleton",alternate);
 </cfscript>`
	r, dir := writeDIApp(t, `component {}`, startup)

	lookup := r.SetterLookup(filepath.Join(dir, "model", "services", "Consumer.cfc"))
	for name, want := range map[string]string{"forced": filepath.Join(dir, "model", "beans", "User.cfc"), "aliasForced": filepath.Join(dir, "model", "beans", "User.cfc"), "jobSingleton": filepath.Join(dir, "model", "jobs", "Job.cfc"), "never": "", "aliasNever": "", "uncertain": "", "cycleA": "", "reporter": ""} {
		if got := lookup(name); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
	// The older FW/1 loader injects any contained bean, not just singletons.
	admin := filepath.Join(dir, "admin")
	if err := os.MkdirAll(admin, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(admin, "Application.cfc"), []byte(`component extends="framework" { function setupApplication() {setBeanFactory(factory);} }`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(admin, "framework.cfc"), []byte(fw1InjectionSource), 0o600); err != nil {
		t.Fatal(err)
	}

	fallback := r.SetterLookup(filepath.Join(admin, "controllers", "Controller.cfc"))
	if fallback == nil || fallback("user") != r.BeanLookup("user") {
		t.Fatal("framework fallback acquired DI/1 transient pruning")
	}
}

func TestDI1PropertyAndIndexedSetterTypes(t *testing.T) {
	r, dir := writeDIApp(t, `component extends="framework.one" {}`, "")
	consumer := filepath.Join(dir, "model", "services", "Consumer.cfc")

	defs := r.EnsureIndexed(consumer)
	for _, fn := range defs {
		if fn.Name == "setUser" && fn.Arguments[0].Component != "" {
			t.Fatal("transient setter typed")
		}

		if fn.Name == "setReporter" && fn.Arguments[0].Component == "" {
			t.Fatal("singleton setter not typed")
		}

		if fn.Name == "getUser" && fn.ReturnComponent != "" {
			t.Fatal("transient property getter typed")
		}

		if fn.Name == "getReporter" && fn.ReturnComponent == "" {
			t.Fatal("singleton property getter not typed")
		}
	}

	property := r.InjectionPropertyLookup(consumer)
	for _, attrs := range []map[string]string{{"default": ""}, {"type": "string"}, {"setter": "false"}} {
		if property("reporter", attrs) != "" {
			t.Errorf("ineligible property typed: %v", attrs)
		}
	}

	if property("reporter", map[string]string{"type": "any"}) == "" {
		t.Fatal("any property incorrectly omitted")
	}

	overridden, _ := writeDIApp(t, `component extends="framework.one" { variables.framework={diConfig:{omitTypedProperties:false,omitDefaultedProperties:false}}; }`, "")

	getter := overridden.InjectionPropertyLookup(filepath.Join(overridden.WorkspaceFolders[0], "model", "services", "Consumer.cfc"))
	if getter("reporter", map[string]string{"type": "string", "default": ""}) == "" {
		t.Fatal("explicit property policy ignored")
	}
}

func TestDI1PropertyMetadataControlsExplicitSetters(t *testing.T) {
	for _, test := range []struct {
		name, attrs, componentAttrs string
		injected                    bool
	}{
		{"default", `default=""`, `accessors=true`, false},
		{"typed", `type="string"`, `accessors=true`, false},
		{"any", `type="any"`, `accessors=true`, true},
		{"disabled implicit", `default="" setter=false`, `accessors=true`, true},
		{"no accessors", `default=""`, ``, true},
		{"persistent", `default=""`, `persistent=true`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, dir := writeDIApp(t, `component extends="framework.one" {}`, "")
			file := filepath.Join(dir, "model", "services", "Consumer.cfc")
			source := `component ` + test.componentAttrs + ` {
 function setReporter(reporter) { variables.reporter=arguments.reporter; }
 property name="reporter" ` + test.attrs + `;
 }`

			pr := parser.ParseWithOptions(uri.File(file), source, &parser.ParseOptions{SetterLookup: r.SetterLookup(file), BeanLookup: r.InjectionBeanLookup(file), PropertyBeanLookup: r.InjectionPropertyLookup(file)})
			for _, fn := range pr.Funcs {
				if fn.Name == "setReporter" && (fn.Arguments[0].Component != "") != test.injected {
					t.Fatalf("setter: %+v", fn)
				}
			}

			pr.ApplyFullReplace(`component accessors=true { property name="reporter"; function setReporter(reporter) {variables.reporter=arguments.reporter;} }`)

			for _, fn := range pr.Funcs {
				if fn.Name == "setReporter" && fn.Arguments[0].Component == "" {
					t.Fatal("property edit did not refresh injection")
				}
			}

			pr.ApplyFullReplace(source)

			for _, fn := range pr.Funcs {
				if fn.Name == "setReporter" && (fn.Arguments[0].Component != "") != test.injected {
					t.Fatal("property edit retained stale injection")
				}
			}
		})
	}
}

func TestDI1SelectedStartupRegistrations(t *testing.T) {
	for _, mode := range []string{"inherited", "override", "super", "guarded"} {
		t.Run(mode, func(t *testing.T) {
			child := `component extends="parents.Base" {}`

			if mode != "inherited" {
				body := ""
				if mode == "super" {
					body = "super.onApplicationStart();"
				}

				if mode == "guarded" {
					body = "if (flag) {super.onApplicationStart();}"
				}

				child = `component extends="parents.Base" {function onApplicationStart(){` + body + `}}`
			}

			r, dir := writeDIApp(t, child, "")
			write := startupFixtureWriter(t, dir)
			write("parents/Base.cfc", `component {
 application.factory=new framework.ioc("model");
 function unused(){application.factory.declareBean("never","model.beans.User");include "unused.cfm";}
 function onApplicationStart(){application.factory.declareBean("forced","model.beans.User");}
 }`)
			write("parents/unused.cfm", `<cfset application.factory.declareBean("includedUnused","model.beans.User")>`)

			lookup := r.SetterLookup(filepath.Join(dir, "model", "services", "Consumer.cfc"))
			if lookup == nil {
				t.Fatal("constructor factory missing")
			}

			want := mode == "inherited" || mode == "super"
			if got := lookup("forced"); (got != "") != want {
				t.Fatalf("selected registration: %q, want %v", got, want)
			}

			for _, name := range []string{"never", "includedUnused"} {
				if got := lookup(name); got != "" {
					t.Fatalf("unused registration %s: %s", name, got)
				}
			}
		})
	}
}

func TestFW1ConfiguredControllersAndSelectedHooks(t *testing.T) {
	for _, hook := range []string{"setupApplication", "unused"} {
		r, dir := writeDIApp(t, `component extends="framework.one" {
 variables.framework={controllersFolder:"handlers"};
 function `+hook+`(){setBeanFactory(custom);}
 }`, "")
		startupFixtureWriter(t, dir)("handlers/Handler.cfc", `component {}`)

		lookup := r.ConstructorLookup(filepath.Join(dir, "model", "services", "Consumer.cfc"))
		if hook == "setupApplication" {
			if lookup != nil {
				t.Fatal("custom factory acquired automatic DI/1 policy")
			}

			continue
		}

		if lookup == nil {
			t.Fatal("unused method disabled automatic DI/1")
		}

		if lookup("reporter") == "" {
			t.Fatal("model root omitted")
		}

		if r.ConstructorLookup(filepath.Join(dir, "handlers", "Handler.cfc")) == nil {
			t.Fatal("configured controllers omitted")
		}

		if r.ConstructorLookup(filepath.Join(dir, "controllers", "Controller.cfc")) != nil {
			t.Fatal("default controllers retained")
		}
	}
}

func TestFW1InheritedAutomaticPolicy(t *testing.T) {
	for _, mode := range []string{"inherited", "unquoted", "custom factory", "missing parent", "cycle", "wrong framework"} {
		t.Run(mode, func(t *testing.T) {
			r, dir := writeDIApp(t, `component extends="parents.Base" {}`, "")
			write := startupFixtureWriter(t, dir)

			base := `component extends="framework.one" {variables.framework={controllersFolder:"handlers"};}`
			if mode == "unquoted" {
				base = `component extends=framework.one {variables.framework={controllersFolder:"handlers"};}`
			}

			if mode == "custom factory" {
				base = `component extends="framework.one" {function setupApplication(){setBeanFactory(custom);}}`
			}

			if mode == "cycle" {
				base = `component extends="parents.Base" {}`
			}

			if mode == "wrong framework" {
				base = `component extends="other.one" {}`
			}

			if mode != "missing parent" {
				write("parents/Base.cfc", base)
			}

			write("handlers/Handler.cfc", `component {}`)

			lookup := r.ConstructorLookup(filepath.Join(dir, "model", "services", "Consumer.cfc"))
			if mode != "inherited" && mode != "unquoted" {
				if lookup != nil {
					t.Fatal("unproved ancestry or custom factory acquired automatic DI")
				}

				return
			}

			if lookup == nil || lookup("reporter") == "" {
				t.Fatal("inherited policy omitted model")
			}

			if r.ConstructorLookup(filepath.Join(dir, "handlers", "Handler.cfc")) == nil {
				t.Fatal("inherited controllersFolder omitted")
			}
		})
	}
}

func TestFW1GuardedConfigurationDoesNotEstablishPolicy(t *testing.T) {
	for _, settings := range []string{
		`if(flag){variables.framework={controllersFolder:"handlers"};}`,
		`if(flag) variables.framework={controllersFolder:"handlers"};`,
		`if(flag){variables.framework.controllersFolder="handlers";}`,
		`function setupApplication(){variables.framework={controllersFolder:"handlers"};}`,
	} {
		r, dir := writeDIApp(t, `component extends="framework.one" {`+settings+`}`, "")
		if lookup := r.ConstructorLookup(filepath.Join(dir, "model", "services", "Consumer.cfc")); lookup != nil {
			t.Fatal("guarded configuration established automatic DI/1", settings)
		}
	}

	r, dir := writeDIApp(t, `<cfcomponent extends="framework.one"><cfset variables.framework={controllersFolder:"handlers"}></cfcomponent>`, "")
	if lookup := r.ConstructorLookup(filepath.Join(dir, "model", "services", "Consumer.cfc")); lookup != nil {
		t.Fatal("unsupported tag configuration silently became defaults")
	}
}

func TestFW1ConfigurationUsesOrderedConstructorWrites(t *testing.T) {
	for _, test := range []struct {
		name, settings, folder string
	}{
		{"member override", `variables.framework={controllersFolder:"controllers"}; variables.framework.controllersFolder="handlers";`, "handlers"},
		{"case insensitive", `variables.framework={controllersFolder:"controllers"}; variables.framework.CONTROLLERSFOLDER="handlers";`, "handlers"},
		{"reverse order", `variables.framework.controllersFolder="handlers"; variables.framework.controllersFolder="controllers";`, "controllers"},
		{"whole replacement", `variables.framework={controllersFolder:"handlers"}; variables.framework={controllersFolder:"controllers"};`, "controllers"},
		{"disable then enable", `variables.framework={diEngine:"wirebox"}; variables.framework.diEngine="di1";`, "controllers"},
		{"literal bracket", `variables.framework={controllersFolder:"controllers"}; variables.framework["CONTROLLERSFOLDER"]="handlers";`, "handlers"},
		{"scoped literal brackets", `variables["framework"]={}; variables["framework"]["controllersFolder"]="handlers";`, "handlers"},
		{"mixed literal brackets", `variables["framework"]={}; variables["framework"].controllersFolder="handlers";`, "handlers"},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, dir := writeDIApp(t, `component extends="framework.one" {`+test.settings+`}`, "")
			write := startupFixtureWriter(t, dir)
			write("handlers/Consumer.cfc", `component {}`)
			write("controllers/Consumer.cfc", `component {}`)

			lookup := r.ConstructorLookup(filepath.Join(dir, test.folder, "Consumer.cfc"))
			if lookup == nil || lookup("reporter") == "" {
				t.Fatal("last constructor write did not establish the selected policy")
			}

			other := "handlers"
			if test.folder == other {
				other = "controllers"
			}

			if r.ConstructorLookup(filepath.Join(dir, other, "Consumer.cfc")) != nil {
				t.Fatal("superseded controller root retained")
			}
		})
	}

	for _, settings := range []string{
		`variables.framework={diEngine:"di1"}; variables.framework.diEngine="wirebox";`,
		`variables.framework={controllersFolder:"handlers"}; variables.framework.controllersFolder=settings.folder;`,
		`variables.framework={diConfig:{}}; variables.framework.diConfig.constants={reporter:"scalar"};`,
		`variables.framework={controllersFolder:"handlers"}; if(flag) variables.framework.controllersFolder="controllers";`,
		`variables.framework={}; variables.framework[key]="handlers";`,
		`variables.framework={}; alias=variables.framework; alias.controllersFolder="handlers";`,
		`variables.framework={}; alias=variables["framework"]; alias.controllersFolder="handlers";`,
		`variables.framework={}; mutate(variables.framework);`,
		`variables.framework={}; mutate(variables.framework,"a","b","c","d","e","f","g","h","i","j");`,
		`variables.framework={}; variables.framework.extra=mutate(variables.framework);`,
		`variables.framework={}; variables.framework["diConfig"]["constants"]={reporter:"scalar"};`,
	} {
		r, dir := writeDIApp(t, `component extends="framework.one" {`+settings+`}`, "")
		if r.ConstructorLookup(filepath.Join(dir, "model", "services", "Consumer.cfc")) != nil {
			t.Fatal("unsupported or non-DI/1 final configuration established automatic DI", settings)
		}
	}
}

// Conditional retrieval exposes only an API common to every registered target.
// Singleton injection still requires one exact target.
func TestDI1ConditionalRetrievalKeepsInjectionExact(t *testing.T) {
	startup := `<cfscript>f=new framework.ioc("/model",{recurse:true});if(engine){f.addAlias("selected","Reporter");}else{f.addAlias("selected","Job");}</cfscript>`
	r, dir := writeDIApp(t, `component {this.mappings["/model"]=expandPath("./model");}`, startup)

	job := filepath.Join(dir, "model", "jobs", "Job.cfc")
	if err := os.WriteFile(job, []byte(`component extends="model.services.Reporter" {function subclassOnly(){}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	r.Index.RemoveFile(uri.File(job))
	r.InvalidatePaths()

	consumer := filepath.Join(dir, "model", "services", "Consumer.cfc")

	constructor := r.ConstructorLookup(consumer)
	if constructor == nil {
		t.Fatal("missing factory lookup")
	}

	want := filepath.Join(dir, "model", "services", "Reporter.cfc")
	if got := constructor("selected"); got != want {
		t.Fatalf("retrieval %q, want %q", got, want)
	}

	setter := r.SetterLookup(consumer)
	if setter == nil {
		t.Fatal("missing setter lookup")
	}

	if got := setter("selected"); got != "" {
		t.Fatalf("conditional injection gained a target: %q", got)
	}
}
