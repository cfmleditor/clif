# Engine and extension ORM contracts

Research checkpoint, 2026-10-10. Historical audit; the native contract batch
implements the bounded session/factory producer and result rules described in
[the native follow-up](RESOLUTION-NATIVE-TYPE-RESEARCH.md#implementation-follow-up-and-compatibility-assumptions).

The subsequent [native type audit](RESOLUTION-NATIVE-TYPE-RESEARCH.md) examines
Object methods, collection/JDBC chains and report filtering that can hide gaps.

Engine-provided objects explain real unresolved calls. A CFC-only search cannot
find Hibernate session, factory, statistics, metadata, or query methods. This is
different from identifying an entity CFC and synthesizing its property methods.

## Contracts and existing coverage

- Adobe documents generated persistent-property getters and setters:
  [generated accessors](https://guides.adobe.com/coldfusion/en/docs/develop-coldfusion-applications/__references__/generated-accessors.html).
  clif already synthesizes accessors in `internal/parser/result.go` and
  relationship methods in `internal/parser/delegates.go`; entity-name lookup and
  literal entity-load collection handling also already exist. These are not new
  prospective gains.
- [Lucee ORMGetSession](https://docs.lucee.org/reference/functions/ormgetsession.html)
  requires the Hibernate extension and returns a native Hibernate Session. The
  documentation distinguishes Adobe's SessionWrapper from Lucee's direct return.
  clif's generated builtin documentation records only `object`, and there is no
  dedicated session/factory producer contract in the resolver.
- Hibernate's [Session](https://docs.hibernate.org/orm/5.6/javadocs/org/hibernate/Session.html)
  returns SessionStatistics from getStatistics and SessionFactory from
  getSessionFactory. Its
  [SessionFactory](https://docs.hibernate.org/orm/5.6/javadocs/org/hibernate/SessionFactory.html)
  returns a different Statistics interface from getStatistics.
- [SessionStatistics](https://docs.hibernate.org/orm/5.6/javadocs/org/hibernate/stat/SessionStatistics.html)
  defines getCollectionCount, getCollectionKeys, getEntityCount, and getEntityKeys.
  Factory [Statistics](https://docs.hibernate.org/orm/5.6/javadocs/org/hibernate/stat/Statistics.html)
  defines statistics enablement and session counters. Equal method spelling does
  not imply equal result type.

The 5.6 references establish these API contracts, not the installed Hibernate
version of every corpus application. Extension/version selection must be checked
before accepting deprecated or version-specific methods.

## Concrete checkpoint-30 findings

Reading the existing seven reports and their source identified ten unresolved
entries on explicit native ORM producer paths:

| Source | Entries | Producer and missing connections |
| --- | ---: | --- |
| ColdBox `tests/tmp/ormTest.cfm` | 7 | ormGetSession to getStatistics; four stats methods; ormGetSessionFactory to getStatistics to getSecondLevelCacheStatistics |
| Lucee `test/tickets/LDEV4339/ormSessionCheck.cfm` | 3 | ormGetSession to getSessionFactory to getStatistics, then setStatisticsEnabled/getSessionOpenCount/getSessionCloseCount |

These are a source-backed opportunity, not measured removals. They account for
about 0.10% of the 10,136 baseline entries. The last ColdBox cache-statistics API
needs engine/wrapper/version verification separately. Diagnostic line origins
are imperfect: consult source expressions rather than trusting line proximity.

Audit artifacts are under `target/resolution/engine-orm-audit/`:
`session-producer-files.json` contains 20 entries in files mentioning session
producers, only ten of which are the explicit paths above. A ten-line proximity
search (`near-native-producers.json`) found nine entries: eight native paths and
one HQL entity-loop entry. Neither search is a complete native-object census.
File co-occurrence and proximity are discovery aids, not receiver proof.

Lucee `LDEV6138/LDEV6138.cfm` also has an unresolved r.setName inside a loop over
ormExecuteQuery("FROM TestEntity WHERE name LIKE :prefix", ...). That is entity
collection typing from HQL, not a native session method. A future rule needs to
distinguish whole-entity selection from scalar, tuple, aggregate, constructor,
join, and dynamic-query results.

## Proposed approach

1. Represent native receiver identities separately from component file paths.
   Start with builtin producers that are not shadowed by application functions.
2. Supply engine/extension/version-aware interface contracts and propagate their
   declared result types across assignments and chains. Preserve SessionStatistics
   versus factory Statistics and Java collection results.
3. Resolve methods only on a proven native receiver; avoid globally whitelisting
   names such as save, getStatistics, or getEntityName. Existing broad member/Java
   acceptance can hide some calls, so audit resolved records too.
4. Add fail-before regressions, invalid-method negatives, shadowed-builtin cases,
   and wrong-statistics-interface cases before measuring per-entry corpus changes.
5. Extend the census to metadata, query objects, JDBC, Java dates, and native query
   objects using actual producers. The ten explicit ORM entries do not establish
   the potential of that broader category.

Mura beanORM and cborm service APIs still need their own source-backed framework
contracts. The presence of ORM terminology does not make arbitrary bean.save(),
loadBy(), or getIterator() methods engine-generated.

Required environment preflight passed after retry with approved network access;
the initial sandbox run could not resolve go.dev. This was a documentation and
report audit; no new corpus scan or runtime test was performed.
