# install.ps1
# A script to configure and install the env-vault tool on Windows using PowerShell.

Write-Host "--- 🛡️  env-vault Setup for Windows ---"

# --- Step 1: Install env-vault ---
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

# ---  Step 2: Install bws ---

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