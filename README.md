# env-vault

**Secure .env file manager with encryption and version control integration**

`env-vault` is a command-line tool that encrypts `.env` files and stores them outside Git, linked to a repository.

## Features

- **Strong Encryption**: Uses Argon2 key derivation and **AES-256-GCM**
- **Integrity Verification**: Command to verify that secrets haven't been tampered with (`verify`)
- **Secret Providers**: Integrates with **Bitwarden Secrets Manager** for automatic password fetching.
- **Git Integration**: Automatically detects repository context (`user/repo` format)
- **Cloud Sync**: Optional backend integration with Google Firestore
- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Secure Storage**: Uses local vault with file locking and audit logs

-----

## Installation

### Pre-compiled binaries

Download the latest release from the **GitHub Releases** page.

### From source

```bash
# Clone the repository
git clone [https://github.com/yourusername/env-vault.git](https://github.com/yourusername/env-vault.git)
cd env-vault

# Build the project (requires Go 1.24+)
make build

# Or compile and install
go install .
```

-----

## Quick Start

The recommended flow is to use an external secret provider (like Bitwarden) to store your vault's master key.

### 1\. Bitwarden Configuration (Recommended)

Follow the steps to set up your account and obtain your **Bitwarden Secrets Manager Project ID**.

```bash
# Export your Bitwarden Project ID
export BWS_PROJECT_ID="your_bitwarden_project_id"

# Ensure the 'bws' CLI is authenticated
bws login
```

### 2\. Save and Sync a New Secret

To add a new `.env` file to your vault:

```bash
# Ensure your .env file is in the current directory
cd /path/to/your/project

# Save and use Bitwarden (saves the secret locally and the key in Bitwarden)
# If a key for this repo doesn't exist, a new one will be generated, and you
# will be prompted to save it in Bitwarden.
env-vault save --use-bitwarden

# Optional: Synchronize the encrypted secret with Firestore
env-vault sync-save
```

### 3\. Retrieve a Secret on a New Machine

To retrieve secrets in a new environment (assuming you have Bitwarden configured):

```bash
cd /path/to/your/project

# Retrieves the key from Bitwarden, decrypts the local secret (vault.json), and creates the .env file
env-vault get --use-bitwarden

# If an existing .env is found, it will be saved as .env.recovered.
```

-----

## Usage and Commands

```bash
env-vault <command>
```

| Command | Description |
|:---|:---|
| `save` | Encrypts and stores the `.env` file for the current repository. |
| `get` | Retrieves and decrypts the `.env` file for the current repository. |
| `rotate-pass` | Rotates the master password for the current repository. Useful for updating the key in Bitwarden. |
| `generate-pass` | Generates a new, secure 16-character master password. |
| `verify` | Verifies the HMAC integrity of all secrets in `vault.json` (requires the password). |
| `sync-save` | Uploads the encrypted secret for the current repo to Firestore. |
| `sync-load` | Downloads all secrets from Firestore and merges them into the local vault. |
| `users` | Manage vault users (`list`, `add`, `remove`). |
| `acl` | Manage per-repo access control (`grant`, `revoke`, `list`). |
| `audit` | View audit logs (`list`, `tail`) with optional filters. |
| `keyring` | Manage OS keyring entries (`set`, `get`, `delete`, `status`). |
| `mnemonic` | Generate or recover a BIP39 recovery phrase. |
| `backup` | Create a rotated backup of the vault (keeps 10 most recent). |
| `restore` | Restore a vault backup (confirms overwrite unless `--force`). |
| `p2p` / `peer` / `relay` | Peer-to-peer sync via relay/rendezvous. |
| `banner` | Print the local `beNotAfraid.txt` ASCII art. |

### Global Options

| Option | Description |
|:---|:---|
| `--use-bitwarden` | Uses Bitwarden to fetch the master password instead of prompting for it. |
| `--sync-strategy` | Conflict resolution for `sync-load`: `newest`, `remote`, `local`, or `skip`. |
| `--force` | Skip confirmation prompts (e.g. `restore`). |

### Environment Variables (non-interactive)

| Variable | Description |
|:---|:---|
| `ENV_VAULT_PASSWORD` | Master password for `save` / `get` / `verify` / `rotate-pass` (skips the prompt). Useful for CI. |
| `ENV_VAULT_NEW_PASSWORD` | New master password for non-interactive `rotate-pass`. |
| `SYNC_DB_PASSPHRASE` | Passphrase used to encrypt the SQLite sync backend. |

-----

## Provider Configuration

`env-vault` requires you to define at least one method to obtain the master password: manually or through a secret provider.

### Bitwarden Secrets Manager

`env-vault` uses the Bitwarden CLI (`bws`) to manage the vault's master password.

| Variable | Description | Required |
|:---|:---|:----:|
| `BWS_PROJECT_ID` | The Bitwarden Secrets Manager Project ID where the keys will be stored. | Yes |

The secret is stored in Bitwarden under a key with the format `env-vault-<repo-hash>`.

### Google Firestore (Vault Synchronization)

For encrypted vault synchronization, you must configure Google Cloud authentication.

| Variable | Description | Required |
|:---|:---|:----:|
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to the Google Cloud service account key JSON file. | Yes (for `sync-save`/`sync-load`) |

**Note on Authentication:** The current workflow is designed for manual authentication or using the integrated Bitwarden/Firestore providers.

-----

## Sync Backends

Select a backend with `SYNC_BACKEND` (default `file`):

| Backend | Description |
|:---|:---|
| `file` | Local JSON files in the user config directory (default). |
| `sqlite` | Local SQLite database; set `SYNC_DB_PASSPHRASE` to encrypt payloads. |
| `firestore` | Google Firestore (requires `GOOGLE_APPLICATION_CREDENTIALS`). |
| `p2p` / `peer` / `relay` / `rendezvous` | Peer-to-peer sync via relay/rendezvous. |

Example (local SQLite):

```bash
export SYNC_BACKEND=sqlite
export SYNC_DB_PASSPHRASE="a strong passphrase"
env-vault sync-save
env-vault sync-load --sync-strategy=newest
```

## Secret Providers

- **Bitwarden Secrets Manager**: set `BWS_PROJECT_ID` and use `--use-bitwarden`.
- **OS keyring**: use `--use-keyring` to store/retrieve the master password in the OS keyring.
- **Interactive/env**: use `ENV_VAULT_PASSWORD` or the password prompt.

-----

- Master passwords are never stored on disk. They are securely cleared from memory (`secureZero`) immediately after use.
- **HMAC Verification:** Each secret has an HMAC verified with the master key to ensure integrity and detect tampering.
- The `vault.json` file uses restricted permissions (0600) and file locking (`flock`) during writing.
- Each entry uses a **unique salt** for Argon2id key derivation, making each decryption resistant.

-----

## Development

### Prerequisites

- Go 1.24+
- Make (optional, for build commands)

### Building

```bash
# Standard build
make build

# Build and run an example command
make run ARGS="save"
```

### Running Tests

```bash
# Unit + e2e (temp git repos, no external services)
go test ./...
# or
make test

# Integration tests against real Firestore + Bitwarden (uses credentials from .env)
make test-integration
```

Integration tests skip automatically when credentials/CLI are missing. Firestore writes use `FIRESTORE_COLLECTION` (default `secrets`; tests set `env-vault-e2e`). Bitwarden secrets are created under unique keys and deleted afterwards.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see the [**LICENSE**](LICENSE) file

## Acknowledgments

- **Argon2id** for key derivation
- **AES-256-GCM** for strong encryption
- **Google Cloud** for Firestore integration
