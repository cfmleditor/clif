# Galaxie Blog: what is left

[Galaxie Blog](https://github.com/GregoryAlexander77/Galaxie-Blog) `d40017b` (417 CFML files, no `.clif.json`) was added
to the public corpus. This records what is still open after that work, so it can be picked up without re-deriving it.
What was fixed along the way is in `RESOLUTION-GAPS.md`, `GRAMMAR-GAPS.md` and `PARSER-GAPS.md`. Nothing below has
been investigated beyond what each section says.

Reproduce, with an absolute path (a relative root gave different results before clif#319):

```sh
clif unresolved --json /abs/path/to/Galaxie-Blog          # the report; "line" in the JSON is 0-based, the text report's is 1-based
clif unresolved --json --candidates /abs/path/...         # adds the liberal matches per finding
clif explain --root /abs/path/to/Galaxie-Blog <file> <line> [call]   # the lookup steps for one call (1-based line)
clif scan /abs/path/to/Galaxie-Blog                       # grammar parse errors
make gapcheck CORPUS=/abs/path/to/Galaxie-Blog            # parser vs grammar, per file
make corpus CORPUS=/abs/path/to/Galaxie-Blog BASELINE=<an earlier report>
```

## 1. Unresolved calls: 746

Counts are from `unresolved --json` with the build that has clif#325 (879 before it, 3,752 before any of this work).

| Kind | Findings |
|---|---:|
| `variable 'x' has no component ref` | 595 |
| `x (no qualifier, not in file)` | 102 |
| `method 'f' has no component return type (chain to 'g')` | 18 |
| `method 'f' not found in X` | 18 |
| `method … not found on builtin native.string` | 9 (`WebForms.cfc` 4, `ThemeEditor.cfc` 3, `Setup.cfc` 2: `source` 3, `oldField` 3, `owner` 2, `oldPlacement` 1) |
| `method … not found on builtin native.zoneddatetime` | 3 |
| `component 'ChangeLogType' does not exist` | 1 |

Most findings are in a few files: `common/cfc/Shop.cfc` 93, `org/captcha/captchaService.cfc` 67, `common/cfc/JSoup.cfc` 61,
`org/camden/blog/blog.cfc` 46, `common/cfc/WebForms.cfc` 45, `Bookings.cfc` 42, `Payments.cfc` 38, `TimeZone.cfc` 35,
`Events.cfc` 34, `includes/templates/blogContentHtml.cfm` 32.

### "No component ref", by receiver (595)

Where each receiver's findings are, from the report. What each is assigned from has **not** been looked at for any of them;
that is the work.

| Receiver | n | Files |
|---|---:|---|
| `html` | 63 | `WebForms.cfc` 40, `Events.cfc` 23 |
| `e` | 45 | `Bookings.cfc` 30, `Payments.cfc` 8, `Shop.cfc` 7 (98 before the HQL change in clif#319) |
| `o` | 42 | `Shop.cfc` 42 |
| `owner` | 27 | `Setup.cfc` (all). `owner` is also typed as a string somewhere: 2 calls on it are "not found on builtin native.string" at `Setup.cfc:612` |
| `entity` | 25 | `Payments.cfc` 22, `Members.cfc` 3 |
| `arguments.entity` | 22 | `Locations.cfc` 12, `Members.cfc` 10 |
| `arguments.e` | 12 | `Shop.cfc` 12 |
| `order` | 24 | `Shop.cfc` |
| `product` | 7 | `Shop.cfc` |
| `variables.tzObj`, `tmptz` | 19, 10 | `TimeZone.cfc` |
| `startInstant` | 6 | `Bookings.cfc` |
| `PostJSoupObj`, `JSoupObj`, `JSoupDocObj`, `FragmentObj` | 16, 14, 12, 6 | all `JSoup.cfc`; `application.jsoupComponentPath` (7) is the same family (`ProxyController.cfc`, `Renderer.cfc`, `importBlogCfc.cfm`) |
| `graphics`, `arguments.graphics`, `point` | 15, 11, 9 | `captchaService.cfc` |
| `rgb`, `javaColor` | 6, 9 | `blog.cfc` (6 each), `Image.cfc` (3 of `javaColor`) |
| `setup` | 15 | `ProxyController.cfc` 9, `AdminHome.cfc` 6 |
| `scenes`, `editor` | 7, 7 | `ProxyController.cfc` |
| `item.post` | 12 | `TestUtility.cfc` |
| `queryObj` | 7 | `cfJson.cfc` |
| `RendererObj`, `HtmlUtilsObj` | 9, 8 | `renderKendoCardGrid.cfm` 7 and `importBlogCfc.cfm` 2; `bingMap.cfm` 5, `roles.cfm` 2, `bingMapRoute.cfm` 1. Both are made with `<cfobject component="#application.…#">`, the family the include fix in clif#319 covered for 490 of 498; why these still read as untyped was not examined |

The fastest way in is `clif explain --root <Galaxie> <file> <line>` on one call per row: the trace names the step that gave up.

### "No qualifier, not in file" (102)

A bare call to a function the file neither declares nor inherits. By name: `getPageMode` 40, `getXmlOverride` 11, `addToLog` 9,
`isPageMode` 8, `buildCacheKey` 5, `ensureColumns` 4, `formatRSSOffset` 4, `getTemplatePathByPageName` 3, `getUserId` 3,
`formatAtomDate` 3, then a tail. Not traced: whether they are UDFs defined in an included template that the include graph
does not reach (a computed `<cfinclude template="#…#">`), or genuinely missing.

### "Method not found in X" (18)

Mostly entity setters called by the legacy import script, e.g. `setPostRef` on `Subscriber` and `Media`,
`setEntriesPerBlogPage` and `setBlogModerated` on `Blog`, `setBlogRef` on `Capability`. Not checked against the entity
definitions: these may be real mismatches between `common/db/importBlogCfc.cfm` and the current entities (a one-off
import utility), not resolver gaps.

## 2. Grammar parse errors: 8 with tree-sitter-cfml v0.26.44

Fixed by the v0.26.44 pin (15 → 8): the scanner gaps tree-sitter-cfml#172, #173, #174 and the tag-comment gap #176.
The 8 left:

| Where | What | Status |
|---|---|---|
| `blogContentCss.cfm:58`, `galaxiePlayer.cfm:219` | an element or an attribute quote split across `<cfif>` branches | tree-sitter-cfml#119 class. **Unsupported by decision**, recorded in that repo's `LIMITATIONS.md` and as probes recorded `fail`. Issue stays open |
| `Application.cfc:430`, `:440` | `<cfset … / >` | tree-sitter-cfml#181. Probably valid to an engine; **not run on Lucee or Adobe ColdFusion** |
| `themeSettings.cfm:2919`, `webp.cfm:118` | unclosed `<input`, `</p` | malformed HTML inside valid CFML; the grammar models HTML structure |
| `importBlogCfc.cfm:454`, `:941` | `name"x"`, a `</cftry>` with no opener | CF tag errors in a one-off import utility; probably rejected by an engine, not checked |

A pull request to the Galaxie Blog project (from the `cfmleditor/Galaxie-Blog` fork, branch `ccr-b65f0ac3-psafqh`) fixes
the `/ >`, the unclosed `<input`, the `</p` and the missing `=`, and asks the maintainer about the `</cftry>`. Its outcome
is not known here. tree-sitter-cfml#177 (a literal `<` before a hash expression) is also unsupported by decision; it
occurs in MuraCMS, not Galaxie.

## 3. Parser against grammar

`make gapcheck CORPUS=<Galaxie>` lists, per file, the calls only the grammar sees and the calls only the parser sees. With
the merged grammar the parser-only side is 262 entries, all `<cfinvoke … method="x">`, which the parser records as calls
and the grammar sees as tags: deliberate, like `createObject`. They are not in `expectedDifferences` (that list holds the
repo's fixtures to recorded differences; a corpus run only reports).

## 4. Other open items

- **Does an engine accept an unclosed `<cfscript>` at the end of a file?** Two corpus files end that way, one of them
  Lucee's own `test/tickets/LDEV6377_regression.cfm`. Needs a Lucee run; the grammar rejects it (tree-sitter-cfml#175 has the note).
- **`/ >` and the six errors above** need the same engine check to settle whether any is a grammar gap or invalid input.
- **tree-sitter-cfml#175** (corpus tracking) has every cluster accounted for and can be closed.
- **v0.26.44 publish**: the tag exists; that the npm, crates.io and PyPI publishes completed was not checked.
- **`TestFoldingUnterminatedTagsScalesLinearly`** (`internal/parser`) failed twice in local whole-suite runs under load and
  passes alone, in its package, and in CI. It is an allocation-ratio test; making it less load-sensitive would stop the false alarm.
- **`make vuln` locally**: it fails here on clean `main` too (govulncheck wants `go mod tidy`, which trips over the Go
  toolchain's own test directory under `target/` in this environment); CI's `vuln` job is the check.
- **Two parser behaviours** found on the way and fixed are worth remembering as the places to look for the next: `tagBody`
  once looked for a space (not any whitespace) after the tag name, and the span scan once duplicated the expression scan
  for tags that are only an expression (`PARSER-GAPS.md`).
