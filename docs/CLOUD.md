# Lextures Cloud Deployment Guide

This document describes recommended cloud architectures for self-hosting or operating Lextures across the three major cloud providers — **AWS**, **Google Cloud Platform (GCP)**, and **Microsoft Azure**. It covers small, medium, and large deployment tiers, gives ballpark monthly cost estimates, and outlines a scaling scenario targeting **100,000 concurrent users**.

## 1. Architecture inputs

Lextures consists of:

| Layer       | Technology                                                                 | Cloud considerations                                                                     |
| ----------- | -------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| Web (SPA)   | React 19 + Vite static bundle, served by nginx (`clients/web`, `www`)      | Static assets — CDN-friendly, no server runtime needed.                                  |
| API         | Go 1.25 + Chi (`server/`), stateless HTTP, JWT auth, pgx Postgres driver   | Horizontally scalable behind an L7 load balancer. Container-native (`server/Dockerfile`). |
| Database    | PostgreSQL 16                                                              | Single primary with read replicas at scale. Heavy use of relational features.            |
| Object data | User uploads, exports, backups                                             | S3-class object storage.                                                                 |
| AI (opt.)   | OpenRouter API egress                                                      | Outbound HTTPS only; no inbound infra needed.                                            |
| Cache/queue | Optional Redis for sessions, rate limits, background jobs                  | Managed Redis recommended from medium tier upward.                                       |

Deployment tiers used throughout this document:

- **Small** — pilot / single school / < 1,000 MAU, < 100 concurrent.
- **Medium** — district or multi-school / 1k–25k MAU, ~500–2,000 concurrent.
- **Large** — multi-district SaaS / 25k–250k MAU, ~5,000–20,000 concurrent.

All cost estimates are **USD/month, on-demand, single region, list price**. Reserved/committed-use discounts (1–3 yr) typically cut compute and database costs by 30–55%. Egress and OpenRouter usage are excluded unless noted.

---

## 2. AWS

### 2.1 Small (AWS)

| Component       | Service                                                      | Sizing                          |
| --------------- | ------------------------------------------------------------ | ------------------------------- |
| Web (SPA)       | S3 + CloudFront                                              | Standard tier                   |
| API             | ECS Fargate behind ALB                                       | 1 task, 0.5 vCPU / 1 GB         |
| Database        | RDS for PostgreSQL 16, Single-AZ                             | db.t4g.small, 50 GB gp3         |
| Object storage  | S3                                                           | 50 GB                           |
| Secrets / logs  | Secrets Manager, CloudWatch Logs                             | Minimal                         |
| DNS / TLS       | Route 53 + ACM                                               | 1 hosted zone                   |

**Estimated cost:** **~$130–180/mo**

### 2.2 Medium (AWS)

| Component       | Service                                                      | Sizing                                  |
| --------------- | ------------------------------------------------------------ | --------------------------------------- |
| Web (SPA)       | S3 + CloudFront                                              | + WAF                                   |
| API             | ECS Fargate behind ALB, autoscaling 2–6                      | 1 vCPU / 2 GB tasks                     |
| Database        | RDS PostgreSQL 16, **Multi-AZ**, 1 read replica              | db.m7g.large, 200 GB gp3, PITR on       |
| Cache           | ElastiCache Redis (cluster mode off)                         | cache.t4g.small × 2 (replica)           |
| Object storage  | S3 + lifecycle to IA                                         | 500 GB                                  |
| Observability   | CloudWatch + AWS X-Ray, OpenSearch (small)                   |                                         |
| Secrets         | Secrets Manager, KMS                                         |                                         |

**Estimated cost:** **~$900–1,300/mo**

### 2.3 Large (AWS)

| Component       | Service                                                                              | Sizing                                  |
| --------------- | ------------------------------------------------------------------------------------ | --------------------------------------- |
| Web (SPA)       | S3 + CloudFront (multi-region), AWS WAF, Shield Standard                             |                                         |
| API             | EKS (or Fargate) behind ALB, HPA + Cluster Autoscaler, 10–40 pods                    | 2 vCPU / 4 GB per pod                   |
| Database        | Aurora PostgreSQL-Compatible, writer + 2–3 readers, Performance Insights             | db.r7g.2xlarge writer, r7g.xlarge readers |
| Cache           | ElastiCache Redis cluster mode on, 3 shards × 2 replicas                             | cache.r7g.large                         |
| Async jobs      | SQS + a worker service on Fargate/EKS                                                 |                                         |
| Object storage  | S3 + CloudFront origin + Intelligent-Tiering                                         | Multi-TB                                |
| Observability   | CloudWatch, X-Ray, managed Prometheus + Grafana                                       |                                         |
| Network         | Multi-AZ VPC, PrivateLink for RDS, NAT GW pair, Transit Gateway if multi-account     |                                         |

**Estimated cost:** **~$6,500–10,000/mo**

---

## 3. Google Cloud Platform (GCP)

### 3.1 Small (GCP)

| Component       | Service                                                      | Sizing                          |
| --------------- | ------------------------------------------------------------ | ------------------------------- |
| Web (SPA)       | Cloud Storage + Cloud CDN (with HTTPS LB)                    | Standard                        |
| API             | Cloud Run                                                    | 1 vCPU / 512 MB, min 1 instance |
| Database        | Cloud SQL for PostgreSQL 16, zonal                           | db-custom-1-3840, 50 GB SSD     |
| Object storage  | Cloud Storage                                                | 50 GB                           |
| Secrets / logs  | Secret Manager, Cloud Logging                                | Minimal                         |
| DNS / TLS       | Cloud DNS + managed certs                                    |                                 |

**Estimated cost:** **~$110–160/mo**

### 3.2 Medium (GCP)

| Component       | Service                                                                  | Sizing                                  |
| --------------- | ------------------------------------------------------------------------ | --------------------------------------- |
| Web (SPA)       | GCS + Cloud CDN behind external HTTPS LB, Cloud Armor                    |                                         |
| API             | Cloud Run (or GKE Autopilot), min 2 / max 20 instances                   | 1 vCPU / 2 GB                           |
| Database        | Cloud SQL PostgreSQL 16, **HA (regional)**, 1 read replica               | db-custom-2-7680, 200 GB SSD, PITR      |
| Cache           | Memorystore for Redis (Standard tier)                                    | 1–5 GB                                  |
| Object storage  | GCS + lifecycle to Nearline                                              | 500 GB                                  |
| Observability   | Cloud Monitoring, Cloud Trace, Cloud Logging                             |                                         |

**Estimated cost:** **~$800–1,200/mo**

### 3.3 Large (GCP)

| Component       | Service                                                                                | Sizing                                  |
| --------------- | -------------------------------------------------------------------------------------- | --------------------------------------- |
| Web (SPA)       | GCS + Cloud CDN, Cloud Armor (WAF + DDoS), global HTTPS LB                             |                                         |
| API             | GKE Standard regional cluster, HPA + node autoscaling, 10–40 pods                      | 2 vCPU / 4 GB per pod                   |
| Database        | **AlloyDB for PostgreSQL** (or Cloud SQL Enterprise Plus), 1 primary + 2 read pool nodes | 16 vCPU primary, 8 vCPU readers         |
| Cache           | Memorystore for Redis Cluster                                                          | 30–60 GB                                |
| Async jobs      | Pub/Sub + Cloud Run workers (or GKE worker deployment)                                 |                                         |
| Object storage  | GCS Multi-Region + CDN + Autoclass                                                     | Multi-TB                                |
| Observability   | Cloud Ops suite, Managed Service for Prometheus                                        |                                         |
| Network         | VPC with Private Service Connect to AlloyDB, Cloud NAT pair                            |                                         |

**Estimated cost:** **~$6,000–9,500/mo**

---

## 4. Microsoft Azure

### 4.1 Small (Azure)

| Component       | Service                                                      | Sizing                          |
| --------------- | ------------------------------------------------------------ | ------------------------------- |
| Web (SPA)       | Azure Static Web Apps (or Blob Storage + Azure Front Door)   | Standard                        |
| API             | Azure Container Apps                                         | 0.5 vCPU / 1 GB, 1 replica min  |
| Database        | Azure Database for PostgreSQL Flexible Server, Burstable     | B2s, 64 GB                      |
| Object storage  | Blob Storage (Hot)                                           | 50 GB                           |
| Secrets / logs  | Key Vault, Log Analytics                                     |                                 |
| DNS / TLS       | Azure DNS + managed certs                                    |                                 |

**Estimated cost:** **~$140–200/mo**

### 4.2 Medium (Azure)

| Component       | Service                                                              | Sizing                                  |
| --------------- | -------------------------------------------------------------------- | --------------------------------------- |
| Web (SPA)       | Static Web Apps + Front Door Standard (WAF)                          |                                         |
| API             | Container Apps with autoscaling 2–10 replicas                        | 1 vCPU / 2 GB                           |
| Database        | PostgreSQL Flexible Server **Zone-Redundant HA**, 1 read replica     | General Purpose D2ds_v5, 256 GB         |
| Cache           | Azure Cache for Redis Standard                                       | C1                                      |
| Object storage  | Blob + lifecycle to Cool                                             | 500 GB                                  |
| Observability   | Application Insights, Log Analytics                                  |                                         |

**Estimated cost:** **~$1,000–1,400/mo**

### 4.3 Large (Azure)

| Component       | Service                                                                              | Sizing                                  |
| --------------- | ------------------------------------------------------------------------------------ | --------------------------------------- |
| Web (SPA)       | Static Web Apps + Front Door Premium (WAF + DDoS Standard)                           |                                         |
| API             | AKS (or Container Apps Environment), HPA + cluster autoscaler, 10–40 pods            | 2 vCPU / 4 GB per pod                   |
| Database        | PostgreSQL Flexible Server **Business Critical** or Cosmos DB for PostgreSQL cluster | 8–16 vCPU primary + 2 read replicas     |
| Cache           | Azure Cache for Redis Enterprise                                                     | E10                                     |
| Async jobs      | Service Bus + worker container app                                                   |                                         |
| Object storage  | Blob Premium + Azure CDN                                                             | Multi-TB                                |
| Observability   | Azure Monitor, App Insights, Managed Grafana                                         |                                         |
| Network         | Hub-spoke VNet, Private Endpoints for DB/Redis/Blob, NAT GW pair                     |                                         |

**Estimated cost:** **~$7,000–10,500/mo**

---

## 5. Cross-provider cost summary

| Tier   | AWS               | GCP               | Azure              |
| ------ | ----------------- | ----------------- | ------------------ |
| Small  | $130–180          | $110–160          | $140–200           |
| Medium | $900–1,300        | $800–1,200        | $1,000–1,400       |
| Large  | $6,500–10,000     | $6,000–9,500      | $7,000–10,500      |

**Notes:**

- All prices are list-price on-demand. Apply 30–55% off for 1–3 year reservations / Savings Plans / CUDs.
- Egress is the biggest unpredictable line item; budget separately based on expected media delivery.
- OpenRouter / LLM spend is fully variable and excluded.

---

## 6. Scaling to 100,000 concurrent users

"Concurrent users" here means **100k simultaneous active sessions** with typical Lextures behavior: reading course content, taking quizzes (with IRT scoring), submitting assignments. Empirically this maps to roughly:

- **~20,000–35,000 API requests per second** at peak (assuming ~0.25 RPS per user, mix of cached SPA navigation and dynamic API calls).
- **~3,000–5,000 PostgreSQL transactions per second**, read-heavy (~85% reads).
- **~50–150 Gbps of egress** if assets are not aggressively cached at the edge.

### 6.1 Required architectural changes

The "Large" tier above is sized for ~20k concurrent. To reach 100k concurrent comfortably:

1. **Edge & static delivery**
   - All SPA assets must be served from a global CDN (CloudFront / Cloud CDN / Front Door) with long TTLs and immutable hashed asset filenames (already produced by Vite).
   - Enable HTTP/3, Brotli, and stale-while-revalidate.
   - Cache LTI tool launches and public catalog endpoints at the edge with short TTLs where safe.

2. **API layer**
   - Run **80–200 API pods** (2 vCPU / 4 GB) across at least 3 AZ/zones.
   - Use HPA based on **RPS or p95 latency**, not just CPU.
   - Move the API to **Kubernetes** (EKS / GKE / AKS) for finer-grained scheduling, PodDisruptionBudgets, and topology spread; Cloud Run / Container Apps work but cold-start and per-instance concurrency limits become harder to tune at this scale.
   - Add a **regional API gateway / WAF** with rate limiting per JWT subject and per IP.

3. **Database**
   - The single-writer Postgres becomes the hardest bottleneck. Options:
     - **AWS**: Aurora PostgreSQL with **6–15 read replicas** (or **Aurora Limitless** for write sharding), writer on `db.r7g.8xlarge`+.
     - **GCP**: **AlloyDB** with a read pool of 4–8 nodes, columnar engine on for analytics.
     - **Azure**: **Cosmos DB for PostgreSQL** (Citus) with **distributed tables** for high-volume tables (`quiz_responses`, `attendance_records`, `grade_entries`), or PostgreSQL Flexible Server Business Critical + read replicas if write volume permits.
   - Application changes required:
     - **Read/write splitting** in the Go API (`server/internal/db`): route safe SELECTs to a replica pool via a separate `*pgxpool.Pool`.
     - **Connection pooling** via **PgBouncer** (transaction mode) or **RDS Proxy / AlloyDB connection pooling**; otherwise 100+ pods × pool size will exhaust Postgres `max_connections`.
     - Add **partitioning** on the largest time-series tables (responses, events, audit logs) by month.

4. **Cache & sessions**
   - Promote Redis to a **clustered, multi-shard deployment** (≥ 6 shards × 2 replicas, 100+ GB RAM total).
   - Cache:
     - JWT public keys / introspection results.
     - Course structure trees, gradebook headers, standards taxonomies.
     - IRT item parameters (read-mostly, frequently joined).
   - Add a per-request **cache-aside** layer for hot quiz items.

5. **Asynchronous work**
   - Move heavy operations off the request path onto a queue (SQS / Pub/Sub / Service Bus):
     - AI-assisted quiz generation and misconception detection (OpenRouter calls).
     - Standards rollups, gradebook recalculation, report card generation, SCIM sync.
     - Email / push notification fan-out.
   - Run a dedicated worker deployment with its own HPA.

6. **Object storage & media**
   - Serve all user-uploaded content through the CDN with signed URLs.
   - Enable multipart uploads directly from the browser to object storage (presigned PUTs) instead of streaming through the API.

7. **Observability & SRE**
   - SLOs: API p95 < 300 ms, availability ≥ 99.9%.
   - **Distributed tracing** (OpenTelemetry → X-Ray / Cloud Trace / App Insights) is no longer optional.
   - Synthetic probes per region, real-user monitoring on the SPA.
   - **Load testing** with k6 / Locust at 1.5× peak before each major release.
   - Incident runbooks (`docs/runbooks/`) extended for: replica lag, Redis failover, hot-shard mitigation, edge cache poisoning.

8. **Multi-region (optional, > 100k or strict RPO/RTO)**
   - Active/passive: async DB replication + DNS failover (Route 53 / Cloud DNS / Traffic Manager).
   - Active/active is only justified for global latency requirements — it forces conflict resolution work on writable tables and significantly raises cost.

### 6.2 Indicative cost at 100k concurrent

| Provider | Compute (API + workers) | Database                  | Cache / queue | CDN + egress | **Total / mo**     |
| -------- | ----------------------- | ------------------------- | ------------- | ------------ | ------------------ |
| AWS      | ~$18k (EKS, 150 pods)   | ~$22k (Aurora + replicas) | ~$4k          | ~$15k        | **~$60k–80k**      |
| GCP      | ~$16k (GKE, 150 pods)   | ~$20k (AlloyDB pool)      | ~$3.5k        | ~$13k        | **~$55k–75k**      |
| Azure    | ~$19k (AKS, 150 pods)   | ~$24k (Cosmos PG / BC)    | ~$4.5k        | ~$15k        | **~$65k–85k**      |

With 3-year reserved capacity and committed-use discounts, expect **30–45% reduction**. Egress is the most volatile component — aggressive CDN caching can cut it by half or more.

### 6.3 Application-side prerequisites (independent of cloud)

These are the changes to the codebase that must precede a 100k-user rollout, regardless of provider:

- Read/write Postgres pool split in `server/internal/db`.
- PgBouncer (transaction pooling) in front of every Postgres endpoint.
- Time-based partitioning on high-volume tables; verify all queries use the partition key.
- Redis-backed caching layer with explicit invalidation hooks on writes.
- Background job runner with idempotent handlers (e.g. for OpenRouter retries).
- Per-tenant rate limits and quotas, surfaced in the API and gateway.
- Structured logs with tenant + request IDs; tracing spans on every external call (DB, Redis, OpenRouter, object storage).
- Load-test harness checked into `e2e/` or `scripts/` that can hit a staging environment at ≥ 50k virtual users.

---

## 7. Choosing between providers

| Criterion                          | AWS                            | GCP                              | Azure                                       |
| ---------------------------------- | ------------------------------ | -------------------------------- | ------------------------------------------- |
| Best managed Postgres for scale    | Aurora / Aurora Limitless      | **AlloyDB** (strongest read-pool ergonomics) | Cosmos DB for PostgreSQL (Citus sharding)   |
| Easiest small-tier path            | ECS Fargate + RDS              | **Cloud Run + Cloud SQL**        | Container Apps + PG Flexible Server         |
| Education-sector procurement       | Strong (GovCloud, FedRAMP)     | Good (Workspace tie-in)          | **Strongest** (EDU agreements, EntraID/SSO) |
| Identity tie-in for K-12 districts | Cognito / IAM Identity Center  | Identity Platform                | **Entra ID / SCIM**, often already in use   |
| CDN + edge                         | CloudFront + Lambda@Edge       | Cloud CDN + Cloud Armor          | Front Door Premium                          |
| Typical lowest list price          | —                              | **Slightly cheapest** at scale   | —                                           |

**Recommendation:**

- Pilot / small district → **GCP (Cloud Run + Cloud SQL)** for the lowest operational overhead, or **AWS (ECS Fargate + RDS)** if the customer is already on AWS.
- Mid-market SaaS → **AWS** for breadth of managed services and education-vertical familiarity.
- K-12 / higher-ed already standardized on Microsoft 365 → **Azure**, primarily for Entra ID / SCIM alignment.
- Hyper-scale (> 50k concurrent) → **GCP AlloyDB** or **AWS Aurora** are the two strongest data-tier options; pick based on existing platform expertise.

---

## 8. Kubernetes-native deployments

This section describes what Lextures looks like if you commit to **Kubernetes as the primary runtime** on each provider — instead of the serverless-leaning options (Cloud Run, Container Apps, ECS Fargate) used in §2–4. Kubernetes is the right choice when you want a single deployment model across providers, you already operate clusters, or you expect to grow into the Large / 100k tier where K8s pays off.

### 8.1 What stays the same regardless of provider

- **Workloads**
  - `api` Deployment (Go), 2 vCPU / 4 GB requests, HPA on RPS or p95 latency.
  - `worker` Deployment for async jobs (AI generation, gradebook rollups, notifications).
  - `web` is **not** in-cluster — the SPA ships to object storage + CDN. Running nginx pods is wasteful at scale.
  - `migrate` Job that runs `server/migrations` on deploy; gated by a Helm/Argo hook.
- **Cluster add-ons (same on every provider)**
  - **Ingress**: NGINX Ingress or the provider's gateway controller.
  - **TLS**: cert-manager + Let's Encrypt (or provider-managed certs).
  - **Autoscaling**: HPA (workload) + Cluster Autoscaler / Karpenter-equivalent (nodes) + KEDA for queue-driven worker scaling.
  - **Observability**: OpenTelemetry Collector → provider backend; Prometheus + Grafana (managed where available); Loki or provider logs.
  - **Policy / supply chain**: OPA Gatekeeper or Kyverno, image signing (cosign), Trivy scanning in CI.
  - **Secrets**: External Secrets Operator pulling from the provider's secret manager.
  - **GitOps**: Argo CD or Flux, one app per environment.
- **What stays outside the cluster**
  - **PostgreSQL** — always use the managed service (RDS / Cloud SQL / AlloyDB / Azure PG). Running Postgres in-cluster is not recommended for Lextures' workload.
  - **Redis** — managed (ElastiCache / Memorystore / Azure Cache) unless cost-constrained at the small tier.
  - **Object storage** and **CDN** — managed.

### 8.2 AWS — EKS

| Concern              | Choice                                                                                       |
| -------------------- | -------------------------------------------------------------------------------------------- |
| Cluster              | **EKS** (control plane $73/mo) in a private VPC, 3 AZs                                       |
| Nodes                | **Karpenter** with mixed Graviton (`m7g`, `c7g`) on-demand + spot for workers                |
| Ingress              | AWS Load Balancer Controller → ALB (or NLB for gRPC)                                         |
| DB / cache           | RDS / Aurora PostgreSQL + ElastiCache Redis, accessed via VPC endpoints                      |
| Identity for pods    | **IRSA** (IAM Roles for Service Accounts) — no static creds                                  |
| Storage              | EBS CSI (gp3) for any stateful add-ons; EFS CSI only if shared RW needed                     |
| Secrets              | AWS Secrets Manager via External Secrets Operator                                            |
| Observability        | CloudWatch Container Insights, AMP (Prometheus), AMG (Grafana), ADOT collector               |
| Image registry       | ECR with image scanning + lifecycle policies                                                 |
| Network policy       | VPC CNI + Network Policies (or Cilium for L7)                                                |

**Indicative monthly costs**

| Tier   | EKS + nodes                      | RDS / Aurora     | Redis / LB / misc | **Total**        |
| ------ | -------------------------------- | ---------------- | ----------------- | ---------------- |
| Small  | $73 CP + ~$120 (2× t4g.medium)   | ~$80             | ~$60              | **~$330–400**    |
| Medium | $73 + ~$700 (6–10 nodes)         | ~$500            | ~$250             | **~$1.5k–1.9k**  |
| Large  | $73 + ~$5k (Karpenter pool)      | ~$3.5k           | ~$1.5k            | **~$10k–13k**    |

Small-tier EKS is meaningfully more expensive than ECS Fargate ($330+ vs ~$150) because of the control-plane fee and minimum node count. EKS only "wins" from medium upward.

### 8.3 GCP — GKE

| Concern              | Choice                                                                                       |
| -------------------- | -------------------------------------------------------------------------------------------- |
| Cluster              | **GKE Autopilot** for small/medium (no node mgmt), **GKE Standard regional** for large       |
| Nodes (Standard)     | `e2`/`n2d` for general, `t2d` (AMD) or `c3` for API; node auto-provisioning on               |
| Ingress              | GKE Gateway API → Global External HTTPS LB, Cloud Armor (WAF)                                |
| DB / cache           | Cloud SQL or **AlloyDB** + Memorystore Redis, via Private Service Connect                    |
| Identity for pods    | **Workload Identity** federating KSA → GSA                                                   |
| Storage              | PD-Balanced / PD-SSD CSI; Filestore if shared RW needed                                      |
| Secrets              | Secret Manager via External Secrets Operator (or GCP CSI Secret Store driver)                |
| Observability        | **Managed Service for Prometheus** + Cloud Monitoring + Cloud Trace; GMP is essentially free for moderate volume |
| Image registry       | Artifact Registry with vulnerability scanning                                                |
| Network policy       | Dataplane V2 (Cilium-based) — enable on cluster creation                                     |

**Indicative monthly costs**

| Tier   | GKE + nodes                              | DB (Cloud SQL / AlloyDB) | Redis / LB / misc | **Total**        |
| ------ | ---------------------------------------- | ------------------------ | ----------------- | ---------------- |
| Small  | ~$75 (Autopilot, low usage)              | ~$70                     | ~$50              | **~$220–280**    |
| Medium | ~$600 (Autopilot or 3 e2-standard-4)     | ~$450                    | ~$220             | **~$1.3k–1.7k**  |
| Large  | ~$4.5k (Standard regional + autoprov.)   | ~$3.2k (AlloyDB)         | ~$1.3k            | **~$9k–12k**     |

GKE Autopilot is usually the **cheapest Kubernetes path** at small/medium tier because there is no control-plane fee for the first zonal cluster ($0.10/hr per cluster after the free one) and you only pay for pod resources requested. AlloyDB on GKE Standard is the strongest large-tier story across all three providers.

### 8.4 Azure — AKS

| Concern              | Choice                                                                                       |
| -------------------- | -------------------------------------------------------------------------------------------- |
| Cluster              | **AKS** (free tier control plane for non-prod; **Standard tier $73/mo** for prod SLA)        |
| Nodes                | `Dasv5` / `Dadsv5` (AMD) general; spot node pool for workers                                 |
| Ingress              | **Application Gateway for Containers** (AGC) or AKS-managed NGINX; Front Door in front       |
| DB / cache           | PostgreSQL Flexible Server (or Cosmos DB for PostgreSQL) + Azure Cache for Redis, Private Endpoints |
| Identity for pods    | **Workload Identity** (federated) → Entra ID; replaces AAD Pod Identity                      |
| Storage              | Azure Disk CSI (Premium SSD v2); Azure Files for shared RW                                   |
| Secrets              | Key Vault via Secrets Store CSI driver or External Secrets Operator                          |
| Observability        | **Azure Monitor managed Prometheus** + Container Insights + Managed Grafana                  |
| Image registry       | Azure Container Registry (Premium for geo-replication) with Defender for Containers          |
| Network policy       | Azure CNI Overlay + Cilium (preview/GA varies by region) for L7 policy                       |

**Indicative monthly costs**

| Tier   | AKS + nodes                            | PostgreSQL FS      | Redis / LB / misc | **Total**         |
| ------ | -------------------------------------- | ------------------ | ----------------- | ----------------- |
| Small  | ~$130 (free CP + 2 D2s_v5)             | ~$80               | ~$70              | **~$320–400**     |
| Medium | $73 + ~$700                            | ~$550              | ~$250             | **~$1.6k–2.0k**   |
| Large  | $73 + ~$5.2k                           | ~$4k (BC or Cosmos)| ~$1.6k            | **~$11k–14k**     |

Azure's AKS pricing is broadly similar to EKS. Choose Azure when you specifically want Entra ID + SCIM tie-in for K-12 districts already on Microsoft 365.

### 8.5 100k concurrent on Kubernetes — what changes vs §6

Most of §6's recommendations are inherently Kubernetes-shaped. The K8s-specific adjustments are:

1. **Cluster topology**
   - Run a **regional** cluster (3 zones). 3+ node pools: `api` (compute-optimized), `worker` (general, spot-friendly), `system` (ingress, observability, no spot).
   - Enable **PodTopologySpreadConstraints** across zones for `api` and `worker` Deployments.
   - PDBs with `maxUnavailable: 10%` on `api`.
2. **Autoscaling**
   - HPA on **custom metrics** — RPS via Prometheus Adapter, or p95 latency. CPU alone underscales Go services.
   - **KEDA** for `worker`, scaling on SQS / Pub/Sub / Service Bus queue depth.
   - Cluster autoscaler / Karpenter / GKE node auto-provisioning sized to add nodes in < 60s — use pre-warmed capacity headroom (e.g. 10–20% overprovision via low-priority pause pods).
3. **Connection management**
   - Run **PgBouncer as a Deployment** (transaction pooling, ~10 replicas, behind a Service) between `api`/`worker` pods and the managed Postgres endpoint. Without this, 150+ pods × pool size will exhaust DB connections.
   - Alternatively use the provider's pooler: RDS Proxy (AWS), AlloyDB built-in pooling (GCP), PgBouncer integrated in Azure PG Flexible Server.
4. **Traffic management**
   - Gateway API > Ingress for north-south at this scale.
   - Add a **service mesh** (Istio Ambient, Linkerd, or Cilium Service Mesh) only if you need mTLS between services or per-tenant traffic policy — otherwise it is overhead.
5. **Resource governance**
   - Set **requests = limits** for `api` to get Guaranteed QoS and predictable latency.
   - Namespace-level ResourceQuotas + LimitRanges to prevent noisy-neighbor pods from starving the cluster.
6. **Rollouts**
   - **Argo Rollouts** (or Flagger) for progressive delivery — canary on `api`, blue/green on schema-coupled releases that need to coordinate with `migrate` Jobs.
7. **Disaster recovery**
   - **Velero** for cluster state backup (manifests, PVs).
   - Managed DB handles its own PITR; rehearse restore quarterly.

### 8.6 Kubernetes vs serverless — when to pick which

| Situation                                                                 | Recommendation                                                  |
| ------------------------------------------------------------------------- | --------------------------------------------------------------- |
| Single small/medium tenant, no platform team                              | **Serverless** (Cloud Run / Container Apps / Fargate). Cheaper, less ops. |
| Multi-tenant SaaS, multiple environments, need per-tenant isolation knobs | **Kubernetes**. Namespaces + network policy + quotas pay off.   |
| Already running other workloads on K8s                                    | **Kubernetes**. Single platform.                                |
| Strict cold-start / latency floor                                         | **Kubernetes** (always-on pods).                                |
| Need portability across AWS / GCP / Azure                                 | **Kubernetes** with the per-provider mappings above.            |
| Bursty, idle most of the day                                              | **Serverless** — Lextures scales to near-zero between class periods. |
| Targeting 100k concurrent                                                 | **Kubernetes** — at this scale K8s controls and observability tooling are unmatched. |

---

## 9. References

- Architecture overview: [docs/ARCH.md](ARCH.md)
- Infrastructure-as-code skeletons: [iac/modules/aws](../iac/modules/aws), [iac/modules/gcp](../iac/modules/gcp), [iac/modules/azure](../iac/modules/azure)
- Production Terraform composition: [iac/production](../iac/production)
- Self-hosting guide: [www/src/docs/self-hosting.md](../www/src/docs/self-hosting.md)
- Runbooks: [docs/runbooks](runbooks)
- Kubernetes-native deployments: see §8 above
