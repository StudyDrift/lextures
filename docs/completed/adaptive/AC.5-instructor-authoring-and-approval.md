# AC.5 — Instructor Authoring & Human-in-the-Loop Approval

> Implementation plan. Source: authoring surface for ACE. Folder overview: [README](../../plan/adaptive/README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.5 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Frontend team + backend platform |
| **Depends on** | AC.1, AC.3 (preview), AC.2 (pre-assessment picker) |
| **Unblocks** | AC.6 (only approved variants serve when approval required) |

---

## 1. Problem Statement

The engine can generate fidelity-checked variants, but instructors still need a control plane: to configure *how* a unit adapts, set guardrails (key terms, allowed axes, minimum fidelity), **preview** what each learner archetype will see, and **review, revoke, edit, or (where required) approve** variants. By default variants auto-serve once they clear the fidelity/safety/a11y gates (the automated trust mechanism), so review here is primarily *post-hoc* — but any instructor or org can switch a unit/course to require sign-off first, and AC.8 forces that on for high-risk/minor contexts. This story delivers the human-oversight surface (pre- or post-hoc) that keeps ACE trustworthy and legally defensible.

## 2. Goals

- One coherent "Adaptive Content" workspace in the course editor to create/configure units and set guardrails.
- Preview variants for real and synthetic learner profiles, with a base-vs-variant diff and fidelity/a11y badges (reusing AC.3's preview endpoint).
- A review queue where instructors approve / edit-and-approve / reject generated variants; approval is required by default before serving.
- Let instructors mark **key terms** that must survive rewrites and pick allowed adaptation axes per unit.
- Full audit trail of every configuration and approval decision.

## 3. Non-Goals

- The generation engine, fidelity gate, or prompt (AC.3).
- The async pipeline/budget internals (AC.4) — this UI *surfaces* budget/pre-warm but does not implement them.
- Student-facing runtime (AC.6).
- Effectiveness dashboards (AC.9) — though the workspace links to them.

## 4. Personas & User Stories

- **As an instructor**, I want to set up an adaptive unit in a few clicks: pick the content page, the pre-check, the axes, and go.
- **As an instructor**, I want to see exactly what a struggling student vs. an advanced student will read, side-by-side with my original.
- **As a cautious instructor**, I want every AI variant to wait for my approval before any student sees it.
- **As a department lead**, I want to lock certain terminology so no rewrite ever changes it.
- **As a TA**, I want to help review the queue but not change course-wide settings (role-scoped).

## 5. Functional Requirements

- **FR-1.** The system MUST provide an "Adaptive Content" section in the course editor listing units with status, coverage, and effectiveness (deep-linking AC.9).
- **FR-2.** The unit editor MUST let instructors set: base content page, pre/post assessment (AC.2), allowed axes, `min_fidelity`, key terms, trigger mode, and `require_instructor_approval`.
- **FR-3.** The preview MUST call AC.3's preview endpoint for a chosen emphasis mode / synthetic profile OR a real (anonymized) cohort signature, rendering base-vs-variant with an inline diff and badges (`fidelity`, `a11y`, `safety`, token cost).
- **FR-4.** The system MUST provide a **review queue** of `pending_review` variants; an instructor MAY: **approve** (→ `approved`, servable), **edit-and-approve** (persist an instructor-edited variant, marked human-edited, → `approved`), or **reject** (→ `rejected`, base served) with an optional reason.
- **FR-5.** When `require_instructor_approval=false` (**default**), a variant that passes the fidelity + safety + a11y gate MAY `auto_serve` immediately — fully logged, revocable, and subject to post-hoc review; when `true` (instructor/org opt-in, or forced by AC.8 for high-risk/minor contexts), a variant MUST NOT serve until an instructor `approves` it.
- **FR-6.** The system MUST let an instructor **revoke** an approved variant (→ `superseded`), instantly reverting affected students to base.
- **FR-7.** The system MUST support **bulk** approve/reject across a unit's pending variants, with a confirm step showing aggregate fidelity/a11y warnings.
- **FR-8.** All actions MUST write `adaptive_content_events` (config change, approve, edit, reject, revoke) with actor and before/after.
- **FR-9.** Permissions: unit config + approval require `course:{code}:item:create`; a TA-review-only capability MAY approve/reject but not change course settings or budgets.
- **FR-10.** The workspace MUST surface AC.4 budget state and a "pre-warm now" action, and warn before enabling a unit whose budget is exhausted.

## 6. Non-Functional Requirements

- **Performance** — Workspace list p95 ≤ 200 ms; preview inherits AC.3 (≤ 8 s) with a clear loading state; review-queue pagination for large cohorts.
- **Security** — Server re-checks permissions on every mutation; instructor edits are sanitized through the same block-editor allow-list as base content. Approval cannot bypass the fidelity/safety gate — a rejected-by-gate variant cannot be force-approved unless the instructor explicitly overrides *and* the override is audited (and blocked entirely for `must_appear` key-term failures).
- **Privacy & Compliance** — Preview against real cohorts uses anonymized signatures (no student identity). Human oversight here is the control that satisfies GDPR Art. 22 "meaningful human involvement" and EU AI Act human-oversight (S13); approvals are the evidence trail (S06, S21).
- **Accessibility** — Diff view is not color-only (add/remove marked with icons + text); side-by-side collapses responsively; all controls keyboard-operable; focus management on modal open/close; WCAG 2.1 AA.
- **Scalability** — Review queue handles hundreds of variants via pagination + bulk actions; diff computed client-side on demand.
- **Reliability** — Optimistic UI with server confirmation; approval writes are transactional; concurrent edits detected (variant version check) to avoid clobbering another reviewer.
- **Observability** — Counters `adaptive_content.approved`, `.rejected`, `.edited`, `.revoked`, `.auto_served`; time-in-queue histogram.
- **Maintainability** — Reuses the existing block editor and the `outcome-links-editor` / features-form component patterns; new components under `clients/web/src/components/lms/adaptive-content/`.
- **Internationalization** — All labels/help via i18n keys; diff works for RTL.
- **Backward compatibility** — Absent config, a unit stays `draft` and serves base; existing content editing is unaffected.

## 7. Acceptance Criteria

- **AC-1.** *Given* an instructor opens the Adaptive Content workspace, *When* they create a unit, pick a content page + pre-check + axes, *Then* the unit persists as `draft` and appears in the list.
- **AC-2.** *Given* a unit, *When* the instructor previews "as a remediate learner", *Then* a variant renders beside the base with a fidelity badge and an add/remove diff.
- **AC-3.** *Given* `require_instructor_approval=true` and a `pending_review` variant, *When* a student would be served, *Then* base content is served until the instructor approves.
- **AC-4.** *Given* a pending variant, *When* the instructor edits a sentence and approves, *Then* the stored variant reflects the edit, is marked human-edited, and becomes servable.
- **AC-5.** *Given* an approved variant, *When* the instructor revokes it, *Then* affected students immediately revert to base and an audit event is written.
- **AC-6.** *Given* a variant that failed a `must_appear` key-term check, *When* the instructor tries to approve it, *Then* approval is blocked with an explanation (cannot override a hard failure).
- **AC-7.** *Given* a TA with review-only capability, *When* they open the workspace, *Then* they can approve/reject but cannot change axes, budget, or the flag.
- **AC-8.** *Given* a unit whose course budget is exhausted, *When* the instructor activates it, *Then* they see a warning that students will see the original until budget resets.

## 8. Data Model

Reserves `443_adaptive_content_authoring.sql`. Mostly reuses AC.1/AC.3 tables; adds review metadata + a review-only capability.

```sql
-- 443_adaptive_content_authoring.sql
ALTER TABLE course.content_variants
    ADD COLUMN IF NOT EXISTS human_edited BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS reviewed_by UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS review_note TEXT,
    ADD COLUMN IF NOT EXISTS variant_version INTEGER NOT NULL DEFAULT 1;  -- optimistic concurrency

-- Review-only capability for TAs.
INSERT INTO "user".permissions (permission_string, description)
VALUES ('course:adaptive_content:review',
        'Approve or reject adaptive content variants without changing course-level adaptive settings.')
ON CONFLICT (permission_string) DO NOTHING;
```

**Backfill:** none.

## 9. API Surface

```
GET   /api/v1/courses/{course_code}/adaptive-content/units                         instructor
POST  /api/v1/courses/{course_code}/adaptive-content/units                         instructor
PATCH /api/v1/courses/{course_code}/adaptive-content/units/{id}                    instructor
GET   /api/v1/courses/{course_code}/adaptive-content/units/{id}/key-terms          instructor
PUT   /api/v1/courses/{course_code}/adaptive-content/units/{id}/key-terms          instructor
POST  /api/v1/courses/{course_code}/adaptive-content/units/{id}/variants/preview   instructor  (AC.3)
GET   /api/v1/courses/{course_code}/adaptive-content/units/{id}/variants           instructor|reviewer
POST  /api/v1/courses/{course_code}/adaptive-content/variants/{vid}/approve        instructor|reviewer
POST  /api/v1/courses/{course_code}/adaptive-content/variants/{vid}/reject         instructor|reviewer
PUT   /api/v1/courses/{course_code}/adaptive-content/variants/{vid}                instructor|reviewer  (edit-and-approve)
POST  /api/v1/courses/{course_code}/adaptive-content/variants/{vid}/revoke         instructor
POST  /api/v1/courses/{course_code}/adaptive-content/units/{id}/variants/bulk      instructor|reviewer  ({action, variantIds[]})
```

Approve/reject/edit include `expectedVariantVersion` for optimistic concurrency (409 on mismatch).

## 10. UI / UX

**Adaptive Content workspace** (new tab in the course editor, `clients/web/src/pages/lms/`):
1. **Units list** — table: unit, target (module/outcome), content page, status, coverage (# variants, # approved), effectiveness (lift chip → AC.9), budget chip.
2. **Unit editor drawer** — content-page picker, pre/post assessment pickers (AC.2), axis checkboxes with descriptions, `min_fidelity` slider, key-terms tag input, trigger mode, approval toggle.
3. **Preview** — "Preview as…" selector (emphasis archetypes or a real anonymized signature) → base | variant split with inline diff, badges, token cost, "regenerate" and "approve/edit/reject" actions.
4. **Review queue** — filterable list of `pending_review` variants across units; bulk approve/reject; per-variant fidelity/a11y/safety flags surfaced up front.

- **Empty state:** "Turn on Adaptive Content and add your first unit."
- **Loading:** skeletons for list; spinner + "checking fidelity…" for preview.
- **Error:** generation failure → "Couldn't generate — students see the original"; budget exhausted → amber banner.
- **Mobile:** split preview stacks; diff toggles between base/variant; review queue is a card list.
- **Accessibility:** diff uses icon+text markers; modals trap focus and restore it; live region announces approve/reject results; sliders have text inputs.

## 11. AI / ML Considerations

No new model surface. This story is the **human-oversight** layer over AC.3's generation. Instructor edits create a `human_edited` variant that is *not* re-generated but *is* re-run through the fidelity/safety/a11y gates before it can serve (so a human can't accidentally re-introduce a hallucination or break a key term either).

## 12. Integration Points

- `clients/web/src/pages/lms/course-editor` + new `components/lms/adaptive-content/*` (reuse block editor, `outcome-links-editor` patterns from commit #530).
- `clients/web/src/lib/adaptive-content-api.ts` (new) + `courses-api-schemas.ts` additions.
- `server/internal/httpserver/adaptive_content_units.go`, `adaptive_content_variants.go` (new); reuse `requireCourseItemCreate`.
- `server/internal/courseroles/` — add the `course:adaptive_content:review` capability wiring.
- `server/migrations/443_adaptive_content_authoring.sql` (+ down).
- Related: [AC.3](AC.3-content-generation-engine.md) (preview/generate), [AC.6](AC.6-student-runtime-and-transparency.md) (serves approved), [AC.9](../../plan/adaptive/AC.9-analytics-reporting-and-operability.md) (effectiveness links).

## 13. Dependencies & Sequencing

- **Must ship after:** AC.1 (units/settings), AC.3 (preview + gate), AC.2 (pre-assessment picker).
- **Must ship before:** AC.6 in `require_instructor_approval` mode (nothing serves until approved).
- **Shared infra:** block editor, RBAC, i18n.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Review queue overwhelms instructors (too many variants) | M | H | Auto-serve-after-gates is the default (no queue unless approval is opted in); signature caching bounds variants/unit (~12); bulk actions when approval is required |
| Instructor override re-introduces bad content | M | M | Human edits re-run the gate; hard key-term failures un-overridable |
| Concurrent reviewers clobber edits | M | M | Optimistic `variant_version` + 409 |
| Config complexity deters adoption | M | M | Sensible defaults; "quick setup" that picks axes automatically; progressive disclosure |
| TA overreach | L | M | Scoped `review` capability separate from course settings |

## 15. Rollout Plan

- **Feature flag:** course `adaptive_content_enabled` (AC.1); `require_instructor_approval` default **false** (auto-serve after gates), opt-in per unit/course, forced true by AC.8 for high-risk/minor contexts.
- **Sequencing:** deploy migration → ship workspace (list/preview + post-hoc review, revoke) → enable auto-serve of gate-passing variants (AC.6) → verify the opt-in approval-required path also serves correctly.
- **Pilot cohort:** the AC.3 pilot instructors, who now approve real variants for their classes.
- **GA criteria:** instructors can configure a unit, preview all four modes, and approve/reject with audit; usability tested with ≥ 3 instructors; a11y audit passes on the workspace.
- **Rollback:** disable the workspace tab via flag; pending variants simply never serve (base remains).

## 16. Test Plan

- **Unit** — reducer/state for the workspace; diff renderer; permission gating of controls; optimistic-version conflict handling.
- **Integration** — create unit → preview → approve → variant becomes servable; edit-and-approve persists human edit + re-gates; revoke reverts; bulk approve; key-term hard-fail blocks approval.
- **End-to-end** — Playwright: full instructor journey create→preview→approve; TA review-only cannot edit settings; budget-exhausted warning shows.
- **Security** — authz matrix (instructor vs TA-review vs student); server re-validates; sanitize instructor edits; cannot override hard failures.
- **Accessibility** — axe on workspace + modals; keyboard-only approve/reject; diff not color-only; focus restore.
- **Performance** — list p95 ≤ 200 ms; queue pagination under 500 variants.
- **Manual exploratory** — attempt to approve a gate-failed variant; concurrent reviewer edit; revoke mid-class.

## 17. Documentation & Training

- Instructor guide + short video: "Set up, preview, and approve adaptive content."
- "Reviewer" quick-start for TAs.
- Compliance note: how approval discharges human-oversight obligations (link S13/S06).
- Help center: "Why do my students see the original until I approve?"

## 18. Open Questions

1. Auto-serve-after-gates is the platform default; should some orgs/markets (e.g., stricter EU deployments) get an org-level default of approval-required that inverts it? (AC.8 already forces approval for high-risk/minor; open whether blanket per-org approval is wanted.)
2. Do we allow per-axis approval (approve reading-level but not restructuring)? (v1: whole-variant approval; revisit.)
3. Should department/org admins be able to pre-approve vetted prompt templates for instructors? (Coordinate with AC.8 governance.)
4. How do we present effectiveness (AC.9 lift) inline to inform approval before enough data exists? (Show "insufficient data" until threshold.)

## 19. References

- Existing files: `clients/web/src/components/outcomes/outcome-links-editor.tsx`, `clients/web/src/components/lms/build-content-page-with-ai-modal.tsx` (commit #530), `server/internal/courseroles/`, `server/internal/httpserver/course_features.go`.
- Related plans: [AC.3](AC.3-content-generation-engine.md), [AC.6](AC.6-student-runtime-and-transparency.md), [AC.8](../../plan/adaptive/AC.8-governance-safety-fairness-privacy.md), `../standards/S13-eu-ai-act-high-risk.md`.
- External: EU AI Act Art. 14 (human oversight); GDPR Art. 22(3) (human intervention).
