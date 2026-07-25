# AC.6 — Student Runtime Experience & Transparency

> Implementation plan. Source: student-facing serving layer for ACE. Folder overview: [README](README.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | AC.6 |
| **Section** | Adaptive Content Engine (ACE) |
| **Severity** | BLOCKER |
| **Markets** | K12 / HE / HS |
| **Status (today)** | DONE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Frontend team + backend platform |
| **Depends on** | AC.2 (profile), AC.3 (variant), AC.4 (cache serving), AC.5 (approval) |
| **Unblocks** | AC.7 (needs a serving record to attribute lift) |

---

## 1. Problem Statement

Everything upstream produces an approved, cached, fidelity-checked variant keyed to a learner's profile — but the student still sees the static page until we actually *serve* the variant, transparently and accessibly. This story is where a learner feels "the environment that adapts": they take the entry ticket, and the content page they open is the one written for them — clearly labeled as adapted, with a one-tap way to see the original, and an opt-out. It also writes the **serving record** (including holdout/control assignment) that AC.7 needs to prove the adaptation worked.

## 2. Goals

- Serve the approved variant matching the student's profile signature for a unit; fall back to base on any miss/error/opt-out — never block or slow the page.
- Make adaptation **transparent**: label adapted sections and offer "view original".
- Assign and honor **holdout/control** membership so effectiveness can be measured causally (AC.7).
- Provide a student **opt-out** of adaptive content (when the course allows it).
- Record a `adaptation_servings` row for every content view under an active unit.

## 3. Non-Goals

- Generating variants (AC.3) or scheduling them (AC.4).
- Computing lift (AC.7) — this story only records the serving + holdout assignment.
- Instructor tools (AC.5) or dashboards (AC.9).
- Changing the entry-ticket quiz mechanics (reuses the quiz runner; AC.2 owns binding).

## 4. Personas & User Stories

- **As a student**, when I open a chapter after my entry ticket, I want it already tailored to me, loading instantly.
- **As a student**, I want to know the page was adapted for me and be able to read the original if I prefer.
- **As a privacy-conscious student (or my guardian)**, I want to opt out of AI-adapted content and just get the standard material.
- **As a student in a control group**, I should get the standard content (unknowingly) so the school can measure whether adaptation helps — but my transparency/opt-out rights are unchanged.
- **As a student on a phone / spotty connection**, I want the adapted or original page to work offline-ish and degrade gracefully.

## 5. Functional Requirements

- **FR-1.** When a student opens a content page that is a unit's `base_content_item_id` and the unit is `active`, the system MUST resolve serving: compute/lookup the student's profile (AC.2) → find an `approved`/`auto_served` variant for `(unit, signature, current content_version)` → serve it; else serve base.
- **FR-2.** The system MUST write exactly one `adaptation_servings` row per (enrollment, unit, content_version) exposure, recording `variant_id` (or null for base), `profile_id`, `was_holdout`, `was_fallback`, and `served_at`; re-opening the same version updates `served_at`/view-count, not a new row.
- **FR-3.** The system MUST assign holdout membership deterministically per (unit, enrollment) using the unit/course `holdout_percent` (stable hash of enrollment+unit), so a student's group is consistent across visits; holdout students are served base and flagged `was_holdout=true`.
- **FR-4.** Adapted content MUST render with a visible, dismissible **"Adapted for you"** indicator and a **"View original"** toggle that swaps to base content client-side without another server round-trip.
- **FR-5.** When `student_optout_allowed` (AC.1) is true, the system MUST offer a per-course student setting "Use standard (non-adapted) content"; opting out serves base with `was_fallback=true` and suppresses the indicator, and MUST be honored on every subsequent view.
- **FR-6.** Serving MUST be non-blocking: if the profile isn't ready, the variant isn't cached, the gateway denies, or any error occurs, the student gets base content immediately and a generation job may be enqueued (AC.4) for next time.
- **FR-7.** For minors / COPPA-gated users, if `aigateway` would deny AI features, the student MUST transparently receive base content (no adapted view, no indicator), consistent with S08.
- **FR-8.** The system MUST expose the serving decision to the student UI via the content-page fetch response (`servedVariantId`, `isAdapted`, `axesApplied`, `canViewOriginal`, `optedOut`) so the client renders correctly in one request.
- **FR-9.** The system MUST announce adaptation to assistive tech (e.g., "This section has been adapted to your progress") via an ARIA live region on first render.

## 6. Non-Functional Requirements

- **Performance** — Adapted-page load adds ≤ 30 ms over a normal content-page fetch (single indexed variant lookup; no model call on the hot path). "View original" toggle is instant (base shipped alongside or lazily fetched once).
- **Security** — A student can only be served variants for units in courses they're enrolled in; profile resolution is server-side; the client cannot request an arbitrary signature/variant. Opt-out is per-user and server-enforced.
- **Privacy & Compliance** — Serving records are education records (FERPA); holdout assignment carries no sensitive attribute (hash of ids only). Transparency + opt-out satisfy AI-disclosure (10.17) and AI-Act transparency (S13). Guardian visibility of opt-out via the parent portal where applicable.
- **Accessibility** — Indicator and toggle are keyboard-operable, labeled, not color-only; adapted content passes the same a11y bar as base (heading order, contrast); live-region announcement respects reduced-motion; "View original" preserves scroll position and focus.
- **Scalability** — Hot-path is a cache read; holdout is a pure function; serving-record write is a single upsert. Handles class-open spikes via AC.4 pre-warm.
- **Reliability** — Base content is the always-available fallback; a serving-record write failure MUST NOT block rendering (fire-and-forget with retry/log). Content-version mismatch ⇒ serve base until a fresh variant exists.
- **Observability** — Counters `adaptive_content.served_variant`, `.served_base`, `.served_holdout`, `.served_fallback`, `.view_original_clicks`, `.optout`; histogram serve latency.
- **Maintainability** — Serving resolution in `service/adaptivecontent/serve.go`; the content-page handler gains a thin resolve call; the web content-page component gains an adapted-indicator subcomponent.
- **Internationalization** — Indicator/toggle strings via i18n; adapted content in the base language; RTL preserved.
- **Backward compatibility** — Content pages not under an active unit render exactly as today; the response fields are additive/optional.

## 7. Acceptance Criteria

- **AC-1.** *Given* an approved variant for a student's signature, *When* they open the unit's content page, *Then* the variant renders with an "Adapted for you" indicator and a serving row is written with that `variant_id`.
- **AC-2.** *Given* no cached variant yet, *When* the student opens the page, *Then* base content renders immediately, the serving row has `was_fallback=true`, and a generation job is enqueued.
- **AC-3.** *Given* `holdout_percent=20`, *When* many students open the unit, *Then* ~20% are deterministically assigned holdout, served base, and flagged `was_holdout=true`, stably across their revisits.
- **AC-4.** *Given* an adapted page, *When* the student clicks "View original", *Then* base content shows instantly with focus/scroll preserved and no new server call (or one cached fetch).
- **AC-5.** *Given* the course allows opt-out, *When* a student opts out, *Then* all their subsequent unit views serve base (no indicator) and this persists across sessions.
- **AC-6.** *Given* a COPPA-gated minor whose AI features are denied, *When* they open the unit, *Then* they see base content with no adapted indicator and a base serving record.
- **AC-7.** *Given* the content page is edited (content_version bump) and no fresh variant exists, *When* a student opens it, *Then* base is served (not a stale variant) and regeneration is enqueued.

## 8. Data Model

Reserves `444_adaptive_content_serving.sql`. Extends `adaptation_servings` (AC.1) + adds a student opt-out and view accounting.

```sql
-- 444_adaptive_content_serving.sql
ALTER TABLE course.adaptation_servings
    ADD COLUMN IF NOT EXISTS content_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS view_count INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS first_viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS view_original_clicks INTEGER NOT NULL DEFAULT 0;

-- One serving row per exposure of a content version.
CREATE UNIQUE INDEX IF NOT EXISTS ux_ac_servings_exposure
    ON course.adaptation_servings (unit_id, enrollment_id, content_version);

-- Per-student, per-course opt-out of adaptive content.
CREATE TABLE course.adaptive_content_optouts (
    course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES "user".users (id) ON DELETE CASCADE,
    opted_out BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (course_id, user_id)
);
```

**Backfill:** none.

## 9. API Surface

```
GET  /api/v1/courses/{course_code}/content-pages/{item_id}   student
  -- existing route; response gains adaptive fields when the page is an active unit base:
  -- { markdown|blocks, adaptive?: { unitId, isAdapted, servedVariantId?, axesApplied[],
  --     canViewOriginal, optedOut, isHoldout } , originalMarkdown? (only if canViewOriginal) }

GET  /api/v1/courses/{course_code}/adaptive-content/optout    student  -> { optedOut }
PUT  /api/v1/courses/{course_code}/adaptive-content/optout    student  -> { optedOut }
POST /api/v1/courses/{course_code}/adaptive-content/units/{id}/viewed-original  student  (increments counter)
```

Serving resolution runs server-side inside the content-page GET so the client renders in a single request; `originalMarkdown` is included only when adapted so "View original" needs no extra call.

## 10. UI / UX

**Student content page** (`clients/web/src/pages/lms/course-module-content-page.tsx`):
1. **Entry-ticket gate (from AC.2):** if the unit has a pre-assessment the student hasn't completed, route them to it first, then to the (now personalized) content.
2. **Adapted content** renders in the normal reader with a slim banner: *"✦ Adapted for you — [why, e.g. 'extra practice on fractions'] · View original"*.
3. **View original** toggles to base inline; the banner switches to *"Showing the original · View adapted"*.
4. **Settings link** *"Prefer standard content?"* opens the opt-out toggle.

- **Empty/first-visit:** "Personalizing this section…" interstitial (brief; only while resolving; times out to base fast).
- **Loading:** content skeleton; never a spinner tied to a model call on the hot path.
- **Error/offline:** base content served; banner hidden; a retry happens silently next visit.
- **Mobile:** banner collapses to an icon + tap-to-expand; "View original" is a bottom-sheet toggle.
- **Accessibility:** banner is a labeled region; toggle is a real button with `aria-pressed`; first render announces adaptation via `role="status"`; reduced-motion disables the interstitial animation.

## 11. AI / ML Considerations

No model call on the student hot path — serving is a cache read plus a deterministic holdout function. If a variant is missing, AC.6 enqueues an AC.4 job and serves base *now*; the student may see the adapted version on a later visit. This keeps the learner experience fast and the AI cost bounded, and it means a provider outage never degrades the reading experience below "the normal page".

## 12. Integration Points

- `server/internal/httpserver/` content-page GET handler (module content page) — call `adaptivecontent.ResolveServing`.
- `server/internal/service/adaptivecontent/serve.go`, `holdout.go` (new).
- `server/internal/repos/adaptivecontent/servings.go`, `optouts.go` (new).
- `server/internal/service/aigateway/` — reuse deny signal for COPPA/opt-out ⇒ base.
- `clients/web/src/pages/lms/course-module-content-page.tsx` + new `components/lms/adaptive-content/adapted-banner.tsx`.
- `clients/web/src/lib/adaptive-content-api.ts` (opt-out, viewed-original).
- `server/migrations/444_adaptive_content_serving.sql` (+ down).
- Related: [AC.2](AC.2-pre-assessment-and-adaptation-profile.md), [AC.4](AC.4-generation-pipeline-caching-cost.md), [AC.7](AC.7-post-assessment-and-effectiveness.md).

## 13. Dependencies & Sequencing

- **Must ship after:** AC.2 (profile), AC.3 (variant), AC.4 (cache/serve latency), AC.5 (approval before serve).
- **Must ship before:** AC.7 (attributes lift to serving rows + holdout).
- **Shared infra:** content-page delivery, aigateway, i18n.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Students confused / distrust "the AI changed my lesson" | M | M | Clear transparency banner + always-available "View original" + opt-out |
| Holdout perceived as unfair (some kids get less help) | M | H | Small default holdout %, time-boxed experiments, instructor-configurable/zero; equity review in AC.8 |
| Stale variant served after content edit | M | M | `content_version` in the serving key; mismatch ⇒ base + regenerate |
| Serving-record write failures skew AC.7 | L | M | Fire-and-forget with retry/log; AC.7 tolerates missing rows |
| Interstitial adds perceived latency | M | L | Cap interstitial to a short timeout; skip entirely when variant is already cached |

## 15. Rollout Plan

- **Feature flag:** course `adaptive_content_enabled` (AC.1) + unit `active` + approved variant. Any layer off ⇒ base content.
- **Sequencing:** deploy migration → ship serving resolution returning base for everyone (records servings, no variants yet) → enable variant serving for approved units → enable holdout for measurement (AC.7).
- **Pilot cohort:** the AC.5 pilot courses' real students, starting with holdout=0 (everyone adapted) then introducing a holdout for measurement.
- **GA criteria:** hot-path adds ≤ 30 ms; transparency + opt-out verified; holdout stable and correctly proportioned; a11y audit passes; no learner-visible model waits.
- **Rollback:** unit `paused` / course flag off / kill-switch ⇒ instant base for all; opt-outs preserved.

## 16. Test Plan

- **Unit** — serving resolution decision table (variant present/absent, opted-out, holdout, COPPA-deny, version mismatch); deterministic holdout proportions & stability; serving upsert idempotency.
- **Integration** — content-page GET returns adapted payload with variant; base fallback + job enqueue on miss; opt-out persists; viewed-original increments; version bump serves base.
- **End-to-end** — Playwright: student completes entry ticket, sees adapted page + banner, toggles original, opts out, revisits and gets base.
- **Security** — cannot request arbitrary variant/signature; enrollment enforced; opt-out server-enforced.
- **Accessibility** — axe on adapted page + banner; keyboard toggle; `aria-pressed`; live-region announce; reduced-motion honored; focus/scroll preserved on toggle.
- **Performance / load** — k6 class-open spike; cache-hit serve p95 ≤ 30 ms over base.
- **Manual exploratory** — spotty-network base fallback; holdout fairness spot-check; guardian view of opt-out.

## 17. Documentation & Training

- Student help: "Why does my content look different? (Adapted for you)" + "How to turn off adaptive content."
- Guardian help: adaptive content, transparency, and opt-out for your child.
- Instructor note: what students see, the banner, holdout, and opt-out implications for coverage.

## 18. Open Questions

1. Should "View original" default-expanded for the first exposure to build trust, then collapse? (A/B candidate.)
2. Should opt-out be per-course or account-wide, and can guardians set it for minors? (Coordinate with AC.8/parent portal; lean per-course + guardian override for minors.)
3. Do we show the *reason* for adaptation to the student ("extra practice on X") or just that it happened? (Lean: short, non-judgmental reason; validate it doesn't feel labeling.)
4. How long should the personalizing interstitial wait before giving up to base? (Start ~800 ms; tune.)

## 19. References

- Existing files: `clients/web/src/pages/lms/course-module-content-page.tsx`, `server/migrations/021_module_content_pages.sql`, `server/internal/service/aigateway/service.go`.
- Related plans: [AC.2](AC.2-pre-assessment-and-adaptation-profile.md), [AC.4](AC.4-generation-pipeline-caching-cost.md), [AC.7](AC.7-post-assessment-and-effectiveness.md), [AC.8](../../plan/adaptive/AC.8-governance-safety-fairness-privacy.md), `../01-adaptive-learning-core/1.10-misconception-detection-remediation.md`.
- External: EU AI Act Art. 52 (transparency to users); FERPA (serving records as education record); WCAG 2.1 AA.
