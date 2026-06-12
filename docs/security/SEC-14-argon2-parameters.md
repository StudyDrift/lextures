# SEC-14 — Argon2id parameters below current OWASP guidance

- **Severity:** Medium
- **Status:** Confirmed present
- **Area:** Server / auth
- **File:** [server/internal/auth/password.go](../../server/internal/auth/password.go)

## Problem

Password hashing uses parameters inherited from the legacy Rust `Argon2::default()`:

```go
var rustArgon2idParams = &argon2id.Params{
    Memory:      19456, // 19 MiB
    Iterations:  2,
    Parallelism: 1,
    SaltLength:  16,
    KeyLength:   32,
}
```

These are below current OWASP Argon2id guidance (e.g. ≥ 47 MiB memory, or m=12 MiB / t=3 / p=1 as a minimum floor).

## Risk

If the user table is exfiltrated, weaker KDF parameters make offline cracking cheaper. This matters specifically in the breach scenario this audit targets: a dumped `password_hash` column should be as expensive as possible to attack.

## Fix

1. Raise to an OWASP-aligned profile, benchmarked on production hardware so `HashPassword` takes ~250–500 ms (e.g. `Memory: 47104, Iterations: 1, Parallelism: 1`, or `Memory: 19456, Iterations: 3`). Tune to the latency budget.
2. Transparent rehash on next successful login: after verifying with the stored parameters, if they are below the current target, re-hash the plaintext (already in hand) and update the row. The `argon2id` PHC string encodes its own params, so old and new hashes coexist.

## Verification

- New signups produce hashes with the upgraded parameters (visible in the PHC string).
- An existing user logging in with a legacy hash gets transparently upgraded.
- `HashPassword` latency is within the chosen 250–500 ms target on prod hardware.
