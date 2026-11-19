package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	docsources "github.com/cfmleditor/clif/internal/docs/sources"
)

type pipeline struct {
	root       string
	cfdocsRepo string
	luceeURL   string
	command    func(context.Context, string, ...string) ([]byte, error)
	retryDelay time.Duration
}

func newPipeline() *pipeline {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		exit(1)
	}

	return &pipeline{root: root, cfdocsRepo: "https://github.com/cfmleditor/cfdocs.git", luceeURL: "https://docs.lucee.org/lucee-docs.zip", command: commandRunner, retryDelay: 5 * time.Second}
}

func (p *pipeline) path(name string) string { return filepath.Join(p.root, filepath.FromSlash(name)) }

func (p *pipeline) dispatch(ctx context.Context, args []string) error {
	command, rest := args[0], args[1:]
	if command == "pin" {
		return p.pin(rest)
	}

	if len(rest) != 0 {
		return usageError(command + " takes no arguments")
	}

	switch command {
	case "docs", "docs-cfdocs":
		if err := p.fetchCFDocs(ctx, false); err != nil {
			return err
		}

		return p.assemble(ctx)
	case "generate":
		if err := p.fetchCFDocs(ctx, false); err != nil {
			return err
		}

		if err := p.assemble(ctx); err != nil {
			return err
		}

		generate()

		return nil
	case "generate-only":
		generate()

		return nil
	case "bump":
		return p.bump(ctx)
	case "assemble":
		return p.assemble(ctx)
	case "fetch-cfdocs":
		return p.fetchCFDocs(ctx, os.Getenv("DOCS_BUMP") == "1")
	case "fetch-lucee":
		if os.Getenv("DOCS_BUMP") == "1" {
			return p.fetchLucee(ctx, nil)
		}

		fmt.Println("Lucee docs vendored in docs/lucee; make docs-bump takes Lucee's latest")

		return nil
	default:
		return usageError(fmt.Sprintf("unknown docs command %q", command))
	}
}

func (p *pipeline) recover() error {
	for _, name := range []string{"docs/src/cfdocs", "docs/lucee"} {
		if err := recoverDir(p.path(name)); err != nil {
			return err
		}

		for _, suffix := range []string{".new", ".new.*", ".new-*"} {
			matches, _ := filepath.Glob(p.path(name) + suffix)
			for _, old := range matches {
				if err := os.RemoveAll(old); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *pipeline) retry(ctx context.Context, name string, args ...string) ([]byte, error) {
	var (
		out []byte
		err error
	)
	for attempt := range 3 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		out, err = p.command(ctx, name, args...)
		if err == nil || ctx.Err() != nil {
			return out, err
		}

		if attempt < 2 {
			delay := time.Duration(attempt+1) * p.retryDelay
			fmt.Fprintf(os.Stderr, "  (failed; retrying in %v)\n", delay)

			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()

				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}

	return out, err
}

func validCommit(value string) bool {
	if len(value) != 40 {
		return false
	}

	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}

	return true
}

func (p *pipeline) pin(args []string) error {
	switch {
	case len(args) == 2 && args[0] == "get":
		value, err := p.readPin(args[1])
		if err == nil {
			fmt.Println(value)
		}

		return err
	case len(args) == 2 && args[0] == "check":
		if !validCommit(args[1]) {
			return fmt.Errorf("%q is not a 40-character commit", args[1])
		}

		return nil
	case len(args) == 3 && args[0] == "set":
		return p.writePin(args[1], args[2])
	default:
		return usageError("usage: docpipe pin get <source> | set <source> <commit> | check <commit>")
	}
}

func (p *pipeline) readPin(source string) (string, error) {
	b, err := readFile(p.path(lockFile))
	if err != nil {
		return "", err
	}

	pins, err := docsources.ReadPairs(lockFile, b)
	if err != nil {
		return "", err
	}

	value := pins[source]
	if !validCommit(value) {
		return "", fmt.Errorf("%s does not pin %s to a commit; make docs-bump pins the latest", lockFile, source)
	}

	return value, nil
}

func (p *pipeline) writePin(source, commit string) error {
	if !validCommit(commit) {
		return fmt.Errorf("%q is not a 40-character commit", commit)
	}

	if len(strings.Fields(source)) != 1 || strings.HasPrefix(source, "#") {
		return errors.New("invalid source name")
	}

	b, err := readFile(p.path(lockFile))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	var lines []string

	for line := range strings.SplitSeq(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) > 0 && f[0] != source {
			lines = append(lines, line)
		}
	}

	lines = append(lines, source+" "+commit)

	return atomicWrite(p.path(lockFile), []byte(strings.Join(lines, "\n")+"\n"))
}

func (p *pipeline) assemble(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := os.Stat(p.path("docs/src/cfdocs")); err != nil {
		return fmt.Errorf("cfdocs is not staged; run make docs: %w", err)
	}

	if _, err := readFile(p.path("docs/.sha-cfdocs")); err != nil {
		return err
	}

	if files, _ := filepath.Glob(p.path("docs/lucee/*.json")); len(files) == 0 {
		return errors.New("no vendored Lucee docs; restore docs/lucee from git")
	}

	if err := os.MkdirAll(p.path("docs/retained"), 0o755); err != nil { //nolint:gosec // docs sources, generated files and their lock retain their public repository modes
		return err
	}

	const order = "cfdocs docs/src/cfdocs\nretained docs/retained\nlucee docs/lucee\n"

	return atomicWrite(p.path(orderFile), []byte(order))
}

func (p *pipeline) bump(ctx context.Context) error {
	if err := p.fetchCFDocs(ctx, true); err != nil {
		return err
	}

	b, err := readFile(p.path("docs/.sha-cfdocs"))
	if err != nil {
		return err
	}

	commit := strings.TrimSpace(string(b))
	if !validCommit(commit) {
		return errors.New("staged cfdocs marker names no commit")
	}
	// Both sources stage before any committed input changes. If pin writing
	// fails, the Lucee swap rolls back as part of the same locked function.
	if err := p.fetchLucee(ctx, func() error { return p.writePin("cfdocs", commit) }); err != nil {
		return err
	}

	return p.assemble(ctx)
}
