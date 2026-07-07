# W01 — Application-Wide Internationalization & RTL Coverage

> Implementation plan. Source: web market-readiness scan (2026-07-06). Related: [docs/completed/11-i18n-l10n/11.1-i18n-framework.md](../../completed/11-i18n-l10n/11.1-i18n-framework.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | W01 |
| **Section** | Web / Internationalization & Localization |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | PARTIAL |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Frontend platform team + Localization |
| **Depends on** | 11.1 (i18n framework, completed), 11.2 (RTL, completed), 11.3 (locale-aware dates/numbers, completed) |
| **Unblocks** | W02 (parent portal ships localized), international GTM |

---

## 1. Problem Statement

The i18n *framework* is done (`i18next` + ICU plugin + HTTP backend, plan 11.1), but the *rollout* is
not: only three namespaces — `common`, `auth`, `compliance` — are externalized and translated to
`es`/`fr`. Just **5 of 199 page components and 57 of 508 components** call `t()`; a spot count found
~348 hardcoded English strings in `pages/lms/*.tsx` alone. The locale switcher offers Arabic and
Hebrew, but `resolveResourceLanguage` maps everything except `es`/`fr` back to English, so `ar`/`he`
users get English strings in an RTL layout. The result: the product is effectively English-only. This
blocks Spanish-language K-12 districts (parent-facing communications in a family's home language are a
Title VI civil-rights obligation), and caps the addressable market for international HE and self-learners.

## 2. Goals

- Externalize all user-facing strings in the signed-in app into i18n namespaces (no hardcoded copy in
  render paths).
- Ship complete `es` and `fr` bundles for every namespace, and at least one RTL language (`ar`) with a
  real translation bundle so RTL is exercised end-to-end.
- Make "add a page/component" fail CI if it introduces an untranslated literal (extend the existing
  `eslint-plugin-lextures-i18n`).
- Prioritize the K-12 parent/guardian and student surfaces (§W02) and the SL onboarding/checkout funnel
  for first-wave translation.
- Establish a repeatable extraction → translation-memory → review pipeline (plan 11.5) so coverage does
  not regress.

## 3. Non-Goals

- Machine-translating instructor-authored *content* (course pages, questions) — that is content
  translation (plan 11.x content pipeline), not UI localization.
- Adding new languages beyond `es`, `fr`, `ar` in this plan (they become cheap once the pipeline exists).
- Locale-aware *formatting* internals (dates/numbers/tz) — already delivered by 11.3/11.4; this plan
  only ensures new strings use them.

## 4. Personas & User Stories

- **As a Spanish-speaking parent (K-12)**, I want the family dashboard, grades, and messages in Spanish
  so that I can follow my child's progress without a translator.
- **As a newcomer EL student**, I want the course, modules, and quiz UI in my language so the interface
  is not a second barrier on top of the subject matter.
- **As an international self-learner**, I want the catalog, checkout, and certificate flows localized so
  I trust the purchase.
- **As an Arabic-speaking HE student**, I want the app mirrored (RTL) *and* translated, not mirrored
  English.
- **As a frontend engineer**, I want the linter to stop me from shipping a hardcoded string so coverage
  can only go up.

## 5. Functional Requirements

- **FR-1.** All user-facing text rendered by components under `clients/web/src/pages/**` and
  `clients/web/src/components/**` MUST resolve through `t()` / `<Trans>`; no literal display strings in
  JSX text nodes, `aria-label`, `title`, `placeholder`, or toast/error copy.
- **FR-2.** The system MUST provide complete `en`, `es`, and `fr` bundles for every namespace, plus a
  complete `ar` bundle to validate RTL.
- **FR-3.** `SUPPORTED_LOCALES` and `resolveResourceLanguage` MUST resolve `ar` to the `ar` bundle (not
  `en`); the locale switcher MUST NOT advertise a language that falls back to English.
- **FR-4.** New namespaces SHOULD be split by domain (e.g. `courses`, `gradebook`, `assessment`,
  `admin`, `billing`, `parent`) and lazy-loaded via the existing `i18next-http-backend`.
- **FR-5.** `recordMissingTranslationKey` MUST report missing keys to telemetry in production so coverage
  gaps are observable, and MUST fail the test run in CI.
- **FR-6.** The `eslint-plugin-lextures-i18n` rule MUST flag hardcoded user-facing literals in new/edited
  files and be enabled at error level in `pages/**` and `components/**`.
- **FR-7.** Pluralization and interpolation MUST use ICU message format (already wired via
  `icu-format-plugin.ts`); string concatenation to build sentences is prohibited.
- **FR-8.** `<html lang>` and `dir` MUST track the active locale (RTL for `ar`/`he`) — verify the
  existing `apply-document-locale.ts` covers every entry point.

## 6. Non-Functional Requirements

- **Performance** — Namespace bundles lazy-loaded per route; initial bundle only ships `common` +
  route's namespace. No measurable regression to LCP; translation JSON gzipped.
- **Security** — Translation files are static assets; no user input is interpolated as a key. Guard
  against XSS in `<Trans>` by disallowing raw HTML from translation values except vetted tags.
- **Privacy & Compliance** — Title VI (US) language-access for K-12 parent communications; WCAG 3.1.1
  (Language of Page) and 3.1.2 (Language of Parts). No PII in translation strings.
- **Accessibility** — Screen readers must announce content in the correct language (`lang` on switched
  regions). RTL must not break focus order or reading order.
- **Scalability** — Adding a locale is a data task (new bundle), not code; extraction is automated.
- **Reliability** — Missing key falls back to `en` value (never a raw key) so the UI never shows
  `courses.title.missing`.
- **Observability** — `missing_translation_key` counter by `{locale, namespace, key}`; dashboard of
  coverage % per namespace.
- **Maintainability** — One key-naming convention (`namespace:domain.subject.action`); translations live
  under `public/locales/<lang>/<namespace>.json`.
- **Internationalization** — This *is* the i18n plan; every downstream plan (W02) inherits the pipeline.
- **Backward compatibility** — `en` remains the source of truth and default; no behavior change for
  English users beyond keys resolving through `t()`.

## 7. Acceptance Criteria

- **AC-1.** *Given* the app set to `es`, *When* a user navigates the dashboard, a course, the gradebook,
  the parent portal, onboarding, and checkout, *Then* zero English strings render (verified by an
  automated "no untranslated node" Playwright sweep against a pseudo-locale).
- **AC-2.** *Given* the app set to `ar`, *When* any signed-in page loads, *Then* `dir="rtl"` is applied
  and text is Arabic, not English.
- **AC-3.** *Given* a PR adds a hardcoded JSX literal in `pages/**`, *When* CI runs, *Then* lint fails
  with the i18n rule.
- **AC-4.** *Given* a key is missing in `fr`, *When* the page renders, *Then* the English fallback text
  shows (not the raw key) and a `missing_translation_key` metric is emitted.
- **AC-5.** *Given* the pseudo-locale (`en-XA` accented/expanded) is active, *When* every route is
  visited, *Then* no clipped/overflowing layout is detected by the visual sweep.

## 8. Data Model

- No database changes. Locale preference already persists via `locale-storage.ts` (localStorage) and the
  user profile locale field used by 11.1.
- Translation assets: `clients/web/public/locales/<lang>/<namespace>.json`, added per new namespace.
- Extraction manifest (build-time): `clients/web/scripts/i18n-extract.*` output listing keys and source
  locations for the translation-memory step (plan 11.5).

## 9. API Surface

- No new HTTP routes. Consumes the existing `GET /api/v1/platform/features` and the user profile locale
  field. Namespace JSON is served as static assets via `i18next-http-backend`.
- If server-driven enum labels (e.g. status strings) are currently returned as English text, add a
  parallel machine-readable code so the web client can localize them; otherwise the client localizes by
  code. (Track any such server responses as sub-tasks.)

## 10. UI / UX

- **New:** per-namespace lazy loading; expanded locale switcher (remove languages without a real bundle,
  or gate them behind a "beta" tag once a bundle exists).
- **Flows:** (1) user picks a language → (2) `apply-document-locale` sets `lang`/`dir` → (3) route loads
  its namespace → (4) all copy resolves; missing keys fall back to `en`.
- **States:** loading of a namespace must not flash raw keys — suspend or show `common` skeleton copy.
- **Responsive/RTL:** verify flex/grid mirroring, icon direction (chevrons, progress), and that
  `ps-*`/`pe-*` logical Tailwind utilities are used instead of `pl-*`/`pr-*` in touched components.
- **Accessibility:** switched-language regions carry `lang`; focus order unchanged under RTL.
- **Copy & i18n keys:** first-wave namespaces — `parent`, `courses`, `gradebook`, `assessment`,
  `onboarding`, `billing`, `dashboard`.

## 11. AI / ML Considerations

- Optional: use the translation-memory + MT suggestion pipeline (plan 11.5) to pre-fill `es`/`fr`/`ar`
  drafts for human review. PII redaction not needed (UI strings only). Cost is bounded by one-time
  extraction volume; human review is the gate before publish.

## 12. Integration Points

- `clients/web/src/i18n/*` (index, supported-locales, missing-key, apply-document-locale, icu plugin).
- `clients/web/eslint-plugin-lextures-i18n.js` and `.oxlintrc.json` / eslint config (enable at error).
- `clients/web/public/locales/**` (new namespace bundles).
- Every touched page/component under `pages/**`, `components/**`.
- Translation-memory tooling (plan 11.5).

## 13. Dependencies & Sequencing

- **Must ship after:** 11.1, 11.2, 11.3 (all completed).
- **Must ship before:** W02 GA (so the parent portal ships localized).
- **Shared infra:** translation-memory service (11.5), CI lint gate, telemetry sink for missing keys.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Big-bang extraction stalls | H | M | Namespace-by-namespace rollout behind the lint gate; ship value per surface. |
| Layout breaks under longer `de`/`fr`/pseudo-locale strings | M | M | Pseudo-locale visual sweep in CI (AC-5); logical CSS utilities. |
| RTL regressions in complex components (gradebook, editor) | M | M | RTL Playwright pass on high-traffic routes; ship `ar` to force real testing. |
| Server-returned English enums leak through | M | M | Audit API responses; localize by code client-side. |
| Coverage regresses after launch | M | M | Lint gate + `missing_translation_key` metric + coverage dashboard. |

## 15. Rollout Plan

- **Feature flag:** none needed for `en`. Gate non-English switcher entries behind `ffI18nBeta` per
  language until its bundle is complete.
- **Sequencing:** enable lint gate on *new* code first → externalize namespace by namespace → publish
  `es` → `fr` → `ar` per namespace → flip switcher entry live when a language's bundle hits 100%.
- **Pilot:** a Spanish-language K-12 pilot district on the `parent` + `student` namespaces.
- **GA criteria:** 100% of first-wave namespaces translated to `es`/`fr`, `ar` bundle passing RTL sweep,
  zero untranslated nodes in the automated sweep.
- **Rollback:** switcher entry can be pulled per language without code changes; `en` fallback guarantees
  no broken UI.

## 16. Test Plan

- **Unit** — `resolveResourceLanguage` resolves `ar`→`ar`; ICU plural/interpolation for `es`/`fr`/`ar`.
- **Integration** — namespace lazy-load; missing-key fallback path emits metric.
- **End-to-end** — Playwright "no untranslated node" sweep under pseudo-locale across all routes;
  RTL smoke on dashboard/course/gradebook/parent.
- **Security** — `<Trans>` cannot inject arbitrary HTML from a translation value.
- **Accessibility** — axe: `html[lang]` correct; language-of-parts on switched regions; RTL focus order.
- **Performance / load** — bundle-size budget check per namespace; LCP unchanged on cold load.
- **Manual exploratory** — native-speaker review of `es`/`fr`/`ar` for the parent + checkout flows.

## 17. Documentation & Training

- Engineering: "How to add a localized string / new namespace" in `clients/web/README.md`.
- Localization: translator handbook + glossary (education terms: "rubric", "syllabus", "term").
- Help center: language-switch instructions per market.
- Runbook: how to read the coverage dashboard and triage missing-key alerts.

## 18. Open Questions

1. Do we localize server-returned enum/status text via server codes, or maintain a client-side label map?
2. Which additional languages does GTM want first after `es`/`fr`/`ar` (e.g. `zh`, `pt-BR`)?
3. Should Hebrew (`he`) ship in this plan or wait until a second RTL language is funded?

## 19. References

- `clients/web/src/i18n/supported-locales.ts`, `index.ts`, `missing-key.ts`, `apply-document-locale.ts`,
  `icu-format-plugin.ts`.
- `clients/web/public/locales/{en,es,fr}/{common,auth,compliance}.json`.
- `clients/web/eslint-plugin-lextures-i18n.js`.
- Related plans: [11.1](../../completed/11-i18n-l10n/11.1-i18n-framework.md),
  [11.2](../../completed/11-i18n-l10n/11.2-rtl-support.md),
  [11.5](../../completed/11-i18n-l10n/11.5-translation-memory.md), [W02](W02-parent-guardian-portal-completeness.md).
- Standards: WCAG 2.1 SC 3.1.1 / 3.1.2; US Title VI language access; ICU MessageFormat.
