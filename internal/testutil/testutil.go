// Package testutil holds helpers the packages' tests share. Production code
// does not import it.
package testutil

import (
	"path/filepath"
	"runtime"
)

// Testdata is the repository's testdata directory, or a directory beneath it,
// located from this source file rather than the working directory, which is
// the package under test's.
func Testdata(sub ...string) string {
	_, file, _, _ := runtime.Caller(0)

	return filepath.Join(append([]string{filepath.Dir(file), "..", "..", "testdata"}, sub...)...)
}
