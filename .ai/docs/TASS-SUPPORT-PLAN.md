# TASS support plan

What cfmleditor-lsp would need to know about the TASS code base to give accurate lint
results, safe quick-fixes and useful navigation there. It comes from the TASS-14904 / IMS-76
lint clean-up across prs, kiosk, parentlounge, studentcafe, tassims and tassweb (October
2026), where every fix was made by hand or by one-off scripts. Each item says how it showed
up, so it can be re-checked against the repositories rather than taken on trust.

Items are grouped by area, and each area ends with what to build. The last section orders
the work.

## 1. CFLint integration

### 1.1 `cflint --write` writes to the nearest config, not the scanned project

`cfmleditor-lsp cflint --write ~/tassdev/prs` found `~/tassdev/.cfmleditor.json` (prs has no
config of its own) and replaced `~/tassdev/.cfmleditor-cflint.txt`, the workspace report,
instead of `prs/.cfmleditor-cflint.txt`. Scanning without `--write` and filtering stdout
works but loses the report header.

**Build:** an `--out <file>` flag, or a per-project mode that writes
`<project>/.cfmleditor-cflint.txt` with the header and repo-relative paths whichever config
was used.

### 1.2 The project scan and a per-file scan disagree

CFLint reports an `IMPLICIT_SCOPE` or `MISSING_VAR` name once per run, so a project scan shows
the first occurrence of a name and hides the rest. Fixing that occurrence exposes the next on
the following scan. In tassweb this hid five `MISSING_VAR` findings (`dsnInfo`, `i`) and a
dozen implicit-scope reads until each file was linted on its own. The editor lints on save,
per file, so developers see more than the committed report says.

**Build:** generate the report from per-file results (or dedupe per file rather than per run)
so the report matches what the editor shows.

### 1.3 Reported lines drift inside multi-line tags

For a multi-line `<cfquery>`, CFLint's line for `CFQUERYPARAM_REQ` is offset from the line
holding the `#var#`. The clean-up had to search ±25 lines for the variable text to find it.
An early script walked back to the nearest tag instead and put 31 ignores above `<cfquery>`
in prs, silencing whole queries.

**Build:** map each CFLint finding to the expression it names (the message carries the
variable text) before showing it or offering a fix.

### 1.4 Where an ignore comment has to go

Established by testing each placement against the shipped CFLint (cfmleditor-lsp bundled
cflint v0.3.3):

| Context | Works | Does not work |
|---|---|---|
| Tag code | `<!--- @CFLintIgnore RULE --->` on the line above the tag | |
| Inside `<cfquery>` | the same comment directly above the SQL line holding `#var#`; it covers that line only | above `<cfquery>`, which covers the whole query |
| cfscript | `//cflint ignore:RULE` at the end of the same line | `//` or `/* */` on the line above |
| Loop index read inside `<cfoutput>` (see 1.5) | above the `<cfoutput>` | above the `<cfloop>` or inside the block |

**Build:**
- a quick-fix that inserts the right form for the context;
- a diagnostic for misplaced ignores: above `<cfquery>`, above a line with no matching finding, or stale after the code moved;
- a warning when an ignore at `<cfoutput>` level would hide more than the one finding.

### 1.5 False positives worth fixing in CFLint (or filtering in the LSP)

Each was reproduced on a minimal file:

- **Loop index or query column inside `<cfoutput>`.** `<cfoutput><cfloop index="i" ...>#i#</cfloop></cfoutput>` is flagged on the `<cfoutput>` line. The same loop outside `<cfoutput>` is not flagged. Raised as cfmleditor/CFLint#76.
- **Literal text joined to `#expr#` inside `<cfoutput>`.** `'<cfoutput>purchasing/#a#&rowid=#b#</cfoutput>'` flags `purchasing` and `rowid`, and `id="<cfoutput>yeargroups-#i#</cfoutput>"` flags `yeargroups`. They look like arithmetic to the parser. Moving the literal text outside `<cfoutput>` clears it, which is the workaround used in kiosk and tassweb.
- **`<!--->` as a comment opener** (asas `juniorschool/service.cfc`). CFLint treats the commented block as code and reports names inside it.
- **A named argument called `case`.** `getTypicalParameterSetUp(abbrev="u", label="x", case="l")` is a `PARSE_ERROR` in `subservices/urlmaker/service.cfc`. `argumentCollection={"case"="l"}` parses.
- **The `ERROR` scope in error templates** (`packages/tass/core/error.cfm`) is reported as an implicit-scope variable.
- **CR-only line endings.** Several tassims model files (`binaryDAO.cfc`, `contactGateway.cfc` and others) use old Mac line endings, so any line-based tool sees one line.

## 2. How TASS code is structured

Knowledge the resolver and scope analysis need. None of it can be inferred from a single file.

### 2.1 Where page variables live

- **Kiosk views** are included by `index.cfm` at page level (`<cfinclude template="#REQUEST.context.getView('body')#">`), or by `Application.cfc` `renderTemplate()` inside a function with no `var` for the view's variables. Either way, unscoped page variables end up in VARIABLES.
- **tassweb webroot pages** are requested directly.
- **The rule:** a template's own variables and loop indexes are VARIABLES, and plain `#i#` is correct. Scope suggestions should not push `VARIABLES.i` onto loop indexes; the TASS team rejected that as noise.

### 2.2 Custom tags set variables on the caller

`cfimport` prefixes in use:
- `container:` is `tassweb/webroot/containers` and `tassweb/ui/web/containers`.
- `control:` is `tassweb/ui/web/controls`.
- `content:` is the page's own directory.
- `report:` is `tassreporting/containers`.

Tags return values to the calling page:
- **`returnvariable`:** for example `<container:lockaction ... returnvariable="recordLocked">`.
- **Direct `caller.X =` assignments:** `webroot/community/Community.cfm`, used as `<content:community>`, sets about 25 `*Perms` lists on the calling page.

**Build:** treat a custom-tag call as defining its `returnvariable` and every `caller.X` its source assigns, from the call onwards. That removes most of the community-page implicit-scope findings.

### 2.3 Query columns from service calls

Many loops run over queries returned by a method rather than a local `<cfquery>`:
- `REQUEST.kernel.getSandBox("tasscomfactory").getfamilyObj().getFamilyLinkInfo(...)`;
- `getkinObj().getKinRecords(...)`;
- `getmailObj().getMailDefs(cmpy, "mail_num,mail_desc", ...)`;
- `queryNew("template,templateId,parentId,...")` in `core/debug.cfm`.

**Build:** infer a returned query's columns from the callee's `<cfquery>` select list, `QueryNew` column list or a literal column-list argument. Then a bare read inside `<cfloop query>`/`<cfoutput query>` resolves to that query. Fall back to "unknown", not "implicit", when the source is `select *` or dynamic.

### 2.4 process/* components

- **`init()` pattern:** every `tassweb/process/**/*.cfc` `init()` copies its arguments with `<cfloop list="#structKeyList(arguments)#" index="key"><cfset variables[key] = arguments[key]></cfloop>`, then reads `variables.globals.*`.
- **Company chain:** processes call each other through `variables.linkmap.<cfc>.<method>(cmpy="#variables.cmpy#", ...)`, so a method's `cmpy` argument always equals the process's `variables.cmpy`.
- **Deliberate shared state:** `all_OK` in `PP1.cfc` is returned, while `m_par_count`/`m_stud_count` in `fupstat.cfc` are read across functions.

**Build:** recognise the copy loop so `variables.X` and `arguments.X` are known to be equal after it. That fixes the `ARG_VAR_MIXED` findings it causes without a rewrite.

### 2.5 subservices

`subsObj` is the shared context. Each service's `persist` is started by
`subslib.startPersist(subsObj, serviceName)`, and schema persists exist per engine
(`subservices/schema/{ids,mysql,postgresql,sqlserver}`). `variables.serviceName` (the component's
own name, sometimes with `/dbp` appended) and `arguments.serviceName` (the service being
started) are different values on purpose. `ARG_VAR_MIXED` should not suggest unifying them.

### 2.6 SQL fragment builders

These return SQL text, not values. Interpolating them is expected:
- `_objInit.formatNumericClause`, `formatDateClause`, `formatQuerySearchField`, `formatQuerySearchString`;
- `REQUEST.kernel.getObjInit().getSelectEncryptedCharData(...)` / `setVarBinaryCharData(...)`;
- `variables.prs.alterAddLeft()`, `alterAfter()`, `sqlFirst()`;
- `variables.subsObj.schema.supportsRecordID(...)`;
- the kiosk factory's `formatQuerySearchField`.

`.cflintrc` already lists some of them in `QueryParamChecker.allowVariableExpression`.

**Build:** a CFQUERYPARAM triage that splits findings into:
- **identifier:** table, `col_*` column, `ORDER BY`, alias or prefix;
- **fragment:** one of the builders above, or `preserveSingleQuotes` of a locally built clause;
- **value:** the real findings.

Each class gets its own quick-fix: ignore for the first two, convert to `cfqueryparam` for values. Flag an identifier that comes from request data (`packages/enrolments/persist.cfc` parses `field` from a `whereClause` string), because ignoring it would hide an injection.

### 2.7 JVM method-size limit

Large report and persist functions are close to the JVM 64 KB method / short branch-offset limit:
- `getReport` in the BGGS and St Margaret's custom report services;
- `packages/person/persist.cfc`.

Converting interpolations to `cfqueryparam` in `person/persist.cfc` broke compilation and had to be reverted (TASS-14904), and TASS-14914 split two `getReport` functions to compile at all.

**Build:** an approximate per-function size measure, a warning above a threshold, and a note on quick-fixes that add bytecode (`cfqueryparam`, `var` declarations) in functions near it.

## 3. Safe automated fixes

These rules were applied by script across the six repos. Each is precise enough to be a
quick-fix, and each failure mode below actually happened.

### 3.1 Make a scope explicit (`IMPLICIT_SCOPE`)

The order an unscoped read resolves in is:
1. LOCAL / ARGUMENTS;
2. the current query-loop row;
3. VARIABLES;
4. CGI, URL, FORM, COOKIE.

A fix may only add the scope the read resolves to today.

- **Query columns:** use the *innermost* loop's query at that line, checked against its columns. Applying one decision to every occurrence in a file put `getCourseSubjects.sub_desc` inside a `subjects` loop (kiosk `TIACourseDetails.cfm`) and `qryCurRels2` inside a `qryCurRels1` loop (tassweb `addperson2.cfm`).
- **URL before FORM:** a cfparam'd `FORM.x` does not make a bare `x` a FORM read. Use `URL.x` when the URL cfparam defaults from FORM, or keep the precedence explicitly (`structKeyExists(URL,"x") ? URL.x : FORM.x`) when a page is linked with the name in its query string.
- **Never inside string literals:** that includes a literal nested inside `#...#`. One rewrite turned `getValue('rowid')` into `getValue('VARIABLES.rowid')`.
- **Rewrite every read in the region, not just the reported one,** so a line never ends up half-scoped, as in `VARIABLES.class NEQ class`.

### 3.2 Declare a local (`MISSING_VAR`)

Safe when all of these hold for every function in the file that uses the name:
- the function writes the name before reading it;
- none of those functions calls another of them, so no value is handed over through VARIABLES;
- nothing outside functions uses the name;
- there are no `variables.X`, `this.X` or string-keyed references;
- there is no `cfthread`, `cfinclude` or `evaluate` in the function.

A function that writes a flag, calls a sibling, then reads the flag back is the case to refuse. Declare the name in every qualifying function at once. 72 tassweb locals and 125 prs locals met these rules.

### 3.3 Separate an argument from a local (`ARG_VAR_MIXED` / `ARG_VAR_CONFLICT`)

- **Bare reads of an argument with no local of the same name:** use `ARGUMENTS.x`. Unscoped writes already land in ARGUMENTS, so writes can be rewritten too.
- **A loop index or `LOCAL.x` that shadows an argument:** rename the local, after checking nothing reads the argument after the loop.
- **The `init()` copy pattern (2.4):** `arguments.x` after the copy.
- **Deliberately different values (2.5):** leave and ignore.
- **cfscript helpers:** read arguments inside `"f#NumberFormat(j,'00')#"` strings, so the `#...#` parts of string literals need rewriting too.

### 3.4 Debug leftovers

- **A dump of a caught error:** `cfdump` of `cfcatch` with `abort` inside `<cfcatch>` should be `<cfrethrow>`. If the catch already has rollback-then-rethrow, delete the leftover rather than add a second rethrow. One replacement left `<cfrethrow><cftransaction action="rollback"/><cfrethrow>` (`process/ap/P34debit.cfc`).
- **Debug branches** become `<cfthrow message=... detail=...>`.
- **Uncalled `dump()` helpers:** delete them. About 20 report action pages define one whose every call is commented out.
- **A `cfdump`/`cfabort` that guards real logic** must not be removed silently. `webroot/fundraising/event_saveAttendList.cfm` dumps and aborts before the actual save, so removing it would turn saving on.

### 3.5 Output attribute (`OUTPUT_ATTR`)

`output="false"` is safe only when the function and everything it calls write nothing.
`subservices/process/service.cfc` `runProcess` writes nothing itself but calls process
`main()` functions that may, and so does `payment_run_create.cfm` `publish` (it calls
`publishCH`). Build the call graph before suggesting it.

## 4. Whole-workspace checks

- **Uncalled functions across repos.** The search found `dumpCFQuery`, `subslib.trace`, the six `delete*` service methods that passed an undefined `args`, an unused `osbornediff_rev13.cfc`, and the `setEncryptedCharData` payroll helper in tassims.
- **Undefined component state.** `customer/persist.cfc` reads `_collectionsPath`, which only the Lucene components set. tassims `datasourceSettings.cfc` reads `_payrollSeed`, which nothing sets. `employeepd/persist.cfc` uses `arrEmpty`, which is never defined.
- **Values echoed into HTML/JS without encoding,** found while scoping:
  - `?word=` in both spellcheckers;
  - `URL.msg` in `create_ato_file.cfm`;
  - `modifiedStudCodeList` in a redirect.

  A taint check from URL/FORM to output would find the rest.

## 5. Compile checks

`cfcompile.sh` in the `tass_coldfusion` container is the only check that catches syntax and
bytecode-limit breakage. It needs the workspace layout to be useful:
- **Layout:** tassweb under `/tassweb`, plus `tassreporting`. Without it, 944 of 6,621 tassweb templates fail; with it, 222.
- **Before/after comparison:** an identical failure set on both sides is the signal. Per-file attribution has to follow the `<file>...` progress markers, because the summary line belongs to the run.

**Build:** an `lsp` command that compiles changed files against a baseline tree, and reports only differences.

## Order of work

1. Per-file report generation and `--out` (1.1, 1.2). Every other number depends on it.
2. Finding-to-expression mapping and ignore placement, with misplaced-ignore diagnostics (1.3, 1.4).
3. Scope resolution with custom-tag caller variables and service-query columns (2.2, 2.3), then the explicit-scope quick-fix (3.1).
4. `MISSING_VAR` and `ARG_VAR` quick-fixes with the cross-function rules (3.2, 3.3, 2.4).
5. CFQUERYPARAM triage (2.6) and the method-size warning (2.7).
6. Debug-leftover and output-attribute assists (3.4, 3.5), then the workspace checks (4) and the compile command (5).
