package resolve

import (
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/config"
	"github.com/cfmleditor/clif/internal/docs"
	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
)

// Which template a custom tag call runs. The engine decides at run time, from
// where the caller is and the custom tag paths in force; these are the paths
// the source and the configuration state, searched in the engine's order:
//
//	<cf_name>, cf_name()   the caller's own directory, then the custom tag
//	                       roots: the Application file's this.customTagPaths,
//	                       the customTagPaths config entries without a prefix,
//	                       and the server CFConfig's customTagMappings
//	<cfmodule name="a.b">  a/b under the custom tag roots, not the caller's
//	                       directory
//	<cfmodule template>    the path, found as a cfinclude's is
//	<p:name>               name under the cfimport taglib, found as a
//	                       cfinclude's path is
//	<cfabridge>            bridge under each customTagPaths entry whose
//	                       prefix is "cfa"; never a built-in tag
//
// A name is tried as .cfm, then .cfc (Lucee runs a component as a custom
// tag), then .cfml, and matched without case: the engines run on
// case-insensitive filesystems as often as not. A root is searched
// recursively when its config entry says so, or the server's CFConfig sets
// customTagDeepSearch; the shallowest match wins, then the first by name.

// customTagExtensions are the file types a custom tag name is tried as.
var customTagExtensions = []string{".cfm", ".cfc", ".cfml"}

// maxTagSearchDirs bounds a recursive search of one custom tag root.
const maxTagSearchDirs = 4096

// recursiveMissTTL is how long tagFileIn trusts that a recursive search of a
// root found nothing. A search that lists only the directories named holds
// while none of them changes (tagFileEntry.listed); a recursive one may list
// thousands, too many to check on every request, and is redone this often.
// A tag directory can sit outside the workspace, or outside the files the
// editor reports changes to, so no event says when a template appears in it.
const recursiveMissTTL = time.Minute

// tagFileEntry is one tagFileIn answer: the template, or "" with what makes
// the miss stale, the directories the search listed and when.
type tagFileEntry struct {
	file   string
	listed []dirStamp
	missed time.Time
}

// dirStamp is a directory as a search listed it: an entry added to or removed
// from it moves its modification time.
type dirStamp struct {
	dir string
	mod time.Time
}

// CustomTagTemplate is the file the custom tag call c, made in fromFile, runs,
// or "" when it names none the source and configuration can find.
func (r *Resolver) CustomTagTemplate(c *parser.CustomTagCall, fromFile string) string {
	if c.Dynamic || strings.TrimSpace(c.Name) == "" {
		return ""
	}

	switch c.Kind {
	case parser.CustomTagModuleTemplate:
		return r.IncludePath(c.Name, fromFile)
	case parser.CustomTagUnderscore:
		if f := r.tagFileIn(filepath.Dir(fromFile), c.Name, false); f != "" {
			return f
		}

		for _, root := range r.customTagRoots(fromFile) {
			if f := r.tagFileIn(root.Dir, c.Name, root.Recursive); f != "" {
				return f
			}
		}
	case parser.CustomTagModuleName:
		rel := strings.ReplaceAll(strings.Trim(c.Name, "."), ".", "/")

		for _, root := range r.customTagRoots(fromFile) {
			if f := r.tagFileIn(root.Dir, rel, false); f != "" {
				return f
			}
		}
	case parser.CustomTagImported:
		if strings.Contains(c.Taglib, "#") {
			return ""
		}

		lib := strings.TrimRight(filepath.ToSlash(c.Taglib), "/")
		for _, ext := range customTagExtensions {
			if f := r.IncludePath(lib+"/"+c.Name+ext, fromFile); f != "" {
				return f
			}
		}
	case parser.CustomTagPrefixed:
		if _, builtin := docs.LookupTag(c.Prefix + c.Name); builtin {
			return ""
		}

		for _, p := range r.CustomTagPaths {
			if p.Prefix != c.Prefix {
				continue
			}

			if f := r.tagFileIn(p.Directory, c.Name, p.Recursive); f != "" {
				return f
			}
		}
	}

	return ""
}

// CustomTagPrefixes are the configured tag library prefixes, for
// parser.CustomTagCalls.
func (r *Resolver) CustomTagPrefixes() []string {
	return config.CustomTagPrefixes(r.CustomTagPaths)
}

// customTagRoots are the directories a <cf_name> made in fromFile is looked
// for in after its own directory, in order. They depend on the directory
// alone, and are kept by it until InvalidatePaths: a page calls many tags.
func (r *Resolver) customTagRoots(fromFile string) []cfpath.TagRoot {
	baseDir := filepath.Dir(fromFile)

	r.mu.RLock()
	roots, ok := r.tagRoots[baseDir]
	r.mu.RUnlock()

	if ok {
		return roots
	}

	roots = r.findTagRoots(baseDir)

	r.mu.Lock()
	if r.tagRoots == nil {
		r.tagRoots = map[string][]cfpath.TagRoot{}
	}

	r.tagRoots[baseDir] = roots
	r.mu.Unlock()

	return roots
}

func (r *Resolver) findTagRoots(baseDir string) []cfpath.TagRoot {
	appDir := r.FindApplicationRoot(baseDir)
	if appDir == "" {
		appDir = baseDir
	}

	server, deep := cfpath.ServerCustomTagRoots(appDir)

	var out []cfpath.TagRoot

	for _, dir := range cfpath.LoadAppCustomTagPaths(appDir, r.Mappings) {
		out = append(out, cfpath.TagRoot{Dir: r.tagRootDir(dir, appDir), Recursive: deep})
	}

	for _, p := range r.CustomTagPaths {
		if p.Prefix == "" {
			out = append(out, cfpath.TagRoot{Dir: p.Directory, Recursive: p.Recursive || deep})
		}
	}

	return append(out, server...)
}

// tagRootDir is dir, an Application file's custom tag path, as a directory
// that exists. expandPath("/tags") resolves against the web root and the
// mappings, not the filesystem root, and reads back as "/tags": that is
// tried under the application's directory and a mapping its first segment
// names.
func (r *Resolver) tagRootDir(dir, appDir string) string {
	if r.isDir(dir) {
		return dir
	}

	rel := strings.TrimPrefix(filepath.ToSlash(dir), "/")
	if rel == "" || filepath.VolumeName(dir) != "" {
		return dir
	}

	if c := filepath.Join(appDir, filepath.FromSlash(rel)); r.isDir(c) {
		return c
	}

	if seg, rest, _ := strings.Cut(rel, "/"); seg != "" {
		for key, root := range r.EffectiveMappings(appDir) {
			if strings.EqualFold(strings.Trim(key, "/"), seg) {
				if c := filepath.Join(root, filepath.FromSlash(rest)); r.isDir(c) {
					return c
				}
			}
		}
	}

	return dir
}

func (r *Resolver) isDir(dir string) bool {
	info, err := r.fs().Stat(dir)

	return err == nil && info.IsDir()
}

// tagFileIn is the template rel (a name, or a slash path for a dotted
// cfmodule name) names in dir, or under it when recursive, matched without
// case; "" for none. A template found is kept while it exists; a miss while
// the directories searched are unchanged, or for recursiveMissTTL after a
// recursive search (for good in a batch scan, whose files do not change
// under it). InvalidatePaths drops both.
func (r *Resolver) tagFileIn(dir, rel string, recursive bool) string {
	key := dir + "\x00" + strings.ToLower(rel)
	if recursive {
		key += "\x00r"
	}

	r.mu.RLock()
	e, ok := r.tagFiles[key]
	r.mu.RUnlock()

	if ok && r.tagEntryHolds(&e, recursive) {
		return e.file
	}

	var listed []dirStamp

	e = tagFileEntry{file: r.findTagFile(dir, rel, &listed)}

	if e.file == "" && recursive {
		e.file = r.findTagFileBelow(dir, rel)
	}

	if e.file == "" {
		e.listed, e.missed = listed, time.Now()
	}

	r.mu.Lock()
	if r.tagFiles == nil {
		r.tagFiles = map[string]tagFileEntry{}
	}

	r.tagFiles[key] = e
	r.mu.Unlock()

	return e.file
}

// tagEntryHolds reports whether a kept tagFileIn answer still stands.
func (r *Resolver) tagEntryHolds(e *tagFileEntry, recursive bool) bool {
	if e.file != "" {
		_, err := r.fs().Stat(e.file)

		return err == nil
	}

	if r.SnapshotScan {
		return true
	}

	if recursive && time.Since(e.missed) >= recursiveMissTTL {
		return false
	}

	for _, d := range e.listed {
		info, err := r.fs().Stat(d.dir)
		if err != nil || !info.ModTime().Equal(d.mod) {
			return false
		}
	}

	return true
}

// findTagFile is rel's template directly in dir, each segment matched
// without case. Each directory it lists is added to listed when listed is
// not nil, with its modification time.
func (r *Resolver) findTagFile(dir, rel string, listed *[]dirStamp) string {
	segments := strings.Split(rel, "/")

	for _, seg := range segments[:len(segments)-1] {
		next := r.entryNamed(dir, listed, func(name string, isDir bool) bool { return isDir && strings.EqualFold(name, seg) })
		if next == "" {
			return ""
		}

		dir = next
	}

	last := segments[len(segments)-1]

	for _, ext := range customTagExtensions {
		if f := r.entryNamed(dir, listed, func(name string, isDir bool) bool { return !isDir && strings.EqualFold(name, last+ext) }); f != "" {
			return f
		}
	}

	return ""
}

// entryNamed is the first entry of dir, by name, that match accepts. dir is
// added to listed, as findTagFile says.
func (r *Resolver) entryNamed(dir string, listed *[]dirStamp, match func(name string, isDir bool) bool) string {
	if listed != nil {
		// Stamped before the listing, so an entry added while it runs makes
		// the stamp stale rather than the miss permanent.
		if info, err := r.fs().Stat(dir); err == nil {
			if n := len(*listed); n == 0 || (*listed)[n-1].dir != dir {
				*listed = append(*listed, dirStamp{dir: dir, mod: info.ModTime()})
			}
		} else {
			// A directory that is not there is stamped as missing: one made
			// later is then a change.
			*listed = append(*listed, dirStamp{dir: dir})
		}
	}

	entries, err := r.fs().ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		if match(e.Name(), e.IsDir()) {
			return filepath.Join(dir, e.Name())
		}
	}

	return ""
}

// findTagFileBelow is rel's template in a subdirectory of dir, breadth first
// so the shallowest wins, each level in name order.
func (r *Resolver) findTagFileBelow(dir, rel string) string {
	level, seen := []string{dir}, 0

	for len(level) > 0 && seen < maxTagSearchDirs {
		var next []string

		for _, d := range level {
			entries, err := r.fs().ReadDir(d)
			if err != nil {
				continue
			}

			for _, e := range entries {
				if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
					next = append(next, filepath.Join(d, e.Name()))
				}
			}
		}

		slices.Sort(next)

		for _, d := range next {
			seen++
			if seen > maxTagSearchDirs {
				return ""
			}

			if f := r.findTagFile(d, rel, nil); f != "" {
				return f
			}
		}

		level = next
	}

	return ""
}
