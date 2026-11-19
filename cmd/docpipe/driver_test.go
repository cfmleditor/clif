package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	docsources "github.com/cfmleditor/clif/internal/docs/sources"
)

const fixturePin = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

// Run the real entry point in a child so refusals and signals exercise cleanup.
func TestDocpipeProcess(t *testing.T) {
	encoded := os.Getenv("DOCPIPE_TEST_ARGS")
	if encoded == "" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(encoded), &args); err != nil {
		t.Fatal(err)
	}

	if ready := os.Getenv("DOCPIPE_TEST_BLOCK"); ready != "" {
		commandRunner = func(ctx context.Context, _ string, _ ...string) ([]byte, error) {
			if err := os.WriteFile(ready, []byte("ready"), 0o600); err != nil {
				return nil, err
			}

			<-ctx.Done()

			return nil, ctx.Err()
		}
	}

	os.Args = append([]string{"docpipe"}, args...)

	main()
}

func driverCommand(t *testing.T, root string, extra []string, args ...string) *exec.Cmd {
	t.Helper()

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.CommandContext(t.Context(), exe, "-test.run=^TestDocpipeProcess$")
	cmd.Dir = root

	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "DOCS_") && !strings.HasPrefix(e, "DOCPIPE_TEST_") {
			cmd.Env = append(cmd.Env, e)
		}
	}

	cmd.Env = append(cmd.Env, "DOCPIPE_TEST_ARGS="+string(encoded))
	cmd.Env = append(cmd.Env, extra...)

	return cmd
}

func runDriver(t *testing.T, root string, env []string, args ...string) (string, int) {
	t.Helper()

	out, err := driverCommand(t, root, env, args...).CombinedOutput()
	if err == nil {
		return string(out), 0
	}

	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		return string(out), exit.ExitCode()
	}

	t.Fatal(err)

	return "", -1
}

func put(t *testing.T, root, name, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func gitFixture(t *testing.T, root string, args ...string) string {
	t.Helper()

	out, err := exec.CommandContext(t.Context(), "git", append([]string{"-C", root}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}

	return strings.TrimSpace(string(out))
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put(t, root, "go.mod", "module example.invalid/docs\n")
	put(t, root, "docs/sources.lock", "cfdocs "+fixturePin+"\n")
	put(t, root, "docs/.sha-cfdocs", fixturePin+"\n")
	put(t, root, "docs/data/order", "cfdocs docs/src/cfdocs\nretained docs/retained\nlucee docs/lucee\n")
	put(t, root, "docs/src/cfdocs/cfonly.json", `{"name":"cfOnly","type":"function","syntax":"cfOnly()"}`)
	put(t, root, "docs/src/cfdocs/shared.json", `{"name":"shared","type":"function","description":"cfdocs","syntax":"shared()"}`)
	put(t, root, "docs/lucee/luceeonly.json", `{"name":"luceeOnly","type":"function","syntax":"luceeOnly()"}`)
	put(t, root, "docs/lucee/shared.json", `{"name":"shared","type":"function","description":"lucee","syntax":"shared(x)"}`)

	if err := os.MkdirAll(filepath.Join(root, "docs", "retained"), 0o750); err != nil {
		t.Fatal(err)
	}

	put(t, root, outFile, "original generated data\n")
	put(t, root, sourcesFile, docsources.OrderPrefix+"cfdocs retained lucee\nfunction cfonly\tcfdocs\nfunction luceeonly\tlucee\nfunction shared\tlucee\n")
	put(t, root, "docs/allowed-removals", "# Explicit changes only\n")
	gitFixture(t, root, "init", "-q")
	gitFixture(t, root, "add", ".")
	gitFixture(t, root, "-c", "user.name=Docs test", "-c", "user.email=docs@example.invalid", "commit", "-qm", "baseline")

	return root
}

func TestGenerateUsesPinnedSourcesAndSourcePrecedence(t *testing.T) {
	root := fixture(t)
	if out, code := runDriver(t, root, nil, "generate"); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}

	b, err := readFile(filepath.Join(root, outFile))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(b, []byte(`Description: "lucee"`)) || bytes.Contains(b, []byte(`Description: "cfdocs"`)) {
		t.Fatalf("wrong source precedence: %s", b)
	}

	first := append([]byte{}, b...)

	if out, code := runDriver(t, root, nil, "generate"); code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}

	b, err = readFile(filepath.Join(root, outFile))
	if err != nil || !bytes.Equal(first, b) {
		t.Fatalf("generation changed on repeat: %v", err)
	}
}

func TestGeneratorRefusalsPreserveFilesAndReleaseLock(t *testing.T) {
	for _, problem := range []string{"marker", "retained", "removal", "fallback", "missing-source", "no-baseline"} {
		t.Run(problem, func(t *testing.T) {
			root := fixture(t)

			before, err := readFile(filepath.Join(root, sourcesFile))
			if err != nil {
				t.Fatal(err)
			}

			switch problem {
			case "marker":
				put(t, root, "docs/.sha-cfdocs", strings.Repeat("b", 40))
			case "retained":
				put(t, root, "docs/retained/broken.json", "{")
			case "removal":
				if err := os.Remove(filepath.Join(root, "docs", "src", "cfdocs", "cfonly.json")); err != nil {
					t.Fatal(err)
				}
			case "fallback":
				if err := os.Remove(filepath.Join(root, "docs", "lucee", "shared.json")); err != nil {
					t.Fatal(err)
				}
			case "missing-source":
				if err := os.RemoveAll(filepath.Join(root, "docs", "lucee")); err != nil {
					t.Fatal(err)
				}
			case "no-baseline":
				gitFixture(t, root, "rm", "--cached", sourcesFile)
				gitFixture(t, root, "-c", "user.name=Docs test", "-c", "user.email=docs@example.invalid", "commit", "-qm", "remove baseline")
			}

			if out, code := runDriver(t, root, nil, "generate-only"); code != 1 || !strings.Contains(out, "DOCS NOT REGENERATED") {
				t.Fatalf("exit %d: %s", code, out)
			}

			after, err := readFile(filepath.Join(root, sourcesFile))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("refusal changed sources: %v", err)
			}

			generated, err := readFile(filepath.Join(root, outFile))
			if err != nil || string(generated) != "original generated data\n" {
				t.Fatalf("refusal changed generated data: %v", err)
			}

			if out, code := runDriver(t, root, nil, "pin", "set", "cfdocs", fixturePin); code != 0 {
				t.Fatalf("lock not released: exit %d: %s", code, out)
			}
		})
	}
}

func TestAllowedRemovalNeedsAnAddedLine(t *testing.T) {
	root := fixture(t)
	if err := os.Remove(filepath.Join(root, "docs", "src", "cfdocs", "cfonly.json")); err != nil {
		t.Fatal(err)
	}

	put(t, root, "docs/allowed-removals", "function cfOnly # intentionally removed\n")

	env := []string{"DOCS_ALLOWED_REMOVALS_FILE=docs/allowed-removals"}
	if out, code := runDriver(t, root, env, "generate-only"); code != 0 {
		t.Fatalf("added removal: exit %d: %s", code, out)
	}
	// Once the line is at the baseline it must not waive the same loss again.
	gitFixture(t, root, "add", "docs/allowed-removals")
	gitFixture(t, root, "-c", "user.name=Docs test", "-c", "user.email=docs@example.invalid", "commit", "-qm", "acceptance already present")

	if out, code := runDriver(t, root, env, "generate-only"); code != 1 || !strings.Contains(out, "left over") {
		t.Fatalf("leftover acceptance: exit %d: %s", code, out)
	}
}

func TestKilledDriverCannotLeaveAnUnlockedWriter(t *testing.T) {
	root := fixture(t)
	if err := os.Remove(filepath.Join(root, "docs", ".sha-cfdocs")); err != nil {
		t.Fatal(err)
	}

	ready := filepath.Join(root, "ready")
	cmd := driverCommand(t, root, []string{"DOCPIPE_TEST_BLOCK=" + ready}, "docs")

	var output bytes.Buffer

	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	deadline := time.Now().Add(5 * time.Second)

	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}

		if time.Now().After(deadline) {
			t.Fatal("driver did not reach its blocked fetch")
		}

		time.Sleep(10 * time.Millisecond)
	}

	if out, code := runDriver(t, root, nil, "pin", "set", "cfdocs", fixturePin); code != 1 || !strings.Contains(out, "another docs run") {
		t.Fatalf("second writer was not excluded: exit %d: %s", code, out)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}

	_ = cmd.Wait()

	if out, code := runDriver(t, root, nil, "pin", "set", "cfdocs", fixturePin); code != 0 {
		t.Fatalf("killed holder kept the lock: exit %d: %s", code, out)
	}

	b, err := readFile(filepath.Join(root, outFile))
	if err != nil || string(b) != "original generated data\n" {
		t.Fatalf("killed fetch changed generated docs: %v", err)
	}
}
