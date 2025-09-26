package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

var (
	useBitwarden = flag.Bool("use-bitwarden", false, "Use Bitwarden to fetch the master password")
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("No se pudo cargar el archivo .env, asegurate de tener las variables de entorno configuradas.")
	}

	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	if githubClientID == "" || githubClientSecret == "" {
		fmt.Println("Error: Debes configurar las variables de entorno GITHUB_CLIENT_ID y GITHUB_CLIENT_SECRET.")
		os.Exit(1)
	}

	initOAuth(githubClientID, githubClientSecret)

	flag.Parse()

	if flag.NArg() < 1 {
		printHelp()
		return
	}
	command := flag.Arg(0)

	switch command {
	case "login":
		handleLogin()
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
	fmt.Println("🔒 Guardando el archivo .env...")

	repoName, err := getCurrentRepoName()
	if err != nil {
		fmt.Printf("Error al obtener el nombre del repositorio: %v\n", err)
		return
	}

	content, err := os.ReadFile(".env")
	if err != nil {
		fmt.Printf("Error: No se pudo leer el archivo '.env': %v\n", err)
		return
	}

	// Comprobamos si ya existe una entrada para decidir qué flujo de contraseña usar
	vault, err := loadVault()
	if err != nil {
		fmt.Printf("Error al cargar el almacén: %v\n", err)
		return
	}

	var password []byte
	var passErr error
	if *useBitwarden {
		// Fetch from Bitwarden if the flag is set
		password, passErr = getMasterPasswordFromBitwarden()
	} else {
		if _, exists := vault[repoName]; exists {
			// Si ya existe, solo pedimos la contraseña una vez
			password, passErr = promptForPassword("Introduce tu contraseña maestra para actualizar: ")
		} else {
			// Si es una entrada nueva, usamos el flujo completo de validación
			password, passErr = promptForNewPassword()
		}
	}

	if passErr != nil {
		fmt.Printf("\nError de contraseña: %v\n", passErr)
		return
	}

	if err := saveEnv(repoName, content, password); err != nil {
		fmt.Printf("Error al guardar el .env en el almacén: %v\n", err)
		return
	}

	fmt.Printf("Do you wanna save the password in Bitwarden Secrets Manager? (y/n): ")
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) == "y" {
		if err := saveMasterPasswordToBitwarden(password); err != nil {
			fmt.Printf("Error al guardar la contraseña maestra en Bitwarden: %v\n", err)
			return
		}
		fmt.Println("✅ Contraseña maestra guardada en Bitwarden Secrets Manager.")
	} else {
		fmt.Println("⚠️  Advertencia: No se guardó la contraseña maestra. Asegúrate de recordarla, ya que la necesitarás para descifrar tus .env.")
	}
	fmt.Printf("✅ ¡Archivo .env para '%s' guardado de forma segura!\n", repoName)
}

// handleGet se encarga de la lógica para obtener un .env
func handleGet() {
	fmt.Println("🔓 Obteniendo el archivo .env...")

	repoName, err := getCurrentRepoName()
	if err != nil {
		fmt.Printf("Error al obtener el nombre del repositorio: %v\n", err)
		return
	}

	var password []byte
	var passErr error

	if *useBitwarden {
		password, passErr = getMasterPasswordFromBitwarden()
	} else {
		password, passErr = promptForPassword("Introduce tu contraseña maestra para descifrar: ")
	}
	if passErr != nil {
		fmt.Printf("\nError al leer la contraseña: %v\n", err)
		return
	}

	content, err := getEnv(repoName, password)
	if err != nil {
		fmt.Printf("Error al obtener el .env: %v\n", err)
		return
	}
	dirName := strings.ReplaceAll(repoName, "/", "-")

	// Create the directory if it doesn't exist.
	if err := os.MkdirAll(dirName, 0755); err != nil {
		fmt.Printf("Error al crear el directorio '%s': %v\n", dirName, err)
		return
	}

	filePath := filepath.Join(dirName, ".env")

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		fmt.Printf("Error al escribir el archivo .env en '%s': %v\n", filePath, err)
		return
	}

	fmt.Printf("✅ ¡Archivo .env para '%s' recuperado y guardado localmente!\n", repoName)
}

func handleGeneratePassword() {
	// You can make the length configurable with a flag later (e.g., --length 24)
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

// handleSyncSave saves the current repo's secret to Firestore.
func handleSyncSave() {
	fmt.Println("☁️  Saving current repo's secret to Firestore...")
	ctx := context.Background()

	// Initialize Firestore client
	client, err := initFirestoreClient(ctx)
	if err != nil {
		fmt.Printf("Error: Could not connect to Firebase: %v\n", err)
		return
	}
	defer client.Close()

	// Get repo name
	repoName, err := getCurrentRepoName()
	if err != nil {
		fmt.Printf("Error getting repo name: %v\n", err)
		return
	}

	// Load the local vault to get the secret
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

	// Save to Firebase
	if err := saveSecretToFirebase(ctx, client, repoName, secret); err != nil {
		fmt.Printf("Error saving to Firestore: %v\n", err)
		return
	}

	fmt.Printf("✅ Secret for '%s' successfully saved to Firestore.\n", repoName)
}

// handleSyncLoad loads all secrets from Firestore to the local vault.
func handleSyncLoad() {
	fmt.Println("☁️  Loading all secrets from Firestore to local vault.json...")
	ctx := context.Background()

	// Initialize Firestore client
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

// printHelp muestra un mensaje de ayuda simple
func printHelp() {
	fmt.Println("Bienvenido a env-vault 🛡️")
	fmt.Println("Uso: env-vault <comando>")
	fmt.Println("\nComandos disponibles:")
	fmt.Println("  login   - Autoriza la aplicación con tu cuenta de GitHub.")
	fmt.Println("  save    - Cifra y guarda el .env del repositorio actual.")
	fmt.Println("  get     - Recupera y descifra el .env para el repositorio actual.")
	fmt.Println("  generate-pass   - Genera una nueva contraseña maestra segura.")
	fmt.Println("  sync-save       - Guarda el secreto del repo actual en Firestore.")
	fmt.Println("  sync-load       - Carga todos los secretos de Firestore a vault.json.")
}
