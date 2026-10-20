# Uninstalls git-commit-sentinel on Windows, undoing install.ps1 and
# setup: runs `git-commit-sentinel uninstall` to remove the global hook
# and settings, deletes the binary, and removes the user PATH entry
# install.ps1 added once the install directory is empty.
#
# Usage:
#   irm https://raw.githubusercontent.com/blue-noah/git-commit-sentinel/main/scripts/uninstall.ps1 | iex
#
# Env overrides:
#   $env:GIT_COMMIT_SENTINEL_INSTALL_DIR = "C:\some\dir" where it was installed (default: $HOME\bin)
#   $env:GIT_COMMIT_SENTINEL_KEEP_CONFIG = "1"           keep the commitsentinel.* git settings
#
# Keep this file ASCII-only: Windows PowerShell 5.1 reads it as
# Windows-1252, where UTF-8 punctuation can turn into a closing quote.

# `irm | iex` runs this in the caller's session: the script block keeps
# $ErrorActionPreference and every variable below out of it.
& {
    $ErrorActionPreference = "Stop"

    $InstallDir = if ($env:GIT_COMMIT_SENTINEL_INSTALL_DIR) { $env:GIT_COMMIT_SENTINEL_INSTALL_DIR } else { Join-Path $HOME "bin" }
    $Exe = Join-Path $InstallDir "git-commit-sentinel.exe"

    if (Test-Path -LiteralPath $Exe -PathType Leaf) {
        $uninstallArgs = @("uninstall")
        if ($env:GIT_COMMIT_SENTINEL_KEEP_CONFIG -eq "1") { $uninstallArgs += "-keep-config" }
        & $Exe @uninstallArgs
        if ($LASTEXITCODE -ne 0) {
            throw ("error: ``git-commit-sentinel uninstall`` failed, so nothing was deleted.`n" +
                "It needs v0.0.4 or later: if it reported an unknown command, upgrade`n" +
                "by running install.ps1, then run this script again.")
        }
        Remove-Item -LiteralPath $Exe -Force
        Write-Host "==> removed $Exe"
    }
    else {
        Write-Host "==> no binary at $Exe (already removed? set `$env:GIT_COMMIT_SENTINEL_INSTALL_DIR if you installed elsewhere)"
    }

    # The PATH entry also serves anything else in that directory: it goes
    # only when the directory is empty, and the empty directory goes with it.
    if ((Test-Path -LiteralPath $InstallDir) -and (Get-ChildItem -LiteralPath $InstallDir -Force | Select-Object -First 1)) {
        Write-Host "==> kept $InstallDir on PATH: it holds other files"
    }
    else {
        if (Test-Path -LiteralPath $InstallDir) {
            Remove-Item -LiteralPath $InstallDir -Force
        }

        $normalize = { param($entry) [Environment]::ExpandEnvironmentVariables($entry).TrimEnd("\") }
        $target = & $normalize $InstallDir

        # Raw registry value, as in install.ps1, so %VAR% references in
        # other entries stay unexpanded.
        $envKey = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey("Environment")
        try {
            $currentPath = $envKey.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
            $entries = @($currentPath -split ";" | Where-Object { $_ })
            $kept = @($entries | Where-Object { (& $normalize $_) -ne $target })
            if ($kept.Count -ne $entries.Count) {
                $envKey.SetValue("Path", ($kept -join ";"), [Microsoft.Win32.RegistryValueKind]::ExpandString)
                # Same notification trick as install.ps1.
                [Environment]::SetEnvironmentVariable("GIT_COMMIT_SENTINEL_PATH_REFRESH", "1", "User")
                [Environment]::SetEnvironmentVariable("GIT_COMMIT_SENTINEL_PATH_REFRESH", $null, "User")
                Write-Host "==> removed $InstallDir from your user PATH"
            }
        }
        finally {
            $envKey.Close()
        }

        $env:Path = (@($env:Path -split ";" | Where-Object { $_ -and (& $normalize $_) -ne $target }) -join ";")
    }

    Write-Host ""
    Write-Host "==> git-commit-sentinel is uninstalled. Restart other open terminals and IDEs for the PATH change."
    Write-Host ""
    Write-Host "Repositories set up with ``setup -scope local`` keep their own hook, which now"
    Write-Host "fails every commit with `"git-commit-sentinel: not found`". Delete"
    Write-Host ".git\hooks\commit-msg in each of them."
}
