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
  `include "x"`, `include template="x"` and `cfinclude(template="x")`, as
  `includeScriptAt` does; `cfmodule(…)` and `module template=` come from the
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
- The Application.cfc walk (`mapping_sources.go`) still reads an include at
  a keyword its own walk finds, with `includeFormAt`; the expression
  differential holds those readers.

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
| 5 | Lazy refs filed like the full parse's; then pre-fill (section 3) |
| 6 (done) | The resolver's function finders read strings as the parse does, or the parse's spans (section 5) | 417 to 1,208 lost functions per finder |
| 7 (done) | One line counter, the LSP's (section 6) | Every position in 66 CR-only files |
| 8 (done) | Tag attributes from one reader, with a CF and an HTML mode (section 7) | 1,815, 1,702, 524 and 516 wrong attributes in the other four |
| 9 (done) | `tagEnd` stops at a `<` inside a quoted value (section 7) | Every reader's attributes after it | A body reparse for 1,309 functions |

Each step is judged the same way: the classifier and judge before and after,
and the six-corpus `unresolved --json` diff. Here a changed `unresolved`
result is expected, and each changed finding has to be one of the classes
above.
