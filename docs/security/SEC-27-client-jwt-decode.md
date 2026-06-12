# SEC-27 — Client decodes unverified JWT payload

- **Severity:** Informational
- **Status:** Confirmed present
- **Area:** Web client
- **File:** [clients/web/src/lib/auth.ts](../../clients/web/src/lib/auth.ts) (`getJwtSubject`)

## Problem

`getJwtSubject` base64-decodes the JWT payload in the browser without verifying the signature. The return value is used only for cosmetic purposes (display), which is fine — but there is no comment marking the value as untrusted, so a future change could mistakenly use it for an authorization or gating decision on the client.

## Risk

Informational. The decoded claims are attacker-controllable (the client can hold any token shape it likes), so any client-side authorization built on them would be trivially bypassable. No such misuse exists today.

## Fix

1. Add a clear comment that the decoded payload is **unverified and must never drive authorization** — it is display-only.
2. Optionally rename to `getUnverifiedJwtSubject` to make the contract obvious at every call site.
3. Keep all authorization decisions server-side (they are today).

## Verification

- The function is documented as unverified/display-only.
- No client-side branch grants access based on decoded JWT claims.
