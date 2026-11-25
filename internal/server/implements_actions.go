package server

import (
	"path/filepath"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type missingIfaceMethod struct {
	fn        *parser.FunctionDef
	ifaceName string
}

// implementsActions offers refactor code actions that insert stub
// implementations for interface methods declared in a CFC's `implements`
// attribute (including transitive parent interfaces) that are not yet declared
// on the component or inherited from its `extends` chain.
func (s *Server) implementsActions(docURI uri.URI, content string, at protocol.Range) []protocol.CodeAction {
	file := cfpath.FromURI(string(docURI))
	if !strings.EqualFold(filepath.Ext(file), ".cfc") {
		return nil
	}

	ifaces := s.implementsForOpenDoc(docURI, content)
	if len(ifaces) == 0 {
		return nil
	}

	// Asked on every cursor move. The declaration test is bounded; finding the
	// component's end is a scan of the whole body, so it is made only where the
	// closer can be: on the declaration, or on lines that hold a closing
	// token at all.
	onHeader := cursorInDeclaration(content, int(at.Start.Line))
	if !onHeader && !rangeMayHoldComponentClose(content, int(at.Start.Line), int(at.End.Line)) {
		return nil
	}

	end, tags, ok := parser.ComponentBodyEnd(content)
	if !ok {
		return nil
	}

	lines := parser.NewLineTable(content)
	endLine := lines.LineOf(end)

	onCloser := int(at.Start.Line) <= endLine && endLine <= int(at.End.Line)

	if !onHeader && !onCloser {
		return nil
	}

	missing := s.missingInterfaceMethods(docURI, file, content, ifaces)
	if len(missing) == 0 {
		return nil
	}

	var (
		actions  []protocol.CodeAction
		allStubs = make([]lifecycleStub, 0, len(missing))
	)

	for _, m := range missing {
		allStubs = append(allStubs, interfaceMethodStub(m.fn))
	}

	add := func(title string, stubs []lifecycleStub) {
		kind := protocol.CodeActionKindRefactor

		actions = append(actions, protocol.CodeAction{
			Title: title,
			Kind:  &kind,
			Edit: &protocol.WorkspaceEdit{Changes: map[uri.URI][]protocol.TextEdit{
				docURI: {insertStubs(content, end, tags, stubs)},
			}},
		})
	}

	if len(missing) > 1 {
		add("Implement all missing interface methods", allStubs)
	}

	for i, m := range missing {
		add("Implement "+m.fn.Name+"() from "+m.ifaceName, []lifecycleStub{allStubs[i]})
	}

	return s.withCRLineBreaks(docURI, actions)
}

// maxClosePrefilterLines bounds the lines rangeMayHoldComponentClose reads; a
// selection wider than that is answered "maybe" without looking.
const maxClosePrefilterLines = 256

// rangeMayHoldComponentClose reports whether lines startLine to endLine (0-based,
// inclusive) could hold the end of a component's body. ComponentBodyEnd answers
// with the `}` that closes a script component or the start of the last
// `</cfcomponent>` or `</cfinterface>`, so a line that has neither a `}` nor
// `</cf` cannot be that line. A line is found by counting newlines, a scan
// orders of magnitude cheaper than tokenizing the component.
func rangeMayHoldComponentClose(content string, startLine, endLine int) bool {
	if startLine < 0 || endLine < startLine || endLine-startLine > maxClosePrefilterLines {
		return true
	}

	from := 0

	for range startLine {
		i := strings.IndexByte(content[from:], '\n')
		if i < 0 {
			return false
		}

		from += i + 1
	}

	to := from

	for range endLine - startLine + 1 {
		i := strings.IndexByte(content[to:], '\n')
		if i < 0 {
			to = len(content)

			break
		}

		to += i + 1
	}

	seg := content[from:to]

	return strings.IndexByte(seg, '}') >= 0 || parser.IndexFold(seg, "</cf") >= 0
}

func (s *Server) implementsForOpenDoc(docURI uri.URI, content string) []string {
	if pr := s.parseResultOf(docURI); pr != nil && pr.Content == content {
		return pr.Implements
	}

	if ifaces := s.index.ImplementsForFile(docURI); len(ifaces) > 0 {
		return ifaces
	}

	if parser.IndexFold(content, "implements") < 0 {
		return nil
	}

	return parser.Parse(docURI, content).Implements
}

func (s *Server) missingInterfaceMethods(docURI uri.URI, file, content string, ifaces []string) []missingIfaceMethod {
	res := s.getResolver()

	// Methods already present in the current buffer or inherited via `extends`.
	implemented := declaredFunctionNames(content)

	extSeen := map[string]bool{strings.ToLower(file): true}
	curExt, _ := s.index.ExtendsForFile(docURI)
	curDir := filepath.Dir(file)

	if curExt == "" {
		if pr := s.parseResultOf(docURI); pr != nil && pr.Content == content {
			curExt = pr.Extends
		}
	}

	for hops := 0; curExt != "" && hops < 64; hops++ {
		parentPath := res.ComponentPath(curExt, curDir)
		if parentPath == "" || extSeen[strings.ToLower(parentPath)] {
			break
		}

		extSeen[strings.ToLower(parentPath)] = true

		for _, fn := range res.EnsureIndexed(parentPath) {
			implemented[strings.ToLower(fn.Name)] = true
		}

		curExt, _ = s.index.ExtendsForFile(cfpath.ToURI(parentPath))
		curDir = filepath.Dir(parentPath)
	}

	// BFS over declared interfaces and their parent interfaces (`extends`).
	var (
		ifaceSeen = map[string]bool{strings.ToLower(file): true}
		queue     []string
	)

	enqueueIface := func(name, dir string) {
		p := res.ComponentPath(name, dir)
		if p == "" || ifaceSeen[strings.ToLower(p)] {
			return
		}

		ifaceSeen[strings.ToLower(p)] = true
		queue = append(queue, p)
	}

	for _, name := range ifaces {
		enqueueIface(name, filepath.Dir(file))
	}

	var (
		missing    []missingIfaceMethod
		methodSeen = map[string]bool{}
	)

	for hops := 0; len(queue) > 0 && hops < 64; hops++ {
		ifacePath := queue[0]
		queue = queue[1:]

		ifaceBase := strings.TrimSuffix(filepath.Base(ifacePath), filepath.Ext(ifacePath))

		for _, fn := range res.EnsureIndexed(ifacePath) {
			if fn.Generated || fn.Name == "" {
				continue
			}

			key := strings.ToLower(fn.Name)
			if implemented[key] || methodSeen[key] {
				continue
			}

			methodSeen[key] = true

			missing = append(missing, missingIfaceMethod{fn: fn, ifaceName: ifaceBase})
		}

		if parentExt, _ := s.index.ExtendsForFile(cfpath.ToURI(ifacePath)); parentExt != "" {
			enqueueIface(parentExt, filepath.Dir(ifacePath))
		}
	}

	return missing
}

func interfaceMethodStub(fn *parser.FunctionDef) lifecycleStub {
	return lifecycleStub{
		name:   fn.Name,
		script: formatScriptMethodStub(fn),
		tags:   formatTagMethodStub(fn),
	}
}

func formatScriptMethodStub(fn *parser.FunctionDef) string {
	var b strings.Builder

	b.WriteString("public ")

	if fn.ReturnType != "" {
		b.WriteString(fn.ReturnType)
		b.WriteByte(' ')
	}

	b.WriteString("function ")
	b.WriteString(fn.Name)
	b.WriteByte('(')

	if len(fn.Arguments) > 0 {
		b.WriteByte(' ')

		for i, arg := range fn.Arguments {
			if i > 0 {
				b.WriteString(", ")
			}

			if arg.Required {
				b.WriteString("required ")
			}

			if arg.Type != "" {
				b.WriteString(arg.Type)
				b.WriteByte(' ')
			}

			b.WriteString(arg.Name)
		}

		b.WriteByte(' ')
	}

	b.WriteString(") {\n\t// TODO: implement\n}")

	return b.String()
}

func formatTagMethodStub(fn *parser.FunctionDef) string {
	var b strings.Builder

	b.WriteString(`<cffunction name="`)
	b.WriteString(fn.Name)
	b.WriteString(`" access="public"`)

	if fn.ReturnType != "" {
		b.WriteString(` returntype="`)
		b.WriteString(fn.ReturnType)
		b.WriteByte('"')
	}

	b.WriteString(">\n")

	for _, arg := range fn.Arguments {
		b.WriteString("\t<cfargument name=\"")
		b.WriteString(arg.Name)
		b.WriteByte('"')

		if arg.Type != "" {
			b.WriteString(` type="`)
			b.WriteString(arg.Type)
			b.WriteByte('"')
		}

		if arg.Required {
			b.WriteString(` required="true"`)
		}

		b.WriteString(">\n")
	}

	b.WriteString("</cffunction>")

	return b.String()
}
