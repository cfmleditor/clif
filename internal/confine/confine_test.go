package confine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// layout is a workspace and a directory outside it, the outside one holding a
// file a planted link would overwrite.
func layout(t *testing.T) (ws, outside, victim string) {
	t.Helper()

	base := t.TempDir()
	ws = filepath.Join(base, "ws")
	outside = filepath.Join(base, "outside")

	for _, d := range []string{ws, outside} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}

	victim = filepath.Join(outside, "victim")
	if err := os.WriteFile(victim, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	return ws, outside, victim
}

func assertUntouched(t *testing.T, victim string) {
	t.Helper()

	got, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "keep" {
		t.Fatalf("the file outside the workspace was overwritten: %q", got)
	}
}

func TestWriteFileRefusesALinkOutOfTheRoots(t *testing.T) {
	ws, _, victim := layout(t)

	link := filepath.Join(ws, ".clif-unresolved.txt")
	if err := os.Symlink(victim, link); err != nil {
		t.Fatal(err)
	}

	err := WriteFile(link, []byte("report"), 0o600, Roots(ws))
	if !errors.Is(err, ErrOutside) {
		t.Fatalf("want a refusal, got %v", err)
	}

	assertUntouched(t, victim)
}

func TestWriteFileRefusesADanglingLinkOutOfTheRoots(t *testing.T) {
	ws, outside, _ := layout(t)

	target := filepath.Join(outside, "created")
	link := filepath.Join(ws, "report.txt")

	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(link, []byte("report"), 0o600, Roots(ws)); !errors.Is(err, ErrOutside) {
		t.Fatalf("want a refusal, got %v", err)
	}

	if _, err := os.Stat(target); err == nil {
		t.Fatal("the write created a file outside the workspace")
	}
}

func TestWriteFileRefusesASymlinkedDirectoryOutOfTheRoots(t *testing.T) {
	ws, outside, victim := layout(t)

	if err := os.Symlink(outside, filepath.Join(ws, "reports")); err != nil {
		t.Fatal(err)
	}

	err := WriteFile(filepath.Join(ws, "reports", "victim"), []byte("report"), 0o600, Roots(ws))
	if !errors.Is(err, ErrOutside) {
		t.Fatalf("want a refusal, got %v", err)
	}

	assertUntouched(t, victim)
}

func TestWriteFileRefusesDotDot(t *testing.T) {
	ws, _, victim := layout(t)

	err := WriteFile(filepath.Join(ws, "..", "outside", "victim"), []byte("report"), 0o600, Roots(ws))
	if !errors.Is(err, ErrOutside) {
		t.Fatalf("want a refusal, got %v", err)
	}

	assertUntouched(t, victim)
}

func TestWriteFileFollowsALinkThatStaysInside(t *testing.T) {
	ws, _, _ := layout(t)

	kept := filepath.Join(ws, "kept", "report.txt")
	if err := os.MkdirAll(filepath.Dir(kept), 0o750); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(ws, "report.txt")
	if err := os.Symlink(filepath.Join("kept", "report.txt"), link); err != nil {
		t.Fatal(err)
	}

	if err := WriteFile(link, []byte("report"), 0o600, Roots(ws)); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(kept)
	if err != nil || string(got) != "report" {
		t.Fatalf("the link's target was not written: %q, %v", got, err)
	}

	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("the link was replaced rather than followed")
	}
}

func TestMkdirAllRefusesASymlinkedDirectoryOutOfTheRoots(t *testing.T) {
	ws, outside, _ := layout(t)

	if err := os.Symlink(outside, filepath.Join(ws, ".clif")); err != nil {
		t.Fatal(err)
	}

	if err := MkdirAll(filepath.Join(ws, ".clif", "deep"), 0o750, Roots(ws)); !errors.Is(err, ErrOutside) {
		t.Fatalf("want a refusal, got %v", err)
	}

	if _, err := os.Stat(filepath.Join(outside, "deep")); err == nil {
		t.Fatal("a directory was created outside the workspace")
	}
}

func TestResolveFollowsLinkLoopsToAnError(t *testing.T) {
	ws, _, _ := layout(t)

	a, b := filepath.Join(ws, "a"), filepath.Join(ws, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}

	if _, err := Resolve(a); err == nil {
		t.Fatal("a symlink loop resolved")
	}
}

// TestWithinTreatsADotDotNamedEntryAsInside: "..cache" is a file name, not a
// parent directory. Several callers once tested the prefix ".." alone.
func TestWithinTreatsADotDotNamedEntryAsInside(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "w", "root")

	if !Within(filepath.Join(root, "..cache", "a.cfc"), []string{root}) {
		t.Error("a ..cache directory under the root was reported outside")
	}

	if Within(filepath.Join(root, "..", "other"), []string{root}) {
		t.Error("a sibling of the root was reported inside")
	}
}
