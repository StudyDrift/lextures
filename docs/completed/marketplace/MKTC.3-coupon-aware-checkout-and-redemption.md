# MKTC.3 — Coupon-Aware Preview, Checkout & Redemption

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the MKTC Course Coupon Codes epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKTC.3 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | HS (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Commerce / Growth squad (backend) |
| **Depends on** | MKTC.1, MKTC.2 |
| **Unblocks** | MKTC.5, MKTC.6, MKTC.7 |

---

## 1. Problem Statement

This is the story where a code becomes money. A learner must be able to submit a code, see the exact
discounted total *before* committing, and then complete a purchase at that price — with the platform
guaranteeing that the price came from the database, that the seat was actually available, and that
the seat is consumed exactly once even when Stripe re-delivers a webhook. Today
`handleMarketplaceCheckout` (`server/internal/httpserver/marketplace_purchase_http.go:149`) sends
the course's list price straight to `svcBilling.CreateCheckoutSession` with no discount concept, and
`handleMarketplaceClaim` refuses any non-free course with `402`. Both need a coupon-aware path, and
the webhook in `service/billing/stripe.go:176` needs to close the loop on the ledger.

## 2. Goals

- A **preview endpoint** that validates a code against a course and returns the exact line-item
  breakdown (list, discount, charged, currency) plus a typed reason when it cannot be applied.
- **Coupon-aware checkout**: the discounted amount is what reaches the payment provider, and a
  reservation is taken so the seat cannot be sold twice.
- **A free path for 100 %-off codes**: no payment provider round-trip, immediate entitlement with
  `acquisition_source='coupon'` and enrollment, reusing MKT4's idempotent claim machinery.
- **Exactly-once redemption**: webhook completion promotes the reservation; duplicate events,
  double clicks, and abandoned sessions all resolve to one seat or none.
- **Correct downstream money**: tax, revenue share and the receipt/invoice all reflect the
  discounted amount, and the ledger reconciles with Stripe.

## 3. Non-Goals

- Creator CRUD (MKTC.2) and any UI (MKTC.4/MKTC.5/MKTC.6).
- Subscription-plan coupons (`plan=monthly|annual` on `POST /api/v1/billing/checkout`) — the
  learner-facing coupon surface is course-scoped only in this epic.
- Stacking two coupons, or combining a coupon with a Stripe-side promotion code.
- Anonymous (signed-out) coupon validation — see §18 Q1; www carries the code through the handoff
  instead (MKTC.5).
- Apple IAP discounting (MKTC.6 defines the honest iOS behaviour).

## 4. Personas & User Stories

- **As a learner**, I want to type a code and immediately see the new total, so that I know it
  worked before I hand over a card.
- **As a learner arriving from a share link**, I want the code already applied when the page loads,
  so that I never have to retype it.
- **As a learner with a 100 %-off code**, I want to land in the course immediately without a
  checkout screen, so that a comped seat feels comped.
- **As a learner whose code just ran out**, I want to be told *why* — expired, all claimed, already
  used by me — so that I do not think the product is broken.
- **As a course creator**, I want my "first 50" to mean 50 even if 300 people click at once.
- **As a finance analyst**, I want the Stripe charge, the entitlement, the redemption row and the
  earnings ledger to agree on the amount.

## 5. Functional Requirements

- **FR-1.** The system MUST expose
  `POST /api/v1/marketplace/courses/{slug}/coupon/preview` with body `{ "code": "LAUNCH25" }`,
  returning `200` with `{applied, code, reason, listPriceCents, discountCents, chargedCents,
  currency, freeAfterDiscount, endsAt, seatsRemaining}` for both success and typed-failure cases.
  A failure is `applied:false` with a `reason` from the MKTC.1 vocabulary — **not** an HTTP error,
  so clients render inline messages without exception plumbing.
- **FR-2.** Preview MUST require an authenticated session, MUST be gated by
  `ffCourseMarketplace && ffCourseCoupons`, and MUST NOT create a reservation (it is a pure read).
- **FR-3.** Preview MUST be rate limited per user (proposed 15/min, 60/hour) and per IP, returning
  `429 RATE_LIMITED` on trip. Every attempt MUST increment `coupon_apply_total{result}`.
- **FR-4.** `POST /api/v1/marketplace/courses/{slug}/checkout` MUST accept an optional
  `{"couponCode": "LAUNCH25"}` body field. When present the server MUST:
  (a) resolve the coupon for that course, (b) re-evaluate eligibility server-side, (c) reserve a
  seat inside a transaction, (d) pass the **discounted** amount to the provider, (e) record the
  provider session id on the reservation.
- **FR-5.** When the coupon is not applicable at checkout time, the server MUST return `422` with
  `code: UNPROCESSABLE_ENTITY`, the typed `reason`, and the undiscounted price, so the client can
  offer "continue at full price" rather than silently charging more than was previewed.
- **FR-6.** The server MUST NEVER accept a client-supplied amount, discount, or price. The only
  client input is the code string. (Direct extension of MKT4 FR "price resolved server-side".)
- **FR-7.** When the discounted amount is `0` (exact, or clamped per MKTC.1 FR-12), checkout MUST
  NOT contact the payment provider. Instead it MUST grant the entitlement with
  `acquisition_source='coupon'`, `amount_paid_cents=0`, enroll the learner via
  `courseroles.EnrollStudentWithGrants`, mark the redemption `redeemed`, and return the same shape
  as the free-claim response (`{enrolled, entitlementId, courseCode, firstItemId}`) with
  `grantedFree: true`.
- **FR-8.** `POST /api/v1/marketplace/courses/{slug}/claim` MUST also accept `couponCode` so a
  share link pointing at the claim route works for a 100 %-off code on a paid course. Behaviour is
  identical to FR-7. Without a code, a paid course still returns `402` exactly as today.
- **FR-9.** A Stripe Checkout Session created with a first-party coupon MUST set
  `AllowPromotionCodes: false` (today it is unconditionally `true` at
  `service/paymentprovider/stripe.go:42`) so discounts cannot stack, and MUST carry metadata
  `coupon_id`, `coupon_code`, `coupon_discount_cents`, and `list_price_cents`.
- **FR-10.** The webhook branch in `handleCheckoutCompleted` MUST, when `coupon_id` metadata is
  present, promote the reservation to `redeemed` idempotently on `event.ID`
  (`provider_event_id`), link `entitlement_id`, and record the final `charged_cents` from
  `session.AmountTotal` minus tax where tax is separable. Reservation missing (swept, or created
  before a restart) MUST fall back to creating a `redeemed` row directly, still idempotent.
- **FR-11.** On `charge.refunded` for a purchase that carried a coupon, the system MUST set the
  redemption to `released` (returning the seat to the pool) alongside the existing
  `RefundCourseEntitlement` call. This implements MKTC.1 §18 Q6's proposed answer.
- **FR-12.** Revenue share MUST be computed on the **charged** amount, not the list price: the
  creator absorbs the cost of their own coupon and the platform fee percentage is unchanged. This
  is the behaviour that already falls out of `RecordSaleEarnings` reading `session.AmountTotal`;
  this FR exists so it is deliberate and test-pinned rather than incidental.
- **FR-13.** Tax MUST be computed by the provider on the discounted line item (no change needed
  beyond passing the discounted `UnitAmount`), and `POST /api/v1/checkout/quote` MUST accept an
  optional `couponCode` so the tax preview total matches what the learner will pay.
- **FR-14.** The tax invoice PDF and the purchase receipt MUST show the list price, the coupon code,
  the discount, and the charged amount as separate lines.
- **FR-15.** A learner who already owns the course MUST get `reason: owned` from preview and the
  existing "already owned" short-circuit from checkout, and MUST NOT consume a seat.
- **FR-16.** An outstanding reservation MUST be honoured for its TTL even if the coupon is
  subsequently disabled, archived, or exhausted by others — the learner completes at the quoted
  price. (Pins MKTC.2 §18 Q4.)
- **FR-17.** Cancelling checkout (`/checkout/cancel`) SHOULD release the reservation eagerly when
  the client reports the cancellation, and MUST in any case be released by the TTL sweeper.
- **FR-18.** `GET /api/v1/marketplace/courses/{slug}` (detail) MUST accept an optional `?coupon=`
  query parameter and, when the code is valid, include a `coupon` object with the same shape as the
  preview response, so a share-link landing renders the discounted price in one round trip.
- **FR-19.** Both new/changed routes MUST be documented in OpenAPI, and the route inventory
  refreshed.

## 6. Non-Functional Requirements

- **Performance** — Preview p95 < 120 ms (two indexed reads, no provider call). Checkout with a
  coupon p95 < 900 ms (adds one short transaction to the existing Stripe call). Free-grant path p95
  < 250 ms.
- **Security** — Server-side price resolution (FR-6) is the core control against price tampering.
  Reservation happens under the MKTC.1 row lock. Codes are rate limited per user *and* per IP to
  bound enumeration (MKTC.7 hardens further). Webhook authenticity is unchanged (signature
  verification in `HandleWebhook`). The preview endpoint must not reveal the coupon's internal
  `note`, `created_by`, or absolute `max_redemptions` — only `seatsRemaining` when a cap exists,
  and only when it is small enough to be a legitimate urgency signal (see §18 Q3).
- **Privacy & Compliance** — Redemption rows are financial records (15.13 retention). The receipt
  now names the coupon code, which is not personal data. No new PII flows to Stripe beyond the
  existing metadata; `coupon_code` is a non-identifying string.
- **Accessibility** — Server contributes typed reasons and complete-sentence fallback messages so
  MKTC.5 can announce them in a live region. No UI here.
- **Scalability** — One extra transaction per paid checkout. Contention is per-coupon; a viral code
  serializes on one row, which is acceptable at the expected volume and is the price of correctness.
- **Reliability** — Exactly-once via three independent guards: the `(user_id, course_id)` partial
  unique index on entitlements (MKT4), `provider_event_id` uniqueness on redemptions, and the
  reservation TTL. A crash between reservation and provider call leaks a seat for at most the TTL.
- **Observability** — `coupon_apply_total{result}`, `coupon_checkout_created_total{discounted}`,
  `coupon_redeemed_total`, `coupon_discount_cents_total`, `coupon_free_grant_total`,
  `coupon_released_total{reason}`. Alert when `coupon_redeemed_total` diverges from
  `marketplace_purchase_completed` for sessions carrying `coupon_id` metadata over a 1 h window.
- **Maintainability** — Coupon resolution + reservation live in a new
  `server/internal/service/billing/coupon_checkout.go`; the HTTP handlers stay thin. The provider
  layer gains a `DiscountCents`/`ChargedCents` pair on `CheckoutRequest` rather than a Stripe-only
  concept, so PayPal keeps working.
- **Internationalization** — Reasons are tokens; amounts are minor units with an explicit currency;
  no server-side money formatting.
- **Backward compatibility** — `couponCode` is optional everywhere. A request without it produces
  byte-identical behaviour to today (verified against the TD.1 characterization goldens). The one
  intentional change is `AllowPromotionCodes` flipping to `false` **only** for sessions that carry
  a first-party coupon.

## 7. Acceptance Criteria

- **AC-1.** *Given* an active 25 % coupon on a $40 course, *When* I POST preview with the code,
  *Then* I get `applied:true, listPriceCents:4000, discountCents:1000, chargedCents:3000` and no
  reservation row exists.
- **AC-2.** *Given* the same coupon, *When* I POST checkout with `couponCode`, *Then* the Stripe
  session is created with `unit_amount = 3000`, `allow_promotion_codes = false`, metadata
  `coupon_code=LAUNCH25`, and exactly one `reserved` redemption row exists carrying the session id.
- **AC-3.** *Given* that checkout session completes, *When* the webhook processes, *Then* the
  redemption is `redeemed`, linked to the entitlement, the learner is enrolled, and
  `redeemed_count` is 1.
- **AC-4.** *Given* Stripe re-delivers the same `checkout.session.completed` event, *When* it
  processes twice, *Then* there is still exactly one `redeemed` row, one entitlement and one
  enrollment.
- **AC-5.** *Given* a coupon with `max_redemptions=1` already reserved by learner A, *When* learner
  B posts checkout with the code, *Then* B gets `422` with `reason: exhausted` and no session is
  created.
- **AC-6.** *Given* a 100 % coupon on a $40 course, *When* I post checkout (or claim) with the code,
  *Then* no Stripe call is made, an entitlement with `acquisition_source='coupon'` and
  `amount_paid_cents=0` exists, I am enrolled, the redemption is `redeemed`, and the response
  carries `grantedFree:true`.
- **AC-7.** *Given* a 99 % coupon that clamps below the provider minimum, *When* I check out,
  *Then* the free-grant path runs (AC-6 behaviour) and `coupon_clamped_to_free_total` increments.
- **AC-8.** *Given* an expired coupon, *When* I preview, *Then* `applied:false, reason:"expired"`
  with `chargedCents` equal to the list price; *When* I check out with it, *Then* `422` with the
  same reason and the full price in the body.
- **AC-9.** *Given* I reserved a seat and the creator then archives the coupon, *When* I complete
  payment within the TTL, *Then* the purchase completes at the discounted price.
- **AC-10.** *Given* I abandon checkout, *When* the TTL passes and the sweeper runs, *Then* the seat
  is available to another learner and my redemption row is `released`.
- **AC-11.** *Given* a completed coupon purchase is refunded, *When* the refund webhook processes,
  *Then* the entitlement is `refunded`, the redemption is `released`, and the seat is available.
- **AC-12.** *Given* revenue share is enabled and a $40 course sells for $30 with a coupon, *When*
  earnings are recorded, *Then* the creator ledger entry is computed from 3000¢, not 4000¢.
- **AC-13.** *Given* tax collection is enabled, *When* I request a checkout quote with a coupon,
  *Then* the tax is computed on the discounted subtotal and the returned total matches the amount
  Stripe subsequently charges.
- **AC-14.** *Given* I already own the course, *When* I preview a valid code, *Then*
  `applied:false, reason:"owned"` and no seat is consumed.
- **AC-15.** *Given* I request the course detail with `?coupon=LAUNCH25`, *When* the code is valid,
  *Then* the response includes the coupon breakdown; *when* it is not, *then* the detail still
  renders with `coupon.applied:false` and a reason — never a 4xx on the detail route.
- **AC-16.** *Given* a checkout request with no `couponCode`, *When* it runs, *Then* the outgoing
  Stripe parameters are byte-identical to today's (characterization golden unchanged).

## 8. Data Model

No new tables or columns — MKTC.1 provides both. This story fills in:

- `coupon_redemptions.checkout_session_id` at reservation time (unique partial index prevents two
  reservations pointing at one session).
- `coupon_redemptions.provider_event_id` + `redeemed_at` + `entitlement_id` at webhook time.
- `coupon_redemptions.released_at` on cancel, TTL sweep, or refund.
- `user_entitlements.acquisition_source='coupon'` on the free-grant path.

Reconciliation invariant, asserted by a test and documented in the runbook:

```sql
-- every redeemed row must point at a live entitlement for the same (user, course)
SELECT r.id FROM billing.coupon_redemptions r
LEFT JOIN billing.user_entitlements e ON e.id = r.entitlement_id
WHERE r.status = 'redeemed'
  AND (e.id IS NULL OR e.user_id <> r.user_id OR e.course_id <> r.course_id);
```

## 9. API Surface

| Verb | Path | Change | Auth |
|---|---|---|---|
| POST | `/api/v1/marketplace/courses/{slug}/coupon/preview` | **new** | session |
| POST | `/api/v1/marketplace/courses/{slug}/checkout` | body gains `couponCode?` | session |
| POST | `/api/v1/marketplace/courses/{slug}/claim` | body gains `couponCode?` | session |
| GET | `/api/v1/marketplace/courses/{slug}` | query gains `?coupon=` | session |
| POST | `/api/v1/checkout/quote` | body gains `couponCode?` | session |

```ts
// POST .../coupon/preview  →  200 for both applied and not-applied
type CouponPreviewResponse = {
  applied: boolean
  code: string                     // normalized echo of the submitted code
  reason: 'ok' | 'not_found' | 'inactive' | 'not_started' | 'expired'
        | 'exhausted' | 'already_used' | 'currency_mismatch' | 'course_free' | 'owned'
  listPriceCents: number
  discountCents: number            // 0 when not applied
  chargedCents: number             // == listPriceCents when not applied
  currency: string
  freeAfterDiscount: boolean
  endsAt: string | null            // so the client can show "ends in 2 days"
  seatsRemaining: number | null    // null when uncapped or above the disclosure threshold
}

// POST .../checkout  (body)
type CheckoutBody = { couponCode?: string }
// 200 — paid path
type CheckoutOk = { sessionId: string; checkoutUrl: string; chargedCents: number; currency: string }
// 200 — free-after-coupon path
type CheckoutGranted = {
  grantedFree: true; enrolled: true; entitlementId: string
  courseCode: string; firstItemId?: string
}
// 422 — coupon no longer applicable
type CheckoutCouponRejected = {
  error: { code: 'UNPROCESSABLE_ENTITY'; message: string }
  reason: CouponPreviewResponse['reason']
  listPriceCents: number
  currency: string
}
```

WebSocket: none. Rate limits: preview 15/min + 60/hour per user and a per-IP bucket; checkout
continues to use `checkBillingCheckoutRateLimit`. OpenAPI: document the new route and the three
changed schemas; regenerate `clients/web/src/lib/generated/openapi-types.ts`.

## 10. UI / UX

No UI in this story. Contracts fixed here that MKTC.5/MKTC.6 depend on:

- Preview returns `200` for typed failures, so clients render inline field errors rather than
  toasts-on-exceptions.
- `chargedCents` always equals what the learner will be charged before tax, so the client never
  recomputes a discount.
- The free-after-coupon response is shape-compatible with the existing claim response, so the
  "navigate into the course" path is unchanged.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — Stripe Checkout Sessions (discounted `unit_amount`, `allow_promotion_codes:false`,
  extra metadata) and Stripe Tax (unchanged mechanism, new base). PayPal via
  `service/paymentprovider/paypal.go` receives the same discounted amount through the shared
  `CheckoutRequest`.
- **Internal** —
  `server/internal/httpserver/marketplace_purchase_http.go` (claim + checkout + new preview route),
  `server/internal/httpserver/marketplace_courses_http.go` (detail `?coupon=`),
  `server/internal/httpserver/tax_http.go` (quote `couponCode`),
  `server/internal/service/billing/coupon_checkout.go` (**new** — resolve, evaluate, reserve),
  `server/internal/service/billing/stripe.go:176` (`handleCheckoutCompleted` redemption promotion;
  `handleChargeRefunded` release), `server/internal/service/paymentprovider/checkout.go` +
  `stripe.go` + `types.go` (discounted amount plumbed through `CheckoutRequest`),
  `server/internal/service/billing/invoice_pdf.go` + `transcript_receipt.go` pattern for the
  receipt lines, `server/internal/repos/billing/coupons.go` (MKTC.1),
  `server/internal/telemetry/default.go`.
- **Events** — reuse the MKT4 course-purchase notification; add `couponCode` to its payload so the
  confirmation email can say "LAUNCH25 applied — you saved $10".

## 13. Dependencies & Sequencing

- **Must ship after** — MKTC.1 (engine + ledger), MKTC.2 (flag + `couponsFeatureOff` + coupons to
  test against).
- **Must ship before** — MKTC.5 (web learner UI), MKTC.6 (mobile), MKTC.7 (rollout).
- **Shared infra** — Stripe test mode, the payment webhook worker
  (`background/payment_webhook_worker.go`), the scheduler for the TTL sweeper, email for receipts.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Price tampering via client-supplied amounts | L | H | FR-6: the only client input is the code; explicit test asserting a body-supplied `chargedCents` is ignored |
| Discount stacking with Stripe promotion codes | M | H | `AllowPromotionCodes:false` whenever a first-party coupon is applied (FR-9); asserted in a provider unit test |
| Seat leak between reservation and provider failure | M | M | TTL sweeper + eager release on cancel + release on provider error before returning |
| Webhook arrives with no matching reservation | M | M | Fallback creates a `redeemed` row idempotently (FR-10) so money and ledger never disagree |
| Revenue share computed on list price | M | M | FR-12 pinned by AC-12; ledger test with a discounted sale |
| Tax computed on pre-discount base | L | H | Discount applied to `unit_amount` *before* Stripe Tax runs; AC-13 compares quote to charge |
| Preview becomes a code-enumeration oracle | M | M | Per-user + per-IP rate limits here; deeper controls (attempt log, lockout, entropy floor) in MKTC.7 |
| Behaviour drift on the no-coupon path | M | H | AC-16 characterization golden + TD.1 route inventory |
| A 100 % code becomes a free-course farm | L | M | Per-user cap defaults to 1; creator sets total cap; telemetry alert on free-grant spikes (MKTC.7) |

## 15. Rollout Plan

- **Feature flag** — `ffCourseCoupons` (MKTC.2), still default **OFF**. With the flag off, preview
  404s and `couponCode` is ignored on checkout/claim so no behaviour changes.
- **Sequencing** — service layer + provider plumbing → preview route → checkout/claim integration →
  webhook promotion + refund release → quote/receipt/invoice → OpenAPI + inventory.
- **Dogfood** — on staging with Stripe test mode: a 25 % code end-to-end (card `4242…`), a 100 %
  code, an exhausted code, an abandoned checkout (verify the seat returns), and a refund (verify
  the seat returns and the entitlement flips).
- **GA criteria** — idempotency verified under duplicate webhook delivery; concurrency test shows no
  oversell through the HTTP layer; quote-vs-charge equality with tax on; no-coupon golden unchanged;
  reconciliation query returns zero rows after the dogfood run.
- **Rollback** — flag off. Reservations already taken expire by TTL; redeemed rows and entitlements
  remain valid (learners keep what they bought). No migration to reverse.

## 16. Test Plan

- **Unit** (`*_nodb_test.go` + `service/billing`) — preview response mapping for all ten reasons;
  checkout body decode with and without `couponCode`; provider parameter construction
  (discounted `unit_amount`, `allow_promotion_codes`, metadata) using the existing
  `paymentprovider` test harness; free-grant branch selection at exactly 0 and at the clamp
  boundary; refund handler releasing a redemption.
- **Integration** (DB + fake provider) — reserve→redeem→enroll happy path; duplicate webhook;
  reservation TTL expiry frees the seat; concurrent checkout on a 1-seat coupon through the HTTP
  handler; archive-mid-flight honours the reservation; refund releases; entitlement/redemption
  reconciliation query returns empty; `acquisition_source='coupon'` written on the free path.
- **End-to-end** (Playwright + Stripe test mode, `e2e/tests/course-marketplace-coupons.spec.ts`) —
  apply a code and buy; land on a `?coupon=` URL and buy; 100 %-off code lands straight in the
  course; exhausted code shows the typed reason and full-price fallback; cancel returns and the
  seat is released.
- **Security** — price-tamper attempt (body `chargedCents`, `discountCents`, `priceCents`) ignored;
  coupon from course B rejected on course A; preview rate-limit trip; unauthenticated preview
  rejected; webhook without signature rejected; verify `note`/`createdBy` never appear in a
  learner-facing response.
- **Accessibility** — n/a (no UI); reason tokens verified to have i18n keys reserved in MKTC.5.
- **Performance / load** — 200 concurrent checkouts against a 50-seat coupon: exactly 50 sessions
  created, p95 < 900 ms, zero oversell.
- **Manual exploratory** — network drop between reservation and Stripe redirect; two browser tabs
  with the same code; refund a coupon purchase and immediately re-buy with the same code
  (per-user cap should block re-use unless the cap allows it — verify the released seat does not
  resurrect the per-user allowance incorrectly).

## 17. Documentation & Training

- **End-user docs** — "Using a coupon code" help page: where to enter it, why a code may not work
  (one line per reason), and what a share link does.
- **Admin / instructor docs** — "What a coupon costs you": revenue-share and tax interaction, with a
  worked $40 → $30 example.
- **API reference** — preview route plus the three changed request bodies in `openapi.json`; append
  to `docs/api-changelog-course-coupons.md`.
- **Internal runbook** — extend `docs/runbooks/coupons.md`: "a learner says the discount did not
  apply" (check reservation, session metadata, webhook delivery), "release a stuck reservation",
  "reconcile ledger vs Stripe for a discounted sale".

## 18. Open Questions

1. **Anonymous preview.** Should signed-out visitors on www be able to validate a code before
   creating an account? (Proposed: no — it doubles the enumeration surface and www already carries
   the code through the handoff. Revisit if conversion data argues otherwise.)
2. **Full-price fallback UX.** On a `422` should the client auto-continue at full price, or require
   a second explicit click? (Proposed: require the click; the server returns the full price so the
   client can render it.)
3. **`seatsRemaining` disclosure.** Showing "3 seats left" converts well but leaks cap state.
   (Proposed: expose only when `remaining <= 10`, otherwise `null`.)
4. **Refund releases the seat** (FR-11). Confirm with product; the alternative is that a refunded
   buyer permanently consumes one of the creator's seats.
5. Should the confirmation email name the code and the saved amount? (Proposed: yes — it is the
   creator's marketing signal.)
6. Do we ever want `couponCode` on the subscription checkout path? (Out of scope here; the field is
   deliberately not added to `plan=` checkouts.)

## 19. References

- Existing files: `server/internal/httpserver/marketplace_purchase_http.go` (claim L37, checkout
  L149, 402 hint L244), `server/internal/service/billing/stripe.go` (`CreateCheckoutSession` L73,
  `handleCheckoutCompleted` L176, `handleChargeRefunded` L375),
  `server/internal/service/paymentprovider/checkout.go` (`StartCheckout` L35),
  `server/internal/service/paymentprovider/stripe.go` (`AllowPromotionCodes` L42, line items L63),
  `server/internal/service/billing/revenue_share.go` (`RecordSaleEarnings` L62),
  `server/internal/httpserver/tax_http.go` (`handleCheckoutQuote` L36),
  `server/internal/background/payment_webhook_worker.go`.
- Related plans: [MKTC.1](MKTC.1-coupon-data-model-and-discount-engine.md),
  [MKTC.2](MKTC.2-creator-coupon-management-api.md),
  [MKTC.5](../../plan/marketplace/MKTC.5-web-learner-coupon-entry-and-url-codes.md),
  [MKT4](MKT4-course-purchase-entitlement-flow.md),
  [15.13 tax compliance](../15-self-learner-specific/15.13-tax-compliance.md).
