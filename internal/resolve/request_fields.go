package resolve

import (
	"strings"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// onlyFileWritesRequest reports whether, among every file a batch scan
// reads, file is the only one that writes request.<name>: by assignment,
// through a bracketed key, structInsert or cfparam. Outside a batch scan the
// workspace's files are not all known, and nothing is claimed.
func (r *Resolver) onlyFileWritesRequest(name, file string) bool {
	o := r.owner()
	if len(o.InferArgsFiles) == 0 {
		return false
	}

	for _, writer := range r.requestWriters()[strings.ToLower(name)] {
		if !cfpath.SamePath(writer, file) {
			return false
		}
	}

	return true
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

		for _, key := range requestWriteKeys(text) {
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

// requestWriteKeys is the lowercased request keys content writes, each once.
func requestWriteKeys(content string) []string {
	counts := requestWriteCounts(content)
	keys := make([]string, 0, len(counts))

	for key := range counts {
		keys = append(keys, key)
	}

	return keys
}

// requestWriteCounts is how many writes of each lowercased request key content
// holds: `request.k = v` (not `==`), `request[ "k" ] = v`, `structInsert(
// request, "k"` and a cfparam's `name="request.k"`, case-insensitively, as the
// requestWriteRe this replaced matched them. One pass per form over the text,
// which is searched case-insensitively rather than lowercased whole; only the
// keys found are lowercased.
func requestWriteCounts(content string) map[string]int {
	counts := map[string]int{}
	add := func(key string) {
		if key != "" {
			counts[strings.ToLower(key)]++
		}
	}

	for at := 0; ; {
		i := parser.IndexFoldFrom(content, "request", at)
		if i < 0 {
			break
		}

		at = i + len("request")

		add(requestKeyWrittenAt(content, i, at))
	}

	for at := 0; ; {
		i := parser.IndexFoldFrom(content, "structinsert", at)
		if i < 0 {
			break
		}

		at = i + len("structinsert")

		j := skipSpace(content, at)
		if j >= len(content) || content[j] != '(' {
			continue
		}

		j = skipSpace(content, j+1)
		if !parser.HasPrefixFold(content[j:], "request") {
			continue
		}

		j = skipSpace(content, j+len("request"))
		if j >= len(content) || content[j] != ',' {
			continue
		}

		if key, _, ok := quotedAt(content, skipSpace(content, j+1)); ok {
			add(key)
		}
	}

	for at := 0; ; {
		i := parser.IndexFoldFrom(content, "name", at)
		if i < 0 {
			break
		}

		at = i + len("name")

		j := skipSpace(content, at)
		if j >= len(content) || content[j] != '=' {
			continue
		}

		value, _, ok := quotedAt(content, skipSpace(content, j+1))
		if ok && parser.HasPrefixFold(value, "request.") {
			add(value[len("request."):])
		}
	}

	return counts
}

// requestKeyWrittenAt is the key a write starting at content[start:]
// ("request" ending at end) assigns, or "": `request . k =` or `request [ "k"
// ] =`, with the `=` not the first of `==`.
func requestKeyWrittenAt(content string, start, end int) string {
	if start > 0 && isWordByte(content[start-1]) {
		return ""
	}

	j := skipSpace(content, end)
	if j >= len(content) {
		return ""
	}

	var key string

	switch content[j] {
	case '.':
		j = skipSpace(content, j+1)
		k := j

		for k < len(content) && isWordByte(content[k]) {
			k++
		}

		key, j = content[j:k], k
	case '[':
		var ok bool
		if key, j, ok = quotedAt(content, skipSpace(content, j+1)); !ok {
			return ""
		}

		j = skipSpace(content, j)
		if j >= len(content) || content[j] != ']' {
			return ""
		}

		j++
	default:
		return ""
	}

	j = skipSpace(content, j)
	if j+1 >= len(content) || content[j] != '=' || content[j+1] == '=' {
		return ""
	}

	return key
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

func skipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\f' || s[i] == '\v') {
		i++
	}

	return i
}

func isWordByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
