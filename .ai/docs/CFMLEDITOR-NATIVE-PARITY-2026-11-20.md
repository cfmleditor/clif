# CFMLEditor native-provider parity

The comparison uses CFMLEditor 0.7.8 at
[12585b1](https://github.com/cfmleditor/cfmleditor/tree/12585b1ce60cb960641deb2e985664ffb029941e)
and clif at `2e7161b`. CFMLEditor suspends its native providers when the LSP
takes ownership, so differences affect users who enable it.
[Native provider registration](https://github.com/cfmleditor/cfmleditor/blob/12585b1ce60cb960641deb2e985664ffb029941e/src/cfmlMain.ts#L84)

The baseline passed all 41 cases in `make conformance`, and the upstream
workspace fixtures matched the copies in clif. That run established which file
each definition lookup could reach. The native tests also assert the exact
declaration for several cases. The remaining evidence comes from reading the
native providers and sending requests to clif over stdio. A live VS Code
comparison is pending.
[Native definition tests](https://github.com/cfmleditor/cfmleditor/blob/12585b1ce60cb960641deb2e985664ffb029941e/src/test/CFMLDefinitionProvider.test.ts) ·
[clif definition replay](../../internal/server/definition_conformance_test.go)

| Area | Behavior and test cases | Status |
| --- | --- | --- |
| Go to Definition | Assert the exact declaration URI and identifier range for arguments, scoped variables, `cfparam`, `cfloop`, and shared scopes. Extend the same assertions to tag and script methods and user functions. A baseline method lookup returned an empty range at column zero. [Native target assertions](https://github.com/cfmleditor/cfmleditor/blob/12585b1ce60cb960641deb2e985664ffb029941e/src/test/CFMLDefinitionProvider.test.ts#L205) | Exact ranges are now checked for all 16 variable cases and all 5 method and user-function cases, including multiline headers and closed-file lookup. |
| Signature help | Select named arguments by name and handle multiline calls, quoted commas, comments, and nested literals. Test unfinished input over LSP, where the cursor is placed before the user has finished typing the call. [Native signature provider](https://github.com/cfmleditor/cfmleditor/blob/12585b1ce60cb960641deb2e985664ffb029941e/src/features/signatureHelpProvider.ts#L46) | Implemented in this pass. The workflow suite covers the reproduced failures and editing edge cases. |
| CFC outline | Return a class or interface with property and method children and meaningful selection ranges. Test script and tag components, an interface, and the constructor kind. The baseline returned a flat list of functions and synthetic accessors with zero-width ranges. [Native document-symbol provider](https://github.com/cfmleditor/cfmleditor/blob/12585b1ce60cb960641deb2e985664ffb029941e/src/features/documentSymbolProvider.ts#L50) | Implemented. Returns Class/Interface containers with Property, Constructor, and Method children, exact ranges, and excludes synthetic accessors; `.cfm` templates return flat Function symbols. |
| CSS color swatches and picker | Implement `documentColor` and `colorPresentation`. Assert literal ranges and RGBA values in CSS, and verify that a picker edit writes `##ff0000` inside `<cfoutput>` and `#ff0000` outside it. The baseline advertised neither method and both requests returned JSON-RPC method-not-found. [Native color provider](https://github.com/cfmleditor/cfmleditor/blob/12585b1ce60cb960641deb2e985664ffb029941e/src/features/colorProvider.ts#L21) | Implemented. Advertises `colorProvider` and serves `documentColor` (hex, `rgb`/`rgba`, `hsl`/`hsla` in `<style>` and `style=`) and `colorPresentation` (writing `##` in evaluated output contexts and `#` outside). |
| Go to Type Definition | Compare native and LSP behavior for a typed parameter with default settings. The native provider is registered by default. clif supports the request behind `features.typeDefinition`, and the extension's initialization options carry formatting settings. [Native registration](https://github.com/cfmleditor/cfmleditor/blob/12585b1ce60cb960641deb2e985664ffb029941e/src/cfmlMain.ts#L90) · [Extension initialization options](https://github.com/cfmleditor/cfmleditor/blob/12585b1ce60cb960641deb2e985664ffb029941e/src/lsp/cfmlLspClient.ts#L356) | Default capability decision pending. |

Method and user-function definitions now retain their declared identifier's
UTF-16 range during parsing (including multiline headers, function values,
nested script functions, and mid-line regions) and serve it directly from the
indexed `FunctionDef` for closed-file lookup without re-reading the target file.
