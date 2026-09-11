# Installs git-commit-sentinel on Windows: downloads the matching release
# binary, verifies its checksum, and puts it on the user PATH. No Go
# toolchain, no gh CLI, no admin rights required.
#
# Usage:
#   irm https://raw.githubusercontent.com/diuis/git-commit-sentinel/main/scripts/install.ps1 | iex
#
# Env overrides:
#   $env:VERSION = "v0.0.1"          install a specific release (default: latest)
#   $env:INSTALL_DIR = "C:\some\dir" install location (default: $HOME\bin)

$ErrorActionPreference = "Stop"

$Repo = "diuis/git-commit-sentinel"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { Join-Path $HOME "bin" }

if (-not [Environment]::Is64BitOperatingSystem) {
    Write-Error "error: unsupported architecture (only 64-bit Windows is published)"
    exit 1
}
$Arch = "amd64"

$Version = $env:VERSION
if (-not $Version) {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
}
if (-not $Version) {
    Write-Error "error: could not resolve the latest release version"
    exit 1
}

$VersionNum = $Version.TrimStart("v")
$File = "git-commit-sentinel-$VersionNum-windows-$Arch.exe"
$BaseUrl = "https://github.com/$Repo/releases/download/$Version"

Write-Host "==> installing git-commit-sentinel $Version (windows/$Arch)"

$Tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([System.IO.Path]::GetRandomFileName()))
try {
    $ExePath = Join-Path $Tmp $File
    $SumsPath = Join-Path $Tmp "SHA256SUMS.txt"

    Invoke-WebRequest -Uri "$BaseUrl/$File" -OutFile $ExePath
    Invoke-WebRequest -Uri "$BaseUrl/SHA256SUMS.txt" -OutFile $SumsPath

    Write-Host "==> verifying checksum"
    $expected = (Select-String -Path $SumsPath -Pattern ([Regex]::Escape($File))).Line.Split(" ")[0]
    $actual = (Get-FileHash -Path $ExePath -Algorithm SHA256).Hash.ToLower()
    if ($expected -ne $actual) {
        Write-Error "error: checksum mismatch for $File (expected $expected, got $actual)"
        exit 1
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $Dest = Join-Path $InstallDir "git-commit-sentinel.exe"
    Move-Item -Force -Path $ExePath -Destination $Dest
    Write-Host "==> installed to $Dest"
}
finally {
    Remove-Item -Recurse -Force $Tmp
}

# Idempotent PATH check: only touch the persisted user PATH if InstallDir
# isn't already in it, so re-running this script never duplicates it.
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
$pathEntries = @()
if ($currentPath) { $pathEntries = $currentPath -split ";" }

if ($pathEntries -notcontains $InstallDir) {
    $newPath = if ($currentPath) { "$currentPath;$InstallDir" } else { $InstallDir }
    [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
    Write-Host "==> added $InstallDir to your user PATH — restart your terminal for it to take effect"
}
else {
    Write-Host "==> $InstallDir is already on your user PATH"
}
if ($env:Path -notlike "*$InstallDir*") {
    $env:Path += ";$InstallDir"
}

Write-Host ""
Write-Host "==> next: git-commit-sentinel setup"
