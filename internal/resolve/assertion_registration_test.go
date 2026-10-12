package resolve

import "testing"

func TestLiteralAssertionsFollowSelectedSetup(t *testing.T) {
	for _, tc := range []struct {
		name, setup, extra, call, registrar string
		resolved                            bool
	}{
		{name: "literal closure", setup: `addAssertions({awesome:function(){return true;}});`, resolved: true},
		{name: "quoted name", setup: `addAssertions({"awesome":function(){return true;}});`, resolved: true},
		{name: "extra registrar", setup: `addAssertions({awesome:function(){}});`, extra: `function replace(){addAssertions({awesome:1});}`},
		{name: "scope key replacement", setup: `addAssertions({awesome:function(){}});`, extra: `function replace(){this["$assert"]={};}`},
		{name: "interpolated replacement", setup: `addAssertions({awesome:function(){}});`, call: `var text="#mutate(this)#";`},
		{name: "builtin override", setup: `addAssertions({awesome:function(){}});`, extra: `function isStruct(value){return false;}`},
		{name: "whole scope replacement", setup: `addAssertions({awesome:function(){}});`, extra: `function replace(){variables={};}`},
		{name: "conditional", setup: `if(runtime()){addAssertions({awesome:function(){}});}`},
		{name: "primitive member", setup: `addAssertions({awesome:1});`},
		{name: "overwrite", setup: `addAssertions({awesome:function(){}});addAssertions({awesome:1});`},
		{name: "registrar override", setup: `addAssertions({awesome:function(){}});`, extra: `function addAssertions(assertions){return this;}`},
		{name: "assertion replacement", setup: `addAssertions({awesome:function(){}});`, call: `$assert=new testbox.system.Assertion();`},
		{name: "whole instance escape", setup: `addAssertions({awesome:function(){}});`, call: `mutate(this);`},
		{name: "other writer", setup: `addAssertions({awesome:function(){}});`, extra: `function replace(){variables.$assert=new testbox.system.Assertion();}`},
		{name: "later hook", setup: `addAssertions({awesome:function(){}});`, extra: `function setup(){variables.$assert={};}`},
		{name: "false overwrite flag", setup: `addAssertions({awesome:function(){}});`, registrar: `if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,false);return this;}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			registrar := tc.registrar
			if registrar == "" {
				registrar = `if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,true);return this;}`
			}

			writeFiles(t, dir, map[string]string{
				"testbox/system/Assertion.cfc": `component {function ordinary(){}}`,
				"testbox/system/BaseSpec.cfc":  `component {variables.$assert=this.$assert=new testbox.system.Assertion();function addAssertions(required any assertions){` + registrar + `}}`,
				"Spec.cfc":                     `component extends="testbox.system.BaseSpec" {function beforeTests(){` + tc.setup + `} ` + tc.extra + ` function testCustom(){` + tc.call + ` $assert.awesome();}}`,
			})
			got := reasonsIn(t, dir, "Spec.cfc")

			reason, ok := got["$assert.awesome"]
			if !ok || (reason == "") != tc.resolved {
				t.Fatalf("got %q exists %v want resolved %v", reason, ok, tc.resolved)
			}
		})
	}
}

func TestLiteralAssertionsRespectHookOverride(t *testing.T) {
	for _, override := range []bool{false, true} {
		dir := t.TempDir()

		child := ""
		if override {
			child = `function beforeTests(){}`
		}

		writeFiles(t, dir, map[string]string{
			"testbox/system/Assertion.cfc": `component {}`,
			"testbox/system/BaseSpec.cfc":  `component {variables.$assert=this.$assert=new testbox.system.Assertion();function addAssertions(required any assertions){if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,true);return this;}}}`,
			"Parent.cfc":                   `component extends="testbox.system.BaseSpec" {function beforeTests(){addAssertions({awesome:function(){}});}}`,
			"Spec.cfc":                     `component extends="Parent" {` + child + `function testCustom(){$assert.awesome();}}`,
		})

		got := reasonsIn(t, dir, "Spec.cfc")
		if (got["$assert.awesome"] == "") == override {
			t.Fatalf("override %v got %v", override, got)
		}
	}
}
