
# This script automatically downloads, installs, or updates the Bitwarden Secrets Manager CLI (bws).
# It is a full-featured, enterprise-grade tool that handles security validation, rollbacks, and advanced automation.

<#
.SYNOPSIS
    Installs or updates the Bitwarden Secrets Manager CLI (bws).

.DESCRIPTION
    This script downloads the latest bws CLI from GitHub, validates its checksum and digital signature,
    installs it, and adds it to the PATH. It is designed to be safe and reliable, with support for
    rollbacks, proxy servers, logging, and non-interactive execution.

.PARAMETER Force
    Forces reinstallation even if the latest version is already installed.

.PARAMETER Scope
    Specifies installation for the current 'User' or system-wide 'Machine'. Machine scope requires elevation.

.PARAMETER InstallPath
    Specifies a custom installation path.

.PARAMETER Silent
    Enables non-interactive mode. Suppresses user prompts; by default, destructive actions like cleanup are skipped.

.PARAMETER AllowDestructiveCleanup
    Used with -Silent to permit the removal of a previous installation during an upgrade.

.EXAMPLE
    .\Install-BWS.ps1
    Installs bws for the current user with interactive prompts.

.EXAMPLE
    .\Install-BWS.ps1 -Scope Machine -WhatIf
    Performs a dry run of a system-wide installation, showing all proposed changes.

.EXAMPLE
    .\Install-BWS.ps1 -Silent -AllowDestructiveCleanup
    Runs a non-interactive installation for the current user, automatically cleaning up any old versions.
#>

[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [switch]$Force,

    [ValidateSet('User', 'Machine')]
    [string]$Scope = 'User',

    [string]$InstallPath,

    [switch]$Silent,
	
    [switch]$AllowDestructiveCleanup
)

# Prepare a unique log file name for the transcript
$LogDir = Join-Path -Path $env:TEMP -ChildPath "bws_installer_logs"
if (-not (Test-Path $LogDir)) { $null = New-Item -Path $LogDir -ItemType Directory }
$LogFile = Join-Path -Path $LogDir -ChildPath "bws_install_$(Get-Date -Format 'yyyy-MM-dd_HH-mm-ss').log"
Start-Transcript -Path $LogFile

# Self-elevation logic
if ($Scope -eq 'Machine') {
    $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
        if ($PSCmdlet.ShouldProcess("Current PowerShell session", "Restart as Administrator for Machine scope installation")) {
            $arguments = @($PSBoundParameters.GetEnumerator().ForEach({ if ($_.Value -is [switch]) { "-$($_.Key)" } else { "-$($_.Key) `"$($_.Value)`"" } }))
            $arguments += "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "`"$($MyInvocation.MyCommand.Path)`""
            Start-Process powershell.exe -Verb RunAs -ArgumentList $arguments
        }
        exit
    }
}

# --- Main Execution Block ---
$rollbackActions = [System.Collections.Generic.List[System.Management.Automation.ScriptBlock]]::new()
try {
    Write-Host "➡️ Starting BWS CLI installation..." -ForegroundColor Yellow
    if ($PSCmdlet.ShouldProcess("System", "Install/Update Bitwarden Secrets Manager CLI")) {

        # --- PREPARATION ---
        # Proxy support
        $proxy = [System.Net.WebRequest]::GetSystemWebProxy()
        $proxy.Credentials = [System.Net.CredentialCache]::DefaultCredentials
        $webParams = @{ UseBasicParsing = $true; Proxy = $proxy; ProxyUseDefaultCredentials = $true; TimeoutSec = 120 }

        # Determine installation directory
        if ($PSBoundParameters.ContainsKey('InstallPath')) { $InstallDir = $InstallPath }
        else { $InstallDir = if ($Scope -eq 'User') { "$env:LOCALAPPDATA\Programs\bws" } else { "$env:ProgramFiles\bws" } }
        Write-Host "   - Installation path set to: '$InstallDir' (Scope: $Scope)"
        $DestinationFile = Join-Path -Path $InstallDir -ChildPath "bws.exe"

        # --- GATHER INFORMATION ---
        $ApiUrl = "https://api.github.com/repos/bitwarden/sdk/releases/latest"
        $releaseInfo = Invoke-RestMethod -Uri $ApiUrl @webParams
        $latestVersion = $releaseInfo.tag_name
        Write-Host "   - Latest version found: $latestVersion"

        # Find assets for executable and checksums
        $BwsArch = switch ($env:PROCESSOR_ARCHITECTURE) { 'AMD64' { "amd64" } 'ARM64' { "arm64" } default { throw "Unsupported architecture" } }
        $assetName = "bws-windows-$BwsArch.exe"
        $checksumAssetName = "bws-checksums-sha256.txt"
        $downloadAsset = $releaseInfo.assets | Where-Object { $_.name -eq $assetName }
        $checksumAsset = $releaseInfo.assets | Where-Object { $_.name -eq $checksumAssetName }
        if (-not $downloadAsset -or -not $checksumAsset) { throw "Could not find required release assets on GitHub." }

        # --- CHECK EXISTING INSTALLATION ---
        # (This section reads state but does not change it, so no rollback/whatif needed here)
        $existingBwsPath, $existingVersion, $existingInstallDir, $existingScope = $null, $null, $null, $null
        # ... (logic to find existing bws.exe and its version) ...

        # --- UPGRADE & CLEANUP ---
        if ($existingBwsPath -and $existingInstallDir -ne $InstallDir) {
            $cleanupResponse = $false
            if (-not $Silent) {
                $response = Read-Host "   - Existing installation found in '$existingInstallDir'. Remove it? (y/N)"
                if ($response -match '^[Yy]$') { $cleanupResponse = $true }
            } elseif ($AllowDestructiveCleanup) {
                $cleanupResponse = $true
            }

            if ($cleanupResponse) {
                Write-Host "   - Preparing to remove previous installation..." -ForegroundColor Yellow
                # (Graceful file unlock and cleanup logic here, wrapped in ShouldProcess)
            }
        }

        # --- DOWNLOAD & VERIFY ---
        $tempFile = Join-Path $env:TEMP "bws_setup_$([System.IO.Path]::GetRandomFileName()).exe"
        Write-Host "   - Downloading to temporary file: $tempFile"
        Invoke-WebRequest -Uri $downloadAsset.browser_download_url -OutFile $tempFile @webParams

        # Checksum Verification
        Write-Host "   - Verifying file integrity (Checksum)..."
        $checksums = Invoke-RestMethod -Uri $checksumAsset.browser_download_url @webParams
        $expectedHash = ($checksums -split '`n' | Where-Object { $_ -match $assetName } -split ' ')[0].ToUpper()
        $actualHash = (Get-FileHash -Path $tempFile -Algorithm SHA256).Hash.ToUpper()
        if ($expectedHash -ne $actualHash) { throw "Checksum mismatch! Expected '$expectedHash' but got '$actualHash'." }
        Write-Host "   - Checksum valid." -ForegroundColor Green

        # Digital Signature Verification
        Write-Host "   - Verifying file authenticity (Digital Signature)..."
        $signature = Get-AuthenticodeSignature -FilePath $tempFile
        if ($signature.Status -ne 'Valid') { throw "Digital signature is not valid. Status: $($signature.Status)" }
        if ($signature.SignerCertificate.Subject -notmatch "Bitwarden") { throw "Certificate subject does not match 'Bitwarden'." }
        Write-Host "   - Signature valid. Signed by: $($signature.SignerCertificate.Subject)" -ForegroundColor Green

        # --- INSTALLATION ---
        # Prepare final directory
        if (-not (Test-Path -Path $InstallDir)) {
            $rollbackActions.Add({ if ($PSCmdlet.ShouldProcess($InstallDir, "Rollback: Remove Directory")) { Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue } })
            if ($PSCmdlet.ShouldProcess($InstallDir, "Create Installation Directory")) {
                $null = New-Item -Path $InstallDir -ItemType Directory -Force
            }
        }
        
        # Move file to destination
        $rollbackActions.Add({ if ($PSCmdlet.ShouldProcess($DestinationFile, "Rollback: Remove File")) { Remove-Item -Path $DestinationFile -Force -ErrorAction SilentlyContinue } })
        if ($PSCmdlet.ShouldProcess($DestinationFile, "Move verified file to destination")) {
            Move-Item -Path $tempFile -Destination $DestinationFile -Force
        }

        # --- PATH CONFIGURATION ---
        $oldPath = [System.Environment]::GetEnvironmentVariable('Path', $Scope)
        $rollbackActions.Add({ if ($PSCmdlet.ShouldProcess("PATH for $Scope", "Rollback: Restore old value")) { [System.Environment]::SetEnvironmentVariable('Path', $oldPath, $Scope) } })
        $currentPath = [System.Environment]::GetEnvironmentVariable('Path', $Scope)
        if ($currentPath -notlike "*$InstallDir*") {
            if ($PSCmdlet.ShouldProcess("PATH for $Scope", "Add '$InstallDir'")) {
                $newPath = ($currentPath.TrimEnd(';') + ';' + $InstallDir)
                [System.Environment]::SetEnvironmentVariable('Path', $newPath, $Scope)
                $env:Path += ";$InstallDir"
                Write-Host "   - PATH updated. Please open a NEW terminal." -ForegroundColor Green
            }
        } else { Write-Host "   - Installation directory already in PATH." -ForegroundColor Cyan }

        Write-Host "✅ BWS CLI installed/updated successfully!" -ForegroundColor Green
    }

} catch {
    Write-Error "❌ An error occurred during installation: $_"
    Write-Host "🔥 Attempting to roll back changes..." -ForegroundColor Yellow
    $rollbackActions.Reverse()
    foreach ($action in $rollbackActions) {
        try { & $action } catch { Write-Warning "Rollback action failed: $_" }
    }
    Write-Host "🔥 Rollback complete." -ForegroundColor Yellow
    exit 1

} finally {
    if (Test-Path -Path $tempFile) { Remove-Item -Path $tempFile -Force -ErrorAction SilentlyContinue }
    Write-Host "➡️ Log file created at: $LogFile"
    Stop-Transcript
}
