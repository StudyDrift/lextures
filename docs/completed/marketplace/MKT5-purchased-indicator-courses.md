# MKT5 — "Purchased" Indicator & My Purchases

> Implementation plan. Source: [docs/plan/marketplace/README.md](README.md). Part of the Marketplace epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT5 |
| **Section** | Marketplace |
| **Severity** | MINOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | SHIPPED |
| **Estimated effort** | S (1w) |
| **Owner (proposed)** | Web platform team |
| **Depends on** | MKT4 |
| **Unblocks** | MKT6 (mobile parity) |

---

## 1. Problem Statement

After MKT4, a learner can claim a free course or buy a paid one, gaining an entitlement and an enrollment. But on the **Courses page** (`courses.tsx`) a marketplace-acquired course looks identical to one an instructor added them to. The user requirement is explicit: *when a user has "purchased" a course — even if it's free — indicate that it's a purchased course on the courses page.* This story surfaces that acquisition state as a badge and adds a lightweight "My purchases" view so learners can see everything they've acquired through the marketplace.

## 2. Goals

- Show a **"Purchased"** badge on course cards for courses the viewer acquired via the marketplace (free claim or paid), using the entitlement as the source of truth.
- Extend the Courses list API to return an acquisition indicator per course without extra round-trips.
- Add a "My purchases" filter/section (reusing the existing entitlements/billing surfaces) so learners can review acquisitions and access receipts.
- Keep the badge distinct from existing catalog status pills (enrolled/invitation/etc.).

## 3. Non-Goals

- The buy/claim flow (MKT4) or storefront (MKT3).
- Mobile courses list (MKT6).
- Full billing history UI (already exists via 15.3 `/me/billing`, `billing-settings.tsx`); this story links to it, doesn't rebuild it.
- Instructor-side sales dashboards (15.8 revenue share).

## 4. Personas & User Stories

- **As a learner**, I want my Courses page to mark which courses I obtained through the marketplace so that I can tell them apart from courses I was added to.
- **As a learner**, I want a "Purchased/Free" cue even for free claims so that it's clear those were my choice.
- **As a learner**, I want to find all my marketplace acquisitions in one place with links to receipts.

## 5. Functional Requirements

- **FR-1.** The Courses list endpoint MUST include, per course, an acquisition indicator: `acquiredViaMarketplace: boolean` and `acquisitionSource: 'free' | 'stripe' | 'comp' | null`, derived from active `course_purchase` entitlements for the viewer.
- **FR-2.** The Courses page MUST render a **"Purchased"** badge on cards where `acquiredViaMarketplace = true`. Copy MAY differentiate "Purchased" (paid) vs "Free" (claimed) via `acquisitionSource`, but both count as purchased per the requirement.
- **FR-3.** The badge MUST be visually and semantically distinct from the existing `CourseCatalogStatusPill` (enrolled/invitation) and MUST NOT replace status information — both may show.
- **FR-4.** The indicator MUST be computed in a single batched query over the viewer's courses (no per-card request).
- **FR-5.** The system MUST add a "My purchases" view (route `/me/purchases` or a filter on the Courses page) listing marketplace-acquired courses with price paid, date, source, and a link to the receipt (`/me/billing`) for paid items.
- **FR-6.** When the marketplace flag is off, the badge and "My purchases" view MUST be hidden, but existing entitlement records remain (data preserved).
- **FR-7.** A refunded entitlement (`status='refunded'`) MUST NOT show "Purchased" (only active entitlements count).
- **FR-8.** The badge MUST render in all Courses page views (cards, list, gallery, table) consistently.

## 6. Non-Functional Requirements

- **Performance** — One extra batched query (or a `LEFT JOIN`) on the Courses list; p95 unchanged (< 10 ms added). No N+1.
- **Security** — Indicator computed for the requesting user only; entitlement rows never leaked cross-user.
- **Privacy & Compliance** — Amount-paid shown only in "My purchases" to the owner; not on shared/instructor views. Financial data governed by 15.13 retention.
- **Accessibility** — Badge has a text label (not color-only) and an accessible name distinct from status; WCAG 2.1 AA; sufficient contrast in light/dark.
- **Scalability** — Batched query bounded by the viewer's course count.
- **Reliability** — Indicator failure degrades gracefully (hide badge, don't block the list).
- **Observability** — Emit `my_purchases_view`, and count badge renders for adoption insight.
- **Maintainability** — Reuse the existing courses-list repo/query and `courses-api-schemas.ts`; add a field, not a new endpoint.
- **Internationalization** — Badge/label + "My purchases" copy externalised; date/price localized.
- **Backward compatibility** — Additive response field; older clients ignore it.

## 7. Acceptance Criteria

- **AC-1.** *Given* I claimed a free course, *When* I open Courses, *Then* that course shows a "Purchased" (or "Free") badge.
- **AC-2.** *Given* I bought a paid course, *When* I open Courses, *Then* it shows "Purchased" with the paid source.
- **AC-3.** *Given* an instructor added me to a course (no entitlement), *When* I open Courses, *Then* no purchased badge appears.
- **AC-4.** *Given* my purchase was refunded, *When* I open Courses, *Then* no purchased badge appears.
- **AC-5.** *Given* I open "My purchases", *Then* I see all active marketplace acquisitions with source, date, price, and a receipt link for paid ones.
- **AC-6.** *Given* the flag is off, *When* I open Courses, *Then* no purchased badges and no "My purchases" entry appear.
- **AC-7.** *Given* the badge shows, *When* inspected by a screen reader, *Then* it announces "Purchased" distinctly from the enrollment status pill.

## 8. Data Model

No schema changes. Reads `billing.user_entitlements` (active `course_purchase`, with `acquisition_source`) joined to the viewer's courses. Add a repo helper `billing.PurchasedCourseMap(userID, courseIDs) map[uuid.UUID]string` (courseID → acquisitionSource) for batched lookup, reused from MKT3's ownership query.

## 9. API Surface

- Extend the existing Courses list endpoint (backing `courses-api.ts`) response: each course gains `acquiredViaMarketplace: boolean` and `acquisitionSource: string | null`. No new route.
- New (optional) `GET /api/v1/me/purchases` → `{ purchases: [{ courseCode, title, priceCents, currency, source, acquiredAt, receiptUrl? }] }`, gated by `courseMarketplaceOff`. (Alternatively reuse `/me/entitlements` filtered to `course_purchase` + join course metadata.)
- Update `courses-api-schemas.ts` (Zod) with the new fields.
- OpenAPI: update Courses list + add `/me/purchases`.

## 10. UI / UX

- **Courses page** (`clients/web/src/pages/lms/courses.tsx`) — add a "Purchased" badge alongside `CourseCatalogStatusPill` in each view renderer (cards/list/gallery/table). New small component `CoursePurchasedBadge` (or extend the pill set) with an icon (e.g. `BadgeCheck` / `ShoppingBag`) + label.
- **My purchases** — either a filter chip on the Courses page ("Purchased") or a route `/me/purchases` linked from the sidenav "Account" group / billing. Lists acquisitions with receipt links.
- **States** — badge only when acquired; "My purchases" empty state ("You haven't enrolled through the marketplace yet" + link to `/marketplace`); error hides badge.
- **Responsive** — badge wraps/truncates gracefully; doesn't overflow small cards.
- **Accessibility** — text label, contrast, distinct accessible name.
- **Copy & i18n** — `courses.badge.purchased`, `courses.badge.free`, `purchases.title`, `purchases.empty`, `purchases.receipt`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **Internal** — `clients/web/src/pages/lms/courses.tsx`, `lib/courses-api.ts` + `courses-api-schemas.ts`, server Courses list handler/repo, `repos/billing` (`PurchasedCourseMap`). Reuse `billing-settings.tsx` / `/me/billing` for receipts.
- **Feature context** — `ffCourseMarketplace` gates badge + view.

## 13. Dependencies & Sequencing

- **After** — MKT4 (entitlements + enrollments exist to indicate).
- **Before** — MKT6 (mobile parity mirrors this indicator).
- **Shared infra** — none new.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Badge clutters cards / competes with status pill | M | L | Distinct placement + design review; secondary styling |
| Extra join slows Courses list | L | M | Batched map / single LEFT JOIN; index on `(user_id, course_id)` from MKT1 |
| "Free" badge confuses (looks like it's free to enroll) | M | L | Label "Purchased" primary; "Free" only in detail/My purchases, or "Enrolled — Free" wording |
| Refunded still showing purchased | L | M | Filter `status='active'` only (AC-4 test) |

## 15. Rollout Plan

- **Flag** — `ffCourseMarketplace` (MKT1). Badge/view appear when on.
- **Sequencing** — after MKT4; ships as a small follow-up.
- **Dogfood** — verify badges after internal free/paid acquisitions.
- **GA criteria** — badges correct for free/paid/added/refunded; My purchases lists correctly.
- **Rollback** — flag off hides indicators; data intact.

## 16. Test Plan

- **Unit** — `PurchasedCourseMap` batching; badge visibility logic (active vs refunded vs none).
- **Integration** — Courses list returns correct `acquiredViaMarketplace`/`acquisitionSource`; `/me/purchases` lists active only.
- **End-to-end (Playwright)** — claim free → badge; buy → badge; instructor-added → no badge; refund → badge disappears; My purchases content + receipt link.
- **Security** — no cross-user entitlement leakage; amounts only on owner's view.
- **Accessibility** — axe; screen-reader distinguishes badge from status pill; contrast in light/dark.
- **Performance** — added query cost measured; no N+1.
- **Manual** — all four Courses views show the badge consistently; RTL.

## 17. Documentation & Training

- **Learner docs** — "Finding your purchased/enrolled courses" + receipts.
- **API reference** — Courses list new fields + `/me/purchases`.

## 18. Open Questions

1. Separate labels "Purchased" vs "Free"/"Enrolled — Free", or a single "Purchased" for both? (Requirement says even free counts as purchased; default: single "Purchased" badge, with source visible in My purchases.)
2. "My purchases" as a Courses-page filter chip or a dedicated route? (Default: filter chip on Courses + link from billing; dedicated route optional.)
3. Should `comp` (complimentary/admin-granted) show as "Purchased"? (Default: yes — it's a marketplace acquisition; label neutral "Purchased.")
4. Do we surface the badge on instructor/roster views of a learner? (Default: no — owner-only.)

## 19. References

- Existing files: `clients/web/src/pages/lms/courses.tsx` (card renderers ~L245–880, `CourseCatalogStatusPill`), `lib/courses-api.ts`, `lib/courses-api-schemas.ts`, `pages/lms/billing-settings.tsx`, `repos/billing/entitlements.go`.
- Related plans: [MKT4](../../completed/marketplace/MKT4-course-purchase-entitlement-flow.md), [MKT3](../../completed/marketplace/MKT3-marketplace-discovery-web.md), [MKT6](MKT6-marketplace-mobile.md).
