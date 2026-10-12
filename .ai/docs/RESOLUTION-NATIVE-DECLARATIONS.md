# Explicit collection declarations and public runtime collection methods

This batch reduces the strict seven-project total from **13,249 to 13,088**:
**161 removed, zero added**. Ranked/native plus open-world mode changes from
**788 to 702** retained findings, **86 removed, zero added**. Its assumptions
change from **6,725 to 6,592**, **133 removed, zero added** per entry. Retained
findings plus assumptions are **7,294**; these remain different proof levels.

## Implemented relationships

Explicit source functions declared `array` or `query` now supply the corresponding
native runtime interface, even when their producer cannot establish a CFC return.
Assignments, direct chains and collection callbacks use the same return contract.
An array's elements remain unknown without separate element evidence. Query
callbacks retain the existing row/index/query layout. Script and tag function
declarations are covered.

Generated property and relationship accessors now carry a `Generated` marker.
They do not acquire this contract from property metadata: a property's type is
not proof of what its generated getter returns. Explicit accessor overrides are
source functions and can establish it. Index edits adding or removing an explicit
override refresh the marker and return contract. This avoids source-file reads
merely to distinguish generated functions, and works for in-memory indexing.

The supported native API profile also adds array `addAll` (boolean result), array
`toJson` (string), and query `valueList`/`toJson` (string). These are API contracts,
not CFC paths or element contracts. Native missing methods remain diagnosed.

## Assumptions and sources

An explicit array/query function declaration describes its normal returned value;
this does not prove that the producer exists, that the call executes successfully,
or that an array contains one particular component. Adobe documents return types
and conversion/error behavior in [cffunction](https://helpx.adobe.com/coldfusion/cfml-reference/coldfusion-tags/tags-f/cffunction.html)
and [creating user-defined functions](https://helpx.adobe.com/dk/coldfusion/developing-applications/building-blocks-of-coldfusion-applications/writing-and-calling-user-defined-functions/creating-user-defined-functions.html).

The public native profile describes supported runtime APIs without verifying an
installed engine/version. Lucee's [ArraySupport source](https://github.com/lucee/Lucee/blob/6.2/core/src/main/java/lucee/runtime/type/util/ArraySupport.java)
extends `AbstractList`, implements `List`, and exposes `boolean addAll(Collection)`.
The fetched source is retained in the validation artifacts. BoxLang also documents
Java List methods on its [arrays](https://boxlang.ortusbooks.com/boxlang-language/syntax/arrays).
Lucee documents [any.toJson](https://docs.lucee.org/reference/objects/any/tojson.html)
as producing a string. Query member `valueList` is exercised by the official
Lucee corpus's `test/functions/QuerySort.cfc`; the underlying operation and string
result are described in [ValueList](https://docs.lucee.org/reference/functions/valuelist.html).
This batch supplies only those bounded collection entries, not every Java List
method or a universal serialization contract on arbitrary CFCs.

`struct` declarations remain excluded: CFML permits a CFC instance to satisfy
that declaration, and a CFC's custom `each` does not imply struct callback layout.
The existing source inference for a concrete struct-returned CFC is preserved.
Shared registry value types, closure writer lifetimes, and map output elements
remain open. This does not type AsyncManager's executor values or LogBox appenders.

## Reviewed probe and corpus results

The initial probe included scalar declarations and synthetic accessor return
types. It exposed false positives around ColdBox's timezone getter, FW/1's
conditional output, and Wheels' object guard. Those changes were not shipped.
The final implementation excludes scalar declarations and generated getters;
none of those probe additions remains in the final comparison.

| Project | Strict before | Strict after | Removed | Added |
|---|---:|---:|---:|---:|
| Wheels | 2,636 | 2,575 | 61 | 0 |
| Masa | 3,689 | 3,687 | 2 | 0 |
| ContentBox | 1,786 | 1,754 | 32 | 0 |
| ColdBox | 1,654 | 1,640 | 14 | 0 |
| Lucee | 2,729 | 2,677 | 52 | 0 |
| TestBox | 396 | 396 | 0 | 0 |
| FW/1 | 359 | 359 | 0 | 0 |

The removed method-name inventory includes 74 `addAll`, 15 `toJson`, 16 `valueList`,
and 56 other container operations/chains. This is a symptom inventory, not an
independent causal attribution to each rule. All 14 ColdBox removals were audited
in the actual scan: each reaches a builtin native array interface with an empty
reason and no CFC URI. The instrumented report exactly matches the ordinary report.

Coverage is unchanged at **8,151** indexed/scanned files and **189,154** CLI
accepted/reported/assumed records. Missing-base summaries are unchanged per entry,
representing **1,133** unchecked calls. This is not an instrumented raw parser
extraction comparison. Liberal assumption categories now total runtime receiver
**4,496**, runtime return **1,225**, runtime function/mixin **605**, and external
component **266**.

## Verification

The positive declaration and callback tests fail with the return rule disabled;
the new native method contracts also fail independently when removed. Disabling
the generated-function exclusion incorrectly supplies a query type to a synthetic
getter. Tests cover missing methods, unknown array elements, struct-returned CFCs,
script/tag source declarations, explicit accessor overrides and index edits.

Artifacts are under `target/resolution/native-declarations/`: `probe/` retains the
broader rejected scan, `checked/` contains final reports and per-entry differences,
`review.json` records coverage and assumption differences, `audit-targets.json`
contains the ColdBox target audit, and the counterfactual/validation logs retain
their actual results. No CFML engine execution or published-head CI claim is made.

Final build, vet, full short tests, full short race tests, pinned lint and diff
checks pass. Full tests ran with approved local socket access, `GOMAXPROCS=2` and
`-p 1`, after corpus scans. The final production hashes match the corpus snapshot;
late changes add a tag declaration regression and test formatting only.
