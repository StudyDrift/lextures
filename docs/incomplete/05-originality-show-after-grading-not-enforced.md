# 05 — Originality "show after grading" visibility policy is not enforced

- **Category:** Bug (configured policy ignored)
- **Severity:** P1
- **Area:** Submissions / grading integrity — originality reports (plan 3.14 / 3.5)

## Summary

The originality report student-visibility setting supports `show_after_grading`, but the
access check returns `true` immediately for that mode **without verifying the submission has
been graded**. A student can therefore view their originality report **before grading**,
contradicting the configured policy.

## Evidence

`server/internal/httpserver/originality_http.go`:

```go
// ~line 122-133
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

The inline comment ("grade check deferred") confirms the gap: `show_after_grading` behaves
identically to `show`.

## Impact

- Instructors who chose "show after grading" to avoid students gaming similarity scores
  before feedback get the opposite behaviour — reports are visible pre-grade.
- Quietly weakens academic-integrity workflows that depend on the timing of disclosure.

## Suggested fix

- In the `show_after_grading` branch, return `true` only when the submission has a posted
  grade (respecting grade-posting policies, plan 3.8); otherwise return `403`/`404`.
- Add a unit test for each `StudentVisibility` mode, including the
  graded vs. ungraded distinction for `show_after_grading`.

## Acceptance criteria

- With `student_visibility = show_after_grading`, the submitting student receives the
  originality report only after the submission is graded/posted, and is denied before then.
- `show` and `disabled`/default modes are unchanged.
