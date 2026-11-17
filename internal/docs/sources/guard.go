package sources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfmleditor/clif/internal/vfs"
)

// The checks scripts/generate_docs.go makes before it writes the docs. They
// live here, not in that build-ignored script, so they are compiled, vetted,
// linted and tested like the rest of the module.

// Removed lists the keys baseline holds that now does not, sorted: entries a
// regeneration would drop. An upstream source that stops publishing a
// function would otherwise take it out of the builtins without anyone
// deciding to, and the unresolved report then flags every call to it.
func Removed(now, baseline map[string]string) []string {
	var removed []string

	for k := range baseline {
		if _, ok := now[k]; !ok {
			removed = append(removed, k)
		}
	}

	sort.Strings(removed)

	return removed
}

// FellBack finds the entries that would now come from a lower-ranked source
// than in baseline, ranked by baselineOrder (the order the baseline was
// generated under, so a source missing today still ranks where it did):
// Lucee's, or a retained, entry replaced by cfdocs' Adobe one of the same
// name. Nothing is removed, so only this sees it. lost maps each key to a
// description of the move.
//
// A move to Retained is never a fallback, though it ranks below Lucee:
// copying an entry Lucee dropped into docs/retained is the fix for losing it.
// Those moves come back in toRetained instead, sorted, for the caller to
// report: the retained copy may hold an older signature than the one Lucee
// last published. A move to a source baselineOrder does not know is not a
// lower one, and a baseline entry with no source recorded has nothing to
// compare.
func FellBack(now, baseline map[string]string, baselineOrder []string) (lost map[string]string, toRetained []string) {
	lost = map[string]string{}

	for k, was := range baseline {
		cur, ok := now[k]
		if !ok || cur == was || was == "" {
			continue
		}

		if cur == Retained {
			toRetained = append(toRetained, fmt.Sprintf("%s (was %s)", k, was))

			continue
		}

		wasRank, curRank := Rank(baselineOrder, was), Rank(baselineOrder, cur)
		if wasRank > 0 && curRank > 0 && curRank < wasRank {
			lost[k] = fmt.Sprintf("%s (%s -> %s)", k, was, cur)
		}
	}

	sort.Strings(toRetained)

	return lost, toRetained
}

// Acceptance is what DOCS_ALLOW_REMOVALS accepts: every removal and fallback
// in a run (All, from "1"), or only the entries named.
type Acceptance struct {
	All  bool
	Keys map[string]bool
}

// ParseAcceptance reads a DOCS_ALLOW_REMOVALS value: "1", or comma-separated
// entry keys ("function foo,tag bar"), so a removal someone decided on cannot
// carry an accidental one (a partial download) through with it. ignored lists
// the parts that are not a "function <name>" or "tag <name>".
func ParseAcceptance(v string) (a Acceptance, ignored []string) {
	if strings.TrimSpace(v) == "1" {
		return Acceptance{All: true}, nil
	}

	a.Keys = map[string]bool{}

	for k := range strings.SplitSeq(v, ",") {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}

		key, ok := ParseKey(k)
		if !ok {
			ignored = append(ignored, k)

			continue
		}

		a.Keys[key] = true
	}

	return a, ignored
}

// Accepts reports whether key's loss is accepted.
func (a Acceptance) Accepts(key string) bool { return a.All || a.Keys[key] }

// Candidate is one entry a source documents, in the file it came from.
type Candidate struct {
	Key, Source, File string
}

// Twice is a name one source documents in two files.
type Twice struct {
	Key, Source string
	Files       [2]string
}

func (t Twice) String() string {
	return fmt.Sprintf("%s: %s and %s in %s", t.Key, t.Files[0], t.Files[1], t.Source)
}

// Pick chooses one candidate per key, from the highest-ranked source, so
// precedence is per entry whatever each source names its files. cands must be
// in rank order, lowest first, and each source's files in name order, so a
// later candidate for a key always ranks at least as high as the one kept;
// keep lists the chosen indexes into cands, in first-seen order.
//
// twice lists a source documenting a name in two files (the later is kept)
// only where it decides the output: a duplicate in a source a higher one then
// replaces changes nothing, and noting it would teach readers to skip the
// notes that matter. shadowedRetained lists the Retained files a higher source
// replaces, sorted: those can go.
func Pick(cands []Candidate) (keep []int, twice []Twice, shadowedRetained []string) {
	best := map[string]int{} // key -> index into keep
	dup := map[string][]Twice{}

	for i, c := range cands {
		slot, seen := best[c.Key]
		if !seen {
			best[c.Key] = len(keep)
			keep = append(keep, i)

			continue
		}

		cur := cands[keep[slot]]
		if c.Source == cur.Source {
			dup[c.Key] = append(dup[c.Key], Twice{Key: c.Key, Source: c.Source, Files: [2]string{cur.File, c.File}})
		} else {
			if cur.Source == Retained {
				shadowedRetained = append(shadowedRetained, cur.File)
			}

			delete(dup, c.Key) // the lower source's duplicates no longer matter
		}

		keep[slot] = i
	}

	for _, ds := range dup {
		twice = append(twice, ds...)
	}

	sort.Slice(twice, func(i, j int) bool { return twice[i].String() < twice[j].String() })
	sort.Strings(shadowedRetained)

	return keep, twice, shadowedRetained
}

// ReadPairs reads "<name> <value>" lines (docs/sources.lock),
// blank lines and # comments aside, strictly: a malformed line or a name given
// twice is an error. label names the file in errors.
func ReadPairs(label string, b []byte) (map[string]string, error) {
	m := map[string]string{}

	for i, line := range Lines(b) {
		if t := strings.TrimSpace(line); t == "" || strings.HasPrefix(t, "#") {
			continue
		}

		f := strings.Fields(line)
		if len(f) != 2 {
			return nil, fmt.Errorf("%s:%d: want \"<name> <value>\", have %q", label, i+1, line)
		}

		if _, dup := m[f[0]]; dup {
			return nil, fmt.Errorf("%s:%d: %s given twice", label, i+1, f[0])
		}

		m[f[0]] = f[1]
	}

	return m, nil
}

// CheckEntryFile reports what is wrong with a doc source's JSON file, or nil:
// it must parse, and a function or tag entry must have a ValidName. Files of
// other types (Lucee's listings, all.json and the like) are fine as they are.
// `make docs-bump` runs it on every file it is about to vendor, so a broken
// one from upstream is refused there, before it can be committed.
func CheckEntryFile(data []byte) error {
	var e struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}

	if (e.Type == "function" || e.Type == "tag") && !ValidName(e.Name) {
		return fmt.Errorf("%s name %q is empty or holds whitespace or a control character", e.Type, e.Name)
	}

	return nil
}

// ManifestHash is the SHA-256, in hex, of a set of files given by name:
// over one "<sha256 of the file>  <name>" line per file, in byte order of
// name. docs/lucee/SOURCE records it for the vendored JSON, and a test checks
// it, so the files cannot be edited by hand while SOURCE still claims they
// are Lucee's zip as downloaded.
func ManifestHash(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}

	sort.Strings(names)

	h := sha256.New()
	for _, n := range names {
		_, _ = fmt.Fprintf(h, "%x  %s\n", sha256.Sum256(files[n]), n) // a hash.Hash never fails a write
	}

	return hex.EncodeToString(h.Sum(nil))
}

// HashDir reads every *.json in dir and returns their ManifestHash, the files
// it could not hash (unhashed: unreadable, or not a regular file), and those
// it hashed that fail CheckEntryFile (invalid: bad JSON, an invalid name). It
// is what `make docs-bump` records in docs/lucee/SOURCE (through
// scripts/check_vendored.go, which refuses both kinds) and what
// TestVendoredLuceeMatchesItsSource compares with it (which fails on unhashed
// ones only: whether a file's content passes the checks is the bump's
// business, and the checks may tighten after it was vendored), so the two
// cannot read or hash the files differently.
func HashDir(dir string) (hash string, unhashed, invalid []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, nil, err
	}

	files := map[string][]byte{}
	listed := 0

	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".json") {
			continue
		}

		listed++

		// The entry's own type, not what a symlink points to: vendored files
		// are kept byte for byte as downloaded, and a zip extracts no link.
		// The read itself goes through vfs.ReadRegular, which checks the
		// opened file, so nothing swapped in after this check (a FIFO, a
		// device) can hang it or read without limit.
		if !e.Type().IsRegular() {
			unhashed = append(unhashed, fmt.Sprintf("%s: not a regular file (%s)", n, e.Type()))

			continue
		}

		b, err := vfs.ReadRegular(filepath.Join(dir, n))
		if err != nil {
			unhashed = append(unhashed, fmt.Sprintf("%s: %v", n, err))

			continue
		}

		files[n] = b // hashed whatever its content: CheckEntryFile is separate

		if err := CheckEntryFile(b); err != nil {
			invalid = append(invalid, fmt.Sprintf("%s: %v", n, err))
		}
	}

	if listed == 0 {
		return "", nil, nil, fmt.Errorf("no *.json in %s", dir)
	}

	return ManifestHash(files), unhashed, invalid, nil
}

// ParseKey reads "function <name>" or "tag <name>" (the type in any case,
// any whitespace between) into an entry key: the one spelling rule for both
// DOCS_ALLOW_REMOVALS and docs/allowed-removals, so the two cannot accept
// different spellings of the same entry.
func ParseKey(s string) (key string, ok bool) {
	f := strings.Fields(s)
	if len(f) != 2 {
		return "", false
	}

	typ := strings.ToLower(f[0])
	if (typ != "function" && typ != "tag") || !ValidName(f[1]) {
		return "", false
	}

	return Key(typ, f[1]), true
}

// ParseAllowedRemovals reads docs/allowed-removals: one "function <name>" or
// "tag <name>" per line (ParseKey), with an optional trailing "# reason";
// blank lines and whole-line comments aside. It maps each key to its line in
// a normal form: the entry as its key (type and name lowercased, as CFML
// reads them) and the comment with each run of whitespace made one space. CI
// tells a line a pull request added or changed from one left over from an
// earlier pull request by comparing these, so re-listing an entry (a
// deliberate re-removal) means changing its reason. Spacing or the entry's
// letter case alone (an editor's tabs, a formatter's tidy) changes nothing,
// so cannot quietly make a left-over line accept a loss again.
//
// It is strict, because CI's comparison with a pull request's base accepts
// exactly these losses: a line it cannot read is an error naming it, never a
// warning. "1", which DOCS_ALLOW_REMOVALS takes for "accept everything", is
// refused here: a pull request cannot switch the comparison off by adding one
// line. label names the file in errors.
func ParseAllowedRemovals(label string, b []byte) (map[string]string, error) {
	lines := map[string]string{}

	for i, raw := range Lines(b) {
		line := strings.TrimSpace(raw)

		entry := line
		if c := strings.IndexByte(entry, '#'); c >= 0 {
			entry = strings.TrimSpace(entry[:c])
		}

		if entry == "" {
			continue
		}

		k, ok := ParseKey(entry)
		if !ok {
			return nil, fmt.Errorf("%s:%d: want \"function <name>\" or \"tag <name>\", have %q", label, i+1, entry)
		}

		if _, dup := lines[k]; dup {
			return nil, fmt.Errorf("%s:%d: %q listed twice", label, i+1, k)
		}

		normal := k
		if c := strings.IndexByte(line, '#'); c >= 0 {
			normal += " " + strings.Join(strings.Fields(line[c:]), " ")
		}

		lines[k] = normal
	}

	return lines, nil
}

// AddedLines lists the keys whose line in now is not in base exactly (a new
// entry, or a re-listed one whose line was changed), and, apart, those whose
// line is unchanged from base: left over from an earlier pull request.
func AddedLines(now, base map[string]string) (added map[string]bool, leftOver []string) {
	added = map[string]bool{}

	for k, line := range now {
		if base[k] == line {
			leftOver = append(leftOver, k)
		} else {
			added[k] = true
		}
	}

	sort.Strings(leftOver)

	return added, leftOver
}
