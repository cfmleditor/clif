// Package confine keeps a write inside the directories it is meant for, after
// symlinks.
//
// A path checked as text says nothing about where a write to it lands. A
// cloned repository can commit a symlink at a report's default name, or a
// symlinked directory where a report goes, and a write that follows either
// replaces a file anywhere the user can write: ~/.zshrc, a key, another
// project. Every check here resolves the symlinks first and compares the
// result with the roots resolved the same way, so a link that stays inside the
// roots is still followed and one that leaves them is refused.
package confine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxLinks bounds how many symlinks Resolve follows at a final name, as the
// kernel bounds a lookup, so two links pointing at each other end in an error
// rather than a loop.
const maxLinks = 40

// Roots makes each root absolute and resolves its symlinks, so it compares like
// with like against a resolved path. On macOS the system temp directory is
// itself behind a symlink (/var is /private/var), and comparing a resolved path
// against an unresolved root refuses every write beneath it. A root that does
// not exist is kept as its absolute path.
func Roots(roots ...string) []string {
	out := make([]string, 0, len(roots))

	for _, r := range roots {
		if r == "" {
			continue
		}

		if resolved, err := Resolve(r); err == nil {
			out = append(out, resolved)
		}
	}

	return out
}

// Resolve is path made absolute with every symlink in it resolved, including a
// symlink at its final name, whether or not the path exists. The part of the
// path that does not exist yet is joined, as written, to the resolved part
// that does, because that is where creating it would put it.
func Resolve(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return resolve(abs, 0)
}

func resolve(abs string, depth int) (string, error) {
	if depth > maxLinks {
		return "", fmt.Errorf("%s: too many levels of symbolic links", abs)
	}

	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}

	// A dangling link at the final name: EvalSymlinks fails on it, and
	// writing to it creates whatever it points at, so that is the path to
	// check.
	if info, err := os.Lstat(abs); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		target, err := os.Readlink(abs)
		if err != nil {
			return "", err
		}

		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(abs), target)
		}

		return resolve(filepath.Clean(target), depth+1)
	}

	parent := filepath.Dir(abs)
	if parent == abs {
		return abs, nil
	}

	dir, err := resolve(parent, depth)
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, filepath.Base(abs)), nil
}

// Within reports whether path is one of roots or lies beneath one. Both sides
// are compared as given: pass a resolved path and Roots.
func Within(path string, roots []string) bool {
	for _, r := range roots {
		rel, err := filepath.Rel(r, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true
		}
	}

	return false
}

// ErrOutside is what every refusal here wraps.
var ErrOutside = errors.New("outside the directories it may be written in")

// Check resolves path and returns where a write to it lands, or an error
// wrapping ErrOutside when that is not under one of roots. roots must come from
// Roots.
func Check(path string, roots []string) (string, error) {
	resolved, err := Resolve(path)
	if err != nil {
		return "", err
	}

	if !Within(resolved, roots) {
		if resolved != filepath.Clean(path) {
			return "", fmt.Errorf("refusing to write %s: it resolves to %s, %w", path, resolved, ErrOutside)
		}

		return "", fmt.Errorf("refusing to write %s: %w", path, ErrOutside)
	}

	return resolved, nil
}

// WriteFile writes data to path when, after symlinks, it lands under one of
// roots. A symlink at path that stays inside is followed, so a project that
// keeps a report behind a link on purpose still has it rewritten; one that
// leaves is refused. The write itself goes through an os.Root on the resolved
// directory, so a link swapped in after the check cannot carry it out either.
func WriteFile(path string, data []byte, perm fs.FileMode, roots []string) error {
	f, err := Create(path, perm, roots)
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()

		return err
	}

	return f.Close()
}

// Create is WriteFile for a file written as a stream: it opens path for
// writing, truncated, under the same rule.
func Create(path string, perm fs.FileMode, roots []string) (*os.File, error) {
	resolved, err := Check(path, roots)
	if err != nil {
		return nil, err
	}

	return CreateNamed(resolved, perm)
}

// CreateNamed opens path for writing, truncated, through an os.Root on its
// directory: only the file name is confined, so a symlink planted at the name is
// refused, while a directory above it may be a link (a workspace reached through
// one still gets its reports). Create adds the check of where the file lands;
// a caller that has checked the path as text uses this alone.
func CreateNamed(path string, perm fs.FileMode) (*os.File, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}

	defer func() { _ = root.Close() }()

	return root.OpenFile(filepath.Base(path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
}

// MkdirAll creates dir and its parents when, after symlinks, it lands under one
// of roots. Checking first matters: os.MkdirAll follows a committed symlinked
// directory and creates the rest wherever it points. The result is checked
// again, since what was created may itself have been a link by then.
func MkdirAll(dir string, perm fs.FileMode, roots []string) error {
	if _, err := Check(dir, roots); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return err
	}

	_, err := Check(dir, roots)

	return err
}
