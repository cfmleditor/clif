package config

import (
	"io/fs"
	"path/filepath"
	"sync/atomic"
)

// TrustedCandidates is every config file in dir that exists and may be read,
// in the order FileNames gives. stat is the filesystem's stat (os.Stat, or a
// vfs.FS's), and skipped, when not nil, is told about each file left out and
// why; when nil, the reporter set by ReportUntrustedTo is.
//
// Every walk that looks for a config file goes through here, because config
// discovery walks up to the filesystem root: a .clif.json in /tmp or
// /Users/Shared governs every project opened below it, and another local user
// can put one there. On unix a file is read only when it is owned by the
// current user or root and no other user can write it (Untrusted). A file
// left out does not end the walk; the next directory up is still searched.
// TestEveryConfigLookupChecksOwnership holds every caller to this.
func TrustedCandidates(dir string, stat func(string) (fs.FileInfo, error), skipped func(path, reason string)) []string {
	if skipped == nil {
		if r := untrustedReporter.Load(); r != nil {
			skipped = *r
		}
	}

	out := make([]string, 0, len(FileNames))

	for _, n := range FileNames {
		p := filepath.Join(dir, n)

		info, err := stat(p)
		if err != nil || info.IsDir() {
			continue
		}

		if why := Untrusted(info); why != "" {
			if skipped != nil {
				skipped(p, why)
			}

			continue
		}

		out = append(out, p)
	}

	return out
}

var untrustedReporter atomic.Pointer[func(path, reason string)]

// ReportUntrustedTo sets what CandidatesIn tells about a config file it leaves
// out, for the callers that have no logger of their own to pass. nil reports
// nothing.
func ReportUntrustedTo(f func(path, reason string)) {
	if f == nil {
		untrustedReporter.Store(nil)

		return
	}

	untrustedReporter.Store(&f)
}
