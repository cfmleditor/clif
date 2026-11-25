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

	linesOnce sync.Once
	lines     []string

	treeOnce sync.Once
	tree     *sitter.Tree
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
