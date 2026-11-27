package lint

import (
	"context"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/config"
)

// NewEngine constructs the requested lint Engine ("cflint", "native", or
// "hybrid"). Empty engineName defaults to "cflint" during migration.
func NewEngine(ctx context.Context, engineName, minSeverity, version string) (Engine, error) {
	switch strings.ToLower(strings.TrimSpace(engineName)) {
	case config.LintEngineNative:
		return NewNativeRunner(DefaultRegistry, minSeverity, version), nil
	case config.LintEngineHybrid:
		native := NewNativeRunner(DefaultRegistry, minSeverity, version)

		// When CFLint is unavailable in hybrid mode, still allow native rules
		// (which covers 100% of a native-only .cflintrc).
		fb, _ := cflint.NewRunner(ctx, minSeverity)

		return NewHybridRunner(native, fb), nil
	default:
		return cflint.NewRunner(ctx, minSeverity)
	}
}
