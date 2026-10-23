# Features Roadmap

This document outlines planned features for `env-vault`. Each item represents a module that builds on the core functionality.

---

## Task List

- [x] **Team-Based Access Control (ACL)** (basic)
- Implement a system for creating teams and assigning roles (e.g., `admin`, `member`, `read-only`). Secrets should be shareable and accessible based on team membership and permissions.

- [x] **Audit Logging**
- Create a secure, append-only log that records all critical actions. This includes who accessed a secret, who updated a secret, user logins, and administrative changes.

- [x] **Cloud Backup & Synchronization** (backup/restore + Firestore sync)
- Integrate with a cloud storage provider (like AWS S3 or Google Cloud Storage) to enable automatic backup and synchronization of the `vault.json` file. Used for team collaboration and disaster recovery.
- Pluggable sync backends: `file` (default), `sqlite`, `firestore`, and `p2p`/`peer`/`relay`/`rendezvous`.
- Current cloud backend is Firestore only; S3/GCS are not yet implemented.

- [ ] **Admin Dashboard (Web UI)**
- Develop a web-based dashboard for administrators to manage users, teams, and view audit logs.

- [x] **User Management System** (basic local registry)
- Build functionality for inviting new users to the system, revoking access, and managing user profiles. This would likely integrate with the team-based access control system.

---

### Alternative Providers (not yet implemented)

- [ ] **Alternative secret providers** (e.g. AWS Secrets Manager, 1Password, HashiCorp Vault) beyond Bitwarden and OS keyring.
- [ ] **Alternative cloud sync backends** (e.g. AWS S3, Google Cloud Storage) beyond Firestore.

- [x] **OS Keyring integration** (`--use-keyring`)
- Store/retrieve the master password in the OS keyring (via `zalando/go-keyring`) instead of env vars/prompts.

- [x] **Encrypted at-rest storage (application-level AES-GCM)**
- Encrypt sync DB payloads with AES-GCM using SYNC_DB_PASSPHRASE (sqlcipher requires CGO).

- [ ] **Secure memory handling (memguard)** (deferred)
- Lock password/key bytes in memory and prevent swapping. (memguard caused runtime faults; needs refactor to LockedBuffer throughout.)

- [x] **Recovery phrases (BIP39 mnemonic)**
- Generate and recover master keys from a mnemonic seed phrase.

- [x] **Peer-to-peer sync (relay/rendezvous)**
- Sync vaults directly between trusted devices without cloud providers.
