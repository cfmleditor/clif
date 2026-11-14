package main

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// The expressions readers.go replaced, kept as the oracle each reader is held
// to.
var (
	oracleIntegrate      = regexp.MustCompile(`\$integrateComponents\(\s*["']([\w.]+)["']\s*\)`)
	oracleValidType      = regexp.MustCompile(`^[A-Za-z_][\w.]*(\[\])?$`)
	oracleIdent          = regexp.MustCompile(`^[A-Za-z_$][\w$]*$`)
	oracleInterfaceDoc   = regexp.MustCompile(`(?im)^@returns?\s+([A-Za-z_]\w*(?:\.\w+)*)\b`)
	oracleDocReturn      = regexp.MustCompile(`(?im)^@returns?\s+([A-Za-z_]\w*(?:\.\w+)+)\b`)
	oracleDelegate       = regexp.MustCompile(`(?is)^\s*\{\s*return\s+([A-Za-z_]\w*)\s*\(\s*\)\s*;?\s*\}\s*$`)
	oracleInterface      = regexp.MustCompile(`(?im)^\s*interface\b|<cfinterface\b`)
	oracleAccess         = regexp.MustCompile(`(?i)\b(public|private|package|remote)\b[^(]*\bfunction\b`)
	oracleTagAccess      = regexp.MustCompile(`(?i)\baccess\s*=\s*["']?(public|private|package|remote)`)
	oracleHint           = regexp.MustCompile(`(?is)\bhint\s*=\s*"([^"]*)"|\bhint\s*=\s*'([^']*)'`)
	oracleComponent      = regexp.MustCompile(`(?im)^\s*(abstract\s+|final\s+)?component\b|<cfcomponent\b`)
	oracleScriptProperty = regexp.MustCompile(`(?ims)^[ \t]*property\s+[^;{}]*?;`)
	oracleTagProperty    = regexp.MustCompile(`(?is)<cfproperty\b([^>]*?)/?>`)
)

var cfmlCases = []string{
	"component {\n}", "abstract component {", "final\tcomponent", "interface {", "  interface\n{", "\n\n  component hint=\"x\" {", "component", "components {", "xcomponent {",
	"<cfcomponent hint=\"x\">", "<cfcomponent>", "<cfcomponents>", "<CFCOMPONENT", "<cfinterface>", "<cfinterfaces>", "text <cfinterface", "  <cfcomponent",
	"property name=\"a\";", "\tproperty name=\"a\" type=\"b\";\nproperty x;", "property name=\"a\" default={};", "property\nname=\"a\"\n;", "property;", "property ;", "propertyx a;", "  property a b c ;",
	"property a; property b;", "x property a;", "<cfproperty name=\"a\">", "<cfproperty name=\"a\"/>", "<cfproperty name=\"a\" / >", "<cfproperty/>", "<cfproperty>", "<cfpropertyx a>", "<CFPROPERTY a=b>",
	"<cfproperty a=\"x>y\">", "<cfproperty a", "<cfproperty a/b>", "<cfproperty a//>",
	"@return string", "@returns pkg.Name", "@return a.b.c", "@return a.", "@return a..b", "@return .a", "@return  \n  x.y", " @return x", "x\n@return y.z\n", "@returnx a", "@returnsx a", "@Return A.B", "@return 1a", "@return _a.b_c",
	"{ return f(); }", "{return f();}", "{ return f( ) }", "{\n\treturn\tf\n(\n)\n;\n}\n", "{ return f(); } x", "x { return f(); }", "{ return f(1); }", "{ return f; }", "{ return 1f(); }", "{ return f();; }", "{ RETURN F(); }", "{return(f());}", "{ returnf(); }",
	"public function f() {", "private any function g(", "remote\tfunction h", "package function x", "public function", "public (function", "public static function f", "x public function", "publicfunction f", "public functions f", "public  function(  )",
	"public string function a(b) {", "PUBLIC FUNCTION F", "function public f", "public (a) function", "public a(b) function",
	"access=\"public\"", "access = 'private'", "ACCESS=remote", "access=\"packagex\"", "access=\"\"", "xaccess=\"public\"", "access public", "access=\n\"public\"", "access='remote' access='public'", "access=publi",
	"hint=\"a b\"", "hint='x'", "HINT = \"y\"", "hint=\"a'b\"", "hint='a\"b'", "hint=\"unclosed", "hint='unclosed", "xhint=\"a\"", "hint=a", "hint=\"\"", "hint=''", "hint=\"a\nb\"", "name=\"x\" hint=\"y\" access=\"public\"", "hints=\"a\"",
	"$integrateComponents(\"wheels.controller\")", "$integrateComponents( 'a.b' )", "$integrateComponents(\"\")", "$integrateComponents(\"a b\")", "$integrateComponents(\"a.b\"", "$integrateComponents(\"a.b\")$integrateComponents('c')", "$integratecomponents(\"a\")", "x$integrateComponents(\"a\")",
	"string", "pkg.Name", "pkg.Name[]", "a[]", "[]", "a.[]", "1a", "_a", "a b", "a-b", "$x", "x$", "a.b.c[]", "a[]b", "", " ", "é", "\xff",
}

var cfmlFragments = []string{
	"component", "interface", "abstract", "final", "property", "name=\"a\"", "hint=\"x\"", "hint='y'", "access=\"public\"", "access=", "public", "private", "package", "remote", "function", "(", ")", "{", "}", ";", ",", "=", "/", ">", "<", "\"", "'",
	"<cfproperty", "<cfcomponent", "<cfinterface", "<cffunction", "@return", "@returns", "return", "a.b", "pkg.Name", "x", "A", "1", "_", ".", "$", "[]", "$integrateComponents(", "\"wheels.x\"", " ", "  ", "\t", "\n", "\r", "\f", "\v", "é", "\xff",
	"{ return f(); }", "{ return", "f()", "f( )", "; }", "$integrateComponents(\"wheels.controller\")", "$integrateComponents( 'a.b' )", "\")", "\" )",
}

func randomCFML(rng *rand.Rand) string {
	if rng.IntN(2) == 0 {
		s := cfmlCases[rng.IntN(len(cfmlCases))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + cfmlFragments[rng.IntN(len(cfmlFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + cfmlFragments[rng.IntN(len(cfmlFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(14) {
		b.WriteString(cfmlFragments[rng.IntN(len(cfmlFragments))])
	}

	return b.String()
}

// randomDelegate is `{ return f(); }` with whitespace in each gap, and now and
// then a deviation: a missing piece, a case change, something after it.
func randomDelegate(rng *rand.Rand) string {
	gap := func() string { return []string{"", " ", "\n", "\t", "  ", " \n\t "}[rng.IntN(6)] }
	space := func() string { return []string{" ", "\n", "\t", "  "}[rng.IntN(4)] }
	pieces := []string{
		gap(), "{", gap(), []string{"return", "RETURN", "Return"}[rng.IntN(3)], space(), []string{"f", "doIt", "_x", "a1", "1a", "f.g"}[rng.IntN(6)], gap(), "(", gap(), ")",
		gap(), []string{"", ";", ";;"}[rng.IntN(3)], gap(), "}", gap(),
	}

	if rng.IntN(3) == 0 {
		i := rng.IntN(len(pieces))

		switch rng.IntN(3) {
		case 0:
			pieces[i] = ""
		case 1:
			pieces[i] = cfmlFragments[rng.IntN(len(cfmlFragments))]
		default:
			pieces = append(pieces, cfmlFragments[rng.IntN(len(cfmlFragments))])
		}
	}

	return strings.Join(pieces, "")
}

func groups(re *regexp.Regexp, s string) []string {
	var out []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}

	return out
}

func cfmlDisagreement(s string) []string {
	var bad []string

	check := func(name string, got, want any) {
		if g, w := fmt.Sprintf("%q", got), fmt.Sprintf("%q", want); g != w {
			bad = append(bad, fmt.Sprintf("%s(%q) = %s, want %s", name, s, g, w))
		}
	}

	check("integrateTargets", integrateTargets(s), groups(oracleIntegrate, s))
	check("validTypeName", validTypeName(s), oracleValidType.MatchString(s))
	check("identName", identName(s), oracleIdent.MatchString(s))

	for _, c := range []struct {
		name   string
		dotted bool
		re     *regexp.Regexp
	}{{"docReturnName(any)", false, oracleInterfaceDoc}, {"docReturnName(dotted)", true, oracleDocReturn}} {
		got, ok := docReturnName(s, c.dotted)
		m := c.re.FindStringSubmatch(s)

		want := ""
		if m != nil {
			want = m[1]
		}

		check(c.name, []any{got, ok}, []any{want, m != nil})
	}

	name, ok := delegateName(s)
	m := oracleDelegate.FindStringSubmatch(s)
	wantName := ""

	if m != nil {
		wantName = m[1]
	}

	check("delegateName", []any{name, ok}, []any{wantName, m != nil})
	check("declaresInterface", declaresInterface(s), oracleInterface.MatchString(s))

	word, ok := accessWord(s)
	m = oracleAccess.FindStringSubmatch(s)
	wantName = ""

	if m != nil {
		wantName = m[1]
	}

	check("accessWord", []any{word, ok}, []any{wantName, m != nil})

	word, ok = tagAccessWord(s)
	m = oracleTagAccess.FindStringSubmatch(s)
	wantName = ""

	if m != nil {
		wantName = m[1]
	}

	check("tagAccessWord", []any{word, ok}, []any{wantName, m != nil})

	double, single, found := hintAttr(s)
	m = oracleHint.FindStringSubmatch(s)

	var wantDouble, wantSingle string
	if m != nil {
		wantDouble, wantSingle = m[1], m[2]
	}

	check("hintAttr", []any{double, single, found}, []any{wantDouble, wantSingle, m != nil})

	start, found := componentStart(s)
	loc := oracleComponent.FindStringIndex(s)
	wantStart := -1

	if loc != nil {
		wantStart = loc[0]
	}

	if !found {
		start = -1
	}

	check("componentStart", start, wantStart)

	spans := scriptProperties(s)
	want := oracleScriptProperty.FindAllStringIndex(s, -1)

	var wantSpans [][2]int
	for _, l := range want {
		wantSpans = append(wantSpans, [2]int{l[0], l[1]})
	}

	check("scriptProperties", spans, wantSpans)
	check("tagProperties", tagProperties(s), groups(oracleTagProperty, s))

	return bad
}

// unfoldedCFMLRune is a character a regular expression's (?i) folds to ASCII
// and the readers, folding in ASCII alone, do not: the Kelvin sign and the long s.
func unfoldedCFMLRune(s string) bool { return strings.ContainsAny(s, "Kſ") }

// Every reader answers what its expression answered, positions and groups
// included, on declarations and doc comments of the forms the generator reads,
// near misses of each, and texts built from their fragments.
func TestReadersMatchTheirExpressions(t *testing.T) {
	n := 200_000
	if testing.Short() {
		n = 20_000
	}

	texts := append([]string(nil), cfmlCases...)

	rng := rand.New(rand.NewPCG(1, 7))
	for range n {
		texts = append(texts, randomCFML(rng))
	}

	for range n / 4 {
		texts = append(texts, randomDelegate(rng))
	}

	failures, hits := 0, map[string]int{}

	for _, s := range texts {
		if unfoldedCFMLRune(s) {
			continue
		}

		for _, msg := range cfmlDisagreement(s) {
			t.Error(msg)

			failures++

			if failures == 10 {
				return
			}
		}

		for name, re := range map[string]*regexp.Regexp{
			"integrate": oracleIntegrate, "doc": oracleInterfaceDoc, "delegate": oracleDelegate, "interface": oracleInterface, "access": oracleAccess,
			"tagaccess": oracleTagAccess, "hint": oracleHint, "component": oracleComponent, "scriptprop": oracleScriptProperty, "tagprop": oracleTagProperty,
		} {
			if re.MatchString(s) {
				hits[name]++
			}
		}
	}

	for name, n := range hits {
		if n < len(texts)/100 {
			t.Errorf("%s: %d of %d texts match; the generated texts do not reach the reader", name, n, len(texts))
		}
	}
}

// FuzzReaders holds every reader to its expression on any text.
func FuzzReaders(f *testing.F) {
	for _, s := range cfmlCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		if unfoldedCFMLRune(s) {
			t.Skip()
		}

		for _, msg := range cfmlDisagreement(s) {
			t.Error(msg)
		}
	})
}
