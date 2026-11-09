// The completion cache: snippet policy, what a document's cache holds and how it is rebuilt.

package server

import (
	"sort"
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/cache"
	cflog "github.com/cfmleditor/clif/internal/log"
	"github.com/cfmleditor/clif/internal/parser"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// completionFromCache returns cached items for the cursor's scope.
// File cache already contains builtins + globals. Function cache has local vars only.
// applySnippetPolicy blanks the snippet insert text of the item kinds the
// config turned off, copying only if it has something to change.
//
// It used to write over items in place at the end of the handler, and items is
// frequently a slice the server handed over rather than copied: the
// process-wide builtin function list behind a sync.Once, or a document's
// completion cache entry. So with functionSnippets or tagSnippets off, serving
// one request permanently rewrote the shared list for every session in the
// process. That was invisible because the edit is idempotent and the settings
// are process-wide, and it is the reason the handler could not hand out the
// cached slice without copying it first.
//
// Both flags default on, so the common path returns the input untouched and
// costs nothing.
func (s *Server) applySnippetPolicy(items []protocol.CompletionItem) []protocol.CompletionItem {
	if s.TagSnippets && s.FunctionSnippets {
		return items
	}

	var out []protocol.CompletionItem

	for i := range items {
		if items[i].InsertTextFormat != protocol.InsertTextFormatSnippet {
			continue
		}

		isTag := items[i].Kind == protocol.CompletionItemKindKeyword
		isFunc := items[i].Kind == protocol.CompletionItemKindFunction || items[i].Kind == protocol.CompletionItemKindMethod

		if (isTag && !s.TagSnippets) || (isFunc && !s.FunctionSnippets) {
			if out == nil {
				out = append(make([]protocol.CompletionItem, 0, len(items)), items...)
			}

			out[i].InsertText = protocol.Optional[string]{}
			out[i].InsertTextFormat = protocol.InsertTextFormatPlainText
		}
	}

	if out == nil {
		return items
	}

	return out
}

func (s *Server) completionFromCache(docURI uri.URI, line int) []protocol.CompletionItem {
	s.mu.RLock()
	funcs := s.funcRanges[docURI]
	s.mu.RUnlock()

	fileItems := s.compCache.GetFile(docURI)
	if fileItems == nil {
		fileItems = s.builtinFuncItems()
	}

	idx := sort.Search(len(funcs), func(i int) bool {
		return funcs[i].End >= line
	})
	if idx < len(funcs) && line >= funcs[idx].Start {
		funcItems := s.compCache.GetFuncStale(docURI, funcs[idx].Name)
		if len(funcItems) == 0 {
			return fileItems
		}

		items := make([]protocol.CompletionItem, 0, len(fileItems)+len(funcItems))
		items = append(items, fileItems...)
		items = append(items, funcItems...)

		return items
	}

	return fileItems
}

// rebuildCompletionCache pre-computes completion items for scopes in a file.
// editLine indicates which line was edited; only the function containing that line
// has its local vars scanned. Pass -1 to skip function scanning (file-level only).
func (s *Server) rebuildCompletionCache(docURI uri.URI, content string, editLine int) {
	start := time.Now()

	defer s.lockDoc(docURI)()

	s.mu.RLock()
	funcs := s.funcRanges[docURI]
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	if editLine >= 0 {
		// Binary search — funcRanges is sorted by Start line
		idx := sort.Search(len(funcs), func(i int) bool {
			return funcs[i].End >= editLine
		})
		if idx < len(funcs) && editLine >= funcs[idx].Start {
			f := funcs[idx]
			hash := cache.HashScope(content, f.Start, f.End)

			if s.compCache.GetFunc(docURI, f.Name, hash) == nil {
				var vars []string
				if pr != nil {
					vars = pr.FuncVars(f.Start, f.End)
				} else {
					vars = parser.VarsInFunc(content, f.Start, f.End)
				}

				items := make([]protocol.CompletionItem, 0, len(vars))
				for _, v := range vars {
					items = append(items, protocol.CompletionItem{Label: v, Kind: protocol.CompletionItemKindVariable, SortText: optStr(SortLocalVariables + v)})
				}

				s.compCache.PutFunc(docURI, f.Name, hash, items)
				s.log.Debug("completion: func vars rebuilt",
					cflog.String("uri", string(docURI)),
					cflog.String("func", f.Name),
					cflog.Int("vars", len(vars)),
					cflog.Duration("dur", time.Since(start)),
				)
			}
		}
	}
}

// rebuildFileCompletionCache rebuilds the file-level completion cache (builtins + globals).
// Called on didOpen and didSave. Builtins are included here since this only runs
// on open/save, avoiding per-request copies.
func (s *Server) rebuildFileCompletionCache(docURI uri.URI) {
	defer s.lockDoc(docURI)()

	s.mu.RLock()
	pr := s.parseResults[docURI]
	s.mu.RUnlock()

	if pr != nil {
		s.rebuildFileCompletionCacheFromPR(docURI, pr)

		return
	}

	content, ok := s.getDocument(docURI)
	if !ok {
		return
	}

	newPR := s.parseContent(docURI, content)
	s.mu.Lock()
	s.parseResults[docURI] = newPR
	s.mu.Unlock()
	s.rebuildFileCompletionCacheFromPR(docURI, newPR)
}

// rebuildFileCompletionCacheFromPR rebuilds file-level completion from an existing ParseResult.
//
// The caller must hold the document's lock: pr.VariablesVars and pr.ThisVars
// memoise in place. It is not taken here because rebuildFileCompletionCache
// already holds it by the time it calls this, and lockDoc hands out plain
// sync.Mutexes — taking it twice on one goroutine deadlocks rather than
// nesting.
func (s *Server) rebuildFileCompletionCacheFromPR(docURI uri.URI, pr *parser.ParseResult) {
	start := time.Now()

	builtins := s.builtinFuncItems()
	globals := pr.VariablesVars()
	thisVarNames := pr.ThisVars()
	items := make([]protocol.CompletionItem, 0, len(builtins)+len(globals)+len(thisVarNames)+len(pr.Funcs))
	items = append(items, builtins...)
	varPrefix := s.scopePrefix("variables")
	thisPrefix := s.scopePrefix("this")

	items = appendScopedVars(items, globals, varPrefix, protocol.CompletionItemKindVariable)
	items = appendScopedVars(items, thisVarNames, thisPrefix, protocol.CompletionItemKindProperty)

	for j := range pr.Funcs {
		f := &pr.Funcs[j]

		var detail strings.Builder
		detail.WriteString(f.Name)
		detail.WriteByte('(')

		writeArgDetail(&detail, f.Arguments)
		insertText := callSnippet(f.Name, requiredArgs(f.Arguments))
		items = append(items, protocol.CompletionItem{
			Label:  f.Name,
			Kind:   protocol.CompletionItemKindFunction,
			Detail: optStr(detail.String()),

			InsertText: optStr(insertText),

			InsertTextFormat: protocol.InsertTextFormatSnippet,
			SortText:         optStr(SortUserFunctions + f.Name),
		})
	}

	s.compCache.PutFile(docURI, items)

	varsItems := make([]protocol.CompletionItem, 0, len(pr.VariablesVars()))
	for _, v := range pr.VariablesVars() {
		varsItems = append(varsItems, protocol.CompletionItem{Label: v, Kind: protocol.CompletionItemKindVariable, SortText: optStr(SortLocalVariables + v)})
	}

	s.compCache.PutFunc(docURI, "__variables__", 0, varsItems)

	thisItems := make([]protocol.CompletionItem, 0, len(pr.Funcs))
	for _, v := range pr.ThisVars() {
		thisItems = append(thisItems, protocol.CompletionItem{Label: v, Kind: protocol.CompletionItemKindProperty, SortText: optStr(SortLocalVariables + v)})
	}

	for j := range pr.Funcs {
		f := &pr.Funcs[j]

		var detail strings.Builder
		detail.WriteString(f.Name)
		detail.WriteByte('(')

		for i, arg := range f.Arguments {
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
		thisItems = append(thisItems, protocol.CompletionItem{
			Label:  f.Name,
			Kind:   protocol.CompletionItemKindMethod,
			Detail: optStr(detail.String()),

			SortText: optStr(SortUserFunctions + f.Name),
		})
	}

	s.compCache.PutFunc(docURI, "__this__", 0, thisItems)
	s.log.Debug("completion: file globals rebuilt",
		cflog.String("uri", string(docURI)),
		cflog.Int("globals", len(globals)),
		cflog.Duration("dur", time.Since(start)),
	)
}
