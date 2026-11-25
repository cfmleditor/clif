// Package refs provides shared logic for finding function/component references across files.
package refs

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

// Entry represents a single reference found.
type Entry struct {
	File      string `json:"file"`
	Function  string `json:"function,omitempty"` // enclosing function containing the call/ref
	Variable  string `json:"variable,omitempty"`
	Call      string `json:"call,omitempty"`
	Component string `json:"component,omitempty"`
	Line      uint32 `json:"line"`
	Resolved  bool   `json:"resolved"`
	// Depth and Via describe how this entry relates to the originally traced
	// function: Depth 0 is a direct call to it. Depth N>0 means this call is
	// to Via (a same-named wrapper function found ViaFile away) which itself,
	// N-1 hops further on, reaches the traced function. Set by Trace; zero
	// value for entries returned directly from Find.
	Depth   int    `json:"depth"`
	Via     string `json:"via,omitempty"`
	ViaFile string `json:"viaFile,omitempty"`
	// Reason explains why an unresolved call couldn't be verified against the
	// traced target (e.g. "variable 'x' has no component ref"). Empty when
	// Resolved is true, or when Options.Reason wasn't supplied.
	Reason string `json:"reason,omitempty"`
}

// Options configures what to search for.
type Options struct {
	FuncName   string // function name to find calls to (empty = skip)
	Component  string // component dot-path to find refs to (empty = skip)
	SourceFile string // absolute path of the source file (same-file calls always match)

	// AlsoSources are further files whose calls are references to the same
	// function: an interface and the components implementing it. A call
	// matches when its receiver resolves to SourceFile or to any of them.
	AlsoSources []string

	// Resolver is the workspace's. Every file is parsed as the workspace types
	// it (resolve.Resolver.ParseFile), a call's component is checked to define
	// the function, and, with a SourceFile, to be the one searched for. The
	// editor's find-references and `clif refs` each wired their own closures to
	// do this, and the command line wired none, so the two answered one
	// question differently.
	Resolver *resolve.Resolver

	// Explain gives a call that did not resolve the reason it did not, from the
	// same resolution `unresolved` runs.
	Explain bool
}

// Find scans all CFML files under roots and returns matching references.
func Find(fsys vfs.FS, roots []string, opts *Options) []Entry {
	entries, _ := FindCounted(fsys, roots, opts)

	return entries
}

// FindCounted is [Find] plus the size of the file set it searched.
//
// The count is how many CFML files the walk of roots turned up, which is the
// scope of the search and so what a caller reporting "files scanned" means. It
// is deliberately not the number of files parsed: findInFiles skips parsing any
// file whose text cannot contain the target at all, and reporting that smaller
// number would describe an implementation detail instead of the search.
func FindCounted(fsys vfs.FS, roots []string, opts *Options) ([]Entry, int) {
	files := cfpath.CollectCFML(fsys, roots)

	return findInFiles(fsys, files, opts), len(files)
}

// FindCalls is a convenience wrapper for finding function calls. It returns the
// entries and the number of files searched, as [FindCounted] defines it.
func FindCalls(fsys vfs.FS, roots []string, funcName string, r *resolve.Resolver) ([]Entry, int) {
	return FindCounted(fsys, roots, &Options{FuncName: funcName, Resolver: r})
}

// FindComponentRefs is a convenience wrapper for finding component references.
// It returns the entries and the number of files searched, as [FindCounted]
// defines it.
func FindComponentRefs(fsys vfs.FS, roots []string, component string, r *resolve.Resolver) ([]Entry, int) {
	return FindCounted(fsys, roots, &Options{Component: component, Resolver: r})
}

// FindCallsMany scans all CFML files under roots once and returns matched call
// entries grouped by lowercased function name for every function in funcNames
// declared in sourceFile. When sourceFile is non-empty and not yet on disk
// (such as an unsaved editor buffer served through fsys), it is still included.
//
// also names, by lowercased function name, the further files whose calls count
// as references to that function ([Options.AlsoSources]); nil for none.
func FindCallsMany(fsys vfs.FS, roots []string, funcNames []string, sourceFile string, also map[string][]string, r *resolve.Resolver) map[string][]Entry {
	if len(funcNames) == 0 {
		return nil
	}

	targets := make([]string, 0, len(funcNames))
	seen := make(map[string]struct{}, len(funcNames))

	for _, name := range funcNames {
		lower := strings.ToLower(name)
		if lower == "" {
			continue
		}

		if _, dup := seen[lower]; dup {
			continue
		}

		seen[lower] = struct{}{}

		targets = append(targets, lower)
	}

	if len(targets) == 0 {
		return nil
	}

	files := cfpath.CollectCFML(fsys, roots)
	if sourceFile != "" && !slices.ContainsFunc(files, func(f string) bool { return cfpath.SamePath(f, sourceFile) }) {
		files = append(files, sourceFile)
	}

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make(map[string][]Entry, len(targets))
		sem     = make(chan struct{}, 8)
	)

	for _, f := range files {
		wg.Add(1)

		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			perFile := findManyInFileRecovered(fsys, f, targets, sourceFile, also, r)
			if len(perFile) == 0 {
				return
			}

			mu.Lock()

			for k, entries := range perFile {
				results[k] = append(results[k], entries...)
			}

			mu.Unlock()
		}()
	}

	wg.Wait()

	return results
}

func findManyInFileRecovered(fsys vfs.FS, f string, targets []string, sourceFile string, also map[string][]string, r *resolve.Resolver) (entries map[string][]Entry) {
	defer func() {
		if rec := recover(); rec != nil {
			cflog.Recovered(nil, "reference search panic", rec, cflog.String("file", f))

			entries = nil
		}
	}()

	return findManyInFile(fsys, f, targets, sourceFile, also, r)
}

func findManyInFile(fsys vfs.FS, f string, targets []string, sourceFile string, also map[string][]string, r *resolve.Resolver) map[string][]Entry {
	data, err := fsys.ReadFile(f)
	if err != nil {
		return nil
	}

	data, isText := cfpath.AsText(data)
	if !isText {
		return nil
	}

	var matched []string

	for _, t := range targets {
		if containsFold(data, t) {
			matched = append(matched, t)
		}
	}

	if len(matched) == 0 {
		return nil
	}

	content := string(data)
	spec := resolve.ParseSpec{Calls: true, AllScopes: true, ByResolversOnly: true, FindCalls: matched}
	pr := r.ParseFile(f, content, spec)

	if len(pr.Calls) == 0 {
		return nil
	}

	var contentLines []string

	lines := func() []string {
		if contentLines == nil {
			contentLines = strings.Split(content, "\n")
		}

		return contentLines
	}

	out := make(map[string][]Entry)

	for j := range pr.Calls {
		call := &pr.Calls[j]
		key := strings.ToLower(call.FuncName)

		// Per function: the related files are those of the name called.
		opts := &Options{SourceFile: sourceFile, AlsoSources: also[key], Resolver: r}

		resolved, ok := matchCall(call, pr, f, opts)
		if !ok {
			continue
		}

		out[key] = append(out[key], callEntry(call, pr, f, resolved, lines, opts))
	}

	return out
}

func findInFiles(fsys vfs.FS, files []string, opts *Options) []Entry {
	var mu sync.Mutex

	var results []Entry

	var wg sync.WaitGroup

	sem := make(chan struct{}, 8)

	for _, f := range files {
		wg.Add(1)

		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if entries := findInFileRecovered(fsys, f, opts); len(entries) > 0 {
				mu.Lock()

				results = append(results, entries...)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return results
}

// findInFileRecovered is findInFile, with a panic on the file recorded and
// the file read as holding no references. It runs on a worker goroutine,
// which the caller's recovery does not cover: textDocument/references and
// clif.findRefs search inside the language server, and one bad file ended it.
func findInFileRecovered(fsys vfs.FS, f string, opts *Options) (entries []Entry) {
	defer func() {
		if rec := recover(); rec != nil {
			cflog.Recovered(nil, "reference search panic", rec, cflog.String("file", f))

			entries = nil
		}
	}()

	return findInFile(fsys, f, opts)
}

// findInFile is findInFiles for one file.
func findInFile(fsys vfs.FS, f string, opts *Options) []Entry {
	funcTarget := strings.ToLower(opts.FuncName)
	compTarget := strings.ToLower(opts.Component)

	data, err := fsys.ReadFile(f)
	if err != nil {
		return nil
	}

	data, isText := cfpath.AsText(data)
	if !isText {
		return nil
	}

	// Quick check: skip files that don't contain the function name
	if funcTarget != "" && !containsFold(data, funcTarget) {
		return nil
	}

	content := string(data)
	absPath := f

	// Parsed once, as the workspace types a file, with every scope's calls. A
	// call is typed by the configured resolvers and not by what the called
	// function declares it returns: a reference is to the component the
	// resolver names, not to `any`.
	spec := resolve.ParseSpec{Calls: true, AllScopes: true, ByResolversOnly: true}
	if funcTarget != "" {
		spec.FindCalls = []string{opts.FuncName}
	}

	pr := opts.Resolver.ParseFile(f, content, spec)

	var entries []Entry

	// Component ref matching (from parsed refs — includes all scopes)
	if compTarget != "" {
		for i := range pr.ComponentRefs {
			ref := &pr.ComponentRefs[i]

			if strings.EqualFold(ref.Component, compTarget) {
				entries = append(entries, Entry{
					File: f, Variable: ref.Variable, Line: ref.Line, Resolved: true,
				})
			}
		}
	}

	if funcTarget == "" {
		return entries
	}

	// Call sites don't carry their own source text (CallSite.Text is only
	// populated by the uncached fallback parse path, not the ExtractCalls
	// path used here) — split lines so we can pull the call's line text
	// ourselves for display.
	//
	// Split on first use, not here. Every file reaching this point merely
	// *contains* the name somewhere; most yield no call site at all, and
	// one that does may already carry its own text. Splitting eagerly
	// billed every candidate file for a slice the common case never
	// reads, on a request that walks the whole workspace.
	var contentLines []string

	lines := func() []string {
		if contentLines == nil {
			contentLines = strings.Split(content, "\n")
		}

		return contentLines
	}

	// Function call matching (from parsed call sites — includes all scopes)
	for j := range pr.Calls {
		call := &pr.Calls[j]

		resolved, ok := matchCall(call, pr, absPath, opts)
		if !ok {
			continue
		}

		entries = append(entries, callEntry(call, pr, f, resolved, lines, opts))
	}

	return entries
}

// isSource reports whether path is the file searched for or one of the files
// that count as it (Options.AlsoSources).
func (o *Options) isSource(path string) bool {
	if cfpath.SamePath(path, o.SourceFile) {
		return true
	}

	return slices.ContainsFunc(o.AlsoSources, func(s string) bool { return cfpath.SamePath(path, s) })
}

// componentCalls reports whether a call to funcName on component is a call to
// the function being searched for: the component has it, declared or inherited
// through its extends chain, and is the file searched for or one that counts as
// it (Options.AlsoSources). Inheriting counts: a component that does not
// redeclare the function still answers the call, and whether that call is a
// reference to the function searched for is the file test's to decide. A
// component may be several alternatives, "A|B", when what a receiver holds
// depends on the subclass; any one matching is enough.
func (o *Options) componentCalls(component, funcName, dir string) bool {
	for alt := range strings.SplitSeq(component, "|") {
		if !o.Resolver.HasFunction(alt, funcName, dir) && o.Resolver.ResolveFunc(alt, funcName, dir) == nil {
			continue
		}

		if o.SourceFile == "" || o.isSource(o.Resolver.ComponentPath(alt, dir)) {
			return true
		}
	}

	return false
}

// namesFiles reports whether every alternative of component resolves to a file.
func namesFiles(r *resolve.Resolver, component, dir string) bool {
	for alt := range strings.SplitSeq(component, "|") {
		if r.ComponentPath(alt, dir) == "" {
			return false
		}
	}

	return true
}

// matchCall reports whether call is a reference to the function being
// searched for, and whether it resolved.
func matchCall(call *parser.CallSite, pr *parser.ParseResult, absPath string, opts *Options) (resolved, ok bool) {
	component, typed := call.Component, false

	// A receiver the parse left untyped (a script `var s = new Foo()`, which
	// only the resolver follows from the variable to its declaration) is typed
	// by the same lookup `unresolved` runs. One it cannot type stays a
	// candidate, below.
	if component == "" && call.Variable != "" {
		dir := filepath.Dir(absPath)

		// Only a type that names files: a WireBox id (`authorService@contentbox`)
		// resolves the call but is no file to compare with the target.
		if c := opts.Resolver.ComponentOf(call.Variable, call.Line, pr, dir); c != "" && namesFiles(opts.Resolver, c, dir) {
			component, typed = c, true
		}
	}

	if component != "" && !opts.componentCalls(component, call.FuncName, filepath.Dir(absPath)) {
		return false, false
	}

	if call.Resolved || call.Component != "" {
		return call.Resolved, true
	}

	if typed {
		return true, true
	}

	// Check if the function exists in the same file
	sameFile := false

	for i := range pr.Funcs {
		fn := &pr.Funcs[i]

		if strings.EqualFold(fn.Name, call.FuncName) {
			sameFile = true

			break
		}
	}

	if sameFile {
		// Same-file call — only a match if this IS the source file
		if !opts.isSource(absPath) {
			return false, false
		}

		return true, true
	}

	// Truly unqualified, function not in this file — skip
	return false, call.Variable != ""
}

// callEntry is the report entry for a matched call.
func callEntry(call *parser.CallSite, pr *parser.ParseResult, f string, resolved bool, lines func() []string, opts *Options) Entry {
	callText := call.Text
	if callText == "" {
		if src := lines(); int(call.Line) < len(src) {
			callText = strings.TrimSpace(strings.TrimSuffix(src[call.Line], "\r"))
		}
	}

	reason := ""
	if !resolved && opts.Explain {
		reason = opts.Resolver.CanResolveCall(call, pr, filepath.Dir(f))
		if reason != "" {
			// Restate the call target so the reason reads standalone —
			// readers scanning one line at a time (without the group
			// header above it) can otherwise lose track of which
			// function/hop this failure was checked against.
			target := call.FuncName
			if call.Variable != "" {
				target = call.Variable + "." + call.FuncName
			}

			reason = fmt.Sprintf("%s (needed for %s())", reason, target)
		}
	}

	return Entry{
		File: f, Function: call.Caller, Call: callText,
		Component: call.Component, Variable: call.Variable,
		Line: call.Line, Resolved: resolved, Reason: reason,
	}
}

// containsFold reports whether data contains target (case-insensitive, ASCII only).
// Does not allocate.
func containsFold(data []byte, target string) bool {
	tLen := len(target)
	if tLen == 0 {
		return true
	}

	end := len(data) - tLen
	for i := 0; i <= end; i++ {
		match := true

		for j := range tLen {
			if data[i+j]|0x20 != target[j]|0x20 {
				match = false

				break
			}
		}

		if match {
			return true
		}
	}

	return false
}
