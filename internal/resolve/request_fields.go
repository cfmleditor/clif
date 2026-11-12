package resolve

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// requestWrittenBy reports whether the file at file is among the files
// writing request key name, in a batch scan that knows every file. The other
// writers must agree (otherRequestWritersAgree).
func (r *Resolver) requestWrittenBy(name, file string) bool {
	if len(r.owner().InferArgsFiles) == 0 {
		return false
	}

	return slices.ContainsFunc(r.requestWriters()[strings.ToLower(name)], func(writer string) bool { return cfpath.SamePath(writer, file) })
}

// requestWriters is, for every request key some file writes, the files that
// write it, read in one pass over the batch's files. It replaced a scan of
// every file per key, each with a regular expression compiled for that key,
// which was the largest regex cost in a MuraCMS scan.
func (r *Resolver) requestWriters() map[string][]string {
	o := r.owner()

	o.mu.RLock()
	writers := o.requestWriterIdx
	o.mu.RUnlock()

	if writers != nil {
		return writers
	}

	writers = map[string][]string{}

	for _, path := range o.InferArgsFiles {
		text, err := r.readText(path)
		if err != nil {
			continue
		}

		for key := range requestWritesIn(path, text) {
			writers[key] = append(writers[key], path)
		}
	}

	o.mu.Lock()
	if o.requestWriterIdx == nil {
		o.requestWriterIdx = writers
	}

	writers = o.requestWriterIdx
	o.mu.Unlock()

	return writers
}

// requestWritesIn is how many writes of each lowercased request key the
// parse reads in content, the text of file (requestWriteCounts). The calls
// are the plain parse's, made only when the text names structInsert.
func requestWritesIn(file, content string) map[string]int {
	var inserts []parsedCall
	if mentions(content, "structInsert") {
		inserts = parsedCalls(plainParseText(file, content), "structInsert")
	}

	return requestWriteCounts(parser.AssignmentsOf(file, content), inserts)
}

// requestWriteCounts is how many writes of each lowercased request key the
// parse read, in as and in the structInsert calls inserts: `request.k = v`
// and `request[ "k" ] = v` (with any assigning operator, ++ and --, or as a
// for-in loop's variable), an attribute or named argument naming
// "request.k" as the variable it fills (cfparam's name, cfquery's name,
// cfsavecontent's variable, …), and `structInsert( request, "k", v )`. A
// write in a comment, in markup or in a string is none, as the parse reads
// them; a member of a key (request.k.x) writes no key.
func requestWriteCounts(as []parser.Assignment, inserts []parsedCall) map[string]int {
	counts := map[string]int{}
	add := func(key string) {
		if key != "" {
			counts[strings.ToLower(key)]++
		}
	}

	for i := range as {
		a := &as[i]

		switch a.Op {
		case parser.AssignAttribute:
			if a.HasLiteral && parser.HasPrefixFold(a.Literal, "request.") {
				add(a.Literal[len("request."):])
			}
		case parser.AssignSet, parser.AssignCompound, parser.AssignStep, parser.AssignForIn:
			add(requestKeyOf(a.Target))
		default:
		}
	}

	for _, c := range inserts {
		pieces := c.pieces()
		if len(pieces) < 2 || len(pieces[0]) != 1 || !isWord(pieces[0][0], "request") {
			continue
		}

		if key, ok := soleString(pieces[1]); ok {
			add(key)
		}
	}

	return counts
}

// requestKeyOf is the request key a write's target names, as
// parser.Assignment spells it: k for request.k or request[ "k" ]; "" for
// any other target.
func requestKeyOf(target string) string {
	steps := targetSteps(target)
	if len(steps) != 2 || !strings.EqualFold(steps[0], "request") {
		return ""
	}

	key := steps[1]
	if !strings.HasPrefix(key, "[") {
		return key
	}

	inner := strings.TrimSpace(key[1 : len(key)-1])
	if lit, _, ok := quotedAt(inner, 0); ok && len(lit)+2 == len(inner) {
		return lit
	}

	return ""
}

// quotedAt is the text of a single- or double-quoted string starting at i,
// and the index past its closing quote.
func quotedAt(s string, i int) (string, int, bool) {
	if i >= len(s) || s[i] != '"' && s[i] != '\'' {
		return "", i, false
	}

	end := strings.IndexByte(s[i+1:], s[i])
	if end < 0 {
		return "", i, false
	}

	return s[i+1 : i+1+end], i + end + 2, true
}

// withoutComments is content with every byte outside the scanner's tokens
// blanked, newlines kept: comments of all three forms go, and strings stay
// whole. A write in a comment writes nothing, and FW/1's Application.cfc
// keeps two commented-out alternatives beside the write it makes.
func withoutComments(content string) string {
	if !strings.Contains(content, "//") && !strings.Contains(content, "/*") && !strings.Contains(content, "<!---") {
		return content
	}

	b := []byte(content)
	keep := make([]bool, len(b))

	// Walked, not tokenized into a list through allTokens: the list was used
	// once and thrown away, and passing whole files through the shared memo
	// evicted the short windows it is there for.
	sc := parser.NewSourceScanner(content)
	for t := sc.NextSkipComments(); t.Kind != parser.TokEOF; t = sc.NextSkipComments() {
		for i := t.Offset; i < t.Offset+len(t.Value) && i < len(keep); i++ {
			keep[i] = true
		}
	}

	for i := range b {
		if !keep[i] && b[i] != '\n' {
			b[i] = ' '
		}
	}

	return string(b)
}

// otherApplication reports whether file and writer are the Application.cfc
// of two different applications, neither extending the other: a request runs
// one of them, and the other's code never runs in it.
func (r *Resolver) otherApplication(file, writer string) bool {
	if !strings.EqualFold(filepath.Base(file), "Application.cfc") || !strings.EqualFold(filepath.Base(writer), "Application.cfc") || cfpath.SamePath(filepath.Dir(file), filepath.Dir(writer)) {
		return false
	}

	return !r.extendsFile(file, writer) && !r.extendsFile(writer, file)
}

// extendsFile reports whether the component at from extends the one at
// target, directly or through its chain. A chain that cannot be followed to
// its end is taken to reach it.
func (r *Resolver) extendsFile(from, target string) bool {
	seen := map[string]bool{}

	for path, depth := from, 0; depth < 16; depth++ {
		if seen[pathKey(path)] {
			return false
		}

		seen[pathKey(path)] = true

		ext, ok := r.extendsOf(path, cfpath.ToURI(path))
		if !ok {
			return true
		}

		if ext == "" {
			return false
		}

		next := r.ComponentPath(ext, filepath.Dir(path))
		if next == "" {
			return true
		}

		if cfpath.SamePath(next, target) {
			return true
		}

		path = next
	}

	return true
}
