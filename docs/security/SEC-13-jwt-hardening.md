# SEC-13 — JWT: HS256 single key, no `kid`/`iss`/`aud`, no rotation

- **Severity:** Medium
- **Status:** Confirmed present
- **Area:** Server / auth
- **File:** [server/internal/auth/jwt.go](../../server/internal/auth/jwt.go)

## Problem

All token classes (login access, refresh-linked, MFA-pending, LTI embed) are signed with a single static HS256 secret. There is no `kid` header to identify which key signed a token, and verification sets/checks neither `iss` (issuer) nor `aud` (audience).

## Risk

- **No rotation path.** Because there is one key and no `kid`, rotating the secret logs every user out simultaneously. That friction discourages rotation, which extends the breach window after a suspected leak (SEC-01) — the opposite of what you want under the ShinyHunters "rotate fast on suspicion" posture.
- **No audience separation.** If a sibling service is ever issued tokens with the same secret, they cross-validate. An MFA-pending token and a full login token are distinguished only by claims, not by a cryptographic audience binding.

## Fix

1. Introduce a key map `JWT_SECRETS_JSON = {"<kid>": "<secret>"}`. Sign with the current `kid` (set it in the JWT header); verification accepts the current and previous `kid` during an overlap window. This makes rotation a non-event.
2. Set `iss = "lextures"` and a per-class `aud` (`login`, `mfa_pending`, `lti_embed`); reject tokens whose `iss`/`aud` don't match the consuming endpoint's expectation.
3. Consider migrating to an asymmetric algorithm (RS256/EdDSA) so verifiers (e.g. LTI tool consumers) can hold only the public key.

## Verification

- A token signed under the previous `kid` still verifies during the overlap window; after the window it is rejected.
- An `mfa_pending`-audience token is rejected by a `login`-only endpoint.
- Rotating the active `kid` does not invalidate currently-valid sessions signed with the prior key.
