package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var repoRegex = regexp.MustCompile(`(?:github\.com[:/])([\w-]+\/[\w-.]+?)(\.git)?$`)

const (
	lowercaseChars = "abcdefghijklmnopqrstuvwxyz"
	uppercaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberChars    = "0123456789"
	symbolChars    = "!@#$%^&*()_+-=[]{}|;':,./<>?"
)

func getCurrentRepoName() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no se pudo ejecutar 'git remote get-url origin': %w", err)
	}

	url := strings.TrimSpace(string(output))
	matches := repoRegex.FindStringSubmatch(url)

	if len(matches) < 2 {
		return "", fmt.Errorf("no se pudo parsear el nombre del repositorio desde la URL: %s", url)
	}

	repoName := strings.TrimSuffix(matches[1], ".git")
	return repoName, nil
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

func getOrPromptEnvVar(varName, promptText string, isSecret bool) (string, error) {
	value := os.Getenv(varName)
	if value != "" {
		return value, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("environment variable %s is not set and this is not an interactive session", varName)
	}

	Info("Configuration needed: %s\n", promptText)
	if isSecret {
		byteValue, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("could not read secret input: %w", err)
		}
		value = string(byteValue)
		fmt.Println() // New line after password input
	} else {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		value = strings.TrimSpace(text)
	}

	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", varName)
	}

	if err := saveToDotEnv(varName, value); err != nil {
		Warn("Warning: could not save %s to .env file: %v\n", varName, err)
	}

	os.Setenv(varName, value)
	return value, nil
}

func saveToDotEnv(key, value string) error {
	file, err := os.OpenFile(".env", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "\n%s=\"%s\"", key, value); err != nil {
		return err
	}
	return nil
}

func generateSecurePassword(length int) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("password length must be at least 8 characters to ensure complexity")
	}

	allChars := lowercaseChars + uppercaseChars + numberChars + symbolChars

	var builder strings.Builder
	builder.Grow(length)

	guaranteedChars := []string{lowercaseChars, uppercaseChars, numberChars, symbolChars}
	for _, charSet := range guaranteedChars {
		randomChar, err := getRandomChar(charSet)
		if err != nil {
			return "", err
		}
		builder.WriteRune(randomChar)
	}

	remainingLength := length - len(guaranteedChars)
	for i := 0; i < remainingLength; i++ {
		randomChar, err := getRandomChar(allChars)
		if err != nil {
			return "", err
		}
		builder.WriteRune(randomChar)
	}

	passwordRunes := []rune(builder.String())
	for i := range passwordRunes {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordRunes))))
		if err != nil {
			return "", err
		}
		j := jBig.Int64()
		passwordRunes[i], passwordRunes[j] = passwordRunes[j], passwordRunes[i]
	}

	return string(passwordRunes), nil
}

func getRandomChar(charSet string) (rune, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random number: %w", err)
	}
	return rune(charSet[n.Int64()]), nil
}
