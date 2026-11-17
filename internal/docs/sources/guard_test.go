package sources

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRemovedListsWhatTheBaselineHadAndNowLacks(t *testing.T) {
	now := map[string]string{"function abs": "lucee"}
	baseline := map[string]string{"function abs": "lucee", "tag cfstatic": "lucee", "function jwtsign": "retained"}

	if got, want := Removed(now, baseline), []string{"function jwtsign", "tag cfstatic"}; !slices.Equal(got, want) {
		t.Errorf("Removed = %v, want %v", got, want)
	}
}

// TestFellBack: Lucee's entry replaced by cfdocs' is a fallback; a move into
// docs/retained is reported apart, never refused; a baseline entry with no
// source, or a move to a source the baseline order does not know, is neither.
func TestFellBack(t *testing.T) {
	order := []string{"cfdocs", Retained, "lucee"}
	baseline := map[string]string{
		"function a": "lucee",    // -> cfdocs: fallback
		"function b": Retained,   // -> cfdocs: fallback
		"function c": "lucee",    // -> retained: reported, not refused
		"function d": "cfdocs",   // -> lucee: an upgrade
		"function e": "",         // no source recorded: nothing to compare
		"function f": "lucee",    // -> a new source: not lower
		"function g": "lucee",    // unchanged
		"function h": "gone-src", // a source the order does not know
	}
	now := map[string]string{
		"function a": "cfdocs", "function b": "cfdocs", "function c": Retained, "function d": "lucee",
		"function e": "cfdocs", "function f": "brand-new", "function g": "lucee", "function h": "cfdocs",
	}

	lost, toRetained := FellBack(now, baseline, order)

	if len(lost) != 2 || lost["function a"] == "" || lost["function b"] == "" {
		t.Errorf("lost = %v, want function a and function b", lost)
	}

	if want := []string{"function c (was lucee)"}; !slices.Equal(toRetained, want) {
		t.Errorf("toRetained = %v, want %v", toRetained, want)
	}
}

func TestParseAcceptance(t *testing.T) {
	if a, _ := ParseAcceptance("1"); !a.All || !a.Accepts("tag anything") {
		t.Error(`"1" does not accept everything`)
	}

	a, ignored := ParseAcceptance(" Tag CFStatic , function JwtSign,Funktion x,function a b,")
	if !a.Accepts("tag cfstatic") || !a.Accepts("function jwtsign") || a.Accepts("function abs") || a.All {
		t.Errorf("Keys = %v", a.Keys)
	}

	if want := []string{"Funktion x", "function a b"}; !slices.Equal(ignored, want) {
		t.Errorf("ignored = %v, want %v", ignored, want)
	}
}

// TestPickTakesTheHighestSourceAndNotesOnlyWhatDecides: precedence is per
// entry; a duplicate in a source a higher one replaces is not reported, one
// in the winning source is; a replaced retained file is listed.
func TestPickTakesTheHighestSourceAndNotesOnlyWhatDecides(t *testing.T) {
	cands := []Candidate{
		{"function a", "cfdocs", "a1.json"},
		{"function a", "cfdocs", "a2.json"}, // replaced below by lucee: not reported
		{"function b", Retained, "b.json"},
		{"function c", Retained, "c.json"},
		{"function a", "lucee", "a.json"},
		{"function b", "lucee", "b.json"}, // shadows the retained b
		{"function d", "lucee", "d1.json"},
		{"function d", "lucee", "d2.json"}, // decides d: reported
	}

	keep, twice, shadowed := Pick(cands)

	var got []string
	for _, i := range keep {
		got = append(got, cands[i].Key+"@"+cands[i].Source+"/"+cands[i].File)
	}

	want := []string{"function a@lucee/a.json", "function b@lucee/b.json", "function c@retained/c.json", "function d@lucee/d2.json"}
	if !slices.Equal(got, want) {
		t.Errorf("kept %v, want %v", got, want)
	}

	if len(twice) != 1 || twice[0].Key != "function d" || twice[0].Files != [2]string{"d1.json", "d2.json"} {
		t.Errorf("twice = %v, want only function d in lucee", twice)
	}

	if !slices.Equal(shadowed, []string{"b.json"}) {
		t.Errorf("shadowedRetained = %v, want [b.json]", shadowed)
	}
}

func TestReadPairs(t *testing.T) {
	m, err := ReadPairs("lock", []byte("# c\r\ncfdocs abc\r\n\r\nlucee def\n"))
	if err != nil || m["cfdocs"] != "abc" || m["lucee"] != "def" || len(m) != 2 {
		t.Fatalf("ReadPairs = %v, %v", m, err)
	}

	for name, text := range map[string]string{
		"three fields": "cfdocs a b\n",
		"one field":    "cfdocs\n",
		"twice":        "cfdocs a\ncfdocs b\n",
	} {
		if _, err := ReadPairs("lock", []byte(text)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCheckEntryFile(t *testing.T) {
	for name, tc := range map[string]struct {
		json string
		ok   bool
	}{
		"function":         {`{"name": "abs", "type": "function"}`, true},
		"listing":          {`{"name": "all", "type": "listing"}`, true},
		"no name, listing": {`{"type": "category"}`, true},
		"merge conflict":   {"{\n<<<<<<< HEAD\n", false},
		"bad name":         {`{"name": "abs ", "type": "function"}`, false},
		"empty tag name":   {`{"name": "", "type": "tag"}`, false},
	} {
		if err := CheckEntryFile([]byte(tc.json)); (err == nil) != tc.ok {
			t.Errorf("%s: CheckEntryFile = %v, want ok %v", name, err, tc.ok)
		}
	}
}

// TestManifestHash: a stable value for the same files whatever the map's
// order, and a different one when any file changes, is added or is renamed.
func TestManifestHash(t *testing.T) {
	base := map[string][]byte{"a.json": []byte("1"), "b.json": []byte("2")}
	h := ManifestHash(base)

	if h != ManifestHash(map[string][]byte{"b.json": []byte("2"), "a.json": []byte("1")}) {
		t.Error("the hash depends on map order")
	}

	for name, files := range map[string]map[string][]byte{
		"changed": {"a.json": []byte("1"), "b.json": []byte("3")},
		"added":   {"a.json": []byte("1"), "b.json": []byte("2"), "c.json": nil},
		"renamed": {"a.json": []byte("1"), "c.json": []byte("2")},
	} {
		if ManifestHash(files) == h {
			t.Errorf("%s: same hash", name)
		}
	}
}

func TestParseAllowedRemovals(t *testing.T) {
	lines, err := ParseAllowedRemovals("f", []byte("# header\r\n\nfunction JwtSign   # dropped by the engine\r\nTag\tcfstatic\n  # indented comment\n"))
	if err != nil {
		t.Fatal(err)
	}

	if len(lines) != 2 || lines["function jwtsign"] != "function JwtSign   # dropped by the engine" || lines["tag cfstatic"] != "Tag\tcfstatic" {
		t.Errorf("lines = %q", lines)
	}

	// Each would otherwise accept a loss nobody listed, or refuse one that
	// was, with nothing louder than a warning.
	for name, text := range map[string]string{
		"accept everything": "1\n",
		"no type":           "jwtsign\n",
		"bad type":          "funktion jwtsign\n",
		"three words":       "function jwt sign\n",
		"twice":             "function a\nfunction A\n",
	} {
		if _, err := ParseAllowedRemovals("f", []byte(text)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	if lines, err := ParseAllowedRemovals("f", nil); err != nil || len(lines) != 0 {
		t.Errorf("an empty file = %v, %v; want no lines and no error", lines, err)
	}
}

// TestParseKeyIsTheOneSpellingRule: both DOCS_ALLOW_REMOVALS and the file
// take the same spellings, tabs and doubled spaces included.
func TestParseKeyIsTheOneSpellingRule(t *testing.T) {
	for _, s := range []string{"function foo", "Function  foo", "function\tfoo", " FUNCTION Foo "} {
		if k, ok := ParseKey(s); !ok || k != "function foo" {
			t.Errorf("ParseKey(%q) = %q, %v", s, k, ok)
		}
	}

	if a, _ := ParseAcceptance("function\tfoo,tag  bar"); !a.Accepts("function foo") || !a.Accepts("tag bar") {
		t.Errorf("ParseAcceptance does not follow ParseKey: %v", a.Keys)
	}
}

// TestAddedLines: an unchanged line is left over and accepts nothing; a new
// or changed one (a re-listing with a new reason) is added.
func TestAddedLines(t *testing.T) {
	base := map[string]string{"function a": "function a", "function b": "function b # 2025"}
	now := map[string]string{"function a": "function a", "function b": "function b # again, 2026", "tag c": "tag c"}

	added, left := AddedLines(now, base)
	if len(added) != 2 || !added["function b"] || !added["tag c"] {
		t.Errorf("added = %v, want function b (re-listed) and tag c (new)", added)
	}

	if !slices.Equal(left, []string{"function a"}) {
		t.Errorf("leftOver = %v, want [function a]", left)
	}
}

// TestHashDirRefusesASymlink: os.DirFS follows a symlink out of the folder;
// a vendored file must be the bytes downloaded, not a link elsewhere.
func TestHashDirRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "elsewhere.json")

	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{"name": "a", "type": "function"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(outside, []byte(`{"name": "b", "type": "function"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(outside, filepath.Join(dir, "b.json")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}

	_, broken, err := HashDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(broken) != 1 || !strings.HasPrefix(broken[0], "b.json") {
		t.Errorf("broken = %v, want b.json refused as a symlink", broken)
	}
}
