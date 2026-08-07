# App Store Guideline 3.1.1 — Path A (In-App Purchase)

**Status:** Implemented in codebase (StoreKit 2 + server verify).  
**Companion rejection notes:** [ios-app-store-review-rejection-2026-08-04.md](./ios-app-store-review-rejection-2026-08-04.md)

This document is the operator + engineer guide for selling **digital courses and subscriptions** on iOS via **Apple In-App Purchase**, so multiplatform access satisfies **3.1.3(b)** (content bought outside iOS may unlock in the app **because the same items are also available via IAP**).

---

## Product model

| Content | Purchase channel on iOS | Server acquisition source |
|---------|-------------------------|---------------------------|
| Institutional / school seats (SSO, roster) | Not sold in app | N/A (org provisioning) |
| One-time course (catalog / marketplace) | StoreKit non-consumable | `apple` |
| Platform monthly / annual plan | StoreKit auto-renewable | `apple` |
| Same course / plan on web | Stripe Checkout | `stripe` |

Web and Android keep Stripe (and Play Billing when added). iOS **does not** open Stripe for digital course checkout.

---

## App Store Connect setup (required before Review)

1. **Agreements** — Paid Applications agreement active; banking/tax complete.
2. **In-App Purchases** (bundle `com.lextures.ios`):
   - Auto-renewable subscription group, e.g. **Lextures Access**:
     - `com.lextures.ios.sub.monthly`
     - `com.lextures.ios.sub.annual`
   - Non-consumable **per paid course** you want buyable on iOS, e.g. `com.lextures.ios.course.<slug-or-uuid>`.
3. Submit each IAP with the app binary (or as a metadata update). Clear review screenshots and sandbox tester accounts.
4. **Sandbox** — Create StoreKit sandbox Apple IDs for App Review notes.

### Map products in Lextures

| Kind | Configuration |
|------|----------------|
| Monthly sub | Env `APPLE_IAP_MONTHLY_PRODUCT_ID` |
| Annual sub | Env `APPLE_IAP_ANNUAL_PRODUCT_ID` |
| Bundle id | Env `APPLE_IAP_BUNDLE_ID=com.lextures.ios` (or `OIDC_APPLE_NATIVE_AUDIENCE`) |
| Course | SQL / admin: `course.courses.apple_product_id = 'com.lextures.ios.course…'` |

Courses **without** `apple_product_id` cannot be purchased on iOS (Purchase sheet shows “not available for In-App Purchase”). Set the column for every marketplace/catalog SKU that should sell on iOS.

Optional: `APPLE_IAP_ROOT_CA_PATH` pointing at Apple Root CA PEMs for full x5c chain verification. Local only: `APPLE_IAP_SKIP_SIGNATURE_VERIFY=true` (ignored unless `APP_ENV=local`).

---

## Runtime flow

```
iOS PurchaseFlowSheet
  → GET /api/v1/billing/apple/products?courseId=
  → StoreKit Product.products + product.purchase(appAccountToken: userUUID)
  → POST /api/v1/billing/apple/verify { signedTransaction, courseId? }
  → server DecodeAppleTransactionJWS → map product → billing.user_entitlements
  → enroll student for course_purchase
  → CheckoutReturnHandler / course workspace
```

**Restore purchases** (Profile → Billing) walks `Transaction.currentEntitlements` and re-posts each JWS to verify.

Unfinished transactions sync on app foreground when signed in.

---

## API

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/billing/apple/products` | yes | Product ids for course (+ env subs) |
| POST | `/api/v1/billing/apple/verify` | yes | Verify StoreKit JWS; grant entitlement |

Billing feature flags (`ffStripeBilling` / `ffPaymentsEnabled`) still gate the billing surface.

---

## App Review reply template (3.1.1)

> Digital courses and homeschool subscriptions are available for purchase **in the app using In-App Purchase** (StoreKit).  
>  
> **How to purchase in the app**  
> 1. Sign in with the demo account.  
> 2. Open a paid course (Catalog or Marketplace).  
> 3. Tap **Purchase** and complete the App Store payment sheet.  
> 4. The server verifies the signed transaction and unlocks the course.  
>  
> **Product IDs**  
> - Monthly: `com.lextures.ios.sub.monthly`  
> - Annual: `com.lextures.ios.sub.annual`  
> - Courses: non-consumables listed in App Store Connect; mapped via `course.courses.apple_product_id`.  
>  
> **Multiplatform (3.1.3(b))**  
> Users may also buy on the website (Stripe). Content acquired on the web is accessible in the iOS app; the same digital offerings are available as In-App Purchases.  
>  
> **Institutional access**  
> School-provisioned seats are not consumer IAPs; students sign in with school accounts.  
>  
> **Restore**  
> Profile → Billing → **Restore purchases**.

---

## Local testing

1. Set env (API):

```bash
export APPLE_IAP_BUNDLE_ID=com.lextures.ios
export APPLE_IAP_MONTHLY_PRODUCT_ID=com.lextures.ios.sub.monthly
export APPLE_IAP_ANNUAL_PRODUCT_ID=com.lextures.ios.sub.annual
export APPLE_IAP_SKIP_SIGNATURE_VERIFY=true   # APP_ENV=local only
export APP_ENV=local
```

2. Map a course:

```sql
UPDATE course.courses
SET apple_product_id = 'com.lextures.ios.course.demo'
WHERE code = 'your-demo-course';
```

3. Xcode scheme → **StoreKit Configuration** → `clients/ios/Configuration.storekit`.

4. Run app, purchase demo product, confirm entitlement in Profile → Billing / My Purchases.

---

## Code map

| Area | Path |
|------|------|
| Migration | `server/migrations/466_apple_iap.sql` |
| Verify + grant | `server/internal/service/billing/apple_iap.go` |
| HTTP | `server/internal/httpserver/billing_apple_iap_http.go` |
| StoreKit | `clients/ios/Lextures/Core/LMS/StoreKitPurchaseService.swift` |
| Purchase UI | `clients/ios/Lextures/Features/Billing/PurchaseFlow.swift` |
| Sandbox config | `clients/ios/Configuration.storekit` |

---

## Follow-ups (not blocking first IAP resubmit)

- App Store Server Notifications V2 webhook for refunds/revocations (mirror Stripe refund handling).
- Admin UI to set `apple_product_id` on courses.
- Auto-create ASC products via App Store Connect API when listing marketplace courses.
- Hide “Manage web subscription” outside US if Review objects to Stripe portal CTAs (currently secondary to App Store subscription management).
