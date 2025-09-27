package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/term"
)

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
		return "", fmt.Errorf("failed to run 'git remote get-url origin': %w", err)
	}

	url := strings.TrimSpace(string(output))
	if url == "" {
		return "", fmt.Errorf("git remote URL is empty")
	}

	repoID, err := parseGitRemoteURL(url)
	if err != nil {
		return "", fmt.Errorf("failed to parse git remote URL '%s': %w", url, err)
	}

	return repoID, nil
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

func promptYesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
		fmt.Println("Please enter 'y' or 'n'.")
	}
}

func parseGitRemoteURL(rawURL string) (string, error) {
	if rawURL == "" {
		return "", fmt.Errorf("empty URL")
	}

	if strings.HasPrefix(rawURL, "git@") || (!strings.Contains(rawURL, "://") && strings.Contains(rawURL, ":")) {

		parts := strings.SplitN(rawURL, "@", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid SSH URL format")
		}

		hostPath := parts[1]

		colonIndex := strings.Index(hostPath, ":")
		if colonIndex == -1 {
			rawURL = "https://" + strings.TrimPrefix(hostPath, "/")
		} else {
			cleanPath := strings.Replace(hostPath, ":", "/", 1)
			rawURL = "https://" + cleanPath
		}
	}

	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	if u.Host == "" {
		return "", fmt.Errorf("no host found in URL")
	}

	path := strings.Trim(u.Path, "/")
	if path == "" {
		return "", fmt.Errorf("no path found in URL")
	}

	path = strings.TrimSuffix(path, ".git")

	if path == "" {
		return "", fmt.Errorf("path is empty after cleaning")
	}

	return u.Hostname() + "/" + path, nil
}

func secureZero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
