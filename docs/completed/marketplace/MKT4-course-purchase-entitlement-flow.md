# MKT4 — Course Purchase & Entitlement Flow (Free + Paid)

> Implementation plan. Source: [docs/plan/marketplace/README.md](../plan/marketplace/README.md). Part of the Marketplace epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT4 |
| **Section** | Marketplace |
| **Severity** | BLOCKER (for the epic) |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | SHIPPED |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Commerce / Growth squad (backend + web) |
| **Depends on** | MKT1, MKT3, 15.3 Stripe billing |
| **Unblocks** | MKT5, MKT6 |

---

## 1. Problem Statement

The storefront (MKT3) can render "Enroll — Free" and "Buy — $X" buttons, but nothing happens when a learner clicks them. This is the transactional heart of the marketplace: converting a click into an **entitlement** and an **enrollment** so the learner gains access and the course shows up on their Courses page. Two paths must exist — an instant **Free claim** (no payment) and a **paid checkout** that completes through the existing Stripe webhook. Both must be idempotent, must not double-charge or double-enroll, and must gate content access on a real entitlement.

## 2. Goals

- Implement a **Free claim** endpoint: for `price_cents = 0` marketplace courses, create a `course_purchase` entitlement (`acquisition_source='free'`, `amount_paid_cents=0`) and enroll the user — instantly, idempotently.
- Wire **paid purchase** to the existing Stripe checkout (15.3): create a checkout session for the course, and on webhook success create the entitlement and enroll the user.
- Enforce **access gating**: a paid course's content/enrollment is only granted with an active entitlement (return `402 Payment Required` otherwise), closing the TODO in `course_self_paced.go`.
- Make purchase → entitlement → enrollment a single, idempotent, observable transaction on each path.
- Handle refunds/chargebacks (existing webhook events) by expiring the entitlement and optionally unenrolling.

## 3. Non-Goals

- The storefront/detail UI (MKT3) — this story consumes its CTA handoff.
- Mobile in-app purchase (MKT6 owns the App Store/Play decision).
- Coupons/discounts, subscriptions, tax (already in 15.3/15.13; reused, not extended).
- Revenue share/payouts (15.8; reused).
- The Courses-page "Purchased" badge (MKT5).

## 4. Personas & User Stories

- **As a learner**, I want to click "Enroll — Free" and immediately land in the course so that free courses feel frictionless.
- **As a learner**, I want to click "Buy", pay, and be enrolled automatically so that paid access "just works."
- **As a learner who double-clicks or refreshes**, I want to not be charged twice or enrolled twice.
- **As a learner who was refunded**, I want my access to reflect that.
- **As an instructor**, I want paid-course content to be inaccessible without payment so that the paywall is real.

## 5. Functional Requirements

- **FR-1.** The system MUST expose `POST /api/v1/marketplace/courses/{slug}/claim` for **free** courses: if `price_cents = 0` and marketplace-listed+published, create (idempotently) a `course_purchase` entitlement (`acquisition_source='free'`) and enroll the user as `student`, then return the first item id (mirror `handleCourseSelfEnroll`).
- **FR-2.** `claim` MUST reject non-free courses with `402` and a body directing the client to checkout.
- **FR-3.** The system MUST expose `POST /api/v1/marketplace/courses/{slug}/checkout` for **paid** courses: create a Stripe Checkout Session (reuse `svcBilling.CreateCheckoutSession`) for that course's `price_cents`/`price_currency`, with success/cancel URLs returning to the course/marketplace, and return `checkoutUrl`.
- **FR-4.** On Stripe `checkout.session.completed` / invoice paid for a course purchase, the webhook worker MUST create the `course_purchase` entitlement (`acquisition_source='stripe'`, idempotent on `stripe_event_id`) **and** enroll the user as `student` + refresh role grants.
- **FR-5.** Both paths MUST be idempotent: repeated claims, double-clicks, or duplicate webhooks MUST yield exactly one active entitlement and one enrollment (enforced by MKT1's `(user_id, course_id)` partial unique index + `ON CONFLICT DO NOTHING` enrollment insert).
- **FR-6.** Content/enrollment for a **paid** course MUST require an active entitlement: `handleCourseSelfEnroll` and any self-paced content access MUST call `billing.MarketplaceAccess` and return `402` when absent (replacing the current "free for now" TODO at `course_self_paced.go:91`).
- **FR-7.** Free courses MUST remain accessible without an entitlement (`price_cents = 0` ⇒ `HasCourseAccess` true), but a claim SHOULD still create a `free` entitlement so the purchase is recorded (drives MKT5's indicator).
- **FR-8.** On refund/chargeback webhook (existing events), the system MUST set the entitlement `status='refunded'`; unenrollment on refund is configurable (default: keep enrollment but revoke access via entitlement status — see §18).
- **FR-9.** All purchase attempts MUST be rate-limited (reuse `billingCheckoutRateLimitPerMinute`) and require an authenticated session.
- **FR-10.** The client MUST handle: free → optimistic navigate to course on success; paid → redirect to Stripe, then `checkout/success.tsx` polls `/me/entitlements` (or the course) until the entitlement lands (webhook latency), then routes into the course.
- **FR-11.** A guard MUST prevent claiming/buying a course the user already owns (return the existing entitlement / "already owned" and route to the course).

## 6. Non-Functional Requirements

- **Performance** — Free claim p95 < 200 ms (single tx). Checkout session creation p95 < 800 ms (Stripe call). Webhook processing async via existing `payment_webhook_worker`.
- **Security** — Price and course identity are resolved server-side from the DB, never trusted from the client (prevents price tampering). Entitlement creation is authorized by the authenticated session; webhooks verified via `StripeWebhookSecret`. Enrollment side-effects run in the same transaction as entitlement creation where possible.
- **Privacy & Compliance** — Entitlements are financial records subject to 15.13 tax + retention. Free claims store no payment PII. Amounts and currency recorded for reporting.
- **Accessibility** — CTA buttons show pending/disabled state during the async flow; success/error announced via live region; redirect to Stripe preserves focus context on return. WCAG 2.1 AA.
- **Scalability** — Idempotency indexes bound duplicate work; webhook worker already handles Stripe volume.
- **Reliability** — Exactly-once effects via idempotency keys; webhook retries safe; free-claim tx atomic (entitlement + enrollment + grants). Failure after Stripe payment but before enrollment is reconciled by the webhook being the source of truth (entitlement+enroll happen there, not client-side).
- **Observability** — Metrics `marketplace_claim_total{result}`, `marketplace_checkout_created`, `marketplace_purchase_completed{amount>0}`, `marketplace_refund_total`; structured logs with course id + user id (no card data); alert on webhook failure (reuse 15.3 alerting).
- **Maintainability** — Reuse `service/billing` and `background/payment_webhook_worker.go`; add a course-purchase branch rather than a parallel system.
- **Internationalization** — Currency/amount localized; Stripe Checkout locale set from user locale; error copy externalised.
- **Backward compatibility** — Closing the `course_self_paced.go` paywall TODO changes behavior for *paid* self-paced courses; verify no currently-paid open-enrollment courses rely on the old "free for now." (Today all such are effectively free.)

## 7. Acceptance Criteria

- **AC-1.** *Given* a free listed course I don't own, *When* I click "Enroll — Free", *Then* I'm enrolled, a `free` entitlement exists, and I land on the first item.
- **AC-2.** *Given* I double-click "Enroll — Free", *When* both requests process, *Then* exactly one entitlement and one enrollment exist.
- **AC-3.** *Given* a $20 course, *When* I click "Buy" and complete Stripe checkout, *Then* the webhook creates a `stripe` entitlement, enrolls me, and the course appears on my Courses page.
- **AC-4.** *Given* Stripe re-delivers the same `checkout.session.completed` event, *When* it's processed twice, *Then* exactly one entitlement/enrollment results.
- **AC-5.** *Given* a paid course I have not paid for, *When* I try to self-enroll/access content, *Then* I get `402` and am routed to checkout.
- **AC-6.** *Given* I already own a course, *When* I open its storefront CTA, *Then* it says "Go to course" and re-claim/buy is prevented.
- **AC-7.** *Given* a completed purchase is refunded, *When* the refund webhook processes, *Then* the entitlement is `refunded` and paid content access is revoked.
- **AC-8.** *Given* payment succeeds but the webhook is delayed, *When* I land on the success page, *Then* it polls and routes me into the course once the entitlement appears (with a timeout + "we'll email you" fallback).

## 8. Data Model

No new tables. Uses MKT1 changes:
- `billing.user_entitlements` with nullable `stripe_event_id`, `acquisition_source`, and the `(user_id, course_id)` partial unique index.
- Free-claim idempotency: `INSERT ... ON CONFLICT (user_id, course_id) WHERE entitlement_type='course_purchase' AND status='active' DO NOTHING`, then re-select.
- Enrollment reuses `course.course_enrollments (course_id, user_id, role='student')` with `ON CONFLICT DO NOTHING` + `courseroles.RefreshManagedGrantsForCourseUser`.
- New repo helpers: `billing.CreateFreeCourseEntitlement`, `billing.CreateStripeCourseEntitlement` (extend `CreateIdempotent`), `billing.RefundCourseEntitlement`.

## 9. API Surface

New routes (gated by `courseMarketplaceOff`, authenticated, rate-limited):

- `POST /api/v1/marketplace/courses/{slug}/claim` → `{ enrolled, entitlementId, firstItemId? }` (free only; `402` if paid; `409`/idempotent "already owned").
- `POST /api/v1/marketplace/courses/{slug}/checkout` → `{ checkoutUrl }` (paid; `400` if free — use claim instead).
- **Webhook** — extend `handleStripeWebhook` / `payment_webhook_worker.go` course-purchase branch to enroll + refresh grants after entitlement creation (currently entitlement only).
- **Access gate** — modify `handleCourseSelfEnroll` (`course_self_paced.go`) and content-access checks to call `billing.MarketplaceAccess` and return `402` for paid-unowned.
- Reuse `GET /api/v1/me/entitlements` for the success-page poll.
- OpenAPI: document claim/checkout endpoints and the `402` contract.

```ts
// 402 body
{ code: 'payment_required', message: 'Purchase required.', checkoutHint: '/marketplace/{slug}' }
```

## 10. UI / UX

- **Storefront/detail CTA** (from MKT3) wired here:
  - Free, unowned → "Enroll — Free" → POST claim → navigate to course.
  - Paid, unowned → "Buy — $X" → POST checkout → `window.location = checkoutUrl` (Stripe) → return to `checkout/success.tsx`.
  - Owned → "Go to course".
- **Success page** (`clients/web/src/pages/checkout/success.tsx`, extend) — poll `/me/entitlements` (or course access) up to N seconds, then route into the course; timeout fallback "Your enrollment is processing; we'll email you."
- **Cancel page** (`checkout/cancel.tsx`) — return to the course detail with CTA intact.
- **States** — CTA pending/disabled during request; error toast + retry; "already owned" short-circuit; 402 → redirect to checkout.
- **Accessibility** — pending state announced; buttons keep discernible text; live-region success/error.
- **Copy & i18n** — `marketplace.cta.free|buy|owned|processing`, `.success.processing`, `.error.retry`, `.alreadyOwned`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — Stripe (Checkout Sessions, webhooks) via existing `service/paymentprovider` + `service/billing`.
- **Internal** — `httpserver/marketplace_courses_http.go` (new claim/checkout), `httpserver/course_self_paced.go` (paywall), `background/payment_webhook_worker.go` (enroll on paid), `repos/billing/entitlements.go`, `courseroles`, `enrollment`, `learnerprogress`; web `pages/marketplace/*`, `pages/checkout/*`.
- **Events** — reuse payment webhook events; emit an internal "course_purchased" notification (reuse notif events) for receipts/confirmation.

## 13. Dependencies & Sequencing

- **After** — MKT1 (idempotency + entitlement generalization), MKT3 (CTA), 15.3 (checkout + webhook worker).
- **Before** — MKT5 (badge reads entitlements), MKT6 (mobile purchase).
- **Shared infra** — Stripe, job queue/webhook worker, notifications/email (receipt).

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Double-charge / double-enroll on retries | M | H | Idempotency: Stripe event id unique + `(user,course)` partial unique + `ON CONFLICT` enroll |
| Payment succeeds, enrollment fails (partial) | M | H | Make webhook the source of truth (entitlement+enroll together, retried by Stripe); client success page polls, never enrolls |
| Closing the paywall TODO breaks existing "free" self-paced courses | M | M | Audit: gate only when `price_cents > 0`; free courses unaffected; add regression test |
| Price tampering via client | L | H | Resolve price/course server-side from DB; ignore client-supplied amounts |
| Refund policy ambiguity (unenroll or not) | M | M | Default: revoke access via `status='refunded'`, keep enrollment record; make configurable (§18) |
| Webhook latency confuses learners | M | M | Success-page polling + email fallback |

## 15. Rollout Plan

- **Flag** — `ffCourseMarketplace` (MKT1). Paid path additionally requires `ffStripeBilling`/`ffPaymentsEnabled` (15.3); if billing off, only free claims are offered and paid courses show "unavailable to purchase."
- **Sequencing** — free-claim path first (no payment dependency), then paid path once billing verified in the tenant.
- **Dogfood** — internal free claim end-to-end, then a live-mode $1 test purchase + refund.
- **GA criteria** — idempotency verified under concurrency; webhook enroll reliable; paywall returns 402; refund revokes access.
- **Rollback** — flag off disables both endpoints; paywall change guarded so free courses are never affected.

## 16. Test Plan

- **Unit** — free-claim idempotency; `MarketplaceAccess` gate; price→checkout mapping; refund status transition.
- **Integration** — concurrent double-claim → one row; duplicate webhook → one entitlement+enroll; `402` for paid-unowned self-enroll; enrollment + role grants created on paid success; refund sets `refunded` and revokes access.
- **End-to-end (Playwright + Stripe test mode)** — free claim → land in course; paid buy → Stripe test card → webhook → course appears; cancel returns cleanly; already-owned short-circuit.
- **Security** — price tamper rejected; unauthenticated rejected; webhook signature required; authz on enroll.
- **Accessibility** — CTA pending/announce; success/error live regions.
- **Performance / load** — burst of concurrent claims idempotent; webhook worker throughput unaffected.
- **Manual exploratory** — network drop mid-checkout; webhook delayed; double browser tabs.

## 17. Documentation & Training

- **Learner docs** — "Enrolling in and buying courses," receipts, refunds.
- **Instructor docs** — how purchases translate to enrollments; refund effects.
- **API reference** — claim/checkout + `402` contract.
- **Runbook** — reconciling a stuck purchase (webhook failed): how to replay and manually grant a `comp`/`stripe` entitlement.

## 18. Open Questions

1. On refund, do we **unenroll** or only revoke access? (Default: revoke access via `status='refunded'`, keep enrollment row; product to confirm.)
2. Should free claims create an entitlement at all, or only an enrollment? (Default: create a `free` entitlement so MKT5's "Purchased" indicator and analytics work uniformly.)
3. Success-page poll timeout + fallback duration? (Default: ~20s poll, then "processing, we'll email you.")
4. Do we send a receipt/confirmation email on free claims too, or only paid? (Default: paid → receipt; free → in-app confirmation only.)
5. Does buying grant lifetime access or a term? (Default: lifetime for one-time `course_purchase`; `valid_until=NULL`. Subscriptions already handled by 15.3.)

## 19. References

- Existing files: `server/internal/httpserver/course_self_paced.go` (paywall TODO ~L91), `httpserver/billing_http.go` (checkout ~L109, webhook ~L305), `background/payment_webhook_worker.go`, `service/billing`, `repos/billing/entitlements.go`, `clients/web/src/pages/checkout/success.tsx`.
- Related plans: [MKT1](MKT1-marketplace-platform-foundation.md), [MKT3](MKT3-marketplace-discovery-web.md), [MKT5](../../plan/marketplace/MKT5-purchased-indicator-courses.md), `docs/completed/15-self-learner-specific/15.2-self-paced-enrollment.md`, `15.3-billing-stripe.md`.
