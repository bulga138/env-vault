# env-vault 🛡️

**Secure .env file manager with encryption and version control integration**

`env-vault` is a command-line tool that securely encrypts and stores your environment variables, linking them to specific Git repositories. Never worry about accidentally committing sensitive credentials again!

## 🌟 Features

- 🔐 **Strong Encryption**: Uses Argon2 key derivation + AES-256-GCM
- 🔄 **Git Integration**: Automatically detects repository context
- 🔑 **Password Validation**: Enforces strong master password requirements
- 🔗 **GitHub Authentication**: Secure OAuth2 integration
- 📱 **Cross-Platform**: Works on Windows, macOS, and Linux
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
copy
```

## 🚀 Quick Start

1. Authenticate with GitHub:

``` bash
    env-vault login
```
2. Create a strong master password (minimum 8 chars, uppercase, lowercase, number, symbol)

3. Store your .env file:

``` bash
    cd /path/to/your/project
    env-vault save
```

4. Retrieve your .env file:

``` bash
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
