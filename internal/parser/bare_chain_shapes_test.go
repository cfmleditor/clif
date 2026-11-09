package parser

import (
	"fmt"
	"strings"
	"testing"
)

// TestABareCallRecordsEveryHopOfItsChain pins what recordBareCallAndChain
// records for a call written bare and chained on: each call's name, its Chain
// (the calls before it, with their arguments), its Caller and its line. The
// shapes are the ones the walk has a case for: a plain hop, an index between
// hops, a call written inside a hop's arguments, a resolver naming the first
// call's component, and a chain that stops at a non-call.
func TestABareCallRecordsEveryHopOfItsChain(t *testing.T) {
	src := `component {
	function run() {
		a().b().c();
		d()[ key ].e();
		f( g() ).h( i() ).j();
		k().field;
		l();
	}
}`

	pr := ParseWithOptions(testURI, src, &ParseOptions{ExtractCalls: true})

	var got []string

	calls := pr.AllCalls()
	for i := range calls {
		c := &calls[i]
		got = append(got, fmt.Sprintf("%d:%s chain=%v caller=%s comp=%q", c.Line, c.FuncName, c.Chain, c.Caller, c.Component))
	}

	// Lines are the parser's own, counted from zero; a chain's earlier calls
	// are written with their arguments ($call:f( g() )).
	want := []string{
		"2:a chain=[] caller=run comp=\"\"",
		"2:b chain=[$call:a()] caller=run comp=\"\"",
		"2:c chain=[$call:a() $call:b()] caller=run comp=\"\"",
		"3:d chain=[] caller=run comp=\"\"",
		"3:e chain=[$call:d()] caller=run comp=\"\"",
		"4:f chain=[] caller=run comp=\"\"",
		"4:g chain=[] caller=run comp=\"\"",
		"4:h chain=[$call:f( g() )] caller=run comp=\"\"",
		"4:i chain=[] caller=run comp=\"\"",
		"4:j chain=[$call:f( g() ) $call:h( i() )] caller=run comp=\"\"",
		"5:k chain=[] caller=run comp=\"\"",
		"6:l chain=[] caller=run comp=\"\"",
	}

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// A resolver that names the first call's component makes it the receiver of
// the first hop, so the call is not repeated at the front of the chain.
func TestAResolvedBareCallIsTheFirstHopsReceiver(t *testing.T) {
	src := `component {
	function run() {
		getSvc().first().second();
	}
}`

	resolvers := []Resolver{{Match: "getSvc()", Resolve: "app.Svc", Prefix: "getSvc"}}
	pr := ParseWithOptions(testURI, src, &ParseOptions{ExtractCalls: true, Resolvers: resolvers})

	var got []string

	calls := pr.AllCalls()
	for i := range calls {
		c := &calls[i]
		got = append(got, fmt.Sprintf("%s chain=%v comp=%q resolved=%v", c.FuncName, c.Chain, c.Component, c.Resolved))
	}

	want := `first chain=[] comp="app.Svc" resolved=true
getSvc chain=[] comp="" resolved=false
second chain=[$call:first()] comp="app.Svc" resolved=true`

	if strings.Join(got, "\n") != want {
		t.Errorf("calls:\n%s\nwant:\n%s", strings.Join(got, "\n"), want)
	}
}
