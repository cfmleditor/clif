package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfmleditor/clif/internal/vfs"
)

func TestCheckRemainingReadErrors(t *testing.T) {
	for _, name := range []string{"format", "unresolved"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "large.cfm")

			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}

			if err := f.Truncate(vfs.MaxFileSize + 1); err != nil {
				t.Fatal(err)
			}

			if err := f.Close(); err != nil {
				t.Fatal(err)
			}

			reportPath := filepath.Join(dir, "unresolved.txt")
			if err := os.WriteFile(reportPath, []byte("existing findings\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			out, code := runClif(t, dir, "check", "--checks", name, "--format", "json", dir)
			if code != exitCheckError {
				t.Fatalf("exit %d, want %d for unreadable target; output:\n%s", code, exitCheckError, out)
			}

			if name == "unresolved" {
				cfg := `{"knownIssues":{"files":[{"file":"unresolved.txt","generate":"unresolved"}]}}`
				if err := os.WriteFile(filepath.Join(dir, ".clif.json"), []byte(cfg), 0o600); err != nil {
					t.Fatal(err)
				}

				out, code := runClif(t, dir, "unresolved", "--write", dir)
				if code != exitCheckError {
					t.Errorf("unresolved --write: exit %d, want %d; output:\n%s", code, exitCheckError, out)
				}

				data, err := os.ReadFile(reportPath)
				if err != nil || string(data) != "existing findings\n" {
					t.Errorf("report changed after an incomplete scan: %q, %v", data, err)
				}

				if _, err := mcpUnresolved(dir, []string{dir}, false, 10); err == nil {
					t.Error("MCP returned a successful result for an unreadable target")
				}
			}
		})
	}
}

func TestCheckRemainingWalkErrors(t *testing.T) {
	for _, name := range []string{"format", "unresolved", "cflint"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()

			blocked := filepath.Join(dir, "blocked")
			if err := os.Mkdir(blocked, 0o700); err != nil {
				t.Fatal(err)
			}

			file := filepath.Join(blocked, "broken.cfm")
			if err := os.WriteFile(file, []byte("<cfif x>\n<cfset x = >\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			if err := os.Chmod(blocked, 0o000); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chmod(blocked, 0o700) }()

			if _, err := os.ReadDir(blocked); err == nil {
				t.Skip("cannot exercise directory permission failures with this account")
			}

			env := fakeCFLint(t, file)

			out, code := runClifEnv(t, dir, env, "check", "--checks", name, "--format", "json", dir)
			if code != exitCheckError {
				t.Fatalf("exit %d, want %d for inaccessible directory; output:\n%s", code, exitCheckError, out)
			}
		})
	}
}

func TestCheckUnreadableFile(t *testing.T) {
	for _, name := range []string{"format", "unresolved"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()

			file := filepath.Join(dir, "blocked.cfm")
			if err := os.WriteFile(file, []byte("<cfset result = missingService.run()>\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			if err := os.Chmod(file, 0o000); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chmod(file, 0o600) }()

			if _, err := os.ReadFile(file); err == nil {
				t.Skip("this account can read files without permission")
			}

			out, code := runClif(t, dir, "check", "--checks", name, "--format", "json", dir)
			if code != exitCheckError {
				t.Fatalf("exit %d, want %d for permission-denied input; output:\n%s", code, exitCheckError, out)
			}
		})
	}
}

// A grammar refusal is deliberately left to the scan check, unlike an I/O failure.
func TestCheckFormatGrammarRefusalStillSkips(t *testing.T) {
	dir, _ := checkProject(t, "", "<cfif x>\n<cfset y = >\n")

	out, code := runClif(t, dir, "check", "--checks", "format", "--format", "json", dir)
	if code != 0 || !strings.Contains(out, `"skipped": 1`) {
		t.Fatalf("exit %d, want a skipped refusal; output:\n%s", code, out)
	}
}
