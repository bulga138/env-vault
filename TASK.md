# 🚀 Enterprise Features Roadmap

This document outlines the major enterprise-level features to be developed for `env-vault`. Each item represents a significant module that builds upon the core functionality.

---

### ## ✅ Task List

- [x] **Team-Based Access Control (ACL)** (basic)
  - Implement a system for creating teams and assigning roles (e.g., `admin`, `member`, `read-only`). Secrets should be shareable and accessible based on team membership and permissions.

- [x] **Audit Logging**
  - Create a secure, append-only log that records all critical actions. This includes who accessed a secret, who updated a secret, user logins, and administrative changes.

- [x] **Cloud Backup & Synchronization** (backup/restore + Firestore sync)
  - Integrate with a cloud storage provider (like AWS S3 or Google Cloud Storage) to enable automatic backup and synchronization of the `vault.json` file. This is crucial for team collaboration and disaster recovery.

- [ ] **Admin Dashboard (Web UI)**
  - Develop a simple web-based dashboard for administrators to manage users, teams, and view audit logs. This provides a user-friendly interface for system management.

- [x] **User Management System** (basic local registry)
  - Build functionality for inviting new users to the system, revoking access, and managing user profiles. This would likely integrate with the team-based access control system.