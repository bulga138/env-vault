package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bulga138/env-vault/providers"
	"github.com/bulga138/env-vault/utils"
	"github.com/joho/godotenv"
)

var (
	useBitWarden      = flag.Bool("use-bitwarden", false, "Use BitWarden to fetch the master password")
	useKeyring        = flag.Bool("use-keyring", false, "Use OS keyring to fetch/store the master password")
	repoOverride      = flag.String("repo", "", "Specify repository name (e.g., 'user/repo')")
	auditFilterAction = flag.String("audit-action", "", "Filter audit log by action")
	auditFilterUser   = flag.String("audit-user", "", "Filter audit log by user")
	auditLimit        = flag.Int("audit-limit", 0, "Limit audit entries")
	syncStrategy      = flag.String("sync-strategy", "newest", "Conflict strategy: newest, remote, local, skip")
	forceFlag         = flag.Bool("force", false, "Force overwrite without confirmation")
)

type AuditLog struct {
	Timestamp    time.Time `json:"timestamp"`
	Action       string    `json:"action"`
	RepoName     string    `json:"repo_name"` // hashed for privacy
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	User         string    `json:"user,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
}

func logAudit(action, repoName string, success bool, errorMsg string) {
	audit := AuditLog{
		Timestamp:    time.Now(),
		Action:       action,
		RepoName:     utils.GetRepoKey(repoName),
		Success:      success,
		ErrorMessage: errorMsg,
		User:         os.Getenv("USER"),
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		utils.Warn("Could not get config directory for audit log: %v", err)
		return
	}

	auditDir := filepath.Join(configDir, "env-vault", "audit")
	if err := os.MkdirAll(auditDir, 0700); err != nil {
		utils.Warn("Could not create audit directory: %v", err)
		return
	}

	auditFile := filepath.Join(auditDir, "audit.log")

	f, err := os.OpenFile(auditFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		utils.Warn("Could not open audit log file: %v", err)
		return
	}
	defer f.Close()

	auditJSON, err := json.Marshal(audit)
	if err != nil {
		utils.Warn("Could not marshal audit log: %v", err)
		return
	}

	if _, err := f.WriteString(string(auditJSON) + "\n"); err != nil {
		utils.Warn("Could not write to audit log: %v", err)
	}
}

func main() {
	utils.InitLogging()

	configPath := os.Getenv("ENV_VAULT_CONFIG_PATH")
	if configPath != "" {
		err := godotenv.Load(configPath)
		if err != nil {
			utils.Warn("Error loading config file from ENV_VAULT_CONFIG_PATH: %v", err)
		} else {
			utils.Info("Configuration loaded from: %s", configPath)
		}
	}

	err := godotenv.Load()
	if err != nil {
		utils.Warn("Error loading .env file, proceeding without it. Error: %v", err)
	}

	flag.Parse()

	if flag.NArg() < 1 {
		printHelp()
		return
	}
	command := flag.Arg(0)

	switch command {
	case "save":
		handleSave()
	case "get":
		handleGet()
	case "generate-pass":
		handleGeneratePassword()
	case "sync-save":
		handleSyncSave()
	case "sync-load":
		handleSyncLoad()
	case "verify":
		handleVerify()
	case "rotate-pass":
		handleRotatePassword()
	case "setup-config":
		handleSetupConfig()
	case "users":
		handleUsers()
	case "acl":
		handleACL()
	case "audit":
		handleAudit()
	case "keyring":
		handleKeyring()
	case "mnemonic":
		handleMnemonic()
	case "backup":
		handleBackup()
	case "restore":
		handleRestore()
	case "init", "wizard":
		handleInit()
	case "doctor":
		handleDoctor()
	case "p2p", "peer", "relay":
		handleP2P()
	case "admin":
		if err := runAdminTUI(); err != nil {
			fmt.Printf("Admin dashboard error: %v\n", err)
		}
	default:
		printHelp()
	}
}

func handleSave() {
	utils.Info("Saving .env file...")
	repoName, err := getCurrentRepoName()
	if err != nil {
		logAudit("save", "", false, fmt.Sprintf("Failed to get repository name: %v", err))
		utils.Fatal("Failed to get repository name: %v", err)
		return
	}
	if hasACL(repoName) && !checkPermission(currentUser(), repoName, "write") {
		logAudit("save", repoName, false, "Permission denied (write)")
		utils.Fatal("Permission denied: you do not have write access to this repository.")
		return
	}
	hashedRepoName := utils.GetRepoKey(repoName)

	vault, err := loadVault()
	if err != nil {
		logAudit("save", repoName, false, fmt.Sprintf("Error loading vault: %v", err))
		utils.Fatal("Error loading vault: %v", err)
		return
	}

	var password []byte
	var passErr error
	var passwordGenerated bool
	if *useBitWarden {
		secretProvider, err := providers.GetSecretProvider()
		if err != nil {
			utils.Fatal(ErrNoSecretProvider.Error(), ":%v", err)
			return
		}
		password, passErr = secretProvider.Get(hashedRepoName)
	} else if p := os.Getenv("ENV_VAULT_PASSWORD"); p != "" {
		password = []byte(p)
	} else if *useKeyring {
		if p, err := keyringGet(hashedRepoName); err == nil {
			password = p
		} else {
			password, passErr = promptForPassword("Enter your master password to store in keyring: ")
			if passErr == nil {
				_ = keyringSet(hashedRepoName, password)
			}
		}
	} else if _, exists := vault[hashedRepoName]; exists {
		password, passErr = promptForPassword("Enter your master password again: ")
	} else {
		utils.Info("\nNo secret found for this repository locally. Generating a new secure password...")
		passStr, genErr := generateSecurePassword(16)
		if genErr != nil {
			passErr = genErr
		} else {
			password = []byte(passStr)
			passwordGenerated = true
			utils.Info("\n✅ New master password generated for this repository:")
			fmt.Printf("\n%s\n\n", string(password))
			utils.Info("---------------------------------------------------------------")
		}
	}

	if passErr != nil {
		defer secureZero(password)
		logAudit("save", repoName, false, fmt.Sprintf("Password error: %v", passErr))
		utils.Fatal("Password error: %v", passErr)
		return
	}

	if password != nil {
	}

	if passwordGenerated {
	// Bitwarden Secrets Manager
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) == "y" {
			secretProvider, err := providers.GetSecretProvider()
			if err != nil {
				utils.Fatal(ErrNoSecretProvider.Error(), ":%v", err)
				return
			}
			if err := secretProvider.Set(hashedRepoName, password); err != nil {
				utils.Fatal("Error saving to %s: %v", secretProvider.Name(), err)
				return
			}
	// Bitwarden Secrets Manager
		} else {
			utils.Warn("Master password not saved to BitWarden. Make sure to remember it, as you will need it to decrypt your .env files.")
		}
	}

	content, err := os.ReadFile(".env")
	if err != nil {
		logAudit("save", repoName, false, fmt.Sprintf("Error reading '.env' file: %v", err))
		utils.Fatal("Error reading '.env' file: %v", err)
		return
	}

	if err := saveEnv(repoName, content, password); err != nil {
		logAudit("save", repoName, false, fmt.Sprintf("Error saving .env to vault: %v", err))
		utils.Fatal("Error saving .env to vault: %v", err)
		return
	}
	if *useKeyring {
		_ = keyringSet(hashedRepoName, password)
	}
	logAudit("save", repoName, true, "")
	utils.Info(".env file for '%s' safely saved!", repoName)
}

func handleGet() {
	utils.Info("Getting .env file...")

	outputFilename := ".env"
	if _, err := os.Stat(".env"); err == nil {
		overrideFilename := os.Getenv("ENV_VAULT_EXISTING_FILENAME")
		if overrideFilename != "" {
			outputFilename = overrideFilename
		} else {
			outputFilename = ".env.recovered"
		}
		utils.Warn("Found local .env file. It will be saved as '%s' to avoid overwriting.", outputFilename)
	}

	repoName, err := getCurrentRepoName()
	if err != nil {
		logAudit("get", "", false, fmt.Sprintf("Failed to get repository name: %v", err))
		utils.Fatal("Error getting repo name: %v", err)
		return
	}
	if hasACL(repoName) && !checkPermission(currentUser(), repoName, "read") {
		logAudit("get", repoName, false, "Permission denied (read)")
		utils.Fatal("Permission denied: you do not have read access to this repository.")
		return
	}
	hashedRepoName := utils.GetRepoKey(repoName)

	var password []byte
	var passErr error

	if *useBitWarden {
		secretProvider, err := providers.GetSecretProvider()
		if err != nil {
			utils.Fatal(ErrNoSecretProvider.Error(), ":%v", err)
			return
		}
		password, passErr = secretProvider.Get(hashedRepoName)
		if passErr != nil {
			utils.Fatal("Failed to fetch master password from %s: %v", secretProvider.Name(), passErr)
			return
		}
	} else if p := os.Getenv("ENV_VAULT_PASSWORD"); p != "" {
		password = []byte(p)
	} else {
		if *useKeyring {
			if p, err := keyringGet(hashedRepoName); err == nil {
				password = p
			} else {
				password, passErr = promptForPassword("Enter your master password to store in keyring: ")
				if passErr == nil {
					_ = keyringSet(hashedRepoName, password)
				}
			}
		} else {
			password, passErr = promptForPassword("Enter your master password to decrypt: ")
		}
	}
	if passErr != nil {
		utils.Fatal("Error reading password: %v", passErr)
		return
	}

	if password != nil {
	}

	content, err := getEnv(repoName, password)
	if err != nil {
		logAudit("get", repoName, false, fmt.Sprintf("Error getting .env from vault: %v", err))
		utils.Fatal("Error getting .env from vault: %v", err)
		return
	}

	comment := fmt.Sprintf("# Sourced from env-vault for %s\n\n", repoName)
	finalContent := append([]byte(comment), content...)

	if err := os.WriteFile(outputFilename, finalContent, 0600); err != nil {
		utils.Fatal("Error writing file '%s': %v", outputFilename, err)
		return
	}

	utils.Info(".env file for '%s' recovered and saved as '%s'!", repoName, outputFilename)
	logAudit("get", repoName, true, "")
}

func handleGeneratePassword() {
	const passwordLength = 16

	utils.Info("🔒 Generating a new %d-character secure password...", passwordLength)
	password, err := generateSecurePassword(passwordLength)
	if err != nil {
		utils.Fatal("Error: Could not generate password: %v", err)
		return
	}
	utils.Info("✅ Your new secure password is:")
	fmt.Printf("\n%s\n\n", password)
	utils.Info("Copy this password and store it in a safe place (like a password manager).")
}

func handleRotatePassword() {
	utils.Info("🔄 Rotating master password for repository...")

	var repoName string
	var err error

	if *repoOverride != "" {
		repoName = *repoOverride
	} else {
		repoName, err = getCurrentRepoName()
		if err != nil {
			logAudit("rotate-pass", "", false, fmt.Sprintf("Failed to detect repository name: %v", err))
			utils.Fatal("Failed to detect repository name: %v", err)
			return
		}
	}

	vault, err := loadVault()
	if err != nil {
		logAudit("rotate-pass", repoName, false, fmt.Sprintf("Error loading vault: %v", err))
		utils.Fatal("Error loading vault: %v", err)
		return
	}

	hashedRepoName := utils.GetRepoKey(repoName)
	if _, exists := vault[hashedRepoName]; !exists {
		logAudit("rotate-pass", repoName, false, "No entry found for repository in vault.")
		utils.Fatal("No entry found for repository '%s' in vault. Run 'save' first.", repoName)
		return
	}

	var oldPassword []byte
	if *useBitWarden {
		secretProvider, err := providers.GetSecretProvider()
		if err != nil {
			utils.Fatal(ErrNoSecretProvider.Error(), ":%v", err)
			return
		}
		oldPassword, err = secretProvider.Get(hashedRepoName)
		if err != nil {
			utils.Fatal("Failed to get password from secret provider: %v", err)
			return
		}
	} else if p := os.Getenv("ENV_VAULT_PASSWORD"); p != "" {
		oldPassword = []byte(p)
	} else {
		if *useKeyring {
			if p, err2 := keyringGet(hashedRepoName); err2 == nil {
				oldPassword = p
			} else {
				oldPassword, err = promptForPassword("Enter CURRENT master password to decrypt: ")
				if err != nil {
					utils.Fatal("Failed to read current password: %v", err)
					return
				}
			}
		} else {
			oldPassword, err = promptForPassword("Enter CURRENT master password to decrypt: ")
			if err != nil {
				utils.Fatal("Failed to read current password: %v", err)
				return
			}
		}
	}

	if oldPassword == nil {
		utils.Fatal("Failed to obtain current password")
		return
	}

	plaintext, err := getEnv(repoName, oldPassword)
	if err != nil {
		logAudit("rotate-pass", repoName, false, fmt.Sprintf("Decryption failed: %v", err))
		utils.Fatal(ErrDecryptionFailed.Error(), ":%v", err)
		return
	}

	var newPassword []byte
	var secretProvider providers.SecretProvider

	utils.Info("\n🔐 Setting NEW master password...")
	if p := os.Getenv("ENV_VAULT_NEW_PASSWORD"); p != "" {
		newPassword = []byte(p)
	} else if promptYesNo("Generate a secure password? (y/n): ") {
		passStr, err := generateSecurePassword(16)
		if err != nil {
			utils.Fatal("Password generation failed: %v", err)
			return
		}
		newPassword = []byte(passStr)
		utils.Info("✅ New secure password generated:")
		fmt.Printf("\n%s\n\n", string(newPassword))
	} else {
		p1, err := promptForPassword("Enter new master password: ")
		if err != nil {
			utils.Fatal("Failed to read new password: %v", err)
			return
		}
		p2, err := promptForPassword("Confirm new master password: ")
		if err != nil {
			utils.Fatal("Failed to read confirmation: %v", err)
			return
		}
		if !bytes.Equal(p1, p2) {
			utils.Fatal("Passwords do not match.")
			return
		}
		newPassword = p1
	}

	if *useBitWarden {
		secretProvider, err = providers.GetSecretProvider()
		if err != nil {
			utils.Fatal(ErrNoSecretProvider.Error(), ":%v", err)
			return
		}
		if err := secretProvider.Set(hashedRepoName, newPassword); err != nil {
			utils.Fatal("Error saving new password to %s (vault NOT re-encrypted): %v", secretProvider.Name(), err)
			return
		}
		utils.Info("✅ Master password updated in %s.", secretProvider.Name())
	}

	if err := saveEnv(repoName, plaintext, newPassword); err != nil {
		logAudit("rotate-pass", repoName, false, fmt.Sprintf("Failed to re-encrypt: %v", err))
		utils.Fatal("Failed to re-encrypt with new password: %v", err)
		return
	}

	if *useKeyring {
		_ = keyringSet(hashedRepoName, newPassword)
	}
	secureZero(plaintext)

	logAudit("rotate-pass", repoName, true, "")
	utils.Success("✅ Master password rotated successfully for '%s'!", repoName)
}

func handleVerify() {
	utils.Info("🔍 Starting integrity verification for all vault entries...")

	getPasswordFunc := func(hashedKey string) ([]byte, error) {
		if *useBitWarden {
			secretProvider, err := providers.GetSecretProvider()
			if err != nil {
				return nil, fmt.Errorf("no secret provider configured: %w", err)
			}
			return secretProvider.Get(hashedKey)
		} else if p := os.Getenv("ENV_VAULT_PASSWORD"); p != "" {
			return []byte(p), nil
		}
		prompt := fmt.Sprintf("Enter master password for key %s (required for HMAC check): ", hashedKey)
		return promptForPassword(prompt)
	}

	results, err := VerifyAll(getPasswordFunc)
	if err != nil {
		utils.Fatal("Error loading vault for verification: %v", err)
		return
	}

	successCount := 0
	failureCount := 0

	utils.Info("\n--- Verification results ---")
	for hashedKey, err := range results {
		status := "✅ OK"
		if err != nil {
			status = fmt.Sprintf("❌ FAILED: %v", err)
			failureCount++
			logAudit("verify", hashedKey, false, err.Error())
		} else {
			successCount++
			logAudit("verify", hashedKey, true, "")
		}
		fmt.Printf("Key %s: %s\n", hashedKey, status)
	}

	utils.Info("\n--- Summary ---")
	utils.Info("Success: %d", successCount)
	utils.Info("Failures: %d", failureCount)

	if failureCount > 0 {
		utils.Warn("\nATTENTION: verification failures detected. Data may have been tampered with, or the master password was wrong.")
	} else {
		utils.Success("🎉 Integrity verification completed. All secrets are valid.")
	}
}

func handleSetupConfig() {
	utils.Info("Starting env-vault configuration setup...")
	configDir, err := os.UserConfigDir()
	if err != nil {
		utils.Fatal("Error: Could not determine user configuration directory: %v", err)
		return
	}
	appConfigDir := filepath.Join(configDir, "env-vault")

	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		utils.Fatal("Error creating config directory: %v", err)
		return
	}

	configFilePath := filepath.Join(appConfigDir, "env-vault.env")
	utils.Info("Configuration file path: %s", configFilePath)
	pepper := promptForInput("Enter a strong, random string for ENV_VAULT_PEPPER (required for security, leave empty to skip): ")
	bwsProjectID := promptForInput("Enter your BWS_PROJECT_ID (Bitwarden, leave empty to skip): ")
	googleCreds := promptForInput("Enter the FULL PATH to your Google Service Account Key JSON (for Firestore, leave empty to skip): ")

	content := fmt.Sprintf(`
# env-vault configuration file. DO NOT COMMIT TO GIT.
# ---------------------------------------------------

# Core Security Settings
VAULT_FILE_PATH=%s/vault.json
ENV_VAULT_PEPPER=%s

	// Bitwarden Secrets Manager
BWS_PROJECT_ID=%s

# Google Firestore/Cloud Sync
GOOGLE_APPLICATION_CREDENTIALS=%s

# Logging Configuration (Defaults)
LOG_OUTPUT=TERMINAL
LOG_FILE=%s/env-vault.log
LOG_LEVEL=INFO
`,
		appConfigDir,
		pepper,
		bwsProjectID,
		googleCreds,
		appConfigDir,
	)

	if err := os.WriteFile(configFilePath, []byte(content), 0600); err != nil {
		utils.Fatal("Error writing config file: %v", err)
		return
	}

	utils.Success("\nConfiguration file created successfully at:")
	fmt.Printf("   %s\n\n", configFilePath)

	configCommand := fmt.Sprintf("export ENV_VAULT_CONFIG_PATH=\"%s\"", configFilePath)
	shellName := getShellName()

	utils.Info("Final Step: Load Configuration")
	utils.Warn("You need to add the following line to your shell startup file.")

	switch shellName {
	case "powershell":
		configCommand = fmt.Sprintf("$env:ENV_VAULT_CONFIG_PATH=\"%s\"", configFilePath)
		utils.Info("Use this command (PowerShell syntax):")
		fmt.Printf("\n   %s\n", configCommand)
	case "bash", "zsh":
		utils.Info("Use this command (Bash/Zsh syntax):")
		fmt.Printf("\n   %s\n", configCommand)
	default:
		utils.Warn("\nCould not automatically detect your shell. Use one of the following commands:")
		fmt.Printf("\n   Bash/Zsh: export ENV_VAULT_CONFIG_PATH=\"%s\"", configFilePath)
		fmt.Printf("\n   PowerShell: $env:ENV_VAULT_CONFIG_PATH=\"%s\"", configFilePath)
		fmt.Printf("\n   CMD: set ENV_VAULT_CONFIG_PATH=\"%s\"\n", configFilePath)
	}

	shellConfigPath := getShellConfigPath()
	if shellConfigPath != "" && shellName != "powershell" {
	utils.Info("Final Step: Load Configuration")
		utils.Warn("You need to add the following line to your shell startup file:")

		fmt.Printf("\nexport ENV_VAULT_CONFIG_PATH=\"%s\"\n", configFilePath)

		utils.Info("\nWe will now attempt to open your shell config file (%s) in your default editor.", shellConfigPath)
		utils.Info("Please copy and paste the 'export' line above into the file.")

		if err := openFileInEditor(shellConfigPath); err != nil {
			utils.Warn("\nFailed to automatically open shell config file: %v", err)
			utils.Warn("Please open %s manually and add the 'export' line.", shellConfigPath)
		} else {
			utils.Success("\nShell config file opened. Complete the step and then restart your terminal.")
		}
	} else if shellName != "powershell" {
	utils.Info("Final Step: Load Configuration")
		utils.Warn("Could not automatically detect your shell configuration file.")
		utils.Warn("Please add the following line to your shell's startup file (e.g., ~/.bashrc, ~/.zshrc):")
		fmt.Printf("\nexport ENV_VAULT_CONFIG_PATH=\"%s\"\n\n", configFilePath)
	}

	utils.Info("Then, restart your terminal or run 'source [your shell config file]'.")
	utils.Success("Setup complete! You can now run 'env-vault save' in any Git repository.")
}

func handleUsers() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: env-vault users <list|add|remove> [args]")
		return
	}
	sub := flag.Arg(1)
	switch sub {
	case "list":
		users, err := ListUsers()
		if err != nil {
			utils.Fatal("Error listing users: %v", err)
			return
		}
		if len(users) == 0 {
			utils.Info("No users registered.")
			return
		}
		for _, u := range users {
			fmt.Printf("%-20s %-12s %s\n", u.Name, u.Role, u.CreatedAt.Format(time.RFC3339))
		}
		logAudit("users-list", "", true, "")
	case "add":
		if flag.NArg() < 3 {
			fmt.Println("Usage: env-vault users add <name> [role]")
			return
		}
		name := flag.Arg(2)
		role := "member"
		if flag.NArg() >= 4 {
			role = flag.Arg(3)
		}
		if err := AddUser(name, role); err != nil {
			logAudit("users-add", "", false, err.Error())
			utils.Fatal("Error adding user: %v", err)
			return
		}
		logAudit("users-add", "", true, "")
		utils.Success("User %q added with role %q.", name, role)
	case "remove":
		if flag.NArg() < 3 {
			fmt.Println("Usage: env-vault users remove <name>")
			return
		}
		name := flag.Arg(2)
		if err := RemoveUser(name); err != nil {
			logAudit("users-remove", "", false, err.Error())
			utils.Fatal("Error removing user: %v", err)
			return
		}
		logAudit("users-remove", "", true, "")
		utils.Success("User %q removed.", name)
	default:
		fmt.Println("Unknown users subcommand. Use list, add, or remove.")
	}
}

func handleACL() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: env-vault acl <grant|revoke|list> [args]")
		return
	}
	sub := flag.Arg(1)
	switch sub {
	case "list":
		entries, err := listACL()
		if err != nil {
			utils.Fatal("Error listing ACL: %v", err)
			return
		}
		if len(entries) == 0 {
			utils.Info("No ACL entries.")
			return
		}
		for _, e := range entries {
			fmt.Printf("%-20s %-44s %s\n", e.User, e.RepoKey, e.Permission)
		}
	case "grant":
		if flag.NArg() < 5 {
			fmt.Println("Usage: env-vault acl grant <user> <repo-name> <read|write|admin>")
			return
		}
		user := flag.Arg(2)
		repo := flag.Arg(3)
		perm := flag.Arg(4)
		if err := grantACL(user, repo, perm); err != nil {
			logAudit("acl-grant", repo, false, err.Error())
			utils.Fatal("Error granting ACL: %v", err)
			return
		}
		logAudit("acl-grant", repo, true, "")
		utils.Success("Granted %s to %s on %s.", perm, user, repo)
	case "revoke":
		if flag.NArg() < 4 {
			fmt.Println("Usage: env-vault acl revoke <user> <repo-name>")
			return
		}
		user := flag.Arg(2)
		repo := flag.Arg(3)
		if err := revokeACL(user, repo); err != nil {
			logAudit("acl-revoke", repo, false, err.Error())
			utils.Fatal("Error revoking ACL: %v", err)
			return
		}
		logAudit("acl-revoke", repo, true, "")
		utils.Success("Revoked access for %s on %s.", user, repo)
	default:
		fmt.Println("Unknown acl subcommand. Use grant, revoke, or list.")
	}
}

func handleBanner() {
	name := "beNotAfraid.txt"
	if flag.NArg() > 1 {
		name = flag.Arg(1)
	}
	candidates := []string{name}
	if name == "beNotAfraid.txt" {
		candidates = []string{"beNotAfraid.txt"}
	}

	var data []byte
	var used string
	var lastErr error
	for _, c := range candidates {
		b, err := os.ReadFile(c)
		if err == nil {
			data = b
			used = c
			break
		}
		lastErr = err
	}

	if data == nil {
		fmt.Printf("No banner file found (%v): %v\n", candidates, lastErr)
		return
	}
	fmt.Printf("Banner: %s\n", used)
	utils.PrintBanner(used, string(data))
}

func handleDoctor() {
	utils.Info("env-vault doctor")
	if repo, err := getCurrentRepoName(); err != nil {
		utils.Warn("Git repo detection: %v", err)
	} else {
		utils.Success("Git repo: %s", repo)
	}
	if _, err := exec.LookPath("git"); err != nil {
		utils.Warn("git not found in PATH")
	} else {
		utils.Success("git available")
	}
	if _, err := exec.LookPath("bws"); err != nil {
		utils.Warn("bws (Bitwarden CLI) not found (optional unless using --use-bitwarden)")
	} else {
		utils.Success("bws available")
	}
	if configPath := os.Getenv("ENV_VAULT_CONFIG_PATH"); configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			utils.Warn("ENV_VAULT_CONFIG_PATH set but file missing: %s", configPath)
		} else {
			utils.Success("Config file: %s", configPath)
		}
	} else {
		utils.Warn("ENV_VAULT_CONFIG_PATH not set (run 'env-vault init')")
	}
	configDir, _ := os.UserConfigDir()
	vaultPath := filepath.Join(configDir, "env-vault", "vault.json")
	if _, err := os.Stat(vaultPath); err != nil {
		utils.Info("Vault not initialized yet: %s", vaultPath)
	} else {
		utils.Success("Vault exists: %s", vaultPath)
	}
}

func handleInit() {
	handleSetupConfig()
	utils.Info("Next steps:")
	utils.Info("  1. cd into a git repository")
	utils.Info("  2. env-vault save   (or: env-vault save --use-keyring)")
	utils.Info("  3. env-vault verify")
}

func printHelp() {
	fmt.Println("Welcome to env-vault 🛡️")
	fmt.Println("Usage: env-vault [OPTION] COMMAND")
	fmt.Println("\nCOMMANDS:")
	fmt.Println("  save          - Encrypt and save the current repository's .env file.")
	fmt.Println("  get           - Retrieve and decrypt the .env file for the current repository.")
	fmt.Println("  generate-pass - Generate a new secure master password.")
	fmt.Println("  sync-save     - Save the current repo's secret to Firestore.")
	fmt.Println("  sync-load     - Load all secrets from Firestore to vault.json.")
	fmt.Println("  users         - Manage vault users (list/add/remove).")
	fmt.Println("  acl           - Manage per-repo access control.")
	fmt.Println("  audit         - View audit logs.")
	fmt.Println("  keyring       - Manage OS keyring entries.")
	fmt.Println("  mnemonic      - Generate or recover a BIP39 recovery phrase.")
	fmt.Println("  backup        - Create a rotated backup of the vault.")
	fmt.Println("  restore       - Restore a vault backup.")
	fmt.Println("  p2p           - Peer-to-peer sync via relay/rendezvous.")
	fmt.Println("  init          - Interactive configuration wizard.")
	fmt.Println("  doctor        - Check environment and setup status.")
	fmt.Println("  admin         - Terminal dashboard for users, ACL, and audit logs.")
	fmt.Println("  rotate-pass   - Rotate the master password for the current (or specified) repository.")
	fmt.Println("  verify        - Verify the integrity of all secrets in vault.json (requires master password).")
	fmt.Println("  setup-config  - Interactive setup for env-vault configuration.")
	fmt.Println("\nOPTIONS:")
	fmt.Println("  -use-bitwarden  - Use BitWarden to get the master password instead of prompting.")
	fmt.Println("  -use-keyring    - Use OS keyring to get/store the master password.")
	fmt.Println("  -repo=NAME      - Specify repository name (e.g., 'user/repo') for rotate-pass.")
	fmt.Println("\nENVIRONMENT:")
	fmt.Println("  ENV_VAULT_PASSWORD      - Master password for non-interactive save/get/verify/rotate.")
	fmt.Println("  ENV_VAULT_NEW_PASSWORD  - New master password for non-interactive rotate-pass.")
	fmt.Println("\nMake sure to set the necessary environment variables before using the tool.")
	fmt.Println("For more information, visit: https://github.com/bulga138/env-vault")
}
