# Marketing content publishing

Publishing an article commits the database change immediately and queues a static-site rebuild. Normal publishes are coalesced for the configured quiet period (three minutes by default); unpublishes and manual rebuilds dispatch immediately. The database remains authoritative if GitHub Actions is unavailable—the currently deployed site remains online and the workspace reports the failed or timed-out build.

## Configure GitHub dispatch

In the marketing build settings API, set:

- `provider` to `github` (the safe default is `none`)
- `repository` to the single allowed `owner/repository`
- `workflowRef` to the production branch, normally `main`
- a fine-grained GitHub token scoped only to that repository and the permissions needed to dispatch the Pages workflow
- `quietSeconds` and `maxWaitSeconds` (defaults: 180 and 900)

The token requires `PLATFORM_SECRETS_KEY` to be configured and is encrypted at rest. Reads expose only `tokenConfigured`; the plaintext token is never returned. Rotate it by submitting a replacement token. Set `provider` to `none` to stop dispatches without blocking content publishing.

## Diagnose a delayed publish

1. Check `GET /api/v1/admin/marketing/builds` and the article's `latestBuild`.
2. For `failed` or `timed_out`, open `providerRunUrl` when present and correct the workflow or credential problem.
3. Request a manual rebuild with `POST /api/v1/admin/marketing/builds` (limited to six per user per hour).
4. If GitHub dispatch is unavailable, a manual Pages workflow dispatch remains the operational fallback.

The scheduler jobs `marketing_content_publish_due` and `marketing_content_build_dispatch` must remain enabled. Both run every minute. Scheduled validation failures leave the article scheduled and write a publish event with an error so an editor can correct the findings.

## Monitoring

Dispatch failures log `marketing_content_dispatch_failures_total` with the build ID. Builds still active after 30 minutes become `timed_out`; successful and failed workflow conclusions are reflected in the build API. The Pages workflow continues its sitemap and IndexNow submission after repository-dispatch deploys.
