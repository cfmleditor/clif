package server

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/config"
)

// TestExportCFLintLintsTheFoldersOpen lays out a workspace like ~/tassdev: a
// root config listing the repos, and a tassweb config listing its siblings for
// resolution. exportCFLint linted every workspace folder, so a tassweb window
// linted prs too and kept only tassweb's findings.
func TestExportCFLintLintsTheFoldersOpen(t *testing.T) {
	ws, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	tassweb := filepath.Join(ws, "tassweb")
	prs := filepath.Join(ws, "prs")

	for _, d := range []string{tassweb, prs} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	write(t, filepath.Join(ws, ".cfmleditor.json"), `{"workspaceName":"tass","workspacePaths":["tassweb","prs"]}`)
	write(t, filepath.Join(tassweb, ".cfmleditor.json"), `{"workspaceName":"tass","workspacePaths":["../prs","../tassweb"]}`)
	write(t, filepath.Join(prs, "a.cfm"), "")

	session := func(open string, folders ...string) *Server {
		srv := newTestServer()
		srv.addWorkspaceRoot(open)
		srv.WorkspaceFolders = folders

		return srv
	}

	t.Run("a project whose config lists its siblings", func(t *testing.T) {
		srv := session(tassweb, prs, tassweb)

		roots, targets, _, err := srv.exportScope(config.GenerateCFLint)
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{tassweb}; !slices.Equal(roots, want) {
			t.Errorf("linted %v, want %v", roots, want)
		}

		if want := []string{filepath.Join(tassweb, ".clif-cflint.txt")}; !slices.Equal(targets, want) {
			t.Errorf("writes %v, want %v", targets, want)
		}

		// The unresolved report still reads every folder: resolution needs them.
		if roots, _, _, _ := srv.exportScope(config.GenerateUnresolved); !slices.Equal(roots, []string{prs, tassweb}) {
			t.Errorf("unresolved scans %v, want every workspace folder", roots)
		}
	})

	t.Run("the workspace root", func(t *testing.T) {
		roots, targets, _, err := session(ws, tassweb, prs).exportScope(config.GenerateCFLint)
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{tassweb, prs}; !slices.Equal(roots, want) {
			t.Errorf("linted %v, want %v", roots, want)
		}

		if want := []string{filepath.Join(ws, ".clif-cflint.txt")}; !slices.Equal(targets, want) {
			t.Errorf("writes %v, want %v", targets, want)
		}
	})

	t.Run("a project with no config of its own", func(t *testing.T) {
		// prs is governed by the workspace config, whose report covers tassweb
		// too: rewriting it from a prs lint would delete tassweb's entries.
		_, _, _, err := session(prs, tassweb, prs).exportScope(config.GenerateCFLint)
		if err == nil || !strings.Contains(err.Error(), filepath.Join(ws, ".clif-cflint.txt")) {
			t.Errorf("err = %v, want a refusal naming the workspace report", err)
		}
	})
}

// TestExportRefusesAReportOutsideTheWorkspace is a cloned repository whose
// .clif.json names a report outside it, or which commits a symlink at the
// default report's name. The export wrote either with os.WriteFile, so the
// "Export unresolved calls report" code action overwrote the file named.
func TestExportRefusesAReportOutsideTheWorkspace(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	repo := filepath.Join(base, "repo")
	victim := filepath.Join(base, "victim")

	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}

	write(t, victim, "keep")
	write(t, filepath.Join(repo, ".clif.json"), `{"workspaceName":"repo"}`)

	session := func(issues ...config.KnownIssues) *Server {
		srv := newTestServer()
		srv.addWorkspaceRoot(repo)
		// A configured folder does not widen where a report may go: the
		// config naming the report would otherwise name its own licence.
		srv.WorkspaceFolders = []string{base}
		srv.KnownIssues = config.ResolveKnownIssues(&config.KnownIssuesConfig{Files: issues}, repo)

		return srv
	}

	for _, kind := range []string{config.GenerateUnresolved, config.GenerateCFLint} {
		t.Run(kind+" named outside", func(t *testing.T) {
			srv := session(config.KnownIssues{File: "../victim", Generate: kind})

			if _, _, _, err := srv.exportScope(kind); err == nil {
				t.Fatal("an export to ../victim was allowed")
			}
		})
	}

	t.Run("a symlink at the default name", func(t *testing.T) {
		link := filepath.Join(repo, ".clif-unresolved.txt")
		if err := os.Symlink(victim, link); err != nil {
			t.Fatal(err)
		}

		defer func() { _ = os.Remove(link) }()

		if _, _, _, err := session().exportScope(config.GenerateUnresolved); err == nil {
			t.Fatal("an export through a symlink out of the workspace was allowed")
		}
	})

	t.Run("a symlink inside a linted folder", func(t *testing.T) {
		link := filepath.Join(repo, ".clif-cflint.txt")
		if err := os.Symlink(victim, link); err != nil {
			t.Fatal(err)
		}

		defer func() { _ = os.Remove(link) }()

		if _, _, _, err := session().exportScope(config.GenerateCFLint); err == nil {
			t.Fatal("a CFLint export through a symlink out of the workspace was allowed")
		}
	})

	t.Run("the default report", func(t *testing.T) {
		_, targets, _, err := session().exportScope(config.GenerateUnresolved)
		if err != nil {
			t.Fatal(err)
		}

		if want := []string{filepath.Join(repo, ".clif-unresolved.txt")}; !slices.Equal(targets, want) {
			t.Errorf("writes %v, want %v", targets, want)
		}
	})
}
