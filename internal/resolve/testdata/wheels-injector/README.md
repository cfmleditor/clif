# Wheels Injector source contract

`Injector.cfc` is copied from the reviewed corpus file
`cfwheels_cfwheels/vendor/wheels/Injector.cfc` on 2026-10-10. It is framework
source, not a synthetic implementation. The contract's lexical fingerprint
is pinned by `TestWheelsInjectorStartup`; comments, whitespace and CFML case
normalization use the existing Wheels token comparison convention.

The complete source is retained so reviewers can inspect constructor effects,
registration, mutable mapping/cache behavior, helper calls, auto-wiring and
lifecycle callbacks. The implementation recognizes only this reviewed lexical
version. A different fingerprint requires a fresh source review, not an automatic
hash update. Returning the created object after calling init and onDIcomplete
is essential: neither callback's return value supplies the result type.

The new inference additionally requires two adjacent outer startup statements:
a literal Injector construction and its immediate literal getInstance lookup.
The binder must be a single public configure method, with no pseudo-constructor
effects or ancestry, containing only literal map/bind/mapInstance-to chains.
Duplicate aliases, flags, computed values, effects and container escape are
withheld. Later rebinding cannot change the identity of the already returned
object. This does not type an arbitrary later lookup on a mutable container.
