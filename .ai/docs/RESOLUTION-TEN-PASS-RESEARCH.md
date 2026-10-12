# Ten research passes over the remaining resolution gaps

Further review: [Execution-context audit](RESOLUTION-CONTEXT-AUDIT.md) separates
definition lookup from dynamic acceptance in the admin scenario, traces its
factory owner, surveys eleven spanless save locations, and examines record
merge/alias boundaries. No subsequent implementation is included.

10 October 2026. Research only, following checkpoint 30. No production parser,
resolver, preset, test or corpus configuration was edited in this batch. The
default remains **10,136 diagnostic entries across 8,151 files**. Experiments
used Go overlays and binaries under `target/resolution/research-ten-pass/`.
Those ignored artifacts are local evidence, not shipped behavior.

## 1. Refresh the population before selecting fixes

Rebuilt the source-context inventory from all seven checkpoint-30 JSON reports.
It contains 4,233 groups of file/caller/receiver/reason, not 4,233 independent
defects. Inputs and source hashes are recorded in `inventory-summary.json` and
`source-evidence.json` in the artifact directory.

| Final diagnostic symptom | Entries |
| --- | ---: |
| Unknown receiver | 7,703 |
| Unknown component return | 968 |
| Unqualified call | 702 |
| Missing method | 438 |
| Missing component | 279 |
| Missing-base summaries | 46 |
| Total | 10,136 |

Receiver/return symptoms account for 85.5%. This does not attribute their causes.
The 46 base summaries represent 1,132 unchecked calls; expanding only these gives
11,222 unresolved/unchecked occurrences. Exact duplicate source groups account
for just 84 entries in 17 groups, so deduplication cannot explain the bulk.

Decision: prioritize producer and execution-context proofs. A further 30% net
reduction requires at least 3,041 entries; the work below does not establish that
much safely achievable coverage.

## 2. Follow the changeset veto to its next parser path

The previous chain-span fix lets argument inference pass the zero-argument save
at `admin/core/controllers/carch.cfc:567`. A logging-only overlay now confirms
the next veto at `core/mura/category/categoryManager.cfc:274`:

```cfml
for(var obj in addObjects){
    obj.save();
}
```

This call has `HasSpan=false`, zero byte bounds, and fails the omission check.
Instrumentation identifies the **top-level named-call branch in scriptParser.parse**,
not the chain-continuation branch fixed in checkpoint 30. The script region is
seeded with a tag function's scope but still enters parse's top-level dispatch.
That branch calls `skipParensResolving` before constructing/emitting the call.
The scanner is at the semicolon when `addCall` tries to capture the origin:
token kind 12, region offset 77, base offset 11,827. The opening parenthesis was
at region offset 75. See `changeset-emitter-trace.log` and the instrumented
copies of `script_parser.go` and `call_spans.go`.

The source mechanism and live emitting path agree. A future fix should capture
identity before consumption in this separate branch, preserve function ownership,
and test a script island inside a tag function, repeated methods, multiline calls,
and nested arguments. Audit other emit-after-consumption paths as well; do not
assume all missing spans originate here.

Impact boundary: 65 downstream `changesetManager.save/arguments.bean` entries
remain blocked. Correcting this veto is necessary, but later unknown owners or
supplied values can still veto inference. **No 65-entry gain is established.**

## 3. Recover the embedded admin execution context

The refreshed Mura report has exactly 253 unqualified `rbKey` and 37 `buildURL`
entries. `admin/Application.cfc:44` extends the embedded `framework`, declares
rbKey at :533, and its framework renders views/layouts with computed includes
at `admin/framework.cfc:1512,1538`. This relationship was researched and
implemented for the explicit FW/1 preset earlier; it is absent from the current
automatic Mura selection.

`AssumedFrameworks` recognizes Mura's root mapping and returns `mura`; it does
not discover the nested admin owner from that root. The existing ApplicationBase
rule can supply the governing Application only after a view gets that implicit
base. `resolve.applicationBase` verifies the Application chain has both view and
buildURL before using it. Merely finding an Application.cfc is insufficient.

Decision: investigate an owner-scoped embedded framework context, not an
unconditional project-wide additional preset. The next pass tests why scope and
policy selection matter.

## 4. Test preset interaction, then isolate the useful part

Research overlays apply variants to the **same 839 Mura files**. Each JSON diff
uses multiplicities and file/line/function/reason/caller/receiver/unchecked keys.
The off-switch control matches checkpoint 30 with zero removals/additions.

| Research scenario | Mura entries | Removed / added | Resolved records |
| --- | ---: | ---: | ---: |
| Current control | 3,723 | 0 / 0 | 30,015 |
| Entire additional FW/1 preset | 6,320 | 518 / 3,115 | 27,305 |
| Only additional FW/1 resolver rules | 6,693 | 226 / 3,196 | 26,932 |
| Only FW/1 implicit view/layout base, all matching paths | 3,345 | 380 / 2 | 30,423 |
| Same implicit base restricted to paths under admin | **3,363** | **362 / 2** | **30,376** |

The resolver-only experiment reproduces substantial interference without adding
FW/1 stubs or changing implicit bases. A paired explanation at
`admin/Application.cfc:474` shows the consequence: the control follows the
startup assignment to the real settingsManager and its settingsBean return;
the additional rules fall back to a variable-name preset and cannot establish
getSite's return. This localizes the failing layer, but is not a complete
attribution of all 3,196 additions to one individual rule.

Both full-preset and resolver-only scenarios report 33,625 resolved-plus-unresolved
records, 113 fewer than the control despite the same file count. Their removal
counts cannot all be credited as successful resolutions. The interference needs
an extraction/ownership audit as well as a diagnostic audit.

The admin-only scenario removes all 290 helper findings plus other admin receiver
findings. Its two additions are **reason replacements at existing call sites**:
`dsp_displaycontent.cfm` report lines 44 and 71 move to unknown getParent and
getDisplayInterval returns. All 362 removals are under admin. Net reduction is
360 (3.6% of the overall default); a hypothetical combined total is 9,776.
This is a measured conditional scenario, not an implemented result or a claim
that all removed calls are semantically correct at runtime.

Coverage review: the admin-only scenario has 33,739 resolved-plus-unresolved
records versus 33,738 in the control. A verbose-label multiset audit finds one
additional resolved `loadcomments.cfm:280: local.item.getName` and no missing
occurrence labels. Verbose labels omit full paths, so equal basenames can collide;
the diagnostic diff retains full paths. The broader base-only scenario adds 30
records, another reason to prefer the narrower experiment for further review.

Future boundary: use the governing application and its proven embedded rendering
chain, preserve explicit config precedence and other projects' contexts, avoid
generic FW/1 getBean rules in Mura's factory domain, and review fallback stubs and
newly exposed calls. `/admin/` is a probe boundary, not a sufficient detection rule
for a shipped implementation. This is a refinement of earlier explicit-preset
research, not a newly discovered framework behavior.

## 5. Trace template arguments through the renderer

Two exact cohorts total 128 entries: `arguments.item` in collection's
`dsp_content_list.cfm` (68) and `dsp_meta_list.cfm` (60). These are template bodies,
not declared functions with ordinary caller arguments.

The list layout calls `variables.$.dspObject_include` with a literal theFile and
an iterator. contentRenderer's dspObject_Include accepts theFile, computes a
site-selected filepath, and includes it inside its function (:1547 onward;
lookup/include at :1615–1620). The list template writes
`arguments.item=arguments.iterator.next()` (:111–113), then forwards item through
additional renderer calls. Ordinary call-argument inference cannot reconstruct
that scope just by scanning for a same-named function.

settingsBean.lookupDisplayObjectFilePath (:2015–2047) checks alternate themes,
a cached selection and an ordered displayObjectLookUpArray, with file-existence
and legal-path conditions. A literal theFile therefore identifies a logical
template, not necessarily this core file. Registered custom directories can
shadow it.

Proposed proof: logical-template selection plus retained callee argument scope,
iterator element origin, and nested rendering propagation. A core-template
assumption needs documented override/unknown boundaries for themes, custom
directories, cached choices and unrelated renderers. **128 is an investigation
cohort; it is not a verified gain.**

## 6. Separate record-field origin from dynamic module dispatch

ColdBox has 25 `results.ehBean` entries in Controller._runEvent and 26
`results.config` entries in ModuleService.loadModuleConfiguration: 51 total.
The syntax is similar but the proof obligations differ.

Controller writes results.ehBean from handlerService.getHandlerBean(event)
followed by setIsPrivate (:820–823). HandlerService.getHandlerBean (:227 onward)
returns either a cached value or an explicitly constructed EventHandlerBean and
stores that bean into the cache on several branches. A useful rule must carry
return identity through the cache, fluent setter and record member assignment;
matching the final field name is insufficient. Cache replacement/unknown writers
remain relevant.

ModuleService writes results.config from a dynamically composed ModuleConfig
factory ID (:1146). Later methods such as injectPropertyMixin operate on that
object. Concrete module selection and installed mixins require their own facts.
Do not turn every `.config` member into one nominal component or count the two
cohorts as one homogeneous record fix. Existing memberReceiver already tries
recordReceiver, builderMember and assignedFromCall; these paths should be traced
before introducing a competing lookup.

## 7. Identify where selector and negative-guard narrowing are needed

Mura apiUtility.getLinks has 42 unknown `entity` entries. The source distinguishes
content, category, site and feed through getEntityName values (:3375 onward).
A content branch uses getContentID/getFilename, while the feed branch uses
getFeedID. getLinks receives an entity forwarded by the serializer (:2184), not
a uniform contentBean. A shared bean API cannot justify entity-specific methods;
selector-to-origin correlation and extensible entity registration matter.

Separately, Lucee has 140 exact unknown `field` entries across ten contexts.
The logging consumer loops over driver.getCustomFields and continues when
isInstanceOf(field,"Group") succeeds (:264–272). Later methods are Field-only.
Appender's factory helpers explicitly construct Field and Group, and drivers
populate their field arrays; ResourceAppender also duplicates and modifies the
array. The existing guardedReceiver establishes positive exact builtin guards,
but explicitly excludes negative/alternative branches.

To infer Field after the continue, establish a closed incoming Field|Group set,
the correct relative component identity, and the terminating branch. Excluding
Group from an arbitrary unknown value does not prove Field. Reassignment and
extension drivers can invalidate a closed-set assumption. Both cohorts need
value alternatives and branch correlation; neither is resolved by a method-name
catalog. **42 and 140 are measured populations, not additive promised gains.**

## 8. Review native API boundaries and possible false positives

The 46 ScheduledTask `now` entries come from getJavaNow (:1491), through
DateTimeHelper.toLocalDateTime (:92–100), into a Java time chain. They need a
native object contract rather than a CFC mapping. The source identifies the
producer; no native API catalog or runtime method check was added.

Lucee LDEV3734 has 25 missing getColumn diagnostics; its own test comment explains
query-column get calls (:43), and queryExecute supplies the query. These are
engine-specific/native API candidates, not evidence for inventing a user CFC
method. Another large unqualified group, LDEV6377_regression.cfm, calls writeln
39 times. No local declaration was found in that file. Its actual execution
environment and availability of that helper remain unverified; do not suppress
it merely because it appears in an engine test.

The local corpus does not include the expected QueryImpl Java implementation at
the checked path. Further native-catalog work needs the matching engine source
or official versioned API evidence before declaring these calls valid.

## 9. Revisit missing-base coverage without substituting test doubles

Wheels' missing modules.BaseModule alone summarizes 763 unchecked calls. The
test-only source explicitly labels itself a LuCLI test double, and its test
Application supplies a modules mapping. It does not establish the production
base for oneModule. Connecting an arbitrary stub could shrink a summary while
leaving substantial methods unchecked or introducing wrong ownership.

Required next evidence: host module/version contract, actual inherited methods,
test-vs-production execution roots, and expanded per-call before/after coverage.
The other base summaries still represent 369 unchecked calls. This category can
materially improve coverage even when its headline entry count is small; do not
rank it exclusively by the 46 compressed entries.

## 10. Review overlap and choose the next proof sequence

| Next investigation | Evidence reached | What is still missing |
| --- | --- | --- |
| Embedded admin view context | Controlled net 360 reduction; broad preset regresses | Owner-aware detection, config precedence, per-call validity and extraction review |
| Remaining lexical origins | Live parser emission path confirmed; 65 downstream entries | Focused regression and complete competing-caller trace after correction |
| Renderer-to-template relationships | 128 entries and selection/scope path traced | Override-aware template selection and iterator element proof |
| Returned records/caches | Two distinct cohorts, 51 entries | Full return/writer trace; fluent and mixin contracts |
| Selector/guard alternatives | 42 Mura entity and 140 Lucee field entries | Closed alternatives, origin-preserving branch narrowing, extensions |
| Native and missing-host contracts | 46 native-time entries; 763 calls behind one base | Versioned definitions and execution context |

These rows are not an implementation estimate. Several require earlier producer
facts and can overlap with broader return/caller work. In particular, the 104
feed argument entries retain the earlier bean-versus-array counterexample; do
not skip that caller to create consensus. Wheels adapter publication/reload
research remains open and should be reused rather than reclassified as a proven
gain.

The strongest new measured opportunity is **scoped execution context**. The
largest general engineering opportunity remains preserving value origin through
returns, fields, collections and branches. Before implementation, turn each into
a failure-before regression, enumerate competing values/writers, and measure the
same corpus entries and coverage. A first fix exposing a second blocker is useful
evidence, but not grounds to credit the whole downstream cohort.

Validation: repository preflight passes with approved network permissions;
sandbox DNS failed. All research binaries compile. The no-scenario Mura report
matches checkpoint 30 per entry. Four configuration scenarios have independent
JSON reports, stderr coverage counts and multiset diffs; parser tracing changes
only logging in ignored overlays. The production arg_callers.go SHA-256 still
matches the earlier pre-probe hash. No production tests or runtime execution were
run for this documentation-only batch, and no new CI pass is claimed.


## Implementation follow-up: script-island call identity

The named-call branch described in pass 2 now captures the call span before
`skipParensResolving` consumes its arguments. A tag function script-island
regression covers repeated, multiline, nested and loop calls with shifted source
bytes and retained function ownership; it fails before the fix and passes after.

On unchanged Mura scan coverage, the current strict comparison removes **106
sites**, including all **65** identified `changesetManager.save/arguments.bean`
findings, 25 contentDAO argument findings and six trashManager restored-value
findings. The broader native/reporting batch changes diagnostic reasons too;
see `target/resolution/native-contracts/comparisons/masa/` for full multiset diffs.
Passes 3–7 and 9 retain the separate obligations in the
[completion ledger](RESOLUTION-COMPLETION-AUDIT.md). Their research cohorts are
not implementation gains.
