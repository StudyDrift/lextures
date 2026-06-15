# LH.3 — Global Dashboard Dark Mode: Accessibility

> Implementation plan. Source: [docs/lighthouse/global-dashboard-darkmode.json](../../lighthouse/global-dashboard-darkmode.json) — accessibility category (blocked by `NO_FCP` until LH.1 ships).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LH.3 |
| **Section** | Lighthouse remediation |
| **Severity** | MAJOR |
| **Markets** | All |
| **Status (today)** | PARTIAL (axe contrast CI exists; Lighthouse a11y baseline missing) |
| **Estimated effort** | S (1–2w) |
| **Owner (proposed)** | Frontend platform team |
| **Depends on** | LH.1 (valid Lighthouse baseline), 12.3 (design-token contrast contract) |
| **Unblocks** | VPAT evidence for dashboard, 10.7 WCAG program |

---

## 1. Problem Statement

Lighthouse weights several accessibility audits heavily on the global dashboard: `button-name` (10), `color-contrast` (7), `link-name` (7), `heading-order` (3), and `image-alt` (10). The 2026-06-15 dark-mode report failed before scores were collected, but the dashboard and shell combine many semantic regions (`aria-label` sections, collapse toggles, quick-link chips, violet recommendation chips, indigo badges, and progress labels using `text-neutral-500` on `neutral-800`/`neutral-900`). Plan 12.3 established approved dark-mode token pairs, and `e2e/tests/contrast.spec.ts` already gates dashboard axe contrast — yet Lighthouse uses a different engine path and includes audits axe disables in `wcag.spec.ts` (e.g. dynamic Tailwind contrast). A dedicated remediation pass is needed so the dashboard achieves **Accessibility = 100** (or ≥ 95 with documented exceptions) in dark mode on mobile.

## 2. Goals

- Achieve Lighthouse **Accessibility ≥ 0.95** on `/` in dark mode (target 1.0).
- Resolve all weighted failures in groups: `a11y-color-contrast`, `a11y-names-labels`, `a11y-navigation`.
- Align dashboard markup with heading hierarchy (`h1` from `LmsPage` → `h2` sections → `h3` subsections).
- Ensure every icon-only control in dashboard sections has an accessible name.
- Add a Playwright + Lighthouse accessibility regression check to CI after LH.1.

## 3. Non-Goals

- Full-screen-reader audit of every LMS page (plan 12.1 scope).
- High-contrast theme (plan 12.7).
- Fixing accessibility issues limited to `SideNav` / `TopBar` unless they appear on the dashboard Lighthouse run (may file separate shell plan).

## 4. Personas & User Stories

- **As a student with low vision using dark mode**, I want all dashboard text and controls to meet contrast requirements so that I can read deadlines and navigate without strain.
- **As a screen reader user**, I want collapse toggles and quick links announced with clear names so that I understand each control's purpose.
- **As a district accessibility coordinator**, I want Lighthouse accessibility scores for the landing page documented for procurement.
- **As a developer**, I want failing elements identified with selectors in CI output so that I can fix regressions quickly.

## 5. Functional Requirements

- **FR-1.** After LH.1 baseline, the dashboard MUST achieve Lighthouse accessibility score ≥ 0.95 in dark mode.
- **FR-2.** Every interactive element on the dashboard MUST have an accessible name (`button-name`, `link-name`, `input-button-name` audits pass).
- **FR-3.** All text and UI component contrast on the dashboard in dark mode MUST meet WCAG 2.1 AA (`color-contrast`, `link-in-text-block` audits pass), using only pairs from `docs/design-tokens.md` or newly approved entries in `contrast-config.json`.
- **FR-4.** Heading levels MUST not skip (`heading-order` audit pass): single `h1` ("Dashboard" via `LmsPage`), section titles as `h2`, nested titles as `h3`.
- **FR-5.** Decorative icons (`lucide-react`) MUST be `aria-hidden` when adjacent text provides the label.
- **FR-6.** Progress indicators (week bar, grade snippets) MUST expose text alternatives for percentage / status, not color alone (SC 1.4.1).
- **FR-7.** The system SHOULD add `e2e/tests/lighthouse-dashboard-a11y.spec.ts` (or extend contrast spec) that fails on Lighthouse `color-contrast` violations in dark mode.

## 6. Non-Functional Requirements

- **Performance** — Accessibility fixes must not add > 5 KB gzip to dashboard chunk.
- **Security** — N/A.
- **Privacy & Compliance** — WCAG 2.1 AA SC 1.4.3, 1.4.11, 1.4.1, 2.4.6 (headings). Feeds VPAT §502.
- **Accessibility** — Manual spot-check with VoiceOver (macOS) and NVDA (Windows) on dashboard collapse sections and quick links.
- **Scalability** — N/A.
- **Reliability** — axe + Lighthouse both pass on same seeded fixture.
- **Observability** — CI publishes count of a11y audit failures from Lighthouse JSON.
- **Maintainability** — Prefer semantic tokens (`dark:text-neutral-400`) over raw hex; update `contrast-config.json` when adding pairs.
- **Internationalization** — `aria-label` strings use i18n keys, not hardcoded English.
- **Backward compatibility** — Light mode contrast must not regress.

## 7. Acceptance Criteria

- **AC-1.** *Given* LH.1 harness runs in dark mode, *When* the JSON report is generated, *Then* `categories.accessibility.score ≥ 0.95`.
- **AC-2.** *Given* the dashboard with enrolled courses, *When* axe runs with `color-contrast` enabled (as in `contrast.spec.ts`), *Then* zero violations on `/`.
- **AC-3.** *Given* the student overview collapse toggle, *When* inspected with Accessibility tree, *Then* the button exposes "Collapse Learning" / "Expand Learning" (already present — regression test).
- **AC-4.** *Given* recommendation violet chips (`dark:text-violet-100` on `dark:bg-neutral-900`), *When* contrast is measured, *Then* ratio ≥ 4.5:1 or tokens are adjusted.
- **AC-5.** *Given* progress label `dark:text-neutral-500` on `dark:bg-neutral-800`, *When* contrast is measured, *Then* ratio ≥ 4.5:1 or label promoted to `neutral-400`.
- **AC-6.** *Given* the page structure, *When* heading outline is exported, *Then* no level is skipped between `h1` and visible section headings.

## 8. Data Model

No database changes.

## 9. API Surface

No API changes.

## 10. UI / UX

Audit scope on `clients/web/src/pages/lms/dashboard.tsx` and shared shell visible on `/`:

| Area | Risk | Lighthouse audit |
|---|---|---|
| Quick link chips (Inbox, Courses, …) | Link text present | `link-name` |
| Section collapse buttons | `aria-label` present | `button-name` |
| `h2` section labels ("Learning", "Teaching", …) | Order under `h1` | `heading-order` |
| Violet recommendation chips | Custom colors | `color-contrast` |
| Indigo registration badges (`dark:text-indigo-200`) | Custom bg | `color-contrast` |
| Week progress micro-labels (`dark:text-neutral-500`) | Small text on bar | `color-contrast` |
| `LmsPage` description (`dark:text-neutral-400`) | Muted on `neutral-900` | `color-contrast` |
| Empty state "No courses yet" | Muted subtitle | `color-contrast` |

Shared shell (if in viewport during audit):
- `SideNav` landmark names, `TopBar` icon buttons — verify `button-name` / `image-alt`.

Remediation patterns:
- Promote failing muted text one step on the neutral scale (`500` → `400`).
- Add `aria-label` to any icon-only control missing names.
- Ensure status badges include text, not color-only state.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- `clients/web/src/pages/lms/dashboard.tsx`
- `clients/web/src/pages/lms/lms-page.tsx` — `h1` + description contrast
- `clients/web/src/components/layout/side-nav.tsx`, `top-bar.tsx` — shell controls in audit viewport
- `clients/web/contrast-config.json`, `docs/design-tokens.md`
- `e2e/tests/contrast.spec.ts` — extend or mirror for Lighthouse
- `docs/lighthouse/global-dashboard-darkmode.json` — post-fix baseline

## 13. Dependencies & Sequencing

- Must ship after: LH.1 (baseline), 12.3 (token contract — completed).
- Must ship before: VPAT annual update, 10.7 evidence pack.
- May parallel: LH.2 (performance) if changes do not conflict.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Fixing dark contrast breaks light mode aesthetics | M | M | Run both themes in CI |
| Lighthouse contrast differs from axe on Tailwind utilities | H | M | Prefer token pairs validated in `contrast-config.json` |
| Shell vs dashboard failures conflated | M | L | Document viewport and seed state in LH.1 harness |
| i18n for `aria-label` delayed | M | L | English keys first; file i18n follow-up |

## 15. Rollout Plan

- No feature flag.
- Sequence: LH.1 baseline → fix contrast pairs → fix names/headings → re-audit → enable CI gate.
- Dogfood with screen reader smoke test.
- GA criteria: AC-1 through AC-6; updated Lighthouse JSON committed.
- Rollback: revert token changes per PR.

## 16. Test Plan

- **Unit** — `wcag-contrast.test.ts` for any new token pairs.
- **Integration** — `npm run contrast:check` passes with new dark pairs.
- **End-to-end** — `contrast.spec.ts` dashboard dark test; new Lighthouse a11y spec post-LH.1.
- **Security** — N/A.
- **Accessibility** — Manual VoiceOver: quick links, collapse, recommendation CTA.
- **Performance / load** — N/A.
- **Manual exploratory** — Colour Contrast Analyser on violet/indigo chips in dark mode.

## 17. Documentation & Training

- Update `docs/design-tokens.md` approved dark-mode pairs table.
- Update `docs/plan/lighthouse/README.md` with accessibility score.

## 18. Open Questions

1. Should Lighthouse accessibility gate run on every PR or only with `clients/web` changes?
2. Are violet recommendation chips brand-required at current hues, or can they shift to approved indigo tokens?
3. Should shell components failing outside dashboard be tracked as LH.4-shell-chrome-accessibility?
4. Re-enable `color-contrast` in `wcag.spec.ts` once Lighthouse and axe agree?

## 19. References

- Source report: [`docs/lighthouse/global-dashboard-darkmode.json`](../../lighthouse/global-dashboard-darkmode.json) (accessibility audits errored; weighted refs include `button-name`, `color-contrast`, `heading-order`, `link-name`, `image-alt`).
- Dashboard: `clients/web/src/pages/lms/dashboard.tsx`.
- Contrast CI: `e2e/tests/contrast.spec.ts`, `clients/web/contrast-config.json`.
- Completed: [12.3 color-contrast compliance](../../completed/12-accessibility/12.3-color-contrast-compliance.md).
- Related plans: [LH.1](LH.1-global-dashboard-darkmode-audit-harness.md), [LH.2](LH.2-global-dashboard-darkmode-performance.md), [12.7 high-contrast](../../completed/12-accessibility/12.7-high-contrast-reduced-motion.md).
- External: [Lighthouse accessibility scoring](https://developer.chrome.com/docs/lighthouse/accessibility/scoring/), [axe color-contrast](https://dequeuniversity.com/rules/axe/4.10/color-contrast).
