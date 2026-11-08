package formatter

import (
	"errors"
	"fmt"

	cfpath "github.com/cfmleditor/clif/internal/path"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// SourceError refuses what is not UTF-8 text before it is parsed: a binary file
// has no CFML to format and costs the grammar seconds to recover from, and a
// UTF-16 file would be rewritten as UTF-8, changing its encoding with its
// layout. Both formatting entry points call it ahead of ParseError, so a file
// one refuses is a file the other refuses.
func SourceError(src []byte) error {
	switch {
	case cfpath.IsUTF16(src):
		return errors.New("the file is UTF-16, which clif does not format (it would be rewritten as UTF-8)")
	case cfpath.IsBinary(src):
		return errors.New("the file is binary, not CFML")
	}

	return nil
}

// ParseError reports the first ERROR or MISSING node in tree, or nil when the
// tree is clean.
//
// Formatting a source file the grammar could not parse is how a CST gap turns
// into rewritten source: the walk emits whatever it can make sense of and
// silently drops the rest, leaving only [Options.WhitespaceOnly] between a
// grammar gap and the user's file. Both entry points that format a whole file
// — the LSP's textDocument/formatting handler and the `format` subcommand —
// gate on this first, so a file the editor refuses is a file the CLI refuses.
func ParseError(tree *sitter.Tree, src []byte) error {
	// No tree is a parse that did not finish (language.MaxParseTime): there
	// is nothing to format from, and reading one panicked the request.
	if tree == nil {
		return errors.New("the document took too long to parse, cannot format")
	}

	root := tree.RootNode()
	if !root.HasError() {
		return nil
	}

	errNode := findErrorNode(root)
	if errNode == nil {
		return errors.New("parse error in document, cannot format")
	}

	pos := errNode.StartPosition()

	snippet := string(src[errNode.StartByte():errNode.EndByte()])
	if len(snippet) > 50 {
		snippet = snippet[:50] + "..."
	}

	return fmt.Errorf("parse error at line %d, col %d near %q", pos.Row+1, pos.Column+1, snippet)
}

// findErrorNode returns the first ERROR or MISSING node in the subtree at n.
func findErrorNode(n *sitter.Node) *sitter.Node {
	if n.IsError() || n.IsMissing() {
		return n
	}

	for i := range n.ChildCount() {
		if found := findErrorNode(n.Child(i)); found != nil {
			return found
		}
	}

	return nil
}
