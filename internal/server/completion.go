package server

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/docs"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// Completion feature flags — set to false to disable specific providers.
const (
	CompletionBuiltinFunctions = true
	CompletionLocalVariables   = true
	CompletionDotMethods       = true
	CompletionMemberFunctions  = true
	CompletionTags             = true
	CompletionCloseTags        = true
	CompletionAttributes       = true
)

// Sort priority prefixes for completion items. Lower ASCII = higher priority.
// Each category has a unique prefix to ensure deliberate ordering.
const (
	SortCloseTags       = "0" // immediate actions (close tag, remove duplicate >)
	SortFuncArguments   = "1" // named function arguments (companyCode=)
	SortLocalVariables  = "2" // local/function-scoped variables
	SortUserFunctions   = "3" // user-defined functions and methods in current file
	SortProperties      = "4" // this/variables properties
	SortGlobalVariables = "5" // file-scope global variables
	SortTags            = "6" // CF and HTML tags
	SortScopes          = "7" // scope keywords (VARIABLES, ARGUMENTS, etc.)
	SortBuiltinFuncs    = "8" // built-in CFML functions
	SortMemberFuncs     = "9" // member functions (after dot on unresolved type)
)

// Cached global completion items (never change at runtime).
var (
	builtinFuncItems     []protocol.CompletionItem
	builtinFuncItemsOnce sync.Once
	memberFuncItems      []protocol.CompletionItem
	memberFuncItemsOnce  sync.Once
	// memberFuncEntries[i] is the docs entry memberFuncItems[i] came from,
	// which completionItem/resolve needs: see memberKey.
	memberFuncEntries []string
)

func getBuiltinFuncItems() []protocol.CompletionItem {
	builtinFuncItemsOnce.Do(func() {
		fns := docs.AllFunctions()
		scopes := []string{"VARIABLES", "ARGUMENTS", "THIS", "SERVER", "APPLICATION", "REQUEST", "SESSION"}
		builtinFuncItems = make([]protocol.CompletionItem, 0, len(fns)+len(scopes))

		for _, fn := range fns {
			var required []string

			for i := range fn.Params {
				if p := &fn.Params[i]; p.Required {
					required = append(required, p.Name)
				}
			}

			insertText := callSnippet(fn.Name, required)
			builtinFuncItems = append(builtinFuncItems, protocol.CompletionItem{
				Label:            fn.Name,
				Kind:             protocol.CompletionItemKindFunction,
				Detail:           optStr(fn.Syntax),
				Documentation:    tooltip(fn.Description),
				InsertText:       optStr(insertText),
				InsertTextFormat: protocol.InsertTextFormatSnippet,
				SortText:         optStr(SortBuiltinFuncs + fn.Name),
			})
		}

		for i, s := range scopes {
			builtinFuncItems = append(builtinFuncItems, protocol.CompletionItem{
				Label:    s,
				Kind:     protocol.CompletionItemKindKeyword,
				SortText: optStr(fmt.Sprintf("%s%d%s", SortScopes, i, s)),
			})
		}
	})

	return builtinFuncItems
}

func getMemberFuncItems() []protocol.CompletionItem {
	memberFuncItemsOnce.Do(func() {
		mfs := docs.AllMemberFunctions()
		memberFuncItems = make([]protocol.CompletionItem, 0, len(mfs))
		memberFuncEntries = make([]string, 0, len(mfs))

		for _, mf := range mfs {
			memberFuncEntries = append(memberFuncEntries, mf.Entry.Name)
			memberFuncItems = append(memberFuncItems, protocol.CompletionItem{
				Label:         mf.Name,
				Kind:          protocol.CompletionItemKindMethod,
				Detail:        optStr(mf.Entry.Syntax),
				Documentation: tooltip(mf.Entry.Description),
				SortText:      optStr(SortMemberFuncs + mf.Name),
			})
		}
	})

	return memberFuncItems
}

// completionContext is what the cursor's position says about the request.
type completionContext struct {
	tagName     string
	triggerChar string
	char        int // the cursor as a byte column; params.Position stays in UTF-16

	triggeredByTag, triggeredByClose, triggeredByDot bool
	closing, typingTag, inHashExpr, inAttrValue      bool
}

func completionContextAt(params *protocol.CompletionParams, content string, hasDoc bool) completionContext {
	var c completionContext

	line := int(params.Position.Line)

	// The cursor as a byte column, which is what every helper below reads the
	// line with; params.Position stays the client's, in UTF-16 units, for the
	// edits that start at the cursor.
	if hasDoc {
		c.char = byteCol(content, line, params.Position.Character)
		c.tagName = parser.FindEnclosingTag(content, line, c.char)
	}

	if params.Context.TriggerCharacter != nil {
		c.triggerChar = *params.Context.TriggerCharacter
	}

	byChar := params.Context.TriggerKind == protocol.CompletionTriggerKindTriggerCharacter

	c.triggeredByTag = (byChar && c.triggerChar == "<") ||
		(hasDoc && strings.HasSuffix(parser.TextBeforeCursor(content, line, c.char), "<"))
	c.triggeredByClose = byChar && c.triggerChar == ">"
	c.triggeredByDot = (byChar && c.triggerChar == ".") ||
		(hasDoc && parser.WordBeforeDot(content, line, c.char) != "")

	if hasDoc {
		c.closing = parser.IsClosingTagContext(content, line, c.char)
		if !c.closing && c.tagName == "" {
			c.typingTag = parser.IsTypingTagName(content, line, c.char)
		}

		c.inHashExpr = parser.IsInsideHashExpr(content, line, c.char)
		c.inAttrValue = parser.IsInsideAttrValue(content, line, c.char)
	}

	return c
}

func (s *Server) handleCompletion(_ context.Context, rawParams []byte) (any, error) {
	totalStart := time.Now()

	var params protocol.CompletionParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}

	defer s.lockDoc(params.TextDocument.URI)()

	content, hasDoc := s.getDocument(params.TextDocument.URI)

	t0 := time.Now()
	cc := completionContextAt(&params, content, hasDoc)
	contextDur := time.Since(t0)

	triggerKind := triggerKindName(params.Context.TriggerKind, cc.triggerChar)

	s.log.Debug("completion: request",
		cflog.String("trigger", triggerKind),
		cflog.Bool("triggeredByDot", cc.triggeredByDot),
		cflog.Bool("inHashExpr", cc.inHashExpr),
		cflog.Bool("inAttrValue", cc.inAttrValue),
		cflog.String("tagName", cc.tagName),
		cflog.Uint32("line", params.Position.Line),
		cflog.Uint32("char", params.Position.Character),
	)

	items := s.completionItemsFor(&params, &cc, content, hasDoc)

	s.log.Debug("completion: total",
		cflog.Duration("context", contextDur),
		cflog.Duration("total", time.Since(totalStart)),
		cflog.Int("items", len(items)),
	)

	items = s.applySnippetPolicy(items)

	if items == nil {
		items = []protocol.CompletionItem{}
	}

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// triggerKindName is the trigger of a completion request as the debug log
// spells it.
func triggerKindName(kind protocol.CompletionTriggerKind, triggerChar string) string {
	switch kind {
	case protocol.CompletionTriggerKindInvoked:
		return "invoked"
	case protocol.CompletionTriggerKindTriggerCharacter:
		return "char:" + triggerChar
	case protocol.CompletionTriggerKindTriggerForIncompleteCompletions:
		return "incomplete"
	}

	return ""
}

// completionItemsFor is what the cursor's context offers, the first of these
// that applies: a doc block's tags, a hash expression's variables, an
// attribute value's values, the characters that close a tag, the attributes of
// the tag around the cursor, a tag name, the methods after a dot, and
// otherwise the named arguments of the call around the cursor and the
// document's cached variables and functions.
func (s *Server) completionItemsFor(params *protocol.CompletionParams, cc *completionContext, content string, hasDoc bool) []protocol.CompletionItem {
	char, tagName := cc.char, cc.tagName
	items := []protocol.CompletionItem(nil)

	docItems, inDocBlock := []protocol.CompletionItem(nil), false
	if hasDoc {
		docItems, inDocBlock = s.docBlockItems(content, params)
	}

	switch {
	case inDocBlock:
		items = docItems
	case cc.inHashExpr:
		items = s.completionFromCache(params.TextDocument.URI, int(params.Position.Line))
	case cc.inAttrValue:
		if CompletionAttributes {
			items = attrValueItems(content, int(params.Position.Line), char, tagName)
		}

		if CompletionBuiltinFunctions {
			items = append(items, s.builtinFuncItems()...)
		}
	case cc.triggeredByClose && hasDoc:
		if CompletionCloseTags {
			if item, ok := duplicateGtCompletion(content, int(params.Position.Line), char); ok {
				items = append(items, item)
			}

			if item, ok := closeTagCompletion(content, int(params.Position.Line), char); ok {
				items = append(items, item)
			}
		}
	case cc.closing:
		if CompletionCloseTags {
			t1 := time.Now()
			items = s.closeTagItems(content, hasDoc, params.TextDocument.URI, params.Position, char)

			s.log.Debug("completion: closeTags", cflog.Duration("dur", time.Since(t1)))
		}
	case tagName != "" && !parser.IsSpecialTag(tagName):
		if CompletionAttributes {
			t1 := time.Now()
			items = tagAttributeItems(tagName)

			s.log.Debug("completion: attributes", cflog.Duration("dur", time.Since(t1)))
		}
	case tagName == "cfelse":
		items = append(items, cfelseifItem(content, params.Position, char))
	case cc.triggeredByTag:
		if CompletionTags {
			t1 := time.Now()
			items = tagNameItems()

			s.log.Debug("completion: tags", cflog.Duration("dur", time.Since(t1)))
		}
	case cc.typingTag:
		if CompletionTags {
			items = tagNameItems()
		}
	case cc.triggeredByDot && hasDoc:
		if CompletionDotMethods {
			t1 := time.Now()

			if methods := s.dotCompletionMethods(content, params.TextDocument.URI, int(params.Position.Line), char); len(methods) > 0 {
				items = append(items, methods...)

				s.log.Debug("completion: dotMethods", cflog.Duration("dur", time.Since(t1)))
			} else if CompletionMemberFunctions {
				items = append(items, s.memberFuncItems()...)

				s.log.Debug("completion: memberFunctions", cflog.Duration("dur", time.Since(t1)))
			}
		}
	default:
		// Check if inside a function call — offer named argument completions first
		if hasDoc {
			if argItems := s.argumentCompletion(content, params.TextDocument.URI, int(params.Position.Line), char); len(argItems) > 0 {
				items = append(items, argItems...)
			}
		}

		// Nothing may be appended to items after this. The cached slice is
		// handed over rather than copied when there is nothing to merge it
		// with, so items can be the process-wide builtin list or a document's
		// cache entry — either of which a later append could write into past
		// its length, where a concurrent request is appending too. The copy
		// this avoids is around half the bytes of a completion round trip:
		// 383KB against the 742KB the whole request and its marshalling cost.
		if cached := s.completionFromCache(params.TextDocument.URI, int(params.Position.Line)); len(items) == 0 {
			items = cached
		} else {
			items = append(items, cached...)
		}
	}

	return items
}

// scopeArgumentsCompletion returns the declared arguments for the enclosing function.
func (s *Server) scopeArgumentsCompletion(docURI uri.URI, line int) []protocol.CompletionItem {
	s.mu.RLock()
	funcs := s.funcRanges[docURI]
	s.mu.RUnlock()

	idx := sort.Search(len(funcs), func(i int) bool {
		return funcs[i].End >= line
	})
	if idx >= len(funcs) || line < funcs[idx].Start {
		return nil
	}

	funcName := funcs[idx].Name

	defs := s.index.FunctionsForFile(docURI)
	for _, d := range defs {
		if strings.EqualFold(d.Name, funcName) {
			items := make([]protocol.CompletionItem, 0, len(d.Arguments))

			for _, arg := range d.Arguments {
				detail := arg.Type
				if arg.Required {
					detail += " required"
				}

				items = append(items, protocol.CompletionItem{
					Label:  arg.Name,
					Kind:   protocol.CompletionItemKindVariable,
					Detail: optStr(strings.TrimSpace(detail)),

					SortText: optStr(SortProperties + arg.Name),
				})
			}

			return items
		}
	}

	return nil
}

// superCompletion returns functions from the parent component (extends).
func (s *Server) superCompletion(docURI uri.URI) []protocol.CompletionItem {
	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	if pr == nil || pr.Extends == "" {
		return nil
	}

	currentPath := docURI.Path()
	baseDir := filepath.Dir(currentPath)

	cfcPath := s.getResolver().ComponentPath(pr.Extends, baseDir)
	if cfcPath == "" {
		return nil
	}

	defs := s.getResolver().EnsureIndexed(cfcPath)

	items := make([]protocol.CompletionItem, 0, len(defs))

	for _, d := range defs {
		var detail strings.Builder
		detail.WriteString(d.Name)
		detail.WriteByte('(')

		for i, arg := range d.Arguments {
			if i > 0 {
				detail.WriteString(", ")
			}

			if arg.Type != "" {
				detail.WriteString(arg.Type)
				detail.WriteByte(' ')
			}

			detail.WriteString(arg.Name)
		}

		detail.WriteString(")")
		items = append(items, protocol.CompletionItem{
			Label:  d.Name,
			Kind:   protocol.CompletionItemKindMethod,
			Detail: optStr(detail.String()),

			SortText: optStr(SortUserFunctions + d.Name),
		})
	}

	return items
}

// dotCompletionMethods returns completion items for methods on a component
// instance variable. It extracts the word before the dot, looks up the
// component ref, resolves the CFC path, and returns its function defs.
func (s *Server) dotCompletionMethods(content string, docURI uri.URI, line, char int) []protocol.CompletionItem {
	// Extract variable name before the dot
	varName := parser.WordBeforeDot(content, line, char)

	// If no simple word, check for call expression before dot: e.g. getService("tours").
	if varName == "" {
		callExpr := callExprBeforeDot(parser.LineTextAt(content, line), char)
		if callExpr == "" {
			return nil
		}

		comp := s.cfResolverSet().Resolve(callExpr)
		if comp == "" {
			return nil
		}

		cfcPath := s.getResolver().ComponentPath(comp, filepath.Dir(docURI.Path()))
		if cfcPath == "" {
			return nil
		}

		return s.methodCompletionItems(cfcPath)
	}

	// Scope dot completion: VARIABLES., ARGUMENTS., THIS., SUPER.
	switch strings.ToUpper(varName) {
	case "VARIABLES":
		return s.compCache.GetFuncStale(docURI, "__variables__")
	case "THIS":
		return s.compCache.GetFuncStale(docURI, "__this__")
	case "ARGUMENTS":
		return s.scopeArgumentsCompletion(docURI, line)
	case "SUPER":
		return s.superCompletion(docURI)
	}

	// Look up component ref for this variable in the current file
	ref := s.index.LookupComponentRefInFile(varName, docURI, conv.Uint32(line))

	var component string

	if ref != nil {
		component = ref.Component
	} else if comp := s.cfResolverSet().Resolve(varName); comp != "" {
		component = comp
	} else {
		return nil
	}

	return s.componentMemberItems(component, docURI)
}

// callExprBeforeDot returns the call expression that ends just before the dot
// at char-1 — `getService("tours")` in `getService("tours").` — or "" when
// the dot does not follow a closing paren.
func callExprBeforeDot(lineText string, char int) string {
	dotPos := char - 1
	if dotPos <= 0 || dotPos >= len(lineText) || lineText[dotPos] != '.' || lineText[dotPos-1] != ')' {
		return ""
	}

	// Find matching open paren
	depth := 0

	for i := dotPos - 1; i >= 0; i-- {
		switch lineText[i] {
		case ')':
			depth++
		case '(':
			depth--
			if depth != 0 {
				continue
			}

			// Find function name start
			fnStart := i - 1
			for fnStart >= 0 && parser.IsWordChar(lineText[fnStart]) {
				fnStart--
			}

			return lineText[fnStart+1 : dotPos]
		}
	}

	return ""
}

// componentMemberItems lists the this-scope properties and the methods of
// component, resolved relative to the requesting document.
func (s *Server) componentMemberItems(component string, docURI uri.URI) []protocol.CompletionItem {
	// Resolve the dot-path to a CFC file relative to the current file's directory
	baseDir := filepath.Dir(docURI.Path())

	var cfcPath string

	if filepath.IsAbs(component) {
		if _, err := s.FS.Stat(component); err == nil {
			cfcPath = component
		}
	}

	if cfcPath == "" {
		cfcPath = s.getResolver().ComponentPath(component, baseDir)
	}

	if cfcPath == "" {
		return nil
	}

	// Get function defs from the index (already cached), fall back to parsing
	cfcURI := cfpath.ToURI(cfcPath)
	defs := s.getResolver().EnsureIndexed(cfcPath)

	thisVars := s.index.ThisVarsForFile(cfcURI)

	if len(defs) == 0 && len(thisVars) == 0 {
		return nil
	}

	items := make([]protocol.CompletionItem, 0, len(defs)+len(thisVars))
	for _, v := range thisVars {
		items = append(items, protocol.CompletionItem{
			Label:    v,
			Kind:     protocol.CompletionItemKindProperty,
			SortText: optStr(SortLocalVariables + v),
		})
	}

	for _, d := range defs {
		item := protocol.CompletionItem{
			Label:  d.Name,
			Kind:   protocol.CompletionItemKindMethod,
			Detail: optStr(signatureDetail(d.Name, d.Arguments)),

			SortText: optStr(SortUserFunctions + d.Name),
		}
		if doc := stubDocMarkup(d); doc != nil {
			item.Documentation = doc
		}

		items = append(items, item)
	}

	return items
}

// signatureDetail renders `name(type arg, arg)` for a completion item.
func signatureDetail(name string, args []parser.Argument) string {
	var detail strings.Builder
	detail.WriteString(name)
	detail.WriteByte('(')

	for i, arg := range args {
		if i > 0 {
			detail.WriteString(", ")
		}

		if arg.Type != "" {
			detail.WriteString(arg.Type)
			detail.WriteByte(' ')
		}

		detail.WriteString(arg.Name)
	}

	detail.WriteString(")")

	return detail.String()
}

// argumentCompletion returns named argument completions when cursor is inside function parens.
func (s *Server) argumentCompletion(content string, docURI uri.URI, line, char int) []protocol.CompletionItem {
	funcName, qualifier, _ := parser.FindCallContext(content, line, char)
	if funcName == "" {
		return nil
	}

	var def *parser.FunctionDef

	// Try builtin
	if e, ok := docs.LookupFunction(funcName); ok {
		var items []protocol.CompletionItem

		for i := range e.Params {
			p := &e.Params[i]

			items = append(items, protocol.CompletionItem{
				Label:  p.Name + "=",
				Kind:   protocol.CompletionItemKindField,
				Detail: optStr(p.Type),

				InsertText: optStr(p.Name + "="),

				InsertTextFormat: protocol.InsertTextFormatPlainText,
				SortText:         optStr(SortFuncArguments + p.Name),
			})
		}

		return items
	}

	// Try qualified user function
	if qualifier != "" {
		def = s.resolveUserFunc(qualifier, funcName, docURI, conv.Uint32(line))
	}
	// Try unqualified
	if def == nil {
		// Prefer current file, else the first match. This runs on every
		// keystroke inside an argument list, so it must not copy the name's
		// bucket to read one entry out of it.
		if match, _, total := s.index.LookupPreferred(funcName, docURI); total > 0 {
			def = match
		}
	}

	if def == nil || len(def.Arguments) == 0 {
		return nil
	}

	var items []protocol.CompletionItem
	for _, arg := range def.Arguments {
		items = append(items, protocol.CompletionItem{
			Label:  arg.Name + "=",
			Kind:   protocol.CompletionItemKindField,
			Detail: optStr(arg.Type),

			InsertText: optStr(arg.Name + "="),

			InsertTextFormat: protocol.InsertTextFormatPlainText,
			SortText:         optStr(SortFuncArguments + arg.Name),
		})
	}

	return items
}

// methodCompletionItems returns completion items for all functions in a CFC file.
func (s *Server) methodCompletionItems(cfcPath string) []protocol.CompletionItem {
	defs := s.getResolver().EnsureIndexed(cfcPath)
	items := make([]protocol.CompletionItem, 0, len(defs))

	for _, d := range defs {
		var detail strings.Builder
		detail.WriteString(d.Name)
		detail.WriteByte('(')

		writeArgDetail(&detail, d.Arguments)
		insertText := callSnippet(d.Name, requiredArgs(d.Arguments))

		item := protocol.CompletionItem{
			Label:  d.Name,
			Kind:   protocol.CompletionItemKindMethod,
			Detail: optStr(detail.String()),

			InsertText: optStr(insertText),

			InsertTextFormat: protocol.InsertTextFormatSnippet,
			SortText:         optStr(SortUserFunctions + d.Name),
		}
		if doc := stubDocMarkup(d); doc != nil {
			item.Documentation = doc
		}

		items = append(items, item)
	}

	return items
}

// callSnippet is a call to name with a placeholder for each required
// argument, and the cursor inside the parentheses when there is none. Every
// argument used to get one, so completing writeDump filled in every optional
// argument it takes (intellilucee#47); signature help shows the optional
// ones as they are needed, and a tag's snippet has always held only its
// required attributes (buildTagSnippet).
func callSnippet(name string, required []string) string {
	if len(required) == 0 {
		return name + "($0)"
	}

	var b strings.Builder

	b.WriteString(name)
	b.WriteByte('(')

	for i, arg := range required {
		if i > 0 {
			b.WriteString(", ")
		}

		fmt.Fprintf(&b, "${%d:%s}", i+1, arg)
	}

	b.WriteByte(')')

	return b.String()
}

func requiredArgs(args []parser.Argument) []string {
	var out []string

	for _, a := range args {
		if a.Required {
			out = append(out, a.Name)
		}
	}

	return out
}

// writeArgDetail writes the argument list a function's completion item shows
// as its detail, every argument included, and the closing parenthesis.
func writeArgDetail(detail *strings.Builder, args []parser.Argument) {
	for i, arg := range args {
		if i > 0 {
			detail.WriteString(", ")
		}

		if arg.Type != "" {
			detail.WriteString(arg.Type)
			detail.WriteByte(' ')
		}

		detail.WriteString(arg.Name)
	}

	detail.WriteString(")")
}

// buildTagSnippet creates a snippet for a CF tag with required attributes as tab stops.
func buildTagSnippet(tag *docs.Entry) string {
	params := tag.Params

	var b strings.Builder

	b.WriteString(tag.Name)

	tabIdx := 1

	for i := range params {
		p := &params[i]

		if p.Required {
			fmt.Fprintf(&b, ` %s="${%d:%s}"`, p.Name, tabIdx, p.Name)

			tabIdx++
		}
	}

	b.WriteString(">")

	return b.String()
}

func (s *Server) scopePrefix(scope string) string {
	switch s.Formatting.ScopeCase {
	case "upper":
		return strings.ToUpper(scope)
	case "lower":
		return strings.ToLower(scope)
	default:
		return scope
	}
}

// appendScopedVars offers each name twice: spelled with its scope, and bare
// but inserting the scoped spelling.
func appendScopedVars(items []protocol.CompletionItem, names []string, prefix string, kind protocol.CompletionItemKind) []protocol.CompletionItem {
	for _, v := range names {
		scoped := prefix + "." + v
		items = append(items, protocol.CompletionItem{
			Label:    scoped,
			Kind:     kind,
			SortText: optStr(SortGlobalVariables + v),
		}, protocol.CompletionItem{
			Label:  v,
			Kind:   kind,
			Detail: optStr(scoped),

			InsertText: optStr(scoped),

			SortText: optStr(SortGlobalVariables + v),
		})
	}

	return items
}
