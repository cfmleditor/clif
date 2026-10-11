package parser

import "testing"

func TestDynamicScopeCollectionChainsKeepProducerOrigin(t *testing.T) {
	pr := ParseWithOptions(testURI, `component {function run(){request.keyArray().filter(function(key){return key.findNoCase("prefix");});}}`, &ParseOptions{ExtractCalls: true})
	for _, call := range pr.AllCalls() {
		if call.FuncName != "filter" {
			continue
		}

		if call.Component != "$any" || call.Variable != "request" || len(call.Chain) != 1 || CallHopName(call.Chain[0]) != "keyArray" {
			t.Fatalf("lost scope producer: %+v", call)
		}

		return
	}

	t.Fatal("filter call missing")
}
