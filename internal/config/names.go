package config

import (
	"os"
	"path/filepath"
	"slices"
)

// FileName is the project config file. LegacyFileName is what it was called
// while the project was cfmleditor-lsp. Both are read, the new name first, so
// no project has to rename its config; every new file is written under the new
// name.
const (
	FileName       = ".clif.json"
	LegacyFileName = ".cfmleditor.json"
)

// FileNames are the config file's names, in the order a directory is searched.
var FileNames = []string{FileName, LegacyFileName}

// IsFileName reports whether base, a file's base name, names a config file.
func IsFileName(base string) bool {
	return slices.Contains(FileNames, base)
}

// CandidatesIn returns the config files in dir that exist and may be read, in
// the order to try them: a directory holding both is configured by .clif.json.
// A file another local user could have written is left out (see
// TrustedCandidates); the reporter set by ReportUntrustedTo is told why.
func CandidatesIn(dir string) []string {
	return TrustedCandidates(dir, os.Stat, nil)
}

// defaultGenerated is where each kind is written when no knownIssues entry is
// marked to receive it, beside the config file: the current name first, then
// legacy names in preference order. See defaultReport.
var defaultGenerated = map[string][]string{
	GenerateUnresolved: {".clif-unresolved.txt", ".cfmleditor-unresolved.txt"},
	GenerateLint:       {".clif-lint.txt", ".clif-cflint.txt", ".cfmleditor-cflint.txt"},
}

// defaultReport is a kind's implicit report file in dir: the first existing
// legacy file when none of the current name exists, so a project that
// committed its report under an old name keeps it, and the current name
// otherwise. "" for a kind with no default.
func defaultReport(dir, kind string) string {
	names, ok := defaultGenerated[normalizeGenerateKind(kind)]
	if !ok || len(names) == 0 {
		return ""
	}

	current := filepath.Join(dir, names[0])
	if exists(current) {
		return current
	}

	for _, legacyName := range names[1:] {
		legacy := filepath.Join(dir, legacyName)
		if exists(legacy) {
			return legacy
		}
	}

	return current
}

// isDefaultReport reports the kind whose default file, under either name, is
// path in dir.
func isDefaultReport(path, dir string) (string, bool) {
	for kind, names := range defaultGenerated {
		for _, n := range names {
			if path == filepath.Join(dir, n) {
				return kind, true
			}
		}
	}

	return "", false
}

func exists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}
