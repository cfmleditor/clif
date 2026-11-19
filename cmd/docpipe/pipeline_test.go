package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	docsources "github.com/cfmleditor/clif/internal/docs/sources"
)

func testPipeline(t *testing.T) *pipeline {
	t.Helper()

	p := newPipeline()
	p.root = t.TempDir()
	p.retryDelay = 0

	return p
}

func TestPinnedFetchStagesLocalGitAndUsesCache(t *testing.T) {
	remote := t.TempDir()
	put(t, remote, "data/en/one.json", `{"name":"one","type":"function"}`)
	gitFixture(t, remote, "init", "-qb", "master")
	gitFixture(t, remote, "add", ".")
	gitFixture(t, remote, "-c", "user.name=Docs test", "-c", "user.email=docs@example.invalid", "commit", "-qm", "docs")
	commit := gitFixture(t, remote, "rev-parse", "HEAD")
	p := testPipeline(t)
	p.cfdocsRepo = remote
	put(t, p.root, lockFile, "cfdocs "+commit+"\n")

	if err := p.fetchCFDocs(t.Context(), false); err != nil {
		t.Fatal(err)
	}

	b, err := readFile(p.path("docs/.sha-cfdocs"))
	if err != nil || strings.TrimSpace(string(b)) != commit {
		t.Fatalf("marker = %q, %v", b, err)
	}

	b, err = readFile(p.path("docs/src/cfdocs/one.json"))
	if err != nil || string(b) != `{"name":"one","type":"function"}` {
		t.Fatalf("pinned source not copied intact: %q, %v", b, err)
	}

	p.command = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("cached fetch invoked a subprocess")

		return nil, nil
	}
	if err := p.fetchCFDocs(t.Context(), false); err != nil {
		t.Fatal(err)
	}
}

func TestRetryIsBoundedAndStopsOnCancellation(t *testing.T) {
	p := testPipeline(t)
	attempts := 0
	failure := errors.New("offline")

	p.command = func(context.Context, string, ...string) ([]byte, error) {
		attempts++

		return nil, failure
	}
	if _, err := p.retry(t.Context(), "git", "ls-remote"); !errors.Is(err, failure) || attempts != 3 {
		t.Fatalf("attempts %d, err %v", attempts, err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	attempts = 0
	if _, err := p.retry(ctx, "git", "ls-remote"); err == nil || attempts > 1 {
		t.Fatalf("cancellation: attempts %d, err %v", attempts, err)
	}
}

func TestSwapRecoversAndRollsBackAssociatedWrite(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs")
	put(t, root, "docs.old/value", "old")

	if err := recoverDir(dir); err != nil {
		t.Fatal(err)
	}

	staged, err := stageDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	put(t, staged, "value", "new")

	failure := errors.New("pin write failed")
	if err := installDir(staged, dir, func() error { return failure }); !errors.Is(err, failure) {
		t.Fatal(err)
	}

	b, err := readFile(filepath.Join(dir, "value"))
	if err != nil || string(b) != "old" {
		t.Fatalf("previous copy lost: %q, %v", b, err)
	}

	if _, err := os.Stat(dir + ".old"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rollback left an old copy: %v", err)
	}
}

type zipEntry struct{ name, body string }

func archiveBytes(t *testing.T, entries []zipEntry) []byte {
	t.Helper()

	var b bytes.Buffer

	z := zip.NewWriter(&b)
	for _, e := range entries {
		w, err := z.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}

		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}

	if err := z.Close(); err != nil {
		t.Fatal(err)
	}

	return b.Bytes()
}

func luceeArchive(t *testing.T, body string) []byte {
	t.Helper()
	inner := archiveBytes(t, []zipEntry{{"one.json", body}})
	middle := archiveBytes(t, []zipEntry{{"lucee-docs-json.zip", string(inner)}})

	return archiveBytes(t, []zipEntry{{"lucee-docs-json-zipped.zip", string(middle)}})
}

func fakeDownload(t *testing.T, archive []byte) func(context.Context, string, ...string) ([]byte, error) {
	t.Helper()

	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "curl" {
			t.Fatalf("unexpected subprocess %s", name)
		}

		for i, arg := range args {
			if arg == "-o" {
				if err := os.WriteFile(args[i+1], archive, 0o600); err != nil {
					return nil, err
				}
			}

			if arg == "-D" {
				if err := os.WriteFile(args[i+1], []byte("HTTP/1.1 302 Found\r\nETag: redirect\r\nHTTP/2 200\r\nETag:\"final\"\r\n"), 0o600); err != nil {
					return nil, err
				}
			}
		}

		return nil, nil
	}
}

func TestLuceeStagesValidatesAndPreservesNotes(t *testing.T) {
	p := testPipeline(t)
	put(t, p.root, "docs/lucee/old.json", `{"name":"old","type":"function"}`)
	put(t, p.root, "docs/lucee/SOURCE", "# provenance\njson-sha256 old\n")
	put(t, p.root, "docs/lucee/notes/README", "keep me")

	p.command = fakeDownload(t, luceeArchive(t, `{"name":"one","type":"function"}`))
	if err := p.fetchLucee(t.Context(), nil); err != nil {
		t.Fatal(err)
	}

	hash, unhashed, invalid, err := docsources.HashDir(p.path("docs/lucee"))
	if err != nil || len(unhashed)+len(invalid) > 0 {
		t.Fatalf("%v, %v, %v", err, unhashed, invalid)
	}

	source, err := readFile(p.path("docs/lucee/SOURCE"))
	if err != nil || !strings.Contains(string(source), "json-sha256 "+hash) || !strings.Contains(string(source), "etag \"final\"") {
		t.Fatalf("source = %q, %v", source, err)
	}

	if b, err := readFile(p.path("docs/lucee/notes/README")); err != nil || string(b) != "keep me" {
		t.Fatalf("note lost: %q, %v", b, err)
	}

	if _, err := os.Stat(p.path("docs/lucee/old.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old JSON remains: %v", err)
	}
}

func TestBrokenLuceeLeavesCommittedCopyAlone(t *testing.T) {
	p := testPipeline(t)
	put(t, p.root, "docs/lucee/old.json", `{"name":"old","type":"function"}`)

	p.command = fakeDownload(t, luceeArchive(t, "{"))
	if err := p.fetchLucee(t.Context(), nil); err == nil {
		t.Fatal("broken Lucee JSON accepted")
	}

	if _, err := readFile(p.path("docs/lucee/old.json")); err != nil {
		t.Fatal("old copy lost:", err)
	}
}

func TestZipPathsAndDuplicatesDoNotEscapeStaging(t *testing.T) {
	for _, entries := range [][]zipEntry{
		{{"../escape.json", `{"name":"escape","type":"function"}`}},
		{{"one.json", "{}"}, {"one.json", "{}"}},
	} {
		root := t.TempDir()

		archive := filepath.Join(root, "input.zip")
		if err := os.WriteFile(archive, archiveBytes(t, entries), 0o600); err != nil {
			t.Fatal(err)
		}

		out := filepath.Join(root, "out")
		if err := os.Mkdir(out, 0o700); err != nil {
			t.Fatal(err)
		}

		if _, err := extractJSON(archive, out); err == nil {
			t.Fatal("invalid archive accepted")
		}

		if _, err := os.Stat(filepath.Join(root, "escape.json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("zip escaped staging: %v", err)
		}
	}
}

func TestFinalResponseOwnsETag(t *testing.T) {
	headers := []byte("HTTP/1.1 302 Found\r\nETag: redirect\r\nHTTP/2 200\r\n")
	if got := responseETag(headers); got != "" {
		t.Fatalf("redirect ETag retained: %s", got)
	}

	if got := responseETag(append(headers, []byte("eTaG:\"final\"\r\n")...)); got != "\"final\"" {
		t.Fatal(got)
	}
}

func TestNoShellSubprocessCanOwnDocsWrites(t *testing.T) {
	if _, err := commandOutput(t.Context(), "bash", "-c", "true"); err == nil {
		t.Fatal("driver accepted a shell subprocess")
	}
}

func TestBumpWritesPinOnlyAfterBothSourcesStage(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "broken-lucee"}[broken], func(t *testing.T) {
			p := testPipeline(t)
			put(t, p.root, lockFile, "cfdocs "+fixturePin+"\n")
			put(t, p.root, "docs/.sha-cfdocs", fixturePin+"\n")
			put(t, p.root, "docs/src/cfdocs/old.json", `{"name":"old","type":"function"}`)

			oldLucee := `{"name":"oldLucee","type":"function"}`
			put(t, p.root, "docs/lucee/old.json", oldLucee)

			next := strings.Repeat("b", 40)

			body := `{"name":"newLucee","type":"function"}`
			if broken {
				body = "{"
			}

			download := fakeDownload(t, luceeArchive(t, body))
			p.command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				if name == "curl" {
					pin, err := p.readPin("cfdocs")
					if err != nil || pin != fixturePin {
						t.Fatalf("pin changed before Lucee staged: %s, %v", pin, err)
					}

					return download(ctx, name, args...)
				}

				if args[0] == "ls-remote" {
					return []byte(next + "\trefs/heads/master\n"), nil
				}

				if len(args) > 2 && args[2] == "checkout" {
					put(t, args[1], "data/en/new.json", `{"name":"newCFDocs","type":"function"}`)
				}

				return nil, nil
			}
			err := p.bump(t.Context())

			pin, pinErr := p.readPin("cfdocs")
			if broken {
				if err == nil || pinErr != nil || pin != fixturePin {
					t.Fatalf("failed bump changed pin: %s, %v, %v", pin, pinErr, err)
				}

				b, readErr := readFile(p.path("docs/lucee/old.json"))
				if readErr != nil || string(b) != oldLucee {
					t.Fatalf("failed bump replaced Lucee: %q, %v", b, readErr)
				}
			} else if err != nil || pinErr != nil || pin != next {
				t.Fatalf("bump failed: pin %s, %v, %v", pin, pinErr, err)
			}
		})
	}
}

func TestFailedSecondDirectoryRenameRestoresOldCopy(t *testing.T) {
	root := t.TempDir()
	put(t, root, "source/value", "original")

	dir := filepath.Join(root, "source")
	if err := installDir(filepath.Join(root, "missing"), dir, nil); err == nil {
		t.Fatal("missing staged copy accepted")
	}

	b, err := readFile(filepath.Join(dir, "value"))
	if err != nil || string(b) != "original" {
		t.Fatalf("original not recovered: %q, %v", b, err)
	}
}
