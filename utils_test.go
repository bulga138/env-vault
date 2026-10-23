package main

import (
	"fmt"
	"regexp"
	"testing"
)

func TestParseGitRemoteURL(t *testing.T) {

	tests := []struct {
		name     string
		rawURL   string
		expected string
		err      bool
	}{

		{
			name:     "HTTPS-Standard",
			rawURL:   "https://github.com/user/repo.git",
			expected: "github.com/user/repo",
			err:      false,
		},
		{
			name:     "HTTP-NoGitSuffix",
			rawURL:   "http://gitlab.com/group/project",
			expected: "gitlab.com/group/project",
			err:      false,
		},

		{
			name:     "SSH-Standard",
			rawURL:   "git@github.com:bulga138/gowine.git",
			expected: "github.com/bulga138/gowine",
			err:      false,
		},
		{
			name:     "SSH-WithPort",
			rawURL:   "ssh://git@myhost.com:2222/path/to/repo.git",
			expected: "myhost.com/path/to/repo",
			err:      false,
		},

		{
			name:     "EmptyURL",
			rawURL:   "",
			expected: "",
			err:      true,
		},
		{
			name:     "InvalidURLFormat",
			rawURL:   "//invalid-url",
			expected: "",
			err:      true,
		},
		{
			name:     "NoHostInURL",
			rawURL:   "https:///path/to/repo.git",
			expected: "",
			err:      true,
		},
		{
			name:     "PathIsEmptyAfterCleaning",
			rawURL:   "https://github.com/.git",
			expected: "",
			err:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseGitRemoteURL(tt.rawURL)

			if (err != nil) != tt.err {
				t.Errorf("parseGitRemoteURL() error = %v, expected error = %v", err, tt.err)
				return
			}
			if result != tt.expected {
				t.Errorf("parseGitRemoteURL() result = %v, expected = %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {

	t.Run("LengthTooShort", func(t *testing.T) {
		_, err := generateSecurePassword(7)
		if err == nil {
			t.Errorf("generateSecurePassword() did not return error for length < 8")
		}
	})

	for length := 8; length <= 32; length += 8 {
		t.Run(fmt.Sprintf("ComplexityAndLength_%d", length), func(t *testing.T) {
			password, err := generateSecurePassword(length)
			if err != nil {
				t.Fatalf("generateSecurePassword() failed: %v", err)
			}

			if len(password) != length {
				t.Errorf("Expected length %d, got %d", length, len(password))
			}

			// Expresiones regulares para verificar la complejidad
			// Debe tener al menos una minúscula
			if ok, _ := regexp.MatchString(`[a-z]`, password); !ok {
				t.Errorf("Password lacks lowercase character: %s", password)
			}
			// Debe tener al menos una mayúscula
			if ok, _ := regexp.MatchString(`[A-Z]`, password); !ok {
				t.Errorf("Password lacks uppercase character: %s", password)
			}
			// Debe tener al menos un número
			if ok, _ := regexp.MatchString(`[0-9]`, password); !ok {
				t.Errorf("Password lacks number character: %s", password)
			}
			// Debe tener al menos un símbolo (usando una clase que cubre todos los símbolos definidos)
			symbolRegex := `[!@#\$%\^&\*\(\)_\+=\-\[\]\{\}\|;':,./<>?]`
			if ok, _ := regexp.MatchString(symbolRegex, password); !ok {
				t.Errorf("Password lacks symbol character: %s", password)
			}
		})
	}
}

func TestSecureZero(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	secureZero(data)

	expected := []byte{0, 0, 0, 0, 0}
	for i, v := range data {
		if v != expected[i] {
			t.Errorf("secureZero failed. Expected byte at index %d to be 0, got %d", i, v)
		}
	}
}

// --- Notas sobre las funciones no probadas ---
// * getCurrentRepoName: Requiere un entorno git configurado o mockeo de os/exec.
// * promptForPassword: Requiere mockeo de os.Stdin y el paquete golang.org/x/term.
// * promptYesNo: Requiere mockeo de os.Stdin y lectura de entrada simulada.
