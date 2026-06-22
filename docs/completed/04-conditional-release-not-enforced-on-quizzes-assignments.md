# 04 — Conditional release is not enforced server-side for quizzes & assignment submissions

- **Category:** Bug / integrity gap (documented MUST not enforced)
- **Severity:** P1
- **Area:** Adaptive learning core / conditional release & module requirements (plan 1.11)
- **Status:** Fixed (2026-06-22)

## Summary

Conditional release was enforced for content pages but not for quiz start/submit, assignment
submission upload, or survey responses. Students could bypass module locks via direct API calls.

## Fix

Added `enforceConditionalReleaseForLearner` in `server/internal/httpserver/conditional_release_http.go`
and wired it into:

- `handleQuizStart` (`quiz_delivery_http.go`)
- `handleQuizSubmit` (`quiz_submit_http.go`)
- `handlePostAssignmentSubmissionUpload` (`assignment_submission_upload_http.go`)
- `handleSurveyRespond` (`surveys_api.go`)

Instructors (`course:CODE:item:create`) still bypass gating. Locked students receive `403` with a
`reason` payload, matching content-page behaviour.

E2E coverage added in `e2e/tests/conditional-release.spec.ts` for quiz start, quiz submit, and
assignment upload behind a module prerequisite.

## Acceptance criteria

- With conditional release enabled and requirements set, a student who has not met the
  prerequisite receives `403` (with reason) from quiz-start, quiz-submit, and
  assignment-submit endpoints — not just content-page views.
- Instructors (`canEdit`) still bypass gating.