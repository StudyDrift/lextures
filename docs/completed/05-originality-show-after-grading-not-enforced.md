# 05 — Originality "show after grading" visibility policy is not enforced

- **Category:** Bug (configured policy ignored)
- **Severity:** P1
- **Area:** Submissions / grading integrity — originality reports (plan 3.14 / 3.5)
- **Status:** Fixed (2026-06-22)

## Summary

The originality report student-visibility setting supports `show_after_grading`, but the
access check returned `true` immediately for that mode **without verifying the submission had
been graded**. A student could therefore view their originality report **before grading**,
contradicting the configured policy.

## Fix

`server/internal/httpserver/originality_http.go` now checks `course.course_grades.posted_at`
for the submitting student before allowing access when `student_visibility = show_after_grading`.
Unposted grades return `403 Forbidden`. `show` and `hide`/default modes are unchanged.

Unit tests in `server/internal/httpserver/originality_nodb_test.go` cover each visibility mode,
including graded vs. ungraded for `show_after_grading`.

## Original evidence

```go
// ~line 122-133 (before fix)
if sc.SubmittedBy == viewer {
    switch sc.StudentVisibility {
    case "show":
        return true
    case "show_after_grading":
        // MVP: allow student view when visibility is configured (grade check deferred).
        return true                      // <-- no "is graded?" check
    default:
        apierr.WriteJSON(w, http.StatusForbidden, apierr.CodeForbidden, "Forbidden.")
        return false
    }
}
```

## Acceptance criteria

- With `student_visibility = show_after_grading`, the submitting student receives the
  originality report only after the submission is graded/posted, and is denied before then.
- `show` and `disabled`/default modes are unchanged.