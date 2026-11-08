package resolve

import (
	"maps"
	"testing"
)

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

// Each contract the registration rule checks is one more way for it to decline:
// the calling method, the receiver's spelling, the registrar's shape and file,
// the assertion component, the selected hook's shape, the chain up to the
// registrar. A call that fails any is reported as it was before the rule.
func TestLiteralAssertionsDeclineWhenAnyContractFails(t *testing.T) {
	base := map[string]string{
		"testbox/system/Assertion.cfc": `component {function ordinary(){}}`,
		"testbox/system/BaseSpec.cfc":  `component {variables.$assert=this.$assert=new testbox.system.Assertion();function addAssertions(required any assertions){if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,true);return this;}}}`,
		"Spec.cfc":                     `component extends="testbox.system.BaseSpec" {function beforeTests(){addAssertions({awesome:function(){return true;}});} function testCustom(){$assert.awesome();}}`,
	}

	for _, tc := range []struct {
		name     string
		override map[string]string
		resolved bool
	}{
		{name: "the baseline is accepted", resolved: true},
		{name: "this-scoped receiver", resolved: true, override: map[string]string{"Spec.cfc": `component extends="testbox.system.BaseSpec" {function beforeTests(){addAssertions({awesome:function(){}});} function testCustom(){this.$assert.awesome();}}`}},
		{name: "not a test method", override: map[string]string{"Spec.cfc": `component extends="testbox.system.BaseSpec" {function beforeTests(){addAssertions({awesome:function(){}});} function helper(){$assert.awesome();}}`}},
		{name: "another receiver", override: map[string]string{"Spec.cfc": `component extends="testbox.system.BaseSpec" {function beforeTests(){addAssertions({awesome:function(){}});} function testCustom(){var a=$assert; a.awesome();}}`}},
		{name: "registrar argument renamed", override: map[string]string{"testbox/system/BaseSpec.cfc": `component {variables.$assert=this.$assert=new testbox.system.Assertion();function addAssertions(required any items){if(isStruct(arguments.items)){structAppend(this.$assert,arguments.items,true);return this;}}}`}},
		{name: "registrar not public", override: map[string]string{"testbox/system/BaseSpec.cfc": `component {variables.$assert=this.$assert=new testbox.system.Assertion();private function addAssertions(required any assertions){if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,true);return this;}}}`}},
		{name: "registrar in a base of another name", override: map[string]string{"testbox/system/Other.cfc": `component {variables.$assert=this.$assert=new testbox.system.Assertion();function addAssertions(required any assertions){if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,true);return this;}}}`, "Spec.cfc": `component extends="testbox.system.Other" {function beforeTests(){addAssertions({awesome:function(){}});} function testCustom(){$assert.awesome();}}`}},
		{name: "registrar outside BaseSpec", override: map[string]string{"testbox/system/BaseSpec.cfc": `component {variables.$assert=this.$assert=new testbox.system.Assertion();}`, "Spec.cfc": `component extends="testbox.system.BaseSpec" {function addAssertions(required any assertions){if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,true);return this;}} function beforeTests(){addAssertions({awesome:function(){}});} function testCustom(){$assert.awesome();}}`}},
		{name: "assertion built elsewhere", override: map[string]string{"testbox/system/BaseSpec.cfc": `component {function addAssertions(required any assertions){if(isStruct(arguments.assertions)){structAppend(this.$assert,arguments.assertions,true);return this;}}}`}},
		{name: "no setup hook", override: map[string]string{"Spec.cfc": `component extends="testbox.system.BaseSpec" {function testCustom(){$assert.awesome();}}`}},
		{name: "hook takes an argument", override: map[string]string{"Spec.cfc": `component extends="testbox.system.BaseSpec" {function beforeTests(x){addAssertions({awesome:function(){}});} function testCustom(){$assert.awesome();}}`}},
		{name: "hook not public", override: map[string]string{"Spec.cfc": `component extends="testbox.system.BaseSpec" {private function beforeTests(){addAssertions({awesome:function(){}});} function testCustom(){$assert.awesome();}}`}},
		{name: "empty hook", override: map[string]string{"Spec.cfc": `component extends="testbox.system.BaseSpec" {function beforeTests(){} function testCustom(){$assert.awesome();}}`}},
		{name: "another name registered", override: map[string]string{"Spec.cfc": `component extends="testbox.system.BaseSpec" {function beforeTests(){addAssertions({other:function(){}});} function testCustom(){$assert.awesome();}}`}},
		{name: "a parent with a harmless hook", resolved: true, override: map[string]string{"Parent.cfc": `component extends="testbox.system.BaseSpec" {function beforeTests(){}}`, "Spec.cfc": `component extends="Parent" {function beforeTests(){super.beforeTests(); addAssertions({awesome:function(){}});} function testCustom(){$assert.awesome();}}`}},
		{name: "a parent hook that does something", override: map[string]string{"Parent.cfc": `component extends="testbox.system.BaseSpec" {function beforeTests(){doSomething();}}`, "Spec.cfc": `component extends="Parent" {function beforeTests(){super.beforeTests(); addAssertions({awesome:function(){}});} function testCustom(){$assert.awesome();}}`}},
		{name: "a parent that cannot be found", override: map[string]string{"Spec.cfc": `component extends="Missing.Parent" {function beforeTests(){addAssertions({awesome:function(){}});} function testCustom(){$assert.awesome();}}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{}

			maps.Copy(files, base)
			maps.Copy(files, tc.override)

			writeFiles(t, dir, files)

			got := reasonsIn(t, dir, "Spec.cfc")

			var (
				reason string
				ok     bool
			)

			for _, key := range []string{"$assert.awesome", "this.$assert.awesome", "a.awesome"} {
				if reason, ok = got[key]; ok {
					break
				}
			}

			if !ok || (reason == "") != tc.resolved {
				t.Fatalf("got %q exists %v want resolved %v (all: %v)", reason, ok, tc.resolved, got)
			}
		})
	}
}
