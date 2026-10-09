package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

func TestLiteralSwitchProducer(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Product.cfc": `component {function work(){}}`,
		"Other.cfc":   `component {function other(){}}`,
		"Factory.cfc": `component {
 function make(name="product") {
  switch(arguments.name) {
   case "product": return new Product();
   case "other": return new Other();
   default: return unknown;
  }
 }
 function forwarded(name) {return make(arguments.name);}
 function assigned(name) {
  var selected=arguments.name;
  var result=unknown;
  switch(selected) {
   case "product": result=new Product(); break;
   case "alias":
   case "other": result=new Other(); break;
   default: result=new Product();
  }
  return result;
 }
 function fallthrough(name) {
  var result=unknown;
  switch(name) {
   default: result=new Product();
   case "other": result=new Other(); break;
   case "product": result=new Product();
  }
  return result;
 }
 function nested(name) {
  switch(name) {case "product": switch(name) {case "product": return new Product();}}
 }
 function fallback(name) {
  switch(name) {case "product": return new Product(); break;}
  return this;
 }
}`,
		"Page.cfc": `component {function run(){var factory=new Factory();
 var item=factory.make("product");
 item.work();
 factory.make("product").missing();
}}`,
		"TagPage.cfm": `<cfset factory=new Factory()><cfset item=factory.make("product")><cfset item.work()>`,
		"Mixed.cfc": `<cfcomponent><cffunction name="make"><cfargument name="name"><cfscript>
 switch(arguments.name) {case "product": return new Product(); default: return unknown;}
 </cfscript></cffunction></cfcomponent>`,
	})

	r := &Resolver{}
	expectReasons(t, reasonsWith(t, r, dir, "Page.cfc"), map[string]string{
		"item.work": "", "factory.make.missing": "method 'missing' not found in Product",
	})
	expectReasons(t, reasonsWith(t, r, dir, "TagPage.cfm"), map[string]string{"item.work": ""})

	lookup := r.FuncLookup(dir)
	for _, tc := range []struct{ component, call, want string }{
		{"Factory", `make()`, "Product"},
		{"Factory", `make("PRODUCT")`, "Product"},
		{"Factory", `make(name="other")`, "Other"},
		{"Factory", `make("absent")`, ""},
		{"Factory", `make(unknown)`, ""},
		{"Factory", `make(1)`, ""},
		{"Factory", `make("01")`, ""},
		{"Factory", `forwarded("product")`, "Product"},
		{"Factory", `assigned("product")`, "Product"},
		{"Factory", `assigned("alias")`, "Other"},
		{"Factory", `assigned("absent")`, "Product"},
		{"Factory", `assigned(unknown)`, ""},
		{"Factory", `fallthrough("absent")`, "Other"},
		{"Factory", `fallthrough("product")`, "Product"},
		{"Factory", `nested("product")`, "Product"},
		{"Factory", `nested("absent")`, ""},
		{"Factory", `fallback("product")`, "Product"},
		{"Factory", `fallback("absent")`, "Factory"},
		{"Factory", `fallback(unknown)`, ""},
		{"Mixed", `make("product")`, "Product"},
	} {
		got := lookup(tc.component, parser.CallHop(tc.call))
		if tc.want == "" {
			if got != "" {
				t.Errorf("%s.%s guessed %q", tc.component, tc.call, got)
			}
		} else if r.ComponentPath(got, dir) != filepath.Join(dir, tc.want+".cfc") {
			t.Errorf("%s.%s return %q, want %s", tc.component, tc.call, got, tc.want)
		}
	}
}

func TestSwitchProducerBoundaries(t *testing.T) {
	for _, body := range []string{
		`switch(name){case computed: return new Product();}`,
		`switch(name){case "#computed#": return new Product();}`,
		`switch(name){case 1: return new Product();}`,
		`switch(name){case "product": if(flag){break;} return new Product();}`,
		`switch(name){case "product": continue; return new Product();}`,
		`switch(name){case "product": return new Product(); case "PRODUCT": return new Other();}`,
		`switch(name){case 'product': return new Product(); case "PRODUCT": return new Other();}`,
		`switch(name){default: return new Product(); default: return new Other();}`,
		`switch((name=unknown)){case "product": return new Product();}`,
		`switch(mutate(arguments)){case "product": return new Product();}`,
		`switch(name){case "product": var bean=new Product(); mutate(local); return bean;}`,
		`switch(name){case "product": var bean=new Product(); break;} return unknown;`,
	} {
		t.Run(body, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"Product.cfc": `component {function work(){}}`, "Other.cfc": `component {}`,
				"Factory.cfc": `component {function make(name){` + body + `}}`,
				"Page.cfc":    `component {function run(){var factory=new Factory();factory.make("product").work();}}`,
			})

			r := &Resolver{}
			reasonsWith(t, r, dir, "Page.cfc")

			if got := r.FuncLookup(dir)("Factory", parser.CallHop(`make("product")`)); got != "" {
				t.Fatalf("unsafe switch inference %q", got)
			}
		})
	}
}

func TestSwitchProducerRefreshesSource(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Product.cfc": `component {function work(){}}`, "Other.cfc": `component {}`,
		"Factory.cfc": `component {function make(name){switch(name){case "product": return new Product();}}}`,
		"Page.cfc":    `component {function run(){var factory=new Factory();factory.make("product").work();}}`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Page.cfc")

	lookup := r.FuncLookup(dir)
	if got := lookup("Factory", parser.CallHop(`make("product")`)); r.ComponentPath(got, dir) != filepath.Join(dir, "Product.cfc") {
		t.Fatalf("initial switch return %q", got)
	}

	if err := os.WriteFile(filepath.Join(dir, "Factory.cfc"), []byte(`component {function make(name){switch(name){case "product": return new Other();}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := lookup("Factory", parser.CallHop(`make("product")`)); r.ComponentPath(got, dir) != filepath.Join(dir, "Other.cfc") {
		t.Fatalf("stale switch return %q", got)
	}
}
