// Package unresolved finds the calls the resolver cannot follow to a
// definition, and writes them as a known-issues file. The `clif
// unresolved` command and the server's clif.exportUnresolved command
// both run it, so the report is the same whichever wrote it.
package unresolved

import (
	"cmp"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/frameworkapi"
	"github.com/cfmleditor/clif/internal/index"
	"github.com/cfmleditor/clif/internal/knownissues"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/resolve"
	"github.com/cfmleditor/clif/internal/vfs"
)

// Call is one call the resolver could not follow.
type Call struct {
	This     bool   `json:"this,omitempty"` // explicit this.method() must not become an assumed mixin
	File     string `json:"file"`
	Line     uint32 `json:"line"`
	Caller   string `json:"caller,omitempty"`
	Variable string `json:"variable,omitempty"`
	Function string `json:"function"`
	Reason   string `json:"reason"`
	Text     string `json:"text"`
	// Unchecked is, on the one entry for a file whose extends chain breaks,
	// how many inherited calls that leaves unchecked. Function holds the
	// base that does not resolve.
	Unchecked int `json:"unchecked,omitempty"`
	// Category, Candidates and CandidateCount are set by a scan with
	// Options.Candidates: which link is missing, and the liberal matches
	// offered for it (candidates.go). The finding stands either way.
	Category       string      `json:"category,omitempty"`
	Candidates     []Candidate `json:"candidates,omitempty"`
	CandidateCount int         `json:"candidateCount,omitempty"`
	// Definitions counts indexed declarations of the reported function name,
	// including stubs. Zero does not establish absence at runtime; a match
	// does not establish receiver ownership. Chains may fail on another name.
	Definitions *int `json:"definitions,omitempty"`
	// MethodEvidence distinguishes name availability from the missing link.
	MethodEvidence *MethodEvidence `json:"methodEvidence,omitempty"`
	methodTargets  []Candidate
}

// Options is what a scan resolves with: the .clif.json settings that
// shape parsing and resolution.
type Options struct {
	SpeculativeMethods  bool // opt-in direct global declaration shortcuts
	methodTargets       bool
	SpeculativeRanked   bool // opt-in unique best receiver-role candidate
	AssumeOpenWorld     bool // classify unproven runtime connections separately
	receiverRoles       bool
	unlimitedCandidates bool
	SpeculativeNative   bool // opt-in native family guesses, then primitive-family fallback

	NativeEngine string // auto source-layout detection, or explicit generic/lucee

	Speculative          bool // opt-in sole-candidate receiver guesses; not source-proven
	speculativeReceivers map[resolve.SpeculativeReceiver]string

	CallerInferenceLimits    config.CallerInferenceLimits
	Resolvers                []parser.Resolver
	Mappings                 map[string]string
	StartupFiles             []string
	ExpressionMappings       map[string]string
	ServicePropertyResolvers map[string]string
	PropertyResolvers        []parser.PropertyResolver
	BeanPaths                map[string]string // configured beanPaths; Application.cfc's are added
	WorkspaceFolders         []string
	ImplicitExtends          func(path string) string // frameworks' implicit bases; see config.ImplicitExtends
	HelperScope              func(path string) bool   // files frameworks mix helpers into; see config.HelperScope
	Stubs                    *frameworkapi.Set        // frameworks' API when their source is absent; see internal/frameworkapi
	InterpolateAll           bool                     // features.outputContextInterpolation off
	GlobalDefs               bool                     // accept a bare call any indexed file defines
	Candidates               bool                     // annotate each finding with its category and liberal matches; see candidates.go
	InferArgs                bool                     // type an untyped argument from what every caller passes it (resolve.Resolver.InferArgsFiles)
	Verbose                  io.Writer
}

// Settings is the resolution half of the options, as the batch resolver takes it.
func (o *Options) Settings() *resolve.Settings {
	return &resolve.Settings{
		NativeEngine:             o.NativeEngine,
		SpeculativeReceivers:     o.speculativeReceivers,
		CallerInferenceLimits:    o.CallerInferenceLimits,
		Resolvers:                o.Resolvers,
		Mappings:                 o.Mappings,
		StartupFiles:             o.StartupFiles,
		ExpressionMappings:       o.ExpressionMappings,
		PropertyResolvers:        o.PropertyResolvers,
		ServicePropertyResolvers: o.ServicePropertyResolvers,
		BeanPaths:                o.BeanPaths,
		WorkspaceFolders:         o.WorkspaceFolders,
		ImplicitExtends:          o.ImplicitExtends,
		HelperScope:              o.HelperScope,
		Stubs:                    o.Stubs,
		InterpolateAllText:       o.InterpolateAll,
	}
}

// UseSettings takes the resolution half of the options from s. The native
// engine is a command-line choice, not the configuration's, and is left alone.
func (o *Options) UseSettings(s *resolve.Settings) {
	o.CallerInferenceLimits = s.CallerInferenceLimits
	o.Resolvers = s.Resolvers
	o.Mappings = s.Mappings
	o.StartupFiles = s.StartupFiles
	o.ExpressionMappings = s.ExpressionMappings
	o.PropertyResolvers = s.PropertyResolvers
	o.ServicePropertyResolvers = s.ServicePropertyResolvers
	o.BeanPaths = s.BeanPaths
	o.WorkspaceFolders = s.WorkspaceFolders
	o.ImplicitExtends = s.ImplicitExtends
	o.HelperScope = s.HelperScope
	o.Stubs = s.Stubs
	o.InterpolateAll = s.InterpolateAllText
}

// Report is a scan's result.
type Report struct {
	SpeculativeMethodConnections []SpeculativeMethodConnection
	AssumedCalls                 []AssumedCall
	SpeculativeConnections       []SpeculativeConnection

	Calls     []Call
	Resolved  int
	Indexed   int
	Scanned   int
	IndexTime time.Duration
	ScanTime  time.Duration
}

// Scan indexes files, then reports the unresolved calls in targets, or in
// every one of files when targets is empty.
func Scan(fsys vfs.FS, files, targets []string, opt *Options) Report {
	// A batch scan assumes its input tree is stable until it completes. Reuse
	// metadata across indexing, resolution and any speculative passes.
	fsys = vfs.CacheMetadata(fsys)

	if opt.SpeculativeMethods {
		closed := *opt
		closed.SpeculativeMethods = false
		closed.AssumeOpenWorld = false
		closed.Candidates = true
		closed.methodTargets = true
		rep := Scan(fsys, files, targets, &closed)
		classifyMethodShortcuts(&rep, opt.Verbose)

		if opt.AssumeOpenWorld {
			classifyOpenWorld(&rep, opt.Verbose)
		}

		return rep
	}

	if opt.AssumeOpenWorld {
		closed := *opt
		closed.AssumeOpenWorld = false
		rep := Scan(fsys, files, targets, &closed)
		classifyOpenWorld(&rep, opt.Verbose)

		return rep
	}

	if opt.Speculative || opt.SpeculativeNative || opt.SpeculativeRanked {
		return scanSpeculative(fsys, files, targets, opt)
	}

	started := time.Now()
	resolver := NewResolver(fsys, files, opt)
	rep := Report{Indexed: len(files), IndexTime: time.Since(started)}

	if len(targets) == 0 {
		targets = files
	}

	rep.Scanned = len(targets)
	started = time.Now()

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	sem := make(chan struct{}, runtime.GOMAXPROCS(0))

	for _, f := range targets {
		wg.Add(1)

		sem <- struct{}{}

		go func(file string) {
			defer wg.Done()
			defer func() { <-sem }()

			calls, resolved := scanFile(fsys, resolver, file, opt)

			mu.Lock()

			rep.Calls = append(rep.Calls, calls...)
			rep.Resolved += resolved
			mu.Unlock()
		}(f)
	}

	wg.Wait()

	if opt.Candidates {
		annotateAll(fsys, resolver, files, rep.Calls, opt)
	}

	rep.ScanTime = time.Since(started)

	// Every field takes part, so two calls on one line come out in the same
	// order whichever goroutine finished first: a report is diffed against
	// an earlier one.
	sort.Slice(rep.Calls, func(i, j int) bool {
		a, b := &rep.Calls[i], &rep.Calls[j]

		return cmp.Or(
			cmp.Compare(a.File, b.File),
			cmp.Compare(a.Line, b.Line),
			cmp.Compare(a.Variable, b.Variable),
			cmp.Compare(a.Function, b.Function),
			cmp.Compare(a.Reason, b.Reason),
			cmp.Compare(a.Caller, b.Caller),
			cmp.Compare(a.Text, b.Text),
			cmp.Compare(a.Unchecked, b.Unchecked),
		) < 0
	})

	return rep
}

// NewResolver is the resolver a scan checks calls with, its index built from
// files. The explain command builds its resolver here too, so a trace and the
// strict report it explains cannot disagree: explain used to build its own, without
// beanPaths or the setter and constructor policies, and rejected calls the
// report accepted. Speculative batch scans add their own fallback snapshot;
// explain retains strict behavior.
func NewResolver(fsys vfs.FS, files []string, opt *Options) *resolve.Resolver {
	settings := opt.Settings()
	if opt.InferArgs {
		settings.InferArgsFiles = files
	}

	resolver := resolve.NewBatchResolver(fsys, index.New(), settings)
	// Discovery lazily indexes factory metadata with the original rules.
	// The scan must index every return using the augmented rules.
	resolver.Index = index.New()
	resolver.LoadBeans(opt.BeanPaths)

	resolver.IndexFiles(files, nil)

	return resolver
}

func scanFile(fsys vfs.FS, resolver *resolve.Resolver, file string, opt *Options) (out []Call, resolved int) {
	content, err := vfs.ReadString(fsys, file)
	if err != nil || cfpath.IsBinary(content) {
		return nil, 0
	}

	baseDir := filepath.Dir(file)
	pr := Parse(resolver, file, content)
	calls := pr.AllCalls()

	// Calls into a base that does not resolve are one finding per file and
	// base, not one per call: see resolve.MissingBaseReason. The base is the
	// file's own, for an inherited call, or a receiver's, for a call on a
	// component whose chain breaks.
	var bases baseGroups

	for i := range calls {
		call := &calls[i]

		c := resolver.ClassifyCall(call, pr, baseDir, resolve.ClassifyOptions{GlobalDefs: opt.GlobalDefs})

		switch c.Verdict {
		case resolve.Accepted:
			resolved++

			if opt.Verbose != nil {
				_, _ = fmt.Fprintf(opt.Verbose, "  ✓ %s:%d: %s.%s\n", filepath.Base(file), call.Line+1, call.Variable, call.FuncName)
			}

			continue
		case resolve.Builtin:
			continue
		case resolve.GlobalDef:
			resolved++

			if opt.Verbose != nil {
				_, _ = fmt.Fprintf(opt.Verbose, "  ✓ %s:%d: %s (global def)\n", filepath.Base(file), call.Line+1, call.FuncName)
			}

			continue
		case resolve.MissingBase:
			bases.add(c.Base, call)

			continue
		case resolve.Unresolved:
		}

		reason := c.Reason

		entry := Call{
			This:     call.This,
			File:     file,
			Line:     call.Line,
			Caller:   call.Caller,
			Variable: call.Variable,
			Function: call.FuncName,
			Reason:   reason,
			Text:     call.Text,
		}

		out = append(out, entry)
	}

	if len(bases.order) > 0 {
		own := ""
		if pr.Extends != "" {
			own = resolver.MissingBase(pr.Extends, baseDir)
		}

		for _, g := range bases.order {
			entry := receiverBaseCall(file, g)
			if strings.EqualFold(g.base, own) {
				entry = missingBaseCall(file, content, pr.Extends, g.base, g.calls)
			}

			if opt.Candidates {
				entry.Category = CategoryObject
			}

			out = append(out, entry)
		}
	}

	return out, resolved
}

// baseGroup is the calls in one file left unchecked by one missing base, and
// the first of them.
type baseGroup struct {
	first *parser.CallSite
	base  string
	calls int
}

// baseGroups collects baseGroup by base, in the order each base is first met.
type baseGroups struct {
	by    map[string]*baseGroup
	order []*baseGroup
}

func (b *baseGroups) add(base string, call *parser.CallSite) {
	key := strings.ToLower(base)
	if g, ok := b.by[key]; ok {
		g.calls++

		return
	}

	if b.by == nil {
		b.by = map[string]*baseGroup{}
	}

	g := &baseGroup{base: base, calls: 1, first: call}
	b.by[key] = g
	b.order = append(b.order, g)
}

// receiverBaseCall is the one entry for the calls in a file made on
// components whose extends chain breaks at a base that is not the file's own:
// ContentBox's services extend cborm's VirtualEntityService, and with cborm
// missing every findWhere and save on one is unchecked. It sits on the first
// such call.
func receiverBaseCall(file string, g *baseGroup) Call {
	return Call{
		File:      file,
		Line:      g.first.Line,
		Caller:    g.first.Caller,
		Function:  g.base,
		Reason:    fmt.Sprintf("calls a component whose chain breaks at %s, which does not resolve; %d %s not checked", g.base, g.calls, plural(g.calls, "call", "calls")),
		Text:      g.first.Text,
		Unchecked: g.calls,
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}

	return many
}

// missingBaseCall is the one entry for a file whose extends chain breaks at
// base, on the line of the file's own extends, which is where the chain
// starts: base may be further up it, in a component the file never names.
func missingBaseCall(file, content, extends, base string, unchecked int) Call {
	line := uint32(0)
	lowerExtends := strings.ToLower(extends)

	for i, l := range strings.Split(content, "\n") {
		if l = strings.ToLower(l); strings.Contains(l, "extends") && strings.Contains(l, lowerExtends) {
			line = conv.Uint32(i)

			break
		}
	}

	noun := plural(unchecked, "call", "calls")

	reason := fmt.Sprintf("base component does not resolve; %d inherited %s not checked", unchecked, noun)
	if !strings.EqualFold(extends, base) {
		reason = fmt.Sprintf("extends %s, whose chain breaks at %s, which does not resolve; %d inherited %s not checked", extends, base, unchecked, noun)
	}

	return Call{
		File:      file,
		Line:      line,
		Function:  base,
		Reason:    reason,
		Text:      "extends " + extends,
		Unchecked: unchecked,
	}
}

// MissingBase is one base component that does not resolve, across a report.
type MissingBase struct {
	Component string
	Files     int
	Calls     int
}

// MissingBases totals the report's missing bases, most calls first. Each is
// usually one mapping or workspace path away from checking every call it
// accounts for.
func MissingBases(calls []Call) []MissingBase {
	by := map[string]*MissingBase{}

	for i := range calls {
		c := &calls[i]
		if c.Unchecked == 0 {
			continue
		}

		key := strings.ToLower(c.Function)
		if by[key] == nil {
			by[key] = &MissingBase{Component: c.Function}
		}

		by[key].Files++
		by[key].Calls += c.Unchecked
	}

	out := make([]MissingBase, 0, len(by))
	for _, m := range by {
		out = append(out, *m)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Calls != out[j].Calls {
			return out[i].Calls > out[j].Calls
		}

		return out[i].Component < out[j].Component
	})

	return out
}

// CallText is how a call is written in the report: variable.function.
func (c *Call) CallText() string {
	if c.Variable != "" {
		return c.Variable + "." + c.Function
	}

	return c.Function
}

// RegenerateHint is the report header's how-to-regenerate line when it is
// written to its configured files. It is the same whichever writes it, so a
// report regenerated from the editor and from the command line diffs clean.
const RegenerateHint = "clif unresolved --write <project>, or the editor's clif.exportUnresolved command"

// WriteKnownIssues writes calls as a known-issues file: the unresolved line
// format with paths relative to baseDir, sorted by path and line, under a #
// header saying how to regenerate and use it. Listed under knownIssues in a
// .clif.json, the server publishes each entry as a diagnostic.
//
// A path is never written absolute: the file is committed to one project and
// read on other machines, where it would not name the same file. A call
// outside baseDir, a workspacePaths neighbour, is left out and counted, unless
// includeWorkspace asks for it, when it is written as a ../ path the way
// workspacePaths name it. One that cannot be made relative at all, on another
// volume, is always left out.
//
// regenerate is the header's how-to-regenerate line. The header carries no
// date, so regenerating an unchanged report changes nothing a diff would show.
func WriteKnownIssues(w io.Writer, calls []Call, baseDir string, includeWorkspace bool, regenerate, version string) (skipped int) {
	type row struct {
		path string
		line int
		text string
	}

	rows := make([]row, 0, len(calls))

	for i := range calls {
		c := &calls[i]

		rel, ok := relativePath(baseDir, c.File)
		if !ok || (isOutside(rel) && !includeWorkspace) {
			skipped++

			continue
		}

		rows = append(rows, row{path: filepath.ToSlash(rel), line: int(c.Line) + 1, text: c.CallText() + " (" + c.Reason + ")"})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].path != rows[j].path {
			return rows[i].path < rows[j].path
		}

		if rows[i].line != rows[j].line {
			return rows[i].line < rows[j].line
		}

		return rows[i].text < rows[j].text
	})

	_, _ = fmt.Fprintf(w, "# Calls clif cannot resolve, one per line: path:line: call (reason).\n")
	_, _ = fmt.Fprintf(w, "# Paths are relative to this file's directory. List the file under knownIssues in\n")
	_, _ = fmt.Fprintf(w, "# .clif.json to show the entries as editor diagnostics. Regenerate with:\n")
	_, _ = fmt.Fprintf(w, "#   %s\n", regenerate)
	_, _ = fmt.Fprintf(w, "# %d entries; clif %s.\n", len(rows), version)

	for _, r := range rows {
		_, _ = fmt.Fprintf(w, "%s:%d: %s\n", r.path, r.line, r.text)
	}

	return skipped
}

// SplitByTarget assigns each call to the target file whose directory holds it,
// the deepest when directories nest, for writing one report per target. A
// call under none of them is returned in rest.
func SplitByTarget(calls []Call, targets []string) (byTarget map[string][]Call, rest []Call) {
	byTarget = make(map[string][]Call, len(targets))
	for _, t := range targets {
		byTarget[t] = nil
	}

	for i := range calls {
		c := &calls[i]

		best := knownissues.DeepestTarget(c.File, targets)
		if best == "" {
			rest = append(rest, *c)

			continue
		}

		byTarget[best] = append(byTarget[best], *c)
	}

	return byTarget, rest
}

func relativePath(base, file string) (string, bool) {
	rel, err := filepath.Rel(base, file)
	if err != nil || filepath.IsAbs(rel) {
		return "", false
	}

	return rel, true
}

func isOutside(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Parse is file parsed as a scan checks it: with the resolver's lookups, so
// what a variable is assigned from is typed the same way in both commands.
func Parse(resolver *resolve.Resolver, file, content string) *parser.ParseResult {
	return resolver.ParseFile(file, content, resolve.BatchSpec())
}
