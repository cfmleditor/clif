# InternalRequest result field resolution

For the subsequent refinement of fresh local assignments, scalar reads and
named capture checks, see [the lifetime follow-up](RESOLUTION-INTERNAL-REQUEST-LIFETIMES.md).
The counts and earlier-use boundary below describe this report's original checkpoint.

This follow-up investigates engine-created result fields rather than guessing a
receiver's type from its method name. The original `result.filecontent` sample
was primarily Lucee's InternalRequest response, not an HTTP tag output.

## Verified relationship

Lucee's [InternalRequest reference](https://docs.lucee.org/reference/functions/internalrequest.html)
returns a struct. Its public name appeared in Lucee 6; older versions expose
`_InternalRequest`. The [5.4 implementation](https://github.com/lucee/Lucee/blob/5.4/core/src/main/java/lucee/runtime/functions/system/InternalRequest.java)
and [7.1 implementation](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/functions/system/InternalRequest.java)
both place text into `filecontent` and binary data into a separate
`filecontent_binary` field. Therefore `filecontent`, when present, is a string.
This does not establish that every response contains that field.

The resolver now recognizes this relationship with the Lucee engine selected,
for explicit `var result = InternalRequest(...)` or
`local.result = _InternalRequest(...)` initializers. Names and field spelling
are case insensitive; the local variable need not be named `result`.
An existing exact member reference retains priority.

The [string.deserializeJSON contract](https://docs.lucee.org/reference/objects/string/deserializejson.html)
is also implemented with erased `$any` return identity. JSON can decode to
multiple value types; accepting that member does not establish a struct or
array for subsequent calls.

## Lifetime and proof boundaries

The initializer must be a complete, unqualified builtin call in the enclosing
function. Custom declared, inherited, local, and argument callables shadow the
engine API. A qualified method with the same name does not acquire its shape.
Only explicit locals qualify; bare assignments and shared scopes remain open.

The bounded source scan checks normal-path completion before the lookup. It
withholds inference across closed branches and sibling closures, unbraced
conditional initializers, root rebindings, field writes, dynamic indexing,
whole-result mutation, aliases, escapes, interpolated intervening strings and
unknown earlier member calls. Repeated supported string member reads qualify.
Nested block closures retaining the container also withhold inference, even outside
the initializer-to-lookup interval. Earlier uses of the same local before the
selected initializer are conservatively withheld.

The scan retains existing token and multiline-initializer bounds. Exceeding
those bounds withholds the new type; it does not skip a corpus file or finding.
No parser structure size changes, open-world acceptance, or new configuration
flag are required. Engine selection remains the existing override boundary.

Other engine result fields, cfhttp outputs, aliases, binary field methods,
missing-field existence checks, shared lifetimes, and arbitrary wrappers are
not implemented by this rule.

## Verification and measurement

The positive regression fails without the resolver hook, including repeated
reads, multiline initializers, both builtin names, explicit local scopes and
native.string identity on a deliberately missing method. The new JSON API
contract test also fails without its addition. Negative fixtures cover scope,
mutation, escape, callback capture and callable shadowing boundaries.

Research artifacts are under
`target/resolution/internal-request-fields/`; the baseline is
`target/resolution/query-output-bindings/checked/`.

### Final corpus comparison

All fourteen strict/liberal entry comparisons use the same roots and config as
the query-output baseline. No files, base summaries, or unchecked calls disappear.

| Entry | Strict before | Strict after | Removed | Added |
|---|---:|---:|---:|---:|
| Wheels, including explicit vendor root | 2,570 | 2,570 | 0 | 0 |
| Masa | 3,687 | 3,687 | 0 | 0 |
| ContentBox | 1,754 | 1,754 | 0 | 0 |
| ColdBox | 1,637 | 1,637 | 0 | 0 |
| Lucee | 2,574 | 2,151 | 423 | 0 |
| TestBox | 396 | 396 | 0 | 0 |
| FW/1 | 359 | 359 | 0 | 0 |
| Total | **12,977** | **12,554** | **423** | **0** |

The Lucee filecontent group moves **479 → 57**, resolving 422 of its findings
(**88.10%**). The broader strict corpus reduction is 423 of 12,977 (**3.26%**).
The remaining field findings need separate lifetime/scope/producer proof.

The removals span 154 files: 421 filecontent trim calls, one filecontent
JSON decode call, and one already-known string JSON decode call. A source audit
of all 422 field removals confirms preceding explicit `_InternalRequest`
initializers. Explain probes also confirm field/string JSON acceptance under
the corpus root configuration.

The initial version with the JSON contract removed 434 findings. Adding the
capture and earlier-use safeguards conservatively withholds eleven of those
removals, including repeated reinitializations after earlier uses. Those
lifetimes remain an opportunity for a more precise proof; they are not treated
as runtime errors by this research. The guard's capture regression fails when
that guard is removed, and independent sibling callback locals remain resolvable.

Liberal ranked/native/open-world mode moves **570 → 568** findings and
**6,943 → 6,936** runtime assumptions, with no added findings or assumptions.
The two finding removals are JSON members. Seven runtime-receiver assumptions
are replaced by engine evidence. Its combined findings/assumptions move
**7,513 → 7,504**; strict improvements must not be added again to that partition.

Speculative receiver connections move **2,113 → 1,954**: 169 old string
connections disappear, while ten conservative remaining result receivers
become separate string guesses through the existing opt-in policy. All changed
connections concern filecontent; most newly strict calls had already been
accepted speculatively. Connections count receiver groups, not call sites, so
this is not a one-for-one comparison with the 423 strict removals.
All other entries' findings, assumption and connection partitions are unchanged.

Coverage remains **8,151 indexed/scanned files**, **189,154 CLI counting
records**, and **46 missing-base summaries representing 1,133 unchecked calls**.
The runner asserts per-entry record and missing-base equality; the review also
checks each entry's file count and compares assumption/connection identities.

Validation: build, vet (including cflint_embed), focused resolver/native/unresolved
checks, counterfactual regressions, full short and full short-race suites, lint,
Go formatting and diff whitespace checks pass. Full suites ran sequentially
after the corpus scans with approved local Unix socket access. Production and
test source hashes remain unchanged after validation. No CFML runtime execution or
published-commit CI pass is claimed.
