# Database migrations

Versioned SQL migrations for the Lextures Postgres schema. Applied at API startup when
`RUN_MIGRATIONS=true` (or manually via `go run ./cmd/migrate`).

## Naming convention

| File | Purpose |
|---|---|
| `NNN_short_description.sql` | **Up** migration — applied in ascending numeric order |
| `NNN_short_description.down.sql` | **Down** companion — manual rollback SQL (required for every up file) |

- `NNN` is a zero-padded sequence number (e.g. `001`, `345`).
- Use lowercase snake_case for the description.
- One logical change per file when possible.

## Adding a migration

1. Pick the next sequence number: `ls server/migrations/*.sql | grep -v down | tail -1`
2. Create `NNN_your_change.sql` with the forward DDL/DML.
3. Create `NNN_your_change.down.sql` with rollback SQL, or a documented stub:

   ```sql
   -- Rollback not supported: restore from backup
   -- Companion to: NNN_your_change.sql
   -- See docs/runbooks/database-migration-rollback.md
   ```

4. Run `go run ./cmd/migrate lint` from `server/`.
5. Run `go test ./internal/migrate/...` with `DATABASE_URL` set.

## Expand / contract pattern (backward-compatible changes)

Blue/green deploys require the **old** app version to keep running while the **new**
version is canaried. Schema changes must therefore be backward-compatible during the
overlap window.

Follow the [expand / contract / migrate / contract](https://www.martinfowler.com/bliki/ParallelChange.html)
sequence:

### 1. Expand

Add new columns, tables, or indexes **without removing** old ones.

```sql
-- Expand: add nullable column; old app ignores it
ALTER TABLE course.courses ADD COLUMN IF NOT EXISTS new_field TEXT;
```

### 2. Migrate (dual-write / backfill)

Backfill data in **batched** transactions (1,000 rows per batch) to avoid long table locks:

```sql
DO $$
DECLARE
  batch_size INT := 1000;
  rows_updated INT;
BEGIN
  LOOP
    UPDATE course.courses
    SET new_field = legacy_field
    WHERE id IN (
      SELECT id FROM course.courses
      WHERE new_field IS NULL AND legacy_field IS NOT NULL
      LIMIT batch_size
    );
    GET DIAGNOSTICS rows_updated = ROW_COUNT;
    EXIT WHEN rows_updated = 0;
    COMMIT;  -- in a script: use separate transactions per batch
  END LOOP;
END $$;
```

Deploy app code that **writes to both** old and new columns during the transition.

### 3. Contract

Only after **all** running instances use the new schema, remove old columns in a
separate migration:

```sql
-- Contract: safe only after every instance reads/writes new_field
ALTER TABLE course.courses DROP COLUMN legacy_field;
```

Destructive operations (`DROP COLUMN`, `DROP TABLE`, `RENAME COLUMN`, `ALTER COLUMN TYPE`)
emit **CI warnings** and require explicit reviewer sign-off.

## NOT NULL and type changes

- Adding `NOT NULL` without a `DEFAULT` rewrites the whole table on Postgres — add the
  column nullable, backfill, then `SET NOT NULL` in a follow-up migration.
- Column type changes require expand/contract with a new column and batched copy.

## Rollback

Automatic down-migration is **not** run at startup. To roll back the latest migration
manually:

```bash
cd server
DATABASE_URL=postgres://... go run ./cmd/migrate rollback
```

See `docs/runbooks/database-migration-rollback.md` for production procedures.

## Deploy tracking

Set `DEPLOY_ID` (e.g. Git SHA or release tag) when applying migrations so
`_sqlx_migrations.deploy_id` correlates schema changes with deploys during incidents.

## CI checks

`go run ./cmd/migrate-lint` (also run in CI) validates:

- Every up migration has a companion `.down.sql` (**blocks** on missing files)
- No inline secrets in SQL files (**blocks**)
- Destructive DDL patterns (**warns** — requires reviewer attention)
- Full-table `UPDATE` without batching (**warns**)

## References

- ADR: `docs/adr/0002-expand-contract-migrations.md`
- Runbook: `docs/runbooks/database-migration-rollback.md`
- Implementation: `server/internal/migrate/`
