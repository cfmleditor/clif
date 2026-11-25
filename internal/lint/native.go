package lint

import (
	"cmp"
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/vfs"
	"go.lsp.dev/protocol"
)

// NativeRunner executes native pure-Go lint rules from a Registry, honoring
// per-directory .cflintrc configs, inline suppressions, and minSeverity.
type NativeRunner struct {
	registry    *Registry
	minSeverity string
	minRank     int
	version     string
}

// NewNativeRunner constructs a NativeRunner using reg (or DefaultRegistry when
// nil) and filtering findings below minSeverity.
func NewNativeRunner(reg *Registry, minSeverity, version string) *NativeRunner {
	if reg == nil {
		reg = DefaultRegistry
	}

	minRank := 0

	if minSeverity != "" {
		if r, ok := cflint.MinSeverityRank(minSeverity); ok {
			minRank = r
		}
	}

	if version == "" {
		version = "clif-native"
	}

	return &NativeRunner{
		registry:    reg,
		minSeverity: minSeverity,
		minRank:     minRank,
		version:     version,
	}
}

// Registry returns the underlying rule registry.
func (n *NativeRunner) Registry() *Registry {
	return n.registry
}

// Lint runs native rules over files read from disk.
func (n *NativeRunner) Lint(ctx context.Context, files []string) (*cflint.Run, error) {
	sources := make([]cflint.Source, 0, len(files))

	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		data, err := vfs.ReadRegular(f)
		if err != nil {
			return nil, err
		}

		sources = append(sources, cflint.Source{Path: f, Content: data})
	}

	return n.LintContent(ctx, sources)
}

// LintContent runs native rules over in-memory sources.
func (n *NativeRunner) LintContent(ctx context.Context, sources []cflint.Source) (*cflint.Run, error) {
	run := &cflint.Run{Version: n.version}
	rcCache := &rcDirCache{byDir: make(map[string]*RCConfig)}
	rules := n.registry.Rules()

	for i := range sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		src := sources[i]

		rc, err := rcCache.forFile(src.Path)
		if err != nil {
			return nil, err
		}

		if rc.FileExcluded(src.Path) {
			continue
		}

		run.TotalFiles++
		run.TotalLines += countLines(src.Content)

		issues := n.lintOne(src.Path, src.Content, rc, rules)
		run.Issues = append(run.Issues, issues...)
	}

	sortRunIssues(run.Issues)

	return run, nil
}

// Scan lints a single file from disk and returns LSP diagnostics.
func (n *NativeRunner) Scan(ctx context.Context, filePath string) ([]protocol.Diagnostic, error) {
	run, err := n.Lint(ctx, []string{filePath})
	if err != nil {
		return nil, err
	}

	diags := make([]protocol.Diagnostic, 0, len(run.Issues))

	for i := range run.Issues {
		iss := &run.Issues[i].Issue
		if len(iss.Locations) == 0 {
			continue
		}

		loc := iss.Locations[0]
		line := max(loc.Line-1, 0)
		col := max(loc.Column-1, 0)
		msg := loc.Message

		if msg == "" {
			msg = iss.Message
		}

		diags = append(diags, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: conv.Uint32(line), Character: conv.Uint32(col)},
				End:   protocol.Position{Line: conv.Uint32(line), Character: conv.Uint32(col)},
			},
			Severity: mapLintSeverity(iss.Severity),
			Code:     protocol.String(iss.ID),
			Source:   protocol.NewOptional("lint"),
			Message:  protocol.String(msg),
		})
	}

	return diags, nil
}

func (n *NativeRunner) lintOne(filePath string, content []byte, rc *RCConfig, rules []Rule) []cflint.RunIssue {
	fc := &FileContext{
		Path:    filePath,
		Content: content,
		RC:      rc,
	}
	defer fc.Close()

	suppressions := ParseSuppressions(string(content))

	var out []cflint.RunIssue

	for _, rule := range rules {
		if !rc.RuleEnabled(rule.Code) {
			continue
		}

		sev := rc.RuleSeverity(rule.Code, rule.DefaultSeverity)
		if n.minRank > 0 {
			if rank, ok := cflint.MinSeverityRank(sev); ok && rank > n.minRank {
				continue
			}
		}

		for _, iss := range rule.Check(fc) {
			iss.ID = rule.Code
			if iss.Severity == "" {
				iss.Severity = sev
			} else {
				iss.Severity = rc.RuleSeverity(rule.Code, iss.Severity)
			}

			for i := range iss.Locations {
				loc := iss.Locations[i]
				if suppressions.IsSuppressed(loc.Line, rule.Code) {
					continue
				}

				if loc.File == "" {
					loc.File = filePath
				}

				if loc.FileName == "" {
					loc.FileName = filepath.Base(filePath)
				}

				single := cflint.Issue{
					Severity:  iss.Severity,
					ID:        iss.ID,
					Message:   iss.Message,
					Locations: []cflint.Location{loc},
				}

				raw, err := json.Marshal(single)
				if err != nil {
					continue
				}

				out = append(out, cflint.RunIssue{Issue: single, Raw: raw})
			}
		}
	}

	return out
}

type rcDirCache struct {
	mu    sync.Mutex
	byDir map[string]*RCConfig
}

func (c *rcDirCache) forFile(filePath string) (*RCConfig, error) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(abs)

	c.mu.Lock()
	if rc, ok := c.byDir[dir]; ok {
		c.mu.Unlock()

		return rc, nil
	}
	c.mu.Unlock()

	rc, err := LoadRCForDir(dir)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.byDir[dir] = rc
	c.mu.Unlock()

	return rc, nil
}

func countLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}

	n := 1

	for _, b := range content {
		if b == '\n' {
			n++
		}
	}

	if content[len(content)-1] == '\n' {
		n--
	}

	return n
}

func sortRunIssues(issues []cflint.RunIssue) {
	slices.SortFunc(issues, func(a, b cflint.RunIssue) int {
		la, lb := firstLoc(a.Issue), firstLoc(b.Issue)

		return cmp.Or(
			strings.Compare(la.File, lb.File),
			cmp.Compare(la.Line, lb.Line),
			cmp.Compare(la.Column, lb.Column),
			strings.Compare(a.ID, b.ID),
			strings.Compare(la.Message, lb.Message),
		)
	})
}

func firstLoc(iss cflint.Issue) cflint.Location {
	if len(iss.Locations) == 0 {
		return cflint.Location{}
	}

	return iss.Locations[0]
}

func mapLintSeverity(sev string) protocol.DiagnosticSeverity {
	switch strings.ToUpper(strings.TrimSpace(sev)) {
	case "FATAL", "CRITICAL", "ERROR":
		return protocol.DiagnosticSeverityError
	default:
		return protocol.DiagnosticSeverityWarning
	}
}
