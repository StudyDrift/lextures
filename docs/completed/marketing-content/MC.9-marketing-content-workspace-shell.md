# MC.9 — Marketing Content Workspace: Navigation, Gating & Content List

> Completed implementation. Source: [docs/plan/marketing-content/README.md](../../plan/marketing-content/README.md) §Plans.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MC.9 |
| **Section** | MC — Marketing Content Platform |
| **Severity** | MAJOR |
| **Markets** | K12 / HE / HS (internal staff surface) |
| **Status (today)** | COMPLETE — guarded workspace, navigation, filters, actions, and content list shipped |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web platform |
| **Depends on** | MC.2 |
| **Unblocks** | MC.10, MC.11, MC.13 |

---

## 1. Problem Statement

This is the plan where the feature becomes visible. The API can create and publish articles, but a
content expert has nowhere to go: no link in the global navigation, no list of what exists, no way to
see what is a draft, what is live, what is overdue for review. Without the workspace shell — the
nav entry, the route, the permission gate and a genuinely useful list — the rest of the program is an
API nobody can use.

## 2. Goals

- Add a **Marketing Content** link to the global side navigation that appears only when
  `ff_marketing_content` is on **and** the viewer holds `global:app:marketing-content:view`.
- Ship a content list that answers the four questions a content team actually asks: what is live,
  what is in progress, what is scheduled, and what is overdue for review.
- Make the list fast, filterable, keyboard-navigable and screen-reader-sound, using the existing UI
  primitives and semantic tokens — no hand-rolled buttons, dialogs or colour literals.
- Establish the workspace layout (header, tabs, filters, table, bulk actions) that MC.10, MC.11 and
  MC.13 extend rather than redesign.

## 3. Non-Goals

- No editor — [MC.10](MC.10-article-editor.md) owns article creation and editing (this plan's "New
  article" button routes there).
- No editorial calendar, review queue or freshness dashboard — [MC.11](MC.11-editorial-workflow-and-governance.md).
- No media library UI — [MC.5](MC.5-marketing-media-library.md) / MC.10.
- No changes to the existing admin console IA or any other nav entry.
- No mobile app surface (iOS/Android) for content authoring.

## 4. Personas & User Stories

- **As a content expert**, I want a Marketing Content link in the same place I find everything else,
  so I do not need a bookmark or a URL from a colleague.
- **As a content expert**, I want to see every article with its status, category, author and last
  update, so I can pick up where I left off.
- **As an editor**, I want to filter to "in review" or "overdue for review", so my queue is one click
  away.
- **As a teacher or student**, I want to never see this link, because it is not for me.
- **As a platform admin**, I want turning the flag on to be safe — nobody gains access until I grant
  the permission.

## 5. Functional Requirements

- **FR-1.** A `SideNavLink` labelled **Marketing Content** MUST be added to the administration
  section of `clients/web/src/components/layout/side-nav-admin-links.tsx`, routing to
  `/admin/marketing-content`, using the `Newspaper` icon (distinct from `Megaphone`, used by
  Broadcasts).
- **FR-2.** The link MUST render only when `ffMarketingContent === true` **and**
  `allows('global:app:marketing-content:view')`, and MUST NOT render while permissions are loading.
- **FR-3.** `ffMarketingContent` MUST be added to `PlatformFeatures`, its defaults, and the
  `GET /api/v1/platform/features` mapping in `platform-features-context.tsx`, defaulting to `false`.
- **FR-4.** New permission constants MUST be exported from `clients/web/src/lib/rbac-api.ts`:
  `PERM_MARKETING_CONTENT_VIEW`, `_AUTHOR`, `_REVIEW`, `_PUBLISH`, `_ADMIN`.
- **FR-5.** The route `/admin/marketing-content` MUST be registered in `app.tsx` with a lazily loaded
  page (`lazy-pages.ts`), and MUST render a "not available" state (not a crash, not a blank page)
  when the flag is off or the permission is missing — direct URL entry must be safe.
- **FR-6.** The list MUST support tabs/segments: **All**, **Blog**, **Help center**, and status
  filters (`Draft`, `In review`, `Changes requested`, `Scheduled`, `Published`, `Archived`).
- **FR-7.** The list MUST show per row: title (link), kind, category, status pill (with live status
  from MC.8), author, reviewer, last updated (relative + absolute on hover/focus), quality score, and
  review-due indicator when overdue.
- **FR-8.** The list MUST support text search (server-side `q`), sorting by updated/published/title,
  and cursor pagination with a "Load more" control (not infinite scroll).
- **FR-9.** Filters and search MUST be reflected in the URL query string so a filtered view is
  shareable and survives refresh and back/forward.
- **FR-10.** A **New article** action MUST be present for `…:author` holders, offering "Blog post" or
  "Help article" and routing to the editor.
- **FR-11.** Row actions (kebab menu, using the UI `Menu` primitive) MUST include: Open, Preview,
  Duplicate (if MC.10 ships it), Publish/Unpublish (permission-gated), Archive, Copy public URL.
- **FR-12.** Bulk selection MUST support archive and publish for permitted users, with a confirmation
  dialog that names the count and lists affected paths.
- **FR-13.** A site-status strip (MC.8 FR-11) MUST appear above the list showing the last build state
  and a manual **Rebuild site** action for `…:publish` holders.
- **FR-14.** The page MUST show correct empty states: no content at all ("No articles yet — create
  your first"), no results for a filter ("No articles match these filters" + clear-filters action),
  and a permission-limited state for view-only users (no create/publish affordances rendered at all).
- **FR-15.** All data access MUST go through a new `clients/web/src/lib/marketing-content-api.ts`
  module (typed against the generated OpenAPI types), never ad-hoc `fetch` in components.
- **FR-16.** The page MUST be added to the command palette (`components/command-palette`) as
  "Marketing Content" for permitted users.

## 6. Non-Functional Requirements

- **Performance** — First meaningful list render < 1.5 s on a warm cache; list requests paginate at
  50 rows; no layout shift when status pills load (reserve space). The page is lazily loaded so it
  costs nothing for users without access.
- **Security** — Client gating is convenience only; every action is authorized server-side (MC.2).
  Direct URL entry without permission shows the not-available state and makes no privileged request.
  No content is rendered from HTML — list cells are text.
- **Privacy & Compliance** — Displays staff names (author/reviewer/updated-by), which is ordinary
  workplace processing; no learner data.
- **Accessibility** — WCAG 2.1 AA. The table uses a real `<table>` with `<caption>`, `scope`d
  headers, and a live region announcing result counts after filtering. Status is text + icon, never
  colour alone. Focus order follows visual order; the kebab menu uses the UI `Menu` primitive with
  full keyboard support; bulk-selection state is announced. Filter controls are labelled and grouped
  in a `<fieldset>`/`role="group"` with an accessible name.
- **Scalability** — Cursor pagination; server-side filtering and search; virtualization is *not*
  needed below 500 rows and is explicitly deferred.
- **Reliability** — Failed list loads show a retry affordance and preserve filters; a failed row
  action shows an inline error without losing selection.
- **Observability** — Client analytics events `marketing_content.list_viewed`,
  `.filter_applied`, `.row_action` (action name only, no content), routed through the existing
  analytics module.
- **Maintainability** — Page under `pages/admin/marketing-content/`; components under
  `components/marketing-content/`; file budgets respected; design tokens only (`bg-surface-raised`,
  `text-fg-muted`, status `*-surface`/`*-fg`); UI primitives from `components/ui`.
- **Internationalization** — All copy through i18n keys `marketingContent.*`; dates and relative
  times formatted with the viewer's locale/timezone; the layout must tolerate RTL.
- **Backward compatibility** — Purely additive. With the flag off, no bundle chunk is fetched and no
  nav entry exists.

## 7. Acceptance Criteria

- **AC-1.** *Given* `ffMarketingContent` is false, *when* any user loads the app, *then* no Marketing
  Content link is rendered and `/admin/marketing-content` shows the not-available state.
- **AC-2.** *Given* the flag is true and the user lacks `…:view`, *when* they load the app, *then* the
  link is absent and direct navigation shows the not-available state without firing a list request.
- **AC-3.** *Given* the flag is true and the user holds `…:view`, *when* they open the app, *then*
  the link appears in the administration section and navigates to the workspace.
- **AC-4.** *Given* the workspace with 120 articles, *when* it loads, *then* the first 50 render with
  status, author and updated columns, and "Load more" fetches the next page.
- **AC-5.** *Given* a filter selection, *when* applied, *then* the URL query updates, the result count
  is announced in a live region, and a refresh restores the same view.
- **AC-6.** *Given* a view-only user, *when* the list renders, *then* no New article, publish, archive
  or bulk controls are present in the DOM.
- **AC-7.** *Given* an `…:author` user, *when* they click New article and choose Help article, *then*
  they land in the editor with `kind=doc` preselected.
- **AC-8.** *Given* an article whose `review_due_on` has passed, *when* the list renders, *then* the
  row shows an "Overdue" indicator with accessible text, and the "Overdue" filter includes it.
- **AC-9.** *Given* keyboard-only navigation, *when* tabbing through the page, *then* every control is
  reachable, the kebab menu opens with Enter/Space and closes with Escape restoring focus, and axe
  reports zero serious/critical violations.
- **AC-10.** *Given* a failed list request, *when* it errors, *then* an inline error with a Retry
  button appears and filters are preserved.
- **AC-11.** *Given* the site-status strip, *when* the last build failed, *then* it shows the failure
  with a link to the run and (for `…:publish`) a Rebuild action.
- **AC-12.** *Given* the design-system checks, *when* CI runs, *then* `npm run tokens:purity`,
  `npm run ds:coverage` and `npm run contrast:check` pass for the new files.

## 8. Data Model

No database changes. Client-side view model only:

```ts
type ArticleRow = {
  id: string; kind: 'blog'|'doc'; slug: string; path: string; title: string
  categorySlug: string | null; status: Status; liveStatus: LiveStatus
  authorSlug: string; authorName: string; reviewerName: string | null
  qualityScore: number | null; reviewDueOn: string | null; overdue: boolean
  updatedAt: string; updatedByName: string | null; publishedAt: string | null
}
type ListQuery = { kind?, status?, category?, author?, q?, overdue?, sort?, cursor?, limit? }
```

Persisted UI state: filter/sort in the URL; column preferences are **not** persisted in v1.

## 9. API Surface

Consumes MC.2 and MC.8 only:

```
GET  /api/v1/admin/marketing/articles?kind=&status=&category=&author=&q=&overdue=&sort=&cursor=&limit=
POST /api/v1/admin/marketing/articles                     (New article → editor)
POST /api/v1/admin/marketing/articles/{id}/transition     (row/bulk publish, archive)
GET  /api/v1/admin/marketing/builds                       (status strip)
POST /api/v1/admin/marketing/builds                       (Rebuild site)
GET  /api/v1/platform/features                            (ffMarketingContent)
GET  /api/v1/me/permissions                               (existing permissions payload)
```

New client module `clients/web/src/lib/marketing-content-api.ts` exporting typed functions and DTOs
generated from OpenAPI (`npm run openapi:types`). No component calls `fetch` directly.

## 10. UI / UX

**Layout**

```
┌ Marketing Content ─────────────────────────────── [ New article ▾ ] ┐
│ Site: Last build succeeded 12 min ago · [Rebuild site]              │
├─────────────────────────────────────────────────────────────────────┤
│ [All] [Blog] [Help center]      🔍 search      Status ▾  Author ▾  … │
├─────────────────────────────────────────────────────────────────────┤
│ ☐ Title                      Status      Category   Author  Updated │
│ ☐ Rethinking assessment…     ● Live      —          Chase   2d  ⋯   │
│ ☐ Finding your course        ◐ In review Courses    Sam     4h  ⋯   │
│ ☐ Coupons for creators       ○ Draft     Marketplace Chase  1h  ⋯   │
└─────────────────────────────────────────────────────────────────────┘
```

**Key flows**

1. Open workspace → default tab **All**, sorted by Updated desc.
2. Filter to In review → URL updates → count announced → open an item → editor (MC.10).
3. New article → choose kind → editor with kind preselected.
4. Row kebab → Preview opens the in-app preview (MC.10) in a new tab.
5. Select rows → Bulk archive → confirmation naming count and paths → result toast with undo-less
   summary (archive is reversible via the row action, which the toast states).

**States** — loading skeleton (8 rows), empty (no content), empty (filters), error (retry),
permission-limited (read-only), offline (banner + disabled actions).

**Responsive** — below `md`, the table collapses to a card list: title, status pill, meta line
(kind · category · author · updated), kebab. Filters move into a bottom-sheet.

**Accessibility annotations** — `<table>` with `<caption class="sr-only">Marketing content
articles</caption>`; sortable headers are `<button>`s inside `<th>` with `aria-sort`; the selection
checkbox column has a header checkbox labelled "Select all on this page"; the result-count live
region is `aria-live="polite"`, `aria-atomic="true"`; the status pill exposes text
(`Live`, `Scheduled for 12 May, 09:00`).

**Copy & i18n keys** — `marketingContent.title`, `.nav.label`, `.new.blog`, `.new.doc`,
`.filters.status`, `.empty.noContent`, `.empty.noResults`, `.error.loadFailed`,
`.notAvailable.title`, `.notAvailable.body`, `.status.*`, `.overdue`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Client modules touched:** `components/layout/side-nav-admin-links.tsx` (nav entry),
  `context/platform-features-context.tsx` (flag), `lib/rbac-api.ts` (permission constants),
  `app.tsx` + `lazy-pages.ts` (route), `components/command-palette/*` (palette entry),
  new `pages/admin/marketing-content/*`, `components/marketing-content/*`,
  `lib/marketing-content-api.ts`.
- **Server:** `platform_features.go` must include `ffMarketingContent` in its response mapping.
- **Design system:** `components/ui/*` primitives; `docs/design-tokens.md` tokens only.
- **Analytics:** existing client analytics module.

## 13. Dependencies & Sequencing

- Must ship after: MC.2 (list/transition API). MC.8's build endpoints are optional — the status strip
  degrades to hidden when they are absent.
- Must ship before: MC.10 (editor is reached from here), MC.11, MC.13.
- Shared infra: none.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Nav clutter — the admin section is already long | **H** | M | Place it in the administration group next to Notices/Email templates; it appears only for permission holders, which is a small audience |
| Client-side gating mistaken for security | M | H | Every endpoint is server-authorized (MC.2); AC-2 asserts no privileged request is made without permission |
| Design-system drift (hand-rolled table controls) | M | M | `ds:coverage` and `tokens:purity` CI checks; review requires UI primitives |
| Filter state in URL conflicts with future saved views | L | L | Query-param names chosen to match the API; saved views deferred |
| Table a11y regressions as columns grow | M | M | axe in CI on this route; screen-reader script in the manual checklist |

## 15. Rollout Plan

- **Feature flag:** `ff_marketing_content` (still OFF in production). Enabled in staging.
- **Sequencing:** flag plumbing to the client → permission constants → route + not-available state →
  list page → nav entry last (so the link never points at a half-built page).
- **Dogfood:** content team + docs owner use the staging workspace against MC.6-imported content.
- **GA criteria:** all ACs; axe clean; the content team can find and filter every existing article
  without help.
- **Rollback:** flag off — the link disappears and the route shows the not-available state; no data
  implications.

## 16. Test Plan

- **Unit (vitest)** — nav visibility matrix (flag × permission × loading); URL⇄filter state sync;
  row action permission gating; empty/error state selection; date formatting.
- **Integration** — list page against a mocked API: pagination, search debounce, sort, retry on
  error, bulk action confirmation payloads.
- **End-to-end (Playwright)** — `e2e/tests/marketing-content-workspace.spec.ts`: flag off → no link;
  flag on + no permission → no link and safe direct URL; flag on + author → link, list, filter,
  create routes to editor; bulk archive updates rows.
- **Security** — assert no privileged request fires without permission; assert row actions absent
  from the DOM (not merely hidden) for view-only users.
- **Accessibility** — axe on the route in both themes; keyboard-only run through filters, table, menu
  and bulk actions; screen-reader script (NVDA/VoiceOver) for the table and live region.
- **Performance** — bundle-size check that the chunk is lazy and under budget; render timing with 500
  rows loaded across pages.
- **Manual exploratory** — QA checklist: RTL locale, narrow viewport, long titles, missing
  category/author, 0/1/many results.

## 17. Documentation & Training

- Help article (internal-facing, written in the workspace): "Finding your way around Marketing
  Content".
- Admin docs: how to grant marketing-content permissions (RBAC screen walkthrough).
- `docs/guides/component-library.md` — no changes expected; note the new page as an example of table
  + filter composition if useful.

## 18. Open Questions

1. Should the nav entry live under Administration or as a top-level item for content-only staff whose
   entire job is this workspace? (Proposed: Administration group; revisit if content-only accounts
   become common — a top-level entry for users whose *only* permission is marketing-content would be
   friendlier.)
2. Do we need saved views ("my drafts", "overdue help articles")? (Proposed: v2; URL sharing covers
   most of it.)
3. Should the list show a thumbnail (hero image) column? (Proposed: no — it costs a column and adds
   requests; the editor shows it.)
4. Do we surface the quality score in the list, or is it noise outside the editor? (Proposed: show it
   as a compact numeric column, sortable — it is how editors find weak pages.)

## 19. References

- Files this work touches: `clients/web/src/components/layout/side-nav-admin-links.tsx`,
  `clients/web/src/context/platform-features-context.tsx`, `clients/web/src/lib/rbac-api.ts`,
  `clients/web/src/app.tsx`, `clients/web/src/lazy-pages.ts`,
  `clients/web/src/pages/admin/marketing-content/*`, `clients/web/src/lib/marketing-content-api.ts`,
  `server/internal/httpserver/platform_features.go`.
- Precedents: `pages/admin/AdminBanners.tsx` (admin CRUD page shape), `side-nav-admin-links.tsx`
  (flag + permission gating), `components/ui/*` (primitives).
- Standards: WCAG 2.1 AA (1.3.1, 1.4.1, 2.1.1, 2.4.3, 4.1.2, 4.1.3).
- Related plans: [MC.10](MC.10-article-editor.md), [MC.11](MC.11-editorial-workflow-and-governance.md),
  [MC.8](../../completed/marketing-content/MC.8-publish-pipeline-and-scheduling.md).
