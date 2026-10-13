package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/confine"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/unresolved"
)

// report is one generated known-issues file's content, before it is written.
type report struct {
	path    string
	content string
	entries int
}

// handleExport answers clif.exportUnresolved and clif.exportCFLint:
// it scans what exportScope says from disk and writes the findings to the
// knownIssues files marked generate kind, or to that kind's default file
// beside .clif.json (see config.GenerateTargets), then reloads the ones
// knownIssues lists so their entries are published straight away.
//
// Like generateCodeMap, it returns the paths at once and works in the
// background, reporting through window/showMessage.
func (s *Server) handleExport(kind string) (any, error) {
	roots, targets, writeRoots, err := s.exportScope(kind)
	if err != nil {
		// Zed ignores a command's result, error or not, so say it as well.
		s.notifyError(context.Background(), "The "+kind+" report was not written: "+err.Error())

		return nil, err
	}

	if _, busy := s.exporting.LoadOrStore(kind, true); busy {
		return nil, fmt.Errorf("a %s export is already running", kind)
	}

	s.safeGo("export:"+kind, func() {
		defer s.exporting.Delete(kind)

		// context.Background: this goroutine outlives the handler, whose ctx
		// is pooled and reset on return.
		ctx := context.Background()
		started := time.Now()

		s.notifyInfo(ctx, "Scanning the workspace for the "+kind+" report…")

		files := collectWorkspaceCFMLFiles(s, roots)

		var (
			reports []report
			left    int
			err     error
		)

		switch kind {
		case config.GenerateCFLint:
			reports, left, err = s.cflintReports(ctx, files, targets)
		default:
			reports, left = s.unresolvedReports(files, targets)
		}

		if err != nil {
			s.notifyError(ctx, "The "+kind+" report failed: "+err.Error())

			return
		}

		var written, unlisted []string

		for _, r := range reports {
			// Not writeReport, which refuses any symlink at the name: the
			// known-issues file is named in the config, and a project may keep
			// it behind a symlink on purpose. confine follows such a link only
			// while it stays inside writeRoots, so a cloned repository's link to
			// ~/.zshrc is refused at the moment of writing as well as by
			// exportScope.
			if err := confine.WriteFile(r.path, []byte(r.content), reportPerm, writeRoots); err != nil {
				s.notifyError(ctx, "Could not write "+r.path+": "+err.Error())

				return
			}

			written = append(written, fmt.Sprintf("%s (%d)", r.path, r.entries))

			// The file watcher would reload it too, but not every client
			// watches, and the export should show its result either way.
			if s.isKnownIssuesFile(r.path) {
				s.loadKnownIssuesFile(ctx, r.path)
			} else {
				unlisted = append(unlisted, filepath.Base(r.path))
			}
		}

		msg := fmt.Sprintf("The %s report: %s in %s.", kind, strings.Join(written, ", "), time.Since(started).Round(time.Millisecond))
		if left > 0 {
			msg += fmt.Sprintf(" %d findings under no report's directory were left out.", left)
		}

		if len(unlisted) > 0 {
			msg += fmt.Sprintf(` List %s under knownIssues in .clif.json to see the entries as diagnostics.`, strings.Join(unlisted, ", "))
		}

		s.log.Info("known issues exported", cflog.String("kind", kind), cflog.Strings("files", written), cflog.Int("leftOut", left))
		s.notifyInfo(ctx, msg)
	})

	return map[string]any{"paths": targets, "started": true}, nil
}

// exportScope is what an export scans and the files it writes. The unresolved
// report scans every workspace folder, since resolving a call reads them all.
// The CFLint report scans only the folders open in the editor (cflint.ScanRoots,
// as the cflint CLI does with its arguments), and refuses a report file whose
// directory holds more than that, which rewriting would empty of everything
// else's entries. It used to lint every workspace folder: a tassweb window,
// whose config lists all twelve sibling repos for resolution, linted all
// twelve and kept only tassweb's findings.
//
// writeRoots is where a report may be written, after symlinks: the folders open
// in the editor and the config's own directory. Not the configured
// workspacePaths, since the config that names the report would then be naming
// where it may go, and a cloned repository's .clif.json could list "/" there
// and "../../.zshrc" as its report. Every target is checked against them here,
// so the export refuses before scanning rather than after.
func (s *Server) exportScope(kind string) (roots, targets, writeRoots []string, err error) {
	roots = s.searchRoots()
	if len(roots) == 0 {
		return nil, nil, nil, errors.New("no workspace root to scan; open a folder first")
	}

	configDir := roots[0]
	if p, _ := s.governingConfig(); p != "" {
		configDir = filepath.Dir(p)
	}

	open := s.editorRoots()
	writeRoots = confine.Roots(append(slices.Clone(open), configDir)...)

	targets = config.GenerateTargets(s.KnownIssues, kind, configDir)
	if kind != config.GenerateCFLint {
		if err := checkTargets(targets, writeRoots); err != nil {
			return nil, nil, nil, err
		}

		return roots, targets, writeRoots, nil
	}

	if len(open) == 0 {
		open = []string{configDir}
	}

	targets, err = cflint.WriteTargets(targets, open)
	if err != nil {
		return nil, nil, nil, fmt.Errorf(`%w. Open that folder, or list a report file under %s in knownIssues with "generate": "cflint"`,
			err, strings.Join(open, ", "))
	}

	if err := checkTargets(targets, writeRoots); err != nil {
		return nil, nil, nil, err
	}

	roots = cflint.ScanRoots(open, configDir, s.WorkspaceFolders)

	return roots, targets, writeRoots, nil
}

// checkTargets refuses the export when any report would land, after symlinks,
// outside writeRoots.
func checkTargets(targets, writeRoots []string) error {
	for _, t := range targets {
		if _, err := confine.Check(t, writeRoots); err != nil {
			return err
		}
	}

	return nil
}

// unresolvedReports runs the unresolved scan the CLI runs, with this
// session's settings, and splits it across targets.
func (s *Server) unresolvedReports(files, targets []string) ([]report, int) {
	resolver := s.getResolver()
	opt := unresolved.Options{
		CallerInferenceLimits:    s.CallerInferenceLimits,
		Mappings:                 s.Mappings,
		ExpressionMappings:       s.ExpressionMappings,
		ServicePropertyResolvers: s.ServicePropertyResolvers,
		PropertyResolvers:        s.cfPropertyResolvers(),
		BeanPaths:                s.BeanPaths,
		WorkspaceFolders:         s.WorkspaceFolders,
		InterpolateAll:           !s.Features.OutputContextInterpolation,
	}

	opt.Resolvers = resolver.Resolvers
	opt.StartupFiles = resolver.StartupFiles
	opt.ImplicitExtends = resolver.ImplicitExtends
	opt.HelperScope = resolver.HelperScope
	opt.Stubs = resolver.Stubs

	rep := unresolved.Scan(s.FS, files, nil, &opt)
	byTarget, rest := unresolved.SplitByTarget(rep.Calls, targets)

	out := make([]report, 0, len(targets))

	for _, t := range targets {
		var b strings.Builder

		skipped := unresolved.WriteKnownIssues(&b, byTarget[t], filepath.Dir(t), false, unresolved.RegenerateHint, s.Version)
		out = append(out, report{path: t, content: b.String(), entries: len(byTarget[t]) - skipped})
	}

	return out, len(rest)
}

// cflintReports runs CFLint over files and splits its findings across
// targets. It uses the session's runner, or starts one when linting on save
// is off: a project report is still wanted where per-save linting is not.
func (s *Server) cflintReports(ctx context.Context, files, targets []string) ([]report, int, error) {
	s.mu.RLock()
	runner := s.linter
	s.mu.RUnlock()

	if runner == nil {
		r, err := cflint.NewRunner(ctx, s.LintMinSeverity)
		if err != nil {
			return nil, 0, err
		}

		runner = r
	}

	found, err := runner.ScanFiles(ctx, files)
	if err != nil {
		return nil, 0, err
	}

	reports, left := cflint.Reports(found, targets, s.Version)

	out := make([]report, 0, len(reports))
	for _, r := range reports {
		out = append(out, report{path: r.Path, content: r.Content, entries: r.Entries})
	}

	return out, left, nil
}
