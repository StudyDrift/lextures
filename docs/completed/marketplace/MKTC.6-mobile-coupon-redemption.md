# MKTC.6 — Mobile: Coupons on iOS & Android

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the MKTC Course Coupon Codes epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKTC.6 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | HS (primary) · HE · K12 |
| **Status (today)** | COMPLETE |
| **Estimated effort** | M (2–4w) |
| **Owner (proposed)** | Mobile squad (iOS + Android) with Commerce review |
| **Depends on** | MKTC.3, MKTC.5 |
| **Unblocks** | MKTC.7 |

---

## 1. Problem Statement

Share links get opened on phones. Today a `?coupon=` link either lands in the mobile app's
marketplace detail screen, which knows nothing about coupons, or bounces to a browser — and the two
apps buy in fundamentally different ways. Android opens the server-provided Stripe Checkout URL in a
Custom Tab (`clients/android/.../billing/BillingCheckout.kt`), so a discounted amount flows through
naturally. iOS buys through **StoreKit 2 against fixed App Store price points**
(`clients/ios/Lextures/Features/Billing/PurchaseFlow.swift`, App Store guideline 3.1.1 Path A),
where "$49 minus 35 %" simply does not exist as a product. Doing nothing means learners are charged
full price after being promised a discount; pretending it works means an App Store rejection or a
chargeback. This story ships the honest behaviour on both platforms.

## 2. Goals

- Coupon entry and auto-apply from a deep link on **both** apps, using the same reason vocabulary
  and price presentation as web.
- **Android**: full parity with web — the discounted Stripe Checkout URL opens in the Custom Tab.
- **iOS**: 100 %-off codes redeem natively as a free grant (no IAP involved); codes that leave a
  non-zero price are validated, shown with their real value, and clearly routed to the web with no
  silent full-price charge.
- Deep links (`lextures://` / universal links / app links) carrying `?coupon=` route to the course
  detail screen with the code applied.
- No App Store or Play policy violation, and no dark pattern in either direction.

## 3. Non-Goals

- Creator coupon management on mobile (view-only is not shipped either; creators manage on web).
  A future story may add it — see §18 Q1.
- Apple **Offer Codes** / promotional offers (subscription-only in StoreKit; the marketplace sells
  one-time non-consumables).
- Creating additional StoreKit price-point products per discount tier (see §18 Q2 — explicitly
  rejected for v1 as unbounded product sprawl).
- Google Play Billing migration (Android continues to use web checkout as today).
- Offline coupon validation.

## 4. Personas & User Stories

- **As a learner on Android**, I want to open a share link and buy at the discounted price without
  leaving the app awkwardly, so that mobile is not a second-class purchase path.
- **As a learner on iOS with a 100 % code**, I want to be enrolled immediately, so that a comped
  seat is comped.
- **As a learner on iOS with a 30 % code**, I want to be told plainly that this code is redeemed on
  the web and given a way to get there, rather than being charged full price.
- **As a learner**, I want to type a code on the detail screen if I did not arrive from a link.
- **As a VoiceOver / TalkBack user**, I want the price change and any coupon error announced.
- **As the company**, I want neither app store to be able to say we misled a buyer.

## 5. Functional Requirements

### Shared

- **FR-1.** Both apps MUST add a coupon control to the marketplace course detail screen
  (`MarketplaceDetailView.swift`, `MarketplaceDetailScreen.kt`) for paid, unowned courses when
  `ffCourseCoupons` is on in the platform features payload; otherwise the control MUST be absent.
- **FR-2.** Both apps MUST call `POST /api/v1/marketplace/courses/{slug}/coupon/preview` on apply
  and render the typed reason from the shared vocabulary, localized through the existing
  `Localizable.xcstrings` / `strings.xml` mechanisms with keys mirroring web's
  `marketplace.coupon.reason.*`.
- **FR-3.** On success both apps MUST display the discounted price as primary with the list price
  struck through and a "you save X" line, and MUST update the CTA label to the discounted amount.
- **FR-4.** Both apps MUST accept `?coupon=` on deep links and universal/app links to the course
  detail destination, normalize it to upper case, and auto-apply, routing through the existing
  `LinkOpener` policy (MB.1) so the link stays in-app.
- **FR-5.** Both apps MUST hold the pending code in memory for the session only (no disk
  persistence), and MUST clear it on successful purchase, on remove, or when ownership is detected.
  Nothing coupon-related may be written to the offline cache.
- **FR-6.** Both apps MUST handle a `422` at purchase time (code lapsed between preview and buy) by
  clearing the discount, showing the reason, and requiring a second explicit tap to buy at full
  price.
- **FR-7.** Both apps MUST handle `429` with a cooldown message rather than a generic failure.
- **FR-8.** Both apps MUST emit the existing `MarketplaceObservability` events with coupon
  attributes: `coupon_applied{result}`, `coupon_from_deeplink`, `coupon_checkout_started`,
  `coupon_free_grant`, `coupon_web_redirect` (iOS only).

### Android

- **FR-9.** `LmsApi.checkoutMarketplaceCourse` MUST send `couponCode` and MUST surface the
  discounted `chargedCents` in the purchase sheet total.
- **FR-10.** When checkout returns the free-grant shape, the app MUST skip the Custom Tab entirely,
  refresh entitlements, and route into the course exactly as the free-claim path does.
- **FR-11.** `PurchaseFlowSheet` MUST show the coupon code, the discount and the final total
  (including the tax quote when `ffTaxCollection` is on) before the learner taps to continue, so
  the amount shown matches the Custom Tab.

### iOS

- **FR-12.** When a previewed coupon results in `freeAfterDiscount: true`, the app MUST call
  `POST /api/v1/marketplace/courses/{slug}/claim` with the code, MUST NOT invoke StoreKit, and MUST
  route into the course on success. (No payment occurs, so no IAP obligation arises.)
- **FR-13.** When a previewed coupon results in a **non-zero** discounted price, the app MUST:
  (a) show the discounted price and the code as validated, (b) disable the in-app StoreKit purchase
  button for the discounted amount, (c) present a clearly labelled explanation
  ("This code is redeemed on the web") with a **Copy code** action and a **Continue in browser**
  action that opens the storefront URL with the coupon through `LinkOpener`'s external/system path,
  and (d) offer an explicit secondary "Buy at full price in the app" action that uses the normal
  StoreKit flow at the undiscounted price, with the full price stated on the button.
- **FR-14.** The iOS app MUST NOT charge the StoreKit full price while a discount is displayed as
  applied; the two prices must never be presented as if the code is being honoured in-app.
- **FR-15.** The external-browser affordance MUST comply with the App Store rules in force at
  release. The implementation MUST be reviewed against guideline 3.1.1 / 3.1.3 and the current
  external-purchase-link rules **before submission**, and the copy MUST be signed off by the same
  reviewer who handled the 3.1.1 Path A submission (`docs/ios-app-store-3.1.1-iap-path-a.md`).
  If review guidance forbids the link at submission time, the fallback is copy-the-code plus
  "open lextures.com in your browser" with no tappable external purchase link — the behaviour is
  gated behind a remote-config-able boolean so it can be adjusted without a resubmission.
- **FR-16.** iOS MUST NOT create per-discount StoreKit products; the App Store price point remains
  the list price.

## 6. Non-Functional Requirements

- **Performance** — Preview adds one network call on apply / deep-link land; both apps MUST show an
  inline pending state and MUST NOT block screen rendering on it. p95 < 400 ms perceived on a warm
  connection; timeouts fall back to full price with a retry affordance.
- **Security** — The code is an opaque string; neither app computes a price. Deep-link parameters
  are validated (length ≤ 32, character class) before use and never interpolated into a WebView URL
  unescaped. The iOS "continue in browser" URL is built from the server-provided storefront origin,
  not from user input.
- **Privacy & Compliance** — No coupon data persisted to disk (FR-5), so nothing new appears in
  device backups. App Store / Play policy compliance is a first-class acceptance criterion
  (FR-15, AC-10). No new data collected, so no App Privacy label change.
- **Accessibility** — iOS: Dynamic Type at accessibility sizes without truncation; VoiceOver
  announces the applied price change via an accessibility notification; the struck-through price
  has an explicit accessibility label ("was $40.00"); the disabled in-app purchase button explains
  *why* it is disabled through its accessibility hint, not just visually. Android: TalkBack
  equivalents, `contentDescription` on the price pair, minimum 48 dp touch targets, and error text
  associated with the field.
- **Scalability** — n/a (client).
- **Reliability** — Apply is idempotent and retryable. A failed preview never blocks a full-price
  purchase. Android's Custom Tab return path (`CheckoutReturnOverlay` / `CheckoutReturnHandler`) is
  unchanged and continues to poll entitlements.
- **Observability** — FR-8 events, plus an iOS-specific counter for how often the web-redirect path
  is taken — the primary input to §18 Q2.
- **Maintainability** — Shared pure logic goes into the existing testable logic layers
  (`MarketplaceLogic.swift`, `MarketplaceLogic.kt`): normalization, reason→string-key mapping,
  price presentation, and the platform decision (`native free grant` | `in-app purchase` |
  `web redirect`) as one pure function per platform so the branch is unit-tested without UI.
- **Internationalization** — All strings in `Localizable.xcstrings` / `strings.xml`, including the
  nine reasons; currency formatted by `BillingLogic.formatMoney`; RTL layouts verified.
- **Backward compatibility** — Flag off ⇒ no coupon UI, `?coupon=` ignored, purchase flows
  byte-identical to today. Older app versions ignore the parameter and buy at full price, which is
  the correct degradation — MKTC.7's rollout notes call this out for support.

## 7. Acceptance Criteria

- **AC-1.** *Given* Android, a paid course and a valid 25 % code, *When* I apply it, *Then* the
  detail screen shows the discounted price with the list price struck through and the CTA shows the
  new amount.
- **AC-2.** *Given* Android and an applied code, *When* I tap Buy, *Then* the purchase sheet shows
  the code, the discount and the final total, and the Custom Tab opens a Stripe session whose
  amount equals the previewed `chargedCents`.
- **AC-3.** *Given* Android and a 100 % code, *When* I tap the CTA, *Then* no Custom Tab opens, I am
  enrolled, and I land in the course.
- **AC-4.** *Given* iOS and a 100 % code, *When* I tap the CTA, *Then* StoreKit is never invoked, I
  am enrolled via the claim endpoint, and I land in the course.
- **AC-5.** *Given* iOS and a 30 % code, *When* I apply it, *Then* the discounted price is shown,
  the in-app purchase button for that amount is disabled with an explanatory hint, and I am offered
  Copy code plus the browser affordance and an explicit full-price in-app purchase option labelled
  with the full price.
- **AC-6.** *Given* iOS with a discount displayed, *When* I tap the full-price option, *Then* the
  button text and the confirmation both state the full price, and StoreKit charges the list price —
  never the discounted price.
- **AC-7.** *Given* either app, *When* I open a universal/app link `…/marketplace/{slug}?coupon=X`,
  *Then* the app opens the course detail in-app with the code applied and an accessibility
  announcement of the new price.
- **AC-8.** *Given* either app and an expired code, *When* I apply it, *Then* the localized
  "This code has expired" message appears and full-price purchase remains available.
- **AC-9.** *Given* either app and a code that lapses between preview and purchase, *When* the
  server returns 422, *Then* the discount is cleared, the reason is shown, and a second tap is
  required to buy at full price.
- **AC-10.** *Given* the iOS build, *When* it is reviewed against App Store guideline 3.1.1/3.1.3
  by the designated reviewer, *Then* the coupon flow is signed off in writing before submission and
  the remote-config kill switch for the external affordance is verified to work.
- **AC-11.** *Given* the flag is off, *When* either app opens a `?coupon=` link, *Then* no coupon UI
  appears and no preview request is made.
- **AC-12.** *Given* accessibility audits (VoiceOver on iOS, TalkBack + Accessibility Scanner on
  Android), *When* they run on the coupon states, *Then* there are no blocking findings and every
  price pair reads correctly.
- **AC-13.** *Given* the pure decision function on each platform, *When* unit tests run, *Then*
  every branch (free grant, in-app purchase, web redirect, rejected reason, flag off) is covered.

## 8. Data Model

No persistence. In-memory session state only:

- iOS: `@State private var coupon: CouponState?` on the detail view, plus
  `AppShellModel.pendingCoupon` for the deep-link handoff, cleared on purchase/ownership.
- Android: `MarketplaceDetailScreen` state holder, plus `HomeShellState.pendingCoupon`.

Explicitly **not** written to `OfflineModels` / Room / `UserDefaults` / DataStore (FR-5).

## 9. API Surface

No new server routes. Mobile clients gain:

```swift
// clients/ios/Lextures/Core/LMS/LMSAPIMarketplace.swift
static func previewMarketplaceCoupon(slug: String, code: String, accessToken: String) async throws -> CouponPreview
static func claimMarketplaceCourse(slug: String, couponCode: String?, accessToken: String) async throws -> MarketplaceClaimResult
static func checkoutMarketplaceCourse(slug: String, couponCode: String?, accessToken: String) async throws -> MarketplaceCheckoutResult
```

```kotlin
// clients/android/.../core/lms/LmsApi.kt
suspend fun previewMarketplaceCoupon(slug: String, code: String, accessToken: String): CouponPreview
suspend fun claimMarketplaceCourse(slug: String, couponCode: String?, accessToken: String): MarketplaceClaimResult
suspend fun checkoutMarketplaceCourse(slug: String, couponCode: String?, accessToken: String): MarketplaceCheckoutResult
```

Pure logic added to `MarketplaceLogic` on both platforms:

```
normalizeCouponCode(raw) -> String
couponReasonKey(reason) -> String
purchaseRoute(preview, platform, flags) -> .freeGrant | .inAppPurchase(fullPriceCents) | .webRedirect(url)
```

## 10. UI / UX

**iOS — `MarketplaceDetailView.swift`**
A `LMSCard` below the price block: "Have a coupon code?" disclosure → text field (auto-capitalized,
no autocorrect) + Apply button → on success a green-toned summary row with the code, the savings and
a Remove button. For a non-zero discounted price, an `LMSInlineNotice` explains the web redemption
with two buttons (Copy code, Continue in browser) and a separate, visually secondary
"Buy at full price — $40.00" button.

**Android — `MarketplaceDetailScreen.kt`**
`LmsCard` with an `OutlinedTextField` + `Button` row, the applied summary as an `AssistChip` plus a
savings line, and the same price presentation. The purchase sheet (`PurchaseFlowSheet.kt`) gains
three rows: Subtotal, Coupon (−$10.00, `LAUNCH25`), Total — sitting above the existing tax quote
rows so the arithmetic reads top to bottom.

**Flows**

1. *Deep link* — link tapped → `LinkOpener` routes in-app → detail screen opens → code auto-applied
   → price updates with an accessibility announcement.
2. *Typed* — expand → type → Apply → pending → applied.
3. *Android buy* — CTA → purchase sheet (subtotal/coupon/total) → Custom Tab → return handler polls
   entitlements → course.
4. *iOS free grant* — CTA reads "Enroll — Free" → claim → course.
5. *iOS discounted* — explanation notice → Copy code or Continue in browser; or the explicit
   full-price in-app purchase.

**States** — idle, checking (button spinner, field disabled), applied, rejected (inline error), rate
limited, offline (Apply disabled with "You're offline" copy).

**Accessibility annotations** — iOS: `accessibilityLabel` on the price pair
("Now $30.00, was $40.00"), `.accessibilityHint` on the disabled purchase button, VoiceOver
announcement via `AccessibilityNotification.Announcement` when the price changes, Dynamic Type
tested at XXXL. Android: `contentDescription` on the price pair, `liveRegion = Polite` on the price
composable, error text linked to the field via semantics, 48 dp targets.

**Copy keys** — mirror web's `marketplace.coupon.*` under the mobile prefix
(`mobile.marketplace.coupon.*`), including the nine reasons plus iOS-specific
`mobile.marketplace.coupon.webOnly`, `.copyCode`, `.continueInBrowser`, `.buyFullPrice`.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — StoreKit 2 (iOS, unchanged products), Chrome Custom Tabs (Android), the Stripe
  Checkout URL from the server.
- **Internal (iOS)** — `Features/Marketplace/MarketplaceDetailView.swift`,
  `Features/Billing/PurchaseFlow.swift`, `Features/Billing/CheckoutReturnHandler.swift`,
  `Core/LMS/LMSAPIMarketplace.swift`, `Core/LMS/MarketplaceLogic.swift`,
  `Core/LMS/LMSFeatureModelsMarketplace.swift`, `Core/Routing/MobileDestinations.swift`,
  `Core/Links/LinkOpener` (MB.1), `Resources/Localizable.xcstrings`.
- **Internal (Android)** — `features/marketplace/MarketplaceDetailScreen.kt`,
  `features/billing/PurchaseFlowSheet.kt`, `features/billing/BillingCheckout.kt`,
  `features/billing/CheckoutReturnOverlay.kt`, `core/lms/LmsApi.kt`,
  `core/lms/MarketplaceLogic.kt`, `core/navigation/MobileDestinations.kt`, `res/values/strings.xml`.
- **Events** — `MarketplaceObservability` on both platforms.

## 13. Dependencies & Sequencing

- **Must ship after** — MKTC.3 (API) and MKTC.5 (the web fallback the iOS path depends on must
  already work, and the reason vocabulary must be settled).
- **Must ship before** — MKTC.7 (GA + docs).
- **Shared infra** — app store release trains; the iOS piece is gated on the review sign-off in
  FR-15, so plan it against a release cycle rather than a sprint boundary.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| App Store rejection over the external purchase affordance | M | H | Pre-submission review sign-off (FR-15, AC-10); remote-config kill switch; documented fallback copy with no tappable link |
| iOS learner charged full price after seeing a discount | L | H | FR-14 forbids it; the full-price button states the full price; AC-6 pins it |
| iOS discounted path feels like a dead end, hurting conversion | H | M | Copy code + one-tap browser handoff; telemetry on `coupon_web_redirect` feeds the §18 Q2 decision |
| Deep-link parameter mishandled or injected into a WebView | M | M | Validate length/charset before use; build the browser URL from the server origin |
| Two platforms drift in reason copy | M | M | One vocabulary defined in MKTC.1; a test on each platform asserts every reason has a localized string |
| Android Custom Tab total differs from the sheet total | M | M | Sheet renders server numbers only (`chargedCents` + tax quote); AC-2 compares |
| Older app versions silently ignore the code | H | M | Correct degradation to full price, called out in support docs; server telemetry distinguishes app version |

## 15. Rollout Plan

- **Feature flag** — `ffCourseCoupons` from the platform features payload; both apps hide all coupon
  UI when off, so the server flag alone controls exposure without an app release.
- **Sequencing** — shared logic + API clients → Android UI + purchase sheet → iOS UI + free-grant
  path → iOS discounted-path affordance behind remote config → deep-link handling → localization →
  store submissions.
- **Dogfood** — internal TestFlight / internal Play track: redeem a 100 % code on both platforms,
  a 25 % code on Android, and walk the iOS web-redirect path end to end.
- **GA criteria** — accessibility audits clean; App Store sign-off obtained; deep links verified on
  both platforms from mail, notes and a browser; no coupon data on disk (verified by inspecting a
  device backup / app sandbox).
- **Rollback** — server flag off disables all coupon UI on both apps instantly. The iOS external
  affordance has its own remote-config switch so it can be disabled without disabling coupons.

## 16. Test Plan

- **Unit** — `MarketplaceLogic` on both platforms: `normalizeCouponCode`, `couponReasonKey`
  (a test fails if a reason has no localized string), `purchaseRoute` across every branch
  (free grant / in-app purchase / web redirect / rejected / flag off), price-pair formatting per
  currency and locale, deep-link parameter parsing and rejection of oversized or malformed values.
- **Integration** — API client encoding of `couponCode` on claim/checkout/preview; 422 and 429
  handling; free-grant response routing; Android purchase-sheet totals against a stubbed
  server; iOS StoreKit path asserted to be *not* invoked on the free-grant branch (StoreKit test
  configuration `Configuration.storekit`).
- **End-to-end** — extend `e2e/tests/mobile-marketplace-purchases.spec.ts` coverage where the
  harness allows; plus platform UI tests (XCUITest / Espresso) for: apply a code, deep-link land,
  free grant, expired code, and the iOS discounted-path affordance.
- **Security** — deep-link fuzzing with oversized/encoded/script-bearing `coupon` values; verify no
  coupon is persisted (inspect app container after backgrounding and relaunch); verify the browser
  handoff URL cannot be influenced by the parameter.
- **Accessibility** — VoiceOver and TalkBack scripts over all coupon states; Dynamic Type XXXL and
  Android font scale 2.0 without truncation; Accessibility Scanner clean; RTL screenshots.
- **Performance / load** — apply latency on a throttled connection; ensure the detail screen renders
  before the preview resolves.
- **Manual exploratory** — background the app mid-Custom-Tab and return; airplane mode during apply;
  a code that expires while the purchase sheet is open; an iOS learner who copies the code and
  completes on the web, then returns to the app (ownership must be reflected after refresh).

## 17. Documentation & Training

- **End-user docs** — add a mobile section to "Using a coupon code": how links open in the app,
  and — stated plainly — that on iPhone and iPad a partial-discount code is redeemed on the web.
- **Admin / instructor docs** — add to the creator help page: "your share links work on phones;
  partial discounts are completed on the web on iOS", so creators set expectations in their own
  marketing.
- **API reference** — no change.
- **Internal runbook** — "iOS learner says the discount did not apply": how to confirm the app
  version, the flag state, and the web-redirect telemetry; how to grant a comp entitlement if the
  learner is stuck.
- **Support macro** — a canned answer for the iOS partial-discount case with the web link.

## 18. Open Questions

1. Should creators be able to *view* (not edit) their coupons on mobile? (Proposed: no for v1;
   management stays on web.)
2. Should we introduce a small ladder of StoreKit price-point products (e.g. 25 %/50 %/75 % tiers)
   so common discounts can be honoured in-app? (Proposed: decide after 60 days of
   `coupon_web_redirect` volume; it trades product sprawl for iOS conversion.)
3. Does the current App Store guidance permit a tappable external purchase link for this case, and
   under which entitlement? Owner: whoever handled the 3.1.1 Path A submission. **Blocks iOS
   submission, not iOS development** — build behind the remote-config switch either way.
4. Should a 100 %-off native grant on iOS be limited in any way to avoid looking like an IAP bypass
   (e.g. a per-user cap enforced server-side)? (Proposed: rely on the coupon's own caps; document
   the reasoning for review.)
5. Should Android eventually move to Play Billing, and would that recreate the iOS problem?
   (Out of scope; flag for the mobile roadmap.)

## 19. References

- Existing files: `clients/ios/Lextures/Features/Billing/PurchaseFlow.swift` (StoreKit purchase),
  `clients/ios/Lextures/Features/Marketplace/MarketplaceDetailView.swift`,
  `clients/ios/Lextures/Core/LMS/LMSAPIMarketplace.swift:74-94`,
  `clients/ios/Lextures/Core/LMS/MarketplaceLogic.swift`,
  `clients/android/.../features/billing/BillingCheckout.kt`,
  `clients/android/.../features/billing/PurchaseFlowSheet.kt`,
  `clients/android/.../core/lms/LmsApi.kt:3199-3217`,
  `clients/android/.../core/lms/MarketplaceLogic.kt:58`.
- External standards: App Store Review Guidelines 3.1.1 / 3.1.3 (and the current external-purchase
  link rules), Google Play Payments policy, WCAG 2.1 AA as applied to native apps,
  Apple HIG and Material accessibility guidance.
- Related plans: [MKTC.3](../../completed/marketplace/MKTC.3-coupon-aware-checkout-and-redemption.md),
  [MKTC.5](../../completed/marketplace/MKTC.5-web-learner-coupon-entry-and-url-codes.md),
  [MKT6](../../completed/marketplace/MKT6-marketplace-mobile.md),
  [MB.1 link handling](../../completed/mobile/),
  `docs/ios-app-store-3.1.1-iap-path-a.md`.
