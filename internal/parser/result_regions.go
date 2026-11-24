package parser

import (
	"slices"
)

// keepRegionFacts keeps what a tag region's walk found that later passes would
// otherwise walk the region again for: where it met the tags the collection
// passes read.
func (pr *ParseResult) keepRegionFacts(r *Region, tp *tagParser) {
	if pr.collectionTags == nil {
		pr.collectionTags = map[int][]int{}
	}

	pr.collectionTags[r.Offset] = tp.collectionTags
}

// keptCollectionTags is where the parse's walk met the tags collectWrites
// reads in the tag region at offset, when read from the current content.
func (pr *ParseResult) keptCollectionTags(offset int) ([]int, bool) {
	if pr.collectionTags == nil || !pr.regionFactsCurrent() {
		return nil, false
	}

	sites, ok := pr.collectionTags[offset]

	return sites, ok
}

// regionFactsCurrent reports whether what the region merge kept was read from
// the current content. When not, it is dropped, so the old document is not
// kept alive by it.
func (pr *ParseResult) regionFactsCurrent() bool {
	// Equal strings sharing their bytes compare without reading them.
	if pr.collectionTagsOf == pr.Content {
		return true
	}

	pr.collectionTags, pr.collectionTagsOf = nil, ""

	return false
}

// DocumentLinks returns the file path references in the document, from the
// record its walk made (file_refs.go), or, for a parse that did not record
// links (ExtractLinks) or whose content an edit has changed since, one read
// now and kept. Read over the whole text rather than per region, a link has
// its column on its own line wherever a region started.
func (pr *ParseResult) DocumentLinks() []DocumentLink {
	content := pr.Content
	rec := pr.refRecord(true)

	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.textLinks == nil || pr.textLinksOf != content {
		pr.textLinks = rec.links(content, pr.lineStartsTable())
		if pr.textLinks == nil {
			pr.textLinks = []DocumentLink{}
		}

		pr.textLinksOf = content
	}

	return pr.textLinks
}

// contentRegions is Content split as the parse splits it, which knows a
// markup template by its name: its own regions when they are of this
// content. pr.mu is held.
func (pr *ParseResult) contentRegions() []Region {
	if pr.regionsOf == pr.Content && pr.Regions != nil {
		return pr.Regions
	}

	regions, _ := pr.classifyRegions()

	return regions
}

// IncludeSites is parser.IncludeSites of Content, from the includes the
// parse's walk recorded.
func (pr *ParseResult) IncludeSites() []IncludeSite {
	content := pr.Content
	if IndexFold(content, "include") < 0 {
		return nil
	}

	rec := pr.refRecord(false)

	pr.mu.Lock()
	regions := pr.contentRegions()
	pr.mu.Unlock()

	return includeSitesOf(content, rec, regions, pr.includeWrappers)
}

// RegionKindFrom is the kind of the first of the parse's regions of Content
// that ends after offset: the region offset is in, or, when offset is in the
// blank text or the <cfscript> tag between two, the one after it. A line's
// start asks about the code on that line, so a `<cfscript>function f() {`
// line is script. RegionTag when no region ends after offset.
func (pr *ParseResult) RegionKindFrom(offset int) RegionKind {
	pr.mu.Lock()
	regions := pr.contentRegions()
	pr.mu.Unlock()

	i, _ := slices.BinarySearchFunc(regions, offset, func(r Region, offset int) int {
		if r.Offset+len(r.Text) <= offset {
			return -1
		}

		return 1
	})
	if i == len(regions) {
		return RegionTag
	}

	return regions[i].Kind
}

// Includes is parser.ExtractIncludes of Content, from IncludeSites.
func (pr *ParseResult) Includes() []string {
	return includePaths(pr.IncludeSites())
}
