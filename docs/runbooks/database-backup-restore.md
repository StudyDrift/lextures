# Database backup and restore procedure

Plan 10.15 — RPO ≤ 1 hour (Postgres WAL streaming), RTO ≤ 4 hours (full restore).

## Daily operations

1. **Continuous WAL archive** — `archive_command` pushes segments via WAL-G to the encrypted backup bucket (`iac/modules/aws/backup.tf`).
2. **Daily base backup** — Cron at 02:00–04:00 UTC runs `wal-g backup-push` (see `server/backup/walg.env.example`).
3. **Heartbeat** — After each job: `go run ./cmd/backup-report -tier=postgres -success -wal-lag-seconds=N -duration-seconds=N`.
4. **Verify** — Global Admin opens **Admin → Backup & restore** or `GET /api/v1/internal/ops/backup-status`. Alert if `alerts` is non-empty.

## Full restore (disaster recovery)

1. Provision an isolated Postgres 16 instance in the same region.
2. Restore the latest base: `wal-g backup-fetch LATEST /var/lib/postgresql/data`.
3. Replay WAL to the target time: `wal-g wal-fetch` / point-in-time recovery per WAL-G docs.
4. Point `DATABASE_URL` at the restored instance only after smoke tests pass.
5. Record the drill: `POST /api/v1/internal/ops/restore-drill` with RPO/RTO achieved minutes and smoke test output.

## Retention

- Daily: 30 days (RDS `backup_retention_period` + S3 lifecycle).
- Weekly/monthly: S3 lifecycle transitions (see `backup.tf`).
