package resolve

import "testing"

func TestDirectClosureLiteralReturn(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Builder.cfc": `component {
 function literal(){return {run:function(value){return value;},'quoted':function(){},count:1};}
 function wrapper(){return literal();}
 function duplicate(){return {run:function(){},RUN:1};}
 function computed(){return {'#key#':function(){}};}
 function conflicting(){if(flag){return unknown;}return {run:function(){}};}
 function run(){
 literal().run("value").anything();
 literal().quoted();
 literal().missing();
 wrapper().run("value");
 duplicate().run();
 computed().run();
 conflicting().run();
 }
}`,
	})
	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Builder.cfc"), map[string]string{
		"literal.run.anything": "", "literal.quoted": "", "wrapper.run": "",
		"literal.missing": "method 'missing' is not a member of the struct 'literal' in Builder returns",
		"duplicate.run":   "method 'duplicate' has no component return type (chain to 'run')",
		"computed.run":    "method 'computed' has no component return type (chain to 'run')",
		"conflicting.run": "method 'conflicting' has no component return type (chain to 'run')",
	})
}

func TestForwardedReturnMutationBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"scope escape", `mutate(this);return renderer();`},
		{"scope alias", `var alias=variables;mutate(alias);return renderer();`},
		{"scope replacement", `variables=unknown;return renderer();`},
		{"uninitialized shadow", `var renderer;return renderer();`},
		{"condition assignment", `if((renderer=unknown)){}return renderer();`},
		{"closure mutation", `var closure=function(){variables.renderer=unknown;};closure();return renderer();`},
		{"evaluated mutation", `evaluate(code);return renderer();`},
		{"computed replacement", `variables[key]=unknown;return renderer();`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"Builder.cfc": `component {
 function renderer(){var builder={run:function(){}};return builder;}
 function forwarded(){` + tc.body + `}
 function run(){forwarded().run();}
}`})
			expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Builder.cfc"), map[string]string{
				"forwarded.run": "method 'forwarded' has no component return type (chain to 'run')",
			})
		})
	}
}

func TestAWrapperReturnsAnEngineValue(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Response.cfc": `component {
 function response(){return getPageContext().getResponse();}
 function forwarded(){return response();}
 function twice(){return variables.forwarded();}
 function conflicting(){if(flag){return unknown;}return response();}
 function supplied(response){return response();}
 function shadowed(){var response=unknown;return response();}
 function cycle(){return cycle();}
 function run(){
 forwarded().isCommitted();
 twice().flush();
 conflicting().flush();
 supplied().flush();
 shadowed().flush();
 cycle().flush();
 }
}`,
		"Custom.cfc": `component extends="Response" {
 function response(){return unknown;}
 function run(){forwarded().flush();}
}`,
	})

	r := &Resolver{}
	expectReasons(t, reasonsWith(t, r, dir, "Response.cfc"), map[string]string{
		"forwarded.isCommitted": "", "twice.flush": "",
		"conflicting.flush": "method 'conflicting' has no component return type (chain to 'flush')",
		"supplied.flush":    "method 'supplied' has no component return type (chain to 'flush')",
		"shadowed.flush":    "method 'shadowed' has no component return type (chain to 'flush')",
		"cycle.flush":       "method 'cycle' has no component return type (chain to 'flush')",
	})
	expectReasons(t, reasonsWith(t, r, dir, "Custom.cfc"), map[string]string{
		"forwarded.flush": "method 'forwarded' has no component return type (chain to 'flush')",
	})
}

func TestAWrapperReturnsTheClosureBuilder(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Builder.cfc": `component {
 function renderer(){var builder={};structAppend(builder,{data:function(value){return builder;},type:function(value){return builder;}});return builder;}
 function renderData(data){if(data != '') request.warning=true;request.renderData={data=arguments.data};return renderer();}
 function forwarded(){return this.renderData();}
 function cycle(){return cycle();}
 function conflicting(){if(flag){return unknown;}return renderer();}
 function shadowed(){var renderer=unknown;return renderer();}
 function supplied(renderer){return renderer();}
 function withArgs(){return renderData(unknown);}
 function run(){
 renderData().data("x").type("json");
 renderData().missing();
 forwarded().type("json");
 cycle().type("json");
 conflicting().type("json");
 shadowed().type("json");
 supplied().type("json");
 withArgs().type("json");
 }
}`,
		"Custom.cfc": `component extends="Builder" {function renderer(){var builder={};structAppend(builder,{custom:function(){return builder;}});return builder;}function run(){renderData().custom();renderData().data();}}`,
	})

	r := &Resolver{}
	expectReasons(t, reasonsWith(t, r, dir, "Builder.cfc"), map[string]string{
		"renderData.data.type": "", "renderData.missing": "method 'missing' is not a member of the struct 'renderData' in Builder returns",
		"forwarded.type":   "method 'forwarded' has no component return type (chain to 'type')",
		"cycle.type":       "method 'cycle' has no component return type (chain to 'type')",
		"conflicting.type": "method 'conflicting' has no component return type (chain to 'type')",
		"shadowed.type":    "method 'shadowed' has no component return type (chain to 'type')",
		"supplied.type":    "method 'supplied' has no component return type (chain to 'type')",
		"withArgs.type":    "method 'withArgs' has no component return type (chain to 'type')",
	})
	expectReasons(t, reasonsWith(t, r, dir, "Custom.cfc"), map[string]string{
		"renderData.custom": "", "renderData.data": "method 'data' is not a member of the struct 'renderData' in Custom returns",
	})
}

// TestAStructOfClosuresAnswersItsMembers: DI/1's declare() returns a struct
// whose members are closures that return it again, so
// declare( "x" ).instanceOf( "y" ).asSingleton() is a chain of members. A name
// the struct does not hold is still reported, and a member returning anything
// else ends what can be checked.
func TestAStructOfClosuresAnswersItsMembers(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"ioc.cfc": `component {
	public any function declare( string beanName ) {
		var declaration = { beanName : beanName, built : false };
		var beanFactory = this;
		structAppend( declaration, {
			instanceOf : function( string dottedPath ) {
				if ( declaration.built ) throw "done";
				declaration.built = true;
				return declaration;
			},
			asSingleton : function() {
				return declaration;
			},
			done : function() {
				return beanFactory;
			}
		} );
		return declaration;
	}

	public any function plain() {
		var s = { a : 1 };
		return s;
	}
}`,
		"Main.cfc": `component {
	function setup() {
		var bf = new ioc();
		bf.declare( "a" ).instanceOf( "x.y" ).asSingleton();
		bf.declare( "b" ).instanceOf( "x.y" ).done().whatever();
		bf.declare( "c" ).notAMember();
		bf.plain().nothing();
	}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Main.cfc"), map[string]string{
		"bf.declare.instanceOf":               "",
		"bf.declare.instanceOf.asSingleton":   "",
		"bf.declare.instanceOf.done.whatever": "",
		"bf.declare.notAMember":               "method 'notAMember' is not a member of the struct 'declare' in ioc returns",
		"bf.plain.nothing":                    "method 'plain' in ioc has no component return type (chain to 'nothing')",
	})
}

// TestAReturnOfABuiltInsValueIsDynamic: TestBox's getPageContextResponse()
// returns getPageContext().getResponse(), or a struct standing in for it, so
// a call on what it returns is a call on what the engine hands back. A
// function with any other kind of return is still reported.
func TestAReturnOfABuiltInsValueIsDynamic(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"Reporter.cfc": `component {
	function getPageContextResponse() {
		if ( !getFunctionList().keyExists( "getPageContext" ) ) {
			return {
				"setContentType" : function() {}
			};
		}
		return getPageContext().getResponse();
	}

	function getHeld() {
		if ( x ) {
			return getPageContext();
		}
		return variables.held;
	}

	private function getResponse() {
		return server.keyExists( "lucee" ) ? getPageContext().getResponse() : getPageContext()
			.getResponse()
			.getResponse();
	}

	private function getEither() {
		return x ? getPageContext() : variables.held;
	}

	function run() {
		getPageContextResponse().setContentType( "text/html" );
		getHeld().anything();
		getResponse().isCommitted();
		getEither().anything();
	}
}`,
	})

	expectReasons(t, reasonsWith(t, &Resolver{}, dir, "Reporter.cfc"), map[string]string{
		"getPageContextResponse.setContentType": "",
		"getHeld.anything":                      "method 'getHeld' has no component return type (chain to 'anything')",
		"getResponse.isCommitted":               "",
		"getEither.anything":                    "method 'getEither' has no component return type (chain to 'anything')",
	})
}
