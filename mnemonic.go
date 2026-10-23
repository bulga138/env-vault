package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"strings"

	"github.com/bulga138/env-vault/utils"
	"github.com/tyler-smith/go-bip39"
)

func handleMnemonic() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: env-vault mnemonic <generate|recover> [words...]")
		return
	}
	sub := flag.Arg(1)
	switch sub {
	case "generate":
		entropy, err := bip39.NewEntropy(128)
		if err != nil {
			utils.Fatal("Error generating entropy: %v", err)
			return
		}
		mnemonic, err := bip39.NewMnemonic(entropy)
		if err != nil {
			utils.Fatal("Error generating mnemonic: %v", err)
			return
		}
		fmt.Println("Mnemonic (store securely):")
		fmt.Println(mnemonic)
		// Derive a master password from the mnemonic
		h := sha256.Sum256([]byte(mnemonic))
		fmt.Println("\nDerived master password:")
		fmt.Println(hex.EncodeToString(h[:]))
	case "recover":
		if flag.NArg() < 3 {
			fmt.Println("Usage: env-vault mnemonic recover <word1> <word2> ...")
			return
		}
		words := flag.Args()[2:]
		mnemonic := strings.Join(words, " ")
		if !bip39.IsMnemonicValid(mnemonic) {
			utils.Fatal("Invalid mnemonic phrase.")
			return
		}
		h := sha256.Sum256([]byte(mnemonic))
		fmt.Println("Recovered master password:")
		fmt.Println(hex.EncodeToString(h[:]))
	default:
		fmt.Println("Unknown mnemonic subcommand. Use generate or recover.")
	}
}
