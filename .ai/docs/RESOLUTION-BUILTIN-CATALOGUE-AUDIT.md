# Builtin catalogue and return-contract audit — 2026-10-11

This follow-up starts at 12,468 strict findings and 512 liberal findings.
The two modes express different evidence levels. The liberal mode uses ranked
connections, native guesses and open-world assumptions; its remaining count is
not the strict count. Corpus projects, configuration and flags are unchanged.
Measurement artifacts are in `target/resolution/builtin-catalogue-audit/`.

## Recognition and resolution are separate

The committed `internal/docs/generated_docs.go` catalogue contains 970 function
entries and 174 tag entries. `internal/docs/undocumented.go` supplements builtin
recognition with 48 additional Lucee functions (hidden or newer than the docs),
with no overlap: 1,018 function spellings before tag-as-function recognition.
The earlier conversational figure of 1,013 documented functions was incorrect. `IsBuiltinFunction` recognizes a spelling;
that alone does not give its result a type, establish its fields, or verify
availability on a particular engine/version.

Return typing is separately curated in `internal/nativetype` and the legacy
`internal/docs/builtin_returns.go` table. This batch adds 33 fixed collection
producers to `internal/nativetype/collection_producers.go`. Each agrees with both
the committed catalogue and the [Lucee 7.1 function library](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/resource/fld/core-base.fld).
This is a bounded extension, not automatic typing of every documented return.

The initial backwards text probe found 499 nearby producer candidates. It can
cross lexical scopes and mistake a field assignment for its parent receiver.
That number is a research queue, not a forecast of resolved findings.

## Implemented contracts

| Result | Global functions |
|---|---|
| array | `arrayfilter`, `arrayfindall`, `arrayfindallnocase`, `arraymap`, `arraymerge`, `arraymid`, `arrayremoveduplicates`, `arrayreverse`, `arrayslice`, `arraysplice`, `componentlistpackage`, `entitynamearray`, `listqualifiedtoarray`, `listtoarray`, `querycolumnarray`, `querycolumndata`, `rematch`, `rematchnocase`, `structfindkey`, `structfindvalue`, `structkeyarray`, `structsort`, `structvaluearray` |
| struct | `directoryinfo`, `getapplicationmetadata`, `getapplicationsettings`, `gethttprequestdata`, `gethttprequestheaders`, `gettagdata`, `gettaglist`, `structfilter`, `structmap`, `structtosorted` |

Only the container interface is known. Array elements and struct values remain
erased. Argument-dependent producers such as `deserializeJSON`, `directoryList`
and `duplicate` are excluded. The existing function catalogue's availability
assumption remains: recognition is not a deployment compatibility check.

Callable parameters, earlier local bindings and callback parameters shadow builtin
spellings in parsing and resolution. Declared source functions retain priority.
Callback shadow recovery in the resolver is line-based and conservatively withholds
evidence when sibling callbacks share a line with a shadowing parameter.

Native function-scoped assignments may type calls at or after their assignment,
but cannot provide forward evidence before it. This prevents an incoming iterator
argument in MuraScope.each from becoming an array merely because a later branch
assigns `listToArray` to it. Existing CFC forward-reference behavior is retained.
This is a bounded ordering correction, not a general control-flow analysis.

Struct property hops are accepted as erased values, rather than diagnosed as
nonexistent methods. A property named `mappings` does not establish a struct
value. On direct builtin chains the existing erased-value policy accepts later
calls dynamically; assigned unknown field receivers can remain unresolved.
Arrays are not granted arbitrary property hops.

Added array APIs: `getDimension` returns numeric, `toArray` returns an erased Java
array, and `subList` returns a Java List with erased elements. Evidence comes from
[Lucee Array](https://github.com/lucee/Lucee/blob/7.1/loader/src/main/java/lucee/runtime/type/Array.java),
[ArraySupport](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/type/util/ArraySupport.java),
and the [Java List contract](https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/util/List.html).
The existing native array API convention includes Lucee's Java interoperability;
it is not a guarantee that every engine/version exposes the same methods.

## Probe review

The first producer-only comparison reduced strict findings by 75 but introduced
10 entries (including changed diagnostic reasons); liberal findings increased
by 10. Every addition was reviewed before accepting the batch:

- Four MuraScope iterator calls borrowed a future array assignment: fixed by
  withholding forward native references.
- Mura resourceBundle.toArray, Wheels.subList and Lucee.getDimension exposed
  supported array APIs: added verified public contracts.
- Three BoxLangMappingHelper copies treated `.mappings` as a nonexistent struct
  method: corrected property access to an erased value.

The broader assignment-order correction also withdraws a Wheels `local.val.hashCode()`
resolution that had borrowed later native assignment evidence. FW/1 `pathInfo.split()`
changes from a wrong array-method diagnostic to an unknown receiver diagnostic;
its preceding value still needs a string producer/CGI relationship. These are
reviewed losses of unsupported evidence, not new array methods to invent.

The initial run is retained in `probe-checked/`; final measurements use a rebuilt
binary and go into `checked/`. No initial result is presented as a final gain.

## Validation

Producer, callable shadow, assignment-order, struct-property and public-interface
regressions were exercised with the relevant fixes removed, and failed. Focused
positive and boundary tests pass with the fixes restored. Counterfactual logs are
retained in the artifact directory. Wrong-family methods stay diagnosed, and
contract tests assert erased Java array, List element and struct field results.
Full validation passed: `go build ./...`, embedded build, `go vet ./...`,
embedded vet, `go test -short ./...`, `go test -short -race ./...`, `make lint`,
`bash scripts/check-go-format.sh` and `git diff --check`. Full suites ran with
approved local socket access after the corpus scans finished. Short and race
logs are retained beside the corpus artifacts. No published PR/CI result is claimed.

## Final measured result

| Project | Strict before | Strict after | Liberal before | Liberal after |
|---|---:|---:|---:|---:|
| wheels | 2,570 | 2,565 | 143 | 143 |
| masa | 3,687 | 3,686 | 21 | 21 |
| contentbox | 1,754 | 1,746 | 44 | 44 |
| coldbox | 1,637 | 1,628 | 93 | 93 |
| lucee | 2,065 | 2,007 | 198 | 196 |
| testbox | 396 | 395 | 7 | 7 |
| fw1 | 359 | 359 | 6 | 5 |
| **Total** | **12,468** | **12,386** | **512** | **509** |

Strict comparison: 84 entries removed, two added, net 82 fewer. One addition is
FW/1's changed diagnostic; the other withdraws unsupported Wheels forward typing.
Liberal comparison: three removed, zero added. FW/1's corrected unknown receiver
now qualifies for the existing optional string-family guess; this is not a strict
string resolution. Runtime assumptions move from 6,936 to 6,915: 22 removed and
one added (Wheels hashCode). Speculative receiver connections move from 1,928 to
1,899: 31 removed and two added. The other added connection is Lucee StructToSorted
at a site that still needs better assignment/callback ownership recovery.

Coverage is unchanged: 8,151 indexed files, 189,154 CLI counting records,
46 unchanged base summaries representing 1,133 unchecked calls. Per-entry
comparisons assert unchanged counting totals and base summary records, and the
review asserts unchanged per-project file counts.

The remaining larger gaps still concern source relationships: argument/CGI string
producers, branch and callback ownership, wrapper return shapes, and dynamic
framework receiver identity. Extending the recognized-name list alone does not
solve those connections.
