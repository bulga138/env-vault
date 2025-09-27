package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bulga138/env-vault/providers"
	"github.com/bulga138/env-vault/utils"
	"github.com/joho/godotenv"
)

var (
	useBitWarden = flag.Bool("use-bitwarden", false, "Use BitWarden to fetch the master password")
	repoOverride = flag.String("repo", "", "Specify repository name (e.g., 'user/repo')")
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

	// Get audit log file path
	configDir, err := os.UserConfigDir()
	if err != nil {
		utils.Warn("Could not get config directory for audit log: %v", err)
		return
	}

	auditDir := filepath.Join(configDir, "env-vault", "audit")
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		utils.Warn("Could not create audit directory: %v", err)
		return
	}

	auditFile := filepath.Join(auditDir, "audit.log")

	// Append to audit log
	f, err := os.OpenFile(auditFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	default:
		printHelp()
	}
}

func handleSave() {
	utils.Info("Saving .env file...")
	repoName, err := getCurrentRepoName()
	if err != nil {
		logAudit("save", "", false, fmt.Sprintf("Failed to get repository name: %v", err))
		utils.Error("Failed to get repository name: %v", err)
		return
	}
	hashedRepoName := utils.GetRepoKey(repoName)

	vault, err := loadVault()
	if err != nil {
		utils.Error("Error loading vault: %v", err)
		return
	}

	var password []byte
	var passErr error
	var passwordGenerated bool
	if *useBitWarden {
		secretProvider, err := providers.GetSecretProvider()
		if err != nil {
			utils.Error(ErrNoSecretProvider.Error(), ":%v", err)
			return
		}
		password, passErr = secretProvider.Get(hashedRepoName)
	} else {
		// Use hashed repo name for local vault check
		if _, exists := vault[hashedRepoName]; exists {
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
	}

	if passErr != nil {
		defer secureZero(password)
		utils.Error("Password error: %v", passErr)
		return
	}

	if passwordGenerated {
		utils.Info("Do you wanna save this new password in BitWarden Secrets Manager? (y/n): ")
		var response string
		fmt.Scanln(&response)

		if strings.ToLower(response) == "y" {
			secretProvider, err := providers.GetSecretProvider() // Get the provider instance
			if err != nil {
				utils.Error(ErrNoSecretProvider.Error(), ":%v", err)
				return
			}
			if err := secretProvider.Set(hashedRepoName, password); err != nil { // Call method on instance
				utils.Error("Error saving to %s: %v", secretProvider.Name(), err) // Call method on instance
				return
			}
			utils.Info("Master password saved to BitWarden Secrets Manager.")
		} else {
			utils.Warn("Master password not saved to BitWarden. Make sure to remember it, as you will need it to decrypt your .env files.")
		}
	}

	content, err := os.ReadFile(".env")
	if err != nil {
		utils.Error("Error reading '.env' file: %v", err)
		return
	}

	if err := saveEnv(repoName, content, password); err != nil {
		logAudit("save", repoName, false, fmt.Sprintf("Error saving .env to vault: %v", err))
		utils.Error("Error saving .env to vault: %v", err)
		return
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
		utils.Error("Error getting repo name: %v", err)
		return
	}
	hashedRepoName := utils.GetRepoKey(repoName)

	var password []byte
	var passErr error

	if *useBitWarden {
		secretProvider, err := providers.GetSecretProvider()
		if err != nil {
			utils.Error(ErrNoSecretProvider.Error(), ":%v", err)
			return
		}
		password, passErr = secretProvider.Get(hashedRepoName)
		if passErr != nil {
			utils.Error("Failed to fetch master password from %s: %v", secretProvider.Name(), passErr)
			return
		}
	} else {
		password, passErr = promptForPassword("Enter your master password to decrypt: ")
	}
	if passErr != nil {
		utils.Error("Error reading password: %v", passErr)
		return
	}

	content, err := getEnv(repoName, password)
	if err != nil {
		utils.Error("Error getting .env from vault: %v", err)
		return
	}

	comment := fmt.Sprintf("# Sourced from env-vault for %s\n\n", repoName)
	finalContent := append([]byte(comment), content...)

	if err := os.WriteFile(outputFilename, finalContent, 0644); err != nil {
		utils.Error("Error writing file '%s': %v", outputFilename, err)
		return
	}

	utils.Info(".env file for '%s' recovered and saved as '%s'!", repoName, outputFilename)
}

func handleGeneratePassword() {
	const passwordLength = 16

	utils.Info("🔒 Generating a new %d-character secure password...", passwordLength)
	password, err := generateSecurePassword(passwordLength)
	if err != nil {
		utils.Error("Error: Could not generate password: %v", err)
		return
	}
	utils.Info("✅ Your new secure password is:")
	fmt.Printf("\n%s\n\n", password)
	utils.Info("Copy this password and store it in a safe place (like a password manager).")
}

func handleSyncSave() {
	utils.Info("☁️  Saving current repo's secret to Firestore...")
	ctx := context.Background()

	client, err := initFirestoreClient(ctx)
	if err != nil {
		utils.Error("Error: Could not connect to Firebase: %v", err)
		return
	}
	defer client.Close()

	repoName, err := getCurrentRepoName()
	if err != nil {
		utils.Error("Error getting repo name: %v", err)
		return
	}

	vault, err := loadVault()
	if err != nil {
		utils.Error("Error loading local vault: %v", err)
		return
	}

	hashedRepoName := utils.GetRepoKey(repoName)
	secret, ok := vault[hashedRepoName]
	if !ok {
		utils.Error("Error: No secret found for '%s' in local vault.json. Please run 'save' first.", repoName)
		logAudit("sync-save", repoName, false, "No secret found for repository in local vault.json.")
		return
	}

	if err := saveSecretToFirebase(ctx, client, repoName, secret); err != nil {
		utils.Error("Error saving to Firestore: %v", err)
		return
	}

	utils.Info("✅ Secret for '%s' successfully saved to Firestore.", repoName)
}

func handleSyncLoad() {
	utils.Info("☁️  Loading all secrets from Firestore to local vault.json...")
	ctx := context.Background()

	client, err := initFirestoreClient(ctx)
	if err != nil {
		utils.Error("Error: Could not connect to Firebase: %v", err)
		return
	}
	defer client.Close()

	if err := loadSecretsFromFirebase(ctx, client); err != nil {
		utils.Error("Error loading from Firestore: %v", err)
		return
	}
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
			utils.Error("Failed to detect repository name: %v", err)
			return
		}
	}

	vault, err := loadVault()
	if err != nil {
		utils.Error("Error loading vault: %v", err)
		return
	}

	hashedRepoName := utils.GetRepoKey(repoName)
	if _, exists := vault[hashedRepoName]; !exists {
		utils.Error("No entry found for repository '%s' in vault. Run 'save' first.", repoName)
		return
	}

	var oldPassword []byte
	if *useBitWarden {
		secretProvider, err := providers.GetSecretProvider()
		if err != nil {
			utils.Error(ErrNoSecretProvider.Error(), ":%v", err)
			return
		}
		oldPassword, err = secretProvider.Get(hashedRepoName)
		if err != nil {
			utils.Error("Failed to get password from secret provider: %v", err)
			return
		}
	} else {
		oldPassword, err = promptForPassword("Enter CURRENT master password to decrypt: ")
		if err != nil {
			utils.Error("Failed to read current password: %v", err)
			return
		}
	}

	if oldPassword == nil {
		utils.Error("Failed to obtain current password")
		return
	}

	plaintext, err := getEnv(repoName, oldPassword)
	if err != nil {
		utils.Error(ErrDecryptionFailed.Error(), ":%v", err)
		return
	}

	var newPassword []byte
	var newPasswordGenerated bool
	var secretProvider providers.SecretProvider

	utils.Info("\n🔐 Setting NEW master password...")
	response := promptYesNo("Generate a secure password? (y/n): ")
	if response {
		passStr, err := generateSecurePassword(16)
		if err != nil {
			utils.Error("Password generation failed: %v", err)
			return
		}
		newPassword = []byte(passStr)
		newPasswordGenerated = true
		utils.Info("✅ New secure password generated:")
		fmt.Printf("\n%s\n\n", string(newPassword))
	} else {
		p1, err := promptForPassword("Enter new master password: ")
		if err != nil {
			utils.Error("Failed to read new password: %v", err)
			return
		}
		p2, err := promptForPassword("Confirm new master password: ")
		if err != nil {
			utils.Error("Failed to read confirmation: %v", err)
			return
		}
		if !bytes.Equal(p1, p2) {
			utils.Error("Passwords do not match.")
			return
		}
		newPassword = p1
	}

	if err := saveEnv(repoName, plaintext, newPassword); err != nil {
		utils.Error("Failed to re-encrypt with new password: %v", err)
		return
	}

	if newPasswordGenerated && *useBitWarden {
		if promptYesNo("Save new password to BitWarden Secrets Manager? (y/n): ") {
			secretProvider, err = providers.GetSecretProvider()
			if err != nil {
				utils.Error(ErrNoSecretProvider.Error(), ":%v", err)
				return
			}
			if err := secretProvider.Set(hashedRepoName, newPassword); err != nil {
				utils.Error("Error saving to %s: %v", secretProvider.Name(), err)
				return
			}
			utils.Info("✅ Master password updated in BitWarden.")
		}
	}

	utils.Success("✅ Master password rotated successfully for '%s'!", repoName)
}

func handleVerify() {
	utils.Info("🔍 Iniciando verificación de integridad para todas las entradas del vault...")

	getPasswordFunc := func(hashedKey string) ([]byte, error) {
		if *useBitWarden {
			secretProvider, err := providers.GetSecretProvider()
			if err != nil {
				return nil, fmt.Errorf("no secret provider configured: %w", err)
			}
			return secretProvider.Get(hashedKey)
		} else {
			prompt := fmt.Sprintf("Enter master password for key %s (required for HMAC check): ", hashedKey)
			return promptForPassword(prompt)
		}
	}

	results, err := VerifyAll(getPasswordFunc)
	if err != nil {
		utils.Error("Error al cargar el vault para la verificación: %v", err)
		return
	}

	successCount := 0
	failureCount := 0

	utils.Info("\n--- Resultados de la verificación ---")
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

	utils.Info("\n--- Resumen ---")
	utils.Info("Éxito: %d", successCount)
	utils.Info("Fallo: %d", failureCount)

	if failureCount > 0 {
		utils.Warn("\nATENCIÓN: Se detectaron fallos de verificación. Sus datos podrían haber sido manipulados o la contraseña maestra era incorrecta.")
	} else {
		utils.Success("🎉 Verificación de integridad completada. Todos los secretos son válidos.")
	}
}

func handleSetupConfig() {
	utils.Info("🚀 Starting env-vault configuration setup...")
	configDir, err := os.UserConfigDir()
	if err != nil {
		utils.Error("Error: Could not determine user configuration directory: %v", err)
		return
	}
	appConfigDir := filepath.Join(configDir, "env-vault")

	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		utils.Error("Error creating config directory: %v", err)
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

# 🛡️ Core Security Settings
VAULT_FILE_PATH=%s/vault.json
ENV_VAULT_PEPPER=%s

# 🔑 Bitwarden Secrets Manager
BWS_PROJECT_ID=%s

# ☁️ Google Firestore/Cloud Sync
GOOGLE_APPLICATION_CREDENTIALS=%s

# ⚙️ Logging Configuration (Defaults)
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
		utils.Error("Error writing config file: %v", err)
		return
	}

	utils.Success("\n✅ Configuration file created successfully at:")
	fmt.Printf("   %s\n\n", configFilePath)

	configCommand := fmt.Sprintf("export ENV_VAULT_CONFIG_PATH=\"%s\"", configFilePath)
	shellName := getShellName()

	utils.Info("⚠️ Final Step: Load Configuration")
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
		utils.Info("⚠️ Final Step: Load Configuration")
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
	} else {
		utils.Info("⚠️ Final Step: Load Configuration")
		utils.Warn("Could not automatically detect your shell configuration file.")
		utils.Warn("Please add the following line to your shell's startup file (e.g., ~/.bashrc, ~/.zshrc):")
		fmt.Printf("\nexport ENV_VAULT_CONFIG_PATH=\"%s\"\n\n", configFilePath)
	}

	utils.Info("Then, restart your terminal or run 'source [your shell config file]'.")
	utils.Success("Setup complete! You can now run 'env-vault save' in any Git repository.")
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
	fmt.Println("  rotate-pass   - Rotate the master password for the current (or specified) repository.")
	fmt.Println("  verify        - Verify the integrity of all secrets in vault.json (requires master password).")
	fmt.Println("  setup-config  - Interactive setup for env-vault configuration.")
	fmt.Println("\nOPTIONS:")
	fmt.Println("  -use-bitwarden  - Use BitWarden to get the master password instead of prompting.")
	fmt.Println("  -repo=NAME      - Specify repository name (e.g., 'user/repo') for rotate-pass.")
	fmt.Println("\nMake sure to set the necessary environment variables before using the tool.")
	fmt.Println("For more information, visit: https://github.com/bulga138/env-vault")
}
