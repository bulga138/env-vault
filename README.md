# env-vault 🛡️

**Secure .env file manager with encryption and version control integration**

`env-vault` is a command-line tool that securely encrypts and stores your environment variables, linking them to specific Git repositories. Never worry about accidentally committing sensitive credentials again!

## 🌟 Features

  - 🔐 **Strong Encryption**: Uses Argon2 key derivation and **AES-256-GCM** 
  - 🛡️ **Integrity Verification**: Command to verify that secrets haven't been tampered with (`verify`)
  - 🔑 **Secret Providers**: Integrates with **Bitwarden Secrets Manager** for automatic password fetching. 
  - 🔄 **Git Integration**: Automatically detects repository context (`user/repo` format) 
  - ☁️ **Cloud Sync**: Optional backend integration with Google Firestore 
  - 📱 **Cross-Platform**: Works on Windows, macOS, and Linux
  - 🔒 **Secure Storage**: Uses local vault with file locking and audit logs 

-----

## 📦 Installation

### Pre-compiled binaries

Download the latest release from [GitHub Releases](https://github.com/yourusername/env-vault/releases)

### From source

```bash
# Clone the repository
git clone https://github.com/yourusername/env-vault.git
cd env-vault

# Build the project (requires Go 1.24+)
make build

# Or compile and install
go install .
```

-----

## 🚀 Quick Start

El flujo más seguro y recomendado es usar un proveedor de secretos externo (como Bitwarden) para almacenar la clave maestra de tu vault.

### 1\. Configuración de Bitwarden (Recomendado)

Sigue los pasos para configurar tu cuenta y obtener tu **ID de proyecto de Bitwarden Secrets Manager**.

```bash
# Exporta el ID de tu proyecto de Bitwarden
export BWS_PROJECT_ID="your_bitwarden_project_id"

# Asegúrate de tener el CLI 'bws' autenticado
bws login
```

### 2\. Guardar y Sincronizar un Nuevo Secreto

Para añadir un nuevo archivo `.env` a tu vault:

```bash
# Asegúrate de que tu archivo .env esté en el directorio actual
cd /path/to/your/project

# Guardar y usar Bitwarden (guarda el secreto localmente y la clave en Bitwarden)
# Si no existe una clave para este repo, se generará una nueva y te preguntará
# si quieres guardarla en Bitwarden.
env-vault save --use-bitwarden

# Opcional: Sincronizar el secreto cifrado con Firestore
env-vault sync-save
```

### 3\. Obtener un Secreto en una Máquina Nueva

Para obtener secretos en un nuevo entorno (asumiendo que tienes configurado Bitwarden):

```bash
cd /path/to/your/project

# Obtiene la clave de Bitwarden, descifra el secreto local (vault.json) y crea el .env
env-vault get --use-bitwarden

# Si se encuentra un .env existente, se guardará como .env.recovered.
```

-----

## 📖 Uso y Comandos

```bash
env-vault <command>
```

| Comando | Descripción |
|:---|:---|
| `save` | Cifra y almacena el archivo `.env` para el repositorio actual. |
| `get` | Recupera y descifra el archivo `.env` para el repositorio actual. |
| `rotate-pass` | Rota la contraseña maestra para el repositorio actual. Útil para actualizar la clave de Bitwarden. |
| `generate-pass` | Genera una nueva contraseña maestra segura de 16 caracteres. |
| `verify` | Verifica la integridad HMAC de todos los secretos en `vault.json` (requiere la contraseña). |
| `sync-save` | Sube el secreto cifrado para el repo actual a Firestore. |
| `sync-load` | Descarga todos los secretos de Firestore al vault local. |

### Opciones Globales

| Opción | Descripción |
|:---|:---|
| `--use-bitwarden` | Utiliza Bitwarden para obtener la contraseña maestra en lugar de solicitarla. |
| `--repo=NAME` | Especifica el nombre del repositorio (p. ej., `user/repo`) para `rotate-pass`. |

-----

## 🔧 Configuración de Proveedores

`env-vault` requiere que se defina al menos un método para obtener la contraseña maestra: de forma manual o a través de un proveedor de secretos.

### 🔑 Bitwarden Secrets Manager

`env-vault` utiliza el Bitwarden CLI (`bws`) para gestionar la contraseña maestra del vault.

| Variable | Descripción | Requerido |
|:---|:---|:----:|
| `BWS_PROJECT_ID` | El ID del proyecto de Bitwarden Secrets Manager donde se almacenarán las claves.  | ✅ Sí |

El secreto se almacena en Bitwarden bajo una clave con el formato `env-vault-<hash-del-repo>`.

### ☁️ Google Firestore (Sincronización de Vault)

Para la sincronización del vault cifrado, debes configurar la autenticación de Google Cloud.

| Variable | Descripción | Requerido |
|:---|:---|:----:|
| `GOOGLE_APPLICATION_CREDENTIALS` | Ruta al archivo JSON de la clave de la cuenta de servicio de Google Cloud. | ✅ Sí (para `sync-save`/`sync-load`) |

**Nota Importante:** El código original mencionaba credenciales de GitHub para `login`. Dado que no se incluyó la función `login` en el código, hemos eliminado las variables `GITHUB_CLIENT_ID` y `GITHUB_CLIENT_SECRET`. **El flujo de trabajo actual asume autenticación manual o a través de Bitwarden/Firestore, no de GitHub OAuth.**

-----

## 🔒 Seguridad

  - 🔑 Las contraseñas maestras nunca se almacenan en el disco. Se limpian de la memoria (`secureZero`) inmediatamente después de usarse. 
  - 🛡️ **Verificación HMAC:** Cada secreto tiene un HMAC verificado con la clave maestra para asegurar la integridad y detectar la manipulación. 
  - 🔐 El archivo `vault.json` utiliza permisos restringidos (0600) y bloqueo de archivos (`flock`) durante la escritura. 
  - 🔄 Cada entrada utiliza un **salt único** para la derivación de claves Argon2id, haciendo que cada descifrado sea resistente. 

-----

## 🛠️ Desarrollo

### Prerrequisitos

  - Go 1.24+
  - Make (opcional, para comandos de build)

### Building

```bash
# Standard build
make build

# Build y corre un comando de ejemplo
make run ARGS="save"
```

### Running Tests

```bash
go test ./...
```

## 🤝 Contributing

1.  Fork the repository
2.  Create your feature branch (`git checkout -b feature/amazing-feature`)
3.  Commit your changes (`git commit -m 'Add amazing feature'`)
4.  Push to the branch (`git push origin feature/amazing-feature`)
5.  Open a Pull Request

## 📄 License

MIT License - see [LICENSE](https://www.google.com/search?q=./LICENSE) file

## 🙏 Acknowledgments

  - **Argon2id** for robust key derivation 
  - **AES-256-GCM** for strong encryption 
  - **Google Cloud** for Firestore integration 
