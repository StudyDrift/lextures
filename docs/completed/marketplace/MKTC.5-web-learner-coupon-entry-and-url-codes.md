# MKTC.5 — Web: Learner Coupon Entry, URL Codes & Storefront Handoff

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the MKTC Course Coupon Codes epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKTC.5 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | HS (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Commerce / Growth squad (web + www) |
| **Depends on** | MKTC.3 |
| **Unblocks** | MKTC.6, MKTC.7 |

---

## 1. Problem Statement

A coupon that a learner cannot enter is worth nothing. The marketplace detail page
(`clients/web/src/pages/marketplace/marketplace-course-page.tsx`) shows one price and one CTA that
goes straight to Stripe. There is nowhere to type a code, nothing reads `?coupon=` from the URL, and
the marketing site's enroll handoff (`www/src/lib/marketplace-api.ts:195` →
`/explore/{slug}?ref=www-courses`) drops any extra query parameters. A code also has to survive the
sign-in redirect: the most common share-link recipient is a logged-out person who clicks a link in a
newsletter. This story makes both entry paths work and keeps the code attached from first click
through to the receipt.

## 2. Goals

- A coupon field on the marketplace course detail page that previews the discounted total before
  purchase, with typed, translated error messages.
- `?coupon=CODE` read from the URL on every storefront entry point, applied automatically, and
  reflected in the visible price.
- The code **survives sign-in / sign-up** and the www → app handoff, so a share link works for a
  logged-out recipient.
- 100 %-off codes route through the free-grant response straight into the course, with no checkout
  screen.
- The applied code is visible on the success page and in the receipt so the learner can see what
  they saved.

## 3. Non-Goals

- Creator management UI (MKTC.4).
- Mobile (MKTC.6).
- Anonymous server-side validation of codes on www — the marketing site carries the code forward but
  does not validate it (MKTC.3 §18 Q1).
- Coupon entry on the subscription/billing settings checkout (`plan=monthly|annual`).
- Any new server route; this story is a pure client of MKTC.3.

## 4. Personas & User Stories

- **As a learner with a code from a newsletter**, I want the link to open the course with the
  discount already applied, so that I never type anything.
- **As a learner who was given a code verbally**, I want a clearly labelled field to type it into
  and immediate confirmation of the new total.
- **As a logged-out learner clicking a share link**, I want the discount still applied after I sign
  up, so that the promotion is not lost in the sign-in redirect.
- **As a learner whose code is expired**, I want to be told exactly that, in my language, and still
  be able to buy at full price if I want.
- **As a learner with a 100 %-off code**, I want to land in the course immediately.
- **As a screen-reader user**, I want the price change and the coupon result announced, not silently
  re-rendered.

## 5. Functional Requirements

- **FR-1.** The marketplace course detail page MUST render a **coupon entry** control for paid,
  unowned courses when `ffCourseCoupons` is on: a labelled text input (auto-uppercasing) plus an
  **Apply** button, collapsed behind a "Have a coupon code?" disclosure when no code is present and
  expanded automatically when one is.
- **FR-2.** Applying a code MUST call `POST /api/v1/marketplace/courses/{slug}/coupon/preview` and,
  on `applied:true`, update the displayed price to show the discounted amount as primary with the
  list price struck through, reusing `MarketplacePriceBadge`'s existing `listPriceCents` treatment.
- **FR-3.** On `applied:false`, the page MUST render the translated reason inline
  (`marketplace.coupon.reason.*`) next to the field with `aria-invalid` and `role="alert"`, keep the
  full price displayed, and leave the CTA functional at full price.
- **FR-4.** A **Remove** affordance MUST clear an applied code, restore the full price, clear the
  URL parameter, and return focus to the coupon input.
- **FR-5.** On mount, the page MUST read `?coupon=` from the URL (case-insensitively, normalized to
  upper case) and apply it automatically, announcing the outcome in a polite live region.
  Alternatively the detail request MAY pass `?coupon=` and use MKTC.3 FR-18's embedded result to
  avoid a second round trip; the visible outcome MUST be identical either way.
- **FR-6.** When an applied code is present, the CTA MUST pass `couponCode` to
  `checkoutMarketplaceCourse` / `claimMarketplaceCourse`, and MUST show the discounted price in its
  label ("Buy — $30.00").
- **FR-7.** When checkout returns the free-grant shape (`grantedFree:true`), the page MUST navigate
  into the course exactly as the free-claim path does today, with a toast confirming the code was
  applied.
- **FR-8.** When checkout returns `422` with a coupon reason (the code lapsed between preview and
  purchase), the page MUST clear the applied discount, show the reason, restore the full price, and
  require a second explicit click to buy at full price — never silently charge more than previewed
  (MKTC.3 §18 Q2).
- **FR-9.** The deep-link routes `/marketplace/:slug/claim` and `/marketplace/:slug/checkout`
  (`marketplace-purchase-action-page.tsx`) MUST forward `?coupon=` into the API call, so a share
  link may point directly at the purchase action.
- **FR-10.** A pending coupon MUST be persisted per course in `sessionStorage` under
  `lextures.coupon.{slug}` when the page loads with a `?coupon=` parameter, and re-read after an
  auth redirect, so a logged-out learner who signs in keeps the discount. It MUST be cleared on
  successful purchase, on **Remove**, and when the learner is found to already own the course.
  Rationale for `sessionStorage` over the 30-day cookie used for affiliate refs
  (`AFFILIATE_REF_COOKIE`): a coupon is a per-visit intent, and a stale cross-session code produces
  confusing prices weeks later.
- **FR-11.** The sign-in / sign-up flow MUST preserve the full return URL including `?coupon=` (the
  existing `redirect`/`next` handling), verified by test rather than assumed.
- **FR-12.** `www` MUST accept `?coupon=` on `/courses/{slug}` and forward it through
  `enrollHandoffUrl` as `{APP_ORIGIN}/marketplace/{slug}?ref=www-courses&coupon=CODE`. The www
  price display MUST remain the list price (no anonymous validation) but the enroll panel MUST show
  a neutral line — "Coupon CODE will be applied at checkout" — so the link's promise is visible.
- **FR-13.** `www` MUST NOT call any coupon API and MUST NOT echo an unvalidated code into HTML
  without escaping; the code MUST be rendered as text only, truncated to 32 characters.
- **FR-14.** The `/explore/{slug}` public catalog page MUST forward `?coupon=` to the marketplace
  detail route when the learner clicks through, alongside the existing affiliate `?ref=` handling.
- **FR-15.** `/checkout/success` MUST display the applied code and the saved amount when present
  (read from the URL parameters the server adds to `successURL`), and MUST clear the stored pending
  coupon for that course.
- **FR-16.** `/checkout/cancel` MUST keep the applied code (so the learner can retry) and SHOULD
  notify the server so the reservation is released eagerly (MKTC.3 FR-17).
- **FR-17.** All coupon copy MUST be i18n keys shared with MKTC.4 where the vocabulary overlaps
  (`marketplace.coupon.*`), present in every shipped locale including the RTL locale.
- **FR-18.** All new UI MUST use `components/ui` primitives and semantic tokens; the coupon input
  MUST be a `Field` + `Input` pair, the disclosure the shared `Disclosure`, and the apply control a
  `Button` with a pending state.

## 6. Non-Functional Requirements

- **Performance** — Auto-apply from a URL adds at most one request on mount, or zero when the
  detail endpoint carries the result (FR-5). Apply p95 < 300 ms perceived, with the button in a
  pending state throughout. No additional bundle beyond the marketplace route chunk.
- **Security** — The client never computes or sends a price. The code is sent as an opaque string.
  Codes are read from the URL and echoed back into the DOM as text only (FR-13) — no `innerHTML`,
  no attribute injection into `href`. `sessionStorage` (not `localStorage`) bounds the lifetime of
  a code found in a URL. Apply requests inherit MKTC.3's per-user rate limit; the UI MUST disable
  the button and show a "Too many attempts, try again shortly" message on `429`.
- **Privacy & Compliance** — No new personal data. A coupon code in a URL is not personal data, but
  it is retained in browser history; the app MUST replace the URL parameter with the applied state
  (`history.replaceState`) after applying, so the code does not linger in the address bar after
  purchase.
- **Accessibility (WCAG 2.1 AA)** — The price element is an `aria-live="polite"` region so a
  discount is announced when applied. The coupon field has a visible label, help text via
  `aria-describedby`, `aria-invalid` on failure, and `role="alert"` error text. The disclosure is a
  real button with `aria-expanded`. The CTA's accessible name includes the price it will charge.
  Focus is never trapped or lost across apply/remove. Strike-through list price carries a text
  alternative ("was $40.00") rather than relying on visual styling.
- **Scalability** — n/a (client).
- **Reliability** — Apply is idempotent and safe to retry. A failed preview never blocks purchase at
  full price. The pending-coupon store is namespaced per slug so two tabs on two courses cannot
  cross-contaminate.
- **Observability** — `coupon_field_opened`, `coupon_applied{result}`, `coupon_from_url`,
  `coupon_removed`, `coupon_checkout_started{discounted}`, `coupon_free_grant`. www emits
  `course_detail_view` with a `hasCoupon` attribute through the existing `gtag` call.
- **Maintainability** — Coupon client logic lives in `clients/web/src/lib/marketplace-coupon.ts`
  (normalize, persist, read-from-URL, reason→i18n key) and the API call joins
  `lib/marketplace-api.ts`. The detail page gains one child component,
  `marketplace-coupon-field.tsx`, keeping the page under the 500-LOC budget.
- **Internationalization** — Reason tokens map to keys; amounts formatted with
  `formatMarketplacePrice` and the active locale; RTL verified for the struck-through price pair.
- **Backward compatibility** — With `ffCourseCoupons` off, the field is absent, `?coupon=` is
  ignored, and every existing flow is byte-identical.

## 7. Acceptance Criteria

- **AC-1.** *Given* a paid course and an active 25 % code, *When* I expand "Have a coupon code?",
  type it and apply, *Then* the price shows $30.00 with $40.00 struck through, and the CTA reads
  "Buy — $30.00".
- **AC-2.** *Given* I open `/marketplace/{slug}?coupon=LAUNCH25`, *When* the page loads, *Then* the
  discount is applied automatically, the field shows the code, and a polite announcement states the
  new price.
- **AC-3.** *Given* I open the same URL logged out, *When* I sign in and return, *Then* the discount
  is still applied.
- **AC-4.** *Given* an expired code in the URL, *When* the page loads, *Then* the full price is
  shown, the field displays "This code has expired", and the CTA still buys at full price.
- **AC-5.** *Given* a code that is valid at preview but exhausted by the time I click Buy, *When*
  checkout returns 422, *Then* the discount is cleared, the reason is shown, the CTA reverts to the
  full price, and no redirect to Stripe happens until I click again.
- **AC-6.** *Given* a 100 %-off code, *When* I click the CTA, *Then* I am enrolled and land in the
  course with a confirmation toast and no Stripe redirect.
- **AC-7.** *Given* I click Remove, *When* it completes, *Then* the full price returns, the URL no
  longer carries `coupon`, the stored code is cleared, and focus is on the coupon input.
- **AC-8.** *Given* I complete a discounted purchase, *When* I land on `/checkout/success`, *Then*
  it shows the code and the amount saved, and the stored pending coupon is gone.
- **AC-9.** *Given* I open `www` `/courses/{slug}?coupon=LAUNCH25`, *When* the page renders, *Then*
  the enroll CTA links to `{APP_ORIGIN}/marketplace/{slug}?ref=www-courses&coupon=LAUNCH25` and the
  panel says the code will be applied at checkout.
- **AC-10.** *Given* a malicious `?coupon=<script>alert(1)</script>`, *When* www and the app render,
  *Then* the value is escaped as text, truncated, and no script executes (asserted by test).
- **AC-11.** *Given* `ffCourseCoupons` is off, *When* I open a course detail page with `?coupon=`,
  *Then* no coupon UI renders and no coupon request is made.
- **AC-12.** *Given* I hit the apply rate limit, *When* the API returns 429, *Then* the UI shows a
  cooldown message and disables Apply briefly rather than showing a generic failure.
- **AC-13.** *Given* an axe scan of the detail page with the coupon field open, applied, and in
  error, *When* CI runs, *Then* there are zero violations.
- **AC-14.** *Given* the RTL locale, *When* a discount is applied, *Then* the discounted and struck
  prices render in the correct order with no clipping.

## 8. Data Model

Client-side only:

- `sessionStorage['lextures.coupon.{slug}'] = "LAUNCH25"` — pending coupon per course, cleared on
  purchase, removal, or detected ownership (FR-10).
- Component state: `{ code, status: 'idle'|'checking'|'applied'|'rejected', preview: CouponPreview | null }`.

No cookies (deliberately unlike `AFFILIATE_REF_COOKIE`), no server persistence, no IndexedDB.

## 9. API Surface

Consumes MKTC.3 only; extends the existing web client module:

```ts
// clients/web/src/lib/marketplace-api.ts (extended)
export type CouponPreview = {
  applied: boolean
  code: string
  reason: CouponReason
  listPriceCents: number
  discountCents: number
  chargedCents: number
  currency: string
  freeAfterDiscount: boolean
  endsAt: string | null
  seatsRemaining: number | null
}
export async function previewMarketplaceCoupon(slug: string, code: string): Promise<CouponPreview>
export async function checkoutMarketplaceCourse(slug: string, opts?: { couponCode?: string }): Promise<MarketplaceCheckoutResult | MarketplaceGrantedFreeResult>
export async function claimMarketplaceCourse(slug: string, opts?: { couponCode?: string }): Promise<MarketplaceClaimResult>
export async function fetchMarketplaceCourse(slug: string, opts?: { coupon?: string }): Promise<MarketplaceCourseDetail>

// clients/web/src/lib/marketplace-coupon.ts (new)
export function normalizeCouponInput(raw: string): string
export function readCouponFromLocation(search: string): string | null
export function rememberPendingCoupon(slug: string, code: string): void
export function readPendingCoupon(slug: string): string | null
export function clearPendingCoupon(slug: string): void
export function couponReasonKey(reason: CouponReason): string   // → i18n key
```

`www/src/lib/marketplace-api.ts`: `enrollHandoffUrl(slug, opts?: { coupon?: string })` appends the
parameter. No new www API calls.

## 10. UI / UX

**Surfaces touched**

1. `clients/web/src/pages/marketplace/marketplace-course-page.tsx` — coupon disclosure + field +
   price treatment + CTA wiring.
2. `clients/web/src/pages/marketplace/marketplace-coupon-field.tsx` (**new**) — the control itself.
3. `clients/web/src/pages/marketplace/marketplace-purchase-action-page.tsx` — forward `?coupon=`.
4. `clients/web/src/pages/explore-course-page.tsx` — carry `?coupon=` into the marketplace link.
5. `clients/web/src/pages/checkout/success.tsx` — show code + savings; clear pending coupon.
6. `clients/web/src/pages/checkout/cancel.tsx` — retain code; notify release.
7. `www/src/pages/course-detail-page.tsx` + `www/src/components/courses/enroll-panel.tsx` — read and
   forward the parameter, show the "will be applied" line.

**Key flows**

1. *Typed code* — Expand "Have a coupon code?" → type → Apply → pending spinner in the button →
   price updates, live region announces "Coupon LAUNCH25 applied. New price $30.00" → Buy.
2. *URL code* — Land on `…?coupon=LAUNCH25` → field pre-filled and auto-applied → same announcement
   → the parameter is replaced out of the address bar via `history.replaceState`.
3. *Logged-out URL code* — Land → code stored in `sessionStorage` → sign in → return to the detail
   route → code re-applied from storage.
4. *www handoff* — `lextures.com/courses/{slug}?coupon=CODE` → enroll CTA → app storefront with the
   code → flow 2.
5. *Free after discount* — Apply a 100 % code → CTA becomes "Enroll — Free" → click → enrolled,
   navigate into the course.

**States** — Idle (collapsed disclosure), checking (button pending, input disabled), applied
(success styling, Remove affordance, savings line), rejected (inline `role="alert"`, full price
retained), rate-limited (cooldown message), offline (Apply disabled with an explanation).

**Mobile / responsive** — The field and Apply button stack vertically below `sm`; the price block
keeps the discounted amount first in reading order; the sticky mobile CTA on www shows the list
price plus the "code will be applied" line without overflowing.

**Accessibility annotations** — Focus order: disclosure → input → Apply → (Remove when applied) →
CTA. The price container is the polite live region; it must not also be a focus target. The CTA's
`aria-describedby` continues to reference the price element, so the announced name always matches
the charge. Error text is referenced by `aria-describedby` **and** `role="alert"` so it is announced
once, not twice.

**Copy & i18n keys**

```
marketplace.coupon.disclosure        "Have a coupon code?"
marketplace.coupon.label             "Coupon code"
marketplace.coupon.placeholder       "e.g. LAUNCH25"
marketplace.coupon.apply             "Apply"
marketplace.coupon.applying          "Applying…"
marketplace.coupon.remove            "Remove coupon"
marketplace.coupon.applied           "Coupon {code} applied. New price {price}."
marketplace.coupon.savings           "You save {amount}"
marketplace.coupon.was               "was {price}"
marketplace.coupon.endsSoon          "Ends {date}"
marketplace.coupon.seatsLeft         "{count} left"
marketplace.coupon.willApply         "Coupon {code} will be applied at checkout."
marketplace.coupon.rateLimited       "Too many attempts. Try again in a moment."
marketplace.coupon.reason.not_found        "We couldn't find that code for this course."
marketplace.coupon.reason.inactive         "This code is paused."
marketplace.coupon.reason.not_started      "This code isn't active yet."
marketplace.coupon.reason.expired          "This code has expired."
marketplace.coupon.reason.exhausted        "This code has been fully claimed."
marketplace.coupon.reason.already_used     "You've already used this code."
marketplace.coupon.reason.currency_mismatch "This code doesn't apply to this course's currency."
marketplace.coupon.reason.course_free      "This course is already free."
marketplace.coupon.reason.owned            "You already have access to this course."
```

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — none (Stripe redirect is unchanged apart from the amount).
- **Internal** — the seven files listed in §10, plus `clients/web/src/lib/marketplace-api.ts`,
  `clients/web/src/lib/marketplace-coupon.ts` (**new**),
  `clients/web/src/pages/marketplace/marketplace-price-badge.tsx` (reuse `listPriceCents`),
  `clients/web/src/context/platform-features-context.tsx`,
  `clients/web/public/locales/*/common.json`, `www/src/lib/marketplace-api.ts`,
  `www/src/lib/courses-copy.ts`.
- **Events** — analytics only.

## 13. Dependencies & Sequencing

- **Must ship after** — MKTC.3 (preview + coupon-aware checkout/claim + detail `?coupon=`).
- **Must ship before** — MKTC.6 (mobile mirrors this UX and reuses the reason vocabulary), MKTC.7.
- **Shared infra** — none; www and the app deploy independently, so the www change (FR-12) is safe
  to ship first — an unknown parameter is ignored by the app until this story lands.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Code lost across the sign-in redirect | H | H | `sessionStorage` persistence (FR-10) plus an explicit test through the real auth redirect (AC-3) |
| Learner charged more than previewed | L | H | 422 handling clears the discount and requires a second click (FR-8, AC-5) |
| XSS via the `coupon` query parameter | M | H | Render as text only, truncate, never interpolate into `href`/`innerHTML`; AC-10 test on both apps |
| Stale code applied weeks later from a cookie | M | M | Deliberately `sessionStorage`, not a cookie (FR-10 rationale) |
| Price announced twice or not at all by screen readers | M | M | Single polite live region on the price block; `role="alert"` only on the error text; screen-reader script in the test plan |
| www shows a discount it cannot verify | M | M | www never shows a discounted number — only "will be applied at checkout" (FR-12) |
| Rate limit surfaces as a generic error | M | L | Explicit 429 handling and copy (FR/AC-12) |
| Two tabs, two courses, one stored code | L | M | Storage key namespaced per slug |

## 15. Rollout Plan

- **Feature flag** — `ffCourseCoupons`. Off ⇒ no field, `?coupon=` ignored (AC-11). www's forwarding
  is flag-independent and harmless.
- **Sequencing** — client helpers + API module → coupon field on detail → URL/auto-apply + storage →
  purchase-action + explore forwarding → success/cancel → www → i18n for all locales.
- **Dogfood** — internal creator makes a code in MKTC.4, posts the share link in the team channel,
  and several people (some logged out) redeem it on staging.
- **GA criteria** — the full share-link journey works from a cold, logged-out browser; axe clean;
  RTL verified; all locales carry the keys; no console errors on a malformed `?coupon=` value.
- **Rollback** — flag off. www's extra query parameter is inert.

## 16. Test Plan

- **Unit** (Vitest) — `normalizeCouponInput` (case, whitespace, length cap, disallowed chars);
  `readCouponFromLocation` for present/absent/duplicate/oversized parameters; pending-coupon
  storage set/read/clear and per-slug isolation; `couponReasonKey` covers all ten reasons (a test
  fails if a new reason has no key); `enrollHandoffUrl` with and without a coupon.
- **Integration** (React Testing Library) — apply success updates price and CTA; apply failure
  renders the right message per reason; auto-apply from URL; remove restores state and focus;
  422-at-checkout path; free-grant navigation; 429 cooldown; flag-off renders nothing.
- **End-to-end** (Playwright, extending `e2e/tests/course-marketplace-coupons.spec.ts` from MKTC.3
  and `course-coupons.spec.ts` from MKTC.4) — cold logged-out browser opens a share link, signs up,
  and completes a discounted Stripe test purchase; 100 %-off link lands in the course; expired code
  shows the reason and full-price purchase still works; www course page → app storefront carries the
  code; success page shows the savings; mobile viewport layout.
- **Security** — XSS payload in `?coupon=` on both apps (AC-10); ensure no coupon code is written to
  a cookie; ensure the client cannot influence the charged amount by editing storage or the URL
  (server recomputes — assert the resulting Stripe amount).
- **Accessibility** — axe on all three coupon states; screen-reader script: apply a code and confirm
  a single announcement of the new price; keyboard-only journey from disclosure to purchase; RTL
  snapshot.
- **Performance / load** — Lighthouse on the marketplace detail route with a coupon in the URL; no
  regression beyond ±2 points against the current report.
- **Manual exploratory** — back/forward navigation after `history.replaceState`; two tabs with
  different codes for the same course; applying a code, backgrounding the tab past the reservation
  TTL, then buying; browser autofill interference with the code field.

## 17. Documentation & Training

- **End-user docs** — "Using a coupon code" (started in MKTC.3) gains screenshots of the field, the
  applied state, and the share-link journey, plus a "why isn't my code working?" section keyed to
  the nine reasons.
- **Admin / instructor docs** — a note in the creator help page that share links work for logged-out
  recipients and survive sign-up.
- **API reference** — no change.
- **Internal runbook** — "learner says the code worked on the page but the price was full at
  Stripe": how to read the reservation, the session metadata and the 422 telemetry.

## 18. Open Questions

1. Should the coupon disclosure be expanded by default on paid courses (higher discovery, but it
   advertises that discounts exist and can depress full-price conversion)? Proposed: collapsed by
   default, expanded when a code is present.
2. Should `seatsRemaining` be shown as urgency ("3 left") when the server discloses it? Proposed:
   yes when present, phrased neutrally.
3. Should www validate the code anonymously so the marketing page can show the real discounted
   price? (Blocked on MKTC.3 §18 Q1.)
4. Should a pending coupon survive a full browser restart (i.e. `localStorage` with a short TTL)
   for learners who click a link, close the tab, and return? Proposed: no for v1; revisit with
   drop-off data.
5. Do we surface the code on the "My purchases" page for the learner's own records? Proposed: yes
   if trivial, otherwise defer to MKTC.7.

## 19. References

- Existing files: `clients/web/src/pages/marketplace/marketplace-course-page.tsx`,
  `marketplace-purchase-action-page.tsx`, `marketplace-price-badge.tsx`,
  `clients/web/src/pages/checkout/success.tsx` (poll loop), `checkout/cancel.tsx`,
  `clients/web/src/pages/explore-course-page.tsx` (affiliate `?ref=` precedent),
  `clients/web/src/lib/revenue-share-api.ts:96-110` (`AFFILIATE_REF_COOKIE` — the pattern this
  story deliberately diverges from), `clients/web/src/lib/marketplace-api.ts`,
  `www/src/lib/marketplace-api.ts:195` (`enrollHandoffUrl`),
  `www/src/components/courses/enroll-panel.tsx`.
- Standards: WCAG 2.1 AA (1.4.1 use of colour, 3.3.1/3.3.3 error identification and suggestion,
  4.1.3 status messages).
- Related plans: [MKTC.3](MKTC.3-coupon-aware-checkout-and-redemption.md),
  [MKTC.4](MKTC.4-web-creator-coupon-manager.md), [MKTC.6](../../plan/marketplace/MKTC.6-mobile-coupon-redemption.md),
  [MKT3](MKT3-marketplace-discovery-web.md),
  [MKT9](MKT9-www-course-detail-enroll.md).
