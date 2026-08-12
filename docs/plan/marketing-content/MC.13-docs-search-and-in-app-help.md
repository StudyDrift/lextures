# MC.13 — Docs Search & In-App Help Integration

> Implementation plan. Source: [docs/plan/marketing-content/README.md](README.md) §Plans.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.13 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS |
| **Status (today)** | PARTIAL — `www` ships a 150 KB static search index built from files; the in-app help widget serves a **hard-coded** route→article mapping in `support_widget_http.go` whose comment still says "the help center is not yet built" |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web platform + Server platform |
| **Depends on** | MC.3, MC.7 |
| **Unblocks** | — |

---

## 1. Problem Statement

Two search surfaces read the help center today, and neither will survive the migration intact. `www`
generates `dist/docs-search-index.json` by walking `src/docs/**` — that directory is going away. The
in-app help widget calls `GET /api/v1/help/contextual-articles`, which returns a *static Go slice* of
three or four article stubs per route prefix; it has never reflected the real help center, and it
already contains stale URLs (`/docs/finding-your-course` rather than the categorised path). Once
articles live in the database with `roles`, `segments` and `relatedTo` metadata, both surfaces can be
correct — and the widget can finally answer "help me with *this* screen" from real content.

## 2. Goals

- Keep the public docs search working, sourced from the database, within the same 150 KB index budget
  and with no UX change.
- Replace the hard-coded contextual-help mapping with a real query over published help articles,
  matched by route → category/keyword mapping and filtered by the viewer's role.
- Give the in-app widget real search, not just a canned list, using the MC.3 search endpoint.
- Make it possible to read a help article inside the app (sanitized HTML) instead of bouncing to
  `lextures.com`, while keeping the external link available.
- Keep everything degradable: no network, no problem — the widget still links to the help center.

## 3. Non-Goals

- No AI answer generation over the help center (a natural follow-on, explicitly out of scope here).
- No support-ticket integration or live chat changes.
- No search analytics platform; only counters and top-queries logging.
- No redesign of the help widget or the `/docs` index page.
- No mobile-app help surface changes.

## 4. Personas & User Stories

- **As a learner or teacher in the app**, I want the help button to show articles about the screen I
  am on, so I do not search for what I cannot name.
- **As a learner**, I want to read the answer without leaving the app and losing my place.
- **As a visitor on lextures.com**, I want the docs search to keep working exactly as it does.
- **As a content expert**, I want a new help article to become findable in-app without an engineer
  editing a Go file.
- **As an SRE**, I want in-app help to fail gracefully when the content API is unavailable.

## 5. Functional Requirements

- **FR-1.** `www/scripts/generate-docs-search.mjs` MUST build its index from the content source
  (MC.7) rather than the filesystem, preserving the current item shape
  (`{title, description, category, path, headings[]}`) and the 150 KB hard cap.
- **FR-2.** The index MUST exclude `noindex` and unpublished articles and MUST include heading text
  extracted from the markdown body (same regex semantics as today).
- **FR-3.** `GET /api/v1/help/contextual-articles?route=` MUST be reimplemented to query published
  help articles instead of the static slice, returning up to 5 articles with
  `{title, url, slug, categorySlug, summary}`.
- **FR-4.** Route→content matching MUST use, in precedence order: (a) an explicit mapping table
  (`marketing.content_route_hints`) maintained by content admins, (b) the article's `related_to[]`
  paths, (c) category mapping derived from `content_categories.platform_path`, (d) full-text search
  on route-derived keywords. The first tier that yields results wins.
- **FR-5.** Results MUST be filtered by the viewer's role when the article declares `roles[]`
  (e.g. an instructor-only article is not shown to a student), using the viewer's effective role set.
- **FR-6.** The endpoint MUST remain available and non-breaking when `ff_marketing_content` is off,
  falling back to the existing static mapping so the widget never regresses.
- **FR-7.** The widget MUST gain a search box that queries
  `GET /api/v1/public/content/search?q=&kind=doc` (debounced 300 ms, max 8 results) with keyboard
  navigation of results.
- **FR-8.** Selecting an article MUST open an in-widget reader rendering sanitized HTML from
  `GET /api/v1/public/content/articles/docs/{category}/{slug}?render=html`, with a visible "Open in
  help center" external link and a Back control.
- **FR-9.** The in-widget reader MUST render content styles consistent with the app theme (light and
  dark) and MUST NOT execute any script from content (sanitizer guarantee, MC.4 FR-5).
- **FR-10.** All widget network calls MUST have a 5 s timeout and MUST degrade to: cached results →
  contextual list → "Open help center" link, with no error dialog.
- **FR-11.** Contextual results MUST be cached client-side per route for the session and server-side
  in the object cache for 5 minutes.
- **FR-12.** A route-hints admin surface (`…:admin`) MUST allow mapping an app route prefix to one or
  more articles, with a test field that previews what a given route returns.
- **FR-13.** `www`'s `/docs` search UX MUST remain unchanged: same input, same result list, same
  keyboard behaviour, same lazy index load on focus.
- **FR-14.** Search queries MUST be logged in aggregate (query text hashed or truncated, count) to
  identify content gaps, with no user identifier attached.

## 6. Non-Functional Requirements

- **Performance** — Contextual lookup p95 < 80 ms (cached), < 200 ms cold; widget search p95 < 250 ms;
  the reader fetch < 300 ms. The `www` index stays ≤ 150 KB (build fails above it, as today).
- **Security** — The in-app reader renders sanitized HTML produced by the MC.4 renderer; the widget
  must set no `dangerouslySetInnerHTML` on anything not passed through that endpoint. Search input is
  parameterized. Role filtering happens server-side; the client never receives articles it may not
  see.
- **Privacy & Compliance** — Search-query logging is aggregate and unattributed; no learner identifier
  is stored with a query. The widget makes authenticated calls for contextual articles (role
  filtering) and anonymous calls for public search.
- **Accessibility** — Widget search is a `role="searchbox"` with `aria-controls` on the result list,
  results are a listbox with arrow-key navigation and `aria-activedescendant`, result count is
  announced politely, the reader has a labelled region and a focus-managed Back control, and the
  external link states that it opens in a new tab. Content HTML retains heading ids and landmarks.
- **Scalability** — Index size grows with the help center; at 60+ articles (SEO.7's target) the
  current index shape is ~60 KB, leaving headroom. Beyond 150 KB the build fails and we move `www`
  search to the API endpoint (documented fallback plan).
- **Reliability** — Every surface degrades to a link. No help failure blocks app usage.
- **Observability** — `help_contextual_requests_total{tier}` (which matching tier answered),
  `help_search_requests_total`, `help_article_reads_total`, `help_zero_result_queries_total`;
  weekly report of top zero-result queries for the content team (feeds MC.11's gap analysis).
- **Maintainability** — Route hints are data, not code; the static Go mapping is retained only as the
  flag-off fallback and is marked deprecated.
- **Internationalization** — Contextual and search queries pass the viewer's locale; with one locale
  the behaviour is unchanged. Reader content sets `lang` on the container.
- **Backward compatibility** — The endpoint's response shape is extended (new optional fields), not
  changed; the existing widget continues to work if only the server ships.

## 7. Acceptance Criteria

- **AC-1.** *Given* the DB-sourced build, *when* `generate-docs-search.mjs` runs, *then*
  `dist/docs-search-index.json` contains one entry per published, indexable help article with
  headings, and is ≤ 150 KB.
- **AC-2.** *Given* the `/docs` page, *when* a visitor searches, *then* behaviour and results are
  identical to the file-sourced build for the same corpus (parity harness covers the artefact).
- **AC-3.** *Given* a viewer on `/courses/{code}/assignments`, *when* the widget opens, *then* the
  contextual list contains help articles from the "Assessment & quizzes"/"Courses & content"
  categories or explicit hints — not the static default list.
- **AC-4.** *Given* an article with `roles: [instructor]`, *when* a student opens the widget, *then*
  that article is absent from the response (server-filtered).
- **AC-5.** *Given* `ff_marketing_content` is off, *when* the widget requests contextual articles,
  *then* the legacy static mapping is returned and the widget behaves exactly as today.
- **AC-6.** *Given* a query typed in the widget, *when* 300 ms pass, *then* up to 8 results appear,
  are keyboard-navigable, and the count is announced.
- **AC-7.** *Given* a selected result, *when* it opens, *then* the article renders in the widget with
  correct theme styling, a Back control, and an external "Open in help center" link.
- **AC-8.** *Given* content containing `<script>` injected at the source, *when* the reader renders
  it, *then* nothing executes (sanitizer test).
- **AC-9.** *Given* the content API is unreachable, *when* the widget opens, *then* it shows the
  cached or contextual list and the help-center link, with no error dialog and no console exception.
- **AC-10.** *Given* a route hint mapping `/gradebook → grading/using-rubrics`, *when* a viewer on
  `/gradebook` opens the widget, *then* that article is first in the list.
- **AC-11.** *Given* 50 searches with no results, *when* the weekly report runs, *then* those queries
  appear in the zero-result report available to the content team.
- **AC-12.** *Given* axe runs on the widget (open, searching, reading), *then* there are zero
  serious/critical violations in both themes.

## 8. Data Model

Migration `483_marketing_content_route_hints.sql` (indicative number):

```sql
CREATE TABLE marketing.content_route_hints (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    route_prefix TEXT NOT NULL,
    article_id   UUID NOT NULL REFERENCES marketing.content_articles (id) ON DELETE CASCADE,
    position     INTEGER NOT NULL DEFAULT 100,
    created_by   UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (route_prefix, article_id)
);
CREATE INDEX idx_mc_route_hints_prefix ON marketing.content_route_hints (route_prefix);

CREATE TABLE marketing.content_search_queries (
    day        DATE NOT NULL,
    query      TEXT NOT NULL,
    surface    TEXT NOT NULL CHECK (surface IN ('widget', 'www', 'workspace')),
    hits       INTEGER NOT NULL DEFAULT 0,
    results    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (day, query, surface)
);
```

**Backfill:** seed `content_route_hints` from the existing static `articleMapping` in
`support_widget_http.go` so day-one behaviour is at least as good as today.

## 9. API Surface

| Verb | Path | Auth | Change |
|---|---|---|---|
| GET | `/api/v1/help/contextual-articles?route=&locale=` | session | reimplemented; response gains `categorySlug`, `summary`, `tier` |
| GET | `/api/v1/public/content/search?q=&kind=doc&limit=` | anonymous | MC.3 (consumed here) |
| GET | `/api/v1/public/content/articles/docs/{category}/{slug}?render=html` | anonymous | MC.3 (consumed here) |
| GET/POST | `/api/v1/admin/marketing/route-hints` · DELETE `/{id}` | `…:view` / `…:admin` | new |
| GET | `/api/v1/admin/marketing/route-hints/preview?route=` | `…:admin` | returns what the widget would show |
| GET | `/api/v1/admin/marketing/search-gaps?days=30` | `…:view` | zero-result query report |

```ts
type ContextualArticle = {
  title: string; url: string; slug: string
  categorySlug: string | null; summary: string
  tier: 'hint' | 'related' | 'category' | 'search' | 'fallback'
}
```

## 10. UI / UX

**In-app widget** (`clients/web/src/components/layout/help-widget.tsx`)

1. Open → contextual list for the current route (unchanged position and trigger).
2. Type in the new search field → results replace the contextual list; Escape clears back to
   contextual.
3. Select a result → in-widget reader: title, category breadcrumb, body, "Open in help center ↗",
   Back.
4. States: loading skeleton (3 rows), empty contextual ("No articles for this screen yet — search or
   open the help center"), empty search ("No results for *x*"), offline ("Help center unavailable —
   open lextures.com/docs"), reader error (falls back to the external link).
5. Responsive: the widget panel is already constrained; the reader scrolls internally with a sticky
   header; on narrow viewports it becomes full-screen.
6. Accessibility: search input labelled; listbox pattern with `aria-activedescendant`; reader region
   `role="document"`-like container with an accessible name equal to the article title; focus moves
   to the reader heading on open and returns to the result on Back; external link announces "opens in
   a new tab".
7. Copy/i18n: `help.widget.search.label`, `.results.count`, `.empty.contextual`, `.empty.search`,
   `.offline`, `.openInHelpCenter`, `.back`.

**`www` /docs search** — unchanged UI; only the index source changes.

**Route-hints admin** — a small table under the Marketing Content workspace (Settings tab): route
prefix, article, position, plus a "Preview for route" input showing the resolved list and the tier
that produced it.

## 11. AI / ML Considerations

Not AI-touching. Deliberate sequencing note: this plan produces exactly the substrate an AI help
answer would need (published corpus, sanitized text extraction, search endpoint, route context). If
that is built later it must cite the underlying articles, must not answer from model memory, and must
be disclosed per `internal/aidisclosure` — but it is out of scope here.

## 12. Integration Points

- **Server:** `internal/httpserver/support_widget_http.go` (reimplemented handler + legacy fallback),
  `internal/service/marketingcontent` (contextual resolution), `internal/repos/marketingcontent`,
  `internal/objectcache`, new admin routes.
- **Client:** `components/layout/help-widget.tsx`, `lib/marketing-content-api.ts` (public search
  client), new route-hints admin panel.
- **`www`:** `scripts/generate-docs-search.mjs`, `src/pages/docs-index.tsx` (unchanged behaviour).
- **Related:** `components/feature-help/*` (feature help dock) — out of scope, but the same contextual
  resolution could feed it later; noted, not built.

## 13. Dependencies & Sequencing

- Must ship after: MC.3 (search + render endpoints), MC.7 (index generation from the source).
- Must ship before: nothing blocking; ideally before MC.15 so the widget improvement lands with the
  cutover.
- Shared infra: object cache.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Contextual matching returns irrelevant articles | **H** | M | Tiered resolution with an explicit hint table first; a preview tool for admins; zero-result and low-relevance reporting feeding content work |
| In-app rendering of external content is an XSS surface | M | **H** | Only the sanitized `?render=html` output is rendered; sanitizer corpus test (MC.4); CSP already restricts inline script in the app |
| Search index outgrows 150 KB | M | M | Build fails loudly (existing behaviour); documented fallback is to switch `www` search to the API endpoint with a client-side debounce |
| Widget becomes slow on every route change | M | M | Session cache per route; contextual fetch only when the widget opens (already the case) |
| Role filtering leaks instructor-only guidance | L | M | Server-side filtering only (AC-4); tested with student/instructor/admin fixtures |
| Legacy static mapping rots | M | L | Marked deprecated; seeded into route hints; removed in MC.15 |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content` gates the DB-backed path; the legacy mapping is the
  automatic fallback (FR-6), so the widget is never worse than today.
- **Sequencing:** index generation from source → contextual endpoint reimplementation (behind flag) →
  route-hints table + seed → widget search → in-widget reader → admin panel.
- **Dogfood:** staff use the staging app for a week; the zero-result report drives the first batch of
  new help articles (SEO.7 work).
- **GA criteria:** all ACs; contextual relevance spot-checked on 10 key routes; axe clean; no p95
  regression on widget open.
- **Rollback:** flag off — contextual falls back to the static mapping, widget search hides, `www`
  search returns to file-sourced index (with `WWW_CONTENT_SOURCE=files`).

## 16. Test Plan

- **Unit** — tier resolution order; role filtering; route-prefix matching (longest prefix wins);
  index item shape and size cap; query normalization and logging aggregation.
- **Integration** — contextual endpoint against seeded articles for several routes and roles; cache
  behaviour and invalidation on publish; admin preview endpoint agreement with the live endpoint.
- **End-to-end (Playwright)** — `e2e/tests/help-widget-content.spec.ts`: open widget on a course
  route → contextual articles present → search → open reader → external link present → offline
  simulation degrades gracefully.
- **Security** — XSS through article body into the reader; authz on admin route-hint endpoints; role
  filtering cannot be bypassed by query parameters; search input injection.
- **Accessibility** — axe across widget states in both themes; keyboard-only script (open, search,
  arrow to result, open, back, close, focus restore); screen-reader announcement checks.
- **Performance** — contextual and search latency; index generation time; widget open time with cold
  cache.
- **Manual exploratory** — 10 representative app routes checked for contextual relevance; a new
  article published and confirmed searchable in-app without a deploy.

## 17. Documentation & Training

- Help article: "Getting help inside Lextures" (explains the widget, search and the reader).
- Internal: how route hints work and how to add one; how to read the zero-result report.
- Update the deprecated note in `support_widget_http.go` and remove the "help center is not yet
  built" comment — it has been untrue since SEO.7.

## 18. Open Questions

1. Should the widget reader support images? (Proposed: yes, via the localised media URLs; verify
   sizing inside the narrow panel.)
2. Do we keep the static `www` search index at all once the API search exists, or serve both?
   (Proposed: keep the static index — it works offline-ish, needs no API, and is already budgeted.)
3. Should `feature-help` docks reuse contextual resolution? (Proposed: evaluate after this ships;
   they currently use their own JSON.)
4. Do we want per-role default article sets curated by content admins (e.g. "new teacher" starter
   set)? (Likely valuable; deferred.)

## 19. References

- Files this work touches: `server/internal/httpserver/support_widget_http.go`,
  `server/internal/service/marketingcontent/*`, `server/migrations/483_*`,
  `clients/web/src/components/layout/help-widget.tsx`, `www/scripts/generate-docs-search.mjs`,
  `www/src/pages/docs-index.tsx`.
- Related plans: [MC.3](MC.3-public-content-read-api.md),
  [MC.4](MC.4-content-rendering-and-validation-service.md),
  [MC.7](MC.7-www-build-time-content-integration.md),
  [MC.11](MC.11-editorial-workflow-and-governance.md); SEO.7 (help center expansion).
- Standards: WAI-ARIA Authoring Practices — Combobox/Listbox pattern; WCAG 2.1 AA.
