package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/confine"
	"github.com/cfmleditor/clif/internal/daemon"
	"github.com/cfmleditor/clif/internal/frameworkapi"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/unresolved"
	"github.com/cfmleditor/clif/internal/vfs"
)

const unresolvedUsage = `usage: clif unresolved [options] <dir> [...]

Report every component or method call that does not resolve: a receiver with
no known component, a method the component does not declare, a component that
does not exist. Reads the .clif.json above the first directory.

  --json               print the results as JSON
  --known-issues       print them as a known-issues report, paths relative to
                       the config's directory
  --relative-to <dir>  with --known-issues, make paths relative to this instead
  --include-workspace  with --known-issues, keep entries outside that
                       directory, as ../ paths
  --write              write the report files the config's knownIssues entries
                       with "generate": "unresolved" name
  --verbose            also list each call that resolved, on stderr
  --global-defs        resolve bare calls against functions declared anywhere
                       in the workspace
  --candidates         give each finding a category (object, variable,
                       return-type, method) and the liberal matches a strict
                       resolver refuses, each with a confidence; they never
                       resolve it. Shown with --json
  --speculative        opt in to unreliable sole-candidate receiver connections;
                       --verbose lists proposed guesses (off by default)
  --no-speculative     explicitly disable speculation; the last flag wins
  --speculative-native guess a native family, then a primitive family fallback
  --no-speculative-native explicitly disable native family guessing
  --speculative-methods link calls directly to ranked global source declarations
  --no-speculative-methods disable direct method shortcuts (off by default)
  --speculative-ranked guess the unique best candidate matching receiver roles
  --no-speculative-ranked disable ranked guesses; ties remain unresolved
  --assume-open-world classify unknown receivers/returns, absent components and
                       bare runtime mixins as assumptions, not resolved calls;
                       --json emits {calls, assumedCalls, ...} with audit evidence
  --no-assume-open-world restore ordinary diagnostic reporting
  --native-engine <name> runtime API profile: auto (default), generic or lucee
  --no-infer-args      do not type an untyped argument from what its callers
                       pass it

Optional batch limits are set with callerInferenceLimits in .clif.json.
They are unlimited by default; see README.md for the policy keys.

To see why one call resolved as it did: clif explain <file> <line>
`

// unresolvedFlags is what cmdUnresolved's flags ask for.
type unresolvedFlags struct {
	speculativeMethods bool
	speculativeRanked  bool
	assumeOpenWorld    bool
	relativeTo         string
	nativeEngine       string
	json               bool
	knownIssues        bool
	write              bool
	includeWorkspace   bool
	verbose            bool
	globalDefs         bool
	candidates         bool
	noInferArgs        bool
	speculative        bool
	speculativeNative  bool
}

// parseUnresolvedFlags splits the flags from the files and directories.
func parseUnresolvedFlags(args []string) (unresolvedFlags, []string) {
	var (
		fl    unresolvedFlags
		paths []string
	)

	for i := 0; i < len(args); i++ {
		a := args[i]

		if v, ok := strings.CutPrefix(a, "--relative-to="); ok {
			fl.relativeTo = v

			continue
		}

		if v, ok := strings.CutPrefix(a, "--native-engine="); ok {
			fl.nativeEngine = nativeEngineFlag(v)

			continue
		}

		if a == "--native-engine" && i+1 < len(args) {
			i++
			fl.nativeEngine = nativeEngineFlag(args[i])

			continue
		}

		if a == "--relative-to" && i+1 < len(args) {
			i++
			fl.relativeTo = args[i]

			continue
		}

		switch a {
		case "--json":
			fl.json = true
		case "--known-issues":
			fl.knownIssues = true
		case "--write":
			fl.write = true
		case "--include-workspace":
			fl.includeWorkspace = true
		case "--verbose":
			fl.verbose = true
		case "--global-defs":
			fl.globalDefs = true
		case "--speculative-methods":
			fl.speculativeMethods = true
		case "--no-speculative-methods":
			fl.speculativeMethods = false
		case "--candidates":
			fl.candidates = true
		case "--speculative":
			fl.speculative = true
		case "--speculative-ranked":
			fl.speculativeRanked = true
		case "--no-speculative-ranked":
			fl.speculativeRanked = false
		case "--assume-open-world":
			fl.assumeOpenWorld = true
		case "--no-assume-open-world":
			fl.assumeOpenWorld = false
		case "--speculative-native":
			fl.speculativeNative = true
		case "--no-speculative-native":
			fl.speculativeNative = false
		case "--no-speculative":
			fl.speculative = false
		case "--no-infer-args":
			fl.noInferArgs = true
		default:
			paths = append(paths, positional(a, unresolvedUsage))
		}
	}

	return fl, paths
}

func cmdUnresolved(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, unresolvedUsage)
		os.Exit(1)
	}

	fl, args := parseUnresolvedFlags(args)
	if (fl.assumeOpenWorld || fl.speculativeMethods) && (fl.write || fl.knownIssues) {
		fmt.Fprintln(os.Stderr, "--assume-open-world and --speculative-methods need an audit report; use --json instead of --write or --known-issues")
		os.Exit(1)
	}

	if len(args) == 0 {
		fmt.Fprint(os.Stderr, unresolvedUsage)
		os.Exit(1)
	}

	rep, cfg, searchDir := scanUnresolved(args, &fl)
	if (fl.assumeOpenWorld || fl.speculativeMethods) && fl.json && !fl.write {
		if err := writeOpenWorldJSON(os.Stdout, &rep); err != nil {
			fmt.Fprintf(os.Stderr, "writing JSON: %v\n", err)
			os.Exit(1)
		}
	} else if !writeUnresolved(rep.Calls, cfg, args, searchDir, &fl) {
		if fl.speculativeMethods {
			fmt.Fprintf(os.Stderr, "%d provisional method connections retained in audit (not resolved). Use --json to review.\n", len(rep.SpeculativeMethodConnections))
		}

		if fl.assumeOpenWorld {
			fmt.Fprintf(os.Stderr, "%d assumed calls retained in open-world audit (not resolved). Use --json to review.\n", len(rep.AssumedCalls))
		}

		return
	}

	fmt.Fprintf(os.Stderr, "%d unresolved calls found (%d resolved)\n", len(rep.Calls), rep.Resolved)

	if fl.speculativeMethods {
		fmt.Fprintf(os.Stderr, "Method shortcuts: %d provisional declaration connections, separate from resolved calls. Use --json to review targets and original diagnostics.\n", len(rep.SpeculativeMethodConnections))
	}

	if fl.speculative || fl.speculativeNative || fl.speculativeRanked {
		fmt.Fprintf(os.Stderr, "Speculative mode: %d proposed receiver connections; reductions are not verified fixes. Use --verbose to review.\n", len(rep.SpeculativeConnections))
	}

	if fl.assumeOpenWorld {
		fmt.Fprintf(os.Stderr, "Open-world mode: %d assumed calls, separate from %d resolved calls. Assumptions can conceal defects; use --json to review original diagnostics.\n", len(rep.AssumedCalls), rep.Resolved)
	}

	if bases := unresolved.MissingBases(rep.Calls); len(bases) > 0 {
		fmt.Fprintf(os.Stderr, "\nBase components that do not resolve (a mapping or workspace path for each checks its calls):\n")

		for _, b := range bases {
			fmt.Fprintf(os.Stderr, "  %-50s %5d files %7d calls\n", b.Component, b.Files, b.Calls)
		}
	}

	if hint := presetHint(cfg, searchDir); hint != "" {
		fmt.Fprintf(os.Stderr, "\n%s\n", hint)
	}

	fmt.Fprintf(os.Stderr, "\nBenchmark:\n")
	fmt.Fprintf(os.Stderr, "  Index:  %v (%d files)\n", rep.IndexTime, rep.Indexed)
	fmt.Fprintf(os.Stderr, "  Scan:   %v (%d files)\n", rep.ScanTime, rep.Scanned)
	fmt.Fprintf(os.Stderr, "  Total:  %v\n", rep.IndexTime+rep.ScanTime)
}

func writeOpenWorldJSON(out io.Writer, rep *unresolved.Report) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")

	audit := struct {
		Calls        []unresolved.Call                        `json:"calls"`
		AssumedCalls []unresolved.AssumedCall                 `json:"assumedCalls"`
		Connections  []unresolved.SpeculativeConnection       `json:"speculativeConnections"`
		Methods      []unresolved.SpeculativeMethodConnection `json:"speculativeMethodConnections,omitempty"`
		Resolved     int                                      `json:"resolved"`
		Indexed      int                                      `json:"indexed"`
		Scanned      int                                      `json:"scanned"`
	}{Calls: rep.Calls, AssumedCalls: rep.AssumedCalls, Connections: rep.SpeculativeConnections, Methods: rep.SpeculativeMethodConnections, Resolved: rep.Resolved, Indexed: rep.Indexed, Scanned: rep.Scanned}

	return enc.Encode(audit)
}

// scanUnresolved runs the scan the unresolved command reports: it indexes the
// workspace the config above the first path names, and checks the calls in the
// paths given. The MCP server's find_unresolved_calls runs it too.
func scanUnresolved(args []string, fl *unresolvedFlags) (unresolved.Report, *daemon.Config, string) {
	fsys := vfs.OS{}

	// Find .clif.json config based on the first file/dir argument
	searchDir, _ := filepath.Abs(args[0])
	if info, err := os.Stat(searchDir); err == nil && !info.IsDir() {
		searchDir = filepath.Dir(searchDir)
	}

	cfg, _ := daemon.FindConfig(searchDir)
	opt := unresolvedOptions(cfg, args, fl)

	// Collect files from workspace folders or args
	scanRoots := args

	configuredRoots := cfg != nil && len(cfg.WorkspaceFolders()) > 0
	if configuredRoots {
		scanRoots = opt.WorkspaceFolders
	}

	files := collectCFMLFiles(fsys, scanRoots)

	scanFiles := scanTargets(fsys, args, configuredRoots)

	fmt.Fprintf(os.Stderr, "Indexing %d files, then scanning for unresolved calls...\n", len(files))

	return unresolved.Scan(fsys, files, scanFiles, opt), cfg, searchDir
}

// unresolvedOptions builds the scan's options from the config, or from the
// paths given when there is none.
func unresolvedOptions(cfg *daemon.Config, args []string, fl *unresolvedFlags) *unresolved.Options {
	opt := &unresolved.Options{SpeculativeMethods: fl.speculativeMethods, SpeculativeRanked: fl.speculativeRanked, AssumeOpenWorld: fl.assumeOpenWorld, SpeculativeNative: fl.speculativeNative, NativeEngine: fl.nativeEngine, Speculative: fl.speculative, GlobalDefs: fl.globalDefs, Candidates: fl.candidates, InferArgs: !fl.noInferArgs, WorkspaceFolders: cliWorkspaceFolders(vfs.OS{}, cfg, args)}
	if fl.verbose {
		opt.Verbose = os.Stderr
	}

	if cfg != nil {
		opt.CallerInferenceLimits = cfg.CallerInferenceLimits()
		opt.Mappings = cfg.Mappings()
		opt.StartupFiles = cfg.StartupFiles()
		opt.ExpressionMappings = cfg.ExpressionMappings()
		opt.ServicePropertyResolvers = cfg.ServicePropertyResolvers()
		opt.BeanPaths = cfg.BeanPaths()
		opt.PropertyResolvers = configPropertyResolvers(cfg)
		opt.ImplicitExtends = config.ImplicitExtends(cfg.Frameworks())
		opt.HelperScope = config.HelperScope(cfg.Frameworks())
		opt.Stubs = frameworkapi.For(cfg.Frameworks())
		opt.InterpolateAll = !cfg.ResolvedFeatures().OutputContextInterpolation

		if unknown := config.UnknownFrameworks(cfg.Frameworks()); len(unknown) > 0 {
			fmt.Fprintf(os.Stderr, "frameworks: no preset named %s (known: %s)\n",
				strings.Join(unknown, ", "), strings.Join(config.KnownFrameworks(), ", "))
		}

		for _, r := range cfg.ComponentResolvers() {
			opt.Resolvers = append(opt.Resolvers, r.Parser())
		}

		fmt.Fprintf(os.Stderr, "Using config: %s\n", cfg.Path)

		return opt
	}

	if fl.write {
		fmt.Fprintf(os.Stderr, "--write needs a .clif.json to say where the report goes\n")
		os.Exit(1)
	}

	assumed := config.AssumedFrameworks(opt.WorkspaceFolders, vfs.OS{}.ReadFile, vfs.OS{}.ReadDir)
	for _, rule := range config.FrameworkResolvers(assumed) {
		opt.Resolvers = append(opt.Resolvers, rule.Parser())
	}

	opt.ImplicitExtends = config.ImplicitExtends(assumed)
	opt.HelperScope = config.HelperScope(assumed)

	opt.Stubs = frameworkapi.For(assumed)
	if len(assumed) > 0 {
		fmt.Fprintf(os.Stderr, "Assuming framework context from project source: %s (override with .clif.json)\n", strings.Join(assumed, ", "))
	}

	return opt
}

// writeUnresolved writes the results the way the flags ask, and reports
// whether the summary should follow: an empty result says so and stops.
func writeUnresolved(results []unresolved.Call, cfg *daemon.Config, args []string, searchDir string, fl *unresolvedFlags) bool {
	switch {
	case fl.write:
		configDir := filepath.Dir(cfg.Path)
		targets := config.GenerateTargets(cfg.KnownIssues(), config.GenerateUnresolved, configDir)
		roots := reportRoots(args, configDir)

		for _, t := range targets {
			if _, err := confine.Check(t, roots); err != nil {
				fmt.Fprintf(os.Stderr, "--write: %v\n", err)
				os.Exit(1)
			}
		}

		byTarget, rest := unresolved.SplitByTarget(results, targets)
		for _, t := range targets {
			n, err := writeReport(t, byTarget[t], unresolved.RegenerateHint, roots)
			if err != nil {
				fmt.Fprintf(os.Stderr, "could not write %s: %v\n", t, err)
				os.Exit(1)
			}

			fmt.Fprintf(os.Stderr, "Wrote %s (%d entries)\n", t, n)
		}

		if len(rest) > 0 {
			fmt.Fprintf(os.Stderr, "%d entries under no report's directory left out\n", len(rest))
		}
	case len(results) == 0:
		fmt.Fprintf(os.Stderr, "No unresolved calls found.\n")

		return false
	case fl.json:
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		if err := enc.Encode(results); err != nil {
			fmt.Fprintf(os.Stderr, "writing JSON: %v\n", err)
			os.Exit(1)
		}
	case fl.knownIssues:
		baseDir, _ := filepath.Abs(searchDir)
		if cfg != nil {
			baseDir = filepath.Dir(cfg.Path)
		}

		if fl.relativeTo != "" {
			if abs, err := filepath.Abs(fl.relativeTo); err == nil {
				baseDir = abs
			}
		}

		regenerate := "clif unresolved --known-issues " + strings.Join(args, " ")
		if skipped := unresolved.WriteKnownIssues(os.Stdout, results, baseDir, fl.includeWorkspace, regenerate, version); skipped > 0 {
			fmt.Fprintf(os.Stderr, "%d entries outside %s left out; --include-workspace writes them as ../ paths\n", skipped, baseDir)
		}
	default:
		for i := range results {
			r := &results[i]

			fmt.Printf("%s:%d: %s (%s)\n", r.File, r.Line+1, r.CallText(), r.Reason)
		}
	}

	return true
}

// writeReport writes calls to path as a known-issues file relative to its own
// directory, and returns how many entries it holds. The write stays under
// roots after symlinks (reportRoots).
func writeReport(path string, calls []unresolved.Call, regenerate string, roots []string) (int, error) {
	var b strings.Builder

	skipped := unresolved.WriteKnownIssues(&b, calls, filepath.Dir(path), false, regenerate, version)

	// Owner-only, as the server writes the same report for .exportUnresolved.
	return len(calls) - skipped, confine.WriteFile(path, []byte(b.String()), 0o600, roots)
}

// reportRoots is where --write may put a report named in the config: under a
// directory the command was asked to scan (a file's own directory, for a file)
// or the config's directory, after symlinks. The config names the report, and
// it comes with the repository, so a cloned project's .clif.json naming
// "../../.zshrc", or a committed symlink at the default report's name, would
// otherwise have the command overwrite that file.
func reportRoots(args []string, configDir string) []string {
	dirs := make([]string, 0, 1+len(args))
	dirs = append(dirs, configDir)

	for _, a := range args {
		if info, err := os.Stat(a); err == nil && !info.IsDir() {
			a = filepath.Dir(a)
		}

		dirs = append(dirs, a)
	}

	return confine.Roots(dirs...)
}

func collectCFMLFiles(fsys vfs.FS, roots []string) []string {
	var files []string

	for _, root := range roots {
		_ = fsys.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				if path != root && skipDir(info.Name()) {
					return filepath.SkipDir
				}

				return nil
			}

			if cfpath.IsCFMLFile(path) {
				files = append(files, path)
			}

			return nil
		})
	}

	return files
}

// skipDir reports whether a directory is one no scan should descend into.
//
// Every dot-directory is skipped, not a hand-kept list of them. The list was
// ".git", ".svn", "node_modules", "target" and "vendor", and on a real workspace
// it let in ".claude/worktrees" — git worktrees living inside the project, each a
// complete second copy of the codebase. A whole-project map built over that counts
// every component twice, reports every function as having a mysterious duplicate,
// and inflates the unreferenced list with thousands of entries from a checkout
// nobody is editing. The same applies to any other tool's dot-directory, which is
// why this is a rule rather than another name on a list.
//
// The root itself is exempt, so pointing a scan at a dot-directory still works.
func skipDir(name string) bool {
	if strings.HasPrefix(name, ".") && name != "." && name != ".." {
		return true
	}

	switch name {
	case "node_modules", "target", "vendor", "bower_components":
		return true
	default:
		return false
	}
}

// presetHint suggests the framework presets the project's box.json implies
// and its config does not name — how a project finds out a preset exists.
// It reads box.json beside the config, or in the scanned directory without one.
func presetHint(cfg *daemon.Config, searchDir string) string {
	dir, have := searchDir, []string(nil)
	if cfg != nil {
		dir, have = filepath.Dir(cfg.Path), cfg.Frameworks()
	}

	data, err := vfs.ReadRegular(filepath.Join(dir, "box.json"))
	if err != nil {
		return ""
	}

	suggest := config.SuggestFrameworks(data, have)
	if len(suggest) == 0 {
		return ""
	}

	all := slices.Concat(have, suggest)

	where := "a .clif.json"
	if cfg != nil {
		where = cfg.Path
	}

	return fmt.Sprintf("box.json names %s, which have framework presets: add `\"frameworks\": [\"%s\"]` to %s\n"+
		"so the values those frameworks hand your code are typed rather than reported.",
		strings.Join(suggest, ", "), strings.Join(all, `", "`), where)
}

// scanTargets is the files a report covers, as absolute paths; nil covers
// everything indexed. Files are always named. A directory narrows the report
// only when the config's workspace folders set the index wider than the
// arguments: the index must stay whole, since resolving a call reads every
// other component, but the report should hold what was asked for. Without
// workspace folders the arguments are the whole index and nothing is narrowed.
func scanTargets(fsys vfs.FS, args []string, indexWiderThanArgs bool) []string {
	abs := make([]string, 0, len(args))

	for _, a := range args {
		p, err := filepath.Abs(a)
		if err != nil {
			p = a
		}

		abs = append(abs, p)
	}

	if indexWiderThanArgs {
		return collectCFMLFiles(fsys, abs)
	}

	var files []string

	for _, p := range abs {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			files = append(files, p)
		}
	}

	return files
}

func nativeEngineFlag(value string) string {
	switch value {
	case "auto":
		return ""
	case "generic", "lucee":
		return value
	default:
		fmt.Fprintln(os.Stderr, "native-engine must be auto, generic or lucee")
		os.Exit(1)

		return ""
	}
}
