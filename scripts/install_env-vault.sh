#!/bin/bash

# A script to configure and install the env-vault tool.

echo "---  env-vault Setup ---"

# --- Step 1: Install env-vault ---
echo
echo "1. Installing env-vault..."
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
echo "--- Setup Complete! ---"