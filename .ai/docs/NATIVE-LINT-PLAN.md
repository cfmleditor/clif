# Native Lint Plan: Replacing Embedded CFLint with `clif`

This plan outlines how to replace the internal CFLint subprocess (`internal/cflint`) with native Go lint rules inside `clif`, grounded in a real production `.cflintrc` (41 rules + 2 `QueryParamChecker` parameters).

## Goals & Payoffs

1. **Drop the ~90 MB embedded CFLint binary** from release builds and eliminate first-use downloads and subprocess startup overhead.
2. **Live, on-type linting** in the LSP (debounced like `structureDiagnostics`) rather than save-only subprocess runs, and 10–50× faster CI/pre-commit scans.
3. **Zero config migration for existing repos:** keep full compatibility with `.cflintrc` (`includes`, `excludes`, `parameters`, `inheritParent`, `excludeFiles`), `@CFLintIgnore`, `// cflint ignore:`, baselines, and `report`/`text`/`json`/`sarif` outputs.
4. **Corpus-verified parity:** validate rule-by-rule against CFLint on real repositories using a shadow/compare mode before switching the default engine.

---

## Inventory of the 41 Production Rules

### Tier A — Table-Driven Tag, Call, Attribute & File Rules (26 rules, low complexity)

All 26 can be implemented in one single-pass CST/token walk with near-zero false-positive risk:

| Group | Count | Rules | Implementation |
| --- | ---: | --- | --- |
| Banned CFML tags | 14 | `AVOID_USING_CFDUMP_TAG`, `AVOID_USING_CFABORT_TAG`, `AVOID_USING_CFEXECUTE_TAG`, `AVOID_USING_CFINSERT_TAG`, `AVOID_USING_CFUPDATE_TAG`, `AVOID_USING_CFFORM_TAG`, `AVOID_USING_CFINPUT_TAG`, `AVOID_USING_CFINVOKE_TAG`, `AVOID_USING_CFFILE_TAG`, `AVOID_USING_CFDIRECTORY_TAG`, `AVOID_USING_CFCOOKIE_TAG`, `AVOID_USING_CFHTTP_TAG`, `AVOID_USING_CFMODULE_TAG`, `AVOID_USING_CFINCLUDE_TAG` | Single lookup table over tag open nodes (excluding comments/strings). |
| Banned script calls / statements | 6 | `AVOID_USING_WRITEDUMP`, `AVOID_USING_ABORT`, `AVOID_USING_ISDEBUGMODE`, `AVOID_USING_CREATEOBJECT`, `AVOID_USING_EVALUATE`, `AVOID_USING_ISDEFINED` | Single lookup table over script call nodes and `abort` statements. |
| Tag attribute & file checks | 6 | `AVOID_USING_DEBUG_ATTR`, `AVOID_USING_CFSETTING_DEBUG`, `ADDTOKEN_ATTR`, `FILE_SHOULD_START_WITH_LOWERCASE`, `AVOID_EMPTY_FILES`, `NEVER_USE_QUERY_IN_CFM` | Inspect tag attributes (`debug`, `showdebugoutput`, `addtoken`), filename casing on `.cfm`, non-whitespace/comment emptiness, and `<cfquery>`/`queryExecute` in `.cfm`. |

### Tier B — Structural CST Rules (9 rules, low–medium complexity)

| Rule | Implementation |
| --- | --- |
| `PARSE_ERROR` | Already implemented via `tree-sitter-cfml` (`clif scan`). |
| `OUTPUT_ATTR` | Check `<cfcomponent>`/`component` and `<cffunction>`/`function` for explicit `output` attribute. |
| `NESTED_CFOUTPUT` | Track `<cfoutput>` nesting depth during CST walk; flag nested `<cfoutput>` when neither outer nor inner carries `group`. |
| `ARG_TYPE_MISSING` | Check `<cfargument>` and script parameter nodes for missing `type`. |
| `FUNCTION_ACCESS_REMOTE` | Flag `<cffunction access="remote">` and `remote function`. |
| `NO_DEFAULT_INSIDE_SWITCH` | Check `<cfswitch>` for `<cfdefaultcase>` child and script `switch` for `default:` clause. |
| `COMPARE_INSTEAD_OF_ASSIGN` | Flag bare assignment `=` inside `if`/`elseif`/`while` conditions and standalone `==`/`!=` comparison statements. |
| `SQL_SELECT_STAR` | Scan `<cfquery>` body text (and `queryExecute` SQL string literals) for `SELECT *` outside SQL comments/strings. |
| `GLOBAL_VAR` | Flag reads/writes to shared scopes (`application`, `session`, `server`, `request`, `form`, `url`, `cookie`, `cgi`, `client`) inside CFCs/functions. |

### Tier C — Function Scope & Variable Tracking Rules (4 rules, medium–high complexity)

These are the highest-value rules and require careful CFML scope semantics (already partially modeled in `internal/parser/rename_scope.go` and `internal/parser/script_assign_refs.go`):

| Rule | Semantics & Edge Cases |
| --- | --- |
| `MISSING_VAR` | Unscoped variable assignment inside a function that is not `var`-declared, not an argument, not explicitly scoped (`local.`, `variables.`, `this.`, `arguments.`, etc.), and not bound by a tag attribute (`<cfloop index/item>`, `<cfquery name/result>`, `<cfcatch name>`, `<cfhttp result>`, `<cffile variable>`, etc.). |
| `IMPLICIT_SCOPE` | Unscoped variable read or write inside a function/CFC (outside explicit scope prefixes and `var` declarations). |
| `ARG_VAR_CONFLICT` | Local `var x` declaration colliding with a parameter name `x` in the same function. |
| `ARG_VAR_MIXED` | Same parameter accessed both via `arguments.x` and unscoped `x` in the same function. |

### Tier D — QueryParam Rules with `.cflintrc` Parameters (2 rules, medium–high complexity)

| Rule | Implementation |
| --- | --- |
| `CFQUERYPARAM_REQ` | Inside `<cfquery>` bodies, find `#...#` interpolations not inside `<cfqueryparam>` or SQL comments. |
| `QUERYPARAM_REQ` | Inside `queryExecute(...)` / `new Query().setSQL(...)` string arguments, find `#...#` interpolations. |
| `.cflintrc` parameters | Support `QueryParamChecker.allowVariableExpression` (regex matched against the `#...#` expression) and `QueryParamChecker.allowLineExpression` (regex template with `${variable}` substituted via `regexp.QuoteMeta`, matched against the full source line). Both production patterns are standard RE2-compatible with Go's `regexp` package. |

---

## Architecture

### 1. Backend-Agnostic `internal/lint` Package

Move shared lint infrastructure out of `internal/cflint` into `internal/lint`:

- `Linter` interface:
  - `Lint(ctx context.Context, files []string) (*Run, error)`
  - `LintContent(ctx context.Context, sources []Source) (*Run, error)`
  - `Scan(ctx context.Context, filePath string) ([]protocol.Diagnostic, error)`
- Shared types & utilities: `Run`, `Issue`, `RunIssue`, `Source`, severity ranking, `.cflintrc` hierarchy resolver (`includes`, `excludes`, `parameters`, `inheritParent`, `excludeFiles`), inline suppression matcher (`@CFLintIgnore`, `// cflint ignore:`), and `report`/`text`/`json`/`sarif` formatters.
- Two backends:
  - `internal/cflint` — existing CFLint subprocess runner.
  - `internal/lint/native` — pure-Go rule engine over `tree-sitter-cfml` CST + `internal/parser`.

### 2. Config Switch (`linting.engine`)

Add `linting.engine` to `.clif.json` (and `--engine` on `clif lint`):

- `"cflint"` — legacy CFLint subprocess (initial default during migration).
- `"native"` — pure-Go `clif` linter.
- `"hybrid"` — runs native rules for every rule `clif` implements and delegates only remaining unimplemented `.cflintrc` rules to CFLint (zero subprocess if all enabled rules are native).

### 3. Corpus Parity Harness (`clif lint --compare`)

Follow the same discipline used in `.ai/docs/RESOLUTION-GAPS.md`:
- Run both `native` and `cflint` over real repositories.
- Diff findings by `(file, line, rule)` and classify every difference:
  - **Missing finding (FN):** native missed a real issue -> fix native rule.
  - **Extra finding (FP):** native flagged a non-issue -> fix native rule.
  - **Known CFLint bug:** CFLint itself misparsed valid CFML (e.g. script closures, member functions, modern Lucee syntax) -> document and test as an intentional improvement.

---

## Phased Delivery

1. **Phase 1 — Engine plumbing & `.cflintrc` rule config (~2 days)**
   - Create `internal/lint` interface, move shared formatters/suppressions/report writers, implement full `.cflintrc` inheritance (`includes`/`excludes`/`parameters`/`excludeFiles`), and wire `linting.engine` (`cflint` | `native` | `hybrid`).
2. **Phase 2 — Tier A + Tier B rules (35 of 41 rules, ~3–4 days)**
   - Implement all 26 table-driven tag/call/attribute/file rules and 9 CST structural rules with unit tests + parity check against CFLint.
3. **Phase 3 — Tier C (scope rules) + Tier D (QueryParam rules) (6 of 41 rules, ~4–6 days)**
   - Implement `MISSING_VAR`, `IMPLICIT_SCOPE`, `ARG_VAR_CONFLICT`, `ARG_VAR_MIXED`, `CFQUERYPARAM_REQ`, and `QUERYPARAM_REQ` (including `allowVariableExpression` and `allowLineExpression`).
   - Run corpus parity comparison on production codebases and tune edge cases until precision/recall matches or exceeds CFLint.
4. **Phase 4 — Cutover & live linting (~2 days)**
   - Flip `linting.engine` default to `"native"` (with `"hybrid"` / `"cflint"` opt-in for external CFLint users).
   - Enable debounced live on-type linting in the LSP server (no subprocess on save).
   - Remove the ~90 MB embedded CFLint binary from default release builds.
5. **Phase 5 — Native-only workspace rules (post-cutover)**
   - Rules CFLint could never implement: missing interface methods, invalid `extends`/`implements` targets, unused private methods, and unresolved call diagnostics.

---

## Open Questions to Confirm Before Phase 1

1. **Severity mapping:** Should native rules use CFLint's exact default severity per rule (`ERROR`, `WARNING`, `INFO`, etc.) so `--min-severity` and baselines match 1:1? *(Recommended: yes.)*
2. **Per-file vs. per-variable deduplication:** CFLint notoriously reports `MISSING_VAR` and `IMPLICIT_SCOPE` only once per variable name per run (not per occurrence). Should native mode match CFLint's per-function/per-file deduplication or report every occurrence (or gate via a flag)? *(Recommended: once per variable per function, which fixes CFLint's cross-file suppression bug while staying noise-free.)*
3. **Non-`.cflintrc` native config:** Keep `.cflintrc` as the primary rule config, or also allow declaring rules directly inside `.clif.json` under `linting.rules`? *(Recommended: support `.cflintrc` first for drop-in parity, add `.clif.json` support in Phase 4.)*
