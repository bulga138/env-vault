package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"syscall"

	"golang.org/x/term"
)

// Definimos los requisitos de la contraseña con expresiones regulares.
var (
	hasMinLen  = regexp.MustCompile(`.{8,}`)
	hasUpper   = regexp.MustCompile(`[A-Z]`)
	hasLower   = regexp.MustCompile(`[a-z]`)
	hasNumber  = regexp.MustCompile(`[0-9]`)
	hasSymbol  = regexp.MustCompile(`[\W_]`) // \W es "no palabra", incluye símbolos.
)

// validatePasswordStrength comprueba que la contraseña cumple con los requisitos de complejidad.
func validatePasswordStrength(password []byte) error {
	if !hasMinLen.Match(password) {
		return fmt.Errorf("la contraseña debe tener al menos 8 caracteres")
	}
	if !hasUpper.Match(password) {
		return fmt.Errorf("la contraseña debe contener al menos una letra mayúscula")
	}
	if !hasLower.Match(password) {
		return fmt.Errorf("la contraseña debe contener al menos una letra minúscula")
	}
	if !hasNumber.Match(password) {
		return fmt.Errorf("la contraseña debe contener al menos un número")
	}
	if !hasSymbol.Match(password) {
		return fmt.Errorf("la contraseña debe contener al menos un símbolo")
	}
	return nil
}

// promptForPassword pide una contraseña al usuario una sola vez.
func promptForPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	fmt.Println()
	return password, nil
}

// promptForNewPassword pide una nueva contraseña, la confirma y valida su fortaleza.
func promptForNewPassword() ([]byte, error) {
	// Pedir la contraseña inicial
	password, err := promptForPassword("Introduce tu nueva contraseña maestra: ")
	if err != nil {
		return nil, fmt.Errorf("error al leer la contraseña: %w", err)
	}

	// Pedir la confirmación
	confirmPassword, err := promptForPassword("Confirma tu contraseña maestra: ")
	if err != nil {
		return nil, fmt.Errorf("error al leer la confirmación: %w", err)
	}

	// Verificar que coinciden
	if !bytes.Equal(password, confirmPassword) {
		return nil, fmt.Errorf("las contraseñas no coinciden")
	}

	// Validar la fortaleza de la contraseña
	if err := validatePasswordStrength(password); err != nil {
		return nil, fmt.Errorf("contraseña débil: %w", err)
	}

	fmt.Println("✅ Contraseña aceptada.")
	return password, nil
}

// func getMasterPasswordFromBitwarden() ([]byte, error) {
//     // The ID of the secret you stored in Bitwarden Secrets Manager.
//     // This could also come from a config file.
//     secretID := "d39fb552-13da-4fc8-9dbf-b36401088e73"

//     // Check if the BWS_ACCESS_TOKEN is set.
//     if os.Getenv("BWS_ACCESS_TOKEN") == "" {
//         return nil, fmt.Errorf("Bitwarden Secrets Manager token not found. Please set the BWS_ACCESS_TOKEN environment variable")
//     }

//     // Execute the 'bws' command.
//     cmd := exec.Command("bws", "secret", "get", secretID)
//     var out bytes.Buffer
//     var stderr bytes.Buffer
//     cmd.Stdout = &out
//     cmd.Stderr = &stderr

//     err := cmd.Run()
//     if err != nil {
//         return nil, fmt.Errorf("failed to fetch secret from Bitwarden: %s", stderr.String())
//     }

//     // The CLI output includes extra text and newlines, so we need to clean it.
//     // Example output: `Password: "the_actual_password"`
//     // This parsing logic may need adjustment based on the exact output.
//     // For simplicity, this example assumes the output is just the password.
//     return bytes.TrimSpace(out.Bytes()), nil
// }

func saveMasterPasswordToBitwarden(password []byte) error {
	// The ID of the secret you want to store in Bitwarden Secrets Manager.
	// This could also come from a config file.
	secretID := "d39fb552-13da-4fc8-9dbf-b36401088e73"

	// Check if the BWS_ACCESS_TOKEN is set.
	if os.Getenv("BWS_ACCESS_TOKEN") == "" {
		return fmt.Errorf("Bitwarden Secrets Manager token not found. Please set the BWS_ACCESS_TOKEN environment variable")
	}
	// Execute the 'bws' command to store the secret.
	cmd := exec.Command("bws", "secret", "set", secretID, string(password))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to save secret to Bitwarden: %s", stderr.String())
	}
	return nil
}