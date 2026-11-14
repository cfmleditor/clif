package resolve

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// The expressions wheels_dispatch.go replaced, built as they were, kept as the
// oracle each reader is held to.
func oracleDispatchStart(body, name string) (int, bool) {
	condition := `if\t\(\targuments\t\.\t` + regexp.QuoteMeta(name) + `\t(?:eq|=\t=)\t"[^"#]*"\t\)\t\{\treturn\t[^;{}]+\t;\t\}`
	pattern := regexp.MustCompile(condition + `(?:\telse\t` + condition + `)*\telse\t\{\treturn\tthis\t;\t\}\t$`)

	loc := pattern.FindStringIndex(body)
	if loc == nil {
		return 0, false
	}

	return loc[0], true
}

func oracleDispatchModes(text, name string) []string {
	selector := regexp.MustCompile(`arguments\t\.\t` + regexp.QuoteMeta(name) + `\t(?:eq|=\t=)\t"([^"#]*)"`)

	var out []string
	for _, m := range selector.FindAllStringSubmatch(text, -1) {
		out = append(out, m[1])
	}

	return out
}

var dispatchNames = []string{"mode", "m", "a.b", "x+y", "$z", "mode2"}

// dispatchPieces are the grammar's own tokens and the near misses of each; a
// body is some of them joined with tabs, each followed by one, as the method's
// body text is.
var dispatchPieces = []string{
	"if", "(", ")", "{", "}", ";", "else", "return", "this", "arguments", ".", "eq", "=", "==", "\"a\"", "\"self\"", "\"\"", "\"a#b\"", "\"a\"b\"", "\"x y\"",
	"x", "y", "result", "mode", "m", "a.b", "x+y", "$z", "if(", "elif", "xif", "else{", "return;", ";;", "#", "neq", "eqq", "EQ", "é", "\"",
	"=\t=", "\t", "if\t(\targuments\t.\tmode\teq\t\"a\"\t)\t{\treturn\tx\t;\t}", "else\t{\treturn\tthis\t;\t}",
}

// dispatchChain is a chain the grammar reads, with a few edits.
func dispatchChain(rng *rand.Rand, name string) string {
	var b strings.Builder

	op := []string{"eq", "=\t="}[rng.IntN(2)]
	cond := func() string {
		lit := []string{"\"a\"", "\"b\"", "\"self\"", "\"\"", "\"x y\""}[rng.IntN(5)]
		ret := []string{"x", "this", "a\tb", "x\t.\ty\t(\t)", "\"s\"", "arguments\t.\tz"}[rng.IntN(6)]

		return "if\t(\targuments\t.\t" + name + "\t" + op + "\t" + lit + "\t)\t{\treturn\t" + ret + "\t;\t}\t"
	}

	b.WriteString(cond())

	for range rng.IntN(4) {
		b.WriteString("else\t" + cond())
	}

	b.WriteString("else\t{\treturn\tthis\t;\t}\t")

	return b.String()
}

func randomDispatchBody(rng *rand.Rand) (body, name string) {
	name = dispatchNames[rng.IntN(len(dispatchNames))]

	var b strings.Builder

	for range rng.IntN(5) {
		b.WriteString(dispatchPieces[rng.IntN(len(dispatchPieces))] + "\t")
	}

	if rng.IntN(4) > 0 {
		b.WriteString(dispatchChain(rng, name))
	}

	s := b.String()

	// A few edits: a piece inserted, bytes cut, or bytes replaced.
	for range rng.IntN(4) {
		i := rng.IntN(len(s) + 1)
		j := min(len(s), i+rng.IntN(4))

		switch rng.IntN(3) {
		case 0:
			s = s[:i] + dispatchPieces[rng.IntN(len(dispatchPieces))] + "\t" + s[i:]
		case 1:
			s = s[:i] + s[j:]
		default:
			s = s[:i] + dispatchPieces[rng.IntN(len(dispatchPieces))] + s[j:]
		}
	}

	if rng.IntN(3) == 0 {
		s += dispatchPieces[rng.IntN(len(dispatchPieces))] + "\t"
	}

	return s, name
}

func dispatchDisagreement(body, name string) []string {
	var bad []string

	got, gok := selfDispatchStart(body, name)
	want, wok := oracleDispatchStart(body, name)

	if gok != wok || got != want {
		bad = append(bad, fmt.Sprintf("selfDispatchStart(%q, %q) = %d %v, want %d %v", body, name, got, gok, want, wok))
	}

	if g, w := fmt.Sprintf("%q", selfDispatchModes(body, name)), fmt.Sprintf("%q", oracleDispatchModes(body, name)); g != w {
		bad = append(bad, fmt.Sprintf("selfDispatchModes(%q, %q) = %s, want %s", body, name, g, w))
	}

	return bad
}

// Both readers answer what their expressions answered, the start of the chain
// and each mode, on bodies built from the grammar's tokens, near misses and
// edits of a chain it reads; one body in five holds a chain, so the comparison
// is not of two refusals.
func TestDispatchReadersMatchTheirExpressions(t *testing.T) {
	n := 200_000
	if testing.Short() {
		n = 20_000
	}

	rng := rand.New(rand.NewPCG(1, 7))
	failures, chains, modes := 0, 0, 0

	for range n {
		body, name := randomDispatchBody(rng)

		for _, msg := range dispatchDisagreement(body, name) {
			t.Error(msg)

			failures++

			if failures == 8 {
				return
			}
		}

		if _, ok := oracleDispatchStart(body, name); ok {
			chains++
		}

		if len(oracleDispatchModes(body, name)) > 0 {
			modes++
		}
	}

	if chains < n/5 || modes < n/5 {
		t.Errorf("%d chains and %d with a mode of %d bodies; the generated bodies do not reach the readers", chains, modes, n)
	}
}

// FuzzDispatchReaders holds both readers to their expressions on any body.
func FuzzDispatchReaders(f *testing.F) {
	f.Add("if\t(\targuments\t.\tmode\teq\t\"a\"\t)\t{\treturn\tx\t;\t}\telse\t{\treturn\tthis\t;\t}\t", "mode")
	f.Add("if\t(\targuments\t.\tm\t=\t=\t\"a\"\t)\t{\treturn\tx\t;\t}\telse\tif\t(\targuments\t.\tm\teq\t\"b\"\t)\t{\treturn\ty\t;\t}\telse\t{\treturn\tthis\t;\t}\t", "m")
	f.Add("x\tif\t(\targuments\t.\tmode\teq\t\"a\"\t)\t{\treturn\tx\t;\t}\t", "mode")

	f.Fuzz(func(t *testing.T, body, name string) {
		if name == "" || strings.ToValidUTF8(name, "") != name {
			t.Skip()
		}

		for _, msg := range dispatchDisagreement(body, name) {
			t.Error(msg)
		}
	})
}
