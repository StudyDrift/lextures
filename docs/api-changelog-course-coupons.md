# API changelog — Course Coupons (MKTC.2 / MKTC.3 / MKTC.7)

## 2026-08 — Abuse prevention, analytics & platform ceiling (MKTC.7)

Creator routes (same auth as list: `item:create` + `ffCourseMarketplace` + `ffCourseCoupons`):

| Verb | Path | Notes |
|---|---|---|
| GET | `/api/v1/courses/{course_code}/coupons/summary` | Per-coupon performance: `redeemedCount`, `refundedCount`, `grossListCents`, `discountCents`, `netChargedCents` (refunds excluded from net) |
| GET | `/api/v1/courses/{course_code}/coupons/{coupon_id}/redemptions.csv` | Streamed CSV; **5 exports/hour/user**; audit event `course.coupon.redemptions_exported` |

Learner preview changes:

| Verb | Path | Notes |
|---|---|---|
| POST | `/api/v1/marketplace/courses/{slug}/coupon/preview` | Layered limits: **15/min & 60/hour per (user,course)**, **100/hour per user**, **200/hour per IP**. **10 consecutive fails → 15 min cool-down**. `429` body includes `reason` (`rate_limited` \| `cooldown`) and `Retry-After`. Failed applies logged to `billing.coupon_attempts` (salted code hash; IP prefix only). |

Platform settings:

| Field | Notes |
|---|---|
| `couponMaxPercentOff` | Optional cap on percent coupons (default **100** = uncapped). Create above ceiling → `422`. |
| `ffCourseCoupons` | **Default ON** at GA (MKTC.7). Explicit platform setting `false` stays off. |

Create response may include `warnings: ["low_entropy"]` for codes under 6 characters or dictionary words (non-blocking).

Metrics added/confirmed: `coupon_apply_cooldown_total`, `coupon_web_redirect_total{platform}` (plus existing MKTC.1–3 coupon series).

See OpenAPI, [runbook](runbooks/coupons.md), [threat model](security/coupon-enumeration-threat-model.md), [dashboard](monitoring/coupon-dashboard.md).

## 2026-08 — Learner preview, checkout & redemption (MKTC.3)

Authenticated learner routes (gated by `ffCourseMarketplace`; coupon-bearing
paths also require `ffCourseCoupons`, default **OFF** until MKTC.7):

| Verb | Path | Notes |
|---|---|---|
| POST | `/api/v1/marketplace/courses/{slug}/coupon/preview` | Body `{ "code" }`; **always 200** with `{applied, reason, listPriceCents, discountCents, chargedCents, currency, freeAfterDiscount, endsAt, seatsRemaining}` — failures use typed `reason`, not HTTP errors |
| POST | `/api/v1/marketplace/courses/{slug}/checkout` | Optional body `{ "couponCode" }`; paid path returns `{sessionId, checkoutUrl, chargedCents, currency}`; 100%-off returns claim shape + `grantedFree:true`; inapplicable code → `422` with `reason` + full price |
| POST | `/api/v1/marketplace/courses/{slug}/claim` | Optional `{ "couponCode" }` for 100%-off on a paid course; without code, paid courses still `402` |
| GET | `/api/v1/marketplace/courses/{slug}?coupon=` | Optional query; response may include `coupon` object (same shape as preview); bad codes never 4xx the detail |
| POST | `/api/v1/checkout/quote` | Optional `couponCode`; tax subtotal uses the discounted amount when the code applies |

Server rules:

- Client-supplied amounts (`chargedCents`, `discountCents`, `priceCents`) are **ignored**
- First-party coupon sessions set Stripe `allow_promotion_codes=false` and metadata `coupon_id`, `coupon_code`, `coupon_discount_cents`, `list_price_cents`
- Preview rate limit: 15/min + 60/hour per user (plus per-IP); checkout uses the billing checkout limit
- With `ffCourseCoupons` off, preview 404s and `couponCode` is ignored on checkout/claim

See OpenAPI and [MKTC.3 plan](completed/marketplace/MKTC.3-coupon-aware-checkout-and-redemption.md).

## 2026-08 — Creator coupon management API (MKTC.2)

Authenticated staff routes under `/api/v1/courses/{course_code}/coupons…`
(gated by `ffCourseMarketplace` **and** `ffCourseCoupons`, both required;
`ffCourseCoupons` defaults **OFF** until GA / MKTC.7):

| Verb | Path | Notes |
|---|---|---|
| GET | `/coupons` | List with live seat counts; `?includeArchived=true` opts archived rows in |
| POST | `/coupons` | Create; code normalized upper-case; `201` + `shareUrl` / `publicShareUrl` |
| PATCH | `/coupons/{coupon_id}` | Partial: `note`, window, caps, `status` (`active`\|`disabled`); discount fields immutable → `422` |
| DELETE | `/coupons/{coupon_id}` | Soft archive (`status=archived`); redemptions retained |
| GET | `/coupons/{coupon_id}/redemptions` | Cursor-paginated (`limit` default 25, max 100) |

Authorisation: course `item:create` (owner / instructor / teacher / designer).
Students, TAs, observers, and other non-staff roles receive `403`. Unknown or
inaccessible courses receive `404` with the same body as other course routes.

Key validation:

- Free course (`price_cents = 0`) → `422` on create
- Fixed coupon currency must match course `price_currency` → `422`
- Duplicate non-archived code → `409`
- `maxRedemptions` below current consumed seats → `422`
- Write rate limit: 30 requests/min/user → `429`

See OpenAPI (`server/internal/openapi/openapi.json`) and
[MKTC.2 plan](completed/marketplace/MKTC.2-creator-coupon-management-api.md).
