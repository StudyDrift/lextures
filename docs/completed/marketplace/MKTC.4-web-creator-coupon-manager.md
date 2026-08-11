# MKTC.4 — Web: Creator Coupon Manager

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the MKTC Course Coupon Codes epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKTC.4 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | HS (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Commerce / Growth squad (web) |
| **Depends on** | MKTC.2 |
| **Unblocks** | MKTC.7 |

---

## 1. Problem Statement

MKTC.2 exposes coupon CRUD, but a course creator is not going to `curl` it. The marketplace section
of course settings today
(`clients/web/src/pages/lms/course-marketplace-settings-section.tsx`) is a single form with a
listing toggle, a fee and a currency. Creators need a table of their codes with live usage, a
create dialog that makes the four decisions (code, amount, window, seat cap) obvious, and — the
request that motivates the whole surface — a **Copy share link** button on every row so a code can
be handed to an audience without anyone typing it.

## 2. Goals

- A **Coupons** panel inside the existing course Marketplace settings, visible only to users who
  hold `course:{code}:item:create`.
- A table of coupons showing code, discount, window, usage (`12 / 50`), status, and per-row actions.
- A create dialog with live preview of the resulting learner price, and clear validation.
- A per-row **Copy share link** action that copies the server-provided `shareUrl` (and, for public
  courses, offers the marketing-site link).
- A redemptions drawer per coupon so the creator can see who used a code.
- Full WCAG 2.1 AA conformance, semantic design tokens only, and the shared `components/ui`
  primitives — no hand-rolled buttons, dialogs or menus.

## 3. Non-Goals

- Learner-side coupon entry (MKTC.5).
- Mobile creator management (MKTC.6 scopes what iOS/Android get).
- Bulk code generation/import, per-recipient unique codes, CSV export of redemptions (MKTC.7 §18
  tracks export as a candidate follow-up).
- Editing a code's value after creation — the API forbids it (MKTC.2 FR-4); the UI offers
  archive-and-recreate instead.
- Coupon performance analytics beyond raw counts (MKTC.7).

## 4. Personas & User Stories

- **As a course creator**, I want to create a code in under 30 seconds with an obvious preview of
  what a learner will pay, so that I do not fear making a pricing mistake.
- **As a course creator**, I want a copy button that gives me a link with the code baked in, so that
  I can paste it into a newsletter, a slide, or a DM.
- **As a course creator**, I want to see "34 of 50 claimed" at a glance so that I know when to
  extend a promotion.
- **As a course creator whose code leaked**, I want a one-click pause that stops redemptions without
  destroying my history.
- **As a co-teacher**, I want the same panel the owner sees.
- **As a screen-reader user**, I want the table, the dialog and the copy confirmation to be
  announced properly, so that the panel is usable without sight.

## 5. Functional Requirements

- **FR-1.** A **Coupons** panel MUST render inside the course Marketplace settings surface, below
  the existing fee form, only when `ffCourseCoupons` is on (from
  `usePlatformFeatures()`), the course is marketplace-listed **or** priced, and the viewer holds
  `courseItemCreatePermission(courseCode)`. Otherwise it MUST be **absent**, not disabled.
- **FR-2.** When the course price is `0`, the panel MUST render an explanatory empty state
  ("Coupons apply to paid courses. Set a fee above to use them.") and MUST NOT offer create.
- **FR-3.** The table MUST show, per coupon: `code` (monospace, selectable), discount
  ("25% off" / "$10.00 off"), window ("Always" | "Until Mar 3" | "Mar 1 – Mar 3", in the viewer's
  locale and timezone), usage (`12 / 50` or `12 / ∞` with a text alternative "12 of unlimited"),
  status badge, and an actions menu.
- **FR-4.** Each row MUST carry a **Copy share link** action rendered as a visible icon button in
  the row (not hidden behind the overflow menu), because it is the primary workflow. Activating it
  copies `shareUrl` to the clipboard, swaps the icon to a check for ~2 s, and announces
  "Share link copied" in a polite live region.
- **FR-5.** When `publicShareUrl` is non-null, the copy control MUST become a split control: the
  primary action copies the in-app link; the secondary menu offers "Copy public site link". When it
  is null, only the single button renders.
- **FR-6.** Clipboard failures (permission denied, insecure context) MUST fall back to selecting the
  URL in a focusable read-only input inside a dialog with instructions, and MUST NOT show a silent
  failure.
- **FR-7.** A **New coupon** dialog MUST collect: code (auto-uppercasing as typed), discount type
  (percent | fixed amount) via a segmented control, the amount, optional start and end dates
  (date + time, defaulting to the viewer's timezone and converted to UTC on submit), optional total
  redemption limit, optional per-learner limit (default 1), and an optional internal note.
- **FR-8.** The dialog MUST offer a **Generate** action that produces a random 8-character code from
  an unambiguous alphabet (no `O`/`0`/`I`/`1`), and MUST still validate it server-side.
- **FR-9.** The dialog MUST show a **live preview** — "Learners pay **$30.00** (was $40.00)" —
  recomputed on every input change from the course's current price using the same rounding rules,
  and MUST show a warning when the result would be free ("This code makes the course free").
- **FR-10.** Client-side validation MUST mirror the server: code shape `^[A-Z0-9][A-Z0-9_-]{3,31}$`,
  percent in `(0, 100]`, fixed amount `> 0` and `<=` course price, end after start. Server errors
  (409 duplicate, 422 rules) MUST be surfaced on the offending field, not only as a toast.
- **FR-11.** The actions menu MUST offer **Edit** (window, limits, note, status only), **Pause** /
  **Resume** (status `disabled` ↔ `active`), **Archive** (confirm dialog naming the code and the
  fact that redemptions are retained), and **View redemptions**.
- **FR-12.** **View redemptions** MUST open a drawer/dialog listing learner name, email, status,
  charged amount, discount, and date, paginated with the API cursor, with an empty state.
- **FR-13.** Archived coupons MUST be hidden by default behind a "Show archived" toggle that adds
  `?includeArchived=true`.
- **FR-14.** All list mutations MUST optimistically update or refetch, and MUST surface failures via
  `toastMutationError` consistent with the sibling fee form.
- **FR-15.** The panel MUST have loading (skeleton), empty ("No coupon codes yet" + primary create
  button), error (retry) and offline states per the UX.12 state contract.
- **FR-16.** All copy MUST be i18n keys under `course.settings.coupons.*` in
  `clients/web/public/locales/en/common.json`, with the reason vocabulary from MKTC.1 mapped to
  `marketplace.coupon.reason.*` (shared with MKTC.5).
- **FR-17.** All UI MUST use `clients/web/src/components/ui/*` primitives (`Table`, `Dialog`,
  `Menu`, `Button`, `IconButton`, `SplitButton`, `Field`, `Input`, `Select`, `SegmentedControl`,
  `DatePicker`, `Badge`, `EmptyState`, `Sheet`, `Pagination`) and semantic design tokens only —
  no raw Tailwind palette literals (`npm run tokens:purity` and `npm run ds:coverage` must pass).
- **FR-18.** A new API module `clients/web/src/lib/course-coupons-api.ts` MUST own every HTTP call
  and DTO type; components MUST NOT call `authorizedFetch` directly.

## 6. Non-Functional Requirements

- **Performance** — Panel adds one request on mount (list). Table renders 100 rows without
  virtualization under 16 ms/frame. The create dialog is code-split with the rest of the course
  settings route; no new eager bundle weight on the dashboard.
- **Security** — The panel is permission-gated client-side for UX only; the server is the authority
  (MKTC.2 FR-7). Codes are rendered only to authorized staff. Clipboard writes use the async
  Clipboard API with the documented fallback; no `document.execCommand` on a hidden element.
- **Privacy & Compliance** — The redemptions drawer shows learner name/email to course staff who
  already have roster access. No export in this story, so no new bulk-egress path.
- **Accessibility (WCAG 2.1 AA)** — Table uses real `<table>` semantics with `<caption>` and
  `scope` on headers. The copy button has a discernible name including the code
  ("Copy share link for LAUNCH25"), and the copied confirmation is announced via
  `aria-live="polite"` (not by changing the button's accessible name mid-interaction). The dialog
  traps focus, is labelled by its heading, restores focus to the invoking control, and closes on
  Escape. Every field has a programmatic label, `aria-describedby` help text, and `aria-invalid` +
  `role="alert"` error text. The usage column's `∞` has a text alternative. Status colour is never
  the only signal — badges carry text. Target sizes ≥ 24×24 CSS px (WCAG 2.2 §2.5.8, per UX.5).
- **Scalability** — Tens of coupons per course expected; pagination is only needed on redemptions.
- **Reliability** — Double-submit protection on the create dialog (disabled + pending state); a
  duplicate-code 409 is rendered as a field error, never a lost form.
- **Observability** — Client events `coupon_manager_opened`, `coupon_created`,
  `coupon_share_link_copied{target: app|public}`, `coupon_paused`, `coupon_archived`,
  `coupon_redemptions_viewed`, through the existing web analytics helper.
- **Maintainability** — New files: `course-coupons-panel.tsx`, `course-coupon-create-dialog.tsx`,
  `course-coupon-row-actions.tsx`, `course-coupon-redemptions-drawer.tsx` under
  `clients/web/src/pages/lms/`, plus `lib/course-coupons-api.ts`. Each file ≤ 500 LOC (TS budget);
  kebab-case filenames.
- **Internationalization** — All strings externalized; dates formatted with the existing locale
  helpers and shown with an explicit timezone hint; RTL verified (the copy-icon affordance and the
  table must mirror); currency formatted with `formatMarketplacePrice`.
- **Backward compatibility** — Additive panel; the existing fee form is untouched apart from the
  panel being rendered after it.

## 7. Acceptance Criteria

- **AC-1.** *Given* I am a course owner with `ffCourseCoupons` on and a $40 course, *When* I open
  course settings → Marketplace, *Then* I see a Coupons panel with an empty state and a
  "New coupon" button.
- **AC-2.** *Given* the flag is off, *When* I open the same page, *Then* the Coupons panel is not in
  the DOM.
- **AC-3.** *Given* I am a student, *When* I somehow reach the settings route, *Then* the panel is
  absent and no coupon request is issued.
- **AC-4.** *Given* the create dialog, *When* I type `launch25`, *Then* the field shows `LAUNCH25`,
  and *when* I set 25 %, *Then* the preview reads "Learners pay $30.00 (was $40.00)".
- **AC-5.** *Given* I submit a code that already exists, *When* the server returns 409, *Then* the
  code field shows an inline error naming the conflict and the dialog stays open with my input
  intact.
- **AC-6.** *Given* a coupon row, *When* I activate **Copy share link**, *Then* the clipboard
  contains `{origin}/marketplace/{slug}?coupon=LAUNCH25`, the icon shows a check for ~2 s, and a
  polite live region announces "Share link copied".
- **AC-7.** *Given* the course is public, *When* I open the copy control's secondary menu, *Then*
  "Copy public site link" copies the marketing-site URL with the same parameter.
- **AC-8.** *Given* clipboard permission is denied, *When* I activate copy, *Then* a dialog appears
  with the URL pre-selected in a read-only input and instructions.
- **AC-9.** *Given* a coupon with 12 of 50 used, *When* the table renders, *Then* the usage cell
  reads `12 / 50` and its accessible text is "12 of 50 claimed".
- **AC-10.** *Given* an uncapped coupon, *When* the table renders, *Then* the usage cell shows the
  count with "of unlimited" available to assistive technology.
- **AC-11.** *Given* I pause a coupon, *When* the request succeeds, *Then* the status badge changes
  to "Paused" without a full page reload and the row's actions offer "Resume".
- **AC-12.** *Given* I archive a coupon, *When* I confirm, *Then* it disappears from the default
  view and reappears under "Show archived" with an "Archived" badge.
- **AC-13.** *Given* a coupon with redemptions, *When* I open "View redemptions", *Then* I see rows
  with learner, amount charged, discount and date, and paging works to the end.
- **AC-14.** *Given* the course price is 0, *When* the panel renders, *Then* it explains that
  coupons need a paid course and the create button is absent.
- **AC-15.** *Given* an axe scan of the panel, the create dialog and the drawer, *When* it runs in
  CI, *Then* there are zero violations; keyboard-only traversal reaches every control in a logical
  order and focus returns to the invoker on dialog close.
- **AC-16.** *Given* `npm run tokens:purity`, `npm run ds:coverage`, `npm run lint`,
  `npm run typecheck` and `npm run test`, *When* CI runs, *Then* all pass.

## 8. Data Model

No client-side persistence beyond React state. Types mirror the MKTC.2 DTOs in
`clients/web/src/lib/course-coupons-api.ts` and are validated against the regenerated
`lib/generated/openapi-types.ts` (`npm run openapi:types`). No local storage, no cache keys beyond
the component's own fetch state.

## 9. API Surface

Consumes MKTC.2 only:

```ts
// clients/web/src/lib/course-coupons-api.ts
export async function fetchCourseCoupons(courseCode: string, opts?: { includeArchived?: boolean }): Promise<CourseCoupon[]>
export async function createCourseCoupon(courseCode: string, body: CreateCouponBody): Promise<CourseCoupon>
export async function updateCourseCoupon(courseCode: string, couponId: string, body: UpdateCouponBody): Promise<CourseCoupon>
export async function archiveCourseCoupon(courseCode: string, couponId: string): Promise<CourseCoupon>
export async function fetchCouponRedemptions(courseCode: string, couponId: string, opts?: { cursor?: string; limit?: number }): Promise<{ rows: CouponRedemptionRow[]; nextCursor: string }>
export function generateCouponCode(length?: number): string   // client-side, server-validated
export function describeDiscount(c: CourseCoupon, locale: string): string
export function previewCouponPrice(priceCents: number, currency: string, draft: CouponDraft): { chargedCents: number; free: boolean }
```

No new server routes. No WebSocket events.

## 10. UI / UX

**Placement.** A new `<CourseCouponsPanel />` rendered by
`clients/web/src/pages/lms/course-marketplace-settings-section.tsx` immediately below the existing
form, as its own card with heading "Coupon codes" and a one-line description.

**Primary flows**

1. *Create* — Click **New coupon** → dialog → enter/generate code → pick percent or fixed → set
   amount → (optional) window, total limit, per-learner limit, note → live preview updates →
   **Create** → dialog closes, row appears at the top, focus moves to the new row's copy button, a
   toast confirms.
2. *Share* — Click the row's copy icon → clipboard receives `shareUrl` → icon becomes a check →
   live region announces. (Public courses: split control with "Copy public site link".)
3. *Pause* — Row menu → **Pause** → status badge flips to "Paused"; learners applying the code now
   get `reason: inactive`.
4. *Archive* — Row menu → **Archive** → confirm dialog ("Archive LAUNCH25? Learners can no longer
   use it. Past redemptions are kept.") → row leaves the default view.
5. *Inspect* — Row menu → **View redemptions** → sheet with the paginated table → Escape or Close
   returns focus to the menu trigger.

**States** — Loading: three skeleton rows with `aria-busy`. Empty: `EmptyState` with a ticket icon,
"No coupon codes yet", body explaining share links, and the create button. Error: `ErrorState` with
retry. Offline: the panel keeps the last data with an inline "Reconnecting…" banner and disables
mutations (UX.12 contract).

**Mobile / responsive** — Below `sm`, the table collapses to a stacked card list: code as the
heading, discount and window as description-list rows, usage as a meter with text, and the copy
button as a full-width secondary button. No horizontal page scroll; if the table is kept at
intermediate widths it lives in an `overflow-x:auto` container.

**Accessibility annotations** — Focus order: panel heading → Show archived toggle → New coupon →
table rows in DOM order (copy button, then actions menu) → pagination. The actions menu is the
shared `Menu` primitive (roving tabindex, Escape closes, arrow keys move). The create dialog is
`Dialog` (focus trap, `aria-labelledby`, Escape). Live region: one polite `<span class="sr-only">`
owned by the panel, reused for copy confirmations and mutation results.

**Copy & i18n keys** (`common.json`)

```
course.settings.coupons.title            "Coupon codes"
course.settings.coupons.description      "Create codes that discount this course at checkout."
course.settings.coupons.new              "New coupon"
course.settings.coupons.generate         "Generate"
course.settings.coupons.code             "Code"
course.settings.coupons.discount         "Discount"
course.settings.coupons.window           "Active"
course.settings.coupons.usage            "Claimed"
course.settings.coupons.usageOf          "{used} of {limit} claimed"
course.settings.coupons.usageUnlimited   "{used} of unlimited claimed"
course.settings.coupons.status           "Status"
course.settings.coupons.copyLink         "Copy share link for {code}"
course.settings.coupons.copyPublicLink   "Copy public site link"
course.settings.coupons.copied           "Share link copied"
course.settings.coupons.copyFallback     "Copy this link"
course.settings.coupons.percentOff       "{percent}% off"
course.settings.coupons.amountOff        "{amount} off"
course.settings.coupons.preview          "Learners pay {now} (was {before})"
course.settings.coupons.previewFree      "This code makes the course free."
course.settings.coupons.pause            "Pause"
course.settings.coupons.resume           "Resume"
course.settings.coupons.archive          "Archive"
course.settings.coupons.archiveConfirm   "Archive {code}? Learners can no longer use it. Past redemptions are kept."
course.settings.coupons.redemptions      "View redemptions"
course.settings.coupons.showArchived     "Show archived"
course.settings.coupons.emptyTitle       "No coupon codes yet"
course.settings.coupons.emptyBody        "Create a code, then copy its share link to hand it out."
course.settings.coupons.freeCourse       "Coupons apply to paid courses. Set a fee above to use them."
course.settings.coupons.error.duplicate  "A coupon with this code already exists on this course."
course.settings.coupons.error.save       "Could not save the coupon."
course.settings.coupons.error.load       "Could not load coupon codes."
```

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — the browser Clipboard API (async `navigator.clipboard.writeText`) with the dialog
  fallback.
- **Internal** —
  `clients/web/src/pages/lms/course-marketplace-settings-section.tsx` (mount point),
  `clients/web/src/pages/lms/course-coupons-panel.tsx` (**new**),
  `clients/web/src/pages/lms/course-coupon-create-dialog.tsx` (**new**),
  `clients/web/src/pages/lms/course-coupon-row-actions.tsx` (**new**),
  `clients/web/src/pages/lms/course-coupon-redemptions-drawer.tsx` (**new**),
  `clients/web/src/lib/course-coupons-api.ts` (**new**),
  `clients/web/src/lib/marketplace-price.ts` (reuse formatting + minor-unit helpers),
  `clients/web/src/context/platform-features-context.tsx` (`ffCourseCoupons`),
  `clients/web/src/lib/lms-toast.ts`, `clients/web/src/components/ui/*`,
  `clients/web/public/locales/*/common.json`.
- **Events** — analytics only; no server events originate here.

## 13. Dependencies & Sequencing

- **Must ship after** — MKTC.2 (API + flag).
- **Must ship before** — MKTC.7 (which flips the flag on and writes the help docs with screenshots
  from this UI).
- **Shared infra** — none beyond the existing web app; `ffCourseCoupons` must be togglable in the
  admin platform settings panel (delivered in MKTC.2 FR-19).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Creator mis-sets a percent and gives the course away | M | H | Live preview (FR-9) plus an explicit "This code makes the course free" warning and a confirm step in the dialog when charged would be 0 |
| Clipboard blocked in the user's browser/context | M | M | Documented fallback dialog with a selectable input (FR-6, AC-8) |
| Timezone confusion on the window | H | M | Dates entered and displayed in the viewer's timezone with an explicit tz label; converted to UTC on submit; round-trip test |
| Table becomes unusable on phones | M | M | Stacked-card layout below `sm` (§10) with an e2e viewport test |
| Copy button hidden in an overflow menu, killing the core workflow | M | M | FR-4 mandates a visible in-row control; e2e asserts it is reachable without opening a menu |
| Raw Tailwind literals slip in | M | L | `npm run tokens:purity` + `ds:coverage` in CI (AC-16) |
| File exceeds the 500-LOC TS budget | M | L | Four files by responsibility from the start |

## 15. Rollout Plan

- **Feature flag** — `ffCourseCoupons` (default OFF). The panel is absent when off.
- **Sequencing** — API module + types → panel + table → create dialog → row actions → redemptions
  drawer → i18n for all shipped locales → e2e.
- **Dogfood** — enable on staging for the internal demo course; have a non-engineer create a code
  and share it; observe whether the copy button is found without prompting.
- **GA criteria** — axe clean; keyboard-only walkthrough passes; the create→copy→redeem loop works
  end-to-end with MKTC.5; all locale files carry the new keys (missing-key check green).
- **Rollback** — flag off, or revert the panel mount (one line in the settings section).

## 16. Test Plan

- **Unit** (Vitest) — `describeDiscount` for percent/fixed across locales and currencies;
  `previewCouponPrice` parity with the server's rounding, including JPY and the clamp-to-free
  boundary; `generateCouponCode` alphabet and length; local↔UTC date conversion; API module error
  mapping (409 → duplicate field error).
- **Integration** (React Testing Library) — panel renders for permitted roles only; empty/loading/
  error states; create dialog validation for each rule; server 409/422 surfaces on the right field;
  pause/resume/archive optimistic updates and rollback on failure; redemptions pagination; copy
  handler success and fallback paths (mocked clipboard).
- **End-to-end** (Playwright, `e2e/tests/course-coupons.spec.ts`) — creator creates a percent
  coupon, copies its share link, and the clipboard content is asserted; the same link is opened in
  a learner context and the discount shows (joint with MKTC.5); pause makes the code stop working;
  archive hides the row; mobile viewport renders the stacked layout.
- **Security** — non-permitted roles never trigger a coupon request (network assertion); no coupon
  data in the DOM for a student.
- **Accessibility** — axe on panel, dialog and drawer; screen-reader script (NVDA + VoiceOver):
  navigate the table by row, hear the usage cell correctly, activate copy and hear the
  confirmation, open and close the dialog with focus restored; RTL rendering check.
- **Performance / load** — render 100 coupons, assert no layout thrash and a single network call.
- **Manual exploratory** — create a coupon while the fee form has unsaved changes; toggle the flag
  off with the panel open; very long note text; a 32-character code's table layout.

## 17. Documentation & Training

- **End-user docs** — none (this surface is staff-only).
- **Admin / instructor docs** — complete the "Create a coupon code" help page started in MKTC.2 with
  screenshots of the panel, the dialog and the share-link copy, plus a "what each field means"
  table and the archive-vs-pause distinction.
- **API reference** — no change (MKTC.2 owns it).
- **Internal runbook** — add "creator says the copy button does nothing" (clipboard permission,
  insecure origin) to `docs/runbooks/coupons.md`.

## 18. Open Questions

1. Should the panel live inside the Marketplace settings card or become its own left-nav settings
   tab once the course has many coupons? (Proposed: inside Marketplace settings; revisit if
   creators routinely exceed ~20 codes.)
2. Should "Pause" be a row-level switch rather than a menu item, given it is the emergency action?
   (Proposed: menu item now, promote to a switch if support data shows urgency.)
3. Do creators need a duplicate/"clone this coupon" action? (Proposed: defer; archive-and-recreate
   plus Generate covers it.)
4. Should the redemptions drawer offer CSV export? (Deferred to MKTC.7 §18; adds a bulk-egress path
   that needs a privacy review.)
5. Should the preview use the *saved* course price or the unsaved value in the fee form above?
   (Proposed: the saved price, with a hint when the fee form is dirty.)

## 19. References

- Existing files: `clients/web/src/pages/lms/course-marketplace-settings-section.tsx` (mount point
  and the existing form's conventions), `clients/web/src/lib/marketplace-price.ts`,
  `clients/web/src/lib/lms-toast.ts`, `clients/web/src/lib/courses-api.ts:3472`
  (`courseItemCreatePermission`), `clients/web/src/components/ui/index.ts`,
  `clients/web/src/context/platform-features-context.tsx`.
- Standards: WCAG 2.1 AA (+ 2.2 target size per UX.5), [docs/design-tokens.md](../../design-tokens.md),
  [docs/guides/component-library.md](../../guides/component-library.md).
- Related plans: [MKTC.2](MKTC.2-creator-coupon-management-api.md),
  [MKTC.5](../../plan/marketplace/MKTC.5-web-learner-coupon-entry-and-url-codes.md),
  [UX.12 loading/empty/error/offline states](../../plan/ui-ux/UX.12-loading-empty-error-offline-states.md),
  [MKT2](MKT2-course-marketplace-listing-settings.md).
