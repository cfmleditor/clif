package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/vfs"

	"github.com/cfmleditor/clif/internal/language"
	cfpath "github.com/cfmleditor/clif/internal/path"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func cmdScan(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, scanUsage)
		os.Exit(1)
	}

	var files []string

	for _, arg := range args {
		info, err := os.Stat(positional(arg, scanUsage))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", arg, err)
			os.Exit(1)
		}

		if info.IsDir() {
			files = append(files, cfpath.CollectCFML(vfs.OS{}, []string{arg})...)
		} else {
			files = append(files, arg)
		}
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "no CFML files found\n")
		os.Exit(1)
	}

	var totalErrors int

	for _, f := range files {
		content, err := vfs.ReadRegular(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", f, err)

			continue
		}

		for _, e := range parseErrors(f, content) {
			fmt.Println(e.String())

			totalErrors++
		}
	}

	if totalErrors == 0 {
		fmt.Printf("No parse errors found in %d files.\n", len(files))
	}
}

// scanError is one place the grammar could not parse.
type scanError struct {
	file    string
	lang    string // cfml, cfscript, or an injected language
	message string // "missing X" or `parse error near "..."`
	line    int    // 1-based
	col     int    // 1-based
}

func (e *scanError) String() string {
	return fmt.Sprintf("%s:%d:%d: [%s] %s", e.file, e.line, e.col, e.lang, e.message)
}

// parseErrors parses content with the grammar its extension calls for, and
// the languages injected into it, and returns every error node, in document
// order. A binary file has none. clif scan prints them; clif check counts them.
func parseErrors(file string, content []byte) []scanError {
	if cfpath.IsBinary(content) {
		return nil
	}

	// Determine initial grammar by extension (matching zed-cfml config)
	grammar, label := language.CFML, "cfml"
	if strings.EqualFold(filepath.Ext(file), ".cfs") {
		grammar, label = language.CFScript, "cfscript"
	}

	tree := language.Parse(grammar, content, nil)
	if tree == nil {
		return nil
	}

	defer tree.Close()

	var out []scanError

	if tree.RootNode().HasError() {
		out = collectErrorNodes(out, file, label, tree.RootNode(), content, 0)
	}

	// Scan injection languages based on injections.scm rules
	return scanInjections(out, file, tree, content)
}

// collectErrorNodes appends every error or missing node under n, stopping at
// the first in each branch, with lines offset by lineOffset for an injected
// language parsed on its own. The count it returns is what tells clif scan
// "clean" from "reported problems": the one it once kept was never
// incremented, so it printed each error and then "No parse errors found".
func collectErrorNodes(out []scanError, file, lang string, n *sitter.Node, src []byte, lineOffset uint) []scanError {
	if n.IsError() || n.IsMissing() {
		pos := n.StartPosition()

		snippet := string(src[n.StartByte():n.EndByte()])
		if len(snippet) > 50 {
			snippet = snippet[:50] + "..."
		}

		snippet = strings.ReplaceAll(snippet, "\n", "\\n")

		msg := fmt.Sprintf("parse error near %q", snippet)
		if n.IsMissing() {
			msg = "missing " + n.Kind()
		}

		return append(out, scanError{
			file:    file,
			lang:    lang,
			message: msg,
			line:    int(pos.Row+lineOffset) + 1,
			col:     int(pos.Column) + 1,
		})
	}

	for i := range n.ChildCount() {
		out = collectErrorNodes(out, file, lang, n.Child(i), src, lineOffset)
	}

	return out
}

func scanInjections(out []scanError, file string, tree *sitter.Tree, src []byte) []scanError {
	for _, inj := range language.FindInjections(tree, src) {
		grammar := language.GrammarForLanguage(inj.Language)
		if grammar < 0 {
			continue
		}

		content := src[inj.Node.StartByte():inj.Node.EndByte()]

		injTree := language.Parse(grammar, content, nil)
		if injTree != nil {
			if injTree.RootNode().HasError() {
				out = collectErrorNodes(out, file, inj.Language, injTree.RootNode(), content, inj.Node.StartPosition().Row)
			}

			injTree.Close()
		}
	}

	return out
}
