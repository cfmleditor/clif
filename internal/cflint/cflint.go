// Package cflint provides CFLint binary management and execution.
package cflint

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.lsp.dev/protocol"

	"github.com/cfmleditor/clif/internal/conv"
)

// pinnedVersion is the CFLint clif runs unless CLIF_CFLINT or PATH names
// another: the one a release build embeds (scripts/fetch-cflint.sh pins the
// same one, TestPinnedVersionIsTheEmbeddedVersion) and the only one any build
// downloads. It used to download whatever `releases/latest` pointed at, and
// ran it unchecked, so whoever could publish a CFLint release could run code
// on every machine with a source-built clif and linting on.
const pinnedVersion = "1.5.17"

// executableSums is the sha256 of pinnedVersion's executable for each
// platform, keyed by binaryNameFor. It is the digest GitHub lists for the raw
// asset, which is also the file each .tar.gz and .zip holds, so one sum checks
// a download in either shape and a binary already in the cache:
//
//	gh api repos/cfmleditor/CFLint/releases/tags/<version> --jq '.assets[] | "\(.name) \(.digest)"'
//
// Moving pinnedVersion means replacing all five. A var so a test can serve
// executables of its own.
var executableSums = map[string]string{
	"cflint-macos-aarch64":     "ccea4da23b1af871fad7197edf66039c2c0ee42c88e75f0782d330489c79dd3b",
	"cflint-macos-amd64":       "18162a70ef8c84552d646c5f77c160a862f2cf8ac4b0a7a3e2fb06d8ceb32157",
	"cflint-linux-aarch64":     "48dc2c6ad226939f652d8170737bb3ef23b15dc4f1186c2bf41240d454a0ffaf",
	"cflint-linux-amd64":       "656b1993d5510676155fb5f06b083636e2cec20d0b9369141ff181b815ee2265",
	"cflint-windows-amd64.exe": "e4855f15c4eccc384e86333a325943df7bd5c746021688325fb252288b18a97c",
}

// downloadBase is where release assets are fetched from. A var rather than a
// const so a test can point it somewhere that is not the internet.
var downloadBase = "https://github.com/cfmleditor/CFLint/releases/download/"

// Result represents CFLint JSON output.
type Result struct {
	Issues []Issue `json:"issues"`
}

// Issue represents a single CFLint issue.
type Issue struct {
	Severity  string     `json:"severity"`
	ID        string     `json:"id"`
	Message   string     `json:"message"`
	Locations []Location `json:"locations"`
}

// Location represents where an issue occurs.
type Location struct {
	File     string `json:"file"`
	FileName string `json:"fileName"`
	Function string `json:"function"`
	Column   int    `json:"column"`
	Line     int    `json:"line"`
	Message  string `json:"message"`
	Variable string `json:"variable"`
	// Expression is the source CFLint reports the issue against; Offset is
	// its character offset in the file. Both are carried for the text and
	// JSON reports, which print them as CFLint does.
	Expression string `json:"expression"`
	Offset     int    `json:"offset"`
}

// Runner manages the CFLint binary.
type Runner struct {
	binPath string

	// minRank is the least severe CFLint level still reported, as a rank from
	// severityRank. noSeverityFloor reports everything, which is the default.
	minRank int
}

// NewRunner creates a Runner, downloading the binary if needed.
//
// minSeverity is a CFLint level name (`linting.minSeverity`); issues less
// severe than it are dropped. An empty or unrecognised value reports
// everything — see MinSeverityRank, which is where a caller should check a
// configured value if it wants to warn about a typo.
func NewRunner(ctx context.Context, minSeverity string) (*Runner, error) {
	binPath, err := ensureBinary(ctx)
	if err != nil {
		return nil, err
	}

	rank, ok := MinSeverityRank(minSeverity)
	if !ok {
		rank = noSeverityFloor
	}

	return &Runner{binPath: binPath, minRank: rank}, nil
}

// Scan runs CFLint on the given file and returns LSP diagnostics.
func (r *Runner) Scan(ctx context.Context, filePath string) ([]protocol.Diagnostic, error) {
	cmd := exec.CommandContext(ctx, r.binPath, "-stdin", filepath.Base(filePath), "-json", "-stdout", "-q") //nolint:gosec // the CFLint binary this package downloaded, given a file name and fixed flags

	content, err := os.ReadFile(filePath) //nolint:gosec // the workspace file being linted
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	cmd.Stdin = strings.NewReader(string(content))
	cmd.Dir = filepath.Dir(filePath)

	var stderr strings.Builder

	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if len(out) == 0 {
			return nil, fmt.Errorf("cflint failed: %w; stderr: %s", err, stderr.String())
		}
	}

	var result Result
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parsing cflint output: %w\nstdout: %s\nstderr: %s", err, string(out), stderr.String())
	}

	return toDiagnostics(result, r.minRank), nil
}

func toDiagnostics(result Result, minRank int) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	eachDiagnostic(result, minRank, func(_ string, d protocol.Diagnostic) {
		diags = append(diags, d)
	})

	return diags
}

// eachDiagnostic calls fn with every issue location that meets the floor, and
// the file CFLint names for it.
//
// A line or column below 1 is taken as the start: CFLint reports a negative
// column for some tag-file issues, which cast to a uint32 put the range past
// the end of the line.
func eachDiagnostic(result Result, minRank int, fn func(file string, d protocol.Diagnostic)) {
	for _, issue := range result.Issues {
		if !meetsFloor(issue.Severity, minRank) {
			continue
		}

		sev := mapSeverity(issue.Severity)

		for i := range issue.Locations {
			loc := &issue.Locations[i]

			line := max(loc.Line-1, 0)
			col := max(loc.Column-1, 0)

			fn(loc.File, protocol.Diagnostic{
				Range: protocol.Range{
					Start: protocol.Position{Line: conv.Uint32(line), Character: conv.Uint32(col)},
					End:   protocol.Position{Line: conv.Uint32(line), Character: conv.Uint32(col)},
				},
				Severity: sev,
				Source:   protocol.NewOptional("cflint"),
				Code:     protocol.String(issue.ID),
				Message:  protocol.String(loc.Message),
			})
		}
	}
}

// filesPerRun is how many files one CFLint run for ScanFiles is given. The
// list goes on the command line, and Windows caps a command line at 32K
// characters.
const filesPerRun = 100

// ScanFiles runs CFLint over files, from disk, and returns each file's
// diagnostics keyed by its cleaned path, for a report over a whole project. A
// file CFLint finds nothing in has no key.
//
// Files are passed by name, several to a run and a few runs at once, rather
// than as a folder: a folder scan descends into every dot-directory, and a git
// worktree under .claude/ is a second copy of the project.
func (r *Runner) ScanFiles(ctx context.Context, files []string) (map[string][]protocol.Diagnostic, error) {
	out := make(map[string][]protocol.Diagnostic)

	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		firstErr error
	)

	sem := make(chan struct{}, 4)

	for start := 0; start < len(files); start += filesPerRun {
		batch := files[start:min(start+filesPerRun, len(files))]

		wg.Add(1)

		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			result, err := r.run(ctx, "", "-file", strings.Join(batch, ","), "-json", "-stdout", "-q")

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				if firstErr == nil {
					firstErr = err
				}

				return
			}

			eachDiagnostic(result, r.minRank, func(file string, d protocol.Diagnostic) {
				file = filepath.Clean(file)
				out[file] = append(out[file], d)
			})
		}()
	}

	wg.Wait()

	if firstErr == nil {
		firstErr = ctx.Err()
	}

	return out, firstErr
}

// run runs the binary with args, in dir when it is not empty, and decodes its
// JSON report.
func (r *Runner) run(ctx context.Context, dir string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, r.binPath, args...) //nolint:gosec // the CFLint binary this package downloaded
	cmd.Dir = dir

	var stderr strings.Builder

	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return Result{}, fmt.Errorf("cflint failed: %w; stderr: %s", err, stderr.String())
	}

	var result Result
	if err := json.Unmarshal(out, &result); err != nil {
		return Result{}, fmt.Errorf("parsing cflint output: %w\nstdout: %s\nstderr: %s", err, string(out), stderr.String())
	}

	return result, nil
}

// noSeverityFloor is the minRank that reports every issue, however trivial. It
// is deliberately larger than any rank in severityRank, so the comparison in
// meetsFloor needs no special case for "no floor configured".
const noSeverityFloor = 1 << 30

// severityRank orders CFLint's levels, most severe first. The order is
// com.cflint.Levels', which is the scale CFLint's own rule docs use, so
// `linting.minSeverity` is written in the vocabulary a user reads there rather
// than in LSP severities.
//
// The filter deliberately runs on this raw scale rather than on what
// mapSeverity returns. mapSeverity currently folds INFO and COSMETIC up onto
// Warning so they stay visible, which makes the mapped severities too coarse to
// filter on — a floor of WARNING applied after the fold would keep every INFO
// it was meant to drop.
var severityRank = map[string]int{
	"FATAL":    0,
	"CRITICAL": 1,
	"ERROR":    2,
	"WARNING":  3,
	"CAUTION":  4,
	"INFO":     5,
	"COSMETIC": 6,
}

// MinSeverityRank turns a configured `linting.minSeverity` into a rank for
// NewRunner, reporting whether the name was recognised. An empty string is not
// an error — it is the default, meaning no floor — but it is not recognised
// either, so a caller can tell "unset" from "set to something valid" and warn
// only about a genuine typo.
func MinSeverityRank(name string) (int, bool) {
	rank, ok := severityRank[strings.ToUpper(strings.TrimSpace(name))]

	return rank, ok
}

// meetsFloor reports whether an issue at the given CFLint level is severe
// enough to report.
//
// A severity CFLint's enum does not list — UNKNOWN, or a level added upstream
// — is always reported, whatever the floor. The alternative is that an
// unrecognised level is silently dropped by a setting that never mentioned it,
// which is the same disappearing-diagnostic failure mapSeverity's default arm
// exists to avoid.
func meetsFloor(severity string, minRank int) bool {
	rank, ok := MinSeverityRank(severity)
	if !ok {
		return true
	}

	return rank <= minRank
}

// mapSeverity folds CFLint's eight levels onto the LSP's four. The full set is
// com.cflint.Levels: FATAL, CRITICAL, ERROR, WARNING, CAUTION, INFO, COSMETIC,
// UNKNOWN.
//
// Nothing here may return Hint, and nothing may return Information: an editor
// shows neither by default. VS Code draws a Hint as a faint underline and keeps
// it out of the Problems panel entirely, and its Problems filter hides
// Information unless "Show Infos" is ticked -- so a CFLint issue mapped to
// either is reported by the server, counted in the "cflint scan complete" log
// line, and then invisible, which reads as a diagnostic that was never
// produced. Every level therefore lands on Error or Warning, including the
// default arm, so an unrecognised severity surfaces rather than disappearing.
//
// INFO and COSMETIC being Warnings is the interim part: they are advisory in
// CFLint's own scale, and the honest mapping is Information. Restoring that
// means changing the one arm below -- and is worth doing together with a
// config key, so the choice belongs to the workspace rather than to this
// switch.
func mapSeverity(s string) protocol.DiagnosticSeverity {
	switch strings.ToUpper(s) {
	case "ERROR", "FATAL", "CRITICAL":
		return protocol.DiagnosticSeverityError
	case "WARNING", "CAUTION":
		return protocol.DiagnosticSeverityWarning
	case "INFO", "COSMETIC":
		return protocol.DiagnosticSeverityWarning
	default:
		return protocol.DiagnosticSeverityWarning
	}
}

func binaryName() string {
	return binaryNameFor(runtime.GOOS, runtime.GOARCH)
}

// binaryNameFor is binaryName with the platform passed in, so the mapping can
// be tested from a machine that is only ever one of them. An asset that is
// named wrong here is not a build error anywhere: it is an HTTP 404 at the
// first lint, on someone else's hardware.
func binaryNameFor(goos, goarch string) string {
	switch {
	case goos == "darwin" && goarch == "arm64":
		return "cflint-macos-aarch64"
	case goos == "darwin" && goarch == "amd64":
		return "cflint-macos-amd64"
	case goos == "linux" && goarch == "arm64":
		return "cflint-linux-aarch64"
	case goos == "linux" && goarch == "amd64":
		return "cflint-linux-amd64"
	case goos == "windows" && goarch == "amd64":
		return "cflint-windows-amd64.exe"
	default:
		return ""
	}
}

func cacheDir(version string) (string, error) {
	// CLIF_CFLINT_DIR moves where CFLint is unpacked or downloaded to: a
	// machine that refuses to run programs from the user's cache (AppLocker's
	// default rules, a noexec mount) can name a directory it allows.
	// Made absolute, so the LSP daemon and the CLI, started from different
	// directories, agree on where it is.
	if dir := os.Getenv("CLIF_CFLINT_DIR"); dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("CLIF_CFLINT_DIR=%s: %w", dir, err)
		}

		p := filepath.Join(abs, version)
		if err := os.MkdirAll(p, 0o750); err != nil { //nolint:gosec // the user names this directory on purpose
			return "", fmt.Errorf("CLIF_CFLINT_DIR=%s: %w", dir, err)
		}

		return p, nil
	}

	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	p := filepath.Join(dir, "clif", "cflint", version)

	// A CFLint downloaded while this server was cfmleditor-lsp is used where it
	// is, rather than downloaded again: a machine that cannot reach GitHub
	// would otherwise lose linting with the rename.
	if _, err := os.Stat(p); err != nil {
		legacy := filepath.Join(dir, "cfmleditor-lsp", "cflint", version)
		if info, err := os.Stat(legacy); err == nil && info.IsDir() {
			return legacy, nil
		}
	}

	return p, os.MkdirAll(p, 0o750)
}

// downloadClient replaces http.DefaultClient, which has no timeout at all: a
// server that accepts the connection and then stalls leaves the goroutine — and
// the linting it was setting up — hung for the life of the process.
var downloadClient = &http.Client{Timeout: 5 * time.Minute}

// assetKind is how a release asset has to be unpacked.
type assetKind int

const (
	// rawBinary is the executable itself, published with no container.
	rawBinary assetKind = iota
	// tarGz and zipped hold the executable under its plain name.
	tarGz
	zipped
)

// maxDownloadSize caps what is written out of an asset. CFLint is a GraalVM
// image, so ~90 MB is normal and this is generous — it is here so a corrupt or
// hostile archive cannot fill the disk while claiming to be a linter.
const maxDownloadSize = 512 << 20

// errAssetMissing is the one download failure worth trying another asset for.
var errAssetMissing = errors.New("asset not published for this release")

// asset is one candidate download for a platform's CFLint build.
type asset struct {
	name string
	kind assetKind
}

// assetsFor lists what to try, best first.
//
// The compressed assets are ~28 MB against ~90 MB for the raw binary — a
// GraalVM image is mostly zeroes and repeated metadata — so they are worth
// preferring for a download that happens in the middle of somebody's first
// edit. They are recent, though, and not every published build has one yet, so
// the raw binary stays as the fallback rather than the lint simply failing.
func assetsFor(goos, goarch string) []asset {
	name := binaryNameFor(goos, goarch)
	if name == "" {
		return nil
	}

	if goos == "windows" {
		return []asset{
			{name: strings.TrimSuffix(name, ".exe") + ".zip", kind: zipped},
			{name: name, kind: rawBinary},
		}
	}

	return []asset{
		{name: name + ".tar.gz", kind: tarGz},
		{name: name, kind: rawBinary},
	}
}

// ensureBinary finds CFLint: the file CLIF_CFLINT names; one on PATH, which
// has always won so a team can pin its own; then the one a release build of
// clif carries inside it (embedded.go), unpacked into the cache once; then a
// cached download, then a download. CLIF_CFLINT_DIR moves the cache.
func ensureBinary(ctx context.Context) (string, error) {
	// CLIF_CFLINT names the binary outright, ahead of everything else. Set and
	// missing is an error rather than a fall-through: whoever set it meant
	// that one.
	if p := os.Getenv("CLIF_CFLINT"); p != "" {
		return namedBinary(p)
	}

	if p, err := exec.LookPath("cflint"); err == nil {
		return p, nil
	}

	if embeddedCFLint != "" {
		return embeddedBinary()
	}

	name := binaryName()
	if name == "" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return downloadedBinary(ctx, name)
}

// downloadedBinary returns pinnedVersion's binary from the cache, downloading
// it the first time. A cached file is used only when it has the pinned sum: an
// older clif downloaded into the same directory without checking anything, so
// what is there may not be what the pin names.
func downloadedBinary(ctx context.Context, name string) (string, error) {
	want, ok := executableSums[name]
	if !ok {
		return "", fmt.Errorf("no pinned checksum for %s", name)
	}

	dir, err := cacheDir(pinnedVersion)
	if err != nil {
		return "", err
	}

	binPath := filepath.Join(dir, name)
	if _, err := os.Stat(binPath); err == nil {
		if fileHasSum(binPath, want) {
			return binPath, nil
		}

		// Removed rather than renamed over: installAsset accepts a file left at
		// binPath when its rename fails, which is how it tolerates a Windows
		// clif running its own copy, and this one is not to be accepted.
		if err := os.Remove(binPath); err != nil {
			return "", fmt.Errorf("cflint at %s does not have the pinned checksum and cannot be replaced: %w", binPath, err)
		}
	}

	assets := assetsFor(runtime.GOOS, runtime.GOARCH)
	if len(assets) == 0 {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	var missing error

	for _, a := range assets {
		err := fetchAsset(ctx, downloadBase+pinnedVersion+"/"+a.name, binPath, a.kind, want)
		if err == nil {
			verified.Store(binPath, want)

			return binPath, nil
		}

		// Only a release that does not carry this asset is worth asking for
		// another one. A refused connection, a timeout or a wrong checksum
		// would fail the same way twice.
		if errors.Is(err, errAssetMissing) {
			missing = err

			continue
		}

		return "", err
	}

	return "", fmt.Errorf("cflint %s: %w", pinnedVersion, missing)
}

// verified remembers which cached binaries this process has already found to
// have their sum, so a second Runner does not read 90MB again.
var verified sync.Map

// fileHasSum says whether the file at p has the sha256 want.
func fileHasSum(p, want string) bool {
	if got, ok := verified.Load(p); ok && got == want {
		return true
	}

	f, err := os.Open(p) //nolint:gosec // a path in clif's own cache
	if err != nil {
		return false
	}

	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}

	if hex.EncodeToString(h.Sum(nil)) != want {
		return false
	}

	verified.Store(p, want)

	return true
}

// fetchAsset downloads one asset and leaves the executable at binPath.
//
// Written beside the target and renamed into place. Writing binPath directly
// meant an interruption — the process killed, the machine losing power, two
// sessions downloading at once — left a truncated file at exactly the path
// ensureBinary accepts as a cached binary, so linting stayed broken for every
// later run until someone deleted it by hand. Rename is atomic within a
// directory, so binPath either does not exist or is a complete download.
//
// The executable is installed only when its sha256 is wantSum, checked before
// the rename, so a download that fails the check leaves nothing behind.
func fetchAsset(ctx context.Context, url, binPath string, kind assetKind, wantSum string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("downloading cflint: %w", err)
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading cflint: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%s: %w", url, errAssetMissing)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading cflint: HTTP %d", resp.StatusCode)
	}

	return installAsset(resp.Body, binPath, kind, wantSum)
}

// rename is os.Rename, replaced by a test to lose the race installAsset
// tolerates.
var rename = os.Rename

// installAsset unpacks an asset's executable to binPath, beside it first and
// renamed into place, for the reason fetchAsset gives. With a wantSum, an
// executable with any other sha256 is refused; the embedded asset passes none,
// since scripts/fetch-cflint.sh checked it before it was compiled in.
func installAsset(src io.Reader, binPath string, kind assetKind, wantSum string) error {
	dir := filepath.Dir(binPath)

	tmp, err := os.CreateTemp(dir, filepath.Base(binPath)+".part-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()

	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath) // no-op once the rename below has succeeded
	}()

	h := sha256.New()
	if err := writeBinary(io.MultiWriter(tmp, h), src, kind, dir); err != nil {
		return err
	}

	if got := hex.EncodeToString(h.Sum(nil)); wantSum != "" && got != wantSum {
		return fmt.Errorf("cflint %s for %s has sha256 %s, not the pinned %s; refusing it",
			pinnedVersion, filepath.Base(binPath), got, wantSum)
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil { //nolint:gosec // the downloaded CFLint binary has to be executable
		return err
	}

	if err := rename(tmpPath, binPath); err != nil {
		// Another clif unpacking or downloading the same binary got there
		// first. Windows refuses to rename over an executable that is running,
		// so its copy, already in use, is the answer rather than an error.
		if info, statErr := os.Stat(binPath); statErr == nil && info.Mode().IsRegular() {
			return nil
		}

		return err
	}

	return nil
}

// writeBinary copies the executable out of a downloaded asset.
func writeBinary(dst io.Writer, src io.Reader, kind assetKind, tmpDir string) error {
	switch kind {
	case rawBinary:
		return copyBinary(dst, src)
	case tarGz:
		return copyFromTarGz(dst, src)
	case zipped:
		return copyFromZip(dst, src, tmpDir)
	default:
		return fmt.Errorf("unknown asset kind %d", kind)
	}
}

// copyBinary copies up to maxDownloadSize, and treats reaching it as a failure
// rather than quietly installing a truncated executable.
func copyBinary(dst io.Writer, src io.Reader) error {
	written, err := io.Copy(dst, io.LimitReader(src, maxDownloadSize+1))
	if err != nil {
		return err
	}

	if written > maxDownloadSize {
		return fmt.Errorf("cflint is larger than %d bytes, refusing it", int64(maxDownloadSize))
	}

	return nil
}

// copyFromTarGz pulls the first regular file out of the archive. The release
// archives hold exactly one, the executable under its plain name, so this does
// not have to know what the platform calls it.
func copyFromTarGz(dst io.Writer, src io.Reader) error {
	gz, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("reading cflint archive: %w", err)
	}

	defer func() { _ = gz.Close() }()

	archive := tar.NewReader(gz)

	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return errors.New("cflint archive held no executable")
		}

		if err != nil {
			return fmt.Errorf("reading cflint archive: %w", err)
		}

		if header.Typeflag == tar.TypeReg {
			return copyBinary(dst, archive)
		}
	}
}

// copyFromZip does the same for a zip, which cannot be streamed: the central
// directory is at the end, so the archive is spooled to disk first.
func copyFromZip(dst io.Writer, src io.Reader, tmpDir string) error {
	spool, err := os.CreateTemp(tmpDir, "cflint-zip-*")
	if err != nil {
		return err
	}

	spoolPath := spool.Name()

	defer func() {
		_ = spool.Close()
		_ = os.Remove(spoolPath)
	}()

	size, err := io.Copy(spool, io.LimitReader(src, maxDownloadSize+1))
	if err != nil {
		return err
	}

	if size > maxDownloadSize {
		return fmt.Errorf("cflint archive is larger than %d bytes, refusing it", int64(maxDownloadSize))
	}

	archive, err := zip.NewReader(spool, size)
	if err != nil {
		return fmt.Errorf("reading cflint archive: %w", err)
	}

	var executable *zip.File

	for _, file := range archive.File {
		if !file.FileInfo().IsDir() {
			executable = file

			break
		}
	}

	if executable == nil {
		return errors.New("cflint archive held no executable")
	}

	entry, err := executable.Open()
	if err != nil {
		return fmt.Errorf("reading cflint archive: %w", err)
	}

	defer func() { _ = entry.Close() }()

	return copyBinary(dst, entry)
}

// embeddedBinary unpacks the CFLint this build carries into the cache, where
// pinnedVersion's download would go, once; later runs find it there. With no
// usable cache ($HOME unset, a read-only home) it unpacks into a fresh private
// temporary directory instead, so a locked-down machine still lints.
func embeddedBinary() (string, error) {
	name := binaryName()
	if name == "" {
		return "", fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	dir, err := cacheDir(pinnedVersion)
	if err == nil {
		var p string
		if p, err = unpackEmbedded(dir, name); err == nil {
			return p, nil
		}
	}

	// A directory someone named is where it goes or nowhere: falling back to
	// temp would run CFLint from the place CLIF_CFLINT_DIR exists to avoid,
	// and fail there with an error that hides the real problem.
	if os.Getenv("CLIF_CFLINT_DIR") != "" {
		return "", err
	}

	tmp, tmpErr := os.MkdirTemp("", "clif-cflint-")
	if tmpErr != nil {
		return "", fmt.Errorf("unpacking the embedded cflint: %w", errors.Join(err, tmpErr))
	}

	return unpackEmbedded(tmp, name)
}

// unpackEmbedded leaves the embedded CFLint at dir/name, unless it is there.
// A file there of another size is replaced: a truncated one, which a download
// from before downloads were atomic could leave, would otherwise fail every
// lint although this binary holds a good copy.
func unpackEmbedded(dir, name string) (string, error) {
	kind := tarGz
	if runtime.GOOS == "windows" {
		kind = zipped
	}

	binPath := filepath.Join(dir, name)
	if info, err := os.Stat(binPath); err == nil && sizeMatchesEmbedded(info.Size(), kind) {
		return binPath, nil
	}

	if err := installAsset(strings.NewReader(embeddedCFLint), binPath, kind, ""); err != nil {
		return "", fmt.Errorf("unpacking the embedded cflint: %w", err)
	}

	return binPath, nil
}

// sizeMatchesEmbedded says whether a cached binary is the embedded one's size,
// and trusts it when that size cannot be read.
func sizeMatchesEmbedded(size int64, kind assetKind) bool {
	want, err := embeddedSize(kind)

	return err != nil || size == want
}

// embeddedSize is the size of the executable in the embedded archive, read
// from the tar header or the zip directory without unpacking it.
func embeddedSize(kind assetKind) (int64, error) {
	if kind == zipped {
		archive, err := zip.NewReader(strings.NewReader(embeddedCFLint), int64(len(embeddedCFLint)))
		if err != nil {
			return 0, err
		}

		for _, file := range archive.File {
			if !file.FileInfo().IsDir() {
				return int64(file.UncompressedSize64), nil //nolint:gosec // an executable's size fits in int64
			}
		}

		return 0, errors.New("cflint archive held no executable")
	}

	gz, err := gzip.NewReader(strings.NewReader(embeddedCFLint))
	if err != nil {
		return 0, err
	}

	defer func() { _ = gz.Close() }()

	archive := tar.NewReader(gz)

	for {
		header, err := archive.Next()
		if err != nil {
			return 0, err
		}

		if header.Typeflag == tar.TypeReg {
			return header.Size, nil
		}
	}
}

// namedBinary checks the CFLint CLIF_CFLINT names and makes its path
// absolute: Runner.run sets the command's directory to the one being linted,
// and a relative path would be looked for there.
func namedBinary(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("CLIF_CFLINT=%s: %w", p, err)
	}

	info, err := os.Stat(abs) //nolint:gosec // the user names this binary on purpose
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("CLIF_CFLINT=%s is not a file", p)
	}

	// Windows runs a file by its extension; everywhere else it needs the bit.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("CLIF_CFLINT=%s is not executable", p)
	}

	return abs, nil
}
