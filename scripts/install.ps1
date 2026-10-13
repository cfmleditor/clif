# Install one clif release into a directory you own: no administrator rights,
# no package manager, the same binary on every machine that pins the same
# version. The Windows twin of scripts/install.sh.
#
#   powershell -ExecutionPolicy Bypass -File scripts\install.ps1 [-Version 0.5.0] [-Dir <dir>]
#
# -Version  0.5.0, v0.5.0 or latest. Left out, the first of clif.version and
#           scripts\clif.version in the working directory names it, so a
#           project, its developers and its CI install one version from one
#           file; with neither, latest.
# -Dir      where clif.exe goes: $env:CLIF_INSTALL_DIR, else
#           %LOCALAPPDATA%\clif\bin. It is not added to PATH; the script says
#           how when it is not there already.
#
# CLIF_DOWNLOAD_URL replaces https://github.com/cfmleditor/clif/releases, for a
# mirror laid out the same way. It must be https. The archive is checked
# against the release's checksums.txt, which every release from 0.5.1 on
# publishes: for those, and for latest, a missing checksums.txt stops the
# install; older releases are checked when they have one. When gh is installed
# and signed in, an archive from a release that publishes build provenance
# (after 0.5.3) is also checked with `gh attestation verify`; without gh that
# check is skipped. Releases from before the rename to clif publish
# cfmleditor-lsp archives only; one of those is installed as clif.exe. Windows
# on ARM runs the amd64 build.
[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$Dir = ""
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

# Windows PowerShell 5.1 on .NET Framework may still offer only TLS 1.0, which
# GitHub refuses.
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

if (-not $Version) {
    foreach ($f in @('clif.version', 'scripts\clif.version')) {
        if (Test-Path $f) {
            $Version = (Get-Content $f -Raw).Trim()
            break
        }
    }
}
if (-not $Version) { $Version = 'latest' }

if (-not $Dir) {
    $Dir = if ($env:CLIF_INSTALL_DIR) { $env:CLIF_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'clif\bin' }
}

$repo = 'cfmleditor/clif'
$firstChecksums = '0.5.1' # the first release with a checksums.txt
$firstAttested = '0.5.4'  # the first release built with provenance attestations

$releases = if ($env:CLIF_DOWNLOAD_URL) { $env:CLIF_DOWNLOAD_URL } else { 'https://github.com/cfmleditor/clif/releases' }
if (-not $releases.StartsWith('https://', [StringComparison]::OrdinalIgnoreCase)) {
    throw "clif install: CLIF_DOWNLOAD_URL must be an https:// URL, not $releases"
}
if ($Version -ne 'latest' -and $Version -notmatch '\Av?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?\z') {
    throw "clif install: '$Version' is not a version (expected 0.5.0, v0.5.0 or latest)"
}

# Whether a version's X.Y.Z is $min or later. A prerelease counts as its
# release, which is built by the same workflow.
function Test-AtLeast([string]$v, [string]$min) {
    $core = ($v.TrimStart('v') -split '[-+]')[0]
    return ([version]$core -ge [version]$min)
}

# gh is used only to verify provenance, and only when it can: installed and
# signed in (or given GH_TOKEN). Asked here so latest can be resolved to the
# version it is, which decides whether there is an attestation to check. A
# native command's stderr is an error record to Windows PowerShell, which
# 'Stop' would turn into a throw, so these run under 'Continue'.
$attest = $false
if (Get-Command gh -ErrorAction SilentlyContinue) {
    $eap = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & gh auth status *> $null
        if ($LASTEXITCODE -eq 0) {
            $attest = $true
            if ($Version -eq 'latest') {
                $tag = & gh release view --repo $repo --json tagName -q .tagName 2> $null
                if ($LASTEXITCODE -eq 0 -and $tag) { $Version = "$tag".Trim().TrimStart('v') }
            }
        }
    } catch {
        $attest = $false
    } finally {
        $ErrorActionPreference = $eap
        # A failed probe (gh installed but signed out) leaves its exit code
        # behind, and a caller that ends with `exit $LASTEXITCODE`, as a CI
        # step does, would report the install as failed.
        $global:LASTEXITCODE = 0
    }
}
if ($attest -and ($Version -eq 'latest' -or -not (Test-AtLeast $Version $firstAttested))) { $attest = $false }

$base = if ($Version -eq 'latest') { "$releases/latest/download" } else { "$releases/download/v$($Version.TrimStart('v'))" }

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("clif-install-" + [System.Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
    $archive = Join-Path $tmp 'archive.zip'
    $name = 'clif'
    try {
        Invoke-WebRequest -UseBasicParsing -Uri "$base/clif-windows-amd64.zip" -OutFile $archive
    } catch {
        $name = 'cfmleditor-lsp'
        try {
            Invoke-WebRequest -UseBasicParsing -Uri "$base/cfmleditor-lsp-windows-amd64.zip" -OutFile $archive
        } catch {
            throw "clif install: no windows/amd64 archive for $Version at $base"
        }
    }

    $asset = "$name-windows-amd64.zip"
    $sums = Join-Path $tmp 'checksums.txt'
    $haveSums = $true
    try {
        Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile $sums
    } catch {
        $haveSums = $false
    }

    if ($haveSums) {
        $want = ''
        foreach ($line in Get-Content $sums) {
            $parts = $line -split '\s+', 2
            if ($parts.Count -eq 2 -and ($parts[1] -eq $asset -or $parts[1] -eq "*$asset")) { $want = $parts[0].ToLower() }
        }
        $got = (Get-FileHash $archive -Algorithm SHA256).Hash.ToLower()
        if (-not $want -or $want -ne $got) {
            throw "clif install: $asset does not match the release's checksums.txt (want $want, got $got)"
        }
    } elseif ($Version -eq 'latest' -or (Test-AtLeast $Version $firstChecksums)) {
        # Every release since 0.5.1 has one, so its absence is a fault in the
        # download, not an old release: install nothing unverified.
        throw "clif install: could not fetch checksums.txt for $Version from $base, so $asset cannot be verified"
    } else {
        Write-Warning "clif install: $Version has no checksums.txt, so $asset was not verified"
    }

    if ($attest) {
        $eap = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        try {
            $out = & gh attestation verify $archive --repo $repo --signer-workflow "$repo/.github/workflows/release.yml" 2>&1
            $ok = ($LASTEXITCODE -eq 0)
        } finally {
            $ErrorActionPreference = $eap
        }
        if (-not $ok) { throw "clif install: $asset has no valid build provenance from ${repo}: $out" }
        Write-Output "clif install: verified $asset's build provenance with gh attestation verify"
    }

    Expand-Archive -Path $archive -DestinationPath $tmp -Force
    New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    $exe = Join-Path $Dir 'clif.exe'
    Copy-Item (Join-Path $tmp "$name.exe") $exe -Force

    $installed = & $exe version
    Write-Output "Installed $installed to $exe"

    $onPath = ($env:PATH -split ';') | Where-Object { $_.TrimEnd('\') -ieq $Dir.TrimEnd('\') }
    if (-not $onPath) {
        Write-Warning "clif install: $Dir is not on PATH. For this user, from now on: [Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';$Dir', 'User')"
    }
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
