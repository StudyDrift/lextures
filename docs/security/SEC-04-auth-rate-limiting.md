# SEC-04 — No rate limiting on password / session endpoints

- **Severity:** High
- **Status:** Confirmed present
- **Area:** Server / auth
- **Files:** [server/internal/httpserver/server.go](../../server/internal/httpserver/server.go), [server/internal/httpserver/auth.go](../../server/internal/httpserver/auth.go), [server/internal/service/authservice/credentials.go](../../server/internal/service/authservice/credentials.go)

## Problem

The following endpoints accept unlimited requests with no IP- or account-keyed throttle:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/signup`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`
- `POST /api/v1/auth/refresh`

The only throttles that exist are:

- **MFA lockout** (`server/internal/service/mfaservice/service.go`): a per-user `mfa_lockout_until` after repeated TOTP/backup failures — good, but only covers the second factor.
- **Magic-link limit** (`magicLinkRateMax = 3 / 5 min`): per *known user* only, and unknown-email requests skip the count entirely (see SEC-25).

There is no general login/signup/reset throttle keyed on IP or email.

## Risk

- **Credential stuffing** against `/login` at full speed.
- **Account enumeration** via the timing oracle in SEC-23 (Argon2 only runs when the email exists).
- **Password-reset spray** and signup abuse.

These are the reconnaissance and access steps that precede the token-theft pattern this audit targets.

## Fix

Add IP- and account-keyed token-bucket middleware (e.g. `go-chi/httprate`, or a Redis bucket since production already provisions Redis). Suggested limits:

| Endpoint | Limit |
|----------|-------|
| `login` | 5 failures / 15 min per IP, 10 / hr per email |
| `signup` | 3 / hr per IP |
| `forgot-password` | 3 / hr per email + 10 / hr per IP |
| `reset-password` | 10 / hr per IP |
| `refresh` | 60 / hr per IP |

Key the IP off a trusted forwarded header — nginx already sets `X-Real-IP` / `X-Forwarded-For`; make sure the limiter reads the real client IP and not the proxy's.

## Verification

- 11 failed logins from one IP in 60 s return `429 Too Many Requests`.
- A reset-token spray for one email is throttled after 3 attempts.
- Legitimate single-user login/refresh is unaffected.
