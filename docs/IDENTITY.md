# Identity implementation: E01 credential foundation

Stage E is In progress. The first E01 slice implements password and bearer-token primitives in `internal/identity`. It does not enable owner setup, sign-in or session authentication. Existing HTTP routes retain their development-scaffold behavior; keep them bound to loopback.

## Credential format

Passwords contain 15–1024 UTF-8 bytes and are neither normalized nor truncated. `HashPassword` uses PBKDF2-HMAC-SHA256 with 600,000 iterations, a fresh 128-bit salt and a 256-bit derived key. The stored format is `pbkdf2-sha256$v1$600000$SALT$KEY`, with unpadded standard base64 fields. Verification accepts only that exact version and cost, bounds the encoded input before parsing, rejects malformed encodings and compares derived keys in constant time. There is no fallback to a weaker or caller-selected algorithm. Future format/cost changes require explicit migration and rehash support.

`NewToken` generates a 256-bit random secret encoded as unpadded base64url. `TokenDigest` validates the canonical encoding and produces SHA-256 over its decoded random bytes. Session/bootstrap storage must retain only this digest, never the bearer token. Passwords require the deliberately expensive password function; fast token hashing must not be used for passwords.

## Remaining E01 work

- Operator-authorized one-time owner setup, guarded against concurrent initialization and remote takeover. A public first-visitor claim is prohibited.
- Durable users and sessions with expiration, rotation, logout/revocation and recovery that revokes existing sessions.
- Sign-in throttling, bounded concurrent password verification and dummy verification for unknown users. The primitive alone does not prevent online guessing or account enumeration.
- Secure HttpOnly/SameSite cookies, configured public origin, CSRF checks, reverse-proxy handling and authenticated HTTP integration.
- Real PostgreSQL concurrent bootstrap, session expiry and revocation tests; positive and negative HTTP tests.

E02–E04 workspace roles, authorization and onboarding remain planned. No public authenticated API, operational identity boundary or completed E01 milestone is claimed by this slice. No schema migration is needed for the credential primitives.

## Verification

Unit tests cover fresh salts, correct and incorrect passwords, byte limits, Unicode, truncation, unsupported formats/costs, token uniqueness and canonical encoding. A fuzz target covers malformed credential inputs. The repository's normal Go vet/race/build checks apply; PostgreSQL tests belong to the durable session follow-up.
