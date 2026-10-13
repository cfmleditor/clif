# clif.exe is embedded in this package (tools\clif.exe), put there by the
# release workflow (package-managers.yml) from the release's
# clif-windows-amd64.zip; legal\VERIFICATION.txt says how to check it. Nothing
# is downloaded, so an install needs no access to GitHub.
#
# Only an amd64 build is published. Windows on ARM runs it under emulation.
$ErrorActionPreference = 'Stop'
$toolsDir = Split-Path -Parent $MyInvocation.MyCommand.Definition

# Chocolatey shims clif.exe on its own. The name before the rename is shimmed
# too, for editor extensions that still look for cfmleditor-lsp on PATH.
Install-BinFile -Name 'cfmleditor-lsp' -Path (Join-Path $toolsDir 'clif.exe')
