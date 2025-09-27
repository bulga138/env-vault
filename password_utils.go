package main

import (
	"bytes"
	"fmt"
	"regexp"
	"syscall"

	"golang.org/x/term"
)

var (
	hasMinLen = regexp.MustCompile(`.{8,}`)
	hasUpper  = regexp.MustCompile(`[A-Z]`)
	hasLower  = regexp.MustCompile(`[a-z]`)
	hasNumber = regexp.MustCompile(`[0-9]`)
	hasSymbol = regexp.MustCompile(`[\W_]`)
)

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

func promptForPassword(prompt string) ([]byte, error) {
	fmt.Print(prompt)
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	fmt.Println()
	return password, nil
}

func promptForNewPassword() ([]byte, error) {
	password, err := promptForPassword("Introduce tu nueva contraseña maestra: ")
	if err != nil {
		return nil, fmt.Errorf("error al leer la contraseña: %w", err)
	}

	confirmPassword, err := promptForPassword("Confirma tu contraseña maestra: ")
	if err != nil {
		return nil, fmt.Errorf("error al leer la confirmación: %w", err)
	}

	if !bytes.Equal(password, confirmPassword) {
		return nil, fmt.Errorf("las contraseñas no coinciden")
	}

	if err := validatePasswordStrength(password); err != nil {
		return nil, fmt.Errorf("contraseña débil: %w", err)
	}

	fmt.Println("✅ Contraseña aceptada.")
	return password, nil
}
