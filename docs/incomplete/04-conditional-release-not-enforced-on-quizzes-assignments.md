# 04 — Conditional release is not enforced server-side for quizzes & assignment submissions

- **Category:** Bug / integrity gap (documented MUST not enforced)
- **Severity:** P1
- **Area:** Adaptive learning core / conditional release & module requirements (plan 1.11)

## Summary

Conditional release (module prerequisites, completion requirements, date locks) is
implemented (`competencygating` service + migration `303_conditional_release.sql`) and
**enforced for content pages**, but the enforcement gate is **not applied to quizzes or
assignment submissions**. A locked quiz or assignment remains startable/submittable via its
normal API route, which violates the feature's own requirement.

Plan `docs/completed/01-adaptive-learning-core/1.11-conditional-release-module-requirements.md`
**FR-4** (MUST):

> The system MUST enforce gating server-side: a locked item/module's content is not served
> and **submissions are rejected, regardless of direct URL access**.

## Evidence

The enforcement helper exists and is generic over item type:

```go
// server/internal/httpserver/conditional_release_http.go:349
func (d Deps) enforceConditionalRelease(w, r, courseID, viewer, itemID, canEdit) bool { ... }
```

…but it is **only called from the content-page handler**:

```
$ grep -rn "enforceConditionalRelease" server/internal/httpserver --include=*.go | grep -v conditional_release_http.go
server/internal/httpserver/module_content_page.go:131: if !d.enforceConditionalRelease(w, r, *cid, viewer, itemID, canEdit) {
```

**Quiz start** uses only `QuizVisibleToStudent`, which by its own doc comment **excludes
competency/conditional gating**:

```go
// server/internal/repos/coursestructure/quiz_visible.go:15
// QuizVisibleToStudent mirrors `course_structure::quiz_visible_to_student`
// (competency gating not yet ported).
```

```go
// server/internal/httpserver/quiz_delivery_http.go:65
visible, err := coursestructure.QuizVisibleToStudent(ctx, d.Pool, *cid, itemID, viewer, now)
// no CheckItemAccess / enforceConditionalRelease call on this path
```

**Assignment submission upload** has no gating reference at all:

```
$ grep -n "CheckItemAccess\|enforceConditional\|AssignmentVisibleToStudent\|locked" \
    server/internal/httpserver/assignment_submission_upload_http.go
(no matches)
```

The sibling visibility functions carry the same "competency gating not yet ported" caveat:
`server/internal/repos/coursestructure/assignment_visible.go:15`,
`server/internal/repos/coursestructure/survey_visible.go:14`,
`server/internal/repos/coursestructure/list.go:424`.

## Impact

- A student can **bypass a module/prerequisite lock** for quizzes and assignments by
  hitting the start/submit endpoint directly (or via a stale link), defeating
  mastery-based progression / CBE sequencing — the entire point of the feature.
- Inconsistent UX: content pages respect the lock, quizzes/assignments do not.

## Suggested fix

1. Call `enforceConditionalRelease(...)` (or `gatingService().CheckItemAccess(...)`) on:
   - quiz attempt **start** (`quiz_delivery_http.go`),
   - quiz attempt **submit**,
   - assignment **submission upload/create** (`assignment_submission_upload_http.go`,
     `assignment_submissions_http.go`),
   - survey responses, if surveys participate in gating.
2. Alternatively, fold the conditional-release check into the shared
   `*VisibleToStudent` functions in `repos/coursestructure` (removing the "not yet ported"
   caveat) so every delivery path inherits it.
3. Add an e2e test: lock module B behind module A, then assert a direct
   `POST .../attempts` and `POST .../submission` for a B item returns 403 with the lock
   reason for a student who has not satisfied A.

## Acceptance criteria

- With conditional release enabled and requirements set, a student who has not met the
  prerequisite receives `403` (with reason) from quiz-start, quiz-submit, and
  assignment-submit endpoints — not just content-page views.
- Instructors (`canEdit`) still bypass gating.
