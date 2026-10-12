# Execution context and resolution evidence follow-up

Subsequent user-authorized experiment: [Speculative connections](RESOLUTION-SPECULATIVE-CONNECTIONS.md)
measures unreliable caller and candidate links. Its strongest isolated scenario
reduces diagnostic entries by 36.93%, with demonstrated counterexamples. It does
not change the production baseline or validate those connections.

10 October 2026. Research after the ten-pass review. Production behavior remains
checkpoint 30: **10,136 entries**. No shipped code, presets, tests or corpus
configuration changed. Artifacts are under
`target/resolution/admin-context-followup/`.

## What the 360-entry admin scenario actually establishes

Audited one representative call per file/receiver/function group among the
previous scenario's 362 removals: 61 groups. The traces classify them as follows.

| Representative resolution route | Groups | Entries represented |
| --- | ---: | ---: |
| Ordinary method definition | 43 | 334 |
| Receiver found, method accepted through onMissingMethod | 16 | 26 |
| Still unresolved, with a replacement reason | 2 | 2 |

The 334 include all 253 rbKey and 37 buildURL entries. The remaining definition
groups include iterator operations and declared bean methods. The 26 dynamic
entries are bean getter calls, mostly on changeset assignment items; two are
comment-item getters. This classification follows representative explanations,
not runtime execution or a separate explanation for every occurrence. Multiple
same-named calls on one line can share a trace output; full original findings
are retained beside each representative in `audited-groups.json`.

Representative receiver paths are supported by source:

- cchangesets.assignments writes rc.siteAssignments from
  changesetManager.getAssignmentsIterator (:125). That method constructs the
  contentIterator (:563–571); queryIterator.next delegates to packageRecord, and
  contentIterator.packageRecord constructs/updates contentNavBean (:61–72).
- ccomments.loadcomments writes rc.itComments from getCommentsIterator (:215).
  contentCommentManager.getCommentsIterator constructs contentCommentIterator
  (:224–231). Its packageRecord obtains a comment bean and returns it (:58–70).
- The views assign their item from those iterators before calling its methods.
  The scenario activates the existing FW/1 controller-to-view relationship;
  it does not invent a type from the variable name item.

The two replacement reasons remain getParent/getDisplayInterval return gaps in
dsp_displaycontent. The one additional checked record is local.item.getName at
loadcomments:280. These coverage differences were retained in the ten-pass report.

**Interpretation:** net 360 fewer diagnostics is measured. Stronger language such
as “360 methods proven valid” is not supported. The distinction between locating
a receiver, locating a method, and permitting dynamic dispatch matters.

## A concrete boundary in the current dynamic-dispatch policy

`resolve.go` accepts a method when its component defines onMissingMethod, even
without a declaration for that method. A research-only overlay probe creates a
component whose onMissingMethod unconditionally throws and calls notSupported on
it. The parser extracts the call and the resolver reports an empty reason.
This confirms the current dispatch policy without executing CFML.

Mura bean.OnMissingMethod is also conditional: a registry entry can dispatch an
expression, get/set prefixes take fallback paths, and other names throw. Earlier
[Mura registry research](MURA-DYNAMIC-RELATIONSHIPS.md) covers those registrations
and lifetimes. The admin dynamic getters fit a plausible dispatch family, but
their precise stored property/registry value is not established by merely finding
the handler.

Planning consequence: preserve the existing TargetDynamic distinction in any
future resolution audit or report. A contextual improvement can locate the right
bean while still relying on dynamic acceptance. Do not silently count that as a
new declared-method proof, or tighten all dispatch handling without reviewing
framework-specific contracts and intentionally dynamic code.

## Why embedded framework context must retain its factory owner

The source supplies a more precise boundary than “Mura also uses FW/1.”
admin/Application.cfc enables subsystems, sets the default subsystem to core,
sets siteWideLayoutSubsystem to common, and names diEngine as mura (:119–127).
It calls `setBeanFactory(application.serviceFactory)` at :166. The embedded
framework's getBeanFactory supports subsystem/default factory selection (:209
onward). Thus FW/1 rendering ownership does not establish a DI/1 factory.

The earlier resolver-only experiment's regressions are consistent with this
distinction. Its paired settingsManager explanation shows lost startup-derived
identity after adding generic rules; it still does not attribute every regression
to a single rule.

A prospective automatic rule should establish these facts independently:

1. Which Application and embedded framework own the rendering path.
2. Which subsystem's controllers supply a view's request context.
3. Which object supplies the factory, with its existing Mura startup policy.
4. Which explicit configuration, nested application or custom rendering setup
   overrides those defaults.

Use the physical governing Application/ancestry and verified rendering contract,
not a globally enabled preset or an `/admin/` substring alone. The framework base
URL can change using the configured admin directory (:140); that is not proof
that a hardcoded physical directory detector is general. Existing caller-context
mapping logic should remain authoritative when a library is entered from another
application. Matching view/buildURL declarations is a current safeguard, but a
custom component can declare those names too; rendering-chain provenance remains
an open detection obligation.

## The remaining save origins are a repeated population

A logging-only overlay enumerates matching save records from the 85 lexical
candidate files before the normal inference's early veto. It observes 178 unique
file/line/span/omission tuples, of which 154 prove omission and 11 have no span.
This is a location census, not 178 distinct receiver owners or exact byte-level
call identities: same-line records can collapse in that tuple key.

| Spanless save locations, one-based source lines | Visible source |
| --- | --- |
| categoryManager:274,725 | obj.save() |
| contentCommentBean:504 | obj.save() |
| extendManager:1881,1899,1924,1935,1980,1985 | subtype, extend-set, attribute, related-set, image-size and site save() |
| userManager:485,627 | obj.save() |

All eleven visibly omit arguments. The earlier live trace proves the exact
emitter at categoryManager:274: the script parser's top-level named-call branch
consumes parentheses before origin capture. The other locations share the source
shape, but were not individually instrumented to establish the identical emitter.

This changes the implementation plan from one additional example to a family:
test the seeded script-island dispatcher, survey every emission after consuming
arguments, then rerun the complete competing-owner inference. The census itself
does not resolve owners or identify supplied values. The 65 changeset argument
entries remain unresolved, and no gain is credited for repairing these spans.

## Record support has an explicit merge and alias boundary

The existing producerValue can hold fields, but producerMerge preserves component
alternatives, literals and limited instance identity without merging the fields
map. A small research overlay exercises current behavior: merging the same
component retains A; merging the same record with a bean field drops the fields.
Unknown alternatives still produce unknown. This is an observed representation
boundary, not a corpus attribution or a tested improvement.

Assignments also conservatively reject aliasing record values and invalidate
parent records on member writes. Existing
TestAliasedPrivateRegistrationTablesWithholdState deliberately withholds identity
when two names share mutable registration tables. Keeping fields through joins
without modeling alias/write effects could violate that safety boundary.

ColdBox's results.ehBean explanation still has no receiver. Moreover,
assignedFromCall explicitly rejects non-special dotted targets such as
results.ehBean; it supports rc/prc and simple variables. The relevant source is
also a multiline fluent assignment, with multiple later replacements and a
cache-returning producer. Thus a producerMerge change alone is not established
as a fix for these 25 entries. The 26 results.config entries have additional
dynamic module/mixin obligations.

A syntax inventory finds 546 unknown-receiver entries in 188 groups whose
receiver has one dot and whose root is outside the usual CFML scopes. This broad
population includes native values, test expectation objects, dynamic application
indexes and other non-records. It is a search population, **not 546 fixable record
findings**. A safe record plan needs field origins, branch joins, alias identity,
overwrite/escape handling, multiline expressions and producer contracts together.

## Revised next proof steps

The scoped admin context has the strongest measured conditional result, with
receiver/definition/dispatch evidence now separated. The parser-origin family
has eleven observed omission blockers and one proven emitting path. Record
inference has concrete implementation boundaries but no measured corpus gain.

Before implementation, create focused fail-before cases for the proposed context
and parser paths, retain counterexamples for custom factories and rendering
owners, and measure each change against the same checkpoint and coverage. For
records, first trace a complete producer-to-field consumer rather than changing
the merge function in isolation. The current evidence still does not support a
30% additive gains claim.

Validation: required preflight passes with approved network access after sandbox
DNS failure. Research binaries compile; two observation probes pass through an
overlay test file that is absent from production. All 61 representative admin
explanations and the save census are retained locally. No runtime CFML result,
new full-suite validation or published CI pass is claimed.
