package lint

import (
	"context"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"go.lsp.dev/protocol"
)

// HybridRunner runs NativeRunner for every rule implemented natively and
// delegates remaining .cflintrc rules to a fallback Engine (typically
// *cflint.Runner). When every rule enabled by a file's .cflintrc is already
// implemented natively, the fallback is never invoked.
type HybridRunner struct {
	native   *NativeRunner
	fallback Engine
}

// NewHybridRunner constructs a HybridRunner. fallback may be nil, in which
// case only native rules run.
func NewHybridRunner(native *NativeRunner, fallback Engine) *HybridRunner {
	return &HybridRunner{native: native, fallback: fallback}
}

// Lint runs native rules and, if needed, merges fallback findings for rules
// not yet covered natively.
func (h *HybridRunner) Lint(ctx context.Context, files []string) (*cflint.Run, error) {
	nativeRun, err := h.native.Lint(ctx, files)
	if err != nil {
		return nil, err
	}

	if h.fallback == nil || !h.needsFallback(files) {
		return nativeRun, nil
	}

	fbRun, err := h.fallback.Lint(ctx, files)
	if err != nil {
		return nil, err
	}

	return h.merge(nativeRun, fbRun), nil
}

// LintContent runs native rules over in-memory sources and merges fallback
// findings when needed.
func (h *HybridRunner) LintContent(ctx context.Context, sources []cflint.Source) (*cflint.Run, error) {
	nativeRun, err := h.native.LintContent(ctx, sources)
	if err != nil {
		return nil, err
	}

	files := make([]string, len(sources))
	for i := range sources {
		files[i] = sources[i].Path
	}

	if h.fallback == nil || !h.needsFallback(files) {
		return nativeRun, nil
	}

	fbRun, err := h.fallback.LintContent(ctx, sources)
	if err != nil {
		return nil, err
	}

	return h.merge(nativeRun, fbRun), nil
}

// Scan lints a single file and merges native + fallback LSP diagnostics.
func (h *HybridRunner) Scan(ctx context.Context, filePath string) ([]protocol.Diagnostic, error) {
	nativeDiags, err := h.native.Scan(ctx, filePath)
	if err != nil {
		return nil, err
	}

	if h.fallback == nil || !h.needsFallback([]string{filePath}) {
		return nativeDiags, nil
	}

	fbDiags, err := h.fallback.Scan(ctx, filePath)
	if err != nil {
		return nil, err
	}

	reg := h.native.Registry()
	out := nativeDiags

	for i := range fbDiags {
		d := fbDiags[i]
		code, _ := d.Code.(protocol.String)

		if reg.Has(string(code)) {
			continue
		}

		out = append(out, d)
	}

	return out, nil
}

// needsFallback reports whether any file's .cflintrc enables a rule not
// registered in the native runner. When a .cflintrc has no explicit includes
// list (all default rules active), fallback is needed until full parity.
func (h *HybridRunner) needsFallback(files []string) bool {
	reg := h.native.Registry()
	cache := &rcDirCache{byDir: make(map[string]*RCConfig)}

	for _, f := range files {
		rc, err := cache.forFile(f)
		if err != nil || !rc.HasIncludes {
			return true
		}

		for code := range rc.Includes {
			if rc.Excludes[code] {
				continue
			}

			if !reg.Has(code) {
				return true
			}
		}
	}

	return false
}

func (h *HybridRunner) merge(nativeRun, fbRun *cflint.Run) *cflint.Run {
	reg := h.native.Registry()
	out := &cflint.Run{
		Version:    nativeRun.Version,
		TotalFiles: max(nativeRun.TotalFiles, fbRun.TotalFiles),
		TotalLines: max(nativeRun.TotalLines, fbRun.TotalLines),
		Issues:     append([]cflint.RunIssue(nil), nativeRun.Issues...),
	}

	for i := range fbRun.Issues {
		iss := fbRun.Issues[i]
		if reg.Has(strings.ToUpper(iss.ID)) {
			continue
		}

		out.Issues = append(out.Issues, iss)
	}

	sortRunIssues(out.Issues)

	return out
}
