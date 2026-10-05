# Detect if we're in a Windows-native environment
ifeq ($(OS), Windows_NT)
  ifeq ($(MSYSTEM),)
    # Native Windows (cmd.exe or PowerShell via make)
    USE_COLOR = no
  else
    # MSYS2, Git Bash, etc.
    USE_COLOR = yes
  endif
else
  # Real Unix/Linux/macOS
  USE_COLOR = yes
endif

# --- Color & Style Definitions ---
# These are ANSI escape codes for coloring the output.
CYAN   := \033[36m
GREEN  := \033[32m
YELLOW := \033[33m
RESET  := \033[0m

# --- Logging Macros ---
# These macros print messages with different severity levels.
# They use colors if supported by the terminal.
ifeq ($(USE_COLOR), yes)
  log_info    = @printf '%b\n' "$(CYAN)[INFO]$(RESET)  $(1)"
  log_warn    = @printf '%b\n' "$(YELLOW)[WARN]$(RESET)  $(1)"
else
  log_info    = @echo [INFO] $(1)
  log_warn    = @echo [WARN] $(1)
endif


# --- Configuration Variables ---
# Binary name
BINARY_NAME=env-vault
# Binary directory
BUILD_DIR=build
# Flags to optimize the binary for production
LDFLAGS=-ldflags="-s -w"
# Find all .go files in the current directory
GO_FILES := $(wildcard *.go)

# --- OS Detection ---
# By default, assume a Unix-like OS
BINARY_FULL_NAME=$(BUILD_DIR)/$(BINARY_NAME)
# If the operating system is Windows, adjust the binary name and mkdir command
ifeq ($(OS), Windows_NT)
	BINARY_FULL_NAME := $(BUILD_DIR)/$(BINARY_NAME).exe
	MKDIR_P = if not exist $(subst /,\,$(BUILD_DIR)) mkdir $(subst /,\,$(BUILD_DIR))
else
	MKDIR_P = mkdir -p $(BUILD_DIR)
endif

# --- Rules ---
# The default target that runs with just 'make'
all: build

# The 'build' target is now just an alias for the real binary target.
build: $(BINARY_FULL_NAME)

# This is the main build rule.
# It only runs if the binary doesn't exist or if any .go file is newer than the binary.
$(BINARY_FULL_NAME): $(GO_FILES)
	$(call log_info, Compiling for $(OS)...)
	@$(MKDIR_P)
	@go build $(LDFLAGS) -o $(BINARY_FULL_NAME) .
	$(call log_info, Binary available at: $(BINARY_FULL_NAME))

# Clean target to remove build artifacts
clean:
	$(call log_info, Cleaning up build artifacts...)
	@rm -rf $(BUILD_DIR)
	$(call log_info, Working directory cleaned.)

# Target to run the application locally
run: build
	$(call log_info, Running: $(BINARY_FULL_NAME) $(ARGS))
	@$(BINARY_FULL_NAME) $(ARGS)

# Target to manage project dependencies
tidy:
	$(call log_info, Tidying up Go modules...)
	@go mod tidy

# Unit + e2e (no external services)
test:
	$(call log_info, Running unit and e2e tests...)
	@go test ./... -count=1

# Integration tests against real Firestore / Bitwarden (uses .env credentials)
test-integration:
	$(call log_info, Running integration tests (Firestore + Bitwarden)...)
	@go test ./... -count=1 -run Integration -v

# --- Command Shortcuts ---
login:
	@make run ARGS=login

save:
	@make run ARGS=save

get:
	@make run ARGS=get

generate-pass:
	@make run ARGS=generate-pass

sync-save:
	@make run ARGS=sync-save

sync-load:
	@make run ARGS=sync-load

verify:
	@make run ARGS=verify

save-bw:
	@make run ARGS="--use-bitwarden save"

get-bw:
	@make run ARGS="--use-bitwarden get"

setup-bw:
	$(call log_info, => 1. Syncing all secrets from cloud...)
	@make run ARGS="sync-load"
	$(call log_info, \n=> 2. Getting .env for current repo with BitWarden...)
	@make run ARGS="--use-bitwarden get"

# Declare non-file targets to avoid conflicts
.PHONY: all build clean run tidy test test-integration login save get generate-pass sync-save sync-load save-bw get-bw setup-bw verify

