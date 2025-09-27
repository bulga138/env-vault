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
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file: ", err)
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
	fmt.Println("Saving .env file...")

	repoName, err := getCurrentRepoName()
	if err != nil {
		fmt.Printf("Error getting repository name: %v\n", err)
		return
	}

	content, err := os.ReadFile(".env")
	if err != nil {
		fmt.Printf("Error reading '.env' file: %v\n", err)
		return
	}

	vault, err := loadVault()
	if err != nil {
		fmt.Printf("Error loading vault: %v\n", err)
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
			fmt.Println("\nNo password found for this repository. Generating a new secure password...")
			passStr, passErr := generateSecurePassword(16)
			password = []byte(passStr)
			if passErr == nil {
				fmt.Println("\n✅ New master password generated for this repository:")
				fmt.Printf("\n%s\n\n", string(password))
				fmt.Println("You will be asked if you want to save this to BitWarden shortly.")
				fmt.Println("---------------------------------------------------------------")
			}
		}
	}

	if passErr != nil {
		fmt.Printf("\nPassword error: %v\n", passErr)
		return
	}

	if err := saveEnv(repoName, content, password); err != nil {
		fmt.Printf("Error saving .env to vault: %v\n", err)
		return
	}

	fmt.Printf("Do you wanna save the password in BitWarden Secrets Manager? (y/n): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) == "y" {
		if err := saveMasterPasswordToBitWarden(repoName, password); err != nil {
			fmt.Printf("Error saving master password to BitWarden: %v\n", err)
			return
		}
		fmt.Println("Master password saved to BitWarden Secrets Manager.")
	} else {
		fmt.Println("Warning: Master password not saved. Make sure to remember it, as you will need it to decrypt your .env files.")
	}
	fmt.Printf(".env file for '%s' safely saved!\n", repoName)
}

func handleGet() {
	fmt.Println("Getting .env file...")

	outputFilename := ".env"
	if _, err := os.Stat(".env"); err == nil {
		overrideFilename := os.Getenv("ENV_VAULT_EXISTING_FILENAME")
		if overrideFilename != "" {
			outputFilename = overrideFilename
		} else {
			outputFilename = ".env.env-vault"
		}
		fmt.Printf("⚠️  Found local .env file. It will be saved as '%s' to avoid overwriting.\n", outputFilename)
	}

	repoName, err := getCurrentRepoName()
	if err != nil {
		fmt.Printf("Error getting repo name: %v\n", err)
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
		fmt.Printf("\nError reading password: %v\n", passErr)
		return
	}

	content, err := getEnv(repoName, password)
	if err != nil {
		fmt.Printf("Error al obtener el .env: %v\n", err)
		return
	}

	comment := fmt.Sprintf("# Sourced from env-vault for %s\n\n", repoName)

	finalContent := append([]byte(comment), content...)

	if err := os.WriteFile(outputFilename, finalContent, 0644); err != nil {
		fmt.Printf("Error writing file '%s': %v\n", outputFilename, err)
		return
	}

	fmt.Printf(".env file for '%s' recovered and saved as '%s'!\n", repoName, outputFilename)
}

func handleGeneratePassword() {
	const passwordLength = 16

	fmt.Printf("🔒 Generating a new %d-character secure password...\n", passwordLength)
	password, err := generateSecurePassword(passwordLength)
	if err != nil {
		fmt.Printf("Error: Could not generate password: %v\n", err)
		return
	}
	fmt.Println("✅ Your new secure password is:")
	fmt.Printf("\n%s\n\n", password)
	fmt.Println("Copy this password and store it in a safe place (like a password manager).")
}

func handleSyncSave() {
	fmt.Println("☁️  Saving current repo's secret to Firestore...")
	ctx := context.Background()

	client, err := initFirestoreClient(ctx)
	if err != nil {
		fmt.Printf("Error: Could not connect to Firebase: %v\n", err)
		return
	}
	defer client.Close()

	repoName, err := getCurrentRepoName()
	if err != nil {
		fmt.Printf("Error getting repo name: %v\n", err)
		return
	}

	vault, err := loadVault()
	if err != nil {
		fmt.Printf("Error loading local vault: %v\n", err)
		return
	}
	secret, ok := vault[repoName]
	if !ok {
		fmt.Printf("Error: No secret found for '%s' in local vault.json. Please run 'save' first.\n", repoName)
		return
	}

	if err := saveSecretToFirebase(ctx, client, repoName, secret); err != nil {
		fmt.Printf("Error saving to Firestore: %v\n", err)
		return
	}

	fmt.Printf("✅ Secret for '%s' successfully saved to Firestore.\n", repoName)
}

func handleSyncLoad() {
	fmt.Println("☁️  Loading all secrets from Firestore to local vault.json...")
	ctx := context.Background()

	client, err := initFirestoreClient(ctx)
	if err != nil {
		fmt.Printf("Error: Could not connect to Firebase: %v\n", err)
		return
	}
	defer client.Close()

	if err := loadSecretsFromFirebase(ctx, client); err != nil {
		fmt.Printf("Error loading from Firestore: %v\n", err)
		return
	}
}

func printHelp() {
	fmt.Println("Welcome to env-vault 🛡️")
	fmt.Println("Usage: env-vault <command>")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  save    - Encrypt and save the current repository's .env file.")
	fmt.Println("  get     - Retrieve and decrypt the .env file for the current repository.")
	fmt.Println("  generate-pass   - Generate a new secure master password.")
	fmt.Println("  sync-save       - Save the current repo's secret to Firestore.")
	fmt.Println("  sync-load       - Load all secrets from Firestore to vault.json.")
	fmt.Println("\nOptions:")
	fmt.Println("  -use-bitwarden  - Use BitWarden to get the master password instead of prompting.")
	fmt.Println("\nMake sure to set the necessary environment variables before using the tool.")
	fmt.Println("For more information, visit: https://github.com/bulga138/env-vault")
}
