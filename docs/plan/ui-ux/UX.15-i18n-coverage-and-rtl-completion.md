# UX.15 — Internationalization Coverage and RTL Completion

> Implementation plan. Source: [audit.md](audit.md) §6 G-10.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | UX.15 |
| **Section** | UI/UX — Cross-cutting |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — excellent pipeline, 34% component coverage |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Web + Localisation |
| **Depends on** | UX.2 |
| **Unblocks** | UX.7 (nav labels must be keys); non-English market sales |

---

## 1. Problem Statement

Lextures ships four locales — `en`, `es`, `fr` and `ar` — at **exact key parity
(3,746 keys each)**, with ICU message formatting, an RTL locale registry,
missing-key telemetry, an ESLint plugin and a CI parity check. The infrastructure
is genuinely good. **Coverage is not: only 273 of 795 component files (34%) use
`useTranslation`**, across 10 namespaces. Roughly two-thirds of the product's
user-visible text is hardcoded English — including *both navigation files*, so an
Arabic user today navigates an English sidebar reading "Dashboard", "Courses",
"Modules", "Gradebook". Because `ar` is a shipped RTL locale, this is not a future
concern; it is a present defect for every non-English user.

## 2. Goals

- Take i18n coverage from 34% to **≥98%** of user-visible strings, enforced by CI.
- Complete **RTL correctness** end to end — layout work is largely done, strings
  and directional assets are not.
- Make locale-aware formatting (dates, times, numbers, currency, names, plurals)
  systematic rather than incidental.
- Establish a **translation workflow** that keeps four locales at parity without
  manual key management.

## 3. Non-Goals

- Adding new locales. The mechanism should make adding one cheap, but this plan
  ships the existing four correctly.
- Translating *user-authored content* (course material, discussions) — that is the
  existing course-translations feature.
- Native clients' localisation.
- Locale-specific pedagogy, grading schemes or academic calendars.

## 4. Personas & User Stories

- **As an Arabic-speaking student**, I want the entire interface in Arabic, laid
  out right-to-left, not an English sidebar around Arabic content.
- **As a Spanish-speaking parent**, I want notifications and the family dashboard
  in Spanish.
- **As a French-speaking instructor**, I want dates, times and grades formatted the
  way I read them.
- **As a localisation manager**, I want to see exactly which strings are missing in
  which locale.
- **As an engineer**, I want CI to stop me shipping a hardcoded string.
- **As a sales engineer**, I want to demo the product in a prospect's language
  without hitting English fragments.

## 5. Functional Requirements

### Coverage

- **FR-1.** All user-visible strings MUST come from i18n keys. This explicitly
  includes: `aria-label`, `alt`, `title`, `placeholder`, `aria-description`,
  toast/error/empty-state copy, chart labels, and document `<title>`.
- **FR-2.** The existing `eslint-plugin-lextures-i18n` MUST be extended to detect
  hardcoded strings in JSX text nodes **and** in the attributes above, and MUST run
  as an **error** with a ratcheting allowlist.
- **FR-3.** A coverage metric (`i18n-coverage` = translated strings / total
  user-visible strings) MUST be computed in CI and MUST NOT decrease.
- **FR-4.** Namespaces MUST be reorganised by feature area so a translator can work
  on one coherent area. Current: 10 namespaces for a 200-route product.
- **FR-5.** `clients/web/scripts/i18n-extract.mjs` MUST support extracting new keys
  from source into locale files, and CI MUST fail on keys present in code but
  missing from any locale.

### Formatting

- **FR-6.** Dates, times, relative times, numbers, percentages and currency MUST use
  `Intl` via shared helpers — never hand-built strings. The existing
  `lib/format.ts`, `components/ui/locale-time.tsx` and
  `components/timezone/deadline-datetime.tsx` are the sanctioned paths.
- **FR-7.** Pluralisation MUST use ICU plural rules — never `count === 1 ? … : …`.
  Arabic has six plural categories; a ternary is wrong there by construction.
- **FR-8.** Person names MUST be rendered through a name-formatting helper that
  respects locale ordering, not `${first} ${last}`.
- **FR-9.** Lists MUST use `Intl.ListFormat`; ranges `Intl.DateTimeFormat.formatRange`.
- **FR-10.** All times MUST render in the user's configured timezone with an
  explicit indicator where ambiguity matters (due dates especially).

### RTL

- **FR-11.** All layout MUST use logical properties (`ms-`/`me-`/`ps-`/`pe-`/
  `start-`/`end-`/`text-start`/`text-end`). A lint rule MUST forbid physical
  directional utilities. The existing `convert-physical-tailwind.mjs` script is the
  migration tool.
- **FR-12.** Directional **icons** (arrows, chevrons, back/next, undo/redo, indent)
  MUST flip in RTL; non-directional icons MUST NOT.
- **FR-13.** Keyboard arrow semantics MUST invert in RTL for tabs, menus and grids
  ([UX.4](UX.4-aria-widget-and-focus-management-remediation.md) FR-1,
  [UX.11](UX.11-data-table-and-gradebook-system.md) FR-10).
- **FR-14.** Mixed-direction content (an English course code inside Arabic prose, a
  URL, a formula) MUST render correctly using `dir="auto"` and Unicode isolation
  where needed.
- **FR-15.** Charts, progress bars, timelines and gradebook column order MUST
  respect reading direction.
- **FR-16.** The `dir` attribute MUST be set on `<html>` from the active locale and
  MUST update without reload on locale change.

### Workflow

- **FR-17.** A translation pipeline MUST exist: extract → export → translate →
  import → verify parity, so adding a locale does not require engineering per
  string.
- **FR-18.** Missing keys MUST fall back to `en` and MUST be reported through the
  existing `i18n/missing-key.ts` telemetry, not rendered as raw key paths.
- **FR-19.** A **pseudo-locale** (`en-XA`: accented, ~40% longer, bracketed) MUST be
  available in development and CI visual regression to catch truncation before
  translation.

## 6. Non-Functional Requirements

- **Performance** — Locale bundles MUST be lazily loaded per namespace; only the
  active locale ships. Adding namespaces MUST NOT increase the entry bundle. Locale
  switch ≤300 ms with no full reload.
- **Security** — Translated strings are content, not markup: ICU interpolation MUST
  escape by default and no locale file may inject HTML.
- **Privacy & Compliance** — Language preference is personal data; covered by the
  existing settings RoPA entry. Supports the accessibility-law obligation to
  provide content in the user's language where required
  (`../standards/S20-accessibility-legal-mandates.md`).
- **Accessibility** — `lang` MUST be set correctly on `<html>` and on any
  inline-language spans (WCAG 3.1.1, 3.1.2) so screen readers use the right voice.
- **Scalability** — Adding a locale = adding a translated file set; zero code
  changes.
- **Reliability** — A missing key MUST never render a raw key path or crash a page.
- **Observability** — Emit `i18n_missing_key` (key, locale, route) and
  `locale_switched`. Missing-key volume by locale is the health metric.
- **Maintainability** — One namespace per feature area; namespace ownership
  documented.
- **Backward compatibility** — Key renames MUST go through a deprecation alias for
  one release so external translation work is not invalidated.

## 7. Acceptance Criteria

- **AC-1.** *Given* the codebase, *When* the i18n lint runs, *Then* hardcoded
  user-visible strings number **0** and the allowlist is empty.
- **AC-2.** *Given* the coverage metric, *When* CI runs, *Then* `i18n-coverage`
  ≥98% and the gate fails on any decrease.
- **AC-3.** *Given* `ar` locale, *When* any of the top 40 routes renders, *Then*
  **no English text appears** in chrome, navigation, labels or messages.
- **AC-4.** *Given* `ar` locale, *When* the app renders, *Then* `dir="rtl"` is set,
  layout mirrors, directional icons flip, non-directional icons do not, and arrow
  keys invert in tabs/menus/grids.
- **AC-5.** *Given* any locale, *When* a count is displayed, *Then* ICU plural
  rules are used — verified by an Arabic test asserting all six categories.
- **AC-6.** *Given* any locale, *When* a date or number is displayed, *Then* it is
  produced by `Intl` and matches the locale's conventions.
- **AC-7.** *Given* the locale files, *When* the parity check runs, *Then* all four
  locales have identical key sets with zero missing values.
- **AC-8.** *Given* a key exists in code but not in a locale file, *When* CI runs,
  *Then* it fails.
- **AC-9.** *Given* a missing key at runtime, *When* rendered, *Then* the `en`
  fallback is shown (never a raw key path) and telemetry records it.
- **AC-10.** *Given* the pseudo-locale, *When* the top 40 routes render at 390 px
  and 1440 px, *Then* no text is truncated, clipped or overlapping.
- **AC-11.** *Given* a locale switch, *When* performed, *Then* the UI updates
  within 300 ms without a full page reload and `<html lang>`/`dir` update.
- **AC-12.** *Given* mixed-direction content, *When* rendered in `ar`, *Then*
  English course codes, URLs and formulae display correctly without character
  reordering.
- **AC-13.** *Given* a screen reader in `ar`, *When* it reads a page, *Then* the
  Arabic voice is used and inline-English spans are announced with the correct
  language.

## 8. Data Model

The user's locale preference is already persisted (`i18n/locale-storage.ts` plus
existing account settings). One additive column is required only if per-user
locale is not yet server-side:

```sql
-- server/migrations/NNN_user_locale.sql  (only if not already present)
ALTER TABLE users
  ADD COLUMN locale text,                 -- BCP-47, NULL = detect from Accept-Language
  ADD CONSTRAINT users_locale_fmt CHECK (locale IS NULL OR locale ~ '^[a-z]{2}(-[A-Za-z0-9]+)*$');
```

- **Backfill** — none; `NULL` means auto-detect.
- No other tables, enums or indexes required.

## 9. API Surface

No new routes. Two requirements on existing behaviour:

- **Server-generated user-facing text** — email templates, notification bodies,
  PDF/transcript exports and error `message` fields — MUST be localised to the
  recipient's locale, not the actor's. This is a real gap: the client can be 100%
  translated while emails remain English.
- The account settings endpoint MUST accept and return `locale`.

```ts
// Existing: PUT /api/v1/settings/account
type AccountSettings = {
  // …existing fields
  locale: string | null      // BCP-47; null = auto-detect
}
```

- No WebSocket changes. Existing rate limits apply.
- **OpenAPI** — the `locale` field MUST be documented; `make openapi-check` passes.

## 10. UI / UX

- **New pages** — none. The existing locale switcher
  (`components/settings/locale-switcher.tsx`) is surfaced more prominently,
  including on unauthenticated pages.
- **Modified pages** — the ~522 component files not currently using
  `useTranslation`; most visibly `side-nav-main-links.tsx` and
  `side-nav-course-links.tsx` (~90 hardcoded labels between them).
- **Key user flows**
  1. User picks Arabic in settings → interface switches, mirrors, and persists
     across devices.
  2. New user's locale is detected from `Accept-Language` on first load.
  3. A parent receives a notification email in their own language.
- **States** — locale switch: switching (brief), applied, failed (falls back to
  previous with an explanation). Missing translation: silent `en` fallback with
  telemetry — never a visible defect.
- **Mobile/responsive** — RTL and long-string layouts are part of the
  [UX.14](UX.14-responsive-and-small-viewport-experience.md) regression matrix.
- **Accessibility annotations** — `<html lang>` and `dir` update on switch; inline
  foreign-language spans carry `lang`; the locale switcher is a labelled listbox
  with each option in its own language and script ("العربية", not "Arabic").
- **Copy & i18n** — this plan *is* the copy work. Namespace reorganisation (FR-4)
  should mirror the [UX.7](UX.7-navigation-information-architecture.md) taxonomy so
  translators see a coherent product.

## 11. AI / ML Considerations

Machine translation MAY be used to produce **draft** translations for new keys,
but:

- **Model** — any provider via the existing AP.* multi-provider layer; no new
  dependency.
- **Prompts** — must include the key path, surrounding context and a glossary of
  product terms (`docs/brand/homeschool-terminology.md`) so terminology stays
  consistent.
- **Eval metric** — human review rate; a draft is never shipped unreviewed for
  learner-facing or legally-significant copy (consent, privacy, grades, accessibility).
- **Fallback path** — if translation is unavailable, the `en` fallback applies.
- **PII redaction** — locale files contain no user data; only static strings are
  sent. Interpolation placeholders MUST be preserved verbatim, and a validation
  step MUST reject any draft that drops or renames a placeholder.
- **Cost budget** — one-off extraction cost per new key; negligible at steady state.

## 12. Integration Points

- **External** — a translation management system if adopted (see §18 Q1);
  `intl-messageformat` and `i18next` are already dependencies.
- **Internal**
  - `clients/web/src/i18n/**` (index, locale-storage, missing-key, rtl-locales,
    supported-locales, icu-format-plugin, apply-document-locale)
  - `clients/web/public/locales/{en,es,fr,ar}/**`
  - `clients/web/eslint-plugin-lextures-i18n.js` — extended
  - `clients/web/scripts/{i18n-extract,check-i18n-locales,convert-physical-tailwind}.mjs`
  - `clients/web/src/lib/format.ts`, `components/ui/locale-time.tsx`,
    `components/timezone/**`
  - `clients/web/src/components/layout/side-nav-*.tsx` — the largest single block
    of hardcoded strings
  - `server/internal/httpserver` — localised emails, notifications, exports
  - `docs/completed/emails/` — email template localisation
- **Events** — i18n telemetry into `server/internal/telemetry`.

## 13. Dependencies & Sequencing

- **Must ship after** — [UX.2](UX.2-core-component-library-and-adoption-ratchet.md)
  (components take strings as props, so translating a component translates every
  call site).
- **Should ship before or with** — [UX.7](UX.7-navigation-information-architecture.md).
  The navigation registry centralises labels; doing i18n first means the registry is
  born with keys rather than literals. **These two should be coordinated closely.**
- **Feeds** — [UX.14](UX.14-responsive-and-small-viewport-experience.md) (pseudo-locale
  in the regression matrix).
- **Shared infra** — translation vendor or internal translators for the new key
  volume (potentially ~7,000 additional keys).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Externalising ~7,000 strings is a very large mechanical effort | **H** | **H** | Codemod extracts and replaces automatically, emitting `en` values; human review is for key *naming*, not extraction. Migrate by directory with a ratcheting lint |
| Translation cost and turnaround for 3 locales × ~7,000 keys | **H** | M | AI drafts (§11) with human review; prioritise learner-facing and chrome copy first, admin surfaces last |
| Auto-generated key names are poor and hard to translate | **H** | M | Codemod proposes names from component path + content; a review pass renames before any translation is commissioned. Deprecation aliases (§6) protect later renames |
| RTL defects only appear in Arabic, which few reviewers read | **H** | M | Pseudo-RTL mode in dev; `ar` in the visual regression matrix; recruit at least one Arabic-reading reviewer |
| Server-generated emails stay English while the UI is translated | **H** | **H** | Explicitly in scope (§9); track email/notification localisation as its own workstream with the same parity check |
| Terminology drifts between locales | M | M | Glossary in the translation brief; the existing terminology guard extended to check translated terms |
| Key renames invalidate paid translation work | M | M | One-release deprecation aliases (§6 backward compatibility) |

## 15. Rollout Plan

- **Feature flag** — none for the extraction (behaviour-identical in `en`).
  Locale *availability* is already controlled by `supported-locales.ts`; a locale
  is only advertised once its coverage passes AC-7.
- **Sequencing**
  1. Extend the lint and coverage metric; baseline recorded.
  2. Namespace reorganisation (FR-4) and pseudo-locale (FR-19).
  3. Codemod extraction by directory, `en`-only, behaviour-identical. Chrome and
     navigation first (highest user impact), then learner surfaces, then instructor,
     then admin.
  4. Key-name review pass.
  5. Translation (AI draft → human review) per batch.
  6. RTL sweep: logical properties, icon flipping, arrow inversion, mixed-direction.
  7. Server-side email/notification/export localisation.
  8. Lint flipped to error; allowlist deleted; coverage gate on.
- **Dogfood** — internal org with `ar` and `es` as daily-driver locales for
  volunteers; a "pseudo-locale day" to catch truncation.
- **GA criteria** — AC-1…AC-13 green; `i18n_missing_key` volume near zero per
  locale for 14 days; Arabic reviewer sign-off on the top 40 routes.
- **Rollback** — per-batch revert; the `en` fallback means a bad translation batch
  degrades gracefully rather than breaking.

## 16. Test Plan

- **Unit** — key resolution and fallback; ICU plural rules for all six Arabic
  categories (AC-5); `Intl` date/number/currency helpers per locale; name
  formatting; RTL icon-flip decision function; `dir` application.
- **Integration** — locale switch end-to-end including persistence and
  `Accept-Language` detection; localised email rendering per recipient locale;
  export localisation.
- **End-to-end** — Playwright in each of the four locales across the top 40 routes:
  no English leakage in `ar`/`es`/`fr` (AC-3); RTL layout and arrow-key inversion;
  locale switch without reload; mixed-direction content.
- **Security** — attempt HTML injection through a locale file value; verify ICU
  interpolation escapes; verify placeholders cannot be used to leak context.
- **Accessibility** — `lang`/`dir` correctness (WCAG 3.1.1/3.1.2); screen-reader
  pass in Arabic with an Arabic-speaking tester (AC-13); axe in all four locales.
- **Performance / load** — locale bundle sizes; switch latency (AC-11); entry
  bundle unchanged by namespace growth.
- **Visual regression** — top 40 routes × 4 locales + pseudo-locale × 390/1440 px
  (AC-10).
- **Manual exploratory** — native-speaker review of `es`, `fr`, `ar` on the top 40
  routes, with a terminology checklist.

## 17. Documentation & Training

- **End-user** — help-centre: "Changing your language", listing what is and is not
  translated (user-authored course content is separate).
- **Admin** — note on setting an organisation default locale, and on which
  server-generated communications are localised.
- **Engineer** — `docs/guides/i18n.md`: how to add a string, key-naming
  conventions, namespace ownership, ICU plurals (with the Arabic example), the
  logical-properties rule, the RTL icon-flip rule, how to run the pseudo-locale.
- **Translator** — a translation brief with the glossary, placeholder rules, and
  screenshots per namespace.
- **API reference** — the `locale` field in account settings.
- **Runbook** — "i18n coverage check failed" and "A user reports untranslated text"
  (read `i18n_missing_key` telemetry).
- **Update** `AGENTS.md` — never write a user-visible string literal in
  `clients/web/src`.

## 18. Open Questions

1. Do we adopt a translation management system (Crowdin, Lokalise, Weblate) or keep
   JSON files in-repo? *Recommendation: adopt one — 4 locales × ~11,000 keys is past
   the point where manual JSON management is safe. Decide before the extraction
   batches start producing translation work.*
2. How much of the ~7,000 new keys can be AI-drafted vs must be human-translated?
   *Recommendation: AI-draft everything, human-review everything learner-facing,
   legally significant, or in the glossary.*
3. Should admin-only surfaces be translated at all in v1, or English-only with a
   documented limitation? *Recommendation: translate — admins are the buyers, and a
   half-translated product demos badly.*
4. Are we adding locales (de, pt-BR, zh) after this? The namespace and workflow
   design should assume yes.
5. Who owns the glossary, and how does it stay in sync with
   `docs/brand/homeschool-terminology.md` and the terminology CI guard?
6. Do PDF exports (transcripts, diplomas, report cards) need locale-specific
   layouts as well as strings?

## 19. References

- Existing files: `clients/web/src/i18n/` (all 7 modules),
  `clients/web/public/locales/{en,es,fr,ar}/` (10 namespaces, 3,746 keys each),
  `clients/web/eslint-plugin-lextures-i18n.js`,
  `clients/web/scripts/i18n-extract.mjs`, `check-i18n-locales.mjs`,
  `convert-physical-tailwind.mjs`, `clients/web/src/lib/format.ts`,
  `clients/web/src/components/settings/locale-switcher.tsx`,
  `clients/web/src/components/layout/side-nav-main-links.tsx`,
  `side-nav-course-links.tsx` (~90 hardcoded labels)
- Research: [research.md](research.md) R-35, §11
- Audit: [audit.md](audit.md) G-10, G-5
- External: [WCAG 2.2 SC 3.1.1 / 3.1.2](https://www.w3.org/TR/WCAG22/),
  [Unicode CLDR Plural Rules](https://cldr.unicode.org/index/cldr-spec/plural-rules),
  [MDN — Intl](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl)
- Related plans: [UX.2](UX.2-core-component-library-and-adoption-ratchet.md),
  [UX.7](UX.7-navigation-information-architecture.md),
  [UX.14](UX.14-responsive-and-small-viewport-experience.md),
  `../../completed/11-i18n-l10n/`, `../ai-providers/` (AP.* for translation drafts)
