# Installs git-commit-sentinel on Windows: downloads the matching release
# binary, verifies its checksum, and puts it on the user PATH. No Go
# toolchain, no gh CLI, no admin rights required.
#
# Usage:
#   irm https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/install.ps1 | iex
#
# Env overrides:
#   $env:GIT_COMMIT_SENTINEL_VERSION = "v0.0.1"          install a specific release (default: latest)
#   $env:GIT_COMMIT_SENTINEL_INSTALL_DIR = "C:\some\dir" install location (default: $HOME\bin)
#
# Keep this file ASCII-only: Windows PowerShell 5.1 reads it as
# Windows-1252, where UTF-8 punctuation can turn into a closing quote.

# `irm | iex` runs this in the caller's session: the script block keeps
# $ErrorActionPreference and every variable below out of it. $env:Path
# is process-wide, so the update at the end still reaches the session.
& {
    $ErrorActionPreference = "Stop"
    # Windows PowerShell 5.1 redraws the progress bar per chunk, slowing downloads.
    $ProgressPreference = "SilentlyContinue"

    $Repo = "blue-noah/git-commit-sentinel"
    $InstallDir = if ($env:GIT_COMMIT_SENTINEL_INSTALL_DIR) { $env:GIT_COMMIT_SENTINEL_INSTALL_DIR } else { Join-Path $HOME "bin" }

    # Win32_Processor reports the hardware architecture even when this
    # PowerShell runs under x64 emulation on ARM64, where
    # $env:PROCESSOR_ARCHITECTURE would say AMD64.
    $cpuArch = (Get-CimInstance -ClassName Win32_Processor | Select-Object -First 1).Architecture
    switch ($cpuArch) {
        9 { $Arch = "amd64" }
        12 {
            throw ("error: Windows on ARM is not supported.`n" +
                "git-commit-sentinel is published for 64-bit Intel/AMD Windows (x64) only,`n" +
                "so there is no binary to install on this machine.")
        }
        default {
            throw ("error: unsupported processor architecture (Win32_Processor.Architecture = $cpuArch).`n" +
                "git-commit-sentinel is published for 64-bit Intel/AMD Windows (x64) only.")
        }
    }

    # Only Windows PowerShell 5.1 downloads through ServicePointManager, which
    # can be pinned to protocols GitHub refuses (0 = SystemDefault: Windows
    # chooses). The fix is left to the user so their session isn't changed.
    if ($PSVersionTable.PSEdition -eq "Desktop") {
        $protocols = [Net.ServicePointManager]::SecurityProtocol
        # 3072 = Tls12, 12288 = Tls13 (numeric: older .NET lacks the Tls13 name).
        if ([int]$protocols -ne 0 -and -not ([int]$protocols -band (3072 -bor 12288))) {
            throw ("error: TLS 1.2 or newer, which GitHub requires, is not enabled in this PowerShell session (enabled: $protocols).`n" +
                "Enable it for this session, then run the installer again:`n" +
                "  [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12")
        }
    }

    # -UseBasicParsing: without it Windows PowerShell 5.1 needs the Internet
    # Explorer engine, missing on Server Core. PowerShell 7 ignores it.
    $Version = $env:GIT_COMMIT_SENTINEL_VERSION
    if (-not $Version) {
        try {
            $release = Invoke-RestMethod -UseBasicParsing -Uri "https://api.github.com/repos/$Repo/releases/latest"
        }
        catch {
            throw ("error: could not resolve the latest release version: $($_.Exception.Message)`n" +
                "To skip this lookup, pick a release from https://github.com/$Repo/releases`n" +
                "and set it before running the installer again, e.g. `$env:GIT_COMMIT_SENTINEL_VERSION = `"v1.2.3`"")
        }
        $Version = $release.tag_name
    }
    if (-not $Version) {
        throw "error: could not resolve the latest release version"
    }

    $VersionNum = $Version.TrimStart("v")
    $Version = "v$VersionNum"
    $File = "git-commit-sentinel-$VersionNum-windows-$Arch.exe"
    $BaseUrl = "https://github.com/$Repo/releases/download/$Version"

    Write-Host "==> installing git-commit-sentinel $Version (windows/$Arch)"

    $Tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ([System.IO.Path]::GetRandomFileName()))
    try {
        $ExePath = Join-Path $Tmp $File
        $SumsPath = Join-Path $Tmp "SHA256SUMS.txt"

        try {
            Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/$File" -OutFile $ExePath
            Invoke-WebRequest -UseBasicParsing -Uri "$BaseUrl/SHA256SUMS.txt" -OutFile $SumsPath
        }
        catch {
            throw "error: could not download $File from release $Version (does that release exist?): $($_.Exception.Message)"
        }

        Write-Host "==> verifying checksum"
        $entry = Select-String -Path $SumsPath -Pattern ([Regex]::Escape($File)) | Select-Object -First 1
        if (-not $entry) {
            throw "error: SHA256SUMS.txt in release $Version has no checksum for $File"
        }
        $expected = $entry.Line.Split(" ")[0]
        $actual = (Get-FileHash -Path $ExePath -Algorithm SHA256).Hash.ToLower()
        if ($expected -ne $actual) {
            throw "error: checksum mismatch for $File (expected $expected, got $actual)"
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
    $normalize = { param($entry) [Environment]::ExpandEnvironmentVariables($entry).TrimEnd("\") }
    $target = & $normalize $InstallDir

    # The user PATH is read and written as the raw registry value, not via
    # [Environment]::Get/SetEnvironmentVariable: those return it with %VAR%
    # references expanded and write it back as plain text, which would
    # freeze entries like %JAVA_HOME%\bin to today's values.
    $envKey = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey("Environment")
    try {
        $currentPath = $envKey.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $pathEntries = @($currentPath -split ";" | Where-Object { $_ } | ForEach-Object { & $normalize $_ })

        if ($pathEntries -notcontains $target) {
            $newPath = if ($currentPath) { "$($currentPath.TrimEnd(';'));$InstallDir" } else { $InstallDir }
            $envKey.SetValue("Path", $newPath, [Microsoft.Win32.RegistryValueKind]::ExpandString)
            # A direct registry write doesn't notify running programs, so
            # terminals opened later from Explorer wouldn't see the new PATH
            # until the next logon. Setting and removing a throwaway user
            # variable makes .NET broadcast that notification for us.
            [Environment]::SetEnvironmentVariable("GIT_COMMIT_SENTINEL_PATH_REFRESH", "1", "User")
            [Environment]::SetEnvironmentVariable("GIT_COMMIT_SENTINEL_PATH_REFRESH", $null, "User")
            Write-Host "==> added $InstallDir to your user PATH - restart your terminal for it to take effect"
        }
        else {
            Write-Host "==> $InstallDir is already on your user PATH"
        }
    }
    finally {
        $envKey.Close()
    }

    $sessionEntries = @($env:Path -split ";" | Where-Object { $_ } | ForEach-Object { & $normalize $_ })
    if ($sessionEntries -notcontains $target) {
        $env:Path += ";$InstallDir"
    }

    Write-Host ""
    Write-Host "==> next: git-commit-sentinel setup"
}
