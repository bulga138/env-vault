package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	lowercaseChars = "abcdefghijklmnopqrstuvwxyz"
	uppercaseChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numberChars    = "0123456789"
	symbolChars    = "!@#$%^&*()_+-=[]{}|;':,./<>?"
)

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
