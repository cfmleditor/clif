package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

func TestTypedRuntimeFactoryUsesPrivateInstanceState(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "resolution-runtime", "instance-state", "Factory.cfc"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc":       `component {}`,
		"B.cfc":       `component {}`,
		"Factory.cfc": string(fixture),
		"Use.cfc": `component {
 function initial(){var factory=new Factory({service:"A"});return factory.get("service");}
 function replaced(){var factory=new Factory({service:"A"});factory.replace({service:"B"});return factory.get("service");}
 function forwarded(){var factory=new Factory({service:"A"});factory.select("service").to("B");return factory.get("service");}
 function forwardedChain(){return new Factory({service:"A"}).mapInstance("service").to("B").get("service");}
 function unknownForward(name){var factory=new Factory({service:"A"});factory.select(arguments.name).to("B");return factory.get("service");}
 }`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	for _, tc := range []struct{ method, want string }{{"initial", "A"}, {"replaced", "B"}, {"forwarded", "B"}, {"forwardedChain", "B"}} {
		if got := r.FuncLookup(dir)("Use", tc.method); r.ComponentPath(got, dir) != filepath.Join(dir, tc.want+".cfc") {
			t.Fatalf("%s got %s", tc.method, got)
		}
	}

	if got := r.FuncLookup(dir)("Use", "unknownForward"); got != "" {
		t.Fatalf("unknown forwarded selector guessed %s", got)
	}
}

func TestInstanceForwardingRequiresClosedUnshadowedSetter(t *testing.T) {
	for _, alias := range []string{
		`function alias(name){return alias(arguments.name);}`,
		`function alias(name){return other.alias(arguments.name);}`,
		`function alias(name){return map(unknown(arguments.name));}`,
		`function alias(name){return map(arguments.name).to("B");}`,
		`function alias(map){return map(arguments.map);}`,
		`function alias(name){return map(argumentCollection=variables);}`,
		`public string function alias(name){return map(arguments.name);}`,
		`function alias(name){return second(arguments.name);}function second(name){return alias(arguments.name);}`,
	} {
		t.Run(alias, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"A.cfc": `component {}`,
				"B.cfc": `component {}`,
				"Factory.cfc": `component {
 function init(){variables.path="A";return this;}
 function map(name){variables.path=arguments.name;return this;}
 function get(){return createObject("component",variables.path);}
 ` + alias + `}`,
				"Use.cfc": `component {function read(){var factory=new Factory();factory.alias("B");return factory.get();}}`,
			})

			r := &Resolver{}
			reasonsWith(t, r, dir, "Use.cfc")

			if got := r.FuncLookup(dir)("Use", "read"); got != "" {
				t.Fatalf("unsupported alias guessed %s", got)
			}
		})
	}
}

func TestInstanceSnapshotsRequireCompatibleConstructorArguments(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc":               `component {}`,
		"StringFactory.cfc":   `component {public any function init(string bindings){variables.bindings=arguments.bindings;return this;}public any function get(){return createObject("component",variables.bindings["service"]);}}`,
		"RequiredFactory.cfc": `component {public any function init(required any unused){variables.path="A";return this;}public any function get(){return createObject("component",variables.path);}}`,
		"Use.cfc": `component {
 function wrongType(){var factory=new StringFactory({service:"A"});return factory.get();}
 function missing(){var factory=new RequiredFactory();return factory.get();}
 }`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	for _, method := range []string{"wrongType", "missing"} {
		if got := r.FuncLookup(dir)("Use", method); got != "" {
			t.Fatalf("%s guessed %s", method, got)
		}
	}
}

func TestOrderedPrivateRegistrationWrites(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component {}`,
		"B.cfc": `component {}`,
		"Factory.cfc": `component {
 public any function init(){variables.bindings={};return this;}
 public any function bind(string name,string path){variables.bindings[arguments.name]=arguments.path;return this;}
 public any function get(string name){return createObject("component",variables.bindings[arguments.name]);}
 }`,
		"Use.cfc": `component {
 function registered(){var factory=new Factory();factory.bind("service","A");return factory.get("service");}
 function rebound(){var factory=new Factory();factory.bind("service","A");factory.bind("SERVICE","B");return factory.get("service");}
 function chained(){return new Factory().bind("service","A").bind("service","B").get("service");}
 function unknownKey(name){var factory=new Factory();factory.bind(arguments.name,"A");return factory.get("service");}
 function unknownValue(path){var factory=new Factory();factory.bind("service",arguments.path);return factory.get("service");}
 function uncertain(){var factory=new Factory();factory.bind("service","A");if(flag){factory.bind("service","B");}return factory.get("service");}
 function escaped(){var factory=new Factory();factory.bind("service","A");unknown(factory);return factory.get("service");}
 }`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	for _, tc := range []struct{ method, want string }{
		{"registered", "A"},
		{"rebound", "B"},
		{"chained", "B"},
		{"unknownKey", ""},
		{"unknownValue", ""},
		{"uncertain", ""},
		{"escaped", ""},
	} {
		got := r.FuncLookup(dir)("Use", tc.method)
		if tc.want == "" {
			if got != "" {
				t.Fatalf("%s guessed %s", tc.method, got)
			}
		} else if r.ComponentPath(got, dir) != filepath.Join(dir, tc.want+".cfc") {
			t.Fatalf("%s got %s", tc.method, got)
		}
	}
}

func TestClosedFluentPrivateRegistrations(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component {}`,
		"B.cfc": `component {}`,
		"Factory.cfc": `component {
 function init(){variables.bindings={};variables.current="";return this;}
 function map(string name){variables.current=arguments.name;return this;}
 function to(string path){variables.bindings[variables.current]=arguments.path;variables.current="";return this;}
 function register(string name,string path){variables.current=arguments.name;variables.bindings[variables.current]=arguments.path;variables.current="";return this;}
 function get(string name){return createObject("component",variables.bindings[arguments.name]);}
 }`,
		"Use.cfc": `component {
 function fluent(){var factory=new Factory();factory.map("service").to("A");return factory.get("service");}
 function rebound(){var factory=new Factory();factory.map("service").to("A");factory.map("SERVICE").to("B");return factory.get("service");}
 function ordered(){var factory=new Factory();factory.register("service","A");return factory.get("service");}
 function expressionChain(){return new Factory().map("service").to("B").get("service");}
 function independent(){var first=new Factory();var second=new Factory();first.map("service");second.map("other");first.to("A");second.to("B");return first.get("service");}
 function reset(){var factory=new Factory();factory.map("service").to("A");factory.to("B");return factory.get("service");}
 function uncertain(name){var factory=new Factory();factory.map(arguments.name).to("A");return factory.get("service");}
 function nested(){var factory=new Factory();factory.map(unknown()).to("A");return factory.get("service");}
 function unknownHop(){var factory=new Factory();factory.map("service").unknown().to("A");return factory.get("service");}
 function guarded(){var factory=new Factory();factory.map("service").to("A");if(flag){factory.map("service").to("B");}return factory.get("service");}
 }`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	for _, tc := range []struct{ method, want string }{
		{"fluent", "A"},
		{"rebound", "B"},
		{"ordered", "A"},
		{"expressionChain", "B"},
		{"independent", "A"},
		{"reset", "A"},
		{"uncertain", ""},
		{"nested", ""},
		{"unknownHop", ""},
		{"guarded", ""},
	} {
		got := r.FuncLookup(dir)("Use", tc.method)
		if tc.want == "" {
			if got != "" {
				t.Fatalf("%s guessed %s", tc.method, got)
			}
		} else if r.ComponentPath(got, dir) != filepath.Join(dir, tc.want+".cfc") {
			t.Fatalf("%s got %s", tc.method, got)
		}
	}
}

func TestTypedClosedSettersApplyTheirWrites(t *testing.T) {
	for _, returnType := range []string{"any", "Factory", "string", "Other"} {
		t.Run(returnType, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"A.cfc":     `component {}`,
				"B.cfc":     `component {}`,
				"Other.cfc": `component {}`,
				"Factory.cfc": `component {
 public Factory function init(){variables.path="A";return this;}
 public ` + returnType + ` function replace(string path){variables.path=arguments.path;return this;}
 public any function get(){return createObject("component",variables.path);}
 }`,
				"Use.cfc": `component {
 function replaced(){var factory=new Factory();factory.replace("B");return factory.get();}
 function chained(){return new Factory().replace("B").get();}
 function unknown(path){var factory=new Factory();factory.replace(arguments.path);return factory.get();}
 }`,
			})

			r := &Resolver{}
			reasonsWith(t, r, dir, "Use.cfc")

			for _, name := range []string{"replaced", "chained", "unknown"} {
				got := r.FuncLookup(dir)("Use", name)
				if name == "unknown" || returnType == "string" || returnType == "Other" {
					if got != "" {
						t.Fatalf("%s acquired an incompatible or unknown setter result: %s", name, got)
					}
				} else if r.ComponentPath(got, dir) != filepath.Join(dir, "B.cfc") {
					t.Fatalf("%s ignored setter writes: %s", name, got)
				}
			}
		})
	}
}

func TestPrivateInstanceDefaultsMustBeClosed(t *testing.T) {
	for _, defaultValue := range []string{`"literal"`, `{key:"literal"}`, `["literal"]`, `unknown(this)`, `mutate()`, `variables.shared`, `"#unknown(this)#"`} {
		t.Run(defaultValue, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"A.cfc": `component {}`,
				"Factory.cfc": `component {
 function init(){variables.path="A";return this;}
 function mutate(){variables.path="B";return this;}
 function get(any unused=` + defaultValue + `){return createObject("component",variables.path);}
 }`,
				"Use.cfc": `component {function value(){return new Factory().get();}}`,
			})

			r := &Resolver{}
			reasonsWith(t, r, dir, "Use.cfc")

			got := r.FuncLookup(dir)("Use", "value")
			if defaultValue == `"literal"` || defaultValue == `{key:"literal"}` || defaultValue == `["literal"]` {
				if r.ComponentPath(got, dir) != filepath.Join(dir, "A.cfc") {
					t.Fatalf("literal default withheld the snapshot: %s", got)
				}
			} else if got != "" {
				t.Fatalf("effectful or contextual default retained a private snapshot: %s", got)
			}
		})
	}
}

func TestAliasedPrivateRegistrationTablesWithholdState(t *testing.T) {
	for _, alias := range []string{"variables.alias=variables.bindings;", "variables.alias=arguments.bindings;"} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{
			"A.cfc": `component {}`,
			"B.cfc": `component {}`,
			"Factory.cfc": `component {
 function init(bindings){variables.bindings=arguments.bindings;` + alias + `return this;}
 function bind(name,path){variables.bindings[arguments.name]=arguments.path;return this;}
 function get(name){return createObject("component",variables.alias[arguments.name]);}
 }`,
			"Use.cfc": `component {function aliased(){var factory=new Factory({service:"A"});factory.bind("service","B");return factory.get("service");}}`,
		})

		r := &Resolver{}
		reasonsWith(t, r, dir, "Use.cfc")

		if got := r.FuncLookup(dir)("Use", "aliased"); got != "" {
			t.Fatal("aliased records treated independently", got)
		}
	}

	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component {}`,
		"B.cfc": `component {}`,
		"Factory.cfc": `component {
 function init(){variables.bindings={};variables.alias={};return this;}
 function both(bindings){variables.bindings=arguments.bindings;variables.alias=arguments.bindings;return this;}
 function bind(name,path){variables.bindings[arguments.name]=arguments.path;return this;}
 function get(name){return createObject("component",variables.alias[arguments.name]);}
 }`,
		"Use.cfc": `component {function aliased(){var factory=new Factory();factory.both({service:"A"});factory.bind("service","B");return factory.get("service");}}`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	if got := r.FuncLookup(dir)("Use", "aliased"); got != "" {
		t.Fatal("setter-created aliases treated independently", got)
	}
}

func TestPrivateInstanceStateDoesNotBorrowAnotherApplication(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"app/Application.cfc":         `component {this.mappings["/targets"]=expandPath("./targets");}`,
		"app/targets/Service.cfc":     `component {}`,
		"library/Application.cfc":     `component {this.mappings["/targets"]=expandPath("./targets");}`,
		"library/targets/Service.cfc": `component {}`,
		"library/Factory.cfc":         `component {function init(path){variables.path=arguments.path;return this;}function get(){return createObject("component",variables.path);}}`,
		"app/Use.cfc":                 `component {function make(){var factory=new library.Factory("targets.Service");return factory.get();}}`,
	})
	r := &Resolver{Mappings: map[string]string{"library": filepath.Join(dir, "library")}}
	reasonsWith(t, r, dir, "app/Use.cfc")

	if got := r.FuncLookup(filepath.Join(dir, "app"))("Use", "make"); got != "" {
		t.Fatal("borrowed library application mappings", got)
	}
}

func TestConstructorValuesBelongToTheInstance(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component {function a(){}}`,
		"B.cfc": `component {function b(){}}`,
		"Factory.cfc": `component {
 function init(path){variables.path=arguments.path;return this;}
 function get(){return createObject("component",variables.path);}
 }`,
		"Use.cfc": `component {
 function first(){var factory=new Factory("A");return factory.get();}
 function second(){var factory=new Factory("B");return factory.get();}
 function supplied(path){var factory=new Factory(arguments.path);return factory.get();}
 function escaped(){var factory=new Factory("A");unknown(factory);return factory.get();}
 function reinitialized(){var factory=new Factory("A");factory.init("B");return factory.get();}
 }`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	lookup := r.FuncLookup(dir)
	for _, tc := range []struct{ method, want string }{
		{"first", "A"},
		{"second", "B"},
		{parser.CallHop(`supplied("B")`), "B"},
		{"escaped", ""},
		{"reinitialized", ""},
		{parser.CallHop(`supplied(unknown)`), ""},
	} {
		got := lookup("Use", tc.method)
		if tc.want == "" {
			if got != "" {
				t.Fatalf("%s guessed %s", tc.method, got)
			}

			continue
		}

		if r.ComponentPath(got, dir) != filepath.Join(dir, tc.want+".cfc") {
			t.Fatalf("%s got %s", tc.method, got)
		}
	}
}

func TestMutableFactoryDoesNotRetainConstructorSnapshot(t *testing.T) {
	for _, extra := range []string{
		`function setPath(path){variables.path=runtime(arguments.path);}`,
		`function expose(){return variables;}`,
		`function uncertain(){unknown(variables);return this;}`,
		`function INIT(path){variables.path="B";return this;}`,
		`function GET(){return createObject("component",variables.path);}`,
	} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{
			"A.cfc":       `component {}`,
			"Factory.cfc": `component {function init(path){variables.path=arguments.path;return this;}function get(){return createObject("component",variables.path);}` + extra + `}`,
			"Use.cfc":     `component {function make(){var factory=new Factory("A");return factory.get();}}`,
		})

		r := &Resolver{}
		reasonsWith(t, r, dir, "Use.cfc")

		if got := r.FuncLookup(dir)("Use", "make"); got != "" {
			t.Fatalf("mutable snapshot: %s", got)
		}
	}
}

func TestOrderedPrivateInstanceSetters(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component {}`,
		"B.cfc": `component {}`,
		"Factory.cfc": `component {
 function init(path){variables.path=arguments.path;return this;}
 function setPath(path){variables.path=arguments.path;return this;}
 function get(){return createObject("component",variables.path);}
 function getUnused(value){return createObject("component",variables.path);}
 }`,
		"Use.cfc": `component {
 function ordered(){var factory=new Factory("A");factory.setPath("B");return factory.get();}
 function reverse(){var factory=new Factory("B");factory.setPath("A");return factory.get();}
 function chained(){return new Factory("A").setPath("B").get();}
 function unknown(){var factory=new Factory("A");factory.setPath(runtime);return factory.get();}
 function guarded(){var factory=new Factory("A");if(flag){factory.setPath("B");}return factory.get();}
 function looped(){var factory=new Factory("A");for(x in list){factory.setPath("B");}return factory.get();}
 function published(){application.factory=new Factory("A");return application.factory.get();}
 function stored(){variables.factory=new Factory("A");return variables.factory.get();}
 function hiddenEscape(){var factory=new Factory("A");return factory.getUnused(external(factory));}
 string function external(factory){arguments.factory.setPath("B");return "ignored";}
 }`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	for _, tc := range []struct{ method, want string }{
		{"ordered", "B"},
		{"reverse", "A"},
		{"chained", "B"},
		{"unknown", ""},
		{"guarded", ""},
		{"looped", ""},
		{"published", ""},
		{"stored", ""},
		{"hiddenEscape", ""},
	} {
		got := r.FuncLookup(dir)("Use", tc.method)
		if tc.want == "" {
			if got != "" {
				t.Fatalf("%s guessed %s", tc.method, got)
			}
		} else if r.ComponentPath(got, dir) != filepath.Join(dir, tc.want+".cfc") {
			t.Fatalf("%s got %s", tc.method, got)
		}
	}
}

func TestPrivateLiteralBindingsBelongToTheInstance(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"A.cfc": `component {}`,
		"B.cfc": `component {}`,
		"Factory.cfc": `component {
 function init(bindings){variables.bindings=arguments.bindings;return this;}
 function get(name){return createObject("component",variables.bindings[arguments.name]);}
 function replace(bindings){variables.bindings=arguments.bindings;return this;}
 }`,
		"Use.cfc": `component {
 function first(){var factory=new Factory({service:"A"});return factory.get("service");}
 function second(){var factory=new Factory({SERVICE:"B"});return factory.get("service");}
 function absent(){var factory=new Factory({other:"A"});return factory.get("service");}
 function unknownKey(name){var factory=new Factory({service:"A"});return factory.get(arguments.name);}
 function unknownValue(path){var factory=new Factory({service:arguments.path});return factory.get("service");}
 function nested(){var factory=new Factory({service:{path:"A"}});return factory.get("service");}
 function duplicate(){var factory=new Factory({service:"A",SERVICE:"B"});return factory.get("service");}
 function replaced(){var factory=new Factory({service:"A"});factory.replace({service:"B"});return factory.get("service");}
 function unknownReplacement(){var factory=new Factory({service:"A"});factory.replace(runtime);return factory.get("service");}
 function sharedBindings(){application.bindings={service:"A"};return new Factory(application.bindings).get("service");}
 function escaped(){var factory=new Factory({service:"A"});unknown(factory);return factory.get("service");}
 }`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	for _, tc := range []struct{ method, want string }{
		{"first", "A"},
		{"second", "B"},
		{"replaced", "B"},
		{"unknownReplacement", ""},
		{"sharedBindings", ""},
		{"absent", ""},
		{"unknownKey", ""},
		{"unknownValue", ""},
		{"nested", ""},
		{"duplicate", ""},
		{"escaped", ""},
	} {
		got := r.FuncLookup(dir)("Use", tc.method)
		if tc.want == "" {
			if got != "" {
				t.Fatalf("%s guessed %s", tc.method, got)
			}
		} else if r.ComponentPath(got, dir) != filepath.Join(dir, tc.want+".cfc") {
			t.Fatalf("%s got %s", tc.method, got)
		}
	}
}

func TestExposedLiteralBindingsDoNotRetainSnapshot(t *testing.T) {
	for _, extra := range []string{
		`function expose(){return variables.bindings;}`,
		`function replace(bindings){variables.bindings=arguments.bindings;}`,
		`function mutate(){variables.bindings.service="B";}`,
	} {
		dir := t.TempDir()
		writeFiles(t, dir, map[string]string{
			"A.cfc":       `component {}`,
			"Factory.cfc": `component {function init(bindings){variables.bindings=arguments.bindings;return this;}function get(name){return createObject("component",variables.bindings[arguments.name]);}` + extra + `}`,
			"Use.cfc":     `component {function make(){var factory=new Factory({service:"A"});return factory.get("service");}}`,
		})

		r := &Resolver{}
		reasonsWith(t, r, dir, "Use.cfc")

		if got := r.FuncLookup(dir)("Use", "make"); got != "" {
			t.Fatalf("exposed binding snapshot: %s", got)
		}
	}
}
