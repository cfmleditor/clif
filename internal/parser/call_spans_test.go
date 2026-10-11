package parser

import (
	"strings"
	"testing"
)

func TestCallSpansPreserveSourceOrigins(t *testing.T) {
	for _, src := range []string{
		`component {function run(){each(xs,function(c){c.go();each(xs,function(c){c.go();});});each(xs,function(c){c.go();});}}`,
		`<cfcomponent><cffunction name="run"><cfscript>each(xs,function(c){c.go();});each(xs,function(c){c.go();});</cfscript></cffunction></cfcomponent>`,
		`component {function run(){writeOutput("value #c.go()#");}}`,
		`<cfcomponent><cffunction name="run"><cfif c.go()><cfreturn c.go()></cfif></cffunction></cfcomponent>`,
		`<cfoutput>#c.go()# #c.go()#</cfoutput>`,
		`<cfcomponent><cffunction name="run"><cfset x=c.go()><cfset local.y=c.go()></cffunction></cfcomponent>`,
		"<cfcomponent><cffunction name=\"run\"><cfset\n\tx =\n c.go()><cfset\tlocal.y =\n c.go()></cffunction></cfcomponent>",
		"<cfcomponent><cffunction name=\"run\"><cfset other.send(\n\"#c.go()#\")></cffunction></cfcomponent>",
		`<cfcomponent><cffunction name="run"><cfif c.go()><cfset other.send(c.go())></cfif></cffunction></cfcomponent>`,
		"<cfcomponent><cffunction name=\"run\"><cfset other.send(\n c.go(),\n\"#c.go()#\")></cffunction></cfcomponent>",
		`<cfcomponent><cffunction name="run"><cfset other.send(c.go(),"#c.go()#")></cffunction></cfcomponent>`,
		`<cfcomponent><cffunction name="run"><cfset local.result=other.send(c.go(),"#c.go()#")></cffunction></cfcomponent>`,
		`<cfcomponent><cffunction name="run"><cfset x="#c.next().go()#"><cfset other.send("#c.next().go()#")></cffunction></cfcomponent>`,
		`<cfcomponent><cffunction name="run"><cfset x=replace(x,'src="/','src="#c.next('scheme')#://#c.next().go()#/','ALL')></cffunction></cfcomponent>`,
		"<cfcomponent><cffunction name=\"run\"><cfset\tlocal.instance= new Example()><cfset instance.go()></cffunction></cfcomponent>",
	} {
		for _, prefix := range []string{"", "\n/* shifted */\n"} {
			text := prefix + src
			pr := ParseWithOptions("file:///tmp/Origins.cfc", text, &ParseOptions{ExtractCalls: true})
			seen := map[uint32]bool{}
			count := 0

			if strings.Contains(src, "local.instance=") {
				for _, ref := range pr.ComponentRefs {
					if ref.Variable == "instance" {
						t.Fatal("a local assignment became a component-level reference")
					}
				}
			}

			for _, c := range pr.AllCalls() {
				if c.FuncName != "go" {
					continue
				}

				count++

				if strings.Contains(src, "local.instance=") && c.Component != "Example" {
					t.Fatalf("leading whitespace lost the local assignment: %+v", c)
				}

				if !c.HasSpan || int(c.EndByte) > len(text) || text[c.StartByte:c.EndByte] != "go()" {
					t.Fatalf("bad origin %+v in %s", c, text)
				}

				if seen[c.StartByte] {
					t.Fatalf("shared origin: %+v", c)
				}

				seen[c.StartByte] = true
				if int(c.Line) != strings.Count(text[:c.StartByte], "\n") {
					t.Fatalf("wrong source line: %+v in %s", c, text)
				}
			}

			if count != strings.Count(src, "go()") {
				t.Fatalf("found %d calls in %s", count, src)
			}
		}
	}
}

func TestTagMemberStoresRetainClosureByteOrigins(t *testing.T) {
	for _, prefix := range []string{"", "\n<!--- shifted --->\n"} {
		source := prefix + `<cfcomponent><cffunction name="run"><cfset each(items,function(){local.box.itemA=buildA();})><cfset each(items,function(){local.box.itemB=buildB();})></cffunction></cfcomponent>`
		p := newTagParser(source, "file:///tmp/Closures.cfc")
		p.extractCalls = true
		p.parse()

		count := 0

		for _, call := range p.pendingCalls {
			if !call.memberSet {
				continue
			}

			count++

			if !call.hasClosureSpan || int(call.closureTo) > len(source) || !strings.Contains(source[call.closureFrom:call.closureTo], "local.box."+call.funcName+"=") {
				t.Fatalf("member store lost its source closure: %+v", call)
			}
		}

		if count != 2 {
			t.Fatalf("recorded %d member stores", count)
		}
	}
}

func TestChainedCallsKeepEachHopByteIdentity(t *testing.T) {
	for _, body := range []string{
		"component {function run(x){x.getStats()\n.setLockID(1)\n.save();x.getStats()\n.save(value=unknownValue);}}",
		`component {function run(x){x.getStats().setLockID(1).save();x.getStats().save(value=unknownValue);}}`,
		`<cfcomponent><cffunction name="run"><cfset x.getStats().setLockID(1).save()><cfset x.getStats().save(value=unknownValue)></cffunction></cfcomponent>`,
		`<cfcomponent><cffunction name="run"><cfset local.result=x.getStats().setLockID(1).save()><cfset local.other=x.getStats().save(value=unknownValue)></cffunction></cfcomponent>`,
	} {
		source := "\n<!--- source offset --->\n" + body
		pr := ParseWithOptions("file:///tmp/Chain.cfc", source, &ParseOptions{ExtractCalls: true})
		seen := map[uint32]bool{}
		count := 0

		for _, call := range pr.AllCalls() {
			if call.FuncName != "save" {
				continue
			}

			count++

			if !call.HasSpan || int(call.EndByte) > len(source) {
				t.Fatalf("chain call lost identity: %+v", call)
			}

			text := source[call.StartByte:call.EndByte]
			if text != "save()" && text != "save(value=unknownValue)" {
				t.Fatalf("wrong hop span: %q %+v", text, call)
			}

			if seen[call.StartByte] {
				t.Fatalf("two calls share a byte origin: %+v", call)
			}

			seen[call.StartByte] = true
			if int(call.Line) != strings.Count(source[:call.StartByte], "\n") {
				t.Fatalf("wrong hop line: %+v", call)
			}
		}

		if count != 2 {
			t.Fatalf("got %d save calls in %s", count, source)
		}
	}
}

func TestChainedInterpolationCallsAreNotDuplicated(t *testing.T) {
	source := `<cfcomponent><cffunction name="run"><cfset variables.mailer.sendText("Your #variables.settingsManager.getSite(id).getSite()# website")></cffunction></cfcomponent>`
	pr := ParseWithOptions("file:///tmp/Message.cfc", source, &ParseOptions{ExtractCalls: true})
	count := 0
	seen := map[uint32]bool{}

	for _, call := range pr.AllCalls() {
		if call.FuncName != "getSite" {
			continue
		}

		count++

		if !call.HasSpan || seen[call.StartByte] {
			t.Fatalf("duplicate or unidentified interpolation call: %+v", call)
		}

		seen[call.StartByte] = true
	}

	if count != 2 {
		t.Fatalf("one root and one chained call produced %d calls", count)
	}
}

func TestTagFunctionScriptIslandNamedCallsKeepIdentity(t *testing.T) {
	for _, prefix := range []string{"", "\n<!--- shifted --->\n"} {
		source := prefix + `<cfcomponent><cffunction name="run"><cfscript>
   x.save();
   x.save(
     value=y.save()
   );
   for(var x in items){x.save();}
  </cfscript></cffunction></cfcomponent>`
		pr := ParseWithOptions("file:///tmp/Island.cfc", source, &ParseOptions{ExtractCalls: true})
		count := 0
		seen := map[uint32]bool{}

		for _, call := range pr.AllCalls() {
			if call.FuncName != "save" {
				continue
			}

			count++

			if !call.HasSpan || int(call.EndByte) > len(source) || seen[call.StartByte] {
				t.Fatalf("lost named-call identity: %+v", call)
			}

			if !strings.HasPrefix(source[call.StartByte:call.EndByte], "save(") || call.Caller != "run" || int(call.Line) != strings.Count(source[:call.StartByte], "\n") {
				t.Fatalf("wrong ownership/origin: %+v", call)
			}

			seen[call.StartByte] = true
		}

		if count != 4 {
			t.Fatalf("got %d save calls", count)
		}
	}
}
