package server

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/language"
	cflog "github.com/cfmleditor/clif/internal/log"
	cfpath "github.com/cfmleditor/clif/internal/path"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

const defaultParseErrorReport = "clif-parse-errors.md"

type parseErrorFinding struct {
	file    string
	line    uint32 // 1-based
	col     uint32 // 1-based
	message string
	source  string
}

// scanFiles is every CFML file scanWorkspace will read, from searchRoots
// rather than the configured folders alone — without that a session with no
// .clif.json scanned nothing and reported "0 parse errors in 0 files",
// which reads like a clean workspace.
//
// Split out from scanWorkspace because the scan itself only reports through
// notifications, so this is the part a test can see.
func (s *Server) scanFiles() []string {
	return cfpath.CollectCFML(s.FS, s.searchRoots())
}

func (s *Server) scanWorkspace(ctx context.Context, outName string) {
	files := s.scanFiles()
	slices.Sort(files)

	s.log.Info("scanWorkspace: starting", cflog.Int("files", len(files)))

	var (
		totalErrors int
		findings    []parseErrorFinding
	)

	roots := s.searchRoots()
	baseRoot := ""

	if len(roots) > 0 {
		baseRoot = roots[0]
	}

	for _, f := range files {
		data, err := s.FS.ReadFile(f)
		if err != nil {
			continue
		}

		data, isText := cfpath.AsText(data)
		if !isText {
			continue
		}

		grammar := language.CFML
		if strings.EqualFold(filepath.Ext(f), ".cfs") {
			grammar = language.CFScript
		}

		tree := language.Parse(grammar, data, nil)
		if tree == nil {
			continue
		}

		if !tree.RootNode().HasError() {
			tree.Close()

			// A file an earlier scan found errors in and that now parses
			// clean has them withdrawn, rather than left until it is opened.
			if s.hasDiagnostics(uri.File(f), sourceParse) {
				s.setDiagnostics(ctx, uri.File(f), sourceParse, nil)
			}

			continue
		}

		diags := collectErrorDiagnostics(tree.RootNode(), data)
		tree.Close()

		if len(diags) > 0 {
			totalErrors += len(diags)
			s.setDiagnostics(ctx, uri.File(f), sourceParse, diags)

			lines := strings.Split(string(data), "\n")
			displayFile := relativeScanPath(baseRoot, f)

			for i := range diags {
				d := &diags[i]

				var msg string
				if s, ok := d.Message.(protocol.String); ok {
					msg = string(s)
				}

				srcLine := ""

				if int(d.Range.Start.Line) < len(lines) {
					srcLine = strings.TrimSpace(strings.TrimSuffix(lines[d.Range.Start.Line], "\r"))
				}

				findings = append(findings, parseErrorFinding{
					file:    displayFile,
					line:    d.Range.Start.Line + 1,
					col:     d.Range.Start.Character + 1,
					message: msg,
					source:  srcLine,
				})
			}
		}
	}

	s.log.Info("scanWorkspace: complete", cflog.Int("files", len(files)), cflog.Int("errors", totalErrors))

	msg := fmt.Sprintf("Scan complete: %d parse errors in %d files", totalErrors, len(files))

	// Write a Markdown report whenever parse errors were found, or when the
	// caller explicitly requested an output file.
	if (totalErrors > 0 || outName != "") && baseRoot != "" {
		reportName := outName
		if reportName == "" {
			reportName = defaultParseErrorReport
		}

		outPath, err := s.reportPath(baseRoot, reportName)
		if err != nil {
			s.notifyError(ctx, err.Error())

			return
		}

		doc := formatParseErrorReport(findings, len(files))
		if err := writeReport(outPath, []byte(doc)); err != nil {
			s.log.Error("failed to write parse error report", cflog.String("path", outPath), cflog.Err(err))
			s.notifyError(ctx, "Could not write "+outPath+": "+err.Error())

			return
		}

		msg += " — wrote " + outPath
	}

	s.notify(ctx, protocol.MethodWindowShowMessage, &protocol.ShowMessageParams{
		Type:    protocol.MessageTypeInfo,
		Message: msg,
	})
}

func relativeScanPath(root, file string) string {
	if root != "" {
		if rel, err := filepath.Rel(root, file); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}

	return filepath.ToSlash(file)
}

func formatParseErrorReport(findings []parseErrorFinding, scannedFiles int) string {
	var b strings.Builder
	b.WriteString("# Parse Errors\n\n")

	if len(findings) == 0 {
		fmt.Fprintf(&b, "No parse errors found in %d files.\n", scannedFiles)

		return b.String()
	}

	type fileGroup struct {
		file  string
		items []parseErrorFinding
	}

	var groups []fileGroup

	groupIdx := map[string]int{}

	for _, f := range findings {
		idx, ok := groupIdx[f.file]
		if !ok {
			idx = len(groups)
			groupIdx[f.file] = idx
			groups = append(groups, fileGroup{file: f.file})
		}

		groups[idx].items = append(groups[idx].items, f)
	}

	fmt.Fprintf(&b, "Scanned %d files: %d parse error(s) in %d file(s).\n", scannedFiles, len(findings), len(groups))

	for _, g := range groups {
		fmt.Fprintf(&b, "\n## `%s` (%d)\n\n", g.file, len(g.items))

		for _, it := range g.items {
			msg := strings.Join(strings.Fields(it.message), " ")
			fmt.Fprintf(&b, "- `%s:%d:%d` — %s\n", it.file, it.line, it.col, msg)

			if it.source != "" {
				fence := "```"
				if strings.Contains(it.source, "```") {
					fence = "````"
				}

				fmt.Fprintf(&b, "  %scfml\n  %s\n  %s\n", fence, it.source, fence)
			}
		}
	}

	return b.String()
}

func collectErrorDiagnostics(n *sitter.Node, src []byte) []protocol.Diagnostic {
	var diags []protocol.Diagnostic
	collectErrors(n, src, newColMapper(string(src)), &diags)

	return diags
}

// tree-sitter reports columns in bytes; cols turns them into LSP characters.
func collectErrors(n *sitter.Node, src []byte, cols *colMapper, diags *[]protocol.Diagnostic) {
	if n.IsError() || n.IsMissing() {
		start := n.StartPosition()
		end := n.EndPosition()

		snippet := string(src[n.StartByte():n.EndByte()])
		if len(snippet) > 50 {
			snippet = snippet[:50] + "..."
		}

		msg := "Parse error"
		if n.IsMissing() {
			msg = "Missing " + n.Kind()
		} else if snippet != "" {
			msg = "Parse error near " + strings.TrimSpace(snippet)
		}

		*diags = append(*diags, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: conv.Uint32FromUint(start.Row), Character: cols.col(conv.Uint32FromUint(start.Row), conv.Uint32FromUint(start.Column))},
				End:   protocol.Position{Line: conv.Uint32FromUint(end.Row), Character: cols.col(conv.Uint32FromUint(end.Row), conv.Uint32FromUint(end.Column))},
			},
			Severity: protocol.DiagnosticSeverityError,
			Source:   protocol.NewOptional("cfmleditor"),
			Message:  protocol.String(msg),
		})

		return
	}

	for i := range n.ChildCount() {
		collectErrors(n.Child(i), src, cols, diags)
	}
}
