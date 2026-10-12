package resolve

import (
	"path/filepath"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

func TestReturnedStructValuesHaveSeparateCollectionContract(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc": `component {function ready(){}}`,
		"Registry.cfc": `component {
 variables.items=structNew();
 function fill(){variables.items[key]=new Item();}
 struct function all(){return variables.items;}
 }`,
		"Use.cfc": `component {function run(){
 var registry=new Registry();
 registry.all().each(function(key,value){key.findNoCase("x");value.ready();value.absent();});
 var entries=registry.all();
 entries.each(function(key,entry){entry.ready();});
 var values=registry.all().valueArray();
 for(var arrayEntry in values){arrayEntry.ready();arrayEntry.absent();}
 structEach(registry.all(),function(key,globalValue){globalValue.ready();});
 var globalValues=structValueArray(registry.all());
 for(var globalEntry in globalValues){globalEntry.ready();}
 var mutated=registry.all();
 mutated[other]=unknown;
 mutated.each(function(key,changed){changed.ready();});
 var escaped=registry.all();
 mutate(escaped);
 escaped.each(function(key,escapedValue){escapedValue.ready();});
 var sameLine=registry.all();
 sameLine[other]=unknown;sameLine.each(function(key,sameLineValue){sameLineValue.ready();});
 registry.all().map(function(key,value){return unknown;}).each(function(key,mapped){mapped.ready();});
 }}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"key.findNoCase", "value.ready", "entry.ready", "arrayEntry.ready", "globalValue.ready", "globalEntry.ready"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"value.absent", "arrayEntry.absent", "changed.ready", "escapedValue.ready", "sameLineValue.ready", "mapped.ready"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}
}

func TestReturnedStructValuesFollowCurrentProducerContracts(t *testing.T) {
	dir := t.TempDir()
	source := `component {function run(){
 var registry=new Registry();
 registry.all().each(function(key,value){value.ready();});
 }}`
	writeFiles(t, dir, map[string]string{
		"Item.cfc":     `component {function ready(){}}`,
		"Producer.cfc": `component {Item function read(){}}`,
		"Registry.cfc": `component {
 variables.items={};variables.producer=new Producer();
 function fill(){variables.items[key]=variables.producer.read();}
 struct function all(){return variables.items;}
 }`,
		"Use.cfc": source,
	})

	r := &Resolver{}

	got := reasonsWith(t, r, dir, "Use.cfc")
	if reason := got["value.ready"]; reason != "" {
		t.Fatal(reason)
	}

	pr := parser.ParseWithOptions(cfpath.ToURI(filepath.Join(dir, "Use.cfc")), source, &parser.ParseOptions{ExtractCalls: true})
	r.Index.IndexFile(cfpath.ToURI(filepath.Join(dir, "Producer.cfc")), `component {function read(){}}`)

	for _, call := range pr.AllCalls() {
		if call.Variable == "value" && call.FuncName == "ready" {
			if reason := r.CanResolveCall(&call, pr, dir); reason == "" {
				t.Fatal("stale producer contract survived an index update")
			}

			return
		}
	}

	t.Fatal("value call missing")
}

func TestReturnedStructValuesRejectInheritedInitializerOverride(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc": `component {function ready(){}}`,
		"Base.cfc": `component {array function structNew(){return [];}}`,
		"Registry.cfc": `component extends="Base" {
 variables.items=structNew();
 function fill(){variables.items[key]=new Item();}
 function all(){return variables.items;}
 }`,
		"Use.cfc": `component {function run(){
 var registry=new Registry();
 registry.all().each(function(value,index){index.ready();});
 }}`,
	})
	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")

	if reason, ok := got["index.ready"]; !ok || reason == "" {
		t.Fatalf("array index acquired struct value type: %q present=%v", reason, ok)
	}
}
