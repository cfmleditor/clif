# Grammar gaps: constructs `tree-sitter-cfml` cannot parse

`FORMATTER-ISSUES.md` counts files the grammar refuses — currently 22 documents
and 32 embedded cfscript/cfquery regions. A count is not something a grammar
maintainer can act on. This file turns the ones that are worth acting on into
constructs, and says which are not worth acting on at all.

## Method

Two parsers over the same six-project corpus (5,624 files):

- **`tree-sitter-cfml`**, through `make corpus`.
- **[RustCFML](https://github.com/RustCFML/RustCFML)**, a CFML interpreter in
  Rust. Its parser is a useful oracle precisely because it is part of a working
  engine: it has to accept what a real engine accepts. Driven through a harness
  over its `cfml-compiler` crate that mirrors `cfml-vm`'s own pipeline —
  `has_cfml_tags` → `tags_to_script_checked` → `Parser::parse`.

|  | RustCFML parses | RustCFML refuses |
|---|---|---|
| **tree-sitter parses** | 5320 | 247 |
| **tree-sitter refuses** | **32** | 25 |

The 32 are the interesting cell. Each was reduced with `make shrink` to the
smallest contiguous fragment that still fails, and each fragment was then run
through *both* parsers again — a fragment that loses the context that made
RustCFML accept the file is not a repro. 24 of the 32 survive that.

**RustCFML is the more permissive parser, so "RustCFML accepts it" is evidence,
not proof.** Six of the 24 are Lucee's own *negative* fixtures — files whose
names say they are meant to fail — and there tree-sitter is right and RustCFML
is wrong. They are listed below so nobody files them. The remaining eighteen
first read as six constructs; isolating each by hand cut that to four, filed as
tree-sitter-cfml
[#116](https://github.com/cfmleditor/tree-sitter-cfml/issues/116)–[#119](https://github.com/cfmleditor/tree-sitter-cfml/issues/119).

The 247 in the other direction are RustCFML's gaps on a Lucee-heavy corpus,
not ours.

## Not gaps: negative fixtures

tree-sitter correctly refuses these. RustCFML accepting them is RustCFML being
lax about errors it would raise at runtime instead.

| File | Fragment |
|---|---|
| `Lucee/test/tickets/LDEV5900/bad1.cfm` | `<cfset x = >` |
| `Lucee/test/tickets/LDEV5900/bad2.cfm` | `<cfset unclosed string` |
| `Lucee/test/general/Struct/invalid1.cfm` | `var x={susi.sorglos,peter};` |
| `Lucee/test/general/Struct/invalid2.cfm` | `var x={variables.susi,peter};` |
| `Lucee/test/general/Struct/invalid3.cfm` | `var x={susi(),peter};` |
| `Lucee/test/tickets/LDEV3060/invalidcomponent.cfc` | `domain_id :: oDnsDomain.getId()` — `::` where `:` was meant |

## Verified against the grammar, one construct at a time

A fragment is a *file*, and a file that fails is not yet a construct. Each of the
six candidates was therefore re-cut into the smallest standalone source that
still reproduces, run against `tree-sitter-cfml` **HEAD** (`a4b5b72`, `v0.26.35`
— the version this repo pins, and the newest published), and checked against the
grammar's own `LIMITATIONS.md` and issue tracker before being called a gap.

That step changed five of the six. `component { package final whatever function
f() {} }` parses on its own, and so does every one of `final component`, `final
function f()`, `final public function f()`, a `thread { … }` statement, a tag
island, and a `<cfif>` spanning whole tag attributes. What fails is narrower than
the file made it look, and in two cases what fails is already settled upstream.

### Already known upstream — do not file

| Construct | Status |
|---|---|
| `final` on a parameter — `function f( final required s )` | In `LIMITATIONS.md`. Implemented, measured, rejected: `( final (` is ambiguous at every parameter list in the language. |
| `final component`, `final function f()`, `final public function f()` | All parse. Fixed by [#77](https://github.com/cfmleditor/tree-sitter-cfml/issues/77) / [#69](https://github.com/cfmleditor/tree-sitter-cfml/issues/69). |

So Lucee's `test3671.cfc` is one gap, not three, and it is a closed question.

## Filed: four constructs

Each reproduces standalone on HEAD and RustCFML parses each. All four are now
open against `tree-sitter-cfml`:
[#116](https://github.com/cfmleditor/tree-sitter-cfml/issues/116),
[#117](https://github.com/cfmleditor/tree-sitter-cfml/issues/117),
[#118](https://github.com/cfmleditor/tree-sitter-cfml/issues/118),
[#119](https://github.com/cfmleditor/tree-sitter-cfml/issues/119).

### 1. An arrow function with an empty body ([#116](https://github.com/cfmleditor/tree-sitter-cfml/issues/116))

`Lucee/test/tickets/LDEV4062/LDEV4062.cfm`. Fails in `cfscript/grammar.js` and,
reached through `<cfset f = function() { … }>`, in `common/define-grammar.js`.
`=>` and `->` both fail; the parameter list may be empty or not.

```cfml
x = () => ;
y = 1;
```

Not [#75](https://github.com/cfmleditor/tree-sitter-cfml/issues/75), which is a
*statement* as the body. The empty body is worse than an error: at end of file
`x = () => ;` yields an `arrow_function` whose body is a `number` holding a
MISSING token, and with a following statement the parser takes `y = 1` as the
lambda body.

### 2. A return type between two access modifiers ([#117](https://github.com/cfmleditor/tree-sitter-cfml/issues/117))

`Lucee/test/general/modifiers/All.cfc`

```cfml
component { public struct static function f() {} }
```

`struct public static function f()` parses ([#88](https://github.com/cfmleditor/tree-sitter-cfml/issues/88))
and so does `public static struct function f()`. Only the interleaved spelling
fails — the return type is accepted before the modifier run or after it, not
inside it. Lucee's file writes four such members with a user-defined type name.

### 3. A `thread { … }` statement followed by a tag island ([#118](https://github.com/cfmleditor/tree-sitter-cfml/issues/118))

`Lucee/test/tickets/LDEV4157/LDEV4157.cfm` and `test4157.cfc`

````cfml
thread name="t" {
	thread.test = "thread";
}

```
	<cfset res = "works">
```
````

The statement alone parses and the island alone parses; together the grammar
reports a MISSING `;` at the closing brace. Already recorded in the grammar's
`LIMITATIONS.md`, but never filed, so nothing tracks it.

### 4. A start tag split across `<cfif>` branches ([#119](https://github.com/cfmleditor/tree-sitter-cfml/issues/119))

Two shapes, both from ContentBox, both ordinary CFML. The closing `>` inside the
branches (`views/authors/editor.cfm`):

```cfml
<a
	title="Back"
	<cfif x>
		href="a">
	<cfelse>
		href="b">
	</cfif>
	Back</a>
```

and the whole tag *opening* inside them, closed after the `</cfif>`
(`views/settings/rawSettingsTable.cfm`):

```cfml
<cfif x>
	<a
		disabled="disabled"
<cfelse>
	<a
		class="confirmIt"
</cfif>
	title="Delete"
>
```

A `<cfif>` that spans only whole *attributes* — `<a <cfif x>href="a"<cfelse>href="b"</cfif>>`
— parses. It is putting the `>` on the branch side that fails, in both HTML and
`<cf…>` tags.

## Galaxie Blog: 15 parse errors, in 11 files

[Galaxie-Blog](https://github.com/GregoryAlexander77/Galaxie-Blog) `d40017b` (417 files) is refused
at 15 places by `clif scan`, against `tree-sitter-cfml` v0.26.43 (this repo's pin). Each was cut
down to a standalone source and run through `clif scan` again, as above; none of this was checked
against RustCFML or an engine, so "invalid source" below means the source is plainly malformed, not
that an engine rejects it.

**Three new grammar gaps, fixed on an unreleased branch** (A and B below, C after the table). See "Fixed" at the end of this section.

*A. `<!--` inside a CFML comment* (3 errors: `admin/adminInterface/postScene.cfm:17`,
`sceneDetail.cfm:34`, `common/cfc/Renderer.cfc:196`). Fails; the same comment with `("</")` alone
parses:

```cfml
<!--- a ("<!--") b --->
<cfset x = 1>
```

*B. `</script` followed by anything in a `<script>` body* (1 error: `includes/templates/blogJsContent.cfm:2018`,
the string `'</script'` at line 2139). The grammar ends the element at the first `</script`. HTML
ends it only when those letters are followed by whitespace, `/` or `>`:

```cfml
<script>
var s = '</scripty';
</script>
```

`'</script'` in a comment (`// a '</script' b`) fails the same way.

**Already filed as [#119](https://github.com/cfmleditor/tree-sitter-cfml/issues/119)** (a start
tag or element whose structure is split across `<cfif>` branches; 2 errors):

- `includes/templates/blogContentCss.cfm:58`: `<cfif not standAlone><style><cfelse>…`, the `<style>`
  opened in one branch only. Repro: `<cfif a><style><cfelse><cfcontent type="text/css" /></cfif>`.
- `galaxiePlayer.cfm:219` (reported on the `<cfif>` after `</style>`): an attribute quote opened
  inside a `<cfif>` and closed after it, `<video <cfif a>poster="<cfoutput>#u#</cfoutput></cfif>" id="p">`.
  Without the `poster=` line the file's other `<cfif>`s inside the `<video>` tag parse.

**Invalid source, nothing to file** (6 errors):

| Where | What |
|---|---|
| `Application.cfc:430`, `:440` | `<cfset … / >`: a space between `/` and `>` |
| `common/db/importBlogCfc.cfm:454` | `<cfparam name"x" …>`: no `=` |
| `common/db/importBlogCfc.cfm:941` | `</cftry>` with no `<cftry>` before it (the only open tag is at 1069) |
| `admin/adminInterface/themeSettings.cfm:2919` | `<input …` never closed with `>` |
| `includes/postContent/entry/webp.cfm:118` | `</p` with no `>` |

**The third new gap**, the last 3 errors (3 + 1 + 2 + 6 + 3 = 15) (`common/cfc/Renderer.cfc:3061`, 2 errors, and
`common/cfc/StringUtils.cfc:20`):

*C. A string holding `<!--<` followed anywhere later by a comment.* The comment can be CFML or HTML;
the string's tag-shaped text is what matters (`"<!--x"` and `"<a"` alone are fine):

```cfml
<cfset s = "<!--<a">
<!--- c --->
```

This is `replaceNoCase( s, "<!--<attachScript", "<script", "all" )` in Galaxie's script-protection
helpers. Neutralising gap A's comment in `Renderer.cfc` left the line 3061 errors in place, so
C is independent of A.

### Fixed on `tree-sitter-cfml` branch `ccr-b65f0ac3-psafqh` (unreleased, v0.26.43 base)

All three are scanner-only (`common/scanner.h`), so the parse table is unchanged.

- **A**: `scan_comment` counted a two-dash `<!--` as a nested CFML comment. A CFML comment now nests on `<!---` and closes on `--->` only. (Closing on `-->` alone had to go too: MuraCMS's `core/modules/v1/feed/index.cfm` writes an HTML comment inside a CFML comment, which balanced only because the opener was counted. The first version of the fix broke that file; the corpus scan caught it.)
- **B**: `scan_raw_text` and `scan_html_text` ended a script or style element at the first `</script` / `</style`. They now require the whole name followed by whitespace, `/` or `>`.
- **C**: in a state that accepts nothing but the comment (the inside of a string), `<!--<` was taken for an IE conditional comment. A two-dash opener there is now text; `<!---` still tokenises, as the engine strips it.

Verified in `tree-sitter-cfml`: four new corpus tests and three probes, each of which fails with `scanner.h` reverted; `npm test`, `probe`, `check:keywords` and `lint` pass; the 17,176-file corpus scan is identical per file (235 error lines in 96 files) and `treediff` reports no tree-shape change in 15,793 files that parsed before.

Verified here with the local grammar through a `-modfile` replace (`go.mod` untouched): `clif scan` on Galaxie goes from 15 errors to 8; `go test -short ./cmd/... ./internal/...` passes; the formatter corpus moves 405 -> 410 clean and 11 -> 6 grammar-refused, and the five files that changed verdict (`postScene.cfm`, `sceneDetail.cfm`, `Renderer.cfc`, `StringUtils.cfc`, `blogJsContent.cfm`) are the five that were refused for A, B and C. Nothing here changes `internal/parser`: `gapcheck` already agreed with the grammar on these files.

To take it: release `tree-sitter-cfml`, then `make update-grammar`. Until then clif still reports the 15.

What remains is the 8: 2 are #119 (`blogContentCss.cfm`, `galaxiePlayer.cfm`; not fixed) and 6 are invalid source (the table above).

### Tickets

[tree-sitter-cfml#172](https://github.com/cfmleditor/tree-sitter-cfml/issues/172) (gap A),
[#173](https://github.com/cfmleditor/tree-sitter-cfml/issues/173) (B) and
[#174](https://github.com/cfmleditor/tree-sitter-cfml/issues/174) (C), fixed on the branch above;
[#119](https://github.com/cfmleditor/tree-sitter-cfml/issues/119) (Galaxie's two shapes added as a comment);
[#176](https://github.com/cfmleditor/tree-sitter-cfml/issues/176) (a tag comment after `cflock(…){ }`, found
reducing the corpus; fixed on the same branch: the tag statement demanded a `;` after its block, so the comment was
never offered to the scanner);
[#75](https://github.com/cfmleditor/tree-sitter-cfml/issues/75) (`(s)=>return s` added as a comment);
[#177](https://github.com/cfmleditor/tree-sitter-cfml/issues/177) (a literal `<` before a hash expression,
`<span><#c#</span>`, is read as an unclosed dynamic tag; MuraCMS's setup form; reduced, not fixed);
[#175](https://github.com/cfmleditor/tree-sitter-cfml/issues/175) tracks the corpus scan's remaining errors, and
every one of its five largest clusters is now accounted for: one real gap (#176, fixed), one more (#177), one that is
the existing #75 (`(s)=>return s`), BoxLang's `static foo = …` (out of scope, not filed), and the rest malformed
source or raw JavaScript in a `.cfm`.
One open question there: two files end with an unclosed `<cfscript>`, which Lucee's own test suite includes; whether
an engine accepts that has not been checked. clif#317 is the path-spelling difference in
`unresolved`, clif#318 the grammar bump.

## Shrank poorly: twelve files still to reduce

These reproduce, but `make shrink` could not isolate a small fragment — the
failure needs context the contiguous-fragment search cannot keep. They need
reducing by hand or a smarter shrinker.

| Chars | Refusal | File |
|---|---|---|
| 842 | script | `Lucee/test/tickets/LDEV5389.cfc` |
| 899 | script | `Lucee/test/tickets/_LDEV3623.cfc` |
| 1202 | document | `Lucee/test/tickets/LDEV1750.cfc` |
| 1236 | script | `Lucee/test/tickets/LDEV3113.cfc` |
| 2411 | document | `Lucee/test/tickets/LDEV0869.cfc` |
| 2829 | document | `Lucee/core/src/main/cfml/context/wddx.cfm` |
| 6863 | document | `Lucee/core/.../admin/debugging.templates.create.cfm` |
| 10028 | document | `Lucee/core/.../admin/services.gateway.create.cfm` |
| 21520 | document | `Lucee/core/src/main/cfml/context/formtag-form.cfm` |
| 21558 | document | `Lucee/core/src/main/cfml/context/form.cfm` |
| 25684 | script | `Lucee/core/src/main/cfml/context/admin/Jira.cfc` |
| 78998 | script | `ContentBox/modules/contentbox/models/system/CBHelper.cfc` |

## Reproducing

```bash
CORPUS=/src/Lucee:/src/ContentBox:/src/coldbox-platform:/src/fw1:/src/testbox:/src/cfmleditor
make corpus CORPUS=$CORPUS REPORT=/tmp/corpus.tsv
make shrink REPORT=/tmp/corpus.tsv
```

For the RustCFML side, build a harness against its compiler crate — three
dependencies, about 35 seconds — that mirrors `crates/cfml-vm/src/lib.rs`'s
parse pipeline, and run it over the same file list. It parses the whole corpus
in under a second, so the comparison is cheap enough to repeat whenever the
grammar is bumped.

### Taken: tree-sitter-cfml v0.26.44

The pin moved from v0.26.43 to v0.26.44, which carries the four fixes above (cfmleditor/tree-sitter-cfml#172, #173,
#174, #176; released as v0.26.44, `6ea9e53`). `injections.scm` is identical at the new tag.

Measured against the old pin on the same `main`:

- `clif scan` on Galaxie Blog: **15 parse errors -> 8** (the remaining 8 are #119 and malformed source, above).
- The formatter corpus (17,567 files: the `tree-sitter-cfml` corpus and Galaxie), per file with `make corpus BASELINE=`:
  **seven files change verdict, all refused -> clean, none the other way**: Galaxie's `postScene.cfm`, `sceneDetail.cfm`,
  `Renderer.cfc`, `StringUtils.cfc` and `blogJsContent.cfm`, plus Lucee's `test/functions/DirectoryCreate.cfc` and
  RustCFML's `bench/ws/probe.cfm` (#176). Galaxie: 406 -> 411 clean.
- `go test -short ./cmd/... ./internal/...` and `golangci-lint` pass.
