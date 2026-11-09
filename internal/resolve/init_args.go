package resolve

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// A component's constructor argument is what every construction passes it.
// cfwheels' CLI services are built as `new services.Templates( helpers = h )`
// and keep it as `variables.helpers = arguments.helpers;`, and ColdBox's
// BoxLangProvider makes its stats with `new BoxLangStats( this )`. Nothing in
// the component says what the argument is, and every call on the variable was
// "has no component ref".
//
// inferInitArgument is argumentFromCallers for init: its callers are found by
// the component's file name, since `init(` is in every file. A `new X( … )`
// whose path resolves to the component is a construction, as is an `init()`
// call the existing caller check places on it (createObject( … ).init( … )).
// Any other mention of the name in a string (getInstance( "X" ),
// createObject without an init on its line) is a construction that cannot be
// read, and leaves the argument untyped.
//
// initArgMember carries the answer into the component: a variables.x whose
// only assignments are `variables.x = arguments.p` inside init holds what p is
// given.

func (r *Resolver) inferInitArgument(fd *parser.FunctionDef, pos int, file string, ctx lookupCtx) string {
	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	files := r.callerFiles(base)
	for _, f := range r.callerFiles(quotedCallerKey(base)) {
		if !slices.Contains(files, f) {
			files = append(files, f)
		}
	}

	if len(files) == 0 || exceedsCallerLimit(len(files), r.CallerInferenceLimits.MaxConstructorFiles) {
		return ""
	}

	acc := &initArgSites{r: r, fd: fd, pos: pos, file: file, base: base, argName: fd.Arguments[pos].Name, ctx: ctx}

	for _, path := range files {
		if !acc.scan(path) {
			return ""
		}
	}

	if acc.sites == 0 {
		return ""
	}

	slices.Sort(acc.comps)

	return strings.Join(acc.comps, "|")
}

// initArgSites gathers what the constructions of one component pass one init
// argument, file by file.
type initArgSites struct {
	r          *Resolver
	fd         *parser.FunctionDef
	ctx        lookupCtx
	file, base string
	argName    string
	comps      []string
	pos, sites int
}

// take records what one construction passes, expr read at line in caller of
// hpr. It reports false when the construction cannot be typed.
func (a *initArgSites) take(expr []parser.Token, line uint32, caller string, hpr *parser.ParseResult, dir string) bool {
	a.sites++
	if exceedsCallerLimit(a.sites, a.r.CallerInferenceLimits.MaxCallSites) {
		return false
	}

	comp := a.r.argumentExprComponent(expr, line, caller, hpr, dir, a.ctx)
	if comp == "" && forwardedArg(expr) {
		a.sites--

		return true
	}

	if comp == "" {
		return false
	}

	for alt := range strings.SplitSeq(a.r.pathsOf(comp, dir), "|") {
		if !lex.ContainsFold(a.comps, alt) {
			a.comps = append(a.comps, alt)
		}
	}

	return true
}

// scan reads the constructions in the file at path. It reports false when
// one cannot be read or typed.
func (a *initArgSites) scan(path string) bool {
	hpr := a.r.handlerParse(path)
	if hpr == nil {
		return false
	}

	dir := filepath.Dir(path)
	tokens := allTokens(hpr.Content)
	read := map[int]bool{}

	for _, site := range a.r.constructions(tokens, a.base, a.file, dir) {
		read[site.line] = true

		expr, passed := callArgumentAt(tokens, site.at, a.pos, a.argName)
		if passed && !a.take(expr, conv.Uint32(site.line), hpr.FuncScopeAt(site.line).Name, hpr, dir) {
			return false
		}
	}

	calls := hpr.AllCalls()
	for ci := range calls {
		call := &calls[ci]
		if !strings.EqualFold(call.FuncName, "init") {
			continue
		}

		if ours, known := a.r.callIsTo(call, hpr, path, a.fd, a.file, dir, a.ctx); !ours || !known {
			continue
		}

		read[int(call.Line)] = true

		expr, passed := callSiteArgument(tokens, call, a.pos, a.argName)
		if passed && !a.take(expr, call.Line, call.Caller, hpr, dir) {
			return false
		}
	}

	for _, line := range quotedNameLines(hpr.Content, a.base) {
		if !read[line] {
			return false
		}
	}

	return true
}

type construction struct {
	at   int // the index of the component's name token
	line int
}

// constructions are the `new a.b.Name(` sites in tokens whose path, read from
// dir, is file.
func (r *Resolver) constructions(tokens []parser.Token, base, file, dir string) []construction {
	var out []construction

	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].Kind != parser.TokIdent || !strings.EqualFold(tokens[i].Value, "new") {
			continue
		}

		j := i + 1

		var dotted strings.Builder

		for j < len(tokens) && tokens[j].Kind == parser.TokIdent {
			dotted.WriteString(tokens[j].Value)

			if j+1 < len(tokens) && tokens[j+1].Kind == parser.TokDot {
				dotted.WriteByte('.')

				j += 2

				continue
			}

			break
		}

		if j+1 >= len(tokens) || tokens[j+1].Kind != parser.TokLParen || !strings.EqualFold(tokens[j].Value, base) {
			continue
		}

		if p := r.ComponentPath(dotted.String(), dir); p != "" && cfpath.SamePath(p, file) {
			out = append(out, construction{at: j, line: int(tokens[j].Line)})
		}
	}

	return out
}

// quotedNameLines are the lines on which name ends a quoted string.
func quotedNameLines(content, name string) []int {
	if !mentions(content, name) {
		return nil
	}

	var starts []int
	if plainName(name) {
		starts = quotedNameStarts(content, name)
	} else {
		starts = quotedNameStartsRegex(content, name)
	}

	// Lines are counted on from the last match, not from the top each time.
	out := make([]int, 0, len(starts))
	line, from := 0, 0

	for _, start := range starts {
		line += strings.Count(content[from:start], "\n")
		from = start

		out = append(out, line)
	}

	return out
}

// quotedNameStartsRegex is quotedNameStarts as the expression it replaces, for a
// name the scanner does not take, and the reference it is tested against.
func quotedNameStartsRegex(content, name string) []int {
	re := compiledPattern(`(?i)\b` + regexp.QuoteMeta(name) + `["']`)
	matches := re.FindAllStringIndex(content, -1)
	starts := make([]int, 0, len(matches))

	for _, m := range matches {
		starts = append(starts, m[0])
	}

	return starts
}

var (
	initArgAssignRe = regexp.MustCompile(`(?i)^\s*(?:variables\.)?(\w+)\s*=\s*arguments\.(\w+)\s*;?\s*$`)
	initArgSetRe    = regexp.MustCompile(`(?i)^\s*(?:this\.|variables\.)?set(\w+)\(\s*arguments\.(\w+)\s*\)\s*;?\s*$`)
)

// initArgMember is what variables.name holds when the file stores only an
// init argument in it (initStoredArg): what every construction passes.
func (r *Resolver) initArgMember(variable string, pr *parser.ParseResult, ctx lookupCtx) string {
	name, ok := strings.CutPrefix(strings.ToLower(variable), "variables.")
	if !ok || name == "" || strings.ContainsAny(name, ".[(") || !pr.URI.IsFile() {
		return ""
	}

	return r.initArgType(pr, initStoredArg(pr, name), ctx)
}

// initArgGetter is what the generated getter def returns when its property is
// only ever stored from an init argument: BoxLangStats' init does
// `setCacheProvider( arguments.cacheProvider )` and reads it back through
// getCacheProvider(). A getter the file writes itself is not generated.
func (r *Resolver) initArgGetter(def *parser.FunctionDef) string {
	if def == nil || !def.URI.IsFile() || len(def.Name) <= 3 || !strings.EqualFold(def.Name[:3], "get") {
		return ""
	}

	pr := r.handlerParse(def.URI.Path())
	if pr == nil {
		return ""
	}

	// A getter the index holds and the source never writes is the one
	// generated for a property.
	name := def.Name[3:]
	if hasWrittenFunc(pr, def.Name) {
		return ""
	}

	return r.initArgType(pr, initStoredArg(pr, name), lookupCtx{})
}

// initArgType is what init's argument param holds: its declared component
// type (BoxLangStats documents its provider's), else what every construction
// passes it.
func (r *Resolver) initArgType(pr *parser.ParseResult, param string, ctx lookupCtx) string {
	if param == "" {
		return ""
	}

	if t := r.initArgDeclared(pr, param); t != "" {
		return t
	}

	return r.argumentFromCallers("arguments."+param, "init", pr, ctx)
}

// initArgDeclared is the component init's argument param declares: a dotted
// type, or a bare name read as namedType reads one. Blueprint's init takes
// `required SchemaBuilder schemaBuilder`.
func (r *Resolver) initArgDeclared(pr *parser.ParseResult, param string) string {
	arg := argumentOf(pr, "init", param)
	if param == "" || arg == nil {
		return ""
	}

	if strings.Contains(arg.Type, ".") {
		return arg.Type
	}

	return r.bareArgumentComponent(arg.Type, pr)
}

// lineWritesNameRegex is lineWritesName as the expression it replaces, for a name
// the scanner does not take, and the reference it is tested against.
func lineWritesNameRegex(line, name string) bool {
	q := regexp.QuoteMeta(name)

	return compiledPattern(`(?i)(?:(?:^|[^\w.])(?:variables\.)?` + q + `\s*=[^=])|(?:\bset` + q + `\s*\()`).MatchString(line)
}

// hasWrittenFunc reports whether pr's source declares a function called name,
// as opposed to one generated for a property.
func hasWrittenFunc(pr *parser.ParseResult, name string) bool {
	if !mentions(pr.Content, name) {
		return false
	}

	if plainName(name) {
		return writesFunction(pr.Content, name)
	}

	return writesFunctionRegex(pr.Content, name)
}

// writesFunctionRegex is writesFunction as the expression it replaces, for a name
// the scanner does not take, and the reference the scanner is tested against.
func writesFunctionRegex(content, name string) bool {
	return compiledPattern(`(?i)\bfunction\s+` + regexp.QuoteMeta(name) + `\s*\(|<cffunction[^>]+name\s*=\s*["']` + regexp.QuoteMeta(name) + `["']`).MatchString(content)
}

// initStoredArg is the init argument p when every write to name in pr is
// `variables.name = arguments.p` or `setName( arguments.p )` inside init, and
// "" otherwise.
func initStoredArg(pr *parser.ParseResult, name string) string {
	scope, ok := funcScope(pr, "init")
	if !ok {
		return ""
	}

	// Every write the expression below matches names the variable.
	if !mentions(pr.Content, name) {
		return ""
	}

	plain := plainName(name)
	write := func(line string) bool { return lineWritesNameRegex(line, name) }

	if plain {
		write = func(line string) bool { return lineWritesName(line, name) }
	}

	param := ""

	i := -1

	for line := range strings.Lines(pr.Content) {
		i++
		line = strings.TrimSuffix(line, "\n")

		if !mentions(line, name) || !write(line) {
			continue
		}

		m := initArgAssignRe.FindStringSubmatch(line)
		if m == nil {
			m = initArgSetRe.FindStringSubmatch(line)
		}

		if m == nil || !strings.EqualFold(m[1], name) || i < scope.Start || i > scope.End || param != "" && !strings.EqualFold(param, m[2]) {
			return ""
		}

		param = m[2]
	}

	return param
}
