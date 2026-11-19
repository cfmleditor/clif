package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // docs sources, generated files and their lock retain their public repository modes
		return err
	}

	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(f.Name()) }()

	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()

		return err
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()

		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(f.Name(), path)
}

func recoverDir(dir string) error {
	_, err := os.Stat(dir)
	if err == nil {
		return nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	info, err := os.Stat(dir + ".old")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s.old is not a directory", dir)
	}

	return os.Rename(dir+".old", dir)
}

// installDir keeps the old copy until the optional associated write succeeds.
// Ordinary errors restore it; a killed process's missing directory is recovered
// by the next lock holder before it reads or stages anything.
func installDir(staged, dir string, commit func() error) (err error) {
	if err := recoverDir(dir); err != nil {
		return err
	}

	if err := os.RemoveAll(dir + ".old"); err != nil {
		return err
	}

	hadOld := false

	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, dir+".old"); err != nil {
			return err
		}

		hadOld = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	installed, committed := false, false

	defer func() {
		if err == nil || committed {
			return
		}

		if installed {
			err = errors.Join(err, os.RemoveAll(dir))
		}

		if hadOld {
			err = errors.Join(err, recoverDir(dir))
		}
	}()

	if renameErr := os.Rename(staged, dir); renameErr != nil {
		return renameErr
	}

	installed = true

	if commit != nil {
		if commitErr := commit(); commitErr != nil {
			return commitErr
		}
	}

	committed = true

	return os.RemoveAll(dir + ".old")
}

func copyFile(src, dst string) error {
	b, err := readFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, b, 0o644) //nolint:gosec // docs sources retain their public repository modes
}

func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}

		return os.Symlink(target, dst)
	}

	if !info.IsDir() {
		if err := copyFile(src, dst); err != nil {
			return err
		}

		return os.Chmod(dst, info.Mode().Perm())
	}

	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
			return err
		}
	}

	return nil
}

func stageDir(dir string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil { //nolint:gosec // docs sources, generated files and their lock retain their public repository modes
		return "", err
	}

	staged, err := os.MkdirTemp(filepath.Dir(dir), filepath.Base(dir)+".new-")
	if err != nil {
		return "", err
	}

	mode := os.FileMode(0o755)
	if info, err := os.Stat(dir); err == nil {
		mode = info.Mode().Perm()
	}

	if err := os.Chmod(staged, mode); err != nil {
		_ = os.Remove(staged)

		return "", err
	}

	return staged, nil
}
