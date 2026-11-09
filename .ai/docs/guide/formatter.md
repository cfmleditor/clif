# Formatter

_The tree-sitter formatter (`internal/formatter`) and the guards around it. Moved out of CLAUDE.md unchanged._

## Formatter (`internal/formatter`)

Walks the tree-sitter CST. All rules (tag/attribute case, indentation, quotes, comma position,
SQL keyword casing, line width, attribute break threshold) come from `formatting` config. The
`whitespaceOnly` guard rejects any result that changes non-whitespace characters. `queryFormat`
is off by default — `<cfquery>` bodies are emitted verbatim unless opted in.

**Two entry points format a whole file and they must not drift**: the LSP's
`textDocument/formatting` handler (`internal/server/formatting.go`) and the `format` subcommand
(`cmd/clif/main.go`). Both go through the same two shared pieces, and new behaviour
belongs in them rather than at either call site:

- `config.ResolvedFormatting.FormatterOptions()` (`internal/config/formatting.go`) maps config
  onto `formatter.Options`, starting from `formatter.DefaultOptions()`. Zero-valued int fields
  mean "unset", so a partial `formatting` block keeps defaults for the rest. It deliberately
  leaves the three parse hooks nil, since those need `internal/language`; each caller installs
  them. When the CLI built its own option struct instead, `format -w` ignored every `formatting`
  key and emitted different bytes than the editor did for the same file.
- `formatter.ParseError(tree, src)` (`internal/formatter/parse_error.go`) refuses a file the
  grammar could not parse, so a CST gap can't be written back as deleted source. The
  `whitespaceOnly` guard is a second line of defence, not a substitute — it does not catch every
  shape of loss (see `.ai/docs/FORMATTER-ISSUES.md`).

**The formatter has three budgets, and a file past any of them is refused, not
formatted** (`formatter.go`). An argument list, a struct or an array that may go
multi-line renders its entries at two levels to decide, and a pair renders its
value twice, so the work doubled per level of nesting: a 160-byte file of calls
52 deep took 38 seconds, and the LSP runs a format on a goroutine it does not
wait for. `expr` is memoised by node, level and column (`exprKey`, the
generalisation of `closureKey`), which makes it polynomial;
`TestNestingCostIsPolynomial` counts renders at two depths. Polynomial is still
too much at absurd depths, so `treeTooDeep` refuses a tree deeper than
`maxTreeDepth` (1000; the corpus's deepest is 421, a long concatenation) or
nesting argument lists and literals past `maxLayoutNesting` (64; the corpus's
deepest is 11), checked on the document and on every sub-parse.
`maxFormatTime` (30s) is the backstop for whatever is still slow.
`TestTooDeepToFormatIsRefused`, `TestFormatGivesUpAtItsDeadline`. The corpus
formats byte-identically with all three in place.

`UseTabs` is the one setting with no config key: the LSP derives it from the editor's
`insertSpaces`, and the CLI keeps the `formatter.DefaultOptions()` value (tabs). A workspace
`indentWidth` outranks the editor's `tabSize`.

The `-w` write path (`writeFileInPlace`) writes a sibling temp file, fsyncs, restores the
replaced file's permission bits, re-checks that the file on disk still matches what was read,
and renames over the target — following a symlink to its target rather than replacing the link.
It follows a link only to a file in the link's own directory or beneath it, or beneath `--root`
(`symlinkRule`), unless `--follow-symlinks` is given: following every link let a cloned
repository commit `x.cfm` as a link to `~/.zshrc` and have formatting the tree rewrite it.
`TestFormatOneFileRefusesALinkOutOfItsDirectory`.
Unchanged output is not written at all, so formatting an already-clean file leaves its mtime
alone.
