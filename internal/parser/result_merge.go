package parser

import (
	"slices"
)

// extractSignatures does a shallow parse: function names/args, component refs, scopes.
// tagFunctionScopes are the scopes of the tag functions of the whole file, and
// whether it has a tag region at all.
//
// Pre-computed from the whole file (not per region) so a function whose body is
// interrupted by a nested <cfscript> block (which splits the file into separate
// Tag/Script/Tag... regions) still gets a correct end line and in-function
// tracking in every region it spans, not just the region containing its
// opening <cffunction> tag.
func (pr *ParseResult) tagFunctionScopes() (scopes []FuncScope, hasTagRegion bool) {
	if ctx := pr.bodies.context; ctx != nil {
		return ctx.walkTagScopes()
	}

	for _, r := range pr.Regions {
		if r.Kind == RegionTag {
			return findTagFuncScopesIdx(pr.Content, 0, pr.contentLineIdx, pr.Regions), true
		}
	}

	return nil, false
}

// mergeRegions parses each region of the file into the result, in file order,
// and returns the calls whose receivers the later passes still have to type.
func (pr *ParseResult) mergeRegions(tagScopes []FuncScope) []pendingCall {
	var allPendingCalls []pendingCall

	// The locals of a tag function a region split, carried to the regions
	// after the split.
	var open openLocals

	// The tag regions share one output state, in file order, from the top.
	pr.output = outputState{}
	pr.collectionTags, pr.collectionTagsOf = nil, pr.Content
	pr.bodies.vars, pr.bodies.outputAt = pr.bodies.vars[:0], nil

	regions := pr.Regions
	if body := pr.bodies.only; body != nil {
		regions = pr.bodyRegions(body)
		pr.output = pr.outputAtBody(regions)
	}

	for _, r := range regions {
		// A RegionSkip is a literal <script> block, left out of the script
		// regions so its JavaScript is never fed to the CFScript scanner. Its
		// `#...#` spans are CFML all the same — a <script> body inside
		// <cfoutput> is where a page writes `var id = "#o.getContentID()#";` —
		// and dropping the region dropped those with it.
		//
		// It goes to the tag parser below, which needs no mode of its own:
		// findScriptSkipSpans only makes a span of a block that holds no `<cf`
		// tag at all, so the walk can find nothing in one *but* its
		// interpolation. A flag restricting it to the spans measured
		// identically over the corpus and against tag-shaped text inside a JS
		// string, which is the case it was written for — and which is not a
		// skip region, because it contains `<cf`.
		if r.Kind == RegionSkip && !pr.walksCalls() {
			continue
		}

		// The tag functions' scopes and spans come from tagScopes, below.
		if pr.scopesOnly && r.Kind != RegionScript {
			continue
		}

		switch {
		case r.Kind == RegionScript:
			allPendingCalls = append(allPendingCalls, pr.mergeScriptRegion(&r, tagScopes, &open)...)
		case r.Island:
			// A tag island's code is the script function's around it, whose
			// scope the script region before it has just filed.
			allPendingCalls = append(allPendingCalls, pr.mergeTagRegion(&r, slices.Concat(tagScopes, pr.Scopes), &open)...)
		default:
			allPendingCalls = append(allPendingCalls, pr.mergeTagRegion(&r, tagScopes, &open)...)
		}
	}

	return allPendingCalls
}

// addTagScopes adds the tag functions' scopes the regions did not record. They
// are keyed by start line rather than found by a scan of pr.Scopes per
// candidate: both lists are one entry per function, so the pair was quadratic
// in a component's method count.
func (pr *ParseResult) addTagScopes(tagScopes []FuncScope) {
	seen := make(map[int]struct{}, len(pr.Scopes))
	for _, existing := range pr.Scopes {
		seen[existing.Start] = struct{}{}
	}

	for _, sc := range tagScopes {
		// A body's walk keeps the functions on its lines alone.
		if body := pr.bodies.only; body != nil && (sc.Start < body.Start || sc.End > body.End) {
			continue
		}

		// Every tag function's span is kept, whichever walk filed its scope.
		pr.keepSpan(&sc)

		if _, dup := seen[sc.Start]; dup {
			continue
		}

		seen[sc.Start] = struct{}{}

		pr.Scopes = append(pr.Scopes, sc)
	}
}
