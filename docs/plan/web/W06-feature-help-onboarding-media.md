# W06 — Feature-Help Onboarding Walkthrough Media

> Implementation plan. Source: web market-readiness scan (2026-07-06).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W06 |
| **Section** | Web / Onboarding & Help |
| **Severity** | MINOR |
| **Markets** | SL / K12 / HE |
| **Status (today)** | THIN |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Frontend platform team + Content/UX |
| **Depends on** | none |
| **Unblocks** | Credible in-product feature discovery |

---

## 1. Problem Statement

The in-product feature-help dock (`components/feature-help/feature-help-dock.tsx`) renders a prominent
16:9 region at the top of every help panel that ships a literal placeholder to end users: a gradient box
reading "Short walkthrough clip / **Placeholder for a ~20s silent demo GIF. Swap this region for a hosted
clip URL when ready.**" This is the only shipped placeholder region in the web client. Feature discovery
matters most to **self-learners** (no instructor to onboard them) and **K-12** teachers/students, and a
visible "placeholder … when ready" undercuts trust in the exact surface meant to build it.

## 2. Goals

- Replace the placeholder with real, lightweight walkthrough media (or remove the region cleanly when a
  feature has none).
- Support a per-feature media source so panels can show the right clip.
- Keep it self-contained/performant (short, silent, captioned, lazy-loaded) and accessible.

## 3. Non-Goals

- Building an authoring tool for the clips (assets are produced by Content/UX out-of-band).
- A full interactive product-tour engine.
- Gating features behind watching the media.

## 4. Personas & User Stories

- **As a self-learner**, I want a short visual of how a feature works so that I can use it without
  external docs.
- **As a K-12 teacher**, I want the help panel to show a real walkthrough so that I trust the guidance.
- **As a user on a metered connection**, I want the clip to load only when I open the panel and to be
  small.

## 5. Functional Requirements

- **FR-1.** Each feature-help entry MUST reference an optional media asset (hosted clip/GIF/short MP4)
  keyed by feature.
- **FR-2.** When a media asset exists, the dock MUST render it (lazy-loaded on panel open); when none
  exists, the dock MUST omit the media region entirely (no placeholder).
- **FR-3.** No "placeholder … when ready" copy may ship to end users.
- **FR-4.** Media MUST be captioned or purely visual/silent with an accessible text alternative; if
  video, it MUST have controls or be a non-essential decorative loop with an equivalent text description.
- **FR-5.** Media MUST be lazy-loaded and size-budgeted; the panel MUST render text help immediately
  regardless of media load state.

## 6. Non-Functional Requirements

- **Performance** — Clip loads only when the panel opens; ≤ a defined size budget (e.g. ≤2MB); no impact
  on route LCP.
- **Security** — Assets served from a trusted origin allowed by CSP; no third-party trackers.
- **Privacy & Compliance** — No PII in clips; captions for accessibility.
- **Accessibility** — WCAG 2.1: video controls / non-auto-play or decorative + text alternative; the
  region is not `aria-hidden` if it conveys information; captions for any narration (clips are silent by
  design).
- **Reliability** — Media load failure degrades to text-only help (no broken image/box).
- **Observability** — Optional: `feature_help_media_view` per feature; media error rate.
- **Maintainability** — Media map lives alongside the feature-help content registry.
- **Internationalization** — Prefer silent/visual clips so one asset serves all locales; captions/text
  alternatives localized under W01.
- **Backward compatibility** — Features without media simply omit the region.

## 7. Acceptance Criteria

- **AC-1.** *Given* any feature-help panel, *When* it opens, *Then* no "placeholder for a demo GIF" text
  appears.
- **AC-2.** *Given* a feature with a media asset, *When* the panel opens, *Then* the clip lazy-loads and
  plays/shows with an accessible text alternative.
- **AC-3.** *Given* a feature without a media asset, *When* the panel opens, *Then* the media region is
  absent and the text help fills the space.
- **AC-4.** *Given* the media fails to load, *When* the panel opens, *Then* text help still renders and
  no broken box shows.

## 8. Data Model

- No DB change. A static per-feature media registry (URL + alt/caption) alongside the existing help
  content definitions.

## 9. API Surface

- None (static assets). If media is later CMS-managed, that is a follow-up.

## 10. UI / UX

- **Modified:** `components/feature-help/feature-help-dock.tsx` (media region), feature-help content
  registry (add media field).
- **Flows:** open help → text renders immediately → media lazy-loads if present.
- **States:** no-media (region omitted), loading (skeleton within the 16:9 box), error (region omitted /
  text-only), loaded.
- **Accessibility:** remove the current `aria-hidden` on a region that will carry meaningful media, or
  provide a proper text alternative; ensure focus/controls are reachable if it becomes a video.
- **Copy & i18n:** alt text/captions via `t()`.

## 11. AI / ML Considerations

- Not applicable.

## 12. Integration Points

- `clients/web/src/components/feature-help/feature-help-dock.tsx` and the feature-help content registry;
  asset hosting/CSP config.

## 13. Dependencies & Sequencing

- **Must ship after:** none.
- **Must ship before:** none.
- **Shared infra:** static asset hosting for clips (CSP-allowed origin).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Clips never get produced, region stays empty | M | M | FR-2: omit the region when no asset — ship the removal even before clips land. |
| Large media hurts performance | M | M | Size budget + lazy load + silent short clips. |
| CSP blocks hosted media | L | M | Serve from an allowed origin / bundle small assets. |

## 15. Rollout Plan

- **Feature flag:** none. Phase 1: remove the placeholder / omit-when-empty (ships immediately). Phase 2:
  add clips per feature as Content/UX delivers them.
- **Sequencing:** placeholder removal → wire media field → add first-wave clips (top features by usage).
- **GA criteria:** AC-1 holds everywhere; first-wave features have real clips.
- **Rollback:** omit-when-empty is the safe default.

## 16. Test Plan

- **Unit** — dock renders media when present, omits region when absent, degrades on error.
- **Integration** — feature-help registry media wiring.
- **End-to-end** — Playwright: no placeholder copy in any help panel; media lazy-loads on open.
- **Accessibility** — axe on the panel; text alternative present; no meaningful content left `aria-hidden`.
- **Performance** — media not fetched until panel open; within size budget.

## 17. Documentation & Training

- Content/UX guide: clip spec (length, silent, dimensions, size budget, caption/alt requirements).
- Engineering: how to attach media to a feature-help entry.

## 18. Open Questions

1. Format: silent looping MP4 vs. GIF (MP4 is far smaller for equal quality)?
2. Which features get first-wave clips (rank by help-panel open rate)?
3. Host clips as bundled assets or from a CDN origin added to CSP?

## 19. References

- `clients/web/src/components/feature-help/feature-help-dock.tsx:59` ("Placeholder for a ~20s silent demo
  GIF. Swap this region for a hosted clip URL when ready.").
- Related plans: [W01](W01-i18n-application-coverage.md).
- Standards: WCAG 2.1 SC 1.2.x (captions/alternatives), 1.4.2 (audio control — clips are silent).
