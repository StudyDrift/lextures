# Runbook: Scheduling a maintenance window

Use this for planned maintenance that may affect Lextures availability.

## Prerequisites

- Statuspage.io admin access
- Maintenance window approved by platform lead

## Steps

1. Open the Statuspage admin for `status.lextures.io`.
2. Click **Create maintenance** (scheduled maintenance).
3. Select affected components and the maintenance window start/end times.
4. Schedule at least 48 hours in advance when possible so email subscribers receive advance notice.
5. Describe expected user impact (read-only mode, brief login disruption, etc.).
6. Post a reminder update 1 hour before the window begins.
7. Mark maintenance **In progress** at start time and **Completed** when finished.

## Subscriber notifications

Email and RSS subscribers are notified automatically by Statuspage when the maintenance is scheduled and when it begins or ends. No separate mailing list update is required.

## Rollback

If maintenance must be cancelled, edit the scheduled maintenance in Statuspage and mark it completed with a cancellation note.