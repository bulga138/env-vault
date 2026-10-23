# env-vault quick start (local notes)

## First-time setup
env-vault init
env-vault doctor

## Save / get a repo secret
cd /path/to/project
env-vault save
env-vault get
env-vault verify

## Use OS keyring instead of prompts
env-vault save --use-keyring
env-vault get --use-keyring

## Sync backends
export SYNC_BACKEND=file      # default
export SYNC_BACKEND=sqlite
export SYNC_DB_PASSPHRASE="strong-passphrase"
export SYNC_BACKEND=firestore
env-vault sync-save
env-vault sync-load --sync-strategy=newest

## Users
env-vault users list
env-vault users add alice admin
env-vault users remove alice

## Access control
env-vault acl grant alice owner/repo write
env-vault acl revoke alice owner/repo
env-vault acl list

## Audit
env-vault audit list
env-vault audit tail --audit-limit 20

## Backup / restore
env-vault backup
env-vault restore --force

## Admin TUI
env-vault admin

## Recovery
env-vault mnemonic generate
env-vault mnemonic recover

## P2P sync
env-vault p2p
env-vault peer
env-vault relay

## Config location
Windows: %APPDATA%\env-vault\env-vault.env
Linux/macOS: ~/.config/env-vault/env-vault.env
Vault data: same config directory (vault.json, users.json, acl.json, audit/)

## Tips
- Run `env-vault doctor` if something is not detected.
- Use `ENV_VAULT_PASSWORD` for non-interactive/CI runs.
- `--repo=NAME` overrides repo detection.
