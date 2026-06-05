# Credential Token Encryption at Rest

## Purpose

Credential tokens (PATs, kubeconfigs, service account keys) are stored in PostgreSQL. Today they are plaintext. This spec defines encryption at rest using AES-256-GCM with a versioned encryption key stored in a Kubernetes Secret, providing confidentiality if the database is compromised. This is a stepping stone toward Vault-only secret storage — the encryption layer is internal to the API server and invisible to all consumers.

> **Glossary:** "token" in this spec refers exclusively to the credential's stored secret value (PAT, kubeconfig, service account key), not HTTP bearer tokens or auth tokens.

## Requirements

### Requirement: Encrypted Storage

The API server SHALL encrypt credential tokens before writing them to PostgreSQL and decrypt them when reading. The `token` column SHALL contain only ciphertext after encryption is enabled. No consumer (sidecar, runner, SDK, CLI) SHALL be aware of the encryption — the API contract is unchanged.

Encryption and decryption SHALL occur at the service layer (inside `CredentialService`), not in handlers or presenters. This ensures all code paths that touch the token — including any future presenters — always receive plaintext after decryption and never accidentally expose ciphertext.

No DDL or schema migration is required. The existing `token` column is PostgreSQL `TEXT` (unbounded) and accommodates the ciphertext format without modification.

#### Scenario: Create credential with encryption enabled

- GIVEN the API server has a valid encryption key configured
- WHEN a user creates a credential with `provider=github` and `token=ghp_abc123`
- THEN the `token` column in PostgreSQL contains a versioned ciphertext blob (not `ghp_abc123`)
- AND the `GET /credentials/{id}/token` response returns the original plaintext `ghp_abc123`

#### Scenario: Rotate token on existing credential

- GIVEN an encrypted credential exists
- WHEN the user patches it with a new token value
- THEN the old ciphertext is replaced with a new ciphertext of the new token
- AND the key version tag reflects the current active key

#### Scenario: API contract unchanged

- GIVEN encryption is enabled
- WHEN a sidecar calls `GET /credentials/{id}/token`
- THEN it receives the same JSON shape and plaintext token as before encryption was enabled

### Requirement: Encryption Key Management

The encryption key SHALL be provided as an environment variable (`CREDENTIAL_ENCRYPTION_KEY`) sourced from a Kubernetes Secret. The key SHALL be exactly 32 bytes (256 bits), base64-encoded in the Secret.

The encryption key value MUST NOT appear in log output, error messages, API responses, or debug traces. Log messages about the key SHALL reference it by version number only (e.g., "using encryption key v2").

#### Scenario: Server startup with valid key

- GIVEN `CREDENTIAL_ENCRYPTION_KEY` is set to a valid base64-encoded 32-byte value
- WHEN the API server starts
- THEN it initializes the encryption subsystem and serves requests normally

#### Scenario: Server startup with missing key

- GIVEN `CREDENTIAL_ENCRYPTION_KEY` is not set
- AND at least one credential in the database has an encrypted token (version-prefixed ciphertext)
- WHEN the API server starts
- THEN it SHALL refuse to start and log a fatal error: "encryption key required but not configured"

#### Scenario: Server startup with no key and no encrypted tokens

- GIVEN `CREDENTIAL_ENCRYPTION_KEY` is not set
- AND no credentials in the database have encrypted tokens (all plaintext or empty)
- WHEN the API server starts
- THEN it SHALL start normally with encryption disabled
- AND write and read tokens as plaintext (backward-compatible)

### Requirement: Ciphertext Format

Each encrypted token SHALL be stored as a version-tagged string with the format:

```
enc:v{N}:{base64(nonce + ciphertext + tag)}
```

Where:
- `enc:` is a fixed prefix distinguishing ciphertext from plaintext
- `v{N}` is the key version (monotonically increasing integer, starting at `1`)
- The base64 payload contains the 12-byte GCM nonce prepended to the ciphertext and authentication tag

Plaintext tokens (pre-migration) lack the `enc:` prefix, enabling the system to distinguish encrypted from unencrypted values.

Note: a credential token that legitimately begins with the literal string `enc:` would be misidentified as ciphertext. This is not expected for any supported provider's token format (GitHub PATs start with `ghp_`/`gho_`, GitLab with `glpat-`, Jira uses ATATT, kubeconfigs start with `apiVersion`, GCP keys start with `{`).

#### Scenario: Distinguish encrypted from plaintext

- GIVEN a credential with token `enc:v1:SGVsbG8gV29ybGQ=`
- WHEN the API server reads this token
- THEN it detects the `enc:` prefix and decrypts using key version 1

#### Scenario: Legacy plaintext token

- GIVEN a credential with token `ghp_abc123` (no `enc:` prefix)
- AND `CREDENTIAL_ENCRYPTION_KEY` is configured
- WHEN the API server reads this token via `GET /credentials/{id}/token`
- THEN it returns the plaintext value as-is (no decryption needed)
- AND the token remains plaintext in the database until explicitly migrated

### Requirement: Key Rotation

The API server SHALL support rotating the encryption key via the `encrypt-credentials` CLI command, which bulk re-encrypts all tokens with the current key.

#### Scenario: Bulk re-encrypt

- GIVEN 50 credentials exist, all encrypted with key version 1
- AND the operator deploys a new encryption key (version 2) via the K8s Secret
- WHEN the operator runs `ambient-api-server encrypt-credentials`
- THEN all 50 tokens are decrypted with the version-1 key and re-encrypted with the version-2 key
- AND each token's version tag updates from `v1` to `v2`
- AND the command reports: "50 credentials re-encrypted from v1 to v2"

#### Scenario: Interrupted re-encryption

- GIVEN a bulk re-encrypt is running
- WHEN it fails after processing 30 of 50 credentials
- THEN 30 credentials are tagged `v2` and 20 remain tagged `v1`
- AND the command exits with an error listing the 20 unprocessed credential IDs
- AND a subsequent run of `encrypt-credentials` processes only the remaining `v1` credentials

#### Scenario: Key version tracking

- GIVEN a K8s Secret containing the encryption key
- WHEN the operator needs to rotate
- THEN they update the Secret with a new 32-byte key
- AND set `CREDENTIAL_ENCRYPTION_KEY_VERSION` to the new version number (e.g., `2`)
- AND the previous key is retained in `CREDENTIAL_ENCRYPTION_KEY_PREVIOUS` for decryption during the migration window
- AND `CREDENTIAL_ENCRYPTION_KEY_PREVIOUS` MUST be retained until `encrypt-credentials` completes successfully and all tokens are confirmed re-encrypted

### Requirement: Initial Migration

The `encrypt-credentials` CLI command SHALL encrypt all existing plaintext tokens in-place.

#### Scenario: First-time encryption

- GIVEN 100 credentials exist with plaintext tokens (no `enc:` prefix)
- AND `CREDENTIAL_ENCRYPTION_KEY` is configured with version 1
- WHEN the operator runs `ambient-api-server encrypt-credentials`
- THEN all 100 tokens are encrypted with the version-1 key
- AND each token is updated to `enc:v1:{ciphertext}`
- AND the command reports: "100 credentials encrypted (plaintext → v1)"

#### Scenario: Idempotent execution

- GIVEN all credentials are already encrypted with version 1
- WHEN the operator runs `encrypt-credentials` with the same version-1 key
- THEN the command reports: "0 credentials need encryption. All up to date."
- AND no database writes occur

### Requirement: Decryption Rollback

The `encrypt-credentials` command SHALL support a `--decrypt` flag that reverses all encrypted tokens to plaintext in the database.

#### Scenario: Bulk decrypt

- GIVEN 50 credentials exist with encrypted tokens (all `enc:v1:...`)
- WHEN the operator runs `ambient-api-server encrypt-credentials --decrypt`
- THEN all 50 tokens are decrypted and stored as plaintext (no `enc:` prefix)
- AND the command reports: "50 credentials decrypted to plaintext"

#### Scenario: Decrypt after partial rotation

- GIVEN 30 credentials are `enc:v2` and 20 are `enc:v1`
- AND both keys are available (`CREDENTIAL_ENCRYPTION_KEY` + `CREDENTIAL_ENCRYPTION_KEY_PREVIOUS`)
- WHEN the operator runs `encrypt-credentials --decrypt`
- THEN all 50 tokens are decrypted to plaintext using their respective key versions

### Requirement: CLI Integration

The `encrypt-credentials` command SHALL be a cobra subcommand of `ambient-api-server`, alongside the existing `serve` and `migrate` commands. It SHALL reuse the existing `SessionFactory` for database access and the standard environment system for configuration.

The command operates directly on the database — it does not go through the REST API. It is a privileged operation intended for platform operators with direct database and K8s Secret access. No application-level RBAC role grants access to this command; authorization is enforced by infrastructure access (K8s RBAC on the pod/namespace and database credentials).

Decrypted token values are never exposed to end users. The `GET /credentials/{id}/token` endpoint requires the `credential:token-reader` role, which is granted only to runner service accounts — not to human users.

#### Scenario: Command execution

- GIVEN the operator has `kubectl exec` access to the API server pod (or runs the binary locally with DB connectivity)
- WHEN they run `ambient-api-server encrypt-credentials`
- THEN the command connects to PostgreSQL, processes all credential tokens, and exits

#### Scenario: Dry run

- GIVEN credentials exist in mixed states (some plaintext, some encrypted)
- WHEN the operator runs `ambient-api-server encrypt-credentials --dry-run`
- THEN the command reports what it would do without modifying any data
- AND outputs: "Would encrypt: 30 plaintext, Would re-encrypt: 20 (v1 → v2), Already current: 50"

### Requirement: Vault Migration Path

The encryption layer SHALL be implemented as an internal concern of the credential plugin, not exposed in the API schema or OpenAPI spec. This enables a future migration to Vault-only storage by:
1. Replacing the encrypt/decrypt functions with Vault Transit API calls
2. Or replacing the token column with a Vault path reference

No API, SDK, CLI, sidecar, or runner changes SHALL be required when the storage backend changes.

#### Scenario: Future Vault adoption

- GIVEN the API server currently uses AES-256-GCM with a K8s Secret key
- WHEN the team migrates to Vault Transit
- THEN only the encryption/decryption functions inside the credential plugin change
- AND the `GET /credentials/{id}/token` response is identical
- AND no consumer needs modification

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `CREDENTIAL_ENCRYPTION_KEY` | No (see startup scenarios) | Base64-encoded 32-byte AES-256 key. Sourced from K8s Secret. |
| `CREDENTIAL_ENCRYPTION_KEY_VERSION` | When key is set | Integer version of the current key (e.g., `1`, `2`). |
| `CREDENTIAL_ENCRYPTION_KEY_PREVIOUS` | During rotation only | Base64-encoded previous key. MUST be retained until `encrypt-credentials` completes. |

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| AES-256-GCM | Authenticated encryption. Go stdlib (`crypto/aes` + `crypto/cipher`). No external dependencies. Industry standard. |
| Version-tagged ciphertext | Enables safe key rotation — the system always knows which key encrypted a given token. Also distinguishes encrypted from legacy plaintext. |
| Env var for key, not file mount | Consistent with existing API server config pattern (DB credentials use env vars). Simpler code. Rotation requires pod restart, which is acceptable for a security-critical operation. |
| Explicit CLI command for migration | Follows industry practice (Rails, Django, Kubernetes, Vault). Never auto-encrypt on startup — the operation is privileged, auditable, and must be recoverable from partial failure. |
| Encryption at service layer | Decrypt in `CredentialService.Get()`, encrypt in `CredentialService.Create()`/`Replace()`. Handlers and presenters never see ciphertext. Prevents accidental exposure if new presenters are added. |
| Encryption invisible to API consumers | The `GET /credentials/{id}/token` contract is unchanged. Sidecars, runners, SDK, CLI are unaware. This maximizes the migration surface to Vault later. |
| Graceful degradation without key | If no key is configured and no encrypted tokens exist, the server runs in plaintext mode. Backward-compatible for dev environments and existing deployments before key provisioning. |
| No DDL migration required | The `token` column is PostgreSQL `TEXT` (unbounded). Ciphertext with the `enc:v1:...` prefix fits without schema changes. |
| `--decrypt` rollback supported | The decrypt capability exists inherently (needed for `GET /token`). A `--decrypt` flag on the CLI command reverses encryption if the feature must be rolled back. |
| Cobra subcommand, not gormigrate | `encrypt-credentials` is a standalone subcommand like `serve` and `migrate`, not a numbered migration. It's re-runnable, idempotent, and supports `--dry-run` and `--decrypt` flags. |
