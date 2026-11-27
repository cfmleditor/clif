// Package lint provides the backend-agnostic lint engine interface, .cflintrc
// rule configuration loader, inline suppression matcher, native pure-Go rule
// runner, hybrid fallback runner, and parity comparison harness.
package lint

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/language"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	sitter "github.com/tree-sitter/go-tree-sitter"
	"go.lsp.dev/protocol"
)

// Engine is a lint backend capable of linting files on disk, in-memory staged
// sources, or a single file for LSP diagnostics. *cflint.Runner satisfies this
// interface directly.
type Engine interface {
	Lint(ctx context.Context, files []string) (*cflint.Run, error)
	LintContent(ctx context.Context, sources []cflint.Source) (*cflint.Run, error)
	Scan(ctx context.Context, filePath string) ([]protocol.Diagnostic, error)
	ScanFiles(ctx context.Context, files []string) (map[string][]protocol.Diagnostic, error)
}

// Rule describes one native lint rule.
type Rule struct {
	// Code is the canonical rule identifier (e.g. "MISSING_VAR", "OUTPUT_ATTR").
	Code string
	// DefaultSeverity is the rule's default severity on the standard scale
	// (FATAL, CRITICAL, ERROR, WARNING, CAUTION, INFO, COSMETIC).
	DefaultSeverity string
	// Check inspects one file and returns any raw issues found. The runner
	// applies .cflintrc severity overrides, inline suppressions, and
	// minSeverity filtering automatically.
	Check func(fc *FileContext) []cflint.Issue
}

// Registry holds a set of native lint rules keyed by uppercased rule code.
type Registry struct {
	mu    sync.RWMutex
	rules map[string]Rule
	order []string
}

// NewRegistry returns an empty rule registry.
func NewRegistry() *Registry {
	return &Registry{rules: make(map[string]Rule)}
}

// Register adds or replaces a rule in the registry.
func (r *Registry) Register(rule Rule) {
	code := strings.ToUpper(strings.TrimSpace(rule.Code))
	rule.Code = code

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rules[code]; !exists {
		r.order = append(r.order, code)
	}

	r.rules[code] = rule
}

// Has reports whether the registry implements code.
func (r *Registry) Has(code string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.rules[strings.ToUpper(strings.TrimSpace(code))]

	return ok
}

// Rules returns all registered rules in deterministic registration order.
func (r *Registry) Rules() []Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Rule, 0, len(r.order))
	for _, code := range r.order {
		out = append(out, r.rules[code])
	}

	return out
}

// Codes returns all registered rule codes sorted alphabetically.
func (r *Registry) Codes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := slices.Clone(r.order)
	slices.Sort(out)

	return out
}

// DefaultRegistry is the global registry of native clif lint rules.
var DefaultRegistry = NewRegistry()

// FileContext provides parsed and raw representations of a single CFML file to
// native lint rules, lazily parsing the tree-sitter CST once per file.
type FileContext struct {
	Path    string
	Content []byte
	RC      *RCConfig

	textOnce sync.Once
	text     string

	linesOnce sync.Once
	lines     []string

	lineTableOnce sync.Once
	lineTable     parser.LineTable

	regionsOnce sync.Once
	regions     []parser.Region

	parseOnce sync.Once
	parsed    *parser.ParseResult

	scopesOnce sync.Once
	scopes     []*funcScopeAnalysis

	treeOnce sync.Once
	tree     *sitter.Tree
}

// functionScopes lazily analyzes every FunctionSpan in the file once so all
// scope rules (MISSING_VAR, IMPLICIT_SCOPE, ARG_VAR_CONFLICT, ARG_VAR_MIXED)
// share a single pass.
func (fc *FileContext) functionScopes() []*funcScopeAnalysis {
	fc.scopesOnce.Do(func() {
		src := fc.Text()
		spans := fc.ParseResult().FunctionSpans()
		out := make([]*funcScopeAnalysis, len(spans))

		for i := range spans {
			out[i] = analyzeFunctionScope(src, &spans[i])
		}

		fc.scopes = out
	})

	return fc.scopes
}

// ParseResult lazily parses the file with clif's internal parser and caches
// the result (including FunctionSpans) for rules that inspect function scopes.
func (fc *FileContext) ParseResult() *parser.ParseResult {
	fc.parseOnce.Do(func() {
		fc.parsed = parser.Parse(cfpath.ToURI(fc.Path), fc.Text())
	})

	return fc.parsed
}

// Text returns Content as a string, cached once per file.
func (fc *FileContext) Text() string {
	fc.textOnce.Do(func() {
		fc.text = string(fc.Content)
	})

	return fc.text
}

// LineTable returns the file's LineTable for O(log N) offset -> line mapping.
func (fc *FileContext) LineTable() parser.LineTable {
	fc.lineTableOnce.Do(func() {
		fc.lineTable = parser.NewLineTable(fc.Text())
	})

	return fc.lineTable
}

// Regions returns the classified tag/script regions of the file.
func (fc *FileContext) Regions() []parser.Region {
	fc.regionsOnce.Do(func() {
		src := fc.Text()
		regions := parser.ClassifyRegions(src)

		// A .cfm template is never pure CFScript outside explicit <cfscript>
		// tags; ClassifyRegions falls back to RegionScript when a file has no
		// <cf...> tags at all (e.g. a pure HTML partial), so keep it RegionTag.
		if strings.EqualFold(filepath.Ext(fc.Path), ".cfm") &&
			len(regions) == 1 && regions[0].Kind == parser.RegionScript &&
			parser.IndexFold(src, "<cfscript") < 0 {
			regions[0].Kind = parser.RegionTag
		}

		fc.regions = regions
	})

	return fc.regions
}

// Pos1 converts a 0-based byte offset into 1-based (line, column).
func (fc *FileContext) Pos1(offset int) (line, col int) {
	lt := fc.LineTable()
	l := lt.LineOf(offset)
	start, _ := lt.Start(l)

	return l + 1, offset - start + 1
}

// IssueAt constructs a single-location cflint.Issue at byte offset.
func (fc *FileContext) IssueAt(offset int, msg, variable string) cflint.Issue {
	line, col := fc.Pos1(offset)

	return cflint.Issue{
		Message: msg,
		Locations: []cflint.Location{{
			File:     fc.Path,
			FileName: filepath.Base(fc.Path),
			Line:     line,
			Column:   col,
			Message:  msg,
			Variable: variable,
			Offset:   offset,
		}},
	}
}

// Lines returns the file split into lines (with trailing \r trimmed).
func (fc *FileContext) Lines() []string {
	fc.linesOnce.Do(func() {
		raw := strings.Split(string(fc.Content), "\n")
		for i, l := range raw {
			raw[i] = strings.TrimSuffix(l, "\r")
		}

		fc.lines = raw
	})

	return fc.lines
}

// Tree lazily parses Content with tree-sitter-cfml (CFScript for .cfs, CFML
// for .cfm/.cfc) and returns the CST. Callers must not Close the returned tree;
// FileContext.Close owns its lifetime.
func (fc *FileContext) Tree() *sitter.Tree {
	fc.treeOnce.Do(func() {
		g := language.CFML
		if strings.EqualFold(filepath.Ext(fc.Path), ".cfs") {
			g = language.CFScript
		}

		fc.tree = language.Parse(g, fc.Content, nil)
	})

	return fc.tree
}

// Close releases any native resources held by the FileContext (such as the
// tree-sitter CST).
func (fc *FileContext) Close() {
	if fc.tree != nil {
		fc.tree.Close()
		fc.tree = nil
	}
}
