package resolve

import "testing"

func TestStructKeyArrayElements(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Custom.cfc": `component {array function keyArray(){return [];}}`,
		"Use.cfc": `component {function run(){
 var record=structNew();
 record.keyArray().filter(function(key){key.findNoCase("prefix");key.absent();});
 var selected=record.keyArray().filter(function(key){return true;}).slice(1);
 for(var selectedKey in selected){selectedKey.findNoCase("prefix");}
 structKeyArray(record).each(function(globalKey){globalKey.findNoCase("prefix");});
 structKeyArray().each(function(noInputKey){noInputKey.findNoCase("prefix");});
 var keys=structKeyArray(record);
 for(var loopKey in keys){loopKey.findNoCase("prefix");}
 request.keyArray().filter(function(requestKey){return arguments.requestKey.findNoCase("prefix");});
 var custom=new Custom();
 custom.keyArray().each(function(customKey){customKey.findNoCase("prefix");});
 record.missing().keyArray().each(function(brokenKey){brokenKey.findNoCase("prefix");});
 unknown.keyArray().each(function(unknownKey){unknownKey.findNoCase("prefix");});
 record.keyArray().map(function(key){return unknown;}).each(function(mappedKey){mappedKey.findNoCase("prefix");});
 record.keyArray().each(function(changedKey){changedKey=unknown;changedKey.findNoCase("prefix");});
 }}`,
	})

	got := reasonsWith(t, &Resolver{}, dir, "Use.cfc")
	for _, name := range []string{"key.findNoCase", "selectedKey.findNoCase", "globalKey.findNoCase", "loopKey.findNoCase", "arguments.requestKey.findNoCase"} {
		if reason, ok := got[name]; !ok || reason != "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}

	for _, name := range []string{"key.absent", "noInputKey.findNoCase", "customKey.findNoCase", "brokenKey.findNoCase", "unknownKey.findNoCase", "mappedKey.findNoCase", "changedKey.findNoCase"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s must remain diagnosed: %q present=%v", name, reason, ok)
		}
	}
}

func TestStructKeyArrayRejectsShadowedGlobalAndSpeculativeReceiver(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"Use.cfc": `component {
 array function structKeyArray(value){return [];}
 function run(){
 structKeyArray(unknown).each(function(shadowedKey){shadowedKey.findNoCase("prefix");});
 unknown.keyArray().each(function(guessedKey){guessedKey.findNoCase("prefix");});
 }
 }`})
	r := &Resolver{SpeculativeReceivers: map[SpeculativeReceiver]string{{File: dir + "/Use.cfc", Caller: "run", Variable: "unknown"}: "$builtin.native.struct"}}

	got := reasonsWith(t, r, dir, "Use.cfc")
	for _, name := range []string{"shadowedKey.findNoCase", "guessedKey.findNoCase"} {
		if reason, ok := got[name]; !ok || reason == "" {
			t.Errorf("%s: %q present=%v", name, reason, ok)
		}
	}
}
