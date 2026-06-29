# Emergency rollback to previous version (plan 17.9 AC-5)

Use this runbook when a production deploy causes elevated errors and you need the previous API version serving traffic within two minutes.

## Fast path (previous image tag)

1. Identify the last known-good image tag from GHCR or the previous workflow run (commit SHA tags are immutable).
2. Re-deploy blue/stable color with the previous tag:
   ```bash
   kubectl -n lextures set image deployment/lextures-api-blue \
     api=ghcr.io/ORG/server-go:<previous-sha>
   ```
3. Shift traffic to 100% stable:
   ```bash
   deploy/scripts/traffic_split.sh terraform production 0
   ```
4. Verify:
   ```bash
   curl -fsS https://api.example.com/health/ready
   ```
5. Notify on-call channel:
   ```bash
   deploy/scripts/notify_deploy.sh rollback production <previous-sha> manual "emergency rollback"
   ```

## When canary auto-rollback already ran

If the deploy pipeline rolled back automatically, confirm traffic is on blue (`deploy_canary_weight=0` in Terraform) and delete unhealthy green pods:

```bash
kubectl -n lextures delete pods -l deploy-color=green
```

## Database considerations

- Do **not** run down migrations during emergency rollback unless plan 17.10 runbook explicitly covers the change.
- If the failed deploy applied a backward-incompatible migration, restore from RDS snapshot per `docs/runbooks/database-backup-restore.md`.

## Post-incident

- File an incident in the status page runbook if user-visible impact occurred.
- Capture Prometheus/Grafana snapshots from the canary window.
- Open a follow-up to fix the regression before re-attempting deploy.
