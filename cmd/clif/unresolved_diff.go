package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

const unresolvedDiffUsage = `usage: clif unresolved-diff [--show <n>] [--root <dir>] <before.json> <after.json>

Compare two runs of "clif unresolved --json" finding by finding, so a change to
resolution can be judged by what it did to each call and not by a total: a
change that breaks one file and fixes another leaves the total where it was.

  --show <n>     list the removed and added findings grouped by reason, up to
                 n examples a group (default 5; 0 prints the counts line alone)
  --root <dir>   show paths relative to this directory, and compare them that
                 way, so runs made in two checkouts of one project line up

The reports are the "--json" output of any two builds over the same project:

  old/clif unresolved --json ~/src/app > before.json
  new/clif unresolved --json ~/src/app > after.json
  clif unresolved-diff --root ~/src/app before.json after.json

A finding is the same in both when its file, line, function and reason are; a
count inside a reason ("12 inherited calls not checked") is not a difference.
Exit status is 0 whatever the result: the output is for a person to read.
`

func cmdUnresolvedDiff(args []string) {
	show := 5

	var (
		root  string
		files []string
	)

	// want names the option whose value is the next argument.
	var want string

	for _, a := range args {
		switch want {
		case "--show":
			n, err := strconv.Atoi(a)
			if err != nil || n < 0 {
				fatalf("--show needs a number of examples, not %q\n", a)
			}

			show, want = n, ""

			continue
		case "--root":
			root, want = a, ""

			continue
		}

		switch a {
		case "--show", "--root":
			want = a
		default:
			files = append(files, positional(a, unresolvedDiffUsage))
		}
	}

	if want != "" {
		fatalf("%s needs a value\n%s", want, unresolvedDiffUsage)
	}

	if len(files) != 2 {
		fmt.Fprint(os.Stderr, unresolvedDiffUsage)
		os.Exit(1)
	}

	before, after := readUnresolvedReport(files[0]), readUnresolvedReport(files[1])

	opt := &unresolved.DiffOptions{}
	if root != "" {
		opt.Rel = unresolved.RelativeTo(root)
	}

	delta := unresolved.Diff(before, after, opt)
	unresolved.WriteDelta(os.Stdout, "findings", &delta, show, opt)
}

// readUnresolvedReport reads the JSON "clif unresolved --json" writes.
func readUnresolvedReport(path string) []unresolved.Call {
	data, err := vfs.ReadRegular(path)
	if err != nil {
		fatalf("%v\n", err)
	}

	var calls []unresolved.Call
	if err := json.Unmarshal(data, &calls); err != nil {
		fatalf("%s is not the output of clif unresolved --json: %v\n", path, err)
	}

	return calls
}
