package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/cfmleditor/clif/internal/cflint"
	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/knownissues"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

const checkUsage = `usage: clif check [options] <path> [...]

Run the checks a project's .clif.json names over the paths given, print what
each found and a summary line per check, and fail if any check found more than
it allows. Made for CI. The config is the one above the first path.

Checks (run in the order named):
  cflint      CFLint, as clif cflint runs it: .cflintrc and @CFLintIgnore
              apply, the pinned CFLint is used, and linting.minSeverity is the
              default floor (the default check)
  unresolved  calls clif cannot resolve, as clif unresolved reports them
  scan        files the grammar cannot parse, as clif scan reports them
  format      files clif format would change. A file the formatter refuses
              is listed on stderr and not counted; scan is the check for it

Configuration ("check" in .clif.json, every key optional):
  "check": {
    "checks": ["cflint", "unresolved", "format"],
    "cflint": {"minSeverity": "WARNING", "maxFindings": 0, "baseline": "..."},
    "unresolved": {"maxFindings": 0, "baseline": "none"},
    "scan": {"maxFindings": 0},
    "format": {"maxFindings": 0}
  }
  maxFindings  how many findings the check may have and still pass (0)
  baseline     a known-issues file, relative to the config, whose entries are
               not counted. cflint and unresolved default to the reports the
               config's knownIssues entries generate for them, as
               clif cflint --write and clif unresolved --write keep them;
               "none" counts every finding
  minSeverity  cflint only: the least severe CFLint level counted

Options:
  --checks <list>     comma-separated checks to run, replacing check.checks
  --format <f>        text (default): findings, then a summary line per check
                      json: {"passed": ..., "checks": [...]} with each finding
                      github: GitHub Actions ::error annotations, then the
                      summary
  --min-severity <l>  the CFLint floor, replacing check.cflint.minSeverity
  --no-baseline       count every finding, whatever the baselines say
  -q, --quiet         no progress on stderr (the unresolved scan still says
                      which config it read and how much it indexes)

Exit status:
  0  every check passed
  1  a check found more than it allows
  2  a check could not run (CFLint unavailable, a .cflintrc CFLint cannot
     parse, a baseline that does not exist), a path that does not exist, or a
     usage error
`

// clif check's exit statuses, as clif cflint's: 1 is the code's findings, 2 a
// run that cannot be trusted, so a broken check never passes for a clean one.
const (
	exitCheckFailed = 1
	exitCheckError  = 2
)

// checkFormats are the --format values.
var checkFormats = []string{"text", "json", "github"}

// checkStatus is one check's outcome.
type checkStatus string

const (
	checkPass  checkStatus = "pass"
	checkFail  checkStatus = "fail"
	checkError checkStatus = "error"
)

// checkFinding is one finding of any check.
type checkFinding struct {
	File     string `json:"file"`
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message"`
	Line     int    `json:"line"`             // 1-based
	Column   int    `json:"column,omitempty"` // 1-based, 0 for none
}

// checkResult is what one check found.
type checkResult struct {
	Name     string         `json:"name"`
	Status   checkStatus    `json:"status"`
	Error    string         `json:"error,omitempty"`
	Findings []checkFinding `json:"findings"`
	// Baselined is how many findings a baseline listed and so were not counted.
	Baselined   int `json:"baselined,omitempty"`
	MaxFindings int `json:"maxFindings"`
	Files       int `json:"files"`
	// Skipped is how many files the check could not judge (format's refusals).
	Skipped int `json:"skipped,omitempty"`
}

// checkFlags is what cmdCheck's arguments ask for.
type checkFlags struct {
	checks      []string
	roots       []string
	format      string
	minSeverity string
	noBaseline  bool
	quiet       bool
}

// checkEnv is what every check runs against.
type checkEnv struct {
	cfg       *daemon.Config // nil with no .clif.json
	configDir string
	rules     config.ResolvedCheck
	fl        *checkFlags
	roots     []string // absolute, each existing
	logf      func(string, ...any)
}

func checkFailf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(exitCheckError)
}

func parseCheckFlags(args []string) checkFlags {
	fl := checkFlags{format: "text"}
	rest := args

	for len(rest) > 0 {
		arg := rest[0]
		rest = rest[1:]
		name, inline, hasInline := strings.Cut(arg, "=")

		value := func() string {
			if hasInline {
				return inline
			}

			if len(rest) == 0 {
				checkFailf("%s needs a value\n\n%s", name, checkUsage)
			}

			v := rest[0]
			rest = rest[1:]

			return v
		}

		switch name {
		case "--checks":
			fl.checks = config.NormalizeChecks(strings.Split(value(), ","))
		case "--format":
			fl.format = value()
		case "--min-severity":
			fl.minSeverity = value()
		case "--no-baseline":
			fl.noBaseline = true
		case "-q", "--quiet":
			fl.quiet = true
		default:
			fl.roots = append(fl.roots, positionalOr(arg, checkUsage, exitCheckError))
		}
	}

	switch {
	case len(fl.roots) == 0:
		fmt.Fprint(os.Stderr, checkUsage)
		os.Exit(exitCheckError)
	case !slices.Contains(checkFormats, fl.format):
		checkFailf("--format %q: want one of %s\n\n%s", fl.format, strings.Join(checkFormats, ", "), checkUsage)
	case fl.checks != nil && len(fl.checks) == 0:
		checkFailf("--checks names no checks; want some of %s\n", strings.Join(config.KnownChecks(), ", "))
	}

	if fl.minSeverity != "" {
		if _, ok := cflint.MinSeverityRank(fl.minSeverity); !ok {
			checkFailf("--min-severity %q is not a CFLint level\n\n%s", fl.minSeverity, checkUsage)
		}
	}

	return fl
}

// cmdCheck runs the selected checks and exits with the worst outcome.
func cmdCheck(args []string) {
	fl := parseCheckFlags(args)
	env := newCheckEnv(&fl)

	checks := env.rules.Checks
	if fl.checks != nil {
		checks = fl.checks
	}

	if unknown := config.UnknownChecks(checks); len(unknown) > 0 {
		checkFailf("no check named %s (known: %s)\n", strings.Join(unknown, ", "), strings.Join(config.KnownChecks(), ", "))
	}

	results := make([]checkResult, 0, len(checks))
	for _, name := range checks {
		results = append(results, env.run(name))
	}

	if err := writeCheckResults(os.Stdout, results, fl.format); err != nil {
		checkFailf("writing the results: %v\n", err)
	}

	if code := checkExitCode(results); code != 0 {
		os.Exit(code)
	}
}

// checkExitCode is the status a run exits with: a check that could not run
// outranks one that failed, since its findings are unknown.
func checkExitCode(results []checkResult) int {
	code := 0

	for i := range results {
		switch results[i].Status {
		case checkError:
			return exitCheckError
		case checkFail:
			code = exitCheckFailed
		case checkPass:
		}
	}

	return code
}

func newCheckEnv(fl *checkFlags) *checkEnv {
	env := &checkEnv{fl: fl, logf: func(format string, args ...any) {
		if !fl.quiet {
			fmt.Fprintf(os.Stderr, format, args...)
		}
	}}

	for _, r := range fl.roots {
		abs, err := filepath.Abs(r)
		if err != nil {
			checkFailf("%s: %v\n", r, err)
		}

		if _, err := os.Stat(abs); err != nil {
			checkFailf("%s does not exist\n", r)
		}

		env.roots = append(env.roots, abs)
	}

	dir := env.roots[0]
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	env.configDir = dir
	env.rules = config.ResolveCheck(nil, dir)

	if cfg, _ := daemon.FindConfig(dir); cfg != nil {
		env.cfg = cfg
		env.configDir = filepath.Dir(cfg.Path)
		env.rules = cfg.ResolvedCheck()
		env.logf("Using config: %s\n", cfg.Path)
	}

	return env
}

// run runs one check and judges it against its threshold.
func (env *checkEnv) run(name string) checkResult {
	rule := env.rules.Rule(name)
	res := checkResult{Name: name, MaxFindings: rule.MaxFindings}

	var err error

	switch name {
	case config.CheckCFLint:
		err = env.runCFLint(&res, &rule)
	case config.CheckUnresolved:
		err = env.runUnresolved(&res, &rule)
	case config.CheckScan:
		env.runScan(&res)
	case config.CheckFormat:
		env.runFormat(&res)
	}

	switch {
	case err != nil:
		res.Status = checkError
		res.Error = err.Error()
	case len(res.Findings) > rule.MaxFindings:
		res.Status = checkFail
	default:
		res.Status = checkPass
	}

	if res.Findings == nil {
		res.Findings = []checkFinding{}
	}

	return res
}

// files is every CFML file under the paths given.
func (env *checkEnv) files() []string {
	return cfpath.CollectCFML(vfs.OS{}, env.roots)
}

func (env *checkEnv) runScan(res *checkResult) {
	files := env.files()
	res.Files = len(files)

	env.logf("scan: parsing %d files...\n", len(files))

	for _, f := range files {
		content, err := vfs.ReadRegular(f)
		if err != nil {
			env.logf("  skip %s: %v\n", f, err)

			res.Skipped++

			continue
		}

		for _, e := range parseErrors(f, content) {
			res.Findings = append(res.Findings, checkFinding{
				File:    e.file,
				Line:    e.line,
				Column:  e.col,
				Rule:    e.lang,
				Message: e.message,
			})
		}
	}
}

func (env *checkEnv) runFormat(res *checkResult) {
	files := env.files()
	res.Files = len(files)
	optionsFor := formatOptionsFor("", false)

	env.logf("format: checking %d files...\n", len(files))

	for _, f := range files {
		content, out, err := formatFile(f, new(optionsFor(f)))
		if err != nil {
			env.logf("  format refused %s: %v\n", f, err)

			res.Skipped++

			continue
		}

		if !bytes.Equal(content, out) {
			res.Findings = append(res.Findings, checkFinding{
				File:    f,
				Line:    firstDifferentLine(content, out),
				Message: "would be reformatted; run clif format -w",
			})
		}
	}
}

// firstDifferentLine is the 1-based line where a and b first differ.
func firstDifferentLine(a, b []byte) int {
	n := min(len(a), len(b))
	line := 1

	for i := range n {
		if a[i] != b[i] {
			return line
		}

		if a[i] == '\n' {
			line++
		}
	}

	return line
}

func (env *checkEnv) runCFLint(res *checkResult, rule *config.ResolvedCheckRule) error {
	scanRoots := env.roots
	minSeverity := ""

	if env.cfg != nil {
		scanRoots = cflint.ScanRoots(env.roots, env.configDir, env.cfg.WorkspaceFolders())
		minSeverity = env.cfg.LintMinSeverity()
	}

	switch {
	case env.fl.minSeverity != "":
		minSeverity = env.fl.minSeverity
	case rule.MinSeverity != "":
		if _, ok := cflint.MinSeverityRank(rule.MinSeverity); !ok {
			return fmt.Errorf("check.cflint.minSeverity %q is not a CFLint level", rule.MinSeverity)
		}

		minSeverity = rule.MinSeverity
	}

	files := cfpath.CollectCFML(vfs.OS{}, scanRoots)
	res.Files = len(files)

	baseline, err := env.baseline(rule, config.GenerateCFLint)
	if err != nil {
		return err
	}

	if err := cflint.CheckConfigs(files); err != nil {
		return err
	}

	runner, err := cflint.NewRunner(context.Background(), minSeverity)
	if err != nil {
		return fmt.Errorf("cflint unavailable: %w", err)
	}

	env.logf("cflint: linting %d files...\n", len(files))

	run, err := runner.Lint(context.Background(), files)
	if err != nil {
		return err
	}

	if baseline != nil {
		res.Baselined, _ = run.Subtract(baseline)
	}

	for i := range run.Issues {
		is := &run.Issues[i]
		loc := firstLocation(&is.Issue)

		res.Findings = append(res.Findings, checkFinding{
			File:     loc.File,
			Line:     max(loc.Line, 1),
			Column:   max(loc.Column, 0),
			Rule:     is.ID,
			Severity: is.Severity,
			Message:  loc.Message,
		})
	}

	return nil
}

// firstLocation is where a CFLint issue is reported.
func firstLocation(issue *cflint.Issue) cflint.Location {
	if len(issue.Locations) == 0 {
		return cflint.Location{Message: issue.Message}
	}

	return issue.Locations[0]
}

func (env *checkEnv) runUnresolved(res *checkResult, rule *config.ResolvedCheckRule) error {
	baseline, err := env.baseline(rule, config.GenerateUnresolved)
	if err != nil {
		return err
	}

	rep, _, _ := scanUnresolved(env.roots, &unresolvedFlags{})
	res.Files = rep.Scanned

	calls := rep.Calls
	if baseline != nil {
		calls, res.Baselined = subtractUnresolved(calls, baseline)
	}

	for i := range calls {
		c := &calls[i]

		res.Findings = append(res.Findings, checkFinding{
			File:    c.File,
			Line:    int(c.Line) + 1,
			Message: unresolvedEntryText(c.CallText(), c.Reason),
		})
	}

	return nil
}

// unresolvedEntryText is the message a known-issues report holds for a call,
// as unresolved.WriteKnownIssues writes it.
func unresolvedEntryText(call, reason string) string {
	return call + " (" + reason + ")"
}

// baseline reads the known-issues entries a check's findings are matched
// against: the rule's file, or the reports the config generates for kind
// that exist. It returns nil for no baseline.
func (env *checkEnv) baseline(rule *config.ResolvedCheckRule, kind string) ([]knownissues.Entry, error) {
	if env.fl.noBaseline || rule.Baseline == config.BaselineNone {
		return nil, nil
	}

	if rule.Baseline != "" {
		entries, err := readBaseline(rule.Baseline)
		if err != nil {
			return nil, fmt.Errorf("baseline: %w", err)
		}

		return entries, nil
	}

	if env.cfg == nil {
		return nil, nil
	}

	var out []knownissues.Entry

	for _, t := range config.GenerateTargets(env.cfg.KnownIssues(), kind, env.configDir) {
		if _, err := os.Stat(t); err != nil {
			continue
		}

		entries, err := readBaseline(t)
		if err != nil {
			return nil, fmt.Errorf("baseline: %w", err)
		}

		env.logf("%s: baseline %s (%d entries)\n", kind, t, len(entries))

		out = append(out, entries...)
	}

	return out, nil
}

func readBaseline(path string) ([]knownissues.Entry, error) {
	data, err := vfs.ReadRegular(path)
	if err != nil {
		return nil, err
	}

	// Never nil, so a baseline that exists and lists nothing is still one.
	return append([]knownissues.Entry{}, knownissues.Parse(string(data), filepath.Dir(path))...), nil
}

// unresolvedKey is what matches a call to a baseline entry: file and text, not
// line, so an edit above a call does not make it new. Each entry matches one
// call.
type unresolvedKey struct{ file, text string }

// subtractUnresolved drops the calls the baseline lists, returning the rest
// and how many it dropped.
func subtractUnresolved(calls []unresolved.Call, baseline []knownissues.Entry) ([]unresolved.Call, int) {
	left := map[unresolvedKey]int{}

	for i := range baseline {
		e := &baseline[i]
		text := e.Message

		if e.Code != "" {
			text = "[" + e.Code + "] " + text
		}

		left[unresolvedKey{filepath.Clean(e.Path), text}]++
	}

	kept := make([]unresolved.Call, 0, len(calls))
	matched := 0

	for i := range calls {
		c := &calls[i]

		k := unresolvedKey{filepath.Clean(c.File), unresolvedEntryText(c.CallText(), c.Reason)}
		if left[k] > 0 {
			left[k]--
			matched++

			continue
		}

		kept = append(kept, *c)
	}

	return kept, matched
}

// writeCheckResults writes the results in the format asked for.
func writeCheckResults(w io.Writer, results []checkResult, format string) error {
	if format == "json" {
		passed := checkExitCode(results) == 0

		return writeJSON(w, struct {
			Checks []checkResult `json:"checks"`
			Passed bool          `json:"passed"`
		}{results, passed})
	}

	for i := range results {
		r := &results[i]

		for j := range r.Findings {
			f := &r.Findings[j]

			if format == "github" {
				_, _ = fmt.Fprintln(w, githubAnnotation(r.Name, f))
			} else {
				_, _ = fmt.Fprintln(w, findingLine(r.Name, f))
			}
		}
	}

	if len(results) > 0 {
		_, _ = fmt.Fprintln(w)
	}

	for i := range results {
		_, _ = fmt.Fprintln(w, summaryLine(&results[i]))
	}

	if checkExitCode(results) == 0 {
		_, _ = fmt.Fprintln(w, "check passed")
	} else {
		_, _ = fmt.Fprintln(w, "check failed")
	}

	return nil
}

// findingLine is path:line[:col]: [check rule] message, the shape a
// known-issues report and an editor's problem matcher read.
func findingLine(check string, f *checkFinding) string {
	pos := strconv.Itoa(f.Line)
	if f.Column > 0 {
		pos += ":" + strconv.Itoa(f.Column)
	}

	tag := strings.TrimSpace(check + " " + f.Rule)

	return fmt.Sprintf("%s:%s: [%s] %s", displayPath(f.File), pos, tag, f.Message)
}

// githubAnnotation is an ::error workflow command for f.
func githubAnnotation(check string, f *checkFinding) string {
	props := "file=" + githubEscape(displayPath(f.File), true) + ",line=" + strconv.Itoa(f.Line)
	if f.Column > 0 {
		props += ",col=" + strconv.Itoa(f.Column)
	}

	title := strings.TrimSpace("clif " + check + " " + f.Rule)

	return "::error " + props + ",title=" + githubEscape(title, true) + "::" + githubEscape(f.Message, false)
}

// githubEscape escapes s for a workflow command's message, or, with prop, for
// one of its properties.
func githubEscape(s string, prop bool) string {
	s = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
	if prop {
		s = strings.NewReplacer(":", "%3A", ",", "%2C").Replace(s)
	}

	return s
}

func summaryLine(r *checkResult) string {
	switch r.Status {
	case checkError:
		return fmt.Sprintf("%-10s  ERROR  %s", r.Name, r.Error)
	case checkFail, checkPass:
	}

	status := "pass "
	if r.Status == checkFail {
		status = "FAIL "
	}

	s := fmt.Sprintf("%-10s  %s  %d findings in %d files", r.Name, status, len(r.Findings), r.Files)
	if r.MaxFindings > 0 {
		s += fmt.Sprintf(", %d allowed", r.MaxFindings)
	}

	if r.Baselined > 0 {
		s += fmt.Sprintf(", %d more in the baseline", r.Baselined)
	}

	if r.Skipped > 0 {
		s += fmt.Sprintf(", %d files skipped", r.Skipped)
	}

	return s
}

// displayPath is p relative to the working directory when it is beneath it,
// with forward slashes, so CI annotations name files as the repository does.
func displayPath(p string) string {
	wd, err := os.Getwd()
	if err != nil || !filepath.IsAbs(p) {
		return filepath.ToSlash(p)
	}

	if rel, ok := relativeBeneath(wd, p); ok {
		return rel
	}

	// The working directory may be reported through a symlink the path was
	// not, as macOS's /var is /private/var.
	rwd, err1 := filepath.EvalSymlinks(wd)
	rp, err2 := filepath.EvalSymlinks(p)

	if err1 == nil && err2 == nil {
		if rel, ok := relativeBeneath(rwd, rp); ok {
			return rel
		}
	}

	return filepath.ToSlash(p)
}

func relativeBeneath(dir, p string) (string, bool) {
	rel, err := filepath.Rel(dir, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}

	return filepath.ToSlash(rel), true
}
