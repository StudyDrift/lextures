# Production infrastructure (Terraform)

Multi-cloud production IaC for Lextures. Select the target cloud with `cloud_provider`; **AWS is fully implemented** (EKS, RDS PostgreSQL, ElastiCache Redis, S3). Azure and GCP directories under `iac/modules/` are reserved for future work.

## Layout

```
iac/
├── modules/
│   ├── aws/          # VPC, EKS, RDS, Redis, S3, Secrets Manager, IRSA
│   ├── azure/        # planned
│   └── gcp/          # planned
└── production/       # root module (this directory)
```

## AWS resources

| Component | Service | Notes |
|-----------|---------|--------|
| Networking | VPC (3 AZ), NAT | Private subnets for workloads; public for load balancers |
| Compute | Amazon EKS | Managed node group; deploy API/web via Helm/Kubernetes |
| Database | RDS PostgreSQL 16 | Private; credentials in Secrets Manager |
| Cache | ElastiCache Redis 7 | TLS + auth token; URL in Secrets Manager |
| Object storage | S3 | Course files (`COURSE_FILES_ROOT`); IRSA role for `lextures:api` |
| Secrets | Secrets Manager | `database-url`, `redis-url` ARNs exported (not values) |

Sizing defaults differ for `environment = staging` vs `production` (instance classes, Multi-AZ, backup retention, Redis replica count).

## Prerequisites

- Terraform >= 1.5
- AWS credentials with permissions to create VPC, EKS, RDS, ElastiCache, S3, IAM, Secrets Manager
- Optional: remote backend (see `backend.tf.example`) or Terraform Cloud workspace

## Quick start (AWS)

```bash
cd iac/production
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars

terraform init
terraform workspace new staging   # or: production
terraform plan
terraform apply
```

Configure kubectl:

```bash
terraform output -raw kubectl_config_command | bash
```

Wire the API deployment (Helm/manifests) to:

- `database_url_secret_arn` — mount or sync as `DATABASE_URL`
- `redis_url_secret_arn` — future horizontal scaling / cache (17.2)
- `course_files_bucket_name` + `course_files_irsa_role_arn` — annotate ServiceAccount `lextures:api`

## Variables

| Variable | Description |
|----------|-------------|
| `cloud_provider` | `aws` (implemented), `azure`, `gcp` (planned) |
| `environment` | `staging` or `production` |
| `aws_region` | AWS region |

See `variables.tf` for EKS/RDS/Redis sizing overrides.

## Workspaces

Use Terraform workspaces (or separate `.tfvars`) for `staging` and `production`. The `environment` variable should match the workspace intent so resource names and sizing stay consistent.

## CI

Add a `terraform fmt -check`, `validate`, and `plan` job for PRs touching `iac/production/` or `iac/modules/` (see `iac/demo` and `.github/workflows/deploy-demo.yml` for HCP Terraform patterns).

## Next steps (not in Terraform)

- Kubernetes manifests / Helm chart for Go API and React web
- AWS Load Balancer Controller + Ingress for public traffic
- External Secrets Operator to inject Secrets Manager values into pods
- Azure (AKS + flexible PostgreSQL + Azure Cache) and GCP (GKE + Cloud SQL + Memorystore) modules under `iac/modules/`
