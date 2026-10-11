package resolve

import "testing"

func TestUnrelatedClosuresPreserveLocalCollectionResolution(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Item.cfc": `component {function ready(){}}`,
		"Registry.cfc": `component {
 function one(){var items={};items[key]=new Item();return items[key];}
 struct function all(){var items={};items[key]=new Item();return items;}
 function other(){var callback=function(){var items={};items[key]=false;};}
 }`,
		"Use.cfc": `component {function run(){
 var registry=new Registry();
 var selected=registry.one();
 selected.ready();selected.absent();
 registry.all().each(function(key,value){value.ready();value.absent();});
 }}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"selected.ready", "value.ready"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"selected.absent", "value.absent"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}
}
