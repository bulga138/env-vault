# env-vault 🛡️

**Secure .env file manager with encryption and version control integration**

`env-vault` is a command-line tool that securely encrypts and stores your environment variables, linking them to specific Git repositories. Never worry about accidentally committing sensitive credentials again!

## 🌟 Features

- 🔐 **Strong Encryption**: Uses Argon2 key derivation + AES-256-GCM
- 🔄 **Git Integration**: Automatically detects repository context
- 🔑 **Password Validation**: Enforces strong master password requirements
- 🔗 **GitHub Authentication**: Secure OAuth2 integration
- 📱 **Cross-Platform**: Works on Windows, macOS, and Linux
- ☁️ **Cloud Sync**: Optional backend integration with Google Firestore
- 🛡️ **Secure Storage**: OS keyring integration and file locking

## 📦 Installation

### Pre-compiled binaries

Download the latest release from [GitHub Releases](https://github.com/yourusername/env-vault/releases)

### From source

```bash
# Clone the repository
git clone https://github.com/yourusername/env-vault.git
cd env-vault

# Build the project
make build

# Or compile and install
go install .
```

### Required Setup

Set up your GitHub OAuth application credentials:

``` bash
export GITHUB_CLIENT_ID="your_github_client_id"
export GITHUB_CLIENT_SECRET="your_github_client_secret"
```

## 🚀 Quick Start

### Step 1: Login

First, authorize the application with your GitHub account. This is a one-time setup.

```bash
env-vault login
```

### Step 2: Save and Sync a New Secret

To add a new `.env` file to the vault:

```bash
# Make sure your .env file is in the current directory
cd /path/to/your/project

# 1. Save it to your local vault (creates/updates vault.json)
env-vault save

# 2. Sync it to the shared Firestore database
env-vault sync-save
```

### Step 3: Get a Secret on a New Machine

To get secrets onto a new computer: 

```bash
# 1. Sync all secrets from Firestore to your local vault.json
env-vault sync-load

# 2. Get the specific .env file for your current project.
# This will create a "./project-name/.env" file.
cd /path/to/your/project
env-vault get
```

## 📖 Usage

``` bash
env-vault <command>

Commands:
  login   - Authenticate with your GitHub account
  save    - Encrypt and store the .env file for current repository
  get     - Retrieve and decrypt the .env file for current repository
  generate-pass - Generates a new secure master password

Cloud Sync Commands
  sync-save - Uploads the secret for the current repo to Firestore
  sync-load - Downloads all secrets from Firestore to the local vault
```

### Password Requirements
For new vault entries, passwords must contain:

- At least 8 characters
- At least one uppercase letter
- At least one lowercase letter
- At least one number
- At least one symbol

## 🔧 Configuration

### Environment Variables

|Variable|Description|Required|
|:---|:---|:----:|
GITHUB_CLIENT_ID|GitHub OAuth App Client ID|	✅ Yes||
GITHUB_CLIENT_SECRET|GitHub OAuth App Client Secret|✅ Yes||
GOOGLE_APPLICATION_CREDENTIALS|Path to your Google Cloud service account JSON key|    ☑️ Optional|

### GitHub OAuth App Setup

1. Go to GitHub Settings → Developer settings → OAuth Apps
2. Click "New OAuth App"
3. Set:
   - Application name: `env-vault`
   - Homepage URL: `http://localhost:8080`
   - Authorization callback URL: `http://localhost:8080/github/callback`
   - Copy the Client ID and Client Secret

## 🔒 Security

- 🔑 Master passwords are never stored
- 🔐 Each repository entry uses a unique salt for key derivation
- 🛡️ Encrypted data stored with OS-appropriate permissions (0600)
- 🔒 Access tokens stored in system keyring
- 🚫 Never commits sensitive data to version control

## ☁️ Cloud Deployment & Sync

You can optionally use Google Firestore as a centralized, secure backend for your vault, and deploy env-vault to a cloud provider like Fly.io to access it from anywhere.

### Firebase Firestore Setup

1. Create a Firebase Project: Go to the Firebase Console and create a new project.
2. Enable Firestore: In your project, navigate to the Firestore Database section and create a new database.
3. Generate Service Account Key:
    - Go to Project Settings > Service accounts.
    - Click "Generate new private key". A JSON file will be downloaded.
4. Set Environment Variable: For the application to authenticate with Google Cloud, set the following environment variable to the path of your downloaded key:
```bash
    export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your/serviceAccountKey.json"
```

### Deployment to Fly.io (On-Demand SSH Server)

This setup deploys env-vault as a "sleeper" VM that only runs when you are actively connected, keeping costs minimal.

#### 1. Prerequisites
- Install the Fly.io CLI: `curl -L https://fly.io/install.sh | sh`
- An SSH key pair (usually located at `~/.ssh/id_rsa.pub`).

#### 2. Prepare for Deployment
Create a local authorized_keys file and add the public SSH keys (one per line) of all users who should have access.

```bash
 # authorized_keys 
 ssh-rsa AAAAB3... user@machine1 
 ssh-rsa AAAAB3... user@machine2 
```

#### 3. Configure Fly.io Secrets
Run the following commands from the project's root directory to securely configure your application's environment.

```bash 
# Set the authorized SSH keys 
fly secrets set AUTHORIZED_KEYS="$(cat ./authorized_keys)"  
# Set the Google Cloud credentials 
fly secrets set GOOGLE_APPLICATION_CREDENTIALS="$(cat /path/to/your/serviceAccountKey.json)"  
# Set the GitHub OAuth credentials 
fly secrets set GITHUB_CLIENT_ID="your_client_id" GITHUB_CLIENT_SECRET="your_client_secret"
```

#### 4. Deploy and Connect

Deploy the application and then connect to it via SSH.
```bash
fly deploy fly ssh console 
```

Once connected, you can use all the env-vault commands (e.g., env-vault sync-load, env-vault get) remotely.

## 🛠️ Development
### Prerequisites

- Go 1.24+
- Make (optional, for build commands)

### Building

``` bash
# Standard build
make build

# Clean build artifacts
make clean

# Build and run
make run ARGS="save"

# Update dependencies
make tidy
```

### Running Tests
```bash
go test ./...
```

## Bitwarden Integration (Optional)

`env-vault` can integrate with your personal Bitwarden vault to securely fetch your master password, so you never have to type it.

### Setup
1.  **Install the Bitwarden CLI** by following the [official guide](https://bitwarden.com/help/cli/).
2.  **Log in to your account**:
    ```bash
    bw login
    ```
3.  **Store your `env-vault` master password** as a new item (e.g., a Password item) in your Bitwarden vault.
4.  **Find the Item ID** of the item you just created. You can get this from the URL in the web vault or by using the command `bw list items --search "your-item-name"`.
5.  **Set the environment variable**:
    ```bash
    export BITWARDEN_ITEM_ID="the-id-you-found"
    ```

### Usage
Run any command with the `--use-bitwarden` flag, or use the Makefile shortcuts (`make save-bw`, `make get-bw`). The application will prompt you for your main Bitwarden password to unlock the vault and retrieve the secret.

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (git checkout -b feature/amazing-feature)
3. Commit your changes (git commit -m 'Add amazing feature')
4. Push to the branch (git push origin feature/amazing-feature)
5. Open a Pull Request

## 📄 License
MIT License - see [LICENSE](./LICENSE) file

## 🙏 Acknowledgments

- Argon2 for password hashing
- Go Cryptography for encryption
- go-keyring for secure token storage
