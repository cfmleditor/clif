// Package language provides tree-sitter language and parser setup for CFML grammars.
package language

import (
	"fmt"
	"time"

	tree_sitter_cfml "github.com/cfmleditor/tree-sitter-cfml/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Grammar identifies which tree-sitter grammar to use.
type Grammar int

// Supported grammars.
const (
	CFML Grammar = iota
	CFScript
	CFQuery
)

// Language returns the tree-sitter Language for the given grammar.
func Language(g Grammar) *sitter.Language {
	switch g {
	case CFML:
		return sitter.NewLanguage(tree_sitter_cfml.LanguageCfml())
	case CFScript:
		return sitter.NewLanguage(tree_sitter_cfml.LanguageCfscript())
	case CFQuery:
		return sitter.NewLanguage(tree_sitter_cfml.LanguageCfquery())
	default:
		return nil
	}
}

// NewParser creates a tree-sitter parser configured for the given grammar.
// The caller must call Close() on the returned parser when done.
func NewParser(g Grammar) *sitter.Parser {
	p := sitter.NewParser()

	// A panic, not an exit: this runs inside a request (formatting) whose
	// recovery fails that one request, where os.Exit took the language
	// server and every editor of a daemon with it.
	if err := p.SetLanguage(Language(g)); err != nil {
		p.Close()
		panic(fmt.Sprintf("tree-sitter: cannot set the %v grammar: %v", g, err))
	}

	return p
}

// MaxParseTime bounds one tree-sitter parse. tree-sitter's parser never
// returned on a 3MB page of 200,000 nested <cfif> tags, and a format request
// or a workspace scan then held a core for good. The corpus's largest files
// parse in well under a second.
var MaxParseTime = 10 * time.Second

// Parse parses source code using the given grammar and returns the tree, or
// nil when the parse did not finish within MaxParseTime. The caller must call
// Close() on the returned tree when done.
func Parse(g Grammar, src []byte, oldTree *sitter.Tree) *sitter.Tree {
	p := NewParser(g)
	defer p.Close()

	deadline := time.Now().Add(MaxParseTime)
	length := len(src)

	return p.ParseWithOptions(func(i int, _ sitter.Point) []byte {
		if i < length {
			return src[i:]
		}

		return []byte{}
	}, oldTree, &sitter.ParseOptions{ProgressCallback: func(sitter.ParseState) bool {
		return time.Now().After(deadline)
	}})
}
