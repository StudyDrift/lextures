# SEC-23 — Login timing oracle enables user enumeration

- **Severity:** Low
- **Status:** Confirmed present
- **Area:** Server / auth
- **File:** [server/internal/service/authservice/credentials.go](../../server/internal/service/authservice/credentials.go) (login path ~L128–L144)

## Problem

On login, `user.FindByEmail` is called first; if it returns nil (unknown email) the handler returns immediately. Only when the email exists does the code run `pauth.VerifyPassword`, which performs an Argon2id verification (~250 ms today, more after SEC-14):

```go
row, err := user.FindByEmail(ctx, pool, email)   // returns fast if unknown
// ...
ok, err := pauth.VerifyPassword(req.Password, row.PasswordHash) // ~250ms, only if known
```

The response-time difference between "known email" (slow, runs Argon2) and "unknown email" (fast, no Argon2) is a reliable timing distinguisher.

## Risk

An attacker can enumerate which email addresses are registered users by measuring response latency, then focus credential-stuffing (SEC-04) on confirmed accounts. User enumeration is a recon primitive that precedes the access steps in this threat model.

## Fix

Always perform an Argon2id verification, even when the email is unknown, against a constant dummy hash:

```go
if row == nil {
    _, _ = pauth.VerifyPassword(req.Password, dummyArgon2Hash) // burn equivalent time
    return ErrInvalidCredentials
}
```

Keep error messages identical for both cases. This equalizes the timing and is cheap once rate limiting (SEC-04) caps the request volume.

## Verification

- Response-time distributions for known vs. unknown emails are statistically indistinguishable.
- Both cases return the same generic error body and status.
