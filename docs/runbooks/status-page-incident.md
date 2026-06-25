# Runbook: Posting an incident update during an outage

Use this when Lextures is degraded or unavailable and users need a public status update.

## Prerequisites

- Statuspage.io admin access for `status.lextures.io`
- On-call engineer or platform operator role

## Steps

1. Open the [Statuspage admin](https://manage.statuspage.io/) for the Lextures page.
2. Click **Create incident**.
3. Select affected components (API, Web App, Database, Job Queue, AI Services, Media/File Storage).
4. Set impact (`Minor`, `Major`, or `Critical`) and initial status (`Investigating`).
5. Publish the first update with a plain-language summary for instructors and students.
6. Post follow-up updates at least every 30 minutes until resolved.
7. Resolve the incident and publish a short postmortem summary when service is stable.

## In-app banner

The web app polls `GET /api/v1/status-summary` every five minutes. Active incidents appear automatically in the incident banner with a link to `status.lextures.io`. No code deployment is required.

## Automated component updates

When Alertmanager fires a sustained alert (≥ 5 minutes), it posts to:

`POST /api/v1/internal/ops/alertmanager-webhook`

with `Authorization: Bearer <ALERTMANAGER_WEBHOOK_SECRET>`. Alerts should include a `statuspage_component` label (`api`, `web_app`, `database`, `job_queue`, `ai_services`, `media_storage`).

If automation over-reports degradation, tune alert thresholds in Prometheus and resolve the incident manually in Statuspage.