package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bulga138/env-vault/utils"
)

func handleBackup() {
	vaultFile, err := getVaultFilePath()
	if err != nil {
		utils.Fatal("Error getting vault path: %v", err)
		return
	}
	if _, err := os.Stat(vaultFile); err != nil {
		utils.Fatal("No vault file found to back up: %v", err)
		return
	}

	configDir, _ := os.UserConfigDir()
	backupDir := filepath.Join(configDir, "env-vault", "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		utils.Fatal("Error creating backup directory: %v", err)
		return
	}
	backupFile := filepath.Join(backupDir, fmt.Sprintf("vault-%s.json", time.Now().Format("20060102-150405")))
	data, err := os.ReadFile(vaultFile)
	if err != nil {
		utils.Fatal("Error reading vault: %v", err)
		return
	}
	if err := os.WriteFile(backupFile, data, 0600); err != nil {
		utils.Fatal("Error writing backup: %v", err)
		return
	}
	logAudit("backup", "", true, "")
	utils.Success("Backup created at %s", backupFile)
}

func handleRestore() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: env-vault restore <backup-file>")
		return
	}
	backupFile := flag.Arg(1)
	data, err := os.ReadFile(backupFile)
	if err != nil {
		logAudit("restore", "", false, err.Error())
		utils.Fatal("Error reading backup file: %v", err)
		return
	}
	// Validate JSON
	var v vaultData
	if err := json.Unmarshal(data, &v); err != nil {
		logAudit("restore", "", false, err.Error())
		utils.Fatal("Invalid backup file: %v", err)
		return
	}
	vaultFile, err := getVaultFilePath()
	if err != nil {
		utils.Fatal("Error getting vault path: %v", err)
		return
	}
	if err := os.WriteFile(vaultFile, data, 0600); err != nil {
		logAudit("restore", "", false, err.Error())
		utils.Fatal("Error restoring vault: %v", err)
		return
	}
	logAudit("restore", "", true, "")
	utils.Success("Vault restored from %s", backupFile)
}