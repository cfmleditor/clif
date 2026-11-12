package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
)

type lexicalSourceFacts struct {
	starts     []int
	queryWords [][2]int
}

// One pass records line boundaries and word occurrences. Facts contain no
// inferred types or caller context and are immutable.
func scanSourceFacts(content string) lexicalSourceFacts {
	facts := lexicalSourceFacts{starts: []int{0}}

	for i := 0; i < len(content); {
		if content[i] == '\n' {
			i++
			facts.starts = append(facts.starts, i)

			continue
		}

		if lex.IsWord(content[i]) {
			start := i
			for i < len(content) && lex.IsWord(content[i]) {
				i++
			}

			word := content[start:i]
			if strings.EqualFold(word, "query") || strings.EqualFold(word, "cfquery") || strings.EqualFold(word, "cfdbinfo") {
				facts.queryWords = append(facts.queryWords, [2]int{start, i})
			}

			continue
		}

		i++
	}

	return facts
}

func (r *Resolver) lexicalSource(pr *parser.ParseResult) *lexicalSourceFacts {
	if r.indexer != nil {
		return r.indexer.lexicalSource(pr)
	}

	facts := r.sourceFacts.get(&r.mu, pr, func() []lexicalSourceFacts { return []lexicalSourceFacts{scanSourceFacts(pr.Content)} })

	return &facts[0]
}

func (r *Resolver) sourceLineStarts(pr *parser.ParseResult) []int {
	return r.lexicalSource(pr).starts
}

func sourceLine(content string, starts []int, line int) string {
	end := len(content)
	if line+1 < len(starts) {
		end = starts[line+1] - 1
	}

	return content[starts[line]:end]
}
