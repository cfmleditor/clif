package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	docsources "github.com/cfmleditor/clif/internal/docs/sources"
	"github.com/cfmleditor/clif/internal/vfs"
)

// Nested archive members may be large, but a zip bomb must not fill the disk.
const maxArchiveMember = 512 << 20

func extractMember(archive, name, out string) error {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = z.Close() }()

	for _, f := range z.File {
		if f.Name == name {
			return extractZipFile(f, out, maxArchiveMember)
		}
	}

	return fmt.Errorf("%s has no %s", archive, name)
}

func extractZipFile(f *zip.File, out string, limit int64) error {
	if !f.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular zip member", f.Name)
	}

	if limit < 0 {
		return errors.New("negative archive member limit")
	}

	if f.UncompressedSize64 > uint64(limit) {
		return fmt.Errorf("%s exceeds the archive member limit", f.Name)
	}

	r, err := f.Open()
	if err != nil {
		return err
	}

	defer func() { _ = r.Close() }()

	w, err := os.Create(out)
	if err != nil {
		return err
	}

	n, err := io.Copy(w, io.LimitReader(r, limit+1))
	closeErr := w.Close()

	if n > limit {
		return fmt.Errorf("%s exceeds the archive member limit", f.Name)
	}

	return errors.Join(err, closeErr)
}

func extractJSON(archive, dir string) (int, error) {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return 0, err
	}
	defer func() { _ = z.Close() }()

	count := 0
	seen := map[string]bool{}

	for _, f := range z.File {
		// Lucee's vendored docs are top-level JSON only. Never join a zip's
		// arbitrary paths to the output directory or follow a zip symlink.
		if !strings.HasSuffix(f.Name, ".json") || strings.ContainsAny(f.Name, "/\\") {
			continue
		}

		if seen[f.Name] {
			return 0, fmt.Errorf("duplicate zip member %s", f.Name)
		}

		seen[f.Name] = true
		if err := extractZipFile(f, filepath.Join(dir, filepath.Base(f.Name)), vfs.MaxFileSize); err != nil {
			return 0, err
		}

		count++
	}

	if count == 0 {
		return 0, errors.New("lucee archive held no top-level JSON docs")
	}

	return count, nil
}

func archiveHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func responseETag(headers []byte) string {
	etag := ""

	for line := range strings.SplitSeq(strings.ReplaceAll(string(headers), "\r", ""), "\n") {
		if strings.HasPrefix(line, "HTTP/") {
			etag = ""
		}

		if key, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(key, "ETag") {
			etag = strings.TrimSpace(value)
		}
	}

	if len(strings.Fields(etag)) != 1 {
		return ""
	}

	return etag
}

func (p *pipeline) fetchLucee(ctx context.Context, commit func() error) error {
	tmp, err := os.MkdirTemp("", "clif-lucee-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	archive, headers := filepath.Join(tmp, "lucee-docs.zip"), filepath.Join(tmp, "headers")

	fmt.Println("Downloading lucee-docs.zip...")

	if _, err := p.retry(ctx, "curl", "-fsSL", "--connect-timeout", "10", "--max-time", "600", "-D", headers, p.luceeURL, "-o", archive); err != nil {
		return err
	}

	middle, inner := filepath.Join(tmp, "lucee-docs-json-zipped.zip"), filepath.Join(tmp, "lucee-docs-json.zip")
	if err := extractMember(archive, "lucee-docs-json-zipped.zip", middle); err != nil {
		return err
	}

	if err := extractMember(middle, "lucee-docs-json.zip", inner); err != nil {
		return err
	}

	dir := p.path("docs/lucee")

	staged, err := stageDir(dir)
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(staged) }()

	count, err := extractJSON(inner, staged)
	if err != nil {
		return err
	}

	hash, unhashed, invalid, err := docsources.HashDir(staged)
	if err != nil {
		return err
	}

	if len(unhashed)+len(invalid) > 0 {
		return fmt.Errorf("lucee archive has broken files: %s", strings.Join(append(unhashed, invalid...), "; "))
	}

	zipHash, err := archiveHash(archive)
	if err != nil {
		return err
	}

	old, _ := readFile(filepath.Join(dir, "SOURCE"))

	var source strings.Builder

	for line := range strings.SplitSeq(string(old), "\n") {
		if strings.HasPrefix(line, "#") {
			source.WriteString(line + "\n")
		}
	}

	if source.Len() == 0 {
		source.WriteString("# Lucee's docs, vendored from " + p.luceeURL + "\n# make docs-bump replaces these JSON files and this provenance record.\n")
	}

	fmt.Fprintf(&source, "zip-sha256 %s\njson-sha256 %s\n", zipHash, hash)

	headerData, err := readFile(headers)
	if err != nil {
		return err
	}

	if etag := responseETag(headerData); etag != "" {
		fmt.Fprintf(&source, "etag %s\n", etag)
	}

	fmt.Fprintf(&source, "fetched %s\n", time.Now().UTC().Format("2006-01-02"))

	if err := os.WriteFile(filepath.Join(staged, "SOURCE"), []byte(source.String()), 0o644); err != nil { //nolint:gosec // docs sources, generated files and their lock retain their public repository modes
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") || e.Name() == "SOURCE" {
			continue
		}

		if err := copyTree(filepath.Join(dir, e.Name()), filepath.Join(staged, e.Name())); err != nil {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := installDir(staged, dir, commit); err != nil {
		return err
	}

	fmt.Printf("Vendored %d Lucee JSON files; review and commit the diff\n", count)

	return nil
}
