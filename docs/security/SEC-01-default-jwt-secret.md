# SEC-01 — Default `JWT_SECRET` committed in `docker-compose.yml`

- **Severity:** Critical
- **Status:** Confirmed present (audit date)
- **Area:** Server / secrets management
- **File:** [docker-compose.yml:58](../../docker-compose.yml), and [docker-compose.dev.yml:25](../../docker-compose.dev.yml), [docker-compose.e2e.yml:57](../../docker-compose.e2e.yml)

## Problem

The default compose file hard-codes the JWT signing key:

```yaml
JWT_SECRET: change-me-in-production-use-a-long-random-string
```

`config.Validate()` only requires the secret to be ≥ 32 characters (`server/internal/config/config.go`). This literal is 47 characters, so it passes validation. Anyone who runs `docker compose up` with the default file — or copies the file and forgets to override the value — boots the server with a **globally known signing key**.

`docker-compose.dev.yml` (`dev-only-change-me-in-production`) and `docker-compose.e2e.yml` (`e2e-test-secret-do-not-use-outside-tests`) have the same shape. The deploy/prod files correctly require `${JWT_SECRET}` from the environment, so the risk is the default/dev files being used somewhere internet-reachable.

## Risk

JWTs are HS256-signed with this single symmetric key. If the key is known, an attacker can **forge an access token for any user, including a Global Admin**, with no credentials and no interaction. This is the cleanest possible path to full tenant takeover and exactly the "replay a forged credential out-of-band" pattern this audit targets.

## Fix

1. Remove the literal default. Require the secret from the environment so the stack fails closed:
   ```yaml
   JWT_SECRET: ${JWT_SECRET:?JWT_SECRET is required}
   ```
2. In `config.Load()`, reject any `JWT_SECRET` matching a known-default denylist (`change-me*`, `dev-only-change-me*`, `e2e-test-secret*`, `dev-secret-do-not-use*`) unless an explicit `ALLOW_INSECURE_JWT=1` escape hatch is set (for local dev only).
3. Rotate `JWT_SECRET` on any environment that has *ever* booted with a default value, which invalidates all existing sessions. See SEC-13 for a `kid`-based rotation scheme that makes this non-disruptive in the future.

## Verification

- `grep -rn 'change-me\|do-not-use\|dev-only' docker-compose*.yml` returns nothing in any file used outside a developer laptop.
- Booting the server with `JWT_SECRET=change-me-in-production-use-a-long-random-string` exits non-zero.
- A token signed with the old default secret is rejected with 401 after rotation.
