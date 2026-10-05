# A script to configure and install the env-vault tool on Windows using PowerShell.

Write-Host "---  env-vault Setup for Windows ---"
# Step 0: Check for Go installation
Write-Host "`n0. Checking for Go installation..."
$goVersion = & go version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Error "❌ Error: Go is not installed or not found in PATH. Please install Go from https://golang.org/dl/ and ensure it's in your PATH."
    exit 1
}
else {
    Write-Host "✅ Go is installed: $goVersion"
}

# --- Step 1: Install env-vault ---
Write-Host "`n1. Check if env-vault is already installed..."
$envVaultVersion = & env-vault --version 2>$null
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ env-vault is already installed: $envVaultVersion"
    $userResponse = Read-Host -Prompt "Do you want to reinstall it? (y/N): "
    if ($userResponse -eq "y" -or $userResponse -eq "Y") {
        Write-Host "Reinstalling env-vault..."
    }
    else {
        Write-Host "Skipping installation. Exiting script."
        exit 0
    }
}
else {
    Write-Host "env-vault not found. Proceeding with installation..."
}

Write-Host "`n1. Installing env-vault..."
# Ensure Go bin is in the PATH for the install command to be found later
$env:Path += ";$(go env GOPATH)\bin"
go install "github.com/bulga138/env-vault@latest"

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ env-vault successfully installed!"
}
else {
    Write-Error "❌ Error: go install failed. Please check for errors above."
    exit 1
}

# Step 1.5: Check for bws installation
Write-Host "`n1.5. Checking for bws installation..."
$bwsVersion = & bws --version 2>$null | Out-String
if ($LASTEXITCODE -ne 0) {
    $userResponse = Read-Host -Prompt "⚠️ 'bws' not found. Do you want to install it now? (y/N): "
    if ($userResponse -eq "y" -or $userResponse -eq "Y") {
        Write-Host "Installing 'bws'..."
        .\install_bws.ps1
        Write-Host "✅ 'bws' successfully installed!"
    }
    else {
        Write-Host "Skipping 'bws' installation. Exiting script."
        exit 0
    }
}
else {
    Write-Host "✅ 'bws' is installed: ${bwsVersion}."
}

# --- Step 2: Configure env-vault ---
Write-Host "`n2. Configuring env-vault..."
$envVaultConfigPath = "$HOME\.env-vault"
if (-not (Test-Path -Path $envVaultConfigPath)) {
    Write-Host "Creating env-vault config directory at $envVaultConfigPath"
    New-Item -ItemType Directory -Path $envVaultConfigPath | Out-Null
}
$configFilePath = "$envVaultConfigPath\.env"
if (-not (Test-Path -Path $configFilePath)) {
    Write-Host "Creating env-vault config file at $configFilePath"
    New-Item -ItemType File -Path $configFilePath | Out-Null
}
else {
    Write-Host "Config file already exists at $configFilePath"
}

$userResponse = Read-Host -Prompt "Do you want to config the env-vault config file? (y/N): "
if ($userResponse -eq "y" -or $userResponse -eq "Y") {
    Write-Host "Proceeding with configuration..."
}
else {
    Write-Host "Skipping configuration. Exiting script."
    exit 0
}

# Prompt user for configuration details
Write-Host "Please enter the following details to configure env-vault:"
$bwsAccessToken = Read-Host -Prompt "BWS Access Token: "
$bwsProjectId = Read-Host -Prompt "BWS Project ID: "
$googleApplicationCredentials = Read-Host -Prompt "Google Application Credentials Path (default: ./serviceAccountKey.json): "
if ([string]::IsNullOrWhiteSpace($googleApplicationCredentials)) {
    $googleApplicationCredentials = "./serviceAccountKey.json"
}
$vaultFilePath = Read-Host -Prompt "Vault File Path (default: ./vault.json): "
if ([string]::IsNullOrWhiteSpace($vaultFilePath)) {
    $vaultFilePath = "./vault.json"
}
$envVaultExistingFilename = Read-Host -Prompt "Existing Filename (optional): "
$envVaultPepper = Read-Host -Prompt "Pepper (optional): "
$logOutput = Read-Host -Prompt "Log Output (default: TERMINAL): "
if ([string]::IsNullOrWhiteSpace($logOutput)) {
    $logOutput = "TERMINAL"
}
$logLevel = Read-Host -Prompt "Log Level (default: DEBUG): "
if ([string]::IsNullOrWhiteSpace($logLevel)) {
    $logLevel = "DEBUG"
}
$logFile = Read-Host -Prompt "Log File (default: env-vault.log): "
if ([string]::IsNullOrWhiteSpace($logFile)) {
    $logFile = "env-vault.log"
}

# Write to env-vault config file
@"
# Environment variables for env-vault
VAULT_FILE_PATH=$vaultFilePath
ENV_VAULT_EXISTING_FILENAME=$envVaultExistingFilename
ENV_VAULT_PEPPER=$envVaultPepper

# Logging configuration
LOG_OUTPUT=$logOutput
LOG_FILE=$logFile
LOG_LEVEL=$logLevel

# Path to your Google service account credentials
GOOGLE_APPLICATION_CREDENTIALS=$googleApplicationCredentials

# BitWarden Secrets Manager configuration
BWS_ACCESS_TOKEN=$bwsAccessToken
BWS_PROJECT_ID=$bwsProjectId
"@ | Set-Content -Path $configFilePath

Write-Host "✅ env-vault config file created at $configFilePath"