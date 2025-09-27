package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var (
	useBitWarden = flag.Bool("use-bitwarden", false, "Use BitWarden to fetch the master password")
)

func main() {
	initLogging()

	err := godotenv.Load()
	if err != nil {
		Warn("Error loading .env file, proceeding without it.", err)
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
	default:
		printHelp()
	}
}

func handleSave() {
	Info("Saving .env file...")
	repoName, err := getCurrentRepoName()
	if err != nil {
		Error("Failed to get repository name: %v", err)
		return
	}

	content, err := os.ReadFile(".env")
	if err != nil {
		Error("Error reading '.env' file: %v", err)
		return
	}

	vault, err := loadVault()
	if err != nil {
		Error("Error loading vault: %v", err)
		return
	}

	var password []byte
	var passErr error
	if *useBitWarden {
		password, passErr = getMasterPasswordFromSecretsManager(repoName)
	} else {
		if _, exists := vault[repoName]; exists {
			password, passErr = promptForPassword("Enter your master password again: ")
		} else {
			Info("\nNo password found for this repository. Generating a new secure password...")
			passStr, genErr := generateSecurePassword(16)
			if genErr != nil {
				passErr = genErr
			} else {
				password = []byte(passStr)
				Info("\n✅ New master password generated for this repository:")
				fmt.Printf("\n%s\n\n", string(password))
				Info("You will be asked if you want to save this to BitWarden shortly.")
				Info("---------------------------------------------------------------")
			}
		}
	}

	if passErr != nil {
		Error("Password error: %v", passErr)
		return
	}

	if err := saveEnv(repoName, content, password); err != nil {
		Error("Error saving .env to vault: %v", err)
		return
	}

	Info("Do you wanna save the password in BitWarden Secrets Manager? (y/n): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) == "y" {
		if err := saveMasterPasswordToBitWarden(repoName, password); err != nil {
			Error("Error saving master password to BitWarden: %v", err)
			return
		}
		Info("Master password saved to BitWarden Secrets Manager.")
	} else {
		Warn("Master password not saved. Make sure to remember it, as you will need it to decrypt your .env files.")
	}
	Info(".env file for '%s' safely saved!", repoName)
}

func handleGet() {
	Info("Getting .env file...")

	outputFilename := ".env"
	if _, err := os.Stat(".env"); err == nil {
		overrideFilename := os.Getenv("ENV_VAULT_EXISTING_FILENAME")
		if overrideFilename != "" {
			outputFilename = overrideFilename
		} else {
			outputFilename = ".env.env-vault"
		}
		Warn("Found local .env file. It will be saved as '%s' to avoid overwriting.", outputFilename)
	}

	repoName, err := getCurrentRepoName()
	if err != nil {
		Error("Error getting repo name: %v", err)
		return
	}

	var password []byte
	var passErr error

	if *useBitWarden {
		password, passErr = getMasterPasswordFromSecretsManager(repoName)
	} else {
		password, passErr = promptForPassword("Enter your master password to decrypt: ")
	}
	if passErr != nil {
		Error("Error reading password: %v", passErr)
		return
	}

	content, err := getEnv(repoName, password)
	if err != nil {
		Error("Error getting .env from vault: %v", err)
		return
	}

	comment := fmt.Sprintf("# Sourced from env-vault for %s\n\n", repoName)
	finalContent := append([]byte(comment), content...)

	if err := os.WriteFile(outputFilename, finalContent, 0644); err != nil {
		Error("Error writing file '%s': %v", outputFilename, err)
		return
	}

	Info(".env file for '%s' recovered and saved as '%s'!", repoName, outputFilename)
}

func handleGeneratePassword() {
	const passwordLength = 16

	Info("🔒 Generating a new %d-character secure password...", passwordLength)
	password, err := generateSecurePassword(passwordLength)
	if err != nil {
		Error("Error: Could not generate password: %v", err)
		return
	}
	Info("✅ Your new secure password is:")
	fmt.Printf("\n%s\n\n", password)
	Info("Copy this password and store it in a safe place (like a password manager).")
}

func handleSyncSave() {
	Info("☁️  Saving current repo's secret to Firestore...")
	ctx := context.Background()

	client, err := initFirestoreClient(ctx)
	if err != nil {
		Error("Error: Could not connect to Firebase: %v", err)
		return
	}
	defer client.Close()

	repoName, err := getCurrentRepoName()
	if err != nil {
		Error("Error getting repo name: %v", err)
		return
	}

	vault, err := loadVault()
	if err != nil {
		Error("Error loading local vault: %v", err)
		return
	}
	secret, ok := vault[repoName]
	if !ok {
		Error("Error: No secret found for '%s' in local vault.json. Please run 'save' first.", repoName)
		return
	}

	if err := saveSecretToFirebase(ctx, client, repoName, secret); err != nil {
		Error("Error saving to Firestore: %v", err)
		return
	}

	Info("✅ Secret for '%s' successfully saved to Firestore.", repoName)
}

func handleSyncLoad() {
	Info("☁️  Loading all secrets from Firestore to local vault.json...")
	ctx := context.Background()

	client, err := initFirestoreClient(ctx)
	if err != nil {
		Error("Error: Could not connect to Firebase: %v", err)
		return
	}
	defer client.Close()

	if err := loadSecretsFromFirebase(ctx, client); err != nil {
		Error("Error loading from Firestore: %v", err)
		return
	}
}

func printHelp() {
	fmt.Println("Welcome to env-vault 🛡️")
	fmt.Println("Usage: env-vault <command>")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  save          - Encrypt and save the current repository's .env file.")
	fmt.Println("  get           - Retrieve and decrypt the .env file for the current repository.")
	fmt.Println("  generate-pass - Generate a new secure master password.")
	fmt.Println("  sync-save     - Save the current repo's secret to Firestore.")
	fmt.Println("  sync-load     - Load all secrets from Firestore to vault.json.")
	fmt.Println("\nOptions:")
	fmt.Println("  -use-bitwarden  - Use BitWarden to get the master password instead of prompting.")
	fmt.Println("\nMake sure to set the necessary environment variables before using the tool.")
	fmt.Println("For more information, visit: https://github.com/bulga138/env-vault")
}
