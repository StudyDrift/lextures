# Marketing content authoring API

The authenticated authoring API is rooted at `/api/v1/admin/marketing`. When the
`ff_marketing_content` platform flag is disabled, every route returns `404`.

Article updates use optimistic concurrency. Read `revisionNo`, send it back as
`expectedRevisionNo` on `PATCH` and transition/restore requests, and reload when a `409`
returns a newer `currentRevisionNo`, `updatedBy`, and `updatedAt`. Every accepted write appends a
revision; restoring an old snapshot creates another revision and never rewrites history.

Status is not writable through `PATCH`. Use `POST /articles/{id}/transition` with one of:

```text
draft --submit_review--> in_review --approve--> draft
  |                         |--request_changes--> changes_requested --restore_draft--> draft
  |--publish--> published --unpublish--> draft
  |--schedule--> scheduled --publish--> published
published|scheduled --archive--> archived --restore_draft--> draft
```

Scheduling requires a future `scheduledFor`. Deleting an article that has ever been published
requires publish permission and a `redirectTo`. Changing a published path automatically records a
301 redirect. Preview tokens are article- and revision-bound and expire after 30 minutes.

Permissions are independent: `view`, `author`, `review`, `publish`, and `admin` map to the
corresponding `global:app:marketing-content:*` permission. Holding one does not imply another.
