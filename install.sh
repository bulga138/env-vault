#!/bin/bash

# A script to configure and install the env-vault tool.

echo "--- 🛡️  env-vault Setup ---"

# --- Step 1: Set GOPRIVATE ---
echo
echo "1. Configuring Go to access private modules..."
GOPRIVATE_PATH="github.com/bulga138/*"
go env -w GOPRIVATE=$GOPRIVATE_PATH
echo "✅ GOPRIVATE set to $GOPRIVATE_PATH"

# --- Step 2: Get GitHub Credentials ---
echo
echo "2. Please enter your GitHub OAuth credentials."
read -p "Enter your GITHUB_CLIENT_ID: " GITHUB_CLIENT_ID
read -sp "Enter your GITHUB_CLIENT_SECRET: " GITHUB_CLIENT_SECRET
echo # for a new line after the secret input

if [ -z "$GITHUB_CLIENT_ID" ] || [ -z "$GITHUB_CLIENT_SECRET" ]; then
    echo "❌ Error: Both Client ID and Client Secret are required. Aborting."
    exit 1
fi

# --- Step 3: Update Shell Profile ---
echo
echo "3. Detecting your shell to save environment variables..."
SHELL_PROFILE=""
if [[ "$SHELL" == *"zsh"* ]]; then
    SHELL_PROFILE="$HOME/.zshrc"
elif [[ "$SHELL" == *"bash"* ]]; then
    SHELL_PROFILE="$HOME/.bashrc"
else
    echo "⚠️ Could not detect .zshrc or .bashrc. You will need to set the variables manually."
fi

if [ -n "$SHELL_PROFILE" ]; then
    echo "   Appending variables to $SHELL_PROFILE"
    echo "" >> "$SHELL_PROFILE"
    echo "# env-vault configuration" >> "$SHELL_PROFILE"
    echo "export GITHUB_CLIENT_ID=\"$GITHUB_CLIENT_ID\"" >> "$SHELL_PROFILE"
    echo "export GITHUB_CLIENT_SECRET=\"$GITHUB_CLIENT_SECRET\"" >> "$SHELL_PROFILE"
    echo "✅ Shell profile updated."
    echo "   Please run 'source $SHELL_PROFILE' or restart your terminal to apply changes."
fi

# --- Step 4: Install env-vault ---
echo
echo "4. Installing env-vault..."
# Ensure Go bin is in the PATH for the install command to be found later
export PATH=$PATH:$(go env GOPATH)/bin
go install github.com/bulga138/env-vault@latest

if [ $? -eq 0 ]; then
    echo "✅ env-vault successfully installed!"
else
    echo "❌ Error: go install failed. Please check for errors above."
    exit 1
fi

echo
echo "--- 🎉 Setup Complete! ---"