# AWS self-host stack (`iac/self-aws`)

Managed AWS infrastructure for Lextures that **does not** co-locate Postgres, Redis, and RabbitMQ in Docker on a single EC2/VM.

This directory is **independent of Oracle Cloud** (`iac/self` + `iac/modules/oracle`). Applying here does not touch OCI resources.

## Architecture

```mermaid
flowchart TB
  Users --> CF[CloudFront]
  CF -->|default SPA| ALB[Application Load Balancer]
  CF -->|/api /health /tus| ALB
  ALB -->|default| Web[ECS Fargate — nginx web image]
  ALB -->|/api /health /tus| API[ECS Fargate — Go API]

  subgraph private [Private subnets]
    RDS[(RDS PostgreSQL 16)]
    Redis[(ElastiCache Redis 7)]
  end

  API --> RDS
  API --> Redis
  API --> SQS[SQS standard queues + DLQs]
  API --> S3Files[(S3 course files)]
  SM[Secrets Manager] -.-> API
```

| Concern | AWS service | Cost notes |
|---------|-------------|------------|
| **Web (SPA)** | **ECS Fargate** (`web_image`) + ALB, or **S3** when `web_image` is empty | Prefer the GHCR web image from `publish-images` |
| **API** | **ECS Fargate** + ALB | CloudFront proxies `/api/*` (and ALB path-routes when web is on Fargate) |
| CDN | **CloudFront** | HTTPS + optional custom domain; default origin is ALB when `web_image` is set |
| Database | **RDS PostgreSQL 16** (`db.t4g.micro` default) | Free-tier eligible size; single-AZ |
| Cache | **ElastiCache Redis 7** (`cache.t3.micro` default) | Free-tier eligible size; single node; TLS + auth |
| Queues | **SQS** (4 queues + DLQs) | Always Free: 1M requests/month |
| Files | **S3** (course files) | SSE-S3; IAM task role |
| Secrets | **Secrets Manager** | `DATABASE_URL`, `REDIS_URL`, JWT, SQS URLs, storage; optional registry pull auth |

Default networking places Fargate tasks in **public subnets with public IPs** so a NAT gateway is not required (~$32/mo savings). RDS and Redis stay private. Set `enable_nat_gateway = true` for private-subnet tasks.

### Web image vs static S3

- **Preferred:** set `web_image` (e.g. `ghcr.io/<org>/<repo>/web:latest` from `.github/workflows/publish-images.yml`). Terraform runs nginx on Fargate; ALB path-routes API vs SPA. CloudFront’s default origin is the ALB.
- **Legacy / offline build:** leave `web_image` empty and run `scripts/deploy-web.sh` to build Vite and sync `clients/web/dist` to S3. CloudFront serves S3 and only proxies API paths to the ALB.

Build the SPA (image or static) with **empty `VITE_API_URL`** so the browser uses same-origin (`window.location.origin`). The published web image already does this.

ALB path patterns for the API: `/api/*`, `/health`, `/health/*`, `/tus/*`. The nginx image’s built-in reverse proxy to Docker hostname `server` is **not** used on Fargate; the ALB owns that routing.

### Deploying new versions

**CI (recommended):** [`.github/workflows/deploy-self-aws.yml`](../../.github/workflows/deploy-self-aws.yml) runs after [Publish container images](../../.github/workflows/publish-images.yml) succeeds on `main` (same trigger as Deploy Self). It applies this module with immutable `server_image` / `web_image` tags `ghcr.io/<org>/<repo>/{server,web}:<git-sha>`, which updates ECS task definitions and rolls the services. Manual runs: **Actions → Deploy Self AWS → Run workflow**.

Repository secrets: `TF_TOKEN`, `TF_CLOUD_ORGANIZATION` (shared with Deploy Self). AWS credentials, optional GHCR pull credentials (`registry_username` / `registry_password`), and optional `bootstrap_admin_email` belong in the HCP workspace `lextures-self-aws-production` — use a long-lived PAT for private packages, not `GITHUB_TOKEN` (it expires and would be stored in Secrets Manager).

**Local / force redeploy** after images are already in the registry:

```bash
# Roll the nginx SPA (force-new-deployment pulls the tag in web_image)
./iac/self-aws/scripts/deploy-web.sh

# Roll the Go API
./iac/self-aws/scripts/deploy-api.sh
```

To pin a new immutable tag manually, change `server_image` / `web_image` in `terraform.tfvars` (or HCP vars) and `terraform apply` (creates new task definitions). With `:latest`, the force-deploy scripts above re-pull after a registry push.

If `web_image` is empty, `deploy-web.sh` falls back to build + S3 sync + CloudFront invalidation.

## Prerequisites

- Terraform >= 1.5
- AWS credentials with permission to create VPC, RDS, ElastiCache, SQS, S3, CloudFront, ECS, ALB, IAM, Secrets Manager, CloudWatch Logs
- Container images for the **API** and (recommended) **web** when `enable_ecs = true`
- For **private** GHCR packages: `registry_username` + `registry_password` (PAT with `read:packages`)
- Node.js + npm only if using the static S3 path (`web_image` empty)

## Quick start

```bash
cd iac/self-aws
cp terraform.tfvars.example terraform.tfvars
# Edit region, server_image, web_image, optional public_web_origin / custom domain

terraform init
terraform plan
terraform apply

# Roll tasks after images are in the registry (or after changing :latest contents)
./scripts/deploy-web.sh
./scripts/deploy-api.sh
```

Data plane only (no ALB/ECS/CloudFront API proxy yet):

```hcl
enable_ecs         = false
enable_static_site = true   # can still host a built SPA on S3
```

Then enable the app:

```hcl
enable_ecs     = true
server_image   = "ghcr.io/YOUR_ORG/lextures/server:latest"
web_image      = "ghcr.io/YOUR_ORG/lextures/web:latest"
# registry_username / registry_password if GHCR packages are private
```

Custom domain (optional). Without `web_domain_names`, CloudFront serves HTTPS on `*.cloudfront.net` automatically.

**Terraform-managed cert (recommended):** set only the domain list — no certificate UUID needed.

```hcl
web_domain_names  = ["beta.example.com"]
public_web_origin = "https://beta.example.com"
# web_acm_certificate_arn left empty → ACM cert created in us-east-1
```

1. Set `web_domain_names` (and `public_web_origin`) in HCP or tfvars.
2. Preferred first apply (creates the cert without waiting forever on DNS):
   ```bash
   cd iac/self-aws
   terraform apply -target=aws_acm_certificate.web
   terraform output acm_dns_validation_records
   ```
3. Create each record as a **CNAME** in Cloudflare (**DNS only** / grey cloud).
4. Full apply (validates cert, attaches it to CloudFront):
   ```bash
   terraform apply
   ```
5. Point the site CNAME: `beta` → CloudFront domain (`terraform output cloudfront_domain_name`), also **DNS only**.

A full apply without the validation CNAMEs in place waits up to **45 minutes** for ACM issuance and then fails if DNS is still missing.

**Existing cert:** set `web_acm_certificate_arn` to a real us-east-1 ACM ARN instead of leaving it empty.

## Application configuration

Secrets Manager secret `${project}-${environment}/app` is a JSON object. ECS injects keys as environment variables:

| Key | Purpose |
|-----|---------|
| `DATABASE_URL` | RDS (`sslmode=require`) |
| `REDIS_URL` | ElastiCache (`rediss://` TLS + auth) |
| `JWT_SECRET` | Auth signing |
| `QUEUE_BACKEND` | `sqs` |
| `SQS_*_URL` | Per-queue SQS URLs |
| `STORAGE_BACKEND` | `s3` |
| `STORAGE_BUCKET` / `STORAGE_REGION` | Course files |

`PUBLIC_WEB_ORIGIN` on the API task defaults to the CloudFront HTTPS URL (or `public_web_origin` when set).

`BOOTSTRAP_ADMIN_EMAIL` comes from Terraform variable `bootstrap_admin_email` (HCP workspace or `terraform.tfvars`). When non-empty, the **first** password signup whose email matches (trimmed, lowercased) gets Global Admin if no human users exist yet. Leave empty and use `go run ./cmd/bootstrap-admin -email=…` against RDS to promote an account after the fact.

Local / Oracle dev remains unchanged (`RABBITMQ_URL`, local Vite, etc.).

## App code changes (SQS)

- Shared transport: `server/internal/mq` (RabbitMQ **or** SQS by URL scheme)
- Queue packages use that transport; config via `QUEUE_BACKEND` + `SQS_*_URL`
- Postgres-backed job queue (ADR 0001) is unchanged

## Estimated monthly cost (ballpark, us-east-1)

| Resource | ~USD/mo |
|----------|---------|
| RDS `db.t4g.micro` single-AZ 20 GB | ~$12–15 (often $0 in free tier year 1) |
| ElastiCache `cache.t3.micro` | ~$12 (often $0 in free tier year 1) |
| SQS | ~$0 at modest volume |
| S3 (optional web bucket + course files) | storage + requests (often <$2 early) |
| CloudFront | free tier 1 TB / 10M requests (year 1), then usage |
| ALB | ~$16+ |
| Fargate API 0.5 vCPU / 1 GB × 1 | ~$15–25 |
| Fargate web 0.25 vCPU / 0.5 GB × 1 | ~$8–12 |
| NAT (optional) | ~$32 |
| **Typical lean total (no NAT, web + API on Fargate)** | **~$65–85** (lower during free tier) |

## Migration notes (Oracle → AWS)

1. Apply this stack (data plane first is fine).
2. Dump OCI Postgres → restore into RDS.
3. Sync course files into the course-files S3 bucket; set `web_image` / `server_image` and apply (or use static `deploy-web.sh` if not using the web image).
4. Point DNS at CloudFront (`cloudfront_domain_name` or custom alias); set `public_web_origin` if needed.
5. Leave the Oracle stack running until cutover validation is complete — **no resources here modify OCI**.

## Outputs

```bash
terraform output cloudfront_domain_name
terraform output web_bucket
terraform output alb_dns_name
terraform output ecs_cluster_name
terraform output ecs_web_service_name
terraform output ecs_api_service_name
terraform output -raw database_url   # sensitive
terraform output sqs_queue_urls
terraform output course_files_bucket
terraform output app_secret_arn
```
