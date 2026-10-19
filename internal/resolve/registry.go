package resolve

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/cfmleditor/clif/internal/parser"
	cfpath "github.com/cfmleditor/clif/internal/path"
	"github.com/cfmleditor/clif/internal/vfs"
)

// A Lucee admin template fills a struct with the components of a package:
//
//	driverNames = ComponentListPackageAsStruct("lucee.admin.mailservers", driverNames);
//	loop collection=driverNames index="name" item="componentPath" {
//		if(name == 'MailServer') continue;
//		drivers[name] = createObject("component", componentPath);
//	}
//
// Every call on an element of that struct is a call on one of the package's
// components — driver.getShortName() with the driver under whichever name the
// form chose. The family is the package directory's files, less the names the
// template's own guards compare the listing key against: the abstract base
// and the Field/Group helpers. A method is accepted only when every family
// member declares it, by its file or its extends chain, since the receiver is
// any one of them.
//
// The packages name the components under core/src/main/java/resource, while
// the templates run in core/src/main/cfml — the two halves of one deployed
// admin tree in a Lucee checkout. The recipe file's own path and both halves
// stat'ed beside it tell the family's directory. Nothing configured takes
// part, so a layout that is not this checkout's is untouched.

const familyPrefix = "$family:"

type registryFact struct {
	pkg       string   // the listing literal, for a reason's display
	container string   // the struct the writes fill, lowercased, without variables.
	listing   string   // the struct the listing merges into, lowercased, without variables.
	members   []string // absolute .cfc paths of the family, sorted
}

var (
	// ComponentListPackageAsStruct("lucee.admin.debug", driverNames): a listing
	// literal and the struct it merges into.
	pkgListRe = regexp.MustCompile(`(?i)\bComponentListPackageAsStruct\s*\(\s*["']([^"']*)["']\s*,\s*((?:variables\.)?[A-Za-z_]\w*)\s*\)`)

	// <cfloop ...> and </cfloop>, for a listing loop's span. Group 1 is the
	// close tag's slash, group 2 is the open tag's attribute text.
	loopTagPartsRe = regexp.MustCompile(`(?is)<(/?)cfloop\b([^>]*)>`)

	// Lucee's script loop statement: loop collection=driverNames ... { .
	// Group 1 is the attribute text; the opening brace closes the match.
	scriptLoopPartsRe = regexp.MustCompile(`(?is)\bloop\b([^\{;]*?)\{`)

	// The header's attributes, quoted or bare.
	loopAttrPairsRe = regexp.MustCompile(`(?i)\b(collection|struct|array|index|item|list)\s*=\s*(?:"([^"]*)"|'([^']*)'|([A-Za-z_][\w]*))`)

	// A compare of the listing key with a name. Group 2 is the operator:
	// equal to a name and skipping keeps it out, as does unequal to one and
	// constructing; registrySkips reads which.
	skipCompareRe = regexp.MustCompile(`(?i)\b([A-Za-z_]\w*)\s*(===?|!==|!=|\bEQ\b|\bNEQ\b)\s*["']([^"']+)["']`)

	// R[key] = createObject("component", V), written right on the store.
	directStoreRe = regexp.MustCompile(`(?i)((?:variables\.)?[A-Za-z_]\w*)\s*\[[^\]]*\]\s*=\s*createObject\s*\(\s*["']\s*component\s*["']\s*,\s*([^)]*?)\s*\)`)

	// R[key] = V: the store of a variable into the registry.
	registryStoreRe = regexp.MustCompile(`(?i)((?:variables\.)?[A-Za-z_]\w*)\s*\[[^\]]*\]\s*=\s*([A-Za-z_]\w*)\s*;?`)

	// V = createObject("component", path): the variable the construction was
	// assigned to, and the path it spells.
	createdAssignRe = regexp.MustCompile(`(?i)\b([A-Za-z_]\w*)\s*=\s*createObject\s*\(\s*["']\s*component\s*["']\s*,\s*([^)]*?)\s*\)`)

	// arrayAppend(ARR, createObject("component", V)): the constructions the
	// listing's loop ships into an array a factory call turns into the
	// registry.
	arrayAppendCreateRe = regexp.MustCompile(`(?i)\barrayAppend\s*\(\s*((?:variables\.)?[A-Za-z_]\w*)\s*,\s*createObject\s*\(\s*["']\s*component\s*["']\s*,\s*([^)]*?)\s*\)`)

	// arrayAppend(ARR, V), where V is the variable the construction was
	// assigned to.
	arrayAppendVarRe = regexp.MustCompile(`(?i)\barrayAppend\s*\(\s*((?:variables\.)?[A-Za-z_]\w*)\s*,\s*([A-Za-z_]\w*)\s*\)`)

	// CONTAINER = FN(ARR ...): the assignment whose right side is the call
	// that turns an appended array into the struct the page reads back.
	factoryCallRe = regexp.MustCompile(`(?i)\b((?:variables\.)?[A-Za-z_]\w*)\s*=\s*([A-Za-z_]\w*)\s*\(\s*((?:variables\.)?[A-Za-z_]\w*)\s*(?:,|\))`)

	// function FN(...): the script definition of a factory function's name
	// and its parameter list.
	factoryDefRe = regexp.MustCompile(`(?i)\bfunction\s+([A-Za-z_]\w*)\s*\(([^)]*)\)`)

	// loop array=arguments.P item="X": the factory body's walk of the array
	// its argument received.
	factoryLoopRe = regexp.MustCompile(`(?i)\bloop\s+array=arguments\.([A-Za-z_]\w*)\s+item="([^"]*)"`)

	// for (X in arguments.P): the factory body's walk, script for-in form.
	factoryForRe = regexp.MustCompile(`(?i)\bfor\s*\(\s*(?:var\s+)?([A-Za-z_]\w*)\s+in\s+arguments\.([A-Za-z_]\w*)\b`)
)

// registriesOf is the registry facts the template's own text holds, read for
// every element read the template makes on one of its containers.
func (r *Resolver) registriesOf(pr *parser.ParseResult) []registryFact {
	if pr == nil || !strings.Contains(pr.Content, "ComponentListPackageAsStruct") {
		return nil
	}

	o := r.owner()

	return o.registryCache.get(&o.mu, pr, func() [][]registryFact {
		return [][]registryFact{r.findRegistries(pr.Content, cfpath.FromURI(string(pr.URI)))}
	})[0]
}

// includeRegistries is the registry facts some includer of the template
// holds: a recipe fills the struct in the template that declares it, and the
// consumer it flows to reads its elements back where it is included. Cached
// per includer path while its size and modification time are what they were
// read at; an includer that never mentions the recipe's listing is
// remembered with no facts, so most lookups end at the stat. The lock is
// held only to read and write the cache, never across the file's I/O.
func (r *Resolver) includeRegistries(file string) []registryFact {
	g := r.includes()
	if g == nil {
		return nil
	}

	o := r.owner()

	var out []registryFact

	for _, key := range g.rev[pathKey(file)] {
		includer := g.paths[key]

		if includer == "" || strings.EqualFold(includer, file) {
			continue
		}

		info, err := o.fs().Stat(includer)
		if err != nil {
			continue
		}

		o.mu.RLock()
		hit, ok := o.registryTexts[pathKey(includer)]
		o.mu.RUnlock()

		if ok && hit.size == info.Size() && hit.modTime.Equal(info.ModTime()) {
			out = append(out, hit.facts...)

			continue
		}

		text, err := o.readText(includer)
		if err != nil {
			continue
		}

		var facts []registryFact
		if mentions(text, "ComponentListPackageAsStruct") {
			facts = r.findRegistries(text, includer)
		}

		o.mu.Lock()
		if o.registryTexts == nil {
			o.registryTexts = map[string]registryText{}
		}

		o.registryTexts[pathKey(includer)] = registryText{size: info.Size(), modTime: info.ModTime(), facts: facts}
		o.mu.Unlock()

		out = append(out, facts...)
	}

	return out
}

// registryText is one includer's registry facts and the stamp of the text
// they were read from.
type registryText struct {
	size    int64
	modTime time.Time
	facts   []registryFact
}

// registryFamily is the fact for container, where the template's own text or
// one includer's holds the recipe that fills it.
func (r *Resolver) registryFamily(container string, pr *parser.ParseResult) (registryFact, bool) {
	name := strings.ToLower(strings.TrimPrefix(container, "variables."))
	if name == "" {
		return registryFact{}, false
	}

	for _, fact := range r.registriesOf(pr) {
		if fact.container == name {
			return fact, true
		}
	}

	if pr == nil || pr.URI == "" {
		return registryFact{}, false
	}

	for _, fact := range r.includeRegistries(cfpath.FromURI(string(pr.URI))) {
		if fact.container == name {
			return fact, true
		}
	}

	return registryFact{}, false
}

// registryListingFamily is the fact of the recipe whose listing merges into
// name: the construction of a listing read — createObject("component",
// listing[key]) — is the recipes' own construction of their listing's keys,
// a member of the family the recipes' guards serve.
func (r *Resolver) registryListingFamily(name string, pr *parser.ParseResult) (registryFact, bool) {
	name = strings.ToLower(strings.TrimPrefix(name, "variables."))

	for _, fact := range r.registriesOf(pr) {
		if fact.listing == name {
			return fact, true
		}
	}

	if pr == nil || pr.URI == "" {
		return registryFact{}, false
	}

	for _, fact := range r.includeRegistries(cfpath.FromURI(string(pr.URI))) {
		if fact.listing == name {
			return fact, true
		}
	}

	return registryFact{}, false
}

// findRegistries reads a template's registry recipes out of its text: each
// listing whose loop constructs components with the loop's item and stores
// them in a struct, with the names the template's guards keep out of it.
// The loop is the first one naming the listing's struct after the listing
// itself — a template that fills two registries reuses one struct's name,
// and each listing's own loop is the one its recipe fills.
func (r *Resolver) findRegistries(content, file string) []registryFact {
	var facts []registryFact

	for _, listing := range pkgListRe.FindAllStringSubmatchIndex(content, -1) {
		pkg := content[listing[2]:listing[3]]
		target := strings.TrimPrefix(content[listing[4]:listing[5]], "variables.")

		span, ok := listingLoopSpan(content, target, listing[1])
		if !ok {
			continue
		}

		spanText := content[span.start:span.end]
		container, ok := registryContainer(spanText, span.item)

		if !ok {
			container, ok = factoryContainer(content, span.end, span.item)

			if !ok {
				continue
			}
		}

		name := strings.ToLower(strings.TrimPrefix(container, "variables."))

		if slices.ContainsFunc(facts, func(x registryFact) bool { return x.container == name }) {
			continue
		}

		skips := registrySkips(spanText, span.index)
		members := r.registryMembers(file, pkg, skips)

		if len(members) == 0 {
			continue
		}

		facts = append(facts, registryFact{pkg: pkg, container: name, listing: strings.ToLower(target), members: members})
	}

	return facts
}

// registrySpan is a listing loop's span, with the element and key variables
// its header binds.
type registrySpan struct {
	start, end int
	item       string
	index      string
}

// listingLoopSpan answers the span of the first loop after from whose
// collection or struct names listingVar, and the item and index variables
// its header binds.
func listingLoopSpan(content, listingVar string, from int) (registrySpan, bool) {
	for _, m := range loopTagPartsRe.FindAllStringSubmatchIndex(content, -1) {
		if m[0] < from {
			continue
		}

		if content[m[2]:m[3]] != "" { // a close tag
			continue
		}

		attrs := loopAttrs(content[m[4]:m[5]])

		if !strings.EqualFold(loopIterated(attrs), listingVar) {
			continue
		}

		end, ok := tagLoopEnd(content, m[1])
		if !ok {
			continue
		}

		return registrySpan{start: m[1], end: end, item: attrs["item"], index: attrs["index"]}, true
	}

	for _, m := range scriptLoopPartsRe.FindAllStringSubmatchIndex(content, -1) {
		if m[0] < from {
			continue
		}

		attrs := loopAttrs(content[m[2]:m[3]])

		if !strings.EqualFold(loopIterated(attrs), listingVar) {
			continue
		}

		end, ok := braceEnd(content, m[1]-1)
		if !ok {
			continue
		}

		return registrySpan{start: m[1], end: end, item: attrs["item"], index: attrs["index"]}, true
	}

	return registrySpan{}, false
}

// loopAttrs is a loop header's attribute name→value pairs, quotes and #s off.
func loopAttrs(header string) map[string]string {
	attrs := map[string]string{}

	for _, a := range loopAttrPairsRe.FindAllStringSubmatch(header, -1) {
		attrs[strings.ToLower(a[1])] = loopAttrValue(a)
	}

	return attrs
}

// loopIterated is the collection the loop header names: the tag's collection
// attribute, or the script loop's collection or struct.
func loopIterated(attrs map[string]string) string {
	name := attrs["collection"]
	if name == "" {
		name = attrs["struct"]
	}

	return strings.Trim(name, "#")
}

// loopAttrValue is the attribute match's value, whichever quote it was in.
func loopAttrValue(a []string) string {
	for _, v := range a[2:] {
		if v != "" {
			return v
		}
	}

	return ""
}

// tagLoopEnd walks cfloop tags from the open ending at after, answering the
// offset of its matching close tag's start.
func tagLoopEnd(content string, after int) (int, bool) {
	depth := 1

	for _, m := range loopTagPartsRe.FindAllStringSubmatchIndex(content, -1) {
		if m[0] <= after {
			continue
		}

		if content[m[2]:m[3]] == "" { // another open tag
			depth++

			continue
		}

		depth--

		if depth == 0 {
			return m[0], true
		}
	}

	return 0, false
}

// braceEnd walks braces from the open at open, answering the offset of its
// matching close brace. Braces inside quoted strings do not count.
func braceEnd(content string, open int) (int, bool) {
	depth := 0

	for i := open; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, true
			}
		case '"', '\'':
			i = skipQuoted(content, i)
		}
	}

	return 0, false
}

// skipQuoted walks the quote at i forward to its close, answering that
// closer's offset. CFML escapes a quote by doubling it; a backslash is an
// ordinary character.
func skipQuoted(content string, i int) int {
	quote := content[i]

	for i++; i < len(content); i++ {
		if content[i] != quote {
			continue
		}

		if i+1 < len(content) && content[i+1] == quote {
			i++

			continue
		}

		return i
	}

	return len(content) - 1
}

// registryContainer is the struct the loop's writes fill, when one of them
// stores what the listing's item constructed: a createObject written right
// on the store, or the variable an earlier createObject was assigned to.
func registryContainer(spanText, item string) (string, bool) {
	for _, m := range directStoreRe.FindAllStringSubmatch(spanText, -1) {
		if pathsToItem(spanText, m[2], item) {
			return m[1], true
		}
	}

	for _, m := range registryStoreRe.FindAllStringSubmatch(spanText, -1) {
		if assignedFromCreate(spanText, m[2], item) {
			return m[1], true
		}
	}

	return "", false
}

// factoryContainer is the struct a factory call fills, when the listing's
// loop ships its constructions into an array with arrayAppend and hands the
// array to a function:
//
//	<cfset arrayAppend(arr,createObject("component",fn))>
//	<cfset appenders=doSortedStruct(arr)>
//
// doSortedStruct is the same template's own function, whose body walks the
// array its argument received and stores each walked element in the struct
// it returns — appenders is that struct. The call is the first after the
// loop's span: a template that fills two registries hands both to the same
// factory, and each loop's own call is the one naming its container. The
// function must prove the shape: its first parameter the call's argument, a
// walk of `arguments.` it on the item a store and a return agree on.
func factoryContainer(content string, from int, item string) (string, bool) {
	spanText := content[:from]

	var arrs []string

	for _, m := range arrayAppendCreateRe.FindAllStringSubmatch(spanText, -1) {
		if pathsToItem(spanText, m[2], item) && !slices.ContainsFunc(arrs, func(a string) bool { return strings.EqualFold(a, m[1]) }) {
			arrs = append(arrs, m[1])
		}
	}

	for _, m := range arrayAppendVarRe.FindAllStringSubmatch(spanText, -1) {
		if assignedFromCreate(spanText, m[2], item) && !slices.ContainsFunc(arrs, func(a string) bool { return strings.EqualFold(a, m[1]) }) {
			arrs = append(arrs, m[1])
		}
	}

	if len(arrs) == 0 {
		return "", false
	}

	for _, m := range factoryCallRe.FindAllStringSubmatch(content[from:], -1) {
		if !slices.ContainsFunc(arrs, func(a string) bool { return strings.EqualFold(a, m[3]) }) {
			continue
		}

		if factoryFillsRegistry(content, m[2]) {
			return m[1], true
		}
	}

	return "", false
}

// factoryFillsRegistry is whether content's fn walks the array the call's
// argument handed it and stores each walked element in the struct it
// returns:
//
//	function doSortedStruct(arr) {
//		var sct=structNew('ordered');
//		loop array=arguments.arr item="local.el" {
//			sct[el.getClass()]=el;
//		}
//		return sct;
//	}
//
// The call's argument binding the factory proved itself in factoryContainer:
// its first argument is one of the arrays the appends shipped into.
func factoryFillsRegistry(content, fn string) bool {
	var params []string

	for _, m := range factoryDefRe.FindAllStringSubmatch(content, -1) {
		if strings.EqualFold(m[1], fn) {
			params = strings.Split(m[2], ",")

			break
		}
	}

	if len(params) == 0 {
		return false
	}

	words := strings.Fields(params[0])
	if len(words) == 0 {
		return false
	}

	param := words[len(words)-1]

	// The body begins at the first brace after the signature.
	for _, m := range factoryDefRe.FindAllStringSubmatchIndex(content, -1) {
		if !strings.EqualFold(content[m[2]:m[3]], fn) {
			continue
		}

		open := strings.Index(content[m[1]:], "{")
		if open < 0 {
			return false
		}

		end, ok := braceEnd(content, m[1]+open)
		if !ok {
			return false
		}

		return factoryBodyFills(content[m[1]+open:end+1], param)
	}

	return false
}

// factoryBodyFills is whether the function body walks the array its param
// received, binding an item, and stores each walked element in the struct it
// returns: one store `sct[...]=item`, spelled under the same name the body's
// return names. A walk the store's key or value does not keep to leaves the
// shape unproven.
func factoryBodyFills(body, param string) bool {
	var item string

	if m := factoryLoopRe.FindStringSubmatch(body); m != nil && strings.EqualFold(m[1], param) {
		item = strings.TrimPrefix(strings.TrimPrefix(m[2], "local."), "var ")
	} else if m := factoryForRe.FindStringSubmatch(body); m != nil && strings.EqualFold(m[2], param) {
		item = m[1]
	} else {
		return false
	}

	if item == "" {
		return false
	}

	store := regexp.MustCompile(`(?i)\b([A-Za-z_]\w*)\s*\[[^\]]*\]\s*=\s*(?:local\.)?` + regexp.QuoteMeta(item) + `\b`)

	m := store.FindStringSubmatch(body)
	if m == nil {
		return false
	}

	ret := regexp.MustCompile(`(?i)\breturn\s+(?:variables\.)?` + regexp.QuoteMeta(m[1]) + `\b`)

	return ret.MatchString(body)
}

// pathsToItem is whether the construction's path names the listing's item,
// or a variable the item was assigned to in the span.
func pathsToItem(spanText, pathVar, item string) bool {
	if strings.EqualFold(strings.TrimPrefix(pathVar, "variables."), item) {
		return true
	}

	re := regexp.MustCompile(`(?i)\b(?:var\s+)?([A-Za-z_]\w*)\s*=\s*` + regexp.QuoteMeta(item) + `\b`)

	for _, m := range re.FindAllStringSubmatch(spanText, -1) {
		if strings.EqualFold(m[1], strings.TrimPrefix(pathVar, "variables.")) {
			return true
		}
	}

	return false
}

// assignedFromCreate is whether name is the variable an in-span
// createObject(...) of the listing's item was assigned to.
func assignedFromCreate(spanText, name, item string) bool {
	if strings.ContainsAny(name, "([(\"") {
		return false
	}

	for _, m := range createdAssignRe.FindAllStringSubmatch(spanText, -1) {
		if !strings.EqualFold(m[1], name) {
			continue
		}

		if pathsToItem(spanText, m[2], item) {
			return true
		}
	}

	return false
}

// registrySkips is the names the key variable is guarded against in the
// loop's span: equal to a name where the guard skips the turn, or unequal to
// one where it constructs. Only a compare with the key counts; the guards
// name the base class and the field/group helpers. A construction made only
// when the key equals a name builds that member, which stays in.
func registrySkips(spanText, index string) []string {
	if index == "" {
		return nil
	}

	var skips []string

	for _, m := range skipCompareRe.FindAllStringSubmatchIndex(spanText, -1) {
		if !strings.EqualFold(spanText[m[2]:m[3]], index) {
			continue
		}

		op := strings.ToLower(spanText[m[4]:m[5]])
		equal := op == "==" || op == "===" || op == "eq"

		if equal != guardSkips(spanText[m[1]:]) {
			continue
		}

		if name := spanText[m[6]:m[7]]; !slices.Contains(skips, name) {
			skips = append(skips, name)
		}
	}

	return skips
}

// guardSkips is whether the condition the rest of a compare finishes is
// followed by a continue: the script if's closing parenthesis, or the
// cfif's closing angle bracket, then `continue` or `<cfcontinue>`.
func guardSkips(rest string) bool {
	depth := 0

	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '(':
			depth++
		case ')':
			depth--
		case '>':
			if depth > 0 {
				continue
			}
		case '"', '\'':
			i = skipQuoted(rest, i)

			continue
		default:
			continue
		}

		if depth >= 0 && rest[i] != '>' {
			continue
		}

		after := strings.TrimLeft(rest[i+1:], " \t\r\n{")

		return hasPrefixFold(after, "continue") || hasPrefixFold(after, "<cfcontinue")
	}

	return false
}

// hasPrefixFold is strings.HasPrefix, ignoring case.
func hasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix)
}

// registryMembers is the files of the package's directory, less the names
// the template guards out of the registry.
func (r *Resolver) registryMembers(file, pkg string, skips []string) []string {
	cfmlAdmin, javaAdmin, ok := adminHalves(r.fs(), file)
	if !ok {
		return nil
	}

	var members []string

	for _, dir := range registryPackageDirs(r.fs(), cfmlAdmin, javaAdmin, pkg) {
		entries, err := r.fs().ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			name := entry.Name()

			if entry.IsDir() || !strings.EqualFold(filepath.Ext(name), ".cfc") {
				continue
			}

			base := strings.TrimSuffix(name, filepath.Ext(name))

			if slices.ContainsFunc(skips, func(s string) bool { return strings.EqualFold(s, base) }) {
				continue
			}

			members = append(members, filepath.Join(dir, name))
		}
	}

	slices.Sort(members)
	members = slices.Compact(members)

	return members
}

// adminHalves are the two admin directories one checkout deploys together,
// the templates' and the components', read from the recipe file's own path.
// Both are stat'ed, so a layout naming only one half does not claim one.
func adminHalves(fsys vfs.FS, file string) (cfmlAdmin, javaAdmin string, ok bool) {
	if file == "" {
		return "", "", false
	}

	// The file lives in one half; the mark spells the half's admin root.
	cfmlMark := "core/src/main/cfml/context/admin"
	javaMark := "core/src/main/java/resource/context/admin"

	lower := strings.ToLower(file)

	var root string

	switch {
	case strings.Contains(lower, cfmlMark):
		root = filepath.Dir(file[:strings.LastIndex(lower, cfmlMark)])
	case strings.Contains(lower, javaMark):
		root = filepath.Dir(file[:strings.LastIndex(lower, javaMark)])
	default:
		return "", "", false
	}

	cfmlAdmin = filepath.Join(root, "core", "src", "main", "cfml", "context", "admin")
	javaAdmin = filepath.Join(root, "core", "src", "main", "java", "resource", "context", "admin")

	for _, dir := range []string{cfmlAdmin, javaAdmin} {
		info, err := fsys.Stat(dir)
		if err != nil || !info.IsDir() {
			return "", "", false
		}
	}

	return cfmlAdmin, javaAdmin, true
}

// registryPackageDirs is the directories the package's name spells, in the
// deployed tree it is installed as: lucee.admin.rest, lucee-server.admin.rest
// and lucee-web.admin.rest install into the admin's own root, and a bare
// name is that root's folder.
func registryPackageDirs(fsys vfs.FS, cfmlAdmin, javaAdmin, pkg string) []string {
	rest := pkg

	lower := strings.ToLower(pkg)

	for _, prefix := range []string{"lucee-server.admin.", "lucee-web.admin.", "lucee.admin."} {
		if strings.HasPrefix(lower, prefix) {
			rest = pkg[len(prefix):]

			break
		}
	}

	var dirs []string

	seen := map[string]bool{}

	for _, admin := range []string{javaAdmin, cfmlAdmin} {
		dir := filepath.Join(admin, filepath.FromSlash(strings.ReplaceAll(rest, ".", "/")))
		if seen[dir] {
			continue
		}

		seen[dir] = true

		if info, err := fsys.Stat(dir); err == nil && info.IsDir() {
			dirs = append(dirs, dir)
		}
	}

	return dirs
}

// familyComp is the receiver's value when it is an element of a registry:
// the package's display name and the fact's members, kept as absolute paths
// for the check they take — "$family:<pkg>#<member>|<member>".
func familyComp(fact registryFact) string {
	return familyPrefix + fact.pkg + "#" + strings.Join(fact.members, "|")
}

// familyRef is one family marker's package and member file paths.
type familyRef struct {
	pkg     string
	members []string
}

// familyRefs is the families a receiver's marker names: the single family
// `$family:pkg#m1|m2`, or each comma-joined marker of a multi-family
// receiver. A comp that names no family answers none.
func familyRefs(comp string) []familyRef {
	var refs []familyRef

	for piece := range strings.SplitSeq(comp, ",") {
		pkg, members := familyParts(piece)
		if len(members) > 0 {
			refs = append(refs, familyRef{pkg: pkg, members: members})
		}
	}

	return refs
}

// familyParts is the package a family receiver names and its member file
// paths.
func familyParts(comp string) (pkg string, members []string) {
	rest, ok := strings.CutPrefix(comp, familyPrefix)
	if !ok {
		return "", nil
	}

	pkg, rest, ok = strings.Cut(rest, "#")
	if !ok {
		return pkg, nil
	}

	return pkg, strings.Split(rest, "|")
}

// familyPaths is the member file paths of a family receiver — of every
// family a multi-family receiver names, each file once.
func familyPaths(comp string) []string {
	var paths []string

	for _, ref := range familyRefs(comp) {
		for _, member := range ref.members {
			if !slices.Contains(paths, member) {
				paths = append(paths, member)
			}
		}
	}

	slices.Sort(paths)

	return paths
}

// checkFamiliesMethod is checkFamilyMethod for a receiver each of refs'
// families holds on some turn: the call must be every family's, and the
// first family lacking it names the finding.
func (r *Resolver) checkFamiliesMethod(refs []familyRef, call *parser.CallSite, pr *parser.ParseResult, tr *callTrace) string {
	for _, ref := range refs {
		if why := r.checkFamilyMethod(ref.pkg, ref.members, call, pr, tr); why != "" {
			return why
		}
	}

	return ""
}

func (r *Resolver) checkFamilyMethod(pkg string, members []string, call *parser.CallSite, pr *parser.ParseResult, tr *callTrace) string {
	funcName := call.FuncName

	var (
		found *parser.FunctionDef
		lack  []string
	)

	for _, member := range members {
		def := r.LookupFuncWithExtends(member, funcName)
		if def == nil {
			lack = append(lack, displayMember(member))

			continue
		}

		if found == nil {
			found = def
		}
	}

	if len(lack) > 0 && r.methodExistenceGuarded(pr, call, funcName) {
		// The template itself checks whether the method exists before it
		// calls it — structKeyExists(driver,'getBundleName'),
		// !isNull(driver.getBundleName) — so a driver without it answers
		// the guard, not the call. Accepted as dynamic.
		tr.addf("%q is guarded by a method-existence check; a family member without it takes the guard's branch", funcName)
		tr.hit(TargetDynamic, "", nil)

		return ""
	}

	if len(lack) > 0 {
		return "method '" + funcName + "' not found in the " + pkg + " family — missing in " + strings.Join(lack, ", ")
	}

	tr.hit(TargetComponent, displayFamilyMembers(members), found)

	return ""
}

// methodExistenceGuarded is whether the call's neighbourhood tests the
// method's existence on the receiver first: an isNull or structKeyExists
// guard within three lines either way of the call. The templates guard
// optional driver contract methods this way, and a member without the
// method answers the guard, so the call is not a finding. A negated
// structKeyExists reads the other way and does not guard.
func (r *Resolver) methodExistenceGuarded(pr *parser.ParseResult, call *parser.CallSite, funcName string) bool {
	if pr == nil {
		return false
	}

	text := methodGuardWindow(pr.Content, call.Line, 3, 3)

	name := regexp.QuoteMeta(funcName)

	// A reference of the method itself questioned, the way
	// !isNull(driver.getBundleName) spells it.
	refCheck := regexp.MustCompile(`(?i)(?:^|[^\w])isNull\s*\(\s*[^\n()]*\.?[^\n()]*` + name + `\s*\)`)
	if refCheck.MatchString(text) {
		return true
	}

	// structKeyExists(recv, 'm') — a ternary, or an if whose branch calls
	// it. A negated one reads the other way and does not guard.
	keyCheck := regexp.MustCompile(`(?i)\bstruct[Kk]ey[Ee]xists\s*\(\s*[^(),\n]*\s*,\s*["']\s*` + name + `\s*["']`)
	for _, m := range keyCheck.FindAllStringIndex(text, -1) {
		if precededBy(text, m[0], "!", "not ") {
			continue
		}

		return true
	}

	return false
}

// methodGuardWindow is the lines around the call's own, from `before` above
// to `after` below it. Call lines are 0-based.
func methodGuardWindow(content string, line uint32, before, after int) string {
	var lines []string

	for at := int(line) - before; at <= int(line)+after; at++ {
		if at < 0 {
			continue
		}

		if l := lineOfContent(content, at); l != "" {
			lines = append(lines, l)
		}
	}

	return strings.Join(lines, "\n")
}

// precededBy is whether at is preceded by one of the words, which turns a
// structKeyExists guard the other way.
func precededBy(text string, at int, negs ...string) bool {
	for _, neg := range negs {
		if at >= len(neg) && strings.EqualFold(text[at-len(neg):at], neg) {
			return true
		}
	}

	return false
}

// displayFamilyMembers is how a trace names the family: its members' names.
func displayFamilyMembers(members []string) string {
	names := make([]string, 0, len(members))

	for _, member := range members {
		names = append(names, displayMember(member))
	}

	return strings.Join(names, "|")
}

// displayMember is a member file's own name.
func displayMember(member string) string {
	return strings.TrimSuffix(filepath.Base(member), filepath.Ext(member))
}

// registryElementRes is the component an element read of a registry answers:
// `driver = drivers[trim(form.type)]` reads the struct the listing loop
// filled, and each element of it is one of the family it stores. Only a
// right-hand side that is the element read and nothing else: a call on the
// element, an argument, a nested read or a builtin over it holds something
// else. A container may be written as variables.drivers.
func (r *Resolver) registryElementRes(rhs string, pr *parser.ParseResult, tr *callTrace) string {
	rhs = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rhs), ";"))

	// `drivers[k] ?: nullValue()` holds the element or nothing: the fallback
	// adds no value of its own, so it is the read alone.
	if m := nullFallbackRe.FindStringSubmatch(rhs); m != nil {
		rhs = strings.TrimSpace(m[1])
	}

	if !exactKeyedReadRe.MatchString(rhs) {
		return ""
	}

	m := indexedReadRe.FindStringSubmatch(rhs)
	if m == nil {
		return ""
	}

	container := strings.TrimPrefix(strings.ToLower(m[1]), "variables.")
	if strings.Contains(container, ".") || scopeWordOf[container] {
		return ""
	}

	fact, ok := r.registryFamily(container, pr)
	if !ok {
		return ""
	}

	tr.addf("%q is an element of %q, the registry the %q package's components were stored in — the family has %d components", m[1], fact.container, fact.pkg, len(fact.members))

	return familyComp(fact)
}

// nullFallbackRe is an expression with an Elvis fallback to null.
var nullFallbackRe = regexp.MustCompile(`(?is)^(.+?)\s*\?:\s*(?:nullValue\s*\(\s*\)|javaCast\s*\(\s*["']null["']\s*,\s*["']{2}\s*\))$`)

// scopeWordOf is the scope names, which name no registry.
var scopeWordOf = map[string]bool{"variables": true, "local": true, "arguments": true, "url": true, "form": true, "cgi": true, "cookie": true, "client": true, "application": true, "request": true, "session": true, "server": true, "this": true, "super": true}

// registryExprName is the plain variable name a collection expression is,
// when it reads a registry: `drivers` or `variables.drivers`, no scope
// word's name, no member.
func registryExprName(expression string) (string, bool) {
	name := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(expression)), "variables.")
	if name == "" || strings.ContainsAny(name, ".[(") || scopeWordOf[name] {
		return "", false
	}

	return name, true
}
