# W04 — Report-Card AI Comment: Attendance Wiring (Bug)

> Implementation plan. Source: web market-readiness scan (2026-07-06). Related: [docs/completed/13-k12-specific/13.4-report-cards.md](../13-k12-specific/13.4-report-cards.md).

## Implementation notes (2026-07)

- **Server:** `GET /api/v1/courses/:code/report-cards/:period` now includes optional `absences` per card when attendance is enabled, the grading period resolves to calendar dates, and at least one attendance record exists in range. Counts combine section-based (`attendance_records`) and session-based (`attendance_session_records`) data.
- **Grading period dates:** `reportcards.ResolveGradingPeriodDateRange` prefers an org term name match, then parses `Q1-2026` / `S1-2026` labels.
- **AI endpoint:** `POST /api/v1/ai/report-card-comment` accepts optional `absences`; when omitted the prompt explicitly avoids attendance claims.
- **Client:** `absencesForAIComment()` in `report-cards-api.ts` reads the batched card payload; `fetchAICommentSuggestion` omits `absences` from the JSON body when unknown.
- **Tests:** Vitest (`report-cards-api.test.ts`), Go unit tests (`grading_period_test.go`), Playwright (`report-cards.spec.ts` AI payload).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W04 |
| **Section** | Web / K-12 Specific (bug fix) |
| **Severity** | MINOR |
| **Markets** | K12 |
| **Status (today)** | DONE |
| **Estimated effort** | XS (≤1d) |
| **Owner (proposed)** | K-12 pod |
| **Depends on** | 13.2 (attendance), 13.4 (report cards) |
| **Unblocks** | Trustworthy AI report-card comments |

---

## 1. Problem Statement

On the standards-based report-card page, the "AI comment suggestion" feature generates a teacher comment
from the course name, the student's final grade, and their **absences** — but the web client passes a
hardcoded `const absences = 0` (`pages/lms/CourseReportCards.tsx:227`, with a live `// TODO: wire from
attendance summary`). Every AI-generated comment is produced as if the student has perfect attendance.
For a K-12 teacher writing dozens of report cards, this silently injects wrong attendance framing into
comments the teacher may not catch before releasing to families.

## 2. Goals

- Pass the student's real absence count (for the report-card period) into the AI comment suggestion.
- Remove the hardcoded `absences = 0` and the TODO.
- Ensure the value degrades safely when attendance data is unavailable.

## 3. Non-Goals

- Redesigning the AI comment prompt or the report-card page.
- Adding attendance *display* to the report card (that is report-card scope, 13.4).
- Parent-facing attendance (that is W02).

## 4. Personas & User Stories

- **As a teacher**, I want the AI-suggested comment to reflect my student's actual attendance so that I
  don't have to rewrite comments that wrongly imply perfect attendance.
- **As a parent**, I want report-card comments to be accurate about my child's attendance.

## 5. Functional Requirements

- **FR-1.** The report-card page MUST source the student's absence count for the selected reporting
  period from the attendance summary (13.2) and pass it to `fetchAICommentSuggestion`.
- **FR-2.** If attendance data is unavailable for the student/period, the request MUST omit absences (or
  pass an explicit "unknown") so the prompt does NOT assert perfect attendance — it MUST NOT default to 0.
- **FR-3.** The hardcoded `absences = 0` and its TODO MUST be removed.
- **FR-4.** The absence count MUST be scoped to the same reporting period as the report card being edited.

## 6. Non-Functional Requirements

- **Performance** — Attendance summary fetched once per card load (or batched with existing card data),
  not per suggestion click.
- **Security** — Instructor authorization for the course/section already enforced; no new surface.
- **Privacy & Compliance** — FERPA: attendance is an education record; it stays within the instructor's
  authorized scope and is only used to draft a comment the instructor reviews/edits before release.
- **Accessibility** — No UI change; existing `announce()` on suggestion insertion retained.
- **Reliability** — Attendance fetch failure MUST NOT block comment generation; it degrades to
  "attendance unknown".
- **Observability** — Optional: metric `report_card_ai_comment_absences_source` = {actual|unknown}.
- **Maintainability** — Absence lookup lives in the report-card data layer, not inline in the handler.
- **Internationalization** — Comment prompt/output localization tracked under W01.

## 7. Acceptance Criteria

- **AC-1.** *Given* a student with 3 absences in the reporting period, *When* the teacher clicks "AI
  suggest", *Then* the suggestion is generated with `absences = 3` (verified via the request payload).
- **AC-2.** *Given* attendance data is missing, *When* the teacher clicks "AI suggest", *Then* the
  request does not send `absences = 0` and the generated comment makes no perfect-attendance claim.
- **AC-3.** *Given* the code, *When* reviewed, *Then* no hardcoded `absences = 0` / TODO remains at
  `CourseReportCards.tsx:~227`.

## 8. Data Model

- No schema change. Reads the existing attendance summary for `(student, period, course/section)`.

## 9. API Surface

- Reuse the attendance-summary endpoint (13.2) scoped to the reporting period, or extend the report-card
  card payload to include `absences` per student so the client has it without an extra round-trip.
- `fetchAICommentSuggestion` signature unchanged (already accepts an absences argument).

## 10. UI / UX

- **Modified:** `pages/lms/CourseReportCards.tsx` `handleAISuggest`.
- **States:** unchanged; existing `aiLoading`, `announce`, and error handling retained.
- No visual change.

## 11. AI / ML Considerations

- Model/prompt already exists (`fetchAICommentSuggestion`). This plan only corrects an **input** to the
  prompt. Fallback: omit absences when unknown. No new cost/PII surface (instructor-scoped data the
  instructor already sees).

## 12. Integration Points

- `clients/web/src/pages/lms/CourseReportCards.tsx`, the report-card data layer, and the attendance
  summary API (13.2).

## 13. Dependencies & Sequencing

- **Must ship after:** 13.2, 13.4 (completed).
- **Must ship before:** any marketing of "AI report-card comments" as accurate.
- **Shared infra:** none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Attendance period misaligned with report-card period | M | M | Scope the query to the card's reporting period; test AC-1. |
| Extra fetch slows card page | L | L | Batch absences into the existing card payload. |
| Missing data silently becomes 0 again | L | M | FR-2 + AC-2 explicitly forbid the 0 default. |

## 15. Rollout Plan

- **Feature flag:** none (bug fix).
- **Sequencing:** wire attendance → remove hardcode → ship.
- **GA criteria:** ACs pass.
- **Rollback:** revert single commit.

## 16. Test Plan

- **Unit** — `handleAISuggest` passes the fetched absence count; unknown → omitted, not 0.
- **Integration** — attendance summary → suggestion request payload.
- **End-to-end** — Playwright: teacher opens report cards, clicks AI suggest, request carries real absences.
- **Manual exploratory** — student with absences vs. student with no attendance records.

## 17. Documentation & Training

- Note in the report-card instructor help that AI comments consider attendance.
- Changelog entry (bug fix).

## 18. Open Questions

1. Include *tardies* as well as absences in the comment input?
2. Enrich the card payload with `absences` server-side vs. a separate client fetch? **Resolved:** server-side batch on list endpoint.

## 19. References

- `clients/web/src/pages/lms/CourseReportCards.tsx` (removed hardcoded absences).
- Related plans: [13.4](../13-k12-specific/13.4-report-cards.md),
  [13.2](../13-k12-specific/13.2-daily-attendance.md),
  [W02](W02-parent-guardian-portal-completeness.md).