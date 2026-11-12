package parser

import (
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/conv"
	"github.com/cfmleditor/clif/internal/lex"
	"github.com/cfmleditor/clif/internal/log"
)

// EditKind describes what part of the file was affected by an edit.
type EditKind int

// EditKind classifies where an edit occurred.
const (
	EditInFunc EditKind = iota // Edit was inside a function body
	EditGlobal                 // Edit was outside all function bodies (requires re-parse of signatures)
	EditFull                   // Full document replacement
)

// ApplyEdit updates the ParseResult incrementally after a text edit.
// It updates the content, shifts line numbers, and re-parses only what's needed.
// Returns the EditKind indicating what was affected.
func (pr *ParseResult) ApplyEdit(startLine, startChar, endLine, endChar int, newText string) EditKind {
	return pr.applyEdit(startLine, startChar, endLine, newText, func() string {
		startOff := posOffset(pr.Content, startLine, startChar)
		endOff := posOffset(pr.Content, endLine, endChar)

		return pr.Content[:startOff] + newText + pr.Content[endOff:]
	})
}

// ApplyEditResult is ApplyEdit for a caller that has already applied the edit
// to its own copy of the document: edited is the whole text after the edit, and
// becomes pr.Content as it is.
//
// The server keeps the document text and this ParseResult side by side and
// used to build the edited text once for each — two full copies of the
// document allocated per keystroke, and two retained for as long as it is
// open. Taking the caller's string makes them one allocation and one copy. It
// also makes the caller's text the one the parse sees, which is the right way
// round: the server's copy is the document, and after a burst of rapid changes
// it is ahead of Content until the deferred full replace lands.
func (pr *ParseResult) ApplyEditResult(startLine, startChar, endLine, _ int, newText, edited string) EditKind {
	return pr.applyEdit(startLine, startChar, endLine, newText, func() string { return edited })
}

func (pr *ParseResult) applyEdit(startLine, startChar, endLine int, newText string, edited func() string) (kind EditKind) {
	defer func() {
		if r := recover(); r != nil {
			log.Recovered(pr.log, "parse panic in ApplyEdit", r, "uri", string(pr.URI))
			pr.reparseShallow()

			kind = EditGlobal
		}
	}()

	start := time.Now()

	oldLines := endLine - startLine
	newLines := strings.Count(newText, "\n")
	delta := newLines - oldLines

	// Determine if edit is inside a function body (before modifying content)
	funcIdx := pr.funcContaining(startLine, endLine, startChar)

	old := pr.Content

	startOff := -1
	if funcIdx >= 0 {
		startOff = pr.offsetAt(startLine, startChar)
	}

	// Apply the text edit
	pr.Content = edited()

	if funcIdx >= 0 {
		// Edit is inside a function body — just shift and invalidate
		removed := len(old) - len(pr.Content) + len(newText)
		pr.editLineStarts(old, startOff, removed, newText)
		pr.shiftSplit(old, startOff, removed, newText)
		pr.shiftAfter(startLine, delta)
		pr.InvalidateFunc(pr.Scopes[funcIdx].Start, pr.Scopes[funcIdx].End)
		pr.resetGlobalCaches()
		pr.logDebug("applyEdit", "uri", string(pr.URI), "kind", "inFunc", "line", startLine, "funcStart", pr.Scopes[funcIdx].Start, "funcEnd", pr.Scopes[funcIdx].End, "dur", time.Since(start))

		return EditInFunc
	}

	// Edit is in global scope — need to re-parse signatures
	pr.reparseShallow()
	pr.logDebug("applyEdit", "uri", string(pr.URI), "kind", "global", "dur", time.Since(start))

	return EditGlobal
}

// EditInFunction reports whether an edit of the given range falls inside a
// function body, which ApplyEditResult handles by shifting and invalidating
// that one function. Anything else costs a reparse of the whole file's
// signatures, which is what a caller may prefer to defer.
func (pr *ParseResult) EditInFunction(startLine, endLine, startChar int) bool {
	return pr.funcContaining(startLine, endLine, startChar) >= 0
}

// ApplyFullReplace replaces the entire content and re-parses shallowly.
func (pr *ParseResult) ApplyFullReplace(content string) {
	pr.Content = content
	pr.reparseShallow()
}

// funcContaining returns the index into pr.Scopes of the function whose body
// contains the edit range, or -1 if the edit spans outside all functions.
func (pr *ParseResult) funcContaining(startLine, endLine, startChar int) int {
	for i, s := range pr.Scopes {
		if startLine > s.Start && endLine < s.End {
			return i
		}
		// Edit is on the closing line — check if it's before the closing tag/brace
		if startLine > s.Start && endLine == s.End {
			closeCol := pr.closingTokenCol(s.End)
			if closeCol >= 0 && startChar <= closeCol {
				return i
			}
		}
	}

	return -1
}

// closingTokenCol returns the column of the closing token (} or </cffunction>)
// on the given line, or -1 if not found. The column is in UTF-16 code units,
// since the edit column funcContaining compares it against is an LSP one.
func (pr *ParseResult) closingTokenCol(line int) int {
	lineStart := 0
	for range line {
		idx := strings.IndexByte(pr.Content[lineStart:], '\n')
		if idx < 0 {
			return -1
		}

		lineStart += idx + 1
	}

	lineEnd := strings.IndexByte(pr.Content[lineStart:], '\n')
	if lineEnd < 0 {
		lineEnd = len(pr.Content) - lineStart
	}

	lineText := pr.Content[lineStart : lineStart+lineEnd]

	// Check for </cffunction (tag-based)
	if idx := indexCFTag(lineText, "/cffunction"); idx >= 0 {
		return lex.UTF16Len(lineText[:idx])
	}
	// Check for closing brace (script-based)
	if before, _, ok := strings.CutLast(lineText, "}"); ok {
		return lex.UTF16Len(before)
	}

	return -1
}

// shiftAfter adjusts line numbers for scopes, funcs, and refs after editLine.
func (pr *ParseResult) shiftAfter(editLine, delta int) {
	if delta == 0 {
		return
	}

	// Keyed by function and line, both of which just moved. They are read
	// only when a call is checked, which works from a fresh parse.
	pr.memberSets = nil
	// Scopes move in place below, so their index is rebuilt with the rest.
	pr.resetNameIndexes()

	for i := range pr.Scopes {
		if pr.Scopes[i].Start > editLine {
			pr.Scopes[i].Start += delta
			pr.Scopes[i].End += delta
		} else if pr.Scopes[i].End >= editLine {
			pr.Scopes[i].End += delta
		}
	}

	for i := range pr.Funcs {
		if int(pr.Funcs[i].Line) > editLine {
			pr.Funcs[i].Line = conv.Uint32(int(pr.Funcs[i].Line) + delta)
		}
	}

	for i := range pr.ComponentRefs {
		if int(pr.ComponentRefs[i].Line) > editLine {
			pr.ComponentRefs[i].Line = conv.Uint32(int(pr.ComponentRefs[i].Line) + delta)
		}
	}

	// What the walk filed per function moves with its function, so a body
	// the edit did not touch keeps the parse's answers.
	pr.shiftBodies(editLine, delta)
}

// reparseShallow re-runs the shallow parse on the current content,
// replacing Funcs, Refs, Scopes, and Regions. Clears all caches.
func (pr *ParseResult) reparseShallow() {
	start := time.Now()
	pr.Regions, pr.contentLineIdx = pr.classifyRegions()
	pr.regionsOf = pr.Content
	pr.outputScanned = false
	pr.Accessors = false
	pr.Persistent = false
	pr.Funcs = pr.Funcs[:0]
	pr.ComponentRefs = pr.ComponentRefs[:0]
	pr.Scopes = pr.Scopes[:0]
	// The calls are recorded afresh like the rest. Kept, the last parse's
	// top-level calls were listed again after every reparse, each copy
	// keeping its version of the document alive.
	pr.Calls = nil
	pr.memberSets = nil
	// The per-function answers are dropped before the walk files its own:
	// what a function's walk files is appended to its entry.
	pr.resetFuncCaches()
	pr.extractSignatures()
	pr.resetGlobalCaches()
	pr.resetNameIndexes()
	pr.logDebug("reparseShallow", "uri", string(pr.URI), "funcs", len(pr.Funcs), "dur", time.Since(start))
}

// resetGlobalCaches resets the lazily-computed global/variables/this var caches.
// resetFuncCaches drops every per-function result, before a reparse files its
// walk's afresh: FuncVars/FuncRefs/FuncCalls/FuncLinks then answer from it.
//
// Only funcVars used to be cleared here. The other three kept serving pre-edit
// answers after a full replace — which the rapid-change path takes on every
// large paste — so a hover could report the component a variable held before
// the edit, and the server would faithfully re-index that stale ref. The keys
// are line ranges, so they also silently point at the wrong function once lines
// move; clearing is the only correct response to content that has been
// wholesale replaced. They were cleared after the reparse, which dropped what
// its walk had filed, and every function was then read again on its own.
func (pr *ParseResult) resetFuncCaches() {
	pr.funcVarsMu.Lock()
	pr.funcVars = make(map[string][]string)
	pr.funcVarsMu.Unlock()

	pr.funcRefsMu.Lock()
	pr.funcRefsMap = nil
	pr.funcLinksMap = nil
	pr.funcCallsMap = nil
	pr.funcRefsMu.Unlock()

	pr.resetNameIndexes()
}

// resetNameIndexes drops the lookups built over Scopes, ComponentRefs, the
// calls, memberSets and FuncVars, each under its own lock. They check the
// slice they were built from, but a reparse refills the same arrays and an
// edit shifts lines in place, which that check cannot see.
func (pr *ParseResult) resetNameIndexes() {
	pr.memberSetsMu.Lock()
	pr.memberSetsByName = nil
	pr.memberSetsMu.Unlock()

	pr.refsMu.Lock()
	pr.refsByName = nil
	pr.refsMu.Unlock()

	pr.dropCallsIndex()

	pr.scopeAtMu.Lock()
	pr.scopeAt = nil
	pr.scopeAtMu.Unlock()

	pr.funcVarsMu.Lock()
	pr.funcVarSets = nil
	pr.funcVarsMu.Unlock()
}

func (pr *ParseResult) resetGlobalCaches() {
	pr.mu.Lock()
	// A writer in any function can affect a whole returned collection.
	// Rebuild these contracts on the next full parse, rather than retaining
	// a contract computed from the previous file contents.
	for i := range pr.Funcs {
		pr.Funcs[i].CollectionValues = nil
	}

	pr.scopedDone = false
	pr.allVarsDone = false
	pr.globalVars = nil
	pr.variablesVars = nil
	pr.thisVars = nil
	pr.mu.Unlock()

	// Every edit comes through here, including one inside a function that
	// moves no line, which resets no name index.
	pr.memoMu.Lock()
	pr.memo = nil
	pr.memoMu.Unlock()
}

// posOffset converts an LSP line/character position to a byte offset.
// editLineStarts carries the line table of old, the text before an edit, over
// to Content, the text after it: the edit replaced removed bytes at off with
// inserted. A table that was not old's is left to be built when asked for.
func (pr *ParseResult) editLineStarts(old string, off, removed int, inserted string) {
	pr.lineStartsMu.Lock()
	defer pr.lineStartsMu.Unlock()

	if pr.lineStarts == nil || off < 0 || pr.lineStartsContent != old {
		return
	}

	pr.lineStarts, pr.lineStartsContent = pr.lineStarts.Edited(off, removed, inserted), pr.Content
}

// offsetAt is PositionToOffset of Content, found from its line table.
func (pr *ParseResult) offsetAt(line, char int) int {
	start, ok := pr.lineStartsTable().Start(line)
	if !ok {
		return -1
	}

	return start + PositionToOffset(pr.Content[start:], 0, char)
}

func posOffset(content string, line, char int) int {
	return PositionToOffset(content, line, char)
}
