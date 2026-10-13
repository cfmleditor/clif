package unresolved

import (
	"cmp"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Delta is how one run's findings differ from an earlier run's. A count of
// either side alone hides one file breaking while another is fixed, so the
// findings themselves are kept.
type Delta struct {
	Removed []Call // in the earlier run, not in the later
	Added   []Call // in the later run, not in the earlier
	Before  int
	After   int
}

// DiffOptions are the choices a comparison makes.
type DiffOptions struct {
	// Rel shows a path relative to wherever the run was made, and makes two
	// runs of one project on different machines comparable. It defaults to
	// the path as it is.
	Rel func(path string) string
}

func (o *DiffOptions) rel(path string) string {
	if o == nil || o.Rel == nil {
		return path
	}

	return o.Rel(path)
}

var (
	uncheckedRe = regexp.MustCompile(`; \d+ (?:inherited )?calls? not checked`)
	quotedRe    = regexp.MustCompile(`'[^']*'|"[^"]*"`)
	numberRe    = regexp.MustCompile(`\b\d+\b`)
)

// findingKey identifies a finding across runs. A count changed only by a number
// in its reason is not a change: the entry for a broken extends chain says how
// many calls it leaves unchecked, and that moves whenever any of them does.
func (o *DiffOptions) findingKey(c *Call) string {
	return fmt.Sprintf("%s\x00%d\x00%s\x00%s", o.rel(c.File), c.Line, c.Function, uncheckedRe.ReplaceAllString(c.Reason, ""))
}

// Diff compares two runs finding by finding, as multisets: two findings with
// one key are two findings, and one of them going away is a removal.
func Diff(before, after []Call, opt *DiffOptions) Delta {
	d := Delta{Before: len(before), After: len(after)}

	// Counts of the earlier run's findings still unmatched, and where each key's
	// findings sit, so a removal can be reported as the finding it was.
	left := map[string][]int{}

	for i := range before {
		k := opt.findingKey(&before[i])
		left[k] = append(left[k], i)
	}

	for i := range after {
		k := opt.findingKey(&after[i])
		if n := left[k]; len(n) > 0 {
			left[k] = n[1:]

			continue
		}

		d.Added = append(d.Added, after[i])
	}

	for _, idx := range left {
		for _, i := range idx {
			d.Removed = append(d.Removed, before[i])
		}
	}

	byPlace := func(a, b Call) int {
		return cmp.Or(
			cmp.Compare(opt.rel(a.File), opt.rel(b.File)),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Function, b.Function),
			cmp.Compare(a.Reason, b.Reason),
		)
	}

	slices.SortFunc(d.Removed, byPlace)
	slices.SortFunc(d.Added, byPlace)

	return d
}

// ReasonGroup is the findings that share a reason once what is particular to
// each (a name, a number) is taken out.
type ReasonGroup struct {
	Reason string
	Calls  []Call
}

// GroupByReason groups findings by the shape of their reason, largest first:
// "variable 'svc' has no component ref" and "variable 'dao' has no component
// ref" are one cause, and the cause is what a resolution change moves.
func GroupByReason(calls []Call) []ReasonGroup {
	by := map[string]*ReasonGroup{}

	var order []*ReasonGroup

	for i := range calls {
		shape := reasonShape(calls[i].Reason)

		g := by[shape]
		if g == nil {
			g = &ReasonGroup{Reason: shape}
			by[shape] = g
			order = append(order, g)
		}

		g.Calls = append(g.Calls, calls[i])
	}

	out := make([]ReasonGroup, 0, len(order))
	for _, g := range order {
		out = append(out, *g)
	}

	slices.SortStableFunc(out, func(a, b ReasonGroup) int {
		return cmp.Or(cmp.Compare(len(b.Calls), len(a.Calls)), cmp.Compare(a.Reason, b.Reason))
	})

	return out
}

func reasonShape(reason string) string {
	reason = uncheckedRe.ReplaceAllString(reason, "")
	reason = quotedRe.ReplaceAllString(reason, "'_'")

	return numberRe.ReplaceAllString(reason, "N")
}

// WriteDelta prints a comparison: one line of counts, and with show > 0 the
// removed and added findings grouped by reason, each group with up to show
// examples. name labels the line, as a scan name does in a multi-scan report.
func WriteDelta(w io.Writer, name string, d *Delta, show int, opt *DiffOptions) {
	_, _ = fmt.Fprintf(w, "%-10s %6d -> %6d  removed %5d  added %5d\n", name, d.Before, d.After, len(d.Removed), len(d.Added))

	if show <= 0 {
		return
	}

	for _, side := range []struct {
		title string
		calls []Call
	}{{"removed", d.Removed}, {"added", d.Added}} {
		if len(side.calls) == 0 {
			continue
		}

		_, _ = fmt.Fprintf(w, "\n  %s %d:\n", side.title, len(side.calls))

		for _, g := range GroupByReason(side.calls) {
			_, _ = fmt.Fprintf(w, "    %5d  %s\n", len(g.Calls), g.Reason)

			for i := range min(show, len(g.Calls)) {
				c := &g.Calls[i]
				// Lines in a report are 1-based; the JSON is the scanner's own, 0-based.
				_, _ = fmt.Fprintf(w, "           %s:%d  %s\n", opt.rel(c.File), c.Line+1, callLabel(c))
			}

			if more := len(g.Calls) - show; more > 0 {
				_, _ = fmt.Fprintf(w, "           ... and %d more\n", more)
			}
		}
	}

	_, _ = fmt.Fprintln(w)
}

func callLabel(c *Call) string {
	label := c.CallText()
	if c.Caller != "" {
		label += "  (in " + c.Caller + ")"
	}

	return strings.TrimSpace(label)
}

// RelativeTo returns a Rel that shows paths under root without it, and other
// paths as they are.
func RelativeTo(root string) func(string) string {
	root = filepath.Clean(root)

	return func(path string) string {
		if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}

		return path
	}
}
