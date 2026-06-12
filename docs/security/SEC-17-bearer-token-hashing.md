# SEC-17 — SCIM / OneRoster bearer tokens stored as unsalted SHA-256

- **Severity:** Medium
- **Status:** Confirmed present
- **Area:** Server / provisioning
- **File:** [server/internal/provisioning/scim/bearer.go](../../server/internal/provisioning/scim/bearer.go)

## Problem

Provisioning bearer tokens are 32 random bytes (good entropy), but they are stored and compared as a bare `sha256(token)`:

```go
h := sha256.Sum256([]byte(t))
```

There is no salt and no pepper. For high-entropy tokens an unsalted hash is *acceptable* against precomputation, but it provides no defense-in-depth if the token table leaks alongside any place the plaintext token might have been logged (proxy logs, error traces), and there is no per-token usage metadata to detect anomalous reuse.

## Risk

If the provisioning-token table is exfiltrated and a token ever appeared in a log or proxy capture, validation against the stored hash is trivial. SCIM/OneRoster tokens are exactly the kind of long-lived integration secret ShinyHunters-style actors hunt for, because they grant bulk directory access.

## Fix

1. Store tokens with a keyed construction: `HMAC-SHA256(serverPepper, token)` (pepper held outside the DB, in env/secrets manager), or Argon2id if you want memory-hardness.
2. Add a `last_used_at` column; surface tokens that are dormant for > 30 days and then suddenly used, and support revocation.
3. Ensure the plaintext token is shown exactly once at creation and never logged.

## Verification

- Stored token values are HMAC/Argon2 outputs that cannot be validated without the server pepper.
- `last_used_at` updates on each successful authentication.
- Token revocation immediately rejects subsequent requests.
