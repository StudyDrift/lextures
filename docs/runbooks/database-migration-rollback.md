# Runbook — Rolling Back a Database Migration (Plan 17.10)

Use this runbook when a migration reached staging or production and causes errors
(schema mismatch, constraint too strict, data loss risk). Goal: restore a working
schema in **under 10 minutes** without a full backup restore when possible.

## Prerequisites

- `DATABASE_URL` for the target environment
- `kubectl` / SSH access to run commands against the app host or a migration jump box
- Git checkout matching the deploy under investigation
- `DEPLOY_ID` from the failing deploy (Git SHA / release tag) for correlation

## Decision tree

```
Bad migration applied?
├─ Simple additive migration (CREATE TABLE / ADD nullable column) with tested down.sql
│  └─ Path A: Manual rollback (fast)
├─ Data migration or contract step (DROP/RENAME/TYPE change)
│  └─ Path B: Corrective forward migration (preferred)
└─ Unknown / down.sql is a stub
   └─ Path C: Backup restore (see database-backup-restore.md)
```

## Path A — Manual rollback (down.sql)

**When:** Latest migration is additive and has executable SQL in its `.down.sql`
(not a "Rollback not supported" stub). Tested in staging first.

1. **Stop new deploys** — pause CI/CD or hold the canary so no new instances start
   with the bad migration while you recover.

2. **Drain traffic** (production) — remove instances from the load balancer or scale
   to zero so no requests hit a schema-incompatible binary.

3. **Verify the target migration:**

   ```bash
   cd server
   psql "$DATABASE_URL" -c \
     "SELECT version, description, deploy_id, installed_on FROM _sqlx_migrations ORDER BY version DESC LIMIT 3;"
   ```

4. **Dry-run the down SQL** (staging only):

   ```bash
   latest=$(ls migrations/*.sql | grep -v down | sort -V | tail -1)
   down="${latest%.sql}.down.sql"
   cat "$down"
   ```

5. **Execute rollback:**

   ```bash
   DATABASE_URL=... go run ./cmd/migrate rollback
   ```

   Exit codes: `0` success, `1` error, `3` rollback not supported (use Path B or C).

6. **Verify schema** — confirm the rolled-back object is gone or reverted:

   ```bash
   psql "$DATABASE_URL" -c "\d settings.device_push_tokens"   # example
   ```

7. **Smoke test** — start the **previous** app version (before the bad migration)
   against the database:

   ```bash
   curl -fsS http://localhost:8080/health/ready
   ```

8. **Re-deploy** — ship a fixed forward migration in a new PR; do not re-apply the
   bad migration file.

### Timing target

| Step | Target |
|---|---|
| Identify latest migration | 1 min |
| Rollback command | 2 min |
| Smoke test | 5 min |

## Path B — Corrective forward migration

**When:** The bad migration altered or deleted data, renamed columns, or the
`down.sql` is a stub. This is the **preferred production path** for contract-phase
changes.

1. Revert the application deploy to the last known-good release (traffic on old code).

2. Author a **new** migration that restores compatibility:

   ```sql
   -- 346_restore_device_push_tokens.sql
   -- Corrective migration: re-add column dropped prematurely in 345_bad.sql
   ALTER TABLE settings.device_push_tokens ADD COLUMN IF NOT EXISTS token TEXT;
   ```

3. Include a `down.sql` (may be stub if rollback is restore-from-backup).

4. Test on a staging snapshot:

   ```bash
   DATABASE_URL=... go test ./internal/migrate/... -run TestRun_FullMigrations -count=1
   ```

5. Apply via normal deploy (`RUN_MIGRATIONS=true`).

## Path C — Backup restore

**When:** Data was corrupted, down.sql is unsupported, or Paths A/B are unsafe.

Follow `docs/runbooks/database-backup-restore.md`. Expect longer RTO and possible
data loss since the last backup.

## Staging validation (required before production)

Run this checklist whenever adding executable `down.sql` to a new migration:

1. Apply all migrations on a fresh DB.
2. `go run ./cmd/migrate rollback` — confirm success.
3. `go run ./cmd/server` (or integration tests) — confirm app starts.
4. Re-apply migrations — confirm idempotency.

CI runs `TestRollbackLatest_Integration` against Postgres when `DATABASE_URL` is set.

## Correlating incidents with deploys

After migration `345_migration_deploy_tracking`, each new row in `_sqlx_migrations`
may include `deploy_id`:

```sql
SELECT version, description, deploy_id, installed_on
FROM _sqlx_migrations
WHERE installed_on > now() - interval '24 hours'
ORDER BY version DESC;
```

Set `DEPLOY_ID` in the deployment environment to populate this field.

## Related documents

- `server/migrations/README.md` — expand/contract pattern and naming
- `docs/adr/0002-expand-contract-migrations.md` — architecture decision
- `docs/runbooks/database-backup-restore.md` — full restore procedure
