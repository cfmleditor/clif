package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
)

// activeTokens is the walk behind the readers that take a file's configuration
// from its tokens, so comments and quoted examples cannot manufacture a
// setting: script regions are read throughout, tag regions only inside a
// <cfset ...>. A literal <script> block is skipped.
type activeTokens struct {
	Scanner  *parser.Scanner
	Previous []parser.Token // the tokens just before the current one, oldest first
	Region   parser.Region
	window   int
}

// walkActiveTokens calls visit for every token inside code. visit returns true
// when it consumed the token (and the construct after it), in which case the
// token is not added to Previous; window bounds how many previous tokens are
// kept, so a large file does not grow the slice.
func walkActiveTokens(content string, window int, visit func(a *activeTokens, tok *parser.Token) (consumed bool)) {
	for _, region := range parser.ClassifyRegions(content) {
		if region.Kind == parser.RegionSkip {
			continue
		}

		a := &activeTokens{Scanner: parser.NewScanner(region.Text), Previous: []parser.Token{}, Region: region, window: window}
		active := region.Kind == parser.RegionScript

		for {
			tok := a.Scanner.NextSkipComments()
			if tok.Kind == parser.TokEOF {
				break
			}

			if region.Kind == parser.RegionTag {
				if tok.Kind == parser.TokIdent && strings.EqualFold(tok.Value, "cfset") && len(a.Previous) > 0 && a.Previous[len(a.Previous)-1].Kind == parser.TokLT {
					active = true
					a.Previous = nil

					continue
				}

				if tok.Kind == parser.TokGT {
					active = false
					a.Previous = nil

					continue
				}
			}

			if active && visit(a, &tok) {
				a.Previous = nil

				continue
			}

			a.Previous = append(a.Previous, tok)
			if len(a.Previous) > a.window {
				a.Previous = a.Previous[1:]
			}
		}
	}
}
