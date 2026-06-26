# AWS production module

Composable AWS infrastructure for Lextures production:

- `vpc.tf` — VPC, public/private subnets, NAT
- `eks.tf` — EKS cluster and managed node group
- `rds.tf` — PostgreSQL 16 (private)
- `elasticache.tf` — Redis 7 replication group
- `s3.tf` — encrypted course-files bucket
- `secrets.tf` — Secrets Manager for `DATABASE_URL`, `REDIS_URL`, and `RABBITMQ_URL`
- `iam_s3.tf` — IRSA role for namespace `lextures`, service account `api`
- `bastion.tf` — optional SSM-managed jump host for emergency Postgres access

Invoked from `iac/production/` when `cloud_provider = "aws"`.
