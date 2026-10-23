
#!/usr/bin/env bash

# This script automatically downloads, installs, or updates the Bitwarden Secrets Manager CLI (bws).
# It installs the Bitwarden Secrets Manager CLI (bws).

# --- Strict Mode ---
set -euo pipefail

# --- Global Variables ---
FORCE=false
SCOPE="User" # 'User' or 'Machine'
INSTALL_PATH=""
SILENT=false
ALLOW_DESTRUCTIVE_CLEANUP=false
WHAT_IF=false
LOG_DIR="${TMPDIR:-/tmp}/bws_installer_logs"
LOG_FILE="${LOG_DIR}/bws_install_$(date +'%Y-%m-%d_%H-%M-%S').log"
ROLLBACK_ACTIONS=()
TEMP_FILE=""

# --- Help Text ---
usage() {
  cat << EOF
Installs or updates the Bitwarden Secrets Manager CLI (bws).

USAGE:
    ./install-bws.sh [OPTIONS]

OPTIONS:
    --scope <SCOPE>                 Set installation scope: 'User' (default) or 'Machine' (requires sudo).
    --install-path <PATH>           Specify a custom installation path.
    --force                         Force reinstallation even if the latest version is present.
    --silent                        Enable non-interactive mode. Suppresses prompts.
    --allow-destructive-cleanup     Used with --silent to permit removing a previous installation.
    --what-if                       Perform a dry run without making any changes to the system.
    -h, --help                      Show this help message.
EOF
}

# --- Logging and Output Functions ---
# A logging function that respects the --silent flag for regular messages.
log() {
  if ! $SILENT; then
    echo -e "$@"
  fi
}
# A function for logging errors or critical info that should always be shown.
log_critical() {
  echo -e "$@" >&2
}

# --- Core Utility Functions ---
# Wrapper to execute commands, respecting the --what-if flag.
run_cmd() {
  local cmd_desc="$1"
  shift
  if $WHAT_IF; then
    log "DRY RUN: Would execute: $cmd_desc -> $*"
  else
    log "EXECUTE: $cmd_desc"
    "$@"
  fi
}

# Add a command to the rollback stack.
add_rollback() {
  ROLLBACK_ACTIONS+=("$1")
}

# The main cleanup function, triggered on exit or error.
cleanup() {
  local exit_code=$?
  if [ ${#ROLLBACK_ACTIONS[@]} -gt 0 ]; then
    log_critical "\n🔥 An error occurred or script is exiting. Rolling back changes..."
    # Reverse the actions for proper LIFO rollback
    for ((i=${#ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
      log_critical "   - Rolling back: ${ROLLBACK_ACTIONS[i]}"
      eval "${ROLLBACK_ACTIONS[i]}" || log_critical "     - Warning: Rollback command failed."
    done
    log_critical "🔥 Rollback complete."
  fi
  # Clean up temporary file if it exists
  if [[ -n "$TEMP_FILE" && -f "$TEMP_FILE" ]]; then
    rm -f "$TEMP_FILE"
  fi
  # Stop the transcript-like logging
  exec 1>&- 2>&-
  wait
  log_critical "➡️ Log file created at: $LOG_FILE"
  exit $exit_code
}

# Set traps to call the cleanup function on ERR, EXIT, INT, TERM.
trap cleanup EXIT ERR INT TERM

# --- Main Logic ---
main() {
  # --- Argument Parsing ---
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --scope) SCOPE="$2"; shift 2 ;;
      --install-path) INSTALL_PATH="$2"; shift 2 ;;
      --force) FORCE=true; shift 1 ;;
      --silent) SILENT=true; shift 1 ;;
      --allow-destructive-cleanup) ALLOW_DESTRUCTIVE_CLEANUP=true; shift 1 ;;
      --what-if) WHAT_IF=true; shift 1 ;;
      -h|--help) usage; exit 0 ;;
      *) log_critical "Unknown option: $1"; usage; exit 1 ;;
    esac
  done

  # --- Transcript Logging ---
  mkdir -p "$LOG_DIR"
  # Redirect stdout and stderr to both the console and the log file
  exec > >(tee -a "$LOG_FILE") 2>&1

  log "➡️ Starting BWS CLI installation..."
  log "   - For detailed logs, see: $LOG_FILE"

  # --- Self-Elevation for Machine Scope ---
  if [[ "$SCOPE" == "Machine" && "$EUID" -ne 0 ]]; then
    log_critical "Machine scope requires root privileges. Attempting to re-run with sudo..."
    # Re-launch the script with all original arguments under sudo
    sudo -n "$0" "$@" || { log_critical "❌ Sudo failed. Please run the script with sudo for Machine scope."; exit 1; }
    exit $?
  elif [[ "$SCOPE" == "Machine" ]]; then
    log "   - Administrator privileges confirmed."
  fi

  # --- Determine OS, Arch, and Paths ---
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)
  case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) log_critical "❌ Unsupported architecture: $ARCH"; exit 1 ;;
  esac
  log "   - System detected: $OS-$ARCH"

  if [[ -z "$INSTALL_PATH" ]]; then
    if [[ "$SCOPE" == "User" ]]; then
      INSTALL_PATH="${HOME}/.local/bin"
    else # Machine
      INSTALL_PATH="/usr/local/bin"
    fi
  fi
  DESTINATION_FILE="${INSTALL_PATH}/bws"
  log "   - Installation path set to: '$INSTALL_PATH' (Scope: $SCOPE)"

  # --- Get Latest Release Info from GitHub ---
  log "   - Finding the latest release..."
  API_URL="https://api.github.com/repos/bitwarden/sdk/releases/latest"
  RELEASE_INFO=$(curl --silent --location --get "$API_URL")
  LATEST_VERSION=$(echo "$RELEASE_INFO" | jq -r '.tag_name')
  log "   - Latest version found: $LATEST_VERSION"

  ASSET_NAME="bws-$OS-$ARCH"
  CHECKSUM_ASSET_NAME="bws-checksums-sha256.txt"
  DOWNLOAD_URL=$(echo "$RELEASE_INFO" | jq -r ".assets[] | select(.name==\"$ASSET_NAME\") | .browser_download_url")
  CHECKSUM_URL=$(echo "$RELEASE_INFO" | jq -r ".assets[] | select(.name==\"$CHECKSUM_ASSET_NAME\") | .browser_download_url")
  if [[ -z "$DOWNLOAD_URL" || "$DOWNLOAD_URL" == "null" || -z "$CHECKSUM_URL" || "$CHECKSUM_URL" == "null" ]]; then
    log_critical "❌ Could not find required release assets on GitHub."
    exit 1
  fi
  
  # Checksum command availability
  SHASUM_CMD=""
  if command -v sha256sum &> /dev/null; then SHASUM_CMD="sha256sum"; elif command -v shasum &> /dev/null; then SHASUM_CMD="shasum -a 256"; fi
  if [[ -z "$SHASUM_CMD" ]]; then log_critical "❌ sha256sum or shasum command not found. Cannot verify checksums."; exit 1; fi

  # --- Download and Verification ---
  TEMP_FILE=$(mktemp "${TMPDIR:-/tmp}/bws_setup_XXXXXXXX.tmp")
  add_rollback "rm -f '$TEMP_FILE'"

  log "   - Downloading to temporary file: $TEMP_FILE"
  curl --progress-bar --location "$DOWNLOAD_URL" --output "$TEMP_FILE"

  log "   - Verifying file integrity (Checksum)..."
  CHECKSUMS=$(curl --silent --location "$CHECKSUM_URL")
  EXPECTED_HASH=$(echo "$CHECKSUMS" | grep "$ASSET_NAME" | awk '{print $1}')
  ACTUAL_HASH=$($SHASUM_CMD "$TEMP_FILE" | awk '{print $1}')
  if [[ "${EXPECTED_HASH,,}" != "${ACTUAL_HASH,,}" ]]; then
    log_critical "❌ Checksum mismatch! Expected '$EXPECTED_HASH' but got '$ACTUAL_HASH'."
    exit 1
  fi
  log "   - Checksum valid."

  # --- Installation ---
  if [[ ! -d "$INSTALL_PATH" ]]; then
    run_cmd "Create installation directory" mkdir -p "$INSTALL_PATH"
    add_rollback "rmdir '$INSTALL_PATH' 2>/dev/null || true"
  fi

  # Backup existing file if it exists, for rollback
  if [[ -f "$DESTINATION_FILE" ]]; then
    BACKUP_FILE="${DESTINATION_FILE}.bak"
    run_cmd "Backup existing bws" mv "$DESTINATION_FILE" "$BACKUP_FILE"
    add_rollback "mv '$BACKUP_FILE' '$DESTINATION_FILE' 2>/dev/null || true"
  fi

  run_cmd "Set executable permissions" chmod +x "$TEMP_FILE"
  run_cmd "Move file to destination" mv "$TEMP_FILE" "$DESTINATION_FILE"
  
  # Clear the TEMP_FILE variable as it's been moved, not just deleted
  TEMP_FILE="" 

  log "✅ BWS CLI installed/updated successfully!"
  log "   - Verifying installation..."
  "$DESTINATION_FILE" --version
}

# --- Script Entry Point ---
# Pass all script arguments to the main function
main "$@"
