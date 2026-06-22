# 07 — Daily email digest is sent at a fixed 07:00 UTC, ignoring per-user timezone

- **Category:** Feature not fully implemented (simplification)
- **Severity:** P3/P4
- **Area:** Communication / email notifications (plan 6.2)

## Summary

The daily email worker fires its digest window at a fixed **07:00 UTC** for all users. The
per-user timezone handling that would deliver "07:00 in the learner's local time" was
deferred, so users in other timezones receive the daily digest at an off-hours local time.

## Evidence

`server/internal/background/email_worker.go`:

```go
// line 102
// 07:00 UTC daily window (simplified; per-user TZ deferred).
```

## Impact

- A user in, say, US Pacific receives the "morning" digest at ~23:00–00:00 local; users in
  Asia/Pacific receive it mid-afternoon. Reduces open rates and feels broken to end users.
- Cosmetic from an engineering standpoint, but user-visible.

## Suggested fix

- Resolve each recipient's timezone (already captured elsewhere for scheduling/calendar)
  and bucket the send so the digest lands near 07:00 local. A per-timezone cron sweep or a
  per-user "next send at" column both work.
- If single-timezone delivery is acceptable for now, make the hour configurable and
  document the behaviour.

## Acceptance criteria

- Two users in different timezones each receive the daily digest near 07:00 in **their**
  local time.
