//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	docsources "github.com/cfmleditor/clif/internal/docs/sources"
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

// errNoHEAD is why there is no baseline when HEAD does not resolve.
var errNoHEAD = errors.New("HEAD does not resolve to a commit")

// errNoGit is why there is no baseline when git cannot answer.
var errNoGit = errors.New("git is not available here, or does not trust this checkout")

// committed is a generated file as committed at HEAD, never as it is on disk:
// an uncommitted regeneration that already lost entries must not become the
// baseline the next one is compared with. When git cannot say (not
// installed, not a checkout, or a checkout git refuses as unsafe) there is no
// baseline at all, and the caller refuses rather than trust the disk.
//
// "HEAD:./path" is relative to the working directory; "HEAD:path" would be
// relative to the repository root, and miss when clif is a subdirectory of
// another repository (vendored, or in a monorepo).
//
// Three answers: (nil, nil) when the file is not at HEAD (or there is no HEAD
// yet), (nil, errNoGit) when git cannot say, and the content otherwise, never
// nil even when empty. Any other git failure (a shallow or partial clone
// missing the object) is an error, not "not committed": that would let
// DOCS_ALLOW_REMOVALS=1 drop the baseline without anyone meaning to.
func committed(path string) ([]byte, error) {
	g := gitInfo()
	if !g.available {
		return nil, errNoGit
	}

	if !g.head {
		// No commits yet, or a HEAD git cannot resolve (a corrupt ref, a
		// worktree on a pruned branch): either way there is no baseline to
		// read, and the caller refuses unless DOCS_ALLOW_REMOVALS=1 says,
		// deliberately, to go ahead without one.
		return nil, errNoHEAD
	}

	at, err := atHEAD(path)
	if err != nil {
		return nil, err
	}
	if !at {
		return nil, nil
	}

	b, err := exec.Command("git", "show", "HEAD:./"+path).Output()
	if err != nil {
		return nil, fmt.Errorf("git show HEAD:./%s: %w", path, err)
	}
	if b == nil {
		b = []byte{}
	}

	return b, nil
}

// atHEAD reports whether HEAD has path (relative to the working directory),
// for both the baseline read and the banner's notes, so the two cannot
// disagree. ls-tree lists the path when HEAD has it and nothing when it does
// not; it fails only when git does.
func atHEAD(path string) (bool, error) {
	listed, err := exec.Command("git", "ls-tree", "--name-only", "HEAD", "--", path).Output()
	if err != nil {
		return false, fmt.Errorf("git ls-tree HEAD -- %s: %w", path, err)
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
		available: exec.Command("git", "ls-files", "--error-unmatch", "--", "scripts/generate_docs.go").Run() == nil,
	}
	g.head = g.available && exec.Command("git", "rev-parse", "--verify", "--quiet", "HEAD").Run() == nil

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
		at, err := atHEAD(f)
		switch {
		case err != nil:
			continue // git cannot say: no note rather than a wrong one
		case !at:
			note = "is not committed" // at HEAD; staged is not committed
		default:
			err := exec.Command("git", "diff", "--quiet", "HEAD", "--", f).Run()
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
	lock, err := read(lockFile)
	if err != nil {
		refuse("Cannot read the pins: "+err.Error(), "Restore docs/sources.lock from git, or `make docs-bump`.")
	}
	var off []string
	for src, pin := range lock {
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
	accept, ignored := docsources.ParseAcceptance(os.Getenv("DOCS_ALLOW_REMOVALS"))
	for _, k := range ignored {
		fmt.Fprintf(os.Stderr, "warning: DOCS_ALLOW_REMOVALS: %q is not \"function <name>\" or \"tag <name>\"; ignored\n", k)
	}
	committedRaw, gitErr := committed(sourcesFile)

	noBaseline := errors.Is(gitErr, errNoGit) || errors.Is(gitErr, errNoHEAD) || (gitErr == nil && committedRaw == nil)
	if noBaseline {
		if !accept.All {
			why := fmt.Sprintf("No committed %s to compare the new entries with", sourcesFile)
			if gitErr != nil {
				why += " (" + gitErr.Error() + ")"
			}
			refuse(why+", so a loss could not be seen.",
				"Commit it with generated_docs.go (both are written together). To create it from nothing,",
				"rerun `make generate` with DOCS_ALLOW_REMOVALS=1 and review the generated_docs.go diff.")
		}
	} else if gitErr != nil {
		// git is here and HEAD has the file, but reading it failed: never
		// "no baseline", which DOCS_ALLOW_REMOVALS=1 would wave through.
		refuse("Could not read the committed baseline: "+gitErr.Error(),
			"A shallow or partial clone may lack it: fetch the commit, then rerun.")
	}

	parsed, err := docsources.Parse(sourcesFile, committedRaw)
	committedSources, committedOrder := parsed.Entries, parsed.Order
	if err != nil {
		refuse(err.Error(), "Fix the committed file (a merge left it broken?), or regenerate from a good commit.")
	}
	if committedRaw != nil && len(committedSources) == 0 {
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
	for k, move := range lostByKey {
		if accept.Accepts(k) {
			accepted = append(accepted, move)
		} else {
			refusedLost = append(refusedLost, move)
		}
	}
	sort.Strings(accepted)
	sort.Strings(refusedLost)

	if len(accepted) > 0 {
		fmt.Fprintf(os.Stderr, "DOCS_ALLOW_REMOVALS: accepting %d: %s\n", len(accepted), strings.Join(accepted, ", "))
	}
	named := make([]string, 0, len(accept.Keys))
	for k := range accept.Keys {
		named = append(named, k)
	}
	sort.Strings(named)
	for _, k := range named {
		if !slices.Contains(removed, k) && lostByKey[k] == "" {
			fmt.Fprintf(os.Stderr, "warning: DOCS_ALLOW_REMOVALS names %q, which this run neither removes nor moves\n", k)
		}
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
			"README). To accept a loss deliberately, rerun `make generate` with DOCS_ALLOW_REMOVALS set to",
			"the entries, comma-separated (DOCS_ALLOW_REMOVALS=\"function foo,tag bar\"), or to 1 for all.")
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
	if err := writeAll([]generated{{outFile, out.Bytes()}, {sourcesFile, []byte(sb.String())}}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "generated %s with %d entries, and %s\n", outFile, len(entries), sourcesFile)
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
		tmp := f.path + ".tmp"
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
