# Cache invalidation patterns (plan 17.5)

Redis object cache keys follow `{prefix}:{entity_type}:{id}:{variant}`:

| Key | Invalidated when |
|---|---|
| `cache:course:{id}:structure:staff` | Course structure create/update/delete/reorder |
| `cache:course:{id}:structure:student` | Same as staff variant |
| `cache:course:{id}:enrollments` | Enrollment add/update/remove |
| `cache:catalog:page:{hash}` | Course publish/unpublish to public catalog |
| `cache:user:{id}:calendar` | Assignment due date change, calendar token rotation |

Call the `Deps` helpers in `server/internal/httpserver/cache_layer.go` from mutation handlers in the same function that writes to Postgres.

Feature flag: `ffRedisCache` in Settings → Global platform (default off). When disabled, handlers fall through to the database with no error.

To flush cache for a course manually (runbook):

```bash
redis-cli DEL "cache:course:<course-uuid>:structure:staff" "cache:course:<course-uuid>:structure:student" "cache:course:<course-uuid>:enrollments"
redis-cli --scan --pattern "cache:user:*:calendar:course:<course-uuid>" | xargs redis-cli DEL
```
