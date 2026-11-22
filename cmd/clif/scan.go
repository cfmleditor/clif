package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/language"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

func cmdScan(args []string) {
	var (
		outPath string
		format  string
		paths   []string
	)

	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--out":
			if i+1 >= len(args) {
				fatalf("--out needs a value\n\n%s", scanUsage)
			}

			i++
			outPath = args[i]
		case "--format":
			if i+1 >= len(args) {
				fatalf("--format needs a value\n\n%s", scanUsage)
			}

			i++
			format = args[i]
		case "--json":
			format = "json"
		default:
			paths = append(paths, positional(a, scanUsage))
		}
	}

	if len(paths) == 0 {
		fmt.Fprint(os.Stderr, scanUsage)
		os.Exit(1)
	}

	if format == "" {
		format = defaultScanFormat(outPath)
	}

	switch format {
	case "text", "md", "markdown", "json":
	default:
		fatalf("unknown format %q (want text, md or json)\n", format)
	}

	var files []string

	for _, arg := range paths {
		info, err := os.Stat(arg)
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

	slices.Sort(files)

	var errs []scanError

	for _, f := range files {
		content, err := vfs.ReadRegular(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  skip %s: %v\n", f, err)

			continue
		}

		errs = append(errs, parseErrors(f, content)...)
	}

	out := os.Stdout

	if outPath != "" {
		file, err := os.Create(outPath)
		if err != nil {
			fatalf("creating %s: %v\n", outPath, err)
		}

		defer func() { _ = file.Close() }()

		out = file
	}

	w := bufio.NewWriter(out)

	var err error

	switch format {
	case "json":
		err = writeScanJSON(w, errs, len(files))
	case "md", "markdown":
		err = writeScanMarkdown(w, errs, len(files))
	default:
		err = writeScanText(w, errs, len(files))
	}

	if err != nil {
		fatalf("%v\n", err)
	}

	if err := w.Flush(); err != nil {
		fatalf("writing output: %v\n", err)
	}

	if outPath != "" {
		fmt.Fprintf(os.Stderr, "Wrote %s (%d parse errors in %d files)\n", outPath, len(errs), len(files))
	}
}

func defaultScanFormat(outPath string) string {
	switch strings.ToLower(filepath.Ext(outPath)) {
	case ".md", ".markdown":
		return "md"
	case ".json":
		return "json"
	default:
		return "text"
	}
}

func writeScanText(w io.Writer, errs []scanError, scanned int) error {
	for i := range errs {
		if _, err := fmt.Fprintln(w, errs[i].String()); err != nil {
			return err
		}
	}

	if len(errs) == 0 {
		_, err := fmt.Fprintf(w, "No parse errors found in %d files.\n", scanned)

		return err
	}

	return nil
}

func writeScanJSON(w io.Writer, errs []scanError, scanned int) error {
	type jsonErr struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Col     int    `json:"col"`
		Lang    string `json:"lang"`
		Message string `json:"message"`
		Source  string `json:"source,omitempty"`
	}

	failed := map[string]struct{}{}
	out := make([]jsonErr, 0, len(errs))

	for i := range errs {
		e := &errs[i]
		failed[e.file] = struct{}{}
		out = append(out, jsonErr{
			File:    filepath.ToSlash(e.file),
			Line:    e.line,
			Col:     e.col,
			Lang:    e.lang,
			Message: e.message,
			Source:  e.source,
		})
	}

	return writeJSON(w, map[string]any{
		"scanned":     scanned,
		"failedFiles": len(failed),
		"errors":      out,
	})
}

func writeScanMarkdown(w io.Writer, errs []scanError, scanned int) error {
	var b strings.Builder
	b.WriteString("# Parse Errors\n\n")

	if len(errs) == 0 {
		fmt.Fprintf(&b, "No parse errors found in %d files.\n", scanned)
		_, err := io.WriteString(w, b.String())

		return err
	}

	type fileGroup struct {
		file  string
		items []*scanError
	}

	var groups []fileGroup

	groupIdx := map[string]int{}

	for i := range errs {
		e := &errs[i]
		key := filepath.ToSlash(e.file)

		idx, ok := groupIdx[key]
		if !ok {
			idx = len(groups)
			groupIdx[key] = idx
			groups = append(groups, fileGroup{file: key})
		}

		groups[idx].items = append(groups[idx].items, e)
	}

	fmt.Fprintf(&b, "Scanned %d files: %d parse error(s) in %d file(s).\n", scanned, len(errs), len(groups))

	for _, g := range groups {
		fmt.Fprintf(&b, "\n## `%s` (%d)\n\n", g.file, len(g.items))

		for _, it := range g.items {
			msg := strings.Join(strings.Fields(it.message), " ")
			fmt.Fprintf(&b, "- `%s:%d:%d` — [%s] %s\n", g.file, it.line, it.col, it.lang, msg)

			if it.source != "" {
				fence := "```"
				if strings.Contains(it.source, "```") {
					fence = "````"
				}

				fmt.Fprintf(&b, "  %scfml\n  %s\n  %s\n", fence, it.source, fence)
			}
		}
	}

	_, err := io.WriteString(w, b.String())

	return err
}

// scanError is one place the grammar could not parse.
type scanError struct {
	file    string
	lang    string // cfml, cfscript, or an injected language
	message string // "missing X" or `parse error near "..."`
	source  string // trimmed source line where the error begins
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
	content, isText := cfpath.AsText(content)
	if !isText {
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
			source:  lineAt(src, pos.Row),
			line:    int(pos.Row+lineOffset) + 1,
			col:     int(pos.Column) + 1,
		})
	}

	for i := range n.ChildCount() {
		out = collectErrorNodes(out, file, lang, n.Child(i), src, lineOffset)
	}

	return out
}

func lineAt(src []byte, row uint) string {
	start := 0

	for range row {
		idx := bytes.IndexByte(src[start:], '\n')
		if idx < 0 {
			return ""
		}

		start += idx + 1
	}

	rest := src[start:]
	if idx := bytes.IndexByte(rest, '\n'); idx >= 0 {
		rest = rest[:idx]
	}

	return strings.TrimSpace(strings.TrimSuffix(string(rest), "\r"))
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
