# LH.1 — Global Dashboard Dark Mode: Reproducible Lighthouse Harness

> Implementation plan. Source: [docs/lighthouse/global-dashboard-darkmode.json](../../lighthouse/global-dashboard-darkmode.json) — runtime error `NO_FCP`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | LH.1 |
| **Section** | Lighthouse remediation |
| **Severity** | BLOCKER |
| **Markets** | All |
| **Status (today)** | MISSING |
| **Estimated effort** | S (1 w) |
| **Owner (proposed)** | Frontend platform team |
| **Depends on** | None |
| **Unblocks** | LH.2, LH.3, future `docs/lighthouse/*` baselines |

---

## 1. Problem Statement

The 2026-06-15 Lighthouse run against the global dashboard (`http://localhost:5173/`) in dark mode produced a **failed report** with runtime error `NO_FCP` ("The page did not paint any content"). All 155 audits returned `scoreDisplayMode: error`; category scores for Performance, Accessibility, Best Practices, and SEO are `null`. A secondary warning notes that **IndexedDB** data may skew performance measurements. Without a reproducible, authenticated, dark-mode Lighthouse harness, remediation plans for the dashboard cannot be prioritized or verified. This blocks LH.2 and LH.3 and any CI performance budget work referenced in plan 21.2.

## 2. Goals

- Produce a valid Lighthouse JSON report for the signed-in global dashboard with `.dark` applied.
- Document a one-command local workflow and a CI job that engineers can run without DevTools guesswork.
- Eliminate false `NO_FCP` failures caused by auth redirects, storage clearing, or backgrounded browser windows.
- Store committed baseline artifacts under `docs/lighthouse/` for diff review on PRs.

## 3. Non-Goals

- Fixing performance or accessibility findings (owned by LH.2 and LH.3 once a baseline exists).
- Lighthouse audits of marketing (`www/`) or unauthenticated login/signup pages.
- Production HTTPS / HSTS / CSP audits (localhost HTTP is expected in dev; production checks belong in deploy runbooks).

## 4. Personas & User Stories

- **As a frontend engineer**, I want to run Lighthouse on the dashboard in dark mode with one command so that I can verify regressions before opening a PR.
- **As a platform engineer**, I want CI to fail when Lighthouse cannot gather a page load so that we do not merge invalid reports.
- **As a QA engineer**, I want committed JSON baselines with category scores so that I can compare runs over time.
- **As an accessibility reviewer**, I want dark mode applied before the audit starts so that contrast results match the user's chosen theme.

## 5. Functional Requirements

- **FR-1.** The harness MUST authenticate before navigating to `/` (inject JWT into `localStorage` or set session cookie) so that `RequireAuth` does not redirect to `/login` mid-audit.
- **FR-2.** The harness MUST apply dark mode (`document.documentElement.classList.add('dark')`) before first contentful paint, matching `UiThemeSync` / `readStoredUiTheme()` behavior.
- **FR-3.** The harness MUST wait for a stable dashboard ready signal (e.g. `getByRole('navigation', { name: 'Main' })` visible and `DashboardLoadingSkeleton` absent) before ending the navigation phase.
- **FR-4.** The harness MUST either run in a fresh browser profile with IndexedDB cleared, or document `--disable-storage-reset` plus explicit IndexedDB wipe so the IndexedDB warning from the source report does not skew scores.
- **FR-5.** The harness MUST keep the browser window in the foreground (headless Chrome with `--window-size` is acceptable; DevTools manual runs MUST document the foreground requirement).
- **FR-6.** The harness MUST write output to `docs/lighthouse/global-dashboard-darkmode.json` (or a timestamped sibling committed on baseline updates).
- **FR-7.** The harness SHOULD use the same throttling profile as the source report (`formFactor: mobile`, simulated throttling) for comparable scores.
- **FR-8.** The harness SHOULD seed a test user with at least two enrolled courses so dashboard sections render (student and/or staff rows).

## 6. Non-Functional Requirements

- **Performance** — Harness completes in < 3 min locally; CI job < 8 min including stack boot.
- **Security** — Test credentials are ephemeral (e2e fixtures); no production secrets in scripts.
- **Privacy & Compliance** — Reports contain no real user PII; use synthetic e2e accounts only.
- **Accessibility** — N/A for the harness itself.
- **Scalability** — Script is parameterized (`PAGE_URL`, `THEME`) for reuse on other routes.
- **Reliability** — Three consecutive local runs produce valid JSON (no `runtimeError`) before merging.
- **Observability** — CI uploads Lighthouse JSON as a build artifact; category scores echoed in job summary.
- **Maintainability** — Script lives in `e2e/` or `clients/web/scripts/` beside existing Playwright fixtures.
- **Internationalization** — Default locale `en-US` matches source report.
- **Backward compatibility** — Does not change production auth or theme behavior.

## 7. Acceptance Criteria

- **AC-1.** *Given* the harness runs against a local dev stack with a seeded e2e user, *When* the JSON report is written, *Then* `runtimeError` is absent and `categories.performance.score` is a number between 0 and 1.
- **AC-2.** *Given* the harness runs with `THEME=dark`, *When* a screenshot or DOM snapshot is taken at audit time, *Then* `document.documentElement` has class `dark`.
- **AC-3.** *Given* the harness runs without an auth token, *When* it attempts `/`, *Then* the script fails fast with a clear error ("auth required") rather than producing a `NO_FCP` report.
- **AC-4.** *Given* three consecutive CI runs on `main`, *When* reports are compared, *Then* all three lack `NO_FCP` and category scores vary by < 5 points (noise tolerance).
- **AC-5.** *Given* the committed baseline JSON, *When* a reviewer opens it, *Then* `requestedUrl` is `/` and `configSettings.onlyCategories` includes performance and accessibility.

## 8. Data Model

No database changes. Optional: document a standard e2e seed fixture (courses + enrollments) reused by Lighthouse and Playwright.

## 9. API Surface

No API changes. Harness uses existing signup/login endpoints from `e2e/fixtures/api.ts`.

## 10. UI / UX

No user-facing UI. Operator docs only:

1. Start stack (`make dev` or e2e-local script).
2. Run `npm run lighthouse:dashboard:dark` (name TBD).
3. Inspect `docs/lighthouse/global-dashboard-darkmode.json`.
4. If `NO_FCP`, check auth token, foreground window, and API availability.

## 11. AI / ML Considerations

Not applicable.

## 12. Integration Points

- `e2e/fixtures/api.ts`, `e2e/fixtures/test.ts` — auth and seed helpers.
- `clients/web/src/auth/require-auth.tsx` — redirect behavior the harness must avoid.
- `clients/web/src/components/layout/ui-theme-sync.tsx` — theme application reference.
- `clients/web/src/lib/ui-theme.ts` — `applyUiTheme`, `readStoredUiTheme`.
- `docs/lighthouse/global-dashboard-darkmode.json` — baseline artifact.
- Optional: `@lhci/cli` or `lighthouse` npm package with Puppeteer/Playwright driver.

## 13. Dependencies & Sequencing

- Must ship after: nothing.
- Must ship before: LH.2, LH.3, Lighthouse CI budgets in plan 21.2.
- Shared infra needed: Chromium, running web + API stack.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Lighthouse CLI version drift vs DevTools | M | M | Pin `lighthouse` version in `package.json`; match major version 13.x from source report |
| Flaky scores from network/API variance | H | M | Seed deterministic fixture data; mock optional secondary APIs if needed |
| Auth token expiry mid-audit | L | H | Mint fresh token immediately before navigation |
| Headless Chrome still triggers `NO_FCP` on M-series Macs | M | H | Use Playwright `page.goto` + `lighthouse` programmatic API with `disableStorageReset` |

## 15. Rollout Plan

- No feature flag.
- Sequence: local script → document in `AGENTS.md` → optional CI job on `main` nightly (non-blocking) → make CI blocking after LH.2/LH.3 targets are set.
- Dogfood: frontend team runs script once before merging LH.2 work.
- GA criteria: AC-1 through AC-5 pass; valid baseline committed replacing the failed `NO_FCP` report.
- Rollback: remove CI job; script remains for manual use.

## 16. Test Plan

- **Unit** — Helper that applies dark mode and asserts `classList.contains('dark')`.
- **Integration** — Script against e2e-local stack produces JSON without `runtimeError`.
- **End-to-end** — Reuse Playwright auth injection pattern from `e2e/tests/contrast.spec.ts`.
- **Security** — Script refuses to run against non-localhost origins without explicit env override.
- **Accessibility** — N/A.
- **Performance / load** — N/A (this plan enables measurement).
- **Manual exploratory** — Compare DevTools Lighthouse vs CLI scores after harness lands; document any intentional delta.

## 17. Documentation & Training

- `AGENTS.md` — add Lighthouse commands table entry.
- `docs/plan/lighthouse/README.md` — update report status row when baseline is valid.
- Internal runbook: "Troubleshooting `NO_FCP` on authenticated SPAs."

## 18. Open Questions

1. Should Lighthouse run in CI on every PR or only nightly / on `clients/web/**` changes?
2. Use `@lhci/cli` with assertions or a thin custom script around `lighthouse` + Playwright?
3. Should dark mode follow stored user preference from API instead of forcing `.dark` for all dark baselines?
4. Commit screenshots alongside JSON for visual diff review?

## 19. References

- Source report: [`docs/lighthouse/global-dashboard-darkmode.json`](../../lighthouse/global-dashboard-darkmode.json) (`runtimeError.code: NO_FCP`, `runWarnings` IndexedDB).
- Auth gate: `clients/web/src/auth/require-auth.tsx`.
- Dashboard page: `clients/web/src/pages/lms/dashboard.tsx`.
- Related plans: [LH.2](LH.2-global-dashboard-darkmode-performance.md), [LH.3](LH.3-global-dashboard-darkmode-accessibility.md), [21.2 mobile-responsive review](../21-mobile-offline-cross-platform/21.2-mobile-responsive-review.md).
- External: [Lighthouse NO_FCP error](https://github.com/GoogleChrome/lighthouse/blob/main/docs/error-handling.md), [Chrome Lighthouse authenticated pages guidance](https://developer.chrome.com/docs/lighthouse/overview/).
