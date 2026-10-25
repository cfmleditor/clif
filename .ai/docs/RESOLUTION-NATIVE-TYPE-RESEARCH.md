# Native object and type methods

Research checkpoint, 2026-10-10; implementation follow-up below. Extends
[engine ORM research](RESOLUTION-ENGINE-ORM-RESEARCH.md) and informs future
[opt-in connection policies](RESOLUTION-SPECULATIVE-CONNECTIONS.md).

## What toString reveals

Java objects inherit public methods such as toString, equals, hashCode and
getClass from [java.lang.Object](https://docs.oracle.com/en/java/javase/17/docs/api/java.base/java/lang/Object.html).
Their results are String, boolean, int and Class respectively. Object.clone and
finalize are protected; the existence of a method in an ancestor is not enough
to establish that it is publicly callable. Java visibility and engine exposure
must be represented. CFML also has a global
[ToString conversion function](https://docs.lucee.org/reference/functions/tostring.html);
a builtin global name is not itself proof of a member call's receiver or result.

clif already includes tostring, equals, hashcode and getclass in the broad
`IsMemberMethod` table (`internal/parser/result.go`), along with Class/reflection,
CFML collection and JavaScript methods. Normal CFC method lookup takes priority
before this fallback. The table is name-based, not receiver-specific, and does
not supply native return contracts.

The `$builtin.*` branch in `internal/resolve/resolve.go` instead checks the
producer's finite method table in `internal/docs/builtin_returns.go` and returns
before the broad member fallback. Those builtin objects only carry method
names, not per-method result types.

An isolated source probe demonstrates two inconsistencies:

```cfml
component {
    function run() {
        var q = queryNew("id", "integer");
        q.toString();
        q.getColumn("id");
        q.toString().trim();
    }
}
```

`explain` identifies q as `$builtin.querynew` but rejects toString because that
name is absent from the query producer table. On the chain it checks trim on
querynew rather than propagating the string result. The unresolved report emits
only getColumn: `scanFile` drops failures whose final method name is also a global
builtin, regardless of receiver qualification. Thus zero reported toString
findings cannot establish successful resolution. The companion top-level CFM
probe fails to infer q at all, revealing a separate assignment coverage difference.
These probes inspect clif behavior; they do not run the CFML engine.

Probe sources, JSON and explain output are retained under
`target/resolution/native-method-audit/probe/`, with an empty local config to
avoid inheriting repository settings. The binary is the checkpoint-31 build.

## Discovery inventory

Search of checkpoint-31 strict reports found the following method-name cohorts.
Counts include either final method names or matching method names in chain
failure reasons. Overlap is possible; these are not confirmed native receivers.

| Names or family | Entries matching each name |
| --- | ---: |
| toString / hashCode / equals / getClass / getName | 0 each |
| getColumn | 30 |
| iterator | 24 |
| hasNext | 52 |
| next | 84 |
| entrySet / keySet | 2 / 3 |
| size | 30 |
| toLocalDateTime | 9 |
| format | 4 |
| getStatistics | 2 |

The full selected-name inventory has 240 distinct report entries and is retained
in `target/resolution/native-method-audit/method-name-inventory.json`. It is a
discovery census, not a gain estimate. Many Mura hasNext/next calls refer to its
own CFML iterator abstraction; they require source-backed framework contracts.

Source examples establish promising families:

- Lucee LDEV3640/LDEV3734: getColumn on native queries, including known
  `$builtin.querynew`/queryexecute receivers. Verify engine implementation and
  supported versions before extending these producer tables.
- ColdBox InterceptorState.rebuildInterceptorChain: getInterceptors().entrySet()
  .iterator(), then hasNext/next/getKey/getValue. The missing connection starts
  at getInterceptors/getPool and continues through native collection interfaces.
- Lucee _setupTestServices: getPageContext().getConfig().getAllRHExtensions(),
  iterator and metadata/JDBC collections. This requires engine API and generic
  element contracts, not a guessed CFC declaring iterator.
- Wheels base.cfc: conn.createStatement()/prepareStatement(), executeQuery(),
  rs.next() and close(). These are JDBC interfaces with producer dependencies.
  Traced 2026-10-25 and left blocked: 208 findings, all in the CLI's
  `db/dump.cfc` (81), `db/create.cfc` (53), `base.cfc` (37) and
  `db/drop.cfc` (37). Every connection is made the same way: a loop over
  `local.dbConfig.driverClasses`, arrays of literal driver class names
  (`com.mysql.cj.jdbc.Driver`, `org.mariadb.jdbc.Driver`, …) that another
  function builds and returns as a struct field, runs
  `local.driver = createObject( "java", local.driverClass )`, and then
  `local.conn = local.driver.connect( local.url, local.props )`. A Java
  class named by a variable is `$any`, so connect() has no contract to be
  checked against, and nothing after it is typed. Three pieces would be
  needed, and only the first is small:
  1. a `java.sql.Driver` contract (connect → connection, acceptsURL →
     boolean), beside the connection/statement/resultset ones;
  2. proof that `local.driverClass` only ever holds driver class names,
     which means following string arrays through a field of a struct
     another function returns; nothing does that;
  3. for `dump.cfc`, the `connection` key of the struct
     `getDatabaseConnection()` returns. `CollectionValues` types a returned
     struct's values only when every value is one component, and this one
     also holds strings, so it would need values typed per key.
  `arguments.conn` in `checkDatabaseExists` and its siblings follows from
  (1) and (2), since every caller passes `local.conn`. Typing any Java
  object called as `connect( url, props )` as a driver would close it, and
  is a guess. The contract alone would type
  `createObject( "java", "<literal driver class>" )`, which this corpus
  never writes.
- ColdBox DateTimeHelper: Java time chains through toInstant, atZone and
  toLocalDateTime. The fluent result types are central to downstream checks.
- Lucee LDEV0866: structNew("linked") calling keySet/entrySet/iterator, guarded
  to skip Java versions above 8 because of access restrictions. A method's Java
  interface existence alone does not prove this CFML bridge call works on all
  configured versions.

## Receiver-specific contracts and experimental assumptions

[Iterator.next](https://docs.oracle.com/en/java/javase/17/docs/api/java.base/java/util/Iterator.html)
returns its element type, whereas
[ResultSet.next](https://docs.oracle.com/en/java/javase/17/docs/api/java.sql/java/sql/ResultSet.html)
returns boolean. [Map.entrySet](https://docs.oracle.com/en/java/javase/17/docs/api/java.base/java/util/Map.html)
returns a set of entries. A single next/entrySet/size whitelist cannot propagate
these types. Public Object methods should be inherited by proven Java receiver
types, with explicit CFC/custom implementations taking priority.

Next implementation work should first make reporting expose qualified builtin
failures instead of hiding them, recording any apparent increase as restored
diagnostic coverage. Then add producer and method return contracts: Object/Class,
String, Map/Set/Iterator/Entry, JDBC, engine queries, java.time and ORM interfaces.
Carry generic elements when supported; do not turn unknown elements into CFCs.
Engine, Java and extension versions must qualify compatibility assumptions.

For the unreliable path, any future native-family guess should be a separately
documented opt-in policy, emit the proposed family and evidence, and retain
method/result checks on that family. The assumptions would be that an observed
producer/usage shape identifies the native family and that runtime bridge
visibility matches the selected interface. Do not silently widen the existing
sole-CFC-candidate flag or add every native method name to global acceptance.

Validation needs wrong-family negatives (iterator next versus JDBC next), custom
toString return overrides, protected methods, builtin shadowing, chain results,
and report/explain agreement. Compare both unresolved and accepted record
identities: name filtering and broad acceptance can hide gaps in the headline.

Required preflight passed with approved network access after sandbox DNS failure.
This audit reused strict corpus reports and ran isolated CLI probes; no new
corpus scan, implementation gain, or runtime accuracy result is claimed.


## Implementation follow-up and compatibility assumptions

The current batch implements synthetic `$builtin.native.*` identities,
non-shadowed global ORM/date/collection producers, supported explicit Java class
identities, assignment/chain results and receiver-specific public methods.
Session versus factory statistics, generic Map entry iterators, JDBC boolean
next results and native time returns retain distinct contracts. Unknown Java
classes keep existing dynamic handling. Explicit configured Java resolvers and
real CFC methods have priority; a custom CFC toString can return a CFC.

Qualified builtin failures are now reported. A reporting-only binary is the
comparison control, because the previous global-name filter hid member failures.
Whole string/numeric/array literal assignments are recognized. Primitive CFC
return declarations are not globally promoted: that shortcut displaced existing
Wheels struct/factory contracts and violated established dynamic boundaries.

`--speculative-native` enables a separate unreliable policy, disabled by default.
Every unknown finding on the file/caller/receiver group must fit exactly one
catalogued family. Ambiguous/common names and incomplete groups are rejected.
`--no-speculative-native` restores strict handling; verbose output marks the
selected family and unproven origin. It can be combined with `--speculative`;
the sole CFC candidate policy gets first opportunity, followed by the native
family policy. Scalar/unknown actual origins can still be falsely accepted.

`--native-engine auto|generic|lucee` chooses the batch engine profile. Auto only
recognizes the existing marked Lucee source layout across all workspace roots;
generic withholds getPageContext's Lucee-specific producer. Explicit lucee is a
user-selected compatibility assumption, not a discovered deployment version.

The Java catalog assumes public Java 17 APIs and a compatible CFML Java bridge.
CFML member helpers and query methods assume the documented engine APIs. ORM
contracts assume the Hibernate 5.6 interface and compatible engine wrapper;
installed extension availability is not verified. Lucee internals assume the
corpus source profile. These are method/result contracts, not execution checks
for arguments, overloads, permissions, native exceptions or bridge visibility.
Version selection beyond the engine profile remains an open configuration need.

Erased Map/Iterator results and overload-dependent/unsupported array results
remain `$any`; Java arrays are not mislabeled as Lists. Map.values returns a
Collection rather than a List. Duration.dividedBy retains an unknown result
because overloads return different types. Public clone is limited to the known
HashMap family, and protected Object clone/finalize are rejected. Generic element
propagation through arbitrary loops, HQL entity selection, Field/Group narrowing
and Mura iterator instance state remain open; native terminology does not close
them. See the [completion ledger](RESOLUTION-COMPLETION-AUDIT.md).

Validation includes source overrides, wrong-family results, protected methods,
family ambiguity, incomplete groups, reversible flags and declared-source
priority. The native regression fails with the method/result catalog disabled,
and passes with it enabled. The tag script-island call-identity regression also
failed before its separate parser fix. Per-entry comparisons and current numbers
are recorded in `RESOLUTION-IMPLEMENTATION-STATUS.md` and
`target/resolution/native-contracts/`.


## Staged primitive usage guesses (checkpoint 34)

The strongest ten-pass shortcut experiment is now implemented under the existing
`--speculative-native` flag. A unique current interface-family answer retains
priority; only an unanswered stage tries String/Array/Struct/StringBuilder.
Every observed method must be supported by the chosen family. Ambiguity within
that stage and unsupported methods withhold it. This differs from requiring a
unique answer across all interface and primitive families together.

Native receiver origins remain unproven, including the recorded numeric-to-
StringBuilder counterexample. No method-argument, overload, CFML coercion or
runtime bridge verification is added. Common Object names remain insufficient
to select a family. Existing CFC precedence, known receiver checks and off flags
retain their behavior. The implementation shares the bounded method/result
catalog; no global builtin-report filter or primitive-return promotion changes.
See [policy and boundaries](RESOLUTION-SPECULATIVE-CONNECTIONS.md#delivered-primitive-family-fallback-checkpoint-34)
and checkpoint 34 of [implementation status](RESOLUTION-IMPLEMENTATION-STATUS.md)
for production corpus comparisons and validation.
