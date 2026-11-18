//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/confine"
	docsources "github.com/cfmleditor/clif/internal/docs/sources"
	"github.com/cfmleditor/clif/internal/flock"
	"github.com/cfmleditor/clif/internal/vfs"
)

const (
	outFile = "internal/docs/generated_docs.go"

	// sourcesFile records which doc source each entry came from. It is
	// committed beside outFile, so the next regeneration can tell when an
	// entry that was Lucee's now comes from cfdocs: same name, Adobe's
	// signature, nothing removed for removedEntries to see.
	sourcesFile = "internal/docs/generated_sources.txt"

	// orderFile, written by scripts/assemble-docs.sh (the only place the
	// order is written down), names the doc sources from lowest precedence to
	// highest, each with the directory its files are read from, in place.
	orderFile = "docs/data/order"

	// lockFile pins each fetched source; stagedMarkers names the file each
	// fetch writes its staged version to, once the staged copy is in place.
	lockFile = "docs/sources.lock"

	// retainedSource is docs/retained. Its files are this repository's own,
	// written by hand, so a mistake in one is always refused, never skipped.
	retainedSource = docsources.Retained
)

// stagedMarkers maps each fetched source to the file holding what it was
// staged at (fetch-docs-cfdocs.sh removes it before a swap, and writes it
// only once the new copy is whole).
var stagedMarkers = map[string]string{"cfdocs": "docs/.sha-cfdocs"}

// baselineRef is the commit whose committed files are the baseline: HEAD,
// unless DOCS_BASELINE_REF names another. CI names what main held before, so
// a PR whose own commit already holds a lossy regeneration (its HEAD then has
// nothing to miss) is still compared with what it would change.
func baselineRef() string {
	r := strings.TrimSpace(os.Getenv("DOCS_BASELINE_REF"))
	if r == "" {
		return "HEAD"
	}

	// git would take a value starting with "-" for an option, not a commit.
	if strings.HasPrefix(r, "-") {
		refuse(fmt.Sprintf("DOCS_BASELINE_REF=%q is not a commit: it starts with \"-\".", r))
	}

	return r
}

// repoRelative gives path relative to the working directory, the
// repository's root (the generator runs from there), for `git show REF:./…`,
// which reads an absolute path as one inside the commit and finds nothing. A
// path outside the repository is refused: there is no copy of it at a base.
func repoRelative(path string) string {
	if path == "" {
		return path
	}

	wd, err := os.Getwd()
	if err != nil {
		refuse("Cannot find the working directory: " + err.Error())
	}

	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(wd, abs)
	}

	// Relative or absolute alike: "../x" is as far outside as "/x", and the
	// repository itself names no file. The directory is compared with its
	// symlinks resolved (confine.Resolve, through any part not there yet),
	// so another spelling of the repository's path (/tmp and /private/tmp
	// on macOS) is inside; the final name is kept as given, so git reads
	// the file the variable names, not what a link there points to.
	roots := confine.Roots(wd)
	dir, err := confine.Resolve(filepath.Dir(abs))
	if err != nil || len(roots) == 0 {
		refuse(fmt.Sprintf("Cannot resolve DOCS_ALLOWED_REMOVALS_FILE=%q: %v", path, err))
	}

	full := filepath.Join(dir, filepath.Base(abs))
	rel, err := filepath.Rel(roots[0], full)
	if err != nil || !confine.Within(full, roots) || rel == "." {
		refuse(fmt.Sprintf("DOCS_ALLOWED_REMOVALS_FILE=%q is outside the repository, so has no copy at a base to compare with.", path))
	}

	return filepath.ToSlash(rel)
}

// stopSeen is closed once this process has received a stopping signal; its
// handler then ends the run by it (stopOnSignal). caught: the stopping
// signals it handles, those not ignored at the start.
var (
	stopSeen = make(chan struct{})
	stopOnce sync.Once
	caught   []os.Signal
)

// git runs git with args and returns what it printed. A git that a stopping
// signal ended (SIGINT, SIGTERM, SIGHUP) may be no answer to report (a
// missing commit, an unreadable baseline): a Ctrl-C at the terminal reaches
// it and this process alike, and then this process's own handler stops the
// run by it. So that is waited for, and nothing more is done meanwhile:
// should this process have got the signal too, the handler ends it. One
// sent to git alone (a timeout, a pkill) reaches no handler here, and after
// a while is reported as the failure it is.
func git(args ...string) ([]byte, error) {
	out, err := exec.Command("git", args...).Output()

	// Both waits are bounded: should the handler not end the run (its
	// signal held for a write, say), the failure is returned as any.
	// Only for a signal this process handles: one it ignores (a SIGINT it
	// was started with ignored) reaches no handler here, whoever sent it.
	if sig, ok := flock.SignalOf(err); ok && slices.Contains(caught, sig) {
		select {
		case <-stopSeen:
			time.Sleep(3 * time.Second) // the handler is ending this run
		case <-time.After(500 * time.Millisecond): // it would have seen one by now
		}
	}

	return out, err
}

// gitOK reports whether git with args succeeds.
func gitOK(args ...string) bool {
	_, err := git(args...)

	return err == nil
}

// resolves reports whether ref names a commit. HEAD's answer comes from
// gitInfo, the one the banner's notes use, so the baseline read and the
// banner cannot disagree about it within one run.
func resolves(ref string) bool {
	if ref == "HEAD" {
		return gitInfo().head
	}

	_, err := git("rev-parse", "--verify", "--quiet", ref+"^{commit}")

	return err == nil
}

// baselineStatus says what reading a committed file found. Each case is its
// own value, not a combination of nils and errors to reassemble: confusing
// "could not read the baseline" with "there is none" would let
// DOCS_ALLOW_REMOVALS=1 drop it without anyone meaning to.
type baselineStatus int

const (
	baselinePresent    baselineStatus = iota // the file at the ref: data holds it, never nil
	baselineAbsent                           // the ref has no such file
	baselineNoGit                            // git is missing, or distrusts this checkout
	baselineNoRef                            // the ref does not resolve (no commits yet, a corrupt ref)
	baselineUnreadable                       // the ref has it, but git failed to read it (err says why)
)

// committed reads a generated file as committed at baselineRef, never as it
// is on disk: an uncommitted regeneration that already lost entries must not
// become the baseline the next one is compared with.
//
// "REF:./path" is relative to the working directory; "REF:path" would be
// relative to the repository root, and miss when clif is a subdirectory of
// another repository (vendored, or in a monorepo).
func committed(ref, path string) (status baselineStatus, data []byte, err error) {
	g := gitInfo()
	if !g.available {
		return baselineNoGit, nil, nil
	}

	if !resolves(ref) {
		return baselineNoRef, nil, nil
	}

	at, err := atRef(ref, path)
	if err != nil {
		return baselineUnreadable, nil, err
	}

	if !at {
		return baselineAbsent, nil, nil
	}

	b, err := git("show", ref+":./"+path)
	if err != nil {
		return baselineUnreadable, nil, fmt.Errorf("git show %s:./%s: %w", ref, path, err)
	}

	if b == nil {
		b = []byte{}
	}

	return baselinePresent, b, nil
}

// atRef reports whether ref has path (relative to the working directory).
// ls-tree lists the path when ref has it and nothing when it does not; it
// fails only when git does.
func atRef(ref, path string) (bool, error) {
	listed, err := git("ls-tree", "--name-only", ref, "--", path)
	if err != nil {
		return false, fmt.Errorf("git ls-tree %s -- %s: %w", ref, path, err)
	}

	return strings.TrimSpace(string(listed)) != "", nil
}

// gitState is what git can say here, asked once per run (gitInfo), so the
// checks and the banner cannot get different answers within one run.
type gitState struct {
	// available: git is installed, trusts this checkout, and the checkout is
	// clif's (this script is tracked in it). A source tarball unpacked inside
	// some other repository (a dotfiles home, say) is inside a work tree, but
	// not one that has ever held these files, and asking it about HEAD would
	// give wrong answers ("not committed").
	available bool
	// head: HEAD resolves to a commit.
	head bool
}

var gitInfo = sync.OnceValue(func() gitState {
	g := gitState{
		available: gitOK("ls-files", "--error-unmatch", "--", "scripts/generate_docs.go"),
	}
	g.head = g.available && gitOK("rev-parse", "--verify", "--quiet", "HEAD")

	return g
})

// sourced is an entry with the file and source it came from.
type sourced struct {
	Entry
	file, source string
}

type Param struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Required    bool              `json:"required"`
	RawValues   []json.RawMessage `json:"values"`
}

func (p *Param) Values() []string {
	var out []string
	for _, raw := range p.RawValues {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			out = append(out, s)
		} else {
			out = append(out, strings.TrimSpace(string(raw)))
		}
	}
	return out
}

type Entry struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Syntax      string  `json:"syntax"`
	Member      string  `json:"member"`
	Script      string  `json:"script"`
	Returns     string  `json:"returns"`
	Type        string  `json:"type"`
	Params      []Param `json:"params"`
	// MemberDetails is the member-function form a source records for a
	// function: `"abc".reFindNoCase( … )` is rEFindNoCase on a string.
	MemberDetails *struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"member_details"`
}

// docsLock is held by scripts/docslock.go (the OS's own file lock) for the
// whole of `make generate`: the fetch, the assemble and this generator, so no
// fetch or bump can swap a source while it is read. The generator takes no
// lock of its own: docslock gave the run a token (DOCS_LOCK_TOKEN), recorded
// in docs/.flock, and flock.Verify checks it is still the holder's, the same
// check `docslock --held` makes for the scripts.
const docsLock = "docs/.flock"

func requireLock() {
	if !holdsLock() {
		fmt.Fprintln(os.Stderr, "error: run the generator through `make generate` (scripts/generate.sh), which holds the docs lock")
		fmt.Fprintln(os.Stderr, "       across the fetch and the generation, so no other docs run can swap a source meanwhile.")
		os.Exit(1)
	}
}

// lockFD is the lock's descriptor, when docslock passed it (`docslock --then`,
// as `make generate` runs this): kept for the whole run, as an *os.File
// nothing refers to is closed by its finalizer, and with it the hold.
var lockFD *os.File

// holdsLock reports whether this run holds the docs lock. Given the lock's
// descriptor (DOCS_LOCK_FD), this process holds it itself, and keeps it should
// docslock be killed: it checks the descriptor is docs/.flock as on disk and
// the record holds its token, and marks it close-on-exec, so the git it runs
// cannot keep the lock past it. Otherwise (Windows, which passes no
// descriptor, or run some other way under the lock) the token must be the
// live holder's (flock.Verify).
func holdsLock() bool {
	token := os.Getenv("DOCS_LOCK_TOKEN")

	// Taken (and marked close-on-exec) only once it proves to be the lock's
	// descriptor for this run: a DOCS_LOCK_FD from elsewhere (a leaked
	// variable, an IDE's own fd 3) is left alone, and the token checked as
	// for any process under the lock.
	if lockFD == nil {
		if fd, err := strconv.Atoi(os.Getenv("DOCS_LOCK_FD")); err == nil {
			lockFD = flock.Inherited(fd, docsLock, token)
		}
	}

	if lockFD != nil {
		return flock.Holds(lockFD, docsLock, token)
	}

	return flock.Verify(docsLock, token)
}

var (
	// sigMu guards writing and pending. A signal while the two generated
	// files are being written is held in pending, not acted on, so the pair
	// is never left split; main then exits with that signal's code once both
	// are in place. At any other time the handler exits at once.
	sigMu   sync.Mutex
	writing bool
	written bool // both generated files are in place
	pending os.Signal
)

// stoppedBy prints why the run stopped and exits with the shell's code for
// sig (128 + its number).
//
// It ends by sig itself where it can (flock.Raise), not merely with its
// code, so what ran it sees an interrupted run: docslock passes a Ctrl-C on
// to the make that ran it only for a command that died of one.
func stoppedBy(sig os.Signal, what string) {
	fmt.Fprintf(os.Stderr, "\n%v: stopped; %s\n", sig, what)
	flock.Exit(sig)
}

// stopOnSignal exits on Ctrl-C, a kill or a hang-up with the shell's code for
// that signal, at once, or, during the write, as soon as both files are in
// place. A signal this process was started with ignored (nohup's SIGHUP) is
// left so; with none left to catch, nothing is (signal.Notify with no
// signals relays every one).
func stopOnSignal() {
	c := make(chan os.Signal, 1)
	if caught = flock.NotifyStops(c); len(caught) == 0 {
		return
	}

	go func() {
		for sig := range c {
			stopOnce.Do(func() { close(stopSeen) })

			sigMu.Lock()
			if writing {
				if pending == nil {
					pending = sig
				}
				sigMu.Unlock()

				continue
			}
			// Exiting with sigMu still held: released first, main could
			// start the write in the gap and be cut off between the two
			// renames, the split this is here to prevent.
			if written {
				stoppedBy(sig, "both generated files had already been written")
			}
			stoppedBy(sig, "nothing was written")
		}
	}()
}

// refuse stops without writing anything, and exits 1.
func refuse(reason ...string) {
	bar := strings.Repeat("=", 78)
	fmt.Fprintln(os.Stderr, bar)
	fmt.Fprintf(os.Stderr, "DOCS NOT REGENERATED: %s on disk was not rewritten.\n\n", outFile)
	for _, r := range reason {
		fmt.Fprintln(os.Stderr, r)
	}

	// Say when the files on disk are not the committed ones: changed since (an
	// earlier DOCS_ALLOW_REMOVALS run, an edit), or never committed at all,
	// which git diff cannot see. `make build` compiles whatever is on disk.
	// Without git (a tarball, a checkout git distrusts) or a HEAD, it cannot be
	// told, and "not committed" would be false: say nothing.
	files := []string{outFile, sourcesFile}
	if g := gitInfo(); !g.available || !g.head {
		files = nil
	}
	for _, f := range files {
		var note string
		at, err := atRef("HEAD", f)
		switch {
		case err != nil:
			continue // git cannot say: no note rather than a wrong one
		case !at:
			note = "is not committed" // at HEAD; staged is not committed
		default:
			_, err := git("diff", "--quiet", "HEAD", "--", f)
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
				note = "differs from HEAD's"
			}
		}
		if note != "" {
			fmt.Fprintf(os.Stderr, "\nNote: %s on disk %s (`git checkout -- %s` restores the committed one).\n", f, note, f)
		}
	}

	fmt.Fprintln(os.Stderr, bar)
	os.Exit(1)
}

// checkPins refuses unless every source docs/sources.lock pins was staged at
// exactly that version: docs from a version nobody pinned (a fetch that
// failed after the pin moved, leaving the old copy staged) must not be
// written, however few entries they would change.
func checkPins() {
	read := func(path string) (map[string]string, error) {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		return docsources.ReadPairs(path, b)
	}
	pins, err := read(lockFile)
	if err != nil {
		refuse("Cannot read the pins: "+err.Error(), "Restore docs/sources.lock from git, or `make docs-bump`.")
	}
	var off []string
	for src := range stagedMarkers {
		if pins[src] == "" {
			off = append(off, fmt.Sprintf("%s: not pinned in %s at all", src, lockFile))
		}
	}
	for src, pin := range pins {
		got := ""
		if marker, ok := stagedMarkers[src]; ok {
			if b, err := os.ReadFile(marker); err == nil {
				got = strings.TrimSpace(string(b))
			}
		}
		if got != pin {
			if got == "" {
				got = "nothing"
			}
			off = append(off, fmt.Sprintf("%s: staged %s, pinned %s", src, got, pin))
		}
	}
	if len(off) > 0 {
		sort.Strings(off)
		refuse(append(append([]string{"Staged docs are not the versions docs/sources.lock pins:"}, off...),
			"", "Rerun `make docs` (which fetches the pinned versions), or `make docs-bump` to move the pins.")...)
	}
}

func main() {
	// The handler first, so a signal at any point is handled the one way:
	// any write in progress finishes both renames, then the run exits.
	stopOnSignal()
	requireLock()
	ob, err := os.ReadFile(orderFile)
	if err != nil {
		refuse(fmt.Sprintf("No %s: run `make docs` first (fetches cfdocs and assembles the order).", orderFile))
	}
	checkPins()
	// Each line names a source and the directory its files are read from, in
	// place: docs/src/cfdocs (the staged fetch), docs/retained and docs/lucee
	// (committed). Nothing is copied, so no copy can go stale.
	var order []string
	dirs := map[string]string{}
	for i, l := range docsources.Lines(ob) {
		f := strings.Fields(l)
		if len(f) == 0 {
			continue
		}
		if len(f) != 2 || dirs[f[0]] != "" {
			refuse(fmt.Sprintf("%s:%d: want \"<source> <directory>\" once per source, have %q", orderFile, i+1, l),
				"Rerun `make docs`.")
		}
		order = append(order, f[0])
		dirs[f[0]] = f[1]
	}

	// Every docs/retained file must parse: one skipped would vanish without a
	// word. A fetched one that does not is reported (below). A retained file
	// must also be an entry: upstream ships listing files (all.json, functions.json) of
	// other types, but a retained file with "Function" for its type is a
	// mistake that would otherwise drop the entry it was kept for.
	var all []sourced
	var broken, brokenOurs []string
	for _, src := range order {
		files, err := filepath.Glob(filepath.Join(dirs[src], "*.json"))
		if err != nil {
			refuse(fmt.Sprintf("Cannot list %s: %v", dirs[src], err))
		}
		// A source with no files at all (its directory moved, deleted, or
		// emptied after the order was written) would otherwise pass for one
		// that documents nothing, and with DOCS_ALLOW_REMOVALS=1 take all its
		// entries out unnoticed. docs/retained alone may be empty: its last
		// file goes once Lucee documents that name again.
		if len(files) == 0 && src != retainedSource {
			refuse(fmt.Sprintf("%s (the %s docs) holds no JSON files.", dirs[src], src),
				"Run `make docs` (cfdocs), or restore docs/lucee from git.")
		}
		for _, f := range files {
			// docs/retained is written by hand here: a file there that does
			// not parse, or names no valid entry, is a mistake to fix, and
			// refused. Upstream's files (the fetched cfdocs, the vendored
			// docs/lucee, kept byte for byte as downloaded) are upstream's to
			// fix: skipped with a warning. `make docs-bump` refuses to vendor
			// a broken Lucee file in the first place.
			bad := &broken
			if src == retainedSource {
				bad = &brokenOurs
			}

			data, err := os.ReadFile(f)
			if err != nil {
				*bad = append(*bad, err.Error())

				continue
			}
			var e Entry
			if err := json.Unmarshal(data, &e); err != nil {
				*bad = append(*bad, fmt.Sprintf("%s: %v", f, err))

				continue
			}
			if e.Type != "function" && e.Type != "tag" {
				if src == retainedSource {
					*bad = append(*bad, fmt.Sprintf("%s: type %q, want \"function\" or \"tag\"", f, e.Type))
				}

				continue
			}
			// A name is an identifier. One with whitespace or a control
			// character ("abs ", a stray newline) would be written into
			// generated_sources.txt as-is, and then never match (a removal
			// nobody can make) or break the file for every later run.
			if !docsources.ValidName(e.Name) {
				*bad = append(*bad, fmt.Sprintf("%s: name %q is empty or holds whitespace or a control character", f, e.Name))

				continue
			}
			all = append(all, sourced{Entry: e, file: filepath.Base(f), source: src})
		}
	}
	if len(brokenOurs) > 0 {
		refuse(append([]string{"Broken docs/retained files:"}, brokenOurs...)...)
	}
	// A broken upstream file is upstream's to fix, and refusing every
	// regeneration until they do would hold back all their other updates.
	// It is skipped, loudly; if that costs an entry the committed docs had,
	// the removal check below refuses for it by name. (cfdocs has shipped
	// cfcatch.json and cfcol.json with markup in them, hidden for as long as
	// Lucee's same-named files overwrote them.)
	for _, b := range broken {
		fmt.Fprintf(os.Stderr, "warning: skipping an unreadable upstream doc file: %s\n", b)
	}
	if len(all) == 0 {
		refuse("No entries in any doc source: run `make docs` first.")
	}

	cands := make([]docsources.Candidate, len(all))
	for i, e := range all {
		cands[i] = docsources.Candidate{Key: docsources.Key(e.Type, e.Name), Source: e.source, File: e.file}
	}
	keep, sameSource, shadowed := docsources.Pick(cands)
	kept := make([]sourced, len(keep))
	for i, idx := range keep {
		kept[i] = all[idx]
	}
	// Two docs/retained files for one name are a mistake here, whatever
	// outranks them today (one would be silently dropped, with its signature,
	// the day Lucee stops documenting the name): refused. Upstream's alias or
	// leftover is upstream's to sort out: Pick keeps the later file, noted.
	retainedFiles := map[string][]string{}
	for _, c := range cands {
		if c.Source == retainedSource {
			retainedFiles[c.Key] = append(retainedFiles[c.Key], c.File)
		}
	}
	var twiceOurs []string
	for k, files := range retainedFiles {
		if len(files) > 1 {
			twiceOurs = append(twiceOurs, k+": "+strings.Join(files, ", "))
		}
	}
	if len(twiceOurs) > 0 {
		sort.Strings(twiceOurs)
		refuse(append([]string{"docs/retained documents a name twice; keep one file per name:"}, twiceOurs...)...)
	}
	for _, d := range sameSource {
		if d.Source != retainedSource {
			fmt.Fprintf(os.Stderr, "note: one source documents a name twice, keeping the later file: %s\n", d)
		}
	}
	for _, f := range shadowed {
		fmt.Fprintf(os.Stderr, "note: a higher source documents docs/retained/%s again; it can be deleted\n", f)
	}

	entries := make([]Entry, 0, len(kept))
	sources := make(map[string]string, len(kept))
	for _, e := range kept {
		entries = append(entries, e.Entry)
		sources[docsources.Key(e.Type, e.Name)] = e.source
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Type != entries[j].Type {
			return entries[i].Type == "tag"
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	// Refuse to lose entries silently. A fetch that failed drops a whole
	// source; an upstream that stopped publishing drops a few names. Either
	// way the fix is a decision, not a regeneration: keep the entry in
	// docs/retained, or accept the removal with DOCS_ALLOW_REMOVALS=1.
	//
	// The baseline is the committed sourcesFile. Without one there is
	// nothing to compare with, and every check would pass: that is refused
	// too, unless DOCS_ALLOW_REMOVALS=1 says this run is the one that
	// creates it.
	// DOCS_ALLOWED_REMOVALS_FILE (CI's comparison with a pull request's
	// base) takes the accepted losses from a committed file, read strictly;
	// DOCS_ALLOW_REMOVALS (a person at a terminal) from the environment.
	//
	// In the file, only the lines this pull request adds (those the base's
	// copy lacks) accept anything. A line the base already has is left over
	// from an earlier pull request: it accepts nothing, so it can neither wave
	// a later loss through by accident nor fail anyone, and only warns until
	// cleared.
	ref := baselineRef()
	allowedFile := repoRelative(os.Getenv("DOCS_ALLOWED_REMOVALS_FILE"))
	// A symlink would be read here through the link, but `git show` at the
	// base gives the link's own text, a path, which then cannot parse: every
	// left-over line would count as added and accept a loss again.
	if info, err := os.Lstat(allowedFile); allowedFile != "" && err == nil && info.Mode()&os.ModeSymlink != 0 {
		refuse(fmt.Sprintf("%s is a symlink: it must be the file itself, as git compares it with the base's copy.", allowedFile))
	}
	acceptFrom := "DOCS_ALLOW_REMOVALS"
	var accept docsources.Acceptance
	var leftOver []string
	var baseLines map[string]string
	baseBroken := false
	if allowedFile != "" {
		acceptFrom = allowedFile
		if os.Getenv("DOCS_ALLOW_REMOVALS") != "" {
			refuse("DOCS_ALLOW_REMOVALS and DOCS_ALLOWED_REMOVALS_FILE are both set: use one.")
		}
		b, err := vfs.ReadRegular(allowedFile)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			refuse("Cannot read " + allowedFile + ": " + err.Error())
		}
		lines, err := docsources.ParseAllowedRemovals(allowedFile, b)
		if err != nil {
			refuse(err.Error())
		}
		baseStatus, baseRaw, baseErr := committed(ref, allowedFile)
		if baseStatus == baselineUnreadable {
			refuse(fmt.Sprintf("Could not read %s at %s: %v", allowedFile, ref, baseErr))
		}
		// The base's copy only says which lines are left over. One that no
		// longer parses (a rule since tightened, a bad push to main) must not
		// fail every pull request, the one that fixes the file included: it
		// is warned about, and every line here counts as added.
		baseLines, err = docsources.ParseAllowedRemovals(ref+":"+allowedFile, baseRaw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: the base's %s does not parse (%v); treating every line here as added\n", allowedFile, err)
			baseLines, baseBroken = nil, true
		}
		var added map[string]bool
		added, leftOver = docsources.AddedLines(lines, baseLines)
		accept = docsources.Acceptance{Keys: added}
	} else {
		var ignored []string
		accept, ignored = docsources.ParseAcceptance(os.Getenv("DOCS_ALLOW_REMOVALS"))
		for _, k := range ignored {
			fmt.Fprintf(os.Stderr, "warning: DOCS_ALLOW_REMOVALS: %q is not \"function <name>\" or \"tag <name>\"; ignored\n", k)
		}
	}
	status, committedRaw, readErr := committed(ref, sourcesFile)

	switch {
	case status == baselinePresent:
	case status == baselineUnreadable:
		// The ref has the file but git could not read it: never "no
		// baseline", which DOCS_ALLOW_REMOVALS=1 would wave through.
		refuse(fmt.Sprintf("Could not read the baseline at %s: %v", ref, readErr),
			"A shallow or partial clone may lack it: fetch the commit, then rerun.")
	default:
		if !accept.All {
			why := map[baselineStatus]string{
				baselineAbsent: fmt.Sprintf("%s has no %s", ref, sourcesFile),
				baselineNoGit:  "git is not available here, or does not trust this checkout",
				baselineNoRef:  fmt.Sprintf("%s does not resolve to a commit", ref),
			}[status]
			refuse(fmt.Sprintf("No baseline to compare the new entries with (%s), so a loss could not be seen.", why),
				"Commit generated_sources.txt with generated_docs.go (both are written together). To create it",
				"from nothing, rerun `make generate` with DOCS_ALLOW_REMOVALS=1 and review the generated_docs.go diff.")
		}
	}

	parsed, err := docsources.Parse(sourcesFile, committedRaw)
	committedSources, committedOrder := parsed.Entries, parsed.Order
	if err != nil {
		refuse(err.Error(), "Fix the committed file (a merge left it broken?), or regenerate from a good commit.")
	}
	if status == baselinePresent && len(committedSources) == 0 {
		refuse(fmt.Sprintf("The committed %s lists no entries: a baseline of nothing protects nothing.", sourcesFile),
			"Restore it from an earlier commit.")
	}
	if committedOrder == nil {
		committedOrder = order // a file from before it recorded one
	}

	removed := docsources.Removed(sources, committedSources)
	lostByKey, toRetained := docsources.FellBack(sources, committedSources, committedOrder)

	var accepted, refusedRemoved, refusedLost []string
	for _, k := range removed {
		if accept.Accepts(k) {
			accepted = append(accepted, k+" (removed)")
		} else {
			refusedRemoved = append(refusedRemoved, k)
		}
	}
	var refusedLostKeys []string
	for k, move := range lostByKey {
		if accept.Accepts(k) {
			accepted = append(accepted, move)
		} else {
			refusedLost = append(refusedLost, move)
			refusedLostKeys = append(refusedLostKeys, k)
		}
	}
	sort.Strings(refusedLostKeys)
	sort.Strings(accepted)
	sort.Strings(refusedLost)

	if len(accepted) > 0 {
		fmt.Fprintf(os.Stderr, "%s: accepting %d: %s\n", acceptFrom, len(accepted), strings.Join(accepted, ", "))
	}
	for _, k := range leftOver {
		fmt.Fprintf(os.Stderr, "warning: %s still lists %q from an earlier pull request; it accepts nothing now, so clear it\n", acceptFrom, k)
	}
	named := make([]string, 0, len(accept.Keys))
	for k := range accept.Keys {
		named = append(named, k)
	}
	sort.Strings(named)
	var stale []string
	for _, k := range named {
		if !slices.Contains(removed, k) && lostByKey[k] == "" {
			stale = append(stale, k)
		}
	}
	// An entry new to the file that accepts nothing is a mistake (a typo, or
	// a removal since undone): refused, so it is not merged to sit there. One
	// the base's copy already lists, with only its line changed (a comment
	// tidied, say), is no new claim: it only warns. With the base's copy
	// unreadable there is no telling a new entry from an old one, so each
	// only warns, rather than block the pull request that fixes that copy.
	// Under DOCS_ALLOW_REMOVALS everything here only warns.
	var staleNew []string
	for _, k := range stale {
		if _, listed := baseLines[k]; allowedFile != "" && !baseBroken && !listed {
			staleNew = append(staleNew, k)

			continue
		}
		fmt.Fprintf(os.Stderr, "warning: %s names %q, which this run neither removes nor moves\n", acceptFrom, k)
	}
	if len(staleNew) > 0 {
		reason := append([]string{allowedFile + ": this pull request adds entries it neither removes nor moves:"}, staleNew...)
		refuse(append(reason, "", "Delete those lines, or check the name (\"function <name>\" or \"tag <name>\").")...)
	}

	if len(refusedRemoved)+len(refusedLost) > 0 {
		var reason []string
		if len(refusedRemoved) > 0 {
			reason = append(reason, fmt.Sprintf("Committed entries missing from every doc source (%d):\n  %s\n",
				len(refusedRemoved), strings.Join(refusedRemoved, ", ")))
		}
		if len(refusedLost) > 0 {
			reason = append(reason, fmt.Sprintf("Entries falling back to a lower source, Adobe's cfdocs for Lucee's (%d):\n  %s\n",
				len(refusedLost), strings.Join(refusedLost, ", ")))
		}
		reason = append(reason,
			"So this build has none of the sources' other updates either. If a source failed to fetch,",
			"fix that. If upstream stopped publishing them, copy each entry into docs/retained/ (see its",
			"README).")
		for _, k := range append(append([]string{}, refusedRemoved...), refusedLostKeys...) {
			if slices.Contains(leftOver, k) {
				reason = append(reason, fmt.Sprintf("%s is listed in %s, but its line is left over from an earlier pull request,", k, allowedFile),
					"and a leftover line accepts nothing. To remove it again on purpose, change that line (a new # reason).")
			}
		}
		if allowedFile != "" {
			// CI's comparison with what main held before (a pull request's
			// base, or the commit a push moved main from): the banner's HEAD
			// notes below are about the checkout, not this baseline.
			reason = append(reason,
				fmt.Sprintf("This compared with %s, what main held before this change, not HEAD: a local `make generate`", ref),
				"can pass while this fails. To drop an entry on purpose, list it in "+allowedFile+".")
		} else {
			reason = append(reason,
				"To accept a loss deliberately, rerun `make generate` with DOCS_ALLOW_REMOVALS set to the entries,",
				"comma-separated (DOCS_ALLOW_REMOVALS=\"function foo,tag bar\"), or to 1 for all.")
		}
		refuse(reason...)
	}

	// Going ahead: now say which entries moved into docs/retained.
	for _, m := range toRetained {
		fmt.Fprintf(os.Stderr, "note: %s now comes from docs/retained; check its signature in the generated_docs.go diff\n", m)
	}

	if err := os.MkdirAll("internal/docs", 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Both files are built in memory and written to temporary files first,
	// then renamed into place, so a failed write leaves both as they were
	// rather than one new and one old (the next run's baseline).
	out := &bytes.Buffer{}

	fmt.Fprintln(out, "// Code generated by scripts/generate_docs.go; DO NOT EDIT.")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "package docs")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "func builtinEntries() []Entry {")
	fmt.Fprintln(out, "\treturn []Entry{")

	for _, e := range entries {
		fmt.Fprintf(out, "\t\t{Name: %s, Description: %s, Syntax: %s, Member: %s, Script: %s, Returns: %s, Type: %s",
			goStr(e.Name), goStr(e.Description), goStr(e.Syntax), goStr(e.Member), goStr(e.Script), goStr(e.Returns), goStr(e.Type))
		if len(e.Params) > 0 {
			fmt.Fprintln(out, ", Params: []Param{")
			for _, p := range e.Params {
				vals := p.Values()
				if len(vals) > 0 {
					fmt.Fprintf(out, "\t\t\t{Name: %s, Description: %s, Type: %s, Required: %v, Values: []string{",
						goStr(p.Name), goStr(p.Description), goStr(p.Type), p.Required)
					for i, v := range vals {
						if i > 0 {
							fmt.Fprint(out, ", ")
						}
						fmt.Fprint(out, goStr(v))
					}
					fmt.Fprintln(out, "}},")
				} else {
					fmt.Fprintf(out, "\t\t\t{Name: %s, Description: %s, Type: %s, Required: %v},\n",
						goStr(p.Name), goStr(p.Description), goStr(p.Type), p.Required)
				}
			}
			fmt.Fprintln(out, "\t\t}},")
		} else {
			fmt.Fprintln(out, "},")
		}
	}

	fmt.Fprintln(out, "\t}")
	fmt.Fprintln(out, "}")

	// The member functions, in a list of their own so a source that records
	// none leaves the entries above as they were.
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "func builtinMembers() []Member {")
	fmt.Fprintln(out, "\treturn []Member{")

	for _, e := range entries {
		if e.Type != "function" || e.MemberDetails == nil || e.MemberDetails.Name == "" || e.MemberDetails.Type == "" {
			continue
		}

		fmt.Fprintf(out, "\t\t{Type: %s, Name: %s, Function: %s},\n",
			goStr(strings.ToLower(e.MemberDetails.Type)), goStr(e.MemberDetails.Name), goStr(e.Name))
	}

	fmt.Fprintln(out, "\t}")
	fmt.Fprintln(out, "}")

	keys := make([]string, 0, len(sources))
	for k := range sources {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("# Generated by scripts/generate_docs.go; DO NOT EDIT. Which doc source each entry in\n")
	sb.WriteString("# generated_docs.go came from, so a regeneration can see Lucee's entry fall back to cfdocs'.\n")
	sb.WriteString(docsources.OrderPrefix + strings.Join(order, " ") + "\n")
	for _, k := range keys {
		sb.WriteString(k + "\t" + sources[k] + "\n")
	}
	// Checked again just before writing: the file deleted and created again
	// meanwhile, or, without the descriptor, the lock released (its docslock
	// killed), would let another docs run take it and swap a source under
	// this one.
	if !holdsLock() {
		refuse("This run no longer holds the docs lock ("+docsLock+"): its docslock was stopped, or the file was",
			"deleted, so another docs run may have changed a source meanwhile. Run `make generate` again.")
	}

	// A signal from here until both renames are done is held, then acted on.
	sigMu.Lock()
	writing = true
	sigMu.Unlock()

	clearStaleTemps(outFile, sourcesFile)
	err = writeAll([]generated{{path: outFile, data: out.Bytes()}, {path: sourcesFile, data: []byte(sb.String())}})

	sigMu.Lock()
	writing, written = false, err == nil
	sig := pending
	sigMu.Unlock()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if sig != nil {
			// Interrupted as well: the signal's code, so the run reads as
			// stopped, not as an ordinary failure.
			stoppedBy(sig, "the write failed, as above")
		}
		os.Exit(1)
	}
	if sig != nil {
		stoppedBy(sig, "both generated files had already been written")
	}

	fmt.Fprintf(os.Stderr, "generated %s with %d entries, and %s\n", outFile, len(entries), sourcesFile)
}

// runTag names this run's scratch: the first 16 characters of its token, as
// lib/underlock.sh's docs_scratch does, or its PID when run with none.
func runTag() string {
	if t := os.Getenv("DOCS_LOCK_TOKEN"); len(t) >= 16 {
		return t[:16]
	}

	return strconv.Itoa(os.Getpid())
}

// clearStaleTemps removes the temporary files killed generators left beside
// paths (PATH.tmp.<run>, and PATH.tmp and PATH.<pid>.tmp as earlier versions
// named them), which git would otherwise show as new files, as
// docs_clear_scratch does for the scripts. Called holding the lock, before
// this run writes its own.
//
// Only once ten minutes old: on Windows a generator left running after its
// docslock was killed may still be writing its own (it breaks away from the
// job, and holds no descriptor), and removing them would split its pair.
func clearStaleTemps(paths ...string) {
	for _, p := range paths {
		a, _ := filepath.Glob(p + ".tmp.*")
		b, _ := filepath.Glob(p + ".*.tmp")
		for _, s := range append(append(a, b...), p+".tmp") {
			if info, err := os.Stat(s); err == nil && time.Since(info.ModTime()) >= 10*time.Minute {
				_ = os.Remove(s)
			}
		}
	}
}

// generated is one file writeAll writes.
type generated struct {
	path string
	data []byte
}

// writeAll writes every file to a temporary beside it, and renames them into
// place, in the order given, only once all have been written. A full disk or
// a read-only file therefore leaves every file as it was, temporaries
// included. A rename failing part way can still split them: the next
// regeneration is unaffected (its baseline comes from HEAD, never the disk),
// but TestGeneratedSourcesMatchTheDocs fails on the mismatched pair until a
// regeneration rewrites both.
func writeAll(files []generated) error {
	var tmps []string
	defer func() {
		for _, tmp := range tmps {
			_ = os.Remove(tmp)
		}
	}()

	for _, f := range files {
		// This run's own name, PATH.tmp.<run> as the scripts name theirs
		// (lib/underlock.sh's docs_scratch), never one another generator
		// could be writing: on Windows, where the generator holds no
		// descriptor, one left running after its docslock was killed might
		// be.
		tmp := f.path + ".tmp." + runTag()
		tmps = append(tmps, tmp) // before writing: a partial write is cleaned up too
		if err := os.WriteFile(tmp, f.data, 0o644); err != nil {
			return err
		}
	}

	for i, f := range files {
		if err := os.Rename(tmps[i], f.path); err != nil {
			tmps = tmps[i:]

			return err
		}
	}
	tmps = nil

	return nil
}

func goStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
