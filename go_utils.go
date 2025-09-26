package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Expresión regular para encontrar "owner/repo" en URLs de Git
var repoRegex = regexp.MustCompile(`(?:github\.com[:/])([\w-]+\/[\w-.]+?)(\.git)?$`)

// getCurrentRepoName ejecuta un comando de git para obtener la URL remota
// y extrae el nombre del repositorio.
func getCurrentRepoName() (string, error) {
	// Obtenemos la URL del remote "origin"
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

	// La expresión regular captura "owner/repo.git", así que eliminamos el ".git" si existe.
	repoName := strings.TrimSuffix(matches[1], ".git")
	return repoName, nil
}