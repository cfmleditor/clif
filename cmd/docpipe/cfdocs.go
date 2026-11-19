package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (p *pipeline) cfdocsTarget(ctx context.Context, bump bool) (string, error) {
	if !bump {
		return p.readPin("cfdocs")
	}

	b, err := p.retry(ctx, "git", "ls-remote", p.cfdocsRepo, "refs/heads/master")
	if err != nil {
		return "", err
	}

	fields := strings.Fields(string(b))
	if len(fields) != 2 || fields[1] != "refs/heads/master" || !validCommit(fields[0]) {
		return "", errors.New("cfdocs master did not name a commit")
	}

	return fields[0], nil
}

func (p *pipeline) fetchCFDocs(ctx context.Context, bump bool) error {
	target, err := p.cfdocsTarget(ctx, bump)
	if err != nil {
		return err
	}

	dir, marker := p.path("docs/src/cfdocs"), p.path("docs/.sha-cfdocs")
	if b, err := readFile(marker); err == nil && strings.TrimSpace(string(b)) == target {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			fmt.Printf("cfdocs at %.8s, already staged\n", target)

			return nil
		}
	}

	tmp, err := os.MkdirTemp("", "clif-cfdocs-*")
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(tmp) }()

	checkout := filepath.Join(tmp, "cfdocs")

	fmt.Printf("Downloading cfdocs data/en at %.8s...\n", target)

	for _, args := range [][]string{
		{"init", "-q", checkout},
		{"-C", checkout, "remote", "add", "origin", p.cfdocsRepo},
		{"-C", checkout, "sparse-checkout", "set", "data/en"},
	} {
		if _, err := p.command(ctx, "git", args...); err != nil {
			return err
		}
	}

	if _, err := p.retry(ctx, "git", "-C", checkout, "fetch", "-q", "--depth", "1", "--filter=blob:none", "origin", target); err != nil {
		return fmt.Errorf("fetching cfdocs %s: %w", target, err)
	}

	if _, err := p.retry(ctx, "git", "-C", checkout, "checkout", "-q", "FETCH_HEAD"); err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(checkout, "data", "en", "*.json"))
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return errors.New("cfdocs commit has no data/en/*.json")
	}

	staged, err := stageDir(dir)
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(staged) }()

	for _, file := range files {
		if err := copyFile(file, filepath.Join(staged, filepath.Base(file))); err != nil {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	// Never leave a marker claiming the old pin for a newly installed copy.
	if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := installDir(staged, dir, func() error { return atomicWrite(marker, []byte(target+"\n")) }); err != nil {
		return err
	}

	fmt.Printf("Copied %d JSON files to docs/src/cfdocs/\n", len(files))

	return nil
}
