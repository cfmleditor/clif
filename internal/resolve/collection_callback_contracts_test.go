package resolve

import (
	"strings"
	"testing"
)

func TestCollectionCallbackInvocationContracts(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Provider.cfc": `component {function filter(callback){callback(new Item());}}`,
		"Item.cfc":     `component persistent="true" entityname="LoadedItem" {function ready(){}}`,
		"Use.cfc": `component {function run(){
 var rows=entityLoad("LoadedItem");
 rows.filter(function(filtered){filtered.ready();filtered.absent();});
 rows.map(function(mapped){mapped.ready();});
 rows.some(function(someItem){someItem.ready();});
 rows.every(function(everyItem){everyItem.ready();});
 var values=[];
 values.filter(function(value,position,collection){position.toString();collection.len();value.ready();});
 var record=structNew();
 record.filter(function(key,value,collection){key.findNoCase("a");collection.keyArray();value.ready();});
 var query=queryNew("id");
 query.each(function(row,position,collection){row.keyArray();position.toString();collection.columnArray();});
 record.filter(function(changed){changed=unknown;changed.findNoCase("a");});
 arrayFilter(rows,function(globalItem){globalItem.ready();globalItem.absent();});
 structEach(record,function(globalKey,globalValue){globalKey.findNoCase("a");globalValue.ready();});
 queryEach(query,function(globalRow){globalRow.keyArray();});
 var custom=new Provider();
 custom.filter(function(customValue){customValue.ready();customValue.absent();});
 record.keyArray().each(function(erased,index,allKeys){allKeys.len();index.toString();erased.ready();});
 record.missing().each(function(broken,index,badCollection){badCollection.len();});
 var unknown=runtimeList();
 unknown.filter(function(untyped){untyped.ready();});
 }}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"filtered.ready", "mapped.ready", "someItem.ready", "everyItem.ready", "collection.len", "position.toString", "key.findNoCase", "collection.keyArray", "row.keyArray", "collection.columnArray", "customValue.ready", "allKeys.len", "index.toString", "globalItem.ready", "globalKey.findNoCase", "globalRow.keyArray"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"filtered.absent", "value.ready", "changed.findNoCase", "untyped.ready", "customValue.absent", "erased.ready", "badCollection.len", "globalItem.absent", "globalValue.ready"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed, got %q present=%v", name, reason, ok)
		}
	}

	if !strings.Contains(got["filtered.absent"], "not found in Item") {
		t.Fatal(got["filtered.absent"])
	}
}

func TestGlobalCollectionCallbackRejectsShadowedAndReassignedParameters(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Use.cfc": `component {
 function queryEach(value,callback){callback(runtimeValue);}
 function run(){
 queryEach(unknown,function(shadowed){shadowed.keyArray();});
 structEach(unknown,function(changed){changed=runtimeValue;changed.findNoCase("a");});
 arrayEach(unknown,function(erased,index,collection){erased.ready();index.toString();collection.len();});
 }
 }`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"shadowed.keyArray", "changed.findNoCase", "erased.ready"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"index.toString", "collection.len"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}
}

func TestSpeculativeCollectionDoesNotEstablishCallbackContract(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {function run(){unknown.each(function(key,value){value.ready();});}}`})
	r := &Resolver{SpeculativeReceivers: map[SpeculativeReceiver]string{{File: dir + "/Use.cfc", Caller: "run", Variable: "unknown"}: "$builtin.native.array"}}

	got := reasonsWith(t, r, dir, "Use.cfc")
	if reason := got["value.ready"]; reason != "variable 'value' has no component ref" {
		t.Fatalf("guessed array manufactured a callback index: %q", reason)
	}
}
