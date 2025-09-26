// filename: store.go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gofrs/flock"
)

const vaultFile = "vault.json"

// StoredEnv ahora contiene el texto cifrado y la sal necesaria para recrear la clave.
type StoredEnv struct {
	Ciphertext string `json:"ciphertext"`
	Salt       string `json:"salt"`
}

type vaultData map[string]StoredEnv

// saveEnv ahora deriva la clave y guarda la sal.
func saveEnv(repoName string, content []byte, password []byte) error {
	// 1. Generar una sal criptográficamente segura
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("error al generar la sal: %w", err)
	}

	// 2. Derivar la clave de cifrado
	key := deriveKey(password, salt)

	// 3. Cifrar el contenido con la clave derivada
	encryptedContent, err := encrypt(content, key)
	if err != nil {
		return err
	}

	// 4. Guardar el contenido cifrado y la sal (codificados en Base64)
	vault, err := loadVault()
	if err != nil {
		return err
	}
	vault[repoName] = StoredEnv{
		Ciphertext: base64.StdEncoding.EncodeToString(encryptedContent),
		Salt:       base64.StdEncoding.EncodeToString(salt),
	}
	return saveVault(vault)
}

// getEnv ahora usa la sal guardada para derivar la clave y descifrar.
func getEnv(repoName string, password []byte) ([]byte, error) {
	vault, err := loadVault()
	if err != nil {
		return nil, err
	}

	stored, ok := vault[repoName]
	if !ok {
		return nil, fmt.Errorf("no se encontró ningún .env para el repositorio '%s'", repoName)
	}

	// 1. Decodificar la sal y el texto cifrado
	salt, err := base64.StdEncoding.DecodeString(stored.Salt)
	if err != nil {
		return nil, fmt.Errorf("sal corrupta: %w", err)
	}
	encryptedContent, err := base64.StdEncoding.DecodeString(stored.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("contenido cifrado corrupto: %w", err)
	}

	// 2. Derivar la clave usando la misma contraseña y la sal guardada
	key := deriveKey(password, salt)

	// 3. Descifrar
	return decrypt(encryptedContent, key)
}

func saveVault(vault vaultData) error {
	// Creamos un bloqueo en un archivo .lock para evitar condiciones de carrera.
	lock := flock.New(vaultFile + ".lock")
	locked, err := lock.TryLock()

	// Si no podemos obtener el bloqueo o hay un error, fallamos.
	if err != nil || !locked {
		return fmt.Errorf("no se pudo obtener el bloqueo del archivo del almacén, inténtalo de nuevo")
	}
	// `defer` asegura que el bloqueo se libere al final, incluso si hay errores.
	defer lock.Unlock()

	data, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(vaultFile, data, 0600)
}

// loadVault lee el archivo JSON y lo carga en memoria.
func loadVault() (vaultData, error) {
	data, err := os.ReadFile(vaultFile)
	if err != nil {
		// Si el archivo no existe, creamos un almacén vacío. Es normal la primera vez.
		if os.IsNotExist(err) {
			return make(vaultData), nil
		}
		return nil, err
	}

	var vault vaultData
	if err := json.Unmarshal(data, &vault); err != nil {
		return nil, err
	}
	return vault, nil
}