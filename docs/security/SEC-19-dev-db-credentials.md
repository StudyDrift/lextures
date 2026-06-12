# SEC-19 — Hardcoded dev DB credentials + host-exposed Postgres port

- **Severity:** Medium
- **Status:** Confirmed present
- **Area:** Infra / local dev
- **Files:** [docker-compose.yml:10-14](../../docker-compose.yml), [docker-compose.dev.yml](../../docker-compose.dev.yml)

## Problem

The default compose stack uses static credentials `studydrift / studydrift` and binds Postgres to all host interfaces:

```yaml
ports:
  - "5432:5432"
```

`5432:5432` (without a `127.0.0.1` prefix) listens on every interface of the developer's machine, with a credential pair that is published in the repo. The same applies to RabbitMQ management (`15672`) and ClamAV (`3310`).

## Risk

On a shared or untrusted network (coffee-shop wifi, office LAN), anyone who can reach the laptop can connect to Postgres with known credentials. If a developer ever loads production-shaped data locally for debugging, that data is exposed on the network. This is a lateral-movement and data-exposure footgun, not a server bug.

## Fix

1. Bind data-store ports to loopback only: `127.0.0.1:5432:5432`, `127.0.0.1:15672:15672`, etc.
2. Generate per-developer credentials in a gitignored `.env.local` rather than shipping `studydrift/studydrift` in the committed compose file.
3. Document that the bundled credentials are for ephemeral local use only and must never be reused in any shared environment.

## Verification

- `docker compose up` exposes Postgres only on `127.0.0.1`, confirmed with `ss -tlnp | grep 5432`.
- A second machine on the LAN cannot reach `5432` on the developer host.
