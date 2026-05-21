# GCP module (planned)

Future production stack:

- **GKE** — Kubernetes
- **Cloud SQL for PostgreSQL**
- **Memorystore for Redis**
- **Cloud Storage** — course files
- **Secret Manager** — credentials

Root module will call `../modules/gcp` when `cloud_provider = "gcp"`. Until implemented, use `cloud_provider = "aws"`.
