# One Go docs driver (#387): design and validation

`cmd/docpipe` is the only process that mutates docs sources, the pin, order and generated files. It holds the existing OS file lock for the whole operation. Keep Make entry points and thin compatibility shell launchers. Build the driver before taking the lock; do not pass the lock or proof tokens to children. Only git and curl are subprocesses; ZIP handling, source checks, swaps and writes are Go functions.

Move the generator without changing entry selection, refusal checks, allowed-removal comparison or serialization beyond provenance headers. Replace process exits in the generator with a typed unwind handled by the driver, and use one signal context. Cancellation stops external commands and is observed at phase boundaries; a synchronous filesystem commit and both generator renames finish before cleanup and signal exit. Killed processes leave recoverable directory backups; no child can continue writing shared docs paths.

Verify source pin/cache behavior, local fake fetches, nested zip handling and validation, rollback/recovery, exclusion of a second process, lock release on failure and interruption, generator refusals, exact generated data reproduction and pinned docs CI comparison. Run build, vet, short race tests and pinned lint. Publish a separate PR closing #387.

## Validation

- `make generate` and regeneration with `DOCS_BASELINE_REF=HEAD^1 DOCS_ALLOWED_REMOVALS_FILE=docs/allowed-removals` produced the same 1,144 entries and source manifest. Only the two generated provenance headers changed. No source pin or vendored JSON changed.
- Local-git and fake-download tests exercise pinned/cache fetches, nested archives, duplicate/traversal refusal, source validation, preserved notes, final-response ETags, bump ordering and rollback. The generator integration tests cover missing/falling-back entries, malformed retained files, missing or mismatched sources/baselines, and added-versus-leftover allowed removals.
- Real child-process tests verify writer exclusion, lock release after SIGKILL, and cancellation plus signal exit on SIGTERM. No child receives the lock or mutates shared docs paths.
- Overlay mutations disabling lock exclusion, backup recovery, and the removal guard each caused the corresponding behavioral regressions to fail. Repository sources stayed unchanged during these checks.
- Build, vet, the full short race suite, pinned lint and shell syntax checks passed. The Go driver and its tests cross-compile for windows/amd64 with CGO disabled; Windows execution was not tested locally.

## Remaining limits

A SIGKILL or power loss between two filesystem renames can still leave a missing source directory (recovered on the next run) or split generated files (detected by the existing consistency test). Cancellation is handled at phase/commit boundaries so synchronous commits finish while the same process still holds the lock. Killed downloads can leave isolated system-temp scratch; they cannot become a writer to the staged or committed docs after the lock is released. Launcher build artifacts are not pruned while a run may execute them; make clean removes them.
