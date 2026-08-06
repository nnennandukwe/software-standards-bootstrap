<#
.SYNOPSIS
Install Software Standards Bootstrap from a published GitHub release.

.DESCRIPTION
The Windows counterpart of install.sh. It downloads the release archive for the
current architecture, verifies its SHA-256 checksum against the published
checksums.txt, and installs the ssb binary. It never replaces a developer-owned
Agent Skill and never leaves a partially installed binary behind.

.PARAMETER Version
Release tag to install, such as v0.1.0. Defaults to the latest published
release. Also read from the SSB_VERSION environment variable.

.PARAMETER InstallDir
Destination for the ssb binary. Defaults to $HOME\.local\bin. Also read from
the SSB_INSTALL_DIR environment variable.

.PARAMETER SkillDir
Also install the Agent Skill from the same release tag beneath .agents\skills
or .claude\skills. Also read from the SSB_SKILL_DIR environment variable.

.PARAMETER Help
Show usage and exit.

.EXAMPLE
.\install.ps1

.EXAMPLE
.\install.ps1 -Version v0.1.0 -SkillDir .claude\skills
#>

#Requires -Version 5.0

[CmdletBinding(PositionalBinding = $false)]
param(
    # Every parameter carries an explicit attribute: without them, adding the
    # remaining-arguments parameter makes Windows PowerShell fail to resolve a
    # parameter set once several named options are supplied together.
    [Parameter()][string]$Version,
    [Parameter()][string]$InstallDir,
    [Parameter()][string]$SkillDir,
    [Parameter()][switch]$Help,
    [Parameter(ValueFromRemainingArguments = $true)][string[]]$RemainingArguments
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$repository = 'nnennandukwe/software-standards-bootstrap'
$releaseBaseUrl = "https://github.com/$repository/releases"

$tempDir = $null
$targetTemp = $null
$skillTemp = $null
$skillTarget = $null
$skillInstalled = $false
$exitCode = 0

function Show-Usage {
    Write-Output @'
Install Software Standards Bootstrap from a published GitHub release.

Usage:
  install.ps1 [-Version VERSION] [-InstallDir DIRECTORY] [-SkillDir DIRECTORY]

Options:
  -Version VERSION        Release tag to install, such as v0.1.0.
                          Defaults to the latest published release.
  -InstallDir DIRECTORY   Destination for the ssb binary.
                          Defaults to $HOME\.local\bin.
  -SkillDir DIRECTORY     Also install the Agent Skill from the same release
                          tag beneath .agents\skills or .claude\skills.
  -Help                   Show this help.

Environment variables:
  SSB_VERSION             Alternative to -Version.
  SSB_INSTALL_DIR         Alternative to -InstallDir.
  SSB_SKILL_DIR           Alternative to -SkillDir.
'@
}

# Test-Path reports $false for a dangling symlink or junction, so a broken link
# left by a developer would slip past an overwrite guard. install.sh covers this
# with its explicit [ -L ] test; enumerating the parent directory is the
# equivalent here, because a broken link still owns its directory entry.
function Test-EntryExists {
    param([string]$Path)
    if (Test-Path -LiteralPath $Path) {
        return $true
    }
    # Split-Path -LiteralPath cannot be combined with -Parent or -Leaf, and
    # -Path would expand wildcards in a path we must treat literally.
    $parent = [System.IO.Path]::GetDirectoryName($Path)
    $leaf = [System.IO.Path]::GetFileName($Path)
    if (-not $parent -or -not (Test-Path -LiteralPath $parent -PathType Container)) {
        return $false
    }
    $entry = Get-ChildItem -LiteralPath $parent -Force -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -eq $leaf } |
        Select-Object -First 1
    return [bool]$entry
}

function Remove-IfPresent {
    param([string]$Path)
    if ($Path -and (Test-EntryExists $Path)) {
        Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Invoke-Cleanup {
    Remove-IfPresent $script:targetTemp
    Remove-IfPresent $script:skillTemp
    if ($script:skillInstalled) {
        Remove-IfPresent $script:skillTarget
    }
    Remove-IfPresent $script:tempDir
}

function Get-RequiredCommand {
    param([string]$Name)
    # PATH can hold several matches; the first one is what the shell would run.
    $command = Get-Command $Name -CommandType Application -ErrorAction SilentlyContinue |
        Select-Object -First 1
    if (-not $command) {
        throw "required command not found: $Name"
    }
    return $command.Source
}

# Native stderr is wrapped in an ErrorRecord under $ErrorActionPreference='Stop',
# so a harmless warning from a working binary would terminate the script. Exit
# status is the signal we actually want here.
function Invoke-NativeCommand {
    param(
        [string]$FilePath,
        [string[]]$CommandArguments
    )
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        & $FilePath @CommandArguments 2>&1 | Out-Null
        return $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previous
    }
}

try {
    if ($Help) {
        Show-Usage
        exit 0
    }

    if ($RemainingArguments) {
        throw "unknown option: $($RemainingArguments[0])"
    }

    if (-not $Version) { $Version = $env:SSB_VERSION }
    if (-not $InstallDir) { $InstallDir = $env:SSB_INSTALL_DIR }
    if (-not $SkillDir) { $SkillDir = $env:SSB_SKILL_DIR }

    $platform = 'Win32NT'
    if ($PSVersionTable.PSObject.Properties['Platform']) {
        $platform = $PSVersionTable.Platform
    }
    if ($platform -ne 'Win32NT') {
        throw "Unsupported operating system: $platform. install.ps1 supports Windows; use install.sh on macOS and Linux."
    }

    # Architecture is resolved before any network access so an unsupported
    # machine fails without downloading anything.
    $architecture = $env:PROCESSOR_ARCHITECTURE
    if ($env:PROCESSOR_ARCHITEW6432) { $architecture = $env:PROCESSOR_ARCHITEW6432 }
    switch ($architecture) {
        'AMD64' { $releaseArch = 'amd64' }
        'x86_64' { $releaseArch = 'amd64' }
        'ARM64' { $releaseArch = 'arm64' }
        default {
            throw "Unsupported architecture: $architecture. install.ps1 supports amd64 and arm64."
        }
    }

    $curlPath = Get-RequiredCommand 'curl.exe'

    if (-not $InstallDir) {
        $homeDirectory = $env:USERPROFILE
        if (-not $homeDirectory) { $homeDirectory = $HOME }
        if (-not $homeDirectory) {
            throw 'USERPROFILE is not set; pass -InstallDir DIRECTORY'
        }
        $InstallDir = Join-Path $homeDirectory '.local\bin'
    }

    $target = Join-Path $InstallDir 'ssb.exe'
    if (Test-Path -LiteralPath $target -PathType Container) {
        throw "install target is a directory: $target"
    }

    if ($SkillDir) {
        $skillTarget = Join-Path $SkillDir 'software-standards-bootstrap'
        if (Test-EntryExists $skillTarget) {
            throw "Agent Skill already exists at $skillTarget; review or remove it before installing"
        }
        $gitPath = Get-RequiredCommand 'git.exe'
    }

    if (-not $Version) {
        $latestUrl = & $curlPath -fsSL -o NUL -w '%{url_effective}' "$releaseBaseUrl/latest"
        if ($LASTEXITCODE -ne 0 -or -not $latestUrl) {
            throw 'could not resolve the latest published release'
        }
        $Version = ($latestUrl.TrimEnd('/') -split '/')[-1]
    }

    # \z rather than $, because .NET's $ also matches before a trailing newline.
    if ($Version -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+[A-Za-z0-9._+-]*\z') {
        throw "invalid release tag: $Version. Expected a tag such as v0.1.0."
    }

    $asset = "ssb_${Version}_windows_${releaseArch}.zip"
    $downloadBase = "$releaseBaseUrl/download/$Version"

    $tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("ssb-install." + [System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    $archivePath = Join-Path $tempDir $asset
    $checksumsPath = Join-Path $tempDir 'checksums.txt'

    & $curlPath -fsSL "$downloadBase/$asset" -o $archivePath
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $archivePath)) {
        throw "could not download $asset from release $Version"
    }
    & $curlPath -fsSL "$downloadBase/checksums.txt" -o $checksumsPath
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $checksumsPath)) {
        throw "could not download checksums.txt from release $Version"
    }

    # @() keeps a one-token or empty line from collapsing to a scalar or $null,
    # which Set-StrictMode would then reject for lacking a Count property.
    $expected = @(
        Get-Content -LiteralPath $checksumsPath | ForEach-Object {
            $fields = @($_ -split '\s+' | Where-Object { $_ })
            if ($fields.Count -ge 2 -and $fields[1] -eq $asset) { $fields[0] }
        }
    )
    if ($expected.Count -ne 1) {
        throw "checksums.txt does not contain exactly one entry for $asset"
    }

    $actual = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash
    # PowerShell string -ne is case-insensitive, which is the comparison we want.
    if ($actual -ne $expected[0]) {
        throw "checksum verification failed for $asset"
    }

    $extractDir = Join-Path $tempDir 'extract'
    try {
        Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force
    } catch {
        throw "could not extract ssb.exe from $asset"
    }
    $extracted = Join-Path $extractDir 'ssb.exe'
    if (-not (Test-Path -LiteralPath $extracted -PathType Leaf)) {
        throw "$asset does not contain the ssb.exe binary"
    }

    if ((Invoke-NativeCommand $extracted @('--help')) -ne 0) {
        throw "downloaded ssb.exe could not run on windows/$releaseArch"
    }

    if ($SkillDir) {
        $sourceCheckout = Join-Path $tempDir 'source'
        $cloneExit = Invoke-NativeCommand $gitPath @(
            '-c', 'advice.detachedHead=false', 'clone', '--quiet', '--depth', '1',
            '--branch', $Version, '--single-branch',
            "https://github.com/$repository.git", $sourceCheckout
        )
        if ($cloneExit -ne 0) {
            throw "could not download the Agent Skill from release tag $Version"
        }
        $skillSource = Join-Path $sourceCheckout 'skills\software-standards-bootstrap'
        if (-not (Test-Path -LiteralPath (Join-Path $skillSource 'SKILL.md') -PathType Leaf)) {
            throw "release tag $Version does not contain the software-standards-bootstrap Agent Skill"
        }
        New-Item -ItemType Directory -Path $SkillDir -Force | Out-Null
        $skillTemp = Join-Path $SkillDir ('.software-standards-bootstrap.install.' + [System.IO.Path]::GetRandomFileName())
        New-Item -ItemType Directory -Path $skillTemp -Force | Out-Null
        # -Force is load-bearing: without it Copy-Item skips hidden entries.
        Copy-Item -Path (Join-Path $skillSource '*') -Destination $skillTemp -Recurse -Force
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    if (Test-Path -LiteralPath $target -PathType Container) {
        throw "install target is a directory: $target"
    }
    $targetTemp = Join-Path $InstallDir ('.ssb-install.' + [System.IO.Path]::GetRandomFileName())
    Copy-Item -LiteralPath $extracted -Destination $targetTemp -Force

    if ($SkillDir) {
        if (Test-EntryExists $skillTarget) {
            throw "Agent Skill appeared during installation at $skillTarget; no files were replaced"
        }
        # Directory.Move throws when the destination exists. Move-Item -Force
        # would nest the source inside it and report success instead.
        [System.IO.Directory]::Move($skillTemp, $skillTarget)
        $skillTemp = $null
        $skillInstalled = $true
    }

    # File.Move has no overwrite overload on .NET Framework, and deleting the
    # target first would leave no ssb.exe at all if the move then failed.
    # File.Replace swaps the directory entry in one step; it needs an existing
    # destination on the same volume, which $targetTemp is by construction. It
    # also succeeds while the old binary is running, where Delete fails
    # outright. A fresh install has nothing to replace and nothing to lose, so
    # it moves directly.
    if (Test-Path -LiteralPath $target -PathType Leaf) {
        [System.IO.File]::Replace($targetTemp, $target, [NullString]::Value)
    } else {
        [System.IO.File]::Move($targetTemp, $target)
    }
    $targetTemp = $null
    $skillInstalled = $false

    Write-Output "Installed ssb $Version to $target"
    if ($skillTarget) {
        Write-Output "Installed Agent Skill to $skillTarget"
    }

    $pathEntries = @()
    if ($env:PATH) { $pathEntries = @($env:PATH -split ';' | Where-Object { $_ }) }
    if ($pathEntries -notcontains $InstallDir) {
        Write-Output ''
        Write-Output 'Add ssb to your PATH for this shell:'
        Write-Output "  `$env:PATH = `"$InstallDir;`$env:PATH`""
    }
} catch {
    [Console]::Error.WriteLine("install.ps1: $($_.Exception.Message)")
    $exitCode = 1
} finally {
    # Runs on success, on a thrown failure, and on Ctrl-C, matching install.sh's
    # trap cleanup 0 HUP INT TERM. A completed install has already cleared
    # $targetTemp and $skillTemp and reset $skillInstalled, so on the success
    # path this removes the download directory and nothing else.
    Invoke-Cleanup
}

exit $exitCode
