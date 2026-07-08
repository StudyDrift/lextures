# IC05 — Progress Tracking, Completion & Completion Credential

> Implementation plan. Source: product direction — the intro course is a *course* users complete;
> closes the onboarding loop. Follows [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | IC05 |
| **Section** | Intro Course |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Backend platform + web |
| **Depends on** | IC02 (students), IC03 (content), IC04 (grades) |
| **Unblocks** | IC06 (surfaces show progress/completion) |

---

## 1. Problem Statement

Once students are enrolled (IC02), taught (IC03), and graded (IC04), the onboarding loop needs a
close: a legible sense of **progress** ("3 of 7 modules done"), a definition of **completion**,
and a small **reward** (a completion credential / certificate) that marks the user as onboarded.
Without this, the course has no finish line, the platform can't tell who is activated, and there's
no signal to stop nudging a finished user or to celebrate them.

## 2. Goals

- Compute and expose per-student **progress** through the intro course (modules/items completed,
  percent, running grade).
- Define **completion** deterministically (e.g. every module's quiz attempted + capstone
  submitted, or ≥ threshold final grade) and record a **completed_at** per student.
- On completion, issue an optional **completion credential** via the existing credentials
  machinery (`ff_completion_credentials`) — a shareable "Lextures Onboarding" certificate.
- Emit a **completion signal/event** other systems consume (stop onboarding nudges, mark the user
  "activated", light up IC06 surfaces, feed activation analytics).
- Make it all **idempotent** and recomputable from grades/progress (no hidden state).

## 3. Non-Goals

- The grading itself (IC04) or content (IC03).
- General-purpose course completion for *all* courses (this plan scopes to the intro course; it
  reuses generic machinery where it exists but does not build a platform-wide completion engine).
- Discovery/entry-point and celebration UI placement (IC06 renders; this plan provides the data +
  the credential).
- Mobile rendering specifics (IC07).

## 4. Personas & User Stories

- **As a student**, I want to see how far through the intro course I am and what's left, so I know
  the finish line.
- **As a student**, I want a certificate when I finish, so completing feels worthwhile and
  shareable.
- **As the platform**, I want to know which users have completed onboarding, so I can stop nudging
  them and measure activation.
- **As an admin**, I want intro-course completion rates, so I can judge onboarding effectiveness.

## 5. Functional Requirements

- **FR-1.** The system MUST compute per-student progress: count of required items completed vs.
  total, a percent, and the current running grade (from IC04's gradebook). "Reading" a content
  page counts toward participation via existing engagement events.
- **FR-2.** The system MUST define **completion** by a single documented rule (default: all module
  knowledge-check quizzes attempted **and** the capstone submitted; final grade ≥ configurable
  threshold, default 0 since items auto-credit). It MUST record `completed_at` once, idempotently.
- **FR-3.** On first completion, when `ff_completion_credentials` is on, the system MUST issue a
  completion credential ("Welcome to Lextures — Onboarding Complete") through the existing
  credentials service; when off, completion is still recorded (no certificate).
- **FR-4.** The system MUST emit a **completion event** (business event / notification) exactly
  once per student, consumable to: suppress onboarding nudges, mark activation, and (optionally)
  send a congratulatory inbox/email message.
- **FR-5.** Progress and completion MUST be **recomputable** from grades + engagement (no
  divergent source of truth); a recompute after a grade change MUST update progress and, if newly
  qualifying, set `completed_at` and issue the credential once.
- **FR-6.** Endpoints MUST expose the caller's own progress (`GET /api/v1/me/intro-course`) and an
  admin aggregate (completion rate, funnel by module) for IC08/analytics.
- **FR-7.** Everything MUST be flag-gated by `intro_course_enabled`; disabling stops new
  completions/credentials but retains recorded completions.

## 6. Non-Functional Requirements

- **Performance** — `GET /me/intro-course` p95 ≤ 80 ms (reads materialized progress or computes
  from a small bounded item set). Completion check on grade-write is O(items) and cheap.
- **Security** — A learner reads only their own progress; admin aggregate requires admin scope.
  Credential issuance is server-side and idempotent (no client-forged certificates).
- **Privacy & Compliance** — Completion + credential are education records: exported/erased via
  existing machinery. A shareable certificate (LinkedIn share exists, migration 283) MUST only
  expose what the user consents to share.
- **Accessibility** — Progress and certificate UI (rendered in IC06) WCAG 2.1 AA; progress
  conveyed by text + not color alone.
- **Scalability** — Progress derivable on read from a bounded item set; optionally cached per
  student, invalidated on grade change. Completion event fired once (dedup key).
- **Reliability** — Idempotent `completed_at` (set-once), idempotent credential issuance (unique
  per user+course), at-least-once completion event with dedup.
- **Observability** — `intro_course_completion_total`, `intro_course_progress_recompute_total`,
  `intro_course_credential_issued_total`; a completion-rate gauge for dashboards.
- **Maintainability** — Completion rule in one place (`introcourse/completion.go`), documented and
  testable; no rule duplication in UI.
- **Internationalization** — Certificate + congrats copy are i18n keys (IC08).
- **Backward compatibility** — Additive; reuses credentials + notifications infra.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student has finished 3 of 7 modules, *when* they call
  `GET /me/intro-course`, *then* it returns modules-complete=3/7, a percent, and the running grade.
- **AC-2.** *Given* a student meets the completion rule, *when* their last qualifying grade is
  written, *then* `completed_at` is set exactly once and a completion event fires once.
- **AC-3.** *Given* `ff_completion_credentials` is on and a student completes, *then* exactly one
  completion credential is issued; re-running the check issues no duplicate.
- **AC-4.** *Given* `ff_completion_credentials` is off, *when* a student completes, *then*
  completion is recorded but no credential is issued (no error).
- **AC-5.** *Given* a completed student, *when* onboarding nudges run, *then* they are suppressed
  for that user (consumes the completion signal).
- **AC-6.** *Given* an admin, *when* they query the aggregate, *then* they see completion rate and
  a per-module funnel; a non-admin cannot.
- **AC-7.** *Given* a grade correction that pushes a previously-incomplete student over the bar,
  *when* recompute runs, *then* they complete (event + credential once).

## 8. Data Model

```sql
-- server/migrations/364_intro_course_completion.sql
CREATE TABLE settings.intro_course_completions (
    user_id       UUID PRIMARY KEY REFERENCES "user".users (id) ON DELETE CASCADE,
    completed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    final_grade   DOUBLE PRECISION,          -- snapshot at completion
    credential_id UUID,                      -- existing credentials table FK when issued
    event_sent    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Progress itself is **derived** (not stored) from `course.course_grades` + engagement page-views +
`settings.intro_course_items`; optionally memoized in a cache keyed by user, invalidated on grade
write. `credential_id` references the existing credentials/certificate table (confirm exact table
in `server/internal/httpserver/credentials_http.go`). Backfill: on enable, a recompute pass can
mark already-qualifying students complete (idempotent; credential issuance rate-limited).

## 9. API Surface

```
GET /api/v1/me/intro-course                     Auth: self
  200: { enrolled, modulesComplete, modulesTotal, percent, runningGrade,
         completedAt?, credentialId?, nextItem?: { slug, title, route } }

GET /api/v1/admin/intro-course/analytics        Auth: platform-admin
  200: { enrolled, completed, completionRate, perModuleFunnel: [...] }
```

Internal:

```go
// server/internal/service/introcourse/completion.go
func LoadProgress(ctx, exec, userID) (Progress, error)
func RecheckCompletion(ctx, pool, userID) (Progress, error) // sets completed_at + issues credential + fires event, all once
```

`RecheckCompletion` is invoked from IC04's grade-write hook and a nightly sweep.

## 10. UI / UX

Data + credential provided here; **placement** is IC06 (dashboard progress ring, in-course
"next up", completion celebration). This plan defines:

1. Progress payload (percent, modules done, next item + deep link).
2. Completion state (`completedAt`, `credentialId`) so IC06 can show a certificate/badge and a
   "You've completed Welcome to Lextures 🎉" celebration.
3. Optional congratulatory inbox message (via existing communication) on completion.

Empty/loading/error: pre-start shows 0%; error falls back to a link into the course.

## 11. AI / ML Considerations

None (deterministic rule). If IC04's grader-agent produced capstone feedback, IC05 may surface it
alongside completion, but completion itself is rule-based.

## 12. Integration Points

- **Grades:** IC04 gradebook / `course.course_grades` (completion trigger + running grade).
- **Engagement:** page-view events (participation toward completion).
- **Credentials:** existing credential/certificate service + `ff_completion_credentials`
  (`credentials_http.go`, LinkedIn share migration 283).
- **Notifications:** existing communication/inbox + email (`background/jobqueue_email.go`) for the
  congrats message.
- **Onboarding nudges / activation analytics:** consume the completion event.
- **Nightly sweep:** `server/internal/background/scheduled_jobs.go`.

## 13. Dependencies & Sequencing

- **After:** IC02 (students), IC03 (content/items), IC04 (grades to complete against).
- **Before:** IC06 (renders progress/completion), analytics dashboards (IC08).
- **Shared infra:** credentials, notifications, scheduler — present.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Duplicate credentials / repeated congrats | M | M | Set-once `completed_at`, `event_sent` flag, unique credential per user+course |
| Completion rule too strict → low completion | M | M | Auto-crediting items (IC04) + generous rule; threshold configurable |
| Progress cache diverges from grades | M | M | Invalidate on grade write; nightly recompute; progress is derivable, cache optional |
| Credential issuance floods on backfill | M | M | Rate-limit issuance; issue lazily on next read for pre-existing completers |
| Shared course + moving completion bar on content change | L | M | Completion rule keyed to *current* required items; re-sync updates required set deliberately |

## 15. Rollout Plan

- **Flag:** `intro_course_enabled`; credential additionally gated by `ff_completion_credentials`.
- **Sequencing:** progress compute + endpoint → completion rule + `completed_at` + event →
  credential issuance (flagged) → nightly sweep → enable.
- **Dogfood:** internal users complete the course; verify progress, single completion event,
  certificate, and nudge suppression.
- **GA criteria:** idempotent completion + credential; admin analytics correct; nudges suppressed
  for completers.
- **Rollback:** disable credential issuance (completion still recorded) or `intro_course_enabled`.

## 16. Test Plan

- **Unit** — completion rule; progress math; set-once completion; single-issue credential; event
  dedup.
- **Integration (DB)** — grade write crosses the bar → complete once (+event+credential); recheck
  is idempotent; flag-off skips credential; erase removes completion row.
- **End-to-end** — student finishes course → `GET /me/intro-course` shows completed + credential;
  admin analytics reflect it.
- **Security** — self-only progress; admin-only analytics; no client-forged completion/credential.
- **Accessibility** — progress + certificate states (rendered via IC06) pass axe.
- **Performance** — progress read p95; completion check overhead on grade write negligible.
- **Manual exploratory** — grade correction re-triggers completion; backfill of pre-existing
  completers issues one credential each.

## 17. Documentation & Training

- Runbook: how completion is defined, force a recompute, re-issue a lost credential.
- Admin doc: reading completion-rate analytics; what the certificate is.
- Help-center: "Completing the Welcome to Lextures course & your certificate."

## 18. Open Questions

1. Exact completion rule and grade threshold (product): all-quizzes-attempted + capstone, or a
   grade floor? (Leaning attempt-based since items auto-credit.)
2. Should completion feed the [learner profile](../learner-profile/README.md) / recommendations as
   an explicit "onboarded" signal, or is behavioural signal enough? (Leaning behavioural only.)
3. Certificate design/branding and whether sharing (LinkedIn) is on by default or opt-in. (Leaning
   opt-in share.)
4. Do we re-open completion if the curriculum later adds required modules? (Leaning: no — keep
   prior completion; new content is optional for already-completed users.)

## 19. References

- Existing files: `server/internal/httpserver/credentials_http.go`,
  `server/migrations/283_credentials_linkedin_share.sql`,
  `server/internal/background/{scheduled_jobs.go,jobqueue_email.go}`, `course.course_grades`.
- Related plans: [IC02](IC02-automatic-enrollment.md), [IC04](IC04-graded-assessments-autograding.md),
  [IC06](IC06-web-onboarding-surfaces.md), [IC08](IC08-admin-governance-localization.md).
