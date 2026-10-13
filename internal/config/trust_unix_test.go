//go:build unix

package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMode writes a config file and sets its mode outright, past the umask.
func writeMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()

	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

// fakeOwner makes every file appear owned by uid and gid for the test.
func fakeOwner(t *testing.T, uid, gid int64) {
	t.Helper()

	prev := fileOwner
	fileOwner = func(fs.FileInfo) (int64, int64, bool) { return uid, gid, true }

	t.Cleanup(func() { fileOwner = prev })
}

// TestAConfigOthersCanWriteIsSkipped: a world-writable or group-writable
// config file is left out, and so is the reason given to the reporter.
func TestAConfigOthersCanWriteIsSkipped(t *testing.T) {
	prevPrivate := privateGroup
	privateGroup = func(int64) bool { return false }

	t.Cleanup(func() { privateGroup = prevPrivate })

	for _, tc := range []struct {
		mode os.FileMode
		want string
	}{
		{0o666, "other users"},
		{0o602, "other users"},
		{0o664, "its group"},
	} {
		dir := t.TempDir()
		writeMode(t, filepath.Join(dir, FileName), tc.mode)

		var reasons []string

		got := TrustedCandidates(dir, os.Stat, func(_, why string) { reasons = append(reasons, why) })
		if len(got) != 0 {
			t.Errorf("mode %04o: read %v", tc.mode, got)
		}

		if len(reasons) != 1 || !strings.Contains(reasons[0], tc.want) {
			t.Errorf("mode %04o: reasons %q, want one naming %q", tc.mode, reasons, tc.want)
		}
	}
}

// TestAConfigOwnedByAnotherUserIsSkipped: a file another uid owns is left
// out, one root owns is read, and the user's own is read.
func TestAConfigOwnedByAnotherUserIsSkipped(t *testing.T) {
	dir := t.TempDir()
	writeMode(t, filepath.Join(dir, FileName), 0o644)

	me := int64(os.Getuid())

	for _, tc := range []struct {
		uid  int64
		read bool
	}{
		{me + 1, false},
		{0, true},
		{me, true},
	} {
		fakeOwner(t, tc.uid, int64(os.Getgid()))

		if got := len(TrustedCandidates(dir, os.Stat, nil)) == 1; got != tc.read {
			t.Errorf("owner uid %d: read %v, want %v", tc.uid, got, tc.read)
		}
	}
}

// TestAUsersPrivateGroupMayWrite: the umask 002 a Linux account with a
// private group gets makes a checkout group-writable, and that is the user's
// own file. A group other users share is not.
func TestAUsersPrivateGroupMayWrite(t *testing.T) {
	dir := t.TempDir()
	writeMode(t, filepath.Join(dir, FileName), 0o664)
	fakeOwner(t, int64(os.Getuid()), int64(os.Getgid()))

	prev := privateGroup

	t.Cleanup(func() { privateGroup = prev })

	privateGroup = func(int64) bool { return true }

	if len(TrustedCandidates(dir, os.Stat, nil)) != 1 {
		t.Error("a file writable by the user's private group was skipped")
	}

	privateGroup = func(int64) bool { return false }

	if len(TrustedCandidates(dir, os.Stat, nil)) != 0 {
		t.Error("a file writable by a shared group was read")
	}
}

// TestCandidatesInReportsWhatItSkips: callers with no logger of their own
// still have the skip reported, through ReportUntrustedTo.
func TestCandidatesInReportsWhatItSkips(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LegacyFileName)
	writeMode(t, path, 0o666)

	var got []string

	ReportUntrustedTo(func(p, _ string) { got = append(got, p) })
	t.Cleanup(func() { ReportUntrustedTo(nil) })

	if c := CandidatesIn(dir); len(c) != 0 {
		t.Errorf("read %v", c)
	}

	if len(got) != 1 || got[0] != path {
		t.Errorf("reported %v, want %q", got, path)
	}
}
