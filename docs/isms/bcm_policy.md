# Business continuity management (BCM) policy

**Control mapping:** ISO/IEC 27001:2022 A.5.29, A.5.30; SOC 2 Availability TSC A1.2; NIST SP 800-53 CP-9, CP-10.

## Objectives

Lextures maintains backup and recovery capabilities so that loss of Postgres or object storage does not result in unrecoverable loss of student or grade data.

## Recovery targets

- **Postgres:** RPO ≤ 1 hour, RTO ≤ 4 hours (see plan 10.15).
- **Course files (object storage):** RPO ≤ 24 hours, RTO ≤ 4 hours.

## Controls

1. Automated backups codified in `iac/production/backup.tf` and `iac/modules/aws/backup.tf`.
2. Encrypted backup storage with Object Lock (30-day WORM) on production buckets.
3. Quarterly restore drills logged in `compliance.restore_drills`.
4. Ops dashboard and `GET /api/v1/internal/ops/backup-status` for continuous monitoring.

## Review

This policy is reviewed annually and after any failed restore drill or production data-loss incident.
