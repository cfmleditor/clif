# docs/retained

Doc entries a source once published and then stopped publishing, kept so that a
regeneration does not drop them. This directory is committed, as are `docs/lucee/`
(Lucee's docs, vendored) and `docs/sources.lock` (the cfdocs commit to fetch); the rest
of `docs/` (the fetched cfdocs, the assembled `docs/data/`) is git-ignored.

`scripts/assemble-docs.sh` lists the sources in precedence order (cfdocs, then this
directory, then Lucee), and `scripts/generate_docs.go` takes each entry from the highest
source that documents it:

- **A retained entry outranks cfdocs.** When Lucee drops a function that cfdocs also
  documents (Adobe's `generateBCryptHash(plaintext, options)`), the Lucee signature
  stays, not Adobe's.
- **Lucee outranks a retained entry.** When docs.lucee.org publishes the name again, its
  entry wins, and the generator prints a note saying the retained file can be deleted.

Precedence is per entry, so a file's name does not matter; by convention it is the
lowercased function name. Each file has the same shape as a Lucee docs JSON file. One
that does not parse, or whose `type` is not exactly `function` or `tag`, makes
`make generate` refuse, rather than being skipped, and so do two files for one name.

## Why

In 2026-11 docs.lucee.org stopped listing about 50 functions from Lucee's crypto
extensions: `JwtSign`/`JwtVerify`, `BCryptHash`/`BCryptVerify`, `Argon2Hash`, the
TOTP, HOTP, Cbor, Pem, Kyber and Keystore families, and others. A regeneration
dropped every one of them. Where cfdocs documents the same name (`GenerateBCryptHash`,
`VerifySCryptHash`, and later `Argon2CheckHash` and `GenerateArgon2Hash`), Adobe's
entry replaced Lucee's instead. `docs.IsBuiltinFunction` then said no, and the unresolved
report flagged each call to them as a call to an undeclared function. These files
were rebuilt from the last generated docs that still held those entries.

## Adding to it

`make generate` refuses to write a file that is missing entries the committed docs have
at `HEAD`, or in which an entry that was Lucee's (or retained) would come from cfdocs
(`internal/docs/generated_sources.txt` records each entry's source), and lists them.
(`make build` never regenerates; it compiles the committed docs.) For each name it lists:

1. If a fetch failed, fix the fetch. Don't add anything here.
2. If the engine still has the function, recover its JSON (from an older Lucee docs
   zip, or rebuilt from the committed generated file) and put it here.
3. If the function really is gone, rerun `make generate` with
   `DOCS_ALLOW_REMOVALS="function foo"` (entry keys, comma-separated; `1` accepts
   every loss in the run, which can hide one nobody meant), and list it in
   `docs/allowed-removals`, which CI's comparison with the pull request's base reads.

`TestRetainedLuceeEntriesAreDocumented` (`internal/docs/retained_test.go`) fails when
an entry here does not reach the generated docs with its own syntax and parameters.
If docs.lucee.org publishes the name again, its entry wins, the generator and the test
both note that the retained file can be deleted, and nothing fails.
