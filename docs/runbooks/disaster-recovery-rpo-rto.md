# Disaster recovery playbook (RPO / RTO)

| Tier | RPO target | RTO target | Mechanism |
|------|------------|------------|-----------|
| Postgres | ≤ 1 hour | ≤ 4 hours | WAL-G continuous WAL + daily base backup |
| Object storage (course files) | ≤ 24 hours | ≤ 4 hours | S3 versioning + daily snapshot to backup bucket |

## Roles

- **Incident commander** — Declares DR, coordinates comms.
- **Ops engineer** — Executes restore runbook (`database-backup-restore.md`).
- **Compliance** — Records restore drill results in Lextures (`restore_drills` table).

## Quarterly restore drill (FR-7)

1. Spin up an isolated environment (staging VPC or ephemeral RDS).
2. Restore from the most recent production base backup + WAL to a fixed timestamp.
3. Run smoke tests: auth login, grade read, quiz attempt list, course module fetch.
4. Submit results via **Admin → Backup & restore** or `POST /api/v1/internal/ops/restore-drill`.
5. File evidence in the SOC 2 / ISO BCM package.

## Escalation

If `backup-status` reports WAL lag > 15 minutes or last success > 25 hours, page on-call and inspect WAL-G / RDS storage IOPS.
