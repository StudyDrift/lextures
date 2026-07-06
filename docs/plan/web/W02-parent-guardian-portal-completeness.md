# W02 — K-12 Parent/Guardian Portal Completeness

> Implementation plan. Source: web market-readiness scan (2026-07-06). Related: [docs/completed/13-k12-specific/13.1-parent-portal.md](../../completed/13-k12-specific/13.1-parent-portal.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W02 |
| **Section** | Web / K-12 Specific |
| **Severity** | MAJOR |
| **Markets** | K12 |
| **Status (today)** | THIN |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | K-12 pod (frontend + backend) |
| **Depends on** | 13.1 (parent portal), 13.2 (attendance), 13.3 (behavior/PBIS), 13.4 (report cards), W01 (localization) |
| **Unblocks** | K-12 district adoption; parent-facing NPS |

---

## 1. Problem Statement

The parent/guardian experience on web is a single **251-line, read-only page**
(`pages/lms/parent/parent-dashboard.tsx`) backed by three endpoints (children, grades, assignments).
It has a disqualifying legibility defect: grades render as **raw item-ID prefixes** — the API returns
`grades: Record<itemId, score>` with no titles, so the UI shows `a1b2c3d4… 95` and a parent cannot tell
which assignment a score belongs to. It also omits the things K-12 parents check most — **attendance,
behavior/conduct, and report cards** — even though the platform already has those systems (plans
13.2/13.3/13.4). Messaging is a bare link to the generic Inbox. For a K-12 buyer, the parent portal is a
primary evaluation surface; in its current state it undercuts the sale.

## 2. Goals

- Make grades legible: show assignment title, category, points/percentage, due date, and status.
- Surface attendance (present/absent/tardy summary + recent days) and behavior/PBIS entries per child.
- Give parents read access to released report cards (link to the existing report-card PDF).
- Provide a real parent→teacher message path (pre-addressed to the child's teacher), not just an Inbox link.
- Ship the whole surface localized (per W01), since parent communications are the sharpest language-access case.

## 3. Non-Goals

- Parent *editing* of student data (portal stays read-only except messaging).
- Parent-teacher conference *scheduling* UI — already built (`/parent/conferences`, plan 13.12); this
  plan only links to it.
- Payment/billing for K-12 (not part of the guardian relationship here).
- A separate native parent app (mobile parent surface is tracked in the mobile plans).

## 4. Personas & User Stories

- **As a parent of two students**, I want to switch between my children and see each one's real
  assignment names and scores so that I can help with the right work.
- **As a parent**, I want to see whether my child was marked absent or tardy this week so that I can
  address attendance early.
- **As a parent**, I want to read the report card the teacher released so that I don't have to ask for a
  paper copy.
- **As a parent**, I want to message my child's teacher in one click from the portal so that I don't
  have to figure out who to contact.
- **As a K-12 admin evaluating Lextures**, I want the parent portal to look complete and trustworthy so
  that families adopt it.

## 5. Functional Requirements

- **FR-1.** The grades section MUST display, per graded item: assignment/quiz **title**, category (if
  any), score as points and/or percentage, and posted/graded status — never a raw ID.
- **FR-2.** The parent grades API MUST return item titles (and category) alongside scores, OR the client
  MUST join grades to the already-available assignments list by `itemId`; the UI MUST show `title`.
- **FR-3.** The portal MUST show an attendance summary per child (counts of present/absent/tardy for the
  current term and the most recent N days), sourced from the attendance system (13.2), respecting the
  parent's authorization.
- **FR-4.** The portal MUST show behavior/PBIS entries visible to guardians (13.3), or explicitly hide
  the section when the org has behavior tracking disabled.
- **FR-5.** When a report card has been **released** for the child (13.4), the portal MUST link to its
  PDF (`card.pdfUrl`) for viewing/download.
- **FR-6.** A "Message teacher" action MUST open a compose flow pre-addressed to the relevant course's
  teacher for the selected child.
- **FR-7.** All sections MUST enforce the guardian↔student link authorization server-side; a parent MUST
  only see data for linked, active students.
- **FR-8.** Every section MUST have explicit empty, loading, and error states, and MUST be fully
  localized (W01, namespace `parent`).

## 6. Non-Functional Requirements

- **Performance** — p95 < 400ms for the per-child aggregate load; batch child data in ≤2 requests.
- **Security** — Authorization keyed on the guardian link, not the requested `studentId`; deny by
  default; no IDOR (a parent cannot fetch an unlinked student by guessing an ID).
- **Privacy & Compliance** — FERPA: guardian access to education records; the portal shows only
  guardian-viewable fields. COPPA: no new data collection from students. Behavior data disclosure follows
  district policy toggles.
- **Accessibility** — WCAG 2.1 AA; the "Viewing as parent … read only" banner is a live region; the
  child switcher is a proper listbox (already is) and keyboard operable.
- **Scalability** — Aggregations computed server-side; avoid N+1 per assignment on the client.
- **Reliability** — A failure in one section (e.g. behavior) MUST NOT blank the page; degrade per-section.
- **Observability** — Metrics: `parent_portal_view`, `parent_grades_missing_title` (should trend to 0),
  per-section error rate.
- **Maintainability** — Parent API types in `lib/parent-api.ts`; one `parent` i18n namespace.
- **Internationalization** — Ships under W01's `parent` namespace with `es`/`fr`/`ar`.
- **Backward compatibility** — Existing `/parent` route and links unchanged; additive API fields.

## 7. Acceptance Criteria

- **AC-1.** *Given* a linked child with graded work, *When* the parent opens the portal, *Then* each
  score shows its assignment **title** and percentage — no `id.slice(0,8)…` appears anywhere.
- **AC-2.** *Given* the child has attendance records this term, *When* the parent views the child,
  *Then* present/absent/tardy counts and the most recent days render.
- **AC-3.** *Given* a report card was released, *When* the parent views the child, *Then* a link to the
  report-card PDF is shown and opens the correct document.
- **AC-4.** *Given* a parent clicks "Message teacher", *When* the compose opens, *Then* it is
  pre-addressed to the selected child's course teacher.
- **AC-5.** *Given* a parent requests an unlinked student's data via a crafted request, *When* the API
  handles it, *Then* it returns 403 and logs an authz denial.
- **AC-6.** *Given* the locale is `es`, *When* the portal renders, *Then* all section headings and labels
  are Spanish (W01).

## 8. Data Model

- No new tables expected; reuse guardian-link, attendance (13.2), behavior (13.3), report-card (13.4),
  and enrollment/grade tables.
- API response additions: `title`, `category`, `percentage`, `status` on the parent grade rows;
  attendance-summary and behavior-summary payloads scoped to guardian-viewable fields.
- Indexes: ensure attendance/behavior queries are indexed by `(student_id, term)` for the aggregate.

## 9. API Surface

- `GET /api/v1/parent/students/{id}/grades` — extend rows with `title`, `category`, `percentage`,
  `status` (or add a companion endpoint the client joins on `itemId`).
- `GET /api/v1/parent/students/{id}/attendance-summary` — new; present/absent/tardy counts + recent days.
- `GET /api/v1/parent/students/{id}/behavior` — new (or reuse), guardian-scoped.
- `GET /api/v1/parent/students/{id}/report-cards` — new; released cards with `pdfUrl`.
- All under the parent auth scope; guardian-link authorization enforced; rate-limited per parent.
- OpenAPI updated for each.

## 10. UI / UX

- **Modified page:** `pages/lms/parent/parent-dashboard.tsx`.
- **New sections:** Attendance summary card; Behavior card (conditional on org toggle); Report cards
  list; "Message teacher" CTA per course.
- **Flows:** (1) select child → (2) grades/attendance/behavior/report cards load per section →
  (3) parent taps a grade to see item detail → (4) parent messages the teacher.
- **States:** per-section empty ("No absences this term"), loading skeletons, and error banners that do
  not blank sibling sections.
- **Responsive:** cards stack on mobile; child switcher wraps.
- **Accessibility:** existing read-only live-region banner retained; new cards have headings in the
  landmark outline.
- **Copy & i18n:** all keys under the `parent` namespace (W01).

## 11. AI / ML Considerations

- Not AI-touching. (The report-card *comment* generation is a separate instructor flow — see W04.)

## 12. Integration Points

- `clients/web/src/pages/lms/parent/parent-dashboard.tsx`, `clients/web/src/lib/parent-api.ts`.
- Server parent handlers/repos; attendance (13.2), behavior (13.3), report cards (13.4), messaging/inbox.
- Conference scheduling link (13.12) already present.

## 13. Dependencies & Sequencing

- **Must ship after:** 13.1–13.4 (completed); coordinate with W01 for the `parent` namespace.
- **Must ship before:** K-12 district GA marketing of the parent portal.
- **Shared infra:** email (message notifications), object storage (report-card PDFs).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Guardian-link authz gap (IDOR) | M | H | Server authz keyed on link; add authz matrix test (AC-5). |
| Behavior data over-disclosure to parents | M | H | Respect district toggle; default hidden; legal review of visible fields. |
| Grade title join is expensive on the client | M | M | Prefer server-side title enrichment (FR-2 option A). |
| Scope creep into a full parent app | M | M | Non-goals fixed; link out to conferences/inbox rather than rebuild. |

## 15. Rollout Plan

- **Feature flag:** `ffParentPortalV2` (default off) gating the new sections; grade-legibility fix ships
  unflagged (it is a bug fix).
- **Sequencing:** ship grade-title fix → attendance → report cards → behavior → message-teacher → flip
  flag for pilot → GA.
- **Pilot:** one elementary + one secondary K-12 site.
- **GA criteria:** all ACs pass; authz matrix green; localized `es` bundle complete.
- **Rollback:** flag off reverts to the current single-page portal; the grade-title fix stays.

## 16. Test Plan

- **Unit** — grade-row rendering shows title; per-section error isolation.
- **Integration** — attendance/behavior/report-card aggregates; guardian-link authz.
- **End-to-end** — Playwright: multi-child switch; grade legibility; report-card open; message teacher.
- **Security** — IDOR/authz matrix (linked vs unlinked student; suspended link).
- **Accessibility** — axe on the expanded page; screen-reader pass on the read-only banner + cards.
- **Performance** — aggregate load under budget with a 6-course child.
- **Manual exploratory** — parent-of-two, no-data child, behavior-disabled org.

## 17. Documentation & Training

- Help center: "Using the family dashboard" (parent-facing, localized).
- Admin docs: how guardian links + behavior visibility toggles work.
- API reference: new parent endpoints.
- Runbook: triaging `parent_grades_missing_title` and authz-denial alerts.

## 18. Open Questions

1. Which behavior/PBIS fields are guardian-viewable by default vs. district-configurable?
2. Do we enrich grade titles server-side (preferred) or join on the client from the assignments list?
3. Should "Message teacher" reuse the existing Inbox compose or a scoped parent-message endpoint?

## 19. References

- `clients/web/src/pages/lms/parent/parent-dashboard.tsx` (grades render `itemId.slice(0,8)…` at ~line 196).
- `clients/web/src/lib/parent-api.ts` (`ParentCourseGradesRow.grades: Record<string,string>`).
- Related plans: [13.1](../../completed/13-k12-specific/13.1-parent-portal.md),
  [13.2](../../completed/13-k12-specific/13.2-daily-attendance.md),
  [13.3](../../completed/13-k12-specific/13.3-behavior-pbis-tracking.md),
  [13.4](../../completed/13-k12-specific/13.4-report-cards.md),
  [13.12](../../completed/13-k12-specific/13.12-parent-teacher-conference-scheduling.md),
  [W01](W01-i18n-application-coverage.md), [W05](W05-human-readable-entity-labels.md).
- Standards: FERPA guardian access; COPPA.
