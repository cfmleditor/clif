package resolve

import (
	"testing"

	"github.com/cfmleditor/clif/internal/nativetype"
)

func TestNativeReturnDeclarationsSupplyRuntimeContracts(t *testing.T) {
	for _, tc := range []struct{ declared, method, native string }{
		{"ARRAY", "len", "array"},
		{"query", "getColumnNames", "query"},
	} {
		t.Run(tc.declared, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{
				"Data.cfc": "component {" + tc.declared + " function value(){return runtimeValue();}}",
				"Use.cfc": `component {function run(){
 var data=new Data();
 var value=data.value();
 value.` + tc.method + `();value.absent();
 data.value().` + tc.method + `();data.value().absent();
 }}`,
			})

			r := &Resolver{}
			got := reasonsWith(t, r, dir, "Use.cfc")

			if ret := r.FuncLookup(dir)("Data", "value"); ret != nativetype.Type(tc.native) {
				t.Errorf("return=%q", ret)
			}

			for _, name := range []string{"value." + tc.method, "data.value." + tc.method} {
				if reason, ok := got[name]; !ok || reason != "" {
					t.Errorf("%s: %q present=%v", name, reason, ok)
				}
			}

			for _, name := range []string{"value.absent", "data.value.absent"} {
				if reason, ok := got[name]; !ok || reason == "" {
					t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
				}
			}
		})
	}
}

func TestNativeReturnDeclarationsPreserveUnknownElementsAndStructComponents(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc": `component {function ready(){} function each(callback){}}`,
		"Data.cfc": `component accessors="true" {
 property name="generated" type="query";
 query function rows(){return runtimeValue();}
 array function items(){return runtimeValue();}
 struct function item(){return new Item();}
 function unknown(){return runtimeValue();}
 }`,
		"Use.cfc": `component {function run(){
 var data=new Data();
 data.items().each(function(value,index,collection){value.ready();index.toString();collection.len();});
 data.rows().each(function(row,index,collection){row.keyExists("id");index.toString();collection.getColumnNames();});
 data.item().ready();
 data.item().each(function(key,value){key.length();value.ready();});
 var unknown=data.unknown();unknown.length();
 }}`,
	})

	r := &Resolver{}
	got := reasonsWith(t, r, dir, "Use.cfc")

	for _, name := range []string{"index.toString", "collection.len", "row.keyExists", "collection.getColumnNames", "data.item.ready"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"value.ready", "key.length"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}

	for _, method := range []string{"unknown", "getGenerated"} {
		if ret := r.FuncLookup(dir)("Data", method); ret != "" && ret != "$any" {
			t.Errorf("%s acquired an unproven contract: %q", method, ret)
		}
	}
}

func TestNativeCollectionDeclarationsRefreshAfterAccessorEdits(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Data.cfc": `component accessors="true" {property name="rows" type="query";}`,
		"Use.cfc":  `component {function run(){var data=new Data();data.getRows();}}`,
	})

	r := &Resolver{}
	reasonsWith(t, r, dir, "Use.cfc")

	fd := r.ResolveFunc("Data", "getRows", dir)
	if got := r.ReturnComponentOf(fd); got != "" {
		t.Fatalf("synthetic getter acquired return contract: %q", got)
	}

	r.Index.IndexFile(fd.URI, `component accessors="true" {
 property name="rows" type="query";
 query function getRows(){return runtimeValue();}
 }`)

	if got := r.ReturnComponentOf(r.ResolveFunc("Data", "getRows", dir)); got != nativetype.Type("query") {
		t.Fatalf("explicit override lost declaration: %q", got)
	}

	r.Index.IndexFile(fd.URI, `component accessors="true" {property name="rows" type="query";}`)

	if got := r.ReturnComponentOf(r.ResolveFunc("Data", "getRows", dir)); got != "" {
		t.Fatalf("stale explicit declaration survived: %q", got)
	}
}

func TestTagNativeCollectionDeclarations(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Data.cfc": `<cfcomponent>
 <cffunction name="rows" returntype="query"><cfreturn runtimeValue()></cffunction>
 <cffunction name="items" returntype="array"><cfreturn runtimeValue()></cffunction>
 </cfcomponent>`,
		"Use.cfc": `component {function run(){
 var data=new Data();data.rows().getColumnNames();data.items().len();
 data.rows().absent();data.items().absent();
 }}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")

	for _, name := range []string{"data.rows.getColumnNames", "data.items.len"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"data.rows.absent", "data.items.absent"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}
}

func TestNativeArrayListContractsPreserveErasedElements(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Items.cfc": `component {function iterator(){} }`,
		"Use.cfc": `component {function run(){
 var items=[];
 items.add("value");items.containsAll([]);items.removeAll([]);items.retainAll([]);
 items.size().toString();
 items.iterator().hasNext();items.iterator().next().ready();
 items.get(0).ready();items.remove(0).ready();items.absent();
 var custom=new Items();custom.iterator().hasNext();
 }}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"items.add", "items.containsAll", "items.removeAll", "items.retainAll", "items.size.toString", "items.iterator.hasNext"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"items.absent", "custom.iterator.hasNext"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}
}
