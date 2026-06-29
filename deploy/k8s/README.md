# Kubernetes deploy manifests for blue/green and canary releases (plan 17.9).

## Layout

| File | Purpose |
|------|---------|
| `lextures-api-rollout.yaml` | API Deployment with zero-downtime `RollingUpdate` (`maxUnavailable: 0`) and separate blue/green Services |
| `ingress-canary.yaml` | ALB Ingress with weighted target groups for canary traffic |

## Prerequisites

- EKS cluster from `iac/production` with AWS Load Balancer Controller installed
- Pods labelled `deploy-color: blue` (stable) or `deploy-color: green` (canary)
- `DEPLOY_COLOR` env wired from the pod label for Prometheus canary queries (plan 17.9)

## Rolling restart (Phase 1)

```bash
kubectl rollout restart deployment/lextures-api -n lextures
# or
deploy/scripts/rolling_restart.sh kubectl lextures lextures-api
```

## Blue/green + canary (Phase 2–3)

1. Deploy green replicas with `deploy-color: green` and new image tag.
2. Shift traffic: `deploy/scripts/traffic_split.sh terraform staging 5`
3. Run canary analysis: `python3 deploy/scripts/canary_analyze.py --canary-color green`
4. Promote (`100`) or rollback (`0`) via `traffic_split.sh`.

See `docs/runbooks/production-deploy-canary.md` for the full operator procedure.
