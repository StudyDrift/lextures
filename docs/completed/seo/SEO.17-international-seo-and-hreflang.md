# SEO.17 — International SEO & hreflang

> Completed architecture implementation. Source: [docs/plan/seo/audit.md](../../plan/seo/audit.md) §S4 (F-23).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | SEO.17 |
| **Section** | SEO — Organic & AI-Search Ranking |
| **Severity** | MINOR (MAJOR once a non-English market is targeted commercially) |
| **Markets** | K12 / HE / HS — non-US |
| **Status** | COMPLETE (locale architecture and publication gates shipped; first non-English locale awaits the evidence and human-review gate) |
| **Estimated effort** | M (2–4w for the system; ongoing for translation) |
| **Owner (proposed)** | Web platform |
| **Depends on** | SEO.1, SEO.2, SEO.5, SEO.16 |
| **Unblocks** | — |

---

## 1. Problem Statement

The product ships internationalisation including an RTL locale (per the UX plan set), and the
standards plan set covers privacy regimes in Canada, the EU/UK, Australia/NZ, Brazil, India and APAC —
but `www` is English-only with `<html lang="en">` hard-coded, no `hreflang`, and no locale routing
(audit F-23). Non-English institutional buyers searching in their own language find nothing, and
English-language buyers in other countries get US-centric pricing and compliance framing with no
signal that we serve them. This is correctly a lower priority than SEO.1–SEO.16 — there is no point
localising pages that 404 — but the *architecture* decisions must be made now, because retrofitting
locale routing onto 500 pages is expensive and retrofitting it onto 30 is not.

## 2. Goals

- Establish the URL, routing and annotation architecture for multi-locale www **before** the page
  count makes it expensive.
- Ship the first non-English locale end-to-end, chosen on evidence rather than convenience.
- Serve English-language markets outside the US correctly (regional variants, local compliance framing,
  currency) without duplicating the whole site.
- Ensure `hreflang`, canonicals and sitemaps are mutually consistent — the most common and most
  damaging failure mode in international SEO.

## 3. Non-Goals

- Localising the entire corpus. A curated high-value subset is the deliverable; localising 500 pages
  into five languages is a different program.
- Machine translation published without human review.
- Localising the marketplace catalog (creator-authored content, different problem).
- In-product i18n — that is [UX.15](../../plan/ui-ux/UX.15-i18n-coverage-and-rtl-completion.md).

## 4. Personas & User Stories

- **As a school administrator in Quebec searching in French**, I want Lextures information in French
  with Law 25 framing, so that I can evaluate it for a French-language school board.
- **As a UK university procurement officer**, I want UK-relevant pricing, GDPR/UK-GDPR framing and
  accessibility law references (EN 301 549), so that the page answers my actual questions.
- **As a search engine**, I want unambiguous `hreflang` annotations, so that I serve the right locale
  and do not treat variants as duplicates.
- **As a translator**, I want a clear source-of-truth and a review workflow, so that translated pages
  stay in sync when the English changes.
- **As a web engineer**, I want locale to be a manifest field rather than a fork of the site, so that
  maintenance cost scales sub-linearly.

## 5. Functional Requirements

**Architecture**

- **FR-1.** Locale MUST be expressed as a **subdirectory**: `/es/pricing`, `/fr-ca/pricing`, with
  English at the root (`/pricing`, no `/en` prefix, to avoid redirecting all existing URLs).
  Subdirectories are chosen over subdomains or ccTLDs because they inherit domain authority and are
  cheapest to operate.
- **FR-2.** The route manifest MUST gain `locale` and `translationOf` fields; a localised page is a
  manifest entry referencing its English source, not a separate site.
- **FR-3.** Locale codes MUST be BCP 47 and MUST distinguish language-only from regional variants
  (`es`, `es-mx`, `fr-ca`, `en-gb`) with a documented rule for when a regional variant is justified
  (materially different content — pricing, compliance, terminology — not just spelling).
- **FR-4.** `<html lang>` and `dir` MUST be set per locale, with `dir="rtl"` supported for future RTL
  locales (the design system already handles RTL per UX.15).

**hreflang & canonicals**

- **FR-5.** Every localised page MUST emit reciprocal `hreflang` annotations covering **all** locale
  variants plus `x-default`, and each page MUST include a self-referencing `hreflang`.
- **FR-6.** `hreflang` MUST be emitted in the HTML `<head>` **and** in the sitemap as
  `xhtml:link rel="alternate"` entries (SEO.2 NFR), because engines use both and inconsistency between
  them is treated as a signal error.
- **FR-7.** Canonicals MUST be **self-referential per locale** — a localised page must never canonical
  to the English original (that de-indexes it).
- **FR-8.** `x-default` MUST point to the English root version.
- **FR-9.** CI MUST fail on: a non-reciprocal `hreflang` pair, a missing self-reference, a
  `hreflang` pointing at a `noindex` or non-200 URL, an invalid locale code, or a mismatch between the
  HTML and sitemap annotations (extends SEO.16).

**Content & locale selection**

- **FR-10.** Locale priority MUST be chosen from evidence, not convenience: existing traffic by
  country (GSC), product usage by locale, the standards plan set's market coverage, and sales
  pipeline. The decision and its data MUST be recorded before translation begins.
- **FR-11.** The initial localised set MUST be a curated **~30-page core**: homepage, three segment
  hubs, `/platform/*`, `/pricing`, `/about`, `/trust` and its children, `/get-started`,
  `/request-information`, and the 10 highest-traffic help articles. Blog, guides, glossary and
  comparisons are **not** localised in phase 1.
- **FR-12.** Translations MUST be human-produced or human-reviewed by a native speaker with domain
  knowledge. Education terminology is jurisdiction-specific ("gymnasium", "sixth form", "collège")
  and a mistranslated term destroys credibility with exactly the buyer we are courting.
- **FR-13.** Localised pages MUST adapt **substance**, not only strings: currency and pricing where it
  differs, local compliance references (Law 25, UK GDPR, EN 301 549, LGPD), local terminology for
  grade levels and role titles, and locally relevant examples.
- **FR-14.** A localised page MUST NOT publish until it is complete. A half-translated page is worse
  than an English page — it MUST be excluded from routing, `hreflang` and sitemaps until finished.
- **FR-15.** Translation MUST be kept in sync: when an English source page changes materially, its
  translations MUST be flagged stale in the SEO.16 lifecycle report and MUST display an "English
  version updated" notice if stale beyond 90 days.

**User experience**

- **FR-16.** A **locale switcher** MUST appear in the header and footer, listing only locales in which
  the current page exists (falling back to the locale's homepage otherwise), with each option labelled
  in its own language.
- **FR-17.** Automatic redirection by IP or `Accept-Language` is **prohibited**. Detection MAY show a
  dismissible suggestion banner; the user's explicit choice MUST persist in a first-party cookie.
  (Auto-redirect breaks crawling and traps users in the wrong locale.)
- **FR-18.** Locale choice MUST NOT change the URL's meaning: `/fr-ca/pricing` is always French-Canadian
  pricing regardless of cookie state.

**Machine surfaces**

- **FR-19.** `llms.txt` MUST list the English set; per-locale `llms.txt` MAY be added at
  `/{locale}/llms.txt` once a locale has ≥30 pages.
- **FR-20.** Schema MUST set `inLanguage` per page, and `Organization.availableLanguage` MUST list
  supported locales (SEO.3).
- **FR-21.** Sitemaps MUST be per-locale sections within the sitemap index (SEO.2 FR-6).

## 6. Non-Functional Requirements

- **Performance** — locale must not multiply bundle size: translation strings are per-locale chunks
  loaded only for the active locale, and localised pages are prerendered exactly like English ones
  (SEO.1). Budgets from SEO.4 apply per locale.
- **Security** — the locale cookie is a first-party, non-sensitive preference; locale values MUST be
  validated against an allowlist before use in routing (path-traversal and header-injection guard).
- **Privacy & Compliance** — localised trust pages MUST accurately reflect that jurisdiction's posture
  per the standards plan set; claiming Law 25 or LGPD alignment we have not implemented is a
  compliance misstatement, not a marketing choice. Each localised trust page requires compliance review.
- **Accessibility** — `lang` and `dir` correctness is a WCAG 3.1.1/3.1.2 requirement; the locale
  switcher must be keyboard operable with each option's `lang` attribute set so screen readers
  pronounce them correctly.
- **Scalability** — architecture must support 5+ locales without a linear increase in engineering
  effort; content cost scales with translation, which is why FR-11 curates.
- **Reliability** — FR-9's CI checks prevent the classic failure where a partial rollout de-indexes
  the English pages.
- **Observability** — GSC international targeting report; per-locale index coverage, clicks and
  position; `hreflang` error counts; translation staleness.
- **Maintainability** — one source of truth per page (English), translations referenced via
  `translationOf`; no forked page components.
- **Internationalization** — this plan is the i18n plan for www; it must align with the product's
  locale set (UX.15) so a user does not find a French marketing page leading to an English product.
- **Backward compatibility** — English URLs do not change (FR-1). No existing URL moves.

## 7. Acceptance Criteria

- **AC-1.** *Given* a localised page, *When* its `<head>` is inspected, *Then* it contains reciprocal
  `hreflang` for every variant plus `x-default` and a self-reference, and its canonical points to
  itself.
- **AC-2.** *Given* the sitemap, *When* inspected, *Then* `xhtml:link` alternates match the HTML
  annotations exactly for every localised page.
- **AC-3.** *Given* a non-reciprocal `hreflang` introduced in a PR, *When* CI runs, *Then* it fails
  naming both pages.
- **AC-4.** *Given* an incomplete translation, *When* the build runs, *Then* the page is not routed,
  not in `hreflang`, and not in any sitemap.
- **AC-5.** *Given* a French-Canadian visitor, *When* they load `/pricing`, *Then* they are **not**
  redirected; they may see a dismissible suggestion, and choosing French persists.
- **AC-6.** *Given* a localised page, *When* checked, *Then* `<html lang>` and `dir` are correct,
  `inLanguage` is set in schema, and the locale switcher lists only locales where the page exists.
- **AC-7.** *Given* an English page updated materially, *When* the lifecycle report runs, *Then* its
  translations are listed as stale; *And* after 90 days the localised page displays the notice.
- **AC-8.** *Given* GSC 60 days after the first locale launches, *When* reviewed, *Then* zero
  `hreflang` errors are reported and ≥80% of localised URLs are indexed.
- **AC-9.** *Given* the locale switcher, *When* operated by keyboard and screen reader, *Then* each
  option is announced in its own language with the correct `lang` attribute.
- **AC-10.** *Given* a localised trust page, *When* published, *Then* it carries compliance review
  sign-off for that jurisdiction.

## 8. Data Model

No database changes.

```ts
// route-manifest.ts additions
type RouteDescriptor = {
  // …SEO.1 + SEO.5 fields
  locale: string              // BCP 47; 'en' for root
  translationOf?: string      // English source path
  translationStatus?: 'complete' | 'in-progress' | 'stale'
  sourceUpdatedAt?: string    // English source's `updated`, for staleness (FR-15)
}
```

```
www/src/locales/<locale>/       # translated content + UI strings
www/src/lib/locales.ts          # allowlist: code, name (native), dir, currency, status
docs/plan/seo/locale-priority.md  # FR-10 evidence and decision
```

## 9. API Surface

None. The marketplace API's `inLanguage` is already per-course; localised course pages are out of
scope (§3).

## 10. UI / UX

- **New components:** `<LocaleSwitcher>` (header + footer), `<LocaleSuggestion>` (dismissible banner),
  `<TranslationStaleNotice>`.
- **Modified:** header, footer, and the route resolver; `<html lang|dir>` set at generation.
- **Flows**
  1. Visitor lands on an English page → suggestion banner offers French → accepts → navigates to
     `/fr-ca/…` → preference persists.
  2. Visitor uses the footer switcher → same page in the target locale, or that locale's homepage if
     the page is not translated.
  3. Search engine crawls `/fr-ca/pricing` → `hreflang` cluster → indexes correctly for French Canada.
- **States** — page not available in the selected locale: switcher links to the locale home with an
  explanation, never a 404. Stale translation: notice with a link to the current English version.
- **Responsive** — switcher is a compact control on mobile, not a full menu.
- **Accessibility** — see NFRs and AC-9; the suggestion banner must be dismissible by keyboard and must
  not trap focus or auto-dismiss.
- **Copy & i18n** — every www string must be externalised as part of this work; hard-coded English in
  components is the blocker to discover first.

## 11. AI / ML Considerations

- **Machine translation is permitted as a first draft only** (FR-12). Assistants answering in French
  will cite French sources; a visibly machine-translated page reads as low quality to both readers and
  models, and education terminology is exactly where MT fails.
- **Locale-specific AI visibility must be measured separately** — SEO.15's prompt set is US-English;
  a new locale needs its own 20–30 prompt set in that language, or we will not know whether the
  investment worked.
- `inLanguage` and `availableLanguage` (FR-20) are how an assistant knows a French answer exists to
  cite.

## 12. Integration Points

- **External:** translation vendor or in-house native reviewers; GSC international targeting.
- **Internal modules touched:** `www/index.html` (`lang`/`dir` templating),
  `www/src/lib/route-manifest.ts`, `www/src/app.tsx` (locale-aware routing),
  `www/scripts/generate-site.mjs`, `www/src/components/header.tsx`, `site-footer.tsx`,
  `www/src/lib/schema/*` (`inLanguage`), `www/scripts/seo-check/checks/hreflang.mjs` (new),
  `www/src/lib/locales.ts` (new).
- **Events:** locale switch → GA4.

## 13. Dependencies & Sequencing

- **Must ship after:** [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md),
  [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md) (the `hreflang` checks extend that suite).
  Locale selection (FR-10) needs [SEO.15](SEO.15-measurement-search-console-and-ai-share-of-voice.md)
  country data.
- **Must ship before:** nothing. **However**, FR-1–FR-4 (the architecture decisions) SHOULD land during
  SEO.1/SEO.5 even if no locale ships, because retrofitting locale routing across 500 pages is
  materially more expensive than across 30.
- **Coordinates with:** [UX.15](../../plan/ui-ux/UX.15-i18n-coverage-and-rtl-completion.md) — the marketing
  locale set must not exceed the product locale set.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `hreflang` errors de-index English pages | M | **H** | FR-7 self-referential canonicals; FR-9 CI reciprocity checks; staged launch with one locale and GSC monitoring (AC-8) |
| Translation quality damages credibility with the exact buyer we want | M | **H** | FR-12 native domain reviewer; FR-13 substance adaptation; never publish MT unreviewed |
| Localised compliance claims are inaccurate | M | **H** | FR-13 + AC-10 compliance review per jurisdiction, tied to the standards plan set |
| Translations go stale and diverge from English | **H** | M | FR-15 staleness flagging + visible notice; curated 30-page scope keeps sync tractable |
| Marketing locale exists but the product does not support it | M | H | Locale set gated on UX.15 product support; a French page leading to an English app is a worse experience than no French page |
| Effort spent before evidence justifies it | M | M | FR-10 evidence-based selection; this plan is deliberately last in sequence |
| Auto-redirect implemented "for convenience" and breaks crawling | M | H | FR-17 explicit prohibition; CI check that no locale redirect exists on the English routes |

## 15. Rollout Plan

- **Feature flag:** none; locales ship when complete (FR-14 is the gate).
- **Sequencing**
  1. **Now (with SEO.1/SEO.5):** land FR-1–FR-4 architecture — `locale` in the manifest, `lang`/`dir`
     templating, externalise all www strings. No locale ships; cost of future work drops sharply.
  2. **After SEO.15 has 3 months of data:** run FR-10 locale-priority analysis; record the decision.
  3. Build `hreflang` emission + CI checks (FR-5–FR-9) and verify with a single test locale on staging.
  4. Translate and review the 30-page core for locale #1; publish only when complete.
  5. Monitor GSC international targeting for 60 days (AC-8) before starting locale #2.
  6. Add locale-specific AI prompt sets to SEO.15.
  7. Expand scope within a locale (guides, glossary) only where locale #1's data justifies it.
- **Dogfood:** a native speaker on staff or a customer in the target market reviews the full 30 pages
  before launch.
- **GA criteria:** AC-1…AC-10; zero `hreflang` errors at 60 days; locale #1 producing measurable
  non-brand clicks.
- **Rollback:** a locale can be removed by dropping its manifest entries — but its URLs must then 410
  and be removed from sitemaps and `hreflang` clusters in the same deploy, or the remaining
  annotations become invalid.

## 16. Test Plan

- **Unit** — `hreflang` cluster generation and reciprocity; locale-code validation against the
  allowlist; canonical-per-locale construction; staleness computation from `sourceUpdatedAt`;
  switcher target resolution (page exists vs. fallback to locale home).
- **Integration** — build a two-locale fixture; assert HTML and sitemap annotations match (AC-2);
  assert an incomplete translation is excluded everywhere (AC-4); assert a non-reciprocal pair fails
  (AC-3).
- **End-to-end (Playwright)** — no auto-redirect for any `Accept-Language` (AC-5); switcher behaviour
  and persistence; `lang`/`dir` correctness; RTL rendering on a test RTL locale.
- **Security** — locale path segments validated against the allowlist; no path traversal via locale;
  cookie value validated on read.
- **Accessibility** — axe on localised pages; screen-reader pass on the switcher with per-option `lang`
  (AC-9); RTL layout audit; verify `lang` changes are announced.
- **Performance / load** — per-locale bundle isolation; localised pages meet the SEO.4 budget.
- **Manual exploratory** — native-speaker review of all 30 pages; procurement-officer read-through in
  the target market; GSC international targeting review at 30 and 60 days.

## 17. Documentation & Training

- `www/docs/internationalization.md` — URL strategy, locale codes, when a regional variant is
  justified, `hreflang` rules, and the no-auto-redirect rule with its reasoning.
- `www/docs/translation-workflow.md` — source of truth, reviewer requirements, substance-adaptation
  checklist, staleness handling.
- `docs/plan/seo/locale-priority.md` — the evidence and decision for locale selection (FR-10).
- Update the add-a-page checklist with locale fields.

## 18. Open Questions

1. Which locale first? Candidates on current evidence: `es` (US Spanish-speaking districts — arguably
   the highest-value and does not require a new market), `fr-ca` (Quebec, with Law 25 alignment already
   planned in [S14](../../plan/standards/S14-canada-pipeda-quebec-law25.md)), `en-gb`. Needs FR-10 data.
2. Is `es` for US districts a *locale* or a *segment*? (A Spanish-language page for US buyers may not
   need a regional variant at all.)
3. Do we have budget for professional translation with domain review, or in-house native speakers?
4. Does the product support the target locale end-to-end (UX.15), including RTL if applicable?
5. Should localised pricing show local currency, and does billing support it?
6. Do localised trust pages need separate legal review per jurisdiction, and who provides it?

## 19. References

- Existing files: `www/index.html` (`<html lang="en">`), `www/src/app.tsx`,
  `www/src/components/header.tsx`, `www/src/lib/route-manifest.ts` (from SEO.1)
- Audit findings: [F-23](../../plan/seo/audit.md#f-23-no-internationalisation-signals)
- Research: [§1](../../plan/seo/research.md#1-the-structural-shift-retrieval-replaced-ranking) (retrieval is
  language-specific), [§4](../../plan/seo/research.md#4-entity-seo-is-the-highest-roieffort-ratio-available)
- External: [Google — Localized versions of your pages](https://developers.google.com/search/docs/specialty/international/localized-versions),
  [Google — Managing multi-regional and multilingual sites](https://developers.google.com/search/docs/specialty/international/managing-multi-regional-sites),
  [BCP 47 language tags](https://www.rfc-editor.org/info/bcp47),
  [WCAG 2.2 — 3.1.1 Language of Page](https://www.w3.org/WAI/WCAG22/Understanding/language-of-page)
- Related plans: [SEO.1](SEO.1-static-rendering-and-crawlability.md),
  [SEO.2](SEO.2-crawler-access-sitemaps-and-llms-txt.md),
  [SEO.5](SEO.5-information-architecture-and-internal-linking.md),
  [SEO.16](SEO.16-seo-governance-and-ci-guardrails.md),
  [UX.15 — i18n coverage & RTL](../../plan/ui-ux/UX.15-i18n-coverage-and-rtl-completion.md),
  [S14 — Canada / Quebec Law 25](../../plan/standards/S14-canada-pipeda-quebec-law25.md),
  [S20 — accessibility legal mandates](../../plan/standards/S20-accessibility-legal-mandates.md)
