# Which walker is right where they disagree

The parser has several walkers doing the same job, and they disagree on real
files (see PARSE-PASSES-PLAN.md, steps 5 to 7). This records which one is
right, case by case, judged by the tree-sitter grammar over the whole corpus
(`~/corpus`, 14,949 `.cfc` and `.cfm` files), measured 2026-11-10.

The harnesses (temporary tests, not committed):

1. A classifier in `internal/parser` lists every item one walker reports and
   the other does not, with its file, line and source text.
2. A judge in `internal/tsoracle` parses the same file with the CFML grammar,
   and the CFScript grammar for each injected script region, and decides
   each item from what the grammar sees on that line.

## 1. Variables declared outside functions

The two walkers:

- `globalScriptParser`: `VariablesVars`, `ThisVars`, and through them the
  index and completion.
- The main parse's script walk, `scriptParser.vars`. `ParseVars`, and so
  `AllVars`, uses the same walker.

The comparison covers variables- and this-scope declarations outside any
function, in script regions. 4,853 declarations differ.

The judge counts a declaration only for these:

- a bare, `variables.x` or `this.x` assignment that is not a named argument;
- a `var` declarator;
- a `for (x in …)` target.

It also requires the declaration to be outside every function and closure.

### Seen only by the global scan (3,245)

| Grammar's reading | Count | Verdict |
|---|---|---|
| A key of another struct: `url.x =`, `fckEditor.config.x =`, `variables.framework.base =` | 2,300 | Global scan wrong: it takes the last name of any member assignment |
| `local.x =` outside a function | 505 | Global scan wrong as written (it sets a key of `local`); see note |
| A comparison: `if (form.action == 'x')` | 220 | Global scan wrong: it reads `==` as an assignment |
| Inside a function or closure | 27 | Global scan wrong |
| Struct key or named argument | 9 | Global scan wrong |
| Not script to the grammar | 32 | Region split disagrees (see 4) |
| A real declaration | 120 | Global scan right. Most are a top-level `var x`, which the main parse records as local and `ParseVars` turns into variables, so the two agree after that step |
| Other | 32 | Multi-line literals whose line the walkers place differently |

### Seen only by the main parse (1,608)

| Grammar's reading | Count | Verdict |
|---|---|---|
| A struct literal key, or a named argument written one per line: `{reload = false}`, `foo(\n append = "",\n …)` | 769 | Main parse wrong |
| Inside a function or closure | 369 | Mixed, see below |
| A comparison | 7 | Main parse wrong |
| Not script to the grammar: a settings `.ini.cfm` read as script | 33 | Both are wrong to treat it as script (see 4) |
| A real declaration: `x = 1;`, `for (x in …)` | 391 | Main parse right; the global scan misses plain top-level assignments and for-in targets |
| Other | 39 | `catch (any e)` at the top level, multi-line literals |

**Inside a closure** needs a ruling rather than a count.
- `variables.x = …` inside a TestBox `describe(…, function(){ … })` writes
  the component's variables scope, so for completion it is a real
  variables name. The main parse is right to keep it, and the grammar's
  "outside functions" rule is too strict there.
- An unscoped `x = …` inside a closure is not a top-level declaration. Its
  scope depends on the engine's local mode.

**`local.x` outside a function** is also a ruling. The grammar reads it as a
key of a struct named `local`. Lucee and ACF give a component's
pseudo-constructor no local scope, so the global scan's reading
(variables.x) has no engine behind it.

### Verdict

Neither walker is right overall. Each has a defect class the other does
not:

| Walker | Wrong about |
|---|---|
| Global scan | Member assignments (`a.b =`), `==`, `local.x` |
| Main parse | Struct literal keys and named arguments spread over lines |
| Both | Settings files classified as script |

The global scan's errors are about 3,060 and the main parse's about 810
(plus closures, by ruling). The main parse's defect class is the smaller and
more local one, in how it reads `key = value` inside `{…}` and `(…)`, so it
is the one to keep:

1. **Fix the main parse's struct-key and named-argument reading.** A `name =`
   inside an open `{` or `(` of a call is not an assignment.
2. **Normalize a top-level `var` to variables,** as `ParseVars` already
   does.
3. **Rule on closures.** Keep `variables.x` and `this.x` inside a closure;
   drop unscoped names.
4. **Then retire `globalScriptParser`.** `computeScopedVars` would read the
   main parse's kept declarations, as the tag side already does
   (`keepTagScopeVars`).

After step 1, re-run the classifier: what remains should be only the global
scan's three defect classes, which is the evidence that replacing it is a
fix, not a change of opinion.

**Done.** Changes to the main parse:

- **Statement level.** Its two loops track the brackets nothing consumed
  (`trackOpen`), and a bare `name =` declares only at statement level: in a
  block or a `for` header, not in a call's arguments or a struct literal. A
  `{` after `=`, `(`, `[`, `,`, `:`, `?` or `return` is a literal.
- **Values ending in a literal.** A right-hand side that is a struct or array
  literal ends there. `x = {}` followed by `y = 2` on the next line with no
  semicolon used to take `y` for `x`'s type and never declare it; the global
  scan got that right, so this had to be fixed first.

Re-judged, the main parse's struct-key and named-argument group fell from
766 to 1. What it still gets wrong: 7 comparisons, 3 named arguments, and 33
lines of settings files read as script. Every other difference is the global
scan's error, or a closure, which is a ruling.

`computeScopedVars` then reads the parse's kept declarations for each script
region (`keepScriptScopeVars`), as it already did for tag regions, and walks
the region with the same `scriptParser` when an edit inside a function has
made them stale. `globalScriptParser` is deleted.

- `unresolved` on RustCFML loses 9 false findings, calls to a variable a bare
  top-level assignment holds (`add5 = makeAdder(5)`), and gains none. The
  other five corpora are byte-identical.
- Token passes fell further: ContentBox from about 13.4 to 11.8, Mura from
  about 15 to 14.1.
- `TestOneScanGivesEachScopeWhatItsOwnScanDid` now compares against the same
  walk per scope, and `TestScopedVarsAreWhatTheGrammarReads` pins the
  grammar's verdicts. That test fails on 3 counts with the global scan
  restored.
- `scriptParser` grows to 416 bytes, the end of its allocation class.

## 2. Function scopes

The two walkers:

- The main parse's `Scopes`.
- `findFuncScopesIn` (`findScriptFuncScopes` and the tag scope finder), used
  only by `ParseVars` and `GlobalVars`.

3,213 scopes differ across 354 files.

| Case | Count | Verdict |
|---|---|---|
| Same start, different end, the parse's end is the grammar's | 1,381 | Parse right. The finder ends a tag `<cffunction>` early (213–217 for a 213–227 function) and overruns some script ones |
| Same start, different end, the finder's end is the grammar's | 0 | |
| Only the parse has it, and the grammar has the function | 339 | Parse right; the finder misses them |
| Only the parse has it, grammar has no named function there | 103 | Closures assigned to a name (`f = () => true`): real bodies, so the parse is right |
| Only the finder has it | 3 | Finder right in 2 (hoisted `<cffunction>`, a `ModuleConfig.configure`) |
| Neither end matches | 3 | Both wrong |

**Verdict:** the parse is right in effect everywhere but 2 cases. Replace
`findFuncScopesIn` in `ParseVars` and `GlobalVars` with the parse's
`Scopes` (via `AllVars`, which has the parse). This also explains the 230
extra `AllVars` differences seen in plan step 6. Then look at the 2
finder-only functions, where the parse should learn from the finder.

## 3. A function's refs: full parse vs lazy body parse

`funcRefsUncached` parses a function body on its own when the full parse
filed no refs under that function. It finds refs in 1,309 of 48,268 such
functions, 2,369 refs in all.

| Where the full parse had the ref | Count |
|---|---|
| In the file's own refs (`ComponentRefs`) | 2,355 |
| Under another function's key (a nested function) | 13 |
| Nowhere: the "ref" is a line comment the body parse read as code | 1 |

**Correction (2026-11-10).** The measurement above ran with no resolvers
and counted typed refs only, so it saw only one filing difference. Pre-filling
empty entries, as plan step 7 proposed, failed six tests, which show the
lazy body parse is part of the design:

- **Script functions:** an unscoped assignment inside a function
  (`svc = _kernel();`) is filed by the full parse at file level, for the
  variables scope it writes. Only the body parse files the copy under the
  function, and `FuncComponentRefs` types the variable within the function
  from that copy.
- **Tag functions:** the full parse does not read a `<cffunction>` body's
  refs at all. The tag parse leaves them to `parseTagBody`.
- **Top-level closures:** these depend on it too
  (`TestTopLevelClosureConstructorOwnership`).

**Verdict:** the two are not one walker doing a job twice. The body parse
is the only one that files a function's own view of its unscoped writes,
and of a tag function's body. To retire it, the full parse would have to
file both copies of an unscoped write and parse tag function bodies itself,
which is a design change.

**Measured cost**, in an `unresolved` scan:

| Corpus | Body parses | Text read | Share |
|---|---|---|---|
| ContentBox | 695 | 417KB of its 3.07MB | about 0.14 of its 11.8 passes |
| Mura | 883 | 844KB of 9.26MB | about 0.09 of 14.1 |

About 1% of the work in each case, so the redesign is not worth its risk for
speed. Leave it unless a correctness case needs the two filings to agree.

## 4. Region split

Two cases: `.ini.cfm` settings files, and plain HTML templates with no CF
tag. `ClassifyRegions` reads them as one script region, so every walker that
uses it invents declarations; the grammar reads them as markup.

`ParseResult.classifyRegions` already handles the HTML case
(`isMarkupTemplate`) but not settings files. The free `ClassifyRegions`
handles neither. One split, used by every walker, would fix both.

**Done.** Two of the 11 files the grammar called "not script" were the
grammar's own gaps (real `<cfscript>` it did not inject) or RustCFML shebang
scripts. The rest were text files a `.cfm` extension lets the engine serve.

- **`ClassifyRegionsIdx`**, the split both splitters share, now reads a
  tagless file as text when its first line could not begin CFScript
  (`startsAsText`): a `#` (a Markdown heading, a properties comment) or an
  INI `[section]` header. A `#!` shebang stays script. Over the corpus this
  changes exactly 5 files, all text:
  - Mura's settings and cffp INI templates;
  - ContentBox's `runtime.properties.cfm`;
  - cbmarkdown's `markdown.md.cfm`;
  - a Wheels view whose body is `[200-OK]`.
- **`AllVars`** uses the parse's own split rather than the free one, so a
  markup template is no script region there either.

`unresolved` is unchanged on the six corpora.

## 5. Other function finders: one string rule

Three resolver-side walkers find a component's functions themselves, each
with a plain `parser.NewScanner`:

- `parser.applicationScan`, behind `ApplicationSections`, `ApplicationMethod`
  and `ApplicationMethodNames`;
- `newProducerPlans`, via `scriptProducerMethods`;
- `wheelsMethods`, which reads script only by design.

Compared with the parse's functions over 9,896 components:

| Finder | Misses a function the parse has: bodiless (by design) | Misses: other | Finds one the parse does not |
|---|---|---|---|
| `applicationSections` | 625 | 417 | 160 |
| producer plans | 749 | 459 | 160 |
| `wheelsMethods` | 752 | 1,208 (plus 5,849 tag functions, by design) | 2 |

The "other" misses share one cause. A plain scanner does not read a string
the way the parse does (`interpStrings`). In ColdBox's `Router.cfc`,
`"{#listLast( thisPattern, "{" )#)"` holds a quoted `"{"` inside its `#...#`.
A plain scanner ends the string at the inner quote and counts the `{` as a
brace, and every function after it, from `initRouteDefinition` through
`header`, is lost to all three finders. The parse reads them all.

**Verdict:** the parse is right. Give these finders the parse's string rule:
an exported scanner constructor for CFScript text with `interpStrings` on.
Better still, have them read the parse's function spans (`Scopes`, with
names). That is "functions only" as a mode of the one walker. The 160 "finds
one the parse does not" are mostly names in CR-only files (see 6) and are
to be re-measured after.

**Done** (`parser.NewScriptScanner`, used where the text is CFScript as a
whole, `parser.IsScriptSource`). Real misses fell from 417, 459 and 1,208 to
61, 60 and 136. What remains is mostly:

- closures assigned to a variable, which the parse counts as functions and
  these finders do not, by design;
- `PerformanceSuite.cfc`, where the grammar disagrees with the parse too.

`unresolved` is byte-identical on the six corpora.

## 6. Line counting

66 corpus files end lines with a bare CR (old Mac line endings). The parser,
its line index and tree-sitter's rows count `\n` alone, so in those files
every line is line 0. The LSP specification counts `\r`, `\n` and `\r\n`, so
every position the server reports in such a file is wrong in the editor.
One line counter, the LSP's, should serve the parse and the server.

**Done**, by making the text right where it is read rather than teaching
every counter a second rule. `lex.NormalizeLineBreaks` turns a bare `\r` into
`\n`, which keeps every byte offset. It is applied:

- in `path.AsText`/`AsTextString`, which the scan and workspace indexing use;
- in the resolver's `readText`;
- in the server, on `didOpen` and every change, through
  `lspdoc.ChangeRangeAndText`.

A document opened with CR line endings alone gets them back in formatting
edits (`withDocumentLineBreaks`). The CLI `format` reads raw bytes, so a
file's endings are kept.

On `ColdBox_coldbox-samples` (56 CR-only files) the scan's findings went
from 428 to 411. Every changed finding is in a file with bare carriage
returns: findings that sat on line 1 now carry their lines and functions.
The six-corpus check is byte-identical.

## 7. Tag attribute readers

Five readers take a whole tag and give its attributes. Over the corpus
(51,143 CF tags, 100,123 attributes):

- **`getAttr`** (with `attrSpan`): 33 call sites, the tag parser's.
- **`extractAllAttrs`**: 2 call sites.
- **`TagAttributes`**: added for custom tags in #355.
- **`listingAttrs`**: 2 call sites, in the include scan. It is a byte port of
  an old regex.
- **`resolve/orm.tagAttr`**: 3 call sites, still a regex.

They disagree on 4,374 attributes. The grammar's `cf_attribute` nodes judged
the 3,338 it could place on the same tag:

| Reader | Agrees with the grammar | Differs | Why it differs |
|---|---|---|---|
| `TagAttributes` | 3,199 | 139 | A doubled quote inside a value (`""`, CFML's escape) ends it early; the rest it shares with every reader (below) |
| `getAttr` | 2,822 | 516 | Finds a name inside another's value; shares the tag-extent cases |
| `ormTagAttr` | 2,814 | 524 | Names with digits (`show3d`) and unquoted values (`basictab=false`, `default=0`) are not read |
| `listingAttrs` | 1,636 | 1,702 | Either quote ends a value, so `detail="The 'CacheFactory' …"` reads as `The ` |
| `extractAllAttrs` | 1,523 | 1,815 | Space before `=` loses the attribute (`name   ="toXML"`); a name runs over line breaks (500 corpus names hold one) |

Two classes belong to no single reader:

- **Tag extent (shared).** A quoted value that holds a `<`, such as
  `hint="… <br />"` or `output="</cfif>"`, ends the tag early (`tagEnd`),
  so every reader misses the attributes after it.
- **No grammar tag at that offset (918).** Tags the grammar sees elsewhere,
  or not at all; not judged.

**Verdict:** `TagAttributes` is the reader to keep.

1. Teach it the doubled-quote escape.
2. Make `getAttr` and `attrSpan` its lookups. They are the hot path: 33
   call sites, read for every tag. A single-attribute scan with the same
   rules keeps that cheap; the work is matching the rules, not building a
   map.
3. Replace `extractAllAttrs` and `orm.tagAttr` with it.
4. Rule on `listingAttrs`. It reproduces the include scan's old regex on
   purpose, so changing it changes which includes are found. Judge that
   change with the include oracle and the six-corpus diff.
5. Separately, find where `tagEnd` stops at a `<` inside a quoted value.
   That one change is shared by every reader.

### Done: one scanner, two modes

`attrScanner` (`internal/parser/attr_scan.go`) reads a tag's attributes in
one of two modes, which is the one place the rules for a CF tag and a markup
tag differ. The grammar draws the same line.

| | `cfAttrs`: built-in CF tags, custom tags, `cfmodule`, `template=` | `htmlAttrs`: markup, `href=`, `action=` |
|---|---|---|
| `""` inside a value | An escaped quote | Ends the value |
| `#…#` inside a value | An expression whose strings do not end the value (`##` is a literal `#`; a stray `#fff` is text) | Text |

Both modes share the rest: names `[\w:.-]`, space around `=`, unquoted
values to a space, `>` or `/>`, and the first of a repeated name. The readers
now built on it:

- `getAttr` and `attrSpan`;
- `extractAllAttrs` and `TagAttributes`;
- `parser.TagAttribute`, which replaces the ORM reader's regex;
- `ExtractLinks`, whose `template=` values are read in CF mode and `href=` in HTML mode (see "Done: document links").

`listingAttrs` is gone too: see "Done: includes".

**`tagEnd`** now ignores a `<` inside the tag's quoted values
(`tagStartOutsideQuotes`). A `<` starting a tag outside them still ends it,
as `<input <cfif a>checked</cfif>>` needs, and so does any `<` when the
tag's quotes do not pair.

**Re-judged** over the corpus, the shared reader agrees with the grammar on
2,125 of the 2,127 attributes where any readers still differ. Those are the
deferred `listingAttrs` and the old ORM regex's shapes.

**`unresolved`**, old build against new:

| Corpus | Change |
|---|---|
| Lucee, Mura, FW/1, ContentBox, ColdBox, Slatwall | Byte-identical |
| RustCFML | 9 false findings removed |
| cfwheels | 7 false findings removed |
| Preside | 8 removed: `mockColdBox.$` now typed through `createMock` |
| ColdBox samples | Line-number corrections from section 6 |

All of these come from the earlier steps; the attribute and `tagEnd`
changes move no finding.

### Done: document links

`ExtractLinks` read every line with its own rules: a `template=`, `href=` or
`action=` anywhere on it, in a comment, in prose, in a `<cfset>` or as a CF
tag's `action="list"` keyword. It now reads the file as the parse splits it
(`internal/parser/links.go`), with the attribute scanner and the script
scanner:

- **Tag regions:** CF comments are skipped and each tag is read whole by
  `attrScanner`, in CF mode for a CF tag and HTML mode for markup. `template`
  links on a CF tag, `href` on markup, `action` only on `form`/`cfform`; a
  value must be quoted, and an expression tag (`cfset`, `cfif`, `cfelseif`,
  `cfreturn`) links nothing. A markup tag cut by a CF tag
  (`<a <cfif x>class="on"</cfif> href="p.cfm">`) keeps its later attributes.
- **Script regions:** the script scanner, skipping comments, reads
  `include "x"`, `include template="x"` and `cfinclude(template="x")`; `cfmodule(…)` and `module template=` come from the
  custom tag scanner; a CF tag written among the statements is read as a tag.
  A member call (`arr.include(…)`) and a struct key named `template` are not
  links.
- **Regions:** a tag-free file that begins with markup is markup, as in
  `StructureSpans`, and `DocumentLinks` reuses the parse's regions.

Over the corpus the old reader's extra links were junk (CF `action`
keywords, prose, `data-action`, `#buildURL(action=…)#`) or paths in comments.
The code map's include edges, which come from these links, differ on five of
47 projects, and every edge lost is one that should go: an include inside a
`<!--- --->` comment (cfwheels 1, cfdocs 4, Lucee 9, Mura 1), a struct key
named `template` (RustCFML 2), or a Lucee admin `<cfmodule>` after the
source's own unbalanced quote (`list=auto,…inherit"`), 1. No edge is added.

### Done: includes

`IncludeSites`, behind `ExtractIncludes`, the index's includes and every
include scope the resolver reads, was a byte scan for the word `include`
anywhere in the file that skipped `<!--- --->` comments alone. The code map
took its include edges from the links instead, so the two could disagree.
Includes are now the include kind of the links walk (`fileRefs` in
`links.go`): `refInclude` for a `<cfinclude>` template or an include
statement, `refTemplate`, `refHref`, `refAction`, and `refListing` for a
`<cfdirectory>`. Links keep the literal paths of every kind; includes keep
the include kind, computed paths read as before. An `includesOnly` mode
scans the attributes of `<cfinclude>` and `<cfdirectory>` alone.

- `listingAttrs` is replaced by the CF-mode attribute scanner, and the
  listing's own `<cfinclude>` is read from the walk's include path
  (`listingIncludePath`).
- The resolver's `constructorIncludes` read script includes with a plain
  scanner and a third rule; it reads `IncludeSites` now, and its scanner
  (`NewScriptScanner`) only says which include statements stand in the
  component body.
- The Application.cfc walk (`mapping_sources.go`) still read an include at
  a keyword its own walk found; see "Done: the Application.cfc walk's
  includes".

- A function that includes the template an argument names is an include
  wrapper (`wrapperIncludes`): Wheels' `Global.cfc` declares
  `$include(template)` as `include "#LCase(arguments.template)#"` and loads
  the routes with `$include(template = "/config/routes.cfm")`, which runs them
  in its own variables scope. An unqualified call of the wrapper in the same
  file, with a literal path by name or in its place, is an include there. The
  old scan read these by accident, the `$` letting `include(template=` match;
  without them cfwheels' `routes.cfm` lost `mapper()` and gained 4 findings.

Over the 47 corpus projects the links are identical, and the includes differ
on 20, all of them correct:

| Change | Count |
|---|---|
| Lost: in a `//` or `/* */` comment | 10 |
| Lost: in a string (`fileWrite(…, 'component { include "x.cfm"; }')`) | 3 |
| Lost: `application.wo.$include(…)`, which runs the template in another object's scope | 7 |
| Added: RustCFML's `renderWith(tpl = "x.cfm")`, a wrapper of `include "#arguments.tpl#"` | 1 |

`unresolved` is byte-identical on RustCFML, cfwheels, Mura, FW/1,
ContentBox, ColdBox, Slatwall and Lucee. The include scan costs more: 117ms
against 65ms over the corpus's 57MB (best of five), most of it the region
split and the script scanner. Callers that hold a parse (the server's
indexing, DI sources, included locals) reuse its regions through
`pr.Includes()` and `pr.IncludeSites()`.
### Done: one string rule for every walk

Section 5 gave three function finders the parse's string rule. 23 more
scanners in the resolver and `internal/path`, and 7 whole-text walks in the parser, made a plain
`parser.NewScanner` over a file, a function body or an expression, the
resolver's shared token memo (`allTokens`) among them, and read every
`"#f("x")#"` in CFScript as two strings with code between.

`parser.NewSourceScanner` is the scanner for text whose kind the caller does
not know. It reads strings by CFScript's rule where the text is CFScript as a
whole (`isScriptFile`), and in a tag file inside its `<cfscript>` regions
alone (`scannerAux.scriptSpans`, from the same split `ClassifyRegions`
makes), leaving markup's strings plain. Every scanner in `internal/resolve`
and `internal/path` starts with it, as do the parser's whole-text walks
(`startup_loops`, `request_mappings`, `mapping_sources`,
`application_sections`); `scriptCustomTags` and `walkActiveTokens`'s script
regions take `NewScriptScanner`, since they know they hold script. The
scanner's rarely set state, a replaying scanner's tokens and a tag file's
script spans, shares one pointer (`aux`), so it stays within 120 bytes.

- **Effect on tokens:** 854 of the corpus's 14,949 files now tokenise
  differently, 718 script files and 136 tag files with `<cfscript>`.
- **Effect on findings:** `unresolved` is byte-identical on RustCFML,
  cfwheels, Mura, FW/1, ContentBox, ColdBox, Slatwall and Lucee. No answer
  depended on those strings, but each walk now reads them as the parse does,
  and a mismatch can no longer come from one.
- **Cost:** none measurable in wall time (ContentBox and Mura, three
  alternated runs each); Mura spends about 2% more CPU.

The rule mattered once in testing: Wheels' mapper compares a method body read
from a tag file (`objects.cfm`, one `<cfscript>` block) with pinned bodies
tokenised alone. Reading the file plain and the pins as script made them
differ; reading both by region makes them agree.

### Done: one comment rule

`<!--- --->` comments were skipped by six readers. The grammar's rule
(`skip_cfml_comment_body` in tree-sitter-cfml's `scanner.h`): comments nest,
an opener is `<!` and three or more dashes, a close three or more dashes and
`>`.

| Reader | Rule it had |
|---|---|
| `skipTagComment`, `skipCFMLComment`, `findScriptSkipSpans` | Nested, `<!---` and `--->`: the same rule written three times |
| The scanner's comment token | Nested, but closed at `-->` |
| `blockEvents` (reaching refs), `producerTags` (producer plans) | Not nested: closed at the first `--->` |

Over the corpus 28 files nest a comment, and in 19 of them a CF tag follows
the inner close while still inside the outer comment, which the two resolver
walks read as live. They now share one reader, `parser.CFMLCommentEnd` /
`SkipCFMLComment` (`internal/parser/comments.go`), written to the grammar's
rule, dash runs included.

`unresolved` is byte-identical on the eight corpora: none of the 19 files'
commented-out tags reached a finding.

Left as they are, since they ask something else of a comment:
`precedingComment` (the doc comment just before a `<cffunction>`, found
backwards), the region split's running depth count, and `scanSourceFacts`'
test for a line that starts as a comment.

### Done: one include-wrapper fact

Two readers knew that a function can include the template its argument
names. The include scan (#364) read any such function, but only for calls in
the same file. The resolver's Wheels reader (`wheels_include_wrapper.go`)
reads calls across files and decides where the include runs
(`application.wo.$include(…)` runs in `Global`), but proved a wrapper only
against newer Wheels' pinned bodies, which route the include through
`$resolveGlobalIncludeTemplate` and `$tryIncludeTemplate`. Older Wheels, the
corpus's own `Global.cfc`, writes `include "#LCase(arguments.template)#"` in
the wrapper itself, and was never proven.

The parser now exports the fact, `parser.IncludeWrappers` (function, argument,
position), and both read it: the include scan for same-file calls, and the
resolver as a second proof beside the pinned bodies (`wheelsIncludeWrapperProven`,
cached per source). `unresolved` is byte-identical on cfwheels: the templates
it includes through `application.wo` have no findings on `main` to lose.

### Done: the Application.cfc walk's includes

The walk that reads an Application.cfc's mappings and custom tag paths
(`mapping_sources.go`) was the last to read an include outside the include
walk. It looked for "include" in every identifier token and read the form by
hand (`includeFormAt`, `includeTagAt`/`includeScriptAt`, a 512-byte window),
so an include in a page's prose or an HTML comment was followed as if it ran.
It now takes the literal-template sites `IncludeSites` reads and follows each
when its token walk reaches the site's offset, so includes still run in
source order between the assignments around them; sites inside a method body
the walk skips are dropped as before. The hand readers and their
regexp-differential test are gone. `unresolved` is byte-identical on the eight
corpora; `TestApplicationSourcesFollowOnlyLiveIncludes` covers comments,
strings, prose and method bodies.

### Done: shared byte classes

About twenty byte-class helpers that several packages each defined byte for
byte (regexp `\s`, quote, word start, identifier, `\w`, ASCII letter, ASCII
lowering) live once in `internal/lex` (`IsRegexpSpace`, `SkipRegexpSpace`,
`IsQuote`, `IsWordStart`, and the existing `IsIdent`, `IsWord`,
`IsASCIILetter`, `LowerASCII`), each tested over all 256 bytes against every
copy it replaced. Three tag-name byte sets in the parser were one, now
`isTagNameByte`. Near-duplicates that differ in a byte keep their own names:
the scanner's table-driven `isIdentPart`, the server's `isIdentByte` (with
bytes past 0x7f), `isNameByte` (with `.` and `$`), the routes' name sets (with
`$`), and `isWhitespace` (with `\v`).

### Done: one line table

Five line tables were built by hand: the parser's `buildLineIdx` and
`lineStartTable`, the tag parser's own binary search, the resolver's
`lineStartsOf` and `newLineIndex` (newline offsets rather than line starts).
They are one `parser.LineTable` (`line_table.go`): line starts as int32,
`LineOf` by binary search, `Start` and `Span`. `TestLineTableMatchesTheTablesItReplaced`
holds it to each copy over generated texts. The resolver's cached tables take
half the memory. `ParseResult.LineOf` serves callers that asked a file's line
once per site by counting from the top (FW/1 `setView`s, included locals,
computed includes, include wrappers). The server's two single-line lookups
stay, since a table per request would cost more. `unresolved` is
byte-identical on ContentBox and Mura.

### Done: the resolver's tag regexes

The resolver read `<cfloop>` (twice over, in `loop_element.go` and
`registry.go`), `<cfargument>` (`orm.go`, `lazy_getter.go`), `<cffunction>`
(`owned_keys.go`, `orm.go`), `<cfreturn>` (`return_element.go`) and
shared-scope `<cfset>` (`startup.go`) with regular expressions of its own:
`<cfloop\b([^>]*)>` and the like. Each ended a tag at the first `>`, even
inside a quoted value, read a commented-out tag as live, and read a value to
either quote. They now read `parser.CFTags` (`internal/parser/cf_tags.go`),
the parse's own reading of a tag: outside `<!--- --->` comments, its extent
past a `>` inside a quoted value (`tagEnd`), its attributes by the CF-mode
attribute scanner, with `Body` for an expression tag's expression and
`WithoutCFTags` for a body without its `<cfargument>`s.
`TestTheResolversTagReadersSkipCommentsAndQuotedGT` holds each reader to a
commented-out tag and a quoted `>`. `unresolved` is byte-identical on the eight
corpora.

### Done: the resolver's call patterns

About thirty regular expressions in `internal/resolve` found framework calls
and statements by running over whole files or function bodies: ColdBox's
setView/view/renderView, announce and customErrorTemplate; FW/1's setView,
view and rc hand-offs; Wheels associations; WireBox `map().to()`; cborm
`super.init()`; ColdBox helpers and includeUDF; Mura `dspObject_include`;
application names; computed includes; and Lucee's registry recipe. They
matched commented-out code and text inside strings, and each read arguments
with a quote rule of its own. They now read the scanner's tokens through
`codeLists`/`codeCalls` (`code_calls.go`): comments are skipped, strings follow
the parse's rule, and a call's arguments are the tokens between its
parentheses. Each `#…#` expression inside a string is read as code in a list
of its own, through the parse's `parser.InterpolatedSpans`, so
`"#renderView('x')#"` is still a render. The hand ports in `text_rules.go` (since deleted) that
existed only to make those regexes fast went with them.
`TestTheCallReadersReadOnlyCode` holds every reader to a commented-out or
in-string call beside a real one, and fails on the regex readers. Left for
another pass: the dynamic per-name `compiledPattern` assignment detectors
(`fw1Assigns`, the prc and lazy-getter writes, isNull/structKeyExists guards).

### Done: the resolver's expression shapes

The resolver's 29 fixed-shape regular expressions that read one short
expression or value are byte readers now, in
`internal/resolve/expression_shapes.go` on an `exprCursor` like the parser's
`textCursor`: keyed reads, `createObject`/`new` constructions, null casts and
Elvis fallbacks, an init's argument copies, `duplicate()`, dotted names, a
`loadBy`'s `returnFormat`. A reader standing for `FindStringSubmatch` returns
the same slice. Each expression stays in `expression_shapes_test.go` as the
oracle, checked on 200,000 generated texts per shape and under
`FuzzExpressionShapes`. `localDeclRe` (a whole line) was left here, since its input
is not one expression.

### Done: function finders read the parse

The parse now records where each function is written:
`ParseResult.FunctionSpans`, with each function's kind, stated access, literal
string defaults and the byte offsets of its declaration, parameters and body.
The script walk records them as it reads each declaration, and the parse's
`<cffunction>` finder records the tag ones. `parser.FunctionSpansOf` is the
functions-only reading of the same region walk, held equal to a full parse
over the fixtures. The resolver's finders read these spans, once per source,
instead of finding functions themselves: `wheelsMethods`, the script producer
plans, the tag function bodies, `functionBodyNamed`, `hasWrittenFunc`, the
cborm init default, the registry factory, the ColdBox handler actions and
`ApplicationSections`. Their token walks and expressions are gone
(`scanApplication`, `writesFunction`, `actionStartRe`, `factoryDefRe`,
`scriptInitRe`). Over the corpus every difference was the old finder reading a
comment, JavaScript in a `.cfm` page, a brace in a string or a `>` in a hint,
or missing a tag handler's actions or an interface's declarations. The
differential also found headers the parse itself lost (`string[]` returns,
modifiers before or after the access, `component function`,
`function function`, colon attributes, a self-closing `<cffunction>`, a
`<cffunction>` in a `<cfscript>` string), now read. What remains:
`<cffunction>` inside triple-backtick tag islands, which the parse blanks, and
`mapping_sources.go`'s own function skipper.

### Done: per-name write detectors

The resolver asked whether code writes or guards a name with a regular
expression built per name at run time and run over the text: `setsName`,
`fw1Assigns`, `throughAssigns`, the last `prc.x =`, `membersWrittenOn`, the
lazy getter's field writes, `leafLiteral`, `initStoredArg` and
`quotedNameLines`; the structKeyExists, keyExists, isDefined, isInstanceOf and
isNull guards; and a ColdBox test's `model=` attribute. Each counted a write or
guard in a comment, or spelled inside a string, as code. They now read the
scanner's tokens in `name_writes.go`: `codeWrites` finds the writes to a dotted
name under a `writeRule` saying which spellings count (a scope before it, any
qualifier, a member, compound operators), a write's `value()` ends with its
statement, and the guards are `codeCalls`. Whole files go through a `textCache`
(`codeOf`) or the parse's memo (`fileCode`), and windows are cut from the
file's tokens, so a comment opened above a window is still one. `text_rules.go`,
the hand ports that made the expressions fast, went with them.
`TestTheWriteReadersReadOnlyCode` holds each to a commented-out or in-string
write beside a real one. Left as expressions: the patterns built from a spelled
value and matched against a name (interception points, Mura object paths,
invoke selectors), and the Wheels self-dispatch, which already runs over the
comment-free token stream.

### Done: the last body regexes

The resolver's last regular expressions over CFML bodies and files read the
scanner's tokens and `parser.CFTags`: script `return` statements counted over a
function (`scriptReturns`, `elementReadAt`), Lucee's script `loop … { }` and
its attributes in the registry recipe (with its string-only brace walker
`braceEnd`), a lazy getter's whole-body guard-and-return shape in tags and
script, a Preside service's header, and a line's `var`/`local.` declaration
(`lineDeclaresLocal`, which was `localDeclRe`). Each took a commented-out
statement, or a word inside a string or a tag's hint, as code; a `}` in a
comment ended a loop's body. The Preside reader still reads a comment on
purpose, but only a `/**` doc comment in the script header, where the engine
reads metadata; for a tag component it reads the `<cfcomponent>` attribute
alone. `TestTheBodyReadersReadOnlyCode` holds each reader to a commented-out or
in-string case beside a real one. `unresolved` is byte-identical on nine
corpora, Preside's included.

### Done: one brace rule

Five readers counted braces or brackets with a rule of their own; they now take
the parser's reading, and no brace in a string, in a `#…#` argument or in a
comment opens or closes anything. The reaching walk's `braceEvents` reads
script with `NewScriptScanner`, and `tagBlocks` asks the parse's region split
(`ParseResult.RegionKindFrom`) in place of an extension check and a 200-byte
look for `<cffunction>`, which changed 4,044 of 68,374 scope starts (mostly tag
component pseudo-constructors and all-script `.cfm` files). The Application.cfc
walk skips `FunctionSpansOf`'s spans in place of `skipMappingFunction`, which
ran a body-less declaration, a `<cffunction />` or prose's "function" on
through the code after it; its expression splitters are one token splitter.
`controls` (guarded calls), `joinAssignmentLine` (`parser.OpenGroups`) and
`componentAttr` (which stopped at a doc comment's `{struct}` and lost ColdBox
`ColdBoxScheduler`'s and cbsecurity `User`'s `delegates`) read scanner tokens.
`ApplicationSources` is identical on all 630 Application files.

With these the resolver has no CFML reader left with rules of its own: what
remains as regular expressions is user patterns, HQL, `tagSharedAssign`
(anchored at a `CFTags` tag start), and patterns matched against names rather
than CFML.

### Done: one route into the parser

`ParseFunctionDefs`, `ParseComponentRefs`, `ParseVars`, `FindFuncScopes`,
`FunctionSpansOf` and `ParseResult.AllVars` ran the script and tag walkers
themselves (`walkRegions`), without the parse's region split by file name, its
output-context gate, the whole-file `<cffunction>` scopes it gives the tag
walk, or its seeding of a `<cfscript>` island with the tag function around it.
Each is now a reading of a shallow `ParseWithOptions`, and the two lighter
readings are options of that one parse: `KeepVars` keeps every declaration the
walk records (for `AllVars` and `ParseVars`), and `ScopesOnly` skips the tag
regions' walks (for `FunctionSpansOf` and `FindFuncScopes`). Over 14,996 files
the scopes, spans and include wrappers are identical; every other difference
is the parse being right: 832 island declarations and 312 island component
refs are filed in their tag function rather than at file level, 12
declarations follow the output gate or the whole-file tag scopes, and the
function definitions carry the parse's return facts and its primitive-return
rule. The readings cost 5 to 20% more each (a `ParseResult` per call); on the
server they run where no parse is cached. The function-body re-walks in
`parse_vars.go`, `parse_func_refs.go` and `parse_calls.go` remain, and are the
lazy-refs step.

### Done: call readers read the parse's calls

The resolver's framework call readers (ColdBox renders and announcements,
FW/1's `view()`, `setView()` and rc hand-offs, Mura's `dspObject_include`,
Wheels associations, WireBox's `map().to()`, cborm's `super.init()`,
`includeUDF` and Lucee's package listings) no longer walk a file's tokens. Each
takes the file's parse and reads the calls it recorded, reading a call's
arguments from the tokens of its span (`parsedCalls`). The plain parse a source
keeps records their names through `ParseOptions.FindCalls`, which without
`ExtractCalls` now runs the walk `ExtractCalls` runs and records only those
names' calls, with spans, receivers and chains, where it searched each line's
text for `name(`. Over 14,996 files such a parse records the same 25,112 calls
as an `ExtractCalls` parse, and every other field equals the plain parse's.
Each reader's answer equals its old token walk's except where the walk took a
function's own declaration for a call: 201 FW/1 rc hand-offs and 16
announcement declarations. The resolver's call index (`textCalls`) is gone.

### Done: write readers read the parse's assignments

The resolver's write readers found a name's writes by walking the scanner's
tokens over the whole text, once per name: `setsName`, `fw1Assigns`,
`throughAssigns`, the last `prc.x`, `membersWrittenOn`, a lazy getter's field
writes, a subclass's literal, `initStoredArg` and a public property's writes.
They now read `parser.Assignment` records, which the parse reads over its own
regions (`ParseOptions.RecordAssignments`, `ParseResult.Assignments`, and
`AssignmentsOf` for text with no parse): the target as written, the operator,
the value's and statement's spans, the line and the tag. The records take the
parse's rules: a write stands only where a statement can (`atStatementLevel`),
`ident ident =` is a script-syntax tag whose attributes are not assignments,
and a value ends where `newlineEndsStatement`, a `;`, a `,` or its bracket ends
it. A differential over 14,996 files found every difference to be the token
walk reading markup, JavaScript, SQL, tag attributes, named arguments or
struct keys as writes, cutting a value at a line break, or losing code after an
unbalanced quote, except two parse gaps left as they are: `=>` inside a
`<cfset>`, and `"<cfscript>"` inside a `<cfset>` string. `codeWrites` remains
only as that test's oracle. Still on their own readers: the guards
(`existsGuards`, `methodExistenceGuarded`), `requestWriteCounts` (a proof
count matched against the producer plans'), and the line-based assignment
reader behind `native_reassignment.go` and `reaching_refs.go`, which feeds
receiver typing and wants its own differential.

For all three, `unresolved` is byte-identical on eight corpora.

### Done: guards and request counts read the parse

The guards (`guardedByExistsCheck`, `guardedInstanceOf`,
`methodExistenceGuarded`) no longer walk a window of the file's tokens. They
read the calls the parse recorded (`guardCalls`): its own when it holds them
(`ParseResult.RecordsCallsOf`), the file's plain parse's otherwise, which now
records `structKeyExists`, `keyExists`, `isDefined`, `isNull`,
`isInstanceOf`, `checkForInstanceOf` and `structInsert`. Over 14,996 files
every guard answers as the token walk did, on 790,759 questions (659 of them
held), and the plain parse and an `ExtractCalls` parse agree on every one. A
guard written in markup outside any output now guards nothing.
`requestWriteCounts` counts the parse's assignments and `structInsert(
request, "k" )` calls rather than searching the text. It differs from the
search on 60 keys, all corrections: 37 keys beginning with `$`, which the
search could not read; 9 written through `variable=`, `returnVariable=` or
`result=`; 5 compound or step writes; and 9 the search wrongly took for
writes (a member of another variable named `request`, and `objectname=` read
as `name=`). Its comparison with the producer plans' count stays
conservative either way. `fileCodeOn` and `fileCodeBetween` are gone. Still
on the scanner: the check that a guard controls the call it precedes
(`controls`), which needs an `if` block's span on the call record, and the
producer plans' statement reader. `unresolved` is byte-identical on eight
corpora.

### Done: receiver typing reads the parse's assignments

Receiver typing's last-assignment lookup (`localAssignmentAt`), the native
reassignment and reach checks (`nativeReassignedAt`,
`nativeClaimMayNotReach`), `reachingUnion`'s walk back through a name's
assignments, `criteriaList` and `assignedFromUntypedFunction` read the
parse's `Assignment` records (`lastAssignmentTo`) in place of a table of
lines beginning `name =` (`matchSourceAssignment`), whose values
`joinAssignmentLine` joined over the lines after them. A typing assignment is
an `=` that starts its statement or is a `var` declaration, to the name bare
or after `local.` or `variables.`; a for loop's initializer is not one. Its
value is `Assignment.ValueLine`, the parse's value written as one line. The
line reader's guard against named arguments, which tokenised everything above
an assignment and gave up past 4,096 tokens or when markup above left a
bracket open, is gone with it, as are `contentFacts` and
`parser.WithoutTrailingComment`. A differential over 14,996 files (683,323
lookups, 1,039,713 reassignment checks, 547,580 reach checks) found every
difference to be the line reader taking markup, JavaScript, comments, tag and
script-tag attributes, named arguments or struct keys for assignments,
missing one not first on its line, joining a value wrongly, or missing a
reassignment its guard gave up on. The exceptions were parse gaps the new
reader inherited, since closed: a `<!-----…----->` comment of dashes alone
never closed, which hid 16 files' code from the region split and `CFTags`
(`CFMLCommentEnd` now closes it as the grammar's comment scanner does);
Lucee tag islands in script (`splitTagIslands` cuts each into a tag region);
and `=>` inside a `<cfset>` (`arrowContinues`). `unresolved` is
byte-identical on seven of the eight corpora; RustCFML loses 2 findings, both
assignments inside a one-line `if` block that the line reader could not see.

### Done: lazy body reads are the parse's walk

`FuncVars`, `FuncRefs`, `FuncCalls` and `HasScopedAssignment` read a function
body the parse had not walked, or one an edit had changed, with walkers of
their own (`parseFuncBody`, `computeAnyScopedVars`, `initBodyVars`, the lazy
ref and call walks), which knew nothing of the regions, output state, doc
comment, imports or typing around the body. Each now runs the parse's own
walk over that one body, `ParseWithOptions` narrowed by `ParseOptions.Body`
(`internal/parser/body_walk.go`), seeded with the output state the whole walk
has where the body starts, and files what it found as the full parse files
it. A write to the variables scope inside a function is a file-level ref, as
the parse files it, where the lazy walk sometimes also filed it under the
function; a tag function's `FuncVars` holds its `<cfargument>` names; text
inside a string no longer declares; an edit that adds a line keeps the calls
of the functions after it; and `HasScopedAssignment` sees an edit that adds a
variables-scope write. Over 14,994 files and 53,519 functions the old lazy
readers differed from the full parse in about 18,000 answers and the new ones
in about 640, of which 631 are member-typed refs whose order of passes
differs (left open). Reads after a keystroke inside a function are as fast or
faster, except `FuncCalls` (up to +71% on Lucee's largest file, 2.3ms to
3.8ms), which only the deps export and FW/1's rc reader call. `unresolved` is
byte-identical on eight corpora, and batch scans read their text about a
tenth fewer times (ContentBox 13.6 to 12.4 passes, Mura 15.2 to 13.1).

### Done: file references are recorded by the parse's walk

Includes, document links and custom tag calls were each read by walks of their
own over the parse's regions: a script region's tokens were scanned once for
its custom tags and again for its includes after the parse had scanned them,
and a document-links request scanned them again. The parse's walk now records
them (`internal/parser/file_refs.go`). A script region's scanner hands each
token it produces to a recorder (`scriptRefs`) that reads include statements,
`cfinclude()`, `cfmodule()`, Lucee's module statement, `cf_name()` and the CF
tags among the statements, recording a token the walk reads again after a
lookahead only once and scanning what the walk leaves unread. The tag walk
drives a reference reader (`tagRefReader`) at each tag it meets, which reads
the tags the walk stepped over as it catches up. Every parse records includes;
`ExtractLinks` adds links, custom tags and tag names, and files the walk's
links from the record; `ParseOptions.FileRefsOnly` reads references and
nothing else, which the text entry points (`IncludeSites`, `ExtractLinks`,
`CustomTagCalls`, `IncludeWrappers`) and a parse asked for what it did not
record now read. Include wrappers use the parse's own functions. Over 14,994
files every answer equals the old readers', once their two readings of a
closing tag became the custom-tag reader's: a closing tag ends where a tag
opens before its `>`, and a `</` with no name is a stray `<` (three files'
prefixed tags; no link or include changes). `unresolved` is byte-identical on
eight corpora and the code map's include edges are unchanged. Left: after a
keystroke inside a function the record is read again on the next links
question rather than spliced; `wrapperIncludes` and `listsTemplates` still
read text, in the few files that need them.

## Speed is an option of the one walker, not a second walker

Some walkers exist because they do less. `globalScriptParser` reads only
declarations, `findScriptFuncScopes` only braces, and the shallow parse skips
calls. Consolidating keeps that speed: the one walker takes options for how
much it extracts, rather than each lighter job growing its own walker that
then drifts. The parse already works this way through `ParseOptions`
(`ByResolversOnly`, `ExtractCalls`, `ScanAllScopes`).

Concretely:

- **Variables only:** an option of the script walk that records declarations
  and scopes and skips refs, calls, links and typing. This replaces
  `globalScriptParser`, with the same reading of what a declaration is.
- **Scopes only:** an option that tracks function boundaries alone. This
  replaces `findScriptFuncScopes` where no full parse is at hand.
- **Shared rules:** each option skips work, never a rule. The struct-key and
  named-argument fix, the `==` handling and the region split apply in every
  mode, so a light pass and a full pass cannot disagree.
- **Proof:** a differential test runs each light mode against the full walk
  over the corpus fixtures, and requires its output to be a subset of the
  full walk's, with nothing extra.

## Order

| Step | Change | Size of the defect it removes |
|---|---|---|
| 1 (done) | Function scopes from the parse, or its scopes-only mode where no parse is at hand; drop `findFuncScopesIn` (section 2) | 1,723 wrong scopes; 230 `AllVars` differences |
| 2 (done) | One region split for every walker (section 4) | 65 invented declarations, and every walker agreeing on what is script |
| 3 (done) | Main parse: struct keys and named arguments (section 1, step 1) | 776 invented declarations |
| 4 (done) | Replace `globalScriptParser` with the main walk's kept declarations (section 1, steps 2–4) | 3,060 wrong declarations, and a second script walk per region |
| 5 (done) | Lazy refs filed like the full parse's: a body read is the parse's walk of that body (`ParseOptions.Body`) | About 18,000 lazy answers that differed from the full parse's |
| 6 (done) | The resolver's function finders read strings as the parse does, or the parse's spans (section 5) | 417 to 1,208 lost functions per finder |
| 7 (done) | One line counter, the LSP's (section 6) | Every position in 66 CR-only files |
| 8 (done) | Tag attributes from one reader, with a CF and an HTML mode (section 7) | 1,815, 1,702, 524 and 516 wrong attributes in the other four |
| 9 (done) | `tagEnd` stops at a `<` inside a quoted value (section 7) | Every reader's attributes after it | A body reparse for 1,309 functions |

Each step is judged the same way: the classifier and judge before and after,
and the six-corpus `unresolved --json` diff. Here a changed `unresolved`
result is expected, and each changed finding has to be one of the classes
above.
