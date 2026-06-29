# ADR 0002 — Expand/Contract Migrations Over Automatic Down-Migrations

- **Status:** Accepted
- **Date:** 2026-06-29
- **Plan:** [17.10 Database Migration Rollback Strategy](../completed/17-platform-performance-operability/17.10-database-migration-rollback.md)

## Context

Lextures applies forward-only SQL migrations at startup via an internal runner
compatible with the `_sqlx_migrations` table. Production incidents from bad
migrations historically required a full database restore — slow and data-losing.
Blue/green deploys (plan 17.9) additionally require schema changes that remain
compatible with the previous app version during the canary window.

Two rollback strategies were considered:

1. **Traditional up/down** — every migration has reversible SQL; a `rollback`
   command reverts the latest version automatically.
2. **Expand/contract with corrective forward migrations** — schema changes are
   staged across multiple deploys; rollbacks are either manual `down.sql` for
   simple additive changes or a new forward migration that restores the prior state.

## Decision

Adopt **expand/contract as the default development process**, with **companion
`down.sql` files required for every migration** but automatic rollback limited to
recent, tested additive changes.

Rationale:

1. Expand/contract is the only safe pattern for zero-downtime blue/green deploys;
   a single reversible `down.sql` cannot undo data migrations or contract steps
   without data loss.
2. Requiring `down.sql` (even as documented stubs) forces authors to consider
   rollback before merge and gives on-call a starting point for manual recovery.
3. Corrective forward migrations are safer for production because they follow the
   same tested apply path as normal deploys and do not delete `_sqlx_migrations`
   history rows silently.
4. The existing `_sqlx_migrations` checksum model already prevents editing
   applied migrations; down files are not executed automatically, avoiding
   accidental schema drift on restart.

## Consequences

- All new migrations must include `NNN_*.down.sql`; CI blocks PRs without them.
- Destructive DDL in up migrations emits CI warnings; reviewers must confirm an
  expand phase already shipped.
- `go run ./cmd/migrate rollback` is available for emergencies on migrations with
  executable down SQL; stubs direct operators to the runbook and backup restore.
- `DEPLOY_ID` is recorded on new migration rows for incident correlation.
- Large backfills must use batched `UPDATE` (documented in `server/migrations/README.md`).

## Alternatives considered

- **sqlx `migrate revert`** — not used; our runner is custom and shares the
  `_sqlx_migrations` table with a legacy Rust service. Revert semantics differ and
  risk diverging checksum history.
- **ORM-generated migrations** — rejected (plan non-goal); SQL files remain the
  source of truth for auditability.
