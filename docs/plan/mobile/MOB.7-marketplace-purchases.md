# MOB.7 — Marketplace Purchases & Purchased Courses (mobile)

> Implementation plan. Source: Mobile ↔ web parity gap analysis (2026-07-17).
> Web reference: [`clients/web/src/pages/marketplace/*`](../../../clients/web/src/pages/marketplace/),
> `pages/checkout/*`, `lib/marketplace-api.ts`, `lib/billing-api.ts`
> (`/api/v1/me/purchases`, `/api/v1/me/entitlements`).

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MOB.7 |
| **Section** | Mobile parity |
| **Severity** | MAJOR |
| **Markets** | SL / HE (CE) |
| **Status (today)** | PARTIAL |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile team + Billing |
| **Depends on** | App Store / Play purchase-policy decision (§18 Q1) |
| **Unblocks** | — |

## 1. Problem Statement

On web, a learner can browse the marketplace, open a course, **claim** free
courses, **buy** paid courses via Stripe Checkout, and later see everything they
own under **My Purchases** (`/me/purchases`, `/me/entitlements`). On mobile the
loop is broken: the marketplace list and detail exist and show prices, and
`Billing/PurchaseFlow` can already hand off to Stripe web checkout with
success/cancel return handling — but the **marketplace detail routes paid
courses to a "buy on web" button** instead of that flow, and there is **no
purchased-courses library**. Learners therefore discover courses on their phone
but cannot reliably complete a purchase or find what they already own, losing
self-learner revenue and creating a confusing entitlement story.

## 2. Goals

- Complete the purchase loop on mobile: discover → (claim free | buy paid) →
  own → find under a purchased-courses library.
- Wire the existing `Billing/PurchaseFlow` (tax quote + Stripe checkout handoff
  + `CheckoutReturnHandler`) into the marketplace detail's paid path, replacing
  "buy on web".
- Add a **My Purchases / Purchased courses** screen backed by
  `/me/purchases` + `/me/entitlements`.
- Reflect ownership everywhere (marketplace `owned` flag, course access) right
  after purchase/claim.
- Land on a **compliant** purchase mechanism for iOS App Store and Google Play.

## 3. Non-Goals

- Building the payments backend (Stripe, tax, revenue-share, entitlements are
  shipped).
- Subscriptions/coupons/refunds UI beyond what web marketplace exposes.
- Creator earnings dashboards (separate `/creator/earnings`).
- Changing pricing or catalog logic.

## 4. Personas & User Stories

- **As a self-learner**, I want to buy a paid course on my phone and start it
  immediately.
- **As a self-learner**, I want to claim a free course in-app without bouncing to
  the web.
- **As a returning learner**, I want a "My Purchases" list so I can reopen
  anything I've bought.
- **As a finance-conscious buyer**, I want a clear price/tax breakdown before I
  pay.

## 5. Functional Requirements

- **FR-1.** The marketplace detail MUST offer **claim** for free courses in-app
  (`POST /api/v1/marketplace/courses/{slug}/claim` → entitlement + enrollment)
  and land the user in the course.
- **FR-2.** For paid courses, the detail MUST start the purchase via the sanctioned
  mechanism decided in §18 Q1 (external Stripe checkout **or** platform IAP),
  reusing `PurchaseFlow`/`BillingLogic` where applicable, instead of "buy on
  web".
- **FR-3.** The purchase MUST show a price/tax quote before payment (existing tax
  quote path).
- **FR-4.** On successful return (`CheckoutReturnHandler`), the app MUST refresh
  entitlement/ownership and route into the purchased course.
- **FR-5.** A **Purchased courses** screen MUST list owned courses from
  `/api/v1/me/purchases` (and/or `/me/entitlements`) with open actions.
- **FR-6.** The marketplace `owned` flag MUST render correctly (owned courses
  show "Open", not "Buy").
- **FR-7.** Errors (already owned, payment failed, cancelled) MUST show clear,
  localized states; "already owned" MUST offer "Open".
- **FR-8.** Entitlement checks MUST gate course access consistently with web.

## 6. Non-Functional Requirements

- **Performance** — checkout handoff opens < 1 s; purchases list p95 < 1 s;
  ownership refresh immediate on return.
- **Security** — checkout URLs/sessions are server-issued and single-use;
  entitlement is server-authoritative (never trust client "owned"); no card data
  touches the app.
- **Privacy & Compliance** — purchase records are financial PII; tax handling per
  the shipped tax plan; comply with **App Store** and **Play** payment policies
  (§14/§18) and with consumer-law disclosures (price incl. tax, refund terms).
- **Accessibility** — WCAG 2.1 AA; price/tax and buttons labelled; return states
  announced.
- **Scalability** — n/a (per-user).
- **Reliability** — idempotent claim/checkout (already-owned handled); return
  handler tolerant of app relaunch mid-checkout; deep-link back from browser.
- **Observability** — `marketplace_{viewed,claim,checkout_started,purchase_succeeded,purchase_failed,cancelled}`
  and `purchases_list_viewed` (no card/PII).
- **Maintainability** — reuse `MarketplaceLogic`, `BillingLogic`,
  `LMSAPIMarketplace`, `LMSAPIBilling`, `LMSAPIWallet`.
- **Internationalization** — `mobile.marketplace.*`, `mobile.billing.*`;
  currency/tax formatting via `CurrencyExponent`.
- **Backward compatibility** — no API change (unless IAP path chosen → new
  server receipt-validation endpoint; see §18).

## 7. Acceptance Criteria

- **AC-1.** *Given* a free course, *when* the user taps Claim, *then* they are
  enrolled and land in the course (no web bounce).
- **AC-2.** *Given* a paid course, *when* the user completes the sanctioned
  purchase flow, *then* ownership is reflected and the course opens.
- **AC-3.** *Given* a completed purchase, *when* the user opens Purchased
  courses, *then* the course is listed and opens.
- **AC-4.** *Given* an already-owned course, *when* the detail loads, *then* it
  shows "Open", not "Buy".
- **AC-5.** *Given* a cancelled checkout, *when* the user returns, *then* a
  cancelled state is shown and no entitlement is granted.
- **AC-6.** *Given* the chosen purchase mechanism, *then* it passes App Store and
  Play review (validated in TestFlight / internal track).

## 8. Data Model

- **No new tables** for the external-checkout path (entitlements/purchases exist).
- **If platform IAP is chosen:** a server-side receipt-validation +
  entitlement-grant path is required (App Store Server Notifications / Play
  Developer API) — new server work, called out as a dependency, not built here.
- Client adds a purchases list view model only.

## 9. API Surface

Existing (reused):

- `GET /api/v1/marketplace/courses` / `…/courses/{slug}` — list/detail (`owned`).
- `POST /api/v1/marketplace/courses/{slug}/claim` — free claim.
- `POST /api/v1/marketplace/courses/{slug}/checkout` — Stripe Checkout URL.
- `GET /api/v1/me/purchases`, `GET /api/v1/me/entitlements`,
  `GET /api/v1/internal/entitlements/check` — ownership.
- Tax quote (via `BillingLogic`) before paid checkout.

Potential new (only if IAP path): receipt-validation webhook/endpoint (server).

## 10. UI / UX

- **Modified:** `Marketplace/MarketplaceDetailView` — replace "buy on web" with
  the sanctioned purchase action; show price + tax; show "Open" when owned.
- **New:** Purchased courses screen (link from profile/wallet, mirroring
  `/me/purchases`).
- **Reused:** `Billing/PurchaseFlow`, `CheckoutReturnHandler`.
- **Flows:** browse → detail → claim/buy → return → open; profile → Purchased
  courses → open.
- **States:** loading, price/tax quote, redirecting to checkout, returned
  success/cancel, already-owned, error, empty purchases list, offline.
- **Mobile/responsive:** clear price/tax rows; primary CTA; SafariVC/Custom Tab
  for external checkout with return deep link.
- **Accessibility:** labelled price/CTA; return announced; focus returns to
  detail after checkout.
- **Copy & i18n:** `mobile.marketplace.*`, `mobile.billing.*` (remove
  `buyOnWeb` fallback where replaced).

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- iOS: `Features/Marketplace/MarketplaceDetailView.swift`,
  `Features/Billing/PurchaseFlow.swift`, `CheckoutReturnHandler.swift`,
  `Core/LMS/LMSAPIMarketplace.swift`, `LMSAPIBilling.swift`, `LMSAPIWallet.swift`,
  `BillingLogic.swift`, `MarketplaceLogic.swift`.
- Android: `features/marketplace/*`, `core/lms/MarketplaceLogic.kt`,
  `BillingLogic.kt`, `MarketplaceApi`/`BillingApi`.
- Deep-link/return routing via app URL scheme + universal/app links.

## 13. Dependencies & Sequencing

- Must ship after: the §18 Q1 policy decision (external checkout vs IAP).
- Must ship before: —.
- Shared infra: Stripe checkout + return deep links; (if IAP) StoreKit 2 /
  Play Billing + server receipt validation.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| **App Store 3.1.1**: Apple may require IAP for in-app digital course purchases, rejecting external Stripe checkout | H | H | Decide policy stance (§18 Q1): use External Purchase Link Entitlement / link-out where permitted, or implement StoreKit IAP; validate in review before GA |
| Play Billing policy parity issue | M | H | Same decision; use Play Billing if required for digital goods |
| 30% platform fee erodes creator revenue-share economics | M | H | Model economics per path; possibly restrict paid purchase to web with clear messaging if IAP is uneconomical |
| Return deep-link lost when app is killed mid-checkout | M | M | Robust return handler + entitlement re-check on next launch |
| "Owned" shown stale | M | M | Server-authoritative entitlement re-check after return |

## 15. Rollout Plan

- Flag: `ff_mobile_marketplace_purchase` (default off).
- Sequence: in-app **free claim** + **Purchased courses** first (low policy
  risk) → paid purchase after the policy decision and a review pass →
  staged GA per platform.
- GA criteria: AC-1..6 pass; **passes App Store + Play review**; purchase success
  rate ≥ web parity in pilot.
- Rollback: flag off restores browse-only + "buy on web".

## 16. Test Plan

- **Unit** — claim/checkout request building; owned-state logic; return handling;
  tax quote formatting.
- **Integration** — claim → owned; checkout → return → owned; purchases list.
- **End-to-end** — free claim on device; paid purchase in sandbox (Stripe test /
  StoreKit sandbox / Play internal); app-killed-mid-checkout recovery.
- **Security** — entitlement server-authoritative; single-use sessions; no card
  data in app/logs; (IAP) receipt validation.
- **Compliance** — App Store + Play review dry-run of the chosen mechanism.
- **Accessibility** — price/CTA/return screen-reader run.
- **Manual** — cancelled checkout; already-owned; currency/locale.

## 17. Documentation & Training

- "Buy and find your courses on mobile" help article.
- Internal note documenting the chosen purchase mechanism and why (policy).
- Refund/entitlement support runbook.

## 18. Open Questions

1. **(Blocking)** Purchase mechanism: external Stripe checkout (via link-out /
   External Purchase Link Entitlement) vs. platform IAP (StoreKit 2 / Play
   Billing) vs. keep paid on web only? This drives scope, economics, and review
   risk. Needs product + legal + finance sign-off.
2. If IAP: who builds server receipt validation + entitlement grant, and how do
   revenue-share splits reconcile against the platform fee?
3. Where does "Purchased courses" live in mobile IA — profile, wallet, or a
   top-level tab?
4. Do we surface CE-transcript/receipts (`/me/ce-transcript`, receipts) here or
   in a later plan?

## 19. References

- Web: `clients/web/src/pages/marketplace/*`, `pages/checkout/*`,
  `lib/marketplace-api.ts`, `lib/billing-api.ts`.
- iOS: `Features/Marketplace/*`, `Features/Billing/PurchaseFlow.swift`,
  `CheckoutReturnHandler.swift`, `Core/LMS/LMSAPIMarketplace.swift`,
  `LMSAPIBilling.swift`.
- Android: `features/marketplace/*`, `core/lms/MarketplaceLogic.kt`,
  `BillingLogic.kt`.
- External: Apple App Store Review Guideline 3.1.1 & External Purchase Link
  Entitlement; Google Play Payments policy; Stripe Checkout.
