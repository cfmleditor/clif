package parser

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/cfmleditor/clif/internal/lex"
)

// columnFromLineStart is what a column was before the cursor: the UTF-16 length
// of the line up to the offset, measured from the line's start.
func columnFromLineStart(src string, offset int) int {
	return lex.UTF16Len(src[strings.LastIndexByte(src[:offset], '\n')+1 : offset])
}

// runeStarts are the offsets a parse can ask about: where a character begins.
func runeStarts(src string) []int {
	var out []int

	for i := 0; i <= len(src); {
		out = append(out, i)

		if i == len(src) {
			break
		}

		_, size := utf8.DecodeRuneInString(src[i:])
		i += size
	}

	return out
}

func TestColumnCursorAgreesWithMeasuringFromTheLineStart(t *testing.T) {
	texts := map[string]string{
		"lines":           "component {\n\tfunction a() {}\n\tfunction b() {}\n}\n",
		"wide characters": "é = 1; 😀 function 日本語() {} 😀😀 function z() {}\nx = \"é\"; 😀\n\n日本\n",
		"crlf":            "a\r\nfunction f() {}  function g() {}\r\n\r\nlast",
		"one long line":   strings.Repeat("function f(){} é ", 400) + "\nnext line\n",
		"no newline":      "function f() {} function g() {}",
		"empty":           "",
	}

	for name, src := range texts {
		starts := runeStarts(src)
		rng := rand.New(rand.NewPCG(1, 2))

		orders := map[string][]int{"ascending": starts}

		descending := make([]int, len(starts))
		for i, off := range starts {
			descending[len(starts)-1-i] = off
		}

		orders["descending"] = descending

		random := make([]int, 3000)
		for i := range random {
			random[i] = starts[rng.IntN(len(starts))]
		}

		orders["random"] = random

		// A property asks for its start, its end and then its name between.
		var nearby []int

		for _, off := range starts {
			nearby = append(nearby, off, min(off+7, len(src)), off+1)
		}

		orders["start, end, then between"] = nearby

		for order, offsets := range orders {
			var cur columnCursor

			for _, off := range offsets {
				if off > len(src) || !utf8.RuneStart(byteAt(src, off)) {
					continue
				}

				if got, want := cur.column(src, off), columnFromLineStart(src, off); got != want {
					t.Fatalf("%s, %s: column at offset %d = %d, want %d", name, order, off, got, want)
				}
			}
		}
	}
}

// byteAt is src[i], or a rune start for the offset past the end.
func byteAt(src string, i int) byte {
	if i >= len(src) {
		return 'x'
	}

	return src[i]
}

// A cursor that meets another text answers for that one, not the last.
func TestColumnCursorFollowsTheText(t *testing.T) {
	var cur columnCursor

	a := "abc def"
	b := "xyz\n日本語 ghi"

	if got := cur.column(a, 4); got != 4 {
		t.Fatalf("column in a = %d, want 4", got)
	}

	if got, want := cur.column(b, len("xyz\n日本語 ")), 4; got != want {
		t.Errorf("column in another text = %d, want %d", got, want)
	}

	// Back to the first, with the cursor now on the second.
	if got := cur.column(a, 2); got != 2 {
		t.Errorf("column back in a = %d, want 2", got)
	}
}

func TestColumnCursorWrappersMatchTheStatelessFunctions(t *testing.T) {
	src := "é function a() {} 😀 function bb() {}\nx 日本 function ccc() {}\n"

	var cur columnCursor

	for _, name := range []string{"a", "bb", "ccc"} {
		off := strings.Index(src, "function "+name) + len("function ")

		line := strings.Count(src[:off], "\n")

		gl, gc, ge := cur.nameRange(src, off, line, name)
		wl, wc, we := nameRangeAt(src, off, line, name)

		if gl != wl || gc != wc || ge != we {
			t.Errorf("nameRange(%s) = %d:%d-%d, want %d:%d-%d", name, gl, gc, ge, wl, wc, we)
		}

		if got, want := cur.prefix(src, off), linePrefixUTF16(src, off); got != want {
			t.Errorf("prefix(%s) = %d, want %d", name, got, want)
		}
	}

	// The guards of both: nothing at a line's start, an empty name, an offset out of range.
	if got := cur.prefix(src, 0); got != 0 {
		t.Errorf("prefix at 0 = %d, want 0", got)
	}

	if l, c, e := cur.nameRange(src, 3, 0, ""); l != 0 || c != 0 || e != 0 {
		t.Errorf("nameRange of an empty name = %d:%d-%d, want zeros", l, c, e)
	}

	if l, c, e := cur.nameRange(src, len(src)+1, 0, "x"); l != 0 || c != 0 || e != 0 {
		t.Errorf("nameRange past the end = %d:%d-%d, want zeros", l, c, e)
	}
}

// Declarations written on one very long line are found in time proportional to
// the line: 16,000 functions parsed in 2.1 s when each name's column was
// measured from the line's start, and take a fraction of that now.
func TestParseOfManyFunctionsOnOneLineIsLinear(t *testing.T) {
	src := "component {\n" + strings.Repeat("function f(){}", 16000) + "\n}\n"

	start := time.Now()
	pr := Parse("file:///w/Long.cfc", src)

	if took := time.Since(start); took > 1200*time.Millisecond {
		t.Errorf("parsing 16,000 functions on one line took %v, want well under that", took)
	}

	if len(pr.Funcs) != 16000 {
		t.Fatalf("found %d functions, want 16000", len(pr.Funcs))
	}

	// The last function's name is where it is written: after 15,999 of the
	// 14-character declarations and the word "function ".
	last := pr.Funcs[len(pr.Funcs)-1]
	if want := uint32(len("function f(){}")*15999 + len("function ")); last.NameCol != want || last.NameEndCol != want+1 {
		t.Errorf("last name at %d-%d, want %d-%d", last.NameCol, last.NameEndCol, want, want+1)
	}
}

// The same holds for the other declarations that find a column: properties in
// script, and functions and properties written as tags. 8,000 properties on one
// line parsed in 3.9 s (4.3 s as tags) when each was measured from the line's
// start; the bound is generous, the parse takes tens of milliseconds.
func TestParseOfManyDeclarationsOnOneLineIsLinear(t *testing.T) {
	for _, c := range []struct {
		name  string
		src   string
		want  int
		props bool
	}{
		{"script properties", "component {\n" + strings.Repeat(`property name="p" type="string";`, 8000) + "\n}\n", 8000, true},
		{"tag properties", "<cfcomponent>\n" + strings.Repeat(`<cfproperty name="p" type="string">`, 8000) + "\n</cfcomponent>\n", 8000, true},
		{"tag functions", "<cfcomponent>\n" + strings.Repeat(`<cffunction name="f"></cffunction>`, 16000) + "\n</cfcomponent>\n", 16000, false},
	} {
		start := time.Now()
		pr := Parse("file:///w/Long.cfc", c.src)

		if took := time.Since(start); took > time.Second {
			t.Errorf("%s: parsing %d on one line took %v, want well under a second", c.name, c.want, took)
		}

		line := strings.Split(c.src, "\n")[1]

		if c.props {
			props := pr.PropertySymbols()
			if len(props) != c.want {
				t.Errorf("%s: found %d properties, want %d", c.name, len(props), c.want)

				continue
			}

			// The name is the letter inside the quotes of the last name="p".
			last := props[len(props)-1]
			if want := uint32(strings.LastIndex(line, `name="p"`) + len(`name="`)); last.NameCol != want || last.NameEndCol != want+1 {
				t.Errorf("%s: last property's name at %d-%d, want %d-%d", c.name, last.NameCol, last.NameEndCol, want, want+1)
			}

			continue
		}

		if len(pr.Funcs) != c.want {
			t.Errorf("%s: found %d functions, want %d", c.name, len(pr.Funcs), c.want)

			continue
		}

		last := pr.Funcs[len(pr.Funcs)-1]
		if want := uint32(strings.LastIndex(line, `name="f"`) + len(`name="`)); last.NameCol != want || last.NameEndCol != want+1 {
			t.Errorf("%s: last function's name at %d-%d, want %d-%d", c.name, last.NameCol, last.NameEndCol, want, want+1)
		}
	}
}
