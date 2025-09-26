package main

import (
	"fmt"
	"os"
)

var (
	githubClientID     = os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
)

func main() {
	// Comprobamos que se hayan proporcionado los secretos de la app de OAuth
	if githubClientID == "" || githubClientSecret == "" {
		fmt.Println("Error: Debes configurar las variables de entorno GITHUB_CLIENT_ID y GITHUB_CLIENT_SECRET.")
		os.Exit(1)
	}

	// Comprobamos que el usuario haya pasado un comando
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	// Seleccionamos la acción basada en el comando
	switch os.Args[1] {
	case "login":
		handleLogin()
	case "save":
		handleSave()
	case "get":
		handleGet()
	default:
		printHelp()
	}
}

// handleSave se encarga de la lógica para guardar un .env
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
	if _, exists := vault[repoName]; exists {
		// Si ya existe, solo pedimos la contraseña una vez
		password, err = promptForPassword("Introduce tu contraseña maestra para actualizar: ")
	} else {
		// Si es una entrada nueva, usamos el flujo completo de validación
		password, err = promptForNewPassword()
	}

	if err != nil {
		fmt.Printf("\nError de contraseña: %v\n", err)
		return
	}

	if err := saveEnv(repoName, content, password); err != nil {
		fmt.Printf("Error al guardar el .env en el almacén: %v\n", err)
		return
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
	
	// Para obtener, siempre pedimos la contraseña una sola vez.
	password, err := promptForPassword("Introduce tu contraseña maestra para descifrar: ")
	if err != nil {
		fmt.Printf("\nError al leer la contraseña: %v\n", err)
		return
	}

	content, err := getEnv(repoName, password)
	if err != nil {
		fmt.Printf("Error al obtener el .env: %v\n", err)
		return
	}

	if err := os.WriteFile(".env", content, 0644); err != nil {
		fmt.Printf("Error al escribir el archivo .env local: %v\n", err)
		return
	}

	fmt.Printf("✅ ¡Archivo .env para '%s' recuperado y guardado localmente!\n", repoName)
}

// printHelp muestra un mensaje de ayuda simple
func printHelp() {
	fmt.Println("Bienvenido a env-vault 🛡️")
	fmt.Println("Uso: env-vault <comando>")
	fmt.Println("\nComandos disponibles:")
	fmt.Println("  login   - Autoriza la aplicación con tu cuenta de GitHub.")
	fmt.Println("  save    - Cifra y guarda el .env del repositorio actual.")
	fmt.Println("  get     - Recupera y descifra el .env para el repositorio actual.")
}