# Azure module (planned)

Future production stack:

- **AKS** — Kubernetes for API and web workloads
- **Azure Database for PostgreSQL** — flexible server, private endpoint
- **Azure Cache for Redis**
- **Azure Blob Storage** — course files
- **Key Vault** — connection strings (no secrets in Terraform state)

Root module will call `../modules/azure` when `cloud_provider = "azure"`. Until implemented, `terraform apply` with `cloud_provider = "azure"` fails the `cloud_provider_implemented` check in `iac/production/main.tf`.
