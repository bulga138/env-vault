package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var repoRegex = regexp.MustCompile(`(?:github\.com[:/])([\w-]+\/[\w-.]+?)(\.git)?$`)

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
