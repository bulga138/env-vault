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
	@echo "Compiling for $(OS)..."
	@$(MKDIR_P)
	@go build $(LDFLAGS) -o $(BINARY_FULL_NAME) .
	@echo "Binary available at: $(BINARY_FULL_NAME)"
# Clean target to remove build artifacts
clean:
	@echo "Cleaning up build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Working directory cleaned."
# Target to run the application locally
run: build
	@echo "Running the application locally..."
	@$(BINARY_FULL_NAME) $(ARGS)
# Target to manage project dependencies
tidy:
	@echo "Tidying up Go modules..."
	@go mod tidy

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

save-bw:
	@make run ARGS="--use-bitwarden save"

get-bw:
	@make run ARGS="--use-bitwarden get"

setup-bw:
	@echo "=> 1. Syncing all secrets from cloud..."
	@make run ARGS="sync-load"
	@echo "\n=> 2. Getting .env for current repo with BitWarden..."
	@make run ARGS="--use-bitwarden get"

# Declare non-file targets to avoid conflicts
.PHONY: all clean run tidy login save get generate-pass sync-save sync-load save-bw get-bw setup-bw