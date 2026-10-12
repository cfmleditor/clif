# String producers and native reassignment review — 2026-10-11

This batch follows the builtin catalogue audit: 12,386 strict findings,
509 liberal findings, 6,915 runtime assumptions and 1,899 speculative receiver
connections. Existing corpus configuration and mode flags are unchanged.
Artifacts live in `target/resolution/string-producer-followup/`.

## Implemented relationships

Recognition of a global builtin does not establish its result's interface.
Added 37 fixed string results, each checked against both the committed generated
catalogue and the [Lucee function library](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/resource/fld/core-base.fld):

`left`, `right`, `mid`, `trim`, `ltrim`, `rtrim`, `lcase`, `ucase`, `reverse`, `replace`, `replacenocase`, `rereplace`, `rereplacenocase`, `listfirst`, `listlast`, `listrest`, `listgetat`, `listappend`, `listprepend`, `listchangedelims`, `listdeleteat`, `listinsertat`, `listsetat`, `listsort`, `listqualify`, `listremoveduplicates`, `getdirectoryfrompath`, `getfilefrompath`, `expandpath`, `gettempdirectory`, `serializejson`, `tostring`, `hash`, `urlencode`, `urldecode`, `charsetencode`, `repeatstring`.

These return a native string contract. Qualified service methods and source
functions retain their own identities. Callable parameters, earlier locals and
callback parameters cannot borrow a builtin spelling. Binary producers such as
`charsetDecode` and `toBinary`, dynamic JSON decoding, and extension functions
absent from the checked core library are excluded. The catalogue is not an
engine/version compatibility check; this follows the existing bounded native
interface convention rather than promising identical interop across engines.

Added boolean string member results for `hasPrefix` and `hasSuffix`. Lucee's
hidden `StringHasPrefix` and `StringHasSuffix` functions explicitly register
these member names in the function library. Their implementations are
[StringStartsWith](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/functions/string/StringStartsWith.java)
and [StringEndsWith](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/functions/string/StringEndsWith.java).
This closes the two new missing-method findings exposed by the initial scan.

Added the fixed date result of [DateAdd](https://docs.lucee.org/reference/functions/dateadd.html).
A date variable reassigned from DateAdd must not lose its interface merely because
the previous type was known only from CreateDate. Its FLD result is datetime.

Native function-scoped references are now withheld when a later separate
statement-line assignment intervenes. Previously `var x=right(...);` on one
line followed by `x=unknown();` on another could retain a string interface.
The same correction applies to earlier array/date/Java-native references.
It changes stale evidence into an unknown receiver, not an invented new type.

The first reassignment probe exposed named-argument false matches in the shared
assignment recovery regex. `buffer=buffer` inside a call names an argument;
it does not reassign the caller's variable. Parentheses/bracket context and
expression-key boundaries now exclude these matches, including simple struct
literal keys. Recovery continues backwards past named arguments so they cannot
hide a real earlier reassignment.

## Evidence and remaining scope gaps

`flow-probe.log` tested unknown reassignment, closed branches, conditional writes,
sibling callbacks and same-line future assignments. Sibling callback locals were
already separated correctly. The new guard handles separate statement-line
reassignments. It does not prove branch dominance, every same-line write, or
arbitrary expression-key context; these remain explicit limits.

The report does not call this a complete native lifetime model. A producer inside
a branch may still type calls after the branch, and same-line assignments can
still lack ordering evidence. A fuller implementation needs byte-identified
assignments, lexical owner identity, branch completion, and invalidation shared
by parser refs and resolver fallbacks. Local alias/capture and compound writes
also need a common flow model. The earlier InternalRequest field lifetime proof
is narrower and cannot safely be generalized by matching a variable name alone.

Two Mura getAllValues removals were traced separately. They resolve through an
existing `$any` ComponentRef and dynamic acceptance, not through a verified CFC
method target. They count as report removals but are not claimed as strict method
identity proofs. The remaining string/CGI and wrapper relationships likewise
must be separated from dynamic acceptance when forecasting larger gains.

## Validation and measurement

The string producer, hidden-member, native reassignment, named-argument boundary
and DateAdd regressions fail with their respective fixes removed. Counterfactual
logs are retained. Focused tests cover source-function priority, argument/local/
callback shadowing, binary boundaries, wrong-family methods and valid native
reassignments. The initial producer scan and the intermediate reassignment scan
are retained as `probe-checked/` and `reassignment-probe-checked/`.
Final measurement uses the rebuilt binary in `checked/`.

## Next investigations from remaining findings

The final strict finding categories are 9,799 variable receivers, 1,200 return
types, 922 missing methods and 434 object findings. Variable receivers make up
79.3% of the 12,355 entries. The builtin spelling list is not the main bottleneck.

Within the variable category, leading receiver roots include `arguments` (1,580),
`variables` (988), `local` (529), `rc` (211), `field` (205), `content` (176),
`item` (146), `prc` (136), and `application` (131). These are investigation buckets,
not predicted recoveries: the same spelling can belong to different source
owners and runtime relationships. The machine-readable category recount is
`remaining-categories.json` in the artifact directory.

Next work should prioritize scoped incoming argument/caller evidence, shared
component field factories, and callback element relationships, then connect
wrapper return shapes to those receivers. Before expanding lifetime inference,
assignments need byte identities and lexical owners shared by parsing and
resolution; branch completion and all surviving writes must agree. API additions
remain useful for the separate 922-method bucket, but cannot resolve most of the
9,799 unknown receiver entries on their own.

## Final corpus comparison

| Project | Strict before | Strict after | Liberal before | Liberal after |
|---|---:|---:|---:|---:|
| wheels | 2,565 | 2,556 | 143 | 143 |
| masa | 3,686 | 3,682 | 21 | 21 |
| contentbox | 1,746 | 1,743 | 44 | 44 |
| coldbox | 1,628 | 1,627 | 93 | 93 |
| lucee | 2,007 | 2,000 | 196 | 196 |
| testbox | 395 | 389 | 7 | 7 |
| fw1 | 359 | 358 | 5 | 5 |
| **Total** | **12,386** | **12,355** | **509** | **509** |

Strict findings: 31 removed, zero added. Two removals are the reviewed Mura
dynamic cases described above. Liberal findings: unchanged per entry. Runtime
assumptions drop from 6,915 to 6,905 (ten removed, no additions). Speculative
receiver connections drop from 1,899 to 1,884 (16 removed, one added).
The added connection guesses string for Lucee `_testRunner`'s `st.template`;
it is recorded as optional evidence, not a proven field interface.

Coverage remains 8,151 indexed files and 189,154 CLI counting records, with
46 unchanged base summaries representing 1,133 unchecked calls. The comparison
asserts unchanged per-project counts and unchanged base entries, rather than
comparing only the headline totals.

Full validation passed: normal and `cflint_embed` builds and vet, full
`go test -short ./...`, full `go test -short -race ./...`, `make lint`,
`bash scripts/check-go-format.sh` and `git diff --check`. Full tests ran with
approved local socket access after corpus scans completed. Logs are retained
in the artifact directory. No published PR/CI result is claimed.
