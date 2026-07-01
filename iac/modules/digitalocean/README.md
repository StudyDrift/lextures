# DigitalOcean single-VM module

Single droplet with attached block storage for small-school deployments (~300 students, ~15 teachers). All services run via `docker-compose.deploy.yml` on the VM:

- PostgreSQL 16 (data on block volume)
- Redis, RabbitMQ, Go API, nginx web

**Estimated cost:** ~$57/mo (s-4vcpu-8gb + 50 GB volume + reserved IP). Optional weekly droplet backups add ~20% of droplet cost.

Used by `iac/production/` when `deployment_tier = "small"` and `cloud_provider = "digitalocean"`.
