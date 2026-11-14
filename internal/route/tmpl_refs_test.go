package route

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// The expression tmplRefAt and tmplRefs replaced, kept as the oracle they are
// held to.
var (
	oracleTmplRx      = regexp.MustCompile(`\$\{(\d+)(\+|-\d+)?(?::([a-z]+))?\}`)
	oracleTmplAnchord = regexp.MustCompile(`^\$\{(\d+)(\+|-\d+)?(?::([a-z]+))?\}`)
)

var tmplCases = []string{
	"${1}", "${1+}", "${1-3}", "${2:lower}", "${1+:search}", "${1-2:concat}", "${3:slash}", "${1:dotted}", "${12}", "${012}", "${0}",
	"a.${1}.b", "${1}${2}", "${1}.${2}", "${1}${2+}", "${1}-${2}", "$${1}", "${${1}}", "$ {1}", "${ 1}", "${1 }", "${1", "${", "$", "{1}", "${}",
	"${1+", "${1-}", "${1-a}", "${1+-2}", "${1+2}", "${1--2}", "${1:}", "${1:Upper}", "${1:up per}", "${1:upper", "${1:upper1}", "${1::upper}", "${1:a:b}",
	"${1-2-3}", "${1+:search} x", "x ${1+:search}", " ${1}", "${1} ", "${1}}", "{${1}", "${1}${", "${1}$", "${99999999999999999999}", "${1-99999999999999999999}",
	"${１}", "${1}é", "é${1}", "\xff${1}", "${1}\xff", "${1\xff}", "", " ", "\n${1}\n", "${1}\n${2}",
}

var tmplFragments = []string{
	"${", "}", "$", "{", "1", "23", "0", "+", "-", ":", "lower", "upper", "search", "concat", "slash", "dotted", "x", "Z", ".", "/", " ", "\n", "é", "\xff", "${1}", "${2+}", "${1-3}",
	"${1:lower}", "${1+:search}", "-4", ":a", "+}", "}}",
}

func randomTmpl(rng *rand.Rand) string {
	if rng.IntN(2) == 0 {
		s := tmplCases[rng.IntN(len(tmplCases))]

		for range rng.IntN(4) {
			i := rng.IntN(len(s) + 1)
			j := min(len(s), i+rng.IntN(3))

			switch rng.IntN(3) {
			case 0:
				s = s[:i] + tmplFragments[rng.IntN(len(tmplFragments))] + s[i:]
			case 1:
				s = s[:i] + s[j:]
			default:
				s = s[:i] + tmplFragments[rng.IntN(len(tmplFragments))] + s[j:]
			}
		}

		return s
	}

	var b strings.Builder

	for range 1 + rng.IntN(12) {
		b.WriteString(tmplFragments[rng.IntN(len(tmplFragments))])
	}

	return b.String()
}

func tmplDisagreement(s string) []string {
	var bad []string

	if got, want := fmt.Sprint(tmplRefs(s)), fmt.Sprint(oracleTmplRx.FindAllStringSubmatchIndex(s, -1)); got != want {
		bad = append(bad, fmt.Sprintf("tmplRefs(%q) = %s, want %s", s, got, want))
	}

	if got, want := fmt.Sprint(tmplRefAt(s)), fmt.Sprint(oracleTmplAnchord.FindStringSubmatchIndex(s)); got != want {
		bad = append(bad, fmt.Sprintf("tmplRefAt(%q) = %s, want %s", s, got, want))
	}

	return bad
}

// Both readers answer what the expression answered, every index included, on
// the templates a route rule is written with, their near misses, and texts
// built from fragments of them; one text in ten holds a reference.
func TestTemplateReadersMatchTheirExpression(t *testing.T) {
	n := 300_000
	if testing.Short() {
		n = 30_000
	}

	texts := append([]string(nil), tmplCases...)

	rng := rand.New(rand.NewPCG(1, 7))
	for range n {
		texts = append(texts, randomTmpl(rng))
	}

	failures, found, several := 0, 0, 0

	for _, s := range texts {
		for _, msg := range tmplDisagreement(s) {
			t.Error(msg)

			failures++

			if failures == 10 {
				return
			}
		}

		refs := len(oracleTmplRx.FindAllStringIndex(s, -1))
		if refs > 0 {
			found++
		}

		if refs > 1 {
			several++
		}
	}

	if found < len(texts)/10 || several < len(texts)/50 {
		t.Errorf("%d texts with a reference, %d with several, of %d; the generated texts do not reach the readers", found, several, len(texts))
	}
}

// FuzzTemplateReaders holds both readers to the expression on any text.
func FuzzTemplateReaders(f *testing.F) {
	for _, s := range tmplCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		for _, msg := range tmplDisagreement(s) {
			t.Error(msg)
		}
	})
}
