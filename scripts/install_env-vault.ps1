# A script to configure and install the env-vault tool on Windows using PowerShell.

Write-Host "---  env-vault Setup for Windows ---"

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