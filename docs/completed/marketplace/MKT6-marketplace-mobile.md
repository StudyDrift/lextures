# MKT6 — Marketplace on Mobile (iOS + Android)

> Implementation plan. Source: [docs/plan/marketplace/README.md](../../plan/marketplace/README.md). Part of the Marketplace epic.

## Metadata

| Field | Value |
|---|---|
| **Feature ID** | MKT6 |
| **Section** | Marketplace |
| **Severity** | MAJOR |
| **Markets** | SL (primary) · HE · K12 |
| **Status (today)** | SHIPPED |
| **Estimated effort** | L (1–2mo) |
| **Owner (proposed)** | Mobile platform team (iOS + Android) |
| **Depends on** | MKT1, MKT2, MKT3, MKT4, MKT5 |
| **Unblocks** | — (epic parity complete) |

---

## 1. Problem Statement

The marketplace ships on web (MKT2–MKT5) but the native iOS and Android apps have no storefront, no course-settings marketplace toggle, no purchase flow, and no purchased indicator. The user requirement is explicit that the **marketplace appears in the mobile sidenav** and that purchased courses are indicated on the mobile Courses list. Mobile also introduces a hard constraint the web doesn't have: **Apple App Store and Google Play require in-app purchase (IAP) for digital content**, which changes how *paid* purchases must work. Free claims are unaffected. This story brings the whole flow to mobile and resolves the IAP path.

## 2. Goals

- Add a **Marketplace** destination to both mobile apps' navigation (Android `MoreDestination`, iOS `MoreDestination`), gated on the new `ffCourseMarketplace` platform feature.
- Build the mobile storefront (browse/filter/detail) reusing the MKT3 endpoints.
- Implement mobile **free claim** in-app (no IAP), and resolve the **paid** purchase path per App Store / Play policy (StoreKit 2 / Play Billing vs. web hand-off) — see §14/§18.
- Add the marketplace listing + fee control to mobile **course settings** (parity with MKT2), extending the actively-developed mobile course-settings surface.
- Show the **"Purchased"** indicator on the mobile Courses list (parity with MKT5).
- Extend the mobile platform-feature models with `ffCourseMarketplace`.

## 3. Non-Goals

- Web surfaces (MKT2–MKT5).
- New backend endpoints beyond what MKT3/MKT4 already define (mobile reuses them), except server-side IAP **receipt validation** if the StoreKit/Play path is chosen (scoped here, implemented as a sub-task).
- Offline purchasing (purchases require connectivity).
- Redesigning the mobile IA beyond adding the marketplace destination.

## 4. Personas & User Stories

- **As a mobile learner**, I want a Marketplace entry in the app so that I can discover and enroll in courses on my phone.
- **As a mobile learner**, I want to claim free courses in one tap so that starting is frictionless.
- **As a mobile learner buying a paid course**, I want a purchase flow that complies with the platform I'm on so that the app isn't rejected and I'm charged correctly.
- **As an instructor on mobile**, I want to list my course and set a price from course settings so that I'm not forced to a laptop.
- **As a mobile learner**, I want purchased courses marked on my Courses list so that I can tell them apart.

## 5. Functional Requirements

- **FR-1.** Both apps MUST add a "Marketplace" destination, shown only when `ffCourseMarketplace` is true (Android: add `MoreDestination.Marketplace` in the `moreDestinations` builder; iOS: add `case marketplace` to `MoreDestination` and its `moreDestinations(...)` gating), placed near `Catalog`/`Paths`.
- **FR-2.** Both apps MUST fetch and render the storefront via MKT3's `GET /api/v1/marketplace/courses` (+ categories, detail), with search, filters, Free/price badges, and an `owned` state.
- **FR-3.** For **free** courses, both apps MUST call MKT4's `POST /api/v1/marketplace/courses/{slug}/claim` and, on success, route into the course — no IAP.
- **FR-4.** For **paid** courses, the app MUST follow the chosen policy path (§14):
  - **Path A (native IAP)** — present StoreKit 2 (iOS) / Play Billing (Android) purchase for a product mapped to the course; on completion, send the receipt/purchase token to a **server receipt-validation endpoint** that verifies with Apple/Google and then creates the entitlement + enrollment (reusing MKT4's entitlement/enroll logic).
  - **Path B (web hand-off)** — paid courses are browse-only in-app with a compliant message; purchase completes on web. (Higher rejection risk; see §18.)
- **FR-5.** Both apps MUST reflect `owned` on storefront cards/detail as "Owned / Go to course" and prevent re-purchase.
- **FR-6.** Mobile **course settings** MUST add a "Marketplace" section (list toggle + fee, default Free), writing via MKT2's extended `PUT /api/v1/courses/{code}/catalog-listing`, on both Android (`CourseSettingsScreens.kt`) and iOS (`CourseSettingsHostView.swift`).
- **FR-7.** The mobile **Courses list** MUST show a "Purchased" indicator using the MKT5 fields (`acquiredViaMarketplace`, `acquisitionSource`).
- **FR-8.** Both apps' platform-feature models MUST parse `ffCourseMarketplace` (Android `MobilePlatformFeatures`, iOS feature struct) with default false client-side (server default ON governs actual state).
- **FR-9.** All new strings MUST be localized across the app's supported locales (en, es, fr, ar, en-XA pseudo) in each platform's string resources.
- **FR-10.** When the flag is off, no marketplace nav item, no settings section, and no purchased badge MUST render.

## 6. Non-Functional Requirements

- **Performance** — Storefront list renders incrementally with paging; images lazy-loaded; p95 interaction < 300 ms after data. Reuse platform image/caching stacks.
- **Security** — Session-authenticated API calls; paid path validates receipts **server-side** (never trust client claims of purchase); price resolved server-side. No secrets in the app.
- **Privacy & Compliance** — App Store / Play data-safety disclosures updated for purchases; entitlements are financial records (15.13). Free claims store no payment PII.
- **Accessibility** — VoiceOver (iOS) / TalkBack (Android): storefront cards, price/Free/owned in accessible labels; purchase buttons labelled with state; dynamic type / font scaling; sufficient contrast in light/dark; RTL (ar) verified.
- **Scalability** — Reuses server endpoints; no new hot paths beyond receipt validation.
- **Reliability** — Purchase is idempotent end-to-end (MKT4 guarantees + receipt validation dedupes by transaction id); interrupted purchases reconcile on next launch (StoreKit/Play transaction queue).
- **Observability** — Client analytics: `mobile_marketplace_view`, `mobile_claim`, `mobile_purchase{platform,path}`, plus server metrics from MKT4 + receipt-validation success/failure.
- **Maintainability** — Reuse the shared mobile LMS API layer (`LmsApi.kt`, iOS LMS core) and feature-model plumbing; mirror web behavior to keep parity.
- **Internationalization** — All copy in `strings.xml` (+ locale variants) / `Localizable.xcstrings`; currency via platform formatters; RTL-safe.
- **Backward compatibility** — Additive nav/models; older app versions without the flag simply don't show the marketplace (server flag ON is harmless to them).

## 7. Acceptance Criteria

- **AC-1.** *Given* `ffCourseMarketplace` is on, *When* I open either app's nav, *Then* a "Marketplace" item appears and opens the storefront.
- **AC-2.** *Given* a free listed course, *When* I tap "Enroll — Free", *Then* I'm claimed+enrolled and land in the course (no payment sheet).
- **AC-3.** *Given* a paid course and Path A, *When* I complete the native purchase, *Then* the server validates the receipt, creates the entitlement+enrollment, and I access the course.
- **AC-4.** *Given* I own a course, *When* I view it in the mobile storefront, *Then* it shows "Go to course" and cannot be re-purchased.
- **AC-5.** *Given* I'm an instructor on mobile, *When* I open course settings, *Then* I can list the course and set a fee (default Free), persisting via the shared endpoint.
- **AC-6.** *Given* I acquired a course via the marketplace, *When* I open the mobile Courses list, *Then* it shows a "Purchased" indicator.
- **AC-7.** *Given* the flag is off, *When* I use either app, *Then* no marketplace nav, settings section, or badge appears.
- **AC-8.** *Given* an interrupted paid purchase, *When* I relaunch, *Then* the transaction reconciles and access is granted exactly once.

## 8. Data Model

No new server tables. If Path A (native IAP) is chosen, add mapping/validation records:
- `billing.iap_receipts` (or extend `user_entitlements`) to store `platform` (`apple`/`google`), `transaction_id`/`purchase_token` (unique for idempotency), `product_id`, `course_id`, `user_id`, `validated_at`. The `transaction_id` unique constraint is the IAP idempotency key feeding MKT4's entitlement creation (`acquisition_source` extended with `apple`/`google` or reuse `stripe` semantics via a generic `paid` source — decide in §18).
- Client stores no durable purchase state beyond the platform transaction queue.

## 9. API Surface

Mobile reuses MKT3/MKT4 endpoints. Path A adds one server endpoint:
- `POST /api/v1/marketplace/iap/validate` → body `{ platform, courseSlug, receipt|purchaseToken, productId }`; verifies with Apple App Store Server API / Google Play Developer API, then creates entitlement + enrollment (idempotent on transaction id), returns `{ owned: true, firstItemId? }`. Gated by `courseMarketplaceOff` + billing flag.
- Reuse: storefront list/detail (MKT3), free `claim` (MKT4), Courses list with purchased fields (MKT5), `/platform/features` (`ffCourseMarketplace`).
- OpenAPI: document the IAP validation endpoint (if Path A).

## 10. UI / UX

- **Navigation** — Android: `MoreDestination.Marketplace` (icon + `mobile_ia_more_marketplace` string) added in `MobileDestinations.kt` `moreDestinations` list under the `platform.ffCourseMarketplace` gate; iOS: `case marketplace` in `MoreDestination` with SF Symbol (e.g. `bag`) and gating in `moreDestinations(...)`.
- **Storefront screens** — list (search + filter sheet + Free/price + owned badges), detail (description, what's-included, price, CTA), mirroring MKT3. New Compose screens (Android) + SwiftUI views (iOS).
- **Purchase** — free: simple confirm → claim → navigate. Paid Path A: native purchase sheet → pending → server-validated → navigate. Paid Path B: informational sheet + "open on web".
- **Course settings** — "Marketplace" section in `CourseSettingsScreens.kt` / `CourseSettingsHostView.swift`: toggle + fee editor (default Free), parity with MKT2, using the shared LMS models (`LmsFeatureModels*`).
- **Courses list** — "Purchased" chip on course rows using MKT5 fields.
- **States** — loading skeletons, empty ("No courses available yet"), error+retry, offline banner (browse cached, purchase disabled offline), pending purchase, owned.
- **Accessibility** — VoiceOver/TalkBack labels incl. price/Free/owned/state; dynamic type; RTL.
- **Copy & i18n** — new keys in `strings.xml` (+ `values-es/fr/ar/en-rXA`) and `Localizable.xcstrings`; mirror the mobile locale JSON (`clients/mobile/locales/*.json`) if the shared RN/i18n layer is used.

## 11. AI / ML Considerations

Not AI-touching.

## 12. Integration Points

- **External** — Apple App Store Server API / StoreKit 2; Google Play Billing + Play Developer API (Path A only); existing Stripe path stays web.
- **Internal (Android)** — `core/navigation/MobileDestinations.kt`, `core/lms/LmsApi.kt`, `LmsFeatureModels*.kt`, `features/courses/settings/CourseSettingsScreens.kt`, courses list screen, `res/values*/strings.xml`.
- **Internal (iOS)** — `Core/Routing/MobileDestinations.swift`, `Core/LMS/LMSFeatureModelsPlatform.swift`, `Features/Courses/Settings/CourseSettingsHostView.swift`, courses list view, `Resources/Localizable.xcstrings`, `Lextures.xcodeproj`.
- **Server** — reuse MKT3/MKT4/MKT5; add IAP validation (Path A).

## 13. Dependencies & Sequencing

- **After** — MKT1–MKT5 (all server + web behavior settled; mobile mirrors it).
- **Before** — none (epic parity).
- **Shared infra** — Apple/Google billing accounts + product setup (Path A); server receipt validation; existing mobile feature-flag + LMS API plumbing.

## 14. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| App Store/Play reject web-only paid purchases (Path B) | H | H | Prefer Path A (native IAP) for paid; if Path B, restrict to compliant "reader" patterns and legal review; free claims always allowed in-app |
| IAP commission (15–30%) breaks creator economics | H | M | Price modeling; possibly paid courses web-only for margin while free/preview in-app; product decision (§18) |
| Duplicate grant across web + IAP (user buys twice) | M | H | Server idempotency on `(user,course)` (MKT1) + show `owned` everywhere before CTA |
| Receipt-validation complexity/outages | M | M | Server-side validation with retries + reconciliation on relaunch; StoreKit/Play transaction queue as source of truth |
| Localization gaps across 5 locales | M | L | String extraction gate in CI; pseudo-locale (en-XA) smoke test |
| Nav IA clutter in "More" | L | L | Group with Catalog/Paths; feature-gated |

## 15. Rollout Plan

- **Flag** — `ffCourseMarketplace` (MKT1) gates all mobile surfaces; paid path additionally gated by billing flags + store product availability.
- **Sequencing** — ship browse + free claim + purchased badge + settings toggle first (no store review risk); paid IAP path in a follow-up app release after store product setup + review.
- **Dogfood** — internal TestFlight / internal Play track: free claim + settings + badge, then sandbox IAP purchase + refund.
- **GA criteria** — nav gated correctly; free claim + purchased indicator work; settings persist; (Path A) sandbox purchase validates and grants exactly once; store review passed.
- **Rollback** — server flag off hides mobile surfaces without an app update; disable paid CTA via a remote sub-flag if store issues arise.

## 16. Test Plan

- **Unit** — feature-model parsing of `ffCourseMarketplace`; nav gating; price/Free/owned label logic (both platforms).
- **Integration** — storefront list/detail against MKT3; free claim against MKT4; settings write against MKT2; Courses list purchased fields (MKT5); IAP validation endpoint idempotency (Path A).
- **End-to-end** — Android (Espresso/Compose test) + iOS (XCUITest): nav→storefront→filter→detail→free claim→course; settings list+fee; purchased badge; sandbox paid purchase (Path A).
- **Security** — server receipt validation rejects forged/replayed receipts; price server-authoritative; no cross-user grants.
- **Accessibility** — VoiceOver/TalkBack scripts; dynamic type; RTL (ar); contrast light/dark.
- **Performance** — list scrolling/paging; image caching.
- **Manual exploratory** — interrupted purchase reconciles on relaunch; airplane-mode purchase blocked gracefully; store sandbox refund.

## 17. Documentation & Training

- **Learner docs** — mobile-specific "buy/enroll on iOS/Android," including IAP notes.
- **Instructor docs** — listing a course + setting a fee from the mobile app.
- **Store listings** — data-safety / purchase disclosures.
- **Runbook** — reconciling stuck IAP transactions; validating a receipt manually.

## 18. Open Questions

1. **Paid path: native IAP (Path A) or keep paid web-only (Path B)?** Primary decision — impacts store-review risk and creator margin (15–30% platform fee). Recommendation: free claims in-app always; paid via native IAP if we want in-app conversion, else clearly web-only. **Needs product + legal sign-off before build.**
2. If IAP: how are courses mapped to store products (per-course products vs. a small set of price tiers)? (Store product management overhead.)
3. Extend `acquisition_source` with `apple`/`google`, or collapse to a generic `paid`? (Default: add `apple`/`google` for reporting.)
4. Does the mobile marketplace live in the "More" drawer/tab or get promoted to a primary tab? (Default: "More" destination alongside Catalog/Paths.)
5. Should mobile reuse the shared RN/i18n locale JSON (`clients/mobile/locales/*.json`) or per-native strings only? (Follow the pattern the active M13.x course-settings work established.)

## 19. References

- Existing files: `clients/android/app/src/main/kotlin/com/lextures/android/core/navigation/MobileDestinations.kt` (`MoreDestination`, `moreDestinations`, `MobilePlatformFeatures`), `core/lms/LmsApi.kt`, `features/courses/settings/CourseSettingsScreens.kt`; `clients/ios/Lextures/Core/Routing/MobileDestinations.swift` (`MoreDestination`, `moreDestinations`), `Core/LMS/LMSFeatureModelsPlatform.swift`, `Features/Courses/Settings/CourseSettingsHostView.swift`; string resources `res/values*/strings.xml`, `Resources/Localizable.xcstrings`, `clients/mobile/locales/*.json`.
- Related plans: [MKT1](MKT1-marketplace-platform-foundation.md)–[MKT5](MKT5-purchased-indicator-courses.md); `docs/plan/mobile/README.md`, `docs/MOBILE_PLAN.md`.
- External standards: Apple App Store Review Guidelines §3.1 (In-App Purchase), Google Play Payments policy, StoreKit 2, Google Play Billing Library.
