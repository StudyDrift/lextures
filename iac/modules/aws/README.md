# AWS production module

Composable AWS infrastructure for Lextures production:

- `vpc.tf` — VPC, public/private subnets, NAT
- `eks.tf` — EKS cluster and managed node group
- `rds.tf` — PostgreSQL 16 (private)
- `elasticache.tf` — Redis 7 replication group
- `s3.tf` — encrypted course-files bucket
- `secrets.tf` — Secrets Manager for `DATABASE_URL` and `REDIS_URL`
- `iam_s3.tf` — IRSA role for namespace `lextures`, service account `api`

Invoked from `iac/production/` when `cloud_provider = "aws"`.
