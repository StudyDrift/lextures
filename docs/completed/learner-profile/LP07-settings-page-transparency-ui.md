# LP07 — Learner Profile Page (User Settings) + Transparency UI

> Implementation plan. The learner-facing surface for the profile built by LP01–LP06. Follows
> [../_TEMPLATE.md](../_TEMPLATE.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LP07 |
| **Section** | Learner Profile |
| **Severity** | BLOCKER (the profile is invisible without it) |
| **Markets** | K12 / HE / SL |
| **Status (today)** | COMPLETED |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Web frontend team |
| **Depends on** | LP01 (read API) + ≥1 facet (LP02–LP06); GA needs LP08 |
| **Unblocks** | LP10 (mobile mirrors this), user trust in the adaptive claim |

---

## 1. Problem Statement

The learner profile is built autonomously by LP01–LP06, but a profile the learner can't see fails the
product's core promise ("adapts to every student") and the transparency constraint. This plan adds a
**User Settings → Learner Profile** page that renders every facet in plain language, and — crucially —
makes it **transparent**: each insight shows *where it came from* ("derived from 12 quiz attempts
across 3 courses"), when it was last updated, and how confident it is. This is the page that turns an
opaque algorithm into something a learner trusts.

## 2. Goals

- Add a **Learner Profile** entry under the existing "User Settings" nav group (alongside Account,
  Notifications, Integrations, Privacy Center).
- Render each facet (LP02–LP06) as its own clearly-labelled section with plain-language insights.
- For **every insight**, show provenance ("Derived from …"), last-computed time, and a confidence
  indicator — the transparency guarantee, front and centre.
- Handle empty / still-building / insufficient-data / paused states gracefully.
- Meet WCAG 2.1 AA (this epic's first real UI).

## 3. Non-Goals

- Any editing of profile data (it is autonomous — read-only by design).
- Privacy *controls* (pause/reset/export) — those live in LP08 but this page hosts the entry points.
- Mobile UI (LP10). Instructor/guardian views (out of scope for this epic).

## 4. Personas & User Stories

- **As a student**, I want to open Settings → Learner Profile and understand, in plain language, what
  the platform has learned about how I learn, and exactly where each conclusion came from.
- **As a skeptical learner**, I want to click "why do you think this?" on any insight and see the
  evidence, so I trust it (or dismiss it) on the merits.
- **As a parent/self-learner**, I want to see the profile is still building when there isn't enough
  data yet, rather than a blank or a wrong guess.

## 5. Functional Requirements

- **FR-1.** Add route `/settings/learner-profile` and a "Learner Profile" nav link in the **User
  Settings** group of `side-nav-settings-links.tsx`; extend `SettingsNavView` +
  `settingsViewFromPathname` (`side-nav-path-utils.ts`) with `learner-profile`. Gated by
  `learner_profile_enabled`.
- **FR-2.** The page MUST fetch `GET /api/v1/me/learner-profile` (LP01) and render one section per
  returned facet, ordered by a stable priority, each with a plain-language heading and summary.
- **FR-3.** Each insight MUST display a **provenance affordance**: an always-visible "Derived from
  {sources}" line and an expandable evidence panel listing `sourceKind`, counts, courses, and time
  window from `.../facets/{key}/evidence`.
- **FR-4.** Each facet MUST show **last computed** time and a **confidence** indicator (text + icon,
  not color alone).
- **FR-5.** The page MUST handle: whole-profile `insufficient_data` (a friendly "your profile is still
  building — keep learning" empty state), per-facet `insufficient_data`, `paused` (LP08) state, and
  fetch error/loading states.
- **FR-6.** The page MUST host entry points for LP08 controls (Download, Pause, Reset) without
  implementing them (LP08 owns behaviour) — a clearly labelled "Manage your profile" area.
- **FR-7.** The page MUST include a short, honest **"How this works"** explainer: autonomous, derived
  from your activity, never entered manually, learner-owned.
- **FR-8.** All copy MUST be i18n keys; no hard-coded strings.

## 6. Non-Functional Requirements

- **Performance** — First meaningful render ≤ 1 s p95 on the LP01 read (single request); evidence
  panels lazy-load on expand. No layout shift when facets load.
- **Security** — Self-only; page reads only the authenticated user's profile. No profile data in URL.
- **Privacy & Compliance** — Displays the FERPA/GDPR notice + link to Privacy Center. No profile data
  cached in analytics/telemetry client-side.
- **Accessibility** — WCAG 2.1 AA: logical heading order, keyboard-operable evidence disclosures
  (`aria-expanded`), charts have text-table alternatives, confidence not by color alone, focus mgmt.
- **Scalability** — Pure read UI; no heavy client compute.
- **Reliability** — Degrades to a usable page if a single facet errors (render the rest).
- **Observability** — Client emits a page-view + "evidence expanded" interaction event (no PII).
- **Internationalization** — Full i18n; RTL-safe; localised dates/numbers.
- **Backward compatibility** — Additive route/nav; no change to existing settings pages.

## 7. Acceptance Criteria

- **AC-1.** *Given* the flag on and a populated profile, *when* the learner opens
  `/settings/learner-profile`, *then* each facet renders as a section with a plain-language summary.
- **AC-2.** *Given* any insight, *when* the learner expands its evidence, *then* they see the sources,
  counts, courses, and time window that produced it.
- **AC-3.** *Given* a brand-new learner, *then* the page shows the "still building" empty state, not a
  blank or a fabricated insight.
- **AC-4.** *Given* a paused profile (LP08), *then* the page shows the paused state and a resume entry.
- **AC-5.** *Given* an axe scan and keyboard-only navigation, *then* no critical a11y violations and
  all evidence disclosures are operable.
- **AC-6.** *Given* the flag off, *then* the nav link and route are absent.

## 8. Data Model

None — read-only consumer of LP01's tables/API.

## 9. API Surface

Consumes LP01: `GET /api/v1/me/learner-profile`, `.../facets/{key}`, `.../facets/{key}/evidence`. No
new endpoints (control endpoints are LP08).

## 10. UI / UX

- New page `clients/web/src/pages/lms/learner-profile.tsx` (or a settings panel
  `components/settings/learner-profile-panel.tsx` rendered by `settings.tsx`), matching the existing
  settings-panel visual system (`settings-section.tsx`).
- Layout: intro "How this works" → facet sections (Study rhythm, How you like to learn, Strengths &
  growth, What you're drawn to, How you approach challenges) → "Manage your profile" (LP08 entry).
- Each facet section: heading, plain-language summary, insight rows; each insight row has a
  "Derived from …" line + expandable evidence; facet footer shows last-computed + confidence.
- States: loading skeleton, whole-profile empty ("still building"), per-facet insufficient, paused,
  error-per-facet.
- Charts (rhythm heatmap, modality bars) each ship a text-table alternative.
- i18n keys: `learnerProfile.title`, `learnerProfile.howItWorks`, `learnerProfile.facet.*`,
  `learnerProfile.evidence.derivedFrom`, `learnerProfile.confidence.*`, `learnerProfile.empty.*`.

## 11. AI / ML Considerations

No inference in the UI. If LP09 later adds an LLM-written summary, it MUST be clearly labelled as
AI-generated (reuse the platform AI-disclosure pattern, `ai_disclosure_http.go`) and sit *above*, not
*replacing*, the evidence-backed facets.

## 12. Integration Points

- `clients/web/src/components/layout/side-nav-settings-links.tsx` (nav link, User Settings group).
- `clients/web/src/components/layout/side-nav-path-utils.ts` (`SettingsNavView` + resolver).
- `clients/web/src/pages/lms/settings.tsx` / `app.tsx` route (`/settings/:tab`).
- `clients/web/src/components/settings/settings-section.tsx` (visual system).
- `clients/web/src/lib/platform-features.ts` (flag). New API client `lib/learner-profile-api.ts`.

## 13. Dependencies & Sequencing

- After LP01 + ≥1 facet. Can ship progressively (sections appear as facets land). GA requires LP08
  controls present. Precedes LP10.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Insights feel creepy without context | M | H | Provenance-first design; "How this works"; LP08 controls visible |
| Charts fail a11y | M | H | Text-table alternatives mandatory; axe in CI; keyboard disclosures |
| Facet sprawl overwhelms | M | M | Prioritised order, collapsible sections, top-N insights per facet (LP04/LP06 caps) |
| Empty state reads as "broken" | M | M | Explicit "still building" copy tied to `insufficient_data` |

## 15. Rollout Plan

Behind `learner_profile_enabled`. Ship nav + page → dogfood internal → pilot cohort → GA once LP08
lands. Rollback: flag hides nav + route.

## 16. Test Plan

- **Unit** — facet/insight rendering; evidence expand; state machine (empty/insufficient/paused/error).
- **Integration** — mock LP01 responses for each state; nav gating by flag.
- **E2E (Playwright)** — open page with seeded profile; expand evidence; verify provenance text;
  empty-state path; paused path.
- **Accessibility** — axe scan (zero critical); keyboard-only walkthrough; screen-reader script for
  evidence disclosures and chart alternatives.
- **Performance** — first render ≤ 1 s p95; evidence lazy-load.

## 17. Documentation & Training

- Student help-center: "Your Learner Profile — what it is, where it comes from, how to manage it."
- Screenshot walkthrough of provenance affordances.

## 18. Open Questions

1. Standalone page vs. a panel inside the existing `settings.tsx` switch — match whichever the newest
   User Settings panels use.
2. Default facet order and which (if any, e.g. LP06) start collapsed.
3. Should "How this works" link to a public trust page (docs/trust, plan 20.x)?

## 19. References

- [LP01](LP01-foundation-derivation-engine.md), [LP08](LP08-privacy-consent-controls.md).
- Existing: `side-nav-settings-links.tsx`, `side-nav-path-utils.ts`, `pages/lms/settings.tsx`,
  `components/settings/settings-section.tsx`, `ai_disclosure_http.go`.
- External: WCAG 2.1 AA; GDPR Art. 13–15 (transparency/access).
