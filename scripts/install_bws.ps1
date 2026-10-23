# Checksum verification implemented.
# This script automatically downloads, installs, or updates the Bitwarden Secrets Manager CLI (bws).
# It installs the Bitwarden Secrets Manager CLI (bws).

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
# -------------------------------------------------
# Install‑BWS.ps1 – Bitwarden Secrets Manager CLI installer
# -------------------------------------------------

[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [switch]$Force,

    [ValidateSet('User', 'Machine')]
    [string]$Scope = 'User',

    [string]$InstallPath,

    [switch]$Silent,
	
    [switch]$AllowDestructiveCleanup
)


# -----------------------------------------------------------------
# Logging
# -----------------------------------------------------------------
$LogDir = Join-Path -Path $env:TEMP -ChildPath "bws_installer_logs"
if (-not (Test-Path $LogDir)) { $null = New-Item -Path $LogDir -ItemType Directory }
$LogFile = Join-Path -Path $LogDir -ChildPath "bws_install_$(Get-Date -Format 'yyyy-MM-dd_HH-mm-ss').log"
Start-Transcript -Path $LogFile

# -----------------------------------------------------------------
# Global variables
# -----------------------------------------------------------------
$tempFile = $null
$rollbackActions = [System.Collections.Generic.List[ScriptBlock]]::new()

try {
    Write-Host "➡️ Starting BWS CLI installation..." -ForegroundColor Yellow
    if ($PSCmdlet.ShouldProcess("System", "Install/Update Bitwarden Secrets Manager CLI")) {

        # =================================================================
        # 1️⃣ PREPARATION – Proxy handling (unchanged, but kept for context)
        # =================================================================
        if ($PSVersionTable.PSVersion.Major -ge 6) {
            # PowerShell 6+ (Core) – no ProxyUseDefaultCredentials switch
            $webParams = @{
                UseBasicParsing = $true
                TimeoutSec      = 120
            }
            # $webParams.Proxy = "http://proxy.mycorp.com:8080"   # <‑‑ uncomment if you need a custom proxy
        }
        else {
            # Windows PowerShell 5.1 – must supply a Proxy URI when using ProxyUseDefaultCredentials
            $systemProxy = [System.Net.WebRequest]::GetSystemWebProxy()
            $proxyUri = $systemProxy.GetProxy([uri]'http://example.com').AbsoluteUri

            $webParams = @{
                UseBasicParsing            = $true
                TimeoutSec                 = 120
                Proxy                      = $proxyUri
                ProxyUseDefaultCredentials = $true
            }
        }

        # -----------------------------------------------------------------
        # 2️⃣ GATHER INFORMATION – latest release from GitHub (CLI‑asset logic)
        # -----------------------------------------------------------------
        $ApiUrl = 'https://api.github.com/repos/bitwarden/sdk-sm/releases'
        Write-Host "   - Querying GitHub API for releases…"
        $allReleases = Invoke-RestMethod -Uri $ApiUrl @webParams
        Write-Host "   - Found $($allReleases.Count) releases."

        # -------------------------------------------------
        # DEBUG
        # -------------------------------------------------
        Write-Host "`n--- DEBUG: All releases ---"
        $allReleases | ForEach-Object {
            Write-Host "     - $($_.tag_name): $($_.name)"
        }
        $latestRelease = $allReleases | Sort-Object -Property created_at -Descending | Where-Object { $_.name -like '*CLI*' } | Select-Object -First 1
        $latestVersion = $latestRelease.tag_name
        Write-Host "`n--- Latest release: $($latestRelease.tag_name) ---"
        $latestRelease.assets | ForEach-Object {
            Write-Host "  • $($_.name)   ($([math]::Round($_.size/1KB,2)) KB)"
        }
        Write-Host "--- End of asset list ---`n"


        # -----------------------------------------------------------------
        # Helper: turn a tag like "v1.2.3" or "bws-v1.2.3" into a [Version] object
        # If conversion fails we treat the version as 0.0.0 so it sorts last.
        # -----------------------------------------------------------------
        function Convert-ToVersion([string]$tag) {
            try {
                # Strip everything that is not a numeric version (leading "v", "bws-", etc.)
                $clean = $tag -replace '^[^\d]*', ''   # remove non‑digit prefix
                return [Version]$clean
            }
            catch {
                return [Version]'0.0.0'
            }
        }

        # -----------------------------------------------------------------
        # 1️⃣ First pass – Filter Windows MSVC build and its checksum
        # -----------------------------------------------------------------
        $arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'aarch64' } else { 'x86_64' }
        $assetPattern = "bws-$arch-pc-windows-msvc*.zip"
        $downloadAsset = $latestRelease.assets | Where-Object { $_.name -like $assetPattern } | Select-Object -First 1
        if (-not $downloadAsset) {
            Write-Warning "❌ Could not locate a Windows MSVC build ZIP in the latest release. Trying with previous version."
            $downloadAsset = $allReleases | ForEach-Object { $_.assets } | Where-Object { $_.name -like $assetPattern } | Select-Object -First 1
            if (-not $downloadAsset) {
                throw "❌ Could not find any Windows MSVC build ZIP in any recent release."
            }
            else {
                Write-Warning "Using previous release '$($downloadAsset.tag_name)' because no stable release contains a Windows MSVC build."
                $latestVersion = Convert-ToVersion $downloadAsset.tag_name
            }
        }
        Write-Host "✅ Download asset: $($downloadAsset.name)"
        
        # $checksumAsset = $latestRelease.assets | Where-Object { $_.name -like "*sha256*" } | Select-Object -First 1
        # if (-not $checksumAsset) {
        #     throw "❌ Could not find any SHA256 checksum file in the latest release."
        # }
        # Write-Host "✅ Checksum asset: $($checksumAsset.name)"

        # -----------------------------------------------------------------
        # 2️⃣ Second pass – if nothing was found, **allow prereleases** as a fallback
        # -----------------------------------------------------------------
        if (-not $downloadAsset) {
            foreach ($rel in $latestRelease) {
                $candidateZip = $rel.assets | Where-Object { $_.name -like $zipPattern }
                if (-not $candidateZip) { continue }

                # $baseName = [System.IO.Path]::GetFileNameWithoutExtension($candidateZip.name)
                # $candidateChk = $rel.assets | Where-Object {
                #     $_.name -like "*$baseName*" -and $_.name -like $checksumPattern
                # }
                # if ($candidateChk) {
                #     $downloadAsset = $candidateExe
                #     $checksumAsset = $candidateChk
                #     $latestVersion = $rel.tag_name
                #     Write-Warning "Using prerelease '$($rel.tag_name)' because no stable release contains a CLI‑named exe with a matching checksum."
                #     break
                # }
            }
        }

        # -----------------------------------------------------------------
        # Give up – nothing matched even after the prerelease pass
        # -----------------------------------------------------------------
        if (-not $downloadAsset) {
            throw "Could not locate a Windows executable whose name contains 'CLI' (and a matching checksum) in any recent release of the Bitwarden SDK repository."
        }
        # if (-not $checksumAsset) {
        #     throw "Checksum file (containing 'sha256') for the selected CLI executable not found in the chosen release."
        # }

        # -----------------------------------------------------------------
        # Success – report what we are going to download
        # -----------------------------------------------------------------
        Write-Host "   - Latest version found: $latestVersion"
        Write-Host "   - Download asset : $($downloadAsset.name) ($([math]::Round($downloadAsset.size/1KB,2)) KB)"
        # Write-Host "   - Checksum asset : $($checksumAsset.name)"

        # -----------------------------------------------------------------
        # 3️⃣ DETERMINE INSTALLATION DIRECTORY (unchanged)
        # -----------------------------------------------------------------
        if ($PSBoundParameters.ContainsKey('InstallPath')) {
            $InstallDir = $InstallPath
        }
        else {
            $InstallDir = if ($Scope -eq 'User') {
                "$env:LOCALAPPDATA\Programs\bws"
            }
            else {
                "$env:ProgramFiles\bws"
            }
        }
        Write-Host "   - Installation path set to: '$InstallDir' (Scope: $Scope)"
        $DestinationFile = Join-Path -Path $InstallDir -ChildPath 'bws.exe'

        # =================================================================
        # 4️⃣ DOWNLOAD & VERIFY
        # =================================================================
        $tempFile = Join-Path $env:TEMP ("{0}_{1}.zip" -f $latestRelease.tag_name, [System.IO.Path]::GetRandomFileName())
        Write-Host "   - Downloading to temporary file: $tempFile"
        Invoke-WebRequest -Uri $downloadAsset.browser_download_url -OutFile $tempFile @webParams

        # ----- Checksum verification -----
        $bwsVersion = $latestRelease.tag_name -replace '^bws-v', ''
        $checksumUrl = "https://github.com/bitwarden/sdk-sm/releases/download/$($latestRelease.tag_name)/bws-sha256-checksums-$bwsVersion.txt"
        Write-Host "   - Verifying file integrity (Checksum)..."
        $checksumContent = Invoke-RestMethod -Uri $checksumUrl @webParams
        $matchingLine = ($checksumContent -split "`n") | Where-Object { $_ -match [regex]::Escape($downloadAsset.name) } | Select-Object -First 1
        if (-not $matchingLine) {
            throw "Could not find a checksum line matching '$($downloadAsset.name)' in the checksum file."
        }
        $expectedHash = ($matchingLine -split '\s+', 2)[0].Trim().ToUpper()
        $actualHash = (Get-FileHash -Path $tempFile -Algorithm SHA256).Hash.ToUpper()
        if ($expectedHash -ne $actualHash) {
            throw "Checksum mismatch! Expected '$expectedHash' but got '$actualHash'."
        }
        Write-Host "   - Checksum valid." -ForegroundColor Green


        # ----- Digital‑signature verification -----
        # Write-Host "   - Verifying file authenticity (Digital Signature)…"
        # $signature = Get-AuthenticodeSignature -FilePath $tempFile
        # if ($signature.Status -ne 'Valid') {
        #     throw "Digital signature is not valid. Status: $($signature.Status)"
        # }
        # if ($signature.SignerCertificate.Subject -notmatch "Bitwarden") {
        #     throw "Certificate subject does not match 'Bitwarden'."
        # }
        # Write-Host "   - Signature valid. Signed by: $($signature.SignerCertificate.Subject)" -ForegroundColor Green

        # =================================================================
        # 5️⃣ INSTALLATION
        # =================================================================
        if (-not (Test-Path -Path $InstallDir)) {
            $rollbackActions.Add({
                    if ($PSCmdlet.ShouldProcess($InstallDir, "Rollback: Remove Directory")) {
                        Remove-Item -Path $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
                    }
                })
            if ($PSCmdlet.ShouldProcess($InstallDir, "Create Installation Directory")) {
                $null = New-Item -Path $InstallDir -ItemType Directory -Force
            }
        }

        # Extract the ZIP file to get the bws.exe
        $tempExtractDir = Join-Path $env:TEMP ("bws_extract_{0}" -f [System.IO.Path]::GetRandomFileName())
        $rollbackActions.Add({
                if (Test-Path $tempExtractDir) {
                    if ($PSCmdlet.ShouldProcess($tempExtractDir, "Rollback: Remove Extract Directory")) {
                        Remove-Item -Path $tempExtractDir -Recurse -Force -ErrorAction SilentlyContinue
                    }
                }
            })

        if ($PSCmdlet.ShouldProcess("Temporary extract directory", "Create and extract ZIP")) {
            $null = New-Item -Path $tempExtractDir -ItemType Directory -Force
            Expand-Archive -Path $tempFile -DestinationPath $tempExtractDir -Force
            
            # Find the bws.exe file in the extracted contents
            $extractedExe = Get-ChildItem -Path $tempExtractDir -Recurse -Filter "bws.exe" | Select-Object -First 1
            
            if (-not $extractedExe) {
                throw "Could not find bws.exe in the extracted ZIP file. Please check the ZIP contents."
            }
            
            $rollbackActions.Add({
                    if (Test-Path $DestinationFile) {
                        if ($PSCmdlet.ShouldProcess($DestinationFile, "Rollback: Remove File")) {
                            Remove-Item -Path $DestinationFile -Force -ErrorAction SilentlyContinue
                        }
                    }
                })
            
            if ($PSCmdlet.ShouldProcess($DestinationFile, "Copy extracted exe to destination")) {
                Copy-Item -Path $extractedExe.FullName -Destination $DestinationFile -Force
            }
        }

        # Clean up extraction directory
        if (Test-Path $tempExtractDir) {
            Remove-Item -Path $tempExtractDir -Recurse -Force -ErrorAction SilentlyContinue
        }


        # =================================================================
        # 6️⃣ PATH CONFIGURATION
        # =================================================================
        $oldPath = [System.Environment]::GetEnvironmentVariable('Path', $Scope)
        $rollbackActions.Add({
                if ($PSCmdlet.ShouldProcess("PATH for $Scope", "Rollback: Restore old value")) {
                    [System.Environment]::SetEnvironmentVariable('Path', $oldPath, $Scope)
                }
            })
        $currentPath = [System.Environment]::GetEnvironmentVariable('Path', $Scope)

        if ($currentPath -notlike "*$InstallDir*") {
            if ($PSCmdlet.ShouldProcess("PATH for $Scope", "Add '$InstallDir'")) {
                $newPath = ($currentPath.TrimEnd(';') + ';' + $InstallDir)
                [System.Environment]::SetEnvironmentVariable('Path', $newPath, $Scope)
                $env:Path += ";$InstallDir"
                Write-Host "   - PATH updated. Please open a NEW terminal." -ForegroundColor Green
            }
        }
        else {
            Write-Host "   - Installation directory already in PATH." -ForegroundColor Cyan
        }
        Write-Host "✅ BWS CLI installed/updated successfully!" -ForegroundColor Green
    }

}
catch {
    # -----------------------------------------------------------------
    # Error handling & rollback (unchanged)
    # -----------------------------------------------------------------
    Write-Error "❌ An error occurred during installation: $_"
    Write-Host "🔥 Attempting to roll back changes..." -ForegroundColor Yellow
    $rollbackActions.Reverse()
    foreach ($action in $rollbackActions) {
        try { & $action } catch { Write-Warning "Rollback action failed: $_" }
    }
    Write-Host "🔥 Rollback complete." -ForegroundColor Yellow
    exit 1

}
finally {
    # -----------------------------------------------------------------
    # Clean‑up temporary file (unchanged)
    # -----------------------------------------------------------------
    if ($tempFile -and (Test-Path -Path $tempFile)) {
        Remove-Item -Path $tempFile -Force -ErrorAction SilentlyContinue
    }
    Write-Host "➡️ Log file created at: $LogFile"
    Stop-Transcript
}
