# Native array List contracts

This follow-up addresses already identified native array receivers that lacked
public List methods. It does not infer a collection from a method name or add
methods to arbitrary CFCs.

## Evidence and implementation

[Lucee ArraySupport](https://github.com/lucee/Lucee/blob/6.2/core/src/main/java/lucee/runtime/type/util/ArraySupport.java)
extends AbstractList and implements List. Its explicit removeAll and retainAll
implementations return boolean. The reviewed source is saved in
`target/resolution/native-declarations/ArraySupport.java`.
[Java List](https://docs.oracle.com/en/java/javase/21/docs/api/java.base/java/util/List.html)
defines iterator, size, get, containsAll and the overloaded add/remove methods.
[BoxLang array documentation](https://boxlang.ortusbooks.com/boxlang-language/syntax/arrays)
also documents public Java collection operations.

The native array contract now exposes size as numeric, iterator as a native
iterator, and containsAll/removeAll/retainAll as boolean. get returns an erased
value; add and remove also remain erased because their overloads have different
return contracts. Existing iterator.next remains erased. No element type or
successful runtime execution is inferred. These contracts share the existing
public native API profile; they do not verify engine/version availability.

The additional get contract makes a lone get observation ambiguous between
arrays and structs in the primitive speculative fallback. Its previous struct
guess is deliberately withdrawn. This is a consequence of improving the API
catalogue, not an additional receiver inference rule.

## Review and validation

The contract table and resolver regression failed before the production change;
`target/resolution/array-list-contracts/before.log` records the failures. The
regression checks the public iterator chain, primitive size result, absent array
methods, and custom CFC iterator ownership. The table independently checks
that get/add/remove return $any and iterator.next retains $any. The resolver's
existing handling of erased values may accept downstream calls dynamically;
that is not evidence that any particular element method exists.

Corpus reports compare all seven projects in strict and liberal modes per
entry against `target/resolution/native-declarations/checked`, including
assumption partitions, file coverage, counting records and missing bases.
Artifacts are in `target/resolution/array-list-contracts/`.

## Measured result

| Mode | Before | After | Removed | Added |
|---|---:|---:|---:|---:|
| Strict | 13,088 | 13,051 | 37 | 0 |
| Liberal retained findings | 702 | 665 | 37 | 0 |

Strict removals are Wheels add (5), Lucee add (13), iterator (13), and
containsAll (6). The other five projects have no strict per-entry changes.
Coverage remains 8,151 files, 189,154 CLI counting records, and the same
46 missing-base summaries representing 1,133 unchecked calls. Counting records
are CLI accounting, not a new raw-parser coverage measurement.

Liberal assumptions increase from **6,592 to 6,738**: 14 old entries removed,
160 added. Every added entry is runtime-receiver get. Previously those calls
were accepted through a speculative struct guess; the expanded public array
API exposes that get alone cannot distinguish struct from array. This makes
uncertainty visible in the assumption report. Some examples actually refer to
futures or services, reinforcing why get alone is weak receiver evidence.
The combined retained-plus-assumed total consequently increases from 7,294
to 7,403 despite the 37 strict API fixes. Do not describe the liberal retained
count alone as an overall uncertainty reduction.

Build, vet, focused resolver/native tests, full short and race suites, lint
and git diff --check pass. Logs accompany the corpus artifacts. No CFML
engine execution or published CI run is claimed.
