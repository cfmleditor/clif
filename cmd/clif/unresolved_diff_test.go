package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnresolvedDiffNamesWhatMoved(t *testing.T) {
	dir := t.TempDir()
	before := filepath.Join(dir, "before.json")
	after := filepath.Join(dir, "after.json")

	write := func(path, body string) {
		t.Helper()

		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	root := filepath.Join(dir, "app")
	write(before, `[{"file":"`+root+`/a.cfc","line":1,"function":"save","variable":"svc","reason":"variable 'svc' has no component ref","text":""},
{"file":"`+root+`/b.cfc","line":2,"function":"load","variable":"svc","reason":"variable 'svc' has no component ref","text":""}]`)
	write(after, `[{"file":"`+root+`/b.cfc","line":2,"function":"load","variable":"svc","reason":"variable 'svc' has no component ref","text":""},
{"file":"`+root+`/c.cfc","line":4,"function":"drop","variable":"dao","reason":"method 'drop' not found in Dao","text":""}]`)

	out := captureStdout(t, func() { cmdUnresolvedDiff([]string{"--root", root, before, after}) })

	for _, want := range []string{"2 ->      2", "removed     1  added     1", "a.cfc:2  svc.save", "c.cfc:5  dao.drop", "variable '_' has no component ref"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}

	if strings.Contains(out, root) {
		t.Errorf("--root should make the paths relative:\n%s", out)
	}

	out = captureStdout(t, func() { cmdUnresolvedDiff([]string{"--show", "0", before, after}) })
	if strings.Count(out, "\n") != 1 {
		t.Errorf("--show 0 should print the counts line alone:\n%s", out)
	}
}
