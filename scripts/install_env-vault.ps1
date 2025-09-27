# install.ps1
# A script to configure and install the env-vault tool on Windows using PowerShell.

Write-Host "--- 🛡️  env-vault Setup for Windows ---"

# --- Step 1: Set GOPRIVATE ---
Write-Host "`n1. Configuring Go to access private modules..."
$GOPRIVATE_PATH = "github.com/bulga138/*"
go env -w GOPRIVATE=$GOPRIVATE_PATH
Write-Host "✅ GOPRIVATE set to $GOPRIVATE_PATH"

# --- Step 2: Get GitHub Credentials ---
Write-Host "`n2. Please enter your GitHub OAuth credentials."
$GITHUB_CLIENT_ID = Read-Host "Enter your GITHUB_CLIENT_ID"
$GITHUB_CLIENT_SECRET_SECURE = Read-Host "Enter your GITHUB_CLIENT_SECRET" -AsSecureString
# Convert the secure string to plain text for the environment variable
$GITHUB_CLIENT_SECRET = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto([System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($GITHUB_CLIENT_SECRET_SECURE))

if (-not $GITHUB_CLIENT_ID -or -not $GITHUB_CLIENT_SECRET) {
    Write-Error "❌ Error: Both Client ID and Client Secret are required. Aborting."
    exit 1
}

# --- Step 3: Set Environment Variables Permanently ---
Write-Host "`n3. Saving environment variables to your user profile..."
# Sets the variables permanently for the current user
[System.Environment]::SetEnvironmentVariable("GITHUB_CLIENT_ID", $GITHUB_CLIENT_ID, "User")
[System.Environment]::SetEnvironmentVariable("GITHUB_CLIENT_SECRET", $GITHUB_CLIENT_SECRET, "User")
Write-Host "✅ Environment variables set. Please restart your terminal to apply changes."

# --- Step 4: Install env-vault ---
Write-Host "`n4. Installing env-vault..."
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

# ---  Step 5: Install bws ---

Invoke-WebRequest -useb https://raw.githubusercontent.com/bitwarden/sdk/main/scripts/install.ps1 | Invoke-Expression
if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ bws successfully installed!"
}
else {
    Write-Error "❌ Error: bws installation failed. Please check for errors above."
    exit 1
}

Write-Host "`n--- 🎉 Setup Complete! ---"

# macOS / Linux
curl -sSfL https://raw.githubusercontent.com/bitwarden/sdk/main/scripts/install.sh | sh

# Windows (PowerShell)