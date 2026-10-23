package main

import (
	"flag"
	"fmt"

	"github.com/bulga138/env-vault/utils"
)

func handleKeyring() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: env-vault keyring <set|get|delete|status>")
		return
	}
	sub := flag.Arg(1)

	repoName, err := getCurrentRepoName()
	if err != nil {
		utils.Fatal("Error getting repo name: %v", err)
		return
	}
	repoKey := utils.GetRepoKey(repoName)

	switch sub {
	case "set":
		pw, err := promptForPassword("Enter master password to store in keyring: ")
		if err != nil {
			utils.Fatal("Error reading password: %v", err)
			return
		}
		pwBuf, pw := lockBytes(pw)
		defer destroyLocked(pwBuf)
		if err := keyringSet(repoKey, pw); err != nil {
			logAudit("keyring-set", repoName, false, err.Error())
			utils.Fatal("Error storing in keyring: %v", err)
			return
		}
		logAudit("keyring-set", repoName, true, "")
		utils.Success("Password stored in OS keyring for this repository.")
	case "get":
		pw, err := keyringGet(repoKey)
		if err != nil {
			utils.Fatal("No password in keyring: %v", err)
			return
		}
		pwBuf, pw := lockBytes(pw)
		defer destroyLocked(pwBuf)
		// Print to stdout for scripting (use with care)
		fmt.Println(string(pw))
	case "delete":
		if err := keyringDelete(repoKey); err != nil {
			logAudit("keyring-delete", repoName, false, err.Error())
			utils.Fatal("Error deleting from keyring: %v", err)
			return
		}
		logAudit("keyring-delete", repoName, true, "")
		utils.Success("Password removed from OS keyring.")
	case "status":
		if _, err := keyringGet(repoKey); err != nil {
			utils.Info("No password stored in keyring for this repository.")
		} else {
			utils.Success("Password is stored in OS keyring for this repository.")
		}
	default:
		fmt.Println("Unknown keyring subcommand. Use set, get, delete, or status.")
	}
}
