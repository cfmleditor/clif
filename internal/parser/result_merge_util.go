package parser

import (
	"strings"
)

// enclosingTagScope is the tag function a region starting at startLine opens
// inside of: one whose opening <cffunction> is in an earlier region, the file
// having been split around a nested <cfscript>.
func enclosingTagScope(tagScopes []FuncScope, startLine int) (FuncScope, bool) {
	for _, s := range tagScopes {
		if s.Start < startLine && startLine <= s.End {
			return s, true
		}
	}

	return FuncScope{}, false
}

// argsOfFuncAt is the argument list of the first function declared at line.
func (pr *ParseResult) argsOfFuncAt(line int) []Argument {
	for i := range pr.Funcs {
		if int(pr.Funcs[i].Line) == line {
			return pr.Funcs[i].Arguments
		}
	}

	return nil
}

// parseFnKey is the key String wrote as s, for the collection readers, which
// intern a function's key as text with the names declared in it.
func parseFnKey(s string) fnKey {
	start, end, ok := strings.Cut(s, ":")
	if !ok {
		return noFn
	}

	return funcKey(atoi(start), atoi(end))
}

// shiftFuncKey moves a function key by delta lines.
func shiftFuncKey(k fnKey, delta int) fnKey {
	if !k.in {
		return k
	}

	return funcKey(int(k.start)+delta, int(k.end)+delta)
}

// appendSlice returns src directly when dst is empty, avoiding an extra copy
// for single-region files where src is already an owned slice.
func appendSlice[T any](dst, src []T) []T {
	if len(src) == 0 {
		return dst
	}

	if len(dst) == 0 {
		return src
	}

	return append(dst, src...)
}

// mergeKeyedMap moves src directly into dst when dst is nil, otherwise appends
// each entry through appendKeyed.
func mergeKeyedMap[T any](dst, src map[fnKey][]T) map[fnKey][]T {
	if len(src) == 0 {
		return dst
	}

	if dst == nil {
		return src
	}

	for k, v := range src {
		dst = appendKeyed(dst, k, v)
	}

	return dst
}

// appendKeyed appends v to m[k], making the map on first use, and returns it.
func appendKeyed[T any](m map[fnKey][]T, k fnKey, v []T) map[fnKey][]T {
	if len(v) == 0 {
		return m
	}

	if m == nil {
		m = make(map[fnKey][]T)
	}

	if cur := m[k]; len(cur) == 0 {
		m[k] = v
	} else {
		m[k] = append(cur, v...)
	}

	return m
}
