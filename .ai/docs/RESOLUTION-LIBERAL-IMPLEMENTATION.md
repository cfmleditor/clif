# Liberal resolution implementation and audit

11 October 2026. This batch implements source-backed relationships and two
separately enabled batch policies. The request was to pursue fewer than 1,000
outstanding findings with more liberal assumptions. That is achievable as a
triage count; it is not equivalent to proving every formerly unresolved call.

## Implemented source relationships

Top-level block callbacks now record constructor locals and settled factory
locals in their own reference bucket, with existing closure byte spans. They
retain global call records without becoming global variable bindings. The
resolver reads this bucket directly, including when a script region starts
later in a template. Sibling and nested callbacks retain distinct ownership.
An unknown callback parameter shadows an outer same-named binding; provider
and criteria-builder invocation contracts may still supply its type.

Assigned constructor chains retain argument-bearing hops. For example,
`new Factory().choose(new Product())` can use the producer's supplied object
instead of losing the argument and becoming dynamic. The regression checks
both a declared Product method and a missing method. This does not implement
the experimental scalar branch selector or broadly change unknown-return
fallbacks; those changes previously exposed thousands of additional findings.
Literal-array inference also remains deferred because its conservative research
variant provided no corpus gain.

The strict target audit reaches 46 Wheels checkout declarations and two committed
CBORM framework-stub declarations (`Restrictions.like`). The latter are preset
API contracts, not declarations installed in the ContentBox checkout. All 48
audited entries have empty resolution reasons; no speculative policy is enabled
in this audit.

## Ranked receiver assumptions

`--speculative-ranked` intersects the full candidate pool for each unknown
file/caller/variable group. It distinguishes methods on the base receiver from
methods on returned objects, then uses existing name-affinity confidence to
select a unique best component. Ties remain unresolved; path proximity is not
a tie-breaker. Property-first chains and ambiguous same-line receiver roles
withhold guessing. Display truncation does not limit the matching pool.

This remains unreliable. Method/name compatibility does not prove runtime
origin, assignment lifetime or DI ownership. A supplied primitive can still
match a component's methods, as the earlier sole-candidate controls demonstrate.
Groups do not distinguish every callback or assignment lifetime. Source-known
receiver errors are not redirected to another class. Each proposed connection
records its source, group, chosen component and `ranked-cfc-role` policy;
verbose output exposes it. Native-family guessing remains a separate switch.

## Open-world assumptions

`--assume-open-world` partitions an ordinary or speculative report into
remaining `calls` and retained `assumedCalls`. It does not modify the resolver,
create definitions or increment `resolved` for an assumed call.

| Policy | Assumption | How it can fail |
| --- | --- | --- |
| runtime-receiver | An untyped receiver is supplied by runtime code | The value may be primitive, absent or the wrong object |
| runtime-return | A method with no object return contract supplies a usable receiver | Its return may be null, primitive or another class |
| external-component | An absent literal component path is provided in deployment | The path may be misspelled or genuinely absent |
| runtime-function-mixin | A bare unqualified function is injected at runtime | The function may be misspelled or never registered |

Explicit `this.method()` errors, known receiver missing methods, broken chains
and missing-base summaries remain findings. The latter retain their unchecked
call counts. Unknown receivers are still assumed even when their method has no
indexed declaration; this is an explicit open-world assumption, not a candidate
match. Every assumed entry preserves its original diagnostic and candidate
metadata, rather than treating the absence of an error as evidence of a target.

With `--json`, this mode emits an object containing `calls`, `assumedCalls`,
`speculativeConnections`, `resolved`, `indexed` and `scanned`. Ordinary JSON
mode remains the existing array. Open-world mode cannot write a known-issues
baseline because that would discard the assumption audit. Both new switches
are off by default, have explicit off switches, and apply to the batch CLI;
editor diagnostics retain ordinary resolution.

```sh
clif unresolved --json --speculative-ranked --speculative-native --assume-open-world project > resolution-audit.json
```

## Final corpus results

Same seven-project roots and configuration as the prior 13,346 / 8,575
baseline, including explicit Wheels vendor coverage. The final batch contains
35 comparisons, following 35 initial scans and two strict target-audit scans.

| Mode | Remaining findings | Retained assumptions | Removed / added versus prior baseline |
| --- | ---: | ---: | ---: |
| Strict (before 13,346) | 13,300 | 0 | 48 / 2 |
| Existing sole/native flags (before 8,575) | 8,532 | 0 | 44 / 1 |
| Ranked/native flags (before 8,575) | 7,552 | 0 | 1,102 / 79 |
| Open world, without guesses (before 13,346) | 739 | 12,561 | 12,607 / 0 |
| Ranked/native + open world (before 8,575) | 788 | 6,764 | 7,834 / 47 |

Open-world removals include reclassification into assumptions. They are not
source-resolution fixes. Strict removes 48/adds two, net 46 fewer. Existing
sole/native mode removes 44/adds one, net 43 fewer. Ranked/native mode removes
1,102/adds 79 versus the prior both-flags baseline; these are speculative
connections and replacement diagnostics, not 1,102 verified fixes.

The two strict additions are callback parameters in ColdBox: `arguments.key`
inside RequestScope.clear's filter and `thisAppender` inside Logger.logMessage's
filter. They no longer borrow an enclosing argument or outer local. Their
collection callback contracts need further evidence. The latter remains an
addition with the existing speculative flags; the former receives a native guess.

All modes preserve **8,151 indexed/scanned files**, **189,154 checked/report
records**, **46 missing-base summaries** and **1,133 unchecked inherited calls**.
The exact missing-base diagnostic multisets are unchanged. Those 1,133 calls
remain unchecked even though the audit has fewer than 1,000 remaining entries.
Each open-world partition equals its corresponding final ordinary report per
diagnostic multiset: `calls + assumedCalls`, with unchanged accepted-call count.
In speculative modes, `resolved` includes guessed acceptances, as in the
existing CLI; it is not a runtime verification count.

| Assumption policy | Open world alone | Ranked/native + open world |
| --- | ---: | ---: |
| runtime-receiver | 10,437 | 4,615 |
| runtime-return | 1,227 | 1,252 |
| external-component | 292 | 292 |
| runtime-function-mixin | 605 | 605 |

The combined remainder is higher because assigning guessed types can expose
more specific failures that the open-world rules preserve. Prominent remaining
groups include native-array addAll (74), add (18) and iterator (13), native-query
to (16 in combined mode), native-string columnData (14), dbinfo columns/version
(13 each), and missing mapper chains. These are review cohorts, not verified
application defects. Native API omissions and incorrect prior type guesses can
both contribute.

| Project | Strict | Ranked/native | Open world alone | Combined |
| --- | ---: | ---: | ---: | ---: |
| wheels | 2,636 | 1,505 | 190 | 225 |
| masa | 3,689 | 2,069 | 23 | 23 |
| contentbox | 1,835 | 897 | 51 | 52 |
| coldbox | 1,655 | 789 | 97 | 105 |
| lucee | 2,730 | 1,812 | 366 | 370 |
| testbox | 396 | 259 | 6 | 7 |
| fw1 | 359 | 221 | 6 | 6 |

## Verification and limits

The top-level constructor/parameter regressions failed before their fixes.
Disabling argument-bearing constructor hops makes the missing-method regression
fail again. Independently disabling ranked scanning and open-world partitioning
makes their integration controls fail. Positive, negative, shadow, nesting,
escape, shifted-template, tie, more-than-eight-candidate and ambiguous-chain
controls exercise the relationships and policy boundaries. JSON round-trip
coverage checks preservation of original assumed diagnostics and counts.

Plain Go build, vet, full short tests, full short race tests, pinned lint and
diff checks pass. The first sandboxed full suite could not bind local Unix
sockets; the complete suites pass with approved socket access. No tests were
weakened. Nine implementation source hashes remain unchanged between final
build, corpus comparisons and acceptance audits. The final review script checks
coverage, exact partitions, target audits and source hashes.

Artifacts are under `target/resolution/liberal-implementation/`. The root modes
and summary retain the initial pass; `final/` retains comparisons for the final
implementation. Reports keep per-entry removals and additions, each mode's
complete JSON and verbose log, coverage totals, policy counts, mutation controls
and source hashes. Assumption removals are classifications, not verified fixes.
No CFML engine execution or published CI result is claimed.

## What remains to reach fewer than 1,000 actual unknowns

The low audit count depends on thousands of unproven runtime relationships.
Turning those into source-backed resolution requires callback collection element
contracts, record-field propagation, return alternatives and discriminants,
framework DI/decorator ownership, and deployment dependency evidence. Known
method errors still require review against runtime/native contracts. The audit
now supplies concrete cohorts for that work rather than erasing the underlying
uncertainty. The existing scalar combined prototype should still be decomposed
before implementation, particularly where unknown returns became `$any`.

## Collection callback follow-up

See [collection callback invocation contracts](RESOLUTION-COLLECTION-CALLBACKS.md)
for the next implemented source relationships, speculative provenance guard,
per-entry corpus review, and remaining collection ownership gaps.
