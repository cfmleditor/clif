# Mura template-param handoff: the chain analysis for the `dspObject_include` slice

Status as of 2026-10-19, after slices 9 (#281) and 10 (#283). Canon 11,233
findings on the seven scans; baseline `target/resolution/after-beanchain/`.

## The family

Mura's display-object templates receive their params through
`$.dspObject_include(theFile='<file>', <param>=<expr>, ...)`, defined at
`contentRenderer.cfc:1547` (`dspObject_Include`), whose include is computed:

```
var filePath = variables.$.siteConfig().lookupDisplayObjectFilePath(arguments.theFile)
<cfinclude template="#filepath#" />
```

A cfinclude inside a cffunction shares the function's ARGUMENTS scope and
variables scope, so the named args passed to `dspObject_include` ARE the
template's `arguments.<param>`; bare names (`content`) read the
function's variables scope.

Rows hanging on it (masa-c): `arguments.item` 128 (dsp_content_list 68,
dsp_meta_list 60), `content` 119, `feed` 66 — the configurators and dsp
templates.

## The chains

1. **dsp_content_list's `arguments.iterator`** — passed by
   `collection/layouts/list/index.cfm` as `iterator=iterator`, a bare local
   the module's `index.cfm` builds in branches: `$.getBean('contentManager')…`,
   `$.content().getRelatedContentIterator(...)`,
   `calendarUtility.getCalendarItems(..., returnFormat='iterator')`. The
   layout's own include is INTERPOLATED:
   `theFile='collection/layouts/#objectParams.layout#/index.cfm'` — the
   value provable from `objectParams.layout`'s cfparam default
   (`default="default"`, index.cfm:47) plus its branches.
2. **`arguments.item`** — `arguments.item=arguments.iterator.next()`
   (dsp_content_list.cfm:113). Needs `assignedFromCall` to read an
   `arguments.x` assignment (done in slice 9) and the iterator's
   `next()` element: contentIterator's `packageRecord` returns
   `variables.content = getBean("contentNavBean").set(item,this)` — the
   getBean locator rule names contentNavBean.
3. **The admin views are self-contained** (no context rule needed):
   `loadquickedit.cfm:45` builds `$=application.serviceFactory.getBean("MuraScope").init(...)`
   and `content=$.getBean("content").loadBy(...)`. After slice 10's
   `hopCallReturn` the chain's hops resolve, but the VARIABLE `content`
   still breaks at the parse's EMPTY ComponentRef (nearest preceding
   assignment) — an empty ref is a weak claim the deeper routes should
   sharpen, like $any is (`familyOverGuess`).

## Roots that remain deep

- `variables.feedManager` in feedBean.cfc is INJECTED by Mura's
  `beanFactory` via `setFeedManager` setters — the injection is not
  spelled per-file (no XML config in the corpus), so pass-through
  arguments (`arguments.feedBean` 104) refuse at the DI body.
- `settingsManager.getSite` returns `variables.sites['#siteid#']` — a
  cache the `setSites` setter fills with keyed stores whose keys are
  computed (`sites[bean.getSiteID()]=bean`).

## Candidate implementation order

1. ~~Empty-ref sharpening~~. Measured 2026-10-25, the premise does not
   hold: `receiverComponentD` already passes over an empty ref and reaches
   `assignedFromCall`. The admin views broke one hop later, at
   `contentBean.loadBy()`, which returns `contentManager.read()`'s
   cache-or-DAO bean and declares nothing. Done instead as a Mura rule,
   `resolve/mura_loadby.go`: a Mura bean's `loadBy()` returns the receiver
   (beanORM's only for `returnFormat` "self" or none). masa-c 3,145 →
   2,476 (679 removed, 10 added: 5 calls one hop past a newly typed
   variable, and 5 `oauthclient.generateToken().getToken()` that `$any`
   used to accept); the other six scans unchanged entry for entry.
2. `dspObject_include` recognition: the `$.`/`variables.$` receiver +
   the `theFile` literal → the file (workspace suffix match, like the
   view handoff) + the named args typed by the view-args machinery
   (`renderedValue`). The computed-layout case needs the cfparam default
   first.
3. The iterator element: contentIterator's `packageRecord` build
   (getBean locator + hopReturns' CallHop argument evaluation).
