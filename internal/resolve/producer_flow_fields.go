package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// sharedFieldContract returns the one component assigned to a guarded shared
// field anywhere in its component. Every explicit write must agree. Primitive
// writes are admitted only for isSimpleValue's sentinel pattern.
func (e *producerEvaluation) sharedFieldContract(path string, allowPrimitive bool) producerValue {
	path = normalizeProducerPath(path)
	if !e.guardedField(path) {
		return producerUnknown()
	}

	source := e.resolver.sourceInfo(e.fd.URI.Path())
	methods := sourcePlans(source).all()
	component := ""
	found := false
	valid := true
	request := strings.HasPrefix(path, "request.")
	writes := 0

	// A statement the plan could not read may write the field, unless the
	// file's text shows it cannot (fieldTextProved).
	unread := false
	write := func(expression string) {
		value := e.expression(expression, producerEnvironment{})
		switch {
		case value.unknown, value.fields != nil, value.primitive && !allowPrimitive, len(value.components) > 1:
			valid = false
		case value.primitive:
			found = true
		case len(value.components) == 1:
			found = true

			if component == "" {
				component = value.components[0]
			} else if component != value.components[0] {
				valid = false
			}
		default:
			valid = false
		}
	}

	var visit func([]producerNode)

	visit = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			// A request field's writes are counted in the text instead.
			if node.kind == "unsafe" && !request && producerMayTouchField(node.expression, path) {
				unread = true
			}

			// A param on the field writes its default whenever the field is
			// absent, and the default is not read here: the writes cannot be
			// shown to agree.
			if node.kind == "param" && normalizeProducerPath(node.target) == path {
				valid = false
			}

			if node.kind == "set" {
				target := normalizeProducerPath(node.target)
				if target == "variables" || target == "this" {
					valid = false
				} else if !strings.Contains(node.target, ".") {
					target = "variables." + target
				}

				if target == path {
					writes++

					write(node.expression)
				}
			}

			visit(node.body)
			visit(node.alternative)
		}
	}

	for _, method := range methods {
		visit(method.body)
	}

	// Function plans deliberately exclude component-body initialization. It is
	// still an explicit write to the field and must participate in the same
	// contract; otherwise a conflicting startup value would be ignored.
	for _, assignment := range source.startupAssignments(e.fd.URI) {
		if normalizeProducerPath(assignment.Variable) != path {
			continue
		}

		if assignment.Expression == "" {
			valid = false
		} else {
			write(assignment.Expression)
		}
	}

	if unread && !e.fieldTextProved(&source, methods, path) {
		valid = false
	}

	// Every write of a request field in the file's text must be one the
	// plan read: a statement it could not read may hold another.
	if request && source.requestWriteCounts()[strings.TrimPrefix(path, "request.")] != writes {
		valid = false
	}

	if !valid || !found || component == "" {
		return producerUnknown()
	}

	if request {
		others, read := e.requestWriterComponents(path)
		want := e.resolver.ComponentPath(component, e.baseDir)

		if !read || want == "" || slices.ContainsFunc(others, func(f string) bool { return !cfpath.SamePath(f, want) }) {
			return producerUnknown()
		}
	}

	return e.concrete(component, e.baseDir)
}

// fieldTextProved reports whether the file's text shows that no statement the
// plans could not read touches the variables field path: every reference to
// the variable holding it (variablesRoot) is in a statement a plan read, or a
// component-body assignment, and the scope is never used whole. A large
// component always holds a statement the plans cannot read, and the
// cfwheels CLI's 8,000-line Module.cfc lost every guarded field to one.
func (e *producerEvaluation) fieldTextProved(source *sourceInfo, methods map[string]*producerMethod, path string) bool {
	root := variablesRoot(path)
	if root == "" {
		return false
	}

	// The answer depends on the file and the root alone, and the walk is every
	// method of the file: asked once per guarded field each call reads, it
	// was most of the time a 15,000-line component took to open.
	if source.derived != nil {
		if proved, ok := source.derived.fieldProofs.Load(root); ok {
			return proved.(bool)
		}
	}

	proved := e.fieldTextProvedUncached(source, methods, root)

	if source.derived != nil {
		source.derived.fieldProofs.Store(root, proved)
	}

	return proved
}

func (e *producerEvaluation) fieldTextProvedUncached(source *sourceInfo, methods map[string]*producerMethod, root string) bool {
	read := 0
	planned := map[string]bool{}

	var visit func([]producerNode)

	visit = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]
			if node.kind != "unsafe" {
				read += variablesRootReferences(producerTokens(node.target+" "+node.expression), root)

				if node.kind == "set" {
					planned[producerStatementKey(node.target, node.expression)] = true
				}
			}

			visit(node.body)
			visit(node.alternative)
		}
	}

	for _, method := range methods {
		visit(method.body)
	}

	// A component-body assignment is in the text once, and in a plan too
	// when an init() holds it.
	for _, assignment := range source.startupAssignments(e.fd.URI) {
		if !planned[producerStatementKey(assignment.Variable, assignment.Expression)] {
			read += variablesRootReferences(producerTokens(assignment.Variable+" = "+assignment.Expression), root)
		}
	}

	return source.variablesReferences().proves(root, read)
}

// guardedField reports whether path is a field whose writes the component
// holds: its own variables or this scope, or a request-scope key this file
// writes in a batch scan, where every other writer that can share its request
// must agree (requestWriterComponents). Mura's getCurrentUser() creates
// request.currentUser when it is absent and nothing else ever sets it.
func (e *producerEvaluation) guardedField(path string) bool {
	if strings.HasPrefix(path, "variables.") || strings.HasPrefix(path, "this.") {
		return true
	}

	name, ok := strings.CutPrefix(path, "request.")

	return ok && name != "" && !strings.Contains(name, ".") && e.resolver.requestWrittenBy(name, e.fd.URI.Path())
}

// requestWriterComponents is what every other file that can write the request
// key path in the same request stores in it, as component files, or ok false
// when one cannot be read: a file's writes in the text, outside comments, must
// be the writes its plans hold, each a value that is one component. A request
// runs its own application's Application.cfc, so another application's
// Application.cfc writes another request, unless one of the two extends the
// other. FW/1's five example applications each create
// request._framework_one in their own Application.cfc.
func (e *producerEvaluation) requestWriterComponents(path string) (files []string, ok bool) {
	name := strings.TrimPrefix(path, "request.")
	file := e.fd.URI.Path()

	for _, writer := range e.resolver.requestWriters()[name] {
		if cfpath.SamePath(writer, file) || e.resolver.otherApplication(file, writer) {
			continue
		}

		stored, read := e.writerComponents(writer, path)
		if !read {
			return nil, false
		}

		for _, file := range stored {
			if !slices.ContainsFunc(files, func(f string) bool { return cfpath.SamePath(f, file) }) {
				files = append(files, file)
			}
		}
	}

	return files, true
}

// writerComponents is the component files the file at writer stores in the
// request key path, or read false when a write there is not one its plans
// read as one component.
func (e *producerEvaluation) writerComponents(writer, path string) (files []string, read bool) {
	source := e.resolver.sourceInfo(writer)
	other := producerEvaluation{resolver: e.resolver, fd: &parser.FunctionDef{URI: cfpath.ToURI(writer)}, baseDir: filepath.Dir(writer), depth: e.depth + 1, budget: e.budget}
	read = true
	writes := 0

	var visit func([]producerNode)

	visit = func(nodes []producerNode) {
		for i := range nodes {
			node := &nodes[i]

			switch {
			case node.kind == "unsafe" && producerMayTouchField(node.expression, path),
				node.kind == "param" && normalizeProducerPath(node.target) == path:
				read = false
			case node.kind == "set" && normalizeProducerPath(node.target) == path:
				writes++

				value := other.expression(node.expression, producerEnvironment{})

				file := ""
				if !value.unknown && len(value.components) == 1 {
					file = e.resolver.ComponentPath(value.components[0], other.baseDir)
				}

				if file == "" {
					read = false
				} else if !slices.ContainsFunc(files, func(f string) bool { return cfpath.SamePath(f, file) }) {
					files = append(files, file)
				}
			}

			visit(node.body)
			visit(node.alternative)
		}
	}

	for _, method := range sourcePlans(source).all() {
		visit(method.body)
	}

	return files, read && writes > 0 && source.requestWriteCounts()[strings.TrimPrefix(path, "request.")] == writes
}

// Unsupported plans cannot supply an all-writes-agree proof. A scope or field
// reference may hide a write, an indexed assignment, or a scope escape. An
// exhausted scan likewise withholds the contract.
func producerMayTouchField(expression, path string) bool {
	tokens := producerTokens(expression)
	if len(tokens) == 0 {
		return true
	}

	scope, field, _ := strings.Cut(path, ".")
	for _, token := range tokens {
		if token.Kind == parser.TokIdent && (strings.EqualFold(token.Value, scope) || strings.EqualFold(token.Value, field)) {
			return true
		}
	}

	return false
}
