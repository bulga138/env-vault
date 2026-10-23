package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bulga138/env-vault/utils"
)

func getAuditFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "env-vault", "audit", "audit.log"), nil
}

func readAuditEntries() ([]AuditLog, error) {
	path, err := getAuditFilePath()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []AuditLog{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []AuditLog
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		var e AuditLog
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

func handleAudit() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: env-vault audit <list|tail> [--audit-action A] [--audit-user U] [--audit-limit N]")
		return
	}
	sub := flag.Arg(1)
	entries, err := readAuditEntries()
	if err != nil {
		utils.Fatal("Error reading audit log: %v", err)
		return
	}

	var filtered []AuditLog
	for _, e := range entries {
		if *auditFilterAction != "" && e.Action != *auditFilterAction {
			continue
		}
		if *auditFilterUser != "" && e.User != *auditFilterUser {
			continue
		}
		filtered = append(filtered, e)
	}

	if sub == "tail" {
		n := *auditLimit
		if n <= 0 {
			n = 20
		}
		if len(filtered) > n {
			filtered = filtered[len(filtered)-n:]
		}
	} else if *auditLimit > 0 && len(filtered) > *auditLimit {
		filtered = filtered[:*auditLimit]
	}

	if len(filtered) == 0 {
		utils.Info("No audit entries.")
		return
	}
	for _, e := range filtered {
		status := "OK"
		if !e.Success {
			status = "FAIL"
		}
		fmt.Printf("%s %-12s %-40s %-5s user=%s %s\n",
			e.Timestamp.Format(time.RFC3339), e.Action, e.RepoName, status, e.User, e.ErrorMessage)
	}
}