# FB2 — Web Feedback Admin Page (Global Settings)

> Implementation plan. Source: Product request — in-app "Share Feedback" mechanism (2026-07-10). Follows [../_TEMPLATE.md](../_TEMPLATE.md). Consumes [FB0](./FB0-feedback-foundation-api.md).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | FB2 |
| **Section** | Feedback — In-App Feedback & Admin Review |
| **Severity** | MINOR |
| **Markets** | K12 / HE / SL |
| **Status (today)** | SHIPPED — Feedback admin page in Global Settings |
| **Estimated effort** | S (1w) |
| **Platforms** | Web (React + Tailwind) |
| **Owner (proposed)** | Platform |
| **Depends on** | [FB0](./FB0-feedback-foundation-api.md) |
| **Unblocks** | Feedback triage workflow |
| **Permission** | `global:app:rbac:manage` |

---

## 1. Problem Statement

Feedback captured by FB0 has no reader. Admins need a place inside Global Settings to browse submissions in a table, filter/search them, open one to read it in full, and move it through a triage status. This story adds a **Feedback** page to the settings area — matching the look and interaction of the existing settings panels — so the feedback loop is actually actionable.

## 2. Goals

- A new **Feedback** entry in the Global/System Settings navigation, RBAC-gated.
- A table of submissions with the columns and filters an admin needs to triage at a glance.
- A detail view (drill-in) showing the full message, submitter, context, and status controls.
- Status + internal-note editing that persists via FB0's admin API.
- Visual and interaction parity with the rest of the settings surface.

## 3. Non-Goals

- No backend/schema/endpoint work (FB0).
- No end-user submit UI (FB1/FB3).
- No cross-org analytics/dashboards or CSV export (future — §18).
- No reply-to-user / two-way messaging (future).
- No bulk-merge / duplicate detection (future).

## 4. Personas & User Stories

- **As a platform admin**, I open **Global Settings → Feedback** and see the newest submissions in a table.
- **As a platform admin**, I filter by status/category/source and search message text to find what matters.
- **As a platform admin**, I click a row to read the full feedback with who sent it and from where.
- **As a platform admin**, I set a submission to *Triaged* / *Resolved* and jot an internal note.

## 5. Functional Requirements

- **FR-1.** A **Feedback** link MUST appear in `side-nav-settings-links.tsx` within the RBAC-gated **System Settings** section (near Global platform), shown only to `global:app:rbac:manage` holders.
- **FR-2.** Route `/settings/feedback` MUST render a `FeedbackAdminPanel` via the existing settings view dispatch (`settingsViewFromPathname` → `settings.tsx`); the `SettingsView` union MUST gain a `'feedback'` member.
- **FR-3.** The panel MUST show a paginated table: columns **When**, **Submitter**, **Category**, **Source**, **Status**, **Message preview**.
- **FR-4.** The table MUST offer filters for **status**, **category**, **source**, a **date range**, and a **free-text search** box, mapped to FB0 list query params.
- **FR-5.** Clicking a row MUST open a detail view (side panel or `/settings/feedback/:id`) with the full message, submitter name/email, parsed context (route, app version, platform), timestamps, and status/note controls.
- **FR-6.** The detail view MUST let an admin change **status** and edit an **internal note**, persisting via `PATCH /api/v1/admin/feedback/{id}` and reflecting the update in the table.
- **FR-7.** The panel MUST show empty, loading, and error states consistent with other settings panels.
- **FR-8.** The message MUST render as **escaped plain text** (never HTML), including in preview and detail.
- **FR-9.** A status **badge** MUST use a consistent color scheme (e.g. new=indigo, triaged=amber, in_progress=blue, resolved=green, wont_fix=slate, archived=neutral).

## 6. Non-Functional Requirements

- **Performance** — list p95 < 600 ms; keyset/offset pagination (default 25/page); debounced search; no full-table client render.
- **Security** — server enforces `global:app:rbac:manage` on every call (client gating is UX only); no HTML injection from message/context.
- **Privacy** — feedback may contain PII; access is admin-only and audited (FB0 §FR-11); avoid logging message bodies client-side.
- **Accessibility** — WCAG 2.1 AA: semantic `<table>` with headers, sortable-column semantics, keyboard row activation, focus management into/out of the detail panel, status badge not color-only (icon/text too).
- **Reliability** — optimistic status update with rollback on failure; retry on transient errors.
- **Internationalization** — all UI strings localized; dates via existing locale-aware formatting; RTL-safe.
- **Maintainability** — new `components/settings/feedback-admin-panel.tsx` mirroring existing panels (e.g. `platform-settings-panel.tsx`, `admin-service-tokens-panel.tsx`).

## 7. Acceptance Criteria

- **AC-1.** *Given* an admin with `global:app:rbac:manage`, *Then* a **Feedback** link shows in System Settings and `/settings/feedback` renders the table.
- **AC-2.** *Given* a non-manager, *Then* the link is absent and the route/API returns forbidden (server-enforced).
- **AC-3.** *Given* submissions exist, *Then* the table lists them newest-first with When/Submitter/Category/Source/Status/preview and paginates.
- **AC-4.** *Given* a status filter of `new`, *When* applied, *Then* only `new` rows show; combined with a search term, results narrow accordingly.
- **AC-5.** *Given* a row is clicked, *Then* the detail view shows the full escaped message, submitter, parsed context, and status/note controls.
- **AC-6.** *Given* the admin sets status to *Resolved* and saves a note, *Then* `PATCH` persists, the badge/table update, and `resolved_by`/`resolved_at` are recorded (FB0).
- **AC-7.** *Given* a message containing `<script>`, *Then* it renders as inert text everywhere.
- **AC-8.** *Given* no submissions, *Then* a friendly empty state shows.

## 8. Data Model

- None (client only). Consumes FB0 admin endpoints.

## 9. API Surface

- Consumes `GET /api/v1/admin/feedback` (list + filters), `GET /api/v1/admin/feedback/{id}` (detail), `PATCH /api/v1/admin/feedback/{id}` (status/note) — all FB0 §9. No new endpoints.

## 10. UI / UX

- **Nav:** add `SideNavLink to="/settings/feedback"` with a lucide icon (`MessageSquare` / `Inbox`) in the `canManageRbac` block of `side-nav-settings-links.tsx`, active-state via `view === 'feedback'`.
- **List:** table with sticky header, status badges, relative timestamps (absolute on hover), truncated message preview, filter bar above (status/category/source selects + date range + search input). Row hover + keyboard focus.
- **Detail:** right-hand drawer or dedicated route `/settings/feedback/:id`; sections — Message (full, escaped), Submitter (avatar/name/email via existing enrollment/user components), Context (route, platform, app version, locale, submitted-at), Admin (status select, internal note textarea, Save). Show `resolved_by`/`resolved_at` when set.
- **States:** loading skeleton, empty ("No feedback yet"), error (retry), saving (disabled Save + spinner), forbidden (shouldn't reach — belt-and-suspenders message).
- **Responsive:** table becomes stacked cards on narrow screens; detail becomes full-screen sheet.
- **Copy/i18n keys:** `settings.feedback.title`, column headers, filter labels, status labels, `settings.feedback.empty`, `settings.feedback.saveNote`, `settings.feedback.statusUpdated`.
- **Accessibility:** table headers + scope, `aria-sort` if sortable, focus moves into detail on open and back to the row on close, badges carry text not just color.

## 11. AI / ML Considerations

- Out of scope. Future: a "Themes" tab summarizing clusters — depends on FB0 §11.

## 12. Integration Points

- `clients/web/src/components/layout/side-nav-settings-links.tsx` (nav link).
- `clients/web/src/components/layout/side-nav-path-utils.ts` (`SettingsView` union + `'feedback'` mapping).
- `clients/web/src/pages/lms/settings.tsx` (dispatch `activeView === 'feedback' && <FeedbackAdminPanel />`).
- New `clients/web/src/components/settings/feedback-admin-panel.tsx` (+ detail component).
- `clients/web/src/lib/api.ts` (`authorizedFetch`), permissions context (`usePermissions` / `PERM_RBAC_MANAGE`), existing enrollment/user avatar components, web i18n.

## 13. Dependencies & Sequencing

- After FB0 (admin endpoints). Independent of FB1/FB3; can develop in parallel with them.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Stored XSS from message/context | M | H | Render escaped text only; no `dangerouslySetInnerHTML` |
| Large queue is unwieldy | M | M | Server-side filters + pagination + search; bulk-archive as fast-follow |
| Client-only gating mistaken for security | L | H | Server enforces `global:app:rbac:manage` on every endpoint (FB0) |
| "Global Settings" placement ambiguity | L | L | Place under existing RBAC-gated System Settings section, matching Global platform |

## 15. Rollout Plan

- Visible to `global:app:rbac:manage` holders once FB0 endpoints ship; no separate flag needed (page is inert without data). Optionally reuse `ff_feedback` to hide the nav link pre-launch.
- Dogfood with the internal admin team on real feedback before GA.
- Rollback: remove the nav link / route (or flag off).

## 16. Test Plan

- **Unit** — nav link visibility by permission; view mapping; filter → query-param mapping; badge color/text; escaped rendering.
- **Integration** — mocked list/detail/patch; pagination; optimistic status update + rollback.
- **E2E (Playwright)** — admin navigates to Feedback, filters, opens a row, changes status + note, sees table update; non-admin can't reach it.
- **Security** — server 403 on non-admin; XSS payload inert; audit emitted.
- **Accessibility** — axe on table + detail; keyboard row activation; focus management; badge not color-only.
- **Manual** — RTL, responsive stacked cards, empty/error states.

## 17. Documentation & Training

- Admin runbook: "Triage the feedback queue" — statuses, notes, filters.
- Help-center admin doc + screenshot.

## 18. Open Questions

1. **Detail as drawer vs. route** (`/settings/feedback/:id`) — route enables deep-linking/sharing a submission; drawer is lighter. Recommend route.
2. **Dedicated `feedback:manage` permission + org-scoped view** vs. reusing `global:app:rbac:manage` (MVP). Tie to FB0 §18.2.
3. **Bulk actions** (archive/resolve multiple) — fast-follow.
4. **CSV export** of filtered feedback — likely wanted; scope later.
5. **Sortable columns** beyond default newest-first — needed?

## 19. References

- Existing files this work touches: `clients/web/src/pages/lms/settings.tsx`, `clients/web/src/components/layout/side-nav-settings-links.tsx`, `clients/web/src/components/layout/side-nav-path-utils.ts`, `clients/web/src/components/settings/platform-settings-panel.tsx` (panel pattern), `clients/web/src/lib/api.ts`.
- Related plans: [FB0](./FB0-feedback-foundation-api.md), [FB1](./FB1-web-share-feedback-button.md), [FB3](./FB3-mobile-share-feedback.md).
