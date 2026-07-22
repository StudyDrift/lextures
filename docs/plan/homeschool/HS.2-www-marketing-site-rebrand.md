# HS.2 — www marketing site rebrand (self-learner → Homeschool)

> Implementation plan. Source: product rebrand of the **self-learner** segment to **Homeschool**.
> Terminology and copy are fixed by [HS.1](HS.1-terminology-copy-deck-and-guardrails.md).
> Code references: `www/src/app.tsx`, `www/src/pages/self-learner-page.tsx`,
> `www/src/components/{header,site-footer}.tsx`, `www/src/lib/{site-links,api-base}.ts`,
> `www/src/pages/{pricing,get-started,k12,higher-ed,parents}-page.tsx`,
> `www/scripts/prerender-courses.mjs`, `.github/workflows/pages-www.yml`.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | HS.2 |
| **Section** | Marketing site (www) |
| **Status (today)** | PARTIAL — every audience surface names the segment "Self-learner"; the route is `/self-learner` and is indexed via `sitemap.xml` |
| **Severity** | MAJOR — public, indexed surface; a bad route change loses the segment's organic entry point |
| **Markets** | K12 / HE / HS |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web/Marketing |
| **Depends on** | HS.1 (copy deck, slug decision OQ-1) |
| **Unblocks** | HS.3 (shared copy), HS.6 (guard flip) |

---

## 1. Problem Statement

`lextures.com` presents three audiences — Higher education, K–12, Parents — plus **Self-learners**,
which is the label on the header dropdown, the footer, the `/self-learner` audience page, the
pricing table's middle card, and the first card of `/get-started`. The segment is being rebranded to
**Homeschool**, so every one of those surfaces is now wrong, and the route slug itself carries the
old name. This is the highest-visibility surface in the rename and the only one with SEO exposure:
`/self-learner` sits in `sitemap.xml` at priority 0.6 and is linked from four other pages. Getting
the redirect wrong drops the segment's landing page out of the index; getting it right converts the
existing link equity to `/homeschool`.

## 2. Goals

- Serve the audience page at `/homeschool` with all copy, nav, and CTA labels using the HS.1 terms.
- Keep `/self-learner` reachable forever as a redirect that both browsers and crawlers follow, and
  point its canonical at `/homeschool`.
- Rename the page component, file, and every `selfLearner` symbol in `www/src/lib/` so no dead term
  survives in the source.
- Update the pricing page (card label, card body, FAQ, hero sentence) and `/get-started` (card
  title, body, analytics `program` value).
- Update `sitemap.xml` so `/homeschool` is listed and `/self-learner` is not.

## 3. Non-Goals

- No change to `self.lextures.com` — every `SITE_LINKS` target and every body-copy mention of the
  hosted app host stays exactly as written today.
- No redesign of the audience page layout, `MarketingPageShell`, or the pricing table structure.
- No change to `/parents`, `/k-12`, `/higher-ed` beyond the one CTA-link symbol rename each (their
  CTA *labels* already say "Open self.lextures.com", which is a host, not the segment).
- No new content sections (homeschool co-op scheduling, portfolio/state reporting) — that is a
  content project gated on [HS.1 OQ-3](HS.1-terminology-copy-deck-and-guardrails.md#18-open-questions).
- No move off GitHub Pages, and therefore no server-side 301.

## 4. Personas & User Stories

- **As a homeschooling parent** landing from search, I want the page I reach to say "Homeschool" so I
  immediately know the product is meant for me.
- **As a visitor with a bookmarked `/self-learner` link**, I want to land on the right page rather
  than a 404.
- **As a search crawler**, I want a machine-readable signal that `/self-learner` has moved to
  `/homeschool` so ranking transfers.
- **As a prospective certification learner**, I want the use-case cards to still name my case, so the
  narrower label does not read as "not for you".
- **As an engineer**, I want `SITE_LINKS.homeschool` to be the only way to reach the hosted app from
  marketing pages, as it is today.

## 5. Functional Requirements

- **FR-1.** `www/src/pages/self-learner-page.tsx` MUST be renamed to `homeschool-page.tsx` and its
  export `SelfLearnerPage` to `HomeschoolPage`; `www/src/app.tsx:22` import and `app.tsx:108` route
  MUST follow.
- **FR-2.** The app router MUST serve the page at `/homeschool` and MUST keep a `/self-learner` case
  that performs `window.location.replace('/homeschool')` and renders nothing.
- **FR-3.** The build MUST emit a static `dist/self-learner/index.html` containing
  `<meta http-equiv="refresh" content="0; url=/homeschool">`,
  `<link rel="canonical" href="{SITE_ORIGIN}/homeschool">`, and a visible one-line fallback link —
  so crawlers and no-JS clients follow the move without executing the SPA. Emitting it belongs in
  `www/scripts/prerender-courses.mjs` beside the existing `dist/courses/**` writer.
- **FR-4.** `STATIC_ROUTES` in `www/scripts/prerender-courses.mjs:56` MUST list `/homeschool`
  (priority `0.6`) and MUST NOT list `/self-learner`.
- **FR-5.** `www/src/lib/site-links.ts` MUST rename `SELF_LEARNER_ORIGIN` → `HOMESCHOOL_ORIGIN` and
  `SITE_LINKS.selfLearner` → `SITE_LINKS.homeschool`, keeping the **value** `https://self.lextures.com`
  unchanged, and MUST keep `APP_ORIGIN` as the alias it is today.
- **FR-6.** All five `SITE_LINKS.selfLearner` call sites MUST be updated:
  `pages/self-learner-page.tsx:50,89`, `pages/pricing-page.tsx:201`, `pages/k12-page.tsx:91`,
  `pages/higher-ed-page.tsx:102`, `pages/parents-page.tsx:69`, `components/site-footer.tsx:10`,
  `pages/get-started-page.tsx:240`.
- **FR-7.** `components/header.tsx:15` and `components/site-footer.tsx:19` MUST read
  `{ label: 'Homeschool', href: '/homeschool' }`.
- **FR-8.** The audience page MUST use `eyebrow="Homeschool"` and the HS.1 hero lead; the
  "Independent study" use-case card MUST be retitled to keep the homeschool case explicit (proposed:
  **"Homeschool and independent study"**).
- **FR-9.** `pages/pricing-page.tsx` MUST update: the `SELF_LEARNER_FEATURES` constant name
  (→ `HOMESCHOOL_FEATURES`, line 16), the card `label` (line 187), the card `description` (line 198),
  the FAQ question at line 62, and the hero sentence tail at lines 152–153 — all per the HS.1 deck.
  The FAQ answer at line 55 ("K–12, higher-ed, and self-learner capabilities") MUST read
  "K–12, higher-ed, and homeschool capabilities".
- **FR-10.** `pages/get-started-page.tsx` MUST rename the `Path` union member `'self-learner'` →
  `'homeschool'` (line 39), the `PATHS[0].id`/`title`/`description` (lines 44–47), and the branch at
  lines 238–240; `trackOnboarding('homeschool')` MUST be sent.
- **FR-10a.** The `/get-started` first-card icon MUST change from `BrainCircuit` to `House`
  (`lucide-react`), matching the icon swap in [HS.4 FR-9/FR-15](HS.4-mobile-clients-rebrand.md#5-functional-requirements)
  so the three clients present the same card.
- **FR-11.** `trackOnboarding('homeschool')` MUST NOT be shipped before the server accepts the value
  — see [HS.5 FR-1](HS.5-server-copy-and-onboarding-program.md#5-functional-requirements). If HS.5
  has not landed, this PR MUST keep sending `'self-learner'` behind a one-line constant so the flip
  is a single edit.
- **FR-12.** The doc comments in `www/src/lib/api-base.ts:3,6` and
  `www/scripts/prerender-courses.mjs:35` MUST say "homeschool app origin" rather than "self-learner
  origin".
- **FR-13.** The audience page SHOULD call `useDocumentHead` (as `courses-page.tsx:45` does) to set a
  `/homeschool`-specific `<title>`, description, and canonical, since the SPA otherwise serves the
  generic `index.html` head for this route.
- **FR-14.** No string on the site MAY still match the HS.1 banned-term list after this plan, except
  inside `dist/self-learner/index.html`'s redirect body copy.

## 6. Non-Functional Requirements

- **Performance** — no new runtime dependency; the redirect stub adds one ~600-byte file. Lighthouse
  performance on `/homeschool` MUST be within 2 points of today's `/self-learner`.
- **Security** — the redirect stub is a static file with no user input; `escapeHtml` from the
  prerender module MUST be used for any interpolated value.
- **Privacy & Compliance** — `trackOnboarding` keeps the same fields; only the `program` value
  string changes. No new data collected, so no privacy-notice change.
- **Accessibility** — WCAG 2.1 AA maintained: the redirect stub MUST include a visible, focusable
  fallback link (a bare `meta refresh` with no link fails SC 2.2.1 for users who cannot follow it);
  nav label length unchanged so no reflow at 320 px.
- **Scalability** — n/a (static site).
- **Reliability** — `/self-learner` MUST keep working indefinitely; there is no expiry on the stub.
- **Observability** — GA4 (`G-JX182Q6KKX`, `www/index.html`) will show `/homeschool` as a new path;
  record the cutover date so the two paths can be summed in reporting.
- **Maintainability** — one origin constant, one route constant; no slug string duplicated across
  files except the redirect stub.
- **Internationalization** — the marketing site is English-only today; no locale files touched.
- **Backward compatibility** — inbound links, the sitemap's previous contents, and any external
  backlink to `/self-learner` all resolve.

## 7. Acceptance Criteria

- **AC-1.** *Given* a browser at `https://lextures.com/homeschool`, *When* the page loads, *Then* the
  hero eyebrow reads "Homeschool" and no rendered text matches `/self.?learn/i`.
- **AC-2.** *Given* a browser at `https://lextures.com/self-learner`, *When* the page loads, *Then*
  the address bar ends at `/homeschool` and the audience page renders.
- **AC-3.** *Given* `curl -s https://lextures.com/self-learner/` with no JS, *Then* the response body
  contains `http-equiv="refresh"` targeting `/homeschool`, a `rel="canonical"` pointing at
  `/homeschool`, and an `<a>` to `/homeschool`.
- **AC-4.** *Given* the built `dist/sitemap.xml`, *Then* it contains `<loc>…/homeschool</loc>` and no
  `self-learner` substring.
- **AC-5.** *Given* the header "Who it's for" dropdown and the footer "Institutions" column, *Then*
  both show "Homeschool" linking to `/homeschool`.
- **AC-6.** *Given* `/get-started`, *When* the first card is clicked, *Then* the browser navigates to
  `https://self.lextures.com/` and a `sendBeacon` fires with `program` equal to the value HS.5
  accepts.
- **AC-7.** *Given* `/pricing`, *Then* the middle card is labelled "Homeschool", the FAQ contains
  "How do homeschool accounts work?", and no FAQ answer mentions "self-learner".
- **AC-8.** *Given* `npm run lint && npm run test && npm run build` in `www/`, *Then* all pass and
  `dist/self-learner/index.html` exists.

## 8. Data Model

None in this plan. The only persisted value the site produces is `onboarding_events.program`, owned
by [HS.5](HS.5-server-copy-and-onboarding-program.md).

## 9. API Surface

No new routes. One changed request body value on the existing endpoint:

```
POST /api/v1/public/onboarding/track   (unauthenticated, rate-limited, 204 No Content)
  body.program: 'k-12' | 'higher-ed' | 'self-learner' | 'school'
             → 'k-12' | 'higher-ed' | 'homeschool'   | 'school'   (+ legacy accepted)
```

Server-side acceptance of `'homeschool'` is [HS.5 FR-1](HS.5-server-copy-and-onboarding-program.md#5-functional-requirements).
The endpoint swallows insert errors by design, so a value the server rejects is **silently dropped** —
this is exactly why FR-11 gates the flip.

## 10. UI / UX

**Changed pages**

| Page | Change |
|---|---|
| `/homeschool` (was `/self-learner`) | eyebrow, hero lead, use-case card title, both CTAs |
| `/pricing` | hero sentence, card label + description, FAQ Q&A ×2 |
| `/get-started` | first path card title + description; `Path` union; analytics value |
| header / footer | audience link label + href |
| `/k-12`, `/higher-ed`, `/parents` | CTA `primaryHref` symbol only (no visible change) |
| `/self-learner` | new: redirect stub |

**Key flows**

1. Search → `/homeschool` → "Start studying" → `self.lextures.com`.
2. Old backlink → `/self-learner` → (meta refresh or JS replace) → `/homeschool`.
3. `/get-started` → "Homeschool" card → beacon → `self.lextures.com`.
4. `/pricing` → Homeschool card "Sign up" → `self.lextures.com`.

**States** — the audience page is fully static: no loading, empty, or error states. The redirect stub
has one state and MUST render its fallback link immediately (no spinner).

**Responsive** — nav label "Homeschool" (10 chars) is shorter than "Self-learners" (13), so the 52 px
dropdown and the mobile drawer both keep their current geometry.

**Accessibility annotations** — dropdown focus order unchanged (`AUDIENCE_LINKS` order preserved);
the redirect stub's fallback link is the first focusable element; `AudienceHero` keeps its `h1`.

**Copy & i18n keys** — English-only, hardcoded in TSX per the site's existing convention; strings
come verbatim from the [HS.1 copy deck](HS.1-terminology-copy-deck-and-guardrails.md#10-ui--ux).

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- External: GitHub Pages (static hosting; no redirect rules available — hence FR-3), Google
  Analytics 4 property `G-JX182Q6KKX`, Google Search Console (submit the updated sitemap).
- Internal: `www/src/lib/site-links.ts` (single source for the app origin),
  `www/scripts/prerender-courses.mjs` (sitemap + `robots.txt` + stub writer),
  `.github/workflows/pages-www.yml` (path-filtered on `www/**`, runs `npm run lint` then
  `npm run build` with `SKIP_COURSE_PRERENDER=1`).
- Emissions: `POST /api/v1/public/onboarding/track` beacon.

## 13. Dependencies & Sequencing

- Must ship after: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md) (specifically OQ-1, the
  one-word/two-word decision — it fixes the slug and cannot be changed cheaply once indexed).
- Should ship before: [HS.5](HS.5-server-copy-and-onboarding-program.md) is **not** required, thanks
  to FR-11; but the `program` flip is a follow-up commit once HS.5 is deployed.
- Must ship before: nothing blocks on it.
- Shared infra: none beyond the existing Pages workflow.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `/self-learner` drops out of the index and the segment loses organic traffic | M | H | Static meta-refresh stub + canonical (FR-3), sitemap updated (FR-4), resubmit in Search Console, monitor impressions for both paths for 60 days |
| GitHub Pages cannot 301, so link equity transfers imperfectly | H | M | Meta refresh with `content="0"` plus canonical is Google's documented fallback; accept and monitor |
| `trackOnboarding('homeschool')` ships before the server accepts it → events silently lost | M | M | FR-11 gates it behind a constant; AC-6 checks the value actually accepted |
| The narrower label reads as "K–12 only" and suppresses certification/language signups | M | M | Keep all three use-case cards; watch `/pricing` → homeschool-card CTR before/after |
| A stale internal link to `/self-learner` survives review | M | L | HS.1 guard covers `www/**`; AC-4 covers the sitemap |
| The redirect stub is clobbered by a future `vite build` | L | M | It is written by the prerender step that runs *after* `vite build` in `npm run build`; covered by AC-8 |

## 15. Rollout Plan

- Feature flag: none. A static marketing site behind a flag adds more risk than it removes.
- Sequencing: merge to `main` → `pages-www.yml` builds and deploys → verify AC-1…AC-5 on production →
  resubmit `sitemap.xml` in Search Console → (after HS.5) flip the `program` constant.
- Dogfood: `npm run dev` walkthrough of all six changed pages plus the redirect, reviewed by product
  before merge.
- GA criteria: all ACs pass on production; Search Console shows `/homeschool` indexed within 14 days.
- Rollback: revert the PR. `/self-learner` returns to being the canonical route; the stub disappears
  with it. Since no data model changes, rollback is complete and instant.

## 16. Test Plan

- **Unit** (`node --test`, `www/package.json` `test` script):
  - extend `scripts/prerender-courses.test.mjs` — `buildSitemap` includes `/homeschool` and excludes
    `self-learner`; the new stub builder emits refresh + canonical + fallback link and escapes the
    origin.
  - `src/lib/document-head.test.mjs` — if FR-13 is implemented, assert the `/homeschool` head opts.
- **Integration** — run `npm run build` with `SKIP_COURSE_PRERENDER=1` and assert `dist/homeschool`
  routing works under `npm run preview`, and that `dist/self-learner/index.html` exists.
- **End-to-end** — manual (www has no Playwright suite): load `/self-learner` with JS disabled and
  confirm the meta refresh lands on `/homeschool`.
- **Security** — confirm no interpolated value in the stub bypasses `escapeHtml`.
- **Accessibility** — axe on `/homeschool` and on the stub; keyboard-only pass through the header
  dropdown and mobile drawer.
- **Performance / load** — Lighthouse on `/homeschool`; compare against the current `/self-learner`
  report.
- **Manual exploratory** — click every internal link that pointed at the old route (7 call sites in
  FR-6) and confirm each resolves without a redirect hop.

## 17. Documentation & Training

- End-user docs: none (`www/src/docs/*.md` never mentions the segment).
- Admin/instructor docs: none.
- API reference: `program` enum note carried by HS.5.
- Internal: note the GA4 path-split cutover date in the marketing analytics doc; add the redirect
  stub to the www build's README/comment header so it is not mistaken for a stray artefact.

## 18. Open Questions

1. Depends on **[HS.1 OQ-1](HS.1-terminology-copy-deck-and-guardrails.md#18-open-questions)** — the
   slug is not reversible once indexed, so this must be answered before merge.
2. Should the header dropdown keep the audience order (`Higher education, K–12, Parents, Homeschool`)
   or promote Homeschool now that it is a named segment?
3. Does the audience page need homeschool-specific content (co-op scheduling, portfolio records,
   state reporting) in this pass, or is a label-only rebrand acceptable for v1? (HS.1 OQ-3.)
4. Do we want a `/self-learner` → `/homeschool` note in `robots.txt` or Search Console's Change of
   Address tool? (The latter is domain-level only, so likely no.)
5. Should `/pricing`'s card ordering change now that the middle card's audience is narrower?

## 19. References

- Existing files this work touches: `www/src/app.tsx`, `www/src/pages/self-learner-page.tsx`,
  `www/src/pages/{pricing,get-started,k12,higher-ed,parents}-page.tsx`,
  `www/src/components/{header,site-footer}.tsx`, `www/src/lib/{site-links,api-base}.ts`,
  `www/scripts/prerender-courses.mjs`, `www/scripts/prerender-courses.test.mjs`.
- External standards: Google Search Central — *Redirects and Google Search* (meta refresh as a
  soft-redirect signal), *Consolidate duplicate URLs* (rel=canonical), WCAG 2.1 SC 2.2.1
  (Timing Adjustable) for the meta-refresh fallback link.
- Related plans: [HS.1](HS.1-terminology-copy-deck-and-guardrails.md),
  [HS.5](HS.5-server-copy-and-onboarding-program.md),
  [MKT10 — www marketplace SEO](../../completed/marketplace/MKT10-www-marketplace-seo.md).
