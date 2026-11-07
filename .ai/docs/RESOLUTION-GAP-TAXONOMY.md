# What `unresolved` still reports, and why (November 2026)

Seven corpora (Galaxie, TestBox, FW/1, ColdBox, ContentBox, Lucee, MasaCMS), measured
with `clif unresolved --json` and `unresolved-diff`. 8,215 findings when the long-tail
work began; **1,855** after the fixes below. This note classifies those 4,214 by the
information that would resolve each one, so the next piece of work can be picked by
yield and risk. The scripts that produced the numbers are not committed; the method is
under "How to reproduce".

## The taxonomy

Classified from the assignment that made the variable (the nearest `x = …`,
`var x`, `<cfset x`, loop header, parameter list or callback), then confirmed on a
sample with `clif explain`.

| Share | Class | What would resolve it | Ceiling |
|---|---|---|---|
| 20.9% (881) | **Untyped parameter** (`arguments.x`, or a bare `x` that is one) | a type, or the callers' types | see below |
| 8.7% (366) | Origin not found by the classifier | (assignments the line scan misses: multi-line, tag attributes, `cfloop query`) | unknown |
| 8.4% (356) | **Unknown function** (`no qualifier, not in file`, `not found in extends chain`) | a definition that is not in the workspace | mostly real |
| 8.2% (345) | **Loop variable** over a collection of unknown element type | the element type of the collection | 168 are ContentBox views |
| 6.6% (277) | `x = recv.method()` where the call's result is unknown | the method's return | Java/native interop |
| 6.5% (275) | **Field** (`variables.x`) set elsewhere | what every writer assigns | writers disagree or are untyped |
| 5.4% (227) | **Callback / closure parameter** | the provider's contract | per framework |
| 5.0% (209) | Method missing on a known component | nothing: these are findings | real |
| 4.6% (195) | `x = f()` where f is declared with a return type or is not declared | the return type | see below |
| 4.2% (178) | **`prc.x` / `rc.x`** | the handler that set it | ContentBox views |
| 4.1% (174) | Component does not exist | the component | real, or an absent module |
| 3.2% (135) | `attributes.x` (custom tags) | what the tag's callers pass | Masa 100 |
| rest | indexed reads, aliases, ternaries, scopes (`request.`, `application.`) | each its own | small |

### 1. Untyped parameters are 21% of everything, and few parameters carry it

265 distinct parameters carry 1,086 findings; **Masa's 564 findings are 45 parameters
and ContentBox's 257 are 59**. The names are conventional (`feedBean` 113, `contentBean`
79, `content` 63, `site`, `author`, `page`, `comment`, `setup`, `item` 132).

Where each stands against the inference rule ("every caller must type it"):

- 0 callers in the workspace (public API, event handlers, `remote` functions): 11%.
  Nothing in the source can say. A `@param` type or a convention is the only evidence.
- 1 caller: 19%. Almost all of these forward **their own** untyped parameter
  (`createSite( arguments.setup )` from `public createSite( setup )`), so the chain
  ends at an entry point whose callers are the framework or a test.
- 2 or more callers: 69%. Inference declines when any one caller passes something
  untyped, which in a codebase of untyped parameters is the norm.

So the gap is not an inference weakness; it is that the information is not in the
source. Three kinds of evidence exist outside it, in order of cost:

1. **A name convention in the preset** (data, not code). Done for Mura's `contentBean`,
   `feedBean` and `userBean`: 221 of Masa's findings, none added. The same table
   for ContentBox's `site`/`author`/`page`/`comment`/`category` would take most of its
   259, but `content` is Page, Entry or ContentStore, and a convention that is
   usually right is a claim: it belongs in a ContentBox preset that declares that.
2. A `@param` or `@arg.type` in the doc comment (ColdBox documents most of its
   parameters by name only; no type).
3. The test suites: a spec that calls `createSite( mock )` says what the parameter is.
   Nothing reads these today.

### 2. A function that declares no return type (done) and one that does (not)

`x = f()` with `f` declared and untyped was 321 of the 486 bare-call findings. It is
accepted as dynamic now, as `x = svc.get()` already was for a method
(`assignedFromUntypedFunction`, 305 findings). What remains in that class:

- **83** where `f` declares `struct`/`array`/`string`. Typing `x` from the declaration
  is sound for string, array, number, boolean, date and query (the engine enforces
  them) but **not for `struct`: a component satisfies it**, and Wheels' `$mapper`
  declares `struct` and returns a Mapper. Using it broke two Wheels tests and added
  false findings. A function that declares `struct` and whose body returns only struct
  literals could be typed, but no accessor says so (`returnsStruct` is unexported and
  means "some return is a literal").
- **81** whose callee is a built-in the catalogue lacks a return for (`xmlElemNew`).

### 3. Loop variables

ContentBox's views loop over what a handler left in `prc`. The chain is: handler sets
`prc.roles = roleService.list( sortOrder = "role", asQuery = false )`, the service is
an injected cborm `VirtualEntityService` bound to an entity, `list()` with
`asQuery = false` is an array of that entity, and the view's `<cfloop array="#prc.roles#"
index="role">` is one. Every link existed except `list( asQuery = false )`, which only
`getAll()` had; adding it removed 48. What remains of that class is the same chain with
a link missing: a finder with no entity binding, an array built in a loop, a struct of
arrays. `rawSettingsTable.cfm` (22) and `permissions/index.cfm` (17) are two of them.

### 4. Unknown functions are mostly true

`getStringName` (TestBox, 41 calls to a function that does not exist), Lucee's
`writeln` (33), `valueEquals` (a retired MXUnit tag), `verifyCsrfToken` in Galaxie's
`blog.cfc` (10, defined only in `ProxyController.cfc`). The few that are not: BoxLang's
`println`/`cache` and Lucee's MongoDB and AI functions, now listed.

### 5. What a dynamic value taught us

Eight of the largest removals came from one idea: a value that is `$any` stays `$any`
through a call on it, a property of it, an element of it, a conditional with it as a
branch, an argument every caller passes it, and a function that returns nothing the
source can type. The parse already did this when it could see the receiver; the
resolver and the report did not. Each of those is now consistent with the parse.

## What was done in this round, with the finding counts

Total 8,215 → 4,214. By mechanism (the changelog has the exact figures):

- Dynamic propagation (call on, property of, element of, conditional, argument by
  callers, property as receiver): about 1,150.
- Two report-level policies decided in `ClassifyReason`, so the resolver's own answer
  and `explain` keep the reason: an untyped hop (`svc.get().foo()`, about 1,700 with
  the next) and a member function on an unknown receiver when no component declares
  that name (`x.len()`).
- A variable assigned the result of an untyped function: 305.
- Java/native contracts and built-in return types (Lucee page context, config, query,
  `TimeZone`, `Duration`, XML DOM, `getCanonicalPath`…): about 330.
- Tag-syntax parity (`<cfset x = createObject( "java", … ).factory()>`, `<cfinvoke>`,
  `<cfinclude template="dir/#file#">` after a `<cfdirectory>` listing): about 60.
- A preset convention for Mura's beans: 221.
- cborm `list( asQuery = false )`/`findAllWhere`, struct/array field literals: about 75.

Flow-aware work (the dominating assignment, a use after a branch, the same in tags)
changed no finding count on these corpora and one test fixture each; it makes the
claims sounder, not fewer.

## What was tried and dropped

- Typing a variable from a function's declared native return type: unsound for
  `struct` (above), nothing gained for the rest.
- A `helpers` implicit base for ColdBox: removed 17 findings and introduced 2.
- `this.x` field assignments in the assignment table: nothing measurable.
- `execute()` in ColdBox's `BaseTestCase`: the function adds members to the event it
  returns (`producerAugmentsReturn` declines on purpose), so `e.getRenderedContent()`
  has no sound answer.

## Ranked next steps

| Step | Findings at stake | Risk |
|---|---|---|
| A ContentBox preset: parameter names (`site`, `author`, `page`, `comment`, `category`, `setup`) | ~250 | A claim by name; needs `dynamicIfMissing` and the `content` ambiguity decided |
| Read the specs: a parameter is what the tests pass it | untyped parameters with 0–1 callers, ~300 | Large; the tests are the only caller of many public methods |
| `attributes.x` from the callers of a custom tag, for Masa | 100 | The rule exists (`customTagAttribute`); Masa's tags pass `attributes.bean` through other tags |
| Callback parameters of cborm's criteria builder (`when`, `$or`) | ~60 | Needs the stub bodies the generator empties |
| A narrowing guard: `isObject( x ) ? x.m() : …` | ~50 native-string false findings in Galaxie and Lucee | Control-flow typing of conditions |
| The classifier's 366 unknown origins | unknown | Look first; they may be one more shape |

## How to reproduce

1. Build the binary at the commit under test and the one before.
2. For each corpus: `clif unresolved --json . > new.json`, then
   `clif unresolved-diff --show 5 --root . base.json new.json` (the `.clif.json` presets
   used for ColdBox, ContentBox, TestBox and FW/1 are the plain `frameworks` ones; Masa
   uses `scripts/corpus/masacms.json`).
3. Classify each finding from the line that assigns its variable (the scripts did this
   with regular expressions over the source, then `clif explain --root . <file> <line>
   <method>` on a sample of each class to confirm).
