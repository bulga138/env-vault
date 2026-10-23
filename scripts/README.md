# env-vault Installation Scripts

This directory contains utility scripts to simplify the installation and configuration of **env-vault** and its core dependency, the **Bitwarden Secrets Manager CLI (`bws`)**, specifically tailored for **Windows PowerShell**.

---

## Available Scripts

| Script Name | Purpose | Scope | Dependencies |
|:---|:---|:---|:---|
| `install_env-vault.ps1` | **Main Installer.** Handles checking for Go, installing `env-vault` via `go install`, and interactively configuring the necessary environment variables (`BWS_PROJECT_ID`, `VAULT_FILE_PATH`, etc.). | Windows | **Go (1.24+)** |
| `install_bws.ps1` | **Dependency Installer.** PowerShell script that downloads, installs, and adds the official **Bitwarden Secrets Manager CLI (`bws.exe`)** to the system or user PATH. | Windows | **PowerShell 5.1+** |

---

## Quick Start (Windows)

The recommended way to set up the entire environment is to run the main installer script.

1. **Prerequisites:**
 * Ensure **Go (version 1.24 or newer)** is installed and available in your system's PATH.
 * Ensure you are running **PowerShell 5.1 or PowerShell Core (6+)**.

2. **Execute the Main Installer:**
 Open PowerShell and run the main script. This script will automatically check for the `bws` dependency and prompt you to run its installer if it's missing.

 ```powershell
 # Ensure execution policy allows running local scripts (run as Administrator once)
 Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass

 # Run the main installation script
 .\install_env-vault.ps1
 ```

3. **Follow Prompts:**
 The script will guide you through:
 * Installing or reinstalling `env-vault`.
 * *Optionally* running `install_bws.ps1` to install the Bitwarden CLI.
 * Setting the configuration values for `env-vault` (e.g., `BWS_PROJECT_ID`).

---

## Script Details

### `install_env-vault.ps1`

This script automates the full setup flow for `env-vault`.

**Key Actions:**
* Verifies that the **Go** programming language is installed.
* Installs/reinstalls `env-vault` using `go install`.
* **Interactively checks for and optionally installs the `bws` CLI** by calling `install_bws.ps1`.
* Creates the configuration directory (`$HOME\.env-vault`) and config file (`.env`).
* Prompts the user to set essential configuration variables like `BWS_PROJECT_ID` and `GOOGLE_APPLICATION_CREDENTIALS`.

### `install_bws.ps1`

This installer downloads and configures the Bitwarden CLI (bws).

**Features:**
* **Automatic Latest Release Detection:** Queries the GitHub API for the latest stable release.
* **Installation Scope:** Supports installation for the current **User** (default: `$env:LOCALAPPDATA\Programs\bws`) or system-wide **Machine** (requires elevation, default: `$env:ProgramFiles\bws`).
* **Path Configuration:** Automatically adds the installation directory to the chosen PATH scope.
* **Rollbacks:** Implements rollback actions to revert changes if the installation fails (e.g., removing newly created directories or restoring the old PATH).
* **Logging:** Creates a detailed transcript log in the `%TEMP%\bws_installer_logs` directory.

**Usage Examples (within PowerShell):**

| Command | Description |
|:---|:---|
| `.\install_bws.ps1` | Installs `bws` for the current **User** with interactive prompts. |
| `.\install_bws.ps1 -Scope Machine` | Attempts a system-wide installation (requires elevation). |
| `.\install_bws.ps1 -Silent -Force` | Non-interactive install, forcing reinstallation even if already present. |
