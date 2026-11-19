package resolve

import (
	"reflect"
	"slices"
	"testing"

	"github.com/cfmleditor/clif/internal/parser"
)

func TestWithoutNewlinesPreservesSharedTokens(t *testing.T) {
	input := []parser.Token{{Kind: parser.TokIdent, Value: "one", Offset: 4}, {Kind: parser.TokNewline}, {Kind: parser.TokIdent, Value: "two", Offset: 12}}
	before := slices.Clone(input)

	got := withoutNewlines(input)
	if !slices.Equal(got, []parser.Token{input[0], input[2]}) || !slices.Equal(input, before) {
		t.Fatalf("filtered %v; input %v", got, input)
	}

	got[0].Value = "changed"

	if !slices.Equal(input, before) {
		t.Fatal("filtering reused the shared input's storage")
	}

	input = before[:1]

	allocs := testing.AllocsPerRun(100, func() {
		if got := withoutNewlines(input); &got[0] != &input[0] {
			panic("unfiltered input was copied")
		}
	})
	if allocs != 0 {
		t.Fatalf("unfiltered tokens allocate %v times", allocs)
	}

	if withoutNewlines(nil) != nil {
		t.Fatal("nil tokens became non-nil")
	}
}

func TestCallArgumentScannerPreservesStreamingReader(t *testing.T) {
	for _, expression := range []string{
		`factory()`, `factory /* before */ ( /* inside */ ) /* after */`,
		`factory(name="(x,y)", {one:[1,2]}, nested(a,b))`,
		`factory("#nested('value')#")`, `factory(1).next()`,
		`factory(1) + other`, `factory(`, `factory({]})`, `value`, ``,
	} {
		t.Run(expression, func(t *testing.T) {
			// The previous Wheels reader is the oracle for argument tokens,
			// including empty lists, interpolation and malformed input.
			scanner := parser.NewSourceScanner(expression)
			found := false

			for scanner.PeekSkipComments().Kind != parser.TokEOF {
				if scanner.NextSkipComments().Kind == parser.TokLParen {
					found = true

					break
				}
			}

			var want [][]parser.Token

			if found {
				want = beanArguments(scanner)
				if scanner.NextSkipComments().Kind != parser.TokEOF {
					want = nil
				}
			}

			var got [][]parser.Token

			if scanner := callArgumentScanner(expression); scanner != nil {
				got = beanArguments(scanner)
				if scanner.NextSkipComments().Kind != parser.TokEOF {
					got = nil
				}
			}

			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}
