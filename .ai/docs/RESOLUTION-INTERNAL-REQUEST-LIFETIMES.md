# InternalRequest local lifetime refinement

This follow-up starts from the 12,554 strict findings recorded in
[the result field report](RESOLUTION-INTERNAL-REQUEST-FIELDS.md). It reviews the
57 remaining Lucee filecontent findings before changing the resolver.

## Relationships and implementation

Both official implementations inspected in the previous batch allocate a new
result struct for each InternalRequest call:
[5.4 source](https://github.com/lucee/Lucee/blob/5.4/core/src/main/java/lucee/runtime/functions/system/InternalRequest.java)
and [7.1 source](https://github.com/lucee/Lucee/blob/7.1/core/src/main/java/lucee/runtime/functions/system/InternalRequest.java).
A new explicit local assignment therefore starts a new result lifetime. Earlier
reads, field writes, and aliases of the previous struct do not establish a write
to the new struct. The blanket veto on any earlier mention of the local is
removed; the selected initializer and its subsequent lifetime remain checked.

This does not make a captured variable safe: a closure can retain the local
binding and access its newly assigned value. Existing anonymous block closure
checks remain. A new named nested function check also withholds inference when
its body retains the result. The enclosing named function itself is excluded
using the lookup's byte position. This closes a missing safety condition in the
previous strict field inference without introducing a new assumption policy.
The existing opt-in native guessing mode can independently accept receivers
withheld by these guards; those remain speculative connections.

The response's status and status_code fields are numeric scalars in both engine
implementations. Passing these values to assertions or comparing them using
supported comparison syntax does not expose the mutable response struct. Such
reads now preserve later filecontent typing. Writes, increment operations,
unknown property member calls, and escapes of the response itself still
withhold the contract. Struct/array fields, such as cookies or headers, require
separate alias reasoning and are not covered by this scalar rule.

Engine selection, explicit local assignment, builtin shadow checks, completed
initializer, lexical ownership, normal-path dominance and bounded scanning
requirements remain. No parser metadata, default file limits, or new config
flag changes are introduced.

## Remaining opportunities

The 57 findings also include bare assignments in functions declaring
localmode=true, repeated bare assignments after an explicit local initializer,
source wrappers returning engine responses, cfhttp outputs, nested callback
writers, and semicolon-free producer prefixes. Those require distinct scope,
producer, binary/text, execution, or statement-boundary proofs. Their shared
filecontent spelling is insufficient evidence of an identical contract.

Lucee's [local mode guide](https://docs.lucee.org/recipes/migrate.html) confirms
that an explicit function localmode=true / modern declaration makes unscoped
assignments local. Classic mode instead cascades through existing local/argument
keys before the variables scope. A future rule can inspect the literal function
header, then retain the same lifetime checks; it must distinguish function-level
settings from application/server defaults and sibling callbacks. This is verified
scope research, not an implemented mapping or a quantified corpus gain.

The corpus includes HTTP binary-to-string use. InternalRequest's separate
binary field contract must not be transferred to cfhttp, which needs its own
engine result shape and content-mode investigation.

## Verification

New positive regressions cover fresh assignments after earlier reads, aliasing,
escapes and writes, plus status_code reads and comparisons. A negative regression
covers named nested functions that write a field or pass the response to another
call. All fail against the previous production source and pass with this change.
Existing scope, anonymous capture, mutation, escape, builtin shadow and independent
callback fixtures are retained.

Artifacts and per-entry comparison live under
`target/resolution/internal-request-lifetimes/`; the baseline is
`target/resolution/internal-request-fields/checked/`.

### Final corpus result

| Entry | Strict before | Strict after | Removed | Added |
|---|---:|---:|---:|---:|
| Wheels, including explicit vendor root | 2,570 | 2,570 | 0 | 0 |
| Masa | 3,687 | 3,687 | 0 | 0 |
| ContentBox | 1,754 | 1,754 | 0 | 0 |
| ColdBox | 1,637 | 1,637 | 0 | 0 |
| Lucee | 2,151 | 2,136 | 15 | 0 |
| TestBox | 396 | 396 | 0 | 0 |
| FW/1 | 359 | 359 | 0 | 0 |
| Total | **12,554** | **12,539** | **15** | **0** |

All fifteen removals are filecontent trim calls. A source audit confirms explicit
engine initializers for each. The Lucee filecontent remainder moves **57 → 42**.
The gains include reinitializations, a formerly colliding unrelated property
name, and status reads. Ordinary aliases and escapes of an earlier object do
not survive the new assignment as references to the new object; lexical captures
continue to be withheld.

Liberal ranked/native/open-world mode is unchanged at **568 findings** plus
**6,936 runtime assumptions**. Its combined remainder is **7,504**. These calls
were already accepted through optional guesses. Speculative receiver connections
move **1,954 → 1,942**: thirteen old filecontent string connections disappear and
one remaining receiver becomes a separate string guess under the existing opt-in
policy. Connection counts describe receiver groups, rather than individual calls.
No runtime assumptions are added or removed. All other corpus entries' findings,
assumptions and connections are unchanged.

All fourteen comparisons retain **8,151 indexed/scanned files**, **189,154 CLI
counting records**, and **46 missing-base summaries / 1,133 unchecked calls**.
The runner checks these partitions per entry. After extracting scalar-read
validation into a helper to meet the lint complexity limit, the final executable
reruns both Lucee modes and reproduces their complete JSON reports exactly.

Build, vet (including cflint_embed), focused tests, full short and full short-race
suites, lint, formatting and whitespace checks pass. Full suites ran sequentially
after corpus scanning with approved local Unix socket access. Production/test
source hashes remain unchanged after validation. No CFML runtime execution or
published-commit CI pass is claimed.
