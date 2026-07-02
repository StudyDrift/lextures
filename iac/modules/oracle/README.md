# Oracle Cloud Infrastructure single-VM module

Single Ampere A1 compute instance with attached block storage for small-school deployments (~300 students, ~15 teachers). All services run via `docker-compose.deploy.yml` on the VM:

- PostgreSQL 16 (data on block volume)
- Redis, RabbitMQ, Go API, nginx web

**Default sizing** targets OCI **Always Free**: `VM.Standard.A1.Flex` with 2 OCPUs and 12 GB RAM, 50 GB boot + 100 GB data volume (150 GB total block storage within the 200 GB free quota).

When `deploy_enabled = true` (default in `iac/production/`), cloud-init also writes `/opt/lextures/docker-compose.deploy.yml` and `.env`, pulls container images, and runs `docker compose up -d --wait` on first boot. Set `deploy_server_image` and `deploy_web_image` in `terraform.tfvars` (Arm64 images required). Generated secrets are available via `terraform output -raw deploy_postgres_password` and `deploy_jwt_secret`. Set `deploy_turnstile_secret_key` (or `TF_VAR_deploy_turnstile_secret_key`) to pass Cloudflare Turnstile verification to the API as `TURNSTILE_SECRET_KEY`.

**Important:** Always Free resources must be created in the tenancy **home region**. Build and deploy **linux/arm64** container images (Ampere is Arm). If instance creation fails with "out of host capacity", retry another availability domain or upgrade to Pay As You Go (Always Free resources remain free).

Used by `iac/production/` when `deployment_tier = "small"` and `cloud_provider = "oracle"`.

## Authentication

Configure the OCI provider via `~/.oci/config` (recommended) or set in the root module:

- `tenancy_ocid`, `user_ocid`, `fingerprint`, `private_key_path`, `region`

See [OCI Terraform provider docs](https://docs.oracle.com/en-us/iaas/Content/API/SDKDocs/terraformprovider.htm).
