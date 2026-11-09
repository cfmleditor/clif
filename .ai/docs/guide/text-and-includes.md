# `#...#` in text and cfinclude scope

_Where `#...#` in text is CFML, and how a bare call is looked up through `cfinclude`. Moved out of CLAUDE.md unchanged._

## Where `#...#` in text is CFML

A tag file's text between tags is scanned for `#...#` only inside an output
context (`internal/parser/output_context.go`), and a `.cfm` with no CF tags that
holds HTML is a tag region, not CFScript. Outside an output context a hash is a
literal character, and pairing stray ones (`href="#"`, CSS colours, jQuery
`$('#id')`, Handlebars `{{#each}}`) swallowed the markup between them and turned
every `name(` in it into a call. `features.outputContextInterpolation` (default
on) switches it off; it reaches the parser as `ParseOptions.InterpolateAllText`,
which every call-extracting `ParseOptions` site must pass.

- **The rules were measured against Adobe ColdFusion, not remembered.** One
  probe template per context in the `tass_coldfusion` container. Evaluated:
  `<cfoutput>` (HTML attributes and `<script>` in it included), `<cfquery>`,
  `<cfmail>`, `<cffunction output="true|yes">`, and the attributes of every
  `<cf...>`, `<cf_...>`, `<cfmodule>` and cfimport-prefixed tag anywhere. Not
  evaluated: plain text, HTML attributes and `<script>` outside those,
  `<cfsavecontent>`, `<cfxml>` and custom-tag bodies, functions with `output`
  unset or false, and a template `<cfinclude>`d from inside `<cfoutput>`.
  `<cfdocument*>` and `<cfcomponent output="true">` could not be exercised and are
  taken as evaluated, because a wrong guess that way costs a false positive and
  the other way a call.
- **Attributes are always scanned.** A handled tag scans its own; a declined
  `<cf...>` tag and a `<prefix:tag>` whose prefix the file `<cfimport>`s scan
  theirs in `stepOverEvaluatedTag` and are stepped over whole, so the gate never
  sees them. Leaving them in the next text gap is what the ungated walk does, and
  gating that gap lost them: tassweb has 3,296 `#fn()#` calls in custom-tag
  attributes.
- **Output ranges are file offsets.** A tag region is cut around `<cfscript>`
  blocks and a `<cfoutput>` can open before one and close after it, so
  `outputContext` runs on the whole file and each region carries `Offset`.
- **The tag walk's span stepping is gated too.** `nextTagStart` steps over a
  `#...#` span because markup inside one is an argument. Outside an output
  context that is wrong: a Handlebars `{{#` paired with a hash far below and the
  walk stepped over real `<cfif>` tags, whose calls the old reading recovered
  only by scanning the whole run as one expression. `nextTag` uses a plain `<`
  search there. Finding it took a corpus diff of every extracted call, not of
  unresolved ones: the loss showed as a resolved builtin going missing.
- **It fails open.** An unclosed output tag runs to the end of the file, and a
  close with nothing open is ignored.
- **A `<script>` block with no CF tag is a `RegionSkip`, and still read.** It is
  kept from the CFScript scanner but handed to the tag parser, which finds
  nothing in it but `#...#` spans, and those only in an output context.
- **`Caller` is filled from the line.** A sub-parser for a tag expression or a
  `#...#` span, and a region cut from a function body, name no caller of their
  own; `fillCallers` gives every call without one the function whose scope holds
  its line. Before it, 514,852 of 895,451 calls in the TASS workspace had none.

## cfinclude scope

A bare or `this.` call that neither the file nor its extends chain answers is looked up through
cfinclude (`internal/resolve/includes.go`), and lands as `TargetInclude`. An included template
runs in its includer's variables scope, so the scope is every file that includes the calling
file, transitively, plus everything each of those — and the file itself — includes, plus the
extends chain of any component among them. A template included into `api.cfc` beside forty
others reaches `api.cfc`, its base, and every sibling. A file with several includers resolves
against all of them; which one ran is not in the source.

- `parser.ExtractIncludes` reads only `<cfinclude template>`, `include "…"`, `include
  template=` and `cfinclude(template=)` naming a `.cfm`/`.cfml`, skipping `#…#` paths and
  `<!--- --->` comments. Not `ExtractLinks`: `<cfmodule template>` runs in a scope of its own.
  It runs on every indexed file, so it finds the keyword and matches only there — two
  whole-file case-insensitive regexes took tassweb's index pass from 1.5s to 5.5s.
- **Two computed paths are read as well.** `specs/#file#`, a whole name computed in a written
  directory, is that directory's `*.cfm` (`listedNameGlob`), but only beside a non-recursive
  `directoryList` of the same directory filtered to `*.cfm` (`listsTemplates`): spreadsheet-cfml's
  `test/suite.cfc` includes each spec it lists that way. `#attributes.file#` in a custom tag is
  reported in that one spelling (`parser.AttributeInclude`), and the graph reads its targets from
  the tag's callers: each literal value a `<cf_name>` or `<cfmodule>` passes, resolved as an include
  in the tag (`attributeIncludeTargets`, over `customTagSites`, so batch only). The graph records
  whether it read the callers (`includeGraph.tagCalls`) and is rebuilt when `InferArgsFiles` is
  set after it was built. RustCFML's runner calls `<cf_runtest file="…">` 600 times.
  `TestASuiteIncludingEveryListedSpecSeesTheSuite`, `TestATemplateACustomTagIncludesByAttributeSeesTheTag`.
- The index holds each file's raw include paths and an `includeGen` counter; the resolver
  resolves them (file dir, `Application.cfc` root, mappings by first segment, workspace
  folders) into a forward and reverse graph and rebuilds it only when the counter moves.
- **`removeFileEntries` must not touch includes.** Every re-index runs it first, and clearing
  them there moved the counter on every re-index of an unchanged file, rebuilding the graph —
  thousands of stats — on each lazy index during a scan and each edit in the editor. Removal
  goes through `setIncludesLocked(uri, nil)` in `RemoveFile`, which compares before it bumps.
- **A template reads what its includer holds at the include** (`includerHeld`,
  `internal/resolve/included_locals.go`): an unscoped name the template never sets is, at
  each `<cfinclude>` that reaches it (`parser.IncludeSites`, which keeps each statement's
  offset), whatever the includer's receiver lookup gives at that line — a local or argument
  when the include is inside a function. Every site must type it; their union is the answer.
  It is the last step of `inferredReceiver`, cached per template, name and include generation.
  `TestATemplateReadsWhatItsIncluderHoldsAtTheInclude`.
- **A Mura display object reads what `dspObject_include` passes it** (`muraObjectArg`,
  `internal/resolve/mura_object_args.go`, before `includerHeld`): the renderer includes the
  template inside the function, so `arguments.X`, and a bare X the template never sets, are
  the call's named argument, typed where the call is made (`renderedValueD`, the `viewArgs`
  machinery). A computed `theFile` names every template its literal parts allow. Every call
  passing X must type it; one omitting it says nothing about `arguments.X` but declines a bare
  X; an `argumentCollection` declines; a call naming its file only by position is passed over
  unless it passes X. Batch only (the caller index finds the calls). The value's lookup counts
  the depth, which bounds templates handing a name on; the bound is a level past an assignment's
  (`maxHandOffDepth`), since typing `arguments.item = arguments.iterator.next()` spends one before the
  hand-offs start. `TestAMuraDisplayObjectReadsWhatItsIncludeIsPassed`,
  `TestADisplayObjectHandedOnTwiceTypesAnAssignmentFromIt`.
- **A ColdBox interceptor reads what the announce passes it** (`interceptData`,
  `internal/resolve/coldbox_intercept_data.go`, in `memberReceiver` after `viewArgs`): in a
  method `point( event, data )` (or `interceptData`) that does not assign its data itself,
  `arguments.data.KEY` and a bare `data.KEY` are the KEY of the struct literal each
  `announce`, `announceInterception` or `processState` call naming that point passes, typed
  where the call is made (`renderedValueD`); a local assigned from one follows through the
  alias step. A computed state (`"cbadmin_post#variables.entity#Save"`) names every point its
  literal parts allow. Every call passing KEY must type it; one passing no data or omitting KEY
  says nothing; data that is not a struct literal declines, and so does KEY passed under a state
  not spelled at all. Batch only. On ContentBox it typed 76 findings in the notification,
  cache-cleanup and renderer interceptors (1,670 → 1,598). `TestAnInterceptorReadsWhatItsAnnounceIsPassed`.
- **A custom tag reads what the tags calling it pass** (`customTagAttribute`,
  `internal/resolve/custom_tag_attributes.go`, in `memberReceiver` before `includerHeld`): a
  `.cfm`'s `attributes.X` is the X of every `<cf_name …>` and `<cfmodule template="…/name.cfm">`
  in the workspace, given as one `#expr#` and typed where the call is made (`renderedValueD`);
  a tag handing an attribute on to another tag passes what it holds. Which file a tag name runs
  is a runtime fact (customTagPaths), so a name two `.cfm`s answer to is not claimed. Text, an
  `attributeCollection`, or a tag setting the attribute itself leaves it untyped; an omitting
  call says nothing, and a `cfparam` default is not a set. Batch only. On MasaCMS it moves one
  finding: `fileselector`'s fourth caller passes `extendAttribute.renderAttribute()`'s own
  `arguments.bean`, whose callers are on an untyped loop element.
  `TestACustomTagReadsWhatItsCallsPass`.
- A writer that indexes from a parse result calls `Index.SetIncludes` beside `SetThisVars`;
  `IndexFile` sets them itself. The `unresolved` command records a template's includes without
  indexing its functions, since a page that includes a helper can call what the helper declares.
