# Marketing media operations

Marketing content images are stored through the shared file-storage driver under
`marketing/media/{asset-id}/`. The database row in `marketing.content_media` is the inventory;
`marketing.content_article_media` is a derived usage index refreshed whenever an article is saved.

Uploads are limited to 10 MB, MIME-sniffed, checksum-deduplicated, and scanned when `CLAMAV_ADDR`
is configured (or `CLAMAV_STUB=1` in development). SVG and animated GIF uploads are rejected in v1.
Every non-decorative image requires human-approved alt text. Treat all uploaded screenshots as
public and never include real learner data.

The original plus 1600, 800, and 400 pixel renditions are stored when the source is large enough.
Public URLs are immutable. S3 deployments redirect reads to the configured CDN/storage URL; the
marketing build later localises referenced files into its own output.

## Reconciliation and emergency removal

Compare `Storage.ListObjects("marketing/media/")` with all live `storage_key` and rendition keys in
`marketing.content_media`. Unmatched objects are orphans and should be reviewed before deletion.
Missing renditions should be regenerated from `original`; public delivery falls back to original.

For a legal or privacy removal, first remove every article/author reference, soft-delete the asset,
delete every key below its exact `marketing/media/{asset-id}/` prefix, and invalidate that exact CDN
prefix. Record the incident and affected article paths in the admin audit trail.
