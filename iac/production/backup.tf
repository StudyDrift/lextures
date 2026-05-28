# Plan 10.15 — Backup / restore infrastructure (production root).
# WAL-G archives Postgres WAL + daily base backups; object storage snapshots land in the backup bucket.

output "backup_bucket_name" {
  description = "Encrypted S3 bucket for Postgres WAL-G and object-storage backups."
  value       = module.aws[0].backup_bucket_name
}

output "backup_writer_policy_arn" {
  description = "IAM policy ARN for the backup cron / WAL-G role (no application runtime access)."
  value       = module.aws[0].backup_writer_policy_arn
}
